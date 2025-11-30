package types

import (
	"time"
)

// ToolCallRecord 工具调用记录
type ToolCallRecord struct {
	ID           string               `json:"id"`                     // 工具调用 ID
	Name         string               `json:"name"`                   // 工具名称（新字段）
	ToolName     string               `json:"tool_name"`              // 工具名称（兼容）
	Input        map[string]any       `json:"input"`                  // 输入参数
	Output       any                  `json:"output"`                 // 输出结果
	Result       any                  `json:"result"`                 // 执行结果（新字段）
	Error        string               `json:"error,omitempty"`        // 错误信息
	IsError      bool                 `json:"is_error"`               // 是否有错误（新字段）
	Progress     float64              `json:"progress"`               // 执行进度 0-1
	Intermediate map[string]any       `json:"intermediate,omitempty"` // 中间结果
	StartTime    time.Time            `json:"start_time"`             // 开始时间
	EndTime      time.Time            `json:"end_time"`               // 结束时间
	StartedAt    *time.Time           `json:"started_at"`             // 开始时间（新字段，指针）
	CompletedAt  *time.Time           `json:"completed_at"`           // 完成时间（新字段，指针）
	DurationMs   *int64               `json:"duration_ms"`            // 执行时长（毫秒，改为指针）
	Status       ToolCallStatus       `json:"status"`                 // 调用状态
	State        ToolCallState        `json:"state"`                  // 工具调用状态（新字段）
	Approval     ToolCallApproval     `json:"approval"`               // 审批信息（新字段）
	CreatedAt    time.Time            `json:"created_at"`             // 创建时间（新字段）
	UpdatedAt    time.Time            `json:"updated_at"`             // 更新时间（新字段）
	AuditTrail   []ToolCallAuditEntry `json:"audit_trail"`            // 审计跟踪（新字段）
}

// ToolCallStatus 工具调用状态
type ToolCallStatus string

const (
	ToolCallStatusPending   ToolCallStatus = "pending"   // 待执行
	ToolCallStatusRunning   ToolCallStatus = "running"   // 执行中
	ToolCallStatusCompleted ToolCallStatus = "completed" // 已完成
	ToolCallStatusFailed    ToolCallStatus = "failed"    // 失败
	ToolCallStatusCancelled ToolCallStatus = "cancelled" // 已取消
)

// Snapshot Agent 状态快照
type Snapshot struct {
	ID          string           `json:"id"`                    // 快照 ID
	AgentID     string           `json:"agent_id"`              // Agent ID
	Timestamp   time.Time        `json:"timestamp"`             // 快照时间
	Messages    []Message        `json:"messages"`              // 消息历史
	ToolCalls   []ToolCallRecord `json:"tool_calls"`            // 工具调用记录
	State       AgentState       `json:"state"`                 // Agent 状态
	StepCount   int              `json:"step_count"`            // 步骤计数
	Cursor      int              `json:"cursor"`                // 当前位置
	Metadata    map[string]any   `json:"metadata"`              // 元数据
	Description string           `json:"description,omitempty"` // 快照描述
}

// AgentState Agent 状态
type AgentState string

const (
	AgentStateIdle      AgentState = "idle"      // 空闲
	AgentStateRunning   AgentState = "running"   // 运行中
	AgentStateWaiting   AgentState = "waiting"   // 等待中（等待工具调用结果）
	AgentStateCompleted AgentState = "completed" // 已完成
	AgentStateFailed    AgentState = "failed"    // 失败
	AgentStateCancelled AgentState = "cancelled" // 已取消
)

// AgentStatus Agent 实时状态
type AgentStatus struct {
	AgentID      string            `json:"agent_id"`       // Agent ID
	State        AgentRuntimeState `json:"state"`          // 运行时状态
	StepCount    int               `json:"step_count"`     // 步骤数
	LastSfpIndex int               `json:"last_sfp_index"` // 最后 SFP 索引
	LastBookmark *Bookmark         `json:"last_bookmark"`  // 最后书签
	Cursor       int64             `json:"cursor"`         // 游标
	Breakpoint   BreakpointState   `json:"breakpoint"`     // 断点状态
}

// AgentInfo Agent 元信息
type AgentInfo struct {
	ID            string         `json:"id"`                    // Agent ID
	AgentID       string         `json:"agent_id"`              // Agent ID（别名）
	TemplateID    string         `json:"template_id"`           // 使用的模板 ID
	Model         string         `json:"model"`                 // 模型名称
	CreatedAt     time.Time      `json:"created_at"`            // 创建时间
	UpdatedAt     time.Time      `json:"updated_at"`            // 更新时间
	State         AgentState     `json:"state"`                 // 当前状态
	StepCount     int            `json:"step_count"`            // 总步骤数
	Cursor        int            `json:"cursor"`                // 当前游标
	MessageCount  int            `json:"message_count"`         // 消息数量（新字段）
	Lineage       []string       `json:"lineage"`               // 血缘关系（新字段）
	ConfigVersion string         `json:"config_version"`        // 配置版本（新字段）
	Metadata      map[string]any `json:"metadata"`              // 自定义元数据
	Tags          []string       `json:"tags,omitempty"`        // 标签
	Description   string         `json:"description,omitempty"` // 描述
}

// Event 事件类型（用于工具和 Agent 之间的通信）
type Event struct {
	Type      string         `json:"type"`               // 事件类型
	Data      any            `json:"data"`               // 事件数据
	Timestamp time.Time      `json:"timestamp"`          // 时间戳
	Source    string         `json:"source,omitempty"`   // 事件源
	Metadata  map[string]any `json:"metadata,omitempty"` // 元数据
}

// ToolDefinition 工具定义
type ToolDefinition struct {
	Name        string         `json:"name"`         // 工具名称
	Description string         `json:"description"`  // 工具描述
	InputSchema map[string]any `json:"input_schema"` // 输入 Schema
}

// ToolSchema 工具 Schema 定义
type ToolSchema struct {
	Type        string                     `json:"type"`                  // "object"
	Properties  map[string]*PropertySchema `json:"properties"`            // 属性定义
	Required    []string                   `json:"required,omitempty"`    // 必需字段
	Description string                     `json:"description,omitempty"` // 描述
}

// PropertySchema 属性 Schema 定义
type PropertySchema struct {
	Type        string                     `json:"type"`                  // string, number, boolean, array, object
	Description string                     `json:"description,omitempty"` // 描述
	Enum        []any                      `json:"enum,omitempty"`        // 枚举值
	Items       *PropertySchema            `json:"items,omitempty"`       // 数组元素 schema
	Properties  map[string]*PropertySchema `json:"properties,omitempty"`  // 对象属性（嵌套）
	Required    []string                   `json:"required,omitempty"`    // 必需字段（对象）
	Default     any                        `json:"default,omitempty"`     // 默认值
}

// ToolCallState 工具调用状态（用于并发控制）
type ToolCallState string

const (
	ToolCallStatePending    ToolCallState = "pending"    // 待执行
	ToolCallStateQueued     ToolCallState = "queued"     // 已排队
	ToolCallStateExecuting  ToolCallState = "executing"  // 执行中
	ToolCallStateCompleted  ToolCallState = "completed"  // 已完成
	ToolCallStateFailed     ToolCallState = "failed"     // 失败
	ToolCallStateCancelling ToolCallState = "cancelling" // 取消中
	ToolCallStateCancelled  ToolCallState = "cancelled"  // 已取消
)

// ToolCallApproval 工具调用审批
type ToolCallApproval struct {
	CallID     string    `json:"call_id"`               // 工具调用 ID
	Required   bool      `json:"required"`              // 是否需要审批（新字段）
	Approved   bool      `json:"approved"`              // 是否批准
	Reason     string    `json:"reason,omitempty"`      // 原因
	Timestamp  time.Time `json:"timestamp"`             // 审批时间
	ApprovedBy string    `json:"approved_by,omitempty"` // 审批人
}

// ToolCallAuditEntry 工具调用审计条目
type ToolCallAuditEntry struct {
	State     ToolCallState `json:"state"`     // 状态
	Timestamp time.Time     `json:"timestamp"` // 时间戳
	Note      string        `json:"note"`      // 备注
}
