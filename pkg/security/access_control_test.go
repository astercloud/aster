package security

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// mockAuditLog 用于测试的审计日志
type mockAuditLog struct {
	events []AuditEvent
}

func (m *mockAuditLog) LogEvent(event AuditEvent) error {
	m.events = append(m.events, event)
	return nil
}

func (m *mockAuditLog) LogEventAsync(event AuditEvent) error {
	m.events = append(m.events, event)
	return nil
}

func (m *mockAuditLog) LogEvents(events []AuditEvent) error {
	m.events = append(m.events, events...)
	return nil
}

func (m *mockAuditLog) QueryEvents(ctx context.Context, query *AuditQuery) (*AuditResult, error) {
	events := make([]*AuditEvent, len(m.events))
	for i := range m.events {
		events[i] = &m.events[i]
	}
	return &AuditResult{Events: events, Total: int64(len(events))}, nil
}

func (m *mockAuditLog) GetEvent(eventID string) (*AuditEvent, error) {
	for i := range m.events {
		if m.events[i].ID == eventID {
			return &m.events[i], nil
		}
	}
	return nil, fmt.Errorf("event not found")
}

func (m *mockAuditLog) GetEventsByUser(userID string, limit int) ([]*AuditEvent, error) {
	var result []*AuditEvent
	for i := range m.events {
		if m.events[i].UserID == userID {
			result = append(result, &m.events[i])
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockAuditLog) GetEventsByType(eventType AuditType, limit int) ([]*AuditEvent, error) {
	var result []*AuditEvent
	for i := range m.events {
		if m.events[i].Type == eventType {
			result = append(result, &m.events[i])
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockAuditLog) GetEventsByTimeRange(start, end time.Time, limit int) ([]*AuditEvent, error) {
	var result []*AuditEvent
	for i := range m.events {
		if m.events[i].Timestamp.After(start) && m.events[i].Timestamp.Before(end) {
			result = append(result, &m.events[i])
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockAuditLog) GetStatistics(ctx context.Context, filters *AuditFilters) (*AuditStatistics, error) {
	return &AuditStatistics{}, nil
}

func (m *mockAuditLog) GetEventSummary(timeRange TimeRange) (*EventSummary, error) {
	return &EventSummary{}, nil
}

func (m *mockAuditLog) ArchiveEvents(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}

func (m *mockAuditLog) PurgeEvents(ctx context.Context, before time.Time) (int64, error) {
	count := int64(0)
	newEvents := []AuditEvent{}
	for _, event := range m.events {
		if event.Timestamp.Before(before) {
			count++
		} else {
			newEvents = append(newEvents, event)
		}
	}
	m.events = newEvents
	return count, nil
}

func (m *mockAuditLog) ExportEvents(ctx context.Context, query *AuditQuery, format ExportFormat) ([]byte, error) {
	return []byte("{}"), nil
}

func (m *mockAuditLog) GetConfiguration() *AuditConfiguration {
	return &AuditConfiguration{}
}

func (m *mockAuditLog) UpdateConfiguration(config *AuditConfiguration) error {
	return nil
}

func (m *mockAuditLog) GetStatus() *AuditLogStatus {
	return &AuditLogStatus{}
}

func (m *mockAuditLog) Close() error {
	return nil
}

func TestNewAccessController(t *testing.T) {
	config := &AccessControlConfig{
		SessionTimeout:     30 * time.Minute,
		MaxSessionsPerUser: 5,
		EnableAudit:        true,
	}

	auditLog := &mockAuditLog{}
	ac := NewAccessController(config, auditLog)

	if ac == nil {
		t.Fatal("expected AccessController, got nil")
	}

	if ac.config.SessionTimeout != 30*time.Minute {
		t.Errorf("expected SessionTimeout 30m, got %v", ac.config.SessionTimeout)
	}
}

func TestAccessController_CreateUser(t *testing.T) {
	ac := NewAccessController(&AccessControlConfig{}, &mockAuditLog{})

	user := &User{
		ID:       "user1",
		Username: "testuser",
		Email:    "test@example.com",
		Status:   UserStatusActive,
	}

	err := ac.CreateUser(user)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// 验证用户已创建
	retrieved, err := ac.GetUser("user1")
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}

	if retrieved.Username != "testuser" {
		t.Errorf("expected username 'testuser', got '%s'", retrieved.Username)
	}
}

func TestAccessController_CreateUser_Duplicate(t *testing.T) {
	ac := NewAccessController(&AccessControlConfig{}, &mockAuditLog{})

	user := &User{
		ID:       "user1",
		Username: "testuser",
		Status:   UserStatusActive,
	}

	// 第一次创建应该成功
	err := ac.CreateUser(user)
	if err != nil {
		t.Fatalf("first CreateUser failed: %v", err)
	}

	// 第二次创建相同ID应该失败
	err = ac.CreateUser(user)
	if err == nil {
		t.Error("expected error for duplicate user, got nil")
	}
}

func TestAccessController_GetUser_NotFound(t *testing.T) {
	ac := NewAccessController(&AccessControlConfig{}, &mockAuditLog{})

	_, err := ac.GetUser("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent user, got nil")
	}
}

func TestAccessController_AssignRole(t *testing.T) {
	ac := NewAccessController(&AccessControlConfig{}, &mockAuditLog{})

	// 创建用户
	user := &User{
		ID:       "user1",
		Username: "testuser",
		Status:   UserStatusActive,
	}
	_ = ac.CreateUser(user)

	// 创建角色
	role := &Role{
		ID:          "role1",
		Name:        "Admin",
		Description: "Administrator role",
	}
	ac.roles = map[string]*Role{"role1": role}

	// 分配角色
	err := ac.AssignRole("user1", "role1")
	if err != nil {
		t.Fatalf("AssignRole failed: %v", err)
	}

	// 验证角色已分配
	userRoles := ac.userRoles["user1"]
	if len(userRoles) != 1 || userRoles[0] != "role1" {
		t.Errorf("expected user to have role 'role1', got %v", userRoles)
	}
}

func TestAccessController_RevokeRole(t *testing.T) {
	ac := NewAccessController(&AccessControlConfig{}, &mockAuditLog{})

	// 创建用户
	user := &User{
		ID:       "user1",
		Username: "testuser",
		Status:   UserStatusActive,
	}
	_ = ac.CreateUser(user)

	// 创建并分配角色
	role := &Role{
		ID:   "role1",
		Name: "Admin",
	}
	ac.roles = map[string]*Role{"role1": role}
	_ = ac.AssignRole("user1", "role1")

	// 撤销角色
	err := ac.RevokeRole("user1", "role1")
	if err != nil {
		t.Fatalf("RevokeRole failed: %v", err)
	}

	// 验证角色已撤销
	userRoles := ac.userRoles["user1"]
	if len(userRoles) != 0 {
		t.Errorf("expected user to have no roles, got %v", userRoles)
	}
}

func TestAccessController_CheckPermission(t *testing.T) {
	ac := NewAccessController(&AccessControlConfig{}, &mockAuditLog{})

	// 创建用户
	user := &User{
		ID:       "user1",
		Username: "testuser",
		Status:   UserStatusActive,
		Enabled:  true,
	}
	_ = ac.CreateUser(user)

	// 创建权限
	perm := &Permission{
		ID:       "perm1",
		Resource: "document",
		Action:   "read",
		Enabled:  true,
	}
	ac.permissions = map[string]*Permission{"perm1": perm}

	// 创建角色并添加权限
	role := &Role{
		ID:          "role1",
		Name:        "Reader",
		Permissions: []string{"perm1"},
	}
	ac.roles = map[string]*Role{"role1": role}
	ac.rolePermissions = map[string][]string{"role1": {"perm1"}}

	// 分配角色给用户
	_ = ac.AssignRole("user1", "role1")

	// 检查权限
	decision, err := ac.CheckPermission("user1", "document", "read", nil)
	if err != nil {
		t.Fatalf("CheckPermission failed: %v", err)
	}

	if !decision.Allowed {
		t.Error("expected permission to be allowed")
	}
}

func TestAccessController_CheckPermission_Denied(t *testing.T) {
	ac := NewAccessController(&AccessControlConfig{}, &mockAuditLog{})

	// 创建用户（没有任何权限）
	user := &User{
		ID:       "user1",
		Username: "testuser",
		Status:   UserStatusActive,
	}
	_ = ac.CreateUser(user)

	// 检查权限
	decision, err := ac.CheckPermission("user1", "document", "delete", nil)
	if err != nil {
		t.Fatalf("CheckPermission failed: %v", err)
	}

	if decision.Allowed {
		t.Error("expected permission to be denied")
	}
}

func TestAccessController_DeleteUser(t *testing.T) {
	ac := NewAccessController(&AccessControlConfig{}, &mockAuditLog{})

	// 创建用户
	user := &User{
		ID:       "user1",
		Username: "testuser",
		Status:   UserStatusActive,
	}
	_ = ac.CreateUser(user)

	// 删除用户
	err := ac.DeleteUser("user1")
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	// 验证用户已删除
	_, err = ac.GetUser("user1")
	if err == nil {
		t.Error("expected error for deleted user, got nil")
	}
}

func TestAccessController_UpdateUser(t *testing.T) {
	ac := NewAccessController(&AccessControlConfig{}, &mockAuditLog{})

	// 创建用户
	user := &User{
		ID:       "user1",
		Username: "testuser",
		Email:    "old@example.com",
		Status:   UserStatusActive,
	}
	_ = ac.CreateUser(user)

	// 更新用户
	user.Email = "new@example.com"
	err := ac.UpdateUser(user)
	if err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}

	// 验证更新
	retrieved, _ := ac.GetUser("user1")
	if retrieved.Email != "new@example.com" {
		t.Errorf("expected email 'new@example.com', got '%s'", retrieved.Email)
	}
}

func TestAccessController_ConcurrentAccess(t *testing.T) {
	ac := NewAccessController(&AccessControlConfig{}, &mockAuditLog{})

	// 并发创建多个用户
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			user := &User{
				ID:       fmt.Sprintf("user%d", id),
				Username: fmt.Sprintf("testuser%d", id),
				Status:   UserStatusActive,
			}
			_ = ac.CreateUser(user)
			done <- true
		}(i)
	}

	// 等待所有操作完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证所有用户都创建成功
	for i := 0; i < 10; i++ {
		_, err := ac.GetUser(fmt.Sprintf("user%d", i))
		if err != nil {
			t.Errorf("user%d not found", i)
		}
	}
}
