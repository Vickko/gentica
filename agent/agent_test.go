package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gentica/tools"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
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
	).WithModel("openai/gpt-5-mini").
		WithTemperature(0.3).
		WithMaxTokens(500).
		WithLogging(true).
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
		"You are a file system explorer. You MUST use the tree tool to explore directories. Never answer file structure questions without using the tree tool first.",
	).WithTools(genkitTreeTool).
		WithModel("openai/gpt-5-mini").
		WithMaxRounds(16).
		WithLogging(true).
		Build()

	// 测试执行
	ctx := context.Background()
	result, err := agent.Run(ctx, "Use the tree tool to list the files in the current directory")
	require.NoError(t, err)
	require.NotEmpty(t, result)

	t.Logf("File explorer result: %s", result)

	// 检查消息历史
	messages := agent.GetMessages()
	t.Logf("Total messages in history: %d", len(messages))
	for i, msg := range messages {
		t.Logf("Message %d - Role: %s", i+1, msg.Role)

		// 检查工具调用
		for _, part := range msg.Content {
			if part.IsToolRequest() {
				t.Logf("  Tool request: %s", part.ToolRequest.Name)
			}
			if part.ToolResponse != nil {
				t.Logf("  Tool response from: %s", part.ToolResponse.Name)
			}
		}
	}

	// 验证结果包含文件信息即可
	// 因为 Genkit 的 Generate 会自动处理工具调用，我们可能看不到中间的工具消息
	assert.Contains(t, result, ".go", "Result should contain file information")
}

func TestSubagentToolCall(t *testing.T) {
	// 创建独立的 Genkit 实例避免工具注册冲突
	localG := genkit.Init(
		context.Background(),
		genkit.WithPlugins(&openai.OpenAI{
			APIKey: apiKey,
			Opts: []option.RequestOption{
				option.WithBaseURL(baseURL),
			},
		}),
	)

	// 获取工作目录
	workingDir, err := os.Getwd()
	require.NoError(t, err)

	// 使用标准方法创建工具，无需担心命名冲突
	treeTool := tools.NewTreeTool(workingDir)
	genkitTreeTool := tools.AdaptBaseToolToGenkit(localG, treeTool)

	// 创建下层 Agent（文件浏览器）
	fileExplorerAgent := NewBuilder(
		localG,
		"file_explorer",
		"Explores file system structure",
		"You are a file system explorer. Use the tree tool to explore directories and list files.",
	).WithInputSchema(
		map[string]any{
			"request": map[string]any{
				"type":        "string",
				"description": "What to explore in the file system",
			},
		},
		"request", // 必需字段
	).WithTools(genkitTreeTool).
		WithModel("openai/gpt-5-mini").
		WithMaxRounds(8).
		WithLogging(true).
		Build()

	// 将下层 Agent 转换为工具
	// Step 1: Agent -> BaseTool
	fileExplorerBaseTool := AsToolAdapter(fileExplorerAgent)

	// Step 2: BaseTool -> Genkit Tool
	fileExplorerGenkitTool := tools.AdaptBaseToolToGenkit(localG, fileExplorerBaseTool)

	// 创建上层 Agent（协调者）- 只有下层 agent 作为工具
	coordinatorAgent := NewBuilder(
		localG,
		"coordinator",
		"Coordinates file exploration tasks",
		"You are a coordinator. You MUST use the file_explorer tool to get file system information. Always report the exact output from the tool in your response.",
	).WithTools(fileExplorerGenkitTool).
		WithModel("openai/gpt-5-mini").
		WithMaxRounds(8).
		WithLogging(true).
		Build()

	// 测试执行 - 要求列出当前目录的文件
	ctx := context.Background()
	result, err := coordinatorAgent.Run(ctx, "Use the file_explorer tool to list files in the current directory and report what .go files you found")
	require.NoError(t, err)
	require.NotEmpty(t, result)

	t.Logf("Coordinator agent result: %s", result)

	// 验证结果包含实际的文件信息
	// 由于协调者只能通过 file_explorer 获取信息，
	// 如果结果包含正确的文件名，说明调用链成功
	assert.Contains(t, result, ".go", "Result should contain Go file information from the tool")

	// 可选：检查更具体的文件名
	possibleFiles := []string{"agent_test.go", "agent.go", "builder.go"}
	foundFile := false
	for _, file := range possibleFiles {
		if strings.Contains(result, file) {
			foundFile = true
			t.Logf("Found expected file in response: %s", file)
			break
		}
	}
	assert.True(t, foundFile, "Result should contain at least one actual file name from the directory")
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
		if strings.Contains(input, "math") || strings.Contains(input, "calculate") ||
			strings.Contains(input, "number") || strings.Contains(input, "equation") {
			return "math", nil
		}
		if strings.Contains(input, "history") || strings.Contains(input, "historical") ||
			strings.Contains(input, "past") || strings.Contains(input, "century") {
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
				return strings.Contains(previousResult, "valid")
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

func TestAgentWithResourceDirectoryTools(t *testing.T) {
	// 创建临时工作目录
	workingDir := t.TempDir()

	// 初始化完整的文件操作和资源管理工具集
	resourceTools := []tools.BaseTool{
		tools.NewResourceDirectoryListTool(workingDir),
		tools.NewResourceDirectoryAddTool(workingDir),
		tools.NewResourceDirectoryRemoveTool(workingDir),
		tools.NewLsTool(workingDir),    // 列出目录文件
		tools.NewViewTool(workingDir),  // 查看文件内容
		tools.NewWriteTool(workingDir), // 创建/覆写文件
		tools.NewEditTool(workingDir),  // 编辑文件内容
	}

	// 转换为 Genkit 工具
	genkitTools := tools.BatchAdaptTools(g, resourceTools...)

	// 预先创建 TODO 目录，使用详细的描述说明其用途
	mgr := tools.GetResourceManager(workingDir)
	todoDir, err := mgr.Add("TODO", "Task management directory for tracking progress. Use TODO.md file with GitHub-style task lists (- [ ] for pending, - [x] for completed)")
	require.NoError(t, err)

	// 创建厨师角色 Agent
	agent := NewBuilder(
		g,
		"chef_assistant",
		"Professional chef preparing dinner",
		`You are a professional chef preparing dinner. You MUST use the TODO system to plan and track your cooking tasks.

IMPORTANT INSTRUCTIONS:
1. First, use resource_directory_list to find the TODO directory and understand its purpose
2. Create a TODO.md file in the TODO directory with your cooking plan using GitHub-style task lists:
   - Use "- [ ]" for pending tasks
   - Use "- [x]" for completed tasks
3. As you progress through cooking (simulated via chat), update the TODO.md file marking completed tasks
4. Use view tool to read TODO.md and edit tool to update it

Task format example:
- [ ] Gather ingredients
- [ ] Prepare vegetables
- [ ] Cook pasta
- [x] Set the table (completed)

Remember: Professional chefs always plan before cooking! Be concise but thorough in your planning.`,
	).WithTools(genkitTools...).
		WithModel("openai/gpt-5-mini").
		WithTemperature(0.1).
		WithMaxRounds(16).
		WithLogging(true).
		Build()

	ctx := context.Background()

	// 第一轮：要求厨师开始准备晚餐并制定计划
	t.Run("Round1_CreatePlan", func(t *testing.T) {
		result, err := agent.Run(ctx, "You need to prepare an Italian pasta dinner for 4 people. Start by creating your cooking plan in the TODO system.")
		require.NoError(t, err)
		require.NotEmpty(t, result)

		// 验证 Agent 创建了 TODO.md 文件
		todoMdPath := filepath.Join(todoDir.Path, "TODO.md")
		_, err = os.Stat(todoMdPath)
		assert.NoError(t, err, "TODO.md should be created")

		// 读取并验证 TODO.md 内容
		content, err := os.ReadFile(todoMdPath)
		require.NoError(t, err)
		todoContent := string(content)

		// 验证包含 GitHub 风格的任务列表
		assert.Contains(t, todoContent, "- [ ]", "Should contain pending tasks")
		assert.Contains(t, strings.ToLower(todoContent), "pasta", "Should mention pasta")

		t.Logf("Round 1 - Plan created:\n%s", result)
		t.Logf("TODO.md content:\n%s", todoContent)
	})

	// 第二轮：模拟开始准备食材
	t.Run("Round2_StartCooking", func(t *testing.T) {
		result, err := agent.Run(ctx, "Great! Now start with the first tasks. (Simulation: You've successfully gathered all ingredients and prepared the vegetables. Update your TODO.)")
		require.NoError(t, err)
		require.NotEmpty(t, result)

		// 验证 TODO.md 被更新
		todoMdPath := filepath.Join(todoDir.Path, "TODO.md")
		content, err := os.ReadFile(todoMdPath)
		require.NoError(t, err)
		todoContent := string(content)

		// 应该有一些任务被标记为完成
		assert.Contains(t, todoContent, "- [x]", "Should have completed tasks")
		assert.Contains(t, todoContent, "- [ ]", "Should still have pending tasks")

		t.Logf("Round 2 - Progress update:\n%s", result)
		t.Logf("Updated TODO.md:\n%s", todoContent)
	})

	// 第三轮：继续烹饪过程
	t.Run("Round3_ContinueCooking", func(t *testing.T) {
		result, err := agent.Run(ctx, "Continue cooking. (Simulation: The water is boiling, pasta is cooking perfectly, and the sauce is simmering. Update your progress.)")
		require.NoError(t, err)
		require.NotEmpty(t, result)

		// 验证更多任务被完成
		todoMdPath := filepath.Join(todoDir.Path, "TODO.md")
		content, err := os.ReadFile(todoMdPath)
		require.NoError(t, err)
		todoContent := string(content)

		// 统计完成的任务数量应该增加
		completedCount := strings.Count(todoContent, "- [x]")
		assert.Greater(t, completedCount, 1, "Should have multiple completed tasks")

		t.Logf("Round 3 - Cooking progress:\n%s", result)
		t.Logf("TODO.md progress:\n%s", todoContent)
	})

	// 第四轮：完成晚餐
	t.Run("Round4_CompleteDinner", func(t *testing.T) {
		result, err := agent.Run(ctx, "Excellent! Finish the meal. (Simulation: Pasta is perfectly al dente, sauce is ready, table is set, and everything is plated beautifully. Complete all remaining tasks.)")
		require.NoError(t, err)
		require.NotEmpty(t, result)

		// 验证所有主要任务都已完成
		todoMdPath := filepath.Join(todoDir.Path, "TODO.md")
		content, err := os.ReadFile(todoMdPath)
		require.NoError(t, err)
		todoContent := string(content)

		// 大部分任务应该被标记为完成
		completedCount := strings.Count(todoContent, "- [x]")
		pendingCount := strings.Count(todoContent, "- [ ]")

		t.Logf("Final status - Completed: %d, Pending: %d", completedCount, pendingCount)
		assert.Greater(t, completedCount, 3, "Should have completed most tasks")

		// 验证关键烹饪步骤都完成了
		todoLower := strings.ToLower(todoContent)
		if strings.Contains(todoLower, "pasta") && strings.Contains(todoLower, "- [x]") {
			t.Log("Pasta cooking task completed ✓")
		}

		t.Logf("Round 4 - Dinner complete:\n%s", result)
		t.Logf("Final TODO.md:\n%s", todoContent)
	})

	// 最终验证：检查 Agent 是否成功使用了资源管理系统
	t.Run("FinalValidation", func(t *testing.T) {
		// 验证 TODO.md 文件存在且格式正确
		todoMdPath := filepath.Join(todoDir.Path, "TODO.md")
		content, err := os.ReadFile(todoMdPath)
		require.NoError(t, err)

		todoContent := string(content)

		// 验证文件不为空
		assert.NotEmpty(t, todoContent, "TODO.md should not be empty")

		// 验证包含正确的 Markdown 格式
		assert.Contains(t, todoContent, "- [", "Should use GitHub-style task lists")

		// 验证有任务进度（既有完成也有可能的未完成）
		hasProgress := strings.Contains(todoContent, "- [x]")
		assert.True(t, hasProgress, "Should show task completion progress")

		t.Logf("✅ Chef successfully used the TODO system to plan and track dinner preparation")
		t.Logf("✅ TODO.md was created and updated throughout the cooking process")
		t.Logf("✅ GitHub-style task lists were used correctly")
	})
}
