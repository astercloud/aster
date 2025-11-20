package simple

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/astercloud/aster/pkg/vector"
)

// PipelineConfig 简化知识管线配置。
type PipelineConfig struct {
	Store       vector.VectorStore
	Embedder    vector.Embedder
	Namespace   string
	DefaultTopK int
}

// Pipeline 提供最小可用的知识入库与检索流程。
// 目标：少依赖、易落地，便于示例/测试快速验证 RAG。
type Pipeline struct {
	store     vector.VectorStore
	embedder  vector.Embedder
	namespace string
	defaultK  int
}

// NewPipeline 创建知识管线。
func NewPipeline(cfg PipelineConfig) (*Pipeline, error) {
	if cfg.Store == nil {
		return nil, fmt.Errorf("knowledge: store is required")
	}
	if cfg.Embedder == nil {
		return nil, fmt.Errorf("knowledge: embedder is required")
	}
	if cfg.DefaultTopK <= 0 {
		cfg.DefaultTopK = 5
	}
	ns := cfg.Namespace
	if ns == "" {
		ns = "default"
	}
	return &Pipeline{
		store:     cfg.Store,
		embedder:  cfg.Embedder,
		namespace: ns,
		defaultK:  cfg.DefaultTopK,
	}, nil
}

var (
	defaultOnce sync.Once
	defaultPipe *Pipeline
	defaultErr  error
)

// DefaultInMemoryPipeline 返回基于内存向量库和 MockEmbedder 的默认管线（单例）。
func DefaultInMemoryPipeline() (*Pipeline, error) {
	defaultOnce.Do(func() {
		defaultPipe, defaultErr = NewPipeline(PipelineConfig{
			Store:       vector.NewMemoryStore(),
			Embedder:    vector.NewMockEmbedder(32),
			Namespace:   "default",
			DefaultTopK: 5,
		})
	})
	return defaultPipe, defaultErr
}

// UpsertText 将纯文本写入向量库。
// - 自动按段落切分；ID 会追加 chunk 索引（id#0, id#1...）
// - metadata 会透传到向量文档。
func (p *Pipeline) UpsertText(ctx context.Context, id, text string, metadata map[string]interface{}) ([]string, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("knowledge: id is required")
	}
	chunks := splitParagraphs(text)
	if len(chunks) == 0 {
		return nil, fmt.Errorf("knowledge: text is empty")
	}

	vecs, err := p.embedder.EmbedText(ctx, chunks)
	if err != nil {
		return nil, fmt.Errorf("embed text: %w", err)
	}
	if len(vecs) != len(chunks) {
		return nil, fmt.Errorf("embedder returned %d vectors for %d chunks", len(vecs), len(chunks))
	}

	docs := make([]vector.Document, 0, len(chunks))
	ids := make([]string, 0, len(chunks))
	ns := p.namespace
	if metadata != nil {
		if override, ok := metadata["namespace"].(string); ok && strings.TrimSpace(override) != "" {
			ns = strings.TrimSpace(override)
		}
	}
	for i, chunk := range chunks {
		chunkID := fmt.Sprintf("%s#%d", id, i)
		metaCopy := make(map[string]interface{}, len(metadata)+2)
		for k, v := range metadata {
			metaCopy[k] = v
		}
		metaCopy["text"] = chunk
		metaCopy["chunk_index"] = i

		docs = append(docs, vector.Document{
			ID:        chunkID,
			Text:      chunk,
			Embedding: vecs[i],
			Metadata:  metaCopy,
			Namespace: ns,
		})
		ids = append(ids, chunkID)
	}

	if err := p.store.Upsert(ctx, docs); err != nil {
		return nil, fmt.Errorf("upsert vector docs: %w", err)
	}
	return ids, nil
}

// Search 根据查询语句执行向量检索。
// 返回向量命中列表；不在此处做 rerank/过滤，保持简洁。
func (p *Pipeline) Search(ctx context.Context, query string, topK int, metadata map[string]interface{}) ([]vector.Hit, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("knowledge: query is empty")
	}
	if topK <= 0 {
		topK = p.defaultK
	}

	vecs, err := p.embedder.EmbedText(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	if len(vecs) == 0 {
		return nil, fmt.Errorf("embedder returned empty vectors")
	}

	ns := p.namespace
	if metadata != nil {
		if override, ok := metadata["namespace"].(string); ok && strings.TrimSpace(override) != "" {
			ns = strings.TrimSpace(override)
		}
	}

	return p.store.Query(ctx, vector.Query{
		Vector:    vecs[0],
		TopK:      topK,
		Namespace: ns,
		Filter:    metadata,
	})
}

func splitParagraphs(text string) []string {
	raw := strings.Split(text, "\n\n")
	out := make([]string, 0, len(raw))
	for _, seg := range raw {
		seg = strings.TrimSpace(seg)
		if seg != "" {
			out = append(out, seg)
		}
	}
	return out
}
