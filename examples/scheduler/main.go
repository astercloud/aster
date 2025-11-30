// Scheduler 演示任务调度器功能，包括步骤触发、时间间隔触发、步骤监听器、
// 多任务调度、任务取消和触发回调监控。
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/astercloud/aster/pkg/core"
)

func main() {
	fmt.Println("=== Agent SDK - Scheduler 示例 ===")

	// 示例 1: 步骤触发
	demonstrateStepTrigger()

	// 示例 2: 时间间隔触发
	demonstrateIntervalTrigger()

	// 示例 3: 步骤监听器
	demonstrateStepListeners()

	// 示例 4: 多任务调度
	demonstrateMultipleTasks()

	// 示例 5: 任务取消
	demonstrateTaskCancellation()

	// 示例 6: 触发回调监控
	demonstrateTriggerMonitoring()

	fmt.Println("\n=== 所有示例完成 ===")
}

// 示例 1: 步骤触发
func demonstrateStepTrigger() {
	fmt.Println("--- 示例 1: 步骤触发 ---")

	scheduler := core.NewScheduler(nil)
	defer scheduler.Shutdown()

	// 每 3 步执行一次
	taskID, err := scheduler.EverySteps(3, func(ctx context.Context, stepCount int) error {
		fmt.Printf("✓ 步骤任务触发: 第 %d 步\n", stepCount)
		return nil
	})

	if err != nil {
		log.Fatalf("创建步骤任务失败: %v", err)
	}

	fmt.Printf("步骤任务已创建: %s\n", taskID)

	// 模拟 Agent 执行多个步骤
	for i := 1; i <= 10; i++ {
		scheduler.NotifyStep(i)
		time.Sleep(100 * time.Millisecond)
	}

	time.Sleep(200 * time.Millisecond)
	fmt.Println()
}

// 示例 2: 时间间隔触发
func demonstrateIntervalTrigger() {
	fmt.Println("--- 示例 2: 时间间隔触发 ---")

	scheduler := core.NewScheduler(nil)
	defer scheduler.Shutdown()

	// 每 500ms 执行一次
	taskID, err := scheduler.EveryInterval(500*time.Millisecond, func(ctx context.Context) error {
		fmt.Printf("✓ 定时任务触发: %s\n", time.Now().Format("15:04:05.000"))
		return nil
	})

	if err != nil {
		log.Fatalf("创建定时任务失败: %v", err)
	}

	fmt.Printf("定时任务已创建: %s\n", taskID)
	fmt.Println("运行 2 秒...")

	// 运行 2 秒
	time.Sleep(2 * time.Second)
	fmt.Println()
}

// 示例 3: 步骤监听器
func demonstrateStepListeners() {
	fmt.Println("--- 示例 3: 步骤监听器 ---")

	scheduler := core.NewScheduler(nil)
	defer scheduler.Shutdown()

	// 添加全局步骤监听器
	cancel := scheduler.OnStep(func(ctx context.Context, stepCount int) error {
		fmt.Printf("  [监听器] 步骤 %d 执行完成\n", stepCount)
		return nil
	})
	defer cancel()

	fmt.Println("已添加步骤监听器")

	// 模拟几个步骤
	for i := 1; i <= 5; i++ {
		scheduler.NotifyStep(i)
		time.Sleep(100 * time.Millisecond)
	}

	time.Sleep(200 * time.Millisecond)
	fmt.Println()
}

// 示例 4: 多任务调度
func demonstrateMultipleTasks() {
	fmt.Println("--- 示例 4: 多任务调度 ---")

	scheduler := core.NewScheduler(nil)
	defer scheduler.Shutdown()

	// 任务 1: 每 2 步
	_, _ = scheduler.EverySteps(2, func(ctx context.Context, stepCount int) error {
		fmt.Printf("  [任务A] 每2步 - 第 %d 步\n", stepCount)
		return nil
	})

	// 任务 2: 每 5 步
	_, _ = scheduler.EverySteps(5, func(ctx context.Context, stepCount int) error {
		fmt.Printf("  [任务B] 每5步 - 第 %d 步\n", stepCount)
		return nil
	})

	// 任务 3: 每 300ms
	_, _ = scheduler.EveryInterval(300*time.Millisecond, func(ctx context.Context) error {
		fmt.Printf("  [任务C] 定时触发\n")
		return nil
	})

	fmt.Printf("已创建 3 个任务\n")
	fmt.Printf("步骤任务: %d, 定时任务: %d\n",
		scheduler.GetStepTaskCount(),
		scheduler.GetIntervalTaskCount())

	// 模拟步骤
	for i := 1; i <= 10; i++ {
		scheduler.NotifyStep(i)
		time.Sleep(150 * time.Millisecond)
	}

	time.Sleep(200 * time.Millisecond)
	fmt.Println()
}

// 示例 5: 任务取消
func demonstrateTaskCancellation() {
	fmt.Println("--- 示例 5: 任务取消 ---")

	scheduler := core.NewScheduler(nil)
	defer scheduler.Shutdown()

	// 创建定时任务
	taskID, _ := scheduler.EveryInterval(300*time.Millisecond, func(ctx context.Context) error {
		fmt.Println("  ⏰ 定时任务执行")
		return nil
	})

	fmt.Printf("任务 %s 已创建,运行 1 秒...\n", taskID)
	time.Sleep(1 * time.Second)

	// 取消任务
	err := scheduler.Cancel(taskID)
	if err != nil {
		log.Printf("取消任务失败: %v", err)
	} else {
		fmt.Println("✓ 任务已取消")
	}

	fmt.Println("再等待 1 秒,验证任务已停止...")
	time.Sleep(1 * time.Second)
	fmt.Println()
}

// 示例 6: 触发回调监控
func demonstrateTriggerMonitoring() {
	fmt.Println("--- 示例 6: 触发回调监控 ---")

	var triggerCount int

	scheduler := core.NewScheduler(&core.SchedulerOptions{
		OnTrigger: func(taskID string, spec string, kind core.TriggerKind) {
			triggerCount++
			fmt.Printf("  [监控] 任务触发 - ID: %s, 类型: %s, 规格: %s\n",
				taskID[:8], kind, spec)
		},
	})
	defer scheduler.Shutdown()

	// 创建步骤任务
	_, err := scheduler.EverySteps(2, func(ctx context.Context, stepCount int) error {
		return nil
	})
	if err != nil {
		log.Fatalf("创建步骤任务失败: %v", err)
	}

	// 创建定时任务
	_, err = scheduler.EveryInterval(200*time.Millisecond, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		log.Fatalf("创建定时任务失败: %v", err)
	}

	// 模拟执行
	for i := 1; i <= 6; i++ {
		scheduler.NotifyStep(i)
		time.Sleep(150 * time.Millisecond)
	}

	time.Sleep(300 * time.Millisecond)

	fmt.Printf("\n总触发次数: %d\n", triggerCount)
	fmt.Println()
}
