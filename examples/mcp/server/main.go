// Server 演示如何使用 pkg/mcpserver 暴露 MCP HTTP Server，
// 支持 tools/list 和 tools/call，提供一个 echo 工具供客户端调用。
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/astercloud/aster/pkg/mcpserver"
	"github.com/astercloud/aster/pkg/tools"
)

// EchoTool 简单回显工具, 用于演示 MCP 工具调用
type EchoTool struct{}

func (t *EchoTool) Name() string        { return "echo" }
func (t *EchoTool) Description() string { return "Echo the input text with an optional prefix." }

func (t *EchoTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"text": map[string]any{
				"type":        "string",
				"description": "Text to echo.",
			},
			"prefix": map[string]any{
				"type":        "string",
				"description": "Optional prefix to add before the text.",
			},
		},
		"required": []string{"text"},
	}
}

func (t *EchoTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	text, _ := input["text"].(string)
	prefix, _ := input["prefix"].(string)

	if prefix != "" {
		return fmt.Sprintf("%s%s", prefix, text), nil
	}
	return text, nil
}

func (t *EchoTool) Prompt() string {
	return "Use this tool to echo back user-provided text, optionally with a prefix."
}

func main() {
	// 1. 注册工具
	registry := tools.NewRegistry()
	registry.Register("echo", func(config map[string]any) (tools.Tool, error) {
		return &EchoTool{}, nil
	})

	// 2. 创建 MCP Server
	srv, err := mcpserver.New(&mcpserver.Config{
		Registry: registry,
	})
	if err != nil {
		log.Fatalf("create MCP server failed: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/mcp", srv.Handler())

	addr := ":8090"
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	fmt.Printf("MCP HTTP server started at http://localhost%s/mcp\n", addr)
	fmt.Println("Supports tools/list and tools/call JSON-RPC methods.")

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("MCP server failed: %v", err)
	}
}
