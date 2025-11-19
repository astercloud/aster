package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/session"
	"github.com/astercloud/aster/pkg/types"
)

// generateEventID 生成事件ID
func generateEventID() string {
	return fmt.Sprintf("event_%d", time.Now().UnixNano())
}

// ParallelWorkFlowAgent 并行工作流Agent
type ParallelWorkFlowAgent struct {
	name     string
	branches []ParallelBranch
	joinType JoinType
	timeout  time.Duration
	onError  ErrorHandling
	metrics  *ParallelMetrics
}

// ParallelBranch 并行分支
type ParallelBranch struct {
	Name    string
	ID      string
	Agent   AgentRef
	Timeout time.Duration
}

// ErrorHandling 错误处理策略
type ErrorHandling string

const (
	ErrorHandlingStop     ErrorHandling = "stop"     // 遇到错误停止所有分支
	ErrorHandlingContinue ErrorHandling = "continue" // 继续其他分支
	ErrorHandlingRetry    ErrorHandling = "retry"    // 重试失败的分支
	ErrorHandlingIgnore   ErrorHandling = "ignore"   // 忽略错误
)

// ParallelMetrics 并行执行指标
type ParallelMetrics struct {
	TotalBranches     int           `json:"total_branches"`
	CompletedBranches int           `json:"completed_branches"`
	FailedBranches    int           `json:"failed_branches"`
	TotalDuration     time.Duration `json:"total_duration"`
	AverageDuration   time.Duration `json:"average_duration"`
	MaxDuration       time.Duration `json:"max_duration"`
	MinDuration       time.Duration `json:"min_duration"`
	StartTime         time.Time     `json:"start_time"`
	EndTime           time.Time     `json:"end_time"`
}

// ParallelBranchResult 并行分支执行结果
type ParallelBranchResult struct {
	Branch   ParallelBranch
	Success  bool
	Duration time.Duration
	Output   interface{}
	Error    error
	Metrics  map[string]interface{}
}

// NewParallelWorkFlowAgent 创建并行工作流Agent
func NewParallelWorkFlowAgent(name string, branches []ParallelBranch, joinType JoinType, timeout time.Duration) *ParallelWorkFlowAgent {
	if timeout == 0 {
		timeout = time.Minute * 5
	}

	if joinType == "" {
		joinType = JoinTypeWait
	}

	return &ParallelWorkFlowAgent{
		name:     name,
		branches: branches,
		joinType: joinType,
		timeout:  timeout,
		onError:  ErrorHandlingContinue,
		metrics:  &ParallelMetrics{},
	}
}

// Name 返回Agent名称
func (p *ParallelWorkFlowAgent) Name() string {
	return p.name
}

// Execute 执行并行工作流
func (p *ParallelWorkFlowAgent) Execute(ctx context.Context, message types.Message, yield func(*session.Event, error) bool) error {
	p.metrics.StartTime = time.Now()

	// 发送开始事件
	yield(&session.Event{
		ID:        generateEventID(),
		Timestamp: time.Now(),
		AgentID:   p.name,
		Author:    "system",
		Content:   types.Message{Content: fmt.Sprintf("Starting parallel workflow: %d branches", len(p.branches))},
		Metadata: map[string]interface{}{
			"workflow_type":  "parallel",
			"total_branches": len(p.branches),
			"join_type":      string(p.joinType),
		},
	}, nil)

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	// 执行所有分支
	results := p.executeBranches(ctx, yield, message)

	// 等待分支完成
	completedResults := p.waitForCompletion(ctx, results)

	// 合并结果
	mergedOutputs := p.mergeResults(completedResults)

	p.metrics.EndTime = time.Now()
	p.metrics.TotalDuration = p.metrics.EndTime.Sub(p.metrics.StartTime)

	// 发送完成事件
	yield(&session.Event{
		ID:        generateEventID(),
		Timestamp: time.Now(),
		AgentID:   p.name,
		Author:    "system",
		Content:   types.Message{Content: fmt.Sprintf("Parallel workflow completed. Results: %d", len(completedResults))},
		Metadata: map[string]interface{}{
			"workflow_type":      "parallel",
			"completed_branches": len(completedResults),
			"total_duration":     p.metrics.TotalDuration,
			"outputs":            mergedOutputs,
		},
	}, nil)

	return nil
}

// executeBranches 执行所有分支
func (p *ParallelWorkFlowAgent) executeBranches(ctx context.Context, yield func(*session.Event, error) bool, message types.Message) []*ParallelBranchResult {
	var wg sync.WaitGroup
	results := make([]*ParallelBranchResult, len(p.branches))
	resultChan := make(chan *ParallelBranchResult, len(p.branches))

	p.metrics.TotalBranches = len(p.branches)

	for i, branch := range p.branches {
		wg.Add(1)
		go func(index int, b ParallelBranch) {
			defer wg.Done()

			result := &ParallelBranchResult{
				Branch:   b,
				Success:  false,
				Duration: 0,
				Output:   nil,
				Error:    nil,
				Metrics:  make(map[string]interface{}),
			}

			start := time.Now()

			// 模拟分支执行
			select {
			case <-time.After(time.Duration(100+i*50) * time.Millisecond):
				result.Success = true
				result.Duration = time.Since(start)
				result.Output = fmt.Sprintf("Branch %s result", b.Name)
			case <-ctx.Done():
				result.Error = ctx.Err()
				result.Duration = time.Since(start)
			}

			resultChan <- result
		}(i, branch)
	}

	// 等待所有分支完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	i := 0
	for result := range resultChan {
		results[i] = result
		i++
	}

	return results
}

// waitForCompletion 等待分支完成
func (p *ParallelWorkFlowAgent) waitForCompletion(ctx context.Context, results []*ParallelBranchResult) []*ParallelBranchResult {
	switch p.joinType {
	case JoinTypeWait:
		// 等待所有分支完成
		return results

	case JoinTypeFirst:
		// 等待第一个分支完成
		for _, result := range results {
			if result.Success {
				return []*ParallelBranchResult{result}
			}
		}
		return results

	case JoinTypeSuccess:
		// 等待一个成功分支完成
		for _, result := range results {
			if result.Success {
				return []*ParallelBranchResult{result}
			}
		}
		return results

	default:
		return results
	}
}

// mergeResults 合并结果
func (p *ParallelWorkFlowAgent) mergeResults(results []*ParallelBranchResult) map[string]interface{} {
	merged := make(map[string]interface{})

	// 统计指标
	successCount := 0
	failedCount := 0
	var totalDuration time.Duration

	for _, result := range results {
		if result.Success {
			successCount++
		} else {
			failedCount++
		}
		totalDuration += result.Duration

		// 合并输出
		if result.Output != nil {
			merged[fmt.Sprintf("branch_%s", result.Branch.ID)] = result.Output
		}
	}

	p.metrics.CompletedBranches = successCount
	p.metrics.FailedBranches = failedCount
	if successCount > 0 {
		p.metrics.AverageDuration = totalDuration / time.Duration(successCount)
	}

	merged["summary"] = map[string]interface{}{
		"total_branches":      len(results),
		"successful_branches": successCount,
		"failed_branches":     failedCount,
		"success_rate":        float64(successCount) / float64(len(results)),
		"total_duration":      totalDuration,
	}

	return merged
}

// GetMetrics 获取并行执行指标
func (p *ParallelWorkFlowAgent) GetMetrics() *ParallelMetrics {
	return p.metrics
}

// AsyncParallelAgent 异步并行Agent
type AsyncParallelAgent struct {
	name         string
	joinType     JoinType
	timeout      time.Duration
	pendingQueue chan *AsyncBranch
	branchesMap  map[string]*AsyncBranch
	mu           sync.RWMutex
	metrics      *AsyncMetrics
}

// AsyncBranch 异步分支
type AsyncBranch struct {
	ID        string
	Name      string
	Status    AsyncBranchStatus
	CreatedAt time.Time
	StartedAt time.Time
	UpdatedAt time.Time
	Result    interface{}
	Error     error
	Metadata  map[string]interface{}
}

// AsyncBranchStatus 异步分支状态
type AsyncBranchStatus string

const (
	AsyncBranchStatusPending   AsyncBranchStatus = "pending"
	AsyncBranchStatusRunning   AsyncBranchStatus = "running"
	AsyncBranchStatusCompleted AsyncBranchStatus = "completed"
	AsyncBranchStatusFailed    AsyncBranchStatus = "failed"
	AsyncBranchStatusCancelled AsyncBranchStatus = "cancelled"
)

// AsyncMetrics 异步指标
type AsyncMetrics struct {
	TotalBranches     int           `json:"total_branches"`
	ActiveBranches    int           `json:"active_branches"`
	CompletedBranches int           `json:"completed_branches"`
	TotalRequests     int64         `json:"total_requests"`
	AverageLatency    time.Duration `json:"average_latency"`
	MaxConcurrent     int           `json:"max_concurrent"`
}

// NewAsyncParallelAgent 创建异步并行Agent
func NewAsyncParallelAgent(name string, joinType JoinType, timeout time.Duration) *AsyncParallelAgent {
	if timeout == 0 {
		timeout = time.Minute * 5
	}

	return &AsyncParallelAgent{
		name:         name,
		joinType:     joinType,
		timeout:      timeout,
		pendingQueue: make(chan *AsyncBranch, 100),
		branchesMap:  make(map[string]*AsyncBranch),
		metrics:      &AsyncMetrics{},
	}
}

// AddBranch 动态添加分支
func (a *AsyncParallelAgent) AddBranch(branch *AsyncBranch) {
	a.mu.Lock()
	defer a.mu.Unlock()

	branch.Status = AsyncBranchStatusPending
	branch.CreatedAt = time.Now()

	a.branchesMap[branch.ID] = branch

	// 非阻塞地添加到队列
	select {
	case a.pendingQueue <- branch:
	default:
		// 队列满，直接执行
		go a.executeAsyncBranch(context.Background(), nil, branch)
	}
}

// Execute 执行异步并行工作流
func (a *AsyncParallelAgent) Execute(ctx context.Context, message types.Message, yield func(*session.Event, error) bool) error {
	// 发送开始事件
	yield(&session.Event{
		ID:        generateEventID(),
		Timestamp: time.Now(),
		AgentID:   a.name,
		Author:    "system",
		Content:   types.Message{Content: "Starting async parallel workflow"},
		Metadata: map[string]interface{}{
			"workflow_type": "async_parallel",
		},
	}, nil)

	// 启动队列处理器
	go a.processQueue(ctx, yield)

	// 启动主执行循环
	a.mu.RLock()
	totalBranches := len(a.branchesMap)
	a.mu.RUnlock()

	completedBranches := a.waitForAsyncCompletion(ctx, totalBranches)

	// 发送完成事件
	yield(&session.Event{
		ID:        generateEventID(),
		Timestamp: time.Now(),
		AgentID:   a.name,
		Author:    "system",
		Content:   types.Message{Content: fmt.Sprintf("Async parallel workflow completed with %d branches", len(completedBranches))},
		Metadata: map[string]interface{}{
			"workflow_type":      "async_parallel",
			"completed_branches": len(completedBranches),
		},
	}, nil)

	return nil
}

// processQueue 处理队列中的分支
func (a *AsyncParallelAgent) processQueue(ctx context.Context, yield func(*session.Event, error) bool) {
	for {
		select {
		case branch := <-a.pendingQueue:
			a.mu.Lock()
			a.branchesMap[branch.ID] = branch
			a.mu.Unlock()

			go a.executeAsyncBranch(ctx, yield, branch)

		case <-ctx.Done():
			return
		}
	}
}

// executeAsyncBranch 执行异步分支
func (a *AsyncParallelAgent) executeAsyncBranch(ctx context.Context, yield func(*session.Event, error) bool, branch *AsyncBranch) {
	a.mu.Lock()
	branch.Status = AsyncBranchStatusRunning
	branch.StartedAt = time.Now()
	a.mu.Unlock()

	// 模拟分支执行
	time.Sleep(time.Duration(100) * time.Millisecond)

	a.mu.Lock()
	branch.Status = AsyncBranchStatusCompleted
	branch.UpdatedAt = time.Now()
	branch.Result = fmt.Sprintf("Async branch %s completed", branch.Name)
	a.mu.Unlock()
}

// waitForAsyncCompletion 等待异步完成
func (a *AsyncParallelAgent) waitForAsyncCompletion(ctx context.Context, expectedCount int) []*AsyncBranch {
	var completedBranches []*AsyncBranch

	for {
		a.mu.RLock()
		completedCount := 0
		for _, branch := range a.branchesMap {
			if branch.Status == AsyncBranchStatusCompleted {
				completedCount++
			}
		}

		if completedCount >= expectedCount {
			for _, branch := range a.branchesMap {
				if branch.Status == AsyncBranchStatusCompleted {
					completedBranches = append(completedBranches, branch)
				}
			}
			a.mu.RUnlock()
			break
		}
		a.mu.RUnlock()

		select {
		case <-ctx.Done():
			return completedBranches
		case <-time.After(time.Millisecond * 100):
			// 继续等待
		}
	}

	return completedBranches
}

// GetAsyncMetrics 获取异步指标
func (a *AsyncParallelAgent) GetAsyncMetrics() *AsyncMetrics {
	a.mu.RLock()
	defer a.mu.RUnlock()

	activeCount := 0
	completedCount := 0

	for _, branch := range a.branchesMap {
		switch branch.Status {
		case AsyncBranchStatusRunning, AsyncBranchStatusPending:
			activeCount++
		case AsyncBranchStatusCompleted:
			completedCount++
		}
	}

	a.metrics.ActiveBranches = activeCount
	a.metrics.CompletedBranches = completedCount

	return a.metrics
}
