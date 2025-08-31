package tools

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestReadFile(t *testing.T) {
	// 创建临时目录用于测试
	tempDir := t.TempDir()

	// 创建测试文件
	testFiles := map[string]string{
		"empty.txt":       "",
		"single_line.txt": "line 1",
		"multi_line.txt": strings.Join([]string{
			"line 1",
			"line 2",
			"line 3",
			"line 4",
			"line 5",
		}, "\n"),
		"with_empty_lines.txt": strings.Join([]string{
			"line 1",
			"",
			"line 3",
			"",
			"line 5",
		}, "\n"),
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	tests := []struct {
		name        string
		filename    string
		startLine   int
		endLine     int
		expected    []string
		expectError bool
		errorMsg    string
	}{
		// 正常情况测试
		{
			name:        "读取全部文件 - startLine=endLine=0",
			filename:    "multi_line.txt",
			startLine:   0,
			endLine:     0,
			expected:    []string{"line 1", "line 2", "line 3", "line 4", "line 5"},
			expectError: false,
		},
		{
			name:        "读取部分行 - 中间范围",
			filename:    "multi_line.txt",
			startLine:   2,
			endLine:     4,
			expected:    []string{"line 2", "line 3"},
			expectError: false,
		},
		{
			name:        "读取第一行",
			filename:    "multi_line.txt",
			startLine:   1,
			endLine:     2,
			expected:    []string{"line 1"},
			expectError: false,
		},
		{
			name:        "读取最后一行",
			filename:    "multi_line.txt",
			startLine:   5,
			endLine:     6,
			expected:    []string{"line 5"},
			expectError: false,
		},
		{
			name:        "读取单行文件全部内容",
			filename:    "single_line.txt",
			startLine:   0,
			endLine:     0,
			expected:    []string{"line 1"},
			expectError: false,
		},
		{
			name:        "读取单行文件指定范围",
			filename:    "single_line.txt",
			startLine:   1,
			endLine:     2,
			expected:    []string{"line 1"},
			expectError: false,
		},
		{
			name:        "读取空文件",
			filename:    "empty.txt",
			startLine:   0,
			endLine:     0,
			expected:    []string{},
			expectError: false,
		},
		{
			name:        "读取包含空行的文件",
			filename:    "with_empty_lines.txt",
			startLine:   1,
			endLine:     4,
			expected:    []string{"line 1", "", "line 3"},
			expectError: false,
		},

		// 错误情况测试
		{
			name:        "startLine为0但endLine不为0",
			filename:    "multi_line.txt",
			startLine:   0,
			endLine:     3,
			expectError: true,
			errorMsg:    "startLine must be greater than 0",
		},
		{
			name:        "startLine为负数",
			filename:    "multi_line.txt",
			startLine:   -1,
			endLine:     3,
			expectError: true,
			errorMsg:    "startLine must be greater than 0",
		},
		{
			name:        "endLine等于startLine",
			filename:    "multi_line.txt",
			startLine:   3,
			endLine:     3,
			expectError: true,
			errorMsg:    "endLine must be greater than startLine",
		},
		{
			name:        "endLine小于startLine",
			filename:    "multi_line.txt",
			startLine:   4,
			endLine:     2,
			expectError: true,
			errorMsg:    "endLine must be greater than startLine",
		},
		{
			name:        "startLine超出文件行数",
			filename:    "multi_line.txt",
			startLine:   10,
			endLine:     15,
			expectError: true,
			errorMsg:    "startLine 10 exceeds file length",
		},
		{
			name:        "startLine超出单行文件",
			filename:    "single_line.txt",
			startLine:   2,
			endLine:     3,
			expectError: true,
			errorMsg:    "startLine 2 exceeds file length",
		},
		{
			name:        "startLine超出空文件",
			filename:    "empty.txt",
			startLine:   1,
			endLine:     2,
			expectError: true,
			errorMsg:    "startLine 1 exceeds file length",
		},
		{
			name:        "文件不存在",
			filename:    "nonexistent.txt",
			startLine:   1,
			endLine:     2,
			expectError: true,
			errorMsg:    "failed to open file",
		},

		// 边界情况测试
		{
			name:        "endLine远超文件行数",
			filename:    "multi_line.txt",
			startLine:   3,
			endLine:     100,
			expected:    []string{"line 3", "line 4", "line 5"},
			expectError: false,
		},
		{
			name:        "读取文件所有行的精确范围",
			filename:    "multi_line.txt",
			startLine:   1,
			endLine:     6,
			expected:    []string{"line 1", "line 2", "line 3", "line 4", "line 5"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var filePath string
			if tt.filename == "nonexistent.txt" {
				filePath = filepath.Join(tempDir, tt.filename)
			} else {
				filePath = filepath.Join(tempDir, tt.filename)
			}

			result, err := readFile(filePath, tt.startLine, tt.endLine)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// 特殊处理空切片的比较
			if len(tt.expected) == 0 && len(result) == 0 {
				// 两个都是空切片，认为相等
				return
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestReadAllLines(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		content     string
		expected    []string
		expectError bool
	}{
		{
			name:     "正常多行文件",
			content:  "line 1\nline 2\nline 3",
			expected: []string{"line 1", "line 2", "line 3"},
		},
		{
			name:     "单行文件",
			content:  "single line",
			expected: []string{"single line"},
		},
		{
			name:     "空文件",
			content:  "",
			expected: []string{},
		},
		{
			name:     "包含空行的文件",
			content:  "line 1\n\nline 3",
			expected: []string{"line 1", "", "line 3"},
		},
		{
			name:     "只有换行符的文件",
			content:  "\n\n\n",
			expected: []string{"", "", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := filepath.Join(tempDir, "test_"+strings.ReplaceAll(tt.name, " ", "_")+".txt")
			if err := os.WriteFile(filename, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			result, err := readAllLines(filename)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// 特殊处理空切片的比较
			if len(tt.expected) == 0 && len(result) == 0 {
				// 两个都是空切片，认为相等
				return
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestReadAllLines_FileNotFound(t *testing.T) {
	tempDir := t.TempDir()
	nonexistentFile := filepath.Join(tempDir, "nonexistent.txt")

	_, err := readAllLines(nonexistentFile)
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "failed to open file") {
		t.Errorf("Expected error message to contain 'failed to open file', got '%s'", err.Error())
	}
}

// 基准测试
func BenchmarkReadFile_SmallFile(b *testing.B) {
	tempDir := b.TempDir()
	filename := filepath.Join(tempDir, "small.txt")
	content := strings.Repeat("line\n", 100)
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := readFile(filename, 10, 20)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkReadFile_LargeFile(b *testing.B) {
	tempDir := b.TempDir()
	filename := filepath.Join(tempDir, "large.txt")
	content := strings.Repeat("line\n", 10000)
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := readFile(filename, 100, 200)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkReadFile_ReadAll(b *testing.B) {
	tempDir := b.TempDir()
	filename := filepath.Join(tempDir, "all.txt")
	content := strings.Repeat("line\n", 1000)
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := readFile(filename, 0, 0)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}
