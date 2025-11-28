package handlers

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/middleware"
)

// HITLManager 管理 Human-in-the-Loop 审批流程
type HITLManager struct {
	// 待审批请求
	pendingRequests map[string]*HITLRequest
	// 请求通道 (用于等待审批结果)
	requestChannels map[string]chan *HITLDecision
	mu              sync.RWMutex

	// WebSocket 连接管理器 (用于发送审批请求)
	wsHandler *WebSocketHandler
}

// HITLRequest 审批请求
type HITLRequest struct {
	ID         string                 `json:"id"`
	ToolCallID string                 `json:"tool_call_id"`
	ToolName   string                 `json:"tool_name"`
	ToolInput  map[string]interface{} `json:"tool_input"`
	Message    string                 `json:"message"`
	CreatedAt  time.Time              `json:"created_at"`
	AgentID    string                 `json:"agent_id"`
	ConnID     string                 `json:"conn_id"` // WebSocket 连接 ID
}

// HITLDecision 审批决策
type HITLDecision struct {
	RequestID   string                 `json:"request_id"`
	Decision    string                 `json:"decision"` // approve, reject, edit
	EditedInput map[string]interface{} `json:"edited_input,omitempty"`
	Reason      string                 `json:"reason,omitempty"`
	DecidedAt   time.Time              `json:"decided_at"`
}

// NewHITLManager 创建 HITL 管理器
func NewHITLManager(wsHandler *WebSocketHandler) *HITLManager {
	return &HITLManager{
		pendingRequests: make(map[string]*HITLRequest),
		requestChannels: make(map[string]chan *HITLDecision),
		wsHandler:       wsHandler,
	}
}

// CreateApprovalHandler 创建用于 HITL 中间件的审批处理器
func (m *HITLManager) CreateApprovalHandler(connID string) middleware.ApprovalHandler {
	return func(ctx context.Context, request *middleware.ReviewRequest) ([]middleware.Decision, error) {
		decisions := make([]middleware.Decision, len(request.ActionRequests))

		for i, action := range request.ActionRequests {
			// 创建审批请求
			hitlReq := &HITLRequest{
				ID:         fmt.Sprintf("hitl_%d", time.Now().UnixNano()),
				ToolCallID: action.ToolCallID,
				ToolName:   action.ToolName,
				ToolInput:  action.Input,
				Message:    action.Message,
				CreatedAt:  time.Now(),
				ConnID:     connID,
			}

			// 注册请求
			m.mu.Lock()
			m.pendingRequests[hitlReq.ID] = hitlReq
			ch := make(chan *HITLDecision, 1)
			m.requestChannels[hitlReq.ID] = ch
			m.mu.Unlock()

			// 发送审批请求到前端
			m.sendApprovalRequest(connID, hitlReq)

			// 等待审批结果
			select {
			case decision := <-ch:
				decisions[i] = m.convertDecision(decision)
			case <-ctx.Done():
				decisions[i] = middleware.Decision{
					Type:   middleware.DecisionReject,
					Reason: "Context cancelled",
				}
			case <-time.After(5 * time.Minute): // 5分钟超时
				decisions[i] = middleware.Decision{
					Type:   middleware.DecisionReject,
					Reason: "Approval timeout",
				}
			}

			// 清理
			m.mu.Lock()
			delete(m.pendingRequests, hitlReq.ID)
			delete(m.requestChannels, hitlReq.ID)
			m.mu.Unlock()
		}

		return decisions, nil
	}
}

// sendApprovalRequest 发送审批请求到前端
func (m *HITLManager) sendApprovalRequest(connID string, req *HITLRequest) {
	m.wsHandler.mu.RLock()
	conn, exists := m.wsHandler.connections[connID]
	m.wsHandler.mu.RUnlock()

	if !exists {
		log.Printf("[HITLManager] Connection %s not found", connID)
		return
	}

	payload := map[string]interface{}{
		"request_id":   req.ID,
		"tool_call_id": req.ToolCallID,
		"call": map[string]interface{}{
			"name":      req.ToolName,
			"arguments": req.ToolInput,
		},
		"message":    req.Message,
		"created_at": req.CreatedAt.Format(time.RFC3339),
	}

	m.wsHandler.sendMessage(conn, "permission_required", payload)
	log.Printf("[HITLManager] Sent approval request %s for tool %s", req.ID, req.ToolName)
}

// HandleDecision 处理前端发来的审批决策
func (m *HITLManager) HandleDecision(requestID string, decision string, editedInput map[string]interface{}, reason string) error {
	m.mu.RLock()
	ch, exists := m.requestChannels[requestID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("request %s not found or already processed", requestID)
	}

	hitlDecision := &HITLDecision{
		RequestID:   requestID,
		Decision:    decision,
		EditedInput: editedInput,
		Reason:      reason,
		DecidedAt:   time.Now(),
	}

	select {
	case ch <- hitlDecision:
		log.Printf("[HITLManager] Decision received for request %s: %s", requestID, decision)
		return nil
	default:
		return fmt.Errorf("failed to send decision for request %s", requestID)
	}
}

// convertDecision 转换决策类型
func (m *HITLManager) convertDecision(d *HITLDecision) middleware.Decision {
	var decisionType middleware.DecisionType
	switch d.Decision {
	case "approve":
		decisionType = middleware.DecisionApprove
	case "reject":
		decisionType = middleware.DecisionReject
	case "edit":
		decisionType = middleware.DecisionEdit
	default:
		decisionType = middleware.DecisionReject
	}

	return middleware.Decision{
		Type:        decisionType,
		EditedInput: d.EditedInput,
		Reason:      d.Reason,
	}
}

// GetPendingRequests 获取待审批请求列表
func (m *HITLManager) GetPendingRequests() []*HITLRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	requests := make([]*HITLRequest, 0, len(m.pendingRequests))
	for _, req := range m.pendingRequests {
		requests = append(requests, req)
	}
	return requests
}

// GetPendingRequestsForConn 获取指定连接的待审批请求
func (m *HITLManager) GetPendingRequestsForConn(connID string) []*HITLRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	requests := make([]*HITLRequest, 0)
	for _, req := range m.pendingRequests {
		if req.ConnID == connID {
			requests = append(requests, req)
		}
	}
	return requests
}

// CancelPendingRequests 取消指定连接的所有待审批请求
func (m *HITLManager) CancelPendingRequests(connID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, req := range m.pendingRequests {
		if req.ConnID == connID {
			if ch, exists := m.requestChannels[id]; exists {
				// 发送拒绝决策
				select {
				case ch <- &HITLDecision{
					RequestID: id,
					Decision:  "reject",
					Reason:    "Connection closed",
					DecidedAt: time.Now(),
				}:
				default:
				}
			}
			delete(m.pendingRequests, id)
			delete(m.requestChannels, id)
		}
	}
}

// ==================
// WebSocket Handler 扩展
// ==================

// handlePermissionDecision 处理审批决策消息
func (h *WebSocketHandler) handlePermissionDecision(wsConn *WebSocketConnection, payload map[string]interface{}) {
	if h.hitlManager == nil {
		h.sendError(wsConn, "hitl_not_enabled", "HITL manager not initialized")
		return
	}

	requestID, _ := payload["request_id"].(string)
	decision, _ := payload["decision"].(string)
	reason, _ := payload["reason"].(string)

	var editedInput map[string]interface{}
	if edited, ok := payload["edited_input"].(map[string]interface{}); ok {
		editedInput = edited
	}

	if requestID == "" || decision == "" {
		h.sendError(wsConn, "invalid_decision", "request_id and decision are required")
		return
	}

	if err := h.hitlManager.HandleDecision(requestID, decision, editedInput, reason); err != nil {
		h.sendError(wsConn, "decision_failed", err.Error())
		return
	}

	// 发送确认
	h.sendMessage(wsConn, "permission_decision_ack", map[string]interface{}{
		"request_id": requestID,
		"decision":   decision,
	})
}
