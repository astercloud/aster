package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/router"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/builtin"
	"github.com/astercloud/aster/pkg/types"
	"github.com/astercloud/aster/server"
)

func main() {
	fmt.Println("🚀 aster 星尘云枢 Production Server")
	fmt.Println("================================")

	// Initialize store
	st, err := store.NewJSONStore(".data")
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}

	// Initialize tool registry
	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry)

	// Initialize factories
	sandboxFactory := sandbox.NewFactory()
	providerFactory := provider.NewMultiProviderFactory()

	// Initialize template registry
	templateRegistry := agent.NewTemplateRegistry()
	// TODO: Register builtin templates

	// Initialize router
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	defaultModel := &types.ModelConfig{
		Provider: "anthropic",
		Model:    "claude-sonnet-4-5",
		APIKey:   anthropicKey,
	}
	routes := []router.StaticRouteEntry{
		{Task: "chat", Priority: router.PriorityQuality, Model: defaultModel},
	}
	rt := router.NewStaticRouter(defaultModel, routes)

	// Create agent dependencies
	agentDeps := &agent.Dependencies{
		Store:            st,
		ToolRegistry:     toolRegistry,
		SandboxFactory:   sandboxFactory,
		ProviderFactory:  providerFactory,
		TemplateRegistry: templateRegistry,
		Router:           rt,
	}

	// Create server dependencies
	deps := &server.Dependencies{
		Store:     st,
		AgentDeps: agentDeps,
	}

	// Load configuration (use default for now)
	config := server.DefaultConfig()

	// Override with environment variables if needed
	if port := os.Getenv("PORT"); port != "" {
		_, _ = fmt.Sscanf(port, "%d", &config.Port)
	}
	if host := os.Getenv("HOST"); host != "" {
		config.Host = host
	}
	if apiKey := os.Getenv("API_KEY"); apiKey != "" {
		config.Auth.APIKey.Keys = []string{apiKey}
	}

	// Create server
	srv, err := server.New(config, deps)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start server in a goroutine
	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n🛑 Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Stop(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	fmt.Println("✅ Server exited properly")
}
