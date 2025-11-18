# Server 架构

aster Server 是一个生产级的 HTTP 服务器实现，提供完整的认证、授权、可观测性和速率限制功能。

## 架构概览

```
┌─────────────────────────────────────────────────────────┐
│                    aster Server                       │
├─────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────┐ │
│  │   Handlers  │  │     Auth     │  │ Observability  │ │
│  │  (8 core)   │  │   Manager    │  │   (Metrics)    │ │
│  └─────────────┘  └──────────────┘  └────────────────┘ │
│                                                          │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────┐ │
│  │   Routing   │  │     RBAC     │  │    Tracing     │ │
│  │  (Gin-based)│  │   Control    │  │ (OpenTelemetry)│ │
│  └─────────────┘  └──────────────┘  └────────────────┘ │
│                                                          │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────┐ │
│  │ Middleware  │  │ Rate Limiter │  │  Health Check  │ │
│  │   Stack     │  │ (Token Bucket)│ │   (Enhanced)   │ │
│  └─────────────┘  └──────────────┘  └────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

## 核心组件

### 1. Server 核心 (`server/server.go`)

主服务器类，管理所有组件的生命周期。

```go
type Server struct {
    config *Config
    router *gin.Engine
    store  store.Store
    
    // Auth & Observability
    authManager   *auth.Manager
    rbac          *auth.RBAC
    metrics       *observability.MetricsManager
    healthChecker *observability.HealthChecker
    tracing       *observability.TracingManager
    rateLimiter   ratelimit.Limiter
}
```

**功能**:
- 服务器生命周期管理
- 中间件配置
- 路由注册
- 优雅关闭

### 2. 认证系统 (`server/auth/`)

完整的认证和授权系统。

#### 认证管理器
```go
// 支持多种认证方法
authManager := auth.NewManager(auth.AuthMethodAPIKey)

// API Key 认证
apiKeyAuth := auth.NewAPIKeyAuthenticator(store)
authManager.Register(apiKeyAuth)

// JWT 认证
jwtAuth := auth.NewJWTAuthenticator(auth.JWTConfig{
    SecretKey: "your-secret",
    Issuer: "aster",
    ExpiryDuration: 24 * time.Hour,
})
authManager.Register(jwtAuth)
```

#### RBAC 权限控制
```go
rbac := auth.NewRBAC()

// 检查权限
hasPermission := rbac.HasPermission(ctx, user, "agents", "create")

// 预定义角色
// - admin: 完全权限
// - user: 基础 CRUD
// - viewer: 只读
// - developer: 开发者权限
```

### 3. 可观测性 (`server/observability/`)

#### Prometheus Metrics
```go
metrics := observability.NewMetricsManager("aster")

// HTTP 指标
aster_http_requests_total{method,path,status}
aster_http_request_duration_seconds{method,path}

// 业务指标
aster_agents_total
aster_sessions_active
aster_workflows_running
```

#### OpenTelemetry 追踪
```go
tracing, _ := observability.NewTracingManager(observability.TracingConfig{
    Enabled: true,
    ServiceName: "aster",
    OTLPEndpoint: "localhost:4318",
    SamplingRate: 1.0,
})

// 自动追踪 HTTP 请求
// 支持 Jaeger, Zipkin, OTLP
```

#### 增强健康检查
```go
healthChecker := observability.NewHealthChecker("v0.11.0")

// 注册自定义检查
storeCheck := observability.NewStoreHealthCheck("store", checkFunc)
healthChecker.RegisterCheck(storeCheck)

// 响应包含详细信息
{
  "status": "healthy",
  "uptime": "2h30m",
  "checks": {
    "store": {"status": "healthy", "latency": "5ms"}
  }
}
```

### 4. 速率限制 (`server/ratelimit/`)

支持两种算法：

#### 令牌桶 (Token Bucket)
```go
limiter := ratelimit.NewTokenBucketLimiter(
    rate,     // 令牌补充速率
    capacity, // 桶容量
    window,   // 清理窗口
)
```

#### 滑动窗口 (Sliding Window)
```go
limiter := ratelimit.NewSlidingWindowLimiter(
    limit,  // 请求限制
    window, // 时间窗口
)
```

#### 中间件集成
```go
// 基于 IP 限流
ratelimit.Middleware(config, limiter)

// 基于用户限流
ratelimit.PerUserMiddleware(config, limiter)

// 基于端点限流
ratelimit.PerEndpointMiddleware(config, limiter)
```

### 5. Handlers (`server/handlers/`)

8 个核心业务 Handler：

- `agent.go` - Agent 管理
- `memory.go` - 内存管理
- `session.go` - 会话管理
- `workflow.go` - 工作流
- `tool.go` - 工具管理
- `telemetry.go` - 遥测
- `eval.go` - 评估
- `mcp.go` - MCP 服务器

所有 Handler 使用统一模式：
```go
type Handler struct {
    store *store.Store
}

func NewHandler(st store.Store) *Handler {
    return &Handler{store: &st}
}

func (h *Handler) Create(c *gin.Context) {
    // 实现...
}
```

## 中间件栈

请求经过以下中间件（按顺序）：

1. **Recovery** - Panic 恢复
2. **Request ID** - 请求追踪
3. **Tracing** - 分布式追踪
4. **Logging** - 结构化日志
5. **CORS** - 跨域处理
6. **Metrics** - 指标收集
7. **Auth** - 认证验证
8. **Rate Limit** - 速率限制

## 配置

### 完整配置示例

```go
config := &server.Config{
    Host: "0.0.0.0",
    Port: 8080,
    Mode: "production",
    
    // 认证
    Auth: server.AuthConfig{
        APIKey: server.APIKeyConfig{
            Enabled: true,
            Keys: []string{"your-api-key"},
        },
        JWT: server.JWTConfig{
            Enabled: true,
            Secret: "your-jwt-secret",
            Expiry: 86400, // 24 hours
        },
    },
    
    // CORS
    CORS: server.CORSConfig{
        Enabled: true,
        AllowOrigins: []string{"*"},
        AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
    },
    
    // 速率限制
    RateLimit: server.RateLimitConfig{
        Enabled: true,
        RequestsPerIP: 100,
        WindowSize: time.Minute,
        BurstSize: 20,
    },
    
    // 可观测性
    Observability: server.ObservabilityConfig{
        Enabled: true,
        Metrics: server.MetricsConfig{
            Enabled: true,
            Endpoint: "/metrics",
        },
        Tracing: server.TracingConfig{
            Enabled: true,
            ServiceName: "aster",
            OTLPEndpoint: "localhost:4318",
            SamplingRate: 1.0,
        },
        HealthCheck: server.HealthCheckConfig{
            Enabled: true,
            Endpoint: "/health",
        },
    },
}
```

## 使用示例

### 基础启动

```go
package main

import (
    "github.com/astercloud/aster/pkg/store"
    "github.com/astercloud/aster/server"
)

func main() {
    // 创建依赖
    st, _ := store.NewJSONStore(".aster")
    deps := &server.Dependencies{
        Store: st,
    }
    
    // 创建服务器
    srv, _ := server.New(server.DefaultConfig(), deps)
    
    // 启动
    srv.Start()
}
```

### 自定义配置

```go
config := server.DefaultConfig()
config.Port = 3000
config.Auth.APIKey.Enabled = true
config.Auth.APIKey.Keys = []string{"my-secret-key"}

srv, _ := server.New(config, deps)
srv.Start()
```

### 优雅关闭

```go
// 监听信号
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

go func() {
    if err := srv.Start(); err != nil {
        log.Fatal(err)
    }
}()

<-sigChan

// 优雅关闭
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := srv.Stop(ctx); err != nil {
    log.Printf("Server shutdown error: %v", err)
}
```

## 部署模式

### 开发模式 (`cmd/aster`)

简化配置，快速启动：
```bash
aster serve --port 8080 --mode debug
```

特点：
- 无认证
- CORS 允许所有来源
- 详细日志
- 热重载支持

### 生产模式 (`cmd/aster-server`)

完整特性：
```bash
export API_KEY=your-secret-key
aster-server
```

特点：
- API Key/JWT 认证
- 速率限制
- 结构化日志
- Prometheus metrics
- OpenTelemetry 追踪
- 健康检查
- 优雅关闭

## 性能

- **吞吐量**: 10,000+ req/s (基准测试)
- **延迟**: p50 < 5ms, p99 < 50ms
- **内存**: ~50MB (空闲), ~200MB (负载)
- **并发**: 支持 10,000+ 并发连接

## 安全

- TLS/HTTPS 支持
- API Key 认证
- JWT 令牌验证
- RBAC 权限控制
- 速率限制
- CORS 配置
- 请求验证
- SQL 注入防护

## 扩展

### 自定义 Handler

```go
type CustomHandler struct {
    store *store.Store
}

func (h *CustomHandler) Handle(c *gin.Context) {
    // 自定义逻辑
}

// 注册路由
srv.Router().GET("/custom", customHandler.Handle)
```

### 自定义中间件

```go
func customMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 前置处理
        c.Next()
        // 后置处理
    }
}

srv.Router().Use(customMiddleware())
```

### 自定义健康检查

```go
customCheck := observability.NewSimpleHealthCheck("database", func() error {
    return db.Ping()
})

srv.healthChecker.RegisterCheck(customCheck)
```

## 最佳实践

1. **使用环境变量管理敏感配置**
2. **启用结构化日志用于生产**
3. **配置适当的速率限制**
4. **启用 metrics 和 tracing**
5. **实现自定义健康检查**
6. **使用 HTTPS**
7. **定期更新依赖**
8. **监控资源使用**

## 相关文档

- [部署指南](../09.deployment/)
- [可观测性](../10.observability/)
- [API 参考](../14.api-reference/)
- [最佳实践](../15.best-practices/)
