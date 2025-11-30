// MemorySemantic 演示 SemanticMemory 的语义检索能力，使用内存 VectorStore
// 和 MockEmbedder 实现基础的向量相似度搜索。
//
// 注意: 该示例仅用于演示接口用法，不适合作为生产环境的 RAG 实现。
package main

import (
	"context"
	"fmt"

	"github.com/astercloud/aster/pkg/memory"
	"github.com/astercloud/aster/pkg/vector"
)

func main() {
	ctx := context.Background()

	// 1. 创建向量存储和 embedder
	store := vector.NewMemoryStore()
	embedder := vector.NewMockEmbedder(16)

	// 2. 创建语义记忆组件
	semMem := memory.NewSemanticMemory(memory.SemanticMemoryConfig{
		Store:          store,
		Embedder:       embedder,
		NamespaceScope: "resource",
		TopK:           3,
	})

	// 3. 索引几段示例文本
	docs := []struct {
		id   string
		text string
		meta map[string]any
	}{
		{
			id:   "doc-1",
			text: "Paris is the capital of France.",
			meta: map[string]any{"user_id": "alice", "resource_id": "europe-notes"},
		},
		{
			id:   "doc-2",
			text: "Berlin is the capital of Germany.",
			meta: map[string]any{"user_id": "alice", "resource_id": "europe-notes"},
		},
		{
			id:   "doc-3",
			text: "Tokyo is the capital of Japan.",
			meta: map[string]any{"user_id": "bob", "resource_id": "asia-notes"},
		},
	}

	for _, d := range docs {
		if err := semMem.Index(ctx, d.id, d.text, d.meta); err != nil {
			panic(fmt.Sprintf("index %s: %v", d.id, err))
		}
	}

	// 4. 在 Alice 的 europe-notes 命名空间内进行语义检索
	query := "What is the capital of France?"
	meta := map[string]any{"user_id": "alice", "resource_id": "europe-notes"}

	hits, err := semMem.Search(ctx, query, meta, 3)
	if err != nil {
		panic(fmt.Sprintf("semantic search failed: %v", err))
	}

	fmt.Printf("Query: %q\n", query)
	fmt.Println("Semantic search hits:")
	for _, h := range hits {
		fmt.Printf("  ID=%s, score=%.4f, metadata=%v\n", h.ID, h.Score, h.Metadata)
	}
}
