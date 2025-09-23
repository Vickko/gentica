package agents

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gentica/agent"
	"gentica/tools"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// PageGenerateInput 页面生成输入
type PageGenerateInput struct {
	XMLData   string   `json:"xml_data"`   // PPT单页的XML数据
	Templates []string `json:"templates"`  // 模板代码列表
}

// PageGenerateResult 页面生成结果
type PageGenerateResult struct {
	HTMLContent   string `json:"html_content"`   // 生成的HTML内容
	FilePath      string `json:"file_path"`      // 最终文件路径
	DirectoryName string `json:"directory_name"` // 资源目录名称
	Iterations    int    `json:"iterations"`     // 迭代次数
	Status        string `json:"status"`         // 任务状态：success 或 failed
	Message       string `json:"message"`        // 状态消息
}

// PageGenerateAgentDependencies 页面生成器的依赖
type PageGenerateAgentDependencies struct {
	DirectoryAddTool  ai.Tool // 资源目录创建工具
	DirectoryListTool ai.Tool // 资源目录列表工具
	WriteTool         ai.Tool // 文件写入工具
	ViewTool          ai.Tool // 文件查看工具
	HtmlSizeTool      ai.Tool // HTML尺寸验证工具
	LsTool            ai.Tool // 目录列表工具
}

// NewPageGenerateAgent 创建页面生成 Agent（使用默认依赖）
func NewPageGenerateAgent(g *genkit.Genkit, workingDir string) agent.Agent {
	// 创建默认工具
	deps := &PageGenerateAgentDependencies{
		DirectoryAddTool:  tools.AdaptBaseToolToGenkit(g, tools.NewResourceDirectoryAddTool(workingDir)),
		DirectoryListTool: tools.AdaptBaseToolToGenkit(g, tools.NewResourceDirectoryListTool(workingDir)),
		WriteTool:         tools.AdaptBaseToolToGenkit(g, tools.NewWriteTool(workingDir)),
		ViewTool:          tools.AdaptBaseToolToGenkit(g, tools.NewViewTool(workingDir)),
		HtmlSizeTool:      tools.AdaptBaseToolToGenkit(g, tools.NewHtmlSizeTool(workingDir)),
		LsTool:            tools.AdaptBaseToolToGenkit(g, tools.NewLsTool(workingDir)),
	}
	return NewPageGenerateAgentWithDeps(g, deps)
}

// NewPageGenerateAgentWithDeps 创建带依赖注入的页面生成 Agent
func NewPageGenerateAgentWithDeps(g *genkit.Genkit, deps *PageGenerateAgentDependencies) agent.Agent {
	// 系统提示
	systemPrompt := `你是一个专业的PPT页面生成专家。请根据大纲和模板，生成符合规范的HTML PPT单页。

## 任务描述
根据提供的大纲文件路径、模板目录和页码，生成指定的PPT页面HTML。

## 生成规则

### 1. 内容提取与处理
- 从 XML 中提取：
  - 主题（page_title）
  - 副标题/核心内容（core_content）
  - 类型（page_type）
  - 图片信息（如有）
- 如果提供了research_directory：
  - 使用ls工具列出资料目录中的文件
  - 根据页面主题查找相关的.md资料文件
  - 使用view工具读取相关资料，充实页面内容
  - 在生成的HTML中添加meta标签记录使用的资料：
    <meta name="research-source" content="使用的资料文件名.md">

### 2. 精简与格式优化
- 删除冗余字词，保留核心观点
- 对较长内容进行摘要化或分点列表化
- 限制总字数 ≤ 200 字，总行数 ≤ 6 行，避免溢出
- 保持模板整体布局、配色和风格不被破坏，仅进行必要的文字与图片替换

### 3. 图片替换规则
模板中如果存在 <img> 元素：
- 保留 <img> 标签及其原有的 class、style、宽高等属性
- 仅替换其 src 属性为图片关键词和宽高比占位字符串，格式如下：
  src="__PIC_SRC__{keyword,ratio}"

其中：
- keyword = 可读英文短语（Title Case，1~6 个英文单词）
- ratio = 宽高比小数（宽 ÷ 高，保留两位小数，例如 1.78 表示约 16:9）
- 如果缺少 size 信息，由你根据布局推理
- alt 属性直接与 keyword 一致
- 宽度、高度在 HTML 标签中保持（来自原数据或合理推断）
- 超出图片宽度时，直接截断

图片宽高比生成原则：
1. 如果 <img> 标签提供了 width 和 height 数值：ratio = width / height
2. 否则，根据父容器 CSS 尺寸或布局推测
3. 保留 2 位小数（四舍五入）
4. 常见比例参考：1.78≈16:9，1.33≈4:3，1.00=1:1

图片关键词生成原则：
1. 关键词将直接用于 Google Image Search API 搜索
2. 必须是**准确、易于检索的英文短语**，长度建议 1~6 个英文单词
3. 需与页面主题、核心内容或图像用途高度相关
4. 如果原始数据不存在描述，则结合 XML 的 core_content 或场景推断
5. 避免模糊词（如 thing, object, stuff）和无意义形容词（如 nice, beautiful, amazing）
6. 如果 <img> 为背景/装饰类，可以用抽象风格短语（如 Blue Gradient Lines、Data Light Effect）

禁止：
- 在模板中原本没有 <img> 的地方新增图片
- 删除 <img> 标签或替换为其他标签类型

## 固定基础框架
<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8"/>
<meta content="width=device-width, initial-scale=1.0" name="viewport"/>
<title>[PPT标题]</title>
<link href="https://static-cdn-test.camscanner.com/genspark/css/tailwind.min.css" rel="stylesheet"/>
<link href="https://static-cdn-test.camscanner.com/genspark/css/fontawesome.min.css" rel="stylesheet"/>
<link href="https://static-cdn-test.camscanner.com/genspark/css/fonts-googleapis.css" rel="stylesheet"/>
<style>
body {
  margin: 0;
  padding: 0;
  overflow: hidden;
  width: 1280px;
  height: 720px;
}
* { box-sizing: border-box; }
img {
  object-fit: cover;
}
</style>
</head>
<body>
<!-- 内容在这里 -->
</body>
</html>

## 布局安全准则
✅ 必须：
- 必须包含 .slide-container 元素
- .slide-container 尺寸必须严格是 1280x720px
- 使用 px 固定高度
- overflow: hidden 防止溢出
- 统一 box-sizing

❌ 禁止：
- height: auto 或无高度
- 不当混用 flex + absolute + float
- 依赖内容撑开容器高度
- 使用响应式单位破坏固定布局
- 缺少 .slide-container 元素

## 生成检查清单
[ ] 包含 .slide-container 元素
[ ] 容器严格固定为 1280x720px（不允许偏差）
[ ] overflow:hidden
[ ] 所有 <img> 标签保留并已替换 src 为占位符格式
[ ] 未新增额外图片
[ ] 内容不溢出
[ ] 小屏下无水平滚动条

## 工作流程

1. 读取大纲文件（使用view工具）
2. 解析XML获取指定页码的页面数据（page_type, page_title, core_content）
3. 如果提供了research_directory：
   - 使用ls工具列出资料目录中的文件
   - 根据页面主题选择相关的资料文件
   - 使用view工具读取资料内容
4. 根据页面类型（page_type）读取对应的模板文件：
   - 封面页 -> cover.html
   - 目录页 -> toc.html
   - 内容页 -> content.html
   - 数据页 -> data.html
   - 结尾页 -> ending.html
5. 创建资源目录（格式：page_generate_[timestamp]_p[页码]）
6. 将页面内容替换到模板中生成HTML
   - 如果使用了研究资料，添加meta标签记录来源
7. 保存HTML文件（iterations/attempt_1.html）
8. 使用html_size工具验证尺寸
9. 如果尺寸不符合（必须严格是1280x720）：
   - 调整HTML代码
   - 保存新版本（iterations/attempt_2.html）
   - 重新验证
10. 最多迭代5次
11. 将最终版本保存为final.html
12. 返回结果

## 返回格式
完成所有工作后，返回以下JSON格式的结果：
{
  "status": "success" 或 "failed",
  "directory_name": "资源目录名称",
  "file_path": "最终HTML文件路径",
  "iterations": 迭代次数,
  "message": "生成结果说明"
}`

	// 准备工具列表
	toolList := []ai.Tool{}
	if deps != nil {
		if deps.DirectoryAddTool != nil {
			toolList = append(toolList, deps.DirectoryAddTool)
		}
		if deps.DirectoryListTool != nil {
			toolList = append(toolList, deps.DirectoryListTool)
		}
		if deps.WriteTool != nil {
			toolList = append(toolList, deps.WriteTool)
		}
		if deps.ViewTool != nil {
			toolList = append(toolList, deps.ViewTool)
		}
		if deps.HtmlSizeTool != nil {
			toolList = append(toolList, deps.HtmlSizeTool)
		}
		if deps.LsTool != nil {
			toolList = append(toolList, deps.LsTool)
		}
	}

	// 创建并返回 Agent
	return agent.NewBuilder(
		g,
		"page_generate_agent",
		"生成PPT单页HTML",
		systemPrompt,
	).WithInputSchema(
		map[string]any{
			"outline_path": map[string]any{
				"type":        "string",
				"description": "PPT大纲XML文件路径",
			},
			"template_dir": map[string]any{
				"type":        "string",
				"description": "模板目录路径",
			},
			"page_number": map[string]any{
				"type":        "integer",
				"description": "要生成的页码（从1开始）",
			},
			"research_directory": map[string]any{
				"type":        "string",
				"description": "研究资料目录路径（可选）",
			},
		},
		"outline_path", "template_dir", "page_number", // 必需字段
	).WithTools(toolList...).
		WithModel("openai/gpt-5-mini"). // 使用gpt-5-mini
		WithTemperature(0.5).            // 代码生成需要更确定性的输出
		WithMaxTokens(10000).            // 代码生成需要较多token
		WithMaxRounds(10).               // 支持多轮迭代
		WithLogging(true).
		Build()
}

// FormatPageGenerateInput 格式化输入以便传递给Agent
func FormatPageGenerateInput(input PageGenerateInput) string {
	// 将模板列表合并为单个字符串
	templatesStr := strings.Join(input.Templates, "\n\n---模板分隔---\n\n")

	return fmt.Sprintf(`XML数据：
%s

可用模板：
%s`, input.XMLData, templatesStr)
}

// ParsePageGenerateResult 解析Agent返回的JSON结果
func ParsePageGenerateResult(result string) (*PageGenerateResult, error) {
	var pageResult struct {
		Status        string `json:"status"`
		DirectoryName string `json:"directory_name"`
		FilePath      string `json:"file_path"`
		Iterations    int    `json:"iterations"`
		Message       string `json:"message"`
	}

	// 尝试找到 JSON 开始和结束位置
	startIdx := -1
	endIdx := -1

	// 查找第一个 { 和最后一个 }
	for i, ch := range result {
		if ch == '{' && startIdx == -1 {
			startIdx = i
		}
		if ch == '}' {
			endIdx = i
		}
	}

	if startIdx == -1 || endIdx == -1 || startIdx >= endIdx {
		// 如果找不到 JSON 边界，尝试直接解析整个结果
		if err := json.Unmarshal([]byte(result), &pageResult); err != nil {
			return nil, fmt.Errorf("failed to parse page generate result: %w", err)
		}
	} else {
		// 提取 JSON 部分
		jsonStr := result[startIdx : endIdx+1]
		if err := json.Unmarshal([]byte(jsonStr), &pageResult); err != nil {
			return nil, fmt.Errorf("failed to parse page generate JSON: %w", err)
		}
	}

	// 验证必需字段
	if pageResult.Status == "" {
		return nil, fmt.Errorf("status field is empty")
	}

	// 构建返回结果
	genResult := &PageGenerateResult{
		Status:        pageResult.Status,
		DirectoryName: pageResult.DirectoryName,
		FilePath:      pageResult.FilePath,
		Iterations:    pageResult.Iterations,
		Message:       pageResult.Message,
	}

	return genResult, nil
}

// GeneratePageDirectoryName 生成页面资源目录名
func GeneratePageDirectoryName() string {
	timestamp := time.Now().Format("20060102_150405")
	return fmt.Sprintf("page_generate_%s", timestamp)
}

// ExtractPageInfo 从XML提取页面信息（辅助函数）
func ExtractPageInfo(xmlData string) (pageType, pageTitle, coreContent string) {
	// 简单的XML解析，提取关键信息
	// 实际使用时可能需要更完善的XML解析

	// 提取page_type
	if start := strings.Index(xmlData, "<page_type>"); start != -1 {
		end := strings.Index(xmlData[start:], "</page_type>")
		if end != -1 {
			pageType = xmlData[start+11 : start+end]
		}
	}

	// 提取page_title
	if start := strings.Index(xmlData, "<page_title>"); start != -1 {
		end := strings.Index(xmlData[start:], "</page_title>")
		if end != -1 {
			pageTitle = xmlData[start+12 : start+end]
		}
	}

	// 提取core_content
	if start := strings.Index(xmlData, "<core_content>"); start != -1 {
		end := strings.Index(xmlData[start:], "</core_content>")
		if end != -1 {
			coreContent = xmlData[start+14 : start+end]
		}
	}

	return
}

// GetTemplateFileByPageType 根据页面类型获取模板文件名
func GetTemplateFileByPageType(pageType string) string {
	switch pageType {
	case "封面页":
		return "cover.html"
	case "目录页":
		return "toc.html"
	case "内容页":
		return "content.html"
	case "数据页":
		return "data.html"
	case "结尾页":
		return "ending.html"
	default:
		// 默认使用内容页模板
		return "content.html"
	}
}

// GetPageFromPlan 从PPT计划中获取指定页面的数据
// 注意：这个函数在实际使用时，需要先通过ViewTool读取XML文件内容，
// 然后调用ParsePPTPlan解析，最后调用此函数获取指定页面
func GetPageFromPlan(xmlContent string, pageNumber int) (*Page, error) {
	// 解析XML
	plan, err := ParsePPTPlan(xmlContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PPT plan: %w", err)
	}

	if pageNumber <= 0 || pageNumber > len(plan.PageStructure.Pages) {
		return nil, fmt.Errorf("invalid page number %d, total pages: %d",
			pageNumber, len(plan.PageStructure.Pages))
	}

	// 页码从1开始，数组索引从0开始
	return &plan.PageStructure.Pages[pageNumber-1], nil
}

// ValidateGeneratedHTML 验证生成的HTML（辅助函数）
func ValidateGeneratedHTML(htmlContent string) error {
	// 基本的HTML验证
	if !strings.Contains(htmlContent, "<!DOCTYPE html>") {
		return fmt.Errorf("缺少DOCTYPE声明")
	}
	if !strings.Contains(htmlContent, "1280") || !strings.Contains(htmlContent, "720") {
		return fmt.Errorf("未设置正确的宽高尺寸")
	}
	if !strings.Contains(htmlContent, "slide-container") && !strings.Contains(htmlContent, "overflow: hidden") {
		return fmt.Errorf("缺少必要的容器或溢出控制")
	}
	return nil
}

// GenerateIterationFileName 生成迭代文件名
func GenerateIterationFileName(iteration int) string {
	return fmt.Sprintf("iterations/attempt_%d.html", iteration)
}

