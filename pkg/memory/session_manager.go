package memory

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// MemoryScope 定义记忆的作用域
type MemoryScope string

const (
	ScopePrivate MemoryScope = "private" // 私有，仅当前会话可访问
	ScopeShared  MemoryScope = "shared"  // 共享，指定会话可访问
	ScopeGlobal  MemoryScope = "global"  // 全局，所有会话可访问
)

// AccessLevel 访问级别
type AccessLevel int

const (
	AccessNone        AccessLevel = 0 // 无访问权限
	AccessRead        AccessLevel = 1 // 只读
	AccessWrite       AccessLevel = 2 // 读写
	AccessFullControl AccessLevel = 3 // 完全控制（含删除）
)

// SharedMemory 共享记忆
type SharedMemory struct {
	ID         string                 // 记忆 ID
	Content    string                 // 内容
	Metadata   map[string]any         // 元数据
	Scope      MemoryScope            // 作用域
	OwnerID    string                 // 所有者会话 ID
	SharedWith map[string]AccessLevel // 共享给哪些会话（会话ID -> 访问级别）
	CreatedAt  time.Time              // 创建时间
	UpdatedAt  time.Time              // 更新时间

	// 记忆溯源
	Provenance *MemoryProvenance
}

// SessionMemoryManager 会话记忆管理器
// 管理跨会话的记忆共享和访问控制
type SessionMemoryManager struct {
	mu sync.RWMutex

	// 记忆存储（记忆ID -> SharedMemory）
	memories map[string]*SharedMemory

	// 会话到记忆的索引（会话ID -> 记忆ID列表）
	sessionIndex map[string][]string

	// 全局记忆列表
	globalMemories []string

	// 配置
	config SessionManagerConfig
}

// SessionManagerConfig 会话管理器配置
type SessionManagerConfig struct {
	// 默认作用域
	DefaultScope MemoryScope

	// 是否允许跨会话共享
	EnableSharing bool

	// 是否允许全局记忆
	EnableGlobal bool

	// 记忆过期时间
	MemoryTTL time.Duration

	// 最大共享数量（单个记忆最多共享给多少会话）
	MaxSharedSessions int
}

// DefaultSessionManagerConfig 返回默认配置
func DefaultSessionManagerConfig() SessionManagerConfig {
	return SessionManagerConfig{
		DefaultScope:      ScopePrivate,
		EnableSharing:     true,
		EnableGlobal:      true,
		MemoryTTL:         7 * 24 * time.Hour, // 7 天
		MaxSharedSessions: 100,
	}
}

// NewSessionMemoryManager 创建会话记忆管理器
func NewSessionMemoryManager(config SessionManagerConfig) *SessionMemoryManager {
	return &SessionMemoryManager{
		memories:       make(map[string]*SharedMemory),
		sessionIndex:   make(map[string][]string),
		globalMemories: []string{},
		config:         config,
	}
}

// AddMemory 添加记忆
func (m *SessionMemoryManager) AddMemory(
	ctx context.Context,
	sessionID string,
	content string,
	metadata map[string]any,
	scope MemoryScope,
) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 生成记忆 ID
	memID := fmt.Sprintf("mem-%s-%d", sessionID, time.Now().UnixNano())

	// 创建记忆
	memory := &SharedMemory{
		ID:         memID,
		Content:    content,
		Metadata:   metadata,
		Scope:      scope,
		OwnerID:    sessionID,
		SharedWith: make(map[string]AccessLevel),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Provenance: NewProvenance(SourceUserInput, sessionID),
	}

	// 存储记忆
	m.memories[memID] = memory

	// 更新索引
	m.sessionIndex[sessionID] = append(m.sessionIndex[sessionID], memID)

	// 如果是全局记忆，添加到全局列表
	if scope == ScopeGlobal {
		m.globalMemories = append(m.globalMemories, memID)
	}

	return memID, nil
}

// ShareMemory 共享记忆给其他会话
func (m *SessionMemoryManager) ShareMemory(
	ctx context.Context,
	memoryID string,
	fromSessionID string,
	toSessionID string,
	accessLevel AccessLevel,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查记忆是否存在
	memory, exists := m.memories[memoryID]
	if !exists {
		return fmt.Errorf("memory not found: %s", memoryID)
	}

	// 检查权限（只有所有者可以共享）
	if memory.OwnerID != fromSessionID {
		return errors.New("only owner can share memory")
	}

	// 检查是否启用共享
	if !m.config.EnableSharing {
		return errors.New("sharing is disabled")
	}

	// 检查共享数量限制
	if len(memory.SharedWith) >= m.config.MaxSharedSessions {
		return errors.New("max shared sessions limit reached")
	}

	// 添加共享权限
	memory.SharedWith[toSessionID] = accessLevel
	memory.UpdatedAt = time.Now()

	// 更新目标会话的索引
	m.sessionIndex[toSessionID] = append(m.sessionIndex[toSessionID], memoryID)

	return nil
}

// RevokeAccess 撤销访问权限
func (m *SessionMemoryManager) RevokeAccess(
	ctx context.Context,
	memoryID string,
	fromSessionID string,
	toSessionID string,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	memory, exists := m.memories[memoryID]
	if !exists {
		return fmt.Errorf("memory not found: %s", memoryID)
	}

	// 检查权限
	if memory.OwnerID != fromSessionID {
		return errors.New("only owner can revoke access")
	}

	// 删除共享权限
	delete(memory.SharedWith, toSessionID)
	memory.UpdatedAt = time.Now()

	// 从目标会话索引中移除
	m.removeFromSessionIndex(toSessionID, memoryID)

	return nil
}

// GetMemory 获取记忆（带权限检查）
func (m *SessionMemoryManager) GetMemory(
	ctx context.Context,
	memoryID string,
	sessionID string,
) (*SharedMemory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	memory, exists := m.memories[memoryID]
	if !exists {
		return nil, fmt.Errorf("memory not found: %s", memoryID)
	}

	// 检查访问权限
	if !m.hasAccess(memory, sessionID, AccessRead) {
		return nil, errors.New("access denied")
	}

	return memory, nil
}

// UpdateMemory 更新记忆（带权限检查）
func (m *SessionMemoryManager) UpdateMemory(
	ctx context.Context,
	memoryID string,
	sessionID string,
	content string,
	metadata map[string]any,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	memory, exists := m.memories[memoryID]
	if !exists {
		return fmt.Errorf("memory not found: %s", memoryID)
	}

	// 检查写权限
	if !m.hasAccess(memory, sessionID, AccessWrite) {
		return errors.New("write access denied")
	}

	// 更新内容
	memory.Content = content
	if metadata != nil {
		memory.Metadata = metadata
	}
	memory.UpdatedAt = time.Now()

	return nil
}

// DeleteMemory 删除记忆（仅所有者）
func (m *SessionMemoryManager) DeleteMemory(
	ctx context.Context,
	memoryID string,
	sessionID string,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	memory, exists := m.memories[memoryID]
	if !exists {
		return fmt.Errorf("memory not found: %s", memoryID)
	}

	// 只有所有者可以删除
	if memory.OwnerID != sessionID {
		return errors.New("only owner can delete memory")
	}

	// 从所有索引中移除
	m.removeFromSessionIndex(sessionID, memoryID)
	for sharedSession := range memory.SharedWith {
		m.removeFromSessionIndex(sharedSession, memoryID)
	}

	// 从全局列表移除
	if memory.Scope == ScopeGlobal {
		m.removeFromGlobalList(memoryID)
	}

	// 删除记忆
	delete(m.memories, memoryID)

	return nil
}

// ListSessionMemories 列出会话可访问的所有记忆
func (m *SessionMemoryManager) ListSessionMemories(
	ctx context.Context,
	sessionID string,
	scope MemoryScope,
) ([]*SharedMemory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := []*SharedMemory{}

	// 获取会话的记忆 ID 列表
	memIDs := m.sessionIndex[sessionID]

	for _, memID := range memIDs {
		memory, exists := m.memories[memID]
		if !exists {
			continue
		}

		// 跳过全局记忆（它们会在后面单独处理）
		if memory.Scope == ScopeGlobal {
			continue
		}

		// 过滤作用域
		if scope != "" && memory.Scope != scope {
			continue
		}

		// 检查访问权限
		if m.hasAccess(memory, sessionID, AccessRead) {
			result = append(result, memory)
		}
	}

	// 添加全局记忆
	if scope == "" || scope == ScopeGlobal {
		for _, memID := range m.globalMemories {
			memory := m.memories[memID]
			result = append(result, memory)
		}
	}

	return result, nil
}

// hasAccess 检查会话是否有访问权限
func (m *SessionMemoryManager) hasAccess(
	memory *SharedMemory,
	sessionID string,
	requiredLevel AccessLevel,
) bool {
	// 所有者有完全权限
	if memory.OwnerID == sessionID {
		return true
	}

	// 全局记忆所有人都可以读
	if memory.Scope == ScopeGlobal && requiredLevel == AccessRead {
		return true
	}

	// 检查共享权限
	if level, shared := memory.SharedWith[sessionID]; shared {
		return level >= requiredLevel
	}

	return false
}

// removeFromSessionIndex 从会话索引中移除记忆
func (m *SessionMemoryManager) removeFromSessionIndex(sessionID, memoryID string) {
	memIDs := m.sessionIndex[sessionID]
	newIDs := []string{}
	for _, id := range memIDs {
		if id != memoryID {
			newIDs = append(newIDs, id)
		}
	}
	m.sessionIndex[sessionID] = newIDs
}

// removeFromGlobalList 从全局列表移除
func (m *SessionMemoryManager) removeFromGlobalList(memoryID string) {
	newList := []string{}
	for _, id := range m.globalMemories {
		if id != memoryID {
			newList = append(newList, id)
		}
	}
	m.globalMemories = newList
}

// GetStats 获取统计信息
func (m *SessionMemoryManager) GetStats() SessionStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := SessionStats{
		TotalMemories:     len(m.memories),
		TotalSessions:     len(m.sessionIndex),
		GlobalMemories:    len(m.globalMemories),
		ScopeDistribution: make(map[MemoryScope]int),
	}

	for _, memory := range m.memories {
		stats.ScopeDistribution[memory.Scope]++
	}

	return stats
}

// SessionStats 会话统计信息
type SessionStats struct {
	TotalMemories     int                 // 总记忆数
	TotalSessions     int                 // 总会话数
	GlobalMemories    int                 // 全局记忆数
	ScopeDistribution map[MemoryScope]int // 作用域分布
}

// CleanupExpired 清理过期记忆
func (m *SessionMemoryManager) CleanupExpired(ctx context.Context) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.config.MemoryTTL == 0 {
		return 0 // 未设置 TTL，不清理
	}

	expireTime := time.Now().Add(-m.config.MemoryTTL)
	removed := 0

	for memID, memory := range m.memories {
		if memory.UpdatedAt.Before(expireTime) {
			// 删除过期记忆
			m.removeFromSessionIndex(memory.OwnerID, memID)
			for sharedSession := range memory.SharedWith {
				m.removeFromSessionIndex(sharedSession, memID)
			}
			if memory.Scope == ScopeGlobal {
				m.removeFromGlobalList(memID)
			}
			delete(m.memories, memID)
			removed++
		}
	}

	return removed
}
