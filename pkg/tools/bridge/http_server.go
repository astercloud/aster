package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/tools"
)

// 性能优化: Schema 缓存
type schemaCache struct {
	schemas map[string]any
	mu      sync.RWMutex
	ttl     time.Duration
	entries map[string]*cacheEntry
}

type cacheEntry struct {
	data      any
	timestamp time.Time
}

func newSchemaCache(ttl time.Duration) *schemaCache {
	return &schemaCache{
		schemas: make(map[string]any),
		entries: make(map[string]*cacheEntry),
		ttl:     ttl,
	}
}

func (c *schemaCache) get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}

	// 检查是否过期
	if time.Since(entry.timestamp) > c.ttl {
		return nil, false
	}

	return entry.data, true
}

func (c *schemaCache) set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &cacheEntry{
		data:      value,
		timestamp: time.Now(),
	}
}

// HTTPBridgeServer HTTP 桥接服务器
// 提供 HTTP API 供 Python/Node.js 代码调用 Go 侧的工具
type HTTPBridgeServer struct {
	bridge *ToolBridge
	server *http.Server
	mu     sync.RWMutex

	// 工具上下文工厂
	contextFactory func() *tools.ToolContext

	// 性能优化: Schema 缓存
	schemaCache *schemaCache
}

// NewHTTPBridgeServer 创建 HTTP 桥接服务器
func NewHTTPBridgeServer(bridge *ToolBridge, addr string) *HTTPBridgeServer {
	s := &HTTPBridgeServer{
		bridge: bridge,
		server: &http.Server{
			Addr:         addr,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			// 性能优化: 配置更大的缓冲区
			ReadHeaderTimeout: 5 * time.Second,
			MaxHeaderBytes:    1 << 20, // 1MB
			// 启用 keep-alive 连接复用
			IdleTimeout: 120 * time.Second,
		},
		// 初始化 Schema 缓存(5分钟TTL)
		schemaCache: newSchemaCache(5 * time.Minute),
	}

	// 设置路由
	mux := http.NewServeMux()
	mux.HandleFunc("/tools/call", s.handleToolCall)
	mux.HandleFunc("/tools/list", s.handleToolList)
	mux.HandleFunc("/tools/schema", s.handleToolSchema)
	mux.HandleFunc("/health", s.handleHealth)

	s.server.Handler = mux
	return s
}

// SetContextFactory 设置工具上下文工厂
func (s *HTTPBridgeServer) SetContextFactory(factory func() *tools.ToolContext) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.contextFactory = factory
}

// ToolCallRequest 工具调用请求
type ToolCallRequest struct {
	Tool  string         `json:"tool"`
	Input map[string]any `json:"input"`
}

// ToolCallResponse 工具调用响应
type ToolCallResponse struct {
	Success bool   `json:"success"`
	Result  any    `json:"result,omitempty"`
	Error   string `json:"error,omitempty"`
}

// handleToolCall 处理工具调用请求
func (s *HTTPBridgeServer) handleToolCall(w http.ResponseWriter, r *http.Request) {
	// 仅支持 POST
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求
	var req ToolCallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	// 验证参数
	if req.Tool == "" {
		s.sendError(w, "Tool name is required", http.StatusBadRequest)
		return
	}

	// 获取工具上下文
	tc := s.getToolContext()

	// 调用工具
	ctx := r.Context()
	result, _ := s.bridge.CallTool(ctx, req.Tool, req.Input, tc)

	// 返回响应
	s.sendJSON(w, &ToolCallResponse{
		Success: result.Success,
		Result:  result.Result,
		Error:   result.Error,
	})
}

// handleToolList 处理工具列表请求
func (s *HTTPBridgeServer) handleToolList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tools := s.bridge.ListAvailableTools()
	s.sendJSON(w, map[string]any{
		"tools": tools,
	})
}

// handleToolSchema 处理工具 Schema 请求(带缓存)
func (s *HTTPBridgeServer) handleToolSchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	toolName := r.URL.Query().Get("name")
	if toolName == "" {
		s.sendError(w, "Tool name is required", http.StatusBadRequest)
		return
	}

	// 尝试从缓存获取
	if cachedSchema, ok := s.schemaCache.get(toolName); ok {
		s.sendJSON(w, cachedSchema)
		return
	}

	// 缓存未命中,从 bridge 获取
	schema, err := s.bridge.GetToolSchema(toolName)
	if err != nil {
		s.sendError(w, err.Error(), http.StatusNotFound)
		return
	}

	// 存入缓存
	s.schemaCache.set(toolName, schema)

	s.sendJSON(w, schema)
}

// handleHealth 健康检查
func (s *HTTPBridgeServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.sendJSON(w, map[string]string{
		"status": "ok",
	})
}

// getToolContext 获取工具上下文
func (s *HTTPBridgeServer) getToolContext() *tools.ToolContext {
	s.mu.RLock()
	factory := s.contextFactory
	s.mu.RUnlock()

	if factory != nil {
		return factory()
	}

	// 默认返回空上下文
	return &tools.ToolContext{
		Services: make(map[string]any),
	}
}

// sendJSON 发送 JSON 响应
func (s *HTTPBridgeServer) sendJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// 无法发送错误响应给客户端，仅记录日志
		fmt.Fprintf(os.Stderr, "Failed to encode JSON response: %v\n", err)
	}
}

// sendError 发送错误响应
func (s *HTTPBridgeServer) sendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	}); err != nil {
		// 无法发送错误响应给客户端，仅记录日志
		fmt.Fprintf(os.Stderr, "Failed to encode error response: %v\n", err)
	}
}

// Start 启动服务器
func (s *HTTPBridgeServer) Start() error {
	fmt.Printf("HTTP Bridge Server listening on %s\n", s.server.Addr)
	return s.server.ListenAndServe()
}

// StartAsync 异步启动服务器
func (s *HTTPBridgeServer) StartAsync() error {
	go func() {
		if err := s.Start(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("HTTP Bridge Server error: %v\n", err)
		}
	}()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)
	return nil
}

// Shutdown 关闭服务器
func (s *HTTPBridgeServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
