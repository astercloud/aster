// Workflow 演示 Aster 工作流引擎，包括顺序步骤执行、步骤间数据传递和
// 基于流的事件处理。
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/astercloud/aster/pkg/workflow"
)

func main() {
	fmt.Println("=== Aster Workflow 示例 ===")

	ctx := context.Background()

	// 创建 Workflow
	wf := workflow.New("DataProcessing").
		WithStream().
		WithDebug()

	// 步骤 1: 加载数据
	wf.AddStep(workflow.NewFunctionStep("load", func(ctx context.Context, input *workflow.StepInput) (*workflow.StepOutput, error) {
		fmt.Println("📥 Step 1: Loading data...")
		return &workflow.StepOutput{
			Content: map[string]any{
				"data":  []string{"item1", "item2", "item3"},
				"count": 3,
			},
			Metadata: make(map[string]any),
		}, nil
	}))

	// 步骤 2: 处理数据
	wf.AddStep(workflow.NewFunctionStep("process", func(ctx context.Context, input *workflow.StepInput) (*workflow.StepOutput, error) {
		fmt.Println("⚙️  Step 2: Processing data...")

		// 从前一步获取数据
		var dataMap map[string]any
		if input.PreviousStepContent != nil {
			dataMap, _ = input.PreviousStepContent.(map[string]any)
		}

		if dataMap != nil {
			if data, ok := dataMap["data"].([]string); ok {
				processed := make([]string, len(data))
				for i, item := range data {
					processed[i] = fmt.Sprintf("processed_%s", item)
				}
				return &workflow.StepOutput{
					Content: map[string]any{
						"processed": processed,
						"count":     len(processed),
					},
					Metadata: make(map[string]any),
				}, nil
			}
		}

		return nil, fmt.Errorf("invalid input: expected map with 'data' field")
	}))

	// 步骤 3: 转换数据
	wf.AddStep(workflow.NewFunctionStep("transform", func(ctx context.Context, input *workflow.StepInput) (*workflow.StepOutput, error) {
		fmt.Println("🔄 Step 3: Transforming data...")

		result := "No data"
		if input.PreviousStepContent != nil {
			if dataMap, ok := input.PreviousStepContent.(map[string]any); ok {
				if processed, ok := dataMap["processed"].([]string); ok {
					result = fmt.Sprintf("✅ Final Result: %v", processed)
				}
			}
		}

		return &workflow.StepOutput{
			Content:  result,
			Metadata: make(map[string]any),
		}, nil
	}))

	// 验证
	if err := wf.Validate(); err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	fmt.Println("\n=== 执行 Workflow ===")

	// 执行
	input := &workflow.WorkflowInput{
		Input: "start",
	}

	eventCount := 0
	reader := wf.Execute(ctx, input)
	for {
		event, err := reader.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			log.Printf("❌ Error: %v", err)
			continue
		}

		eventCount++
		fmt.Printf("\n[Event %d] %s\n", eventCount, event.Type)

		switch event.Type {
		case workflow.EventWorkflowStarted:
			fmt.Println("  ▶ Workflow started")

		case workflow.EventStepStarted:
			fmt.Printf("  ▶ Step: %s\n", event.StepName)

		case workflow.EventStepCompleted:
			fmt.Printf("  ✓ Step completed: %s\n", event.StepName)
			if data, ok := event.Data.(map[string]any); ok {
				if output, ok := data["output"].(*workflow.StepOutput); ok {
					fmt.Printf("    Output: %v\n", output.Content)
					if output.Metrics != nil {
						fmt.Printf("    Time: %.3fs\n", output.Metrics.ExecutionTime)
					}
				}
			}

		case workflow.EventWorkflowCompleted:
			fmt.Println("  ✓ Workflow completed")
			if data, ok := event.Data.(map[string]any); ok {
				if output, ok := data["output"]; ok {
					fmt.Printf("    Final output: %v\n", output)
				}
				if metrics, ok := data["metrics"].(*workflow.RunMetrics); ok {
					fmt.Printf("    Total time: %.3fs\n", metrics.TotalExecutionTime)
					fmt.Printf("    Steps: %d total, %d succeeded\n",
						metrics.TotalSteps, metrics.SuccessfulSteps)
				}
			}
		}
	}

	fmt.Printf("\n=== 完成 ===\n共处理 %d 个事件\n", eventCount)
}
