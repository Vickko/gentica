package agents

// AgentConfig 简化的Agent配置结构
type AgentConfig struct {
	Name         string
	SystemPrompt string
	Model        string
}

// 查询分析器的系统提示词
const queryAnalyzerSystemPrompt = `分析用户查询，判断是否需要进行网络搜索。

以下类型的查询不需要搜索，可以直接回答：
1. 纯粹的逻辑推理（如数学计算、逻辑谜题）
2. 静态概念解释（如编程语言基础、科学定义）
3. 时效性很低的稳定知识（如历史事件、经典理论）
4. 代码实现问题（如算法实现、代码修复）
5. 个人意见或建议（如最佳实践、方案选择）

需要搜索的查询包括：
1. 最新信息（如当前新闻、最新技术动态）
2. 特定产品或服务信息（如价格、功能对比）
3. 实时数据（如股票价格、天气预报）
4. 人物或公司当前状态
5. 最新研究或发展

请分析查询并返回JSON格式：
{
    "need_search": true/false,
    "reason": "简短说明原因",
    "query_type": "查询类型（如：logic_reasoning/static_knowledge/latest_info等）"
}`

// QueryAnalyzerAgentConfig 查询分析 Agent 配置
var QueryAnalyzerAgentConfig = AgentConfig{
	Name:         "Query Analyzer Agent",
	SystemPrompt: queryAnalyzerSystemPrompt,
	Model:        "gemini-2.5-pro", // 直接指定模型
}
