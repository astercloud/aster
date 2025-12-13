package handlers

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/astercloud/aster/pkg/dashboard"
	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/types"
	"github.com/gin-gonic/gin"
)

// DashboardHandler handles dashboard-related requests
type DashboardHandler struct {
	aggregator *dashboard.Aggregator
	registry   *RuntimeAgentRegistry
	store      *store.Store
}

// NewDashboardHandler creates a new DashboardHandler
func NewDashboardHandler(st store.Store) *DashboardHandler {
	return &DashboardHandler{
		aggregator: dashboard.NewAggregator(st),
		registry:   nil,
		store:      &st,
	}
}

// NewDashboardHandlerWithRegistry creates a new DashboardHandler with RuntimeAgentRegistry
func NewDashboardHandlerWithRegistry(registry *RuntimeAgentRegistry, st store.Store) *DashboardHandler {
	return &DashboardHandler{
		aggregator: dashboard.NewAggregator(st),
		registry:   registry,
		store:      &st,
	}
}

// GetOverview returns overview statistics
func (h *DashboardHandler) GetOverview(c *gin.Context) {
	ctx := c.Request.Context()
	period := c.DefaultQuery("period", "24h")

	var stats *dashboard.OverviewStats
	var err error

	// 如果有 registry，从所有 Agent 的 EventBus 聚合数据
	if h.registry != nil {
		stats, err = h.aggregator.GetOverviewStatsFromEventBuses(ctx, period, h.registry)
	} else {
		// 否则使用默认方法（从 Store 读取）
		stats, err = h.aggregator.GetOverviewStats(ctx, period)
	}

	if err != nil {
		logging.Error(ctx, "dashboard.overview.error", map[string]any{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// ListTraces returns a list of traces
func (h *DashboardHandler) ListTraces(c *gin.Context) {
	ctx := c.Request.Context()

	opts := dashboard.TraceQueryOpts{
		Status:  c.Query("status"),
		AgentID: c.Query("agent_id"),
		Limit:   50,
		Offset:  0,
	}

	// 解析时间参数
	if startStr := c.Query("start"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			opts.StartTime = &t
		}
	}

	if endStr := c.Query("end"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			opts.EndTime = &t
		}
	}

	// 解析分页参数
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			opts.Limit = limit
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			opts.Offset = offset
		}
	}

	result, err := h.aggregator.QueryTraces(ctx, opts)
	if err != nil {
		logging.Error(ctx, "dashboard.traces.list.error", map[string]any{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetTrace returns a specific trace detail
func (h *DashboardHandler) GetTrace(c *gin.Context) {
	ctx := c.Request.Context()
	traceID := c.Param("id")

	detail, err := h.aggregator.GetTraceDetail(ctx, traceID)
	if err != nil {
		logging.Error(ctx, "dashboard.trace.get.error", map[string]any{
			"trace_id": traceID,
			"error":    err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	if detail == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_found",
				"message": "Trace not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    detail,
	})
}

// GetTokenUsage returns token usage statistics
func (h *DashboardHandler) GetTokenUsage(c *gin.Context) {
	ctx := c.Request.Context()

	opts := dashboard.TokenQueryOpts{
		Period:  c.DefaultQuery("period", "24h"),
		AgentID: c.Query("agent_id"),
		Model:   c.Query("model"),
	}

	// 解析时间参数
	if startStr := c.Query("start"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			opts.StartTime = &t
		}
	}

	if endStr := c.Query("end"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			opts.EndTime = &t
		}
	}

	stats, err := h.aggregator.GetTokenUsage(ctx, opts)
	if err != nil {
		logging.Error(ctx, "dashboard.tokens.error", map[string]any{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetCosts returns cost breakdown
func (h *DashboardHandler) GetCosts(c *gin.Context) {
	ctx := c.Request.Context()

	opts := dashboard.CostQueryOpts{
		Period:  c.DefaultQuery("period", "24h"),
		AgentID: c.Query("agent_id"),
	}

	// 解析时间参数
	if startStr := c.Query("start"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			opts.StartTime = &t
		}
	}

	if endStr := c.Query("end"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			opts.EndTime = &t
		}
	}

	breakdown, err := h.aggregator.GetCostBreakdown(ctx, opts)
	if err != nil {
		logging.Error(ctx, "dashboard.costs.error", map[string]any{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    breakdown,
	})
}

// GetPerformance returns performance statistics
func (h *DashboardHandler) GetPerformance(c *gin.Context) {
	ctx := c.Request.Context()
	period := c.DefaultQuery("period", "24h")

	stats, err := h.aggregator.GetPerformanceStats(ctx, period)
	if err != nil {
		logging.Error(ctx, "dashboard.performance.error", map[string]any{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetInsights returns improvement insights
func (h *DashboardHandler) GetInsights(c *gin.Context) {
	ctx := c.Request.Context()

	insights, err := h.aggregator.GetInsights(ctx)
	if err != nil {
		logging.Error(ctx, "dashboard.insights.error", map[string]any{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    insights,
	})
}

// GetRecentEvents returns recent events from the timeline
func (h *DashboardHandler) GetRecentEvents(c *gin.Context) {
	ctx := c.Request.Context()

	// 检查 Registry 是否可用
	if h.registry == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"events": []gin.H{},
				"cursor": int64(0),
				"message": "Registry not available, real-time events disabled",
			},
		})
		return
	}

	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	// 从所有 Agent 的 EventBus 聚合事件
	var allEvents []types.AgentEventEnvelope
	for _, eb := range h.registry.GetEventBuses() {
		if eb != nil {
			evts := eb.GetTimelineRange(0, limit)
			allEvents = append(allEvents, evts...)
		}
	}

	// 按时间戳排序（最新的在前）
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].Bookmark.Timestamp > allEvents[j].Bookmark.Timestamp
	})

	// 限制数量
	if len(allEvents) > limit {
		allEvents = allEvents[:limit]
	}

	// 转换为响应格式
	result := make([]gin.H, 0, len(allEvents))
	for _, env := range allEvents {
		result = append(result, gin.H{
			"cursor":    env.Cursor,
			"timestamp": time.UnixMilli(env.Bookmark.Timestamp),
			"event":     env.Event,
		})
	}

	logging.Info(ctx, "dashboard.events.list", map[string]any{
		"count": len(result),
	})

	// 获取最新的 cursor（如果有事件的话）
	maxCursor := int64(0)
	if len(allEvents) > 0 {
		maxCursor = allEvents[0].Cursor
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"events": result,
			"cursor": maxCursor,
		},
	})
}

// GetEventsSince returns events since a cursor
func (h *DashboardHandler) GetEventsSince(c *gin.Context) {
	ctx := c.Request.Context()

	// 检查 Registry 是否可用
	if h.registry == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"events":      []gin.H{},
				"next_cursor": int64(0),
				"message":     "Registry not available, real-time events disabled",
			},
		})
		return
	}

	cursorStr := c.Param("cursor")
	cursor, err := strconv.ParseInt(cursorStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "bad_request",
				"message": "Invalid cursor",
			},
		})
		return
	}

	// 从所有 Agent 的 EventBus 聚合事件
	var allEvents []types.AgentEventEnvelope
	maxCursor := cursor
	for _, eb := range h.registry.GetEventBuses() {
		if eb != nil {
			evts := eb.GetTimelineSince(cursor)
			allEvents = append(allEvents, evts...)
			// 更新最大 cursor
			if ebCursor := eb.GetCursor(); ebCursor > maxCursor {
				maxCursor = ebCursor
			}
		}
	}

	// 按时间戳排序（最新的在前）
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].Bookmark.Timestamp > allEvents[j].Bookmark.Timestamp
	})

	// 转换为响应格式
	result := make([]gin.H, 0, len(allEvents))
	for _, env := range allEvents {
		result = append(result, gin.H{
			"cursor":    env.Cursor,
			"timestamp": time.UnixMilli(env.Bookmark.Timestamp),
			"event":     env.Event,
		})
	}

	logging.Info(ctx, "dashboard.events.since", map[string]any{
		"cursor": cursor,
		"count":  len(result),
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"events":      result,
			"next_cursor": maxCursor,
		},
	})
}

// GetPricing returns model pricing information
func (h *DashboardHandler) GetPricing(c *gin.Context) {
	pricing := dashboard.DefaultModelPricing

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    pricing,
	})
}

// UpdatePricing updates model pricing (for custom pricing)
func (h *DashboardHandler) UpdatePricing(c *gin.Context) {
	var req struct {
		Model           string  `json:"model" binding:"required"`
		InputPricePerM  float64 `json:"input_price_per_m" binding:"required"`
		OutputPricePerM float64 `json:"output_price_per_m" binding:"required"`
		Currency        string  `json:"currency"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "bad_request",
				"message": err.Error(),
			},
		})
		return
	}

	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}

	// 更新定价
	dashboard.DefaultModelPricing[req.Model] = dashboard.ModelPricing{
		Model:           req.Model,
		InputPricePerM:  req.InputPricePerM,
		OutputPricePerM: req.OutputPricePerM,
		Currency:        currency,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"model":   req.Model,
			"pricing": dashboard.DefaultModelPricing[req.Model],
		},
	})
}

// SessionSummary represents a session summary for dashboard
type SessionSummary struct {
	ID          string         `json:"id"`
	AgentID     string         `json:"agent_id,omitempty"`
	AgentName   string         `json:"agent_name,omitempty"`
	Status      string         `json:"status"`
	MessageCount int           `json:"message_count"`
	TokenUsage  TokenCount     `json:"token_usage"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// TokenCount represents token counts
type TokenCount struct {
	Input  int64 `json:"input"`
	Output int64 `json:"output"`
	Total  int64 `json:"total"`
}

// SessionListResult represents a list of sessions
type SessionListResult struct {
	Sessions []SessionSummary `json:"sessions"`
	Total    int              `json:"total"`
	HasMore  bool             `json:"has_more"`
}

// ListSessions returns a list of sessions for the dashboard
func (h *DashboardHandler) ListSessions(c *gin.Context) {
	ctx := c.Request.Context()

	// 解析分页参数
	limit := 50
	offset := 0

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// 从 store 获取 sessions
	sessions := []SessionSummary{}

	// 尝试从 sessions bucket 获取数据
	// 注意：List 返回的是完整的 JSON 对象列表，不是 key 列表
	items, err := (*h.store).List(ctx, "sessions")
	if err != nil {
		// bucket 不存在或为空，返回空列表
		logging.Info(ctx, "dashboard.sessions.list.empty", map[string]any{
			"error": err.Error(),
		})
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": SessionListResult{
				Sessions: sessions,
				Total:    0,
				HasMore:  false,
			},
		})
		return
	}

	// 解析每个 session 记录
	for i, item := range items {
		if i < offset {
			continue
		}
		if len(sessions) >= limit {
			break
		}

		// 使用 DecodeValue 将 any 转换为 SessionRecord
		var record SessionRecord
		if err := store.DecodeValue(item, &record); err != nil {
			logging.Warn(ctx, "dashboard.sessions.decode.error", map[string]any{
				"error": err.Error(),
			})
			continue
		}

		// Token usage from session metadata
		var tokenUsage TokenCount
		if record.Metadata != nil {
			if usage, ok := record.Metadata["token_usage"].(map[string]any); ok {
				if input, ok := usage["input"].(float64); ok {
					tokenUsage.Input = int64(input)
				}
				if output, ok := usage["output"].(float64); ok {
					tokenUsage.Output = int64(output)
				}
			}
		}
		tokenUsage.Total = tokenUsage.Input + tokenUsage.Output

		sessions = append(sessions, SessionSummary{
			ID:           record.ID,
			AgentID:      record.AgentID,
			Status:       record.Status,
			MessageCount: len(record.Messages),
			TokenUsage:   tokenUsage,
			CreatedAt:    record.CreatedAt,
			UpdatedAt:    record.UpdatedAt,
			Metadata:     record.Metadata,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": SessionListResult{
			Sessions: sessions,
			Total:    len(items),
			HasMore:  len(items) > offset+limit,
		},
	})
}

// SessionDetail represents detailed session info
type SessionDetail struct {
	SessionSummary
	Messages []MessageSummary `json:"messages"`
}

// MessageSummary represents a message in the session
type MessageSummary struct {
	ID        string         `json:"id,omitempty"`
	Role      string         `json:"role"`
	Content   string         `json:"content"`
	Timestamp time.Time      `json:"timestamp"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// GetSession returns detailed session info
func (h *DashboardHandler) GetSession(c *gin.Context) {
	ctx := c.Request.Context()
	sessionID := c.Param("id")

	var record SessionRecord
	if err := (*h.store).Get(ctx, "sessions", sessionID, &record); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_found",
				"message": "Session not found",
			},
		})
		return
	}

	// 转换消息
	messages := make([]MessageSummary, 0, len(record.Messages))

	for i, msg := range record.Messages {
		// 获取消息内容
		content := msg.Content
		if content == "" && len(msg.ContentBlocks) > 0 {
			// 从 ContentBlocks 提取文本
			for _, block := range msg.ContentBlocks {
				if textBlock, ok := block.(*types.TextBlock); ok {
					content += textBlock.Text
				}
			}
		}

		// 使用消息索引作为时间戳的偏移
		timestamp := record.CreatedAt.Add(time.Duration(i) * time.Second)

		messages = append(messages, MessageSummary{
			Role:      string(msg.Role),
			Content:   content,
			Timestamp: timestamp,
		})
	}

	// Token usage from session metadata
	var tokenUsage TokenCount
	if record.Metadata != nil {
		if usage, ok := record.Metadata["token_usage"].(map[string]any); ok {
			if input, ok := usage["input"].(float64); ok {
				tokenUsage.Input = int64(input)
			}
			if output, ok := usage["output"].(float64); ok {
				tokenUsage.Output = int64(output)
			}
		}
	}
	tokenUsage.Total = tokenUsage.Input + tokenUsage.Output

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": SessionDetail{
			SessionSummary: SessionSummary{
				ID:           record.ID,
				AgentID:      record.AgentID,
				Status:       record.Status,
				MessageCount: len(record.Messages),
				TokenUsage:   tokenUsage,
				CreatedAt:    record.CreatedAt,
				UpdatedAt:    record.UpdatedAt,
				Metadata:     record.Metadata,
			},
			Messages: messages,
		},
	})
}
