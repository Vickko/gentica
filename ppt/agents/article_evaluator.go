package agents

import (
	"encoding/json"
	"fmt"

	"gentica/agent"
	"gentica/tools"

	"github.com/firebase/genkit/go/genkit"
)

// ArticleEvaluation 评分结果结构
type ArticleEvaluation struct {
	Keyword  string       `json:"keyword"`
	Result   ArticleScore `json:"result"`
	Status   string       `json:"status"`
}

// ArticleScore 单篇文章的评分
type ArticleScore struct {
	Title      string `json:"title"`
	Score      int    `json:"score"`
	Relevance  int    `json:"relevance"`
	Quality    int    `json:"quality"`
	Timeliness int    `json:"timeliness"`
	Comments   string `json:"comments"`
}

// NewArticleEvaluator 创建文章评估 Agent
func NewArticleEvaluator(g *genkit.Genkit, workingDir string) agent.Agent {
	// 准备文件读取工具
	viewTool := tools.AdaptBaseToolToGenkit(g, tools.NewViewTool(workingDir))

	// 系统提示，包含详细的评分规则
	systemPrompt := `你是一个专业的文章质量评估专家。你需要根据给定的研究主题，对文章内容进行严格的三维度评分。

## 你的任务
1. 读取指定路径的文件内容
2. 根据研究主题对文章进行评估
3. 按照严格的评分标准进行打分
4. 以 JSON 格式返回评分结果

## 重要评分原则
- **批判性评分**：以审慎、严肃、批判的态度评估每篇文章
- **拉开区间**：充分利用0-100的分数范围，避免分数集中
- **区分度明显**：即使质量相近的文章也要找出细微差异，给出不同分数
- **避免堆积**：防止多篇文章获得相同或相近的分数
- **严格标准**：不可滥给高分，除非确实内容非常高质量且合适，避免打出极高分

## 三个评分维度

### 相关性评分（0-40分）
- 直接相关（35-40分）：内容直接、完整、准确地回答研究主题核心问题
- 高度相关（25-34分）：内容与主题密切相关，有实质性帮助
- 中度相关（15-24分）：内容有一定关联，部分内容有用
- 低度相关（5-14分）：仅涉及主题边缘，帮助有限
- 不相关（0-4分）：与主题无关或仅有表面联系

### 质量评分（0-30分）
- 权威专业（25-30分）：顶级来源，内容极其深入，有独特见解
- 优质内容（18-24分）：信息准确，有一定深度
- 中等质量（12-17分）：内容基本准确，但深度一般
- 质量一般（6-11分）：内容浅显或来源一般
- 质量较差（0-5分）：内容空洞、错误或来源不可靠

### 时效性评分（0-30分）
- 最新内容（25-30分）：3个月内的最新信息
- 较新内容（18-24分）：6个月内的信息
- 适中时效（12-17分）：1年内的信息
- 较旧内容（6-11分）：1-3年的信息
- 过时内容（0-5分）：3年以上或已不适用

**特殊情况**：
- 时间不敏感内容（如基础概念、原理、教程等）：根据内容质量给15-25分
- 经典文档（如官方文档、权威教程）：即使较旧但仍有效，给15-20分
- 历史性内容（如发展历程、版本演进）：根据记录完整性给10-20分

## 输出格式
你必须严格按照以下 JSON 格式输出评分结果：
{
  "keyword": "研究主题或关键词",
  "result": {
    "title": "文章标题",
    "score": 总分（0-100）,
    "relevance": 相关性得分,
    "quality": 质量得分,
    "timeliness": 时效性得分,
    "comments": "评分理由，简要说明各维度评分的依据，包括文章的优点和不足"
  },
  "status": "completed"
}

## 评分步骤
1. 首先使用 view 工具读取文件内容
2. 分析文章与研究主题的关系
3. 评估文章的质量和权威性
4. 判断文章的时效性
5. 计算三个维度的分数
6. 生成 JSON 格式的评分结果

请记住：你的评分必须严格、公正、有区分度。`

	// 创建并返回 Agent
	return agent.NewBuilder(
		g,
		"article_evaluator",
		"评估文章质量并返回评分",
		systemPrompt,
	).WithInputSchema(
		map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "要评估的文件路径",
			},
			"research_topic": map[string]any{
				"type":        "string",
				"description": "研究主题或关键词",
			},
		},
		"file_path", "research_topic", // 必需字段
	).WithTools(viewTool).
		WithModel("openai/gpt-5-mini").
		WithTemperature(0.3). // 降低温度以获得更一致的评分
		WithMaxTokens(2000).
		WithMaxRounds(3). // 可能需要多轮来读取文件并评分
		WithLogging(true).
		Build()
}

// ParseEvaluationResult 解析评估结果
func ParseEvaluationResult(result string) (*ArticleEvaluation, error) {
	var evaluation ArticleEvaluation

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
		if err := json.Unmarshal([]byte(result), &evaluation); err != nil {
			return nil, fmt.Errorf("failed to parse evaluation result: %w", err)
		}
	} else {
		// 提取 JSON 部分
		jsonStr := result[startIdx : endIdx+1]
		if err := json.Unmarshal([]byte(jsonStr), &evaluation); err != nil {
			return nil, fmt.Errorf("failed to parse evaluation JSON: %w", err)
		}
	}

	return &evaluation, nil
}