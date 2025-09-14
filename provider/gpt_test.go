package provider

import (
	"context"
	"testing"
	"time"

	"gentica/message"
	"gentica/tools"
)

// 辅助函数：获取测试配置
func getTestGPTConfig() (ProviderConfig, *ProviderOptions) {
	// 使用提供的 API 配置
	apiKey := "sk-6kgtZQDkmZDQMfCo28C360320cEf45FaAf1577Ef08F4032b"
	baseURL := "https://aihubmix.com/v1"

	config := ProviderConfig{
		Type:      TypeOpenAI,
		BaseURL:   baseURL,
		APIKey:    apiKey,
		Model:     "gpt-4.1-mini",
		MaxTokens: 4096,
	}

	opts := &ProviderOptions{
		SystemMessage: "You are a helpful assistant.",
		DisableCache:  false,
	}

	return config, opts
}

// TestGPTBasicSetup 测试基础设置
func TestGPTBasicSetup(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	config, opts := getTestGPTConfig()
	client := newOpenAIClient(config, opts)

	if client == nil {
		t.Fatal("Failed to create GPT client")
	}

	model := client.Model()
	if model.ID != config.Model {
		t.Errorf("Expected model ID to be %s, got %s", config.Model, model.ID)
	}

	t.Logf("Successfully created GPT client with model: %s", model.Name)
}

// TestGPTSend 测试同步发送消息
func TestGPTSend(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	config, opts := getTestGPTConfig()
	client := newOpenAIClient(config, opts)

	// 创建测试消息
	messages := []message.Message{
		{
			Role: message.User,
			Parts: []message.ContentPart{
				message.TextContent{Text: "Say 'Hello World' and nothing else."},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 发送消息
	response, err := client.Send(ctx, messages, nil)
	if err != nil {
		t.Logf("Error type: %T", err)
		t.Logf("Error details: %+v", err)
		t.Fatalf("Failed to send message: %v", err)
	}

	// 验证响应
	if response.Content == "" {
		t.Error("Expected non-empty response content")
	}

	if response.Usage.InputTokens == 0 {
		t.Error("Expected input tokens to be greater than 0")
	}

	if response.Usage.OutputTokens == 0 {
		t.Error("Expected output tokens to be greater than 0")
	}

	t.Logf("Response: %s", response.Content)
	t.Logf("Usage - Input tokens: %d, Output tokens: %d",
		response.Usage.InputTokens, response.Usage.OutputTokens)
}

// TestGPTStream 测试流式响应
func TestGPTStream(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	config, opts := getTestGPTConfig()
	client := newOpenAIClient(config, opts)

	// 创建测试消息
	messages := []message.Message{
		{
			Role: message.User,
			Parts: []message.ContentPart{
				message.TextContent{Text: "Count from 1 to 5."},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 获取流式响应
	eventChan := client.Stream(ctx, messages, nil)

	var fullContent string
	var eventCount int
	var hasCompleteEvent bool

	for event := range eventChan {
		eventCount++

		switch event.Type {
		case EventContentDelta:
			fullContent += event.Content
			t.Logf("Content delta: %s", event.Content)
		case EventComplete:
			hasCompleteEvent = true
			if event.Response != nil {
				t.Logf("Complete - Usage: Input=%d, Output=%d",
					event.Response.Usage.InputTokens,
					event.Response.Usage.OutputTokens)
			}
		case EventError:
			t.Fatalf("Stream error: %v", event.Error)
		}
	}

	if !hasCompleteEvent {
		t.Error("Expected to receive a complete event")
	}

	if fullContent == "" {
		t.Error("Expected non-empty streamed content")
	}

	if eventCount == 0 {
		t.Error("Expected to receive at least one event")
	}

	t.Logf("Full streamed content: %s", fullContent)
	t.Logf("Total events received: %d", eventCount)
}

// TestGPTWithTools 测试工具调用
func TestGPTWithTools(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	config, opts := getTestGPTConfig()
	client := newOpenAIClient(config, opts)

	// 创建一个简单的工具 - 只包含 properties，不需要外层的 type: object
	weatherTool := &mockGPTTool{
		name:        "get_weather",
		description: "Get the current weather for a location",
		parameters: map[string]any{
			"location": map[string]any{
				"type":        "string",
				"description": "The city and state, e.g. San Francisco, CA",
			},
		},
		required: []string{"location"},
	}

	// 创建测试消息
	messages := []message.Message{
		{
			Role: message.User,
			Parts: []message.ContentPart{
				message.TextContent{Text: "What's the weather in San Francisco?"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 发送带工具的消息
	response, err := client.Send(ctx, messages, []tools.BaseTool{weatherTool})
	if err != nil {
		t.Fatalf("Failed to send message with tools: %v", err)
	}

	// 验证响应
	if len(response.ToolCalls) > 0 {
		for _, toolCall := range response.ToolCalls {
			t.Logf("Tool call: %s (ID: %s)", toolCall.Name, toolCall.ID)
			t.Logf("Tool input: %s", toolCall.Input)
		}
	} else {
		// 即使没有工具调用，也应该有文本响应
		if response.Content == "" {
			t.Error("Expected either tool calls or text content")
		}
		t.Logf("Response (no tool calls): %s", response.Content)
	}
}

// TestGPTErrorHandling 测试错误处理
func TestGPTErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 测试无效的 API key
	config, opts := getTestGPTConfig()
	config.APIKey = "invalid-api-key"
	client := newOpenAIClient(config, opts)

	messages := []message.Message{
		{
			Role: message.User,
			Parts: []message.ContentPart{
				message.TextContent{Text: "Hello"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	response, err := client.Send(ctx, messages, nil)
	// 注意：某些代理可能不会正确验证 API key
	if err == nil {
		if response != nil && response.Content != "" {
			t.Log("Warning: Invalid API key was accepted by the provider, got response:", response.Content)
			t.Skip("Provider does not validate API keys properly")
		} else {
			t.Error("Expected error with invalid API key, but got nil error and empty response")
		}
	} else {
		t.Logf("Got expected error with invalid API key: %v", err)
	}
}

// TestGPTContextCancellation 测试上下文取消
func TestGPTContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	config, opts := getTestGPTConfig()
	client := newOpenAIClient(config, opts)

	messages := []message.Message{
		{
			Role: message.User,
			Parts: []message.ContentPart{
				message.TextContent{Text: "Write a very long story about a robot."},
			},
		},
	}

	// 创建一个很短的超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// 等待一下确保上下文已取消
	time.Sleep(2 * time.Millisecond)

	eventChan := client.Stream(ctx, messages, nil)

	hasError := false
	for event := range eventChan {
		if event.Type == EventError {
			hasError = true
			t.Logf("Got expected context cancellation error: %v", event.Error)
			break
		}
	}

	if !hasError {
		t.Error("Expected context cancellation error")
	}
}

// TestGPTMultiTurn 测试多轮对话
func TestGPTMultiTurn(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	config, opts := getTestGPTConfig()
	client := newOpenAIClient(config, opts)

	// 创建多轮对话消息
	messages := []message.Message{
		{
			Role: message.User,
			Parts: []message.ContentPart{
				message.TextContent{Text: "My name is Alice."},
			},
		},
		{
			Role: message.Assistant,
			Parts: []message.ContentPart{
				message.TextContent{Text: "Nice to meet you, Alice! How can I help you today?"},
			},
		},
		{
			Role: message.User,
			Parts: []message.ContentPart{
				message.TextContent{Text: "What's my name?"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 发送消息
	response, err := client.Send(ctx, messages, nil)
	if err != nil {
		t.Fatalf("Failed to send multi-turn message: %v", err)
	}

	// 验证响应
	if response.Content == "" {
		t.Error("Expected non-empty response content")
	}

	// 检查响应是否包含名字 "Alice"
	if !containsString(response.Content, "Alice") {
		t.Logf("Response doesn't mention 'Alice': %s", response.Content)
		t.Log("Note: Model may not have correctly maintained context")
	} else {
		t.Logf("Response correctly identifies the name: %s", response.Content)
	}
}

// mockGPTTool 实现 tools.BaseTool 接口用于测试
type mockGPTTool struct {
	name        string
	description string
	parameters  map[string]any
	required    []string
}

func (m *mockGPTTool) Info() tools.ToolInfo {
	return tools.ToolInfo{
		Name:        m.name,
		Description: m.description,
		Parameters:  m.parameters, // 直接返回 properties，不需要外层包装
		Required:    m.required,
	}
}

func (m *mockGPTTool) Name() string {
	return m.name
}

func (m *mockGPTTool) Run(_ context.Context, _ tools.ToolCall) (tools.ToolResponse, error) {
	return tools.ToolResponse{
		Type:    "text",
		Content: "Mock weather: Sunny, 72°F",
		IsError: false,
	}, nil
}
