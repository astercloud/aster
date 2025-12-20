package memory

import (
	"testing"
	"time"
)

func TestNewProvenance(t *testing.T) {
	tests := []struct {
		name           string
		sourceType     SourceType
		sourceID       string
		wantConfidence float64
	}{
		{
			name:           "bootstrapped source",
			sourceType:     SourceBootstrapped,
			sourceID:       "crm-001",
			wantConfidence: 0.95,
		},
		{
			name:           "user input source",
			sourceType:     SourceUserInput,
			sourceID:       "session-123",
			wantConfidence: 0.70,
		},
		{
			name:           "agent source",
			sourceType:     SourceAgent,
			sourceID:       "agent-gpt4",
			wantConfidence: 0.60,
		},
		{
			name:           "tool output source",
			sourceType:     SourceToolOutput,
			sourceID:       "tool-search",
			wantConfidence: 0.50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewProvenance(tt.sourceType, tt.sourceID)

			if p.SourceType != tt.sourceType {
				t.Errorf("SourceType = %v, want %v", p.SourceType, tt.sourceType)
			}

			if p.Confidence != tt.wantConfidence {
				t.Errorf("Confidence = %v, want %v", p.Confidence, tt.wantConfidence)
			}

			if len(p.Sources) != 1 || p.Sources[0] != tt.sourceID {
				t.Errorf("Sources = %v, want [%v]", p.Sources, tt.sourceID)
			}

			if p.Version != 1 {
				t.Errorf("Version = %v, want 1", p.Version)
			}

			if p.IsExplicit {
				t.Error("IsExplicit should be false for implicit provenance")
			}
		})
	}
}

func TestNewExplicitProvenance(t *testing.T) {
	p := NewExplicitProvenance(SourceUserInput, "session-456")

	if !p.IsExplicit {
		t.Error("IsExplicit should be true for explicit provenance")
	}

	if p.Confidence != 0.90 {
		t.Errorf("Explicit user input confidence = %v, want 0.90", p.Confidence)
	}
}

func TestProvenance_AddSource(t *testing.T) {
	p := NewProvenance(SourceUserInput, "source-1")
	initialVersion := p.Version

	p.AddSource("source-2")

	if len(p.Sources) != 2 {
		t.Errorf("Sources length = %d, want 2", len(p.Sources))
	}

	if p.Sources[1] != "source-2" {
		t.Errorf("Second source = %v, want source-2", p.Sources[1])
	}

	if p.Version != initialVersion+1 {
		t.Errorf("Version = %v, want %v", p.Version, initialVersion+1)
	}

	// Test duplicate source
	p.AddSource("source-2")
	if len(p.Sources) != 2 {
		t.Error("Should not add duplicate source")
	}
}

func TestProvenance_Corroborate(t *testing.T) {
	p := NewProvenance(SourceUserInput, "source-1")
	initialConfidence := p.Confidence

	p.Corroborate("source-2")

	if p.CorroborationCount != 1 {
		t.Errorf("CorroborationCount = %v, want 1", p.CorroborationCount)
	}

	if p.Confidence <= initialConfidence {
		t.Errorf("Confidence should increase after corroboration, got %v", p.Confidence)
	}

	// Multiple corroborations
	for i := range 5 {
		p.Corroborate("source-" + string(rune('3'+i)))
	}

	// Should cap at initial + 0.20
	maxConfidence := initialConfidence + 0.20
	if p.Confidence > maxConfidence+0.01 { // Allow small floating point error
		t.Errorf("Confidence = %v, should not exceed %v", p.Confidence, maxConfidence)
	}
}

func TestProvenance_ToMetadata(t *testing.T) {
	p := NewProvenance(SourceUserInput, "session-123")
	p.Tags = []string{"important", "customer"}

	meta := p.ToMetadata()

	provData, ok := meta["provenance"].(map[string]any)
	if !ok {
		t.Fatal("provenance not found in metadata")
	}

	if provData["source_type"] != string(SourceUserInput) {
		t.Errorf("source_type = %v, want %v", provData["source_type"], SourceUserInput)
	}

	if provData["confidence"] != p.Confidence {
		t.Errorf("confidence = %v, want %v", provData["confidence"], p.Confidence)
	}

	sources, ok := provData["sources"].([]string)
	if !ok || len(sources) != 1 || sources[0] != "session-123" {
		t.Errorf("sources = %v, want [session-123]", provData["sources"])
	}

	tags, ok := provData["tags"].([]string)
	if !ok || len(tags) != 2 {
		t.Errorf("tags = %v, want 2 tags", provData["tags"])
	}
}

func TestFromMetadata(t *testing.T) {
	original := NewProvenance(SourceUserInput, "session-789")
	original.Tags = []string{"test"}
	original.Corroborate("session-790")

	meta := original.ToMetadata()
	restored := FromMetadata(meta)

	if restored == nil {
		t.Fatal("FromMetadata returned nil")
	}

	if restored.SourceType != original.SourceType {
		t.Errorf("SourceType = %v, want %v", restored.SourceType, original.SourceType)
	}

	if restored.Confidence != original.Confidence {
		t.Errorf("Confidence = %v, want %v", restored.Confidence, original.Confidence)
	}

	if len(restored.Sources) != len(original.Sources) {
		t.Errorf("Sources length = %d, want %d", len(restored.Sources), len(original.Sources))
	}

	if restored.Version != original.Version {
		t.Errorf("Version = %v, want %v", restored.Version, original.Version)
	}

	if restored.CorroborationCount != original.CorroborationCount {
		t.Errorf("CorroborationCount = %v, want %v", restored.CorroborationCount, original.CorroborationCount)
	}

	if len(restored.Tags) != len(original.Tags) {
		t.Errorf("Tags length = %d, want %d", len(restored.Tags), len(original.Tags))
	}
}

func TestProvenance_MarkAccessed(t *testing.T) {
	p := NewProvenance(SourceUserInput, "session-1")

	if p.LastAccessedAt != nil {
		t.Error("LastAccessedAt should be nil initially")
	}

	p.MarkAccessed()

	if p.LastAccessedAt == nil {
		t.Fatal("LastAccessedAt should be set after MarkAccessed")
	}

	if time.Since(*p.LastAccessedAt) > time.Second {
		t.Error("LastAccessedAt should be very recent")
	}
}

func TestProvenance_Age(t *testing.T) {
	p := NewProvenance(SourceUserInput, "session-1")

	// Sleep a tiny bit to ensure age > 0
	time.Sleep(10 * time.Millisecond)

	age := p.Age()
	if age <= 0 {
		t.Error("Age should be positive")
	}

	if age > time.Second {
		t.Error("Age should be very small for newly created provenance")
	}
}

func TestProvenance_Freshness(t *testing.T) {
	p := NewProvenance(SourceUserInput, "session-1")

	time.Sleep(10 * time.Millisecond)

	// Update the provenance
	p.AddSource("session-2")

	freshness := p.Freshness()
	if freshness <= 0 {
		t.Error("Freshness should be positive")
	}

	// Freshness should be smaller than age after update
	if freshness > time.Second {
		t.Error("Freshness should be very small after recent update")
	}
}
