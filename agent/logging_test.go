package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"gentica/tools"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/openai/openai-go/option"
	"github.com/stretchr/testify/require"
)

// 测试 Agent 的 logging 功能
func TestAgentWithLogging(t *testing.T) {
	ctx := context.Background()

	// 初始化 Genkit
	apiKey := "sk-6kgtZQDkmZDQMfCo28C360320cEf45FaAf1577Ef08F4032b"
	baseURL := "https://aihubmix.com/v1"

	oai := &openai.OpenAI{
		APIKey: apiKey,
		Opts: []option.RequestOption{
			option.WithBaseURL(baseURL),
		},
	}

	g := genkit.Init(
		ctx,
		genkit.WithPlugins(oai),
	)

	// 创建一个简单的工具作为测试
	bashTool := tools.AdaptBaseToolToGenkit(g, tools.NewBashTool("/tmp"))

	t.Run("Agent with logging enabled", func(t *testing.T) {
		// 使用 Builder 创建一个启用 logging 的 Agent
		agent := NewBuilder(
			g,
			"BashAssistant",
			"Bash命令助手",
			"你是一个命令行助手，可以执行bash命令。",
		).
			WithTools(bashTool).
			WithLogging(true). // 启用 logging
			Build()

		// 执行查询
		input, _ := json.Marshal(map[string]any{
			"query": "请执行 echo 'Hello from Agent' 命令",
		})

		result, err := agent.Run(ctx, string(input))
		require.NoError(t, err)
		require.NotEmpty(t, result)

		t.Logf("Agent 响应: %s", result)
	})

	t.Run("Agent without logging", func(t *testing.T) {
		// 创建一个未启用 logging 的 Agent 作为对比
		agent := NewBuilder(
			g,
			"BashAssistantQuiet",
			"静默Bash助手",
			"你是一个命令行助手，可以执行bash命令。",
		).
			WithTools(bashTool).
			WithLogging(false). // 不启用 logging
			Build()

		input, _ := json.Marshal(map[string]any{
			"query": "请执行 ls 命令",
		})

		result, err := agent.Run(ctx, string(input))
		require.NoError(t, err)
		require.NotEmpty(t, result)

		t.Logf("Agent 响应（无日志）: %s", result)
	})
}

// CustomTestLogger 自定义 Logger 实现（用于测试）
type CustomTestLogger struct {
	logs []string
	t    *testing.T
}

func (tl *CustomTestLogger) Logf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	tl.logs = append(tl.logs, msg)
	tl.t.Logf("[CustomLogger] %s", msg)
}

// 测试自定义 Logger
func TestAgentWithCustomLogger(t *testing.T) {
	ctx := context.Background()

	// 初始化 Genkit
	apiKey := "sk-6kgtZQDkmZDQMfCo28C360320cEf45FaAf1577Ef08F4032b"
	baseURL := "https://aihubmix.com/v1"

	oai := &openai.OpenAI{
		APIKey: apiKey,
		Opts: []option.RequestOption{
			option.WithBaseURL(baseURL),
		},
	}

	g := genkit.Init(
		ctx,
		genkit.WithPlugins(oai),
	)

	t.Run("Agent with custom logger", func(t *testing.T) {
		testLogger := &CustomTestLogger{t: t, logs: make([]string, 0)}

		// 使用 Builder 创建 Agent，直接启用 logging
		// 注意：Agent 现在使用中间件方式，不再支持注入自定义 logger
		agent := NewBuilder(
			g,
			"CustomLogAgent",
			"自定义日志 Agent",
			"你是一个简单的助手，请回答用户的问题。",
		).
			WithLogging(true). // 启用默认日志
			Build()

		input := "你好，请介绍一下自己。"
		result, err := agent.Run(ctx, input)
		require.NoError(t, err)
		require.NotEmpty(t, result)

		// 测试直接使用中间件和自定义 logger
		loggingMiddleware := CreateConversationLogger("TestAgent", testLogger)

		// 你可以在直接调用 Generate 时使用中间件
		_, err = genkit.Generate(ctx, g,
			ai.WithModelName("openai/gpt-5-mini"),
			ai.WithSystem("你是一个助手"),
			ai.WithPrompt("你好"),
			ai.WithMiddleware(loggingMiddleware),
		)
		require.NoError(t, err)

		// 验证日志被记录
		require.NotEmpty(t, testLogger.logs)
		t.Logf("共记录了 %d 条日志", len(testLogger.logs))
	})
}

// 测试 Builder 的链式调用
func TestBuilderChaining(t *testing.T) {
	ctx := context.Background()

	// 初始化 Genkit
	apiKey := "sk-6kgtZQDkmZDQMfCo28C360320cEf45FaAf1577Ef08F4032b"
	baseURL := "https://aihubmix.com/v1"

	oai := &openai.OpenAI{
		APIKey: apiKey,
		Opts: []option.RequestOption{
			option.WithBaseURL(baseURL),
		},
	}

	g := genkit.Init(
		ctx,
		genkit.WithPlugins(oai),
	)

	// 演示完整的链式调用
	agent := NewBuilder(g, "ChainedAgent", "链式构建的Agent", "系统提示").
		WithModel("openai/gpt-5-mini").
		WithTemperature(0.5).
		WithMaxTokens(1000).
		WithMaxRounds(3).
		WithLogging(true). // 便捷地启用 logging
		WithInputSchema(
			map[string]any{
				"message": "用户消息",
			},
			"message",
		).
		Build()

	// 验证配置
	config := agent.GetConfig()
	require.Equal(t, "openai/gpt-5-mini", config.Model)
	require.Equal(t, float32(0.5), config.Temperature)
	require.Equal(t, 1000, config.MaxTokens)
	require.Equal(t, 3, config.MaxRounds)
	require.True(t, config.EnableLogging) // 验证 logging 已启用

	t.Logf("Agent 配置: %+v", config)
}