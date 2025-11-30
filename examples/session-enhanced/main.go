// SessionEnhanced 演示增强版会话管理功能，包括与 Agent 的深度集成、
// 会话状态持久化和多轮对话上下文管理。
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/session"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/builtin"
	"github.com/astercloud/aster/pkg/types"
)

func main() {
	ctx := context.Background()

	// 创建依赖
	jsonStore, err := store.NewJSONStore(".aster-session-enhanced")
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

	// 创建 Agent
	config := &types.AgentConfig{
		TemplateID: "chat-agent",
		ModelConfig: &types.ModelConfig{
			Provider: "anthropic",
			Model:    "claude-3-5-sonnet-20241022",
		},
	}

	ag, err := agent.Create(ctx, config, deps)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	defer func() {
		if err := ag.Close(); err != nil {
			log.Printf("Failed to close agent: %v", err)
		}
	}()

	// 模拟对话
	fmt.Println("=== Simulating Conversation ===")
	fmt.Println()

	conversations := []string{
		"What is the capital of France?",
		"Tell me about the Eiffel Tower",
		"What are some famous French dishes?",
		"How do I make croissants?",
		"What's the history of Paris?",
	}

	for i, msg := range conversations {
		fmt.Printf("User: %s\n", msg)
		result, err := ag.Chat(ctx, msg)
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}
		fmt.Printf("Assistant: %s\n\n", truncate(result.Text, 100))

		// 每隔几条消息生成摘要
		if (i+1)%3 == 0 {
			fmt.Println("--- Generating Summary ---")
			demonstrateSummary(ctx, ag.ID(), jsonStore, deps.ProviderFactory)
			fmt.Println()
		}
	}

	// 演示搜索功能
	fmt.Println()
	fmt.Println("=== Demonstrating Search ===")
	fmt.Println()
	demonstrateSearch(ctx, ag.ID(), jsonStore)

	fmt.Println("\n=== Session Complete ===")
}

func demonstrateSummary(ctx context.Context, agentID string, st store.Store, provFactory provider.Factory) {
	// 加载消息
	messages, err := st.LoadMessages(ctx, agentID)
	if err != nil {
		log.Printf("Failed to load messages: %v", err)
		return
	}

	// 创建 Provider
	prov, err := provFactory.Create(&types.ModelConfig{
		Provider: "anthropic",
		Model:    "claude-3-5-sonnet-20241022",
	})
	if err != nil {
		log.Printf("Failed to create provider: %v", err)
		return
	}
	defer func() {
		if err := prov.Close(); err != nil {
			log.Printf("Failed to close provider: %v", err)
		}
	}()

	// 创建摘要器
	summarizer := session.NewSummarizer(session.SummarizerConfig{
		Provider:               prov,
		MaxMessagesPerCall:     50,
		MinMessagesToSummarize: 3,
	})

	// 生成摘要
	summary, err := summarizer.SummarizeSession(ctx, messages)
	if err != nil {
		log.Printf("Failed to generate summary: %v", err)
		return
	}

	fmt.Printf("Summary (%d messages):\n", summary.MessageCount)
	fmt.Printf("%s\n", truncate(summary.Text, 200))

	if len(summary.KeyTopics) > 0 {
		fmt.Printf("Key Topics: %v\n", summary.KeyTopics)
	}
}

func demonstrateSearch(ctx context.Context, agentID string, st store.Store) {
	// 创建搜索器
	searcher := session.NewSearcher(session.SearcherConfig{
		Store: st,
	})

	// 搜索示例
	searchQueries := []string{
		"Paris",
		"food",
		"Eiffel",
	}

	for _, query := range searchQueries {
		fmt.Printf("Searching for: '%s'\n", query)

		results, err := searcher.SearchHistory(ctx, session.SearchOptions{
			Query:     query,
			AgentID:   agentID,
			Limit:     3,
			MatchMode: "contains",
		})

		if err != nil {
			log.Printf("Search failed: %v", err)
			continue
		}

		fmt.Printf("Found %d results:\n", len(results))
		for i, result := range results {
			fmt.Printf("  %d. [Relevance: %.2f] %s\n",
				i+1, result.Relevance, truncate(result.Snippet, 80))
		}
		fmt.Println()
	}
}

func createTemplateRegistry() *agent.TemplateRegistry {
	registry := agent.NewTemplateRegistry()

	registry.Register(&types.AgentTemplateDefinition{
		ID:           "chat-agent",
		SystemPrompt: "You are a helpful assistant.",
		Model:        "claude-3-5-sonnet-20241022",
		Tools:        []string{},
	})

	return registry
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
