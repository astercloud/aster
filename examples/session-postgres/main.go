// SessionPostgres 演示 PostgreSQL Session 持久化功能，使用 JSONB 列类型
// 实现高效的会话状态和事件存储。
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/astercloud/aster/pkg/session"
	"github.com/astercloud/aster/pkg/session/postgres"
	"github.com/astercloud/aster/pkg/types"
	"gorm.io/gorm/logger"
)

func main() {
	ctx := context.Background()

	// ====== 示例 1: 基础连接和配置 ======
	fmt.Println("=== Example 1: Basic Connection ===")
	service := setupPostgreSQLService()
	defer func() { _ = service.Close() }()

	// ====== 示例 2: 创建 Session ======
	fmt.Println("\n=== Example 2: Create Session ===")
	sess := createSessionExample(ctx, service)

	// ====== 示例 3: 追加事件 ======
	fmt.Println("\n=== Example 3: Append Events ===")
	appendEventsExample(ctx, service, sess.ID)

	// ====== 示例 4: 状态管理 ======
	fmt.Println("\n=== Example 4: State Management ===")
	stateManagementExample(ctx, service, sess.ID)

	// ====== 示例 5: 工件版本管理 ======
	fmt.Println("\n=== Example 5: Artifact Versioning ===")
	artifactVersioningExample(ctx, service, sess.ID)

	// ====== 示例 6: 查询和过滤 ======
	fmt.Println("\n=== Example 6: Query and Filter ===")
	queryExample(ctx, service, sess.UserID)

	fmt.Println("\n✅ All examples completed successfully!")
}

// setupPostgreSQLService 设置 PostgreSQL 服务
func setupPostgreSQLService() *postgres.Service {
	cfg := &postgres.Config{
		// 连接字符串 - 请根据实际环境修改
		DSN: "host=localhost user=postgres password=postgres dbname=aster port=5432 sslmode=disable",

		// 连接池配置
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: time.Hour,

		// 日志级别
		LogLevel: logger.Info,

		// 自动迁移（开发环境）
		AutoMigrate: true,
	}

	service, err := postgres.NewService(cfg)
	if err != nil {
		log.Fatalf("Failed to create PostgreSQL service: %v", err)
	}

	fmt.Println("✅ Connected to PostgreSQL")
	return service
}

// createSessionExample 创建 Session 示例
func createSessionExample(ctx context.Context, service *postgres.Service) *session.SessionData {
	req := &session.CreateRequest{
		AppName: "demo-app",
		UserID:  "user-001",
		AgentID: "agent-customer-support",
	}

	sess, err := service.Create(ctx, req)
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	fmt.Printf("✅ Created session: %s\n", sess.ID)
	fmt.Printf("   App: %s, User: %s, Agent: %s\n", sess.AppName, sess.UserID, sess.AgentID)
	return sess
}

// appendEventsExample 追加事件示例
func appendEventsExample(ctx context.Context, service *postgres.Service, sessionID string) {
	// 1. 用户消息事件
	userEvent := &session.Event{
		ID:           "evt-" + generateID(),
		Timestamp:    time.Now(),
		InvocationID: "inv-001",
		AgentID:      "agent-customer-support",
		Branch:       "root",
		Author:       "user",
		Content: types.Message{
			Role:    types.RoleUser,
			Content: "我需要帮助",
		},
	}

	if err := service.AppendEvent(ctx, sessionID, userEvent); err != nil {
		log.Fatalf("Failed to append user event: %v", err)
	}

	// 2. Agent 响应事件（带状态变更）
	agentEvent := &session.Event{
		ID:           "evt-" + generateID(),
		Timestamp:    time.Now(),
		InvocationID: "inv-001",
		AgentID:      "agent-customer-support",
		Branch:       "root",
		Author:       "assistant",
		Content: types.Message{
			Role:    types.RoleAssistant,
			Content: "我很乐意帮助您！请问有什么问题吗？",
		},
		Actions: session.EventActions{
			StateDelta: map[string]any{
				"session:last_response_time": time.Now().Unix(),
				"session:message_count":      1,
			},
		},
	}

	if err := service.AppendEvent(ctx, sessionID, agentEvent); err != nil {
		log.Fatalf("Failed to append agent event: %v", err)
	}

	fmt.Println("✅ Appended 2 events")
}

// stateManagementExample 状态管理示例
func stateManagementExample(ctx context.Context, service *postgres.Service, sessionID string) {
	// 1. 添加不同作用域的状态
	event := &session.Event{
		ID:           "evt-" + generateID(),
		Timestamp:    time.Now(),
		InvocationID: "inv-002",
		AgentID:      "agent-customer-support",
		Branch:       "root",
		Author:       "system",
		Content: types.Message{
			Role:    types.RoleSystem,
			Content: "State update",
		},
		Actions: session.EventActions{
			StateDelta: map[string]any{
				// App 级状态（所有用户共享）
				"app:version": "1.0.0",
				"app:feature_flags": map[string]bool{
					"enable_voice": true,
					"enable_video": false,
				},

				// User 级状态（该用户所有会话共享）
				"user:language": "zh-CN",
				"user:theme":    "dark",
				"user:timezone": "Asia/Shanghai",

				// Session 级状态（当前会话）
				"session:start_time":         time.Now().Unix(),
				"session:conversation_topic": "帮助咨询",

				// Temp 级状态（临时）
				"temp:cache_key": "temp-value",
			},
		},
	}

	if err := service.AppendEvent(ctx, sessionID, event); err != nil {
		log.Fatalf("Failed to update state: %v", err)
	}

	// 2. 查询状态
	state, err := service.GetState(ctx, sessionID, "")
	if err != nil {
		log.Fatalf("Failed to get state: %v", err)
	}

	fmt.Printf("✅ State updated (%d keys)\n", len(state))
	fmt.Println("   App state: version =", state["app:version"])
	fmt.Println("   User state: language =", state["user:language"])
	fmt.Println("   Session state: topic =", state["session:conversation_topic"])
}

// artifactVersioningExample 工件版本管理示例
func artifactVersioningExample(ctx context.Context, service *postgres.Service, sessionID string) {
	// 模拟生成报告的多个版本
	for version := 1; version <= 3; version++ {
		event := &session.Event{
			ID:           "evt-" + generateID(),
			Timestamp:    time.Now(),
			InvocationID: fmt.Sprintf("inv-artifact-%d", version),
			AgentID:      "agent-customer-support",
			Branch:       "root",
			Author:       "assistant",
			Content: types.Message{
				Role:    types.RoleAssistant,
				Content: fmt.Sprintf("生成报告版本 %d", version),
			},
			Actions: session.EventActions{
				ArtifactDelta: map[string]int64{
					"customer_report.pdf": int64(version),
				},
			},
		}

		if err := service.AppendEvent(ctx, sessionID, event); err != nil {
			log.Fatalf("Failed to append artifact event: %v", err)
		}

		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("✅ Created 3 artifact versions")
}

// queryExample 查询和过滤示例
func queryExample(ctx context.Context, service *postgres.Service, userID string) {
	// 1. 列出用户的所有 Session
	sessions, err := service.List(ctx, userID, &session.ListOptions{
		Limit: 10,
	})
	if err != nil {
		log.Fatalf("Failed to list sessions: %v", err)
	}

	fmt.Printf("✅ Found %d sessions for user %s\n", len(sessions), userID)

	// 2. 获取 Session 的事件
	if len(sessions) > 0 {
		sessionID := sessions[0].ID
		events, err := service.GetEvents(ctx, sessionID, &session.EventOptions{
			Limit: 10,
		})
		if err != nil {
			log.Fatalf("Failed to get events: %v", err)
		}

		fmt.Printf("✅ Session %s has %d events\n", sessionID[:8], len(events))
		for i, evt := range events {
			fmt.Printf("   [%d] %s: %s\n", i+1, evt.Author, truncate(evt.Content.Content, 30))
		}
	}

	// 3. 按分支过滤事件
	if len(sessions) > 0 {
		sessionID := sessions[0].ID
		events, err := service.GetEvents(ctx, sessionID, &session.EventOptions{
			Branch: "root",
			Limit:  5,
		})
		if err != nil {
			log.Fatalf("Failed to filter events by branch: %v", err)
		}

		fmt.Printf("✅ Found %d events in branch 'root'\n", len(events))
	}
}

// 辅助函数

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// 生产环境最佳实践

var _ = func() {
	/*
		1. 连接字符串安全
		   - 使用环境变量存储 DSN
		   - 不要将密码硬编码在代码中

		   import "os"
		   dsn := os.Getenv("POSTGRES_DSN")
		   if dsn == "" {
		       dsn = "host=localhost user=postgres password=postgres dbname=aster port=5432 sslmode=require"
		   }

		2. 连接池优化
		   cfg := &postgres.Config{
		       DSN:             dsn,
		       MaxIdleConns:    10,
		       MaxOpenConns:    100,
		       ConnMaxLifetime: time.Hour,
		   }

		3. 错误处理和重试
		   service, err := postgres.NewService(cfg)
		   if err != nil {
		       // 实现重试逻辑
		       for i := 0; i < 3; i++ {
		           service, err = postgres.NewService(cfg)
		           if err == nil {
		               break
		           }
		           time.Sleep(time.Second * time.Duration(i+1))
		       }
		   }

		4. 健康检查
		   func healthCheck(service *postgres.Service) error {
		       ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		       defer cancel()

		       // 尝试查询
		       _, err := service.Get(ctx, "non-existent-id")
		       if err != nil && err != session.ErrSessionNotFound {
		           return err
		       }
		       return nil
		   }

		5. 优雅关闭
		   func gracefulShutdown(service *postgres.Service) {
		       ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		       defer cancel()

		       // 刷新待处理的写入
		       if err := service.Close(); err != nil {
		           log.Printf("Error closing service: %v", err)
		       }
		   }

		6. 监控和指标
		   - 连接池使用率
		   - 查询延迟
		   - 错误率
		   - 事件吞吐量

		7. 定期清理
		   // 每天清理 90 天前的旧会话
		   func cleanupOldSessions() {
		       // 使用数据库函数
		       db.Exec("SELECT cleanup_old_sessions(90)")
		   }
	*/
}
