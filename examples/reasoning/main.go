package main

import (
	"context"
	"fmt"
	"log"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/types"
)

func main() {
	ctx := context.Background()

	// 创建依赖
	deps := &agent.Dependencies{
		Store:            store.NewMemoryStore(),
		SandboxFactory:   sandbox.NewFactory(),
		ToolRegistry:     tools.DefaultRegistry,
		ProviderFactory:  provider.NewFactory(),
		TemplateRegistry: createTemplateRegistry(),
	}

	// 创建 Agent 配置（启用 Reasoning Middleware）
	config := &types.AgentConfig{
		TemplateID: "reasoning-agent",
		ModelConfig: &types.ModelConfig{
			Provider: "anthropic",
			Model:    "claude-3-5-sonnet-20241022",
		},
		Middlewares: []string{"reasoning"},
		MiddlewareConfig: map[string]map[string]interface{}{
			"reasoning": {
				"enabled":        true,
				"min_steps":      2,
				"max_steps":      5,
				"min_confidence": 0.7,
				"use_json":       true,
			},
		},
		Metadata: map[string]interface{}{
			"enable_reasoning": true,
		},
	}

	// 创建 Agent
	ag, err := agent.Create(ctx, config, deps)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	defer ag.Close()

	// 测试推理能力
	testProblems := []string{
		"How can we optimize database query performance for a high-traffic web application?",
		"What are the trade-offs between microservices and monolithic architecture?",
		"Design a caching strategy for an e-commerce platform",
	}

	for i, problem := range testProblems {
		fmt.Printf("\n=== Problem %d ===\n", i+1)
		fmt.Printf("Question: %s\n\n", problem)

		result, err := ag.Chat(ctx, problem)
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}

		fmt.Printf("Answer:\n%s\n", result.Text)
		fmt.Println(strings.Repeat("=", 80))
	}
}

func createTemplateRegistry() *agent.TemplateRegistry {
	registry := agent.NewTemplateRegistry()

	registry.Register(&types.AgentTemplateDefinition{
		ID:          "reasoning-agent",
		Name:        "Reasoning Agent",
		Description: "An agent with advanced reasoning capabilities",
		SystemPrompt: `You are an AI assistant with advanced reasoning capabilities.

When solving complex problems:
1. Break down the problem into smaller components
2. Analyze each component systematically
3. Consider multiple approaches
4. Evaluate trade-offs
5. Provide clear, structured reasoning

Use the reasoning_chain tool for complex problems that require step-by-step analysis.`,
		Model: "claude-3-5-sonnet-20241022",
		Tools: []interface{}{"*"},
	})

	return registry
}
