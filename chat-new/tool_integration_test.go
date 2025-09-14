package chat

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gentica/llm/tools"
)

func TestToolAdapterConversion(t *testing.T) {
	// Get current working directory
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Test with ViewTool as an example
	viewTool := tools.NewViewTool(workingDir)
	adapter := NewToolAdapter(viewTool)
	function := adapter.ConvertToFunction()

	// Check function definition
	if function.Definition.Name != "view" {
		t.Errorf("Expected tool name 'view', got %s", function.Definition.Name)
	}

	if function.Definition.Description == "" {
		t.Error("Tool description should not be empty")
	}

	// Check parameters conversion
	params, ok := function.Definition.Parameters.(map[string]interface{})
	if !ok {
		t.Fatal("Parameters should be a map")
	}

	if params["type"] != "object" {
		t.Error("Parameters should have type 'object'")
	}

	properties, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Parameters should have properties")
	}

	// Check that file_path parameter exists
	if _, exists := properties["file_path"]; !exists {
		t.Error("Missing file_path parameter")
	}
}

func TestToolHandlerExecution(t *testing.T) {
	// Create a temporary test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World!\nThis is a test file."

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create ViewTool adapter
	viewTool := tools.NewViewTool(tempDir)
	adapter := NewToolAdapter(viewTool)
	function := adapter.ConvertToFunction()

	// Test the handler
	args := map[string]interface{}{
		"file_path": testFile,
	}

	result, err := function.Handler(args)
	if err != nil {
		t.Fatalf("Handler failed: %v", err)
	}

	// Check that the result contains the file content
	if !strings.Contains(result, "Hello, World!") {
		t.Errorf("Result should contain file content, got: %s", result)
	}
}

func TestLsToolIntegration(t *testing.T) {
	// Create a temporary test directory structure
	tempDir := t.TempDir()

	// Create some test files and directories
	os.MkdirAll(filepath.Join(tempDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tempDir, "file2.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tempDir, "subdir", "file3.txt"), []byte("test"), 0644)

	// Create LsTool adapter
	lsTool := tools.NewLsTool(tempDir)
	adapter := NewToolAdapter(lsTool)
	function := adapter.ConvertToFunction()

	// Test the handler
	args := map[string]interface{}{
		"path": tempDir,
	}

	result, err := function.Handler(args)
	if err != nil {
		t.Fatalf("Handler failed: %v", err)
	}

	// Check that the result contains expected files
	if !strings.Contains(result, "file1.txt") {
		t.Errorf("Result should contain file1.txt")
	}
	if !strings.Contains(result, "file2.go") {
		t.Errorf("Result should contain file2.go")
	}
	if !strings.Contains(result, "subdir") {
		t.Errorf("Result should contain subdir")
	}
}

func TestBashToolIntegration(t *testing.T) {
	// Get current working directory
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Create BashTool adapter
	bashTool := tools.NewBashTool(workingDir)
	adapter := NewToolAdapter(bashTool)
	function := adapter.ConvertToFunction()

	// Test a simple echo command
	args := map[string]interface{}{
		"command": "echo 'Hello from bash'",
	}

	result, err := function.Handler(args)
	if err != nil {
		t.Fatalf("Handler failed: %v", err)
	}

	// Check the output
	if !strings.Contains(result, "Hello from bash") {
		t.Errorf("Expected 'Hello from bash' in output, got: %s", result)
	}
}

func TestWriteToolIntegration(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "output.txt")

	// Create WriteTool adapter
	writeTool := tools.NewWriteTool(tempDir)
	adapter := NewToolAdapter(writeTool)
	function := adapter.ConvertToFunction()

	// Test writing a file
	testContent := "This is test content\nLine 2\nLine 3"
	args := map[string]interface{}{
		"file_path": testFile,
		"content":   testContent,
	}

	result, err := function.Handler(args)
	if err != nil {
		t.Fatalf("Handler failed: %v", err)
	}

	// Verify the file was created
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("File content mismatch. Expected:\n%s\nGot:\n%s", testContent, string(content))
	}

	// Check success message
	if !strings.Contains(result, "successfully wrote") {
		t.Errorf("Expected success message, got: %s", result)
	}
}

func TestErrorHandling(t *testing.T) {
	// Test with ViewTool trying to read a non-existent file
	tempDir := t.TempDir()
	viewTool := tools.NewViewTool(tempDir)
	adapter := NewToolAdapter(viewTool)
	function := adapter.ConvertToFunction()

	args := map[string]interface{}{
		"file_path": "/nonexistent/file.txt",
	}

	_, err := function.Handler(args)
	if err == nil {
		t.Error("Expected an error for non-existent file")
	}
}

func TestRegistryIntegration(t *testing.T) {
	// Create a new function registry
	registry := NewFunctionRegistry()

	// Register all LLM tools
	tempDir := t.TempDir()
	RegisterLLMTools(registry, tempDir)

	// Check that tools are registered
	expectedTools := []string{
		"bash", "view", "write", "edit", "multiedit",
		"grep", "glob", "ls", "fetch", "download",
	}

	for _, toolName := range expectedTools {
		if _, exists := registry.GetFunction(toolName); !exists {
			t.Errorf("Tool %s should be registered", toolName)
		}
	}

	// Get all tools and verify count
	tools := registry.GetTools()
	if len(tools) < len(expectedTools) {
		t.Errorf("Expected at least %d tools, got %d", len(expectedTools), len(tools))
	}
}

func TestJSONParameterMarshaling(t *testing.T) {
	// Test that complex parameters are correctly marshaled to JSON
	tempDir := t.TempDir()
	grepTool := tools.NewGrepTool(tempDir)
	adapter := NewToolAdapter(grepTool)
	function := adapter.ConvertToFunction()

	// Complex args with different types
	args := map[string]interface{}{
		"pattern":      "test.*pattern",
		"path":         tempDir,
		"include":      "*.go",
		"literal_text": true,
	}

	// This should not panic and should marshal correctly
	_, err := function.Handler(args)
	// We expect an error since there are no files, but marshaling should work
	if err != nil && strings.Contains(err.Error(), "marshal") {
		t.Errorf("JSON marshaling failed: %v", err)
	}
}

func TestToolCallIDGeneration(t *testing.T) {
	// Ensure tool call IDs are unique
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateToolCallID()
		if ids[id] {
			t.Errorf("Duplicate tool call ID generated: %s", id)
		}
		ids[id] = true
	}
}
