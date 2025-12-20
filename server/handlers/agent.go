package handlers

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
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
		MiddlewareCfg map[string]map[string]any  `json:"middleware_config"`
		Metadata      map[string]any             `json:"metadata"`
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
		TemplateID:       req.TemplateID,
		ModelConfig:      req.ModelConfig,
		Sandbox:          req.Sandbox,
		Middlewares:      req.Middlewares,
		MiddlewareConfig: req.MiddlewareCfg,
		Metadata:         req.Metadata,
		SkillsPackage:    req.SkillsPackage,
	}

	// 创建 Agent 实例
	ag, err := agent.Create(ctx, config, h.deps)
	if err != nil {
		logging.Error(ctx, "agent.create.error", map[string]any{
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
		Metadata: map[string]any{
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

	logging.Info(ctx, "agent.created", map[string]any{
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
		if errors.Is(err, store.ErrNotFound) {
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
		if errors.Is(err, store.ErrNotFound) {
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

	logging.Info(ctx, "agent.deleted", map[string]any{
		"agent_id": id,
	})

	c.Status(http.StatusNoContent)
}

// Update updates an agent
func (h *AgentHandler) Update(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var req struct {
		Name     *string        `json:"name"`
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

	// 获取现有 Agent
	var agentRecord AgentRecord
	if err := (*h.store).Get(ctx, "agents", id, &agentRecord); err != nil {
		if errors.Is(err, store.ErrNotFound) {
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
			agentRecord.Metadata = make(map[string]any)
		}
		agentRecord.Metadata["name"] = *req.Name
	}

	if req.Metadata != nil {
		for k, v := range req.Metadata {
			if agentRecord.Metadata == nil {
				agentRecord.Metadata = make(map[string]any)
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

	logging.Info(ctx, "agent.updated", map[string]any{
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

// Run runs an agent with a message
func (h *AgentHandler) Run(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var req struct {
		Message string         `json:"message" binding:"required"`
		Context map[string]any `json:"context"`
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

	// Get agent record
	var agentRecord AgentRecord
	if err := (*h.store).Get(ctx, "agents", id, &agentRecord); err != nil {
		if errors.Is(err, store.ErrNotFound) {
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

	// Create agent instance
	ag, err := agent.Create(ctx, agentRecord.Config, h.deps)
	if err != nil {
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

	// Send message
	if err := ag.Send(ctx, req.Message); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to run agent: " + err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "agent.run", map[string]any{
		"agent_id": id,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"status":  "running",
			"message": "Agent task started",
		},
	})
}

// Send sends a message to an agent
func (h *AgentHandler) Send(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var req struct {
		Message string `json:"message" binding:"required"`
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

	// Get agent record
	var agentRecord AgentRecord
	if err := (*h.store).Get(ctx, "agents", id, &agentRecord); err != nil {
		if errors.Is(err, store.ErrNotFound) {
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

	// Create agent instance
	ag, err := agent.Create(ctx, agentRecord.Config, h.deps)
	if err != nil {
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

	// Send message
	if err := ag.Send(ctx, req.Message); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to send message: " + err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "agent.send", map[string]any{
		"agent_id": id,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message": "Message sent successfully",
		},
	})
}

// GetStatus retrieves agent status
func (h *AgentHandler) GetStatus(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	// Get agent record
	var agentRecord AgentRecord
	if err := (*h.store).Get(ctx, "agents", id, &agentRecord); err != nil {
		if errors.Is(err, store.ErrNotFound) {
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
		"data": gin.H{
			"agent_id": id,
			"status":   agentRecord.Status,
		},
	})
}

// Resume resumes an agent from storage
func (h *AgentHandler) Resume(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	// Get agent record
	var agentRecord AgentRecord
	if err := (*h.store).Get(ctx, "agents", id, &agentRecord); err != nil {
		if errors.Is(err, store.ErrNotFound) {
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

	// Create agent instance (will load from storage)
	ag, err := agent.Create(ctx, agentRecord.Config, h.deps)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to resume agent: " + err.Error(),
			},
		})
		return
	}
	defer func() { _ = ag.Close() }()

	logging.Info(ctx, "agent.resumed", map[string]any{
		"agent_id": id,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"agent_id": ag.ID(),
			"status":   ag.Status(),
		},
	})
}

// Chat handles chat requests
func (h *AgentHandler) Chat(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		TemplateID  string               `json:"template_id" binding:"required"`
		Input       string               `json:"input" binding:"required"`
		ModelConfig *types.ModelConfig   `json:"model_config"`
		Sandbox     *types.SandboxConfig `json:"sandbox"`
		Middlewares []string             `json:"middlewares"`
		Metadata    map[string]any       `json:"metadata"`
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

	// Create agent configuration
	cfg := &types.AgentConfig{
		TemplateID:  req.TemplateID,
		ModelConfig: req.ModelConfig,
		Sandbox:     req.Sandbox,
		Middlewares: req.Middlewares,
		Metadata:    req.Metadata,
	}

	// If ModelConfig is provided but missing API key, try to fill from environment
	if cfg.ModelConfig != nil && cfg.ModelConfig.APIKey == "" {
		provider := cfg.ModelConfig.Provider
		var apiKey string
		switch provider {
		case "deepseek":
			apiKey = os.Getenv("DEEPSEEK_API_KEY")
		case "anthropic":
			apiKey = os.Getenv("ANTHROPIC_API_KEY")
		case "openai":
			apiKey = os.Getenv("OPENAI_API_KEY")
		default:
			apiKey = os.Getenv(strings.ToUpper(provider) + "_API_KEY")
		}
		if apiKey != "" {
			cfg.ModelConfig.APIKey = apiKey
		}
	}

	// Create agent instance
	ag, err := agent.Create(ctx, cfg, h.deps)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "agent_creation_failed",
				"message": err.Error(),
			},
		})
		return
	}

	// Execute chat
	result, err := ag.Chat(ctx, req.Input)
	if err != nil {
		logging.Error(ctx, "chat.failed", map[string]any{
			"agent_id": ag.ID(),
			"error":    err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "chat_failed",
				"message": err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "chat.completed", map[string]any{
		"agent_id":    ag.ID(),
		"text_length": len(result.Text),
		"status":      result.Status,
	})

	// Return response
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"agent_id": ag.ID(),
		"text":     result.Text,
		"output":   result.Text,
		"status":   result.Status,
	})
}

// StreamChat handles streaming chat requests
func (h *AgentHandler) StreamChat(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		TemplateID  string               `json:"template_id" binding:"required"`
		Input       string               `json:"input" binding:"required"`
		ModelConfig *types.ModelConfig   `json:"model_config"`
		Sandbox     *types.SandboxConfig `json:"sandbox"`
		Middlewares []string             `json:"middlewares"`
		Metadata    map[string]any       `json:"metadata"`
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

	// Create agent configuration
	cfg := &types.AgentConfig{
		TemplateID:  req.TemplateID,
		ModelConfig: req.ModelConfig,
		Sandbox:     req.Sandbox,
		Middlewares: req.Middlewares,
		Metadata:    req.Metadata,
	}

	// Fill API key from environment if missing
	if cfg.ModelConfig != nil && cfg.ModelConfig.APIKey == "" {
		provider := cfg.ModelConfig.Provider
		var apiKey string
		switch provider {
		case "deepseek":
			apiKey = os.Getenv("DEEPSEEK_API_KEY")
		case "anthropic":
			apiKey = os.Getenv("ANTHROPIC_API_KEY")
		case "openai":
			apiKey = os.Getenv("OPENAI_API_KEY")
		default:
			apiKey = os.Getenv(strings.ToUpper(provider) + "_API_KEY")
		}
		if apiKey != "" {
			cfg.ModelConfig.APIKey = apiKey
		}
	}

	// Create agent instance
	ag, err := agent.Create(ctx, cfg, h.deps)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "agent_creation_failed",
				"message": err.Error(),
			},
		})
		return
	}
	defer func() { _ = ag.Close() }()

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Stream events to client
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "streaming_not_supported",
				"message": "Streaming not supported",
			},
		})
		return
	}

	// Use Stream iterator
	reader := ag.Stream(ctx, req.Input)
	for {
		event, err := reader.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			logging.Error(ctx, "stream.error", map[string]any{
				"agent_id": ag.ID(),
				"error":    err.Error(),
			})
			c.SSEvent("error", gin.H{
				"code":    "stream_error",
				"message": err.Error(),
			})
			flusher.Flush()
			break
		}

		if event != nil {
			// Extract text content from event
			var textContent string
			var textContentSb781 strings.Builder
			for _, block := range event.Content.ContentBlocks {
				if tb, ok := block.(*types.TextBlock); ok {
					textContentSb781.WriteString(tb.Text)
				}
			}
			textContent += textContentSb781.String()

			if textContent != "" {
				c.SSEvent("message", gin.H{
					"type": "text_delta",
					"text": textContent,
				})
				flusher.Flush()
			}
		}
	}

	logging.Info(ctx, "stream.completed", map[string]any{
		"agent_id": ag.ID(),
	})
}
