// Agent 演示核心 Agent API，包括工具注册、事件订阅和多轮对话能力。
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/builtin"
	"github.com/astercloud/aster/pkg/types"
)

func main() {
	// 检查API Key
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("ANTHROPIC_API_KEY environment variable is required")
	}

	// 创建上下文
	ctx := context.Background()

	// 1. 创建工具注册表并注册内置工具
	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry)

	// 2. 创建Sandbox工厂
	sandboxFactory := sandbox.NewFactory()

	// 3. 创建Provider工厂
	providerFactory := &provider.AnthropicFactory{}

	// 4. 创建Store
	storePath := ".aster"
	jsonStore, err := store.NewJSONStore(storePath)
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}

	// 5. 创建模板注册表
	templateRegistry := agent.NewTemplateRegistry()

	// 注册一个简单的助手模板
	templateRegistry.Register(&types.AgentTemplateDefinition{
		ID:           "simple-assistant",
		Model:        "claude-sonnet-4-5",
		SystemPrompt: "You are a helpful assistant that can read and write files. When users ask you to read or write files, use the available tools.",
		Tools:        []any{"Read", "Write", "Bash"},
	})

	// 6. 创建依赖
	deps := &agent.Dependencies{
		Store:            jsonStore,
		SandboxFactory:   sandboxFactory,
		ToolRegistry:     toolRegistry,
		ProviderFactory:  providerFactory,
		TemplateRegistry: templateRegistry,
	}

	// 7. 创建Agent配置
	config := &types.AgentConfig{
		TemplateID: "simple-assistant",
		ModelConfig: &types.ModelConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-5",
			APIKey:   apiKey,
		},
		Sandbox: &types.SandboxConfig{
			Kind:    types.SandboxKindLocal,
			WorkDir: "./workspace",
		},
	}

	// 8. 创建Agent
	ag, err := agent.Create(ctx, config, deps)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	defer func() { _ = ag.Close() }()

	fmt.Printf("Agent created: %s\n", ag.ID())

	// 9. 订阅事件
	eventCh := ag.Subscribe([]types.AgentChannel{
		types.ChannelProgress,
		types.ChannelMonitor,
	}, nil)

	// 启动事件监听
	go func() {
		for envelope := range eventCh {
			// Event 必须实现 EventType 接口才能获取 Channel
			if evt, ok := envelope.Event.(types.EventType); ok {
				switch evt.Channel() {
				case types.ChannelProgress:
					handleProgressEvent(envelope.Event)
				case types.ChannelMonitor:
					handleMonitorEvent(envelope.Event)
				}
			}
		}
	}()

	// 10. 发送消息并等待完成
	fmt.Println("\n--- Test 1: Create a test file ---")
	result, err := ag.Chat(ctx, "Please create a file called test.txt with content 'Hello World'")
	if err != nil {
		log.Fatalf("Chat failed: %v", err)
	}
	fmt.Printf("\nAssistant: %s\n", result.Text)

	// 等待一下确保事件处理完成
	time.Sleep(1 * time.Second)

	fmt.Println("\n--- Test 2: Read the file back ---")
	result, err = ag.Chat(ctx, "Please read the test.txt file")
	if err != nil {
		log.Fatalf("Chat failed: %v", err)
	}
	fmt.Printf("\nAssistant: %s\n", result.Text)

	time.Sleep(1 * time.Second)

	fmt.Println("\n--- Test 3: Run a bash command ---")
	result, err = ag.Chat(ctx, "Please run 'ls -la' command")
	if err != nil {
		log.Fatalf("Chat failed: %v", err)
	}
	fmt.Printf("\nAssistant: %s\n", result.Text)

	// 输出状态
	status := ag.Status()
	fmt.Printf("\n\nFinal Status:\n")
	fmt.Printf("  Agent ID: %s\n", status.AgentID)
	fmt.Printf("  State: %s\n", status.State)
	fmt.Printf("  Steps: %d\n", status.StepCount)
	fmt.Printf("  Cursor: %d\n", status.Cursor)
}

func handleProgressEvent(event any) {
	switch e := event.(type) {
	case *types.ProgressTextChunkEvent:
		fmt.Print(e.Delta)
	case *types.ProgressTextChunkStartEvent:
		fmt.Print("\n[Assistant] ")
	case *types.ProgressTextChunkEndEvent:
		// 文本块结束
	case *types.ProgressToolStartEvent:
		fmt.Printf("\n[Tool Start] %s (ID: %s)\n", e.Call.Name, e.Call.ID)
	case *types.ProgressToolEndEvent:
		fmt.Printf("[Tool End] %s - State: %s\n", e.Call.Name, e.Call.State)
	case *types.ProgressToolErrorEvent:
		fmt.Printf("[Tool Error] %s - Error: %s\n", e.Call.Name, e.Error)
	case *types.ProgressDoneEvent:
		fmt.Printf("\n[Done] Step %d - Reason: %s\n", e.Step, e.Reason)
	}
}

func handleMonitorEvent(event any) {
	switch e := event.(type) {
	case *types.MonitorStateChangedEvent:
		fmt.Printf("[State Changed] %s\n", e.State)
	case *types.MonitorTokenUsageEvent:
		fmt.Printf("[Token Usage] Input: %d, Output: %d, Total: %d\n",
			e.InputTokens, e.OutputTokens, e.TotalTokens)
	case *types.MonitorErrorEvent:
		fmt.Printf("[Error] [%s] %s: %s\n", e.Severity, e.Phase, e.Message)
	case *types.MonitorBreakpointChangedEvent:
		fmt.Printf("[Breakpoint] %v -> %v\n", e.Previous, e.Current)
	}
}
