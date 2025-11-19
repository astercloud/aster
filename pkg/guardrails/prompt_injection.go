package guardrails

import (
	"context"
	"regexp"
	"strings"
)

// PromptInjectionGuardrail 提示注入攻击检测防护栏
type PromptInjectionGuardrail struct {
	patterns        []*regexp.Regexp
	keywordPatterns []string
	caseSensitive   bool
}

// NewPromptInjectionGuardrail 创建提示注入检测防护栏
func NewPromptInjectionGuardrail(opts ...PromptInjectionOption) *PromptInjectionGuardrail {
	g := &PromptInjectionGuardrail{
		patterns:      make([]*regexp.Regexp, 0),
		caseSensitive: false,
	}

	// 默认的危险模式
	defaultPatterns := []string{
		// 忽略前面的指令
		`ignore\s+(all\s+)?(previous|prior|above)\s+instructions?`,
		`disregard\s+(all\s+)?(previous|prior|above)\s+(instructions?|directives?)`,

		// 系统提示词泄露
		`(show|reveal|display|print|tell me|what('?s| is| are))\s+(your|the)\s+(system\s+)?(prompt|instruction|directive)`,
		`repeat\s+(your|the)\s+(system\s+)?(prompt|instruction)`,

		// 角色切换
		`you\s+are\s+now`,
		`act\s+as\s+(if\s+)?you\s+(are|were)`,
		`pretend\s+(to\s+be|you\s+are)`,

		// 规则绕过
		`ignore\s+(all\s+)?rules?`,
		`bypass\s+(all\s+)?rules?`,
		`override\s+(all\s+)?(restrictions?|rules?)`,

		// DAN (Do Anything Now) 模式
		`you\s+can\s+do\s+anything\s+now`,
		`DAN\s+mode`,

		// 编码绕过
		`base64`,
		`rot13`,
		`hex\s+encode`,
	}

	for _, pattern := range defaultPatterns {
		if !g.caseSensitive {
			pattern = `(?i)` + pattern
		}
		re := regexp.MustCompile(pattern)
		g.patterns = append(g.patterns, re)
	}

	// 危险关键词
	g.keywordPatterns = []string{
		"system:",
		"<|im_start|>",
		"<|im_end|>",
		"[INST]",
		"[/INST]",
	}

	for _, opt := range opts {
		opt(g)
	}

	return g
}

// PromptInjectionOption 提示注入选项
type PromptInjectionOption func(*PromptInjectionGuardrail)

// WithCaseSensitive 启用大小写敏感
func WithCaseSensitive(sensitive bool) PromptInjectionOption {
	return func(g *PromptInjectionGuardrail) {
		g.caseSensitive = sensitive
	}
}

// WithCustomInjectionPattern 添加自定义注入模式
func WithCustomInjectionPattern(pattern string) PromptInjectionOption {
	return func(g *PromptInjectionGuardrail) {
		re := regexp.MustCompile(pattern)
		g.patterns = append(g.patterns, re)
	}
}

// WithCustomKeyword 添加自定义关键词
func WithCustomKeyword(keyword string) PromptInjectionOption {
	return func(g *PromptInjectionGuardrail) {
		g.keywordPatterns = append(g.keywordPatterns, keyword)
	}
}

// Name 返回防护栏名称
func (g *PromptInjectionGuardrail) Name() string {
	return "PromptInjection"
}

// Description 返回防护栏描述
func (g *PromptInjectionGuardrail) Description() string {
	return "检测提示注入攻击尝试"
}

// Check 检查内容
func (g *PromptInjectionGuardrail) Check(ctx context.Context, input *GuardrailInput) error {
	content := input.Content
	if !g.caseSensitive {
		content = strings.ToLower(content)
	}

	detectedPatterns := []string{}

	// 检查正则模式
	for _, pattern := range g.patterns {
		if pattern.MatchString(content) {
			match := pattern.FindString(content)
			detectedPatterns = append(detectedPatterns, match)
		}
	}

	// 检查关键词
	contentLower := strings.ToLower(input.Content)
	for _, keyword := range g.keywordPatterns {
		if strings.Contains(contentLower, strings.ToLower(keyword)) {
			detectedPatterns = append(detectedPatterns, keyword)
		}
	}

	if len(detectedPatterns) > 0 {
		return &GuardrailError{
			GuardrailName: g.Name(),
			Trigger:       CheckTriggerPromptInjection,
			Message:       "检测到可能的提示注入攻击",
			Details: map[string]interface{}{
				"detected_patterns": detectedPatterns,
				"pattern_count":     len(detectedPatterns),
			},
		}
	}

	return nil
}
