package agent

import (
	"context"

	"gentica/tools"
)

// AgentTool 将 Agent 包装为 BaseTool
type AgentTool struct {
	agent        Agent
	stateful     bool // 是否保持状态
	resetOnError bool // 错误时是否重置
}

// Name 返回工具名称
func (at *AgentTool) Name() string {
	return at.agent.Name()
}

// Info 返回工具信息
func (at *AgentTool) Info() tools.ToolInfo {
	config := at.agent.GetConfig()
	return tools.ToolInfo{
		Name:        at.agent.Name(),
		Description: at.agent.Description(),
		Parameters:  at.agent.InputSchema(),
		Required:    config.Required,
	}
}

// Run 执行工具
func (at *AgentTool) Run(ctx context.Context, call tools.ToolCall) (tools.ToolResponse, error) {
	// 如果非状态化，每次执行前清理历史
	if !at.stateful {
		at.agent.ClearHistory()
	}

	// 直接调用 Agent 的 Run 方法
	result, err := at.agent.Run(ctx, call.Input)
	if err != nil {
		// 错误时根据配置决定是否重置
		if at.resetOnError {
			at.agent.ClearHistory()
		}
		// 返回错误响应，但不返回 error（让 LLM 自己处理）
		return tools.NewTextErrorResponse(err.Error()), nil
	}

	return tools.NewTextResponse(result), nil
}

// AsToolAdapter 将 Agent 转换为无状态的 Tool（默认）
func AsToolAdapter(agent Agent) tools.BaseTool {
	return &AgentTool{
		agent:        agent,
		stateful:     false,
		resetOnError: true,
	}
}

// AsStatefulToolAdapter 将 Agent 转换为有状态的 Tool
func AsStatefulToolAdapter(agent Agent) tools.BaseTool {
	return &AgentTool{
		agent:        agent,
		stateful:     true,
		resetOnError: false,
	}
}

// AsToolAdapterWithConfig 使用自定义配置将 Agent 转换为 Tool
func AsToolAdapterWithConfig(agent Agent, stateful, resetOnError bool) tools.BaseTool {
	return &AgentTool{
		agent:        agent,
		stateful:     stateful,
		resetOnError: resetOnError,
	}
}

// AsGenkitTool 将 Agent 直接适配为 Genkit AI Tool
// 这提供了另一种集成方式，直接生成 Genkit 可用的 Tool
func AsGenkitTool(agent Agent) tools.BaseTool {
	// 直接返回 AgentTool，它已经实现了 BaseTool 接口
	// 可以通过 tools.AdaptBaseToolToGenkit 转换为 ai.Tool
	return AsToolAdapter(agent)
}