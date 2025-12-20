package mysql

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/astercloud/aster/pkg/session"
	"github.com/astercloud/aster/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/gorm/logger"
)

// TestMain 设置测试环境
func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

// setupMySQLContainer 启动 MySQL 8.0+ 容器用于测试
func setupMySQLContainer(t *testing.T) (service *Service, cleanup func()) {
	t.Helper()

	// 检查是否在 CI 环境或需要跳过集成测试
	if testing.Short() {
		t.Skip("Skipping MySQL integration test in short mode")
	}
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping MySQL integration test (SKIP_INTEGRATION_TESTS is set)")
	}

	// 捕获 Docker 不可用时的 panic
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("Docker not available, skipping MySQL integration test: %v", r)
		}
	}()

	ctx := context.Background()

	// 创建 MySQL 容器
	req := testcontainers.ContainerRequest{
		Image:        "mysql:8.0",
		ExposedPorts: []string{"3306/tcp"},
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": "test",
			"MYSQL_DATABASE":      "testdb",
		},
		WaitingFor: wait.ForLog("port: 3306  MySQL Community Server").
			WithStartupTimeout(120 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Skipf("Failed to start MySQL container (Docker may not be available): %v", err)
	}

	// 获取容器端口
	host, err := container.Host(ctx)
	if err != nil {
		t.Skipf("Failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "3306")
	if err != nil {
		t.Skipf("Failed to get container port: %v", err)
	}

	// 构建 DSN
	dsn := fmt.Sprintf("root:test@tcp(%s:%s)/testdb?charset=utf8mb4&parseTime=True&loc=Local",
		host, port.Port())

	// 创建服务
	cfg := &Config{
		DSN:             dsn,
		MaxIdleConns:    5,
		MaxOpenConns:    10,
		ConnMaxLifetime: time.Hour,
		LogLevel:        logger.Silent,
		AutoMigrate:     true,
	}

	service, err = NewService(cfg)
	require.NoError(t, err, "Failed to create MySQL service")

	cleanup = func() {
		if err := service.Close(); err != nil {
			t.Errorf("Failed to close service: %v", err)
		}
		if err := container.Terminate(ctx); err != nil {
			t.Errorf("Failed to terminate container: %v", err)
		}
	}

	return service, cleanup
}

// TestMySQLService_Create 测试创建 Session
func TestMySQLService_Create(t *testing.T) {
	service, cleanup := setupMySQLContainer(t)
	defer cleanup()

	ctx := context.Background()

	req := &session.CreateRequest{
		AppName: "test-app",
		UserID:  "user-001",
		AgentID: "agent-001",
	}

	sess, err := service.Create(ctx, req)
	require.NoError(t, err)
	assert.NotEmpty(t, sess.ID)
	assert.Equal(t, req.AppName, sess.AppName)
	assert.Equal(t, req.UserID, sess.UserID)
	assert.Equal(t, req.AgentID, sess.AgentID)
	assert.NotZero(t, sess.CreatedAt)

	// 验证可以获取
	retrieved, err := service.Get(ctx, sess.ID)
	require.NoError(t, err)
	assert.Equal(t, sess.ID, retrieved.ID)
}

// TestMySQLService_AppendEvent 测试追加事件
func TestMySQLService_AppendEvent(t *testing.T) {
	service, cleanup := setupMySQLContainer(t)
	defer cleanup()

	ctx := context.Background()

	sess, err := service.Create(ctx, &session.CreateRequest{
		AppName: "test-app",
		UserID:  "user-001",
		AgentID: "agent-001",
	})
	require.NoError(t, err)

	t.Run("append basic event", func(t *testing.T) {
		event := &session.Event{
			ID:           "evt-001",
			Timestamp:    time.Now(),
			InvocationID: "inv-001",
			AgentID:      "agent-001",
			Branch:       "root",
			Author:       "user",
			Content: types.Message{
				Role:    types.RoleUser,
				Content: "Hello MySQL",
			},
		}

		err := service.AppendEvent(ctx, sess.ID, event)
		require.NoError(t, err)

		// 验证事件已存储
		events, err := service.GetEvents(ctx, sess.ID, nil)
		require.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, event.ID, events[0].ID)
	})

	t.Run("append event with state delta", func(t *testing.T) {
		event := &session.Event{
			ID:           "evt-002",
			Timestamp:    time.Now(),
			InvocationID: "inv-001",
			AgentID:      "agent-001",
			Branch:       "root",
			Author:       "assistant",
			Content: types.Message{
				Role:    types.RoleAssistant,
				Content: "Testing MySQL JSON columns",
			},
			Actions: session.EventActions{
				StateDelta: map[string]any{
					"session:counter": 42,
					"session:name":    "test",
				},
			},
		}

		err := service.AppendEvent(ctx, sess.ID, event)
		require.NoError(t, err)

		// 验证状态已更新
		state, err := service.GetState(ctx, sess.ID, "session")
		require.NoError(t, err)
		assert.Equal(t, float64(42), state["counter"])
		assert.Equal(t, "test", state["name"])
	})

	t.Run("append event with tool calls", func(t *testing.T) {
		event := &session.Event{
			ID:           "evt-003",
			Timestamp:    time.Now(),
			InvocationID: "inv-001",
			AgentID:      "agent-001",
			Branch:       "root",
			Author:       "assistant",
			Content: types.Message{
				Role:    types.RoleAssistant,
				Content: "Using tools",
				ToolCalls: []types.ToolCall{
					{
						ID:   "call-1",
						Name: "search",
						Arguments: map[string]any{
							"query": "MySQL JSON",
						},
					},
				},
			},
		}

		err := service.AppendEvent(ctx, sess.ID, event)
		require.NoError(t, err)

		// 验证 ToolCalls 存储正确
		events, err := service.GetEvents(ctx, sess.ID, nil)
		require.NoError(t, err)

		var found bool
		for _, e := range events {
			if e.ID == "evt-003" {
				found = true
				assert.Len(t, e.Content.ToolCalls, 1)
				assert.Equal(t, "search", e.Content.ToolCalls[0].Name)
			}
		}
		assert.True(t, found)
	})
}

// TestMySQLService_List 测试列出 Sessions
func TestMySQLService_List(t *testing.T) {
	service, cleanup := setupMySQLContainer(t)
	defer cleanup()

	ctx := context.Background()

	// 创建多个 Sessions
	userID := "user-list-test"
	for i := range 5 {
		_, err := service.Create(ctx, &session.CreateRequest{
			AppName: "test-app",
			UserID:  userID,
			AgentID: fmt.Sprintf("agent-%d", i),
		})
		require.NoError(t, err)
	}

	t.Run("list all for user", func(t *testing.T) {
		sessions, err := service.List(ctx, userID, nil)
		require.NoError(t, err)
		assert.Len(t, sessions, 5)
	})

	t.Run("list with limit", func(t *testing.T) {
		sessions, err := service.List(ctx, userID, &session.ListOptions{
			Limit: 3,
		})
		require.NoError(t, err)
		assert.Len(t, sessions, 3)
	})

	t.Run("list with offset", func(t *testing.T) {
		sessions, err := service.List(ctx, userID, &session.ListOptions{
			Limit:  2,
			Offset: 2,
		})
		require.NoError(t, err)
		assert.Len(t, sessions, 2)
	})
}

// TestMySQLService_Delete 测试删除 Session
func TestMySQLService_Delete(t *testing.T) {
	service, cleanup := setupMySQLContainer(t)
	defer cleanup()

	ctx := context.Background()

	sess, err := service.Create(ctx, &session.CreateRequest{
		AppName: "test-app",
		UserID:  "user-001",
		AgentID: "agent-001",
	})
	require.NoError(t, err)

	// 添加一些事件
	event := &session.Event{
		ID:           "evt-001",
		Timestamp:    time.Now(),
		InvocationID: "inv-001",
		AgentID:      "agent-001",
		Branch:       "root",
		Author:       "user",
		Content: types.Message{
			Role:    types.RoleUser,
			Content: "test",
		},
	}
	err = service.AppendEvent(ctx, sess.ID, event)
	require.NoError(t, err)

	// 删除 Session
	err = service.Delete(ctx, sess.ID)
	require.NoError(t, err)

	// 验证 Session 已删除
	_, err = service.Get(ctx, sess.ID)
	assert.Error(t, err)
}

// TestMySQLService_GetEvents 测试获取事件
func TestMySQLService_GetEvents(t *testing.T) {
	service, cleanup := setupMySQLContainer(t)
	defer cleanup()

	ctx := context.Background()

	sess, err := service.Create(ctx, &session.CreateRequest{
		AppName: "test-app",
		UserID:  "user-001",
		AgentID: "agent-001",
	})
	require.NoError(t, err)

	// 创建多个事件
	for i := range 10 {
		event := &session.Event{
			ID:           fmt.Sprintf("evt-%03d", i),
			Timestamp:    time.Now().Add(time.Duration(i) * time.Millisecond),
			InvocationID: fmt.Sprintf("inv-%d", i%3),
			AgentID:      "agent-001",
			Branch:       "root",
			Author:       "user",
			Content: types.Message{
				Role:    types.RoleUser,
				Content: fmt.Sprintf("Message %d", i),
			},
		}
		err := service.AppendEvent(ctx, sess.ID, event)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond)
	}

	t.Run("get all events", func(t *testing.T) {
		events, err := service.GetEvents(ctx, sess.ID, nil)
		require.NoError(t, err)
		assert.Len(t, events, 10)
	})

	t.Run("get events with limit", func(t *testing.T) {
		events, err := service.GetEvents(ctx, sess.ID, &session.EventOptions{
			Limit: 5,
		})
		require.NoError(t, err)
		assert.Len(t, events, 5)
	})

	t.Run("filter by invocation_id", func(t *testing.T) {
		invocationID := "inv-0"
		events, err := service.GetEvents(ctx, sess.ID, &session.EventOptions{
			InvocationID: invocationID,
		})
		require.NoError(t, err)
		for _, e := range events {
			assert.Equal(t, invocationID, e.InvocationID)
		}
	})
}

// TestMySQLService_Concurrency 测试并发安全性
func TestMySQLService_Concurrency(t *testing.T) {
	service, cleanup := setupMySQLContainer(t)
	defer cleanup()

	ctx := context.Background()

	sess, err := service.Create(ctx, &session.CreateRequest{
		AppName: "test-app",
		UserID:  "user-001",
		AgentID: "agent-001",
	})
	require.NoError(t, err)

	// 并发追加事件
	const numGoroutines = 10
	const eventsPerGoroutine = 10

	errCh := make(chan error, numGoroutines*eventsPerGoroutine)
	doneCh := make(chan struct{})

	for i := range numGoroutines {
		go func(goroutineID int) {
			for j := range eventsPerGoroutine {
				event := &session.Event{
					ID:           fmt.Sprintf("evt-g%d-e%d", goroutineID, j),
					Timestamp:    time.Now(),
					InvocationID: "inv-concurrent",
					AgentID:      "agent-001",
					Branch:       "root",
					Author:       "user",
					Content: types.Message{
						Role:    types.RoleUser,
						Content: fmt.Sprintf("Message from goroutine %d, event %d", goroutineID, j),
					},
				}

				if err := service.AppendEvent(ctx, sess.ID, event); err != nil {
					errCh <- err
				}
			}
			doneCh <- struct{}{}
		}(i)
	}

	// 等待所有 goroutine 完成
	for range numGoroutines {
		<-doneCh
	}
	close(errCh)

	// 检查错误
	var errors []error
	for err := range errCh {
		errors = append(errors, err)
	}
	assert.Empty(t, errors, "No errors should occur during concurrent operations")

	// 验证所有事件都已插入
	events, err := service.GetEvents(ctx, sess.ID, nil)
	require.NoError(t, err)
	assert.Len(t, events, numGoroutines*eventsPerGoroutine)
}

// TestMySQLService_JSONColumns 测试 MySQL JSON 列功能
func TestMySQLService_JSONColumns(t *testing.T) {
	service, cleanup := setupMySQLContainer(t)
	defer cleanup()

	ctx := context.Background()

	sess, err := service.Create(ctx, &session.CreateRequest{
		AppName: "test-app",
		UserID:  "user-001",
		AgentID: "agent-001",
	})
	require.NoError(t, err)

	// 测试复杂嵌套 JSON
	event := &session.Event{
		ID:           "evt-json",
		Timestamp:    time.Now(),
		InvocationID: "inv-001",
		AgentID:      "agent-001",
		Branch:       "root",
		Author:       "assistant",
		Content: types.Message{
			Role:    types.RoleAssistant,
			Content: "Complex JSON test",
		},
		Metadata: map[string]any{
			"nested": map[string]any{
				"level1": map[string]any{
					"level2": "deep_value",
				},
			},
			"array":   []string{"item1", "item2", "item3"},
			"chinese": "测试中文",
			"emoji":   "😀🎉",
		},
	}

	err = service.AppendEvent(ctx, sess.ID, event)
	require.NoError(t, err)

	// 验证 JSON 数据正确存储和检索
	events, err := service.GetEvents(ctx, sess.ID, nil)
	require.NoError(t, err)
	assert.Len(t, events, 1)

	metadata := events[0].Metadata
	assert.Equal(t, "测试中文", metadata["chinese"])
	assert.Equal(t, "😀🎉", metadata["emoji"])

	nested := metadata["nested"].(map[string]any)
	level1 := nested["level1"].(map[string]any)
	assert.Equal(t, "deep_value", level1["level2"])
}
