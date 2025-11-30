package handlers

import (
	"net/http"
	"time"

	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// WorkingMemoryRecord Working Memory 记录
type WorkingMemoryRecord struct {
	ID        string         `json:"id"`
	SessionID string         `json:"session_id,omitempty"`
	AgentID   string         `json:"agent_id,omitempty"`
	Key       string         `json:"key"`
	Value     any            `json:"value"`
	Type      string         `json:"type"`
	TTL       int            `json:"ttl,omitempty"`
	ExpiresAt *time.Time     `json:"expires_at,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// SemanticMemoryRecord Semantic Memory 记录
type SemanticMemoryRecord struct {
	ID        string         `json:"id"`
	Content   string         `json:"content"`
	Embedding []float64      `json:"embedding,omitempty"`
	Source    string         `json:"source,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
	AgentID   string         `json:"agent_id,omitempty"`
	Tags      []string       `json:"tags,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// ProvenanceRecord Provenance 溯源记录
type ProvenanceRecord struct {
	ID         string         `json:"id"`
	EntityType string         `json:"entity_type"`
	EntityID   string         `json:"entity_id"`
	Operation  string         `json:"operation"`
	Actor      string         `json:"actor,omitempty"`
	Timestamp  time.Time      `json:"timestamp"`
	Changes    map[string]any `json:"changes,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// MemoryHandler handles memory-related requests
type MemoryHandler struct {
	store *store.Store
}

// NewMemoryHandler creates a new MemoryHandler
func NewMemoryHandler(st store.Store) *MemoryHandler {
	return &MemoryHandler{store: &st}
}

// CreateWorkingMemory creates working memory
func (h *MemoryHandler) CreateWorkingMemory(c *gin.Context) {
	var req struct {
		SessionID string         `json:"session_id"`
		AgentID   string         `json:"agent_id"`
		Key       string         `json:"key" binding:"required"`
		Value     any            `json:"value" binding:"required"`
		Type      string         `json:"type"`
		TTL       int            `json:"ttl"`
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
	now := time.Now()

	record := &WorkingMemoryRecord{
		ID:        uuid.New().String(),
		SessionID: req.SessionID,
		AgentID:   req.AgentID,
		Key:       req.Key,
		Value:     req.Value,
		Type:      req.Type,
		TTL:       req.TTL,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  req.Metadata,
	}

	if req.TTL > 0 {
		expiresAt := now.Add(time.Duration(req.TTL) * time.Second)
		record.ExpiresAt = &expiresAt
	}

	if err := (*h.store).Set(ctx, "working_memory", record.ID, record); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to create working memory: " + err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "working_memory.created", map[string]any{
		"id":  record.ID,
		"key": req.Key,
	})

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    record,
	})
}

// ListWorkingMemory lists working memory
func (h *MemoryHandler) ListWorkingMemory(c *gin.Context) {
	ctx := c.Request.Context()
	sessionID := c.Query("session_id")
	agentID := c.Query("agent_id")

	records, err := (*h.store).List(ctx, "working_memory")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to list working memory: " + err.Error(),
			},
		})
		return
	}

	memories := make([]*WorkingMemoryRecord, 0)
	now := time.Now()

	for _, record := range records {
		var memory WorkingMemoryRecord
		if err := store.DecodeValue(record, &memory); err != nil {
			continue
		}

		// 检查是否过期
		if memory.ExpiresAt != nil && memory.ExpiresAt.Before(now) {
			continue
		}

		// 过滤
		if sessionID != "" && memory.SessionID != sessionID {
			continue
		}
		if agentID != "" && memory.AgentID != agentID {
			continue
		}

		memories = append(memories, &memory)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    memories,
	})
}

// GetWorkingMemory gets a single working memory
func (h *MemoryHandler) GetWorkingMemory(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var memory WorkingMemoryRecord
	if err := (*h.store).Get(ctx, "working_memory", id, &memory); err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Working memory not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to get working memory: " + err.Error(),
			},
		})
		return
	}

	// 检查是否过期
	if memory.ExpiresAt != nil && memory.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "expired",
				"message": "Working memory has expired",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    &memory,
	})
}

// UpdateWorkingMemory updates working memory
func (h *MemoryHandler) UpdateWorkingMemory(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var req struct {
		Value    any            `json:"value"`
		TTL      *int           `json:"ttl"`
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

	var memory WorkingMemoryRecord
	if err := (*h.store).Get(ctx, "working_memory", id, &memory); err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Working memory not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to get working memory: " + err.Error(),
			},
		})
		return
	}

	// 更新字段
	if req.Value != nil {
		memory.Value = req.Value
	}
	if req.TTL != nil {
		memory.TTL = *req.TTL
		if *req.TTL > 0 {
			expiresAt := time.Now().Add(time.Duration(*req.TTL) * time.Second)
			memory.ExpiresAt = &expiresAt
		} else {
			memory.ExpiresAt = nil
		}
	}
	if req.Metadata != nil {
		memory.Metadata = req.Metadata
	}
	memory.UpdatedAt = time.Now()

	if err := (*h.store).Set(ctx, "working_memory", id, &memory); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to update working memory: " + err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "working_memory.updated", map[string]any{
		"id": id,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    &memory,
	})
}

// DeleteWorkingMemory deletes working memory
func (h *MemoryHandler) DeleteWorkingMemory(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	if err := (*h.store).Delete(ctx, "working_memory", id); err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Working memory not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to delete working memory: " + err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "working_memory.deleted", map[string]any{
		"id": id,
	})

	c.Status(http.StatusNoContent)
}

// ClearWorkingMemory clears working memory
func (h *MemoryHandler) ClearWorkingMemory(c *gin.Context) {
	ctx := c.Request.Context()
	sessionID := c.Query("session_id")
	agentID := c.Query("agent_id")

	records, err := (*h.store).List(ctx, "working_memory")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to list working memory: " + err.Error(),
			},
		})
		return
	}

	deleted := 0
	for _, record := range records {
		var memory WorkingMemoryRecord
		if err := store.DecodeValue(record, &memory); err != nil {
			continue
		}

		// 过滤
		if sessionID != "" && memory.SessionID != sessionID {
			continue
		}
		if agentID != "" && memory.AgentID != agentID {
			continue
		}

		if err := (*h.store).Delete(ctx, "working_memory", memory.ID); err == nil {
			deleted++
		}
	}

	logging.Info(ctx, "working_memory.cleared", map[string]any{
		"deleted_count": deleted,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"deleted_count": deleted,
		},
	})
}

// CreateSemanticMemory creates semantic memory
func (h *MemoryHandler) CreateSemanticMemory(c *gin.Context) {
	var req struct {
		Content   string         `json:"content" binding:"required"`
		Embedding []float64      `json:"embedding"`
		Source    string         `json:"source"`
		SessionID string         `json:"session_id"`
		AgentID   string         `json:"agent_id"`
		Tags      []string       `json:"tags"`
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

	record := &SemanticMemoryRecord{
		ID:        uuid.New().String(),
		Content:   req.Content,
		Embedding: req.Embedding,
		Source:    req.Source,
		SessionID: req.SessionID,
		AgentID:   req.AgentID,
		Tags:      req.Tags,
		CreatedAt: time.Now(),
		Metadata:  req.Metadata,
	}

	if err := (*h.store).Set(ctx, "semantic_memory", record.ID, record); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to create semantic memory: " + err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "semantic_memory.created", map[string]any{
		"id": record.ID,
	})

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    record,
	})
}

// SearchSemanticMemory searches semantic memory
func (h *MemoryHandler) SearchSemanticMemory(c *gin.Context) {
	var req struct {
		Query     string    `json:"query"`
		Embedding []float64 `json:"embedding"`
		Limit     int       `json:"limit"`
		SessionID string    `json:"session_id"`
		AgentID   string    `json:"agent_id"`
		Tags      []string  `json:"tags"`
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

	// TODO: Implement vector search
	// For now, return simple list
	records, err := (*h.store).List(ctx, "semantic_memory")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to search semantic memory: " + err.Error(),
			},
		})
		return
	}

	memories := make([]*SemanticMemoryRecord, 0)
	for _, record := range records {
		var memory SemanticMemoryRecord
		if err := store.DecodeValue(record, &memory); err != nil {
			continue
		}
		memories = append(memories, &memory)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    memories,
	})
}

// GetProvenance gets memory provenance
func (h *MemoryHandler) GetProvenance(c *gin.Context) {
	id := c.Param("id")

	// TODO: Implement provenance tracking
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"entity_id": id,
			"history":   []any{},
		},
	})
}

// ConsolidateMemory consolidates memory
func (h *MemoryHandler) ConsolidateMemory(c *gin.Context) {
	// TODO: Implement memory consolidation logic
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message": "Consolidation started",
			"job_id":  uuid.New().String(),
		},
	})
}
