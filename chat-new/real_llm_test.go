package chat

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestRealLLMWithAllTools tests all tools with a real LLM
// This test requires a valid config file with API credentials
func TestRealLLMWithAllTools(t *testing.T) {
	// Skip this test in CI/CD or when SKIP_LLM_TEST is set
	if os.Getenv("SKIP_LLM_TEST") != "" {
		t.Skip("Skipping LLM integration test")
	}

	// Check if config file exists
	configPath := "./configs/config.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Try parent directory config
		configPath = "../configs/config.yaml"
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Skip("Config file not found, skipping LLM test")
		}
	}

	// Initialize the chat system
	err := InitializeChat()
	if err != nil {
		// Try with parent directory config
		os.Chdir("..")
		err = InitializeChat()
		if err != nil {
			t.Fatalf("Failed to initialize chat: %v", err)
		}
	}

	// Enable verbose logging
	t.Log("=== Starting LLM Integration Test ===")
	t.Log("Testing all available tools with real LLM API")

	// Create a test directory for our operations
	testDir := filepath.Join(os.TempDir(), fmt.Sprintf("llm_test_%d", time.Now().Unix()))
	err = os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir) // Cleanup after test

	// Get available tools
	tools := GetAvailableTools()

	// Create a comprehensive prompt that will test all tools
	prompt := fmt.Sprintf(`You are a helpful assistant with access to various tools. I need you to help me test that all tools are working correctly. Please perform the following tasks in order:

1. First, use the 'ls' tool to list the contents of the directory: %s

2. Use the 'write' tool to create a file called "test.txt" in %s with the content:
   "Hello World
   This is a test file
   Created by LLM"

3. Use the 'view' tool to read the file you just created at %s

4. Use the 'edit' tool to change "Hello World" to "Hello LLM" in the test.txt file

5. Use the 'grep' tool to search for the pattern "LLM" in the directory %s

6. Use the 'glob' tool to find all .txt files in %s using pattern "*.txt"

7. Use the 'bash' tool to run the command: echo "Tool test successful"

8. Use the 'write' tool to create another file called "test2.md" with content "# Test Document"

9. Use the 'multiedit' tool to make two changes to test2.md:
   - Change "# Test Document" to "# Test Document Updated"
   - Add a new line "This is a test" at the end

10. Use the 'fetch' tool to get the content from https://api.github.com/zen (this returns a zen quote)

11. Finally, use 'ls' again to show all files created in %s

After completing all tasks, provide a summary of what was accomplished.`,
		testDir, testDir, filepath.Join(testDir, "test.txt"), testDir, testDir, testDir)

	// Send the message with tools
	t.Log("\n=== Sending prompt to LLM ===")
	t.Logf("Prompt:\n%s\n", prompt)
	t.Log("\n=== Executing tools (this may take a while) ===")

	response, err := SendChatMessageWithTools(prompt, tools)
	if err != nil {
		t.Fatalf("Failed to send chat message: %v", err)
	}

	// Print the response for debugging
	t.Log("\n=== Final LLM Response ===")
	t.Logf("%s\n", response)

	// Verify that files were created
	t.Log("\n=== Verifying File Operations ===")
	testFile := filepath.Join(testDir, "test.txt")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("❌ test.txt was not created")
	} else {
		content, _ := os.ReadFile(testFile)
		t.Log("✅ test.txt created")
		t.Logf("   Content: %s", string(content))

		// Check if edit was applied
		if !strings.Contains(string(content), "Hello LLM") {
			t.Error("❌ Edit tool didn't change 'Hello World' to 'Hello LLM'")
		} else {
			t.Log("✅ Edit successfully changed 'Hello World' to 'Hello LLM'")
		}
	}

	test2File := filepath.Join(testDir, "test2.md")
	if _, err := os.Stat(test2File); os.IsNotExist(err) {
		t.Error("❌ test2.md was not created")
	} else {
		content, _ := os.ReadFile(test2File)
		t.Log("✅ test2.md created")
		t.Logf("   Content: %s", string(content))

		// Check if multiedit was applied
		if !strings.Contains(string(content), "Test Document Updated") {
			t.Error("❌ MultiEdit tool didn't update the document title")
		} else {
			t.Log("✅ MultiEdit successfully updated the document")
		}
	}

	// Check the chat history to see which tools were called
	history := GetChatHistory()
	toolsUsed := make(map[string]int)

	t.Log("\n=== Detailed Tool Call History ===")
	for i, msg := range history {
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			t.Logf("\n[Step %d] Assistant called %d tool(s):", i+1, len(msg.ToolCalls))
			for j, toolCall := range msg.ToolCalls {
				toolsUsed[toolCall.Function.Name]++
				t.Logf("  Tool %d: %s", j+1, toolCall.Function.Name)
				t.Logf("    Args: %s", toolCall.Function.Arguments)
			}
		} else if msg.Role == "tool" {
			// Show tool results
			resultPreview := msg.Content
			if len(resultPreview) > 200 {
				resultPreview = resultPreview[:200] + "..."
			}
			t.Logf("  Tool Result (%s): %s", msg.Name, resultPreview)
		}
	}

	t.Log("\n=== Summary ===")
	t.Logf("Tools used: %v", toolsUsed)

	// Verify that key tools were used
	t.Log("\n=== Tool Usage Verification ===")
	expectedTools := []string{"ls", "write", "view", "edit", "grep", "glob", "bash"}
	for _, toolName := range expectedTools {
		if toolsUsed[toolName] == 0 {
			t.Errorf("❌ Tool '%s' was not used", toolName)
		} else {
			t.Logf("✅ Tool '%s' was used %d time(s)", toolName, toolsUsed[toolName])
		}
	}

	// Print summary
	t.Log("\n=== Test Summary ===")
	t.Logf("Test completed. %d different tools were used", len(toolsUsed))
	t.Logf("Test directory: %s", testDir)
}

// TestRealLLMSimpleToolCall tests a simple tool call with real LLM
func TestRealLLMSimpleToolCall(t *testing.T) {
	// Skip this test in CI/CD or when SKIP_LLM_TEST is set
	if os.Getenv("SKIP_LLM_TEST") != "" {
		t.Skip("Skipping LLM integration test")
	}

	t.Log("=== Starting Simple Tool Call Test ===")

	// Check if config file exists
	configPath := "./configs/config.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Try parent directory config
		configPath = "../configs/config.yaml"
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Skip("Config file not found, skipping LLM test")
		}
	}

	// Initialize the chat system
	err := InitializeChat()
	if err != nil {
		// Try with parent directory config
		os.Chdir("..")
		err = InitializeChat()
		if err != nil {
			t.Fatalf("Failed to initialize chat: %v", err)
		}
	}

	// Clear any previous history
	ClearChatHistory()

	// Get available tools
	tools := GetAvailableTools()

	// Simple prompt that should trigger bash tool
	prompt := "Please use the bash tool to run: echo 'Hello from bash tool'"
	t.Logf("Prompt: %s\n", prompt)

	// Send the message with tools
	t.Log("Sending message to LLM...")
	response, err := SendChatMessageWithTools(prompt, tools)
	if err != nil {
		t.Fatalf("Failed to send chat message: %v", err)
	}

	t.Log("\n=== LLM Response ===")
	t.Logf("%s\n", response)

	// Check if bash tool was used
	t.Log("\n=== Tool Call Details ===")
	history := GetChatHistory()
	bashUsed := false
	for _, msg := range history {
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			for _, toolCall := range msg.ToolCalls {
				if toolCall.Function.Name == "bash" {
					bashUsed = true
					t.Logf("Tool called: %s", toolCall.Function.Name)
					t.Logf("Arguments: %s", toolCall.Function.Arguments)
				}
			}
		}
		if msg.Role == "tool" {
			t.Logf("Tool result: %s", msg.Content)
			if strings.Contains(msg.Content, "Hello from bash tool") {
				t.Log("✅ Bash tool executed successfully")
			} else {
				t.Log("❌ Expected 'Hello from bash tool' in output")
			}
		}
	}

	if !bashUsed {
		t.Error("❌ Bash tool was not used when explicitly requested")
	} else {
		t.Log("\n=== Test Result: PASS ✅ ===")
	}
}

// TestRealLLMToolChaining tests tool chaining with real LLM
func TestRealLLMToolChaining(t *testing.T) {
	// Skip this test in CI/CD or when SKIP_LLM_TEST is set
	if os.Getenv("SKIP_LLM_TEST") != "" {
		t.Skip("Skipping LLM integration test")
	}

	// Initialize the chat system
	err := InitializeChat()
	if err != nil {
		// Try with parent directory config
		os.Chdir("..")
		err = InitializeChat()
		if err != nil {
			t.Skip("Failed to initialize chat, skipping test")
		}
	}

	// Clear any previous history
	ClearChatHistory()

	// Create a test directory
	testDir := filepath.Join(os.TempDir(), fmt.Sprintf("chain_test_%d", time.Now().Unix()))
	err = os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Get available tools
	tools := GetAvailableTools()

	// Prompt that requires chaining multiple tools
	prompt := fmt.Sprintf(`Please perform these tasks in sequence:
1. Write a file called "data.json" in %s with content: {"name": "test", "value": 42}
2. View the file to confirm it was created
3. Use grep to search for "value" in the file
4. Report back what you found`, testDir)

	t.Log("=== Starting Tool Chaining Test ===")
	t.Logf("Test directory: %s", testDir)
	t.Logf("Prompt:\n%s\n", prompt)

	// Send the message
	t.Log("Sending message to LLM...")
	response, err := SendChatMessageWithTools(prompt, tools)
	if err != nil {
		t.Fatalf("Failed to send chat message: %v", err)
	}

	t.Log("\n=== LLM Response ===")
	t.Logf("%s\n", response)

	// Verify the file was created
	jsonFile := filepath.Join(testDir, "data.json")
	if _, err := os.Stat(jsonFile); os.IsNotExist(err) {
		t.Error("data.json was not created")
	} else {
		content, _ := os.ReadFile(jsonFile)
		t.Logf("data.json content: %s", string(content))
	}

	// Check which tools were used
	t.Log("\n=== Tool Execution Sequence ===")
	history := GetChatHistory()
	toolSequence := []string{}

	for _, msg := range history {
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			for _, toolCall := range msg.ToolCalls {
				toolSequence = append(toolSequence, toolCall.Function.Name)
				t.Logf("Tool %d: %s", len(toolSequence), toolCall.Function.Name)
				t.Logf("  Args: %s", toolCall.Function.Arguments)
			}
		}
		if msg.Role == "tool" && msg.Content != "" {
			resultPreview := msg.Content
			if len(resultPreview) > 100 {
				resultPreview = resultPreview[:100] + "..."
			}
			t.Logf("  Result: %s", resultPreview)
		}
	}

	t.Log("\n=== Summary ===")
	t.Logf("Tool execution sequence: %v", toolSequence)

	// Verify expected tools were used
	if len(toolSequence) < 2 {
		t.Error("Expected at least 2 tools to be used")
	}

	// Should have used write and view at minimum
	hasWrite := false
	hasView := false
	for _, tool := range toolSequence {
		if tool == "write" {
			hasWrite = true
		}
		if tool == "view" {
			hasView = true
		}
	}

	if !hasWrite {
		t.Error("Write tool was not used")
	}
	if !hasView {
		t.Error("View tool was not used")
	}
}
