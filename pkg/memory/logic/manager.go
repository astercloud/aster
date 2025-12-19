package logic

import (
	"context"
	"errors"
	"maps"
	"time"

	"github.com/astercloud/aster/pkg/memory"
	"github.com/google/uuid"
)

// Manager Logic Memory 核心管理器
// 负责 Memory 的捕获、存储、检索、合并和清理
type Manager struct {
	// store 存储后端
	store LogicMemoryStore

	// matchers 模式匹配器列表（支持多个 Matcher 并行工作）
	matchers []PatternMatcher

	// config 管理器配置
	config *ManagerConfig
}

// ManagerConfig Manager 配置
type ManagerConfig struct {
	// Store 存储后端（必需）
	Store LogicMemoryStore

	// Matchers 模式匹配器列表（可选，至少提供一个才能自动捕获）
	Matchers []PatternMatcher

	// DefaultProvenance 默认溯源信息（可选）
	DefaultProvenance *memory.MemoryProvenance

	// AutoConsolidate 是否自动合并相似 Memory（默认 false）
	AutoConsolidate bool

	// ConsolidationThreshold 相似度阈值（用于自动合并，默认 0.85）
	ConsolidationThreshold float64

	// ConfidenceBoost 每次重复出现的置信度提升（默认 0.05）
	ConfidenceBoost float64
}

// NewManager 创建 Logic Memory Manager
func NewManager(config *ManagerConfig) (*Manager, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if config.Store == nil {
		return nil, errors.New("store is required")
	}

	// 设置默认值
	if config.ConsolidationThreshold == 0 {
		config.ConsolidationThreshold = 0.85
	}
	if config.ConfidenceBoost == 0 {
		config.ConfidenceBoost = 0.05
	}

	return &Manager{
		store:    config.Store,
		matchers: config.Matchers,
		config:   config,
	}, nil
}

// RecordMemory 主动记录 Memory（应用层手动调用）
func (m *Manager) RecordMemory(ctx context.Context, memory *LogicMemory) error {
	// 设置 ID
	if memory.ID == "" {
		memory.ID = uuid.New().String()
	}

	// 设置默认溯源
	if memory.Provenance == nil && m.config.DefaultProvenance != nil {
		memory.Provenance = m.config.DefaultProvenance
	}

	// 检查是否已存在
	existing, err := m.store.Get(ctx, memory.Namespace, memory.Key)
	if err == nil && existing != nil {
		// 更新已有 Memory
		m.mergeMemory(existing, memory)
		return m.store.Save(ctx, existing)
	}

	// 创建新 Memory
	return m.store.Save(ctx, memory)
}

// ProcessEvent 处理事件，自动识别和记录 Memory（被动触发）
// 这是核心的自动捕获逻辑
func (m *Manager) ProcessEvent(ctx context.Context, event Event) error {
	if len(m.matchers) == 0 {
		// 没有 Matcher，跳过
		return nil
	}

	// 1. 遍历所有 Matcher
	var allMemories []*LogicMemory
	for _, matcher := range m.matchers {
		// 检查 Matcher 是否支持此事件类型
		if !m.supportsEventType(matcher, event.Type) {
			continue
		}

		// 识别 Memory
		memories, err := matcher.MatchEvent(ctx, event)
		if err != nil {
			// 记录错误但不中断处理
			// TODO: 可以添加日志
			continue
		}

		allMemories = append(allMemories, memories...)
	}

	// 2. 保存或更新 Memory
	for _, mem := range allMemories {
		// 设置 ID
		if mem.ID == "" {
			mem.ID = uuid.New().String()
		}

		// 设置默认溯源
		if mem.Provenance == nil && m.config.DefaultProvenance != nil {
			mem.Provenance = m.config.DefaultProvenance
		}

		// 检查是否已存在
		existing, err := m.store.Get(ctx, mem.Namespace, mem.Key)
		if err == nil && existing != nil {
			// 更新已有 Memory（提升置信度、累积证据）
			m.mergeMemory(existing, mem)
			if err := m.store.Save(ctx, existing); err != nil {
				// TODO: 记录错误
				continue
			}
		} else {
			// 创建新 Memory
			if err := m.store.Save(ctx, mem); err != nil {
				// TODO: 记录错误
				continue
			}
		}
	}

	return nil
}

// RetrieveMemories 检索 Memory（用于 Prompt 注入）
func (m *Manager) RetrieveMemories(
	ctx context.Context,
	namespace string,
	filters ...Filter,
) ([]*LogicMemory, error) {
	// 检索 Memory
	memories, err := m.store.List(ctx, namespace, filters...)
	if err != nil {
		return nil, err
	}

	// 更新访问计数（异步，不阻塞）
	go func() {
		for _, mem := range memories {
			// 忽略错误，访问计数不是关键操作
			_ = m.store.IncrementAccessCount(context.Background(), mem.Namespace, mem.Key)
		}
	}()

	return memories, nil
}

// GetMemory 获取单个 Memory
func (m *Manager) GetMemory(ctx context.Context, namespace, key string) (*LogicMemory, error) {
	memory, err := m.store.Get(ctx, namespace, key)
	if err != nil {
		return nil, err
	}

	// 更新访问计数
	go func() {
		_ = m.store.IncrementAccessCount(context.Background(), namespace, key)
	}()

	return memory, nil
}

// DeleteMemory 删除 Memory
func (m *Manager) DeleteMemory(ctx context.Context, namespace, key string) error {
	return m.store.Delete(ctx, namespace, key)
}

// GetStats 获取统计信息
func (m *Manager) GetStats(ctx context.Context, namespace string) (*MemoryStats, error) {
	return m.store.GetStats(ctx, namespace)
}

// PruneMemories 清理低价值 Memory（定期任务）
func (m *Manager) PruneMemories(ctx context.Context, criteria PruneCriteria) (int, error) {
	return m.store.Prune(ctx, criteria)
}

// Close 关闭 Manager
func (m *Manager) Close() error {
	return m.store.Close()
}

// mergeMemory 合并新旧 Memory（提升置信度）
// 这是一个简单的启发式实现，未来可以用 LLM 或 RL 优化
func (m *Manager) mergeMemory(existing, new *LogicMemory) {
	// 1. 增加访问计数
	existing.AccessCount++

	// 2. 提升置信度（最多 1.0）
	if existing.Provenance != nil && new.Provenance != nil {
		boost := m.config.ConfidenceBoost
		existing.Provenance.Confidence = min(existing.Provenance.Confidence+boost, 1.0)

		// 合并 Sources
		if len(new.Provenance.Sources) > 0 {
			existing.Provenance.Sources = append(existing.Provenance.Sources, new.Provenance.Sources...)
		}
	}

	// 3. 更新 Description（如果新的更详细）
	if len(new.Description) > len(existing.Description) {
		existing.Description = new.Description
	}

	// 4. 合并 Value（简单覆盖，应用层可以自定义）
	// TODO: 支持自定义合并逻辑
	existing.Value = new.Value

	// 5. 合并 Metadata
	if len(new.Metadata) > 0 {
		if existing.Metadata == nil {
			existing.Metadata = make(map[string]any)
		}
		maps.Copy(existing.Metadata, new.Metadata)
	}

	// 6. 更新时间
	existing.UpdatedAt = time.Now()
	existing.LastAccessed = time.Now()
}

// supportsEventType 检查 Matcher 是否支持指定的事件类型
func (m *Manager) supportsEventType(matcher PatternMatcher, eventType string) bool {
	supported := matcher.SupportedEventTypes()

	// 空列表表示支持所有事件
	if len(supported) == 0 {
		return true
	}

	// 检查是否支持通配符 "*"
	for _, t := range supported {
		if t == "*" || t == eventType {
			return true
		}
	}

	return false
}

// min 返回两个 float64 的最小值
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
