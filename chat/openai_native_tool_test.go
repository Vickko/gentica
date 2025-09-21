package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"gentica/agent"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
	oaigo "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
	"github.com/stretchr/testify/require"
)

func TestOpenAINativeToolWithMessage(t *testing.T) {
	ctx := context.Background()

	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	}

	client := oaigo.NewClient(opts...)

	getWeatherFunc := func(city string) WeatherResponse {
		return WeatherResponse{
			City:        city,
			Temperature: "25°C",
			Weather:     "晴天",
		}
	}

	params := oaigo.ChatCompletionNewParams{
		Model: shared.ChatModelGPT4oMini,
		Messages: []oaigo.ChatCompletionMessageParamUnion{
			{
				OfSystem: &oaigo.ChatCompletionSystemMessageParam{
					Content: oaigo.ChatCompletionSystemMessageParamContentUnion{
						OfString: oaigo.String("你是一个友好的天气助手。当用户询问天气时，你要先说一句友好的话，然后调用工具获取天气信息。"),
					},
				},
			},
			{
				OfUser: &oaigo.ChatCompletionUserMessageParam{
					Content: oaigo.ChatCompletionUserMessageParamContentUnion{
						OfString: oaigo.String("北京今天天气怎么样？"),
					},
				},
			},
		},
		Tools: []oaigo.ChatCompletionToolParam{
			{
				Function: shared.FunctionDefinitionParam{
					Name:        "getWeather",
					Description: oaigo.String("获取指定城市的天气信息"),
					Parameters: shared.FunctionParameters{
						"type": "object",
						"properties": map[string]interface{}{
							"city": map[string]interface{}{
								"type":        "string",
								"description": "要查询天气的城市名称",
							},
						},
						"required": []string{"city"},
					},
				},
			},
		},
		ParallelToolCalls: oaigo.Bool(true),
	}

	completion, err := client.Chat.Completions.New(ctx, params)
	require.NoError(t, err)

	if len(completion.Choices) == 0 {
		t.Fatal("No choices returned")
	}

	message := completion.Choices[0].Message
	hasTextContent := message.Content != ""
	hasToolCalls := len(message.ToolCalls) > 0

	t.Logf("════════════════════════════════════════")
	t.Logf("测试结果：模型是否能同时返回文本和工具调用")
	t.Logf("════════════════════════════════════════")
	t.Logf("✅ 包含文本内容: %v", hasTextContent)
	if hasTextContent {
		t.Logf("   文本内容: %s", message.Content)
	}
	t.Logf("✅ 包含工具调用: %v", hasToolCalls)

	if hasToolCalls {
		t.Logf("   工具调用数量: %d", len(message.ToolCalls))
		for i, toolCall := range message.ToolCalls {
			t.Logf("   工具调用 #%d:", i+1)
			t.Logf("     - ID: %s", toolCall.ID)
			t.Logf("     - 函数名: %s", toolCall.Function.Name)
			t.Logf("     - 参数: %s", toolCall.Function.Arguments)

			var args map[string]string
			err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
			if err == nil && args["city"] != "" {
				weather := getWeatherFunc(args["city"])
				t.Logf("     - 模拟返回: %+v", weather)
			}
		}
	}

	t.Logf("════════════════════════════════════════")
	t.Logf("结论: 模型%s在工具调用的同时返回文本", func() string {
		if hasTextContent && hasToolCalls {
			return "✅ 可以"
		}
		return "❌ 不能"
	}())
	t.Logf("════════════════════════════════════════")

	// 第二轮对话，处理工具响应
	if hasToolCalls && len(message.ToolCalls) > 0 {
		messages2 := []oaigo.ChatCompletionMessageParamUnion{
			{
				OfSystem: &oaigo.ChatCompletionSystemMessageParam{
					Content: oaigo.ChatCompletionSystemMessageParamContentUnion{
						OfString: oaigo.String("你是一个友好的天气助手。当用户询问天气时，你要先说一句友好的话，然后调用工具获取天气信息。"),
					},
				},
			},
			{
				OfUser: &oaigo.ChatCompletionUserMessageParam{
					Content: oaigo.ChatCompletionUserMessageParamContentUnion{
						OfString: oaigo.String("北京今天天气怎么样？"),
					},
				},
			},
		}

		// 添加 assistant 的工具调用响应
		assistantToolCalls := []oaigo.ChatCompletionMessageToolCallParam{}
		for _, toolCall := range message.ToolCalls {
			assistantToolCalls = append(assistantToolCalls, oaigo.ChatCompletionMessageToolCallParam{
				ID:   toolCall.ID,
				Type: "function",
				Function: oaigo.ChatCompletionMessageToolCallFunctionParam{
					Name:      toolCall.Function.Name,
					Arguments: toolCall.Function.Arguments,
				},
			})
		}

		messages2 = append(messages2, oaigo.ChatCompletionMessageParamUnion{
			OfAssistant: &oaigo.ChatCompletionAssistantMessageParam{
				Content: oaigo.ChatCompletionAssistantMessageParamContentUnion{
					OfString: oaigo.String(message.Content),
				},
				ToolCalls: assistantToolCalls,
			},
		})

		// 添加工具响应
		for _, toolCall := range message.ToolCalls {
			var args map[string]string
			json.Unmarshal([]byte(toolCall.Function.Arguments), &args)

			weather := getWeatherFunc(args["city"])
			weatherJSON, _ := json.Marshal(weather)

			messages2 = append(messages2, oaigo.ChatCompletionMessageParamUnion{
				OfTool: &oaigo.ChatCompletionToolMessageParam{
					Content: oaigo.ChatCompletionToolMessageParamContentUnion{
						OfString: oaigo.String(string(weatherJSON)),
					},
					ToolCallID: toolCall.ID,
				},
			})
		}

		params2 := oaigo.ChatCompletionNewParams{
			Model:    shared.ChatModelGPT4oMini,
			Messages: messages2,
		}

		completion2, err := client.Chat.Completions.New(ctx, params2)
		require.NoError(t, err)

		if len(completion2.Choices) > 0 {
			t.Logf("\n最终回复:")
			t.Logf("════════════════════════════════════════")
			t.Logf("%s", completion2.Choices[0].Message.Content)
			t.Logf("════════════════════════════════════════")
		}
	}
}

func TestOpenAINativeParallelToolCalls(t *testing.T) {
	ctx := context.Background()

	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	}

	client := oaigo.NewClient(opts...)

	getWeatherFunc := func(city string) WeatherResponse {
		return WeatherResponse{
			City:        city,
			Temperature: "25°C",
			Weather:     "晴天",
		}
	}

	params := oaigo.ChatCompletionNewParams{
		Model: shared.ChatModelGPT4oMini,
		Messages: []oaigo.ChatCompletionMessageParamUnion{
			{
				OfSystem: &oaigo.ChatCompletionSystemMessageParam{
					Content: oaigo.ChatCompletionSystemMessageParamContentUnion{
						OfString: oaigo.String("你是一个天气助手。用户询问多个城市天气时，要并行调用工具。"),
					},
				},
			},
			{
				OfUser: &oaigo.ChatCompletionUserMessageParam{
					Content: oaigo.ChatCompletionUserMessageParamContentUnion{
						OfString: oaigo.String("告诉我北京、上海和广州的天气"),
					},
				},
			},
		},
		Tools: []oaigo.ChatCompletionToolParam{
			{
				Function: shared.FunctionDefinitionParam{
					Name:        "getWeather",
					Description: oaigo.String("获取指定城市的天气信息"),
					Parameters: shared.FunctionParameters{
						"type": "object",
						"properties": map[string]interface{}{
							"city": map[string]interface{}{
								"type":        "string",
								"description": "要查询天气的城市名称",
							},
						},
						"required": []string{"city"},
					},
				},
			},
		},
		ParallelToolCalls: oaigo.Bool(true),
	}

	completion, err := client.Chat.Completions.New(ctx, params)
	require.NoError(t, err)

	if len(completion.Choices) == 0 {
		t.Fatal("No choices returned")
	}

	message := completion.Choices[0].Message

	t.Logf("════════════════════════════════════════")
	t.Logf("测试并行工具调用")
	t.Logf("════════════════════════════════════════")
	t.Logf("工具调用数量: %d", len(message.ToolCalls))

	if message.Content != "" {
		t.Logf("同时返回的文本: %s", message.Content)
	}

	var allWeatherResults []WeatherResponse
	for i, toolCall := range message.ToolCalls {
		t.Logf("\n工具调用 #%d:", i+1)
		t.Logf("  - ID: %s", toolCall.ID)
		t.Logf("  - 函数: %s", toolCall.Function.Name)
		t.Logf("  - 参数: %s", toolCall.Function.Arguments)

		var args map[string]string
		json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
		if city := args["city"]; city != "" {
			weather := getWeatherFunc(city)
			allWeatherResults = append(allWeatherResults, weather)
			t.Logf("  - 返回: %+v", weather)
		}
	}

	t.Logf("\n════════════════════════════════════════")
	t.Logf("✅ 成功并行调用 %d 个工具", len(message.ToolCalls))
	t.Logf("════════════════════════════════════════")
}

// Genkit initialization
var (
	gNative           *genkit.Genkit
	weatherToolNative ai.Tool
)

func init() {
	// Initialize Genkit if not already done
	if gNative == nil {
		oai := &openai.OpenAI{
			APIKey: apiKey,
			Opts: []option.RequestOption{
				option.WithBaseURL(baseURL),
			},
		}

		gNative = genkit.Init(
			context.Background(),
			genkit.WithPlugins(oai),
		)
	}

	// Define mock weather tool for Genkit
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

	weatherToolNative = genkit.DefineToolWithInputSchema(gNative, "getWeather", "获取指定城市的天气信息", inputSchema,
		func(ctx *ai.ToolContext, input any) (WeatherResponse, error) {
			// Parse input
			data := input.(map[string]any)
			city := data["city"].(string)

			// Return fixed mock weather data
			return WeatherResponse{
				City:        city,
				Temperature: "25°C",
				Weather:     "晴天",
			}, nil
		},
	)
}

// TestGenkitToolWithMessage 测试使用 Genkit 框架进行工具调用
func TestGenkitToolWithMessage(t *testing.T) {
	ctx := context.Background()

	// 导入 agent 包的日志中间件
	loggingMiddleware := agent.CreateConversationLogger(t)

	// 第一轮对话：调用工具（使用 WithReturnToolRequests 来手动处理工具调用）
	response, err := genkit.Generate(ctx, gNative,
		ai.WithModelName("openai/gpt-5-mini"),
		ai.WithSystem("你是一个友好的天气助手。当用户询问天气时，你**MUST**发言汇报你即将调用的工具，然后调用工具获取天气信息。"),
		ai.WithTools(weatherToolNative),
		ai.WithPrompt("北京今天天气怎么样？"),
		ai.WithReturnToolRequests(true), // 关键：不自动执行工具，只返回工具请求
		ai.WithMiddleware(loggingMiddleware), // 添加日志中间件
	)

	require.NoError(t, err)

	// 检查文本内容和工具调用
	hasTextContent := response.Text() != ""
	var hasToolCalls bool
	var toolCalls []*ai.ToolRequest

	// 从 response.Message 中提取工具调用
	if response.Message != nil {
		for _, part := range response.Message.Content {
			if part.IsToolRequest() {
				hasToolCalls = true
				toolCalls = append(toolCalls, part.ToolRequest)
			}
		}
	}

	t.Logf("════════════════════════════════════════")
	t.Logf("Genkit 测试结果：模型是否能同时返回文本和工具调用")
	t.Logf("════════════════════════════════════════")
	t.Logf("✅ 包含文本内容: %v", hasTextContent)
	if hasTextContent {
		t.Logf("   文本内容: %s", response.Text())
	}
	t.Logf("✅ 包含工具调用: %v", hasToolCalls)

	if hasToolCalls {
		t.Logf("   工具调用数量: %d", len(toolCalls))
		for i, toolReq := range toolCalls {
			t.Logf("   工具调用 #%d:", i+1)
			t.Logf("     - 工具名称: %s", toolReq.Name)
			t.Logf("     - 调用参数: %+v", toolReq.Input)

			// 模拟工具返回
			if inputMap, ok := toolReq.Input.(map[string]any); ok {
				if city, ok := inputMap["city"].(string); ok {
					weather := WeatherResponse{
						City:        city,
						Temperature: "25°C",
						Weather:     "晴天",
					}
					t.Logf("     - 模拟返回: %+v", weather)
				}
			}
		}
	}

	t.Logf("════════════════════════════════════════")
	t.Logf("结论: 模型%s在工具调用的同时返回文本", func() string {
		if hasTextContent && hasToolCalls {
			return "✅ 可以"
		}
		return "❌ 不能"
	}())
	t.Logf("════════════════════════════════════════")

	// 手动处理工具响应并进行第二轮对话
	if hasToolCalls && response.Message != nil {
		// 构建消息历史
		messages := []*ai.Message{
			ai.NewSystemTextMessage("你是一个友好的天气助手。当用户询问天气时，你要先说一句友好的话，然后调用工具获取天气信息。"),
			ai.NewUserTextMessage("北京今天天气怎么样？"),
			response.Message, // 添加包含工具调用的助手响应
		}

		// 手动执行工具并创建工具响应消息
		toolResponses := make([]*ai.Part, 0)
		for _, toolReq := range toolCalls {
			// 执行工具
			output, err := weatherToolNative.RunRaw(ctx, toolReq.Input)
			if err != nil {
				// 创建错误响应
				toolResponses = append(toolResponses, ai.NewToolResponsePart(&ai.ToolResponse{
					Name:   toolReq.Name,
					Ref:    toolReq.Ref,
					Output: fmt.Sprintf("Tool execution failed: %v", err),
				}))
			} else {
				// 创建成功响应
				toolResponses = append(toolResponses, ai.NewToolResponsePart(&ai.ToolResponse{
					Name:   toolReq.Name,
					Ref:    toolReq.Ref,
					Output: output,
				}))
				t.Logf("     - 实际执行工具返回: %+v", output)
			}
		}

		// 创建工具响应消息
		toolMessage := &ai.Message{
			Role:    ai.RoleTool,
			Content: toolResponses,
		}
		messages = append(messages, toolMessage)

		// 第二轮对话：处理工具响应
		response2, err := genkit.Generate(ctx, gNative,
			ai.WithModelName("openai/gpt-5-mini"),
			ai.WithMessages(messages...),
		)

		require.NoError(t, err)

		t.Logf("\n最终回复:")
		t.Logf("════════════════════════════════════════")
		t.Logf("%s", response2.Text())
		t.Logf("════════════════════════════════════════")
	}
}

// TestGenkitParallelToolCalls 测试使用 Genkit 框架进行并行工具调用
func TestGenkitParallelToolCalls(t *testing.T) {
	ctx := context.Background()

	// 调用模型并请求并行工具调用（使用 WithReturnToolRequests 来手动处理）
	response, err := genkit.Generate(ctx, gNative,
		ai.WithModelName("openai/gpt-5-mini"),
		ai.WithSystem("你是一个天气助手。用户询问多个城市天气时，要并行调用工具。"),
		ai.WithTools(weatherToolNative),
		ai.WithPrompt("告诉我北京、上海和广州的天气"),
		ai.WithReturnToolRequests(true), // 不自动执行工具
	)

	require.NoError(t, err)

	// 从 response.Message 中提取工具调用
	var toolCalls []*ai.ToolRequest
	if response.Message != nil {
		for _, part := range response.Message.Content {
			if part.IsToolRequest() {
				toolCalls = append(toolCalls, part.ToolRequest)
			}
		}
	}

	t.Logf("════════════════════════════════════════")
	t.Logf("Genkit 测试并行工具调用")
	t.Logf("════════════════════════════════════════")
	t.Logf("工具调用数量: %d", len(toolCalls))

	if response.Text() != "" {
		t.Logf("同时返回的文本: %s", response.Text())
	}

	var allWeatherResults []WeatherResponse
	for i, toolReq := range toolCalls {
		t.Logf("\n工具调用 #%d:", i+1)
		t.Logf("  - 工具名称: %s", toolReq.Name)
		t.Logf("  - 调用参数: %+v", toolReq.Input)

		// 执行工具并获取结果
		if inputMap, ok := toolReq.Input.(map[string]any); ok {
			if city, ok := inputMap["city"].(string); ok {
				weather := WeatherResponse{
					City:        city,
					Temperature: "25°C",
					Weather:     "晴天",
				}
				allWeatherResults = append(allWeatherResults, weather)
				t.Logf("  - 返回: %+v", weather)
			}
		}
	}

	t.Logf("\n════════════════════════════════════════")
	t.Logf("✅ 成功并行调用 %d 个工具", len(toolCalls))
	t.Logf("════════════════════════════════════════")

	// 如果有工具调用，进行第二轮对话
	if len(toolCalls) > 0 && response.Message != nil {
		// 构建消息历史
		messages := []*ai.Message{
			ai.NewSystemTextMessage("你是一个天气助手。用户询问多个城市天气时，要并行调用工具。"),
			ai.NewUserTextMessage("告诉我北京、上海和广州的天气"),
			response.Message, // 添加包含工具调用的助手响应
		}

		// 创建工具响应消息
		toolResponses := make([]*ai.Part, 0)
		for _, toolReq := range toolCalls {
			// 执行工具
			output, err := weatherToolNative.RunRaw(ctx, toolReq.Input)
			if err != nil {
				toolResponses = append(toolResponses, ai.NewToolResponsePart(&ai.ToolResponse{
					Name:   toolReq.Name,
					Ref:    toolReq.Ref,
					Output: fmt.Sprintf("Tool execution failed: %v", err),
				}))
			} else {
				toolResponses = append(toolResponses, ai.NewToolResponsePart(&ai.ToolResponse{
					Name:   toolReq.Name,
					Ref:    toolReq.Ref,
					Output: output,
				}))
			}
		}

		// 创建工具响应消息
		toolMessage := &ai.Message{
			Role:    ai.RoleTool,
			Content: toolResponses,
		}
		messages = append(messages, toolMessage)

		// 第二轮对话：获取最终回复
		response2, err := genkit.Generate(ctx, gNative,
			ai.WithModelName("openai/gpt-5-mini"),
			ai.WithMessages(messages...),
		)

		require.NoError(t, err)

		t.Logf("\n最终回复:")
		t.Logf("════════════════════════════════════════")
		t.Logf("%s", response2.Text())
		t.Logf("════════════════════════════════════════")
	}
}
