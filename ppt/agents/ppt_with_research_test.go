package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"gentica/tools"

	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/openai/openai-go/option"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPageGenerateWithRealResources 测试使用真实资源生成PPT页面
func TestPageGenerateWithRealResources(t *testing.T) {
	// 初始化 Genkit
	apiKey := "sk-6kgtZQDkmZDQMfCo28C360320cEf45FaAf1577Ef08F4032b"
	baseURL := "https://aihubmix.com/v1"

	oai := &openai.OpenAI{
		APIKey: apiKey,
		Opts: []option.RequestOption{
			option.WithBaseURL(baseURL),
		},
	}

	g := genkit.Init(
		context.Background(),
		genkit.WithPlugins(oai),
	)

	// 使用实际的资源目录
	baseDir := "/var/folders/ny/dlt8f5l11cnfcpkgsb4cs8bhqspg91/T/research_collector_test_593932661/.tmp/b8d01e"
	outlinePath := baseDir + "/ppt_outline_20250923/outline.xml"
	templateDir := baseDir + "/ppt_template_techedu_20250923_143000"
	researchDir := baseDir + "/research_人工智能教育_0923"

	// 验证资源文件是否存在
	if _, err := os.Stat(outlinePath); os.IsNotExist(err) {
		t.Skipf("Outline file not found: %s", outlinePath)
	}
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		t.Skipf("Template directory not found: %s", templateDir)
	}
	if _, err := os.Stat(researchDir); os.IsNotExist(err) {
		t.Skipf("Research directory not found: %s", researchDir)
	}

	// 创建工作目录
	workingDir, err := os.MkdirTemp("", "page_generate_real_test_*")
	require.NoError(t, err)
	defer func() {
		// 保留生成的文件以供检查
		t.Logf("Generated files saved at: %s", workingDir)
		// 如果需要清理，取消下面的注释
		// os.RemoveAll(workingDir)
	}()

	// 创建页面生成agent的依赖
	deps := &PageGenerateAgentDependencies{
		DirectoryAddTool:  tools.AdaptBaseToolToGenkit(g, tools.NewResourceDirectoryAddTool(workingDir)),
		DirectoryListTool: tools.AdaptBaseToolToGenkit(g, tools.NewResourceDirectoryListTool(workingDir)),
		WriteTool:         tools.AdaptBaseToolToGenkit(g, tools.NewWriteTool(workingDir)),
		ViewTool:          tools.AdaptBaseToolToGenkit(g, tools.NewViewTool(workingDir)),
		HtmlSizeTool:      tools.AdaptBaseToolToGenkit(g, tools.NewHtmlSizeTool(workingDir)),
		LsTool:            tools.AdaptBaseToolToGenkit(g, tools.NewLsTool(workingDir)),
		BashTool:          tools.AdaptBaseToolToGenkit(g, tools.NewBashTool(workingDir)),
	}

	// 创建页面生成agent
	pageGenAgent := NewPageGenerateAgentWithDeps(g, deps)
	require.NotNil(t, pageGenAgent)

	// 准备输入参数 - 生成第6页（AI的价值与机遇：个性化学习路径）
	input := map[string]any{
		"outline_path":        outlinePath,
		"template_dir":        templateDir,
		"page_number":         6,
		"research_directory": researchDir, // 包含研究资料目录
	}

	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	// 输出测试信息
	t.Logf("=== 开始生成PPT页面 ===")
	t.Logf("大纲文件: %s", outlinePath)
	t.Logf("模板目录: %s", templateDir)
	t.Logf("研究资料: %s", researchDir)
	t.Logf("生成页码: %d (AI的价值与机遇：个性化学习路径)", 6)
	t.Logf("工作目录: %s", workingDir)

	// 执行页面生成
	ctx := context.Background()
	result, err := pageGenAgent.Run(ctx, string(inputJSON))
	require.NoError(t, err, "页面生成失败")
	require.NotEmpty(t, result, "返回结果为空")

	// 解析结果
	pageResult, err := ParsePageGenerateResult(result)
	require.NoError(t, err, "解析结果失败")
	require.NotNil(t, pageResult)

	// 验证生成状态
	assert.Equal(t, "success", pageResult.Status, "生成状态应为success")
	assert.NotEmpty(t, pageResult.DirectoryName, "资源目录名称不应为空")
	assert.NotEmpty(t, pageResult.FilePath, "文件路径不应为空")

	// 验证final.html文件存在
	assert.Contains(t, pageResult.FilePath, "final.html", "文件路径应包含final.html")

	// 验证迭代次数
	assert.GreaterOrEqual(t, pageResult.Iterations, 1, "至少应有1次迭代")
	assert.LessOrEqual(t, pageResult.Iterations, 16, "迭代次数不应超过16次")

	// 尝试各种可能的路径找到生成的HTML文件
	var htmlContent []byte
	var fullFilePath string

	// 先尝试直接使用返回的路径
	if pageResult.FilePath != "" {
		// 清理路径（处理可能的描述性文本）
		cleanPath := pageResult.FilePath
		if idx := strings.Index(cleanPath, "("); idx > 0 {
			cleanPath = strings.TrimSpace(cleanPath[:idx])
		}
		if strings.Contains(cleanPath, "->") {
			parts := strings.Split(cleanPath, "->")
			cleanPath = strings.TrimSpace(parts[len(parts)-1])
		}

		// 如果是相对路径，拼接工作目录
		if !strings.HasPrefix(cleanPath, "/") {
			cleanPath = workingDir + "/" + cleanPath
		}

		htmlContent, err = os.ReadFile(cleanPath)
		if err == nil {
			fullFilePath = cleanPath
		}
	}

	// 如果失败，尝试标准路径
	if fullFilePath == "" {
		possiblePaths := []string{
			workingDir + "/final.html",
			workingDir + "/iterations/attempt_1.html",
		}

		for _, path := range possiblePaths {
			if content, err := os.ReadFile(path); err == nil {
				htmlContent = content
				fullFilePath = path
				t.Logf("找到文件: %s", path)
				break
			}
		}
	}

	require.NotNil(t, htmlContent, "无法找到生成的HTML文件")
	require.NotEmpty(t, fullFilePath, "文件路径为空")
	htmlStr := string(htmlContent)

	// 验证HTML基本结构
	t.Log("=== 验证HTML结构 ===")
	// DOCTYPE可能是大写或小写
	assert.True(t, strings.Contains(strings.ToLower(htmlStr), "<!doctype html>"), "应包含DOCTYPE声明")
	assert.Contains(t, htmlStr, "slide-container", "应包含slide-container元素")
	assert.Contains(t, htmlStr, "1280", "应包含宽度1280")
	assert.Contains(t, htmlStr, "720", "应包含高度720")
	assert.Contains(t, htmlStr, "overflow", "应包含overflow样式")

	// 验证内容已被替换
	t.Log("=== 验证内容替换 ===")
	assert.NotContains(t, htmlStr, "[页面标题]", "模板占位符应已被替换")
	assert.NotContains(t, htmlStr, "[要点1]", "模板占位符应已被替换")
	assert.NotContains(t, htmlStr, "教学活动设计", "模板默认内容应已被替换")

	// 验证是否包含个性化学习相关内容
	t.Log("=== 验证研究资料融入 ===")
	lowerHTML := strings.ToLower(htmlStr)
	hasRelevantContent := false

	// 检查是否包含相关关键词
	relevantKeywords := []string{
		"个性化", "自适应", "学习路径", "算法", "推荐",
		"学生", "教育", "ai", "人工智能", "智能",
	}

	foundKeywords := []string{}
	for _, keyword := range relevantKeywords {
		if strings.Contains(lowerHTML, keyword) {
			hasRelevantContent = true
			foundKeywords = append(foundKeywords, keyword)
		}
	}

	assert.True(t, hasRelevantContent, "页面应包含与个性化学习相关的内容")
	t.Logf("找到的相关关键词: %v", foundKeywords)

	// 验证图片占位符（如果模板中有图片）
	if strings.Contains(htmlStr, "<img") {
		assert.Contains(t, htmlStr, "__PIC_SRC__", "图片src应被替换为占位符格式")
		t.Log("图片已正确替换为占位符格式")
	}

	// 输出结果摘要
	t.Log("=== 生成结果摘要 ===")
	t.Logf("状态: %s", pageResult.Status)
	t.Logf("资源目录: %s", pageResult.DirectoryName)
	t.Logf("最终文件: %s", pageResult.FilePath)
	t.Logf("迭代次数: %d", pageResult.Iterations)
	t.Logf("消息: %s", pageResult.Message)

	// 输出HTML预览（前800个字符）
	previewLength := 800
	if len(htmlStr) > previewLength {
		t.Logf("\n=== HTML内容预览 ===\n%s...\n[总长度: %d字符]",
			htmlStr[:previewLength], len(htmlStr))
	} else {
		t.Logf("\n=== HTML内容 ===\n%s", htmlStr)
	}

	// 提取并显示主要文本内容（去除HTML标签）
	t.Log("\n=== 页面文本内容提取 ===")
	textContent := extractTextFromHTML(htmlStr)
	if len(textContent) > 500 {
		t.Logf("%s...", textContent[:500])
	} else {
		t.Logf("%s", textContent)
	}
}

// extractTextFromHTML 简单提取HTML中的文本内容
func extractTextFromHTML(html string) string {
	// 移除script和style标签内容
	html = removeTagContent(html, "script")
	html = removeTagContent(html, "style")

	// 移除所有HTML标签
	inTag := false
	var result strings.Builder

	for _, ch := range html {
		if ch == '<' {
			inTag = true
			result.WriteString(" ")
		} else if ch == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(ch)
		}
	}

	// 清理多余空格和换行
	text := result.String()
	lines := strings.Split(text, "\n")
	var cleanLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, "\n")
}

// removeTagContent 移除指定标签及其内容
func removeTagContent(html, tag string) string {
	startTag := fmt.Sprintf("<%s", tag)
	endTag := fmt.Sprintf("</%s>", tag)

	for {
		startIdx := strings.Index(html, startTag)
		if startIdx == -1 {
			break
		}

		endIdx := strings.Index(html[startIdx:], endTag)
		if endIdx == -1 {
			break
		}

		endIdx += startIdx + len(endTag)
		html = html[:startIdx] + html[endIdx:]
	}

	return html
}

// TestPageGenerateWithDifferentPages 测试生成不同类型的页面
func TestPageGenerateWithDifferentPages(t *testing.T) {
	// 可以测试不同页码的生成
	testCases := []struct {
		pageNumber int
		pageType   string
		pageTitle  string
	}{
		{1, "封面页", "人工智能在教育领域的应用与实践"},
		{2, "目录页", "议程概览"},
		{7, "内容页", "AI的价值与机遇：智能评估与学习分析"},
		{12, "内容页", "实施路径与落地建议：分阶段实施与技术选型"},
		{14, "结尾页", "谢谢观看"},
	}

	// 这里可以根据需要添加更多测试用例
	_ = testCases
	t.Log("可以测试更多页面类型，暂时跳过")
}