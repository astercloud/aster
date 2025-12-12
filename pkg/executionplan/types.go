package executionplan

import (
	"time"
)

// Status 执行计划状态
type Status string

const (
	StatusDraft           Status = "draft"            // 草稿状态
	StatusPendingApproval Status = "pending_approval" // 等待用户审批
	StatusApproved        Status = "approved"         // 已审批，准备执行
	StatusExecuting       Status = "executing"        // 执行中
	StatusCompleted       Status = "completed"        // 执行完成
	StatusFailed          Status = "failed"           // 执行失败
	StatusCancelled       Status = "cancelled"        // 已取消
	StatusPartial         Status = "partial"          // 部分完成
)

// StepStatus 步骤状态
type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"    // 待执行
	StepStatusRunning   StepStatus = "running"    // 执行中
	StepStatusCompleted StepStatus = "completed"  // 执行完成
	StepStatusFailed    StepStatus = "failed"     // 执行失败
	StepStatusSkipped   StepStatus = "skipped"    // 已跳过
)

// Step 执行计划中的单个步骤
type Step struct {
	// 基础信息
	ID          string `json:"id"`                    // 步骤唯一ID
	Index       int    `json:"index"`                 // 步骤序号（从0开始）
	ToolName    string `json:"tool_name"`             // 要调用的工具名称
	Description string `json:"description"`           // 步骤描述（自然语言）
	
	// 参数
	Input      string         `json:"input,omitempty"`      // 原始输入（LLM 生成的字符串）
	Parameters map[string]any `json:"parameters,omitempty"` // 解析后的参数
	
	// 执行状态
	Status      StepStatus `json:"status"`
	Result      any        `json:"result,omitempty"`       // 执行结果
	Error       string     `json:"error,omitempty"`        // 错误信息
	StartedAt   *time.Time `json:"started_at,omitempty"`   // 开始时间
	CompletedAt *time.Time `json:"completed_at,omitempty"` // 完成时间
	DurationMs  int64      `json:"duration_ms,omitempty"`  // 执行耗时（毫秒）
	
	// 依赖关系
	DependsOn []string `json:"depends_on,omitempty"` // 依赖的步骤ID列表
	
	// 重试信息
	RetryCount    int `json:"retry_count,omitempty"`    // 已重试次数
	MaxRetries    int `json:"max_retries,omitempty"`    // 最大重试次数
	RetryDelayMs  int `json:"retry_delay_ms,omitempty"` // 重试间隔（毫秒）
}

// ExecutionPlan 执行计划
type ExecutionPlan struct {
	// 基础信息
	ID          string `json:"id"`                    // 计划唯一ID
	TaskID      string `json:"task_id,omitempty"`     // 关联的任务ID
	Name        string `json:"name,omitempty"`        // 计划名称
	Description string `json:"description"`           // 计划描述（自然语言）
	
	// 步骤列表
	Steps []Step `json:"steps"`
	
	// 审批状态
	UserApproved   bool       `json:"user_approved"`              // 是否已用户审批
	ApprovedAt     *time.Time `json:"approved_at,omitempty"`      // 审批时间
	ApprovedBy     string     `json:"approved_by,omitempty"`      // 审批人
	RejectionNote  string     `json:"rejection_note,omitempty"`   // 拒绝原因
	
	// 执行状态
	Status        Status     `json:"status"`
	CurrentStep   int        `json:"current_step"`               // 当前执行到的步骤索引
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	TotalDurationMs int64    `json:"total_duration_ms,omitempty"`
	
	// 上下文信息
	AgentID  string         `json:"agent_id,omitempty"`  // 关联的 Agent ID
	OrgID    string         `json:"org_id,omitempty"`    // 组织 ID（多租户）
	TenantID string         `json:"tenant_id,omitempty"` // 租户 ID（多租户）
	Metadata map[string]any `json:"metadata,omitempty"`  // 自定义元数据
	
	// 执行选项
	Options *ExecutionOptions `json:"options,omitempty"`
}

// ExecutionOptions 执行选项
type ExecutionOptions struct {
	// 并行执行
	AllowParallel    bool `json:"allow_parallel,omitempty"`     // 是否允许并行执行无依赖的步骤
	MaxParallelSteps int  `json:"max_parallel_steps,omitempty"` // 最大并行步骤数
	
	// 错误处理
	StopOnError     bool `json:"stop_on_error,omitempty"`      // 出错时是否停止
	ContinueOnError bool `json:"continue_on_error,omitempty"`  // 出错时是否继续
	
	// 超时控制
	StepTimeoutMs  int64 `json:"step_timeout_ms,omitempty"`  // 单步超时（毫秒）
	TotalTimeoutMs int64 `json:"total_timeout_ms,omitempty"` // 总超时（毫秒）
	
	// 审批要求
	RequireApproval     bool `json:"require_approval,omitempty"`      // 是否需要用户审批
	AutoApprove         bool `json:"auto_approve,omitempty"`          // 是否自动审批
	ApprovalTimeoutMs   int64 `json:"approval_timeout_ms,omitempty"`  // 审批超时（毫秒）
}

// NewExecutionPlan 创建新的执行计划
func NewExecutionPlan(description string) *ExecutionPlan {
	now := time.Now()
	return &ExecutionPlan{
		ID:          generatePlanID(),
		Description: description,
		Steps:       []Step{},
		Status:      StatusDraft,
		CurrentStep: 0,
		CreatedAt:   now,
		UpdatedAt:   now,
		Options: &ExecutionOptions{
			RequireApproval: true,
			StopOnError:     true,
		},
	}
}

// AddStep 添加步骤
func (p *ExecutionPlan) AddStep(toolName, description string, params map[string]any) *Step {
	step := Step{
		ID:          generateStepID(),
		Index:       len(p.Steps),
		ToolName:    toolName,
		Description: description,
		Parameters:  params,
		Status:      StepStatusPending,
	}
	p.Steps = append(p.Steps, step)
	p.UpdatedAt = time.Now()
	return &p.Steps[len(p.Steps)-1]
}

// GetStep 获取指定步骤
func (p *ExecutionPlan) GetStep(index int) *Step {
	if index < 0 || index >= len(p.Steps) {
		return nil
	}
	return &p.Steps[index]
}

// GetCurrentStep 获取当前步骤
func (p *ExecutionPlan) GetCurrentStep() *Step {
	return p.GetStep(p.CurrentStep)
}

// IsCompleted 检查计划是否已完成
func (p *ExecutionPlan) IsCompleted() bool {
	return p.Status == StatusCompleted || p.Status == StatusFailed || p.Status == StatusCancelled
}

// IsApproved 检查计划是否已审批
func (p *ExecutionPlan) IsApproved() bool {
	return p.UserApproved || (p.Options != nil && p.Options.AutoApprove)
}

// CanExecute 检查计划是否可以执行
func (p *ExecutionPlan) CanExecute() bool {
	if p.Status == StatusExecuting || p.IsCompleted() {
		return false
	}
	if p.Options != nil && p.Options.RequireApproval && !p.IsApproved() {
		return false
	}
	return true
}

// Approve 审批计划
func (p *ExecutionPlan) Approve(approvedBy string) {
	p.UserApproved = true
	p.ApprovedBy = approvedBy
	now := time.Now()
	p.ApprovedAt = &now
	p.Status = StatusApproved
	p.UpdatedAt = now
}

// Reject 拒绝计划
func (p *ExecutionPlan) Reject(note string) {
	p.UserApproved = false
	p.RejectionNote = note
	p.Status = StatusCancelled
	p.UpdatedAt = time.Now()
}

// MarkStepStarted 标记步骤开始
func (p *ExecutionPlan) MarkStepStarted(index int) {
	if step := p.GetStep(index); step != nil {
		now := time.Now()
		step.Status = StepStatusRunning
		step.StartedAt = &now
		p.CurrentStep = index
		p.UpdatedAt = now
	}
}

// MarkStepCompleted 标记步骤完成
func (p *ExecutionPlan) MarkStepCompleted(index int, result any) {
	if step := p.GetStep(index); step != nil {
		now := time.Now()
		step.Status = StepStatusCompleted
		step.Result = result
		step.CompletedAt = &now
		if step.StartedAt != nil {
			step.DurationMs = now.Sub(*step.StartedAt).Milliseconds()
		}
		p.UpdatedAt = now
	}
}

// MarkStepFailed 标记步骤失败
func (p *ExecutionPlan) MarkStepFailed(index int, err error) {
	if step := p.GetStep(index); step != nil {
		now := time.Now()
		step.Status = StepStatusFailed
		step.Error = err.Error()
		step.CompletedAt = &now
		if step.StartedAt != nil {
			step.DurationMs = now.Sub(*step.StartedAt).Milliseconds()
		}
		p.UpdatedAt = now
	}
}

// Summary 返回计划摘要
func (p *ExecutionPlan) Summary() PlanSummary {
	var completed, failed, pending, running int
	for _, step := range p.Steps {
		switch step.Status {
		case StepStatusCompleted:
			completed++
		case StepStatusFailed:
			failed++
		case StepStatusPending:
			pending++
		case StepStatusRunning:
			running++
		}
	}
	
	return PlanSummary{
		ID:          p.ID,
		Description: p.Description,
		Status:      p.Status,
		TotalSteps:  len(p.Steps),
		Completed:   completed,
		Failed:      failed,
		Pending:     pending,
		Running:     running,
		Progress:    float64(completed) / float64(len(p.Steps)) * 100,
	}
}

// PlanSummary 计划摘要
type PlanSummary struct {
	ID          string  `json:"id"`
	Description string  `json:"description"`
	Status      Status  `json:"status"`
	TotalSteps  int     `json:"total_steps"`
	Completed   int     `json:"completed"`
	Failed      int     `json:"failed"`
	Pending     int     `json:"pending"`
	Running     int     `json:"running"`
	Progress    float64 `json:"progress"` // 完成百分比
}

// generatePlanID 生成计划ID
func generatePlanID() string {
	return "plan_" + time.Now().Format("20060102150405") + "_" + randomString(6)
}

// generateStepID 生成步骤ID
func generateStepID() string {
	return "step_" + randomString(8)
}

// randomString 生成随机字符串
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}
