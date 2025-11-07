# Eino 框架架构分析

## 目录

1. [概述](#概述)
2. [核心概念实体](#核心概念实体)
   - [1. Runnable（可执行对象）](#1-runnable可执行对象)
   - [2. ChatModel（聊天模型）](#2-chatmodel聊天模型)
   - [3. Tool（工具）](#3-tool工具)
   - [4. ChatTemplate（提示词模板）](#4-chattemplate提示词模板)
   - [5. Retriever（检索器）](#5-retriever检索器)
   - [6. Embedder（嵌入器）](#6-embedder嵌入器)
   - [7. Indexer（索引器）](#7-indexer索引器)
   - [8. Document Loader（文档加载器）](#8-document-loader文档加载器)
   - [9. Document Transformer（文档转换器）](#9-document-transformer文档转换器)
   - [10. Chain（链式编排）](#10-chain链式编排)
   - [11. Graph（图编排）](#11-graph图编排)
   - [12. ToolsNode（工具节点）](#12-toolsnode工具节点)
   - [13. Lambda（自定义函数）](#13-lambda自定义函数)
   - [14. Message（消息）](#14-message消息)
   - [15. Document（文档）](#15-document文档)
   - [16. StreamReader/StreamWriter（流读写器）](#16-streamreaderstreamwriter流读写器)
   - [17. Callback Handler（回调处理器）](#17-callback-handler回调处理器)
   - [18. State（状态管理）](#18-state状态管理)
   - [19. Branch（分支）](#19-branch分支)
   - [20. Flow（预定义流程）](#20-flow预定义流程)
3. [协同工作机制](#协同工作机制)
4. [设计原则](#设计原则)
5. [架构图](#架构图)

---

## 概述

**Eino** 是由 ByteDance CloudWeGo 开源的 Go 语言 LLM/AI 应用开发框架。Eino 的设计目标是提供简洁、可扩展、可靠且高效的 AI 应用开发体验，遵循 Go 语言编程规范。

**核心特性：**
- **组件抽象化**：精心设计的组件接口，易于重用和组合
- **强大的编排能力**：支持 Chain、Graph、Workflow 三种编排方式
- **完整的流处理**：自动处理流的拼接、分流、合并、转换
- **类型安全**：使用 Go 泛型确保编译时类型检查
- **可扩展的切面（Callbacks）**：支持日志、追踪、指标等横切关注点
- **四种流范式**：Invoke、Stream、Collect、Transform，满足不同数据流需求

---

## 核心概念实体

### 1. Runnable（可执行对象）

**定义**：Runnable 是 Eino 中最核心的抽象，表示任何可执行的对象。

**位置**：`compose/runnable.go`

**接口定义**：
```go
type Runnable[I, O any] interface {
    Invoke(ctx context.Context, input I, opts ...Option) (output O, err error)
    Stream(ctx context.Context, input I, opts ...Option) (output *schema.StreamReader[O], err error)
    Collect(ctx context.Context, input *schema.StreamReader[I], opts ...Option) (output O, err error)
    Transform(ctx context.Context, input *schema.StreamReader[I], opts ...Option) (output *schema.StreamReader[O], err error)
}
```

**核心特点**：
- **四种流范式**：
  - `Invoke`：非流输入 → 非流输出
  - `Stream`：非流输入 → 流输出
  - `Collect`：流输入 → 非流输出
  - `Transform`：流输入 → 流输出
- **自动降级**：如果组件只实现部分方法，框架会自动转换
- **类型安全**：使用泛型 `[I, O any]` 确保类型匹配

**内部实现**：
- `composableRunnable`：包装用户提供的可执行对象
- 包含 `invoke` 和 `transform` 两个核心函数
- 携带输入/输出类型信息用于编译时验证

---

### 2. ChatModel（聊天模型）

**定义**：ChatModel 是与 LLM 交互的核心组件。

**位置**：`components/model/interface.go`

**接口定义**：
```go
type BaseChatModel interface {
    Generate(ctx context.Context, input []*schema.Message, opts ...Option) (*schema.Message, error)
    Stream(ctx context.Context, input []*schema.Message, opts ...Option) (*schema.StreamReader[*schema.Message], error)
}

type ToolCallingChatModel interface {
    BaseChatModel
    WithTools(tools []*schema.ToolInfo) (ToolCallingChatModel, error)
}
```

**核心特点**：
- **BaseChatModel**：基础聊天模型，支持 Generate 和 Stream
- **ToolCallingChatModel**：支持工具调用的模型，推荐使用（线程安全）
- **Deprecated ChatModel**：旧版本，存在并发问题
- **输入**：`[]*schema.Message` - 消息列表
- **输出**：`*schema.Message` - 单条消息（可能包含 ToolCalls）

**使用示例**：
```go
model, _ := openai.NewChatModel(ctx, config)
message, _ := model.Generate(ctx, []*schema.Message{
    schema.SystemMessage("you are a helpful assistant."),
    schema.UserMessage("what does the future AI App look like?"),
})
```

---

### 3. Tool（工具）

**定义**：Tool 是 LLM 可以调用的外部函数。

**位置**：`components/tool/interface.go`

**接口定义**：
```go
type BaseTool interface {
    Info(ctx context.Context) (*schema.ToolInfo, error)
}

type InvokableTool interface {
    BaseTool
    InvokableRun(ctx context.Context, argumentsInJSON string, opts ...Option) (string, error)
}

type StreamableTool interface {
    BaseTool
    StreamableRun(ctx context.Context, argumentsInJSON string, opts ...Option) (*schema.StreamReader[string], error)
}
```

**核心特点**：
- **BaseTool**：提供工具信息（名称、描述、参数 Schema）
- **InvokableTool**：同步执行工具
- **StreamableTool**：流式执行工具
- **输入**：JSON 格式的参数字符串
- **输出**：字符串结果或流

**ToolInfo 结构**：
```go
type ToolInfo struct {
    Name        string           // 工具名称
    Desc        string           // 工具描述
    ParamsOneOf []*ParameterInfo // 参数定义（JSONSchema）
}
```

---

### 4. ChatTemplate（提示词模板）

**定义**：ChatTemplate 用于格式化提示词，生成消息列表。

**位置**：`components/prompt/interface.go`

**接口定义**：
```go
type ChatTemplate interface {
    Format(ctx context.Context, vs map[string]any, opts ...Option) ([]*schema.Message, error)
}
```

**核心特点**：
- **支持三种模板格式**：
  - FString：Python f-string 风格（`pyfmt`）
  - GoTemplate：Go 标准模板
  - Jinja2：Jinja2 模板引擎（`gonja`）
- **MessagesPlaceholder**：支持插入消息列表占位符
- **输入**：`map[string]any` - 模板变量
- **输出**：`[]*schema.Message` - 格式化后的消息列表

**使用示例**：
```go
chatTemplate := prompt.FromMessages(
    schema.SystemMessage("you are eino helper"),
    schema.MessagesPlaceholder("history", false),
    schema.UserMessage("query: {query}"),
)
msgs, err := chatTemplate.Format(ctx, map[string]any{
    "history": []*schema.Message{...},
    "query": "how to use eino?",
})
```

---

### 5. Retriever（检索器）

**定义**：Retriever 从向量数据库或搜索引擎检索相关文档。

**位置**：`components/retriever/interface.go`

**接口定义**：
```go
type Retriever interface {
    Retrieve(ctx context.Context, query string, opts ...Option) ([]*schema.Document, error)
}
```

**核心特点**：
- **输入**：查询字符串
- **输出**：相关文档列表（`[]*schema.Document`）
- **常用 Options**：TopK、ScoreThreshold 等
- **用途**：RAG（检索增强生成）场景

**使用示例**：
```go
retriever, err := redis.NewRetriever(ctx, &redis.RetrieverConfig{})
docs, err := retriever.Retrieve(ctx, "query", retriever.WithTopK(3))
```

---

### 6. Embedder（嵌入器）

**定义**：Embedder 将文本转换为向量表示。

**位置**：`components/embedding/interface.go`

**接口定义**：
```go
type Embedder interface {
    EmbedStrings(ctx context.Context, texts []string, opts ...Option) ([][]float64, error)
}
```

**核心特点**：
- **输入**：文本列表 `[]string`
- **输出**：向量列表 `[][]float64`
- **用途**：语义搜索、文档索引

---

### 7. Indexer（索引器）

**定义**：Indexer 将文档存储到向量数据库。

**位置**：`components/indexer/interface.go`

**接口定义**：
```go
type Indexer interface {
    Store(ctx context.Context, docs []*schema.Document, opts ...Option) (ids []string, err error)
}
```

**核心特点**：
- **输入**：文档列表 `[]*schema.Document`
- **输出**：文档 ID 列表
- **用途**：构建向量数据库索引

---

### 8. Document Loader（文档加载器）

**定义**：从各种来源加载文档。

**位置**：`components/document/interface.go`

**接口定义**：
```go
type Loader interface {
    Load(ctx context.Context, src Source, opts ...LoaderOption) ([]*schema.Document, error)
}

type Source struct {
    URI string // 文档来源 URI
}
```

**核心特点**：
- **输入**：`Source` - 文档 URI（URL、文件路径等）
- **输出**：`[]*schema.Document` - 加载的文档列表
- **支持格式**：PDF、DOCX、TXT、Markdown 等

---

### 9. Document Transformer（文档转换器）

**定义**：对文档进行转换，如分割、过滤。

**位置**：`components/document/interface.go`

**接口定义**：
```go
type Transformer interface {
    Transform(ctx context.Context, src []*schema.Document, opts ...TransformerOption) ([]*schema.Document, error)
}
```

**核心特点**：
- **输入/输出**：都是 `[]*schema.Document`
- **常见用途**：
  - 文档分割（TextSplitter）
  - 文档过滤
  - 元数据提取

---

### 10. Chain（链式编排）

**定义**：Chain 是简单的链式有向图，只能单向前进。

**位置**：`compose/chain.go`

**核心特点**：
- **Builder 模式**：`chain.AppendXX().AppendXX().Compile()`
- **类型安全**：`NewChain[I, O]()`，泛型约束输入输出类型
- **支持组件**：ChatTemplate、ChatModel、Retriever、Embedder、ToolsNode、Lambda 等
- **支持并行**：`AppendParallel()`
- **支持分支**：`AppendBranch()`

**使用示例**：
```go
chain, _ := compose.NewChain[map[string]any, *schema.Message]().
    AppendChatTemplate(prompt).
    AppendChatModel(model).
    Compile(ctx)

message, _ := chain.Invoke(ctx, map[string]any{"query": "what's your name?"})
```

**编译过程**：
1. 自动添加 START → 第一个节点
2. 顺序连接所有节点
3. 最后节点 → END
4. 编译为 Runnable

---

### 11. Graph（图编排）

**定义**：Graph 是强大灵活的图编排，支持循环和复杂逻辑。

**位置**：`compose/graph.go`

**核心特点**：
- **两种运行模式**：
  - **Pregel**：支持循环图，适合大规模图处理（如 Agent）
  - **DAG**：有向无环图，适合标准工作流
- **节点类型**：ChatModel、Tool、Retriever、Lambda、Graph、Chain 等
- **边类型**：
  - 数据边（Data Edge）
  - 控制边（Control Edge）
  - 条件分支（Branch）
- **状态管理**：支持全局状态的读写
- **触发模式**：
  - `AnyPredecessor`：任一前驱节点完成即触发
  - `AllPredecessor`：所有前驱节点完成才触发

**使用示例**：
```go
graph := compose.NewGraph[map[string]any, *schema.Message]()

_ = graph.AddChatTemplateNode("node_template", chatTpl)
_ = graph.AddChatModelNode("node_model", chatModel)
_ = graph.AddToolsNode("node_tools", toolsNode)

_ = graph.AddEdge(compose.START, "node_template")
_ = graph.AddEdge("node_template", "node_model")
_ = graph.AddBranch("node_model", branch)
_ = graph.AddEdge("node_tools", compose.END)

compiledGraph, err := graph.Compile(ctx)
out, err := compiledGraph.Invoke(ctx, map[string]any{"query": "Beijing's weather"})
```

**Pregel 执行模型**：
1. **SuperStep**：将节点按拓扑顺序分组
2. **并发执行**：同一 SuperStep 的节点并发执行
3. **状态同步**：SuperStep 之间同步状态
4. **循环支持**：支持回到之前的节点

---

### 12. ToolsNode（工具节点）

**定义**：ToolsNode 是专门用于执行工具调用的图节点。

**位置**：`compose/tool_node.go`

**接口**：
```go
// Input: *schema.Message (AssistantMessage with ToolCalls)
// Output: []*schema.Message (ToolMessages)
Invoke(ctx context.Context, input *schema.Message, opts ...ToolsNodeOption) ([]*schema.Message, error)
Stream(ctx context.Context, input *schema.Message, opts ...ToolsNodeOption) (*schema.StreamReader[[]*schema.Message], error)
```

**核心特点**：
- **输入**：包含 ToolCalls 的 AssistantMessage
- **输出**：ToolMessage 数组，顺序对应 ToolCalls
- **并行/顺序执行**：可配置 `ExecuteSequentially`
- **错误处理**：
  - `UnknownToolsHandler`：处理幻觉工具调用
  - `ToolArgumentsHandler`：预处理工具参数
- **工具匹配**：根据 ToolCall.Function.Name 匹配工具

**配置示例**：
```go
toolsNode, _ := compose.NewToolsNode(ctx, &compose.ToolsNodeConfig{
    Tools: []tool.BaseTool{weatherTool, calculatorTool},
    UnknownToolsHandler: func(ctx context.Context, name, input string) (string, error) {
        return fmt.Sprintf("Tool %s not found", name), nil
    },
    ExecuteSequentially: false, // 并行执行
})
```

---

### 13. Lambda（自定义函数）

**定义**：Lambda 允许在编排中嵌入自定义逻辑。

**位置**：`compose/lambda.go`

**使用方式**：
```go
// 定义 Lambda 函数
lambda := func(ctx context.Context, input string) (string, error) {
    return strings.ToUpper(input), nil
}

// 在 Graph 中使用
graph.AddLambdaNode("uppercase", compose.InvokableLambda(lambda))

// 在 Chain 中使用
chain.AppendLambda(compose.InvokableLambda(lambda))
```

**核心特点**：
- **灵活性**：可以实现任意自定义逻辑
- **类型安全**：泛型约束输入输出类型
- **支持四种流范式**：Invoke、Stream、Collect、Transform

---

### 14. Message（消息）

**定义**：Message 是模型输入输出的核心数据结构。

**位置**：`schema/message.go`

**结构定义**：
```go
type Message struct {
    Role    RoleType // "user" | "assistant" | "system" | "tool"
    Content string

    // 多模态内容
    UserInputMultiContent    []MessageInputPart  // 用户输入的多模态内容
    AssistantGenMultiContent []MessageOutputPart // 模型生成的多模态内容

    // 工具调用
    ToolCalls  []ToolCall // Assistant 消息中的工具调用
    ToolCallID string     // Tool 消息的调用 ID
    ToolName   string     // Tool 消息的工具名称

    // 元信息
    ResponseMeta *ResponseMeta // FinishReason、Usage、LogProbs

    // 推理内容（如 OpenAI O1）
    ReasoningContent string

    Extra map[string]any
}
```

**核心特点**：
- **多角色**：User、Assistant、System、Tool
- **多模态支持**：文本、图片、音频、视频、文件
- **工具调用**：ToolCall 结构包含工具名称和参数
- **流拼接**：`ConcatMessages` 函数可以拼接流式消息块
- **模板支持**：实现 `MessagesTemplate` 接口

**消息创建**：
```go
schema.UserMessage("hello")
schema.SystemMessage("you are a helpful assistant")
schema.AssistantMessage("Hi there!", toolCalls)
schema.ToolMessage(result, toolCallID, schema.WithToolName("calculator"))
```

---

### 15. Document（文档）

**定义**：Document 表示一段文本及其元数据。

**位置**：`schema/document.go`

**结构定义**：
```go
type Document struct {
    ID       string
    Content  string
    MetaData map[string]any
}
```

**核心特点**：
- **ID**：唯一标识符
- **Content**：文档内容
- **MetaData**：元数据（子索引、分数、向量等）

**特殊方法**：
```go
doc.WithScore(0.95)           // 设置相关性分数
doc.WithSubIndexes([]string{...}) // 设置子索引
doc.WithExtraInfo(info)       // 设置额外信息
```

---

### 16. StreamReader/StreamWriter（流读写器）

**定义**：StreamReader 和 StreamWriter 是流处理的核心抽象。

**位置**：`schema/stream.go`, `compose/stream_reader.go`

**接口定义**：
```go
type StreamReader[T any] struct {
    // Recv 接收下一个数据块
    Recv() (T, error) // 返回 io.EOF 表示结束

    // Close 关闭流
    Close() error

    // Copy 复制流为多个独立的流
    Copy(n int) []*StreamReader[T]
}

type StreamWriter[T any] struct {
    // Send 发送数据块
    Send(T) error

    // Close 关闭流
    Close() error
}
```

**核心特点**：
- **流创建**：`schema.Pipe[T](capacity)` 创建读写对
- **流拷贝**：`Copy(n)` 将流分发到多个下游
- **流合并**：`MergeStreamReaders` 合并多个流
- **流转换**：`StreamReaderWithConvert` 转换流元素类型
- **自动处理**：框架自动处理流的拼接、分流、合并

**内部机制**：
- **Concat**：自动将流式输出拼接为完整对象（给不支持流的下游）
- **Box**：自动将非流对象包装为流（给需要流输入的下游）
- **Merge**：合并多个流到单一下游节点
- **Copy**：分发流到多个下游节点或回调

---

### 17. Callback Handler（回调处理器）

**定义**：Callback Handler 处理横切关注点，如日志、追踪、指标。

**位置**：`callbacks/interface.go`

**接口定义**：
```go
type Handler interface {
    // 五种回调时机
    OnStart(ctx context.Context, info *RunInfo, input CallbackInput) context.Context
    OnEnd(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context
    OnError(ctx context.Context, info *RunInfo, err error) context.Context
    OnStartWithStreamInput(ctx context.Context, info *RunInfo, input *schema.StreamReader[CallbackInput]) context.Context
    OnEndWithStreamOutput(ctx context.Context, info *RunInfo, output *schema.StreamReader[CallbackOutput]) context.Context
}

type RunInfo struct {
    Name      string            // 组件名称
    Type      string            // 组件类型
    Component component         // 组件标识
    Extra     map[string]any    // 额外信息
}
```

**核心特点**：
- **五种切面时机**：
  - `OnStart`：组件开始执行
  - `OnEnd`：组件成功结束
  - `OnError`：组件执行出错
  - `OnStartWithStreamInput`：流式输入开始
  - `OnEndWithStreamOutput`：流式输出结束
- **自动注入**：框架自动注入回调到所有组件
- **全局/局部**：支持全局回调和运行时回调
- **类型检查**：`TimingChecker` 接口优化回调执行

**使用示例**：
```go
handler := callbacks.NewHandlerBuilder().
    OnStartFn(func(ctx context.Context, info *RunInfo, input CallbackInput) context.Context {
        log.Infof("onStart, runInfo: %v, input: %v", info, input)
        return ctx
    }).
    OnEndFn(func(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context {
        log.Infof("onEnd, runInfo: %v, out: %v", info, output)
        return ctx
    }).
    Build()

// 全局回调
callbacks.AppendGlobalHandlers(handler)

// 运行时回调
compiledGraph.Invoke(ctx, input, compose.WithCallbacks(handler))

// 指定节点回调
compiledGraph.Invoke(ctx, input, compose.WithCallbacks(handler).DesignateNode("node_1"))

// 指定组件类型回调
compiledGraph.Invoke(ctx, input, compose.WithChatModelOption(model.WithTemperature(0.5)))
```

---

### 18. State（状态管理）

**定义**：State 用于在 Graph 执行过程中共享数据。

**位置**：在 Graph 编排中使用

**核心特点**：
- **全局共享**：Graph 中所有节点可以读写状态
- **并发安全**：框架保证状态操作的并发安全
- **自定义类型**：用户自定义状态结构体
- **State Handlers**：
  - `StatePreHandler`：节点执行前处理状态
  - `StatePostHandler`：节点执行后处理状态

**使用示例**：
```go
type MyState struct {
    Messages []*schema.Message
    Context  string
}

graph := compose.NewGraphWithState[Input, Output, *MyState](
    func(ctx context.Context) any {
        return &MyState{Messages: []*schema.Message{}}
    },
)

// 在 StatePreHandler 中读写状态
graph.AddStatePreHandler("node_1", func(ctx context.Context, state *MyState) (*MyState, error) {
    // 读取和修改状态
    state.Messages = append(state.Messages, newMessage)
    return state, nil
})
```

**典型应用**：
- **ReAct Agent**：累积对话历史
- **多轮对话**：保持上下文
- **复杂工作流**：节点间传递中间结果

---

### 19. Branch（分支）

**定义**：Branch 用于在 Graph 中实现条件路由。

**位置**：`compose/branch.go`

**核心特点**：
- **条件路由**：根据输入决定下一个执行的节点
- **支持多分支**：可以返回多个目标节点
- **类型安全**：分支函数的输入类型必须匹配上游节点输出

**分支类型**：
```go
type GraphBranch struct {
    Condition func(ctx context.Context, input any) ([]string, error) // 返回目标节点名称列表
    Then      string // 默认分支
}
```

**使用示例**：
```go
// 定义分支逻辑
branch := &compose.GraphBranch{
    Condition: func(ctx context.Context, input any) ([]string, error) {
        msg := input.(*schema.Message)
        if len(msg.ToolCalls) > 0 {
            return []string{"node_tools"}, nil // 有工具调用，执行工具节点
        }
        return []string{"node_converter"}, nil // 无工具调用，直接返回
    },
}

// 在 Graph 中添加分支
graph.AddBranch("node_model", branch)
```

**实际应用**：
- **工具调用判断**：模型输出是否包含工具调用
- **错误处理**：根据执行结果选择不同路径
- **动态路由**：基于业务逻辑的条件跳转

---

### 20. Flow（预定义流程）

**定义**：Flow 是预定义的、封装了最佳实践的复杂工作流。

**位置**：`flow/` 目录

**核心特点**：
- **开箱即用**：提供常见 AI 模式的完整实现
- **基于 Graph**：使用 Graph 编排实现
- **可定制**：支持配置和扩展

**典型 Flow：ReAct Agent**

**位置**：`flow/agent/react/react.go`

**配置**：
```go
type AgentConfig struct {
    ToolCallingModel model.ToolCallingChatModel
    ToolsConfig      compose.ToolsNodeConfig
    MessageModifier  MessageModifier // 消息预处理
    MessageRewriter  MessageModifier // 消息历史压缩
    MaxStep          int             // 最大步数
    ToolReturnDirectly map[string]struct{} // 直接返回的工具
    StreamToolCallChecker func(...) (bool, error) // 流式工具调用检查
}
```

**工作流程**：
1. **Input**：用户消息
2. **ChatModel**：生成响应或工具调用
3. **Branch**：判断是否需要调用工具
   - 有工具调用 → ToolsNode
   - 无工具调用 → 返回结果
4. **ToolsNode**：执行工具调用
5. **Loop**：将工具结果追加到消息历史，回到步骤 2
6. **Output**：最终 Assistant 消息

**使用示例**：
```go
agent, err := react.NewAgent(ctx, &react.AgentConfig{
    ToolCallingModel: model,
    ToolsConfig: compose.ToolsNodeConfig{
        Tools: []tool.BaseTool{weatherTool, calculatorTool},
    },
    MaxStep: 10,
})

message, err := agent.Generate(ctx, []*schema.Message{
    schema.UserMessage("What's the weather in Beijing and calculate 123 + 456?"),
})
```

**其他 Flow**：
- `flow/retriever/multi_query`：多查询检索器
- `flow/indexer`：文档索引流程
- 更多 Flow 在 `eino-examples` 仓库

---

## 协同工作机制

### 1. 基本工作流

```
用户输入 → ChatTemplate → ChatModel → 输出
```

**详细步骤**：
1. 用户提供输入参数（map[string]any）
2. ChatTemplate 格式化为消息列表
3. ChatModel 生成响应
4. 返回最终消息

### 2. RAG 工作流

```
用户查询 → Retriever → 检索文档 → ChatTemplate → ChatModel → 输出
```

**详细步骤**：
1. 用户提供查询字符串
2. Retriever 检索相关文档
3. 将文档和查询组合到模板中
4. ChatTemplate 格式化为消息
5. ChatModel 基于文档生成回答

### 3. 工具调用工作流（ReAct）

```
用户输入 → ChatModel → [判断] → ToolsNode → [循环] → 最终输出
                           ↓
                       直接输出
```

**详细步骤**：
1. 用户消息 → ChatModel（绑定工具信息）
2. ChatModel 决定：
   - **生成答案**：直接返回 Assistant 消息
   - **调用工具**：返回包含 ToolCalls 的 Assistant 消息
3. 如果有 ToolCalls：
   - ToolsNode 执行工具调用
   - 生成 ToolMessage（包含工具执行结果）
   - 将 ToolMessage 追加到消息历史
   - 回到步骤 1（ChatModel 重新决策）
4. 循环直到 ChatModel 生成最终答案或达到最大步数

### 4. 文档索引工作流

```
文档源 → Document Loader → Document Transformer → Embedder → Indexer → 向量数据库
```

**详细步骤**：
1. Document Loader 加载文档（PDF、DOCX等）
2. Document Transformer 分割文档（TextSplitter）
3. Embedder 生成文档向量
4. Indexer 存储到向量数据库

### 5. 编排工作流

**Chain 编排**：
```
Input → Node1 → Node2 → Node3 → Output
```
- 简单线性流程
- 支持并行节点
- 支持分支

**Graph 编排（Pregel）**：
```
         ┌─→ NodeB ─┐
Input → NodeA      → NodeD → Output
         └─→ NodeC ─┘
              ↓ (loop)
           [State]
```
- 支持循环
- 支持状态管理
- 支持条件分支
- 支持并发执行

### 6. 流处理工作流

**四种流范式自动转换**：

```
组件A (Invoke) → 框架 (Box) → 组件B (Transform)
      ↓                             ↓
   非流输出      →   包装为流   →  流输入

组件C (Stream) → 框架 (Concat) → 组件D (Invoke)
      ↓                              ↓
    流输出      →   拼接为完整   →  非流输入
```

**流的分发和合并**：
```
                ┌→ Node2
StreamReader → Copy(3) → Node3
                └→ Callback

Node1 → StreamReader ┐
Node2 → StreamReader  → Merge → Node4
Node3 → StreamReader ┘
```

---

## 设计原则

### 1. 组件抽象化

**原则**：每种功能都定义为清晰的接口，实现细节对用户透明。

**好处**：
- 易于替换实现（如切换不同的 LLM 提供商）
- 统一的使用方式
- 便于测试（可以 Mock）

**示例**：
```go
// 接口定义
type ChatModel interface {
    Generate(ctx, messages, opts) (message, error)
}

// 不同实现
openai.NewChatModel(config)   // OpenAI 实现
claude.NewChatModel(config)   // Anthropic 实现
custom.NewChatModel(config)   // 自定义实现

// 使用时无差别
model.Generate(ctx, messages)
```

### 2. 类型安全

**原则**：使用 Go 泛型确保编译时类型检查。

**好处**：
- 提前发现类型错误
- 减少运行时错误
- 更好的 IDE 支持

**示例**：
```go
// 编译时检查类型匹配
chain := NewChain[map[string]any, *schema.Message]()
chain.AppendChatTemplate(template) // template 输出必须是 []*schema.Message
chain.AppendChatModel(model)       // model 输入必须是 []*schema.Message
```

### 3. 流处理优先

**原则**：默认支持流式处理，自动处理流和非流的转换。

**好处**：
- 更好的用户体验（实时响应）
- 降低首字节延迟
- 自动适配不同组件的流能力

**实现**：
- 框架自动 concat 流为完整对象
- 框架自动 box 非流对象为流
- 框架自动 merge/copy 流

### 4. 编排能力分层

**三层编排 API**：

| API      | 特点                     | 适用场景                 |
| -------- | ------------------------ | ------------------------ |
| Chain    | 简单链式，Builder 模式   | 简单顺序流程             |
| Graph    | 强大灵活，支持循环和状态 | 复杂 AI 应用（Agent）    |
| Workflow | 字段级映射，结构化数据   | 数据转换和集成（Alpha）  |

**渐进式复杂度**：
- 简单任务用 Chain
- 复杂任务用 Graph
- 特殊需求用 Workflow

### 5. 可观测性

**原则**：通过 Callback 机制提供完整的可观测性。

**支持的横切关注点**：
- **日志**：记录每个组件的输入输出
- **追踪**：分布式追踪（OpenTelemetry）
- **指标**：性能指标（延迟、吞吐量）
- **调试**：可视化调试工具

**实现**：
```go
// 全局回调（所有组件）
callbacks.AppendGlobalHandlers(tracingHandler, loggingHandler)

// 运行时回调（特定执行）
graph.Invoke(ctx, input, compose.WithCallbacks(debugHandler))

// 节点级回调（特定节点）
graph.Invoke(ctx, input, compose.WithCallbacks(handler).DesignateNode("critical_node"))
```

### 6. 组合优于继承

**原则**：组件可以嵌套，复杂组件由简单组件组合而成。

**好处**：
- 更灵活的代码重用
- 降低耦合度
- 易于扩展

**示例**：
```go
// MultiQueryRetriever 组合了多个 Retriever
type MultiQueryRetriever struct {
    baseRetriever retriever.Retriever
    queryExpander Runnable[string, []string]
}

// 从外部看，它仍然是一个 Retriever
var _ retriever.Retriever = &MultiQueryRetriever{}
```

### 7. 配置与代码分离

**原则**：组件配置通过结构体传递，而非硬编码。

**好处**：
- 易于测试（不同配置）
- 易于维护
- 支持配置文件/环境变量

**示例**：
```go
config := &openai.ChatModelConfig{
    APIKey:      os.Getenv("OPENAI_API_KEY"),
    Model:       "gpt-4",
    Temperature: 0.7,
}
model, err := openai.NewChatModel(ctx, config)
```

---

## 架构图

### 整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        Eino Framework                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                   Orchestration Layer                    │    │
│  │                                                           │    │
│  │  ┌────────┐    ┌────────┐    ┌──────────┐              │    │
│  │  │ Chain  │    │ Graph  │    │ Workflow │              │    │
│  │  └────────┘    └────────┘    └──────────┘              │    │
│  │       │              │              │                    │    │
│  │       └──────────────┴──────────────┘                    │    │
│  │                      │                                    │    │
│  │                 ┌────▼────┐                              │    │
│  │                 │ Runnable │  ← 核心抽象                │    │
│  │                 └─────────┘                              │    │
│  └─────────────────────────────────────────────────────────┘    │
│                          │                                       │
│  ┌───────────────────────┼───────────────────────────────────┐  │
│  │              Component Abstraction Layer                   │  │
│  │                       │                                     │  │
│  │  ┌──────────┐  ┌─────┴──────┐  ┌──────────┐             │  │
│  │  │ChatModel │  │   Tool     │  │ Retriever│             │  │
│  │  └──────────┘  └────────────┘  └──────────┘             │  │
│  │                                                            │  │
│  │  ┌──────────┐  ┌────────────┐  ┌──────────┐             │  │
│  │  │ Embedder │  │  Indexer   │  │ Template │             │  │
│  │  └──────────┘  └────────────┘  └──────────┘             │  │
│  │                                                            │  │
│  │  ┌──────────┐  ┌────────────┐  ┌──────────┐             │  │
│  │  │  Loader  │  │Transformer │  │  Lambda  │             │  │
│  │  └──────────┘  └────────────┘  └──────────┘             │  │
│  └────────────────────────────────────────────────────────────┘  │
│                          │                                       │
│  ┌───────────────────────┼───────────────────────────────────┐  │
│  │                  Schema Layer                              │  │
│  │                       │                                     │  │
│  │  ┌─────────┐   ┌─────▼──────┐   ┌────────────┐           │  │
│  │  │ Message │   │  Document  │   │StreamReader│           │  │
│  │  └─────────┘   └────────────┘   └────────────┘           │  │
│  └────────────────────────────────────────────────────────────┘  │
│                          │                                       │
│  ┌───────────────────────┼───────────────────────────────────┐  │
│  │               Cross-Cutting Concerns                       │  │
│  │                       │                                     │  │
│  │         ┌─────────────┴────────────┐                       │  │
│  │         │   Callback Mechanism     │                       │  │
│  │         │  (Logging, Tracing, etc) │                       │  │
│  │         └──────────────────────────┘                       │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

### ReAct Agent 流程图

```
                    ┌──────────────┐
                    │  User Input  │
                    └──────┬───────┘
                           │
                    ┌──────▼───────┐
                    │   ChatModel  │ ← 绑定工具信息
                    │  (with Tools)│
                    └──────┬───────┘
                           │
                    ┌──────▼───────┐
                    │   Branch     │
                    └──────┬───────┘
                           │
              ┌────────────┴────────────┐
              │                         │
     ┌────────▼────────┐      ┌────────▼─────────┐
     │  Has ToolCalls? │      │ No ToolCalls     │
     │      YES        │      │ Return Message   │
     └────────┬────────┘      └──────────────────┘
              │
     ┌────────▼────────┐
     │   ToolsNode     │
     │ Execute Tools   │
     └────────┬────────┘
              │
     ┌────────▼────────┐
     │  Append Tool    │
     │  Messages to    │
     │  State.Messages │
     └────────┬────────┘
              │
              └──────────┐
                         │ Loop back
                         ▼
                    (ChatModel)
```

### 流处理机制图

```
┌──────────────────────────────────────────────────────────┐
│              Stream Processing Mechanism                  │
├──────────────────────────────────────────────────────────┤
│                                                           │
│  Component A                     Component B              │
│  (Stream)                        (Invoke)                │
│      │                               │                    │
│      │  StreamReader[T]              │                    │
│      └────────┬──────────────────────┘                    │
│               │                                           │
│          ┌────▼────┐                                      │
│          │ Concat  │ ← 框架自动拼接                       │
│          └────┬────┘                                      │
│               │  T                                        │
│               └─────────────→ Component B                 │
│                                                           │
│  ─────────────────────────────────────────────────────   │
│                                                           │
│  Component C                     Component D              │
│  (Invoke)                        (Transform)             │
│      │                               │                    │
│      │  T                            │                    │
│      └────────┬──────────────────────┘                    │
│               │                                           │
│          ┌────▼────┐                                      │
│          │   Box   │ ← 框架自动包装                       │
│          └────┬────┘                                      │
│               │  StreamReader[T]                          │
│               └─────────────→ Component D                 │
│                                                           │
│  ─────────────────────────────────────────────────────   │
│                                                           │
│  StreamReader[T]                                          │
│      │                                                    │
│      └─────┬─────────┐                                    │
│            │         │                                    │
│       ┌────▼────┐    └────────┐                          │
│       │ Copy(3) │             │                          │
│       └────┬────┘             │                          │
│            │                  │                          │
│      ┌─────┴─────┬────────────┘                          │
│      │           │                                       │
│   Node1       Node2        Callback                      │
│                                                           │
└──────────────────────────────────────────────────────────┘
```

---

## 总结

Eino 框架通过**组件抽象化**、**强大的编排能力**、**完整的流处理**和**类型安全**，为 Go 开发者提供了构建复杂 AI 应用的完整解决方案。

**核心优势**：
1. **简洁的 API**：符合 Go 语言习惯，易于上手
2. **类型安全**：泛型确保编译时类型检查
3. **高度可扩展**：组件化设计，易于替换和扩展
4. **完善的流处理**：自动处理流式数据，提升用户体验
5. **强大的编排**：Chain、Graph、Workflow 满足不同需求
6. **可观测性**：完善的回调机制支持日志、追踪、调试

**适用场景**：
- **AI 助手**：多轮对话、工具调用
- **RAG 应用**：检索增强生成
- **Agent 系统**：自主决策、复杂推理
- **文档处理**：加载、分割、索引、检索
- **工作流自动化**：复杂业务流程编排

---

**文档版本**：1.0
**框架版本**：基于 github.com/cloudwego/eino (2024年11月)
**分析时间**：2025年11月7日
