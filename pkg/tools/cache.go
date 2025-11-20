package tools

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CacheStrategy 缓存策略
type CacheStrategy string

const (
	CacheStrategyMemory CacheStrategy = "memory" // 内存缓存
	CacheStrategyFile   CacheStrategy = "file"   // 文件缓存
	CacheStrategyBoth   CacheStrategy = "both"   // 内存+文件双层缓存
)

// CacheConfig 缓存配置
type CacheConfig struct {
	// Enabled 是否启用缓存
	Enabled bool

	// Strategy 缓存策略
	Strategy CacheStrategy

	// TTL 缓存过期时间
	TTL time.Duration

	// CacheDir 文件缓存目录（仅用于 file 和 both 策略）
	CacheDir string

	// MaxMemoryItems 内存缓存最大条目数（0 表示无限制）
	MaxMemoryItems int

	// MaxFileSize 单个缓存文件最大大小（字节，0 表示无限制）
	MaxFileSize int64
}

// DefaultCacheConfig 默认缓存配置
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		Enabled:        false,
		Strategy:       CacheStrategyMemory,
		TTL:            1 * time.Hour,
		CacheDir:       ".cache/tools",
		MaxMemoryItems: 1000,
		MaxFileSize:    10 * 1024 * 1024, // 10MB
	}
}

// CacheEntry 缓存条目
type CacheEntry struct {
	Key       string
	Value     interface{}
	CreatedAt time.Time
	ExpiresAt time.Time
	Size      int64
}

// IsExpired 检查是否过期
func (e *CacheEntry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// ToolCache 工具缓存
type ToolCache struct {
	config *CacheConfig

	// 内存缓存
	memoryCache map[string]*CacheEntry
	memoryMu    sync.RWMutex

	// 统计信息
	stats *CacheStats
}

// CacheStats 缓存统计
type CacheStats struct {
	Hits          int64
	Misses        int64
	Sets          int64
	Evictions     int64
	Errors        int64
	TotalSize     int64
	ItemCount     int64
	LastCleanupAt time.Time
}

// NewToolCache 创建工具缓存
func NewToolCache(config *CacheConfig) *ToolCache {
	if config == nil {
		config = DefaultCacheConfig()
	}

	cache := &ToolCache{
		config:      config,
		memoryCache: make(map[string]*CacheEntry),
		stats: &CacheStats{
			LastCleanupAt: time.Now(),
		},
	}

	// 启动后台清理任务
	if config.Enabled {
		go cache.cleanupLoop()
	}

	return cache
}

// GenerateKey 生成缓存键
func (c *ToolCache) GenerateKey(toolName string, input map[string]interface{}) string {
	// 序列化输入参数
	data, err := json.Marshal(input)
	if err != nil {
		// 如果序列化失败，使用简单的字符串拼接
		return fmt.Sprintf("%s_%v", toolName, input)
	}

	// 计算 SHA256 哈希
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%s_%x", toolName, hash[:16]) // 使用前16字节
}

// Get 获取缓存
func (c *ToolCache) Get(ctx context.Context, key string) (interface{}, bool) {
	if !c.config.Enabled {
		return nil, false
	}

	// 先尝试内存缓存
	if c.config.Strategy == CacheStrategyMemory || c.config.Strategy == CacheStrategyBoth {
		if value, ok := c.getFromMemory(key); ok {
			c.stats.Hits++
			return value, true
		}
	}

	// 再尝试文件缓存
	if c.config.Strategy == CacheStrategyFile || c.config.Strategy == CacheStrategyBoth {
		if value, ok := c.getFromFile(key); ok {
			c.stats.Hits++

			// 如果是双层缓存，将文件缓存加载到内存
			if c.config.Strategy == CacheStrategyBoth {
				c.setToMemory(key, value, c.config.TTL)
			}

			return value, true
		}
	}

	c.stats.Misses++
	return nil, false
}

// Set 设置缓存
func (c *ToolCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if !c.config.Enabled {
		return nil
	}

	c.stats.Sets++

	// 设置到内存缓存
	if c.config.Strategy == CacheStrategyMemory || c.config.Strategy == CacheStrategyBoth {
		c.setToMemory(key, value, ttl)
	}

	// 设置到文件缓存
	if c.config.Strategy == CacheStrategyFile || c.config.Strategy == CacheStrategyBoth {
		if err := c.setToFile(key, value, ttl); err != nil {
			c.stats.Errors++
			return fmt.Errorf("failed to set file cache: %w", err)
		}
	}

	return nil
}

// Delete 删除缓存
func (c *ToolCache) Delete(key string) error {
	if !c.config.Enabled {
		return nil
	}

	// 从内存删除
	if c.config.Strategy == CacheStrategyMemory || c.config.Strategy == CacheStrategyBoth {
		c.deleteFromMemory(key)
	}

	// 从文件删除
	if c.config.Strategy == CacheStrategyFile || c.config.Strategy == CacheStrategyBoth {
		if err := c.deleteFromFile(key); err != nil {
			c.stats.Errors++
			return fmt.Errorf("failed to delete file cache: %w", err)
		}
	}

	return nil
}

// Clear 清空所有缓存
func (c *ToolCache) Clear() error {
	if !c.config.Enabled {
		return nil
	}

	// 清空内存缓存
	if c.config.Strategy == CacheStrategyMemory || c.config.Strategy == CacheStrategyBoth {
		c.memoryMu.Lock()
		c.memoryCache = make(map[string]*CacheEntry)
		c.stats.ItemCount = 0
		c.stats.TotalSize = 0
		c.memoryMu.Unlock()
	}

	// 清空文件缓存
	if c.config.Strategy == CacheStrategyFile || c.config.Strategy == CacheStrategyBoth {
		if err := os.RemoveAll(c.config.CacheDir); err != nil {
			c.stats.Errors++
			return fmt.Errorf("failed to clear file cache: %w", err)
		}
	}

	return nil
}

// GetStats 获取统计信息
func (c *ToolCache) GetStats() *CacheStats {
	return c.stats
}

// 内存缓存操作

func (c *ToolCache) getFromMemory(key string) (interface{}, bool) {
	c.memoryMu.RLock()
	defer c.memoryMu.RUnlock()

	entry, ok := c.memoryCache[key]
	if !ok {
		return nil, false
	}

	// 检查是否过期
	if entry.IsExpired() {
		return nil, false
	}

	return entry.Value, true
}

func (c *ToolCache) setToMemory(key string, value interface{}, ttl time.Duration) {
	c.memoryMu.Lock()
	defer c.memoryMu.Unlock()

	// 检查是否需要驱逐
	if c.config.MaxMemoryItems > 0 && len(c.memoryCache) >= c.config.MaxMemoryItems {
		c.evictOldest()
	}

	// 计算大小（粗略估计）
	size := int64(len(fmt.Sprintf("%v", value)))

	entry := &CacheEntry{
		Key:       key,
		Value:     value,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(ttl),
		Size:      size,
	}

	c.memoryCache[key] = entry
	c.stats.ItemCount++
	c.stats.TotalSize += size
}

func (c *ToolCache) deleteFromMemory(key string) {
	c.memoryMu.Lock()
	defer c.memoryMu.Unlock()

	if entry, ok := c.memoryCache[key]; ok {
		delete(c.memoryCache, key)
		c.stats.ItemCount--
		c.stats.TotalSize -= entry.Size
	}
}

func (c *ToolCache) evictOldest() {
	// 找到最旧的条目
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.memoryCache {
		if oldestKey == "" || entry.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.CreatedAt
		}
	}

	if oldestKey != "" {
		if entry, ok := c.memoryCache[oldestKey]; ok {
			delete(c.memoryCache, oldestKey)
			c.stats.ItemCount--
			c.stats.TotalSize -= entry.Size
			c.stats.Evictions++
		}
	}
}

// 文件缓存操作

func (c *ToolCache) getFromFile(key string) (interface{}, bool) {
	filePath := c.getCacheFilePath(key)

	// 读取文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, false
	}

	// 反序列化
	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		c.stats.Errors++
		return nil, false
	}

	// 检查是否过期
	if entry.IsExpired() {
		// 删除过期文件
		os.Remove(filePath)
		return nil, false
	}

	return entry.Value, true
}

func (c *ToolCache) setToFile(key string, value interface{}, ttl time.Duration) error {
	// 确保缓存目录存在
	if err := os.MkdirAll(c.config.CacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	entry := &CacheEntry{
		Key:       key,
		Value:     value,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(ttl),
	}

	// 序列化
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	// 检查文件大小限制
	if c.config.MaxFileSize > 0 && int64(len(data)) > c.config.MaxFileSize {
		return fmt.Errorf("cache entry too large: %d bytes (max: %d)", len(data), c.config.MaxFileSize)
	}

	// 写入文件
	filePath := c.getCacheFilePath(key)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

func (c *ToolCache) deleteFromFile(key string) error {
	filePath := c.getCacheFilePath(key)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete cache file: %w", err)
	}
	return nil
}

func (c *ToolCache) getCacheFilePath(key string) string {
	return filepath.Join(c.config.CacheDir, key+".json")
}

// 后台清理任务

func (c *ToolCache) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

func (c *ToolCache) cleanup() {
	c.stats.LastCleanupAt = time.Now()

	// 清理内存缓存
	if c.config.Strategy == CacheStrategyMemory || c.config.Strategy == CacheStrategyBoth {
		c.cleanupMemory()
	}

	// 清理文件缓存
	if c.config.Strategy == CacheStrategyFile || c.config.Strategy == CacheStrategyBoth {
		c.cleanupFiles()
	}
}

func (c *ToolCache) cleanupMemory() {
	c.memoryMu.Lock()
	defer c.memoryMu.Unlock()

	for key, entry := range c.memoryCache {
		if entry.IsExpired() {
			delete(c.memoryCache, key)
			c.stats.ItemCount--
			c.stats.TotalSize -= entry.Size
			c.stats.Evictions++
		}
	}
}

func (c *ToolCache) cleanupFiles() {
	if _, err := os.Stat(c.config.CacheDir); os.IsNotExist(err) {
		return
	}

	entries, err := os.ReadDir(c.config.CacheDir)
	if err != nil {
		c.stats.Errors++
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(c.config.CacheDir, entry.Name())

		// 读取并检查是否过期
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var cacheEntry CacheEntry
		if err := json.Unmarshal(data, &cacheEntry); err != nil {
			continue
		}

		if cacheEntry.IsExpired() {
			os.Remove(filePath)
			c.stats.Evictions++
		}
	}
}

// CachedTool 带缓存的工具包装器
type CachedTool struct {
	tool  Tool
	cache *ToolCache
}

// NewCachedTool 创建带缓存的工具
func NewCachedTool(tool Tool, cache *ToolCache) *CachedTool {
	return &CachedTool{
		tool:  tool,
		cache: cache,
	}
}

// Name 实现 Tool 接口
func (ct *CachedTool) Name() string {
	return ct.tool.Name()
}

// Description 实现 Tool 接口
func (ct *CachedTool) Description() string {
	return ct.tool.Description()
}

// InputSchema 实现 Tool 接口
func (ct *CachedTool) InputSchema() map[string]interface{} {
	return ct.tool.InputSchema()
}

// Prompt 实现 Tool 接口
func (ct *CachedTool) Prompt() string {
	return ct.tool.Prompt()
}

// Execute 实现 Tool 接口（带缓存）
func (ct *CachedTool) Execute(ctx context.Context, input map[string]interface{}, tc *ToolContext) (interface{}, error) {
	// 生成缓存键
	key := ct.cache.GenerateKey(ct.tool.Name(), input)

	// 尝试从缓存获取
	if cached, ok := ct.cache.Get(ctx, key); ok {
		return cached, nil
	}

	// 执行工具
	result, err := ct.tool.Execute(ctx, input, tc)
	if err != nil {
		return nil, err
	}

	// 缓存结果
	if err := ct.cache.Set(ctx, key, result, ct.cache.config.TTL); err != nil {
		// 缓存失败不影响结果返回，只记录错误
		_ = err // 忽略缓存错误
	}

	return result, nil
}
