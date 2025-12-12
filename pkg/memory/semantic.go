package memory

import (
	"context"
	"fmt"

	"github.com/astercloud/aster/pkg/vector"
)

// SemanticMemoryConfig 语义记忆配置。
// 核心运行时仅依赖接口, 不关心具体 VectorStore/Embedder 实现。
type SemanticMemoryConfig struct {
	Store          vector.VectorStore
	Embedder       vector.Embedder
	NamespaceScope string // "user" | "project" | "resource" | "global"
	TopK           int

	// EnableProvenance 是否启用记忆溯源追踪。
	EnableProvenance bool

	// ConfidenceCalculator 置信度计算器。
	// 如果为 nil, 使用默认配置。
	ConfidenceCalculator *ConfidenceCalculator

	// LineageManager 谱系管理器。
	// 如果为 nil, 不追踪记忆谱系。
	LineageManager *LineageManager

	// DefaultSourceType 默认的数据源类型。
	DefaultSourceType SourceType
}

// SemanticMemory 语义记忆组件, 用于对文本进行向量索引和检索。
// 如果 Store 或 Embedder 为空, 所有方法将成为 no-op。
type SemanticMemory struct {
	cfg SemanticMemoryConfig
}

// NewSemanticMemory 创建语义记忆组件。
func NewSemanticMemory(cfg SemanticMemoryConfig) *SemanticMemory {
	if cfg.TopK <= 0 {
		cfg.TopK = 5
	}

	// 初始化默认的置信度计算器
	if cfg.EnableProvenance && cfg.ConfidenceCalculator == nil {
		cfg.ConfidenceCalculator = NewConfidenceCalculator(DefaultConfidenceConfig())
	}

	// 初始化默认的谱系管理器
	if cfg.EnableProvenance && cfg.LineageManager == nil {
		cfg.LineageManager = NewLineageManager()
	}

	// 设置默认数据源类型
	if cfg.DefaultSourceType == "" {
		cfg.DefaultSourceType = SourceUserInput
	}

	return &SemanticMemory{cfg: cfg}
}

// Enabled 返回语义记忆是否启用。
func (sm *SemanticMemory) Enabled() bool {
	return sm != nil && sm.cfg.Store != nil && sm.cfg.Embedder != nil
}

// Close 关闭底层 VectorStore。
func (sm *SemanticMemory) Close() error {
	if sm == nil || sm.cfg.Store == nil {
		return nil
	}
	return sm.cfg.Store.Close()
}

// Index 将一段文本写入向量索引。
// docID 应全局唯一, meta 中可包含 user_id/project_id/resource_id 等信息。
// 如果启用 Provenance, 将自动创建溯源信息。
func (sm *SemanticMemory) Index(ctx context.Context, docID string, text string, meta map[string]any) error {
	if sm.cfg.EnableProvenance {
		// 自动创建 Provenance
		sourceID := docID // 使用 docID 作为默认 sourceID
		if sid, ok := meta["source_id"].(string); ok && sid != "" {
			sourceID = sid
		}

		sourceType := sm.cfg.DefaultSourceType
		if st, ok := meta["source_type"].(string); ok && st != "" {
			sourceType = SourceType(st)
		}

		isExplicit := false
		if exp, ok := meta["is_explicit"].(bool); ok {
			isExplicit = exp
		}

		var provenance *MemoryProvenance
		if isExplicit {
			provenance = NewExplicitProvenance(sourceType, sourceID)
		} else {
			provenance = NewProvenance(sourceType, sourceID)
		}

		return sm.IndexWithProvenance(ctx, docID, text, meta, provenance, nil)
	}

	// 不启用 Provenance 时的原始逻辑
	return sm.indexInternal(ctx, docID, text, meta, nil)
}

// IndexWithProvenance 使用指定的 Provenance 索引文本。
// derivedFromIDs 表示该记忆派生自哪些其他记忆。
func (sm *SemanticMemory) IndexWithProvenance(ctx context.Context, docID string, text string, meta map[string]any, provenance *MemoryProvenance, derivedFromIDs []string) error {
	if sm == nil || sm.cfg.Store == nil || sm.cfg.Embedder == nil {
		return nil
	}
	if !sm.cfg.EnableProvenance {
		return fmt.Errorf("provenance not enabled")
	}
	if provenance == nil {
		return fmt.Errorf("provenance is required")
	}

	// 追踪谱系
	if sm.cfg.LineageManager != nil {
		err := sm.cfg.LineageManager.TrackMemoryCreation(docID, provenance, derivedFromIDs)
		if err != nil {
			return fmt.Errorf("track lineage: %w", err)
		}
	}

	return sm.indexInternal(ctx, docID, text, meta, provenance)
}

// indexInternal 内部索引方法。
func (sm *SemanticMemory) indexInternal(ctx context.Context, docID string, text string, meta map[string]any, provenance *MemoryProvenance) error {
	if docID == "" || text == "" {
		return fmt.Errorf("docID and text are required")
	}

	vecs, err := sm.cfg.Embedder.EmbedText(ctx, []string{text})
	if err != nil {
		return fmt.Errorf("embed text: %w", err)
	}
	if len(vecs) == 0 {
		return fmt.Errorf("embedder returned empty vectors")
	}

	// 将文本复制到 metadata 中, 方便检索结果直接携带原文片段。
	metaCopy := make(map[string]any, len(meta)+2)
	for k, v := range meta {
		metaCopy[k] = v
	}
	metaCopy["text"] = text

	// 添加 Provenance 到 metadata
	if provenance != nil {
		provenanceMeta := provenance.ToMetadata()
		for k, v := range provenanceMeta {
			metaCopy[k] = v
		}
	}

	return sm.cfg.Store.Upsert(ctx, []vector.Document{
		{
			ID:        docID,
			Text:      text,
			Embedding: vecs[0],
			Metadata:  metaCopy,
			Namespace: sm.namespaceFromMeta(meta),
		},
	})
}

// Search 在指定命名空间内执行向量检索。
// query 为用户自然语言查询, meta 用于构造 namespace/过滤。
func (sm *SemanticMemory) Search(ctx context.Context, query string, meta map[string]any, topK int) ([]vector.Hit, error) {
	if sm == nil || sm.cfg.Store == nil || sm.cfg.Embedder == nil {
		return nil, nil
	}
	if query == "" {
		return nil, nil
	}
	if topK <= 0 {
		topK = sm.cfg.TopK
	}

	vecs, err := sm.cfg.Embedder.EmbedText(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	if len(vecs) == 0 {
		return nil, fmt.Errorf("embedder returned empty vectors")
	}

	return sm.cfg.Store.Query(ctx, vector.Query{
		Vector:    vecs[0],
		TopK:      topK,
		Namespace: sm.namespaceFromMeta(meta),
		Filter:    meta,
	})
}

func (sm *SemanticMemory) namespaceFromMeta(meta map[string]any) string {
	if meta == nil {
		return ""
	}
	scope := sm.cfg.NamespaceScope
	switch scope {
	case "user":
		if v, ok := meta["user_id"].(string); ok && v != "" {
			return "users/" + v
		}
	case "project":
		if v, ok := meta["project_id"].(string); ok && v != "" {
			return "projects/" + v
		}
	case "resource":
		user := ""
		if u, ok := meta["user_id"].(string); ok && u != "" {
			user = "users/" + u + "/"
		}
		if pid, ok := meta["project_id"].(string); ok && pid != "" {
			return user + "projects/" + pid
		}
		if rid, ok := meta["resource_id"].(string); ok && rid != "" {
			return user + "resources/" + rid
		}
	}
	// 默认命名空间
	return ""
}

// SearchWithConfidenceFilter 执行检索并按置信度过滤。
// minConfidence: 最低置信度阈值（0.0-1.0）
func (sm *SemanticMemory) SearchWithConfidenceFilter(ctx context.Context, query string, meta map[string]any, topK int, minConfidence float64) ([]vector.Hit, error) {
	if !sm.cfg.EnableProvenance {
		return sm.Search(ctx, query, meta, topK)
	}

	// 先执行标准检索
	hits, err := sm.Search(ctx, query, meta, topK*2) // 获取更多结果以便过滤
	if err != nil {
		return nil, err
	}

	// 过滤和重排序
	var filtered []vector.Hit
	for _, hit := range hits {
		provenance := FromMetadata(hit.Metadata)
		if provenance == nil {
			continue
		}

		// 更新置信度
		if sm.cfg.ConfidenceCalculator != nil {
			sm.cfg.ConfidenceCalculator.UpdateConfidence(provenance)
		}

		// 过滤低置信度
		if provenance.Confidence < minConfidence {
			continue
		}

		// 标记访问
		provenance.MarkAccessed()

		// 重新计算相关性得分（结合置信度）
		if sm.cfg.ConfidenceCalculator != nil {
			hit.Score = sm.cfg.ConfidenceCalculator.ScoreByRelevance(hit.Score, provenance)
		}

		filtered = append(filtered, hit)
	}

	// 限制返回数量
	if len(filtered) > topK {
		filtered = filtered[:topK]
	}

	return filtered, nil
}

// SearchBySourceType 按来源类型检索记忆。
func (sm *SemanticMemory) SearchBySourceType(ctx context.Context, query string, meta map[string]any, topK int, sourceTypes []SourceType) ([]vector.Hit, error) {
	if !sm.cfg.EnableProvenance {
		return sm.Search(ctx, query, meta, topK)
	}

	// 先执行标准检索
	hits, err := sm.Search(ctx, query, meta, topK*2)
	if err != nil {
		return nil, err
	}

	// 按来源类型过滤
	sourceTypeMap := make(map[SourceType]bool)
	for _, st := range sourceTypes {
		sourceTypeMap[st] = true
	}

	var filtered []vector.Hit
	for _, hit := range hits {
		provenance := FromMetadata(hit.Metadata)
		if provenance == nil {
			continue
		}

		if sourceTypeMap[provenance.SourceType] {
			filtered = append(filtered, hit)
		}
	}

	// 限制返回数量
	if len(filtered) > topK {
		filtered = filtered[:topK]
	}

	return filtered, nil
}

// PruneMemories 剪枝（删除）低置信度记忆。
// 返回被删除的记忆ID列表。
func (sm *SemanticMemory) PruneMemories(ctx context.Context, namespace string) ([]string, error) {
	if !sm.cfg.EnableProvenance || sm.cfg.ConfidenceCalculator == nil {
		return nil, fmt.Errorf("provenance or confidence calculator not enabled")
	}

	// TODO: 这需要 VectorStore 支持列举所有文档的功能
	// 当前 VectorStore 接口不支持，需要扩展
	// 临时实现：返回空列表
	return nil, nil
}

// DeleteMemoryWithLineage 删除记忆及其派生记忆。
func (sm *SemanticMemory) DeleteMemoryWithLineage(ctx context.Context, memoryID string, cascade bool) error {
	if !sm.cfg.EnableProvenance || sm.cfg.LineageManager == nil {
		// 直接删除
		return sm.cfg.Store.Delete(ctx, []string{memoryID})
	}

	// 获取需要删除的记忆列表
	deletedIDs, err := sm.cfg.LineageManager.DeleteMemoryWithLineage(ctx, memoryID, cascade)
	if err != nil {
		return fmt.Errorf("delete lineage: %w", err)
	}

	// 从向量存储中删除
	if len(deletedIDs) > 0 {
		return sm.cfg.Store.Delete(ctx, deletedIDs)
	}

	return nil
}

// GetMemoryProvenance 获取记忆的溯源信息。
// 这需要先检索记忆，然后提取 Provenance。
func (sm *SemanticMemory) GetMemoryProvenance(ctx context.Context, query string, meta map[string]any) (*MemoryProvenance, error) {
	if !sm.cfg.EnableProvenance {
		return nil, fmt.Errorf("provenance not enabled")
	}

	hits, err := sm.Search(ctx, query, meta, 1)
	if err != nil {
		return nil, err
	}

	if len(hits) == 0 {
		return nil, fmt.Errorf("no memory found")
	}

	return FromMetadata(hits[0].Metadata), nil
}

// Delete 删除单个记忆（不考虑谱系）。
func (sm *SemanticMemory) Delete(ctx context.Context, docID string) error {
	if sm == nil || sm.cfg.Store == nil {
		return nil
	}

	return sm.cfg.Store.Delete(ctx, []string{docID})
}

// UpdateMetadata 更新记忆的元数据。
func (sm *SemanticMemory) UpdateMetadata(ctx context.Context, docID string, metadata map[string]any) error {
	if sm == nil || sm.cfg.Store == nil {
		return nil
	}

	// 注意：这是一个简化实现
	// 实际场景可能需要先获取现有数据，合并元数据，然后重新索引
	// 这里假设向量存储支持元数据更新

	// TODO: 实现元数据更新逻辑
	// 目前大多数向量数据库不支持原地更新元数据
	// 需要重新索引或使用专门的更新 API

	return fmt.Errorf("UpdateMetadata not fully implemented yet")
}

// SearchAndFormat 执行向量检索并格式化为 Markdown，用于 RAG 场景。
// 返回的字符串可以直接注入到 Prompt 中作为上下文。
func (sm *SemanticMemory) SearchAndFormat(ctx context.Context, query string, meta map[string]any, topK int) (string, error) {
	if sm == nil || sm.cfg.Store == nil || sm.cfg.Embedder == nil {
		return "", nil
	}

	// 执行检索
	hits, err := sm.Search(ctx, query, meta, topK)
	if err != nil {
		return "", fmt.Errorf("search: %w", err)
	}

	if len(hits) == 0 {
		return "", nil
	}

	// 格式化为 Markdown
	var result string
	result += "## Relevant Context\n\n"
	result += fmt.Sprintf("Found %d relevant documents:\n\n", len(hits))

	for i, hit := range hits {
		// 获取文本内容
		text := ""
		if t, ok := hit.Metadata["text"].(string); ok {
			text = t
		}

		// 格式化分数（0-1 范围）
		score := hit.Score
		scorePercent := int(score * 100)

		result += fmt.Sprintf("### %d. (Relevance: %d%%)\n\n", i+1, scorePercent)

		if text != "" {
			result += text + "\n\n"
		}

		// 添加元数据（可选）
		if len(hit.Metadata) > 1 { // 除了 text 之外还有其他元数据
			result += "**Metadata:**\n"
			for k, v := range hit.Metadata {
				if k != "text" { // 跳过已显示的 text
					result += fmt.Sprintf("- %s: %v\n", k, v)
				}
			}
			result += "\n"
		}
	}

	return result, nil
}
