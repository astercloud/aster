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

// 文件批处理示例
// 演示如何使用 PTC 进行复杂的文件处理任务
func main() {
	// 检查 API Key
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("请设置环境变量 ANTHROPIC_API_KEY")
	}

	fmt.Println("=== Aster PTC 文件处理示例 ===")

	// 1. 创建工具生态系统
	registry := tools.NewRegistry()
	builtin.RegisterAll(registry)

	toolBridge := bridge.NewToolBridge(registry)
	codeExecTool := builtin.NewCodeExecuteToolWithBridge(toolBridge)

	// 2. 准备工具列表(启用 PTC)
	toolSchemas := []provider.ToolSchema{
		{
			Name:           "CodeExecute",
			Description:    codeExecTool.Description(),
			InputSchema:    codeExecTool.InputSchema(),
			AllowedCallers: []string{"direct"},
		},
	}

	// 添加可在 Python 中调用的工具
	allowedTools := []string{"Read", "Write", "Glob", "Grep", "Bash"}
	for _, toolName := range allowedTools {
		tool, _ := registry.Create(toolName, nil)
		toolSchemas = append(toolSchemas, provider.ToolSchema{
			Name:           toolName,
			Description:    tool.Description(),
			InputSchema:    tool.InputSchema(),
			AllowedCallers: []string{"direct", "code_execution_20250825"},
		})
	}

	// 3. 创建 Anthropic Provider
	providerConfig := &types.ModelConfig{
		Provider: "anthropic",
		Model:    "claude-3-5-sonnet-20241022",
		APIKey:   apiKey,
	}

	anthropicProvider, err := provider.NewAnthropicProvider(providerConfig)
	if err != nil {
		log.Fatalf("创建 Provider 失败: %v", err)
	}

	// 4. 定义复杂任务
	task := `请编写 Python 代码完成以下文件处理任务:

1. 使用 Glob 查找当前目录下所有 .go 文件
2. 使用 Grep 在这些文件中搜索包含 "TODO" 或 "FIXME" 的注释
3. 统计每个文件中待办事项的数量
4. 生成一份 Markdown 格式的报告,包含:
   - 总待办事项数量
   - 每个文件的待办事项列表
   - 按优先级分类(FIXME > TODO)
5. 使用 Write 将报告保存到 TODO_REPORT.md

要求:
- 使用异步编程提升性能
- 对每个待办事项包含行号和内容
- 报告格式清晰美观`

	fmt.Printf("任务: %s\n\n", task)

	messages := []types.Message{
		{
			Role:    types.RoleUser,
			Content: task,
		},
	}

	// 5. 调用 LLM
	opts := &provider.StreamOptions{
		Tools:       toolSchemas,
		MaxTokens:   4096,
		Temperature: 0.7,
		System:      "你是一个专业的 Python 开发者,擅长使用工具完成文件处理任务。代码要简洁高效,充分利用 asyncio 并发能力。",
	}

	fmt.Println("正在调用 LLM 生成代码...")
	ctx := context.Background()

	response, err := anthropicProvider.Complete(ctx, messages, opts)
	if err != nil {
		log.Fatalf("LLM 调用失败: %v", err)
	}

	// 6. 处理响应并执行工具
	fmt.Println("=== LLM 响应 ===")

	for _, block := range response.Message.ContentBlocks {
		switch b := block.(type) {
		case *types.TextBlock:
			fmt.Printf("\n[文本]\n%s\n", b.Text)

		case *types.ToolUseBlock:
			fmt.Printf("\n[工具调用: %s]\n", b.Name)

			// 打印 Caller 信息
			if b.Caller != nil {
				fmt.Printf("调用方式: %s\n", b.Caller.Type)
			} else {
				fmt.Printf("调用方式: direct (LLM 直接调用)\n")
			}

			// 执行工具
			var tool tools.Tool
			if b.Name == "CodeExecute" {
				tool = codeExecTool
			} else {
				tool, _ = registry.Create(b.Name, nil)
			}

			fmt.Println("执行中...")
			result, err := tool.Execute(ctx, b.Input, &tools.ToolContext{})
			if err != nil {
				fmt.Printf("❌ 执行失败: %v\n", err)
				continue
			}

			// 打印结果
			if resultMap, ok := result.(map[string]any); ok {
				if success, ok := resultMap["success"].(bool); ok && success {
					fmt.Println("✅ 执行成功")

					// 如果是 CodeExecute,打印输出
					if b.Name == "CodeExecute" {
						if stdout, ok := resultMap["stdout"].(string); ok && stdout != "" {
							fmt.Printf("\n输出:\n%s\n", stdout)
						}
						if stderr, ok := resultMap["stderr"].(string); ok && stderr != "" {
							fmt.Printf("\n错误输出:\n%s\n", stderr)
						}
					}
				} else {
					fmt.Printf("❌ 执行失败: %v\n", resultMap["error"])
				}
			}
		}
	}

	// 7. 显示统计信息
	if response.Usage != nil {
		fmt.Printf("\n\n=== 统计信息 ===\n")
		fmt.Printf("输入 Tokens: %d\n", response.Usage.InputTokens)
		fmt.Printf("输出 Tokens: %d\n", response.Usage.OutputTokens)
		fmt.Printf("总计 Tokens: %d\n", response.Usage.InputTokens+response.Usage.OutputTokens)
	}

	fmt.Println("\n\n=== 完成 ===")
	fmt.Println("如果执行成功,请查看 TODO_REPORT.md 文件")
}
