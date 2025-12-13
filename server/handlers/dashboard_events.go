package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/types"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// DashboardEventHandler handles WebSocket connections for dashboard event streaming
type DashboardEventHandler struct {
	registry *RuntimeAgentRegistry

	// Connection management
	connections map[string]*DashboardEventConnection
	mu          sync.RWMutex
}

// agentSubscription tracks subscription to a specific agent
type agentSubscription struct {
	agentID string
	eventCh <-chan types.AgentEventEnvelope
	ctx     context.Context
	cancel  context.CancelFunc
}

// DashboardEventConnection represents a single dashboard WebSocket connection
type DashboardEventConnection struct {
	ID            string
	Conn          *websocket.Conn
	Send          chan []byte
	ctx           context.Context
	cancel        context.CancelFunc
	filters       *EventStreamFilters
	subscriptions map[string]*agentSubscription // agentID -> subscription
	subMu         sync.Mutex
	handler       *DashboardEventHandler
}

// EventStreamFilters defines filtering options for event streaming
type EventStreamFilters struct {
	Channels   []string `json:"channels"`    // ["progress", "control", "monitor"]
	EventTypes []string `json:"event_types"` // ["token_usage", "tool_executed", "error"]
	AgentIDs   []string `json:"agent_ids"`   // Filter by specific agent IDs
	MinLevel   string   `json:"min_level"`   // "debug", "info", "warn", "error"
}

// EventStreamMessage represents a message sent over the event stream WebSocket
type EventStreamMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

// EventStreamSubscribeRequest represents a subscription request
type EventStreamSubscribeRequest struct {
	Action  string              `json:"action"`  // "subscribe", "unsubscribe"
	Filters *EventStreamFilters `json:"filters"` // Filtering options
}

// NewDashboardEventHandler creates a new DashboardEventHandler
func NewDashboardEventHandler(registry *RuntimeAgentRegistry) *DashboardEventHandler {
	h := &DashboardEventHandler{
		registry:    registry,
		connections: make(map[string]*DashboardEventConnection),
	}

	// Listen for agent registration/unregistration events
	if registry != nil {
		registry.AddListener(h.onAgentRegistryChange)
		registry.AddRemoteAgentListener(h.onRemoteAgentRegistryChange)
	}

	return h
}

// onAgentRegistryChange handles agent registration/unregistration events
func (h *DashboardEventHandler) onAgentRegistryChange(agentID string, ag *agent.Agent, registered bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Notify all connections about the change
	for _, wsConn := range h.connections {
		if registered && ag != nil {
			// Subscribe the connection to the new agent if filters match
			wsConn.subscribeToAgent(ag)
		} else {
			// Unsubscribe from the removed agent
			wsConn.unsubscribeFromAgent(agentID)
		}
	}
}

// onRemoteAgentRegistryChange handles remote agent registration/unregistration events
func (h *DashboardEventHandler) onRemoteAgentRegistryChange(agentID string, ra *agent.RemoteAgent, registered bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	logging.Info(context.Background(), "dashboard.remote_agent_change", map[string]any{
		"agent_id":         agentID,
		"registered":       registered,
		"connection_count": len(h.connections),
	})

	// Notify all connections about the change
	for _, wsConn := range h.connections {
		if registered && ra != nil {
			// Subscribe the connection to the new remote agent if filters match
			logging.Info(context.Background(), "dashboard.subscribing_to_remote_agent", map[string]any{
				"agent_id":      agentID,
				"connection_id": wsConn.ID,
			})
			wsConn.subscribeToRemoteAgent(ra)
		} else {
			// Unsubscribe from the removed agent
			wsConn.unsubscribeFromAgent(agentID)
		}
	}
}

// HandleEventStream handles WebSocket upgrade for event streaming
func (h *DashboardEventHandler) HandleEventStream(c *gin.Context) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logging.Error(c.Request.Context(), "dashboard.websocket.upgrade.failed", map[string]any{
			"error": err.Error(),
		})
		return
	}

	// Create connection context
	ctx, cancel := context.WithCancel(context.Background())

	// Create connection object
	wsConn := &DashboardEventConnection{
		ID:            fmt.Sprintf("dash-ws-%d", time.Now().UnixNano()),
		Conn:          conn,
		Send:          make(chan []byte, 256),
		ctx:           ctx,
		cancel:        cancel,
		subscriptions: make(map[string]*agentSubscription),
		handler:       h,
		filters: &EventStreamFilters{
			Channels: []string{"monitor"}, // Default to monitor channel only
		},
	}

	// Register connection
	h.mu.Lock()
	h.connections[wsConn.ID] = wsConn
	h.mu.Unlock()

	logging.Info(ctx, "dashboard.websocket.connected", map[string]any{
		"connection_id": wsConn.ID,
	})

	// Send welcome message
	h.sendMessage(wsConn, "connected", map[string]any{
		"connection_id": wsConn.ID,
		"message":       "Connected to Aster Studio event stream",
	})

	// Start write pump in goroutine
	go h.writePump(wsConn)

	// Run read pump in current goroutine (blocks until connection closes)
	h.readPump(wsConn)
}

// readPump reads messages from the WebSocket connection
func (h *DashboardEventHandler) readPump(wsConn *DashboardEventConnection) {
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
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logging.Error(wsConn.ctx, "dashboard.websocket.read.error", map[string]any{
					"connection_id": wsConn.ID,
					"error":         err.Error(),
				})
			}
			break
		}

		// Parse message
		var msg EventStreamSubscribeRequest
		if err := json.Unmarshal(message, &msg); err != nil {
			h.sendError(wsConn, "invalid_message", "Failed to parse message")
			continue
		}

		// Handle message
		h.handleMessage(wsConn, &msg)
	}
}

// writePump writes messages to the WebSocket connection
func (h *DashboardEventHandler) writePump(wsConn *DashboardEventConnection) {
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
func (h *DashboardEventHandler) handleMessage(wsConn *DashboardEventConnection, msg *EventStreamSubscribeRequest) {
	switch msg.Action {
	case "subscribe":
		h.handleSubscribe(wsConn, msg.Filters)
	case "unsubscribe":
		h.handleUnsubscribe(wsConn)
	case "ping":
		h.sendMessage(wsConn, "pong", nil)
	default:
		h.sendError(wsConn, "unknown_action", fmt.Sprintf("Unknown action: %s", msg.Action))
	}
}

// handleSubscribe handles subscription requests
func (h *DashboardEventHandler) handleSubscribe(wsConn *DashboardEventConnection, filters *EventStreamFilters) {
	// Unsubscribe from all previous subscriptions
	wsConn.unsubscribeAll()

	// Update filters
	if filters != nil {
		wsConn.filters = filters
	}

	// Subscribe to all existing agents (local and remote)
	if h.registry != nil {
		// Local agents
		for _, ag := range h.registry.List() {
			wsConn.subscribeToAgent(ag)
		}
		// Remote agents
		for _, ra := range h.registry.ListRemoteAgents() {
			wsConn.subscribeToRemoteAgent(ra)
		}
	}

	// Send confirmation
	h.sendMessage(wsConn, "subscribed", map[string]any{
		"channels":    wsConn.filters.Channels,
		"event_types": wsConn.filters.EventTypes,
		"agent_ids":   wsConn.filters.AgentIDs,
		"agent_count": len(wsConn.subscriptions),
	})

	logging.Info(wsConn.ctx, "dashboard.websocket.subscribed", map[string]any{
		"connection_id": wsConn.ID,
		"channels":      wsConn.filters.Channels,
		"agent_count":   len(wsConn.subscriptions),
	})
}

// handleUnsubscribe handles unsubscription requests
func (h *DashboardEventHandler) handleUnsubscribe(wsConn *DashboardEventConnection) {
	wsConn.unsubscribeAll()

	h.sendMessage(wsConn, "unsubscribed", nil)

	logging.Info(wsConn.ctx, "dashboard.websocket.unsubscribed", map[string]any{
		"connection_id": wsConn.ID,
	})
}

// subscribeToAgent subscribes this connection to an agent's event bus
func (c *DashboardEventConnection) subscribeToAgent(ag *agent.Agent) {
	if ag == nil {
		return
	}

	agentID := ag.ID()

	// Check if filters allow this agent
	if len(c.filters.AgentIDs) > 0 && !slices.Contains(c.filters.AgentIDs, agentID) {
		return
	}

	c.subMu.Lock()
	defer c.subMu.Unlock()

	// Already subscribed
	if _, exists := c.subscriptions[agentID]; exists {
		return
	}

	// Determine channels to subscribe
	var channels []types.AgentChannel
	if len(c.filters.Channels) > 0 {
		for _, ch := range c.filters.Channels {
			switch strings.ToLower(ch) {
			case "progress":
				channels = append(channels, types.ChannelProgress)
			case "control":
				channels = append(channels, types.ChannelControl)
			case "monitor":
				channels = append(channels, types.ChannelMonitor)
			}
		}
	} else {
		// Default to all channels
		channels = []types.AgentChannel{types.ChannelProgress, types.ChannelControl, types.ChannelMonitor}
	}

	// Subscribe to agent's event bus
	eventCh := ag.Subscribe(channels, nil)

	// Create subscription context
	subCtx, subCancel := context.WithCancel(c.ctx)

	sub := &agentSubscription{
		agentID: agentID,
		eventCh: eventCh,
		ctx:     subCtx,
		cancel:  subCancel,
	}

	c.subscriptions[agentID] = sub

	// Start forwarding events from this agent
	go c.forwardAgentEvents(ag, sub)
}

// subscribeToRemoteAgent subscribes this connection to a remote agent's event bus
func (c *DashboardEventConnection) subscribeToRemoteAgent(ra *agent.RemoteAgent) {
	if ra == nil {
		return
	}

	agentID := ra.ID()

	// Check if filters allow this agent
	if len(c.filters.AgentIDs) > 0 && !slices.Contains(c.filters.AgentIDs, agentID) {
		return
	}

	c.subMu.Lock()
	defer c.subMu.Unlock()

	// Already subscribed
	if _, exists := c.subscriptions[agentID]; exists {
		return
	}

	// Determine channels to subscribe
	var channels []types.AgentChannel
	if len(c.filters.Channels) > 0 {
		for _, ch := range c.filters.Channels {
			switch strings.ToLower(ch) {
			case "progress":
				channels = append(channels, types.ChannelProgress)
			case "control":
				channels = append(channels, types.ChannelControl)
			case "monitor":
				channels = append(channels, types.ChannelMonitor)
			}
		}
	} else {
		// Default to all channels
		channels = []types.AgentChannel{types.ChannelProgress, types.ChannelControl, types.ChannelMonitor}
	}

	// Subscribe to remote agent's event bus
	eventCh := ra.Subscribe(channels, nil)

	// Create subscription context
	subCtx, subCancel := context.WithCancel(c.ctx)

	sub := &agentSubscription{
		agentID: agentID,
		eventCh: eventCh,
		ctx:     subCtx,
		cancel:  subCancel,
	}

	c.subscriptions[agentID] = sub

	// Start forwarding events from this remote agent
	go c.forwardRemoteAgentEvents(ra, sub)
}

// forwardRemoteAgentEvents forwards events from a remote agent to the WebSocket connection
func (c *DashboardEventConnection) forwardRemoteAgentEvents(ra *agent.RemoteAgent, sub *agentSubscription) {
	defer ra.Unsubscribe(sub.eventCh)

	logging.Info(context.Background(), "dashboard.forward_remote_events.started", map[string]any{
		"agent_id":      sub.agentID,
		"connection_id": c.ID,
	})

	for {
		select {
		case envelope, ok := <-sub.eventCh:
			if !ok {
				logging.Info(context.Background(), "dashboard.forward_remote_events.channel_closed", map[string]any{
					"agent_id": sub.agentID,
				})
				return
			}

			logging.Debug(context.Background(), "dashboard.forward_remote_events.received", map[string]any{
				"agent_id":   sub.agentID,
				"event_type": fmt.Sprintf("%T", envelope.Event),
			})

			// Apply filters
			if !c.shouldForward(envelope) {
				logging.Debug(context.Background(), "dashboard.forward_remote_events.filtered", map[string]any{
					"agent_id": sub.agentID,
				})
				continue
			}

			// Extract event info
			eventInfo := c.extractEventInfo(sub.agentID, envelope)

			logging.Debug(context.Background(), "dashboard.forward_remote_events.sending", map[string]any{
				"agent_id": sub.agentID,
				"info":     eventInfo,
			})

			// Send event to client
			c.handler.sendMessage(c, "event", eventInfo)

		case <-sub.ctx.Done():
			return
		case <-c.ctx.Done():
			return
		}
	}
}

// unsubscribeFromAgent unsubscribes from a specific agent
func (c *DashboardEventConnection) unsubscribeFromAgent(agentID string) {
	c.subMu.Lock()
	defer c.subMu.Unlock()

	if sub, exists := c.subscriptions[agentID]; exists {
		sub.cancel()
		delete(c.subscriptions, agentID)
	}
}

// unsubscribeAll unsubscribes from all agents
func (c *DashboardEventConnection) unsubscribeAll() {
	c.subMu.Lock()
	defer c.subMu.Unlock()

	for _, sub := range c.subscriptions {
		sub.cancel()
	}
	c.subscriptions = make(map[string]*agentSubscription)
}

// forwardAgentEvents forwards events from an agent to the WebSocket connection
func (c *DashboardEventConnection) forwardAgentEvents(ag *agent.Agent, sub *agentSubscription) {
	defer ag.Unsubscribe(sub.eventCh)

	for {
		select {
		case envelope, ok := <-sub.eventCh:
			if !ok {
				return
			}

			// Apply filters
			if !c.shouldForward(envelope) {
				continue
			}

			// Extract event info
			eventInfo := c.extractEventInfo(sub.agentID, envelope)

			// Send event to client
			c.handler.sendMessage(c, "event", eventInfo)

		case <-sub.ctx.Done():
			return
		case <-c.ctx.Done():
			return
		}
	}
}

// shouldForward checks if an event should be forwarded based on filters
func (c *DashboardEventConnection) shouldForward(envelope types.AgentEventEnvelope) bool {
	if c.filters == nil {
		return true
	}

	// Extract event type and channel
	eventType := ""
	channel := ""

	if ev, ok := envelope.Event.(types.EventType); ok {
		eventType = ev.EventType()
		channel = string(ev.Channel())
	} else if eventMap, ok := envelope.Event.(map[string]any); ok {
		// 处理远程 Agent 发送的 map[string]any 类型事件
		if et, ok := eventMap["type"].(string); ok {
			eventType = et
		} else if et, ok := eventMap["event_type"].(string); ok {
			eventType = et
		}
		if ch, ok := eventMap["channel"].(string); ok {
			channel = ch
		}
	}

	// Filter by channels
	if len(c.filters.Channels) > 0 && channel != "" {
		found := false
		for _, ch := range c.filters.Channels {
			if strings.EqualFold(ch, channel) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by event types
	if len(c.filters.EventTypes) > 0 && eventType != "" {
		found := false
		for _, et := range c.filters.EventTypes {
			if strings.EqualFold(et, eventType) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// extractEventInfo extracts event information for sending to client
func (c *DashboardEventConnection) extractEventInfo(agentID string, envelope types.AgentEventEnvelope) map[string]any {
	// 将 Unix 秒级时间戳转换为 ISO 8601 格式字符串
	timestamp := time.Unix(envelope.Bookmark.Timestamp, 0).Format(time.RFC3339)

	info := map[string]any{
		"cursor":    envelope.Cursor,
		"timestamp": timestamp,
		"agent_id":  agentID,
	}

	// 处理类型化事件
	if ev, ok := envelope.Event.(types.EventType); ok {
		info["channel"] = string(ev.Channel())
		info["type"] = ev.EventType()
	} else if eventMap, ok := envelope.Event.(map[string]any); ok {
		// 处理从 JSON 反序列化的事件（远程 Agent 发送的事件）
		if ch, ok := eventMap["channel"].(string); ok {
			info["channel"] = ch
		}
		if et, ok := eventMap["type"].(string); ok {
			info["type"] = et
		} else if et, ok := eventMap["event_type"].(string); ok {
			info["type"] = et
		}
		// 直接使用 eventMap 作为 data，前端可以直接展示
		info["data"] = eventMap
		return info
	}

	// Add event-specific data based on actual event structures
	switch e := envelope.Event.(type) {
	case *types.MonitorTokenUsageEvent:
		info["data"] = map[string]any{
			"agent_id":      agentID,
			"input_tokens":  e.InputTokens,
			"output_tokens": e.OutputTokens,
			"total_tokens":  e.TotalTokens,
		}
	case *types.MonitorToolExecutedEvent:
		info["data"] = map[string]any{
			"agent_id":  agentID,
			"tool_id":   e.Call.ID,
			"tool_name": e.Call.Name,
			"state":     string(e.Call.State),
			"progress":  e.Call.Progress,
			"error":     e.Call.Error,
			"arguments": e.Call.Arguments,
			"result":    e.Call.Result,
		}
	case *types.MonitorStepCompleteEvent:
		info["data"] = map[string]any{
			"agent_id":    agentID,
			"step":        e.Step,
			"duration_ms": e.DurationMs,
		}
	case *types.MonitorErrorEvent:
		info["data"] = map[string]any{
			"agent_id": agentID,
			"severity": e.Severity,
			"phase":    e.Phase,
			"message":  e.Message,
			"detail":   e.Detail,
		}
	case *types.MonitorStateChangedEvent:
		info["data"] = map[string]any{
			"agent_id": agentID,
			"state":    string(e.State),
		}
	case *types.ProgressTextChunkEvent:
		info["data"] = map[string]any{
			"step":  e.Step,
			"delta": e.Delta,
		}
	case *types.ProgressTextChunkStartEvent:
		info["data"] = map[string]any{
			"step": e.Step,
		}
	case *types.ProgressTextChunkEndEvent:
		info["data"] = map[string]any{
			"step": e.Step,
			"text": e.Text,
		}
	case *types.ProgressThinkChunkEvent:
		info["data"] = map[string]any{
			"step":      e.Step,
			"delta":     e.Delta,
			"stage":     e.Stage,
			"reasoning": e.Reasoning,
			"decision":  e.Decision,
		}
	case *types.ProgressToolStartEvent:
		info["data"] = map[string]any{
			"tool_id":   e.Call.ID,
			"tool_name": e.Call.Name,
			"arguments": e.Call.Arguments,
		}
	case *types.ProgressToolEndEvent:
		info["data"] = map[string]any{
			"tool_id": e.Call.ID,
			"state":   string(e.Call.State),
			"result":  e.Call.Result,
			"error":   e.Call.Error,
		}
	case *types.ProgressToolProgressEvent:
		info["data"] = map[string]any{
			"tool_id":  e.Call.ID,
			"progress": e.Progress,
			"message":  e.Message,
			"step":     e.Step,
			"total":    e.Total,
		}
	case *types.ProgressDoneEvent:
		info["data"] = map[string]any{
			"step":   e.Step,
			"reason": e.Reason,
		}
	case *types.ControlPermissionRequiredEvent:
		info["data"] = map[string]any{
			"tool_id":   e.Call.ID,
			"tool_name": e.Call.Name,
			"arguments": e.Call.Arguments,
		}
	case *types.ControlPermissionDecidedEvent:
		info["data"] = map[string]any{
			"call_id":    e.CallID,
			"decision":   e.Decision,
			"decided_by": e.DecidedBy,
			"note":       e.Note,
		}
	default:
		// For other events, include the raw event
		info["data"] = envelope.Event
	}

	return info
}

// sendMessage sends a message to the WebSocket client
func (h *DashboardEventHandler) sendMessage(wsConn *DashboardEventConnection, msgType string, payload any) {
	// Check if context is cancelled
	if wsConn.ctx.Err() != nil {
		return
	}

	msg := EventStreamMessage{
		Type:    msgType,
		Payload: payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logging.Error(wsConn.ctx, "dashboard.websocket.marshal.error", map[string]any{
			"error": err.Error(),
		})
		return
	}

	// Use defer recover to prevent panic when sending to closed channel
	defer func() {
		if r := recover(); r != nil {
			logging.Warn(wsConn.ctx, "dashboard.websocket.send.recovered", map[string]any{
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
		logging.Warn(wsConn.ctx, "dashboard.websocket.send.dropped", map[string]any{
			"connection_id": wsConn.ID,
			"message_type":  msgType,
		})
	}
}

// sendError sends an error message to the WebSocket client
func (h *DashboardEventHandler) sendError(wsConn *DashboardEventConnection, code, message string) {
	h.sendMessage(wsConn, "error", map[string]any{
		"code":    code,
		"message": message,
	})
}

// closeConnection closes a WebSocket connection
func (h *DashboardEventHandler) closeConnection(wsConn *DashboardEventConnection) {
	h.mu.Lock()
	delete(h.connections, wsConn.ID)
	h.mu.Unlock()

	// Unsubscribe from all agents
	wsConn.unsubscribeAll()

	wsConn.cancel()
	close(wsConn.Send)

	logging.Info(wsConn.ctx, "dashboard.websocket.disconnected", map[string]any{
		"connection_id": wsConn.ID,
	})
}

// GetStats returns dashboard WebSocket statistics
func (h *DashboardEventHandler) GetStats(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"active_connections": len(h.connections),
		},
	})
}

// BroadcastEvent broadcasts an event to all connected dashboard clients
func (h *DashboardEventHandler) BroadcastEvent(msgType string, payload any) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, wsConn := range h.connections {
		h.sendMessage(wsConn, msgType, payload)
	}
}
