package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/devops"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"gentica1/rp"
)

const (
	DevOpsServerPort = 52539
	ProxyPort        = 52538
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

// DialogueState 对话状态
type DialogueState struct {
	CurrentRound int
	MaxRounds    int
	Records      []DialogueRecord
}

// AgentContext 封装 Agent 的上下文信息
type AgentContext struct {
	Name             string
	CharacterSetting string
	SystemPrompt     string
	SidecarModel     model.ToolCallingChatModel
	Agent            *adk.ChatModelAgent
	MessageHistory   []*schema.Message
}

// DialogueRecord 对话记录
type DialogueRecord struct {
	Round              int
	Speaker            string
	RawMessage         string
	Listener           string
	ListenerPerception string
	ListenerResponse   string
}

// DialogueInput 对话输入
type DialogueInput struct {
	InitialMessage string
}

// DialogueOutput 对话输出
type DialogueOutput struct {
	Records []DialogueRecord
}

// TurnData 单轮对话中传递的数据
type TurnData struct {
	RawMessage string // 原始发言
	Speaker    string // 发言者名称
	Perception string // 感知内容 (Sidecar 生成后填充)
}

// loadMarkdownFile 读取 Markdown 文件内容
func loadMarkdownFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取文件失败 %s: %w", path, err)
	}
	return string(data), nil
}

// buildSidecarPrompt 构建 Sidecar 的系统提示词
func buildSidecarPrompt(listener *AgentContext) string {
	var historyBuilder strings.Builder

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

	return fmt.Sprintf(SidecarSystemPromptTemplate,
		listener.Name, listener.Name, listener.Name,
		listener.CharacterSetting,
		listener.Name,
		historyStr,
	)
}

// runSidecar 执行 Sidecar 逻辑
func runSidecar(ctx context.Context, agent *AgentContext, input *TurnData) (*TurnData, error) {
	log.Printf("[%s Sidecar] 正在为 %s 生成感知...", agent.Name, agent.Name)
	log.Printf("[%s Sidecar] 输入: %s 说: %s", agent.Name, input.Speaker, input.RawMessage)

	systemPrompt := buildSidecarPrompt(agent)
	currentSituation := fmt.Sprintf(`
[当前情景]
对方角色： %s
对方原始行为/发言：
%s

请作为 DM，向 %s 描述此刻的所见所闻（使用第二人称"你"）：`,
		input.Speaker, input.RawMessage, agent.Name)

	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(currentSituation),
	}

	out, err := agent.SidecarModel.Generate(ctx, messages)
	if err != nil {
		perception := fmt.Sprintf("我听见 %s 说了些什么，但没听清。", input.Speaker)
		log.Printf("[%s Sidecar] Error: %v, 使用默认感知", agent.Name, err)
		input.Perception = perception
	} else {
		input.Perception = strings.TrimSpace(out.Content)
	}

	log.Printf("[%s Sidecar] 感知结果:\n%s", agent.Name, input.Perception)

	// 将感知存入历史
	formattedPerception := fmt.Sprintf("> **Perception:**\n%s", input.Perception)
	agent.MessageHistory = append(agent.MessageHistory, schema.UserMessage(formattedPerception))

	return input, nil
}

// runAgent 执行 Agent 逻辑
func runAgent(ctx context.Context, agent *AgentContext) (*TurnData, error) {
	log.Printf("[%s Agent] 正在生成回复...", agent.Name)

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
			return nil, fmt.Errorf("[%s Agent] 运行错误: %w", agent.Name, event.Err)
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
		return nil, fmt.Errorf("[%s Agent] 未生成有效消息", agent.Name)
	}

	agent.MessageHistory = append(agent.MessageHistory, assistantMessage)
	log.Printf("[%s Agent] 回复:\n%s", agent.Name, assistantMessage.Content)

	return &TurnData{
		RawMessage: assistantMessage.Content,
		Speaker:    agent.Name,
	}, nil
}

// createDialogueGraph 创建对话实验的 Graph
// 流程: Carlotta发言 → Zani Sidecar → Zani Agent → Carlotta Sidecar → Carlotta Agent → 循环...
func createDialogueGraph(ctx context.Context, carlotta, zani *AgentContext, maxRounds int) (compose.Runnable[*DialogueInput, *DialogueOutput], error) {
	g := compose.NewGraph[*DialogueInput, *DialogueOutput](
		compose.WithGenLocalState(func(ctx context.Context) *DialogueState {
			return &DialogueState{
				CurrentRound: 0,
				MaxRounds:    maxRounds,
				Records:      []DialogueRecord{},
			}
		}),
	)

	// ===== 节点定义 =====

	// 初始化节点
	initNode := compose.InvokableLambda(func(ctx context.Context, input *DialogueInput) (*TurnData, error) {
		log.Printf("=== 开始对话实验 ===")
		log.Printf("[初始化] Carlotta 开场: %s", input.InitialMessage)
		return &TurnData{
			RawMessage: input.InitialMessage,
			Speaker:    carlotta.Name,
		}, nil
	})

	// Zani Sidecar 节点
	zaniSidecarNode := compose.InvokableLambda(func(ctx context.Context, input *TurnData) (*TurnData, error) {
		return runSidecar(ctx, zani, input)
	})

	// Zani Agent 节点
	zaniAgentNode := compose.InvokableLambda(func(ctx context.Context, input *TurnData) (*TurnData, error) {
		return runAgent(ctx, zani)
	})

	// Zani Agent 的 pre/post handler
	zaniAgentPreHandler := func(ctx context.Context, input *TurnData, state *DialogueState) (*TurnData, error) {
		state.CurrentRound++
		log.Printf("\n--- 第 %d 轮 (Zani 回应) ---", state.CurrentRound)
		return input, nil
	}
	zaniAgentPostHandler := func(ctx context.Context, output *TurnData, state *DialogueState) (*TurnData, error) {
		var perception string
		for i := len(zani.MessageHistory) - 2; i >= 0; i-- {
			if zani.MessageHistory[i].Role == schema.User {
				perception = strings.TrimPrefix(zani.MessageHistory[i].Content, "> **Perception:**\n")
				break
			}
		}
		state.Records = append(state.Records, DialogueRecord{
			Round:              state.CurrentRound,
			Speaker:            carlotta.Name,
			RawMessage:         "",
			Listener:           zani.Name,
			ListenerPerception: perception,
			ListenerResponse:   output.RawMessage,
		})
		return output, nil
	}

	// Carlotta Sidecar 节点
	carlottaSidecarNode := compose.InvokableLambda(func(ctx context.Context, input *TurnData) (*TurnData, error) {
		return runSidecar(ctx, carlotta, input)
	})

	// Carlotta Agent 节点
	carlottaAgentNode := compose.InvokableLambda(func(ctx context.Context, input *TurnData) (*TurnData, error) {
		return runAgent(ctx, carlotta)
	})

	// Carlotta Agent 的 pre/post handler
	carlottaAgentPreHandler := func(ctx context.Context, input *TurnData, state *DialogueState) (*TurnData, error) {
		state.CurrentRound++
		log.Printf("\n--- 第 %d 轮 (Carlotta 回应) ---", state.CurrentRound)
		return input, nil
	}
	carlottaAgentPostHandler := func(ctx context.Context, output *TurnData, state *DialogueState) (*TurnData, error) {
		var perception string
		for i := len(carlotta.MessageHistory) - 2; i >= 0; i-- {
			if carlotta.MessageHistory[i].Role == schema.User {
				perception = strings.TrimPrefix(carlotta.MessageHistory[i].Content, "> **Perception:**\n")
				break
			}
		}
		state.Records = append(state.Records, DialogueRecord{
			Round:              state.CurrentRound,
			Speaker:            zani.Name,
			RawMessage:         "",
			Listener:           carlotta.Name,
			ListenerPerception: perception,
			ListenerResponse:   output.RawMessage,
		})
		return output, nil
	}

	// 输出节点
	outputNode := compose.InvokableLambda(func(ctx context.Context, _ *TurnData) (*DialogueOutput, error) {
		var output *DialogueOutput
		err := compose.ProcessState[*DialogueState](ctx, func(ctx context.Context, state *DialogueState) error {
			log.Printf("=== 对话实验结束，共 %d 轮 ===", state.CurrentRound)
			output = &DialogueOutput{Records: state.Records}
			saveRecords(state.Records)
			return nil
		})
		if err != nil {
			return nil, err
		}
		return output, nil
	})

	// ===== 添加节点 =====
	_ = g.AddLambdaNode("init", initNode, compose.WithNodeName("初始化"))

	_ = g.AddLambdaNode("zani_sidecar", zaniSidecarNode, compose.WithNodeName("Zani-Sidecar"))
	_ = g.AddLambdaNode("zani_agent", zaniAgentNode,
		compose.WithStatePreHandler[*TurnData, *DialogueState](zaniAgentPreHandler),
		compose.WithStatePostHandler[*TurnData, *DialogueState](zaniAgentPostHandler),
		compose.WithNodeName("Zani-Agent"))

	_ = g.AddLambdaNode("carlotta_sidecar", carlottaSidecarNode, compose.WithNodeName("Carlotta-Sidecar"))
	_ = g.AddLambdaNode("carlotta_agent", carlottaAgentNode,
		compose.WithStatePreHandler[*TurnData, *DialogueState](carlottaAgentPreHandler),
		compose.WithStatePostHandler[*TurnData, *DialogueState](carlottaAgentPostHandler),
		compose.WithNodeName("Carlotta-Agent"))

	_ = g.AddLambdaNode("output", outputNode, compose.WithNodeName("输出结果"))

	// ===== 添加边 =====
	_ = g.AddEdge(compose.START, "init")
	_ = g.AddEdge("init", "zani_sidecar")
	_ = g.AddEdge("zani_sidecar", "zani_agent")
	_ = g.AddEdge("zani_agent", "carlotta_sidecar")
	_ = g.AddEdge("carlotta_sidecar", "carlotta_agent")

	// 分支: carlotta_agent 之后判断是否继续
	_ = g.AddBranch("carlotta_agent", compose.NewGraphBranch(
		func(ctx context.Context, output *TurnData) (string, error) {
			var next string
			err := compose.ProcessState[*DialogueState](ctx, func(ctx context.Context, state *DialogueState) error {
				if state.CurrentRound >= state.MaxRounds {
					log.Printf("已达到最大轮次 %d，结束对话", state.MaxRounds)
					next = "output"
				} else {
					next = "zani_sidecar"
				}
				return nil
			})
			if err != nil {
				return "", err
			}
			return next, nil
		},
		map[string]bool{"zani_sidecar": true, "output": true},
	))

	_ = g.AddEdge("output", compose.END)

	return g.Compile(ctx, compose.WithGraphName("dialogue_experiment"))
}

func main() {
	ctx := context.Background()

	// 初始化 DevOps server
	log.Println("正在初始化 DevOps server...")
	err := devops.Init(ctx, devops.WithDevServerPort(fmt.Sprintf("%d", DevOpsServerPort)))
	if err != nil {
		log.Fatalf("Failed to initialize DevOps server: %v", err)
	}

	// API 配置
	apiKey := "sk-6kgtZQDkmZDQMfCo28C360320cEf45FaAf1577Ef08F4032b"
	baseURL := "https://aihubmix.com/v1"
	agentModelName := "DeepSeek-V3.2-Exp"
	sidecarModelName := "grok-4-fast-non-reasoning"

	// 文件路径配置
	zaniConfigPath := "rp/examples/zani.yaml"
	carlottaConfigPath := "rp/examples/carlotta.yaml"
	zaniInnerWorldPath := "docs/wutheringwaves/character-inner-world-zani.md"
	carlottaInnerWorldPath := "docs/wutheringwaves/character-inner-world-carlotta.md"

	// 加载配置
	log.Println("正在加载配置...")
	zaniConfig, err := rp.LoadRoleConfig(zaniConfigPath)
	if err != nil {
		log.Fatalf("加载 Zani 配置失败: %v", err)
	}
	zaniInnerWorld, _ := loadMarkdownFile(zaniInnerWorldPath)
	carlottaConfig, err := rp.LoadRoleConfig(carlottaConfigPath)
	if err != nil {
		log.Fatalf("加载 Carlotta 配置失败: %v", err)
	}
	carlottaInnerWorld, _ := loadMarkdownFile(carlottaInnerWorldPath)

	// 构建 Prompt
	zaniFullSystemPrompt := zaniConfig.CharacterSetting + "\n\n---\n\n# 当前时刻的内心世界状态\n\n" + zaniInnerWorld + "\n\n" + zaniConfig.RoleInstruction
	zaniSidecarSetting := zaniConfig.CharacterSetting + "\n\n---\n\n# 当前时刻的内心世界状态\n\n" + zaniInnerWorld

	carlottaFullSystemPrompt := carlottaConfig.CharacterSetting + "\n\n---\n\n# 当前时刻的内心世界状态\n\n" + carlottaInnerWorld + "\n\n" + carlottaConfig.RoleInstruction
	carlottaSidecarSetting := carlottaConfig.CharacterSetting + "\n\n---\n\n# 当前时刻的内心世界状态\n\n" + carlottaInnerWorld

	// 创建模型和 Agent
	log.Println("正在初始化 Agent...")

	// Carlotta
	carlottaAgentModel, _ := rp.CreateChatModel(ctx, &rp.ModelClientConfig{Model: agentModelName, APIKey: apiKey, BaseURL: baseURL})
	carlottaSidecarModel, _ := rp.CreateChatModel(ctx, &rp.ModelClientConfig{Model: sidecarModelName, APIKey: apiKey, BaseURL: baseURL})
	carlottaAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name: carlottaConfig.Name, Description: carlottaConfig.Name, Instruction: carlottaFullSystemPrompt, Model: carlottaAgentModel, MaxIterations: 10,
	})
	carlottaCtx := &AgentContext{
		Name:             carlottaConfig.Name,
		CharacterSetting: carlottaSidecarSetting,
		SystemPrompt:     carlottaFullSystemPrompt,
		SidecarModel:     carlottaSidecarModel,
		Agent:            carlottaAgent,
		MessageHistory:   []*schema.Message{},
	}

	// Zani
	zaniAgentModel, _ := rp.CreateChatModel(ctx, &rp.ModelClientConfig{Model: agentModelName, APIKey: apiKey, BaseURL: baseURL})
	zaniSidecarModel, _ := rp.CreateChatModel(ctx, &rp.ModelClientConfig{Model: sidecarModelName, APIKey: apiKey, BaseURL: baseURL})
	zaniAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name: zaniConfig.Name, Description: zaniConfig.Name, Instruction: zaniFullSystemPrompt, Model: zaniAgentModel, MaxIterations: 10,
	})
	zaniCtx := &AgentContext{
		Name:             zaniConfig.Name,
		CharacterSetting: zaniSidecarSetting,
		SystemPrompt:     zaniFullSystemPrompt,
		SidecarModel:     zaniSidecarModel,
		Agent:            zaniAgent,
		MessageHistory:   []*schema.Message{},
	}

	// 创建对话 Graph (不自动运行，由 debugger start 触发)
	_, err = createDialogueGraph(ctx, carlottaCtx, zaniCtx, 8)
	if err != nil {
		log.Fatalf("Failed to create dialogue graph: %v", err)
	}

	// 启动代理服务器 (解决 CORS 问题)
	target, _ := url.Parse(fmt.Sprintf("http://localhost:%d", DevOpsServerPort))
	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Set("Access-Control-Allow-Origin", "*")
		resp.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		resp.Header.Set("Access-Control-Allow-Headers", "*")
		resp.Header.Set("Access-Control-Expose-Headers", "*")
		return nil
	}

	proxyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			w.Header().Set("Access-Control-Expose-Headers", "*")
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		proxy.ServeHTTP(w, r)
	})

	log.Printf("✓ DevOps server started on port %d", DevOpsServerPort)
	log.Printf("✓ Proxy server starting on port %d", ProxyPort)
	log.Printf("✓ CORS enabled for all origins")
	log.Printf("✓ Graph 'dialogue_experiment' registered")
	log.Printf("\n访问 DevOps debugger: http://localhost:%d", ProxyPort)
	log.Printf("使用 debugger 的 Start 按钮启动对话流程")
	log.Printf("\n输入示例: {\"InitialMessage\": \"（推门而入，语气比平时更严肃，但保持优雅）晚上好，赞妮。看起来你正准备下班？\"}")

	if err := http.ListenAndServe(fmt.Sprintf(":%d", ProxyPort), proxyHandler); err != nil {
		log.Fatalf("Proxy server failed: %v", err)
	}
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
		log.Println("\n记录已保存到", outputFile)
	}
}
