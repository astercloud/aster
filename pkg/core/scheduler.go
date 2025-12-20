package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/logging"
)

var schedulerLog = logging.ForComponent("Scheduler")

// TaskCallback 任务回调函数
type TaskCallback func(ctx context.Context) error

// StepCallback 步骤回调函数
type StepCallback func(ctx context.Context, stepCount int) error

// TriggerKind 触发类型
type TriggerKind string

const (
	TriggerKindStep      TriggerKind = "step"     // 步骤触发
	TriggerKindInterval  TriggerKind = "interval" // 时间间隔触发
	TriggerKindCron      TriggerKind = "cron"     // Cron 表达式触发 (未实现)
	TriggerKindFileWatch TriggerKind = "file"     // 文件变化触发 (未实现)
)

// ScheduledTask 调度任务
type ScheduledTask struct {
	ID           string
	Kind         TriggerKind
	Spec         string // 任务规格: "step:5", "interval:10s", "cron:* * * * *"
	Callback     TaskCallback
	Agent        *agent.Agent // 可选: 关联的 Agent
	LastTrigger  time.Time
	TriggerCount int64
	Enabled      bool
}

// StepTask 步骤任务
type StepTask struct {
	ID            string
	Every         int // 每 N 步触发一次
	Callback      StepCallback
	LastTriggered int
}

// IntervalTask 时间间隔任务
type IntervalTask struct {
	ID       string
	Interval time.Duration
	Callback TaskCallback
	ticker   *time.Ticker
	stopCh   chan struct{}
}

// SchedulerOptions Scheduler 配置
type SchedulerOptions struct {
	// 触发回调 (用于监控和日志)
	OnTrigger func(taskID string, spec string, kind TriggerKind)
}

// Scheduler 任务调度器
// 支持步骤触发、定时触发、Cron 表达式 (TODO)、文件监听 (TODO)
type Scheduler struct {
	mu sync.RWMutex

	// 步骤任务
	stepTasks     map[string]*StepTask
	stepListeners []StepCallback

	// 时间间隔任务
	intervalTasks map[string]*IntervalTask

	// 配置
	opts *SchedulerOptions

	// 控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewScheduler 创建调度器
func NewScheduler(opts *SchedulerOptions) *Scheduler {
	if opts == nil {
		opts = &SchedulerOptions{}
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		stepTasks:     make(map[string]*StepTask),
		stepListeners: make([]StepCallback, 0),
		intervalTasks: make(map[string]*IntervalTask),
		opts:          opts,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// EverySteps 每 N 步执行一次
func (s *Scheduler) EverySteps(every int, callback StepCallback) (string, error) {
	if every <= 0 {
		return "", fmt.Errorf("every must be positive, got %d", every)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := generateTaskID("step")
	task := &StepTask{
		ID:            id,
		Every:         every,
		Callback:      callback,
		LastTriggered: 0,
	}

	s.stepTasks[id] = task
	return id, nil
}

// OnStep 监听每一步
func (s *Scheduler) OnStep(callback StepCallback) func() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 记录索引
	index := len(s.stepListeners)
	s.stepListeners = append(s.stepListeners, callback)

	// 返回取消函数
	canceled := false
	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		if canceled {
			return
		}
		canceled = true

		// 从列表中移除
		if index < len(s.stepListeners) {
			// 用最后一个元素替换当前元素,然后截断
			s.stepListeners[index] = nil
			// 使用标记删除而不是实际删除,避免索引问题
		}
	}
}

// NotifyStep 通知步骤变化 (由 Agent 调用)
func (s *Scheduler) NotifyStep(stepCount int) {
	s.mu.RLock()

	// 复制监听器和任务列表,避免长时间持锁
	listeners := make([]StepCallback, len(s.stepListeners))
	copy(listeners, s.stepListeners)

	tasks := make([]*StepTask, 0, len(s.stepTasks))
	for _, task := range s.stepTasks {
		tasks = append(tasks, task)
	}

	s.mu.RUnlock()

	// 通知监听器
	for _, listener := range listeners {
		if listener == nil {
			continue // 跳过已取消的监听器
		}
		go func(cb StepCallback) {
			if err := cb(s.ctx, stepCount); err != nil {
				schedulerLog.Warn(s.ctx, "step callback error", map[string]any{"step": stepCount, "error": err})
			}
		}(listener)
	}

	// 检查并触发任务
	for _, task := range tasks {
		shouldTrigger := stepCount-task.LastTriggered >= task.Every
		if !shouldTrigger {
			continue
		}

		// 更新触发时间
		s.mu.Lock()
		task.LastTriggered = stepCount
		s.mu.Unlock()

		// 异步执行回调
		go func(t *StepTask) {
			if err := t.Callback(s.ctx, stepCount); err != nil {
				schedulerLog.Warn(s.ctx, "step task callback error", map[string]any{"task_id": t.ID, "error": err})
			}

			// 通知触发
			if s.opts.OnTrigger != nil {
				s.opts.OnTrigger(t.ID, fmt.Sprintf("step:%d", t.Every), TriggerKindStep)
			}
		}(task)
	}
}

// EveryInterval 每隔一段时间执行
func (s *Scheduler) EveryInterval(interval time.Duration, callback TaskCallback) (string, error) {
	if interval <= 0 {
		return "", fmt.Errorf("interval must be positive, got %v", interval)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := generateTaskID("interval")
	ticker := time.NewTicker(interval)
	stopCh := make(chan struct{})

	task := &IntervalTask{
		ID:       id,
		Interval: interval,
		Callback: callback,
		ticker:   ticker,
		stopCh:   stopCh,
	}

	s.intervalTasks[id] = task

	// 启动定时器
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		for {
			select {
			case <-ticker.C:
				// 执行回调
				if err := callback(s.ctx); err != nil {
					schedulerLog.Warn(s.ctx, "interval task callback error", map[string]any{"task_id": id, "error": err})
				}

				// 通知触发
				if s.opts.OnTrigger != nil {
					s.opts.OnTrigger(id, fmt.Sprintf("interval:%v", interval), TriggerKindInterval)
				}

			case <-stopCh:
				ticker.Stop()
				return

			case <-s.ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()

	return id, nil
}

// Schedule 使用调度规格创建任务
func (s *Scheduler) Schedule(spec string, callback TaskCallback) (string, error) {
	// 解析规格
	// 支持格式:
	// - "step:N" - 每 N 步
	// - "interval:Ns" - 每 N 秒
	// - "interval:Nm" - 每 N 分钟
	// - "cron:* * * * *" - Cron 表达式 (TODO)

	// 简化实现:仅支持 interval
	var duration time.Duration
	_, err := fmt.Sscanf(spec, "interval:%s", &duration)
	if err != nil {
		return "", fmt.Errorf("invalid schedule spec: %s", spec)
	}

	return s.EveryInterval(duration, callback)
}

// Cancel 取消任务
func (s *Scheduler) Cancel(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查步骤任务
	if _, exists := s.stepTasks[taskID]; exists {
		delete(s.stepTasks, taskID)
		return nil
	}

	// 检查时间间隔任务
	if task, exists := s.intervalTasks[taskID]; exists {
		close(task.stopCh)
		delete(s.intervalTasks, taskID)
		return nil
	}

	return fmt.Errorf("task not found: %s", taskID)
}

// Clear 清空所有任务
func (s *Scheduler) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 停止所有时间间隔任务
	for _, task := range s.intervalTasks {
		close(task.stopCh)
	}

	// 清空
	s.stepTasks = make(map[string]*StepTask)
	s.stepListeners = make([]StepCallback, 0)
	s.intervalTasks = make(map[string]*IntervalTask)
}

// Shutdown 关闭调度器
func (s *Scheduler) Shutdown() {
	s.cancel()
	s.Clear()
	s.wg.Wait()
}

// GetStepTaskCount 获取步骤任务数量
func (s *Scheduler) GetStepTaskCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.stepTasks)
}

// GetIntervalTaskCount 获取时间间隔任务数量
func (s *Scheduler) GetIntervalTaskCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.intervalTasks)
}

// GetStepListenerCount 获取步骤监听器数量
func (s *Scheduler) GetStepListenerCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.stepListeners)
}

// generateTaskID 生成任务 ID
func generateTaskID(prefix string) string {
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), time.Now().Nanosecond()%1000)
}
