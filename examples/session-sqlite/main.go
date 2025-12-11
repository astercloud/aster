// Session-SQLite 演示 SQLite 会话存储，适用于桌面应用和单机场景。
// 这是轻量级的本地持久化方案，使用 WAL 模式提高性能。
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/astercloud/aster/pkg/session"
	"github.com/astercloud/aster/pkg/session/sqlite"
	"github.com/astercloud/aster/pkg/types"
)

func main() {
	ctx := context.Background()

	fmt.Println("🗄️  SQLite Session Store 示例")
	fmt.Println("================================")

	// 1. 创建临时目录存储数据库
	tmpDir, err := os.MkdirTemp("", "aster-sqlite-demo")
	if err != nil {
		log.Fatalf("创建临时目录失败: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	dbPath := filepath.Join(tmpDir, "sessions.db")
	fmt.Printf("\n📁 数据库路径: %s\n", dbPath)

	// 2. 创建 SQLite Session 服务
	fmt.Println("\n✓ 创建 SQLite Session 服务...")
	service, err := sqlite.New(dbPath)
	if err != nil {
		log.Fatalf("创建 SQLite 服务失败: %v", err)
	}
	defer func() { _ = service.Close() }()

	// 3. 创建会话
	fmt.Println("\n✓ 创建会话...")
	sess, err := service.Create(ctx, &session.CreateRequest{
		AppName: "desktop-app",
		UserID:  "user-001",
		AgentID: "agent-001",
		Metadata: map[string]any{
			"platform": "macos",
			"version":  "1.0.0",
		},
	})
	if err != nil {
		log.Fatalf("创建会话失败: %v", err)
	}

	sessionID := sess.ID()
	fmt.Printf("  会话 ID: %s\n", sessionID)

	// 4. 添加事件/消息
	fmt.Println("\n✓ 添加消息...")
	messages := []struct {
		author    string
		content   string
		reasoning string
	}{
		{"user", "你好，请帮我分析这段代码。", ""},
		{"assistant", "好的，我来帮你分析代码。首先让我看一下...", "用户请求代码分析，需要先理解代码内容"},
		{"user", "代码在 main.go 文件中", ""},
	}

	for _, msg := range messages {
		event := session.NewEvent("inv-001")
		event.Author = msg.author
		event.Content = types.Message{
			Role:    types.Role(msg.author),
			Content: msg.content,
		}
		event.Reasoning = msg.reasoning

		err := service.AppendEvent(ctx, sessionID, event)
		if err != nil {
			log.Fatalf("添加消息失败: %v", err)
		}
		fmt.Printf("  添加: [%s] %s\n", msg.author, truncate(msg.content, 40))
	}

	// 5. 获取所有事件
	fmt.Println("\n✓ 读取所有事件...")
	events, err := service.GetEvents(ctx, sessionID, nil)
	if err != nil {
		log.Fatalf("获取事件失败: %v", err)
	}
	for i, event := range events {
		fmt.Printf("  #%d [%s] %s\n", i+1, event.Author, truncate(event.Content.Content, 50))
	}

	// 6. 创建第二个会话
	fmt.Println("\n✓ 创建第二个会话...")
	sess2, err := service.Create(ctx, &session.CreateRequest{
		AppName: "desktop-app",
		UserID:  "user-001",
		AgentID: "agent-002",
		Metadata: map[string]any{
			"task": "code-review",
		},
	})
	if err != nil {
		log.Fatalf("创建第二个会话失败: %v", err)
	}
	fmt.Printf("  会话 ID: %s\n", sess2.ID())

	// 添加一些消息
	event := session.NewEvent("inv-002")
	event.Author = "user"
	event.Content = types.Message{
		Role:    types.RoleUser,
		Content: "请帮我审查 PR #123",
	}
	_ = service.AppendEvent(ctx, sess2.ID(), event)

	// 7. 列出所有会话
	fmt.Println("\n✓ 列出所有会话...")
	sessions, err := service.List(ctx, &session.ListRequest{
		AppName: "desktop-app",
		UserID:  "user-001",
	})
	if err != nil {
		log.Fatalf("列出会话失败: %v", err)
	}

	fmt.Printf("  共 %d 个会话:\n", len(sessions))
	for _, s := range sessions {
		fmt.Printf("    - %s (Agent: %s)\n", (*s).ID(), (*s).AgentID())
	}

	// 8. 更新会话元数据
	fmt.Println("\n✓ 更新会话元数据...")
	err = service.Update(ctx, &session.UpdateRequest{
		SessionID: sessionID,
		Metadata: map[string]any{
			"platform": "macos",
			"version":  "1.0.0",
			"status":   "completed",
		},
	})
	if err != nil {
		log.Fatalf("更新会话失败: %v", err)
	}
	fmt.Println("  元数据已更新")

	// 9. 重新获取会话（模拟应用重启）
	fmt.Println("\n✓ 重新获取会话（模拟应用重启）...")
	reloadedSess, err := service.Get(ctx, &session.GetRequest{
		AppName:   "desktop-app",
		UserID:    "user-001",
		SessionID: sessionID,
	})
	if err != nil {
		log.Fatalf("获取会话失败: %v", err)
	}

	fmt.Printf("  会话 ID: %s\n", reloadedSess.ID())
	fmt.Printf("  Agent ID: %s\n", reloadedSess.AgentID())

	// 统计事件数量
	reloadedEvents, _ := service.GetEvents(ctx, reloadedSess.ID(), nil)
	fmt.Printf("  事件数量: %d\n", len(reloadedEvents))

	// 10. 删除会话
	fmt.Println("\n✓ 删除会话...")
	err = service.Delete(ctx, sess2.ID())
	if err != nil {
		log.Fatalf("删除会话失败: %v", err)
	}
	fmt.Printf("  已删除会话: %s\n", sess2.ID())

	// 验证删除
	remainingSessions, _ := service.List(ctx, &session.ListRequest{
		AppName: "desktop-app",
		UserID:  "user-001",
	})
	fmt.Printf("  剩余会话数: %d\n", len(remainingSessions))

	fmt.Println("\n✅ SQLite Session Store 示例完成!")
	fmt.Println("\n💡 提示:")
	fmt.Println("  - SQLite 适合桌面应用和单用户场景")
	fmt.Println("  - 使用 WAL 模式提高并发性能")
	fmt.Println("  - 数据持久化在本地文件中")
	fmt.Println("  - 支持与 PostgreSQL/MySQL 相同的接口")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
