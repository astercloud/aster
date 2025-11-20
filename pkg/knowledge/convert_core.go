package knowledge

import "github.com/astercloud/aster/pkg/knowledge/core"

// 将核心管线的 SearchHit 转为高级 SearchResult（最小映射）。
func convertCoreHits(hits []core.SearchHit) []*SearchResult {
	results := make([]*SearchResult, 0, len(hits))
	for _, h := range hits {
		results = append(results, &SearchResult{
			Item: KnowledgeItem{
				ID:        h.ID,
				Content:   h.Text,
				Metadata:  h.Metadata,
				Namespace: "",
			},
			Score: h.Score,
		})
	}
	return results
}
