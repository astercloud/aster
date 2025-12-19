package security

import (
	"context"
	"testing"
	"time"
)

func TestNewInMemoryAuditLog(t *testing.T) {
	auditLog := NewInMemoryAuditLog(nil)

	if auditLog == nil {
		t.Fatal("expected InMemoryAuditLog, got nil")
	}

	if auditLog.config.StorageType != StorageTypeMemory {
		t.Errorf("expected StorageType Memory, got %v", auditLog.config.StorageType)
	}

	// 清理
	_ = auditLog.Close()
}

func TestInMemoryAuditLog_LogEvent(t *testing.T) {
	auditLog := NewInMemoryAuditLog(nil)
	defer func() { _ = auditLog.Close() }()

	event := AuditEvent{
		Type:      AuditTypeUserLogin,
		UserID:    "user1",
		Message:   "User logged in",
		Timestamp: time.Now(),
	}

	err := auditLog.LogEvent(event)
	if err != nil {
		t.Fatalf("LogEvent failed: %v", err)
	}

	// 使用GetStatus而不是直接访问events字段
	status := auditLog.GetStatus()
	if status.TotalEvents != 1 {
		t.Errorf("expected 1 event, got %d", status.TotalEvents)
	}

	// 使用GetEventsByUser验证用户ID
	events, _ := auditLog.GetEventsByUser("user1", 10)
	if len(events) > 0 && events[0].UserID != "user1" {
		t.Errorf("expected UserID 'user1', got '%s'", events[0].UserID)
	}
}

func TestInMemoryAuditLog_LogEventAsync(t *testing.T) {
	auditLog := NewInMemoryAuditLog(nil)
	defer func() { _ = auditLog.Close() }()

	event := AuditEvent{
		Type:    AuditTypeUserLogin,
		UserID:  "user1",
		Message: "User logged in",
	}

	err := auditLog.LogEventAsync(event)
	if err != nil {
		t.Fatalf("LogEventAsync failed: %v", err)
	}

	// 等待异步处理完成
	time.Sleep(100 * time.Millisecond)

	// 使用GetEventsByUser而不是直接访问events字段
	events, err := auditLog.GetEventsByUser("user1", 10)
	if err != nil {
		t.Fatalf("GetEventsByUser failed: %v", err)
	}

	if len(events) == 0 {
		t.Error("expected event to be logged asynchronously")
	}
}

func TestInMemoryAuditLog_LogEvents(t *testing.T) {
	auditLog := NewInMemoryAuditLog(nil)
	defer func() { _ = auditLog.Close() }()

	events := []AuditEvent{
		{Type: AuditTypeUserLogin, UserID: "user1", Message: "Login 1"},
		{Type: AuditTypeUserLogout, UserID: "user2", Message: "Logout 1"},
		{Type: AuditTypeUserLogin, UserID: "user3", Message: "Login 2"},
	}

	err := auditLog.LogEvents(events)
	if err != nil {
		t.Fatalf("LogEvents failed: %v", err)
	}

	// 等待异步处理完成
	time.Sleep(200 * time.Millisecond)

	// 使用GetStatus而不是直接访问events字段
	status := auditLog.GetStatus()
	if status.TotalEvents != 3 {
		t.Errorf("expected 3 events, got %d", status.TotalEvents)
	}
}

func TestInMemoryAuditLog_GetEvent(t *testing.T) {
	auditLog := NewInMemoryAuditLog(nil)
	defer func() { _ = auditLog.Close() }()

	event := AuditEvent{
		ID:      "event1",
		Type:    AuditTypeUserLogin,
		UserID:  "user1",
		Message: "User logged in",
	}

	_ = auditLog.LogEvent(event)

	// 获取事件
	retrieved, err := auditLog.GetEvent("event1")
	if err != nil {
		t.Fatalf("GetEvent failed: %v", err)
	}

	if retrieved.UserID != "user1" {
		t.Errorf("expected UserID 'user1', got '%s'", retrieved.UserID)
	}
}

func TestInMemoryAuditLog_GetEvent_NotFound(t *testing.T) {
	auditLog := NewInMemoryAuditLog(nil)
	defer func() { _ = auditLog.Close() }()

	_, err := auditLog.GetEvent("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent event, got nil")
	}
}

func TestInMemoryAuditLog_GetEventsByUser(t *testing.T) {
	auditLog := NewInMemoryAuditLog(nil)
	defer func() { _ = auditLog.Close() }()

	// 添加多个用户的事件
	_ = auditLog.LogEvent(AuditEvent{Type: AuditTypeUserLogin, UserID: "user1", Message: "Login 1"})
	_ = auditLog.LogEvent(AuditEvent{Type: AuditTypeUserLogout, UserID: "user2", Message: "Logout 1"})
	_ = auditLog.LogEvent(AuditEvent{Type: AuditTypeUserLogin, UserID: "user1", Message: "Login 2"})

	// 获取user1的事件
	events, err := auditLog.GetEventsByUser("user1", 10)
	if err != nil {
		t.Fatalf("GetEventsByUser failed: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("expected 2 events for user1, got %d", len(events))
	}
}

func TestInMemoryAuditLog_GetEventsByType(t *testing.T) {
	auditLog := NewInMemoryAuditLog(nil)
	defer func() { _ = auditLog.Close() }()

	// 添加不同类型的事件
	_ = auditLog.LogEvent(AuditEvent{Type: AuditTypeUserLogin, UserID: "user1", Message: "Login 1"})
	_ = auditLog.LogEvent(AuditEvent{Type: AuditTypeUserLogout, UserID: "user2", Message: "Logout 1"})
	_ = auditLog.LogEvent(AuditEvent{Type: AuditTypeUserLogin, UserID: "user3", Message: "Login 2"})

	// 获取登录类型的事件
	events, err := auditLog.GetEventsByType(AuditTypeUserLogin, 10)
	if err != nil {
		t.Fatalf("GetEventsByType failed: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("expected 2 login events, got %d", len(events))
	}
}

func TestInMemoryAuditLog_GetEventsByTimeRange(t *testing.T) {
	auditLog := NewInMemoryAuditLog(nil)
	defer func() { _ = auditLog.Close() }()

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	tomorrow := now.Add(24 * time.Hour)

	// 添加不同时间的事件
	_ = auditLog.LogEvent(AuditEvent{
		Type:      AuditTypeUserLogin,
		UserID:    "user1",
		Message:   "Old login",
		Timestamp: yesterday.Add(-1 * time.Hour),
	})
	_ = auditLog.LogEvent(AuditEvent{
		Type:      AuditTypeUserLogin,
		UserID:    "user2",
		Message:   "Recent login",
		Timestamp: now,
	})

	// 获取昨天到明天之间的事件
	events, err := auditLog.GetEventsByTimeRange(yesterday, tomorrow, 10)
	if err != nil {
		t.Fatalf("GetEventsByTimeRange failed: %v", err)
	}

	// 应该只有一个在时间范围内的事件
	if len(events) != 1 {
		t.Errorf("expected 1 event in time range, got %d", len(events))
	}

	if len(events) > 0 && events[0].UserID != "user2" {
		t.Errorf("expected user2, got %s", events[0].UserID)
	}
}

func TestInMemoryAuditLog_PurgeEvents(t *testing.T) {
	auditLog := NewInMemoryAuditLog(nil)
	defer func() { _ = auditLog.Close() }()

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	// 添加旧事件和新事件
	_ = auditLog.LogEvent(AuditEvent{
		Type:      AuditTypeUserLogin,
		UserID:    "user1",
		Message:   "Old login",
		Timestamp: yesterday.Add(-1 * time.Hour),
	})
	_ = auditLog.LogEvent(AuditEvent{
		Type:      AuditTypeUserLogin,
		UserID:    "user2",
		Message:   "Recent login",
		Timestamp: now,
	})

	// 清除昨天之前的事件
	count, err := auditLog.PurgeEvents(context.Background(), yesterday)
	if err != nil {
		t.Fatalf("PurgeEvents failed: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 event purged, got %d", count)
	}

	// 使用GetStatus验证剩余事件数
	status := auditLog.GetStatus()
	if status.TotalEvents != 1 {
		t.Errorf("expected 1 event remaining, got %d", status.TotalEvents)
	}

	// 使用GetEventsByUser验证user2的事件还在
	events, _ := auditLog.GetEventsByUser("user2", 10)
	if len(events) == 0 {
		t.Error("expected user2 event to remain")
	} else if events[0].UserID != "user2" {
		t.Errorf("expected user2 to remain, got %s", events[0].UserID)
	}
}

func TestInMemoryAuditLog_GetStatus(t *testing.T) {
	auditLog := NewInMemoryAuditLog(nil)
	defer func() { _ = auditLog.Close() }()

	// 添加一些事件
	_ = auditLog.LogEvent(AuditEvent{Type: AuditTypeUserLogin, UserID: "user1"})
	_ = auditLog.LogEvent(AuditEvent{Type: AuditTypeUserLogout, UserID: "user2"})

	status := auditLog.GetStatus()
	if status == nil {
		t.Fatal("expected status, got nil")
	}

	if status.TotalEvents != 2 {
		t.Errorf("expected TotalEvents 2, got %d", status.TotalEvents)
	}
}

func TestInMemoryAuditLog_Close(t *testing.T) {
	auditLog := NewInMemoryAuditLog(nil)

	// 添加一个事件
	_ = auditLog.LogEvent(AuditEvent{Type: AuditTypeUserLogin, UserID: "user1"})

	// 关闭
	err := auditLog.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestInMemoryAuditLog_ConcurrentWrites(t *testing.T) {
	auditLog := NewInMemoryAuditLog(nil)
	defer func() { _ = auditLog.Close() }()

	// 并发写入事件
	done := make(chan bool)
	for i := range 10 {
		go func(id int) {
			event := AuditEvent{
				Type:    AuditTypeUserLogin,
				UserID:  "user" + string(rune('0'+id)),
				Message: "Concurrent login",
			}
			_ = auditLog.LogEvent(event)
			done <- true
		}(i)
	}

	// 等待所有操作完成
	for range 10 {
		<-done
	}

	// 使用GetStatus而不是直接访问events字段
	status := auditLog.GetStatus()
	if status.TotalEvents != 10 {
		t.Errorf("expected 10 events, got %d", status.TotalEvents)
	}
}
