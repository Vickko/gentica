package agents

// ChatAgentConfig 普通对话 Agent 配置
var ChatAgentConfig = AgentConfig{
	Name:         "Chat Agent",
	SystemPrompt: "你是一个智能助手，可以处理普通对话请求，并在需要时调用其他Agent工具。",
	Model:        "gemini-2.5-pro", // 直接指定模型
}
