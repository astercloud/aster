package handlers

import (
	"net/http"
	"time"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/types"
	"github.com/gin-gonic/gin"
)

// AgentHandler handles agent-related requests
type AgentHandler struct {
	store *store.Store
	deps  *agent.Dependencies
}

// NewAgentHandler creates a new AgentHandler
func NewAgentHandler(st store.Store, deps *agent.Dependencies) *AgentHandler {
	return &AgentHandler{
		store: &st,
		deps:  deps,
	}
}

// Create creates a new agent
func (h *AgentHandler) Create(c *gin.Context) {
	var req struct {
		TemplateID    string                     `json:"template_id" binding:"required"`
		Name          string                     `json:"name"`
		ModelConfig   *types.ModelConfig         `json:"model_config"`
		Sandbox       *types.SandboxConfig       `json:"sandbox"`
		Middlewares   []string                   `json:"middlewares"`
		Metadata      map[string]interface{}     `json:"metadata"`
		SkillsPackage *types.SkillsPackageConfig `json:"skills_package"`
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

	// 创建 Agent 配置
	config := &types.AgentConfig{
		TemplateID:    req.TemplateID,
		ModelConfig:   req.ModelConfig,
		Sandbox:       req.Sandbox,
		Middlewares:   req.Middlewares,
		Metadata:      req.Metadata,
		SkillsPackage: req.SkillsPackage,
	}

	// 创建 Agent 实例
	ag, err := agent.Create(ctx, config, h.deps)
	if err != nil {
		logging.Error(ctx, "agent.create.error", map[string]interface{}{
			"template_id": req.TemplateID,
			"error":       err.Error(),
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
	defer func() { _ = ag.Close() }()

	// 保存 Agent 记录
	record := &AgentRecord{
		ID:        ag.ID(),
		Config:    config,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata: map[string]interface{}{
			"name": req.Name,
		},
	}

	if err := (*h.store).Set(ctx, "agents", ag.ID(), record); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to save agent: " + err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "agent.created", map[string]interface{}{
		"agent_id":    ag.ID(),
		"template_id": req.TemplateID,
	})

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    record,
	})
}

// List lists all agents
func (h *AgentHandler) List(c *gin.Context) {
	ctx := c.Request.Context()

	records, err := (*h.store).List(ctx, "agents")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to list agents: " + err.Error(),
			},
		})
		return
	}

	agents := make([]*AgentRecord, 0, len(records))
	for _, record := range records {
		var agent AgentRecord
		if err := store.DecodeValue(record, &agent); err != nil {
			continue
		}
		agents = append(agents, &agent)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    agents,
	})
}

// Get retrieves a single agent
func (h *AgentHandler) Get(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var agent AgentRecord
	if err := (*h.store).Get(ctx, "agents", id, &agent); err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Agent not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to get agent: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    &agent,
	})
}

// Delete deletes an agent
func (h *AgentHandler) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	if err := (*h.store).Delete(ctx, "agents", id); err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Agent not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to delete agent: " + err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "agent.deleted", map[string]interface{}{
		"agent_id": id,
	})

	c.Status(http.StatusNoContent)
}

// Update updates an agent
func (h *AgentHandler) Update(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var req struct {
		Name     *string                `json:"name"`
		Metadata map[string]interface{} `json:"metadata"`
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

	// 获取现有 Agent
	var agentRecord AgentRecord
	if err := (*h.store).Get(ctx, "agents", id, &agentRecord); err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Agent not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to get agent: " + err.Error(),
			},
		})
		return
	}

	// 更新字段
	if req.Name != nil {
		if agentRecord.Metadata == nil {
			agentRecord.Metadata = make(map[string]interface{})
		}
		agentRecord.Metadata["name"] = *req.Name
	}

	if req.Metadata != nil {
		for k, v := range req.Metadata {
			if agentRecord.Metadata == nil {
				agentRecord.Metadata = make(map[string]interface{})
			}
			agentRecord.Metadata[k] = v
		}
	}

	agentRecord.UpdatedAt = time.Now()

	// 保存更新
	if err := (*h.store).Set(ctx, "agents", id, &agentRecord); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to update agent: " + err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "agent.updated", map[string]interface{}{
		"agent_id": id,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    &agentRecord,
	})
}

// GetStats retrieves agent statistics
func (h *AgentHandler) GetStats(c *gin.Context) {
	id := c.Param("id")

	// TODO: Implement real statistics
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"agent_id":          id,
			"total_sessions":    0,
			"total_messages":    0,
			"avg_response_time": 0,
		},
	})
}

// Chat handles chat requests
func (h *AgentHandler) Chat(c *gin.Context) {
	// TODO: Implement chat logic from cmd/aster
	c.JSON(http.StatusNotImplemented, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "not_implemented",
			"message": "Chat endpoint not yet implemented",
		},
	})
}

// StreamChat handles streaming chat requests
func (h *AgentHandler) StreamChat(c *gin.Context) {
	// TODO: Implement streaming chat logic
	c.JSON(http.StatusNotImplemented, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "not_implemented",
			"message": "Stream chat endpoint not yet implemented",
		},
	})
}
