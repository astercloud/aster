// Package middleware provides the telemetry middleware for OpenTelemetry GenAI Semantic Conventions.
package middleware

import (
	"context"
	"time"

	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/telemetry"
	"github.com/astercloud/aster/pkg/telemetry/genai"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/types"
)

// TelemetryMiddlewareConfig 遥测中间件配置
type TelemetryMiddlewareConfig struct {
	// Tracer 自定义 tracer (可选，默认使用全局 tracer)
	Tracer telemetry.Tracer

	// AgentID Agent 唯一标识
	AgentID string

	// AgentName Agent 名称
	AgentName string

	// Provider LLM 提供商名称
	Provider string

	// Model 模型名称
	Model string

	// ConversationID 会话 ID (可选)
	ConversationID string

	// RecordPrompts 是否记录提示词内容 (默认 false，出于安全考虑)
	RecordPrompts bool

	// RecordCompletions 是否记录完成内容 (默认 false，出于安全考虑)
	RecordCompletions bool
}

// TelemetryMiddleware 遥测中间件
// 实现 OpenTelemetry GenAI Semantic Conventions
// 用于追踪 Agent 的 LLM 调用和工具执行
type TelemetryMiddleware struct {
	*BaseMiddleware

	tracer            telemetry.Tracer
	agentID           string
	agentName         string
	provider          string
	model             string
	conversationID    string
	recordPrompts     bool
	recordCompletions bool
}

var telemetryLog = logging.ForComponent("TelemetryMiddleware")

// NewTelemetryMiddleware 创建遥测中间件
func NewTelemetryMiddleware(config *TelemetryMiddlewareConfig) *TelemetryMiddleware {
	if config == nil {
		config = &TelemetryMiddlewareConfig{}
	}

	tracer := config.Tracer
	if tracer == nil {
		tracer = telemetry.GetGlobalTracer()
	}

	m := &TelemetryMiddleware{
		// Priority 5: 非常早执行，确保所有后续中间件和操作都在追踪范围内
		BaseMiddleware:    NewBaseMiddleware("telemetry", 5),
		tracer:            tracer,
		agentID:           config.AgentID,
		agentName:         config.AgentName,
		provider:          config.Provider,
		model:             config.Model,
		conversationID:    config.ConversationID,
		recordPrompts:     config.RecordPrompts,
		recordCompletions: config.RecordCompletions,
	}

	telemetryLog.Info(context.Background(), "initialized", map[string]any{
		"agent_id":   m.agentID,
		"agent_name": m.agentName,
		"provider":   m.provider,
		"model":      m.model,
	})

	return m
}

// Tools 返回中间件提供的工具 (无)
func (m *TelemetryMiddleware) Tools() []tools.Tool {
	return nil
}

// WrapModelCall 包装模型调用，添加 GenAI 追踪
func (m *TelemetryMiddleware) WrapModelCall(ctx context.Context, req *ModelRequest, handler ModelCallHandler) (*ModelResponse, error) {
	startTime := time.Now()

	// 构建 span 名称: "chat {model}"
	spanName := genai.ChatSpanName(m.model)

	// 构建初始属性
	attrs := []telemetry.Attribute{
		telemetry.String(genai.AttrOperationName, genai.OpChat),
		telemetry.String(genai.AttrProviderName, m.provider),
		telemetry.String(genai.AttrAgentID, m.agentID),
		telemetry.String(genai.AttrAgentName, m.agentName),
		telemetry.String(genai.AttrRequestModel, m.model),
	}

	// 添加会话 ID (如果有)
	if m.conversationID != "" {
		attrs = append(attrs, telemetry.String(genai.AttrConversationID, m.conversationID))
	}

	// 从 Metadata 中提取额外信息
	if req.Metadata != nil {
		if maxTokens, ok := req.Metadata["max_tokens"].(int); ok {
			attrs = append(attrs, telemetry.Int(genai.AttrRequestMaxTokens, maxTokens))
		}
		if temperature, ok := req.Metadata["temperature"].(float64); ok {
			attrs = append(attrs, telemetry.Float64(genai.AttrRequestTemperature, temperature))
		}
		if topP, ok := req.Metadata["top_p"].(float64); ok {
			attrs = append(attrs, telemetry.Float64(genai.AttrRequestTopP, topP))
		}
		// 覆盖模型名称（如果 metadata 中指定了）
		if model, ok := req.Metadata["model"].(string); ok && model != "" {
			attrs = append(attrs, telemetry.String(genai.AttrRequestModel, model))
		}
		// 覆盖 provider（如果 metadata 中指定了）
		if provider, ok := req.Metadata["provider"].(string); ok && provider != "" {
			attrs = append(attrs, telemetry.String(genai.AttrProviderName, provider))
		}
	}

	// 开始 span (CLIENT 类型，因为是调用外部 LLM)
	ctx, span := m.tracer.StartSpan(ctx, spanName,
		telemetry.WithSpanKind(telemetry.SpanKindClient),
		telemetry.WithAttributes(attrs...),
	)
	defer span.End()

	// 记录提示词事件 (如果启用)
	if m.recordPrompts && len(req.Messages) > 0 {
		m.recordPromptEvent(span, req)
	}

	// 调用下一层
	resp, err := handler(ctx, req)

	// 计算延迟
	latencyMs := time.Since(startTime).Milliseconds()
	span.SetAttributes(telemetry.Int64(genai.AttrLatencyTotal, latencyMs))

	// 处理错误
	if err != nil {
		span.RecordError(err)
		span.SetStatus(telemetry.StatusCodeError, err.Error())
		m.setErrorTypeAttribute(span, err)
		return resp, err
	}

	// 记录完成事件 (如果启用)
	if m.recordCompletions && resp != nil {
		m.recordCompletionEvent(span, resp)
	}

	// 从响应 Metadata 中提取 token 使用情况
	if resp != nil && resp.Metadata != nil {
		m.extractTokenUsage(span, resp.Metadata)
		m.extractResponseInfo(span, resp.Metadata)
	}

	span.SetStatus(telemetry.StatusCodeOK, "")
	return resp, nil
}

// WrapToolCall 包装工具调用，添加 GenAI 追踪
func (m *TelemetryMiddleware) WrapToolCall(ctx context.Context, req *ToolCallRequest, handler ToolCallHandler) (*ToolCallResponse, error) {
	startTime := time.Now()

	// 构建 span 名称: "execute_tool {tool_name}"
	spanName := genai.ToolSpanName(req.ToolName)

	// 构建初始属性
	attrs := []telemetry.Attribute{
		telemetry.String(genai.AttrOperationName, genai.OpExecuteTool),
		telemetry.String(genai.AttrToolName, req.ToolName),
		telemetry.String(genai.AttrToolCallID, req.ToolCallID),
		telemetry.String(genai.AttrAgentID, m.agentID),
		telemetry.String(genai.AttrAgentName, m.agentName),
	}

	// 添加会话 ID (如果有)
	if m.conversationID != "" {
		attrs = append(attrs, telemetry.String(genai.AttrConversationID, m.conversationID))
	}

	// 开始 span (INTERNAL 类型，因为是进程内操作)
	ctx, span := m.tracer.StartSpan(ctx, spanName,
		telemetry.WithSpanKind(telemetry.SpanKindInternal),
		telemetry.WithAttributes(attrs...),
	)
	defer span.End()

	// 记录工具调用事件
	span.AddEvent(genai.EventToolCall,
		telemetry.String(genai.AttrToolName, req.ToolName),
		telemetry.String(genai.AttrToolCallID, req.ToolCallID),
	)

	// 调用下一层
	resp, err := handler(ctx, req)

	// 计算延迟
	latencyMs := time.Since(startTime).Milliseconds()
	span.SetAttributes(telemetry.Int64(genai.AttrLatencyTotal, latencyMs))

	// 处理错误
	if err != nil {
		span.RecordError(err)
		span.SetStatus(telemetry.StatusCodeError, err.Error())
		return resp, err
	}

	// 记录工具结果事件
	span.AddEvent(genai.EventToolResult,
		telemetry.String(genai.AttrToolName, req.ToolName),
		telemetry.String(genai.AttrToolCallID, req.ToolCallID),
		telemetry.Int64("duration_ms", latencyMs),
	)

	span.SetStatus(telemetry.StatusCodeOK, "")
	return resp, nil
}

// OnAgentStart Agent 启动时创建根 span
func (m *TelemetryMiddleware) OnAgentStart(ctx context.Context, agentID string) error {
	telemetryLog.Info(ctx, "agent starting", map[string]any{
		"agent_id":   agentID,
		"agent_name": m.agentName,
	})
	return nil
}

// OnAgentStop Agent 停止时的回调
func (m *TelemetryMiddleware) OnAgentStop(ctx context.Context, agentID string) error {
	telemetryLog.Info(ctx, "agent stopping", map[string]any{
		"agent_id":   agentID,
		"agent_name": m.agentName,
	})
	return nil
}

// recordPromptEvent 记录提示词事件
func (m *TelemetryMiddleware) recordPromptEvent(span telemetry.Span, req *ModelRequest) {
	attrs := []telemetry.Attribute{
		telemetry.Int("message_count", len(req.Messages)),
	}

	// 添加系统提示词长度（不记录内容）
	if req.SystemPrompt != "" {
		attrs = append(attrs, telemetry.Int("system_prompt_length", len(req.SystemPrompt)))
	}

	// 添加工具数量
	if len(req.Tools) > 0 {
		attrs = append(attrs, telemetry.Int("tool_count", len(req.Tools)))
	}

	span.AddEvent(genai.EventPrompt, attrs...)
}

// recordCompletionEvent 记录完成事件
func (m *TelemetryMiddleware) recordCompletionEvent(span telemetry.Span, resp *ModelResponse) {
	attrs := []telemetry.Attribute{}

	// 记录响应内容长度（不记录具体内容）
	if resp.Message.Content != "" {
		attrs = append(attrs, telemetry.Int("content_length", len(resp.Message.Content)))
	}

	// 记录工具调用数量
	if len(resp.Message.ContentBlocks) > 0 {
		toolCallCount := 0
		for _, block := range resp.Message.ContentBlocks {
			// 使用类型断言检查是否为 ToolUseBlock
			if _, ok := block.(*types.ToolUseBlock); ok {
				toolCallCount++
			}
		}
		if toolCallCount > 0 {
			attrs = append(attrs, telemetry.Int("tool_call_count", toolCallCount))
		}
	}

	if len(attrs) > 0 {
		span.AddEvent(genai.EventCompletion, attrs...)
	}
}

// extractTokenUsage 从响应 Metadata 中提取 token 使用情况
func (m *TelemetryMiddleware) extractTokenUsage(span telemetry.Span, metadata map[string]any) {
	if inputTokens, ok := metadata["input_tokens"].(int64); ok {
		span.SetAttributes(telemetry.Int64(genai.AttrUsageInputTokens, inputTokens))
	} else if inputTokens, ok := metadata["input_tokens"].(int); ok {
		span.SetAttributes(telemetry.Int64(genai.AttrUsageInputTokens, int64(inputTokens)))
	}

	if outputTokens, ok := metadata["output_tokens"].(int64); ok {
		span.SetAttributes(telemetry.Int64(genai.AttrUsageOutputTokens, outputTokens))
	} else if outputTokens, ok := metadata["output_tokens"].(int); ok {
		span.SetAttributes(telemetry.Int64(genai.AttrUsageOutputTokens, int64(outputTokens)))
	}

	// 支持 usage 嵌套结构
	if usage, ok := metadata["usage"].(map[string]any); ok {
		if inputTokens, ok := usage["input_tokens"].(int); ok {
			span.SetAttributes(telemetry.Int64(genai.AttrUsageInputTokens, int64(inputTokens)))
		}
		if outputTokens, ok := usage["output_tokens"].(int); ok {
			span.SetAttributes(telemetry.Int64(genai.AttrUsageOutputTokens, int64(outputTokens)))
		}
	}
}

// extractResponseInfo 从响应 Metadata 中提取响应信息
func (m *TelemetryMiddleware) extractResponseInfo(span telemetry.Span, metadata map[string]any) {
	if responseID, ok := metadata["response_id"].(string); ok {
		span.SetAttributes(telemetry.String(genai.AttrResponseID, responseID))
	}

	if responseModel, ok := metadata["model"].(string); ok {
		span.SetAttributes(telemetry.String(genai.AttrResponseModel, responseModel))
	}

	if finishReason, ok := metadata["finish_reason"].(string); ok {
		span.SetAttributes(telemetry.String(genai.AttrResponseFinishReason, finishReason))
	} else if stopReason, ok := metadata["stop_reason"].(string); ok {
		span.SetAttributes(telemetry.String(genai.AttrResponseFinishReason, stopReason))
	}
}

// setErrorTypeAttribute 设置错误类型属性
func (m *TelemetryMiddleware) setErrorTypeAttribute(span telemetry.Span, err error) {
	errStr := err.Error()
	var errType string

	// 简单的错误类型推断
	switch {
	case contains(errStr, "timeout"):
		errType = genai.ErrorTypeTimeout
	case contains(errStr, "rate limit") || contains(errStr, "429"):
		errType = genai.ErrorTypeRateLimit
	case contains(errStr, "authentication") || contains(errStr, "401"):
		errType = genai.ErrorTypeAuthentication
	case contains(errStr, "permission") || contains(errStr, "403"):
		errType = genai.ErrorTypePermission
	case contains(errStr, "not found") || contains(errStr, "404"):
		errType = genai.ErrorTypeNotFound
	case contains(errStr, "context length") || contains(errStr, "too long"):
		errType = genai.ErrorTypeContextLengthExceeded
	case contains(errStr, "content filter") || contains(errStr, "blocked"):
		errType = genai.ErrorTypeContentFilter
	case contains(errStr, "500") || contains(errStr, "server error"):
		errType = genai.ErrorTypeServerError
	default:
		errType = "unknown"
	}

	span.SetAttributes(telemetry.String(genai.AttrErrorType, errType))
}

// contains 检查字符串是否包含子串 (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > 0 && len(substr) > 0 &&
			(s[0] == substr[0] || s[0]+32 == substr[0] || s[0] == substr[0]+32) &&
			contains(s[1:], substr[1:]) ||
		contains(s[1:], substr))
}

// SetConversationID 动态设置会话 ID
func (m *TelemetryMiddleware) SetConversationID(conversationID string) {
	m.conversationID = conversationID
}

// SetModel 动态设置模型名称
func (m *TelemetryMiddleware) SetModel(model string) {
	m.model = model
}

// SetProvider 动态设置提供商
func (m *TelemetryMiddleware) SetProvider(provider string) {
	m.provider = provider
}

// GetTracer 获取 tracer 实例
func (m *TelemetryMiddleware) GetTracer() telemetry.Tracer {
	return m.tracer
}
