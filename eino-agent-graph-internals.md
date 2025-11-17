# Agent 底层 Graph 实现揭秘：以掷骰子为例

## 问题

当我们创建一个 Agent 时：
```go
agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Name:  "dice_agent",
    Model: chatModel,
    ToolsConfig: adk.ToolsConfig{
        ToolsNodeConfig: compose.ToolsNodeConfig{
            Tools: []tool.BaseTool{diceTool, calculatorTool, searchTool},  // 3个tools
        },
    },
})
```

**底层的 Graph 是完全展开的还是压缩的？**

---

## 答案：**压缩的！**

通过追踪 eino 源码，我发现 Agent 底层的 Graph 拓扑是**固定的、极简的**，只有 **3-4 个节点**，无论你有多少个 Tools！

---

## 源码追踪链路

### 1. Agent.Run → buildRunFunc → newReact

**文件**: `adk/chatmodel.go:507-651`

```go
func (a *ChatModelAgent) buildRunFunc(ctx context.Context) runFunc {
    a.once.Do(func() {
        // ... 省略准备工作

        // 第 595 行：关键！调用 newReact 创建底层 Graph
        g, err := newReact(ctx, conf)
        if err != nil {
            a.run = errFunc(err)
            return
        }

        // 第 610 行：编译 Graph
        runnable, err_ := g.Compile(ctx, compileOptions...)

        // ... 使用 runnable 执行
    })
    return a.run
}
```

**关键点**: Agent 内部通过 `newReact` 函数创建底层 Graph。

---

### 2. newReact: 构建固定拓扑的 Graph

**文件**: `adk/react.go:121-260`

```go
func newReact(ctx context.Context, config *reactConfig) (reactGraph, error) {
    // 第 136-139 行：只定义了 2 个节点名称
    const (
        chatModel_ = "ChatModel"
        toolNode_  = "ToolNode"
    )

    // 第 141 行：创建 Graph
    g := compose.NewGraph[[]Message, Message](compose.WithGenLocalState(genState))

    // 第 143-150 行：绑定所有 Tools 到 ChatModel
    toolsInfo, err := genToolInfos(ctx, config.toolsConfig)
    chatModel, err := config.model.WithTools(toolsInfo)

    // 第 153 行：创建 ToolsNode（包含所有 tools）
    toolsNode, err := compose.NewToolNode(ctx, config.toolsConfig)

    // ====== 关键！构建固定的 Graph 拓扑 ======

    // 第 167 行：添加 ChatModel 节点（只有1个）
    _ = g.AddChatModelNode(chatModel_, chatModel, ...)

    // 第 190 行：添加 ToolNode 节点（只有1个，包含所有tools）
    _ = g.AddToolsNode(toolNode_, toolsNode, ...)

    // 第 193 行：START → ChatModel
    _ = g.AddEdge(compose.START, chatModel_)

    // 第 195-211 行：ChatModel 后的分支判断
    toolCallCheck := func(ctx context.Context, sMsg MessageStream) (string, error) {
        // 检查是否有 ToolCalls
        if len(chunk.ToolCalls) > 0 {
            return toolNode_, nil   // 去 ToolNode
        }
        return compose.END, nil     // 去 END
    }
    branch := compose.NewStreamGraphBranch(toolCallCheck,
        map[string]bool{compose.END: true, toolNode_: true})
    _ = g.AddBranch(chatModel_, branch)

    // 第 216 行：ToolNode → ChatModel（形成循环）
    _ = g.AddEdge(toolNode_, chatModel_)

    return g, nil
}
```

**Graph 拓扑结构**（固定的！）：

```
         ┌─────────┐
         │  START  │
         └────┬────┘
              │
              ▼
         ┌─────────┐
    ┌───►│ChatModel│◄──────┐
    │    └────┬────┘       │
    │         │             │
    │    (Branch)           │
    │      ┌──┴───┐         │
    │      │      │         │
    │  ToolCalls? │         │
    │   ┌─Yes No─┐│         │
    │   │        ││         │
    │   ▼        ▼▼         │
    │ ┌────┐   ┌───┐       │
    │ │Tool│   │END│       │
    │ │Node│   └───┘       │
    │ └─┬──┘               │
    │   │                  │
    └───┘                  │
```

**节点数量**：
- 不管有 1 个 Tool 还是 100 个 Tools
- Graph 永远只有 **4 个节点**：START, ChatModel, ToolNode, END

---

### 3. ToolNode: 一个节点包含所有 Tools

**文件**: `compose/tool_node.go:70-198`

```go
type ToolsNode struct {
    tuple                *toolsTuple  // ← 包含所有 tools 的元组
    unknownToolHandler   func(...)
    executeSequentially  bool
}

type toolsTuple struct {
    indexes map[string]int                               // ← tool name → index 映射
    meta    []*executorMeta                              // ← 所有 tools 的元数据
    rps     []*runnablePacker[string, string, tool.Option]  // ← 所有 tools 的执行器
}

// 第 150-198 行：将 Tools 数组转换为 tuple
func convTools(ctx context.Context, tools []tool.BaseTool) (*toolsTuple, error) {
    ret := &toolsTuple{
        indexes: make(map[string]int),
        meta:    make([]*executorMeta, len(tools)),
        rps:     make([]*runnablePacker[...], len(tools)),
    }

    for idx, bt := range tools {
        tl, err := bt.Info(ctx)
        toolName := tl.Name

        // 将每个 tool 存储到数组中，建立 name → index 映射
        ret.indexes[toolName] = idx     // ← 关键：map 查找
        ret.meta[idx] = meta
        ret.rps[idx] = newRunnablePacker(invokable, streamable, ...)
    }

    return ret, nil
}
```

**关键点**：
- 一个 `ToolsNode` 实例包含**所有** tools
- 使用 `map[string]int` 存储 tool name → 数组 index 的映射
- 所有 tools 的执行器存储在数组中

---

### 4. ToolNode.Invoke: 运行时动态查找

**文件**: `compose/tool_node.go:215-267, 361-417`

```go
// 第 215 行：根据 ToolCalls 生成任务列表
func (tn *ToolsNode) genToolCallTasks(ctx context.Context, tuple *toolsTuple,
    input *schema.Message, ...) ([]toolCallTask, error) {

    n := len(input.ToolCalls)  // ← LLM 返回了多少个 ToolCalls
    toolCallTasks := make([]toolCallTask, n)

    for i := 0; i < n; i++ {
        toolCall := input.ToolCalls[i]  // ← 获取 LLM 的 ToolCall

        // 第 243 行：关键！通过 map 查找对应的 tool
        index, ok := tuple.indexes[toolCall.Function.Name]
        if !ok {
            return nil, fmt.Errorf("tool %s not found", toolCall.Function.Name)
        }

        // 第 250-262 行：构造任务，指向对应的 tool 执行器
        toolCallTasks[i].r = tuple.rps[index]       // ← 找到对应的 tool
        toolCallTasks[i].meta = tuple.meta[index]
        toolCallTasks[i].name = toolCall.Function.Name
        toolCallTasks[i].arg = toolCall.Function.Arguments
    }

    return toolCallTasks, nil
}

// 第 361 行：ToolNode 的 Invoke 方法
func (tn *ToolsNode) Invoke(ctx context.Context, input *schema.Message, ...) ([]*schema.Message, error) {
    // 第 374 行：生成任务列表（根据 LLM 的 ToolCalls）
    tasks, err := tn.genToolCallTasks(ctx, tuple, input, ...)

    // 第 379-383 行：执行任务（并行或串行）
    if tn.executeSequentially {
        sequentialRunToolCall(ctx, runToolCallTaskByInvoke, tasks, ...)
    } else {
        parallelRunToolCall(ctx, runToolCallTaskByInvoke, tasks, ...)
    }

    // 第 386-416 行：收集结果并返回
    output := make([]*schema.Message, len(tasks))
    for i := 0; i < len(tasks); i++ {
        output[i] = schema.ToolMessage(tasks[i].output, tasks[i].callID, ...)
    }

    return output, nil
}
```

**执行流程**：

```
输入: input.ToolCalls = [
    {Function: {Name: "throw_dice", Arguments: "{\"sides\": 20}"}},
    {Function: {Name: "calculator", Arguments: "{\"expr\": \"1+1\"}"}}
]
    ↓
第1步: genToolCallTasks
    → 遍历每个 ToolCall
    → 通过 indexes["throw_dice"] 查找 → index = 0
    → 通过 indexes["calculator"] 查找 → index = 1
    → 创建 2 个 toolCallTask，分别指向 rps[0] 和 rps[1]
    ↓
第2步: parallelRunToolCall
    → 并行执行 2 个 tasks
    → task[0] 调用 diceTool.InvokableRun(...)
    → task[1] 调用 calculatorTool.InvokableRun(...)
    ↓
第3步: 收集结果
    → output = [ToolMessage("Result: 15"), ToolMessage("Result: 2")]
    ↓
返回: []*schema.Message
```

---

## 对比：静态 Graph vs Agent 的 Graph

### 静态 Graph（完全展开）

假设有 3 个 Tools: A, B, C

**需要的节点**（展开所有可能的调用路径）：

```
START
ChatModel_Initial
Branch_Initial
ToolA_Node         (单独调用 A)
ToolB_Node         (单独调用 B)
ToolC_Node         (单独调用 C)
ToolAB_Node        (先调用 A 再调用 B)
ToolAC_Node        (先调用 A 再调用 C)
ToolBA_Node        (先调用 B 再调用 A)
ToolBC_Node        (先调用 B 再调用 C)
ToolCA_Node        (先调用 C 再调用 A)
ToolCB_Node        (先调用 C 再调用 B)
ToolABC_Node       (A → B → C)
ToolACB_Node       (A → C → B)
ToolBAC_Node       (B → A → C)
ToolBCA_Node       (B → C → A)
ToolCAB_Node       (C → A → B)
ToolCBA_Node       (C → B → A)
ChatModel_AfterA
ChatModel_AfterB
ChatModel_AfterC
ChatModel_AfterAB
... (指数级增长)
END
```

**节点数量**: O(N! + N^2 + N^3 + ...) = **指数级**

**边的数量**: 更多（每个节点之间需要连接）

---

### Agent 的 Graph（压缩）

同样 3 个 Tools: A, B, C

**实际的节点**：

```
START
ChatModel         (只有1个)
ToolNode          (只有1个，内部包含 A, B, C)
END
```

**节点数量**: **4 个固定节点**（O(1)）

**ToolNode 内部结构**：

```go
ToolNode {
    tuple: {
        indexes: {
            "throw_dice": 0,
            "calculator": 1,
            "search": 2,
        },
        rps: [diceTool, calculatorTool, searchTool],  // 数组
    }
}
```

**动态路径生成**：

```
用户: "掷一个20面骰子然后计算结果的平方"
    ↓
ChatModel 推理 → ToolCalls: [{Name: "throw_dice", Args: "{\"sides\": 20}"}]
    ↓
ToolNode 查找: indexes["throw_dice"] = 0 → 执行 rps[0] (diceTool)
    ↓ (返回: "Result: 15")
ChatModel 推理 → ToolCalls: [{Name: "calculator", Args: "{\"expr\": \"15*15\"}"}]
    ↓
ToolNode 查找: indexes["calculator"] = 1 → 执行 rps[1] (calculatorTool)
    ↓ (返回: "Result: 225")
ChatModel 推理 → 没有 ToolCalls → 生成最终回答
    ↓
END
```

**所有可能的路径都通过同一个 Graph 拓扑**，只是：
- LLM 决定调用哪个 Tool（在 `ToolCalls` 中指定 name）
- ToolNode 通过 map 查找动态调度

---

## 核心机制对比

### 静态 Graph: 空间换时间

```
原理: 预先枚举所有可能的路径
实现: 为每个可能的调用序列创建专门的节点
优点: 执行路径确定，无需运行时查找
缺点:
  - 节点数量指数级增长 (O(2^N))
  - 添加 Tool 需要重构 Graph
  - 无法处理动态长度的调用序列
```

---

### Agent: 时间换空间 + 智能

```
原理: 运行时动态查找 + LLM 路由
实现:
  1. 固定的 Graph 拓扑（4个节点）
  2. ToolNode 包含所有 Tools (map + 数组)
  3. LLM 在 ToolCalls 中指定要调用的 Tool
  4. ToolNode 通过 map.indexes[name] 查找
优点:
  - 节点数量固定 (O(1))
  - 添加 Tool 只需更新数组
  - 支持任意长度的调用序列
  - LLM 智能决策调用策略
缺点:
  - 运行时需要 map 查找 (O(1) 但有常数开销)
  - 路径不确定（LLM 黑盒决策）
```

---

## 可视化对比

### 静态 Graph（3个Tools的部分展开）

```
                    ┌──────────┐
                    │  START   │
                    └─────┬────┘
                          │
                          ▼
                   ┌─────────────┐
                   │ ChatModel   │
                   └──────┬──────┘
                          │
                    (需要调用哪个Tool?)
         ┌────────────────┼────────────────┐
         │                │                │
         ▼                ▼                ▼
    ┌────────┐       ┌────────┐      ┌────────┐
    │ToolA   │       │ToolB   │      │ToolC   │
    │Node    │       │Node    │      │Node    │
    └───┬────┘       └───┬────┘      └───┬────┘
        │                │                │
        │      (需要再调用吗?)           │
        ├─────┬────────┬─────┬─────────┬┤
        │     │        │     │         ││
        ▼     ▼        ▼     ▼         ▼▼
    ┌─────┐ ┌──────┐ ┌──────┐ ┌──────┐ ...
    │ToolB│ │ToolC │ │ToolA │ │ToolC │ (更多组合)
    │After│ │After │ │After │ │After │
    │A    │ │B     │ │B     │ │C     │
    └─┬───┘ └───┬──┘ └───┬──┘ └───┬──┘
      │         │        │        │
      └─────────┴────────┴────────┘
                 │
                 ▼
            ┌────────┐
            │  END   │
            └────────┘

节点数量: 20+ (只展开了2层)
实际上: 需要 O(N^depth) 个节点
```

---

### Agent 的 Graph（3个Tools，任意深度）

```
         ┌─────────┐
         │  START  │
         └────┬────┘
              │
              ▼
         ┌──────────────┐
    ┌───►│ ChatModel    │◄──────┐
    │    └──────┬───────┘       │
    │           │                │
    │      (Branch:              │
    │    检查 ToolCalls?)        │
    │      ┌────┴────┐           │
    │      │         │           │
    │   有ToolCalls 无          │
    │      │         │           │
    │      ▼         ▼           │
    │  ┌──────────┐ ┌────┐      │
    │  │ToolNode  │ │END │      │
    │  │─────────│ └────┘      │
    │  │indexes: │              │
    │  │ A → 0   │              │
    │  │ B → 1   │              │
    │  │ C → 2   │              │
    │  │─────────│              │
    │  │rps:     │              │
    │  │ [0]=ToolA             │
    │  │ [1]=ToolB             │
    │  │ [2]=ToolC             │
    │  └─────┬────┘              │
    │        │                   │
    └────────┘                   │
     (循环: 可以多次调用)        │

节点数量: 4 (固定)
支持深度: 无限 (MaxIterations 限制)
```

---

## 掷骰子示例的实际执行

### 输入

```go
agent.Run(ctx, &adk.AgentInput{
    Messages: []adk.Message{
        schema.UserMessage("Throw 3 dice with 20 sides and calculate the total"),
    },
})
```

---

### 执行流程（Graph 拓扑固定）

```
第1次循环:
START → ChatModel
    ↓ (分析用户请求)
    output: ToolCalls = [{Name: "throw_dice", Args: "{\"sides\": 20, \"count\": 3}"}]
    ↓ (Branch: 有 ToolCalls)
    → ToolNode
        ↓ (genToolCallTasks)
        查找: indexes["throw_dice"] = 0
        执行: rps[0].InvokableRun("{\"sides\": 20, \"count\": 3}")
        返回: "Threw 3 dice with 20 sides. Results: [15, 8, 12], Total: 35"
    ↓ (ToolNode → ChatModel)

第2次循环:
→ ChatModel
    ↓ (分析工具结果)
    output: 没有 ToolCalls，生成最终回答
           "The total of the three 20-sided dice is 35."
    ↓ (Branch: 无 ToolCalls)
    → END

结束
```

**Graph 拓扑**: 始终是 4 个节点
**实际路径**: START → ChatModel → ToolNode → ChatModel → END
**循环次数**: 2 次（第1次调用 tool，第2次生成回答）

---

## 如果有100个Tools呢？

### 静态 Graph

```
节点数量: O(100!) ≈ 10^157 个节点（不可能实现）
```

### Agent 的 Graph

```
节点数量: 4 个（固定）
ToolNode 内部:
  indexes: {
    "tool_1": 0,
    "tool_2": 1,
    ...
    "tool_100": 99,
  }
  rps: [tool_1, tool_2, ..., tool_100]  // 数组长度 100
```

**map 查找**: O(1) 时间复杂度
**添加新 Tool**: 只需在数组末尾追加

---

## 总结

### Agent 底层 Graph 的本质

1. **拓扑固定**：
   - 只有 4 个节点：START, ChatModel, ToolNode, END
   - 边的连接固定：形成 ReAct 循环
   - **与 Tools 数量无关**

2. **ToolNode 是"万能节点"**：
   - 内部包含所有 Tools（map + 数组）
   - 运行时根据 LLM 的 `ToolCalls` 动态查找
   - 可以并行执行多个 Tools

3. **LLM 作为智能路由器**：
   - 决定调用哪个 Tool（在 `ToolCalls.Function.Name` 中指定）
   - 决定调用顺序（可以连续多次调用）
   - 决定何时结束（不再生成 ToolCalls）

4. **压缩机制**：
   ```
   静态 Graph: 枚举所有路径 → O(2^N) 个节点
   Agent:      压缩到 LLM 推理 → O(1) 个节点
   ```

5. **类比**：
   ```
   静态 Graph = 有限状态机（显式枚举所有状态）
   Agent      = 图灵机（LLM 作为控制器，动态决策）
   ```

---

### 回答原问题

> "以掷骰子为例，Agent 底层的 Graph 是完全展开的还是怎么样？"

**答案**: **压缩的！**

- Graph 拓扑只有 **4 个固定节点**
- 所有 Tools 被压缩到 **1 个 ToolNode** 中
- **不是**为每个 Tool 创建一个节点
- **不是**为每个可能的调用序列创建路径
- 通过 **map 查找 + LLM 动态决策**，用 O(1) 的节点数实现了 O(2^N) 的状态空间

---

### 关键代码位置

```
Agent 创建:
  adk/chatmodel.go:177 (NewChatModelAgent)

底层 Graph 构建:
  adk/chatmodel.go:595 (newReact)
  adk/react.go:121-260 (newReact 实现)

ToolNode 实现:
  compose/tool_node.go:70 (ToolsNode 结构)
  compose/tool_node.go:150 (convTools - 构建 map)
  compose/tool_node.go:215 (genToolCallTasks - 运行时查找)
  compose/tool_node.go:361 (Invoke - 执行)
```

---

### 设计智慧

eino 的 Agent 设计体现了经典的**时空权衡**（Time-Space Tradeoff）：

```
传统方法: 空间换时间
  → 预先展开所有路径（空间爆炸）
  → 运行时直接跳转（时间快）

eino 的方法: 时间换空间 + 智能
  → 固定的小 Graph（空间 O(1)）
  → 运行时 map 查找 + LLM 路由（时间 O(1) + 推理开销）
  → 用 LLM 的"智能"替代"显式枚举"
```

**这就是 Agent "压缩" Graph 的本质！**
