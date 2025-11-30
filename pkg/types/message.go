package types

import "time"

// Role 定义消息角色
type Role string

const (
	// RoleUser 用户角色
	RoleUser Role = "user"

	// RoleAssistant AI助手角色
	RoleAssistant Role = "assistant"

	// RoleSystem 系统角色
	RoleSystem Role = "system"

	// RoleTool 工具角色
	RoleTool Role = "tool"

	// 兼容性别名
	MessageRoleSystem    = RoleSystem
	MessageRoleAssistant = RoleAssistant
	MessageRoleUser      = RoleUser
	MessageRoleTool      = RoleTool
)

// ContentBlock 内容块接口
type ContentBlock interface {
	IsContentBlock()
}

// TextBlock 文本内容块
type TextBlock struct {
	Text string `json:"text"`
}

func (t *TextBlock) IsContentBlock() {}

// ToolUseBlock 工具使用块
type ToolUseBlock struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Input  map[string]any `json:"input"`
	Caller *ToolCaller    `json:"caller,omitempty"` // PTC: 调用者信息
}

// ToolCaller 工具调用者信息 (PTC 支持)
type ToolCaller struct {
	// Type 调用者类型: "direct" (LLM直接调用) 或 "code_execution_20250825" (代码执行中调用)
	Type string `json:"type"`

	// ToolID 代码执行工具的 ID (当 Type="code_execution_20250825" 时)
	ToolID string `json:"tool_id,omitempty"`
}

func (t *ToolUseBlock) IsContentBlock() {}

// ToolResultBlock 工具结果块
type ToolResultBlock struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`

	// 以下字段用于可恢复压缩（Manus Context Engineering）
	// Compressed 标记内容是否已被压缩
	Compressed bool `json:"compressed,omitempty"`
	// OriginalLength 原始内容长度（用于统计和验证）
	OriginalLength int `json:"original_length,omitempty"`
	// ContentHash 原始内容哈希（用于验证恢复）
	ContentHash string `json:"content_hash,omitempty"`
	// References 可恢复的引用列表（文件路径、URL 等）
	References []ToolResultReference `json:"references,omitempty"`
}

// ToolResultReference 工具结果中的引用
type ToolResultReference struct {
	// Type 引用类型: "file_path", "url", "function", "class"
	Type string `json:"type"`
	// Value 引用值
	Value string `json:"value"`
	// Context 上下文信息（可选）
	Context string `json:"context,omitempty"`
}

func (t *ToolResultBlock) IsContentBlock() {}

// Message 表示一条消息
type Message struct {
	// Role 消息角色
	Role Role `json:"role"`

	// Content 消息内容（简单文本格式，与 ContentBlocks 二选一）
	Content string `json:"content,omitempty"`

	// ContentBlocks 消息内容块（复杂格式，与 Content 二选一）
	// 用于支持多模态内容（文本、工具调用、工具结果等）
	ContentBlocks []ContentBlock `json:"-"`

	// Name 可选的名称字段（用于function/tool角色）
	Name string `json:"name,omitempty"`

	// ToolCalls 工具调用列表（仅assistant角色）
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// ToolCallID 工具调用ID（仅tool角色）
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// GetContent 获取消息内容，优先返回 Content，如果为空则从 ContentBlocks 提取
func (m *Message) GetContent() string {
	if m.Content != "" {
		return m.Content
	}
	// 从 ContentBlocks 提取文本
	for _, block := range m.ContentBlocks {
		if tb, ok := block.(*TextBlock); ok {
			return tb.Text
		}
	}
	return ""
}

// SetContent 设置消息内容（简单文本格式）
func (m *Message) SetContent(content string) {
	m.Content = content
	m.ContentBlocks = nil
}

// SetContentBlocks 设置消息内容块（复杂格式）
func (m *Message) SetContentBlocks(blocks []ContentBlock) {
	m.ContentBlocks = blocks
	m.Content = ""
}

// ToolCall 表示一个工具调用
type ToolCall struct {
	// ID 工具调用的唯一标识符
	ID string `json:"id"`

	// Type 工具类型，通常为 "function"
	Type string `json:"type,omitempty"`

	// Name 工具名称
	Name string `json:"name"`

	// Arguments 工具参数（JSON对象）
	Arguments map[string]any `json:"arguments,omitempty"`
}

// ToolResult 表示工具执行结果
type ToolResult struct {
	// ToolCallID 关联的工具调用ID
	ToolCallID string `json:"tool_call_id"`

	// Content 工具执行结果
	Content string `json:"content"`

	// Error 错误信息（如果有）
	Error string `json:"error,omitempty"`
}

// Bookmark 表示事件流的书签位置
type Bookmark struct {
	// Cursor 游标位置
	Cursor int64 `json:"cursor"`

	// Timestamp 时间戳
	Timestamp int64 `json:"timestamp,omitempty"`
}

// ToolCallSnapshot 工具调用快照
type ToolCallSnapshot struct {
	// ID 工具调用ID
	ID string `json:"id"`

	// Name 工具名称
	Name string `json:"name"`

	// State 工具调用状态
	State ToolCallState `json:"state,omitempty"`

	// Progress 进度 0-1
	Progress float64 `json:"progress,omitempty"`

	// Arguments 工具参数
	Arguments map[string]any `json:"arguments,omitempty"`

	// Result 工具执行结果
	Result any `json:"result,omitempty"`

	// Error 错误信息
	Error string `json:"error,omitempty"`

	// Intermediate 中间结果
	Intermediate map[string]any `json:"intermediate,omitempty"`

	// 时间信息
	StartedAt time.Time `json:"started_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`

	// 控制能力标识
	Cancelable bool `json:"cancelable,omitempty"`
	Pausable   bool `json:"pausable,omitempty"`
}

// AgentRuntimeState Agent运行时状态
type AgentRuntimeState string

const (
	// AgentStateReady Agent就绪
	AgentStateReady AgentRuntimeState = "ready"

	// AgentStateWorking Agent工作中
	AgentStateWorking AgentRuntimeState = "working"

	// StateIdle Agent空闲
	StateIdle AgentRuntimeState = "idle"

	// StateRunning Agent运行中
	StateRunning AgentRuntimeState = "running"

	// StatePaused Agent暂停
	StatePaused AgentRuntimeState = "paused"

	// StateCompleted Agent完成
	StateCompleted AgentRuntimeState = "completed"

	// StateFailed Agent失败
	StateFailed AgentRuntimeState = "failed"
)

// BreakpointState 断点状态
type BreakpointState struct {
	// Enabled 是否启用
	Enabled bool `json:"enabled"`

	// Condition 断点条件
	Condition string `json:"condition,omitempty"`

	// HitCount 命中次数
	HitCount int `json:"hit_count,omitempty"`
}

// BreakpointReady 就绪状态的断点（未启用）
var BreakpointReady = BreakpointState{
	Enabled: false,
}

// BreakpointPreModel 模型调用前的断点
var BreakpointPreModel = BreakpointState{
	Enabled:   true,
	Condition: "pre_model",
}

// BreakpointStreamingModel 模型流式响应中的断点
var BreakpointStreamingModel = BreakpointState{
	Enabled:   true,
	Condition: "streaming_model",
}

// BreakpointToolPending 工具调用待处理的断点
var BreakpointToolPending = BreakpointState{
	Enabled:   true,
	Condition: "tool_pending",
}

// BreakpointPreTool 工具执行前的断点
var BreakpointPreTool = BreakpointState{
	Enabled:   true,
	Condition: "pre_tool",
}

// BreakpointToolExecuting 工具执行中的断点
var BreakpointToolExecuting = BreakpointState{
	Enabled:   true,
	Condition: "tool_executing",
}

// BreakpointPostTool 工具执行后的断点
var BreakpointPostTool = BreakpointState{
	Enabled:   true,
	Condition: "post_tool",
}
