package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
)

// BaseAgent 是 Agent 接口的基础实现
type BaseAgent struct {
	g        *genkit.Genkit
	config   AgentConfig
	messages []*ai.Message
	mu       sync.RWMutex
}

// Name 返回 Agent 名称
func (a *BaseAgent) Name() string {
	return a.config.Name
}

// Description 返回 Agent 描述
func (a *BaseAgent) Description() string {
	return a.config.Description
}

// InputSchema 返回输入参数 schema
func (a *BaseAgent) InputSchema() map[string]any {
	return a.config.InputSchema
}

// GetConfig 返回 Agent 配置
func (a *BaseAgent) GetConfig() AgentConfig {
	return a.config
}

// Run 执行 Agent 任务 - 解析输入并执行
func (a *BaseAgent) Run(ctx context.Context, input string) (string, error) {
	// 1. 根据 schema 解析 JSON input
	params, err := a.parseInput(input)
	if err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	// 2. 验证必需参数
	if err := a.validateRequired(params); err != nil {
		return "", err
	}

	// 3. 构建用户消息
	userMessage := a.buildUserMessage(params)
	a.AddMessage(userMessage)

	// 4. 执行对话（自动处理多轮工具调用）
	response, err := a.executeWithTools(ctx)
	if err != nil {
		return "", err
	}

	// 5. 将响应加入历史
	if response.Message != nil {
		a.AddMessage(response.Message)
	}

	// 6. 返回结果
	return response.Text(), nil
}

// executeWithTools 手动处理工具调用循环，保留完整历史
func (a *BaseAgent) executeWithTools(ctx context.Context) (*ai.ModelResponse, error) {
	var lastResponse *ai.ModelResponse
	maxRounds := a.config.MaxRounds
	if maxRounds <= 0 {
		maxRounds = 5
	}

	// 创建日志中间件（如果启用） - 只创建一次，在所有轮次中复用
	var loggingMiddleware func(core.StreamingFunc[*ai.ModelRequest, *ai.ModelResponse, *ai.ModelResponseChunk]) core.StreamingFunc[*ai.ModelRequest, *ai.ModelResponse, *ai.ModelResponseChunk]
	if a.config.EnableLogging {
		loggingMiddleware = CreateConversationLogger(nil)
	}

	for round := 0; round < maxRounds; round++ {
		// 1. 过滤消息以发送给 LLM（移除旧的工具调用）
		filteredMessages := a.filterMessagesForLLM()

		// 2. 准备生成选项
		opts := []ai.GenerateOption{
			ai.WithModelName(a.config.Model),
			ai.WithSystem(a.config.SystemPrompt),
			ai.WithMessages(filteredMessages...),
			ai.WithReturnToolRequests(true), // 关键：不自动执行工具，只返回工具请求
		}

		// 添加工具（如果有）
		if len(a.config.Tools) > 0 {
			toolRefs := make([]ai.ToolRef, len(a.config.Tools))
			for i, tool := range a.config.Tools {
				toolRefs[i] = ai.ToolRef(tool)
			}
			opts = append(opts, ai.WithTools(toolRefs...))
		}

		// 添加日志中间件（如果启用）
		if loggingMiddleware != nil {
			opts = append(opts, ai.WithMiddleware(loggingMiddleware))
		}

		// 3. 调用模型（返回工具请求但不自动执行）
		response, err := genkit.Generate(ctx, a.g, opts...)
		if err != nil {
			return nil, fmt.Errorf("model generation failed: %w", err)
		}

		lastResponse = response

		// 4. 将助手响应添加到完整历史（包括工具调用）
		if response.Message != nil {
			a.AddMessage(response.Message)
		}

		// 5. 检查是否有工具调用
		hasToolCalls := false
		if response.Message != nil {
			for _, part := range response.Message.Content {
				if part.IsToolRequest() {
					hasToolCalls = true
					break
				}
			}
		}

		// 如果没有工具调用，完成
		if !hasToolCalls {
			break
		}

		// 6. 执行工具调用并创建工具响应消息
		toolResponses := make([]*ai.Part, 0)
		for _, part := range response.Message.Content {
			if !part.IsToolRequest() {
				continue
			}

			toolReq := part.ToolRequest
			// 查找并执行工具
			toolResp, err := a.executeToolByName(ctx, toolReq.Name, toolReq.Input)
			if err != nil {
				// 创建错误响应
				toolResponses = append(toolResponses, ai.NewToolResponsePart(&ai.ToolResponse{
					Name:   toolReq.Name,
					Ref:    toolReq.Ref,
					Output: fmt.Sprintf("Tool execution failed: %v", err),
				}))
			} else {
				// 创建成功响应
				toolResponses = append(toolResponses, ai.NewToolResponsePart(&ai.ToolResponse{
					Name:   toolReq.Name,
					Ref:    toolReq.Ref,
					Output: toolResp,
				}))
			}
		}

		// 7. 创建工具响应消息并添加到历史
		if len(toolResponses) > 0 {
			toolMessage := &ai.Message{
				Role:    ai.RoleTool,
				Content: toolResponses,
			}
			a.AddMessage(toolMessage)
		}
	}

	return lastResponse, nil
}

// filterMessagesForLLM 过滤消息以发送给 LLM
// 保留所有消息，但过滤掉被普通回复隔断的旧工具调用
// 保留连续延伸到消息列表末尾的工具调用序列
func (a *BaseAgent) filterMessagesForLLM() []*ai.Message {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(a.messages) == 0 {
		return []*ai.Message{}
	}

	// 找到最后一个普通回复的位置
	cutoffIndex := -1
	for i := len(a.messages) - 1; i >= 0; i-- {
		if a.isNormalAssistantReply(a.messages[i]) {
			cutoffIndex = i
			break
		}
	}

	// 构建过滤后的消息
	filtered := make([]*ai.Message, 0, len(a.messages))
	for i, msg := range a.messages {
		// 在 cutoff 之后的消息全部保留
		if cutoffIndex < 0 || i > cutoffIndex {
			filtered = append(filtered, msg)
			continue
		}

		// 在 cutoff 之前的消息需要过滤
		switch msg.Role {
		case ai.RoleModel:
			// 过滤掉工具调用部分，只保留文本
			if textParts := a.extractTextParts(msg); len(textParts) > 0 {
				filtered = append(filtered, &ai.Message{
					Role:    msg.Role,
					Content: textParts,
				})
			}
		case ai.RoleTool:
			// 工具响应直接过滤掉
		default:
			// User 等其他消息保留
			filtered = append(filtered, msg)
		}
	}

	return filtered
}

// isNormalAssistantReply 检查是否是不包含工具调用的普通助手回复
func (a *BaseAgent) isNormalAssistantReply(msg *ai.Message) bool {
	if msg.Role != ai.RoleModel {
		return false
	}

	hasToolRequest := false
	hasTextContent := false
	for _, part := range msg.Content {
		if part.IsToolRequest() {
			hasToolRequest = true
		}
		if part.IsText() && part.Text != "" {
			hasTextContent = true
		}
	}

	return !hasToolRequest && hasTextContent
}

// extractTextParts 提取消息中的文本部分
func (a *BaseAgent) extractTextParts(msg *ai.Message) []*ai.Part {
	var textParts []*ai.Part
	for _, part := range msg.Content {
		if part.IsText() {
			textParts = append(textParts, part)
		}
	}
	return textParts
}

// executeToolByName 根据名称执行工具
func (a *BaseAgent) executeToolByName(ctx context.Context, toolName string, input any) (any, error) {
	// 查找工具
	var targetTool ai.Tool
	for _, tool := range a.config.Tools {
		// 使用 Tool 接口的 Name() 方法获取工具名称
		if tool.Name() == toolName {
			targetTool = tool
			break
		}
	}

	if targetTool == nil {
		return nil, fmt.Errorf("tool %s not found", toolName)
	}

	// 执行工具 - 使用 RunRaw 方法
	result, err := targetTool.RunRaw(ctx, input)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetMessages 获取消息历史
func (a *BaseAgent) GetMessages() []*ai.Message {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// 返回副本，避免外部修改
	messages := make([]*ai.Message, len(a.messages))
	copy(messages, a.messages)
	return messages
}

// SetMessages 设置消息历史（覆盖）
func (a *BaseAgent) SetMessages(messages []*ai.Message) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.messages = make([]*ai.Message, len(messages))
	copy(a.messages, messages)
}

// AddMessage 添加单条消息
func (a *BaseAgent) AddMessage(message *ai.Message) {
	if message == nil {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.messages = append(a.messages, message)
}

// ClearHistory 清空消息历史
func (a *BaseAgent) ClearHistory() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.messages = make([]*ai.Message, 0)
}

// parseInput 解析输入参数
func (a *BaseAgent) parseInput(input string) (map[string]any, error) {
	// 如果没有定义 schema，直接返回原始输入作为 "input" 参数
	if a.config.InputSchema == nil || len(a.config.InputSchema) == 0 {
		return map[string]any{"input": input}, nil
	}

	// 尝试解析为 JSON
	var params map[string]any
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		// 如果解析失败，且 schema 中只有一个字段，将整个输入作为该字段的值
		if len(a.config.InputSchema) == 1 {
			for key := range a.config.InputSchema {
				return map[string]any{key: input}, nil
			}
		}
		return nil, fmt.Errorf("failed to parse JSON input: %w", err)
	}

	return params, nil
}

// validateRequired 验证必需参数
func (a *BaseAgent) validateRequired(params map[string]any) error {
	for _, required := range a.config.Required {
		if _, ok := params[required]; !ok {
			return fmt.Errorf("missing required parameter: %s", required)
		}
	}
	return nil
}

// buildUserMessage 构建用户消息
func (a *BaseAgent) buildUserMessage(params map[string]any) *ai.Message {
	// 如果只有一个参数且名为 "input"，直接使用其值
	if len(params) == 1 {
		if input, ok := params["input"]; ok {
			if str, ok := input.(string); ok {
				return ai.NewUserTextMessage(str)
			}
		}
	}

	// 否则，将参数格式化为结构化的提示
	var prompt string
	for key, value := range params {
		prompt += fmt.Sprintf("%s: %v\n", key, value)
	}

	return ai.NewUserTextMessage(prompt)
}