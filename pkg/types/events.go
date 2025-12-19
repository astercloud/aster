package types

import "time"

// AgentChannel 事件通道类型
type AgentChannel string

const (
	ChannelProgress AgentChannel = "progress"
	ChannelControl  AgentChannel = "control"
	ChannelMonitor  AgentChannel = "monitor"
)

// EventType 事件类型基础接口
type EventType interface {
	Channel() AgentChannel
	EventType() string
}

// AgentEventEnvelope 事件封装(带Bookmark)
type AgentEventEnvelope struct {
	Cursor   int64    `json:"cursor"`
	Bookmark Bookmark `json:"bookmark"`
	Event    any      `json:"event"`
}

// ===================
// Progress Channel Events
// ===================

// ProgressThinkChunkStartEvent 思考块开始事件
type ProgressThinkChunkStartEvent struct {
	Step int `json:"step"`
}

func (e *ProgressThinkChunkStartEvent) Channel() AgentChannel { return ChannelProgress }
func (e *ProgressThinkChunkStartEvent) EventType() string     { return "think_chunk_start" }

// ProgressThinkChunkEvent 思考块内容事件
// 统一的思考事件格式，支持各种 LLM 模型的推理输出
type ProgressThinkChunkEvent struct {
	Step      int    `json:"step"`
	Delta     string `json:"delta"`               // 增量内容 (流式使用)
	ID        string `json:"id,omitempty"`        // 事件ID
	Stage     string `json:"stage,omitempty"`     // 阶段名称: "任务规划", "推理分析", "工具规划", "结果总结"
	Reasoning string `json:"reasoning,omitempty"` // 完整推理内容 (非流式使用)
	Decision  string `json:"decision,omitempty"`  // 决策/结论
	Context   any    `json:"context,omitempty"`   // 上下文信息
	Timestamp string `json:"timestamp,omitempty"` // ISO8601 时间戳
}

func (e *ProgressThinkChunkEvent) Channel() AgentChannel { return ChannelProgress }
func (e *ProgressThinkChunkEvent) EventType() string     { return "think_chunk" }

// NewThinkingEvent 创建思考事件的便捷方法
func NewThinkingEvent(stage, reasoning, decision string) *ProgressThinkChunkEvent {
	return &ProgressThinkChunkEvent{
		Stage:     stage,
		Reasoning: reasoning,
		Decision:  decision,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// ThinkingStage 思考阶段常量
const (
	ThinkingStageTaskPlanning  = "任务规划"
	ThinkingStageReasoning     = "推理分析"
	ThinkingStageToolPlanning  = "工具规划"
	ThinkingStageToolExecuting = "工具执行"
	ThinkingStageSummary       = "结果总结"
)

// ProgressThinkChunkEndEvent 思考块结束事件
type ProgressThinkChunkEndEvent struct {
	Step int `json:"step"`
}

func (e *ProgressThinkChunkEndEvent) Channel() AgentChannel { return ChannelProgress }
func (e *ProgressThinkChunkEndEvent) EventType() string     { return "think_chunk_end" }

// ProgressTextChunkStartEvent 文本块开始事件
type ProgressTextChunkStartEvent struct {
	Step int `json:"step"`
}

func (e *ProgressTextChunkStartEvent) Channel() AgentChannel { return ChannelProgress }
func (e *ProgressTextChunkStartEvent) EventType() string     { return "text_chunk_start" }

// ProgressTextChunkEvent 文本块内容事件
type ProgressTextChunkEvent struct {
	Step  int    `json:"step"`
	Delta string `json:"delta"`
}

func (e *ProgressTextChunkEvent) Channel() AgentChannel { return ChannelProgress }
func (e *ProgressTextChunkEvent) EventType() string     { return "text_chunk" }

// ProgressTextChunkEndEvent 文本块结束事件
type ProgressTextChunkEndEvent struct {
	Step int    `json:"step"`
	Text string `json:"text"`
}

func (e *ProgressTextChunkEndEvent) Channel() AgentChannel { return ChannelProgress }
func (e *ProgressTextChunkEndEvent) EventType() string     { return "text_chunk_end" }

// ProgressToolStartEvent 工具开始执行事件
type ProgressToolStartEvent struct {
	Call ToolCallSnapshot `json:"call"`
}

func (e *ProgressToolStartEvent) Channel() AgentChannel { return ChannelProgress }
func (e *ProgressToolStartEvent) EventType() string     { return "tool:start" }

// ProgressToolEndEvent 工具执行结束事件
type ProgressToolEndEvent struct {
	Call ToolCallSnapshot `json:"call"`
}

func (e *ProgressToolEndEvent) Channel() AgentChannel { return ChannelProgress }
func (e *ProgressToolEndEvent) EventType() string     { return "tool:end" }

// ProgressToolProgressEvent 工具执行进度事件
type ProgressToolProgressEvent struct {
	Call     ToolCallSnapshot `json:"call"`
	Progress float64          `json:"progress"`           // 0.0 - 1.0
	Message  string           `json:"message,omitempty"`  // 进度描述
	Step     int              `json:"step,omitempty"`     // 当前步骤
	Total    int              `json:"total,omitempty"`    // 总步骤
	Metadata map[string]any   `json:"metadata,omitempty"` // 额外元数据
	ETA      int64            `json:"eta_ms,omitempty"`   // 预估剩余时间(ms)
}

func (e *ProgressToolProgressEvent) Channel() AgentChannel { return ChannelProgress }
func (e *ProgressToolProgressEvent) EventType() string     { return "tool:progress" }

// ProgressToolIntermediateEvent 工具中间结果事件
type ProgressToolIntermediateEvent struct {
	Call  ToolCallSnapshot `json:"call"`
	Label string           `json:"label,omitempty"`
	Data  any              `json:"data,omitempty"`
}

func (e *ProgressToolIntermediateEvent) Channel() AgentChannel { return ChannelProgress }
func (e *ProgressToolIntermediateEvent) EventType() string     { return "tool:intermediate" }

// ProgressToolCancelledEvent 工具取消事件
type ProgressToolCancelledEvent struct {
	Call   ToolCallSnapshot `json:"call"`
	Reason string           `json:"reason,omitempty"`
}

func (e *ProgressToolCancelledEvent) Channel() AgentChannel { return ChannelProgress }
func (e *ProgressToolCancelledEvent) EventType() string     { return "tool:canceled" }

// ProgressToolErrorEvent 工具执行错误事件
type ProgressToolErrorEvent struct {
	Call  ToolCallSnapshot `json:"call"`
	Error string           `json:"error"`
}

func (e *ProgressToolErrorEvent) Channel() AgentChannel { return ChannelProgress }
func (e *ProgressToolErrorEvent) EventType() string     { return "tool:error" }

// ProgressDoneEvent 单轮完成事件
type ProgressDoneEvent struct {
	Step   int    `json:"step"`
	Reason string `json:"reason"` // "completed" or "interrupted"
}

func (e *ProgressDoneEvent) Channel() AgentChannel { return ChannelProgress }
func (e *ProgressDoneEvent) EventType() string     { return "done" }

// ProgressSessionSummarizedEvent 会话历史已汇总事件
// 当 SummarizationMiddleware 压缩历史消息时发送
type ProgressSessionSummarizedEvent struct {
	MessagesBefore   int     `json:"messages_before"`   // 压缩前消息数
	MessagesAfter    int     `json:"messages_after"`    // 压缩后消息数
	TokensBefore     int     `json:"tokens_before"`     // 压缩前 Token 数
	TokensAfter      int     `json:"tokens_after"`      // 压缩后 Token 数
	TokensSaved      int     `json:"tokens_saved"`      // 节省的 Token 数
	CompressionRatio float64 `json:"compression_ratio"` // 压缩比 (0-1)
	SummaryPreview   string  `json:"summary_preview"`   // 摘要预览 (前150字符)
}

func (e *ProgressSessionSummarizedEvent) Channel() AgentChannel { return ChannelProgress }
func (e *ProgressSessionSummarizedEvent) EventType() string     { return "session_summarized" }

// ===================
// Control Channel Events
// ===================

// RespondFunc 审批响应回调函数
type RespondFunc func(decision string, note string) error

// ControlPermissionRequiredEvent 权限请求事件
type ControlPermissionRequiredEvent struct {
	Call    ToolCallSnapshot `json:"call"`
	Respond RespondFunc      `json:"-"` // 不序列化回调函数
}

func (e *ControlPermissionRequiredEvent) Channel() AgentChannel { return ChannelControl }
func (e *ControlPermissionRequiredEvent) EventType() string     { return "permission_required" }

// ControlPermissionDecidedEvent 权限决策事件
type ControlPermissionDecidedEvent struct {
	CallID    string `json:"call_id"`
	Decision  string `json:"decision"` // "allow" or "deny"
	DecidedBy string `json:"decided_by"`
	Note      string `json:"note,omitempty"`
}

func (e *ControlPermissionDecidedEvent) Channel() AgentChannel { return ChannelControl }
func (e *ControlPermissionDecidedEvent) EventType() string     { return "permission_decided" }

// ControlIterationLimitEvent 迭代限制事件
type ControlIterationLimitEvent struct {
	CurrentIteration int    `json:"current_iteration"`
	MaxIteration     int    `json:"max_iteration"`
	Message          string `json:"message"`
}

func (e *ControlIterationLimitEvent) Channel() AgentChannel { return ChannelControl }
func (e *ControlIterationLimitEvent) EventType() string     { return "iteration_limit" }

// ControlToolControlEvent 工具控制指令事件（入站）
type ControlToolControlEvent struct {
	CallID string `json:"call_id"`
	Action string `json:"action"` // pause|resume|cancel
	Note   string `json:"note,omitempty"`
}

func (e *ControlToolControlEvent) Channel() AgentChannel { return ChannelControl }
func (e *ControlToolControlEvent) EventType() string     { return "tool_control" }

// ControlToolControlResponseEvent 工具控制响应事件（出站）
type ControlToolControlResponseEvent struct {
	CallID string `json:"call_id"`
	Action string `json:"action"` // pause|resume|cancel
	OK     bool   `json:"ok"`
	Reason string `json:"reason,omitempty"`
}

func (e *ControlToolControlResponseEvent) Channel() AgentChannel { return ChannelControl }
func (e *ControlToolControlResponseEvent) EventType() string     { return "tool_control_response" }

// ===================
// Monitor Channel Events
// ===================

// MonitorStateChangedEvent 状态变更事件
type MonitorStateChangedEvent struct {
	State AgentRuntimeState `json:"state"`
}

func (e *MonitorStateChangedEvent) Channel() AgentChannel { return ChannelMonitor }
func (e *MonitorStateChangedEvent) EventType() string     { return "state_changed" }

// MonitorStepCompleteEvent 步骤完成事件
type MonitorStepCompleteEvent struct {
	Step       int   `json:"step"`
	DurationMs int64 `json:"duration_ms,omitempty"`
}

func (e *MonitorStepCompleteEvent) Channel() AgentChannel { return ChannelMonitor }
func (e *MonitorStepCompleteEvent) EventType() string     { return "step_complete" }

// MonitorErrorEvent 错误事件
type MonitorErrorEvent struct {
	Severity string         `json:"severity"` // "info", "warn", "error"
	Phase    string         `json:"phase"`    // "model", "tool", "system", "lifecycle"
	Message  string         `json:"message"`
	Detail   map[string]any `json:"detail,omitempty"`
}

func (e *MonitorErrorEvent) Channel() AgentChannel { return ChannelMonitor }
func (e *MonitorErrorEvent) EventType() string     { return "error" }

// MonitorTokenUsageEvent Token使用统计事件
type MonitorTokenUsageEvent struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	TotalTokens  int64 `json:"total_tokens"`
}

func (e *MonitorTokenUsageEvent) Channel() AgentChannel { return ChannelMonitor }
func (e *MonitorTokenUsageEvent) EventType() string     { return "token_usage" }

// MonitorToolExecutedEvent 工具执行完成事件
type MonitorToolExecutedEvent struct {
	Call ToolCallSnapshot `json:"call"`
}

func (e *MonitorToolExecutedEvent) Channel() AgentChannel { return ChannelMonitor }
func (e *MonitorToolExecutedEvent) EventType() string     { return "tool_executed" }

// MonitorAgentResumedEvent Agent恢复事件
type MonitorAgentResumedEvent struct {
	Strategy string             `json:"strategy"` // "crash" or "manual"
	Sealed   []ToolCallSnapshot `json:"sealed"`
}

func (e *MonitorAgentResumedEvent) Channel() AgentChannel { return ChannelMonitor }
func (e *MonitorAgentResumedEvent) EventType() string     { return "agent_resumed" }

// MonitorBreakpointChangedEvent 断点变更事件
type MonitorBreakpointChangedEvent struct {
	Previous  BreakpointState `json:"previous"`
	Current   BreakpointState `json:"current"`
	Timestamp time.Time       `json:"timestamp"`
}

func (e *MonitorBreakpointChangedEvent) Channel() AgentChannel { return ChannelMonitor }
func (e *MonitorBreakpointChangedEvent) EventType() string     { return "breakpoint_changed" }

// MonitorFileChangedEvent 文件变更事件
type MonitorFileChangedEvent struct {
	Path  string    `json:"path"`
	Mtime time.Time `json:"mtime"`
}

func (e *MonitorFileChangedEvent) Channel() AgentChannel { return ChannelMonitor }
func (e *MonitorFileChangedEvent) EventType() string     { return "file_changed" }

// MonitorReminderSentEvent 系统提醒事件
type MonitorReminderSentEvent struct {
	Category string `json:"category"` // "file", "todo", "security", "performance", "general"
	Content  string `json:"content"`
}

func (e *MonitorReminderSentEvent) Channel() AgentChannel { return ChannelMonitor }
func (e *MonitorReminderSentEvent) EventType() string     { return "reminder_sent" }

// MonitorContextCompressionEvent 上下文压缩事件
type MonitorContextCompressionEvent struct {
	Phase   string  `json:"phase"` // "start" or "end"
	Summary string  `json:"summary,omitempty"`
	Ratio   float64 `json:"ratio,omitempty"`
}

func (e *MonitorContextCompressionEvent) Channel() AgentChannel { return ChannelMonitor }
func (e *MonitorContextCompressionEvent) EventType() string     { return "context_compression" }

// MonitorSchedulerTriggeredEvent 调度器触发事件
type MonitorSchedulerTriggeredEvent struct {
	TaskID      string    `json:"task_id"`
	Spec        string    `json:"spec"`
	Kind        string    `json:"kind"` // "steps", "time", "cron"
	TriggeredAt time.Time `json:"triggered_at"`
}

func (e *MonitorSchedulerTriggeredEvent) Channel() AgentChannel { return ChannelMonitor }
func (e *MonitorSchedulerTriggeredEvent) EventType() string     { return "scheduler_triggered" }

// MonitorToolManualUpdatedEvent 工具手册更新事件
type MonitorToolManualUpdatedEvent struct {
	Tools     []string  `json:"tools"`
	Timestamp time.Time `json:"timestamp"`
}

func (e *MonitorToolManualUpdatedEvent) Channel() AgentChannel { return ChannelMonitor }
func (e *MonitorToolManualUpdatedEvent) EventType() string     { return "tool_manual_updated" }

// ===================
// AskUserQuestion Events (Control Channel)
// ===================

// QuestionOption 问题选项
type QuestionOption struct {
	Label       string `json:"label"`       // 选项标签，1-5个词
	Description string `json:"description"` // 选项说明
}

// Question 结构化问题
type Question struct {
	Question    string           `json:"question"`     // 完整的问题文本
	Header      string           `json:"header"`       // 简短标签，最多12字符
	Options     []QuestionOption `json:"options"`      // 2-4个选项
	MultiSelect bool             `json:"multi_select"` // 是否多选
}

// ControlAskUserEvent 请求用户回答问题事件
type ControlAskUserEvent struct {
	RequestID string                             `json:"request_id"`
	Questions []Question                         `json:"questions"`
	Respond   func(answers map[string]any) error `json:"-"` // 响应回调
}

func (e *ControlAskUserEvent) Channel() AgentChannel { return ChannelControl }
func (e *ControlAskUserEvent) EventType() string     { return "ask_user" }

// ControlUserAnswerEvent 用户回答事件
type ControlUserAnswerEvent struct {
	RequestID string         `json:"request_id"`
	Answers   map[string]any `json:"answers"` // question_index -> answer(s)
}

func (e *ControlUserAnswerEvent) Channel() AgentChannel { return ChannelControl }
func (e *ControlUserAnswerEvent) EventType() string     { return "user_answer" }

// ===================
// Todo Events (Progress Channel)
// ===================

// TodoItem Todo项目
type TodoItem struct {
	ID         string    `json:"id"`
	Content    string    `json:"content"`     // 祈使句形式: "Run tests"
	ActiveForm string    `json:"active_form"` // 进行时形式: "Running tests"
	Status     string    `json:"status"`      // "pending", "in_progress", "completed"
	Priority   int       `json:"priority,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ProgressTodoUpdateEvent Todo列表更新事件
type ProgressTodoUpdateEvent struct {
	Todos []TodoItem `json:"todos"`
}

func (e *ProgressTodoUpdateEvent) Channel() AgentChannel { return ChannelProgress }
func (e *ProgressTodoUpdateEvent) EventType() string     { return "todo_update" }
