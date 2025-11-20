package simple

import (
	"context"
	"testing"

	"github.com/astercloud/aster/pkg/vector"
)

func TestPipeline_UpsertAndSearch(t *testing.T) {
	store := vector.NewMemoryStore()
	embedder := vector.NewMockEmbedder(8)

	pipe, err := NewPipeline(PipelineConfig{
		Store:       store,
		Embedder:    embedder,
		Namespace:   "test",
		DefaultTopK: 3,
	})
	if err != nil {
		t.Fatalf("new pipeline: %v", err)
	}

	text := "Go is great for high-performance systems.\n\nIt has built-in concurrency with goroutines."
	ids, err := pipe.UpsertText(context.Background(), "doc1", text, map[string]interface{}{
		"source":    "testdoc",
		"type":      "note",
		"namespace": "test",
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(ids))
	}

	hits, err := pipe.Search(context.Background(), "goroutines", 2, map[string]interface{}{
		"namespace": "test",
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) == 0 {
		t.Fatalf("expected at least 1 hit")
	}
	if hits[0].Metadata["source"] != "testdoc" {
		t.Fatalf("metadata not preserved in hit")
	}
}

func TestPipeline_EmptyText(t *testing.T) {
	store := vector.NewMemoryStore()
	embedder := vector.NewMockEmbedder(4)

	pipe, _ := NewPipeline(PipelineConfig{
		Store:    store,
		Embedder: embedder,
	})

	_, err := pipe.UpsertText(context.Background(), "doc2", "   ", nil)
	if err == nil {
		t.Fatalf("expected error for empty text")
	}
}
