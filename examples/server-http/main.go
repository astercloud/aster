// ServerHTTP 演示如何使用 pkg/server 暴露 HTTP Chat 接口，提供
// POST /v1/agents/chat 端点用于与 Agent 进行对话交互。
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/server"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/builtin"
	"github.com/astercloud/aster/pkg/types"
)

//     -H "Content-Type: application/json" \
//     -d '{
//       "template_id": "assistant",
//       "input": "请帮我总结一下 README",
//       "metadata": {"user_id": "alice"},
//       "middlewares": ["filesystem", "agent_memory"]
//     }'
//
// 示例响应:
//   {
//     "agent_id": "agt-...",
//     "text": "...",
//     "status": "ok"
//   }

func main() {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Println("[WARN] ANTHROPIC_API_KEY not set, server will still start but chat requests will fail until a key is provided.")
	}

	ctx := context.Background()

	// 1. 创建工具注册表并注册内置工具
	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry)

	// 2. 创建 Sandbox 工厂
	sandboxFactory := sandbox.NewFactory()

	// 3. 创建 Provider 工厂
	providerFactory := &provider.AnthropicFactory{}

	// 4. 创建 Store
	storePath := ".aster-server"
	jsonStore, err := store.NewJSONStore(storePath)
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}

	// 5. 创建模板注册表, 注册一个简单模板
	templateRegistry := agent.NewTemplateRegistry()
	templateRegistry.Register(&types.AgentTemplateDefinition{
		ID:    "assistant",
		Model: "claude-sonnet-4-5",
		SystemPrompt: "You are a helpful assistant with file and memory access. " +
			"Use filesystem and memory tools when appropriate.",
		Tools: []any{"Read", "Write", "Bash"},
	})

	deps := &agent.Dependencies{
		Store:            jsonStore,
		SandboxFactory:   sandboxFactory,
		ToolRegistry:     toolRegistry,
		ProviderFactory:  providerFactory,
		TemplateRegistry: templateRegistry,
	}

	// 6. 创建 Server
	srv := server.New(deps)

	// 7. 注册 HTTP 路由
	mux := http.NewServeMux()
	mux.Handle("/v1/agents/chat", srv.ChatHandler())
	mux.Handle("/v1/agents/chat/stream", srv.ChatStreamHandler())

	addr := ":8080"
	s := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	fmt.Printf("HTTP server started at http://localhost%s\n", addr)
	fmt.Println("POST /v1/agents/chat with JSON body for sync chat.")
	fmt.Println("POST /v1/agents/chat/stream with JSON body for SSE streaming.")

	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP server failed: %v", err)
	}

	// 优雅关闭示例, 实际生产中可以捕获信号后调用 s.Shutdown(ctx)
	_ = ctx
}
