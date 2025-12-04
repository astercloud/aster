# 认证与授权

aster Server 提供完整的认证和授权系统，支持 API Key、JWT 和 RBAC 权限控制。

## 认证方法

### API Key 认证

最简单的认证方式，适用于服务间通信和 API 集成。

#### 配置

```go
config := &server.Config{
    Auth: server.AuthConfig{
        APIKey: server.APIKeyConfig{
            Enabled: true,
            Keys: []string{
                "sk_abc123...",  // 预定义的 key
            },
        },
    },
}
```

#### 使用

```bash
# Header 方式
curl -H "X-API-Key: sk_abc123..." http://localhost:8080/v1/agents

# Bearer Token 方式
curl -H "Authorization: Bearer sk_abc123..." http://localhost:8080/v1/agents

# Query 参数方式
curl http://localhost:8080/v1/agents?api_key=sk_abc123...
```

#### 生成 API Key

```go
import "github.com/astercloud/aster/server/auth"

// 生成新的 API Key
key, _ := auth.GenerateAPIKey()
// 返回: sk_a1b2c3d4e5f6...

// 存储 API Key
apiKeyStore := auth.NewMemoryAPIKeyStore()
apiKeyStore.Create(ctx, &auth.APIKeyInfo{
    Key: key,
    UserID: "user123",
    Roles: []string{"user"},
    ExpiresAt: &expiryTime,
})
```

### JWT 认证

基于令牌的认证，支持无状态会话和细粒度控制。

#### 配置

```go
config := &server.Config{
    Auth: server.AuthConfig{
        JWT: server.JWTConfig{
            Enabled: true,
            Secret: "your-jwt-secret-key",
            Issuer: "aster",
            Expiry: 86400, // 24 hours
        },
    },
}
```

#### 生成 Token

```go
jwtAuth := auth.NewJWTAuthenticator(auth.JWTConfig{
    SecretKey: "your-secret",
    Issuer: "aster",
    ExpiryDuration: 24 * time.Hour,
})

user := &auth.User{
    ID: "user123",
    Username: "john",
    Email: "john@example.com",
    Roles: []string{"user"},
}

token, expiresAt, _ := jwtAuth.GenerateToken(user)
// 返回: eyJhbGciOiJIUzI1NiIs...
```

#### 使用

```bash
curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  http://localhost:8080/v1/agents
```

#### 刷新 Token

```go
newToken, expiresAt, _ := jwtAuth.RefreshToken(oldToken)
```

### 混合认证

同时支持多种认证方法：

```go
config := &server.Config{
    Auth: server.AuthConfig{
        APIKey: server.APIKeyConfig{Enabled: true, ...},
        JWT: server.JWTConfig{Enabled: true, ...},
    },
}

// 请求会依次尝试：
// 1. API Key 认证
// 2. JWT 认证
```

## RBAC 权限控制

基于角色的访问控制，支持细粒度权限管理。

### 预定义角色

#### Admin - 管理员

```go
{
    Name: "admin",
    Permissions: []Permission{
        {Resource: "*", Actions: []string{"*"}},
    },
}
// 完全权限
```

#### User - 普通用户

```go
{
    Name: "user",
    Permissions: []Permission{
        {Resource: "agents", Actions: []string{"create", "read", "update", "delete"}},
        {Resource: "sessions", Actions: []string{"create", "read", "update", "delete"}},
        {Resource: "workflows", Actions: []string{"read", "execute"}},
    },
}
// 基础 CRUD + 工作流执行
```

#### Viewer - 查看者

```go
{
    Name: "viewer",
    Permissions: []Permission{
        {Resource: "*", Actions: []string{"read"}},
    },
}
// 只读权限
```

#### Developer - 开发者

```go
{
    Name: "developer",
    Permissions: []Permission{
        {Resource: "agents", Actions: []string{"*"}},
        {Resource: "workflows", Actions: []string{"*"}},
        {Resource: "tools", Actions: []string{"*"}},
        {Resource: "eval", Actions: []string{"*"}},
    },
}
// 开发相关完全权限
```

### 自定义角色

```go
rbac := auth.NewRBAC()

// 添加自定义角色
customRole := &auth.Role{
    Name: "analyst",
    Description: "Data analyst with read and eval permissions",
    Permissions: []auth.Permission{
        {Resource: "agents", Actions: []string{"read"}},
        {Resource: "sessions", Actions: []string{"read"}},
        {Resource: "eval", Actions: []string{"*"}},
    },
}
rbac.AddRole(customRole)
```

### 权限检查

#### 在 Handler 中检查

```go
func (h *AgentHandler) Delete(c *gin.Context) {
    user := getUserFromContext(c)  // 从 context 获取用户

    // 检查权限
    if !h.rbac.HasPermission(c.Request.Context(), user, "agents", "delete") {
        c.JSON(http.StatusForbidden, gin.H{
            "error": "Permission denied",
        })
        return
    }

    // 执行删除...
}
```

#### 使用中间件

```go
func RequirePermission(rbac *auth.RBAC, resource, action string) gin.HandlerFunc {
    return func(c *gin.Context) {
        user := getUserFromContext(c)

        if !rbac.HasPermission(c.Request.Context(), user, resource, action) {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                "error": "Permission denied",
            })
            return
        }

        c.Next()
    }
}

// 使用
router.DELETE("/agents/:id",
    RequirePermission(rbac, "agents", "delete"),
    h.Delete,
)
```

### 查询用户权限

```go
permissions := rbac.GetUserPermissions(user)
// 返回: []Permission{
//     {Resource: "agents", Actions: ["create", "read", "update"]},
//     {Resource: "sessions", Actions: ["read"]},
// }
```

## 最佳实践

### 1. API Key 管理

**✅ 推荐**:

- 使用环境变量存储 API Keys
- 定期轮换 keys
- 设置过期时间
- 使用 `sk_` 前缀标识
- 记录 key 使用情况

```go
apiKeyInfo := &auth.APIKeyInfo{
    Key: "sk_xxx",
    UserID: "user123",
    Name: "Production API Key",
    Roles: []string{"user"},
    ExpiresAt: &expiryTime,
    Metadata: map[string]interface{}{
        "environment": "production",
        "created_by": "admin",
    },
}
```

**❌ 避免**:

- 在代码中硬编码 keys
- 使用简单的 keys
- 永不过期的 keys
- 共享 keys

### 2. JWT 配置

**✅ 推荐**:

- 使用强密钥 (>= 32 字符)
- 设置合理的过期时间
- 包含必要的 claims
- 验证 issuer 和 audience

```go
jwtConfig := auth.JWTConfig{
    SecretKey: os.Getenv("JWT_SECRET"),  // 从环境变量
    Issuer: "aster",
    ExpiryDuration: 24 * time.Hour,      // 24 小时
}
```

**❌ 避免**:

- 弱密钥
- 过长的过期时间 (> 7 天)
- 在 token 中存储敏感信息
- 忽略 token 验证

### 3. RBAC 设计

**✅ 推荐**:

- 遵循最小权限原则
- 使用角色而非直接权限
- 定期审计权限
- 文档化角色和权限

```go
// 按职能划分角色
roles := []string{
    "admin",      // 系统管理员
    "developer",  // 开发者
    "analyst",    // 数据分析师
    "viewer",     // 只读用户
}
```

**❌ 避免**:

- 过度授权
- 复杂的权限层级
- 未文档化的权限
- 直接授予 admin 权限

### 4. 安全传输

**✅ 推荐**:

- 使用 HTTPS
- 启用 HSTS
- 验证 SSL 证书

```go
config := &server.Config{
    TLS: server.TLSConfig{
        Enabled: true,
        CertFile: "/path/to/cert.pem",
        KeyFile: "/path/to/key.pem",
    },
}
```

## 实战示例

### 完整认证流程

```go
package main

import (
    "github.com/astercloud/aster/server"
    "github.com/astercloud/aster/server/auth"
)

func main() {
    // 1. 创建认证管理器
    authManager := auth.NewManager(auth.AuthMethodAPIKey)

    // 2. 注册 API Key 认证
    apiKeyStore := auth.NewMemoryAPIKeyStore()
    apiKeyAuth := auth.NewAPIKeyAuthenticator(apiKeyStore)
    authManager.Register(apiKeyAuth)

    // 3. 注册 JWT 认证
    jwtAuth := auth.NewJWTAuthenticator(auth.JWTConfig{
        SecretKey: os.Getenv("JWT_SECRET"),
        Issuer: "aster",
        ExpiryDuration: 24 * time.Hour,
    })
    authManager.Register(jwtAuth)

    // 4. 创建 RBAC
    rbac := auth.NewRBAC()

    // 5. 配置服务器
    config := &server.Config{
        Auth: server.AuthConfig{
            APIKey: server.APIKeyConfig{Enabled: true},
            JWT: server.JWTConfig{Enabled: true},
        },
    }

    srv, _ := server.New(config, deps)
    srv.Start()
}
```

### 登录端点

```go
func (h *AuthHandler) Login(c *gin.Context) {
    var req struct {
        Username string `json:"username"`
        Password string `json:"password"`
    }

    c.ShouldBindJSON(&req)

    // 验证用户名密码
    user, err := validateCredentials(req.Username, req.Password)
    if err != nil {
        c.JSON(401, gin.H{"error": "Invalid credentials"})
        return
    }

    // 生成 JWT
    token, expiresAt, _ := h.jwtAuth.GenerateToken(user)

    c.JSON(200, gin.H{
        "token": token,
        "expires_at": expiresAt,
        "user": user,
    })
}
```

### 权限中间件

```go
func AuthMiddleware(authManager *auth.Manager, rbac *auth.RBAC) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 提取 token
        token := extractToken(c)
        if token == "" {
            c.AbortWithStatusJSON(401, gin.H{"error": "Missing token"})
            return
        }

        // 验证 token
        user, err := authManager.Validate(c.Request.Context(), auth.AuthMethodJWT, token)
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{"error": "Invalid token"})
            return
        }

        // 存储用户信息
        c.Set("user", user)
        c.Next()
    }
}
```

## 相关资源

- [JWT 规范](https://jwt.io/)
- [OAuth 2.0](https://oauth.net/2/)
- [RBAC 模型](https://en.wikipedia.org/wiki/Role-based_access_control)
- [API Key 最佳实践](https://cloud.google.com/endpoints/docs/openapi/when-why-api-key)
