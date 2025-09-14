package provider

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gentica/message"
	"gentica/tools"
)

// 辅助函数：获取测试配置
func getTestGeminiConfig() (ProviderConfig, *ProviderOptions) {
	// 使用 aihubmix 的 Gemini 端点 - 需要完整路径
	apiKey := "sk-6kgtZQDkmZDQMfCo28C360320cEf45FaAf1577Ef08F4032b"
	baseURL := "https://aihubmix.com/gemini" // 用户需要自己提供完整路径

	config := ProviderConfig{
		Type:    TypeGemini,
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   "gemini-2.5-flash",
	}

	opts := &ProviderOptions{
		SystemMessage: "You are a helpful assistant.",
		DisableCache:  false,
	}

	return config, opts
}

// TestGeminiBasicSetup 测试基础设置
func TestGeminiBasicSetup(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	config, opts := getTestGeminiConfig()
	client := newGeminiClient(config, opts)

	if client == nil {
		t.Fatal("Failed to create Gemini client")
	}

	model := client.Model()
	if model.ID != config.Model {
		t.Errorf("Expected model ID to be %s, got %s", config.Model, model.ID)
	}

	t.Logf("Successfully created Gemini client with model: %s", model.Name)
}

// TestGeminiSend 测试同步发送消息
func TestGeminiSend(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	config, opts := getTestGeminiConfig()
	client := newGeminiClient(config, opts)

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

// TestGeminiStream 测试流式响应
func TestGeminiStream(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	config, opts := getTestGeminiConfig()
	client := newGeminiClient(config, opts)

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

// TestGeminiWithTools 测试工具调用
func TestGeminiWithTools(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	config, opts := getTestGeminiConfig()
	client := newGeminiClient(config, opts)

	// 创建一个简单的工具 - 注意参数结构应该只包含 properties，不需要外层的 type: object
	weatherTool := &mockGeminiTool{
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

	// 打印工具信息用于调试
	toolInfo := weatherTool.Info()
	t.Logf("Tool info - Name: %s, Description: %s", toolInfo.Name, toolInfo.Description)
	t.Logf("Tool parameters: %+v", toolInfo.Parameters)
	t.Logf("Tool required: %+v", toolInfo.Required)

	// 创建测试消息 - 更明确地请求工具调用
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

// TestGeminiErrorHandling 测试错误处理
func TestGeminiErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 测试无效的 API key
	config, opts := getTestGeminiConfig()
	config.APIKey = "invalid-api-key"
	client := newGeminiClient(config, opts)

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

// TestGeminiContextCancellation 测试上下文取消
func TestGeminiContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	config, opts := getTestGeminiConfig()
	client := newGeminiClient(config, opts)

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

// TestGeminiLongStreamStability 测试长流连接稳定性
func TestGeminiLongStreamStability(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	config, opts := getTestGeminiConfig()
	client := newGeminiClient(config, opts)

	// 创建测试消息 - 请求写一篇5000字的文章
	messages := []message.Message{
		{
			Role: message.User,
			Parts: []message.ContentPart{
				message.TextContent{Text: "请写一篇关于人工智能发展历程的详细文章，至少5000字，包含以下内容：1. AI的起源和早期发展 2. 机器学习的兴起 3. 深度学习革命 4. 现代AI应用 5. 未来展望。请详细展开每个部分，提供具体例子和技术细节。"},
			},
		},
	}

	// 设置较长的超时时间，预期长文本生成需要更多时间
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 记录开始时间
	startTime := time.Now()
	t.Logf("开始长流测试，时间: %v", startTime.Format("15:04:05.000"))

	// 获取流式响应
	eventChan := client.Stream(ctx, messages, nil)

	var fullContent string
	var eventCount int
	var contentDeltaCount int
	var hasCompleteEvent bool
	var lastEventTime time.Time = startTime
	var maxEventGap time.Duration
	var totalEventGap time.Duration
	var firstContentTime time.Time
	var lastContentTime time.Time
	var finishReason message.FinishReason // 添加完成原因跟踪

	// 监控流事件
	for event := range eventChan {
		currentTime := time.Now()
		eventCount++

		// 计算事件间隔
		if !lastEventTime.IsZero() {
			eventGap := currentTime.Sub(lastEventTime)
			totalEventGap += eventGap
			if eventGap > maxEventGap {
				maxEventGap = eventGap
			}

			// 如果事件间隔过长，记录警告
			if eventGap > 10*time.Second {
				t.Logf("警告: 事件间隔过长 %v，时间: %v", eventGap, currentTime.Format("15:04:05.000"))
			}
		}
		lastEventTime = currentTime

		switch event.Type {
		case EventContentDelta:
			contentDeltaCount++
			fullContent += event.Content

			// 记录首次和最后一次内容时间
			if firstContentTime.IsZero() {
				firstContentTime = currentTime
				t.Logf("收到首个内容块，时间: %v, 延迟: %v",
					currentTime.Format("15:04:05.000"),
					currentTime.Sub(startTime))
			}
			lastContentTime = currentTime

			// 每100个内容块打印一次进度
			if contentDeltaCount%100 == 0 {
				elapsed := currentTime.Sub(startTime)
				t.Logf("进度: 收到 %d 个内容块, 总字符数: %d, 已用时: %v",
					contentDeltaCount, len(fullContent), elapsed)
			}

		case EventComplete:
			hasCompleteEvent = true
			completeTime := currentTime
			totalDuration := completeTime.Sub(startTime)

			t.Logf("流完成，时间: %v, 总用时: %v",
				completeTime.Format("15:04:05.000"), totalDuration)

			if event.Response != nil {
				finishReason = event.Response.FinishReason // 记录完成原因
				t.Logf("完成 - 用量统计: 输入=%d tokens, 输出=%d tokens, 完成原因: %v",
					event.Response.Usage.InputTokens,
					event.Response.Usage.OutputTokens,
					finishReason)

				// 计算生成速度
				if totalDuration > 0 && event.Response.Usage.OutputTokens > 0 {
					tokensPerSecond := float64(event.Response.Usage.OutputTokens) / totalDuration.Seconds()
					t.Logf("生成速度: %.2f tokens/秒", tokensPerSecond)
				}
			}

		case EventError:
			errorTime := currentTime
			t.Logf("流错误，时间: %v, 错误: %v", errorTime.Format("15:04:05.000"), event.Error)
			t.Fatalf("长流测试失败: %v", event.Error)
		}
	}

	// 计算总体统计
	endTime := time.Now()
	totalDuration := endTime.Sub(startTime)

	// 验证结果
	if !hasCompleteEvent {
		t.Error("期望收到完成事件")
	}

	if fullContent == "" {
		t.Error("期望收到非空的流式内容")
	}

	if eventCount == 0 {
		t.Error("期望至少收到一个事件")
	}

	// 字符数检查 - 5000字大约对应10000-15000个字符（包括标点符号等）
	contentLength := len(fullContent)
	if contentLength < 8000 {
		t.Logf("警告: 内容长度 %d 字符可能少于预期的5000字", contentLength)
	}

	// 计算平均事件间隔
	var avgEventGap time.Duration
	if eventCount > 1 {
		avgEventGap = totalEventGap / time.Duration(eventCount-1)
	}

	// 输出详细统计
	t.Logf("=== 长流稳定性测试统计 ===")
	t.Logf("总用时: %v", totalDuration)
	t.Logf("内容总长度: %d 字符", contentLength)
	t.Logf("总事件数: %d", eventCount)
	t.Logf("内容块数: %d", contentDeltaCount)
	t.Logf("最大事件间隔: %v", maxEventGap)
	t.Logf("平均事件间隔: %v", avgEventGap)
	t.Logf("完成原因: %v", finishReason)

	// 检查是否因为达到 token 限制而截断
	if finishReason == message.FinishReasonMaxTokens {
		t.Errorf("内容被截断：达到最大 token 限制 (%d tokens)", config.MaxTokens)
	}

	if !firstContentTime.IsZero() {
		t.Logf("首次响应延迟: %v", firstContentTime.Sub(startTime))
	}

	if !lastContentTime.IsZero() && !firstContentTime.IsZero() {
		streamingDuration := lastContentTime.Sub(firstContentTime)
		t.Logf("流式传输持续时间: %v", streamingDuration)

		if streamingDuration > 0 {
			charsPerSecond := float64(contentLength) / streamingDuration.Seconds()
			t.Logf("流式传输速度: %.2f 字符/秒", charsPerSecond)
		}
	}

	// 稳定性检查
	if maxEventGap > 30*time.Second {
		t.Errorf("连接稳定性问题: 最大事件间隔 %v 超过30秒", maxEventGap)
	}

	if contentDeltaCount == 0 {
		t.Error("连接问题: 未收到任何内容块")
	}

	t.Logf("长流稳定性测试完成")
	fmt.Println(fullContent)
}

// mockGeminiTool 实现 tools.BaseTool 接口用于测试
type mockGeminiTool struct {
	name        string
	description string
	parameters  map[string]any
	required    []string
}

func (m *mockGeminiTool) Info() tools.ToolInfo {
	return tools.ToolInfo{
		Name:        m.name,
		Description: m.description,
		Parameters:  m.parameters, // 直接返回 properties，不需要外层包装
		Required:    m.required,
	}
}

func (m *mockGeminiTool) Name() string {
	return m.name
}

func (m *mockGeminiTool) Run(_ context.Context, _ tools.ToolCall) (tools.ToolResponse, error) {
	return tools.ToolResponse{
		Type:    "text",
		Content: "Mock weather: Sunny, 72°F",
		IsError: false,
	}, nil
}
