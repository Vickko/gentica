package agents

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTemplateDesignAgent 独立运行的典型场景测试
// 复用包级别配置，发送真实API请求验证完整流程
// 展示Agent的标准使用方式
func TestTemplateDesignAgent(t *testing.T) {
	// 检查共享依赖是否可用
	if g == nil || sharedTemplateAgent == nil {
		t.Skip("Skipping test: genkit not initialized or shared dependencies not available")
	}

	// 使用典型的科技风格作为测试场景
	input := map[string]any{
		"style_description": "现代科技风格",
	}

	// 序列化输入
	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	// 执行 Agent - 这是标准的使用方式
	t.Logf("执行模板设计 Agent，风格：%s", input["style_description"])
	result, err := sharedTemplateAgent.Run(context.Background(), string(inputJSON))
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// 解析并验证结果
	designResult, err := ParseTemplateDesignResult(result)
	require.NoError(t, err)

	// 验证基本状态
	assert.Equal(t, "success", designResult.Status)
	assert.NotEmpty(t, designResult.DirectoryName)
	assert.NotEmpty(t, designResult.Summary)

	// 验证生成了所有必需的文件
	expectedPages := []string{"cover", "toc", "content", "data", "ending"}
	assert.Equal(t, len(expectedPages), len(designResult.FilePaths))

	for _, pageType := range expectedPages {
		assert.Contains(t, designResult.FilePaths, pageType, "应该包含%s页面", pageType)
		filePath := designResult.FilePaths[pageType]
		assert.NotEmpty(t, filePath, "%s页面的文件路径不应为空", pageType)

		// 验证文件确实存在
		_, err := os.Stat(filePath)
		assert.NoError(t, err, "%s页面文件应该存在: %s", pageType, filePath)
	}

	// 输出测试结果信息
	t.Logf("✅ 模板设计完成")
	t.Logf("状态: %s", designResult.Status)
	t.Logf("目录: %s", designResult.DirectoryName)
	t.Logf("摘要: %s", designResult.Summary)
	t.Logf("生成文件数量: %d", len(designResult.FilePaths))
}