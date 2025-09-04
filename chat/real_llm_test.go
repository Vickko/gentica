package chat

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLLMRealToolUsage 使用真实LLM测试工具调用（需要配置文件）
func TestLLMRealToolUsage(t *testing.T) {
	// 只有在有配置文件时才运行真实LLM测试
	configPath := "../config/config.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skip("Skipping real LLM test: config file not found")
	}

	// 先切换到正确的目录让InitializeChat能找到配置文件
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir("..")

	// 创建临时测试环境
	tempDir := t.TempDir()
	setupTestEnvironment(t, tempDir)

	// 初始化chat系统
	err := InitializeChat()
	if err != nil {
		t.Fatalf("Failed to initialize chat: %v", err)
	}

	// 获取可用工具
	tools := GetAvailableTools()
	t.Logf("Available tools: %d", len(tools))

	// 构造简单指令让LLM使用工具
	instruction := `I need your help to analyze and work with a test project. Please do the following tasks in order:

1. First, use the current time function to show me what time it is right now.

2. List all files and directories in the path "` + tempDir + `" recursively to understand the project structure.

3. Read the main.go file located at "` + filepath.Join(tempDir, "main.go") + `" to understand the code.

4. Search for all occurrences of "fmt.Println" in the directory "` + tempDir + `" to find all print statements.

5. Finally, edit the file "` + filepath.Join(tempDir, "main.go") + `" to change "Hello, World!" to "Hello, Testing!" using a diff patch.

Please complete each task and provide detailed results. Use the appropriate tools for each task.`

	t.Logf("Sending instruction to LLM:\n%s", instruction)

	// 发送指令给LLM
	response, err := SendChatMessageWithTools(instruction, tools)
	if err != nil {
		t.Fatalf("Failed to send message with tools: %v", err)
	}

	t.Logf("LLM Response:\n%s", response)

	// 检查工具调用历史
	history := GetChatHistory()
	toolCallCount := 0
	toolsUsed := make(map[string]bool)

	for _, msg := range history {
		if len(msg.ToolCalls) > 0 {
			toolCallCount += len(msg.ToolCalls)
			for _, tc := range msg.ToolCalls {
				toolsUsed[tc.Function.Name] = true
				t.Logf("Tool called: %s with args: %s", tc.Function.Name, tc.Function.Arguments)
			}
		}
		// 打印工具调用返回结果
		if msg.ToolCallID != "" {
			t.Logf("Tool result for %s: %s", msg.ToolCallID, msg.Content)
		}
	}

	t.Logf("Total tool calls: %d", toolCallCount)
	t.Logf("Unique tools used: %d", len(toolsUsed))

	// 验证至少使用了一些工具
	if toolCallCount == 0 {
		t.Error("Expected LLM to make tool calls, but none were made")
	}

	// 期望的工具
	expectedTools := []string{"get_current_time", "list_files", "read_file"}
	for _, tool := range expectedTools {
		if toolsUsed[tool] {
			t.Logf("✓ LLM used tool: %s", tool)
		} else {
			t.Logf("- LLM did not use tool: %s", tool)
		}
	}
}
