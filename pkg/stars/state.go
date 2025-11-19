package stars

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// StateManager 状态管理器
type StateManager struct {
	// 基础存储
	states map[string]*SharedState
	mu     sync.RWMutex

	// 事件系统
	eventBus EventBus

	// 持久化
	persistence PersistenceLayer

	// 配置
	config *StateManagerConfig

	// 指标
	metrics *StateMetrics
}

// StateManagerConfig 状态管理器配置
type StateManagerConfig struct {
	// 同步配置
	SyncInterval time.Duration `json:"sync_interval"`
	MaxRetries   int           `json:"max_retries"`
	RetryDelay   time.Duration `json:"retry_delay"`

	// 持久化配置
	EnablePersistence bool            `json:"enable_persistence"`
	PersistenceType   PersistenceType `json:"persistence_type"`
	FlushInterval     time.Duration   `json:"flush_interval"`

	// 一致性配置
	ConsistencyLevel   ConsistencyLevel   `json:"consistency_level"`
	ConflictResolution ConflictResolution `json:"conflict_resolution"`

	// 监控配置
	EnableMetrics bool `json:"enable_metrics"`
	EnableAudit   bool `json:"enable_audit"`
}

// ConsistencyLevel 一致性级别
type ConsistencyLevel string

const (
	ConsistencyLevelStrong   ConsistencyLevel = "strong"   // 强一致性
	ConsistencyLevelWeak     ConsistencyLevel = "weak"     // 弱一致性
	ConsistencyLevelEventual ConsistencyLevel = "eventual" // 最终一致性
)

// ConflictResolution 冲突解决策略
type ConflictResolution string

const (
	ConflictResolutionLastWrite ConflictResolution = "last_write" // 最后写入优先
	ConflictResolutionMerge     ConflictResolution = "merge"      // 合并冲突
	ConflictResolutionReject    ConflictResolution = "reject"     // 拒绝冲突
	ConflictResolutionCustom    ConflictResolution = "custom"     // 自定义策略
)

// PersistenceType 持久化类型
type PersistenceType string

const (
	PersistenceTypeMemory   PersistenceType = "memory"   // 内存持久化
	PersistenceTypeFile     PersistenceType = "file"     // 文件持久化
	PersistenceTypeDatabase PersistenceType = "database" // 数据库持久化
	PersistenceTypeRedis    PersistenceType = "redis"    // Redis持久化
)

// SharedState 共享状态
type SharedState struct {
	// 基本信息
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Type    StateType `json:"type"`
	Owners  []string  `json:"owners"`  // 状态拥有者(Agent列表)
	Readers []string  `json:"readers"` // 状态读取者

	// 状态数据
	Data      map[string]interface{} `json:"data"`
	Version   int64                  `json:"version"`
	Timestamp time.Time              `json:"timestamp"`

	// 元数据
	Metadata map[string]interface{} `json:"metadata"`
	Tags     []string               `json:"tags"`

	// 同步信息
	LastSync   time.Time          `json:"last_sync"`
	SyncStatus SyncStatus         `json:"sync_status"`
	PendingOps []PendingOperation `json:"pending_ops"`

	// 访问控制
	AccessPolicy AccessPolicy `json:"access_policy"`

	// 互斥锁
	mu sync.RWMutex
}

// StateType 状态类型
type StateType string

const (
	StateTypeGlobal   StateType = "global"   // 全局状态
	StateTypeSession  StateType = "session"  // 会话状态
	StateTypeWorkflow StateType = "workflow" // 工作流状态
	StateTypeTask     StateType = "task"     // 任务状态
	StateTypeAgent    StateType = "agent"    // Agent状态
	StateTypeResource StateType = "resource" // 资源状态
)

// SyncStatus 同步状态
type SyncStatus string

const (
	SyncStatusSynced   SyncStatus = "synced"   // 已同步
	SyncStatusPending  SyncStatus = "pending"  // 待同步
	SyncStatusConflict SyncStatus = "conflict" // 冲突
	SyncStatusError    SyncStatus = "error"    // 错误
	SyncStatusSyncing  SyncStatus = "syncing"  // 同步中
)

// PendingOperation 待处理操作
type PendingOperation struct {
	ID        string                 `json:"id"`
	Type      OperationType          `json:"type"`
	Key       string                 `json:"key"`
	Value     interface{}            `json:"value"`
	OldValue  interface{}            `json:"old_value"`
	Timestamp time.Time              `json:"timestamp"`
	AgentID   string                 `json:"agent_id"`
	Retry     int                    `json:"retry"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// OperationType 操作类型
type OperationType string

const (
	OperationTypeSet    OperationType = "set"    // 设置值
	OperationTypeDelete OperationType = "delete" // 删除值
	OperationTypeMerge  OperationType = "merge"  // 合并值
	OperationTypeCAS    OperationType = "cas"    // Compare-And-Set
)

// AccessPolicy 访问策略
type AccessPolicy struct {
	ReadAllow   []string `json:"read_allow"`   // 允许读取的Agent
	WriteAllow  []string `json:"write_allow"`  // 允许写入的Agent
	DeleteAllow []string `json:"delete_allow"` // 允许删除的Agent
	AdminAllow  []string `json:"admin_allow"`  // 允许管理的Agent
	Public      bool     `json:"public"`       // 是否公开
}

// StateMetrics 状态指标
type StateMetrics struct {
	TotalStates       int64         `json:"total_states"`
	ActiveStates      int64         `json:"active_states"`
	PendingOperations int64         `json:"pending_operations"`
	SyncErrors        int64         `json:"sync_errors"`
	Conflicts         int64         `json:"conflicts"`
	ReadOps           int64         `json:"read_ops"`
	WriteOps          int64         `json:"write_ops"`
	DeleteOps         int64         `json:"delete_ops"`
	AverageSyncTime   time.Duration `json:"average_sync_time"`
	LastSyncTime      time.Time     `json:"last_sync_time"`
}

// NewStateManager 创建状态管理器
func NewStateManager(config *StateManagerConfig, eventBus EventBus, persistence PersistenceLayer) *StateManager {
	if config == nil {
		config = &StateManagerConfig{
			SyncInterval:       time.Second * 5,
			MaxRetries:         3,
			RetryDelay:         time.Millisecond * 500,
			EnablePersistence:  false,
			ConsistencyLevel:   ConsistencyLevelEventual,
			ConflictResolution: ConflictResolutionLastWrite,
			EnableMetrics:      true,
			EnableAudit:        false,
		}
	}

	sm := &StateManager{
		states:      make(map[string]*SharedState),
		eventBus:    eventBus,
		persistence: persistence,
		config:      config,
		metrics:     &StateMetrics{},
	}

	// 启动同步协程
	if config.SyncInterval > 0 {
		go sm.startSyncWorker()
	}

	return sm
}

// CreateState 创建共享状态
func (sm *StateManager) CreateState(ctx context.Context, stateID, name string, stateType StateType, owners []string) (*SharedState, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.states[stateID]; exists {
		return nil, fmt.Errorf("state %s already exists", stateID)
	}

	state := &SharedState{
		ID:         stateID,
		Name:       name,
		Type:       stateType,
		Owners:     owners,
		Readers:    []string{},
		Data:       make(map[string]interface{}),
		Version:    1,
		Timestamp:  time.Now(),
		Metadata:   make(map[string]interface{}),
		Tags:       []string{},
		LastSync:   time.Now(),
		SyncStatus: SyncStatusSynced,
		PendingOps: []PendingOperation{},
		AccessPolicy: AccessPolicy{
			ReadAllow:   owners,
			WriteAllow:  owners,
			DeleteAllow: owners,
			AdminAllow:  owners,
			Public:      false,
		},
	}

	sm.states[stateID] = state
	sm.metrics.TotalStates++
	sm.metrics.ActiveStates++

	// 发送创建事件
	if sm.eventBus != nil {
		_ = sm.eventBus.Publish(StateEvent{
			Type:      EventTypeStateCreated,
			StateID:   stateID,
			State:     state,
			Timestamp: time.Now(),
		})
	}

	// 持久化
	if sm.persistence != nil && sm.config.EnablePersistence {
		_ = sm.persistence.SaveState(ctx, state)
	}

	return state, nil
}

// GetState 获取共享状态
func (sm *StateManager) GetState(stateID string) (*SharedState, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	state, exists := sm.states[stateID]
	if !exists {
		return nil, fmt.Errorf("state %s not found", stateID)
	}

	return state, nil
}

// UpdateState 更新共享状态
func (sm *StateManager) UpdateState(ctx context.Context, stateID string, agentID string, updates map[string]interface{}, metadata map[string]interface{}) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, exists := sm.states[stateID]
	if !exists {
		return fmt.Errorf("state %s not found", stateID)
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	// 权限检查
	if !sm.canWrite(state, agentID) {
		return fmt.Errorf("agent %s does not have write permission for state %s", agentID, stateID)
	}

	// 创建待处理操作
	op := PendingOperation{
		ID:        generateOperationID(),
		Type:      OperationTypeSet,
		Key:       "batch_update",
		Value:     updates,
		OldValue:  sm.copyStateData(state.Data),
		Timestamp: time.Now(),
		AgentID:   agentID,
		Retry:     0,
		Metadata:  metadata,
	}

	state.PendingOps = append(state.PendingOps, op)

	// 应用更新
	for key, value := range updates {
		state.Data[key] = value
	}

	state.Version++
	state.Timestamp = time.Now()
	state.SyncStatus = SyncStatusPending
	state.LastSync = time.Now()

	// 更新指标
	sm.metrics.WriteOps++
	sm.metrics.PendingOperations++

	// 发送更新事件
	if sm.eventBus != nil {
		_ = sm.eventBus.Publish(StateEvent{
			Type:      EventTypeStateUpdated,
			StateID:   stateID,
			State:     state,
			AgentID:   agentID,
			Operation: &op,
			Timestamp: time.Now(),
		})
	}

	return nil
}

// DeleteState 删除共享状态
func (sm *StateManager) DeleteState(ctx context.Context, stateID, agentID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, exists := sm.states[stateID]
	if !exists {
		return fmt.Errorf("state %s not found", stateID)
	}

	// 权限检查
	if !sm.canDelete(state, agentID) {
		return fmt.Errorf("agent %s does not have delete permission for state %s", agentID, stateID)
	}

	delete(sm.states, stateID)
	sm.metrics.ActiveStates--
	sm.metrics.DeleteOps++

	// 发送删除事件
	if sm.eventBus != nil {
		_ = sm.eventBus.Publish(StateEvent{
			Type:      EventTypeStateDeleted,
			StateID:   stateID,
			State:     state,
			AgentID:   agentID,
			Timestamp: time.Now(),
		})
	}

	// 持久化删除
	if sm.persistence != nil && sm.config.EnablePersistence {
		_ = sm.persistence.DeleteState(ctx, stateID)
	}

	return nil
}

// canWrite 检查写入权限
func (sm *StateManager) canWrite(state *SharedState, agentID string) bool {
	if state.AccessPolicy.Public {
		return true
	}

	for _, allowed := range state.AccessPolicy.WriteAllow {
		if allowed == agentID {
			return true
		}
	}

	return false
}

// canDelete 检查删除权限
func (sm *StateManager) canDelete(state *SharedState, agentID string) bool {
	for _, owner := range state.Owners {
		if owner == agentID {
			return true
		}
	}

	for _, allowed := range state.AccessPolicy.AdminAllow {
		if allowed == agentID {
			return true
		}
	}

	return false
}

// copyStateData 复制状态数据
func (sm *StateManager) copyStateData(data map[string]interface{}) map[string]interface{} {
	copy := make(map[string]interface{})
	for k, v := range data {
		copy[k] = v
	}
	return copy
}

// startSyncWorker 启动同步工作协程
func (sm *StateManager) startSyncWorker() {
	ticker := time.NewTicker(sm.config.SyncInterval)
	defer ticker.Stop()

	for range ticker.C {
		sm.syncPendingOperations()
	}
}

// syncPendingOperations 同步待处理操作
func (sm *StateManager) syncPendingOperations() {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, state := range sm.states {
		if state.SyncStatus == SyncStatusPending && len(state.PendingOps) > 0 {
			go sm.syncState(state)
		}
	}
}

// syncState 同步单个状态
func (sm *StateManager) syncState(state *SharedState) {
	state.mu.Lock()
	defer state.mu.Unlock()

	start := time.Now()
	state.SyncStatus = SyncStatusSyncing

	// 尝试持久化
	if sm.persistence != nil && sm.config.EnablePersistence {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		if err := sm.persistence.UpdateState(ctx, state); err != nil {
			state.SyncStatus = SyncStatusError
			sm.metrics.SyncErrors++
			return
		}
	}

	// 清理已完成的操作
	state.PendingOps = state.PendingOps[:0]
	state.SyncStatus = SyncStatusSynced
	state.LastSync = time.Now()

	// 更新指标
	_ = time.Since(start) // 可用于记录同步耗时
	sm.metrics.PendingOperations = int64(len(state.PendingOps))
	sm.metrics.LastSyncTime = state.LastSync
}

// GetMetrics 获取状态管理指标
func (sm *StateManager) GetMetrics() *StateMetrics {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// 复制指标
	metrics := *sm.metrics
	metrics.ActiveStates = int64(len(sm.states))

	return &metrics
}

// generateOperationID 生成操作ID
func generateOperationID() string {
	return fmt.Sprintf("op_%d", time.Now().UnixNano())
}
