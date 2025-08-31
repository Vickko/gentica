package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// 通用测试用例结构
type testCase struct {
	name          string
	original      string
	diff          string
	expected      string
	expectError   bool
	errorContains string
}

// 包级别测试用例变量 - 基于Unix patch行为
var editTestCases = []testCase{
	{
		name:     "Simple line addition",
		original: "line 1\nline 2\nline 3\n",
		diff: `@@ -1,3 +1,4 @@
 line 1
+new line
 line 2
 line 3`,
		expected:    "line 1\nnew line\nline 2\nline 3\n",
		expectError: false,
	},
	{
		name:     "Simple line deletion",
		original: "line 1\nline 2\nline 3\n",
		diff: `@@ -1,3 +1,2 @@
 line 1
-line 2
 line 3`,
		expected:    "line 1\nline 3\n",
		expectError: false,
	},
	{
		name:     "Line modification",
		original: "line 1\nline 2\nline 3\n",
		diff: `@@ -1,3 +1,3 @@
 line 1
-line 2
+modified line 2
 line 3`,
		expected:    "line 1\nmodified line 2\nline 3\n",
		expectError: false,
	},
	{
		name:     "Multiple changes",
		original: "line 1\nline 2\nline 3\nline 4\n",
		diff: `@@ -1,4 +1,5 @@
 line 1
-line 2
+modified line 2
+new line
 line 3
 line 4`,
		expected:    "line 1\nmodified line 2\nnew line\nline 3\nline 4\n",
		expectError: false,
	},
	{
		name:     "Empty file to content",
		original: "",
		diff: `@@ -1,0 +1,2 @@
+first line
+second line`,
		expected:    "first line\nsecond line", // Unix patch不会添加尾部换行符
		expectError: false,
	},
	{
		name:     "Content to empty file - Unix patch sensitive",
		original: "line 1\nline 2\n",
		diff: `@@ -1,2 +0,0 @@
-line 1
-line 2`,
		expected:      "",
		expectError:   true, // Unix patch对这种情况处理困难
		errorContains: "patch failed",
	},
	{
		name:     "Single character change",
		original: "hello world\n",
		diff: `@@ -1 +1 @@
-hello world
+hello world!`,
		expected:    "hello world!", // Unix patch丢失尾部换行符
		expectError: false,
	},
	{
		name:     "Unicode characters",
		original: "你好世界\n测试\n",
		diff: `@@ -1,2 +1,2 @@
 你好世界
-测试
+测试完成`,
		expected:    "你好世界\n测试完成", // Unix patch丢失尾部换行符
		expectError: false,
	},
	{
		name:     "Long lines",
		original: "This is a very long line that contains many words and should test the diff algorithm's ability to handle lengthy content without issues\n",
		diff: `@@ -1 +1 @@
-This is a very long line that contains many words and should test the diff algorithm's ability to handle lengthy content without issues
+This is a very long line that contains many words and should test the diff algorithm's ability to handle lengthy content without any issues whatsoever`,
		expected:    "This is a very long line that contains many words and should test the diff algorithm's ability to handle lengthy content without any issues whatsoever", // Unix patch丢失尾部换行符
		expectError: false,
	},
	{
		name:     "No newline at end",
		original: "line without newline",
		diff: `@@ -1 +1 @@
-line without newline
\ No newline at end of file
+line without newline modified
\ No newline at end of file`,
		expected:    "line without newline modified",
		expectError: false,
	},
	{
		name:     "Mixed line endings - not supported by patch",
		original: "line 1\nline 2\r\nline 3\n",
		diff: `@@ -1,3 +1,3 @@
 line 1
-line 2
+modified line 2
 line 3`,
		expected:      "",
		expectError:   true,
		errorContains: "patch failed",
	},
	{
		name:     "Large context diff",
		original: "line 1\nline 2\nline 3\nline 4\nline 5\nline 6\nline 7\nline 8\nline 9\nline 10\n",
		diff: `@@ -2,7 +2,7 @@
 line 2
 line 3
 line 4
-line 5
+modified line 5
 line 6
 line 7
 line 8`,
		expected:    "line 1\nline 2\nline 3\nline 4\nmodified line 5\nline 6\nline 7\nline 8\nline 9\nline 10\n",
		expectError: false,
	},
	{
		name:          "Invalid diff format",
		original:      "line 1\nline 2\n",
		diff:          "invalid diff format",
		expected:      "",
		expectError:   true,
		errorContains: "no files found in diff",
	},
}

func TestEditFile(t *testing.T) {
	// Create temporary directory for tests
	tempDir, err := os.MkdirTemp("", "edit_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	for _, tt := range editTestCases {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			testFile := filepath.Join(tempDir, "test_"+strings.ReplaceAll(tt.name, " ", "_")+".txt")
			err := os.WriteFile(testFile, []byte(tt.original), 0644)
			if err != nil {
				t.Fatal(err)
			}

			// Apply diff
			err = EditFile(testFile, tt.diff)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Read result
			result, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatal(err)
			}

			if string(result) != tt.expected {
				t.Errorf("Expected:\n%q\nGot:\n%q", tt.expected, string(result))
			}
		})
	}
}

func TestUnixDiffPatchVerification(t *testing.T) {
	// Create temporary directory for tests
	tempDir, err := os.MkdirTemp("", "unix_verify_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// 使用相同的测试用例验证Unix patch行为
	for _, tt := range editTestCases {
		t.Run(tt.name, func(t *testing.T) {
			// Create original file
			originalFile := filepath.Join(tempDir, "original_"+strings.ReplaceAll(tt.name, " ", "_")+".txt")
			err := os.WriteFile(originalFile, []byte(tt.original), 0644)
			if err != nil {
				t.Fatal(err)
			}

			// Create a copy for Unix patch test
			testFile := filepath.Join(tempDir, "test_"+strings.ReplaceAll(tt.name, " ", "_")+".txt")
			err = os.WriteFile(testFile, []byte(tt.original), 0644)
			if err != nil {
				t.Fatal(err)
			}

			// Create patch file
			patchFile := filepath.Join(tempDir, "patch_"+strings.ReplaceAll(tt.name, " ", "_")+".patch")
			fullDiff := "--- a/" + testFile + "\n+++ b/" + testFile + "\n" + tt.diff
			err = os.WriteFile(patchFile, []byte(fullDiff), 0644)
			if err != nil {
				t.Fatal(err)
			}

			// Apply patch using Unix patch command
			cmd := exec.Command("patch", testFile, patchFile)
			output, err := cmd.CombinedOutput()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but Unix patch succeeded")
					return
				}
				t.Logf("Unix patch failed as expected: %s", output)
				return
			}

			if err != nil {
				t.Errorf("Unix patch failed unexpectedly: %v\nOutput: %s", err, output)
				return
			}

			// Read result
			result, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatal(err)
			}

			if string(result) != tt.expected {
				t.Errorf("Unix patch result differs from expected:\nExpected:\n%q\nGot:\n%q", tt.expected, string(result))
			} else {
				t.Logf("Unix patch test passed for: %s", tt.name)
			}
		})
	}
}

func TestEditFileFileOperations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "edit_test_file_ops")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("Non-existent file", func(t *testing.T) {
		nonExistentFile := filepath.Join(tempDir, "does_not_exist.txt")
		err := EditFile(nonExistentFile, "@@ -1 +1 @@\n-old\n+new")
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
		if !contains(err.Error(), "failed to read") {
			t.Errorf("Expected 'failed to read file' error, got: %v", err)
		}
	})

	t.Run("Permission denied write", func(t *testing.T) {
		// Create a file and make directory read-only
		readOnlyDir := filepath.Join(tempDir, "readonly")
		err := os.Mkdir(readOnlyDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		testFile := filepath.Join(readOnlyDir, "test.txt")
		err = os.WriteFile(testFile, []byte("original content"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		// Make directory read-only
		err = os.Chmod(readOnlyDir, 0444)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(readOnlyDir, 0755) // Restore permissions for cleanup

		err = EditFile(testFile, "@@ -1 +1 @@\n-original content\n+new content")
		if err == nil {
			t.Error("Expected error for permission denied")
		}
	})
}

func TestEditFileComplexScenarios(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "edit_test_complex")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("Multiple patches in one diff", func(t *testing.T) {
		original := "section 1\nline 2\nline 3\n\nsection 2\nline 6\nline 7\n"
		diff := `@@ -1,3 +1,3 @@
 section 1
-line 2
+modified line 2
 line 3
@@ -5,3 +5,3 @@
 section 2
-line 6
+modified line 6
 line 7`

		testFile := filepath.Join(tempDir, "multi_patch.txt")
		err := os.WriteFile(testFile, []byte(original), 0644)
		if err != nil {
			t.Fatal(err)
		}

		err = EditFile(testFile, diff)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		result, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatal(err)
		}

		expected := "section 1\nmodified line 2\nline 3\n\nsection 2\nmodified line 6\nline 7\n"
		if string(result) != expected {
			t.Errorf("Expected:\n%q\nGot:\n%q", expected, string(result))
		}
	})
}

// Helper function since strings.Contains might not be available in test context
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (len(substr) == 0 || indexof(s, substr) >= 0)
}

func indexof(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
