package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/types"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// RemoteAgentHandler 处理远程 Agent 的 WebSocket 连接
type RemoteAgentHandler struct {
	registry *RuntimeAgentRegistry
	store    *store.Store // 添加 store 用于持久化 agent 记录
	mu       sync.RWMutex
	agents   map[string]*agent.RemoteAgent // agentID -> RemoteAgent
	conns    map[string]*websocket.Conn    // agentID -> WebSocket 连接
}

// RemoteAgentMessage 远程 Agent 消息
type RemoteAgentMessage struct {
	Type    string          `json:"type"`    // "register", "event", "unregister", "ping"
	Payload json.RawMessage `json:"payload"` // 根据 Type 解析不同的 Payload
}

// RegisterPayload 注册消息的 Payload
type RegisterPayload struct {
	AgentID    string         `json:"agent_id"`
	TemplateID string         `json:"template_id"`
	Metadata   map[string]any `json:"metadata"`
}

// RegisterSessionPayload 注册 Session 消息的 Payload
type RegisterSessionPayload struct {
	SessionID    string         `json:"session_id"`
	AgentID      string         `json:"agent_id"`
	Status       string         `json:"status"`
	MessageCount int            `json:"message_count"`
	Metadata     map[string]any `json:"metadata"`
}

// EventPayload 事件消息的 Payload
type EventPayload struct {
	AgentID  string                   `json:"agent_id"`
	Envelope types.AgentEventEnvelope `json:"envelope"`
}

// NewRemoteAgentHandler 创建 RemoteAgentHandler
func NewRemoteAgentHandler(registry *RuntimeAgentRegistry, st store.Store) *RemoteAgentHandler {
	return &RemoteAgentHandler{
		registry: registry,
		store:    &st,
		agents:   make(map[string]*agent.RemoteAgent),
		conns:    make(map[string]*websocket.Conn),
	}
}

// HandleConnect 处理 WebSocket 连接
func (h *RemoteAgentHandler) HandleConnect(c *gin.Context) {
	// 升级 HTTP 连接到 WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logging.Error(c.Request.Context(), "remote_agent.websocket.upgrade.failed", map[string]any{
			"error": err.Error(),
		})
		return
	}

	ctx := context.Background()
	logging.Info(ctx, "remote_agent.websocket.connected", map[string]any{
		"remote_addr": c.Request.RemoteAddr,
	})

	// 发送欢迎消息
	h.sendMessage(conn, "connected", map[string]any{
		"message": "Connected to Aster Studio Remote Agent Service",
	})

	// 处理消息
	h.handleConnection(ctx, conn)
}

// handleConnection 处理 WebSocket 连接的消息循环
func (h *RemoteAgentHandler) handleConnection(ctx context.Context, conn *websocket.Conn) {
	defer func() {
		h.cleanupConnection(conn)
		_ = conn.Close()
	}()

	// 设置读取超时
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// 启动心跳 goroutine
	stopPing := make(chan struct{})
	defer close(stopPing)
	go h.pingLoop(conn, stopPing)

	// 消息循环
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logging.Error(ctx, "remote_agent.websocket.read.error", map[string]any{
					"error": err.Error(),
				})
			}
			break
		}

		// 解析消息
		var msg RemoteAgentMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			h.sendError(conn, "invalid_message", "Failed to parse message")
			continue
		}

		// 处理消息
		if err := h.handleMessage(ctx, conn, &msg); err != nil {
			h.sendError(conn, "message_handling_failed", err.Error())
		}
	}
}

// handleMessage 处理不同类型的消息
func (h *RemoteAgentHandler) handleMessage(ctx context.Context, conn *websocket.Conn, msg *RemoteAgentMessage) error {
	switch msg.Type {
	case "register":
		return h.handleRegister(ctx, conn, msg.Payload)
	case "register_session":
		return h.handleRegisterSession(ctx, conn, msg.Payload)
	case "event":
		return h.handleEvent(ctx, msg.Payload)
	case "unregister":
		return h.handleUnregister(ctx, msg.Payload)
	case "ping":
		h.sendMessage(conn, "pong", nil)
		return nil
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

// handleRegister 处理 Agent 注册
func (h *RemoteAgentHandler) handleRegister(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) error {
	var reg RegisterPayload
	if err := json.Unmarshal(payload, &reg); err != nil {
		return fmt.Errorf("failed to parse register payload: %w", err)
	}

	// 验证必填字段
	if reg.AgentID == "" {
		return fmt.Errorf("agent_id is required")
	}
	if reg.TemplateID == "" {
		return fmt.Errorf("template_id is required")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// 检查是否已注册
	if _, exists := h.agents[reg.AgentID]; exists {
		return fmt.Errorf("agent %s already registered", reg.AgentID)
	}

	// 创建 RemoteAgent
	remoteAgent := agent.NewRemoteAgent(reg.AgentID, reg.TemplateID, reg.Metadata)

	// 注册到本地 map
	h.agents[reg.AgentID] = remoteAgent
	h.conns[reg.AgentID] = conn

	// 注册到 RuntimeAgentRegistry，使 Dashboard 可以获取 EventBus
	if h.registry != nil {
		h.registry.RegisterRemoteAgent(remoteAgent)
	}

	// 同步到 Store，使其在 /v1/agents API 中可见
	if h.store != nil {
		// 添加 remote agent 标记到 metadata
		metadata := reg.Metadata
		if metadata == nil {
			metadata = make(map[string]any)
		}
		metadata["remote"] = true
		metadata["source"] = "remote_agent"

		record := &AgentRecord{
			ID: reg.AgentID,
			Config: &types.AgentConfig{
				AgentID:    reg.AgentID,
				TemplateID: reg.TemplateID,
				Metadata:   metadata,
			},
			Status:    "active",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Metadata:  metadata,
		}

		if err := (*h.store).Set(ctx, "agents", reg.AgentID, record); err != nil {
			logging.Error(ctx, "remote_agent.store.save.error", map[string]any{
				"agent_id": reg.AgentID,
				"error":    err.Error(),
			})
			// 不返回错误，继续注册流程
		} else {
			logging.Info(ctx, "remote_agent.store.saved", map[string]any{
				"agent_id": reg.AgentID,
			})
		}
	}

	logging.Info(ctx, "remote_agent.registered", map[string]any{
		"agent_id":    reg.AgentID,
		"template_id": reg.TemplateID,
		"metadata":    reg.Metadata,
	})

	// 发送确认消息
	h.sendMessage(conn, "registered", map[string]any{
		"agent_id": reg.AgentID,
		"message":  "Agent registered successfully",
	})

	return nil
}

// handleRegisterSession 处理 Session 注册
func (h *RemoteAgentHandler) handleRegisterSession(ctx context.Context, conn *websocket.Conn, payload json.RawMessage) error {
	var reg RegisterSessionPayload
	if err := json.Unmarshal(payload, &reg); err != nil {
		return fmt.Errorf("failed to parse register_session payload: %w", err)
	}

	// 验证必填字段
	if reg.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	if reg.AgentID == "" {
		return fmt.Errorf("agent_id is required")
	}

	// 同步到 Store，使其在 /v1/sessions API 中可见
	if h.store != nil {
		// 添加 remote session 标记到 metadata
		metadata := reg.Metadata
		if metadata == nil {
			metadata = make(map[string]any)
		}
		metadata["remote"] = true
		metadata["source"] = "remote_agent"
		metadata["message_count"] = reg.MessageCount

		status := reg.Status
		if status == "" {
			status = "active"
		}

		record := &SessionRecord{
			ID:        reg.SessionID,
			AgentID:   reg.AgentID,
			Status:    status,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Metadata:  metadata,
		}

		if err := (*h.store).Set(ctx, "sessions", reg.SessionID, record); err != nil {
			logging.Error(ctx, "remote_agent.session.store.save.error", map[string]any{
				"session_id": reg.SessionID,
				"agent_id":   reg.AgentID,
				"error":      err.Error(),
			})
			return fmt.Errorf("failed to save session: %w", err)
		}

		logging.Info(ctx, "remote_agent.session.registered", map[string]any{
			"session_id":    reg.SessionID,
			"agent_id":      reg.AgentID,
			"message_count": reg.MessageCount,
		})
	}

	// 发送确认消息
	h.sendMessage(conn, "session_registered", map[string]any{
		"session_id": reg.SessionID,
		"message":    "Session registered successfully",
	})

	return nil
}

// handleEvent 处理事件推送
func (h *RemoteAgentHandler) handleEvent(ctx context.Context, payload json.RawMessage) error {
	var evt EventPayload
	if err := json.Unmarshal(payload, &evt); err != nil {
		return fmt.Errorf("failed to parse event payload: %w", err)
	}

	logging.Info(ctx, "remote_agent.event.received", map[string]any{
		"agent_id":   evt.AgentID,
		"event_type": fmt.Sprintf("%T", evt.Envelope.Event),
	})

	h.mu.RLock()
	remoteAgent, exists := h.agents[evt.AgentID]
	h.mu.RUnlock()

	if !exists {
		return fmt.Errorf("agent %s not registered", evt.AgentID)
	}

	// 推送事件到 RemoteAgent 的 EventBus
	if err := remoteAgent.PushEvent(evt.Envelope); err != nil {
		return fmt.Errorf("failed to push event: %w", err)
	}

	logging.Info(ctx, "remote_agent.event.pushed", map[string]any{
		"agent_id": evt.AgentID,
	})

	return nil
}

// handleUnregister 处理 Agent 注销
func (h *RemoteAgentHandler) handleUnregister(ctx context.Context, payload json.RawMessage) error {
	var data struct {
		AgentID string `json:"agent_id"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("failed to parse unregister payload: %w", err)
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	remoteAgent, exists := h.agents[data.AgentID]
	if !exists {
		return fmt.Errorf("agent %s not registered", data.AgentID)
	}

	// 关闭 RemoteAgent
	_ = remoteAgent.Close()

	// 从 RuntimeAgentRegistry 中注销
	if h.registry != nil {
		h.registry.UnregisterRemoteAgent(data.AgentID)
	}

	// 从 Store 中删除
	if h.store != nil {
		if err := (*h.store).Delete(ctx, "agents", data.AgentID); err != nil {
			logging.Error(ctx, "remote_agent.store.delete.error", map[string]any{
				"agent_id": data.AgentID,
				"error":    err.Error(),
			})
		}
	}

	// 从本地 map 删除
	delete(h.agents, data.AgentID)
	delete(h.conns, data.AgentID)

	logging.Info(ctx, "remote_agent.unregistered", map[string]any{
		"agent_id": data.AgentID,
	})

	return nil
}

// cleanupConnection 清理连接相关的资源
func (h *RemoteAgentHandler) cleanupConnection(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ctx := context.Background()

	// 找到并清理所有使用此连接的 Agent
	for agentID, c := range h.conns {
		if c == conn {
			if remoteAgent, exists := h.agents[agentID]; exists {
				_ = remoteAgent.Close()
				delete(h.agents, agentID)
			}

			// 从 RuntimeAgentRegistry 中注销
			if h.registry != nil {
				h.registry.UnregisterRemoteAgent(agentID)
			}

			// 从 Store 中删除
			if h.store != nil {
				if err := (*h.store).Delete(ctx, "agents", agentID); err != nil {
					logging.Error(ctx, "remote_agent.store.delete.error", map[string]any{
						"agent_id": agentID,
						"error":    err.Error(),
					})
				}
			}

			delete(h.conns, agentID)

			logging.Info(ctx, "remote_agent.connection.cleanup", map[string]any{
				"agent_id": agentID,
			})
		}
	}
}

// pingLoop 定期发送 ping 消息保持连接
func (h *RemoteAgentHandler) pingLoop(conn *websocket.Conn, stop <-chan struct{}) {
	ticker := time.NewTicker(54 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-stop:
			return
		}
	}
}

// sendMessage 发送消息到客户端
func (h *RemoteAgentHandler) sendMessage(conn *websocket.Conn, msgType string, payload any) {
	msg := map[string]any{
		"type":    msgType,
		"payload": payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logging.Error(context.Background(), "remote_agent.marshal.error", map[string]any{
			"error": err.Error(),
		})
		return
	}

	_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		logging.Error(context.Background(), "remote_agent.send.error", map[string]any{
			"error": err.Error(),
		})
	}
}

// sendError 发送错误消息到客户端
func (h *RemoteAgentHandler) sendError(conn *websocket.Conn, code, message string) {
	h.sendMessage(conn, "error", map[string]any{
		"code":    code,
		"message": message,
	})
}

// GetStats 返回统计信息
func (h *RemoteAgentHandler) GetStats(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	agents := make([]map[string]any, 0, len(h.agents))
	for agentID, remoteAgent := range h.agents {
		agents = append(agents, map[string]any{
			"agent_id": agentID,
			"status":   remoteAgent.Status(),
			"metadata": remoteAgent.GetMetadata(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"remote_agents_count": len(h.agents),
			"agents":              agents,
		},
	})
}
