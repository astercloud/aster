# Cosmos (宇宙)

Cosmos 是 Aster 框架中的 Agent 生命周期管理器，负责创建、获取、删除和管理所有 Agent 实例。

## 概念

**Cosmos (宇宙)** 就像宇宙包容万物一样，Cosmos 管理着所有的 Agent 实例。它提供：

- **统一的生命周期管理**：创建、获取、删除 Agent
- **资源管理**：统一的依赖注入和资源分配
- **容量控制**：限制最大 Agent 数量
- **并发安全**：支持多 goroutine 并发访问

## 核心 API

### 创建 Cosmos

```go
cosmos := cosmos.New(&cosmos.Options{
    Dependencies: deps,
    MaxAgents:    50,  // 最大 Agent 数量
})
defer cosmos.Shutdown()
```

### 创建 Agent

```go
config := &types.AgentConfig{
    AgentID:    "my-agent",
    TemplateID: "assistant",
    ModelConfig: &types.ModelConfig{
        Provider: "anthropic",
        Model:    "claude-sonnet-4-5",
        APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
    },
}

agent, err := cosmos.Create(ctx, config)
```

### 获取 Agent

```go
agent, exists := cosmos.Get("my-agent")
if !exists {
    log.Fatal("Agent not found")
}
```

### 列出 Agent

```go
// 列出所有 Agent
allAgents := cosmos.List("")

// 按前缀过滤
userAgents := cosmos.List("user-")
```

### 获取状态

```go
status, err := cosmos.Status("my-agent")
fmt.Printf("State: %s, Step: %d\n", status.State, status.Step)
```

### 遍历 Agent

```go
cosmos.ForEach(func(agentID string, ag *agent.Agent) error {
    fmt.Printf("Agent: %s\n", agentID)
    return nil
})
```

### 移除 Agent

```go
// 从 Cosmos 中移除（不删除存储）
cosmos.Remove("my-agent")

// 删除 Agent（包括存储）
cosmos.Delete(ctx, "my-agent")
```

### 关闭 Cosmos

```go
cosmos.Shutdown()  // 关闭所有 Agent
```

## 使用场景

### 1. 多租户系统

```go
cosmos := cosmos.New(&cosmos.Options{
    Dependencies: deps,
    MaxAgents:    1000,
})

// 为每个用户创建独立的 Agent
for _, userID := range users {
    config := &types.AgentConfig{
        AgentID:    fmt.Sprintf("user-%s", userID),
        TemplateID: "assistant",
        // ...
    }
    cosmos.Create(ctx, config)
}
```

### 2. 任务队列

```go
// 创建 Worker Agent 池
for i := 0; i < 10; i++ {
    config := &types.AgentConfig{
        AgentID:    fmt.Sprintf("worker-%d", i),
        TemplateID: "worker",
        // ...
    }
    cosmos.Create(ctx, config)
}

// 分配任务给 Worker
workers := cosmos.List("worker-")
for _, workerID := range workers {
    agent, _ := cosmos.Get(workerID)
    agent.Send(ctx, task)
}
```

### 3. 会话管理

```go
// 恢复用户会话
agent, err := cosmos.Resume(ctx, sessionID, config)
if err != nil {
    // 创建新会话
    agent, err = cosmos.Create(ctx, config)
}
```

## 与 Pool 的区别

Cosmos 是 Pool 的重命名版本，功能完全相同，只是名称更符合 Aster 的星空主题：

| Pool | Cosmos |
|------|--------|
| Pool | Cosmos (宇宙) |
| 通用名称 | 星空主题 |

## 示例

查看 [examples/cosmos](../../examples/cosmos) 目录获取完整示例。

## 最佳实践

1. **使用 defer 关闭**：确保在程序退出时关闭 Cosmos
   ```go
   cosmos := cosmos.New(opts)
   defer cosmos.Shutdown()
   ```

2. **设置合理的容量**：根据系统资源设置 MaxAgents
   ```go
   MaxAgents: 100,  // 根据内存和 CPU 调整
   ```

3. **使用前缀组织**：使用前缀来组织不同类型的 Agent
   ```go
   "user-123"    // 用户 Agent
   "worker-1"    // Worker Agent
   "admin-001"   // 管理员 Agent
   ```

4. **错误处理**：始终检查错误
   ```go
   agent, err := cosmos.Create(ctx, config)
   if err != nil {
       log.Printf("Failed to create agent: %v", err)
       return err
   }
   ```

5. **并发安全**：Cosmos 是并发安全的，可以在多个 goroutine 中使用
   ```go
   var wg sync.WaitGroup
   for i := 0; i < 10; i++ {
       wg.Add(1)
       go func(idx int) {
           defer wg.Done()
           cosmos.Create(ctx, config)
       }(i)
   }
   wg.Wait()
   ```
