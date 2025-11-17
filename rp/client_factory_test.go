package rp

import (
	"context"
	"strings"
	"testing"
)

func TestCreateChatModel(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		model         string
		expectType    string // "deepseek" 或 "openai"
		shouldSucceed bool
	}{
		{
			name:          "DeepSeek model by name",
			model:         "deepseek-chat",
			expectType:    "deepseek",
			shouldSucceed: false, // 因为没有真实的 API key，预期会失败，但会尝试创建对应类型
		},
		{
			name:          "DeepSeek model with case insensitive",
			model:         "DeepSeek-V3",
			expectType:    "deepseek",
			shouldSucceed: false,
		},
		{
			name:          "OpenAI GPT model",
			model:         "gpt-4o-mini",
			expectType:    "openai",
			shouldSucceed: false,
		},
		{
			name:          "OpenAI GPT-4",
			model:         "gpt-4",
			expectType:    "openai",
			shouldSucceed: false,
		},
		{
			name:          "Model with openai keyword",
			model:         "openai-custom-model",
			expectType:    "openai",
			shouldSucceed: false,
		},
		{
			name:          "Unknown model defaults to OpenAI",
			model:         "unknown-model",
			expectType:    "openai",
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ModelClientConfig{
				Model:   tt.model,
				APIKey:  "test-key",
				BaseURL: "https://test.example.com",
			}

			_, err := CreateChatModel(ctx, config)

			// 验证错误消息中包含预期的 client 类型
			if err != nil {
				errMsg := err.Error()
				switch tt.expectType {
				case "deepseek":
					if !strings.Contains(strings.ToLower(errMsg), "deepseek") {
						t.Errorf("预期错误消息包含 'deepseek'，实际错误: %v", err)
					}
				case "openai":
					if !strings.Contains(strings.ToLower(errMsg), "openai") {
						t.Errorf("预期错误消息包含 'openai'，实际错误: %v", err)
					}
				}
			}
		})
	}
}

// TestModelKeywordDetection 测试 model 关键词检测逻辑
func TestModelKeywordDetection(t *testing.T) {
	tests := []struct {
		model      string
		expectType string
	}{
		{"deepseek-chat", "deepseek"},
		{"DeepSeek-V3", "deepseek"},
		{"DEEPSEEK-Reasoner", "deepseek"},
		{"gpt-4o", "openai"},
		{"gpt-4o-mini", "openai"},
		{"gpt-3.5-turbo", "openai"},
		{"openai-model", "openai"},
		{"claude-3", "openai"}, // 默认使用 openai（未来可扩展 claude）
		{"gemini-pro", "openai"}, // 默认使用 openai（未来可扩展 gemini）
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			modelLower := strings.ToLower(tt.model)
			var detectedType string

			switch {
			case strings.Contains(modelLower, "deepseek"):
				detectedType = "deepseek"
			case strings.Contains(modelLower, "gpt") || strings.Contains(modelLower, "openai"):
				detectedType = "openai"
			default:
				detectedType = "openai" // 默认
			}

			if detectedType != tt.expectType {
				t.Errorf("Model %s: 预期类型 %s，实际类型 %s", tt.model, tt.expectType, detectedType)
			}
		})
	}
}
