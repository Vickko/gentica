package tools

import (
	"context"
	"testing"

	"github.com/firebase/genkit/go/genkit"
	"github.com/stretchr/testify/require"
)

// mockTool 用于测试的模拟工具
type mockTool struct {
	name        string
	description string
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Info() ToolInfo {
	return ToolInfo{
		Name:        m.name,
		Description: m.description,
		Parameters: map[string]any{
			"input": map[string]any{
				"type":        "string",
				"description": "Test input parameter",
			},
			"count": map[string]any{
				"type":        "integer",
				"description": "Test count parameter",
			},
		},
		Required: []string{"input"},
	}
}

func (m *mockTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	return NewTextResponse("Mock response for: " + call.Input), nil
}

func TestAdaptBaseToolToGenkit(t *testing.T) {
	// 初始化 Genkit
	g := genkit.Init(context.Background())

	// 创建模拟工具
	mockBaseTool := &mockTool{
		name:        "test_tool",
		description: "A test tool for adapter",
	}

	// 使用适配器转换
	genkitTool := AdaptBaseToolToGenkit(g, mockBaseTool)

	// 验证工具被正确创建
	require.NotNil(t, genkitTool)

	// 测试工具执行
	ctx := context.Background()
	input := map[string]any{
		"input": "test input",
		"count": 5,
	}

	result, err := genkitTool.RunRaw(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, result)

	// 验证返回值
	resultStr, ok := result.(string)
	require.True(t, ok)
	require.Contains(t, resultStr, "Mock response")
}

func TestBatchAdaptTools(t *testing.T) {
	// 初始化 Genkit
	g := genkit.Init(context.Background())

	// 创建多个模拟工具
	tools := []BaseTool{
		&mockTool{name: "tool1", description: "First tool"},
		&mockTool{name: "tool2", description: "Second tool"},
		&mockTool{name: "tool3", description: "Third tool"},
	}

	// 批量转换
	genkitTools := BatchAdaptTools(g, tools...)

	// 验证数量
	require.Len(t, genkitTools, 3)

	// 验证每个工具都不为空
	for _, tool := range genkitTools {
		require.NotNil(t, tool)
	}
}