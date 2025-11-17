# Eino Tool 实例管理研究报告

## 研究问题

**多个 ChatModel 可能需要调用同一种 tool，eino 是否对以下场景有设计解决方案：**
- 有时需要这个 tool 的同一个实例（共享状态）
- 有时需要这个 tool 的不同实例（状态隔离）

## 研究结论

✅ **eino 通过 Go 语言的接口引用传递特性，天然支持 tool 实例的共享和隔离**

这不是框架特意设计的 feature，而是利用 Go 语言的特性：
- **共享实例**：多个 ToolsNode 传递同一个 tool 实例引用 → 共享状态
- **独立实例**：每个 ToolsNode 创建新的 tool 实例 → 状态隔离

## 验证实验

### 实验1：共享实例场景

```go
// 创建一个共享的 counter 实例
sharedCounter := NewStatefulCounterTool("shared_counter")

// 两个 ToolsNode 使用同一个实例
toolsNode1, _ := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
    Tools: []tool.BaseTool{sharedCounter},
})

toolsNode2, _ := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
    Tools: []tool.BaseTool{sharedCounter}, // 同一个实例
})
```

**实验结果**：
```
ToolsNode1 第一次调用: Counter value: 5   (增加 5)
ToolsNode2 调用:       Counter value: 8   (从 5 增加 3) ✅ 状态共享
ToolsNode1 第二次调用: Counter value: 10  (从 8 增加 2) ✅ 状态持续累积
```

### 实验2：独立实例场景

```go
// 创建两个独立的 counter 实例
counter1 := NewStatefulCounterTool("counter_1")
counter2 := NewStatefulCounterTool("counter_2")

toolsNode3, _ := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
    Tools: []tool.BaseTool{counter1},
})

toolsNode4, _ := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
    Tools: []tool.BaseTool{counter2}, // 不同的实例
})
```

**实验结果**：
```
ToolsNode3 (counter_1): Counter value: 10  (增加 10)
ToolsNode4 (counter_2): Counter value: 20  (独立计数，增加 20) ✅ 状态隔离
ToolsNode3 再次调用:    Counter value: 20  (从 10 增加 10) ✅ 不受 counter_2 影响
```

### 实验3：Graph 中的场景

在实际的 Graph 编排中，多个 Agent 子图可以：
- **Agent1 和 Agent2**：共享同一个 tool 实例 → 状态累积 (1 → 2 → 3 → 4)
- **Agent3**：使用独立的 tool 实例 → 独立计数 (100)
- **状态隔离**：Agent3 不影响 Agent1/Agent2 的状态 ✅

## Tool 实例管理的四种模式

### 1. 共享实例模式（Shared Instance Pattern）

**用途**：多个 ChatModel/ToolsNode 需要共享状态

**实现**：
```go
sharedTool := NewMyTool()
toolsNode1 := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
    Tools: []tool.BaseTool{sharedTool}
})
toolsNode2 := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
    Tools: []tool.BaseTool{sharedTool} // 同一个实例
})
```

**注意**：⚠️ 需要自行处理并发安全（使用 `sync.Mutex` 等）

**适用场景**：
- 数据库连接池
- 全局计数器
- 共享缓存
- 资源池管理

### 2. 独立实例模式（Isolated Instance Pattern）

**用途**：每个 ChatModel/ToolsNode 需要独立状态

**实现**：
```go
tool1 := NewMyTool()
tool2 := NewMyTool()  // 新实例
toolsNode1 := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
    Tools: []tool.BaseTool{tool1}
})
toolsNode2 := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
    Tools: []tool.BaseTool{tool2}
})
```

**优点**：✅ 状态完全隔离，无需考虑并发问题

**适用场景**：
- 用户会话管理
- 独立的临时状态
- 不同配置的同类工具
- 测试隔离

### 3. 工厂模式（Factory Pattern）

**用途**：根据上下文动态创建 tool 实例

**实现**：
```go
type ToolFactory func() tool.BaseTool

factory := func() tool.BaseTool {
    return NewMyTool()
}

// 在创建 ToolsNode 时调用工厂
toolsNode := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
    Tools: []tool.BaseTool{factory()}
})
```

**优点**：✅ 延迟创建，可根据运行时信息定制

**适用场景**：
- 需要根据请求参数创建不同配置的 tool
- 资源按需创建
- 单元测试中的 mock

### 4. 混合模式（Hybrid Pattern）

**用途**：某些 tools 共享，某些 tools 独立

**实现**：
```go
sharedDB := NewDBTool()   // 共享数据库连接
cache1 := NewCacheTool()  // 独立缓存
cache2 := NewCacheTool()  // 独立缓存

toolsNode1 := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
    Tools: []tool.BaseTool{sharedDB, cache1}
})

toolsNode2 := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
    Tools: []tool.BaseTool{sharedDB, cache2}  // 共享 DB，独立 cache
})
```

**优点**：✅ 灵活性最高，可根据需求组合

**适用场景**：
- 多租户系统（共享基础设施，隔离租户数据）
- 微服务架构（共享服务发现，独立请求上下文）
- 性能优化（共享昂贵资源，隔离轻量状态）

## 并发安全考虑

### 共享实例的线程安全

如果 tool 实例在多个 ToolsNode 中共享，必须确保线程安全：

```go
type StatefulCounterTool struct {
    name    string
    counter int
    mu      sync.Mutex  // ✅ 必需的互斥锁
}

func (t *StatefulCounterTool) InvokableRun(ctx context.Context, args string, opts ...tool.Option) (string, error) {
    t.mu.Lock()         // ✅ 加锁
    defer t.mu.Unlock() // ✅ 解锁

    t.counter++
    return fmt.Sprintf("Counter: %d", t.counter), nil
}
```

### eino 框架的并发保证

根据 `ToolsNodeConfig`：
```go
type ToolsNodeConfig struct {
    // ExecuteSequentially 决定是否顺序执行工具调用
    // true:  按顺序一个接一个执行
    // false: (默认) 并行执行
    ExecuteSequentially bool
}
```

⚠️ **注意**：即使 `ExecuteSequentially=true`，也只保证**同一个 ToolsNode 内**的工具顺序执行，不同 ToolsNode 之间仍可能并发访问共享 tool 实例。

## 设计建议

### ✅ 推荐做法

1. **默认使用独立实例**
   ```go
   // 除非明确需要共享状态，否则为每个节点创建独立实例
   tool1 := NewMyTool()
   tool2 := NewMyTool()
   ```

2. **共享实例必须并发安全**
   ```go
   type SafeTool struct {
       mu sync.Mutex  // 必需
       state map[string]any
   }
   ```

3. **通过配置管理实例策略**
   ```go
   type AppConfig struct {
       UseSharedDBConnection bool
       ToolInstanceStrategy  string // "shared" | "isolated"
   }
   ```

4. **明确命名区分**
   ```go
   sharedCounter := NewStatefulCounterTool("shared_counter")
   isolatedCounter1 := NewStatefulCounterTool("isolated_counter_1")
   isolatedCounter2 := NewStatefulCounterTool("isolated_counter_2")
   ```

### ❌ 避免的陷阱

1. **无意中共享状态**
   ```go
   // ❌ 错误：可能无意共享
   tool := NewMyTool()
   for i := 0; i < 10; i++ {
       node := createNode(tool) // 所有节点共享同一个 tool
   }

   // ✅ 正确：明确创建独立实例
   for i := 0; i < 10; i++ {
       tool := NewMyTool()
       node := createNode(tool)
   }
   ```

2. **共享实例未加锁**
   ```go
   // ❌ 错误：并发不安全
   type UnsafeTool struct {
       counter int  // 多个 goroutine 同时修改
   }

   // ✅ 正确：使用互斥锁
   type SafeTool struct {
       mu      sync.Mutex
       counter int
   }
   ```

3. **过度共享**
   ```go
   // ❌ 错误：所有东西都共享
   sharedEverything := NewComplexTool()
   // 当不需要共享时，会引入不必要的并发复杂性

   // ✅ 正确：只共享必要的资源
   sharedDB := NewDBTool()      // 确实需要共享
   cache1 := NewCacheTool()     // 不需要共享
   cache2 := NewCacheTool()
   ```

## 实际应用案例

### 案例1：多租户 SaaS 系统

```go
// 共享：数据库连接池
sharedDB := NewDBConnectionPool(config)

// 隔离：每个租户的会话状态
tenant1Session := NewSessionTool("tenant_1")
tenant2Session := NewSessionTool("tenant_2")

agent1 := createAgent(sharedDB, tenant1Session)
agent2 := createAgent(sharedDB, tenant2Session)
```

### 案例2：A/B 测试系统

```go
// 共享：实验配置服务
sharedExpConfig := NewExperimentConfig()

// 隔离：不同实验组的特征工具
featureToolA := NewFeatureTool("variant_A")
featureToolB := NewFeatureTool("variant_B")

agentA := createAgent(sharedExpConfig, featureToolA)
agentB := createAgent(sharedExpConfig, featureToolB)
```

### 案例3：流式处理管道

```go
// 共享：消息队列连接
sharedMQ := NewMessageQueue()

// 隔离：每个 worker 的处理状态
worker1State := NewProcessorTool()
worker2State := NewProcessorTool()

pipeline1 := createPipeline(sharedMQ, worker1State)
pipeline2 := createPipeline(sharedMQ, worker2State)
```

## 官方文档说明

⚠️ **注意**：eino 官方文档中**未明确说明** tool 实例管理策略。

这表明：
1. Tool 实例管理是由 **Go 语言特性自然支持**的，而非框架特意设计的功能
2. 框架层面是**中立的**：传递什么实例，就使用什么实例
3. **责任在开发者**：需要根据业务需求选择合适的实例管理策略

## 总结

| 维度 | 共享实例 | 独立实例 |
|------|---------|---------|
| **状态** | 跨节点共享 | 完全隔离 |
| **并发安全** | ⚠️ 需要开发者保证 | ✅ 天然安全 |
| **适用场景** | 资源池、全局计数器 | 会话管理、独立状态 |
| **复杂度** | 高（需处理并发） | 低（无并发问题） |
| **性能** | 节省资源 | 可能冗余 |

**最佳实践**：
- 🎯 **默认使用独立实例**，简单安全
- 🎯 **需要共享时显式标注**，并确保线程安全
- 🎯 **通过配置管理策略**，便于切换和测试
- 🎯 **使用混合模式**，共享昂贵资源，隔离轻量状态

---

**测试代码位置**：
- `eino/tool_instance_test.go` - 基础实例共享测试
- `eino/tool_instance_in_graph_test.go` - Graph 中的应用场景测试

**研究日期**：2025-11-10
