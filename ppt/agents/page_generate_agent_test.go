package agents

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gentica/tools"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPageGenerateAgent(t *testing.T) {
	// 复用包级别配置，检查依赖是否可用
	if g == nil || sharedPageGenAgent == nil {
		t.Skip("Skipping test: genkit not initialized or shared dependencies not available")
	}

	// 准备测试数据 - 大纲XML文件
	outlineXML := `<?xml version="1.0" encoding="UTF-8"?>
<ppt_plan>
	<metadata>
		<title>AI技术在教育领域的应用</title>
		<total_pages>5</total_pages>
		<target_audience>教育工作者与技术研发人员</target_audience>
		<duration>20分钟</duration>
	</metadata>
	<page_structure>
		<page>
			<page_number>1</page_number>
			<page_type>封面页</page_type>
			<page_title>AI技术在教育领域的应用</page_title>
			<core_content>人工智能赋能教育创新：智能化教学的未来</core_content>
		</page>
		<page>
			<page_number>2</page_number>
			<page_type>目录页</page_type>
			<page_title>内容概览</page_title>
			<core_content>1. AI教育现状\n2. 核心技术应用\n3. 实践案例分析\n4. 未来发展趋势</core_content>
		</page>
		<page>
			<page_number>3</page_number>
			<page_type>内容页</page_type>
			<page_title>核心技术应用</page_title>
			<core_content>个性化学习系统：根据学生学习行为和能力自动调整教学内容和进度\n智能批改与反馈：自动批改作业并提供个性化学习建议\n虚拟教学助手：24小时在线答疑，辅助教师教学</core_content>
			<images>
				<image>
					<description>AI教育技术架构图</description>
					<position>右侧配图</position>
				</image>
			</images>
		</page>
		<page>
			<page_number>4</page_number>
			<page_type>数据页</page_type>
			<page_title>教育AI市场增长数据</page_title>
			<core_content>2024年全球教育AI市场规模达40亿美元\n预计2030年将增长至200亿美元\n年复合增长率超过30%</core_content>
			<images>
				<image>
					<description>市场增长趋势图表</description>
					<position>中心大图</position>
				</image>
			</images>
		</page>
		<page>
			<page_number>5</page_number>
			<page_type>结尾页</page_type>
			<page_title>谢谢</page_title>
			<core_content>感谢聆听\n让AI为教育赋能，共创智慧教育新未来</core_content>
		</page>
	</page_structure>
</ppt_plan>`

	// 准备模板文件 - 科技风格的内容页模板
	contentTemplate := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8"/>
<meta content="width=device-width, initial-scale=1.0" name="viewport"/>
<title>内容页</title>
<link href="https://static-cdn-test.camscanner.com/genspark/css/tailwind.min.css" rel="stylesheet"/>
<link href="https://static-cdn-test.camscanner.com/genspark/css/fontawesome.min.css" rel="stylesheet"/>
<style>
body {
  margin: 0;
  padding: 0;
  overflow: hidden;
  width: 1280px;
  height: 720px;
}
* { box-sizing: border-box; }
img { object-fit: cover; }
.slide-container {
  width: 1280px;
  height: 720px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  display: flex;
  align-items: center;
  padding: 80px;
  color: white;
}
.content-area {
  flex: 1;
  padding-right: 40px;
}
.image-area {
  width: 400px;
  height: 400px;
}
.image-area img {
  width: 100%;
  height: 100%;
  border-radius: 20px;
}
h1 {
  font-size: 48px;
  font-weight: bold;
  margin-bottom: 30px;
  text-shadow: 2px 2px 4px rgba(0,0,0,0.2);
}
.bullet-points {
  font-size: 20px;
  line-height: 1.8;
}
.bullet-points li {
  margin-bottom: 20px;
  list-style: none;
  position: relative;
  padding-left: 30px;
}
.bullet-points li:before {
  content: "▸";
  position: absolute;
  left: 0;
  color: #fbbf24;
}
</style>
</head>
<body>
<div class="slide-container">
  <div class="content-area">
    <h1>[页面标题]</h1>
    <ul class="bullet-points">
      <li>[要点1]</li>
      <li>[要点2]</li>
      <li>[要点3]</li>
    </ul>
  </div>
  <div class="image-area">
    <img src="placeholder.jpg" alt="配图">
  </div>
</div>
</body>
</html>`

	// 创建测试工作目录
	testDir := filepath.Join(sharedTempDir, "page_generate_test")
	err := os.MkdirAll(testDir, 0755)
	require.NoError(t, err)

	// 保存大纲文件
	outlinePath := filepath.Join(testDir, "outline.xml")
	err = os.WriteFile(outlinePath, []byte(outlineXML), 0644)
	require.NoError(t, err)

	// 创建模板目录并保存模板文件
	templateDir := filepath.Join(testDir, "templates")
	err = os.MkdirAll(templateDir, 0755)
	require.NoError(t, err)

	templatePath := filepath.Join(templateDir, "content.html")
	err = os.WriteFile(templatePath, []byte(contentTemplate), 0644)
	require.NoError(t, err)

	// 准备输入参数 - 生成第3页（内容页）
	input := map[string]any{
		"outline_path": outlinePath,
		"template_dir": templateDir,
		"page_number":  3,
	}

	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	// 调用 PageGenerateAgent - 发送真实API请求验证完整流程
	t.Logf("执行页面生成 Agent")
	t.Logf("  - 大纲文件: %s", outlinePath)
	t.Logf("  - 模板目录: %s", templateDir)
	t.Logf("  - 生成页码: %d", input["page_number"])

	result, err := sharedPageGenAgent.Run(context.Background(), string(inputJSON))
	require.NoError(t, err)
	require.NotEmpty(t, result)

	// 解析和验证结果
	pageResult, err := ParsePageGenerateResult(result)
	require.NoError(t, err)
	require.NotNil(t, pageResult)

	// 验证生成状态
	assert.Equal(t, "success", pageResult.Status)
	assert.NotEmpty(t, pageResult.DirectoryName)
	assert.NotEmpty(t, pageResult.FilePath)
	assert.Contains(t, pageResult.DirectoryName, "page_generate_")

	// 验证迭代次数（应该在合理范围内）
	assert.GreaterOrEqual(t, pageResult.Iterations, 1)
	assert.LessOrEqual(t, pageResult.Iterations, 5)

	// 验证生成的HTML文件存在
	_, err = os.Stat(pageResult.FilePath)
	assert.NoError(t, err, "生成的HTML文件应该存在")

	// 读取并验证生成的HTML内容
	htmlContent, err := os.ReadFile(pageResult.FilePath)
	require.NoError(t, err)
	htmlStr := string(htmlContent)

	// 验证HTML基本结构
	assert.Contains(t, htmlStr, "<!DOCTYPE html>")
	assert.Contains(t, htmlStr, "1280px")
	assert.Contains(t, htmlStr, "720px")
	assert.Contains(t, htmlStr, "slide-container")
	assert.Contains(t, htmlStr, "overflow: hidden")

	// 验证内容已被替换
	assert.NotContains(t, htmlStr, "[页面标题]", "页面标题应该已被替换")
	assert.NotContains(t, htmlStr, "[要点1]", "要点内容应该已被替换")
	assert.NotContains(t, htmlStr, "placeholder.jpg", "图片应该已被替换为占位符格式")

	// 验证图片占位符格式
	assert.Contains(t, htmlStr, "__PIC_SRC__", "应包含图片占位符")

	// 输出测试结果摘要
	t.Logf("=== 页面生成测试结果 ===")
	t.Logf("状态: %s", pageResult.Status)
	t.Logf("资源目录: %s", pageResult.DirectoryName)
	t.Logf("最终文件: %s", pageResult.FilePath)
	t.Logf("迭代次数: %d", pageResult.Iterations)
	t.Logf("消息: %s", pageResult.Message)

	// 输出HTML前500个字符作为预览
	if len(htmlStr) > 500 {
		t.Logf("HTML预览:\n%s...", htmlStr[:500])
	} else {
		t.Logf("HTML预览:\n%s", htmlStr)
	}
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