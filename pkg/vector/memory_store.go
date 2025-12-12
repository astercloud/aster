package vector

import (
	"context"
	"math"
	"sync"
)

// MemoryStore 一个简单的内存向量存储实现, 仅用于示例和测试。
// 不适合生产环境, 但可以帮助用户快速理解接口用法。
type MemoryStore struct {
	mu    sync.RWMutex
	docs  map[string]Document
	index map[string][]string // namespace -> []docID
}

// NewMemoryStore 创建内存向量存储。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		docs:  make(map[string]Document),
		index: make(map[string][]string),
	}
}

// Upsert 将文档插入或更新到内存存储。
func (s *MemoryStore) Upsert(_ context.Context, docs []Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, d := range docs {
		if d.ID == "" {
			continue
		}
		// 简单覆盖策略
		s.docs[d.ID] = d

		ns := d.Namespace
		if ns == "" {
			ns = "default"
		}
		ids := s.index[ns]
		found := false
		for _, id := range ids {
			if id == d.ID {
				found = true
				break
			}
		}
		if !found {
			s.index[ns] = append(ids, d.ID)
		}
	}

	return nil
}

// Delete 从内存存储中删除文档。
func (s *MemoryStore) Delete(_ context.Context, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range ids {
		doc, ok := s.docs[id]
		if !ok {
			continue
		}
		delete(s.docs, id)

		ns := doc.Namespace
		if ns == "" {
			ns = "default"
		}
		orig := s.index[ns]
		dst := orig[:0]
		for _, did := range orig {
			if did != id {
				dst = append(dst, did)
			}
		}
		if len(dst) == 0 {
			delete(s.index, ns)
		} else {
			s.index[ns] = dst
		}
	}

	return nil
}

// Query 在指定命名空间内执行简单的余弦相似度检索。
func (s *MemoryStore) Query(_ context.Context, q Query) ([]Hit, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ns := q.Namespace
	if ns == "" {
		ns = "default"
	}
	docIDs := s.index[ns]
	if len(docIDs) == 0 {
		return nil, nil
	}

	topK := q.TopK
	if topK <= 0 {
		topK = 5
	}

	type scored struct {
		id    string
		score float64
		meta  map[string]any
	}

	var results []scored
	for _, id := range docIDs {
		doc, ok := s.docs[id]
		if !ok || len(doc.Embedding) == 0 {
			continue
		}

		// 多租户过滤
		if q.OrgID != "" && doc.OrgID != q.OrgID {
			continue
		}
		if q.TenantID != "" && doc.TenantID != q.TenantID {
			continue
		}

		// 自定义过滤
		if len(q.Filter) > 0 {
			match := true
			for k, v := range q.Filter {
				if docVal, ok := doc.Metadata[k]; !ok || docVal != v {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}

		score := cosineSimilarity(q.Vector, doc.Embedding)
		if math.IsNaN(score) {
			continue
		}
		results = append(results, scored{
			id:    id,
			score: score,
			meta:  doc.Metadata,
		})
	}

	// 简单排序: 降序
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].score > results[i].score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	if len(results) > topK {
		results = results[:topK]
	}

	hits := make([]Hit, 0, len(results))
	for _, r := range results {
		hits = append(hits, Hit{
			ID:       r.id,
			Score:    r.score,
			Metadata: r.meta,
		})
	}

	return hits, nil
}

// Close 对内存存储无实际作用。
func (s *MemoryStore) Close() error {
	return nil
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		av := float64(a[i])
		bv := float64(b[i])
		dot += av * bv
		na += av * av
		nb += bv * bv
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
