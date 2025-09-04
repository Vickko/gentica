package chat

import (
	"fmt"
	"testing"
)

func TestRealConnection(t *testing.T) {
	// Initialize chat with real config
	err := InitializeChat()
	if err != nil {
		t.Fatalf("Failed to initialize chat: %v", err)
	}

	// Print config info for debugging
	fmt.Printf("Config loaded:\n")
	fmt.Printf("- API Key: %s...\n", config.LLM.APIKey[:10])
	fmt.Printf("- Base URL: %s\n", config.LLM.BaseURL)
	fmt.Printf("- Model: %s\n", config.LLM.Model)

	// Send a real message
	response, err := SendChatMessage("Hello, this is a test message.")
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Print the response
	fmt.Printf("Response: %s\n", response)
}

func TestFunctionCalling(t *testing.T) {
	// Initialize chat with real config
	err := InitializeChat()
	if err != nil {
		t.Fatalf("Failed to initialize chat: %v", err)
	}

	// Clear chat history for clean test
	ClearChatHistory()

	// Test function calling
	tools := GetAvailableTools()
	fmt.Printf("Available tools: %d\n", len(tools))
	fmt.Printf("Tool name: %s\n", tools[0].Function.Name)

	// Test direct function execution
	result, err := ExecuteFunction("get_current_time", "{}")
	if err != nil {
		t.Fatalf("Failed to execute function directly: %v", err)
	}
	fmt.Printf("Direct function call result: %s\n", result)

	// Test self-describing function access
	fn, exists := functionRegistry.GetFunction("get_current_time")
	if !exists {
		t.Fatalf("Function not found in registry")
	}
	fmt.Printf("Function definition description: %s\n", fn.Definition.Description)

	// Try a more explicit request
	response, err := SendChatMessageWithTools("Please use the get_current_time function to tell me what time it is now.", tools)
	if err != nil {
		t.Fatalf("Failed to send message with tools: %v", err)
	}

	// Print the response
	fmt.Printf("Function call response: '%s'\n", response)

	// Check chat history
	history := GetChatHistory()
	fmt.Printf("Chat history length: %d\n", len(history))
	for i, msg := range history {
		fmt.Printf("Message %d: Role=%s, Content=%s, ToolCalls=%d\n",
			i, msg.Role, msg.Content, len(msg.ToolCalls))
	}
}
