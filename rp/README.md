# Role Play Chat

基于 eino agent 模式的角色扮演对话程序，支持通过配置文件自定义角色，使用 Agent Instruction 管理 system prompt。

## 功能特性

- ✅ 使用 eino agent 模式实现
- ✅ 支持从配置文件读取角色信息
- ✅ 使用 Agent Instruction 字段管理 system prompt
- ✅ 支持 stdin 交互式对话
- ✅ 维护完整的对话历史
- ✅ 多角色示例配置
- ✅ **多 Model Client 支持**：根据 model 名称自动切换 AI 服务（OpenAI、DeepSeek 等）

## 项目结构

```
rp/
├── cmd/
│   └── rpchat/
│       └── main.go          # 主程序
├── config.go                # 配置文件定义和读取
├── client_factory.go        # Model Client 工厂（自动切换）
├── client_factory_test.go   # Client 工厂测试
├── examples/                # 示例角色配置
│   ├── alice.yaml           # 爱丽丝（冒险家）
│   ├── sage.yaml            # 智者（哲学家）
│   ├── detective.yaml       # 侦探
│   └── phoebe.yaml          # 菲比（游戏角色）
└── README.md                # 说明文档
```

## 支持的 Model Client

系统会根据 `-model` 参数中的关键词自动选择对应的 client：

| Model 关键词 | Client 类型 | 示例 Model |
|------------|-----------|-----------|
| `deepseek` | DeepSeek Client | `deepseek-chat`, `DeepSeek-V3` |
| `gpt` 或 `openai` | OpenAI Client | `gpt-4o`, `gpt-4o-mini`, `gpt-3.5-turbo` |
| 其他 | OpenAI Client（默认） | 兼容 OpenAI API 格式的第三方服务 |

### 扩展支持

未来可通过编辑 `client_factory.go` 轻松添加更多 client：
- Claude（Anthropic）
- Gemini（Google）
- Qwen（阿里）
- Ark（字节）
- Ollama（本地部署）

## 配置文件格式

角色配置使用 YAML 格式，支持以下字段：

```yaml
name: "角色名称"              # 必填
system_prompt: |             # 必填
  角色的详细设定和行为描述
  可以多行
```

## 使用方法

### 1. 安装依赖

```bash
go mod tidy
```

### 2. 设置 API Key

```bash
export OPENAI_API_KEY="your-api-key"
```

或者通过命令行参数指定：

```bash
-api-key your-api-key
```

### 3. 运行程序

#### 使用 OpenAI 兼容服务

使用示例配置：

```bash
# 与爱丽丝对话
go run rp/cmd/rpchat/main.go -config rp/examples/alice.yaml \
  -model gpt-4o-mini \
  -api-key your-api-key \
  -base-url https://aihubmix.com/v1

# 与智者对话
go run rp/cmd/rpchat/main.go -config rp/examples/sage.yaml \
  -model gpt-4o \
  -api-key your-api-key

# 与侦探对话
go run rp/cmd/rpchat/main.go -config rp/examples/detective.yaml \
  -model gpt-3.5-turbo \
  -api-key your-api-key
```

#### 使用 DeepSeek

```bash
# 使用 DeepSeek Chat 模型
go run rp/cmd/rpchat/main.go -config rp/examples/phoebe.yaml \
  -model deepseek-chat \
  -api-key your-deepseek-key

# 使用 DeepSeek V3 模型（需要指定 base-url）
go run rp/cmd/rpchat/main.go -config rp/examples/alice.yaml \
  -model DeepSeek-V3 \
  -api-key your-deepseek-key \
  -base-url https://api.deepseek.com
```

#### 编译后运行

```bash
# 编译程序
cd rp/cmd/rpchat
go build

# 使用 OpenAI 兼容服务
./rpchat -config ../../examples/alice.yaml \
  -model gpt-4o-mini \
  -api-key your-api-key

# 使用 DeepSeek
./rpchat -config ../../examples/phoebe.yaml \
  -model deepseek-chat \
  -api-key your-deepseek-key
```

#### 使用自定义配置

```bash
go run rp/cmd/rpchat/main.go -config /path/to/your/config.yaml \
  -model gpt-4o-mini \
  -api-key your-api-key
```

### 4. 命令行参数

| 参数 | 说明 | 默认值 | 必填 |
|-----|------|--------|------|
| `-config` | 角色配置文件路径 | `rp/examples/phoebe.yaml` | 是 |
| `-model` | 模型名称（自动识别 client 类型） | `gpt-4o-mini` | 否 |
| `-api-key` | API Key | 无 | 是 |
| `-base-url` | API 基础 URL | 根据 client 自动选择 | 否 |

**说明**：
- `-model` 参数会根据关键词自动选择对应的 client（见"支持的 Model Client"章节）
- `-base-url` 可选，如不指定则使用各 client 的默认 API 地址

### 5. 交互命令

- 输入消息后按回车发送
- 输入 `exit` 或 `quit` 退出对话

## 示例对话

```
=== Role Play Chat ===
角色: 爱丽丝
======================
输入 'exit' 或 'quit' 退出对话

你: 你好！
爱丽丝: 你好呀！真高兴见到你！今天有什么有趣的事情要和我分享吗？

你: 我想去旅行
爱丽丝: 太棒了！旅行是多么令人兴奋的事情啊！你想去哪里呢？是神秘的森林，还是遥远的海岛？告诉我，我也想听听你的冒险计划！

你: exit
再见！
```

## 技术实现

### 核心技术栈

- **eino framework**: 使用 agent 模式进行对话管理
- **Agent Instruction**: 通过 Agent 的 Instruction 字段管理 system prompt
- **OpenAI API**: LLM 模型调用
- **YAML**: 配置文件格式

### 关键特性

1. **Agent 模式管理 system prompt**
   ```go
   agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
       Name:          roleConfig.Name,
       Instruction:   roleConfig.SystemPrompt,  // system prompt 通过 Instruction 字段管理
       Model:         chatModel,
       MaxIterations: 10,
   })
   ```

2. **对话历史管理**
   ```go
   // 维护消息历史
   messageHistory := []adk.Message{}

   // 添加用户消息
   userMessage := schema.UserMessage(userInput)
   messageHistory = append(messageHistory, userMessage)

   // 运行 Agent
   agentInput := &adk.AgentInput{
       Messages: messageHistory,
   }
   iter := agent.Run(ctx, agentInput)

   // 添加助手回复到历史
   messageHistory = append(messageHistory, assistantMessage)
   ```

3. **事件流处理**
   - Agent 返回异步迭代器，通过事件流获取响应
   - 自动维护完整的对话历史
   - 每轮对话都包含之前的上下文

## 自定义角色

创建自己的角色配置文件：

```yaml
name: "你的角色名称"
system_prompt: |
  详细的角色设定：
  - 角色身份
  - 性格特点
  - 说话方式
  - 行为习惯
  - 价值观念
```

## 注意事项

1. 配置文件必须是有效的 YAML 格式
2. `name` 和 `system_prompt` 字段为必填
3. 确保 API Key 有效且有足够的额度
4. 对话历史会一直保留在内存中，直到程序退出

## 扩展建议

### 添加新的 Model Client

如果想添加对更多 AI 服务的支持（如 Claude、Gemini 等），可以按以下步骤操作：

1. **添加依赖**
   ```bash
   go get github.com/cloudwego/eino-ext/components/model/[client-name]
   ```

2. **编辑 `client_factory.go`**

   在 `CreateChatModel` 函数的 switch 语句中添加新的关键词匹配：
   ```go
   case strings.Contains(modelLower, "claude"):
       return createClaudeClient(ctx, config)
   case strings.Contains(modelLower, "gemini"):
       return createGeminiClient(ctx, config)
   ```

3. **实现创建函数**

   添加对应的 `createXXXClient` 函数：
   ```go
   func createClaudeClient(ctx context.Context, config *ModelClientConfig) (model.ToolCallingChatModel, error) {
       claudeConfig := &claude.ChatModelConfig{
           Model:  config.Model,
           APIKey: config.APIKey,
       }
       if config.BaseURL != "" {
           claudeConfig.BaseURL = config.BaseURL
       }
       return claude.NewChatModel(ctx, claudeConfig)
   }
   ```

4. **测试**
   ```bash
   # 运行测试
   go test -v gentica1/rp

   # 测试实际使用
   go run rp/cmd/rpchat/main.go -config rp/examples/alice.yaml \
     -model claude-3-opus \
     -api-key your-claude-key
   ```

### 其他扩展建议

- 添加对话历史保存功能
- 支持多模态输入（图片、语音等）
- 添加工具调用能力（如搜索、计算等）
- 支持流式输出
