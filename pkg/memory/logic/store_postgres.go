package logic

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/astercloud/aster/pkg/memory"
)

// PostgreSQLStore PostgreSQL 存储实现
type PostgreSQLStore struct {
	db        *sql.DB
	tableName string
	closed    bool
}

// PostgreSQLStoreConfig PostgreSQL 存储配置
type PostgreSQLStoreConfig struct {
	// DB 数据库连接（必需）
	DB *sql.DB

	// TableName 表名（默认 "logic_memories"）
	TableName string

	// AutoMigrate 是否自动创建表（默认 true）
	AutoMigrate bool
}

// NewPostgreSQLStore 创建 PostgreSQL 存储
func NewPostgreSQLStore(config *PostgreSQLStoreConfig) (*PostgreSQLStore, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	if config.DB == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	tableName := config.TableName
	if tableName == "" {
		tableName = "logic_memories"
	}

	store := &PostgreSQLStore{
		db:        config.DB,
		tableName: tableName,
	}

	// 自动创建表
	if config.AutoMigrate {
		if err := store.migrate(); err != nil {
			return nil, fmt.Errorf("failed to migrate: %w", err)
		}
	}

	return store, nil
}

// migrate 创建表结构
func (s *PostgreSQLStore) migrate() error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id VARCHAR(64) PRIMARY KEY,
			namespace VARCHAR(255) NOT NULL,
			scope VARCHAR(20) NOT NULL,
			type VARCHAR(100) NOT NULL,
			category VARCHAR(100),
			key VARCHAR(255) NOT NULL,
			value JSONB NOT NULL DEFAULT '{}',
			description TEXT,
			source_type VARCHAR(50),
			confidence DECIMAL(5,4) DEFAULT 0,
			sources JSONB DEFAULT '[]',
			provenance_created_at TIMESTAMP,
			provenance_updated_at TIMESTAMP,
			provenance_version INT DEFAULT 0,
			access_count INT DEFAULT 0,
			last_accessed TIMESTAMP,
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			UNIQUE(namespace, key)
		);

		CREATE INDEX IF NOT EXISTS idx_%s_namespace ON %s(namespace);
		CREATE INDEX IF NOT EXISTS idx_%s_namespace_type ON %s(namespace, type);
		CREATE INDEX IF NOT EXISTS idx_%s_scope ON %s(scope);
		CREATE INDEX IF NOT EXISTS idx_%s_confidence ON %s(confidence);
		CREATE INDEX IF NOT EXISTS idx_%s_last_accessed ON %s(last_accessed);
	`, s.tableName,
		s.tableName, s.tableName,
		s.tableName, s.tableName,
		s.tableName, s.tableName,
		s.tableName, s.tableName,
		s.tableName, s.tableName)

	_, err := s.db.Exec(query)
	return err
}

// Save 保存或更新 Memory
func (s *PostgreSQLStore) Save(ctx context.Context, mem *LogicMemory) error {
	if s.closed {
		return ErrStoreClosed
	}

	if mem.Namespace == "" {
		return ErrInvalidNamespace
	}

	// 序列化 Value
	valueJSON, err := json.Marshal(mem.Value)
	if err != nil {
		return NewStoreError("MARSHAL_ERROR", "failed to marshal value", err)
	}

	// 序列化 Metadata
	metadataJSON, err := json.Marshal(mem.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	// 处理 Provenance
	var sourceType string
	var confidence float64
	var sourcesJSON []byte
	var provenanceCreatedAt, provenanceUpdatedAt sql.NullTime
	var provenanceVersion int

	if mem.Provenance != nil {
		sourceType = string(mem.Provenance.SourceType)
		confidence = mem.Provenance.Confidence
		sourcesJSON, _ = json.Marshal(mem.Provenance.Sources)
		if !mem.Provenance.CreatedAt.IsZero() {
			provenanceCreatedAt = sql.NullTime{Time: mem.Provenance.CreatedAt, Valid: true}
		}
		if !mem.Provenance.UpdatedAt.IsZero() {
			provenanceUpdatedAt = sql.NullTime{Time: mem.Provenance.UpdatedAt, Valid: true}
		}
		provenanceVersion = mem.Provenance.Version
	} else {
		sourcesJSON = []byte("[]")
	}

	// 处理时间
	now := time.Now()
	if mem.CreatedAt.IsZero() {
		mem.CreatedAt = now
	}
	mem.UpdatedAt = now
	if mem.LastAccessed.IsZero() {
		mem.LastAccessed = now
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (
			id, namespace, scope, type, category, key, value, description,
			source_type, confidence, sources, provenance_created_at, provenance_updated_at, provenance_version,
			access_count, last_accessed, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19
		)
		ON CONFLICT (namespace, key) DO UPDATE SET
			scope = EXCLUDED.scope,
			type = EXCLUDED.type,
			category = EXCLUDED.category,
			value = EXCLUDED.value,
			description = EXCLUDED.description,
			source_type = EXCLUDED.source_type,
			confidence = EXCLUDED.confidence,
			sources = EXCLUDED.sources,
			provenance_created_at = EXCLUDED.provenance_created_at,
			provenance_updated_at = EXCLUDED.provenance_updated_at,
			provenance_version = EXCLUDED.provenance_version,
			access_count = EXCLUDED.access_count,
			last_accessed = EXCLUDED.last_accessed,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at
	`, s.tableName)

	_, err = s.db.ExecContext(ctx, query,
		mem.ID, mem.Namespace, mem.Scope, mem.Type, mem.Category, mem.Key, valueJSON, mem.Description,
		sourceType, confidence, sourcesJSON, provenanceCreatedAt, provenanceUpdatedAt, provenanceVersion,
		mem.AccessCount, mem.LastAccessed, metadataJSON, mem.CreatedAt, mem.UpdatedAt,
	)

	if err != nil {
		return NewStoreError("SAVE_ERROR", "failed to save memory", err)
	}

	return nil
}

// Get 获取单个 Memory
func (s *PostgreSQLStore) Get(ctx context.Context, namespace, key string) (*LogicMemory, error) {
	if s.closed {
		return nil, ErrStoreClosed
	}

	query := fmt.Sprintf(`
		SELECT id, namespace, scope, type, category, key, value, description,
			source_type, confidence, sources, provenance_created_at, provenance_updated_at, provenance_version,
			access_count, last_accessed, metadata, created_at, updated_at
		FROM %s
		WHERE namespace = $1 AND key = $2
	`, s.tableName)

	row := s.db.QueryRowContext(ctx, query, namespace, key)
	return s.scanMemory(row)
}

// Delete 删除 Memory
func (s *PostgreSQLStore) Delete(ctx context.Context, namespace, key string) error {
	if s.closed {
		return ErrStoreClosed
	}

	query := fmt.Sprintf(`DELETE FROM %s WHERE namespace = $1 AND key = $2`, s.tableName)
	_, err := s.db.ExecContext(ctx, query, namespace, key)
	return err
}

// List 列出符合条件的 Memory
func (s *PostgreSQLStore) List(ctx context.Context, namespace string, filters ...Filter) ([]*LogicMemory, error) {
	if s.closed {
		return nil, ErrStoreClosed
	}

	opts := ApplyFilters(filters...)

	// 构建查询
	query := fmt.Sprintf(`
		SELECT id, namespace, scope, type, category, key, value, description,
			source_type, confidence, sources, provenance_created_at, provenance_updated_at, provenance_version,
			access_count, last_accessed, metadata, created_at, updated_at
		FROM %s
		WHERE 1=1
	`, s.tableName)

	args := []any{}
	argIndex := 1

	if namespace != "" {
		query += fmt.Sprintf(" AND namespace = $%d", argIndex)
		args = append(args, namespace)
		argIndex++
	}

	if opts.Type != "" {
		query += fmt.Sprintf(" AND type = $%d", argIndex)
		args = append(args, opts.Type)
		argIndex++
	}

	if opts.Scope != "" {
		query += fmt.Sprintf(" AND scope = $%d", argIndex)
		args = append(args, opts.Scope)
		argIndex++
	}

	if opts.MinConfidence > 0 {
		query += fmt.Sprintf(" AND confidence >= $%d", argIndex)
		args = append(args, opts.MinConfidence)
		// argIndex++ 不需要，后续没有使用
	}

	// 排序
	switch opts.OrderBy {
	case OrderByConfidence:
		query += " ORDER BY confidence DESC"
	case OrderByLastAccessed:
		query += " ORDER BY last_accessed DESC"
	case OrderByCreatedAt:
		query += " ORDER BY created_at DESC"
	case OrderByAccessCount:
		query += " ORDER BY access_count DESC"
	default:
		query += " ORDER BY confidence DESC"
	}

	// 限制数量
	if opts.MaxResults > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.MaxResults)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, NewStoreError("QUERY_ERROR", "failed to list memories", err)
	}
	defer func() { _ = rows.Close() }()

	var memories []*LogicMemory
	for rows.Next() {
		mem, err := s.scanMemoryFromRows(rows)
		if err != nil {
			return nil, err
		}
		memories = append(memories, mem)
	}

	return memories, rows.Err()
}

// SearchByType 按类型搜索
func (s *PostgreSQLStore) SearchByType(ctx context.Context, namespace, memoryType string) ([]*LogicMemory, error) {
	return s.List(ctx, namespace, WithType(memoryType))
}

// SearchByScope 按作用域搜索
func (s *PostgreSQLStore) SearchByScope(ctx context.Context, namespace string, scope MemoryScope) ([]*LogicMemory, error) {
	return s.List(ctx, namespace, WithScope(scope))
}

// GetTopK 获取 TopK Memory
func (s *PostgreSQLStore) GetTopK(ctx context.Context, namespace string, k int, orderBy OrderBy) ([]*LogicMemory, error) {
	return s.List(ctx, namespace, WithTopK(k), WithOrderBy(orderBy))
}

// IncrementAccessCount 增加访问计数
func (s *PostgreSQLStore) IncrementAccessCount(ctx context.Context, namespace, key string) error {
	if s.closed {
		return ErrStoreClosed
	}

	query := fmt.Sprintf(`
		UPDATE %s
		SET access_count = access_count + 1, last_accessed = $1, updated_at = $1
		WHERE namespace = $2 AND key = $3
	`, s.tableName)

	result, err := s.db.ExecContext(ctx, query, time.Now(), namespace, key)
	if err != nil {
		return NewStoreError("UPDATE_ERROR", "failed to increment access count", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrMemoryNotFound
	}

	return nil
}

// GetStats 获取统计信息
func (s *PostgreSQLStore) GetStats(ctx context.Context, namespace string) (*MemoryStats, error) {
	if s.closed {
		return nil, ErrStoreClosed
	}

	stats := &MemoryStats{
		CountByType:  make(map[string]int),
		CountByScope: make(map[MemoryScope]int),
	}

	// 总数和平均置信度
	query := fmt.Sprintf(`
		SELECT COUNT(*), COALESCE(AVG(confidence), 0), COALESCE(MAX(updated_at), NOW())
		FROM %s
		WHERE ($1 = '' OR namespace = $1)
	`, s.tableName)

	var lastUpdated time.Time
	err := s.db.QueryRowContext(ctx, query, namespace).Scan(&stats.TotalCount, &stats.AverageConfidence, &lastUpdated)
	if err != nil {
		return nil, NewStoreError("QUERY_ERROR", "failed to get stats", err)
	}
	stats.LastUpdated = lastUpdated

	// 按类型统计
	query = fmt.Sprintf(`
		SELECT type, COUNT(*)
		FROM %s
		WHERE ($1 = '' OR namespace = $1)
		GROUP BY type
	`, s.tableName)

	rows, err := s.db.QueryContext(ctx, query, namespace)
	if err != nil {
		return nil, NewStoreError("QUERY_ERROR", "failed to get type stats", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var memType string
		var count int
		if err := rows.Scan(&memType, &count); err != nil {
			continue
		}
		stats.CountByType[memType] = count
	}

	// 按作用域统计
	query = fmt.Sprintf(`
		SELECT scope, COUNT(*)
		FROM %s
		WHERE ($1 = '' OR namespace = $1)
		GROUP BY scope
	`, s.tableName)

	rows, err = s.db.QueryContext(ctx, query, namespace)
	if err != nil {
		return nil, NewStoreError("QUERY_ERROR", "failed to get scope stats", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var scope string
		var count int
		if err := rows.Scan(&scope, &count); err != nil {
			continue
		}
		stats.CountByScope[MemoryScope(scope)] = count
	}

	return stats, nil
}

// Prune 清理低价值 Memory
func (s *PostgreSQLStore) Prune(ctx context.Context, criteria PruneCriteria) (int, error) {
	if s.closed {
		return 0, ErrStoreClosed
	}

	args := []any{}
	argIndex := 1

	// 构建 OR 条件
	conditions := []string{}

	if criteria.MinConfidence > 0 {
		conditions = append(conditions, fmt.Sprintf("confidence < $%d", argIndex))
		args = append(args, criteria.MinConfidence)
		argIndex++
	}

	if criteria.SinceLastAccess > 0 {
		conditions = append(conditions, fmt.Sprintf("last_accessed < $%d", argIndex))
		args = append(args, time.Now().Add(-criteria.SinceLastAccess))
		argIndex++
	}

	if criteria.MinAccessCount > 0 && criteria.MaxAge > 0 {
		conditions = append(conditions, fmt.Sprintf("(access_count < $%d AND created_at < $%d)", argIndex, argIndex+1))
		args = append(args, criteria.MinAccessCount, time.Now().Add(-criteria.MaxAge))
		// argIndex += 2 不需要，后续没有使用
	}

	if len(conditions) == 0 {
		return 0, nil
	}

	// 构建查询
	query := fmt.Sprintf(`
		DELETE FROM %s
		WHERE %s
	`, s.tableName, conditions[0])

	for i := 1; i < len(conditions); i++ {
		query += " OR " + conditions[i]
	}

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, NewStoreError("DELETE_ERROR", "failed to prune memories", err)
	}

	rowsAffected, _ := result.RowsAffected()
	return int(rowsAffected), nil
}

// Close 关闭存储
func (s *PostgreSQLStore) Close() error {
	s.closed = true
	// 不关闭 db，因为它是外部传入的
	return nil
}

// scanMemory 从单行扫描 Memory
func (s *PostgreSQLStore) scanMemory(row *sql.Row) (*LogicMemory, error) {
	mem := &LogicMemory{}
	var valueJSON, metadataJSON, sourcesJSON []byte
	var sourceType sql.NullString
	var confidence float64
	var provenanceCreatedAt, provenanceUpdatedAt sql.NullTime
	var provenanceVersion int
	var category sql.NullString
	var description sql.NullString
	var lastAccessed sql.NullTime

	err := row.Scan(
		&mem.ID, &mem.Namespace, &mem.Scope, &mem.Type, &category, &mem.Key, &valueJSON, &description,
		&sourceType, &confidence, &sourcesJSON, &provenanceCreatedAt, &provenanceUpdatedAt, &provenanceVersion,
		&mem.AccessCount, &lastAccessed, &metadataJSON, &mem.CreatedAt, &mem.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrMemoryNotFound
	}
	if err != nil {
		return nil, NewStoreError("SCAN_ERROR", "failed to scan memory", err)
	}

	// 反序列化
	if len(valueJSON) > 0 {
		if err := json.Unmarshal(valueJSON, &mem.Value); err != nil {
			return nil, NewStoreError("UNMARSHAL_ERROR", "failed to unmarshal value", err)
		}
	}
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &mem.Metadata); err != nil {
			// metadata 反序列化失败不是致命错误，使用空 map
			mem.Metadata = make(map[string]any)
		}
	}

	if category.Valid {
		mem.Category = category.String
	}
	if description.Valid {
		mem.Description = description.String
	}
	if lastAccessed.Valid {
		mem.LastAccessed = lastAccessed.Time
	}

	// 构建 Provenance
	if sourceType.Valid {
		var sources []string
		if len(sourcesJSON) > 0 {
			_ = json.Unmarshal(sourcesJSON, &sources) // sources 反序列化失败使用空切片
		}

		mem.Provenance = &memory.MemoryProvenance{
			SourceType: memory.SourceType(sourceType.String),
			Confidence: confidence,
			Sources:    sources,
			Version:    provenanceVersion,
		}
		if provenanceCreatedAt.Valid {
			mem.Provenance.CreatedAt = provenanceCreatedAt.Time
		}
		if provenanceUpdatedAt.Valid {
			mem.Provenance.UpdatedAt = provenanceUpdatedAt.Time
		}
	}

	return mem, nil
}

// scanMemoryFromRows 从 rows 扫描 Memory
func (s *PostgreSQLStore) scanMemoryFromRows(rows *sql.Rows) (*LogicMemory, error) {
	mem := &LogicMemory{}
	var valueJSON, metadataJSON, sourcesJSON []byte
	var sourceType sql.NullString
	var confidence float64
	var provenanceCreatedAt, provenanceUpdatedAt sql.NullTime
	var provenanceVersion int
	var category sql.NullString
	var description sql.NullString
	var lastAccessed sql.NullTime

	err := rows.Scan(
		&mem.ID, &mem.Namespace, &mem.Scope, &mem.Type, &category, &mem.Key, &valueJSON, &description,
		&sourceType, &confidence, &sourcesJSON, &provenanceCreatedAt, &provenanceUpdatedAt, &provenanceVersion,
		&mem.AccessCount, &lastAccessed, &metadataJSON, &mem.CreatedAt, &mem.UpdatedAt,
	)

	if err != nil {
		return nil, NewStoreError("SCAN_ERROR", "failed to scan memory", err)
	}

	// 反序列化
	if len(valueJSON) > 0 {
		if err := json.Unmarshal(valueJSON, &mem.Value); err != nil {
			return nil, NewStoreError("UNMARSHAL_ERROR", "failed to unmarshal value", err)
		}
	}
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &mem.Metadata); err != nil {
			// metadata 反序列化失败不是致命错误，使用空 map
			mem.Metadata = make(map[string]any)
		}
	}

	if category.Valid {
		mem.Category = category.String
	}
	if description.Valid {
		mem.Description = description.String
	}
	if lastAccessed.Valid {
		mem.LastAccessed = lastAccessed.Time
	}

	// 构建 Provenance
	if sourceType.Valid {
		var sources []string
		if len(sourcesJSON) > 0 {
			_ = json.Unmarshal(sourcesJSON, &sources) // sources 反序列化失败使用空切片
		}

		mem.Provenance = &memory.MemoryProvenance{
			SourceType: memory.SourceType(sourceType.String),
			Confidence: confidence,
			Sources:    sources,
			Version:    provenanceVersion,
		}
		if provenanceCreatedAt.Valid {
			mem.Provenance.CreatedAt = provenanceCreatedAt.Time
		}
		if provenanceUpdatedAt.Valid {
			mem.Provenance.UpdatedAt = provenanceUpdatedAt.Time
		}
	}

	return mem, nil
}

// 确保 PostgreSQLStore 实现 LogicMemoryStore 接口
var _ LogicMemoryStore = (*PostgreSQLStore)(nil)
