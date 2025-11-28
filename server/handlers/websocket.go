package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools/builtin"
	"github.com/astercloud/aster/pkg/types"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development
		// TODO: Configure allowed origins in production
		return true
	},
}

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	store       *store.Store
	deps        *agent.Dependencies
	todoManager builtin.TodoManager
	registry    *RuntimeAgentRegistry

	// Connection management
	connections map[string]*WebSocketConnection
	mu          sync.RWMutex

	// HITL Manager for Human-in-the-Loop approval
	hitlManager *HITLManager
}

// WebSocketConnection represents a single WebSocket connection
type WebSocketConnection struct {
	ID     string
	Conn   *websocket.Conn
	Send   chan []byte
	Agent  *agent.Agent
	ctx    context.Context
	cancel context.CancelFunc
}

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload"`
}

// NewWebSocketHandler creates a new WebSocketHandler
func NewWebSocketHandler(st store.Store, deps *agent.Dependencies, reg *RuntimeAgentRegistry) *WebSocketHandler {
	if reg == nil {
		reg = NewRuntimeAgentRegistry()
	}
	h := &WebSocketHandler{
		store:       &st,
		deps:        deps,
		todoManager: builtin.GetGlobalTodoManager(),
		registry:    reg,
		connections: make(map[string]*WebSocketConnection),
	}
	// Initialize HITL Manager
	h.hitlManager = NewHITLManager(h)
	return h
}

// HandleWebSocket handles WebSocket upgrade and communication
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logging.Error(c.Request.Context(), "websocket.upgrade.failed", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Create connection context
	ctx, cancel := context.WithCancel(context.Background())

	// Create connection object
	wsConn := &WebSocketConnection{
		ID:     fmt.Sprintf("ws-%d", time.Now().UnixNano()),
		Conn:   conn,
		Send:   make(chan []byte, 256),
		ctx:    ctx,
		cancel: cancel,
	}

	// Register connection
	h.mu.Lock()
	h.connections[wsConn.ID] = wsConn
	h.mu.Unlock()

	logging.Info(ctx, "websocket.connected", map[string]interface{}{
		"connection_id": wsConn.ID,
	})

	// Start write pump in goroutine
	go h.writePump(wsConn)

	// Run read pump in current goroutine (blocks until connection closes)
	h.readPump(wsConn)
}

// readPump reads messages from the WebSocket connection
func (h *WebSocketHandler) readPump(wsConn *WebSocketConnection) {
	defer func() {
		h.closeConnection(wsConn)
	}()

	_ = wsConn.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	wsConn.Conn.SetPongHandler(func(string) error {
		_ = wsConn.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := wsConn.Conn.ReadMessage()
		if err != nil {
			logging.Info(wsConn.ctx, "websocket.read.closed", map[string]interface{}{
				"connection_id": wsConn.ID,
				"error":         err.Error(),
			})
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logging.Error(wsConn.ctx, "websocket.read.error", map[string]interface{}{
					"connection_id": wsConn.ID,
					"error":         err.Error(),
				})
			}
			break
		}

		logging.Info(wsConn.ctx, "websocket.message.received", map[string]interface{}{
			"connection_id": wsConn.ID,
			"message":       string(message),
		})

		// Parse message
		var msg WebSocketMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			h.sendError(wsConn, "invalid_message", "Failed to parse message")
			continue
		}

		// Handle message based on type
		h.handleMessage(wsConn, &msg)
	}
}

// writePump writes messages to the WebSocket connection
func (h *WebSocketHandler) writePump(wsConn *WebSocketConnection) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		_ = wsConn.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-wsConn.Send:
			_ = wsConn.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = wsConn.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := wsConn.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			_ = wsConn.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := wsConn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-wsConn.ctx.Done():
			return
		}
	}
}

// handleMessage handles different message types
func (h *WebSocketHandler) handleMessage(wsConn *WebSocketConnection, msg *WebSocketMessage) {
	switch msg.Type {
	case "chat":
		h.handleChat(wsConn, msg.Payload)
	case "ping":
		h.sendMessage(wsConn, "pong", nil)
	case "todo_list_request":
		h.handleTodoListRequest(wsConn, msg.Payload)
	case "todo_create":
		h.handleTodoCreate(wsConn, msg.Payload)
	case "todo_update":
		h.handleTodoUpdate(wsConn, msg.Payload)
	case "todo_delete":
		h.handleTodoDelete(wsConn, msg.Payload)
	case "tool:control", "tool_control":
		h.handleToolControl(wsConn, msg.Payload)
	case "permission_decision":
		h.handlePermissionDecision(wsConn, msg.Payload)
	default:
		h.sendError(wsConn, "unknown_message_type", fmt.Sprintf("Unknown message type: %s", msg.Type))
	}
}

// handleChat handles chat messages
func (h *WebSocketHandler) handleChat(wsConn *WebSocketConnection, payload map[string]interface{}) {
	// Reset read deadline to prevent timeout during long-running LLM requests
	_ = wsConn.Conn.SetReadDeadline(time.Now().Add(5 * time.Minute))

	// Extract parameters
	templateID, _ := payload["template_id"].(string)
	if templateID == "" {
		templateID = "chat"
	}

	input, _ := payload["input"].(string)
	if input == "" {
		h.sendError(wsConn, "missing_input", "Input message is required")
		return
	}

	var ag *agent.Agent
	var err error

	// Reuse existing agent if available, otherwise create new one
	if wsConn.Agent != nil {
		ag = wsConn.Agent
		logging.Info(wsConn.ctx, "websocket.agent.reused", map[string]interface{}{
			"agent_id": ag.ID(),
		})
	} else {
		// Create agent configuration
		cfg := &types.AgentConfig{
			TemplateID: templateID,
		}

		// Handle model config if provided
		if modelConfigData, ok := payload["model_config"].(map[string]interface{}); ok {
			modelConfig := &types.ModelConfig{}
			if provider, ok := modelConfigData["provider"].(string); ok {
				modelConfig.Provider = provider
			}
			if model, ok := modelConfigData["model"].(string); ok {
				modelConfig.Model = model
			}
			if apiKey, ok := modelConfigData["api_key"].(string); ok {
				modelConfig.APIKey = apiKey
			}

			// Fill API key from environment if missing
			if modelConfig.APIKey == "" && modelConfig.Provider != "" {
				var envKey string
				switch modelConfig.Provider {
				case "deepseek":
					envKey = os.Getenv("DEEPSEEK_API_KEY")
				case "anthropic":
					envKey = os.Getenv("ANTHROPIC_API_KEY")
				case "openai":
					envKey = os.Getenv("OPENAI_API_KEY")
				default:
					envKey = os.Getenv(strings.ToUpper(modelConfig.Provider) + "_API_KEY")
				}
				if envKey != "" {
					modelConfig.APIKey = envKey
				}
			}

			cfg.ModelConfig = modelConfig
		}

		// Create agent instance
		ag, err = agent.Create(wsConn.ctx, cfg, h.deps)
		if err != nil {
			h.sendError(wsConn, "agent_creation_failed", err.Error())
			return
		}
		wsConn.Agent = ag
		h.registry.Register(ag)
		logging.Info(wsConn.ctx, "websocket.agent.created", map[string]interface{}{
			"agent_id": ag.ID(),
		})

		// 只在创建新 Agent 时订阅事件（避免重复订阅）
		eventCh := ag.Subscribe(
			[]types.AgentChannel{
				types.ChannelProgress,
				types.ChannelControl,
				types.ChannelMonitor,
			},
			nil,
		)

		go func() {
			defer ag.Unsubscribe(eventCh)
			for {
				select {
				case envelope, ok := <-eventCh:
					if !ok {
						return
					}
					channel := ""
					eventType := ""
					if ev, ok := envelope.Event.(types.EventType); ok {
						channel = string(ev.Channel())
						eventType = ev.EventType()
					}
					h.sendMessage(wsConn, "agent_event", map[string]interface{}{
						"channel":  channel,
						"type":     eventType,
						"cursor":   envelope.Cursor,
						"bookmark": envelope.Bookmark,
						"event":    envelope.Event,
					})
				case <-wsConn.ctx.Done():
					return
				}
			}
		}()
	}

	// Send start event
	h.sendMessage(wsConn, "chat_start", map[string]interface{}{
		"agent_id": ag.ID(),
	})

	// 发送 think_chunk_start (思考开始)
	h.sendMessage(wsConn, "think_chunk_start", map[string]interface{}{
		"step": 0,
	})

	// Stream response
	// 注意: 不在这里关闭 Agent,让 Agent 在整个 WebSocket 连接期间保持活跃
	// Agent 会在 WebSocket 断开时由 handleDisconnect 关闭
	go func() {
		logging.Info(wsConn.ctx, "websocket.stream.starting", map[string]interface{}{
			"agent_id": ag.ID(),
			"input":    input,
		})

		reader := ag.Stream(wsConn.ctx, input)
		for {
			event, err := reader.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				logging.Error(wsConn.ctx, "stream.error", map[string]interface{}{
					"agent_id": ag.ID(),
					"error":    err.Error(),
				})
				h.sendError(wsConn, "stream_error", err.Error())
				break
			}

			if event != nil {
				logging.Info(wsConn.ctx, "websocket.stream.event.received", map[string]interface{}{
					"agent_id":      ag.ID(),
					"event_id":      event.ID,
					"author":        event.Author,
					"has_content":   len(event.Content.ContentBlocks) > 0,
					"has_reasoning": event.Reasoning != "",
				})

				// 处理推理内容 (Kimi thinking, DeepSeek reasoner)
				if event.Reasoning != "" {
					logging.Info(wsConn.ctx, "websocket.stream.sending_think_chunk", map[string]interface{}{
						"agent_id":         ag.ID(),
						"reasoning_length": len(event.Reasoning),
					})
					h.sendMessage(wsConn, "think_chunk", map[string]interface{}{
						"delta": event.Reasoning,
					})
				}

				// Extract text content from event
				var textContent string
				for _, block := range event.Content.ContentBlocks {
					if tb, ok := block.(*types.TextBlock); ok {
						textContent += tb.Text
					}
				}

				if textContent != "" {
					logging.Info(wsConn.ctx, "websocket.stream.sending_text_delta", map[string]interface{}{
						"agent_id":    ag.ID(),
						"text_length": len(textContent),
						"preview":     textContent[:min(50, len(textContent))],
					})
					h.sendMessage(wsConn, "text_delta", map[string]interface{}{
						"text": textContent,
					})
				}
			} else {
				logging.Info(wsConn.ctx, "websocket.stream.nil_event", map[string]interface{}{
					"agent_id": ag.ID(),
				})
			}
		}

		// 发送 think_chunk_end (思考结束)
		h.sendMessage(wsConn, "think_chunk_end", map[string]interface{}{
			"step": 0,
		})

		// Send completion event
		h.sendMessage(wsConn, "chat_complete", map[string]interface{}{
			"agent_id": ag.ID(),
		})

		logging.Info(wsConn.ctx, "chat.completed", map[string]interface{}{
			"agent_id": ag.ID(),
		})
	}()
}

// sendMessage sends a message to the WebSocket client
func (h *WebSocketHandler) sendMessage(wsConn *WebSocketConnection, msgType string, payload map[string]interface{}) {
	// 检查 context 是否已取消
	if wsConn.ctx.Err() != nil {
		return
	}

	msg := WebSocketMessage{
		Type:    msgType,
		Payload: payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logging.Error(wsConn.ctx, "websocket.marshal.error", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// 使用 defer recover 防止发送到已关闭的 channel 导致 panic
	defer func() {
		if r := recover(); r != nil {
			logging.Warn(wsConn.ctx, "websocket.send.recovered", map[string]interface{}{
				"connection_id": wsConn.ID,
				"message_type":  msgType,
				"panic":         r,
			})
		}
	}()

	select {
	case wsConn.Send <- data:
	case <-wsConn.ctx.Done():
	default:
		// Channel full, skip message
		logging.Warn(wsConn.ctx, "websocket.send.dropped", map[string]interface{}{
			"connection_id": wsConn.ID,
			"message_type":  msgType,
		})
	}
}

// sendError sends an error message to the WebSocket client
func (h *WebSocketHandler) sendError(wsConn *WebSocketConnection, code, message string) {
	h.sendMessage(wsConn, "error", map[string]interface{}{
		"code":    code,
		"message": message,
	})
}

// closeConnection closes a WebSocket connection
func (h *WebSocketHandler) closeConnection(wsConn *WebSocketConnection) {
	h.mu.Lock()
	delete(h.connections, wsConn.ID)
	h.mu.Unlock()

	// Cancel pending HITL requests for this connection
	if h.hitlManager != nil {
		h.hitlManager.CancelPendingRequests(wsConn.ID)
	}

	wsConn.cancel()
	close(wsConn.Send)

	if wsConn.Agent != nil {
		h.registry.Unregister((*wsConn.Agent).ID())
		if err := (*wsConn.Agent).Close(); err != nil {
			logging.Error(wsConn.ctx, "failed to close agent", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	logging.Info(wsConn.ctx, "websocket.disconnected", map[string]interface{}{
		"connection_id": wsConn.ID,
	})
}

// GetStats returns WebSocket statistics
func (h *WebSocketHandler) GetStats(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"active_connections": len(h.connections),
		},
	})
}

// handleTodoListRequest handles todo list requests
func (h *WebSocketHandler) handleTodoListRequest(wsConn *WebSocketConnection, payload map[string]interface{}) {
	listName := "default"
	if name, ok := payload["list_name"].(string); ok && name != "" {
		listName = name
	}

	todoList, err := h.todoManager.LoadTodoList(listName)
	if err != nil {
		// 如果不存在，返回空列表
		todoList = &builtin.TodoList{
			ID:        fmt.Sprintf("list_%s_%d", listName, time.Now().UnixNano()),
			Name:      listName,
			Todos:     []builtin.TodoItem{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Metadata:  make(map[string]interface{}),
		}
	}

	h.sendMessage(wsConn, "todo_list_response", map[string]interface{}{
		"todos": todoList.Todos,
	})
}

// handleTodoCreate handles todo creation
func (h *WebSocketHandler) handleTodoCreate(wsConn *WebSocketConnection, payload map[string]interface{}) {
	listName := "default"
	if name, ok := payload["list_name"].(string); ok && name != "" {
		listName = name
	}

	todoData, ok := payload["todo"].(map[string]interface{})
	if !ok {
		h.sendError(wsConn, "invalid_todo", "todo data is required")
		return
	}

	// 加载现有任务列表
	todoList, err := h.todoManager.LoadTodoList(listName)
	if err != nil {
		// 创建新的任务列表
		todoList = &builtin.TodoList{
			ID:        fmt.Sprintf("list_%s_%d", listName, time.Now().UnixNano()),
			Name:      listName,
			Todos:     []builtin.TodoItem{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Metadata:  make(map[string]interface{}),
		}
	}

	// 创建新的 TodoItem
	todo := builtin.TodoItem{
		ID:        fmt.Sprintf("todo_%d", time.Now().UnixNano()),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	if content, ok := todoData["content"].(string); ok {
		todo.Content = content
	}
	if completed, ok := todoData["completed"].(bool); ok {
		todo.Status = "pending"
		if completed {
			todo.Status = "completed"
			now := time.Now()
			todo.CompletedAt = &now
		}
		todo.ActiveForm = "任务完成"
	} else {
		todo.Status = "pending"
		todo.ActiveForm = "进行中"
	}
	if priority, ok := todoData["priority"].(string); ok {
		switch priority {
		case "high":
			todo.Priority = 3
		case "medium":
			todo.Priority = 2
		case "low":
			todo.Priority = 1
		default:
			todo.Priority = 0
		}
	}
	if dueDate, ok := todoData["dueDate"].(string); ok {
		todo.Metadata["dueDate"] = dueDate
	}

	// 添加到列表
	todoList.Todos = append(todoList.Todos, todo)
	todoList.UpdatedAt = time.Now()

	// 保存列表
	if err := h.todoManager.StoreTodoList(todoList); err != nil {
		h.sendError(wsConn, "save_failed", fmt.Sprintf("Failed to save todo list: %v", err))
		return
	}

	// 广播给所有连接
	h.broadcastToAll("todo_created", map[string]interface{}{
		"todo": todo,
	})
}

// handleTodoUpdate handles todo updates
func (h *WebSocketHandler) handleTodoUpdate(wsConn *WebSocketConnection, payload map[string]interface{}) {
	listName := "default"
	if name, ok := payload["list_name"].(string); ok && name != "" {
		listName = name
	}

	todoData, ok := payload["todo"].(map[string]interface{})
	if !ok {
		h.sendError(wsConn, "invalid_todo", "todo data is required")
		return
	}

	todoID, ok := todoData["id"].(string)
	if !ok {
		h.sendError(wsConn, "missing_todo_id", "todo id is required")
		return
	}

	// 加载任务列表
	todoList, err := h.todoManager.LoadTodoList(listName)
	if err != nil {
		h.sendError(wsConn, "list_not_found", fmt.Sprintf("Todo list '%s' not found", listName))
		return
	}

	// 查找并更新任务
	updated := false
	for i, existingTodo := range todoList.Todos {
		if existingTodo.ID == todoID {
			// 更新字段
			if content, ok := todoData["content"].(string); ok {
				todoList.Todos[i].Content = content
			}
			if completed, ok := todoData["completed"].(bool); ok {
				wasCompleted := existingTodo.Status == "completed"
				isCompleted := completed

				if isCompleted && !wasCompleted {
					todoList.Todos[i].Status = "completed"
					now := time.Now()
					todoList.Todos[i].CompletedAt = &now
					todoList.Todos[i].ActiveForm = "任务完成"
				} else if !isCompleted && wasCompleted {
					todoList.Todos[i].Status = "pending"
					todoList.Todos[i].CompletedAt = nil
					todoList.Todos[i].ActiveForm = "进行中"
				}
			}
			if priority, ok := todoData["priority"].(string); ok {
				switch priority {
				case "high":
					todoList.Todos[i].Priority = 3
				case "medium":
					todoList.Todos[i].Priority = 2
				case "low":
					todoList.Todos[i].Priority = 1
				default:
					todoList.Todos[i].Priority = 0
				}
			}
			if dueDate, ok := todoData["dueDate"].(string); ok {
				if todoList.Todos[i].Metadata == nil {
					todoList.Todos[i].Metadata = make(map[string]interface{})
				}
				todoList.Todos[i].Metadata["dueDate"] = dueDate
			}

			todoList.Todos[i].UpdatedAt = time.Now()
			updated = true
			break
		}
	}

	if !updated {
		h.sendError(wsConn, "todo_not_found", fmt.Sprintf("Todo with id '%s' not found", todoID))
		return
	}

	// 保存列表
	if err := h.todoManager.StoreTodoList(todoList); err != nil {
		h.sendError(wsConn, "save_failed", fmt.Sprintf("Failed to save todo list: %v", err))
		return
	}

	// 广播给所有连接
	h.broadcastToAll("todo_updated", map[string]interface{}{
		"todo": todoData,
	})
}

// handleTodoDelete handles todo deletion
func (h *WebSocketHandler) handleTodoDelete(wsConn *WebSocketConnection, payload map[string]interface{}) {
	listName := "default"
	if name, ok := payload["list_name"].(string); ok && name != "" {
		listName = name
	}

	todoID, ok := payload["id"].(string)
	if !ok {
		h.sendError(wsConn, "missing_todo_id", "todo id is required")
		return
	}

	// 加载任务列表
	todoList, err := h.todoManager.LoadTodoList(listName)
	if err != nil {
		h.sendError(wsConn, "list_not_found", fmt.Sprintf("Todo list '%s' not found", listName))
		return
	}

	// 查找并删除任务
	deleted := false
	for i, existingTodo := range todoList.Todos {
		if existingTodo.ID == todoID {
			todoList.Todos = append(todoList.Todos[:i], todoList.Todos[i+1:]...)
			todoList.UpdatedAt = time.Now()
			deleted = true
			break
		}
	}

	if !deleted {
		h.sendError(wsConn, "todo_not_found", fmt.Sprintf("Todo with id '%s' not found", todoID))
		return
	}

	// 保存列表
	if err := h.todoManager.StoreTodoList(todoList); err != nil {
		h.sendError(wsConn, "save_failed", fmt.Sprintf("Failed to save todo list: %v", err))
		return
	}

	// 广播给所有连接
	h.broadcastToAll("todo_deleted", map[string]interface{}{
		"id": todoID,
	})
}

// handleToolControl 处理工具控制指令
func (h *WebSocketHandler) handleToolControl(wsConn *WebSocketConnection, payload map[string]interface{}) {
	if wsConn.Agent == nil {
		h.sendError(wsConn, "agent_not_ready", "agent is not initialized")
		return
	}

	callID, _ := payload["tool_call_id"].(string)
	if callID == "" {
		callID, _ = payload["call_id"].(string)
	}
	action, _ := payload["action"].(string)
	note, _ := payload["note"].(string)

	if callID == "" || action == "" {
		h.sendError(wsConn, "invalid_control", "tool_call_id and action are required")
		return
	}

	if err := wsConn.Agent.ControlTool(callID, action, note); err != nil {
		h.sendError(wsConn, "control_failed", err.Error())
		return
	}
}

// broadcastToAll broadcasts a message to all connected WebSocket clients
func (h *WebSocketHandler) broadcastToAll(messageType string, payload map[string]interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, wsConn := range h.connections {
		h.sendMessage(wsConn, messageType, payload)
	}
}
