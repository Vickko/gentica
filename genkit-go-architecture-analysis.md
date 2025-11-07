# Genkit Go 框架架构分析文档

> 基于源码分析 - 版本：Genkit Go 1.0
> 分析日期：2025-11-06
> 仓库：https://github.com/firebase/genkit

## 目录

- [概述](#概述)
- [一、核心架构实体](#一核心架构实体)
  - [0. Genkit Instance (Genkit 实例)](#0-genkit-instance-genkit-实例)
  - [1. ActionDef (Action Definition)](#1-actiondef-action-definition)
  - [2. Flow (流程)](#2-flow-流程)
  - [3. Registry (注册表)](#3-registry-注册表)
  - [4. Plugin (插件)](#4-plugin-插件)
- [二、AI 功能实体](#二ai-功能实体)
  - [5. Model (模型)](#5-model-模型)
  - [6. Tool (工具)](#6-tool-工具)
  - [7. Embedder (嵌入器)](#7-embedder-嵌入器)
  - [8. Retriever (检索器)](#8-retriever-检索器)
  - [9. Prompt (提示词)](#9-prompt-提示词)
  - [10. Evaluator (评估器)](#10-evaluator-评估器)
- [三、实体协同工作方式](#三实体协同工作方式)
- [四、关键设计原则](#四关键设计原则)

---

## 概述

Genkit Go 是 Google 开发的开源 AI 应用开发框架，专为 Go 生态设计。它提供了一个轻量级、provider 无关的框架，通过统一的抽象和接口简化 AI 应用的开发、调试和部署。

**核心特点：**
- 基于 ActionDef 的统一抽象
- 内置可观察性（tracing 和 metrics）
- 插件化架构
- 类型安全（Go 泛型）
- 支持流式输出
- Production-ready 设计

---

## 一、核心架构实体

### 0. Genkit Instance (Genkit 实例)

**源码位置**: `go/genkit/genkit.go:36-43`

Genkit 实例是整个框架的入口点和中心枢纽，封装了 Registry 并提供友好的 API。

#### 核心结构

```go
// Genkit 封装了一个 Genkit 实例，提供对其注册表、配置和核心功能的访问。
// 它作为定义和管理 Genkit 资源（如 flows、models、tools 和 prompts）的中心枢纽。
type Genkit struct {
    reg *registry.Registry  // 用于 actions、values 和其他资源的注册表
}
```

#### 核心特性

- **Registry 封装**：Genkit 实例本质上是对 `Registry` 的封装，提供更友好的 API
- **门面模式**：隐藏了底层 Registry 的复杂性，提供简洁的接口
- **统一入口**：所有 Genkit 操作的中心访问点
- **生命周期管理**：管理插件初始化、开发服务器等

#### 创建 Genkit 实例

```go
// Init 创建并初始化一个新的 Genkit 实例
g := genkit.Init(ctx,
    genkit.WithPlugins(
        googlegenai.NewPlugin(),  // 初始化插件
        pinecone.NewPlugin(),
    ),
    genkit.WithDefaultModel("googleai/gemini-1.5-flash"),  // 设置默认模型
    genkit.WithPromptDir("./prompts"),  // 加载提示词目录
)
```

#### Init 过程详解

```go
func Init(ctx context.Context, opts ...GenkitOption) *Genkit
```

`Init` 函数执行以下步骤：

1. **创建 Registry**
   ```go
   r := registry.New()
   g := &Genkit{reg: r}
   ```

2. **初始化插件**
   ```go
   for _, plugin := range gOpts.Plugins {
       actions := plugin.Init(ctx)  // 调用插件的 Init 方法
       for _, action := range actions {
           action.Register(r)       // 注册插件提供的 actions
       }
       r.RegisterPlugin(plugin.Name(), plugin)
   }
   ```

3. **配置 AI 组件**
   ```go
   ai.ConfigureFormats(r)        // 配置输出格式（json, text 等）
   ai.DefineGenerateAction(ctx, r)  // 定义 generate action
   ```

4. **加载提示词**
   ```go
   ai.LoadPromptDir(r, gOpts.PromptDir, "")  // 从目录加载 .prompt 文件
   ```

5. **注册配置值**
   ```go
   r.RegisterValue(api.DefaultModelKey, gOpts.DefaultModel)
   r.RegisterValue(api.PromptDirKey, gOpts.PromptDir)
   ```

6. **启动开发服务器**（仅在 dev 环境）
   ```go
   if api.CurrentEnvironment() == api.EnvironmentDev {
       go startReflectionServer(ctx, g, ...)  // 启动 Reflection API (端口 3100)
   }
   ```

#### Genkit 实例提供的 API

Genkit 实例封装 Registry，提供以下便捷方法：

**定义组件**：
```go
// Flows
flow := genkit.DefineFlow(g, name, fn)
streamFlow := genkit.DefineStreamingFlow(g, name, fn)

// Models
model := genkit.DefineModel(g, name, opts, fn)

// Tools
tool := genkit.DefineTool(g, name, desc, fn)

// Prompts
prompt := genkit.DefinePrompt(g, name, opts...)

// Retrievers, Embedders, Evaluators 等
retriever := genkit.DefineRetriever(g, name, opts, fn)
embedder := genkit.DefineEmbedder(g, name, opts, fn)
evaluator := genkit.DefineEvaluator(g, name, opts, fn)
```

**查找组件**：
```go
model := genkit.LookupModel(g, "gemini-1.5-flash")
tool := genkit.LookupTool(g, "search")
prompt := genkit.LookupPrompt(g, "summarize")
plugin := genkit.LookupPlugin(g, "googleai")
```

**执行操作**：
```go
// 生成
resp, _ := genkit.Generate(ctx, g, ai.WithPrompt("Hello"))
text, _ := genkit.GenerateText(ctx, g, ai.WithPrompt("Hello"))
data, _, _ := genkit.GenerateData[MyType](ctx, g, opts...)

// 检索和嵌入
docs, _ := genkit.Retrieve(ctx, g, ai.WithRetriever("myDB"), ...)
embeddings, _ := genkit.Embed(ctx, g, ai.WithEmbedder("text-embedding"), ...)

// 评估
results, _ := genkit.Evaluate(ctx, g, ai.WithEvaluator("faithfulness"), ...)
```

**列出组件**：
```go
flows := genkit.ListFlows(g)      // 所有 flows
tools := genkit.ListTools(g)      // 所有 tools
resources := genkit.ListResources(g)  // 所有 resources
```

#### Genkit 实例 vs Registry

```
┌─────────────────────────────────────────────────────────┐
│               Genkit Instance (门面层)                   │
│  ┌───────────────────────────────────────────────────┐  │
│  │  高级 API:                                        │  │
│  │  - DefineFlow(g, ...)                            │  │
│  │  - Generate(ctx, g, ...)                         │  │
│  │  - LookupModel(g, ...)                           │  │
│  └─────────────────┬───────────────────────────────┘  │
│                    │ 委托调用                           │
│  ┌─────────────────▼───────────────────────────────┐  │
│  │          Registry (核心层)                       │  │
│  │  ┌────────────────────────────────────────────┐ │  │
│  │  │ 底层操作:                                  │ │  │
│  │  │ - RegisterAction(key, action)             │ │  │
│  │  │ - LookupAction(key)                       │ │  │
│  │  │ - ResolveAction(key)                      │ │  │
│  │  └────────────────────────────────────────────┘ │  │
│  └───────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

**为什么需要 Genkit 实例？**

1. **简化 API**：
   ```go
   // 使用 Genkit 实例（简洁）
   genkit.DefineFlow(g, "myFlow", fn)

   // 直接使用 Registry（繁琐）
   core.DefineFlow(g.reg, "myFlow", fn)
   ```

2. **统一管理**：
   - 一个应用通常只有一个 Genkit 实例
   - 集中管理所有 AI 组件
   - 便于依赖注入和测试

3. **生命周期控制**：
   - 插件初始化
   - 开发服务器启动/关闭
   - 资源清理

4. **环境感知**：
   - 开发环境：启动 Reflection API 服务器
   - 生产环境：仅核心功能

#### 完整使用示例

```go
package main

import (
    "context"
    "fmt"

    "github.com/firebase/genkit/go/ai"
    "github.com/firebase/genkit/go/genkit"
    "github.com/firebase/genkit/go/plugins/googlegenai"
)

func main() {
    ctx := context.Background()

    // 1. 初始化 Genkit 实例
    g := genkit.Init(ctx,
        genkit.WithPlugins(googlegenai.NewPlugin()),
        genkit.WithDefaultModel("googleai/gemini-1.5-flash"),
    )

    // 2. 定义组件（使用 Genkit 实例）
    myFlow := genkit.DefineFlow(g, "greetingFlow",
        func(ctx context.Context, name string) (string, error) {
            // 3. 在 flow 中使用 Genkit 实例的方法
            resp, err := genkit.Generate(ctx, g,
                ai.WithPrompt(fmt.Sprintf("问候 %s", name)),
            )
            if err != nil {
                return "", err
            }
            return resp.Text(), nil
        },
    )

    // 4. 运行 flow
    result, err := myFlow.Run(ctx, "世界")
    if err != nil {
        panic(err)
    }

    fmt.Println(result)  // 输出问候语
}
```

#### 关键设计理念

1. **单一实例模式**：通常一个应用只创建一个 Genkit 实例
2. **依赖注入**：将 `*Genkit` 传递给需要它的函数
3. **声明式配置**：通过 `GenkitOption` 配置实例
4. **延迟初始化**：某些组件（如动态插件）按需解析

---

### 1. ActionDef (Action Definition)

**源码位置**: `go/core/action.go:42-52`

ActionDef 是 Genkit 最基础的抽象概念，代表任何需要可观察性、监控和调试能力的可追踪操作。

#### 核心结构

```go
type ActionDef[In, Out, Stream any] struct {
    fn   StreamingFunc[In, Out, Stream]  // 运行时调用的函数
    desc *api.ActionDesc                 // Action 的描述符
}
```

#### 类型定义

```go
// 非流式函数
type Func[In, Out any] = func(context.Context, In) (Out, error)

// 流式函数
type StreamingFunc[In, Out, Stream any] = func(context.Context, In, StreamCallback[Stream]) (Out, error)

// 流式回调
type StreamCallback[Stream any] = func(context.Context, Stream) error
```

#### 核心特性

- **泛型参数**：
  - `In`: 输入类型
  - `Out`: 输出类型
  - `Stream`: 流式数据块类型

- **自动能力**：
  - 每次运行创建新的 trace span
  - 自动进行输入/输出 JSON Schema 验证
  - 记录执行指标（延迟、成功/失败）
  - 支持流式和非流式两种模式

#### 主要方法

```go
// 执行 Action
func (a *ActionDef[In, Out, Stream]) Run(ctx context.Context, input In, cb StreamCallback[Stream]) (Out, error)

// 使用 JSON 输入/输出执行
func (a *ActionDef[In, Out, Stream]) RunJSON(ctx context.Context, input json.RawMessage, cb StreamCallback[json.RawMessage]) (json.RawMessage, error)

// 执行并返回遥测信息
func (a *ActionDef[In, Out, Stream]) runWithTelemetry(ctx context.Context, input In, cb StreamCallback[Stream]) (api.ActionRunResult[Out], error)

// 注册到 Registry
func (a *ActionDef[In, Out, Stream]) Register(r api.Registry)
```

#### 创建 Action

```go
// 创建非流式 Action
action := core.NewAction[InputType, OutputType](
    name,
    actionType,
    metadata,
    inputSchema,
    func(ctx context.Context, input InputType) (OutputType, error) {
        // 实现逻辑
    },
)

// 创建并注册非流式 Action
action := core.DefineAction[InputType, OutputType](
    registry,
    name,
    actionType,
    metadata,
    inputSchema,
    fn,
)

// 创建流式 Action
action := core.NewStreamingAction[InputType, OutputType, StreamType](
    name,
    actionType,
    metadata,
    inputSchema,
    func(ctx context.Context, input InputType, cb func(context.Context, StreamType) error) (OutputType, error) {
        // 实现逻辑
    },
)
```

---

### 2. Flow (流程)

**源码位置**: `go/core/flow.go:30-31`

Flow 是用户定义的 Action，用于构建生产就绪的 AI 工作流。实际上是 ActionDef 的类型别名。

#### 类型定义

```go
type Flow[In, Out, Stream any] ActionDef[In, Out, Stream]
```

#### 核心特性

- **类型安全**：使用 Go struct 提供编译时类型检查
- **内置可观察性**：自动 tracing 和 metrics
- **流式支持**：可选的增量输出
- **Developer UI 集成**：可在 UI 中交互式运行和调试
- **轻松部署**：可作为 HTTP 端点部署，最小样板代码
- **步骤缓存**：Flow 内的步骤结果会被缓存，重启时不会重复执行

#### 定义 Flow

```go
// 非流式 Flow
flow := core.DefineFlow[InputType, OutputType](
    registry,
    "myFlow",
    func(ctx context.Context, input InputType) (OutputType, error) {
        // 实现 Flow 逻辑
        return output, nil
    },
)

// 流式 Flow
flow := core.DefineStreamingFlow[InputType, OutputType, StreamType](
    registry,
    "myStreamingFlow",
    func(ctx context.Context, input InputType, cb func(context.Context, StreamType) error) (OutputType, error) {
        // 可以通过 cb 发送流式数据
        cb(ctx, chunk)
        return output, nil
    },
)
```

#### Flow Context

Flow 执行时会创建一个特殊的 context：

```go
type flowContext struct {
    flowName string
}
```

可以通过 `FlowNameFromContext(ctx)` 获取当前 flow 名称。

#### Flow 内的步骤

```go
// 在 Flow 内创建可缓存的步骤
result, err := core.Run[OutputType](ctx, "stepName", func() (OutputType, error) {
    // 步骤逻辑
    return result, nil
})
```

每个步骤：
- 有自己的 trace span
- 结果会被缓存
- Flow 重启时不会重复执行

#### 运行 Flow

```go
// 基本运行
output, err := flow.Run(ctx, input)

// 流式运行
flow.Stream(ctx, input)(func(value *core.StreamingFlowValue[OutputType, StreamType], err error) bool {
    if err != nil {
        // 处理错误
        return false
    }
    if value.Done {
        // 最终输出
        finalOutput := value.Output
    } else {
        // 流式数据块
        chunk := value.Stream
    }
    return true // 继续接收
})
```

---

### 3. Registry (注册表)

**源码位置**: `go/core/api/registry.go:23-86`

Registry 是中央查找服务，管理所有 Genkit 组件的注册、查询和解析。

#### 核心接口

```go
type Registry interface {
    // 创建子注册表
    NewChild() Registry
    IsChild() bool

    // 注册组件
    RegisterPlugin(name string, p Plugin)
    RegisterAction(key string, action Action)
    RegisterValue(name string, value any)

    // 查找组件（先查当前，再查父级）
    LookupPlugin(name string) Plugin
    LookupAction(key string) Action
    LookupValue(name string) any

    // 动态解析（支持 DynamicPlugin）
    ResolveAction(key string) Action

    // 列出所有组件
    ListActions() []Action
    ListPlugins() []Plugin
    ListValues() map[string]any

    // Dotprompt 相关
    RegisterPartial(name string, source string)
    RegisterHelper(name string, fn any)
    Dotprompt() *dotprompt.Dotprompt
}
```

#### 层级结构

Registry 支持父子关系：

```
Parent Registry
    ├─> Child Registry 1
    │   └─> Child Registry 1.1
    └─> Child Registry 2
```

**查找规则**：
1. 先在当前注册表查找
2. 如果未找到，向上查找父注册表
3. 子注册表的组件会覆盖父注册表的同名组件

#### Action Key 格式

```go
// Action 通过 key 标识
key := api.NewKey(actionType, provider, id)
// 格式: "/actionType/provider/id" 或 "/actionType/id"

// 解析名称
provider, id := api.ParseName("google/gemini-1.5-flash")
// provider = "google", id = "gemini-1.5-flash"
```

#### 使用示例

```go
// 注册组件
registry.RegisterAction("/flow/myFlow", flowAction)
registry.RegisterPlugin("vertexai", vertexAIPlugin)

// 查找组件
action := registry.LookupAction("/model/gemini-1.5-flash")

// 解析组件（支持动态插件）
action := registry.ResolveAction("/model/some-dynamic-model")

// 创建子注册表
childRegistry := registry.NewChild()
childRegistry.RegisterAction("/flow/childFlow", childFlowAction)
```

---

### 4. Plugin (插件)

**源码位置**: `go/core/api/plugin.go:23-42`

Plugin 是扩展 Genkit 功能的可配置模块，用于集成外部服务（如模型提供商、向量数据库、监控工具等）。

#### 核心接口

```go
type Plugin interface {
    // 返回插件的唯一标识符
    Name() string

    // 初始化插件，返回注册的 Actions
    Init(ctx context.Context) []Action
}
```

#### 动态插件

```go
type DynamicPlugin interface {
    Plugin

    // 列出插件可以解析的 Action 描述符
    ListActions(ctx context.Context) []ActionDesc

    // 动态解析 Action
    ResolveAction(atype ActionType, name string) Action
}
```

#### 插件实现示例

```go
type MyPlugin struct {
    apiKey string
    config Config
}

func (p *MyPlugin) Name() string {
    return "myplugin"
}

func (p *MyPlugin) Init(ctx context.Context) []Action {
    return []Action{
        // 注册多个 models
        ai.DefineModel(registry, "myplugin/model-1", opts1, fn1),
        ai.DefineModel(registry, "myplugin/model-2", opts2, fn2),

        // 注册 embedder
        ai.DefineEmbedder(registry, "myplugin/embedder", embOpts, embFn),

        // 注册 retriever
        ai.DefineRetriever(registry, "myplugin/retriever", retOpts, retFn),
    }
}
```

#### 动态插件示例

```go
type MyDynamicPlugin struct {
    MyPlugin
}

func (p *MyDynamicPlugin) ListActions(ctx context.Context) []ActionDesc {
    return []ActionDesc{
        {Type: api.ActionTypeModel, Name: "myplugin/dynamic-model-1"},
        {Type: api.ActionTypeModel, Name: "myplugin/dynamic-model-2"},
    }
}

func (p *MyDynamicPlugin) ResolveAction(atype ActionType, name string) Action {
    // 按需创建和返回 Action
    if atype == api.ActionTypeModel {
        return createModelAction(name)
    }
    return nil
}
```

#### 插件注册

```go
// 注册插件
registry.RegisterPlugin("myplugin", &MyPlugin{
    apiKey: "xxx",
    config: config,
})

// 初始化所有插件
for _, plugin := range registry.ListPlugins() {
    actions := plugin.Init(ctx)
    for _, action := range actions {
        registry.RegisterAction(action.Key(), action)
    }
}
```

---

## 二、AI 功能实体

### 5. Model (模型)

**源码位置**: `go/ai/generate.go:35-43`

Model 代表可以基于请求生成内容的 AI 模型。

#### 核心接口

```go
type Model interface {
    // 返回模型的注册名称
    Name() string

    // 应用模型到请求，处理工具请求和流式输出
    Generate(ctx context.Context, req *ModelRequest, cb ModelStreamCallback) (*ModelResponse, error)

    // 注册到 Registry
    Register(r api.Registry)
}
```

#### 内部实现

Model 本质上是 ActionDef 的封装：

```go
type model struct {
    core.ActionDef[*ModelRequest, *ModelResponse, *ModelResponseChunk]
}
```

#### 模型能力 (ModelSupports)

```go
type ModelSupports struct {
    Constrained ConstrainedSupport  // 受约束输出支持
    ContentType []string            // 支持的内容类型
    Context     bool                // 支持上下文
    Media       bool                // 支持多模态（图像、音频等）
    Multiturn   bool                // 支持多轮对话
    Output      []string            // 支持的输出格式
    SystemRole  bool                // 支持系统角色
    ToolChoice  bool                // 支持工具选择控制
    Tools       bool                // 支持工具调用
}
```

#### 定义 Model

```go
model := ai.DefineModel(
    registry,
    "google/gemini-1.5-flash",
    &ai.ModelOptions{
        Label: "Gemini 1.5 Flash",
        Stage: ai.ModelStageStable,
        Supports: &ai.ModelSupports{
            Media:      true,
            Multiturn:  true,
            SystemRole: true,
            Tools:      true,
            ToolChoice: true,
        },
    },
    func(ctx context.Context, req *ai.ModelRequest, cb ai.ModelStreamCallback) (*ai.ModelResponse, error) {
        // 实现模型调用逻辑
        return response, nil
    },
)
```

#### ModelRequest 结构

```go
type ModelRequest struct {
    Config     any                // 模型配置（温度、top-p 等）
    Docs       []*Document        // 上下文文档（用于 RAG）
    Messages   []*Message         // 对话消息历史
    Output     *ModelOutputConfig // 输出格式配置
    ToolChoice ToolChoice         // 工具选择策略
    Tools      []*ToolDefinition  // 可用工具列表
}
```

#### ModelResponse 结构

```go
type ModelResponse struct {
    Custom        any            // 自定义数据
    FinishMessage string         // 完成消息
    FinishReason  FinishReason   // 完成原因
    LatencyMs     float64        // 延迟（毫秒）
    Message       *Message       // 生成的消息
    Request       *ModelRequest  // 原始请求
    Usage         *GenerationUsage // 资源使用情况
}
```

#### 中间件机制

Model 使用中间件链处理请求：

```go
type ModelMiddleware = core.Middleware[*ModelRequest, *ModelResponse, *ModelResponseChunk]

// 内置中间件
mws := []ModelMiddleware{
    simulateSystemPrompt(opts, nil),    // 模拟系统提示词（对不支持的模型）
    augmentWithContext(opts, nil),      // 将文档注入到消息中
    validateSupport(name, opts),        // 验证请求与模型能力匹配
    addAutomaticTelemetry(),           // 添加自动遥测
}
fn = core.ChainMiddleware(mws...)(fn)
```

#### 生成内容

```go
// 基本生成
response, err := ai.Generate(ctx, registry,
    ai.WithModel("gemini-1.5-flash"),
    ai.WithMessages(ai.NewUserMessage(ai.NewTextPart("Hello!"))),
)

// 带工具的生成
response, err := ai.Generate(ctx, registry,
    ai.WithModel("gemini-1.5-flash"),
    ai.WithTools(searchTool, calculatorTool),
    ai.WithMessages(...),
)

// 流式生成
ai.GenerateStream(ctx, registry,
    ai.WithModel("gemini-1.5-flash"),
    ai.WithMessages(...),
)(func(chunk *ai.ModelResponseChunk, err error) bool {
    if err != nil {
        return false
    }
    // 处理流式块
    return true
})
```

---

### 6. Tool (工具)

**源码位置**: `go/ai/tools.go:59-74`

Tool 代表可被 AI 模型调用的功能。

#### 核心接口

```go
type Tool interface {
    // 返回工具名称
    Name() string

    // 返回工具定义（用于传递给模型）
    Definition() *ToolDefinition

    // 使用原始输入运行工具
    RunRaw(ctx context.Context, input any) (any, error)

    // 构造工具响应（用于中断场景）
    Respond(toolReq *Part, outputData any, opts *RespondOptions) *Part

    // 重启工具请求（用于中断场景）
    Restart(toolReq *Part, opts *RestartOptions) *Part

    // 注册到 Registry
    Register(r api.Registry)
}
```

#### ToolContext

工具函数接收特殊的 context：

```go
type ToolContext struct {
    context.Context

    // 中断工具执行，返回控制权给调用者
    Interrupt func(opts *InterruptOptions) error

    // 恢复数据（仅在工具被中断后有值）
    Resumed map[string]any

    // 原始输入（仅在工具被中断后有值）
    OriginalInput any
}
```

#### 工具函数类型

```go
type ToolFunc[In, Out any] = func(ctx *ToolContext, input In) (Out, error)
```

#### 定义工具

```go
// 定义并注册工具
tool := ai.DefineTool[SearchInput, SearchOutput](
    registry,
    "search",
    "搜索互联网获取信息",
    func(ctx *ai.ToolContext, input SearchInput) (SearchOutput, error) {
        // 工具实现
        results := performSearch(input.Query)
        return SearchOutput{Results: results}, nil
    },
)

// 使用自定义 schema 定义工具
tool := ai.DefineToolWithInputSchema[Output](
    registry,
    "customTool",
    "描述",
    customInputSchema,
    func(ctx *ai.ToolContext, input any) (Output, error) {
        // 实现
    },
)

// 创建工具（不注册，直接传递给 Generate）
tool := ai.NewTool[Input, Output](
    "dynamicTool",
    "描述",
    fn,
)
```

#### 工具中断机制

工具可以中断执行，将控制权返回给用户：

```go
tool := ai.DefineTool[Input, Output](
    registry,
    "needsConfirmation",
    "需要用户确认的工具",
    func(ctx *ai.ToolContext, input Input) (Output, error) {
        // 检查是否需要确认
        if needsConfirmation(input) {
            return Output{}, ctx.Interrupt(&ai.InterruptOptions{
                Metadata: map[string]any{
                    "reason": "需要用户确认",
                    "data":   input,
                },
            })
        }

        // 正常执行
        return performAction(input), nil
    },
)

// 处理中断
response, err := ai.Generate(ctx, registry, ...)
if err != nil {
    return err
}

// 检查是否有中断的工具请求
for _, part := range response.Message.Content {
    if part.IsToolRequest() {
        if interrupted, metadata := ai.IsToolInterruptError(part.Metadata["interrupt"]); interrupted {
            // 获取用户确认后

            // 选项 1: 响应工具请求
            toolResp := tool.Respond(part, result, nil)
            response, _ = ai.Generate(ctx, registry,
                ai.WithToolResponses(toolResp),
                // ... 其他选项
            )

            // 选项 2: 重启工具请求
            toolReq := tool.Restart(part, &ai.RestartOptions{
                ReplaceInput: newInput,
            })
            response, _ = ai.Generate(ctx, registry,
                ai.WithToolRestarts(toolReq),
                // ... 其他选项
            )
        }
    }
}
```

#### ToolDefinition

```go
type ToolDefinition struct {
    Name         string         // 工具名称
    Description  string         // 工具描述
    InputSchema  map[string]any // 输入 JSON Schema
    OutputSchema map[string]any // 输出 JSON Schema
}
```

---

### 7. Embedder (嵌入器)

**源码位置**: `go/ai/embedder.go:30-38`

Embedder 将内容（文本、图像、音频等）转换为数值向量，用于语义搜索和相似度计算。

#### 核心接口

```go
type Embedder interface {
    // 返回嵌入器名称
    Name() string

    // 嵌入内容
    Embed(ctx context.Context, req *EmbedRequest) (*EmbedResponse, error)

    // 注册到 Registry
    Register(r api.Registry)
}
```

#### 内部实现

```go
type embedder struct {
    core.ActionDef[*EmbedRequest, *EmbedResponse, struct{}]
}
```

#### Embedder 能力

```go
type EmbedderSupports struct {
    // 支持的输入类型（"text", "image", "video"等）
    Input []string

    // 是否支持多语言
    Multilingual bool
}
```

#### 定义 Embedder

```go
embedder := ai.DefineEmbedder(
    registry,
    "google/text-embedding-004",
    &ai.EmbedderOptions{
        Label: "Text Embedding 004",
        Dimensions: 768,
        Supports: &ai.EmbedderSupports{
            Input:        []string{"text"},
            Multilingual: true,
        },
    },
    func(ctx context.Context, req *ai.EmbedRequest) (*ai.EmbedResponse, error) {
        // 实现嵌入逻辑
        embeddings := performEmbedding(req.Input)
        return &ai.EmbedResponse{Embeddings: embeddings}, nil
    },
)
```

#### EmbedRequest 和 EmbedResponse

```go
type EmbedRequest struct {
    Input   []*Document // 要嵌入的文档列表
    Options any         // 嵌入器特定配置
}

type EmbedResponse struct {
    Embeddings []*Embedding // 生成的嵌入向量
}

type Embedding struct {
    Embedding []float32      // 向量值
    Metadata  map[string]any // 元数据
}
```

#### 使用 Embedder

```go
// 嵌入单个文档
response, err := ai.Embed(ctx, registry,
    ai.WithEmbedder("text-embedding-004"),
    ai.WithDocuments(ai.NewTextDocument("要嵌入的文本")),
)

// 嵌入多个文档
docs := []*ai.Document{
    ai.NewTextDocument("文本1"),
    ai.NewTextDocument("文本2"),
    ai.NewTextDocument("文本3"),
}
response, err := ai.Embed(ctx, registry,
    ai.WithEmbedder("text-embedding-004"),
    ai.WithDocuments(docs...),
)

// 访问嵌入向量
for i, embedding := range response.Embeddings {
    vector := embedding.Embedding // []float32
    fmt.Printf("文档 %d 的向量: %v\n", i, vector)
}
```

---

### 8. Retriever (检索器)

**源码位置**: `go/ai/retriever.go:31-39`

Retriever 从索引（通常是向量数据库）中检索文档，是 RAG（检索增强生成）的核心组件。

#### 核心接口

```go
type Retriever interface {
    // 返回检索器名称
    Name() string

    // 检索文档
    Retrieve(ctx context.Context, req *RetrieverRequest) (*RetrieverResponse, error)

    // 注册到 Registry
    Register(r api.Registry)
}
```

#### 内部实现

```go
type retriever struct {
    core.ActionDef[*RetrieverRequest, *RetrieverResponse, struct{}]
}
```

#### Retriever 能力

```go
type RetrieverSupports struct {
    // 是否支持多媒体内容
    Media bool
}
```

#### 定义 Retriever

```go
retriever := ai.DefineRetriever(
    registry,
    "myVectorStore",
    &ai.RetrieverOptions{
        Label: "My Vector Store",
        Supports: &ai.RetrieverSupports{
            Media: false,
        },
        ConfigSchema: map[string]any{
            "properties": map[string]any{
                "k": map[string]any{
                    "type":        "number",
                    "description": "返回文档数量",
                },
            },
        },
    },
    func(ctx context.Context, req *ai.RetrieverRequest) (*ai.RetrieverResponse, error) {
        // 实现检索逻辑
        // 1. 可能需要先嵌入查询
        // 2. 在向量数据库中搜索
        // 3. 返回最相关的文档

        documents := searchVectorDB(req.Query, req.Options)
        return &ai.RetrieverResponse{Documents: documents}, nil
    },
)
```

#### RetrieverRequest 和 RetrieverResponse

```go
type RetrieverRequest struct {
    Query   *Document // 查询文档
    Options any       // 检索器特定配置（如 k, threshold 等）
}

type RetrieverResponse struct {
    Documents []*Document // 检索到的文档
}
```

#### 使用 Retriever

```go
// 基本检索
response, err := ai.Retrieve(ctx, registry,
    ai.WithRetriever("myVectorStore"),
    ai.WithDocuments(ai.NewTextDocument("查询文本")),
)

// 带配置的检索
response, err := ai.Retrieve(ctx, registry,
    ai.WithRetriever(ai.NewRetrieverRef("myVectorStore", map[string]any{
        "k": 5,  // 返回 top 5 结果
    })),
    ai.WithDocuments(query),
)

// 在 RAG 中使用
retrievedDocs := response.Documents
modelResponse, err := ai.Generate(ctx, registry,
    ai.WithModel("gemini-1.5-flash"),
    ai.WithDocs(retrievedDocs...),  // 注入检索的文档
    ai.WithMessages(ai.NewUserMessage(ai.NewTextPart("基于上下文回答问题"))),
)
```

---

### 9. Prompt (提示词)

**源码位置**: `go/ai/prompt.go:36-44`

Prompt 是可执行和可渲染的提示词模板，集成了变量替换、模型配置和工具定义。

#### 核心接口

```go
type Prompt interface {
    // 返回提示词名称
    Name() string

    // 执行提示词（渲染 + 调用模型）
    Execute(ctx context.Context, opts ...PromptExecuteOption) (*ModelResponse, error)

    // 渲染提示词为 GenerateActionOptions
    Render(ctx context.Context, input any) (*GenerateActionOptions, error)
}
```

#### 内部实现

```go
type prompt struct {
    core.ActionDef[any, *GenerateActionOptions, struct{}]
    promptOptions
    registry api.Registry
}
```

#### 定义 Prompt

```go
// 使用代码定义
prompt := ai.DefinePrompt(
    registry,
    "summarize",
    ai.WithPromptModel("gemini-1.5-flash"),
    ai.WithPromptDescription("总结文本"),
    ai.WithPromptInputSchema(map[string]any{
        "properties": map[string]any{
            "text": map[string]any{"type": "string"},
        },
        "required": []string{"text"},
    }),
    ai.WithPromptTemplate("请总结以下文本:\n\n{{text}}"),
)

// 使用 Dotprompt 文件
// prompts/summarize.prompt:
// ---
// model: gemini-1.5-flash
// input:
//   schema:
//     text: string
// ---
// 请总结以下文本:
//
// {{text}}

prompt := ai.LoadPrompt(registry, "summarize")
```

#### 执行 Prompt

```go
// 基本执行
response, err := prompt.Execute(ctx,
    ai.WithPromptInput(map[string]any{
        "text": "要总结的长文本...",
    }),
)

// 覆盖模型和配置
response, err := prompt.Execute(ctx,
    ai.WithPromptInput(input),
    ai.WithPromptModel("gemini-1.5-pro"),  // 覆盖默认模型
    ai.WithPromptConfig(map[string]any{
        "temperature": 0.7,
    }),
)
```

#### 渲染 Prompt

```go
// 仅渲染，不执行
opts, err := prompt.Render(ctx, map[string]any{
    "text": "输入文本",
})

// opts 包含渲染后的消息、模型、工具等
// 可以手动调用 Generate
response, err := ai.GenerateWithRequest(ctx, registry, opts, nil, nil)
```

#### Dotprompt 集成

Genkit Go 集成了 Dotprompt 模板引擎，支持：

- **变量替换**：`{{variable}}`
- **条件语句**：`{{#if condition}}...{{/if}}`
- **循环**：`{{#each items}}...{{/each}}`
- **Partial**：`{{> partialName}}`
- **Helper 函数**：`{{helper arg1 arg2}}`

注册 partial 和 helper：

```go
registry.RegisterPartial("header", "## {{title}}\n\n")
registry.RegisterHelper("uppercase", strings.ToUpper)
```

---

### 10. Evaluator (评估器)

**源码位置**: `go/ai/evaluator.go`

Evaluator 用于评估 AI 模型输出的质量。

#### 核心接口

```go
type Evaluator interface {
    // 返回评估器名称
    Name() string

    // 评估单个数据点
    Evaluate(ctx context.Context, req *EvalRequest) (*EvalResponse, error)

    // 注册到 Registry
    Register(r api.Registry)
}
```

#### 评估函数类型

```go
type EvaluatorFunc = func(context.Context, *EvalRequest) (EvalResponse, error)
```

#### 定义 Evaluator

```go
evaluator := ai.DefineEvaluator(
    registry,
    "faithfulness",
    &ai.EvaluatorOptions{
        Label:       "Faithfulness Evaluator",
        Description: "评估响应是否忠实于提供的上下文",
    },
    func(ctx context.Context, req *ai.EvalRequest) (ai.EvalResponse, error) {
        // 实现评估逻辑
        var results []any
        for _, dataPoint := range req.Dataset {
            score := evaluateFaithfulness(
                dataPoint.Input,
                dataPoint.Output,
                dataPoint.Context,
            )
            results = append(results, map[string]any{
                "score":  score,
                "passed": score > 0.8,
            })
        }
        return results, nil
    },
)
```

#### EvalRequest 和 EvalResponse

```go
type EvalRequest struct {
    Dataset   []*BaseDataPoint // 评估数据集
    EvalRunID string           // 评估运行 ID
    Options   any              // 评估器特定配置
}

type BaseDataPoint struct {
    Context    map[string]any // 上下文数据
    Input      map[string]any // 输入
    Output     map[string]any // 模型输出
    Reference  map[string]any // 参考答案
    TestCaseID string         // 测试用例 ID
    TraceIDs   []string       // 相关 trace IDs
}

type EvalResponse []any // 评估结果列表
```

#### 使用 Evaluator

```go
// 准备评估数据
dataset := []*ai.BaseDataPoint{
    {
        Input:     map[string]any{"question": "什么是 AI？"},
        Output:    map[string]any{"answer": "AI 是人工智能..."},
        Reference: map[string]any{"answer": "人工智能是..."},
    },
    // 更多数据点...
}

// 运行评估
response, err := evaluator.Evaluate(ctx, &ai.EvalRequest{
    Dataset:   dataset,
    EvalRunID: "run-123",
})

// 分析结果
for i, result := range response {
    resultMap := result.(map[string]any)
    fmt.Printf("数据点 %d: 得分 %.2f, 通过: %v\n",
        i, resultMap["score"], resultMap["passed"])
}
```

---

## 三、实体协同工作方式

### 整体架构图

```
┌─────────────────────────────────────────────────────────────┐
│                    Genkit Instance                           │
│                 (应用入口 - 门面层)                           │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                     Registry                          │  │
│  │              (中央注册表 - 管理所有组件)                │  │
│  └─────────────────────────┬─────────────────────────────┘  │
└────────────────────────────┼────────────────────────────────┘
                             │ 注册/查找
             ┌───────────────┼───────────────┐
             │               │               │
          Plugin         ActionDef          Flow
          (插件)        (基础抽象)        (工作流)
             │               │               │
             │               └───────┬───────┘
             │                       │ 继承/包装
             ├───────────────────────┼───────────────────────┐
             │                       │                       │
          Model                   Tool                  Embedder
          (模型)                  (工具)                (嵌入器)
             │                       │                       │
             │                       │                       │
          Retriever              Prompt              Evaluator
          (检索器)              (提示词)              (评估器)
```

### 1. Action-Based 统一抽象

所有组件都基于或包装 `ActionDef`，获得统一的能力：

```
ActionDef (核心抽象)
    │
    ├─> Flow (类型别名)
    │   └─> type Flow[In, Out, Stream] ActionDef[In, Out, Stream]
    │
    ├─> Model (包装)
    │   └─> type model struct { ActionDef[*ModelRequest, *ModelResponse, *ModelResponseChunk] }
    │
    ├─> Tool (包装)
    │   └─> type tool struct { api.Action }
    │
    ├─> Embedder (包装)
    │   └─> type embedder struct { ActionDef[*EmbedRequest, *EmbedResponse, struct{}] }
    │
    ├─> Retriever (包装)
    │   └─> type retriever struct { ActionDef[*RetrieverRequest, *RetrieverResponse, struct{}] }
    │
    └─> Prompt (包装)
        └─> type prompt struct { ActionDef[any, *GenerateActionOptions, struct{}] }
```

**统一获得的能力：**
- 执行接口（Run, RunJSON）
- Tracing（每次执行创建 span）
- Metrics（延迟、成功/失败）
- 输入/输出验证（JSON Schema）
- 注册到 Registry
- Developer UI 集成

### 2. Registry 查找机制

```go
┌─────────────────────────────────────────────────┐
│               Component Lifecycle                │
└─────────────────────────────────────────────────┘

1. 定义并注册
   ├─> DefineFlow(registry, name, fn)
   ├─> DefineModel(registry, name, opts, fn)
   ├─> DefineTool(registry, name, desc, fn)
   └─> DefineEmbedder(registry, name, opts, fn)
        │
        └─> registry.RegisterAction(key, action)

2. 查找使用
   ├─> LookupModel(registry, name)
   ├─> LookupTool(registry, name)
   └─> LookupRetriever(registry, name)
        │
        └─> registry.LookupAction(key)

3. 动态解析 (通过 DynamicPlugin)
   └─> registry.ResolveAction(key)
        │
        ├─> registry.LookupAction(key)  // 先查本地
        │
        └─> plugin.ResolveAction(atype, name)  // 动态创建
```

### 3. Model 中间件链

Model 使用中间件模式处理请求：

```
ModelRequest
    │
    ▼
┌─────────────────────────┐
│ simulateSystemPrompt    │  模拟系统提示词（不支持时）
└───────────┬─────────────┘
            ▼
┌─────────────────────────┐
│ augmentWithContext      │  将 Docs 注入到 Messages
└───────────┬─────────────┘
            ▼
┌─────────────────────────┐
│ validateSupport         │  验证请求与模型能力匹配
└───────────┬─────────────┘
            ▼
┌─────────────────────────┐
│ addAutomaticTelemetry   │  添加自动遥测
└───────────┬─────────────┘
            ▼
┌─────────────────────────┐
│ 用户定义的 ModelFunc     │  实际模型调用
└───────────┬─────────────┘
            ▼
     ModelResponse
```

### 4. Flow 执行流程

```
用户调用 Flow.Run(input)
    │
    ├─> 创建 flowContext
    │   └─> ctx = flowContextKey.NewContext(ctx, &flowContext{flowName})
    │
    ├─> tracing.RunInNewSpan(ctx, spanMetadata, input, fn)
    │   │
    │   ├─> 创建 trace span (type: "action", subtype: "flow")
    │   │
    │   ├─> 验证输入 (ValidateValue)
    │   │
    │   ├─> 执行用户函数
    │   │   │
    │   │   ├─> 可能调用 Model.Generate()
    │   │   │   └─> 创建子 span (type: "action", subtype: "model")
    │   │   │
    │   │   ├─> 可能调用 Tool.RunRaw()
    │   │   │   └─> 创建子 span (type: "action", subtype: "tool")
    │   │   │
    │   │   ├─> 可能调用 Retriever.Retrieve()
    │   │   │   └─> 创建子 span (type: "action", subtype: "retriever")
    │   │   │
    │   │   ├─> 可能调用 core.Run() 创建步骤
    │   │   │   └─> 创建子 span (type: "flowStep")
    │   │   │
    │   │   └─> 返回输出
    │   │
    │   ├─> 验证输出 (ValidateValue)
    │   │
    │   └─> 记录 metrics (latency, success/failure)
    │
    └─> 返回 ActionRunResult{Result, TraceId, SpanId}
```

### 5. Model-Tool 交互流程

```
1. 定义工具
   └─> tool := DefineTool(registry, "search", desc, fn)

2. 调用 Generate 并传入工具
   └─> Generate(ctx, registry,
           WithModel("gemini-1.5-flash"),
           WithTools(tool),
           WithMessages(...))

3. Generate 内部处理
   ├─> 构建 ModelRequest
   │   └─> ModelRequest.Tools = []*ToolDefinition{tool.Definition()}
   │
   ├─> 调用 Model.Generate(req)
   │
   ├─> 检查 ModelResponse.Message
   │   │
   │   ├─> 如果包含 ToolRequest parts:
   │   │   │
   │   │   ├─> 查找并执行工具
   │   │   │   └─> tool.RunRaw(ctx, toolRequest.Input)
   │   │   │
   │   │   ├─> 构建 ToolResponse
   │   │   │   └─> NewResponseForToolRequest(toolReq, output)
   │   │   │
   │   │   ├─> 将 ToolRequest 和 ToolResponse 添加到消息历史
   │   │   │
   │   │   └─> 递归调用 Model.Generate() (新消息历史)
   │   │
   │   └─> 如果 FinishReason = "stop":
   │       └─> 返回最终响应
   │
   └─> 最多重复 MaxTurns 次

工具中断场景:
   ├─> 工具调用 ctx.Interrupt(opts)
   │
   ├─> 返回 FinishReason = "interrupted"
   │
   ├─> 用户处理中断
   │
   └─> 选项 1: Respond
       └─> Generate(..., WithToolResponses(tool.Respond(part, result)))

   └─> 选项 2: Restart
       └─> Generate(..., WithToolRestarts(tool.Restart(part, opts)))
```

### 6. RAG (检索增强生成) 流程

```
┌─────────────────────────────────────────────────┐
│              RAG Pipeline                        │
└─────────────────────────────────────────────────┘

1. 用户查询
   └─> query := "什么是 Genkit?"

2. 可选: 嵌入查询 (如果需要向量搜索)
   └─> embedResp := Embed(ctx, registry,
           WithEmbedder("text-embedding-004"),
           WithDocuments(NewTextDocument(query)))
       └─> queryVector := embedResp.Embeddings[0].Embedding

3. 检索相关文档
   └─> retrieveResp := Retrieve(ctx, registry,
           WithRetriever("myVectorStore"),
           WithDocuments(NewTextDocument(query)))
       └─> relevantDocs := retrieveResp.Documents

4. 使用检索的文档增强生成
   └─> response := Generate(ctx, registry,
           WithModel("gemini-1.5-flash"),
           WithDocs(relevantDocs...),  // 注入上下文
           WithMessages(NewUserMessage(NewTextPart(query))))

5. Model 内部处理 (通过 augmentWithContext 中间件)
   ├─> 将 Docs 格式化为文本
   │   └─> "\n\nUse the following information:\n\n"
   │       + "Document 1: ..."
   │       + "Document 2: ..."
   │
   ├─> 将格式化的文本插入到最后一条用户消息
   │
   └─> 调用模型生成响应

6. 返回增强的响应
   └─> response.Message.Content[0].Text
```

### 7. Plugin 扩展机制

```
┌─────────────────────────────────────────────────┐
│            Plugin Initialization                 │
└─────────────────────────────────────────────────┘

1. 定义插件
   type MyPlugin struct {
       config Config
   }

2. 实现 Plugin 接口
   func (p *MyPlugin) Init(ctx context.Context) []Action {
       return []Action{
           DefineModel(registry, "myplugin/model1", ...),
           DefineModel(registry, "myplugin/model2", ...),
           DefineEmbedder(registry, "myplugin/embedder", ...),
       }
   }

3. 注册插件
   └─> registry.RegisterPlugin("myplugin", &MyPlugin{...})

4. 初始化 (通常在应用启动时)
   └─> for _, plugin := range registry.ListPlugins() {
           actions := plugin.Init(ctx)
           for _, action := range actions {
               // 已经在 Define* 中注册
           }
       }

动态插件:
   ├─> 实现 DynamicPlugin 接口
   │   ├─> ListActions(ctx) []ActionDesc
   │   └─> ResolveAction(atype, name) Action
   │
   ├─> 用户请求不存在的 action
   │   └─> registry.ResolveAction(key)
   │
   ├─> Registry 检查所有 DynamicPlugin
   │   └─> plugin.ResolveAction(atype, name)
   │
   └─> 动态创建并返回 Action
```

### 8. Tracing 和可观察性

每个 Action 执行都会自动记录：

```
tracing.RunInNewSpan(ctx, spanMetadata, input, fn)
    │
    ├─> 创建 span
    │   ├─> TraceID (如果是根 span 则生成新的)
    │   ├─> SpanID (唯一标识此 span)
    │   ├─> ParentSpanID (如果有父 span)
    │   ├─> Name (action 名称)
    │   ├─> Type ("action", "flowStep" 等)
    │   ├─> Subtype (action 类型: "model", "tool" 等)
    │   └─> Metadata (自动注入 flow name 等)
    │
    ├─> 记录输入
    │   └─> logger.Debug("Action.Run", "input", inputJSON)
    │
    ├─> 执行函数
    │   └─> start := time.Now()
    │       output, err := fn(ctx, input)
    │       latency := time.Since(start)
    │
    ├─> 记录 metrics
    │   ├─> 成功: metrics.WriteActionSuccess(ctx, name, latency)
    │   └─> 失败: metrics.WriteActionFailure(ctx, name, latency, err)
    │
    ├─> 记录输出
    │   └─> logger.Debug("Action.Run", "output", outputJSON, "err", err)
    │
    └─> 返回 ActionRunResult{Result, TraceID, SpanID}

Trace 层级示例:
flow:myFlow [trace-123, span-1]
    ├─> action:model [trace-123, span-2, parent-1]
    │   └─> action:tool [trace-123, span-3, parent-2]
    ├─> action:retriever [trace-123, span-4, parent-1]
    └─> flowStep:processResults [trace-123, span-5, parent-1]
```

### 9. 完整示例：构建 RAG 应用

```go
package main

import (
    "context"
    "fmt"

    "github.com/firebase/genkit/go/ai"
    "github.com/firebase/genkit/go/core"
    "github.com/firebase/genkit/go/genkit"
)

func main() {
    ctx := context.Background()

    // 1. 初始化 Genkit 和插件
    registry, err := genkit.Init(ctx,
        genkit.WithPlugins(
            vertexai.NewPlugin(),      // 提供 models 和 embedder
            pinecone.NewPlugin(),       // 提供 retriever
        ),
    )
    if err != nil {
        panic(err)
    }

    // 2. 定义 RAG Flow
    ragFlow := core.DefineFlow[RAGInput, RAGOutput](
        registry,
        "ragFlow",
        func(ctx context.Context, input RAGInput) (RAGOutput, error) {
            // 步骤 1: 检索相关文档
            retrieveResp, err := core.Run[*ai.RetrieverResponse](ctx, "retrieve", func() (*ai.RetrieverResponse, error) {
                return ai.Retrieve(ctx, registry,
                    ai.WithRetriever("pinecone"),
                    ai.WithDocuments(ai.NewTextDocument(input.Query)),
                )
            })
            if err != nil {
                return RAGOutput{}, err
            }

            // 步骤 2: 生成响应
            genResp, err := core.Run[*ai.ModelResponse](ctx, "generate", func() (*ai.ModelResponse, error) {
                return ai.Generate(ctx, registry,
                    ai.WithModel("gemini-1.5-flash"),
                    ai.WithDocs(retrieveResp.Documents...),
                    ai.WithMessages(
                        ai.NewUserMessage(ai.NewTextPart(input.Query)),
                    ),
                )
            })
            if err != nil {
                return RAGOutput{}, err
            }

            return RAGOutput{
                Answer:    genResp.Message.Content[0].Text,
                Sources:   retrieveResp.Documents,
                TraceID:   genResp.Request.TraceId,
            }, nil
        },
    )

    // 3. 运行 Flow
    result, err := ragFlow.Run(ctx, RAGInput{
        Query: "什么是 Genkit?",
    })
    if err != nil {
        panic(err)
    }

    fmt.Printf("答案: %s\n", result.Answer)
    fmt.Printf("来源: %d 个文档\n", len(result.Sources))
    fmt.Printf("TraceID: %s\n", result.TraceID)
}

type RAGInput struct {
    Query string
}

type RAGOutput struct {
    Answer  string
    Sources []*ai.Document
    TraceID string
}
```

**执行流程：**

```
ragFlow.Run(input)
    │
    ├─> 创建 flow span [trace-1, span-1]
    │
    ├─> core.Run("retrieve")
    │   ├─> 创建 flowStep span [trace-1, span-2, parent-1]
    │   ├─> ai.Retrieve()
    │   │   └─> retriever.Retrieve()
    │   │       └─> 创建 action span [trace-1, span-3, parent-2]
    │   └─> 返回并缓存结果
    │
    ├─> core.Run("generate")
    │   ├─> 创建 flowStep span [trace-1, span-4, parent-1]
    │   ├─> ai.Generate()
    │   │   ├─> 中间件处理
    │   │   │   └─> augmentWithContext: 将 docs 注入消息
    │   │   ├─> model.Generate()
    │   │   │   └─> 创建 action span [trace-1, span-5, parent-4]
    │   │   └─> 返回响应
    │   └─> 返回并缓存结果
    │
    └─> 返回 RAGOutput
```

---

## 四、关键设计原则

### 1. 组合优于继承

Genkit Go 不使用传统的继承层级，而是通过组合 `ActionDef` 获得能力：

```go
// Flow 是 ActionDef 的类型别名
type Flow[In, Out, Stream any] ActionDef[In, Out, Stream]

// Model 包含 ActionDef
type model struct {
    core.ActionDef[*ModelRequest, *ModelResponse, *ModelResponseChunk]
}

// Tool 包含 Action 接口
type tool struct {
    api.Action
}
```

**优点：**
- 代码重用
- 灵活组合
- 避免深层继承
- 符合 Go 的设计哲学

### 2. 类型安全

使用 Go 泛型提供编译时类型检查：

```go
// Flow 的输入输出类型在编译时确定
flow := core.DefineFlow[MyInput, MyOutput](...)

// 编译时类型检查
input := MyInput{...}
output, err := flow.Run(ctx, input)  // output 类型是 MyOutput

// 错误：类型不匹配
wrongInput := OtherInput{...}
flow.Run(ctx, wrongInput)  // 编译错误
```

### 3. 统一接口

所有组件遵循相同的模式：

```go
// 定义模式
Define*(registry, name, options, func)

// 查找模式
Lookup*(registry, name)

// 执行模式
component.Run(ctx, input)
component.Execute(ctx, options)
component.Generate(ctx, request)
```

**好处：**
- 学习曲线低
- 代码一致性
- 易于维护

### 4. 可扩展性

通过 Plugin 机制轻松添加功能：

```go
// 实现 Plugin 接口即可扩展
type MyPlugin struct {}

func (p *MyPlugin) Name() string { return "myplugin" }

func (p *MyPlugin) Init(ctx context.Context) []Action {
    // 注册任意数量的 actions
}

// 动态插件支持按需加载
type MyDynamicPlugin struct {
    MyPlugin
}

func (p *MyDynamicPlugin) ResolveAction(atype ActionType, name string) Action {
    // 动态创建 action
}
```

### 5. 可观察性优先

可观察性不是事后添加，而是内置的：

- 每个 Action 自动创建 trace span
- 自动记录输入/输出
- 自动记录 metrics（延迟、成功/失败）
- 自动注入上下文（如 flow name）
- Developer UI 开箱即用

### 6. Provider 无关

核心框架不依赖特定提供商：

```go
// 可以轻松切换模型提供商
ai.Generate(ctx, registry,
    ai.WithModel("gemini-1.5-flash"),      // Google
    // ai.WithModel("gpt-4"),              // OpenAI
    // ai.WithModel("claude-3-opus"),      // Anthropic
    ai.WithMessages(...),
)
```

### 7. 流式支持

所有 Action 都可以支持流式输出：

```go
// 定义支持流式的 action
action := core.NewStreamingAction[In, Out, Stream](
    name, atype, metadata, schema,
    func(ctx context.Context, input In, cb func(context.Context, Stream) error) (Out, error) {
        // 可以通过 cb 发送流式数据
        for chunk := range generateChunks(input) {
            cb(ctx, chunk)
        }
        return finalOutput, nil
    },
)

// 流式调用
action.Run(ctx, input, func(ctx context.Context, chunk Stream) error {
    // 处理每个流式块
    return nil
})
```

### 8. Production-Ready

框架内置生产环境所需功能：

- **错误处理**：统一的错误类型和处理
- **验证**：自动 JSON Schema 验证
- **重试**：可配置的重试逻辑
- **超时**：Context-based 超时控制
- **并发**：线程安全的 Registry
- **监控**：Metrics 和 Tracing
- **部署**：Flow 可轻松部署为 HTTP 端点

### 9. 开发者体验

- **Developer UI**：交互式运行和调试
- **类型提示**：完整的 IDE 支持
- **文档生成**：从代码自动生成文档
- **热重载**：开发时自动重载
- **调试工具**：详细的 trace 和 log

---

## 总结

Genkit Go 通过以下设计实现了一个强大而灵活的 AI 应用开发框架：

1. **ActionDef 统一抽象**：所有可观察操作的基础
2. **Registry 集中管理**：统一的组件注册和查找
3. **Plugin 可扩展架构**：轻松集成第三方服务
4. **AI 实体丰富**：Model、Tool、Embedder、Retriever、Prompt、Evaluator
5. **协同工作流畅**：通过标准接口和中间件模式无缝集成
6. **可观察性内置**：自动 tracing、metrics 和 logging
7. **类型安全**：Go 泛型提供编译时保证
8. **Production-ready**：内置所有生产环境所需功能

这种设计使得开发者可以快速构建复杂的 AI 应用，同时保持代码的可维护性和可扩展性。
