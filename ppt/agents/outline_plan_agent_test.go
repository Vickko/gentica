package agents

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOutlinePlanAgent(t *testing.T) {
	// 使用包级别的共享配置
	if g == nil || sharedOutlineAgent == nil {
		t.Skip("Skipping test: genkit not initialized or shared dependencies not available")
	}

	// 准备典型测试输入
	input := map[string]any{
		"topic": "软件测试基础培训PPT",
	}
	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	// 调用 OutlinePlanAgent
	t.Logf("Starting outline generation for topic: %s", input["topic"])
	result, err := sharedOutlineAgent.Run(context.Background(), string(inputJSON))
	require.NoError(t, err)
	require.NotEmpty(t, result)

	// 解析和验证结果
	outlineResult, err := ParseOutlineResult(result)
	require.NoError(t, err)
	require.NotNil(t, outlineResult)
	require.Equal(t, "success", outlineResult.Status)

	// 验证关键字段
	require.NotEmpty(t, outlineResult.FilePath)
	require.NotEmpty(t, outlineResult.DirectoryName)

	// 手动读取和解析 XML 文件（因为 ParseOutlineResult 不解析 XML 内容）
	xmlContent, err := os.ReadFile(outlineResult.FilePath)
	require.NoError(t, err)
	require.NotEmpty(t, xmlContent)

	// 解析 XML 内容
	plan, err := ParsePPTPlan(string(xmlContent))
	require.NoError(t, err)
	require.NotNil(t, plan)

	// 输出关键信息
	t.Logf("=== Outline Generation Results ===")
	t.Logf("Status: %s", outlineResult.Status)
	t.Logf("File Path: %s", outlineResult.FilePath)
	t.Logf("Directory: %s", outlineResult.DirectoryName)
	t.Logf("Title: %s", plan.ProjectInfo.Title)
	t.Logf("Total Pages: %d", plan.ProjectInfo.TotalPages)
	t.Logf("Target Audience: %s", plan.ProjectInfo.TargetAudience)
	t.Logf("Design Style: %s", plan.ProjectInfo.DesignStyle)

	// 验证页面结构
	require.True(t, len(plan.PageStructure.Pages) > 0)
	t.Logf("Generated %d pages:", len(plan.PageStructure.Pages))
	for i, page := range plan.PageStructure.Pages {
		if i < 3 { // 只显示前3页避免输出过长
			t.Logf("  Page %d: %s (%s)", page.PageNumber, page.PageTitle, page.PageType)
		}
	}
	if len(plan.PageStructure.Pages) > 3 {
		t.Logf("  ... and %d more pages", len(plan.PageStructure.Pages)-3)
	}
}
