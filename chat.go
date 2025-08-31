package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/sashabaranov/go-openai"
	"gopkg.in/yaml.v2"
)

// Message represents a chat message
type Message struct {
	Role       string            `json:"role"`
	Content    string            `json:"content,omitempty"`
	ToolCalls  []openai.ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
	Name       string            `json:"name,omitempty"`
}

// Config represents the configuration structure
type Config struct {
	LLM struct {
		APIKey      string  `yaml:"api_key"`
		BaseURL     string  `yaml:"base_url"`
		Model       string  `yaml:"model"`
		MaxTokens   int     `yaml:"max_tokens"`
		Temperature float64 `yaml:"temperature"`
	} `yaml:"llm"`
}

// FunctionHandler represents a function that can be called by the LLM
type FunctionHandler func(args map[string]interface{}) (string, error)

// Package-level variables for storing chat history and OpenAI client
var (
	chatHistory      []Message
	historyMux       sync.RWMutex
	config           Config
	client           *openai.Client
	functionHandlers map[string]FunctionHandler
	functionMux      sync.RWMutex
)

// LoadConfig loads configuration from YAML file
func LoadConfig(configPath string) error {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	return nil
}

// AddMessage adds a message to the chat history
func AddMessage(role, content string) {
	historyMux.Lock()
	defer historyMux.Unlock()

	chatHistory = append(chatHistory, Message{
		Role:    role,
		Content: content,
	})
}

// AddMessageWithToolCalls adds a message with tool calls to the chat history
func AddMessageWithToolCalls(role, content string, toolCalls []openai.ToolCall) {
	historyMux.Lock()
	defer historyMux.Unlock()

	chatHistory = append(chatHistory, Message{
		Role:      role,
		Content:   content,
		ToolCalls: toolCalls,
	})
}

// AddToolResult adds a tool result to the chat history
func AddToolResult(toolCallID, functionName, result string) {
	historyMux.Lock()
	defer historyMux.Unlock()

	chatHistory = append(chatHistory, Message{
		Role:       "tool",
		Content:    result,
		ToolCallID: toolCallID,
		Name:       functionName,
	})
}

// GetChatHistory returns a copy of the current chat history
func GetChatHistory() []Message {
	historyMux.RLock()
	defer historyMux.RUnlock()

	history := make([]Message, len(chatHistory))
	copy(history, chatHistory)
	return history
}

// ClearChatHistory clears all messages from chat history
func ClearChatHistory() {
	historyMux.Lock()
	defer historyMux.Unlock()

	chatHistory = chatHistory[:0]
}

// SendChatMessage sends a message to the LLM API and returns the response
func SendChatMessage(userMessage string) (string, error) {
	return SendChatMessageWithTools(userMessage, nil)
}

// SendChatMessageWithTools sends a message to the LLM API with optional tools and returns the response
func SendChatMessageWithTools(userMessage string, tools []openai.Tool) (string, error) {
	// Add user message to history only if not empty
	if userMessage != "" {
		AddMessage("user", userMessage)
	}

	// Convert chat history to OpenAI format
	messages := make([]openai.ChatCompletionMessage, 0)
	for _, msg := range GetChatHistory() {
		openaiMsg := openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}

		// Add tool calls if present
		if len(msg.ToolCalls) > 0 {
			openaiMsg.ToolCalls = msg.ToolCalls
		}

		// Add tool call ID if present
		if msg.ToolCallID != "" {
			openaiMsg.ToolCallID = msg.ToolCallID
		}

		// Add name if present
		if msg.Name != "" {
			openaiMsg.Name = msg.Name
		}

		messages = append(messages, openaiMsg)
	}

	// Create chat completion request
	request := openai.ChatCompletionRequest{
		Model:    config.LLM.Model,
		Messages: messages,
	}

	// Add tools if provided
	if len(tools) > 0 {
		request.Tools = tools
	}

	// Add optional parameters only if they are set
	if config.LLM.MaxTokens > 0 {
		request.MaxTokens = config.LLM.MaxTokens
	}
	if config.LLM.Temperature > 0 {
		request.Temperature = float32(config.LLM.Temperature)
	}

	// Send request to OpenAI
	ctx := context.Background()
	response, err := client.CreateChatCompletion(ctx, request)
	if err != nil {
		return "", fmt.Errorf("failed to create chat completion: %v", err)
	}

	// Extract assistant message
	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	choice := response.Choices[0]
	assistantMessage := choice.Message.Content

	// Debug: print tool calls
	if len(choice.Message.ToolCalls) > 0 {
		fmt.Printf("DEBUG: Received %d tool calls\n", len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			fmt.Printf("DEBUG: Tool call %d: %s with args: %s\n", i, tc.Function.Name, tc.Function.Arguments)
		}
	} else {
		fmt.Printf("DEBUG: No tool calls received\n")
	}

	// Handle tool calls if present
	if len(choice.Message.ToolCalls) > 0 {
		// Add assistant message with tool calls to history
		AddMessageWithToolCalls("assistant", assistantMessage, choice.Message.ToolCalls)

		// Execute tool calls
		for _, toolCall := range choice.Message.ToolCalls {
			if toolCall.Function.Name != "" {
				result, err := ExecuteFunction(toolCall.Function.Name, toolCall.Function.Arguments)
				if err != nil {
					result = fmt.Sprintf("Error executing function: %v", err)
				}

				// Add tool result to history
				AddToolResult(toolCall.ID, toolCall.Function.Name, result)
			}
		}

		// Make another request to get the final response
		return SendChatMessageWithTools("", tools)
	}

	// Add assistant message to history
	AddMessage("assistant", assistantMessage)

	return assistantMessage, nil
}

// InitializeChat initializes the chat system with configuration
func InitializeChat() error {
	// Load configuration
	configPath := "./config/config.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config file not found at %s", configPath)
	}

	err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %v", err)
	}

	// Validate required configuration
	if config.LLM.APIKey == "" {
		return fmt.Errorf("API key is required in configuration")
	}
	if config.LLM.Model == "" {
		return fmt.Errorf("model is required in configuration")
	}

	// Initialize OpenAI client
	clientConfig := openai.DefaultConfig(config.LLM.APIKey)
	if config.LLM.BaseURL != "" {
		clientConfig.BaseURL = config.LLM.BaseURL
	}
	client = openai.NewClientWithConfig(clientConfig)

	// Initialize chat history
	chatHistory = make([]Message, 0)

	// Register built-in functions
	RegisterFunction("get_current_time", GetCurrentTime)

	return nil
}

// GetChatStats returns statistics about the current chat session
func GetChatStats() map[string]int {
	historyMux.RLock()
	defer historyMux.RUnlock()

	stats := map[string]int{
		"total_messages":     len(chatHistory),
		"user_messages":      0,
		"assistant_messages": 0,
	}

	for _, msg := range chatHistory {
		switch msg.Role {
		case "user":
			stats["user_messages"]++
		case "assistant":
			stats["assistant_messages"]++
		}
	}

	return stats
}

// RegisterFunction registers a function handler
func RegisterFunction(name string, handler FunctionHandler) {
	functionMux.Lock()
	defer functionMux.Unlock()

	if functionHandlers == nil {
		functionHandlers = make(map[string]FunctionHandler)
	}
	functionHandlers[name] = handler
}

// GetCurrentTime returns the current time in a formatted string
func GetCurrentTime(args map[string]interface{}) (string, error) {
	now := time.Now()
	format := "2006-01-02 15:04:05"

	// Check if a specific format is requested
	if f, ok := args["format"]; ok {
		if formatStr, ok := f.(string); ok {
			format = formatStr
		}
	}

	return fmt.Sprintf("Current time: %s", now.Format(format)), nil
}

// GetAvailableTools returns the available tools for function calling
func GetAvailableTools() []openai.Tool {
	return []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_current_time",
				Description: "Get the current date and time",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"format": map[string]interface{}{
							"type":        "string",
							"description": "The time format (optional, defaults to 2006-01-02 15:04:05)",
						},
					},
					"required": []string{},
				},
			},
		},
	}
}

// ExecuteFunction executes a registered function by name
func ExecuteFunction(name string, arguments string) (string, error) {
	functionMux.RLock()
	handler, exists := functionHandlers[name]
	functionMux.RUnlock()

	if !exists {
		return "", fmt.Errorf("function %s not found", name)
	}

	// Parse arguments
	var args map[string]interface{}
	if arguments != "" {
		err := json.Unmarshal([]byte(arguments), &args)
		if err != nil {
			return "", fmt.Errorf("failed to parse function arguments: %v", err)
		}
	}

	return handler(args)
}
