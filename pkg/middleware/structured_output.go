package middleware

import (
	"context"
	"fmt"
	"log"

	"github.com/astercloud/aster/pkg/structured"
)

// StructuredOutputMiddleware 在模型响应后尝试解析结构化输出，并将结果写入 Metadata。
// - 若解析成功: Metadata["structured_data"] = 解析后的对象，Metadata["structured_raw_json"] = 原始 JSON 文本
// - 若解析失败: 根据配置决定是否回退；错误记录在 Metadata["structured_error"]
type StructuredOutputMiddleware struct {
	*BaseMiddleware
	spec       structured.OutputSpec
	parser     structured.Parser
	allowError bool
}

// StructuredOutputMiddlewareConfig 配置
type StructuredOutputMiddlewareConfig struct {
	Spec       structured.OutputSpec
	Parser     structured.Parser // 可选，默认 JSONParser
	AllowError bool              // 解析失败时是否忽略错误并回退到原始文本
	Priority   int               // 可选，默认 60
}

// NewStructuredOutputMiddleware 创建中间件实例
func NewStructuredOutputMiddleware(cfg *StructuredOutputMiddlewareConfig) (*StructuredOutputMiddleware, error) {
	if cfg == nil {
		return nil, fmt.Errorf("structured output config is nil")
	}

	parser := cfg.Parser
	if parser == nil {
		parser = structured.NewJSONParser()
	}

	priority := cfg.Priority
	if priority == 0 {
		priority = 60
	}

	return &StructuredOutputMiddleware{
		BaseMiddleware: NewBaseMiddleware("structured_output", priority),
		spec:           cfg.Spec,
		parser:         parser,
		allowError:     cfg.AllowError || cfg.Spec.AllowTextBackup,
	}, nil
}

// WrapModelCall 尝试解析结构化输出
func (m *StructuredOutputMiddleware) WrapModelCall(ctx context.Context, req *ModelRequest, handler ModelCallHandler) (*ModelResponse, error) {
	resp, err := handler(ctx, req)
	if err != nil || resp == nil {
		return resp, err
	}

	if !m.spec.Enabled {
		return resp, nil
	}

	content := resp.Message.GetContent()
	result, parseErr := m.parser.Parse(ctx, content, m.spec)
	if parseErr != nil {
		if m.allowError {
			log.Printf("[StructuredOutputMiddleware] parse failed: %v", parseErr)
			if resp.Metadata == nil {
				resp.Metadata = make(map[string]interface{})
			}
			resp.Metadata["structured_error"] = parseErr.Error()
			return resp, nil
		}
		return resp, fmt.Errorf("structured output parse failed: %w", parseErr)
	}

	if resp.Metadata == nil {
		resp.Metadata = make(map[string]interface{})
	}
	resp.Metadata["structured_data"] = result.Data
	resp.Metadata["structured_raw_json"] = result.RawJSON
	resp.Metadata["structured_missing_fields"] = result.MissingFields

	return resp, nil
}
