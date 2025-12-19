package knowledge

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/astercloud/aster/pkg/knowledge/core"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/vector"
)

// Factory 提供基于 core Pipeline 的 add/search 工具，不默认注册到 builtin registry。
type Factory struct {
	Pipeline *core.Pipeline
}

// NewFactoryPipeline 是一个便捷构造：传入 store + embedder 直接创建 Pipeline。
func NewFactoryPipeline(store vector.VectorStore, embedder vector.Embedder) (*Factory, error) {
	p, err := core.NewPipeline(core.PipelineConfig{
		Store:    store,
		Embedder: embedder,
	})
	if err != nil {
		return nil, err
	}
	return &Factory{Pipeline: p}, nil
}

func NewFactory(p *core.Pipeline) *Factory {
	return &Factory{Pipeline: p}
}

// KnowledgeAddTool 将文本写入核心管线。
func (f *Factory) KnowledgeAddTool() (tools.Tool, error) {
	if f.Pipeline == nil {
		return nil, errors.New("knowledge tool: pipeline is nil")
	}
	return &addTool{pipe: f.Pipeline}, nil
}

// KnowledgeSearchTool 在核心管线中执行向量检索。
func (f *Factory) KnowledgeSearchTool() (tools.Tool, error) {
	if f.Pipeline == nil {
		return nil, errors.New("knowledge tool: pipeline is nil")
	}
	return &searchTool{pipe: f.Pipeline}, nil
}

type addTool struct {
	pipe *core.Pipeline
}

func (t *addTool) Name() string { return "KnowledgeAdd" }
func (t *addTool) Description() string {
	return "Add plain text into knowledge vector store (core pipeline)."
}
func (t *addTool) Prompt() string { return "" }

func (t *addTool) InputSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"text"},
		"properties": map[string]any{
			"id":        map[string]any{"type": "string", "description": "optional document id"},
			"text":      map[string]any{"type": "string", "description": "content to ingest"},
			"namespace": map[string]any{"type": "string", "description": "optional namespace override"},
			"metadata":  map[string]any{"type": "object", "description": "custom metadata map"},
		},
	}
}

func (t *addTool) Execute(ctx context.Context, input map[string]any, _ *tools.ToolContext) (any, error) {
	text, _ := input["text"].(string)
	if text == "" {
		return map[string]any{"ok": false, "error": "text is required"}, nil
	}
	id, _ := input["id"].(string)
	if id == "" {
		id = fmt.Sprintf("doc-%d", time.Now().UnixNano())
	}
	ns, _ := input["namespace"].(string)

	meta := map[string]any{}
	if m, ok := input["metadata"].(map[string]any); ok {
		maps.Copy(meta, m)
	}
	if ns != "" {
		meta["namespace"] = ns
	}

	chunks, err := t.pipe.Ingest(ctx, core.IngestRequest{
		ID:        id,
		Text:      text,
		Namespace: ns,
		Metadata:  meta,
	})
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}, nil
	}

	chunkIDs := make([]string, 0, len(chunks))
	for _, c := range chunks {
		chunkIDs = append(chunkIDs, c.ID)
	}

	return map[string]any{
		"ok":          true,
		"doc_id":      id,
		"chunk_ids":   chunkIDs,
		"namespace":   ns,
		"chunk_count": len(chunks),
	}, nil
}

type searchTool struct {
	pipe *core.Pipeline
}

func (t *searchTool) Name() string        { return "KnowledgeSearch" }
func (t *searchTool) Description() string { return "Semantic search over core knowledge pipeline." }
func (t *searchTool) Prompt() string      { return "" }

func (t *searchTool) InputSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"query"},
		"properties": map[string]any{
			"query":     map[string]any{"type": "string"},
			"top_k":     map[string]any{"type": "integer"},
			"namespace": map[string]any{"type": "string"},
			"metadata":  map[string]any{"type": "object"},
		},
	}
}

func (t *searchTool) Execute(ctx context.Context, input map[string]any, _ *tools.ToolContext) (any, error) {
	query, _ := input["query"].(string)
	if query == "" {
		return map[string]any{"ok": false, "error": "query is required"}, nil
	}
	topK := 5
	if v, ok := input["top_k"].(int); ok && v > 0 {
		topK = v
	}
	ns, _ := input["namespace"].(string)
	meta := map[string]any{}
	if m, ok := input["metadata"].(map[string]any); ok {
		maps.Copy(meta, m)
	}
	if ns != "" {
		meta["namespace"] = ns
	}

	hits, err := t.pipe.Search(ctx, query, topK, meta)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}, nil
	}

	results := make([]map[string]any, 0, len(hits))
	for _, h := range hits {
		results = append(results, map[string]any{
			"id":       h.ID,
			"score":    h.Score,
			"text":     h.Text,
			"metadata": h.Metadata,
		})
	}

	return map[string]any{
		"ok":        true,
		"query":     query,
		"namespace": ns,
		"results":   results,
	}, nil
}
