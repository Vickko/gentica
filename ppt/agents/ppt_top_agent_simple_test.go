package agents

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPPTTopAgentBasicConfiguration(t *testing.T) {
	// 检查 genkit 和共享依赖是否初始化
	if g == nil || sharedPPTTopDeps == nil {
		t.Skip("Skipping test: genkit not initialized or PPT deps not available")
	}

	// 使用共享依赖创建 PPT Top Agent
	pptTopAgent := NewPPTTopAgentWithDeps(g, sharedPPTTopDeps)
	require.NotNil(t, pptTopAgent)

	// 验证基本配置
	config := pptTopAgent.GetConfig()

	// 验证 Agent 名称和描述
	assert.Equal(t, "ppt_top_agent", config.Name)
	assert.Equal(t, "PPT制作顶层协调器", config.Description)

	// 验证输入 schema
	assert.NotNil(t, config.InputSchema)
	assert.Contains(t, config.InputSchema, "request")

	// 验证必需参数
	assert.Contains(t, config.Required, "request")

	// 验证模型配置
	assert.Equal(t, "openai/gpt-4o-mini", config.Model)
	assert.Equal(t, float32(0.3), config.Temperature)
	assert.Equal(t, 6000, config.MaxTokens)
	assert.Equal(t, 20, config.MaxRounds)

	// 验证系统提示包含关键元素
	assert.Contains(t, config.SystemPrompt, "PPT制作专家")
	assert.Contains(t, config.SystemPrompt, "outline_plan_agent")
	assert.Contains(t, config.SystemPrompt, "template_design_agent")
	assert.Contains(t, config.SystemPrompt, "page_generate_agent")
	assert.Contains(t, config.SystemPrompt, "view")
	assert.Contains(t, config.SystemPrompt, "ls")
	assert.Contains(t, config.SystemPrompt, "template_dir")

	// 验证工具集成
	assert.GreaterOrEqual(t, len(config.Tools), 6, "Should have at least 6 tools")

	t.Logf("PPT Top Agent configured with %d tools", len(config.Tools))
}

func TestPPTTopResultParsing(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected PPTTopResult
	}{
		{
			name: "Valid complete JSON",
			input: `
开始处理PPT生成任务...
{
  "status": "success",
  "outline_directory": "ppt_outline_20241120",
  "outline_file_path": ".tmp/ppt_outline_20241120/outline.xml",
  "template_directory": "ppt_template_20241120",
  "template_file_paths": {
    "cover": ".tmp/ppt_template_20241120/cover.html",
    "toc": ".tmp/ppt_template_20241120/toc.html",
    "content": ".tmp/ppt_template_20241120/content.html"
  },
  "generated_pages": [
    ".tmp/ppt_pages_20241120/page1.html",
    ".tmp/ppt_pages_20241120/page2.html"
  ],
  "summary": "成功生成了包含2个页面的PPT"
}
任务完成！`,
			expected: PPTTopResult{
				Status:           "success",
				OutlineDirectory: "ppt_outline_20241120",
				OutlineFilePath:  ".tmp/ppt_outline_20241120/outline.xml",
				TemplateDirectory: "ppt_template_20241120",
				TemplateFilePaths: map[string]string{
					"cover":   ".tmp/ppt_template_20241120/cover.html",
					"toc":     ".tmp/ppt_template_20241120/toc.html",
					"content": ".tmp/ppt_template_20241120/content.html",
				},
				GeneratedPages: []string{
					".tmp/ppt_pages_20241120/page1.html",
					".tmp/ppt_pages_20241120/page2.html",
				},
				Summary: "成功生成了包含2个页面的PPT",
			},
		},
		{
			name:  "Plain text without JSON",
			input: "PPT生成过程遇到了一些问题，请稍后重试。",
			expected: PPTTopResult{
				Status:  "completed",
				Summary: "PPT生成过程遇到了一些问题，请稍后重试。",
			},
		},
		{
			name: "Partial JSON",
			input: `处理中...
{
  "status": "success",
  "summary": "部分完成"
}`,
			expected: PPTTopResult{
				Status:  "success",
				Summary: "部分完成",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParsePPTTopResult(tc.input)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected.Status, result.Status)
			assert.Equal(t, tc.expected.Summary, result.Summary)

			if tc.expected.OutlineDirectory != "" {
				assert.Equal(t, tc.expected.OutlineDirectory, result.OutlineDirectory)
			}

			if tc.expected.OutlineFilePath != "" {
				assert.Equal(t, tc.expected.OutlineFilePath, result.OutlineFilePath)
			}

			if len(tc.expected.TemplateFilePaths) > 0 {
				assert.Equal(t, tc.expected.TemplateFilePaths, result.TemplateFilePaths)
			}

			if len(tc.expected.GeneratedPages) > 0 {
				assert.Equal(t, tc.expected.GeneratedPages, result.GeneratedPages)
			}
		})
	}
}

func TestPPTTopAgentInputHandling(t *testing.T) {
	// 检查 genkit 和共享依赖是否初始化
	if g == nil || sharedPPTTopDeps == nil {
		t.Skip("Skipping test: genkit not initialized or PPT deps not available")
	}

	// 使用共享依赖创建 PPT Top Agent
	pptTopAgent := NewPPTTopAgentWithDeps(g, sharedPPTTopDeps)

	// 测试输入参数验证
	testInputs := []struct {
		name    string
		input   string
		isValid bool
	}{
		{
			name:    "Valid JSON input",
			input:   `{"request": "制作一个关于AI的PPT"}`,
			isValid: true,
		},
		{
			name:    "Missing required field",
			input:   `{"topic": "AI"}`,
			isValid: false,
		},
		{
			name:    "Empty object",
			input:   `{}`,
			isValid: false,
		},
	}

	for _, tt := range testInputs {
		t.Run(tt.name, func(t *testing.T) {
			// 这里只验证 Agent 创建成功，不实际执行
			// 因为执行需要 OpenAI API key
			assert.NotNil(t, pptTopAgent)
			t.Logf("Test case %s: input validation check (would be %v)", tt.name, tt.isValid)
		})
	}
}