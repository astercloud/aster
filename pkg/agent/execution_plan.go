package agent

import (
	"context"
	"errors"
	"fmt"

	"github.com/astercloud/aster/pkg/executionplan"
	"github.com/astercloud/aster/pkg/tools"
)

// ExecutionPlanManager 执行计划管理器
// 封装了 Agent 的执行计划功能
type ExecutionPlanManager struct {
	agent     *Agent
	generator *executionplan.Generator
	executor  *executionplan.Executor

	// 当前活动的执行计划
	currentPlan *executionplan.ExecutionPlan

	// 回调函数
	onPlanGenerated func(plan *executionplan.ExecutionPlan)
	onPlanApproved  func(plan *executionplan.ExecutionPlan)
	onPlanRejected  func(plan *executionplan.ExecutionPlan, reason string)
	onPlanCompleted func(plan *executionplan.ExecutionPlan)
}

// ExecutionPlanConfig 执行计划配置
type ExecutionPlanConfig struct {
	// Enabled 是否启用执行计划功能
	Enabled bool

	// RequireApproval 是否需要用户审批
	RequireApproval bool

	// AutoApprove 是否自动审批
	AutoApprove bool

	// StopOnError 出错时是否停止
	StopOnError bool

	// AllowParallel 是否允许并行执行
	AllowParallel bool

	// MaxParallelSteps 最大并行步骤数
	MaxParallelSteps int
}

// NewExecutionPlanManager 创建执行计划管理器
func NewExecutionPlanManager(agent *Agent) *ExecutionPlanManager {
	// 创建生成器和执行器
	generator := executionplan.NewGenerator(agent.provider, agent.toolMap)
	executor := executionplan.NewExecutor(
		agent.toolMap,
		executionplan.WithOnStepStart(func(plan *executionplan.ExecutionPlan, step *executionplan.Step) {
			agentLog.Debug(context.Background(), "execution plan step started", map[string]any{
				"plan_id":     plan.ID,
				"step_index":  step.Index,
				"step_id":     step.ID,
				"tool_name":   step.ToolName,
				"description": step.Description,
			})
		}),
		executionplan.WithOnStepComplete(func(plan *executionplan.ExecutionPlan, step *executionplan.Step) {
			agentLog.Debug(context.Background(), "execution plan step completed", map[string]any{
				"plan_id":     plan.ID,
				"step_index":  step.Index,
				"step_id":     step.ID,
				"duration_ms": step.DurationMs,
			})
		}),
		executionplan.WithOnStepFailed(func(plan *executionplan.ExecutionPlan, step *executionplan.Step, err error) {
			agentLog.Warn(context.Background(), "execution plan step failed", map[string]any{
				"plan_id":    plan.ID,
				"step_index": step.Index,
				"step_id":    step.ID,
				"error":      err.Error(),
			})
		}),
	)

	return &ExecutionPlanManager{
		agent:     agent,
		generator: generator,
		executor:  executor,
	}
}

// SetCallbacks 设置回调函数
func (m *ExecutionPlanManager) SetCallbacks(
	onGenerated func(*executionplan.ExecutionPlan),
	onApproved func(*executionplan.ExecutionPlan),
	onRejected func(*executionplan.ExecutionPlan, string),
	onCompleted func(*executionplan.ExecutionPlan),
) {
	m.onPlanGenerated = onGenerated
	m.onPlanApproved = onApproved
	m.onPlanRejected = onRejected
	m.onPlanCompleted = onCompleted
}

// GeneratePlan 生成执行计划
func (m *ExecutionPlanManager) GeneratePlan(ctx context.Context, request string, opts *ExecutionPlanConfig) (*executionplan.ExecutionPlan, error) {
	// 构建执行选项
	var execOpts *executionplan.ExecutionOptions
	if opts != nil {
		execOpts = &executionplan.ExecutionOptions{
			RequireApproval:  opts.RequireApproval,
			AutoApprove:      opts.AutoApprove,
			StopOnError:      opts.StopOnError,
			AllowParallel:    opts.AllowParallel,
			MaxParallelSteps: opts.MaxParallelSteps,
		}
	}

	// 生成计划
	plan, err := m.generator.Generate(ctx, &executionplan.PlanRequest{
		UserRequest: request,
		Options:     execOpts,
		Metadata: map[string]any{
			"agent_id": m.agent.id,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("generate execution plan: %w", err)
	}

	// 设置 Agent ID
	plan.AgentID = m.agent.id

	// 保存当前计划
	m.currentPlan = plan

	// 触发回调
	if m.onPlanGenerated != nil {
		m.onPlanGenerated(plan)
	}

	agentLog.Info(ctx, "execution plan generated", map[string]any{
		"plan_id":     plan.ID,
		"description": plan.Description,
		"step_count":  len(plan.Steps),
		"status":      plan.Status,
	})

	return plan, nil
}

// ApprovePlan 审批执行计划
func (m *ExecutionPlanManager) ApprovePlan(approvedBy string) error {
	if m.currentPlan == nil {
		return errors.New("no pending plan to approve")
	}

	if m.currentPlan.Status != executionplan.StatusPendingApproval {
		return fmt.Errorf("plan is not pending approval, current status: %s", m.currentPlan.Status)
	}

	m.currentPlan.Approve(approvedBy)

	// 触发回调
	if m.onPlanApproved != nil {
		m.onPlanApproved(m.currentPlan)
	}

	agentLog.Info(context.Background(), "execution plan approved", map[string]any{
		"plan_id":     m.currentPlan.ID,
		"approved_by": approvedBy,
	})

	return nil
}

// RejectPlan 拒绝执行计划
func (m *ExecutionPlanManager) RejectPlan(reason string) error {
	if m.currentPlan == nil {
		return errors.New("no pending plan to reject")
	}

	m.currentPlan.Reject(reason)

	// 触发回调
	if m.onPlanRejected != nil {
		m.onPlanRejected(m.currentPlan, reason)
	}

	agentLog.Info(context.Background(), "execution plan rejected", map[string]any{
		"plan_id": m.currentPlan.ID,
		"reason":  reason,
	})

	return nil
}

// ExecutePlan 执行当前计划
func (m *ExecutionPlanManager) ExecutePlan(ctx context.Context) error {
	if m.currentPlan == nil {
		return errors.New("no plan to execute")
	}

	// 创建工具上下文
	toolCtx := &tools.ToolContext{
		AgentID: m.agent.id,
		Sandbox: m.agent.sandbox,
		Signal:  ctx,
	}

	// 执行计划
	err := m.executor.Execute(ctx, m.currentPlan, toolCtx)

	// 触发完成回调
	if m.onPlanCompleted != nil {
		m.onPlanCompleted(m.currentPlan)
	}

	agentLog.Info(ctx, "execution plan completed", map[string]any{
		"plan_id":           m.currentPlan.ID,
		"status":            m.currentPlan.Status,
		"total_duration_ms": m.currentPlan.TotalDurationMs,
	})

	return err
}

// ExecutePlanDirect 直接执行指定计划（不设置为当前计划）
func (m *ExecutionPlanManager) ExecutePlanDirect(ctx context.Context, plan *executionplan.ExecutionPlan) error {
	// 创建工具上下文
	toolCtx := &tools.ToolContext{
		AgentID: m.agent.id,
		Sandbox: m.agent.sandbox,
		Signal:  ctx,
	}

	return m.executor.Execute(ctx, plan, toolCtx)
}

// GetCurrentPlan 获取当前执行计划
func (m *ExecutionPlanManager) GetCurrentPlan() *executionplan.ExecutionPlan {
	return m.currentPlan
}

// GetPlanSummary 获取当前计划摘要
func (m *ExecutionPlanManager) GetPlanSummary() *executionplan.PlanSummary {
	if m.currentPlan == nil {
		return nil
	}
	summary := m.currentPlan.Summary()
	return &summary
}

// FormatCurrentPlan 格式化当前计划为可读文本
func (m *ExecutionPlanManager) FormatCurrentPlan() string {
	if m.currentPlan == nil {
		return "No active execution plan"
	}
	return executionplan.FormatPlan(m.currentPlan)
}

// ValidateCurrentPlan 验证当前计划
func (m *ExecutionPlanManager) ValidateCurrentPlan() []error {
	if m.currentPlan == nil {
		return []error{errors.New("no plan to validate")}
	}
	return m.generator.ValidatePlan(m.currentPlan)
}

// CancelPlan 取消当前计划
func (m *ExecutionPlanManager) CancelPlan(reason string) {
	if m.currentPlan == nil {
		return
	}
	m.executor.Cancel(m.currentPlan, reason)

	agentLog.Info(context.Background(), "execution plan canceled", map[string]any{
		"plan_id": m.currentPlan.ID,
		"reason":  reason,
	})
}

// ResumePlan 恢复执行当前计划
func (m *ExecutionPlanManager) ResumePlan(ctx context.Context) error {
	if m.currentPlan == nil {
		return errors.New("no plan to resume")
	}

	toolCtx := &tools.ToolContext{
		AgentID: m.agent.id,
		Sandbox: m.agent.sandbox,
		Signal:  ctx,
	}

	return m.executor.Resume(ctx, m.currentPlan, toolCtx)
}

// ClearPlan 清除当前计划
func (m *ExecutionPlanManager) ClearPlan() {
	m.currentPlan = nil
}
