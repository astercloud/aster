package weaviate

import (
	"context"
	"fmt"
	"time"

	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/multitenancy"
	"github.com/astercloud/aster/pkg/vector"
	"github.com/go-openapi/strfmt"
	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/auth"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
)

var weaviateLog = logging.ForComponent("WeaviateStore")

// Config Weaviate 配置
type Config struct {
	Host      string `json:"host" yaml:"host"`             // Weaviate 服务地址，如 "localhost:8080"
	Scheme    string `json:"scheme" yaml:"scheme"`         // http 或 https
	ClassName string `json:"class_name" yaml:"class_name"` // Weaviate 类名
	APIKey    string `json:"api_key" yaml:"api_key"`       // API Key（可选）
	Namespace string `json:"namespace" yaml:"namespace"`   // 默认命名空间
}

// Store Weaviate 向量存储实现
type Store struct {
	client    *weaviate.Client
	className string
	namespace string
}

// NewStore 创建 Weaviate 存储实例
func NewStore(cfg *Config) (*Store, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("weaviate host is required")
	}
	if cfg.ClassName == "" {
		return nil, fmt.Errorf("weaviate class name is required")
	}

	scheme := cfg.Scheme
	if scheme == "" {
		scheme = "http"
	}

	// 构建 Weaviate 客户端配置
	clientCfg := weaviate.Config{
		Host:   cfg.Host,
		Scheme: scheme,
	}

	// 如果提供了 API Key，添加认证
	if cfg.APIKey != "" {
		clientCfg.AuthConfig = auth.ApiKey{Value: cfg.APIKey}
	}

	// 创建客户端
	client, err := weaviate.NewClient(clientCfg)
	if err != nil {
		return nil, fmt.Errorf("create weaviate client: %w", err)
	}

	store := &Store{
		client:    client,
		className: cfg.ClassName,
		namespace: cfg.Namespace,
	}

	// 确保类存在（如果不存在则创建）
	ctx := context.Background()
	if err := store.ensureClass(ctx); err != nil {
		return nil, fmt.Errorf("ensure class: %w", err)
	}

	weaviateLog.Info(ctx, "weaviate store created", map[string]any{
		"host":      cfg.Host,
		"class":     cfg.ClassName,
		"namespace": cfg.Namespace,
	})

	return store, nil
}

// ensureClass 确保 Weaviate 类存在
func (s *Store) ensureClass(ctx context.Context) error {
	// 检查类是否存在
	exists, err := s.client.Schema().ClassExistenceChecker().
		WithClassName(s.className).
		Do(ctx)

	if err != nil {
		return fmt.Errorf("check class existence: %w", err)
	}

	if exists {
		weaviateLog.Debug(ctx, "class already exists", map[string]any{"class": s.className})
		return nil
	}

	// 创建类
	vectorizer := "none"
	classObj := &models.Class{
		Class:      s.className,
		Vectorizer: vectorizer,
		Properties: []*models.Property{
			{
				Name:     "text",
				DataType: []string{"text"},
			},
			{
				Name:     "org_id",
				DataType: []string{"string"},
			},
			{
				Name:     "tenant_id",
				DataType: []string{"string"},
			},
			{
				Name:     "namespace",
				DataType: []string{"string"},
			},
		},
	}

	if err := s.client.Schema().ClassCreator().
		WithClass(classObj).
		Do(ctx); err != nil {
		return fmt.Errorf("create class: %w", err)
	}

	weaviateLog.Info(ctx, "class created", map[string]any{"class": s.className})
	return nil
}

// Upsert 插入或更新文档
func (s *Store) Upsert(ctx context.Context, docs []vector.Document) error {
	if len(docs) == 0 {
		return nil
	}

	// 从上下文获取多租户信息（如果有）
	orgID, _ := multitenancy.GetOrgID(ctx)
	tenantID, _ := multitenancy.GetTenantID(ctx)

	startTime := time.Now()

	// 批量插入
	batcher := s.client.Batch().ObjectsBatcher()

	for _, doc := range docs {
		// 使用文档中的租户信息，如果没有则使用上下文中的
		docOrgID := doc.OrgID
		if docOrgID == "" {
			docOrgID = orgID
		}
		docTenantID := doc.TenantID
		if docTenantID == "" {
			docTenantID = tenantID
		}
		docNamespace := doc.Namespace
		if docNamespace == "" {
			docNamespace = s.namespace
		}

		// 构建属性
		properties := map[string]any{
			"text":      doc.Text,
			"org_id":    docOrgID,
			"tenant_id": docTenantID,
			"namespace": docNamespace,
		}

		// 添加自定义元数据
		for k, v := range doc.Metadata {
			properties[k] = v
		}

		// 添加对象到批处理
		obj := &models.Object{
			Class:      s.className,
			Properties: properties,
			Vector:     doc.Embedding,
		}

		// 如果有 ID 则设置
		if doc.ID != "" {
			obj.ID = strfmt.UUID(doc.ID)
		}

		batcher = batcher.WithObjects(obj)
	}

	// 执行批量插入
	results, err := batcher.Do(ctx)
	if err != nil {
		return fmt.Errorf("batch upsert: %w", err)
	}

	// 检查错误
	for _, result := range results {
		if result.Result != nil && result.Result.Errors != nil {
			weaviateLog.Warn(ctx, "upsert error", map[string]any{
				"errors": result.Result.Errors,
			})
		}
	}

	weaviateLog.Debug(ctx, "documents upserted", map[string]any{
		"count":    len(docs),
		"duration": time.Since(startTime).Milliseconds(),
	})

	return nil
}

// Query 向量检索
func (s *Store) Query(ctx context.Context, q vector.Query) ([]vector.Hit, error) {
	if len(q.Vector) == 0 {
		return nil, fmt.Errorf("vector is required for query")
	}

	startTime := time.Now()

	// 从上下文获取多租户信息（如果请求中没有）
	orgID := q.OrgID
	if orgID == "" {
		orgID, _ = multitenancy.GetOrgID(ctx)
	}
	tenantID := q.TenantID
	if tenantID == "" {
		tenantID, _ = multitenancy.GetTenantID(ctx)
	}
	namespace := q.Namespace
	if namespace == "" {
		namespace = s.namespace
	}

	// 构建 where 过滤条件
	var whereFilters []*filters.WhereBuilder

	if orgID != "" {
		whereFilters = append(whereFilters, filters.Where().
			WithPath([]string{"org_id"}).
			WithOperator(filters.Equal).
			WithValueString(orgID))
	}

	if tenantID != "" {
		whereFilters = append(whereFilters, filters.Where().
			WithPath([]string{"tenant_id"}).
			WithOperator(filters.Equal).
			WithValueString(tenantID))
	}

	if namespace != "" {
		whereFilters = append(whereFilters, filters.Where().
			WithPath([]string{"namespace"}).
			WithOperator(filters.Equal).
			WithValueString(namespace))
	}

	// 添加自定义过滤条件
	for k, v := range q.Filter {
		if strVal, ok := v.(string); ok {
			whereFilters = append(whereFilters, filters.Where().
				WithPath([]string{k}).
				WithOperator(filters.Equal).
				WithValueString(strVal))
		}
	}

	// 构建 GraphQL 查询
	limit := q.TopK
	if limit == 0 {
		limit = 10
	}

	nearVector := s.client.GraphQL().NearVectorArgBuilder().
		WithVector(q.Vector)

	fields := []graphql.Field{
		{Name: "text"},
		{Name: "org_id"},
		{Name: "tenant_id"},
		{Name: "namespace"},
		{Name: "_additional", Fields: []graphql.Field{
			{Name: "id"},
			{Name: "certainty"},
		}},
	}

	query := s.client.GraphQL().Get().
		WithClassName(s.className).
		WithNearVector(nearVector).
		WithLimit(limit).
		WithFields(fields...)

	// 添加 where 过滤
	if len(whereFilters) > 0 {
		var combinedFilter *filters.WhereBuilder
		if len(whereFilters) == 1 {
			combinedFilter = whereFilters[0]
		} else {
			combinedFilter = filters.Where().
				WithOperator(filters.And).
				WithOperands(whereFilters)
		}
		query = query.WithWhere(combinedFilter)
	}

	// 执行查询
	result, err := query.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("execute query: %w", err)
	}

	// 解析结果
	hits := make([]vector.Hit, 0)

	if result.Data != nil {
		if classData, ok := result.Data["Get"].(map[string]any); ok {
			if objects, ok := classData[s.className].([]any); ok {
				for _, obj := range objects {
					if objMap, ok := obj.(map[string]any); ok {
						hit := vector.Hit{
							Metadata: make(map[string]any),
						}

						// 提取基础字段
						if text, ok := objMap["text"].(string); ok {
							hit.Metadata["text"] = text
						}
						if orgID, ok := objMap["org_id"].(string); ok {
							hit.Metadata["org_id"] = orgID
						}
						if tenantID, ok := objMap["tenant_id"].(string); ok {
							hit.Metadata["tenant_id"] = tenantID
						}
						if namespace, ok := objMap["namespace"].(string); ok {
							hit.Metadata["namespace"] = namespace
						}

						// 提取 ID 和分数
						if additional, ok := objMap["_additional"].(map[string]any); ok {
							if id, ok := additional["id"].(string); ok {
								hit.ID = id
							}
							if certainty, ok := additional["certainty"].(float64); ok {
								hit.Score = certainty
							}
						}

						hits = append(hits, hit)
					}
				}
			}
		}
	}

	duration := time.Since(startTime).Milliseconds()

	weaviateLog.Debug(ctx, "query completed", map[string]any{
		"results":  len(hits),
		"duration": duration,
	})

	return hits, nil
}

// Delete 删除文档
func (s *Store) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	startTime := time.Now()

	for _, id := range ids {
		err := s.client.Data().Deleter().
			WithClassName(s.className).
			WithID(id).
			Do(ctx)

		if err != nil {
			weaviateLog.Warn(ctx, "failed to delete document", map[string]any{
				"id":    id,
				"error": err,
			})
		}
	}

	weaviateLog.Debug(ctx, "documents deleted", map[string]any{
		"count":    len(ids),
		"duration": time.Since(startTime).Milliseconds(),
	})

	return nil
}

// Close 关闭连接
func (s *Store) Close() error {
	// Weaviate Go Client v4 没有显式的 Close 方法
	return nil
}
