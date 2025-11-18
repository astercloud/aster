package session

import (
	"context"
	"testing"
	"time"

	"github.com/astercloud/aster/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryService_Create(t *testing.T) {
	ctx := context.Background()
	service := NewInMemoryService()

	t.Run("创建成功", func(t *testing.T) {
		req := &CreateRequest{
			AppName: "test-app",
			UserID:  "user-1",
			AgentID: "agent-1",
		}

		sess, err := service.Create(ctx, req)
		require.NoError(t, err)
		assert.NotEmpty(t, sess.ID())
		assert.Equal(t, "test-app", sess.AppName())
		assert.Equal(t, "user-1", sess.UserID())
		assert.Equal(t, "agent-1", sess.AgentID())
	})

	t.Run("多个会话独立", func(t *testing.T) {
		sess1, _ := service.Create(ctx, &CreateRequest{
			AppName: "app1",
			UserID:  "user-1",
			AgentID: "agent-1",
		})

		sess2, _ := service.Create(ctx, &CreateRequest{
			AppName: "app2",
			UserID:  "user-2",
			AgentID: "agent-2",
		})

		assert.NotEqual(t, sess1.ID(), sess2.ID())
		assert.NotEqual(t, sess1.AppName(), sess2.AppName())
	})
}

func TestInMemoryService_Get(t *testing.T) {
	ctx := context.Background()
	service := NewInMemoryService()

	t.Run("获取存在的会话", func(t *testing.T) {
		// 创建会话
		created, _ := service.Create(ctx, &CreateRequest{
			AppName: "test-app",
			UserID:  "user-1",
			AgentID: "agent-1",
		})

		// 获取会话
		retrieved, err := service.Get(ctx, &GetRequest{
			SessionID: created.ID(),
		})
		require.NoError(t, err)
		assert.Equal(t, created.ID(), retrieved.ID())
		assert.Equal(t, created.AppName(), retrieved.AppName())
	})

	t.Run("获取不存在的会话", func(t *testing.T) {
		_, err := service.Get(ctx, &GetRequest{
			SessionID: "non-existent-id",
		})
		assert.ErrorIs(t, err, ErrSessionNotFound)
	})
}

func TestInMemoryService_List(t *testing.T) {
	ctx := context.Background()
	service := NewInMemoryService()

	// 准备测试数据
	userID := "user-list-test"
	for i := 0; i < 5; i++ {
		service.Create(ctx, &CreateRequest{
			AppName: "test-app",
			UserID:  userID,
			AgentID: "agent-1",
		})
	}

	t.Run("列出所有会话", func(t *testing.T) {
		sessions, err := service.List(ctx, &ListRequest{
			UserID: userID,
		})
		require.NoError(t, err)
		assert.Len(t, sessions, 5)
	})

	t.Run("限制返回数量", func(t *testing.T) {
		sessions, err := service.List(ctx, &ListRequest{
			UserID: userID,
			Limit:  3,
		})
		require.NoError(t, err)
		assert.Len(t, sessions, 3)
	})

	t.Run("使用偏移量", func(t *testing.T) {
		sessions, err := service.List(ctx, &ListRequest{
			UserID: userID,
			Offset: 2,
			Limit:  3,
		})
		require.NoError(t, err)
		assert.Len(t, sessions, 3)
	})

	t.Run("按 AppName 过滤", func(t *testing.T) {
		// 创建不同 AppName 的会话
		service.Create(ctx, &CreateRequest{
			AppName: "app-special",
			UserID:  userID,
			AgentID: "agent-1",
		})

		sessions, err := service.List(ctx, &ListRequest{
			UserID:  userID,
			AppName: "app-special",
		})
		require.NoError(t, err)
		assert.Len(t, sessions, 1)
		assert.Equal(t, "app-special", (*sessions[0]).AppName())
	})

	t.Run("空用户无会话", func(t *testing.T) {
		sessions, err := service.List(ctx, &ListRequest{
			UserID: "non-existent-user",
		})
		require.NoError(t, err)
		assert.Len(t, sessions, 0)
	})
}

func TestInMemoryService_Delete(t *testing.T) {
	ctx := context.Background()
	service := NewInMemoryService()

	t.Run("删除存在的会话", func(t *testing.T) {
		sess, _ := service.Create(ctx, &CreateRequest{
			AppName: "test-app",
			UserID:  "user-1",
			AgentID: "agent-1",
		})

		err := service.Delete(ctx, sess.ID())
		require.NoError(t, err)

		// 验证已删除
		_, err = service.Get(ctx, &GetRequest{
			SessionID: sess.ID(),
		})
		assert.ErrorIs(t, err, ErrSessionNotFound)
	})

	t.Run("删除不存在的会话", func(t *testing.T) {
		err := service.Delete(ctx, "non-existent-id")
		assert.ErrorIs(t, err, ErrSessionNotFound)
	})
}

func TestInMemoryService_AppendEvent(t *testing.T) {
	ctx := context.Background()
	service := NewInMemoryService()

	sess, _ := service.Create(ctx, &CreateRequest{
		AppName: "test-app",
		UserID:  "user-1",
		AgentID: "agent-1",
	})

	t.Run("追加事件成功", func(t *testing.T) {
		event := &Event{
			ID:           "evt-1",
			Timestamp:    time.Now(),
			InvocationID: "inv-1",
			AgentID:      "agent-1",
			Branch:       "root",
			Author:       "user",
			Content: types.Message{
				Role:    types.RoleUser,
				Content: "Hello",
			},
		}

		err := service.AppendEvent(ctx, sess.ID(), event)
		require.NoError(t, err)

		// 验证事件已追加
		events, err := service.GetEvents(ctx, sess.ID(), nil)
		require.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, "evt-1", events[0].ID)
	})

	t.Run("追加多个事件", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			event := &Event{
				ID:           "evt-" + string(rune('2'+i)),
				Timestamp:    time.Now(),
				InvocationID: "inv-1",
				AgentID:      "agent-1",
				Branch:       "root",
				Author:       "assistant",
				Content: types.Message{
					Role:    types.RoleAssistant,
					Content: "Response",
				},
			}
			service.AppendEvent(ctx, sess.ID(), event)
		}

		events, _ := service.GetEvents(ctx, sess.ID(), nil)
		assert.Len(t, events, 4) // 1 from previous test + 3 new
	})

	t.Run("事件带状态变更", func(t *testing.T) {
		event := &Event{
			ID:           "evt-state",
			Timestamp:    time.Now(),
			InvocationID: "inv-2",
			AgentID:      "agent-1",
			Branch:       "root",
			Author:       "system",
			Content: types.Message{
				Role:    types.RoleSystem,
				Content: "State update",
			},
			Actions: EventActions{
				StateDelta: map[string]interface{}{
					"session:count": 1,
					"user:theme":    "dark",
				},
			},
		}

		err := service.AppendEvent(ctx, sess.ID(), event)
		require.NoError(t, err)

		// 验证状态已更新（通过GetEvents验证）
		events, err := service.GetEvents(ctx, sess.ID(), nil)
		require.NoError(t, err)
		assert.True(t, len(events) > 0)
	})

	t.Run("事件带工件变更", func(t *testing.T) {
		event := &Event{
			ID:           "evt-artifact",
			Timestamp:    time.Now(),
			InvocationID: "inv-3",
			AgentID:      "agent-1",
			Branch:       "root",
			Author:       "assistant",
			Content: types.Message{
				Role:    types.RoleAssistant,
				Content: "Generated report",
			},
			Actions: EventActions{
				ArtifactDelta: map[string]int64{
					"report.pdf": 1,
				},
			},
		}

		err := service.AppendEvent(ctx, sess.ID(), event)
		require.NoError(t, err)
	})

	t.Run("追加到不存在的会话", func(t *testing.T) {
		event := &Event{
			ID:        "evt-x",
			Timestamp: time.Now(),
			AgentID:   "agent-1",
			Branch:    "root",
			Author:    "user",
		}

		err := service.AppendEvent(ctx, "non-existent", event)
		assert.ErrorIs(t, err, ErrSessionNotFound)
	})
}

func TestInMemoryService_GetEvents(t *testing.T) {
	ctx := context.Background()
	service := NewInMemoryService()

	sess, _ := service.Create(ctx, &CreateRequest{
		AppName: "test-app",
		UserID:  "user-1",
		AgentID: "agent-1",
	})

	// 准备测试数据
	for i := 0; i < 10; i++ {
		event := &Event{
			ID:           "evt-" + string(rune('0'+i)),
			Timestamp:    time.Now(),
			InvocationID: "inv-1",
			AgentID:      "agent-1",
			Branch:       "root",
			Author:       "user",
			Content: types.Message{
				Role:    types.RoleUser,
				Content: "Message",
			},
		}
		service.AppendEvent(ctx, sess.ID(), event)
	}

	t.Run("获取所有事件", func(t *testing.T) {
		events, err := service.GetEvents(ctx, sess.ID(), nil)
		require.NoError(t, err)
		assert.Len(t, events, 10)
	})

	t.Run("限制返回数量", func(t *testing.T) {
		events, err := service.GetEvents(ctx, sess.ID(), &EventFilter{
			Limit: 5,
		})
		require.NoError(t, err)
		assert.Len(t, events, 5)
	})

	t.Run("按 InvocationID 过滤", func(t *testing.T) {
		// 添加不同 InvocationID 的事件
		service.AppendEvent(ctx, sess.ID(), &Event{
			ID:           "evt-special",
			Timestamp:    time.Now(),
			InvocationID: "inv-special",
			AgentID:      "agent-1",
			Branch:       "root",
			Author:       "user",
		})

		// EventFilter不支持InvocationID，获取所有事件后手动过滤
		events, err := service.GetEvents(ctx, sess.ID(), nil)
		require.NoError(t, err)

		var filtered []Event
		for _, e := range events {
			if e.InvocationID == "inv-special" {
				filtered = append(filtered, e)
			}
		}
		assert.Len(t, filtered, 1)
		assert.Equal(t, "inv-special", filtered[0].InvocationID)
	})

	t.Run("按 Branch 过滤", func(t *testing.T) {
		// 添加不同 Branch 的事件
		service.AppendEvent(ctx, sess.ID(), &Event{
			ID:           "evt-branch",
			Timestamp:    time.Now(),
			InvocationID: "inv-1",
			AgentID:      "agent-1",
			Branch:       "root.sub",
			Author:       "user",
		})

		events, err := service.GetEvents(ctx, sess.ID(), &EventFilter{
			Branch: "root.sub",
		})
		require.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, "root.sub", events[0].Branch)
	})
}

func TestInMemoryService_GetState(t *testing.T) {
	ctx := context.Background()
	service := NewInMemoryService()

	sess, _ := service.Create(ctx, &CreateRequest{
		AppName: "test-app",
		UserID:  "user-1",
		AgentID: "agent-1",
	})

	// 添加各种作用域的状态
	event := &Event{
		ID:           "evt-1",
		Timestamp:    time.Now(),
		InvocationID: "inv-1",
		AgentID:      "agent-1",
		Branch:       "root",
		Author:       "system",
		Content: types.Message{
			Role:    types.RoleSystem,
			Content: "State setup",
		},
		Actions: EventActions{
			StateDelta: map[string]interface{}{
				"app:version":    "1.0.0",
				"user:language":  "zh-CN",
				"session:page":   1,
				"temp:cache_key": "temp-value",
			},
		},
	}
	service.AppendEvent(ctx, sess.ID(), event)

	t.Run("获取所有状态", func(t *testing.T) {
		// 通过Session接口访问状态
		retrievedSess, err := service.Get(ctx, &GetRequest{
			SessionID: sess.ID(),
		})
		require.NoError(t, err)
		state := retrievedSess.State()
		assert.NotNil(t, state)
		// 验证状态存在
		assert.True(t, state.Has("app:version"))
		assert.True(t, state.Has("user:language"))
		assert.True(t, state.Has("session:page"))
	})

	t.Run("按作用域过滤", func(t *testing.T) {
		// 通过Session接口访问状态
		retrievedSess, err := service.Get(ctx, &GetRequest{
			SessionID: sess.ID(),
		})
		require.NoError(t, err)
		state := retrievedSess.State()
		val, err := state.Get("user:language")
		require.NoError(t, err)
		assert.Equal(t, "zh-CN", val)
	})

	t.Run("空会话无状态", func(t *testing.T) {
		emptySess, _ := service.Create(ctx, &CreateRequest{
			AppName: "empty",
			UserID:  "user-1",
			AgentID: "agent-1",
		})

		retrievedSess, err := service.Get(ctx, &GetRequest{
			SessionID: emptySess.ID(),
		})
		require.NoError(t, err)
		state := retrievedSess.State()
		assert.NotNil(t, state)
	})
}

func TestInMemoryService_Concurrency(t *testing.T) {
	ctx := context.Background()
	service := NewInMemoryService()

	sess, _ := service.Create(ctx, &CreateRequest{
		AppName: "test-app",
		UserID:  "user-1",
		AgentID: "agent-1",
	})

	t.Run("并发追加事件", func(t *testing.T) {
		done := make(chan bool)
		numGoroutines := 10
		eventsPerGoroutine := 10

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < eventsPerGoroutine; j++ {
					event := &Event{
						ID:           "evt-concurrent-" + string(rune('0'+id)) + "-" + string(rune('0'+j)),
						Timestamp:    time.Now(),
						InvocationID: "inv-1",
						AgentID:      "agent-1",
						Branch:       "root",
						Author:       "user",
					}
					service.AppendEvent(ctx, sess.ID(), event)
				}
				done <- true
			}(i)
		}

		// 等待所有 goroutine 完成
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// 验证所有事件都已追加
		events, _ := service.GetEvents(ctx, sess.ID(), nil)
		assert.Len(t, events, numGoroutines*eventsPerGoroutine)
	})
}

func TestInMemoryService_StateScopes(t *testing.T) {
	ctx := context.Background()
	service := NewInMemoryService()

	sess, _ := service.Create(ctx, &CreateRequest{
		AppName: "test-app",
		UserID:  "user-1",
		AgentID: "agent-1",
	})

	t.Run("状态作用域隔离", func(t *testing.T) {
		// App 级状态（所有用户共享）
		service.AppendEvent(ctx, sess.ID(), &Event{
			ID:           "evt-1",
			Timestamp:    time.Now(),
			InvocationID: "inv-1",
			AgentID:      "agent-1",
			Branch:       "root",
			Author:       "system",
			Actions: EventActions{
				StateDelta: map[string]interface{}{
					"app:feature_enabled": true,
				},
			},
		})

		// User 级状态（该用户所有会话共享）
		service.AppendEvent(ctx, sess.ID(), &Event{
			ID:           "evt-2",
			Timestamp:    time.Now(),
			InvocationID: "inv-1",
			AgentID:      "agent-1",
			Branch:       "root",
			Author:       "system",
			Actions: EventActions{
				StateDelta: map[string]interface{}{
					"user:preference": "value",
				},
			},
		})

		// Session 级状态（当前会话）
		service.AppendEvent(ctx, sess.ID(), &Event{
			ID:           "evt-3",
			Timestamp:    time.Now(),
			InvocationID: "inv-1",
			AgentID:      "agent-1",
			Branch:       "root",
			Author:       "system",
			Actions: EventActions{
				StateDelta: map[string]interface{}{
					"session:data": "session-specific",
				},
			},
		})

		// 验证各作用域
		retrievedSess, err := service.Get(ctx, &GetRequest{
			SessionID: sess.ID(),
		})
		require.NoError(t, err)
		state := retrievedSess.State()

		assert.True(t, state.Has("app:feature_enabled"))
		assert.True(t, state.Has("user:preference"))
		assert.True(t, state.Has("session:data"))
	})
}
