package tools

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// ResourceDirectory represents a managed resource directory
type ResourceDirectory struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
}

// ResourceManager manages resource directories for a specific working directory
type ResourceManager struct {
	mu          sync.RWMutex
	directories map[string]*ResourceDirectory
	workingDir  string
	uid         string // Unique ID for this manager instance
}

var (
	managersLock sync.RWMutex
	managers     = make(map[string]*ResourceManager)
)

// GetResourceManager returns the singleton ResourceManager for a specific working directory
func GetResourceManager(workingDir string) *ResourceManager {
	managersLock.RLock()
	if mgr, exists := managers[workingDir]; exists {
		managersLock.RUnlock()
		return mgr
	}
	managersLock.RUnlock()

	managersLock.Lock()
	defer managersLock.Unlock()

	// Double-check after acquiring write lock
	if mgr, exists := managers[workingDir]; exists {
		return mgr
	}

	mgr := &ResourceManager{
		directories: make(map[string]*ResourceDirectory),
		workingDir:  workingDir,
		uid:         generateUID(),
	}
	managers[workingDir] = mgr
	return mgr
}

// generateUID generates a 6-character random UID
func generateUID() string {
	bytes := make([]byte, 3)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// Add adds a new resource directory
func (rm *ResourceManager) Add(name, description string) (*ResourceDirectory, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if _, exists := rm.directories[name]; exists {
		return nil, fmt.Errorf("resource directory '%s' already exists", name)
	}

	relativePath := filepath.Join(".tmp", rm.uid, name)
	absolutePath := filepath.Join(rm.workingDir, relativePath)

	// Create directory, clear if exists
	if info, err := os.Stat(absolutePath); err == nil {
		if info.IsDir() {
			// Directory exists (collision), clear it
			os.RemoveAll(absolutePath)
		}
	}

	if err := os.MkdirAll(absolutePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	rd := &ResourceDirectory{
		Name:        name,
		Description: description,
		Path:        absolutePath,
	}

	rm.directories[name] = rd
	return rd, nil
}

// Remove removes a resource directory from the manager (does not delete files)
func (rm *ResourceManager) Remove(name string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if _, exists := rm.directories[name]; !exists {
		return fmt.Errorf("resource directory '%s' not found", name)
	}

	delete(rm.directories, name)
	return nil
}

// List returns all resource directories
func (rm *ResourceManager) List() []*ResourceDirectory {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	result := make([]*ResourceDirectory, 0, len(rm.directories))
	for _, rd := range rm.directories {
		result = append(result, rd)
	}

	// Sort by name for consistent output
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// GetUID returns the unique ID of this ResourceManager instance
func (rm *ResourceManager) GetUID() string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.uid
}

// ResourceDirectoryListTool - Tool to list all resource directories
type resourceDirectoryListTool struct {
	workingDir string
}

func NewResourceDirectoryListTool(workingDir string) BaseTool {
	return &resourceDirectoryListTool{workingDir: workingDir}
}

func (t *resourceDirectoryListTool) Name() string {
	return "resource_directory_list"
}

func (t *resourceDirectoryListTool) Info() ToolInfo {
	return ToolInfo{
		Name: "resource_directory_list",
		Description: `Lists all registered resource directories managed by the agent.

This tool displays all resource directories that have been created for organizing
agent resources such as documentation, images, data files, etc.

Returns a formatted list with:
- Name: The identifier used to reference the directory
- Description: What the directory is used for
- Path: The absolute file system path
- UID: The unique identifier for avoiding collisions`,
		Parameters: map[string]any{},
		Required:   []string{},
	}
}

func (t *resourceDirectoryListTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	mgr := GetResourceManager(t.workingDir)
	directories := mgr.List()

	if len(directories) == 0 {
		return NewTextResponse("No resource directories registered"), nil
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Resource Directories (%d) - Manager UID: %s\n", len(directories), mgr.GetUID()))
	output.WriteString(strings.Repeat("-", 60) + "\n")

	for _, rd := range directories {
		output.WriteString(fmt.Sprintf("Name: %s\n", rd.Name))
		output.WriteString(fmt.Sprintf("Description: %s\n", rd.Description))
		output.WriteString(fmt.Sprintf("Path: %s\n", rd.Path))
		output.WriteString(strings.Repeat("-", 60) + "\n")
	}

	return NewTextResponse(output.String()), nil
}

// ResourceDirectoryAddTool - Tool to add a new resource directory
type resourceDirectoryAddTool struct {
	workingDir string
}

type ResourceDirectoryAddParams struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func NewResourceDirectoryAddTool(workingDir string) BaseTool {
	return &resourceDirectoryAddTool{workingDir: workingDir}
}

func (t *resourceDirectoryAddTool) Name() string {
	return "resource_directory_add"
}

func (t *resourceDirectoryAddTool) Info() ToolInfo {
	return ToolInfo{
		Name: "resource_directory_add",
		Description: `Creates a new resource directory for organizing agent resources.

This tool creates a managed directory in the project's temporary space (.tmp)
with automatic UID generation to avoid naming conflicts.

The directory will be created at: .tmp/{uid}/{name}
where {uid} is a 6-character random identifier.

If a path collision occurs, the existing directory will be cleared.`,
		Parameters: map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "The name/identifier for the resource directory",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Description of what this directory will be used for",
			},
		},
		Required: []string{"name", "description"},
	}
}

func (t *resourceDirectoryAddTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params ResourceDirectoryAddParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("invalid parameters: %s", err)), nil
	}

	if params.Name == "" {
		return NewTextErrorResponse("name cannot be empty"), nil
	}

	if params.Description == "" {
		return NewTextErrorResponse("description cannot be empty"), nil
	}

	mgr := GetResourceManager(t.workingDir)
	rd, err := mgr.Add(params.Name, params.Description)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("failed to add resource directory: %s", err)), nil
	}

	output := fmt.Sprintf("Successfully created resource directory:\n"+
		"Name: %s\n"+
		"Description: %s\n"+
		"Path: %s\n"+
		"Manager UID: %s",
		rd.Name, rd.Description, rd.Path, mgr.GetUID())

	return NewTextResponse(output), nil
}

// ResourceDirectoryRemoveTool - Tool to remove a resource directory
type resourceDirectoryRemoveTool struct {
	workingDir string
}

type ResourceDirectoryRemoveParams struct {
	Name string `json:"name"`
}

func NewResourceDirectoryRemoveTool(workingDir string) BaseTool {
	return &resourceDirectoryRemoveTool{workingDir: workingDir}
}

func (t *resourceDirectoryRemoveTool) Name() string {
	return "resource_directory_remove"
}

func (t *resourceDirectoryRemoveTool) Info() ToolInfo {
	return ToolInfo{
		Name: "resource_directory_remove",
		Description: `Removes a resource directory from the registry.

This tool removes the directory from the resource manager's registry but
DOES NOT delete the actual files and directories on disk.

The files remain accessible at their original path if needed.`,
		Parameters: map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "The name of the resource directory to remove",
			},
		},
		Required: []string{"name"},
	}
}

func (t *resourceDirectoryRemoveTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params ResourceDirectoryRemoveParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("invalid parameters: %s", err)), nil
	}

	if params.Name == "" {
		return NewTextErrorResponse("name cannot be empty"), nil
	}

	mgr := GetResourceManager(t.workingDir)
	if err := mgr.Remove(params.Name); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("failed to remove resource directory: %s", err)), nil
	}

	output := fmt.Sprintf("Successfully removed resource directory '%s' from registry.\n"+
		"Note: Files and directories remain on disk.", params.Name)

	return NewTextResponse(output), nil
}