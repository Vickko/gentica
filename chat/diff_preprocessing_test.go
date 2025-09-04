package chat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDiffPreprocessing 测试diff预处理功能
func TestDiffPreprocessing(t *testing.T) {
	// 创建测试文件
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")
	content := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
	fmt.Println("This is a test file")
}`
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 初始化工具
	functionRegistry = NewFunctionRegistry()
	RegisterFunction(NewEditFileFunction())

	// 测试LLM生成的完整diff格式
	llmDiff := `--- main.go	2025-09-01 17:51:45
+++ main.go	2025-09-01 17:51:47
@@ -5,4 +5,4 @@
 
 func main() {
-	fmt.Println("Hello, World!")
+	fmt.Println("Hello, Testing!")
 	fmt.Println("This is a test file")`

	t.Logf("Testing LLM-generated diff format:\n%s", llmDiff)

	// 构造参数
	args := map[string]interface{}{
		"file_path":    testFile,
		"diff_content": llmDiff,
	}
	argsJSON, _ := json.Marshal(args)

	// 调用工具
	result, err := functionRegistry.Execute("edit_file", string(argsJSON))
	if err != nil {
		t.Errorf("edit_file failed with LLM diff format: %v", err)
	} else {
		t.Logf("✓ edit_file succeeded: %s", result)

		// 验证文件内容
		modifiedContent, _ := os.ReadFile(testFile)
		if strings.Contains(string(modifiedContent), "Hello, Testing!") {
			t.Log("✓ File successfully modified with LLM diff format")
		} else {
			t.Error("✗ File was not modified correctly")
		}
	}

	// 测试标准hunk格式
	// 重置文件
	os.WriteFile(testFile, []byte(content), 0644)

	standardDiff := `@@ -5,4 +5,4 @@
 
 func main() {
-	fmt.Println("Hello, World!")
+	fmt.Println("Hello, Standard!")
 	fmt.Println("This is a test file")`

	t.Logf("Testing standard hunk format:\n%s", standardDiff)

	args2 := map[string]interface{}{
		"file_path":    testFile,
		"diff_content": standardDiff,
	}
	argsJSON2, _ := json.Marshal(args2)

	result2, err := functionRegistry.Execute("edit_file", string(argsJSON2))
	if err != nil {
		t.Errorf("edit_file failed with standard diff format: %v", err)
	} else {
		t.Logf("✓ edit_file succeeded: %s", result2)

		// 验证文件内容
		modifiedContent2, _ := os.ReadFile(testFile)
		if strings.Contains(string(modifiedContent2), "Hello, Standard!") {
			t.Log("✓ File successfully modified with standard diff format")
		} else {
			t.Error("✗ File was not modified correctly with standard format")
		}
	}
}
