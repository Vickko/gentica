package agents

import (
	"os"
	"path/filepath"
	"testing"

	"gentica/tools"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPageGenerateAgent(t *testing.T) {
	// 跳过测试如果 genkit 没有初始化或共享依赖不可用
	if g == nil || sharedPageGenAgent == nil {
		t.Skip("Skipping test: genkit not initialized or shared dependencies not available")
	}

	// 使用共享的 agent（避免重复注册工具）
	agent := sharedPageGenAgent
	assert.NotNil(t, agent)

	// 验证基本属性
	assert.Equal(t, "page_generate_agent", agent.Name())
	assert.Contains(t, agent.Description(), "PPT单页HTML")
}

func TestPageGenerateHelperFunctions(t *testing.T) {
	t.Run("ExtractPageInfo", func(t *testing.T) {
		xmlData := `
		<page>
			<page_type>内容页</page_type>
			<page_title>AI技术应用</page_title>
			<core_content>人工智能在教育领域的创新应用</core_content>
		</page>
		`

		pageType, pageTitle, coreContent := ExtractPageInfo(xmlData)
		assert.Equal(t, "内容页", pageType)
		assert.Equal(t, "AI技术应用", pageTitle)
		assert.Equal(t, "人工智能在教育领域的创新应用", coreContent)
	})

	t.Run("GetTemplateFileByPageType", func(t *testing.T) {
		// 测试不同页面类型对应的模板文件名
		assert.Equal(t, "cover.html", GetTemplateFileByPageType("封面页"))
		assert.Equal(t, "toc.html", GetTemplateFileByPageType("目录页"))
		assert.Equal(t, "content.html", GetTemplateFileByPageType("内容页"))
		assert.Equal(t, "data.html", GetTemplateFileByPageType("数据页"))
		assert.Equal(t, "ending.html", GetTemplateFileByPageType("结尾页"))

		// 测试未知类型，应该返回默认值
		assert.Equal(t, "content.html", GetTemplateFileByPageType("未知类型"))
	})

	t.Run("ValidateGeneratedHTML", func(t *testing.T) {
		// 有效的HTML
		validHTML := `<!DOCTYPE html>
		<html>
		<head>
			<style>
				body { width: 1280px; height: 720px; overflow: hidden; }
			</style>
		</head>
		<body>
			<div class="slide-container"></div>
		</body>
		</html>`

		err := ValidateGeneratedHTML(validHTML)
		assert.NoError(t, err)

		// 缺少DOCTYPE
		invalidHTML1 := `<html><body></body></html>`
		err = ValidateGeneratedHTML(invalidHTML1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DOCTYPE")

		// 缺少尺寸
		invalidHTML2 := `<!DOCTYPE html><html><body></body></html>`
		err = ValidateGeneratedHTML(invalidHTML2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "尺寸")
	})

	t.Run("GeneratePageDirectoryName", func(t *testing.T) {
		name := GeneratePageDirectoryName()
		assert.Contains(t, name, "page_generate_")
		// Timestamp format: 20060102_150405 = YYYYMMDD_HHMMSS = 15 characters
		assert.Len(t, name, 29) // page_generate_ (14) + timestamp (15)
	})

	t.Run("GenerateIterationFileName", func(t *testing.T) {
		assert.Equal(t, "iterations/attempt_1.html", GenerateIterationFileName(1))
		assert.Equal(t, "iterations/attempt_5.html", GenerateIterationFileName(5))
	})

	t.Run("FormatPageGenerateInput", func(t *testing.T) {
		input := PageGenerateInput{
			XMLData: "<page><page_type>内容页</page_type></page>",
			Templates: []string{
				"<html>模板1</html>",
				"<html>模板2</html>",
			},
		}

		formatted := FormatPageGenerateInput(input)
		assert.Contains(t, formatted, "XML数据")
		assert.Contains(t, formatted, input.XMLData)
		assert.Contains(t, formatted, "模板1")
		assert.Contains(t, formatted, "模板2")
		assert.Contains(t, formatted, "---模板分隔---")
	})

	t.Run("ParsePageGenerateResult", func(t *testing.T) {
		// 有效的JSON结果
		validJSON := `{
			"status": "success",
			"directory_name": "page_generate_20241225",
			"file_path": "/tmp/page_generate_20241225/final.html",
			"iterations": 3,
			"message": "生成成功"
		}`

		result, err := ParsePageGenerateResult(validJSON)
		require.NoError(t, err)
		assert.Equal(t, "success", result.Status)
		assert.Equal(t, "page_generate_20241225", result.DirectoryName)
		assert.Equal(t, "/tmp/page_generate_20241225/final.html", result.FilePath)
		assert.Equal(t, 3, result.Iterations)
		assert.Equal(t, "生成成功", result.Message)

		// 混合文本中的JSON
		mixedText := `Agent生成了以下结果：
		{
			"status": "success",
			"directory_name": "test_dir",
			"file_path": "test.html",
			"iterations": 1,
			"message": "OK"
		}
		任务完成。`

		result, err = ParsePageGenerateResult(mixedText)
		require.NoError(t, err)
		assert.Equal(t, "success", result.Status)
		assert.Equal(t, "test_dir", result.DirectoryName)

		// 无效的JSON
		invalidJSON := `{"incomplete": `
		_, err = ParsePageGenerateResult(invalidJSON)
		assert.Error(t, err)

		// 缺少必需字段
		incompleteJSON := `{"directory_name": "test"}`
		_, err = ParsePageGenerateResult(incompleteJSON)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "status")
	})
}

// TestPageGenerateAgentIntegration 集成测试（需要手动运行，因为需要实际的工具依赖）
func TestPageGenerateAgentIntegration(t *testing.T) {
	// 跳过集成测试，因为工具重复注册问题
	t.Skip("Skipping integration test: tools cannot be registered multiple times")

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 跳过测试如果 genkit 没有初始化
	if g == nil {
		t.Skip("Skipping test: genkit not initialized (run TestMain to initialize)")
	}

	// 创建临时工作目录
	tmpDir, err := os.MkdirTemp("", "page_generate_integration_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// 创建模拟依赖
	deps := &PageGenerateAgentDependencies{
		DirectoryAddTool: tools.AdaptBaseToolToGenkit(g, tools.NewResourceDirectoryAddTool(tmpDir)),
		WriteTool:        tools.AdaptBaseToolToGenkit(g, tools.NewWriteTool(tmpDir)),
		ViewTool:         tools.AdaptBaseToolToGenkit(g, tools.NewViewTool(tmpDir)),
		HtmlSizeTool:     tools.AdaptBaseToolToGenkit(g, tools.NewHtmlSizeTool(tmpDir)),
	}

	// 创建 agent
	agent := NewPageGenerateAgentWithDeps(g, deps)
	assert.NotNil(t, agent)

	// 准备测试数据 - 将在实际测试时使用
	_ = `
	<page>
		<page_number>3</page_number>
		<page_title>AI技术在教育的应用</page_title>
		<page_type>内容页</page_type>
		<core_content>个性化学习路径、智能批改、虚拟助教</core_content>
	</page>
	`

	htmlTemplate := `<!DOCTYPE html>
	<html lang="zh-CN">
	<head>
		<meta charset="utf-8"/>
		<style>
			body {
				margin: 0;
				padding: 0;
				overflow: hidden;
				width: 1280px;
				height: 720px;
			}
			.slide-container {
				width: 1280px;
				height: 720px;
				padding: 60px;
				background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
				display: flex;
				flex-direction: column;
				justify-content: center;
				align-items: center;
				color: white;
			}
			h1 { font-size: 48px; margin-bottom: 30px; }
			p { font-size: 24px; text-align: center; }
		</style>
	</head>
	<body>
		<div class="slide-container">
			<h1>[页面标题]</h1>
			<p>[核心内容]</p>
		</div>
	</body>
	</html>`

	// 保存模板文件供测试
	templatePath := filepath.Join(tmpDir, "test_template.html")
	err = os.WriteFile(templatePath, []byte(htmlTemplate), 0644)
	require.NoError(t, err)

	t.Log("Integration test setup complete")
	t.Logf("Working directory: %s", tmpDir)
	t.Logf("Template saved at: %s", templatePath)

	// 注意：实际的Agent执行需要连接到AI服务，这里只是准备了测试环境
	// 实际测试时可以使用mock或者需要配置真实的AI服务
}