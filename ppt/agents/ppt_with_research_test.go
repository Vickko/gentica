package agents

import (
	"context"
	"testing"

	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/joho/godotenv"
	"github.com/openai/openai-go/option"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPPTWithResearchIntegration 测试包含研究资料收集的完整PPT生成流程
func TestPPTWithResearchIntegration(t *testing.T) {
	// 加载环境变量
	err := godotenv.Load("../../.env")
	if err != nil {
		t.Logf("Warning: .env file not found, using existing environment variables")
	}

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
	tempDir := t.TempDir()
	t.Logf("Test directory: %s", tempDir)

	// 创建带有研究收集功能的 PPT Top Agent
	pptTopAgent := NewPPTTopAgent(g, tempDir)
	require.NotNil(t, pptTopAgent)

	t.Run("Test PPT Generation With Research Collection", func(t *testing.T) {
		ctx := context.Background()

		// 用户请求 - 云原生架构主题
		userRequest := `{"request": "请帮我制作一个关于云原生架构最佳实践的PPT，包含容器化、微服务、DevOps等内容"}`

		t.Logf("Starting PPT generation with research collection...")
		t.Logf("Request: %s", userRequest)

		// 执行 Agent
		result, err := pptTopAgent.Run(ctx, userRequest)
		require.NoError(t, err)
		assert.NotEmpty(t, result)

		t.Logf("PPT Top Agent Result:\n%s", result)

		// 解析结果
		topResult, err := ParsePPTTopResult(result)
		assert.NoError(t, err)

		// 验证结果包含必要信息
		if topResult.Status == "success" {
			// 验证研究资料目录
			if topResult.ResearchDirectory != "" {
				t.Logf("✓ Research directory: %s", topResult.ResearchDirectory)
			} else {
				t.Log("⚠ No research directory (may not need external research)")
			}

			// 验证大纲文件
			if topResult.OutlineFilePath != "" {
				t.Logf("✓ Outline file: %s", topResult.OutlineFilePath)
			}

			// 验证模板目录
			if topResult.TemplateDirectory != "" {
				t.Logf("✓ Template directory: %s", topResult.TemplateDirectory)
			}

			// 验证生成的页面
			if len(topResult.GeneratedPages) > 0 {
				t.Logf("✓ Generated %d pages", len(topResult.GeneratedPages))
				for i, page := range topResult.GeneratedPages {
					t.Logf("  Page %d: %s", i+1, page)
				}
			}

			// 打印总结
			t.Logf("\n=== Summary ===\n%s", topResult.Summary)
		} else {
			t.Logf("Generation failed with status: %s", topResult.Status)
		}
	})
}

// TestSimplePPTWithResearch 测试简单主题的PPT生成（带研究资料）
func TestSimplePPTWithResearch(t *testing.T) {
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
	tempDir := t.TempDir()

	// 创建 PPT Top Agent
	pptTopAgent := NewPPTTopAgent(g, tempDir)

	ctx := context.Background()

	// 简单的测试请求
	userRequest := `{"request": "制作一个关于AI在教育领域应用的简单PPT，5页即可"}`

	t.Logf("Starting simplified PPT generation...")

	// 执行
	result, err := pptTopAgent.Run(ctx, userRequest)
	require.NoError(t, err)

	// 解析并验证
	topResult, err := ParsePPTTopResult(result)
	assert.NoError(t, err)
	assert.Equal(t, "success", topResult.Status)

	t.Logf("Simplified test completed successfully")
}