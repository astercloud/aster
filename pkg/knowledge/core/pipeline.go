package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/astercloud/aster/pkg/vector"
)

// PipelineConfig 轻量 RAG 管线配置。
type PipelineConfig struct {
	Store       vector.VectorStore
	Embedder    vector.Embedder
	Namespace   string
	DefaultTopK int
}

// Pipeline 提供最小 ingest/search 能力，不依赖高级特性。
type Pipeline struct {
	store     vector.VectorStore
	embedder  vector.Embedder
	namespace string
	defaultK  int
}

// NewPipeline 创建管线实例。
func NewPipeline(cfg PipelineConfig) (*Pipeline, error) {
	if cfg.Store == nil {
		return nil, fmt.Errorf("knowledge core: store is required")
	}
	if cfg.Embedder == nil {
		return nil, fmt.Errorf("knowledge core: embedder is required")
	}
	ns := cfg.Namespace
	if ns == "" {
		ns = "default"
	}
	if cfg.DefaultTopK <= 0 {
		cfg.DefaultTopK = 5
	}
	return &Pipeline{
		store:     cfg.Store,
		embedder:  cfg.Embedder,
		namespace: ns,
		defaultK:  cfg.DefaultTopK,
	}, nil
}

// Ingest 将文本切分并写入向量库。
func (p *Pipeline) Ingest(ctx context.Context, req IngestRequest) ([]Chunk, error) {
	if strings.TrimSpace(req.Text) == "" {
		return nil, fmt.Errorf("knowledge core: text is empty")
	}
	id := strings.TrimSpace(req.ID)
	if id == "" {
		id = fmt.Sprintf("doc-%d", time.Now().UnixNano())
	}
	ns := p.namespace
	if strings.TrimSpace(req.Namespace) != "" {
		ns = strings.TrimSpace(req.Namespace)
	}

	meta := make(map[string]interface{})
	for k, v := range req.Metadata {
		meta[k] = v
	}
	if ns != "" {
		meta["namespace"] = ns
	}

	rawChunks := splitParagraphs(req.Text)
	if len(rawChunks) == 0 {
		return nil, fmt.Errorf("knowledge core: no chunks after split")
	}

	vecs, err := p.embedder.EmbedText(ctx, rawChunks)
	if err != nil {
		return nil, fmt.Errorf("embed text: %w", err)
	}
	if len(vecs) != len(rawChunks) {
		return nil, fmt.Errorf("embedder returned %d vectors for %d chunks", len(vecs), len(rawChunks))
	}

	chunks := make([]Chunk, 0, len(rawChunks))
	docs := make([]vector.Document, 0, len(rawChunks))
	for i, ctext := range rawChunks {
		chunkID := fmt.Sprintf("%s#%d", id, i)
		chunkMeta := make(map[string]interface{}, len(meta)+2)
		for k, v := range meta {
			chunkMeta[k] = v
		}
		chunkMeta["text"] = ctext
		chunkMeta["chunk_index"] = i

		chunks = append(chunks, Chunk{
			ID:        chunkID,
			Text:      ctext,
			Namespace: ns,
			Metadata:  chunkMeta,
		})

		docs = append(docs, vector.Document{
			ID:        chunkID,
			Text:      ctext,
			Embedding: vecs[i],
			Metadata:  chunkMeta,
			Namespace: ns,
		})
	}

	if err := p.store.Upsert(ctx, docs); err != nil {
		return nil, fmt.Errorf("upsert vector docs: %w", err)
	}
	return chunks, nil
}

// Search 执行向量检索。
func (p *Pipeline) Search(ctx context.Context, query string, topK int, metadata map[string]interface{}) ([]SearchHit, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("knowledge core: query is empty")
	}
	if topK <= 0 {
		topK = p.defaultK
	}

	ns := p.namespace
	if metadata != nil {
		if override, ok := metadata["namespace"].(string); ok && strings.TrimSpace(override) != "" {
			ns = strings.TrimSpace(override)
		}
	}

	vecs, err := p.embedder.EmbedText(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	if len(vecs) == 0 {
		return nil, fmt.Errorf("embedder returned empty vectors")
	}

	hits, err := p.store.Query(ctx, vector.Query{
		Vector:    vecs[0],
		TopK:      topK,
		Namespace: ns,
		Filter:    metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("vector query: %w", err)
	}

	out := make([]SearchHit, 0, len(hits))
	for _, h := range hits {
		text := ""
		if h.Metadata != nil {
			if t, ok := h.Metadata["text"].(string); ok {
				text = t
			}
		}
		out = append(out, SearchHit{
			ID:       h.ID,
			Score:    h.Score,
			Text:     text,
			Metadata: h.Metadata,
		})
	}
	return out, nil
}

// splitParagraphs 进行简单段落切分。
func splitParagraphs(text string) []string {
	segs := strings.Split(text, "\n\n")
	out := make([]string, 0, len(segs))
	for _, s := range segs {
		trim := strings.TrimSpace(s)
		if trim != "" {
			out = append(out, trim)
		}
	}
	return out
}
