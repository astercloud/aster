// Package dashboard provides observability dashboard functionality for Aster.
// It aggregates metrics, builds trace trees, and calculates costs from agent events.
package dashboard

import (
	"time"
)

// OverviewStats 概览统计
type OverviewStats struct {
	ActiveAgents   int        `json:"active_agents"`
	ActiveSessions int        `json:"active_sessions"`
	TotalRequests  int64      `json:"total_requests"`
	TokenUsage     TokenCount `json:"token_usage"`
	Cost           CostAmount `json:"cost"`
	ErrorRate      float64    `json:"error_rate"`
	AvgLatencyMs   int64      `json:"avg_latency_ms"`
	Period         string     `json:"period"` // "24h", "7d", "30d"
	UpdatedAt      time.Time  `json:"updated_at"`
}

// TokenCount Token 计数
type TokenCount struct {
	Input  int64 `json:"input"`
	Output int64 `json:"output"`
	Total  int64 `json:"total"`
}

// CostAmount 成本金额
type CostAmount struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// TraceNode 追踪节点
type TraceNode struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       TraceNodeType  `json:"type"`
	StartTime  time.Time      `json:"start_time"`
	EndTime    *time.Time     `json:"end_time,omitempty"`
	DurationMs int64          `json:"duration_ms"`
	Status     TraceStatus    `json:"status"`
	Attributes map[string]any `json:"attributes,omitempty"`
	Children   []*TraceNode   `json:"children,omitempty"`
}

// TraceNodeType 追踪节点类型
type TraceNodeType string

const (
	TraceNodeTypeAgent      TraceNodeType = "agent"
	TraceNodeTypeLLM        TraceNodeType = "llm"
	TraceNodeTypeTool       TraceNodeType = "tool"
	TraceNodeTypeMiddleware TraceNodeType = "middleware"
)

// TraceStatus 追踪状态
type TraceStatus string

const (
	TraceStatusOK      TraceStatus = "ok"
	TraceStatusError   TraceStatus = "error"
	TraceStatusRunning TraceStatus = "running"
)

// TraceSummary 追踪摘要（用于列表展示）
type TraceSummary struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	AgentID      string      `json:"agent_id,omitempty"`
	AgentName    string      `json:"agent_name,omitempty"`
	StartTime    time.Time   `json:"start_time"`
	DurationMs   int64       `json:"duration_ms"`
	Status       TraceStatus `json:"status"`
	SpanCount    int         `json:"span_count"`
	TokenUsage   TokenCount  `json:"token_usage"`
	ErrorMessage string      `json:"error_message,omitempty"`
}

// TraceDetail 追踪详情
type TraceDetail struct {
	TraceSummary

	RootSpan   *TraceNode `json:"root_span"`
	TokenUsage TokenCount `json:"token_usage"`
	Cost       CostAmount `json:"cost"`
}

// TraceQueryOpts 追踪查询选项
type TraceQueryOpts struct {
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	AgentID   string     `json:"agent_id,omitempty"`
	Status    string     `json:"status,omitempty"`
	Limit     int        `json:"limit,omitempty"`
	Offset    int        `json:"offset,omitempty"`
}

// TraceListResult 追踪列表结果
type TraceListResult struct {
	Traces  []*TraceSummary `json:"traces"`
	Total   int64           `json:"total"`
	HasMore bool            `json:"has_more"`
}

// TokenUsageStats Token 使用统计
type TokenUsageStats struct {
	Period  string                `json:"period"` // "hour", "day", "week", "month"
	Total   TokenCount            `json:"total"`
	ByAgent map[string]TokenCount `json:"by_agent,omitempty"`
	ByModel map[string]TokenCount `json:"by_model,omitempty"`
	Trend   []TokenTrendPoint     `json:"trend,omitempty"`
	Cost    CostAmount            `json:"cost"`
}

// TokenTrendPoint Token 趋势数据点
type TokenTrendPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Input     int64     `json:"input"`
	Output    int64     `json:"output"`
}

// TokenQueryOpts Token 查询选项
type TokenQueryOpts struct {
	Period    string     `json:"period"` // "hour", "day", "week", "month"
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	AgentID   string     `json:"agent_id,omitempty"`
	Model     string     `json:"model,omitempty"`
}

// CostBreakdown 成本分解
type CostBreakdown struct {
	Period  string                `json:"period"`
	Total   CostAmount            `json:"total"`
	ByAgent map[string]CostAmount `json:"by_agent,omitempty"`
	ByModel map[string]CostAmount `json:"by_model,omitempty"`
	Trend   []CostTrendPoint      `json:"trend,omitempty"`
}

// CostTrendPoint 成本趋势数据点
type CostTrendPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Amount    float64   `json:"amount"`
}

// CostQueryOpts 成本查询选项
type CostQueryOpts struct {
	Period    string     `json:"period"`
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	AgentID   string     `json:"agent_id,omitempty"`
}

// ModelPricing 模型定价
type ModelPricing struct {
	Model           string  `json:"model"`
	InputPricePerM  float64 `json:"input_price_per_m"`  // 每百万 token 价格
	OutputPricePerM float64 `json:"output_price_per_m"` // 每百万 token 价格
	Currency        string  `json:"currency"`
}

// DefaultModelPricing 默认模型定价表（美元/百万 token）
var DefaultModelPricing = map[string]ModelPricing{
	"claude-3-5-sonnet-20241022": {
		Model:           "claude-3-5-sonnet-20241022",
		InputPricePerM:  3.0,
		OutputPricePerM: 15.0,
		Currency:        "USD",
	},
	"claude-3-5-haiku-20241022": {
		Model:           "claude-3-5-haiku-20241022",
		InputPricePerM:  0.8,
		OutputPricePerM: 4.0,
		Currency:        "USD",
	},
	"claude-sonnet-4-20250514": {
		Model:           "claude-sonnet-4-20250514",
		InputPricePerM:  3.0,
		OutputPricePerM: 15.0,
		Currency:        "USD",
	},
	"gpt-4o": {
		Model:           "gpt-4o",
		InputPricePerM:  2.5,
		OutputPricePerM: 10.0,
		Currency:        "USD",
	},
	"gpt-4o-mini": {
		Model:           "gpt-4o-mini",
		InputPricePerM:  0.15,
		OutputPricePerM: 0.6,
		Currency:        "USD",
	},
	"deepseek-chat": {
		Model:           "deepseek-chat",
		InputPricePerM:  0.14,
		OutputPricePerM: 0.28,
		Currency:        "USD",
	},
}

// EventStreamMessage WebSocket 事件流消息
type EventStreamMessage struct {
	Type      string    `json:"type"` // "event", "heartbeat", "error"
	Channel   string    `json:"channel,omitempty"`
	Event     any       `json:"event,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Error     string    `json:"error,omitempty"`
}

// EventSubscription 事件订阅配置
type EventSubscription struct {
	Channels   []string `json:"channels"` // "monitor", "progress", "control"
	AgentID    string   `json:"agent_id,omitempty"`
	EventTypes []string `json:"event_types,omitempty"`
}

// PerformanceStats 性能统计
type PerformanceStats struct {
	Period       string                        `json:"period"`
	TTFT         LatencyPercentiles            `json:"ttft"`           // Time to First Token
	TPOT         LatencyPercentiles            `json:"tpot"`           // Time Per Output Token
	ToolLatency  map[string]LatencyPercentiles `json:"tool_latency"`   // 按工具
	AvgLoopCount float64                       `json:"avg_loop_count"` // 平均循环次数
	RequestCount int64                         `json:"request_count"`
	ErrorCount   int64                         `json:"error_count"`
	ErrorRate    float64                       `json:"error_rate"`
}

// LatencyPercentiles 延迟百分位数
type LatencyPercentiles struct {
	P50 int64 `json:"p50"`
	P95 int64 `json:"p95"`
	P99 int64 `json:"p99"`
	Avg int64 `json:"avg"`
	Max int64 `json:"max"`
}

// Insight 改进建议
type Insight struct {
	ID          string         `json:"id"`
	Type        InsightType    `json:"type"`
	Severity    string         `json:"severity"` // "info", "warning", "critical"
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Suggestion  string         `json:"suggestion"`
	Data        map[string]any `json:"data,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

// InsightType 建议类型
type InsightType string

const (
	InsightTypePerformance InsightType = "performance"
	InsightTypeCost        InsightType = "cost"
	InsightTypeReliability InsightType = "reliability"
	InsightTypeUsage       InsightType = "usage"
)

// SessionTimelineEntry 会话时间线条目
type SessionTimelineEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // "message", "tool_call", "checkpoint", "error"
	Data      any       `json:"data"`
}

// SessionTimeline 会话时间线
type SessionTimeline struct {
	SessionID string                 `json:"session_id"`
	AgentID   string                 `json:"agent_id"`
	Entries   []SessionTimelineEntry `json:"entries"`
	StartTime time.Time              `json:"start_time"`
	EndTime   *time.Time             `json:"end_time,omitempty"`
}
