package handlers

import (
	"net/http"
	"time"

	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ToolExecution Tool 执行记录
type ToolExecution struct {
	ID          string         `json:"id"`
	ToolID      string         `json:"tool_id"`
	Input       map[string]any `json:"input"`
	Output      map[string]any `json:"output,omitempty"`
	Status      string         `json:"status"` // pending, running, completed, failed
	StartedAt   time.Time      `json:"started_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	Error       string         `json:"error,omitempty"`
}

// ToolHandler handles tool-related requests
type ToolHandler struct {
	store *store.Store
}

// NewToolHandler creates a new ToolHandler
func NewToolHandler(st store.Store) *ToolHandler {
	return &ToolHandler{store: &st}
}

// Create creates a new tool
func (h *ToolHandler) Create(c *gin.Context) {
	var req struct {
		Name        string         `json:"name" binding:"required"`
		Description string         `json:"description"`
		Type        string         `json:"type"`
		Schema      map[string]any `json:"schema" binding:"required"`
		Config      map[string]any `json:"config"`
		Metadata    map[string]any `json:"metadata"`
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
	tool := &ToolRecord{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Schema:      req.Schema,
		Config:      req.Config,
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    req.Metadata,
	}

	if err := (*h.store).Set(ctx, "tools", tool.ID, tool); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "tool.created", map[string]any{
		"id":   tool.ID,
		"name": req.Name,
	})

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    tool,
	})
}

// List lists all tools
func (h *ToolHandler) List(c *gin.Context) {
	ctx := c.Request.Context()
	status := c.Query("status")
	toolType := c.Query("type")

	records, err := (*h.store).List(ctx, "tools")
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

	tools := make([]*ToolRecord, 0)
	for _, record := range records {
		var tool ToolRecord
		if err := store.DecodeValue(record, &tool); err != nil {
			continue
		}

		// Filter
		if status != "" && tool.Status != status {
			continue
		}
		if toolType != "" && tool.Type != toolType {
			continue
		}

		tools = append(tools, &tool)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tools,
	})
}

// Get retrieves a single tool
func (h *ToolHandler) Get(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var tool ToolRecord
	if err := (*h.store).Get(ctx, "tools", id, &tool); err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Tool not found",
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
		"data":    &tool,
	})
}

// Update updates a tool
func (h *ToolHandler) Update(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var req struct {
		Name        *string        `json:"name"`
		Description *string        `json:"description"`
		Type        *string        `json:"type"`
		Schema      map[string]any `json:"schema"`
		Config      map[string]any `json:"config"`
		Status      *string        `json:"status"`
		Metadata    map[string]any `json:"metadata"`
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

	var tool ToolRecord
	if err := (*h.store).Get(ctx, "tools", id, &tool); err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Tool not found",
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

	// Update fields
	if req.Name != nil {
		tool.Name = *req.Name
	}
	if req.Description != nil {
		tool.Description = *req.Description
	}
	if req.Type != nil {
		tool.Type = *req.Type
	}
	if req.Schema != nil {
		tool.Schema = req.Schema
	}
	if req.Config != nil {
		tool.Config = req.Config
	}
	if req.Status != nil {
		tool.Status = *req.Status
	}
	if req.Metadata != nil {
		tool.Metadata = req.Metadata
	}
	tool.UpdatedAt = time.Now()

	if err := (*h.store).Set(ctx, "tools", id, &tool); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "tool.updated", map[string]any{
		"id": id,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    &tool,
	})
}

// Delete deletes a tool
func (h *ToolHandler) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	if err := (*h.store).Delete(ctx, "tools", id); err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Tool not found",
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

	logging.Info(ctx, "tool.deleted", map[string]any{
		"id": id,
	})

	c.Status(http.StatusNoContent)
}

// Execute executes a tool
func (h *ToolHandler) Execute(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var req struct {
		Input map[string]any `json:"input" binding:"required"`
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

	// Check if tool exists
	var tool ToolRecord
	if err := (*h.store).Get(ctx, "tools", id, &tool); err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Tool not found",
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

	// Create execution record
	execution := &ToolExecution{
		ID:        uuid.New().String(),
		ToolID:    id,
		Input:     req.Input,
		Status:    "pending",
		StartedAt: time.Now(),
	}

	if err := (*h.store).Set(ctx, "tool_executions", execution.ID, execution); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "tool.execution.started", map[string]any{
		"tool_id":      id,
		"execution_id": execution.ID,
	})

	// TODO: Actually execute the tool
	// For now, just return the execution record

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    execution,
	})
}
