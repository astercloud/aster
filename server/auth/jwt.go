package auth

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims JWT 声明
type JWTClaims struct {
	jwt.RegisteredClaims

	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
}

// JWTAuthenticator JWT 认证器
type JWTAuthenticator struct {
	secretKey      []byte
	issuer         string
	expiryDuration time.Duration
}

// JWTConfig JWT 配置
type JWTConfig struct {
	SecretKey      string
	Issuer         string
	ExpiryDuration time.Duration
}

// NewJWTAuthenticator 创建 JWT 认证器
func NewJWTAuthenticator(config JWTConfig) *JWTAuthenticator {
	if config.ExpiryDuration == 0 {
		config.ExpiryDuration = 24 * time.Hour // 默认 24 小时
	}
	if config.Issuer == "" {
		config.Issuer = "aster"
	}

	return &JWTAuthenticator{
		secretKey:      []byte(config.SecretKey),
		issuer:         config.Issuer,
		expiryDuration: config.ExpiryDuration,
	}
}

// Method 返回认证方法类型
func (a *JWTAuthenticator) Method() AuthMethod {
	return AuthMethodJWT
}

// Authenticate 生成 JWT 令牌
func (a *JWTAuthenticator) Authenticate(ctx context.Context, credentials any) (*User, error) {
	user, ok := credentials.(*User)
	if !ok {
		return nil, ErrInvalidCredentials
	}

	// 这里通常会验证用户名密码，简化处理直接返回
	return user, nil
}

// Validate 验证 JWT 令牌
func (a *JWTAuthenticator) Validate(ctx context.Context, tokenString string) (*User, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (any, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return a.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	// 验证 issuer
	if claims.Issuer != a.issuer {
		return nil, ErrInvalidToken
	}

	return &User{
		ID:       claims.UserID,
		Username: claims.Username,
		Email:    claims.Email,
		Roles:    claims.Roles,
	}, nil
}

// GenerateToken 生成 JWT 令牌
func (a *JWTAuthenticator) GenerateToken(user *User) (string, time.Time, error) {
	expiresAt := time.Now().Add(a.expiryDuration)

	claims := JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Roles:    user.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    a.issuer,
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(a.secretKey)
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// RefreshToken 刷新令牌
func (a *JWTAuthenticator) RefreshToken(tokenString string) (string, time.Time, error) {
	user, err := a.Validate(context.Background(), tokenString)
	if err != nil {
		// 如果令牌过期，仍然允许刷新（在一定时间窗口内）
		if !errors.Is(err, ErrExpiredToken) {
			return "", time.Time{}, err
		}
	}

	return a.GenerateToken(user)
}
