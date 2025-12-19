package knowledge

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// RAGConfig RAG检索增强配置
type RAGConfig struct {
	// 检索配置
	MaxRetrievalResults int     `json:"max_retrieval_results"`
	Reranking           bool    `json:"reranking"`
	DiversityThreshold  float64 `json:"diversity_threshold"`
	RelevanceThreshold  float64 `json:"relevance_threshold"`

	// 混合搜索权重
	TextSearchWeight   float64 `json:"text_search_weight"`
	VectorSearchWeight float64 `json:"vector_search_weight"`
	GraphSearchWeight  float64 `json:"graph_search_weight"`

	// 重排序配置
	RerankerType       RerankerType `json:"reranker_type"`
	CrossEncoderWeight float64      `json:"cross_encoder_weight"`
	BM25Weight         float64      `json:"bm25_weight"`
	EmbeddingWeight    float64      `json:"embedding_weight"`

	// 上下文配置
	MaxContextLength   int             `json:"max_context_length"`
	ContextCompression bool            `json:"context_compression"`
	ContextStrategy    ContextStrategy `json:"context_strategy"`

	// 高级检索
	EnableMultiHop       bool `json:"enable_multi_hop"`
	MaxHops              int  `json:"max_hops"`
	EnableTemporalFilter bool `json:"enable_temporal_filter"`
	EnableQualityFilter  bool `json:"enable_quality_filter"`
}

// RerankerType 重排序器类型
type RerankerType string

const (
	RerankerTypeNone         RerankerType = "none"
	RerankerTypeCrossEncoder RerankerType = "cross_encoder"
	RerankerTypeBM25         RerankerType = "bm25"
	RerankerTypeHybrid       RerankerType = "hybrid"
)

// ContextStrategy 上下文策略
type ContextStrategy string

const (
	ContextStrategyAll     ContextStrategy = "all"     // 包含所有检索结果
	ContextStrategyTopK    ContextStrategy = "topk"    // 只包含top-k结果
	ContextStrategyDiverse ContextStrategy = "diverse" // 包含多样化的结果
	ContextStrategyChain   ContextStrategy = "chain"   // 包含推理链
)

// RAGResult RAG检索结果
type RAGResult struct {
	Query          string           `json:"query"`
	Results        []*RetrievalItem `json:"results"`
	EnhancedQuery  string           `json:"enhanced_query"`
	Context        string           `json:"context"`
	ContextItems   []*KnowledgeItem `json:"context_items"`
	Confidence     float64          `json:"confidence"`
	RetrievalTime  time.Duration    `json:"retrieval_time"`
	ProcessingTime time.Duration    `json:"processing_time"`
	TotalTime      time.Duration    `json:"total_time"`
}

// RetrievalItem 检索项
type RetrievalItem struct {
	Item        *KnowledgeItem `json:"item"`
	Score       float64        `json:"score"`
	Source      string         `json:"source"`      // 检索来源：text, vector, graph
	Explanation string         `json:"explanation"` // 检索解释
	Rank        int            `json:"rank"`        // 排名
}

// RAG RAG检索增强器
type RAG struct {
	manager Manager
	config  *RAGConfig
}

// NewRAG 创建RAG检索增强器
func NewRAG(manager Manager, config *RAGConfig) *RAG {
	if config == nil {
		config = &RAGConfig{
			MaxRetrievalResults: 10,
			Reranking:           true,
			RelevanceThreshold:  0.5,
			TextSearchWeight:    0.3,
			VectorSearchWeight:  0.7,
			RerankerType:        RerankerTypeHybrid,
			ContextStrategy:     ContextStrategyTopK,
			EnableMultiHop:      true,
			MaxHops:             3,
		}
	}

	return &RAG{
		manager: manager,
		config:  config,
	}
}

// Retrieve 检索相关知识
func (r *RAG) Retrieve(ctx context.Context, query string, options ...RAGOption) (*RAGResult, error) {
	startTime := time.Now()

	// 应用选项
	opts := &RAGOptions{}
	for _, opt := range options {
		opt(opts)
	}

	// 执行多源检索
	items, err := r.multiSourceRetrieval(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("knowledge: multi-source retrieval failed: %w", err)
	}

	retrievalTime := time.Since(startTime)
	processingStart := time.Now()

	// 重排序
	if r.config.Reranking {
		items = r.rerank(ctx, query, items)
	}

	// 应用过滤
	items = r.applyFilters(items, opts)

	// 生成上下文
	context, contextItems, err := r.generateContext(ctx, items, query)
	if err != nil {
		return nil, fmt.Errorf("knowledge: failed to generate context: %w", err)
	}

	// 查询增强
	enhancedQuery := r.enhanceQuery(query, items)

	processingTime := time.Since(processingStart)

	return &RAGResult{
		Query:          query,
		Results:        items,
		EnhancedQuery:  enhancedQuery,
		Context:        context,
		ContextItems:   contextItems,
		Confidence:     r.calculateConfidence(items),
		RetrievalTime:  retrievalTime,
		ProcessingTime: processingTime,
		TotalTime:      time.Since(startTime),
	}, nil
}

// RetrieveWithMetadata 带元数据的检索
func (r *RAG) RetrieveWithMetadata(ctx context.Context, query string, metadata map[string]any) (*RAGResult, error) {
	return r.Retrieve(ctx, query, WithMetadata(metadata))
}

// RetrieveSimilar 检索相似内容
func (r *RAG) RetrieveSimilar(ctx context.Context, itemID string, maxResults int) (*RAGResult, error) {
	// 获取原始项目
	item, err := r.manager.Get(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("knowledge: failed to get item: %w", err)
	}

	query := item.Content
	if query == "" {
		query = item.Title + " " + item.Description
	}

	// 检索时包含原始项目ID作为参考
	return r.Retrieve(ctx, query, WithMaxResults(maxResults), WithMetadata(map[string]any{
		"reference_id": itemID,
	}))
}

// RetrieveMultiHop 多跳检索
func (r *RAG) RetrieveMultiHop(ctx context.Context, query string, maxHops int) (*RAGResult, error) {
	if !r.config.EnableMultiHop {
		return r.Retrieve(ctx, query)
	}

	if maxHops <= 0 {
		maxHops = r.config.MaxHops
	}

	var allItems []*RetrievalItem
	var allContextItems []*KnowledgeItem
	var expandedQuery = query

	for range maxHops {
		result, err := r.Retrieve(ctx, expandedQuery, WithMaxResults(r.config.MaxRetrievalResults/2))
		if err != nil {
			break
		}

		if len(result.Results) == 0 {
			break
		}

		// 添加到总结果
		allItems = append(allItems, result.Results...)
		allContextItems = append(allContextItems, result.ContextItems...)

		// 使用top结果扩展查询
		if len(result.Results) > 0 {
			expandedQuery = r.expandQuery(expandedQuery, result.Results[0].Item)
		}
	}

	// 去重和重新排序
	allItems = r.deduplicate(allItems)
	allItems = r.rerank(ctx, query, allItems)
	if len(allItems) > r.config.MaxRetrievalResults {
		allItems = allItems[:r.config.MaxRetrievalResults]
	}

	context, _, _ := r.generateContext(ctx, allItems, query)

	return &RAGResult{
		Query:         query,
		Results:       allItems,
		EnhancedQuery: expandedQuery,
		Context:       context,
		ContextItems:  allContextItems,
		Confidence:    r.calculateConfidence(allItems),
		TotalTime:     time.Since(time.Now()),
	}, nil
}

// RAGOption RAG选项
type RAGOption func(*RAGOptions)

// RAGOptions RAG选项
type RAGOptions struct {
	MaxResults     int            `json:"max_results"`
	KnowledgeType  KnowledgeType  `json:"knowledge_type"`
	Category       string         `json:"category"`
	Tags           []string       `json:"tags"`
	Namespace      string         `json:"namespace"`
	Metadata       map[string]any `json:"metadata"`
	TimeRange      *TimeRange     `json:"time_range"`
	QualityFilter  *QualityFilter `json:"quality_filter"`
	DiversityBoost bool           `json:"diversity_boost"`
}

// TimeRange 时间范围
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// QualityFilter 质量过滤器
type QualityFilter struct {
	MinConfidence float64 `json:"min_confidence"`
	MinQuality    float64 `json:"min_quality"`
}

// 选项构造函数
func WithMaxResults(maxResults int) RAGOption {
	return func(opts *RAGOptions) {
		opts.MaxResults = maxResults
	}
}

func WithKnowledgeType(knowledgeType KnowledgeType) RAGOption {
	return func(opts *RAGOptions) {
		opts.KnowledgeType = knowledgeType
	}
}

func WithCategory(category string) RAGOption {
	return func(opts *RAGOptions) {
		opts.Category = category
	}
}

func WithTags(tags ...string) RAGOption {
	return func(opts *RAGOptions) {
		opts.Tags = tags
	}
}

func WithNamespace(namespace string) RAGOption {
	return func(opts *RAGOptions) {
		opts.Namespace = namespace
	}
}

func WithMetadata(metadata map[string]any) RAGOption {
	return func(opts *RAGOptions) {
		opts.Metadata = metadata
	}
}

func WithTimeRange(start, end time.Time) RAGOption {
	return func(opts *RAGOptions) {
		opts.TimeRange = &TimeRange{Start: start, End: end}
	}
}

func WithQualityFilter(minConfidence, minQuality float64) RAGOption {
	return func(opts *RAGOptions) {
		opts.QualityFilter = &QualityFilter{
			MinConfidence: minConfidence,
			MinQuality:    minQuality,
		}
	}
}

func WithDiversityBoost() RAGOption {
	return func(opts *RAGOptions) {
		opts.DiversityBoost = true
	}
}

// 私有方法

// multiSourceRetrieval 多源检索
func (r *RAG) multiSourceRetrieval(ctx context.Context, query string, opts *RAGOptions) ([]*RetrievalItem, error) {
	var allItems []*RetrievalItem

	// 文本搜索
	if r.config.TextSearchWeight > 0 {
		textItems, err := r.textSearch(ctx, query, opts)
		if err == nil {
			allItems = append(allItems, textItems...)
		}
	}

	// 向量搜索
	if r.config.VectorSearchWeight > 0 {
		vectorItems, err := r.vectorSearch(ctx, query, opts)
		if err == nil {
			allItems = append(allItems, vectorItems...)
		}
	}

	// 图搜索
	if r.config.GraphSearchWeight > 0 {
		graphItems, err := r.graphSearch(ctx, query, opts)
		if err == nil {
			allItems = append(allItems, graphItems...)
		}
	}

	return allItems, nil
}

// textSearch 文本搜索
func (r *RAG) textSearch(ctx context.Context, query string, opts *RAGOptions) ([]*RetrievalItem, error) {
	searchQuery := &SearchQuery{
		Query:      query,
		Strategy:   StrategyText,
		MaxResults: opts.MaxResults,
		Namespace:  opts.Namespace,
	}

	// 应用选项过滤
	if opts.KnowledgeType != "" {
		searchQuery.Type = opts.KnowledgeType
	}
	if opts.Category != "" {
		searchQuery.Category = opts.Category
	}
	if len(opts.Tags) > 0 {
		searchQuery.Tags = opts.Tags
	}
	if opts.TimeRange != nil {
		searchQuery.After = &opts.TimeRange.Start
		searchQuery.Before = &opts.TimeRange.End
	}
	if opts.QualityFilter != nil {
		searchQuery.MinConfidence = opts.QualityFilter.MinConfidence
		searchQuery.MinQuality = opts.QualityFilter.MinQuality
	}

	results, err := r.manager.Search(ctx, searchQuery)
	if err != nil {
		return nil, err
	}

	items := make([]*RetrievalItem, len(results))
	for i, result := range results {
		items[i] = &RetrievalItem{
			Item:        &result.Item,
			Score:       result.Score * r.config.TextSearchWeight,
			Source:      "text",
			Explanation: "Text-based search match",
			Rank:        i + 1,
		}
	}

	return items, nil
}

// vectorSearch 向量搜索
func (r *RAG) vectorSearch(_ context.Context, _ string, _ *RAGOptions) ([]*RetrievalItem, error) {
	// TODO: 需要获取向量化器
	// 这里先返回空结果
	return []*RetrievalItem{}, nil
}

// graphSearch 图搜索
func (r *RAG) graphSearch(_ context.Context, _ string, _ *RAGOptions) ([]*RetrievalItem, error) {
	// TODO: 实现图搜索逻辑
	return []*RetrievalItem{}, nil
}

// rerank 重排序
func (r *RAG) rerank(ctx context.Context, query string, items []*RetrievalItem) []*RetrievalItem {
	switch r.config.RerankerType {
	case RerankerTypeCrossEncoder:
		return r.rerankWithCrossEncoder(ctx, query, items)
	case RerankerTypeBM25:
		return r.rerankWithBM25(ctx, query, items)
	case RerankerTypeHybrid:
		return r.rerankHybrid(ctx, query, items)
	default:
		return items
	}
}

// rerankWithCrossEncoder 使用交叉编码器重排序
func (r *RAG) rerankWithCrossEncoder(_ context.Context, _ string, items []*RetrievalItem) []*RetrievalItem {
	// TODO: 实现交叉编码器重排序
	// 现在只是按原始分数排序
	sort.Slice(items, func(i, j int) bool {
		return items[i].Score > items[j].Score
	})
	return items
}

// rerankWithBM25 使用BM25重排序
func (r *RAG) rerankWithBM25(_ context.Context, _ string, items []*RetrievalItem) []*RetrievalItem {
	// TODO: 实现BM25重排序
	return items
}

// rerankHybrid 混合重排序
func (r *RAG) rerankHybrid(ctx context.Context, query string, items []*RetrievalItem) []*RetrievalItem {
	// 结合多种重排序方法
	items = r.rerankWithCrossEncoder(ctx, query, items)
	items = r.rerankWithBM25(ctx, query, items)
	return items
}

// applyFilters 应用过滤器
func (r *RAG) applyFilters(items []*RetrievalItem, opts *RAGOptions) []*RetrievalItem {
	var filtered []*RetrievalItem

	for _, item := range items {
		// 相关性阈值过滤
		if item.Score < r.config.RelevanceThreshold {
			continue
		}

		// 质量过滤
		if r.config.EnableQualityFilter && opts.QualityFilter != nil {
			if item.Item.Quality < opts.QualityFilter.MinQuality {
				continue
			}
			if item.Item.Confidence < opts.QualityFilter.MinConfidence {
				continue
			}
		}

		// 时效性过滤
		if r.config.EnableTemporalFilter && opts.TimeRange != nil {
			if item.Item.CreatedAt.Before(opts.TimeRange.Start) ||
				item.Item.CreatedAt.After(opts.TimeRange.End) {
				continue
			}
		}

		filtered = append(filtered, item)
	}

	return filtered
}

// generateContext 生成上下文
func (r *RAG) generateContext(_ context.Context, items []*RetrievalItem, _ string) (string, []*KnowledgeItem, error) {
	var contextItems []*KnowledgeItem
	var contextParts []string

	switch r.config.ContextStrategy {
	case ContextStrategyAll:
		for _, item := range items {
			contextItems = append(contextItems, item.Item)
			contextParts = append(contextParts, r.formatContextItem(item.Item))
		}

	case ContextStrategyTopK:
		topK := min(len(items), 5)
		for i := range topK {
			contextItems = append(contextItems, items[i].Item)
			contextParts = append(contextParts, r.formatContextItem(items[i].Item))
		}

	case ContextStrategyDiverse:
		contextItems = r.selectDiverseItems(items, 5)
		for _, item := range contextItems {
			contextParts = append(contextParts, r.formatContextItem(item))
		}

	case ContextStrategyChain:
		// TODO: 实现推理链生成
		for _, item := range items {
			contextItems = append(contextItems, item.Item)
			contextParts = append(contextParts, r.formatContextItem(item.Item))
		}
	}

	context := strings.Join(contextParts, "\n\n")

	// 上下文压缩
	if r.config.ContextCompression && len(context) > r.config.MaxContextLength {
		context = r.compressContext(context)
	}

	return context, contextItems, nil
}

// selectDiverseItems 选择多样化的项目
func (r *RAG) selectDiverseItems(items []*RetrievalItem, maxCount int) []*KnowledgeItem {
	if len(items) <= maxCount {
		var result []*KnowledgeItem
		for _, item := range items {
			result = append(result, item.Item)
		}
		return result
	}

	selected := make([]*KnowledgeItem, 0, maxCount)
	selectedTypes := make(map[KnowledgeType]bool)

	// 首先选择不同类型的项
	for _, item := range items {
		if len(selected) >= maxCount {
			break
		}
		if !selectedTypes[item.Item.Type] {
			selected = append(selected, item.Item)
			selectedTypes[item.Item.Type] = true
		}
	}

	// 如果还有空间，添加最高分的项
	for _, item := range items {
		if len(selected) >= maxCount {
			break
		}
		if !containsItem(selected, item.Item) {
			selected = append(selected, item.Item)
		}
	}

	return selected
}

// formatContextItem 格式化上下文项目
func (r *RAG) formatContextItem(item *KnowledgeItem) string {
	var parts []string

	if item.Title != "" {
		parts = append(parts, "## "+item.Title)
	}

	if item.Description != "" {
		parts = append(parts, item.Description)
	}

	if item.Content != "" {
		parts = append(parts, item.Content)
	}

	if len(item.Tags) > 0 {
		parts = append(parts, "Tags: "+strings.Join(item.Tags, ", "))
	}

	if item.Source != "" {
		parts = append(parts, "Source: "+item.Source)
	}

	return strings.Join(parts, "\n")
}

// compressContext 压缩上下文
func (r *RAG) compressContext(context string) string {
	// TODO: 实现更智能的上下文压缩
	// 现在只是简单截断
	if len(context) <= r.config.MaxContextLength {
		return context
	}

	return context[:r.config.MaxContextLength-3] + "..."
}

// enhanceQuery 增强查询
func (r *RAG) enhanceQuery(originalQuery string, items []*RetrievalItem) string {
	if len(items) == 0 {
		return originalQuery
	}

	// 使用检索结果中的关键词增强查询
	keywords := make(map[string]bool)
	for _, item := range items[:min(3, len(items))] {
		if item.Item.Title != "" {
			words := strings.FieldsSeq(item.Item.Title)
			for word := range words {
				if len(word) > 2 {
					keywords[strings.ToLower(word)] = true
				}
			}
		}
	}

	var enhancedKeywords []string
	for keyword := range keywords {
		enhancedKeywords = append(enhancedKeywords, keyword)
	}

	if len(enhancedKeywords) > 0 {
		return fmt.Sprintf("%s %s", originalQuery, strings.Join(enhancedKeywords, " "))
	}

	return originalQuery
}

// expandQuery 扩展查询
func (r *RAG) expandQuery(query string, item *KnowledgeItem) string {
	if item.Title != "" && !strings.Contains(query, item.Title) {
		return fmt.Sprintf("%s %s", query, item.Title)
	}
	return query
}

// calculateConfidence 计算置信度
func (r *RAG) calculateConfidence(items []*RetrievalItem) float64 {
	if len(items) == 0 {
		return 0.0
	}

	var totalScore float64
	for _, item := range items {
		totalScore += item.Score
	}

	averageScore := totalScore / float64(len(items))

	// 考虑结果数量
	resultCountFactor := math.Min(float64(len(items))/10.0, 1.0)

	return averageScore * resultCountFactor
}

// deduplicate 去重
func (r *RAG) deduplicate(items []*RetrievalItem) []*RetrievalItem {
	seen := make(map[string]bool)
	var result []*RetrievalItem

	for _, item := range items {
		if !seen[item.Item.ID] {
			seen[item.Item.ID] = true
			result = append(result, item)
		}
	}

	return result
}

func containsItem(items []*KnowledgeItem, target *KnowledgeItem) bool {
	for _, item := range items {
		if item.ID == target.ID {
			return true
		}
	}
	return false
}
