package memory

import (
	"context"
	"sync"
	"time"
)

// ReferenceRegistry 引用注册表接口
// 用于跟踪和管理会话中出现的所有引用（文件、URL、函数等）
// 这使得压缩后的内容可以在需要时恢复
type ReferenceRegistry interface {
	// Register 注册一个引用
	Register(ctx context.Context, ref Reference, sourceContext string) error

	// Lookup 查找引用信息
	Lookup(ctx context.Context, refType, value string) (*ReferenceInfo, error)

	// ListByType 按类型列出引用
	ListByType(ctx context.Context, refType string, limit int) ([]ReferenceInfo, error)

	// ListRecent 列出最近的引用
	ListRecent(ctx context.Context, limit int) ([]ReferenceInfo, error)

	// MarkAccessed 标记引用被访问
	MarkAccessed(ctx context.Context, refType, value string) error

	// GetStats 获取统计信息
	GetStats(ctx context.Context) (*RegistryStats, error)

	// Cleanup 清理过期引用
	Cleanup(ctx context.Context, maxAge time.Duration) (int, error)
}

// ReferenceInfo 引用的详细信息
type ReferenceInfo struct {
	// Reference 基本引用信息
	Reference Reference `json:"reference"`

	// SourceContext 引用来源的上下文
	SourceContext string `json:"source_context,omitempty"`

	// FirstSeen 首次出现时间
	FirstSeen time.Time `json:"first_seen"`

	// LastAccessed 最后访问时间
	LastAccessed time.Time `json:"last_accessed"`

	// AccessCount 访问次数
	AccessCount int `json:"access_count"`

	// ToolName 产生此引用的工具
	ToolName string `json:"tool_name,omitempty"`

	// Metadata 额外元数据
	Metadata map[string]any `json:"metadata,omitempty"`
}

// RegistryStats 注册表统计信息
type RegistryStats struct {
	// TotalReferences 总引用数
	TotalReferences int `json:"total_references"`

	// ByType 按类型统计
	ByType map[string]int `json:"by_type"`

	// MostAccessed 最常访问的引用
	MostAccessed []ReferenceInfo `json:"most_accessed,omitempty"`

	// RecentlyAdded 最近添加的引用数
	RecentlyAdded int `json:"recently_added"`
}

// InMemoryReferenceRegistry 内存实现的引用注册表
type InMemoryReferenceRegistry struct {
	mu         sync.RWMutex
	references map[string]*ReferenceInfo // key: "type:value"
	maxSize    int
}

// NewInMemoryReferenceRegistry 创建内存引用注册表
func NewInMemoryReferenceRegistry(maxSize int) *InMemoryReferenceRegistry {
	if maxSize <= 0 {
		maxSize = 1000
	}
	return &InMemoryReferenceRegistry{
		references: make(map[string]*ReferenceInfo),
		maxSize:    maxSize,
	}
}

// makeKey 生成引用的唯一键
func makeKey(refType, value string) string {
	return refType + ":" + value
}

// Register 注册引用
func (r *InMemoryReferenceRegistry) Register(ctx context.Context, ref Reference, sourceContext string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := makeKey(ref.Type, ref.Value)
	now := time.Now()

	if existing, ok := r.references[key]; ok {
		// 更新现有引用
		existing.AccessCount++
		existing.LastAccessed = now
		if sourceContext != "" && existing.SourceContext == "" {
			existing.SourceContext = sourceContext
		}
		return nil
	}

	// 检查是否需要清理
	if len(r.references) >= r.maxSize {
		r.evictOldest()
	}

	// 添加新引用
	r.references[key] = &ReferenceInfo{
		Reference:     ref,
		SourceContext: sourceContext,
		FirstSeen:     now,
		LastAccessed:  now,
		AccessCount:   1,
	}

	return nil
}

// Lookup 查找引用
func (r *InMemoryReferenceRegistry) Lookup(ctx context.Context, refType, value string) (*ReferenceInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := makeKey(refType, value)
	if info, ok := r.references[key]; ok {
		// 返回副本
		copy := *info
		return &copy, nil
	}
	return nil, nil
}

// ListByType 按类型列出引用
func (r *InMemoryReferenceRegistry) ListByType(ctx context.Context, refType string, limit int) ([]ReferenceInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []ReferenceInfo
	for _, info := range r.references {
		if info.Reference.Type == refType {
			results = append(results, *info)
			if limit > 0 && len(results) >= limit {
				break
			}
		}
	}
	return results, nil
}

// ListRecent 列出最近的引用
func (r *InMemoryReferenceRegistry) ListRecent(ctx context.Context, limit int) ([]ReferenceInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 收集所有引用
	all := make([]ReferenceInfo, 0, len(r.references))
	for _, info := range r.references {
		all = append(all, *info)
	}

	// 按最后访问时间排序
	sortByLastAccessed(all)

	// 返回限制数量
	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

// MarkAccessed 标记访问
func (r *InMemoryReferenceRegistry) MarkAccessed(ctx context.Context, refType, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := makeKey(refType, value)
	if info, ok := r.references[key]; ok {
		info.AccessCount++
		info.LastAccessed = time.Now()
	}
	return nil
}

// GetStats 获取统计信息
func (r *InMemoryReferenceRegistry) GetStats(ctx context.Context) (*RegistryStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := &RegistryStats{
		TotalReferences: len(r.references),
		ByType:          make(map[string]int),
	}

	oneHourAgo := time.Now().Add(-1 * time.Hour)

	for _, info := range r.references {
		stats.ByType[info.Reference.Type]++
		if info.FirstSeen.After(oneHourAgo) {
			stats.RecentlyAdded++
		}
	}

	return stats, nil
}

// Cleanup 清理过期引用
func (r *InMemoryReferenceRegistry) Cleanup(ctx context.Context, maxAge time.Duration) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for key, info := range r.references {
		if info.LastAccessed.Before(cutoff) {
			delete(r.references, key)
			removed++
		}
	}

	return removed, nil
}

// evictOldest 移除最旧的引用
func (r *InMemoryReferenceRegistry) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, info := range r.references {
		if oldestKey == "" || info.LastAccessed.Before(oldestTime) {
			oldestKey = key
			oldestTime = info.LastAccessed
		}
	}

	if oldestKey != "" {
		delete(r.references, oldestKey)
	}
}

// sortByLastAccessed 按最后访问时间排序（降序）
func sortByLastAccessed(refs []ReferenceInfo) {
	// 简单的冒泡排序，因为通常数据量不大
	for i := range len(refs) - 1 {
		for j := i + 1; j < len(refs); j++ {
			if refs[j].LastAccessed.After(refs[i].LastAccessed) {
				refs[i], refs[j] = refs[j], refs[i]
			}
		}
	}
}

// ReferenceRegistryMiddleware 引用注册中间件
// 自动从工具结果中提取并注册引用
type ReferenceRegistryMiddleware struct {
	registry   ReferenceRegistry
	compressor *DefaultObservationCompressor
}

// NewReferenceRegistryMiddleware 创建引用注册中间件
func NewReferenceRegistryMiddleware(registry ReferenceRegistry) *ReferenceRegistryMiddleware {
	return &ReferenceRegistryMiddleware{
		registry:   registry,
		compressor: NewDefaultObservationCompressor(),
	}
}

// ProcessToolResult 处理工具结果，提取并注册引用
func (m *ReferenceRegistryMiddleware) ProcessToolResult(ctx context.Context, toolName, content string) error {
	// 提取引用
	refs := m.compressor.extractReferences(content)

	// 注册引用
	for _, ref := range refs {
		if err := m.registry.Register(ctx, ref, toolName); err != nil {
			// 记录错误但继续处理
			continue
		}
	}

	return nil
}
