---
title: Middleware API
description: Middleware接口完整参考文档
---

# Middleware API 参考

本文档提供 Middleware 核心 API 的完整参考，涵盖中间件接口、内置中间件、自定义中间件开发。

## 目录

- [接口概览](#接口概览)
- [使用中间件](#使用中间件)
- [内置中间件](#内置中间件)
- [自定义中间件](#自定义中间件)
- [类型定义](#类型定义)

---

## 接口概览

Middleware 采用洋葱模型，支持对模型调用和工具调用的拦截处理：

```go
type Middleware interface {
    // 基础信息
    Name() string
    Priority() int

    // 工具注入
    Tools() []tools.Tool

    // 拦截器
    WrapModelCall(ctx context.Context, req *ModelRequest, handler ModelCallHandler) (*ModelResponse, error)
    WrapToolCall(ctx context.Context, req *ToolCallRequest, handler ToolCallHandler) (*ToolCallResponse, error)

    // 生命周期
    OnAgentStart(ctx context.Context, agentID string) error
    OnAgentStop(ctx context.Context, agentID string) error
}
```

**洋葱模型执行顺序**：

```
请求 → Middleware A → Middleware B → Middleware C → 核心处理 → Middleware C → Middleware B → Middleware A → 响应
```

优先级数值越小，越早执行（越外层）。

---

## 使用中间件

### 在 Agent 中启用

```go
ag, err := agent.Create(ctx, &types.AgentConfig{
    TemplateID: "assistant",
    ModelConfig: &types.ModelConfig{
        Provider: "anthropic",
        Model:    "claude-sonnet-4-5",
        APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
    },
    Middlewares: []string{
        "filesystem",      // 文件系统中间件
        "summarization",   // 自动总结中间件
        "subagent",        // 子 Agent 中间件
    },
}, deps)
```

### 注册自定义中间件

```go
// 创建中间件注册表
registry := middleware.NewRegistry()

// 注册内置中间件
middleware.RegisterBuiltin(registry)

// 注册自定义中间件
registry.Register("custom", func(config *types.MiddlewareConfig) (middleware.Middleware, error) {
    return NewCustomMiddleware(config), nil
})

// 在依赖中使用
deps := &agent.Dependencies{
    MiddlewareRegistry: registry,
    // ... 其他依赖
}
```

---

## 内置中间件

### FilesystemMiddleware

文件系统操作和内存管理中间件。

**优先级**：`100`

**功能**：

- 自动记录文件操作历史
- 提供文件系统快照
- 管理文件上下文

**启用**：

```go
Middlewares: []string{"filesystem"}
```

**配置**：

```go
MiddlewareConfigs: map[string]*types.MiddlewareConfig{
    "filesystem": {
        Settings: map[string]interface{}{
            "max_files_in_context": 10,
            "track_history": true,
        },
    },
}
```

---

### SummarizationMiddleware

自动上下文总结中间件。

**优先级**：`40`

**功能**：

- 当对话历史超过阈值时自动总结
- 保留最近消息，总结旧消息
- 支持自定义总结策略

**启用**：

```go
Middlewares: []string{"summarization"}
```

**配置**：

```go
MiddlewareConfigs: map[string]*types.MiddlewareConfig{
    "summarization": {
        Settings: map[string]interface{}{
            "message_threshold": 20,  // 超过 20 条消息时触发
            "keep_recent": 5,         // 保留最近 5 条消息
        },
    },
}
```

**示例**：

```go
// 对话历史超过阈值时自动触发总结
ag.Chat(ctx, "第 25 条消息")
// 自动总结前 20 条，保留最近 5 条
```

---

### SubAgentMiddleware

子 Agent 任务委派中间件。

**优先级**：`200`

**功能**：

- 提供 `subagent_delegate` 工具
- 将复杂任务委派给子 Agent
- 支持子 Agent 隔离执行

**启用**：

```go
Middlewares: []string{"subagent"}
```

**使用**：

```go
// Agent 会自动获得 subagent_delegate 工具
ag.Chat(ctx, "创建一个复杂的文件处理系统")
// Agent 可以决定将任务委派给子 Agent
```

---

### AgentMemoryMiddleware

Agent 长期记忆中间件。

**优先级**：`50`

**功能**：

- 自动保存对话历史到记忆系统
- 在新对话中检索相关历史
- 支持向量数据库（Qdrant）

**启用**：

```go
Middlewares: []string{"agent_memory"}
```

**配置**：

```go
MiddlewareConfigs: map[string]*types.MiddlewareConfig{
    "agent_memory": {
        Settings: map[string]interface{}{
            "qdrant_url": "http://localhost:6333",
            "collection": "agent_memories",
            "auto_save": true,
            "retrieve_top_k": 5,
        },
    },
}
```

---

## 自定义中间件

### 基础模板

```go
package mymiddleware

import (
    "context"
    "github.com/astercloud/aster/pkg/middleware"
    "github.com/astercloud/aster/pkg/tools"
)

type CustomMiddleware struct {
    *middleware.BaseMiddleware
    // 自定义字段
}

func NewCustomMiddleware(config *types.MiddlewareConfig) *CustomMiddleware {
    return &CustomMiddleware{
        BaseMiddleware: middleware.NewBaseMiddleware("custom", 500),
    }
}

// 覆盖需要的方法
func (m *CustomMiddleware) WrapModelCall(
    ctx context.Context,
    req *middleware.ModelRequest,
    handler middleware.ModelCallHandler,
) (*middleware.ModelResponse, error) {
    // 前置处理
    log.Printf("模型调用前: %d 条消息", len(req.Messages))

    // 调用下一层
    resp, err := handler(ctx, req)
    if err != nil {
        return nil, err
    }

    // 后置处理
    log.Printf("模型调用后: %s", resp.Message.Content)

    return resp, nil
}
```

### 注入工具

```go
func (m *CustomMiddleware) Tools() []tools.Tool {
    return []tools.Tool{
        {
            Name:        "custom_tool",
            Description: "自定义工具",
            Handler: func(ctx context.Context, tc *tools.ToolContext) (interface{}, error) {
                // 工具实现
                return "result", nil
            },
        },
    }
}
```

### 拦截工具调用

```go
func (m *CustomMiddleware) WrapToolCall(
    ctx context.Context,
    req *middleware.ToolCallRequest,
    handler middleware.ToolCallHandler,
) (*middleware.ToolCallResponse, error) {
    // 前置处理：记录、验证、修改参数
    log.Printf("工具调用: %s", req.ToolName)

    // 可以修改请求
    if req.ToolName == "bash" {
        // 验证安全性
        if !isSafeCommand(req.ToolInput) {
            return nil, fmt.Errorf("不安全的命令")
        }
    }

    // 调用下一层
    resp, err := handler(ctx, req)

    // 后置处理：记录结果、转换格式
    if err == nil {
        log.Printf("工具结果: %v", resp.Result)
    }

    return resp, err
}
```

### 生命周期钩子

```go
func (m *CustomMiddleware) OnAgentStart(ctx context.Context, agentID string) error {
    log.Printf("Agent 启动: %s", agentID)
    // 初始化资源
    return nil
}

func (m *CustomMiddleware) OnAgentStop(ctx context.Context, agentID string) error {
    log.Printf("Agent 停止: %s", agentID)
    // 清理资源
    return nil
}
```

---

## 类型定义

### ModelRequest

模型请求结构。

```go
type ModelRequest struct {
    Messages     []types.Message           // 消息历史
    SystemPrompt string                    // 系统提示词
    Tools        []tools.Tool              // 可用工具
    Metadata     map[string]interface{}    // 元数据
}
```

---

### ModelResponse

模型响应结构。

```go
type ModelResponse struct {
    Message  types.Message              // 响应消息
    Metadata map[string]interface{}     // 元数据
}
```

---

### ToolCallRequest

工具调用请求结构。

```go
type ToolCallRequest struct {
    ToolCallID   string                    // 调用 ID
    ToolName     string                    // 工具名称
    ToolInput    map[string]interface{}    // 输入参数
    Tool         tools.Tool                // 工具实例
    Context      *tools.ToolContext        // 工具上下文
    Metadata     map[string]interface{}    // 元数据
}
```

---

### ToolCallResponse

工具调用响应结构。

```go
type ToolCallResponse struct {
    Result   interface{}                // 执行结果
    Metadata map[string]interface{}     // 元数据
}
```

---

### ModelCallHandler

模型调用处理器。

```go
type ModelCallHandler func(ctx context.Context, req *ModelRequest) (*ModelResponse, error)
```

---

### ToolCallHandler

工具调用处理器。

```go
type ToolCallHandler func(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error)
```

---

## 实战示例

### 日志中间件

记录所有模型调用和工具调用：

```go
type LoggingMiddleware struct {
    *middleware.BaseMiddleware
    logger *log.Logger
}

func NewLoggingMiddleware() *LoggingMiddleware {
    return &LoggingMiddleware{
        BaseMiddleware: middleware.NewBaseMiddleware("logging", 10),
        logger:         log.New(os.Stdout, "[Middleware] ", log.LstdFlags),
    }
}

func (m *LoggingMiddleware) WrapModelCall(
    ctx context.Context,
    req *middleware.ModelRequest,
    handler middleware.ModelCallHandler,
) (*middleware.ModelResponse, error) {
    start := time.Now()
    m.logger.Printf("模型调用开始 - %d 条消息", len(req.Messages))

    resp, err := handler(ctx, req)

    duration := time.Since(start)
    if err != nil {
        m.logger.Printf("模型调用失败 - 耗时: %v, 错误: %v", duration, err)
    } else {
        m.logger.Printf("模型调用成功 - 耗时: %v", duration)
    }

    return resp, err
}

func (m *LoggingMiddleware) WrapToolCall(
    ctx context.Context,
    req *middleware.ToolCallRequest,
    handler middleware.ToolCallHandler,
) (*middleware.ToolCallResponse, error) {
    m.logger.Printf("工具调用: %s(%v)", req.ToolName, req.ToolInput)
    resp, err := handler(ctx, req)
    if err != nil {
        m.logger.Printf("工具失败: %s - %v", req.ToolName, err)
    }
    return resp, err
}
```

---

### 速率限制中间件

限制模型调用频率：

```go
type RateLimitMiddleware struct {
    *middleware.BaseMiddleware
    limiter *rate.Limiter
}

func NewRateLimitMiddleware(rps int) *RateLimitMiddleware {
    return &RateLimitMiddleware{
        BaseMiddleware: middleware.NewBaseMiddleware("ratelimit", 5),
        limiter:        rate.NewLimiter(rate.Limit(rps), rps),
    }
}

func (m *RateLimitMiddleware) WrapModelCall(
    ctx context.Context,
    req *middleware.ModelRequest,
    handler middleware.ModelCallHandler,
) (*middleware.ModelResponse, error) {
    // 等待令牌
    if err := m.limiter.Wait(ctx); err != nil {
        return nil, fmt.Errorf("速率限制: %w", err)
    }

    return handler(ctx, req)
}
```

---

### 缓存中间件

缓存模型响应：

```go
type CacheMiddleware struct {
    *middleware.BaseMiddleware
    cache map[string]*middleware.ModelResponse
    mu    sync.RWMutex
}

func NewCacheMiddleware() *CacheMiddleware {
    return &CacheMiddleware{
        BaseMiddleware: middleware.NewBaseMiddleware("cache", 15),
        cache:          make(map[string]*middleware.ModelResponse),
    }
}

func (m *CacheMiddleware) WrapModelCall(
    ctx context.Context,
    req *middleware.ModelRequest,
    handler middleware.ModelCallHandler,
) (*middleware.ModelResponse, error) {
    // 计算缓存键
    key := computeCacheKey(req.Messages)

    // 查询缓存
    m.mu.RLock()
    if cached, ok := m.cache[key]; ok {
        m.mu.RUnlock()
        log.Printf("缓存命中: %s", key)
        return cached, nil
    }
    m.mu.RUnlock()

    // 调用模型
    resp, err := handler(ctx, req)
    if err != nil {
        return nil, err
    }

    // 写入缓存
    m.mu.Lock()
    m.cache[key] = resp
    m.mu.Unlock()

    return resp, nil
}

func computeCacheKey(messages []types.Message) string {
    // 简化示例：实际应使用更稳定的哈希算法
    return fmt.Sprintf("%v", messages)
}
```

---

## 最佳实践

### 1. 优先级设置

遵循推荐的优先级范围：

```go
// 0-100: 系统核心中间件
const PriorityCore = 10

// 100-500: 功能中间件
const PriorityFeature = 100

// 500-1000: 用户自定义
const PriorityCustom = 500
```

### 2. 调用下一层

始终调用 `handler` 以保持链式调用：

```go
// ✅ 正确
resp, err := handler(ctx, req)

// ❌ 错误：中断了中间件链
return &ModelResponse{...}, nil
```

### 3. 错误处理

在中间件中妥善处理错误：

```go
resp, err := handler(ctx, req)
if err != nil {
    log.Printf("下游错误: %v", err)
    // 可以选择：
    // 1. 直接返回错误
    return nil, err
    // 2. 包装错误
    return nil, fmt.Errorf("中间件处理失败: %w", err)
    // 3. 降级处理
    return fallbackResponse(), nil
}
```

### 4. 资源清理

使用生命周期钩子管理资源：

```go
func (m *MyMiddleware) OnAgentStart(ctx context.Context, agentID string) error {
    m.conn, err = openDatabase()
    return err
}

func (m *MyMiddleware) OnAgentStop(ctx context.Context, agentID string) error {
    return m.conn.Close()
}
```

### 5. 线程安全

如果中间件有状态，确保线程安全：

```go
type StatefulMiddleware struct {
    *middleware.BaseMiddleware
    mu    sync.RWMutex
    cache map[string]string
}
```

---

## 相关资源

- [Agent API 文档](./1.agent-api.md)
- [Tools API 文档](./4.tools-api.md)
- [Middleware 实战指南](../guides/middleware)
- [完整 API 文档 (pkg.go.dev)](https://pkg.go.dev/github.com/astercloud/aster/pkg/middleware)
