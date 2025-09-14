package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"gentica/message"
	"gentica/provider"
	"gentica/tools"
)

// Common errors
var (
	ErrRequestCancelled = errors.New("request cancelled")
	ErrNoResponse       = errors.New("no response from provider")
)

// Config holds the configuration for an agent
type Config struct {
	Name         string
	SystemPrompt string
	Model        string
}

// Agent represents a minimal LLM agent
type Agent struct {
	config   Config
	provider provider.Provider
	tools    []tools.BaseTool
	messages []message.Message // 内部消息历史
}

// NewAgent creates a new agent instance that manages its own provider
func NewAgent(config Config, providerConfig provider.ProviderConfig, tools []tools.BaseTool) (*Agent, error) {
	if config.SystemPrompt == "" {
		config.SystemPrompt = "You are a helpful AI assistant."
	}

	// 使用 Agent 的 SystemPrompt 创建 provider
	providerOpts := &provider.ProviderOptions{
		SystemMessage: config.SystemPrompt, // 使用 Agent 的系统提示
	}

	// 创建 provider
	fmt.Printf("DEBUG Agent.NewAgent: Creating provider with config: %+v\n", providerConfig)
	p, err := provider.NewProvider(providerConfig, providerOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}
	fmt.Printf("DEBUG Agent.NewAgent: Provider created: %v (is nil: %v)\n", p, p == nil)

	if p == nil {
		return nil, fmt.Errorf("provider is nil even though no error was returned")
	}

	return &Agent{
		config:   config,
		provider: p,
		tools:    tools,
		messages: []message.Message{},
	}, nil
}

// NewAgentWithProvider creates an agent with an existing provider (for backward compatibility)
func NewAgentWithProvider(config Config, provider provider.Provider, tools []tools.BaseTool) *Agent {
	if config.SystemPrompt == "" {
		config.SystemPrompt = "You are a helpful AI assistant."
	}

	return &Agent{
		config:   config,
		provider: provider,
		tools:    tools,
		messages: []message.Message{},
	}
}

// Run executes the agent with the given message history
func (a *Agent) Run(ctx context.Context, messages []message.Message) (*message.Message, error) {
	// Main reasoning and tool-calling loop
	for {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return nil, ErrRequestCancelled
		default:
		}

		// Stream response from provider
		response, toolCalls, err := a.streamResponse(ctx, messages)
		if err != nil {
			return nil, fmt.Errorf("failed to get response: %w", err)
		}

		// If no tool calls, we're done
		if len(toolCalls) == 0 {
			return response, nil
		}

		// Execute tool calls
		toolResults := a.executeTools(ctx, toolCalls)

		// Create tool result message
		toolResultMsg := a.createToolResultMessage(toolResults)

		// Append response and tool results to message history
		messages = append(messages, *response, toolResultMsg)
	}
}

// streamResponse streams a response from the provider
func (a *Agent) streamResponse(ctx context.Context, messages []message.Message) (*message.Message, []message.ToolCall, error) {
	// Create assistant message
	assistantMsg := message.Message{
		Role:  message.Assistant,
		Parts: []message.ContentPart{},
	}

	// Stream events from provider
	eventChan := a.provider.Stream(ctx, messages, a.tools)

	var toolCalls []message.ToolCall
	var textContent string

	for event := range eventChan {
		select {
		case <-ctx.Done():
			return nil, nil, ErrRequestCancelled
		default:
		}

		switch event.Type {
		case provider.EventContentDelta:
			textContent += event.Content

		case provider.EventToolUseStart:
			if event.ToolCall != nil {
				toolCalls = append(toolCalls, *event.ToolCall)
			}

		case provider.EventToolUseDelta:
			if event.ToolCall != nil && len(toolCalls) > 0 {
				// Find and update the tool call
				for i := range toolCalls {
					if toolCalls[i].ID == event.ToolCall.ID {
						toolCalls[i].Input += event.ToolCall.Input
						break
					}
				}
			}

		case provider.EventToolUseStop:
			if event.ToolCall != nil && len(toolCalls) > 0 {
				// Mark tool call as finished
				for i := range toolCalls {
					if toolCalls[i].ID == event.ToolCall.ID {
						toolCalls[i].Finished = true
						break
					}
				}
			}

		case provider.EventError:
			return nil, nil, event.Error

		case provider.EventComplete:
			// Add content to message
			if textContent != "" {
				assistantMsg.Parts = append(assistantMsg.Parts, message.TextContent{Text: textContent})
			}

			// Use tool calls from the complete event response if available
			if event.Response != nil && len(event.Response.ToolCalls) > 0 {
				// Use the complete tool calls from the response
				toolCalls = event.Response.ToolCalls
				for _, tc := range toolCalls {
					assistantMsg.Parts = append(assistantMsg.Parts, tc)
				}
			} else {
				// Fall back to accumulated tool calls from stream
				for _, tc := range toolCalls {
					assistantMsg.Parts = append(assistantMsg.Parts, tc)
				}
			}

			// Add finish reason
			if event.Response != nil {
				assistantMsg.Parts = append(assistantMsg.Parts, message.Finish{
					Reason: event.Response.FinishReason,
				})
			}

			return &assistantMsg, toolCalls, nil
		}
	}

	// If we get here without a complete event, return what we have
	if textContent != "" {
		assistantMsg.Parts = append(assistantMsg.Parts, message.TextContent{Text: textContent})
	}

	return &assistantMsg, toolCalls, nil
}

// executeTools executes the given tool calls
func (a *Agent) executeTools(ctx context.Context, toolCalls []message.ToolCall) []message.ToolResult {
	results := make([]message.ToolResult, len(toolCalls))

	for i, toolCall := range toolCalls {
		// Check for cancellation
		select {
		case <-ctx.Done():
			results[i] = message.ToolResult{
				ToolCallID: toolCall.ID,
				Content:    "Tool execution cancelled",
				IsError:    true,
			}
			continue
		default:
		}

		// Find the tool
		var tool tools.BaseTool
		for _, t := range a.tools {
			if t.Name() == toolCall.Name {
				tool = t
				break
			}
		}

		// Tool not found
		if tool == nil {
			slog.Error("Tool not found", "name", toolCall.Name)
			results[i] = message.ToolResult{
				ToolCallID: toolCall.ID,
				Content:    fmt.Sprintf("Tool not found: %s", toolCall.Name),
				IsError:    true,
			}
			continue
		}

		// Execute the tool
		response, err := tool.Run(ctx, tools.ToolCall{
			ID:    toolCall.ID,
			Name:  toolCall.Name,
			Input: toolCall.Input,
		})

		if err != nil {
			slog.Error("Tool execution failed", "name", toolCall.Name, "error", err)
			results[i] = message.ToolResult{
				ToolCallID: toolCall.ID,
				Content:    fmt.Sprintf("Tool execution failed: %v", err),
				IsError:    true,
			}
			continue
		}

		results[i] = message.ToolResult{
			ToolCallID: toolCall.ID,
			Content:    response.Content,
			Metadata:   response.Metadata,
			IsError:    response.IsError,
		}
	}

	return results
}

// createToolResultMessage creates a message containing tool results
func (a *Agent) createToolResultMessage(results []message.ToolResult) message.Message {
	parts := make([]message.ContentPart, len(results))
	for i, result := range results {
		parts[i] = result
	}

	return message.Message{
		Role:  message.Tool,
		Parts: parts,
	}
}

// GetConfig returns the agent configuration
func (a *Agent) GetConfig() Config {
	return a.config
}

// SetSystemPrompt updates the system prompt
func (a *Agent) SetSystemPrompt(prompt string) {
	a.config.SystemPrompt = prompt
}

// Chat is a convenience method that uses internal message history
func (a *Agent) Chat(ctx context.Context, userMessage string) (*message.Message, error) {
	msg := message.Message{
		Role: message.User,
		Parts: []message.ContentPart{
			message.TextContent{Text: userMessage},
		},
	}

	a.messages = append(a.messages, msg)
	response, err := a.Run(ctx, a.messages)
	if err != nil {
		return nil, err
	}

	a.messages = append(a.messages, *response)
	return response, nil
}

// GetHistory returns the current message history
func (a *Agent) GetHistory() []message.Message {
	return append([]message.Message{}, a.messages...)
}

// ClearHistory clears the message history
func (a *Agent) ClearHistory() {
	a.messages = []message.Message{}
}

// SetHistory sets the message history
func (a *Agent) SetHistory(messages []message.Message) {
	a.messages = append([]message.Message{}, messages...)
}

// AddMessage adds a message to the history
func (a *Agent) AddMessage(msg message.Message) {
	a.messages = append(a.messages, msg)
}
