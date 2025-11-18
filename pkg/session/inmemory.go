package session

import (
	"context"
	"fmt"
	"iter"
	"sync"
	"time"

	"github.com/google/uuid"
)

// InMemoryService 内存实现的 Session 服务
// 适用于开发和测试环境
type InMemoryService struct {
	mu       sync.RWMutex
	sessions map[string]*inMemorySession
}

// NewInMemoryService 创建内存 Session 服务
func NewInMemoryService() *InMemoryService {
	return &InMemoryService{
		sessions: make(map[string]*inMemorySession),
	}
}

// Create 创建新会话
func (s *InMemoryService) Create(ctx context.Context, req *CreateRequest) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID := uuid.New().String()
	session := &inMemorySession{
		id:             sessionID,
		appName:        req.AppName,
		userID:         req.UserID,
		agentID:        req.AgentID,
		state:          newInMemoryState(),
		events:         newInMemoryEvents(),
		metadata:       req.Metadata,
		lastUpdateTime: time.Now(),
	}

	if session.metadata == nil {
		session.metadata = make(map[string]interface{})
	}

	s.sessions[sessionID] = session

	return session, nil
}

// Get 获取会话
func (s *InMemoryService) Get(ctx context.Context, req *GetRequest) (Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[req.SessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}

	if session.appName != req.AppName || session.userID != req.UserID {
		return nil, ErrSessionNotFound
	}

	return session, nil
}

// Update 更新会话
func (s *InMemoryService) Update(ctx context.Context, req *UpdateRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[req.SessionID]
	if !ok {
		return ErrSessionNotFound
	}

	if req.Metadata != nil {
		for k, v := range req.Metadata {
			session.metadata[k] = v
		}
	}

	session.lastUpdateTime = time.Now()
	return nil
}

// Delete 删除会话
func (s *InMemoryService) Delete(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, sessionID)
	return nil
}

// List 列出会话
func (s *InMemoryService) List(ctx context.Context, req *ListRequest) ([]*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*Session
	count := 0

	for _, session := range s.sessions {
		if session.appName != req.AppName || session.userID != req.UserID {
			continue
		}

		if count < req.Offset {
			count++
			continue
		}

		if req.Limit > 0 && len(results) >= req.Limit {
			break
		}

		var s Session = session
		results = append(results, &s)
		count++
	}

	return results, nil
}

// AppendEvent 添加事件
func (s *InMemoryService) AppendEvent(ctx context.Context, sessionID string, event *Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return ErrSessionNotFound
	}

	session.events.append(event)
	session.lastUpdateTime = time.Now()
	return nil
}

// GetEvents 获取事件列表
func (s *InMemoryService) GetEvents(ctx context.Context, sessionID string, filter *EventFilter) ([]Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}

	events := session.events.filter(func(e *Event) bool {
		if filter == nil {
			return true
		}

		if filter.AgentID != "" && e.AgentID != filter.AgentID {
			return false
		}

		if filter.Branch != "" && e.Branch != filter.Branch {
			return false
		}

		if filter.Author != "" && e.Author != filter.Author {
			return false
		}

		if filter.StartTime != nil && e.Timestamp.Before(*filter.StartTime) {
			return false
		}

		if filter.EndTime != nil && e.Timestamp.After(*filter.EndTime) {
			return false
		}

		return true
	})

	// 应用分页
	if filter != nil {
		start := filter.Offset
		if start > len(events) {
			return []Event{}, nil
		}

		end := len(events)
		if filter.Limit > 0 && start+filter.Limit < end {
			end = start + filter.Limit
		}

		events = events[start:end]
	}

	return events, nil
}

// UpdateState 更新状态
func (s *InMemoryService) UpdateState(ctx context.Context, sessionID string, delta map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return ErrSessionNotFound
	}

	for k, v := range delta {
		if err := session.state.Set(k, v); err != nil {
			return fmt.Errorf("set state %s: %w", k, err)
		}
	}

	session.lastUpdateTime = time.Now()
	return nil
}

// inMemorySession 内存会话实现
type inMemorySession struct {
	id             string
	appName        string
	userID         string
	agentID        string
	state          *inMemoryState
	events         *inMemoryEvents
	metadata       map[string]interface{}
	lastUpdateTime time.Time
}

func (s *inMemorySession) ID() string {
	return s.id
}

func (s *inMemorySession) AppName() string {
	return s.appName
}

func (s *inMemorySession) UserID() string {
	return s.userID
}

func (s *inMemorySession) AgentID() string {
	return s.agentID
}

func (s *inMemorySession) State() State {
	return s.state
}

func (s *inMemorySession) Events() Events {
	return s.events
}

func (s *inMemorySession) LastUpdateTime() time.Time {
	return s.lastUpdateTime
}

func (s *inMemorySession) Metadata() map[string]interface{} {
	return s.metadata
}

// inMemoryState 内存状态实现
type inMemoryState struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

func newInMemoryState() *inMemoryState {
	return &inMemoryState{
		data: make(map[string]interface{}),
	}
}

func (s *inMemoryState) Get(key string) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	val, ok := s.data[key]
	if !ok {
		return nil, ErrStateKeyNotExist
	}
	return val, nil
}

func (s *inMemoryState) Set(key string, value interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = value
	return nil
}

func (s *inMemoryState) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)
	return nil
}

func (s *inMemoryState) All() iter.Seq2[string, interface{}] {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 复制数据以避免并发问题
	snapshot := make(map[string]interface{}, len(s.data))
	for k, v := range s.data {
		snapshot[k] = v
	}

	return func(yield func(string, interface{}) bool) {
		for k, v := range snapshot {
			if !yield(k, v) {
				return
			}
		}
	}
}

func (s *inMemoryState) Has(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.data[key]
	return ok
}

// inMemoryEvents 内存事件列表实现
type inMemoryEvents struct {
	mu     sync.RWMutex
	events []*Event
}

func newInMemoryEvents() *inMemoryEvents {
	return &inMemoryEvents{
		events: make([]*Event, 0),
	}
}

func (e *inMemoryEvents) All() iter.Seq[*Event] {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// 复制事件列表
	snapshot := make([]*Event, len(e.events))
	copy(snapshot, e.events)

	return func(yield func(*Event) bool) {
		for _, evt := range snapshot {
			if !yield(evt) {
				return
			}
		}
	}
}

func (e *inMemoryEvents) Len() int {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return len(e.events)
}

func (e *inMemoryEvents) At(i int) *Event {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if i < 0 || i >= len(e.events) {
		return nil
	}
	return e.events[i]
}

func (e *inMemoryEvents) Filter(predicate func(*Event) bool) []Event {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var results []Event
	for _, evt := range e.events {
		if predicate(evt) {
			results = append(results, *evt)
		}
	}
	return results
}

func (e *inMemoryEvents) Last() *Event {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.events) == 0 {
		return nil
	}
	return e.events[len(e.events)-1]
}

func (e *inMemoryEvents) append(event *Event) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.events = append(e.events, event)
}

func (e *inMemoryEvents) filter(predicate func(*Event) bool) []Event {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var results []Event
	for _, evt := range e.events {
		if predicate(evt) {
			results = append(results, *evt)
		}
	}
	return results
}
