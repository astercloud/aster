package run

import (
	"encoding/json"
	"time"
)

// Event 统一的运行事件接口
type Event interface {
	// EventType 返回事件类型
	EventType() EventType

	// RunID 返回运行 ID
	RunID() string

	// Timestamp 返回时间戳
	Timestamp() time.Time

	// ToMap 转换为 map
	ToMap() map[string]any

	// ToJSON 转换为 JSON
	ToJSON() ([]byte, error)
}

// EventType 事件类型
type EventType string

const (
	// Agent 相关事件
	EventAgentStart      EventType = "agent.start"
	EventAgentProgress   EventType = "agent.progress"
	EventAgentToolCall   EventType = "agent.tool_call"
	EventAgentToolResult EventType = "agent.tool_result"
	EventAgentComplete   EventType = "agent.complete"
	EventAgentError      EventType = "agent.error"

	// Workflow 相关事件
	EventWorkflowStart    EventType = "workflow.start"
	EventWorkflowStep     EventType = "workflow.step"
	EventWorkflowComplete EventType = "workflow.complete"
	EventWorkflowError    EventType = "workflow.error"

	// Team 相关事件
	EventTeamStart        EventType = "team.start"
	EventTeamMemberStart  EventType = "team.member_start"
	EventTeamMemberResult EventType = "team.member_result"
	EventTeamComplete     EventType = "team.complete"
	EventTeamError        EventType = "team.error"

	// 通用事件
	EventProgress     EventType = "progress"
	EventStatusChange EventType = "status_change"
	EventMetrics      EventType = "metrics"
)

// BaseEvent 基础事件
type BaseEvent struct {
	Type      EventType      `json:"type"`
	ID        string         `json:"run_id"`
	Time      time.Time      `json:"timestamp"`
	SessionID string         `json:"session_id,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
}

// EventType 实现 Event 接口
func (e *BaseEvent) EventType() EventType {
	return e.Type
}

// RunID 实现 Event 接口
func (e *BaseEvent) RunID() string {
	return e.ID
}

// Timestamp 实现 Event 接口
func (e *BaseEvent) Timestamp() time.Time {
	return e.Time
}

// ToMap 实现 Event 接口
func (e *BaseEvent) ToMap() map[string]any {
	m := map[string]any{
		"type":      string(e.Type),
		"run_id":    e.ID,
		"timestamp": e.Time,
	}
	if e.SessionID != "" {
		m["session_id"] = e.SessionID
	}
	if e.Data != nil {
		m["data"] = e.Data
	}
	return m
}

// ToJSON 实现 Event 接口
func (e *BaseEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e.ToMap())
}

// NewBaseEvent 创建基础事件
func NewBaseEvent(eventType EventType, runID string) *BaseEvent {
	return &BaseEvent{
		Type: eventType,
		ID:   runID,
		Time: time.Now(),
		Data: make(map[string]any),
	}
}

// AgentEvent Agent 事件
type AgentEvent struct {
	BaseEvent
	AgentID string         `json:"agent_id,omitempty"`
	Content string         `json:"content,omitempty"`
	Delta   string         `json:"delta,omitempty"`
	Metrics map[string]any `json:"metrics,omitempty"`
}

// WorkflowEvent Workflow 事件
type WorkflowEvent struct {
	BaseEvent
	WorkflowID string         `json:"workflow_id,omitempty"`
	StepID     string         `json:"step_id,omitempty"`
	StepName   string         `json:"step_name,omitempty"`
	StepType   string         `json:"step_type,omitempty"`
	Progress   float64        `json:"progress,omitempty"`
	Metrics    map[string]any `json:"metrics,omitempty"`
}

// TeamEvent Team 事件
type TeamEvent struct {
	BaseEvent
	TeamID   string         `json:"team_id,omitempty"`
	MemberID string         `json:"member_id,omitempty"`
	Role     string         `json:"role,omitempty"`
	Metrics  map[string]any `json:"metrics,omitempty"`
}

// StatusChangeEvent 状态变更事件
type StatusChangeEvent struct {
	BaseEvent
	OldStatus Status `json:"old_status"`
	NewStatus Status `json:"new_status"`
	Reason    string `json:"reason,omitempty"`
}

// MetricsEvent 指标事件
type MetricsEvent struct {
	BaseEvent
	TokensUsed    int     `json:"tokens_used,omitempty"`
	TokensInput   int     `json:"tokens_input,omitempty"`
	TokensOutput  int     `json:"tokens_output,omitempty"`
	ExecutionTime float64 `json:"execution_time,omitempty"`
	ModelCalls    int     `json:"model_calls,omitempty"`
	ToolCalls     int     `json:"tool_calls,omitempty"`
}
