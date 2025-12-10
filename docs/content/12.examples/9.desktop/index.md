---
title: 桌面应用示例
description: 桌面应用开发相关的代码示例
---

# 桌面应用示例

本节包含桌面应用开发相关的完整代码示例。

## 📚 示例列表

### SQLite 会话存储

轻量级本地会话存储，适用于桌面应用。

```bash
go run ./examples/session-sqlite/
```

- 创建和管理会话
- 添加和查询消息
- 会话持久化和恢复
- 数据库维护

### Permission 权限系统

工具执行的权限控制和审批流程。

```bash
go run ./examples/permission/
```

- 三种审批模式 (auto/smart/always_ask)
- 基于风险的智能决策
- 规则配置和持久化
- 与 Agent 集成

### Recipe 配置系统

声明式的 Agent 配置方式。

```bash
go run ./examples/recipe/
```

- YAML 格式配置
- Builder 模式创建
- MCP 扩展配置
- 参数化模板

### 跨平台路径

跨平台的路径管理系统。

```bash
go run ./examples/config-paths/
```

- 标准路径约定
- 便捷的文件路径生成
- 自定义应用名
- 目录创建

### 桌面框架集成

Wails、Tauri、Electron 集成示例。

```bash
go run ./examples/desktop/
```

- Wails 直接绑定
- Tauri HTTP/WebSocket
- Electron HTTP/WebSocket
- 事件流处理

## 🔗 相关文档

- [SQLite 会话存储](/core-concepts/session-sqlite)
- [Permission 系统](/security/permission)
- [Recipe 配置](/core-concepts/recipe)
- [桌面应用部署](/deployment/desktop)
