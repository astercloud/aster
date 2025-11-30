// ModelFallback 演示模型降级策略，当主模型不可用时自动切换到备选模型。
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/types"
)

func main() {
	fmt.Println("=== Model Fallback 示例 ===")
	fmt.Println()

	// 创建 Provider Factory
	factory := provider.NewMultiProviderFactory()

	// 创建 Dependencies
	deps := &agent.Dependencies{
		ProviderFactory: factory,
	}

	// 配置模型降级策略
	fallbacks := []*agent.ModelFallback{
		{
			Config: &types.ModelConfig{
				Provider: "anthropic",
				Model:    "claude-sonnet-4-5",
				APIKey:   "your-anthropic-api-key",
			},
			MaxRetries: 2,
			Enabled:    true,
			Priority:   1, // 最高优先级
		},
		{
			Config: &types.ModelConfig{
				Provider: "openai",
				Model:    "gpt-4o",
				APIKey:   "your-openai-api-key",
			},
			MaxRetries: 1,
			Enabled:    true,
			Priority:   2, // 备用模型
		},
		{
			Config: &types.ModelConfig{
				Provider: "deepseek",
				Model:    "deepseek-chat",
				APIKey:   "your-deepseek-api-key",
			},
			MaxRetries: 0,
			Enabled:    true,
			Priority:   3, // 最后的备用
		},
	}

	// 创建 Model Fallback Manager
	manager, err := agent.NewModelFallbackManager(fallbacks, deps)
	if err != nil {
		log.Fatalf("Failed to create manager: %v", err)
	}

	// 示例 1: 非流式请求
	fmt.Println("示例 1: 非流式请求")
	fmt.Println("---")

	ctx := context.Background()
	messages := []types.Message{
		{
			Role:    "user",
			Content: "请用一句话介绍人工智能。",
		},
	}

	resp, err := manager.Complete(ctx, messages, &provider.StreamOptions{
		MaxTokens:   100,
		Temperature: 0.7,
	})
	if err != nil {
		log.Fatalf("Complete failed: %v", err)
	}

	fmt.Printf("响应: %s\n", resp.Message.Content)
	fmt.Println()

	// 示例 2: 流式请求
	fmt.Println("示例 2: 流式请求")
	fmt.Println("---")

	stream, err := manager.Stream(ctx, messages, &provider.StreamOptions{
		MaxTokens:   100,
		Temperature: 0.7,
	})
	if err != nil {
		log.Fatalf("Stream failed: %v", err)
	}

	fmt.Print("响应: ")
	for chunk := range stream {
		if chunk.Type == "text" {
			fmt.Print(chunk.TextDelta)
		}
	}
	fmt.Println()
	fmt.Println()

	// 示例 3: 查看统计信息
	fmt.Println("示例 3: 统计信息")
	fmt.Println("---")

	stats := manager.GetStats()
	fmt.Printf("总请求数: %d\n", stats.TotalRequests)
	fmt.Printf("成功请求数: %d\n", stats.SuccessRequests)
	fmt.Printf("失败请求数: %d\n", stats.FailedRequests)
	fmt.Printf("降级次数: %d\n", stats.FallbackCount)
	fmt.Println("\n模型使用统计:")
	for model, count := range stats.ModelUsageCount {
		fmt.Printf("  %s: %d 次\n", model, count)
	}
	fmt.Println()

	// 示例 4: 动态启用/禁用模型
	fmt.Println("示例 4: 动态管理模型")
	fmt.Println("---")

	// 列出所有模型
	models := manager.ListModels()
	fmt.Println("当前模型列表:")
	for _, m := range models {
		status := "禁用"
		if m["enabled"].(bool) {
			status = "启用"
		}
		current := ""
		if m["is_current"].(bool) {
			current = " (当前使用)"
		}
		fmt.Printf("  [%s] %s/%s - 优先级: %d, 重试: %d%s\n",
			status,
			m["provider"],
			m["model"],
			m["priority"],
			m["max_retries"],
			current,
		)
	}
	fmt.Println()

	// 禁用主模型
	fmt.Println("禁用主模型 (anthropic/claude-sonnet-4-5)...")
	err = manager.DisableModel("anthropic", "claude-sonnet-4-5")
	if err != nil {
		log.Printf("Failed to disable model: %v", err)
	}

	// 再次请求，应该使用备用模型
	resp, err = manager.Complete(ctx, messages, nil)
	if err != nil {
		log.Fatalf("Complete failed: %v", err)
	}
	fmt.Printf("使用备用模型的响应: %s\n\n", resp.Message.Content)

	// 重新启用主模型
	fmt.Println("重新启用主模型...")
	err = manager.EnableModel("anthropic", "claude-sonnet-4-5")
	if err != nil {
		log.Printf("Failed to enable model: %v", err)
	}

	// 示例 5: 错误处理和重试
	fmt.Println("示例 5: 自动重试和降级")
	fmt.Println("---")
	fmt.Println("当主模型失败时，系统会自动:")
	fmt.Println("1. 重试主模型（根据 MaxRetries 配置）")
	fmt.Println("2. 如果所有重试都失败，自动降级到下一个模型")
	fmt.Println("3. 重复此过程直到成功或所有模型都失败")
	fmt.Println()

	// 最终统计
	fmt.Println("=== 最终统计 ===")
	stats = manager.GetStats()
	fmt.Printf("总请求数: %d\n", stats.TotalRequests)
	fmt.Printf("成功率: %.2f%%\n", float64(stats.SuccessRequests)/float64(stats.TotalRequests)*100)
	fmt.Printf("降级次数: %d\n", stats.FallbackCount)
}
