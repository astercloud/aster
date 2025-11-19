package workflow

import (
	"context"
	"fmt"
	"iter"
	"time"

	"github.com/google/uuid"
)

// Router 动态路由器 - 根据输入动态选择要执行的步骤
// 类似 agno 的 Router，支持返回多个步骤并顺序链接执行
type Router struct {
	id          string
	name        string
	description string
	selector    func(*StepInput) []Step // 选择器函数，返回要执行的步骤列表
	choices     []Step                  // 可选的步骤池
	config      *StepConfig
}

// NewRouter 创建新的 Router
func NewRouter(name string, selector func(*StepInput) []Step, choices []Step) *Router {
	return &Router{
		id:       uuid.New().String(),
		name:     name,
		selector: selector,
		choices:  choices,
		config: &StepConfig{
			Name:        name,
			Type:        StepTypeRouter,
			MaxRetries:  1,
			Timeout:     10 * time.Minute,
			SkipOnError: false,
		},
	}
}

func (r *Router) ID() string          { return r.id }
func (r *Router) Name() string        { return r.name }
func (r *Router) Type() StepType      { return StepTypeRouter }
func (r *Router) Description() string { return r.description }
func (r *Router) Config() *StepConfig { return r.config }

// Execute 执行 Router - 选择步骤并顺序链接执行
func (r *Router) Execute(ctx context.Context, input *StepInput) iter.Seq2[*StepOutput, error] {
	return func(yield func(*StepOutput, error) bool) {
		startTime := time.Now()

		// 调用选择器函数
		stepsToExecute := r.selector(input)

		if len(stepsToExecute) == 0 {
			// 没有选中任何步骤
			output := &StepOutput{
				StepID:      r.id,
				StepName:    r.name,
				StepType:    StepTypeRouter,
				Content:     fmt.Sprintf("Router %s: no steps selected", r.name),
				StartTime:   startTime,
				EndTime:     time.Now(),
				NestedSteps: []*StepOutput{},
				Metadata: map[string]interface{}{
					"selected_steps": 0,
				},
				Metrics: &StepMetrics{ExecutionTime: time.Since(startTime).Seconds()},
			}
			output.Duration = output.EndTime.Sub(output.StartTime).Seconds()
			yield(output, nil)
			return
		}

		// 收集所有执行结果
		allResults := make([]*StepOutput, 0, len(stepsToExecute))
		currentInput := input
		routerStepOutputs := make(map[string]*StepOutput)

		// 顺序执行选中的步骤（链接模式）
		for i, step := range stepsToExecute {
			// 更新输入：使用前一步的输出
			if i > 0 && len(allResults) > 0 {
				lastOutput := allResults[len(allResults)-1]
				currentInput = &StepInput{
					Input:               input.Input,
					PreviousStepContent: lastOutput.Content,
					PreviousStepOutputs: routerStepOutputs,
					AdditionalData:      input.AdditionalData,
					SessionState:        input.SessionState,
					Images:              input.Images,
					Videos:              input.Videos,
					Audio:               input.Audio,
					Files:               input.Files,
					WorkflowSession:     input.WorkflowSession,
				}
			}

			// 执行步骤
			var stepOutput *StepOutput
			var stepError error

			for output, err := range step.Execute(ctx, currentInput) {
				if err != nil {
					stepError = err
					break
				}
				stepOutput = output
			}

			if stepError != nil {
				// 步骤执行失败
				errorOutput := &StepOutput{
					StepID:      r.id,
					StepName:    r.name,
					StepType:    StepTypeRouter,
					Error:       stepError,
					StartTime:   startTime,
					EndTime:     time.Now(),
					NestedSteps: allResults,
					Metadata: map[string]interface{}{
						"selected_steps": len(stepsToExecute),
						"executed_steps": i,
						"failed_step":    step.Name(),
						"error":          stepError.Error(),
					},
				}
				errorOutput.Duration = errorOutput.EndTime.Sub(errorOutput.StartTime).Seconds()
				yield(errorOutput, stepError)
				return
			}

			// 保存步骤输出
			if stepOutput != nil {
				allResults = append(allResults, stepOutput)
				routerStepOutputs[step.Name()] = stepOutput
			}

			// 检查上下文取消
			if ctx.Err() != nil {
				yield(nil, ctx.Err())
				return
			}
		}

		// 构建最终输出
		var finalContent interface{}
		if len(allResults) > 0 {
			finalContent = allResults[len(allResults)-1].Content
		}

		output := &StepOutput{
			StepID:      r.id,
			StepName:    r.name,
			StepType:    StepTypeRouter,
			Content:     finalContent,
			StartTime:   startTime,
			EndTime:     time.Now(),
			NestedSteps: allResults,
			Metadata: map[string]interface{}{
				"selected_steps": len(stepsToExecute),
				"executed_steps": len(allResults),
				"step_names":     getStepNames(stepsToExecute),
			},
			Metrics: &StepMetrics{ExecutionTime: time.Since(startTime).Seconds()},
		}
		output.Duration = output.EndTime.Sub(output.StartTime).Seconds()
		yield(output, nil)
	}
}

// WithDescription 设置描述
func (r *Router) WithDescription(desc string) *Router {
	r.description = desc
	return r
}

// WithTimeout 设置超时
func (r *Router) WithTimeout(timeout time.Duration) *Router {
	r.config.Timeout = timeout
	return r
}

// ExecuteStream 流式执行 Router - 支持实时事件流
func (r *Router) ExecuteStream(ctx context.Context, input *StepInput, streamEvents bool) iter.Seq2[interface{}, error] {
	return func(yield func(interface{}, error) bool) {
		startTime := time.Now()
		routerID := r.id

		// 调用选择器函数
		stepsToExecute := r.selector(input)

		// 发送 Router 开始事件
		if streamEvents {
			startEvent := &RouterEvent{
				Type:          RouterEventStarted,
				RouterName:    r.name,
				SelectedSteps: getStepNames(stepsToExecute),
				ExecutedSteps: 0,
				Timestamp:     startTime,
			}
			if !yield(startEvent, nil) {
				return
			}
		}

		if len(stepsToExecute) == 0 {
			// 没有选中任何步骤
			if streamEvents {
				completeEvent := &RouterEvent{
					Type:          RouterEventCompleted,
					RouterName:    r.name,
					SelectedSteps: []string{},
					ExecutedSteps: 0,
					Timestamp:     time.Now(),
				}
				yield(completeEvent, nil)
			}

			output := &StepOutput{
				StepID:      routerID,
				StepName:    r.name,
				StepType:    StepTypeRouter,
				Content:     fmt.Sprintf("Router %s: no steps selected", r.name),
				StartTime:   startTime,
				EndTime:     time.Now(),
				NestedSteps: []*StepOutput{},
				Metadata: map[string]interface{}{
					"selected_steps": 0,
				},
				Metrics: &StepMetrics{ExecutionTime: time.Since(startTime).Seconds()},
			}
			output.Duration = output.EndTime.Sub(output.StartTime).Seconds()
			yield(output, nil)
			return
		}

		// 收集所有执行结果
		allResults := make([]*StepOutput, 0, len(stepsToExecute))
		currentInput := input
		routerStepOutputs := make(map[string]*StepOutput)

		// 顺序执行选中的步骤（链接模式）
		for i, step := range stepsToExecute {
			// 更新输入：使用前一步的输出
			if i > 0 && len(allResults) > 0 {
				lastOutput := allResults[len(allResults)-1]
				currentInput = &StepInput{
					Input:               input.Input,
					PreviousStepContent: lastOutput.Content,
					PreviousStepOutputs: routerStepOutputs,
					AdditionalData:      input.AdditionalData,
					SessionState:        input.SessionState,
					Images:              input.Images,
					Videos:              input.Videos,
					Audio:               input.Audio,
					Files:               input.Files,
					WorkflowSession:     input.WorkflowSession,
				}
			}

			// 流式执行步骤
			var stepOutput *StepOutput
			var stepError error

			for output, err := range step.Execute(ctx, currentInput) {
				if err != nil {
					stepError = err
					break
				}

				// 转发步骤的流式输出
				if streamEvents && output != nil {
					stepProgressEvent := map[string]interface{}{
						"type":        "router_step_progress",
						"router_name": r.name,
						"step_name":   step.Name(),
						"step_index":  i,
						"output":      output,
					}
					if !yield(stepProgressEvent, nil) {
						return
					}
				}

				stepOutput = output
			}

			if stepError != nil {
				// 步骤执行失败
				if streamEvents {
					failEvent := &RouterEvent{
						Type:          RouterEventFailed,
						RouterName:    r.name,
						SelectedSteps: getStepNames(stepsToExecute),
						ExecutedSteps: i,
						Timestamp:     time.Now(),
					}
					yield(failEvent, nil)
				}

				errorOutput := &StepOutput{
					StepID:      routerID,
					StepName:    r.name,
					StepType:    StepTypeRouter,
					Error:       stepError,
					StartTime:   startTime,
					EndTime:     time.Now(),
					NestedSteps: allResults,
					Metadata: map[string]interface{}{
						"selected_steps": len(stepsToExecute),
						"executed_steps": i,
						"failed_step":    step.Name(),
						"error":          stepError.Error(),
					},
				}
				errorOutput.Duration = errorOutput.EndTime.Sub(errorOutput.StartTime).Seconds()
				yield(errorOutput, stepError)
				return
			}

			// 保存步骤输出
			if stepOutput != nil {
				allResults = append(allResults, stepOutput)
				routerStepOutputs[step.Name()] = stepOutput
			}

			// 检查上下文取消
			if ctx.Err() != nil {
				yield(nil, ctx.Err())
				return
			}
		}

		// 发送 Router 完成事件
		if streamEvents {
			completeEvent := &RouterEvent{
				Type:          RouterEventCompleted,
				RouterName:    r.name,
				SelectedSteps: getStepNames(stepsToExecute),
				ExecutedSteps: len(allResults),
				Timestamp:     time.Now(),
			}
			yield(completeEvent, nil)
		}

		// 构建最终输出
		var finalContent interface{}
		if len(allResults) > 0 {
			finalContent = allResults[len(allResults)-1].Content
		}

		output := &StepOutput{
			StepID:      routerID,
			StepName:    r.name,
			StepType:    StepTypeRouter,
			Content:     finalContent,
			StartTime:   startTime,
			EndTime:     time.Now(),
			NestedSteps: allResults,
			Metadata: map[string]interface{}{
				"selected_steps": len(stepsToExecute),
				"executed_steps": len(allResults),
				"step_names":     getStepNames(stepsToExecute),
			},
			Metrics: &StepMetrics{ExecutionTime: time.Since(startTime).Seconds()},
		}
		output.Duration = output.EndTime.Sub(output.StartTime).Seconds()
		yield(output, nil)
	}
}

// ===== 简化的 Router 构造函数 =====

// SimpleRouter 创建简单的条件路由器
// 根据条件选择单个步骤执行
func SimpleRouter(name string, condition func(*StepInput) string, routes map[string]Step) *Router {
	// 从 map 构建 choices
	choices := make([]Step, 0, len(routes))
	for _, step := range routes {
		choices = append(choices, step)
	}

	// 选择器函数：根据条件返回单个步骤
	selector := func(input *StepInput) []Step {
		routeName := condition(input)
		if step, exists := routes[routeName]; exists {
			return []Step{step}
		}
		return []Step{}
	}

	return NewRouter(name, selector, choices)
}

// ChainRouter 创建链式路由器
// 根据条件选择多个步骤顺序执行
func ChainRouter(name string, selector func(*StepInput) []string, routes map[string]Step) *Router {
	// 从 map 构建 choices
	choices := make([]Step, 0, len(routes))
	for _, step := range routes {
		choices = append(choices, step)
	}

	// 选择器函数：根据条件返回多个步骤
	selectorFunc := func(input *StepInput) []Step {
		stepNames := selector(input)
		steps := make([]Step, 0, len(stepNames))
		for _, name := range stepNames {
			if step, exists := routes[name]; exists {
				steps = append(steps, step)
			}
		}
		return steps
	}

	return NewRouter(name, selectorFunc, choices)
}

// DynamicRouter 创建动态路由器
// 完全自定义的步骤选择逻辑
func DynamicRouter(name string, selector func(*StepInput) []Step) *Router {
	return &Router{
		id:       uuid.New().String(),
		name:     name,
		selector: selector,
		choices:  []Step{}, // 动态路由不需要预定义 choices
		config: &StepConfig{
			Name:        name,
			Type:        StepTypeRouter,
			MaxRetries:  1,
			Timeout:     10 * time.Minute,
			SkipOnError: false,
		},
	}
}

// ===== 辅助函数 =====

func getStepNames(steps []Step) []string {
	names := make([]string, len(steps))
	for i, step := range steps {
		names[i] = step.Name()
	}
	return names
}

// ===== Router 事件（为将来的流式支持准备）=====

type RouterEvent struct {
	Type          string
	RouterName    string
	SelectedSteps []string
	ExecutedSteps int
	Timestamp     time.Time
}

const (
	RouterEventStarted   = "router_started"
	RouterEventCompleted = "router_completed"
	RouterEventFailed    = "router_failed"
)
