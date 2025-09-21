package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResourceManager_Singleton(t *testing.T) {
	// Test that same workingDir returns same instance
	dir1 := t.TempDir()
	mgr1 := GetResourceManager(dir1)
	mgr2 := GetResourceManager(dir1)

	if mgr1 != mgr2 {
		t.Error("Expected same ResourceManager instance for same workingDir")
	}

	// Verify same UID for same instance
	if mgr1.GetUID() != mgr2.GetUID() {
		t.Error("Expected same UID for same ResourceManager instance")
	}

	// Test that different workingDir returns different instances
	dir2 := t.TempDir()
	mgr3 := GetResourceManager(dir2)

	if mgr1 == mgr3 {
		t.Error("Expected different ResourceManager instances for different workingDirs")
	}

	// Verify different UIDs for different instances
	if mgr1.GetUID() == mgr3.GetUID() {
		t.Logf("UID collision detected: %s (this is rare but possible)", mgr1.GetUID())
	}
}

func TestResourceManager_Add(t *testing.T) {
	workingDir := t.TempDir()
	mgr := GetResourceManager(workingDir)

	// Verify manager has a UID
	managerUID := mgr.GetUID()
	if len(managerUID) != 6 {
		t.Errorf("Expected manager UID length 6, got %d", len(managerUID))
	}

	// Test adding a new resource directory
	rd, err := mgr.Add("test-docs", "Test documentation files")
	if err != nil {
		t.Fatalf("Failed to add resource directory: %v", err)
	}

	// Verify properties
	if rd.Name != "test-docs" {
		t.Errorf("Expected name 'test-docs', got '%s'", rd.Name)
	}
	if rd.Description != "Test documentation files" {
		t.Errorf("Expected description 'Test documentation files', got '%s'", rd.Description)
	}

	// Verify path format uses manager's UID
	expectedPathPrefix := filepath.Join(workingDir, ".tmp", managerUID, "test-docs")
	if rd.Path != expectedPathPrefix {
		t.Errorf("Expected path '%s', got '%s'", expectedPathPrefix, rd.Path)
	}

	// Verify directory was created
	if _, err := os.Stat(rd.Path); os.IsNotExist(err) {
		t.Error("Directory was not created")
	}

	// Test adding duplicate name
	_, err = mgr.Add("test-docs", "Another description")
	if err == nil {
		t.Error("Expected error when adding duplicate name")
	}

	// Test adding another directory uses same UID
	rd2, err := mgr.Add("test-images", "Test images")
	if err != nil {
		t.Fatalf("Failed to add second resource directory: %v", err)
	}

	// Verify second directory uses same manager UID
	expectedPath2 := filepath.Join(workingDir, ".tmp", managerUID, "test-images")
	if rd2.Path != expectedPath2 {
		t.Errorf("Expected second directory path '%s', got '%s'", expectedPath2, rd2.Path)
	}
}

func TestResourceManager_Remove(t *testing.T) {
	workingDir := t.TempDir()
	mgr := GetResourceManager(workingDir)

	// Add a resource directory
	rd, _ := mgr.Add("test-resource", "Test resource")
	dirPath := rd.Path

	// Create a test file in the directory
	testFile := filepath.Join(dirPath, "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	// Remove from manager
	err := mgr.Remove("test-resource")
	if err != nil {
		t.Fatalf("Failed to remove resource directory: %v", err)
	}

	// Verify it's removed from manager
	directories := mgr.List()
	for _, d := range directories {
		if d.Name == "test-resource" {
			t.Error("Resource directory still in manager after removal")
		}
	}

	// Verify files still exist
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("Files were deleted but should remain")
	}

	// Test removing non-existent
	err = mgr.Remove("non-existent")
	if err == nil {
		t.Error("Expected error when removing non-existent resource")
	}
}

func TestResourceManager_List(t *testing.T) {
	workingDir := t.TempDir()
	mgr := GetResourceManager(workingDir)

	// Initially empty
	directories := mgr.List()
	if len(directories) != 0 {
		t.Error("Expected empty list initially")
	}

	// Add multiple directories
	mgr.Add("docs", "Documentation")
	mgr.Add("images", "Image files")
	mgr.Add("data", "Data files")

	directories = mgr.List()
	if len(directories) != 3 {
		t.Errorf("Expected 3 directories, got %d", len(directories))
	}

	// Verify sorted order
	if directories[0].Name != "data" || directories[1].Name != "docs" || directories[2].Name != "images" {
		t.Error("Directories not sorted alphabetically")
	}
}

func TestResourceManager_Isolation(t *testing.T) {
	// Create two different working directories
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	mgr1 := GetResourceManager(dir1)
	mgr2 := GetResourceManager(dir2)

	// Add to mgr1
	mgr1.Add("resource1", "Resource in mgr1")

	// Add to mgr2
	mgr2.Add("resource2", "Resource in mgr2")

	// Verify isolation
	list1 := mgr1.List()
	list2 := mgr2.List()

	if len(list1) != 1 || list1[0].Name != "resource1" {
		t.Error("mgr1 should only have resource1")
	}

	if len(list2) != 1 || list2[0].Name != "resource2" {
		t.Error("mgr2 should only have resource2")
	}
}

func TestResourceManager_CollisionHandling(t *testing.T) {
	workingDir := t.TempDir()
	mgr := GetResourceManager(workingDir)

	// Manually create a directory that will collide
	uid := "abc123"
	collidingPath := filepath.Join(workingDir, ".tmp", uid, "testdir")
	os.MkdirAll(collidingPath, 0755)

	// Create a file in the colliding directory
	testFile := filepath.Join(collidingPath, "existing.txt")
	os.WriteFile(testFile, []byte("existing content"), 0644)

	// Mock the UID generation to force collision (this is theoretical,
	// in practice we'd need to patch generateUID)
	// For now, we just verify the directory creation works even if path exists

	rd, err := mgr.Add("testdir", "Test directory")
	if err != nil {
		t.Fatalf("Failed to add resource directory: %v", err)
	}

	// Verify directory was created (with a different UID)
	if _, err := os.Stat(rd.Path); os.IsNotExist(err) {
		t.Error("Directory was not created")
	}
}

func TestResourceDirectoryListTool(t *testing.T) {
	workingDir := t.TempDir()
	tool := NewResourceDirectoryListTool(workingDir)

	// Test tool info
	info := tool.Info()
	if info.Name != "resource_directory_list" {
		t.Errorf("Expected name 'resource_directory_list', got '%s'", info.Name)
	}

	// Add some directories
	mgr := GetResourceManager(workingDir)
	mgr.Add("docs", "Documentation")
	mgr.Add("images", "Image files")

	// Run the tool
	call := ToolCall{
		Name:  "resource_directory_list",
		Input: "{}",
	}

	response, err := tool.Run(context.Background(), call)
	if err != nil {
		t.Fatalf("Tool execution failed: %v", err)
	}

	if response.IsError {
		t.Errorf("Expected success response, got error: %s", response.Content)
	}

	// Verify output contains expected directories
	if !strings.Contains(response.Content, "docs") {
		t.Error("Output should contain 'docs'")
	}
	if !strings.Contains(response.Content, "images") {
		t.Error("Output should contain 'images'")
	}
	if !strings.Contains(response.Content, "Resource Directories (2)") {
		t.Error("Output should show count of 2")
	}
	// Verify manager UID is shown
	if !strings.Contains(response.Content, "Manager UID:") {
		t.Error("Output should show Manager UID")
	}
}

func TestResourceDirectoryAddTool(t *testing.T) {
	workingDir := t.TempDir()
	tool := NewResourceDirectoryAddTool(workingDir)

	// Test tool info
	info := tool.Info()
	if info.Name != "resource_directory_add" {
		t.Errorf("Expected name 'resource_directory_add', got '%s'", info.Name)
	}

	// Test adding a directory
	params := ResourceDirectoryAddParams{
		Name:        "test-data",
		Description: "Test data files",
	}
	paramsJSON, _ := json.Marshal(params)

	call := ToolCall{
		Name:  "resource_directory_add",
		Input: string(paramsJSON),
	}

	response, err := tool.Run(context.Background(), call)
	if err != nil {
		t.Fatalf("Tool execution failed: %v", err)
	}

	if response.IsError {
		t.Errorf("Expected success response, got error: %s", response.Content)
	}

	// Verify output
	if !strings.Contains(response.Content, "Successfully created") {
		t.Error("Output should contain success message")
	}
	if !strings.Contains(response.Content, "test-data") {
		t.Error("Output should contain directory name")
	}

	// Verify directory was actually created
	mgr := GetResourceManager(workingDir)
	directories := mgr.List()
	found := false
	for _, d := range directories {
		if d.Name == "test-data" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Directory not found in manager")
	}

	// Test with empty name
	invalidParams := ResourceDirectoryAddParams{
		Name:        "",
		Description: "Test",
	}
	paramsJSON, _ = json.Marshal(invalidParams)
	call.Input = string(paramsJSON)

	response, err = tool.Run(context.Background(), call)
	if err != nil {
		t.Fatalf("Tool execution failed: %v", err)
	}

	if !response.IsError {
		t.Error("Expected error response for empty name")
	}
}

func TestResourceDirectoryRemoveTool(t *testing.T) {
	workingDir := t.TempDir()
	mgr := GetResourceManager(workingDir)
	tool := NewResourceDirectoryRemoveTool(workingDir)

	// Test tool info
	info := tool.Info()
	if info.Name != "resource_directory_remove" {
		t.Errorf("Expected name 'resource_directory_remove', got '%s'", info.Name)
	}

	// Add a directory first
	rd, _ := mgr.Add("to-remove", "Directory to remove")

	// Create a file in it
	testFile := filepath.Join(rd.Path, "keep-me.txt")
	os.WriteFile(testFile, []byte("content"), 0644)

	// Remove using tool
	params := ResourceDirectoryRemoveParams{
		Name: "to-remove",
	}
	paramsJSON, _ := json.Marshal(params)

	call := ToolCall{
		Name:  "resource_directory_remove",
		Input: string(paramsJSON),
	}

	response, err := tool.Run(context.Background(), call)
	if err != nil {
		t.Fatalf("Tool execution failed: %v", err)
	}

	if response.IsError {
		t.Errorf("Expected success response, got error: %s", response.Content)
	}

	// Verify output
	if !strings.Contains(response.Content, "Successfully removed") {
		t.Error("Output should contain success message")
	}
	if !strings.Contains(response.Content, "Files and directories remain") {
		t.Error("Output should mention files remain")
	}

	// Verify removed from manager
	directories := mgr.List()
	for _, d := range directories {
		if d.Name == "to-remove" {
			t.Error("Directory should be removed from manager")
		}
	}

	// Verify files still exist
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("File should still exist after removal")
	}

	// Test removing non-existent
	params = ResourceDirectoryRemoveParams{
		Name: "non-existent",
	}
	paramsJSON, _ = json.Marshal(params)
	call.Input = string(paramsJSON)

	response, err = tool.Run(context.Background(), call)
	if err != nil {
		t.Fatalf("Tool execution failed: %v", err)
	}

	if !response.IsError {
		t.Error("Expected error response for non-existent directory")
	}
}

func TestGenerateUID(t *testing.T) {
	// Test UID generation
	seen := make(map[string]bool)

	for i := 0; i < 100; i++ {
		uid := generateUID()

		// Check length
		if len(uid) != 6 {
			t.Errorf("Expected UID length 6, got %d", len(uid))
		}

		// Check uniqueness (probabilistic)
		if seen[uid] {
			t.Logf("UID collision detected: %s (this is rare but possible)", uid)
		}
		seen[uid] = true

		// Check format (hexadecimal)
		for _, c := range uid {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("Invalid character in UID: %c", c)
			}
		}
	}
}