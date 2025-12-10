package desktop

import (
	"context"
	"sync"

	"github.com/astercloud/aster/pkg/agent"
)

// WailsBridge provides integration with Wails framework.
// Wails uses direct Go function binding, making it the most efficient option
// for Go-based desktop applications.
//
// Usage in Wails app:
//
//	app := desktop.NewApp(&desktop.AppConfig{Framework: desktop.FrameworkWails})
//	wails.Run(&options.App{
//	    Bind: []interface{}{app.Bridge()},
//	})
type WailsBridge struct {
	app      *App
	handler  MessageHandler
	agents   map[string]*agent.Agent
	agentsMu sync.RWMutex
	eventCh  chan *FrontendEvent
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewWailsBridge creates a new Wails bridge
func NewWailsBridge(app *App) (*WailsBridge, error) {
	return &WailsBridge{
		app:     app,
		agents:  make(map[string]*agent.Agent),
		eventCh: make(chan *FrontendEvent, 100),
	}, nil
}

// Framework returns the framework type
func (b *WailsBridge) Framework() Framework {
	return FrameworkWails
}

// Start starts the bridge
func (b *WailsBridge) Start(ctx context.Context) error {
	b.ctx, b.cancel = context.WithCancel(ctx)
	return nil
}

// Stop stops the bridge
func (b *WailsBridge) Stop(ctx context.Context) error {
	if b.cancel != nil {
		b.cancel()
	}
	close(b.eventCh)
	return nil
}

// RegisterAgent registers an agent with the bridge
func (b *WailsBridge) RegisterAgent(ag *agent.Agent) error {
	b.agentsMu.Lock()
	defer b.agentsMu.Unlock()
	b.agents[ag.ID()] = ag
	return nil
}

// UnregisterAgent unregisters an agent
func (b *WailsBridge) UnregisterAgent(agentID string) error {
	b.agentsMu.Lock()
	defer b.agentsMu.Unlock()
	delete(b.agents, agentID)
	return nil
}

// SendEvent sends an event to the frontend
// In Wails, this uses runtime.EventsEmit
func (b *WailsBridge) SendEvent(event *FrontendEvent) error {
	select {
	case b.eventCh <- event:
		return nil
	default:
		// Channel full, drop event
		return nil
	}
}

// OnMessage sets the handler for messages from frontend
func (b *WailsBridge) OnMessage(handler MessageHandler) {
	b.handler = handler
}

// ============================================
// Wails-exposed methods (called from frontend)
// ============================================

// Chat sends a chat message to an agent
// Exposed to Wails frontend as: window.go.desktop.WailsBridge.Chat(agentID, message)
func (b *WailsBridge) Chat(agentID, message string) (*BackendResponse, error) {
	return b.handler(&FrontendMessage{
		ID:      generateID(),
		Type:    MsgTypeChat,
		AgentID: agentID,
		Payload: mustMarshal(ChatPayload{Message: message}),
	})
}

// Cancel cancels the current operation
// Exposed to Wails frontend as: window.go.desktop.WailsBridge.Cancel(agentID)
func (b *WailsBridge) Cancel(agentID string) (*BackendResponse, error) {
	return b.handler(&FrontendMessage{
		ID:      generateID(),
		Type:    MsgTypeCancel,
		AgentID: agentID,
	})
}

// Approve responds to a permission request
// Exposed to Wails frontend as: window.go.desktop.WailsBridge.Approve(agentID, callID, decision, note)
func (b *WailsBridge) Approve(agentID, callID, decision, note string) (*BackendResponse, error) {
	return b.handler(&FrontendMessage{
		ID:      generateID(),
		Type:    MsgTypeApproval,
		AgentID: agentID,
		Payload: mustMarshal(ApprovalPayload{
			CallID:   callID,
			Decision: decision,
			Note:     note,
		}),
	})
}

// GetStatus gets the agent status
// Exposed to Wails frontend as: window.go.desktop.WailsBridge.GetStatus(agentID)
func (b *WailsBridge) GetStatus(agentID string) (*BackendResponse, error) {
	return b.handler(&FrontendMessage{
		ID:      generateID(),
		Type:    MsgTypeGetStatus,
		AgentID: agentID,
	})
}

// GetHistory gets conversation history
// Exposed to Wails frontend as: window.go.desktop.WailsBridge.GetHistory(agentID)
func (b *WailsBridge) GetHistory(agentID string) (*BackendResponse, error) {
	return b.handler(&FrontendMessage{
		ID:      generateID(),
		Type:    MsgTypeGetHistory,
		AgentID: agentID,
	})
}

// ClearHistory clears conversation history
// Exposed to Wails frontend as: window.go.desktop.WailsBridge.ClearHistory(agentID)
func (b *WailsBridge) ClearHistory(agentID string) (*BackendResponse, error) {
	return b.handler(&FrontendMessage{
		ID:      generateID(),
		Type:    MsgTypeClearHistory,
		AgentID: agentID,
	})
}

// SetConfig sets configuration
// Exposed to Wails frontend as: window.go.desktop.WailsBridge.SetConfig(config)
func (b *WailsBridge) SetConfig(cfg ConfigPayload) (*BackendResponse, error) {
	return b.handler(&FrontendMessage{
		ID:      generateID(),
		Type:    MsgTypeSetConfig,
		Payload: mustMarshal(cfg),
	})
}

// GetConfig gets current configuration
// Exposed to Wails frontend as: window.go.desktop.WailsBridge.GetConfig()
func (b *WailsBridge) GetConfig() (*BackendResponse, error) {
	return b.handler(&FrontendMessage{
		ID:   generateID(),
		Type: MsgTypeGetConfig,
	})
}

// GetEvents returns the event channel for Wails runtime to consume
// Usage: Use with wails runtime.EventsEmit in a goroutine
func (b *WailsBridge) GetEvents() <-chan *FrontendEvent {
	return b.eventCh
}

// WailsInit is called by Wails during initialization
func (b *WailsBridge) WailsInit(ctx context.Context) error {
	b.ctx = ctx
	return nil
}

// WailsShutdown is called by Wails during shutdown
func (b *WailsBridge) WailsShutdown(ctx context.Context) error {
	return b.Stop(ctx)
}
