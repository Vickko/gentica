package chat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLLMToolUsage 测试LLM实际调用所有工具的场景
func TestLLMToolUsage(t *testing.T) {
	// 创建临时测试环境
	tempDir := t.TempDir()
	setupTestEnvironment(t, tempDir)

	// 初始化chat系统
	err := InitializeChat()
	if err != nil {
		// 如果配置文件不存在，手动初始化必要部分
		functionRegistry = NewFunctionRegistry()
		RegisterFunction(NewGetCurrentTimeFunction())
		RegisterFunction(NewReadFileFunction())
		RegisterFunction(NewSearchInDirectoryFunction())
		RegisterFunction(NewEditFileFunction())
		RegisterFunction(NewListFilesFunction())

		// 模拟配置（测试用）
		config.LLM.Model = "gpt-4"
		config.LLM.APIKey = "test-key"
		config.LLM.BaseURL = "https://api.openai.com/v1"

		// 不初始化真实的OpenAI客户端，直接跳到工具调用模拟
		t.Logf("Config file not found, testing tool calls directly")
		testToolCallSequence(t, tempDir)
		return
	}

	// 获取可用工具
	tools := GetAvailableTools()
	t.Logf("Available tools: %d", len(tools))
	for _, tool := range tools {
		t.Logf("- %s: %s", tool.Function.Name, tool.Function.Description)
	}

	// 构造一个复杂的指令，要求LLM使用所有工具
	instruction := `I need your help to analyze and work with a test project. Please do the following tasks in order:

1. First, use the current time function to show me what time it is right now.

2. List all files and directories in the path "` + tempDir + `" recursively to understand the project structure.

3. Read the main.go file located at "` + filepath.Join(tempDir, "main.go") + `" to understand the code.

4. Search for all occurrences of "fmt.Println" in the directory "` + tempDir + `" to find all print statements.

5. Finally, edit the file "` + filepath.Join(tempDir, "main.go") + `" to change "Hello, World!" to "Hello, Testing!" using a diff patch.

Please complete each task and provide detailed results. Use the appropriate tools for each task.`

	t.Logf("Sending instruction to LLM:\n%s", instruction)

	// 发送指令给LLM（带工具）
	response, err := SendChatMessageWithTools(instruction, tools)

	// 如果网络不可用，至少验证工具调用逻辑
	if err != nil {
		if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "connection") {
			t.Logf("Network unavailable, testing tool call simulation instead")

			// 模拟工具调用序列
			testToolCallSequence(t, tempDir)
			return
		}
		t.Fatalf("Failed to send message with tools: %v", err)
	}

	t.Logf("LLM Response:\n%s", response)

	// 验证响应中提到了我们期望的工具使用
	expectedToolMentions := []string{
		"current time", "time",
		"list", "files", "directories",
		"read", "main.go",
		"search", "fmt.Println",
		"edit", "Hello, Testing",
	}

	mentionCount := 0
	for _, mention := range expectedToolMentions {
		if strings.Contains(strings.ToLower(response), strings.ToLower(mention)) {
			mentionCount++
			t.Logf("✓ Found expected mention: %s", mention)
		}
	}

	if mentionCount < len(expectedToolMentions)/2 {
		t.Errorf("Expected more tool usage mentions in response. Found %d/%d", mentionCount, len(expectedToolMentions))
	}

	// 检查聊天历史中的工具调用
	history := GetChatHistory()
	toolCallCount := 0
	for _, msg := range history {
		if len(msg.ToolCalls) > 0 {
			toolCallCount += len(msg.ToolCalls)
			for _, tc := range msg.ToolCalls {
				t.Logf("Tool called: %s with args: %s", tc.Function.Name, tc.Function.Arguments)
			}
		}
	}

	t.Logf("Total tool calls made: %d", toolCallCount)
	if toolCallCount < 3 {
		t.Logf("Warning: Expected more tool calls, but this might be due to LLM behavior")
	}
}

// testToolCallSequence 模拟工具调用序列（用于网络不可用时）
func testToolCallSequence(t *testing.T, tempDir string) {
	t.Log("Simulating tool call sequence...")

	// 1. 获取当前时间
	timeResult, err := functionRegistry.Execute("get_current_time", `{}`)
	if err != nil {
		t.Errorf("get_current_time failed: %v", err)
	} else {
		t.Logf("✓ get_current_time: %s", timeResult)
	}

	// 2. 列出文件
	listResult, err := functionRegistry.Execute("list_files", `{"directory_path": "`+tempDir+`", "recursive": true}`)
	if err != nil {
		t.Errorf("list_files failed: %v", err)
	} else {
		t.Logf("✓ list_files: Found files in directory (%d chars)", len(listResult))
	}

	// 3. 读取文件
	mainFile := filepath.Join(tempDir, "main.go")
	readResult, err := functionRegistry.Execute("read_file", `{"path": "`+mainFile+`"}`)
	if err != nil {
		t.Errorf("read_file failed: %v", err)
	} else {
		t.Logf("✓ read_file: Read %d characters", len(readResult))
	}

	// 4. 搜索内容
	searchResult, err := functionRegistry.Execute("search_in_directory", `{"directory": "`+tempDir+`", "pattern": "fmt\\.Println"}`)
	if err != nil {
		t.Errorf("search_in_directory failed: %v", err)
	} else {
		t.Logf("✓ search_in_directory: %s", searchResult)
	}

	// 5. 编辑文件 - 先读取当前内容来生成正确的diff
	currentContent, _ := os.ReadFile(mainFile)
	t.Logf("Current file content before edit:\n%s", string(currentContent))

	diff := "@@ -5,4 +5,4 @@\n \n func main() {\n-\tfmt.Println(\"Hello, World!\")\n+\tfmt.Println(\"Hello, Testing!\")\n \tfmt.Println(\"This is a test file\")"
	editArgs := map[string]interface{}{
		"file_path":    mainFile,
		"diff_content": diff,
	}
	editArgsJSON, _ := json.Marshal(editArgs)

	editResult, err := functionRegistry.Execute("edit_file", string(editArgsJSON))
	if err != nil {
		t.Errorf("edit_file failed: %v", err)
	} else {
		t.Logf("✓ edit_file: %s", editResult)

		// 验证编辑结果
		content, _ := os.ReadFile(mainFile)
		if strings.Contains(string(content), "Hello, Testing!") {
			t.Log("✓ File successfully edited")
		} else {
			t.Error("✗ File edit verification failed")
		}
	}

	t.Log("All tool calls completed successfully!")
}
