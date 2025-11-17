package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"gentica1/rp"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "rp/examples/phoebe.yaml", "")
	apiKey := flag.String("api-key", "sk-6kgtZQDkmZDQMfCo28C360320cEf45FaAf1577Ef08F4032b", "")
	baseURL := flag.String("base-url", "https://aihubmix.com/v1", "")
	model := flag.String("model", "DeepSeek-V3.1-Terminus", "")
	flag.Parse()

	// 验证配置文件路径
	if *configPath == "" {
		fmt.Fprintf(os.Stderr, "错误: 必须指定配置文件路径\n")
		fmt.Fprintf(os.Stderr, "用法: go run main.go -config <配置文件路径>\n")
		os.Exit(1)
	}

	// 加载角色配置
	roleConfig, err := rp.LoadRoleConfig(*configPath)
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 获取 API Key
	finalAPIKey := *apiKey
	if finalAPIKey == "" {
		log.Fatal("错误: 必须提供 API Key（通过 -api-key 参数或 OPENAI_API_KEY 环境变量）")
	}

	// 创建上下文
	ctx := context.Background()

	// 使用 factory 创建 ChatModel（根据 model 名称自动选择对应的 client）
	chatModel, err := rp.CreateChatModel(ctx, &rp.ModelClientConfig{
		Model:   *model,
		APIKey:  finalAPIKey,
		BaseURL: *baseURL,
	})
	if err != nil {
		log.Fatalf("创建 ChatModel 失败: %v", err)
	}

	// 创建 Agent（使用 Instruction 字段管理 system prompt）
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          roleConfig.Name,
		Description:   roleConfig.Name, // 使用 Name 作为 Description（必填字段）
		Instruction:   roleConfig.SystemPrompt,
		Model:         chatModel,
		MaxIterations: 10,
	})
	if err != nil {
		log.Fatalf("创建 Agent 失败: %v", err)
	}

	// 打印欢迎信息
	fmt.Println("=== Role Play Chat ===")
	fmt.Printf("角色: %s\n", roleConfig.Name)
	fmt.Println("======================")
	fmt.Println("输入 'exit' 或 'quit' 退出对话")
	fmt.Println()

	// 维护对话历史
	messageHistory := []adk.Message{}

	// 创建 stdin 读取器
	reader := bufio.NewReader(os.Stdin)

	// 主循环
	for {
		fmt.Print("你: ")
		userInput, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("读取输入失败: %v", err)
		}

		userInput = strings.TrimSpace(userInput)

		// 检查退出命令
		if userInput == "exit" || userInput == "quit" {
			fmt.Println("再见！")
			break
		}

		// 跳过空输入
		if userInput == "" {
			continue
		}

		// 添加用户消息到历史
		userMessage := schema.UserMessage(userInput)
		messageHistory = append(messageHistory, userMessage)

		// 创建 Agent 输入
		agentInput := &adk.AgentInput{
			Messages: messageHistory,
		}

		// 运行 Agent
		iter := agent.Run(ctx, agentInput)

		// 处理 Agent 输出
		fmt.Printf("%s: ", roleConfig.Name)
		var assistantMessage *schema.Message
		for {
			event, ok := iter.Next()
			if !ok {
				break
			}

			if event.Err != nil {
				log.Printf("Agent 错误: %v", event.Err)
				break
			}

			if event.Output != nil && event.Output.MessageOutput != nil {
				msgVariant := event.Output.MessageOutput
				msg, err := msgVariant.GetMessage()
				if err != nil {
					log.Printf("获取消息失败: %v", err)
					continue
				}

				if msg.Role == schema.Assistant && msg.Content != "" {
					assistantMessage = msg
				}
			}
		}

		if assistantMessage != nil {
			fmt.Println(assistantMessage.Content)
			// 将助手回复添加到历史
			messageHistory = append(messageHistory, assistantMessage)
		}
		fmt.Println()
	}
}
