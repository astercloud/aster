package stars

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// EventType 事件类型
type EventType string

const (
	// 状态事件
	EventTypeStateCreated EventType = "state_created"
	EventTypeStateUpdated EventType = "state_updated"
	EventTypeStateDeleted EventType = "state_deleted"
	EventTypeStateRead    EventType = "state_read"

	// 同步事件
	EventTypeSyncStarted   EventType = "sync_started"
	EventTypeSyncCompleted EventType = "sync_completed"
	EventTypeSyncFailed    EventType = "sync_failed"
	EventTypeSyncConflict  EventType = "sync_conflict"

	// 冲突事件
	EventTypeConflictDetected EventType = "conflict_detected"
	EventTypeConflictResolved EventType = "conflict_resolved"

	// Agent事件
	EventTypeAgentConnected    EventType = "agent_connected"
	EventTypeAgentDisconnected EventType = "agent_disconnected"
	EventTypeAgentHeartbeat    EventType = "agent_heartbeat"

	// 系统事件
	EventTypeSystemStarted  EventType = "system_started"
	EventTypeSystemShutdown EventType = "system_shutdown"
	EventTypeConfigChanged  EventType = "config_changed"
)

// StateEvent 状态事件
type StateEvent struct {
	Type      EventType              `json:"type"`
	StateID   string                 `json:"state_id"`
	State     *SharedState           `json:"state,omitempty"`
	AgentID   string                 `json:"agent_id,omitempty"`
	Operation *PendingOperation      `json:"operation,omitempty"`
	Conflict  *ConflictInfo          `json:"conflict,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// ConflictInfo 冲突信息
type ConflictInfo struct {
	ConflictID  string                  `json:"conflict_id"`
	Type        ConflictType            `json:"type"`
	Description string                  `json:"description"`
	States      map[string]*SharedState `json:"states"`
	Operations  []PendingOperation      `json:"operations"`
	Resolution  ConflictResolution      `json:"resolution"`
	ResolvedAt  time.Time               `json:"resolved_at,omitempty"`
	ResolvedBy  string                  `json:"resolved_by,omitempty"`
	Metadata    map[string]interface{}  `json:"metadata"`
}

// ConflictType 冲突类型
type ConflictType string

const (
	ConflictTypeWriteWrite ConflictType = "write_write" // 写-写冲突
	ConflictTypeReadWrite  ConflictType = "read_write"  // 读-写冲突
	ConflictTypeVersion    ConflictType = "version"     // 版本冲突
	ConflictTypeSchema     ConflictType = "schema"      // 模式冲突
	ConflictTypeCustom     ConflictType = "custom"      // 自定义冲突
)

// EventBus 事件总线接口
type EventBus interface {
	Subscribe(eventType EventType, handler EventHandler) error
	Unsubscribe(eventType EventType, handler EventHandler) error
	Publish(event StateEvent) error
	PublishAsync(event StateEvent) error
	Close() error
}

// EventHandler 事件处理器
type EventHandler interface {
	Handle(ctx context.Context, event StateEvent) error
}

// InMemoryEventBus 内存事件总线
type InMemoryEventBus struct {
	subscribers map[EventType][]EventHandler
	mu          sync.RWMutex
	buffer      chan StateEvent
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	bufferSize  int
}

// NewInMemoryEventBus 创建内存事件总线
func NewInMemoryEventBus(bufferSize int) *InMemoryEventBus {
	ctx, cancel := context.WithCancel(context.Background())

	bus := &InMemoryEventBus{
		subscribers: make(map[EventType][]EventHandler),
		buffer:      make(chan StateEvent, bufferSize),
		ctx:         ctx,
		cancel:      cancel,
		bufferSize:  bufferSize,
	}

	// 启动事件处理协程
	bus.wg.Add(1)
	go bus.eventWorker()

	return bus
}

// Subscribe 订阅事件
func (bus *InMemoryEventBus) Subscribe(eventType EventType, handler EventHandler) error {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if bus.subscribers[eventType] == nil {
		bus.subscribers[eventType] = []EventHandler{}
	}

	bus.subscribers[eventType] = append(bus.subscribers[eventType], handler)
	return nil
}

// Unsubscribe 取消订阅事件
func (bus *InMemoryEventBus) Unsubscribe(eventType EventType, handler EventHandler) error {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	handlers, exists := bus.subscribers[eventType]
	if !exists {
		return nil
	}

	for i, h := range handlers {
		if h == handler {
			bus.subscribers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}

	return nil
}

// Publish 同步发布事件
func (bus *InMemoryEventBus) Publish(event StateEvent) error {
	select {
	case bus.buffer <- event:
		return nil
	case <-bus.ctx.Done():
		return bus.ctx.Err()
	}
}

// PublishAsync 异步发布事件
func (bus *InMemoryEventBus) PublishAsync(event StateEvent) error {
	go func() {
		bus.buffer <- event
	}()
	return nil
}

// eventWorker 事件处理工作协程
func (bus *InMemoryEventBus) eventWorker() {
	defer bus.wg.Done()

	for {
		select {
		case event := <-bus.buffer:
			bus.processEvent(event)
		case <-bus.ctx.Done():
			return
		}
	}
}

// processEvent 处理事件
func (bus *InMemoryEventBus) processEvent(event StateEvent) {
	bus.mu.RLock()
	handlers, exists := bus.subscribers[event.Type]
	bus.mu.RUnlock()

	if !exists {
		return
	}

	for _, handler := range handlers {
		go func(h EventHandler) {
			ctx, cancel := context.WithTimeout(bus.ctx, time.Second*30)
			defer cancel()

			_ = h.Handle(ctx, event) // 忽略错误，不中断其他处理器的执行
		}(handler)
	}
}

// Close 关闭事件总线
func (bus *InMemoryEventBus) Close() error {
	bus.cancel()
	bus.wg.Wait()
	close(bus.buffer)
	return nil
}

// ConflictDetector 冲突检测器
type ConflictDetector struct {
	eventBus EventBus
	rules    []ConflictRule
}

// ConflictRule 冲突规则
type ConflictRule interface {
	DetectConflict(event StateEvent, currentState *SharedState) (*ConflictInfo, error)
	ResolveConflict(conflict *ConflictInfo, strategy ConflictResolution) (*SharedState, error)
}

// NewConflictDetector 创建冲突检测器
func NewConflictDetector(eventBus EventBus) *ConflictDetector {
	return &ConflictDetector{
		eventBus: eventBus,
		rules:    []ConflictRule{},
	}
}

// AddRule 添加冲突规则
func (cd *ConflictDetector) AddRule(rule ConflictRule) {
	cd.rules = append(cd.rules, rule)
}

// DetectConflict 检测冲突
func (cd *ConflictDetector) DetectConflict(event StateEvent, currentState *SharedState) (*ConflictInfo, error) {
	for _, rule := range cd.rules {
		conflict, err := rule.DetectConflict(event, currentState)
		if err != nil {
			return nil, err
		}
		if conflict != nil {
			return conflict, nil
		}
	}
	return nil, nil
}

// ResolveConflict 解决冲突
func (cd *ConflictDetector) ResolveConflict(conflict *ConflictInfo, strategy ConflictResolution) (*SharedState, error) {
	for _, rule := range cd.rules {
		resolved, err := rule.ResolveConflict(conflict, strategy)
		if err != nil {
			return nil, err
		}
		if resolved != nil {
			// 发布冲突解决事件
			if cd.eventBus != nil {
				_ = cd.eventBus.Publish(StateEvent{
					Type:      EventTypeConflictResolved,
					StateID:   conflict.States["current"].ID,
					Conflict:  conflict,
					Timestamp: time.Now(),
				})
			}
			return resolved, nil
		}
	}
	return nil, fmt.Errorf("no rule could resolve the conflict")
}

// DefaultConflictRule 默认冲突规则
type DefaultConflictRule struct{}

// DetectConflict 检测冲突
func (dcr *DefaultConflictRule) DetectConflict(event StateEvent, currentState *SharedState) (*ConflictInfo, error) {
	// 检测写-写冲突
	if event.Type == EventTypeStateUpdated && event.Operation != nil {
		// 这里可以实现更复杂的冲突检测逻辑
		// 例如：检查版本号、时间戳等
		return nil, nil
	}
	return nil, nil
}

// ResolveConflict 解决冲突
func (dcr *DefaultConflictRule) ResolveConflict(conflict *ConflictInfo, strategy ConflictResolution) (*SharedState, error) {
	switch strategy {
	case ConflictResolutionLastWrite:
		// 最后写入优先策略
		if lastState := dcr.findLastWriteState(conflict.States); lastState != nil {
			return lastState, nil
		}
	case ConflictResolutionMerge:
		// 合并策略
		return dcr.mergeStates(conflict.States), nil
	case ConflictResolutionReject:
		// 拒绝策略
		return nil, fmt.Errorf("conflict rejected")
	}
	return nil, fmt.Errorf("unsupported conflict resolution strategy")
}

// findLastWriteState 找到最后写入的状态
func (dcr *DefaultConflictRule) findLastWriteState(states map[string]*SharedState) *SharedState {
	var lastState *SharedState
	var lastTime time.Time

	for _, state := range states {
		if state.Timestamp.After(lastTime) {
			lastState = state
			lastTime = state.Timestamp
		}
	}
	return lastState
}

// mergeStates 合并状态
func (dcr *DefaultConflictRule) mergeStates(states map[string]*SharedState) *SharedState {
	// 简单的合并策略：合并所有键值对
	merged := &SharedState{
		Data: make(map[string]interface{}),
	}

	for _, state := range states {
		for k, v := range state.Data {
			merged.Data[k] = v
		}
	}

	return merged
}
