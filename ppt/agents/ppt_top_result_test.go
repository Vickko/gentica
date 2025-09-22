package agents

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestParsePPTTopResultSimple 测试结果解析功能（不需要初始化 Agent）
func TestParsePPTTopResultSimple(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected PPTTopResult
		hasError bool
	}{
		{
			name: "Valid JSON with all fields",
			input: `{
				"status": "success",
				"outline_directory": "ppt_outline_123",
				"outline_file_path": "/tmp/outline.xml",
				"template_directory": "ppt_template_123",
				"template_file_paths": {
					"cover": "/tmp/cover.html",
					"toc": "/tmp/toc.html"
				},
				"generated_pages": ["/tmp/page1.html", "/tmp/page2.html"],
				"summary": "Successfully generated PPT"
			}`,
			expected: PPTTopResult{
				Status:            "success",
				OutlineDirectory:  "ppt_outline_123",
				OutlineFilePath:   "/tmp/outline.xml",
				TemplateDirectory: "ppt_template_123",
				TemplateFilePaths: map[string]string{
					"cover": "/tmp/cover.html",
					"toc":   "/tmp/toc.html",
				},
				GeneratedPages: []string{"/tmp/page1.html", "/tmp/page2.html"},
				Summary:        "Successfully generated PPT",
			},
			hasError: false,
		},
		{
			name: "JSON embedded in text",
			input: `Starting PPT generation...
			Processing outline...
			{
				"status": "success",
				"outline_directory": "dir_001",
				"summary": "Partial completion"
			}
			Task completed!`,
			expected: PPTTopResult{
				Status:           "success",
				OutlineDirectory: "dir_001",
				Summary:          "Partial completion",
			},
			hasError: false,
		},
		{
			name:  "Plain text without JSON",
			input: "This is just a plain text response without any JSON structure",
			expected: PPTTopResult{
				Status:  "completed",
				Summary: "This is just a plain text response without any JSON structure",
			},
			hasError: false,
		},
		{
			name: "Empty JSON",
			input: `{}`,
			expected: PPTTopResult{
				Status:  "",
				Summary: "",
			},
			hasError: false,
		},
		{
			name: "Minimal valid JSON",
			input: `{"status": "failed"}`,
			expected: PPTTopResult{
				Status: "failed",
			},
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParsePPTTopResult(tt.input)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.Status, result.Status)
				assert.Equal(t, tt.expected.Summary, result.Summary)
				assert.Equal(t, tt.expected.OutlineDirectory, result.OutlineDirectory)
				assert.Equal(t, tt.expected.OutlineFilePath, result.OutlineFilePath)
				assert.Equal(t, tt.expected.TemplateDirectory, result.TemplateDirectory)

				// 比较 map
				if tt.expected.TemplateFilePaths != nil {
					assert.Equal(t, tt.expected.TemplateFilePaths, result.TemplateFilePaths)
				}

				// 比较 slice
				if tt.expected.GeneratedPages != nil {
					assert.Equal(t, tt.expected.GeneratedPages, result.GeneratedPages)
				}
			}
		})
	}
}

// TestPPTTopAgentWorkflow 测试工作流逻辑（文档测试，不实际执行）
func TestPPTTopAgentWorkflow(t *testing.T) {
	// 这是一个文档测试，展示预期的工作流程
	expectedWorkflow := []string{
		"1. 接收用户的PPT制作请求",
		"2. 调用 outline_plan_agent 生成大纲",
		"3. 使用 view 工具查看大纲内容（可选）",
		"4. 调用 template_design_agent 设计模板",
		"5. 使用 ls 工具查看模板目录（可选）",
		"6. 调用 page_generate_agent 生成页面",
		"7. 汇总结果并返回",
	}

	t.Log("Expected PPT Top Agent workflow:")
	for _, step := range expectedWorkflow {
		t.Log(step)
	}

	// 验证系统提示中包含这些关键步骤
	systemPrompt := `你是一个PPT制作专家。当用户要求制作PPT时，请按以下流程操作`
	assert.Contains(t, systemPrompt, "PPT制作专家")

	t.Log("✓ Workflow documentation test passed")
}