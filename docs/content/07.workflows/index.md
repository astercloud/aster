---
title: 工作流系统
description: 使用工作流编排复杂的多步骤任务
navigation: false
---

# 工作流系统

工作流系统支持多种执行模式，适用于不同的业务场景。

## 📚 分类

### [基础工作流](/workflows/basic)

- Sequential - 顺序执行
- Parallel - 并行执行
- Conditional - 条件分支
- Loop - 循环执行

### [高级模式](/workflows/advanced)

- 错误处理与重试
- 工作流持久化
- 动态工作流构建
- 工作流监控

## 🚀 快速开始

```go
// 创建工作流
workflow := NewWorkflow("data-pipeline")
workflow.AddStep("extract", extractAgent)
workflow.AddStep("transform", transformAgent)
workflow.AddStep("load", loadAgent)

// 执行
result, err := workflow.Execute(ctx, input)
```

## 📖 相关文档

- [工作流 API 参考](/api-reference/workflow)
- [工作流示例](/examples/workflows)
- [核心概念：工作流 Agent](/core-concepts/workflow-agents)
