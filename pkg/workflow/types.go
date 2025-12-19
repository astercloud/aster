package workflow

import "time"

// StepType 步骤类型
type StepType string

const (
	StepTypeAgent     StepType = "agent"
	StepTypeRoom      StepType = "room"
	StepTypeFunction  StepType = "function"
	StepTypeCondition StepType = "condition"
	StepTypeLoop      StepType = "loop"
	StepTypeParallel  StepType = "parallel"
	StepTypeRouter    StepType = "router"
	StepTypeSteps     StepType = "steps"
)

// StepInput 步骤输入
type StepInput struct {
	Input               any
	PreviousStepContent any
	PreviousStepOutputs map[string]*StepOutput
	AdditionalData      map[string]any
	SessionState        map[string]any
	Images              []any
	Videos              []any
	Audio               []any
	Files               []any
	WorkflowSession     *WorkflowSession
}

func (si *StepInput) GetInputAsString() string {
	if si.Input == nil {
		return ""
	}
	if s, ok := si.Input.(string); ok {
		return s
	}
	return ""
}

func (si *StepInput) GetStepOutput(stepName string) *StepOutput {
	if si.PreviousStepOutputs == nil {
		return nil
	}
	return si.PreviousStepOutputs[stepName]
}

func (si *StepInput) GetStepContent(stepName string) any {
	output := si.GetStepOutput(stepName)
	if output == nil {
		return nil
	}
	return output.Content
}

// StepOutput 步骤输出
type StepOutput struct {
	StepID      string
	StepName    string
	StepType    StepType
	Content     any
	Error       error
	Metadata    map[string]any
	Metrics     *StepMetrics
	NestedSteps []*StepOutput
	StartTime   time.Time
	EndTime     time.Time
	Duration    float64
}

// StepMetrics 步骤指标
type StepMetrics struct {
	ExecutionTime float64
	InputTokens   int
	OutputTokens  int
	TotalTokens   int
	RetryCount    int
	Custom        map[string]any
}

// WorkflowInput Workflow 输入
type WorkflowInput struct {
	Input          any
	AdditionalData map[string]any
	Images         []any
	Videos         []any
	Audio          []any
	Files          []any
	SessionID      string
	UserID         string
	SessionState   map[string]any
}

// WorkflowOutput Workflow 输出
type WorkflowOutput struct {
	RunID        string
	WorkflowID   string
	WorkflowName string
	Content      any
	Error        error
	StepOutputs  map[string]*StepOutput
	SessionID    string
	SessionState map[string]any
	Metrics      *RunMetrics
	Status       RunStatus
	StartTime    time.Time
	EndTime      time.Time
	Duration     float64
}

// RunMetrics Workflow 运行指标
type RunMetrics struct {
	TotalExecutionTime float64
	TotalSteps         int
	SuccessfulSteps    int
	FailedSteps        int
	SkippedSteps       int
	TotalInputTokens   int
	TotalOutputTokens  int
	TotalTokens        int
	StepMetrics        map[string]*StepMetrics
}

// RunStatus 运行状态
type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
	RunStatusCancelled RunStatus = "canceled"
)

// WorkflowSession Workflow 会话
type WorkflowSession struct {
	ID         string
	WorkflowID string
	State      map[string]any
	History    []*WorkflowRun
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// WorkflowRun Workflow 运行记录
type WorkflowRun struct {
	RunID       string
	SessionID   string
	WorkflowID  string
	Input       any
	Output      any
	StepOutputs map[string]*StepOutput
	Status      RunStatus
	Error       string
	Metrics     *RunMetrics
	StartTime   time.Time
	EndTime     time.Time
	Duration    float64
}

// WorkflowEventType 事件类型
type WorkflowEventType string

const (
	EventWorkflowStarted   WorkflowEventType = "workflow_started"
	EventStepStarted       WorkflowEventType = "step_started"
	EventStepProgress      WorkflowEventType = "step_progress"
	EventStepCompleted     WorkflowEventType = "step_completed"
	EventStepFailed        WorkflowEventType = "step_failed"
	EventStepSkipped       WorkflowEventType = "step_skipped"
	EventWorkflowCompleted WorkflowEventType = "workflow_completed"
	EventWorkflowFailed    WorkflowEventType = "workflow_failed"
	EventWorkflowCancelled WorkflowEventType = "workflow_canceled"
)

// RunEvent Workflow 运行事件
type RunEvent struct {
	Type         WorkflowEventType
	EventID      string
	WorkflowID   string
	WorkflowName string
	RunID        string
	StepID       string
	StepName     string
	Data         any
	Timestamp    time.Time
	Metadata     map[string]any
}

// StepConfig 步骤配置
type StepConfig struct {
	ID                    string
	Name                  string
	Description           string
	Type                  StepType
	MaxRetries            int
	Timeout               time.Duration
	SkipOnError           bool
	StrictInputValidation bool
	Metadata              map[string]any
}
