package agents

import (
	"encoding/json"
	"fmt"

	"gentica/agent"
	"gentica/tools"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// ResearchCollectorResult 研究资料收集结果结构
type ResearchCollectorResult struct {
	Summary       string `json:"summary"`        // 工作汇报和成果总结
	DirectoryName string `json:"directory_name"` // 资源目录名称
	Status        string `json:"status"`         // 任务状态：success 或 failed
}

// ResearchCollectorDependencies 研究收集器的完整依赖
type ResearchCollectorDependencies struct {
	// 工具依赖（已转换为 ai.Tool）
	ViewTool            ai.Tool
	WriteTool           ai.Tool
	LsTool              ai.Tool
	DirectoryListTool   ai.Tool
	DirectoryAddTool    ai.Tool
	DirectoryRemoveTool ai.Tool
	SearchCrawlerTool   ai.Tool

	// Agent 依赖（已转换为 ai.Tool）
	SearchNeedsAnalyzer ai.Tool
	ArticleEvaluator    ai.Tool
}

// NewResearchCollector 创建研究资料收集 Agent（使用默认依赖）
func NewResearchCollector(g *genkit.Genkit, workingDir string) agent.Agent {
	// 创建所有需要的工具
	deps := &ResearchCollectorDependencies{
		ViewTool:            tools.AdaptBaseToolToGenkit(g, tools.NewViewTool(workingDir)),
		WriteTool:           tools.AdaptBaseToolToGenkit(g, tools.NewWriteTool(workingDir)),
		LsTool:              tools.AdaptBaseToolToGenkit(g, tools.NewLsTool(workingDir)),
		DirectoryListTool:   tools.AdaptBaseToolToGenkit(g, tools.NewResourceDirectoryListTool(workingDir)),
		DirectoryAddTool:    tools.AdaptBaseToolToGenkit(g, tools.NewResourceDirectoryAddTool(workingDir)),
		DirectoryRemoveTool: tools.AdaptBaseToolToGenkit(g, tools.NewResourceDirectoryRemoveTool(workingDir)),
		SearchCrawlerTool:   tools.AdaptBaseToolToGenkit(g, tools.NewSearchCrawlerTool(workingDir)),
		SearchNeedsAnalyzer: tools.AdaptBaseToolToGenkit(g, agent.AsToolAdapter(NewSearchNeedsAnalyzer(g))),
		ArticleEvaluator:    tools.AdaptBaseToolToGenkit(g, agent.AsToolAdapter(NewArticleEvaluator(g, workingDir))),
	}
	return NewResearchCollectorWithDeps(g, deps)
}

// NewResearchCollectorWithDeps 创建带依赖注入的研究资料收集 Agent
func NewResearchCollectorWithDeps(g *genkit.Genkit, deps *ResearchCollectorDependencies) agent.Agent {

	// 系统提示，定义完整的工作流程
	systemPrompt := `你是一个专业的研究资料收集专家。你的任务是为给定的研究主题收集高质量的资料。

## 你的任务流程

### 第一步：分析搜索需求
1. 使用 search_needs_analyzer 工具分析研究主题是否需要搜索
2. 输入格式：{"research_topic": "研究主题内容"}
3. 如果返回 need_search: false，立即返回 JSON 结果说明原因并退出

### 第二步：生成搜索关键词（需要搜索时）
1. 为研究主题生成 2-4 个搜索关键词
2. 关键词应该：
   - 覆盖主题的不同方面
   - 包含核心概念和相关术语
   - 有一定的差异性，避免重复
3. 记录生成的关键词列表

### 第三步：创建资源目录
1. 使用 resourceDirectoryAdd 工具创建新的资源目录
2. 目录名称格式：research_[主题关键词]_[时间戳后4位]
3. 记录创建的目录名称

### 第四步：搜索和爬取
1. 对每个关键词使用 searchCrawler 工具
2. 参数设置：
   - query: 搜索关键词
   - result_num: 4（每个关键词获取4个结果）
   - save_path: 创建的资源目录路径
3. 记录每次搜索的结果统计

### 第五步：评估搜索结果
1. 使用 ls 工具查看资源目录中的文件
2. 对每个文件：
   - 使用 article_evaluator 评估质量
   - 输入格式：{"file_path": "文件路径", "research_topic": "研究主题"}
   - 记录评分结果

### 第六步：筛选高质量资料
1. 根据评分筛选资料：
   - 保留总分 >= 60 的高质量资料
   - 原则上保留约50%的结果
   - 如果高质量资料过少（少于总数的30%），考虑降低标准到 >= 50
2. 使用 resourceDirectoryRemove 删除低质量资料
3. 统计最终保留的资料数量

### 第七步：质量检查和补充（如需要）
1. 如果保留的高质量资料过少（少于4篇）：
   - 扩展搜索关键词（添加相关术语、同义词）
   - 对重要但资料不足的关键词增加 result_num 到 6-8
   - 重复第四步到第六步
2. 如果资料充足但某个重要方面缺失：
   - 生成针对性的补充关键词
   - 进行定向搜索和评估

### 第八步：整理和返回结果
1. 使用 resourceDirectoryList 查看最终的资源目录状态
2. 生成工作总结，包括：
   - 搜索了哪些关键词
   - 获取了多少初始结果
   - 筛选后保留了多少高质量资料
   - 资料覆盖了研究主题的哪些方面
3. 返回严格的 JSON 格式结果

## 输出格式
你必须严格按照以下 JSON 格式返回结果：
{
  "summary": "详细的工作汇报，包括搜索过程、筛选标准、最终成果等",
  "directory_name": "资源目录的名称",
  "status": "success" 或 "failed"
}

## 重要原则
1. **质量优先**：宁缺毋滥，只保留真正有价值的资料
2. **覆盖全面**：确保资料覆盖研究主题的主要方面
3. **效率平衡**：在质量和数量之间找到平衡点
4. **透明汇报**：清楚说明筛选标准和决策理由
5. **灵活调整**：根据实际情况调整搜索策略

## 特殊情况处理
1. 如果研究主题不需要搜索：
   - 返回 status: "success"
   - summary 中说明不需要搜索的原因
   - directory_name 为空字符串
2. 如果所有搜索都失败：
   - 返回 status: "failed"
   - summary 中说明失败原因
3. 如果找不到任何高质量资料：
   - 尝试调整搜索策略
   - 如果仍然失败，返回 status: "failed" 并说明原因

请记住：你的目标是为研究主题收集最相关、最高质量的资料，并组织在一个清晰的目录结构中。`

	// 创建并返回 Agent
	return agent.NewBuilder(
		g,
		"research_collector",
		"为研究主题收集和筛选高质量资料",
		systemPrompt,
	).WithInputSchema(
		map[string]any{
			"research_topic": map[string]any{
				"type":        "string",
				"description": "研究主题，可以是短语或较长的纲领",
			},
		},
		"research_topic", // 必需字段
	).WithTools(
		deps.SearchNeedsAnalyzer,
		deps.ArticleEvaluator,
		deps.SearchCrawlerTool,
		deps.DirectoryListTool,
		deps.DirectoryAddTool,
		deps.DirectoryRemoveTool,
		deps.ViewTool,
		deps.WriteTool,
		deps.LsTool,
	).WithModel("openai/gpt-5-mini").
		WithTemperature(0.3). // 低温度以确保稳定和一致的行为
		WithMaxTokens(4000).
		WithMaxRounds(10). // 需要多轮工具调用来完成整个流程
		WithLogging(true).
		Build()
}

// ParseResearchCollectorResult 解析研究收集结果
func ParseResearchCollectorResult(result string) (*ResearchCollectorResult, error) {
	var collectorResult ResearchCollectorResult

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
		if err := json.Unmarshal([]byte(result), &collectorResult); err != nil {
			return nil, fmt.Errorf("failed to parse research collector result: %w", err)
		}
	} else {
		// 提取 JSON 部分
		jsonStr := result[startIdx : endIdx+1]
		if err := json.Unmarshal([]byte(jsonStr), &collectorResult); err != nil {
			return nil, fmt.Errorf("failed to parse research collector JSON: %w", err)
		}
	}

	// 验证必需字段
	if collectorResult.Status == "" {
		return nil, fmt.Errorf("status field is empty")
	}
	if collectorResult.Summary == "" {
		return nil, fmt.Errorf("summary field is empty")
	}

	return &collectorResult, nil
}