# A2A 协议示例

这个示例演示了如何使用 Aster 的 A2A (Agent-to-Agent) 协议实现。

## A2A 协议简介

A2A (Agent-to-Agent) 是一个基于 HTTP/JSON-RPC 2.0 的标准化协议,用于实现 AI Agent 之间的通信。主要特性:

- **标准化接口**: 使用 JSON-RPC 2.0 作为基础协议
- **Agent 发现**: 通过 Agent Card 发布 Agent 的元数据和能力
- **任务管理**: 支持异步任务执行和状态跟踪
- **上下文管理**: 支持对话上下文和线程管理

## 运行示例

```bash
# 设置 API Key
export ANTHROPIC_API_KEY=your-api-key

# 运行示例
go run examples/a2a/main.go
```

## 工作流程

示例程序演示了以下流程:

1. **创建 Actor System** - 初始化 Actor 模型运行时
2. **创建 Agent** - 配置并创建 AI Agent
3. **注册为 Actor** - 将 Agent 包装为 Actor
4. **初始化 A2A Server** - 创建 A2A 协议服务器
5. **获取 Agent Card** - 查询 Agent 的元数据和能力
6. **发送消息** (message/send) - 向 Agent 发送消息并创建任务
7. **查询任务** (tasks/get) - 获取任务的执行状态和结果
8. **查看对话历史** - 显示完整的对话历史记录

## 输出示例

```
=== A2A 协议示例 ===

✅ Agent Actor 已创建: demo-agent
✅ A2A Server 已创建

--- Agent Card ---
{
  "name": "demo-agent",
  "description": "Aster AI Agent: demo-agent",
  "url": "/a2a/demo-agent",
  "provider": {
    "organization": "Aster",
    "url": "https://github.com/astercloud/aster"
  },
  "version": "1.0",
  ...
}

--- 发送消息 ---
✅ 任务已创建: task-xxx

--- 获取任务状态 ---
任务状态:
{
  "id": "task-xxx",
  "contextId": "context-001",
  "status": {
    "state": "completed"
  },
  "history": [
    ...
  ]
}

--- 对话历史 ---
1. [user]
   你好! 请简单介绍一下你自己。

2. [agent]
   你好!我是一个有帮助的 AI 助手...

=== 示例完成 ===
```

## A2A 协议端点

如果将 A2A 集成到 HTTP 服务器中,将提供以下端点:

- `GET /.well-known/{agentId}/agent-card.json` - 获取 Agent Card
- `POST /a2a/{agentId}` - JSON-RPC 2.0 主端点
- `GET /a2a/{agentId}/tasks/{taskId}` - 便捷的任务状态查询端点

## 支持的方法

- `message/send` - 发送消息给 Agent
- `message/stream` - 流式消息 (待实现)
- `tasks/get` - 获取任务状态
- `tasks/cancel` - 取消正在执行的任务

## 相关资源

- [A2A 协议规范](https://github.com/astercloud/aster/tree/main/pkg/a2a)
- [Actor 模型文档](https://github.com/astercloud/aster/tree/main/pkg/actor)
- [Agent 文档](https://github.com/astercloud/aster/tree/main/pkg/agent)
