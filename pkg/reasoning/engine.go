package reasoning

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/types"
)

// Engine 推理引擎
type Engine struct {
	provider provider.Provider
	config   EngineConfig
}

// EngineConfig 推理引擎配置
type EngineConfig struct {
	MinSteps      int     // 最小推理步数
	MaxSteps      int     // 最大推理步数
	MinConfidence float64 // 最小置信度
	UseJSON       bool    // 是否使用 JSON 格式
	Temperature   float64 // 温度参数
}

// NewEngine 创建推理引擎
func NewEngine(provider provider.Provider, config EngineConfig) *Engine {
	if config.MinSteps <= 0 {
		config.MinSteps = 1
	}
	if config.MaxSteps <= 0 {
		config.MaxSteps = 10
	}
	if config.MinConfidence <= 0 {
		config.MinConfidence = 0.7
	}
	if config.Temperature <= 0 {
		config.Temperature = 0.7
	}

	return &Engine{
		provider: provider,
		config:   config,
	}
}

// Reason 执行推理
func (e *Engine) Reason(ctx context.Context, query string, systemPrompt string) (*Chain, error) {
	chain := NewChain(ChainConfig{
		MinSteps:      e.config.MinSteps,
		MaxSteps:      e.config.MaxSteps,
		MinConfidence: e.config.MinConfidence,
	})

	// 构建推理提示词
	reasoningPrompt := e.buildReasoningPrompt(query, systemPrompt)

	// 执行推理循环
	for chain.ShouldContinue() {
		step, err := e.executeReasoningStep(ctx, chain, reasoningPrompt)
		if err != nil {
			return chain, fmt.Errorf("execute reasoning step: %w", err)
		}

		if err := chain.AddStep(*step); err != nil {
			return chain, fmt.Errorf("add reasoning step: %w", err)
		}

		// 如果步骤指示完成，则退出
		if step.NextAction == NextActionComplete {
			break
		}

		// 如果置信度过低且已达到最小步数，则退出
		if step.Confidence < e.config.MinConfidence && len(chain.Steps) >= e.config.MinSteps {
			break
		}
	}

	chain.Complete()
	return chain, nil
}

// executeReasoningStep 执行单个推理步骤
func (e *Engine) executeReasoningStep(ctx context.Context, chain *Chain, basePrompt string) (*Step, error) {
	// 构建当前步骤的提示词
	prompt := e.buildStepPrompt(chain, basePrompt)

	// 调用 Provider
	messages := []types.Message{
		{
			Role: types.MessageRoleUser,
			ContentBlocks: []types.ContentBlock{
				&types.TextBlock{Text: prompt},
			},
		},
	}

	response, err := e.provider.Complete(ctx, messages, nil)
	if err != nil {
		return nil, fmt.Errorf("provider complete: %w", err)
	}

	// 解析响应
	step, err := e.parseStepResponse(response)
	if err != nil {
		return nil, fmt.Errorf("parse step response: %w", err)
	}

	return step, nil
}

// buildReasoningPrompt 构建推理提示词
func (e *Engine) buildReasoningPrompt(query string, systemPrompt string) string {
	var prompt strings.Builder

	if systemPrompt != "" {
		prompt.WriteString(systemPrompt)
		prompt.WriteString("\n\n")
	}

	prompt.WriteString("# Reasoning Mode\n\n")
	prompt.WriteString("You are in reasoning mode. Think step by step to solve the problem.\n\n")
	prompt.WriteString("For each reasoning step, provide:\n")
	prompt.WriteString("1. **Title**: A brief title for this step\n")
	prompt.WriteString("2. **Action**: What you plan to do in this step\n")
	prompt.WriteString("3. **Reasoning**: Your thought process\n")
	prompt.WriteString("4. **Result**: The outcome of this step\n")
	prompt.WriteString("5. **Confidence**: Your confidence level (0.0-1.0)\n")
	prompt.WriteString("6. **Next Action**: continue/complete/retry\n\n")

	if e.config.UseJSON {
		prompt.WriteString("Please respond in JSON format:\n")
		prompt.WriteString("```json\n")
		prompt.WriteString("{\n")
		prompt.WriteString("  \"title\": \"Step title\",\n")
		prompt.WriteString("  \"action\": \"What to do\",\n")
		prompt.WriteString("  \"reasoning\": \"Thought process\",\n")
		prompt.WriteString("  \"result\": \"Outcome\",\n")
		prompt.WriteString("  \"confidence\": 0.85,\n")
		prompt.WriteString("  \"next_action\": \"continue\"\n")
		prompt.WriteString("}\n")
		prompt.WriteString("```\n\n")
	}

	prompt.WriteString("## Query\n\n")
	prompt.WriteString(query)

	return prompt.String()
}

// buildStepPrompt 构建步骤提示词
func (e *Engine) buildStepPrompt(chain *Chain, basePrompt string) string {
	var prompt strings.Builder

	prompt.WriteString(basePrompt)
	prompt.WriteString("\n\n")

	if len(chain.Steps) > 0 {
		prompt.WriteString("## Previous Steps\n\n")
		for i, step := range chain.Steps {
			prompt.WriteString(fmt.Sprintf("### Step %d: %s\n", i+1, step.Title))
			prompt.WriteString(fmt.Sprintf("- Action: %s\n", step.Action))
			prompt.WriteString(fmt.Sprintf("- Result: %s\n", step.Result))
			prompt.WriteString(fmt.Sprintf("- Confidence: %.2f\n\n", step.Confidence))
		}
	}

	prompt.WriteString(fmt.Sprintf("## Current Step: %d/%d\n\n", len(chain.Steps)+1, chain.MaxSteps))
	prompt.WriteString("Please provide the next reasoning step.\n")

	return prompt.String()
}

// parseStepResponse 解析步骤响应
func (e *Engine) parseStepResponse(response *provider.CompleteResponse) (*Step, error) {
	// 提取文本内容
	text := ""
	var textSb182 strings.Builder
	for _, block := range response.Message.ContentBlocks {
		if textBlock, ok := block.(*types.TextBlock); ok {
			textSb182.WriteString(textBlock.Text)
		}
	}
	text += textSb182.String()

	// 尝试 JSON 解析
	if e.config.UseJSON {
		step, err := e.parseJSONStep(text)
		if err == nil {
			return step, nil
		}
	}

	// 回退到文本解析
	return e.parseTextStep(text)
}

// parseJSONStep 解析 JSON 格式的步骤
func (e *Engine) parseJSONStep(text string) (*Step, error) {
	// 提取 JSON 代码块
	jsonPattern := regexp.MustCompile("```json\\s*\\n([\\s\\S]*?)\\n```")
	matches := jsonPattern.FindStringSubmatch(text)

	var jsonText string
	if len(matches) > 1 {
		jsonText = matches[1]
	} else {
		// 尝试直接解析整个文本
		jsonText = text
	}

	var step Step
	if err := json.Unmarshal([]byte(jsonText), &step); err != nil {
		return nil, fmt.Errorf("unmarshal json step: %w", err)
	}

	step.Status = StepStatusCompleted
	return &step, nil
}

// parseTextStep 解析文本格式的步骤
func (e *Engine) parseTextStep(text string) (*Step, error) {
	step := &Step{
		Status:     StepStatusCompleted,
		Confidence: 0.8, // 默认置信度
		NextAction: NextActionContinue,
	}

	// 提取标题
	titlePattern := regexp.MustCompile(`(?i)(?:title|step):\s*(.+)`)
	if matches := titlePattern.FindStringSubmatch(text); len(matches) > 1 {
		step.Title = strings.TrimSpace(matches[1])
	}

	// 提取行动
	actionPattern := regexp.MustCompile(`(?i)action:\s*(.+)`)
	if matches := actionPattern.FindStringSubmatch(text); len(matches) > 1 {
		step.Action = strings.TrimSpace(matches[1])
	}

	// 提取推理
	reasoningPattern := regexp.MustCompile(`(?i)reasoning:\s*(.+)`)
	if matches := reasoningPattern.FindStringSubmatch(text); len(matches) > 1 {
		step.Reasoning = strings.TrimSpace(matches[1])
	}

	// 提取结果
	resultPattern := regexp.MustCompile(`(?i)result:\s*(.+)`)
	if matches := resultPattern.FindStringSubmatch(text); len(matches) > 1 {
		step.Result = strings.TrimSpace(matches[1])
	}

	// 提取置信度
	confidencePattern := regexp.MustCompile(`(?i)confidence:\s*(0?\.\d+|1\.0?)`)
	if matches := confidencePattern.FindStringSubmatch(text); len(matches) > 1 {
		_, _ = fmt.Sscanf(matches[1], "%f", &step.Confidence)
	}

	// 提取下一步行动
	if strings.Contains(strings.ToLower(text), "complete") {
		step.NextAction = NextActionComplete
	} else if strings.Contains(strings.ToLower(text), "retry") {
		step.NextAction = NextActionRetry
	}

	// 如果没有提取到标题，使用整个文本作为结果
	if step.Title == "" && step.Result == "" {
		step.Result = text
		step.Title = "Reasoning step"
	}

	return step, nil
}
