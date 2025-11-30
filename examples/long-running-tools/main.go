// LongRunningTools 演示长时运行工具的使用，包括异步执行、进度跟踪、
// 任务取消和并发多任务处理。
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/astercloud/aster/pkg/tools"
)

func main() {
	ctx := context.Background()

	// ====== 示例 1: 基础异步执行 ======
	fmt.Println("=== Example 1: Basic Async Execution ===")
	basicAsyncExample(ctx)

	// ====== 示例 2: 进度跟踪 ======
	fmt.Println("\n=== Example 2: Progress Tracking ===")
	progressTrackingExample(ctx)

	// ====== 示例 3: 任务取消 ======
	fmt.Println("\n=== Example 3: Task Cancellation ===")
	cancellationExample(ctx)

	// ====== 示例 4: 并发多任务 ======
	fmt.Println("\n=== Example 4: Concurrent Tasks ===")
	concurrentTasksExample(ctx)

	// ====== 示例 5: 任务清理 ======
	fmt.Println("\n=== Example 5: Task Cleanup ===")
	cleanupExample(ctx)

	fmt.Println("\n✅ All examples completed!")
}

// basicAsyncExample 基础异步执行示例
func basicAsyncExample(ctx context.Context) {
	// 1. 创建执行器
	executor := tools.NewLongRunningExecutor()

	// 2. 创建模拟的长时运行工具
	tool := NewMockDataProcessingTool(executor)

	// 3. 启动异步任务
	taskID, err := tool.StartAsync(ctx, map[string]any{
		"data_size": 1000,
		"delay_ms":  500,
	})
	if err != nil {
		log.Fatalf("Failed to start task: %v", err)
	}

	fmt.Printf("✅ Task started: %s\n", taskID)

	// 4. 等待任务完成
	status, err := tools.WaitForCompletion(executor, taskID, 100*time.Millisecond, 5*time.Second)
	if err != nil {
		log.Fatalf("Failed to wait for task: %v", err)
	}

	fmt.Printf("✅ Task completed: state=%s, result=%v\n", status.State, status.Result)
}

// progressTrackingExample 进度跟踪示例
func progressTrackingExample(ctx context.Context) {
	executor := tools.NewLongRunningExecutor()
	tool := NewMockFileUploadTool(executor)

	// 启动任务
	taskID, _ := tool.StartAsync(ctx, map[string]any{
		"file_size":  10000,
		"chunk_size": 1000,
	})

	fmt.Printf("✅ Upload started: %s\n", taskID)

	// 轮询进度
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for range 10 {
		<-ticker.C

		status, err := executor.GetStatus(ctx, taskID)
		if err != nil {
			log.Fatalf("Failed to get status: %v", err)
		}

		fmt.Printf("   Progress: %.1f%% (%s)\n", status.Progress*100, status.State)

		if status.State.IsTerminal() {
			break
		}
	}

	fmt.Println("✅ Upload completed")
}

// cancellationExample 任务取消示例
func cancellationExample(ctx context.Context) {
	executor := tools.NewLongRunningExecutor()
	tool := NewMockDataProcessingTool(executor)

	// 启动一个耗时任务
	taskID, _ := tool.StartAsync(ctx, map[string]any{
		"data_size": 10000,
		"delay_ms":  3000, // 3秒
	})

	fmt.Printf("✅ Task started: %s\n", taskID)

	// 等待一会儿
	time.Sleep(500 * time.Millisecond)

	// 取消任务
	if err := executor.Cancel(ctx, taskID); err != nil {
		log.Fatalf("Failed to cancel task: %v", err)
	}

	fmt.Println("✅ Cancel signal sent")

	// 验证任务已取消
	time.Sleep(200 * time.Millisecond)
	status, _ := executor.GetStatus(ctx, taskID)
	fmt.Printf("✅ Task state: %s\n", status.State)
}

// concurrentTasksExample 并发多任务示例
func concurrentTasksExample(ctx context.Context) {
	executor := tools.NewLongRunningExecutor()
	tool := NewMockDataProcessingTool(executor)

	// 启动多个任务
	taskIDs := make([]string, 5)
	for i := range 5 {
		taskID, _ := tool.StartAsync(ctx, map[string]any{
			"data_size": 100 * (i + 1),
			"delay_ms":  200,
		})
		taskIDs[i] = taskID
	}

	fmt.Printf("✅ Started %d concurrent tasks\n", len(taskIDs))

	// 等待所有任务完成
	for _, taskID := range taskIDs {
		_, err := tools.WaitForCompletion(executor, taskID, 50*time.Millisecond, 5*time.Second)
		if err != nil {
			log.Printf("Task %s failed: %v", taskID, err)
		}
	}

	// 列出所有任务
	allTasks := executor.ListTasks(nil)
	completedTasks := executor.ListTasks(func(s *tools.TaskStatus) bool {
		return s.State == tools.TaskStateCompleted
	})

	fmt.Printf("✅ All tasks: %d, Completed: %d\n", len(allTasks), len(completedTasks))
}

// cleanupExample 任务清理示例
func cleanupExample(ctx context.Context) {
	executor := tools.NewLongRunningExecutor()
	tool := NewMockDataProcessingTool(executor)

	// 创建一些任务
	for range 3 {
		taskID, _ := tool.StartAsync(ctx, map[string]any{
			"data_size": 100,
			"delay_ms":  100,
		})
		_, _ = tools.WaitForCompletion(executor, taskID, 50*time.Millisecond, 2*time.Second)
	}

	fmt.Println("✅ Created 3 completed tasks")

	// 清理 1 秒前的任务
	deleted := executor.Cleanup(time.Now().Add(-1 * time.Second))
	fmt.Printf("✅ Cleaned up %d old tasks\n", deleted)

	// 验证
	remaining := executor.ListTasks(nil)
	fmt.Printf("✅ Remaining tasks: %d\n", len(remaining))
}

// ============================================================
// 模拟工具实现
// ============================================================

// MockDataProcessingTool 模拟数据处理工具
type MockDataProcessingTool struct {
	*tools.BaseLongRunningTool
}

func NewMockDataProcessingTool(executor *tools.LongRunningExecutor) *MockDataProcessingTool {
	return &MockDataProcessingTool{
		BaseLongRunningTool: tools.NewBaseLongRunningTool(
			"mock_data_processing",
			"Mock data processing tool for demonstration",
			executor,
		),
	}
}

func (t *MockDataProcessingTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	dataSize := args["data_size"].(int)
	delayMs := args["delay_ms"].(int)

	// 模拟处理
	select {
	case <-time.After(time.Duration(delayMs) * time.Millisecond):
		return map[string]any{
			"processed_items": dataSize,
			"duration_ms":     delayMs,
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// MockFileUploadTool 模拟文件上传工具
type MockFileUploadTool struct {
	*tools.BaseLongRunningTool
	executor *tools.LongRunningExecutor
}

func NewMockFileUploadTool(executor *tools.LongRunningExecutor) *MockFileUploadTool {
	return &MockFileUploadTool{
		BaseLongRunningTool: tools.NewBaseLongRunningTool(
			"mock_file_upload",
			"Mock file upload tool with progress tracking",
			executor,
		),
		executor: executor,
	}
}

func (t *MockFileUploadTool) StartAsync(ctx context.Context, args map[string]any) (string, error) {
	// 使用自定义的执行逻辑
	taskID := "task_" + fmt.Sprintf("%d", time.Now().UnixNano())

	// 创建任务状态
	status := &tools.TaskStatus{
		TaskID:    taskID,
		State:     tools.TaskStatePending,
		Progress:  0.0,
		StartTime: time.Now(),
		Metadata:  make(map[string]any),
	}
	_ = t.executor.UpdateProgress(taskID, 0.0, nil)

	// 异步执行
	go func() {
		fileSize := args["file_size"].(int)
		chunkSize := args["chunk_size"].(int)

		chunks := fileSize / chunkSize
		for i := range chunks {
			select {
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
				progress := float64(i+1) / float64(chunks)
				_ = t.executor.UpdateProgress(taskID, progress, map[string]any{
					"uploaded_chunks": i + 1,
					"total_chunks":    chunks,
				})
			}
		}

		// 完成
		now := time.Now()
		status.State = tools.TaskStateCompleted
		status.Progress = 1.0
		status.EndTime = &now
		status.Result = map[string]any{
			"uploaded_bytes": fileSize,
			"chunks":         chunks,
		}
	}()

	return taskID, nil
}

func (t *MockFileUploadTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	// 简化的同步执行
	return nil, fmt.Errorf("use StartAsync for file upload")
}

// ============================================================
// 实际应用场景
// ============================================================

// realWorldExamples 实际应用场景示例（仅作为文档参考）
var _ = func() {
	/*
		1. 文件处理工具
		   - 大文件上传/下载
		   - 视频转码
		   - 图像批处理

		   type FileProcessingTool struct {
		       *tools.BaseLongRunningTool
		   }

		   func (t *FileProcessingTool) Execute(ctx context.Context, args map[string]any) (any, error) {
		       filePath := args["file_path"].(string)

		       // 处理文件，定期更新进度
		       for progress := 0.0; progress < 1.0; progress += 0.1 {
		           t.executor.UpdateProgress(taskID, progress, map[string]any{
		               "current_file": filePath,
		           })

		           // ... 实际处理逻辑 ...
		       }

		       return result, nil
		   }

		2. 数据库备份工具
		   type DatabaseBackupTool struct {
		       *tools.BaseLongRunningTool
		   }

		   func (t *DatabaseBackupTool) Execute(ctx context.Context, args map[string]any) (any, error) {
		       dbName := args["database"].(string)

		       // 备份逻辑
		       // - 计算总表数
		       // - 逐表备份，更新进度
		       // - 压缩和存储

		       return backupInfo, nil
		   }

		3. API 批量请求工具
		   type BatchAPITool struct {
		       *tools.BaseLongRunningTool
		   }

		   func (t *BatchAPITool) Execute(ctx context.Context, args map[string]any) (any, error) {
		       requests := args["requests"].([]Request)

		       results := make([]Result, 0)
		       for i, req := range requests {
		           result := callAPI(req)
		           results = append(results, result)

		           // 更新进度
		           progress := float64(i+1) / float64(len(requests))
		           t.executor.UpdateProgress(taskID, progress, nil)
		       }

		       return results, nil
		   }
	*/
}

// ============================================================
// 最佳实践
// ============================================================

// bestPractices 最佳实践示例（仅作为文档参考）
var _ = func() {
	/*
		1. 错误处理
		   - 捕获所有错误并更新任务状态
		   - 提供详细的错误信息
		   - 支持重试逻辑

		2. 进度更新
		   - 定期更新进度（不要过于频繁）
		   - 提供有意义的元数据
		   - 避免阻塞主执行流程

		3. 资源管理
		   - 使用 context 支持取消
		   - 清理临时资源
		   - 限制并发任务数量

		4. 监控和日志
		   - 记录任务开始/结束时间
		   - 追踪任务执行指标
		   - 定期清理已完成任务

		5. 超时处理
		   - 为长时运行任务设置合理的超时
		   - 支持超时后的清理逻辑
		   - 提供超时重试机制

		6. 状态持久化（可选）
		   - 将任务状态保存到数据库
		   - 支持任务恢复
		   - 跨进程任务管理
	*/
}
