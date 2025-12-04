---
title: 中间件系统
description: 使用中间件扩展 Agent 功能，实现洋葱模型架构
navigation: false
---

# 中间件系统

中间件采用洋葱模型架构，每个请求和响应都会依次通过多层中间件。

## 📚 分类

### [内置中间件](/middleware/builtin)

- Filesystem - 文件系统访问控制
- SubAgent - 子 Agent 支持
- Summarization - 自动上下文总结
- Memory - 记忆系统集成
- SimplicityChecker - 简洁性检查（防止过度工程）

### [自定义中间件](/middleware/custom)

- 创建自定义中间件
- 中间件注册
- 优先级配置

## 🎯 洋葱模型

```
请求 → 中间件1 → 中间件2 → Agent → 中间件2 → 中间件1 → 响应
```

优先级数值越大的中间件位于越外层。

## 📖 相关文档

- [中间件 API 参考](/api-reference/middleware)
- [中间件示例](/examples/middleware)
- [核心概念：中间件](/core-concepts/middleware)
