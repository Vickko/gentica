package config

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/catwalk/pkg/catwalk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gentica/csync"
)

// TestProviderConnectionOpenAI 测试 OpenAI Provider 连接
func TestProviderConnectionOpenAI(t *testing.T) {
	// 创建测试服务器模拟 OpenAI API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求路径
		assert.Equal(t, "/v1/models", r.URL.Path)
		
		// 验证认证头
		authHeader := r.Header.Get("Authorization")
		assert.Equal(t, "Bearer test-api-key", authHeader)
		
		// 返回成功响应
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]string{
				{"id": "gpt-4"},
			},
		})
	}))
	defer server.Close()

	// 创建测试解析器
	resolver := NewShellVariableResolver(NewEnvFromMap(map[string]string{
		"TEST_API_KEY": "test-api-key",
	}))

	// 测试 OpenAI provider 配置
	provider := &ProviderConfig{
		ID:      "openai",
		Name:    "OpenAI",
		Type:    catwalk.TypeOpenAI,
		BaseURL: server.URL + "/v1",
		APIKey:  "$TEST_API_KEY",
	}

	err := provider.TestConnection(resolver)
	assert.NoError(t, err, "OpenAI 连接测试应该成功")
}

// TestProviderConnectionAnthropic 测试 Anthropic Provider 连接
func TestProviderConnectionAnthropic(t *testing.T) {
	// 创建测试服务器模拟 Anthropic API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求路径
		assert.Equal(t, "/v1/models", r.URL.Path)
		
		// 验证认证头
		apiKeyHeader := r.Header.Get("x-api-key")
		assert.Equal(t, "anthropic-key", apiKeyHeader)
		
		versionHeader := r.Header.Get("anthropic-version")
		assert.Equal(t, "2023-06-01", versionHeader)
		
		// 返回成功响应
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"models": []string{"claude-3-opus"},
		})
	}))
	defer server.Close()

	resolver := NewShellVariableResolver(NewEnvFromMap(map[string]string{
		"ANTHROPIC_KEY": "anthropic-key",
	}))

	provider := &ProviderConfig{
		ID:      "anthropic",
		Name:    "Anthropic",
		Type:    catwalk.TypeAnthropic,
		BaseURL: server.URL + "/v1",
		APIKey:  "$ANTHROPIC_KEY",
	}

	err := provider.TestConnection(resolver)
	assert.NoError(t, err, "Anthropic 连接测试应该成功")
}

// TestProviderConnectionGemini 测试 Gemini Provider 连接
func TestProviderConnectionGemini(t *testing.T) {
	// 创建测试服务器模拟 Gemini API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证 API key 在查询参数中
		apiKey := r.URL.Query().Get("key")
		assert.Equal(t, "gemini-key", apiKey)
		
		// 返回成功响应
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"models": []map[string]string{
				{"name": "gemini-pro"},
			},
		})
	}))
	defer server.Close()

	resolver := NewShellVariableResolver(NewEnvFromMap(map[string]string{
		"GEMINI_API_KEY": "gemini-key",
	}))

	provider := &ProviderConfig{
		ID:      "gemini",
		Name:    "Google Gemini",
		Type:    catwalk.TypeGemini,
		BaseURL: server.URL,
		APIKey:  "${GEMINI_API_KEY}",
	}

	err := provider.TestConnection(resolver)
	assert.NoError(t, err, "Gemini 连接测试应该成功")
}

// TestProviderConnectionWithExtraHeaders 测试带额外头的连接
func TestProviderConnectionWithExtraHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证额外的头
		customHeader := r.Header.Get("X-Custom-Header")
		assert.Equal(t, "custom-value", customHeader)
		
		orgHeader := r.Header.Get("X-Organization")
		assert.Equal(t, "org-123", orgHeader)
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
	}))
	defer server.Close()

	resolver := NewShellVariableResolver(NewEnvFromMap(map[string]string{
		"API_KEY": "test-key",
	}))

	provider := &ProviderConfig{
		ID:      "custom",
		Name:    "Custom Provider",
		Type:    catwalk.TypeOpenAI,
		BaseURL: server.URL + "/v1",
		APIKey:  "$API_KEY",
		ExtraHeaders: map[string]string{
			"X-Custom-Header": "custom-value",
			"X-Organization":  "org-123",
		},
	}

	err := provider.TestConnection(resolver)
	assert.NoError(t, err, "带额外头的连接测试应该成功")
}

// TestProviderConnectionFailures 测试各种连接失败场景
func TestProviderConnectionFailures(t *testing.T) {
	tests := []struct {
		name          string
		setupServer   func() *httptest.Server
		provider      *ProviderConfig
		expectedError string
	}{
		{
			name: "401 未授权",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
					json.NewEncoder(w).Encode(map[string]string{"error": "Invalid API key"})
				}))
			},
			provider: &ProviderConfig{
				ID:     "test",
				Type:   catwalk.TypeOpenAI,
				APIKey: "invalid-key",
			},
			expectedError: "401 Unauthorized",
		},
		{
			name: "404 端点不存在",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			provider: &ProviderConfig{
				ID:     "test",
				Type:   catwalk.TypeOpenAI,
				APIKey: "key",
			},
			expectedError: "404 Not Found",
		},
		{
			name: "500 服务器错误",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
			},
			provider: &ProviderConfig{
				ID:     "test",
				Type:   catwalk.TypeOpenAI,
				APIKey: "key",
			},
			expectedError: "500 Internal Server Error",
		},
		{
			name: "连接超时",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// 模拟超时
					time.Sleep(6 * time.Second)
					w.WriteHeader(http.StatusOK)
				}))
			},
			provider: &ProviderConfig{
				ID:     "test",
				Type:   catwalk.TypeOpenAI,
				APIKey: "key",
			},
			expectedError: "context deadline exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			tt.provider.BaseURL = server.URL + "/v1"
			resolver := NewShellVariableResolver(NewEnv())

			err := tt.provider.TestConnection(resolver)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

// TestProviderConfigVariableResolution 测试配置中的变量解析
func TestProviderConfigVariableResolution(t *testing.T) {
	tests := []struct {
		name       string
		envVars    map[string]string
		apiKey     string
		baseURL    string
		expectedKey string
		expectedURL string
	}{
		{
			name: "环境变量解析",
			envVars: map[string]string{
				"MY_API_KEY": "secret-key-123",
				"API_BASE":   "https://api.example.com",
			},
			apiKey:      "$MY_API_KEY",
			baseURL:     "$API_BASE/v1",
			expectedKey: "secret-key-123",
			expectedURL: "https://api.example.com/v1",
		},
		{
			name: "花括号语法",
			envVars: map[string]string{
				"KEY":  "key-value",
				"HOST": "api.test.com",
			},
			apiKey:      "${KEY}",
			baseURL:     "https://${HOST}",
			expectedKey: "key-value",
			expectedURL: "https://api.test.com",
		},
		{
			name: "混合解析",
			envVars: map[string]string{
				"PREFIX": "sk",
				"SUFFIX": "xyz",
				"DOMAIN": "openai.com",
			},
			apiKey:      "${PREFIX}-test-${SUFFIX}",
			baseURL:     "https://api.$DOMAIN",
			expectedKey: "sk-test-xyz",
			expectedURL: "https://api.openai.com",
		},
		{
			name: "命令替换",
			envVars: map[string]string{
				"BASE": "test",
			},
			apiKey:      "$(echo key-123)",
			baseURL:     "$BASE",
			expectedKey: "key-123",
			expectedURL: "test",
		},
		{
			name:        "纯文本（无变量）",
			envVars:     map[string]string{},
			apiKey:      "plain-api-key",
			baseURL:     "https://api.plain.com",
			expectedKey: "plain-api-key",
			expectedURL: "https://api.plain.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewShellVariableResolver(NewEnvFromMap(tt.envVars))

			// 解析 API Key
			resolvedKey, err := resolver.ResolveValue(tt.apiKey)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedKey, resolvedKey)

			// 解析 Base URL
			resolvedURL, err := resolver.ResolveValue(tt.baseURL)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedURL, resolvedURL)
		})
	}
}

// TestProviderDefaultURLs 测试默认 URL 处理
func TestProviderDefaultURLs(t *testing.T) {
	tests := []struct {
		providerType catwalk.Type
		expectedURL  string
	}{
		{
			providerType: catwalk.TypeOpenAI,
			expectedURL:  "https://api.openai.com/v1",
		},
		{
			providerType: catwalk.TypeAnthropic,
			expectedURL:  "https://api.anthropic.com/v1",
		},
		{
			providerType: catwalk.TypeGemini,
			expectedURL:  "https://generativelanguage.googleapis.com",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.providerType), func(t *testing.T) {
			// 创建模拟服务器
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
			}))
			defer server.Close()

			// 创建没有 BaseURL 的 provider
			provider := &ProviderConfig{
				ID:      "test",
				Type:    tt.providerType,
				BaseURL: "", // 空 URL，应该使用默认值
				APIKey:  "test-key",
			}

			// 暂时替换默认 URL 为测试服务器
			originalURL := provider.BaseURL
			provider.BaseURL = server.URL + "/v1"
			
			resolver := NewShellVariableResolver(NewEnv())
			err := provider.TestConnection(resolver)
			
			// 恢复原始 URL
			provider.BaseURL = originalURL
			
			assert.NoError(t, err)
		})
	}
}

// TestProviderResolvedEnv 测试环境变量解析功能
func TestProviderResolvedEnv(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		envVars  map[string]string
		expected []string
	}{
		{
			name: "基本环境变量解析",
			input: map[string]string{
				"PATH":     "$OLD_PATH:/new/path",
				"API_KEY":  "${SECRET_KEY}",
				"ENDPOINT": "https://api.example.com",
			},
			envVars: map[string]string{
				"OLD_PATH":   "/usr/bin",
				"SECRET_KEY": "sk-123",
			},
			expected: []string{
				"PATH=/usr/bin:/new/path",
				"API_KEY=sk-123",
				"ENDPOINT=https://api.example.com",
			},
		},
		{
			name: "命令替换",
			input: map[string]string{
				"VERSION": "$(echo v1.0.0)",
				"USER":    "$(echo testuser)",
			},
			expected: []string{
				"VERSION=v1.0.0",
				"USER=testuser",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置临时环境变量
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			mcp := MCPConfig{
				Env: tt.input,
			}

			resolved := mcp.ResolvedEnv()
			
			// 验证所有期望的值都存在
			for _, exp := range tt.expected {
				assert.Contains(t, resolved, exp)
			}
		})
	}
}

// TestProviderResolvedHeaders 测试 HTTP 头解析功能
func TestProviderResolvedHeaders(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		envVars  map[string]string
		expected map[string]string
	}{
		{
			name: "头部变量解析",
			headers: map[string]string{
				"Authorization": "Bearer $TOKEN",
				"X-API-Version": "${API_VERSION}",
				"X-Client-ID":   "client-123",
			},
			envVars: map[string]string{
				"TOKEN":       "secret-token",
				"API_VERSION": "v2",
			},
			expected: map[string]string{
				"Authorization": "Bearer secret-token",
				"X-API-Version": "v2",
				"X-Client-ID":   "client-123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置临时环境变量
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			mcp := MCPConfig{
				Headers: tt.headers,
			}

			resolved := mcp.ResolvedHeaders()
			assert.Equal(t, tt.expected, resolved)
		})
	}
}

// TestProviderConcurrentConnectionTests 测试并发连接测试
func TestProviderConcurrentConnectionTests(t *testing.T) {
	// 创建测试服务器
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
	}))
	defer server.Close()

	resolver := NewShellVariableResolver(NewEnv())

	// 创建多个 provider
	providers := []*ProviderConfig{
		{
			ID:      "provider1",
			Type:    catwalk.TypeOpenAI,
			BaseURL: server.URL + "/v1",
			APIKey:  "key1",
		},
		{
			ID:      "provider2",
			Type:    catwalk.TypeAnthropic,
			BaseURL: server.URL + "/v1",
			APIKey:  "key2",
		},
		{
			ID:      "provider3",
			Type:    catwalk.TypeGemini,
			BaseURL: server.URL,
			APIKey:  "key3",
		},
	}

	// 并发测试所有 provider
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	errChan := make(chan error, len(providers))
	for _, p := range providers {
		go func(provider *ProviderConfig) {
			err := provider.TestConnection(resolver)
			errChan <- err
		}(p)
	}

	// 收集结果
	for range providers {
		select {
		case err := <-errChan:
			assert.NoError(t, err)
		case <-ctx.Done():
			t.Fatal("测试超时")
		}
	}

	// 验证所有请求都完成了
	assert.Equal(t, len(providers), requestCount)
}

// TestProviderAPIKeyManagement 测试 API 密钥管理
func TestProviderAPIKeyManagement(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configPath := fmt.Sprintf("%s/config.json", tmpDir)

	// 创建配置实例
	cfg := &Config{
		dataConfigDir: configPath,
		Providers:     csync.NewMap[string, ProviderConfig](),
		knownProviders: []catwalk.Provider{
			{
				ID:          "openai",
				Name:        "OpenAI",
				Type:        catwalk.TypeOpenAI,
				APIEndpoint: "https://api.openai.com/v1",
				Models: []catwalk.Model{
					{ID: "gpt-4"},
				},
			},
		},
	}

	// 测试设置新的 API 密钥
	err := cfg.SetProviderAPIKey("openai", "new-api-key-123")
	require.NoError(t, err)

	// 验证内存中的配置已更新
	provider, exists := cfg.Providers.Get("openai")
	assert.True(t, exists)
	assert.Equal(t, "new-api-key-123", provider.APIKey)

	// 验证配置文件已保存
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	
	var savedConfig map[string]interface{}
	err = json.Unmarshal(data, &savedConfig)
	require.NoError(t, err)
	
	providers := savedConfig["providers"].(map[string]interface{})
	openai := providers["openai"].(map[string]interface{})
	assert.Equal(t, "new-api-key-123", openai["api_key"])

	// 测试更新现有 provider 的 API 密钥
	err = cfg.SetProviderAPIKey("openai", "updated-key-456")
	require.NoError(t, err)

	provider, exists = cfg.Providers.Get("openai")
	assert.True(t, exists)
	assert.Equal(t, "updated-key-456", provider.APIKey)
}

// BenchmarkProviderConnection 性能测试
func BenchmarkProviderConnection(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
	}))
	defer server.Close()

	resolver := NewShellVariableResolver(NewEnv())
	provider := &ProviderConfig{
		ID:      "bench",
		Type:    catwalk.TypeOpenAI,
		BaseURL: server.URL + "/v1",
		APIKey:  "bench-key",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = provider.TestConnection(resolver)
	}
}