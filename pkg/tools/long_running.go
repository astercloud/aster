package tools

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// LongRunningTool 长时运行工具接口
// 参考 Google ADK-Go 的长时运行工具设计
//
// 使用场景:
// - 文件上传/下载
// - 数据库备份
// - 机器学习训练
// - 大数据处理
type LongRunningTool interface {
	Tool

	// IsLongRunning 标记为长时运行工具
	IsLongRunning() bool

	// StartAsync 异步启动工具执行
	// 返回任务 ID，可用于查询状态或取消
	StartAsync(ctx context.Context, args map[string]interface{}) (string, error)

	// GetStatus 获取任务执行状态
	GetStatus(ctx context.Context, taskID string) (*TaskStatus, error)

	// Cancel 取消正在执行的任务
	Cancel(ctx context.Context, taskID string) error
}

// TaskStatus 任务状态
type TaskStatus struct {
	TaskID    string                 // 任务 ID
	State     TaskState              // 当前状态
	Progress  float64                // 进度 0.0 - 1.0
	Result    interface{}            // 执行结果（完成时）
	Error     error                  // 错误信息（失败时）
	StartTime time.Time              // 开始时间
	EndTime   *time.Time             // 结束时间（完成/失败/取消时）
	Metadata  map[string]interface{} // 额外元数据
}

// TaskState 任务状态枚举
type TaskState int

const (
	TaskStatePending   TaskState = iota // 待执行
	TaskStateRunning                    // 执行中
	TaskStateCompleted                  // 已完成
	TaskStateFailed                     // 失败
	TaskStateCancelled                  // 已取消
)

// String 返回状态的字符串表示
func (s TaskState) String() string {
	switch s {
	case TaskStatePending:
		return "pending"
	case TaskStateRunning:
		return "running"
	case TaskStateCompleted:
		return "completed"
	case TaskStateFailed:
		return "failed"
	case TaskStateCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// IsTerminal 判断是否为终态
func (s TaskState) IsTerminal() bool {
	return s == TaskStateCompleted || s == TaskStateFailed || s == TaskStateCancelled
}

// LongRunningExecutor 长时运行工具执行器
// 管理异步任务的生命周期
type LongRunningExecutor struct {
	tasks   sync.Map // taskID -> *TaskStatus
	cancels sync.Map // taskID -> context.CancelFunc
}

// NewLongRunningExecutor 创建长时运行工具执行器
func NewLongRunningExecutor() *LongRunningExecutor {
	return &LongRunningExecutor{}
}

// StartAsync 异步启动工具
func (e *LongRunningExecutor) StartAsync(
	ctx context.Context,
	tool Tool,
	args map[string]interface{},
) (string, error) {
	// 1. 生成任务 ID
	taskID := generateTaskID()

	// 2. 创建任务状态
	status := &TaskStatus{
		TaskID:    taskID,
		State:     TaskStatePending,
		Progress:  0.0,
		StartTime: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
	e.tasks.Store(taskID, status)

	// 3. 创建可取消的 context
	taskCtx, cancel := context.WithCancel(ctx)
	e.cancels.Store(taskID, cancel)

	// 4. 异步执行
	go func() {
		defer cancel()

		// 更新状态为 Running
		_ = e.updateState(taskID, TaskStateRunning)

		// 执行工具（传递 nil ToolContext，因为 long-running 工具不需要它）
		result, err := tool.Execute(taskCtx, args, nil)

		// 更新最终状态
		now := time.Now()
		if err != nil {
			if taskCtx.Err() == context.Canceled {
				_ = e.updateStatus(taskID, func(s *TaskStatus) {
					s.State = TaskStateCancelled
					s.Error = fmt.Errorf("task cancelled")
					s.EndTime = &now
				})
			} else {
				_ = e.updateStatus(taskID, func(s *TaskStatus) {
					s.State = TaskStateFailed
					s.Error = err
					s.EndTime = &now
				})
			}
		} else {
			_ = e.updateStatus(taskID, func(s *TaskStatus) {
				s.State = TaskStateCompleted
				s.Progress = 1.0
				s.Result = result
				s.EndTime = &now
			})
		}

		// 清理取消函数
		e.cancels.Delete(taskID)
	}()

	return taskID, nil
}

// GetStatus 获取任务状态
func (e *LongRunningExecutor) GetStatus(ctx context.Context, taskID string) (*TaskStatus, error) {
	value, ok := e.tasks.Load(taskID)
	if !ok {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	status := value.(*TaskStatus)

	// 返回副本，避免外部修改
	return &TaskStatus{
		TaskID:    status.TaskID,
		State:     status.State,
		Progress:  status.Progress,
		Result:    status.Result,
		Error:     status.Error,
		StartTime: status.StartTime,
		EndTime:   status.EndTime,
		Metadata:  copyMetadata(status.Metadata),
	}, nil
}

// Cancel 取消任务
func (e *LongRunningExecutor) Cancel(ctx context.Context, taskID string) error {
	// 1. 检查任务是否存在
	value, ok := e.tasks.Load(taskID)
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	status := value.(*TaskStatus)

	// 2. 检查任务是否已终止
	if status.State.IsTerminal() {
		return fmt.Errorf("task already in terminal state: %s", status.State)
	}

	// 3. 调用取消函数
	if cancelFunc, ok := e.cancels.Load(taskID); ok {
		cancelFunc.(context.CancelFunc)()
		return nil
	}

	return fmt.Errorf("cancel function not found for task: %s", taskID)
}

// ListTasks 列出所有任务
func (e *LongRunningExecutor) ListTasks(filter func(*TaskStatus) bool) []*TaskStatus {
	var tasks []*TaskStatus

	e.tasks.Range(func(key, value interface{}) bool {
		status := value.(*TaskStatus)
		if filter == nil || filter(status) {
			tasks = append(tasks, &TaskStatus{
				TaskID:    status.TaskID,
				State:     status.State,
				Progress:  status.Progress,
				Result:    status.Result,
				Error:     status.Error,
				StartTime: status.StartTime,
				EndTime:   status.EndTime,
				Metadata:  copyMetadata(status.Metadata),
			})
		}
		return true
	})

	return tasks
}

// Cleanup 清理已完成的任务
// 清理指定时间之前完成的任务
func (e *LongRunningExecutor) Cleanup(before time.Time) int {
	var deleted int

	e.tasks.Range(func(key, value interface{}) bool {
		status := value.(*TaskStatus)

		// 只清理终态任务
		if status.State.IsTerminal() && status.EndTime != nil {
			if status.EndTime.Before(before) {
				e.tasks.Delete(key)
				e.cancels.Delete(key)
				deleted++
			}
		}
		return true
	})

	return deleted
}

// UpdateProgress 更新任务进度（供工具内部调用）
func (e *LongRunningExecutor) UpdateProgress(taskID string, progress float64, metadata map[string]interface{}) error {
	return e.updateStatus(taskID, func(s *TaskStatus) {
		s.Progress = progress
		if metadata != nil {
			for k, v := range metadata {
				s.Metadata[k] = v
			}
		}
	})
}

// 辅助方法

// updateState 更新任务状态
func (e *LongRunningExecutor) updateState(taskID string, state TaskState) error {
	return e.updateStatus(taskID, func(s *TaskStatus) {
		s.State = state
	})
}

// updateStatus 更新任务状态（通用）
func (e *LongRunningExecutor) updateStatus(taskID string, updater func(*TaskStatus)) error {
	value, ok := e.tasks.Load(taskID)
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	status := value.(*TaskStatus)
	updater(status)
	e.tasks.Store(taskID, status)

	return nil
}

// copyMetadata 复制元数据
func copyMetadata(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}

	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// generateTaskID 生成任务 ID
func generateTaskID() string {
	return "task_" + uuid.New().String()
}

// BaseLongRunningTool 长时运行工具的基础实现
// 可以嵌入到具体工具中
type BaseLongRunningTool struct {
	BaseTool
	executor *LongRunningExecutor
}

// NewBaseLongRunningTool 创建基础长时运行工具
func NewBaseLongRunningTool(name, description string, executor *LongRunningExecutor) *BaseLongRunningTool {
	return &BaseLongRunningTool{
		BaseTool: BaseTool{
			ToolName:        name,
			ToolDescription: description,
		},
		executor: executor,
	}
}

// IsLongRunning 实现 LongRunningTool 接口
func (t *BaseLongRunningTool) IsLongRunning() bool {
	return true
}

// StartAsync 实现 LongRunningTool 接口
func (t *BaseLongRunningTool) StartAsync(ctx context.Context, args map[string]interface{}) (string, error) {
	return t.executor.StartAsync(ctx, t, args)
}

// GetStatus 实现 LongRunningTool 接口
func (t *BaseLongRunningTool) GetStatus(ctx context.Context, taskID string) (*TaskStatus, error) {
	return t.executor.GetStatus(ctx, taskID)
}

// Cancel 实现 LongRunningTool 接口
func (t *BaseLongRunningTool) Cancel(ctx context.Context, taskID string) error {
	return t.executor.Cancel(ctx, taskID)
}

// Execute 需要由具体工具实现
func (t *BaseLongRunningTool) Execute(ctx context.Context, args map[string]interface{}, tc *ToolContext) (interface{}, error) {
	return nil, fmt.Errorf("Execute() must be implemented by concrete tool")
}

// WaitFor 等待任务完成（辅助函数）
func WaitForCompletion(executor *LongRunningExecutor, taskID string, pollInterval time.Duration, timeout time.Duration) (*TaskStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for task: %s", taskID)

		case <-ticker.C:
			status, err := executor.GetStatus(ctx, taskID)
			if err != nil {
				return nil, err
			}

			if status.State.IsTerminal() {
				return status, nil
			}
		}
	}
}
