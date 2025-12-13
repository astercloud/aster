---
title: 事件流
description: 实时查看和筛选 Agent 事件
---

# 事件流

Events 页面提供实时事件流查看功能，支持按通道和事件类型筛选。

## 事件通道

Aster 使用三个事件通道：

| 通道 | 说明 | 典型事件 |
|------|------|----------|
| `progress` | 进度事件，用于 UI 展示 | text_chunk, tool:start, tool:end, done |
| `control` | 控制事件，用于人机交互 | permission_required, permission_decided |
| `monitor` | 监控事件，用于治理和审计 | token_usage, error, state_changed |

## 事件类型

### Monitor 通道事件

| 事件类型 | 说明 |
|----------|------|
| `token_usage` | Token 使用统计 |
| `tool_executed` | 工具执行完成 |
| `step_complete` | 步骤完成 |
| `state_changed` | Agent 状态变更 |
| `error` | 错误事件 |

### Progress 通道事件

| 事件类型 | 说明 |
|----------|------|
| `text_chunk` | 文本流式输出 |
| `text_chunk_start` | 文本输出开始 |
| `text_chunk_end` | 文本输出结束 |
| `think_chunk` | 思考过程输出 |
| `tool:start` | 工具开始执行 |
| `tool:end` | 工具执行结束 |
| `tool:progress` | 工具执行进度 |
| `done` | Agent 执行完成 |

### Control 通道事件

| 事件类型 | 说明 |
|----------|------|
| `permission_required` | 需要用户授权 |
| `permission_decided` | 用户授权决定 |

## 筛选功能

### 按通道筛选

点击筛选器中的通道按钮（progress/control/monitor）可以只显示特定通道的事件。

### 按事件类型筛选

点击事件类型按钮可以筛选特定类型的事件，例如只查看 `error` 事件。

### 搜索

在搜索框中输入关键词，可以在事件内容中搜索匹配的事件。

## WebSocket 订阅

Events 页面通过 WebSocket 连接实时接收事件：

```
ws://localhost:3032/v1/dashboard/events/stream
```

### 订阅消息格式

```json
{
  "action": "subscribe",
  "filters": {
    "channels": ["monitor", "progress"],
    "event_types": ["error", "token_usage"],
    "agent_ids": ["agent-123"]
  }
}
```

### 事件消息格式

```json
{
  "type": "event",
  "payload": {
    "cursor": "abc123",
    "timestamp": "2024-01-01T12:00:00Z",
    "agent_id": "writing-agent",
    "channel": "monitor",
    "type": "token_usage",
    "data": {
      "input_tokens": 100,
      "output_tokens": 50,
      "total_tokens": 150
    }
  }
}
```

## 远程 Agent 事件

当使用远程 Agent（通过 WebSocket 连接到 Aster Server）时，事件会自动转发到 Studio。

远程 Agent 需要在发送事件时包含 `channel` 和 `event_type` 字段：

```json
{
  "channel": "monitor",
  "event_type": "token_usage",
  "input_tokens": 100,
  "output_tokens": 50
}
```
