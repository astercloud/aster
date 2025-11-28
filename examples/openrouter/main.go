// Package main 演示如何使用 OpenRouter 作为 LLM Provider
package main

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/builtin"
	"github.com/astercloud/aster/pkg/types"
)

func main() {
	cmd := &cli.Command{
		Name:  "openrouter-agent",
		Usage: "OpenRouter Agent 演示程序",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "print",
				Aliases: []string{"p"},
				Usage:   "非交互模式：直接执行指定提示词并退出",
			},
			&cli.BoolFlag{
				Name:    "stream",
				Aliases: []string{"s"},
				Usage:   "使用流式模式（实时输出）",
			},
			&cli.StringFlag{
				Name:    "model",
				Aliases: []string{"m"},
				Value:   "anthropic/claude-haiku-4.5",
				Usage:   "指定模型",
			},
		},
		Action: run,
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		slog.Error("运行失败", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cmd *cli.Command) error {
	// 检查 API Key
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("需要设置 OPENROUTER_API_KEY 环境变量")
	}

	baseURL := os.Getenv("OPENROUTER_BASE_URL")
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}

	// 获取参数
	prompt := cmd.String("print")
	streaming := cmd.Bool("stream")
	model := cmd.String("model")

	// 创建 Agent
	ag, err := createAgent(apiKey, baseURL, model, streaming)
	if err != nil {
		return fmt.Errorf("创建 Agent 失败: %w", err)
	}
	defer func() { _ = ag.Close() }()

	// 非交互模式
	if prompt != "" {
		return runOnce(ctx, ag, prompt)
	}

	// 交互模式
	return runInteractive(ctx, ag)
}

// runOnce 非交互模式：执行单次对话并退出
func runOnce(ctx context.Context, ag *agent.Agent, prompt string) error {
	// 订阅事件以捕获文本输出
	var textOutput strings.Builder
	eventCh := ag.Subscribe([]types.AgentChannel{types.ChannelProgress}, nil)

	done := make(chan struct{})
	go func() {
		for envelope := range eventCh {
			switch e := envelope.Event.(type) {
			case *types.ProgressTextChunkEvent:
				textOutput.WriteString(e.Delta)
			}
		}
		close(done)
	}()

	result, err := ag.Chat(ctx, prompt)
	ag.Unsubscribe(eventCh)
	<-done

	if err != nil {
		return fmt.Errorf("对话失败: %w", err)
	}

	// 优先使用事件流收集的文本，其次使用 result.Text
	output := textOutput.String()
	if output == "" {
		output = result.Text
	}

	if output != "" {
		fmt.Println(output)
	} else {
		fmt.Println("[完成]")
	}

	return nil
}

// runInteractive 交互模式：REPL 循环
func runInteractive(ctx context.Context, ag *agent.Agent) error {
	slog.Info("Agent 创建成功", "id", ag.ID())

	// 订阅事件
	eventCh := ag.Subscribe([]types.AgentChannel{types.ChannelProgress}, nil)
	go handleEvents(eventCh)

	fmt.Println("\n🤖 OpenRouter Agent 演示")
	fmt.Println("输入消息与 Agent 对话，输入 'quit' 退出")
	fmt.Println(strings.Repeat("-", 50))

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\n> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if input == "quit" || input == "exit" {
			fmt.Println("👋 再见!")
			break
		}

		result, err := ag.Chat(ctx, input)
		if err != nil {
			slog.Error("对话失败", "error", err)
			continue
		}

		// 显示响应文本
		if result.Text != "" {
			fmt.Printf("\n%s\n", result.Text)
		}
		fmt.Printf("[状态: %s]\n", result.Status)
	}

	return nil
}

// createAgent 创建并配置 Agent
func createAgent(apiKey, baseURL, model string, streaming bool) (*agent.Agent, error) {
	// 工具注册
	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry)

	// 持久化存储
	jsonStore, err := store.NewJSONStore(".aster")
	if err != nil {
		return nil, fmt.Errorf("创建 Store 失败: %w", err)
	}

	// Agent 模板
	templateRegistry := agent.NewTemplateRegistry()
	templateRegistry.Register(&types.AgentTemplateDefinition{
		ID:           "simple-assistant",
		SystemPrompt: "你是一个有用的助手，可以读取和写入文件。当用户要求你读取或写入文件时，请使用可用的工具。",
		Tools:        []interface{}{"Read", "Write", "Bash"},
	})

	// 依赖注入
	deps := &agent.Dependencies{
		Store:            jsonStore,
		SandboxFactory:   sandbox.NewFactory(),
		ToolRegistry:     toolRegistry,
		ProviderFactory:  &provider.OpenRouterFactory{},
		TemplateRegistry: templateRegistry,
	}

	// 执行模式
	execMode := types.ExecutionModeNonStreaming
	if streaming {
		execMode = types.ExecutionModeStreaming
	}

	// Agent 配置
	config := &types.AgentConfig{
		TemplateID: "simple-assistant",
		ModelConfig: &types.ModelConfig{
			Provider:      "openrouter",
			Model:         model,
			APIKey:        apiKey,
			BaseURL:       baseURL,
			ExecutionMode: execMode,
		},
		Sandbox: &types.SandboxConfig{
			Kind:    types.SandboxKindLocal,
			WorkDir: "./workspace",
		},
	}

	return agent.Create(context.TODO(), config, deps)
}

// handleEvents 处理 Agent 事件流
func handleEvents(eventCh <-chan types.AgentEventEnvelope) {
	for envelope := range eventCh {
		switch e := envelope.Event.(type) {
		case *types.ProgressToolStartEvent:
			fmt.Printf("\n🔧 [工具] %s 开始执行...\n", e.Call.Name)
		case *types.ProgressToolEndEvent:
			fmt.Printf("✅ [工具] %s 完成\n", e.Call.Name)
		case *types.ProgressToolErrorEvent:
			fmt.Printf("❌ [错误] %s: %s\n", e.Call.Name, e.Error)
		case *types.ProgressDoneEvent:
			fmt.Printf("📝 [完成] 步骤 %d\n", e.Step)
		}
	}
}
