//go:build ignore

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/types"
)

func main() {
	apiKey := os.Getenv("CLAUDE_API_KEY")
	baseURL := os.Getenv("CLAUDE_BASE_URL")

	if apiKey == "" {
		fmt.Println("❌ 请设置 CLAUDE_API_KEY 环境变量")
		os.Exit(1)
	}
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	fmt.Println("=== Claude API 工具调用测试 ===")
	fmt.Printf("API 端点: %s\n", baseURL)
	fmt.Printf("模型: claude-sonnet-4-5-20250929\n\n")

	// 创建 Provider
	config := &types.ModelConfig{
		Provider: "anthropic",
		Model:    "claude-sonnet-4-5-20250929",
		APIKey:   apiKey,
		BaseURL:  baseURL,
	}

	cp, err := provider.NewCustomClaudeProvider(config)
	if err != nil {
		fmt.Printf("❌ 创建 Provider 失败: %v\n", err)
		os.Exit(1)
	}

	// 测试 1: 计算器工具
	fmt.Println("【测试 1】计算器工具")
	fmt.Println("----------------------------------------")
	testCalculatorTool(cp)

	// 测试 2: 获取时间工具
	fmt.Println("\n【测试 2】获取时间工具")
	fmt.Println("----------------------------------------")
	testGetTimeTool(cp)

	// 测试 3: 多工具组合使用
	fmt.Println("\n【测试 3】多工具组合使用")
	fmt.Println("----------------------------------------")
	testMultipleTools(cp)

	fmt.Println("\n✅ 工具调用测试完成！")
}

// testCalculatorTool 测试计算器工具
func testCalculatorTool(cp *provider.CustomClaudeProvider) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 定义计算器工具
	calculatorTool := provider.ToolSchema{
		Name:        "calculator",
		Description: "执行基本的数学计算（加减乘除）",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"expression": map[string]any{
					"type":        "string",
					"description": "要计算的数学表达式，例如 '123 + 456' 或 '100 * 2'",
				},
			},
			"required": []string{"expression"},
		},
	}

	// 第一次调用：让 Claude 决定使用工具
	userMessage := types.Message{
		Role: types.MessageRoleUser,
		ContentBlocks: []types.ContentBlock{
			&types.TextBlock{
				Text: "请帮我计算 1234 乘以 5678 等于多少？",
			},
		},
	}

	fmt.Println("💬 用户: 请帮我计算 1234 乘以 5678 等于多少？")

	opts := &provider.StreamOptions{
		MaxTokens: 1000,
		Tools:     []provider.ToolSchema{calculatorTool},
	}

	response, err := cp.Complete(ctx, []types.Message{userMessage}, opts)
	if err != nil {
		fmt.Printf("❌ 调用失败: %v\n", err)
		return
	}

	// 检查是否有工具调用
	var toolUse *types.ToolUseBlock
	for _, block := range response.Message.ContentBlocks {
		if tu, ok := block.(*types.ToolUseBlock); ok {
			toolUse = tu
			break
		}
	}

	if toolUse == nil {
		fmt.Println("⚠️  Claude 没有调用工具")
		// 输出文本响应
		for _, block := range response.Message.ContentBlocks {
			if textBlock, ok := block.(*types.TextBlock); ok {
				fmt.Printf("🤖 回复: %s\n", textBlock.Text)
			}
		}
		return
	}

	fmt.Printf("🔧 工具调用: %s\n", toolUse.Name)
	inputJSON, _ := json.MarshalIndent(toolUse.Input, "", "  ")
	fmt.Printf("📥 输入参数:\n%s\n", string(inputJSON))

	// 执行工具（模拟）
	expression := toolUse.Input["expression"].(string)
	result := executeCalculator(expression)
	fmt.Printf("⚙️  执行结果: %s = %s\n", expression, result)

	// 第二次调用：返回工具结果
	messages := []types.Message{
		userMessage,
		response.Message,
		{
			Role: types.MessageRoleUser,
			ContentBlocks: []types.ContentBlock{
				&types.ToolResultBlock{
					ToolUseID: toolUse.ID,
					Content:   result,
					IsError:   false,
				},
			},
		},
	}

	finalResponse, err := cp.Complete(ctx, messages, opts)
	if err != nil {
		fmt.Printf("❌ 获取最终回复失败: %v\n", err)
		return
	}

	// 输出最终回复
	for _, block := range finalResponse.Message.ContentBlocks {
		if textBlock, ok := block.(*types.TextBlock); ok {
			fmt.Printf("🤖 最终回复: %s\n", textBlock.Text)
		}
	}

	if finalResponse.Usage != nil {
		fmt.Printf("📊 Token: 输入=%d, 输出=%d\n",
			finalResponse.Usage.InputTokens, finalResponse.Usage.OutputTokens)
	}
}

// testGetTimeTool 测试获取时间工具
func testGetTimeTool(cp *provider.CustomClaudeProvider) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 定义获取时间工具
	getTimeTool := provider.ToolSchema{
		Name:        "get_current_time",
		Description: "获取当前的日期和时间",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"timezone": map[string]any{
					"type":        "string",
					"description": "时区，例如 'Asia/Shanghai', 'UTC'",
				},
			},
			"required": []string{},
		},
	}

	userMessage := types.Message{
		Role: types.MessageRoleUser,
		ContentBlocks: []types.ContentBlock{
			&types.TextBlock{
				Text: "现在几点了？",
			},
		},
	}

	fmt.Println("💬 用户: 现在几点了？")

	opts := &provider.StreamOptions{
		MaxTokens: 1000,
		Tools:     []provider.ToolSchema{getTimeTool},
	}

	response, err := cp.Complete(ctx, []types.Message{userMessage}, opts)
	if err != nil {
		fmt.Printf("❌ 调用失败: %v\n", err)
		return
	}

	// 检查工具调用
	var toolUse *types.ToolUseBlock
	for _, block := range response.Message.ContentBlocks {
		if tu, ok := block.(*types.ToolUseBlock); ok {
			toolUse = tu
			break
		}
	}

	if toolUse == nil {
		fmt.Println("⚠️  Claude 没有调用工具")
		for _, block := range response.Message.ContentBlocks {
			if textBlock, ok := block.(*types.TextBlock); ok {
				fmt.Printf("🤖 回复: %s\n", textBlock.Text)
			}
		}
		return
	}

	fmt.Printf("🔧 工具调用: %s\n", toolUse.Name)

	// 执行工具
	currentTime := time.Now().Format("2006-01-02 15:04:05 Monday")
	fmt.Printf("⚙️  执行结果: %s\n", currentTime)

	// 返回工具结果
	messages := []types.Message{
		userMessage,
		response.Message,
		{
			Role: types.MessageRoleUser,
			ContentBlocks: []types.ContentBlock{
				&types.ToolResultBlock{
					ToolUseID: toolUse.ID,
					Content:   currentTime,
					IsError:   false,
				},
			},
		},
	}

	finalResponse, err := cp.Complete(ctx, messages, opts)
	if err != nil {
		fmt.Printf("❌ 获取最终回复失败: %v\n", err)
		return
	}

	for _, block := range finalResponse.Message.ContentBlocks {
		if textBlock, ok := block.(*types.TextBlock); ok {
			fmt.Printf("🤖 最终回复: %s\n", textBlock.Text)
		}
	}

	if finalResponse.Usage != nil {
		fmt.Printf("📊 Token: 输入=%d, 输出=%d\n",
			finalResponse.Usage.InputTokens, finalResponse.Usage.OutputTokens)
	}
}

// testMultipleTools 测试多工具组合
func testMultipleTools(cp *provider.CustomClaudeProvider) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 定义多个工具
	tools := []provider.ToolSchema{
		{
			Name:        "calculator",
			Description: "执行数学计算",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"expression": map[string]any{
						"type":        "string",
						"description": "数学表达式",
					},
				},
				"required": []string{"expression"},
			},
		},
		{
			Name:        "get_current_time",
			Description: "获取当前时间",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "get_weather",
			Description: "获取指定城市的天气信息",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"city": map[string]any{
						"type":        "string",
						"description": "城市名称，例如 '北京', '上海'",
					},
				},
				"required": []string{"city"},
			},
		},
	}

	userMessage := types.Message{
		Role: types.MessageRoleUser,
		ContentBlocks: []types.ContentBlock{
			&types.TextBlock{
				Text: "请帮我查一下北京的天气，顺便告诉我现在几点了",
			},
		},
	}

	fmt.Println("💬 用户: 请帮我查一下北京的天气，顺便告诉我现在几点了")

	opts := &provider.StreamOptions{
		MaxTokens: 1500,
		Tools:     tools,
	}

	response, err := cp.Complete(ctx, []types.Message{userMessage}, opts)
	if err != nil {
		fmt.Printf("❌ 调用失败: %v\n", err)
		return
	}

	// 收集所有工具调用
	var toolCalls []*types.ToolUseBlock
	for _, block := range response.Message.ContentBlocks {
		if tu, ok := block.(*types.ToolUseBlock); ok {
			toolCalls = append(toolCalls, tu)
		}
	}

	if len(toolCalls) == 0 {
		fmt.Println("⚠️  Claude 没有调用工具")
		for _, block := range response.Message.ContentBlocks {
			if textBlock, ok := block.(*types.TextBlock); ok {
				fmt.Printf("🤖 回复: %s\n", textBlock.Text)
			}
		}
		return
	}

	fmt.Printf("🔧 检测到 %d 个工具调用\n", len(toolCalls))

	// 执行所有工具
	var toolResults []types.ContentBlock
	for i, toolCall := range toolCalls {
		fmt.Printf("\n[工具 %d/%d]\n", i+1, len(toolCalls))
		fmt.Printf("  名称: %s\n", toolCall.Name)
		inputJSON, _ := json.MarshalIndent(toolCall.Input, "  ", "  ")
		fmt.Printf("  输入: %s\n", string(inputJSON))

		var result string
		switch toolCall.Name {
		case "get_weather":
			city := toolCall.Input["city"].(string)
			result = fmt.Sprintf(`{"city":"%s","temperature":"15℃","condition":"晴天","humidity":"45%%"}`, city)
		case "get_current_time":
			result = time.Now().Format("2006-01-02 15:04:05 Monday")
		case "calculator":
			expression := toolCall.Input["expression"].(string)
			result = executeCalculator(expression)
		default:
			result = "未知工具"
		}

		fmt.Printf("  结果: %s\n", result)

		toolResults = append(toolResults, &types.ToolResultBlock{
			ToolUseID: toolCall.ID,
			Content:   result,
			IsError:   false,
		})
	}

	// 返回所有工具结果
	messages := []types.Message{
		userMessage,
		response.Message,
		{
			Role:          types.MessageRoleUser,
			ContentBlocks: toolResults,
		},
	}

	finalResponse, err := cp.Complete(ctx, messages, opts)
	if err != nil {
		fmt.Printf("❌ 获取最终回复失败: %v\n", err)
		return
	}

	fmt.Println()
	for _, block := range finalResponse.Message.ContentBlocks {
		if textBlock, ok := block.(*types.TextBlock); ok {
			fmt.Printf("🤖 最终回复: %s\n", textBlock.Text)
		}
	}

	if finalResponse.Usage != nil {
		fmt.Printf("📊 Token: 输入=%d, 输出=%d\n",
			finalResponse.Usage.InputTokens, finalResponse.Usage.OutputTokens)
	}
}

// executeCalculator 执行计算（简化版本）
func executeCalculator(expression string) string {
	// 这里简化处理，实际应该用表达式解析器
	// 示例: "1234 * 5678" -> "7006652"
	switch expression {
	case "1234 * 5678", "1234 乘以 5678":
		return "7006652"
	case "100 + 200":
		return "300"
	case "500 - 200":
		return "300"
	default:
		return fmt.Sprintf("计算结果: %s (模拟)", expression)
	}
}
