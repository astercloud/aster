package handlers

import (
	"net/http"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/core"
	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/types"
	"github.com/gin-gonic/gin"
)

// PoolHandler handles pool-related requests
type PoolHandler struct {
	store *store.Store
	deps  *agent.Dependencies
	pool  *core.Pool
}

// NewPoolHandler creates a new PoolHandler
func NewPoolHandler(st store.Store, deps *agent.Dependencies) *PoolHandler {
	// Create pool with dependencies
	pool := core.NewPool(&core.PoolOptions{
		Dependencies: deps,
		MaxAgents:    100,
	})

	return &PoolHandler{
		store: &st,
		deps:  deps,
		pool:  pool,
	}
}

// CreateAgent creates a new agent in the pool
func (h *PoolHandler) CreateAgent(c *gin.Context) {
	var req struct {
		AgentID       string                    `json:"agent_id"`
		TemplateID    string                    `json:"template_id" binding:"required"`
		ModelConfig   *types.ModelConfig        `json:"model_config"`
		Sandbox       *types.SandboxConfig      `json:"sandbox"`
		Middlewares   []string                  `json:"middlewares"`
		MiddlewareCfg map[string]map[string]any `json:"middleware_config"`
		Metadata      map[string]any            `json:"metadata"`
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

	// Create agent config
	config := &types.AgentConfig{
		AgentID:          req.AgentID,
		TemplateID:       req.TemplateID,
		ModelConfig:      req.ModelConfig,
		Sandbox:          req.Sandbox,
		Middlewares:      req.Middlewares,
		MiddlewareConfig: req.MiddlewareCfg,
		Metadata:         req.Metadata,
	}

	// Create agent in pool
	ag, err := h.pool.Create(ctx, config)
	if err != nil {
		logging.Error(ctx, "pool.create.error", map[string]any{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to create agent: " + err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "pool.agent.created", map[string]any{
		"agent_id": ag.ID(),
	})

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"agent_id": ag.ID(),
			"status":   ag.Status(),
		},
	})
}

// ListAgents lists all agents in the pool
func (h *PoolHandler) ListAgents(c *gin.Context) {
	prefix := c.Query("prefix")
	ids := h.pool.List(prefix)

	agents := make([]gin.H, 0, len(ids))
	for _, id := range ids {
		if ag, exists := h.pool.Get(id); exists {
			agents = append(agents, gin.H{
				"agent_id": id,
				"status":   ag.Status(),
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"agents": agents,
			"count":  len(agents),
		},
	})
}

// GetAgent retrieves a single agent from the pool
func (h *PoolHandler) GetAgent(c *gin.Context) {
	id := c.Param("id")

	ag, exists := h.pool.Get(id)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_found",
				"message": "Agent not found in pool",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"agent_id": ag.ID(),
			"status":   ag.Status(),
		},
	})
}

// ResumeAgent resumes an agent from storage
func (h *PoolHandler) ResumeAgent(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		TemplateID    string                    `json:"template_id" binding:"required"`
		ModelConfig   *types.ModelConfig        `json:"model_config"`
		Sandbox       *types.SandboxConfig      `json:"sandbox"`
		Middlewares   []string                  `json:"middlewares"`
		MiddlewareCfg map[string]map[string]any `json:"middleware_config"`
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

	// Create agent config
	config := &types.AgentConfig{
		TemplateID:       req.TemplateID,
		ModelConfig:      req.ModelConfig,
		Sandbox:          req.Sandbox,
		Middlewares:      req.Middlewares,
		MiddlewareConfig: req.MiddlewareCfg,
	}

	// Resume agent
	ag, err := h.pool.Resume(ctx, id, config)
	if err != nil {
		logging.Error(ctx, "pool.resume.error", map[string]any{
			"agent_id": id,
			"error":    err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to resume agent: " + err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "pool.agent.resumed", map[string]any{
		"agent_id": ag.ID(),
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"agent_id": ag.ID(),
			"status":   ag.Status(),
		},
	})
}

// RemoveAgent removes an agent from the pool
func (h *PoolHandler) RemoveAgent(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	if err := h.pool.Remove(id); err != nil {
		logging.Error(ctx, "pool.remove.error", map[string]any{
			"agent_id": id,
			"error":    err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to remove agent: " + err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "pool.agent.removed", map[string]any{
		"agent_id": id,
	})

	c.Status(http.StatusNoContent)
}

// GetStats retrieves pool statistics
func (h *PoolHandler) GetStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"total_agents": h.pool.Size(),
			"max_agents":   100,
		},
	})
}
