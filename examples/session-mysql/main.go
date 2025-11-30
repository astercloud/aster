// SessionMySQL 演示 MySQL 8.0+ Session 持久化功能，利用 MySQL 的 JSON
// 列类型实现会话状态和事件的持久化存储。
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/astercloud/aster/pkg/session"
	"github.com/astercloud/aster/pkg/session/mysql"
	"github.com/astercloud/aster/pkg/types"
	"gorm.io/gorm/logger"
)

func main() {
	ctx := context.Background()

	// ====== 示例 1: 连接 MySQL ======
	fmt.Println("=== Example 1: Connect to MySQL 8.0+ ===")
	service := setupMySQLService()
	defer func() { _ = service.Close() }()

	// ====== 示例 2: 创建 Session ======
	fmt.Println("\n=== Example 2: Create Session ===")
	sess := createSessionExample(ctx, service)

	// ====== 示例 3: 事件追加 ======
	fmt.Println("\n=== Example 3: Append Events ===")
	appendEventsExample(ctx, service, sess.ID)

	// ====== 示例 4: JSON 列的优势 ======
	fmt.Println("\n=== Example 4: JSON Column Benefits ===")
	jsonColumnExample(ctx, service, sess.ID)

	// ====== 示例 5: 查询优化 ======
	fmt.Println("\n=== Example 5: Query Optimization ===")
	queryOptimizationExample(ctx, service, sess.UserID)

	fmt.Println("\n✅ All MySQL examples completed!")
}

// setupMySQLService 设置 MySQL 服务
func setupMySQLService() *mysql.Service {
	cfg := &mysql.Config{
		// 连接字符串 - 请根据实际环境修改
		// 注意: parseTime=True 用于自动解析 TIMESTAMP
		// charset=utf8mb4 支持完整的 UTF-8 字符集
		DSN: "root:password@tcp(127.0.0.1:3306)/aster?charset=utf8mb4&parseTime=True&loc=Local",

		// 连接池配置
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: time.Hour,

		// 日志级别
		LogLevel: logger.Info,

		// 自动迁移（开发环境）
		AutoMigrate: true,
	}

	service, err := mysql.NewService(cfg)
	if err != nil {
		log.Fatalf("Failed to create MySQL service: %v", err)
	}

	fmt.Println("✅ Connected to MySQL 8.0+")
	return service
}

// createSessionExample 创建 Session
func createSessionExample(ctx context.Context, service *mysql.Service) *session.SessionData {
	req := &session.CreateRequest{
		AppName: "mysql-demo-app",
		UserID:  "user-mysql-001",
		AgentID: "agent-mysql-support",
	}

	sess, err := service.Create(ctx, req)
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	fmt.Printf("✅ Created session: %s\n", sess.ID)
	return sess
}

// appendEventsExample 追加事件
func appendEventsExample(ctx context.Context, service *mysql.Service, sessionID string) {
	event := &session.Event{
		ID:           "evt-" + generateID(),
		Timestamp:    time.Now(),
		InvocationID: "inv-mysql-001",
		AgentID:      "agent-mysql-support",
		Branch:       "root",
		Author:       "user",
		Content: types.Message{
			Role:    types.RoleUser,
			Content: "测试 MySQL JSON 列",
		},
		Actions: session.EventActions{
			StateDelta: map[string]any{
				"session:db_type": "mysql",
				"session:version": "8.0+",
			},
		},
	}

	if err := service.AppendEvent(ctx, sessionID, event); err != nil {
		log.Fatalf("Failed to append event: %v", err)
	}

	fmt.Println("✅ Event appended with JSON columns")
}

// jsonColumnExample 演示 JSON 列的优势
func jsonColumnExample(ctx context.Context, service *mysql.Service, sessionID string) {
	// MySQL 8.0+ 的 JSON 列优势:
	// 1. 原生 JSON 类型（不是字符串）
	// 2. 自动验证 JSON 格式
	// 3. 支持 JSON 函数查询
	// 4. 可以为 JSON 字段创建虚拟列索引

	event := &session.Event{
		ID:           "evt-" + generateID(),
		Timestamp:    time.Now(),
		InvocationID: "inv-mysql-002",
		AgentID:      "agent-mysql-support",
		Branch:       "root",
		Author:       "assistant",
		Content: types.Message{
			Role:    types.RoleAssistant,
			Content: "MySQL 支持丰富的 JSON 操作",
			ToolCalls: []types.ToolCall{
				{
					ID:   "call-1",
					Name: "json_demo",
					Arguments: map[string]any{
						"nested": map[string]any{
							"level1": map[string]any{
								"level2": "深层嵌套也可以",
							},
						},
						"array": []string{"item1", "item2", "item3"},
					},
				},
			},
		},
		Metadata: map[string]any{
			"mysql_features": []string{
				"JSON_EXTRACT",
				"JSON_CONTAINS",
				"JSON_SEARCH",
				"Virtual Columns",
			},
		},
	}

	if err := service.AppendEvent(ctx, sessionID, event); err != nil {
		log.Fatalf("Failed to append JSON example event: %v", err)
	}

	fmt.Println("✅ Complex JSON structure stored")
	fmt.Println("   MySQL can query nested fields efficiently")
}

// queryOptimizationExample 查询优化示例
func queryOptimizationExample(ctx context.Context, service *mysql.Service, userID string) {
	// 列出用户的 Sessions
	sessions, err := service.List(ctx, userID, &session.ListOptions{
		Limit: 10,
	})
	if err != nil {
		log.Fatalf("Failed to list sessions: %v", err)
	}

	fmt.Printf("✅ Found %d sessions\n", len(sessions))

	if len(sessions) > 0 {
		// 获取事件
		events, err := service.GetEvents(ctx, sessions[0].ID, &session.EventOptions{
			Limit: 5,
		})
		if err != nil {
			log.Fatalf("Failed to get events: %v", err)
		}

		fmt.Printf("✅ Session has %d events\n", len(events))
	}
}

// 辅助函数

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// MySQL 8.0+ JSON 高级用法

var _ = func() {
	/*
		1. JSON 路径查询
		   SELECT JSON_EXTRACT(content, '$.role') FROM session_events;
		   SELECT content->>'$.role' FROM session_events;  -- 简化语法

		2. JSON 搜索
		   SELECT * FROM session_events
		   WHERE JSON_CONTAINS(content, '"user"', '$.role');

		3. JSON 数组操作
		   SELECT * FROM session_events
		   WHERE JSON_LENGTH(content->'$.tool_calls') > 0;

		4. JSON 更新
		   UPDATE session_events
		   SET content = JSON_SET(content, '$.processed', true)
		   WHERE id = '...';

		5. 虚拟列索引（提升查询性能）
		   ALTER TABLE session_events
		   ADD COLUMN event_role VARCHAR(50) AS (content->>'$.role') VIRTUAL,
		   ADD INDEX idx_event_role (event_role);

		   -- 查询会自动使用索引
		   SELECT * FROM session_events WHERE event_role = 'user';

		6. JSON 聚合
		   SELECT
		       content->>'$.role' AS role,
		       COUNT(*) AS count
		   FROM session_events
		   GROUP BY content->>'$.role';

		7. JSON 数组展开
		   SELECT
		       id,
		       JSON_EXTRACT(tool_call, '$.name') AS tool_name
		   FROM session_events,
		   JSON_TABLE(
		       content->'$.tool_calls',
		       '$[*]' COLUMNS (
		           tool_call JSON PATH '$'
		       )
		   ) AS jt;
	*/
}

// 生产环境最佳实践

var _ = func() {
	/*
		1. 连接字符串配置
		   import "os"

		   dsn := os.Getenv("MYSQL_DSN")
		   if dsn == "" {
		       dsn = "user:password@tcp(localhost:3306)/aster?charset=utf8mb4&parseTime=True&loc=Local"
		   }

		2. MySQL 8.0+ 要求
		   - 确保 MySQL 版本 >= 8.0
		   - 启用 JSON 支持
		   - 配置合适的字符集: utf8mb4

		3. 连接池优化
		   cfg := &mysql.Config{
		       DSN:             dsn,
		       MaxIdleConns:    10,    // 空闲连接
		       MaxOpenConns:    100,   // 最大连接
		       ConnMaxLifetime: time.Hour,  // 连接生命周期
		   }

		4. 索引优化
		   - 为常查询的 JSON 字段创建虚拟列索引
		   - 定期分析慢查询日志
		   - 使用 EXPLAIN 分析查询计划

		5. 分区表（大数据量）
		   ALTER TABLE session_events
		   PARTITION BY RANGE (UNIX_TIMESTAMP(timestamp)) (
		       PARTITION p202501 VALUES LESS THAN (UNIX_TIMESTAMP('2025-02-01')),
		       PARTITION p202502 VALUES LESS THAN (UNIX_TIMESTAMP('2025-03-01')),
		       PARTITION p202503 VALUES LESS THAN (UNIX_TIMESTAMP('2025-04-01'))
		   );

		6. 备份策略
		   - 定期全量备份
		   - 启用二进制日志（binlog）进行增量备份
		   - 测试恢复流程

		7. 监控指标
		   - 连接池使用率
		   - 慢查询数量
		   - 表大小和增长趋势
		   - JSON 列查询性能

		8. 安全性
		   - 使用 SSL 连接: ...&tls=true
		   - 最小权限原则
		   - 定期更新密码
	*/
}

// PostgreSQL vs MySQL 对比

var _ = func() {
	/*
		特性对比:

		1. JSON 支持
		   PostgreSQL: JSONB (二进制存储，更快)
		   MySQL:      JSON (原生类型，8.0+)

		2. 数组支持
		   PostgreSQL: 原生数组类型 (TEXT[], INT[])
		   MySQL:      JSON 数组

		3. UUID
		   PostgreSQL: 原生 UUID 类型
		   MySQL:      VARCHAR(36) 存储

		4. 全文搜索
		   PostgreSQL: 内置全文搜索 (tsvector, tsquery)
		   MySQL:      FULLTEXT 索引

		5. 并发性能
		   PostgreSQL: MVCC (多版本并发控制)
		   MySQL:      行级锁 (InnoDB)

		选择建议:
		- 复杂 JSON 查询: PostgreSQL JSONB 更优
		- 简单 JSON 存储: MySQL 足够
		- 已有 MySQL 基础设施: MySQL
		- 需要高级特性: PostgreSQL
	*/
}
