package eino

import (
	"context"
	"testing"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func TestKimiConnection(t *testing.T) {
	// 设置超时时间，防止长时间阻塞
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 配置 OpenAI 客户端，连接到 Kimi-K2-0905
	// 用户指定使用 aihubmix 的 URL 和 Key
	config := &openai.ChatModelConfig{
		BaseURL: "https://aihubmix.com/v1",
		APIKey:  "sk-6kgtZQDkmZDQMfCo28C360320cEf45FaAf1577Ef08F4032b",
		Model:   "Kimi-K2-0905", // 用户指定的模型名称
		// 如果需要调整超时或其他 HTTP 设置，可能需要查看 Config 是否支持 HTTPClient 或 Timeout 字段
		// 通常 Eino 的 Invoke 会尊崇 context 的 timeout
	}

	// 创建 ChatModel 实例
	chatModel, err := openai.NewChatModel(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create chat model: %v", err)
	}

	// 创建一个简单的图
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
		schema.UserMessage("你好，Kimi！请做一个简短的自我介绍。"),
	}

	t.Log("Sending request to Kimi-K2-0905 via aihubmix...")
	startTime := time.Now()

	// 执行图
	result, err := runnable.Invoke(ctx, messages)
	
	duration := time.Since(startTime)
	t.Logf("Request duration: %v", duration)

	if err != nil {
		// 错误处理分析
		t.Errorf("Failed to invoke graph: %v", err)
		
		// 分析是否需要其他处理方式
		t.Log("\n=== Troubleshooting Tips ===")
		t.Log("1. 如果遇到超时 (DeadlineExceeded):")
		t.Log("   - Kimi 模型可能响应较慢，尝试增加 context timeout")
		t.Log("   - 检查网络连接是否通畅")
		t.Log("2. 如果遇到 404 Model Not Found:")
		t.Log("   - 确认 'Kimi-K2-0905' 模型名称在 aihubmix 是否正确")
		t.Log("   - 尝试使用标准 Kimi 模型名如 'moonshot-v1-8k' 或 'moonshot-v1-32k'")
		t.Log("3. 如果遇到 401 Unauthorized:")
		t.Log("   - 检查 API Key 是否过期或错误")
		t.Log("4. Eino 特定处理:")
		t.Log("   - 目前 Eino 主要通过 openai-compatible 接口支持 Kimi")
		t.Log("   - 如果 openai 组件持续失败，可以检查 eino-ext 是否有更新的 provider，或者使用通用的 HTTP 请求封装")
	} else {
		// 输出结果
		t.Logf("Response: %s", result.Content)
		t.Log("Success! Kimi-K2-0905 is working with standard OpenAI component.")
	}
}

