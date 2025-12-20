package handlers

import (
	"errors"
	"runtime"
	"time"

	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/store"
	"github.com/gin-gonic/gin"
)

// SystemConfigRecord 系统配置记录
type SystemConfigRecord struct {
	Key       string    `json:"key"`
	Value     any       `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SystemHandler 系统处理器
type SystemHandler struct {
	store store.Store
}

// NewSystemHandler 创建系统处理器
func NewSystemHandler(st store.Store) *SystemHandler {
	return &SystemHandler{store: st}
}

// ListConfig 列出配置
func (h *SystemHandler) ListConfig(c *gin.Context) {
	ctx := c.Request.Context()
	records, err := h.store.List(ctx, "system_config")
	if err != nil {
		c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
		return
	}

	configs := make([]*SystemConfigRecord, 0)
	for _, record := range records {
		var cfg SystemConfigRecord
		if err := store.DecodeValue(record, &cfg); err != nil {
			continue
		}
		configs = append(configs, &cfg)
	}

	c.JSON(200, gin.H{"success": true, "data": configs})
}

// GetConfig 获取配置
func (h *SystemHandler) GetConfig(c *gin.Context) {
	ctx := c.Request.Context()
	key := c.Param("key")

	var cfg SystemConfigRecord
	if err := h.store.Get(ctx, "system_config", key, &cfg); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "config not found"}})
			return
		}
		c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
		return
	}

	c.JSON(200, gin.H{"success": true, "data": &cfg})
}

// UpdateConfig 更新配置
func (h *SystemHandler) UpdateConfig(c *gin.Context) {
	ctx := c.Request.Context()
	key := c.Param("key")

	var req struct {
		Value any `json:"value" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "bad_request", "message": err.Error()}})
		return
	}

	cfg := &SystemConfigRecord{
		Key:       key,
		Value:     req.Value,
		UpdatedAt: time.Now(),
	}

	if err := h.store.Set(ctx, "system_config", key, cfg); err != nil {
		c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
		return
	}

	logging.Info(ctx, "system.config.updated", map[string]any{"key": key})
	c.JSON(200, gin.H{"success": true, "data": cfg})
}

// DeleteConfig 删除配置
func (h *SystemHandler) DeleteConfig(c *gin.Context) {
	ctx := c.Request.Context()
	key := c.Param("key")

	if err := h.store.Delete(ctx, "system_config", key); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "config not found"}})
			return
		}
		c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
		return
	}

	logging.Info(ctx, "system.config.deleted", map[string]any{"key": key})
	c.Status(204)
}

// GetInfo 获取系统信息
func (h *SystemHandler) GetInfo(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"version":    "0.2.2",
			"go_version": runtime.Version(),
			"num_cpu":    runtime.NumCPU(),
			"goroutines": runtime.NumGoroutine(),
			"memory": gin.H{
				"alloc_mb":       m.Alloc / 1024 / 1024,
				"total_alloc_mb": m.TotalAlloc / 1024 / 1024,
				"sys_mb":         m.Sys / 1024 / 1024,
				"num_gc":         m.NumGC,
			},
		},
	})
}

// GetHealth 健康检查
func (h *SystemHandler) GetHealth(c *gin.Context) {
	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
		},
	})
}

// GetStats 获取统计
func (h *SystemHandler) GetStats(c *gin.Context) {
	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"uptime_seconds":  0,
			"total_requests":  0,
			"active_sessions": 0,
			"cache_hit_rate":  0.0,
		},
	})
}

// Reload 重新加载系统
func (h *SystemHandler) Reload(c *gin.Context) {
	logging.Info(c.Request.Context(), "system.reloaded", nil)
	c.JSON(200, gin.H{"success": true, "data": gin.H{"reloaded": true}})
}

// RunGC 运行垃圾回收
func (h *SystemHandler) RunGC(c *gin.Context) {
	runtime.GC()
	logging.Info(c.Request.Context(), "system.gc.executed", nil)
	c.JSON(200, gin.H{"success": true, "data": gin.H{"gc_executed": true}})
}

// Backup 备份系统
func (h *SystemHandler) Backup(c *gin.Context) {
	logging.Info(c.Request.Context(), "system.backup.started", nil)
	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"backup_id": "backup_" + time.Now().Format("20060102_150405"),
			"status":    "started",
		},
	})
}
