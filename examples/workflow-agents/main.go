// WorkflowAgents 演示工作流 Agent 的使用，包括 SequentialAgent 顺序工作流、
// ParallelAgent 并行工作流和 LoopAgent 循环工作流等模式。
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/astercloud/aster/pkg/agent/workflow"
	"github.com/astercloud/aster/pkg/session"
	"github.com/astercloud/aster/pkg/stream"
	"github.com/astercloud/aster/pkg/types"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== 工作流 Agent 演示 ===")

	// ====== 示例 1: SequentialAgent - 顺序工作流 ======
	fmt.Println("📝 示例 1: SequentialAgent - 多步骤流水线")
	sequentialExample(ctx)

	// ====== 示例 2: ParallelAgent - 并行执行 ======
	fmt.Println("\n⚡ 示例 2: ParallelAgent - 并行比较方案")
	parallelExample(ctx)

	// ====== 示例 3: LoopAgent - 循环优化 ======
	fmt.Println("\n🔄 示例 3: LoopAgent - 迭代优化")
	loopExample(ctx)

	// ====== 示例 4: 嵌套工作流 ======
	fmt.Println("\n🌳 示例 4: 嵌套工作流 - Sequential + Parallel")
	nestedExample(ctx)

	fmt.Println("\n✅ 所有示例完成！")
}

// sequentialExample 顺序工作流示例
func sequentialExample(ctx context.Context) {
	// 创建子 Agent
	agents := []workflow.Agent{
		NewMockAgent("DataCollector", "收集数据"),
		NewMockAgent("Analyzer", "分析数据"),
		NewMockAgent("Reporter", "生成报告"),
	}

	// 创建 SequentialAgent
	sequential, err := workflow.NewSequentialAgent(workflow.SequentialConfig{
		Name:      "DataPipeline",
		SubAgents: agents,
	})
	if err != nil {
		log.Fatal(err)
	}

	// 执行
	fmt.Println("开始顺序执行:")
	reader := sequential.Execute(ctx, "处理用户数据")
	for {
		event, err := reader.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			log.Printf("错误: %v", err)
			break
		}
		printEvent(event)
	}
}

// parallelExample 并行执行示例
func parallelExample(ctx context.Context) {
	// 创建多个算法 Agent
	agents := []workflow.Agent{
		NewMockAgent("AlgorithmA", "方案A：快速但粗糙"),
		NewMockAgent("AlgorithmB", "方案B：慢但精确"),
		NewMockAgent("AlgorithmC", "方案C：平衡"),
	}

	// 创建 ParallelAgent
	parallel, err := workflow.NewParallelAgent(workflow.ParallelConfig{
		Name:      "MultiAlgorithm",
		SubAgents: agents,
	})
	if err != nil {
		log.Fatal(err)
	}

	// 执行
	fmt.Println("开始并行执行:")
	resultCount := 0
	reader := parallel.Execute(ctx, "求解问题")
	for {
		event, err := reader.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			log.Printf("错误: %v", err)
			continue
		}
		resultCount++
		printEvent(event)
	}
	fmt.Printf("收到 %d 个并行结果\n", resultCount)
}

// loopExample 循环优化示例
func loopExample(ctx context.Context) {
	// 创建优化流程的子 Agent
	agents := []workflow.Agent{
		NewMockAgent("Critic", "评估当前方案"),
		NewMockAgent("Improver", "提出改进建议"),
	}

	// 创建 LoopAgent（最多 3 次迭代）
	loop, err := workflow.NewLoopAgent(workflow.LoopConfig{
		Name:          "OptimizationLoop",
		SubAgents:     agents,
		MaxIterations: 3,
		StopCondition: func(event *session.Event) bool {
			// 如果评分达到 90 分，提前停止
			if score, ok := event.Metadata["quality_score"].(int); ok {
				return score >= 90
			}
			return false
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// 执行
	fmt.Println("开始循环优化:")
	iteration := 0
	reader := loop.Execute(ctx, "优化代码质量")
	for {
		event, err := reader.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			log.Printf("错误: %v", err)
			break
		}

		// 追踪迭代次数
		if iterNum, ok := event.Metadata["loop_iteration"].(uint); ok {
			if uint(iteration) != iterNum {
				iteration = int(iterNum)
				fmt.Printf("\n--- 迭代 %d ---\n", iteration)
			}
		}

		printEvent(event)
	}
}

// nestedExample 嵌套工作流示例
func nestedExample(ctx context.Context) {
	// 第一步：并行收集多个数据源
	dataCollectors := []workflow.Agent{
		NewMockAgent("Source1", "数据源1"),
		NewMockAgent("Source2", "数据源2"),
		NewMockAgent("Source3", "数据源3"),
	}
	parallelCollector, _ := workflow.NewParallelAgent(workflow.ParallelConfig{
		Name:      "ParallelCollector",
		SubAgents: dataCollectors,
	})

	// 第二步：分析
	analyzer := NewMockAgent("Analyzer", "数据分析")

	// 第三步：报告
	reporter := NewMockAgent("Reporter", "生成报告")

	// 组合成顺序工作流
	sequential, err := workflow.NewSequentialAgent(workflow.SequentialConfig{
		Name: "NestedWorkflow",
		SubAgents: []workflow.Agent{
			parallelCollector, // 并行收集
			analyzer,          // 分析
			reporter,          // 报告
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// 执行
	fmt.Println("开始嵌套工作流:")
	reader := sequential.Execute(ctx, "综合数据分析")
	for {
		event, err := reader.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			log.Printf("错误: %v", err)
			break
		}
		printEvent(event)
	}
}

// ============================================================
// Mock Agent 实现（用于演示）
// ============================================================

type MockAgent struct {
	name        string
	description string
}

func NewMockAgent(name, description string) *MockAgent {
	return &MockAgent{
		name:        name,
		description: description,
	}
}

func (a *MockAgent) Name() string {
	return a.name
}

func (a *MockAgent) Execute(ctx context.Context, message string) *stream.Reader[*session.Event] {
	reader, writer := stream.Pipe[*session.Event](10)

	go func() {
		defer writer.Close()

		// 模拟处理时间
		time.Sleep(100 * time.Millisecond)

		event := &session.Event{
			ID:           fmt.Sprintf("evt-%s-%d", a.name, time.Now().UnixNano()),
			Timestamp:    time.Now(),
			InvocationID: "demo-invocation",
			AgentID:      a.name,
			Author:       "agent",
			Content: types.Message{
				Role:    types.RoleAssistant,
				Content: fmt.Sprintf("[%s] %s - 处理: %s", a.name, a.description, message),
			},
			Metadata: map[string]any{
				"agent_description": a.description,
				"quality_score":     85 + (time.Now().Unix() % 10), // 模拟质量分数
			},
		}

		writer.Send(event, nil)
	}()

	return reader
}

// ============================================================
// 辅助函数
// ============================================================

func printEvent(event *session.Event) {
	if event == nil {
		return
	}

	// 打印基本信息
	fmt.Printf("  ✓ [%s] %s\n", event.AgentID, event.Content.Content)

	// 打印元数据
	if branch := event.Branch; branch != "" {
		fmt.Printf("    Branch: %s\n", branch)
	}

	// 打印特殊元数据
	if index, ok := event.Metadata["parallel_index"].(int); ok {
		fmt.Printf("    并行索引: %d\n", index)
	}

	if step, ok := event.Metadata["sequential_step"].(int); ok {
		total := event.Metadata["total_steps"].(int)
		fmt.Printf("    步骤: %d/%d\n", step, total)
	}

	if iter, ok := event.Metadata["loop_iteration"].(uint); ok {
		fmt.Printf("    迭代: %d\n", iter)
	}

	if score, ok := event.Metadata["quality_score"].(int); ok {
		fmt.Printf("    质量分数: %d/100\n", score)
	}
}
