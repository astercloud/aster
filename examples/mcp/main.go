// MCP 演示与 Model Context Protocol 服务器的集成，展示如何连接外部工具
// 服务器并将 MCP 工具与内置工具配合使用。
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/builtin"
	"github.com/astercloud/aster/pkg/tools/mcp"
	"github.com/astercloud/aster/pkg/types"
)

func main() {
	fmt.Println("=== Agent SDK - MCP 集成示例 ===")

	// 1. 创建工具注册表
	toolRegistry := tools.NewRegistry()

	// 2. 注册内置工具
	builtin.RegisterAll(toolRegistry)

	// 3. 创建 MCP Manager 并添加 MCP Server
	mcpManager := mcp.NewMCPManager(toolRegistry)

	// 示例: 添加一个 MCP Server (需要替换为实际的 MCP Server 地址)
	mcpEndpoint := os.Getenv("MCP_ENDPOINT")
	if mcpEndpoint == "" {
		mcpEndpoint = "http://localhost:8080/mcp" // 默认地址
	}

	mcpAccessKey := os.Getenv("MCP_ACCESS_KEY")
	mcpSecretKey := os.Getenv("MCP_SECRET_KEY")

	fmt.Printf("配置 MCP Server:\n")
	fmt.Printf("  Endpoint: %s\n", mcpEndpoint)
	fmt.Printf("  Access Key: %s\n\n", maskString(mcpAccessKey))

	// 添加 MCP Server
	server, err := mcpManager.AddServer(&mcp.MCPServerConfig{
		ServerID:        "my-mcp-server",
		Endpoint:        mcpEndpoint,
		AccessKeyID:     mcpAccessKey,
		AccessKeySecret: mcpSecretKey,
	})

	if err != nil {
		log.Printf("⚠️  添加 MCP Server 失败: %v", err)
		log.Println("   提示: 如果没有实际的 MCP Server，这是正常的")
		log.Println("   继续使用内置工具运行示例...")
	} else {
		fmt.Printf("✓ MCP Server 已添加: %s\n", server.GetServerID())

		// 4. 连接到 MCP Server 并注册工具
		ctx := context.Background()
		if err := mcpManager.ConnectServer(ctx, "my-mcp-server"); err != nil {
			log.Printf("⚠️  连接 MCP Server 失败: %v\n", err)
			log.Println("   提示: 确保 MCP Server 正在运行并且可访问")
			log.Println("   继续使用内置工具运行示例...")
		} else {
			fmt.Printf("✓ 已连接到 MCP Server\n")
			fmt.Printf("  发现工具数量: %d\n\n", server.GetToolCount())

			// 列出所有 MCP 工具
			tools := server.ListTools()
			if len(tools) > 0 {
				fmt.Println("可用的 MCP 工具:")
				for i, tool := range tools {
					fmt.Printf("  %d. %s - %s\n", i+1, tool.Name, tool.Description)
				}
				fmt.Println()
			}
		}
	}

	// 5. 创建依赖
	deps := createDependencies(toolRegistry)

	// 6. 创建 Agent
	fmt.Println("创建 Agent...")
	ag, err := agent.Create(context.Background(), &types.AgentConfig{
		AgentID:    "mcp-demo-agent",
		TemplateID: "assistant",
		ModelConfig: &types.ModelConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-5",
			APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
		},
		Sandbox: &types.SandboxConfig{
			Kind:    types.SandboxKindLocal,
			WorkDir: "./workspace",
		},
	}, deps)

	if err != nil {
		log.Fatalf("创建 Agent 失败: %v", err)
	}
	defer func() { _ = ag.Close() }()

	fmt.Println("✓ Agent 创建成功")

	// 7. 订阅事件
	eventCh := ag.Subscribe([]types.AgentChannel{types.ChannelProgress}, nil)
	go func() {
		for envelope := range eventCh {
			if evt, ok := envelope.Event.(types.EventType); ok {
				switch e := evt.(type) {
				case *types.ProgressTextChunkEvent:
					fmt.Print(e.Delta)
				case *types.ProgressToolStartEvent:
					fmt.Printf("\n[Tool] %s\n", e.Call.Name)
				case *types.ProgressToolErrorEvent:
					fmt.Printf("[Tool Error] %s\n", e.Error)
				}
			}
		}
	}()

	// 8. 与 Agent 对话
	fmt.Println("开始对话 (Agent 可以使用内置工具和 MCP 工具):")
	fmt.Println("---")

	message := "请列出当前目录的文件，然后创建一个 hello.txt 文件，内容是 'Hello from Agent SDK with MCP!'"

	result, err := ag.Chat(context.Background(), message)
	if err != nil {
		log.Fatalf("对话失败: %v", err)
	}

	fmt.Println("\n---")
	fmt.Printf("\n✓ 对话完成\n")
	fmt.Printf("  状态: %s\n", result.Status)

	// 9. 显示 MCP 统计信息
	fmt.Println("\nMCP 统计信息:")
	fmt.Printf("  Server 数量: %d\n", mcpManager.GetServerCount())
	fmt.Printf("  MCP 工具数量: %d\n", mcpManager.GetTotalToolCount())
	fmt.Printf("  内置工具数量: %d\n", len(toolRegistry.List())-mcpManager.GetTotalToolCount())
}

// createDependencies 创建 Agent 依赖
func createDependencies(toolRegistry *tools.Registry) *agent.Dependencies {
	// Store
	jsonStore, err := store.NewJSONStore("./.aster-mcp")
	if err != nil {
		log.Fatalf("创建存储失败: %v", err)
	}

	// Template Registry
	templateRegistry := agent.NewTemplateRegistry()
	templateRegistry.Register(&types.AgentTemplateDefinition{
		ID:           "assistant",
		SystemPrompt: "You are a helpful assistant with access to file system tools and MCP tools.",
		Model:        "claude-sonnet-4-5",
		Tools:        []any{}, // 将使用 Registry 中的所有工具
	})

	return &agent.Dependencies{
		Store:            jsonStore,
		SandboxFactory:   sandbox.NewFactory(),
		ToolRegistry:     toolRegistry,
		ProviderFactory:  &provider.AnthropicFactory{},
		TemplateRegistry: templateRegistry,
	}
}

// maskString 隐藏字符串中间部分
func maskString(s string) string {
	if s == "" {
		return "(未设置)"
	}
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}
