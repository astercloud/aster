package dashboard

import (
	"context"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/events"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/types"
)

// Aggregator 指标聚合器
type Aggregator struct {
	eventBus       *events.EventBus // 可选，用于实时事件
	store          store.Store
	costCalculator *CostCalculator
	traceBuilder   *TraceBuilder

	// 缓存
	mu            sync.RWMutex
	tokenCache    map[string]*TokenUsageStats // key: period
	traceCache    map[string]*TraceDetail     // key: traceID
	cacheTTL      time.Duration
	lastCacheTime time.Time
}

// NewAggregator 创建聚合器
func NewAggregator(st store.Store) *Aggregator {
	return &Aggregator{
		eventBus:       nil, // 可选
		store:          st,
		costCalculator: NewCostCalculator(nil),
		traceBuilder:   NewTraceBuilder(),
		tokenCache:     make(map[string]*TokenUsageStats),
		traceCache:     make(map[string]*TraceDetail),
		cacheTTL:       30 * time.Second,
	}
}

// NewAggregatorWithEventBus 创建带 EventBus 的聚合器
func NewAggregatorWithEventBus(eb *events.EventBus, st store.Store) *Aggregator {
	agg := NewAggregator(st)
	agg.eventBus = eb
	return agg
}

// SetEventBus 设置 EventBus（用于实时事件）
func (a *Aggregator) SetEventBus(eb *events.EventBus) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.eventBus = eb
}

// EventBusProvider 提供 EventBus 列表的接口
type EventBusProvider interface {
	GetEventBuses() []*events.EventBus
}

// GetOverviewStatsFromEventBuses 从多个 EventBus 获取概览统计
func (a *Aggregator) GetOverviewStatsFromEventBuses(ctx context.Context, period string, provider EventBusProvider) (*OverviewStats, error) {
	if period == "" {
		period = "24h"
	}

	now := time.Now()
	var startTime time.Time

	switch period {
	case "1h":
		startTime = now.Add(-1 * time.Hour)
	case "24h":
		startTime = now.Add(-24 * time.Hour)
	case "7d":
		startTime = now.Add(-7 * 24 * time.Hour)
	case "30d":
		startTime = now.Add(-30 * 24 * time.Hour)
	default:
		startTime = now.Add(-24 * time.Hour)
	}

	// 从所有 EventBus 收集事件
	var allEvents []types.AgentEventEnvelope
	if provider != nil {
		for _, eb := range provider.GetEventBuses() {
			if eb != nil {
				events := eb.GetTimelineFiltered(func(env types.AgentEventEnvelope) bool {
					// Bookmark.Timestamp 是秒级时间戳
					ts := time.Unix(env.Bookmark.Timestamp, 0)
					return ts.After(startTime)
				})
				allEvents = append(allEvents, events...)
			}
		}
	}

	// 如果没有事件，返回基本统计（使用 provider 的 agent 数量）
	if len(allEvents) == 0 {
		activeAgents := 0
		if provider != nil {
			activeAgents = len(provider.GetEventBuses())
		}
		return &OverviewStats{
			ActiveAgents:   activeAgents,
			ActiveSessions: 0,
			TotalRequests:  0,
			TokenUsage:     TokenCount{},
			Cost:           CostAmount{Currency: "USD"},
			ErrorRate:      0,
			AvgLatencyMs:   0,
			Period:         period,
			UpdatedAt:      now,
		}, nil
	}

	// 统计数据
	var totalInput, totalOutput int64
	var errorCount, requestCount int64
	var totalDuration int64

	for _, env := range allEvents {
		// 处理类型化事件（本地 Agent）
		switch evt := env.Event.(type) {
		case *types.MonitorTokenUsageEvent:
			totalInput += evt.InputTokens
			totalOutput += evt.OutputTokens
			requestCount++

		case *types.MonitorStepCompleteEvent:
			totalDuration += evt.DurationMs

		case *types.MonitorErrorEvent:
			if evt.Severity == "error" {
				errorCount++
			}

		case *types.MonitorStateChangedEvent:
			// 从事件属性中提取 agentID
			// 这里简化处理

		case map[string]any:
			// 处理远程 Agent 发送的 map[string]any 类型事件
			eventType, _ := evt["event_type"].(string)
			switch eventType {
			case "token_usage":
				if input, ok := evt["input_tokens"].(float64); ok {
					totalInput += int64(input)
				}
				if output, ok := evt["output_tokens"].(float64); ok {
					totalOutput += int64(output)
				}
				requestCount++

			case "step_complete":
				if duration, ok := evt["duration_ms"].(float64); ok {
					totalDuration += int64(duration)
				}

			case "error":
				if severity, ok := evt["severity"].(string); ok && severity == "error" {
					errorCount++
				}
			}
		}
	}

	// 计算活跃 agent 数量（从 provider 获取）
	activeAgents := 0
	if provider != nil {
		activeAgents = len(provider.GetEventBuses())
	}

	// 计算平均延迟
	avgLatency := int64(0)
	if requestCount > 0 {
		avgLatency = totalDuration / requestCount
	}

	// 计算成本
	cost := a.costCalculator.Calculate(totalInput, totalOutput, "")

	// 计算错误率
	errorRate := 0.0
	if requestCount > 0 {
		errorRate = float64(errorCount) / float64(requestCount)
	}

	return &OverviewStats{
		ActiveAgents:   activeAgents,
		ActiveSessions: 0, // TODO: 从 registry 获取
		TotalRequests:  requestCount,
		TokenUsage: TokenCount{
			Input:  totalInput,
			Output: totalOutput,
			Total:  totalInput + totalOutput,
		},
		Cost:         cost,
		ErrorRate:    errorRate,
		AvgLatencyMs: avgLatency,
		Period:       period,
		UpdatedAt:    now,
	}, nil
}

// GetOverviewStats 获取概览统计
func (a *Aggregator) GetOverviewStats(ctx context.Context, period string) (*OverviewStats, error) {
	if period == "" {
		period = "24h"
	}

	now := time.Now()
	var startTime time.Time

	switch period {
	case "1h":
		startTime = now.Add(-1 * time.Hour)
	case "24h":
		startTime = now.Add(-24 * time.Hour)
	case "7d":
		startTime = now.Add(-7 * 24 * time.Hour)
	case "30d":
		startTime = now.Add(-30 * 24 * time.Hour)
	default:
		startTime = now.Add(-24 * time.Hour)
	}

	// 从 EventBus 获取事件（如果有的话）
	var allEvents []types.AgentEventEnvelope
	if a.eventBus != nil {
		allEvents = a.eventBus.GetTimelineFiltered(func(env types.AgentEventEnvelope) bool {
			// 根据时间戳过滤
			ts := time.UnixMilli(env.Bookmark.Timestamp)
			return ts.After(startTime)
		})
	}

	// 如果没有 EventBus 事件，尝试从 Store 读取 telemetry 数据
	if len(allEvents) == 0 {
		return a.getOverviewStatsFromStore(ctx, period, startTime, now)
	}

	// 统计数据
	var totalInput, totalOutput int64
	var errorCount, requestCount int64
	var totalDuration int64
	agentSet := make(map[string]struct{})

	for _, env := range allEvents {
		switch evt := env.Event.(type) {
		case types.MonitorTokenUsageEvent:
			totalInput += evt.InputTokens
			totalOutput += evt.OutputTokens
			requestCount++

		case types.MonitorStepCompleteEvent:
			totalDuration += evt.DurationMs

		case types.MonitorErrorEvent:
			if evt.Severity == "error" {
				errorCount++
			}

		case types.MonitorStateChangedEvent:
			// 从事件属性中提取 agentID（如果有的话）
			// 这里简化处理，实际可能需要从事件上下文获取
		}
	}

	// 从 store 获取活跃 agent 数量
	agents, err := a.store.ListAgents(ctx)
	if err != nil {
		agents = []string{}
	}
	for _, agentID := range agents {
		agentSet[agentID] = struct{}{}
	}

	// 计算成本
	cost := a.costCalculator.Calculate(totalInput, totalOutput, "")

	// 计算错误率
	var errorRate float64
	if requestCount > 0 {
		errorRate = float64(errorCount) / float64(requestCount)
	}

	// 计算平均延迟
	var avgLatency int64
	if requestCount > 0 {
		avgLatency = totalDuration / requestCount
	}

	return &OverviewStats{
		ActiveAgents:   len(agentSet),
		ActiveSessions: 0, // TODO: 从 session store 获取
		TotalRequests:  requestCount,
		TokenUsage: TokenCount{
			Input:  totalInput,
			Output: totalOutput,
			Total:  totalInput + totalOutput,
		},
		Cost:         cost,
		ErrorRate:    errorRate,
		AvgLatencyMs: avgLatency,
		Period:       period,
		UpdatedAt:    now,
	}, nil
}

// GetTokenUsage 获取 Token 使用统计
func (a *Aggregator) GetTokenUsage(ctx context.Context, opts TokenQueryOpts) (*TokenUsageStats, error) {
	// 检查缓存
	cacheKey := opts.Period
	if opts.AgentID != "" {
		cacheKey += ":" + opts.AgentID
	}

	a.mu.RLock()
	if cached, ok := a.tokenCache[cacheKey]; ok && time.Since(a.lastCacheTime) < a.cacheTTL {
		a.mu.RUnlock()
		return cached, nil
	}
	a.mu.RUnlock()

	now := time.Now()
	startTime, endTime := a.getPeriodRange(opts.Period, opts.StartTime, opts.EndTime)

	// 从 EventBus 获取事件 (添加 nil 检查)
	var allEvents []types.AgentEventEnvelope
	if a.eventBus != nil {
		allEvents = a.eventBus.GetTimelineFiltered(func(env types.AgentEventEnvelope) bool {
			ts := time.UnixMilli(env.Bookmark.Timestamp)
			return ts.After(startTime) && ts.Before(endTime)
		})
	}

	// 聚合数据
	var totalInput, totalOutput int64
	byAgent := make(map[string]TokenCount)
	byModel := make(map[string]TokenCount)
	trendMap := make(map[int64]TokenCount) // 按时间桶聚合

	// 计算时间桶大小
	bucketSize := a.getBucketSize(opts.Period)

	for _, env := range allEvents {
		if evt, ok := env.Event.(types.MonitorTokenUsageEvent); ok {
			totalInput += evt.InputTokens
			totalOutput += evt.OutputTokens

			// 按时间桶聚合
			ts := time.UnixMilli(env.Bookmark.Timestamp)
			bucket := ts.Truncate(bucketSize).Unix()
			tc := trendMap[bucket]
			tc.Input += evt.InputTokens
			tc.Output += evt.OutputTokens
			tc.Total += evt.InputTokens + evt.OutputTokens
			trendMap[bucket] = tc

			// TODO: 从事件中提取 agentID 和 model
			// 当前 MonitorTokenUsageEvent 结构中没有这些字段
			// 后续可以通过 OTel 属性扩展
		}
	}

	// 构建趋势数据
	trend := make([]TokenTrendPoint, 0, len(trendMap))
	for ts, tc := range trendMap {
		trend = append(trend, TokenTrendPoint{
			Timestamp: time.Unix(ts, 0),
			Input:     tc.Input,
			Output:    tc.Output,
		})
	}

	// 按时间排序
	sort.Slice(trend, func(i, j int) bool {
		return trend[i].Timestamp.Before(trend[j].Timestamp)
	})

	// 计算成本
	cost := a.costCalculator.Calculate(totalInput, totalOutput, "")

	result := &TokenUsageStats{
		Period: opts.Period,
		Total: TokenCount{
			Input:  totalInput,
			Output: totalOutput,
			Total:  totalInput + totalOutput,
		},
		ByAgent: byAgent,
		ByModel: byModel,
		Trend:   trend,
		Cost:    cost,
	}

	// 更新缓存
	a.mu.Lock()
	a.tokenCache[cacheKey] = result
	a.lastCacheTime = now
	a.mu.Unlock()

	return result, nil
}

// QueryTraces 查询追踪列表
func (a *Aggregator) QueryTraces(ctx context.Context, opts TraceQueryOpts) (*TraceListResult, error) {
	startTime, endTime := a.getPeriodRange("24h", opts.StartTime, opts.EndTime)

	// 从 EventBus 获取事件 (添加 nil 检查)
	var allEvents []types.AgentEventEnvelope
	if a.eventBus != nil {
		allEvents = a.eventBus.GetTimelineFiltered(func(env types.AgentEventEnvelope) bool {
			ts := time.UnixMilli(env.Bookmark.Timestamp)
			return ts.After(startTime) && ts.Before(endTime)
		})
	}

	// 构建追踪
	traces := a.traceBuilder.BuildFromEvents(allEvents)

	// 过滤
	filtered := make([]*TraceSummary, 0)
	for _, trace := range traces {
		// 状态过滤
		if opts.Status != "" && string(trace.Status) != opts.Status {
			continue
		}

		// AgentID 过滤
		if opts.AgentID != "" && trace.AgentID != opts.AgentID {
			continue
		}

		filtered = append(filtered, trace)
	}

	// 按开始时间倒序排序
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].StartTime.After(filtered[j].StartTime)
	})

	// 分页
	total := int64(len(filtered))
	offset := opts.Offset
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}

	if offset >= len(filtered) {
		return &TraceListResult{
			Traces:  []*TraceSummary{},
			Total:   total,
			HasMore: false,
		}, nil
	}

	end := min(offset+limit, len(filtered))

	return &TraceListResult{
		Traces:  filtered[offset:end],
		Total:   total,
		HasMore: end < len(filtered),
	}, nil
}

// GetTraceDetail 获取追踪详情
func (a *Aggregator) GetTraceDetail(ctx context.Context, traceID string) (*TraceDetail, error) {
	// 检查缓存
	a.mu.RLock()
	if cached, ok := a.traceCache[traceID]; ok {
		a.mu.RUnlock()
		return cached, nil
	}
	a.mu.RUnlock()

	// 从 EventBus 获取所有事件
	var allEvents []types.AgentEventEnvelope
	if a.eventBus != nil {
		allEvents = a.eventBus.GetTimeline()
	}

	// 构建追踪树
	detail := a.traceBuilder.BuildTraceDetail(traceID, allEvents)
	if detail == nil {
		return nil, nil
	}

	// 计算成本
	detail.Cost = a.costCalculator.Calculate(
		detail.TokenUsage.Input,
		detail.TokenUsage.Output,
		"",
	)

	// 更新缓存
	a.mu.Lock()
	a.traceCache[traceID] = detail
	a.mu.Unlock()

	return detail, nil
}

// GetCostBreakdown 获取成本分解
func (a *Aggregator) GetCostBreakdown(ctx context.Context, opts CostQueryOpts) (*CostBreakdown, error) {
	tokenOpts := TokenQueryOpts{
		Period:    opts.Period,
		StartTime: opts.StartTime,
		EndTime:   opts.EndTime,
		AgentID:   opts.AgentID,
	}

	tokenStats, err := a.GetTokenUsage(ctx, tokenOpts)
	if err != nil {
		return nil, err
	}

	// 构建成本趋势
	costTrend := make([]CostTrendPoint, 0, len(tokenStats.Trend))
	for _, tp := range tokenStats.Trend {
		cost := a.costCalculator.Calculate(tp.Input, tp.Output, "")
		costTrend = append(costTrend, CostTrendPoint{
			Timestamp: tp.Timestamp,
			Amount:    cost.Amount,
		})
	}

	// 按 Agent 计算成本
	byAgent := make(map[string]CostAmount)
	for agentID, tc := range tokenStats.ByAgent {
		byAgent[agentID] = a.costCalculator.Calculate(tc.Input, tc.Output, "")
	}

	// 按 Model 计算成本
	byModel := make(map[string]CostAmount)
	for model, tc := range tokenStats.ByModel {
		byModel[model] = a.costCalculator.Calculate(tc.Input, tc.Output, model)
	}

	return &CostBreakdown{
		Period:  opts.Period,
		Total:   tokenStats.Cost,
		ByAgent: byAgent,
		ByModel: byModel,
		Trend:   costTrend,
	}, nil
}

// GetPerformanceStats 获取性能统计
func (a *Aggregator) GetPerformanceStats(ctx context.Context, period string) (*PerformanceStats, error) {
	startTime, endTime := a.getPeriodRange(period, nil, nil)

	// 从 EventBus 获取事件
	var allEvents []types.AgentEventEnvelope
	if a.eventBus != nil {
		allEvents = a.eventBus.GetTimelineFiltered(func(env types.AgentEventEnvelope) bool {
			ts := time.UnixMilli(env.Bookmark.Timestamp)
			return ts.After(startTime) && ts.Before(endTime)
		})
	}

	// 收集延迟数据
	var stepLatencies []int64
	toolLatencies := make(map[string][]int64)
	var errorCount, requestCount int64

	for _, env := range allEvents {
		switch evt := env.Event.(type) {
		case types.MonitorStepCompleteEvent:
			stepLatencies = append(stepLatencies, evt.DurationMs)
			requestCount++

		case types.MonitorToolExecutedEvent:
			toolName := evt.Call.Name
			// Note: ToolCallSnapshot doesn't have Duration field
			// We track tool calls but can't measure latency without duration info
			_ = toolName // Track tool was executed

		case types.MonitorErrorEvent:
			if evt.Severity == "error" {
				errorCount++
			}
		}
	}

	// 计算百分位数
	toolLatencyStats := make(map[string]LatencyPercentiles)
	for toolName, latencies := range toolLatencies {
		toolLatencyStats[toolName] = calculatePercentiles(latencies)
	}

	// 计算错误率
	var errorRate float64
	if requestCount > 0 {
		errorRate = float64(errorCount) / float64(requestCount)
	}

	return &PerformanceStats{
		Period:       period,
		TTFT:         calculatePercentiles(stepLatencies), // 简化处理
		TPOT:         LatencyPercentiles{},                // 需要更细粒度的数据
		ToolLatency:  toolLatencyStats,
		AvgLoopCount: 0, // 需要从 agent 执行数据计算
		RequestCount: requestCount,
		ErrorCount:   errorCount,
		ErrorRate:    errorRate,
	}, nil
}

// GetInsights 获取改进建议
func (a *Aggregator) GetInsights(ctx context.Context) ([]Insight, error) {
	var insights []Insight

	// 获取性能统计
	perfStats, err := a.GetPerformanceStats(ctx, "24h")
	if err == nil {
		// 规则 1: 检查高延迟工具
		for toolName, latency := range perfStats.ToolLatency {
			if latency.P95 > 2000 { // > 2s
				insights = append(insights, Insight{
					ID:          "high_tool_latency_" + toolName,
					Type:        InsightTypePerformance,
					Severity:    "warning",
					Title:       "工具延迟过高: " + toolName,
					Description: "工具 " + toolName + " 的 P95 延迟超过 2 秒",
					Suggestion:  "考虑添加缓存、优化工具实现或设置超时",
					Data: map[string]any{
						"tool_name": toolName,
						"p95_ms":    latency.P95,
						"p99_ms":    latency.P99,
					},
					CreatedAt: time.Now(),
				})
			}
		}

		// 规则 2: 检查高错误率
		if perfStats.ErrorRate > 0.05 { // > 5%
			insights = append(insights, Insight{
				ID:          "high_error_rate",
				Type:        InsightTypeReliability,
				Severity:    "critical",
				Title:       "错误率过高",
				Description: "过去 24 小时的错误率超过 5%",
				Suggestion:  "检查错误日志，识别常见错误模式并修复",
				Data: map[string]any{
					"error_rate":    perfStats.ErrorRate,
					"error_count":   perfStats.ErrorCount,
					"request_count": perfStats.RequestCount,
				},
				CreatedAt: time.Now(),
			})
		}
	}

	// 规则 3: 检查 Token 消耗
	tokenStats, err := a.GetTokenUsage(ctx, TokenQueryOpts{Period: "24h"})
	if err == nil {
		if tokenStats.Cost.Amount > 10 { // > $10/天
			insights = append(insights, Insight{
				ID:          "high_daily_cost",
				Type:        InsightTypeCost,
				Severity:    "warning",
				Title:       "每日成本较高",
				Description: "过去 24 小时的 Token 成本超过 $10",
				Suggestion:  "考虑使用上下文压缩中间件、优化 prompt 或切换到更便宜的模型",
				Data: map[string]any{
					"daily_cost":    tokenStats.Cost.Amount,
					"total_tokens":  tokenStats.Total.Total,
					"input_tokens":  tokenStats.Total.Input,
					"output_tokens": tokenStats.Total.Output,
				},
				CreatedAt: time.Now(),
			})
		}
	}

	return insights, nil
}

// getPeriodRange 计算时间范围
func (a *Aggregator) getPeriodRange(period string, start, end *time.Time) (time.Time, time.Time) {
	now := time.Now()

	if start != nil && end != nil {
		return *start, *end
	}

	var startTime time.Time
	switch period {
	case "hour":
		startTime = now.Add(-1 * time.Hour)
	case "day", "24h":
		startTime = now.Add(-24 * time.Hour)
	case "week", "7d":
		startTime = now.Add(-7 * 24 * time.Hour)
	case "month", "30d":
		startTime = now.Add(-30 * 24 * time.Hour)
	default:
		startTime = now.Add(-24 * time.Hour)
	}

	return startTime, now
}

// getBucketSize 获取时间桶大小
func (a *Aggregator) getBucketSize(period string) time.Duration {
	switch period {
	case "hour":
		return 5 * time.Minute
	case "day", "24h":
		return 1 * time.Hour
	case "week", "7d":
		return 6 * time.Hour
	case "month", "30d":
		return 1 * 24 * time.Hour
	default:
		return 1 * time.Hour
	}
}

// calculatePercentiles 计算百分位数
func calculatePercentiles(values []int64) LatencyPercentiles {
	if len(values) == 0 {
		return LatencyPercentiles{}
	}

	// 排序
	sorted := make([]int64, len(values))
	copy(sorted, values)
	slices.Sort(sorted)

	n := len(sorted)

	// 计算总和用于平均值
	var sum int64
	for _, v := range sorted {
		sum += v
	}

	return LatencyPercentiles{
		P50: sorted[n*50/100],
		P95: sorted[n*95/100],
		P99: sorted[n*99/100],
		Avg: sum / int64(n),
		Max: sorted[n-1],
	}
}

// getOverviewStatsFromStore 从 Store 获取概览统计
func (a *Aggregator) getOverviewStatsFromStore(ctx context.Context, period string, startTime, now time.Time) (*OverviewStats, error) {
	var totalInput, totalOutput int64
	var errorCount, requestCount int64

	// 从 metrics collection 读取数据
	records, err := a.store.List(ctx, "metrics")
	if err == nil {
		for _, record := range records {
			var metric map[string]any
			if err := store.DecodeValue(record, &metric); err != nil {
				continue
			}

			// 检查时间范围
			if ts, ok := metric["timestamp"].(string); ok {
				t, err := time.Parse(time.RFC3339, ts)
				if err != nil || t.Before(startTime) {
					continue
				}
			}

			// 提取 Token 数据
			if name, ok := metric["name"].(string); ok {
				switch name {
				case "token_usage":
					if val, ok := metric["value"].(float64); ok {
						if tags, ok := metric["tags"].(map[string]any); ok {
							if tokenType, ok := tags["type"].(string); ok {
								switch tokenType {
								case "input":
									totalInput += int64(val)
								case "output":
									totalOutput += int64(val)
								}
							}
						}
					}
					requestCount++
				case "error":
					errorCount++
				}
			}
		}
	}

	// 从 agents collection 获取活跃 agent 数量
	agents, err := a.store.ListAgents(ctx)
	if err != nil {
		agents = []string{}
	}

	// 计算成本
	cost := a.costCalculator.Calculate(totalInput, totalOutput, "")

	// 计算错误率
	var errorRate float64
	if requestCount > 0 {
		errorRate = float64(errorCount) / float64(requestCount)
	}

	return &OverviewStats{
		ActiveAgents:   len(agents),
		ActiveSessions: 0,
		TotalRequests:  requestCount,
		TokenUsage: TokenCount{
			Input:  totalInput,
			Output: totalOutput,
			Total:  totalInput + totalOutput,
		},
		Cost:         cost,
		ErrorRate:    errorRate,
		AvgLatencyMs: 0,
		Period:       period,
		UpdatedAt:    now,
	}, nil
}
