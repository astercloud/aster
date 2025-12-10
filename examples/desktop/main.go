// Example: Desktop Application with Wails/Tauri/Electron
//
// This example demonstrates how to use Aster as a desktop application
// with different frontend frameworks.
//
// Usage:
//
//	go run main.go -framework wails    # For Wails integration
//	go run main.go -framework tauri    # For Tauri integration
//	go run main.go -framework electron # For Electron integration
//	go run main.go -framework web      # For web development
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/desktop"
	"github.com/astercloud/aster/pkg/permission"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/builtin"
	"github.com/astercloud/aster/pkg/types"
)

func main() {
	// Parse flags
	framework := flag.String("framework", "web", "Desktop framework: wails, tauri, electron, web")
	port := flag.Int("port", 0, "HTTP port (for tauri/electron/web)")
	workDir := flag.String("dir", ".", "Working directory")
	flag.Parse()

	// Validate API key
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("ANTHROPIC_API_KEY environment variable is required")
	}

	// Create desktop app
	app, err := desktop.NewApp(&desktop.AppConfig{
		Framework:      desktop.Framework(*framework),
		Port:           *port,
		PermissionMode: permission.ModeSmartApprove,
		WorkDir:        *workDir,
	})
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// Create agent dependencies
	deps := createDependencies(*workDir)

	// Create agent configuration
	agentConfig := &types.AgentConfig{
		TemplateID: "default",
		ModelConfig: &types.ModelConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-20250514",
			APIKey:   apiKey,
		},
		Metadata: map[string]any{
			"work_dir": *workDir,
		},
	}

	// Create and register agent
	ctx := context.Background()
	ag, err := agent.Create(ctx, agentConfig, deps)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	if err := app.RegisterAgent(ag); err != nil {
		log.Fatalf("Failed to register agent: %v", err)
	}

	// Start the app
	if err := app.Start(ctx); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}

	// Print startup info
	fmt.Printf("🚀 Aster Desktop started\n")
	fmt.Printf("   Framework: %s\n", *framework)
	fmt.Printf("   Agent ID: %s\n", ag.ID())

	switch desktop.Framework(*framework) {
	case desktop.FrameworkWails:
		fmt.Println("   Mode: Direct Go binding (bind to Wails app)")
		fmt.Println("\n   To use with Wails:")
		fmt.Println("   1. Import this package in your Wails main.go")
		fmt.Println("   2. Bind the bridge: wails.Run(&options.App{Bind: []interface{}{app.Bridge()}})")
		fmt.Println("   3. Call from frontend: window.go.desktop.WailsBridge.Chat(agentId, message)")

	case desktop.FrameworkTauri:
		bridge := app.Bridge().(*desktop.TauriBridge)
		fmt.Printf("   HTTP Server: http://127.0.0.1:%d\n", bridge.Port())
		fmt.Println("   SSE Events: GET /api/events")
		fmt.Println("\n   To use with Tauri:")
		fmt.Println("   1. Start this server before Tauri app")
		fmt.Println("   2. In Tauri frontend, connect to the HTTP API")
		fmt.Println("   3. Use SSE for streaming events")

	case desktop.FrameworkElectron:
		bridge := app.Bridge().(*desktop.ElectronBridge)
		fmt.Printf("   HTTP Server: http://127.0.0.1:%d\n", bridge.Port())
		fmt.Println("   SSE Events: GET /api/events")
		fmt.Println("\n   To use with Electron:")
		fmt.Println("   1. Start this server before Electron app")
		fmt.Println("   2. In preload.js, set up HTTP/SSE connection")
		fmt.Println("   3. Expose API via contextBridge")

	case desktop.FrameworkWeb:
		bridge := app.Bridge().(*desktop.WebBridge)
		fmt.Printf("   HTTP Server: http://localhost:%d\n", bridge.Port())
		fmt.Println("   SSE Events: GET /api/events")
		fmt.Println("\n   API Endpoints:")
		fmt.Println("   POST /api/chat     - Send message")
		fmt.Println("   POST /api/cancel   - Cancel operation")
		fmt.Println("   POST /api/approve  - Respond to approval")
		fmt.Println("   GET  /api/status   - Get agent status")
		fmt.Println("   GET  /api/history  - Get conversation history")
		fmt.Println("   GET  /api/config   - Get configuration")
		fmt.Println("   POST /api/config   - Set configuration")
		fmt.Println("   GET  /api/agents   - List agents")
		fmt.Println("   GET  /api/events   - SSE event stream")
	}

	// Wait for interrupt
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\n\n👋 Shutting down...")

	// Cleanup
	app.Stop(ctx)
	ag.Close()
}

func createDependencies(workDir string) *agent.Dependencies {
	// Create store
	dataStore, _ := store.NewJSONStore(workDir)

	// Create sandbox factory
	sandboxFactory := sandbox.NewFactory()

	// Create tool registry
	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry)

	// Create provider factory
	providerFactory := provider.NewMultiProviderFactory()

	return &agent.Dependencies{
		Store:           dataStore,
		SandboxFactory:  sandboxFactory,
		ToolRegistry:    toolRegistry,
		ProviderFactory: providerFactory,
	}
}
