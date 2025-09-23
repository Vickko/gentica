package agents

import (
	"encoding/json"

	"gentica/agent"
	"gentica/tools"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// PPTTopResult PPT生成最终结果
type PPTTopResult struct {
	Status              string            `json:"status"`                // 任务状态：success 或 failed
	ResearchDirectory   string            `json:"research_directory"`    // 研究资料目录
	OutlineDirectory    string            `json:"outline_directory"`     // 大纲资源目录
	OutlineFilePath     string            `json:"outline_file_path"`     // 大纲文件路径
	TemplateDirectory   string            `json:"template_directory"`    // 模板资源目录
	TemplateFilePaths   map[string]string `json:"template_file_paths"`   // 模板文件路径映射
	GeneratedPages      []string          `json:"generated_pages"`       // 生成的页面文件列表
	Summary             string            `json:"summary"`               // 生成摘要
}

// PPTTopAgentDependencies PPT顶层协调器的依赖
type PPTTopAgentDependencies struct {
	ResearchCollectorTool ai.Tool // 研究资料收集工具（已适配的 Agent）
	OutlinePlanTool       ai.Tool // 大纲生成工具（已适配的 Agent）
	TemplateDesignTool    ai.Tool // 模板设计工具（已适配的 Agent）
	PageGenerateTool      ai.Tool // 页面生成工具（已适配的 Agent）
	ViewTool              ai.Tool // 文件查看工具
	LsTool                ai.Tool // 目录列表工具
	DirectoryListTool     ai.Tool // 资源目录列表工具
}

// NewPPTTopAgent 创建PPT顶层协调 Agent（使用默认依赖）
func NewPPTTopAgent(g *genkit.Genkit, workingDir string) agent.Agent {
	// 创建子 agents
	researchCollector := NewResearchCollector(g, workingDir)
	outlineAgent := NewOutlinePlanAgent(g, workingDir)
	templateAgent := NewTemplateDesignAgent(g, workingDir)
	pageGenAgent := NewPageGenerateAgent(g, workingDir)

	// 创建依赖 - 将 agents 转换为工具
	deps := &PPTTopAgentDependencies{
		ResearchCollectorTool: tools.AdaptBaseToolToGenkit(g, agent.AsToolAdapter(researchCollector)),
		OutlinePlanTool:       tools.AdaptBaseToolToGenkit(g, agent.AsToolAdapter(outlineAgent)),
		TemplateDesignTool:    tools.AdaptBaseToolToGenkit(g, agent.AsToolAdapter(templateAgent)),
		PageGenerateTool:      tools.AdaptBaseToolToGenkit(g, agent.AsToolAdapter(pageGenAgent)),
		ViewTool:              tools.AdaptBaseToolToGenkit(g, tools.NewViewTool(workingDir)),
		LsTool:                tools.AdaptBaseToolToGenkit(g, tools.NewLsTool(workingDir)),
		DirectoryListTool:     tools.AdaptBaseToolToGenkit(g, tools.NewResourceDirectoryListTool(workingDir)),
	}

	return NewPPTTopAgentWithDeps(g, deps)
}

// NewPPTTopAgentWithDeps 创建带依赖注入的PPT顶层协调 Agent
func NewPPTTopAgentWithDeps(g *genkit.Genkit, deps *PPTTopAgentDependencies) agent.Agent {
	// 系统提示 - 调整为使用文件系统工具
	systemPrompt := `你是一个PPT制作专家。当用户要求制作PPT时，请按以下流程操作：

## 重要规则
每次回应都必须包含文字内容，不能只调用工具！

## 工作流程
### 步骤1：收集研究资料
- 调用 research_collector
- 输入：{"research_topic": "用户的PPT需求主题"}
- 获得研究资料目录名称（directory_name）
- 如果收集失败或不需要资料，跳过并记录原因

### 步骤2：获取研究资料完整路径（如果步骤1成功）
- 调用 resource_directory_list 获取所有资源目录
- 从列表中找到名称匹配的目录
- 获取该目录的完整路径（path字段）

### 步骤3：生成PPT大纲
- 调用 outline_plan_agent
- 输入：{"topic": "用户的PPT需求", "research_directory": "资料目录完整路径(如有)"}
- 获得大纲文件路径和目录

### 步骤4：提取风格信息
- 使用 view 工具读取大纲XML文件
- 解析 <design_style> 标签内容获取风格描述

### 步骤5：设计PPT模板
- 调用 template_design_agent
- 输入：{"style_description": "从大纲提取的风格描述"}
- 获得模板文件路径映射

### 步骤6：批量生成页面
- 读取大纲获取总页数
- 循环调用 page_generate_agent 生成每一页
- 输入：{
    "outline_path": "大纲文件路径",
    "template_dir": "模板目录路径",
    "page_number": N,
    "research_directory": "资料目录完整路径(如有)"
  }
- 收集所有生成的页面路径

## 执行流程示例
用户："帮我做一个软件测试PPT"

第1轮：
文字："好的，我来为您制作软件测试PPT。首先收集相关研究资料。"
工具：[调用 research_collector，参数: {"research_topic": "软件测试"}]

第2轮：
文字："资料收集完成，保存在目录 [目录名]。让我获取资料的完整路径。"
工具：[调用 resource_directory_list]

第3轮：
文字："找到资料目录路径：[完整路径]。现在基于这些资料生成PPT大纲。"
工具：[调用 outline_plan_agent，参数: {"topic": "软件测试PPT", "research_directory": "完整路径"}]

第4轮：
文字："大纲生成完成。让我查看大纲内容并提取风格信息。"
工具：[调用 view，参数: {"file_path": "大纲文件路径"}]

第5轮：
文字："大纲中设定的风格是[风格描述]。现在基于此风格设计模板。"
工具：[调用 template_design_agent，参数: {"style_description": "提取的风格"}]

第6轮：
文字："模板设计完成。让我查看模板目录结构。"
工具：[调用 ls，参数: {"path": "模板目录路径"}]

第7轮：
文字："模板包含了5个页面类型。大纲共有[N]页，现在开始生成第1页。"
工具：[调用 page_generate_agent，参数: {"outline_path": "大纲路径", "template_dir": "模板目录", "page_number": 1, "research_directory": "完整路径"}]

第8轮：
文字："第1页生成完成。继续生成第2页。"
工具：[调用 page_generate_agent，参数: {"outline_path": "大纲路径", "template_dir": "模板目录", "page_number": 2, "research_directory": "完整路径"}]

[继续生成剩余页面...]

最后：
文字："PPT制作完成！共生成[N]页。您可以在以下目录找到生成的文件：[总结各目录位置]"

## 重要提示
- 每个阶段都要用文字说明当前进度和下一步计划
- 研究资料收集后，必须获取完整路径才能传递给后续agents
- 确保数据在各个agent间正确传递
- 最后汇总所有生成的资源位置

## 最终返回格式
完成所有工作后，返回JSON格式的汇总结果：
{
  "status": "success",
  "research_directory": "研究资料目录完整路径",
  "outline_directory": "大纲资源目录",
  "outline_file_path": "大纲文件路径",
  "template_directory": "模板资源目录",
  "template_file_paths": {...},
  "generated_pages": [...],
  "summary": "生成过程总结"
}`

	// 准备工具列表
	toolList := []ai.Tool{}

	// 添加子 agents 作为工具（已经是 ai.Tool 类型）
	if deps.ResearchCollectorTool != nil {
		toolList = append(toolList, deps.ResearchCollectorTool)
	}
	if deps.OutlinePlanTool != nil {
		toolList = append(toolList, deps.OutlinePlanTool)
	}
	if deps.TemplateDesignTool != nil {
		toolList = append(toolList, deps.TemplateDesignTool)
	}
	if deps.PageGenerateTool != nil {
		toolList = append(toolList, deps.PageGenerateTool)
	}

	// 添加文件系统工具
	if deps.ViewTool != nil {
		toolList = append(toolList, deps.ViewTool)
	}
	if deps.LsTool != nil {
		toolList = append(toolList, deps.LsTool)
	}
	if deps.DirectoryListTool != nil {
		toolList = append(toolList, deps.DirectoryListTool)
	}

	// 创建并返回 Agent
	return agent.NewBuilder(
		g,
		"ppt_top_agent",
		"PPT制作顶层协调器",
		systemPrompt,
	).WithInputSchema(
		map[string]any{
			"request": map[string]any{
				"type":        "string",
				"description": "用户的PPT制作需求",
			},
		},
		"request", // 必需字段
	).WithTools(toolList...).
		WithModel("openai/gpt-4o-mini"). // 可以根据需要改为 claude 模型
		WithTemperature(0.3).            // 降低随机性，确保遵循指令
		WithMaxTokens(6000).             // 增加 max_tokens
		WithMaxRounds(20).               // 支持多轮工具调用
		WithLogging(true).
		Build()
}

// ParsePPTTopResult 解析Agent返回的JSON结果
func ParsePPTTopResult(result string) (*PPTTopResult, error) {
	var topResult PPTTopResult

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
		if err := json.Unmarshal([]byte(result), &topResult); err != nil {
			// 如果解析失败，返回默认结果
			return &PPTTopResult{
				Status:  "completed",
				Summary: result,
			}, nil
		}
	} else {
		// 提取 JSON 部分
		jsonStr := result[startIdx : endIdx+1]
		if err := json.Unmarshal([]byte(jsonStr), &topResult); err != nil {
			// 如果解析失败，返回默认结果
			return &PPTTopResult{
				Status:  "completed",
				Summary: result,
			}, nil
		}
	}

	return &topResult, nil
}