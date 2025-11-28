package a2a

import (
	"fmt"
	"sync"
)

// TaskStore Task 存储接口
type TaskStore interface {
	// Load 加载任务
	Load(agentID, taskID string) (*Task, error)

	// Save 保存任务
	Save(agentID string, task *Task) error

	// Delete 删除任务
	Delete(agentID, taskID string) error

	// List 列出 Agent 的所有任务
	List(agentID string) ([]*Task, error)

	// AddCancellation 添加取消信号
	AddCancellation(taskID string)

	// RemoveCancellation 移除取消信号
	RemoveCancellation(taskID string)

	// IsCanceled 检查是否已取消
	IsCanceled(taskID string) bool
}

// InMemoryTaskStore 内存任务存储
// 使用 Map 存储，支持并发访问
type InMemoryTaskStore struct {
	mu                  sync.RWMutex
	tasks               map[string]*Task // key: "agentID-taskID"
	activeCancellations map[string]bool  // key: taskID
}

// NewInMemoryTaskStore 创建内存任务存储
func NewInMemoryTaskStore() *InMemoryTaskStore {
	return &InMemoryTaskStore{
		tasks:               make(map[string]*Task),
		activeCancellations: make(map[string]bool),
	}
}

// Load 加载任务
func (s *InMemoryTaskStore) Load(agentID, taskID string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := makeKey(agentID, taskID)
	task, exists := s.tasks[key]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	// 返回副本，防止外部修改
	return copyTask(task), nil
}

// Save 保存任务
func (s *InMemoryTaskStore) Save(agentID string, task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := makeKey(agentID, task.ID)
	// 存储副本，防止内部修改
	s.tasks[key] = copyTask(task)

	return nil
}

// Delete 删除任务
func (s *InMemoryTaskStore) Delete(agentID, taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := makeKey(agentID, taskID)
	delete(s.tasks, key)

	return nil
}

// List 列出 Agent 的所有任务
func (s *InMemoryTaskStore) List(agentID string) ([]*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prefix := agentID + "-"
	tasks := make([]*Task, 0)

	for key, task := range s.tasks {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			tasks = append(tasks, copyTask(task))
		}
	}

	return tasks, nil
}

// AddCancellation 添加取消信号
func (s *InMemoryTaskStore) AddCancellation(taskID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.activeCancellations[taskID] = true
}

// RemoveCancellation 移除取消信号
func (s *InMemoryTaskStore) RemoveCancellation(taskID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.activeCancellations, taskID)
}

// IsCanceled 检查是否已取消
func (s *InMemoryTaskStore) IsCanceled(taskID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.activeCancellations[taskID]
}

// makeKey 生成存储键
func makeKey(agentID, taskID string) string {
	return agentID + "-" + taskID
}

// copyTask 深拷贝任务
func copyTask(task *Task) *Task {
	if task == nil {
		return nil
	}

	// 拷贝基本字段
	copied := &Task{
		ID:        task.ID,
		ContextID: task.ContextID,
		Status: TaskStatus{
			State:     task.Status.State,
			Timestamp: task.Status.Timestamp,
		},
		Kind: task.Kind,
	}

	// 拷贝 Status.Message
	if task.Status.Message != nil {
		msg := *task.Status.Message
		copied.Status.Message = &msg
	}

	// 拷贝 History
	if task.History != nil {
		copied.History = make([]Message, len(task.History))
		copy(copied.History, task.History)
	}

	// 拷贝 Artifacts
	if task.Artifacts != nil {
		copied.Artifacts = make([]Artifact, len(task.Artifacts))
		copy(copied.Artifacts, task.Artifacts)
	}

	// 拷贝 Metadata
	if task.Metadata != nil {
		copied.Metadata = make(Metadata)
		for k, v := range task.Metadata {
			copied.Metadata[k] = v
		}
	}

	return copied
}
