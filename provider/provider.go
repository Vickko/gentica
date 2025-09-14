package provider

import (
	"context"
	"fmt"

	"gentica/message"
	"gentica/tools"
)

type EventType string

const maxRetries = 8

const (
	EventContentStart   EventType = "content_start"
	EventToolUseStart   EventType = "tool_use_start"
	EventToolUseDelta   EventType = "tool_use_delta"
	EventToolUseStop    EventType = "tool_use_stop"
	EventContentDelta   EventType = "content_delta"
	EventThinkingDelta  EventType = "thinking_delta"
	EventSignatureDelta EventType = "signature_delta"
	EventContentStop    EventType = "content_stop"
	EventComplete       EventType = "complete"
	EventError          EventType = "error"
	EventWarning        EventType = "warning"
)

type TokenUsage struct {
	InputTokens         int64
	OutputTokens        int64
	CacheCreationTokens int64
	CacheReadTokens     int64
}

type ProviderResponse struct {
	Content      string
	ToolCalls    []message.ToolCall
	Usage        TokenUsage
	FinishReason message.FinishReason
}

type ProviderEvent struct {
	Type EventType

	Content   string
	Thinking  string
	Signature string
	Response  *ProviderResponse
	ToolCall  *message.ToolCall
	Error     error
}

// Provider 统一的Provider接口
type Provider interface {
	Send(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (*ProviderResponse, error)
	Stream(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan ProviderEvent
	Model() Model
}

// ProviderOptions 简化的provider选项
type ProviderOptions struct {
	DisableCache  bool
	SystemMessage string
}

// NewProvider 创建provider实例，简化的工厂函数
func NewProvider(cfg ProviderConfig, opts *ProviderOptions) (Provider, error) {
	// 直接使用传入的配置，不需要解析或全局配置
	fmt.Printf("DEBUG NewProvider: Creating provider of type %s\n", cfg.Type)
	var p Provider
	switch cfg.Type {
	case TypeAnthropic:
		p = newAnthropicClient(cfg, opts, AnthropicClientTypeNormal)
	case TypeOpenAI:
		p = newOpenAIClient(cfg, opts)
	case TypeGemini:
		p = newGeminiClient(cfg, opts)
	case TypeBedrock:
		p = newBedrockClient(cfg, opts)
	case TypeAzure:
		p = newAzureClient(cfg, opts)
	case TypeVertexAI:
		p = newVertexAIClient(cfg, opts)
	default:
		return nil, fmt.Errorf("provider not supported: %s", cfg.Type)
	}

	fmt.Printf("DEBUG NewProvider: Provider created: %v (is nil: %v)\n", p, p == nil)
	if p == nil {
		return nil, fmt.Errorf("failed to create provider: provider is nil")
	}
	return p, nil
}
