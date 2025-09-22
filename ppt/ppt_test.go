package ppt

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gentica/agent"
	"gentica/ppt/agents"
	"gentica/tools"

	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/openai/openai-go/option"
	"github.com/stretchr/testify/require"
)

func TestResearchCollectorFullWorkflow(t *testing.T) {
	// 初始化 OpenAI 配置
	apiKey := "sk-6kgtZQDkmZDQMfCo28C360320cEf45FaAf1577Ef08F4032b"
	baseURL := "https://aihubmix.com/v1"

	oai := &openai.OpenAI{
		APIKey: apiKey,
		Opts: []option.RequestOption{
			option.WithBaseURL(baseURL),
		},
	}

	// 初始化 Genkit
	g := genkit.Init(
		context.Background(),
		genkit.WithPlugins(oai),
	)

	// 创建临时工作目录
	tempDir, err := os.MkdirTemp("", "research_test_*")
	require.NoError(t, err)
	// defer os.RemoveAll(tempDir) // 暂时注释掉，以便调试

	// 构建工具依赖
	// 基础文件操作工具
	viewTool := tools.AdaptBaseToolToGenkit(g, tools.NewViewTool(tempDir))
	writeTool := tools.AdaptBaseToolToGenkit(g, tools.NewWriteTool(tempDir))
	lsTool := tools.AdaptBaseToolToGenkit(g, tools.NewLsTool(tempDir))

	// Bash 工具
	bashTool := tools.AdaptBaseToolToGenkit(g, tools.NewBashTool(tempDir))

	// 资源目录管理工具
	directoryListTool := tools.AdaptBaseToolToGenkit(g, tools.NewResourceDirectoryListTool(tempDir))
	directoryAddTool := tools.AdaptBaseToolToGenkit(g, tools.NewResourceDirectoryAddTool(tempDir))

	// 搜索爬取工具
	searchCrawlerTool := tools.AdaptBaseToolToGenkit(g, tools.NewSearchCrawlerTool(tempDir))

	// 构建 Agent 依赖
	// 搜索需求分析器
	searchNeedsAnalyzer := tools.AdaptBaseToolToGenkit(g, agent.AsToolAdapter(
		agents.NewSearchNeedsAnalyzer(g),
	))

	// 文章评估器 - 需要注入 ViewTool
	articleEvaluator := tools.AdaptBaseToolToGenkit(g, agent.AsToolAdapter(
		agents.NewArticleEvaluatorWithDeps(g, &agents.ArticleEvaluatorDependencies{
			ViewTool: viewTool,
		}),
	))

	// 创建完整的依赖结构
	deps := &agents.ResearchCollectorDependencies{
		ViewTool:            viewTool,
		WriteTool:           writeTool,
		LsTool:              lsTool,
		BashTool:            bashTool,
		DirectoryListTool:   directoryListTool,
		DirectoryAddTool:    directoryAddTool,
		SearchCrawlerTool:   searchCrawlerTool,
		SearchNeedsAnalyzer: searchNeedsAnalyzer,
		ArticleEvaluator:    articleEvaluator,
	}

	// 创建 ResearchCollector
	collector := agents.NewResearchCollectorWithDeps(g, deps)

	// 定义研究主题 - 云原生应用的监控与可观测性最佳实践
	researchTopic := "云原生应用的监控与可观测性最佳实践"

	// 准备输入
	input := map[string]any{
		"research_topic": researchTopic,
	}

	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	// 执行研究收集
	t.Logf("开始研究主题: %s", researchTopic)
	t.Logf("临时目录: %s", tempDir)

	ctx := context.Background()
	result, err := collector.Run(ctx, string(inputJSON))
	require.NoError(t, err, "研究收集应该成功完成")

	// 解析结果
	var collectorResult agents.ResearchCollectorResult
	err = parseResult(result, &collectorResult)
	require.NoError(t, err, "应该能够解析结果为 JSON")

	// 验证结果
	require.Equal(t, "success", collectorResult.Status, "状态应该是 success")
	require.NotEmpty(t, collectorResult.Summary, "总结不应为空")
	require.NotEmpty(t, collectorResult.DirectoryName, "应该创建了资源目录")

	// 检查目录是否真的被创建
	// 资源目录实际创建在 .tmp/{uid}/{name} 路径下
	// 使用 glob 模式查找实际的目录
	globPattern := filepath.Join(tempDir, ".tmp", "*", collectorResult.DirectoryName)
	matches, err := filepath.Glob(globPattern)
	require.NoError(t, err, "查找目录时不应出错")
	require.NotEmpty(t, matches, "应该找到至少一个匹配的目录")

	// 使用找到的第一个匹配目录
	dirPath := matches[0]
	info, err := os.Stat(dirPath)
	require.NoError(t, err, "目录应该存在")
	require.True(t, info.IsDir(), "应该是一个目录")

	// 检查收集的文件
	files, err := os.ReadDir(dirPath)
	require.NoError(t, err)

	// 输出测试结果
	t.Logf("========== 研究收集完成 ==========")
	t.Logf("状态: %s", collectorResult.Status)
	t.Logf("目录: %s", collectorResult.DirectoryName)
	t.Logf("收集文件数: %d", len(files))
	t.Logf("工作总结:\n%s", collectorResult.Summary)

	if len(files) > 0 {
		t.Logf("========== 收集的文件列表 ==========")
		for i, file := range files {
			fileInfo, _ := file.Info()
			t.Logf("%d. %s (%.2f KB)", i+1, file.Name(), float64(fileInfo.Size())/1024)
		}
		require.GreaterOrEqual(t, len(files), 1, "应该至少收集到一些高质量文件")
	}
}

// parseResult 解析 Agent 返回的 JSON 结果
func parseResult(result string, v interface{}) error {
	// 查找 JSON 边界
	startIdx := -1
	endIdx := -1

	for i, ch := range result {
		if ch == '{' && startIdx == -1 {
			startIdx = i
		}
		if ch == '}' {
			endIdx = i
		}
	}

	if startIdx != -1 && endIdx != -1 && startIdx < endIdx {
		jsonStr := result[startIdx : endIdx+1]
		return json.Unmarshal([]byte(jsonStr), v)
	}

	// 尝试直接解析整个结果
	return json.Unmarshal([]byte(result), v)
}
