package guardrails

import (
	"context"
)

// Guardrail 防护栏接口 - 用于检查输入内容的安全性
type Guardrail interface {
	// Check 同步检查输入内容
	// 如果检测到违规内容，返回 error
	Check(ctx context.Context, input *GuardrailInput) error

	// Name 返回防护栏的名称
	Name() string

	// Description 返回防护栏的描述
	Description() string
}

// GuardrailInput 防护栏输入
type GuardrailInput struct {
	// Content 文本内容
	Content string

	// Images 图片 URL 列表
	Images []string

	// Metadata 额外元数据
	Metadata map[string]interface{}

	// UserID 用户 ID（可选）
	UserID string

	// SessionID 会话 ID（可选）
	SessionID string
}

// GuardrailError 防护栏错误
type GuardrailError struct {
	// GuardrailName 触发错误的防护栏名称
	GuardrailName string

	// Trigger 触发类型
	Trigger CheckTrigger

	// Message 错误消息
	Message string

	// Details 详细信息
	Details map[string]interface{}

	// ShouldMask 是否应该掩码而不是拒绝
	ShouldMask bool

	// MaskedContent 掩码后的内容（如果 ShouldMask=true）
	MaskedContent string
}

func (e *GuardrailError) Error() string {
	return e.Message
}

// CheckTrigger 检查触发类型
type CheckTrigger string

const (
	// CheckTriggerInputNotAllowed 输入不允许
	CheckTriggerInputNotAllowed CheckTrigger = "input_not_allowed"

	// CheckTriggerPIIDetected PII 检测到
	CheckTriggerPIIDetected CheckTrigger = "pii_detected"

	// CheckTriggerPromptInjection 提示注入检测到
	CheckTriggerPromptInjection CheckTrigger = "prompt_injection"

	// CheckTriggerToxicContent 有毒内容检测到
	CheckTriggerToxicContent CheckTrigger = "toxic_content"

	// CheckTriggerCustom 自定义触发
	CheckTriggerCustom CheckTrigger = "custom"
)

// GuardrailChain 防护栏链 - 依次执行多个防护栏
type GuardrailChain struct {
	guardrails []Guardrail
}

// NewGuardrailChain 创建防护栏链
func NewGuardrailChain(guardrails ...Guardrail) *GuardrailChain {
	return &GuardrailChain{
		guardrails: guardrails,
	}
}

// Check 依次执行所有防护栏检查
func (gc *GuardrailChain) Check(ctx context.Context, input *GuardrailInput) error {
	for _, g := range gc.guardrails {
		if err := g.Check(ctx, input); err != nil {
			return err
		}
	}
	return nil
}

// Add 添加防护栏到链中
func (gc *GuardrailChain) Add(guardrail Guardrail) *GuardrailChain {
	gc.guardrails = append(gc.guardrails, guardrail)
	return gc
}

// Guardrails 获取所有防护栏
func (gc *GuardrailChain) Guardrails() []Guardrail {
	return gc.guardrails
}
