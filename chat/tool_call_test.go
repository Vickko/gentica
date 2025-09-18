package chat

import (
	"context"
	"os"
	"testing"

	"gentica/tools"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
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
		ai.WithModelName("openai/"+string(openaiGo.ChatModelGPT4o)),
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

// createConversationLogger 创建一个简洁的对话日志中间件
func createConversationLogger(t *testing.T) func(core.StreamingFunc[*ai.ModelRequest, *ai.ModelResponse, *ai.ModelResponseChunk]) core.StreamingFunc[*ai.ModelRequest, *ai.ModelResponse, *ai.ModelResponseChunk] {
	var roundCounter int
	var lastMessageCount int

	return func(next core.StreamingFunc[*ai.ModelRequest, *ai.ModelResponse, *ai.ModelResponseChunk]) core.StreamingFunc[*ai.ModelRequest, *ai.ModelResponse, *ai.ModelResponseChunk] {
		return func(ctx context.Context, req *ai.ModelRequest, cb core.StreamCallback[*ai.ModelResponseChunk]) (*ai.ModelResponse, error) {
			roundCounter++

			// ========== 请求阶段 ==========
			t.Logf("\n━━━ Round %d: Request ━━━", roundCounter)

			// 只打印新增的消息（相比上一轮）
			currentMessageCount := len(req.Messages)
			newMessages := req.Messages[lastMessageCount:]

			// 首轮打印系统提示
			if roundCounter == 1 && len(req.Messages) > 0 {
				for _, msg := range req.Messages {
					if msg.Role == ai.RoleSystem {
						t.Logf("📋 System: %s", msg.Text())
						break
					}
				}
			}

			// 打印新增消息
			for _, msg := range newMessages {
				switch msg.Role {
				case ai.RoleUser:
					t.Logf("👤 User: %s", msg.Text())
				case ai.RoleTool:
					// 工具响应
					for _, part := range msg.Content {
						if part.IsToolResponse() {
							t.Logf("🔧 Tool Response [%s]: %v",
								part.ToolResponse.Name,
								part.ToolResponse.Output)
						}
					}
				}
			}

			// 首轮打印可用工具
			if roundCounter == 1 && len(req.Tools) > 0 {
				t.Logf("🛠️  Available Tools:")
				for _, tool := range req.Tools {
					t.Logf("   • %s: %s", tool.Name, tool.Description)
				}
			}

			// ========== 执行请求 ==========
			resp, err := next(ctx, req, cb)
			if err != nil {
				t.Logf("❌ Error: %v", err)
				return nil, err
			}

			// ========== 响应阶段 ==========
			t.Logf("━━━ Round %d: Response ━━━", roundCounter)

			if resp != nil && resp.Message != nil {
				// 检查响应内容类型
				hasToolCall := false
				hasText := false

				for _, part := range resp.Message.Content {
					if part.IsToolRequest() {
						hasToolCall = true
					}
					if part.IsText() && part.Text != "" {
						hasText = true
					}
				}

				// 根据内容类型打印
				if hasToolCall {
					t.Logf("🔨 Tool Calls:")
					for _, part := range resp.Message.Content {
						if part.IsToolRequest() {
							t.Logf("   → %s(%v)",
								part.ToolRequest.Name,
								part.ToolRequest.Input)
						}
					}
				}

				if hasText {
					t.Logf("🤖 Assistant: %s", resp.Message.Text())
				}

				// 更新消息计数（包含模型响应）
				lastMessageCount = currentMessageCount + 1
			}

			// Token 使用情况
			if resp != nil && resp.Usage != nil {
				t.Logf("📊 Tokens: input=%d, output=%d",
					resp.Usage.InputTokens,
					resp.Usage.OutputTokens)
			}

			return resp, nil
		}
	}
}

// TestRealToolWithMiddlewareLogger 展示如何在真实工具场景中复用日志中间件
func TestRealToolWithMiddlewareLogger(t *testing.T) {
	ctx := context.Background()

	// 获取当前工作目录
	workingDir, err := os.Getwd()
	require.NoError(t, err)

	// 创建 ls 工具
	lsTool := tools.NewLsTool(workingDir)
	genkitLsTool := tools.AdaptBaseToolToGenkit(g, lsTool)

	// 复用日志中间件
	loggingMiddleware := createConversationLogger(t)

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
			ai.WithModelName("openai/"+string(openaiGo.ChatModelGPT4o)),
			ai.WithSystem("你是文件系统助手，请使用ls工具完成任务。"),
			ai.WithTools(genkitLsTool),
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
