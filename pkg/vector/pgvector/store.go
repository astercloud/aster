package pgvector

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/astercloud/aster/pkg/vector"
)

// Config 配置 PgVector 向量存储。
// 需要数据库已安装 pgvector 扩展:
//
//	CREATE EXTENSION IF NOT EXISTS vector;
type Config struct {
	// DSN PostgreSQL 连接串, 如:
	//   postgres://user:password@localhost:5432/dbname?sslmode=disable
	DSN string

	// Table 存储向量的表名, 默认 "agent_vectors"。
	// 表结构示例:
	//   CREATE TABLE agent_vectors (
	//     id TEXT PRIMARY KEY,
	//     namespace TEXT,
	//     embedding VECTOR(1536),
	//     metadata JSONB
	//   );
	Table string

	// Dimension 向量维度, 需要与 embedding 模型一致。
	Dimension int

	// Metric 相似度度量, 当前支持 "cosine" (默认) 或 "l2"。
	Metric string
}

// Store 使用 pgvector 实现的 VectorStore。
type Store struct {
	pool   *pgxpool.Pool
	table  string
	metric string
	dim    int
}

// New 创建 PgVector 向量存储。
func New(cfg *Config) (*Store, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}
	if cfg.DSN == "" {
		return nil, errors.New("dsn is required")
	}
	table := cfg.Table
	if table == "" {
		table = "agent_vectors"
	}
	if cfg.Dimension <= 0 {
		return nil, errors.New("dimension must be > 0")
	}
	metric := strings.ToLower(cfg.Metric)
	if metric == "" {
		metric = "cosine"
	}
	if metric != "cosine" && metric != "l2" {
		return nil, fmt.Errorf("unsupported metric: %s", cfg.Metric)
	}

	pool, err := pgxpool.New(context.Background(), cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("connect pgvector: %w", err)
	}

	return &Store{
		pool:   pool,
		table:  table,
		metric: metric,
		dim:    cfg.Dimension,
	}, nil
}

// Upsert 将文档插入或更新到 pgvector 表。
func (s *Store) Upsert(ctx context.Context, docs []vector.Document) error {
	if len(docs) == 0 {
		return nil
	}

	const tmpl = `
INSERT INTO %s (id, namespace, embedding, metadata)
VALUES ($1, $2, $3, COALESCE($4, '{}'::jsonb))
ON CONFLICT (id) DO UPDATE
SET namespace = EXCLUDED.namespace,
    embedding = EXCLUDED.embedding,
    metadata  = EXCLUDED.metadata;
`
	query := fmt.Sprintf(tmpl, s.table)

	for _, d := range docs {
		if d.ID == "" || len(d.Embedding) == 0 {
			continue
		}
		if len(d.Embedding) != s.dim {
			return fmt.Errorf("embedding dimension mismatch for id=%s: got %d, want %d", d.ID, len(d.Embedding), s.dim)
		}
		_, err := s.pool.Exec(ctx, query, d.ID, d.Namespace, float32SliceToPgVector(d.Embedding), d.Metadata)
		if err != nil {
			return fmt.Errorf("upsert vector %s: %w", d.ID, err)
		}
	}
	return nil
}

// Delete 从 pgvector 表中删除指定文档。
func (s *Store) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	const tmpl = `DELETE FROM %s WHERE id = ANY($1);`
	query := fmt.Sprintf(tmpl, s.table)
	_, err := s.pool.Exec(ctx, query, ids)
	return err
}

// Query 在指定命名空间内执行向量检索。
func (s *Store) Query(ctx context.Context, q vector.Query) ([]vector.Hit, error) {
	if len(q.Vector) != s.dim {
		return nil, fmt.Errorf("query vector dimension mismatch: got %d, want %d", len(q.Vector), s.dim)
	}

	topK := q.TopK
	if topK <= 0 {
		topK = 5
	}

	// pgvector 距离越小越相似, 我们将 score 定义为 1 - distance (cosine) 或负的 L2。
	var distanceExpr string
	switch s.metric {
	case "cosine":
		distanceExpr = "embedding <=> $1" // cosine distance
	case "l2":
		distanceExpr = "embedding <-> $1" // L2 distance
	default:
		distanceExpr = "embedding <=> $1"
	}

	ns := q.Namespace
	if ns == "" {
		ns = "default"
	}

	const baseTmpl = `
SELECT id, %s AS distance, metadata
FROM %s
WHERE namespace = $2
ORDER BY distance ASC
LIMIT $3;
`
	query := fmt.Sprintf(baseTmpl, distanceExpr, s.table)

	rows, err := s.pool.Query(ctx, query, float32SliceToPgVector(q.Vector), ns, topK)
	if err != nil {
		return nil, fmt.Errorf("query vectors: %w", err)
	}
	defer rows.Close()

	var hits []vector.Hit
	for rows.Next() {
		var (
			id       string
			distance float64
			meta     map[string]any
		)
		if err := rows.Scan(&id, &distance, &meta); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		score := distanceToScore(distance, s.metric)
		hits = append(hits, vector.Hit{
			ID:       id,
			Score:    score,
			Metadata: meta,
		})
	}
	return hits, nil
}

// Close 关闭连接池。
func (s *Store) Close() error {
	if s.pool != nil {
		s.pool.Close()
	}
	return nil
}

// float32SliceToPgVector 将 []float32 转为 pgvector 兼容的 []float32。
// 这里直接复用原 slice, pgx 会将其编码为 pgvector 类型。
func float32SliceToPgVector(v []float32) []float32 {
	return v
}

func distanceToScore(distance float64, metric string) float64 {
	switch metric {
	case "cosine":
		// cosine distance in [0,2], 相似度越高距离越小
		score := 1.0 - distance
		if score < -1 {
			score = -1
		}
		if score > 1 {
			score = 1
		}
		return score
	case "l2":
		// 简单取负值, 距离越小得分越高
		return -distance
	default:
		return 1.0 - distance
	}
}
