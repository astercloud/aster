package executionplan

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/tools"
)

// Executor 执行计划执行器
type Executor struct {
	tools map[string]tools.Tool // 工具实例映射

	// 回调函数
	onStepStart    func(plan *ExecutionPlan, step *Step)
	onStepComplete func(plan *ExecutionPlan, step *Step)
	onStepFailed   func(plan *ExecutionPlan, step *Step, err error)
	onPlanComplete func(plan *ExecutionPlan)
}

// ExecutorOption 执行器选项
type ExecutorOption func(*Executor)

// WithOnStepStart 设置步骤开始回调
func WithOnStepStart(fn func(plan *ExecutionPlan, step *Step)) ExecutorOption {
	return func(e *Executor) {
		e.onStepStart = fn
	}
}

// WithOnStepComplete 设置步骤完成回调
func WithOnStepComplete(fn func(plan *ExecutionPlan, step *Step)) ExecutorOption {
	return func(e *Executor) {
		e.onStepComplete = fn
	}
}

// WithOnStepFailed 设置步骤失败回调
func WithOnStepFailed(fn func(plan *ExecutionPlan, step *Step, err error)) ExecutorOption {
	return func(e *Executor) {
		e.onStepFailed = fn
	}
}

// WithOnPlanComplete 设置计划完成回调
func WithOnPlanComplete(fn func(plan *ExecutionPlan)) ExecutorOption {
	return func(e *Executor) {
		e.onPlanComplete = fn
	}
}

// NewExecutor 创建执行计划执行器
// toolMap: 工具名称到工具实例的映射
func NewExecutor(toolMap map[string]tools.Tool, opts ...ExecutorOption) *Executor {
	e := &Executor{
		tools: toolMap,
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// Execute 执行计划
func (e *Executor) Execute(ctx context.Context, plan *ExecutionPlan, toolCtx *tools.ToolContext) error {
	// 检查计划是否可以执行
	if !plan.CanExecute() {
		if plan.Options != nil && plan.Options.RequireApproval && !plan.IsApproved() {
			return fmt.Errorf("plan requires user approval before execution")
		}
		if plan.Status == StatusExecuting {
			return fmt.Errorf("plan is already executing")
		}
		if plan.IsCompleted() {
			return fmt.Errorf("plan has already completed with status: %s", plan.Status)
		}
		return fmt.Errorf("plan cannot be executed in current state: %s", plan.Status)
	}

	// 更新计划状态
	now := time.Now()
	plan.Status = StatusExecuting
	plan.StartedAt = &now
	plan.UpdatedAt = now

	// 根据配置选择执行方式
	var err error
	if plan.Options != nil && plan.Options.AllowParallel {
		err = e.executeParallel(ctx, plan, toolCtx)
	} else {
		err = e.executeSequential(ctx, plan, toolCtx)
	}

	// 更新计划完成状态
	completedAt := time.Now()
	plan.CompletedAt = &completedAt
	plan.TotalDurationMs = completedAt.Sub(*plan.StartedAt).Milliseconds()
	plan.UpdatedAt = completedAt

	// 确定最终状态
	summary := plan.Summary()
	if summary.Failed > 0 {
		if summary.Completed > 0 {
			plan.Status = StatusPartial
		} else {
			plan.Status = StatusFailed
		}
	} else if summary.Completed == summary.TotalSteps {
		plan.Status = StatusCompleted
	}

	// 触发计划完成回调
	if e.onPlanComplete != nil {
		e.onPlanComplete(plan)
	}

	return err
}

// executeSequential 顺序执行步骤
func (e *Executor) executeSequential(ctx context.Context, plan *ExecutionPlan, toolCtx *tools.ToolContext) error {
	var firstError error

	for i := range plan.Steps {
		select {
		case <-ctx.Done():
			// 上下文取消，标记剩余步骤为跳过
			for j := i; j < len(plan.Steps); j++ {
				plan.Steps[j].Status = StepStatusSkipped
			}
			return ctx.Err()
		default:
		}

		step := &plan.Steps[i]

		// 跳过已完成的步骤（用于恢复执行）
		if step.Status == StepStatusCompleted {
			continue
		}

		// 检查依赖是否满足
		if !e.checkDependencies(plan, step) {
			step.Status = StepStatusSkipped
			step.Error = "dependencies not satisfied"
			continue
		}

		// 执行步骤
		err := e.executeStep(ctx, plan, step, toolCtx)
		if err != nil {
			if firstError == nil {
				firstError = err
			}

			// 根据配置决定是否继续
			if plan.Options != nil && plan.Options.StopOnError {
				// 标记剩余步骤为跳过
				for j := i + 1; j < len(plan.Steps); j++ {
					plan.Steps[j].Status = StepStatusSkipped
				}
				return err
			}
		}
	}

	return firstError
}

// executeParallel 并行执行无依赖的步骤
func (e *Executor) executeParallel(ctx context.Context, plan *ExecutionPlan, toolCtx *tools.ToolContext) error {
	// 构建步骤依赖图
	completed := make(map[string]bool)
	var mu sync.Mutex
	var wg sync.WaitGroup
	var firstError error
	var errMu sync.Mutex

	maxParallel := plan.Options.MaxParallelSteps
	if maxParallel <= 0 {
		maxParallel = 3 // 默认最多3个并行
	}
	sem := make(chan struct{}, maxParallel)

	for {
		// 找出所有可以执行的步骤
		var readySteps []*Step
		mu.Lock()
		for i := range plan.Steps {
			step := &plan.Steps[i]
			if step.Status != StepStatusPending {
				continue
			}
			if e.checkDependenciesWithMap(step, completed) {
				readySteps = append(readySteps, step)
			}
		}
		mu.Unlock()

		// 如果没有可执行的步骤，检查是否全部完成
		if len(readySteps) == 0 {
			// 检查是否还有未完成的步骤
			allDone := true
			for i := range plan.Steps {
				if plan.Steps[i].Status == StepStatusPending || plan.Steps[i].Status == StepStatusRunning {
					allDone = false
					break
				}
			}
			if allDone {
				break
			}
			// 等待正在执行的步骤完成
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// 并行执行准备好的步骤
		for _, step := range readySteps {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case sem <- struct{}{}:
			}

			wg.Add(1)
			go func(s *Step) {
				defer wg.Done()
				defer func() { <-sem }()

				err := e.executeStep(ctx, plan, s, toolCtx)

				mu.Lock()
				if s.Status == StepStatusCompleted {
					completed[s.ID] = true
				}
				mu.Unlock()

				if err != nil {
					errMu.Lock()
					if firstError == nil {
						firstError = err
					}
					errMu.Unlock()

					// 如果配置为出错停止，取消后续步骤
					if plan.Options != nil && plan.Options.StopOnError {
						mu.Lock()
						for i := range plan.Steps {
							if plan.Steps[i].Status == StepStatusPending {
								plan.Steps[i].Status = StepStatusSkipped
							}
						}
						mu.Unlock()
					}
				}
			}(step)
		}

		// 等待本批次完成再查找下一批
		wg.Wait()

		// 检查是否有错误且需要停止
		errMu.Lock()
		stopOnErr := firstError != nil && plan.Options != nil && plan.Options.StopOnError
		errMu.Unlock()
		if stopOnErr {
			break
		}
	}

	return firstError
}

// executeStep 执行单个步骤
func (e *Executor) executeStep(ctx context.Context, plan *ExecutionPlan, step *Step, toolCtx *tools.ToolContext) error {
	// 获取工具
	tool, ok := e.tools[step.ToolName]
	if !ok {
		step.Status = StepStatusFailed
		step.Error = fmt.Sprintf("tool not found: %s", step.ToolName)
		return fmt.Errorf("tool not found: %s", step.ToolName)
	}

	// 标记步骤开始
	plan.MarkStepStarted(step.Index)

	// 触发步骤开始回调
	if e.onStepStart != nil {
		e.onStepStart(plan, step)
	}

	// 准备输入参数
	var inputParams map[string]any
	if len(step.Parameters) > 0 {
		inputParams = step.Parameters
	} else if step.Input != "" {
		// 尝试从 Input 字符串解析 JSON
		inputParams = make(map[string]any)
		if err := json.Unmarshal([]byte(step.Input), &inputParams); err != nil {
			// 如果解析失败，将整个输入作为 "input" 参数
			inputParams["input"] = step.Input
		}
	} else {
		inputParams = make(map[string]any)
	}

	// 执行工具（带超时）
	execCtx := ctx
	if plan.Options != nil && plan.Options.StepTimeoutMs > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, time.Duration(plan.Options.StepTimeoutMs)*time.Millisecond)
		defer cancel()
	}

	// 重试逻辑
	var result any
	var execErr error
	maxRetries := step.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 0 // 默认不重试
	}

retryLoop:
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			step.RetryCount = attempt
			// 重试延迟
			if step.RetryDelayMs > 0 {
				select {
				case <-execCtx.Done():
					execErr = execCtx.Err()
					break retryLoop
				case <-time.After(time.Duration(step.RetryDelayMs) * time.Millisecond):
				}
			}
		}

		result, execErr = tool.Execute(execCtx, inputParams, toolCtx)
		if execErr == nil {
			break
		}
	}

	if execErr != nil {
		plan.MarkStepFailed(step.Index, execErr)
		if e.onStepFailed != nil {
			e.onStepFailed(plan, step, execErr)
		}
		return execErr
	}

	// 标记步骤完成
	plan.MarkStepCompleted(step.Index, result)

	// 触发步骤完成回调
	if e.onStepComplete != nil {
		e.onStepComplete(plan, step)
	}

	return nil
}

// checkDependencies 检查步骤依赖是否满足
func (e *Executor) checkDependencies(plan *ExecutionPlan, step *Step) bool {
	if len(step.DependsOn) == 0 {
		return true
	}

	for _, depID := range step.DependsOn {
		found := false
		for i := range plan.Steps {
			if plan.Steps[i].ID == depID {
				if plan.Steps[i].Status != StepStatusCompleted {
					return false
				}
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// checkDependenciesWithMap 使用完成映射检查依赖
func (e *Executor) checkDependenciesWithMap(step *Step, completed map[string]bool) bool {
	if len(step.DependsOn) == 0 {
		return true
	}

	for _, depID := range step.DependsOn {
		if !completed[depID] {
			return false
		}
	}

	return true
}

// Resume 恢复执行（从当前步骤继续）
func (e *Executor) Resume(ctx context.Context, plan *ExecutionPlan, toolCtx *tools.ToolContext) error {
	// 找到第一个未完成的步骤
	startIndex := -1
	for i := range plan.Steps {
		if plan.Steps[i].Status == StepStatusPending || plan.Steps[i].Status == StepStatusFailed {
			startIndex = i
			break
		}
	}

	if startIndex == -1 {
		// 所有步骤已完成
		plan.Status = StatusCompleted
		plan.UpdatedAt = time.Now()
		return nil
	}

	// 重置失败步骤的状态
	for i := startIndex; i < len(plan.Steps); i++ {
		if plan.Steps[i].Status == StepStatusFailed || plan.Steps[i].Status == StepStatusSkipped {
			plan.Steps[i].Status = StepStatusPending
			plan.Steps[i].Error = ""
			plan.Steps[i].Result = nil
		}
	}

	// 重置计划状态以允许执行
	// 如果需要审批，设置为已审批；否则设置为草稿
	if plan.Options != nil && plan.Options.RequireApproval {
		plan.Status = StatusApproved
		plan.UserApproved = true
	} else {
		plan.Status = StatusDraft
	}
	plan.UpdatedAt = time.Now()

	// 继续执行
	return e.Execute(ctx, plan, toolCtx)
}

// Cancel 取消执行
func (e *Executor) Cancel(plan *ExecutionPlan, reason string) {
	plan.Status = StatusCancelled
	plan.UpdatedAt = time.Now()

	// 标记所有待执行和执行中的步骤为跳过
	for i := range plan.Steps {
		if plan.Steps[i].Status == StepStatusPending || plan.Steps[i].Status == StepStatusRunning {
			plan.Steps[i].Status = StepStatusSkipped
			plan.Steps[i].Error = reason
		}
	}
}
