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
	XMLData   string   `json:"xml_data"`  // PPT单页的XML数据
	Templates []string `json:"templates"` // 模板代码列表
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
	BashTool          ai.Tool // Bash命令工具
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
		BashTool:          tools.AdaptBaseToolToGenkit(g, tools.NewBashTool(workingDir)),
	}
	return NewPageGenerateAgentWithDeps(g, deps)
}

// NewPageGenerateAgentWithDeps 创建带依赖注入的页面生成 Agent
func NewPageGenerateAgentWithDeps(g *genkit.Genkit, deps *PageGenerateAgentDependencies) agent.Agent {
	// 系统提示
	systemPrompt := `你是专业的PPT页面生成专家。根据大纲和模板生成规范的HTML页面。

## 核心任务
1. 读取大纲和模板，生成指定页面的HTML
2. **重要**：如果提供了research_directory，必须充分利用搜索资料，在页面中呈现详实内容
3. **关键**：必须将最终结果保存为final.html文件

## 内容处理
### 基础内容
- 从XML提取：page_title、core_content、page_type

### 搜索资料使用（重点）
如果有research_directory：
- 使用ls列出所有资料
- 使用view详细阅读多个与页面主题高度相关的资料
- **有意识地将关键数据、事实、统计、案例等高密度内容深度融入页面，提升内容质量**

## HTML要求
- 必须包含.slide-container元素
- 尺寸1280x720px
- body设置overflow:hidden
- 使用px固定尺寸，避免响应式单位

## 检查清单
- .slide-container 元素必须存在
- 容器尺寸1280x720px（允许±10%容错）
- overflow:hidden 防止溢出
- **检查是否充分使用搜索资料**

## 工作流程

1. 读取大纲文件（view工具）并解析指定页码数据
2. **搜索资料处理**（如果有research_directory）：
   - ls列出所有资料文件
   - 查找并读取**多个**相关资料
   - 提取并整合高价值内容
3. 读取对应模板（根据page_type）
4. 创建资源目录：page_generate_[timestamp]_p[页码]
5. 生成HTML（融入搜索资料内容）
6. 保存为iterations/attempt_N.html
7. html_size验证（允许±10%容错）
8. 如需调整则迭代优化
9. **必须执行**：使用bash工具复制最终版本为final.html
   命令：cp iterations/attempt_N.html final.html && pwd
10. 返回结果（file_path必须是final.html的绝对路径，使用pwd获取）

## 返回格式
返回JSON：
{
  "status": "success" 或 "failed",
  "directory_name": "资源目录名称",
  "file_path": "必须是final.html的完整路径",
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
		if deps.BashTool != nil {
			toolList = append(toolList, deps.BashTool)
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
		WithTemperature(0.5). // 代码生成需要更确定性的输出
		WithMaxTokens(10000). // 代码生成需要较多token
		WithMaxRounds(16). // 支持多轮迭代
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
