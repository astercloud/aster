package mysql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/astercloud/aster/pkg/session"
)

// Service MySQL Session 服务实现
// 支持 MySQL 8.0+ (JSON 列类型)
type Service struct {
	db *gorm.DB
}

// Config MySQL 配置
type Config struct {
	// DSN 数据库连接字符串
	// 例如: "user:password@tcp(127.0.0.1:3306)/agentsdk?charset=utf8mb4&parseTime=True&loc=Local"
	DSN string

	// MaxIdleConns 最大空闲连接数
	MaxIdleConns int

	// MaxOpenConns 最大打开连接数
	MaxOpenConns int

	// ConnMaxLifetime 连接最大生命周期
	ConnMaxLifetime time.Duration

	// LogLevel GORM 日志级别
	LogLevel logger.LogLevel

	// AutoMigrate 是否自动迁移表结构
	AutoMigrate bool
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: time.Hour,
		LogLevel:        logger.Warn,
		AutoMigrate:     true,
	}
}

// NewService 创建 MySQL Session 服务
func NewService(cfg *Config) (*Service, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if cfg.DSN == "" {
		return nil, errors.New("DSN is required")
	}

	// 打开数据库连接
	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(cfg.LogLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("connect to mysql: %w", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// 自动迁移
	if cfg.AutoMigrate {
		if err := db.AutoMigrate(
			&SessionModel{},
			&StateModel{},
			&EventModel{},
			&ArtifactModel{},
		); err != nil {
			return nil, fmt.Errorf("auto migrate: %w", err)
		}

		// MySQL 不支持某些 PostgreSQL 特性，需要额外处理
		if err := createAdditionalIndexes(db); err != nil {
			return nil, fmt.Errorf("create indexes: %w", err)
		}
	}

	return &Service{db: db}, nil
}

// createAdditionalIndexes 创建额外的索引
func createAdditionalIndexes(db *gorm.DB) error {
	// MySQL 8.0+ 支持 JSON 列，但索引创建方式不同
	// 这里可以添加特定的索引创建逻辑
	return nil
}

// Create 实现 session.Service 接口
func (s *Service) Create(ctx context.Context, req *session.CreateRequest) (*session.SessionData, error) {
	id := uuid.New().String()

	model := &SessionModel{
		ID:        id,
		AppName:   req.AppName,
		UserID:    req.UserID,
		AgentID:   req.AgentID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return s.toSession(ctx, model), nil
}

// Get 实现 session.Service 接口
func (s *Service) Get(ctx context.Context, sessionID string) (*session.SessionData, error) {
	var model SessionModel
	if err := s.db.WithContext(ctx).
		Where("id = ?", sessionID).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, session.ErrSessionNotFound
		}
		return nil, fmt.Errorf("get session: %w", err)
	}

	return s.toSession(ctx, &model), nil
}

// List 实现 session.Service 接口
func (s *Service) List(ctx context.Context, userID string, opts *session.ListOptions) ([]*session.SessionData, error) {
	var models []SessionModel
	query := s.db.WithContext(ctx).Where("user_id = ?", userID)

	if opts != nil {
		if opts.AppName != "" {
			query = query.Where("app_name = ?", opts.AppName)
		}
		if opts.Limit > 0 {
			query = query.Limit(opts.Limit)
		}
		if opts.Offset > 0 {
			query = query.Offset(opts.Offset)
		}
	}

	query = query.Order("updated_at DESC")

	if err := query.Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	sessions := make([]*session.SessionData, len(models))
	for i, model := range models {
		sessions[i] = s.toSession(ctx, &model)
	}
	return sessions, nil
}

// Delete 实现 session.Service 接口
func (s *Service) Delete(ctx context.Context, sessionID string) error {
	result := s.db.WithContext(ctx).Delete(&SessionModel{}, "id = ?", sessionID)
	if result.Error != nil {
		return fmt.Errorf("delete session: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return session.ErrSessionNotFound
	}
	return nil
}

// AppendEvent 实现 session.Service 接口
func (s *Service) AppendEvent(ctx context.Context, sessionID string, event *session.Event) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 序列化事件内容
		contentJSON, err := json.Marshal(event.Content)
		if err != nil {
			return fmt.Errorf("marshal content: %w", err)
		}

		// 2. 序列化事件动作
		var actionsJSON []byte
		if event.Actions.StateDelta != nil || event.Actions.ArtifactDelta != nil {
			actionsJSON, err = json.Marshal(event.Actions)
			if err != nil {
				return fmt.Errorf("marshal actions: %w", err)
			}
		}

		// 3. 序列化元数据
		var metadataJSON []byte
		if len(event.Metadata) > 0 {
			metadataJSON, err = json.Marshal(event.Metadata)
			if err != nil {
				return fmt.Errorf("marshal metadata: %w", err)
			}
		}

		// 4. 处理长时运行工具 ID（MySQL 使用 JSON 数组存储）
		longRunningToolsJSON, _ := json.Marshal(event.LongRunningToolIDs)

		// 5. 创建事件记录
		eventModel := &EventModel{
			ID:                 event.ID,
			SessionID:          sessionID,
			InvocationID:       event.InvocationID,
			Branch:             event.Branch,
			Author:             event.Author,
			AgentID:            event.AgentID,
			Timestamp:          event.Timestamp,
			Content:            contentJSON,
			Actions:            actionsJSON,
			LongRunningToolIDs: longRunningToolsJSON,
			Metadata:           metadataJSON,
		}

		if err := tx.Create(eventModel).Error; err != nil {
			return fmt.Errorf("create event: %w", err)
		}

		// 6. 应用状态变更
		if len(event.Actions.StateDelta) > 0 {
			for key, value := range event.Actions.StateDelta {
				scope, actualKey := parseStateKey(key)

				valueJSON, err := json.Marshal(value)
				if err != nil {
					return fmt.Errorf("marshal state value: %w", err)
				}

				now := time.Now()

				// MySQL UPSERT 使用 ON DUPLICATE KEY UPDATE
				if err := tx.Exec(`
					INSERT INTO session_states (session_id, scope, `+"`key`"+`, value, created_at, updated_at)
					VALUES (?, ?, ?, ?, ?, ?)
					ON DUPLICATE KEY UPDATE value = ?, updated_at = ?
				`, sessionID, scope, actualKey,
					valueJSON, now, now,
					valueJSON, now).Error; err != nil {
					return fmt.Errorf("upsert state: %w", err)
				}
			}
		}

		// 7. 应用工件变更
		if len(event.Actions.ArtifactDelta) > 0 {
			for name, version := range event.Actions.ArtifactDelta {
				artifactModel := &ArtifactModel{
					ID:        uuid.New().String(),
					SessionID: sessionID,
					Name:      name,
					Version:   int(version),
					CreatedAt: time.Now(),
				}

				if err := tx.Create(artifactModel).Error; err != nil {
					return fmt.Errorf("create artifact: %w", err)
				}
			}
		}

		// 8. 更新 session 的 updated_at
		if err := tx.Model(&SessionModel{}).
			Where("id = ?", sessionID).
			Update("updated_at", time.Now()).Error; err != nil {
			return fmt.Errorf("update session: %w", err)
		}

		return nil
	})
}

// GetEvents 获取会话的所有事件
func (s *Service) GetEvents(ctx context.Context, sessionID string, opts *session.EventOptions) ([]*session.Event, error) {
	var models []EventModel
	query := s.db.WithContext(ctx).Where("session_id = ?", sessionID)

	if opts != nil {
		if opts.InvocationID != "" {
			query = query.Where("invocation_id = ?", opts.InvocationID)
		}
		if opts.Branch != "" {
			query = query.Where("branch = ?", opts.Branch)
		}
		if opts.Limit > 0 {
			query = query.Limit(opts.Limit)
		}
		if opts.Offset > 0 {
			query = query.Offset(opts.Offset)
		}
	}

	query = query.Order("timestamp ASC")

	if err := query.Find(&models).Error; err != nil {
		return nil, fmt.Errorf("get events: %w", err)
	}

	events := make([]*session.Event, len(models))
	for i, model := range models {
		event, err := s.modelToEvent(&model)
		if err != nil {
			return nil, fmt.Errorf("convert event %d: %w", i, err)
		}
		events[i] = event
	}

	return events, nil
}

// GetState 获取会话状态
func (s *Service) GetState(ctx context.Context, sessionID string, scope string) (map[string]any, error) {
	var models []StateModel
	query := s.db.WithContext(ctx).Where("session_id = ?", sessionID)

	if scope != "" {
		query = query.Where("scope = ?", scope)
	}

	if err := query.Find(&models).Error; err != nil {
		return nil, fmt.Errorf("get state: %w", err)
	}

	state := make(map[string]any)
	for _, model := range models {
		var value any
		if err := json.Unmarshal(model.Value, &value); err != nil {
			return nil, fmt.Errorf("unmarshal state value: %w", err)
		}

		fullKey := model.Scope + ":" + model.Key
		state[fullKey] = value
	}

	return state, nil
}

// Close 关闭数据库连接
func (s *Service) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// 辅助方法

func (s *Service) toSession(ctx context.Context, model *SessionModel) *session.SessionData {
	return &session.SessionData{
		ID:             model.ID,
		AppName:        model.AppName,
		UserID:         model.UserID,
		AgentID:        model.AgentID,
		CreatedAt:      model.CreatedAt,
		LastUpdateTime: model.UpdatedAt,
		Metadata:       make(map[string]any),
	}
}

func (s *Service) modelToEvent(model *EventModel) (*session.Event, error) {
	event := &session.Event{
		ID:           model.ID,
		Timestamp:    model.Timestamp,
		InvocationID: model.InvocationID,
		AgentID:      model.AgentID,
		Branch:       model.Branch,
		Author:       model.Author,
	}

	// 反序列化内容
	if err := json.Unmarshal(model.Content, &event.Content); err != nil {
		return nil, fmt.Errorf("unmarshal content: %w", err)
	}

	// 反序列化动作
	if len(model.Actions) > 0 {
		if err := json.Unmarshal(model.Actions, &event.Actions); err != nil {
			return nil, fmt.Errorf("unmarshal actions: %w", err)
		}
	}

	// 反序列化长时运行工具 ID
	if len(model.LongRunningToolIDs) > 0 {
		if err := json.Unmarshal(model.LongRunningToolIDs, &event.LongRunningToolIDs); err != nil {
			return nil, fmt.Errorf("unmarshal long running tools: %w", err)
		}
	}

	// 反序列化元数据
	if len(model.Metadata) > 0 {
		if err := json.Unmarshal(model.Metadata, &event.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}

	return event, nil
}

func parseStateKey(key string) (scope, actualKey string) {
	parts := strings.SplitN(key, ":", 2)
	if len(parts) == 2 {
		scope = parts[0]
		actualKey = parts[1]

		switch scope {
		case session.KeyPrefixApp, session.KeyPrefixUser, session.KeyPrefixTemp:
			return scope, actualKey
		}
	}

	return "session", key
}
