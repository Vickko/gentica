# LLM Tools Integration Test Guide

## 如何运行测试

### 前提条件
1. 确保 `config/config.yaml` 包含有效的 API 配置
2. 确保网络可以访问配置的 LLM API endpoint

### 运行方式

#### 方式1: 使用交互式脚本（推荐）
```bash
cd chat-new
./run_verbose_test.sh
```

这个脚本会:
- 显示所有 LLM 交互细节
- 彩色高亮不同类型的输出
- 让你选择运行哪个测试

#### 方式2: 直接运行特定测试

**简单测试** (最快，~5秒):
```bash
go test -v ./chat-new -run TestRealLLMSimpleToolCall
```

**工具链测试** (中等，~15秒):
```bash
go test -v ./chat-new -run TestRealLLMToolChaining
```

**全工具测试** (最全面，~30秒):
```bash
go test -v ./chat-new -run TestRealLLMWithAllTools
```

**运行所有测试**:
```bash
go test -v ./chat-new -run TestRealLLM
```

### 测试输出说明

测试会显示以下信息：

#### 1. 初始化信息
```
=== Starting LLM Integration Test ===
Testing all available tools with real LLM API
```

#### 2. 发送的Prompt
```
=== Sending prompt to LLM ===
Prompt:
Please use the bash tool to run: echo 'Hello from bash tool'
```

#### 3. 工具调用详情
```
=== Tool Call Details ===
Tool called: bash
Arguments: {"command":"echo 'Hello from bash tool'"}
Tool result: Hello from bash tool
```

#### 4. LLM响应
```
=== LLM Response ===
The command echo 'Hello from bash tool' was executed successfully...
```

#### 5. 验证结果
```
=== Verifying File Operations ===
✅ test.txt created
   Content: Hello LLM
   This is a test file
   Created by LLM
✅ Edit successfully changed 'Hello World' to 'Hello LLM'
```

#### 6. 工具使用统计
```
=== Tool Usage Verification ===
✅ Tool 'ls' was used 2 time(s)
✅ Tool 'write' was used 2 time(s)
✅ Tool 'view' was used 1 time(s)
...
```

### 查看DEBUG输出

默认情况下会显示一些DEBUG信息（来自chat.go）：
```
DEBUG: Received 1 tool calls
DEBUG: Tool call 0: bash with args: {"command":"echo 'Hello from bash tool'"}
```

### 跳过测试

如果需要跳过LLM测试（比如在CI环境）:
```bash
SKIP_LLM_TEST=1 go test ./chat-new
```

### 测试目录

测试会在临时目录创建文件:
- macOS: `/var/folders/.../T/llm_test_<timestamp>`
- Linux: `/tmp/llm_test_<timestamp>`

测试完成后会自动清理。

### 常见问题

**Q: 测试失败提示 "Config file not found"**
A: 确保 `config/config.yaml` 存在并包含有效配置

**Q: 测试超时**
A: 网络较慢时可增加超时时间:
```bash
go test -v ./chat-new -run TestRealLLMWithAllTools -timeout 600s
```

**Q: 想看更详细的输出**
A: 测试已经包含了详细的日志，会显示:
- 每个工具调用
- 工具参数
- 工具返回结果
- LLM的完整响应

### 测试覆盖的工具

| 工具 | 功能 | 测试内容 |
|-----|------|---------|
| bash | 执行命令 | echo命令 |
| write | 写文件 | 创建test.txt, test2.md, data.json |
| view | 读文件 | 读取创建的文件 |
| edit | 编辑文件 | 修改文本内容 |
| multiedit | 批量编辑 | 多处修改 |
| grep | 搜索内容 | 搜索特定模式 |
| glob | 文件匹配 | 查找*.txt文件 |
| ls | 列目录 | 显示文件列表 |
| fetch | 获取URL | 访问GitHub API |
| download | 下载文件 | (在综合测试中可选) |

### 输出示例

成功的测试输出会类似:
```
=== Starting Simple Tool Call Test ===
Prompt: Please use the bash tool to run: echo 'Hello from bash tool'

Sending message to LLM...

=== Tool Call Details ===
Tool called: bash
Arguments: {"command":"echo 'Hello from bash tool'"}
Tool result: Hello from bash tool
✅ Bash tool executed successfully

=== Test Result: PASS ✅ ===
PASS
ok      gentica/chat-new        4.350s
```