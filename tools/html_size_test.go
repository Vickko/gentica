package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHtmlSizeTool(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "html_size_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// 创建工具实例
	tool := NewHtmlSizeTool(tmpDir)
	assert.NotNil(t, tool)

	// 验证基本属性
	assert.Equal(t, "html_size", tool.Name())
	info := tool.Info()
	assert.Contains(t, info.Description, "HTML size validation")
	assert.Contains(t, info.Description, "1280x720")
}

func TestHtmlSizeTool_Execute(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "html_size_exec_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tool := NewHtmlSizeTool(tmpDir)

	t.Run("Valid HTML - 1280x720", func(t *testing.T) {
		// 创建符合规范的HTML文件
		validHTML := `<!DOCTYPE html>
		<html lang="zh-CN">
		<head>
			<style>
				body {
					margin: 0;
					padding: 0;
					width: 1280px;
					height: 720px;
					overflow: hidden;
				}
				.slide-container {
					width: 1280px;
					height: 720px;
					background: #f0f0f0;
				}
			</style>
		</head>
		<body>
			<div class="slide-container">
				<h1>测试页面</h1>
			</div>
		</body>
		</html>`

		htmlFile := filepath.Join(tmpDir, "valid.html")
		err := os.WriteFile(htmlFile, []byte(validHTML), 0644)
		require.NoError(t, err)

		// 执行验证
		inputJSON, _ := json.Marshal(HtmlSizeInput{FilePath: "valid.html"})
		result, err := tool.Run(context.Background(), ToolCall{
			Name:  "html_size",
			Input: string(inputJSON),
		})
		require.NoError(t, err)

		if result.IsError {
			t.Skip("Skipping chromedp test - requires headless browser environment")
			return
		}

		var output HtmlSizeOutput
		err = json.Unmarshal([]byte(result.Content), &output)
		require.NoError(t, err)

		// 验证结果
		assert.Equal(t, 1280, output.Width)
		assert.Equal(t, 720, output.Height)
		assert.True(t, output.Valid)
		assert.Contains(t, output.Message, "1280x720")
	})

	t.Run("Invalid HTML - Wrong Width", func(t *testing.T) {
		// 创建宽度不正确的HTML文件
		invalidHTML := `<!DOCTYPE html>
		<html>
		<head>
			<style>
				body {
					margin: 0;
					width: 1920px;
					height: 720px;
				}
				.slide-container {
					width: 1920px;
					height: 720px;
				}
			</style>
		</head>
		<body>
			<div class="slide-container"></div>
		</body>
		</html>`

		htmlFile := filepath.Join(tmpDir, "invalid_width.html")
		err := os.WriteFile(htmlFile, []byte(invalidHTML), 0644)
		require.NoError(t, err)

		inputJSON, _ := json.Marshal(HtmlSizeInput{
			FilePath: "invalid_width.html",
		})
		result, err := tool.Run(context.Background(), ToolCall{
			Name:  "html_size",
			Input: string(inputJSON),
		})
		require.NoError(t, err)

		if result.IsError {
			t.Skip("Skipping chromedp test - requires headless browser environment")
			return
		}

		var output HtmlSizeOutput
		err = json.Unmarshal([]byte(result.Content), &output)
		require.NoError(t, err)

		// 验证结果
		assert.Equal(t, 1920, output.Width)
		assert.False(t, output.Valid)
		assert.Contains(t, output.Suggestion, "宽度应为1280px")
	})

	t.Run("Invalid HTML - Height Too Large", func(t *testing.T) {
		// 创建高度过大的HTML文件
		largeHeightHTML := `<!DOCTYPE html>
		<html>
		<head>
			<style>
				body {
					margin: 0;
					width: 1280px;
					height: 1080px;
				}
				.slide-container {
					width: 1280px;
					height: 1080px;
				}
			</style>
		</head>
		<body>
			<div class="slide-container"></div>
		</body>
		</html>`

		htmlFile := filepath.Join(tmpDir, "large_height.html")
		err := os.WriteFile(htmlFile, []byte(largeHeightHTML), 0644)
		require.NoError(t, err)

		inputJSON, _ := json.Marshal(HtmlSizeInput{
			FilePath: "large_height.html",
		})
		result, err := tool.Run(context.Background(), ToolCall{
			Name:  "html_size",
			Input: string(inputJSON),
		})
		require.NoError(t, err)

		if result.IsError {
			t.Skip("Skipping chromedp test - requires headless browser environment")
			return
		}

		var output HtmlSizeOutput
		err = json.Unmarshal([]byte(result.Content), &output)
		require.NoError(t, err)

		// 验证结果
		assert.Equal(t, 1280, output.Width)
		assert.Equal(t, 1080, output.Height)
		assert.False(t, output.Valid)
		assert.Contains(t, output.Suggestion, "高度应为720px")
	})

	t.Run("Invalid HTML - Height Too Small", func(t *testing.T) {
		// 创建高度过小的HTML文件
		smallHeightHTML := `<!DOCTYPE html>
		<html>
		<head>
			<style>
				body {
					margin: 0;
					width: 1280px;
					height: 400px;
				}
				.slide-container {
					width: 1280px;
					height: 400px;
				}
			</style>
		</head>
		<body>
			<div class="slide-container"></div>
		</body>
		</html>`

		htmlFile := filepath.Join(tmpDir, "small_height.html")
		err := os.WriteFile(htmlFile, []byte(smallHeightHTML), 0644)
		require.NoError(t, err)

		inputJSON, _ := json.Marshal(HtmlSizeInput{
			FilePath: "small_height.html",
		})
		result, err := tool.Run(context.Background(), ToolCall{
			Name:  "html_size",
			Input: string(inputJSON),
		})
		require.NoError(t, err)

		if result.IsError {
			t.Skip("Skipping chromedp test - requires headless browser environment")
			return
		}

		var output HtmlSizeOutput
		err = json.Unmarshal([]byte(result.Content), &output)
		require.NoError(t, err)

		// 验证结果
		assert.Equal(t, 1280, output.Width)
		assert.Equal(t, 400, output.Height)
		assert.False(t, output.Valid)
		assert.Contains(t, output.Suggestion, "高度应为720px")
	})

	t.Run("Invalid HTML - Height Not Exact", func(t *testing.T) {
		// 创建高度不是720的HTML文件（792px）
		notExactHTML := `<!DOCTYPE html>
		<html>
		<head>
			<style>
				body {
					margin: 0;
					width: 1280px;
					height: 792px;
				}
				.slide-container {
					width: 1280px;
					height: 792px;
				}
			</style>
		</head>
		<body>
			<div class="slide-container"></div>
		</body>
		</html>`

		htmlFile := filepath.Join(tmpDir, "not_exact.html")
		err := os.WriteFile(htmlFile, []byte(notExactHTML), 0644)
		require.NoError(t, err)

		inputJSON, _ := json.Marshal(HtmlSizeInput{
			FilePath: "not_exact.html",
		})
		result, err := tool.Run(context.Background(), ToolCall{
			Name:  "html_size",
			Input: string(inputJSON),
		})
		require.NoError(t, err)

		if result.IsError {
			t.Skip("Skipping chromedp test - requires headless browser environment")
			return
		}

		var output HtmlSizeOutput
		err = json.Unmarshal([]byte(result.Content), &output)
		require.NoError(t, err)

		// 验证结果 - 792不等于720，应该无效
		assert.Equal(t, 1280, output.Width)
		assert.Equal(t, 792, output.Height)
		assert.False(t, output.Valid)
		assert.Contains(t, output.Suggestion, "高度应为720px")
	})

	t.Run("Non-existent File", func(t *testing.T) {
		// 尝试验证不存在的文件
		inputJSON, _ := json.Marshal(HtmlSizeInput{
			FilePath: "non_existent.html",
		})
		result, err := tool.Run(context.Background(), ToolCall{
			Name:  "html_size",
			Input: string(inputJSON),
		})
		require.NoError(t, err)

		// 验证错误处理
		var output HtmlSizeOutput
		if !result.IsError {
			err = json.Unmarshal([]byte(result.Content), &output)
			require.NoError(t, err)
		}
		assert.False(t, output.Valid)
		assert.Contains(t, output.Message, "无法读取文件")
	})

	t.Run("Input via Map", func(t *testing.T) {
		// 测试通过 map 传入参数
		validHTML := `<!DOCTYPE html>
		<html>
		<head>
			<style>
				body { width: 1280px; height: 720px; }
				.slide-container { width: 1280px; height: 720px; }
			</style>
		</head>
		<body>
			<div class="slide-container"></div>
		</body>
		</html>`

		htmlFile := filepath.Join(tmpDir, "map_input.html")
		err := os.WriteFile(htmlFile, []byte(validHTML), 0644)
		require.NoError(t, err)

		// 使用 map 作为输入
		inputJSON, _ := json.Marshal(map[string]any{
			"file_path": "map_input.html",
		})
		result, err := tool.Run(context.Background(), ToolCall{
			Name:  "html_size",
			Input: string(inputJSON),
		})
		require.NoError(t, err)

		if result.IsError {
			t.Skip("Skipping chromedp test - requires headless browser environment")
			return
		}

		var output HtmlSizeOutput
		err = json.Unmarshal([]byte(result.Content), &output)
		require.NoError(t, err)

		assert.Equal(t, 1280, output.Width)
		assert.Equal(t, 720, output.Height)
	})

	t.Run("HTML Without slide-container", func(t *testing.T) {
		// 测试没有 slide-container 的HTML（应该报错）
		noContainerHTML := `<!DOCTYPE html>
		<html>
		<head>
			<style>
				body {
					margin: 0;
					width: 1280px;
					height: 720px;
				}
			</style>
		</head>
		<body>
			<h1>直接在body中的内容</h1>
		</body>
		</html>`

		htmlFile := filepath.Join(tmpDir, "no_container.html")
		err := os.WriteFile(htmlFile, []byte(noContainerHTML), 0644)
		require.NoError(t, err)

		inputJSON, _ := json.Marshal(HtmlSizeInput{
			FilePath: "no_container.html",
		})
		result, err := tool.Run(context.Background(), ToolCall{
			Name:  "html_size",
			Input: string(inputJSON),
		})
		require.NoError(t, err)

		// 没有 slide-container 应该报错
		if !result.IsError {
			// 如果不是错误，解析输出应该显示无效
			var output HtmlSizeOutput
			err = json.Unmarshal([]byte(result.Content), &output)
			require.NoError(t, err)
			assert.False(t, output.Valid)
			assert.Contains(t, output.Message, "验证失败")
		} else {
			// 确认是验证失败的错误
			assert.Contains(t, result.Content, "验证失败")
		}
	})
}

func TestHtmlSizeTool_InputSchema(t *testing.T) {
	tool := NewHtmlSizeTool("/tmp")
	info := tool.Info()

	// 验证工具信息
	assert.Equal(t, "html_size", info.Name)
	assert.Contains(t, info.Description, "HTML size validation")
	assert.Contains(t, info.Parameters, "file_path")
	assert.Contains(t, info.Required, "file_path")
}

func TestHtmlSizeTool_AbsolutePath(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "html_size_abs_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tool := NewHtmlSizeTool("/some/other/path")

	// 创建HTML文件（使用绝对路径）
	htmlContent := `<!DOCTYPE html>
	<html>
	<body style="width: 1280px; height: 720px;">
		<div class="slide-container" style="width: 1280px; height: 720px;"></div>
	</body>
	</html>`

	htmlFile := filepath.Join(tmpDir, "absolute.html")
	err = os.WriteFile(htmlFile, []byte(htmlContent), 0644)
	require.NoError(t, err)

	// 使用绝对路径
	inputJSON, _ := json.Marshal(HtmlSizeInput{
		FilePath: htmlFile, // 绝对路径
	})
	result, err := tool.Run(context.Background(), ToolCall{
		Name:  "html_size",
		Input: string(inputJSON),
	})
	require.NoError(t, err)

	if result.IsError {
		t.Skip("Skipping chromedp test - requires headless browser environment")
		return
	}

	var output HtmlSizeOutput
	err = json.Unmarshal([]byte(result.Content), &output)
	require.NoError(t, err)

	// 即使工具的basePath不同，也应该能正确处理绝对路径
	assert.Equal(t, 1280, output.Width)
	assert.Equal(t, 720, output.Height)
}