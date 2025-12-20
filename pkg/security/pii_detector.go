package security

import (
	"context"
	"fmt"
	"strings"
)

// PIIMatch 表示一个 PII 匹配结果。
type PIIMatch struct {
	Type       PIIType             // PII 类型
	Value      string              // 原始值
	Start      int                 // 起始位置
	End        int                 // 结束位置
	Confidence float64             // 置信度（0.0-1.0）
	Severity   PIISensitivityLevel // 敏感度级别
}

// PIIDetector PII 检测器接口。
type PIIDetector interface {
	// Detect 检测文本中的所有 PII。
	Detect(ctx context.Context, text string) ([]PIIMatch, error)

	// DetectTypes 检测指定类型的 PII。
	DetectTypes(ctx context.Context, text string, types ...PIIType) ([]PIIMatch, error)

	// ContainsPII 快速检查文本是否包含 PII。
	ContainsPII(ctx context.Context, text string) (bool, error)
}

// RegexPIIDetector 基于正则表达式的 PII 检测器。
type RegexPIIDetector struct {
	patterns []PIIPattern
}

// NewRegexPIIDetector 创建正则表达式 PII 检测器。
func NewRegexPIIDetector() *RegexPIIDetector {
	return &RegexPIIDetector{
		patterns: PIIPatternRegistry,
	}
}

// NewRegexPIIDetectorWithTypes 创建检测特定类型的 PII 检测器。
func NewRegexPIIDetectorWithTypes(types ...PIIType) *RegexPIIDetector {
	return &RegexPIIDetector{
		patterns: GetPatternsByType(types...),
	}
}

// Detect 检测文本中的所有 PII。
func (d *RegexPIIDetector) Detect(ctx context.Context, text string) ([]PIIMatch, error) {
	return d.DetectTypes(ctx, text)
}

// DetectTypes 检测指定类型的 PII。
func (d *RegexPIIDetector) DetectTypes(ctx context.Context, text string, types ...PIIType) ([]PIIMatch, error) {
	if text == "" {
		return nil, nil
	}

	patterns := d.patterns
	if len(types) > 0 {
		patterns = GetPatternsByType(types...)
	}

	var matches []PIIMatch
	matchedRanges := make(map[string]bool) // 防止重复匹配

	for _, pattern := range patterns {
		// 检查上下文是否取消
		select {
		case <-ctx.Done():
			return matches, ctx.Err()
		default:
		}

		indices := pattern.Regex.FindAllStringIndex(text, -1)
		if indices == nil {
			continue
		}

		for _, loc := range indices {
			start, end := loc[0], loc[1]
			value := text[start:end]

			// 检查是否重复
			key := fmt.Sprintf("%d-%d", start, end)
			if matchedRanges[key] {
				continue
			}

			// 额外验证（如果有）
			if pattern.Validator != nil && !pattern.Validator(value) {
				continue
			}

			matchedRanges[key] = true
			matches = append(matches, PIIMatch{
				Type:       pattern.Type,
				Value:      value,
				Start:      start,
				End:        end,
				Confidence: 0.9, // 正则匹配的置信度
				Severity:   GetSensitivityLevel(pattern.Type),
			})
		}
	}

	// 按位置排序
	sortMatchesByPosition(matches)

	return matches, nil
}

// ContainsPII 快速检查文本是否包含 PII。
func (d *RegexPIIDetector) ContainsPII(ctx context.Context, text string) (bool, error) {
	matches, err := d.Detect(ctx, text)
	return len(matches) > 0, err
}

// sortMatchesByPosition 按起始位置排序匹配结果。
func sortMatchesByPosition(matches []PIIMatch) {
	// 简单的冒泡排序
	for i := range len(matches) - 1 {
		for j := range len(matches) - i - 1 {
			if matches[j].Start > matches[j+1].Start {
				matches[j], matches[j+1] = matches[j+1], matches[j]
			}
		}
	}
}

// CompositePIIDetector 组合多个检测器。
type CompositePIIDetector struct {
	detectors []PIIDetector
}

// NewCompositePIIDetector 创建组合检测器。
func NewCompositePIIDetector(detectors ...PIIDetector) *CompositePIIDetector {
	return &CompositePIIDetector{
		detectors: detectors,
	}
}

// Detect 使用所有检测器检测 PII。
func (d *CompositePIIDetector) Detect(ctx context.Context, text string) ([]PIIMatch, error) {
	var allMatches []PIIMatch

	for _, detector := range d.detectors {
		matches, err := detector.Detect(ctx, text)
		if err != nil {
			return allMatches, err
		}
		allMatches = append(allMatches, matches...)
	}

	// 去重
	allMatches = deduplicateMatches(allMatches)

	return allMatches, nil
}

// DetectTypes 检测指定类型的 PII。
func (d *CompositePIIDetector) DetectTypes(ctx context.Context, text string, types ...PIIType) ([]PIIMatch, error) {
	var allMatches []PIIMatch

	for _, detector := range d.detectors {
		matches, err := detector.DetectTypes(ctx, text, types...)
		if err != nil {
			return allMatches, err
		}
		allMatches = append(allMatches, matches...)
	}

	allMatches = deduplicateMatches(allMatches)
	return allMatches, nil
}

// ContainsPII 检查是否包含 PII。
func (d *CompositePIIDetector) ContainsPII(ctx context.Context, text string) (bool, error) {
	for _, detector := range d.detectors {
		hasPII, err := detector.ContainsPII(ctx, text)
		if err != nil {
			return false, err
		}
		if hasPII {
			return true, nil
		}
	}
	return false, nil
}

// deduplicateMatches 去除重复的匹配。
func deduplicateMatches(matches []PIIMatch) []PIIMatch {
	seen := make(map[string]bool)
	var result []PIIMatch

	for _, match := range matches {
		// 使用位置和值作为唯一键
		key := fmt.Sprintf("%d-%d-%s", match.Start, match.End, match.Value)
		if !seen[key] {
			seen[key] = true
			result = append(result, match)
		}
	}

	sortMatchesByPosition(result)
	return result
}

// PIIDetectionResult 检测结果汇总。
type PIIDetectionResult struct {
	Matches      []PIIMatch
	HasPII       bool
	PIITypes     []PIIType
	HighestRisk  PIISensitivityLevel
	TotalMatches int
}

// AnalyzePII 分析文本中的 PII 并返回详细报告。
func AnalyzePII(ctx context.Context, text string, detector PIIDetector) (*PIIDetectionResult, error) {
	matches, err := detector.Detect(ctx, text)
	if err != nil {
		return nil, err
	}

	result := &PIIDetectionResult{
		Matches:      matches,
		HasPII:       len(matches) > 0,
		TotalMatches: len(matches),
		HighestRisk:  SensitivityLow,
	}

	// 统计 PII 类型和最高风险级别
	typeMap := make(map[PIIType]bool)
	for _, match := range matches {
		typeMap[match.Type] = true
		if match.Severity > result.HighestRisk {
			result.HighestRisk = match.Severity
		}
	}

	for piiType := range typeMap {
		result.PIITypes = append(result.PIITypes, piiType)
	}

	return result, nil
}

// PIIContext PII 的上下文信息（用于更好的检测）。
type PIIContext struct {
	// Language 文本语言（zh/en等）
	Language string

	// AllowedTypes 允许的 PII 类型（白名单）
	AllowedTypes []PIIType

	// IgnorePatterns 忽略的模式（如公司邮箱域名）
	IgnorePatterns []string

	// MinConfidence 最低置信度阈值
	MinConfidence float64
}

// FilterMatchesByContext 根据上下文过滤匹配结果。
func FilterMatchesByContext(matches []PIIMatch, ctx *PIIContext) []PIIMatch {
	if ctx == nil {
		return matches
	}

	var filtered []PIIMatch

	// 构建允许类型映射
	var allowedMap map[PIIType]bool
	if len(ctx.AllowedTypes) > 0 {
		allowedMap = make(map[PIIType]bool)
		for _, t := range ctx.AllowedTypes {
			allowedMap[t] = true
		}
	}

	for _, match := range matches {
		// 检查置信度
		if match.Confidence < ctx.MinConfidence {
			continue
		}

		// 检查是否在白名单中
		if allowedMap != nil && allowedMap[match.Type] {
			continue // 白名单中的类型，跳过
		}

		// 检查忽略模式
		shouldIgnore := false
		for _, pattern := range ctx.IgnorePatterns {
			if strings.Contains(match.Value, pattern) {
				shouldIgnore = true
				break
			}
		}
		if shouldIgnore {
			continue
		}

		filtered = append(filtered, match)
	}

	return filtered
}
