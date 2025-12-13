package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/astercloud/aster/pkg/types"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// MySQLConfig MySQL 存储配置
type MySQLConfig struct {
	DSN          string        // MySQL DSN
	MaxOpenConns int           // 最大打开连接数
	MaxIdleConns int           // 最大空闲连接数
	MaxLifetime  time.Duration // 连接最大生命周期
}

// MySQLStore MySQL 存储实现
type MySQLStore struct {
	db *gorm.DB
}

// MySQL 数据模型
type AgentMessage struct {
	ID        uint      `gorm:"primaryKey"`
	AgentID   string    `gorm:"index;size:255"`
	Messages  string    `gorm:"type:longtext"` // JSON 存储
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type AgentToolRecord struct {
	ID        uint      `gorm:"primaryKey"`
	AgentID   string    `gorm:"index;size:255"`
	Records   string    `gorm:"type:longtext"` // JSON 存储
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type AgentSnapshot struct {
	ID         uint      `gorm:"primaryKey"`
	AgentID    string    `gorm:"index;size:255"`
	SnapshotID string    `gorm:"index;size:255"`
	Data       string    `gorm:"type:longtext"` // JSON 存储
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}


type AgentInfo struct {
	ID        uint      `gorm:"primaryKey"`
	AgentID   string    `gorm:"uniqueIndex;size:255"`
	Data      string    `gorm:"type:longtext"` // JSON 存储
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type AgentTodo struct {
	ID        uint      `gorm:"primaryKey"`
	AgentID   string    `gorm:"uniqueIndex;size:255"`
	Data      string    `gorm:"type:longtext"` // JSON 存储
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type CollectionItem struct {
	ID         uint      `gorm:"primaryKey"`
	Collection string    `gorm:"index;size:255"`
	Key        string    `gorm:"index;size:255"`
	Data       string    `gorm:"type:longtext"` // JSON 存储
	CreatedAt  time.Time `gorm:"autoCreateTime"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}

func (CollectionItem) TableName() string {
	return "aster_collections"
}

// NewMySQLStore 创建 MySQL 存储
func NewMySQLStore(config MySQLConfig) (*MySQLStore, error) {
	db, err := gorm.Open(mysql.Open(config.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("connect to mysql: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}

	// 配置连接池
	if config.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	} else {
		sqlDB.SetMaxOpenConns(25)
	}
	if config.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	} else {
		sqlDB.SetMaxIdleConns(10)
	}
	if config.MaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(config.MaxLifetime)
	} else {
		sqlDB.SetConnMaxLifetime(5 * time.Minute)
	}

	// 自动迁移表结构
	if err := db.AutoMigrate(
		&AgentMessage{},
		&AgentToolRecord{},
		&AgentSnapshot{},
		&AgentInfo{},
		&AgentTodo{},
		&CollectionItem{},
	); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}

	return &MySQLStore{db: db}, nil
}


// SaveMessages 保存消息列表
func (s *MySQLStore) SaveMessages(ctx context.Context, agentID string, messages []types.Message) error {
	data, err := json.Marshal(messages)
	if err != nil {
		return fmt.Errorf("marshal messages: %w", err)
	}

	record := AgentMessage{AgentID: agentID, Messages: string(data)}
	result := s.db.WithContext(ctx).Where("agent_id = ?", agentID).Assign(record).FirstOrCreate(&record)
	return result.Error
}

// LoadMessages 加载消息列表
func (s *MySQLStore) LoadMessages(ctx context.Context, agentID string) ([]types.Message, error) {
	var record AgentMessage
	if err := s.db.WithContext(ctx).Where("agent_id = ?", agentID).First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return []types.Message{}, nil
		}
		return nil, err
	}

	var messages []types.Message
	if err := json.Unmarshal([]byte(record.Messages), &messages); err != nil {
		return nil, fmt.Errorf("unmarshal messages: %w", err)
	}
	return messages, nil
}

// TrimMessages 修剪消息列表
func (s *MySQLStore) TrimMessages(ctx context.Context, agentID string, maxMessages int) error {
	if maxMessages <= 0 {
		return nil
	}

	messages, err := s.LoadMessages(ctx, agentID)
	if err != nil {
		return err
	}

	if len(messages) <= maxMessages {
		return nil
	}

	trimmed := messages[len(messages)-maxMessages:]
	return s.SaveMessages(ctx, agentID, trimmed)
}

// SaveToolCallRecords 保存工具调用记录
func (s *MySQLStore) SaveToolCallRecords(ctx context.Context, agentID string, records []types.ToolCallRecord) error {
	data, err := json.Marshal(records)
	if err != nil {
		return fmt.Errorf("marshal records: %w", err)
	}

	record := AgentToolRecord{AgentID: agentID, Records: string(data)}
	result := s.db.WithContext(ctx).Where("agent_id = ?", agentID).Assign(record).FirstOrCreate(&record)
	return result.Error
}

// LoadToolCallRecords 加载工具调用记录
func (s *MySQLStore) LoadToolCallRecords(ctx context.Context, agentID string) ([]types.ToolCallRecord, error) {
	var record AgentToolRecord
	if err := s.db.WithContext(ctx).Where("agent_id = ?", agentID).First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return []types.ToolCallRecord{}, nil
		}
		return nil, err
	}

	var records []types.ToolCallRecord
	if err := json.Unmarshal([]byte(record.Records), &records); err != nil {
		return nil, fmt.Errorf("unmarshal records: %w", err)
	}
	return records, nil
}


// SaveSnapshot 保存快照
func (s *MySQLStore) SaveSnapshot(ctx context.Context, agentID string, snapshot types.Snapshot) error {
	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}

	record := AgentSnapshot{AgentID: agentID, SnapshotID: snapshot.ID, Data: string(data)}
	return s.db.WithContext(ctx).Create(&record).Error
}

// LoadSnapshot 加载快照
func (s *MySQLStore) LoadSnapshot(ctx context.Context, agentID string, snapshotID string) (*types.Snapshot, error) {
	var record AgentSnapshot
	if err := s.db.WithContext(ctx).Where("agent_id = ? AND snapshot_id = ?", agentID, snapshotID).First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	var snapshot types.Snapshot
	if err := json.Unmarshal([]byte(record.Data), &snapshot); err != nil {
		return nil, fmt.Errorf("unmarshal snapshot: %w", err)
	}
	return &snapshot, nil
}

// ListSnapshots 列出快照
func (s *MySQLStore) ListSnapshots(ctx context.Context, agentID string) ([]types.Snapshot, error) {
	var records []AgentSnapshot
	if err := s.db.WithContext(ctx).Where("agent_id = ?", agentID).Find(&records).Error; err != nil {
		return nil, err
	}

	snapshots := make([]types.Snapshot, 0, len(records))
	for _, record := range records {
		var snapshot types.Snapshot
		if err := json.Unmarshal([]byte(record.Data), &snapshot); err != nil {
			continue
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, nil
}

// SaveInfo 保存Agent元信息
func (s *MySQLStore) SaveInfo(ctx context.Context, agentID string, info types.AgentInfo) error {
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("marshal info: %w", err)
	}

	record := AgentInfo{AgentID: agentID, Data: string(data)}
	result := s.db.WithContext(ctx).Where("agent_id = ?", agentID).Assign(record).FirstOrCreate(&record)
	return result.Error
}

// LoadInfo 加载Agent元信息
func (s *MySQLStore) LoadInfo(ctx context.Context, agentID string) (*types.AgentInfo, error) {
	var record AgentInfo
	if err := s.db.WithContext(ctx).Where("agent_id = ?", agentID).First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	var info types.AgentInfo
	if err := json.Unmarshal([]byte(record.Data), &info); err != nil {
		return nil, fmt.Errorf("unmarshal info: %w", err)
	}
	return &info, nil
}

// SaveTodos 保存Todo列表
func (s *MySQLStore) SaveTodos(ctx context.Context, agentID string, todos any) error {
	data, err := json.Marshal(todos)
	if err != nil {
		return fmt.Errorf("marshal todos: %w", err)
	}

	record := AgentTodo{AgentID: agentID, Data: string(data)}
	result := s.db.WithContext(ctx).Where("agent_id = ?", agentID).Assign(record).FirstOrCreate(&record)
	return result.Error
}

// LoadTodos 加载Todo列表
func (s *MySQLStore) LoadTodos(ctx context.Context, agentID string) (any, error) {
	var record AgentTodo
	if err := s.db.WithContext(ctx).Where("agent_id = ?", agentID).First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	var todos any
	if err := json.Unmarshal([]byte(record.Data), &todos); err != nil {
		return nil, fmt.Errorf("unmarshal todos: %w", err)
	}
	return todos, nil
}


// DeleteAgent 删除Agent所有数据
func (s *MySQLStore) DeleteAgent(ctx context.Context, agentID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("agent_id = ?", agentID).Delete(&AgentMessage{}).Error; err != nil {
			return err
		}
		if err := tx.Where("agent_id = ?", agentID).Delete(&AgentToolRecord{}).Error; err != nil {
			return err
		}
		if err := tx.Where("agent_id = ?", agentID).Delete(&AgentSnapshot{}).Error; err != nil {
			return err
		}
		if err := tx.Where("agent_id = ?", agentID).Delete(&AgentInfo{}).Error; err != nil {
			return err
		}
		if err := tx.Where("agent_id = ?", agentID).Delete(&AgentTodo{}).Error; err != nil {
			return err
		}
		return nil
	})
}

// ListAgents 列出所有Agent
func (s *MySQLStore) ListAgents(ctx context.Context) ([]string, error) {
	var infos []AgentInfo
	if err := s.db.WithContext(ctx).Select("agent_id").Find(&infos).Error; err != nil {
		return nil, err
	}

	agents := make([]string, 0, len(infos))
	for _, info := range infos {
		agents = append(agents, info.AgentID)
	}
	return agents, nil
}

// --- 通用 CRUD 方法 ---

// Get 获取单个资源
func (s *MySQLStore) Get(ctx context.Context, collection, key string, dest any) error {
	var item CollectionItem
	if err := s.db.WithContext(ctx).Where("collection = ? AND `key` = ?", collection, key).First(&item).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrNotFound
		}
		return err
	}

	if err := json.Unmarshal([]byte(item.Data), dest); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return nil
}

// Set 设置资源
func (s *MySQLStore) Set(ctx context.Context, collection, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	item := CollectionItem{Collection: collection, Key: key, Data: string(data)}
	result := s.db.WithContext(ctx).Where("collection = ? AND `key` = ?", collection, key).Assign(item).FirstOrCreate(&item)
	return result.Error
}

// Delete 删除资源
func (s *MySQLStore) Delete(ctx context.Context, collection, key string) error {
	result := s.db.WithContext(ctx).Where("collection = ? AND `key` = ?", collection, key).Delete(&CollectionItem{})
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return result.Error
}

// List 列出资源
func (s *MySQLStore) List(ctx context.Context, collection string) ([]any, error) {
	var items []CollectionItem
	if err := s.db.WithContext(ctx).Where("collection = ?", collection).Find(&items).Error; err != nil {
		return nil, err
	}

	result := make([]any, 0, len(items))
	for _, item := range items {
		var data any
		if err := json.Unmarshal([]byte(item.Data), &data); err != nil {
			continue
		}
		result = append(result, data)
	}
	return result, nil
}

// Exists 检查资源是否存在
func (s *MySQLStore) Exists(ctx context.Context, collection, key string) (bool, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&CollectionItem{}).Where("collection = ? AND `key` = ?", collection, key).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// Close 关闭数据库连接
func (s *MySQLStore) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
