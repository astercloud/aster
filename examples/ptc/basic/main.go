package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/bridge"
	"github.com/astercloud/aster/pkg/tools/builtin"
	"github.com/astercloud/aster/pkg/types"
)

// PTC 基础示例
// 演示如何使用 Programmatic Tool Calling 让 LLM 生成 Python 代码并调用工具
func main() {
	// 检查 API Key
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("请设置环境变量 ANTHROPIC_API_KEY")
	}

	// 1. 创建工具注册表并注册内置工具
	registry := tools.NewRegistry()
	builtin.RegisterAll(registry)

	// 2. 创建 ToolBridge 用于程序化工具调用
	toolBridge := bridge.NewToolBridge(registry)

	// 3. 创建 CodeExecute 工具并启用 PTC 支持
	codeExecTool := builtin.NewCodeExecuteToolWithBridge(toolBridge)

	// 可选: 自定义桥接服务器地址
	// codeExecTool.SetBridgeURL("http://localhost:9000")

	// 4. 创建工具实例
	readTool, _ := registry.Create("Read", nil)
	writeTool, _ := registry.Create("Write", nil)
	globTool, _ := registry.Create("Glob", nil)
	bashTool, _ := registry.Create("Bash", nil)

	// 5. 转换为 Provider ToolSchema (添加 AllowedCallers 支持)
	toolSchemas := []provider.ToolSchema{
		{
			Name:        "CodeExecute",
			Description: codeExecTool.Description(),
			InputSchema: codeExecTool.InputSchema(),
			// CodeExecute 只能被 LLM 直接调用
			AllowedCallers: []string{"direct"},
		},
		{
			Name:        "Read",
			Description: readTool.Description(),
			InputSchema: readTool.InputSchema(),
			// Read 可以被 LLM 直接调用,也可以在 Python 代码中调用
			AllowedCallers: []string{"direct", "code_execution_20250825"},
		},
		{
			Name:           "Write",
			Description:    writeTool.Description(),
			InputSchema:    writeTool.InputSchema(),
			AllowedCallers: []string{"direct", "code_execution_20250825"},
		},
		{
			Name:           "Glob",
			Description:    globTool.Description(),
			InputSchema:    globTool.InputSchema(),
			AllowedCallers: []string{"direct", "code_execution_20250825"},
		},
		{
			Name:           "Bash",
			Description:    bashTool.Description(),
			InputSchema:    bashTool.InputSchema(),
			AllowedCallers: []string{"direct", "code_execution_20250825"},
		},
	}

	// 6. 创建 Anthropic Provider
	providerConfig := &types.ModelConfig{
		Provider: "anthropic",
		Model:    "claude-3-5-sonnet-20241022",
		APIKey:   apiKey,
	}

	anthropicProvider, err := provider.NewAnthropicProvider(providerConfig)
	if err != nil {
		log.Fatalf("创建 Provider 失败: %v", err)
	}

	// 7. 准备消息
	messages := []types.Message{
		{
			Role:    types.RoleUser,
			Content: "请用 Python 代码完成以下任务:\n1. 使用 Glob 查找当前目录下所有 .go 文件\n2. 使用 Read 读取每个文件\n3. 统计总行数和总字符数\n4. 输出统计结果",
		},
	}

	// 8. 调用 LLM (非流式)
	ctx := context.Background()
	opts := &provider.StreamOptions{
		Tools:       toolSchemas,
		MaxTokens:   4096,
		Temperature: 0.7,
		System:      "你是一个编程助手,擅长使用 Python 代码调用工具完成任务。",
	}

	fmt.Println("正在调用 LLM 生成 Python 代码...")
	response, err := anthropicProvider.Complete(ctx, messages, opts)
	if err != nil {
		log.Fatalf("LLM 调用失败: %v", err)
	}

	// 9. 处理响应
	fmt.Println("\n=== LLM 响应 ===")
	for _, block := range response.Message.ContentBlocks {
		switch b := block.(type) {
		case *types.TextBlock:
			fmt.Printf("文本: %s\n", b.Text)

		case *types.ToolUseBlock:
			fmt.Printf("\n工具调用: %s\n", b.Name)
			fmt.Printf("参数: %+v\n", b.Input)

			// PTC 支持: 检查 Caller 信息
			if b.Caller != nil {
				fmt.Printf("调用者类型: %s\n", b.Caller.Type)
				if b.Caller.ToolID != "" {
					fmt.Printf("调用者工具ID: %s\n", b.Caller.ToolID)
				}
			}

			// 执行工具
			var tool tools.Tool
			var err error

			switch b.Name {
			case "CodeExecute":
				tool = codeExecTool
			default:
				tool, err = registry.Create(b.Name, nil)
			}

			if err != nil {
				fmt.Printf("工具创建失败: %v\n", err)
				continue
			}

			// 执行工具
			result, err := tool.Execute(ctx, b.Input, &tools.ToolContext{})
			if err != nil {
				fmt.Printf("工具执行失败: %v\n", err)
				continue
			}

			fmt.Printf("执行结果: %+v\n", result)
		}
	}

	// 输出 Token 使用情况
	if response.Usage != nil {
		fmt.Printf("\n=== Token 使用 ===\n")
		fmt.Printf("输入: %d tokens\n", response.Usage.InputTokens)
		fmt.Printf("输出: %d tokens\n", response.Usage.OutputTokens)
		fmt.Printf("总计: %d tokens\n", response.Usage.InputTokens+response.Usage.OutputTokens)
	}
}
