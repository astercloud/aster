package memory

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// QualityDimension 质量维度
type QualityDimension string

const (
	QualityAccuracy     QualityDimension = "accuracy"     // 准确性
	QualityCompleteness QualityDimension = "completeness" // 完整性
	QualityConsistency  QualityDimension = "consistency"  // 一致性
	QualityTimeliness   QualityDimension = "timeliness"   // 时效性
	QualityRelevance    QualityDimension = "relevance"    // 相关性
)

// QualityScore 质量分数（多维度）
type QualityScore struct {
	Accuracy     float64 // 准确性 (0.0-1.0)
	Completeness float64 // 完整性 (0.0-1.0)
	Consistency  float64 // 一致性 (0.0-1.0)
	Timeliness   float64 // 时效性 (0.0-1.0)
	Relevance    float64 // 相关性 (0.0-1.0)

	Overall float64 // 综合得分 (0.0-1.0)

	// 元数据
	CalculatedAt time.Time
	Source       string // 评分来源（如 "automatic", "manual", "llm"）
}

// MemoryQuality 记忆质量信息
type MemoryQuality struct {
	MemoryID  string       // 记忆 ID
	Score     QualityScore // 质量分数
	Issues    []string     // 检测到的问题
	UpdatedAt time.Time    // 更新时间
}

// QualityMetrics 质量评估系统
type QualityMetrics struct {
	mu sync.RWMutex

	// 配置
	config QualityMetricsConfig

	// 质量存储
	qualities map[string]*MemoryQuality // memoryID -> quality

	// 统计
	totalEvaluations int64
	avgQuality       float64
}

// QualityMetricsConfig 质量评估配置
type QualityMetricsConfig struct {
	// 权重配置（各维度的权重，总和应为 1.0）
	AccuracyWeight     float64
	CompletenessWeight float64
	ConsistencyWeight  float64
	TimelinessWeight   float64
	RelevanceWeight    float64

	// 时效性配置
	MaxAge          time.Duration // 最大有效期
	TimelinessDecay float64       // 时效性衰减系数（每天）

	// 质量阈值
	MinQualityThreshold float64 // 最低质量阈值
	WarningThreshold    float64 // 警告阈值

	// 自动清理
	EnableAutoCleanup   bool // 启用自动清理低质量记忆
	AutoCleanupInterval time.Duration
}

// DefaultQualityMetricsConfig 返回默认配置
func DefaultQualityMetricsConfig() QualityMetricsConfig {
	return QualityMetricsConfig{
		// 均等权重
		AccuracyWeight:     0.3,
		CompletenessWeight: 0.2,
		ConsistencyWeight:  0.2,
		TimelinessWeight:   0.15,
		RelevanceWeight:    0.15,

		// 时效性配置
		MaxAge:          90 * 24 * time.Hour, // 90 天
		TimelinessDecay: 0.01,                // 每天衰减 1%

		// 质量阈值
		MinQualityThreshold: 0.3, // 低于 30% 认为是低质量
		WarningThreshold:    0.5, // 低于 50% 发出警告

		// 自动清理
		EnableAutoCleanup:   false,
		AutoCleanupInterval: 24 * time.Hour,
	}
}

// NewQualityMetrics 创建质量评估系统
func NewQualityMetrics(config QualityMetricsConfig) *QualityMetrics {
	qm := &QualityMetrics{
		config:    config,
		qualities: make(map[string]*MemoryQuality),
	}

	// 启动自动清理
	if config.EnableAutoCleanup {
		qm.startAutoCleanup()
	}

	return qm
}

// Evaluate 评估记忆质量
func (qm *QualityMetrics) Evaluate(
	ctx context.Context,
	memoryID string,
	memory *MemoryWithScore,
) (*MemoryQuality, error) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	// 计算各维度分数
	score := QualityScore{
		Source:       "automatic",
		CalculatedAt: time.Now(),
	}

	// 1. 准确性（基于 Provenance 的置信度）
	if memory.Provenance != nil {
		score.Accuracy = memory.Provenance.Confidence
	} else {
		score.Accuracy = 0.5 // 默认中等准确性
	}

	// 2. 完整性（基于内容长度和结构）
	score.Completeness = qm.evaluateCompleteness(memory)

	// 3. 一致性（基于 Provenance 的佐证数量）
	score.Consistency = qm.evaluateConsistency(memory)

	// 4. 时效性（基于创建时间）
	score.Timeliness = qm.evaluateTimeliness(memory)

	// 5. 相关性（基于检索分数）
	score.Relevance = memory.Score

	// 计算综合得分
	score.Overall = qm.calculateOverallScore(score)

	// 检测问题
	issues := qm.detectIssues(score, memory)

	// 创建质量信息
	quality := &MemoryQuality{
		MemoryID:  memoryID,
		Score:     score,
		Issues:    issues,
		UpdatedAt: time.Now(),
	}

	// 存储
	qm.qualities[memoryID] = quality

	// 更新统计
	qm.totalEvaluations++
	qm.updateAvgQuality()

	return quality, nil
}

// evaluateCompleteness 评估完整性
func (qm *QualityMetrics) evaluateCompleteness(memory *MemoryWithScore) float64 {
	if memory == nil || memory.Text == "" {
		return 0.0
	}

	// 基于内容长度
	length := float64(len(memory.Text))

	// 使用 sigmoid 函数，200 字符达到 50%
	score := 1.0 / (1.0 + math.Exp(-0.01*(length-200)))

	// 检查是否有元数据
	if len(memory.Metadata) > 0 {
		score = math.Min(score+0.1, 1.0) // 有元数据加 10%
	}

	return score
}

// evaluateConsistency 评估一致性
func (qm *QualityMetrics) evaluateConsistency(memory *MemoryWithScore) float64 {
	if memory.Provenance == nil {
		return 0.5 // 无溯源信息，中等一致性
	}

	// 基于佐证数量
	corroborationCount := float64(memory.Provenance.CorroborationCount)

	// 使用对数函数，5 个佐证达到约 80%
	if corroborationCount == 0 {
		return 0.3 // 无佐证
	}

	score := 0.3 + 0.7*(math.Log10(corroborationCount+1)/math.Log10(6))
	return math.Min(score, 1.0)
}

// evaluateTimeliness 评估时效性
func (qm *QualityMetrics) evaluateTimeliness(memory *MemoryWithScore) float64 {
	if memory.Provenance == nil {
		return 0.5
	}

	age := time.Since(memory.Provenance.CreatedAt)

	// 超过最大年龄，时效性为 0
	if age > qm.config.MaxAge {
		return 0.0
	}

	// 计算时效性衰减
	days := age.Hours() / 24.0
	decay := math.Exp(-qm.config.TimelinessDecay * days)

	return decay
}

// calculateOverallScore 计算综合得分
func (qm *QualityMetrics) calculateOverallScore(score QualityScore) float64 {
	overall := score.Accuracy*qm.config.AccuracyWeight +
		score.Completeness*qm.config.CompletenessWeight +
		score.Consistency*qm.config.ConsistencyWeight +
		score.Timeliness*qm.config.TimelinessWeight +
		score.Relevance*qm.config.RelevanceWeight

	return math.Min(overall, 1.0)
}

// detectIssues 检测质量问题
func (qm *QualityMetrics) detectIssues(score QualityScore, memory *MemoryWithScore) []string {
	issues := []string{}

	if score.Accuracy < 0.5 {
		issues = append(issues, "低准确性：置信度不足")
	}

	if score.Completeness < 0.3 {
		issues = append(issues, "低完整性：内容过于简短或缺少元数据")
	}

	if score.Consistency < 0.4 {
		issues = append(issues, "低一致性：缺少佐证支持")
	}

	if score.Timeliness < 0.3 {
		issues = append(issues, "低时效性：记忆过于陈旧")
	}

	if score.Relevance < 0.5 {
		issues = append(issues, "低相关性：与查询相关度不高")
	}

	if score.Overall < qm.config.MinQualityThreshold {
		issues = append(issues, fmt.Sprintf("整体质量过低：%.2f < %.2f",
			score.Overall, qm.config.MinQualityThreshold))
	}

	return issues
}

// Get 获取记忆的质量信息
func (qm *QualityMetrics) Get(memoryID string) (*MemoryQuality, bool) {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	quality, exists := qm.qualities[memoryID]
	return quality, exists
}

// GetAll 获取所有质量信息
func (qm *QualityMetrics) GetAll() []*MemoryQuality {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	result := make([]*MemoryQuality, 0, len(qm.qualities))
	for _, quality := range qm.qualities {
		result = append(result, quality)
	}
	return result
}

// GetLowQuality 获取低质量记忆列表
func (qm *QualityMetrics) GetLowQuality() []*MemoryQuality {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	result := []*MemoryQuality{}
	for _, quality := range qm.qualities {
		if quality.Score.Overall < qm.config.MinQualityThreshold {
			result = append(result, quality)
		}
	}
	return result
}

// Remove 删除质量信息
func (qm *QualityMetrics) Remove(memoryID string) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	delete(qm.qualities, memoryID)
	qm.updateAvgQuality()
}

// Clear 清空所有质量信息
func (qm *QualityMetrics) Clear() {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	qm.qualities = make(map[string]*MemoryQuality)
	qm.avgQuality = 0
}

// GetStats 获取统计信息
func (qm *QualityMetrics) GetStats() QualityStats {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	stats := QualityStats{
		TotalMemories:    len(qm.qualities),
		TotalEvaluations: qm.totalEvaluations,
		AverageQuality:   qm.avgQuality,
	}

	// 统计各质量等级
	for _, quality := range qm.qualities {
		if quality.Score.Overall >= 0.8 {
			stats.HighQualityCount++
		} else if quality.Score.Overall >= qm.config.WarningThreshold {
			stats.MediumQualityCount++
		} else {
			stats.LowQualityCount++
		}

		if len(quality.Issues) > 0 {
			stats.MemoriesWithIssues++
		}
	}

	return stats
}

// QualityStats 质量统计信息
type QualityStats struct {
	TotalMemories      int     // 总记忆数
	TotalEvaluations   int64   // 总评估次数
	AverageQuality     float64 // 平均质量
	HighQualityCount   int     // 高质量记忆数 (>= 0.8)
	MediumQualityCount int     // 中等质量记忆数 (>= 0.5)
	LowQualityCount    int     // 低质量记忆数 (< 0.5)
	MemoriesWithIssues int     // 有问题的记忆数
}

// updateAvgQuality 更新平均质量（需要持有锁）
func (qm *QualityMetrics) updateAvgQuality() {
	if len(qm.qualities) == 0 {
		qm.avgQuality = 0
		return
	}

	total := 0.0
	for _, quality := range qm.qualities {
		total += quality.Score.Overall
	}
	qm.avgQuality = total / float64(len(qm.qualities))
}

// startAutoCleanup 启动自动清理低质量记忆
func (qm *QualityMetrics) startAutoCleanup() {
	go func() {
		ticker := time.NewTicker(qm.config.AutoCleanupInterval)
		defer ticker.Stop()

		for range ticker.C {
			qm.cleanupLowQuality()
		}
	}()
}

// cleanupLowQuality 清理低质量记忆
func (qm *QualityMetrics) cleanupLowQuality() {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	removed := []string{}
	for memoryID, quality := range qm.qualities {
		if quality.Score.Overall < qm.config.MinQualityThreshold {
			delete(qm.qualities, memoryID)
			removed = append(removed, memoryID)
		}
	}

	if len(removed) > 0 {
		qm.updateAvgQuality()
	}
}

// RankByQuality 按质量排序记忆列表
func RankByQuality(
	memories []MemoryWithScore,
	qualities map[string]*MemoryQuality,
) []MemoryWithScore {
	// 创建副本以避免修改原始列表
	ranked := make([]MemoryWithScore, len(memories))
	copy(ranked, memories)

	// 按质量综合得分排序
	for i := range ranked {
		for j := i + 1; j < len(ranked); j++ {
			// 获取质量分数
			scoreI := ranked[i].Score
			scoreJ := ranked[j].Score

			if qI, ok := qualities[ranked[i].DocID]; ok {
				scoreI = qI.Score.Overall*0.7 + ranked[i].Score*0.3
			}
			if qJ, ok := qualities[ranked[j].DocID]; ok {
				scoreJ = qJ.Score.Overall*0.7 + ranked[j].Score*0.3
			}

			// 降序排序
			if scoreI < scoreJ {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}

	return ranked
}

// FilterByQuality 根据质量阈值过滤记忆
func FilterByQuality(
	memories []MemoryWithScore,
	qualities map[string]*MemoryQuality,
	minQuality float64,
) []MemoryWithScore {
	filtered := []MemoryWithScore{}

	for _, mem := range memories {
		if quality, ok := qualities[mem.DocID]; ok {
			if quality.Score.Overall >= minQuality {
				filtered = append(filtered, mem)
			}
		} else {
			// 如果没有质量信息，默认保留
			filtered = append(filtered, mem)
		}
	}

	return filtered
}
