package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/astercloud/aster/pkg/core"
	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RoomRecord Room 持久化记录
type RoomRecord struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Members   []core.RoomMember `json:"members"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Metadata  map[string]any    `json:"metadata,omitempty"`
}

// RoomHandler handles room-related requests
type RoomHandler struct {
	store *store.Store
	pool  *core.Pool
	rooms map[string]*core.Room
}

// NewRoomHandler creates a new RoomHandler
func NewRoomHandler(st store.Store, pool *core.Pool) *RoomHandler {
	return &RoomHandler{
		store: &st,
		pool:  pool,
		rooms: make(map[string]*core.Room),
	}
}

// Create creates a new room
func (h *RoomHandler) Create(c *gin.Context) {
	var req struct {
		Name     string         `json:"name" binding:"required"`
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
	now := time.Now()

	// Create room
	room := core.NewRoom(h.pool)
	roomID := uuid.New().String()

	// Save to memory
	h.rooms[roomID] = room

	// Save record
	record := &RoomRecord{
		ID:        roomID,
		Name:      req.Name,
		Members:   []core.RoomMember{},
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  req.Metadata,
	}

	if err := (*h.store).Set(ctx, "rooms", roomID, record); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to create room: " + err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "room.created", map[string]any{
		"room_id": roomID,
		"name":    req.Name,
	})

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    record,
	})
}

// List lists all rooms
func (h *RoomHandler) List(c *gin.Context) {
	ctx := c.Request.Context()

	records, err := (*h.store).List(ctx, "rooms")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to list rooms: " + err.Error(),
			},
		})
		return
	}

	rooms := make([]*RoomRecord, 0)
	for _, record := range records {
		var room RoomRecord
		if err := store.DecodeValue(record, &room); err != nil {
			continue
		}
		rooms = append(rooms, &room)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    rooms,
	})
}

// Get retrieves a single room
func (h *RoomHandler) Get(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var room RoomRecord
	if err := (*h.store).Get(ctx, "rooms", id, &room); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Room not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to get room: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    &room,
	})
}

// Delete deletes a room
func (h *RoomHandler) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	// Remove from memory
	delete(h.rooms, id)

	// Delete from store
	if err := (*h.store).Delete(ctx, "rooms", id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Room not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to delete room: " + err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "room.deleted", map[string]any{
		"room_id": id,
	})

	c.Status(http.StatusNoContent)
}

// Join adds a member to the room
func (h *RoomHandler) Join(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var req struct {
		Name    string `json:"name" binding:"required"`
		AgentID string `json:"agent_id" binding:"required"`
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

	// Get room from memory
	room, exists := h.rooms[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_found",
				"message": "Room not found",
			},
		})
		return
	}

	// Join room
	if err := room.Join(req.Name, req.AgentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to join room: " + err.Error(),
			},
		})
		return
	}

	// Update record
	var record RoomRecord
	if err := (*h.store).Get(ctx, "rooms", id, &record); err == nil {
		record.Members = room.GetMembers()
		record.UpdatedAt = time.Now()
		_ = (*h.store).Set(ctx, "rooms", id, &record)
	}

	logging.Info(ctx, "room.member.joined", map[string]any{
		"room_id": id,
		"member":  req.Name,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message": "Member joined successfully",
		},
	})
}

// Leave removes a member from the room
func (h *RoomHandler) Leave(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var req struct {
		Name string `json:"name" binding:"required"`
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

	// Get room from memory
	room, exists := h.rooms[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_found",
				"message": "Room not found",
			},
		})
		return
	}

	// Leave room
	if err := room.Leave(req.Name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to leave room: " + err.Error(),
			},
		})
		return
	}

	// Update record
	var record RoomRecord
	if err := (*h.store).Get(ctx, "rooms", id, &record); err == nil {
		record.Members = room.GetMembers()
		record.UpdatedAt = time.Now()
		_ = (*h.store).Set(ctx, "rooms", id, &record)
	}

	logging.Info(ctx, "room.member.left", map[string]any{
		"room_id": id,
		"member":  req.Name,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message": "Member left successfully",
		},
	})
}

// Say sends a message in the room
func (h *RoomHandler) Say(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var req struct {
		From string `json:"from" binding:"required"`
		Text string `json:"text" binding:"required"`
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

	// Get room from memory
	room, exists := h.rooms[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_found",
				"message": "Room not found",
			},
		})
		return
	}

	// Send message
	if err := room.Say(ctx, req.From, req.Text); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to send message: " + err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "room.message.sent", map[string]any{
		"room_id": id,
		"from":    req.From,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message": "Message sent successfully",
		},
	})
}

// GetMembers retrieves room members
func (h *RoomHandler) GetMembers(c *gin.Context) {
	id := c.Param("id")

	// Get room from memory
	room, exists := h.rooms[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_found",
				"message": "Room not found",
			},
		})
		return
	}

	members := room.GetMembers()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"members": members,
			"count":   len(members),
		},
	})
}

// GetHistory retrieves room message history
func (h *RoomHandler) GetHistory(c *gin.Context) {
	id := c.Param("id")

	// Get room from memory
	room, exists := h.rooms[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_found",
				"message": "Room not found",
			},
		})
		return
	}

	history := room.GetHistory()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"history": history,
			"count":   len(history),
		},
	})
}
