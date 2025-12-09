package a2a

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/astercloud/aster/pkg/actor"
	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/logging"
)

var a2aLog = logging.ForComponent("A2AServer")

// Server A2A 服务器
// 将 Actor 系统暴露为 A2A 协议端点
type Server struct {
	actorSystem *actor.System
	taskStore   TaskStore
}

// NewServer 创建 A2A 服务器
func NewServer(actorSystem *actor.System, taskStore TaskStore) *Server {
	if taskStore == nil {
		taskStore = NewInMemoryTaskStore()
	}

	return &Server{
		actorSystem: actorSystem,
		taskStore:   taskStore,
	}
}

// HandleRequest 处理 JSON-RPC 请求
func (s *Server) HandleRequest(ctx context.Context, agentID string, req *JSONRPCRequest) *JSONRPCResponse {
	switch req.Method {
	case "message/send":
		return s.handleMessageSend(ctx, agentID, req)
	case "message/stream":
		return s.handleMessageStream(ctx, agentID, req)
	case "tasks/get":
		return s.handleTasksGet(ctx, agentID, req)
	case "tasks/cancel":
		return s.handleTasksCancel(ctx, agentID, req)
	default:
		return NewErrorResponse(req.ID, ErrorCodeMethodNotFound,
			fmt.Sprintf("method not found: %s", req.Method), nil)
	}
}

// GetAgentCard 获取 Agent Card
func (s *Server) GetAgentCard(agentID string) (*AgentCard, error) {
	// 从 Actor 系统获取 Agent
	_, exists := s.actorSystem.GetActor(agentID)
	if !exists {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	// 构建 Agent Card (使用静态信息)
	// TODO: 未来可以从 Agent 的元数据中动态获取这些信息
	card := &AgentCard{
		Name:        agentID,
		Description: fmt.Sprintf("Aster AI Agent: %s", agentID),
		URL:         fmt.Sprintf("/a2a/%s", agentID),
		Provider: Provider{
			Organization: "Aster",
			URL:          "https://github.com/astercloud/aster",
		},
		Version: "1.0",
		Capabilities: Capabilities{
			Streaming:              true,
			PushNotifications:      false,
			StateTransitionHistory: false,
		},
		DefaultInputModes:  []string{"text"},
		DefaultOutputModes: []string{"text"},
		Skills: []Skill{
			{
				Name:        "chat",
				Description: "General conversation and assistance",
			},
		},
	}

	return card, nil
}

// handleMessageSend 处理 message/send 方法
func (s *Server) handleMessageSend(ctx context.Context, agentID string, req *JSONRPCRequest) *JSONRPCResponse {
	// 解析参数
	var params MessageSendParams
	if err := parseParams(req.Params, &params); err != nil {
		return NewErrorResponse(req.ID, ErrorCodeInvalidParams, err.Error(), nil)
	}

	// 获取或创建 Task
	taskID := params.Message.TaskID
	if taskID == "" {
		taskID = generateID()
	}

	task, err := s.loadOrCreateTask(agentID, taskID, params.ContextID, params.Metadata)
	if err != nil {
		return NewErrorResponse(req.ID, ErrorCodeInternalError, err.Error(), nil)
	}

	// 添加用户消息到历史
	task.AddMessage(params.Message)

	// 更新状态为 working
	task.UpdateStatus(TaskStateWorking, nil)
	if err := s.taskStore.Save(agentID, task); err != nil {
		return NewErrorResponse(req.ID, ErrorCodeInternalError, err.Error(), nil)
	}

	// 通过 Actor 系统发送消息
	pid, exists := s.actorSystem.GetActor(agentID)
	if !exists {
		task.UpdateStatus(TaskStateFailed, &Message{
			MessageID: generateID(),
			Role:      "agent",
			Parts:     []Part{{Kind: "text", Text: fmt.Sprintf("agent not found: %s", agentID)}},
			Kind:      "message",
		})
		if err := s.taskStore.Save(agentID, task); err != nil {
			a2aLog.Warn(ctx, "save task error", map[string]any{"error": err})
		}
		return NewErrorResponse(req.ID, ErrorCodeInternalError, fmt.Sprintf("agent not found: %s", agentID), nil)
	}

	// 提取文本内容
	text := extractText(params.Message.Parts)

	// 发送到 Agent Actor
	result, err := pid.Request(&agent.ChatMsg{
		Text:    text,
		Ctx:     ctx,
		ReplyTo: nil,
	}, 30*time.Second)

	if err != nil {
		// 失败
		task.UpdateStatus(TaskStateFailed, &Message{
			MessageID: generateID(),
			Role:      "agent",
			Parts:     []Part{{Kind: "text", Text: fmt.Sprintf("agent error: %v", err)}},
			Kind:      "message",
		})
		if err := s.taskStore.Save(agentID, task); err != nil {
			a2aLog.Warn(ctx, "save task error", map[string]any{"error": err})
		}
		return NewErrorResponse(req.ID, ErrorCodeInternalError, err.Error(), nil)
	}

	// 成功
	chatResult, ok := result.(*agent.ChatResultMsg)
	if !ok {
		task.UpdateStatus(TaskStateFailed, &Message{
			MessageID: generateID(),
			Role:      "agent",
			Parts:     []Part{{Kind: "text", Text: "invalid response type"}},
			Kind:      "message",
		})
		if err := s.taskStore.Save(agentID, task); err != nil {
			a2aLog.Warn(ctx, "save task error", map[string]any{"error": err})
		}
		return NewErrorResponse(req.ID, ErrorCodeInternalError, "invalid response type", nil)
	}

	// 构建响应消息
	responseMsg := Message{
		MessageID: generateID(),
		Role:      "agent",
		Parts:     []Part{{Kind: "text", Text: chatResult.Result.Text}},
		Kind:      "message",
		TaskID:    taskID,
	}

	task.AddMessage(responseMsg)
	task.UpdateStatus(TaskStateCompleted, &responseMsg)
	if err := s.taskStore.Save(agentID, task); err != nil {
		a2aLog.Warn(ctx, "save task error", map[string]any{"error": err})
	}

	return NewSuccessResponse(req.ID, &MessageSendResult{TaskID: taskID})
}

// handleMessageStream 处理 message/stream 方法
func (s *Server) handleMessageStream(_ context.Context, _ string, req *JSONRPCRequest) *JSONRPCResponse {
	// 流式响应需要特殊处理，这里先返回不支持
	return NewErrorResponse(req.ID, ErrorCodeUnsupportedOperation,
		"streaming not yet implemented", nil)
}

// handleTasksGet 处理 tasks/get 方法
func (s *Server) handleTasksGet(_ context.Context, agentID string, req *JSONRPCRequest) *JSONRPCResponse {
	var params TasksGetParams
	if err := parseParams(req.Params, &params); err != nil {
		return NewErrorResponse(req.ID, ErrorCodeInvalidParams, err.Error(), nil)
	}

	task, err := s.taskStore.Load(agentID, params.TaskID)
	if err != nil {
		return NewErrorResponse(req.ID, ErrorCodeTaskNotFound, err.Error(), nil)
	}

	return NewSuccessResponse(req.ID, &TasksGetResult{Task: task})
}

// handleTasksCancel 处理 tasks/cancel 方法
func (s *Server) handleTasksCancel(_ context.Context, agentID string, req *JSONRPCRequest) *JSONRPCResponse {
	var params TasksCancelParams
	if err := parseParams(req.Params, &params); err != nil {
		return NewErrorResponse(req.ID, ErrorCodeInvalidParams, err.Error(), nil)
	}

	task, err := s.taskStore.Load(agentID, params.TaskID)
	if err != nil {
		return NewErrorResponse(req.ID, ErrorCodeTaskNotFound, err.Error(), nil)
	}

	// 检查是否可以取消
	if task.IsFinalState() {
		return NewSuccessResponse(req.ID, &TasksCancelResult{
			Success: false,
			Message: "Task is already in final state",
		})
	}

	// 设置取消信号
	s.taskStore.AddCancellation(params.TaskID)

	// 更新任务状态
	task.UpdateStatus(TaskStateCanceled, &Message{
		MessageID: generateID(),
		Role:      "agent",
		Parts:     []Part{{Kind: "text", Text: "Task cancelled by request."}},
		Kind:      "message",
	})

	if err := s.taskStore.Save(agentID, task); err != nil {
		a2aLog.Warn(context.Background(), "save task error", map[string]any{"error": err})
	}

	return NewSuccessResponse(req.ID, &TasksCancelResult{
		Success: true,
		Message: "Task cancelled successfully",
	})
}

// loadOrCreateTask 加载或创建任务
func (s *Server) loadOrCreateTask(agentID, taskID, contextID string, metadata Metadata) (*Task, error) {
	task, err := s.taskStore.Load(agentID, taskID)
	if err == nil {
		// 任务已存在
		if task.IsFinalState() {
			// 已完成的任务收到新消息，重启任务
			task.UpdateStatus(TaskStateSubmitted, nil)
		} else if task.Status.State == TaskStateInputRequired {
			// 需要输入的任务收到消息，继续工作
			task.UpdateStatus(TaskStateWorking, nil)
		}
		return task, nil
	}

	// 创建新任务
	if contextID == "" {
		contextID = generateID()
	}

	task = NewTask(taskID, contextID)
	if metadata != nil {
		task.Metadata = metadata
	}

	return task, nil
}

// ============== 辅助函数 ==============

// parseParams 解析 JSON-RPC 参数
func parseParams(params any, target any) error {
	if params == nil {
		return fmt.Errorf("params is required")
	}

	// 先序列化再反序列化，确保类型正确
	data, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal params: %w", err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("unmarshal params: %w", err)
	}

	return nil
}

// extractText 从 Parts 中提取文本
func extractText(parts []Part) string {
	for _, part := range parts {
		if part.Kind == "text" {
			return part.Text
		}
	}
	return ""
}

// generateID 生成唯一 ID
func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		a2aLog.Warn(context.Background(), "generate ID error", map[string]any{"error": err})
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
