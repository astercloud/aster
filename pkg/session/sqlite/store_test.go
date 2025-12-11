package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/astercloud/aster/pkg/session"
	"github.com/astercloud/aster/pkg/types"
)

func TestService(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	svc, err := New(dbPath)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer func() { _ = svc.Close() }()

	ctx := context.Background()

	// Test Create
	t.Run("Create", func(t *testing.T) {
		sess, err := svc.Create(ctx, &session.CreateRequest{
			AppName: "test-app",
			UserID:  "user-1",
			AgentID: "agent-1",
			Metadata: map[string]any{
				"test": "value",
			},
		})
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		if sess.ID() == "" {
			t.Error("Session ID should not be empty")
		}
		if sess.AppName() != "test-app" {
			t.Errorf("Expected app name 'test-app', got %q", sess.AppName())
		}
		if sess.UserID() != "user-1" {
			t.Errorf("Expected user ID 'user-1', got %q", sess.UserID())
		}
	})

	// Test Get
	t.Run("Get", func(t *testing.T) {
		// Create a session first
		created, _ := svc.Create(ctx, &session.CreateRequest{
			AppName: "test-app",
			UserID:  "user-1",
			AgentID: "agent-1",
		})

		// Get the session
		sess, err := svc.Get(ctx, &session.GetRequest{
			AppName:   "test-app",
			UserID:    "user-1",
			SessionID: created.ID(),
		})
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if sess.ID() != created.ID() {
			t.Errorf("Expected ID %q, got %q", created.ID(), sess.ID())
		}
	})

	// Test Get not found
	t.Run("GetNotFound", func(t *testing.T) {
		_, err := svc.Get(ctx, &session.GetRequest{
			AppName:   "test-app",
			UserID:    "user-1",
			SessionID: "non-existent",
		})
		if err != session.ErrSessionNotFound {
			t.Errorf("Expected ErrSessionNotFound, got %v", err)
		}
	})

	// Test Update
	t.Run("Update", func(t *testing.T) {
		created, _ := svc.Create(ctx, &session.CreateRequest{
			AppName: "test-app",
			UserID:  "user-1",
			AgentID: "agent-1",
			Metadata: map[string]any{
				"original": "value",
			},
		})

		err := svc.Update(ctx, &session.UpdateRequest{
			SessionID: created.ID(),
			Metadata: map[string]any{
				"new": "data",
			},
		})
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		sess, _ := svc.Get(ctx, &session.GetRequest{
			AppName:   "test-app",
			UserID:    "user-1",
			SessionID: created.ID(),
		})

		meta := sess.Metadata()
		if meta["original"] != "value" {
			t.Error("Original metadata should be preserved")
		}
		if meta["new"] != "data" {
			t.Error("New metadata should be added")
		}
	})

	// Test List
	t.Run("List", func(t *testing.T) {
		// Create multiple sessions
		for i := 0; i < 3; i++ {
			_, _ = svc.Create(ctx, &session.CreateRequest{
				AppName: "list-app",
				UserID:  "list-user",
				AgentID: "agent-1",
			})
		}

		sessions, err := svc.List(ctx, &session.ListRequest{
			AppName: "list-app",
			UserID:  "list-user",
		})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}

		if len(sessions) < 3 {
			t.Errorf("Expected at least 3 sessions, got %d", len(sessions))
		}
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		created, _ := svc.Create(ctx, &session.CreateRequest{
			AppName: "test-app",
			UserID:  "user-1",
			AgentID: "agent-1",
		})

		err := svc.Delete(ctx, created.ID())
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		_, err = svc.Get(ctx, &session.GetRequest{
			AppName:   "test-app",
			UserID:    "user-1",
			SessionID: created.ID(),
		})
		if err != session.ErrSessionNotFound {
			t.Error("Session should be deleted")
		}
	})
}

func TestEvents(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	svc, err := New(dbPath)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer func() { _ = svc.Close() }()

	ctx := context.Background()

	// Create session
	sess, _ := svc.Create(ctx, &session.CreateRequest{
		AppName: "test-app",
		UserID:  "user-1",
		AgentID: "agent-1",
	})

	// Test AppendEvent
	t.Run("AppendEvent", func(t *testing.T) {
		event := &session.Event{
			InvocationID: "inv-1",
			AgentID:      "agent-1",
			Author:       "user",
			Content: types.Message{
				Role:    types.RoleUser,
				Content: "Hello",
			},
			Metadata: map[string]any{
				"test": "value",
			},
		}

		err := svc.AppendEvent(ctx, sess.ID(), event)
		if err != nil {
			t.Fatalf("AppendEvent failed: %v", err)
		}

		if event.ID == "" {
			t.Error("Event ID should be set")
		}
	})

	// Test GetEvents
	t.Run("GetEvents", func(t *testing.T) {
		events, err := svc.GetEvents(ctx, sess.ID(), nil)
		if err != nil {
			t.Fatalf("GetEvents failed: %v", err)
		}

		if len(events) == 0 {
			t.Error("Expected at least one event")
		}

		if events[0].Content.Content != "Hello" {
			t.Errorf("Expected content 'Hello', got %q", events[0].Content.Content)
		}
	})

	// Test Events interface
	t.Run("EventsInterface", func(t *testing.T) {
		events := sess.Events()

		if events.Len() == 0 {
			t.Error("Expected at least one event")
		}

		last := events.Last()
		if last == nil {
			t.Error("Last should return an event")
		}

		// Test iteration
		count := 0
		for range events.All() {
			count++
		}
		if count == 0 {
			t.Error("All should iterate over events")
		}
	})
}

func TestState(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	svc, err := New(dbPath)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer func() { _ = svc.Close() }()

	ctx := context.Background()

	// Create session
	sess, _ := svc.Create(ctx, &session.CreateRequest{
		AppName: "test-app",
		UserID:  "user-1",
		AgentID: "agent-1",
	})

	state := sess.State()

	// Test Set and Get
	t.Run("SetGet", func(t *testing.T) {
		err := state.Set("key1", "value1")
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		val, err := state.Get("key1")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if val != "value1" {
			t.Errorf("Expected 'value1', got %v", val)
		}
	})

	// Test Get not found
	t.Run("GetNotFound", func(t *testing.T) {
		_, err := state.Get("non-existent")
		if err != session.ErrStateKeyNotExist {
			t.Errorf("Expected ErrStateKeyNotExist, got %v", err)
		}
	})

	// Test Has
	t.Run("Has", func(t *testing.T) {
		_ = state.Set("exists", true)

		if !state.Has("exists") {
			t.Error("Has should return true for existing key")
		}
		if state.Has("not-exists") {
			t.Error("Has should return false for non-existing key")
		}
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		_ = state.Set("to-delete", "value")
		err := state.Delete("to-delete")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		if state.Has("to-delete") {
			t.Error("Key should be deleted")
		}
	})

	// Test All
	t.Run("All", func(t *testing.T) {
		_ = state.Set("all-1", "v1")
		_ = state.Set("all-2", "v2")

		count := 0
		for range state.All() {
			count++
		}

		if count < 2 {
			t.Errorf("Expected at least 2 keys, got %d", count)
		}
	})

	// Test UpdateState (batch)
	t.Run("UpdateState", func(t *testing.T) {
		err := svc.UpdateState(ctx, sess.ID(), map[string]any{
			"batch-1": "value1",
			"batch-2": "value2",
		})
		if err != nil {
			t.Fatalf("UpdateState failed: %v", err)
		}

		if !state.Has("batch-1") || !state.Has("batch-2") {
			t.Error("Batch state should be set")
		}
	})
}

func TestDatabasePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "persist.db")
	ctx := context.Background()

	// Create and populate
	svc1, _ := New(dbPath)
	sess, _ := svc1.Create(ctx, &session.CreateRequest{
		AppName: "persist-app",
		UserID:  "user-1",
		AgentID: "agent-1",
	})
	sessionID := sess.ID()

	_ = svc1.AppendEvent(ctx, sessionID, &session.Event{
		InvocationID: "inv-1",
		Author:       "user",
		Content: types.Message{
			Role:    types.RoleUser,
			Content: "Persisted message",
		},
	})
	_ = svc1.Close()

	// Reopen and verify
	svc2, _ := New(dbPath)
	defer func() { _ = svc2.Close() }()

	sess2, err := svc2.Get(ctx, &session.GetRequest{
		AppName:   "persist-app",
		UserID:    "user-1",
		SessionID: sessionID,
	})
	if err != nil {
		t.Fatalf("Session should persist: %v", err)
	}

	events, _ := svc2.GetEvents(ctx, sess2.ID(), nil)
	if len(events) == 0 {
		t.Error("Events should persist")
	}
	if events[0].Content.Content != "Persisted message" {
		t.Error("Event content should persist")
	}
}

func TestEventFiltering(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "filter.db")
	ctx := context.Background()

	svc, _ := New(dbPath)
	defer func() { _ = svc.Close() }()

	sess, _ := svc.Create(ctx, &session.CreateRequest{
		AppName: "test-app",
		UserID:  "user-1",
		AgentID: "agent-1",
	})

	// Add events with different properties
	_ = svc.AppendEvent(ctx, sess.ID(), &session.Event{
		AgentID: "agent-1",
		Author:  "user",
		Content: types.Message{Role: types.RoleUser, Content: "msg1"},
	})
	time.Sleep(10 * time.Millisecond)
	_ = svc.AppendEvent(ctx, sess.ID(), &session.Event{
		AgentID: "agent-2",
		Author:  "assistant",
		Content: types.Message{Role: types.RoleAssistant, Content: "msg2"},
	})

	// Filter by agent
	t.Run("FilterByAgent", func(t *testing.T) {
		events, _ := svc.GetEvents(ctx, sess.ID(), &session.EventFilter{
			AgentID: "agent-1",
		})
		if len(events) != 1 {
			t.Errorf("Expected 1 event, got %d", len(events))
		}
	})

	// Filter by author
	t.Run("FilterByAuthor", func(t *testing.T) {
		events, _ := svc.GetEvents(ctx, sess.ID(), &session.EventFilter{
			Author: "user",
		})
		if len(events) != 1 {
			t.Errorf("Expected 1 event, got %d", len(events))
		}
	})

	// Filter with limit
	t.Run("FilterWithLimit", func(t *testing.T) {
		events, _ := svc.GetEvents(ctx, sess.ID(), &session.EventFilter{
			Limit: 1,
		})
		if len(events) != 1 {
			t.Errorf("Expected 1 event, got %d", len(events))
		}
	})
}

func TestDatabaseFileCreation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "subdir", "test.db")

	// Should fail because parent directory doesn't exist
	_, err := New(dbPath)
	if err == nil {
		t.Error("Should fail when parent directory doesn't exist")
	}

	// Create parent directory
	_ = os.MkdirAll(filepath.Dir(dbPath), 0755)

	svc, err := New(dbPath)
	if err != nil {
		t.Fatalf("New should succeed after creating parent dir: %v", err)
	}
	_ = svc.Close()

	// Verify file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file should be created")
	}
}
