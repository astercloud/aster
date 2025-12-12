package vector

import "context"

// Document 表示一条向量索引文档。
// 不强制要求 Text/Embedding 同时存在，具体策略由上层控制。
type Document struct {
	ID        string         // 全局唯一 ID
	Text      string         // 原文或 chunk 文本(可选)
	Embedding []float32      // 文本向量(可选)
	Metadata  map[string]any // 业务元数据,如 user_id, project_id 等
	Namespace string         // 逻辑命名空间,例如 "users/alice/projects/demo"
	OrgID     string         // 组织 ID（多租户支持）
	TenantID  string         // 租户 ID（多租户支持）
}

// Query 表示一次向量检索请求。
type Query struct {
	Vector    []float32      // 查询向量
	TopK      int            // 返回结果数量
	Namespace string         // 逻辑命名空间
	Filter    map[string]any // 额外过滤条件(可选)
	OrgID     string         // 组织 ID 过滤（多租户支持）
	TenantID  string         // 租户 ID 过滤（多租户支持）
}

// Hit 表示一次检索命中的结果。
type Hit struct {
	ID       string         // Document ID
	Score    float64        // 相似度分数，越大越相关
	Metadata map[string]any // 透传的元数据
}

// VectorStore 抽象向量存储接口。
// 核心运行时只依赖该接口，不关心具体实现(pgvector/Qdrant/内存等)。
type VectorStore interface {
	Upsert(ctx context.Context, docs []Document) error
	Delete(ctx context.Context, ids []string) error
	Query(ctx context.Context, q Query) ([]Hit, error)
	Close() error
}
