package dashboard

import (
	"time"

	"github.com/astercloud/aster/pkg/types"
	"github.com/google/uuid"
)

// TraceBuilder 追踪树构建器
type TraceBuilder struct {
	// 可配置的选项
	maxSpanDepth int
}

// NewTraceBuilder 创建追踪构建器
func NewTraceBuilder() *TraceBuilder {
	return &TraceBuilder{
		maxSpanDepth: 10,
	}
}

// BuildFromEvents 从事件列表构建追踪摘要列表
func (tb *TraceBuilder) BuildFromEvents(events []types.AgentEventEnvelope) []*TraceSummary {
	// 按 session/agent 分组事件
	sessions := tb.groupEventsBySession(events)

	traces := make([]*TraceSummary, 0, len(sessions))

	for sessionID, sessionEvents := range sessions {
		trace := tb.buildTraceSummary(sessionID, sessionEvents)
		if trace != nil {
			traces = append(traces, trace)
		}
	}

	return traces
}

// BuildTraceDetail 构建单个追踪的详情
func (tb *TraceBuilder) BuildTraceDetail(traceID string, allEvents []types.AgentEventEnvelope) *TraceDetail {
	// 简化处理：使用所有事件
	// 实际应该使用 trace_id 属性过滤
	if len(allEvents) == 0 {
		return nil
	}
	traceEvents := allEvents

	// 构建根节点
	rootSpan := tb.buildSpanTree(traceEvents)
	if rootSpan == nil {
		return nil
	}

	// 计算 Token 使用
	tokenUsage := tb.calculateTokenUsage(traceEvents)

	// 构建摘要
	summary := tb.buildTraceSummaryFromSpan(traceID, rootSpan, traceEvents)

	return &TraceDetail{
		TraceSummary: *summary,
		RootSpan:     rootSpan,
		TokenUsage:   tokenUsage,
	}
}

// groupEventsBySession 按会话分组事件
func (tb *TraceBuilder) groupEventsBySession(events []types.AgentEventEnvelope) map[string][]types.AgentEventEnvelope {
	sessions := make(map[string][]types.AgentEventEnvelope)

	// 简化实现：按时间窗口分组（同一分钟内的事件视为同一 session）
	// 实际应该使用 session_id 或 trace_id
	for _, env := range events {
		ts := time.UnixMilli(env.Bookmark.Timestamp)
		// 使用分钟级别的时间窗口作为 session key
		sessionKey := ts.Truncate(time.Minute).Format("2006-01-02T15:04")

		sessions[sessionKey] = append(sessions[sessionKey], env)
	}

	return sessions
}

// buildTraceSummary 构建追踪摘要
func (tb *TraceBuilder) buildTraceSummary(sessionID string, events []types.AgentEventEnvelope) *TraceSummary {
	if len(events) == 0 {
		return nil
	}

	// 找到开始和结束时间
	var startTime, endTime time.Time
	var hasError bool
	var errorMsg string
	var totalInput, totalOutput int64
	spanCount := 0

	for i, env := range events {
		ts := time.UnixMilli(env.Bookmark.Timestamp)

		if i == 0 || ts.Before(startTime) {
			startTime = ts
		}
		if ts.After(endTime) {
			endTime = ts
		}

		switch evt := env.Event.(type) {
		case types.MonitorTokenUsageEvent:
			totalInput += evt.InputTokens
			totalOutput += evt.OutputTokens
			spanCount++

		case types.MonitorStepCompleteEvent:
			spanCount++

		case types.MonitorToolExecutedEvent:
			spanCount++

		case types.MonitorErrorEvent:
			if evt.Severity == "error" {
				hasError = true
				errorMsg = evt.Message
			}
		}
	}

	status := TraceStatusOK
	if hasError {
		status = TraceStatusError
	}

	// 生成唯一 ID
	traceID := uuid.New().String()

	return &TraceSummary{
		ID:        traceID,
		Name:      "agent.run",
		StartTime: startTime,
		DurationMs: endTime.Sub(startTime).Milliseconds(),
		Status:     status,
		SpanCount:  spanCount,
		TokenUsage: TokenCount{
			Input:  totalInput,
			Output: totalOutput,
			Total:  totalInput + totalOutput,
		},
		ErrorMessage: errorMsg,
	}
}

// buildSpanTree 构建 Span 树
func (tb *TraceBuilder) buildSpanTree(events []types.AgentEventEnvelope) *TraceNode {
	if len(events) == 0 {
		return nil
	}

	// 找到时间范围
	var startTime, endTime time.Time
	for i, env := range events {
		ts := time.UnixMilli(env.Bookmark.Timestamp)
		if i == 0 || ts.Before(startTime) {
			startTime = ts
		}
		if ts.After(endTime) {
			endTime = ts
		}
	}

	// 创建根节点
	root := &TraceNode{
		ID:         uuid.New().String(),
		Name:       "agent.run",
		Type:       TraceNodeTypeAgent,
		StartTime:  startTime,
		EndTime:    &endTime,
		DurationMs: endTime.Sub(startTime).Milliseconds(),
		Status:     TraceStatusOK,
		Children:   make([]*TraceNode, 0),
	}

	// 构建子节点
	var currentLLMSpan *TraceNode
	var hasError bool

	for _, env := range events {
		ts := time.UnixMilli(env.Bookmark.Timestamp)

		switch evt := env.Event.(type) {
		case types.MonitorStepCompleteEvent:
			// 创建 LLM Span
			stepEnd := ts
			stepStart := ts.Add(-time.Duration(evt.DurationMs) * time.Millisecond)

			currentLLMSpan = &TraceNode{
				ID:         uuid.New().String(),
				Name:       "llm.chat",
				Type:       TraceNodeTypeLLM,
				StartTime:  stepStart,
				EndTime:    &stepEnd,
				DurationMs: evt.DurationMs,
				Status:     TraceStatusOK,
				Attributes: map[string]any{
					"step": evt.Step,
				},
				Children: make([]*TraceNode, 0),
			}
			root.Children = append(root.Children, currentLLMSpan)

		case types.MonitorToolExecutedEvent:
			// 创建 Tool Span
			toolEnd := ts
			toolStart := ts // No duration info available

			toolStatus := TraceStatusOK
			if evt.Call.Error != "" {
				toolStatus = TraceStatusError
			}

			toolSpan := &TraceNode{
				ID:         uuid.New().String(),
				Name:       "tool." + evt.Call.Name,
				Type:       TraceNodeTypeTool,
				StartTime:  toolStart,
				EndTime:    &toolEnd,
				DurationMs: 0, // Duration not available in ToolCallSnapshot
				Status:     toolStatus,
				Attributes: map[string]any{
					"tool_name": evt.Call.Name,
					"tool_id":   evt.Call.ID,
					"state":     string(evt.Call.State),
				},
				Children: make([]*TraceNode, 0),
			}

			// 添加到当前 LLM Span 或根节点
			if currentLLMSpan != nil {
				currentLLMSpan.Children = append(currentLLMSpan.Children, toolSpan)
			} else {
				root.Children = append(root.Children, toolSpan)
			}

		case types.MonitorTokenUsageEvent:
			// 更新当前 LLM Span 的属性
			if currentLLMSpan != nil {
				if currentLLMSpan.Attributes == nil {
					currentLLMSpan.Attributes = make(map[string]any)
				}
				currentLLMSpan.Attributes["input_tokens"] = evt.InputTokens
				currentLLMSpan.Attributes["output_tokens"] = evt.OutputTokens
				currentLLMSpan.Attributes["total_tokens"] = evt.TotalTokens
			}

		case types.MonitorErrorEvent:
			if evt.Severity == "error" {
				hasError = true
				// 标记相关 Span 为错误状态
				if currentLLMSpan != nil {
					currentLLMSpan.Status = TraceStatusError
					if currentLLMSpan.Attributes == nil {
						currentLLMSpan.Attributes = make(map[string]any)
					}
					currentLLMSpan.Attributes["error_message"] = evt.Message
					currentLLMSpan.Attributes["error_phase"] = evt.Phase
				}
			}
		}
	}

	if hasError {
		root.Status = TraceStatusError
	}

	return root
}

// buildTraceSummaryFromSpan 从 Span 构建摘要
func (tb *TraceBuilder) buildTraceSummaryFromSpan(traceID string, rootSpan *TraceNode, events []types.AgentEventEnvelope) *TraceSummary {
	tokenUsage := tb.calculateTokenUsage(events)

	var errorMsg string
	if rootSpan.Status == TraceStatusError {
		if msg, ok := rootSpan.Attributes["error_message"].(string); ok {
			errorMsg = msg
		}
	}

	return &TraceSummary{
		ID:           traceID,
		Name:         rootSpan.Name,
		StartTime:    rootSpan.StartTime,
		DurationMs:   rootSpan.DurationMs,
		Status:       rootSpan.Status,
		SpanCount:    tb.countSpans(rootSpan),
		TokenUsage:   tokenUsage,
		ErrorMessage: errorMsg,
	}
}

// calculateTokenUsage 计算 Token 使用量
func (tb *TraceBuilder) calculateTokenUsage(events []types.AgentEventEnvelope) TokenCount {
	var totalInput, totalOutput int64

	for _, env := range events {
		if evt, ok := env.Event.(types.MonitorTokenUsageEvent); ok {
			totalInput += evt.InputTokens
			totalOutput += evt.OutputTokens
		}
	}

	return TokenCount{
		Input:  totalInput,
		Output: totalOutput,
		Total:  totalInput + totalOutput,
	}
}

// countSpans 统计 Span 数量
func (tb *TraceBuilder) countSpans(node *TraceNode) int {
	if node == nil {
		return 0
	}

	count := 1
	for _, child := range node.Children {
		count += tb.countSpans(child)
	}

	return count
}

// FlattenSpans 将 Span 树展平为列表
func (tb *TraceBuilder) FlattenSpans(root *TraceNode) []*TraceNode {
	if root == nil {
		return nil
	}

	result := []*TraceNode{root}

	for _, child := range root.Children {
		result = append(result, tb.FlattenSpans(child)...)
	}

	return result
}
