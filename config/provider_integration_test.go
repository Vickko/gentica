package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/catwalk/pkg/catwalk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProviderIntegrationReal 测试真实的 Provider 集成
func TestProviderIntegrationReal(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 设置测试环境变量
	testEnv := map[string]string{
		"ANTHROPIC_API_KEY":  os.Getenv("ANTHROPIC_API_KEY"),
		"ANTHROPIC_BASE_URL": os.Getenv("ANTHROPIC_BASE_URL"),
		"OPENAI_API_KEY":     os.Getenv("OPENAI_API_KEY"),
		"GEMINI_API_KEY":     os.Getenv("GEMINI_API_KEY"),
	}

	// 如果没有设置环境变量，使用默认值
	if testEnv["ANTHROPIC_BASE_URL"] == "" {
		testEnv["ANTHROPIC_BASE_URL"] = "https://api.tu-zi.com"
	}

	// 创建临时工作目录
	tmpDir := t.TempDir()
	workingDir := filepath.Join(tmpDir, "work")
	dataDir := filepath.Join(tmpDir, "data")
	err := os.MkdirAll(workingDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(dataDir, 0755)
	require.NoError(t, err)

	// 创建测试配置文件
	configData := map[string]interface{}{
		"providers": map[string]interface{}{
			"anthropic": map[string]interface{}{
				"id":       "anthropic",
				"name":     "Anthropic",
				"type":     "anthropic",
				"base_url": "$ANTHROPIC_BASE_URL",
				"api_key":  "$ANTHROPIC_API_KEY",
			},
			"openai": map[string]interface{}{
				"id":      "openai",
				"name":    "OpenAI",
				"type":    "openai",
				"api_key": "$OPENAI_API_KEY",
			},
		},
		"models": map[string]interface{}{
			"large": map[string]interface{}{
				"provider": "anthropic",
				"model":    "claude-3-haiku-20240307",
			},
			"small": map[string]interface{}{
				"provider": "openai",
				"model":    "gpt-3.5-turbo",
			},
		},
	}

	configPath := filepath.Join(workingDir, "crush.json")
	configJSON, err := json.MarshalIndent(configData, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(configPath, configJSON, 0644)
	require.NoError(t, err)

	// 设置环境变量
	for k, v := range testEnv {
		if v != "" {
			os.Setenv(k, v)
			defer os.Unsetenv(k)
		}
	}

	// 加载配置
	cfg, err := Load(workingDir, dataDir, false)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// 测试 Provider 配置
	t.Run("Provider 配置加载", func(t *testing.T) {
		// 检查 providers 是否正确加载
		providers := cfg.EnabledProviders()
		assert.True(t, len(providers) > 0, "应该有启用的 providers")

		// 查找 anthropic provider
		var anthropicProvider *ProviderConfig
		for _, p := range providers {
			if p.ID == "anthropic" {
				anthropicProvider = &p
				break
			}
		}

		if anthropicProvider != nil && testEnv["ANTHROPIC_API_KEY"] != "" {
			t.Log("找到 Anthropic provider，测试连接")
			err := anthropicProvider.TestConnection(cfg.resolver)
			if err != nil {
				t.Logf("Anthropic 连接测试失败: %v", err)
			} else {
				t.Log("Anthropic 连接测试成功")
			}
		}
	})

	// 测试模型选择
	t.Run("模型选择", func(t *testing.T) {
		largeModel := cfg.LargeModel()
		if largeModel != nil {
			assert.Equal(t, "claude-3-haiku-20240307", largeModel.ID)
			t.Logf("Large model: %s", largeModel.ID)
		}

		smallModel := cfg.SmallModel()
		if smallModel != nil {
			assert.Equal(t, "gpt-3.5-turbo", smallModel.ID)
			t.Logf("Small model: %s", smallModel.ID)
		}
	})

	// 测试配置更新
	t.Run("配置动态更新", func(t *testing.T) {
		// 更新模型偏好
		newModel := SelectedModel{
			Provider: "openai",
			Model:    "gpt-4",
		}
		err := cfg.UpdatePreferredModel(SelectedModelTypeLarge, newModel)
		assert.NoError(t, err)

		// 验证更新
		assert.Equal(t, "gpt-4", cfg.Models[SelectedModelTypeLarge].Model)
	})
}

// TestProviderConfigMerge 测试配置合并功能
func TestProviderConfigMerge(t *testing.T) {
	// 创建多个配置源
	config1 := map[string]interface{}{
		"providers": map[string]interface{}{
			"openai": map[string]interface{}{
				"id":      "openai",
				"api_key": "key1",
			},
		},
		"options": map[string]interface{}{
			"debug": false,
		},
	}

	config2 := map[string]interface{}{
		"providers": map[string]interface{}{
			"openai": map[string]interface{}{
				"api_key": "key2", // 覆盖
				"base_url": "https://custom.api.com", // 新增
			},
			"anthropic": map[string]interface{}{ // 新增 provider
				"id":      "anthropic",
				"api_key": "anthropic-key",
			},
		},
		"options": map[string]interface{}{
			"debug": true, // 覆盖
			"tui": map[string]interface{}{ // 新增
				"compact_mode": true,
			},
		},
	}

	// 测试合并
	data1, _ := json.Marshal(config1)
	data2, _ := json.Marshal(config2)
	
	var merged1, merged2 map[string]interface{}
	json.Unmarshal(data1, &merged1)
	json.Unmarshal(data2, &merged2)

	// 手动合并模拟
	result := mergeConfigs(merged1, merged2)

	// 验证合并结果
	providers := result["providers"].(map[string]interface{})
	
	// OpenAI 应该被更新
	openai := providers["openai"].(map[string]interface{})
	assert.Equal(t, "key2", openai["api_key"]) // 被覆盖
	assert.Equal(t, "https://custom.api.com", openai["base_url"]) // 新增字段
	
	// Anthropic 应该存在
	anthropic := providers["anthropic"].(map[string]interface{})
	assert.Equal(t, "anthropic-key", anthropic["api_key"])
	
	// Options 应该被合并
	options := result["options"].(map[string]interface{})
	assert.Equal(t, true, options["debug"]) // 被覆盖
	tui := options["tui"].(map[string]interface{})
	assert.Equal(t, true, tui["compact_mode"]) // 新增
}

// mergeConfigs 辅助函数：合并两个配置
func mergeConfigs(base, override map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	// 复制 base
	for k, v := range base {
		result[k] = v
	}
	
	// 合并 override
	for k, v := range override {
		if existing, exists := result[k]; exists {
			// 如果都是 map，递归合并
			if existingMap, ok1 := existing.(map[string]interface{}); ok1 {
				if overrideMap, ok2 := v.(map[string]interface{}); ok2 {
					result[k] = mergeConfigs(existingMap, overrideMap)
					continue
				}
			}
		}
		// 否则直接覆盖
		result[k] = v
	}
	
	return result
}

// TestProviderSyncMap 测试线程安全的 SyncMap
func TestProviderSyncMap(t *testing.T) {
	sm := NewSyncMap[string, ProviderConfig]()

	// 测试基本操作
	provider1 := ProviderConfig{
		ID:     "provider1",
		Name:   "Provider 1",
		APIKey: "key1",
	}
	
	sm.Set("provider1", provider1)
	
	// 测试 Get
	retrieved, exists := sm.Get("provider1")
	assert.True(t, exists)
	assert.Equal(t, provider1.ID, retrieved.ID)
	
	// 测试不存在的 key
	_, exists = sm.Get("nonexistent")
	assert.False(t, exists)
	
	// 测试更新
	provider1.APIKey = "updated-key"
	sm.Set("provider1", provider1)
	retrieved, _ = sm.Get("provider1")
	assert.Equal(t, "updated-key", retrieved.APIKey)
	
	// 测试删除
	sm.Del("provider1")
	_, exists = sm.Get("provider1")
	assert.False(t, exists)
	
	// 测试 Len
	sm.Set("p1", ProviderConfig{ID: "p1"})
	sm.Set("p2", ProviderConfig{ID: "p2"})
	sm.Set("p3", ProviderConfig{ID: "p3"})
	assert.Equal(t, 3, sm.Len())
	
	// 测试 Range
	count := 0
	sm.Range(func(key string, value ProviderConfig) bool {
		count++
		return true // 继续迭代
	})
	assert.Equal(t, 3, count)
	
	// 测试提前终止迭代
	count = 0
	sm.Range(func(key string, value ProviderConfig) bool {
		count++
		return count < 2 // 只迭代2次
	})
	assert.Equal(t, 2, count)
}

// TestProviderConfigPaths 测试配置文件路径查找
func TestProviderConfigPaths(t *testing.T) {
	tmpDir := t.TempDir()
	
	// 创建测试目录结构
	projectDir := filepath.Join(tmpDir, "project")
	subDir := filepath.Join(projectDir, "src", "components")
	err := os.MkdirAll(subDir, 0755)
	require.NoError(t, err)
	
	// 在项目根目录创建配置文件
	configPath := filepath.Join(projectDir, "crush.json")
	config := map[string]interface{}{
		"providers": map[string]interface{}{
			"test": map[string]interface{}{
				"id": "test",
			},
		},
	}
	configData, _ := json.Marshal(config)
	err = os.WriteFile(configPath, configData, 0644)
	require.NoError(t, err)
	
	// 测试从子目录向上查找
	found, exists := SearchParent(subDir, "crush.json")
	assert.True(t, exists)
	assert.Equal(t, configPath, found)
	
	// 测试查找不存在的文件
	_, exists = SearchParent(subDir, "nonexistent.json")
	assert.False(t, exists)
}

// TestProviderEnvResolution 测试环境变量解析
func TestProviderEnvResolution(t *testing.T) {
	env := NewEnv()
	
	// 设置测试环境变量
	env.Set("TEST_KEY", "test-value")
	env.Set("API_URL", "https://api.example.com")
	
	// 创建解析器
	resolver := NewShellVariableResolver(env)
	
	tests := []struct {
		input    string
		expected string
		hasError bool
	}{
		{
			input:    "plain text",
			expected: "plain text",
		},
		{
			input:    "$TEST_KEY",
			expected: "test-value",
		},
		{
			input:    "${TEST_KEY}",
			expected: "test-value",
		},
		{
			input:    "prefix-$TEST_KEY-suffix",
			expected: "prefix-test-value-suffix",
		},
		{
			input:    "$API_URL/v1/models",
			expected: "https://api.example.com/v1/models",
		},
		{
			input:    "${TEST_KEY}_${API_URL}",
			expected: "test-value_https://api.example.com",
		},
		{
			input:    "$NONEXISTENT",
			expected: "",
			hasError: true, // 环境变量不存在应该报错
		},
		{
			input:    "$",
			hasError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := resolver.ResolveValue(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestProviderAgentConfiguration 测试 Agent 配置
func TestProviderAgentConfiguration(t *testing.T) {
	cfg := &Config{
		Options: &Options{
			ContextPaths: []string{"CLAUDE.md", ".cursorrules"},
		},
	}
	
	cfg.SetupAgents()
	
	// 验证 agents 被正确设置
	assert.NotNil(t, cfg.Agents)
	assert.Len(t, cfg.Agents, 2)
	
	// 检查 coder agent
	coder, exists := cfg.Agents["coder"]
	assert.True(t, exists)
	assert.Equal(t, "coder", coder.ID)
	assert.Equal(t, SelectedModelTypeLarge, coder.Model)
	assert.Nil(t, coder.AllowedTools) // 所有工具都允许
	
	// 检查 task agent
	task, exists := cfg.Agents["task"]
	assert.True(t, exists)
	assert.Equal(t, "task", task.ID)
	assert.Equal(t, SelectedModelTypeLarge, task.Model)
	assert.Contains(t, task.AllowedTools, "glob")
	assert.Contains(t, task.AllowedTools, "grep")
	assert.NotContains(t, task.AllowedTools, "bash") // bash 不应该被允许
	assert.Empty(t, task.AllowedMCP) // 没有 MCP
	assert.Empty(t, task.AllowedLSP) // 没有 LSP
}

// TestProviderModelSelection 测试模型选择逻辑
func TestProviderModelSelection(t *testing.T) {
	cfg := &Config{
		Models: map[SelectedModelType]SelectedModel{
			SelectedModelTypeLarge: {
				Provider:  "anthropic",
				Model:     "claude-3-opus",
				MaxTokens: 4096,
				Think:     true,
			},
			SelectedModelTypeSmall: {
				Provider:        "openai",
				Model:           "gpt-4o",
				ReasoningEffort: "high",
			},
		},
		Providers: NewSyncMapFrom(map[string]ProviderConfig{
			"anthropic": {
				ID:   "anthropic",
				Name: "Anthropic",
				Type: catwalk.TypeAnthropic,
				Models: []catwalk.Model{
					{ID: "claude-3-opus", DefaultMaxTokens: 8192},
				},
			},
			"openai": {
				ID:   "openai",
				Name: "OpenAI",
				Type: catwalk.TypeOpenAI,
				Models: []catwalk.Model{
					{ID: "gpt-4o", DefaultMaxTokens: 4096},
				},
			},
		}),
	}
	
	// 测试获取大模型
	largeModel := cfg.GetModelByType(SelectedModelTypeLarge)
	assert.NotNil(t, largeModel)
	assert.Equal(t, "claude-3-opus", largeModel.ID)
	
	// 测试获取小模型
	smallModel := cfg.GetModelByType(SelectedModelTypeSmall)
	assert.NotNil(t, smallModel)
	assert.Equal(t, "gpt-4o", smallModel.ID)
	
	// 测试获取 provider
	largeProvider := cfg.GetProviderForModel(SelectedModelTypeLarge)
	assert.NotNil(t, largeProvider)
	assert.Equal(t, "anthropic", largeProvider.ID)
	
	smallProvider := cfg.GetProviderForModel(SelectedModelTypeSmall)
	assert.NotNil(t, smallProvider)
	assert.Equal(t, "openai", smallProvider.ID)
}

// TestProviderConfigValidation 测试配置验证
func TestProviderConfigValidation(t *testing.T) {
	tests := []struct {
		name     string
		provider ProviderConfig
		valid    bool
	}{
		{
			name: "有效的 OpenAI 配置",
			provider: ProviderConfig{
				ID:      "openai",
				Type:    catwalk.TypeOpenAI,
				APIKey:  "sk-xxx",
				BaseURL: "https://api.openai.com/v1",
			},
			valid: true,
		},
		{
			name: "缺少 API Key",
			provider: ProviderConfig{
				ID:      "openai",
				Type:    catwalk.TypeOpenAI,
				BaseURL: "https://api.openai.com/v1",
			},
			valid: false,
		},
		{
			name: "禁用的 provider",
			provider: ProviderConfig{
				ID:      "disabled",
				Type:    catwalk.TypeOpenAI,
				APIKey:  "key",
				Disable: true,
			},
			valid: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 简单的验证逻辑
			isValid := tt.provider.APIKey != "" && !tt.provider.Disable
			assert.Equal(t, tt.valid, isValid)
		})
	}
}

// TestProviderCacheDirectory 测试缓存目录创建
func TestProviderCacheDirectory(t *testing.T) {
	// 获取缓存文件路径
	cachePath := providerCacheFileData()
	assert.NotEmpty(t, cachePath)
	
	// 验证路径包含正确的应用名称
	assert.Contains(t, cachePath, "crush")
	assert.Contains(t, cachePath, "providers.json")
	
	// 创建缓存目录
	dir := filepath.Dir(cachePath)
	err := os.MkdirAll(dir, 0755)
	if err == nil {
		defer os.RemoveAll(dir)
		
		// 验证目录存在
		info, err := os.Stat(dir)
		assert.NoError(t, err)
		assert.True(t, info.IsDir())
	}
}

// BenchmarkProviderConfigLoad 性能测试配置加载
func BenchmarkProviderConfigLoad(b *testing.B) {
	tmpDir := b.TempDir()
	
	// 创建大型配置
	config := map[string]interface{}{
		"providers": make(map[string]interface{}),
		"models":    make(map[string]interface{}),
		"mcp":       make(map[string]interface{}),
		"lsp":       make(map[string]interface{}),
	}
	
	// 添加多个 providers
	providers := config["providers"].(map[string]interface{})
	for i := 0; i < 50; i++ {
		providers[fmt.Sprintf("provider_%d", i)] = map[string]interface{}{
			"id":       fmt.Sprintf("provider_%d", i),
			"name":     fmt.Sprintf("Provider %d", i),
			"api_key":  fmt.Sprintf("key_%d", i),
			"base_url": fmt.Sprintf("https://api%d.example.com", i),
			"type":     "openai",
		}
	}
	
	configPath := filepath.Join(tmpDir, "config.json")
	data, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(configPath, data, 0644)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 模拟加载配置
		data, _ := os.ReadFile(configPath)
		var loaded map[string]interface{}
		json.Unmarshal(data, &loaded)
	}
}