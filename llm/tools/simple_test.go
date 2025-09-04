package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Simple test without external dependencies
func TestBasicTools(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("TestBashTool_Echo", func(t *testing.T) {
		bashTool := NewBashTool(tempDir)
		params := BashParams{
			Command: "echo 'hello world'",
		}
		paramsJSON, err := json.Marshal(params)
		if err != nil {
			t.Fatalf("Failed to marshal params: %v", err)
		}

		call := ToolCall{Input: string(paramsJSON)}
		response, err := bashTool.Run(context.Background(), call)
		if err != nil {
			t.Fatalf("Failed to run bash command: %v", err)
		}
		if response.IsError {
			t.Fatalf("Command failed: %s", response.Content)
		}
		if !strings.Contains(response.Content, "hello world") {
			t.Errorf("Expected 'hello world' in output, got: %s", response.Content)
		}
	})

	t.Run("TestWriteTool_CreateFile", func(t *testing.T) {
		writeTool := NewWriteTool(tempDir)
		filePath := filepath.Join(tempDir, "test.txt")
		content := "Test content"

		params := WriteParams{
			FilePath: filePath,
			Content:  content,
		}
		paramsJSON, err := json.Marshal(params)
		if err != nil {
			t.Fatalf("Failed to marshal params: %v", err)
		}

		call := ToolCall{Input: string(paramsJSON)}
		response, err := writeTool.Run(context.Background(), call)
		if err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
		if response.IsError {
			t.Fatalf("Write failed: %s", response.Content)
		}

		// Verify file was created
		data, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read created file: %v", err)
		}
		if string(data) != content {
			t.Errorf("File content mismatch. Expected: %s, Got: %s", content, string(data))
		}
	})

	t.Run("TestViewTool_ReadFile", func(t *testing.T) {
		// First create a file
		testFile := filepath.Join(tempDir, "view_test.txt")
		testContent := "Line 1\nLine 2\nLine 3"
		err := os.WriteFile(testFile, []byte(testContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		viewTool := NewViewTool(tempDir)
		params := ViewParams{
			FilePath: testFile,
		}
		paramsJSON, err := json.Marshal(params)
		if err != nil {
			t.Fatalf("Failed to marshal params: %v", err)
		}

		call := ToolCall{Input: string(paramsJSON)}
		response, err := viewTool.Run(context.Background(), call)
		if err != nil {
			t.Fatalf("Failed to view file: %v", err)
		}
		if response.IsError {
			t.Fatalf("View failed: %s", response.Content)
		}
		if !strings.Contains(response.Content, "Line 1") {
			t.Errorf("Expected 'Line 1' in output, got: %s", response.Content)
		}
	})

	t.Run("TestEditTool_CreateNewFile", func(t *testing.T) {
		editTool := NewEditTool(tempDir)
		filePath := filepath.Join(tempDir, "edit_test.txt")

		params := EditParams{
			FilePath:  filePath,
			OldString: "",
			NewString: "New file content",
		}
		paramsJSON, err := json.Marshal(params)
		if err != nil {
			t.Fatalf("Failed to marshal params: %v", err)
		}

		call := ToolCall{Input: string(paramsJSON)}
		response, err := editTool.Run(context.Background(), call)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
		if response.IsError {
			t.Fatalf("Edit failed: %s", response.Content)
		}

		// Verify file was created
		data, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read created file: %v", err)
		}
		if string(data) != "New file content" {
			t.Errorf("File content mismatch. Expected: %s, Got: %s", "New file content", string(data))
		}
	})

	t.Run("TestGlobTool_FindFiles", func(t *testing.T) {
		// Create some test files
		os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(tempDir, "file2.txt"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(tempDir, "file3.go"), []byte("test"), 0644)

		globTool := NewGlobTool(tempDir)
		params := GlobParams{
			Pattern: "*.txt",
			Path:    tempDir,
		}
		paramsJSON, err := json.Marshal(params)
		if err != nil {
			t.Fatalf("Failed to marshal params: %v", err)
		}

		call := ToolCall{Input: string(paramsJSON)}
		response, err := globTool.Run(context.Background(), call)
		if err != nil {
			t.Fatalf("Failed to glob files: %v", err)
		}
		if response.IsError {
			t.Fatalf("Glob failed: %s", response.Content)
		}
		if !strings.Contains(response.Content, "file1.txt") {
			t.Errorf("Expected 'file1.txt' in output, got: %s", response.Content)
		}
		if !strings.Contains(response.Content, "file2.txt") {
			t.Errorf("Expected 'file2.txt' in output, got: %s", response.Content)
		}
		if strings.Contains(response.Content, "file3.go") {
			t.Errorf("Did not expect 'file3.go' in output, got: %s", response.Content)
		}
	})

	t.Run("TestLsTool_ListDirectory", func(t *testing.T) {
		// Create test structure
		os.Mkdir(filepath.Join(tempDir, "testdir"), 0755)
		os.WriteFile(filepath.Join(tempDir, "testfile.txt"), []byte("test"), 0644)

		lsTool := NewLsTool(tempDir)
		params := LSParams{
			Path: tempDir,
		}
		paramsJSON, err := json.Marshal(params)
		if err != nil {
			t.Fatalf("Failed to marshal params: %v", err)
		}

		call := ToolCall{Input: string(paramsJSON)}
		response, err := lsTool.Run(context.Background(), call)
		if err != nil {
			t.Fatalf("Failed to list directory: %v", err)
		}
		if response.IsError {
			t.Fatalf("LS failed: %s", response.Content)
		}
		if !strings.Contains(response.Content, "testdir") {
			t.Errorf("Expected 'testdir' in output, got: %s", response.Content)
		}
		if !strings.Contains(response.Content, "testfile.txt") {
			t.Errorf("Expected 'testfile.txt' in output, got: %s", response.Content)
		}
	})
}