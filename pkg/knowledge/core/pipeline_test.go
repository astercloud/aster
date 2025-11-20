package core

import (
	"context"
	"testing"

	"github.com/astercloud/aster/pkg/vector"
)

func TestPipeline_IngestAndSearch(t *testing.T) {
	pipe, err := NewPipeline(PipelineConfig{
		Store:       vector.NewMemoryStore(),
		Embedder:    vector.NewMockEmbedder(16),
		Namespace:   "demo",
		DefaultTopK: 3,
	})
	if err != nil {
		t.Fatalf("new pipeline: %v", err)
	}

	chunks, err := pipe.Ingest(context.Background(), IngestRequest{
		ID:        "doc1",
		Text:      "Aster is built in Go.\n\nIt supports workflows and memory.",
		Namespace: "demo",
		Metadata: map[string]interface{}{
			"source": "test",
		},
	})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}

	hits, err := pipe.Search(context.Background(), "workflows", 2, map[string]interface{}{
		"namespace": "demo",
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) == 0 {
		t.Fatalf("expected hits")
	}
	if hits[0].Metadata["source"] != "test" {
		t.Fatalf("metadata not propagated")
	}
}

func TestPipeline_IngestEmpty(t *testing.T) {
	pipe, _ := NewPipeline(PipelineConfig{
		Store:    vector.NewMemoryStore(),
		Embedder: vector.NewMockEmbedder(8),
	})
	_, err := pipe.Ingest(context.Background(), IngestRequest{Text: "   "})
	if err == nil {
		t.Fatalf("expected error for empty text")
	}
}
