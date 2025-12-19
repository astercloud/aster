package memory

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

// QualityAnalyzer 质量分析器
// 提供高级质量分析功能，包括不一致性检测、趋势分析等
type QualityAnalyzer struct {
	metrics *QualityMetrics
	memory  *SemanticMemory
}

// NewQualityAnalyzer 创建质量分析器
func NewQualityAnalyzer(metrics *QualityMetrics, memory *SemanticMemory) *QualityAnalyzer {
	return &QualityAnalyzer{
		metrics: metrics,
		memory:  memory,
	}
}

// Inconsistency 不一致性信息
type Inconsistency struct {
	Type        InconsistencyType // 不一致性类型
	MemoryID1   string            // 第一个记忆 ID
	MemoryID2   string            // 第二个记忆 ID（如果适用）
	Description string            // 描述
	Severity    float64           // 严重程度 (0.0-1.0)
	DetectedAt  time.Time         // 检测时间
}

// InconsistencyType 不一致性类型
type InconsistencyType string

const (
	InconsistencyContradiction InconsistencyType = "contradiction"  // 矛盾
	InconsistencyDuplicate     InconsistencyType = "duplicate"      // 重复
	InconsistencyOutdated      InconsistencyType = "outdated"       // 过时
	InconsistencyLowConfidence InconsistencyType = "low_confidence" // 低置信度
	InconsistencyConflict      InconsistencyType = "conflict"       // 冲突
)

// DetectInconsistencies 检测记忆中的不一致性
func (qa *QualityAnalyzer) DetectInconsistencies(
	ctx context.Context,
	memories []MemoryWithScore,
) ([]Inconsistency, error) {
	inconsistencies := []Inconsistency{}

	// 1. 检测矛盾
	contradictions := qa.detectContradictions(memories)
	inconsistencies = append(inconsistencies, contradictions...)

	// 2. 检测重复
	duplicates := qa.detectDuplicates(memories)
	inconsistencies = append(inconsistencies, duplicates...)

	// 3. 检测过时信息
	outdated := qa.detectOutdated(memories)
	inconsistencies = append(inconsistencies, outdated...)

	// 4. 检测低置信度
	lowConfidence := qa.detectLowConfidence(memories)
	inconsistencies = append(inconsistencies, lowConfidence...)

	return inconsistencies, nil
}

// detectContradictions 检测矛盾信息
func (qa *QualityAnalyzer) detectContradictions(memories []MemoryWithScore) []Inconsistency {
	inconsistencies := []Inconsistency{}

	// 简化版：检查内容中的否定关系
	for i := range memories {
		for j := i + 1; j < len(memories); j++ {
			if qa.areContradictory(memories[i], memories[j]) {
				severity := qa.calculateContradictionSeverity(memories[i], memories[j])
				inconsistencies = append(inconsistencies, Inconsistency{
					Type:      InconsistencyContradiction,
					MemoryID1: memories[i].DocID,
					MemoryID2: memories[j].DocID,
					Description: fmt.Sprintf("记忆存在矛盾：'%s' vs '%s'",
						truncate(memories[i].Text, 50),
						truncate(memories[j].Text, 50)),
					Severity:   severity,
					DetectedAt: time.Now(),
				})
			}
		}
	}

	return inconsistencies
}

// areContradictory 检查两个记忆是否矛盾
func (qa *QualityAnalyzer) areContradictory(mem1, mem2 MemoryWithScore) bool {
	content1 := strings.ToLower(mem1.Text)
	content2 := strings.ToLower(mem2.Text)

	// 检测否定模式
	negationPatterns := []struct {
		positive string
		negative string
	}{
		{"is", "is not"},
		{"是", "不是"},
		{"has", "does not have"},
		{"有", "没有"},
		{"can", "cannot"},
		{"能", "不能"},
		{"will", "will not"},
		{"会", "不会"},
	}

	for _, pattern := range negationPatterns {
		if (strings.Contains(content1, pattern.positive) && strings.Contains(content2, pattern.negative)) ||
			(strings.Contains(content1, pattern.negative) && strings.Contains(content2, pattern.positive)) {
			return true
		}
	}

	return false
}

// calculateContradictionSeverity 计算矛盾严重程度
func (qa *QualityAnalyzer) calculateContradictionSeverity(mem1, mem2 MemoryWithScore) float64 {
	// 基于两个记忆的置信度和相似度
	severity := 0.5

	// 如果两个记忆都有高置信度，矛盾更严重
	if mem1.Provenance != nil && mem2.Provenance != nil {
		avgConfidence := (mem1.Provenance.Confidence + mem2.Provenance.Confidence) / 2
		severity += avgConfidence * 0.3
	}

	// 如果内容相似度高但存在矛盾，更严重
	similarity := qa.calculateSimilarity(mem1.Text, mem2.Text)
	severity += similarity * 0.2

	return math.Min(severity, 1.0)
}

// detectDuplicates 检测重复记忆
func (qa *QualityAnalyzer) detectDuplicates(memories []MemoryWithScore) []Inconsistency {
	inconsistencies := []Inconsistency{}

	for i := range memories {
		for j := i + 1; j < len(memories); j++ {
			similarity := qa.calculateSimilarity(memories[i].Text, memories[j].Text)

			// 相似度 > 0.9 认为是重复
			if similarity > 0.9 {
				inconsistencies = append(inconsistencies, Inconsistency{
					Type:      InconsistencyDuplicate,
					MemoryID1: memories[i].DocID,
					MemoryID2: memories[j].DocID,
					Description: fmt.Sprintf("记忆高度重复（相似度: %.2f）：'%s'",
						similarity, truncate(memories[i].Text, 50)),
					Severity:   similarity,
					DetectedAt: time.Now(),
				})
			}
		}
	}

	return inconsistencies
}

// detectOutdated 检测过时信息
func (qa *QualityAnalyzer) detectOutdated(memories []MemoryWithScore) []Inconsistency {
	inconsistencies := []Inconsistency{}

	for _, mem := range memories {
		if mem.Provenance == nil {
			continue
		}

		age := time.Since(mem.Provenance.CreatedAt)

		// 超过 180 天认为可能过时
		if age > 180*24*time.Hour {
			severity := math.Min(age.Hours()/(365*24), 1.0) // 1年=100%严重

			inconsistencies = append(inconsistencies, Inconsistency{
				Type:      InconsistencyOutdated,
				MemoryID1: mem.DocID,
				Description: fmt.Sprintf("记忆可能过时（%d天前）：'%s'",
					int(age.Hours()/24), truncate(mem.Text, 50)),
				Severity:   severity,
				DetectedAt: time.Now(),
			})
		}
	}

	return inconsistencies
}

// detectLowConfidence 检测低置信度记忆
func (qa *QualityAnalyzer) detectLowConfidence(memories []MemoryWithScore) []Inconsistency {
	inconsistencies := []Inconsistency{}

	for _, mem := range memories {
		if mem.Provenance == nil {
			continue
		}

		if mem.Provenance.Confidence < 0.4 {
			inconsistencies = append(inconsistencies, Inconsistency{
				Type:      InconsistencyLowConfidence,
				MemoryID1: mem.DocID,
				Description: fmt.Sprintf("记忆置信度过低（%.2f）：'%s'",
					mem.Provenance.Confidence, truncate(mem.Text, 50)),
				Severity:   1.0 - mem.Provenance.Confidence,
				DetectedAt: time.Now(),
			})
		}
	}

	return inconsistencies
}

// calculateSimilarity 计算两个文本的相似度
// 简化版：使用 Jaccard 相似度
func (qa *QualityAnalyzer) calculateSimilarity(text1, text2 string) float64 {
	// 分词（简单按空格分）
	words1 := strings.Fields(strings.ToLower(text1))
	words2 := strings.Fields(strings.ToLower(text2))

	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	// 计算交集和并集
	set1 := make(map[string]bool)
	set2 := make(map[string]bool)

	for _, w := range words1 {
		set1[w] = true
	}
	for _, w := range words2 {
		set2[w] = true
	}

	intersection := 0
	for w := range set1 {
		if set2[w] {
			intersection++
		}
	}

	union := len(set1) + len(set2) - intersection

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// QualityTrend 质量趋势
type QualityTrend struct {
	Dimension QualityDimension
	Trend     string  // "improving", "declining", "stable"
	Change    float64 // 变化量
	Period    time.Duration
}

// AnalyzeTrends 分析质量趋势（需要历史数据）
func (qa *QualityAnalyzer) AnalyzeTrends(
	ctx context.Context,
	period time.Duration,
) ([]QualityTrend, error) {
	// TODO: 需要存储历史质量数据才能分析趋势
	// 当前返回空列表
	return []QualityTrend{}, nil
}

// QualityReport 质量报告
type QualityReport struct {
	GeneratedAt time.Time
	Stats       QualityStats

	// 维度分布
	DimensionScores map[QualityDimension]float64

	// 问题统计
	Inconsistencies    []Inconsistency
	InconsistencyCount map[InconsistencyType]int
	TopIssues          []string

	// 建议
	Recommendations []string
}

// GenerateReport 生成质量报告
func (qa *QualityAnalyzer) GenerateReport(
	ctx context.Context,
	memories []MemoryWithScore,
) (*QualityReport, error) {
	report := &QualityReport{
		GeneratedAt:        time.Now(),
		DimensionScores:    make(map[QualityDimension]float64),
		InconsistencyCount: make(map[InconsistencyType]int),
	}

	// 获取统计信息
	report.Stats = qa.metrics.GetStats()

	// 计算各维度平均分数
	dimensionTotals := make(map[QualityDimension]float64)
	dimensionCounts := make(map[QualityDimension]int)

	for _, quality := range qa.metrics.GetAll() {
		dimensionTotals[QualityAccuracy] += quality.Score.Accuracy
		dimensionTotals[QualityCompleteness] += quality.Score.Completeness
		dimensionTotals[QualityConsistency] += quality.Score.Consistency
		dimensionTotals[QualityTimeliness] += quality.Score.Timeliness
		dimensionTotals[QualityRelevance] += quality.Score.Relevance

		for dim := range dimensionTotals {
			dimensionCounts[dim]++
		}
	}

	for dim, total := range dimensionTotals {
		if dimensionCounts[dim] > 0 {
			report.DimensionScores[dim] = total / float64(dimensionCounts[dim])
		}
	}

	// 检测不一致性
	inconsistencies, err := qa.DetectInconsistencies(ctx, memories)
	if err != nil {
		return nil, err
	}
	report.Inconsistencies = inconsistencies

	// 统计不一致性类型
	for _, inc := range inconsistencies {
		report.InconsistencyCount[inc.Type]++
	}

	// 收集 TOP 问题
	issueMap := make(map[string]int)
	for _, quality := range qa.metrics.GetAll() {
		for _, issue := range quality.Issues {
			issueMap[issue]++
		}
	}

	topIssues := []struct {
		issue string
		count int
	}{}
	for issue, count := range issueMap {
		topIssues = append(topIssues, struct {
			issue string
			count int
		}{issue, count})
	}

	// 排序 top 5
	for i := 0; i < len(topIssues) && i < 5; i++ {
		for j := i + 1; j < len(topIssues); j++ {
			if topIssues[i].count < topIssues[j].count {
				topIssues[i], topIssues[j] = topIssues[j], topIssues[i]
			}
		}
	}

	for i := 0; i < len(topIssues) && i < 5; i++ {
		report.TopIssues = append(report.TopIssues,
			fmt.Sprintf("%s (%d次)", topIssues[i].issue, topIssues[i].count))
	}

	// 生成建议
	report.Recommendations = qa.generateRecommendations(report)

	return report, nil
}

// generateRecommendations 生成改进建议
func (qa *QualityAnalyzer) generateRecommendations(report *QualityReport) []string {
	recommendations := []string{}

	// 基于平均质量
	if report.Stats.AverageQuality < 0.5 {
		recommendations = append(recommendations,
			"整体质量偏低，建议增加数据来源的可靠性验证")
	}

	// 基于维度得分
	if score, ok := report.DimensionScores[QualityAccuracy]; ok && score < 0.6 {
		recommendations = append(recommendations,
			"准确性较低，建议增加信息来源的佐证")
	}

	if score, ok := report.DimensionScores[QualityCompleteness]; ok && score < 0.6 {
		recommendations = append(recommendations,
			"完整性不足，建议记录更详细的信息和元数据")
	}

	if score, ok := report.DimensionScores[QualityConsistency]; ok && score < 0.6 {
		recommendations = append(recommendations,
			"一致性问题，建议运行记忆合并以解决冲突")
	}

	if score, ok := report.DimensionScores[QualityTimeliness]; ok && score < 0.6 {
		recommendations = append(recommendations,
			"时效性下降，建议定期更新或归档旧记忆")
	}

	// 基于不一致性
	if report.InconsistencyCount[InconsistencyContradiction] > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("发现 %d 处矛盾，建议手动审查并解决",
				report.InconsistencyCount[InconsistencyContradiction]))
	}

	if report.InconsistencyCount[InconsistencyDuplicate] > 3 {
		recommendations = append(recommendations,
			fmt.Sprintf("发现 %d 处重复，建议运行去重操作",
				report.InconsistencyCount[InconsistencyDuplicate]))
	}

	// 基于低质量记忆数量
	if report.Stats.LowQualityCount > report.Stats.TotalMemories/4 {
		recommendations = append(recommendations,
			"超过 25% 的记忆质量较低，建议清理或改进")
	}

	return recommendations
}

// ImproveQuality 质量改进建议
type ImproveQuality struct {
	MemoryID     string
	CurrentScore float64
	Suggestions  []string
}

// SuggestImprovements 为低质量记忆提供改进建议
func (qa *QualityAnalyzer) SuggestImprovements(
	ctx context.Context,
) ([]ImproveQuality, error) {
	improvements := []ImproveQuality{}

	lowQuality := qa.metrics.GetLowQuality()
	for _, quality := range lowQuality {
		suggestions := []string{}

		if quality.Score.Accuracy < 0.5 {
			suggestions = append(suggestions, "添加可靠来源的佐证")
		}

		if quality.Score.Completeness < 0.5 {
			suggestions = append(suggestions, "补充更详细的信息和元数据")
		}

		if quality.Score.Consistency < 0.5 {
			suggestions = append(suggestions, "寻找支持性证据增加一致性")
		}

		if quality.Score.Timeliness < 0.5 {
			suggestions = append(suggestions, "验证信息是否仍然有效，考虑更新或归档")
		}

		improvements = append(improvements, ImproveQuality{
			MemoryID:     quality.MemoryID,
			CurrentScore: quality.Score.Overall,
			Suggestions:  suggestions,
		})
	}

	return improvements, nil
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
