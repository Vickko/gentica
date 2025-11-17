# Eino Agent vs Graph 实战对比：掷骰子工具

## 测试结果

两种方式都成功运行，输出完全相同：

```
Test 1: 掷一个骰子
Graph 方式: ✅ 结果 3
Agent 方式: ✅ 结果 3

Test 2: 掷3个20面骰子
Graph 方式: ✅ 总和 32 (13, 6, 13)
Agent 方式: ✅ 总和 32 (13, 6, 13)

Test 3: 普通对话（不使用工具）
Graph 方式: ✅ "The capital of France is Paris"
Agent 方式: ✅ "The capital of France is Paris"
```

---

## 代码对比

### Graph 方式（TestThrowDiceWithTool）

**代码量**: ~185 行

**核心结构**:
```go
// 1. 创建 ChatModel 和 Tool (10 行)
chatModel, _ := openai.NewChatModel(ctx, config)
diceTool := NewThrowDiceTool()
chatModel.BindTools([]*schema.ToolInfo{...})
toolsNode, _ := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
    Tools: []tool.BaseTool{diceTool},
})

// 2. 定义状态类型 (5 行)
type AgentState struct {
    Messages []*schema.Message
}

// 3. 创建带状态的 Graph (10 行)
graph := compose.NewGraph[[]*schema.Message, *schema.Message](
    compose.WithGenLocalState(func(ctx context.Context) *AgentState {
        return &AgentState{Messages: []*schema.Message{}}
    }),
)

// 4. 定义 ChatModel 的状态处理器 (30 行)
modelStatePreHandler := func(ctx context.Context, input []*schema.Message, state *AgentState) ([]*schema.Message, error) {
    if len(state.Messages) == 0 {
        state.Messages = input
    }
    return state.Messages, nil
}

modelStatePostHandler := func(ctx context.Context, output *schema.Message, state *AgentState) (*schema.Message, error) {
    state.Messages = append(state.Messages, output)
    return output, nil
}

// 5. 定义 ToolsNode 的状态处理器 (25 行)
toolsStatePreHandler := func(ctx context.Context, input *schema.Message, state *AgentState) (*schema.Message, error) {
    if len(state.Messages) == 0 || state.Messages[len(state.Messages)-1] != input {
        state.Messages = append(state.Messages, input)
    }
    return input, nil
}

toolsStatePostHandler := func(ctx context.Context, toolMessages []*schema.Message, state *AgentState) ([]*schema.Message, error) {
    state.Messages = append(state.Messages, toolMessages...)
    return toolMessages, nil
}

// 6. 添加节点 (带状态处理) (10 行)
graph.AddChatModelNode("chat_model", chatModel,
    compose.WithStatePreHandler(modelStatePreHandler),
    compose.WithStatePostHandler(modelStatePostHandler),
)

graph.AddToolsNode("tools_executor", toolsNode,
    compose.WithStatePreHandler(toolsStatePreHandler),
    compose.WithStatePostHandler(toolsStatePostHandler),
)

// 7. 定义分支逻辑 (15 行)
modelBranchCondition := func(ctx context.Context, msg *schema.Message) (map[string]bool, error) {
    if len(msg.ToolCalls) > 0 {
        return map[string]bool{"tools_executor": true}, nil
    }
    return map[string]bool{compose.END: true}, nil
}

modelBranch := compose.NewGraphMultiBranch(modelBranchCondition, map[string]bool{
    "tools_executor": true,
    compose.END:      true,
})

// 8. 构建图的边 (15 行)
graph.AddEdge(compose.START, "chat_model")
graph.AddBranch("chat_model", modelBranch)
graph.AddEdge("tools_executor", "chat_model")  // 形成循环

// 9. 编译和执行 (10 行)
compiledGraph, _ := graph.Compile(ctx)
result, _ := compiledGraph.Invoke(ctx, messages)
```

**手动管理的内容**:
- ✅ Graph 拓扑结构（节点、边、分支）
- ✅ 状态管理（定义 State 结构、Pre/Post Handler）
- ✅ 消息历史的追加逻辑
- ✅ 循环控制（ToolsNode → ChatModel）
- ✅ 结束条件（Branch 判断）
- ✅ 类型转换（确保输入输出匹配）

---

### Agent 方式（TestThrowDiceWithAgent）

**代码量**: ~20 行（核心代码仅 10 行）

**核心结构**:
```go
// 1. 创建 ChatModel 和 Tool (5 行)
chatModel, _ := openai.NewChatModel(ctx, config)
diceTool := NewThrowDiceTool()

// 2. 创建 Agent（一步到位！）(10 行)
agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Name:        "dice_agent",
    Description: "An agent that can throw dice for you",
    Instruction: "You are a helpful assistant that can throw dice for users.",
    Model:       chatModel,
    ToolsConfig: adk.ToolsConfig{
        ToolsNodeConfig: compose.ToolsNodeConfig{
            Tools: []tool.BaseTool{diceTool},
        },
    },
    MaxIterations: 10,
})

// 3. 执行 (5 行)
input := &adk.AgentInput{
    Messages: []adk.Message{
        schema.UserMessage("Please throw a dice for me!"),
    },
}
iter := agent.Run(ctx, input)
result := getAgentFinalResult(t, iter)
```

**Agent 自动管理的内容**:
- ✅ 内部 Graph 构建（START → ChatModel → Branch → ToolsNode）
- ✅ 状态管理（State 结构、消息历史）
- ✅ 循环控制（ReAct 模式的自动循环）
- ✅ 结束条件判断（自动检测 ToolCalls）
- ✅ 类型匹配（内部处理）
- ✅ 迭代次数限制（MaxIterations）

---

## 关键差异对比

### 1. 代码复杂度

| 维度 | Graph 方式 | Agent 方式 | 改进 |
|------|-----------|-----------|------|
| **总行数** | ~185 行 | ~20 行 | **↓ 90%** |
| **核心逻辑** | ~150 行 | ~10 行 | **↓ 93%** |
| **状态管理** | 手动（70行） | 自动 | **完全省略** |
| **Graph 构建** | 手动（50行） | 自动 | **完全省略** |
| **分支逻辑** | 手动（15行） | 自动 | **完全省略** |

---

### 2. 开发心智负担

**Graph 方式需要思考**:
```
1. 我需要几个节点？
   → START, ChatModel, ToolsNode, END

2. 节点之间如何连接？
   → START → ChatModel → Branch → [ToolsNode | END]
   → ToolsNode → ChatModel (形成循环)

3. 如何管理消息历史？
   → 定义 AgentState 结构
   → 实现 4 个状态处理器（Pre/Post for ChatModel and ToolsNode）

4. 如何判断是否需要调用工具？
   → 实现 modelBranchCondition 函数
   → 检查 msg.ToolCalls

5. 类型是否匹配？
   → ChatModel 输出: *schema.Message
   → ToolsNode 输入: *schema.Message
   → Branch 输出: string
   → 需要确保所有连接的类型正确

6. 如何防止无限循环？
   → 需要手动添加 MaxIterations 检查
```

**Agent 方式只需思考**:
```
1. Agent 的职责是什么？
   → 掷骰子助手

2. 需要哪些能力（Tools）？
   → throw_dice

3. 系统提示词是什么？
   → "You are a helpful assistant..."

完成！✅
```

**心智负担对比**: Graph 方式需要理解 **6 个维度** vs Agent 方式只需理解 **3 个维度**

---

### 3. 执行流程

#### Graph 方式的执行路径（显式）

```
用户输入: "Please throw a dice"
    ↓
START 节点
    ↓
ChatModel 节点
    ├→ modelStatePreHandler: 保存输入到 State
    ├→ ChatModel.Generate: 生成 ToolCalls
    └→ modelStatePostHandler: 追加输出到 State
    ↓
Branch 判断
    ├→ 检测到 ToolCalls → 路由到 ToolsNode
    ↓
ToolsNode 节点
    ├→ toolsStatePreHandler: 保存消息到 State
    ├→ 执行 throw_dice Tool
    └→ toolsStatePostHandler: 追加工具结果到 State
    ↓
回到 ChatModel 节点（循环）
    ├→ modelStatePreHandler: 从 State 读取完整历史
    ├→ ChatModel.Generate: 根据工具结果生成回复
    └→ modelStatePostHandler: 追加输出到 State
    ↓
Branch 判断
    └→ 没有 ToolCalls → 路由到 END
    ↓
END 节点
    ↓
返回最终结果
```

**开发者负责**: 每一步的逻辑和连接

---

#### Agent 方式的执行路径（隐式）

```
用户输入: "Please throw a dice"
    ↓
agent.Run(ctx, input)
    ↓
[内部自动执行完整的 ReAct 循环]
    ↓
返回 AgentEvent 流
    ├→ Event 1: Assistant message (空 - 准备调用工具)
    ├→ Event 2: Tool result (Threw 1 dice... Result: 3)
    └→ Event 3: Assistant message (I rolled a die, result is 3!)
    ↓
开发者获取最终结果
```

**开发者负责**: 创建 Agent + 处理结果事件流

**Agent 自动处理**: 所有的 Graph 编排、状态管理、循环控制

---

### 4. 可维护性

#### 添加新的 Tool

**Graph 方式**:
```go
// 1. 创建新 Tool
newTool := NewCalculatorTool()

// 2. 绑定到 ChatModel
chatModel.BindTools([]*schema.ToolInfo{
    mustGetToolInfo(diceTool, ctx, t),
    mustGetToolInfo(newTool, ctx, t),  // 新增
})

// 3. 添加到 ToolsNode
toolsNode, _ := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
    Tools: []tool.BaseTool{diceTool, newTool},  // 新增
})

// 4. 其余 150 行代码不变（但需要验证）
```

**修改点**: 2 处（但需要理解整个 Graph 结构）

---

**Agent 方式**:
```go
// 1. 创建新 Tool
newTool := NewCalculatorTool()

// 2. 添加到 Agent 配置
agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    // ... 其他配置不变
    ToolsConfig: adk.ToolsConfig{
        ToolsNodeConfig: compose.ToolsNodeConfig{
            Tools: []tool.BaseTool{diceTool, newTool},  // 只需在这里添加
        },
    },
})

// 完成！✅
```

**修改点**: 1 处（无需理解内部实现）

---

#### 修改系统提示词

**Graph 方式**:
```go
// 需要在初始输入中添加 System Message
messages := []*schema.Message{
    schema.SystemMessage("You are a dice throwing expert..."),  // 手动添加
    schema.UserMessage("Throw a dice"),
}

// 或者在状态处理器中注入
modelStatePreHandler := func(ctx context.Context, input []*schema.Message, state *AgentState) ([]*schema.Message, error) {
    // 复杂的逻辑来插入系统消息
    ...
}
```

**修改点**: 需要在多处考虑系统消息的注入位置

---

**Agent 方式**:
```go
agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Instruction: "You are a dice throwing expert...",  // 直接修改
    // ... 其他配置
})
```

**修改点**: 1 处配置，Agent 自动在每次调用时注入

---

### 5. 调试体验

#### Graph 方式

**调试时需要追踪**:
- 当前在哪个节点？
- State 中的消息历史是什么？
- 分支判断走了哪条路径？
- 状态处理器是否正确执行？
- 类型转换是否正确？

**调试工具**: 在每个状态处理器中添加日志

```go
modelStatePreHandler := func(ctx context.Context, input []*schema.Message, state *AgentState) ([]*schema.Message, error) {
    t.Logf("ChatModel input: %d messages in state", len(state.Messages))  // 手动日志
    // ...
}
```

---

#### Agent 方式

**调试时只需追踪**:
- AgentEvent 流中的事件

**调试工具**: Agent 天然输出结构化的事件流

```go
for {
    event, ok := iter.Next()
    if !ok { break }

    if event.Output != nil && event.Output.MessageOutput != nil {
        msg, _ := event.Output.MessageOutput.GetMessage()
        if msg.Role == schema.Assistant {
            t.Logf("[Agent Event] Assistant: %s", msg.Content)
        } else if msg.Role == schema.Tool {
            t.Logf("[Agent Event] Tool result: %s", msg.Content)
        }
    }
}
```

**Agent 自动记录**: 每个步骤都以 AgentEvent 的形式暴露

---

## 实际运行对比

### Graph 方式日志

```
ChatModel input: 1 messages in state
ChatModel output: 1 tool calls
Branch from ChatModel: Detected 1 tool calls, routing to tools_executor
ToolsNode executed: Added 1 tool messages to state
ChatModel input: 3 messages in state
ChatModel output: 0 tool calls
Branch from ChatModel: No tool calls, routing to END
User: Please throw a dice for me and tell me the result!
Graph result: I rolled a 6-sided die for you, and the result is **3**!
```

**特点**:
- 需要手动添加日志
- 日志分散在各个状态处理器中
- 需要理解 Graph 结构才能读懂日志

---

### Agent 方式日志

```
[Agent Event] Assistant message:
[Agent Event] Tool result: Threw 1 dice with 6 sides. Results: [3], Total: 3
[Agent Event] Assistant message: I rolled a 6-sided die for you, and the result is **3**!
User: Please throw a dice for me and tell me the result!
Agent result: I rolled a 6-sided die for you, and the result is **3**!
```

**特点**:
- Agent 自动输出结构化事件
- 日志清晰展示执行流程
- 无需理解内部实现即可理解日志

---

## Agent 如何"压缩" Graph

### Graph 的显式状态空间

```
Graph 拓扑:
┌──────┐    ┌───────────┐    ┌────────┐
│START │───→│ ChatModel │───→│ Branch │
└──────┘    └───────────┘    └───┬────┘
                 ↑                 │
                 │            ToolCalls?
                 │             ┌─Yes
                 │             │
            ┌────┴────┐    ┌───▼──────┐
            │循环回来 │    │ToolsNode │
            └─────────┘    └──────────┘
                 ↑             │
                 └─────────────┘

开发者必须显式定义:
- 4 个节点（START, ChatModel, ToolsNode, Branch）
- 5 条边（START→ChatModel, ChatModel→Branch, Branch→ToolsNode,
         ToolsNode→ChatModel, Branch→END）
- 2 个状态处理器对（ChatModel Pre/Post, ToolsNode Pre/Post）
- 1 个分支判断函数
```

**总复杂度**: O(N) 个组件显式声明

---

### Agent 的隐式状态空间

```
Agent 声明:
agent = NewChatModelAgent(config)

内部自动构建相同的 Graph（在 adk/react.go:121-250）:
┌──────┐    ┌───────────┐    ┌────────┐
│START │───→│ ChatModel │───→│ Branch │
└──────┘    └───────────┘    └───┬────┘
                 ↑                 │
                 │            ToolCalls?
                 │             ┌─Yes
                 │             │
            ┌────┴────┐    ┌───▼──────┐
            │自动循环 │    │ToolsNode │
            └─────────┘    └──────────┘
                 ↑             │
                 └─────────────┘

开发者只需声明:
- Agent 的职责（Name, Description）
- Agent 的能力（Tools）
- 业务逻辑（Instruction）
```

**总复杂度**: O(1) 个配置对象

---

### 压缩比

```
Graph 方式 = 显式展开所有组件
Agent 方式 = 将 Graph 结构压缩到框架内部

压缩比 = Graph 代码行数 / Agent 代码行数
       = 185 / 20
       = 9.25:1

即：Agent 将 9.25 行 Graph 代码压缩成 1 行配置
```

---

## 类比：汇编 vs 高级语言

### Graph 方式 = 汇编

```assembly
; 手动管理寄存器、栈、跳转
START:
    MOV R1, [user_input]      ; 加载输入
    CALL chat_model           ; 调用模型
    CMP R2, [has_tool_calls]  ; 检查是否有工具调用
    JZ END                    ; 如果没有，跳转到结束
    CALL tools_node           ; 调用工具
    JMP START                 ; 跳回开始（循环）
END:
    RET R3                    ; 返回结果
```

**特点**:
- 精确控制每一步
- 需要理解底层机制
- 代码冗长但灵活
- 适合性能优化

---

### Agent 方式 = 高级语言

```python
# 声明式配置，框架负责编译成底层指令
agent = Agent(
    name="dice_agent",
    tools=[throw_dice],
    instruction="You are a helpful assistant"
)

result = agent.run(user_input)
```

**特点**:
- 声明式配置
- 隐藏底层细节
- 代码简洁易懂
- 适合快速开发

---

## 何时使用哪种方式？

### 优先使用 Agent

✅ **快速原型开发**
- Agent 的 10 行代码 vs Graph 的 150 行

✅ **标准 ReAct 模式**
- ChatModel + Tools + 自动循环
- 大部分场景都适用

✅ **团队协作**
- 新人容易理解（只需理解 Agent 概念）
- 维护成本低

✅ **业务需求频繁变化**
- 添加/删除 Tool 只需修改配置
- 无需重构 Graph 结构

---

### 使用 Graph（低层控制）

✅ **非标准流程**
```go
// 例如：并行调用多个模型
graph.AddEdge("input", "model_1")
graph.AddEdge("input", "model_2")  // 并行
graph.AddEdge("model_1", "merge")
graph.AddEdge("model_2", "merge")
```

✅ **复杂的分支逻辑**
```go
// 例如：根据用户等级路由到不同处理流程
vipBranch := func(ctx context.Context, user User) string {
    if user.Level == "VIP" {
        return "premium_service"
    }
    return "standard_service"
}
```

✅ **性能关键路径**
```go
// 精确优化每个节点的执行
graph.AddLambdaNode("cache_check", cacheLayer)  // 添加缓存层
```

✅ **教学/学习目的**
- 理解 Agent 的底层实现
- 深入掌握 eino 的 Graph 机制

---

## 总结

### Agent 是 Graph 的高层抽象

```
Agent (高层)
  ↓ 编译
Graph (低层)
  ↓ 执行
Runtime (引擎)
```

### 关键洞察

1. **Agent 不是动态 Graph**，而是**固定拓扑 + 动态路径**
   - 拓扑结构：静态（3-4 个节点）
   - 执行路径：动态（LLM 决策）

2. **Agent 是"压缩的 Graph"**
   - 将指数级的状态空间压缩到 LLM 的推理能力中
   - 用智能换代码（LLM 作为万能路由器）

3. **9.25:1 的压缩比**
   - Graph: 185 行 → Agent: 20 行
   - 90% 的代码被框架吸收

4. **选择原则**
   - 默认使用 Agent（简单、高效）
   - 需要非标准流程时才使用 Graph
   - 可以混合使用（Agent 内部基于 Graph）

### 最佳实践

```
开发流程:
1. 先用 Agent 快速原型
2. 验证业务逻辑
3. 如果遇到 Agent 无法满足的场景，再考虑 Graph
4. 性能优化阶段可以深入到 Graph 层面

技术选型:
- 80% 的场景：Agent 足够
- 15% 的场景：Agent + 自定义 Tool
- 5% 的场景：需要直接使用 Graph
```

---

**文件位置**:
- Graph 测试: `eino/hello_test.go:TestThrowDiceWithTool`
- Agent 测试: `eino/hello_test.go:TestThrowDiceWithAgent`
- Tool 实现: `eino/hello_test.go:ThrowDiceTool`

**运行测试**:
```bash
# Graph 方式
go test -v -run TestThrowDiceWithTool

# Agent 方式
go test -v -run TestThrowDiceWithAgent
```
