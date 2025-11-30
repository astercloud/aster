// SubagentAsync 演示 SubAgent 的异步执行模式，支持并行委派任务给多个
// 专业化子代理并收集结果。
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/astercloud/aster/pkg/backends"
	"github.com/astercloud/aster/pkg/middleware"
	"github.com/astercloud/aster/pkg/tools"
)

func main() {
	fmt.Println("=== SubAgent 异步执行示例 ===")
	fmt.Println()

	// 1. 创建 Backend
	backend := backends.NewStateBackend()
	fmt.Println("✓ 创建 StateBackend")

	// 2. 定义子代理规格
	specs := []middleware.SubAgentSpec{
		{
			Name:        "researcher",
			Description: "深度研究和分析专家",
			Prompt:      "你是一个专注于深度研究的 AI。仔细分析问题,提供详细的研究报告。",
		},
		{
			Name:        "data-analyst",
			Description: "数据分析专家",
			Prompt:      "你是一个数据分析专家。处理和分析大量数据,生成洞察报告。",
		},
	}
	fmt.Printf("✓ 定义了 %d 个子代理规格\n", len(specs))

	// 3. 创建子代理工厂（模拟长时间运行的任务）
	factory := func(ctx context.Context, spec middleware.SubAgentSpec) (middleware.SubAgent, error) {
		execFn := func(ctx context.Context, description string, parentContext map[string]any) (string, error) {
			// 模拟长时间运行的任务
			fmt.Printf("[%s] 开始执行任务: %s\n", spec.Name, description)
			time.Sleep(5 * time.Second) // 模拟5秒的处理时间

			result := fmt.Sprintf(`[%s] 任务完成报告
任务描述: %s
执行时间: 5秒
系统提示: %s
上下文: %v

分析结果:
1. 任务已成功完成
2. 处理了大量数据
3. 生成了详细报告`, spec.Name, description, spec.Prompt, parentContext)

			fmt.Printf("[%s] 任务完成\n", spec.Name)
			return result, nil
		}
		return middleware.NewSimpleSubAgent(spec.Name, spec.Prompt, execFn), nil
	}
	fmt.Println("✓ 创建子代理工厂")

	// 4. 创建 SubAgentMiddleware（启用异步执行）
	subagentMiddleware, err := middleware.NewSubAgentMiddleware(&middleware.SubAgentMiddlewareConfig{
		Specs:                  specs,
		Factory:                factory,
		EnableParallel:         true,
		EnableAsync:            true,  // 启用异步执行
		EnableProcessIsolation: false, // 使用 goroutine 模式（演示用）
		DefaultTimeout:         30 * time.Second,
	})
	if err != nil {
		log.Fatalf("创建 SubAgentMiddleware 失败: %v", err)
	}
	fmt.Println("✓ 创建 SubAgentMiddleware（异步模式）")

	// 5. 创建 FilesystemMiddleware
	fsMiddleware := middleware.NewFilesystemMiddleware(&middleware.FilesystemMiddlewareConfig{
		Backend:        backend,
		EnableEviction: true,
	})
	fmt.Println("✓ 创建 FilesystemMiddleware")

	// 6. 创建 Middleware Stack
	stack := middleware.NewStack([]middleware.Middleware{
		fsMiddleware,
		subagentMiddleware,
	})
	fmt.Printf("✓ 创建 Middleware Stack\n\n")

	// 7. 获取工具
	allTools := stack.Tools()
	toolMap := make(map[string]tools.Tool)
	for _, tool := range allTools {
		toolMap[tool.Name()] = tool
	}

	fmt.Println("=== 可用工具 ===")
	for name := range toolMap {
		fmt.Printf("- %s\n", name)
	}
	fmt.Println()

	// 8. 演示异步执行
	fmt.Println("=== 演示 1: 异步启动多个任务 ===")
	fmt.Println()

	ctx := context.Background()

	// 启动任务 1
	fmt.Println("启动任务 1: 研究 AI Agent 架构...")
	result1, err := toolMap["task"].Execute(ctx, map[string]any{
		"description":   "深度研究 AI Agent 的架构设计模式，包括事件驱动、中间件、工具系统等",
		"subagent_type": "researcher",
		"async":         true, // 异步执行
		"timeout":       30,
	}, nil)
	if err != nil {
		log.Fatalf("启动任务 1 失败: %v", err)
	}
	task1Result := result1.(map[string]any)
	task1ID := task1Result["task_id"].(string)
	fmt.Printf("✓ 任务 1 已启动，task_id: %s\n", task1ID)
	fmt.Printf("  状态: %s\n", task1Result["status"])
	fmt.Printf("  消息: %s\n\n", task1Result["message"])

	// 启动任务 2
	fmt.Println("启动任务 2: 分析大规模数据集...")
	result2, err := toolMap["task"].Execute(ctx, map[string]any{
		"description":   "分析用户行为数据，识别模式和趋势，生成可视化报告",
		"subagent_type": "data-analyst",
		"async":         true,
		"timeout":       30,
	}, nil)
	if err != nil {
		log.Fatalf("启动任务 2 失败: %v", err)
	}
	task2Result := result2.(map[string]any)
	task2ID := task2Result["task_id"].(string)
	fmt.Printf("✓ 任务 2 已启动，task_id: %s\n", task2ID)
	fmt.Printf("  状态: %s\n\n", task2Result["status"])

	// 9. 列出所有任务
	fmt.Println("=== 演示 2: 列出所有任务 ===")
	fmt.Println()
	listResult, _ := toolMap["list_subagents"].Execute(ctx, map[string]any{}, nil)
	listData := listResult.(map[string]any)
	fmt.Printf("当前任务数: %v\n", listData["count"])
	if subagents, ok := listData["subagents"].([]map[string]any); ok {
		for _, sa := range subagents {
			fmt.Printf("- task_id: %s, type: %s, status: %s\n",
				sa["task_id"], sa["subagent_type"], sa["status"])
		}
	}
	fmt.Println()

	// 10. 轮询检查任务状态
	fmt.Println("=== 演示 3: 轮询任务状态 ===")
	fmt.Println()

	checkTask := func(taskID string, taskName string) {
		for range 10 {
			queryResult, _ := toolMap["query_subagent"].Execute(ctx, map[string]any{
				"task_id": taskID,
			}, nil)
			queryData := queryResult.(map[string]any)

			status := queryData["status"].(string)
			duration := queryData["duration"].(float64)

			fmt.Printf("[%s] 状态: %s, 运行时间: %.1f秒\n", taskName, status, duration)

			if status == "completed" {
				fmt.Printf("[%s] ✓ 任务完成！\n", taskName)
				if output, ok := queryData["output"].(string); ok {
					fmt.Printf("输出:\n%s\n\n", output)
				}
				break
			} else if status == "failed" {
				fmt.Printf("[%s] ✗ 任务失败: %v\n\n", taskName, queryData["error"])
				break
			}

			time.Sleep(1 * time.Second)
		}
	}

	// 并行检查两个任务
	done := make(chan bool, 2)

	go func() {
		checkTask(task1ID, "任务1")
		done <- true
	}()

	go func() {
		checkTask(task2ID, "任务2")
		done <- true
	}()

	// 等待两个任务完成
	<-done
	<-done

	// 11. 演示停止和恢复
	fmt.Println("=== 演示 4: 停止和恢复任务 ===")
	fmt.Println()

	// 启动一个新任务
	fmt.Println("启动任务 3: 长时间运行的分析...")
	result3, _ := toolMap["task"].Execute(ctx, map[string]any{
		"description":   "执行复杂的数据挖掘任务",
		"subagent_type": "data-analyst",
		"async":         true,
		"timeout":       60,
	}, nil)
	task3Result := result3.(map[string]any)
	task3ID := task3Result["task_id"].(string)
	fmt.Printf("✓ 任务 3 已启动，task_id: %s\n\n", task3ID)

	// 等待一会儿
	time.Sleep(2 * time.Second)

	// 停止任务
	fmt.Println("停止任务 3...")
	stopResult, _ := toolMap["stop_subagent"].Execute(ctx, map[string]any{
		"task_id": task3ID,
	}, nil)
	stopData := stopResult.(map[string]any)
	fmt.Printf("✓ %s\n\n", stopData["message"])

	// 查询状态
	queryResult, _ := toolMap["query_subagent"].Execute(ctx, map[string]any{
		"task_id": task3ID,
	}, nil)
	queryData := queryResult.(map[string]any)
	fmt.Printf("任务状态: %s\n\n", queryData["status"])

	// 恢复任务
	fmt.Println("恢复任务 3...")
	resumeResult, _ := toolMap["resume_subagent"].Execute(ctx, map[string]any{
		"task_id": task3ID,
	}, nil)
	resumeData := resumeResult.(map[string]any)
	fmt.Printf("✓ %s\n", resumeData["message"])
	fmt.Printf("新的 task_id: %s\n\n", resumeData["new_task_id"])

	// 12. 最终列出所有任务
	fmt.Println("=== 最终任务列表 ===")
	fmt.Println()
	finalListResult, _ := toolMap["list_subagents"].Execute(ctx, map[string]any{}, nil)
	finalListData := finalListResult.(map[string]any)
	fmt.Printf("总任务数: %v\n", finalListData["count"])
	if subagents, ok := finalListData["subagents"].([]map[string]any); ok {
		for _, sa := range subagents {
			fmt.Printf("- task_id: %s, type: %s, status: %s, duration: %.1fs\n",
				sa["task_id"], sa["subagent_type"], sa["status"], sa["duration"])
		}
	}
	fmt.Println()

	// 13. 清理
	fmt.Println("=== 清理资源 ===")
	if err := stack.OnAgentStop(ctx, "demo-agent"); err != nil {
		fmt.Printf("清理失败: %v\n", err)
	} else {
		fmt.Println("✓ 所有中间件已清理")
	}

	fmt.Println("\n=== 示例完成 ===")
	fmt.Println("\n关键特性演示:")
	fmt.Println("✓ 异步启动多个子代理任务")
	fmt.Println("✓ 轮询查询任务状态和结果")
	fmt.Println("✓ 列出所有任务")
	fmt.Println("✓ 停止正在运行的任务")
	fmt.Println("✓ 恢复已停止的任务")
	fmt.Println("✓ 资源监控和管理")
}
