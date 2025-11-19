package stars

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// StateSyncManager 状态同步管理器
type StateSyncManager struct {
	// 核心组件
	stateManager *StateManager
	eventBus     EventBus

	// 同步配置
	config *SyncConfig

	// 连接的Agent
	agents map[string]*AgentConnection
	mu     sync.RWMutex

	// 同步任务
	syncTasks map[string]*SyncTask
	taskMutex sync.Mutex

	// 指标
	metrics *SyncMetrics
}

// SyncConfig 同步配置
type SyncConfig struct {
	// 基础配置
	SyncInterval time.Duration `json:"sync_interval"`
	MaxRetries   int           `json:"max_retries"`
	RetryDelay   time.Duration `json:"retry_delay"`
	Timeout      time.Duration `json:"timeout"`

	// 批处理配置
	BatchSize     int           `json:"batch_size"`
	FlushInterval time.Duration `json:"flush_interval"`

	// 一致性配置
	ConsistencyLevel ConsistencyLevel   `json:"consistency_level"`
	ConflictStrategy ConflictResolution `json:"conflict_strategy"`

	// 压缩配置
	Compression     bool   `json:"compression"`
	CompressionType string `json:"compression_type"`

	// 安全配置
	Encryption    bool   `json:"encryption"`
	EncryptionKey string `json:"encryption_key"`

	// 性能配置
	MaxConcurrentSyncs int `json:"max_concurrent_syncs"`
	QueueBufferSize    int `json:"queue_buffer_size"`
}

// AgentConnection Agent连接
type AgentConnection struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Address          string                 `json:"address"`
	SubscribedStates []string               `json:"subscribed_states"`
	ConnectionType   ConnectionType         `json:"connection_type"`
	Status           ConnectionStatus       `json:"status"`
	LastHeartbeat    time.Time              `json:"last_heartbeat"`
	Metadata         map[string]interface{} `json:"metadata"`

	// 同步状态
	SyncStatus     map[string]SyncStatus `json:"sync_status"`
	PendingUpdates []StateUpdate         `json:"pending_updates"`
	ConflictCount  int64                 `json:"conflict_count"`

	// 通道
	updateChan chan StateUpdate
	mu         sync.RWMutex
}

// ConnectionType 连接类型
type ConnectionType string

const (
	ConnectionTypePush          ConnectionType = "push"          // 推送模式
	ConnectionTypePull          ConnectionType = "pull"          // 拉取模式
	ConnectionTypeBidirectional ConnectionType = "bidirectional" // 双向模式
)

// ConnectionStatus 连接状态
type ConnectionStatus string

const (
	ConnectionStatusConnected    ConnectionStatus = "connected"    // 已连接
	ConnectionStatusDisconnected ConnectionStatus = "disconnected" // 已断开
	ConnectionStatusConnecting   ConnectionStatus = "connecting"   // 连接中
	ConnectionStatusError        ConnectionStatus = "error"        // 错误
	ConnectionStatusReconnecting ConnectionStatus = "reconnecting" // 重连中
)

// SyncTask 同步任务
type SyncTask struct {
	ID           string                 `json:"id"`
	Type         SyncTaskType           `json:"type"`
	Priority     int                    `json:"priority"`
	StateID      string                 `json:"state_id"`
	TargetAgents []string               `json:"target_agents"`
	Updates      []StateUpdate          `json:"updates"`
	CreatedAt    time.Time              `json:"created_at"`
	StartedAt    time.Time              `json:"started_at,omitempty"`
	CompletedAt  time.Time              `json:"completed_at,omitempty"`
	Status       TaskStatus             `json:"status"`
	RetryCount   int                    `json:"retry_count"`
	MaxRetries   int                    `json:"max_retries"`
	Error        string                 `json:"error,omitempty"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// SyncTaskType 同步任务类型
type SyncTaskType string

const (
	SyncTaskTypeBroadcast SyncTaskType = "broadcast"  // 广播同步
	SyncTaskTypeUnicast   SyncTaskType = "unicast"    // 单播同步
	SyncTaskTypeMulticast SyncTaskType = "multicast"  // 多播同步
	SyncTaskTypeFullSync  SyncTaskType = "full_sync"  // 全量同步
	SyncTaskTypeDeltaSync SyncTaskType = "delta_sync" // 增量同步
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"   // 待执行
	TaskStatusRunning   TaskStatus = "running"   // 执行中
	TaskStatusCompleted TaskStatus = "completed" // 已完成
	TaskStatusFailed    TaskStatus = "failed"    // 失败
	TaskStatusCancelled TaskStatus = "cancelled" // 已取消
)

// StateUpdate 状态更新
type StateUpdate struct {
	ID        string                 `json:"id"`
	StateID   string                 `json:"state_id"`
	AgentID   string                 `json:"agent_id"`
	Type      OperationType          `json:"type"`
	Key       string                 `json:"key"`
	OldValue  interface{}            `json:"old_value"`
	NewValue  interface{}            `json:"new_value"`
	Version   int64                  `json:"version"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// SyncMetrics 同步指标
type SyncMetrics struct {
	TotalSyncs        int64         `json:"total_syncs"`
	SuccessfulSyncs   int64         `json:"successful_syncs"`
	FailedSyncs       int64         `json:"failed_syncs"`
	Conflicts         int64         `json:"conflicts"`
	PendingTasks      int64         `json:"pending_tasks"`
	ActiveConnections int64         `json:"active_connections"`
	AverageLatency    time.Duration `json:"average_latency"`
	LastSyncTime      time.Time     `json:"last_sync_time"`
	ThroughputPerSec  float64       `json:"throughput_per_sec"`
}

// NewStateSyncManager 创建状态同步管理器
func NewStateSyncManager(stateManager *StateManager, eventBus EventBus, config *SyncConfig) *StateSyncManager {
	if config == nil {
		config = &SyncConfig{
			SyncInterval:       time.Second * 10,
			MaxRetries:         3,
			RetryDelay:         time.Millisecond * 1000,
			Timeout:            time.Second * 30,
			BatchSize:          100,
			FlushInterval:      time.Second * 5,
			ConsistencyLevel:   ConsistencyLevelEventual,
			ConflictStrategy:   ConflictResolutionLastWrite,
			MaxConcurrentSyncs: 10,
			QueueBufferSize:    1000,
		}
	}

	ssm := &StateSyncManager{
		stateManager: stateManager,
		eventBus:     eventBus,
		config:       config,
		agents:       make(map[string]*AgentConnection),
		syncTasks:    make(map[string]*SyncTask),
		metrics:      &SyncMetrics{},
	}

	// 启动同步工作协程
	go ssm.startSyncWorker()
	go ssm.startHeartbeatWorker()

	return ssm
}

// ConnectAgent 连接Agent
func (ssm *StateSyncManager) ConnectAgent(agentID, name, address string, connType ConnectionType, states []string) (*AgentConnection, error) {
	ssm.mu.Lock()
	defer ssm.mu.Unlock()

	if _, exists := ssm.agents[agentID]; exists {
		return nil, fmt.Errorf("agent %s is already connected", agentID)
	}

	conn := &AgentConnection{
		ID:               agentID,
		Name:             name,
		Address:          address,
		SubscribedStates: states,
		ConnectionType:   connType,
		Status:           ConnectionStatusConnected,
		LastHeartbeat:    time.Now(),
		Metadata:         make(map[string]interface{}),
		SyncStatus:       make(map[string]SyncStatus),
		PendingUpdates:   []StateUpdate{},
		updateChan:       make(chan StateUpdate, ssm.config.QueueBufferSize),
	}

	ssm.agents[agentID] = conn
	ssm.metrics.ActiveConnections++

	// 启动Agent同步协程
	go ssm.agentSyncWorker(conn)

	// 订阅相关状态事件
	if ssm.eventBus != nil {
		for _, stateID := range states {
			_ = ssm.eventBus.Subscribe(EventTypeStateUpdated, &StateEventHandler{
				stateID:     stateID,
				agentID:     agentID,
				syncManager: ssm,
			})
		}
	}

	return conn, nil
}

// DisconnectAgent 断开Agent连接
func (ssm *StateSyncManager) DisconnectAgent(agentID string) error {
	ssm.mu.Lock()
	defer ssm.mu.Unlock()

	conn, exists := ssm.agents[agentID]
	if !exists {
		return fmt.Errorf("agent %s is not connected", agentID)
	}

	conn.Status = ConnectionStatusDisconnected
	close(conn.updateChan)
	delete(ssm.agents, agentID)
	ssm.metrics.ActiveConnections--

	return nil
}

// GetAgentConnection 获取Agent连接
func (ssm *StateSyncManager) GetAgentConnection(agentID string) (*AgentConnection, error) {
	ssm.mu.RLock()
	defer ssm.mu.RUnlock()

	conn, exists := ssm.agents[agentID]
	if !exists {
		return nil, fmt.Errorf("agent %s is not connected", agentID)
	}

	return conn, nil
}

// ListAgentConnections 列出所有Agent连接
func (ssm *StateSyncManager) ListAgentConnections() map[string]*AgentConnection {
	ssm.mu.RLock()
	defer ssm.mu.RUnlock()

	// 创建副本
	connections := make(map[string]*AgentConnection)
	for id, conn := range ssm.agents {
		connections[id] = conn
	}

	return connections
}

// CreateSyncTask 创建同步任务
func (ssm *StateSyncManager) CreateSyncTask(taskType SyncTaskType, stateID string, targetAgents []string, updates []StateUpdate) (*SyncTask, error) {
	task := &SyncTask{
		ID:           generateTaskID(),
		Type:         taskType,
		Priority:     1,
		StateID:      stateID,
		TargetAgents: targetAgents,
		Updates:      updates,
		CreatedAt:    time.Now(),
		Status:       TaskStatusPending,
		MaxRetries:   ssm.config.MaxRetries,
		Metadata:     make(map[string]interface{}),
	}

	ssm.taskMutex.Lock()
	ssm.syncTasks[task.ID] = task
	ssm.taskMutex.Unlock()

	ssm.metrics.PendingTasks++

	// 异步执行任务
	go ssm.executeSyncTask(task)

	return task, nil
}

// executeSyncTask 执行同步任务
func (ssm *StateSyncManager) executeSyncTask(task *SyncTask) {
	task.Status = TaskStatusRunning
	task.StartedAt = time.Now()

	// 检查并发限制
	if ssm.getCurrentSyncCount() >= ssm.config.MaxConcurrentSyncs {
		// 等待或重新调度
		go func() {
			time.Sleep(time.Millisecond * 100)
			ssm.executeSyncTask(task)
		}()
		return
	}

	// 执行同步
	err := ssm.performSync(task)

	if err != nil {
		task.Error = err.Error()
		task.RetryCount++

		if task.RetryCount < task.MaxRetries {
			task.Status = TaskStatusPending
			// 重试延迟
			go func() {
				time.Sleep(ssm.config.RetryDelay * time.Duration(task.RetryCount))
				ssm.executeSyncTask(task)
			}()
		} else {
			task.Status = TaskStatusFailed
			ssm.metrics.FailedSyncs++
		}
	} else {
		task.Status = TaskStatusCompleted
		task.CompletedAt = time.Now()
		ssm.metrics.SuccessfulSyncs++
	}

	ssm.metrics.TotalSyncs++
	ssm.metrics.PendingTasks--
	ssm.metrics.LastSyncTime = time.Now()
}

// performSync 执行同步操作
func (ssm *StateSyncManager) performSync(task *SyncTask) error {
	for _, agentID := range task.TargetAgents {
		conn, err := ssm.GetAgentConnection(agentID)
		if err != nil {
			return err
		}

		for _, update := range task.Updates {
			select {
			case conn.updateChan <- update:
				// 更新发送成功
			case <-time.After(ssm.config.Timeout):
				return fmt.Errorf("timeout sending update to agent %s", agentID)
			}
		}
	}

	return nil
}

// startSyncWorker 启动同步工作协程
func (ssm *StateSyncManager) startSyncWorker() {
	ticker := time.NewTicker(ssm.config.SyncInterval)
	defer ticker.Stop()

	for range ticker.C {
		ssm.processPendingSyncs()
	}
}

// processPendingSyncs 处理待处理的同步
func (ssm *StateSyncManager) processPendingSyncs() {
	ssm.mu.RLock()
	connections := make([]*AgentConnection, 0, len(ssm.agents))
	for _, conn := range ssm.agents {
		connections = append(connections, conn)
	}
	ssm.mu.RUnlock()

	for _, conn := range connections {
		if len(conn.PendingUpdates) > 0 {
			go ssm.syncAgent(conn)
		}
	}
}

// syncAgent 同步Agent
func (ssm *StateSyncManager) syncAgent(conn *AgentConnection) {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if len(conn.PendingUpdates) == 0 {
		return
	}

	// 批量处理更新
	batchSize := ssm.config.BatchSize
	if batchSize <= 0 {
		batchSize = 1
	}

	for i := 0; i < len(conn.PendingUpdates); i += batchSize {
		end := i + batchSize
		if end > len(conn.PendingUpdates) {
			end = len(conn.PendingUpdates)
		}

		batch := conn.PendingUpdates[i:end]
		if err := ssm.sendBatch(conn, batch); err != nil {
			// 记录错误但继续处理其他批次
			continue
		}
	}

	// 清空已处理的更新
	conn.PendingUpdates = conn.PendingUpdates[:0]
}

// sendBatch 发送批量更新
func (ssm *StateSyncManager) sendBatch(conn *AgentConnection, updates []StateUpdate) error {
	// TODO: 实现实际的发送逻辑
	// 这里应该根据连接类型(推送/拉取)进行相应的处理
	for _, update := range updates {
		select {
		case conn.updateChan <- update:
		case <-time.After(ssm.config.Timeout):
			return fmt.Errorf("timeout sending update")
		}
	}
	return nil
}

// agentSyncWorker Agent同步工作协程
func (ssm *StateSyncManager) agentSyncWorker(conn *AgentConnection) {
	for update := range conn.updateChan {
		// 处理更新
		ssm.handleStateUpdate(conn, update)
	}
}

// handleStateUpdate 处理状态更新
func (ssm *StateSyncManager) handleStateUpdate(conn *AgentConnection, update StateUpdate) {
	// TODO: 实现实际的状态更新处理逻辑
	// 这里可以调用Agent的API或发送消息
}

// startHeartbeatWorker 启动心跳工作协程
func (ssm *StateSyncManager) startHeartbeatWorker() {
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()

	for range ticker.C {
		ssm.checkAgentHeartbeats()
	}
}

// checkAgentHeartbeats 检查Agent心跳
func (ssm *StateSyncManager) checkAgentHeartbeats() {
	ssm.mu.RLock()
	defer ssm.mu.RUnlock()

	now := time.Now()
	for _, conn := range ssm.agents {
		if now.Sub(conn.LastHeartbeat) > time.Minute*2 {
			// 标记为断开连接
			conn.Status = ConnectionStatusDisconnected
		}
	}
}

// getCurrentSyncCount 获取当前同步数量
func (ssm *StateSyncManager) getCurrentSyncCount() int {
	ssm.taskMutex.Lock()
	defer ssm.taskMutex.Unlock()

	count := 0
	for _, task := range ssm.syncTasks {
		if task.Status == TaskStatusRunning {
			count++
		}
	}
	return count
}

// GetMetrics 获取同步指标
func (ssm *StateSyncManager) GetMetrics() *SyncMetrics {
	ssm.mu.RLock()
	defer ssm.mu.RUnlock()

	// 复制指标
	metrics := *ssm.metrics
	metrics.ActiveConnections = int64(len(ssm.agents))

	return &metrics
}

// StateEventHandler 状态事件处理器
type StateEventHandler struct {
	stateID     string
	agentID     string
	syncManager *StateSyncManager
}

// Handle 处理状态事件
func (seh *StateEventHandler) Handle(ctx context.Context, event StateEvent) error {
	if event.StateID != seh.stateID {
		return nil
	}

	// 创建状态更新
	update := StateUpdate{
		ID:        generateOperationID(),
		StateID:   event.StateID,
		AgentID:   event.AgentID,
		Timestamp: event.Timestamp,
	}

	if event.Operation != nil {
		update.Type = event.Operation.Type
		update.Key = event.Operation.Key
		update.OldValue = event.Operation.OldValue
		update.NewValue = event.Operation.Value
	}

	// 获取连接并发送更新
	conn, err := seh.syncManager.GetAgentConnection(seh.agentID)
	if err != nil {
		return err
	}

	conn.mu.Lock()
	conn.PendingUpdates = append(conn.PendingUpdates, update)
	conn.mu.Unlock()

	return nil
}

// generateTaskID 生成任务ID
func generateTaskID() string {
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}
