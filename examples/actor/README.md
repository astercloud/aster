# Actor 模型示例

本示例演示如何使用 Aster 的轻量级 Actor 系统实现并发和多 Agent 协作。

## 概述

Actor 模型是一种并发计算模型，每个 Actor 是独立的计算单元，通过异步消息传递进行通信。本示例展示了 `pkg/actor` 包的核心功能：

- **消息传递** - Tell（异步）和 Request（同步）两种模式
- **监督策略** - 自动故障恢复（OneForOne、AllForOne）
- **父子关系** - Actor 层级结构和生命周期管理
- **并发安全** - 每个 Actor 内部状态线程安全

## 架构

```
┌─────────────────────────────────────────────────────────────────┐
│                      Actor System                               │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │   Actor-1   │  │   Actor-2   │  │   Actor-3   │   ...       │
│  │   (PID)     │  │   (PID)     │  │   (PID)     │             │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘             │
│         │                │                │                     │
│         └────────────────┼────────────────┘                     │
│                          │                                      │
│                    ┌─────▼─────┐                                │
│                    │  Mailbox  │                                │
│                    │ Dispatcher│                                │
│                    └───────────┘                                │
└─────────────────────────────────────────────────────────────────┘
```

## 演示命令

| 命令 | 说明 |
|------|------|
| `go run . basic` | 基础 Ping-Pong 消息传递 |
| `go run . counter` | 并发安全计数器 |
| `go run . supervisor` | 监督者故障恢复策略 |
| `go run . pipeline` | 流水线处理模式 |
| `go run . broadcast` | 广播消息给多个订阅者 |

## 运行示例

```bash
# 查看帮助
go run . --help

# 基础演示 - Ping/Pong 消息传递
go run . basic

# 并发计数器 - 验证线程安全
go run . counter

# 监督者策略 - 故障自动恢复
go run . supervisor

# 流水线处理 - 多阶段数据处理
go run . pipeline

# 广播消息 - 一对多通信
go run . broadcast

# 运行测试
go test -v ./...
```

## 核心概念

### 1. Actor 定义

```go
type MyActor struct {
    state int
}

func (a *MyActor) Receive(ctx *actor.Context, msg actor.Message) {
    switch m := msg.(type) {
    case *actor.Started:
        // Actor 启动时调用
    case *MyMessage:
        // 处理业务消息
        ctx.Reply(&MyResponse{})
    case *actor.Stopping:
        // Actor 停止前调用
    }
}
```

### 2. 消息类型

```go
type MyMessage struct {
    Data string
}

// 必须实现 Kind() 方法
func (m *MyMessage) Kind() string { return "my.message" }
```

### 3. 创建和使用 Actor

```go
// 创建 Actor 系统
system := actor.NewSystem("my-system")
defer system.Shutdown()

// 创建 Actor
myActor := &MyActor{}
pid := system.Spawn(myActor, "my-actor")

// 发送消息（异步）
pid.Tell(&MyMessage{Data: "hello"})

// 请求响应（同步）
resp, err := pid.Request(&MyMessage{Data: "hello"}, 5*time.Second)
```

### 4. 监督策略

```go
// 使用 OneForOne 策略（只重启失败的 Actor）
props := &actor.Props{
    Name:               "my-actor",
    MailboxSize:        100,
    SupervisorStrategy: actor.NewOneForOneStrategy(
        3,              // 最大重启次数
        time.Minute,    // 时间窗口
        actor.DefaultDecider,
    ),
}
pid := system.SpawnWithProps(myActor, props)
```

## 与 Agent 集成

Actor 系统可以与现有 Agent 无缝集成：

```go
import (
    "github.com/astercloud/aster/pkg/actor"
    "github.com/astercloud/aster/pkg/agent"
)

// 创建 Agent 并包装为 Actor
ag, _ := agent.Create(ctx, config, deps)
agentActor := agent.NewAgentActor(ag)
pid := system.Spawn(agentActor, "agent-1")

// 通过消息发送对话请求
pid.Tell(&agent.ChatMsg{
    Text:    "你好",
    Ctx:     ctx,
    ReplyTo: replyCh,
})
```

## 使用场景

| 场景 | 适用度 | 说明 |
|------|--------|------|
| 多 Agent 协作 | ⭐⭐⭐⭐⭐ | 每个 Agent 是独立 Actor |
| 并发任务处理 | ⭐⭐⭐⭐⭐ | 消息驱动，无锁并发 |
| 故障隔离 | ⭐⭐⭐⭐⭐ | Actor 故障不影响其他 Actor |
| 流水线处理 | ⭐⭐⭐⭐ | 多阶段异步处理 |
| 发布订阅 | ⭐⭐⭐⭐ | 广播消息给多个订阅者 |

## 性能特点

- **消息吞吐**: 单系统 100K+ msg/s
- **Actor 数量**: 支持 10K+ 并发 Actor
- **内存占用**: 每个 Actor 约 1-2KB
- **延迟**: 本地消息传递 < 1μs
