package knowledge

import (
	"context"
	"testing"

	"github.com/astercloud/aster/pkg/knowledge/core"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/vector"
)

func TestFactory_AddAndSearch(t *testing.T) {
	pipe, err := core.NewPipeline(core.PipelineConfig{
		Store:    vector.NewMemoryStore(),
		Embedder: vector.NewMockEmbedder(16),
	})
	if err != nil {
		t.Fatalf("new pipeline: %v", err)
	}

	f := NewFactory(pipe)
	add, _ := f.KnowledgeAddTool()
	search, _ := f.KnowledgeSearchTool()

	_, err = add.Execute(context.Background(), map[string]interface{}{
		"text":      "Go has goroutines for concurrency.",
		"namespace": "ns1",
	}, &tools.ToolContext{})
	if err != nil {
		t.Fatalf("add exec error: %v", err)
	}

	resp, err := search.Execute(context.Background(), map[string]interface{}{
		"query":     "goroutines",
		"namespace": "ns1",
		"top_k":     3,
	}, &tools.ToolContext{})
	if err != nil {
		t.Fatalf("search exec error: %v", err)
	}

	resMap, ok := resp.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected resp type: %T", resp)
	}
	results, ok := resMap["results"].([]map[string]interface{})
	if !ok || len(results) == 0 {
		t.Fatalf("expected results")
	}
}
