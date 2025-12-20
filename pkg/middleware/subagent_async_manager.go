package middleware

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/tools/builtin"
)

// goroutineSubagentManager implements builtin.SubagentManager using in-memory goroutines.
// It is used when async execution is enabled without process isolation so that tests
// and local runs don't rely on spawning external binaries.
type goroutineSubagentManager struct {
	middleware *SubAgentMiddleware

	mu        sync.RWMutex
	instances map[string]*builtin.SubagentInstance
	cancels   map[string]context.CancelFunc
}

func newGoroutineSubagentManager(mw *SubAgentMiddleware) *goroutineSubagentManager {
	return &goroutineSubagentManager{
		middleware: mw,
		instances:  make(map[string]*builtin.SubagentInstance),
		cancels:    make(map[string]context.CancelFunc),
	}
}

func (gm *goroutineSubagentManager) StartSubagent(ctx context.Context, config *builtin.SubagentConfig) (*builtin.SubagentInstance, error) {
	if config == nil {
		return nil, errors.New("subagent config cannot be nil")
	}

	subagent, err := gm.middleware.GetSubAgent(config.Type)
	if err != nil {
		return nil, err
	}

	taskID := config.ID
	if taskID == "" {
		taskID = fmt.Sprintf("subagent_%d", time.Now().UnixNano())
	}

	configCopy := *config
	configCopy.ID = taskID

	instance := &builtin.SubagentInstance{
		ID:         taskID,
		Type:       configCopy.Type,
		Status:     "running",
		Config:     &configCopy,
		StartTime:  time.Now(),
		Metadata:   make(map[string]string),
		LastUpdate: time.Now(),
		Command:    "goroutine",
	}
	maps.Copy(instance.Metadata, configCopy.Metadata)

	var execCtx context.Context
	var cancel context.CancelFunc
	if ctx == nil {
		ctx = context.Background()
	}
	if configCopy.Timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, configCopy.Timeout)
	} else {
		execCtx, cancel = context.WithCancel(ctx)
	}

	gm.mu.Lock()
	if _, exists := gm.instances[taskID]; exists {
		gm.mu.Unlock()
		cancel()
		return nil, fmt.Errorf("subagent already exists: %s", taskID)
	}
	gm.instances[taskID] = instance
	gm.cancels[taskID] = cancel
	gm.mu.Unlock()

	go gm.runSubagent(execCtx, taskID, subagent, &configCopy)
	return instance, nil
}

func (gm *goroutineSubagentManager) ResumeSubagent(taskID string) (*builtin.SubagentInstance, error) {
	gm.mu.RLock()
	instance, exists := gm.instances[taskID]
	gm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("subagent not found: %s", taskID)
	}

	if instance.Status == "running" {
		return nil, fmt.Errorf("subagent cannot be resumed, current status: %s", instance.Status)
	}

	newConfig := *instance.Config
	newConfig.ID = ""

	return gm.StartSubagent(context.Background(), &newConfig)
}

func (gm *goroutineSubagentManager) GetSubagent(taskID string) (*builtin.SubagentInstance, error) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	instance, exists := gm.instances[taskID]
	if !exists {
		return nil, fmt.Errorf("subagent not found: %s", taskID)
	}

	if instance.Status == "running" {
		instance.Duration = time.Since(instance.StartTime)
		instance.LastUpdate = time.Now()
	}

	return instance, nil
}

func (gm *goroutineSubagentManager) StopSubagent(taskID string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	instance, exists := gm.instances[taskID]
	if !exists {
		return fmt.Errorf("subagent not found: %s", taskID)
	}

	if instance.Status != "running" {
		return fmt.Errorf("subagent is not running, current status: %s", instance.Status)
	}

	if cancel, ok := gm.cancels[taskID]; ok {
		cancel()
	}

	now := time.Now()
	instance.Status = "stopped"
	instance.EndTime = &now
	instance.Duration = now.Sub(instance.StartTime)
	instance.LastUpdate = now
	instance.Error = "subagent stopped by request"

	return nil
}

func (gm *goroutineSubagentManager) ListSubagents() ([]*builtin.SubagentInstance, error) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	list := make([]*builtin.SubagentInstance, 0, len(gm.instances))
	for _, instance := range gm.instances {
		if instance.Status == "running" {
			instance.Duration = time.Since(instance.StartTime)
			instance.LastUpdate = time.Now()
		}
		list = append(list, instance)
	}

	return list, nil
}

func (gm *goroutineSubagentManager) GetSubagentOutput(taskID string) (string, error) {
	instance, err := gm.GetSubagent(taskID)
	if err != nil {
		return "", err
	}
	return instance.Output, nil
}

func (gm *goroutineSubagentManager) CleanupSubagent(taskID string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if cancel, ok := gm.cancels[taskID]; ok {
		cancel()
		delete(gm.cancels, taskID)
	}

	if _, exists := gm.instances[taskID]; !exists {
		return fmt.Errorf("subagent not found: %s", taskID)
	}
	delete(gm.instances, taskID)
	return nil
}

func (gm *goroutineSubagentManager) runSubagent(ctx context.Context, taskID string, subagent SubAgent, config *builtin.SubagentConfig) {
	defer func() {
		gm.mu.Lock()
		delete(gm.cancels, taskID)
		gm.mu.Unlock()
	}()

	result, err := subagent.Execute(ctx, config.Prompt, config.ParentContext)

	gm.mu.Lock()
	defer gm.mu.Unlock()

	instance, exists := gm.instances[taskID]
	if !exists {
		return
	}

	now := time.Now()
	instance.LastUpdate = now
	instance.Duration = now.Sub(instance.StartTime)
	instance.EndTime = &now

	if instance.Status == "stopped" {
		return
	}

	if ctx.Err() == context.DeadlineExceeded {
		instance.Status = "failed"
		instance.Error = "subagent timeout"
		return
	}

	if err != nil {
		instance.Status = "failed"
		instance.Error = err.Error()
		return
	}

	instance.Status = "completed"
	instance.Output = result
	instance.ExitCode = 0
}
