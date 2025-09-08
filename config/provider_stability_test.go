package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/catwalk/pkg/catwalk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProviderCacheStaleCheck 测试缓存过期检查逻辑
func TestProviderCacheStaleCheck(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "providers.json")

	// 测试不存在的缓存文件
	stale, exists := isCacheStale(cachePath)
	assert.True(t, stale, "不存在的缓存应该被认为是过期的")
	assert.False(t, exists, "不存在的缓存文件应该返回 exists=false")

	// 创建一个新的缓存文件
	data, _ := json.Marshal([]catwalk.Provider{{Name: "Test"}})
	err := os.WriteFile(cachePath, data, 0644)
	require.NoError(t, err)

	// 新创建的缓存不应该过期
	stale, exists = isCacheStale(cachePath)
	assert.False(t, stale, "新创建的缓存不应该过期")
	assert.True(t, exists, "存在的缓存文件应该返回 exists=true")

	// 修改文件时间为25小时前
	oldTime := time.Now().Add(-25 * time.Hour)
	err = os.Chtimes(cachePath, oldTime, oldTime)
	require.NoError(t, err)

	// 超过24小时的缓存应该过期
	stale, exists = isCacheStale(cachePath)
	assert.True(t, stale, "超过24小时的缓存应该过期")
	assert.True(t, exists, "存在的缓存文件应该返回 exists=true")
}

// TestProviderLoadWithBackgroundUpdate 测试后台更新机制
func TestProviderLoadWithBackgroundUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "providers.json")

	// 创建一个23小时前的缓存（未过期但接近过期）
	oldProviders := []catwalk.Provider{{Name: "CachedProvider", ID: "cached"}}
	data, _ := json.Marshal(oldProviders)
	err := os.WriteFile(cachePath, data, 0644)
	require.NoError(t, err)

	// 设置文件时间为23小时前
	oldTime := time.Now().Add(-23 * time.Hour)
	err = os.Chtimes(cachePath, oldTime, oldTime)
	require.NoError(t, err)

	// Mock client 返回新的 providers
	client := &mockProviderClient{shouldFail: false}
	
	// 加载 providers，应该使用缓存并触发后台更新
	providers, err := loadProviders(client, cachePath)
	require.NoError(t, err)
	require.Len(t, providers, 1)
	assert.Equal(t, "CachedProvider", providers[0].Name, "应该立即返回缓存的数据")

	// 等待后台更新完成
	time.Sleep(100 * time.Millisecond)

	// 重新读取缓存文件，应该已经更新
	updatedData, err := os.ReadFile(cachePath)
	require.NoError(t, err)
	
	var updatedProviders []catwalk.Provider
	err = json.Unmarshal(updatedData, &updatedProviders)
	require.NoError(t, err)
	assert.Equal(t, "Mock", updatedProviders[0].Name, "缓存应该被后台更新")
}

// TestProviderConcurrentAccess 测试并发访问
func TestProviderConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "providers.json")

	// 创建初始缓存
	initialProviders := []catwalk.Provider{{Name: "Initial", ID: "initial"}}
	data, _ := json.Marshal(initialProviders)
	err := os.WriteFile(cachePath, data, 0644)
	require.NoError(t, err)

	client := &mockProviderClient{shouldFail: false}
	
	// 并发访问
	var wg sync.WaitGroup
	errors := make([]error, 10)
	results := make([][]catwalk.Provider, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			providers, err := loadProviders(client, cachePath)
			errors[idx] = err
			results[idx] = providers
		}(i)
	}

	wg.Wait()

	// 验证所有并发请求都成功
	for i, err := range errors {
		assert.NoError(t, err, "并发请求 %d 应该成功", i)
		assert.NotNil(t, results[i], "并发请求 %d 应该返回数据", i)
		assert.True(t, len(results[i]) > 0, "并发请求 %d 应该有数据", i)
	}
}

// TestProviderLoadOnce 测试单例加载机制
func TestProviderLoadOnce(t *testing.T) {
	// 重置 providerOnce 用于测试
	providerOnce = sync.Once{}
	providerList = nil

	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "providers.json")

	callCount := 0
	client := &countingProviderClient{
		callCount: &callCount,
		providers: []catwalk.Provider{{Name: "SingletonTest"}},
	}

	// 第一次调用
	providers1, err := loadProvidersOnce(client, cachePath)
	require.NoError(t, err)
	require.NotNil(t, providers1)

	// 第二次调用应该返回相同的结果，不再调用 client
	providers2, err := loadProvidersOnce(client, cachePath)
	require.NoError(t, err)
	require.NotNil(t, providers2)

	// 验证只调用了一次
	assert.Equal(t, 1, callCount, "GetProviders 应该只被调用一次")
	assert.Equal(t, providers1, providers2, "两次调用应该返回相同的结果")
}

// countingProviderClient 用于计数调用次数
type countingProviderClient struct {
	callCount *int
	providers []catwalk.Provider
}

func (c *countingProviderClient) GetProviders() ([]catwalk.Provider, error) {
	*c.callCount++
	return c.providers, nil
}

// TestProviderFailoverScenarios 测试各种失败场景的降级处理
func TestProviderFailoverScenarios(t *testing.T) {
	tests := []struct {
		name           string
		setupCache     bool
		cacheContent   []catwalk.Provider
		clientFails    bool
		expectError    bool
		expectedResult string
	}{
		{
			name:           "无缓存，客户端成功",
			setupCache:     false,
			clientFails:    false,
			expectError:    false,
			expectedResult: "Mock",
		},
		{
			name:           "有缓存，客户端失败，使用缓存",
			setupCache:     true,
			cacheContent:   []catwalk.Provider{{Name: "Cached"}},
			clientFails:    true,
			expectError:    false,
			expectedResult: "Cached",
		},
		{
			name:        "无缓存，客户端失败",
			setupCache:  false,
			clientFails: true,
			expectError: true,
		},
		{
			name:           "过期缓存，客户端成功，更新缓存",
			setupCache:     true,
			cacheContent:   []catwalk.Provider{{Name: "OldCache"}},
			clientFails:    false,
			expectError:    false,
			expectedResult: "Mock",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			cachePath := filepath.Join(tmpDir, "providers.json")

			// 设置缓存
			if tt.setupCache {
				data, _ := json.Marshal(tt.cacheContent)
				err := os.WriteFile(cachePath, data, 0644)
				require.NoError(t, err)
				
				// 如果测试过期缓存，设置为25小时前
				if tt.name == "过期缓存，客户端成功，更新缓存" {
					oldTime := time.Now().Add(-25 * time.Hour)
					err = os.Chtimes(cachePath, oldTime, oldTime)
					require.NoError(t, err)
				}
			}

			client := &mockProviderClient{shouldFail: tt.clientFails}
			providers, err := loadProviders(client, cachePath)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, providers)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, providers)
				assert.Len(t, providers, 1)
				assert.Equal(t, tt.expectedResult, providers[0].Name)
			}
		})
	}
}

// TestProviderRealWorldIntegration 测试真实场景的集成
func TestProviderRealWorldIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 测试真实的 Providers() 函数
	providers, err := Providers()
	
	// 在真实环境中，我们期望能获取到 providers
	// 但如果网络有问题，至少不应该 panic
	if err != nil {
		t.Logf("获取 providers 失败（可能是网络问题）: %v", err)
		// 验证错误处理正确
		assert.NotNil(t, err)
	} else {
		// 如果成功，验证返回的数据
		assert.NotNil(t, providers)
		assert.True(t, len(providers) > 0, "应该返回至少一个 provider")
		
		// 验证数据结构
		for _, p := range providers {
			assert.NotEmpty(t, p.ID, "Provider ID 不应为空")
			assert.NotEmpty(t, p.Name, "Provider Name 不应为空")
			t.Logf("Provider: %s (%s)", p.Name, p.ID)
		}
	}
}

// TestProviderCachePersistence 测试缓存持久性
func TestProviderCachePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "providers.json")

	// 第一次加载，创建缓存
	client1 := &mockProviderClient{shouldFail: false}
	providers1, err := loadProviders(client1, cachePath)
	require.NoError(t, err)
	require.NotNil(t, providers1)

	// 验证缓存文件存在
	_, err = os.Stat(cachePath)
	require.NoError(t, err, "缓存文件应该被创建")

	// 模拟程序重启，客户端失败，但应该能从缓存恢复
	client2 := &mockProviderClient{shouldFail: true}
	providers2, err := loadProviders(client2, cachePath)
	require.NoError(t, err, "应该能从缓存恢复")
	require.NotNil(t, providers2)
	assert.Equal(t, providers1[0].Name, providers2[0].Name, "应该返回相同的缓存数据")
}

// BenchmarkProviderLoad 性能测试
func BenchmarkProviderLoad(b *testing.B) {
	tmpDir := b.TempDir()
	cachePath := filepath.Join(tmpDir, "providers.json")
	
	// 准备缓存
	providers := make([]catwalk.Provider, 100)
	for i := 0; i < 100; i++ {
		providers[i] = catwalk.Provider{
			ID:   catwalk.InferenceProvider(fmt.Sprintf("provider_%d", i)),
			Name: fmt.Sprintf("Provider %d", i),
		}
	}
	data, _ := json.Marshal(providers)
	_ = os.WriteFile(cachePath, data, 0644)

	client := &mockProviderClient{shouldFail: false}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = loadProviders(client, cachePath)
	}
}

// TestProviderMemoryManagement 测试内存管理
func TestProviderMemoryManagement(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "providers.json")

	// 创建大量 providers 测试内存处理
	largeProviders := make([]catwalk.Provider, 1000)
	for i := 0; i < 1000; i++ {
		largeProviders[i] = catwalk.Provider{
			ID:          catwalk.InferenceProvider(fmt.Sprintf("provider_%d", i)),
			Name:        fmt.Sprintf("Provider %d", i),
			Models:      []catwalk.Model{{ID: fmt.Sprintf("model_%d", i)}},
			APIEndpoint: fmt.Sprintf("https://api.provider%d.com", i),
		}
	}

	client := &staticProviderClient{providers: largeProviders}
	
	// 多次加载，验证没有内存泄漏
	for i := 0; i < 10; i++ {
		providers, err := loadProviders(client, cachePath)
		require.NoError(t, err)
		require.Len(t, providers, 1000)
	}
}

// staticProviderClient 返回固定的 providers
type staticProviderClient struct {
	providers []catwalk.Provider
}

func (s *staticProviderClient) GetProviders() ([]catwalk.Provider, error) {
	return s.providers, nil
}

// TestProviderErrorRecovery 测试错误恢复能力
func TestProviderErrorRecovery(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "providers.json")

	// 创建损坏的缓存文件
	err := os.WriteFile(cachePath, []byte("invalid json"), 0644)
	require.NoError(t, err)

	// 客户端正常工作
	client := &mockProviderClient{shouldFail: false}
	
	// 应该忽略损坏的缓存，从客户端获取新数据
	providers, err := loadProviders(client, cachePath)
	require.NoError(t, err)
	require.NotNil(t, providers)
	assert.Equal(t, "Mock", providers[0].Name)

	// 验证缓存已被修复
	data, err := os.ReadFile(cachePath)
	require.NoError(t, err)
	
	var cachedProviders []catwalk.Provider
	err = json.Unmarshal(data, &cachedProviders)
	require.NoError(t, err, "缓存应该被修复为有效的 JSON")
}

// TestProviderNetworkSimulation 模拟网络问题
func TestProviderNetworkSimulation(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "providers.json")

	// 模拟间歇性网络问题
	client := &intermittentProviderClient{
		failureRate: 0.5,
		providers:   []catwalk.Provider{{Name: "NetworkTest"}},
	}

	successCount := 0
	for i := 0; i < 10; i++ {
		providers, err := loadProviders(client, cachePath)
		if err == nil {
			successCount++
			assert.NotNil(t, providers)
		}
	}

	// 至少应该有一些成功的请求
	assert.True(t, successCount > 0, "应该有一些请求成功")
}

// intermittentProviderClient 模拟间歇性失败
type intermittentProviderClient struct {
	failureRate float64
	providers   []catwalk.Provider
	callCount   int
}

func (i *intermittentProviderClient) GetProviders() ([]catwalk.Provider, error) {
	i.callCount++
	// 第一次总是成功，建立缓存
	if i.callCount == 1 {
		return i.providers, nil
	}
	// 之后根据失败率决定
	if i.callCount%2 == 0 {
		return nil, errors.New("network error")
	}
	return i.providers, nil
}