package core

// IngestRequest 描述一次知识摄入请求（轻量模型）。
type IngestRequest struct {
	ID        string         // 文档 ID，可为空则自动生成
	Text      string         // 原文文本
	Namespace string         // 命名空间，可为空使用默认
	Metadata  map[string]any // 自定义元数据
}

// Chunk 表示切分后的最小文本单元。
type Chunk struct {
	ID        string
	Text      string
	Namespace string
	Metadata  map[string]any
}

// SearchHit 表示一次检索命中。
type SearchHit struct {
	ID       string
	Score    float64
	Text     string
	Metadata map[string]any
}
