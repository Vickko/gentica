package chat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEditWorkflow 测试编辑文件的正确工作流程
func TestEditWorkflow(t *testing.T) {
	// 创建测试环境
	tempDir := t.TempDir()
	setupTestEnvironment(t, tempDir)

	// 初始化工具
	functionRegistry = NewFunctionRegistry()
	RegisterFunction(NewReadFileFunction())
	RegisterFunction(NewSearchInDirectoryFunction())
	RegisterFunction(NewEditFileFunction())

	mainFile := filepath.Join(tempDir, "main.go")

	// 步骤1: 先读取文件内容（模拟LLM应该做的）
	t.Log("Step 1: Reading file to understand content and structure...")
	readArgs := map[string]interface{}{
		"path": mainFile,
	}
	readArgsJSON, _ := json.Marshal(readArgs)

	readResult, err := functionRegistry.Execute("read_file", string(readArgsJSON))
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	t.Logf("File content:\n%s", readResult)

	// 步骤2: 搜索目标内容确认位置（模拟LLM应该做的）
	t.Log("Step 2: Searching for target content to confirm location...")
	searchArgs := map[string]interface{}{
		"directory": tempDir,
		"pattern":   "Hello, World!",
	}
	searchArgsJSON, _ := json.Marshal(searchArgs)

	searchResult, err := functionRegistry.Execute("search_in_directory", string(searchArgsJSON))
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}
	t.Logf("Search result:\n%s", searchResult)

	// 步骤3: 基于读取到的精确内容创建正确的diff
	t.Log("Step 3: Creating precise diff based on actual file content...")

	// 模拟LLM现在知道：
	// - 文件有8行
	// - 第6行是: \tfmt.Println("Hello, World!")
	// - 使用制表符缩进
	// - 需要将第6行的内容替换

	correctDiff := `@@ -5,4 +5,4 @@
 
 func main() {
-	fmt.Println("Hello, World!")
+	fmt.Println("Hello, Testing!")
 	fmt.Println("This is a test file")`

	editArgs := map[string]interface{}{
		"file_path":    mainFile,
		"diff_content": correctDiff,
	}
	editArgsJSON, _ := json.Marshal(editArgs)

	editResult, err := functionRegistry.Execute("edit_file", string(editArgsJSON))
	if err != nil {
		t.Errorf("Edit failed even with correct workflow: %v", err)
	} else {
		t.Logf("✓ Edit succeeded: %s", editResult)

		// 验证结果
		modifiedContent, _ := os.ReadFile(mainFile)
		if strings.Contains(string(modifiedContent), "Hello, Testing!") &&
			!strings.Contains(string(modifiedContent), "Hello, World!") {
			t.Log("✓ File correctly modified - old content replaced with new")
		} else {
			t.Error("✗ File modification verification failed")
			t.Logf("Modified content:\n%s", string(modifiedContent))
		}
	}

	// 步骤4: 验证新描述确实提供了有用的指导
	tools := []string{"read_file", "search_in_directory", "edit_file"}
	for _, toolName := range tools {
		fn, exists := functionRegistry.GetFunction(toolName)
		if exists {
			desc := fn.Definition.Description
			t.Logf("Tool %s description: %s", toolName, desc)

			// 检查关键指导词汇
			switch toolName {
			case "read_file":
				if strings.Contains(desc, "before edit_file") {
					t.Logf("✓ read_file mentions its relationship to edit_file")
				}
			case "search_in_directory":
				if strings.Contains(desc, "before editing") {
					t.Logf("✓ search_in_directory mentions editing workflow")
				}
			case "edit_file":
				if strings.Contains(desc, "MUST first use read_file") {
					t.Logf("✓ edit_file emphasizes prerequisite steps")
				}
				if strings.Contains(desc, "line numbers") && strings.Contains(desc, "formatting") {
					t.Logf("✓ edit_file emphasizes precision requirements")
				}
			}
		}
	}
}
