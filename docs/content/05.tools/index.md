---
title: 工具系统
description: aster 的工具系统提供丰富的内置工具和灵活的扩展机制
navigation: false
---

# 工具系统

aster 提供了强大的工具系统，让 Agent 能够与外部世界交互。

## 📚 分类

### [内置工具](/tools/builtin)
- 文件系统操作
- Bash 命令执行
- HTTP 请求
- Web 搜索
- Todo 管理

### [MCP 协议](/tools/mcp)
- MCP Client
- MCP Server
- 协议扩展

### [自定义工具](/tools/custom)
- 创建自定义工具
- 工具注册
- 工具生命周期

### [Programmatic Tool Calling](/tools/ptc) ::badge{type="success"}新::
- 让 LLM 生成的 Python 代码直接调用工具
- HTTP 桥接服务器
- 异步 Python SDK
- 错误处理和重试
- 性能优化

## 🚀 快速开始

```go
// 注册内置工具
toolRegistry := tools.NewRegistry()
builtin.RegisterAll(toolRegistry)

// 使用工具
result, err := tool.Execute(ctx, params, toolContext)
```

## 📖 相关文档

- [工具 API 参考](/api-reference/tools)
- [工具示例](/examples/tools)
- [核心概念：工具系统](/core-concepts/tools-system)
