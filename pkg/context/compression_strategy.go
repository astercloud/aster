package context

import (
	"context"
	"fmt"
	"sort"
)

// CompressionStrategy 定义上下文压缩策略接口
type CompressionStrategy interface {
	// Name 返回策略名称
	Name() string

	// Compress 压缩消息列表
	// 返回压缩后的消息列表
	Compress(ctx context.Context, messages []Message, config WindowManagerConfig) ([]Message, error)
}

// SlidingWindowStrategy 滑动窗口策略
// 保留最近的 N 条消息，删除旧消息
type SlidingWindowStrategy struct {
	windowSize int // 窗口大小（消息数量）
}

// NewSlidingWindowStrategy 创建滑动窗口策略
func NewSlidingWindowStrategy(windowSize int) *SlidingWindowStrategy {
	if windowSize <= 0 {
		windowSize = 10 // 默认保留 10 条消息
	}
	return &SlidingWindowStrategy{
		windowSize: windowSize,
	}
}

// Name 实现 CompressionStrategy 接口
func (s *SlidingWindowStrategy) Name() string {
	return "sliding-window"
}

// Compress 实现 CompressionStrategy 接口
func (s *SlidingWindowStrategy) Compress(
	ctx context.Context,
	messages []Message,
	config WindowManagerConfig,
) ([]Message, error) {
	if len(messages) <= s.windowSize {
		// 不需要压缩
		return messages, nil
	}

	// 保留的消息索引
	keepIndices := make(map[int]bool)

	// 1. 始终保留 system 消息
	if config.AlwaysKeepSystem {
		for i, msg := range messages {
			if msg.Role == "system" {
				keepIndices[i] = true
			}
		}
	}

	// 2. 始终保留最近的 N 条消息
	recentCount := min(config.AlwaysKeepRecent, len(messages))
	for i := len(messages) - recentCount; i < len(messages); i++ {
		keepIndices[i] = true
	}

	// 3. 从剩余消息中选择最新的填充到窗口大小
	remainingSlots := s.windowSize - len(keepIndices)
	if remainingSlots > 0 {
		for i := len(messages) - 1; i >= 0 && remainingSlots > 0; i-- {
			if !keepIndices[i] {
				keepIndices[i] = true
				remainingSlots--
			}
		}
	}

	// 构建结果（保持原始顺序）
	result := []Message{}
	for i, msg := range messages {
		if keepIndices[i] {
			result = append(result, msg)
		}
	}

	return result, nil
}

// PriorityBasedStrategy 基于优先级的压缩策略
// 优先保留高优先级消息
type PriorityBasedStrategy struct {
	targetSize         int                // 目标消息数
	priorityCalculator PriorityCalculator // 优先级计算器
}

// NewPriorityBasedStrategy 创建基于优先级的策略
func NewPriorityBasedStrategy(targetSize int, calculator PriorityCalculator) *PriorityBasedStrategy {
	if targetSize <= 0 {
		targetSize = 10
	}
	if calculator == nil {
		calculator = NewDefaultPriorityCalculator()
	}
	return &PriorityBasedStrategy{
		targetSize:         targetSize,
		priorityCalculator: calculator,
	}
}

// Name 实现 CompressionStrategy 接口
func (s *PriorityBasedStrategy) Name() string {
	return "priority-based"
}

// Compress 实现 CompressionStrategy 接口
func (s *PriorityBasedStrategy) Compress(
	ctx context.Context,
	messages []Message,
	config WindowManagerConfig,
) ([]Message, error) {
	if len(messages) <= s.targetSize {
		return messages, nil
	}

	// 计算所有消息的优先级
	withPriority := CalculateMessagePriorities(ctx, messages, s.priorityCalculator)

	// 保留的消息索引
	keepIndices := make(map[int]bool)

	// 1. 始终保留 system 消息
	if config.AlwaysKeepSystem {
		for i, msg := range messages {
			if msg.Role == "system" {
				keepIndices[i] = true
			}
		}
	}

	// 2. 始终保留最近的 N 条消息
	recentCount := min(config.AlwaysKeepRecent, len(messages))
	for i := len(messages) - recentCount; i < len(messages); i++ {
		keepIndices[i] = true
	}

	// 3. 按优先级排序，选择最高优先级的消息
	SortMessagesByPriority(withPriority, true) // 降序排序

	remainingSlots := s.targetSize - len(keepIndices)
	for _, msgWithPri := range withPriority {
		if remainingSlots <= 0 {
			break
		}

		// 找到该消息的原始索引
		for i, msg := range messages {
			if msg.Role == msgWithPri.Message.Role && msg.Content == msgWithPri.Message.Content {
				if !keepIndices[i] {
					keepIndices[i] = true
					remainingSlots--
					break
				}
			}
		}
	}

	// 构建结果（保持原始顺序）
	result := []Message{}
	for i, msg := range messages {
		if keepIndices[i] {
			result = append(result, msg)
		}
	}

	return result, nil
}

// TokenBasedStrategy 基于 Token 预算的压缩策略
// 从最旧的消息开始删除，直到满足 Token 预算
type TokenBasedStrategy struct {
	tokenCounter TokenCounter
	targetUsage  float64 // 目标使用率（0.0-1.0）
}

// NewTokenBasedStrategy 创建基于 Token 的策略
func NewTokenBasedStrategy(tokenCounter TokenCounter, targetUsage float64) *TokenBasedStrategy {
	if targetUsage <= 0 || targetUsage > 1.0 {
		targetUsage = 0.7 // 默认目标 70%
	}
	return &TokenBasedStrategy{
		tokenCounter: tokenCounter,
		targetUsage:  targetUsage,
	}
}

// Name 实现 CompressionStrategy 接口
func (s *TokenBasedStrategy) Name() string {
	return "token-based"
}

// Compress 实现 CompressionStrategy 接口
func (s *TokenBasedStrategy) Compress(
	ctx context.Context,
	messages []Message,
	config WindowManagerConfig,
) ([]Message, error) {
	// 计算目标 Token 数
	targetTokens := int(float64(config.Budget.AvailableTokens()) * s.targetUsage)

	// 计算当前 Token 数
	currentTokens, err := s.tokenCounter.EstimateMessages(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate current tokens: %w", err)
	}

	// 如果已经在目标范围内，不需要压缩
	if currentTokens <= targetTokens {
		return messages, nil
	}

	// 保留的消息索引（从后往前）
	keepIndices := make(map[int]bool)

	// 1. 始终保留 system 消息
	if config.AlwaysKeepSystem {
		for i, msg := range messages {
			if msg.Role == "system" {
				keepIndices[i] = true
			}
		}
	}

	// 2. 始终保留最近的 N 条消息
	recentCount := min(config.AlwaysKeepRecent, len(messages))
	for i := len(messages) - recentCount; i < len(messages); i++ {
		keepIndices[i] = true
	}

	// 3. 从后往前逐步添加消息，直到达到 Token 目标
	for i := len(messages) - 1; i >= 0; i-- {
		if keepIndices[i] {
			continue // 已经保留
		}

		// 尝试添加这条消息
		testIndices := make(map[int]bool)
		for k := range keepIndices {
			testIndices[k] = true
		}
		testIndices[i] = true

		// 构建测试消息列表
		testMessages := []Message{}
		for j := range messages {
			if testIndices[j] {
				testMessages = append(testMessages, messages[j])
			}
		}

		// 计算 Token 数
		testTokens, err := s.tokenCounter.EstimateMessages(ctx, testMessages)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate tokens: %w", err)
		}

		// 如果超过目标，停止添加
		if testTokens > targetTokens {
			break
		}

		// 否则保留这条消息
		keepIndices[i] = true
	}

	// 构建结果（保持原始顺序）
	result := []Message{}
	for i, msg := range messages {
		if keepIndices[i] {
			result = append(result, msg)
		}
	}

	// 确保至少保留最小消息数
	if len(result) < config.MinMessagesToKeep && len(messages) > 0 {
		// 强制保留最近的消息
		result = messages[len(messages)-config.MinMessagesToKeep:]
	}

	return result, nil
}

// HybridStrategy 混合策略
// 结合多种策略的优点
type HybridStrategy struct {
	strategies []CompressionStrategy
	weights    []float64
}

// NewHybridStrategy 创建混合策略
func NewHybridStrategy(strategies []CompressionStrategy, weights []float64) *HybridStrategy {
	if len(strategies) != len(weights) {
		// 使用均等权重
		weights = make([]float64, len(strategies))
		for i := range weights {
			weights[i] = 1.0 / float64(len(strategies))
		}
	}

	return &HybridStrategy{
		strategies: strategies,
		weights:    weights,
	}
}

// Name 实现 CompressionStrategy 接口
func (s *HybridStrategy) Name() string {
	return "hybrid"
}

// Compress 实现 CompressionStrategy 接口
func (s *HybridStrategy) Compress(
	ctx context.Context,
	messages []Message,
	config WindowManagerConfig,
) ([]Message, error) {
	if len(s.strategies) == 0 {
		return messages, nil
	}

	// 为每条消息计算综合分数
	messageScores := make(map[int]float64)

	for strategyIdx, strategy := range s.strategies {
		// 使用该策略压缩
		compressed, err := strategy.Compress(ctx, messages, config)
		if err != nil {
			continue // 跳过失败的策略
		}

		// 为保留的消息增加分数
		compressedSet := make(map[string]bool)
		for _, msg := range compressed {
			key := msg.Role + ":" + msg.Content
			compressedSet[key] = true
		}

		for i, msg := range messages {
			key := msg.Role + ":" + msg.Content
			if compressedSet[key] {
				messageScores[i] += s.weights[strategyIdx]
			}
		}
	}

	// 按分数排序消息索引
	type indexScore struct {
		index int
		score float64
	}
	scores := []indexScore{}
	for i, score := range messageScores {
		scores = append(scores, indexScore{index: i, score: score})
	}
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// 选择得分最高的消息（最多保留原始数量的一半）
	targetSize := max(len(messages)/2, config.MinMessagesToKeep)

	keepIndices := make(map[int]bool)
	for i := 0; i < targetSize && i < len(scores); i++ {
		keepIndices[scores[i].index] = true
	}

	// 确保保留 system 和最近消息
	if config.AlwaysKeepSystem {
		for i, msg := range messages {
			if msg.Role == "system" {
				keepIndices[i] = true
			}
		}
	}

	recentCount := config.AlwaysKeepRecent
	for i := len(messages) - recentCount; i < len(messages); i++ {
		if i >= 0 {
			keepIndices[i] = true
		}
	}

	// 构建结果
	result := []Message{}
	for i, msg := range messages {
		if keepIndices[i] {
			result = append(result, msg)
		}
	}

	return result, nil
}
