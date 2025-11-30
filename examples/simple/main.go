// Simple 演示 Aster Agent SDK 的基础组件，包括 EventBus 事件总线、
// Sandbox 沙箱执行环境和 JSON Store 持久化存储。
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/astercloud/aster/pkg/events"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/types"
)

func main() {
	fmt.Println("🚀 Agent SDK - 简单示例")
	fmt.Println("================================")

	// 1. 创建EventBus
	fmt.Println("\n✓ 创建EventBus...")
	bus := events.NewEventBus()

	// 订阅Progress事件
	progressCh := bus.Subscribe([]types.AgentChannel{types.ChannelProgress}, nil)
	go func() {
		for event := range progressCh {
			fmt.Printf("  [Progress] Cursor=%d, Event=%T\n", event.Cursor, event.Event)
		}
	}()

	// 订阅Monitor事件
	monitorCh := bus.Subscribe([]types.AgentChannel{types.ChannelMonitor}, nil)
	go func() {
		for event := range monitorCh {
			fmt.Printf("  [Monitor] Cursor=%d, Event=%T\n", event.Cursor, event.Event)
		}
	}()

	// 发送一些测试事件
	bus.EmitProgress(&types.ProgressTextChunkStartEvent{Step: 1})
	bus.EmitProgress(&types.ProgressTextChunkEvent{Step: 1, Delta: "Hello"})
	bus.EmitProgress(&types.ProgressTextChunkEvent{Step: 1, Delta: " World"})
	bus.EmitMonitor(&types.MonitorStateChangedEvent{State: types.AgentStateWorking})

	time.Sleep(100 * time.Millisecond) // 等待事件处理

	// 2. 创建本地沙箱
	fmt.Println("\n✓ 创建本地沙箱...")
	sb, err := sandbox.NewLocalSandbox(&sandbox.LocalSandboxConfig{
		WorkDir:         "./workspace",
		EnforceBoundary: true,
		WatchFiles:      false,
	})
	if err != nil {
		log.Fatalf("创建沙箱失败: %v", err)
	}
	defer func() { _ = sb.Dispose() }()

	fmt.Printf("  Kind: %s\n", sb.Kind())
	fmt.Printf("  WorkDir: %s\n", sb.WorkDir())

	// 执行命令
	ctx := context.Background()
	result, err := sb.Exec(ctx, "echo 'Hello from sandbox'", nil)
	if err != nil {
		log.Fatalf("执行命令失败: %v", err)
	}
	fmt.Printf("  执行结果: code=%d, stdout=%s\n", result.Code, result.Stdout)

	// 文件系统操作
	fs := sb.FS()
	testPath := "test.txt"
	testContent := "Hello Agent SDK!"

	fmt.Println("\n✓ 文件系统测试...")
	if err := fs.Write(ctx, testPath, testContent); err != nil {
		log.Fatalf("写入文件失败: %v", err)
	}
	fmt.Printf("  写入文件: %s\n", testPath)

	content, err := fs.Read(ctx, testPath)
	if err != nil {
		log.Fatalf("读取文件失败: %v", err)
	}
	fmt.Printf("  读取内容: %s\n", content)

	// 3. 创建JSON存储
	fmt.Println("\n✓ 创建JSON存储...")
	jsonStore, err := store.NewJSONStore("./.aster-test")
	if err != nil {
		log.Fatalf("创建存储失败: %v", err)
	}

	// 保存测试数据
	agentID := "agt-test123"
	testMessages := []types.Message{
		{
			Role:    types.MessageRoleUser,
			Content: "Hello Agent!",
		},
	}

	if err := jsonStore.SaveMessages(ctx, agentID, testMessages); err != nil {
		log.Fatalf("保存消息失败: %v", err)
	}
	fmt.Printf("  保存消息: agentID=%s, count=%d\n", agentID, len(testMessages))

	// 加载数据
	loadedMessages, err := jsonStore.LoadMessages(ctx, agentID)
	if err != nil {
		log.Fatalf("加载消息失败: %v", err)
	}
	fmt.Printf("  加载消息: count=%d\n", len(loadedMessages))

	// 保存Agent信息
	agentInfo := types.AgentInfo{
		AgentID:       agentID,
		TemplateID:    "test-template",
		CreatedAt:     time.Now(),
		Lineage:       []string{},
		ConfigVersion: "v1.0.0",
		MessageCount:  len(testMessages),
	}

	if err := jsonStore.SaveInfo(ctx, agentID, agentInfo); err != nil {
		log.Fatalf("保存Info失败: %v", err)
	}
	fmt.Printf("  保存AgentInfo: templateID=%s\n", agentInfo.TemplateID)

	// 4. 测试沙箱工厂
	fmt.Println("\n✓ 测试沙箱工厂...")
	factory := sandbox.NewFactory()

	// 创建Mock沙箱
	mockSb, err := factory.Create(&types.SandboxConfig{
		Kind: types.SandboxKindMock,
	})
	if err != nil {
		log.Fatalf("创建Mock沙箱失败: %v", err)
	}
	fmt.Printf("  Mock沙箱: kind=%s\n", mockSb.Kind())

	mockResult, _ := mockSb.Exec(ctx, "test command", nil)
	fmt.Printf("  Mock执行结果: %s\n", mockResult.Stdout)

	// 5. EventBus高级功能测试
	fmt.Println("\n✓ 测试EventBus高级功能...")

	// 获取当前cursor
	cursor := bus.GetCursor()
	fmt.Printf("  当前Cursor: %d\n", cursor)

	// 获取最后的bookmark
	lastBookmark := bus.GetLastBookmark()
	if lastBookmark != nil {
		fmt.Printf("  最后Bookmark: cursor=%d, time=%d\n",
			lastBookmark.Cursor,
			lastBookmark.Timestamp)
	}

	// 获取时间线
	timeline := bus.GetTimeline()
	fmt.Printf("  时间线事件数: %d\n", len(timeline))

	fmt.Println("\n================================")
	fmt.Println("✅ 所有测试完成!")
}
