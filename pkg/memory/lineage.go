package memory

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/astercloud/aster/pkg/logging"
)

var lineageLog = logging.ForComponent("LineageGraph")

// LineageGraph 记忆谱系图。
// 追踪记忆之间的派生关系，支持级联删除和影响分析。
type LineageGraph struct {
	mu sync.RWMutex

	// parentToChildren 父记忆 -> 派生记忆列表
	parentToChildren map[string][]string

	// childToParents 派生记忆 -> 父记忆列表
	childToParents map[string][]string

	// memoryMetadata 记忆ID -> 元数据
	memoryMetadata map[string]*LineageMetadata
}

// LineageMetadata 记忆的谱系元数据。
type LineageMetadata struct {
	ID             string   // 记忆ID
	SourceIDs      []string // 来源标识（session/document ID）
	DerivedFromIDs []string // 派生自哪些记忆
	CreatedAt      int64    // Unix timestamp
}

// NewLineageGraph 创建谱系图。
func NewLineageGraph() *LineageGraph {
	return &LineageGraph{
		parentToChildren: make(map[string][]string),
		childToParents:   make(map[string][]string),
		memoryMetadata:   make(map[string]*LineageMetadata),
	}
}

// TrackMemory 追踪一个新记忆。
func (lg *LineageGraph) TrackMemory(memoryID string, metadata *LineageMetadata) {
	lg.mu.Lock()
	defer lg.mu.Unlock()

	lg.memoryMetadata[memoryID] = metadata

	// 建立派生关系
	for _, parentID := range metadata.DerivedFromIDs {
		if _, exists := lg.parentToChildren[parentID]; !exists {
			lg.parentToChildren[parentID] = []string{}
		}
		lg.parentToChildren[parentID] = append(lg.parentToChildren[parentID], memoryID)

		if _, exists := lg.childToParents[memoryID]; !exists {
			lg.childToParents[memoryID] = []string{}
		}
		lg.childToParents[memoryID] = append(lg.childToParents[memoryID], parentID)
	}
}

// GetDerivedMemories 获取派生自指定记忆的所有记忆（递归）。
func (lg *LineageGraph) GetDerivedMemories(memoryID string) []string {
	lg.mu.RLock()
	defer lg.mu.RUnlock()

	visited := make(map[string]bool)
	var result []string

	var traverse func(string)
	traverse = func(id string) {
		if visited[id] {
			return
		}
		visited[id] = true

		children := lg.parentToChildren[id]
		for _, child := range children {
			result = append(result, child)
			traverse(child)
		}
	}

	traverse(memoryID)
	return result
}

// GetParentMemories 获取指定记忆的所有父记忆（递归）。
func (lg *LineageGraph) GetParentMemories(memoryID string) []string {
	lg.mu.RLock()
	defer lg.mu.RUnlock()

	visited := make(map[string]bool)
	var result []string

	var traverse func(string)
	traverse = func(id string) {
		if visited[id] {
			return
		}
		visited[id] = true

		parents := lg.childToParents[id]
		for _, parent := range parents {
			result = append(result, parent)
			traverse(parent)
		}
	}

	traverse(memoryID)
	return result
}

// GetMemoriesBySource 获取来自特定数据源的所有记忆。
func (lg *LineageGraph) GetMemoriesBySource(sourceID string) []string {
	lg.mu.RLock()
	defer lg.mu.RUnlock()

	var result []string
	for memID, metadata := range lg.memoryMetadata {
		if slices.Contains(metadata.SourceIDs, sourceID) {
			result = append(result, memID)
		}
	}
	return result
}

// RemoveMemory 从谱系图中移除记忆。
func (lg *LineageGraph) RemoveMemory(memoryID string) {
	lg.mu.Lock()
	defer lg.mu.Unlock()

	// 移除元数据
	delete(lg.memoryMetadata, memoryID)

	// 移除父子关系：从所有子节点的父列表中删除此节点
	children := lg.parentToChildren[memoryID]
	for _, child := range children {
		lg.childToParents[child] = lg.removeFromSlice(lg.childToParents[child], memoryID)
	}
	delete(lg.parentToChildren, memoryID)

	// 移除子父关系：从所有父节点的子列表中删除此节点
	parents := lg.childToParents[memoryID]
	for _, parent := range parents {
		lg.parentToChildren[parent] = lg.removeFromSlice(lg.parentToChildren[parent], memoryID)
	}
	delete(lg.childToParents, memoryID)
}

// removeFromSlice 从切片中移除指定值，返回新切片。
func (lg *LineageGraph) removeFromSlice(slice []string, value string) []string {
	result := make([]string, 0, len(slice))
	for _, v := range slice {
		if v != value {
			result = append(result, v)
		}
	}
	return result
}

// LineageManager 记忆谱系管理器。
// 提供高级的谱系追踪和级联删除功能。
type LineageManager struct {
	graph *LineageGraph
}

// NewLineageManager 创建谱系管理器。
func NewLineageManager() *LineageManager {
	return &LineageManager{
		graph: NewLineageGraph(),
	}
}

// TrackMemoryCreation 追踪记忆创建事件。
func (lm *LineageManager) TrackMemoryCreation(memoryID string, provenance *MemoryProvenance, derivedFromIDs []string) error {
	if memoryID == "" {
		return errors.New("memoryID is required")
	}

	metadata := &LineageMetadata{
		ID:             memoryID,
		SourceIDs:      provenance.Sources,
		DerivedFromIDs: derivedFromIDs,
		CreatedAt:      provenance.CreatedAt.Unix(),
	}

	lm.graph.TrackMemory(memoryID, metadata)
	return nil
}

// DeleteMemoryWithLineage 删除记忆及其派生记忆。
func (lm *LineageManager) DeleteMemoryWithLineage(ctx context.Context, memoryID string, cascade bool) ([]string, error) {
	var deletedIDs []string

	if cascade {
		// 级联删除：包括所有派生记忆
		derivedIDs := lm.graph.GetDerivedMemories(memoryID)
		deletedIDs = append([]string{memoryID}, derivedIDs...)
	} else {
		// 仅删除当前记忆
		deletedIDs = []string{memoryID}
	}

	// 从谱系图中移除
	for _, id := range deletedIDs {
		lm.graph.RemoveMemory(id)
	}

	return deletedIDs, nil
}

// RevokeDataSource 撤销数据源权限。
// 删除所有派生自该数据源的记忆。
func (lm *LineageManager) RevokeDataSource(ctx context.Context, sourceID string) ([]string, error) {
	// 找到所有来自该数据源的记忆
	affectedMemories := lm.graph.GetMemoriesBySource(sourceID)

	if len(affectedMemories) == 0 {
		return nil, nil
	}

	// 删除这些记忆及其派生
	var deletedIDs []string
	for _, memID := range affectedMemories {
		deleted, err := lm.DeleteMemoryWithLineage(ctx, memID, true)
		if err != nil {
			// TODO: handle error
			return deletedIDs, fmt.Errorf("delete memory %s: %w", memID, err)
		}
		deletedIDs = append(deletedIDs, deleted...)
	}

	return deletedIDs, nil
}

// RegenerateFromSource 从剩余有效数据源重新生成记忆。
// 这是一个更精确的删除策略，避免过度删除。
func (lm *LineageManager) RegenerateFromSource(ctx context.Context, revokedSourceID string) error {
	// 找到受影响的记忆
	affectedMemories := lm.graph.GetMemoriesBySource(revokedSourceID)

	for _, memID := range affectedMemories {
		metadata := lm.graph.memoryMetadata[memID]
		if metadata == nil {
			continue
		}

		// 检查是否还有其他有效来源
		hasOtherSources := false
		for _, sourceID := range metadata.SourceIDs {
			if sourceID != revokedSourceID {
				hasOtherSources = true
				break
			}
		}

		if !hasOtherSources {
			// 没有其他来源，删除记忆
			_, _ = lm.DeleteMemoryWithLineage(ctx, memID, false)
		} else {
			// 有其他来源，需要重新生成记忆
			// TODO: 调用记忆生成服务从剩余来源重建
			// 这部分逻辑需要与 Memory Consolidation 集成
			lineageLog.Info(ctx, "memory has other sources, regeneration needed", map[string]any{"memory_id": memID})
		}
	}

	return nil
}

// GetLineageDepth 获取记忆的谱系深度。
// 返回从根记忆到当前记忆的最长路径。
func (lm *LineageManager) GetLineageDepth(memoryID string) int {
	parents := lm.graph.GetParentMemories(memoryID)
	if len(parents) == 0 {
		return 0
	}

	maxDepth := 0
	for _, parentID := range parents {
		depth := lm.GetLineageDepth(parentID)
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	return maxDepth + 1
}

// GetLineageStats 获取谱系统计信息。
type LineageStats struct {
	TotalMemories    int            // 总记忆数
	RootMemories     int            // 根记忆数（无父记忆）
	DerivedMemories  int            // 派生记忆数（有父记忆）
	MaxDepth         int            // 最大谱系深度
	MemoriesBySource map[string]int // 每个数据源的记忆数
}

func (lm *LineageManager) GetLineageStats() LineageStats {
	lm.graph.mu.RLock()
	defer lm.graph.mu.RUnlock()

	stats := LineageStats{
		TotalMemories:    len(lm.graph.memoryMetadata),
		MemoriesBySource: make(map[string]int),
	}

	for memID, metadata := range lm.graph.memoryMetadata {
		// 统计根记忆
		if len(metadata.DerivedFromIDs) == 0 {
			stats.RootMemories++
		} else {
			stats.DerivedMemories++
		}

		// 统计谱系深度
		depth := lm.GetLineageDepth(memID)
		if depth > stats.MaxDepth {
			stats.MaxDepth = depth
		}

		// 统计每个数据源
		for _, sourceID := range metadata.SourceIDs {
			stats.MemoriesBySource[sourceID]++
		}
	}

	return stats
}
