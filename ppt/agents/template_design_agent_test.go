package agents

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTemplateDesignAgent_BasicGeneration 测试基本的模板生成功能
// 注意：这个测试需要设置 TestMain 函数来初始化 genkit
// 实际测试时会使用 research_collector_test.go 中的共享 g 变量
func TestTemplateDesignAgent_BasicGeneration(t *testing.T) {
	if g == nil || sharedTemplateAgent == nil {
		t.Skip("Skipping test: genkit not initialized or shared dependencies not available")
	}

	// 使用共享的 TemplateAgent
	templateAgent := sharedTemplateAgent

	// 准备输入的 JSON
	input := map[string]any{
		"style_description": "科技现代风格",
	}
	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	// 执行 Agent
	result, err := templateAgent.Run(context.Background(), string(inputJSON))
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// 解析结果
	designResult, err := ParseTemplateDesignResult(result)
	if err != nil {
		t.Logf("Raw result: %s", result)
	}
	require.NoError(t, err)
	assert.Equal(t, "success", designResult.Status)
	assert.NotEmpty(t, designResult.DirectoryName)
	assert.NotEmpty(t, designResult.Summary)

	// 验证文件是否创建
	if designResult.Status == "success" && len(designResult.FilePaths) > 0 {
		for pageType, filePath := range designResult.FilePaths {
			t.Logf("Checking %s: %s", pageType, filePath)

			// 检查文件是否存在
			_, err := os.Stat(filePath)
			assert.NoError(t, err, "File should exist for %s", pageType)
		}
	}
}

// TestTemplateDesignAgent_ParseOutput 测试HTML输出解析
func TestTemplateDesignAgent_ParseOutput(t *testing.T) {
	// 模拟的Agent输出
	mockOutput := `
生成模板设计中...

<html_block id="cover">
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="utf-8"/>
    <title>封面</title>
</head>
<body>
    <div class="slide-container">封面内容</div>
</body>
</html>
</html_block>

<html_block id="toc">
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="utf-8"/>
    <title>目录</title>
</head>
<body>
    <div class="slide-container">目录内容</div>
</body>
</html>
</html_block>

<html_block id="content">
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="utf-8"/>
    <title>内容</title>
</head>
<body>
    <div class="slide-container">内容页</div>
</body>
</html>
</html_block>

<html_block id="data">
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="utf-8"/>
    <title>数据</title>
</head>
<body>
    <div class="slide-container">数据页</div>
</body>
</html>
</html_block>

<html_block id="ending">
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="utf-8"/>
    <title>结尾</title>
</head>
<body>
    <div class="slide-container">结尾页</div>
</body>
</html>
</html_block>
`

	// 解析输出
	templates, err := ParseTemplateDesignOutput(mockOutput)
	require.NoError(t, err)

	// 验证所有页面都被正确解析
	assert.Contains(t, templates.Cover, "封面内容")
	assert.Contains(t, templates.TOC, "目录内容")
	assert.Contains(t, templates.Content, "内容页")
	assert.Contains(t, templates.Data, "数据页")
	assert.Contains(t, templates.Ending, "结尾页")
}

// TestTemplateDesignAgent_ParseJSONResult 测试JSON结果解析
func TestTemplateDesignAgent_ParseJSONResult(t *testing.T) {
	// 模拟的JSON结果
	mockResult := `
任务完成。

{
  "status": "success",
  "directory_name": "ppt_template_科技_20241201_120000",
  "file_paths": {
    "cover": "/tmp/ppt_template_科技_20241201_120000/cover.html",
    "toc": "/tmp/ppt_template_科技_20241201_120000/toc.html",
    "content": "/tmp/ppt_template_科技_20241201_120000/content.html",
    "data": "/tmp/ppt_template_科技_20241201_120000/data.html",
    "ending": "/tmp/ppt_template_科技_20241201_120000/ending.html"
  },
  "summary": "成功生成科技现代风格的PPT模板，包含5个页面"
}

完成。
`

	// 解析结果
	result, err := ParseTemplateDesignResult(mockResult)
	require.NoError(t, err)

	assert.Equal(t, "success", result.Status)
	assert.Equal(t, "ppt_template_科技_20241201_120000", result.DirectoryName)
	assert.Equal(t, 5, len(result.FilePaths))
	assert.Contains(t, result.Summary, "科技现代风格")

	// 验证所有文件路径都存在
	expectedPages := []string{"cover", "toc", "content", "data", "ending"}
	for _, page := range expectedPages {
		assert.Contains(t, result.FilePaths, page)
		assert.Contains(t, result.FilePaths[page], page+".html")
	}
}

// TestTemplateDesignAgent_DifferentStyles 测试不同风格的模板生成
func TestTemplateDesignAgent_DifferentStyles(t *testing.T) {
	styles := []string{
		"商务简约风格",
		"创意艺术风格",
		"学术严谨风格",
		"活泼卡通风格",
	}

	for _, style := range styles {
		t.Run(style, func(t *testing.T) {
			dirName := GenerateTemplateDirectoryName(style)
			assert.Contains(t, dirName, "ppt_template_")

			// 验证风格关键词提取（最多3个字）
			if len([]rune(style)) > 3 {
				// 确保只取了前3个字符
				styleKey := string([]rune(style)[:3])
				assert.Contains(t, dirName, styleKey)
			}
		})
	}
}

// TestTemplateDesignAgent_WithDependencyInjection 测试依赖注入
// 这个测试验证依赖注入的结构
func TestTemplateDesignAgent_WithDependencyInjection(t *testing.T) {
	if g == nil || sharedTemplateAgent == nil {
		t.Skip("Skipping test: genkit not initialized or shared dependencies not available")
	}

	// 验证共享的 TemplateAgent 存在
	assert.NotNil(t, sharedTemplateAgent)
	assert.Equal(t, "template_design_agent", sharedTemplateAgent.Name())
	assert.Contains(t, sharedTemplateAgent.Description(), "PPT模板设计")
}

// TestTemplateDesignAgent_Integration 集成测试（需要实际调用AI）
func TestTemplateDesignAgent_Integration(t *testing.T) {
	// 跳过集成测试，除非设置了环境变量
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=1 to run")
	}

	if g == nil || sharedTemplateAgent == nil {
		t.Skip("Skipping test: genkit not initialized or shared dependencies not available")
	}

	// 使用共享的 TemplateAgent
	templateAgent := sharedTemplateAgent

	// 测试不同风格
	testCases := []struct {
		name  string
		style string
	}{
		{"Modern", "现代科技风格"},
		{"Business", "商务专业风格"},
		{"Creative", "创意艺术风格"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := map[string]any{
				"style_description": tc.style,
			}
			inputJSON, err := json.Marshal(input)
			require.NoError(t, err)

			// 执行Agent
			result, err := templateAgent.Run(context.Background(), string(inputJSON))
			require.NoError(t, err)

			// 解析结果
			designResult, err := ParseTemplateDesignResult(result)
			require.NoError(t, err)

			if designResult.Status == "success" {
				// 验证所有文件都被创建
				for pageType, filePath := range designResult.FilePaths {
					info, err := os.Stat(filePath)
					require.NoError(t, err)
					assert.True(t, info.Size() > 0, "File %s should not be empty", pageType)

					// 读取并验证HTML内容
					content, err := os.ReadFile(filePath)
					require.NoError(t, err)
					assert.Contains(t, string(content), "<!DOCTYPE html>")
					assert.Contains(t, string(content), "slide-container")
				}

				t.Logf("Successfully generated %s templates in %s", tc.style, designResult.DirectoryName)
			}
		})
	}
}

// 辅助函数被移除：Mock工具在当前架构下不需要

// TestTemplateDesignAgent_ErrorHandling 测试错误处理
func TestTemplateDesignAgent_ErrorHandling(t *testing.T) {
	testCases := []struct {
		name     string
		output   string
		jsonStr  string
		hasError bool
		errMsg   string
	}{
		{
			name:     "Missing HTML blocks",
			output:   "No HTML blocks here",
			hasError: true,
			errMsg:   "未找到有效的HTML块",
		},
		{
			name: "Incomplete HTML blocks",
			output: `
<html_block id="cover">Cover content</html_block>
<html_block id="toc">TOC content</html_block>
`,
			hasError: true,
			errMsg:   "缺少必要的页面模板",
		},
		{
			name:     "Invalid JSON result",
			jsonStr:  "Not a JSON",
			hasError: true,
			errMsg:   "failed to parse",
		},
		{
			name:     "Missing status in JSON",
			jsonStr:  `{"directory_name": "test"}`,
			hasError: true,
			errMsg:   "status field is empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.output != "" {
				_, err := ParseTemplateDesignOutput(tc.output)
				if tc.hasError {
					assert.Error(t, err)
					if tc.errMsg != "" {
						assert.Contains(t, err.Error(), tc.errMsg)
					}
				} else {
					assert.NoError(t, err)
				}
			}

			if tc.jsonStr != "" {
				_, err := ParseTemplateDesignResult(tc.jsonStr)
				if tc.hasError {
					assert.Error(t, err)
					if tc.errMsg != "" {
						assert.Contains(t, err.Error(), tc.errMsg)
					}
				} else {
					assert.NoError(t, err)
				}
			}
		})
	}
}

// TestGenerateTemplateDirectoryName 测试目录名生成
func TestGenerateTemplateDirectoryName(t *testing.T) {
	testCases := []struct {
		style    string
		expected string
	}{
		{"科技", "ppt_template_科技_"},
		{"现代商务", "ppt_template_现代商_"},
		{"简约", "ppt_template_简约_"},
		{"A", "ppt_template_A_"},
	}

	for _, tc := range testCases {
		t.Run(tc.style, func(t *testing.T) {
			dirName := GenerateTemplateDirectoryName(tc.style)
			assert.True(t, strings.HasPrefix(dirName, tc.expected))
			// 验证时间戳部分 - 实际格式是 ppt_template_style_YYYYMMDD_HHMMSS
			parts := strings.Split(dirName, "_")
			assert.Equal(t, 5, len(parts)) // ppt, template, style, YYYYMMDD, HHMMSS
		})
	}
}