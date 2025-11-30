// WithInterfaces 演示 AsterOS 与多种接口的集成，展示如何配置和使用
// 不同的输入输出接口。
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
	"github.com/astercloud/aster/pkg/asteros/interfaces"
	"github.com/astercloud/aster/pkg/core"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/types"
)

func main() {
	fmt.Println("=== AsterOS with Interfaces 示例 ===")
	fmt.Println("演示如何使用多种 Interfaces")
	fmt.Println()

	ctx := context.Background()

	// 1. 创建依赖
	deps := createDependencies()

	// 2. 创建 Pool
	fmt.Println("1. 创建 Pool...")
	pool := core.NewPool(&core.PoolOptions{
		Dependencies: deps,
		MaxAgents:    20,
	})

	// 3. 创建 AsterOS
	fmt.Println("2. 创建 AsterOS...")
	aster, err := asteros.New(&asteros.Options{
		Name:          "MultiInterfaceOS",
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

	// 4. 添加 Interfaces
	fmt.Println("3. 添加 Interfaces...")

	// HTTP Interface（默认）
	httpInterface := interfaces.NewHTTPInterface(&interfaces.HTTPInterfaceOptions{
		EnableLogging: true,
	})
	if err := aster.AddInterface(httpInterface); err != nil {
		log.Fatalf("Failed to add HTTP interface: %v", err)
	}
	fmt.Println("   ✓ HTTP Interface 已添加")

	// A2A Interface
	a2aInterface := interfaces.NewA2AInterface(&interfaces.A2AInterfaceOptions{
		GRPCPort:        9090,
		EnableLogging:   true,
		EnableDiscovery: true,
	})
	if err := aster.AddInterface(a2aInterface); err != nil {
		log.Fatalf("Failed to add A2A interface: %v", err)
	}
	fmt.Println("   ✓ A2A Interface 已添加 (gRPC port: 9090)")

	// AGUI Interface
	aguiInterface := interfaces.NewAGUIInterface(&interfaces.AGUIInterfaceOptions{
		ControlPlaneURL: "https://os.aster.com",
		APIKey:          os.Getenv("AGUI_API_KEY"),
		EnableLogging:   true,
		EnableAutoSync:  true,
	})
	if err := aster.AddInterface(aguiInterface); err != nil {
		log.Fatalf("Failed to add AGUI interface: %v", err)
	}
	fmt.Println("   ✓ AGUI Interface 已添加")

	// 5. 创建并注册 Agents
	fmt.Println("\n4. 创建并注册 Agents...")

	// Leader Agent
	leaderConfig := &types.AgentConfig{
		AgentID:    "leader-1",
		TemplateID: "leader",
		ModelConfig: &types.ModelConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-5",
			APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
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
	for i := 1; i <= 3; i++ {
		workerConfig := &types.AgentConfig{
			AgentID:    fmt.Sprintf("worker-%d", i),
			TemplateID: "worker",
			ModelConfig: &types.ModelConfig{
				Provider: "anthropic",
				Model:    "claude-sonnet-4-5",
				APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
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

	// 6. 创建并注册 Room
	fmt.Println("\n5. 创建并注册 Room...")

	devTeam := core.NewRoom(pool)
	_ = devTeam.Join("leader", "leader-1")
	_ = devTeam.Join("worker1", "worker-1")
	_ = devTeam.Join("worker2", "worker-2")
	_ = devTeam.Join("worker3", "worker-3")

	if err := aster.RegisterRoom("DevTeam", devTeam); err != nil {
		log.Fatalf("Failed to register room: %v", err)
	}

	// 7. 打印可用的 API
	fmt.Println("\n6. 可用的 API 端点:")
	fmt.Println()
	fmt.Println("   HTTP REST API (port 8080):")
	fmt.Println("   ─────────────────────────────")
	fmt.Println("   健康检查:")
	fmt.Println("     GET  http://localhost:8080/health")
	fmt.Println()
	fmt.Println("   Agent API:")
	fmt.Println("     GET  http://localhost:8080/agents")
	fmt.Println("     POST http://localhost:8080/agents/leader-1/run")
	fmt.Println("     POST http://localhost:8080/agents/worker-1/run")
	fmt.Println("     GET  http://localhost:8080/agents/leader-1/status")
	fmt.Println()
	fmt.Println("   Stars API:")
	fmt.Println("     GET  http://localhost:8080/stars")
	fmt.Println("     POST http://localhost:8080/stars/DevTeam/run")
	fmt.Println("     POST http://localhost:8080/stars/DevTeam/join")
	fmt.Println("     POST http://localhost:8080/stars/DevTeam/leave")
	fmt.Println("     GET  http://localhost:8080/stars/DevTeam/members")
	fmt.Println()
	fmt.Println("   A2A Interface (gRPC port 9090):")
	fmt.Println("   ─────────────────────────────")
	fmt.Println("     Agent-to-Agent 通信已启用")
	fmt.Println()
	fmt.Println("   AGUI Interface:")
	fmt.Println("   ─────────────────────────────")
	fmt.Println("     控制平面: https://os.aster.com")
	fmt.Println("     自动同步已启用")
	fmt.Println()

	// 8. 打印测试命令
	fmt.Println("7. 测试命令:")
	fmt.Println()
	fmt.Println("   # 列出所有 Agents")
	fmt.Println("   curl http://localhost:8080/agents")
	fmt.Println()
	fmt.Println("   # 运行 Agent")
	fmt.Println(`   curl -X POST http://localhost:8080/agents/leader-1/run \`)
	fmt.Println(`     -H "Content-Type: application/json" \`)
	fmt.Println(`     -d '{"message": "Hello, Agent!"}'`)
	fmt.Println()
	fmt.Println("   # 获取 Stars 成员")
	fmt.Println("   curl http://localhost:8080/stars/DevTeam/members")
	fmt.Println()
	fmt.Println("   # 运行 Stars")
	fmt.Println(`   curl -X POST http://localhost:8080/stars/DevTeam/run \`)
	fmt.Println(`     -H "Content-Type: application/json" \`)
	fmt.Println(`     -d '{"task": "开发新功能"}'`)
	fmt.Println()

	// 9. 启动服务器
	fmt.Println("按 Ctrl+C 停止服务器")
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

	fmt.Println("\n✓ 示例完成!")
}

func createDependencies() *agent.Dependencies {
	jsonStore, err := store.NewJSONStore("./data")
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}

	toolRegistry := tools.NewRegistry()
	templateRegistry := agent.NewTemplateRegistry()

	templateRegistry.Register(&types.AgentTemplateDefinition{
		ID:           "leader",
		SystemPrompt: "You are a team leader. Coordinate tasks and make decisions.",
		Model:        "claude-sonnet-4-5",
		Tools:        []any{},
	})

	templateRegistry.Register(&types.AgentTemplateDefinition{
		ID:           "worker",
		SystemPrompt: "You are a team worker. Execute tasks assigned to you.",
		Model:        "claude-sonnet-4-5",
		Tools:        []any{},
	})

	providerFactory := &provider.AnthropicFactory{}

	return &agent.Dependencies{
		Store:            jsonStore,
		SandboxFactory:   sandbox.NewFactory(),
		ToolRegistry:     toolRegistry,
		ProviderFactory:  providerFactory,
		TemplateRegistry: templateRegistry,
	}
}
