package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"gentica/pubsub"
	"gentica/session"

	"gentica/llm/tools"
	"gentica/message"
)

// Common errors
var (
	ErrRequestCancelled = errors.New("request canceled by user")
)

// AgentState represents the current state of the agent
type AgentState string

const (
	AgentStateIdle        AgentState = "idle"
	AgentStateProcessing  AgentState = "processing"
)

type AgentEventType string

const (
	AgentEventTypeError     AgentEventType = "error"
	AgentEventTypeResponse  AgentEventType = "response"
)

type AgentEvent struct {
	Type    AgentEventType
	Message message.Message
	Error   error
	Done    bool
}

// LLMProvider defines a generic interface for LLM providers
type LLMProvider interface {
	StreamResponse(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan ProviderEvent
	Model() ModelInfo
}

// ModelInfo contains basic model information
type ModelInfo struct {
	ID                 string
	Name               string
	SupportsImages     bool
	SupportsTools      bool
	CostPer1MIn        float64
	CostPer1MOut       float64
	CostPer1MInCached  float64
	CostPer1MOutCached float64
}

// ProviderEvent represents events from the LLM provider
type ProviderEvent struct {
	Type      ProviderEventType
	Content   string
	Thinking  string
	Signature string
	ToolCall  *tools.ToolCall
	Response  *ProviderResponse
	Error     error
}

type ProviderEventType string

const (
	EventContentDelta   ProviderEventType = "content_delta"
	EventThinkingDelta  ProviderEventType = "thinking_delta"
	EventSignatureDelta ProviderEventType = "signature_delta"
	EventToolUseStart   ProviderEventType = "tool_use_start"
	EventToolUseDelta   ProviderEventType = "tool_use_delta"
	EventToolUseStop    ProviderEventType = "tool_use_stop"
	EventComplete       ProviderEventType = "complete"
	EventError          ProviderEventType = "error"
)

// ProviderResponse contains the complete response from the provider
type ProviderResponse struct {
	Content      string
	FinishReason message.FinishReason
	ToolCalls    []tools.ToolCall
	Usage        TokenUsage
}

// TokenUsage tracks token consumption
type TokenUsage struct {
	InputTokens         int64
	OutputTokens        int64
	CacheCreationTokens int64
	CacheReadTokens     int64
}

// AgentConfig contains configuration for the agent
type AgentConfig struct {
	ID           string
	Name         string
	SystemPrompt string
	Tools        []tools.BaseTool
	Capabilities AgentCapabilities
}

// AgentCapabilities defines what the agent can do
type AgentCapabilities struct {
	SupportsImages bool
	SupportsTools  bool
}

type Service interface {
	pubsub.Suscriber[AgentEvent]
	Model() ModelInfo
	Run(ctx context.Context, sessionID string, content string, attachments ...message.Attachment) (<-chan AgentEvent, error)
	Cancel(sessionID string)
	CancelAll()
	IsSessionBusy(sessionID string) bool
	IsBusy() bool
	QueuedPrompts(sessionID string) int
	ClearQueue(sessionID string)
	GetState() AgentState
}

type agent struct {
	*pubsub.Broker[AgentEvent]
	config   AgentConfig
	provider LLMProvider
	model    ModelInfo
	sessions session.Service
	messages message.Service

	// State management
	state          AgentState
	stateMutex     sync.RWMutex
	activeRequests map[string]context.CancelFunc
	requestMutex   sync.RWMutex
	promptQueue    map[string][]string
	queueMutex     sync.RWMutex
}

// NewAgent creates a new agent with the given configuration and dependencies
func NewAgent(
	config AgentConfig,
	provider LLMProvider,
	model ModelInfo,
	sessions session.Service,
	messages message.Service,
) Service {
	a := &agent{
		Broker:         pubsub.NewBroker[AgentEvent](),
		config:         config,
		provider:       provider,
		model:          model,
		messages:       messages,
		sessions:       sessions,
		state:          AgentStateIdle,
		activeRequests: make(map[string]context.CancelFunc),
		promptQueue:    make(map[string][]string),
	}

	return a
}


func (a *agent) Model() ModelInfo {
	return a.model
}

// GetState returns the current state of the agent
func (a *agent) GetState() AgentState {
	a.stateMutex.RLock()
	defer a.stateMutex.RUnlock()
	return a.state
}

// setState updates the agent's state
func (a *agent) setState(state AgentState) {
	a.stateMutex.Lock()
	defer a.stateMutex.Unlock()
	a.state = state
}

func (a *agent) Cancel(sessionID string) {
	a.requestMutex.Lock()
	defer a.requestMutex.Unlock()

	// Cancel regular requests
	if cancel, ok := a.activeRequests[sessionID]; ok && cancel != nil {
		slog.Info("Request cancellation initiated", "session_id", sessionID)
		cancel()
		delete(a.activeRequests, sessionID)
	}


	a.queueMutex.Lock()
	defer a.queueMutex.Unlock()
	if len(a.promptQueue[sessionID]) > 0 {
		slog.Info("Clearing queued prompts", "session_id", sessionID)
		delete(a.promptQueue, sessionID)
	}
}

func (a *agent) IsBusy() bool {
	a.requestMutex.RLock()
	defer a.requestMutex.RUnlock()
	return len(a.activeRequests) > 0
}

func (a *agent) IsSessionBusy(sessionID string) bool {
	a.requestMutex.RLock()
	defer a.requestMutex.RUnlock()
	_, busy := a.activeRequests[sessionID]
	return busy
}

func (a *agent) QueuedPrompts(sessionID string) int {
	a.queueMutex.RLock()
	defer a.queueMutex.RUnlock()
	return len(a.promptQueue[sessionID])
}


func (a *agent) err(err error) AgentEvent {
	return AgentEvent{
		Type:  AgentEventTypeError,
		Error: err,
	}
}

func (a *agent) Run(ctx context.Context, sessionID string, content string, attachments ...message.Attachment) (<-chan AgentEvent, error) {
	if !a.model.SupportsImages && attachments != nil {
		attachments = nil
	}
	events := make(chan AgentEvent)

	if a.IsSessionBusy(sessionID) {
		a.queueMutex.Lock()
		a.promptQueue[sessionID] = append(a.promptQueue[sessionID], content)
		a.queueMutex.Unlock()
		return nil, nil
	}

	genCtx, cancel := context.WithCancel(ctx)

	a.requestMutex.Lock()
	a.activeRequests[sessionID] = cancel
	a.requestMutex.Unlock()
	a.setState(AgentStateProcessing)

	go func() {
		slog.Debug("Request started", "sessionID", sessionID)
		defer func() {
			if r := recover(); r != nil {
				events <- a.err(fmt.Errorf("panic while running the agent: %v", r))
			}
			close(events)
		}()

		var attachmentParts []message.ContentPart
		for _, attachment := range attachments {
			attachmentParts = append(attachmentParts, message.BinaryContent{Path: attachment.FilePath, MIMEType: attachment.MimeType, Data: attachment.Content})
		}

		result := a.processGeneration(genCtx, sessionID, content, attachmentParts)
		if result.Error != nil && !errors.Is(result.Error, ErrRequestCancelled) && !errors.Is(result.Error, context.Canceled) {
			slog.Error(result.Error.Error())
		}

		slog.Debug("Request completed", "sessionID", sessionID)

		a.requestMutex.Lock()
		delete(a.activeRequests, sessionID)
		a.requestMutex.Unlock()

		if !a.IsBusy() {
			a.setState(AgentStateIdle)
		}

		cancel()
		a.Publish(pubsub.CreatedEvent, result)

		select {
		case events <- result:
		case <-genCtx.Done():
		}
	}()

	return events, nil
}

func (a *agent) processGeneration(ctx context.Context, sessionID, content string, attachmentParts []message.ContentPart) AgentEvent {
	// List existing messages; if none, start title generation asynchronously.
	msgs, err := a.messages.List(ctx, sessionID)
	if err != nil {
		return a.err(fmt.Errorf("failed to list messages: %w", err))
	}


	sess, err := a.sessions.Get(ctx, sessionID)
	if err != nil {
		return a.err(fmt.Errorf("failed to get session: %w", err))
	}
	if sess.SummaryMessageID != "" {
		summaryMsgIndex := -1
		for i, msg := range msgs {
			if msg.ID == sess.SummaryMessageID {
				summaryMsgIndex = i
				break
			}
		}
		if summaryMsgIndex != -1 {
			msgs = msgs[summaryMsgIndex:]
			msgs[0].Role = message.User
		}
	}

	userMsg, err := a.createUserMessage(ctx, sessionID, content, attachmentParts)
	if err != nil {
		return a.err(fmt.Errorf("failed to create user message: %w", err))
	}
	// Append the new user message to the conversation history.
	msgHistory := append(msgs, userMsg)

	for {
		// Check for cancellation before each iteration
		select {
		case <-ctx.Done():
			return a.err(ctx.Err())
		default:
			// Continue processing
		}
		agentMessage, toolResults, err := a.streamAndHandleEvents(ctx, sessionID, msgHistory)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				agentMessage.AddFinish(message.FinishReasonCanceled, "Request cancelled", "")
				a.messages.Update(context.Background(), agentMessage)
				return a.err(ErrRequestCancelled)
			}
			return a.err(fmt.Errorf("failed to process events: %w", err))
		}
		// Debug logging if needed
		slog.Debug("Result", "message", agentMessage.FinishReason(), "toolResults", toolResults != nil)
		if (agentMessage.FinishReason() == message.FinishReasonToolUse) && toolResults != nil {
			// We are not done, we need to respond with the tool response
			msgHistory = append(msgHistory, agentMessage, *toolResults)
			// Check for queued prompts
			a.queueMutex.Lock()
			queuedPrompts := a.promptQueue[sessionID]
			if len(queuedPrompts) > 0 {
				a.promptQueue[sessionID] = nil
			}
			a.queueMutex.Unlock()

			for _, prompt := range queuedPrompts {
				userMsg, err := a.createUserMessage(ctx, sessionID, prompt, nil)
				if err != nil {
					return a.err(fmt.Errorf("failed to create user message for queued prompt: %w", err))
				}
				msgHistory = append(msgHistory, userMsg)
			}

			continue
		} else if agentMessage.FinishReason() == message.FinishReasonEndTurn {
			a.queueMutex.Lock()
			queuedPrompts := a.promptQueue[sessionID]
			if len(queuedPrompts) > 0 {
				a.promptQueue[sessionID] = nil
			}
			a.queueMutex.Unlock()

			for _, prompt := range queuedPrompts {
				if prompt == "" {
					continue
				}
				userMsg, err := a.createUserMessage(ctx, sessionID, prompt, nil)
				if err != nil {
					return a.err(fmt.Errorf("failed to create user message for queued prompt: %w", err))
				}
				msgHistory = append(msgHistory, userMsg)
			}
			if len(queuedPrompts) > 0 {
				continue
			}
		}
		if agentMessage.FinishReason() == "" {
			// Kujtim: could not track down where this is happening but this means its cancelled
			agentMessage.AddFinish(message.FinishReasonCanceled, "Request cancelled", "")
			_ = a.messages.Update(context.Background(), agentMessage)
			return a.err(ErrRequestCancelled)
		}
		return AgentEvent{
			Type:    AgentEventTypeResponse,
			Message: agentMessage,
			Done:    true,
		}
	}
}

func (a *agent) createUserMessage(ctx context.Context, sessionID, content string, attachmentParts []message.ContentPart) (message.Message, error) {
	parts := []message.ContentPart{message.TextContent{Text: content}}
	parts = append(parts, attachmentParts...)
	return a.messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role:  message.User,
		Parts: parts,
	})
}

func (a *agent) streamAndHandleEvents(ctx context.Context, sessionID string, msgHistory []message.Message) (message.Message, *message.Message, error) {
	ctx = context.WithValue(ctx, tools.SessionIDContextKey, sessionID)

	// Create the assistant message first so the spinner shows immediately
	assistantMsg, err := a.messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role:     message.Assistant,
		Parts:    []message.ContentPart{},
		Model:    a.model.ID,
		Provider: a.model.Name, // Use model name as provider identifier
	})
	if err != nil {
		return assistantMsg, nil, fmt.Errorf("failed to create assistant message: %w", err)
	}

	// Stream response from provider with configured tools
	eventChan := a.provider.StreamResponse(ctx, msgHistory, a.config.Tools)

	// Add the session and message ID into the context if needed by tools.
	ctx = context.WithValue(ctx, tools.MessageIDContextKey, assistantMsg.ID)

	// Process each event in the stream.
	for event := range eventChan {
		if processErr := a.processEvent(ctx, sessionID, &assistantMsg, event); processErr != nil {
			if errors.Is(processErr, context.Canceled) {
				a.finishMessage(context.Background(), &assistantMsg, message.FinishReasonCanceled, "Request cancelled", "")
			} else {
				a.finishMessage(ctx, &assistantMsg, message.FinishReasonError, "API Error", processErr.Error())
			}
			return assistantMsg, nil, processErr
		}
		if ctx.Err() != nil {
			a.finishMessage(context.Background(), &assistantMsg, message.FinishReasonCanceled, "Request cancelled", "")
			return assistantMsg, nil, ctx.Err()
		}
	}

	toolResults := make([]message.ToolResult, len(assistantMsg.ToolCalls()))
	toolCalls := assistantMsg.ToolCalls()
	for i, toolCall := range toolCalls {
		select {
		case <-ctx.Done():
			a.finishMessage(context.Background(), &assistantMsg, message.FinishReasonCanceled, "Request cancelled", "")
			// Make all future tool calls cancelled
			for j := i; j < len(toolCalls); j++ {
				toolResults[j] = message.ToolResult{
					ToolCallID: toolCalls[j].ID,
					Content:    "Tool execution canceled by user",
					IsError:    true,
				}
			}
			goto out
		default:
			// Continue processing
			var tool tools.BaseTool
			for _, availableTool := range a.config.Tools {
				if availableTool.Name() == toolCall.Name {
					tool = availableTool
					break
				}
			}

			// Tool not found
			if tool == nil {
				toolResults[i] = message.ToolResult{
					ToolCallID: toolCall.ID,
					Content:    fmt.Sprintf("Tool not found: %s", toolCall.Name),
					IsError:    true,
				}
				continue
			}

			// Run tool in goroutine to allow cancellation
			type toolExecResult struct {
				response tools.ToolResponse
				err      error
			}
			resultChan := make(chan toolExecResult, 1)

			go func() {
				response, err := tool.Run(ctx, tools.ToolCall{
					ID:    toolCall.ID,
					Name:  toolCall.Name,
					Input: toolCall.Input,
				})
				resultChan <- toolExecResult{response: response, err: err}
			}()

			var toolResponse tools.ToolResponse
			var toolErr error

			select {
			case <-ctx.Done():
				a.finishMessage(context.Background(), &assistantMsg, message.FinishReasonCanceled, "Request cancelled", "")
				// Mark remaining tool calls as cancelled
				for j := i; j < len(toolCalls); j++ {
					toolResults[j] = message.ToolResult{
						ToolCallID: toolCalls[j].ID,
						Content:    "Tool execution canceled by user",
						IsError:    true,
					}
				}
				goto out
			case result := <-resultChan:
				toolResponse = result.response
				toolErr = result.err
			}

			if toolErr != nil {
				slog.Error("Tool execution error", "toolCall", toolCall.ID, "error", toolErr)
				if errors.Is(toolErr, fmt.Errorf("permission denied")) {
					toolResults[i] = message.ToolResult{
						ToolCallID: toolCall.ID,
						Content:    "Permission denied",
						IsError:    true,
					}
					for j := i + 1; j < len(toolCalls); j++ {
						toolResults[j] = message.ToolResult{
							ToolCallID: toolCalls[j].ID,
							Content:    "Tool execution canceled by user",
							IsError:    true,
						}
					}
					a.finishMessage(ctx, &assistantMsg, message.FinishReasonPermissionDenied, "Permission denied", "")
					break
				}
			}
			toolResults[i] = message.ToolResult{
				ToolCallID: toolCall.ID,
				Content:    toolResponse.Content,
				Metadata:   toolResponse.Metadata,
				IsError:    toolResponse.IsError,
			}
		}
	}
out:
	if len(toolResults) == 0 {
		return assistantMsg, nil, nil
	}
	parts := make([]message.ContentPart, 0)
	for _, tr := range toolResults {
		parts = append(parts, tr)
	}
	msg, err := a.messages.Create(context.Background(), assistantMsg.SessionID, message.CreateMessageParams{
		Role:     message.Tool,
		Parts:    parts,
		Provider: a.model.Name,
	})
	if err != nil {
		return assistantMsg, nil, fmt.Errorf("failed to create cancelled tool message: %w", err)
	}

	return assistantMsg, &msg, err
}

func (a *agent) finishMessage(ctx context.Context, msg *message.Message, finishReason message.FinishReason, message, details string) {
	msg.AddFinish(finishReason, message, details)
	_ = a.messages.Update(ctx, *msg)
}

func (a *agent) processEvent(ctx context.Context, sessionID string, assistantMsg *message.Message, event ProviderEvent) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue processing.
	}

	switch event.Type {
	case EventThinkingDelta:
		assistantMsg.AppendReasoningContent(event.Thinking)
		return a.messages.Update(ctx, *assistantMsg)
	case EventSignatureDelta:
		assistantMsg.AppendReasoningSignature(event.Signature)
		return a.messages.Update(ctx, *assistantMsg)
	case EventContentDelta:
		assistantMsg.FinishThinking()
		assistantMsg.AppendContent(event.Content)
		return a.messages.Update(ctx, *assistantMsg)
	case EventToolUseStart:
		assistantMsg.FinishThinking()
		slog.Info("Tool call started", "toolCall", event.ToolCall)
		assistantMsg.AddToolCall(message.ToolCall{
			ID:    event.ToolCall.ID,
			Name:  event.ToolCall.Name,
			Input: event.ToolCall.Input,
		})
		return a.messages.Update(ctx, *assistantMsg)
	case EventToolUseDelta:
		assistantMsg.AppendToolCallInput(event.ToolCall.ID, event.ToolCall.Input)
		return a.messages.Update(ctx, *assistantMsg)
	case EventToolUseStop:
		slog.Info("Finished tool call", "toolCall", event.ToolCall)
		assistantMsg.FinishToolCall(event.ToolCall.ID)
		return a.messages.Update(ctx, *assistantMsg)
	case EventError:
		return event.Error
	case EventComplete:
		assistantMsg.FinishThinking()
		// Convert tool calls from provider format to message format
		var msgToolCalls []message.ToolCall
		for _, tc := range event.Response.ToolCalls {
			msgToolCalls = append(msgToolCalls, message.ToolCall{
				ID:    tc.ID,
				Name:  tc.Name,
				Input: tc.Input,
			})
		}
		assistantMsg.SetToolCalls(msgToolCalls)
		assistantMsg.AddFinish(event.Response.FinishReason, "", "")
		if err := a.messages.Update(ctx, *assistantMsg); err != nil {
			return fmt.Errorf("failed to update message: %w", err)
		}
		return a.TrackUsage(ctx, sessionID, a.model, event.Response.Usage)
	}

	return nil
}

func (a *agent) TrackUsage(ctx context.Context, sessionID string, model ModelInfo, usage TokenUsage) error {
	sess, err := a.sessions.Get(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	cost := model.CostPer1MInCached/1e6*float64(usage.CacheCreationTokens) +
		model.CostPer1MOutCached/1e6*float64(usage.CacheReadTokens) +
		model.CostPer1MIn/1e6*float64(usage.InputTokens) +
		model.CostPer1MOut/1e6*float64(usage.OutputTokens)

	sess.Cost += cost
	sess.CompletionTokens = usage.OutputTokens + usage.CacheReadTokens
	sess.PromptTokens = usage.InputTokens + usage.CacheCreationTokens

	_, err = a.sessions.Save(ctx, sess)
	if err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}
	return nil
}


func (a *agent) ClearQueue(sessionID string) {
	a.queueMutex.Lock()
	defer a.queueMutex.Unlock()
	if len(a.promptQueue[sessionID]) > 0 {
		slog.Info("Clearing queued prompts", "session_id", sessionID)
		delete(a.promptQueue, sessionID)
	}
}

func (a *agent) CancelAll() {
	if !a.IsBusy() {
		return
	}
	a.requestMutex.RLock()
	sessionIDs := make([]string, 0, len(a.activeRequests))
	for sessionID := range a.activeRequests {
		sessionIDs = append(sessionIDs, sessionID)
	}
	a.requestMutex.RUnlock()

	for _, sessionID := range sessionIDs {
		a.Cancel(sessionID)
	}

	timeout := time.After(5 * time.Second)
	for a.IsBusy() {
		select {
		case <-timeout:
			return
		default:
			time.Sleep(200 * time.Millisecond)
		}
	}
}

// Removed UpdateModel method - no longer needed with simplified design
