package chat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAllToolsIntegration 测试所有工具的集成效果
func TestAllToolsIntegration(t *testing.T) {
	// 创建临时测试环境
	tempDir := t.TempDir()

	// 创建测试文件和目录结构
	setupTestEnvironment(t, tempDir)

	// 初始化chat系统（无需配置文件，只测试工具注册）
	functionRegistry = NewFunctionRegistry()
	RegisterFunction(NewGetCurrentTimeFunction())
	RegisterFunction(NewReadFileFunction())
	RegisterFunction(NewSearchInDirectoryFunction())
	RegisterFunction(NewEditFileFunction())
	RegisterFunction(NewListFilesFunction())

	// 验证所有工具都已注册
	tools := GetAvailableTools()
	expectedTools := []string{"get_current_time", "read_file", "search_in_directory", "edit_file", "list_files"}

	if len(tools) != len(expectedTools) {
		t.Fatalf("Expected %d tools, got %d", len(expectedTools), len(tools))
	}

	// 测试每个工具
	t.Run("ListFiles", func(t *testing.T) {
		result, err := functionRegistry.Execute("list_files", `{"directory_path": "`+tempDir+`", "recursive": true}`)
		if err != nil {
			t.Fatalf("list_files failed: %v", err)
		}

		// 验证结果包含预期的文件
		if !strings.Contains(result, "main.go") {
			t.Error("Expected main.go in results")
		}
		if !strings.Contains(result, "config.yaml") {
			t.Error("Expected config.yaml in results")
		}
		if !strings.Contains(result, "subdir") {
			t.Error("Expected subdir in results")
		}
		t.Logf("ListFiles result:\n%s", result)
	})

	t.Run("ReadFile", func(t *testing.T) {
		mainFile := filepath.Join(tempDir, "main.go")
		result, err := functionRegistry.Execute("read_file", `{"path": "`+mainFile+`"}`)
		if err != nil {
			t.Fatalf("read_file failed: %v", err)
		}

		// 验证结果包含文件内容
		if !strings.Contains(result, "package main") {
			t.Error("Expected package main in file content")
		}
		if !strings.Contains(result, "fmt.Println") {
			t.Error("Expected fmt.Println in file content")
		}
		t.Logf("ReadFile result:\n%s", result)
	})

	t.Run("ReadFileWithRange", func(t *testing.T) {
		mainFile := filepath.Join(tempDir, "main.go")
		result, err := functionRegistry.Execute("read_file", `{"path": "`+mainFile+`", "start_line": 1, "end_line": 3}`)
		if err != nil {
			t.Fatalf("read_file with range failed: %v", err)
		}

		// 验证只返回指定行数
		lines := strings.Split(strings.TrimPrefix(result, "File content (2 lines):\n"), "\n")
		if len(lines) != 2 {
			t.Errorf("Expected 2 lines, got %d", len(lines))
		}
		t.Logf("ReadFileWithRange result:\n%s", result)
	})

	t.Run("SearchInDirectory", func(t *testing.T) {
		result, err := functionRegistry.Execute("search_in_directory", `{"directory": "`+tempDir+`", "pattern": "fmt\\.Println"}`)
		if err != nil {
			t.Fatalf("search_in_directory failed: %v", err)
		}

		// 验证找到匹配内容
		if !strings.Contains(result, "main.go") {
			t.Error("Expected to find match in main.go")
		}
		if !strings.Contains(result, "fmt.Println") {
			t.Error("Expected to find fmt.Println in search results")
		}
		t.Logf("SearchInDirectory result:\n%s", result)
	})

	t.Run("EditFile", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test_edit.txt")

		// 创建测试文件
		originalContent := "line 1\nline 2\nline 3\n"
		err := os.WriteFile(testFile, []byte(originalContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// 应用diff修改 - 使用JSON编码来正确处理换行符
		diff := "@@ -1,3 +1,3 @@\n line 1\n-line 2\n+modified line 2\n line 3"

		// 构造JSON参数
		args := map[string]interface{}{
			"file_path":    testFile,
			"diff_content": diff,
		}
		argsJSON, _ := json.Marshal(args)

		result, err := functionRegistry.Execute("edit_file", string(argsJSON))
		if err != nil {
			t.Fatalf("edit_file failed: %v", err)
		}

		// 验证文件被修改
		modifiedContent, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read modified file: %v", err)
		}

		expectedContent := "line 1\nmodified line 2\nline 3\n"
		if string(modifiedContent) != expectedContent {
			t.Errorf("File content not modified correctly.\nExpected: %q\nGot: %q", expectedContent, string(modifiedContent))
		}

		t.Logf("EditFile result: %s", result)
	})

	t.Run("GetCurrentTime", func(t *testing.T) {
		result, err := functionRegistry.Execute("get_current_time", `{}`)
		if err != nil {
			t.Fatalf("get_current_time failed: %v", err)
		}

		// 验证时间格式
		if !strings.Contains(result, "Current time:") {
			t.Error("Expected 'Current time:' in result")
		}

		// 测试自定义格式
		result2, err := functionRegistry.Execute("get_current_time", `{"format": "2006-01-02"}`)
		if err != nil {
			t.Fatalf("get_current_time with format failed: %v", err)
		}

		if !strings.Contains(result2, "2025") {
			t.Error("Expected year in custom format result")
		}

		t.Logf("GetCurrentTime results:\n%s\n%s", result, result2)
	})
}

// setupTestEnvironment 设置测试环境
func setupTestEnvironment(t *testing.T, tempDir string) {
	// 创建目录结构
	subDir := filepath.Join(tempDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	// 创建测试文件
	files := map[string]string{
		"main.go": `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
	fmt.Println("This is a test file")
}`,
		"config.yaml": `app:
  name: test-app
  version: 1.0.0
database:
  host: localhost
  port: 5432`,
		"subdir/helper.go": `package helper

import "fmt"

func Helper() {
	fmt.Println("Helper function")
}`,
		"README.md": `# Test Project

This is a test project for validating tool integration.

## Features
- File reading
- Directory searching  
- File editing
- Directory listing`,
	}

	for filename, content := range files {
		fullPath := filepath.Join(tempDir, filename)
		err := os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", filename, err)
		}
	}
}
