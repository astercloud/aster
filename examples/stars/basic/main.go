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
	"github.com/astercloud/aster/pkg/stars"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/types"
)

func main() {
	fmt.Println("=== Stars (群星) 基本示例 ===")
	fmt.Println("Stars 是 Aster 框架中的多 Agent 协作组件")
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

	// 3. 创建 Agents
	fmt.Println("2. 创建 Agents...")

	// 创建 Leader Agent
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

	_, err := cosmos.Create(ctx, leaderConfig)
	if err != nil {
		log.Fatalf("Failed to create leader: %v", err)
	}
	fmt.Println("   ✓ 创建 Leader Agent")

	// 创建 Worker Agents
	for i := 1; i <= 2; i++ {
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

		_, err := cosmos.Create(ctx, workerConfig)
		if err != nil {
			log.Fatalf("Failed to create worker %d: %v", i, err)
		}
		fmt.Printf("   ✓ 创建 Worker Agent %d\n", i)
	}

	// 4. 创建 Stars 群星
	fmt.Println("\n3. 创建 Stars 群星...")
	devTeam := stars.New(cosmos, "DevTeam")
	fmt.Println("   ✓ 创建群星: DevTeam")

	// 5. 添加成员
	fmt.Println("\n4. 添加成员...")
	err = devTeam.Join("leader-1", stars.RoleLeader)
	if err != nil {
		log.Fatalf("Failed to join leader: %v", err)
	}
	fmt.Println("   ✓ leader-1 加入 (Leader)")

	err = devTeam.Join("worker-1", stars.RoleWorker)
	if err != nil {
		log.Fatalf("Failed to join worker-1: %v", err)
	}
	fmt.Println("   ✓ worker-1 加入 (Worker)")

	err = devTeam.Join("worker-2", stars.RoleWorker)
	if err != nil {
		log.Fatalf("Failed to join worker-2: %v", err)
	}
	fmt.Println("   ✓ worker-2 加入 (Worker)")

	// 6. 查看成员
	fmt.Println("\n5. 查看成员...")
	members := devTeam.Members()
	fmt.Printf("   群星成员 (%d 个):\n", len(members))
	for _, member := range members {
		fmt.Printf("   - %s (%s)\n", member.AgentID, member.Role)
	}

	// 7. 发送消息
	fmt.Println("\n6. 发送消息...")
	err = devTeam.Send(ctx, "leader-1", "worker-1", "请处理任务 A")
	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}
	fmt.Println("   ✓ leader-1 → worker-1: 请处理任务 A")

	err = devTeam.Broadcast(ctx, "开始新的迭代")
	if err != nil {
		log.Fatalf("Failed to broadcast: %v", err)
	}
	fmt.Println("   ✓ 广播: 开始新的迭代")

	// 8. 查看消息历史
	fmt.Println("\n7. 查看消息历史...")
	history := devTeam.History()
	fmt.Printf("   消息历史 (%d 条):\n", len(history))
	for i, msg := range history {
		if msg.To == "" {
			fmt.Printf("   %d. [广播] %s: %s\n", i+1, msg.From, msg.Text)
		} else {
			fmt.Printf("   %d. %s → %s: %s\n", i+1, msg.From, msg.To, msg.Text)
		}
	}

	fmt.Println("\n✓ Stars 基本示例完成!")
	fmt.Println("\n提示: 要运行任务执行示例，请确保设置了 ANTHROPIC_API_KEY 环境变量")
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
		Tools:        []interface{}{},
	})

	// 注册 Worker 模板
	templateRegistry.Register(&types.AgentTemplateDefinition{
		ID:           "worker",
		SystemPrompt: "You are a team worker. Execute tasks assigned to you.",
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
