package agent

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gentica/config"
	"gentica/csync"
	"gentica/db"
	"gentica/llm/tools"
	"gentica/message"
	"gentica/session"

	"github.com/charmbracelet/catwalk/pkg/catwalk"
)

var (
	testDB       *sql.DB
	testQueries  *db.Queries
	testSessions session.Service
	testMessages message.Service
	testTempDir  string
)

func init() {
	ctx := context.Background()

	// 创建临时目录用于测试数据库
	var err error
	testTempDir, err = os.MkdirTemp("", "agent-test-*")
	if err != nil {
		panic("创建临时目录失败: " + err.Error())
	}

	// 使用 db.Connect 初始化数据库
	testDB, err = db.Connect(ctx, testTempDir)
	if err != nil {
		panic("连接数据库失败: " + err.Error())
	}

	// 创建数据库查询对象
	testQueries = db.New(testDB)

	// 创建真实的服务
	testSessions = session.NewService(testQueries)
	testMessages = message.NewService(testQueries)

	// 初始化配置
	config.Init(testTempDir, testTempDir, false)

	// 确保配置已初始化
	cfg := config.Get()
	if cfg == nil {
		panic("配置未初始化")
	}

	// 设置 provider 配置 - 使用 Gemini
	providerMap := csync.NewMap[string, config.ProviderConfig]()
	providerMap.Set("gemini", config.ProviderConfig{
		ID:      "gemini",
		Name:    "Gemini",
		Type:    catwalk.TypeGemini,
		BaseURL: "https://aihubmix.com/gemini",
		APIKey:  "sk-6kgtZQDkmZDQMfCo28C360320cEf45FaAf1577Ef08F4032b",
		Models: []catwalk.Model{
			{
				ID:               "gemini-2.5-flash",
				Name:             "Gemini 2.5 Flash",
				DefaultMaxTokens: 1024,
				ContextWindow:    1000000,
				CostPer1MIn:      0.35,
				CostPer1MOut:     1.05,
			},
		},
	})
	cfg.Providers = providerMap

	// 设置 model 配置
	cfg.Models = map[config.SelectedModelType]config.SelectedModel{
		config.SelectedModelTypeLarge: {
			Provider: "gemini",
			Model:    "gemini-2.5-flash",
		},
		config.SelectedModelTypeSmall: {
			Provider: "gemini",
			Model:    "gemini-2.5-flash",
		},
	}
}

// TestSimpleAgentChat 测试简单的 agent 对话 - 使用真实服务
func TestSimpleAgentChat(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 清理函数 - 不要关闭共享的 testDB
	t.Cleanup(func() {
		// 不关闭 testDB，因为它是共享的
	})

	ctx := context.Background()

	// 创建测试 session
	testSession, err := testSessions.Create(ctx, "Test Session")
	if err != nil {
		t.Fatalf("创建 session 失败: %v", err)
	}

	// 创建最简单的 agent 配置
	agentCfg := config.Agent{
		ID:    "test-agent",
		Name:  "Test Agent",
		Model: config.SelectedModelTypeLarge,
		// 不要任何工具，纯聊天
		AllowedTools: []string{},
	}

	// 创建 agent
	agent, err := NewAgent(ctx, agentCfg, testSessions, testMessages, nil)
	if err != nil {
		t.Fatalf("创建 agent 失败: %v", err)
	}

	// 发送一个简单的聊天请求
	content := "Hello World"

	// 使用带超时的 context
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	t.Logf("开始运行 agent，sessionID: %s", testSession.ID)
	t.Logf("使用Provider: %s, Model: %s", config.Get().Models[config.SelectedModelTypeLarge].Provider, config.Get().Models[config.SelectedModelTypeLarge].Model)
	eventChan, err := agent.Run(ctx, testSession.ID, content)
	if err != nil {
		t.Fatalf("运行 agent 失败: %v", err)
	}

	if eventChan == nil {
		t.Fatal("eventChan 为 nil")
	}
	t.Log("成功获取 eventChan")

	// 处理事件
	var responseReceived bool
	var responseContent string
	eventCount := 0

	for event := range eventChan {
		eventCount++
		t.Logf("收到事件 #%d: Type=%v, Done=%v, Error=%v", eventCount, event.Type, event.Done, event.Error)

		if event.Error != nil {
			t.Fatalf("收到错误事件: %v", event.Error)
		}

		if event.Type == AgentEventTypeResponse && event.Done {
			responseReceived = true
			// 获取响应内容
			msgList, _ := testMessages.List(ctx, testSession.ID)
			for _, msg := range msgList {
				if msg.Role == message.Assistant {
					for _, part := range msg.Parts {
						if textPart, ok := part.(message.TextContent); ok {
							responseContent += textPart.Text
						}
					}
				}
			}
			t.Logf("收到响应: %s", responseContent)
		}
	}

	t.Logf("事件处理完成，共收到 %d 个事件", eventCount)

	// 检查是否因为超时而退出
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatal("等待响应超时")
	}

	if !responseReceived {
		t.Error("未收到响应事件")
	}

	if responseContent == "" {
		t.Error("响应内容为空")
	}

	t.Logf("测试成功完成，收到响应: %s", responseContent)
}

// TestAgentWithMultipleTools 测试 agent 调用多个工具
func TestAgentWithMultipleTools(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 清理函数 - 不要关闭共享的 testDB
	t.Cleanup(func() {
		// 不关闭 testDB，因为它是共享的
	})

	ctx := context.Background()

	// 创建测试目录和文件
	testDir, err := os.MkdirTemp("", "agent-tools-test-*")
	if err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	defer os.RemoveAll(testDir)

	// 创建一些测试文件
	testFile1 := filepath.Join(testDir, "test1.go")
	testFile2 := filepath.Join(testDir, "test2.txt")
	testFile3 := filepath.Join(testDir, "config.json")

	err = os.WriteFile(testFile1, []byte(`package main

import "fmt"

func main() {
	fmt.Println("Hello from test1.go")
}
`), 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	err = os.WriteFile(testFile2, []byte("This is a test text file\nWith multiple lines\nFor testing purposes"), 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	err = os.WriteFile(testFile3, []byte(`{
	"name": "test-config",
	"version": "1.0.0",
	"enabled": true
}`), 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 创建测试 session
	testSession, err := testSessions.Create(ctx, "Test Tools Session")
	if err != nil {
		t.Fatalf("创建 session 失败: %v", err)
	}

	// 创建工具
	availableTools := map[string]tools.BaseTool{
		"ls":    tools.NewLsTool(testDir),
		"view":  tools.NewViewTool(testDir),
		"glob":  tools.NewGlobTool(testDir),
		"grep":  tools.NewGrepTool(testDir),
		"write": tools.NewWriteTool(testDir),
	}

	// 创建 agent 配置，允许使用多个工具
	agentCfg := config.Agent{
		ID:           "test-tools-agent",
		Name:         "Test Tools Agent",
		Model:        config.SelectedModelTypeLarge,
		AllowedTools: []string{"ls", "view", "glob", "grep", "write"},
	}

	// 创建 agent
	agent, err := NewAgent(ctx, agentCfg, testSessions, testMessages, availableTools)
	if err != nil {
		t.Fatalf("创建 agent 失败: %v", err)
	}

	// 发送一个需要使用多个工具的请求
	content := fmt.Sprintf("请执行以下任务：1. 列出 %s 目录中的所有文件 2. 找到所有 .go 文件 3. 查看 test1.go 文件的内容 4. 在目录中创建一个新文件 summary.md，内容是对目录中文件的简单总结", testDir)

	// 使用带超时的 context
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	t.Logf("开始运行带工具的 agent，sessionID: %s", testSession.ID)
	eventChan, err := agent.Run(ctx, testSession.ID, content)
	if err != nil {
		t.Fatalf("运行 agent 失败: %v", err)
	}

	// 处理事件
	var responseReceived bool
	eventCount := 0

	for event := range eventChan {
		eventCount++
		t.Logf("收到事件 #%d: Type=%v, Done=%v", eventCount, event.Type, event.Done)

		if event.Error != nil {
			t.Fatalf("收到错误事件: %v", event.Error)
		}

		if event.Type == AgentEventTypeResponse && event.Done {
			responseReceived = true
			t.Log("收到最终响应")
		}
	}

	t.Logf("事件处理完成，共收到 %d 个事件", eventCount)

	// 从消息历史中获取工具调用信息
	msgList, err := testMessages.List(ctx, testSession.ID)
	if err != nil {
		t.Fatalf("获取消息列表失败: %v", err)
	}

	var toolsUsed []string
	for _, msg := range msgList {
		if msg.Role == message.Assistant {
			toolCalls := msg.ToolCalls()
			for _, tc := range toolCalls {
				toolsUsed = append(toolsUsed, tc.Name)
				t.Logf("检测到工具调用: %s (ID: %s)", tc.Name, tc.ID)
			}
		}
	}

	// 同时检查工具结果
	var toolResults []string
	for _, msg := range msgList {
		if msg.Role == message.User {
			results := msg.ToolResults()
			for _, tr := range results {
				toolResults = append(toolResults, tr.Name)
				t.Logf("检测到工具结果: %s (Error: %v)", tr.Name, tr.IsError)
			}
		}
	}

	// 验证结果
	if !responseReceived {
		t.Error("未收到响应事件")
	}

	if len(toolsUsed) == 0 {
		t.Error("Agent 没有调用任何工具")
	} else {
		t.Logf("Agent 共调用了 %d 个工具: %v", len(toolsUsed), toolsUsed)
	}

	// 检查是否创建了 summary.md 文件
	summaryFile := filepath.Join(testDir, "summary.md")
	if _, err := os.Stat(summaryFile); err == nil {
		t.Log("成功创建了 summary.md 文件")
		content, _ := os.ReadFile(summaryFile)
		t.Logf("summary.md 内容: %s", string(content))
	} else {
		t.Log("未创建 summary.md 文件（可能 Agent 选择了不同的方式完成任务）")
	}
	t.Log(msgList)
}

// TestAgentToolError 测试工具错误处理
func TestAgentToolError(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 清理函数 - 不要关闭共享的 testDB
	t.Cleanup(func() {
		// 不关闭 testDB，因为它是共享的
	})

	ctx := context.Background()

	// 创建测试目录
	testDir, err := os.MkdirTemp("", "agent-error-test-*")
	if err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	defer os.RemoveAll(testDir)

	// 创建测试 session
	testSession, err := testSessions.Create(ctx, "Test Error Session")
	if err != nil {
		t.Fatalf("创建 session 失败: %v", err)
	}

	// 创建工具
	availableTools := map[string]tools.BaseTool{
		"view": tools.NewViewTool(testDir),
		"ls":   tools.NewLsTool(testDir),
	}

	// 创建 agent 配置
	agentCfg := config.Agent{
		ID:           "test-error-agent",
		Name:         "Test Error Agent",
		Model:        config.SelectedModelTypeLarge,
		AllowedTools: []string{"view", "ls"},
	}

	// 创建 agent
	agent, err := NewAgent(ctx, agentCfg, testSessions, testMessages, availableTools)
	if err != nil {
		t.Fatalf("创建 agent 失败: %v", err)
	}

	// 发送一个会导致工具错误的请求
	nonExistentFile := filepath.Join(testDir, "does-not-exist.txt")
	content := fmt.Sprintf("请查看文件 %s 的内容，如果文件不存在，请告诉我并列出 %s 目录中实际存在的文件", nonExistentFile, testDir)

	// 使用带超时的 context
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	t.Logf("开始运行错误处理测试，sessionID: %s", testSession.ID)
	eventChan, err := agent.Run(ctx, testSession.ID, content)
	if err != nil {
		t.Fatalf("运行 agent 失败: %v", err)
	}

	// 处理事件
	var responseReceived bool
	eventCount := 0

	for event := range eventChan {
		eventCount++
		t.Logf("收到事件 #%d: Type=%v, Done=%v", eventCount, event.Type, event.Done)

		if event.Error != nil {
			// 这里是 agent 级别的错误，不应该发生
			t.Fatalf("收到 agent 错误事件: %v", event.Error)
		}

		if event.Type == AgentEventTypeResponse && event.Done {
			responseReceived = true
			t.Log("收到最终响应")
		}
	}

	t.Logf("事件处理完成，共收到 %d 个事件", eventCount)

	// 从消息历史中获取信息
	msgList, err := testMessages.List(ctx, testSession.ID)
	if err != nil {
		t.Fatalf("获取消息列表失败: %v", err)
	}

	// 获取工具调用
	var toolsUsed []string
	for _, msg := range msgList {
		if msg.Role == message.Assistant {
			toolCalls := msg.ToolCalls()
			for _, tc := range toolCalls {
				toolsUsed = append(toolsUsed, tc.Name)
				t.Logf("检测到工具调用: %s", tc.Name)
			}
			// 输出 assistant 的响应
			for _, part := range msg.Parts {
				if textPart, ok := part.(message.TextContent); ok {
					t.Logf("Agent 响应: %s", textPart.Text)
				}
			}
		}
	}

	// 获取工具结果和错误
	var toolErrors []string
	for _, msg := range msgList {
		if msg.Role == message.User {
			results := msg.ToolResults()
			for _, tr := range results {
				if tr.IsError {
					toolErrors = append(toolErrors, tr.Content)
					t.Logf("工具返回错误: %s", tr.Content)
				}
			}
		}
	}

	// 验证结果
	if !responseReceived {
		t.Error("未收到响应事件")
	}

	if len(toolsUsed) == 0 {
		t.Error("Agent 没有调用任何工具")
	} else {
		t.Logf("Agent 共调用了 %d 个工具: %v", len(toolsUsed), toolsUsed)
	}

	// 应该有至少一个工具错误（view 不存在的文件）
	if len(toolErrors) > 0 {
		t.Logf("Agent 正确处理了 %d 个工具错误", len(toolErrors))
	} else {
		t.Log("注意：没有检测到工具错误，可能 Agent 避免了调用会出错的工具")
	}

	// Agent 应该能够从错误中恢复并继续执行（比如调用 ls）
	hasLsTool := false
	for _, tool := range toolsUsed {
		if tool == "ls" {
			hasLsTool = true
			break
		}
	}

	if hasLsTool {
		t.Log("Agent 成功从错误中恢复并调用了 ls 工具")
	} else {
		t.Log("Agent 可能选择了不同的策略来处理错误")
	}
}

// TestAgentWithSubAgent 测试主 agent 通过 tool call 启动子 agent
func TestAgentWithSubAgent(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 清理函数 - 不要关闭共享的 testDB
	t.Cleanup(func() {
		// 不关闭 testDB，因为它是共享的
	})

	ctx := context.Background()

	// 创建测试目录和文件
	testDir, err := os.MkdirTemp("", "agent-subagent-test-*")
	if err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	defer os.RemoveAll(testDir)

	// 创建一些测试文件供子 agent 查找
	testFile1 := filepath.Join(testDir, "module1.go")
	testFile2 := filepath.Join(testDir, "module2.go")
	testFile3 := filepath.Join(testDir, "readme.txt")
	subDir := filepath.Join(testDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("创建子目录失败: %v", err)
	}
	testFile4 := filepath.Join(subDir, "helper.go")

	err = os.WriteFile(testFile1, []byte(`package module1

// Module1 provides basic functionality
func Process() {
	println("Processing in module1")
}
`), 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	err = os.WriteFile(testFile2, []byte(`package module2

// Module2 provides advanced features
func Advanced() {
	println("Advanced features in module2")
}
`), 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	err = os.WriteFile(testFile3, []byte("This is a readme file for the test project"), 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	err = os.WriteFile(testFile4, []byte(`package helper

// Helper utilities
func Utility() {
	println("Helper utility")
}
`), 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 创建子 agent - 只有文件操作工具
	subAgentCfg := config.Agent{
		ID:           "sub-agent",
		Name:         "Sub Agent",
		Model:        config.SelectedModelTypeLarge,
		AllowedTools: []string{"ls", "view", "glob", "grep"},
	}

	subAgentTools := map[string]tools.BaseTool{
		"ls":   tools.NewLsTool(testDir),
		"view": tools.NewViewTool(testDir),
		"glob": tools.NewGlobTool(testDir),
		"grep": tools.NewGrepTool(testDir),
	}

	subAgent, err := NewAgent(ctx, subAgentCfg, testSessions, testMessages, subAgentTools)
	if err != nil {
		t.Fatalf("创建子 agent 失败: %v", err)
	}

	// 创建主 agent 的 session
	mainSession, err := testSessions.Create(ctx, "Main Agent Session")
	if err != nil {
		t.Fatalf("创建主 agent session 失败: %v", err)
	}

	// 为主 agent 创建工具，包括子 agent 工具
	subAgentTool := NewAgentToolWithID("sub-agent", subAgent, testSessions, testMessages)
	
	mainAgentTools := map[string]tools.BaseTool{
		"agent_sub-agent": subAgentTool, // 子 agent 作为工具
		"write":           tools.NewWriteTool(testDir),
		"ls":              tools.NewLsTool(testDir), // 主 agent 也有 ls 工具
	}

	// 创建主 agent 配置
	mainAgentCfg := config.Agent{
		ID:           "main-agent",
		Name:         "Main Agent",
		Model:        config.SelectedModelTypeLarge,
		AllowedTools: []string{"agent_sub-agent", "write", "ls"},
	}

	// 创建主 agent
	mainAgent, err := NewAgent(ctx, mainAgentCfg, testSessions, testMessages, mainAgentTools)
	if err != nil {
		t.Fatalf("创建主 agent 失败: %v", err)
	}

	// 发送任务给主 agent
	content := fmt.Sprintf(`请完成以下任务：
1. 首先自己使用 ls 工具快速查看 %s 目录结构
2. 然后委托子 agent (使用 agent_sub-agent 工具) 执行以下任务：
   - 找出目录 %s 中所有的 .go 文件
   - 分析这些 Go 文件的内容
   - 总结每个文件的功能
3. 基于子 agent 的分析结果，使用 write 工具在 %s 目录中创建一个 analysis.md 文件，记录分析结果

注意：你必须调用 agent_sub-agent 工具来完成文件分析任务。`, testDir, testDir, testDir)

	// 使用带超时的 context
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	t.Logf("开始运行主 agent，sessionID: %s", mainSession.ID)
	eventChan, err := mainAgent.Run(ctx, mainSession.ID, content)
	if err != nil {
		t.Fatalf("运行主 agent 失败: %v", err)
	}

	// 处理事件
	var responseReceived bool
	eventCount := 0

	for event := range eventChan {
		eventCount++
		t.Logf("主 agent 事件 #%d: Type=%v, Done=%v", eventCount, event.Type, event.Done)

		if event.Error != nil {
			t.Fatalf("收到错误事件: %v", event.Error)
		}

		if event.Type == AgentEventTypeResponse && event.Done {
			responseReceived = true
			t.Log("主 agent 完成任务")
		}
	}

	t.Logf("主 agent 事件处理完成，共收到 %d 个事件", eventCount)

	// 验证主 agent 的工具调用
	mainMsgList, err := testMessages.List(ctx, mainSession.ID)
	if err != nil {
		t.Fatalf("获取主 agent 消息列表失败: %v", err)
	}

	var mainToolsUsed []string
	var subAgentCalled bool
	for _, msg := range mainMsgList {
		if msg.Role == message.Assistant {
			toolCalls := msg.ToolCalls()
			for _, tc := range toolCalls {
				mainToolsUsed = append(mainToolsUsed, tc.Name)
				t.Logf("主 agent 调用工具: %s", tc.Name)
				if tc.Name == "agent_sub-agent" {
					subAgentCalled = true
				}
			}
		}
	}

	// 验证结果
	if !responseReceived {
		t.Error("主 agent 未收到响应事件")
	}

	if !subAgentCalled {
		t.Error("主 agent 没有调用子 agent")
	} else {
		t.Log("✓ 主 agent 成功调用了子 agent")
	}

	// 检查是否创建了 analysis.md 文件
	analysisFile := filepath.Join(testDir, "analysis.md")
	if fileInfo, err := os.Stat(analysisFile); err == nil {
		t.Logf("✓ 成功创建了 analysis.md 文件 (大小: %d bytes)", fileInfo.Size())
		content, _ := os.ReadFile(analysisFile)
		t.Logf("analysis.md 内容预览:\n%s", string(content[:minInt(500, len(content))]))
	} else {
		t.Error("未创建 analysis.md 文件")
	}

	// 查找子 agent 的 session 并验证它也执行了任务
	allSessions, err := testSessions.List(ctx)
	if err != nil {
		t.Logf("获取所有 sessions 失败: %v", err)
	} else {
		for _, sess := range allSessions {
			if sess.ParentSessionID == mainSession.ID {
				t.Logf("✓ 找到子 agent session: ID=%s, Title=%s", sess.ID, sess.Title)
				
				// 获取子 agent 的消息
				subMsgList, err := testMessages.List(ctx, sess.ID)
				if err == nil {
					var subToolsUsed []string
					for _, msg := range subMsgList {
						if msg.Role == message.Assistant {
							toolCalls := msg.ToolCalls()
							for _, tc := range toolCalls {
								subToolsUsed = append(subToolsUsed, tc.Name)
							}
						}
					}
					if len(subToolsUsed) > 0 {
						t.Logf("✓ 子 agent 使用了工具: %v", subToolsUsed)
					}
				}
			}
		}
	}

	t.Logf("主 agent 共调用了 %d 个工具: %v", len(mainToolsUsed), mainToolsUsed)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
