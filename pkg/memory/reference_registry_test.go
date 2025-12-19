package memory

import (
	"context"
	"testing"
	"time"
)

func TestInMemoryReferenceRegistry_Register(t *testing.T) {
	registry := NewInMemoryReferenceRegistry(100)
	ctx := context.Background()

	ref := Reference{
		Type:  ReferenceTypeFilePath,
		Value: "/path/to/file.go",
	}

	err := registry.Register(ctx, ref, "test context")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// 查找注册的引用
	info, err := registry.Lookup(ctx, ref.Type, ref.Value)
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if info == nil {
		t.Fatal("Expected to find registered reference")
	}
	if info.Reference.Value != ref.Value {
		t.Errorf("Expected value %s, got %s", ref.Value, info.Reference.Value)
	}
	if info.AccessCount != 1 {
		t.Errorf("Expected access count 1, got %d", info.AccessCount)
	}
}

func TestInMemoryReferenceRegistry_RegisterDuplicate(t *testing.T) {
	registry := NewInMemoryReferenceRegistry(100)
	ctx := context.Background()

	ref := Reference{
		Type:  ReferenceTypeFilePath,
		Value: "/path/to/file.go",
	}

	// 注册两次
	_ = registry.Register(ctx, ref, "first")
	_ = registry.Register(ctx, ref, "second")

	info, _ := registry.Lookup(ctx, ref.Type, ref.Value)
	if info.AccessCount != 2 {
		t.Errorf("Expected access count 2 after duplicate, got %d", info.AccessCount)
	}
	// 源上下文应该保持第一个
	if info.SourceContext != "first" {
		t.Errorf("Expected source context 'first', got '%s'", info.SourceContext)
	}
}

func TestInMemoryReferenceRegistry_ListByType(t *testing.T) {
	registry := NewInMemoryReferenceRegistry(100)
	ctx := context.Background()

	// 注册不同类型的引用
	refs := []Reference{
		{Type: ReferenceTypeFilePath, Value: "/file1.go"},
		{Type: ReferenceTypeFilePath, Value: "/file2.go"},
		{Type: ReferenceTypeURL, Value: "https://example.com"},
		{Type: ReferenceTypeFunction, Value: "main"},
	}

	for _, ref := range refs {
		_ = registry.Register(ctx, ref, "")
	}

	// 按类型列出
	filePaths, err := registry.ListByType(ctx, ReferenceTypeFilePath, 0)
	if err != nil {
		t.Fatalf("ListByType failed: %v", err)
	}
	if len(filePaths) != 2 {
		t.Errorf("Expected 2 file paths, got %d", len(filePaths))
	}

	urls, _ := registry.ListByType(ctx, ReferenceTypeURL, 0)
	if len(urls) != 1 {
		t.Errorf("Expected 1 URL, got %d", len(urls))
	}
}

func TestInMemoryReferenceRegistry_ListRecent(t *testing.T) {
	registry := NewInMemoryReferenceRegistry(100)
	ctx := context.Background()

	// 注册多个引用
	for i := range 10 {
		ref := Reference{
			Type:  ReferenceTypeFilePath,
			Value: "/file" + string(rune('0'+i)) + ".go",
		}
		_ = registry.Register(ctx, ref, "")
		time.Sleep(1 * time.Millisecond) // 确保时间戳不同
	}

	// 获取最近 5 个
	recent, err := registry.ListRecent(ctx, 5)
	if err != nil {
		t.Fatalf("ListRecent failed: %v", err)
	}
	if len(recent) != 5 {
		t.Errorf("Expected 5 recent refs, got %d", len(recent))
	}
}

func TestInMemoryReferenceRegistry_MarkAccessed(t *testing.T) {
	registry := NewInMemoryReferenceRegistry(100)
	ctx := context.Background()

	ref := Reference{
		Type:  ReferenceTypeFilePath,
		Value: "/path/to/file.go",
	}

	_ = registry.Register(ctx, ref, "")
	_ = registry.MarkAccessed(ctx, ref.Type, ref.Value)
	_ = registry.MarkAccessed(ctx, ref.Type, ref.Value)

	info, _ := registry.Lookup(ctx, ref.Type, ref.Value)
	if info.AccessCount != 3 { // 1 from register + 2 from mark
		t.Errorf("Expected access count 3, got %d", info.AccessCount)
	}
}

func TestInMemoryReferenceRegistry_GetStats(t *testing.T) {
	registry := NewInMemoryReferenceRegistry(100)
	ctx := context.Background()

	refs := []Reference{
		{Type: ReferenceTypeFilePath, Value: "/file1.go"},
		{Type: ReferenceTypeFilePath, Value: "/file2.go"},
		{Type: ReferenceTypeURL, Value: "https://example.com"},
	}

	for _, ref := range refs {
		_ = registry.Register(ctx, ref, "")
	}

	stats, err := registry.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.TotalReferences != 3 {
		t.Errorf("Expected 3 total references, got %d", stats.TotalReferences)
	}
	if stats.ByType[ReferenceTypeFilePath] != 2 {
		t.Errorf("Expected 2 file paths, got %d", stats.ByType[ReferenceTypeFilePath])
	}
	if stats.ByType[ReferenceTypeURL] != 1 {
		t.Errorf("Expected 1 URL, got %d", stats.ByType[ReferenceTypeURL])
	}
}

func TestInMemoryReferenceRegistry_Cleanup(t *testing.T) {
	registry := NewInMemoryReferenceRegistry(100)
	ctx := context.Background()

	// 注册引用
	ref := Reference{
		Type:  ReferenceTypeFilePath,
		Value: "/old/file.go",
	}
	_ = registry.Register(ctx, ref, "")

	// 立即清理（0 duration 应该清理所有）
	// 但因为刚注册，LastAccessed 是现在，所以不会被清理
	removed, err := registry.Cleanup(ctx, 1*time.Hour)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}
	if removed != 0 {
		t.Errorf("Expected 0 removed (fresh reference), got %d", removed)
	}

	// 验证引用仍存在
	info, _ := registry.Lookup(ctx, ref.Type, ref.Value)
	if info == nil {
		t.Error("Reference should still exist after cleanup")
	}
}

func TestInMemoryReferenceRegistry_MaxSize(t *testing.T) {
	maxSize := 5
	registry := NewInMemoryReferenceRegistry(maxSize)
	ctx := context.Background()

	// 注册超过最大数量的引用
	for i := range 10 {
		ref := Reference{
			Type:  ReferenceTypeFilePath,
			Value: "/file" + string(rune('0'+i)) + ".go",
		}
		_ = registry.Register(ctx, ref, "")
	}

	stats, _ := registry.GetStats(ctx)
	if stats.TotalReferences > maxSize {
		t.Errorf("Expected at most %d references, got %d", maxSize, stats.TotalReferences)
	}
}

func TestReferenceRegistryMiddleware_ProcessToolResult(t *testing.T) {
	registry := NewInMemoryReferenceRegistry(100)
	middleware := NewReferenceRegistryMiddleware(registry)
	ctx := context.Background()

	content := `
Reading file /Users/test/project/main.go
Found function definition at line 10
URL: https://example.com/docs
	`

	err := middleware.ProcessToolResult(ctx, "Read", content)
	if err != nil {
		t.Fatalf("ProcessToolResult failed: %v", err)
	}

	stats, _ := registry.GetStats(ctx)
	if stats.TotalReferences == 0 {
		t.Error("Expected some references to be extracted")
	}
}

func TestMakeKey(t *testing.T) {
	key := makeKey("file_path", "/test/file.go")
	expected := "file_path:/test/file.go"
	if key != expected {
		t.Errorf("Expected key '%s', got '%s'", expected, key)
	}
}
