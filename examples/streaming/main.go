// Streaming 演示基于 Go 1.23 iter.Seq2 迭代器的 Agent 流式 API，
// 包括事件收集、过滤和内存高效的实时处理。
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
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

	// 1. 创建 Agent 依赖
	deps := createDependencies()

	// 2. 创建 Agent 配置
	config := &types.AgentConfig{
		TemplateID: "default",
		ModelConfig: &types.ModelConfig{
			Provider: "anthropic",
			Model:    "claude-3-5-sonnet-20241022",
		},
		Sandbox: &types.SandboxConfig{
			Kind:    types.SandboxKindLocal,
			WorkDir: ".",
		},
		Tools: []string{"Bash", "Read"},
	}

	// 3. 创建 Agent
	ag, err := agent.Create(ctx, config, deps)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// ====== 示例 1: 流式处理事件 ======
	fmt.Println("=== Example 1: Streaming Events ===")
	streamingExample(ctx, ag)

	// ====== 示例 2: 收集所有事件 ======
	fmt.Println("\n=== Example 2: Collect All Events ===")
	collectExample(ctx, ag)

	// ====== 示例 3: 过滤事件 ======
	fmt.Println("\n=== Example 3: Filter Events ===")
	filterExample(ctx, ag)

	// ====== 示例 4: 获取最后一个事件 ======
	fmt.Println("\n=== Example 4: Get Last Event ===")
	lastEventExample(ctx, ag)
}

// streamingExample 演示流式处理事件
func streamingExample(ctx context.Context, ag *agent.Agent) {
	// 使用 Recv 循环迭代事件流
	reader := ag.Stream(ctx, "What is 2+2?")
	for {
		event, err := reader.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			log.Printf("Error: %v", err)
			break
		}

		// 实时处理每个事件
		fmt.Printf("[Event %s] Author: %s, Content: %s\n",
			event.ID[:8],
			event.Author,
			truncateContent(event.Content.Content, 50),
		)

		// 可以根据条件中断流
		if event.Actions.Escalate {
			fmt.Println("Escalation detected, stopping stream")
			break
		}
	}
}

// collectExample 演示收集所有事件
func collectExample(ctx context.Context, ag *agent.Agent) {
	// 收集所有事件到切片
	events, err := agent.StreamCollect(ag.Stream(ctx, "List files in current directory"))
	if err != nil {
		log.Fatalf("Stream error: %v", err)
	}

	fmt.Printf("Total events collected: %d\n", len(events))
	for i, event := range events {
		fmt.Printf("[%d] %s: %s\n", i+1, event.Author, truncateContent(event.Content.Content, 30))
	}
}

// filterExample 演示过滤事件
func filterExample(ctx context.Context, ag *agent.Agent) {
	// 只处理来自 assistant 的事件
	reader := agent.StreamFilter(
		ag.Stream(ctx, "Tell me a joke"),
		func(event *session.Event) bool {
			return event.Author == "assistant"
		},
	)

	for {
		event, err := reader.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			log.Printf("Error: %v", err)
			break
		}
		fmt.Printf("Assistant says: %s\n", event.Content.Content)
	}
}

// lastEventExample 演示获取最后一个事件
func lastEventExample(ctx context.Context, ag *agent.Agent) {
	// 获取最后一个事件（最终响应）
	lastEvent, err := agent.StreamLast(ag.Stream(ctx, "What's the weather?"))
	if err != nil {
		log.Fatalf("Stream error: %v", err)
	}

	fmt.Printf("Final response: %s\n", lastEvent.Content.Content)
}

// 辅助函数

func createDependencies() *agent.Dependencies {
	// 创建模板注册表
	templateRegistry := agent.NewTemplateRegistry()
	templateRegistry.Register(&types.AgentTemplateDefinition{
		ID:           "default",
		SystemPrompt: "You are a helpful AI assistant.",
		Model:        "claude-3-5-sonnet-20241022",
		Tools:        []any{"*"},
	})

	// 创建工具注册表和存储
	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry)

	jsonStore, _ := store.NewJSONStore("./.aster")

	// 创建 Provider 工厂
	providerFactory := &provider.AnthropicFactory{}

	// 创建 Sandbox 工厂
	sandboxFactory := sandbox.NewFactory()

	return &agent.Dependencies{
		TemplateRegistry: templateRegistry,
		ToolRegistry:     toolRegistry,
		ProviderFactory:  providerFactory,
		SandboxFactory:   sandboxFactory,
		Store:            jsonStore,
	}
}

func truncateContent(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "..."
}

// 性能对比示例

var _ = func() {
	// 传统方式 - 加载所有事件到内存
	// events := []Event{...}  // 可能包含数千个事件
	// for _, event := range events {
	//     process(event)
	// }

	// 流式方式 - 按需生成
	// for event, err := range agent.Stream(ctx, msg) {
	//     process(event)  // 只在需要时生成和处理
	//     if someCondition {
	//         break  // 可以提前中断
	//     }
	// }

	// 优势：
	// 1. 内存占用：O(1) vs O(n)
	// 2. 首字节延迟：立即 vs 等待所有事件
	// 3. 可控性：支持背压和取消
}
