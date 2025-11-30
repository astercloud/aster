package handlers

import (
	"net/http"
	"time"

	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MetricRecord 指标记录
type MetricRecord struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Type      string            `json:"type"` // counter, gauge, histogram
	Value     float64           `json:"value"`
	Tags      map[string]string `json:"tags,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]any    `json:"metadata,omitempty"`
}

// TraceRecord 追踪记录
type TraceRecord struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	SpanID     string         `json:"span_id"`
	ParentID   string         `json:"parent_id,omitempty"`
	StartTime  time.Time      `json:"start_time"`
	EndTime    *time.Time     `json:"end_time,omitempty"`
	Duration   int64          `json:"duration,omitempty"` // microseconds
	Status     string         `json:"status"`             // ok, error
	Attributes map[string]any `json:"attributes,omitempty"`
}

// LogRecord 日志记录
type LogRecord struct {
	ID        string         `json:"id"`
	Level     string         `json:"level"` // debug, info, warn, error
	Message   string         `json:"message"`
	Source    string         `json:"source,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// TelemetryHandler handles telemetry-related requests
type TelemetryHandler struct {
	store *store.Store
}

// NewTelemetryHandler creates a new TelemetryHandler
func NewTelemetryHandler(st store.Store) *TelemetryHandler {
	return &TelemetryHandler{store: &st}
}

// RecordMetric records a metric
func (h *TelemetryHandler) RecordMetric(c *gin.Context) {
	var req struct {
		Name     string            `json:"name" binding:"required"`
		Type     string            `json:"type" binding:"required"`
		Value    float64           `json:"value" binding:"required"`
		Tags     map[string]string `json:"tags"`
		Metadata map[string]any    `json:"metadata"`
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

	ctx := c.Request.Context()
	metric := &MetricRecord{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Type:      req.Type,
		Value:     req.Value,
		Tags:      req.Tags,
		Timestamp: time.Now(),
		Metadata:  req.Metadata,
	}

	if err := (*h.store).Set(ctx, "metrics", metric.ID, metric); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "telemetry.metric.recorded", map[string]any{
		"name":  req.Name,
		"value": req.Value,
	})

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    metric,
	})
}

// ListMetrics lists all metrics
func (h *TelemetryHandler) ListMetrics(c *gin.Context) {
	ctx := c.Request.Context()

	records, err := (*h.store).List(ctx, "metrics")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	metrics := make([]*MetricRecord, 0)
	for _, record := range records {
		var m MetricRecord
		if err := store.DecodeValue(record, &m); err != nil {
			continue
		}
		metrics = append(metrics, &m)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    metrics,
	})
}

// RecordTrace records a trace
func (h *TelemetryHandler) RecordTrace(c *gin.Context) {
	var req struct {
		Name       string         `json:"name" binding:"required"`
		SpanID     string         `json:"span_id" binding:"required"`
		ParentID   string         `json:"parent_id"`
		StartTime  time.Time      `json:"start_time"`
		EndTime    *time.Time     `json:"end_time"`
		Duration   int64          `json:"duration"`
		Status     string         `json:"status"`
		Attributes map[string]any `json:"attributes"`
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

	ctx := c.Request.Context()
	trace := &TraceRecord{
		ID:         uuid.New().String(),
		Name:       req.Name,
		SpanID:     req.SpanID,
		ParentID:   req.ParentID,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		Duration:   req.Duration,
		Status:     req.Status,
		Attributes: req.Attributes,
	}

	if err := (*h.store).Set(ctx, "traces", trace.ID, trace); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "telemetry.trace.recorded", map[string]any{
		"trace_id": trace.ID,
		"name":     req.Name,
	})

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    trace,
	})
}

// QueryTraces queries traces
func (h *TelemetryHandler) QueryTraces(c *gin.Context) {
	ctx := c.Request.Context()

	// Get query parameters
	var req struct {
		StartTime *time.Time `json:"start_time"`
		EndTime   *time.Time `json:"end_time"`
		Name      string     `json:"name"`
		Status    string     `json:"status"`
		Limit     int        `json:"limit"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// If no body, try query params
		req.Name = c.Query("name")
		req.Status = c.Query("status")
	}

	records, err := (*h.store).List(ctx, "traces")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	traces := make([]*TraceRecord, 0)
	for _, record := range records {
		var t TraceRecord
		if err := store.DecodeValue(record, &t); err != nil {
			continue
		}

		// Filter
		if req.Name != "" && t.Name != req.Name {
			continue
		}
		if req.Status != "" && t.Status != req.Status {
			continue
		}

		traces = append(traces, &t)

		if req.Limit > 0 && len(traces) >= req.Limit {
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"traces": traces,
		},
	})
}

// RecordLog records a log
func (h *TelemetryHandler) RecordLog(c *gin.Context) {
	var req struct {
		Level    string         `json:"level" binding:"required"`
		Message  string         `json:"message" binding:"required"`
		Source   string         `json:"source"`
		Metadata map[string]any `json:"metadata"`
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

	ctx := c.Request.Context()
	logRecord := &LogRecord{
		ID:        uuid.New().String(),
		Level:     req.Level,
		Message:   req.Message,
		Source:    req.Source,
		Timestamp: time.Now(),
		Metadata:  req.Metadata,
	}

	if err := (*h.store).Set(ctx, "logs", logRecord.ID, logRecord); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "telemetry.log.recorded", map[string]any{
		"level":   req.Level,
		"message": req.Message,
	})

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    logRecord,
	})
}

// QueryLogs queries logs
func (h *TelemetryHandler) QueryLogs(c *gin.Context) {
	ctx := c.Request.Context()

	level := c.Query("level")
	source := c.Query("source")

	records, err := (*h.store).List(ctx, "logs")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	logs := make([]*LogRecord, 0)
	for _, record := range records {
		var l LogRecord
		if err := store.DecodeValue(record, &l); err != nil {
			continue
		}

		// Filter
		if level != "" && l.Level != level {
			continue
		}
		if source != "" && l.Source != source {
			continue
		}

		logs = append(logs, &l)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"logs": logs,
		},
	})
}
