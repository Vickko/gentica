package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"gentica1/rp"
)

// loadMarkdownFile 读取 Markdown 文件内容
func loadMarkdownFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取文件失败 %s: %w", path, err)
	}
	return string(data), nil
}

// AgentContext 封装 Agent 的上下文信息
type AgentContext struct {
	Name          string
	Agent         *adk.ChatModelAgent
	MessageHistory []adk.Message
}

// DialogueRecord 对话记录
type DialogueRecord struct {
	Round   int    // 第几轮
	Speaker string // 发言者
	Message string // 消息内容
}

func main() {
	// API 配置
	apiKey := "sk-6kgtZQDkmZDQMfCo28C360320cEf45FaAf1577Ef08F4032b"
	baseURL := "https://aihubmix.com/v1"
	model := "DeepSeek-V3.1-Terminus"

	// 文件路径配置
	zaniConfigPath := "rp/examples/zani.yaml"
	carlottaConfigPath := "rp/examples/carlotta.yaml"
	zaniInnerWorldPath := "docs/wutheringwaves/character-inner-world-zani.md"
	carlottaInnerWorldPath := "docs/wutheringwaves/character-inner-world-carlotta.md"

	ctx := context.Background()

	// 1. 加载赞妮的配置和内心世界
	fmt.Println("正在加载赞妮的配置...")
	zaniConfig, err := rp.LoadRoleConfig(zaniConfigPath)
	if err != nil {
		log.Fatalf("加载赞妮配置失败: %v", err)
	}

	zaniInnerWorld, err := loadMarkdownFile(zaniInnerWorldPath)
	if err != nil {
		log.Fatalf("加载赞妮内心世界失败: %v", err)
	}

	// 合并系统提示
	zaniSystemPrompt := zaniConfig.SystemPrompt + "\n\n---\n\n# 当前时刻的内心世界状态\n\n" + zaniInnerWorld

	// 2. 加载珂莱塔的配置和内心世界
	fmt.Println("正在加载珂莱塔的配置...")
	carlottaConfig, err := rp.LoadRoleConfig(carlottaConfigPath)
	if err != nil {
		log.Fatalf("加载珂莱塔配置失败: %v", err)
	}

	carlottaInnerWorld, err := loadMarkdownFile(carlottaInnerWorldPath)
	if err != nil {
		log.Fatalf("加载珂莱塔内心世界失败: %v", err)
	}

	// 合并系统提示
	carlottaSystemPrompt := carlottaConfig.SystemPrompt + "\n\n---\n\n# 当前时刻的内心世界状态\n\n" + carlottaInnerWorld

	// 3. 创建赞妮的 ChatModel 和 Agent
	fmt.Println("正在创建赞妮的 Agent...")
	zaniChatModel, err := rp.CreateChatModel(ctx, &rp.ModelClientConfig{
		Model:   model,
		APIKey:  apiKey,
		BaseURL: baseURL,
	})
	if err != nil {
		log.Fatalf("创建赞妮的 ChatModel 失败: %v", err)
	}

	zaniAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          zaniConfig.Name,
		Description:   zaniConfig.Name,
		Instruction:   zaniSystemPrompt,
		Model:         zaniChatModel,
		MaxIterations: 10,
	})
	if err != nil {
		log.Fatalf("创建赞妮的 Agent 失败: %v", err)
	}

	// 4. 创建珂莱塔的 ChatModel 和 Agent
	fmt.Println("正在创建珂莱塔的 Agent...")
	carlottaChatModel, err := rp.CreateChatModel(ctx, &rp.ModelClientConfig{
		Model:   model,
		APIKey:  apiKey,
		BaseURL: baseURL,
	})
	if err != nil {
		log.Fatalf("创建珂莱塔的 ChatModel 失败: %v", err)
	}

	carlottaAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          carlottaConfig.Name,
		Description:   carlottaConfig.Name,
		Instruction:   carlottaSystemPrompt,
		Model:         carlottaChatModel,
		MaxIterations: 10,
	})
	if err != nil {
		log.Fatalf("创建珂莱塔的 Agent 失败: %v", err)
	}

	// 5. 初始化 Agent 上下文
	zaniCtx := &AgentContext{
		Name:           zaniConfig.Name,
		Agent:          zaniAgent,
		MessageHistory: []adk.Message{},
	}

	carlottaCtx := &AgentContext{
		Name:           carlottaConfig.Name,
		Agent:          carlottaAgent,
		MessageHistory: []adk.Message{},
	}

	// 6. 对话记录
	dialogueRecords := []DialogueRecord{}

	// 7. 开始对话循环（32轮）
	fmt.Println("\n=== 开始对话实验 ===")
	fmt.Println("珂莱塔将首先发言（根据场景设定，她推门而入找赞妮）\n")

	const maxRounds = 32
	var currentSpeaker *AgentContext
	var currentListener *AgentContext

	// 珂莱塔先开始
	currentSpeaker = carlottaCtx
	currentListener = zaniCtx

	// 珂莱塔的开场白（根据场景设定）
	initialPrompt := "（推门而入，语气比平时更严肃，但保持优雅）晚上好，赞妮。看起来你正准备下班？"

	for round := 1; round <= maxRounds; round++ {
		fmt.Printf("\n--- 第 %d 轮 ---\n", round)

		var speakerMessage string

		if round == 1 {
			// 第一轮，珂莱塔使用预设的开场白
			speakerMessage = initialPrompt
			fmt.Printf("%s: %s\n", currentSpeaker.Name, speakerMessage)

			// 将开场白加入对话记录
			dialogueRecords = append(dialogueRecords, DialogueRecord{
				Round:   round,
				Speaker: currentSpeaker.Name,
				Message: speakerMessage,
			})

		} else {
			// 后续轮次，当前发言者基于历史生成回复
			agentInput := &adk.AgentInput{
				Messages: currentSpeaker.MessageHistory,
			}

			iter := currentSpeaker.Agent.Run(ctx, agentInput)

			// 收集 Agent 输出
			var assistantMessage *schema.Message
			for {
				event, ok := iter.Next()
				if !ok {
					break
				}

				if event.Err != nil {
					log.Printf("Agent 错误 (%s): %v", currentSpeaker.Name, event.Err)
					break
				}

				if event.Output != nil && event.Output.MessageOutput != nil {
					msgVariant := event.Output.MessageOutput
					msg, err := msgVariant.GetMessage()
					if err != nil {
						log.Printf("获取消息失败 (%s): %v", currentSpeaker.Name, err)
						continue
					}

					if msg.Role == schema.Assistant && msg.Content != "" {
						assistantMessage = msg
					}
				}
			}

			if assistantMessage == nil {
				log.Printf("警告: %s 在第 %d 轮未生成有效消息", currentSpeaker.Name, round)
				break
			}

			speakerMessage = assistantMessage.Content
			fmt.Printf("%s: %s\n", currentSpeaker.Name, speakerMessage)

			// 将发言者的回复加入自己的历史
			currentSpeaker.MessageHistory = append(currentSpeaker.MessageHistory, assistantMessage)

			// 记录对话
			dialogueRecords = append(dialogueRecords, DialogueRecord{
				Round:   round,
				Speaker: currentSpeaker.Name,
				Message: speakerMessage,
			})
		}

		// 将发言者的消息作为 user message 发送给听众
		userMessage := schema.UserMessage(speakerMessage)
		currentListener.MessageHistory = append(currentListener.MessageHistory, userMessage)

		// 交换角色
		currentSpeaker, currentListener = currentListener, currentSpeaker
	}

	// 8. 打印所有对话记录
	fmt.Println("\n\n=== 完整对话记录 ===\n")
	fmt.Println(strings.Repeat("=", 80))

	for _, record := range dialogueRecords {
		fmt.Printf("\n【第 %d 轮】%s:\n", record.Round, record.Speaker)
		fmt.Println(record.Message)
		fmt.Println(strings.Repeat("-", 80))
	}

	fmt.Printf("\n总计 %d 轮对话\n", len(dialogueRecords))
	fmt.Println(strings.Repeat("=", 80))

	// 9. 可选：保存对话记录到文件
	outputFile := "rp/dialogue_output.txt"
	fmt.Printf("\n正在保存对话记录到 %s...\n", outputFile)

	var output strings.Builder
	output.WriteString("=== 赞妮与珂莱塔对话实验记录 ===\n\n")
	output.WriteString(fmt.Sprintf("总计 %d 轮对话\n\n", len(dialogueRecords)))
	output.WriteString(strings.Repeat("=", 80) + "\n\n")

	for _, record := range dialogueRecords {
		output.WriteString(fmt.Sprintf("【第 %d 轮】%s:\n", record.Round, record.Speaker))
		output.WriteString(record.Message + "\n")
		output.WriteString(strings.Repeat("-", 80) + "\n\n")
	}

	if err := os.WriteFile(outputFile, []byte(output.String()), 0644); err != nil {
		log.Printf("保存对话记录失败: %v", err)
	} else {
		fmt.Printf("对话记录已保存到 %s\n", outputFile)
	}
}
