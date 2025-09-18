package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/firebase/genkit/go/ai"
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

// executeWithTools 自动处理工具调用循环
func (a *BaseAgent) executeWithTools(ctx context.Context) (*ai.ModelResponse, error) {
	for round := 0; round < a.config.MaxRounds; round++ {
		// 准备生成选项
		opts := []ai.GenerateOption{
			ai.WithModelName(a.config.Model),
			ai.WithSystem(a.config.SystemPrompt),
			ai.WithMessages(a.messages...),
		}

		// 添加工具（如果有）
		if len(a.config.Tools) > 0 {
			// 转换 ai.Tool 到 ai.ToolRef
			toolRefs := make([]ai.ToolRef, len(a.config.Tools))
			for i, tool := range a.config.Tools {
				toolRefs[i] = ai.ToolRef(tool)
			}
			opts = append(opts, ai.WithTools(toolRefs...))
		}

		// 添加温度参数
		if a.config.Temperature > 0 {
			// Temperature 在 ai.ModelRequest 中设置，需要使用 WithConfig
			// 暂时注释掉，因为 WithTemperature 不存在
			// opts = append(opts, ai.WithTemperature(a.config.Temperature))
		}

		// 添加最大 token 数
		if a.config.MaxTokens > 0 {
			// MaxTokens 在 ai.ModelRequest 中设置
			// 暂时注释掉，因为 WithMaxOutputTokens 不存在
			// opts = append(opts, ai.WithMaxOutputTokens(a.config.MaxTokens))
		}

		// 执行生成
		response, err := genkit.Generate(ctx, a.g, opts...)
		if err != nil {
			return nil, err
		}

		// 如果没有工具调用，直接返回
		if len(response.ToolRequests()) == 0 {
			return response, nil
		}

		// 添加模型响应到历史（包含工具调用）
		if response.Message != nil {
			a.AddMessage(response.Message)
		}

		// 继续下一轮（genkit 会自动处理工具响应）
	}

	return nil, fmt.Errorf("exceeded max rounds (%d)", a.config.MaxRounds)
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