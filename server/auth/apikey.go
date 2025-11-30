package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// APIKeyStore API Key 存储接口
type APIKeyStore interface {
	// Get 获取 API Key 信息
	Get(ctx context.Context, key string) (*APIKeyInfo, error)

	// Create 创建新的 API Key
	Create(ctx context.Context, info *APIKeyInfo) error

	// Delete 删除 API Key
	Delete(ctx context.Context, key string) error

	// List 列出用户的所有 API Keys
	List(ctx context.Context, userID string) ([]*APIKeyInfo, error)
}

// APIKeyInfo API Key 信息
type APIKeyInfo struct {
	Key       string         `json:"key"`
	UserID    string         `json:"user_id"`
	Name      string         `json:"name"`
	Roles     []string       `json:"roles"`
	ExpiresAt *time.Time     `json:"expires_at,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	LastUsed  *time.Time     `json:"last_used,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// APIKeyAuthenticator API Key 认证器
type APIKeyAuthenticator struct {
	store APIKeyStore
}

// NewAPIKeyAuthenticator 创建 API Key 认证器
func NewAPIKeyAuthenticator(store APIKeyStore) *APIKeyAuthenticator {
	return &APIKeyAuthenticator{
		store: store,
	}
}

// Method 返回认证方法类型
func (a *APIKeyAuthenticator) Method() AuthMethod {
	return AuthMethodAPIKey
}

// Authenticate 验证 API Key
func (a *APIKeyAuthenticator) Authenticate(ctx context.Context, credentials any) (*User, error) {
	key, ok := credentials.(string)
	if !ok {
		return nil, ErrInvalidCredentials
	}

	return a.Validate(ctx, key)
}

// Validate 验证 API Key 并返回用户信息
func (a *APIKeyAuthenticator) Validate(ctx context.Context, key string) (*User, error) {
	if key == "" {
		return nil, ErrInvalidCredentials
	}

	info, err := a.store.Get(ctx, key)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// 检查是否过期
	if info.ExpiresAt != nil && time.Now().After(*info.ExpiresAt) {
		return nil, ErrExpiredToken
	}

	// 更新最后使用时间
	now := time.Now()
	info.LastUsed = &now
	// 这里可以异步更新，不阻塞请求
	go func() {
		_ = a.store.Create(context.Background(), info)
	}()

	return &User{
		ID:    info.UserID,
		Roles: info.Roles,
		Metadata: map[string]any{
			"api_key_name": info.Name,
		},
	}, nil
}

// GenerateAPIKey 生成新的 API Key
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "sk_" + hex.EncodeToString(bytes), nil
}

// MemoryAPIKeyStore 内存 API Key 存储（用于测试和开发）
type MemoryAPIKeyStore struct {
	mu   sync.RWMutex
	keys map[string]*APIKeyInfo
}

// NewMemoryAPIKeyStore 创建内存存储
func NewMemoryAPIKeyStore() *MemoryAPIKeyStore {
	return &MemoryAPIKeyStore{
		keys: make(map[string]*APIKeyInfo),
	}
}

// Get 获取 API Key 信息
func (s *MemoryAPIKeyStore) Get(ctx context.Context, key string) (*APIKeyInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	info, exists := s.keys[key]
	if !exists {
		return nil, fmt.Errorf("api key not found")
	}

	return info, nil
}

// Create 创建或更新 API Key
func (s *MemoryAPIKeyStore) Create(ctx context.Context, info *APIKeyInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.keys[info.Key] = info
	return nil
}

// Delete 删除 API Key
func (s *MemoryAPIKeyStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.keys, key)
	return nil
}

// List 列出用户的所有 API Keys
func (s *MemoryAPIKeyStore) List(ctx context.Context, userID string) ([]*APIKeyInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*APIKeyInfo
	for _, info := range s.keys {
		if info.UserID == userID {
			result = append(result, info)
		}
	}

	return result, nil
}
