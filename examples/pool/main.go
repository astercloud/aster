package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/core"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/builtin"
	"github.com/astercloud/aster/pkg/types"
)

func main() {
	fmt.Println("=== Agent SDK - Agent Pool 示例 ===")

	// 1. 创建依赖
	deps := createDependencies()

	// 2. 创建 Agent 池
	pool := core.NewPool(&core.PoolOptions{
		Dependencies: deps,
		MaxAgents:    10, // 最多管理 10 个 Agent
	})
	defer func() { _ = pool.Shutdown() }()

	fmt.Printf("Agent 池已创建,最大容量: 10\n\n")

	// 示例 1: 创建多个 Agent
	fmt.Println("--- 示例 1: 创建多个 Agent ---")
	demonstrateCreateAgents(pool)

	// 示例 2: 列出和获取 Agent
	fmt.Println("\n--- 示例 2: 列出和获取 Agent ---")
	demonstrateListAgents(pool)

	// 示例 3: Agent 状态查询
	fmt.Println("\n--- 示例 3: Agent 状态查询 ---")
	demonstrateAgentStatus(pool)

	// 示例 4: 移除 Agent
	fmt.Println("\n--- 示例 4: 移除 Agent ---")
	demonstrateRemoveAgent(pool)

	// 示例 5: 遍历所有 Agent
	fmt.Println("\n--- 示例 5: 遍历所有 Agent ---")
	demonstrateForEach(pool)

	// 示例 6: 并发访问
	fmt.Println("\n--- 示例 6: 并发访问 ---")
	demonstrateConcurrentAccess(pool)

	fmt.Println("\n=== 所有示例完成 ===")
}

// 创建依赖
func createDependencies() *agent.Dependencies {
	// Store
	jsonStore, err := store.NewJSONStore("./.aster-pool")
	if err != nil {
		log.Fatalf("创建存储失败: %v", err)
	}

	// 工具注册表
	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry)

	// 模板注册表
	templateRegistry := agent.NewTemplateRegistry()
	templateRegistry.Register(&types.AgentTemplateDefinition{
		ID:           "assistant",
		SystemPrompt: "You are a helpful assistant with file and bash access.",
		Model:        "claude-sonnet-4-5",
		Tools:        []interface{}{"Read", "Write", "Bash"},
	})

	return &agent.Dependencies{
		Store:            jsonStore,
		SandboxFactory:   sandbox.NewFactory(),
		ToolRegistry:     toolRegistry,
		ProviderFactory:  &provider.AnthropicFactory{},
		TemplateRegistry: templateRegistry,
	}
}

// 创建 Agent 配置
func createAgentConfig(agentID string) *types.AgentConfig {
	return &types.AgentConfig{
		AgentID:    agentID,
		TemplateID: "assistant",
		ModelConfig: &types.ModelConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-5",
			APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
		},
		Sandbox: &types.SandboxConfig{
			Kind:    types.SandboxKindLocal,
			WorkDir: "./workspace",
		},
	}
}

// 示例 1: 创建多个 Agent
func demonstrateCreateAgents(pool *core.Pool) {
	ctx := context.Background()

	// 创建 3 个 Agent
	agentIDs := []string{"agent-alice", "agent-bob", "agent-charlie"}

	for _, agentID := range agentIDs {
		config := createAgentConfig(agentID)
		ag, err := pool.Create(ctx, config)
		if err != nil {
			log.Printf("创建 Agent %s 失败: %v", agentID, err)
			continue
		}
		fmt.Printf("✓ Agent 创建成功: %s\n", ag.ID())
	}

	fmt.Printf("\n当前池大小: %d\n", pool.Size())
}

// 示例 2: 列出和获取 Agent
func demonstrateListAgents(pool *core.Pool) {
	// 列出所有 Agent
	allAgents := pool.List("")
	fmt.Printf("所有 Agent: %v\n", allAgents)

	// 列出特定前缀的 Agent
	agentAgents := pool.List("agent-")
	fmt.Printf("agent- 前缀的 Agent: %v\n", agentAgents)

	// 获取特定 Agent
	if ag, exists := pool.Get("agent-alice"); exists {
		fmt.Printf("✓ 成功获取 Agent: %s\n", ag.ID())
	}
}

// 示例 3: Agent 状态查询
func demonstrateAgentStatus(pool *core.Pool) {
	status, err := pool.Status("agent-bob")
	if err != nil {
		log.Printf("获取状态失败: %v", err)
		return
	}

	fmt.Printf("Agent ID: %s\n", status.AgentID)
	fmt.Printf("状态: %s\n", status.State)
	fmt.Printf("步骤数: %d\n", status.StepCount)
	fmt.Printf("最后 SFP 索引: %d\n", status.LastSfpIndex)
	fmt.Printf("Cursor: %d\n", status.Cursor)
}

// 示例 4: 移除 Agent
func demonstrateRemoveAgent(pool *core.Pool) {
	beforeSize := pool.Size()
	fmt.Printf("移除前池大小: %d\n", beforeSize)

	err := pool.Remove("agent-charlie")
	if err != nil {
		log.Printf("移除失败: %v", err)
		return
	}

	afterSize := pool.Size()
	fmt.Printf("✓ Agent 已移除\n")
	fmt.Printf("移除后池大小: %d\n", afterSize)
}

// 示例 5: 遍历所有 Agent
func demonstrateForEach(pool *core.Pool) {
	fmt.Println("遍历所有 Agent:")

	err := pool.ForEach(func(agentID string, ag *agent.Agent) error {
		status := ag.Status()
		fmt.Printf("  - %s (状态: %s, 步骤: %d)\n",
			agentID, status.State, status.StepCount)
		return nil
	})

	if err != nil {
		log.Printf("遍历失败: %v", err)
	}
}

// 示例 6: 并发访问
func demonstrateConcurrentAccess(pool *core.Pool) {
	ctx := context.Background()

	// 并发创建多个 Agent
	workerIDs := []string{"worker-1", "worker-2", "worker-3"}

	done := make(chan string, len(workerIDs))

	for _, workerID := range workerIDs {
		go func(id string) {
			config := createAgentConfig(id)
			_, err := pool.Create(ctx, config)
			if err != nil {
				log.Printf("并发创建 %s 失败: %v", id, err)
				done <- ""
				return
			}
			done <- id
		}(workerID)
	}

	// 等待所有创建完成
	for i := 0; i < len(workerIDs); i++ {
		id := <-done
		if id != "" {
			fmt.Printf("✓ 并发创建成功: %s\n", id)
		}
	}

	fmt.Printf("\n最终池大小: %d\n", pool.Size())
}
