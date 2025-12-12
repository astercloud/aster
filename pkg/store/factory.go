package store

import (
	"fmt"
	"time"
)

// StoreType Store 类型
type StoreType string

const (
	StoreTypeJSON  StoreType = "json"
	StoreTypeRedis StoreType = "redis"
)

// Config Store 配置
type Config struct {
	Type StoreType `json:"type" yaml:"type"` // Store 类型: json, redis

	// JSON Store 配置
	DataDir string `json:"data_dir,omitempty" yaml:"data_dir,omitempty"` // 数据目录

	// Redis Store 配置
	RedisAddr     string        `json:"redis_addr,omitempty" yaml:"redis_addr,omitempty"`         // Redis 地址
	RedisPassword string        `json:"redis_password,omitempty" yaml:"redis_password,omitempty"` // Redis 密码
	RedisDB       int           `json:"redis_db,omitempty" yaml:"redis_db,omitempty"`             // Redis 数据库
	RedisPrefix   string        `json:"redis_prefix,omitempty" yaml:"redis_prefix,omitempty"`     // Redis Key 前缀
	RedisTTL      time.Duration `json:"redis_ttl,omitempty" yaml:"redis_ttl,omitempty"`           // Redis 数据过期时间
}

// NewStore 创建 Store（工厂方法）
func NewStore(config Config) (Store, error) {
	switch config.Type {
	case StoreTypeJSON, "":
		// 默认使用 JSON Store
		dataDir := config.DataDir
		if dataDir == "" {
			dataDir = ".aster"
		}
		return NewJSONStore(dataDir)

	case StoreTypeRedis:
		if config.RedisAddr == "" {
			return nil, fmt.Errorf("redis_addr is required for redis store")
		}

		redisConfig := RedisConfig{
			Addr:     config.RedisAddr,
			Password: config.RedisPassword,
			DB:       config.RedisDB,
			Prefix:   config.RedisPrefix,
			TTL:      config.RedisTTL,
		}

		return NewRedisStore(redisConfig)

	default:
		return nil, fmt.Errorf("unknown store type: %s", config.Type)
	}
}

// MustNewStore 创建 Store，失败时 panic
func MustNewStore(config Config) Store {
	s, err := NewStore(config)
	if err != nil {
		panic(fmt.Sprintf("failed to create store: %v", err))
	}
	return s
}
