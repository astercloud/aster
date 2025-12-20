package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/types"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CheckpointRecord Checkpoint 记录
type CheckpointRecord struct {
	ID        string        `json:"id"`
	SessionID string        `json:"session_id"`
	Snapshot  SessionRecord `json:"snapshot"`
	CreatedAt time.Time     `json:"created_at"`
	Label     string        `json:"label,omitempty"`
}

// SessionHandler handles session-related requests
type SessionHandler struct {
	store *store.Store
}

// NewSessionHandler creates a new SessionHandler
func NewSessionHandler(st store.Store) *SessionHandler {
	return &SessionHandler{store: &st}
}

// Create creates a new session
func (h *SessionHandler) Create(c *gin.Context) {
	var req struct {
		AgentID  string         `json:"agent_id" binding:"required"`
		Context  map[string]any `json:"context"`
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
	record := &SessionRecord{
		ID:        uuid.New().String(),
		AgentID:   req.AgentID,
		Status:    "active",
		Messages:  []types.Message{},
		Context:   req.Context,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  req.Metadata,
	}

	if err := (*h.store).Set(ctx, "sessions", record.ID, record); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to create session: " + err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "session.created", map[string]any{
		"session_id": record.ID,
		"agent_id":   req.AgentID,
	})

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    record,
	})
}

// List lists all sessions
func (h *SessionHandler) List(c *gin.Context) {
	ctx := c.Request.Context()
	agentID := c.Query("agent_id")
	status := c.Query("status")

	records, err := (*h.store).List(ctx, "sessions")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to list sessions: " + err.Error(),
			},
		})
		return
	}

	sessions := make([]*SessionRecord, 0)
	for _, record := range records {
		var session SessionRecord
		if err := store.DecodeValue(record, &session); err != nil {
			continue
		}

		// 过滤
		if agentID != "" && session.AgentID != agentID {
			continue
		}
		if status != "" && session.Status != status {
			continue
		}

		sessions = append(sessions, &session)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    sessions,
	})
}

// Get retrieves a single session
func (h *SessionHandler) Get(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var session SessionRecord
	if err := (*h.store).Get(ctx, "sessions", id, &session); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Session not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to get session: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    &session,
	})
}

// Update updates a session
func (h *SessionHandler) Update(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var req struct {
		Status   *string        `json:"status"`
		Context  map[string]any `json:"context"`
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

	var session SessionRecord
	if err := (*h.store).Get(ctx, "sessions", id, &session); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Session not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to get session: " + err.Error(),
			},
		})
		return
	}

	// 更新字段
	if req.Status != nil {
		session.Status = *req.Status
	}
	if req.Context != nil {
		session.Context = req.Context
	}
	if req.Metadata != nil {
		session.Metadata = req.Metadata
	}
	session.UpdatedAt = time.Now()

	if err := (*h.store).Set(ctx, "sessions", id, &session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to update session: " + err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "session.updated", map[string]any{
		"session_id": id,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    &session,
	})
}

// Delete deletes a session
func (h *SessionHandler) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	if err := (*h.store).Delete(ctx, "sessions", id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Session not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to delete session: " + err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "session.deleted", map[string]any{
		"session_id": id,
	})

	c.Status(http.StatusNoContent)
}

// GetMessages retrieves session messages
func (h *SessionHandler) GetMessages(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var session SessionRecord
	if err := (*h.store).Get(ctx, "sessions", id, &session); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Session not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to get session: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    session.Messages,
	})
}

// GetCheckpoints retrieves session checkpoints
func (h *SessionHandler) GetCheckpoints(c *gin.Context) {
	ctx := c.Request.Context()
	sessionID := c.Param("id")

	records, err := (*h.store).List(ctx, "checkpoints")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to list checkpoints: " + err.Error(),
			},
		})
		return
	}

	checkpoints := make([]*CheckpointRecord, 0)
	for _, record := range records {
		var checkpoint CheckpointRecord
		if err := store.DecodeValue(record, &checkpoint); err != nil {
			continue
		}
		if checkpoint.SessionID == sessionID {
			checkpoints = append(checkpoints, &checkpoint)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    checkpoints,
	})
}

// Resume resumes a session
func (h *SessionHandler) Resume(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var session SessionRecord
	if err := (*h.store).Get(ctx, "sessions", id, &session); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Session not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to get session: " + err.Error(),
			},
		})
		return
	}

	session.Status = "active"
	session.UpdatedAt = time.Now()

	if err := (*h.store).Set(ctx, "sessions", id, &session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to resume session: " + err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "session.resumed", map[string]any{
		"session_id": id,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    &session,
	})
}

// GetStats retrieves session statistics
func (h *SessionHandler) GetStats(c *gin.Context) {
	id := c.Param("id")

	// TODO: Implement real statistics
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"session_id":       id,
			"message_count":    0,
			"duration":         0,
			"checkpoint_count": 0,
		},
	})
}
