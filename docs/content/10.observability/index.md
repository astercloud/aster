---
title: 可观测性
description: 监控、日志、追踪和调试你的 Agent 应用
---

# 可观测性

完整的可观测性支持，帮助你了解 Agent 的运行状态。

## 📚 分类

### [日志](/observability/logging)

- 结构化日志
- 日志级别控制
- 日志采集

### [监控](/observability/monitoring)

- 性能指标
- 业务指标
- 告警配置

### [追踪](/observability/tracing)

- OpenTelemetry 集成
- 分布式追踪
- 链路分析

### [调试](/observability/debugging)

- 断点调试
- 事件回放
- 问题诊断

## 🚀 快速开始

```go
// 启用 OpenTelemetry
tracer := telemetry.NewTracer(config)
agent.WithTracer(tracer)

// 记录日志
logger.Info("agent started", "agent_id", agent.ID())
```

## 📖 相关文档

- [最佳实践：监控](/best-practices/monitoring)
- [部署指南](/deployment)
