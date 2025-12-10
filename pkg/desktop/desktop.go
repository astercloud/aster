// Package desktop provides a unified interface for desktop application frameworks.
// It supports Wails (Go + WebView), Tauri (Rust + WebView), and Electron (Node.js + Chromium).
//
// This package allows Aster to be embedded in desktop applications regardless of
// the chosen framework, providing a consistent API for:
// - IPC (Inter-Process Communication) between frontend and backend
// - Window management
// - System tray integration
// - Native dialogs
// - File system access with platform-specific paths
package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/config"
	"github.com/astercloud/aster/pkg/permission"
	"github.com/astercloud/aster/pkg/types"
)

// Framework represents the desktop framework type
type Framework string

const (
	// FrameworkWails uses Wails (Go + WebView2/WebKit)
	FrameworkWails Framework = "wails"

	// FrameworkTauri uses Tauri (Rust + WebView2/WebKit)
	FrameworkTauri Framework = "tauri"

	// FrameworkElectron uses Electron (Node.js + Chromium)
	FrameworkElectron Framework = "electron"

	// FrameworkWeb uses standard HTTP server (for development/web deployment)
	FrameworkWeb Framework = "web"
)

// Bridge provides the interface between Aster and desktop frameworks.
// Each framework has different IPC mechanisms:
// - Wails: Direct Go function binding
// - Tauri: Rust commands with JSON-RPC style communication
// - Electron: IPC channels via preload scripts
type Bridge interface {
	// Framework returns the framework type
	Framework() Framework

	// Start starts the bridge (may start HTTP server for Tauri/Electron)
	Start(ctx context.Context) error

	// Stop stops the bridge
	Stop(ctx context.Context) error

	// RegisterAgent registers an agent with the bridge
	RegisterAgent(ag *agent.Agent) error

	// UnregisterAgent unregisters an agent
	UnregisterAgent(agentID string) error

	// SendEvent sends an event to the frontend
	SendEvent(event *FrontendEvent) error

	// OnMessage sets the handler for messages from frontend
	OnMessage(handler MessageHandler)
}

// MessageHandler handles messages from the frontend
type MessageHandler func(msg *FrontendMessage) (*BackendResponse, error)

// FrontendMessage represents a message from the frontend
type FrontendMessage struct {
	// ID is the message ID for request-response correlation
	ID string `json:"id"`

	// Type is the message type
	Type MessageType `json:"type"`

	// AgentID is the target agent ID (optional)
	AgentID string `json:"agent_id,omitempty"`

	// Payload is the message payload
	Payload json.RawMessage `json:"payload,omitempty"`
}

// BackendResponse represents a response to the frontend
type BackendResponse struct {
	// ID is the original message ID
	ID string `json:"id"`

	// Success indicates if the operation was successful
	Success bool `json:"success"`

	// Data is the response data
	Data any `json:"data,omitempty"`

	// Error is the error message if not successful
	Error string `json:"error,omitempty"`
}

// FrontendEvent represents an event sent to the frontend
type FrontendEvent struct {
	// Type is the event type
	Type EventType `json:"type"`

	// AgentID is the source agent ID
	AgentID string `json:"agent_id,omitempty"`

	// Data is the event data
	Data any `json:"data"`
}

// MessageType defines frontend message types
type MessageType string

const (
	// MsgTypeChat sends a chat message to the agent
	MsgTypeChat MessageType = "chat"

	// MsgTypeCancel cancels the current operation
	MsgTypeCancel MessageType = "cancel"

	// MsgTypeApproval responds to a permission request
	MsgTypeApproval MessageType = "approval"

	// MsgTypeGetStatus gets agent status
	MsgTypeGetStatus MessageType = "get_status"

	// MsgTypeGetHistory gets conversation history
	MsgTypeGetHistory MessageType = "get_history"

	// MsgTypeClearHistory clears conversation history
	MsgTypeClearHistory MessageType = "clear_history"

	// MsgTypeSetConfig sets agent configuration
	MsgTypeSetConfig MessageType = "set_config"

	// MsgTypeGetConfig gets current configuration
	MsgTypeGetConfig MessageType = "get_config"
)

// EventType defines backend event types
type EventType string

const (
	// EventTypeTextChunk is a streaming text chunk
	EventTypeTextChunk EventType = "text_chunk"

	// EventTypeToolStart indicates tool execution started
	EventTypeToolStart EventType = "tool_start"

	// EventTypeToolEnd indicates tool execution ended
	EventTypeToolEnd EventType = "tool_end"

	// EventTypeToolProgress indicates tool execution progress
	EventTypeToolProgress EventType = "tool_progress"

	// EventTypeApprovalRequired indicates approval is needed
	EventTypeApprovalRequired EventType = "approval_required"

	// EventTypeError indicates an error occurred
	EventTypeError EventType = "error"

	// EventTypeDone indicates the response is complete
	EventTypeDone EventType = "done"

	// EventTypeStatusChange indicates agent status changed
	EventTypeStatusChange EventType = "status_change"
)

// ChatPayload is the payload for chat messages
type ChatPayload struct {
	Message string         `json:"message"`
	Context map[string]any `json:"context,omitempty"`
}

// ApprovalPayload is the payload for approval responses
type ApprovalPayload struct {
	CallID   string `json:"call_id"`
	Decision string `json:"decision"` // "allow", "deny", "allow_always", "deny_always"
	Note     string `json:"note,omitempty"`
}

// ConfigPayload is the payload for configuration
type ConfigPayload struct {
	Provider       string `json:"provider,omitempty"`
	Model          string `json:"model,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	WorkDir        string `json:"work_dir,omitempty"`
}

// App represents a desktop application instance
type App struct {
	bridge     Bridge
	agents     map[string]*agent.Agent
	agentsMu   sync.RWMutex
	handler    MessageHandler
	inspector  *permission.Inspector
	config     *AppConfig
}

// AppConfig is the application configuration
type AppConfig struct {
	// Framework is the desktop framework to use
	Framework Framework `json:"framework"`

	// Port is the HTTP port for Tauri/Electron bridges (0 for auto)
	Port int `json:"port,omitempty"`

	// PermissionMode is the default permission mode
	PermissionMode permission.Mode `json:"permission_mode,omitempty"`

	// WorkDir is the default working directory
	WorkDir string `json:"work_dir,omitempty"`

	// DataDir is the data directory (defaults to platform-specific)
	DataDir string `json:"data_dir,omitempty"`
}

// NewApp creates a new desktop application
func NewApp(cfg *AppConfig) (*App, error) {
	if cfg == nil {
		cfg = &AppConfig{
			Framework:      FrameworkWeb,
			PermissionMode: permission.ModeSmartApprove,
		}
	}

	// Set defaults
	if cfg.DataDir == "" {
		cfg.DataDir = config.DataDir()
	}
	if cfg.WorkDir == "" {
		cfg.WorkDir = "."
	}

	// Create permission inspector
	inspector := permission.NewInspector(cfg.PermissionMode)

	app := &App{
		agents:    make(map[string]*agent.Agent),
		inspector: inspector,
		config:    cfg,
	}

	// Create bridge based on framework
	var err error
	switch cfg.Framework {
	case FrameworkWails:
		app.bridge, err = NewWailsBridge(app)
	case FrameworkTauri:
		app.bridge, err = NewTauriBridge(app, cfg.Port)
	case FrameworkElectron:
		app.bridge, err = NewElectronBridge(app, cfg.Port)
	case FrameworkWeb:
		app.bridge, err = NewWebBridge(app, cfg.Port)
	default:
		return nil, fmt.Errorf("unsupported framework: %s", cfg.Framework)
	}

	if err != nil {
		return nil, fmt.Errorf("create bridge: %w", err)
	}

	// Set message handler
	app.bridge.OnMessage(app.handleMessage)

	return app, nil
}

// Start starts the application
func (a *App) Start(ctx context.Context) error {
	return a.bridge.Start(ctx)
}

// Stop stops the application
func (a *App) Stop(ctx context.Context) error {
	// Close all agents
	a.agentsMu.Lock()
	for _, ag := range a.agents {
		ag.Close()
	}
	a.agents = make(map[string]*agent.Agent)
	a.agentsMu.Unlock()

	return a.bridge.Stop(ctx)
}

// RegisterAgent registers an agent with the app
func (a *App) RegisterAgent(ag *agent.Agent) error {
	a.agentsMu.Lock()
	defer a.agentsMu.Unlock()

	a.agents[ag.ID()] = ag
	return a.bridge.RegisterAgent(ag)
}

// GetAgent returns an agent by ID
func (a *App) GetAgent(id string) (*agent.Agent, bool) {
	a.agentsMu.RLock()
	defer a.agentsMu.RUnlock()

	ag, ok := a.agents[id]
	return ag, ok
}

// Bridge returns the underlying bridge
func (a *App) Bridge() Bridge {
	return a.bridge
}

// Inspector returns the permission inspector
func (a *App) Inspector() *permission.Inspector {
	return a.inspector
}

// handleMessage handles messages from the frontend
func (a *App) handleMessage(msg *FrontendMessage) (*BackendResponse, error) {
	switch msg.Type {
	case MsgTypeChat:
		return a.handleChat(msg)
	case MsgTypeCancel:
		return a.handleCancel(msg)
	case MsgTypeApproval:
		return a.handleApproval(msg)
	case MsgTypeGetStatus:
		return a.handleGetStatus(msg)
	case MsgTypeGetHistory:
		return a.handleGetHistory(msg)
	case MsgTypeClearHistory:
		return a.handleClearHistory(msg)
	case MsgTypeSetConfig:
		return a.handleSetConfig(msg)
	case MsgTypeGetConfig:
		return a.handleGetConfig(msg)
	default:
		return &BackendResponse{
			ID:      msg.ID,
			Success: false,
			Error:   fmt.Sprintf("unknown message type: %s", msg.Type),
		}, nil
	}
}

func (a *App) handleChat(msg *FrontendMessage) (*BackendResponse, error) {
	var payload ChatPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return &BackendResponse{
			ID:      msg.ID,
			Success: false,
			Error:   fmt.Sprintf("invalid payload: %v", err),
		}, nil
	}

	ag, ok := a.GetAgent(msg.AgentID)
	if !ok {
		return &BackendResponse{
			ID:      msg.ID,
			Success: false,
			Error:   fmt.Sprintf("agent not found: %s", msg.AgentID),
		}, nil
	}

	// Subscribe to events and forward to frontend
	go a.forwardAgentEvents(ag, msg.AgentID)

	// Send message to agent (non-blocking, events will be sent via bridge)
	go func() {
		ctx := context.Background()
		err := ag.Send(ctx, payload.Message)
		if err != nil {
			a.bridge.SendEvent(&FrontendEvent{
				Type:    EventTypeError,
				AgentID: msg.AgentID,
				Data:    map[string]string{"error": err.Error()},
			})
		}
		// Note: Done event will be sent when agent finishes processing
	}()

	return &BackendResponse{
		ID:      msg.ID,
		Success: true,
		Data:    map[string]string{"status": "started"},
	}, nil
}

func (a *App) handleCancel(msg *FrontendMessage) (*BackendResponse, error) {
	ag, ok := a.GetAgent(msg.AgentID)
	if !ok {
		return &BackendResponse{
			ID:      msg.ID,
			Success: false,
			Error:   fmt.Sprintf("agent not found: %s", msg.AgentID),
		}, nil
	}

	// Close the agent to cancel operations
	ag.Close()

	return &BackendResponse{
		ID:      msg.ID,
		Success: true,
	}, nil
}

func (a *App) handleApproval(msg *FrontendMessage) (*BackendResponse, error) {
	var payload ApprovalPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return &BackendResponse{
			ID:      msg.ID,
			Success: false,
			Error:   fmt.Sprintf("invalid payload: %v", err),
		}, nil
	}

	// Record decision for future reference
	a.inspector.RecordDecision(&permission.Request{
		CallID: payload.CallID,
	}, permission.Decision(payload.Decision), payload.Note)

	return &BackendResponse{
		ID:      msg.ID,
		Success: true,
	}, nil
}

func (a *App) handleGetStatus(msg *FrontendMessage) (*BackendResponse, error) {
	ag, ok := a.GetAgent(msg.AgentID)
	if !ok {
		return &BackendResponse{
			ID:      msg.ID,
			Success: false,
			Error:   fmt.Sprintf("agent not found: %s", msg.AgentID),
		}, nil
	}

	status := ag.Status()
	return &BackendResponse{
		ID:      msg.ID,
		Success: true,
		Data: map[string]any{
			"id":    ag.ID(),
			"state": status.State,
		},
	}, nil
}

func (a *App) handleGetHistory(msg *FrontendMessage) (*BackendResponse, error) {
	_, ok := a.GetAgent(msg.AgentID)
	if !ok {
		return &BackendResponse{
			ID:      msg.ID,
			Success: false,
			Error:   fmt.Sprintf("agent not found: %s", msg.AgentID),
		}, nil
	}

	// Note: Message history is managed internally by the agent
	// For now, return empty - in production, use session store
	return &BackendResponse{
		ID:      msg.ID,
		Success: true,
		Data:    []any{},
	}, nil
}

func (a *App) handleClearHistory(msg *FrontendMessage) (*BackendResponse, error) {
	_, ok := a.GetAgent(msg.AgentID)
	if !ok {
		return &BackendResponse{
			ID:      msg.ID,
			Success: false,
			Error:   fmt.Sprintf("agent not found: %s", msg.AgentID),
		}, nil
	}

	// Note: Clearing history requires recreating the agent
	// For now, just acknowledge the request
	return &BackendResponse{
		ID:      msg.ID,
		Success: true,
	}, nil
}

func (a *App) handleSetConfig(msg *FrontendMessage) (*BackendResponse, error) {
	var payload ConfigPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return &BackendResponse{
			ID:      msg.ID,
			Success: false,
			Error:   fmt.Sprintf("invalid payload: %v", err),
		}, nil
	}

	if payload.PermissionMode != "" {
		a.inspector.SetMode(permission.Mode(payload.PermissionMode))
	}

	return &BackendResponse{
		ID:      msg.ID,
		Success: true,
	}, nil
}

func (a *App) handleGetConfig(msg *FrontendMessage) (*BackendResponse, error) {
	return &BackendResponse{
		ID:      msg.ID,
		Success: true,
		Data: map[string]any{
			"framework":       a.config.Framework,
			"permission_mode": a.inspector.GetMode(),
			"work_dir":        a.config.WorkDir,
			"data_dir":        a.config.DataDir,
		},
	}, nil
}

// forwardAgentEvents subscribes to agent events and forwards them to the frontend
func (a *App) forwardAgentEvents(ag *agent.Agent, agentID string) {
	eventCh := ag.Subscribe([]types.AgentChannel{
		types.ChannelProgress,
		types.ChannelControl,
	}, nil)

	for envelope := range eventCh {
		var event *FrontendEvent

		switch e := envelope.Event.(type) {
		case *types.ProgressTextChunkEvent:
			event = &FrontendEvent{
				Type:    EventTypeTextChunk,
				AgentID: agentID,
				Data:    map[string]string{"delta": e.Delta},
			}

		case *types.ProgressToolStartEvent:
			event = &FrontendEvent{
				Type:    EventTypeToolStart,
				AgentID: agentID,
				Data: map[string]any{
					"call_id":   e.Call.ID,
					"name":      e.Call.Name,
					"arguments": e.Call.Arguments,
				},
			}

		case *types.ProgressToolEndEvent:
			event = &FrontendEvent{
				Type:    EventTypeToolEnd,
				AgentID: agentID,
				Data: map[string]any{
					"call_id": e.Call.ID,
					"name":    e.Call.Name,
					"result":  e.Call.Result,
					"error":   e.Call.Error,
				},
			}

		case *types.ProgressToolProgressEvent:
			event = &FrontendEvent{
				Type:    EventTypeToolProgress,
				AgentID: agentID,
				Data: map[string]any{
					"call_id":  e.Call.ID,
					"progress": e.Progress,
					"message":  e.Message,
				},
			}

		case *types.ControlPermissionRequiredEvent:
			event = &FrontendEvent{
				Type:    EventTypeApprovalRequired,
				AgentID: agentID,
				Data: map[string]any{
					"call_id":   e.Call.ID,
					"name":      e.Call.Name,
					"arguments": e.Call.Arguments,
				},
			}

		case *types.MonitorErrorEvent:
			event = &FrontendEvent{
				Type:    EventTypeError,
				AgentID: agentID,
				Data: map[string]any{
					"severity": e.Severity,
					"message":  e.Message,
				},
			}
		}

		if event != nil {
			a.bridge.SendEvent(event)
		}
	}
}

// ServeHTTP implements http.Handler for web-based bridges
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if wb, ok := a.bridge.(http.Handler); ok {
		wb.ServeHTTP(w, r)
	} else {
		http.Error(w, "Bridge does not support HTTP", http.StatusNotImplemented)
	}
}
