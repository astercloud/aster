package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/astercloud/aster/pkg/types"
	"github.com/redis/go-redis/v9"
)

// RedisStore Redis 持久化存储
// 适用于分布式环境，支持多节点共享状态
type RedisStore struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr     string        // Redis 地址，格式: "host:port"
	Password string        // 密码
	DB       int           // 数据库编号 (0-15)
	Prefix   string        // Key 前缀，默认 "aster:"
	TTL      time.Duration // 数据过期时间，默认 7 天
}

// NewRedisStore 创建 Redis Store
func NewRedisStore(config RedisConfig) (*RedisStore, error) {
	if config.Addr == "" {
		return nil, fmt.Errorf("redis addr is required")
	}

	client := redis.NewClient(&redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     100,
		MinIdleConns: 10,
		MaxRetries:   3,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	prefix := config.Prefix
	if prefix == "" {
		prefix = "aster:"
	}

	ttl := config.TTL
	if ttl == 0 {
		ttl = 7 * 24 * time.Hour // 默认 7 天
	}

	return &RedisStore{
		client: client,
		prefix: prefix,
		ttl:    ttl,
	}, nil
}

// SaveMessages 保存消息列表
func (rs *RedisStore) SaveMessages(ctx context.Context, agentID string, messages []types.Message) error {
	key := rs.prefix + "messages:" + agentID

	data, err := json.Marshal(messages)
	if err != nil {
		return fmt.Errorf("marshal messages: %w", err)
	}

	if err := rs.client.Set(ctx, key, data, rs.ttl).Err(); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}

	return nil
}

// LoadMessages 加载消息列表
func (rs *RedisStore) LoadMessages(ctx context.Context, agentID string) ([]types.Message, error) {
	key := rs.prefix + "messages:" + agentID

	data, err := rs.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return []types.Message{}, nil // 未找到返回空列表
	}
	if err != nil {
		return nil, fmt.Errorf("redis get: %w", err)
	}

	var messages []types.Message
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, fmt.Errorf("unmarshal messages: %w", err)
	}

	return messages, nil
}

// TrimMessages 修剪消息列表（原子操作）
func (rs *RedisStore) TrimMessages(ctx context.Context, agentID string, maxMessages int) error {
	if maxMessages <= 0 {
		return nil
	}

	key := rs.prefix + "messages:" + agentID

	// 使用 Watch + Transaction 实现原子操作
	return rs.client.Watch(ctx, func(tx *redis.Tx) error {
		// 读取当前消息
		data, err := tx.Get(ctx, key).Bytes()
		if err == redis.Nil {
			return nil // 不存在，无需修剪
		}
		if err != nil {
			return err
		}

		var messages []types.Message
		if err := json.Unmarshal(data, &messages); err != nil {
			return err
		}

		// 检查是否需要修剪
		if len(messages) <= maxMessages {
			return nil
		}

		// FIFO 修剪：保留最近的 N 条
		trimmed := messages[len(messages)-maxMessages:]
		newData, err := json.Marshal(trimmed)
		if err != nil {
			return err
		}

		// 原子更新
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, key, newData, rs.ttl)
			return nil
		})
		return err
	}, key)
}

// SaveToolCallRecords 保存工具调用记录
func (rs *RedisStore) SaveToolCallRecords(ctx context.Context, agentID string, records []types.ToolCallRecord) error {
	key := rs.prefix + "tools:" + agentID

	data, err := json.Marshal(records)
	if err != nil {
		return fmt.Errorf("marshal tool records: %w", err)
	}

	return rs.client.Set(ctx, key, data, rs.ttl).Err()
}

// LoadToolCallRecords 加载工具调用记录
func (rs *RedisStore) LoadToolCallRecords(ctx context.Context, agentID string) ([]types.ToolCallRecord, error) {
	key := rs.prefix + "tools:" + agentID

	data, err := rs.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return []types.ToolCallRecord{}, nil
	}
	if err != nil {
		return nil, err
	}

	var records []types.ToolCallRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}

	return records, nil
}

// SaveSnapshot 保存快照
func (rs *RedisStore) SaveSnapshot(ctx context.Context, agentID string, snapshot types.Snapshot) error {
	key := rs.prefix + "snapshot:" + agentID + ":" + snapshot.ID

	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}

	return rs.client.Set(ctx, key, data, rs.ttl).Err()
}

// LoadSnapshot 加载快照
func (rs *RedisStore) LoadSnapshot(ctx context.Context, agentID string, snapshotID string) (*types.Snapshot, error) {
	key := rs.prefix + "snapshot:" + agentID + ":" + snapshotID

	data, err := rs.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var snapshot types.Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, err
	}

	return &snapshot, nil
}

// ListSnapshots 列出快照
func (rs *RedisStore) ListSnapshots(ctx context.Context, agentID string) ([]types.Snapshot, error) {
	pattern := rs.prefix + "snapshot:" + agentID + ":*"

	var snapshots []types.Snapshot
	iter := rs.client.Scan(ctx, 0, pattern, 100).Iterator()

	for iter.Next(ctx) {
		key := iter.Val()
		data, err := rs.client.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}

		var snapshot types.Snapshot
		if err := json.Unmarshal(data, &snapshot); err != nil {
			continue
		}

		snapshots = append(snapshots, snapshot)
	}

	return snapshots, iter.Err()
}

// SaveInfo 保存 Agent 元信息
func (rs *RedisStore) SaveInfo(ctx context.Context, agentID string, info types.AgentInfo) error {
	key := rs.prefix + "info:" + agentID

	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("marshal info: %w", err)
	}

	return rs.client.Set(ctx, key, data, rs.ttl).Err()
}

// LoadInfo 加载 Agent 元信息
func (rs *RedisStore) LoadInfo(ctx context.Context, agentID string) (*types.AgentInfo, error) {
	key := rs.prefix + "info:" + agentID

	data, err := rs.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var info types.AgentInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// SaveTodos 保存 Todo 列表
func (rs *RedisStore) SaveTodos(ctx context.Context, agentID string, todos any) error {
	key := rs.prefix + "todos:" + agentID

	data, err := json.Marshal(todos)
	if err != nil {
		return fmt.Errorf("marshal todos: %w", err)
	}

	return rs.client.Set(ctx, key, data, rs.ttl).Err()
}

// LoadTodos 加载 Todo 列表
func (rs *RedisStore) LoadTodos(ctx context.Context, agentID string) (any, error) {
	key := rs.prefix + "todos:" + agentID

	data, err := rs.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var todos any
	if err := json.Unmarshal(data, &todos); err != nil {
		return nil, err
	}

	return todos, nil
}

// DeleteAgent 删除 Agent 所有数据
func (rs *RedisStore) DeleteAgent(ctx context.Context, agentID string) error {
	pattern := rs.prefix + "*:" + agentID + "*"

	var keys []string
	iter := rs.client.Scan(ctx, 0, pattern, 1000).Iterator()

	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return err
	}

	if len(keys) > 0 {
		return rs.client.Del(ctx, keys...).Err()
	}

	return nil
}

// ListAgents 列出所有 Agent
func (rs *RedisStore) ListAgents(ctx context.Context) ([]string, error) {
	pattern := rs.prefix + "info:*"

	var agents []string
	iter := rs.client.Scan(ctx, 0, pattern, 1000).Iterator()

	for iter.Next(ctx) {
		key := iter.Val()
		// 提取 agentID: "aster:info:agt-xxx" -> "agt-xxx"
		agentID := key[len(rs.prefix)+len("info:"):]
		agents = append(agents, agentID)
	}

	return agents, iter.Err()
}

// Get 获取单个资源
func (rs *RedisStore) Get(ctx context.Context, collection, key string, dest any) error {
	redisKey := rs.prefix + collection + ":" + key

	data, err := rs.client.Get(ctx, redisKey).Bytes()
	if err == redis.Nil {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	return json.Unmarshal(data, dest)
}

// Set 设置资源
func (rs *RedisStore) Set(ctx context.Context, collection, key string, value any) error {
	redisKey := rs.prefix + collection + ":" + key

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return rs.client.Set(ctx, redisKey, data, rs.ttl).Err()
}

// Delete 删除资源
func (rs *RedisStore) Delete(ctx context.Context, collection, key string) error {
	redisKey := rs.prefix + collection + ":" + key
	return rs.client.Del(ctx, redisKey).Err()
}

// List 列出资源
func (rs *RedisStore) List(ctx context.Context, collection string) ([]any, error) {
	pattern := rs.prefix + collection + ":*"

	var results []any
	iter := rs.client.Scan(ctx, 0, pattern, 1000).Iterator()

	for iter.Next(ctx) {
		key := iter.Val()
		data, err := rs.client.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}

		var item any
		if err := json.Unmarshal(data, &item); err != nil {
			continue
		}

		results = append(results, item)
	}

	return results, iter.Err()
}

// Exists 检查资源是否存在
func (rs *RedisStore) Exists(ctx context.Context, collection, key string) (bool, error) {
	redisKey := rs.prefix + collection + ":" + key

	count, err := rs.client.Exists(ctx, redisKey).Result()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// Close 关闭 Redis 连接
func (rs *RedisStore) Close() error {
	return rs.client.Close()
}

// Ping 检查 Redis 连接
func (rs *RedisStore) Ping(ctx context.Context) error {
	return rs.client.Ping(ctx).Err()
}
