package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/astercloud/aster/pkg/knowledge/core"
	"github.com/astercloud/aster/pkg/vector"
)

// 一个最小的 RAG 示例：使用核心管线 + 内存向量库 + MockEmbedder。
func main() {
	pipe, err := core.NewPipeline(core.PipelineConfig{
		Store:       vector.NewMemoryStore(),
		Embedder:    vector.NewMockEmbedder(32),
		Namespace:   "demo",
		DefaultTopK: 3,
	})
	if err != nil {
		log.Fatalf("init pipeline: %v", err)
	}

	ctx := context.Background()

	// 入库
	text := `Aster 是面向生产的多 Agent 框架。
它内置工作记忆、语义记忆、Workflow 编排以及丰富的安全护栏。`

	chunks, err := pipe.Ingest(ctx, core.IngestRequest{
		ID:        "intro",
		Text:      text,
		Namespace: "demo",
		Metadata: map[string]interface{}{
			"source": "docs/intro",
		},
	})
	if err != nil {
		log.Fatalf("ingest: %v", err)
	}
	fmt.Fprintf(os.Stdout, "Inserted %d chunks\n", len(chunks))

	// 检索
	hits, err := pipe.Search(ctx, "什么是 Aster？", 3, map[string]interface{}{
		"namespace": "demo",
	})
	if err != nil {
		log.Fatalf("search: %v", err)
	}

	fmt.Fprintf(os.Stdout, "Top hits:\n")
	for i, h := range hits {
		fmt.Fprintf(os.Stdout, "%d) score=%.4f text=%.50s...\n", i+1, h.Score, h.Text)
	}
}
