package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// WorkflowStep Workflow 步骤
type WorkflowStep struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Type      string         `json:"type"` // agent, tool, condition, loop
	Config    map[string]any `json:"config,omitempty"`
	DependsOn []string       `json:"depends_on,omitempty"`
	Timeout   int            `json:"timeout,omitempty"` // seconds
}

// WorkflowTrigger Workflow 触发器
type WorkflowTrigger struct {
	Type   string         `json:"type"` // manual, scheduled, event
	Config map[string]any `json:"config,omitempty"`
}

// WorkflowExecution Workflow 执行记录
type WorkflowExecution struct {
	ID          string         `json:"id"`
	WorkflowID  string         `json:"workflow_id"`
	Status      string         `json:"status"` // pending, running, completed, failed, canceled
	StartedAt   time.Time      `json:"started_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	Result      map[string]any `json:"result,omitempty"`
	Error       string         `json:"error,omitempty"`
	Logs        []ExecutionLog `json:"logs,omitempty"`
	Context     map[string]any `json:"context,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// ExecutionLog 执行日志
type ExecutionLog struct {
	Timestamp time.Time      `json:"timestamp"`
	Level     string         `json:"level"` // info, warn, error
	StepID    string         `json:"step_id,omitempty"`
	Message   string         `json:"message"`
	Data      map[string]any `json:"data,omitempty"`
}

// WorkflowHandler handles workflow-related requests
type WorkflowHandler struct {
	store *store.Store
}

// NewWorkflowHandler creates a new WorkflowHandler
func NewWorkflowHandler(st store.Store) *WorkflowHandler {
	return &WorkflowHandler{store: &st}
}

// Create creates a new workflow
func (h *WorkflowHandler) Create(c *gin.Context) {
	var req struct {
		Name        string            `json:"name" binding:"required"`
		Description string            `json:"description"`
		Version     string            `json:"version"`
		Steps       []WorkflowStep    `json:"steps" binding:"required"`
		Triggers    []WorkflowTrigger `json:"triggers"`
		Metadata    map[string]any    `json:"metadata"`
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
	now := time.Now()

	version := req.Version
	if version == "" {
		version = "1.0.0"
	}

	// Convert steps to []any
	steps := make([]any, len(req.Steps))
	for i, step := range req.Steps {
		steps[i] = step
	}

	// Convert triggers to []any
	triggers := make([]any, len(req.Triggers))
	for i, trigger := range req.Triggers {
		triggers[i] = trigger
	}

	workflow := &WorkflowRecord{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Version:     version,
		Steps:       steps,
		Triggers:    triggers,
		Status:      "draft",
		CreatedAt:   now,
		UpdatedAt:   now,
		Metadata:    req.Metadata,
	}

	if err := (*h.store).Set(ctx, "workflows", workflow.ID, workflow); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to create workflow: " + err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "workflow.created", map[string]any{
		"id":   workflow.ID,
		"name": req.Name,
	})

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    workflow,
	})
}

// List lists all workflows
func (h *WorkflowHandler) List(c *gin.Context) {
	ctx := c.Request.Context()
	status := c.Query("status")

	records, err := (*h.store).List(ctx, "workflows")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to list workflows: " + err.Error(),
			},
		})
		return
	}

	workflows := make([]*WorkflowRecord, 0)
	for _, record := range records {
		var wf WorkflowRecord
		if err := store.DecodeValue(record, &wf); err != nil {
			continue
		}

		// Filter by status
		if status != "" && wf.Status != status {
			continue
		}

		workflows = append(workflows, &wf)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    workflows,
	})
}

// Get retrieves a single workflow
func (h *WorkflowHandler) Get(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var workflow WorkflowRecord
	if err := (*h.store).Get(ctx, "workflows", id, &workflow); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Workflow not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to get workflow: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    &workflow,
	})
}

// Update updates a workflow
func (h *WorkflowHandler) Update(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var req struct {
		Name        *string           `json:"name"`
		Description *string           `json:"description"`
		Version     *string           `json:"version"`
		Steps       []WorkflowStep    `json:"steps"`
		Triggers    []WorkflowTrigger `json:"triggers"`
		Status      *string           `json:"status"`
		Metadata    map[string]any    `json:"metadata"`
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

	var workflow WorkflowRecord
	if err := (*h.store).Get(ctx, "workflows", id, &workflow); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Workflow not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to get workflow: " + err.Error(),
			},
		})
		return
	}

	// Update fields
	if req.Name != nil {
		workflow.Name = *req.Name
	}
	if req.Description != nil {
		workflow.Description = *req.Description
	}
	if req.Version != nil {
		workflow.Version = *req.Version
	}
	if req.Steps != nil {
		steps := make([]any, len(req.Steps))
		for i, step := range req.Steps {
			steps[i] = step
		}
		workflow.Steps = steps
	}
	if req.Triggers != nil {
		triggers := make([]any, len(req.Triggers))
		for i, trigger := range req.Triggers {
			triggers[i] = trigger
		}
		workflow.Triggers = triggers
	}
	if req.Status != nil {
		workflow.Status = *req.Status
	}
	if req.Metadata != nil {
		workflow.Metadata = req.Metadata
	}
	workflow.UpdatedAt = time.Now()

	if err := (*h.store).Set(ctx, "workflows", id, &workflow); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to update workflow: " + err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "workflow.updated", map[string]any{
		"id": id,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    &workflow,
	})
}

// Delete deletes a workflow
func (h *WorkflowHandler) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	if err := (*h.store).Delete(ctx, "workflows", id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Workflow not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to delete workflow: " + err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "workflow.deleted", map[string]any{
		"id": id,
	})

	c.Status(http.StatusNoContent)
}

// Execute executes a workflow
func (h *WorkflowHandler) Execute(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var req struct {
		Context  map[string]any `json:"context"`
		Metadata map[string]any `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.Context = make(map[string]any)
	}

	// Check if workflow exists
	var workflow WorkflowRecord
	if err := (*h.store).Get(ctx, "workflows", id, &workflow); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Workflow not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to get workflow: " + err.Error(),
			},
		})
		return
	}

	// Create execution record
	execution := &WorkflowExecution{
		ID:         uuid.New().String(),
		WorkflowID: id,
		Status:     "pending",
		StartedAt:  time.Now(),
		Context:    req.Context,
		Logs:       []ExecutionLog{},
		Metadata:   req.Metadata,
	}

	if err := (*h.store).Set(ctx, "workflow_executions", execution.ID, execution); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to create execution: " + err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "workflow.execution.started", map[string]any{
		"workflow_id":  id,
		"execution_id": execution.ID,
	})

	// TODO: Actually execute the workflow
	// For now, just return the execution record

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    execution,
	})
}

// GetExecutions retrieves workflow executions
func (h *WorkflowHandler) GetExecutions(c *gin.Context) {
	ctx := c.Request.Context()
	workflowID := c.Param("id")

	records, err := (*h.store).List(ctx, "workflow_executions")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to list executions: " + err.Error(),
			},
		})
		return
	}

	executions := make([]*WorkflowExecution, 0)
	for _, record := range records {
		var exec WorkflowExecution
		if err := store.DecodeValue(record, &exec); err != nil {
			continue
		}
		if exec.WorkflowID == workflowID {
			executions = append(executions, &exec)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    executions,
	})
}

// GetExecutionDetails retrieves a single execution
func (h *WorkflowHandler) GetExecutionDetails(c *gin.Context) {
	ctx := c.Request.Context()
	executionID := c.Param("eid")

	var execution WorkflowExecution
	if err := (*h.store).Get(ctx, "workflow_executions", executionID, &execution); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Execution not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to get execution: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    &execution,
	})
}

// Suspend suspends a workflow
func (h *WorkflowHandler) Suspend(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var workflow WorkflowRecord
	if err := (*h.store).Get(ctx, "workflows", id, &workflow); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Workflow not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to get workflow: " + err.Error(),
			},
		})
		return
	}

	workflow.Status = "inactive"
	workflow.UpdatedAt = time.Now()

	if err := (*h.store).Set(ctx, "workflows", id, &workflow); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to suspend workflow: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    &workflow,
	})
}

// Resume resumes a workflow
func (h *WorkflowHandler) Resume(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var workflow WorkflowRecord
	if err := (*h.store).Get(ctx, "workflows", id, &workflow); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Workflow not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to get workflow: " + err.Error(),
			},
		})
		return
	}

	workflow.Status = "active"
	workflow.UpdatedAt = time.Now()

	if err := (*h.store).Set(ctx, "workflows", id, &workflow); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to resume workflow: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    &workflow,
	})
}
