package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gentica/agent"
	"gentica/message"
	"gentica/tools"
)

// AgentTool 用于Agent之间调用的工具
type AgentTool struct {
	targetAgent *agent.Agent
	agentName   string
}

func NewAgentTool(targetAgent *agent.Agent, name string) tools.BaseTool {
	return &AgentTool{
		targetAgent: targetAgent,
		agentName:   name,
	}
}

func (t *AgentTool) Name() string {
	// 使用唯一的工具名，避免重复声明
	return fmt.Sprintf("agent_%s", strings.ReplaceAll(strings.ToLower(t.agentName), " ", "_"))
}

func (t *AgentTool) Info() tools.ToolInfo {
	return tools.ToolInfo{
		Name:        t.Name(), // 使用Name()方法确保一致性
		Description: fmt.Sprintf("调用%s处理任务", t.agentName),
		Parameters: map[string]any{
			"prompt": map[string]any{
				"type":        "string",
				"description": "发送给Agent的提示",
			},
		},
		Required: []string{"prompt"},
	}
}

func (t *AgentTool) Run(ctx context.Context, call tools.ToolCall) (tools.ToolResponse, error) {
	var params struct {
		Prompt string `json:"prompt"`
	}
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return tools.NewTextErrorResponse("参数解析失败: " + err.Error()), nil
	}

	if params.Prompt == "" {
		return tools.NewTextErrorResponse("提示内容不能为空"), nil
	}

	// 构建消息
	messages := []message.Message{
		{
			Role: message.User,
			Parts: []message.ContentPart{
				message.TextContent{Text: params.Prompt},
			},
		},
	}

	// 调用目标Agent
	response, err := t.targetAgent.Run(ctx, messages)
	if err != nil {
		return tools.NewTextErrorResponse(fmt.Sprintf("Agent执行失败: %v", err)), nil
	}

	// 提取文本响应
	var responseText string
	for _, part := range response.Parts {
		if textPart, ok := part.(message.TextContent); ok {
			responseText += textPart.Text
		}
	}

	return tools.NewTextResponse(responseText), nil
}
