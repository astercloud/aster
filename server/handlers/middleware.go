package handlers

import (
	"time"

	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MiddlewareRecord 中间件记录
type MiddlewareRecord struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Type        string         `json:"type"` // builtin, custom
	Description string         `json:"description,omitempty"`
	Config      map[string]any `json:"config,omitempty"`
	Enabled     bool           `json:"enabled"`
	Priority    int            `json:"priority"` // 执行顺序
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// MiddlewareHandler 中间件处理器
type MiddlewareHandler struct {
	store store.Store
}

// NewMiddlewareHandler 创建中间件处理器
func NewMiddlewareHandler(st store.Store) *MiddlewareHandler {
	return &MiddlewareHandler{store: st}
}

// Create 创建中间件
func (h *MiddlewareHandler) Create(c *gin.Context) {
	var req struct {
		Name        string         `json:"name" binding:"required"`
		Type        string         `json:"type"`
		Description string         `json:"description"`
		Config      map[string]any `json:"config"`
		Priority    int            `json:"priority"`
		Metadata    map[string]any `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "bad_request", "message": err.Error()}})
		return
	}

	ctx := c.Request.Context()
	mw := &MiddlewareRecord{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
		Config:      req.Config,
		Enabled:     true,
		Priority:    req.Priority,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    req.Metadata,
	}

	if err := h.store.Set(ctx, "middlewares", mw.ID, mw); err != nil {
		c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
		return
	}

	logging.Info(ctx, "middleware.created", map[string]any{"id": mw.ID, "name": req.Name})
	c.JSON(201, gin.H{"success": true, "data": mw})
}

// List 列出中间件
func (h *MiddlewareHandler) List(c *gin.Context) {
	ctx := c.Request.Context()
	records, err := h.store.List(ctx, "middlewares")
	if err != nil {
		c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
		return
	}

	middlewares := make([]*MiddlewareRecord, 0)
	for _, record := range records {
		var mw MiddlewareRecord
		if err := store.DecodeValue(record, &mw); err != nil {
			continue
		}
		middlewares = append(middlewares, &mw)
	}

	c.JSON(200, gin.H{"success": true, "data": middlewares})
}

// Get 获取中间件
func (h *MiddlewareHandler) Get(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var mw MiddlewareRecord
	if err := h.store.Get(ctx, "middlewares", id, &mw); err != nil {
		if err == store.ErrNotFound {
			c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "middleware not found"}})
			return
		}
		c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
		return
	}

	c.JSON(200, gin.H{"success": true, "data": &mw})
}

// Update 更新中间件
func (h *MiddlewareHandler) Update(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var req struct {
		Name        *string        `json:"name"`
		Description *string        `json:"description"`
		Config      map[string]any `json:"config"`
		Priority    *int           `json:"priority"`
		Metadata    map[string]any `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "bad_request", "message": err.Error()}})
		return
	}

	var mw MiddlewareRecord
	if err := h.store.Get(ctx, "middlewares", id, &mw); err != nil {
		if err == store.ErrNotFound {
			c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "middleware not found"}})
			return
		}
		c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
		return
	}

	if req.Name != nil {
		mw.Name = *req.Name
	}
	if req.Description != nil {
		mw.Description = *req.Description
	}
	if req.Config != nil {
		for k, v := range req.Config {
			if mw.Config == nil {
				mw.Config = make(map[string]any)
			}
			mw.Config[k] = v
		}
	}
	if req.Priority != nil {
		mw.Priority = *req.Priority
	}
	mw.UpdatedAt = time.Now()

	if err := h.store.Set(ctx, "middlewares", id, &mw); err != nil {
		c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
		return
	}

	logging.Info(ctx, "middleware.updated", map[string]any{"id": id})
	c.JSON(200, gin.H{"success": true, "data": &mw})
}

// Delete 删除中间件
func (h *MiddlewareHandler) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	if err := h.store.Delete(ctx, "middlewares", id); err != nil {
		if err == store.ErrNotFound {
			c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "middleware not found"}})
			return
		}
		c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
		return
	}

	logging.Info(ctx, "middleware.deleted", map[string]any{"id": id})
	c.Status(204)
}

// Enable 启用中间件
func (h *MiddlewareHandler) Enable(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var mw MiddlewareRecord
	if err := h.store.Get(ctx, "middlewares", id, &mw); err != nil {
		if err == store.ErrNotFound {
			c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "middleware not found"}})
			return
		}
		c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
		return
	}

	mw.Enabled = true
	mw.UpdatedAt = time.Now()

	if err := h.store.Set(ctx, "middlewares", id, &mw); err != nil {
		c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
		return
	}

	logging.Info(ctx, "middleware.enabled", map[string]any{"id": id})
	c.JSON(200, gin.H{"success": true, "data": &mw})
}

// Disable 禁用中间件
func (h *MiddlewareHandler) Disable(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var mw MiddlewareRecord
	if err := h.store.Get(ctx, "middlewares", id, &mw); err != nil {
		if err == store.ErrNotFound {
			c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "middleware not found"}})
			return
		}
		c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
		return
	}

	mw.Enabled = false
	mw.UpdatedAt = time.Now()

	if err := h.store.Set(ctx, "middlewares", id, &mw); err != nil {
		c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
		return
	}

	logging.Info(ctx, "middleware.disabled", map[string]any{"id": id})
	c.JSON(200, gin.H{"success": true, "data": &mw})
}

// Reload 重新加载中间件
func (h *MiddlewareHandler) Reload(c *gin.Context) {
	id := c.Param("id")
	logging.Info(c.Request.Context(), "middleware.reloaded", map[string]any{"id": id})
	c.JSON(200, gin.H{"success": true, "data": gin.H{"reloaded": true, "middleware_id": id}})
}

// GetStats 获取统计
func (h *MiddlewareHandler) GetStats(c *gin.Context) {
	id := c.Param("id")
	c.JSON(200, gin.H{"success": true, "data": gin.H{
		"middleware_id": id,
		"calls":         0,
		"errors":        0,
		"avg_time_ms":   0.0,
	}})
}

// ListRegistry 列出注册表
func (h *MiddlewareHandler) ListRegistry(c *gin.Context) {
	c.JSON(200, gin.H{"success": true, "data": []gin.H{
		{"id": "logging", "name": "Logging Middleware", "builtin": true},
		{"id": "auth", "name": "Auth Middleware", "builtin": true},
		{"id": "ratelimit", "name": "Rate Limit Middleware", "builtin": false},
	}})
}

// Install 安装中间件
func (h *MiddlewareHandler) Install(c *gin.Context) {
	id := c.Param("id")
	logging.Info(c.Request.Context(), "middleware.installed", map[string]any{"id": id})
	c.JSON(200, gin.H{"success": true, "data": gin.H{"installed": true, "middleware_id": id}})
}

// Uninstall 卸载中间件
func (h *MiddlewareHandler) Uninstall(c *gin.Context) {
	id := c.Param("id")
	logging.Info(c.Request.Context(), "middleware.uninstalled", map[string]any{"id": id})
	c.JSON(200, gin.H{"success": true, "data": gin.H{"uninstalled": true, "middleware_id": id}})
}

// GetInfo 获取信息
func (h *MiddlewareHandler) GetInfo(c *gin.Context) {
	id := c.Param("id")
	c.JSON(200, gin.H{"success": true, "data": gin.H{
		"id":      id,
		"name":    id + " Middleware",
		"version": "1.0.0",
		"builtin": false,
	}})
}

// ReloadAll 重新加载所有
func (h *MiddlewareHandler) ReloadAll(c *gin.Context) {
	logging.Info(c.Request.Context(), "middleware.all.reloaded", nil)
	c.JSON(200, gin.H{"success": true, "data": gin.H{"reloaded": true, "count": 0}})
}
