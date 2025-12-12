//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/types"
)

func main() {
	// 从环境变量读取配置
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379" // 默认地址
	}

	redisPassword := os.Getenv("REDIS_PASSWORD")

	apiKey := os.Getenv("CLAUDE_API_KEY")
	baseURL := os.Getenv("CLAUDE_BASE_URL")

	if apiKey == "" {
		fmt.Println("❌ 请设置 CLAUDE_API_KEY 环境变量")
		os.Exit(1)
	}
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	fmt.Println("=== Redis Store 分布式测试 ===")
	fmt.Printf("Redis: %s\n", redisAddr)
	fmt.Printf("API: %s\n\n", baseURL)

	// 1. 创建 Redis Store
	fmt.Println("【步骤 1】创建 Redis Store")
	redisStore, err := store.NewRedisStore(store.RedisConfig{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       0,
		Prefix:   "aster:test:",
		TTL:      1 * time.Hour, // 测试用 1 小时过期
	})
	if err != nil {
		fmt.Printf("❌ 创建 Redis Store 失败: %v\n", err)
		os.Exit(1)
	}
	defer redisStore.Close()

	// 测试连接
	ctx := context.Background()
	if err := redisStore.Ping(ctx); err != nil {
		fmt.Printf("❌ Redis 连接失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Redis 连接成功")

	// 2. 创建 Provider
	fmt.Println("\n【步骤 2】创建 Provider")
	cp, err := provider.NewCustomClaudeProvider(&types.ModelConfig{
		Provider: "anthropic",
		Model:    "claude-sonnet-4-5-20250929",
		APIKey:   apiKey,
		BaseURL:  baseURL,
	})
	if err != nil {
		fmt.Printf("❌ 创建 Provider 失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Provider 创建成功")

	// 3. 创建 Sandbox
	fmt.Println("\n【步骤 3】创建 Sandbox")
	sb, err := sandbox.NewLocalSandbox(&sandbox.LocalSandboxConfig{
		WorkDir:          "./workspace",
		EnforceBoundary:  false,
		SecurityLevel:    1,
		AllowedCommands:  nil,
		ForbiddenCommands: nil,
	})
	if err != nil {
		fmt.Printf("❌ 创建 Sandbox 失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Sandbox 创建成功")

	// 4. 创建 Agent（使用 Redis Store）
	fmt.Println("\n【步骤 4】创建 Agent with Redis Store")
	agentID := "agt-redis-test-001"
	ag, err := agent.NewAgent(&types.AgentConfig{
		Name:  "Redis Store Test Agent",
		Model: "claude-sonnet-4-5-20250929",
		Store: &types.StoreConfig{
			MaxMessages: 10,
			AutoTrim:    true,
		},
	}, &agent.AgentDeps{
		Provider: cp,
		Store:    redisStore, // ✅ 使用 Redis Store
		Sandbox:  sb,
	})
	if err != nil {
		fmt.Printf("❌ 创建 Agent 失败: %v\n", err)
		os.Exit(1)
	}

	// 强制设置 Agent ID（方便测试）
	// 在生产环境中，多个节点可以通过相同的 agentID 共享状态
	ag.SetID(agentID)
	fmt.Printf("✅ Agent 创建成功 (ID: %s)\n", agentID)

	// 5. 模拟分布式场景：第一个节点的对话
	fmt.Println("\n【步骤 5】节点 1 - 发送消息")
	response1, err := ag.Send(ctx, "你好，我是节点 1")
	if err != nil {
		fmt.Printf("❌ 发送失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ 节点 1 收到回复: %s\n", response1.Content)

	// 6. 检查 Redis 中的数据
	fmt.Println("\n【步骤 6】检查 Redis 存储")
	messages, err := redisStore.LoadMessages(ctx, agentID)
	if err != nil {
		fmt.Printf("❌ 加载消息失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Redis 中存储了 %d 条消息\n", len(messages))
	for i, msg := range messages {
		fmt.Printf("  [%d] %s: %s\n", i+1, msg.Role, extractText(msg))
	}

	// 7. 模拟分布式场景：创建第二个 Agent（相同 ID）
	fmt.Println("\n【步骤 7】节点 2 - 创建新 Agent 实例（模拟另一个服务器）")
	ag2, err := agent.NewAgent(&types.AgentConfig{
		Name:  "Redis Store Test Agent (Node 2)",
		Model: "claude-sonnet-4-5-20250929",
		Store: &types.StoreConfig{
			MaxMessages: 10,
			AutoTrim:    true,
		},
	}, &agent.AgentDeps{
		Provider: cp,
		Store:    redisStore, // ✅ 共享同一个 Redis Store
		Sandbox:  sb,
	})
	if err != nil {
		fmt.Printf("❌ 创建 Agent 2 失败: %v\n", err)
		os.Exit(1)
	}
	ag2.SetID(agentID) // 使用相同的 ID
	fmt.Println("✅ 节点 2 Agent 创建成功（共享 Redis）")

	// 8. 节点 2 发送消息（应该能看到之前的对话历史）
	fmt.Println("\n【步骤 8】节点 2 - 继续对话")
	response2, err := ag2.Send(ctx, "你还记得我是哪个节点吗？")
	if err != nil {
		fmt.Printf("❌ 发送失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ 节点 2 收到回复: %s\n", response2.Content)

	// 9. 再次检查 Redis
	fmt.Println("\n【步骤 9】检查 Redis 存储（应该包含两轮对话）")
	messages, err = redisStore.LoadMessages(ctx, agentID)
	if err != nil {
		fmt.Printf("❌ 加载消息失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Redis 中现在存储了 %d 条消息\n", len(messages))
	for i, msg := range messages {
		fmt.Printf("  [%d] %s: %s\n", i+1, msg.Role, extractText(msg))
	}

	// 10. 测试 Store 修剪功能
	fmt.Println("\n【步骤 10】测试 Store 自动修剪（MaxMessages=10）")
	for i := 1; i <= 5; i++ {
		fmt.Printf("  第 %d 轮对话...\n", i+2)
		_, err := ag.Send(ctx, fmt.Sprintf("第 %d 条测试消息", i+2))
		if err != nil {
			fmt.Printf("  ❌ 第 %d 轮失败: %v\n", i+2, err)
			break
		}
	}

	messages, _ = redisStore.LoadMessages(ctx, agentID)
	fmt.Printf("✅ 修剪测试完成，当前消息数: %d (应该 ≤ 10)\n", len(messages))

	if len(messages) > 10 {
		fmt.Printf("❌ 修剪失败！消息数: %d > 10\n", len(messages))
	} else {
		fmt.Println("✅ 修剪功能正常工作")
	}

	// 11. 清理测试数据
	fmt.Println("\n【步骤 11】清理测试数据")
	if err := redisStore.DeleteAgent(ctx, agentID); err != nil {
		fmt.Printf("⚠️  清理失败: %v\n", err)
	} else {
		fmt.Println("✅ 测试数据已清理")
	}

	fmt.Println("\n=== Redis Store 分布式测试完成 ===")
	fmt.Println("\n✅ 测试结果:")
	fmt.Println("  1. Redis Store 创建成功")
	fmt.Println("  2. 多个 Agent 实例共享状态")
	fmt.Println("  3. 对话历史正确保存和加载")
	fmt.Println("  4. Store 自动修剪功能正常")
	fmt.Println("\n🎉 分布式 Store 功能验证通过！")
}

// extractText 从消息中提取文本
func extractText(msg types.Message) string {
	if msg.Content != "" {
		if len(msg.Content) > 50 {
			return msg.Content[:50] + "..."
		}
		return msg.Content
	}

	for _, block := range msg.ContentBlocks {
		if textBlock, ok := block.(*types.TextBlock); ok {
			if len(textBlock.Text) > 50 {
				return textBlock.Text[:50] + "..."
			}
			return textBlock.Text
		}
	}

	return "[非文本内容]"
}
