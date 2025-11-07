# LangChain Python 1.0 架构分析

## 概述

LangChain 1.0 是 LangChain 框架的首个稳定主版本（2025年10月发布），标志着从"原型游乐场"到"专业工程框架"的重大转变。与 0.x 版本相比，1.0 版本进行了彻底的架构重构，将核心抽象分离到独立包中，并将重点从 LCEL 链式编排转向基于 LangGraph 的 Agent 架构。

**项目信息**
- 官方仓库：https://github.com/langchain-ai/langchain
- 版本：1.0.0+ (2025年10月17日发布)
- 编程语言：Python (>= 3.10)
- 核心特性：Agent 优先、LangGraph 编排、生产级持久化、中间件架构

## 核心概念实体

### 1. **Runnable** - 统一执行接口

**定义**：Runnable 是 LangChain 所有组件的基础抽象，提供统一的执行接口。

**核心方法**：
- `invoke(input, **opts) -> output` - 同步单次调用
- `stream(input, **opts) -> StreamReader[output]` - 流式输出
- `batch(inputs, **opts) -> List[output]` - 批量处理
- `ainvoke/astream/abatch` - 异步版本

**特点**：
- 所有 LangChain 组件（Model、Tool、Chain、Agent）都实现 Runnable
- 支持 4 种执行模式：单次、流式、批量、异步
- 提供统一的组合模式

**示例**：
```python
from langchain_core.runnables import Runnable

# 任何 Runnable 都支持这些方法
result = runnable.invoke(input_data)
stream = runnable.stream(input_data)
batch_results = runnable.batch([input1, input2, input3])
```

---

### 2. **RunnableSequence** - 顺序编排

**定义**：RunnableSequence 将多个 Runnable 顺序连接，前一个的输出作为下一个的输入。

**构造方式**：
1. 使用 `|` 操作符（LCEL 语法，0.x 中常用，1.0 中逐步淡化）
2. 显式创建：`RunnableSequence(first=r1, middle=[r2, r3], last=r4)`

**特点**：
- 自动传递上下文和输出
- 支持流式和批量处理
- 可以包含任意 Runnable

**示例**：
```python
# LCEL 风格（0.x）
chain = prompt | model | output_parser

# 1.0 推荐：使用 LangGraph 或 create_agent
```

---

### 3. **RunnableParallel** - 并行编排

**定义**：RunnableParallel 并发执行多个 Runnable，将相同输入传递给每个分支。

**构造方式**：
1. 使用字典字面量
2. 显式创建：`RunnableParallel({"branch1": r1, "branch2": r2})`

**输出格式**：字典，键为分支名，值为对应 Runnable 的输出

**特点**：
- 真正的并行执行（非串行）
- 支持流式合并
- 常用于检索增强生成（RAG）场景

**示例**：
```python
parallel = RunnableParallel({
    "context": retriever,
    "question": RunnableLambda(lambda x: x)
})
# 输出: {"context": [...docs...], "question": "user query"}
```

---

### 4. **RunnableLambda** - 函数包装器

**定义**：将 Python 函数转换为 Runnable 对象，实现自定义逻辑。

**用途**：
- 数据预处理和后处理
- 条件路由逻辑
- 自定义转换函数

**特点**：
- 支持同步和异步函数
- 可以接收和返回任意类型
- 1.0 中推荐用于条件路由（替代 RunnableBranch）

**示例**：
```python
from langchain_core.runnables import RunnableLambda

def uppercase_text(x: str) -> str:
    return x.upper()

runnable = RunnableLambda(uppercase_text)
result = runnable.invoke("hello")  # "HELLO"
```

---

### 5. **RunnableBranch** - 条件路由（已过时）

**定义**：根据条件选择执行分支的 Runnable。

**状态**：在 1.0 中被标记为 **legacy**，推荐使用 RunnableLambda 实现路由。

**构造**：
```python
branch = RunnableBranch(
    (condition1, runnable1),
    (condition2, runnable2),
    default_runnable
)
```

**替代方案**：
```python
def route_logic(input):
    if condition1(input):
        return runnable1.invoke(input)
    elif condition2(input):
        return runnable2.invoke(input)
    else:
        return default_runnable.invoke(input)

router = RunnableLambda(route_logic)
```

---

### 6. **Message** - 消息抽象

**定义**：LangChain 中的消息类型，代表对话中的不同角色和内容。

**消息类型**：

#### 6.1 HumanMessage
- 代表用户输入
- 支持多模态内容（文本、图片、音频、文件）
- 通常是对话历史中的第一条消息

#### 6.2 AIMessage
- 代表模型的输出
- 包含生成的文本和工具调用
- 特殊属性：`tool_calls`（工具调用列表）

#### 6.3 SystemMessage
- 包含系统级指令
- 几乎总是对话历史中的第一条消息
- 用于设置 AI 行为和角色

#### 6.4 ToolMessage
- 工具执行结果
- 必须包含 `tool_call_id`（与 AIMessage 中的工具调用 ID 对应）
- 包含工具名称和输出内容

#### 6.5 FunctionMessage
- 处理动态函数调用（较少使用）
- 与 ToolMessage 功能类似

**示例**：
```python
from langchain_core.messages import (
    HumanMessage, AIMessage, SystemMessage, ToolMessage
)

messages = [
    SystemMessage(content="You are a helpful assistant"),
    HumanMessage(content="What's the weather?"),
    AIMessage(content="", tool_calls=[
        {"id": "call_1", "name": "get_weather", "args": {"city": "SF"}}
    ]),
    ToolMessage(content="72°F, sunny", tool_call_id="call_1")
]
```

---

### 7. **ChatModel** - 语言模型抽象

**定义**：ChatModel 是与聊天式 LLM 交互的统一接口。

**核心方法**：
- `generate(messages, **opts) -> AIMessage` - 生成单个响应
- `stream(messages, **opts) -> StreamReader[AIMessage]` - 流式生成

**工具调用支持**：
```python
model = ChatOpenAI(model="gpt-4")

# 绑定工具
model_with_tools = model.bind_tools([weather_tool, calculator_tool])

# 调用
response = model_with_tools.invoke([HumanMessage(content="What's 5+3?")])
# response.tool_calls = [{"name": "calculator", "args": {"expression": "5+3"}}]
```

**特点**：
- 支持多模态输入（文本、图片等）
- 工具调用是一等公民（`bind_tools`）
- 流式输出支持 token-by-token 或 chunk-based

---

### 8. **Tool** - 工具抽象

**定义**：Tool 是可被 LLM 调用的函数或 API。

**定义方式**：
1. 使用 `@tool` 装饰器
2. 继承 `BaseTool` 类
3. 传递 Pydantic 模型

**工具信息**：
```python
from langchain_core.tools import tool

@tool
def get_weather(city: str) -> str:
    """Get the current weather for a city."""
    return f"Weather in {city}: 72°F, sunny"

# 工具信息包括：
# - name: "get_weather"
# - description: "Get the current weather for a city."
# - args_schema: {"city": {"type": "string"}}
```

**绑定到模型**：
```python
model_with_tools = model.bind_tools([get_weather])
```

---

### 9. **Agent (create_agent)** - 高级 Agent 抽象

**定义**：Agent 是 LangChain 1.0 的核心抽象，用于构建具备工具调用能力的智能代理。

**1.0 的重大变化**：
- 从 0.x 的复杂 Chain 构造转向简单的 `create_agent` API
- 底层基于 LangGraph 构建，而非 LCEL 链
- 内置支持中间件、持久化、人机协作

**基本用法**：
```python
from langchain.agents import create_agent
from langchain_openai import ChatOpenAI

model = ChatOpenAI(model="gpt-4o")
tools = [weather_tool, calculator_tool]

agent = create_agent(
    model=model,
    tools=tools,
    system_prompt="You are a helpful assistant"
)

result = agent.invoke({"messages": [HumanMessage(content="What's 5+3?")]})
```

**特点**：
- 自动处理工具调用循环（ReAct 模式）
- 支持 checkpointing（会话持久化）
- 支持流式输出
- 支持人机协作（human-in-the-loop）

---

### 10. **StateGraph** - 状态图编排（LangGraph）

**定义**：StateGraph 是 LangGraph 的核心类，用于构建有状态的图形化工作流。

**核心概念**：
- **State**：图中节点共享的状态对象（必须是 TypedDict）
- **Node**：处理步骤（Python 函数）
- **Edge**：节点间的转换关系
- **Conditional Edge**：根据状态动态选择下一个节点

**基本结构**：
```python
from langgraph.graph import StateGraph, START, END
from typing import TypedDict

class State(TypedDict):
    messages: list
    user_input: str
    final_answer: str

# 创建图
graph = StateGraph(State)

# 添加节点
graph.add_node("process", process_function)
graph.add_node("respond", respond_function)

# 添加边
graph.add_edge(START, "process")
graph.add_edge("process", "respond")
graph.add_edge("respond", END)

# 编译
app = graph.compile()

# 执行
result = app.invoke({"user_input": "Hello"})
```

**特点**：
- 支持循环和复杂控制流
- 内置 checkpointing（每个节点执行后保存状态）
- 支持多线程并发（`thread_id`）

---

### 11. **Node** - 图节点

**定义**：Node 是 StateGraph 中的处理单元，执行特定逻辑并更新状态。

**函数签名**：
```python
def node_function(state: State) -> dict:
    # 处理逻辑
    return {"key": "updated_value"}  # 返回状态更新
```

**特点**：
- 接收完整状态对象
- 返回字典形式的状态更新（类似 reducer）
- 可以访问 LangGraph 上下文（如 `thread_id`）

**示例**：
```python
def analyze_node(state: State) -> dict:
    user_msg = state["messages"][-1]
    # 分析逻辑
    return {"analysis": "sentiment: positive"}
```

---

### 12. **Edge** - 图边

**定义**：Edge 定义节点之间的转换关系。

**类型**：

#### 12.1 普通边（Normal Edge）
固定的转换关系：
```python
graph.add_edge("node_a", "node_b")  # node_a 完成后总是执行 node_b
```

#### 12.2 条件边（Conditional Edge）
根据状态动态选择下一个节点：
```python
def route_function(state: State) -> str:
    if state["needs_tool"]:
        return "tool_node"
    else:
        return "respond_node"

graph.add_conditional_edges(
    "analyze",
    route_function,
    {
        "tool_node": "execute_tool",
        "respond_node": "final_response"
    }
)
```

**特点**：
- 条件边使图支持循环和分支
- 可以返回 `END` 终止执行
- 支持多路径映射

---

### 13. **MessagesState** - 预构建消息状态

**定义**：LangGraph 提供的预构建状态类型，专门用于聊天应用。

**定义**：
```python
from langgraph.graph import MessagesState

# MessagesState 等价于：
class MessagesState(TypedDict):
    messages: Annotated[list, add_messages]
```

**特点**：
- 自动处理消息列表的追加（`add_messages` reducer）
- 避免手动合并消息历史
- 简化对话型应用开发

**示例**：
```python
graph = StateGraph(MessagesState)

def chatbot(state: MessagesState):
    response = model.invoke(state["messages"])
    return {"messages": [response]}  # 自动追加到历史
```

---

### 14. **Checkpointer** - 状态持久化

**定义**：Checkpointer 用于保存和恢复图的执行状态。

**内置实现**：
- **MemorySaver**：内存存储（仅用于开发）
- **AsyncSqliteSaver**：SQLite 持久化
- **AsyncPostgresSaver**：PostgreSQL 持久化

**用途**：
- 会话记忆（多轮对话）
- 错误恢复（从失败节点继续）
- 人机协作（暂停等待审批）

**示例**：
```python
from langgraph.checkpoint.memory import MemorySaver

memory = MemorySaver()
app = graph.compile(checkpointer=memory)

# 第一次调用
config = {"configurable": {"thread_id": "user_123"}}
result1 = app.invoke({"messages": [HumanMessage(content="Hi")]}, config)

# 第二次调用（自动加载历史）
result2 = app.invoke({"messages": [HumanMessage(content="What's my name?")]}, config)
```

**核心概念**：
- **thread_id**：会话标识符，用于区分不同用户或对话
- **checkpoint**：每个节点执行后的状态快照

---

### 15. **Document** - 文档对象

**定义**：Document 表示一段文本及其元数据。

**属性**：
```python
from langchain_core.documents import Document

doc = Document(
    page_content="LangChain is a framework for LLM apps",
    metadata={"source": "docs.md", "page": 1}
)
```

**用途**：
- RAG 系统中的核心数据结构
- 由 DocumentLoader 加载
- 经过 TextSplitter 分块
- 存储到 VectorStore

---

### 16. **DocumentLoader** - 文档加载器

**定义**：DocumentLoader 从不同来源加载文档。

**常见实现**：
- `TextLoader` - 加载文本文件
- `PDFLoader` - 加载 PDF
- `WebBaseLoader` - 加载网页
- `WikipediaLoader` - 加载 Wikipedia
- `DirectoryLoader` - 批量加载目录

**示例**：
```python
from langchain_community.document_loaders import TextLoader

loader = TextLoader("data.txt")
documents = loader.load()  # List[Document]
```

---

### 17. **TextSplitter** - 文本分块器

**定义**：TextSplitter 将长文档分割成较小的块，以适应模型上下文窗口。

**常见实现**：
- `CharacterTextSplitter` - 按字符数分割
- `RecursiveCharacterTextSplitter` - 递归分割（推荐）
- `TokenTextSplitter` - 按 token 分割

**示例**：
```python
from langchain_text_splitters import RecursiveCharacterTextSplitter

splitter = RecursiveCharacterTextSplitter(
    chunk_size=1000,
    chunk_overlap=200
)
chunks = splitter.split_documents(documents)
```

---

### 18. **Embeddings** - 嵌入模型

**定义**：Embeddings 将文本转换为向量表示。

**核心方法**：
- `embed_documents(texts: List[str]) -> List[List[float]]` - 批量嵌入文档
- `embed_query(text: str) -> List[float]` - 嵌入查询

**示例**：
```python
from langchain_openai import OpenAIEmbeddings

embeddings = OpenAIEmbeddings(model="text-embedding-3-small")
vectors = embeddings.embed_documents(["text1", "text2"])
query_vector = embeddings.embed_query("search query")
```

---

### 19. **VectorStore** - 向量存储

**定义**：VectorStore 存储文档的向量表示，并支持相似度搜索。

**常见实现**：
- `FAISS` - Facebook AI Similarity Search
- `Chroma` - 开源向量数据库
- `Pinecone` - 云托管向量数据库
- `Weaviate` - 开源向量搜索引擎

**创建方式**：
```python
from langchain_community.vectorstores import FAISS

vectorstore = FAISS.from_documents(
    documents=chunks,
    embedding=embeddings
)
```

**搜索方法**：
- `similarity_search(query, k=4)` - 相似度搜索
- `similarity_search_with_score(query)` - 返回相似度分数
- `max_marginal_relevance_search(query)` - MMR 搜索（多样性）

---

### 20. **Retriever** - 检索器

**定义**：Retriever 是 VectorStore 的检索接口，是 RAG 的标准组件。

**创建方式**：
```python
retriever = vectorstore.as_retriever(
    search_type="similarity",
    search_kwargs={"k": 4}
)
```

**接口**：
```python
docs = retriever.invoke("What is LangChain?")
# 返回: List[Document]
```

**搜索类型**：
- `similarity` - 相似度搜索
- `mmr` - 最大边际相关性
- `similarity_score_threshold` - 相似度阈值过滤

---

### 21. **PromptTemplate** - 提示模板

**定义**：PromptTemplate 用于创建可复用的提示字符串。

**基本用法**：
```python
from langchain_core.prompts import PromptTemplate

prompt = PromptTemplate.from_template(
    "Translate {text} to {language}"
)

formatted = prompt.invoke({"text": "Hello", "language": "Spanish"})
# "Translate Hello to Spanish"
```

**特点**：
- 支持 Python 格式化语法 `{variable}`
- 可以链式组合其他 Runnable
- 用于非聊天模型（completion models）

---

### 22. **ChatPromptTemplate** - 聊天提示模板

**定义**：ChatPromptTemplate 用于创建结构化的聊天提示。

**构造方式**：
```python
from langchain_core.prompts import ChatPromptTemplate

prompt = ChatPromptTemplate.from_messages([
    ("system", "You are a helpful assistant"),
    ("human", "Hello!"),
    ("ai", "Hi there! How can I help?"),
    ("human", "{user_input}")
])

messages = prompt.invoke({"user_input": "Tell me a joke"})
```

**特点**：
- 自动创建对应的 Message 对象
- 支持模板变量
- 支持 MessagesPlaceholder（动态插入消息列表）

---

### 23. **MessagesPlaceholder** - 消息占位符

**定义**：MessagesPlaceholder 用于在提示模板中动态插入消息列表。

**用途**：插入对话历史

**示例**：
```python
from langchain_core.prompts import ChatPromptTemplate, MessagesPlaceholder

prompt = ChatPromptTemplate.from_messages([
    ("system", "You are a helpful assistant"),
    MessagesPlaceholder(variable_name="chat_history"),
    ("human", "{input}")
])

messages = prompt.invoke({
    "chat_history": [
        HumanMessage(content="Hi"),
        AIMessage(content="Hello!")
    ],
    "input": "What's the weather?"
})
```

---

### 24. **OutputParser** - 输出解析器

**定义**：OutputParser 将 LLM 的文本输出解析为结构化数据。

**常见实现**：
- `StrOutputParser` - 提取纯文本
- `JsonOutputParser` - 解析 JSON
- `PydanticOutputParser` - 解析为 Pydantic 模型
- `CommaSeparatedListOutputParser` - 解析逗号分隔列表

**示例**：
```python
from langchain_core.output_parsers import StrOutputParser

parser = StrOutputParser()
output = parser.invoke(ai_message)  # 提取 ai_message.content
```

**与 structured_output 的区别**：
- `OutputParser`：后处理 LLM 输出（通过提示工程）
- `with_structured_output`：使用模型原生结构化输出能力（如 OpenAI function calling）

---

### 25. **BaseCallbackHandler** - 回调处理器

**定义**：BaseCallbackHandler 允许在 LLM 应用的各个阶段注入自定义逻辑。

**关键回调方法**：
- `on_llm_start` - LLM 开始时
- `on_llm_end` - LLM 结束时
- `on_llm_error` - LLM 错误时
- `on_chain_start` - Chain 开始时
- `on_chain_end` - Chain 结束时
- `on_tool_start` - Tool 开始时
- `on_tool_end` - Tool 结束时

**用途**：
- 日志记录
- 性能监控
- 流式输出处理
- 自定义事件触发

**示例**：
```python
from langchain_core.callbacks import BaseCallbackHandler

class LoggingHandler(BaseCallbackHandler):
    def on_llm_end(self, response, **kwargs):
        print(f"LLM response: {response}")

model.invoke(messages, config={"callbacks": [LoggingHandler()]})
```

---

### 26. **Streaming (astream_events)** - 事件流

**定义**：LangChain 1.0 提供统一的事件流接口，用于实时监控执行过程。

**核心方法**：
- `stream()` - 流式输出最终结果
- `astream()` - 异步流式输出
- `astream_events()` - 流式输出中间步骤和最终结果（推荐）

**事件类型**：
- `on_chain_start` / `on_chain_end`
- `on_chat_model_start` / `on_chat_model_stream` / `on_chat_model_end`
- `on_tool_start` / `on_tool_end`
- `on_retriever_start` / `on_retriever_end`

**示例**：
```python
async for event in agent.astream_events(input, version="v2"):
    kind = event["event"]
    if kind == "on_chat_model_stream":
        print(event["data"]["chunk"].content, end="")
    elif kind == "on_tool_end":
        print(f"Tool {event['name']} finished")
```

**1.0 要求**：
- 必须使用 `version="v2"` 参数（langchain-core >= 0.3.37 后默认）

---

### 27. **Middleware** - 中间件（1.0 新增）

**定义**：Middleware 是 LangChain 1.0 引入的新模式，用于在 Agent 执行前后注入逻辑。

**用途**：
- **PIIMiddleware**：数据脱敏和隐私保护
- **SummarizationMiddleware**：对话历史压缩
- **HumanInTheLoopMiddleware**：人工审批工作流

**特点**：
- 单一职责原则（每个中间件专注一个功能）
- 可组合（多个中间件串联）
- 生产级特性（与 Agent 深度集成）

**示例概念**：
```python
from langchain.middleware import PIIMiddleware, SummarizationMiddleware

agent = create_agent(
    model=model,
    tools=tools,
    middleware=[
        PIIMiddleware(),
        SummarizationMiddleware(max_tokens=2000)
    ]
)
```

---

### 28. **with_structured_output** - 结构化输出

**定义**：强制 LLM 输出符合指定 schema 的结构化数据。

**与 bind_tools 的区别**：
- `bind_tools`：注册工具供模型选择性调用
- `with_structured_output`：强制输出特定格式（不调用外部工具）

**示例**：
```python
from pydantic import BaseModel

class Person(BaseModel):
    name: str
    age: int

structured_model = model.with_structured_output(Person)
result = structured_model.invoke("John is 30 years old")
# result = Person(name="John", age=30)
```

**1.0 变化**：
- 必须显式使用策略（`ToolStrategy` 或 `ProviderStrategy`）
- 不再支持直接传递 schema

---

### 29. **Thread** - 会话线程

**定义**：Thread 是 LangGraph 中的会话标识符，用于隔离不同用户或对话的状态。

**用法**：
```python
config1 = {"configurable": {"thread_id": "user_alice"}}
config2 = {"configurable": {"thread_id": "user_bob"}}

app.invoke(input, config1)  # Alice 的会话
app.invoke(input, config2)  # Bob 的会话
```

**特点**：
- 每个 thread_id 对应独立的 checkpoint 历史
- 支持多租户应用
- 会话可以跨进程恢复（使用持久化 checkpointer）

---

### 30. **LCEL（LangChain Expression Language）** - 表达式语言（已淡化）

**定义**：LCEL 是 0.x 版本的核心，使用 `|` 操作符声明式组合 Runnable。

**0.x 示例**：
```python
chain = (
    {"context": retriever, "question": RunnablePassthrough()}
    | prompt
    | model
    | StrOutputParser()
)
```

**1.0 的转变**：
- 官方声明"LCEL pipes are no longer a thing"
- 推荐使用 `create_agent` 和 LangGraph 替代
- Runnable 接口仍然保留，但不再强调链式组合

**原因**：
- 复杂应用需要循环、条件、状态管理（LCEL 不擅长）
- LangGraph 提供更强大的编排能力
- Agent 优先的架构更符合生产需求

---

## 包架构演进

### 0.x 时代：单体包
```
langchain
└── 所有功能（模型、工具、链、Agent、集成）
```

### 0.1 - 0.2：模块化拆分
```
langchain-core        # 核心抽象（Runnable、Message、工具接口）
langchain-community   # 社区集成（各种 Loader、VectorStore）
langchain             # 通用逻辑（基于 core 的链、Agent）
langchain-{provider}  # 提供商专属包（如 langchain-openai）
```

### 1.0：Agent 优先
```
langchain-core        # 基础抽象（不变）
langchain             # 专注于 Agent（create_agent、中间件）
langchain-community   # 第三方集成
langgraph             # 图编排（独立包，1.0 核心依赖）
langchain-classic     # 0.x 遗留功能
```

---

## 协同工作机制

### 1. **简单 Agent 工作流（ReAct 模式）**

```
用户输入
  ↓
[create_agent]
  ↓
StateGraph (LangGraph)
  ├─→ [ChatModel Node] ──→ 生成响应或工具调用
  │        ↓
  │   有工具调用？
  │    ├─ 是 ──→ [Tool Node] ──→ 执行工具 ──┐
  │    │                                   │
  │    └─ 否 ──→ [END]                    │
  │                                        │
  └───────────────────────────────────────┘
                (循环直到无工具调用)
```

**关键点**：
- Agent 底层是一个 StateGraph
- 自动处理工具调用循环
- 每次循环后通过 conditional edge 判断是否继续

---

### 2. **RAG（检索增强生成）工作流**

```
用户查询
  ↓
[Retriever] ──→ 检索相关文档
  │              ↓
  │         [Document]
  │              ↓
  └─────→ [ChatPromptTemplate] ──→ 格式化上下文
                ↓
           [ChatModel] ──→ 生成答案
                ↓
          [StrOutputParser] ──→ 提取文本
                ↓
            最终答案
```

**数据准备流程**：
```
原始文件
  ↓
[DocumentLoader] ──→ 加载文档
  ↓
[TextSplitter] ──→ 分块
  ↓
[Embeddings] ──→ 向量化
  ↓
[VectorStore] ──→ 存储（支持相似度搜索）
```

---

### 3. **对话型 Agent + 持久化**

```
用户消息
  ↓
[Agent (with Checkpointer)]
  ↓
加载历史（根据 thread_id）
  ↓
[StateGraph]
  ├─→ [ChatModel] ──→ 生成响应
  │        ↓
  │   [Checkpoint] ──→ 保存状态
  │        ↓
  └─→ 返回响应

下次请求
  ↓
根据 thread_id 恢复历史
  ↓
继续对话
```

**关键点**：
- Checkpointer 在每个节点后自动保存状态
- thread_id 用于区分不同会话
- 支持跨进程恢复（使用 SQLite/PostgreSQL）

---

### 4. **人机协作工作流**

```
用户输入
  ↓
[StateGraph]
  ├─→ [分析节点] ──→ 判断是否需要人工审批
  │        ↓
  │   需要审批？
  │    ├─ 是 ──→ [暂停] ──→ 等待人工输入
  │    │            ↓
  │    │       人工审批/修改
  │    │            ↓
  │    └──────→ [继续执行] ──→ [工具节点]
  │                                ↓
  └─────────────────────────────→ [END]
```

**实现方式**：
- 使用 `interrupt_before` 或 `interrupt_after` 配置节点
- 通过 checkpointer 暂停状态
- 人工干预后使用相同 thread_id 恢复执行

---

### 5. **多 Agent 协作**

```
主控 Agent
  ↓
[StateGraph]
  ├─→ [路由节点] ──→ 判断任务类型
  │        ↓
  │   ┌────┴────┬────────┬─────────┐
  │   ↓         ↓        ↓         ↓
  │ [Agent1] [Agent2] [Agent3] [Agent4]
  │  搜索      计算     翻译      总结
  │   ↓         ↓        ↓         ↓
  └─→ [汇总节点] ──→ 整合结果 ──→ [END]
```

**实现方式**：
- 每个子 Agent 是独立的 StateGraph
- 使用 conditional_edges 路由到不同 Agent
- 通过共享状态传递中间结果

---

### 6. **流式输出 + 回调**

```
用户请求
  ↓
[Agent.astream_events()]
  ↓
异步事件流
  ├─→ on_chat_model_stream ──→ 实时显示 token
  ├─→ on_tool_start         ──→ 显示"正在调用工具..."
  ├─→ on_tool_end           ──→ 显示工具结果
  └─→ on_chain_end          ──→ 最终响应
```

**用途**：
- 用户体验优化（实时反馈）
- 调试和监控
- 日志记录

---

## 设计原则

### 1. **统一接口抽象（Runnable Protocol）**

**原则**：所有组件都实现 Runnable 接口，提供一致的调用方式。

**好处**：
- 组件可互换（如切换不同的 ChatModel）
- 组合模式统一（Sequential、Parallel）
- 支持流式、批量、异步的透明切换

**体现**：
```python
# ChatModel、Tool、Agent、Chain 都是 Runnable
result1 = model.invoke(input)
result2 = agent.invoke(input)
result3 = chain.invoke(input)
```

---

### 2. **分层架构（Core、Community、Integration）**

**原则**：核心抽象与实现解耦，第三方集成独立维护。

**分层**：
- **langchain-core**：最小化的核心抽象（稳定）
- **langchain**：高层封装（Agent、工作流）
- **langchain-community**：社区贡献的集成
- **langchain-{provider}**：官方提供商集成

**好处**：
- 核心稳定性（core 很少破坏性变更）
- 集成灵活性（新集成不影响核心）
- 版本管理清晰

---

### 3. **声明式 + 编程式混合**

**声明式（LCEL，0.x 主导）**：
```python
chain = prompt | model | parser  # 声明式组合
```

**编程式（LangGraph，1.0 主导）**：
```python
graph = StateGraph(State)
graph.add_node("process", process_func)
graph.add_edge(START, "process")
app = graph.compile()
```

**1.0 的转变**：
- 简单场景：声明式（LCEL）
- 复杂场景：编程式（LangGraph）
- **推荐**：默认使用 `create_agent`（编程式，但封装良好）

---

### 4. **Agent 优先（1.0 的核心转变）**

**0.x 的问题**：
- Chain 难以处理循环和复杂逻辑
- 工具调用需要手动编排
- 状态管理复杂

**1.0 的解决方案**：
- `create_agent` 作为一等公民
- 底层基于 LangGraph（支持循环、状态、持久化）
- 内置 ReAct 模式
- Middleware 支持生产级特性

---

### 5. **持久化和恢复能力（Checkpointing）**

**原则**：生产级 Agent 必须支持状态持久化和错误恢复。

**实现**：
- 每个节点执行后自动保存 checkpoint
- 支持跨进程恢复（使用外部存储）
- 支持人机协作（暂停/恢复）

**应用场景**：
- 长时间运行的任务
- 服务器重启后继续执行
- 多轮对话的历史记忆

---

### 6. **流式优先（Streaming First）**

**原则**：所有 Runnable 都支持流式输出。

**层次**：
- **Token 级别**：`model.stream()` - 实时输出 token
- **Chunk 级别**：`agent.stream()` - 输出中间结果
- **Event 级别**：`agent.astream_events()` - 细粒度事件流

**好处**：
- 用户体验（实时反馈）
- 调试便利（观察中间步骤）
- 降低延迟感知

---

### 7. **类型安全（TypedDict + Generics）**

**原则**：使用 Python 类型提示确保编译时类型检查。

**1.0 要求**：
- StateGraph 的状态必须是 `TypedDict`（不支持 Pydantic）
- Runnable 使用泛型 `Runnable[InputType, OutputType]`

**示例**：
```python
from typing import TypedDict

class State(TypedDict):
    messages: list
    count: int

# 编译时类型检查
graph = StateGraph(State)
```

---

### 8. **工具调用是一等公民**

**原则**：工具调用不是附加功能，而是核心能力。

**体现**：
- `bind_tools` 是 ChatModel 的标准方法
- AIMessage 内置 `tool_calls` 属性
- ToolMessage 作为独立消息类型
- Agent 自动处理工具调用循环

---

### 9. **模块化和可组合性**

**原则**：小而专注的组件，通过组合构建复杂系统。

**示例**：
- Retriever 专注于检索
- ChatModel 专注于生成
- OutputParser 专注于解析
- 通过 RunnableSequence 组合

**反模式（0.x）**：
- 大而全的 Chain 类（如 `RetrievalQA`）
- 1.0 推荐：自己组合小组件

---

### 10. **生产级特性内置（Middleware）**

**原则**：不仅是原型工具，更是生产框架。

**1.0 新增**：
- 中间件架构（数据脱敏、上下文压缩、审批流程）
- Checkpointing（持久化）
- 错误恢复
- 监控和日志（astream_events）

---

## 架构图

### 1. 整体架构分层

```
┌─────────────────────────────────────────────────┐
│           应用层（User Applications）             │
│   聊天机器人 / RAG / Agent / 工作流自动化           │
└─────────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────┐
│         高层抽象（langchain Package）              │
│   create_agent / Middleware / AgentExecutor    │
└─────────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────┐
│         编排层（LangGraph + LCEL）                │
│   StateGraph / Node / Edge / RunnableSequence  │
└─────────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────┐
│        核心抽象层（langchain-core）                │
│   Runnable / Message / ChatModel / Tool        │
└─────────────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────┐
│        集成层（Community + Providers）            │
│   OpenAI / Anthropic / Pinecone / FAISS        │
└─────────────────────────────────────────────────┘
```

---

### 2. Agent 内部架构

```
create_agent(model, tools, prompt)
            ↓
    生成 StateGraph
            ↓
┌───────────────────────────────────┐
│         StateGraph                │
│                                   │
│  ┌──────┐                         │
│  │START │                         │
│  └──┬───┘                         │
│     │                             │
│     ↓                             │
│  ┌──────────────┐                 │
│  │ ChatModel    │←─────┐          │
│  │ (生成响应)    │      │          │
│  └──┬───────────┘      │          │
│     │                  │          │
│     ↓                  │          │
│  条件判断：             │          │
│  有工具调用？            │          │
│   ├─ 是 ──→ ┌────────┐ │          │
│   │         │ Tools  │─┘          │
│   │         │(执行)  │ (循环)      │
│   │         └────────┘            │
│   │                               │
│   └─ 否 ──→ ┌────┐                │
│             │END │                │
│             └────┘                │
└───────────────────────────────────┘
```

---

### 3. RAG 架构

```
离线索引构建：
┌─────────────┐     ┌──────────────┐     ┌───────────┐
│ Documents   │ ──→ │ TextSplitter │ ──→ │ Embeddings│
└─────────────┘     └──────────────┘     └─────┬─────┘
                                                ↓
                                         ┌──────────────┐
                                         │ VectorStore  │
                                         └──────────────┘

在线查询流程：
┌──────────────┐
│  用户查询     │
└──────┬───────┘
       │
       ↓
┌──────────────┐     ┌──────────────┐
│  Embeddings  │ ──→ │ VectorStore  │
│  (查询向量)   │     │ (相似度搜索) │
└──────────────┘     └──────┬───────┘
                            │
                            ↓
                     ┌──────────────┐
                     │  Retriever   │
                     │  (返回文档)   │
                     └──────┬───────┘
                            │
                            ↓
              ┌─────────────────────────┐
              │  ChatPromptTemplate     │
              │  (格式化上下文 + 查询)   │
              └─────────┬───────────────┘
                        │
                        ↓
              ┌─────────────────────────┐
              │      ChatModel          │
              │    (生成答案)            │
              └─────────┬───────────────┘
                        │
                        ↓
              ┌─────────────────────────┐
              │    StrOutputParser      │
              └─────────┬───────────────┘
                        │
                        ↓
                   最终答案
```

---

### 4. 数据流：Message 类型转换

```
用户输入（文本）
       ↓
┌──────────────┐
│ HumanMessage │
└──────┬───────┘
       │
       ↓
  ChatModel
       ↓
┌──────────────┐
│  AIMessage   │ ──→ tool_calls: [{"name": "get_weather", "id": "123"}]
└──────┬───────┘
       │
       ↓
  Tools 执行
       ↓
┌──────────────┐
│ ToolMessage  │ ──→ content: "72°F", tool_call_id: "123"
└──────┬───────┘
       │
       ↓
  ChatModel（第二轮）
       ↓
┌──────────────┐
│  AIMessage   │ ──→ content: "The weather is 72°F and sunny"
└──────┬───────┘
       │
       ↓
   用户看到结果
```

---

### 5. Checkpointing 工作流

```
第一次调用（thread_id="user_1"）
       ↓
┌─────────────────────────────┐
│  StateGraph                 │
│                             │
│  Node1 ──→ [Checkpoint 1]   │
│    ↓                        │
│  Node2 ──→ [Checkpoint 2]   │
│    ↓                        │
│  Node3 ──→ [Checkpoint 3]   │
└─────────────────────────────┘
       ↓
┌─────────────────────────────┐
│  Checkpointer (SQLite)      │
│  thread_id="user_1"         │
│  ├─ checkpoint_1            │
│  ├─ checkpoint_2            │
│  └─ checkpoint_3            │
└─────────────────────────────┘

第二次调用（同一 thread_id）
       ↓
从 Checkpointer 加载 checkpoint_3
       ↓
继续执行 Node4
       ↓
保存 checkpoint_4
```

---

## 与其他框架的对比

### LangChain 1.0 vs Eino (Go)

| 维度 | LangChain 1.0 | Eino |
|------|--------------|------|
| **核心抽象** | Runnable（4方法） | Runnable（4流式范式） |
| **编排方式** | LangGraph（StateGraph） | Chain/Graph/Workflow |
| **状态管理** | TypedDict + Checkpointer | StatePreHandler/PostHandler |
| **Agent** | create_agent（基于 LangGraph） | ReAct Agent（基于 Graph） |
| **工具调用** | bind_tools + tool_calls | ToolsNode + ToolCallingModel |
| **持久化** | MemorySaver/SQLite/PostgreSQL | 需自行实现 |
| **中间件** | 内置（PIIMiddleware 等） | 无明确概念 |
| **执行模型** | Pregel（SuperStep） | Pregel + DAG |

**相似点**：
- 都基于 Runnable 抽象
- 都支持图编排
- 都内置 ReAct Agent

**差异点**：
- LangChain 1.0 更重 Agent 和生产特性（中间件、持久化）
- Eino 更轻量，提供更多编排选择（Chain/Graph/Workflow）

---

### LangChain 1.0 vs Genkit (Go)

| 维度 | LangChain 1.0 | Genkit |
|------|--------------|--------|
| **核心抽象** | Runnable | Action |
| **编排方式** | LangGraph（图） | Flow（链式） |
| **注册机制** | 无全局注册 | Registry 全局注册 |
| **Agent** | create_agent | 需手动实现 ReAct |
| **持久化** | 内置 Checkpointer | FlowState 手动管理 |
| **工具调用** | 原生支持（bind_tools） | 需通过 Tool Action 实现 |
| **监控** | astream_events + Callback | Telemetry + Tracing |

**相似点**：
- 都支持流式输出
- 都有统一的执行接口

**差异点**：
- LangChain 1.0 的图编排能力远超 Genkit 的线性 Flow
- Genkit 的全局注册机制（Registry）LangChain 没有
- LangChain 1.0 的 Agent 抽象更高级

---

## 典型使用场景

### 1. 简单聊天机器人

```python
from langchain_openai import ChatOpenAI
from langchain_core.messages import HumanMessage

model = ChatOpenAI(model="gpt-4")
response = model.invoke([HumanMessage(content="Hello!")])
print(response.content)
```

---

### 2. 带工具的 Agent

```python
from langchain.agents import create_agent
from langchain_core.tools import tool

@tool
def get_weather(city: str) -> str:
    """Get weather for a city."""
    return f"Weather in {city}: 72°F, sunny"

agent = create_agent(
    model=ChatOpenAI(model="gpt-4o"),
    tools=[get_weather],
    system_prompt="You are a helpful assistant"
)

result = agent.invoke({
    "messages": [HumanMessage(content="What's the weather in SF?")]
})
```

---

### 3. RAG 系统

```python
from langchain_community.document_loaders import TextLoader
from langchain_text_splitters import RecursiveCharacterTextSplitter
from langchain_openai import OpenAIEmbeddings
from langchain_community.vectorstores import FAISS
from langchain_openai import ChatOpenAI
from langchain_core.prompts import ChatPromptTemplate

# 1. 加载和分块
loader = TextLoader("docs.txt")
documents = loader.load()
splitter = RecursiveCharacterTextSplitter(chunk_size=1000)
chunks = splitter.split_documents(documents)

# 2. 向量化和存储
embeddings = OpenAIEmbeddings()
vectorstore = FAISS.from_documents(chunks, embeddings)
retriever = vectorstore.as_retriever(k=4)

# 3. 构建 RAG 链
prompt = ChatPromptTemplate.from_messages([
    ("system", "Answer based on context:\n{context}"),
    ("human", "{question}")
])

model = ChatOpenAI(model="gpt-4")

# 4. 查询
docs = retriever.invoke("What is LangChain?")
response = model.invoke(
    prompt.format_messages(
        context="\n".join([doc.page_content for doc in docs]),
        question="What is LangChain?"
    )
)
```

---

### 4. 多轮对话（带记忆）

```python
from langgraph.graph import StateGraph, MessagesState, START, END
from langgraph.checkpoint.memory import MemorySaver

def chatbot(state: MessagesState):
    return {"messages": [model.invoke(state["messages"])]}

graph = StateGraph(MessagesState)
graph.add_node("chat", chatbot)
graph.add_edge(START, "chat")
graph.add_edge("chat", END)

memory = MemorySaver()
app = graph.compile(checkpointer=memory)

# 第一轮
config = {"configurable": {"thread_id": "user_123"}}
app.invoke({"messages": [HumanMessage(content="My name is Alice")]}, config)

# 第二轮（自动记住历史）
app.invoke({"messages": [HumanMessage(content="What's my name?")]}, config)
# 输出: "Your name is Alice"
```

---

### 5. 人机协作 Agent

```python
from langgraph.graph import StateGraph, START, END

def analyze(state):
    # 分析任务，标记是否需要审批
    return {"needs_approval": True, "action": "delete_database"}

def execute(state):
    # 执行操作
    return {"result": "Action executed"}

graph = StateGraph(State)
graph.add_node("analyze", analyze)
graph.add_node("execute", execute)
graph.add_edge(START, "analyze")

# 在 execute 前暂停，等待人工审批
app = graph.compile(interrupt_before=["execute"])

# 执行
config = {"configurable": {"thread_id": "task_1"}}
result = app.invoke(input, config)  # 在 execute 前暂停

# 人工审批后继续
app.invoke(None, config)  # 从断点继续
```

---

## 总结

LangChain 1.0 是一次彻底的架构升级，从"链式组合的原型工具"转变为"图编排的生产框架"。核心变化包括：

1. **Agent 优先**：`create_agent` 成为核心 API，底层基于 LangGraph
2. **LangGraph 崛起**：StateGraph 替代 LCEL 链，支持循环、条件、状态管理
3. **生产级特性**：Checkpointing、Middleware、人机协作、错误恢复
4. **模块化重构**：langchain-core（稳定抽象）+ langchain（高层封装）+ langgraph（编排引擎）
5. **LCEL 淡化**：pipes (`|`) 不再推荐，但 Runnable 接口保留

LangChain 1.0 的设计哲学是"工程优先"，强调类型安全、持久化、监控、中间件等生产必备特性，使其成为构建复杂、可靠的 LLM 应用的首选框架。
