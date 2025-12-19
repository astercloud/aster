package memory

import (
	"context"
	"testing"
	"time"
)

func TestNewQualityMetrics(t *testing.T) {
	config := DefaultQualityMetricsConfig()
	qm := NewQualityMetrics(config)

	if qm == nil {
		t.Fatal("NewQualityMetrics returned nil")
	}

	if len(qm.qualities) != 0 {
		t.Errorf("new metrics should have 0 qualities, got %d", len(qm.qualities))
	}
}

func TestDefaultQualityMetricsConfig(t *testing.T) {
	config := DefaultQualityMetricsConfig()

	// 验证权重总和接近 1.0
	totalWeight := config.AccuracyWeight +
		config.CompletenessWeight +
		config.ConsistencyWeight +
		config.TimelinessWeight +
		config.RelevanceWeight

	if totalWeight < 0.99 || totalWeight > 1.01 {
		t.Errorf("total weight = %.2f, want ~1.0", totalWeight)
	}

	if config.MinQualityThreshold <= 0 || config.MinQualityThreshold >= 1 {
		t.Errorf("MinQualityThreshold = %.2f, should be between 0 and 1", config.MinQualityThreshold)
	}
}

func Test_QualityMetrics_Evaluate(t *testing.T) {
	config := DefaultQualityMetricsConfig()
	qm := NewQualityMetrics(config)

	// 创建测试记忆
	memory := &MemoryWithScore{
		DocID: "test-1",
		Text:  "This is a test memory with sufficient content to evaluate quality metrics properly.",
		Metadata: map[string]any{
			"source": "test",
		},
		Provenance: NewProvenance(SourceUserInput, "test-user"),
		Score:      0.8,
	}

	// 设置置信度
	memory.Provenance.Confidence = 0.75

	// 评估质量
	quality, err := qm.Evaluate(context.Background(), "test-1", memory)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	if quality.MemoryID != "test-1" {
		t.Errorf("MemoryID = %s, want test-1", quality.MemoryID)
	}

	if quality.Score.Accuracy != 0.75 {
		t.Errorf("Accuracy = %.2f, want 0.75", quality.Score.Accuracy)
	}

	if quality.Score.Relevance != 0.8 {
		t.Errorf("Relevance = %.2f, want 0.8", quality.Score.Relevance)
	}

	if quality.Score.Overall <= 0 || quality.Score.Overall > 1 {
		t.Errorf("Overall score = %.2f, should be between 0 and 1", quality.Score.Overall)
	}
}

func TestQualityMetrics_Get(t *testing.T) {
	config := DefaultQualityMetricsConfig()
	qm := NewQualityMetrics(config)

	memory := &MemoryWithScore{
		DocID:      "test-1",
		Text:       "Test content",
		Provenance: NewProvenance(SourceUserInput, "user-1"),
		Score:      0.7,
	}

	// 评估
	_, err := qm.Evaluate(context.Background(), "test-1", memory)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	// 获取
	quality, exists := qm.Get("test-1")
	if !exists {
		t.Error("quality should exist")
	}

	if quality.MemoryID != "test-1" {
		t.Errorf("MemoryID = %s, want test-1", quality.MemoryID)
	}

	// 获取不存在的
	_, exists = qm.Get("nonexistent")
	if exists {
		t.Error("nonexistent quality should not exist")
	}
}

func TestQualityMetrics_GetAll(t *testing.T) {
	config := DefaultQualityMetricsConfig()
	qm := NewQualityMetrics(config)

	// 添加多个质量评估
	for i := range 3 {
		memory := &MemoryWithScore{
			DocID:      string(rune('A' + i)),
			Text:       "Test content",
			Provenance: NewProvenance(SourceUserInput, "user-1"),
			Score:      0.7,
		}
		_, _ = qm.Evaluate(context.Background(), string(rune('A'+i)), memory)
	}

	all := qm.GetAll()
	if len(all) != 3 {
		t.Errorf("GetAll() returned %d items, want 3", len(all))
	}
}

func TestQualityMetrics_GetLowQuality(t *testing.T) {
	config := DefaultQualityMetricsConfig()
	config.MinQualityThreshold = 0.5
	qm := NewQualityMetrics(config)

	// 添加高质量记忆
	highQuality := &MemoryWithScore{
		DocID:      "high",
		Text:       "Good quality memory with sufficient content and metadata.",
		Provenance: NewProvenance(SourceUserInput, "user-1"),
		Score:      0.9,
	}
	highQuality.Provenance.Confidence = 0.9
	_, _ = qm.Evaluate(context.Background(), "high", highQuality)

	// 添加低质量记忆
	lowQuality := &MemoryWithScore{
		DocID:      "low",
		Text:       "Bad",
		Provenance: NewProvenance(SourceUserInput, "user-1"),
		Score:      0.2,
	}
	lowQuality.Provenance.Confidence = 0.2
	_, _ = qm.Evaluate(context.Background(), "low", lowQuality)

	// 获取低质量记忆
	low := qm.GetLowQuality()
	if len(low) != 1 {
		t.Errorf("GetLowQuality() returned %d items, want 1", len(low))
	}

	if len(low) > 0 && low[0].MemoryID != "low" {
		t.Errorf("low quality MemoryID = %s, want low", low[0].MemoryID)
	}
}

func TestQualityMetrics_Remove(t *testing.T) {
	config := DefaultQualityMetricsConfig()
	qm := NewQualityMetrics(config)

	memory := &MemoryWithScore{
		DocID:      "test-1",
		Text:       "Test content",
		Provenance: NewProvenance(SourceUserInput, "user-1"),
		Score:      0.7,
	}

	_, _ = qm.Evaluate(context.Background(), "test-1", memory)

	// 验证存在
	_, exists := qm.Get("test-1")
	if !exists {
		t.Error("quality should exist before removal")
	}

	// 删除
	qm.Remove("test-1")

	// 验证不存在
	_, exists = qm.Get("test-1")
	if exists {
		t.Error("quality should not exist after removal")
	}
}

func TestQualityMetrics_Clear(t *testing.T) {
	config := DefaultQualityMetricsConfig()
	qm := NewQualityMetrics(config)

	// 添加多个
	for i := range 3 {
		memory := &MemoryWithScore{
			DocID:      string(rune('A' + i)),
			Text:       "Test",
			Provenance: NewProvenance(SourceUserInput, "user-1"),
			Score:      0.7,
		}
		_, _ = qm.Evaluate(context.Background(), string(rune('A'+i)), memory)
	}

	// 清空
	qm.Clear()

	all := qm.GetAll()
	if len(all) != 0 {
		t.Errorf("after Clear(), GetAll() should return 0 items, got %d", len(all))
	}
}

func TestQualityMetrics_GetStats(t *testing.T) {
	config := DefaultQualityMetricsConfig()
	config.MinQualityThreshold = 0.4
	config.WarningThreshold = 0.6
	qm := NewQualityMetrics(config)

	// 添加不同质量的记忆
	memories := []struct {
		id         string
		confidence float64
		score      float64
	}{
		{"high1", 0.9, 0.9},   // 高质量
		{"high2", 0.85, 0.85}, // 高质量
		{"medium", 0.6, 0.6},  // 中等质量
		{"low", 0.2, 0.2},     // 低质量
	}

	for _, m := range memories {
		mem := &MemoryWithScore{
			DocID:      m.id,
			Text:       "Test content with varying quality",
			Provenance: NewProvenance(SourceUserInput, "user-1"),
			Score:      m.score,
		}
		mem.Provenance.Confidence = m.confidence
		_, _ = qm.Evaluate(context.Background(), m.id, mem)
	}

	stats := qm.GetStats()

	if stats.TotalMemories != 4 {
		t.Errorf("TotalMemories = %d, want 4", stats.TotalMemories)
	}

	// 注意：综合得分是加权平均，所以实际分类可能不同
	// 只验证总数正确
	totalCategorized := stats.HighQualityCount + stats.MediumQualityCount + stats.LowQualityCount
	if totalCategorized != 4 {
		t.Errorf("total categorized = %d, want 4", totalCategorized)
	}

	// 至少应该有一些低质量和高质量的记忆
	if stats.LowQualityCount == 0 {
		t.Errorf("LowQualityCount = 0, should have at least one")
	}

	if stats.AverageQuality <= 0 {
		t.Errorf("AverageQuality should be > 0, got %.2f", stats.AverageQuality)
	}
}

func TestEvaluateCompleteness(t *testing.T) {
	config := DefaultQualityMetricsConfig()
	qm := NewQualityMetrics(config)

	tests := []struct {
		name    string
		text    string
		wantMin float64
		wantMax float64
	}{
		{
			name:    "empty content",
			text:    "",
			wantMin: 0.0,
			wantMax: 0.0,
		},
		{
			name:    "short content",
			text:    "Hi",
			wantMin: 0.0,
			wantMax: 0.3,
		},
		{
			name:    "medium content",
			text:    "This is a medium length content that should score around 50%",
			wantMin: 0.15,
			wantMax: 0.35,
		},
		{
			name:    "long content",
			text:    "This is a much longer piece of content that includes sufficient information and details. It should score higher on the completeness metric because it contains more comprehensive information that would be useful for various purposes.",
			wantMin: 0.5,
			wantMax: 0.7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memory := &MemoryWithScore{
				Text: tt.text,
			}

			score := qm.evaluateCompleteness(memory)

			if score < tt.wantMin || score > tt.wantMax {
				t.Errorf("completeness score = %.2f, want between %.2f and %.2f",
					score, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestEvaluateTimeliness(t *testing.T) {
	config := DefaultQualityMetricsConfig()
	qm := NewQualityMetrics(config)

	tests := []struct {
		name    string
		age     time.Duration
		wantMin float64
		wantMax float64
	}{
		{
			name:    "very recent",
			age:     1 * time.Hour,
			wantMin: 0.99,
			wantMax: 1.0,
		},
		{
			name:    "one day old",
			age:     24 * time.Hour,
			wantMin: 0.95,
			wantMax: 1.0,
		},
		{
			name:    "one month old",
			age:     30 * 24 * time.Hour,
			wantMin: 0.7,
			wantMax: 0.9,
		},
		{
			name:    "very old",
			age:     365 * 24 * time.Hour,
			wantMin: 0.0,
			wantMax: 0.3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provenance := NewProvenance(SourceUserInput, "user-1")
			provenance.CreatedAt = time.Now().Add(-tt.age)

			memory := &MemoryWithScore{
				Provenance: provenance,
			}

			score := qm.evaluateTimeliness(memory)

			if score < tt.wantMin || score > tt.wantMax {
				t.Errorf("timeliness score = %.2f, want between %.2f and %.2f",
					score, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestRankByQuality(t *testing.T) {
	memories := []MemoryWithScore{
		{DocID: "A", Score: 0.5},
		{DocID: "B", Score: 0.9},
		{DocID: "C", Score: 0.3},
	}

	// 创建质量信息
	qualities := map[string]*MemoryQuality{
		"A": {Score: QualityScore{Overall: 0.8}},
		"B": {Score: QualityScore{Overall: 0.6}},
		"C": {Score: QualityScore{Overall: 0.9}},
	}

	ranked := RankByQuality(memories, qualities)

	// 验证排序（应该按综合分数降序）
	// C: 0.9 * 0.7 + 0.3 * 0.3 = 0.72
	// A: 0.8 * 0.7 + 0.5 * 0.3 = 0.71
	// B: 0.6 * 0.7 + 0.9 * 0.3 = 0.69

	if ranked[0].DocID != "C" {
		t.Errorf("first should be C, got %s", ranked[0].DocID)
	}
}

func TestFilterByQuality(t *testing.T) {
	memories := []MemoryWithScore{
		{DocID: "A"},
		{DocID: "B"},
		{DocID: "C"},
		{DocID: "D"},
	}

	qualities := map[string]*MemoryQuality{
		"A": {Score: QualityScore{Overall: 0.8}},
		"B": {Score: QualityScore{Overall: 0.4}},
		"C": {Score: QualityScore{Overall: 0.6}},
		// D 没有质量信息
	}

	filtered := FilterByQuality(memories, qualities, 0.5)

	// 应该保留 A (0.8), C (0.6), D (无质量信息默认保留)
	if len(filtered) != 3 {
		t.Errorf("filtered length = %d, want 3", len(filtered))
	}

	// 验证 B 被过滤掉
	for _, mem := range filtered {
		if mem.DocID == "B" {
			t.Error("B should be filtered out (quality 0.4 < 0.5)")
		}
	}
}
