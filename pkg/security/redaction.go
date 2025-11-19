package security

import (
	"context"
	"fmt"
	"strings"
)

// ContentRedactor 接口定义了内容脱敏的方法
type ContentRedactor interface {
	Redact(text string) string
}

// PIIRedactor PII脱敏器
type PIIRedactor struct {
	detector PIIDetector
	replacer map[PIIType]string
}

// NewPIIRedactor 创建PII脱敏器
func NewPIIRedactor(detector PIIDetector) *PIIRedactor {
	return &PIIRedactor{
		detector: detector,
		replacer: map[PIIType]string{
			PIIEmail:       "[EMAIL]",
			PIIPhone:       "[PHONE]",
			PIISSNus:       "[SSN]",
			PIICreditCard:  "[CARD]",
			PIIChineseID:   "[ID]",
			PIIPassport:    "[PASSPORT]",
			PIIAddress:     "[ADDRESS]",
			PIIBankAccount: "[ACCOUNT]",
			PIIIPAddress:   "[IP]",
			PIIDateOfBirth: "[DATE]",
			PIICustom:      "[PII]",
		},
	}
}

// Redact 脱敏文本中的PII信息
func (r *PIIRedactor) Redact(text string) string {
	if r.detector == nil || text == "" {
		return text
	}

	ctx := context.Background()
	matches, err := r.detector.Detect(ctx, text)
	if err != nil || len(matches) == 0 {
		return text
	}

	// 按位置倒序排列，从后往前替换，避免位置偏移
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		replacement := r.replacer[match.Type]
		if replacement == "" {
			replacement = "[PII]"
		}

		text = text[:match.Start] + replacement + text[match.End:]
	}

	return text
}

// SetReplacement 设置特定PII类型的替换文本
func (r *PIIRedactor) SetReplacement(piiType PIIType, replacement string) {
	r.replacer[piiType] = replacement
}

// GetReplacement 获取特定PII类型的替换文本
func (r *PIIRedactor) GetReplacement(piiType PIIType) string {
	if replacement, exists := r.replacer[piiType]; exists {
		return replacement
	}
	return "[PII]"
}

// RedactWithMasking 使用掩码脱敏
func (r *PIIRedactor) RedactWithMasking(text string, maskLength int) string {
	if r.detector == nil || text == "" {
		return text
	}

	ctx := context.Background()
	matches, err := r.detector.Detect(ctx, text)
	if err != nil || len(matches) == 0 {
		return text
	}

	// 按位置倒序排列
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		maskedValue := r.maskValue(match.Value, maskLength)
		text = text[:match.Start] + maskedValue + text[match.End:]
	}

	return text
}

// maskValue 创建掩码值
func (r *PIIRedactor) maskValue(value string, maskLength int) string {
	if value == "" {
		return value
	}

	if maskLength <= 0 {
		return strings.Repeat("*", len(value))
	}

	if len(value) <= maskLength {
		return strings.Repeat("*", len(value))
	}

	// 保留首尾字符，中间用*替换
	end := len(value) - 1
	if len(value) <= 2 {
		return strings.Repeat("*", len(value))
	}

	masked := string(value[0]) + strings.Repeat("*", maskLength) + string(value[end])
	return masked
}

// AddCustomPIIType 添加自定义PII类型
func (r *PIIRedactor) AddCustomPIIType(piiType PIIType, replacement string) {
	r.replacer[piiType] = replacement
}

// AnalyzeAndRedact 分析并脱敏，返回脱敏报告
func (r *PIIRedactor) AnalyzeAndRedact(text string) (*RedactionResult, string) {
	if r.detector == nil {
		return &RedactionResult{}, text
	}

	ctx := context.Background()
	matches, err := r.detector.Detect(ctx, text)
	if err != nil {
		return &RedactionResult{
			Error: err.Error(),
		}, text
	}

	if len(matches) == 0 {
		return &RedactionResult{}, text
	}

	result := &RedactionResult{
		OriginalLength: len(text),
		PIIFound:       true,
		MatchedTypes:   make(map[PIIType]int),
		TotalMatches:   len(matches),
		Matches:        matches,
	}

	// 统计每种PII类型的数量
	for _, match := range matches {
		result.MatchedTypes[match.Type]++
	}

	// 执行脱敏
	redactedText := r.Redact(text)
	result.RedactedLength = len(redactedText)

	return result, redactedText
}

// RedactionResult 脱敏结果报告
type RedactionResult struct {
	OriginalLength int             `json:"original_length"`
	RedactedLength int             `json:"redacted_length"`
	PIIFound       bool            `json:"pii_found"`
	MatchedTypes   map[PIIType]int `json:"matched_types"`
	TotalMatches   int             `json:"total_matches"`
	Matches        []PIIMatch      `json:"matches"`
	Error          string          `json:"error,omitempty"`
}

// GetSummary 获取脱敏摘要
func (r *RedactionResult) GetSummary() string {
	if r.Error != "" {
		return fmt.Sprintf("Error during redaction: %s", r.Error)
	}

	if !r.PIIFound {
		return "No PII found in text"
	}

	var typeInfo []string
	for piiType, count := range r.MatchedTypes {
		typeInfo = append(typeInfo, fmt.Sprintf("%s(%d)", piiType, count))
	}

	return fmt.Sprintf("Found %d PII instances: %s. Text length reduced from %d to %d characters.",
		r.TotalMatches, strings.Join(typeInfo, ", "), r.OriginalLength, r.RedactedLength)
}

// CompositeRedactor 组合脱敏器
type CompositeRedactor struct {
	redactors []ContentRedactor
}

// NewCompositeRedactor 创建组合脱敏器
func NewCompositeRedactor(redactors ...ContentRedactor) *CompositeRedactor {
	return &CompositeRedactor{
		redactors: redactors,
	}
}

// Redact 使用所有脱敏器进行脱敏
func (c *CompositeRedactor) Redact(text string) string {
	result := text
	for _, redactor := range c.redactors {
		result = redactor.Redact(result)
	}
	return result
}

// AddRedactor 添加脱敏器
func (c *CompositeRedactor) AddRedactor(redactor ContentRedactor) {
	c.redactors = append(c.redactors, redactor)
}
