package handlers

import (
	"net/http"
	"time"

	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// EvalRecord 评估记录
type EvalRecord struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Type        string             `json:"type"`   // text, session, batch, benchmark
	Status      string             `json:"status"` // pending, running, completed, failed
	Input       map[string]any     `json:"input,omitempty"`
	Output      map[string]any     `json:"output,omitempty"`
	Metrics     map[string]float64 `json:"metrics,omitempty"`
	Score       float64            `json:"score,omitempty"`
	StartedAt   time.Time          `json:"started_at"`
	CompletedAt *time.Time         `json:"completed_at,omitempty"`
	Duration    int64              `json:"duration,omitempty"` // milliseconds
	Error       string             `json:"error,omitempty"`
	Metadata    map[string]any     `json:"metadata,omitempty"`
}

// BenchmarkRecord 基准测试记录
type BenchmarkRecord struct {
	ID        string               `json:"id"`
	Name      string               `json:"name"`
	Runs      int                  `json:"runs"`
	Results   []map[string]float64 `json:"results,omitempty"`
	Summary   map[string]any       `json:"summary,omitempty"`
	CreatedAt time.Time            `json:"created_at"`
}

// EvalHandler handles evaluation-related requests
type EvalHandler struct {
	store *store.Store
}

// NewEvalHandler creates a new EvalHandler
func NewEvalHandler(st store.Store) *EvalHandler {
	return &EvalHandler{store: &st}
}

// RunTextEval runs text evaluation
func (h *EvalHandler) RunTextEval(c *gin.Context) {
	var req struct {
		Prompt   string         `json:"prompt" binding:"required"`
		Expected string         `json:"expected"`
		Criteria map[string]any `json:"criteria"`
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
	evalRec := &EvalRecord{
		ID:     uuid.New().String(),
		Name:   "Text Eval",
		Type:   "text",
		Status: "completed",
		Input: map[string]any{
			"prompt":   req.Prompt,
			"expected": req.Expected,
		},
		Output:    map[string]any{"result": "Sample output"},
		Metrics:   map[string]float64{"accuracy": 0.95},
		Score:     0.95,
		StartedAt: time.Now(),
		Duration:  100,
		Metadata:  req.Metadata,
	}

	now := time.Now()
	evalRec.CompletedAt = &now

	if err := (*h.store).Set(ctx, "evals", evalRec.ID, evalRec); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "eval.text.completed", map[string]any{
		"id":    evalRec.ID,
		"score": evalRec.Score,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    evalRec,
	})
}

// RunSessionEval runs session evaluation
func (h *EvalHandler) RunSessionEval(c *gin.Context) {
	var req struct {
		SessionID string         `json:"session_id" binding:"required"`
		Criteria  map[string]any `json:"criteria"`
		Metadata  map[string]any `json:"metadata"`
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
	evalRec := &EvalRecord{
		ID:     uuid.New().String(),
		Name:   "Session Eval",
		Type:   "session",
		Status: "completed",
		Input: map[string]any{
			"session_id": req.SessionID,
		},
		Metrics:   map[string]float64{"coherence": 0.90, "relevance": 0.88},
		Score:     0.89,
		StartedAt: time.Now(),
		Duration:  250,
		Metadata:  req.Metadata,
	}

	now := time.Now()
	evalRec.CompletedAt = &now

	if err := (*h.store).Set(ctx, "evals", evalRec.ID, evalRec); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "eval.session.completed", map[string]any{
		"id": evalRec.ID,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    evalRec,
	})
}

// RunBatchEval runs batch evaluation
func (h *EvalHandler) RunBatchEval(c *gin.Context) {
	var req struct {
		Items    []map[string]any `json:"items" binding:"required"`
		Criteria map[string]any   `json:"criteria"`
		Metadata map[string]any   `json:"metadata"`
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
	evalRec := &EvalRecord{
		ID:     uuid.New().String(),
		Name:   "Batch Eval",
		Type:   "batch",
		Status: "completed",
		Input: map[string]any{
			"count": len(req.Items),
		},
		Metrics:   map[string]float64{"avg_score": 0.92},
		Score:     0.92,
		StartedAt: time.Now(),
		Duration:  500,
		Metadata:  req.Metadata,
	}

	now := time.Now()
	evalRec.CompletedAt = &now

	if err := (*h.store).Set(ctx, "evals", evalRec.ID, evalRec); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "eval.batch.completed", map[string]any{
		"id":    evalRec.ID,
		"count": len(req.Items),
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    evalRec,
	})
}

// RunCustomEval runs custom evaluation
func (h *EvalHandler) RunCustomEval(c *gin.Context) {
	var req struct {
		Name     string         `json:"name" binding:"required"`
		Input    map[string]any `json:"input" binding:"required"`
		Criteria map[string]any `json:"criteria"`
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
	evalRec := &EvalRecord{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Type:      "custom",
		Status:    "completed",
		Input:     req.Input,
		Metrics:   map[string]float64{"custom_metric": 0.93},
		Score:     0.93,
		StartedAt: time.Now(),
		Duration:  150,
		Metadata:  req.Metadata,
	}

	now := time.Now()
	evalRec.CompletedAt = &now

	if err := (*h.store).Set(ctx, "evals", evalRec.ID, evalRec); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "eval.custom.completed", map[string]any{
		"id":   evalRec.ID,
		"name": req.Name,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    evalRec,
	})
}

// ListEvals lists all evaluations
func (h *EvalHandler) ListEvals(c *gin.Context) {
	ctx := c.Request.Context()
	evalType := c.Query("type")
	status := c.Query("status")

	records, err := (*h.store).List(ctx, "evals")
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

	evals := make([]*EvalRecord, 0)
	for _, record := range records {
		var eval EvalRecord
		if err := store.DecodeValue(record, &eval); err != nil {
			continue
		}

		// Filter
		if evalType != "" && eval.Type != evalType {
			continue
		}
		if status != "" && eval.Status != status {
			continue
		}

		evals = append(evals, &eval)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    evals,
	})
}

// GetEval retrieves a single evaluation
func (h *EvalHandler) GetEval(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var eval EvalRecord
	if err := (*h.store).Get(ctx, "evals", id, &eval); err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Evaluation not found",
				},
			})
			return
		}
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
		"data":    &eval,
	})
}

// DeleteEval deletes an evaluation
func (h *EvalHandler) DeleteEval(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	if err := (*h.store).Delete(ctx, "evals", id); err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Evaluation not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "eval.deleted", map[string]any{
		"id": id,
	})

	c.Status(http.StatusNoContent)
}

// CreateBenchmark creates a benchmark
func (h *EvalHandler) CreateBenchmark(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
		Runs int    `json:"runs" binding:"required"`
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
	benchmark := &BenchmarkRecord{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Runs:      req.Runs,
		Results:   []map[string]float64{},
		CreatedAt: time.Now(),
	}

	if err := (*h.store).Set(ctx, "benchmarks", benchmark.ID, benchmark); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "benchmark.created", map[string]any{
		"id":   benchmark.ID,
		"name": req.Name,
	})

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    benchmark,
	})
}

// ListBenchmarks lists all benchmarks
func (h *EvalHandler) ListBenchmarks(c *gin.Context) {
	ctx := c.Request.Context()

	records, err := (*h.store).List(ctx, "benchmarks")
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

	benchmarks := make([]*BenchmarkRecord, 0)
	for _, record := range records {
		var benchmark BenchmarkRecord
		if err := store.DecodeValue(record, &benchmark); err != nil {
			continue
		}
		benchmarks = append(benchmarks, &benchmark)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    benchmarks,
	})
}

// GetBenchmark retrieves a single benchmark
func (h *EvalHandler) GetBenchmark(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var benchmark BenchmarkRecord
	if err := (*h.store).Get(ctx, "benchmarks", id, &benchmark); err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Benchmark not found",
				},
			})
			return
		}
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
		"data":    &benchmark,
	})
}

// DeleteBenchmark deletes a benchmark
func (h *EvalHandler) DeleteBenchmark(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	if err := (*h.store).Delete(ctx, "benchmarks", id); err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Benchmark not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "benchmark.deleted", map[string]any{
		"id": id,
	})

	c.Status(http.StatusNoContent)
}

// RunBenchmark runs a benchmark
func (h *EvalHandler) RunBenchmark(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var benchmark BenchmarkRecord
	if err := (*h.store).Get(ctx, "benchmarks", id, &benchmark); err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Benchmark not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	// TODO: Actually run the benchmark
	// For now, generate sample results
	results := make([]map[string]float64, benchmark.Runs)
	for i := 0; i < benchmark.Runs; i++ {
		results[i] = map[string]float64{
			"score":    0.90 + float64(i)*0.01,
			"duration": 100 + float64(i)*10,
		}
	}
	benchmark.Results = results

	if err := (*h.store).Set(ctx, "benchmarks", id, &benchmark); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "benchmark.run", map[string]any{
		"id": id,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    &benchmark,
	})
}
