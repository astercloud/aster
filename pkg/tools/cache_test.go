package tools

import (
	"context"
	"os"
	"testing"
	"time"
)

// MockTool 模拟工具
type MockTool struct {
	name        string
	description string
	executeFunc func(context.Context, map[string]any, *ToolContext) (any, error)
	callCount   int
}

func (m *MockTool) Name() string {
	return m.name
}

func (m *MockTool) Description() string {
	return m.description
}

func (m *MockTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"input": map[string]any{
				"type": "string",
			},
		},
	}
}

func (m *MockTool) Prompt() string {
	return ""
}

func (m *MockTool) Execute(ctx context.Context, input map[string]any, tc *ToolContext) (any, error) {
	m.callCount++
	if m.executeFunc != nil {
		return m.executeFunc(ctx, input, tc)
	}
	return map[string]any{
		"result": "mock result",
		"input":  input,
	}, nil
}

func TestToolCache_MemoryCache(t *testing.T) {
	config := &CacheConfig{
		Enabled:        true,
		Strategy:       CacheStrategyMemory,
		TTL:            1 * time.Second,
		MaxMemoryItems: 10,
	}

	cache := NewToolCache(config)

	// 测试设置和获取
	ctx := context.Background()
	key := "test_key"
	value := map[string]any{"data": "test"}

	err := cache.Set(ctx, key, value, config.TTL)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// 获取缓存
	cached, ok := cache.Get(ctx, key)
	if !ok {
		t.Fatal("Expected cache hit")
	}

	cachedMap, ok := cached.(map[string]any)
	if !ok {
		t.Fatal("Expected map[string]any")
	}

	if cachedMap["data"] != "test" {
		t.Errorf("Expected 'test', got: %v", cachedMap["data"])
	}

	// 检查统计
	stats := cache.GetStats()
	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got: %d", stats.Hits)
	}

	if stats.Sets != 1 {
		t.Errorf("Expected 1 set, got: %d", stats.Sets)
	}
}

func TestToolCache_Expiration(t *testing.T) {
	config := &CacheConfig{
		Enabled:  true,
		Strategy: CacheStrategyMemory,
		TTL:      100 * time.Millisecond,
	}

	cache := NewToolCache(config)
	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	// 设置缓存
	err := cache.Set(ctx, key, value, config.TTL)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// 立即获取应该成功
	if _, ok := cache.Get(ctx, key); !ok {
		t.Fatal("Expected cache hit")
	}

	// 等待过期
	time.Sleep(150 * time.Millisecond)

	// 获取应该失败
	if _, ok := cache.Get(ctx, key); ok {
		t.Fatal("Expected cache miss after expiration")
	}

	// 检查统计
	stats := cache.GetStats()
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got: %d", stats.Misses)
	}
}

func TestToolCache_FileCache(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	config := &CacheConfig{
		Enabled:  true,
		Strategy: CacheStrategyFile,
		TTL:      1 * time.Hour,
		CacheDir: tmpDir,
	}

	cache := NewToolCache(config)
	ctx := context.Background()
	key := "test_key"
	value := map[string]any{"data": "test"}

	// 设置缓存
	err := cache.Set(ctx, key, value, config.TTL)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// 获取缓存
	cached, ok := cache.Get(ctx, key)
	if !ok {
		t.Fatal("Expected cache hit")
	}

	cachedMap, ok := cached.(map[string]any)
	if !ok {
		t.Fatal("Expected map[string]any")
	}

	if cachedMap["data"] != "test" {
		t.Errorf("Expected 'test', got: %v", cachedMap["data"])
	}

	// 验证文件存在
	filePath := cache.getCacheFilePath(key)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("Cache file should exist")
	}
}

func TestToolCache_BothStrategy(t *testing.T) {
	tmpDir := t.TempDir()

	config := &CacheConfig{
		Enabled:  true,
		Strategy: CacheStrategyBoth,
		TTL:      1 * time.Hour,
		CacheDir: tmpDir,
	}

	cache := NewToolCache(config)
	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	// 设置缓存
	err := cache.Set(ctx, key, value, config.TTL)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// 从内存获取
	if _, ok := cache.getFromMemory(key); !ok {
		t.Fatal("Expected memory cache hit")
	}

	// 从文件获取
	if _, ok := cache.getFromFile(key); !ok {
		t.Fatal("Expected file cache hit")
	}

	// 清空内存缓存
	cache.memoryMu.Lock()
	cache.memoryCache = make(map[string]*CacheEntry)
	cache.memoryMu.Unlock()

	// 应该从文件加载到内存
	cached, ok := cache.Get(ctx, key)
	if !ok {
		t.Fatal("Expected cache hit from file")
	}

	if cached != value {
		t.Errorf("Expected '%s', got: %v", value, cached)
	}

	// 验证已加载到内存
	if _, ok := cache.getFromMemory(key); !ok {
		t.Fatal("Expected memory cache hit after loading from file")
	}
}

func TestToolCache_MaxMemoryItems(t *testing.T) {
	config := &CacheConfig{
		Enabled:        true,
		Strategy:       CacheStrategyMemory,
		TTL:            1 * time.Hour,
		MaxMemoryItems: 3,
	}

	cache := NewToolCache(config)
	ctx := context.Background()

	// 添加4个条目，应该驱逐最旧的
	for i := range 4 {
		key := cache.GenerateKey("tool", map[string]any{"index": i})
		err := cache.Set(ctx, key, i, config.TTL)
		if err != nil {
			t.Fatalf("Failed to set cache: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // 确保时间戳不同
	}

	// 检查统计
	stats := cache.GetStats()
	if stats.Evictions != 1 {
		t.Errorf("Expected 1 eviction, got: %d", stats.Evictions)
	}

	// 第一个条目应该被驱逐
	firstKey := cache.GenerateKey("tool", map[string]any{"index": 0})
	if _, ok := cache.Get(ctx, firstKey); ok {
		t.Fatal("Expected first entry to be evicted")
	}

	// 最后一个条目应该存在
	lastKey := cache.GenerateKey("tool", map[string]any{"index": 3})
	if _, ok := cache.Get(ctx, lastKey); !ok {
		t.Fatal("Expected last entry to exist")
	}
}

func TestToolCache_Delete(t *testing.T) {
	config := &CacheConfig{
		Enabled:  true,
		Strategy: CacheStrategyMemory,
		TTL:      1 * time.Hour,
	}

	cache := NewToolCache(config)
	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	// 设置缓存
	err := cache.Set(ctx, key, value, config.TTL)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// 验证存在
	if _, ok := cache.Get(ctx, key); !ok {
		t.Fatal("Expected cache hit")
	}

	// 删除
	err = cache.Delete(key)
	if err != nil {
		t.Fatalf("Failed to delete cache: %v", err)
	}

	// 验证已删除
	if _, ok := cache.Get(ctx, key); ok {
		t.Fatal("Expected cache miss after deletion")
	}
}

func TestToolCache_Clear(t *testing.T) {
	config := &CacheConfig{
		Enabled:  true,
		Strategy: CacheStrategyMemory,
		TTL:      1 * time.Hour,
	}

	cache := NewToolCache(config)
	ctx := context.Background()

	// 添加多个条目
	for i := range 5 {
		key := cache.GenerateKey("tool", map[string]any{"index": i})
		err := cache.Set(ctx, key, i, config.TTL)
		if err != nil {
			t.Fatalf("Failed to set cache: %v", err)
		}
	}

	// 验证统计
	stats := cache.GetStats()
	if stats.ItemCount != 5 {
		t.Errorf("Expected 5 items, got: %d", stats.ItemCount)
	}

	// 清空
	err := cache.Clear()
	if err != nil {
		t.Fatalf("Failed to clear cache: %v", err)
	}

	// 验证已清空
	stats = cache.GetStats()
	if stats.ItemCount != 0 {
		t.Errorf("Expected 0 items after clear, got: %d", stats.ItemCount)
	}
}

func TestToolCache_GenerateKey(t *testing.T) {
	cache := NewToolCache(DefaultCacheConfig())

	// 相同输入应该生成相同的键
	input1 := map[string]any{"a": 1, "b": "test"}
	input2 := map[string]any{"a": 1, "b": "test"}

	key1 := cache.GenerateKey("tool1", input1)
	key2 := cache.GenerateKey("tool1", input2)

	if key1 != key2 {
		t.Errorf("Expected same key for same input, got: %s != %s", key1, key2)
	}

	// 不同输入应该生成不同的键
	input3 := map[string]any{"a": 2, "b": "test"}
	key3 := cache.GenerateKey("tool1", input3)

	if key1 == key3 {
		t.Error("Expected different key for different input")
	}

	// 不同工具名应该生成不同的键
	key4 := cache.GenerateKey("tool2", input1)

	if key1 == key4 {
		t.Error("Expected different key for different tool name")
	}
}

func TestCachedTool_Execute(t *testing.T) {
	config := &CacheConfig{
		Enabled:  true,
		Strategy: CacheStrategyMemory,
		TTL:      1 * time.Hour,
	}

	cache := NewToolCache(config)

	// 创建 mock 工具
	mockTool := &MockTool{
		name:        "test_tool",
		description: "Test tool",
	}

	// 创建带缓存的工具
	cachedTool := NewCachedTool(mockTool, cache)

	ctx := context.Background()
	input := map[string]any{"test": "value"}

	// 第一次执行
	result1, err := cachedTool.Execute(ctx, input, nil)
	if err != nil {
		t.Fatalf("Failed to execute: %v", err)
	}

	if mockTool.callCount != 1 {
		t.Errorf("Expected 1 call, got: %d", mockTool.callCount)
	}

	// 第二次执行（应该使用缓存）
	result2, err := cachedTool.Execute(ctx, input, nil)
	if err != nil {
		t.Fatalf("Failed to execute: %v", err)
	}

	if mockTool.callCount != 1 {
		t.Errorf("Expected 1 call (cached), got: %d", mockTool.callCount)
	}

	// 验证结果相同
	result1Map := result1.(map[string]any)
	result2Map := result2.(map[string]any)

	if result1Map["result"] != result2Map["result"] {
		t.Error("Expected same result from cache")
	}

	// 检查缓存统计
	stats := cache.GetStats()
	if stats.Hits != 1 {
		t.Errorf("Expected 1 cache hit, got: %d", stats.Hits)
	}
}

func TestCachedTool_DifferentInputs(t *testing.T) {
	config := &CacheConfig{
		Enabled:  true,
		Strategy: CacheStrategyMemory,
		TTL:      1 * time.Hour,
	}

	cache := NewToolCache(config)

	mockTool := &MockTool{
		name:        "test_tool",
		description: "Test tool",
	}

	cachedTool := NewCachedTool(mockTool, cache)

	ctx := context.Background()

	// 不同的输入应该执行不同的调用
	input1 := map[string]any{"test": "value1"}
	input2 := map[string]any{"test": "value2"}

	_, err := cachedTool.Execute(ctx, input1, nil)
	if err != nil {
		t.Fatalf("Failed to execute: %v", err)
	}

	_, err = cachedTool.Execute(ctx, input2, nil)
	if err != nil {
		t.Fatalf("Failed to execute: %v", err)
	}

	if mockTool.callCount != 2 {
		t.Errorf("Expected 2 calls for different inputs, got: %d", mockTool.callCount)
	}

	// 相同输入应该使用缓存
	_, err = cachedTool.Execute(ctx, input1, nil)
	if err != nil {
		t.Fatalf("Failed to execute: %v", err)
	}

	if mockTool.callCount != 2 {
		t.Errorf("Expected 2 calls (third was cached), got: %d", mockTool.callCount)
	}
}

func TestToolCache_Disabled(t *testing.T) {
	config := &CacheConfig{
		Enabled:  false,
		Strategy: CacheStrategyMemory,
		TTL:      1 * time.Hour,
	}

	cache := NewToolCache(config)
	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	// 设置缓存（应该被忽略）
	err := cache.Set(ctx, key, value, config.TTL)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// 获取缓存（应该失败）
	if _, ok := cache.Get(ctx, key); ok {
		t.Fatal("Expected cache miss when disabled")
	}

	// 统计应该为0
	stats := cache.GetStats()
	if stats.Sets != 0 {
		t.Errorf("Expected 0 sets when disabled, got: %d", stats.Sets)
	}
}

func TestToolCache_Cleanup(t *testing.T) {
	config := &CacheConfig{
		Enabled:  true,
		Strategy: CacheStrategyMemory,
		TTL:      100 * time.Millisecond,
	}

	cache := NewToolCache(config)
	ctx := context.Background()

	// 添加多个条目
	for i := range 5 {
		key := cache.GenerateKey("tool", map[string]any{"index": i})
		err := cache.Set(ctx, key, i, config.TTL)
		if err != nil {
			t.Fatalf("Failed to set cache: %v", err)
		}
	}

	// 验证条目存在
	stats := cache.GetStats()
	if stats.ItemCount != 5 {
		t.Errorf("Expected 5 items, got: %d", stats.ItemCount)
	}

	// 等待过期
	time.Sleep(150 * time.Millisecond)

	// 手动触发清理
	cache.cleanup()

	// 验证已清理
	stats = cache.GetStats()
	if stats.ItemCount != 0 {
		t.Errorf("Expected 0 items after cleanup, got: %d", stats.ItemCount)
	}

	if stats.Evictions != 5 {
		t.Errorf("Expected 5 evictions, got: %d", stats.Evictions)
	}
}
