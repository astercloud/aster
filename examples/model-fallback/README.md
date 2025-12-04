# Model Fallback 示例

本示例演示如何使用 Aster 的模型降级（Model Fallback）功能，实现高可用的 AI 应用。

## 功能特性

### 1. 自动模型降级

当主模型失败时，自动切换到备用模型，确保服务可用性。

### 2. 智能重试机制

每个模型可以配置独立的重试次数，支持指数退避策略。

### 3. 优先级管理

通过优先级控制模型的使用顺序，优先使用性能更好或成本更低的模型。

### 4. 动态模型管理

运行时动态启用/禁用模型，无需重启服务。

### 5. 统计和监控

实时统计模型使用情况、成功率、降级次数等指标。

## 使用场景

### 场景 1: 高可用性保障

```go
fallbacks := []*agent.ModelFallback{
    {
        Config: &types.ModelConfig{
            Provider: "anthropic",
            Model:    "claude-sonnet-4-5",
        },
        MaxRetries: 2,
        Priority:   1,
    },
    {
        Config: &types.ModelConfig{
            Provider: "openai",
            Model:    "gpt-4o",
        },
        MaxRetries: 1,
        Priority:   2,
    },
}
```

### 场景 2: 成本优化

```go
fallbacks := []*agent.ModelFallback{
    {
        Config: &types.ModelConfig{
            Provider: "deepseek",  // 成本低
            Model:    "deepseek-chat",
        },
        MaxRetries: 1,
        Priority:   1,  // 优先使用
    },
    {
        Config: &types.ModelConfig{
            Provider: "anthropic",  // 质量高但成本高
            Model:    "claude-sonnet-4-5",
        },
        MaxRetries: 0,
        Priority:   2,  // 备用
    },
}
```

### 场景 3: 区域容灾

```go
fallbacks := []*agent.ModelFallback{
    {
        Config: &types.ModelConfig{
            Provider: "openai",
            Model:    "gpt-4o",
            BaseURL:  "https://api.openai.com",  // 主区域
        },
        MaxRetries: 1,
        Priority:   1,
    },
    {
        Config: &types.ModelConfig{
            Provider: "openai",
            Model:    "gpt-4o",
            BaseURL:  "https://api.openai-backup.com",  // 备用区域
        },
        MaxRetries: 0,
        Priority:   2,
    },
}
```

## 配置说明

### ModelFallback 配置

| 字段       | 类型                | 说明                         |
| ---------- | ------------------- | ---------------------------- |
| Config     | \*types.ModelConfig | 模型配置                     |
| MaxRetries | int                 | 最大重试次数                 |
| Enabled    | bool                | 是否启用                     |
| Priority   | int                 | 优先级（数字越小优先级越高） |

### 重试策略

- **指数退避**: 重试间隔为 `retry * 500ms`
- **上下文感知**: 支持 context 取消和超时
- **错误日志**: 完整记录每次重试的错误信息

## 运行示例

```bash
# 设置 API Keys
export ANTHROPIC_API_KEY="your-key"
export OPENAI_API_KEY="your-key"
export DEEPSEEK_API_KEY="your-key"

# 运行示例
go run main.go
```

## 输出示例

```
=== Model Fallback 示例 ===

示例 1: 非流式请求
---
[ModelFallback] Success with model anthropic/claude-sonnet-4-5 (retry: 0)
响应: 人工智能是让计算机模拟人类智能行为的技术。

示例 2: 流式请求
---
[ModelFallback] Success with model anthropic/claude-sonnet-4-5 (stream, retry: 0)
响应: 人工智能是让计算机模拟人类智能行为的技术。

示例 3: 统计信息
---
总请求数: 2
成功请求数: 2
失败请求数: 0
降级次数: 0

模型使用统计:
  anthropic/claude-sonnet-4-5: 2 次

示例 4: 动态管理模型
---
当前模型列表:
  [启用] anthropic/claude-sonnet-4-5 - 优先级: 1, 重试: 2 (当前使用)
  [启用] openai/gpt-4o - 优先级: 2, 重试: 1
  [启用] deepseek/deepseek-chat - 优先级: 3, 重试: 0

禁用主模型 (anthropic/claude-sonnet-4-5)...
[ModelFallback] Success with model openai/gpt-4o (retry: 0)
使用备用模型的响应: 人工智能是让计算机模拟人类智能行为的技术。

重新启用主模型...

示例 5: 自动重试和降级
---
当主模型失败时，系统会自动:
1. 重试主模型（根据 MaxRetries 配置）
2. 如果所有重试都失败，自动降级到下一个模型
3. 重复此过程直到成功或所有模型都失败

=== 最终统计 ===
总请求数: 3
成功率: 100.00%
降级次数: 0
```

## 最佳实践

### 1. 合理配置重试次数

- 主模型: 2-3 次重试
- 备用模型: 1-2 次重试
- 最后备用: 0-1 次重试

### 2. 优先级设置

- 按性能/成本/可用性综合考虑
- 定期评估和调整优先级

### 3. 监控和告警

```go
stats := manager.GetStats()
if stats.FallbackCount > threshold {
    // 触发告警
    alert("Model fallback rate is high")
}
```

### 4. 动态调整

```go
// 根据监控指标动态调整
if modelHealth["claude"] < 0.9 {
    manager.DisableModel("anthropic", "claude-sonnet-4-5")
}
```

### 5. 错误处理

```go
resp, err := manager.Complete(ctx, messages, opts)
if err != nil {
    // 所有模型都失败
    log.Error("All models failed", "error", err)
    // 降级到缓存响应或默认响应
    return fallbackResponse()
}
```

## 性能考虑

### 1. 重试开销

- 每次重试增加 500ms \* retry 的延迟
- 建议设置合理的超时时间

### 2. 模型切换

- 模型切换几乎无开销（已预初始化）
- 建议预热所有模型的连接

### 3. 统计开销

- 统计信息使用原子操作，开销极小
- 可以安全地在高并发场景使用

## 故障排查

### 问题 1: 所有模型都失败

```
Error: all models failed, last error: ...
```

**解决方案**:

1. 检查 API Keys 是否正确
2. 检查网络连接
3. 查看详细的错误日志

### 问题 2: 频繁降级

```
FallbackCount: 100
```

**解决方案**:

1. 检查主模型的健康状态
2. 增加主模型的重试次数
3. 考虑更换主模型

### 问题 3: 响应延迟高

```
Average latency: 5s
```

**解决方案**:

1. 减少重试次数
2. 设置合理的超时时间
3. 使用更快的模型作为主模型

## 相关文档

- [Agent 文档](../../docs/agent.md)
- [Provider 文档](../../docs/provider.md)
- [配置指南](../../docs/configuration.md)

## 许可证

Apache 2.0
