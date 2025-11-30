// MemoryAgent 演示带长期记忆的 Agent，使用 FilesystemMiddleware 和
// AgentMemoryMiddleware 实现跨会话的用户信息记忆和检索。
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
	// 检查 API Key
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("ANTHROPIC_API_KEY environment variable is required")
	}

	ctx := context.Background()

	// 1. 工具注册表
	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry)

	// 2. Sandbox
	sandboxFactory := sandbox.NewFactory()

	// 3. Provider 工厂
	providerFactory := &provider.AnthropicFactory{}

	// 4. Store
	jsonStore, err := store.NewJSONStore(".aster-memory-agent")
	if err != nil {
		log.Fatalf("create store failed: %v", err)
	}

	// 5. 模板注册表
	templateRegistry := agent.NewTemplateRegistry()
	templateRegistry.Register(&types.AgentTemplateDefinition{
		ID: "memory-assistant",
		// SystemPrompt 仅描述职责, 具体的长期记忆说明由 AgentMemoryMiddleware 注入
		SystemPrompt: "You are an assistant with file access and long-term memory. " +
			"Always prefer reading from and writing to memory files when users ask to remember or recall information.",
		Model: "claude-sonnet-4-5",
		// 这里只配置基础工具; memory_* 工具由中间件自动注入
		Tools: []any{"Read", "Write", "Bash"},
	})

	deps := &agent.Dependencies{
		Store:            jsonStore,
		SandboxFactory:   sandboxFactory,
		ToolRegistry:     toolRegistry,
		ProviderFactory:  providerFactory,
		TemplateRegistry: templateRegistry,
	}

	// 6. 为 Sandbox 工作目录准备本地路径
	workDir := "./workspace"
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		log.Fatalf("create workspace dir failed: %v", err)
	}

	// 7. 创建 Agent 配置, 启用 filesystem + agent_memory 中间件
	config := &types.AgentConfig{
		TemplateID: "memory-assistant",
		ModelConfig: &types.ModelConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-5",
			APIKey:   apiKey,
		},
		Sandbox: &types.SandboxConfig{
			Kind:    types.SandboxKindLocal,
			WorkDir: workDir,
		},
		// 启用中间件: filesystem + agent_memory
		Middlewares: []string{"filesystem", "agent_memory"},
	}

	// 8. 创建 Agent
	ag, err := agent.Create(ctx, config, deps)
	if err != nil {
		log.Fatalf("create agent failed: %v", err)
	}
	defer func() { _ = ag.Close() }()

	fmt.Printf("✅ Memory Agent created: %s\n", ag.ID())

	// 9. 订阅事件（仅简单展示文本输出）
	eventCh := ag.Subscribe([]types.AgentChannel{types.ChannelProgress}, nil)
	go func() {
		for envelope := range eventCh {
			if evt, ok := envelope.Event.(types.EventType); ok {
				switch e := evt.(type) {
				case *types.ProgressTextChunkStartEvent:
					fmt.Print("\n[Assistant] ")
				case *types.ProgressTextChunkEvent:
					fmt.Print(e.Delta)
				case *types.ProgressDoneEvent:
					fmt.Printf("\n[Done] Step %d (%s)\n", e.Step, e.Reason)
				}
			}
		}
	}()

	// 10. 对话 1: 让 Agent 记录一条用户偏好到记忆中
	fmt.Println("\n--- Conversation 1: 记录偏好到长期记忆 ---")
	prompt1 := `
我叫 Alice, 以后请记住:
- 我喜欢 grep 风格的代码搜索
- 我喜欢简洁的代码 diff
- 遇到复杂问题时, 先搜索已有记忆再回答

请把这些信息保存到你的长期记忆中, 并告诉我你保存到了哪个记忆文件。`

	if _, err := ag.Chat(ctx, prompt1); err != nil {
		log.Fatalf("chat 1 failed: %v", err)
	}

	time.Sleep(2 * time.Second)

	// 11. 对话 2: 在新问题中检索记忆
	fmt.Println("\n--- Conversation 2: 基于长期记忆回答问题 ---")
	prompt2 := `
还记得我对代码工具的偏好吗? 请先在你的长期记忆中搜索相关记录, 然后用一小段话总结出来回答我。`

	if _, err := ag.Chat(ctx, prompt2); err != nil {
		log.Fatalf("chat 2 failed: %v", err)
	}

	time.Sleep(2 * time.Second)

	// 结束状态
	status := ag.Status()
	fmt.Printf("\nFinal Status: steps=%d, cursor=%d, state=%s\n",
		status.StepCount, status.Cursor, status.State)
}
