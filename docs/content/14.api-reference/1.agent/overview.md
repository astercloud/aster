---
title: Agent API
description: Agent核心API完整参考文档
---

# Agent API 参考

本文档提供 Agent 核心 API 的完整参考，包括创建、配置、消息处理、事件订阅等功能。

## 目录

- [创建 Agent](#创建-agent)
- [消息处理](#消息处理)
- [事件订阅](#事件订阅)
- [状态管理](#状态管理)
- [资源管理](#资源管理)
- [类型定义](#类型定义)

---

## 创建 Agent

### agent.Create

创建新的 Agent 实例。

```go
func Create(ctx context.Context, config *types.AgentConfig, deps *Dependencies) (*Agent, error)
```

**参数**：

- `ctx`: 上下文，用于控制 Agent 生命周期
- `config`: Agent 配置，包括模型、工具、中间件等
- `deps`: 依赖注入，提供工具注册表、Provider 工厂等

**返回**：

- `*Agent`: Agent 实例
- `error`: 创建失败时返回错误

**示例**：

```go
import (
    "context"
    "os"
    "github.com/astercloud/aster/pkg/agent"
    "github.com/astercloud/aster/pkg/types"
)

// 创建 Agent
ag, err := agent.Create(ctx, &types.AgentConfig{
    TemplateID: "assistant",
    ModelConfig: &types.ModelConfig{
        Provider: "anthropic",
        Model:    "claude-sonnet-4-5",
        APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
    },
    Tools: []string{"filesystem", "bash"},
    Middlewares: []string{"filesystem", "summarization"},
}, deps)
if err != nil {
    log.Fatal(err)
}
defer ag.Close()
```

**配置说明**：

- `TemplateID`: Agent 模板 ID（如 "assistant"、"code-generator"）
- `ModelConfig`: 模型配置，支持 Anthropic、OpenAI、DeepSeek 等
- `Tools`: 启用的工具列表
- `Middlewares`: 中间件列表，按优先级执行
- `Sandbox`: 沙箱配置（可选，默认本地沙箱）
- `SystemPrompt`: 自定义系统提示词（可选）

---

## 消息处理

### agent.Chat

同步对话，阻塞等待完整响应。适合简单的请求-响应场景。

```go
func (a *Agent) Chat(ctx context.Context, text string) (*types.CompleteResult, error)
```

**参数**：

- `ctx`: 上下文
- `text`: 用户消息文本

**返回**：

- `*types.CompleteResult`: 对话结果，包含响应文本、停止原因、Token 使用量
- `error`: 错误信息

**示例**：

```go
result, err := ag.Chat(ctx, "创建一个 hello.txt 文件，内容为 Hello World")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("响应: %s\n", result.Message.Content)
fmt.Printf("Token 使用: %d\n", result.Usage.TotalTokens)
```

**CompleteResult 结构**：

```go
type CompleteResult struct {
    Message    Message    // 完整的响应消息
    StopReason string     // 停止原因: "end_turn", "tool_use", "max_tokens"
    Usage      TokenUsage // Token 使用量
}
```

---

### agent.Send

异步发送消息，不等待响应。适合需要实时处理事件的场景。

```go
func (a *Agent) Send(ctx context.Context, text string) error
```

**参数**：

- `ctx`: 上下文
- `text`: 用户消息文本

**返回**：

- `error`: 发送失败时返回错误

**示例**：

```go
// 1. 订阅事件通道
eventCh := ag.Subscribe([]types.AgentChannel{
    types.ChannelProgress,
    types.ChannelControl,
}, nil)

// 2. 处理事件
go func() {
    for envelope := range eventCh {
        switch envelope.Event.Type {
        case types.EventTypeTextDelta:
            fmt.Print(envelope.Event.TextDelta)
        case types.EventTypeToolCallRequest:
            // 处理工具调用请求
        }
    }
}()

// 3. 发送消息
err := ag.Send(ctx, "分析当前目录的文件结构")
if err != nil {
    log.Fatal(err)
}
```

**适用场景**：

- 需要实时显示流式输出
- 需要人工审批工具调用
- 需要监控 Agent 执行状态

---

### agent.Stream

流式对话，返回 Reader。适合需要逐步处理响应的场景。

```go
func (a *Agent) Stream(ctx context.Context, message string, opts ...Option) *stream.Reader[*session.Event]
```

**参数**：

- `ctx`: 上下文
- `message`: 用户消息文本
- `opts`: 可选配置

**返回**：

- `*stream.Reader[*session.Event]`: 事件 Reader

**示例**：

```go
reader := ag.Stream(ctx, "介绍一下 Go 语言")
for {
    event, err := reader.Recv()
    if err != nil {
        if errors.Is(err, io.EOF) {
            break
        }
        log.Printf("错误: %v", err)
        break
    }

    switch event.Type {
    case "text_delta":
        fmt.Print(event.TextDelta)
    case "tool_call":
        fmt.Printf("\n[工具调用] %s\n", event.ToolCall.Name)
    }
}
```

---

## 事件订阅

### agent.Subscribe

订阅 Agent 的事件通道。

```go
func (a *Agent) Subscribe(channels []types.AgentChannel, opts *types.SubscribeOptions) <-chan types.AgentEventEnvelope
```

**参数**：

- `channels`: 要订阅的通道列表
  - `types.ChannelProgress`: 进度事件（文本流、工具执行）
  - `types.ChannelControl`: 控制事件（工具审批请求）
  - `types.ChannelMonitor`: 监控事件（内部状态变化）
- `opts`: 订阅选项（可选）
  - `Filter`: 事件过滤器

**返回**：

- `<-chan types.AgentEventEnvelope`: 事件通道

**示例**：

```go
// 订阅进度和控制事件
eventCh := ag.Subscribe([]types.AgentChannel{
    types.ChannelProgress,
    types.ChannelControl,
}, &types.SubscribeOptions{
    Filter: &types.EventFilter{
        EventTypes: []types.EventType{
            types.EventTypeTextDelta,
            types.EventTypeToolCallRequest,
        },
    },
})

for envelope := range eventCh {
    fmt.Printf("[%s] %s: %v\n",
        envelope.Channel,
        envelope.Event.Type,
        envelope.Event)
}
```

**事件类型**：

| 事件类型                   | 通道     | 说明                           |
| -------------------------- | -------- | ------------------------------ |
| `EventTypeTextDelta`       | Progress | 流式文本增量                   |
| `EventTypeTextDone`        | Progress | 文本流结束                     |
| `EventTypeToolCallRequest` | Control  | 工具调用请求                   |
| `EventTypeToolCallResult`  | Progress | 工具执行结果                   |
| `EventTypeGovernance`      | Monitor  | 治理事件（权限检查、速率限制） |
| `EventTypeError`           | Monitor  | 错误事件                       |

---

### agent.Unsubscribe

取消订阅事件通道。

```go
func (a *Agent) Unsubscribe(ch <-chan types.AgentEventEnvelope)
```

**参数**：

- `ch`: 要取消的事件通道

**示例**：

```go
eventCh := ag.Subscribe([]types.AgentChannel{types.ChannelProgress}, nil)

// 使用事件通道...

// 取消订阅
ag.Unsubscribe(eventCh)
```

---

## 状态管理

### agent.ID

获取 Agent 实例的唯一标识符。

```go
func (a *Agent) ID() string
```

**返回**：

- `string`: Agent ID

**示例**：

```go
fmt.Printf("Agent ID: %s\n", ag.ID())
```

---

### agent.Status

获取 Agent 当前状态。

```go
func (a *Agent) Status() *types.AgentStatus
```

**返回**：

- `*types.AgentStatus`: Agent 状态信息

**示例**：

```go
status := ag.Status()
fmt.Printf("状态: %s\n", status.State)
fmt.Printf("消息数: %d\n", status.MessageCount)
fmt.Printf("步数: %d\n", status.StepCount)
fmt.Printf("运行时间: %s\n", status.Uptime)
```

**AgentStatus 结构**：

```go
type AgentStatus struct {
    State        AgentRuntimeState // idle, running, waiting, error
    MessageCount int               // 消息数量
    StepCount    int               // 执行步数
    Uptime       time.Duration     // 运行时间
    LastActivity time.Time         // 最后活动时间
}
```

---

## 资源管理

### agent.Close

关闭 Agent，释放所有资源。

```go
func (a *Agent) Close() error
```

**返回**：

- `error`: 关闭失败时返回错误

**示例**：

```go
// 使用 defer 确保资源释放
defer ag.Close()

// 或手动关闭
if err := ag.Close(); err != nil {
    log.Printf("关闭 Agent 失败: %v", err)
}
```

**关闭行为**：

- 停止所有正在执行的任务
- 关闭所有事件通道
- 释放 Provider 连接
- 清理沙箱资源
- 保存持久化状态（如果配置了 Store）

---

## 类型定义

### AgentConfig

Agent 配置结构。

```go
type AgentConfig struct {
    // 基础配置
    AgentID      string            // Agent ID（可选，自动生成）
    TemplateID   string            // 模板 ID（必需）

    // 模型配置
    ModelConfig  *ModelConfig      // 模型配置

    // 功能配置
    Tools        []string          // 工具列表
    Middlewares  []string          // 中间件列表
    Sandbox      *SandboxConfig    // 沙箱配置

    // 提示词
    SystemPrompt string            // 系统提示词

    // 路由配置（可选）
    RoutingProfile string          // 路由配置：cost, balanced, performance

    // 元数据
    Metadata     map[string]any    // 自定义元数据
}
```

---

### ModelConfig

模型配置结构。

```go
type ModelConfig struct {
    Provider    string            // Provider 名称: anthropic, openai, deepseek
    Model       string            // 模型名称
    APIKey      string            // API Key
    BaseURL     string            // 自定义 API 端点（可选）

    // 生成参数
    Temperature float64           // 温度参数 (0.0-1.0)
    MaxTokens   int               // 最大 Token 数
    TopP        float64           // Top-P 采样
}
```

---

### Dependencies

依赖注入结构。

```go
type Dependencies struct {
    ToolRegistry     *tools.Registry          // 工具注册表（必需）
    SandboxFactory   sandbox.Factory          // 沙箱工厂（必需）
    ProviderFactory  provider.Factory         // Provider 工厂（必需）
    Store            store.Store              // 持久化存储（可选）
    TemplateRegistry *TemplateRegistry        // 模板注册表（可选）
    Router           router.Router            // 模型路由器（可选）
}
```

**创建依赖示例**：

```go
deps := &agent.Dependencies{
    ToolRegistry: tools.NewRegistry(),
    SandboxFactory: sandbox.NewFactory(),
    ProviderFactory: provider.NewMultiProviderFactory(),
    Store: store.NewJSONStore(".aster"),
    TemplateRegistry: agent.NewTemplateRegistry(),
}
```

---

### Message

消息结构。

```go
type Message struct {
    Role          MessageRole      // user, assistant, system
    Content       string           // 文本内容
    ContentBlocks []ContentBlock   // 多模态内容块（可选）
    ToolCalls     []ToolCall       // 工具调用（可选）
    ToolResults   []ToolResult     // 工具结果（可选）
}
```

---

### AgentEventEnvelope

事件封装结构。

```go
type AgentEventEnvelope struct {
    ID        string           // 事件 ID
    AgentID   string           // Agent ID
    Channel   AgentChannel     // 通道类型
    Event     AgentEvent       // 事件数据
    Timestamp time.Time        // 时间戳
}
```

---

### AgentEvent

事件数据结构。

```go
type AgentEvent struct {
    Type          EventType        // 事件类型
    TextDelta     string           // 文本增量
    ToolCall      *ToolCall        // 工具调用
    ToolResult    *ToolResult      // 工具结果
    Error         error            // 错误信息
    Metadata      map[string]any   // 元数据
}
```

---

## 最佳实践

### 1. 资源管理

始终使用 `defer` 确保资源释放：

```go
ag, err := agent.Create(ctx, config, deps)
if err != nil {
    return err
}
defer ag.Close()
```

### 2. 错误处理

检查所有错误，特别是在生产环境：

```go
result, err := ag.Chat(ctx, message)
if err != nil {
    log.Printf("Chat 失败: %v", err)
    // 实现重试逻辑或回退方案
    return err
}
```

### 3. 上下文控制

使用上下文控制超时和取消：

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := ag.Chat(ctx, message)
```

### 4. 事件处理

在独立的 goroutine 中处理事件：

```go
eventCh := ag.Subscribe([]types.AgentChannel{types.ChannelProgress}, nil)

go func() {
    defer ag.Unsubscribe(eventCh)
    for envelope := range eventCh {
        handleEvent(envelope)
    }
}()
```

### 5. 并发控制

避免并发调用 Agent 的方法，Agent 不是线程安全的：

```go
// ❌ 错误：并发调用
go ag.Send(ctx, "message1")
go ag.Send(ctx, "message2")

// ✅ 正确：顺序调用或使用锁
mu.Lock()
ag.Send(ctx, "message1")
mu.Unlock()
```

---

## 相关资源

- [Provider API 文档](./2.provider-api.md)
- [Middleware API 文档](./3.middleware-api.md)
- [Tools API 文档](./4.tools-api.md)
- [完整 API 文档 (pkg.go.dev)](https://pkg.go.dev/github.com/astercloud/aster)
- [GitHub 仓库](https://github.com/astercloud/aster)
