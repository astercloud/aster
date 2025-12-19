package context

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

// ContextWindowManager 管理对话上下文窗口
// 实现自动 Token 计数、滑动窗口和智能压缩
type ContextWindowManager struct {
	mu sync.RWMutex

	// 配置
	config WindowManagerConfig

	// Token 计数器
	tokenCounter TokenCounter

	// 压缩策略
	compressionStrategy CompressionStrategy

	// 消息历史
	messages []Message

	// Token 使用跟踪
	currentTokens int
	totalMessages int

	// 压缩历史
	compressionHistory []CompressionEvent
}

// WindowManagerConfig 窗口管理器配置
type WindowManagerConfig struct {
	// Token 预算
	Budget TokenBudget

	// 自动压缩配置
	AutoCompress         bool    // 是否启用自动压缩
	CompressionThreshold float64 // 触发压缩的使用率阈值 (0.0-1.0)

	// 消息保留策略
	MinMessagesToKeep int  // 最少保留的消息数
	AlwaysKeepSystem  bool // 始终保留 system 消息
	AlwaysKeepRecent  int  // 始终保留最近的 N 条消息

	// 优先级配置
	EnablePrioritization bool // 是否启用消息优先级
}

// DefaultWindowManagerConfig 返回默认配置
func DefaultWindowManagerConfig() WindowManagerConfig {
	return WindowManagerConfig{
		Budget:               DefaultTokenBudget(),
		AutoCompress:         true,
		CompressionThreshold: 0.85, // 85% 时触发压缩
		MinMessagesToKeep:    3,
		AlwaysKeepSystem:     true,
		AlwaysKeepRecent:     2,
		EnablePrioritization: true,
	}
}

// CompressionEvent 记录一次压缩事件
type CompressionEvent struct {
	Timestamp        time.Time
	BeforeMessages   int
	AfterMessages    int
	BeforeTokens     int
	AfterTokens      int
	CompressionRatio float64
	Strategy         string
}

// NewContextWindowManager 创建新的上下文窗口管理器
func NewContextWindowManager(
	config WindowManagerConfig,
	tokenCounter TokenCounter,
	compressionStrategy CompressionStrategy,
) *ContextWindowManager {
	return &ContextWindowManager{
		config:              config,
		tokenCounter:        tokenCounter,
		compressionStrategy: compressionStrategy,
		messages:            []Message{},
		compressionHistory:  []CompressionEvent{},
	}
}

// AddMessage 添加新消息
func (m *ContextWindowManager) AddMessage(ctx context.Context, msg Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 添加消息
	m.messages = append(m.messages, msg)
	m.totalMessages++

	// 重新计算 Token 数
	if err := m.recalculateTokens(ctx); err != nil {
		return fmt.Errorf("failed to recalculate tokens: %w", err)
	}

	// 检查是否需要压缩
	if m.config.AutoCompress && m.shouldCompress() {
		if err := m.compress(ctx); err != nil {
			return fmt.Errorf("auto-compression failed: %w", err)
		}
	}

	return nil
}

// AddMessages 批量添加消息
func (m *ContextWindowManager) AddMessages(ctx context.Context, msgs []Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 添加消息
	m.messages = append(m.messages, msgs...)
	m.totalMessages += len(msgs)

	// 重新计算 Token 数
	if err := m.recalculateTokens(ctx); err != nil {
		return fmt.Errorf("failed to recalculate tokens: %w", err)
	}

	// 检查是否需要压缩
	if m.config.AutoCompress && m.shouldCompress() {
		if err := m.compress(ctx); err != nil {
			return fmt.Errorf("auto-compression failed: %w", err)
		}
	}

	return nil
}

// GetMessages 获取当前所有消息
func (m *ContextWindowManager) GetMessages() []Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 返回副本
	result := make([]Message, len(m.messages))
	copy(result, m.messages)
	return result
}

// GetCurrentTokens 获取当前 Token 使用量
func (m *ContextWindowManager) GetCurrentTokens() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentTokens
}

// GetRemainingTokens 获取剩余可用 Token 数
func (m *ContextWindowManager) GetRemainingTokens() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Budget.RemainingTokens(m.currentTokens)
}

// GetUsagePercentage 获取当前使用百分比
func (m *ContextWindowManager) GetUsagePercentage() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Budget.UsagePercentage(m.currentTokens)
}

// IsWithinBudget 检查是否在预算内
func (m *ContextWindowManager) IsWithinBudget() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Budget.IsWithinBudget(m.currentTokens)
}

// ShouldWarn 检查是否应该发出警告
func (m *ContextWindowManager) ShouldWarn() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Budget.ShouldWarn(m.currentTokens)
}

// Compress 手动触发压缩
func (m *ContextWindowManager) Compress(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.compress(ctx)
}

// compress 内部压缩方法（需要持有锁）
func (m *ContextWindowManager) compress(ctx context.Context) error {
	if m.compressionStrategy == nil {
		return errors.New("compression strategy not configured")
	}

	beforeMessages := len(m.messages)
	beforeTokens := m.currentTokens

	// 执行压缩
	compressed, err := m.compressionStrategy.Compress(ctx, m.messages, m.config)
	if err != nil {
		return fmt.Errorf("compression failed: %w", err)
	}

	// 更新消息列表
	m.messages = compressed

	// 重新计算 Token 数
	if err := m.recalculateTokens(ctx); err != nil {
		return fmt.Errorf("failed to recalculate tokens after compression: %w", err)
	}

	afterMessages := len(m.messages)
	afterTokens := m.currentTokens

	// 记录压缩事件
	event := CompressionEvent{
		Timestamp:      time.Now(),
		BeforeMessages: beforeMessages,
		AfterMessages:  afterMessages,
		BeforeTokens:   beforeTokens,
		AfterTokens:    afterTokens,
		Strategy:       m.compressionStrategy.Name(),
	}

	if beforeTokens > 0 {
		event.CompressionRatio = float64(afterTokens) / float64(beforeTokens)
	}

	m.compressionHistory = append(m.compressionHistory, event)

	return nil
}

// shouldCompress 检查是否应该触发压缩（需要持有读锁）
func (m *ContextWindowManager) shouldCompress() bool {
	if !m.config.AutoCompress {
		return false
	}

	if m.compressionStrategy == nil {
		return false
	}

	// 检查 Token 使用率
	usage := m.config.Budget.UsagePercentage(m.currentTokens)
	return usage >= m.config.CompressionThreshold*100.0
}

// recalculateTokens 重新计算 Token 数（需要持有锁）
func (m *ContextWindowManager) recalculateTokens(ctx context.Context) error {
	tokens, err := m.tokenCounter.EstimateMessages(ctx, m.messages)
	if err != nil {
		return err
	}
	m.currentTokens = tokens
	return nil
}

// GetCompressionHistory 获取压缩历史
func (m *ContextWindowManager) GetCompressionHistory() []CompressionEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]CompressionEvent, len(m.compressionHistory))
	copy(result, m.compressionHistory)
	return result
}

// Clear 清空所有消息
func (m *ContextWindowManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = []Message{}
	m.currentTokens = 0
}

// Reset 重置管理器（包括历史记录）
func (m *ContextWindowManager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = []Message{}
	m.currentTokens = 0
	m.totalMessages = 0
	m.compressionHistory = []CompressionEvent{}
}

// GetStats 获取统计信息
func (m *ContextWindowManager) GetStats() WindowStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return WindowStats{
		CurrentMessages:  len(m.messages),
		TotalMessages:    m.totalMessages,
		CurrentTokens:    m.currentTokens,
		RemainingTokens:  m.config.Budget.RemainingTokens(m.currentTokens),
		UsagePercentage:  m.config.Budget.UsagePercentage(m.currentTokens),
		CompressionCount: len(m.compressionHistory),
		IsWithinBudget:   m.config.Budget.IsWithinBudget(m.currentTokens),
		ShouldWarn:       m.config.Budget.ShouldWarn(m.currentTokens),
	}
}

// WindowStats 窗口统计信息
type WindowStats struct {
	CurrentMessages  int     // 当前消息数
	TotalMessages    int     // 总共添加的消息数
	CurrentTokens    int     // 当前 Token 使用量
	RemainingTokens  int     // 剩余可用 Token
	UsagePercentage  float64 // 使用百分比
	CompressionCount int     // 压缩次数
	IsWithinBudget   bool    // 是否在预算内
	ShouldWarn       bool    // 是否应该警告
}

// MessagePriority 消息优先级
type MessagePriority int

const (
	PriorityLow      MessagePriority = 1
	PriorityMedium   MessagePriority = 2
	PriorityHigh     MessagePriority = 3
	PriorityCritical MessagePriority = 4
)

// MessageWithPriority 带优先级的消息
type MessageWithPriority struct {
	Message  Message
	Priority MessagePriority
	Score    float64 // 优先级分数（用于排序）
}

// PriorityCalculator 优先级计算器接口
type PriorityCalculator interface {
	// CalculatePriority 计算消息的优先级和分数
	CalculatePriority(ctx context.Context, msg Message, position int, totalMessages int) (MessagePriority, float64)
}

// DefaultPriorityCalculator 默认优先级计算器
type DefaultPriorityCalculator struct {
	recencyWeight float64 // 最近度权重
	roleWeight    float64 // 角色权重
	lengthWeight  float64 // 长度权重
}

// NewDefaultPriorityCalculator 创建默认优先级计算器
func NewDefaultPriorityCalculator() *DefaultPriorityCalculator {
	return &DefaultPriorityCalculator{
		recencyWeight: 0.5,
		roleWeight:    0.3,
		lengthWeight:  0.2,
	}
}

// CalculatePriority 实现 PriorityCalculator 接口
func (c *DefaultPriorityCalculator) CalculatePriority(
	ctx context.Context,
	msg Message,
	position int,
	totalMessages int,
) (MessagePriority, float64) {
	score := 0.0

	// 1. 最近度得分（越新越高）
	if totalMessages > 0 {
		recencyScore := float64(position) / float64(totalMessages)
		score += recencyScore * c.recencyWeight
	}

	// 2. 角色得分
	roleScore := 0.0
	switch msg.Role {
	case "system":
		roleScore = 1.0 // system 消息最重要
	case "assistant":
		roleScore = 0.7
	case "user":
		roleScore = 0.8
	}
	score += roleScore * c.roleWeight

	// 3. 长度得分（较长的消息可能包含更多信息）
	lengthScore := float64(len(msg.Content)) / 1000.0
	if lengthScore > 1.0 {
		lengthScore = 1.0
	}
	score += lengthScore * c.lengthWeight

	// 根据分数确定优先级
	var priority MessagePriority
	if score >= 0.8 {
		priority = PriorityCritical
	} else if score >= 0.6 {
		priority = PriorityHigh
	} else if score >= 0.4 {
		priority = PriorityMedium
	} else {
		priority = PriorityLow
	}

	return priority, score
}

// CalculateMessagePriorities 计算所有消息的优先级
func CalculateMessagePriorities(
	ctx context.Context,
	messages []Message,
	calculator PriorityCalculator,
) []MessageWithPriority {
	result := make([]MessageWithPriority, len(messages))

	for i, msg := range messages {
		priority, score := calculator.CalculatePriority(ctx, msg, i, len(messages))
		result[i] = MessageWithPriority{
			Message:  msg,
			Priority: priority,
			Score:    score,
		}
	}

	return result
}

// SortMessagesByPriority 按优先级排序消息
func SortMessagesByPriority(messages []MessageWithPriority, descending bool) {
	sort.Slice(messages, func(i, j int) bool {
		if descending {
			return messages[i].Score > messages[j].Score
		}
		return messages[i].Score < messages[j].Score
	})
}

// FilterMessagesByPriority 过滤低优先级消息
func FilterMessagesByPriority(
	messages []MessageWithPriority,
	minPriority MessagePriority,
) []MessageWithPriority {
	result := []MessageWithPriority{}
	for _, msg := range messages {
		if msg.Priority >= minPriority {
			result = append(result, msg)
		}
	}
	return result
}
