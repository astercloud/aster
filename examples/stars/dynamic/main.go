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
	fmt.Println("=== Stars 动态成员管理示例 ===")
	fmt.Println("演示如何动态添加和移除群星成员")
	fmt.Println()

	ctx := context.Background()

	// 1. 创建依赖
	deps := createDependencies()

	// 2. 创建 Cosmos
	fmt.Println("1. 创建 Cosmos...")
	cosmos := cosmos.New(&cosmos.Options{
		Dependencies: deps,
		MaxAgents:    20,
	})
	defer func() { _ = cosmos.Shutdown() }()

	// 3. 创建初始 Agents
	fmt.Println("\n2. 创建初始 Agents...")
	createAgent(ctx, cosmos, "leader-1", "leader")
	createAgent(ctx, cosmos, "worker-1", "worker")
	createAgent(ctx, cosmos, "worker-2", "worker")

	// 4. 创建 Stars
	fmt.Println("\n3. 创建 Stars...")
	team := stars.New(cosmos, "DynamicTeam")
	_ = team.Join("leader-1", stars.RoleLeader)
	_ = team.Join("worker-1", stars.RoleWorker)
	_ = team.Join("worker-2", stars.RoleWorker)
	printMembers(team)

	// 5. 动态添加成员
	fmt.Println("\n4. 动态添加新成员...")
	createAgent(ctx, cosmos, "worker-3", "worker")
	_ = team.Join("worker-3", stars.RoleWorker)
	fmt.Println("   ✓ 添加 worker-3")
	printMembers(team)

	createAgent(ctx, cosmos, "worker-4", "worker")
	_ = team.Join("worker-4", stars.RoleWorker)
	fmt.Println("   ✓ 添加 worker-4")
	printMembers(team)

	// 6. 移除成员
	fmt.Println("\n5. 移除成员...")
	_ = team.Leave("worker-2")
	fmt.Println("   ✓ 移除 worker-2")
	printMembers(team)

	// 7. 再次添加成员
	fmt.Println("\n6. 再次添加成员...")
	createAgent(ctx, cosmos, "worker-5", "worker")
	_ = team.Join("worker-5", stars.RoleWorker)
	fmt.Println("   ✓ 添加 worker-5")
	printMembers(team)

	// 8. 批量操作
	fmt.Println("\n7. 批量添加成员...")
	for i := 6; i <= 8; i++ {
		agentID := fmt.Sprintf("worker-%d", i)
		createAgent(ctx, cosmos, agentID, "worker")
		_ = team.Join(agentID, stars.RoleWorker)
		fmt.Printf("   ✓ 添加 %s\n", agentID)
	}
	printMembers(team)

	// 9. 查看 Cosmos 中的所有 Agent
	fmt.Println("\n8. Cosmos 中的所有 Agents...")
	allAgents := cosmos.List("")
	fmt.Printf("   总共 %d 个 Agents:\n", len(allAgents))
	for _, agentID := range allAgents {
		fmt.Printf("   - %s\n", agentID)
	}

	// 10. 查看 Stars 成员
	fmt.Println("\n9. Stars 最终成员...")
	printMembers(team)

	fmt.Println("\n✓ 动态成员管理示例完成!")
}

func createAgent(ctx context.Context, cosmos *cosmos.Cosmos, agentID, templateID string) {
	config := &types.AgentConfig{
		AgentID:    agentID,
		TemplateID: templateID,
		ModelConfig: &types.ModelConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-5",
			APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
		},
		Sandbox: &types.SandboxConfig{
			Kind: types.SandboxKindMock,
		},
	}

	_, err := cosmos.Create(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create agent %s: %v", agentID, err)
	}
}

func printMembers(team *stars.Stars) {
	members := team.Members()
	fmt.Printf("   当前成员 (%d 个):\n", len(members))

	// 分类显示
	var leaders, workers []string
	for _, m := range members {
		if m.Role == stars.RoleLeader {
			leaders = append(leaders, m.AgentID)
		} else {
			workers = append(workers, m.AgentID)
		}
	}

	if len(leaders) > 0 {
		fmt.Printf("   Leaders: %v\n", leaders)
	}
	if len(workers) > 0 {
		fmt.Printf("   Workers: %v\n", workers)
	}
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
		SystemPrompt: "You are a team leader.",
		Model:        "claude-sonnet-4-5",
		Tools:        []interface{}{},
	})

	templateRegistry.Register(&types.AgentTemplateDefinition{
		ID:           "worker",
		SystemPrompt: "You are a team worker.",
		Model:        "claude-sonnet-4-5",
		Tools:        []interface{}{},
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
