package agents

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gentica/agent"
	"gentica/tools"
)

func TestPPTTopAgent(t *testing.T) {
	// 检查 genkit 是否初始化
	if g == nil || sharedPPTTopDeps == nil {
		t.Skip("Skipping test: genkit not initialized or PPT deps not available")
	}

	// 加载环境变量
	err := godotenv.Load("../../.env")
	if err != nil {
		t.Logf("Warning: .env file not found, using existing environment variables")
	}

	// 使用共享依赖创建 PPT Top Agent
	pptTopAgent := NewPPTTopAgentWithDeps(g, sharedPPTTopDeps)
	require.NotNil(t, pptTopAgent)

	t.Run("Test Complete PPT Generation Flow", func(t *testing.T) {
		ctx := context.Background()

		// 用户请求
		userRequest := `{"request": "请帮我制作一个关于人工智能在教育领域应用的PPT，主要介绍AI如何改变教学方式"}`

		// 执行 Agent
		result, err := pptTopAgent.Run(ctx, userRequest)
		require.NoError(t, err)
		assert.NotEmpty(t, result)

		t.Logf("PPT Top Agent Result:\n%s", result)

		// 解析结果
		topResult, err := ParsePPTTopResult(result)
		assert.NoError(t, err)

		// 验证结果包含必要信息
		if topResult.Status == "success" {
			// 如果成功，应该有文件路径
			if topResult.OutlineFilePath != "" {
				t.Logf("Outline file: %s", topResult.OutlineFilePath)
				// 验证大纲文件存在
				_, err := os.Stat(topResult.OutlineFilePath)
				if err == nil {
					t.Log("✓ Outline file exists")
				}
			}

			if topResult.TemplateDirectory != "" {
				t.Logf("Template directory: %s", topResult.TemplateDirectory)
				// 验证模板目录存在
				_, err := os.Stat(topResult.TemplateDirectory)
				if err == nil {
					t.Log("✓ Template directory exists")
				}
			}

			if len(topResult.GeneratedPages) > 0 {
				t.Logf("Generated %d pages", len(topResult.GeneratedPages))
				for i, page := range topResult.GeneratedPages {
					t.Logf("  Page %d: %s", i+1, page)
				}
			}
		}

		// 验证生成的内容
		assert.NotEmpty(t, topResult.Summary)
		t.Logf("Summary: %s", topResult.Summary)
	})
}

func TestPPTTopAgentWithMockDependencies(t *testing.T) {
	// 检查 genkit 是否初始化
	if g == nil {
		t.Skip("Skipping test: genkit not initialized")
	}

	// 设置测试工作目录
	workingDir := t.TempDir()

	// 创建模拟的子 agents
	mockOutlineAgent := createMockOutlineAgent(g, workingDir)
	mockTemplateAgent := createMockTemplateAgent(g, workingDir)
	mockPageGenAgent := createMockPageGenAgent(g, workingDir)

	// 创建自定义依赖（使用新的 mock agents 但复用工具）
	deps := &PPTTopAgentDependencies{
		OutlinePlanTool:    tools.AdaptBaseToolToGenkit(g, agent.AsToolAdapter(mockOutlineAgent)),
		TemplateDesignTool: tools.AdaptBaseToolToGenkit(g, agent.AsToolAdapter(mockTemplateAgent)),
		PageGenerateTool:   tools.AdaptBaseToolToGenkit(g, agent.AsToolAdapter(mockPageGenAgent)),
		ViewTool:           sharedPPTTopDeps.ViewTool,          // 复用共享工具
		LsTool:             sharedPPTTopDeps.LsTool,            // 复用共享工具
		DirectoryListTool:  sharedPPTTopDeps.DirectoryListTool, // 复用共享工具
	}

	// 创建 PPT Top Agent
	pptTopAgent := NewPPTTopAgentWithDeps(g, deps)
	require.NotNil(t, pptTopAgent)

	t.Run("Test Agent Coordination", func(t *testing.T) {
		ctx := context.Background()

		// 用户请求
		userRequest := `{"request": "制作一个简单的测试PPT"}`

		// 执行 Agent
		result, err := pptTopAgent.Run(ctx, userRequest)
		require.NoError(t, err)
		assert.NotEmpty(t, result)

		t.Logf("Mock Test Result:\n%s", result)

		// 验证工作流被正确执行
		assert.Contains(t, result, "大纲") // 应该提到大纲生成
		// 注意：由于是mock，实际的工具调用可能不会执行
	})
}

// 创建模拟的大纲生成 Agent
func createMockOutlineAgent(g *genkit.Genkit, workingDir string) agent.Agent {
	systemPrompt := `你是一个模拟的大纲生成器。
当收到请求时，创建一个简单的测试大纲文件并返回文件路径。`

	// 创建一个测试大纲文件
	testDir := filepath.Join(workingDir, ".tmp", "test", "ppt_outline_mock")
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		// 只记录错误，不使用 t.Fatalf
		panic(fmt.Sprintf("Failed to create test directory: %v", err))
	}

	outlineFile := filepath.Join(testDir, "outline.xml")
	outlineContent := `<ppt_plan>
    <project_info>
        <title>测试PPT</title>
        <total_pages>5</total_pages>
    </project_info>
    <page_structure>
        <page>
            <page_number>1</page_number>
            <page_title>封面</page_title>
            <page_type>封面页</page_type>
        </page>
    </page_structure>
</ppt_plan>`

	err = os.WriteFile(outlineFile, []byte(outlineContent), 0644)
	if err != nil {
		// 只记录错误，不使用 t.Fatalf
		panic(fmt.Sprintf("Failed to write outline file: %v", err))
	}

	return agent.NewBuilder(
		g,
		"mock_outline_agent",
		"模拟大纲生成器",
		systemPrompt,
	).WithInputSchema(
		map[string]any{
			"topic": map[string]any{
				"type": "string",
			},
		},
	).WithModel("openai/gpt-4o-mini").
		Build()
}

// 创建模拟的模板设计 Agent
func createMockTemplateAgent(g *genkit.Genkit, workingDir string) agent.Agent {
	systemPrompt := `你是一个模拟的模板设计器。
返回预设的模板文件路径。`

	return agent.NewBuilder(
		g,
		"mock_template_agent",
		"模拟模板设计器",
		systemPrompt,
	).WithInputSchema(
		map[string]any{
			"outline_path": map[string]any{
				"type": "string",
			},
		},
	).WithModel("openai/gpt-4o-mini").
		Build()
}

// 创建模拟的页面生成 Agent
func createMockPageGenAgent(g *genkit.Genkit, workingDir string) agent.Agent {
	systemPrompt := `你是一个模拟的页面生成器。
返回生成的页面列表。`

	return agent.NewBuilder(
		g,
		"mock_page_gen_agent",
		"模拟页面生成器",
		systemPrompt,
	).WithInputSchema(
		map[string]any{
			"outline_path": map[string]any{
				"type": "string",
			},
			"template_paths": map[string]any{
				"type": "object",
			},
		},
	).WithModel("openai/gpt-4o-mini").
		Build()
}

func TestPPTTopAgentSystemPromptBehavior(t *testing.T) {
	// 检查 genkit 和共享依赖是否初始化
	if g == nil || sharedPPTTopDeps == nil {
		t.Skip("Skipping test: genkit not initialized or PPT deps not available")
	}

	// 使用共享依赖创建 PPT Top Agent
	pptTopAgent := NewPPTTopAgentWithDeps(g, sharedPPTTopDeps)
	config := pptTopAgent.GetConfig()

	// 验证系统提示包含关键指令
	assert.Contains(t, config.SystemPrompt, "outline_plan_agent")
	assert.Contains(t, config.SystemPrompt, "template_design_agent")
	assert.Contains(t, config.SystemPrompt, "page_generate_agent")
	assert.Contains(t, config.SystemPrompt, "view")
	assert.Contains(t, config.SystemPrompt, "ls")
	assert.Contains(t, config.SystemPrompt, "resource_directory_list")

	// 验证工具集成
	assert.GreaterOrEqual(t, len(config.Tools), 6) // 至少应该有6个工具

	t.Logf("Agent has %d tools configured", len(config.Tools))
	for _, tool := range config.Tools {
		t.Logf("  - Tool: %s", tool.Name())
	}
}

func TestParseTimeOptimization(t *testing.T) {
	// 检查 genkit 和共享依赖是否初始化
	if g == nil || sharedPPTTopDeps == nil {
		t.Skip("Skipping test: genkit not initialized or PPT deps not available")
	}

	// 使用更短的超时和简化的请求
	_, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 使用共享依赖创建 Agent
	pptTopAgent := NewPPTTopAgentWithDeps(g, sharedPPTTopDeps)

	// 设置简单的消息历史以快速测试
	simpleMessage := ai.NewUserTextMessage("生成一个测试PPT的JSON结果")
	pptTopAgent.SetMessages([]*ai.Message{simpleMessage})

	// 注意：这个测试主要验证Agent创建和配置，
	// 实际执行需要API key和网络连接
	t.Log("PPT Top Agent created successfully with optimized configuration")

	// 验证配置优化
	config := pptTopAgent.GetConfig()
	assert.Equal(t, float32(0.3), config.Temperature) // 低温度确保稳定性
	assert.Equal(t, 20, config.MaxRounds)             // 足够的轮数
	assert.Equal(t, 6000, config.MaxTokens)           // 足够的token限制
}

// TestResultParsing 测试结果解析功能
func TestResultParsing(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected PPTTopResult
	}{
		{
			name: "Valid JSON result",
			input: `Some text before
{
  "status": "success",
  "outline_directory": "ppt_outline_123",
  "outline_file_path": "/tmp/outline.xml",
  "summary": "PPT generated successfully"
}
Some text after`,
			expected: PPTTopResult{
				Status:           "success",
				OutlineDirectory: "ppt_outline_123",
				OutlineFilePath:  "/tmp/outline.xml",
				Summary:          "PPT generated successfully",
			},
		},
		{
			name:  "Plain text result",
			input: "This is just plain text without JSON",
			expected: PPTTopResult{
				Status:  "completed",
				Summary: "This is just plain text without JSON",
			},
		},
		{
			name: "Mixed content with partial JSON",
			input: `Started processing...
{
  "status": "success"
}`,
			expected: PPTTopResult{
				Status: "success",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParsePPTTopResult(tc.input)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected.Status, result.Status)

			if tc.expected.Summary != "" {
				assert.Equal(t, tc.expected.Summary, result.Summary)
			}

			if tc.expected.OutlineDirectory != "" {
				assert.Equal(t, tc.expected.OutlineDirectory, result.OutlineDirectory)
			}
		})
	}
}

// TestIntegrationWithFileSystem 测试与文件系统的集成
func TestIntegrationWithFileSystem(t *testing.T) {
	// 检查 genkit 和共享依赖是否初始化
	if g == nil || sharedDeps == nil {
		t.Skip("Skipping test: genkit not initialized or shared dependencies not available")
	}

	// 使用共享的临时目录
	workingDir := sharedTempDir

	// 创建测试文件结构
	testOutlineDir := filepath.Join(workingDir, ".tmp", "test", "ppt_outline_test")
	testTemplateDir := filepath.Join(workingDir, ".tmp", "test", "ppt_template_test")

	err := os.MkdirAll(testOutlineDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test outline directory: %v", err)
	}
	err = os.MkdirAll(testTemplateDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test template directory: %v", err)
	}

	// 创建测试文件
	outlineFile := filepath.Join(testOutlineDir, "outline.xml")
	err = os.WriteFile(outlineFile, []byte("<ppt_plan>test</ppt_plan>"), 0644)
	if err != nil {
		t.Fatalf("Failed to write outline file: %v", err)
	}

	templateFile := filepath.Join(testTemplateDir, "cover.html")
	err = os.WriteFile(templateFile, []byte("<html>Cover Page</html>"), 0644)
	if err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	// 使用共享的工具（已经是 BaseTool 类型）
	viewTool := tools.NewViewTool(workingDir)
	lsTool := tools.NewLsTool(workingDir)

	// 测试 View 工具
	ctx := context.Background()
	viewCall := tools.ToolCall{
		Name:  "view",
		Input: `{"file_path": "` + outlineFile + `"}`,
	}

	viewResp, err := viewTool.Run(ctx, viewCall)
	assert.NoError(t, err)
	assert.Contains(t, viewResp.Content, "<ppt_plan>test</ppt_plan>")

	// 测试 Ls 工具
	lsCall := tools.ToolCall{
		Name:  "ls",
		Input: `{"path": "` + testOutlineDir + `"}`,
	}

	lsResp, err := lsTool.Run(ctx, lsCall)
	assert.NoError(t, err)
	assert.Contains(t, lsResp.Content, "outline.xml")

	t.Log("File system integration test passed")
}

// TestAgentToToolConversion 测试 Agent 到工具的转换
func TestAgentToToolConversion(t *testing.T) {
	// 检查 genkit 是否初始化
	if g == nil {
		t.Skip("Skipping test: genkit not initialized")
	}

	_ = t.TempDir()

	// 创建一个简单的测试 Agent
	testAgent := agent.NewBuilder(
		g,
		"test_agent",
		"Test Agent",
		"You are a test agent",
	).WithInputSchema(
		map[string]any{
			"input": map[string]any{
				"type": "string",
			},
		},
		"input",
	).Build()

	// 转换为工具
	toolAdapter := agent.AsToolAdapter(testAgent)
	assert.NotNil(t, toolAdapter)

	// 验证工具信息
	toolInfo := toolAdapter.Info()
	assert.Equal(t, "test_agent", toolInfo.Name)
	assert.Equal(t, "Test Agent", toolInfo.Description)
	assert.Contains(t, toolInfo.Required, "input")

	// 转换为 Genkit AI Tool
	genkitTool := tools.AdaptBaseToolToGenkit(g, toolAdapter)
	assert.NotNil(t, genkitTool)
	assert.Equal(t, "test_agent", genkitTool.Name())

	t.Log("Agent to tool conversion test passed")
}