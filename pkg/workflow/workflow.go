package workflow

import (
	"context"
	"fmt"
	"iter"
	"time"

	"github.com/astercloud/aster/pkg/store"
	"github.com/google/uuid"
)

// Workflow 统一的 Workflow 执行系统
type Workflow struct {
	// 标识
	ID          string
	Name        string
	Description string

	// 步骤
	Steps []Step

	// 数据库
	DB store.Store

	// 会话
	SessionID    string
	UserID       string
	SessionState map[string]interface{}
	CacheSession bool

	// 配置
	MaxRetries int
	Timeout    time.Duration
	RetryDelay time.Duration

	// 流式
	Stream               bool
	StreamEvents         bool
	StreamExecutorEvents bool

	// 调试
	DebugMode bool

	// 存储
	StoreEvents          bool
	StoreExecutorOutputs bool
	SkipEvents           []WorkflowEventType

	// 输入验证
	InputSchema interface{} // Type for input validation

	// 元数据
	Metadata map[string]interface{}

	// 历史
	AddWorkflowHistory bool
	NumHistoryRuns     int

	// 内部状态
	workflowSession *WorkflowSession
}

// New 创建新的 Workflow
func New(name string) *Workflow {
	return &Workflow{
		ID:                   uuid.New().String(),
		Name:                 name,
		Steps:                make([]Step, 0),
		MaxRetries:           3,
		Timeout:              30 * time.Minute,
		RetryDelay:           1 * time.Second,
		Stream:               false,
		StreamEvents:         false,
		StreamExecutorEvents: true,
		DebugMode:            false,
		StoreEvents:          false,
		StoreExecutorOutputs: true,
		SkipEvents:           make([]WorkflowEventType, 0),
		Metadata:             make(map[string]interface{}),
		AddWorkflowHistory:   false,
		NumHistoryRuns:       3,
		CacheSession:         false,
	}
}

// ===== 流式 API =====

// AddStep 添加步骤
func (w *Workflow) AddStep(step Step) *Workflow {
	w.Steps = append(w.Steps, step)
	return w
}

// WithStream 启用流式
func (w *Workflow) WithStream() *Workflow {
	w.Stream = true
	w.StreamEvents = true
	return w
}

// WithDebug 启用调试
func (w *Workflow) WithDebug() *Workflow {
	w.DebugMode = true
	return w
}

// WithTimeout 设置超时
func (w *Workflow) WithTimeout(timeout time.Duration) *Workflow {
	w.Timeout = timeout
	return w
}

// WithMetadata 添加元数据
func (w *Workflow) WithMetadata(key string, value interface{}) *Workflow {
	if w.Metadata == nil {
		w.Metadata = make(map[string]interface{})
	}
	w.Metadata[key] = value
	return w
}

// WithDB 设置数据库
func (w *Workflow) WithDB(db store.Store) *Workflow {
	w.DB = db
	return w
}

// WithHistory 启用历史记录
func (w *Workflow) WithHistory(numRuns int) *Workflow {
	w.AddWorkflowHistory = true
	w.NumHistoryRuns = numRuns
	return w
}

// WithSession 设置会话
func (w *Workflow) WithSession(sessionID string) *Workflow {
	w.SessionID = sessionID
	w.CacheSession = true
	return w
}

// ===== 验证 =====

// Validate 验证配置
func (w *Workflow) Validate() error {
	if w.Name == "" {
		return fmt.Errorf("workflow name is required")
	}
	if len(w.Steps) == 0 {
		return fmt.Errorf("workflow must have at least one step")
	}

	stepNames := make(map[string]bool)
	for _, step := range w.Steps {
		if stepNames[step.Name()] {
			return fmt.Errorf("duplicate step name: %s", step.Name())
		}
		stepNames[step.Name()] = true
	}

	if w.AddWorkflowHistory && w.DB == nil {
		// 警告：启用了历史但没有数据库
		fmt.Println("Warning: workflow history enabled but no database configured")
	}

	return nil
}

// ValidateInput 验证输入
func (w *Workflow) ValidateInput(input interface{}) error {
	if w.InputSchema == nil {
		return nil
	}
	// TODO: 实现输入验证逻辑
	return nil
}

// ===== 会话管理 =====

// InitializeSession 初始化会话
func (w *Workflow) InitializeSession(sessionID, userID string) (string, string) {
	if sessionID == "" {
		if w.SessionID != "" {
			sessionID = w.SessionID
		} else {
			sessionID = uuid.New().String()
			w.SessionID = sessionID
		}
	}

	if userID == "" {
		userID = w.UserID
	}

	return sessionID, userID
}

// GetSession 获取会话
func (w *Workflow) GetSession(sessionID string) (*WorkflowSession, error) {
	if sessionID == "" {
		sessionID = w.SessionID
	}

	if sessionID == "" {
		return nil, fmt.Errorf("no session_id provided")
	}

	// 从缓存获取
	if w.workflowSession != nil && w.workflowSession.ID == sessionID {
		return w.workflowSession, nil
	}

	// 从数据库获取（TODO: 实现从数据库读取会话）
	_ = w.DB // 数据库功能待实现

	return nil, fmt.Errorf("session %s not found", sessionID)
}

// CreateSession 创建会话
func (w *Workflow) CreateSession(sessionID, userID string) *WorkflowSession {
	session := &WorkflowSession{
		ID:         sessionID,
		WorkflowID: w.ID,
		State:      make(map[string]interface{}),
		History:    make([]*WorkflowRun, 0),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// 初始化会话状态
	if w.SessionState != nil {
		for k, v := range w.SessionState {
			session.State[k] = v
		}
	}

	// 缓存会话
	if w.CacheSession {
		w.workflowSession = session
	}

	return session
}

// SaveSession 保存会话
func (w *Workflow) SaveSession(session *WorkflowSession) error {
	if w.DB == nil {
		return nil
	}

	// TODO: 实现保存会话到数据库
	session.UpdatedAt = time.Now()
	return nil
}

// GetOrCreateSession 获取或创建会话
func (w *Workflow) GetOrCreateSession(sessionID, userID string) *WorkflowSession {
	session, err := w.GetSession(sessionID)
	if err == nil && session != nil {
		return session
	}
	return w.CreateSession(sessionID, userID)
}

// ===== 运行管理 =====

// GetRun 获取运行记录
func (w *Workflow) GetRun(runID string) (*WorkflowRun, error) {
	if w.workflowSession == nil {
		return nil, fmt.Errorf("no active session")
	}

	for _, run := range w.workflowSession.History {
		if run.RunID == runID {
			return run, nil
		}
	}

	return nil, fmt.Errorf("run %s not found", runID)
}

// GetLastRun 获取最后一次运行
func (w *Workflow) GetLastRun() (*WorkflowRun, error) {
	if w.workflowSession == nil {
		return nil, fmt.Errorf("no active session")
	}

	if len(w.workflowSession.History) == 0 {
		return nil, fmt.Errorf("no runs found")
	}

	return w.workflowSession.History[len(w.workflowSession.History)-1], nil
}

// SaveRun 保存运行记录
func (w *Workflow) SaveRun(run *WorkflowRun) error {
	if w.workflowSession != nil {
		w.workflowSession.History = append(w.workflowSession.History, run)
		return w.SaveSession(w.workflowSession)
	}
	return nil
}

// ===== 执行 =====

// Execute 执行 Workflow
func (w *Workflow) Execute(ctx context.Context, input *WorkflowInput) iter.Seq2[*RunEvent, error] {
	return func(yield func(*RunEvent, error) bool) {
		// 验证输入
		if err := w.ValidateInput(input.Input); err != nil {
			yield(nil, fmt.Errorf("input validation failed: %w", err))
			return
		}

		// 初始化会话
		sessionID, userID := w.InitializeSession(input.SessionID, input.UserID)
		session := w.GetOrCreateSession(sessionID, userID)

		// 生成 RunID
		runID := uuid.New().String()
		startTime := time.Now()

		// 创建运行记录
		run := &WorkflowRun{
			RunID:       runID,
			SessionID:   sessionID,
			WorkflowID:  w.ID,
			Input:       input.Input,
			StepOutputs: make(map[string]*StepOutput),
			Status:      RunStatusRunning,
			StartTime:   startTime,
			Metrics:     &RunMetrics{TotalSteps: len(w.Steps), StepMetrics: make(map[string]*StepMetrics)},
		}

		// 发送开始事件
		if !yield(&RunEvent{
			Type:         EventWorkflowStarted,
			EventID:      uuid.New().String(),
			WorkflowID:   w.ID,
			WorkflowName: w.Name,
			RunID:        runID,
			Timestamp:    startTime,
			Data: map[string]interface{}{
				"input":      input.Input,
				"session_id": sessionID,
				"user_id":    userID,
			},
		}, nil) {
			return
		}

		// 合并会话状态
		sessionState := make(map[string]interface{})
		if session.State != nil {
			for k, v := range session.State {
				sessionState[k] = v
			}
		}
		if input.SessionState != nil {
			for k, v := range input.SessionState {
				sessionState[k] = v
			}
		}

		stepOutputs := make(map[string]*StepOutput)
		var lastOutput *StepOutput

		// 执行步骤
		for i, step := range w.Steps {
			stepInput := &StepInput{
				Input:               input.Input,
				PreviousStepContent: nil,
				PreviousStepOutputs: stepOutputs,
				AdditionalData:      input.AdditionalData,
				SessionState:        sessionState,
				Images:              input.Images,
				Videos:              input.Videos,
				Audio:               input.Audio,
				Files:               input.Files,
				WorkflowSession:     session,
			}

			if lastOutput != nil {
				stepInput.PreviousStepContent = lastOutput.Content
			}

			stepStartTime := time.Now()
			if w.StreamEvents {
				yield(&RunEvent{
					Type:         EventStepStarted,
					EventID:      uuid.New().String(),
					WorkflowID:   w.ID,
					WorkflowName: w.Name,
					RunID:        runID,
					StepID:       step.ID(),
					StepName:     step.Name(),
					Timestamp:    stepStartTime,
					Data: map[string]interface{}{
						"index": i,
						"type":  step.Type(),
					},
				}, nil)
			}

			var stepOutput *StepOutput
			var stepError error

			for output, err := range step.Execute(ctx, stepInput) {
				if err != nil {
					stepError = err
					break
				}
				stepOutput = output

				// 流式进度事件
				if w.StreamEvents && w.StreamExecutorEvents && output != nil {
					yield(&RunEvent{
						Type:         EventStepProgress,
						EventID:      uuid.New().String(),
						WorkflowID:   w.ID,
						WorkflowName: w.Name,
						RunID:        runID,
						StepID:       step.ID(),
						StepName:     step.Name(),
						Timestamp:    time.Now(),
						Data:         output,
					}, nil)
				}
			}

			stepEndTime := time.Now()

			if stepError != nil {
				run.Metrics.FailedSteps++

				if w.StreamEvents {
					yield(&RunEvent{
						Type:         EventStepFailed,
						EventID:      uuid.New().String(),
						WorkflowID:   w.ID,
						WorkflowName: w.Name,
						RunID:        runID,
						StepID:       step.ID(),
						StepName:     step.Name(),
						Timestamp:    stepEndTime,
						Data: map[string]interface{}{
							"error":    stepError.Error(),
							"duration": stepEndTime.Sub(stepStartTime).Seconds(),
						},
					}, nil)
				}

				if !step.Config().SkipOnError {
					// 终止执行
					run.Status = RunStatusFailed
					run.Error = stepError.Error()
					run.EndTime = time.Now()
					run.Duration = run.EndTime.Sub(run.StartTime).Seconds()
					run.Metrics.TotalExecutionTime = run.Duration

					_ = w.SaveRun(run)

					yield(&RunEvent{
						Type:         EventWorkflowFailed,
						EventID:      uuid.New().String(),
						WorkflowID:   w.ID,
						WorkflowName: w.Name,
						RunID:        runID,
						Timestamp:    run.EndTime,
						Data: map[string]interface{}{
							"error":    stepError.Error(),
							"duration": run.Duration,
							"metrics":  run.Metrics,
						},
					}, stepError)
					return
				}

				run.Metrics.SkippedSteps++
				continue
			}

			run.Metrics.SuccessfulSteps++
			if stepOutput != nil {
				stepOutputs[step.Name()] = stepOutput
				run.StepOutputs[step.Name()] = stepOutput
				lastOutput = stepOutput

				if stepOutput.Metrics != nil {
					run.Metrics.StepMetrics[step.Name()] = stepOutput.Metrics
					run.Metrics.TotalInputTokens += stepOutput.Metrics.InputTokens
					run.Metrics.TotalOutputTokens += stepOutput.Metrics.OutputTokens
				}
			}

			if w.StreamEvents {
				yield(&RunEvent{
					Type:         EventStepCompleted,
					EventID:      uuid.New().String(),
					WorkflowID:   w.ID,
					WorkflowName: w.Name,
					RunID:        runID,
					StepID:       step.ID(),
					StepName:     step.Name(),
					Timestamp:    stepEndTime,
					Data: map[string]interface{}{
						"output":   stepOutput,
						"duration": stepEndTime.Sub(stepStartTime).Seconds(),
					},
				}, nil)
			}

			// 检查上下文取消
			if ctx.Err() != nil {
				run.Status = RunStatusCancelled
				run.EndTime = time.Now()
				run.Duration = run.EndTime.Sub(run.StartTime).Seconds()

				_ = w.SaveRun(run) // 显式忽略最后一个 SaveRun 错误

				yield(&RunEvent{
					Type:         EventWorkflowCancelled,
					EventID:      uuid.New().String(),
					WorkflowID:   w.ID,
					WorkflowName: w.Name,
					RunID:        runID,
					Timestamp:    run.EndTime,
					Data: map[string]interface{}{
						"reason": ctx.Err().Error(),
					},
				}, ctx.Err())
				return
			}
		}

		// 完成
		run.Status = RunStatusCompleted
		run.EndTime = time.Now()
		run.Duration = run.EndTime.Sub(run.StartTime).Seconds()
		run.Metrics.TotalExecutionTime = run.Duration
		run.Metrics.TotalTokens = run.Metrics.TotalInputTokens + run.Metrics.TotalOutputTokens

		if lastOutput != nil {
			run.Output = lastOutput.Content
		}

		// 更新会话状态
		session.State = sessionState
		_ = w.SaveRun(run)

		yield(&RunEvent{
			Type:         EventWorkflowCompleted,
			EventID:      uuid.New().String(),
			WorkflowID:   w.ID,
			WorkflowName: w.Name,
			RunID:        runID,
			Timestamp:    run.EndTime,
			Data: map[string]interface{}{
				"output":       run.Output,
				"duration":     run.Duration,
				"metrics":      run.Metrics,
				"session_id":   sessionID,
				"step_outputs": stepOutputs,
			},
		}, nil)
	}
}

// ===== 辅助方法 =====

// GetWorkflowData 获取 Workflow 数据
func (w *Workflow) GetWorkflowData() map[string]interface{} {
	return map[string]interface{}{
		"id":          w.ID,
		"name":        w.Name,
		"description": w.Description,
		"metadata":    w.Metadata,
	}
}
