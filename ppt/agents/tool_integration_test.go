package agents

import (
	"strings"
	"testing"

	"gentica/agent"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTemplateDesignAgent_ToolIntegration 验证工具集成
func TestTemplateDesignAgent_ToolIntegration(t *testing.T) {
	// 如果没有初始化 genkit 或共享依赖，跳过测试
	if g == nil || sharedTemplateAgent == nil {
		t.Skip("Skipping test: genkit not initialized or shared dependencies not available")
	}

	// 使用共享的 TemplateAgent
	templateAgent := sharedTemplateAgent

	// 验证 Agent 类型
	baseAgent, ok := templateAgent.(*agent.BaseAgent)
	require.True(t, ok, "Should be a BaseAgent")

	// 获取配置
	config := baseAgent.GetConfig()

	// 验证基本配置
	assert.Equal(t, "template_design_agent", config.Name)
	assert.Equal(t, float32(0.9), config.Temperature)
	assert.Equal(t, 12000, config.MaxTokens)

	// 验证工具数量 - 应该有5个工具
	assert.Equal(t, 5, len(config.Tools), "Should have 5 tools (DirectoryAdd, DirectoryList, Write, View, Ls)")
}

// TestResearchCollector_ToolIntegration 验证研究收集器工具集成
func TestResearchCollector_ToolIntegration(t *testing.T) {
	// 如果没有初始化 genkit 或共享依赖，跳过测试
	if g == nil || sharedDeps == nil {
		t.Skip("Skipping test: genkit not initialized or shared dependencies not available")
	}

	// 使用共享依赖创建 collector
	collector := NewResearchCollectorWithDeps(g, sharedDeps)

	// 验证 Agent 类型
	baseAgent, ok := collector.(*agent.BaseAgent)
	require.True(t, ok, "Should be a BaseAgent")

	// 获取配置
	config := baseAgent.GetConfig()

	// 验证基本配置
	assert.Equal(t, "research_collector", config.Name)
	assert.Equal(t, float32(0.3), config.Temperature)
	assert.Equal(t, 4000, config.MaxTokens)

	// 验证工具数量 - 应该有9个工具
	assert.Equal(t, 9, len(config.Tools), "Should have 9 tools including Ls and DirectoryList")
}

// TestBothAgentsHaveLsAndDirectoryList 验证两个 Agent 都有 ls 和 directory list 工具
// 注意：这个测试依赖于 TestMain 中的共享依赖，避免重复注册工具
func TestBothAgentsHaveLsAndDirectoryList(t *testing.T) {
	// 如果没有初始化 genkit 或共享依赖，跳过测试
	if g == nil || sharedDeps == nil {
		t.Skip("Skipping test: genkit or shared dependencies not initialized")
	}

	// 测试 TemplateDesignAgent 结构
	t.Run("TemplateDesignAgent_Structure", func(t *testing.T) {
		// 创建一个模拟的依赖结构来验证字段存在
		tempDeps := &TemplateDesignAgentDependencies{}

		// 验证结构体有所有必要的字段（通过类型检查）
		var _ = tempDeps.DirectoryAddTool
		var _ = tempDeps.DirectoryListTool
		var _ = tempDeps.WriteTool
		var _ = tempDeps.ViewTool
		var _ = tempDeps.LsTool

		// 如果能编译通过，说明所有字段都存在
		assert.True(t, true, "TemplateDesignAgent has all required tool fields")
	})

	// 测试 ResearchCollector 使用共享依赖
	t.Run("ResearchCollector_WithSharedDeps", func(t *testing.T) {
		// 使用共享依赖创建 collector
		collector := NewResearchCollectorWithDeps(g, sharedDeps)
		assert.NotNil(t, collector)

		// 获取配置验证工具数量
		if baseAgent, ok := collector.(*agent.BaseAgent); ok {
			config := baseAgent.GetConfig()
			// 研究收集器应该有9个工具
			assert.Equal(t, 9, len(config.Tools), "ResearchCollector should have 9 tools")

			// 验证包含 ls 和 directory list 工具
			var hasLsTool, hasDirectoryListTool bool
			for _, tool := range config.Tools {
				toolName := tool.Name()
				if toolName == "ls" {
					hasLsTool = true
				}
				if strings.Contains(toolName, "resource_directory_list") {
					hasDirectoryListTool = true
				}
			}
			assert.True(t, hasLsTool, "ResearchCollector should have ls tool")
			assert.True(t, hasDirectoryListTool, "ResearchCollector should have resource_directory_list tool")
		}
	})

	// 测试 OutlinePlanAgent 结构
	t.Run("OutlinePlanAgent_Structure", func(t *testing.T) {
		// 创建一个模拟的依赖结构来验证字段存在
		outlineDeps := &OutlinePlanAgentDependencies{}

		// 验证结构体有所有必要的字段（通过类型检查）
		var _ = outlineDeps.DirectoryAddTool
		var _ = outlineDeps.DirectoryListTool
		var _ = outlineDeps.WriteTool
		var _ = outlineDeps.ViewTool
		var _ = outlineDeps.LsTool

		// 如果能编译通过，说明所有字段都存在
		assert.True(t, true, "OutlinePlanAgent has all required tool fields including LsTool and DirectoryListTool")
	})

	// 测试 PageGenerateAgent 结构
	t.Run("PageGenerateAgent_Structure", func(t *testing.T) {
		// 创建一个模拟的依赖结构来验证字段存在
		pageDeps := &PageGenerateAgentDependencies{}

		// 验证结构体有所有必要的字段（通过类型检查）
		var _ = pageDeps.DirectoryAddTool
		var _ = pageDeps.DirectoryListTool
		var _ = pageDeps.WriteTool
		var _ = pageDeps.ViewTool
		var _ = pageDeps.HtmlSizeTool
		var _ = pageDeps.LsTool

		// 如果能编译通过，说明所有字段都存在
		assert.True(t, true, "PageGenerateAgent has all required tool fields including LsTool and DirectoryListTool")
	})
}