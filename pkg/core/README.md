# Core Package

Core 包提供 Aster 框架的核心抽象：Agent 池管理和多 Agent 协作。

## 组件

### Pool - Agent 生命周期管理

Pool 负责管理多个 Agent 的生命周期，提供创建、获取、删除和列表功能。

```go
// 创建 Pool
pool := core.NewPool(&core.PoolOptions{
    Dependencies: deps,
    MaxAgents:    50,
})
defer pool.Shutdown()

// 创建 Agent
config := &types.AgentConfig{
    AgentID:    "my-agent",
    TemplateID: "assistant",
    // ...
}
agent, err := pool.Create(ctx, config)

// 获取 Agent
agent, exists := pool.Get("my-agent")

// 列出所有 Agent
allAgents := pool.List("")

// 按前缀过滤
workers := pool.List("worker-")

// 移除 Agent
pool.Remove("my-agent")
```

### Room - 多 Agent 协作空间

Room 提供多个 Agent 之间的消息路由、广播和点对点通信功能。

```go
// 创建 Room
room := core.NewRoom(pool)

// 添加成员
room.Join("alice", "agent-1")
room.Join("bob", "agent-2")

// 发送消息（支持 @mention）
room.Say(ctx, "alice", "Hello @bob!")

// 广播消息
room.Broadcast(ctx, "Meeting starts now")

// 点对点消息
room.SendTo(ctx, "alice", "bob", "Private message")

// 查看成员
members := room.GetMembers()

// 查看历史
history := room.GetHistory()
```

## 使用场景

### 1. 多租户系统

```go
pool := core.NewPool(&core.PoolOptions{
    Dependencies: deps,
    MaxAgents:    1000,
})

// 为每个用户创建独立的 Agent
for _, userID := range users {
    config := &types.AgentConfig{
        AgentID:    fmt.Sprintf("user-%s", userID),
        TemplateID: "assistant",
    }
    pool.Create(ctx, config)
}
```

### 2. 团队协作

```go
// 创建团队 Room
team := core.NewRoom(pool)
team.Join("leader", "leader-1")
team.Join("dev1", "developer-1")
team.Join("dev2", "developer-2")

// 团队沟通
team.Say(ctx, "leader", "Let's start the sprint planning")
team.Say(ctx, "dev1", "@leader I have a question")
```

### 3. 任务队列

```go
// 创建 Worker Pool
for i := 0; i < 10; i++ {
    config := &types.AgentConfig{
        AgentID:    fmt.Sprintf("worker-%d", i),
        TemplateID: "worker",
    }
    pool.Create(ctx, config)
}

// 分配任务
workers := pool.List("worker-")
for _, workerID := range workers {
    agent, _ := pool.Get(workerID)
    agent.Send(ctx, task)
}
```

## 设计原则

1. **简洁性**：Pool 和 Room 提供最小但完整的功能集
2. **并发安全**：所有操作都是线程安全的
3. **灵活性**：支持动态添加/移除成员
4. **可扩展性**：易于集成到更高层的抽象（如 AsterOS）

## 与其他包的关系

- `pkg/agent`: Pool 管理 Agent 实例
- `pkg/asteros`: AsterOS 使用 Pool 作为核心组件
- `pkg/types`: 使用类型定义进行配置
