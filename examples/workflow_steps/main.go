// WorkflowSteps 演示 Aster Workflow 的所有步骤类型，包括 FunctionStep、
// ConditionStep、ParallelStep、LoopStep、RouterStep 等。
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
	fmt.Println("=== Aster Workflow 所有步骤类型测试 ===")

	ctx := context.Background()

	// 1. FunctionStep
	fmt.Println("1️⃣ FunctionStep")
	funcStep := workflow.NewFunctionStep("func", func(ctx context.Context, input *workflow.StepInput) (*workflow.StepOutput, error) {
		return &workflow.StepOutput{
			Content:  "Function executed",
			Metadata: make(map[string]any),
		}, nil
	})
	fmt.Printf("   ✅ Created: %s (type: %s)\n\n", funcStep.Name(), funcStep.Type())

	// 2. ConditionStep
	fmt.Println("2️⃣ ConditionStep")
	trueStep := workflow.NewFunctionStep("true", func(ctx context.Context, input *workflow.StepInput) (*workflow.StepOutput, error) {
		return &workflow.StepOutput{Content: "True branch", Metadata: make(map[string]any)}, nil
	})
	falseStep := workflow.NewFunctionStep("false", func(ctx context.Context, input *workflow.StepInput) (*workflow.StepOutput, error) {
		return &workflow.StepOutput{Content: "False branch", Metadata: make(map[string]any)}, nil
	})
	condStep := workflow.NewConditionStep("cond", func(input *workflow.StepInput) bool {
		return true
	}, trueStep, falseStep)
	fmt.Printf("   ✅ Created: %s (type: %s)\n\n", condStep.Name(), condStep.Type())

	// 3. LoopStep
	fmt.Println("3️⃣ LoopStep")
	loopBody := workflow.NewFunctionStep("body", func(ctx context.Context, input *workflow.StepInput) (*workflow.StepOutput, error) {
		return &workflow.StepOutput{Content: "Loop iteration", Metadata: make(map[string]any)}, nil
	})
	loopStep := workflow.NewLoopStep("loop", loopBody, 3)
	fmt.Printf("   ✅ Created: %s (type: %s, max: 3 iterations)\n\n", loopStep.Name(), loopStep.Type())

	// 4. ParallelStep
	fmt.Println("4️⃣ ParallelStep")
	task1 := workflow.NewFunctionStep("task1", func(ctx context.Context, input *workflow.StepInput) (*workflow.StepOutput, error) {
		time.Sleep(10 * time.Millisecond)
		return &workflow.StepOutput{Content: "Task 1", Metadata: make(map[string]any)}, nil
	})
	task2 := workflow.NewFunctionStep("task2", func(ctx context.Context, input *workflow.StepInput) (*workflow.StepOutput, error) {
		time.Sleep(10 * time.Millisecond)
		return &workflow.StepOutput{Content: "Task 2", Metadata: make(map[string]any)}, nil
	})
	parallelStep := workflow.NewParallelStep("parallel", task1, task2)
	fmt.Printf("   ✅ Created: %s (type: %s, tasks: 2)\n\n", parallelStep.Name(), parallelStep.Type())

	// 5. RouterStep
	fmt.Println("5️⃣ RouterStep")
	routeA := workflow.NewFunctionStep("route_a", func(ctx context.Context, input *workflow.StepInput) (*workflow.StepOutput, error) {
		return &workflow.StepOutput{Content: "Route A", Metadata: make(map[string]any)}, nil
	})
	routeB := workflow.NewFunctionStep("route_b", func(ctx context.Context, input *workflow.StepInput) (*workflow.StepOutput, error) {
		return &workflow.StepOutput{Content: "Route B", Metadata: make(map[string]any)}, nil
	})
	routerStep := workflow.NewRouterStep("router", func(input *workflow.StepInput) string {
		return "route_a"
	}, map[string]workflow.Step{
		"route_a": routeA,
		"route_b": routeB,
	})
	fmt.Printf("   ✅ Created: %s (type: %s, routes: 2)\n\n", routerStep.Name(), routerStep.Type())

	// 6. StepsGroup
	fmt.Println("6️⃣ StepsGroup")
	step1 := workflow.NewFunctionStep("step1", func(ctx context.Context, input *workflow.StepInput) (*workflow.StepOutput, error) {
		return &workflow.StepOutput{Content: "Step 1", Metadata: make(map[string]any)}, nil
	})
	step2 := workflow.NewFunctionStep("step2", func(ctx context.Context, input *workflow.StepInput) (*workflow.StepOutput, error) {
		return &workflow.StepOutput{Content: "Step 2", Metadata: make(map[string]any)}, nil
	})
	stepsGroup := workflow.NewStepsGroup("group", step1, step2)
	fmt.Printf("   ✅ Created: %s (type: %s, steps: 2)\n\n", stepsGroup.Name(), stepsGroup.Type())

	// 执行测试
	fmt.Println("=== 执行测试 ===")

	// 测试 ConditionStep
	fmt.Println("Testing ConditionStep...")
	input := &workflow.StepInput{Input: "test"}
	condReader := condStep.Execute(ctx, input)
	for {
		output, err := condReader.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			fmt.Printf("❌ Error: %v\n", err)
		} else {
			fmt.Printf("✅ Result: %v\n\n", output.Content)
		}
	}

	// 测试 LoopStep
	fmt.Println("Testing LoopStep...")
	loopReader := loopStep.Execute(ctx, input)
	for {
		output, err := loopReader.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			fmt.Printf("❌ Error: %v\n", err)
		} else {
			fmt.Printf("✅ Completed %d iterations\n\n", len(output.NestedSteps))
		}
	}

	// 测试 ParallelStep
	fmt.Println("Testing ParallelStep...")
	start := time.Now()
	parallelReader := parallelStep.Execute(ctx, input)
	for {
		output, err := parallelReader.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			fmt.Printf("❌ Error: %v\n", err)
		} else {
			duration := time.Since(start)
			fmt.Printf("✅ Completed %d tasks in %v\n\n", len(output.NestedSteps), duration)
		}
	}

	// 测试 RouterStep
	fmt.Println("Testing RouterStep...")
	routerReader := routerStep.Execute(ctx, input)
	for {
		output, err := routerReader.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			fmt.Printf("❌ Error: %v\n", err)
		} else {
			fmt.Printf("✅ Result: %v\n\n", output.Content)
		}
	}

	fmt.Println("=== 所有步骤类型测试完成 ===")
}
