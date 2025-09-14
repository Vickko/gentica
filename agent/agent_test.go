package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"gentica/message"
	"gentica/provider"
	"gentica/tools"
)

// WeatherTool - 复用 provider 测试中已验证的天气工具
type WeatherTool struct{}

func (w *WeatherTool) Info() tools.ToolInfo {
	return tools.ToolInfo{
		Name:        "get_weather",
		Description: "Get the current weather for a location",
		Parameters: map[string]any{
			"location": map[string]any{
				"type":        "string",
				"description": "The city and state, e.g. San Francisco, CA",
			},
		},
		Required: []string{"location"},
	}
}

func (w *WeatherTool) Name() string {
	return "get_weather"
}

func (w *WeatherTool) Run(_ context.Context, params tools.ToolCall) (tools.ToolResponse, error) {
	var weatherParams struct {
		Location string `json:"location"`
	}

	if err := json.Unmarshal([]byte(params.Input), &weatherParams); err != nil {
		return tools.NewTextResponse(fmt.Sprintf("Error parsing parameters: %v", err)), err
	}

	// 模拟返回天气信息
	result := fmt.Sprintf("Weather in %s: Sunny, 72°F", weatherParams.Location)
	return tools.NewTextResponse(result), nil
}

// TimeTool - 获取当前时间的工具
type TimeTool struct{}

func (t *TimeTool) Info() tools.ToolInfo {
	return tools.ToolInfo{
		Name:        "get_current_time",
		Description: "Get the current time in a specific timezone",
		Parameters: map[string]any{
			"timezone": map[string]any{
				"type":        "string",
				"description": "The timezone, e.g. UTC, PST, EST",
			},
		},
		Required: []string{"timezone"},
	}
}

func (t *TimeTool) Name() string {
	return "get_current_time"
}

func (t *TimeTool) Run(_ context.Context, params tools.ToolCall) (tools.ToolResponse, error) {
	var timeParams struct {
		Timezone string `json:"timezone"`
	}

	if err := json.Unmarshal([]byte(params.Input), &timeParams); err != nil {
		return tools.NewTextResponse(fmt.Sprintf("Error parsing parameters: %v", err)), err
	}

	// 模拟返回时间信息
	currentTime := time.Now().Format("15:04:05")
	result := fmt.Sprintf("Current time in %s: %s", timeParams.Timezone, currentTime)
	return tools.NewTextResponse(result), nil
}

// 获取测试用的 provider 配置
func getTestProviderConfig() provider.ProviderConfig {
	// 使用硬编码的测试 API 配置
	apiKey := "sk-6kgtZQDkmZDQMfCo28C360320cEf45FaAf1577Ef08F4032b"
	baseURL := "https://aihubmix.com/v1"

	return provider.ProviderConfig{
		Type:      provider.TypeOpenAI,
		BaseURL:   baseURL,
		APIKey:    apiKey,
		Model:     "gpt-4.1-mini",
		MaxTokens: 4096,
	}
}

// 获取测试用的真实 provider (向后兼容)
func getTestProvider(t *testing.T) provider.Provider {
	cfg := getTestProviderConfig()
	opts := &provider.ProviderOptions{
		SystemMessage: "You are a helpful AI assistant.",
	}

	p, err := provider.NewProvider(cfg, opts)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	return p
}

// Test 1: 简单聊天测试
func TestSimpleChat(t *testing.T) {
	providerConfig := getTestProviderConfig()

	config := Config{
		Name:         "test-agent",
		SystemPrompt: "You are a helpful assistant. You must add meow～ at the end of every sentence.",
		Model:        "gpt-4.1-mini",
	}

	agent, err := NewAgent(config, providerConfig, nil)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	messages := []message.Message{
		{
			Role: message.User,
			Parts: []message.ContentPart{
				message.TextContent{Text: "What is 2 + 2? Just give me the number."},
			},
		},
	}

	ctx := context.Background()
	response, err := agent.Run(ctx, messages)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	// 打印响应以便观察
	for _, part := range response.Parts {
		if tc, ok := part.(message.TextContent); ok {
			t.Logf("Agent response: %s", tc.Text)
		}
	}
}

// Test 2: 使用工具组合完成任务
func TestToolCombination(t *testing.T) {
	providerConfig := getTestProviderConfig()

	config := Config{
		Name:         "test-agent-with-tools",
		SystemPrompt: "You are a helpful assistant that can use tools to complete tasks.",
		Model:        "gpt-4.1-mini",
	}

	// 创建工具集
	weatherTool := &WeatherTool{}
	timeTool := &TimeTool{}

	toolsList := []tools.BaseTool{
		weatherTool,
		timeTool,
	}

	agent, err := NewAgent(config, providerConfig, toolsList)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// 给 agent 一个需要组合使用工具的任务
	messages := []message.Message{
		{
			Role: message.User,
			Parts: []message.ContentPart{
				message.TextContent{
					Text: "What's the weather in San Francisco and what time is it in UTC?",
				},
			},
		},
	}

	ctx := context.Background()
	response, err := agent.Run(ctx, messages)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	// 打印最终响应
	t.Log("=== Final Agent Response ===")
	for _, part := range response.Parts {
		if tc, ok := part.(message.TextContent); ok {
			t.Logf("Agent final response:\n%s", tc.Text)
		}
	}
}

// Test 3: 测试单个工具调用
func TestSingleToolCall(t *testing.T) {
	providerConfig := getTestProviderConfig()

	config := Config{
		Name:         "test-agent-single-tool",
		SystemPrompt: "You are a helpful assistant. Use the available tools to answer questions.",
		Model:        "gpt-4.1-mini",
	}

	weatherTool := &WeatherTool{}
	agent, err := NewAgent(config, providerConfig, []tools.BaseTool{weatherTool})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	messages := []message.Message{
		{
			Role: message.User,
			Parts: []message.ContentPart{
				message.TextContent{Text: "What's the weather in New York?"},
			},
		},
	}

	ctx := context.Background()
	response, err := agent.Run(ctx, messages)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	// 打印响应
	t.Log("=== Single Tool Response ===")
	for _, part := range response.Parts {
		if tc, ok := part.(message.TextContent); ok {
			t.Logf("Response: %s", tc.Text)
		}
	}
}

// Test 4: 测试多轮对话
func TestMultiTurnConversation(t *testing.T) {
	providerConfig := getTestProviderConfig()

	config := Config{
		Name:         "multi-turn-agent",
		SystemPrompt: "You are a helpful assistant. Remember context from previous messages.",
		Model:        "gpt-4.1-mini",
	}

	agent, err := NewAgent(config, providerConfig, nil)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// 第一轮对话
	messages := []message.Message{
		{
			Role: message.User,
			Parts: []message.ContentPart{
				message.TextContent{Text: "My name is Alice."},
			},
		},
	}

	ctx := context.Background()
	response1, err := agent.Run(ctx, messages)
	if err != nil {
		t.Fatalf("First turn error: %v", err)
	}

	t.Log("=== First Turn Response ===")
	for _, part := range response1.Parts {
		if tc, ok := part.(message.TextContent); ok {
			t.Logf("Response: %s", tc.Text)
		}
	}

	// 添加响应到消息历史
	messages = append(messages, *response1)

	// 第二轮对话 - 引用之前的内容
	messages = append(messages, message.Message{
		Role: message.User,
		Parts: []message.ContentPart{
			message.TextContent{Text: "What's my name?"},
		},
	})

	response2, err := agent.Run(ctx, messages)
	if err != nil {
		t.Fatalf("Second turn error: %v", err)
	}

	t.Log("=== Second Turn Response ===")
	for _, part := range response2.Parts {
		if tc, ok := part.(message.TextContent); ok {
			t.Logf("Response: %s", tc.Text)
			// 检查是否记住了名字
			if !contains(tc.Text, "Alice") {
				t.Log("Note: Response doesn't mention 'Alice', context may not be maintained")
			}
		}
	}
}

// Test 5: 测试 Chat 方法和消息历史管理
func TestChatMethodWithHistory(t *testing.T) {
	providerConfig := getTestProviderConfig()

	config := Config{
		Name:         "chat-agent",
		SystemPrompt: "You are a helpful assistant. Remember all context from our conversation.",
		Model:        "gpt-4.1-mini",
	}

	agent, err := NewAgent(config, providerConfig, nil)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	ctx := context.Background()

	// 第一次聊天
	response1, err := agent.Chat(ctx, "My favorite color is blue.")
	if err != nil {
		t.Fatalf("First chat error: %v", err)
	}

	t.Log("=== First Chat Response ===")
	for _, part := range response1.Parts {
		if tc, ok := part.(message.TextContent); ok {
			t.Logf("Response: %s", tc.Text)
		}
	}

	// 第二次聊天 - 应该记住之前的内容
	response2, err := agent.Chat(ctx, "What is my favorite color?")
	if err != nil {
		t.Fatalf("Second chat error: %v", err)
	}

	t.Log("=== Second Chat Response ===")
	for _, part := range response2.Parts {
		if tc, ok := part.(message.TextContent); ok {
			t.Logf("Response: %s", tc.Text)
			if !contains(tc.Text, "blue") {
				t.Log("Note: Response doesn't mention 'blue', context may not be maintained")
			}
		}
	}

	// 测试获取历史
	history := agent.GetHistory()
	if len(history) != 4 { // 2 user messages + 2 assistant responses
		t.Errorf("Expected 4 messages in history, got %d", len(history))
	}

	// 测试清空历史
	agent.ClearHistory()
	history = agent.GetHistory()
	if len(history) != 0 {
		t.Errorf("Expected empty history after clear, got %d messages", len(history))
	}
}

// Test 6: 测试向后兼容的 NewAgentWithProvider
func TestBackwardCompatibility(t *testing.T) {
	p := getTestProvider(t)

	config := Config{
		Name:         "backward-compat-agent",
		SystemPrompt: "You are a test agent.",
		Model:        "gpt-4.1-mini",
	}

	// 使用向后兼容的构造函数
	agent := NewAgentWithProvider(config, p, nil)

	messages := []message.Message{
		{
			Role: message.User,
			Parts: []message.ContentPart{
				message.TextContent{Text: "Hello"},
			},
		},
	}

	ctx := context.Background()
	response, err := agent.Run(ctx, messages)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	t.Log("Backward compatibility test passed")
}

// 辅助函数：检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (len(substr) == 0 || containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
