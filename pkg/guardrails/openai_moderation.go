package guardrails

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/sashabaranov/go-openai"
)

// OpenAIModerationGuardrail OpenAI 内容审核防护栏
type OpenAIModerationGuardrail struct {
	model              string
	apiKey             string
	raiseForCategories []string
}

// ModerationCategory OpenAI 审核类别
type ModerationCategory string

const (
	CategorySexual                ModerationCategory = "sexual"
	CategorySexualMinors          ModerationCategory = "sexual/minors"
	CategoryHarassment            ModerationCategory = "harassment"
	CategoryHarassmentThreatening ModerationCategory = "harassment/threatening"
	CategoryHate                  ModerationCategory = "hate"
	CategoryHateThreatening       ModerationCategory = "hate/threatening"
	CategoryIllicit               ModerationCategory = "illicit"
	CategoryIllicitViolent        ModerationCategory = "illicit/violent"
	CategorySelfHarm              ModerationCategory = "self-harm"
	CategorySelfHarmIntent        ModerationCategory = "self-harm/intent"
	CategorySelfHarmInstructions  ModerationCategory = "self-harm/instructions"
	CategoryViolence              ModerationCategory = "violence"
	CategoryViolenceGraphic       ModerationCategory = "violence/graphic"
)

// NewOpenAIModerationGuardrail 创建 OpenAI 审核防护栏
func NewOpenAIModerationGuardrail(opts ...ModerationOption) *OpenAIModerationGuardrail {
	g := &OpenAIModerationGuardrail{
		model:  openai.ModerationTextStable,
		apiKey: os.Getenv("OPENAI_API_KEY"),
	}

	for _, opt := range opts {
		opt(g)
	}

	return g
}

// ModerationOption 审核选项
type ModerationOption func(*OpenAIModerationGuardrail)

// WithModerationModel 设置审核模型
func WithModerationModel(model string) ModerationOption {
	return func(g *OpenAIModerationGuardrail) {
		g.model = model
	}
}

// WithModerationAPIKey 设置 API Key
func WithModerationAPIKey(apiKey string) ModerationOption {
	return func(g *OpenAIModerationGuardrail) {
		g.apiKey = apiKey
	}
}

// WithRaiseForCategories 设置触发的类别
func WithRaiseForCategories(categories ...string) ModerationOption {
	return func(g *OpenAIModerationGuardrail) {
		g.raiseForCategories = categories
	}
}

// Name 返回防护栏名称
func (g *OpenAIModerationGuardrail) Name() string {
	return "OpenAIModeration"
}

// Description 返回防护栏描述
func (g *OpenAIModerationGuardrail) Description() string {
	return "使用 OpenAI Moderation API 检测违反内容政策的内容"
}

// Check 检查内容
func (g *OpenAIModerationGuardrail) Check(ctx context.Context, input *GuardrailInput) error {
	if g.apiKey == "" {
		return errors.New("OpenAI API key not configured")
	}

	client := openai.NewClient(g.apiKey)

	// 调用 Moderation API
	req := openai.ModerationRequest{
		Input: input.Content,
		Model: g.model,
	}

	resp, err := client.Moderations(ctx, req)
	if err != nil {
		return fmt.Errorf("moderation API error: %w", err)
	}

	if len(resp.Results) == 0 {
		return nil
	}

	result := resp.Results[0]

	// 检查是否标记
	if !result.Flagged {
		return nil
	}

	// 构建详细信息
	details := map[string]any{
		"categories":      result.Categories,
		"category_scores": result.CategoryScores,
	}

	// 检查是否触发
	triggerValidation := false
	if len(g.raiseForCategories) > 0 {
		// 检查特定类别
		for _, category := range g.raiseForCategories {
			if g.isCategoryFlagged(result, category) {
				triggerValidation = true
				break
			}
		}
	} else {
		// 任何类别标记都触发
		triggerValidation = true
	}

	if triggerValidation {
		return &GuardrailError{
			GuardrailName: g.Name(),
			Trigger:       CheckTriggerInputNotAllowed,
			Message:       "检测到违反 OpenAI 内容政策的内容",
			Details:       details,
		}
	}

	return nil
}

// isCategoryFlagged 检查类别是否被标记
func (g *OpenAIModerationGuardrail) isCategoryFlagged(result openai.Result, category string) bool {
	// 简化版本：检查主要类别
	switch category {
	case "sexual":
		return result.Categories.Sexual
	case "hate":
		return result.Categories.Hate
	case "harassment":
		return result.Categories.Harassment
	case "self-harm":
		return result.Categories.SelfHarm
	case "violence":
		return result.Categories.Violence
	default:
		return false
	}
}
