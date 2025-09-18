package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLsTool(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	lsTool := NewLsTool(tempDir)

	// Create test file structure
	testStructure := map[string]bool{ // true for directories, false for files
		"file1.txt":          false,
		"file2.txt":          false,
		"dir1":               true,
		"dir2":               true,
		".hidden_file":       false,
		".hidden_dir":        true,
		"README.md":          false,
		"config.json":        false,
		"src":                true,
		"src/index.js":       false, // This shouldn't appear in ls of root
		"test":               true,
		"test/test.spec.js":  false, // This shouldn't appear in ls of root
	}

	for path, isDir := range testStructure {
		fullPath := filepath.Join(tempDir, path)
		if isDir {
			err := os.MkdirAll(fullPath, 0o755)
			require.NoError(t, err)
		} else {
			err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
			require.NoError(t, err)
			err = os.WriteFile(fullPath, []byte("test content"), 0o644)
			require.NoError(t, err)
		}
	}

	t.Run("ListDirectoryDefault", func(t *testing.T) {
		params := LsParams{
			Path: tempDir,
			All:  false,
		}
		input, err := json.Marshal(params)
		require.NoError(t, err)

		response, err := lsTool.Run(context.Background(), ToolCall{
			Name:  "ls",
			Input: string(input),
		})

		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Equal(t, ToolResponseTypeText, response.Type)

		// Check that output contains visible items
		output := response.Content
		assert.Contains(t, output, "file1.txt")
		assert.Contains(t, output, "file2.txt")
		assert.Contains(t, output, "dir1/")
		assert.Contains(t, output, "dir2/")
		assert.Contains(t, output, "README.md")
		assert.Contains(t, output, "config.json")
		assert.Contains(t, output, "src/")
		assert.Contains(t, output, "test/")

		// Check that hidden items are not shown
		assert.NotContains(t, output, ".hidden_file")
		assert.NotContains(t, output, ".hidden_dir")

		// Check that subdirectory contents are not shown
		assert.NotContains(t, output, "index.js")
		assert.NotContains(t, output, "test.spec.js")

		// Parse metadata
		var metadata LsResponseMetadata
		if response.Metadata != "" {
			err = json.Unmarshal([]byte(response.Metadata), &metadata)
			require.NoError(t, err)
			assert.Equal(t, 8, metadata.NumberOfFiles) // 4 files + 4 directories (no hidden)
			assert.False(t, metadata.Truncated)
		}
	})

	t.Run("ListDirectoryWithHidden", func(t *testing.T) {
		params := LsParams{
			Path: tempDir,
			All:  true,
		}
		input, err := json.Marshal(params)
		require.NoError(t, err)

		response, err := lsTool.Run(context.Background(), ToolCall{
			Name:  "ls",
			Input: string(input),
		})

		require.NoError(t, err)
		assert.False(t, response.IsError)

		// Check that hidden items are shown
		output := response.Content
		assert.Contains(t, output, ".hidden_file")
		assert.Contains(t, output, ".hidden_dir/")

		// Parse metadata
		var metadata LsResponseMetadata
		if response.Metadata != "" {
			err = json.Unmarshal([]byte(response.Metadata), &metadata)
			require.NoError(t, err)
			assert.Equal(t, 10, metadata.NumberOfFiles) // All items including hidden
			assert.False(t, metadata.Truncated)
		}
	})

	t.Run("EmptyDirectory", func(t *testing.T) {
		emptyDir := filepath.Join(tempDir, "empty")
		err := os.MkdirAll(emptyDir, 0o755)
		require.NoError(t, err)

		params := LsParams{
			Path: emptyDir,
			All:  false,
		}
		input, err := json.Marshal(params)
		require.NoError(t, err)

		response, err := lsTool.Run(context.Background(), ToolCall{
			Name:  "ls",
			Input: string(input),
		})

		require.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "(empty)")

		// Parse metadata
		var metadata LsResponseMetadata
		if response.Metadata != "" {
			err = json.Unmarshal([]byte(response.Metadata), &metadata)
			require.NoError(t, err)
			assert.Equal(t, 0, metadata.NumberOfFiles)
			assert.False(t, metadata.Truncated)
		}
	})

	t.Run("NonExistentPath", func(t *testing.T) {
		params := LsParams{
			Path: filepath.Join(tempDir, "nonexistent"),
			All:  false,
		}
		input, err := json.Marshal(params)
		require.NoError(t, err)

		response, err := lsTool.Run(context.Background(), ToolCall{
			Name:  "ls",
			Input: string(input),
		})

		require.NoError(t, err)
		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "path does not exist")
	})

	t.Run("FileInsteadOfDirectory", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "file1.txt")
		params := LsParams{
			Path: filePath,
			All:  false,
		}
		input, err := json.Marshal(params)
		require.NoError(t, err)

		response, err := lsTool.Run(context.Background(), ToolCall{
			Name:  "ls",
			Input: string(input),
		})

		require.NoError(t, err)
		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "not a directory")
	})

	t.Run("DefaultToWorkingDirectory", func(t *testing.T) {
		params := LsParams{
			// Path is empty, should default to working directory
			All: false,
		}
		input, err := json.Marshal(params)
		require.NoError(t, err)

		response, err := lsTool.Run(context.Background(), ToolCall{
			Name:  "ls",
			Input: string(input),
		})

		require.NoError(t, err)
		assert.False(t, response.IsError)
		// Should list contents of tempDir (the working directory)
		assert.Contains(t, response.Content, "file1.txt")
		assert.Contains(t, response.Content, "dir1/")
	})

	t.Run("AlphabeticalSorting", func(t *testing.T) {
		// Create a fresh directory for this test to avoid interference
		sortTestDir := filepath.Join(tempDir, "sorttest")
		err := os.MkdirAll(sortTestDir, 0o755)
		require.NoError(t, err)

		// Create files in non-alphabetical order
		testFiles := []string{"zebra.txt", "apple.txt", "banana.txt", "cherry.txt"}
		for _, name := range testFiles {
			err := os.WriteFile(filepath.Join(sortTestDir, name), []byte(""), 0o644)
			require.NoError(t, err)
		}
		// Create directories
		testDirs := []string{"zoo", "alpha", "beta"}
		for _, name := range testDirs {
			err := os.MkdirAll(filepath.Join(sortTestDir, name), 0o755)
			require.NoError(t, err)
		}

		params := LsParams{
			Path: sortTestDir,
			All:  false,
		}
		input, err := json.Marshal(params)
		require.NoError(t, err)

		response, err := lsTool.Run(context.Background(), ToolCall{
			Name:  "ls",
			Input: string(input),
		})

		require.NoError(t, err)
		assert.False(t, response.IsError)

		// Check that items appear in alphabetical order
		lines := strings.Split(strings.TrimSpace(response.Content), "\n")
		expectedOrder := []string{
			"alpha/",
			"apple.txt",
			"banana.txt",
			"beta/",
			"cherry.txt",
			"zebra.txt",
			"zoo/",
		}
		assert.Equal(t, expectedOrder, lines)
	})
}

func TestLsToolTruncation(t *testing.T) {
	tempDir := t.TempDir()
	lsTool := NewLsTool(tempDir)

	// Create more than MaxLsFiles files
	for i := 0; i < MaxLsFiles+10; i++ {
		fileName := fmt.Sprintf("file_%04d.txt", i)
		filePath := filepath.Join(tempDir, fileName)
		err := os.WriteFile(filePath, []byte("content"), 0o644)
		require.NoError(t, err)
	}

	params := LsParams{
		Path: tempDir,
		All:  false,
	}
	input, err := json.Marshal(params)
	require.NoError(t, err)

	response, err := lsTool.Run(context.Background(), ToolCall{
		Name:  "ls",
		Input: string(input),
	})

	require.NoError(t, err)
	assert.False(t, response.IsError)

	// Check truncation warning
	assert.Contains(t, response.Content, fmt.Sprintf("There are more than %d files", MaxLsFiles))

	// Parse metadata
	var metadata LsResponseMetadata
	if response.Metadata != "" {
		err = json.Unmarshal([]byte(response.Metadata), &metadata)
		require.NoError(t, err)
		assert.Equal(t, MaxLsFiles, metadata.NumberOfFiles)
		assert.True(t, metadata.Truncated)
	}

	// Count actual files listed (excluding warning message)
	lines := strings.Split(response.Content, "\n")
	fileCount := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "file_") {
			fileCount++
		}
	}
	assert.Equal(t, MaxLsFiles, fileCount)
}

func TestListDirectory(t *testing.T) {
	t.Run("BasicListing", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create test structure
		os.WriteFile(filepath.Join(tempDir, "a.txt"), []byte(""), 0o644)
		os.WriteFile(filepath.Join(tempDir, "b.txt"), []byte(""), 0o644)
		os.Mkdir(filepath.Join(tempDir, "dir"), 0o755)
		os.WriteFile(filepath.Join(tempDir, ".hidden"), []byte(""), 0o644)

		output, metadata, err := ListDirectory(tempDir, false, MaxLsFiles)
		require.NoError(t, err)

		assert.Contains(t, output, "a.txt")
		assert.Contains(t, output, "b.txt")
		assert.Contains(t, output, "dir/")
		assert.NotContains(t, output, ".hidden")
		assert.Equal(t, 3, metadata.NumberOfFiles)
		assert.False(t, metadata.Truncated)
	})

	t.Run("ShowAll", func(t *testing.T) {
		tempDir := t.TempDir()

		os.WriteFile(filepath.Join(tempDir, "visible.txt"), []byte(""), 0o644)
		os.WriteFile(filepath.Join(tempDir, ".hidden"), []byte(""), 0o644)
		os.Mkdir(filepath.Join(tempDir, ".hidden_dir"), 0o755)

		output, metadata, err := ListDirectory(tempDir, true, MaxLsFiles)
		require.NoError(t, err)

		assert.Contains(t, output, "visible.txt")
		assert.Contains(t, output, ".hidden")
		assert.Contains(t, output, ".hidden_dir/")
		assert.Equal(t, 3, metadata.NumberOfFiles)
		assert.False(t, metadata.Truncated)
	})
}