package agents

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"gentica/agent"
	"gentica/tools"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// PPTPlan 表示整个PPT计划
type PPTPlan struct {
	XMLName       xml.Name      `xml:"ppt_plan"`
	ProjectInfo   ProjectInfo   `xml:"project_info"`
	PageStructure PageStructure `xml:"page_structure"`
}

// ProjectInfo 表示项目基本信息
type ProjectInfo struct {
	XMLName             xml.Name `xml:"project_info"`
	Title               string   `xml:"title"`
	TotalPages          int      `xml:"total_pages"`
	TargetAudience      string   `xml:"target_audience"`
	DesignStyle         string   `xml:"design_style"`
	UsageScenario       string   `xml:"usage_scenario"`
	PresentationPurpose string   `xml:"presentation_purpose"`
}

// PageStructure 表示页面结构
type PageStructure struct {
	XMLName xml.Name `xml:"page_structure"`
	Pages   []Page   `xml:"page"`
}

// Page 表示单个页面
type Page struct {
	XMLName     xml.Name `xml:"page"`
	PageNumber  int      `xml:"page_number"`
	PageTitle   string   `xml:"page_title"`
	PageType    string   `xml:"page_type"`
	CoreContent string   `xml:"core_content"`
}

// OutlinePlanResult PPT大纲生成结果
type OutlinePlanResult struct {
	FilePath      string `json:"file_path"`      // 保存的文件路径
	DirectoryName string `json:"directory_name"` // 资源目录名称
	Status        string `json:"status"`         // 任务状态：success 或 failed
}

// OutlinePlanAgentDependencies 大纲生成器的依赖
type OutlinePlanAgentDependencies struct {
	DirectoryAddTool  ai.Tool // 资源目录创建工具
	DirectoryListTool ai.Tool // 资源目录列表工具
	WriteTool         ai.Tool // 文件写入工具
	ViewTool          ai.Tool // 文件查看工具（可选，用于验证）
	LsTool            ai.Tool // 目录列表工具
}

// NewOutlinePlanAgent 创建大纲生成 Agent（使用默认依赖）
func NewOutlinePlanAgent(g *genkit.Genkit, workingDir string) agent.Agent {
	// 创建默认工具
	deps := &OutlinePlanAgentDependencies{
		DirectoryAddTool:  tools.AdaptBaseToolToGenkit(g, tools.NewResourceDirectoryAddTool(workingDir)),
		DirectoryListTool: tools.AdaptBaseToolToGenkit(g, tools.NewResourceDirectoryListTool(workingDir)),
		WriteTool:         tools.AdaptBaseToolToGenkit(g, tools.NewWriteTool(workingDir)),
		ViewTool:          tools.AdaptBaseToolToGenkit(g, tools.NewViewTool(workingDir)),
		LsTool:            tools.AdaptBaseToolToGenkit(g, tools.NewLsTool(workingDir)),
	}
	return NewOutlinePlanAgentWithDeps(g, deps)
}

// NewOutlinePlanAgentWithDeps 创建带依赖注入的大纲生成 Agent
func NewOutlinePlanAgentWithDeps(g *genkit.Genkit, deps *OutlinePlanAgentDependencies) agent.Agent {
	// 系统提示 - 保留原始的完整提示
	systemPrompt := `你是一个资深的PPT制作专家和内容策略师。请根据用户提供的资料，生成完整的PPT制作大纲。

## 分析与思考流程
1. **内容解析**：明确主题、核心信息点、风格、受众、演示场景和目的，并识别行业专业程度及潜台词。
   - 如果提供了research_directory，使用ls工具列出资料目录中的文件
   - 使用view工具阅读相关的研究资料文件（.md格式）
   - 从资料中提取关键信息和观点，作为大纲内容的基础
2. **信息补充**：当数据不足以支撑完整大纲时，基于常识和合理推断补充信息，但需确保逻辑自洽。
   - 优先使用研究资料中的内容
   - 资料不足时再进行合理推断
3. **内容澄清**：若出现多义/模糊概念，可利用信息推断或网络常识进行消歧。
4. **主题策略**：优化演示主题，制定分章节的演示逻辑，提炼核心信息。
5. **结构规划与分页策略**：
    - 目录页列出**章节名**（一级主题），不与页面一一匹配。
    - 每个章节根据内容密度、受众吸收能力和演示流畅度自动分页：
        - **一页规则**：章节总内容少于3个核心要点时 → 直接"一页一章"。
        - **多页规则**：章节总内容 ≥3个核心要点时 → 划分为多页，每页专注一个子主题，标题为"章节名：子主题名"。
    - 页面顺序必须严格按照目录顺序展开。
6. **页数分配**：综合控制总页数使其在8～15页之间（包含封面页、目录页、结尾页），除封面页和结尾页以外，全部为目录页或内容页。
7. **页面内容**：封面页、结尾页不包含正式内容，仅做开场白和致谢。

## 工作流程
1. 首先，你需要生成符合规范的XML格式大纲
2. 然后，使用 resourceDirectoryAdd 工具创建资源目录来存储大纲
3. 使用 write 工具将生成的XML保存到文件
4. 返回包含文件路径和状态的JSON结果

## 输出格式要求
- 生成标准 XML 格式的大纲
- 标签结构如下：

<ppt_plan>
    <project_info>
        <title>[PPT主题]</title>
        <total_pages>[X]</total_pages>
        <target_audience>[受众描述]</target_audience>
        <design_style>[风格要求]</design_style>
        <usage_scenario>[演示场景]</usage_scenario>
        <presentation_purpose>[演示目的]</presentation_purpose>
    </project_info>
    <page_structure>
        <page>
            <page_number>1</page_number>
            <page_title>[页面核心标题]</page_title>
            <page_type>[封面页/目录页/内容页/结尾页]</page_type>
            <core_content>[核心内容要点；目录页时为章节列表]</core_content>
        </page>
        ...
    </page_structure>
</ppt_plan>

## 输出示例
<ppt_plan>
    <project_info>
        <title>AI赋能教育：开启智慧教学新时代</title>
        <total_pages>9</total_pages>
        <target_audience>教育工作者和学校管理者</target_audience>
        <design_style>科技现代 + 教育亲和</design_style>
        <usage_scenario>说服性演示</usage_scenario>
        <presentation_purpose>推动AI技术在教育领域的采用</presentation_purpose>
    </project_info>
    <page_structure>
        <page>
            <page_number>1</page_number>
            <page_title>AI赋能教育：开启智慧教学新时代</page_title>
            <page_type>封面页</page_type>
            <core_content>主标题、副标题、演讲者信息</core_content>
        </page>
        <page>
            <page_number>2</page_number>
            <page_title>议程概览</page_title>
            <page_type>目录页</page_type>
            <core_content>
                1. 传统教育的挑战
                2. AI技术在教育的应用
                3. 智慧课堂案例
                4. 推进策略与落地方案
            </core_content>
        </page>
        <page>
            <page_number>3</page_number>
            <page_title>传统教育的挑战</page_title>
            <page_type>内容页</page_type>
            <core_content>主要挑战包括一刀切教学、评估滞后、教育资源分配不均等</core_content>
        </page>
        <page>
            <page_number>4</page_number>
            <page_title>AI技术在教育的应用：个性化学习路径</page_title>
            <page_type>内容页</page_type>
            <core_content>利用AI为不同学生定制学习计划与进度以提升学习效果</core_content>
        </page>
        <page>
            <page_number>5</page_number>
            <page_title>AI技术在教育的应用：自动化评估与反馈</page_title>
            <page_type>内容页</page_type>
            <core_content>通过AI批改作业并提供个性化反馈</core_content>
        </page>
        <page>
            <page_number>6</page_number>
            <page_title>智慧课堂案例</page_title>
            <page_type>内容页</page_type>
            <core_content>某学校引入智慧课堂后的数据与学习效果提升结果</core_content>
        </page>
        <page>
            <page_number>7</page_number>
            <page_title>推进策略与落地方案：实施步骤</page_title>
            <page_type>内容页</page_type>
            <core_content>分阶段实施计划，从试点到全面推广</core_content>
        </page>
        <page>
            <page_number>8</page_number>
            <page_title>推进策略与落地方案：教师培训与评估机制</page_title>
            <page_type>内容页</page_type>
            <core_content>针对教师的培训计划与科学评估体系</core_content>
        </page>
        <page>
            <page_number>9</page_number>
            <page_title>致谢</page_title>
            <page_type>结尾页</page_type>
            <core_content></core_content>
        </page>
    </page_structure>
</ppt_plan>

## 文件保存要求
1. 创建资源目录，名称格式：ppt_outline_[timestamp]
2. 在目录中保存生成的XML文件，文件名：outline.xml
3. 返回JSON格式的结果，包含文件路径和状态

## 最终返回格式
你必须在完成所有工作后，返回以下JSON格式的结果：
{
  "status": "success" 或 "failed",
  "directory_name": "资源目录名称",
  "file_path": "XML文件的完整路径",
  "summary": "生成的大纲摘要"
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
		if deps.LsTool != nil {
			toolList = append(toolList, deps.LsTool)
		}
	}

	// 创建并返回 Agent
	return agent.NewBuilder(
		g,
		"outline_plan_agent",
		"生成PPT制作大纲",
		systemPrompt,
	).WithInputSchema(
		map[string]any{
			"topic": map[string]any{
				"type":        "string",
				"description": "PPT主题或详细资料",
			},
			"research_directory": map[string]any{
				"type":        "string",
				"description": "研究资料目录路径（可选）",
			},
		},
		"topic", // 必需字段
	).WithTools(toolList...).
		WithModel("openai/gpt-5-mini"). // 可以根据需要改为 claude 模型
		WithTemperature(0.8).           // 保持创造性
		WithMaxRounds(16).              // 处理工具调用
		WithLogging(true).
		Build()
}

// ParsePPTPlan 解析PPT计划XML
func ParsePPTPlan(xmlData string) (*PPTPlan, error) {
	if xmlData == "" {
		return nil, fmt.Errorf("XML data is empty")
	}

	var plan PPTPlan

	// 尝试找到 XML 开始和结束位置
	startIdx := strings.Index(xmlData, "<ppt_plan>")
	endIdx := strings.LastIndex(xmlData, "</ppt_plan>")

	if startIdx == -1 || endIdx == -1 || startIdx >= endIdx {
		// 如果找不到标签，尝试直接解析
		decoder := xml.NewDecoder(strings.NewReader(xmlData))
		if err := decoder.Decode(&plan); err != nil {
			return nil, fmt.Errorf("failed to parse XML: %w", err)
		}
	} else {
		// 提取 XML 部分
		xmlContent := xmlData[startIdx : endIdx+len("</ppt_plan>")]
		decoder := xml.NewDecoder(strings.NewReader(xmlContent))
		if err := decoder.Decode(&plan); err != nil {
			return nil, fmt.Errorf("failed to parse extracted XML: %w", err)
		}
	}

	// 验证必需字段
	if plan.ProjectInfo.Title == "" {
		return nil, fmt.Errorf("title field is empty")
	}
	if plan.ProjectInfo.TotalPages <= 0 {
		return nil, fmt.Errorf("total_pages must be positive, got %d", plan.ProjectInfo.TotalPages)
	}
	if len(plan.PageStructure.Pages) == 0 {
		return nil, fmt.Errorf("no pages defined in structure")
	}

	// 验证页面数量是否匹配
	if len(plan.PageStructure.Pages) != plan.ProjectInfo.TotalPages {
		return nil, fmt.Errorf("page count mismatch: total_pages=%d, actual pages=%d",
			plan.ProjectInfo.TotalPages, len(plan.PageStructure.Pages))
	}

	return &plan, nil
}

// ParseOutlineResult 解析Agent返回的JSON结果
func ParseOutlineResult(result string) (*OutlinePlanResult, error) {
	if result == "" {
		return nil, fmt.Errorf("result is empty")
	}

	var outlineResult struct {
		Status        string `json:"status"`
		DirectoryName string `json:"directory_name"`
		FilePath      string `json:"file_path"`
		Summary       string `json:"summary"`
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
		if err := json.Unmarshal([]byte(result), &outlineResult); err != nil {
			return nil, fmt.Errorf("failed to parse outline result: %w", err)
		}
	} else {
		// 提取 JSON 部分
		jsonStr := result[startIdx : endIdx+1]
		if err := json.Unmarshal([]byte(jsonStr), &outlineResult); err != nil {
			return nil, fmt.Errorf("failed to parse outline JSON: %w", err)
		}
	}

	// 验证必需字段
	if outlineResult.Status == "" {
		return nil, fmt.Errorf("status field is empty")
	}

	// 验证状态值
	if outlineResult.Status != "success" && outlineResult.Status != "failed" {
		return nil, fmt.Errorf("invalid status value: %s", outlineResult.Status)
	}

	// 成功时验证文件路径
	if outlineResult.Status == "success" && outlineResult.FilePath == "" {
		return nil, fmt.Errorf("file_path is required when status is success")
	}

	// 构建返回结果
	planResult := &OutlinePlanResult{
		Status:        outlineResult.Status,
		DirectoryName: outlineResult.DirectoryName,
		FilePath:      outlineResult.FilePath,
	}

	return planResult, nil
}

// GenerateOutlineFileName 生成大纲文件名
func GenerateOutlineFileName() string {
	timestamp := time.Now().Format("20060102_150405")
	return fmt.Sprintf("outline_%s.xml", timestamp)
}

// GenerateDirectoryName 生成资源目录名
func GenerateDirectoryName() string {
	timestamp := time.Now().Format("20060102_150405")
	return fmt.Sprintf("ppt_outline_%s", timestamp)
}
