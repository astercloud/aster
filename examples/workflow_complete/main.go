// WorkflowComplete 演示 Aster Workflow 的完整功能测试，包括基础工作流、
// 所有步骤类型、嵌套工作流和错误处理等场景。
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/astercloud/aster/pkg/workflow"
)

func main() {
	fmt.Println("=== Aster Workflow 完整功能测试 ===")

	ctx := context.Background()

	// ===== 测试 1: 基础 Workflow =====
	fmt.Println("📝 测试 1: 基础 Workflow")
	testBasicWorkflow(ctx)

	// ===== 测试 2: 所有步骤类型 =====
	fmt.Println("\n📝 测试 2: 所有步骤类型")
	testAllStepTypes(ctx)

	// ===== 测试 3: Router 路由 =====
	fmt.Println("\n📝 测试 3: Router 路由")
	testRouter(ctx)

	// ===== 测试 4: WorkflowAgent =====
	fmt.Println("\n📝 测试 4: WorkflowAgent")
	testWorkflowAgent(ctx)

	fmt.Println("\n🎉 所有测试完成！")
}

// 测试 1: 基础 Workflow
func testBasicWorkflow(ctx context.Context) {
	wf := workflow.New("BasicTest").WithStream()

	wf.AddStep(workflow.NewFunctionStep("step1", func(ctx context.Context, input *workflow.StepInput) (*workflow.StepOutput, error) {
		return &workflow.StepOutput{
			Content:  "Step 1 完成",
			Metadata: make(map[string]any),
		}, nil
	}))

	wf.AddStep(workflow.NewFunctionStep("step2", func(ctx context.Context, input *workflow.StepInput) (*workflow.StepOutput, error) {
		return &workflow.StepOutput{
			Content:  fmt.Sprintf("Step 2 接收: %v", input.PreviousStepContent),
			Metadata: make(map[string]any),
		}, nil
	}))

	if err := wf.Validate(); err != nil {
		fmt.Printf("  ❌ 验证失败: %v\n", err)
		return
	}

	input := &workflow.WorkflowInput{Input: "测试输入"}
	reader := wf.Execute(ctx, input)
	for {
		event, err := reader.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			fmt.Printf("  ❌ 错误: %v\n", err)
			continue
		}
		if event.Type == workflow.EventWorkflowCompleted {
			fmt.Printf("  ✅ 成功: %v\n", event.Data.(map[string]any)["output"])
		}
	}
}

// 测试 2: 所有步骤类型
func testAllStepTypes(ctx context.Context) {
	wf := workflow.New("AllSteps").WithStream()

	// FunctionStep
	wf.AddStep(workflow.NewFunctionStep("function", func(ctx context.Context, input *workflow.StepInput) (*workflow.StepOutput, error) {
		return &workflow.StepOutput{
			Content:  map[string]any{"type": "function", "value": 1},
			Metadata: make(map[string]any),
		}, nil
	}))

	// ConditionStep
	trueStep := workflow.NewFunctionStep("true", func(ctx context.Context, input *workflow.StepInput) (*workflow.StepOutput, error) {
		return &workflow.StepOutput{Content: "条件为真", Metadata: make(map[string]any)}, nil
	})
	falseStep := workflow.NewFunctionStep("false", func(ctx context.Context, input *workflow.StepInput) (*workflow.StepOutput, error) {
		return &workflow.StepOutput{Content: "条件为假", Metadata: make(map[string]any)}, nil
	})
	condStep := workflow.NewConditionStep("condition", func(input *workflow.StepInput) bool {
		if m, ok := input.PreviousStepContent.(map[string]any); ok {
			if v, ok := m["value"].(int); ok {
				return v > 0
			}
		}
		return false
	}, trueStep, falseStep)
	wf.AddStep(condStep)

	// LoopStep
	loopBody := workflow.NewFunctionStep("loop_body", func(ctx context.Context, input *workflow.StepInput) (*workflow.StepOutput, error) {
		return &workflow.StepOutput{Content: "循环迭代", Metadata: make(map[string]any)}, nil
	})
	wf.AddStep(workflow.NewLoopStep("loop", loopBody, 2))

	// ParallelStep
	task1 := workflow.NewFunctionStep("task1", func(ctx context.Context, input *workflow.StepInput) (*workflow.StepOutput, error) {
		time.Sleep(5 * time.Millisecond)
		return &workflow.StepOutput{Content: "任务1", Metadata: make(map[string]any)}, nil
	})
	task2 := workflow.NewFunctionStep("task2", func(ctx context.Context, input *workflow.StepInput) (*workflow.StepOutput, error) {
		time.Sleep(5 * time.Millisecond)
		return &workflow.StepOutput{Content: "任务2", Metadata: make(map[string]any)}, nil
	})
	wf.AddStep(workflow.NewParallelStep("parallel", task1, task2))

	// StepsGroup
	groupStep1 := workflow.NewFunctionStep("g1", func(ctx context.Context, input *workflow.StepInput) (*workflow.StepOutput, error) {
		return &workflow.StepOutput{Content: "组步骤1", Metadata: make(map[string]any)}, nil
	})
	groupStep2 := workflow.NewFunctionStep("g2", func(ctx context.Context, input *workflow.StepInput) (*workflow.StepOutput, error) {
		return &workflow.StepOutput{Content: "组步骤2", Metadata: make(map[string]any)}, nil
	})
	wf.AddStep(workflow.NewStepsGroup("group", groupStep1, groupStep2))

	input := &workflow.WorkflowInput{Input: "测试所有类型"}
	stepCount := 0
	reader := wf.Execute(ctx, input)
	for {
		event, err := reader.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			fmt.Printf("  ❌ 错误: %v\n", err)
			continue
		}
		if event.Type == workflow.EventStepCompleted {
			stepCount++
		}
		if event.Type == workflow.EventWorkflowCompleted {
			if data, ok := event.Data.(map[string]any); ok {
				if metrics, ok := data["metrics"].(*workflow.RunMetrics); ok {
					fmt.Printf("  ✅ 成功: %d 步骤完成, 耗时 %.3fs\n",
						metrics.SuccessfulSteps, metrics.TotalExecutionTime)
				}
			}
		}
	}
}

// 测试 3: Router 路由
func testRouter(ctx context.Context) {
	// 创建路由目标步骤
	routeA := workflow.NewFunctionStep("route_a", func(ctx context.Context, input *workflow.StepInput) (*workflow.StepOutput, error) {
		return &workflow.StepOutput{Content: "路由A执行", Metadata: make(map[string]any)}, nil
	})

	routeB := workflow.NewFunctionStep("route_b", func(ctx context.Context, input *workflow.StepInput) (*workflow.StepOutput, error) {
		return &workflow.StepOutput{Content: "路由B执行", Metadata: make(map[string]any)}, nil
	})

	finalStep := workflow.NewFunctionStep("final", func(ctx context.Context, input *workflow.StepInput) (*workflow.StepOutput, error) {
		return &workflow.StepOutput{
			Content:  fmt.Sprintf("最终结果: %v", input.PreviousStepContent),
			Metadata: make(map[string]any),
		}, nil
	})

	// 测试 SimpleRouter
	fmt.Println("  测试 SimpleRouter:")
	simpleRouter := workflow.SimpleRouter("simple_router",
		func(input *workflow.StepInput) string {
			if inputStr, ok := input.Input.(string); ok {
				if len(inputStr) > 10 {
					return "route_a"
				}
			}
			return "route_b"
		},
		map[string]workflow.Step{
			"route_a": routeA,
			"route_b": routeB,
		},
	)

	wf1 := workflow.New("SimpleRouterTest").AddStep(simpleRouter)
	input1 := &workflow.WorkflowInput{Input: "short"}

	reader1 := wf1.Execute(ctx, input1)
	for {
		event, err := reader1.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			fmt.Printf("    ❌ 错误: %v\n", err)
			continue
		}
		if event.Type == workflow.EventWorkflowCompleted {
			fmt.Printf("    ✅ SimpleRouter 完成: %v\n",
				event.Data.(map[string]any)["output"])
		}
	}

	// 测试 ChainRouter
	fmt.Println("  测试 ChainRouter:")
	chainRouter := workflow.ChainRouter("chain_router",
		func(input *workflow.StepInput) []string {
			return []string{"route_a", "final"}
		},
		map[string]workflow.Step{
			"route_a": routeA,
			"route_b": routeB,
			"final":   finalStep,
		},
	)

	wf2 := workflow.New("ChainRouterTest").AddStep(chainRouter)
	input2 := &workflow.WorkflowInput{Input: "test"}

	reader2 := wf2.Execute(ctx, input2)
	for {
		event, err := reader2.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			fmt.Printf("    ❌ 错误: %v\n", err)
			continue
		}
		if event.Type == workflow.EventWorkflowCompleted {
			fmt.Printf("    ✅ ChainRouter 完成: %v\n",
				event.Data.(map[string]any)["output"])
		}
	}
}

// 测试 4: WorkflowAgent
func testWorkflowAgent(ctx context.Context) {
	// 创建一个简单的 workflow
	wf := workflow.New("AgentWorkflow")

	wf.AddStep(workflow.NewFunctionStep("process", func(ctx context.Context, input *workflow.StepInput) (*workflow.StepOutput, error) {
		return &workflow.StepOutput{
			Content:  fmt.Sprintf("处理完成: %v", input.Input),
			Metadata: make(map[string]any),
		}, nil
	}))

	// 创建 WorkflowAgent
	agent := workflow.NewWorkflowAgent("gpt-4", "", true, 5)
	agent.AttachWorkflow(wf)

	fmt.Println("  测试同步执行:")
	result, err := agent.Run(ctx, "测试查询")
	if err != nil {
		fmt.Printf("    ❌ 错误: %v\n", err)
	} else {
		fmt.Printf("    ✅ 结果: %v\n", result)
	}

	// 测试流式执行
	fmt.Println("  测试流式执行:")
	eventCount := 0
	for event := range agent.RunStream(ctx, "流式测试") {
		eventCount++
		if event.Type == workflow.AgentEventComplete {
			fmt.Printf("    ✅ 流式完成, 收到 %d 个事件\n", eventCount)
		}
		if event.Error != nil {
			fmt.Printf("    ❌ 错误: %v\n", event.Error)
		}
	}

	// 测试 AgenticExecute
	fmt.Println("  测试 AgenticExecute:")
	result2, err := wf.AgenticExecute(ctx, agent, "Agentic 查询")
	if err != nil {
		fmt.Printf("    ❌ 错误: %v\n", err)
	} else {
		fmt.Printf("    ✅ 结果: %v\n", result2)
	}
}
