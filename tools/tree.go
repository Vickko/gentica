package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type TreeParams struct {
	Path     string   `json:"path"`
	Ignore   []string `json:"ignore"`
	MaxDepth *int     `json:"max_depth,omitempty"` // nil means default, 0 means unlimited
	DirsOnly bool     `json:"dirs_only"`
}

type TreeNode struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	Type     string      `json:"type"` // "file" or "directory"
	Children []*TreeNode `json:"children,omitempty"`
}

type TreeResponseMetadata struct {
	NumberOfFiles int  `json:"number_of_files"`
	Truncated     bool `json:"truncated"`
	DepthLimited  bool `json:"depth_limited"`
	ActualDepth   int  `json:"actual_depth"`
}

type treeTool struct {
	workingDir string
}

const (
	TreeToolName    = "tree"
	MaxTreeFiles    = 512
	DefaultMaxDepth = 4
	treeDescription = `Directory tree visualization tool that shows files and subdirectories in a hierarchical tree structure, helping you explore and understand whats in the directory.

WHEN TO USE THIS TOOL:
- Use when you need to explore the structure of a directory
- Helpful for understanding the organization of a project
- Good first step when getting familiar with a new codebase
- Use dirs_only option to see just the folder structure

HOW TO USE:
- Provide a path to list (defaults to current working directory)
- Optionally specify glob patterns to ignore
- Control depth with max_depth parameter (default: 4, use 0 for unlimited)
- Use dirs_only=true to show only directories

FEATURES:
- Displays a hierarchical view of files and directories
- Depth control: defaults to 4 levels deep (configurable)
- Directory-only mode: can show just folder structure
- Automatically skips hidden files/directories (starting with '.')
- Skips common system directories like __pycache__, node_modules, .git
- Can filter out files matching specific patterns

LIMITATIONS:
- Results are limited to 512 files
- Default depth is limited to 4 levels unless explicitly specified
- Very large directories will be truncated
- Does not show file sizes or permissions
- Use max_depth=0 to show full tree (may produce very large output)

WINDOWS NOTES:
- Hidden file detection uses Unix convention (files starting with '.')
- Windows-specific hidden files (with hidden attribute) are not automatically skipped
- Common Windows directories like System32, Program Files are not in default ignore list
- Path separators are handled automatically (both / and \ work)

TIPS:
- Use max_depth=1 or 2 for a quick overview of structure
- Use dirs_only=true when you only care about folder organization
- Use Glob tool for finding files by name patterns instead of browsing
- Use Grep tool for searching file contents
- Combine with other tools for more effective exploration`
)

func NewTreeTool(workingDir string) BaseTool {
	return &treeTool{
		workingDir: workingDir,
	}
}

func (t *treeTool) Name() string {
	return TreeToolName
}

func (t *treeTool) Info() ToolInfo {
	return ToolInfo{
		Name:        TreeToolName,
		Description: treeDescription,
		Parameters: map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The path to the directory to list (defaults to current working directory)",
			},
			"ignore": map[string]any{
				"type":        "array",
				"description": "List of glob patterns to ignore",
				"items": map[string]any{
					"type": "string",
				},
			},
			"max_depth": map[string]any{
				"type":        "integer",
				"description": "Maximum depth to traverse (default: 4, use 0 for unlimited)",
			},
			"dirs_only": map[string]any{
				"type":        "boolean",
				"description": "Show only directories, not files (default: false)",
			},
		},
		Required: []string{},
	}
}

func (t *treeTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params TreeParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("invalid parameters: %s", err)), nil
	}

	searchPath := params.Path
	if searchPath == "" {
		searchPath = t.workingDir
	}

	// Determine max depth and whether it's using default
	maxDepth := DefaultMaxDepth
	usingDefaultDepth := true
	if params.MaxDepth != nil {
		maxDepth = *params.MaxDepth
		usingDefaultDepth = false
	}

	// Expand home directory if needed
	if strings.HasPrefix(searchPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			searchPath = filepath.Join(homeDir, searchPath[2:])
		}
	}

	if !filepath.IsAbs(searchPath) {
		searchPath = filepath.Join(t.workingDir, searchPath)
	}

	// Check if path is a directory
	info, err := os.Stat(searchPath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewTextErrorResponse(fmt.Sprintf("path does not exist: %s", searchPath)), nil
		}
		return NewTextErrorResponse(fmt.Sprintf("error accessing path: %s", err)), nil
	}

	if !info.IsDir() {
		return NewTextErrorResponse(fmt.Sprintf("not a directory: %s", searchPath)), nil
	}

	output, metadata, err := ListDirectoryTree(searchPath, params.Ignore, maxDepth, params.DirsOnly, usingDefaultDepth)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("error listing directory: %s", err)), nil
	}

	return WithResponseMetadata(
		NewTextResponse(output),
		metadata,
	), nil
}

func ListDirectoryTree(searchPath string, ignore []string, maxDepth int, dirsOnly bool, usingDefaultDepth bool) (string, TreeResponseMetadata, error) {
	files, truncated, err := listDirectory(searchPath, ignore, MaxTreeFiles, dirsOnly)
	if err != nil {
		return "", TreeResponseMetadata{}, fmt.Errorf("error listing directory: %w", err)
	}

	// Handle empty directory
	if len(files) == 0 {
		output := fmt.Sprintf("- %s%c\n  (empty)\n", searchPath, filepath.Separator)
		return output, TreeResponseMetadata{NumberOfFiles: 0}, nil
	}

	tree := createFileTree(files, searchPath)

	// Calculate actual depth and check if depth limited
	actualDepth := calculateTreeDepth(tree)
	depthLimited := false
	if maxDepth > 0 && actualDepth > maxDepth {
		depthLimited = true
	}

	output := printTreeWithDepth(tree, searchPath, maxDepth)

	// Build warning messages
	var warnings []string
	// Only show depth warning if using default depth and actually limited
	if depthLimited && usingDefaultDepth {
		warnings = append(warnings, fmt.Sprintf("Output is limited to %d levels deep by default. Use max_depth=0 to see the full tree or specify a custom depth", maxDepth))
	}
	if truncated {
		warnings = append(warnings, fmt.Sprintf("There are more than %d files in the directory. Use a more specific path or filter files", MaxTreeFiles))
	}

	if len(warnings) > 0 {
		output = strings.Join(warnings, ". ") + ".\n\n" + output
	}

	metadata := TreeResponseMetadata{
		NumberOfFiles: len(files),
		Truncated:     truncated,
		DepthLimited:  depthLimited,
		ActualDepth:   actualDepth,
	}

	return output, metadata, nil
}

// listDirectory lists files in a directory, applying ignore patterns
func listDirectory(rootPath string, ignorePatterns []string, maxFiles int, dirsOnly bool) ([]string, bool, error) {
	var results []string
	truncated := false
	fileCount := 0

	// Default ignore patterns
	defaultIgnore := []string{
		".*",           // Hidden files
		"__pycache__",  // Python cache
		"node_modules", // Node modules
		".git",         // Git directory
	}

	allIgnorePatterns := append(defaultIgnore, ignorePatterns...)

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files/dirs with errors
		}

		// Skip the root directory itself
		if path == rootPath {
			return nil
		}

		// Skip files if dirsOnly is true
		if dirsOnly && !info.IsDir() {
			return nil
		}

		// Check ignore patterns
		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			relPath = path
		}
		baseName := filepath.Base(path)

		for _, pattern := range allIgnorePatterns {
			var matched bool

			// Handle recursive patterns like "test/**"
			if strings.Contains(pattern, "/**") {
				prefix := strings.TrimSuffix(pattern, "/**")
				if strings.HasPrefix(relPath, prefix+string(filepath.Separator)) || relPath == prefix {
					matched = true
				}
			} else if strings.Contains(pattern, "**") {
				// Handle general ** patterns
				matched, _ = filepath.Match(pattern, relPath)
				if !matched {
					matched, _ = filepath.Match(pattern, baseName)
				}
			} else {
				// Simple pattern matching on base name
				matched, _ = filepath.Match(pattern, baseName)
			}

			if matched {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Check if we've reached max files
		if fileCount >= maxFiles {
			truncated = true
			return filepath.SkipAll
		}

		fileCount++
		results = append(results, path)
		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return nil, false, err
	}

	// Sort the results
	sort.Strings(results)

	return results, truncated, nil
}

func createFileTree(sortedPaths []string, rootPath string) []*TreeNode {
	root := []*TreeNode{}
	pathMap := make(map[string]*TreeNode)

	for _, path := range sortedPaths {
		relativePath := strings.TrimPrefix(path, rootPath)
		parts := strings.Split(relativePath, string(filepath.Separator))
		currentPath := ""
		var parentPath string

		var cleanParts []string
		for _, part := range parts {
			if part != "" {
				cleanParts = append(cleanParts, part)
			}
		}
		parts = cleanParts

		if len(parts) == 0 {
			continue
		}

		for i, part := range parts {
			if currentPath == "" {
				currentPath = part
			} else {
				currentPath = filepath.Join(currentPath, part)
			}

			if _, exists := pathMap[currentPath]; exists {
				parentPath = currentPath
				continue
			}

			// Check if it's a directory by checking if it exists as a directory
			fullPath := filepath.Join(rootPath, currentPath)
			info, err := os.Stat(fullPath)
			isDir := err == nil && info.IsDir()

			nodeType := "file"
			if isDir {
				nodeType = "directory"
			}

			newNode := &TreeNode{
				Name:     part,
				Path:     currentPath,
				Type:     nodeType,
				Children: []*TreeNode{},
			}

			pathMap[currentPath] = newNode

			if i > 0 && parentPath != "" {
				if parent, ok := pathMap[parentPath]; ok {
					parent.Children = append(parent.Children, newNode)
				}
			} else {
				root = append(root, newNode)
			}

			parentPath = currentPath
		}
	}

	return root
}

func calculateTreeDepth(tree []*TreeNode) int {
	if len(tree) == 0 {
		return 0
	}
	maxDepth := 1
	for _, node := range tree {
		depth := calculateNodeDepth(node, 1)
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	return maxDepth
}

func calculateNodeDepth(node *TreeNode, currentDepth int) int {
	if node.Type != "directory" || len(node.Children) == 0 {
		return currentDepth
	}
	maxChildDepth := currentDepth
	for _, child := range node.Children {
		childDepth := calculateNodeDepth(child, currentDepth+1)
		if childDepth > maxChildDepth {
			maxChildDepth = childDepth
		}
	}
	return maxChildDepth
}

func printTreeWithDepth(tree []*TreeNode, rootPath string, maxDepth int) string {
	var result strings.Builder

	result.WriteString("- ")
	result.WriteString(rootPath)
	if len(rootPath) > 0 && rootPath[len(rootPath)-1] != filepath.Separator {
		result.WriteByte(filepath.Separator)
	}
	result.WriteByte('\n')

	for _, node := range tree {
		printNodeWithDepth(&result, node, 1, maxDepth)
	}

	return result.String()
}

func printNodeWithDepth(builder *strings.Builder, node *TreeNode, level int, maxDepth int) {
	// Check depth limit
	if maxDepth > 0 && level > maxDepth {
		return
	}

	indent := strings.Repeat("  ", level)

	nodeName := node.Name
	if node.Type == "directory" {
		nodeName = nodeName + string(filepath.Separator)
	}

	fmt.Fprintf(builder, "%s- %s\n", indent, nodeName)

	// Show children if it's a directory and we haven't reached max depth
	if node.Type == "directory" && len(node.Children) > 0 {
		if maxDepth == 0 || level < maxDepth {
			for _, child := range node.Children {
				printNodeWithDepth(builder, child, level+1, maxDepth)
			}
		} else if level == maxDepth && len(node.Children) > 0 {
			// Show indicator that there are more levels
			fmt.Fprintf(builder, "%s  ...\n", indent)
		}
	}
}

// Keep old functions for backward compatibility
func printTree(tree []*TreeNode, rootPath string) string {
	return printTreeWithDepth(tree, rootPath, 0)
}

func printNode(builder *strings.Builder, node *TreeNode, level int) {
	printNodeWithDepth(builder, node, level, 0)
}
