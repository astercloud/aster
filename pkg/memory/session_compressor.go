package memory

import (
	"context"
	"fmt"
	"strings"
	"time"

	agentext "github.com/astercloud/aster/pkg/context"
)

// SessionCompressor 会话压缩器接口
// 提供多层次的会话内容压缩和总结功能
type SessionCompressor interface {
	// SummarizeSession 总结整个会话
	SummarizeSession(ctx context.Context, messages []agentext.Message) (string, error)

	// CompressMessages 压缩消息列表
	CompressMessages(ctx context.Context, messages []agentext.Message) ([]agentext.Message, error)

	// GetCompressionStats 获取压缩统计信息
	GetCompressionStats() CompressionStats
}

// CompressionLevel 压缩级别
type CompressionLevel int

const (
	CompressionLevelNone       CompressionLevel = 0 // 不压缩
	CompressionLevelLight      CompressionLevel = 1 // 轻度压缩（保留大部分细节）
	CompressionLevelModerate   CompressionLevel = 2 // 中度压缩（保留关键信息）
	CompressionLevelAggressive CompressionLevel = 3 // 激进压缩（只保留核心要点）
)

// CompressionStats 压缩统计信息
type CompressionStats struct {
	OriginalMessages   int           // 原始消息数
	CompressedMessages int           // 压缩后消息数
	OriginalTokens     int           // 原始 Token 数
	CompressedTokens   int           // 压缩后 Token 数
	CompressionRatio   float64       // 压缩比率
	Duration           time.Duration // 压缩耗时
}

// LLMSummarizerConfig LLM 总结器配置
type LLMSummarizerConfig struct {
	// LLM 配置
	Provider string // LLM 提供商（如 "openai", "anthropic"）
	Model    string // 模型名称
	APIKey   string // API 密钥
	BaseURL  string // API 基础 URL

	// 总结配置
	Level           CompressionLevel // 压缩级别
	MaxSummaryWords int              // 最大总结字数
	PreserveContext bool             // 是否保留上下文信息
	Language        string           // 总结语言（"zh", "en"）

	// Token 配置
	TokenCounter agentext.TokenCounter // Token 计数器
}

// DefaultLLMSummarizerConfig 返回默认配置
func DefaultLLMSummarizerConfig() LLMSummarizerConfig {
	return LLMSummarizerConfig{
		Provider:        "openai",
		Model:           "gpt-4o-mini",
		Level:           CompressionLevelModerate,
		MaxSummaryWords: 500,
		PreserveContext: true,
		Language:        "zh",
		TokenCounter:    agentext.NewGPT4Counter(),
	}
}

// LLMSummarizer LLM 驱动的会话总结器
type LLMSummarizer struct {
	config LLMSummarizerConfig
	stats  CompressionStats
}

// NewLLMSummarizer 创建 LLM 总结器
func NewLLMSummarizer(config LLMSummarizerConfig) *LLMSummarizer {
	return &LLMSummarizer{
		config: config,
		stats:  CompressionStats{},
	}
}

// SummarizeSession 实现 SessionCompressor 接口
func (s *LLMSummarizer) SummarizeSession(ctx context.Context, messages []agentext.Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	startTime := time.Now()

	// 计算原始 Token 数
	originalTokens, err := s.config.TokenCounter.EstimateMessages(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("failed to count original tokens: %w", err)
	}

	// 构建总结提示词
	prompt := s.buildSummaryPrompt(messages)

	// 调用 LLM 生成总结
	summary, err := s.callLLM(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to call LLM: %w", err)
	}

	// 计算压缩后 Token 数
	compressedTokens, err := s.config.TokenCounter.Count(ctx, summary)
	if err != nil {
		return "", fmt.Errorf("failed to count compressed tokens: %w", err)
	}

	// 更新统计信息
	s.stats = CompressionStats{
		OriginalMessages:   len(messages),
		CompressedMessages: 1,
		OriginalTokens:     originalTokens,
		CompressedTokens:   compressedTokens,
		CompressionRatio:   float64(compressedTokens) / float64(originalTokens),
		Duration:           time.Since(startTime),
	}

	return summary, nil
}

// CompressMessages 实现 SessionCompressor 接口
func (s *LLMSummarizer) CompressMessages(ctx context.Context, messages []agentext.Message) ([]agentext.Message, error) {
	if len(messages) == 0 {
		return messages, nil
	}

	// 生成总结
	summary, err := s.SummarizeSession(ctx, messages)
	if err != nil {
		return nil, err
	}

	// 创建压缩后的消息
	compressed := []agentext.Message{
		{
			Role:    "system",
			Content: "以下是之前会话的总结：\n" + summary,
		},
	}

	return compressed, nil
}

// GetCompressionStats 实现 SessionCompressor 接口
func (s *LLMSummarizer) GetCompressionStats() CompressionStats {
	return s.stats
}

// buildSummaryPrompt 构建总结提示词
func (s *LLMSummarizer) buildSummaryPrompt(messages []agentext.Message) string {
	var sb strings.Builder

	// 添加总结指示
	levelInstructions := map[CompressionLevel]string{
		CompressionLevelLight:      "请详细总结以下会话内容，保留大部分细节和具体信息。",
		CompressionLevelModerate:   "请总结以下会话内容，保留关键信息和重要细节。",
		CompressionLevelAggressive: "请简要总结以下会话内容，只保留核心要点和结论。",
	}

	instruction := levelInstructions[s.config.Level]
	if instruction == "" {
		instruction = levelInstructions[CompressionLevelModerate]
	}

	sb.WriteString(instruction)
	sb.WriteString("\n\n")

	// 添加字数限制
	if s.config.MaxSummaryWords > 0 {
		sb.WriteString(fmt.Sprintf("请将总结控制在 %d 字以内。\n\n", s.config.MaxSummaryWords))
	}

	// 添加上下文保留要求
	if s.config.PreserveContext {
		sb.WriteString("总结时请保留以下信息：\n")
		sb.WriteString("- 用户的关键问题和需求\n")
		sb.WriteString("- 重要的决策和结论\n")
		sb.WriteString("- 关键的数据和事实\n")
		sb.WriteString("- 未解决的问题或待办事项\n\n")
	}

	// 添加会话内容
	sb.WriteString("会话内容：\n\n")
	for i, msg := range messages {
		sb.WriteString(fmt.Sprintf("[消息 %d - %s]\n%s\n\n", i+1, msg.Role, msg.Content))
	}

	return sb.String()
}

// callLLM 调用 LLM API 生成总结
// 注意：这是一个简化实现，实际应该使用真实的 LLM API
func (s *LLMSummarizer) callLLM(ctx context.Context, prompt string) (string, error) {
	// TODO: 集成真实的 LLM API
	// 这里返回一个模拟的总结作为占位符

	// 简单的启发式总结（用于测试）
	lines := strings.Split(prompt, "\n")
	summaryLines := []string{}

	for _, line := range lines {
		// 提取看起来像问题或结论的行
		if strings.Contains(line, "?") || strings.Contains(line, "：") {
			summaryLines = append(summaryLines, line)
		}
	}

	if len(summaryLines) == 0 {
		summaryLines = append(summaryLines, "会话总结：讨论了相关主题并达成初步共识。")
	}

	return strings.Join(summaryLines, "\n"), nil
}

// MultiLevelCompressor 多层次压缩器
// 支持消息级、对话轮次级、会话级的多层压缩
type MultiLevelCompressor struct {
	messageLevelCompressor SessionCompressor
	turnLevelCompressor    SessionCompressor
	sessionLevelCompressor SessionCompressor

	config MultiLevelCompressorConfig
}

// MultiLevelCompressorConfig 多层次压缩器配置
type MultiLevelCompressorConfig struct {
	// 消息级压缩配置
	EnableMessageLevel bool
	MessageThreshold   int // 超过多少条消息启用消息级压缩

	// 对话轮次级压缩配置
	EnableTurnLevel bool
	TurnThreshold   int // 超过多少轮对话启用轮次级压缩

	// 会话级压缩配置
	EnableSessionLevel bool
	SessionThreshold   int // 超过多少条消息启用会话级压缩

	// Token 预算
	TokenBudget agentext.TokenBudget
}

// DefaultMultiLevelCompressorConfig 返回默认配置
func DefaultMultiLevelCompressorConfig() MultiLevelCompressorConfig {
	return MultiLevelCompressorConfig{
		EnableMessageLevel: true,
		MessageThreshold:   100,
		EnableTurnLevel:    true,
		TurnThreshold:      20,
		EnableSessionLevel: true,
		SessionThreshold:   50,
		TokenBudget:        agentext.DefaultTokenBudget(),
	}
}

// NewMultiLevelCompressor 创建多层次压缩器
func NewMultiLevelCompressor(
	messageLevel SessionCompressor,
	turnLevel SessionCompressor,
	sessionLevel SessionCompressor,
	config MultiLevelCompressorConfig,
) *MultiLevelCompressor {
	return &MultiLevelCompressor{
		messageLevelCompressor: messageLevel,
		turnLevelCompressor:    turnLevel,
		sessionLevelCompressor: sessionLevel,
		config:                 config,
	}
}

// SummarizeSession 实现 SessionCompressor 接口
func (m *MultiLevelCompressor) SummarizeSession(ctx context.Context, messages []agentext.Message) (string, error) {
	// 根据消息数量选择合适的压缩级别
	messageCount := len(messages)

	if m.config.EnableSessionLevel && messageCount >= m.config.SessionThreshold {
		return m.sessionLevelCompressor.SummarizeSession(ctx, messages)
	}

	if m.config.EnableTurnLevel && messageCount >= m.config.TurnThreshold {
		return m.turnLevelCompressor.SummarizeSession(ctx, messages)
	}

	if m.config.EnableMessageLevel && messageCount >= m.config.MessageThreshold {
		return m.messageLevelCompressor.SummarizeSession(ctx, messages)
	}

	// 不需要压缩，直接连接所有消息
	var sb strings.Builder
	for _, msg := range messages {
		sb.WriteString(fmt.Sprintf("[%s] %s\n", msg.Role, msg.Content))
	}
	return sb.String(), nil
}

// CompressMessages 实现 SessionCompressor 接口
func (m *MultiLevelCompressor) CompressMessages(ctx context.Context, messages []agentext.Message) ([]agentext.Message, error) {
	if len(messages) == 0 {
		return messages, nil
	}

	// 1. 尝试对话轮次级压缩
	if m.config.EnableTurnLevel {
		compressed, err := m.compressByTurns(ctx, messages)
		if err == nil && len(compressed) < len(messages) {
			messages = compressed
		}
	}

	// 2. 检查是否需要会话级压缩
	if m.config.EnableSessionLevel && len(messages) >= m.config.SessionThreshold {
		return m.sessionLevelCompressor.CompressMessages(ctx, messages)
	}

	return messages, nil
}

// compressByTurns 按对话轮次压缩
func (m *MultiLevelCompressor) compressByTurns(ctx context.Context, messages []agentext.Message) ([]agentext.Message, error) {
	// 将消息分组为对话轮次（user + assistant）
	turns := m.groupIntoTurns(messages)

	// 如果轮次数少于阈值，不压缩
	if len(turns) < m.config.TurnThreshold {
		return messages, nil
	}

	compressed := []agentext.Message{}

	// 保留系统消息
	for _, msg := range messages {
		if msg.Role == "system" {
			compressed = append(compressed, msg)
		}
	}

	// 压缩每一轮对话
	for _, turn := range turns {
		if len(turn) == 0 {
			continue
		}

		// 如果这一轮很短，直接保留
		if len(turn) <= 2 {
			compressed = append(compressed, turn...)
			continue
		}

		// 对这一轮进行总结
		summary, err := m.turnLevelCompressor.SummarizeSession(ctx, turn)
		if err != nil {
			// 总结失败，保留原始消息
			compressed = append(compressed, turn...)
			continue
		}

		// 添加总结
		compressed = append(compressed, agentext.Message{
			Role:    "assistant",
			Content: "[对话轮次总结] " + summary,
		})
	}

	return compressed, nil
}

// groupIntoTurns 将消息分组为对话轮次
func (m *MultiLevelCompressor) groupIntoTurns(messages []agentext.Message) [][]agentext.Message {
	turns := [][]agentext.Message{}
	currentTurn := []agentext.Message{}

	for _, msg := range messages {
		// 跳过系统消息
		if msg.Role == "system" {
			continue
		}

		currentTurn = append(currentTurn, msg)

		// 当遇到 assistant 消息时，认为一轮对话结束
		if msg.Role == "assistant" {
			turns = append(turns, currentTurn)
			currentTurn = []agentext.Message{}
		}
	}

	// 添加最后一轮（如果有）
	if len(currentTurn) > 0 {
		turns = append(turns, currentTurn)
	}

	return turns
}

// GetCompressionStats 实现 SessionCompressor 接口
func (m *MultiLevelCompressor) GetCompressionStats() CompressionStats {
	// 合并所有层级的统计信息
	stats := CompressionStats{}

	if m.messageLevelCompressor != nil {
		msgStats := m.messageLevelCompressor.GetCompressionStats()
		stats.OriginalMessages += msgStats.OriginalMessages
		stats.CompressedMessages += msgStats.CompressedMessages
		stats.OriginalTokens += msgStats.OriginalTokens
		stats.CompressedTokens += msgStats.CompressedTokens
		stats.Duration += msgStats.Duration
	}

	if m.turnLevelCompressor != nil {
		turnStats := m.turnLevelCompressor.GetCompressionStats()
		stats.OriginalMessages += turnStats.OriginalMessages
		stats.CompressedMessages += turnStats.CompressedMessages
		stats.OriginalTokens += turnStats.OriginalTokens
		stats.CompressedTokens += turnStats.CompressedTokens
		stats.Duration += turnStats.Duration
	}

	if m.sessionLevelCompressor != nil {
		sessionStats := m.sessionLevelCompressor.GetCompressionStats()
		stats.OriginalMessages += sessionStats.OriginalMessages
		stats.CompressedMessages += sessionStats.CompressedMessages
		stats.OriginalTokens += sessionStats.OriginalTokens
		stats.CompressedTokens += sessionStats.CompressedTokens
		stats.Duration += sessionStats.Duration
	}

	// 计算总体压缩比率
	if stats.OriginalTokens > 0 {
		stats.CompressionRatio = float64(stats.CompressedTokens) / float64(stats.OriginalTokens)
	}

	return stats
}
