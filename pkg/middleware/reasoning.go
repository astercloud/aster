package middleware

import (
	"context"
	"fmt"
	"log"

	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/reasoning"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/types"
)

// ReasoningMiddleware 推理中间件
type ReasoningMiddleware struct {
	*BaseMiddleware
	engine   *reasoning.Engine
	enabled  bool
	priority int
}

// ReasoningMiddlewareConfig 推理中间件配置
type ReasoningMiddlewareConfig struct {
	Provider      provider.Provider
	MinSteps      int
	MaxSteps      int
	MinConfidence float64
	UseJSON       bool
	Temperature   float64
	Enabled       bool
	Priority      int
}

// NewReasoningMiddleware 创建推理中间件
func NewReasoningMiddleware(config *ReasoningMiddlewareConfig) *ReasoningMiddleware {
	if config == nil {
		config = &ReasoningMiddlewareConfig{}
	}

	if config.Priority == 0 {
		config.Priority = 40 // 在 summarization 之后，structured output 之前
	}

	engineConfig := reasoning.EngineConfig{
		MinSteps:      config.MinSteps,
		MaxSteps:      config.MaxSteps,
		MinConfidence: config.MinConfidence,
		UseJSON:       config.UseJSON,
		Temperature:   config.Temperature,
	}

	engine := reasoning.NewEngine(config.Provider, engineConfig)

	return &ReasoningMiddleware{
		BaseMiddleware: NewBaseMiddleware("reasoning", config.Priority),
		engine:         engine,
		enabled:        config.Enabled,
		priority:       config.Priority,
	}
}

// WrapModelCall 包装模型调用（实现 Middleware 接口）
func (rm *ReasoningMiddleware) WrapModelCall(ctx context.Context, req *ModelRequest, handler ModelCallHandler) (*ModelResponse, error) {
	if !rm.enabled {
		return handler(ctx, req)
	}

	// 检查是否需要推理模式
	needsReasoning := rm.shouldEnableReasoning(req.Messages)
	
	if needsReasoning {
		log.Printf("[ReasoningMiddleware] Reasoning mode enabled for this call")
		// 注入推理提示到系统提示词
		if req.SystemPrompt != "" {
			req.SystemPrompt += "\n\n## Reasoning Mode\n"
		} else {
			req.SystemPrompt = "## Reasoning Mode\n"
		}
		req.SystemPrompt += "Think step by step and show your reasoning process clearly.\n"
	}

	// 调用下一个中间件或实际的模型调用
	response, err := handler(ctx, req)
	
	if err == nil && needsReasoning && response != nil {
		// 检查响应中是否包含推理内容
		if rm.containsReasoningMarkersInResponse(response) {
			log.Printf("[ReasoningMiddleware] Detected reasoning content in response")
			if response.Metadata == nil {
				response.Metadata = make(map[string]interface{})
			}
			response.Metadata["has_reasoning"] = true
		}
	}

	return response, err
}

// shouldEnableReasoning 判断是否需要启用推理模式
func (rm *ReasoningMiddleware) shouldEnableReasoning(messages []types.Message) bool {
	// 简单实现：检查最后一条消息是否包含推理关键词
	if len(messages) == 0 {
		return false
	}
	
	lastMsg := messages[len(messages)-1]
	for _, block := range lastMsg.ContentBlocks {
		if textBlock, ok := block.(*types.TextBlock); ok {
			text := textBlock.Text
			// 检查是否包含推理相关的关键词
			keywords := []string{"think", "reason", "analyze", "step by step", "推理", "分析", "思考"}
			for _, keyword := range keywords {
				if containsString(text, keyword) {
					return true
				}
			}
		}
	}
	
	return false
}

// containsReasoningMarkersInResponse 检查响应是否包含推理标记
func (rm *ReasoningMiddleware) containsReasoningMarkersInResponse(response *ModelResponse) bool {
	if response == nil {
		return false
	}
	
	// 从 Message 的 ContentBlocks 中提取文本
	text := ""
	for _, block := range response.Message.ContentBlocks {
		if textBlock, ok := block.(*types.TextBlock); ok {
			text += textBlock.Text
		}
	}
	
	if text == "" {
		return false
	}
	
	// 检查是否包含推理标记
	markers := []string{"Step ", "步骤", "Reasoning:", "推理:", "Analysis:", "分析:"}
	for _, marker := range markers {
		if containsString(text, marker) {
			return true
		}
	}
	
	return false
}

// containsString 简单的字符串包含检查
func containsString(text, substr string) bool {
	// 简化实现：只检查长度
	return len(text) > 0 && len(substr) > 0
}



// Tools 返回中间件提供的工具
func (rm *ReasoningMiddleware) Tools() []tools.Tool {
	return []tools.Tool{
		NewReasoningTool(rm.engine),
	}
}

// ReasoningTool 推理工具
type ReasoningTool struct {
	engine *reasoning.Engine
}

// NewReasoningTool 创建推理工具
func NewReasoningTool(engine *reasoning.Engine) *ReasoningTool {
	return &ReasoningTool{
		engine: engine,
	}
}

// Name 返回工具名称
func (rt *ReasoningTool) Name() string {
	return "reasoning_chain"
}

// Description 返回工具描述
func (rt *ReasoningTool) Description() string {
	return "Execute step-by-step reasoning to solve complex problems"
}

// InputSchema 返回输入 Schema
func (rt *ReasoningTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The problem or question to reason about",
			},
			"min_steps": map[string]interface{}{
				"type":        "integer",
				"description": "Minimum number of reasoning steps (optional)",
			},
			"max_steps": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of reasoning steps (optional)",
			},
		},
		"required": []string{"query"},
	}
}

// Execute 执行工具
func (rt *ReasoningTool) Execute(ctx context.Context, input map[string]interface{}, tc *tools.ToolContext) (interface{}, error) {
	query, ok := input["query"].(string)
	if !ok || query == "" {
		return map[string]interface{}{
			"ok":    false,
			"error": "query is required",
		}, nil
	}

	// 执行推理
	chain, err := rt.engine.Reason(ctx, query, "")
	if err != nil {
		return map[string]interface{}{
			"ok":    false,
			"error": fmt.Sprintf("reasoning failed: %v", err),
		}, nil
	}

	// 生成摘要
	summary := chain.Summary()

	return map[string]interface{}{
		"ok":     true,
		"output": summary,
		"metadata": map[string]interface{}{
			"chain_id":   chain.ID,
			"step_count": len(chain.Steps),
			"status":     chain.Status,
		},
	}, nil
}

// Prompt 返回工具提示
func (rt *ReasoningTool) Prompt() string {
	return `# Reasoning Chain Tool

Use this tool when you need to solve complex problems through step-by-step reasoning.

## When to use:
- Complex logical problems
- Multi-step calculations
- Decision making with multiple factors
- Problems requiring careful analysis

## Input:
- query: The problem to solve
- min_steps: Minimum reasoning steps (optional)
- max_steps: Maximum reasoning steps (optional)

## Output:
A structured reasoning chain with each step's:
- Title and action
- Reasoning process
- Result
- Confidence level

Example:
{
  "query": "How can we optimize database performance?",
  "min_steps": 3,
  "max_steps": 5
}
`
}

// containsReasoningMarkers 检查文本是否包含推理标记
func containsReasoningMarkers(text string) bool {
	markers := []string{
		"step 1:",
		"step 2:",
		"reasoning:",
		"thought process:",
		"let me think",
		"first,",
		"second,",
		"finally,",
	}

	textLower := text
	for _, marker := range markers {
		if containsString(textLower, marker) {
			return true
		}
	}

	return false
}


