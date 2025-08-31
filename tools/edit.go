package tools

import (
	"fmt"
	"os"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
)

// 常量定义
const (
	mixedLineEndingsMsg = "patch failed: mixed line endings not supported"
	emptyFileMsg        = "patch failed: content to empty file not supported"
	noNewlineMarker     = "\\ No newline at end of file"
	defaultFileMode     = 0644
)

// EditFile 应用 unified diff 到指定文件
// filePath: 目标文件路径
// diffContent: 包含要应用的变更的 unified diff 字符串
func EditFile(filePath, diffContent string) error {
	// 预处理检查
	if err := validateDiffContent(diffContent); err != nil {
		return err
	}

	// 确保diff有正确的文件头
	diffContent = ensureDiffHeader(diffContent, filePath)

	// 解析diff并找到目标文件
	targetFile, err := parseDiffAndFindFile(diffContent, filePath)
	if err != nil {
		return err
	}

	// 读取并验证原始文件
	originalLines, originalStr, err := readAndValidateOriginalFile(filePath)
	if err != nil {
		return err
	}

	// 应用所有hunks
	modifiedLines := applyHunks(originalLines, targetFile.TextFragments)

	// 处理换行符并写回文件
	return writeModifiedFile(filePath, modifiedLines, originalStr, diffContent, targetFile.TextFragments)
}

// 辅助函数

// validateDiffContent 验证diff内容
func validateDiffContent(diffContent string) error {
	if hasMixedLineEndings(diffContent) {
		return fmt.Errorf(mixedLineEndingsMsg)
	}

	if isContentToEmptyFile(diffContent) {
		return fmt.Errorf(emptyFileMsg)
	}

	return nil
}

// hasMixedLineEndings 检查是否有混合行尾
func hasMixedLineEndings(content string) bool {
	return strings.Contains(content, "\r\n") && strings.Contains(content, "\n")
}

// isContentToEmptyFile 检查是否是向空文件添加内容的特殊情况
func isContentToEmptyFile(diffContent string) bool {
	return strings.Contains(diffContent, "@@ -") && strings.Contains(diffContent, " +0,0 @@")
}

// ensureDiffHeader 确保diff有正确的文件头
func ensureDiffHeader(diffContent, filePath string) string {
	if !strings.Contains(diffContent, "---") || !strings.Contains(diffContent, "+++") {
		return fmt.Sprintf("--- a/%s\n+++ b/%s\n%s", filePath, filePath, diffContent)
	}
	return diffContent
}

// parseDiffAndFindFile 解析diff并找到目标文件
func parseDiffAndFindFile(diffContent, filePath string) (*gitdiff.File, error) {
	files, _, err := gitdiff.Parse(strings.NewReader(diffContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse diff: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no files found in diff")
	}

	targetFile := findMatchingFile(files, filePath)
	if targetFile == nil {
		return nil, fmt.Errorf("file %s not found in diff", filePath)
	}

	return targetFile, nil
}

// findMatchingFile 在文件列表中找到匹配的文件
func findMatchingFile(files []*gitdiff.File, filePath string) *gitdiff.File {
	for _, file := range files {
		if isFileMatch(file, filePath) {
			return file
		}
	}
	return nil
}

// isFileMatch 检查文件是否匹配
func isFileMatch(file *gitdiff.File, filePath string) bool {
	return file.NewName == filePath || file.OldName == filePath ||
		strings.HasSuffix(file.NewName, filePath) || strings.HasSuffix(file.OldName, filePath)
}

// readAndValidateOriginalFile 读取并验证原始文件
func readAndValidateOriginalFile(filePath string) ([]string, string, error) {
	originalContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read original file: %w", err)
	}

	originalStr := string(originalContent)
	if hasMixedLineEndings(originalStr) {
		return nil, "", fmt.Errorf("patch failed: original file has mixed line endings")
	}

	originalLines := strings.Split(originalStr, "\n")
	return originalLines, originalStr, nil
}

// applyHunks 应用所有hunks到文件行
func applyHunks(originalLines []string, hunks []*gitdiff.TextFragment) []string {
	// 从后往前应用hunks，避免行号偏移问题
	for i := len(hunks) - 1; i >= 0; i-- {
		hunk := hunks[i]
		if hunk == nil {
			continue
		}

		startIdx := int(hunk.OldPosition) - 1 // 转为0索引
		if startIdx < 0 {
			startIdx = 0
		}

		originalLines = applyHunk(originalLines, hunk, startIdx)
	}
	return originalLines
}

// writeModifiedFile 写入修改后的文件
func writeModifiedFile(filePath string, modifiedLines []string, originalStr, diffContent string, hunks []*gitdiff.TextFragment) error {
	newContent := strings.Join(modifiedLines, "\n")

	// 处理换行符
	if shouldRemoveTrailingNewline(diffContent, originalStr, hunks) {
		newContent = strings.TrimSuffix(newContent, "\n")
	}

	err := os.WriteFile(filePath, []byte(newContent), defaultFileMode)
	if err != nil {
		return fmt.Errorf("failed to write modified file: %w", err)
	}

	return nil
}

// shouldRemoveTrailingNewline 判断是否应该移除尾部换行符
func shouldRemoveTrailingNewline(diffContent, originalStr string, hunks []*gitdiff.TextFragment) bool {
	// 检查"No newline at end of file"标记
	if strings.Contains(diffContent, noNewlineMarker) {
		return true
	}

	// 空文件特殊情况
	if len(originalStr) == 0 {
		return true
	}

	// 检查最后添加的行
	lastAddedLine := getLastAddedLine(hunks)
	originalEndsWithNewline := strings.HasSuffix(originalStr, "\n")

	return lastAddedLine != "" && !strings.HasSuffix(lastAddedLine, "\n") && originalEndsWithNewline
}

// getLastAddedLine 获取最后添加的行
func getLastAddedLine(hunks []*gitdiff.TextFragment) string {
	lastAddedLine := ""
	for _, hunk := range hunks {
		for _, line := range hunk.Lines {
			if line.Op == gitdiff.OpAdd {
				lastAddedLine = line.Line
			}
		}
	}
	return lastAddedLine
}

// applyHunk 应用单个 hunk 到文件行
func applyHunk(originalLines []string, hunk *gitdiff.TextFragment, startIdx int) []string {
	var result []string

	// 添加 hunk 前的行
	if startIdx > 0 {
		result = append(result, originalLines[:startIdx]...)
	}

	// 处理 hunk 中的行
	originalIdx := startIdx
	for _, line := range hunk.Lines {
		switch line.Op {
		case gitdiff.OpContext:
			// 上下文行：优先使用原文件中的行
			if originalIdx < len(originalLines) {
				result = append(result, originalLines[originalIdx])
			} else {
				result = append(result, strings.TrimSuffix(line.Line, "\n"))
			}
			originalIdx++
		case gitdiff.OpDelete:
			// 删除行：跳过原文件中的这一行
			originalIdx++
		case gitdiff.OpAdd:
			// 添加行：添加新行（去掉尾部换行符）
			result = append(result, strings.TrimSuffix(line.Line, "\n"))
		}
	}

	// 添加 hunk 后剩余的行
	if originalIdx < len(originalLines) {
		result = append(result, originalLines[originalIdx:]...)
	}

	return result
}
