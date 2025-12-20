package security

import (
	"context"
	"fmt"
	"sync"

	"github.com/astercloud/aster/pkg/middleware"
	"github.com/astercloud/aster/pkg/types"
)

// PIIRedactionMiddleware PII 脱敏中间件。
// 在消息发送到 LLM 前自动检测和脱敏 PII。
type PIIRedactionMiddleware struct {
	*middleware.BaseMiddleware

	redactor       *Redactor
	enableTracking bool                  // 是否启用追踪（用于还原）
	tracking       map[string][]PIIMatch // 追踪每个 Agent 的 PII 匹配
	mu             sync.RWMutex          // 保护 tracking map
}

// PIIMiddlewareConfig PII 中间件配置。
type PIIMiddlewareConfig struct {
	Detector       PIIDetector       // PII 检测器
	Strategy       RedactionStrategy // 脱敏策略
	EnableTracking bool              // 是否启用 PII 追踪
	Priority       int               // 中间件优先级（默认 200）
}

// NewPIIRedactionMiddleware 创建 PII 脱敏中间件。
func NewPIIRedactionMiddleware(cfg PIIMiddlewareConfig) *PIIRedactionMiddleware {
	if cfg.Priority == 0 {
		cfg.Priority = 200 // 默认优先级，在核心中间件之后，用户中间件之前
	}

	return &PIIRedactionMiddleware{
		BaseMiddleware: middleware.NewBaseMiddleware("pii-redaction", cfg.Priority),
		redactor:       NewRedactor(cfg.Detector, cfg.Strategy),
		enableTracking: cfg.EnableTracking,
		tracking:       make(map[string][]PIIMatch),
	}
}

// NewDefaultPIIMiddleware 创建默认配置的 PII 中间件。
func NewDefaultPIIMiddleware() *PIIRedactionMiddleware {
	return NewPIIRedactionMiddleware(PIIMiddlewareConfig{
		Detector:       NewRegexPIIDetector(),
		Strategy:       NewAdaptiveStrategy(),
		EnableTracking: true,
		Priority:       200,
	})
}

// WrapModelCall 包装模型调用，在发送前脱敏 PII。
func (m *PIIRedactionMiddleware) WrapModelCall(ctx context.Context, req *middleware.ModelRequest, handler middleware.ModelCallHandler) (*middleware.ModelResponse, error) {
	// 提取 Agent ID（如果有）
	agentID := "unknown"
	if req.Metadata != nil {
		if id, ok := req.Metadata["agent_id"].(string); ok {
			agentID = id
		}
	}

	// 处理每个消息
	for i, msg := range req.Messages {
		redactedMsg, err := m.redactMessage(ctx, msg, agentID)
		if err != nil {
			return nil, fmt.Errorf("failed to redact message %d: %w", i, err)
		}
		req.Messages[i] = redactedMsg
	}

	// 处理 System Prompt
	if req.SystemPrompt != "" {
		redacted, _, err := m.redactWithTracking(ctx, req.SystemPrompt, agentID)
		if err != nil {
			return nil, fmt.Errorf("failed to redact system prompt: %w", err)
		}
		req.SystemPrompt = redacted
	}

	// 调用下一层处理器
	resp, err := handler(ctx, req)
	if err != nil {
		return nil, err
	}

	// 响应通常不需要脱敏（LLM 生成的内容）
	// 但如果需要，可以在这里添加

	return resp, nil
}

// redactMessage 脱敏单个消息。
func (m *PIIRedactionMiddleware) redactMessage(ctx context.Context, msg types.Message, agentID string) (types.Message, error) {
	// 处理简单文本内容
	if msg.Content != "" {
		redacted, _, err := m.redactWithTracking(ctx, msg.Content, agentID)
		if err != nil {
			return msg, err
		}
		msg.Content = redacted
	}

	// 处理复杂内容块
	if len(msg.ContentBlocks) > 0 {
		for i, block := range msg.ContentBlocks {
			redactedBlock, err := m.redactContentBlock(ctx, block, agentID)
			if err != nil {
				return msg, err
			}
			msg.ContentBlocks[i] = redactedBlock
		}
	}

	return msg, nil
}

// redactContentBlock 脱敏内容块。
func (m *PIIRedactionMiddleware) redactContentBlock(ctx context.Context, block types.ContentBlock, agentID string) (types.ContentBlock, error) {
	// 类型断言处理不同的内容块类型
	switch b := block.(type) {
	case *types.TextBlock:
		if b.Text != "" {
			redacted, _, err := m.redactWithTracking(ctx, b.Text, agentID)
			if err != nil {
				return block, err
			}
			return &types.TextBlock{Text: redacted}, nil
		}
	case *types.ToolResultBlock:
		// 工具结果也可能包含 PII
		if b.Content != "" {
			redacted, _, err := m.redactWithTracking(ctx, b.Content, agentID)
			if err != nil {
				return block, err
			}
			return &types.ToolResultBlock{
				ToolUseID: b.ToolUseID,
				Content:   redacted,
				IsError:   b.IsError,
			}, nil
		}
	}
	// 其他类型（如 ToolUseBlock）不需要脱敏
	return block, nil
}

// redactWithTracking 脱敏文本并追踪 PII 匹配。
func (m *PIIRedactionMiddleware) redactWithTracking(ctx context.Context, text string, agentID string) (string, []PIIMatch, error) {
	if text == "" {
		return text, nil, nil
	}

	// 检测和脱敏
	matches, err := m.redactor.detector.Detect(ctx, text)
	if err != nil {
		return text, nil, err
	}

	if len(matches) == 0 {
		return text, nil, nil
	}

	// 如果启用追踪，保存匹配信息
	if m.enableTracking {
		m.mu.Lock()
		m.tracking[agentID] = append(m.tracking[agentID], matches...)
		m.mu.Unlock()
	}

	// 脱敏
	redacted, err := m.redactor.Redact(ctx, text)
	if err != nil {
		return text, nil, err
	}

	return redacted, matches, nil
}

// OnAgentStop 在 Agent 停止时清除追踪信息。
func (m *PIIRedactionMiddleware) OnAgentStop(ctx context.Context, agentID string) error {
	m.ClearTracking(agentID)
	return nil
}

// GetPIIMatches 获取 Agent 的 PII 匹配记录。
func (m *PIIRedactionMiddleware) GetPIIMatches(agentID string) []PIIMatch {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tracking[agentID]
}

// ClearTracking 清除 Agent 的追踪信息。
func (m *PIIRedactionMiddleware) ClearTracking(agentID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tracking, agentID)
}

// GetPIISummary 获取 PII 检测摘要。
func (m *PIIRedactionMiddleware) GetPIISummary(agentID string) *PIIDetectionSummary {
	m.mu.RLock()
	matches := m.tracking[agentID]
	m.mu.RUnlock()

	if len(matches) == 0 {
		return &PIIDetectionSummary{
			AgentID:      agentID,
			HasPII:       false,
			TotalMatches: 0,
			TypeCounts:   make(map[PIIType]int),
		}
	}

	summary := &PIIDetectionSummary{
		AgentID:      agentID,
		HasPII:       true,
		TotalMatches: len(matches),
		TypeCounts:   make(map[PIIType]int),
		HighestRisk:  SensitivityLow,
	}

	// 统计类型和风险
	for _, match := range matches {
		summary.TypeCounts[match.Type]++
		if match.Severity > summary.HighestRisk {
			summary.HighestRisk = match.Severity
		}
	}

	return summary
}

// PIIDetectionSummary PII 检测摘要。
type PIIDetectionSummary struct {
	AgentID      string
	HasPII       bool
	TotalMatches int
	TypeCounts   map[PIIType]int
	HighestRisk  PIISensitivityLevel
}

// ConditionalPIIMiddleware 条件 PII 脱敏中间件。
// 根据上下文条件决定是否脱敏。
type ConditionalPIIMiddleware struct {
	*middleware.BaseMiddleware

	redactor  *Redactor
	condition func(context.Context, *middleware.ModelRequest) bool
}

// ConditionalPIIConfig 条件 PII 中间件配置。
type ConditionalPIIConfig struct {
	Detector  PIIDetector
	Strategy  RedactionStrategy
	Condition func(context.Context, *middleware.ModelRequest) bool // 判断是否需要脱敏
	Priority  int
}

// NewConditionalPIIMiddleware 创建条件 PII 中间件。
func NewConditionalPIIMiddleware(cfg ConditionalPIIConfig) *ConditionalPIIMiddleware {
	if cfg.Priority == 0 {
		cfg.Priority = 200
	}

	return &ConditionalPIIMiddleware{
		BaseMiddleware: middleware.NewBaseMiddleware("conditional-pii-redaction", cfg.Priority),
		redactor:       NewRedactor(cfg.Detector, cfg.Strategy),
		condition:      cfg.Condition,
	}
}

// WrapModelCall 包装模型调用。
func (m *ConditionalPIIMiddleware) WrapModelCall(ctx context.Context, req *middleware.ModelRequest, handler middleware.ModelCallHandler) (*middleware.ModelResponse, error) {
	// 检查是否需要脱敏
	if m.condition != nil && !m.condition(ctx, req) {
		return handler(ctx, req)
	}

	// 脱敏逻辑（简化版，不追踪）
	for i, msg := range req.Messages {
		if msg.Content != "" {
			redacted, err := m.redactor.Redact(ctx, msg.Content)
			if err != nil {
				return nil, err
			}
			msg.Content = redacted
		}
		req.Messages[i] = msg
	}

	return handler(ctx, req)
}

// TODO: PII Redaction Tool 可以作为独立工具实现，供 Agent 主动调用
// 需要实现 tools.Tool 接口
