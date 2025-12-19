---
title: 远程 Agent 集成
description: 将外部应用的 Agent 连接到 Aster Studio
---

# 远程 Agent 集成

Aster Studio 支持监控远程 Agent，允许你将任何使用 Aster 框架的应用连接到 Studio 进行统一监控。

## 架构概览

```
┌─────────────────┐     WebSocket      ┌─────────────────┐
│  Your App       │ ─────────────────► │  Aster Studio   │
│  (Agent Host)   │   /v1/remote-      │  (Server)       │
│                 │   agents/connect   │                 │
└─────────────────┘                    └─────────────────┘
        │                                      │
        │ Events                               │ Dashboard
        ▼                                      ▼
   Agent EventBus                         Web Console
```

## 客户端实现

### Go 客户端示例

```go
package main

import (
    "github.com/gorilla/websocket"
    "encoding/json"
)

type StudioClient struct {
    conn     *websocket.Conn
    agentID  string
}

func NewStudioClient(studioURL, agentID string) (*StudioClient, error) {
    conn, _, err := websocket.DefaultDialer.Dial(
        studioURL + "/v1/remote-agents/connect",
        nil,
    )
    if err != nil {
        return nil, err
    }

    client := &StudioClient{conn: conn, agentID: agentID}

    // 注册 Agent
    client.Send(map[string]any{
        "type": "register",
        "agent": map[string]any{
            "id":     agentID,
            "name":   "My Agent",
            "status": "ready",
        },
    })

    return client, nil
}

func (c *StudioClient) SendEvent(event map[string]any) error {
    return c.Send(map[string]any{
        "type":     "event",
        "agent_id": c.agentID,
        "event":    event,
    })
}

func (c *StudioClient) Send(msg map[string]any) error {
    data, _ := json.Marshal(msg)
    return c.conn.WriteMessage(websocket.TextMessage, data)
}
```

### 发送事件

```go
// Token 使用事件
client.SendEvent(map[string]any{
    "channel":       "monitor",
    "event_type":    "token_usage",
    "input_tokens":  100,
    "output_tokens": 50,
})

// 状态变更事件
client.SendEvent(map[string]any{
    "channel":    "monitor",
    "event_type": "state_changed",
    "state":      "working",
})

// 错误事件
client.SendEvent(map[string]any{
    "channel":    "monitor",
    "event_type": "error",
    "severity":   "error",
    "message":    "API call failed",
})

// 文本输出事件
client.SendEvent(map[string]any{
    "channel":    "progress",
    "event_type": "text_chunk",
    "delta":      "Hello, ",
})
```

## 消息协议

### 注册 Agent

```json
{
  "type": "register",
  "agent": {
    "id": "agent-123",
    "name": "Writing Agent",
    "status": "ready",
    "model": "claude-3-5-sonnet",
    "capabilities": ["writing", "editing"]
  }
}
```

### 注册 Session

```json
{
  "type": "register_session",
  "session": {
    "id": "session-456",
    "agent_id": "agent-123",
    "created_at": "2024-01-01T12:00:00Z"
  }
}
```

### 发送事件

```json
{
  "type": "event",
  "agent_id": "agent-123",
  "event": {
    "channel": "monitor",
    "event_type": "token_usage",
    "input_tokens": 100,
    "output_tokens": 50
  }
}
```

### 心跳

```json
{
  "type": "ping"
}
```

响应：

```json
{
  "type": "pong"
}
```

## 环境变量配置

在你的应用中启用 Studio 集成：

```bash
# 启用 Studio 客户端
ASTER_STUDIO_ENABLED=true

# Studio 服务地址
ASTER_STUDIO_URL=ws://localhost:3032
```

## 最佳实践

1. **连接管理**: 实现自动重连机制，处理网络断开情况
2. **事件批量发送**: 对于高频事件，考虑批量发送减少网络开销
3. **错误处理**: 发送失败时不应影响主业务逻辑
4. **资源清理**: 应用退出时正确关闭 WebSocket 连接
