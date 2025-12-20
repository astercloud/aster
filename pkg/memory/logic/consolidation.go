package logic

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/astercloud/aster/pkg/logging"
)

var consolidationLog = logging.ForComponent("ConsolidationEngine")

// ConsolidationEngine Memory 合并引擎
// 用于合并相似的 Memory，减少冗余，提高检索效率
type ConsolidationEngine struct {
	store                LogicMemoryStore
	config               *ConsolidationConfig
	similarityCalculator SimilarityCalculator
}

// ConsolidationConfig 合并配置
type ConsolidationConfig struct {
	// SimilarityThreshold 相似度阈值（0.0-1.0）
	// 超过此阈值的 Memory 将被合并
	SimilarityThreshold float64

	// MinGroupSize 最小合并组大小
	// 只有当相似 Memory 数量 >= 此值时才进行合并
	MinGroupSize int

	// MaxMergeCount 单次合并的最大数量
	// 限制单次操作的 Memory 数量，避免长时间阻塞
	MaxMergeCount int

	// PreserveHighConfidence 保留高置信度的 Memory
	// 高置信度的 Memory 不会被合并到其他 Memory
	PreserveHighConfidenceThreshold float64

	// MergeStrategy 合并策略
	MergeStrategy MergeStrategy
}

// MergeStrategy 合并策略
type MergeStrategy string

const (
	// MergeStrategyKeepNewest 保留最新的 Memory
	MergeStrategyKeepNewest MergeStrategy = "keep_newest"

	// MergeStrategyKeepHighestConfidence 保留置信度最高的 Memory
	MergeStrategyKeepHighestConfidence MergeStrategy = "keep_highest_confidence"

	// MergeStrategyMergeDescriptions 合并描述
	MergeStrategyMergeDescriptions MergeStrategy = "merge_descriptions"
)

// SimilarityCalculator 相似度计算接口
type SimilarityCalculator interface {
	// Calculate 计算两个 Memory 的相似度（0.0-1.0）
	Calculate(a, b *LogicMemory) float64
}

// DefaultSimilarityCalculator 默认相似度计算器
// 基于 Type、Key、Description 的简单相似度计算
type DefaultSimilarityCalculator struct{}

// Calculate 计算相似度
func (c *DefaultSimilarityCalculator) Calculate(a, b *LogicMemory) float64 {
	score := 0.0

	// Type 相同 (+0.3)
	if a.Type == b.Type {
		score += 0.3
	}

	// Key 前缀相同 (+0.3)
	keyA := strings.Split(a.Key, "_")[0]
	keyB := strings.Split(b.Key, "_")[0]
	if keyA == keyB {
		score += 0.3
	}

	// Category 相同 (+0.2)
	if a.Category != "" && a.Category == b.Category {
		score += 0.2
	}

	// Description 相似度 (+0.2)
	descSimilarity := c.calculateStringSimilarity(a.Description, b.Description)
	score += descSimilarity * 0.2

	return score
}

// calculateStringSimilarity 计算字符串相似度（Jaccard）
func (c *DefaultSimilarityCalculator) calculateStringSimilarity(a, b string) float64 {
	if a == "" || b == "" {
		return 0
	}

	wordsA := strings.Fields(strings.ToLower(a))
	wordsB := strings.Fields(strings.ToLower(b))

	setA := make(map[string]bool)
	for _, w := range wordsA {
		setA[w] = true
	}

	setB := make(map[string]bool)
	for _, w := range wordsB {
		setB[w] = true
	}

	// 计算交集
	intersection := 0
	for w := range setA {
		if setB[w] {
			intersection++
		}
	}

	// 计算并集
	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

// NewConsolidationEngine 创建合并引擎
func NewConsolidationEngine(store LogicMemoryStore, config *ConsolidationConfig) *ConsolidationEngine {
	if config == nil {
		config = &ConsolidationConfig{
			SimilarityThreshold:             0.85,
			MinGroupSize:                    2,
			MaxMergeCount:                   100,
			PreserveHighConfidenceThreshold: 0.95,
			MergeStrategy:                   MergeStrategyKeepHighestConfidence,
		}
	}

	return &ConsolidationEngine{
		store:                store,
		config:               config,
		similarityCalculator: &DefaultSimilarityCalculator{},
	}
}

// SetSimilarityCalculator 设置自定义相似度计算器
func (e *ConsolidationEngine) SetSimilarityCalculator(calc SimilarityCalculator) {
	e.similarityCalculator = calc
}

// Consolidate 执行合并操作
// 返回合并的组数和删除的 Memory 数量
func (e *ConsolidationEngine) Consolidate(ctx context.Context, namespace string) (*ConsolidationResult, error) {
	result := &ConsolidationResult{
		StartTime: time.Now(),
	}

	// 1. 获取所有 Memory
	memories, err := e.store.List(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to list memories: %w", err)
	}

	result.TotalMemories = len(memories)

	if len(memories) < e.config.MinGroupSize {
		result.EndTime = time.Now()
		return result, nil
	}

	// 2. 按类型分组
	byType := e.groupByType(memories)

	// 3. 在每个类型组内查找相似 Memory
	var mergeGroups [][]*LogicMemory
	for _, group := range byType {
		if len(group) < e.config.MinGroupSize {
			continue
		}

		similarGroups := e.findSimilarGroups(group)
		mergeGroups = append(mergeGroups, similarGroups...)
	}

	// 4. 执行合并
	for _, group := range mergeGroups {
		if len(group) < e.config.MinGroupSize {
			continue
		}

		merged, deleted, err := e.mergeGroup(ctx, namespace, group)
		if err != nil {
			consolidationLog.Warn(ctx, "failed to merge group", map[string]any{"error": err})
			continue
		}

		result.MergedGroups++
		result.DeletedMemories += deleted
		result.MergedMemories = append(result.MergedMemories, merged)
	}

	result.EndTime = time.Now()
	return result, nil
}

// groupByType 按类型分组
func (e *ConsolidationEngine) groupByType(memories []*LogicMemory) map[string][]*LogicMemory {
	groups := make(map[string][]*LogicMemory)
	for _, mem := range memories {
		groups[mem.Type] = append(groups[mem.Type], mem)
	}
	return groups
}

// findSimilarGroups 查找相似组
func (e *ConsolidationEngine) findSimilarGroups(memories []*LogicMemory) [][]*LogicMemory {
	n := len(memories)
	if n < e.config.MinGroupSize {
		return nil
	}

	// 使用并查集查找相似组
	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}

	var find func(x int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}

	union := func(x, y int) {
		px, py := find(x), find(y)
		if px != py {
			parent[px] = py
		}
	}

	// 计算相似度并合并
	for i := range n {
		// 跳过高置信度的 Memory
		if memories[i].Provenance != nil &&
			memories[i].Provenance.Confidence >= e.config.PreserveHighConfidenceThreshold {
			continue
		}

		for j := i + 1; j < n; j++ {
			similarity := e.similarityCalculator.Calculate(memories[i], memories[j])
			if similarity >= e.config.SimilarityThreshold {
				union(i, j)
			}
		}
	}

	// 收集组
	groups := make(map[int][]*LogicMemory)
	for i := range n {
		root := find(i)
		groups[root] = append(groups[root], memories[i])
	}

	// 过滤小组
	var result [][]*LogicMemory
	for _, group := range groups {
		if len(group) >= e.config.MinGroupSize {
			result = append(result, group)
		}
	}

	return result
}

// mergeGroup 合并一组 Memory
func (e *ConsolidationEngine) mergeGroup(ctx context.Context, namespace string, group []*LogicMemory) (*LogicMemory, int, error) {
	if len(group) == 0 {
		return nil, 0, nil
	}

	// 根据策略选择保留的 Memory
	var keeper *LogicMemory
	switch e.config.MergeStrategy {
	case MergeStrategyKeepNewest:
		keeper = e.selectNewest(group)
	case MergeStrategyKeepHighestConfidence:
		keeper = e.selectHighestConfidence(group)
	case MergeStrategyMergeDescriptions:
		keeper = e.mergeDescriptions(group)
	default:
		keeper = e.selectHighestConfidence(group)
	}

	// 合并元数据
	for _, mem := range group {
		if mem.ID == keeper.ID {
			continue
		}

		// 合并 AccessCount
		keeper.AccessCount += mem.AccessCount

		// 合并 Sources
		if keeper.Provenance != nil && mem.Provenance != nil {
			keeper.Provenance.Sources = append(keeper.Provenance.Sources, mem.Provenance.Sources...)
		}

		// 合并 Metadata
		if mem.Metadata != nil {
			if keeper.Metadata == nil {
				keeper.Metadata = make(map[string]any)
			}
			for k, v := range mem.Metadata {
				if _, exists := keeper.Metadata[k]; !exists {
					keeper.Metadata[k] = v
				}
			}
		}
	}

	// 更新保留的 Memory
	keeper.UpdatedAt = time.Now()
	if err := e.store.Save(ctx, keeper); err != nil {
		return nil, 0, fmt.Errorf("failed to save merged memory: %w", err)
	}

	// 删除其他 Memory
	deleted := 0
	for _, mem := range group {
		if mem.ID == keeper.ID {
			continue
		}

		if err := e.store.Delete(ctx, namespace, mem.Key); err != nil {
			consolidationLog.Warn(ctx, "failed to delete merged memory", map[string]any{"key": mem.Key, "error": err})
			continue
		}
		deleted++
	}

	return keeper, deleted, nil
}

// selectNewest 选择最新的 Memory
func (e *ConsolidationEngine) selectNewest(group []*LogicMemory) *LogicMemory {
	sort.Slice(group, func(i, j int) bool {
		return group[i].UpdatedAt.After(group[j].UpdatedAt)
	})
	return group[0]
}

// selectHighestConfidence 选择置信度最高的 Memory
func (e *ConsolidationEngine) selectHighestConfidence(group []*LogicMemory) *LogicMemory {
	sort.Slice(group, func(i, j int) bool {
		ci, cj := 0.0, 0.0
		if group[i].Provenance != nil {
			ci = group[i].Provenance.Confidence
		}
		if group[j].Provenance != nil {
			cj = group[j].Provenance.Confidence
		}
		return ci > cj
	})
	return group[0]
}

// mergeDescriptions 合并所有描述
func (e *ConsolidationEngine) mergeDescriptions(group []*LogicMemory) *LogicMemory {
	// 选择置信度最高的作为基础
	keeper := e.selectHighestConfidence(group)

	// 合并描述
	descriptions := make([]string, 0, len(group))
	for _, mem := range group {
		if mem.Description != "" {
			descriptions = append(descriptions, mem.Description)
		}
	}

	// 去重并合并
	seen := make(map[string]bool)
	uniqueDescs := make([]string, 0)
	for _, desc := range descriptions {
		if !seen[desc] {
			seen[desc] = true
			uniqueDescs = append(uniqueDescs, desc)
		}
	}

	keeper.Description = strings.Join(uniqueDescs, "; ")
	return keeper
}

// ConsolidationResult 合并结果
type ConsolidationResult struct {
	// TotalMemories 处理的 Memory 总数
	TotalMemories int

	// MergedGroups 合并的组数
	MergedGroups int

	// DeletedMemories 删除的 Memory 数量
	DeletedMemories int

	// MergedMemories 合并后的 Memory 列表
	MergedMemories []*LogicMemory

	// StartTime 开始时间
	StartTime time.Time

	// EndTime 结束时间
	EndTime time.Time
}

// Duration 返回执行时长
func (r *ConsolidationResult) Duration() time.Duration {
	return r.EndTime.Sub(r.StartTime)
}

// String 返回结果摘要
func (r *ConsolidationResult) String() string {
	return fmt.Sprintf("Consolidation: %d memories processed, %d groups merged, %d memories deleted in %v",
		r.TotalMemories, r.MergedGroups, r.DeletedMemories, r.Duration())
}
