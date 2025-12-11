package builtin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/types"
)

// TaskExecutor 任务执行器
// 连接 Task 工具和 SubAgentManager
type TaskExecutor struct {
	mu sync.RWMutex

	// 执行器工厂，用于创建不同类型的子 Agent 执行器
	executorFactory SubAgentExecutorFactory

	// 运行中的任务
	tasks map[string]*TaskExecution
}

// SubAgentExecutorFactory 子 Agent 执行器工厂接口
type SubAgentExecutorFactory interface {
	// Create 创建指定类型的子 Agent 执行器
	Create(agentType string) (types.SubAgentExecutor, error)

	// ListTypes 列出支持的子 Agent 类型
	ListTypes() []string
}

// TaskExecutionHandle 任务执行句柄
type TaskExecutionHandle struct {
	TaskID       string
	AgentType    string
	Status       string // "pending", "running", "completed", "failed", "cancelled"
	StartTime    time.Time
	EndTime      *time.Time
	Result       *types.SubAgentResult
	ProgressChan <-chan *types.SubAgentProgressEvent
	CancelFunc   context.CancelFunc
}

// NewTaskExecutor 创建任务执行器
func NewTaskExecutor() *TaskExecutor {
	return &TaskExecutor{
		tasks: make(map[string]*TaskExecution),
	}
}

// SetExecutorFactory 设置执行器工厂
func (te *TaskExecutor) SetExecutorFactory(factory SubAgentExecutorFactory) {
	te.mu.Lock()
	defer te.mu.Unlock()
	te.executorFactory = factory
}

// Execute 执行任务（同步）
func (te *TaskExecutor) Execute(ctx context.Context, agentType, prompt string, opts *TaskExecuteOptions) (*TaskExecution, error) {
	te.mu.RLock()
	factory := te.executorFactory
	te.mu.RUnlock()

	if factory == nil {
		return nil, fmt.Errorf("executor factory not configured, subagent execution not available")
	}

	// 创建执行器
	executor, err := factory.Create(agentType)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor for %s: %w", agentType, err)
	}

	// 构建请求
	req := &types.SubAgentRequest{
		AgentType: agentType,
		Task:      prompt,
		Context:   opts.Context,
	}

	if opts.Timeout > 0 {
		req.Timeout = opts.Timeout
	}

	// 执行
	startTime := time.Now()
	result, err := executor.Execute(ctx, req)

	execution := &TaskExecution{
		TaskID:    fmt.Sprintf("task_%d", startTime.UnixNano()),
		Subagent:  agentType,
		Model:     opts.Model,
		Status:    "completed",
		StartTime: startTime,
		Duration:  time.Since(startTime),
	}

	if err != nil {
		execution.Status = "failed"
		execution.Error = err.Error()
		return execution, nil
	}

	if result != nil {
		execution.Result = result.Output
		if result.Success {
			execution.Status = "completed"
		} else {
			execution.Status = "failed"
			execution.Error = result.Error
		}
		execution.Metadata = map[string]any{
			"tokens_used": result.TokensUsed,
			"step_count":  result.StepCount,
			"artifacts":   result.Artifacts,
		}
	}

	// 记录任务
	te.mu.Lock()
	te.tasks[execution.TaskID] = execution
	te.mu.Unlock()

	return execution, nil
}

// ExecuteAsync 异步执行任务
func (te *TaskExecutor) ExecuteAsync(ctx context.Context, agentType, prompt string, opts *TaskExecuteOptions) (*TaskExecutionHandle, error) {
	te.mu.RLock()
	factory := te.executorFactory
	te.mu.RUnlock()

	if factory == nil {
		return nil, fmt.Errorf("executor factory not configured")
	}

	// 创建执行器
	executor, err := factory.Create(agentType)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())
	execCtx, cancel := context.WithCancel(ctx)

	handle := &TaskExecutionHandle{
		TaskID:     taskID,
		AgentType:  agentType,
		Status:     "pending",
		StartTime:  time.Now(),
		CancelFunc: cancel,
	}

	// 异步执行
	go func() {
		handle.Status = "running"

		req := &types.SubAgentRequest{
			AgentType: agentType,
			Task:      prompt,
			Context:   opts.Context,
		}

		if opts.Timeout > 0 {
			req.Timeout = opts.Timeout
		}

		result, err := executor.Execute(execCtx, req)

		now := time.Now()
		handle.EndTime = &now

		if err != nil {
			handle.Status = "failed"
			handle.Result = &types.SubAgentResult{
				AgentType: agentType,
				Success:   false,
				Error:     err.Error(),
				Duration:  time.Since(handle.StartTime),
			}
			return
		}

		handle.Result = result
		if result.Success {
			handle.Status = "completed"
		} else {
			handle.Status = "failed"
		}
	}()

	return handle, nil
}

// GetTask 获取任务信息
func (te *TaskExecutor) GetTask(taskID string) (*TaskExecution, error) {
	te.mu.RLock()
	defer te.mu.RUnlock()

	task, ok := te.tasks[taskID]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	return task, nil
}

// ListTasks 列出所有任务
func (te *TaskExecutor) ListTasks() []*TaskExecution {
	te.mu.RLock()
	defer te.mu.RUnlock()

	tasks := make([]*TaskExecution, 0, len(te.tasks))
	for _, t := range te.tasks {
		tasks = append(tasks, t)
	}
	return tasks
}

// TaskExecuteOptions 任务执行选项
type TaskExecuteOptions struct {
	Model    string
	Timeout  time.Duration
	Context  map[string]any
	Priority int
	Async    bool
}

// 全局任务执行器
var globalTaskExecutor *TaskExecutor
var taskExecutorOnce sync.Once

// GetGlobalTaskExecutor 获取全局任务执行器
func GetGlobalTaskExecutor() *TaskExecutor {
	taskExecutorOnce.Do(func() {
		globalTaskExecutor = NewTaskExecutor()
	})
	return globalTaskExecutor
}

// SetGlobalTaskExecutorFactory 设置全局任务执行器工厂
// 应用层在初始化时调用此函数注入 SubAgentManager
func SetGlobalTaskExecutorFactory(factory SubAgentExecutorFactory) {
	GetGlobalTaskExecutor().SetExecutorFactory(factory)
}

// SetFactoryFromAgent agent 包调用此函数注入工厂
// 这是 agent.InitializeTaskExecutor 使用的入口点
func SetFactoryFromAgent(factory SubAgentExecutorFactory) {
	GetGlobalTaskExecutor().SetExecutorFactory(factory)
}
