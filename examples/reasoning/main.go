// Reasoning 演示推理中间件的高级推理能力，支持分步骤的问题分解、
// 系统化分析和结构化推理输出。
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/builtin"
	"github.com/astercloud/aster/pkg/types"
)

func main() {
	ctx := context.Background()

	// 创建依赖
	jsonStore, err := store.NewJSONStore(".aster-reasoning")
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}

	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry)

	deps := &agent.Dependencies{
		Store:            jsonStore,
		SandboxFactory:   sandbox.NewFactory(),
		ToolRegistry:     toolRegistry,
		ProviderFactory:  provider.NewMultiProviderFactory(),
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
		MiddlewareConfig: map[string]map[string]any{
			"reasoning": {
				"enabled":        true,
				"min_steps":      2,
				"max_steps":      5,
				"min_confidence": 0.7,
				"use_json":       true,
			},
		},
		Metadata: map[string]any{
			"enable_reasoning": true,
		},
	}

	// 创建 Agent
	ag, err := agent.Create(ctx, config, deps)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	defer func() {
		if err := ag.Close(); err != nil {
			log.Printf("Failed to close agent: %v", err)
		}
	}()

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
		ID: "reasoning-agent",
		SystemPrompt: `You are an AI assistant with advanced reasoning capabilities.

When solving complex problems:
1. Break down the problem into smaller components
2. Analyze each component systematically
3. Consider multiple approaches
4. Evaluate trade-offs
5. Provide clear, structured reasoning

Use the reasoning_chain tool for complex problems that require step-by-step analysis.`,
		Model: "claude-3-5-sonnet-20241022",
		Tools: "*",
	})

	return registry
}
