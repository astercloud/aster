package asteros

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/agent/workflow"
	"github.com/astercloud/aster/pkg/core"
	"github.com/gin-gonic/gin"
)

// AsterOS Aster 框架的统一运行时系统
// AsterOS 负责管理所有 Agents、Rooms、Workflows，
// 并自动生成 REST API 端点，支持多种 Interface。
type AsterOS struct {
	// 核心组件
	pool     *core.Pool
	registry *Registry
	router   *gin.Engine
	server   *http.Server

	// Interface 层
	interfaces map[string]Interface
	ifMu       sync.RWMutex

	// 配置
	opts *Options

	// 生命周期
	ctx     context.Context
	cancel  context.CancelFunc
	running bool
	mu      sync.RWMutex
}

// New 创建 AsterOS 实例
func New(opts *Options) (*AsterOS, error) {
	// 使用默认配置
	if opts == nil {
		opts = DefaultOptions()
	}

	// 验证配置
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 创建 AsterOS
	os := &AsterOS{
		pool:       opts.Pool,
		registry:   NewRegistry(),
		interfaces: make(map[string]Interface),
		opts:       opts,
		ctx:        ctx,
		cancel:     cancel,
		running:    false,
	}

	// 初始化路由
	os.initRouter()

	return os, nil
}

// initRouter 初始化路由
func (os *AsterOS) initRouter() {
	// 设置 Gin 模式
	if os.opts.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由器
	os.router = gin.New()

	// 添加中间件
	if os.opts.EnableLogging {
		os.router.Use(gin.Logger())
	}
	os.router.Use(gin.Recovery())

	// CORS
	if os.opts.EnableCORS {
		os.router.Use(corsMiddleware())
	}

	// 认证
	if os.opts.EnableAuth {
		os.router.Use(os.authMiddleware())
	}

	// 健康检查
	if os.opts.EnableHealth {
		os.router.GET("/health", os.handleHealth)
	}

	// Prometheus 指标
	if os.opts.EnableMetrics {
		os.router.GET("/metrics", os.handleMetrics)
	}

	// API 路由组
	api := os.router.Group(os.opts.APIPrefix)
	{
		// Agent 路由
		agents := api.Group("/agents")
		{
			agents.GET("", os.handleListAgents)
			agents.POST("/:id/run", os.handleAgentRun)
			agents.GET("/:id/status", os.handleAgentStatus)
		}

		// Rooms 路由
		rooms := api.Group("/rooms")
		{
			rooms.GET("", os.handleListRooms)
			rooms.POST("/:id/say", os.handleRoomSay)
			rooms.POST("/:id/join", os.handleRoomJoin)
			rooms.POST("/:id/leave", os.handleRoomLeave)
			rooms.GET("/:id/members", os.handleRoomMembers)
		}

		// Workflow 路由
		workflows := api.Group("/workflows")
		{
			workflows.GET("", os.handleListWorkflows)
			workflows.POST("/:id/execute", os.handleWorkflowExecute)
		}
	}
}

// RegisterAgent 注册 Agent
func (os *AsterOS) RegisterAgent(id string, ag *agent.Agent) error {
	// 注册到 Registry
	if err := os.registry.RegisterAgent(id, ag); err != nil {
		return err
	}

	// 通知所有 Interfaces
	os.notifyAgentRegistered(ag)

	return nil
}

// RegisterRoom 注册 Room
func (os *AsterOS) RegisterRoom(id string, r *core.Room) error {
	// 注册到 Registry
	if err := os.registry.RegisterRoom(id, r); err != nil {
		return err
	}

	// 通知所有 Interfaces
	os.notifyRoomRegistered(r)

	return nil
}

// RegisterWorkflow 注册 Workflow
func (os *AsterOS) RegisterWorkflow(id string, wf workflow.Agent) error {
	// 注册到 Registry
	if err := os.registry.RegisterWorkflow(id, wf); err != nil {
		return err
	}

	// 通知所有 Interfaces
	os.notifyWorkflowRegistered(wf)

	return nil
}

// AddInterface 添加 Interface
func (os *AsterOS) AddInterface(iface Interface) error {
	os.ifMu.Lock()
	defer os.ifMu.Unlock()

	name := iface.Name()
	if _, exists := os.interfaces[name]; exists {
		return ErrInterfaceExists
	}

	os.interfaces[name] = iface
	return nil
}

// RemoveInterface 移除 Interface
func (os *AsterOS) RemoveInterface(name string) error {
	os.ifMu.Lock()
	defer os.ifMu.Unlock()

	if _, exists := os.interfaces[name]; !exists {
		return ErrInterfaceNotFound
	}

	delete(os.interfaces, name)
	return nil
}

// Serve 启动 AsterOS
func (os *AsterOS) Serve() error {
	os.mu.Lock()
	if os.running {
		os.mu.Unlock()
		return ErrAlreadyRunning
	}
	os.running = true
	os.mu.Unlock()

	// 启动所有 Interfaces
	if err := os.startInterfaces(); err != nil {
		return fmt.Errorf("start interfaces: %w", err)
	}

	// 创建 HTTP 服务器
	os.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", os.opts.Port),
		Handler: os.router,
	}

	// 启动服务器
	fmt.Printf("🌟 AsterOS '%s' is running on http://localhost:%d\n", os.opts.Name, os.opts.Port)
	if err := os.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("listen and serve: %w", err)
	}

	return nil
}

// Shutdown 关闭 AsterOS
func (os *AsterOS) Shutdown() error {
	os.mu.Lock()
	if !os.running {
		os.mu.Unlock()
		return ErrNotRunning
	}
	os.running = false
	os.mu.Unlock()

	// 停止所有 Interfaces
	if err := os.stopInterfaces(); err != nil {
		fmt.Printf("Warning: stop interfaces: %v\n", err)
	}

	// 关闭 HTTP 服务器
	if os.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := os.server.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}
	}

	// 取消上下文
	os.cancel()

	fmt.Printf("✓ AsterOS '%s' shutdown complete\n", os.opts.Name)
	return nil
}

// Pool 获取 Pool 实例
func (os *AsterOS) Pool() *core.Pool {
	return os.pool
}

// Registry 获取 Registry 实例
func (os *AsterOS) Registry() *Registry {
	return os.registry
}

// Name 获取 AsterOS 名称
func (os *AsterOS) Name() string {
	return os.opts.Name
}

// Router 获取 Gin Router
func (os *AsterOS) Router() *gin.Engine {
	return os.router
}

// IsRunning 检查是否正在运行
func (os *AsterOS) IsRunning() bool {
	os.mu.RLock()
	defer os.mu.RUnlock()
	return os.running
}

// startInterfaces 启动所有 Interfaces
func (os *AsterOS) startInterfaces() error {
	os.ifMu.RLock()
	defer os.ifMu.RUnlock()

	for name, iface := range os.interfaces {
		if err := iface.Start(os.ctx, os); err != nil {
			return fmt.Errorf("start interface %s: %w", name, err)
		}
	}

	return nil
}

// stopInterfaces 停止所有 Interfaces
func (os *AsterOS) stopInterfaces() error {
	os.ifMu.RLock()
	defer os.ifMu.RUnlock()

	var lastErr error
	for name, iface := range os.interfaces {
		if err := iface.Stop(os.ctx); err != nil {
			lastErr = fmt.Errorf("stop interface %s: %w", name, err)
		}
	}

	return lastErr
}

// notifyAgentRegistered 通知所有 Interfaces Agent 已注册
func (os *AsterOS) notifyAgentRegistered(ag *agent.Agent) {
	os.ifMu.RLock()
	defer os.ifMu.RUnlock()

	for _, iface := range os.interfaces {
		if err := iface.OnAgentRegistered(ag); err != nil {
			fmt.Printf("Warning: interface %s OnAgentRegistered: %v\n", iface.Name(), err)
		}
	}
}

// notifyRoomRegistered 通知所有 Interfaces Room 已注册
func (os *AsterOS) notifyRoomRegistered(r *core.Room) {
	os.ifMu.RLock()
	defer os.ifMu.RUnlock()

	for _, iface := range os.interfaces {
		if err := iface.OnRoomRegistered(r); err != nil {
			fmt.Printf("Warning: interface %s OnRoomRegistered: %v\n", iface.Name(), err)
		}
	}
}

// notifyWorkflowRegistered 通知所有 Interfaces Workflow 已注册
func (os *AsterOS) notifyWorkflowRegistered(wf workflow.Agent) {
	os.ifMu.RLock()
	defer os.ifMu.RUnlock()

	for _, iface := range os.interfaces {
		if err := iface.OnWorkflowRegistered(wf); err != nil {
			fmt.Printf("Warning: interface %s OnWorkflowRegistered: %v\n", iface.Name(), err)
		}
	}
}

// corsMiddleware CORS 中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// authMiddleware 认证中间件
func (os *AsterOS) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("Authorization")
		if apiKey == "" {
			apiKey = c.Query("api_key")
		}

		if apiKey != "Bearer "+os.opts.APIKey && apiKey != os.opts.APIKey {
			c.JSON(401, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		c.Next()
	}
}
