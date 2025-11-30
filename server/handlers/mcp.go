package handlers

import (
	"net/http"
	"time"

	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MCPServerRecord MCP 服务器记录
type MCPServerRecord struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Type      string         `json:"type"` // stdio, sse, http
	Command   string         `json:"command,omitempty"`
	Args      []string       `json:"args,omitempty"`
	Config    map[string]any `json:"config,omitempty"`
	Status    string         `json:"status"` // stopped, running, error
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// MCPHandler handles MCP-related requests
type MCPHandler struct {
	store *store.Store
}

// NewMCPHandler creates a new MCPHandler
func NewMCPHandler(st store.Store) *MCPHandler {
	return &MCPHandler{store: &st}
}

// Create creates a new MCP server
func (h *MCPHandler) Create(c *gin.Context) {
	var req struct {
		Name     string         `json:"name" binding:"required"`
		Type     string         `json:"type" binding:"required"`
		Command  string         `json:"command"`
		Args     []string       `json:"args"`
		Config   map[string]any `json:"config"`
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
	server := &MCPServerRecord{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Type:      req.Type,
		Command:   req.Command,
		Args:      req.Args,
		Config:    req.Config,
		Status:    "stopped",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  req.Metadata,
	}

	if err := (*h.store).Set(ctx, "mcp_servers", server.ID, server); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "mcp.server.created", map[string]any{
		"id":   server.ID,
		"name": req.Name,
	})

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    server,
	})
}

// List lists all MCP servers
func (h *MCPHandler) List(c *gin.Context) {
	ctx := c.Request.Context()
	status := c.Query("status")
	serverType := c.Query("type")

	records, err := (*h.store).List(ctx, "mcp_servers")
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

	servers := make([]*MCPServerRecord, 0)
	for _, record := range records {
		var srv MCPServerRecord
		if err := store.DecodeValue(record, &srv); err != nil {
			continue
		}

		// Filter
		if status != "" && srv.Status != status {
			continue
		}
		if serverType != "" && srv.Type != serverType {
			continue
		}

		servers = append(servers, &srv)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    servers,
	})
}

// Get retrieves a single MCP server
func (h *MCPHandler) Get(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var server MCPServerRecord
	if err := (*h.store).Get(ctx, "mcp_servers", id, &server); err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "MCP server not found",
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
		"data":    &server,
	})
}

// Update updates an MCP server
func (h *MCPHandler) Update(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var req struct {
		Name     *string        `json:"name"`
		Command  *string        `json:"command"`
		Args     []string       `json:"args"`
		Config   map[string]any `json:"config"`
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

	var server MCPServerRecord
	if err := (*h.store).Get(ctx, "mcp_servers", id, &server); err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "MCP server not found",
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
		server.Name = *req.Name
	}
	if req.Command != nil {
		server.Command = *req.Command
	}
	if req.Args != nil {
		server.Args = req.Args
	}
	if req.Config != nil {
		server.Config = req.Config
	}
	if req.Metadata != nil {
		server.Metadata = req.Metadata
	}
	server.UpdatedAt = time.Now()

	if err := (*h.store).Set(ctx, "mcp_servers", id, &server); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "mcp.server.updated", map[string]any{
		"id": id,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    &server,
	})
}

// Delete deletes an MCP server
func (h *MCPHandler) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	if err := (*h.store).Delete(ctx, "mcp_servers", id); err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "MCP server not found",
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

	logging.Info(ctx, "mcp.server.deleted", map[string]any{
		"id": id,
	})

	c.Status(http.StatusNoContent)
}

// Connect connects to an MCP server (start)
func (h *MCPHandler) Connect(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var server MCPServerRecord
	if err := (*h.store).Get(ctx, "mcp_servers", id, &server); err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "MCP server not found",
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

	// TODO: Actually start the MCP server
	server.Status = "running"
	server.UpdatedAt = time.Now()

	if err := (*h.store).Set(ctx, "mcp_servers", id, &server); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "mcp.server.connected", map[string]any{
		"id": id,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    &server,
	})
}

// Disconnect disconnects from an MCP server (stop)
func (h *MCPHandler) Disconnect(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var server MCPServerRecord
	if err := (*h.store).Get(ctx, "mcp_servers", id, &server); err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "MCP server not found",
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

	// TODO: Actually stop the MCP server
	server.Status = "stopped"
	server.UpdatedAt = time.Now()

	if err := (*h.store).Set(ctx, "mcp_servers", id, &server); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "mcp.server.disconnected", map[string]any{
		"id": id,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    &server,
	})
}
