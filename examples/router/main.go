// Router 演示 StaticRouter 的模型路由功能，根据不同的 routing_profile
// 为 Agent 选择不同的模型配置，如 "quality-first" 或 "cost-first"。
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/router"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/builtin"
	"github.com/astercloud/aster/pkg/types"
)

func main() {
	ctx := context.Background()

	// 简单的JSON存储 + 本地沙箱
	memStore, err := store.NewJSONStore("./.aster-router")
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}
	sbFactory := sandbox.NewFactory()

	// Provider 工厂
	providerFactory := provider.NewMultiProviderFactory()

	// 模板注册：同一个模板 ID，但可以通过 RoutingProfile 决定实际模型
	tplRegistry := agent.NewTemplateRegistry()
	tplRegistry.Register(&types.AgentTemplateDefinition{
		ID:           "router-demo",
		SystemPrompt: "You are a helpful assistant.",
		Model:        "claude-3-5-sonnet-20241022",
		Tools:        []string{},
	})

	// 构造静态路由器：
	// - quality-first: 使用 anthropic 的较强模型（示例）
	// - cost-first: 使用 deepseek 的模型（示例）
	defaultModel := &types.ModelConfig{
		Provider: "anthropic",
		Model:    "claude-3-5-sonnet-20241022",
	}

	routes := []router.StaticRouteEntry{
		{
			Task:     "chat",
			Priority: router.PriorityQuality,
			Model: &types.ModelConfig{
				Provider: "anthropic",
				Model:    "claude-3-5-sonnet-20241022",
			},
		},
		{
			Task:     "chat",
			Priority: router.PriorityCost,
			Model: &types.ModelConfig{
				Provider: "deepseek",
				Model:    "deepseek-chat",
			},
		},
	}

	rt := router.NewStaticRouter(defaultModel, routes)

	deps := &agent.Dependencies{
		Store:            memStore,
		SandboxFactory:   sbFactory,
		ToolRegistry:     toolsRegistry(),
		ProviderFactory:  providerFactory,
		Router:           rt,
		TemplateRegistry: tplRegistry,
	}

	// 创建一个 “quality-first” Agent
	qualityAgent, err := agent.Create(ctx, &types.AgentConfig{
		TemplateID:     "router-demo",
		RoutingProfile: string(router.PriorityQuality),
		Metadata: map[string]any{
			"demo": "quality-first",
		},
	}, deps)
	if err != nil {
		log.Fatalf("create quality-first agent: %v", err)
	}
	defer func() { _ = qualityAgent.Close() }()

	// 创建一个 “cost-first” Agent
	costAgent, err := agent.Create(ctx, &types.AgentConfig{
		TemplateID:     "router-demo",
		RoutingProfile: string(router.PriorityCost),
		Metadata: map[string]any{
			"demo": "cost-first",
		},
	}, deps)
	if err != nil {
		log.Fatalf("create cost-first agent: %v", err)
	}
	defer func() { _ = costAgent.Close() }()

	// 简单调用，打印结果（真正的模型调用依赖你在环境中配置好对应 API Key）
	runDemo := func(name string, a *agent.Agent) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		fmt.Printf("==== %s ====\n", name)
		res, err := a.Chat(ctx, "简单介绍一下你当前使用的模型（可以虚构回答）。")
		if err != nil {
			log.Printf("%s error: %v\n", name, err)
			return
		}
		fmt.Printf("%s reply:\n%s\n\n", name, res.Text)
	}

	runDemo("quality-first", qualityAgent)
	runDemo("cost-first", costAgent)
}

// toolsRegistry 返回一个空的工具注册表，占位用。
// 如果你希望在路由示例中启用具体工具，可以在这里注册。
func toolsRegistry() *tools.Registry {
	reg := tools.NewRegistry()
	builtin.RegisterAll(reg)
	return reg
}
