package agent

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"gentica/tools"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
	openaiGo "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	// 测试配置
	apiKey  = "sk-6kgtZQDkmZDQMfCo28C360320cEf45FaAf1577Ef08F4032b"
	baseURL = "https://aihubmix.com/v1"
)

var g *genkit.Genkit

func TestMain(m *testing.M) {
	// 初始化 Genkit
	oai := &openai.OpenAI{
		APIKey: apiKey,
		Opts: []option.RequestOption{
			option.WithBaseURL(baseURL),
		},
	}

	g = genkit.Init(
		context.Background(),
		genkit.WithPlugins(oai),
	)

	os.Exit(m.Run())
}

func TestSimpleAgent(t *testing.T) {
	// 创建一个简单的 Agent
	agent := NewSimpleAgent(g, "test_assistant", "You are a helpful assistant. Answer concisely.")

	// 测试基本信息
	assert.Equal(t, "test_assistant", agent.Name())
	assert.Equal(t, "test_assistant", agent.Description())

	// 测试执行
	ctx := context.Background()
	result, err := agent.Run(ctx, "What is 2+2?")
	require.NoError(t, err)
	require.NotEmpty(t, result)

	t.Logf("Simple Agent response: %s", result)

	// 验证消息历史
	messages := agent.GetMessages()
	assert.GreaterOrEqual(t, len(messages), 2) // 至少有用户消息和助手回复
}

func TestAgentBuilder(t *testing.T) {
	// 使用 Builder 创建 Agent
	agent := NewBuilder(
		g,
		"code_reviewer",
		"Reviews code and provides feedback",
		"You are a code review expert. Analyze the provided code and give constructive feedback.",
	).WithInputSchema(
		map[string]any{
			"code": map[string]any{
				"type":        "string",
				"description": "Code to review",
			},
			"language": map[string]any{
				"type":        "string",
				"description": "Programming language",
			},
		},
		"code", // 必需字段
	).WithModel("openai/" + string(openaiGo.ChatModelGPT4oMini)).
		WithTemperature(0.3).
		WithMaxTokens(500).
		Build()

	// 测试配置
	config := agent.GetConfig()
	assert.Equal(t, "code_reviewer", config.Name)
	assert.Equal(t, "Reviews code and provides feedback", config.Description)
	assert.Contains(t, config.Required, "code")
	assert.Equal(t, float32(0.3), config.Temperature)

	// 测试带参数的执行
	ctx := context.Background()
	input := map[string]any{
		"code":     "func add(a, b int) int { return a + b }",
		"language": "go",
	}

	inputJSON, _ := json.Marshal(input)
	result, err := agent.Run(ctx, string(inputJSON))
	require.NoError(t, err)
	require.NotEmpty(t, result)

	t.Logf("Code review result: %s", result)
}

func TestAgentWithTools(t *testing.T) {
	// 获取工作目录
	workingDir, err := os.Getwd()
	require.NoError(t, err)

	// 创建工具
	treeTool := tools.NewTreeTool(workingDir)
	genkitTreeTool := tools.AdaptBaseToolToGenkit(g, treeTool)

	// 创建带工具的 Agent
	agent := NewBuilder(
		g,
		"file_explorer",
		"Explores file system structure",
		"You are a file system explorer. Use the tree tool to explore directories and answer questions about file structure.",
	).WithTools(genkitTreeTool).
		WithModel("openai/" + string(openaiGo.ChatModelGPT4oMini)).
		WithMaxRounds(3).
		Build()

	// 测试执行
	ctx := context.Background()
	result, err := agent.Run(ctx, "List the files in the current directory")
	require.NoError(t, err)
	require.NotEmpty(t, result)

	t.Logf("File explorer result: %s", result)

	// 检查是否有工具调用
	messages := agent.GetMessages()
	hasToolCall := false
	for _, msg := range messages {
		if msg.Role == ai.RoleModel {
			for _, part := range msg.Content {
				if part.IsToolRequest() {
					hasToolCall = true
					t.Logf("Tool called: %s", part.ToolRequest.Name)
				}
			}
		}
	}
	assert.True(t, hasToolCall, "Agent should have called the tree tool")
}

func TestAgentAsToolAdapter(t *testing.T) {
	// 创建一个 Agent
	translatorAgent := NewBuilder(
		g,
		"translator",
		"Translates text to specified language",
		"You are a translator. Translate the provided text to the specified language.",
	).WithInputSchema(
		map[string]any{
			"text": map[string]any{
				"type":        "string",
				"description": "Text to translate",
			},
			"target_language": map[string]any{
				"type":        "string",
				"description": "Target language",
			},
		},
		"text", "target_language",
	).WithModel("openai/" + string(openaiGo.ChatModelGPT4oMini)).
		Build()

	// 转换为 Tool
	translatorTool := AsToolAdapter(translatorAgent)

	// 测试 Tool 信息
	info := translatorTool.Info()
	assert.Equal(t, "translator", info.Name)
	assert.Equal(t, "Translates text to specified language", info.Description)
	assert.Contains(t, info.Required, "text")
	assert.Contains(t, info.Required, "target_language")

	// 测试 Tool 执行
	ctx := context.Background()
	toolCall := tools.ToolCall{
		Name: "translator",
		Input: `{
			"text": "Hello, world!",
			"target_language": "Chinese"
		}`,
	}

	response, err := translatorTool.Run(ctx, toolCall)
	require.NoError(t, err)
	assert.False(t, response.IsError)
	assert.NotEmpty(t, response.Content)

	t.Logf("Translation result: %s", response.Content)
}

func TestAgentChain(t *testing.T) {
	// 创建多个 Agent
	agent1 := NewSimpleAgent(g, "summarizer", "You are a summarizer. Summarize the input in one sentence.")
	agent2 := NewSimpleAgent(g, "translator", "You are a translator. Translate the input to Chinese.")

	// 创建 Agent 链
	chain := NewAgentChain("summarize_and_translate", agent1, agent2)

	// 测试链式执行
	ctx := context.Background()
	input := "The quick brown fox jumps over the lazy dog. This is a classic pangram that contains all letters of the English alphabet."
	result, err := chain.Run(ctx, input)
	require.NoError(t, err)
	require.NotEmpty(t, result)

	t.Logf("Chain result: %s", result)
}

func TestAgentPool(t *testing.T) {
	// 创建多个 Agent
	agent1 := NewSimpleAgent(g, "analyzer1", "You are analyzer 1. Analyze from a technical perspective.")
	agent2 := NewSimpleAgent(g, "analyzer2", "You are analyzer 2. Analyze from a business perspective.")

	// 创建 Agent 池
	pool := NewAgentPool("multi_analyzer", agent1, agent2)

	// 测试并行执行
	ctx := context.Background()
	input := "Should we migrate our monolithic application to microservices?"
	results, err := pool.Run(ctx, input)
	require.NoError(t, err)
	require.Len(t, results, 2)

	for i, result := range results {
		t.Logf("Agent %d result: %s", i+1, result)
	}

	// 测试带详细信息的执行
	detailedResults := pool.RunWithDetails(ctx, input)
	require.Len(t, detailedResults, 2)

	for _, result := range detailedResults {
		require.NoError(t, result.Error)
		t.Logf("%s: %s", result.AgentName, result.Result)
	}
}

func TestMessageManager(t *testing.T) {
	mm := NewMessageManager(
		WithWindowSize(5),
		WithTokenLimit(1000),
	)

	// 创建测试消息
	systemMsg := &ai.Message{
		Role:    ai.RoleSystem,
		Content: []*ai.Part{ai.NewTextPart("System prompt")},
	}
	messages := []*ai.Message{
		systemMsg,
		ai.NewUserTextMessage("Message 1"),
		ai.NewModelTextMessage("Response 1"),
		ai.NewUserTextMessage("Message 2"),
		ai.NewModelTextMessage("Response 2"),
		ai.NewUserTextMessage("Message 3"),
		ai.NewModelTextMessage("Response 3"),
	}

	// 测试截断
	truncated := mm.TruncateMessages(messages, 4)
	assert.Len(t, truncated, 4)
	assert.Equal(t, ai.RoleSystem, truncated[0].Role) // 系统消息保留

	// 测试按角色过滤
	userMessages := mm.FilterByRole(messages, ai.RoleUser)
	assert.Len(t, userMessages, 3)
	for _, msg := range userMessages {
		assert.Equal(t, ai.RoleUser, msg.Role)
	}

	// 测试 token 估算
	tokens := mm.EstimateTokens(messages)
	assert.Greater(t, tokens, 0)
}

func TestAgentRouter(t *testing.T) {
	// 创建多个专门的 Agent
	mathAgent := NewSimpleAgent(g, "math_agent", "You are a math expert. Answer math questions.")
	historyAgent := NewSimpleAgent(g, "history_agent", "You are a history expert. Answer history questions.")

	// 创建路由函数
	routerFunc := func(ctx context.Context, input string) (string, error) {
		// 简单的关键词路由
		if containsAny(input, "math", "calculate", "number", "equation") {
			return "math", nil
		}
		if containsAny(input, "history", "historical", "past", "century") {
			return "history", nil
		}
		return "math", nil // 默认路由
	}

	// 创建路由器
	router := NewAgentRouter("subject_router", routerFunc)
	router.AddRoute("math", mathAgent)
	router.AddRoute("history", historyAgent)

	// 测试数学问题路由
	ctx := context.Background()
	mathResult, err := router.Run(ctx, "What is the derivative of x^2?")
	require.NoError(t, err)
	require.NotEmpty(t, mathResult)
	t.Logf("Math question result: %s", mathResult)

	// 测试历史问题路由
	historyResult, err := router.Run(ctx, "What happened in the year 1066?")
	require.NoError(t, err)
	require.NotEmpty(t, historyResult)
	t.Logf("History question result: %s", historyResult)
}

func TestAgentPipeline(t *testing.T) {
	// 创建管道中的 Agent
	validatorAgent := NewSimpleAgent(g, "validator", "You are a validator. Check if the input is valid JSON. Reply 'valid' or 'invalid'.")
	parserAgent := NewSimpleAgent(g, "parser", "You are a JSON parser. Extract the 'name' field from the JSON.")

	// 创建管道
	pipeline := NewAgentPipeline("json_processor").
		AddStep(validatorAgent, nil, nil).
		AddStep(
			parserAgent,
			// 只有在验证通过时才执行解析
			func(ctx context.Context, input string, previousResult string) bool {
				return containsAny(previousResult, "valid")
			},
			// 转换函数：将原始输入传递给解析器
			func(previousResult string) string {
				return `{"name": "John Doe", "age": 30}`
			},
		)

	// 测试管道执行
	ctx := context.Background()
	result, err := pipeline.Run(ctx, `{"name": "John Doe", "age": 30}`)
	require.NoError(t, err)
	require.NotEmpty(t, result)
	t.Logf("Pipeline result: %s", result)
}

func TestAgentMessageManagement(t *testing.T) {
	// 创建 Agent
	agent := NewSimpleAgent(g, "chat_agent", "You are a helpful chat assistant.")

	ctx := context.Background()

	// 进行多轮对话
	_, err := agent.Run(ctx, "Hello!")
	require.NoError(t, err)

	_, err = agent.Run(ctx, "What's the weather like?")
	require.NoError(t, err)

	// 检查消息历史
	messages := agent.GetMessages()
	assert.GreaterOrEqual(t, len(messages), 4) // 至少有2轮对话

	// 测试清空历史
	agent.ClearHistory()
	messages = agent.GetMessages()
	assert.Len(t, messages, 0)

	// 测试手动设置消息
	customMessages := []*ai.Message{
		ai.NewUserTextMessage("Custom message 1"),
		ai.NewModelTextMessage("Custom response 1"),
	}
	agent.SetMessages(customMessages)

	messages = agent.GetMessages()
	assert.Len(t, messages, 2)
	assert.Equal(t, "Custom message 1", messages[0].Text())
}

func TestStatefulVsStatelessToolAdapter(t *testing.T) {
	// 创建 Agent
	agent := NewSimpleAgent(g, "memory_agent", "You are an assistant with memory. Remember what the user tells you.")

	ctx := context.Background()

	// 测试无状态适配器
	statelessTool := AsToolAdapter(agent)

	// 第一次调用
	call1 := tools.ToolCall{Name: "memory_agent", Input: "My name is Alice."}
	resp1, err := statelessTool.Run(ctx, call1)
	require.NoError(t, err)
	t.Logf("Stateless call 1: %s", resp1.Content)

	// 第二次调用（应该没有记忆）
	call2 := tools.ToolCall{Name: "memory_agent", Input: "What is my name?"}
	resp2, err := statelessTool.Run(ctx, call2)
	require.NoError(t, err)
	t.Logf("Stateless call 2: %s", resp2.Content)

	// 测试有状态适配器
	statefulTool := AsStatefulToolAdapter(agent)

	// 第一次调用
	call3 := tools.ToolCall{Name: "memory_agent", Input: "My favorite color is blue."}
	resp3, err := statefulTool.Run(ctx, call3)
	require.NoError(t, err)
	t.Logf("Stateful call 1: %s", resp3.Content)

	// 第二次调用（应该有记忆）
	call4 := tools.ToolCall{Name: "memory_agent", Input: "What is my favorite color?"}
	resp4, err := statefulTool.Run(ctx, call4)
	require.NoError(t, err)
	t.Logf("Stateful call 2: %s", resp4.Content)
}

// 辅助函数：检查字符串是否包含任何关键词
func containsAny(s string, keywords ...string) bool {
	for _, keyword := range keywords {
		if len(keyword) > 0 && len(s) >= len(keyword) {
			for i := 0; i <= len(s)-len(keyword); i++ {
				if s[i:i+len(keyword)] == keyword {
					return true
				}
			}
		}
	}
	return false
}