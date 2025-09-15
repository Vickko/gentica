package chat

import (
	"context"
	"testing"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// HelloAgent 代表一个基本的 agent 结构
type HelloAgent struct {
	client *openai.Client
	model  string
}

// NewHelloAgent 创建一个新的 HelloAgent
func NewHelloAgent(apiKey, baseURL, model string) *HelloAgent {
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = baseURL

	return &HelloAgent{
		client: openai.NewClientWithConfig(config),
		model:  model,
	}
}

// Execute 执行 agent 的主要功能
func (h *HelloAgent) Execute(ctx context.Context, input string) (string, error) {
	// 发送请求到 OpenAI API
	resp, err := h.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: h.model, // 使用配置的 OpenAI 模型
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: input,
				},
			},
			MaxCompletionTokens: 100,
		},
	)

	if err != nil {
		return "", err
	}

	// 提取响应文本
	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	}

	return "", nil
}

func TestGenkitHelloWorld(t *testing.T) {
	// 创建测试上下文
	ctx := context.Background()

	// 硬编码 API 配置
	apiKey := "sk-6kgtZQDkmZDQMfCo28C360320cEf45FaAf1577Ef08F4032b"
	baseURL := "https://aihubmix.com/v1"

	// 创建 agent（使用 OpenAI 模型）
	agent := NewHelloAgent(apiKey, baseURL, openai.GPT3Dot5Turbo)

	// 测试 Hello World
	t.Run("Send Hello World with Agent", func(t *testing.T) {
		// 执行 agent
		response, err := agent.Execute(ctx, "Say 'Hello World' back to me")

		// 验证没有错误
		require.NoError(t, err)

		// 验证响应不为空
		assert.NotEmpty(t, response)

		// 记录响应
		t.Logf("Agent response: %s", response)
	})
}