package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"gentica1/rp"
)

// SidecarSystemPromptTemplate 定义 Sidecar 的系统提示词模板
const SidecarSystemPromptTemplate = `你是 %s 的专属 DM (Dungeon Master) / 叙事旁白。
你的任务是向角色（%s）描述当前发生的事件、看到的景象以及听到的声音。

[工作原则]
1. 第二人称叙事：必须使用第二人称("你")向角色描述当前情况。例如："你看到..."，"你听到..."。
2. 忠实转述：**必须完整**复述对方角色的核心发言内容、动作和场上发生的重要事件（如有），角色只能依赖你的描述获取这些外界信息，**严禁忽略**。
3. 感知过滤：根据角色的当前状态（位置、注意力、环境噪音等），合理判断角色能接收到多少信息和关注的重点。例如，如果距离太远或环境嘈杂，告诉角色"你听不太清对方在说什么，只听到...(关键词)...(关键词)..."。
4. 主观投射与不确定性：参考角色的性格设定，对客观事实进行主观化描述，但在合理时机做主观视角色彩化处理。对于带有情感色彩或可能产生误解的部分，使用不确定性的描述词（如"你似乎感到"、"在你的视角里，对方好像..."、"你隐约觉得..."），从而为角色的心理活动留出空间。
5. 沉浸感：描述应当具有画面感和代入感，引导角色进入情境，但不要代替角色做出反应或决策，不要代替角色思考。
6. 简洁行文：除非情况极其复杂，否则描述应控制在 3 句以内，简明扼要。
7. 避免重复：严禁包含前文历史中已经描述过的内容，只描述最新的变化。

请注意：不要代表角色进行回复或站在角色立场思考，只负责描述输入。

[%s 的设定]
%s

[%s 的过往感知历史]
(以下是你之前的叙述和角色的反应，仅供参考，请勿重复描述)
%s

---
`

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
	Name             string
	CharacterSetting string
	SystemPrompt     string
	Model            model.ToolCallingChatModel
	Agent            *adk.ChatModelAgent
	MessageHistory   []*schema.Message
}

// DialogueRecord 对话记录
type DialogueRecord struct {
	Round              int    // 第几轮
	Speaker            string // 本轮发言者
	RawMessage         string // 发言者的原始消息
	Listener           string // 本轮接收者
	ListenerPerception string // 接收者对发言的感知
	ListenerResponse   string // 接收者的回复
}

// runSidecar 执行 Sidecar Agent 生成感知
func runSidecar(ctx context.Context, listener *AgentContext, speakerName string, rawContent string) (string, error) {
	// 1. 构建历史记录字符串
	// Listener 的 UserMessage (Perception) -> Sidecar 之前的叙述
	// Listener 的 AssistantMessage (Response) -> Sidecar 观察到的宿主反应
	var historyBuilder strings.Builder

	// 仅保留最近的 8 对记录（16条消息）
	startIndex := 0
	if len(listener.MessageHistory) > 16 {
		startIndex = len(listener.MessageHistory) - 16
	}

	for i := startIndex; i < len(listener.MessageHistory); i++ {
		msg := listener.MessageHistory[i]
		if msg.Role == schema.User {
			cleanContent := strings.TrimPrefix(msg.Content, "> **Perception:**\n")
			historyBuilder.WriteString(fmt.Sprintf("[DM叙述]: %s\n", cleanContent))
		} else if msg.Role == schema.Assistant {
			historyBuilder.WriteString(fmt.Sprintf("[角色反应]: %s\n", msg.Content))
		}
	}
	historyStr := historyBuilder.String()
	if historyStr == "" {
		historyStr = "(暂无历史)"
	}

	// 2. 构建 Sidecar 的系统提示词
	systemPrompt := fmt.Sprintf(SidecarSystemPromptTemplate,
		listener.Name, listener.Name, listener.Name,
		listener.CharacterSetting,
		listener.Name,
		historyStr,
	)

	// 3. 构建当前的输入消息 (User Message)
	currentSituation := fmt.Sprintf(`
[当前情景]
对方角色： %s
对方原始行为/发言：
%s

请作为 DM，向 %s 描述此刻的所见所闻（使用第二人称"你"）：`,
		speakerName, rawContent, listener.Name)

	// 组合所有消息 (注意：历史已经作为 System Prompt 的一部分注入了，所以这里不需要再 append history)
	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(currentSituation),
	}

	// 4. 调用模型生成
	out, err := listener.Model.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("sidecar 生成失败: %w", err)
	}

	content := strings.TrimSpace(out.Content)
	return content, nil
}

// runAgent 执行 Character Agent 生成回复
func runAgent(ctx context.Context, agent *AgentContext) (string, error) {
	agentInput := &adk.AgentInput{
		Messages: agent.MessageHistory,
	}

	iter := agent.Agent.Run(ctx, agentInput)

	var assistantMessage *schema.Message
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return "", fmt.Errorf("agent 运行错误: %w", event.Err)
		}
		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				continue
			}
			if msg.Role == schema.Assistant && msg.Content != "" {
				assistantMessage = msg
			}
		}
	}

	if assistantMessage == nil {
		return "", fmt.Errorf("agent 未生成有效消息")
	}

	// 将回复加入历史
	agent.MessageHistory = append(agent.MessageHistory, assistantMessage)
	return assistantMessage.Content, nil
}

func main() {
	// API 配置
	// 注意：如果使用原生 Gemini/Claude 接口，需要提供对应的 API Key，且 BaseURL 可能需要调整或留空。
	apiKey := "sk-6kgtZQDkmZDQMfCo28C360320cEf45FaAf1577Ef08F4032b"
	baseURL := "https://aihubmix.com/v1"

	// modelName 可选值示例:
	// - "Kimi-K2-0905" (OpenAI 兼容)
	// - "gemini-1.5-pro" (使用原生 Gemini 接口，支持自定义 BaseURL)
	// - "claude-3-5-sonnet" (使用原生 Claude 接口，支持自定义 BaseURL)
	// - "deepseek-chat" (使用 DeepSeek client)
	agentModelName := "DeepSeek-V3.2-Exp"
	sidecarModelName := "grok-4-fast-non-reasoning"

	// 文件路径配置
	zaniConfigPath := "rp/examples/zani.yaml"
	carlottaConfigPath := "rp/examples/carlotta.yaml"
	zaniInnerWorldPath := "docs/wutheringwaves/character-inner-world-zani.md"
	carlottaInnerWorldPath := "docs/wutheringwaves/character-inner-world-carlotta.md"

	ctx := context.Background()

	// 1. 加载配置
	fmt.Println("正在加载配置...")
	zaniConfig, _ := rp.LoadRoleConfig(zaniConfigPath)
	zaniInnerWorld, _ := loadMarkdownFile(zaniInnerWorldPath)
	carlottaConfig, _ := rp.LoadRoleConfig(carlottaConfigPath)
	carlottaInnerWorld, _ := loadMarkdownFile(carlottaInnerWorldPath)

	// 构建 Prompt
	zaniFullSystemPrompt := zaniConfig.CharacterSetting + "\n\n---\n\n# 当前时刻的内心世界状态\n\n" + zaniInnerWorld + "\n\n" + zaniConfig.RoleInstruction
	zaniSidecarSetting := zaniConfig.CharacterSetting + "\n\n---\n\n# 当前时刻的内心世界状态\n\n" + zaniInnerWorld

	carlottaFullSystemPrompt := carlottaConfig.CharacterSetting + "\n\n---\n\n# 当前时刻的内心世界状态\n\n" + carlottaInnerWorld + "\n\n" + carlottaConfig.RoleInstruction
	carlottaSidecarSetting := carlottaConfig.CharacterSetting + "\n\n---\n\n# 当前时刻的内心世界状态\n\n" + carlottaInnerWorld

	// 2. 创建模型和 Agent
	fmt.Println("正在初始化 Agent...")

	// Zani
	zaniAgentModel, _ := rp.CreateChatModel(ctx, &rp.ModelClientConfig{Model: agentModelName, APIKey: apiKey, BaseURL: baseURL})
	zaniSidecarModel, _ := rp.CreateChatModel(ctx, &rp.ModelClientConfig{Model: sidecarModelName, APIKey: apiKey, BaseURL: baseURL})
	zaniAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name: zaniConfig.Name, Description: zaniConfig.Name, Instruction: zaniFullSystemPrompt, Model: zaniAgentModel, MaxIterations: 10,
	})

	// Carlotta
	carlottaAgentModel, _ := rp.CreateChatModel(ctx, &rp.ModelClientConfig{Model: agentModelName, APIKey: apiKey, BaseURL: baseURL})
	carlottaSidecarModel, _ := rp.CreateChatModel(ctx, &rp.ModelClientConfig{Model: sidecarModelName, APIKey: apiKey, BaseURL: baseURL})
	carlottaAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name: carlottaConfig.Name, Description: carlottaConfig.Name, Instruction: carlottaFullSystemPrompt, Model: carlottaAgentModel, MaxIterations: 10,
	})

	// 上下文
	zaniCtx := &AgentContext{
		Name: zaniConfig.Name, CharacterSetting: zaniSidecarSetting, SystemPrompt: zaniFullSystemPrompt,
		Model: zaniSidecarModel, Agent: zaniAgent, MessageHistory: []*schema.Message{},
	}
	carlottaCtx := &AgentContext{
		Name: carlottaConfig.Name, CharacterSetting: carlottaSidecarSetting, SystemPrompt: carlottaFullSystemPrompt,
		Model: carlottaSidecarModel, Agent: carlottaAgent, MessageHistory: []*schema.Message{},
	}

	// 3. 对话循环
	fmt.Println("\n=== 开始对话实验 ===")
	dialogueRecords := []DialogueRecord{}
	const maxRounds = 8

	currentSpeaker := carlottaCtx
	currentListener := zaniCtx

	// 初始开场白
	initialPrompt := "（推门而入，语气比平时更严肃，但保持优雅）晚上好，赞妮。看起来你正准备下班？"
	lastRawMessage := initialPrompt

	fmt.Printf("[Speaker] %s: %s\n", currentSpeaker.Name, lastRawMessage)

	for round := 1; round <= maxRounds; round++ {
		fmt.Printf("\n--- 第 %d 轮 ---\n", round)

		// 1. Listener 感知 (Sidecar)
		fmt.Printf("正在生成 %s 的感知...\n", currentListener.Name)
		perception, err := runSidecar(ctx, currentListener, currentSpeaker.Name, lastRawMessage)
		if err != nil {
			log.Printf("Sidecar Error: %v", err)
			perception = fmt.Sprintf("我听见 %s 说了些什么，但没听清。", currentSpeaker.Name)
		}
		fmt.Printf("[Sidecar] %s 感知:\n%s\n", currentListener.Name, perception)

		// 将感知存入 Listener 历史
		// 关键修正：必须确保 Listener 知道这是感知到的对方发言
		formattedPerception := fmt.Sprintf("> **Perception:**\n%s", perception)
		currentListener.MessageHistory = append(currentListener.MessageHistory, schema.UserMessage(formattedPerception))

		// 2. Listener 回复 (Agent)
		fmt.Printf("正在生成 %s 的回复...\n", currentListener.Name)
		response, err := runAgent(ctx, currentListener)
		if err != nil {
			log.Printf("Agent Error: %v", err)
			break
		}
		fmt.Printf("[Agent] %s 回复:\n%s\n", currentListener.Name, response)

		// 记录这一轮
		dialogueRecords = append(dialogueRecords, DialogueRecord{
			Round:              round,
			Speaker:            currentSpeaker.Name,
			RawMessage:         lastRawMessage,
			Listener:           currentListener.Name,
			ListenerPerception: perception,
			ListenerResponse:   response,
		})

		// 准备下一轮
		lastRawMessage = response // Listener 的回复成为下一轮的 Raw Message

		// 交换角色
		currentSpeaker, currentListener = currentListener, currentSpeaker
	}

	// 4. 保存记录
	saveRecords(dialogueRecords)
}

func saveRecords(records []DialogueRecord) {
	outputFile := "rp/dialogue_output.txt"
	var output strings.Builder
	output.WriteString("=== 对话实验记录 ===\n\n")

	for _, r := range records {
		output.WriteString(fmt.Sprintf("【第 %d 轮】\n", r.Round))
		output.WriteString(fmt.Sprintf("Speaker (%s): %s\n", r.Speaker, r.RawMessage))
		output.WriteString(fmt.Sprintf("Listener (%s) Perception:\n> %s\n", r.Listener, r.ListenerPerception))
		output.WriteString(fmt.Sprintf("Listener (%s) Response:\n%s\n", r.Listener, r.ListenerResponse))
		output.WriteString(strings.Repeat("-", 40) + "\n\n")
	}

	if err := os.WriteFile(outputFile, []byte(output.String()), 0644); err != nil {
		log.Println("保存失败:", err)
	} else {
		fmt.Println("\n记录已保存到", outputFile)
	}
}
