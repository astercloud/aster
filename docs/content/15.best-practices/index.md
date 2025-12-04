---
title: 最佳实践
description: aster 开发的最佳实践和生产环境建议
navigation: false
---

# 最佳实践

本章节提供 Agent SDK 开发和生产部署的最佳实践，帮助你构建稳定、高效、安全的 AI Agent 应用。

## 📚 实践指南

<div class="grid grid-cols-1 md:grid-cols-2 gap-4 my-6">
  <a href="/best-practices/error-handling" class="block p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:border-primary-500 transition-colors">
    <h3 class="font-semibold mb-2">🚨 错误处理</h3>
    <p class="text-sm text-gray-600 dark:text-gray-400">
      异常处理、重试策略、优雅降级、错误监控
    </p>
  </a>

  <a href="/best-practices/performance" class="block p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:border-primary-500 transition-colors">
    <h3 class="font-semibold mb-2">⚡ 性能优化</h3>
    <p class="text-sm text-gray-600 dark:text-gray-400">
      Token 优化、并发控制、缓存策略、资源管理
    </p>
  </a>

  <a href="/best-practices/security" class="block p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:border-primary-500 transition-colors">
    <h3 class="font-semibold mb-2">🔒 安全建议</h3>
    <p class="text-sm text-gray-600 dark:text-gray-400">
      API 密钥管理、输入验证、沙箱安全、审计日志
    </p>
  </a>

  <a href="/best-practices/testing" class="block p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:border-primary-500 transition-colors">
    <h3 class="font-semibold mb-2">🧪 测试策略</h3>
    <p class="text-sm text-gray-600 dark:text-gray-400">
      单元测试、集成测试、Mock 策略、测试覆盖率
    </p>
  </a>

  <a href="/best-practices/monitoring" class="block p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:border-primary-500 transition-colors">
    <h3 class="font-semibold mb-2">📊 监控运维</h3>
    <p class="text-sm text-gray-600 dark:text-gray-400">
      日志记录、指标收集、告警配置、问题排查
    </p>
  </a>

  <a href="/best-practices/deployment" class="block p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:border-primary-500 transition-colors">
    <h3 class="font-semibold mb-2">🚀 部署实践</h3>
    <p class="text-sm text-gray-600 dark:text-gray-400">
      容器化部署、高可用架构、灰度发布、回滚策略
    </p>
  </a>
</div>

## 🎯 核心原则

### 1. 可靠性优先

```go
// ✅ 好的实践：完善的错误处理
func createAgent(ctx context.Context) (*agent.Agent, error) {
    ag, err := agent.Create(ctx, config, deps)
    if err != nil {
        return nil, fmt.Errorf("failed to create agent: %w", err)
    }

    // 确保资源释放
    go func() {
        <-ctx.Done()
        if err := ag.Close(); err != nil {
            log.Printf("Failed to close agent: %v", err)
        }
    }()

    return ag, nil
}

// ❌ 不好的实践：忽略错误
func createAgentBad(ctx context.Context) *agent.Agent {
    ag, _ := agent.Create(ctx, config, deps)  // 忽略错误
    return ag  // 可能返回 nil
}
```

### 2. 性能意识

```go
// ✅ 使用 Summarization 中间件管理上下文
summaryMW, _ := middleware.NewSummarizationMiddleware(&middleware.SummarizationMiddlewareConfig{
    MaxTokensBeforeSummary: 150000,
    MessagesToKeep:         6,
})

// ✅ 合理使用工具缓存
type CachedTool struct {
    cache map[string]interface{}
    ttl   time.Duration
}

// ❌ 不管理上下文，导致 Token 溢出
// ❌ 重复执行相同的工具调用
```

### 3. 安全第一

```go
// ✅ 安全的配置管理
config := &types.AgentConfig{
    ModelConfig: &types.ModelConfig{
        APIKey: os.Getenv("ANTHROPIC_API_KEY"),  // 从环境变量读取
    },
    Sandbox: &types.SandboxConfig{
        Kind:    types.SandboxKindLocal,
        WorkDir: "./workspace",  // 限制工作目录
        AllowedPathPrefixes: []string{"/workspace"},  // 路径验证
    },
}

// ❌ 硬编码密钥
config := &types.AgentConfig{
    ModelConfig: &types.ModelConfig{
        APIKey: "sk-ant-xxxxx",  // 危险！
    },
}
```

### 4. 可观测性

```go
// ✅ 结构化日志
log.Printf("[Agent:%s] [Step:%d] Tool call: %s",
    ag.ID(), stepCount, toolName)

// ✅ 指标收集
metrics.Increment("agent.tool_calls.total", 1)
metrics.Histogram("agent.response_time", duration)

// ✅ 事件监听
ag.Subscribe([]types.AgentChannel{
    types.ChannelProgress,
    types.ChannelMonitor,
}, nil)
```

### 5. 渐进式优化

```go
// 第1阶段：基础功能
ag, _ := agent.Create(ctx, basicConfig, baseDeps)

// 第2阶段：添加中间件
stack := middleware.NewStack()
stack.Use(summaryMW)
stack.Use(filesMW)

// 第3阶段：性能优化
stack.Use(cacheMW)
stack.Use(rateLimitMW)

// 第4阶段：生产加固
stack.Use(monitoringMW)
stack.Use(securityMW)
```

## 📊 开发阶段对照表

| 阶段           | 重点     | 关键实践                       |
| -------------- | -------- | ------------------------------ |
| **原型开发**   | 快速验证 | 简单配置、基础工具、单Agent    |
| **功能开发**   | 完善能力 | 中间件、自定义工具、错误处理   |
| **性能优化**   | 提升效率 | Token 优化、缓存、并发控制     |
| **生产准备**   | 稳定可靠 | 监控、日志、安全加固、灰度发布 |
| **规模化运营** | 成本控制 | 多租户隔离、资源池化、成本监控 |

## 🎨 架构模式

### 单一职责

```go
// ✅ 专注的 Agent
config := &types.AgentConfig{
    TemplateID: "data-analyst",  // 单一职责
    SystemPrompt: "你是数据分析专家...",
    Tools: []interface{}{
        "pandas_query",
        "matplotlib",
    },
}

// ❌ 万能 Agent（职责不清）
config := &types.AgentConfig{
    TemplateID: "super-agent",
    SystemPrompt: "你什么都会...",  // 职责混乱
    Tools: []interface{}{
        "*",  // 所有工具
    },
}
```

### 依赖注入

```go
// ✅ 便于测试和替换
type Dependencies struct {
    Store           store.Store
    ToolRegistry    tools.Registry
    ProviderFactory provider.Factory
}

func NewAgent(deps *Dependencies) *Agent {
    return &Agent{
        store:    deps.Store,
        tools:    deps.ToolRegistry,
        provider: deps.ProviderFactory,
    }
}

// ❌ 硬编码依赖
func NewAgentBad() *Agent {
    store := store.NewJSONStore("./data")  // 固定实现
    return &Agent{store: store}
}
```

### 中间件模式

```go
// ✅ 关注点分离
stack := middleware.NewStack()
stack.Use(loggingMW)      // 日志
stack.Use(metricsMW)      // 指标
stack.Use(securityMW)     // 安全
stack.Use(businessMW)     // 业务逻辑

// ✅ 每个中间件单一职责
type LoggingMiddleware struct {
    *middleware.BaseMiddleware
}

func (m *LoggingMiddleware) WrapToolCall(...) {
    log.Printf("Tool call: %s", req.ToolName)
    return handler(ctx, req)
}
```

## 💡 通用建议

### 配置管理

```go
// ✅ 使用配置文件或环境变量
type Config struct {
    APIKey           string        `env:"ANTHROPIC_API_KEY"`
    MaxAgents        int           `env:"MAX_AGENTS" default:"50"`
    WorkDir          string        `env:"WORK_DIR" default:"./workspace"`
    TokenLimit       int           `env:"TOKEN_LIMIT" default:"150000"`
    EnableMonitoring bool          `env:"ENABLE_MONITORING" default:"true"`
}

// 加载配置
cfg := loadConfig()
```

### 资源管理

```go
// ✅ 始终释放资源
ag, err := agent.Create(ctx, config, deps)
if err != nil {
    return err
}
defer ag.Close()  // 确保关闭

// ✅ 超时控制
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

// ✅ 优雅关闭
func shutdown(pool *core.Pool, scheduler *core.Scheduler) {
    log.Println("Shutting down...")
    scheduler.Shutdown()
    pool.Shutdown()
    log.Println("Shutdown complete")
}
```

### 错误边界

```go
// ✅ 在关键点捕获错误
func handleChat(w http.ResponseWriter, r *http.Request) {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Panic recovered: %v", r)
            http.Error(w, "Internal server error", 500)
        }
    }()

    // 业务逻辑
}

// ✅ 错误分类处理
switch err := err.(type) {
case *types.TokenLimitError:
    // Token 超限，触发总结
case *types.ToolExecutionError:
    // 工具执行失败，重试或降级
case *types.NetworkError:
    // 网络错误，重试
default:
    // 其他错误，记录并返回
}
```

## 📖 学习路径

**建议阅读顺序：**

1. **[错误处理](/best-practices/error-handling)** - 构建可靠的 Agent
2. **[性能优化](/best-practices/performance)** - 提升响应速度和降低成本
3. **[安全建议](/best-practices/security)** - 保护系统和数据安全
4. **[测试策略](/best-practices/testing)** - 确保代码质量
5. **[监控运维](/best-practices/monitoring)** - 生产环境可观测性
6. **[部署实践](/best-practices/deployment)** - 上线和运维

## 🔗 相关资源

- [核心概念](/core-concepts) - 理解架构设计
- [示例代码](/examples) - 实际应用参考
- [API 参考](/api-reference) - 详细接口文档
- [FAQ](/faq) - 常见问题解答

## ⚠️ 常见陷阱

### 1. Token 管理不当

```go
// ❌ 不监控 Token 使用
// → 对话过长导致超限

// ✅ 使用 Summarization 中间件
summaryMW, _ := middleware.NewSummarizationMiddleware(...)
```

### 2. 错误处理缺失

```go
// ❌ 忽略错误
ag.Chat(ctx, "message")

// ✅ 处理错误
result, err := ag.Chat(ctx, "message")
if err != nil {
    log.Printf("Chat failed: %v", err)
    return handleError(err)
}
```

### 3. 资源泄漏

```go
// ❌ 忘记关闭
ag, _ := agent.Create(ctx, config, deps)
// ... 使用后没有 Close()

// ✅ 确保关闭
defer ag.Close()
```

### 4. 硬编码配置

```go
// ❌ 硬编码
APIKey: "sk-ant-xxxxx"
Model: "claude-sonnet-4-5"

// ✅ 外部配置
APIKey: os.Getenv("ANTHROPIC_API_KEY")
Model: config.GetString("model")
```

### 5. 过度优化

```go
// ❌ 过早优化
// 在功能还不稳定时就进行复杂的缓存和优化

// ✅ 渐进式优化
// 1. 先确保功能正确
// 2. 测量性能瓶颈
// 3. 针对性优化
// 4. 验证优化效果
```

---

通过遵循这些最佳实践，你可以构建出高质量、可维护的 AI Agent 应用。记住：**先保证正确性，再追求性能；先确保安全，再谈用户体验**。
