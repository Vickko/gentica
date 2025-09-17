package chat

import (
	"context"
	"testing"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
	openaiGo "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/stretchr/testify/require"
)

var (
	// 硬编码 API 配置
	apiKey  = "sk-6kgtZQDkmZDQMfCo28C360320cEf45FaAf1577Ef08F4032b"
	baseURL = "https://aihubmix.com/v1"
)

// WeatherResponse 天气响应结构体
type WeatherResponse struct {
	City        string `json:"city"`
	Temperature string `json:"temperature"`
	Weather     string `json:"weather"`
}

var g *genkit.Genkit
var weatherTool ai.Tool

func init() {
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

func TestGenkitHelloWorld(t *testing.T) {
	// 创建测试上下文
	ctx := context.Background()

	// 使用 GPT-4o 发送真实请求
	response, err := genkit.GenerateText(ctx, g,
		ai.WithModelName("openai/"+string(openaiGo.ChatModelGPT4o)),
		ai.WithPrompt("请用中文回复：你好，世界！"),
	)

	// 验证没有错误
	require.NoError(t, err)

	// 验证响应不为空
	require.NotEmpty(t, response)

	// 记录响应
	t.Logf("GPT-4o 响应: %s", response)
}

func TestMultiTurnChatWithSystemPrompt(t *testing.T) {
	ctx := context.Background()

	// 构建聊天历史
	chatHistory := []*ai.Message{
		ai.NewUserTextMessage("什么是Goroutine？"),
		ai.NewModelTextMessage("Goroutine是Go语言的轻量级线程，由Go运行时管理。它比系统线程更轻量，可以同时运行成千上万个。"),
		ai.NewUserTextMessage("那Channel是做什么的？"),
		ai.NewModelTextMessage("Channel是Goroutine之间通信的管道，遵循'不要通过共享内存来通信，而要通过通信来共享内存'的原则。"),
	}

	t.Logf("发送前的聊天历史:")
	for i, msg := range chatHistory {
		t.Logf("  %d. [%s]: %s", i+1, msg.Role, msg.Text())
	}

	// 测试带 system prompt 和多轮对话的情况
	response, err := genkit.Generate(ctx, g,
		ai.WithModelName("openai/"+string(openaiGo.ChatModelGPT4o)),
		ai.WithSystem("你是一个专业的Go语言编程助手。请用中文回答问题，并且每次回答要简洁明了。"),
		ai.WithMessages(chatHistory...),
		ai.WithPrompt("现在请告诉我Goroutine和Channel如何配合使用？"),
	)

	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotEmpty(t, response.Text())

	t.Logf("模型响应: %s", response.Text())

	// 查看请求中包含的消息
	if response.Request != nil && response.Request.Messages != nil {
		t.Logf("请求中的消息总数: %d", len(response.Request.Messages))
		for i, msg := range response.Request.Messages {
			t.Logf("  请求消息 %d [%s]: %s", i+1, msg.Role, msg.Text())
		}
	}
}

func TestWeatherTool(t *testing.T) {
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
	t.Logf("天气查询响应: %s", response.Text())
}

func TestWeatherToolDirectCall(t *testing.T) {
	// 直接测试 weather tool 的功能（不通过模型）
	ctx := context.Background()

	input := map[string]any{"city": "北京"}
	resp, err := weatherTool.RunRaw(ctx, input)
	require.NoError(t, err)
	t.Logf("直接调用天气: %+v", resp)
}
