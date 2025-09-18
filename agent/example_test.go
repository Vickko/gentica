package agent_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"gentica/agent"
	"gentica/tools"

	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
	openaiGo "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// ExampleNewSimpleAgent 展示如何创建和使用基本 Agent
func ExampleNewSimpleAgent() {
	// 初始化 Genkit
	oai := &openai.OpenAI{
		APIKey: "your-api-key",
		Opts: []option.RequestOption{
			option.WithBaseURL("https://api.openai.com/v1"),
		},
	}

	g := genkit.Init(
		context.Background(),
		genkit.WithPlugins(oai),
	)

	// 创建一个简单的 Agent
	assistant := agent.NewSimpleAgent(
		g,
		"assistant",
		"You are a helpful assistant. Answer concisely.",
	)

	// 使用 Agent
	ctx := context.Background()
	result, err := assistant.Run(ctx, "What is the capital of France?")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Answer:", result)
}

// ExampleNewBuilder 展示如何创建带输入 schema 的 Agent
func ExampleNewBuilder() {
	// 初始化 Genkit
	oai := &openai.OpenAI{
		APIKey: "your-api-key",
		Opts: []option.RequestOption{
			option.WithBaseURL("https://api.openai.com/v1"),
		},
	}

	g := genkit.Init(
		context.Background(),
		genkit.WithPlugins(oai),
	)

	// 创建带输入 schema 的 Agent
	translator := agent.NewBuilder(
		g,
		"translator",
		"Translates text between languages",
		"You are a professional translator. Translate the text to the target language accurately.",
	).WithInputSchema(
		map[string]any{
			"text": map[string]any{
				"type":        "string",
				"description": "Text to translate",
			},
			"target_language": map[string]any{
				"type":        "string",
				"description": "Target language for translation",
			},
		},
		"text", "target_language", // 必需字段
	).WithModel("openai/" + string(openaiGo.ChatModelGPT4oMini)).
		Build()

	// 准备输入
	input := map[string]any{
		"text":            "Hello, world!",
		"target_language": "Spanish",
	}

	inputJSON, _ := json.Marshal(input)

	// 使用 Agent
	ctx := context.Background()
	result, err := translator.Run(ctx, string(inputJSON))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Translation:", result)
}

// ExampleAsToolAdapter 展示如何将 Agent 作为工具在另一个 Agent 中使用
func ExampleAsToolAdapter() {
	// 初始化 Genkit
	oai := &openai.OpenAI{
		APIKey: "your-api-key",
		Opts: []option.RequestOption{
			option.WithBaseURL("https://api.openai.com/v1"),
		},
	}

	g := genkit.Init(
		context.Background(),
		genkit.WithPlugins(oai),
	)

	// 创建专门的翻译 Agent
	translatorAgent := agent.NewToolAgent(
		g,
		"translator_tool",
		"Translates text to any language",
		map[string]any{
			"text": map[string]any{
				"type":        "string",
				"description": "Text to translate",
			},
			"language": map[string]any{
				"type":        "string",
				"description": "Target language",
			},
		},
		[]string{"text", "language"},
		"You are a translator. Translate the text to the specified language.",
	)

	// 将翻译 Agent 转换为工具
	translatorTool := agent.AsToolAdapter(translatorAgent)
	genkitTranslatorTool := tools.AdaptBaseToolToGenkit(g, translatorTool)

	// 创建主 Agent，使用翻译工具
	mainAgent := agent.NewBuilder(
		g,
		"multilingual_assistant",
		"Assists users in multiple languages",
		"You are a multilingual assistant. Use the translator_tool when needed to communicate in different languages.",
	).WithTools(genkitTranslatorTool).
		Build()

	// 使用主 Agent
	ctx := context.Background()
	result, err := mainAgent.Run(ctx, "Translate 'Good morning' to French, Spanish, and German")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Result:", result)
}

// ExampleNewAgentChain 展示如何串行执行多个 Agent
func ExampleNewAgentChain() {
	// 初始化 Genkit
	oai := &openai.OpenAI{
		APIKey: "your-api-key",
		Opts: []option.RequestOption{
			option.WithBaseURL("https://api.openai.com/v1"),
		},
	}

	g := genkit.Init(
		context.Background(),
		genkit.WithPlugins(oai),
	)

	// 创建多个专门的 Agent
	summarizer := agent.NewSimpleAgent(
		g,
		"summarizer",
		"You are a summarizer. Summarize the input text in 2-3 sentences.",
	)

	translator := agent.NewSimpleAgent(
		g,
		"translator",
		"You are a translator. Translate the input to Chinese.",
	)

	formatter := agent.NewSimpleAgent(
		g,
		"formatter",
		"You are a formatter. Format the input as a bullet point list.",
	)

	// 创建 Agent 链
	chain := agent.NewAgentChain(
		"process_chain",
		summarizer,
		translator,
		formatter,
	)

	// 使用链处理文本
	ctx := context.Background()
	longText := `
		Artificial intelligence (AI) is intelligence demonstrated by machines,
		in contrast to the natural intelligence displayed by humans and animals.
		Leading AI textbooks define the field as the study of "intelligent agents":
		any device that perceives its environment and takes actions that maximize
		its chance of successfully achieving its goals.
	`

	result, err := chain.Run(ctx, longText)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Processed result:", result)
}

// ExampleNewAgentPool 展示如何并行执行多个 Agent
func ExampleNewAgentPool() {
	// 初始化 Genkit
	oai := &openai.OpenAI{
		APIKey: "your-api-key",
		Opts: []option.RequestOption{
			option.WithBaseURL("https://api.openai.com/v1"),
		},
	}

	g := genkit.Init(
		context.Background(),
		genkit.WithPlugins(oai),
	)

	// 创建多个分析 Agent
	technicalAnalyzer := agent.NewSimpleAgent(
		g,
		"technical_analyzer",
		"You are a technical analyst. Analyze the technical aspects of the topic.",
	)

	businessAnalyzer := agent.NewSimpleAgent(
		g,
		"business_analyzer",
		"You are a business analyst. Analyze the business implications of the topic.",
	)

	riskAnalyzer := agent.NewSimpleAgent(
		g,
		"risk_analyzer",
		"You are a risk analyst. Identify potential risks and challenges.",
	)

	// 创建 Agent 池
	pool := agent.NewAgentPool(
		"analysis_pool",
		technicalAnalyzer,
		businessAnalyzer,
		riskAnalyzer,
	)

	// 并行分析
	ctx := context.Background()
	topic := "Implementing blockchain technology in supply chain management"

	// 获取详细结果
	results := pool.RunWithDetails(ctx, topic)

	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("%s failed: %v\n", result.AgentName, result.Error)
		} else {
			fmt.Printf("%s analysis:\n%s\n\n", result.AgentName, result.Result)
		}
	}
}

// ExampleAgent_ClearHistory 展示如何管理 Agent 的消息历史
func ExampleAgent_ClearHistory() {
	// 初始化 Genkit
	oai := &openai.OpenAI{
		APIKey: "your-api-key",
		Opts: []option.RequestOption{
			option.WithBaseURL("https://api.openai.com/v1"),
		},
	}

	g := genkit.Init(
		context.Background(),
		genkit.WithPlugins(oai),
	)

	// 创建一个会话 Agent
	chatAgent := agent.NewSimpleAgent(
		g,
		"chat_assistant",
		"You are a helpful chat assistant. Remember the conversation context.",
	)

	ctx := context.Background()

	// 第一轮对话
	_, err := chatAgent.Run(ctx, "My name is Alice and I love programming.")
	if err != nil {
		log.Fatal(err)
	}

	// 第二轮对话（Agent 记住了之前的内容）
	response, err := chatAgent.Run(ctx, "What's my name and what do I love?")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Agent remembers:", response)

	// 清空历史
	chatAgent.ClearHistory()

	// 新对话（Agent 忘记了之前的内容）
	response, err = chatAgent.Run(ctx, "Do you know my name?")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("After clearing history:", response)
}

// ExampleNewAgentRouter 展示如何根据条件路由到不同的 Agent
func ExampleNewAgentRouter() {
	// 初始化 Genkit
	oai := &openai.OpenAI{
		APIKey: "your-api-key",
		Opts: []option.RequestOption{
			option.WithBaseURL("https://api.openai.com/v1"),
		},
	}

	g := genkit.Init(
		context.Background(),
		genkit.WithPlugins(oai),
	)

	// 创建专门的 Agent
	mathAgent := agent.NewSimpleAgent(
		g,
		"math_expert",
		"You are a mathematics expert. Solve math problems step by step.",
	)

	codeAgent := agent.NewSimpleAgent(
		g,
		"code_expert",
		"You are a programming expert. Help with coding questions.",
	)

	generalAgent := agent.NewSimpleAgent(
		g,
		"general_assistant",
		"You are a general assistant. Answer various questions.",
	)

	// 创建路由函数
	routerFunc := func(ctx context.Context, input string) (string, error) {
		// 简单的关键词匹配路由
		if containsKeywords(input, "math", "calculate", "equation", "solve") {
			return "math", nil
		}
		if containsKeywords(input, "code", "program", "function", "debug") {
			return "code", nil
		}
		return "general", nil
	}

	// 创建路由器
	router := agent.NewAgentRouter("smart_router", routerFunc)
	router.AddRoute("math", mathAgent)
	router.AddRoute("code", codeAgent)
	router.AddRoute("general", generalAgent)

	ctx := context.Background()

	// 测试不同类型的问题
	questions := []string{
		"What is the derivative of x^2 + 3x?",
		"How do I write a for loop in Go?",
		"What's the weather like today?",
	}

	for _, q := range questions {
		result, err := router.Run(ctx, q)
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}
		fmt.Printf("Q: %s\nA: %s\n\n", q, result)
	}
}

// 辅助函数
func containsKeywords(text string, keywords ...string) bool {
	for _, keyword := range keywords {
		// 简单的子串匹配
		if len(text) >= len(keyword) {
			for i := 0; i <= len(text)-len(keyword); i++ {
				if text[i:i+len(keyword)] == keyword {
					return true
				}
			}
		}
	}
	return false
}
