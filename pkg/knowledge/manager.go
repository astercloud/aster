package knowledge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strings"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/knowledge/core"
	"github.com/astercloud/aster/pkg/memory"
	"github.com/astercloud/aster/pkg/security"
	"github.com/astercloud/aster/pkg/vector"
)

// manager 统一知识管理器实现
type manager struct {
	config *ManagerConfig

	// 存储组件
	memoryManager *memory.Manager
	vectorStore   vector.VectorStore
	embedder      vector.Embedder

	// 运行时状态
	mu      sync.RWMutex
	cache   map[string]*cacheEntry
	stats   map[string]*KnowledgeStats
	running bool

	// 审计和安全（策略可选）
	auditStrategy AuditStrategy
	piiStrategy   PIIStrategy

	// 轻量核心管线（可选）
	corePipeline *core.Pipeline
}

// cacheEntry 缓存条目
type cacheEntry struct {
	item   *KnowledgeItem
	expire time.Time
}

// AuditRecord 审计记录
type AuditRecord struct {
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	UserID    string    `json:"user_id"`
	ItemID    string    `json:"item_id"`
	Details   string    `json:"details"`
}

// defaultAuditor 默认审计策略：内存记录、可选启用
type defaultAuditor struct {
	enabled bool
	limit   int
	records []AuditRecord
}

func (a *defaultAuditor) Record(action, userID, itemID, details string) {
	if !a.enabled {
		return
	}
	a.records = append(a.records, AuditRecord{
		Timestamp: time.Now(),
		Action:    action,
		UserID:    userID,
		ItemID:    itemID,
		Details:   details,
	})
	if a.limit > 0 && len(a.records) > a.limit {
		// 保留最新记录
		a.records = a.records[len(a.records)-a.limit:]
	}
}

// redactionStrategy 默认 PII 脱敏策略
type redactionStrategy struct {
	redactor security.ContentRedactor
}

func (r *redactionStrategy) Sanitize(item *KnowledgeItem) *KnowledgeItem {
	if r == nil || r.redactor == nil || item == nil {
		return item
	}
	s := *item
	s.Content = r.redactor.Redact(item.Content)
	s.Description = r.redactor.Redact(item.Description)
	s.Title = r.redactor.Redact(item.Title)
	return &s
}

// NewManager 创建知识管理器
func NewManager(config *ManagerConfig) (Manager, error) {
	if config == nil {
		return nil, errors.New("knowledge: config cannot be nil")
	}

	if config.MemoryManager == nil {
		return nil, errors.New("knowledge: memory manager is required")
	}

	if config.VectorStore == nil {
		return nil, errors.New("knowledge: vector store is required")
	}

	if config.Embedder == nil && config.AutoEmbed {
		return nil, errors.New("knowledge: embedder is required when auto embed is enabled")
	}

	m := &manager{
		config:        config,
		memoryManager: config.MemoryManager,
		vectorStore:   config.VectorStore,
		embedder:      config.Embedder,
		cache:         make(map[string]*cacheEntry),
		stats:         make(map[string]*KnowledgeStats),
	}

	// PII 策略注入
	if config.PIIStrategy != nil {
		m.piiStrategy = config.PIIStrategy
	} else if config.EnablePII {
		detector := security.NewRegexPIIDetector()
		m.piiStrategy = &redactionStrategy{
			redactor: security.NewPIIRedactor(detector),
		}
	}

	// 审计策略注入
	if config.AuditStrategy != nil {
		m.auditStrategy = config.AuditStrategy
	} else if config.EnableAudit {
		m.auditStrategy = &defaultAuditor{
			enabled: true,
			limit:   10000,
			records: make([]AuditRecord, 0),
		}
	}

	// 可选：构建轻量核心管线
	if config.UseCorePipeline && config.VectorStore != nil && config.Embedder != nil {
		p, err := core.NewPipeline(core.PipelineConfig{
			Store:       config.VectorStore,
			Embedder:    config.Embedder,
			Namespace:   config.Namespace,
			DefaultTopK: config.MaxResults,
		})
		if err != nil {
			return nil, fmt.Errorf("knowledge: init core pipeline: %w", err)
		}
		m.corePipeline = p
	}

	return m, nil
}

// Start 启动知识管理器
func (m *manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return errors.New("knowledge: manager is already running")
	}

	m.running = true

	// 启动清理协程
	go m.cleanupRoutine(ctx)

	return nil
}

// Stop 停止知识管理器
func (m *manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return errors.New("knowledge: manager is not running")
	}

	m.running = false

	// 关闭向量存储
	if m.vectorStore != nil {
		if err := m.vectorStore.Close(); err != nil {
			return fmt.Errorf("knowledge: failed to close vector store: %w", err)
		}
	}

	return nil
}

// Add 添加知识项
func (m *manager) Add(ctx context.Context, item *KnowledgeItem) error {
	if item == nil {
		return errors.New("knowledge: item cannot be nil")
	}

	if item.ID == "" {
		return errors.New("knowledge: item ID cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否已存在
	if existing, err := m.getItemFromMemory(ctx, item.ID); err == nil && existing != nil {
		return fmt.Errorf("knowledge: item with ID %s already exists", item.ID)
	}

	// 设置时间戳
	now := time.Now()
	item.CreatedAt = now
	item.UpdatedAt = now

	// 设置命名空间
	if item.Namespace == "" {
		item.Namespace = m.config.Namespace
	}

	// PII检测
	if m.piiStrategy != nil {
		if sanitized := m.sanitizeContent(item); sanitized != item {
			item = sanitized
		}
	}

	// 自动生成向量
	if m.config.AutoEmbed && len(item.Embedding) == 0 {
		embedding, err := m.generateEmbedding(ctx, item)
		if err != nil {
			return fmt.Errorf("knowledge: failed to generate embedding: %w", err)
		}
		item.Embedding = embedding
	}

	// 质量检查
	if item.Quality == 0 {
		item.Quality = m.calculateQuality(item)
	}

	// 保存到内存
	if err := m.saveItemToMemory(ctx, item); err != nil {
		return fmt.Errorf("knowledge: failed to save item to memory: %w", err)
	}

	// 保存向量到向量存储
	if len(item.Embedding) > 0 {
		doc := vector.Document{
			ID:        item.ID,
			Text:      item.Content,
			Embedding: item.Embedding,
			Metadata: map[string]any{
				"type":     string(item.Type),
				"category": item.Category,
				"tags":     strings.Join(item.Tags, ","),
				"source":   item.Source,
				"author":   item.Author,
			},
			Namespace: item.Namespace,
		}

		if err := m.vectorStore.Upsert(ctx, []vector.Document{doc}); err != nil {
			// 回滚内存中的保存
			_ = m.removeItemFromMemory(ctx, item.ID)
			return fmt.Errorf("knowledge: failed to save embedding: %w", err)
		}
	}

	// 更新缓存
	if m.config.CacheEnabled {
		m.updateCache(item)
	}

	// 更新统计
	m.updateStats(item.Namespace)

	// 记录审计
	if m.config.EnableAudit {
		m.audit("add", "", item.ID, "Added knowledge item: "+item.Title)
	}

	return nil
}

// Get 获取知识项
func (m *manager) Get(ctx context.Context, id string) (*KnowledgeItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 先检查缓存
	if m.config.CacheEnabled {
		if entry, exists := m.cache[id]; exists {
			if time.Now().Before(entry.expire) {
				return entry.item, nil
			}
			delete(m.cache, id)
		}
	}

	// 从内存获取
	item, err := m.getItemFromMemory(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("knowledge: failed to get item: %w", err)
	}

	// 更新缓存
	if m.config.CacheEnabled {
		m.updateCache(item)
	}

	// 记录审计
	if m.config.EnableAudit {
		m.audit("get", "", item.ID, "Retrieved knowledge item: "+item.Title)
	}

	return item, nil
}

// Update 更新知识项
func (m *manager) Update(ctx context.Context, item *KnowledgeItem) error {
	if item == nil {
		return errors.New("knowledge: item cannot be nil")
	}

	if item.ID == "" {
		return errors.New("knowledge: item ID cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否存在
	existing, err := m.getItemFromMemory(ctx, item.ID)
	if err != nil {
		return fmt.Errorf("knowledge: item not found: %w", err)
	}

	// 更新时间戳
	item.UpdatedAt = time.Now()
	item.CreatedAt = existing.CreatedAt // 保持创建时间

	// PII检测
	if m.piiStrategy != nil {
		if sanitized := m.sanitizeContent(item); sanitized != item {
			item = sanitized
		}
	}

	// 重新生成向量（如果内容变化）
	if m.config.AutoEmbed && (item.Content != existing.Content || len(item.Embedding) == 0) {
		embedding, err := m.generateEmbedding(ctx, item)
		if err != nil {
			return fmt.Errorf("knowledge: failed to generate embedding: %w", err)
		}
		item.Embedding = embedding
	}

	// 重新计算质量
	item.Quality = m.calculateQuality(item)

	// 保存到内存
	if err := m.saveItemToMemory(ctx, item); err != nil {
		return fmt.Errorf("knowledge: failed to save updated item to memory: %w", err)
	}

	// 更新向量存储
	if len(item.Embedding) > 0 {
		doc := vector.Document{
			ID:        item.ID,
			Text:      item.Content,
			Embedding: item.Embedding,
			Metadata: map[string]any{
				"type":     string(item.Type),
				"category": item.Category,
				"tags":     strings.Join(item.Tags, ","),
				"source":   item.Source,
				"author":   item.Author,
			},
			Namespace: item.Namespace,
		}

		if err := m.vectorStore.Upsert(ctx, []vector.Document{doc}); err != nil {
			return fmt.Errorf("knowledge: failed to update embedding: %w", err)
		}
	}

	// 更新缓存
	if m.config.CacheEnabled {
		m.updateCache(item)
	}

	// 更新统计
	m.updateStats(item.Namespace)

	// 记录审计
	if m.config.EnableAudit {
		m.audit("update", "", item.ID, "Updated knowledge item: "+item.Title)
	}

	return nil
}

// Delete 删除知识项
func (m *manager) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 获取项目信息用于审计
	item, err := m.getItemFromMemory(ctx, id)
	if err != nil {
		return fmt.Errorf("knowledge: item not found: %w", err)
	}

	// 从内存删除
	if err := m.removeItemFromMemory(ctx, id); err != nil {
		return fmt.Errorf("knowledge: failed to remove item from memory: %w", err)
	}

	// 从向量存储删除
	if err := m.vectorStore.Delete(ctx, []string{id}); err != nil {
		// 记录警告但不失败
		fmt.Printf("knowledge: warning - failed to remove embedding for item %s: %v\n", id, err)
	}

	// 清理缓存
	if m.config.CacheEnabled {
		delete(m.cache, id)
	}

	// 更新统计
	m.updateStats(item.Namespace)

	// 记录审计
	if m.config.EnableAudit {
		m.audit("delete", "", id, "Deleted knowledge item: "+item.Title)
	}

	return nil
}

// Search 搜索知识
func (m *manager) Search(ctx context.Context, query *SearchQuery) ([]*SearchResult, error) {
	if query == nil {
		return nil, errors.New("knowledge: search query cannot be nil")
	}

	// 轻量路径：仅使用向量检索，避开复杂策略
	if m.corePipeline != nil && (query.Strategy == "" || query.Strategy == StrategyVector) {
		meta := map[string]any{}
		maps.Copy(meta, query.Filters)
		if query.Namespace != "" {
			meta["namespace"] = query.Namespace
		}
		hits, err := m.corePipeline.Search(ctx, query.Query, query.MaxResults, meta)
		if err != nil {
			return nil, fmt.Errorf("knowledge: core search failed: %w", err)
		}
		return convertCoreHits(hits), nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// 设置默认值
	if query.MaxResults == 0 {
		query.MaxResults = m.config.MaxResults
	}
	if query.Namespace == "" {
		query.Namespace = m.config.Namespace
	}

	var results []*SearchResult
	var err error

	// 根据策略执行搜索
	switch query.Strategy {
	case StrategyText:
		results, err = m.searchText(ctx, query)
	case StrategyVector:
		results, err = m.searchVector(ctx, query)
	case StrategyHybrid:
		results, err = m.searchHybrid(ctx, query)
	case StrategyGraph:
		results, err = m.searchGraph(ctx, query)
	default:
		// 默认使用混合搜索
		query.Strategy = StrategyHybrid
		results, err = m.searchHybrid(ctx, query)
	}

	if err != nil {
		return nil, fmt.Errorf("knowledge: search failed: %w", err)
	}

	// 应用过滤器
	results = m.applyFilters(results, query)

	// 限制结果数量
	if len(results) > query.MaxResults {
		results = results[:query.MaxResults]
	}

	// 记录审计
	if m.config.EnableAudit {
		m.audit("search", "", "", fmt.Sprintf("Search query: %s, results: %d", query.Query, len(results)))
	}

	return results, nil
}

// SearchSimilar 搜索相似知识
func (m *manager) SearchSimilar(ctx context.Context, id string, maxResults int) ([]*SearchResult, error) {
	item, err := m.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("knowledge: failed to get item for similarity search: %w", err)
	}

	if len(item.Embedding) == 0 && m.corePipeline == nil {
		return nil, errors.New("knowledge: item has no embedding for similarity search")
	}

	// 优先使用原有向量
	if len(item.Embedding) > 0 {
		query := &SearchQuery{
			Vector:     item.Embedding,
			Strategy:   StrategyVector,
			Namespace:  item.Namespace,
			MaxResults: maxResults,
		}
		return m.Search(ctx, query)
	}

	// 回退到核心管线的向量检索（使用文本重新嵌入）
	if m.corePipeline != nil {
		hits, err := m.corePipeline.Search(ctx, item.Content, maxResults, map[string]any{
			"namespace": item.Namespace,
		})
		if err != nil {
			return nil, fmt.Errorf("knowledge: core similarity search failed: %w", err)
		}
		return convertCoreHits(hits), nil
	}

	return nil, errors.New("knowledge: no embedding available for similarity search")
}

// AddRelation 添加知识关系
func (m *manager) AddRelation(ctx context.Context, fromID, toID string, relationType RelationType, weight float64, label string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 获取知识项
	fromItem, err := m.getItemFromMemory(ctx, fromID)
	if err != nil {
		return fmt.Errorf("knowledge: from item not found: %w", err)
	}

	// 验证目标项目存在
	_, err = m.getItemFromMemory(ctx, toID)
	if err != nil {
		return fmt.Errorf("knowledge: to item not found: %w", err)
	}

	// 添加关系
	relation := KnowledgeRelation{
		Type:     relationType,
		TargetID: toID,
		Weight:   weight,
		Label:    label,
	}

	// 检查关系是否已存在
	for _, existing := range fromItem.Relations {
		if existing.Type == relationType && existing.TargetID == toID {
			return errors.New("knowledge: relation already exists")
		}
	}

	fromItem.Relations = append(fromItem.Relations, relation)

	// 保存更新
	if err := m.saveItemToMemory(ctx, fromItem); err != nil {
		return fmt.Errorf("knowledge: failed to save updated relations: %w", err)
	}

	// 记录审计
	if m.config.EnableAudit {
		m.audit("add_relation", "", fromID, fmt.Sprintf("Added relation to %s: %s", toID, relationType))
	}

	return nil
}

// RemoveRelation 移除知识关系
func (m *manager) RemoveRelation(ctx context.Context, fromID, toID string, relationType RelationType) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 获取知识项
	item, err := m.getItemFromMemory(ctx, fromID)
	if err != nil {
		return fmt.Errorf("knowledge: item not found: %w", err)
	}

	// 移除关系
	relations := make([]KnowledgeRelation, 0, len(item.Relations))
	removed := false
	for _, existing := range item.Relations {
		if existing.Type == relationType && existing.TargetID == toID {
			removed = true
		} else {
			relations = append(relations, existing)
		}
	}

	if !removed {
		return errors.New("knowledge: relation not found")
	}

	item.Relations = relations

	// 保存更新
	if err := m.saveItemToMemory(ctx, item); err != nil {
		return fmt.Errorf("knowledge: failed to save updated relations: %w", err)
	}

	// 记录审计
	if m.config.EnableAudit {
		m.audit("remove_relation", "", fromID, fmt.Sprintf("Removed relation to %s: %s", toID, relationType))
	}

	return nil
}

// GetRelated 获取相关知识
func (m *manager) GetRelated(ctx context.Context, id string, relationTypes []RelationType, maxDepth int) ([]*KnowledgeItem, error) {
	if maxDepth <= 0 {
		maxDepth = 3
	}

	visited := make(map[string]bool)
	var results []*KnowledgeItem

	err := m.traverseRelations(ctx, id, relationTypes, 0, maxDepth, visited, &results)
	if err != nil {
		return nil, fmt.Errorf("knowledge: failed to traverse relations: %w", err)
	}

	return results, nil
}

// BulkAdd 批量添加知识
func (m *manager) BulkAdd(ctx context.Context, items []*KnowledgeItem) error {
	if len(items) == 0 {
		return nil
	}

	for _, item := range items {
		if err := m.Add(ctx, item); err != nil {
			return fmt.Errorf("knowledge: bulk add failed for item %s: %w", item.ID, err)
		}
	}

	return nil
}

// BulkSearch 批量搜索
func (m *manager) BulkSearch(ctx context.Context, queries []*SearchQuery) ([][]*SearchResult, error) {
	results := make([][]*SearchResult, len(queries))

	for i, query := range queries {
		result, err := m.Search(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("knowledge: bulk search failed for query %d: %w", i, err)
		}
		results[i] = result
	}

	return results, nil
}

// GetStats 获取统计信息
func (m *manager) GetStats(ctx context.Context, namespace string) (*KnowledgeStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if stats, exists := m.stats[namespace]; exists {
		return stats, nil
	}

	// 重新计算统计信息
	stats, err := m.calculateStats(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("knowledge: failed to calculate stats: %w", err)
	}

	m.stats[namespace] = stats
	return stats, nil
}

// Validate 验证知识项
func (m *manager) Validate(ctx context.Context, id string) (bool, []string, error) {
	item, err := m.Get(ctx, id)
	if err != nil {
		return false, nil, fmt.Errorf("knowledge: failed to get item for validation: %w", err)
	}

	var warnings []string

	// 检查置信度
	if item.Confidence < m.config.MinConfidence {
		warnings = append(warnings, fmt.Sprintf("confidence %.2f is below threshold %.2f", item.Confidence, m.config.MinConfidence))
	}

	// 检查质量
	if item.Quality < m.config.MinQuality {
		warnings = append(warnings, fmt.Sprintf("quality %.2f is below threshold %.2f", item.Quality, m.config.MinQuality))
	}

	// 检查时效性
	if item.ExpiresAt != nil && time.Now().After(*item.ExpiresAt) {
		warnings = append(warnings, "knowledge item has expired")
	}

	isValid := len(warnings) == 0
	return isValid, warnings, nil
}

// Refresh 刷新知识项
func (m *manager) Refresh(ctx context.Context, id string) error {
	item, err := m.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("knowledge: failed to get item for refresh: %w", err)
	}

	// 重新生成向量
	if m.config.AutoEmbed {
		embedding, err := m.generateEmbedding(ctx, item)
		if err != nil {
			return fmt.Errorf("knowledge: failed to regenerate embedding: %w", err)
		}
		item.Embedding = embedding
	}

	// 重新计算质量
	item.Quality = m.calculateQuality(item)

	// 保存更新
	return m.Update(ctx, item)
}

// Compress 压缩知识库
func (m *manager) Compress(ctx context.Context, namespace string) error {
	// TODO: 实现知识压缩逻辑
	// 1. 识别重复或相似的知识
	// 2. 合并相似知识
	// 3. 删除低质量知识
	// 4. 更新关系
	return errors.New("knowledge: compression not yet implemented")
}

// Reason 知识推理
func (m *manager) Reason(ctx context.Context, query string, maxSteps int) ([]*KnowledgeItem, []string, error) {
	// TODO: 实现知识推理逻辑
	// 1. 理解查询意图
	// 2. 搜索相关知识
	// 3. 应用推理规则
	// 4. 生成推理链
	return nil, nil, errors.New("knowledge: reasoning not yet implemented")
}

// 辅助方法

// generateEmbedding 生成向量嵌入
func (m *manager) generateEmbedding(ctx context.Context, item *KnowledgeItem) ([]float32, error) {
	if m.embedder == nil {
		return nil, errors.New("knowledge: no embedder available")
	}

	text := item.Content
	if text == "" {
		text = item.Title + " " + item.Description
	}

	embeddings, err := m.embedder.EmbedText(ctx, []string{text})
	if err != nil {
		return nil, fmt.Errorf("knowledge: embed failed: %w", err)
	}
	if len(embeddings) == 0 {
		return nil, errors.New("knowledge: no embedding generated")
	}
	embedding := embeddings[0]

	return embedding, nil
}

// calculateQuality 计算知识质量
func (m *manager) calculateQuality(item *KnowledgeItem) float64 {
	score := 0.0

	// 内容长度评分
	if len(item.Content) > 100 {
		score += 0.2
	}

	// 描述评分
	if item.Description != "" {
		score += 0.1
	}

	// 标签评分
	if len(item.Tags) > 0 {
		score += 0.1
	}

	// 来源评分
	if item.Source != "" {
		score += 0.2
	}

	// 作者评分
	if item.Author != "" {
		score += 0.1
	}

	// 置信度评分
	score += item.Confidence * 0.3

	return score
}

// sanitizeContent 内容净化（PII检测）
func (m *manager) sanitizeContent(item *KnowledgeItem) *KnowledgeItem {
	if m.piiStrategy == nil {
		return item
	}
	return m.piiStrategy.Sanitize(item)
}

// updateCache 更新缓存
func (m *manager) updateCache(item *KnowledgeItem) {
	if !m.config.CacheEnabled {
		return
	}

	expire := time.Now().Add(m.config.CacheTTL)
	m.cache[item.ID] = &cacheEntry{
		item:   item,
		expire: expire,
	}
}

// audit 记录审计日志
func (m *manager) audit(action, userID, itemID, details string) {
	if m.auditStrategy == nil {
		return
	}
	m.auditStrategy.Record(action, userID, itemID, details)
}

// cleanupRoutine 清理协程
func (m *manager) cleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.mu.Lock()
			m.cleanupCache()
			m.mu.Unlock()
		}
	}
}

// cleanupCache 清理过期缓存
func (m *manager) cleanupCache() {
	if !m.config.CacheEnabled {
		return
	}

	now := time.Now()
	for id, entry := range m.cache {
		if now.After(entry.expire) {
			delete(m.cache, id)
		}
	}
}

// search methods are placeholder implementations
// 实际的搜索方法将在后续实现
func (m *manager) searchText(_ context.Context, _ *SearchQuery) ([]*SearchResult, error) {
	// TODO: 实现文本搜索
	return nil, nil
}

func (m *manager) searchVector(_ context.Context, _ *SearchQuery) ([]*SearchResult, error) {
	// TODO: 实现向量搜索
	return nil, nil
}

func (m *manager) searchHybrid(_ context.Context, _ *SearchQuery) ([]*SearchResult, error) {
	// TODO: 实现混合搜索
	return nil, nil
}

func (m *manager) searchGraph(_ context.Context, _ *SearchQuery) ([]*SearchResult, error) {
	// TODO: 实现图搜索
	return nil, nil
}

func (m *manager) applyFilters(results []*SearchResult, _ *SearchQuery) []*SearchResult {
	// TODO: 实现过滤逻辑
	return results
}

func (m *manager) traverseRelations(_ context.Context, _ string, _ []RelationType, _, _ int, _ map[string]bool, _ *[]*KnowledgeItem) error {
	// TODO: 实现关系遍历
	return nil
}

// memory operations
func (m *manager) getItemFromMemory(ctx context.Context, id string) (*KnowledgeItem, error) {
	// 从文件读取知识项
	data, err := m.memoryManager.ReadFile(ctx, fmt.Sprintf("knowledge/%s.json", id))
	if err != nil {
		return nil, err
	}

	var item KnowledgeItem
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return nil, err
	}

	return &item, nil
}

func (m *manager) saveItemToMemory(ctx context.Context, item *KnowledgeItem) error {
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}

	_, err = m.memoryManager.OverwriteWithNote(ctx, fmt.Sprintf("knowledge/%s.json", item.ID), item.Title, string(data))
	return err
}

func (m *manager) removeItemFromMemory(_ context.Context, _ string) error {
	// TODO: 实现从内存删除
	return nil
}

func (m *manager) calculateStats(_ context.Context, _ string) (*KnowledgeStats, error) {
	// TODO: 实现统计计算
	return &KnowledgeStats{
		TotalItems:      0,
		ItemsByType:     make(map[string]int64),
		ItemsByCategory: make(map[string]int64),
		AverageQuality:  0.0,
		LastUpdated:     time.Now(),
	}, nil
}

func (m *manager) updateStats(namespace string) {
	// TODO: 实现统计更新
}
