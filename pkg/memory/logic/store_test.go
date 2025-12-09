package logic

import (
	"context"
	"testing"
	"time"

	"github.com/astercloud/aster/pkg/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryStore_BasicCRUD(t *testing.T) {
	store := NewInMemoryStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	namespace := "user:123"

	// 创建 Memory
	mem := &LogicMemory{
		ID:          "mem-1",
		Namespace:   namespace,
		Scope:       ScopeUser,
		Type:        "preference",
		Key:         "writing_tone",
		Value:       map[string]any{"tone": "casual"},
		Description: "用户偏好口语化表达",
		Provenance: &memory.MemoryProvenance{
			SourceType: memory.SourceUserInput,
			Confidence: 0.8,
		},
	}

	// Save
	err := store.Save(ctx, mem)
	require.NoError(t, err)

	// Get
	retrieved, err := store.Get(ctx, namespace, "writing_tone")
	require.NoError(t, err)
	assert.Equal(t, "mem-1", retrieved.ID)
	assert.Equal(t, namespace, retrieved.Namespace)
	assert.Equal(t, "preference", retrieved.Type)
	assert.Equal(t, "用户偏好口语化表达", retrieved.Description)
	assert.NotZero(t, retrieved.CreatedAt)
	assert.NotZero(t, retrieved.UpdatedAt)

	// Update
	mem.Description = "用户偏好简洁口语"
	mem.Provenance.Confidence = 0.9
	err = store.Save(ctx, mem)
	require.NoError(t, err)

	updated, err := store.Get(ctx, namespace, "writing_tone")
	require.NoError(t, err)
	assert.Equal(t, "用户偏好简洁口语", updated.Description)
	assert.Equal(t, 0.9, updated.Provenance.Confidence)

	// Delete
	err = store.Delete(ctx, namespace, "writing_tone")
	require.NoError(t, err)

	_, err = store.Get(ctx, namespace, "writing_tone")
	assert.ErrorIs(t, err, ErrMemoryNotFound)
}

func TestInMemoryStore_List(t *testing.T) {
	store := NewInMemoryStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	namespace := "user:123"

	// 创建多个 Memory
	memories := []*LogicMemory{
		{
			ID:        "mem-1",
			Namespace: namespace,
			Scope:     ScopeUser,
			Type:      "preference",
			Key:       "tone",
			Provenance: &memory.MemoryProvenance{
				Confidence: 0.9,
			},
		},
		{
			ID:        "mem-2",
			Namespace: namespace,
			Scope:     ScopeSession,
			Type:      "behavior",
			Key:       "edit_pattern",
			Provenance: &memory.MemoryProvenance{
				Confidence: 0.7,
			},
		},
		{
			ID:        "mem-3",
			Namespace: namespace,
			Scope:     ScopeUser,
			Type:      "preference",
			Key:       "style",
			Provenance: &memory.MemoryProvenance{
				Confidence: 0.5,
			},
		},
		{
			ID:        "mem-4",
			Namespace: "user:456", // 不同 namespace
			Scope:     ScopeUser,
			Type:      "preference",
			Key:       "tone",
			Provenance: &memory.MemoryProvenance{
				Confidence: 0.8,
			},
		},
	}

	for _, m := range memories {
		require.NoError(t, store.Save(ctx, m))
	}

	// List all in namespace
	result, err := store.List(ctx, namespace)
	require.NoError(t, err)
	assert.Len(t, result, 3)

	// Filter by type
	result, err = store.List(ctx, namespace, WithType("preference"))
	require.NoError(t, err)
	assert.Len(t, result, 2)

	// Filter by scope
	result, err = store.List(ctx, namespace, WithScope(ScopeSession))
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "edit_pattern", result[0].Key)

	// Filter by confidence
	result, err = store.List(ctx, namespace, WithMinConfidence(0.6))
	require.NoError(t, err)
	assert.Len(t, result, 2)

	// TopK with ordering
	result, err = store.List(ctx, namespace, WithTopK(2), WithOrderBy(OrderByConfidence))
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 0.9, result[0].Provenance.Confidence)
	assert.Equal(t, 0.7, result[1].Provenance.Confidence)
}

func TestInMemoryStore_SearchByType(t *testing.T) {
	store := NewInMemoryStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	namespace := "user:123"

	require.NoError(t, store.Save(ctx, &LogicMemory{
		Namespace: namespace,
		Type:      "preference",
		Key:       "key1",
	}))
	require.NoError(t, store.Save(ctx, &LogicMemory{
		Namespace: namespace,
		Type:      "behavior",
		Key:       "key2",
	}))

	result, err := store.SearchByType(ctx, namespace, "preference")
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "preference", result[0].Type)
}

func TestInMemoryStore_SearchByScope(t *testing.T) {
	store := NewInMemoryStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	namespace := "user:123"

	require.NoError(t, store.Save(ctx, &LogicMemory{
		Namespace: namespace,
		Scope:     ScopeUser,
		Key:       "key1",
	}))
	require.NoError(t, store.Save(ctx, &LogicMemory{
		Namespace: namespace,
		Scope:     ScopeGlobal,
		Key:       "key2",
	}))

	result, err := store.SearchByScope(ctx, namespace, ScopeGlobal)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, ScopeGlobal, result[0].Scope)
}

func TestInMemoryStore_GetTopK(t *testing.T) {
	store := NewInMemoryStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	namespace := "user:123"

	for i := 0; i < 5; i++ {
		require.NoError(t, store.Save(ctx, &LogicMemory{
			Namespace: namespace,
			Key:       string(rune('a' + i)),
			Provenance: &memory.MemoryProvenance{
				Confidence: float64(i+1) * 0.1,
			},
		}))
	}

	result, err := store.GetTopK(ctx, namespace, 3, OrderByConfidence)
	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.InDelta(t, 0.5, result[0].Provenance.Confidence, 0.0001)
	assert.InDelta(t, 0.4, result[1].Provenance.Confidence, 0.0001)
	assert.InDelta(t, 0.3, result[2].Provenance.Confidence, 0.0001)
}

func TestInMemoryStore_IncrementAccessCount(t *testing.T) {
	store := NewInMemoryStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	namespace := "user:123"

	require.NoError(t, store.Save(ctx, &LogicMemory{
		Namespace: namespace,
		Key:       "test",
	}))

	// 初始访问次数为 0
	mem, err := store.Get(ctx, namespace, "test")
	require.NoError(t, err)
	assert.Equal(t, 0, mem.AccessCount)

	// 增加访问次数
	require.NoError(t, store.IncrementAccessCount(ctx, namespace, "test"))
	require.NoError(t, store.IncrementAccessCount(ctx, namespace, "test"))

	mem, err = store.Get(ctx, namespace, "test")
	require.NoError(t, err)
	assert.Equal(t, 2, mem.AccessCount)
	assert.False(t, mem.LastAccessed.IsZero())
}

func TestInMemoryStore_GetStats(t *testing.T) {
	store := NewInMemoryStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	namespace := "user:123"

	require.NoError(t, store.Save(ctx, &LogicMemory{
		Namespace: namespace,
		Scope:     ScopeUser,
		Type:      "preference",
		Key:       "key1",
		Provenance: &memory.MemoryProvenance{
			Confidence: 0.8,
		},
	}))
	require.NoError(t, store.Save(ctx, &LogicMemory{
		Namespace: namespace,
		Scope:     ScopeSession,
		Type:      "preference",
		Key:       "key2",
		Provenance: &memory.MemoryProvenance{
			Confidence: 0.6,
		},
	}))
	require.NoError(t, store.Save(ctx, &LogicMemory{
		Namespace: namespace,
		Scope:     ScopeUser,
		Type:      "behavior",
		Key:       "key3",
		Provenance: &memory.MemoryProvenance{
			Confidence: 0.7,
		},
	}))

	stats, err := store.GetStats(ctx, namespace)
	require.NoError(t, err)
	assert.Equal(t, 3, stats.TotalCount)
	assert.Equal(t, 2, stats.CountByType["preference"])
	assert.Equal(t, 1, stats.CountByType["behavior"])
	assert.Equal(t, 2, stats.CountByScope[ScopeUser])
	assert.Equal(t, 1, stats.CountByScope[ScopeSession])
	assert.InDelta(t, 0.7, stats.AverageConfidence, 0.01)
}

func TestInMemoryStore_Prune(t *testing.T) {
	store := NewInMemoryStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	namespace := "user:123"

	// 创建一些 Memory
	oldTime := time.Now().Add(-time.Hour * 24 * 100) // 100 天前
	recentTime := time.Now().Add(-time.Hour)         // 1 小时前

	memories := []*LogicMemory{
		{
			Namespace:    namespace,
			Key:          "low_confidence",
			Provenance:   &memory.MemoryProvenance{Confidence: 0.1},
			LastAccessed: recentTime,
		},
		{
			Namespace:    namespace,
			Key:          "old_unused",
			AccessCount:  1,
			Provenance:   &memory.MemoryProvenance{Confidence: 0.8},
			LastAccessed: oldTime,
		},
		{
			Namespace:    namespace,
			Key:          "good",
			AccessCount:  10,
			Provenance:   &memory.MemoryProvenance{Confidence: 0.9},
			LastAccessed: recentTime,
		},
	}

	for _, m := range memories {
		require.NoError(t, store.Save(ctx, m))
		// 手动设置 LastAccessed（因为 Save 会更新它）
		store.mu.Lock()
		stored := store.memories[makeKey(m.Namespace, m.Key)]
		stored.LastAccessed = m.LastAccessed
		stored.AccessCount = m.AccessCount
		store.mu.Unlock()
	}

	// 清理低置信度
	pruned, err := store.Prune(ctx, PruneCriteria{
		MinConfidence: 0.2,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, pruned)

	// 验证 low_confidence 被删除
	_, err = store.Get(ctx, namespace, "low_confidence")
	assert.ErrorIs(t, err, ErrMemoryNotFound)

	// good 仍然存在
	_, err = store.Get(ctx, namespace, "good")
	require.NoError(t, err)
}

func TestInMemoryStore_Close(t *testing.T) {
	store := NewInMemoryStore()

	ctx := context.Background()
	require.NoError(t, store.Save(ctx, &LogicMemory{
		Namespace: "user:123",
		Key:       "test",
	}))

	// 关闭存储
	require.NoError(t, store.Close())

	// 所有操作应该返回 ErrStoreClosed
	_, err := store.Get(ctx, "user:123", "test")
	assert.ErrorIs(t, err, ErrStoreClosed)

	err = store.Save(ctx, &LogicMemory{})
	assert.ErrorIs(t, err, ErrStoreClosed)

	_, err = store.List(ctx, "user:123")
	assert.ErrorIs(t, err, ErrStoreClosed)
}

func TestInMemoryStore_InvalidNamespace(t *testing.T) {
	store := NewInMemoryStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	err := store.Save(ctx, &LogicMemory{
		Namespace: "", // 空 namespace
		Key:       "test",
	})
	assert.ErrorIs(t, err, ErrInvalidNamespace)
}
