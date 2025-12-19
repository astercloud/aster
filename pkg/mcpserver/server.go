package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/astercloud/aster/pkg/sandbox/cloud"
	"github.com/astercloud/aster/pkg/tools"
)

// Server 实现了一个简单的 MCP HTTP Server, 使用 JSON-RPC 2.0 协议:
// - tools/list: 列出所有可用工具
// - tools/call: 调用指定工具
//
// 该实现与 pkg/sandbox/cloud.MCPClient 使用的协议兼容, 可作为本地 MCP 服务端,
// 被 examples/mcp 或其他 MCP 客户端调用。
type Server struct {
	registry *tools.Registry
	executor *tools.Executor

	// 可选: 每次调用时用于构造 ToolContext 的工厂函数
	contextFactory func(ctx context.Context) *tools.ToolContext
}

// Config MCP Server 配置
type Config struct {
	Registry       *tools.Registry
	Executor       *tools.Executor
	ContextFactory func(ctx context.Context) *tools.ToolContext
}

// New 创建一个 MCP Server 实例
func New(cfg *Config) (*Server, error) {
	if cfg == nil {
		return nil, errors.New("mcpserver.Config cannot be nil")
	}
	if cfg.Registry == nil {
		return nil, errors.New("registry is required")
	}

	executor := cfg.Executor
	if executor == nil {
		executor = tools.NewExecutor(tools.ExecutorConfig{
			MaxConcurrency: 3,
			DefaultTimeout: 60 * time.Second,
		})
	}

	return &Server{
		registry:       cfg.Registry,
		executor:       executor,
		contextFactory: cfg.ContextFactory,
	}, nil
}

// Handler 返回一个 http.Handler, 处理 MCP JSON-RPC 请求。
//
// 典型挂载方式:
//
//	mux.Handle("/mcp", mcpSrv.Handler())
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req cloud.MCPRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, req.ID, -32700, "invalid JSON", err)
			return
		}

		switch req.Method {
		case "tools/list":
			s.handleToolsList(w, r.Context(), &req)
		case "tools/call":
			s.handleToolsCall(w, r.Context(), &req)
		default:
			writeError(w, req.ID, -32601, "method not found", nil)
		}
	})
}

// handleToolsList 处理 tools/list 请求
func (s *Server) handleToolsList(w http.ResponseWriter, ctx context.Context, req *cloud.MCPRequest) {
	names := s.registry.List()
	toolsResp := make([]cloud.MCPTool, 0, len(names))

	for _, name := range names {
		tool, err := s.registry.Create(name, nil)
		if err != nil {
			continue
		}
		toolsResp = append(toolsResp, cloud.MCPTool{
			Name:        name,
			Description: tool.Description(),
			InputSchema: tool.InputSchema(),
		})
	}

	resp := struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int64  `json:"id"`
		Result  struct {
			Tools []cloud.MCPTool `json:"tools"`
		} `json:"result"`
	}{
		JSONRPC: "2.0",
		ID:      req.ID,
	}
	resp.Result.Tools = toolsResp

	writeJSON(w, http.StatusOK, resp)
}

// handleToolsCall 处理 tools/call 请求
func (s *Server) handleToolsCall(w http.ResponseWriter, ctx context.Context, req *cloud.MCPRequest) {
	params := req.Params
	if params.Name == "" {
		writeError(w, req.ID, -32602, "tool name is required", nil)
		return
	}

	tool, err := s.registry.Create(params.Name, nil)
	if err != nil {
		writeError(w, req.ID, -32601, "tool not found: "+params.Name, err)
		return
	}

	var tc *tools.ToolContext
	if s.contextFactory != nil {
		tc = s.contextFactory(ctx)
	}

	// 使用 Executor 执行工具
	result := s.executor.Execute(ctx, &tools.ExecuteRequest{
		Tool:    tool,
		Input:   params.Arguments,
		Context: tc,
	})

	if result.Error != nil {
		writeError(w, req.ID, -32000, "tool execution failed", result.Error)
		return
	}

	// MCPResponse.Result 是任意 JSON, 这里直接编码为 raw message
	raw, err := json.Marshal(result.Output)
	if err != nil {
		writeError(w, req.ID, -32001, "marshal tool result failed", err)
		return
	}

	mcpResp := cloud.MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  raw,
	}

	writeJSON(w, http.StatusOK, mcpResp)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, id int64, code int, msg string, err error) {
	m := msg
	if err != nil {
		m = fmt.Sprintf("%s: %v", msg, err)
	}

	resp := cloud.MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &cloud.MCPError{
			Code:    code,
			Message: m,
		},
	}
	writeJSON(w, http.StatusOK, resp)
}
