package guardrails

import (
	"context"
	"maps"
	"regexp"
	"strings"
)

// PIIDetectionGuardrail PII（个人身份信息）检测防护栏
type PIIDetectionGuardrail struct {
	maskPII        bool
	piiPatterns    map[string]*regexp.Regexp
	customPatterns map[string]*regexp.Regexp
}

// PIIType PII 类型
type PIIType string

const (
	PIITypeSSN        PIIType = "SSN"        // 社会安全号
	PIITypeCreditCard PIIType = "CreditCard" // 信用卡
	PIITypeEmail      PIIType = "Email"      // 邮箱
	PIITypePhone      PIIType = "Phone"      // 电话号码
	PIITypeIPAddress  PIIType = "IPAddress"  // IP 地址
)

// NewPIIDetectionGuardrail 创建 PII 检测防护栏
func NewPIIDetectionGuardrail(opts ...PIIOption) *PIIDetectionGuardrail {
	g := &PIIDetectionGuardrail{
		piiPatterns: make(map[string]*regexp.Regexp),
		maskPII:     false,
	}

	// 默认启用所有检查
	g.piiPatterns[string(PIITypeSSN)] = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	g.piiPatterns[string(PIITypeCreditCard)] = regexp.MustCompile(`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`)
	g.piiPatterns[string(PIITypeEmail)] = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`)
	g.piiPatterns[string(PIITypePhone)] = regexp.MustCompile(`\b\d{3}[\s.-]?\d{3}[\s.-]?\d{4}\b`)
	g.piiPatterns[string(PIITypeIPAddress)] = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)

	for _, opt := range opts {
		opt(g)
	}

	return g
}

// PIIOption PII 选项
type PIIOption func(*PIIDetectionGuardrail)

// WithMaskPII 启用 PII 掩码（而不是拒绝）
func WithMaskPII(mask bool) PIIOption {
	return func(g *PIIDetectionGuardrail) {
		g.maskPII = mask
	}
}

// WithDisableSSNCheck 禁用 SSN 检查
func WithDisableSSNCheck() PIIOption {
	return func(g *PIIDetectionGuardrail) {
		delete(g.piiPatterns, string(PIITypeSSN))
	}
}

// WithDisableCreditCardCheck 禁用信用卡检查
func WithDisableCreditCardCheck() PIIOption {
	return func(g *PIIDetectionGuardrail) {
		delete(g.piiPatterns, string(PIITypeCreditCard))
	}
}

// WithDisableEmailCheck 禁用邮箱检查
func WithDisableEmailCheck() PIIOption {
	return func(g *PIIDetectionGuardrail) {
		delete(g.piiPatterns, string(PIITypeEmail))
	}
}

// WithDisablePhoneCheck 禁用电话检查
func WithDisablePhoneCheck() PIIOption {
	return func(g *PIIDetectionGuardrail) {
		delete(g.piiPatterns, string(PIITypePhone))
	}
}

// WithCustomPattern 添加自定义 PII 模式
func WithCustomPattern(name string, pattern *regexp.Regexp) PIIOption {
	return func(g *PIIDetectionGuardrail) {
		if g.customPatterns == nil {
			g.customPatterns = make(map[string]*regexp.Regexp)
		}
		g.customPatterns[name] = pattern
	}
}

// Name 返回防护栏名称
func (g *PIIDetectionGuardrail) Name() string {
	return "PIIDetection"
}

// Description 返回防护栏描述
func (g *PIIDetectionGuardrail) Description() string {
	return "检测输入中的个人身份信息（PII）"
}

// Check 检查内容
func (g *PIIDetectionGuardrail) Check(ctx context.Context, input *GuardrailInput) error {
	content := input.Content
	detectedPII := []string{}

	// 检查所有模式
	allPatterns := make(map[string]*regexp.Regexp)
	maps.Copy(allPatterns, g.piiPatterns)
	maps.Copy(allPatterns, g.customPatterns)

	for piiType, pattern := range allPatterns {
		if pattern.MatchString(content) {
			detectedPII = append(detectedPII, piiType)
		}
	}

	if len(detectedPII) == 0 {
		return nil
	}

	// 如果启用掩码
	if g.maskPII {
		maskedContent := content
		for _, piiType := range detectedPII {
			pattern := allPatterns[piiType]
			maskedContent = pattern.ReplaceAllStringFunc(maskedContent, func(match string) string {
				return strings.Repeat("*", len(match))
			})
		}

		return &GuardrailError{
			GuardrailName: g.Name(),
			Trigger:       CheckTriggerPIIDetected,
			Message:       "检测到 PII，已自动掩码",
			Details: map[string]any{
				"detected_pii": detectedPII,
			},
			ShouldMask:    true,
			MaskedContent: maskedContent,
		}
	}

	// 拒绝输入
	return &GuardrailError{
		GuardrailName: g.Name(),
		Trigger:       CheckTriggerPIIDetected,
		Message:       "检测到输入中包含个人身份信息（PII）",
		Details: map[string]any{
			"detected_pii": detectedPII,
		},
	}
}
