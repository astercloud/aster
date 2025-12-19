// Package sqlite provides a SQLite-based implementation of the session.Service interface.
// This is ideal for desktop applications and single-user scenarios where a lightweight,
// file-based database is preferred over PostgreSQL or MySQL.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"maps"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/session"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// Service implements session.Service using SQLite.
type Service struct {
	db *sql.DB
	mu sync.RWMutex
}

// New creates a new SQLite session service.
// dbPath is the path to the SQLite database file.
// If the file doesn't exist, it will be created.
func New(dbPath string) (*Service, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	// Set connection pool settings for SQLite
	db.SetMaxOpenConns(1) // SQLite only supports one writer
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	s := &Service{db: db}

	if err := s.migrate(); err != nil {
		_ = db.Close() // Ignore close error, migration error is more important
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	return s, nil
}

// migrate creates the necessary tables if they don't exist.
func (s *Service) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		app_name TEXT NOT NULL,
		user_id TEXT NOT NULL,
		agent_id TEXT NOT NULL,
		metadata TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_app_user ON sessions(app_name, user_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_updated ON sessions(updated_at DESC);

	CREATE TABLE IF NOT EXISTS events (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		invocation_id TEXT,
		agent_id TEXT,
		branch TEXT,
		author TEXT,
		content TEXT,
		reasoning TEXT,
		actions TEXT,
		long_running_tool_ids TEXT,
		metadata TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_events_session ON events(session_id, created_at);
	CREATE INDEX IF NOT EXISTS idx_events_invocation ON events(invocation_id);

	CREATE TABLE IF NOT EXISTS session_state (
		session_id TEXT NOT NULL,
		key TEXT NOT NULL,
		value TEXT,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (session_id, key),
		FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
	);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Close closes the database connection.
func (s *Service) Close() error {
	return s.db.Close()
}

// Create creates a new session.
func (s *Service) Create(ctx context.Context, req *session.CreateRequest) (session.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := uuid.New().String()
	now := time.Now()

	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, app_name, user_id, agent_id, metadata, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, req.AppName, req.UserID, req.AgentID, string(metadata), now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert session: %w", err)
	}

	return &sqliteSession{
		service:        s,
		id:             id,
		appName:        req.AppName,
		userID:         req.UserID,
		agentID:        req.AgentID,
		metadata:       req.Metadata,
		lastUpdateTime: now,
	}, nil
}

// Get retrieves a session by ID.
func (s *Service) Get(ctx context.Context, req *session.GetRequest) (session.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sess sqliteSession
	var metadataJSON string
	var createdAt, updatedAt time.Time

	err := s.db.QueryRowContext(ctx,
		`SELECT id, app_name, user_id, agent_id, metadata, created_at, updated_at
		 FROM sessions WHERE id = ? AND app_name = ? AND user_id = ?`,
		req.SessionID, req.AppName, req.UserID,
	).Scan(&sess.id, &sess.appName, &sess.userID, &sess.agentID, &metadataJSON, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, session.ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query session: %w", err)
	}

	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &sess.metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}

	sess.service = s
	sess.lastUpdateTime = updatedAt

	return &sess, nil
}

// Update updates a session's metadata.
func (s *Service) Update(ctx context.Context, req *session.UpdateRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get existing metadata
	var existingJSON string
	err := s.db.QueryRowContext(ctx,
		`SELECT metadata FROM sessions WHERE id = ?`,
		req.SessionID,
	).Scan(&existingJSON)

	if errors.Is(err, sql.ErrNoRows) {
		return session.ErrSessionNotFound
	}
	if err != nil {
		return fmt.Errorf("query session: %w", err)
	}

	// Merge metadata
	existing := make(map[string]any)
	if existingJSON != "" {
		if err := json.Unmarshal([]byte(existingJSON), &existing); err != nil {
			return fmt.Errorf("unmarshal existing metadata: %w", err)
		}
	}

	maps.Copy(existing, req.Metadata)

	newJSON, err := json.Marshal(existing)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	_, err = s.db.ExecContext(ctx,
		`UPDATE sessions SET metadata = ?, updated_at = ? WHERE id = ?`,
		string(newJSON), time.Now(), req.SessionID,
	)
	return err
}

// Delete deletes a session and all its events.
func (s *Service) Delete(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, sessionID)
	return err
}

// List lists sessions for an app and user.
func (s *Service) List(ctx context.Context, req *session.ListRequest) ([]*session.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, app_name, user_id, agent_id, metadata, created_at, updated_at
			  FROM sessions WHERE app_name = ? AND user_id = ?
			  ORDER BY updated_at DESC`

	args := []any{req.AppName, req.UserID}

	if req.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, req.Limit)
	}
	if req.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, req.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query sessions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []*session.Session
	for rows.Next() {
		var sess sqliteSession
		var metadataJSON string
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&sess.id, &sess.appName, &sess.userID, &sess.agentID, &metadataJSON, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}

		if metadataJSON != "" {
			if err := json.Unmarshal([]byte(metadataJSON), &sess.metadata); err != nil {
				return nil, fmt.Errorf("unmarshal metadata: %w", err)
			}
		}

		sess.service = s
		sess.lastUpdateTime = updatedAt
		var iface session.Session = &sess
		results = append(results, &iface)
	}

	return results, rows.Err()
}

// AppendEvent adds an event to a session.
func (s *Service) AppendEvent(ctx context.Context, sessionID string, event *session.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check session exists
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM sessions WHERE id = ?`, sessionID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return session.ErrSessionNotFound
	}
	if err != nil {
		return fmt.Errorf("check session: %w", err)
	}

	// Serialize fields
	contentJSON, err := json.Marshal(event.Content)
	if err != nil {
		return fmt.Errorf("marshal content: %w", err)
	}

	actionsJSON, err := json.Marshal(event.Actions)
	if err != nil {
		return fmt.Errorf("marshal actions: %w", err)
	}

	toolIDsJSON, err := json.Marshal(event.LongRunningToolIDs)
	if err != nil {
		return fmt.Errorf("marshal tool ids: %w", err)
	}

	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	now := time.Now()
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO events (id, session_id, invocation_id, agent_id, branch, author, content, reasoning, actions, long_running_tool_ids, metadata, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID, sessionID, event.InvocationID, event.AgentID, event.Branch, event.Author,
		string(contentJSON), event.Reasoning, string(actionsJSON), string(toolIDsJSON), string(metadataJSON), now,
	)
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}

	// Update session timestamp
	_, err = s.db.ExecContext(ctx, `UPDATE sessions SET updated_at = ? WHERE id = ?`, now, sessionID)
	return err
}

// GetEvents retrieves events for a session.
func (s *Service) GetEvents(ctx context.Context, sessionID string, filter *session.EventFilter) ([]session.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, invocation_id, agent_id, branch, author, content, reasoning, actions, long_running_tool_ids, metadata, created_at
			  FROM events WHERE session_id = ?`
	args := []any{sessionID}

	if filter != nil {
		if filter.AgentID != "" {
			query += " AND agent_id = ?"
			args = append(args, filter.AgentID)
		}
		if filter.Branch != "" {
			query += " AND branch = ?"
			args = append(args, filter.Branch)
		}
		if filter.Author != "" {
			query += " AND author = ?"
			args = append(args, filter.Author)
		}
		if filter.StartTime != nil {
			query += " AND created_at >= ?"
			args = append(args, *filter.StartTime)
		}
		if filter.EndTime != nil {
			query += " AND created_at <= ?"
			args = append(args, *filter.EndTime)
		}
	}

	query += " ORDER BY created_at ASC"

	if filter != nil {
		if filter.Limit > 0 {
			query += " LIMIT ?"
			args = append(args, filter.Limit)
		}
		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var events []session.Event
	for rows.Next() {
		var evt session.Event
		var contentJSON, actionsJSON, toolIDsJSON, metadataJSON string
		var createdAt time.Time

		if err := rows.Scan(
			&evt.ID, &evt.InvocationID, &evt.AgentID, &evt.Branch, &evt.Author,
			&contentJSON, &evt.Reasoning, &actionsJSON, &toolIDsJSON, &metadataJSON, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}

		evt.Timestamp = createdAt

		if contentJSON != "" {
			if err := json.Unmarshal([]byte(contentJSON), &evt.Content); err != nil {
				return nil, fmt.Errorf("unmarshal content: %w", err)
			}
		}
		if actionsJSON != "" {
			if err := json.Unmarshal([]byte(actionsJSON), &evt.Actions); err != nil {
				return nil, fmt.Errorf("unmarshal actions: %w", err)
			}
		}
		if toolIDsJSON != "" {
			if err := json.Unmarshal([]byte(toolIDsJSON), &evt.LongRunningToolIDs); err != nil {
				return nil, fmt.Errorf("unmarshal tool ids: %w", err)
			}
		}
		if metadataJSON != "" {
			if err := json.Unmarshal([]byte(metadataJSON), &evt.Metadata); err != nil {
				return nil, fmt.Errorf("unmarshal metadata: %w", err)
			}
		}

		events = append(events, evt)
	}

	return events, rows.Err()
}

// UpdateState updates session state.
func (s *Service) UpdateState(ctx context.Context, sessionID string, delta map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // Will be a no-op if tx.Commit() succeeds

	now := time.Now()
	for key, value := range delta {
		valueJSON, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("marshal value: %w", err)
		}

		_, err = tx.ExecContext(ctx,
			`INSERT INTO session_state (session_id, key, value, updated_at)
			 VALUES (?, ?, ?, ?)
			 ON CONFLICT (session_id, key) DO UPDATE SET value = ?, updated_at = ?`,
			sessionID, key, string(valueJSON), now, string(valueJSON), now,
		)
		if err != nil {
			return fmt.Errorf("upsert state: %w", err)
		}
	}

	// Update session timestamp
	_, err = tx.ExecContext(ctx, `UPDATE sessions SET updated_at = ? WHERE id = ?`, now, sessionID)
	if err != nil {
		return fmt.Errorf("update session: %w", err)
	}

	return tx.Commit()
}

// sqliteSession implements session.Session
type sqliteSession struct {
	service        *Service
	id             string
	appName        string
	userID         string
	agentID        string
	metadata       map[string]any
	lastUpdateTime time.Time
}

func (s *sqliteSession) ID() string                { return s.id }
func (s *sqliteSession) AppName() string           { return s.appName }
func (s *sqliteSession) UserID() string            { return s.userID }
func (s *sqliteSession) AgentID() string           { return s.agentID }
func (s *sqliteSession) LastUpdateTime() time.Time { return s.lastUpdateTime }
func (s *sqliteSession) Metadata() map[string]any  { return s.metadata }

func (s *sqliteSession) State() session.State {
	return &sqliteState{service: s.service, sessionID: s.id}
}

func (s *sqliteSession) Events() session.Events {
	return &sqliteEvents{service: s.service, sessionID: s.id}
}

// sqliteState implements session.State
type sqliteState struct {
	service   *Service
	sessionID string
}

func (s *sqliteState) Get(key string) (any, error) {
	s.service.mu.RLock()
	defer s.service.mu.RUnlock()

	var valueJSON string
	err := s.service.db.QueryRow(
		`SELECT value FROM session_state WHERE session_id = ? AND key = ?`,
		s.sessionID, key,
	).Scan(&valueJSON)

	if err == sql.ErrNoRows {
		return nil, session.ErrStateKeyNotExist
	}
	if err != nil {
		return nil, err
	}

	var value any
	if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
		return nil, err
	}

	return value, nil
}

func (s *sqliteState) Set(key string, value any) error {
	s.service.mu.Lock()
	defer s.service.mu.Unlock()

	valueJSON, err := json.Marshal(value)
	if err != nil {
		return err
	}

	_, err = s.service.db.Exec(
		`INSERT INTO session_state (session_id, key, value, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT (session_id, key) DO UPDATE SET value = ?, updated_at = ?`,
		s.sessionID, key, string(valueJSON), time.Now(), string(valueJSON), time.Now(),
	)
	return err
}

func (s *sqliteState) Delete(key string) error {
	s.service.mu.Lock()
	defer s.service.mu.Unlock()

	_, err := s.service.db.Exec(
		`DELETE FROM session_state WHERE session_id = ? AND key = ?`,
		s.sessionID, key,
	)
	return err
}

func (s *sqliteState) All() iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		s.service.mu.RLock()
		defer s.service.mu.RUnlock()

		rows, err := s.service.db.Query(
			`SELECT key, value FROM session_state WHERE session_id = ?`,
			s.sessionID,
		)
		if err != nil {
			return
		}
		defer func() { _ = rows.Close() }()

		for rows.Next() {
			var key, valueJSON string
			if err := rows.Scan(&key, &valueJSON); err != nil {
				return
			}

			var value any
			if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
				return
			}

			if !yield(key, value) {
				return
			}
		}
	}
}

func (s *sqliteState) Has(key string) bool {
	s.service.mu.RLock()
	defer s.service.mu.RUnlock()

	var exists bool
	_ = s.service.db.QueryRow(
		`SELECT 1 FROM session_state WHERE session_id = ? AND key = ?`,
		s.sessionID, key,
	).Scan(&exists) // Ignore error, returns false on error
	return exists
}

// sqliteEvents implements session.Events
type sqliteEvents struct {
	service   *Service
	sessionID string
}

func (e *sqliteEvents) All() iter.Seq[*session.Event] {
	return func(yield func(*session.Event) bool) {
		events, err := e.service.GetEvents(context.Background(), e.sessionID, nil)
		if err != nil {
			return
		}

		for i := range events {
			if !yield(&events[i]) {
				return
			}
		}
	}
}

func (e *sqliteEvents) Len() int {
	e.service.mu.RLock()
	defer e.service.mu.RUnlock()

	var count int
	_ = e.service.db.QueryRow(
		`SELECT COUNT(*) FROM events WHERE session_id = ?`,
		e.sessionID,
	).Scan(&count) // Ignore error, returns 0 on error
	return count
}

func (e *sqliteEvents) At(i int) *session.Event {
	events, err := e.service.GetEvents(context.Background(), e.sessionID, &session.EventFilter{
		Limit:  1,
		Offset: i,
	})
	if err != nil || len(events) == 0 {
		return nil
	}
	return &events[0]
}

func (e *sqliteEvents) Filter(predicate func(*session.Event) bool) []session.Event {
	events, err := e.service.GetEvents(context.Background(), e.sessionID, nil)
	if err != nil {
		return nil
	}

	var result []session.Event
	for _, evt := range events {
		if predicate(&evt) {
			result = append(result, evt)
		}
	}
	return result
}

func (e *sqliteEvents) Last() *session.Event {
	e.service.mu.RLock()
	defer e.service.mu.RUnlock()

	var evt session.Event
	var contentJSON, actionsJSON, toolIDsJSON, metadataJSON string
	var createdAt time.Time

	err := e.service.db.QueryRow(
		`SELECT id, invocation_id, agent_id, branch, author, content, reasoning, actions, long_running_tool_ids, metadata, created_at
		 FROM events WHERE session_id = ? ORDER BY created_at DESC LIMIT 1`,
		e.sessionID,
	).Scan(
		&evt.ID, &evt.InvocationID, &evt.AgentID, &evt.Branch, &evt.Author,
		&contentJSON, &evt.Reasoning, &actionsJSON, &toolIDsJSON, &metadataJSON, &createdAt,
	)

	if err != nil {
		return nil
	}

	evt.Timestamp = createdAt
	_ = json.Unmarshal([]byte(contentJSON), &evt.Content)
	_ = json.Unmarshal([]byte(actionsJSON), &evt.Actions)
	_ = json.Unmarshal([]byte(toolIDsJSON), &evt.LongRunningToolIDs)
	_ = json.Unmarshal([]byte(metadataJSON), &evt.Metadata)

	return &evt
}

// Verify Service implements session.Service
var _ session.Service = (*Service)(nil)
