package memory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// RedundancyStrategy 冗余合并策略。
// 将多条相似度高、内容重复的记忆合并为一条。
type RedundancyStrategy struct {
	similarityThreshold float64 // 相似度阈值
}

// NewRedundancyStrategy 创建冗余合并策略。
func NewRedundancyStrategy(threshold float64) *RedundancyStrategy {
	return &RedundancyStrategy{
		similarityThreshold: threshold,
	}
}

// Name 返回策略名称。
func (s *RedundancyStrategy) Name() string {
	return "redundancy"
}

// ShouldConsolidate 判断是否应该合并。
func (s *RedundancyStrategy) ShouldConsolidate(ctx context.Context, memories []MemoryWithScore) (bool, ConsolidationReason) {
	if len(memories) < 2 {
		return false, ReasonNone
	}

	// 检查相似度
	// 如果所有记忆的相似度都很高，认为是冗余
	for i := 1; i < len(memories); i++ {
		if memories[i].Score < s.similarityThreshold {
			return false, ReasonNone
		}
	}

	return true, ReasonRedundant
}

// Consolidate 执行合并。
func (s *RedundancyStrategy) Consolidate(ctx context.Context, memories []MemoryWithScore, llm LLMProvider) (*ConsolidatedMemory, error) {
	if len(memories) == 0 {
		return nil, errors.New("no memories to consolidate")
	}

	// 构建 LLM 提示
	prompt := s.buildPrompt(memories)

	// 调用 LLM 合并
	mergedText, err := llm.Complete(ctx, prompt, map[string]any{
		"model":       "gpt-4",
		"temperature": 0.3, // 较低温度保证一致性
		"max_tokens":  500,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM consolidation failed: %w", err)
	}

	// 合并元数据
	mergedMetadata := s.mergeMetadata(memories)

	// 创建合并后的溯源
	mergedProvenance := s.mergeProvenance(memories)

	// 收集源记忆 ID
	sourceIDs := make([]string, len(memories))
	for i, mem := range memories {
		sourceIDs[i] = mem.DocID
	}

	return &ConsolidatedMemory{
		Text:           strings.TrimSpace(mergedText),
		Metadata:       mergedMetadata,
		Provenance:     mergedProvenance,
		SourceMemories: sourceIDs,
		Reason:         ReasonRedundant,
		ConsolidatedAt: time.Now(),
	}, nil
}

// buildPrompt 构建 LLM 提示。
func (s *RedundancyStrategy) buildPrompt(memories []MemoryWithScore) string {
	var sb strings.Builder

	sb.WriteString("You are a memory consolidation assistant. ")
	sb.WriteString("The following memory entries are redundant (saying similar things). ")
	sb.WriteString("Please merge them into a single, concise memory that captures all the important information.\n\n")

	sb.WriteString("Redundant Memories:\n")
	for i, mem := range memories {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, mem.Text))
	}

	sb.WriteString("\nInstructions:\n")
	sb.WriteString("- Merge the information into one clear, concise statement\n")
	sb.WriteString("- Preserve all important details\n")
	sb.WriteString("- Remove redundancy\n")
	sb.WriteString("- Keep the same tone and style\n")
	sb.WriteString("- Output only the merged memory, without explanation\n\n")
	sb.WriteString("Merged Memory:")

	return sb.String()
}

// mergeMetadata 合并元数据。
func (s *RedundancyStrategy) mergeMetadata(memories []MemoryWithScore) map[string]any {
	merged := make(map[string]any)

	// 收集所有标签
	allTags := make(map[string]bool)
	for _, mem := range memories {
		if mem.Metadata == nil {
			continue
		}
		if tags, ok := mem.Metadata["tags"].([]string); ok {
			for _, tag := range tags {
				allTags[tag] = true
			}
		}
	}

	// 转换为数组
	if len(allTags) > 0 {
		tags := make([]string, 0, len(allTags))
		for tag := range allTags {
			tags = append(tags, tag)
		}
		merged["tags"] = tags
	}

	// 记录源记忆数量
	merged["source_count"] = len(memories)
	merged["consolidation_strategy"] = "redundancy"

	return merged
}

// mergeProvenance 合并溯源信息。
func (s *RedundancyStrategy) mergeProvenance(memories []MemoryWithScore) *MemoryProvenance {
	if len(memories) == 0 {
		return nil
	}

	// 使用第一个记忆的溯源作为基础
	baseProvenance := memories[0].Provenance
	if baseProvenance == nil {
		baseProvenance = NewProvenance(SourceAgent, "unknown")
	}

	// 创建新的溯源
	merged := &MemoryProvenance{
		SourceType:         SourceAgent, // 合并后的记忆来源于 Agent
		Confidence:         baseProvenance.Confidence,
		Sources:            make([]string, 0),
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		Version:            1,
		IsExplicit:         false,
		CorroborationCount: len(memories), // 佐证数量为源记忆数量
		Tags:               []string{"consolidated", "redundancy"},
	}

	// 收集所有源
	for _, mem := range memories {
		if mem.Provenance != nil {
			merged.Sources = append(merged.Sources, mem.Provenance.Sources...)
		}
		merged.Sources = append(merged.Sources, mem.DocID)
	}

	// 提升置信度（多个来源佐证）
	if len(memories) > 1 {
		boostFactor := 1.0 + float64(len(memories)-1)*0.05 // 每个额外来源增加 5%
		merged.Confidence = min(1.0, merged.Confidence*boostFactor)
	}

	return merged
}

// ConflictResolutionStrategy 冲突解决策略。
// 处理内容相似但有矛盾的记忆。
type ConflictResolutionStrategy struct {
	similarityThreshold  float64
	confidenceCalculator *ConfidenceCalculator
}

// NewConflictResolutionStrategy 创建冲突解决策略。
func NewConflictResolutionStrategy(threshold float64, calculator *ConfidenceCalculator) *ConflictResolutionStrategy {
	return &ConflictResolutionStrategy{
		similarityThreshold:  threshold,
		confidenceCalculator: calculator,
	}
}

// Name 返回策略名称。
func (s *ConflictResolutionStrategy) Name() string {
	return "conflict-resolution"
}

// ShouldConsolidate 判断是否应该合并。
func (s *ConflictResolutionStrategy) ShouldConsolidate(ctx context.Context, memories []MemoryWithScore) (bool, ConsolidationReason) {
	if len(memories) < 2 {
		return false, ReasonNone
	}

	// 检测冲突：相似但不完全一致
	hasConflict := s.detectConflict(memories)
	if !hasConflict {
		return false, ReasonNone
	}

	return true, ReasonConflict
}

// detectConflict 检测是否存在冲突。
func (s *ConflictResolutionStrategy) detectConflict(memories []MemoryWithScore) bool {
	// 简化逻辑：检查文本是否有显著差异
	// 真实场景可能需要更复杂的语义分析
	if len(memories) < 2 {
		return false
	}

	// 检查是否包含矛盾的关键词
	conflictKeywords := []string{"but", "however", "actually", "not", "no", "incorrect", "wrong"}

	for _, mem := range memories {
		textLower := strings.ToLower(mem.Text)
		for _, keyword := range conflictKeywords {
			if strings.Contains(textLower, keyword) {
				return true
			}
		}
	}

	return false
}

// Consolidate 执行合并。
func (s *ConflictResolutionStrategy) Consolidate(ctx context.Context, memories []MemoryWithScore, llm LLMProvider) (*ConsolidatedMemory, error) {
	if len(memories) == 0 {
		return nil, errors.New("no memories to consolidate")
	}

	// 构建冲突解决提示
	prompt := s.buildConflictResolutionPrompt(memories)

	// 调用 LLM 解决冲突
	resolvedText, err := llm.Complete(ctx, prompt, map[string]any{
		"model":       "gpt-4",
		"temperature": 0.2, // 低温度保证客观性
		"max_tokens":  600,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM conflict resolution failed: %w", err)
	}

	// 选择最高置信度的溯源
	bestProvenance := s.selectBestProvenance(memories)

	// 收集源记忆 ID
	sourceIDs := make([]string, len(memories))
	for i, mem := range memories {
		sourceIDs[i] = mem.DocID
	}

	return &ConsolidatedMemory{
		Text:           strings.TrimSpace(resolvedText),
		Metadata:       map[string]any{"consolidation_strategy": "conflict-resolution"},
		Provenance:     bestProvenance,
		SourceMemories: sourceIDs,
		Reason:         ReasonConflict,
		ConsolidatedAt: time.Now(),
	}, nil
}

// buildConflictResolutionPrompt 构建冲突解决提示。
func (s *ConflictResolutionStrategy) buildConflictResolutionPrompt(memories []MemoryWithScore) string {
	var sb strings.Builder

	sb.WriteString("You are a memory conflict resolution assistant. ")
	sb.WriteString("The following memory entries contain conflicting information. ")
	sb.WriteString("Please analyze them and create a single, accurate memory.\n\n")

	sb.WriteString("Conflicting Memories:\n")
	for i, mem := range memories {
		confidence := "unknown"
		if mem.Provenance != nil {
			confidence = fmt.Sprintf("%.2f", mem.Provenance.Confidence)
		}
		sb.WriteString(fmt.Sprintf("%d. %s (Confidence: %s)\n", i+1, mem.Text, confidence))
	}

	sb.WriteString("\nInstructions:\n")
	sb.WriteString("- Analyze the conflicts carefully\n")
	sb.WriteString("- Prefer information from higher confidence sources\n")
	sb.WriteString("- If information is contradictory, indicate uncertainty\n")
	sb.WriteString("- Provide a balanced, objective statement\n")
	sb.WriteString("- Output only the resolved memory, without explanation\n\n")
	sb.WriteString("Resolved Memory:")

	return sb.String()
}

// selectBestProvenance 选择最佳溯源。
func (s *ConflictResolutionStrategy) selectBestProvenance(memories []MemoryWithScore) *MemoryProvenance {
	var best *MemoryProvenance
	maxConfidence := 0.0

	for _, mem := range memories {
		if mem.Provenance == nil {
			continue
		}

		// 计算当前置信度
		confidence := mem.Provenance.Confidence
		if s.confidenceCalculator != nil {
			confidence = s.confidenceCalculator.Calculate(mem.Provenance)
		}

		if confidence > maxConfidence {
			maxConfidence = confidence
			best = mem.Provenance
		}
	}

	if best == nil {
		return NewProvenance(SourceAgent, "conflict-resolution")
	}

	// 创建副本并标记为冲突解决
	resolved := *best
	resolved.Tags = append(resolved.Tags, "conflict-resolved")
	resolved.UpdatedAt = time.Now()

	return &resolved
}

// SummarizationStrategy 总结策略。
// 将多条相关记忆总结为更简洁的表述。
type SummarizationStrategy struct {
	maxMemoriesPerGroup int
}

// NewSummarizationStrategy 创建总结策略。
func NewSummarizationStrategy(maxPerGroup int) *SummarizationStrategy {
	return &SummarizationStrategy{
		maxMemoriesPerGroup: maxPerGroup,
	}
}

// Name 返回策略名称。
func (s *SummarizationStrategy) Name() string {
	return "summarization"
}

// ShouldConsolidate 判断是否应该合并。
func (s *SummarizationStrategy) ShouldConsolidate(ctx context.Context, memories []MemoryWithScore) (bool, ConsolidationReason) {
	// 只有当记忆数量超过阈值时才总结
	if len(memories) < s.maxMemoriesPerGroup {
		return false, ReasonNone
	}

	return true, ReasonSummary
}

// Consolidate 执行合并。
func (s *SummarizationStrategy) Consolidate(ctx context.Context, memories []MemoryWithScore, llm LLMProvider) (*ConsolidatedMemory, error) {
	if len(memories) == 0 {
		return nil, errors.New("no memories to consolidate")
	}

	// 构建总结提示
	prompt := s.buildSummarizationPrompt(memories)

	// 调用 LLM 总结
	summary, err := llm.Complete(ctx, prompt, map[string]any{
		"model":       "gpt-4",
		"temperature": 0.4, // 中等温度允许一定的创造性总结
		"max_tokens":  400,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM summarization failed: %w", err)
	}

	// 合并溯源
	mergedProvenance := s.mergeProvenance(memories)

	// 收集源记忆 ID
	sourceIDs := make([]string, len(memories))
	for i, mem := range memories {
		sourceIDs[i] = mem.DocID
	}

	return &ConsolidatedMemory{
		Text:           strings.TrimSpace(summary),
		Metadata:       map[string]any{"consolidation_strategy": "summarization"},
		Provenance:     mergedProvenance,
		SourceMemories: sourceIDs,
		Reason:         ReasonSummary,
		ConsolidatedAt: time.Now(),
	}, nil
}

// buildSummarizationPrompt 构建总结提示。
func (s *SummarizationStrategy) buildSummarizationPrompt(memories []MemoryWithScore) string {
	var sb strings.Builder

	sb.WriteString("You are a memory summarization assistant. ")
	sb.WriteString("Please create a concise summary of the following related memories.\n\n")

	sb.WriteString("Memories to Summarize:\n")
	for i, mem := range memories {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, mem.Text))
	}

	sb.WriteString("\nInstructions:\n")
	sb.WriteString("- Create a brief, comprehensive summary\n")
	sb.WriteString("- Capture the key information from all memories\n")
	sb.WriteString("- Be concise but complete\n")
	sb.WriteString("- Maintain factual accuracy\n")
	sb.WriteString("- Output only the summary, without explanation\n\n")
	sb.WriteString("Summary:")

	return sb.String()
}

// mergeProvenance 合并溯源。
func (s *SummarizationStrategy) mergeProvenance(memories []MemoryWithScore) *MemoryProvenance {
	if len(memories) == 0 {
		return nil
	}

	// 计算平均置信度
	totalConfidence := 0.0
	for _, mem := range memories {
		if mem.Provenance != nil {
			totalConfidence += mem.Provenance.Confidence
		}
	}
	avgConfidence := totalConfidence / float64(len(memories))

	// 收集所有源
	sources := make([]string, 0)
	for _, mem := range memories {
		if mem.Provenance != nil {
			sources = append(sources, mem.Provenance.Sources...)
		}
		sources = append(sources, mem.DocID)
	}

	return &MemoryProvenance{
		SourceType:         SourceAgent,
		Confidence:         avgConfidence,
		Sources:            sources,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		Version:            1,
		IsExplicit:         false,
		CorroborationCount: len(memories),
		Tags:               []string{"summarized"},
	}
}
