# Stars (群星)

Stars 是 Aster 框架中的多 Agent 协作组件，负责管理多个 Agent 之间的协作、通信和任务执行。

## 概念

**Stars (群星)** 就像夜空中的群星一样，多个 Agent 组成一个协作单元，共同完成任务。Stars 提供：

- **动态成员管理**：Agent 可以随时加入或离开
- **角色分工**：Leader 和 Worker 两种角色
- **消息通信**：广播和点对点消息
- **任务执行**：Leader-Worker 协作模式

## 核心概念

### 角色 (Role)

Stars 支持两种角色：

- **Leader (领导者)**：负责协调和决策，接收任务并分配给 Workers
- **Worker (工作者)**：负责执行具体任务

### 成员 (Member)

每个成员包含：
- `AgentID`: Agent 的唯一标识
- `Role`: 角色（Leader 或 Worker）
- `Tags`: 能力标签（可选）

### 消息 (Message)

消息包含：
- `From`: 发送者 Agent ID
- `To`: 接收者 Agent ID（空表示广播）
- `Text`: 消息内容
- `Time`: 发送时间

## 核心 API

### 创建 Stars

```go
stars := stars.New(cosmos, "DevTeam")
```

### 添加成员

```go
// 添加 Leader
stars.Join("leader-1", stars.RoleLeader)

// 添加 Workers
stars.Join("worker-1", stars.RoleWorker)
stars.Join("worker-2", stars.RoleWorker)
```

### 移除成员

```go
stars.Leave("worker-1")
```

### 查看成员

```go
members := stars.Members()
for _, m := range members {
    fmt.Printf("%s (%s)\n", m.AgentID, m.Role)
}
```

### 发送消息

```go
// 点对点消息
stars.Send(ctx, "leader-1", "worker-1", "请处理任务 A")

// 广播消息
stars.Broadcast(ctx, "开始新的迭代")
```

### 执行任务

```go
// 使用 Leader-Worker 模式执行任务
for event, err := range stars.Run(ctx, "开发用户认证功能") {
    if err != nil {
        log.Printf("Error: %v", err)
        continue
    }

    fmt.Printf("[%s] %s: %s\n",
        event.AgentID,
        event.Type,
        event.Content)
}
```

### 查看历史

```go
history := stars.History()
for _, msg := range history {
    if msg.To == "" {
        fmt.Printf("[广播] %s: %s\n", msg.From, msg.Text)
    } else {
        fmt.Printf("%s → %s: %s\n", msg.From, msg.To, msg.Text)
    }
}
```

## 使用场景

### 1. 开发团队协作

```go
// 创建开发团队
devTeam := stars.New(cosmos, "DevTeam")

// 添加成员
devTeam.Join("tech-lead", stars.RoleLeader)
devTeam.Join("frontend-dev", stars.RoleWorker)
devTeam.Join("backend-dev", stars.RoleWorker)
devTeam.Join("qa-engineer", stars.RoleWorker)

// 执行开发任务
for event := range devTeam.Run(ctx, "开发新功能") {
    // 处理事件
}
```

### 2. 数据处理流水线

```go
// 创建数据处理团队
pipeline := stars.New(cosmos, "DataPipeline")

pipeline.Join("coordinator", stars.RoleLeader)
pipeline.Join("collector", stars.RoleWorker)
pipeline.Join("processor", stars.RoleWorker)
pipeline.Join("analyzer", stars.RoleWorker)

// 协调数据处理
pipeline.Run(ctx, "处理今日数据")
```

### 3. 客服团队

```go
// 创建客服团队
support := stars.New(cosmos, "SupportTeam")

support.Join("supervisor", stars.RoleLeader)
support.Join("agent-1", stars.RoleWorker)
support.Join("agent-2", stars.RoleWorker)
support.Join("agent-3", stars.RoleWorker)

// 处理客户请求
support.Run(ctx, "处理客户咨询")
```

## 协作模式

### Leader-Worker 模式

Stars 使用 Leader-Worker 模式进行任务执行：

1. **任务分配**：Leader 接收任务
2. **任务分解**：Leader 将任务分解为子任务
3. **任务执行**：Workers 执行子任务
4. **结果汇总**：Leader 汇总结果

```
┌─────────┐
│  Task   │
└────┬────┘
     │
     ↓
┌─────────┐
│ Leader  │ ← 接收任务，分解并分配
└────┬────┘
     │
     ├──→ Worker 1 ← 执行子任务 A
     ├──→ Worker 2 ← 执行子任务 B
     └──→ Worker 3 ← 执行子任务 C
     │
     ↓
┌─────────┐
│ Result  │ ← Leader 汇总结果
└─────────┘
```

## 与 Room 的区别

Stars 是 Room 的增强版本，主要改进：

| Room | Stars |
|------|-------|
| Room | Stars (群星) |
| 无角色系统 | Leader/Worker 角色 |
| 简单消息路由 | 完整的协作模式 |
| 无任务执行 | Run() 方法执行任务 |

## 示例

查看 [examples/stars](../../examples/stars) 目录获取完整示例：

- `basic/`: 基本使用示例
- `dynamic/`: 动态成员管理示例

## 最佳实践

1. **明确角色分工**：确保有且只有一个 Leader
   ```go
   // 先添加 Leader
   stars.Join("leader-1", stars.RoleLeader)

   // 再添加 Workers
   stars.Join("worker-1", stars.RoleWorker)
   stars.Join("worker-2", stars.RoleWorker)
   ```

2. **使用有意义的名称**：给 Stars 和 Agent 起有意义的名称
   ```go
   devTeam := stars.New(cosmos, "DevTeam")
   supportTeam := stars.New(cosmos, "SupportTeam")
   ```

3. **动态调整成员**：根据负载动态添加或移除 Workers
   ```go
   // 负载高时添加 Worker
   if load > threshold {
       stars.Join(newWorkerID, stars.RoleWorker)
   }

   // 负载低时移除 Worker
   if load < threshold {
       stars.Leave(idleWorkerID)
   }
   ```

4. **消息历史管理**：定期清理或持久化消息历史
   ```go
   // Stars 自动限制历史记录为最近 100 条
   history := stars.History()
   ```

5. **错误处理**：处理 Run() 返回的错误
   ```go
   for event, err := range stars.Run(ctx, task) {
       if err != nil {
           log.Printf("Error: %v", err)
           // 决定是否继续或中断
           continue
       }
       // 处理事件
   }
   ```

6. **异步消息**：消息发送是异步的，不会阻塞
   ```go
   // 发送消息后立即返回
   stars.Send(ctx, from, to, message)
   stars.Broadcast(ctx, message)
   ```

## 未来扩展

Stars 的设计支持未来扩展：

- **更多角色类型**：Coordinator、Observer、Specialist 等
- **更多协作模式**：Democratic（民主投票）、Swarm（集群自组织）等
- **消息持久化**：将消息历史持久化到数据库
- **更丰富的监控**：成员状态、任务进度等

当前版本保持简单实用，专注于核心功能。
