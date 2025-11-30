package auth

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInvalidToken       = errors.New("invalid token")
	ErrExpiredToken       = errors.New("expired token")
)

// AuthMethod 认证方法类型
type AuthMethod string

const (
	AuthMethodAPIKey AuthMethod = "apikey"
	AuthMethodJWT    AuthMethod = "jwt"
	AuthMethodOAuth  AuthMethod = "oauth"
)

// User 用户信息
type User struct {
	ID       string         `json:"id"`
	Username string         `json:"username"`
	Email    string         `json:"email"`
	Roles    []string       `json:"roles"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Authenticator 认证器接口
type Authenticator interface {
	// Authenticate 验证凭证并返回用户信息
	Authenticate(ctx context.Context, credentials any) (*User, error)

	// Validate 验证令牌是否有效
	Validate(ctx context.Context, token string) (*User, error)

	// Method 返回认证方法类型
	Method() AuthMethod
}

// Manager 认证管理器
type Manager struct {
	authenticators map[AuthMethod]Authenticator
	defaultMethod  AuthMethod
}

// NewManager 创建认证管理器
func NewManager(defaultMethod AuthMethod) *Manager {
	return &Manager{
		authenticators: make(map[AuthMethod]Authenticator),
		defaultMethod:  defaultMethod,
	}
}

// Register 注册认证器
func (m *Manager) Register(auth Authenticator) {
	m.authenticators[auth.Method()] = auth
}

// Authenticate 使用指定方法进行认证
func (m *Manager) Authenticate(ctx context.Context, method AuthMethod, credentials any) (*User, error) {
	auth, exists := m.authenticators[method]
	if !exists {
		return nil, errors.New("unsupported authentication method")
	}

	return auth.Authenticate(ctx, credentials)
}

// Validate 验证令牌
func (m *Manager) Validate(ctx context.Context, method AuthMethod, token string) (*User, error) {
	auth, exists := m.authenticators[method]
	if !exists {
		return nil, errors.New("unsupported authentication method")
	}

	return auth.Validate(ctx, token)
}

// GetAuthenticator 获取指定类型的认证器
func (m *Manager) GetAuthenticator(method AuthMethod) (Authenticator, bool) {
	auth, exists := m.authenticators[method]
	return auth, exists
}

// DefaultMethod 返回默认认证方法
func (m *Manager) DefaultMethod() AuthMethod {
	return m.defaultMethod
}

// TokenInfo 令牌信息
type TokenInfo struct {
	Token     string         `json:"token"`
	ExpiresAt time.Time      `json:"expires_at"`
	User      *User          `json:"user,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}
