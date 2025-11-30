// Basic 演示 AsterOS 统一运行时系统的基本用法，包括 Agent 注册、
// 启动和优雅关闭等核心功能。
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/asteros"
	"github.com/astercloud/aster/pkg/core"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/types"
)

func main() {
	fmt.Println("=== AsterOS 基本示例 ===")
	fmt.Println("AsterOS 是 Aster 框架的统一运行时系统")
	fmt.Println()

	ctx := context.Background()

	// 1. 创建依赖
	deps := createDependencies()

	// 2. 创建 Pool
	fmt.Println("1. 创建 Pool...")
	pool := core.NewPool(&core.PoolOptions{
		Dependencies: deps,
		MaxAgents:    10,
	})

	// 3. 创建 AsterOS
	fmt.Println("2. 创建 AsterOS...")
	aster, err := asteros.New(&asteros.Options{
		Name:          "MyAsterOS",
		Port:          8080,
		Pool:          pool,
		EnableLogging: true,
		EnableCORS:    true,
		EnableMetrics: true,
		EnableHealth:  true,
	})
	if err != nil {
		log.Fatalf("Failed to create AsterOS: %v", err)
	}

	// 4. 创建并注册 Agents
	fmt.Println("3. 创建并注册 Agents...")

	// Leader Agent
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		apiKey = "sk-test-key-for-demo" // 测试用的密钥
	}

	leaderConfig := &types.AgentConfig{
		AgentID:    "leader-1",
		TemplateID: "leader",
		ModelConfig: &types.ModelConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-5",
			APIKey:   apiKey,
		},
		Sandbox: &types.SandboxConfig{
			Kind: types.SandboxKindMock,
		},
	}

	leaderAgent, err := agent.Create(ctx, leaderConfig, deps)
	if err != nil {
		log.Fatalf("Failed to create leader agent: %v", err)
	}

	if err := aster.RegisterAgent("leader-1", leaderAgent); err != nil {
		log.Fatalf("Failed to register leader agent: %v", err)
	}

	// Worker Agents
	for i := 1; i <= 2; i++ {
		workerConfig := &types.AgentConfig{
			AgentID:    fmt.Sprintf("worker-%d", i),
			TemplateID: "worker",
			ModelConfig: &types.ModelConfig{
				Provider: "anthropic",
				Model:    "claude-sonnet-4-5",
				APIKey:   apiKey,
			},
			Sandbox: &types.SandboxConfig{
				Kind: types.SandboxKindMock,
			},
		}

		workerAgent, err := agent.Create(ctx, workerConfig, deps)
		if err != nil {
			log.Fatalf("Failed to create worker agent %d: %v", i, err)
		}

		if err := aster.RegisterAgent(fmt.Sprintf("worker-%d", i), workerAgent); err != nil {
			log.Fatalf("Failed to register worker agent %d: %v", i, err)
		}
	}

	// 5. 创建并注册 Room
	fmt.Println("4. 创建并注册 Room...")

	devTeam := core.NewRoom(pool)
	_ = devTeam.Join("leader", "leader-1")
	_ = devTeam.Join("worker1", "worker-1")
	_ = devTeam.Join("worker2", "worker-2")

	if err := aster.RegisterRoom("DevTeam", devTeam); err != nil {
		log.Fatalf("Failed to register room: %v", err)
	}

	// 6. 打印 API 端点
	fmt.Println("\n5. 可用的 API 端点:")
	fmt.Println("   健康检查:")
	fmt.Println("     GET  http://localhost:8080/health")
	fmt.Println()
	fmt.Println("   Agent API:")
	fmt.Println("     GET  http://localhost:8080/agents")
	fmt.Println("     POST http://localhost:8080/agents/leader-1/run")
	fmt.Println("     GET  http://localhost:8080/agents/leader-1/status")
	fmt.Println()
	fmt.Println("   Room API:")
	fmt.Println("     GET  http://localhost:8080/rooms")
	fmt.Println("     POST http://localhost:8080/rooms/DevTeam/say")
	fmt.Println("     POST http://localhost:8080/rooms/DevTeam/join")
	fmt.Println("     GET  http://localhost:8080/rooms/DevTeam/members")
	fmt.Println()

	// 7. 启动 AsterOS
	fmt.Println("6. 启动 AsterOS...")
	fmt.Println()

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 在 goroutine 中启动服务器
	go func() {
		if err := aster.Serve(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// 等待信号
	<-sigChan
	fmt.Println("\n\n收到停止信号，正在关闭...")

	// 关闭 AsterOS
	if err := aster.Shutdown(); err != nil {
		log.Printf("Shutdown error: %v", err)
	}

	// 关闭 Pool
	if err := pool.Shutdown(); err != nil {
		log.Printf("Pool shutdown error: %v", err)
	}

	fmt.Println("✓ 示例完成!")
}

func createDependencies() *agent.Dependencies {
	// 创建存储
	jsonStore, err := store.NewJSONStore("./data")
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}

	// 创建工具注册表
	toolRegistry := tools.NewRegistry()

	// 创建模板注册表
	templateRegistry := agent.NewTemplateRegistry()

	// 注册 Leader 模板
	templateRegistry.Register(&types.AgentTemplateDefinition{
		ID:           "leader",
		SystemPrompt: "You are a team leader. Coordinate tasks and make decisions.",
		Model:        "claude-sonnet-4-5",
		Tools:        []any{},
	})

	// 注册 Worker 模板
	templateRegistry.Register(&types.AgentTemplateDefinition{
		ID:           "worker",
		SystemPrompt: "You are a team worker. Execute tasks assigned to you.",
		Model:        "claude-sonnet-4-5",
		Tools:        []any{},
	})

	// 创建 Provider 工厂
	providerFactory := &provider.AnthropicFactory{}

	return &agent.Dependencies{
		Store:            jsonStore,
		SandboxFactory:   sandbox.NewFactory(),
		ToolRegistry:     toolRegistry,
		ProviderFactory:  providerFactory,
		TemplateRegistry: templateRegistry,
	}
}
