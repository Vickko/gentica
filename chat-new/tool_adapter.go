package chat

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sashabaranov/go-openai"

	"gentica/llm/tools"
)

// ToolAdapter wraps an llm/tools.BaseTool to work with chat-new's Function interface
type ToolAdapter struct {
	tool tools.BaseTool
	ctx  context.Context
}

// NewToolAdapter creates a new tool adapter
func NewToolAdapter(tool tools.BaseTool) *ToolAdapter {
	return &ToolAdapter{
		tool: tool,
		ctx:  context.Background(),
	}
}

// ConvertToFunction converts an llm/tools.BaseTool to a chat-new.Function
func (ta *ToolAdapter) ConvertToFunction() *Function {
	toolInfo := ta.tool.Info()

	return &Function{
		Definition: openai.FunctionDefinition{
			Name:        toolInfo.Name,
			Description: toolInfo.Description,
			Parameters:  ta.convertParameters(toolInfo),
		},
		Handler: ta.createHandler(),
	}
}

// convertParameters converts tool parameters to OpenAI format
func (ta *ToolAdapter) convertParameters(info tools.ToolInfo) map[string]interface{} {
	// Create the parameters schema
	schema := map[string]interface{}{
		"type":       "object",
		"properties": info.Parameters,
		"required":   info.Required,
	}

	return schema
}

// createHandler creates a handler function that adapts between the interfaces
func (ta *ToolAdapter) createHandler() FunctionHandler {
	return func(args map[string]interface{}) (string, error) {
		// Convert map[string]interface{} to JSON string for ToolCall.Input
		inputJSON, err := json.Marshal(args)
		if err != nil {
			return "", fmt.Errorf("failed to marshal arguments: %w", err)
		}

		// Create a ToolCall
		toolCall := tools.ToolCall{
			ID:    generateToolCallID(),
			Name:  ta.tool.Name(),
			Input: string(inputJSON),
		}

		// Execute the tool
		response, err := ta.tool.Run(ta.ctx, toolCall)
		if err != nil {
			return "", fmt.Errorf("tool execution failed: %w", err)
		}

		// Handle different response types
		if response.IsError {
			return "", fmt.Errorf("%s", response.Content)
		}

		return response.Content, nil
	}
}

// generateToolCallID generates a unique ID for tool calls
func generateToolCallID() string {
	// Generate a random 8-byte ID
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp if random generation fails
		return fmt.Sprintf("call_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("call_%s", hex.EncodeToString(bytes))
}

// RegisterLLMTools registers all llm/tools with the function registry
func RegisterLLMTools(registry *FunctionRegistry, workingDir string) {
	// Initialize all tools from llm/tools package
	llmTools := []tools.BaseTool{
		tools.NewBashTool(workingDir),
		tools.NewViewTool(workingDir),
		tools.NewWriteTool(workingDir),
		tools.NewEditTool(workingDir),
		tools.NewMultiEditTool(workingDir),
		tools.NewGrepTool(workingDir),
		tools.NewGlobTool(workingDir),
		tools.NewLsTool(workingDir),
		tools.NewFetchTool(workingDir),
		tools.NewDownloadTool(workingDir),
	}

	// Convert and register each tool
	for _, tool := range llmTools {
		adapter := NewToolAdapter(tool)
		function := adapter.ConvertToFunction()
		registry.Register(function)
	}
}
