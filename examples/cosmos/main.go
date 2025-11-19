package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/cosmos"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/types"
)

func main() {
	fmt.Println("=== Cosmos (宇宙) 示例 ===")
	fmt.Println("Cosmos 是 Aster 框架中的 Agent 生命周期管理器")
	fmt.Println()

	ctx := context.Background()

	// 1. 创建依赖
	deps := createDependencies()

	// 2. 创建 Cosmos 宇宙
	fmt.Println("1. 创建 Cosmos 宇宙...")
	cosmos := cosmos.New(&cosmos.Options{
		Dependencies: deps,
		MaxAgents:    10,
	})
	defer func() { _ = cosmos.Shutdown() }()

	// 3. 创建多个 Agent
	fmt.Println("2. 创建 Agents...")

	// 创建 Leader Agent
	leaderConfig := &types.AgentConfig{
		AgentID:    "leader-1",
		TemplateID: "assistant",
		ModelConfig: &types.ModelConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-5",
			APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
		},
		Sandbox: &types.SandboxConfig{
			Kind: types.SandboxKindMock,
		},
	}

	leaderAgent, err := cosmos.Create(ctx, leaderConfig)
	if err != nil {
		log.Fatalf("Failed to create leader agent: %v", err)
	}
	fmt.Printf("   ✓ 创建 Leader Agent: %s\n", leaderAgent.ID())

	// 创建 Worker Agents
	for i := 1; i <= 3; i++ {
		workerConfig := &types.AgentConfig{
			AgentID:    fmt.Sprintf("worker-%d", i),
			TemplateID: "assistant",
			ModelConfig: &types.ModelConfig{
				Provider: "anthropic",
				Model:    "claude-sonnet-4-5",
				APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
			},
			Sandbox: &types.SandboxConfig{
				Kind: types.SandboxKindMock,
			},
		}

		workerAgent, err := cosmos.Create(ctx, workerConfig)
		if err != nil {
			log.Fatalf("Failed to create worker agent %d: %v", i, err)
		}
		fmt.Printf("   ✓ 创建 Worker Agent: %s\n", workerAgent.ID())
	}

	// 4. 列出所有 Agent
	fmt.Println("\n3. 列出所有 Agents...")
	allAgents := cosmos.List("")
	fmt.Printf("   总共 %d 个 Agents:\n", len(allAgents))
	for _, agentID := range allAgents {
		fmt.Printf("   - %s\n", agentID)
	}

	// 5. 按前缀过滤
	fmt.Println("\n4. 按前缀过滤...")
	workers := cosmos.List("worker-")
	fmt.Printf("   Worker Agents (%d 个):\n", len(workers))
	for _, agentID := range workers {
		fmt.Printf("   - %s\n", agentID)
	}

	// 6. 获取 Agent 状态
	fmt.Println("\n5. 获取 Agent 状态...")
	status, err := cosmos.Status("leader-1")
	if err != nil {
		log.Fatalf("Failed to get status: %v", err)
	}
	fmt.Printf("   Leader Agent 状态:\n")
	fmt.Printf("   - ID: %s\n", status.AgentID)
	fmt.Printf("   - State: %s\n", status.State)
	fmt.Printf("   - Step: %d\n", status.StepCount)

	// 7. 遍历所有 Agent
	fmt.Println("\n6. 遍历所有 Agents...")
	err = cosmos.ForEach(func(agentID string, ag *agent.Agent) error {
		status := ag.Status()
		fmt.Printf("   - %s: %s\n", agentID, status.State)
		return nil
	})
	if err != nil {
		log.Fatalf("ForEach failed: %v", err)
	}

	// 8. 移除 Agent
	fmt.Println("\n7. 移除 Agent...")
	err = cosmos.Remove("worker-3")
	if err != nil {
		log.Fatalf("Failed to remove agent: %v", err)
	}
	fmt.Printf("   ✓ 移除 worker-3\n")
	fmt.Printf("   剩余 %d 个 Agents\n", cosmos.Size())

	fmt.Println("\n✓ Cosmos 示例完成!")
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
	templateRegistry.Register(&types.AgentTemplateDefinition{
		ID:           "assistant",
		SystemPrompt: "You are a helpful assistant",
		Model:        "claude-sonnet-4-5",
		Tools:        []interface{}{},
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
