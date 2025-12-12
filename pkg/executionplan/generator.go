package executionplan

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/types"
)

// Generator 执行计划生成器
// 使用 LLM 根据用户请求生成执行计划
type Generator struct {
	provider provider.Provider
	tools    map[string]tools.Tool // 工具实例映射
}

// GeneratorOption 生成器选项
type GeneratorOption func(*Generator)

// NewGenerator 创建执行计划生成器
// toolMap: 工具名称到工具实例的映射
func NewGenerator(prov provider.Provider, toolMap map[string]tools.Tool, opts ...GeneratorOption) *Generator {
	g := &Generator{
		provider: prov,
		tools:    toolMap,
	}

	for _, opt := range opts {
		opt(g)
	}

	return g
}

// PlanRequest 计划生成请求
type PlanRequest struct {
	// UserRequest 用户的原始请求
	UserRequest string

	// Context 附加上下文信息（可选）
	Context string

	// AvailableTools 可用工具列表（如果为空，使用注册表中所有工具）
	AvailableTools []string

	// Options 执行选项（可选）
	Options *ExecutionOptions

	// Metadata 自定义元数据
	Metadata map[string]any
}

// planResponse LLM 返回的计划 JSON 结构
type planResponse struct {
	Description string         `json:"description"`
	Steps       []planStepResp `json:"steps"`
}

type planStepResp struct {
	ToolName    string         `json:"tool_name"`
	Description string         `json:"description"`
	Input       string         `json:"input,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
	DependsOn   []int          `json:"depends_on,omitempty"` // 依赖的步骤索引
}

// Generate 生成执行计划
func (g *Generator) Generate(ctx context.Context, req *PlanRequest) (*ExecutionPlan, error) {
	if req.UserRequest == "" {
		return nil, fmt.Errorf("user request cannot be empty")
	}

	// 构建可用工具描述
	toolDescriptions := g.buildToolDescriptions(req.AvailableTools)
	if toolDescriptions == "" {
		return nil, fmt.Errorf("no tools available for plan generation")
	}

	// 构建提示词
	prompt := g.buildPrompt(req.UserRequest, req.Context, toolDescriptions)

	// 调用 LLM 生成计划
	messages := []types.Message{
		{
			Role:    types.MessageRoleUser,
			Content: prompt,
		},
	}

	// 使用结构化输出（如果 provider 支持）
	opts := &provider.StreamOptions{
		MaxTokens:   4096,
		Temperature: 0.2, // 低温度以获得更确定性的输出
	}

	// 尝试使用结构化输出
	if g.provider.Capabilities().SupportStructuredOutput {
		opts.ResponseFormat = &provider.ResponseFormat{
			Type: provider.ResponseFormatJSONSchema,
			Name: "execution_plan",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"description": map[string]any{
						"type":        "string",
						"description": "执行计划的整体描述",
					},
					"steps": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"tool_name": map[string]any{
									"type":        "string",
									"description": "要调用的工具名称",
								},
								"description": map[string]any{
									"type":        "string",
									"description": "步骤描述",
								},
								"input": map[string]any{
									"type":        "string",
									"description": "工具输入（原始字符串）",
								},
								"parameters": map[string]any{
									"type":        "object",
									"description": "工具参数",
								},
								"depends_on": map[string]any{
									"type":        "array",
									"items":       map[string]any{"type": "integer"},
									"description": "依赖的步骤索引（从0开始）",
								},
							},
							"required": []string{"tool_name", "description"},
						},
					},
				},
				"required": []string{"description", "steps"},
			},
		}
	}

	// 调用 Complete
	resp, err := g.provider.Complete(ctx, messages, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate plan: %w", err)
	}

	// 解析响应
	plan, err := g.parseResponse(resp.Message.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plan response: %w", err)
	}

	// 设置选项和元数据
	if req.Options != nil {
		plan.Options = req.Options
	}
	if req.Metadata != nil {
		plan.Metadata = req.Metadata
	}

	return plan, nil
}

// buildToolDescriptions 构建工具描述文本
func (g *Generator) buildToolDescriptions(availableTools []string) string {
	var sb strings.Builder

	if len(g.tools) == 0 {
		return ""
	}

	// 如果指定了可用工具列表，则只包含这些工具
	toolSet := make(map[string]bool)
	if len(availableTools) > 0 {
		for _, name := range availableTools {
			toolSet[name] = true
		}
	}

	for name, tool := range g.tools {
		// 过滤工具
		if len(toolSet) > 0 && !toolSet[name] {
			continue
		}

		sb.WriteString(fmt.Sprintf("### %s\n", tool.Name()))
		sb.WriteString(fmt.Sprintf("描述: %s\n", tool.Description()))

		// 获取参数 Schema
		schema := tool.InputSchema()
		if props, ok := schema["properties"].(map[string]any); ok && len(props) > 0 {
			sb.WriteString("参数:\n")
			for paramName, spec := range props {
				specMap, ok := spec.(map[string]any)
				if !ok {
					continue
				}
				desc := ""
				if d, ok := specMap["description"].(string); ok {
					desc = d
				}
				typeName := "any"
				if t, ok := specMap["type"].(string); ok {
					typeName = t
				}
				sb.WriteString(fmt.Sprintf("  - %s (%s): %s\n", paramName, typeName, desc))
			}
		}

		// 添加工具示例（如果有）
		if exTool, ok := tool.(tools.ExampleableTool); ok {
			if examples := exTool.Examples(); len(examples) > 0 {
				sb.WriteString("示例:\n")
				for _, ex := range examples {
					sb.WriteString(fmt.Sprintf("  - %s\n", ex.Description))
				}
			}
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// buildPrompt 构建生成计划的提示词
func (g *Generator) buildPrompt(userRequest, context, toolDescriptions string) string {
	var sb strings.Builder

	sb.WriteString(`你是一个执行计划生成器。根据用户的请求，创建一个详细的执行计划，使用可用的工具来完成任务。

## 可用工具

`)
	sb.WriteString(toolDescriptions)

	if context != "" {
		sb.WriteString("\n## 上下文信息\n\n")
		sb.WriteString(context)
		sb.WriteString("\n")
	}

	sb.WriteString("\n## 用户请求\n\n")
	sb.WriteString(userRequest)

	sb.WriteString(`

## 输出要求

请以 JSON 格式输出执行计划：

{
  "description": "计划的整体描述",
  "steps": [
    {
      "tool_name": "工具名称",
      "description": "步骤描述",
      "input": "工具输入（可选）",
      "parameters": {
        "参数名": "参数值"
      },
      "depends_on": [0]  // 依赖的步骤索引（可选，从0开始）
    }
  ]
}

## 注意事项

1. 每个步骤必须使用上面列出的有效工具
2. 为每个工具提供所有必需的参数
3. 计划应该全面且有条理地解决用户的请求
4. 如果某些步骤依赖其他步骤的结果，请在 depends_on 中指明
5. 步骤描述应该清晰说明该步骤的目的

请生成执行计划：
`)

	return sb.String()
}

// parseResponse 解析 LLM 响应
func (g *Generator) parseResponse(content string) (*ExecutionPlan, error) {
	// 尝试直接解析 JSON
	var planResp planResponse
	if err := json.Unmarshal([]byte(content), &planResp); err != nil {
		// 如果直接解析失败，尝试提取 JSON 部分
		jsonStr, extractErr := extractJSON(content)
		if extractErr != nil {
			return nil, fmt.Errorf("failed to extract JSON from response: %w (original error: %v)", extractErr, err)
		}
		if err := json.Unmarshal([]byte(jsonStr), &planResp); err != nil {
			return nil, fmt.Errorf("failed to parse extracted JSON: %w", err)
		}
	}

	// 创建执行计划
	plan := NewExecutionPlan(planResp.Description)

	// 添加步骤
	for i, stepResp := range planResp.Steps {
		step := plan.AddStep(stepResp.ToolName, stepResp.Description, stepResp.Parameters)
		step.Input = stepResp.Input

		// 处理依赖关系
		if len(stepResp.DependsOn) > 0 {
			dependsOnIDs := make([]string, 0, len(stepResp.DependsOn))
			for _, depIdx := range stepResp.DependsOn {
				if depIdx >= 0 && depIdx < i {
					// 获取依赖步骤的 ID
					depStep := plan.GetStep(depIdx)
					if depStep != nil {
						dependsOnIDs = append(dependsOnIDs, depStep.ID)
					}
				}
			}
			step.DependsOn = dependsOnIDs
		}
	}

	// 设置状态为待审批
	if plan.Options != nil && plan.Options.RequireApproval && !plan.Options.AutoApprove {
		plan.Status = StatusPendingApproval
	}

	return plan, nil
}

// extractJSON 从文本中提取 JSON 部分
func extractJSON(text string) (string, error) {
	// 查找第一个 { 和最后一个 }
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")

	if start == -1 || end == -1 || end <= start {
		return "", fmt.Errorf("no valid JSON object found in text")
	}

	return text[start : end+1], nil
}

// FormatPlan 格式化执行计划为可读文本
func FormatPlan(plan *ExecutionPlan) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# 执行计划: %s\n\n", plan.Description))
	sb.WriteString(fmt.Sprintf("计划 ID: %s\n", plan.ID))
	sb.WriteString(fmt.Sprintf("状态: %s\n", plan.Status))

	if plan.Options != nil && plan.Options.RequireApproval {
		if plan.UserApproved {
			sb.WriteString("审批状态: ✅ 已审批\n")
		} else {
			sb.WriteString("审批状态: ⏳ 待审批\n")
		}
	}
	sb.WriteString("\n")

	sb.WriteString("## 执行步骤\n\n")
	for i, step := range plan.Steps {
		statusIcon := getStatusIcon(step.Status)
		sb.WriteString(fmt.Sprintf("### 步骤 %d: %s %s\n", i+1, step.Description, statusIcon))
		sb.WriteString(fmt.Sprintf("- 工具: `%s`\n", step.ToolName))

		if step.Input != "" {
			sb.WriteString(fmt.Sprintf("- 输入: %s\n", step.Input))
		}

		if len(step.Parameters) > 0 {
			sb.WriteString("- 参数:\n")
			for name, value := range step.Parameters {
				sb.WriteString(fmt.Sprintf("  - %s: %v\n", name, value))
			}
		}

		if len(step.DependsOn) > 0 {
			sb.WriteString(fmt.Sprintf("- 依赖: %v\n", step.DependsOn))
		}

		if step.Error != "" {
			sb.WriteString(fmt.Sprintf("- 错误: %s\n", step.Error))
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// getStatusIcon 获取状态图标
func getStatusIcon(status StepStatus) string {
	switch status {
	case StepStatusPending:
		return "⏳"
	case StepStatusRunning:
		return "🔄"
	case StepStatusCompleted:
		return "✅"
	case StepStatusFailed:
		return "❌"
	case StepStatusSkipped:
		return "⏭️"
	default:
		return ""
	}
}

// ValidatePlan 验证执行计划
func (g *Generator) ValidatePlan(plan *ExecutionPlan) []error {
	var errors []error

	if plan.Description == "" {
		errors = append(errors, fmt.Errorf("plan description is required"))
	}

	if len(plan.Steps) == 0 {
		errors = append(errors, fmt.Errorf("plan must have at least one step"))
	}

	for i, step := range plan.Steps {
		// 验证工具是否存在
		if _, ok := g.tools[step.ToolName]; !ok {
			errors = append(errors, fmt.Errorf("step %d: unknown tool '%s'", i+1, step.ToolName))
		}

		if step.Description == "" {
			errors = append(errors, fmt.Errorf("step %d: description is required", i+1))
		}

		// 验证依赖关系
		for _, depID := range step.DependsOn {
			found := false
			for j := 0; j < i; j++ {
				if plan.Steps[j].ID == depID {
					found = true
					break
				}
			}
			if !found {
				errors = append(errors, fmt.Errorf("step %d: invalid dependency '%s'", i+1, depID))
			}
		}
	}

	return errors
}
