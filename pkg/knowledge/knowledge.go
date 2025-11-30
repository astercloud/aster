package knowledge

import (
	"context"
	"time"

	"github.com/astercloud/aster/pkg/memory"
	"github.com/astercloud/aster/pkg/vector"
)

// KnowledgeType 知识类型
type KnowledgeType string

const (
	KnowledgeTypeDocument   KnowledgeType = "document"   // 文档知识
	KnowledgeTypeConcept    KnowledgeType = "concept"    // 概念知识
	KnowledgeTypeProcedure  KnowledgeType = "procedure"  // 流程知识
	KnowledgeTypeFact       KnowledgeType = "fact"       // 事实知识
	KnowledgeTypeExperience KnowledgeType = "experience" // 经验知识
	KnowledgeTypeRule       KnowledgeType = "rule"       // 规则知识
	KnowledgeTypePattern    KnowledgeType = "pattern"    // 模式知识
)

// KnowledgeItem 知识项
type KnowledgeItem struct {
	// 基本信息
	ID          string        `json:"id"`
	Type        KnowledgeType `json:"type"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Content     string        `json:"content"`

	// 分类和标签
	Category string   `json:"category"`
	Tags     []string `json:"tags"`

	// 元数据
	Metadata map[string]any `json:"metadata"`

	// 时效性
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// 置信度和质量
	Confidence float64 `json:"confidence"` // 0-1，知识可信度
	Quality    float64 `json:"quality"`    // 0-1，知识质量评分

	// 来源信息
	Source  string `json:"source"`  // 知识来源
	Author  string `json:"author"`  // 作者
	Version string `json:"version"` // 版本

	// 关系信息
	Relations []KnowledgeRelation `json:"relations"` // 与其他知识的关系

	// 索引信息
	Embedding []float32 `json:"embedding,omitempty"` // 向量表示
	Namespace string    `json:"namespace"`           // 命名空间
}

// KnowledgeRelation 知识关系
type KnowledgeRelation struct {
	Type     RelationType `json:"type"`
	TargetID string       `json:"target_id"`
	Weight   float64      `json:"weight"` // 关系强度
	Label    string       `json:"label"`  // 关系标签
}

// RelationType 关系类型
type RelationType string

const (
	RelationTypeContains    RelationType = "contains"    // 包含关系
	RelationTypeImplies     RelationType = "implies"     // 推理关系
	RelationTypeContradicts RelationType = "contradicts" // 矛盾关系
	RelationTypeSimilar     RelationType = "similar"     // 相似关系
	RelationTypeCauses      RelationType = "causes"      // 因果关系
	RelationTypeEnables     RelationType = "enables"     // 使能关系
	RelationTypeRequires    RelationType = "requires"    // 依赖关系
	RelationTypeInstance    RelationType = "instance"    // 实例关系
	RelationTypePartOf      RelationType = "part_of"     // 组成关系
)

// SearchQuery 知识搜索查询
type SearchQuery struct {
	// 基础查询
	Query    string        `json:"query"`    // 文本查询
	Vector   []float32     `json:"vector"`   // 向量查询
	Type     KnowledgeType `json:"type"`     // 知识类型过滤
	Category string        `json:"category"` // 分类过滤
	Tags     []string      `json:"tags"`     // 标签过滤

	// 高级查询
	Relations []RelationType `json:"relations"` // 关系过滤
	Sources   []string       `json:"sources"`   // 来源过滤
	Authors   []string       `json:"authors"`   // 作者过滤

	// 时效性查询
	After      *time.Time `json:"after,omitempty"`  // 创建时间之后
	Before     *time.Time `json:"before,omitempty"` // 创建时间之前
	NotExpired bool       `json:"not_expired"`      // 只查询未过期的

	// 质量过滤
	MinConfidence float64 `json:"min_confidence"` // 最小置信度
	MinQuality    float64 `json:"min_quality"`    // 最小质量

	// 检索配置
	MaxResults int            `json:"max_results"` // 最大结果数
	Namespace  string         `json:"namespace"`   // 命名空间
	Filters    map[string]any `json:"filters"`     // 自定义过滤

	// 搜索策略
	Strategy     SearchStrategy     `json:"strategy"`      // 搜索策略
	HybridWeight HybridSearchWeight `json:"hybrid_weight"` // 混合搜索权重
}

// SearchStrategy 搜索策略
type SearchStrategy string

const (
	StrategyText   SearchStrategy = "text"   // 纯文本搜索
	StrategyVector SearchStrategy = "vector" // 纯向量搜索
	StrategyHybrid SearchStrategy = "hybrid" // 混合搜索
	StrategyGraph  SearchStrategy = "graph"  // 图搜索
)

// HybridSearchWeight 混合搜索权重
type HybridSearchWeight struct {
	TextWeight   float64 `json:"text_weight"`   // 文本搜索权重
	VectorWeight float64 `json:"vector_weight"` // 向量搜索权重
}

// SearchResult 知识搜索结果
type SearchResult struct {
	Item        KnowledgeItem `json:"item"`
	Score       float64       `json:"score"`       // 相关性评分
	Explanation string        `json:"explanation"` // 结果解释
	Path        []string      `json:"path"`        // 知识路径（图搜索时）
}

// KnowledgeStats 知识统计
type KnowledgeStats struct {
	TotalItems      int64            `json:"total_items"`
	ItemsByType     map[string]int64 `json:"items_by_type"`
	ItemsByCategory map[string]int64 `json:"items_by_category"`
	AverageQuality  float64          `json:"average_quality"`
	LastUpdated     time.Time        `json:"last_updated"`
}

// ManagerConfig 知识管理器配置
type ManagerConfig struct {
	// 存储配置
	MemoryManager *memory.Manager    `json:"-"` // 长期记忆管理器
	VectorStore   vector.VectorStore `json:"-"` // 向量存储
	Embedder      vector.Embedder    `json:"-"` // 向量嵌入器

	// 基础配置
	Namespace  string `json:"namespace"`   // 默认命名空间
	MaxResults int    `json:"max_results"` // 默认最大结果数
	AutoEmbed  bool   `json:"auto_embed"`  // 自动生成向量

	// 质量控制
	MinConfidence float64 `json:"min_confidence"` // 最小置信度阈值
	MinQuality    float64 `json:"min_quality"`    // 最小质量阈值

	// 缓存配置
	CacheEnabled bool          `json:"cache_enabled"`
	CacheTTL     time.Duration `json:"cache_ttl"`

	// 安全配置
	EnablePII   bool `json:"enable_pii"`   // 启用PII检测
	EnableAudit bool `json:"enable_audit"` // 启用审计日志

	// 轻量核心管线
	UseCorePipeline bool `json:"use_core_pipeline"` // 启用轻量 ingest/search 管线

	// 可选策略注入
	PIIStrategy   PIIStrategy   `json:"-"`
	AuditStrategy AuditStrategy `json:"-"`
}

// PIIStrategy 定义可插拔的脱敏策略。
type PIIStrategy interface {
	Sanitize(item *KnowledgeItem) *KnowledgeItem
}

// AuditStrategy 定义可插拔的审计策略。
type AuditStrategy interface {
	Record(action, userID, itemID, details string)
}

// Manager 统一知识管理器接口
type Manager interface {
	// 基础操作
	Add(ctx context.Context, item *KnowledgeItem) error
	Get(ctx context.Context, id string) (*KnowledgeItem, error)
	Update(ctx context.Context, item *KnowledgeItem) error
	Delete(ctx context.Context, id string) error

	// 搜索操作
	Search(ctx context.Context, query *SearchQuery) ([]*SearchResult, error)
	SearchSimilar(ctx context.Context, id string, maxResults int) ([]*SearchResult, error)

	// 关系操作
	AddRelation(ctx context.Context, fromID, toID string, relationType RelationType, weight float64, label string) error
	RemoveRelation(ctx context.Context, fromID, toID string, relationType RelationType) error
	GetRelated(ctx context.Context, id string, relationTypes []RelationType, maxDepth int) ([]*KnowledgeItem, error)

	// 批量操作
	BulkAdd(ctx context.Context, items []*KnowledgeItem) error
	BulkSearch(ctx context.Context, queries []*SearchQuery) ([][]*SearchResult, error)

	// 知识推理
	Reason(ctx context.Context, query string, maxSteps int) ([]*KnowledgeItem, []string, error)

	// 维护操作
	Validate(ctx context.Context, id string) (bool, []string, error)
	Refresh(ctx context.Context, id string) error
	Compress(ctx context.Context, namespace string) error

	// 统计信息
	GetStats(ctx context.Context, namespace string) (*KnowledgeStats, error)

	// 生命周期
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}
