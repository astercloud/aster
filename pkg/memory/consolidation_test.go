package memory

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// MockLLMProvider 模拟 LLM 提供者。
type MockLLMProvider struct {
	Response  string
	Error     error
	CallCount int
}

func (m *MockLLMProvider) Complete(ctx context.Context, prompt string, options map[string]any) (string, error) {
	m.CallCount++
	if m.Error != nil {
		return "", m.Error
	}
	if m.Response != "" {
		return m.Response, nil
	}

	// 默认行为：简单合并所有记忆
	if strings.Contains(prompt, "Redundant Memories:") {
		return "This is a consolidated memory combining all redundant information.", nil
	}
	if strings.Contains(prompt, "Conflicting Memories:") {
		return "This is a resolved memory based on the most confident sources.", nil
	}
	if strings.Contains(prompt, "Memories to Summarize:") {
		return "This is a summary of all related memories.", nil
	}

	return "Mock LLM response", nil
}

func TestRedundancyStrategy_ShouldConsolidate(t *testing.T) {
	strategy := NewRedundancyStrategy(0.85)

	tests := []struct {
		name       string
		memories   []MemoryWithScore
		wantShould bool
		wantReason ConsolidationReason
	}{
		{
			name: "high similarity - should consolidate",
			memories: []MemoryWithScore{
				{DocID: "1", Text: "The sky is blue", Score: 1.0},
				{DocID: "2", Text: "The sky is blue", Score: 0.95},
				{DocID: "3", Text: "The sky is blue", Score: 0.90},
			},
			wantShould: true,
			wantReason: ReasonRedundant,
		},
		{
			name: "low similarity - should not consolidate",
			memories: []MemoryWithScore{
				{DocID: "1", Text: "The sky is blue", Score: 1.0},
				{DocID: "2", Text: "The grass is green", Score: 0.70},
			},
			wantShould: false,
			wantReason: ReasonNone,
		},
		{
			name: "single memory - should not consolidate",
			memories: []MemoryWithScore{
				{DocID: "1", Text: "The sky is blue", Score: 1.0},
			},
			wantShould: false,
			wantReason: ReasonNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldConsolidate, reason := strategy.ShouldConsolidate(context.Background(), tt.memories)
			if shouldConsolidate != tt.wantShould {
				t.Errorf("ShouldConsolidate() = %v, want %v", shouldConsolidate, tt.wantShould)
			}
			if reason != tt.wantReason {
				t.Errorf("Reason = %v, want %v", reason, tt.wantReason)
			}
		})
	}
}

func TestRedundancyStrategy_Consolidate(t *testing.T) {
	strategy := NewRedundancyStrategy(0.85)
	llm := &MockLLMProvider{
		Response: "User prefers dark mode theme",
	}

	memories := []MemoryWithScore{
		{
			DocID: "1",
			Text:  "User likes dark mode",
			Metadata: map[string]any{
				"tags": []string{"preference"},
			},
			Provenance: NewProvenance(SourceUserInput, "user-123"),
			Score:      1.0,
		},
		{
			DocID: "2",
			Text:  "User prefers dark theme",
			Metadata: map[string]any{
				"tags": []string{"preference", "theme"},
			},
			Provenance: NewProvenance(SourceUserInput, "user-123"),
			Score:      0.95,
		},
	}

	consolidated, err := strategy.Consolidate(context.Background(), memories, llm)
	if err != nil {
		t.Fatalf("Consolidate() error = %v", err)
	}

	if consolidated == nil {
		t.Fatal("Consolidate() returned nil")
	}

	// 验证合并结果
	if consolidated.Text == "" {
		t.Error("Consolidated text is empty")
	}

	if len(consolidated.SourceMemories) != 2 {
		t.Errorf("SourceMemories length = %d, want 2", len(consolidated.SourceMemories))
	}

	if consolidated.Reason != ReasonRedundant {
		t.Errorf("Reason = %v, want %v", consolidated.Reason, ReasonRedundant)
	}

	// 验证 LLM 被调用
	if llm.CallCount != 1 {
		t.Errorf("LLM CallCount = %d, want 1", llm.CallCount)
	}

	// 验证溯源合并
	if consolidated.Provenance == nil {
		t.Error("Provenance is nil")
	} else {
		if consolidated.Provenance.CorroborationCount != 2 {
			t.Errorf("CorroborationCount = %d, want 2", consolidated.Provenance.CorroborationCount)
		}
	}
}

func TestConflictResolutionStrategy_DetectConflict(t *testing.T) {
	strategy := NewConflictResolutionStrategy(0.75, nil)

	tests := []struct {
		name         string
		memories     []MemoryWithScore
		wantConflict bool
	}{
		{
			name: "contains conflict keywords",
			memories: []MemoryWithScore{
				{DocID: "1", Text: "User likes coffee"},
				{DocID: "2", Text: "User actually prefers tea"},
			},
			wantConflict: true,
		},
		{
			name: "no conflict keywords",
			memories: []MemoryWithScore{
				{DocID: "1", Text: "User likes coffee"},
				{DocID: "2", Text: "User enjoys tea"},
			},
			wantConflict: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasConflict := strategy.detectConflict(tt.memories)
			if hasConflict != tt.wantConflict {
				t.Errorf("detectConflict() = %v, want %v", hasConflict, tt.wantConflict)
			}
		})
	}
}

func TestConflictResolutionStrategy_SelectBestProvenance(t *testing.T) {
	calc := NewConfidenceCalculator(ConfidenceConfig{
		DecayHalfLife: 7 * 24 * time.Hour,
	})
	strategy := NewConflictResolutionStrategy(0.75, calc)

	now := time.Now()
	highConfidence := &MemoryProvenance{
		SourceType: SourceUserInput,
		Confidence: 0.9,
		CreatedAt:  now,
		UpdatedAt:  now,
		IsExplicit: true,
	}

	lowConfidence := &MemoryProvenance{
		SourceType: SourceAgent,
		Confidence: 0.5,
		CreatedAt:  now.Add(-30 * 24 * time.Hour),
		UpdatedAt:  now.Add(-30 * 24 * time.Hour),
		IsExplicit: false,
	}

	memories := []MemoryWithScore{
		{DocID: "1", Provenance: lowConfidence},
		{DocID: "2", Provenance: highConfidence},
	}

	best := strategy.selectBestProvenance(memories)
	if best == nil {
		t.Fatal("selectBestProvenance() returned nil")
	}

	// 应该选择高置信度的
	if best.IsExplicit != highConfidence.IsExplicit {
		t.Error("Did not select the high confidence provenance")
	}
}

func TestSummarizationStrategy(t *testing.T) {
	strategy := NewSummarizationStrategy(3)
	llm := &MockLLMProvider{
		Response: "User enjoys outdoor activities including hiking and cycling.",
	}

	tests := []struct {
		name       string
		memories   []MemoryWithScore
		wantShould bool
	}{
		{
			name: "enough memories to summarize",
			memories: []MemoryWithScore{
				{DocID: "1", Text: "User likes hiking"},
				{DocID: "2", Text: "User enjoys cycling"},
				{DocID: "3", Text: "User loves outdoor activities"},
			},
			wantShould: true,
		},
		{
			name: "not enough memories",
			memories: []MemoryWithScore{
				{DocID: "1", Text: "User likes hiking"},
				{DocID: "2", Text: "User enjoys cycling"},
			},
			wantShould: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldConsolidate, reason := strategy.ShouldConsolidate(context.Background(), tt.memories)
			if shouldConsolidate != tt.wantShould {
				t.Errorf("ShouldConsolidate() = %v, want %v", shouldConsolidate, tt.wantShould)
			}

			if tt.wantShould {
				if reason != ReasonSummary {
					t.Errorf("Reason = %v, want %v", reason, ReasonSummary)
				}

				// 测试合并
				consolidated, err := strategy.Consolidate(context.Background(), tt.memories, llm)
				if err != nil {
					t.Fatalf("Consolidate() error = %v", err)
				}

				if consolidated.Text == "" {
					t.Error("Consolidated text is empty")
				}

				if len(consolidated.SourceMemories) != len(tt.memories) {
					t.Errorf("SourceMemories length = %d, want %d", len(consolidated.SourceMemories), len(tt.memories))
				}
			}
		})
	}
}

func TestConsolidationEngine_Stats(t *testing.T) {
	strategy := NewRedundancyStrategy(0.85)
	llm := &MockLLMProvider{}
	config := DefaultConsolidationConfig()

	// 创建一个 nil 的 SemanticMemory（仅用于测试统计功能）
	engine := NewConsolidationEngine(nil, strategy, llm, config)

	// 验证初始统计
	stats := engine.GetStats()
	if stats.ConsolidationCount != 0 {
		t.Errorf("Initial ConsolidationCount = %d, want 0", stats.ConsolidationCount)
	}
	if stats.MergedMemoriesCount != 0 {
		t.Errorf("Initial MergedMemoriesCount = %d, want 0", stats.MergedMemoriesCount)
	}
}

func TestConsolidationEngine_ShouldAutoConsolidate(t *testing.T) {
	config := DefaultConsolidationConfig()
	config.AutoConsolidateInterval = 1 * time.Hour

	strategy := NewRedundancyStrategy(0.85)
	llm := &MockLLMProvider{}

	engine := NewConsolidationEngine(nil, strategy, llm, config)

	// 刚创建的引擎不应该立即触发
	if engine.ShouldAutoConsolidate() {
		t.Error("Newly created engine should not trigger auto-consolidation")
	}

	// 模拟时间流逝
	engine.lastConsolidation = time.Now().Add(-2 * time.Hour)
	if !engine.ShouldAutoConsolidate() {
		t.Error("Engine should trigger auto-consolidation after interval")
	}
}

func TestMergeMetadata(t *testing.T) {
	strategy := NewRedundancyStrategy(0.85)

	memories := []MemoryWithScore{
		{
			DocID: "1",
			Metadata: map[string]any{
				"tags": []string{"tag1", "tag2"},
			},
		},
		{
			DocID: "2",
			Metadata: map[string]any{
				"tags": []string{"tag2", "tag3"},
			},
		},
	}

	merged := strategy.mergeMetadata(memories)

	if merged == nil {
		t.Fatal("mergeMetadata() returned nil")
	}

	// 验证标签合并
	tags, ok := merged["tags"].([]string)
	if !ok {
		t.Fatal("Tags are not []string")
	}

	// 应该包含所有唯一标签
	if len(tags) < 2 {
		t.Errorf("Merged tags length = %d, want at least 2", len(tags))
	}

	// 验证源数量
	sourceCount, ok := merged["source_count"].(int)
	if !ok || sourceCount != 2 {
		t.Errorf("source_count = %v, want 2", sourceCount)
	}
}

func TestLLMProviderError(t *testing.T) {
	strategy := NewRedundancyStrategy(0.85)
	llm := &MockLLMProvider{
		Error: errors.New("LLM API error"),
	}

	memories := []MemoryWithScore{
		{DocID: "1", Text: "Test memory 1"},
		{DocID: "2", Text: "Test memory 2"},
	}

	_, err := strategy.Consolidate(context.Background(), memories, llm)
	if err == nil {
		t.Error("Expected error from LLM failure, got nil")
	}

	if !strings.Contains(err.Error(), "LLM") {
		t.Errorf("Error message should mention LLM: %v", err)
	}
}
