package structured

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// OutputSpec 描述期望的结构化输出。
// Schema 字段目前仅用于文档/日志，不进行严格校验；RequiredFields 用于轻量必填校验。
type OutputSpec struct {
	Enabled         bool                   // 是否启用结构化解析
	Schema          map[string]interface{} // 可选的 JSON Schema 信息（当前仅透传）
	RequiredFields  []string               // 期望在顶层出现的字段
	AllowTextBackup bool                   // 解析失败时是否允许保留原始文本
}

// ParseResult 结构化解析结果。
type ParseResult struct {
	RawText       string      // 模型原始输出
	RawJSON       string      // 提取出的 JSON 文本
	Data          interface{} // JSON 解析结果
	MissingFields []string    // 缺失的必填字段
}

// Parser 结构化输出解析器接口。
type Parser interface {
	Parse(ctx context.Context, text string, spec OutputSpec) (*ParseResult, error)
}

// JSONParser 尝试从文本中提取 JSON 对象或数组并解析。
type JSONParser struct{}

// NewJSONParser 创建默认解析器。
func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

// Parse 实现结构化解析。
func (p *JSONParser) Parse(ctx context.Context, text string, spec OutputSpec) (*ParseResult, error) {
	if !spec.Enabled {
		return nil, errors.New("structured output is disabled")
	}

	rawJSON, err := extractJSONSegment(text)
	if err != nil {
		return nil, fmt.Errorf("extract json: %w", err)
	}

	var data interface{}
	if err := json.Unmarshal([]byte(rawJSON), &data); err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}

	missing := checkRequiredFields(data, spec.RequiredFields)

	return &ParseResult{
		RawText:       text,
		RawJSON:       rawJSON,
		Data:          data,
		MissingFields: missing,
	}, nil
}

// extractJSONSegment 尝试在文本中找到第一个配平的 JSON 对象/数组片段。
func extractJSONSegment(text string) (string, error) {
	start := -1
	var open, close rune

	for i, r := range text {
		if r == '{' || r == '[' {
			start = i
			if r == '{' {
				open, close = '{', '}'
			} else {
				open, close = '[', ']'
			}
			break
		}
	}

	if start == -1 {
		return "", errors.New("no json object/array found")
	}

	depth := 0
	for i, r := range text[start:] {
		if r == open {
			depth++
		} else if r == close {
			depth--
			if depth == 0 {
				return strings.TrimSpace(text[start : start+i+1]), nil
			}
		}
	}

	return "", errors.New("unbalanced json brackets")
}

// checkRequiredFields 校验顶层字段是否存在。
func checkRequiredFields(data interface{}, required []string) []string {
	if len(required) == 0 {
		return nil
	}

	obj, ok := data.(map[string]interface{})
	if !ok {
		return required
	}

	var missing []string
	for _, field := range required {
		if _, ok := obj[field]; !ok {
			missing = append(missing, field)
		}
	}
	return missing
}
