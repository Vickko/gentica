package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gentica/agent"
	"gentica/tools"

	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/openai/openai-go/option"
	"github.com/stretchr/testify/require"
)

// 包级别的单例依赖和共享目录
var (
	sharedDeps    *ResearchCollectorDependencies
	sharedTempDir string
	g             *genkit.Genkit

	// 测试配置
	apiKey  = "sk-6kgtZQDkmZDQMfCo28C360320cEf45FaAf1577Ef08F4032b"
	baseURL = "https://aihubmix.com/v1"
)

// TestMain 在所有测试运行前初始化共享资源
func TestMain(m *testing.M) {
	// 初始化 Genkit
	oai := &openai.OpenAI{
		APIKey: apiKey,
		Opts: []option.RequestOption{
			option.WithBaseURL(baseURL),
		},
	}

	g = genkit.Init(
		context.Background(),
		genkit.WithPlugins(oai),
	)

	// 创建共享的临时目录
	var err error
	sharedTempDir, err = os.MkdirTemp("", "research_collector_test_*")
	if err != nil {
		panic(err)
	}

	// 初始化共享依赖（只需要一次）
	// 使用同一个 ViewTool 避免重复注册
	viewTool := tools.AdaptBaseToolToGenkit(g, tools.NewViewTool(sharedTempDir))

	sharedDeps = &ResearchCollectorDependencies{
		ViewTool:            viewTool, // 复用同一个实例
		WriteTool:           tools.AdaptBaseToolToGenkit(g, tools.NewWriteTool(sharedTempDir)),
		LsTool:              tools.AdaptBaseToolToGenkit(g, tools.NewLsTool(sharedTempDir)),
		BashTool:            tools.AdaptBaseToolToGenkit(g, tools.NewBashTool(sharedTempDir)),
		DirectoryListTool:   tools.AdaptBaseToolToGenkit(g, tools.NewResourceDirectoryListTool(sharedTempDir)),
		DirectoryAddTool:    tools.AdaptBaseToolToGenkit(g, tools.NewResourceDirectoryAddTool(sharedTempDir)),
		SearchCrawlerTool:   tools.AdaptBaseToolToGenkit(g, tools.NewSearchCrawlerTool(sharedTempDir)),
		SearchNeedsAnalyzer: tools.AdaptBaseToolToGenkit(g, agent.AsToolAdapter(NewSearchNeedsAnalyzer(g))),
		ArticleEvaluator: tools.AdaptBaseToolToGenkit(g, agent.AsToolAdapter(
			NewArticleEvaluatorWithDeps(g, &ArticleEvaluatorDependencies{
				ViewTool: viewTool, // 复用同一个 ViewTool 实例
			}),
		)),
	}

	// 运行测试
	code := m.Run()

	// 清理共享目录
	if err := os.RemoveAll(sharedTempDir); err != nil {
		// 清理失败不应该影响测试结果，只记录错误
		fmt.Fprintf(os.Stderr, "Failed to cleanup temp dir: %v\n", err)
	}

	// 退出
	os.Exit(code)
}

func TestResearchCollector_NoSearchNeeded(t *testing.T) {
	// 创建 ResearchCollector
	collector := NewResearchCollectorWithDeps(g, sharedDeps)

	// 测试不需要搜索的主题
	testCases := []struct {
		name          string
		researchTopic string
		description   string
	}{
		{
			name:          "简单数学问题",
			researchTopic: "2 + 2 等于几",
			description:   "基础数学计算，不需要搜索",
		},
		{
			name:          "基础概念",
			researchTopic: "什么是变量",
			description:   "编程基础概念，知识稳定",
		},
		{
			name:          "哲学问题",
			researchTopic: "生命的意义是什么",
			description:   "主观哲学问题，不需要搜索",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 准备输入
			input := map[string]any{
				"research_topic": tc.researchTopic,
			}

			inputJSON, err := json.Marshal(input)
			require.NoError(t, err)

			// 执行收集
			ctx := context.Background()
			result, err := collector.Run(ctx, string(inputJSON))
			require.NoError(t, err)

			// 解析结果
			collectorResult, err := ParseResearchCollectorResult(result)
			require.NoError(t, err)

			// 验证结果
			require.Equal(t, "success", collectorResult.Status, "Status should be success even when no search is needed")
			require.NotEmpty(t, collectorResult.Summary, "Summary should not be empty")
			require.Empty(t, collectorResult.DirectoryName, "Directory name should be empty when no search is needed")

			// 记录结果
			t.Logf("Topic: %s", tc.researchTopic)
			t.Logf("Summary: %s", collectorResult.Summary)
			t.Logf("Description: %s", tc.description)
		})
	}
}

func TestResearchCollector_WithSearch(t *testing.T) {
	// 跳过需要网络的测试（如果设置了环境变量）
	if os.Getenv("SKIP_NETWORK_TESTS") == "true" {
		t.Skip("Skipping network-dependent test")
	}

	// 创建 ResearchCollector
	collector := NewResearchCollectorWithDeps(g, sharedDeps)

	// 测试需要搜索的主题
	testCases := []struct {
		name          string
		researchTopic string
		description   string
		checkResults  bool
	}{
		{
			name:          "技术研究主题",
			researchTopic: "Go语言中的并发模式和最佳实践",
			description:   "技术性主题，需要搜索相关资料",
			checkResults:  true,
		},
		{
			name:          "时效性主题",
			researchTopic: "2024年人工智能最新发展趋势",
			description:   "包含时效性要求，需要搜索最新信息",
			checkResults:  true,
		},
		{
			name:          "比较性研究",
			researchTopic: "React vs Vue vs Angular 前端框架对比分析",
			description:   "需要收集多个框架的信息进行比较",
			checkResults:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 准备输入
			input := map[string]any{
				"research_topic": tc.researchTopic,
			}

			inputJSON, err := json.Marshal(input)
			require.NoError(t, err)

			// 执行收集（这可能需要较长时间）
			ctx := context.Background()
			result, err := collector.Run(ctx, string(inputJSON))
			require.NoError(t, err)

			// 解析结果
			collectorResult, err := ParseResearchCollectorResult(result)
			require.NoError(t, err)

			// 验证基本结果
			require.Equal(t, "success", collectorResult.Status, "Status should be success")
			require.NotEmpty(t, collectorResult.Summary, "Summary should not be empty")
			require.NotEmpty(t, collectorResult.DirectoryName, "Directory name should not be empty when search is performed")

			// 验证目录是否创建
			if tc.checkResults && collectorResult.DirectoryName != "" {
				dirPath := filepath.Join(sharedTempDir, collectorResult.DirectoryName)
				info, err := os.Stat(dirPath)
				require.NoError(t, err, "Directory should exist")
				require.True(t, info.IsDir(), "Should be a directory")

				// 检查是否有文件被保存
				files, err := os.ReadDir(dirPath)
				require.NoError(t, err)
				t.Logf("Found %d files in directory %s", len(files), collectorResult.DirectoryName)

				// 应该至少有一些高质量文件被保留
				if len(files) > 0 {
					t.Logf("Sample files:")
					for i, file := range files {
						if i >= 3 {
							break
						}
						t.Logf("  - %s", file.Name())
					}
				}
			}

			// 记录结果
			t.Logf("Topic: %s", tc.researchTopic)
			t.Logf("Status: %s", collectorResult.Status)
			t.Logf("Directory: %s", collectorResult.DirectoryName)
			t.Logf("Summary: %s", collectorResult.Summary)
			t.Logf("Description: %s", tc.description)
		})
	}
}

func TestResearchCollector_EdgeCases(t *testing.T) {
	// 创建 ResearchCollector
	collector := NewResearchCollectorWithDeps(g, sharedDeps)

	// 测试边缘情况
	testCases := []struct {
		name          string
		researchTopic string
		expectStatus  string
		description   string
	}{
		{
			name:          "空输入",
			researchTopic: "",
			expectStatus:  "success",
			description:   "空输入应该被识别为不需要搜索",
		},
		{
			name:          "纯空格",
			researchTopic: "   ",
			expectStatus:  "success",
			description:   "纯空格应该被识别为不需要搜索",
		},
		{
			name:          "无意义符号",
			researchTopic: "@#$%^&*()",
			expectStatus:  "success",
			description:   "无意义符号应该被识别为不需要搜索",
		},
		{
			name:          "超长输入",
			researchTopic: strings.Repeat("研究主题", 100),
			expectStatus:  "success",
			description:   "超长输入的处理",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 准备输入
			input := map[string]any{
				"research_topic": tc.researchTopic,
			}

			inputJSON, err := json.Marshal(input)
			require.NoError(t, err)

			// 执行收集
			ctx := context.Background()
			result, err := collector.Run(ctx, string(inputJSON))
			require.NoError(t, err)

			// 解析结果
			collectorResult, err := ParseResearchCollectorResult(result)
			require.NoError(t, err)

			// 验证结果
			require.Equal(t, tc.expectStatus, collectorResult.Status, "Status should match expected")
			require.NotEmpty(t, collectorResult.Summary, "Summary should not be empty")

			// 记录结果
			t.Logf("Topic: %s", tc.researchTopic)
			t.Logf("Status: %s", collectorResult.Status)
			t.Logf("Summary: %s", collectorResult.Summary)
			t.Logf("Description: %s", tc.description)
		})
	}
}

func TestParseResearchCollectorResult(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		shouldError bool
		expected    *ResearchCollectorResult
	}{
		{
			name: "有效的 JSON 结果",
			input: `{
				"summary": "成功收集了5篇高质量资料",
				"directory_name": "research_ai_2024",
				"status": "success"
			}`,
			shouldError: false,
			expected: &ResearchCollectorResult{
				Summary:       "成功收集了5篇高质量资料",
				DirectoryName: "research_ai_2024",
				Status:        "success",
			},
		},
		{
			name: "带额外文本的 JSON",
			input: `分析完成，以下是结果：
			{
				"summary": "未找到相关资料",
				"directory_name": "",
				"status": "failed"
			}
			以上是分析结果。`,
			shouldError: false,
			expected: &ResearchCollectorResult{
				Summary:       "未找到相关资料",
				DirectoryName: "",
				Status:        "failed",
			},
		},
		{
			name:        "无效的 JSON",
			input:       "这不是一个有效的 JSON",
			shouldError: true,
			expected:    nil,
		},
		{
			name: "缺少必需字段",
			input: `{
				"summary": "测试总结"
			}`,
			shouldError: true,
			expected:    nil,
		},
		{
			name: "空 summary 字段",
			input: `{
				"summary": "",
				"directory_name": "test_dir",
				"status": "success"
			}`,
			shouldError: true,
			expected:    nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseResearchCollectorResult(tc.input)

			if tc.shouldError {
				require.Error(t, err, "Should return an error")
				require.Nil(t, result, "Result should be nil on error")
			} else {
				require.NoError(t, err, "Should not return an error")
				require.NotNil(t, result, "Result should not be nil")
				require.Equal(t, tc.expected.Summary, result.Summary)
				require.Equal(t, tc.expected.DirectoryName, result.DirectoryName)
				require.Equal(t, tc.expected.Status, result.Status)
			}
		})
	}
}

// TestResearchCollector_Integration 集成测试，测试完整的工作流程
func TestResearchCollector_Integration(t *testing.T) {
	// 这是一个更完整的集成测试，可以手动运行
	if os.Getenv("RUN_INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TEST=true to run")
	}

	// 创建 ResearchCollector
	collector := NewResearchCollectorWithDeps(g, sharedDeps)

	// 测试一个实际的研究主题
	researchTopic := `研究主题：深度学习在自然语言处理中的应用

需要收集以下方面的资料：
1. Transformer 架构的原理和发展
2. BERT、GPT 等预训练模型的对比
3. 最新的 NLP 应用案例
4. 未来发展趋势和挑战`

	input := map[string]any{
		"research_topic": researchTopic,
	}

	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	// 执行收集
	t.Logf("Starting research collection for topic...")
	ctx := context.Background()
	result, err := collector.Run(ctx, string(inputJSON))
	require.NoError(t, err)

	// 解析结果
	collectorResult, err := ParseResearchCollectorResult(result)
	require.NoError(t, err)

	// 输出详细结果
	t.Logf("=== Research Collection Results ===")
	t.Logf("Status: %s", collectorResult.Status)
	t.Logf("Directory: %s", collectorResult.DirectoryName)
	t.Logf("Summary:\n%s", collectorResult.Summary)

	// 检查目录内容
	if collectorResult.DirectoryName != "" {
		dirPath := filepath.Join(sharedTempDir, collectorResult.DirectoryName)
		files, err := os.ReadDir(dirPath)
		require.NoError(t, err)

		t.Logf("\n=== Collected Files (%d total) ===", len(files))
		for _, file := range files {
			info, _ := file.Info()
			t.Logf("  - %s (%.2f KB)", file.Name(), float64(info.Size())/1024)
		}
	}

	// 验证结果
	require.Equal(t, "success", collectorResult.Status, "Collection should be successful")
	require.NotEmpty(t, collectorResult.DirectoryName, "Should have created a directory")
}
