package chat

import (
	"context"
	"os"
	"testing"

	"gentica/agent"
	"gentica/tools"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
	openaiGo "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/stretchr/testify/require"
)

// WeatherResponse 天气响应结构体
type WeatherResponse struct {
	City        string `json:"city"`
	Temperature string `json:"temperature"`
	Weather     string `json:"weather"`
}

var weatherTool ai.Tool

func init() {
	// 如果 g 已经在 chat_basic_test.go 中初始化，这里只需要定义 tool
	if g == nil {
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
	}

	// 定义 mock weather tool (使用 object schema)
	inputSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"city": map[string]any{
				"type":        "string",
				"description": "要查询天气的城市名称",
			},
		},
		"required": []string{"city"},
	}

	weatherTool = genkit.DefineToolWithInputSchema(g, "getWeather", "获取指定城市的天气信息", inputSchema,
		func(ctx *ai.ToolContext, input any) (WeatherResponse, error) {
			// 解析输入
			data := input.(map[string]any)
			city := data["city"].(string)

			// 返回固定的假天气数据
			return WeatherResponse{
				City:        city,
				Temperature: "25°C",
				Weather:     "晴天",
			}, nil
		},
	)
}

func TestMockWeatherTool(t *testing.T) {
	ctx := context.Background()

	// 测试使用 weather tool 查询天气
	response, err := genkit.Generate(ctx, g,
		ai.WithModelName("openai/gpt-5-mini"),
		ai.WithSystem("你是天气助手，使用getWeather工具查询天气并用中文回复。"),
		ai.WithTools(weatherTool),
		ai.WithPrompt("北京今天天气怎么样？"),
	)

	require.NoError(t, err)
	require.NotEmpty(t, response.Text())

	// 打印工具调用信息
	toolRequests := response.ToolRequests()
	if len(toolRequests) > 0 {
		t.Logf("检测到 %d 个工具调用请求", len(toolRequests))
		for i, toolReq := range toolRequests {
			t.Logf(">>> 工具调用 #%d:", i+1)
			t.Logf("    工具名称: %s", toolReq.Name)
			t.Logf("    调用参数: %+v", toolReq.Input)
		}
	}

	// 检查消息中的工具响应
	if response.Request != nil && len(response.Request.Messages) > 0 {
		for _, msg := range response.Request.Messages {
			if msg.Role == ai.RoleTool {
				for _, part := range msg.Content {
					if part.ToolResponse != nil {
						t.Logf("<<< 工具返回结果:")
						t.Logf("    工具名称: %s", part.ToolResponse.Name)
						t.Logf("    执行结果: %+v", part.ToolResponse.Output)
					}
				}
			}
		}
	}

	t.Logf("天气查询响应: %s", response.Text())
}

func TestMockWeatherToolDirectCall(t *testing.T) {
	// 直接测试 weather tool 的功能（不通过模型）
	ctx := context.Background()

	input := map[string]any{"city": "北京"}
	resp, err := weatherTool.RunRaw(ctx, input)
	require.NoError(t, err)
	t.Logf("直接调用天气: %+v", resp)
}

// TestRealToolWithMiddlewareLogger 展示如何在真实工具场景中复用日志中间件
func TestRealToolWithMiddlewareLogger(t *testing.T) {
	ctx := context.Background()

	// 获取当前工作目录
	workingDir, err := os.Getwd()
	require.NoError(t, err)

	// 创建 tree 工具
	treeTool := tools.NewTreeTool(workingDir)
	genkitTreeTool := tools.AdaptBaseToolToGenkit(g, treeTool)

	// 使用通用的日志中间件
	loggingMiddleware := agent.CreateConversationLogger(t)

	// 初始消息
	messages := []*ai.Message{
		ai.NewUserTextMessage("请列出系统用户目录下有哪些用户的 home 目录，注意这是 macos，用户目录是/Users"),
	}

	t.Logf("╔════════════════════════════════╗")
	t.Logf("║  File System Tool with Logger  ║")
	t.Logf("╚════════════════════════════════╝")

	maxRounds := 5
	for round := 1; round <= maxRounds; round++ {
		response, err := genkit.Generate(ctx, g,
			ai.WithModelName("openai/gpt-5-mini"),
			ai.WithSystem("你是文件系统助手，请使用tree工具完成任务。"),
			ai.WithTools(genkitTreeTool),
			ai.WithMessages(messages...),
			ai.WithMiddleware(loggingMiddleware),
		)

		require.NoError(t, err)

		if response.Message != nil {
			messages = append(messages, response.Message)
		}

		if len(response.ToolRequests()) == 0 {
			t.Logf("\n✅ Task completed")
			break
		}
	}
}
