package postgres

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

// setupPostgresContainer 启动 PostgreSQL 容器用于测试
func setupPostgresContainer(t *testing.T) (service *Service, cleanup func()) {
	t.Helper()

	// 检查是否在 CI 环境或需要跳过集成测试
	if testing.Short() {
		t.Skip("Skipping PostgreSQL integration test in short mode")
	}
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping PostgreSQL integration test (SKIP_INTEGRATION_TESTS is set)")
	}

	// 捕获 Docker 不可用时的 panic
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("Docker not available, skipping PostgreSQL integration test: %v", r)
		}
	}()

	ctx := context.Background()

	// 创建 PostgreSQL 容器
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Skipf("Failed to start PostgreSQL container (Docker may not be available): %v", err)
	}

	// 获取容器端口
	host, err := container.Host(ctx)
	if err != nil {
		t.Skipf("Failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Skipf("Failed to get container port: %v", err)
	}

	// 构建 DSN
	dsn := fmt.Sprintf("host=%s port=%s user=test password=test dbname=testdb sslmode=disable",
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
	require.NoError(t, err, "Failed to create PostgreSQL service")

	cleanup = func() {
		service.Close()
		container.Terminate(ctx)
	}

	return service, cleanup
}

// TestPostgresService_Create 测试创建 Session
func TestPostgresService_Create(t *testing.T) {
	service, cleanup := setupPostgresContainer(t)
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

// TestPostgresService_AppendEvent 测试追加事件
func TestPostgresService_AppendEvent(t *testing.T) {
	service, cleanup := setupPostgresContainer(t)
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
				Content: "Hello",
			},
		}

		err := service.AppendEvent(ctx, sess.ID, event)
		require.NoError(t, err)

		// 验证事件已存储
		events, err := service.GetEvents(ctx, sess.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, 1, len(events))
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
				Content: "World",
			},
			Actions: session.EventActions{
				StateDelta: map[string]interface{}{
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
}

// TestPostgresService_List 测试列出 Sessions
func TestPostgresService_List(t *testing.T) {
	service, cleanup := setupPostgresContainer(t)
	defer cleanup()

	ctx := context.Background()

	// 创建多个 Sessions
	userID := "user-list-test"
	for i := 0; i < 5; i++ {
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
		assert.Equal(t, 5, len(sessions))
	})

	t.Run("list with limit", func(t *testing.T) {
		sessions, err := service.List(ctx, userID, &session.ListOptions{
			Limit: 3,
		})
		require.NoError(t, err)
		assert.Equal(t, 3, len(sessions))
	})
}

// TestPostgresService_Delete 测试删除 Session
func TestPostgresService_Delete(t *testing.T) {
	service, cleanup := setupPostgresContainer(t)
	defer cleanup()

	ctx := context.Background()

	sess, err := service.Create(ctx, &session.CreateRequest{
		AppName: "test-app",
		UserID:  "user-001",
		AgentID: "agent-001",
	})
	require.NoError(t, err)

	// 删除 Session
	err = service.Delete(ctx, sess.ID)
	require.NoError(t, err)

	// 验证 Session 已删除
	_, err = service.Get(ctx, sess.ID)
	assert.Error(t, err)
}

// TestPostgresService_Concurrency 测试并发安全性
func TestPostgresService_Concurrency(t *testing.T) {
	service, cleanup := setupPostgresContainer(t)
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

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < eventsPerGoroutine; j++ {
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
	for i := 0; i < numGoroutines; i++ {
		<-doneCh
	}
	close(errCh)

	// 检查错误
	var errors []error
	for err := range errCh {
		errors = append(errors, err)
	}
	assert.Equal(t, 0, len(errors), "No errors should occur during concurrent operations")

	// 验证所有事件都已插入
	events, err := service.GetEvents(ctx, sess.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, numGoroutines*eventsPerGoroutine, len(events))
}
