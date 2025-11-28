// Package a2a 实现 Agent-to-Agent 通信协议
// 基于 JSON-RPC 2.0 和 A2A 标准
package a2a

import "time"

// ============== Agent Card ==============

// AgentCard Agent 元数据卡片
// 用于 Agent 发现和能力声明
type AgentCard struct {
	Name               string       `json:"name"`
	Description        string       `json:"description"`
	URL                string       `json:"url"`
	Provider           Provider     `json:"provider"`
	Version            string       `json:"version"`
	Capabilities       Capabilities `json:"capabilities"`
	DefaultInputModes  []string     `json:"defaultInputModes"`
	DefaultOutputModes []string     `json:"defaultOutputModes"`
	Skills             []Skill      `json:"skills,omitempty"`
}

// Provider Agent 提供者信息
type Provider struct {
	Organization string `json:"organization"`
	URL          string `json:"url"`
}

// Capabilities Agent 能力声明
type Capabilities struct {
	Streaming              bool `json:"streaming"`
	PushNotifications      bool `json:"pushNotifications"`
	StateTransitionHistory bool `json:"stateTransitionHistory"`
}

// Skill Agent 技能定义
type Skill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags,omitempty"`
}

// ============== Task ==============

// Task 任务对象
// 跟踪 Agent 交互的完整生命周期
type Task struct {
	ID        string     `json:"id"`
	ContextID string     `json:"contextId"`
	Status    TaskStatus `json:"status"`
	Artifacts []Artifact `json:"artifacts,omitempty"`
	History   []Message  `json:"history,omitempty"`
	Metadata  Metadata   `json:"metadata,omitempty"`
	Kind      string     `json:"kind"` // 固定为 "task"
}

// TaskStatus 任务状态
type TaskStatus struct {
	State     TaskState `json:"state"`
	Timestamp string    `json:"timestamp"` // ISO 8601 格式
	Message   *Message  `json:"message,omitempty"`
}

// TaskState 任务状态枚举
type TaskState string

const (
	TaskStateSubmitted     TaskState = "submitted"
	TaskStateWorking       TaskState = "working"
	TaskStateCompleted     TaskState = "completed"
	TaskStateFailed        TaskState = "failed"
	TaskStateCanceled      TaskState = "canceled"
	TaskStateInputRequired TaskState = "input-required"
)

// Artifact 任务产出物
type Artifact struct {
	Name  string `json:"name"`
	Parts []Part `json:"parts"`
}

// Metadata 任务元数据
type Metadata map[string]interface{}

// ============== Message ==============

// Message 消息对象
type Message struct {
	MessageID        string   `json:"messageId"`
	Role             string   `json:"role"` // "user" 或 "agent"
	Parts            []Part   `json:"parts"`
	Kind             string   `json:"kind"` // 固定为 "message"
	ContextID        string   `json:"contextId,omitempty"`
	TaskID           string   `json:"taskId,omitempty"`
	ReferenceTaskIDs []string `json:"referenceTaskIds,omitempty"`
}

// Part 消息部分
// 支持 text、file、data 三种类型
type Part struct {
	Kind string      `json:"kind"` // "text", "file", "data"
	Text string      `json:"text,omitempty"`
	Data interface{} `json:"data,omitempty"`
	// File 相关字段
	Name     string `json:"name,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	URL      string `json:"url,omitempty"`
}

// ============== JSON-RPC 2.0 ==============

// JSONRPCRequest JSON-RPC 2.0 请求
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"` // 固定为 "2.0"
	ID      interface{} `json:"id"`      // 字符串或数字
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse JSON-RPC 2.0 响应
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"` // 固定为 "2.0"
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError JSON-RPC 错误对象
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 标准 JSON-RPC 错误码
const (
	ErrorCodeParseError     = -32700
	ErrorCodeInvalidRequest = -32600
	ErrorCodeMethodNotFound = -32601
	ErrorCodeInvalidParams  = -32602
	ErrorCodeInternalError  = -32603
)

// A2A 特定错误码
const (
	ErrorCodeTaskNotFound                 = -32001
	ErrorCodeTaskNotCancelable            = -32002
	ErrorCodePushNotificationNotSupported = -32003
	ErrorCodeUnsupportedOperation         = -32004
)

// ============== 方法参数 ==============

// MessageSendParams message/send 方法参数
type MessageSendParams struct {
	Message   Message  `json:"message"`
	ContextID string   `json:"contextId,omitempty"`
	Metadata  Metadata `json:"metadata,omitempty"`
}

// MessageStreamParams message/stream 方法参数
type MessageStreamParams struct {
	Message   Message  `json:"message"`
	ContextID string   `json:"contextId,omitempty"`
	Metadata  Metadata `json:"metadata,omitempty"`
}

// TasksGetParams tasks/get 方法参数
type TasksGetParams struct {
	TaskID string `json:"taskId"`
}

// TasksCancelParams tasks/cancel 方法参数
type TasksCancelParams struct {
	TaskID string `json:"taskId"`
}

// ============== 方法返回结果 ==============

// MessageSendResult message/send 方法返回结果
type MessageSendResult struct {
	TaskID string `json:"taskId"`
}

// MessageStreamResult message/stream 方法返回结果
type MessageStreamResult struct {
	TaskID string `json:"taskId"`
}

// TasksGetResult tasks/get 方法返回结果
type TasksGetResult struct {
	Task *Task `json:"task"`
}

// TasksCancelResult tasks/cancel 方法返回结果
type TasksCancelResult struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// ============== 辅助函数 ==============

// NewTask 创建新任务
func NewTask(id, contextID string) *Task {
	return &Task{
		ID:        id,
		ContextID: contextID,
		Status: TaskStatus{
			State:     TaskStateSubmitted,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
		History:   make([]Message, 0),
		Artifacts: make([]Artifact, 0),
		Metadata:  make(Metadata),
		Kind:      "task",
	}
}

// UpdateStatus 更新任务状态
func (t *Task) UpdateStatus(state TaskState, message *Message) {
	t.Status = TaskStatus{
		State:     state,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Message:   message,
	}
}

// AddMessage 添加消息到历史
func (t *Task) AddMessage(msg Message) {
	t.History = append(t.History, msg)
}

// IsFinalState 检查是否为最终状态
func (t *Task) IsFinalState() bool {
	return t.Status.State == TaskStateCompleted ||
		t.Status.State == TaskStateFailed ||
		t.Status.State == TaskStateCanceled
}

// NewTextMessage 创建文本消息
func NewTextMessage(messageID, role, text string) Message {
	return Message{
		MessageID: messageID,
		Role:      role,
		Parts: []Part{
			{Kind: "text", Text: text},
		},
		Kind: "message",
	}
}

// NewSuccessResponse 创建成功响应
func NewSuccessResponse(id interface{}, result interface{}) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(id interface{}, code int, message string, data interface{}) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}
