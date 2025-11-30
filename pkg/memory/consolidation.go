package memory

import (
	"context"
	"fmt"
	"time"
)

// ConsolidationEngine 内存合并引擎。
// 负责检测和合并冗余或冲突的记忆。
type ConsolidationEngine struct {
	memory              *SemanticMemory
	strategy            ConsolidationStrategy
	llmProvider         LLMProvider
	config              ConsolidationConfig
	lastConsolidation   time.Time
	consolidationCount  int64
	mergedMemoriesCount int64
}

// ConsolidationConfig 合并引擎配置。
type ConsolidationConfig struct {
	// 相似度阈值（超过此值认为是冗余）
	SimilarityThreshold float64

	// 冲突检测阈值（语义相似但内容矛盾）
	ConflictThreshold float64

	// 最小记忆数量（少于此数量不触发合并）
	MinMemoryCount int

	// 批处理大小（每次处理的记忆数量）
	BatchSize int

	// 自动合并间隔
	AutoConsolidateInterval time.Duration

	// 是否保留原始记忆（合并后标记为已合并，而不是删除）
	PreserveOriginal bool

	// LLM 模型名称
	LLMModel string

	// 最大重试次数
	MaxRetries int
}

// DefaultConsolidationConfig 返回默认配置。
func DefaultConsolidationConfig() ConsolidationConfig {
	return ConsolidationConfig{
		SimilarityThreshold:     0.85, // 85% 相似度认为冗余
		ConflictThreshold:       0.75, // 75% 相似度但内容不同认为冲突
		MinMemoryCount:          10,   // 至少 10 条记忆才触发
		BatchSize:               50,   // 每次处理 50 条
		AutoConsolidateInterval: 24 * time.Hour,
		PreserveOriginal:        true,
		LLMModel:                "gpt-4",
		MaxRetries:              3,
	}
}

// LLMProvider 提供 LLM 调用能力。
type LLMProvider interface {
	// Complete 完成文本生成
	Complete(ctx context.Context, prompt string, options map[string]any) (string, error)
}

// ConsolidationStrategy 合并策略接口。
type ConsolidationStrategy interface {
	// Name 返回策略名称
	Name() string

	// ShouldConsolidate 判断是否应该合并这些记忆
	ShouldConsolidate(ctx context.Context, memories []MemoryWithScore) (bool, ConsolidationReason)

	// Consolidate 执行合并
	Consolidate(ctx context.Context, memories []MemoryWithScore, llm LLMProvider) (*ConsolidatedMemory, error)
}

// ConsolidationReason 合并原因。
type ConsolidationReason string

const (
	ReasonRedundant ConsolidationReason = "redundant" // 冗余
	ReasonConflict  ConsolidationReason = "conflict"  // 冲突
	ReasonSummary   ConsolidationReason = "summary"   // 总结
	ReasonNone      ConsolidationReason = "none"      // 不需要合并
)

// MemoryWithScore 带相似度分数的记忆。
type MemoryWithScore struct {
	DocID      string
	Text       string
	Metadata   map[string]any
	Provenance *MemoryProvenance
	Score      float64 // 与查询的相似度
}

// ConsolidatedMemory 合并后的记忆。
type ConsolidatedMemory struct {
	Text           string              // 合并后的文本
	Metadata       map[string]any      // 合并后的元数据
	Provenance     *MemoryProvenance   // 合并后的溯源
	SourceMemories []string            // 源记忆 ID 列表
	Reason         ConsolidationReason // 合并原因
	ConsolidatedAt time.Time           // 合并时间
}

// NewConsolidationEngine 创建合并引擎。
func NewConsolidationEngine(
	memory *SemanticMemory,
	strategy ConsolidationStrategy,
	llmProvider LLMProvider,
	config ConsolidationConfig,
) *ConsolidationEngine {
	return &ConsolidationEngine{
		memory:            memory,
		strategy:          strategy,
		llmProvider:       llmProvider,
		config:            config,
		lastConsolidation: time.Now(),
	}
}

// Consolidate 执行内存合并。
func (ce *ConsolidationEngine) Consolidate(ctx context.Context) (*ConsolidationResult, error) {
	startTime := time.Now()

	// 检查是否满足最小记忆数量
	// TODO: 实现获取记忆总数的方法
	// count := ce.memory.Count(ctx)
	// if count < ce.config.MinMemoryCount {
	// 	return &ConsolidationResult{
	// 		Success: false,
	// 		Message: fmt.Sprintf("Memory count %d below threshold %d", count, ce.config.MinMemoryCount),
	// 	}, nil
	// }

	result := &ConsolidationResult{
		StartTime:    startTime,
		Strategy:     ce.strategy.Name(),
		MemoryGroups: make([]MemoryGroup, 0),
	}

	// 查找候选记忆组（相似的记忆）
	groups, err := ce.findCandidateGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find candidate groups: %w", err)
	}

	// 对每组记忆进行合并
	for _, group := range groups {
		// 检查是否应该合并
		shouldConsolidate, reason := ce.strategy.ShouldConsolidate(ctx, group.Memories)
		if !shouldConsolidate {
			continue
		}

		// 执行合并
		consolidated, err := ce.strategy.Consolidate(ctx, group.Memories, ce.llmProvider)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to consolidate group: %v", err))
			continue
		}

		// 保存合并后的记忆
		newDocID, err := ce.saveConsolidatedMemory(ctx, consolidated)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to save consolidated memory: %v", err))
			continue
		}

		// 处理源记忆（删除或标记）
		if err := ce.handleSourceMemories(ctx, group.Memories, newDocID); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to handle source memories: %v", err))
			continue
		}

		// 记录结果
		result.MemoryGroups = append(result.MemoryGroups, MemoryGroup{
			Memories:       group.Memories,
			ConsolidatedID: newDocID,
			Reason:         reason,
		})

		result.MergedCount += len(group.Memories)
		result.NewMemoryCount++
	}

	// 更新统计
	ce.lastConsolidation = time.Now()
	ce.consolidationCount++
	ce.mergedMemoriesCount += int64(result.MergedCount)

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Success = len(result.Errors) == 0

	return result, nil
}

// findCandidateGroups 查找候选记忆组。
func (ce *ConsolidationEngine) findCandidateGroups(ctx context.Context) ([]MemoryGroup, error) {
	// TODO: 实现查找相似记忆的逻辑
	// 1. 获取所有记忆
	// 2. 计算相似度矩阵
	// 3. 聚类相似的记忆

	// 暂时返回空列表
	return []MemoryGroup{}, nil
}

// saveConsolidatedMemory 保存合并后的记忆。
func (ce *ConsolidationEngine) saveConsolidatedMemory(ctx context.Context, consolidated *ConsolidatedMemory) (string, error) {
	// 生成新的 DocID
	docID := fmt.Sprintf("consolidated_%d", time.Now().UnixNano())

	// 创建溯源信息
	provenance := consolidated.Provenance
	if provenance == nil {
		provenance = NewProvenance(SourceAgent, "consolidation-engine")
	}

	// 保存到语义内存
	if err := ce.memory.IndexWithProvenance(
		ctx,
		docID,
		consolidated.Text,
		consolidated.Metadata,
		provenance,
		consolidated.SourceMemories, // 溯源链接
	); err != nil {
		return "", err
	}

	return docID, nil
}

// handleSourceMemories 处理源记忆。
func (ce *ConsolidationEngine) handleSourceMemories(ctx context.Context, memories []MemoryWithScore, consolidatedID string) error {
	for _, mem := range memories {
		if ce.config.PreserveOriginal {
			// 标记为已合并
			metadata := mem.Metadata
			if metadata == nil {
				metadata = make(map[string]any)
			}
			metadata["consolidated"] = true
			metadata["consolidated_to"] = consolidatedID
			metadata["consolidated_at"] = time.Now().Format(time.RFC3339)

			// 更新元数据
			if err := ce.memory.UpdateMetadata(ctx, mem.DocID, metadata); err != nil {
				return fmt.Errorf("failed to update metadata for %s: %w", mem.DocID, err)
			}
		} else {
			// 删除源记忆
			if err := ce.memory.Delete(ctx, mem.DocID); err != nil {
				return fmt.Errorf("failed to delete %s: %w", mem.DocID, err)
			}
		}
	}
	return nil
}

// ShouldAutoConsolidate 检查是否应该自动触发合并。
func (ce *ConsolidationEngine) ShouldAutoConsolidate() bool {
	return time.Since(ce.lastConsolidation) >= ce.config.AutoConsolidateInterval
}

// GetStats 获取合并引擎统计信息。
func (ce *ConsolidationEngine) GetStats() ConsolidationStats {
	return ConsolidationStats{
		LastConsolidation:   ce.lastConsolidation,
		ConsolidationCount:  ce.consolidationCount,
		MergedMemoriesCount: ce.mergedMemoriesCount,
	}
}

// ConsolidationResult 合并结果。
type ConsolidationResult struct {
	Success        bool
	Message        string
	StartTime      time.Time
	EndTime        time.Time
	Duration       time.Duration
	Strategy       string
	MemoryGroups   []MemoryGroup
	MergedCount    int // 合并的记忆数量
	NewMemoryCount int // 生成的新记忆数量
	Errors         []string
}

// MemoryGroup 记忆组。
type MemoryGroup struct {
	Memories       []MemoryWithScore
	ConsolidatedID string
	Reason         ConsolidationReason
}

// ConsolidationStats 合并统计。
type ConsolidationStats struct {
	LastConsolidation   time.Time
	ConsolidationCount  int64
	MergedMemoriesCount int64
}
