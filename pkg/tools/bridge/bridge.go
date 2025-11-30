package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/astercloud/aster/pkg/tools"
)

// ToolBridge 工具桥接器
// 提供统一的工具调用接口，支持程序化调用和批量执行
type ToolBridge struct {
	mu       sync.RWMutex
	registry *tools.Registry
	tools    map[string]tools.Tool
}

// NewToolBridge 创建工具桥接器
func NewToolBridge(registry *tools.Registry) *ToolBridge {
	return &ToolBridge{
		registry: registry,
		tools:    make(map[string]tools.Tool),
	}
}

// GetTool 获取或创建工具实例
func (b *ToolBridge) GetTool(name string) (tools.Tool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 检查缓存
	if tool, exists := b.tools[name]; exists {
		return tool, nil
	}

	// 从 registry 创建
	tool, err := b.registry.Create(name, nil)
	if err != nil {
		return nil, fmt.Errorf("get tool %s: %w", name, err)
	}

	b.tools[name] = tool
	return tool, nil
}

// CallToolInput 工具调用输入
type CallToolInput struct {
	Name  string         `json:"name"`
	Input map[string]any `json:"input"`
}

// CallToolResult 工具调用结果
type CallToolResult struct {
	Name    string `json:"name"`
	Success bool   `json:"success"`
	Result  any    `json:"result,omitempty"`
	Error   string `json:"error,omitempty"`
}

// CallTool 调用单个工具
func (b *ToolBridge) CallTool(ctx context.Context, name string, input map[string]any, tc *tools.ToolContext) (*CallToolResult, error) {
	tool, err := b.GetTool(name)
	if err != nil {
		return &CallToolResult{
			Name:    name,
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	result, err := tool.Execute(ctx, input, tc)
	if err != nil {
		return &CallToolResult{
			Name:    name,
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &CallToolResult{
		Name:    name,
		Success: true,
		Result:  result,
	}, nil
}

// CallToolJSON 使用 JSON 字符串调用工具
func (b *ToolBridge) CallToolJSON(ctx context.Context, name string, inputJSON string, tc *tools.ToolContext) (*CallToolResult, error) {
	var input map[string]any
	if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
		return &CallToolResult{
			Name:    name,
			Success: false,
			Error:   fmt.Sprintf("invalid JSON input: %v", err),
		}, nil
	}

	return b.CallTool(ctx, name, input, tc)
}

// BatchCallResult 批量调用结果
type BatchCallResult struct {
	Results   []*CallToolResult `json:"results"`
	Succeeded int               `json:"succeeded"`
	Failed    int               `json:"failed"`
}

// CallToolsBatch 批量调用工具（顺序执行）
func (b *ToolBridge) CallToolsBatch(ctx context.Context, calls []CallToolInput, tc *tools.ToolContext) *BatchCallResult {
	results := make([]*CallToolResult, len(calls))
	succeeded := 0
	failed := 0

	for i, call := range calls {
		result, _ := b.CallTool(ctx, call.Name, call.Input, tc)
		results[i] = result
		if result.Success {
			succeeded++
		} else {
			failed++
		}
	}

	return &BatchCallResult{
		Results:   results,
		Succeeded: succeeded,
		Failed:    failed,
	}
}

// CallToolsParallel 并行调用工具
func (b *ToolBridge) CallToolsParallel(ctx context.Context, calls []CallToolInput, tc *tools.ToolContext) *BatchCallResult {
	results := make([]*CallToolResult, len(calls))
	var wg sync.WaitGroup
	var mu sync.Mutex
	succeeded := 0
	failed := 0

	for i, call := range calls {
		wg.Add(1)
		go func(idx int, c CallToolInput) {
			defer wg.Done()
			result, _ := b.CallTool(ctx, c.Name, c.Input, tc)

			mu.Lock()
			results[idx] = result
			if result.Success {
				succeeded++
			} else {
				failed++
			}
			mu.Unlock()
		}(i, call)
	}

	wg.Wait()

	return &BatchCallResult{
		Results:   results,
		Succeeded: succeeded,
		Failed:    failed,
	}
}

// ListAvailableTools 列出所有可用工具
func (b *ToolBridge) ListAvailableTools() []string {
	return b.registry.List()
}

// GetToolSchema 获取工具的 schema
func (b *ToolBridge) GetToolSchema(name string) (map[string]any, error) {
	tool, err := b.GetTool(name)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"name":         tool.Name(),
		"description":  tool.Description(),
		"input_schema": tool.InputSchema(),
	}, nil
}

// ToolChain 工具链 - 按顺序执行一系列工具，前一个的输出可作为后一个的输入
type ToolChain struct {
	bridge *ToolBridge
	steps  []ChainStep
}

// ChainStep 链中的一步
type ChainStep struct {
	Name  string         `json:"name"`
	Input map[string]any `json:"input"`
	// InputMapper 可选：将前一步结果映射到当前输入
	InputMapper func(prevResult any) map[string]any `json:"-"`
}

// NewToolChain 创建工具链
func NewToolChain(bridge *ToolBridge) *ToolChain {
	return &ToolChain{
		bridge: bridge,
		steps:  make([]ChainStep, 0),
	}
}

// AddStep 添加步骤
func (c *ToolChain) AddStep(step ChainStep) *ToolChain {
	c.steps = append(c.steps, step)
	return c
}

// ChainResult 链执行结果
type ChainResult struct {
	Steps       []*CallToolResult `json:"steps"`
	FinalResult any               `json:"final_result"`
	Success     bool              `json:"success"`
	Error       string            `json:"error,omitempty"`
}

// Execute 执行工具链
func (c *ToolChain) Execute(ctx context.Context, tc *tools.ToolContext) *ChainResult {
	if len(c.steps) == 0 {
		return &ChainResult{
			Success: false,
			Error:   "no steps in chain",
		}
	}

	results := make([]*CallToolResult, len(c.steps))
	var prevResult any

	for i, step := range c.steps {
		input := step.Input

		// 如果有 mapper 且有前一步结果，使用 mapper 生成输入
		if step.InputMapper != nil && prevResult != nil {
			input = step.InputMapper(prevResult)
		}

		result, _ := c.bridge.CallTool(ctx, step.Name, input, tc)
		results[i] = result

		if !result.Success {
			return &ChainResult{
				Steps:   results[:i+1],
				Success: false,
				Error:   fmt.Sprintf("step %d (%s) failed: %s", i, step.Name, result.Error),
			}
		}

		prevResult = result.Result
	}

	return &ChainResult{
		Steps:       results,
		FinalResult: prevResult,
		Success:     true,
	}
}
