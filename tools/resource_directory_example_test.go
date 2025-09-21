package tools

import (
	"fmt"
	"testing"
)

func ExampleResourceManager_singleUID() {
	// This example demonstrates that all resource directories
	// under the same ResourceManager share a single UID

	workingDir := "/tmp/example"
	mgr := GetResourceManager(workingDir)

	// Get the manager's UID
	managerUID := mgr.GetUID()
	fmt.Printf("Manager UID: %s (all directories will use this)\n", managerUID)

	// Add multiple resource directories
	mgr.Add("docs", "Documentation files")
	mgr.Add("images", "Image assets")
	mgr.Add("data", "Data files")

	// List all directories - they all share the same UID path prefix
	directories := mgr.List()
	for _, dir := range directories {
		fmt.Printf("- %s: %s\n", dir.Name, dir.Path)
		// Path format: /tmp/example/.tmp/{managerUID}/{name}
	}

	// Output format (UID will vary):
	// Manager UID: abc123 (all directories will use this)
	// - data: /tmp/example/.tmp/abc123/data
	// - docs: /tmp/example/.tmp/abc123/docs
	// - images: /tmp/example/.tmp/abc123/images
}

func TestResourceManager_SingleUID(t *testing.T) {
	// This test verifies that all directories under the same manager
	// share the same UID in their paths

	workingDir := t.TempDir()
	mgr := GetResourceManager(workingDir)
	managerUID := mgr.GetUID()

	// Add multiple directories
	dirs := []string{"docs", "images", "data", "config", "cache"}
	for _, name := range dirs {
		rd, err := mgr.Add(name, fmt.Sprintf("%s directory", name))
		if err != nil {
			t.Fatalf("Failed to add directory %s: %v", name, err)
		}

		// Verify each directory uses the manager's UID
		expectedPath := fmt.Sprintf("%s/.tmp/%s/%s", workingDir, managerUID, name)
		if rd.Path != expectedPath {
			t.Errorf("Directory %s has wrong path.\nExpected: %s\nGot: %s",
				name, expectedPath, rd.Path)
		}
	}

	// Verify all directories are under the same UID directory
	allDirs := mgr.List()
	if len(allDirs) != len(dirs) {
		t.Errorf("Expected %d directories, got %d", len(dirs), len(allDirs))
	}

	for _, rd := range allDirs {
		// All paths should contain the same manager UID
		expectedPrefix := fmt.Sprintf("%s/.tmp/%s/", workingDir, managerUID)
		if len(rd.Path) < len(expectedPrefix) || rd.Path[:len(expectedPrefix)] != expectedPrefix {
			t.Errorf("Directory %s doesn't have expected prefix.\nExpected prefix: %s\nGot path: %s",
				rd.Name, expectedPrefix, rd.Path)
		}
	}
}