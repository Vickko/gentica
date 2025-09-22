package agents

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gentica/agent"
	"gentica/tools"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutlinePlanAgent_ParsePPTPlan(t *testing.T) {
	// 测试正常的 XML 解析
	xmlData := `<ppt_plan>
    <project_info>
        <title>测试PPT标题</title>
        <total_pages>2</total_pages>
        <target_audience>测试受众</target_audience>
        <design_style>简约风格</design_style>
        <usage_scenario>内部会议</usage_scenario>
        <presentation_purpose>信息分享</presentation_purpose>
    </project_info>
    <page_structure>
        <page>
            <page_number>1</page_number>
            <page_title>封面</page_title>
            <page_type>封面页</page_type>
            <core_content>标题和副标题</core_content>
        </page>
        <page>
            <page_number>2</page_number>
            <page_title>目录</page_title>
            <page_type>目录页</page_type>
            <core_content>1. 第一章\n2. 第二章</core_content>
        </page>
    </page_structure>
</ppt_plan>`

	plan, err := ParsePPTPlan(xmlData)
	require.NoError(t, err)
	require.NotNil(t, plan)

	// 验证项目信息
	assert.Equal(t, "测试PPT标题", plan.ProjectInfo.Title)
	assert.Equal(t, 2, plan.ProjectInfo.TotalPages)
	assert.Equal(t, "测试受众", plan.ProjectInfo.TargetAudience)
	assert.Equal(t, "简约风格", plan.ProjectInfo.DesignStyle)

	// 验证页面结构
	assert.Len(t, plan.PageStructure.Pages, 2)
	assert.Equal(t, 1, plan.PageStructure.Pages[0].PageNumber)
	assert.Equal(t, "封面", plan.PageStructure.Pages[0].PageTitle)
	assert.Equal(t, "封面页", plan.PageStructure.Pages[0].PageType)
}

func TestOutlinePlanAgent_ParseWithExtraText(t *testing.T) {
	// 测试包含额外文本的 XML 解析
	xmlDataWithExtra := `这是一些额外的文本
<ppt_plan>
    <project_info>
        <title>测试标题</title>
        <total_pages>1</total_pages>
        <target_audience>测试</target_audience>
        <design_style>测试</design_style>
        <usage_scenario>测试</usage_scenario>
        <presentation_purpose>测试</presentation_purpose>
    </project_info>
    <page_structure>
        <page>
            <page_number>1</page_number>
            <page_title>测试页</page_title>
            <page_type>内容页</page_type>
            <core_content>测试内容</core_content>
        </page>
    </page_structure>
</ppt_plan>
还有一些额外的文本`

	plan, err := ParsePPTPlan(xmlDataWithExtra)
	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Equal(t, "测试标题", plan.ProjectInfo.Title)
	assert.Equal(t, 1, plan.ProjectInfo.TotalPages)
}

func TestOutlinePlanAgent_ParseInvalidXML(t *testing.T) {
	// 测试无效的 XML
	invalidXML := `<ppt_plan>
    <project_info>
        <title></title>
        <total_pages>0</total_pages>
    </project_info>
    <page_structure>
    </page_structure>
</ppt_plan>`

	plan, err := ParsePPTPlan(invalidXML)
	require.Error(t, err)
	require.Nil(t, plan)
}

func TestOutlinePlanAgent_Integration(t *testing.T) {
	// 跳过集成测试，如果没有设置环境变量
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=true to run.")
	}

	// 检查 genkit 和共享依赖是否已初始化
	if g == nil || sharedOutlineAgent == nil {
		t.Skip("Skipping test: genkit not initialized or shared dependencies not available")
	}

	// 使用共享的 OutlinePlanAgent
	outlineAgent := sharedOutlineAgent

	// 测试输入
	topic := "人工智能在教育领域的应用"

	// 准备输入
	input := map[string]any{
		"topic": topic,
	}

	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	// 执行 Agent
	ctx := context.Background()
	result, err := outlineAgent.Run(ctx, string(inputJSON))
	require.NoError(t, err)
	require.NotEmpty(t, result)

	t.Logf("Agent result: %s", result)

	// 尝试解析结果
	var agentResult struct {
		Status        string `json:"status"`
		DirectoryName string `json:"directory_name"`
		FilePath      string `json:"file_path"`
		Summary       string `json:"summary"`
	}

	// 查找 JSON 部分
	startIdx := strings.Index(result, "{")
	endIdx := strings.LastIndex(result, "}")
	if startIdx != -1 && endIdx != -1 {
		jsonStr := result[startIdx : endIdx+1]
		err = json.Unmarshal([]byte(jsonStr), &agentResult)
		if err == nil {
			assert.Equal(t, "success", agentResult.Status)
			assert.NotEmpty(t, agentResult.DirectoryName)
			assert.NotEmpty(t, agentResult.FilePath)

			// 检查文件是否存在
			if agentResult.FilePath != "" {
				fullPath := filepath.Join(sharedTempDir, agentResult.FilePath)
				if _, err := os.Stat(fullPath); err == nil {
					// 读取并解析生成的 XML
					content, err := os.ReadFile(fullPath)
					require.NoError(t, err)

					plan, err := ParsePPTPlan(string(content))
					require.NoError(t, err)
					assert.NotEmpty(t, plan.ProjectInfo.Title)
					assert.Greater(t, plan.ProjectInfo.TotalPages, 0)
					assert.NotEmpty(t, plan.PageStructure.Pages)

					t.Logf("Generated PPT outline with %d pages", plan.ProjectInfo.TotalPages)
				}
			}
		}
	}
}

func TestOutlinePlanAgent_AsBaseTool(t *testing.T) {
	// 跳过集成测试，如果没有设置环境变量
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=true to run.")
	}

	// 检查 genkit 和共享依赖是否已初始化
	if g == nil || sharedOutlineAgent == nil {
		t.Skip("Skipping test: genkit not initialized or shared dependencies not available")
	}

	// 使用共享的 OutlinePlanAgent
	outlineAgent := sharedOutlineAgent

	// 将 Agent 转换为 BaseTool
	outlineTool := agent.AsToolAdapter(outlineAgent)
	require.NotNil(t, outlineTool)

	// 验证工具信息
	info := outlineTool.Info()
	assert.Equal(t, "outline_plan_agent", info.Name)
	assert.Equal(t, "生成PPT制作大纲", info.Description)
	assert.Contains(t, info.Parameters, "topic")

	// 测试工具执行
	ctx := context.Background()
	toolCall := tools.ToolCall{
		Name:  "outline_plan_agent",
		Input: `{"topic": "云计算基础知识介绍"}`,
	}

	response, err := outlineTool.Run(ctx, toolCall)
	require.NoError(t, err)
	assert.Equal(t, tools.ToolResponseTypeText, response.Type)
	assert.NotEmpty(t, response.Content)
	assert.False(t, response.IsError)

	t.Logf("Tool response: %s", response.Content)
}

func TestOutlinePlanAgent_CompleteWorkflow(t *testing.T) {
	// 跳过集成测试，如果没有设置环境变量
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=true to run.")
	}

	// 检查 genkit 和共享依赖是否已初始化
	if g == nil || sharedOutlineAgent == nil {
		t.Skip("Skipping test: genkit not initialized or shared dependencies not available")
	}

	// 使用共享的 OutlinePlanAgent
	outlineAgent := sharedOutlineAgent

	t.Logf("Working directory: %s", sharedTempDir)

	// 测试详细的输入
	detailedTopic := `
主题：数字化转型的最佳实践
目标受众：企业高管和IT决策者
演示场景：行业峰会主题演讲
时长：30分钟
关键内容：
1. 数字化转型的定义和价值
2. 成功案例分析（至少3个行业）
3. 实施路线图
4. 常见挑战和解决方案
5. 未来趋势展望
`

	// 准备输入
	input := map[string]any{
		"topic": detailedTopic,
	}

	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	// 执行 Agent
	ctx := context.Background()
	result, err := outlineAgent.Run(ctx, string(inputJSON))
	require.NoError(t, err)
	require.NotEmpty(t, result)

	t.Logf("Complete workflow result:\n%s", result)

	// 解析返回的结果
	outlineResult, err := ParseOutlineResult(result)
	if err != nil {
		t.Logf("Failed to parse result as JSON, raw result: %s", result)
	} else {
		assert.Equal(t, "success", outlineResult.Status)
		assert.NotEmpty(t, outlineResult.DirectoryName)
		assert.NotEmpty(t, outlineResult.FilePath)

		// 验证资源目录被创建
		dirPath := filepath.Join(sharedTempDir, ".tmp")
		entries, err := os.ReadDir(dirPath)
		if err == nil {
			t.Logf("Found %d entries in .tmp directory", len(entries))
			for _, entry := range entries {
				t.Logf("  - %s (dir: %v)", entry.Name(), entry.IsDir())
			}
		}

		// 尝试读取生成的 XML 文件
		if outlineResult.FilePath != "" {
			// 文件路径可能是相对路径或绝对路径
			var xmlPath string
			if filepath.IsAbs(outlineResult.FilePath) {
				xmlPath = outlineResult.FilePath
			} else {
				xmlPath = filepath.Join(sharedTempDir, outlineResult.FilePath)
			}

			if content, err := os.ReadFile(xmlPath); err == nil {
				t.Logf("Generated XML content:\n%s", string(content))

				// 解析 XML 验证格式
				plan, err := ParsePPTPlan(string(content))
				require.NoError(t, err)
				assert.Contains(t, strings.ToLower(plan.ProjectInfo.Title), "数字化转型")
				assert.GreaterOrEqual(t, plan.ProjectInfo.TotalPages, 8)
				assert.LessOrEqual(t, plan.ProjectInfo.TotalPages, 15)

				// 验证页面类型
				hasTitle := false
				hasTableOfContents := false
				hasEnding := false

				for _, page := range plan.PageStructure.Pages {
					switch page.PageType {
					case "封面页":
						hasTitle = true
					case "目录页":
						hasTableOfContents = true
					case "结尾页":
						hasEnding = true
					}
				}

				assert.True(t, hasTitle, "Should have title page")
				assert.True(t, hasTableOfContents, "Should have table of contents")
				assert.True(t, hasEnding, "Should have ending page")
			} else {
				t.Logf("Could not read XML file at %s: %v", xmlPath, err)
			}
		}
	}

	t.Logf("Test completed. Check %s for generated files", sharedTempDir)
}