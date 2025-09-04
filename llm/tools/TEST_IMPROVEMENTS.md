# LLM 工具测试改进方案

基于对现有测试覆盖率的分析，以下是为每个工具推荐的边界情况和改进建议，以实现更健壮的测试：

## 1. bash_test.go

### 缺失的边界情况：
- **命令中的特殊字符**：测试包含引号、反斜杠、管道等的命令
- **超长命令**：测试超过典型命令行限制的命令
- **包含环境变量的命令**：测试 $PATH、$HOME 等的使用
- **标准错误与标准输出的分离**：测试同时向两个流输出但内容不同的命令
- **信号处理**：测试被信号中断的命令
- **Unicode/UTF-8 处理**：测试包含非 ASCII 字符的命令
- **命令注入尝试**：更彻底地测试安全边界
- **并发命令执行**：测试多个 bash 工具同时运行
- **目录变更**：测试 `cd` 命令及其对后续命令的影响
- **后台进程**：测试产生后台进程的命令

### 建议的测试用例：
```go
// Test special characters
t.Run("command with special characters", func(t *testing.T) {
    params := BashParams{
        Command: `echo "test with 'quotes' and \backslash"`,
    }
    // ... verify proper escaping
})

// Test very long output
t.Run("command with output exceeding max length", func(t *testing.T) {
    params := BashParams{
        Command: "seq 1 100000", // generates very long output
    }
    // ... verify truncation behavior
})

// Test environment variables
t.Run("command with environment variables", func(t *testing.T) {
    params := BashParams{
        Command: "echo $HOME",
    }
    // ... verify env var expansion
})
```

## 2. edit_test.go

### 缺失的边界情况：
- **二进制文件编辑**：测试尝试编辑二进制文件的情况
- **超大文件**：测试 >100MB 的文件编辑
- **文件权限**：测试只读文件、仅可执行文件
- **符号链接**：测试通过符号链接进行编辑
- **行结束符差异**：测试 CRLF 与 LF 的处理
- **Unicode 替换**：测试替换非 ASCII 字符
- **多行替换**：测试跨多行的文本替换
- **正则元字符**：测试包含正则元字符的替换文本
- **并发编辑**：测试同时对同一文件进行多次编辑
- **old_string 为空但文件存在**：边界情况校验

### 建议的测试用例：
```go
// Test binary file rejection
t.Run("reject binary file edit", func(t *testing.T) {
    binaryPath := filepath.Join(tempDir, "binary.dat")
    err := os.WriteFile(binaryPath, []byte{0x00, 0xFF, 0xDE, 0xAD}, 0o644)
    // ... verify binary file is rejected
})

// Test symlink handling
t.Run("edit through symlink", func(t *testing.T) {
    targetPath := filepath.Join(tempDir, "target.txt")
    linkPath := filepath.Join(tempDir, "link.txt")
    os.Symlink(targetPath, linkPath)
    // ... verify symlink behavior
})
```

## 3. view_test.go

### 缺失的边界情况：
- **二进制文件查看**：测试二进制文件的检测与处理
- **超大文件**：测试 >100MB 文件的分页查看
- **无换行的文件**：测试单行非常长的文件
- **Unicode/表情符号内容**：测试正确的 UTF-8 渲染
- **特殊行分隔符**：测试包含 CRLF、仅 CR 的文件
- **并发读取**：测试多重同时读取
- **读取过程中文件被修改**：测试文件在读取时被修改的情况
- **命名管道/FIFO**：测试从特殊文件读取
- **权限被拒绝**：测试无读取权限的文件
- **偏移超出文件长度**：测试无效的偏移值

### 建议的测试用例：
```go
// Test binary file detection
t.Run("detect and handle binary file", func(t *testing.T) {
    binaryPath := filepath.Join(tempDir, "binary.exe")
    os.WriteFile(binaryPath, []byte{0x7F, 'E', 'L', 'F'}, 0o644)
    // ... verify binary file handling
})

// Test offset beyond EOF
t.Run("offset beyond file length", func(t *testing.T) {
    params := ViewParams{
        FilePath: smallFile,
        Offset: 1000000,
    }
    // ... verify graceful handling
})
```

## 4. write_test.go

### 缺失的边界情况：
- **磁盘空间耗尽**：测试磁盘已满时的写入
- **权限被拒绝**：测试写入到只读目录
- **符号链接目标**：测试通过符号链接写入
- **特殊文件名**：测试包含空格、Unicode 的文件名
- **并发写入**：测试对同一文件的多次写入
- **超大内容**：测试写入 >100MB 的内容
- **二进制内容**：测试写入二进制数据
- **文件锁定**：测试写入被锁定的文件
- **网络驱动器**：测试写入到挂载的网络路径
- **路径中的非法字符**：测试路径遍历等尝试

### 建议的测试用例：
```go
// Test permission denied
t.Run("write to read-only directory", func(t *testing.T) {
    readOnlyDir := filepath.Join(tempDir, "readonly")
    os.Mkdir(readOnlyDir, 0o444)
    params := WriteParams{
        FilePath: filepath.Join(readOnlyDir, "test.txt"),
        Content: "test",
    }
    // ... verify permission error handling
})
```

## 5. glob_test.go

### 缺失的边界情况：
- **大小写敏感性**：测试大小写敏感与不敏感的匹配
- **隐藏文件**：测试匹配 . 开头的文件（dotfiles）
- **符号链接跟随**：测试通过符号链接的 glob 匹配
- **循环符号链接**：测试防止无限循环
- **超深嵌套**：测试包含深层目录遍历的模式
- **特殊 glob 模式**：测试 `**`、`?`、`[abc]` 等模式
- **大量文件时的性能**：测试 10000+ 文件的情况
- **模式中的 Unicode**：测试包含非 ASCII 的 glob 模式
- **转义字符**：测试模式中把 `*` 和 `?` 当作字面值
- **空目录**：测试匹配到零个文件的模式

### 建议的测试用例：
```go
// Test hidden files
t.Run("match hidden files", func(t *testing.T) {
    params := GlobParams{
        Pattern: ".*",
        Path: tempDir,
    }
    // ... verify dotfile matching
})

// Test symlink loops
t.Run("handle circular symlinks", func(t *testing.T) {
    os.Symlink(".", filepath.Join(tempDir, "loop"))
    params := GlobParams{
        Pattern: "**/loop/*",
    }
    // ... verify loop prevention
})
```

## 6. grep_test.go

### 缺失的边界情况：
- **二进制文件处理**：测试在二进制文件中 grep 的行为
- **超长行**：测试 >10MB 的行
- **大小写不敏感搜索**：测试等价于 -i 标志的行为
- **单词边界**：测试 \b 正则模式
- **多行模式**：测试跨行的匹配模式
- **性能**：测试在 1GB+ 文件中的 grep
- **Unicode 正则**：测试 Unicode 字符类
- **负向前瞻/回顾**：测试高级正则特性
- **文件编码**：测试非 UTF-8 编码文件
- **并发搜索**：测试并行 grep 操作

### 建议的测试用例：
```go
// Test binary file skipping
t.Run("skip binary files", func(t *testing.T) {
    binaryPath := filepath.Join(tempDir, "binary.exe")
    os.WriteFile(binaryPath, []byte{0xFF, 0xFE, 0x00, 0x00}, 0o644)
    params := GrepParams{
        Pattern: "test",
    }
    // ... verify binary files are skipped
})

// Test very long lines
t.Run("handle very long lines", func(t *testing.T) {
    longLine := strings.Repeat("a", 10*1024*1024) + "needle"
    params := GrepParams{
        Pattern: "needle",
    }
    // ... verify long line handling
})
```

## 7. 需要补充的其他工具测试

### ls_test.go
- **符号链接显示**：测试显示符号链接目标
- **权限显示**：测试文件权限格式化
- **大小显示格式**：测试人类可读的大小格式
- **隐藏文件**：测试列出 .dotfile
- **排序选项**：测试不同排序顺序

### multiedit_test.go
- **事务回滚**：测试部分失败场景
- **顺序依赖**：测试依赖于先前编辑的编辑操作
- **冲突编辑**：测试重叠替换的冲突情况
- **性能**：测试单次操作 100+ 次编辑

### download_test.go
- **断点续传**：测试部分下载的续传能力
- **HTTPS 证书**：测试无效证书的处理
- **重定向**：测试跟随 HTTP 重定向
- **Content-Type 验证**：测试 MIME 类型检查
- **压缩处理**：测试 gzip/deflate 的处理

### fetch_test.go
- **JavaScript 渲染**：测试 SPA 内容抓取
- **认证**：测试基本认证、Bearer Token
- **Cookies**：测试 Cookie 的处理
- **速率限制**：测试重试逻辑
- **代理支持**：测试代理配置

## 8. 跨工具集成测试

### 建议的集成测试：
```go
// Test edit after view
t.Run("edit file after viewing", func(t *testing.T) {
    // First view file to establish read record
    viewTool.Run(ctx, viewCall)
    // Then edit the same file
    editTool.Run(ctx, editCall)
    // Verify file records are consistent
})

// Test grep after write
t.Run("grep in newly written file", func(t *testing.T) {
    // Write file with specific content
    writeTool.Run(ctx, writeCall)
    // Immediately grep for that content
    grepTool.Run(ctx, grepCall)
    // Verify content is found
})
```

## 9. 性能与压力测试

### 建议的压力测试：
- **并发工具使用**：同时运行 100 个工具
- **内存泄漏检测**：循环运行工具 10000 次
- **超大文件处理**：测试 1GB+ 的文件
- **深目录树**：测试 1000+ 层嵌套目录
- **大量文件**：测试目录中有 100000+ 个文件

## 10. 安全测试

### 建议的安全测试：
- **路径遍历**：测试 `../../../etc/passwd` 等尝试
- **命令注入**：测试参数中包含 `; rm -rf /` 的情况
- **XML/JSON 注入**：测试格式错误的输入参数
- **资源耗尽**：测试无限循环、fork 炸弹等
- **权限提升**：测试 sudo/su 命令尝试

## 实施优先级

1. **高优先级**（对健壮性至关重要）：
    - 各工具的二进制文件处理
    - 权限与错误处理
    - 并发操作安全
    - 安全边界测试

2. **中等优先级**（对完整性重要）：
    - Unicode/UTF-8 处理
    - 大文件处理
    - 符号链接行为
    - 性能测试

3. **低优先级**（可选项）：
    - 边缘情况的特殊字符
    - 网络路径处理
    - 高级正则特性
    - 压力测试

## 测试最佳实践

1. **对类似的测试用例使用表格驱动测试**
2. **在可能的情况下使用 `t.Parallel()` 并行化测试**
3. **使用 `t.Cleanup()` 或 `defer` 清理资源**
4. **对外部依赖（网络、文件系统）进行模拟**
5. **测试成功路径和失败路径**
6. **不仅验证返回值，还要验证副作用**
7. **使用能描述场景的有意义的测试名称**
8. **为性能关键操作添加基准测试**