package memory

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNewQualityAnalyzer(t *testing.T) {
	config := DefaultQualityMetricsConfig()
	qm := NewQualityMetrics(config)
	qa := NewQualityAnalyzer(qm, nil)

	if qa == nil {
		t.Fatal("NewQualityAnalyzer returned nil")
	}

	if qa.metrics != qm {
		t.Error("analyzer should reference the provided metrics")
	}
}

func TestDetectContradictions(t *testing.T) {
	config := DefaultQualityMetricsConfig()
	qm := NewQualityMetrics(config)
	qa := NewQualityAnalyzer(qm, nil)

	memories := []MemoryWithScore{
		{
			DocID:      "mem1",
			Text:       "The user is online",
			Provenance: NewProvenance(SourceUserInput, "user-1"),
		},
		{
			DocID:      "mem2",
			Text:       "The user is not online",
			Provenance: NewProvenance(SourceUserInput, "user-2"),
		},
		{
			DocID:      "mem3",
			Text:       "The weather is nice",
			Provenance: NewProvenance(SourceUserInput, "user-3"),
		},
	}

	contradictions := qa.detectContradictions(memories)

	// 应该检测到至少 1 个矛盾（mem1 vs mem2）
	if len(contradictions) == 0 {
		t.Error("should detect at least one contradiction")
	}

	// 验证矛盾信息
	found := false
	for _, inc := range contradictions {
		if inc.Type == InconsistencyContradiction {
			if (inc.MemoryID1 == "mem1" && inc.MemoryID2 == "mem2") ||
				(inc.MemoryID1 == "mem2" && inc.MemoryID2 == "mem1") {
				found = true
				break
			}
		}
	}

	if !found {
		t.Error("should detect contradiction between mem1 and mem2")
	}
}

func TestDetectContradictions_Chinese(t *testing.T) {
	config := DefaultQualityMetricsConfig()
	qm := NewQualityMetrics(config)
	qa := NewQualityAnalyzer(qm, nil)

	memories := []MemoryWithScore{
		{
			DocID:      "mem1",
			Text:       "用户有权限访问系统",
			Provenance: NewProvenance(SourceUserInput, "user-1"),
		},
		{
			DocID:      "mem2",
			Text:       "用户没有权限访问系统",
			Provenance: NewProvenance(SourceUserInput, "user-2"),
		},
	}

	contradictions := qa.detectContradictions(memories)

	if len(contradictions) == 0 {
		t.Error("should detect contradiction in Chinese text")
	}
}

func TestDetectDuplicates(t *testing.T) {
	config := DefaultQualityMetricsConfig()
	qm := NewQualityMetrics(config)
	qa := NewQualityAnalyzer(qm, nil)

	memories := []MemoryWithScore{
		{
			DocID: "mem1",
			Text:  "The quick brown fox jumps over the lazy dog",
		},
		{
			DocID: "mem2",
			Text:  "The quick brown fox jumps over the lazy dog",
		},
		{
			DocID: "mem3",
			Text:  "Something completely different",
		},
	}

	duplicates := qa.detectDuplicates(memories)

	// 应该检测到 mem1 和 mem2 是重复的
	if len(duplicates) == 0 {
		t.Error("should detect duplicates")
	}

	// 验证重复信息
	if duplicates[0].Type != InconsistencyDuplicate {
		t.Errorf("type = %s, want %s", duplicates[0].Type, InconsistencyDuplicate)
	}
}

func TestDetectOutdated(t *testing.T) {
	config := DefaultQualityMetricsConfig()
	qm := NewQualityMetrics(config)
	qa := NewQualityAnalyzer(qm, nil)

	// 创建旧记忆（200天前）
	oldProvenance := NewProvenance(SourceUserInput, "user-1")
	oldProvenance.CreatedAt = time.Now().Add(-200 * 24 * time.Hour)

	// 创建新记忆（1天前）
	newProvenance := NewProvenance(SourceUserInput, "user-2")
	newProvenance.CreatedAt = time.Now().Add(-24 * time.Hour)

	memories := []MemoryWithScore{
		{
			DocID:      "old",
			Text:       "Old memory",
			Provenance: oldProvenance,
		},
		{
			DocID:      "new",
			Text:       "New memory",
			Provenance: newProvenance,
		},
	}

	outdated := qa.detectOutdated(memories)

	// 应该只检测到旧记忆
	if len(outdated) != 1 {
		t.Errorf("detected %d outdated memories, want 1", len(outdated))
	}

	if len(outdated) > 0 {
		if outdated[0].Type != InconsistencyOutdated {
			t.Errorf("type = %s, want %s", outdated[0].Type, InconsistencyOutdated)
		}

		if outdated[0].MemoryID1 != "old" {
			t.Errorf("outdated MemoryID = %s, want old", outdated[0].MemoryID1)
		}
	}
}

func TestDetectLowConfidence(t *testing.T) {
	config := DefaultQualityMetricsConfig()
	qm := NewQualityMetrics(config)
	qa := NewQualityAnalyzer(qm, nil)

	highConfProvenance := NewProvenance(SourceUserInput, "user-1")
	highConfProvenance.Confidence = 0.9

	lowConfProvenance := NewProvenance(SourceUserInput, "user-2")
	lowConfProvenance.Confidence = 0.3

	memories := []MemoryWithScore{
		{
			DocID:      "high",
			Text:       "High confidence memory",
			Provenance: highConfProvenance,
		},
		{
			DocID:      "low",
			Text:       "Low confidence memory",
			Provenance: lowConfProvenance,
		},
	}

	lowConf := qa.detectLowConfidence(memories)

	// 应该只检测到低置信度记忆
	if len(lowConf) != 1 {
		t.Errorf("detected %d low confidence memories, want 1", len(lowConf))
	}

	if len(lowConf) > 0 {
		if lowConf[0].Type != InconsistencyLowConfidence {
			t.Errorf("type = %s, want %s", lowConf[0].Type, InconsistencyLowConfidence)
		}

		if lowConf[0].MemoryID1 != "low" {
			t.Errorf("low confidence MemoryID = %s, want low", lowConf[0].MemoryID1)
		}
	}
}

func TestCalculateSimilarity(t *testing.T) {
	config := DefaultQualityMetricsConfig()
	qm := NewQualityMetrics(config)
	qa := NewQualityAnalyzer(qm, nil)

	tests := []struct {
		name    string
		text1   string
		text2   string
		wantMin float64
		wantMax float64
	}{
		{
			name:    "identical texts",
			text1:   "the quick brown fox",
			text2:   "the quick brown fox",
			wantMin: 0.99,
			wantMax: 1.0,
		},
		{
			name:    "similar texts",
			text1:   "the quick brown fox jumps",
			text2:   "the quick brown dog jumps",
			wantMin: 0.6,
			wantMax: 0.9,
		},
		{
			name:    "different texts",
			text1:   "apple banana orange",
			text2:   "car truck bus",
			wantMin: 0.0,
			wantMax: 0.1,
		},
		{
			name:    "empty texts",
			text1:   "",
			text2:   "",
			wantMin: 0.0,
			wantMax: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			similarity := qa.calculateSimilarity(tt.text1, tt.text2)

			if similarity < tt.wantMin || similarity > tt.wantMax {
				t.Errorf("similarity = %.2f, want between %.2f and %.2f",
					similarity, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestDetectInconsistencies(t *testing.T) {
	config := DefaultQualityMetricsConfig()
	qm := NewQualityMetrics(config)
	qa := NewQualityAnalyzer(qm, nil)

	// 创建包含各种问题的记忆集合
	oldProvenance := NewProvenance(SourceUserInput, "user-1")
	oldProvenance.CreatedAt = time.Now().Add(-200 * 24 * time.Hour)
	oldProvenance.Confidence = 0.3

	memories := []MemoryWithScore{
		{
			DocID:      "contradiction1",
			Text:       "The system is running",
			Provenance: NewProvenance(SourceUserInput, "user-1"),
		},
		{
			DocID:      "contradiction2",
			Text:       "The system is not running",
			Provenance: NewProvenance(SourceUserInput, "user-2"),
		},
		{
			DocID:      "duplicate1",
			Text:       "User logged in successfully",
			Provenance: NewProvenance(SourceUserInput, "user-3"),
		},
		{
			DocID:      "duplicate2",
			Text:       "User logged in successfully",
			Provenance: NewProvenance(SourceUserInput, "user-4"),
		},
		{
			DocID:      "old-and-low-conf",
			Text:       "Old and unreliable information",
			Provenance: oldProvenance,
		},
	}

	inconsistencies, err := qa.DetectInconsistencies(context.Background(), memories)
	if err != nil {
		t.Fatalf("DetectInconsistencies failed: %v", err)
	}

	// 应该检测到多种类型的不一致性
	if len(inconsistencies) == 0 {
		t.Error("should detect inconsistencies")
	}

	// 统计各类型
	types := make(map[InconsistencyType]int)
	for _, inc := range inconsistencies {
		types[inc.Type]++
	}

	if types[InconsistencyContradiction] == 0 {
		t.Error("should detect contradictions")
	}

	if types[InconsistencyDuplicate] == 0 {
		t.Error("should detect duplicates")
	}
}

func TestGenerateReport(t *testing.T) {
	config := DefaultQualityMetricsConfig()
	qm := NewQualityMetrics(config)
	qa := NewQualityAnalyzer(qm, nil)

	// 评估一些记忆
	memories := []MemoryWithScore{}
	for i := range 5 {
		prov := NewProvenance(SourceUserInput, "user-1")
		prov.Confidence = float64(i+1) * 0.15

		mem := &MemoryWithScore{
			DocID:      string(rune('A' + i)),
			Text:       "Test memory content with varying quality scores",
			Provenance: prov,
			Score:      float64(i+1) * 0.15,
		}

		_, _ = qm.Evaluate(context.Background(), string(rune('A'+i)), mem)
		memories = append(memories, *mem)
	}

	// 生成报告
	report, err := qa.GenerateReport(context.Background(), memories)
	if err != nil {
		t.Fatalf("GenerateReport failed: %v", err)
	}

	if report == nil {
		t.Fatal("report should not be nil")
	}

	// 验证报告内容
	if report.Stats.TotalMemories != 5 {
		t.Errorf("TotalMemories = %d, want 5", report.Stats.TotalMemories)
	}

	if len(report.DimensionScores) == 0 {
		t.Error("DimensionScores should not be empty")
	}

	// 验证建议不为空
	if len(report.Recommendations) == 0 {
		t.Error("Recommendations should not be empty for varying quality scores")
	}
}

func TestGenerateRecommendations(t *testing.T) {
	config := DefaultQualityMetricsConfig()
	qm := NewQualityMetrics(config)
	qa := NewQualityAnalyzer(qm, nil)

	// 创建低质量场景的报告
	report := &QualityReport{
		Stats: QualityStats{
			AverageQuality:  0.3,
			TotalMemories:   10,
			LowQualityCount: 5,
		},
		DimensionScores: map[QualityDimension]float64{
			QualityAccuracy:     0.4,
			QualityCompleteness: 0.5,
			QualityConsistency:  0.3,
		},
		InconsistencyCount: map[InconsistencyType]int{
			InconsistencyContradiction: 3,
			InconsistencyDuplicate:     5,
		},
	}

	recommendations := qa.generateRecommendations(report)

	// 应该生成多条建议
	if len(recommendations) == 0 {
		t.Error("should generate recommendations for low quality scenario")
	}

	// 验证建议内容相关性
	hasQualityRecommendation := false
	for _, rec := range recommendations {
		if strings.Contains(rec, "质量") || strings.Contains(rec, "矛盾") || strings.Contains(rec, "重复") {
			hasQualityRecommendation = true
			break
		}
	}

	if !hasQualityRecommendation {
		t.Error("recommendations should address quality issues")
	}
}

func TestSuggestImprovements(t *testing.T) {
	config := DefaultQualityMetricsConfig()
	config.MinQualityThreshold = 0.5
	qm := NewQualityMetrics(config)
	qa := NewQualityAnalyzer(qm, nil)

	// 添加低质量记忆
	lowQualityMem := &MemoryWithScore{
		DocID:      "low-quality",
		Text:       "Short",
		Provenance: NewProvenance(SourceUserInput, "user-1"),
		Score:      0.2,
	}
	lowQualityMem.Provenance.Confidence = 0.3

	_, _ = qm.Evaluate(context.Background(), "low-quality", lowQualityMem)

	// 获取改进建议
	improvements, err := qa.SuggestImprovements(context.Background())
	if err != nil {
		t.Fatalf("SuggestImprovements failed: %v", err)
	}

	// 应该有改进建议
	if len(improvements) == 0 {
		t.Error("should have improvement suggestions for low quality memory")
	}

	// 验证建议内容
	if len(improvements) > 0 {
		imp := improvements[0]

		if imp.MemoryID != "low-quality" {
			t.Errorf("MemoryID = %s, want low-quality", imp.MemoryID)
		}

		if len(imp.Suggestions) == 0 {
			t.Error("should have specific suggestions")
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short string",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "exact length",
			input:  "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "long string",
			input:  "hello world this is a long string",
			maxLen: 10,
			want:   "hello worl...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate() = %q, want %q", got, tt.want)
			}
		})
	}
}
