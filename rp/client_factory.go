package rp

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino-ext/components/model/gemini"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"google.golang.org/genai"
)

// ModelClientConfig 定义创建 Model Client 所需的配置
type ModelClientConfig struct {
	Model   string
	APIKey  string
	BaseURL string
}

// CreateChatModel 根据 model 名称自动选择并创建对应的 ChatModel client
func CreateChatModel(ctx context.Context, config *ModelClientConfig) (model.ToolCallingChatModel, error) {
	if config == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	modelLower := strings.ToLower(config.Model)

	// 根据 model 名称中的关键词选择对应的 client
	switch {
	case strings.Contains(modelLower, "deepseek"):
		return createDeepSeekClient(ctx, config)
	case strings.Contains(modelLower, "gemini"):
		return createGeminiClient(ctx, config)
	case strings.Contains(modelLower, "claude"):
		return createClaudeClient(ctx, config)
	case strings.Contains(modelLower, "gpt") || strings.Contains(modelLower, "openai"):
		return createOpenAIClient(ctx, config)
	default:
		// 默认使用 OpenAI client（兼容 OpenAI API 格式的服务）
		return createOpenAIClient(ctx, config)
	}
}

// createDeepSeekClient 创建 DeepSeek client
func createDeepSeekClient(ctx context.Context, config *ModelClientConfig) (model.ToolCallingChatModel, error) {
	dsConfig := &deepseek.ChatModelConfig{
		Model:  config.Model,
		APIKey: config.APIKey,
	}

	// 如果指定了 BaseURL，则使用
	if config.BaseURL != "" {
		dsConfig.BaseURL = config.BaseURL
	}

	client, err := deepseek.NewChatModel(ctx, dsConfig)
	if err != nil {
		return nil, fmt.Errorf("创建 DeepSeek client 失败: %w", err)
	}

	return client, nil
}

// createGeminiClient 创建 Gemini client (原生)
func createGeminiClient(ctx context.Context, config *ModelClientConfig) (model.ToolCallingChatModel, error) {
	// 创建 genai.Client
	genaiConfig := &genai.ClientConfig{
		APIKey:  config.APIKey,
	}
	
	// 设置 BaseURL
	if config.BaseURL != "" {
		genaiConfig.HTTPOptions = genai.HTTPOptions{
			BaseURL: config.BaseURL,
		}
	}
	
	client, err := genai.NewClient(ctx, genaiConfig)
	if err != nil {
		return nil, fmt.Errorf("创建 genai client 失败: %w", err)
	}

	geminiConfig := &gemini.Config{
		Model:  config.Model,
		Client: client,
	}

	m, err := gemini.NewChatModel(ctx, geminiConfig)
	if err != nil {
		return nil, fmt.Errorf("创建 Gemini client 失败: %w", err)
	}

	return m, nil
}

// createClaudeClient 创建 Claude client (原生)
func createClaudeClient(ctx context.Context, config *ModelClientConfig) (model.ToolCallingChatModel, error) {
	claudeConfig := &claude.Config{
		Model:     config.Model,
		APIKey:    config.APIKey,
		MaxTokens: 2048, // Claude 需要指定 MaxTokens，给一个默认值
	}

	// 如果指定了 BaseURL，则使用
	if config.BaseURL != "" {
		claudeConfig.BaseURL = &config.BaseURL
	}

	client, err := claude.NewChatModel(ctx, claudeConfig)
	if err != nil {
		return nil, fmt.Errorf("创建 Claude client 失败: %w", err)
	}

	return client, nil
}

// createOpenAIClient 创建 OpenAI client
func createOpenAIClient(ctx context.Context, config *ModelClientConfig) (model.ToolCallingChatModel, error) {
	oaiConfig := &openai.ChatModelConfig{
		Model:  config.Model,
		APIKey: config.APIKey,
	}

	// 如果指定了 BaseURL，则使用
	if config.BaseURL != "" {
		oaiConfig.BaseURL = config.BaseURL
	}

	client, err := openai.NewChatModel(ctx, oaiConfig)
	if err != nil {
		return nil, fmt.Errorf("创建 OpenAI client 失败: %w", err)
	}

	return client, nil
}
