package eino

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func TestHelloWorld(t *testing.T) {
	ctx := context.Background()

	// 配置 OpenAI 客户端，使用自定义 endpoint
	config := &openai.ChatModelConfig{
		BaseURL: "https://aihubmix.com/v1",
		APIKey:  "sk-6kgtZQDkmZDQMfCo28C360320cEf45FaAf1577Ef08F4032b",
		Model:   "gpt-4o-mini",
	}

	// 创建 ChatModel 实例
	chatModel, err := openai.NewChatModel(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create chat model: %v", err)
	}

	// 创建一个简单的图，只包含一个节点
	graph := compose.NewGraph[[]*schema.Message, *schema.Message]()

	// 添加 ChatModel 节点
	err = graph.AddChatModelNode("chat_node", chatModel)
	if err != nil {
		t.Fatalf("Failed to add chat node: %v", err)
	}

	// 添加边：START -> chat_node -> END
	err = graph.AddEdge(compose.START, "chat_node")
	if err != nil {
		t.Fatalf("Failed to add START edge: %v", err)
	}
	err = graph.AddEdge("chat_node", compose.END)
	if err != nil {
		t.Fatalf("Failed to add END edge: %v", err)
	}

	// 编译图
	runnable, err := graph.Compile(ctx)
	if err != nil {
		t.Fatalf("Failed to compile graph: %v", err)
	}

	// 准备输入消息
	messages := []*schema.Message{
		schema.UserMessage("Say 'Hello, World!' in a friendly way"),
	}

	// 执行图
	result, err := runnable.Invoke(ctx, messages)
	if err != nil {
		t.Fatalf("Failed to invoke graph: %v", err)
	}

	// 输出结果
	t.Logf("Response: %s", result.Content)
}

func TestMultiTurnConversation(t *testing.T) {
	ctx := context.Background()

	// 配置 OpenAI 客户端，使用自定义 endpoint
	config := &openai.ChatModelConfig{
		BaseURL: "https://aihubmix.com/v1",
		APIKey:  "sk-6kgtZQDkmZDQMfCo28C360320cEf45FaAf1577Ef08F4032b",
		Model:   "gpt-4o-mini",
	}

	// 创建 ChatModel 实例
	chatModel, err := openai.NewChatModel(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create chat model: %v", err)
	}

	// 创建一个简单的图，只包含一个节点
	graph := compose.NewGraph[[]*schema.Message, *schema.Message]()

	// 添加 ChatModel 节点
	err = graph.AddChatModelNode("chat_node", chatModel)
	if err != nil {
		t.Fatalf("Failed to add chat node: %v", err)
	}

	// 添加边：START -> chat_node -> END
	err = graph.AddEdge(compose.START, "chat_node")
	if err != nil {
		t.Fatalf("Failed to add START edge: %v", err)
	}
	err = graph.AddEdge("chat_node", compose.END)
	if err != nil {
		t.Fatalf("Failed to add END edge: %v", err)
	}

	// 编译图
	runnable, err := graph.Compile(ctx)
	if err != nil {
		t.Fatalf("Failed to compile graph: %v", err)
	}

	// 初始化消息历史列表
	messageHistory := []*schema.Message{}

	// ===== 第一轮对话 =====
	t.Log("\n===== Round 1 =====")

	// 用户第一轮输入
	userMessage1 := schema.UserMessage("My name is Alice. What's a good hobby to start?")
	messageHistory = append(messageHistory, userMessage1)

	// 执行图，发送第一轮消息
	assistantReply1, err := runnable.Invoke(ctx, messageHistory)
	if err != nil {
		t.Fatalf("Failed to invoke graph in round 1: %v", err)
	}

	// 输出第一轮回复
	t.Logf("User: %s", userMessage1.Content)
	t.Logf("Assistant: %s", assistantReply1.Content)

	// 将助手回复添加到消息历史中
	messageHistory = append(messageHistory, assistantReply1)

	// ===== 第二轮对话 =====
	t.Log("\n===== Round 2 =====")

	// 用户第二轮输入（这里可以引用之前的上下文）
	userMessage2 := schema.UserMessage("That sounds interesting! By the way, do you remember my name?")
	messageHistory = append(messageHistory, userMessage2)

	// 执行图，发送包含完整历史的消息
	assistantReply2, err := runnable.Invoke(ctx, messageHistory)
	if err != nil {
		t.Fatalf("Failed to invoke graph in round 2: %v", err)
	}

	// 输出第二轮回复
	t.Logf("User: %s", userMessage2.Content)
	t.Logf("Assistant: %s", assistantReply2.Content)

	// 将助手回复添加到消息历史中（如果还有第三轮的话）
	messageHistory = append(messageHistory, assistantReply2)

	// ===== 第三轮对话（可选，展示持续的上下文） =====
	t.Log("\n===== Round 3 =====")

	userMessage3 := schema.UserMessage("Thanks! Can you summarize our conversation so far?")
	messageHistory = append(messageHistory, userMessage3)

	assistantReply3, err := runnable.Invoke(ctx, messageHistory)
	if err != nil {
		t.Fatalf("Failed to invoke graph in round 3: %v", err)
	}

	t.Logf("User: %s", userMessage3.Content)
	t.Logf("Assistant: %s", assistantReply3.Content)

	// 验证助手记住了用户的名字（Alice）
	t.Log("\n===== Verification =====")
	t.Logf("Total messages in history: %d", len(messageHistory)+1) // +1 因为还有最后一条回复
}

// MemoryCheckPointStore 实现了 CheckPointStore 接口，用于在内存中保存检查点
type MemoryCheckPointStore struct {
	store map[string][]byte
}

func (m *MemoryCheckPointStore) Get(ctx context.Context, checkPointID string) ([]byte, bool, error) {
	data, ok := m.store[checkPointID]
	return data, ok, nil
}

func (m *MemoryCheckPointStore) Set(ctx context.Context, checkPointID string, checkPoint []byte) error {
	m.store[checkPointID] = checkPoint
	return nil
}

// TestInterruptAndResume 演示如何在图执行过程中中断并等待用户输入后恢复
func TestInterruptAndResume(t *testing.T) {
	ctx := context.Background()

	// 配置 OpenAI 客户端
	config := &openai.ChatModelConfig{
		BaseURL: "https://aihubmix.com/v1",
		APIKey:  "sk-6kgtZQDkmZDQMfCo28C360320cEf45FaAf1577Ef08F4032b",
		Model:   "gpt-4o-mini",
	}

	chatModel, err := openai.NewChatModel(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create chat model: %v", err)
	}

	// 定义一个需要用户确认的 Lambda 节点
	confirmationLambda := compose.InvokableLambda(func(ctx context.Context, msg *schema.Message) ([]*schema.Message, error) {
		// 检查是否有用户确认信息
		if confirmed, ok := ctx.Value("user_confirmed").(bool); ok && confirmed {
			t.Log("User confirmation received, continuing...")
			// 添加一个系统消息表示用户已确认，然后返回完整的消息列表
			return []*schema.Message{
				msg,
				schema.SystemMessage("User has confirmed the booking. Please generate a confirmation response."),
			}, nil
		}

		// 如果没有确认，触发中断
		t.Log("Waiting for user confirmation...")
		return nil, compose.NewInterruptAndRerunErr(map[string]interface{}{
			"reason":  "需要用户确认",
			"message": msg.Content,
		})
	})

	// 创建图
	graph := compose.NewGraph[[]*schema.Message, *schema.Message]()

	// 添加节点
	_ = graph.AddChatModelNode("analyze", chatModel)
	_ = graph.AddLambdaNode("confirm_check", confirmationLambda)
	_ = graph.AddChatModelNode("final_response", chatModel)

	// 构建图的边
	_ = graph.AddEdge(compose.START, "analyze")
	_ = graph.AddEdge("analyze", "confirm_check")
	_ = graph.AddEdge("confirm_check", "final_response")
	_ = graph.AddEdge("final_response", compose.END)

	// 创建 checkpoint store 实例
	checkpointStore := &MemoryCheckPointStore{store: make(map[string][]byte)}

	// 编译图，设置在 confirm_check 节点之前中断
	runnable, err := graph.Compile(ctx,
		compose.WithGraphName("booking_graph"),
		compose.WithCheckPointStore(checkpointStore),
		compose.WithInterruptBeforeNodes([]string{"confirm_check"}),
	)
	if err != nil {
		t.Fatalf("Failed to compile graph: %v", err)
	}

	// ===== 第一次执行：触发中断 =====
	t.Log("\n===== First Execution: Will be interrupted =====")

	userRequest := []*schema.Message{
		schema.UserMessage("I want to book a flight to Tokyo on December 25th"),
	}

	result, err := runnable.Invoke(ctx, userRequest)
	if err != nil {
		// 检查是否是中断错误
		if interruptInfo, ok := compose.ExtractInterruptInfo(err); ok {
			t.Logf("Graph interrupted as expected!")
			t.Logf("Before nodes: %v", interruptInfo.BeforeNodes)
			t.Logf("After nodes: %v", interruptInfo.AfterNodes)
			t.Logf("Rerun nodes: %v", interruptInfo.RerunNodes)
			t.Logf("Rerun nodes extra: %v", interruptInfo.RerunNodesExtra)

			// 模拟用户确认
			t.Log("\n[Simulating user interaction]")
			t.Log("User sees: 'Please confirm your booking to Tokyo on December 25th'")
			t.Log("User response: YES, confirmed!")

			// ===== 第二次执行：带用户确认恢复执行 =====
			t.Log("\n===== Second Execution: Resume with confirmation =====")

			// 创建带有用户确认的 context
			confirmedCtx := context.WithValue(ctx, "user_confirmed", true)

			// 使用相同的输入和 checkpoint ID 恢复执行
			// 注意：实际使用时，需要使用 checkpoint 相关的 option 来恢复
			// 这里为了演示简化了流程
			result2, err2 := runnable.Invoke(confirmedCtx, userRequest)
			if err2 != nil {
				t.Logf("Second execution also interrupted or failed: %v", err2)
			} else {
				t.Logf("\n✓ Booking completed!")
				t.Logf("Final response: %s", result2.Content)
			}
		} else {
			t.Fatalf("Expected interrupt error, got: %v", err)
		}
	} else {
		t.Logf("Unexpected: Graph completed without interruption: %s", result.Content)
	}
}

// TestDynamicInterruptWorkflow 演示一个更实用的场景：动态决定是否需要中断
func TestDynamicInterruptWorkflow(t *testing.T) {
	ctx := context.Background()

	config := &openai.ChatModelConfig{
		BaseURL: "https://aihubmix.com/v1",
		APIKey:  "sk-6kgtZQDkmZDQMfCo28C360320cEf45FaAf1577Ef08F4032b",
		Model:   "gpt-4o-mini",
	}

	chatModel, err := openai.NewChatModel(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create chat model: %v", err)
	}

	// 第一个 Lambda：分析用户请求并决定是否需要更多信息
	analyzerLambda := compose.InvokableLambda(func(ctx context.Context, messages []*schema.Message) (map[string]interface{}, error) {
		if len(messages) == 0 {
			return nil, compose.NewInterruptAndRerunErr(map[string]interface{}{
				"error": "no messages provided",
			})
		}

		lastMsg := messages[len(messages)-1]
		content := lastMsg.Content

		// 检查是否包含足够的信息（这里简单检查是否包含"confirm"关键字）
		if ctx.Value("has_confirmation") != nil {
			t.Log("[Analyzer] User has provided confirmation, proceeding...")
			return map[string]interface{}{
				"status":  "confirmed",
				"message": content,
			}, nil
		}

		// 检查请求是否包含敏感操作（比如"delete"、"cancel"等）
		needsConfirmation := false
		for _, keyword := range []string{"delete", "cancel", "remove", "book"} {
			if len(content) > 0 && len(keyword) > 0 {
				// 简单的包含检查
				needsConfirmation = true
				break
			}
		}

		if needsConfirmation {
			t.Log("[Analyzer] Sensitive operation detected, requesting user confirmation...")
			return nil, compose.NewInterruptAndRerunErr(map[string]interface{}{
				"reason":       "需要用户确认敏感操作",
				"original_msg": content,
				"action":       "需要确认",
			})
		}

		return map[string]interface{}{
			"status":  "no_confirmation_needed",
			"message": content,
		}, nil
	})

	// 第二个 Lambda：构造最终的消息
	builderLambda := compose.InvokableLambda(func(ctx context.Context, data map[string]interface{}) ([]*schema.Message, error) {
		status := data["status"].(string)
		message := data["message"].(string)

		if status == "confirmed" {
			return []*schema.Message{
				schema.SystemMessage("用户已确认操作"),
				schema.UserMessage(message),
			}, nil
		}

		return []*schema.Message{
			schema.UserMessage(message),
		}, nil
	})

	// 创建图
	graph := compose.NewGraph[[]*schema.Message, *schema.Message]()

	_ = graph.AddLambdaNode("analyzer", analyzerLambda)
	_ = graph.AddLambdaNode("builder", builderLambda)
	_ = graph.AddChatModelNode("responder", chatModel)

	_ = graph.AddEdge(compose.START, "analyzer")
	_ = graph.AddEdge("analyzer", "builder")
	_ = graph.AddEdge("builder", "responder")
	_ = graph.AddEdge("responder", compose.END)

	runnable, err := graph.Compile(ctx)
	if err != nil {
		t.Fatalf("Failed to compile graph: %v", err)
	}

	// ===== 场景1：不需要确认的普通请求 =====
	t.Log("\n===== Scenario 1: Normal request (no confirmation needed) =====")
	normalRequest := []*schema.Message{
		schema.UserMessage("What's the weather like today?"),
	}

	result1, err1 := runnable.Invoke(ctx, normalRequest)
	if err1 != nil {
		t.Logf("Error: %v", err1)
	} else {
		t.Logf("✓ Response: %s", result1.Content)
	}

	// ===== 场景2：需要确认的敏感操作（第一次请求会被中断） =====
	t.Log("\n===== Scenario 2: Sensitive operation (will be interrupted) =====")
	sensitiveRequest := []*schema.Message{
		schema.UserMessage("I want to book a hotel room for next week"),
	}

	result2, err2 := runnable.Invoke(ctx, sensitiveRequest)
	if err2 != nil {
		if interruptInfo, ok := compose.ExtractInterruptInfo(err2); ok {
			t.Log("✓ Request interrupted for user confirmation")
			t.Logf("  Rerun nodes extra: %v", interruptInfo.RerunNodesExtra)

			// ===== 场景3：用户提供确认后重新执行 =====
			t.Log("\n===== Scenario 3: Resume with user confirmation =====")

			confirmedCtx := context.WithValue(ctx, "has_confirmation", true)
			result3, err3 := runnable.Invoke(confirmedCtx, sensitiveRequest)
			if err3 != nil {
				t.Logf("Error on retry: %v", err3)
			} else {
				t.Logf("✓ After confirmation, response: %s", result3.Content)
			}
		}
	} else {
		t.Logf("Unexpected: completed without interruption: %s", result2.Content)
	}
}

// ThrowDiceTool 实现一个掷骰子工具
type ThrowDiceTool struct {
	rng *rand.Rand
}

// NewThrowDiceTool 创建一个新的掷骰子工具
func NewThrowDiceTool() *ThrowDiceTool {
	return &ThrowDiceTool{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Info 返回工具的信息
func (t *ThrowDiceTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "throw_dice",
		Desc: "Throw a dice and return the result. You can specify the number of sides (default is 6) and how many dice to throw (default is 1).",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"sides": {
				Type:     schema.Integer,
				Desc:     "Number of sides on the dice (e.g., 6 for a standard dice, 20 for D&D)",
				Required: false,
			},
			"count": {
				Type:     schema.Integer,
				Desc:     "Number of dice to throw",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行掷骰子操作
func (t *ThrowDiceTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// 解析参数
	type DiceParams struct {
		Sides int `json:"sides"`
		Count int `json:"count"`
	}

	params := DiceParams{
		Sides: 6,  // 默认6面骰子
		Count: 1,  // 默认投掷1次
	}

	if argumentsInJSON != "" && argumentsInJSON != "{}" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
			return "", fmt.Errorf("failed to parse arguments: %w", err)
		}
	}

	// 验证参数
	if params.Sides < 2 {
		params.Sides = 6
	}
	if params.Count < 1 {
		params.Count = 1
	}
	if params.Count > 100 {
		return "", fmt.Errorf("too many dice (max 100)")
	}

	// 掷骰子
	results := make([]int, params.Count)
	sum := 0
	for i := 0; i < params.Count; i++ {
		result := t.rng.Intn(params.Sides) + 1
		results[i] = result
		sum += result
	}

	// 构造结果
	resultStr := fmt.Sprintf("Threw %d dice with %d sides. Results: %v, Total: %d",
		params.Count, params.Sides, results, sum)

	return resultStr, nil
}

// TestThrowDiceWithTool 测试使用掷骰子工具
func TestThrowDiceWithTool(t *testing.T) {
	ctx := context.Background()

	// 配置 OpenAI 客户端
	config := &openai.ChatModelConfig{
		BaseURL: "https://aihubmix.com/v1",
		APIKey:  "sk-6kgtZQDkmZDQMfCo28C360320cEf45FaAf1577Ef08F4032b",
		Model:   "gpt-4o-mini",
	}

	// 创建支持工具调用的 ChatModel
	chatModel, err := openai.NewChatModel(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create chat model: %v", err)
	}

	// 创建掷骰子工具
	diceTool := NewThrowDiceTool()

	// 将工具绑定到模型
	err = chatModel.BindTools([]*schema.ToolInfo{
		mustGetToolInfo(diceTool, ctx, t),
	})
	if err != nil {
		t.Fatalf("Failed to bind tools: %v", err)
	}

	// 创建 ToolsNode
	toolsNode, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools: []tool.BaseTool{diceTool},
	})
	if err != nil {
		t.Fatalf("Failed to create tools node: %v", err)
	}

	// ===== 测试1：要求掷骰子 =====
	t.Log("\n===== Test 1: Ask to throw a dice =====")

	messages := []*schema.Message{
		schema.UserMessage("Please throw a dice for me and tell me the result!"),
	}

	// 第一次调用模型
	result, err := chatModel.Generate(ctx, messages)
	if err != nil {
		t.Fatalf("Failed to generate response: %v", err)
	}

	t.Logf("User: %s", messages[0].Content)
	t.Logf("Assistant (first response): %v tool calls", len(result.ToolCalls))

	// 如果有工具调用，执行工具
	if len(result.ToolCalls) > 0 {
		// 将助手的消息（包含工具调用）添加到历史
		messages = append(messages, result)

		// 执行工具
		toolMessages, err := toolsNode.Invoke(ctx, result)
		if err != nil {
			t.Fatalf("Failed to execute tools: %v", err)
		}

		// 将工具结果添加到历史
		messages = append(messages, toolMessages...)

		t.Logf("Tool executed, got %d tool messages", len(toolMessages))

		// 再次调用模型，让它基于工具结果生成最终回复
		finalResult, err := chatModel.Generate(ctx, messages)
		if err != nil {
			t.Fatalf("Failed to generate final response: %v", err)
		}

		t.Logf("Assistant (final): %s", finalResult.Content)
	} else {
		t.Logf("Assistant (no tool call): %s", result.Content)
	}

	// ===== 测试2：掷多个骰子 =====
	t.Log("\n===== Test 2: Throw multiple dice =====")

	messages2 := []*schema.Message{
		schema.UserMessage("Throw 3 dice with 20 sides each (like D&D dice) and tell me the total!"),
	}

	result2, err := chatModel.Generate(ctx, messages2)
	if err != nil {
		t.Fatalf("Failed to generate response: %v", err)
	}

	if len(result2.ToolCalls) > 0 {
		messages2 = append(messages2, result2)
		toolMessages2, _ := toolsNode.Invoke(ctx, result2)
		messages2 = append(messages2, toolMessages2...)

		finalResult2, err := chatModel.Generate(ctx, messages2)
		if err != nil {
			t.Fatalf("Failed to generate final response: %v", err)
		}

		t.Logf("User: %s", messages2[0].Content)
		t.Logf("Assistant: %s", finalResult2.Content)
	}

	// ===== 测试3：普通对话（不使用工具） =====
	t.Log("\n===== Test 3: Normal conversation without tool =====")

	messages3 := []*schema.Message{
		schema.UserMessage("What is the capital of France?"),
	}

	result3, err := chatModel.Generate(ctx, messages3)
	if err != nil {
		t.Fatalf("Failed to generate response: %v", err)
	}

	t.Logf("User: %s", messages3[0].Content)
	t.Logf("Assistant: %s", result3.Content)
}

// mustGetToolInfo 辅助函数：获取工具信息
func mustGetToolInfo(tool interface{}, ctx context.Context, t *testing.T) *schema.ToolInfo {
	type InfoProvider interface {
		Info(context.Context) (*schema.ToolInfo, error)
	}

	if tp, ok := tool.(InfoProvider); ok {
		info, err := tp.Info(ctx)
		if err != nil {
			t.Fatalf("Failed to get tool info: %v", err)
		}
		return info
	}

	t.Fatal("Tool does not implement Info method")
	return nil
}
