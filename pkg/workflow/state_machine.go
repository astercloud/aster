package workflow

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sync"
	"time"
)

// StateMachine 状态机接口
type StateMachine interface {
	// 状态管理
	AddState(name string, handlers ...StateHandler) error
	AddTransition(from, to string, condition TransitionCondition) error
	GetState(name string) (State, error)
	GetCurrentState() string

	// 状态转移
	Transition(ctx context.Context, event string) (string, error)

	// 历史记录
	GetHistory() []*StateTransition
	GetLastTransition() *StateTransition

	// 持久化
	SaveState(ctx context.Context, stateData map[string]any) error
	LoadState(ctx context.Context) (map[string]any, error)
}

// State 状态接口
type State interface {
	Name() string
	Description() string
	Handlers() []StateHandler
}

// StateHandler 状态处理器
type StateHandler interface {
	OnEnter(ctx context.Context) error
	OnExit(ctx context.Context) error
}

// TransitionCondition 转移条件函数
type TransitionCondition func(ctx context.Context) (bool, error)

// StateImpl 状态实现
type StateImpl struct {
	name        string
	description string
	handlers    []StateHandler
}

func NewState(name string, handlers ...StateHandler) *StateImpl {
	return &StateImpl{
		name:     name,
		handlers: handlers,
	}
}

func (s *StateImpl) Name() string             { return s.name }
func (s *StateImpl) Description() string      { return s.description }
func (s *StateImpl) Handlers() []StateHandler { return s.handlers }

func (s *StateImpl) WithDescription(desc string) *StateImpl {
	s.description = desc
	return s
}

// Transition 转移对象
type Transition struct {
	from      string
	to        string
	condition TransitionCondition
}

// StateTransition 状态转移记录
type StateTransition struct {
	From      string
	To        string
	Timestamp time.Time
	Duration  float64
	Metadata  map[string]any
}

// StateMachineImpl 状态机实现
type StateMachineImpl struct {
	name            string
	states          map[string]State
	transitions     map[string][]*Transition
	currentState    string
	history         []*StateTransition
	stateData       map[string]any
	mu              sync.RWMutex
	persistentStore StatePersistentStore
}

// StatePersistentStore 状态持久化接口
type StatePersistentStore interface {
	Save(ctx context.Context, stateID string, data map[string]any) error
	Load(ctx context.Context, stateID string) (map[string]any, error)
}

// NewStateMachine 创建状态机
func NewStateMachine(name string, initialState string, store StatePersistentStore) *StateMachineImpl {
	return &StateMachineImpl{
		name:            name,
		states:          make(map[string]State),
		transitions:     make(map[string][]*Transition),
		currentState:    initialState,
		history:         make([]*StateTransition, 0),
		stateData:       make(map[string]any),
		persistentStore: store,
	}
}

// AddState 添加状态
func (sm *StateMachineImpl) AddState(name string, handlers ...StateHandler) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.states[name]; exists {
		return fmt.Errorf("state already exists: %s", name)
	}

	state := NewState(name, handlers...)
	sm.states[name] = state
	sm.transitions[name] = make([]*Transition, 0)

	return nil
}

// AddTransition 添加转移
func (sm *StateMachineImpl) AddTransition(from, to string, condition TransitionCondition) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.states[from]; !exists {
		return fmt.Errorf("source state not found: %s", from)
	}

	if _, exists := sm.states[to]; !exists {
		return fmt.Errorf("target state not found: %s", to)
	}

	transition := &Transition{
		from:      from,
		to:        to,
		condition: condition,
	}

	sm.transitions[from] = append(sm.transitions[from], transition)
	return nil
}

// GetState 获取状态
func (sm *StateMachineImpl) GetState(name string) (State, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	state, exists := sm.states[name]
	if !exists {
		return nil, fmt.Errorf("state not found: %s", name)
	}

	return state, nil
}

// GetCurrentState 获取当前状态
func (sm *StateMachineImpl) GetCurrentState() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.currentState
}

// Transition 执行转移
func (sm *StateMachineImpl) Transition(ctx context.Context, event string) (string, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	startTime := time.Now()
	currentState := sm.currentState

	// 获取当前状态的转移列表
	transitions, exists := sm.transitions[currentState]
	if !exists {
		return currentState, fmt.Errorf("no transitions from state: %s", currentState)
	}

	// 尝试找到满足条件的转移
	for _, transition := range transitions {
		if transition.condition == nil {
			// 无条件转移，直接转移
			return sm.performTransition(ctx, currentState, transition.to, startTime)
		}

		// 检查条件
		passed, err := transition.condition(ctx)
		if err != nil {
			continue
		}

		if passed {
			return sm.performTransition(ctx, currentState, transition.to, startTime)
		}
	}

	return currentState, fmt.Errorf("no valid transition from state: %s for event: %s", currentState, event)
}

// performTransition 执行转移的内部方法
func (sm *StateMachineImpl) performTransition(ctx context.Context, from, to string, startTime time.Time) (string, error) {
	// 执行当前状态的 OnExit 处理器
	if state, exists := sm.states[from]; exists {
		for _, handler := range state.Handlers() {
			if exitHandler, ok := handler.(interface{ OnExit(context.Context) error }); ok {
				if err := exitHandler.OnExit(ctx); err != nil {
					return from, fmt.Errorf("exit handler failed for state %s: %w", from, err)
				}
			}
		}
	}

	// 执行目标状态的 OnEnter 处理器
	if state, exists := sm.states[to]; exists {
		for _, handler := range state.Handlers() {
			if enterHandler, ok := handler.(interface{ OnEnter(context.Context) error }); ok {
				if err := enterHandler.OnEnter(ctx); err != nil {
					return from, fmt.Errorf("enter handler failed for state %s: %w", to, err)
				}
			}
		}
	}

	// 记录转移历史
	transition := &StateTransition{
		From:      from,
		To:        to,
		Timestamp: startTime,
		Duration:  time.Since(startTime).Seconds(),
		Metadata:  make(map[string]any),
	}
	sm.history = append(sm.history, transition)

	// 更新当前状态
	sm.currentState = to

	return to, nil
}

// GetHistory 获取转移历史
func (sm *StateMachineImpl) GetHistory() []*StateTransition {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	history := make([]*StateTransition, len(sm.history))
	copy(history, sm.history)
	return history
}

// GetLastTransition 获取最后的转移记录
func (sm *StateMachineImpl) GetLastTransition() *StateTransition {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if len(sm.history) == 0 {
		return nil
	}

	return sm.history[len(sm.history)-1]
}

// SaveState 保存状态
func (sm *StateMachineImpl) SaveState(ctx context.Context, stateData map[string]any) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.persistentStore == nil {
		return errors.New("persistent store not configured")
	}

	// 合并当前状态数据
	data := make(map[string]any)
	maps.Copy(data, sm.stateData)
	maps.Copy(data, stateData)

	data["_current_state"] = sm.currentState
	data["_state_machine_name"] = sm.name

	return sm.persistentStore.Save(ctx, sm.name, data)
}

// LoadState 加载状态
func (sm *StateMachineImpl) LoadState(ctx context.Context) (map[string]any, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.persistentStore == nil {
		return nil, errors.New("persistent store not configured")
	}

	data, err := sm.persistentStore.Load(ctx, sm.name)
	if err != nil {
		return nil, err
	}

	// 恢复状态
	if currentState, exists := data["_current_state"]; exists {
		if stateStr, ok := currentState.(string); ok {
			sm.currentState = stateStr
		}
	}

	sm.stateData = data
	return data, nil
}

// ===== 状态处理器实现 =====

// SimpleStateHandler 简单状态处理器
type SimpleStateHandler struct {
	onEnterFn func(context.Context) error
	onExitFn  func(context.Context) error
}

func NewSimpleStateHandler(onEnter, onExit func(context.Context) error) *SimpleStateHandler {
	return &SimpleStateHandler{
		onEnterFn: onEnter,
		onExitFn:  onExit,
	}
}

func (h *SimpleStateHandler) OnEnter(ctx context.Context) error {
	if h.onEnterFn != nil {
		return h.onEnterFn(ctx)
	}
	return nil
}

func (h *SimpleStateHandler) OnExit(ctx context.Context) error {
	if h.onExitFn != nil {
		return h.onExitFn(ctx)
	}
	return nil
}

// ===== 转移条件构建器 =====

// NewCondition 创建无条件转移（总是转移）
func NewCondition() TransitionCondition {
	return func(ctx context.Context) (bool, error) {
		return true, nil
	}
}

// NewTimeoutCondition 创建超时转移条件
func NewTimeoutCondition(duration time.Duration) TransitionCondition {
	startTime := time.Now()
	return func(ctx context.Context) (bool, error) {
		return time.Since(startTime) >= duration, nil
	}
}

// NewContextCondition 创建基于上下文的转移条件
func NewContextCondition(fn func(context.Context) (bool, error)) TransitionCondition {
	return fn
}
