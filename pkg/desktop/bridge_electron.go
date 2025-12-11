package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/agent"
)

// ElectronBridge provides integration with Electron framework.
// Electron uses a local HTTP server similar to Tauri, but with additional
// support for Electron's IPC mechanism via WebSocket.
//
// Communication flow:
// 1. Electron preload script connects to localhost:PORT via WebSocket
// 2. Messages are sent bidirectionally through WebSocket
// 3. Events are pushed from backend to frontend via WebSocket
//
// Usage in Electron:
//
//	// In main.js
//	const { app, BrowserWindow } = require('electron');
//	const { spawn } = require('child_process');
//
//	// Start Aster backend
//	const asterProcess = spawn('./aster-server', ['--port', '9527']);
//
//	// In preload.js
//	const ws = new WebSocket('ws://localhost:9527/ws');
//	ws.onmessage = (event) => {
//	    const data = JSON.parse(event.data);
//	    // Handle events
//	};
//
//	// In renderer.js
//	window.aster.chat(agentId, message);
type ElectronBridge struct {
	app       *App
	handler   MessageHandler
	agents    map[string]*agent.Agent
	agentsMu  sync.RWMutex
	server    *http.Server
	port      int
	wsClients map[string]*wsClient
	wsMu      sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// wsClient represents a WebSocket client
type wsClient struct {
	id      string
	eventCh chan *FrontendEvent
	closeCh chan struct{}
}

// NewElectronBridge creates a new Electron bridge
func NewElectronBridge(app *App, port int) (*ElectronBridge, error) {
	if port == 0 {
		port = 9527 // Default port for Electron bridge
	}

	return &ElectronBridge{
		app:       app,
		agents:    make(map[string]*agent.Agent),
		port:      port,
		wsClients: make(map[string]*wsClient),
	}, nil
}

// Framework returns the framework type
func (b *ElectronBridge) Framework() Framework {
	return FrameworkElectron
}

// Start starts the HTTP/WebSocket server
func (b *ElectronBridge) Start(ctx context.Context) error {
	b.ctx, b.cancel = context.WithCancel(ctx)

	mux := http.NewServeMux()

	// REST API endpoints (same as Tauri)
	mux.HandleFunc("/api/chat", b.handleChat)
	mux.HandleFunc("/api/cancel", b.handleCancel)
	mux.HandleFunc("/api/approve", b.handleApprove)
	mux.HandleFunc("/api/status", b.handleStatus)
	mux.HandleFunc("/api/history", b.handleHistory)
	mux.HandleFunc("/api/config", b.handleConfig)

	// WebSocket endpoint
	mux.HandleFunc("/ws", b.handleWebSocket)

	// SSE endpoint (fallback)
	mux.HandleFunc("/api/events", b.handleSSE)

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":    "ok",
			"framework": "electron",
			"port":      b.port,
		})
	})

	// CORS middleware
	handler := corsMiddleware(mux)

	b.server = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", b.port),
		Handler: handler,
	}

	// Start server in goroutine
	go func() {
		ln, err := net.Listen("tcp", b.server.Addr)
		if err != nil {
			return
		}
		_ = b.server.Serve(ln) // Server error logged by http.Server
	}()

	return nil
}

// Stop stops the HTTP/WebSocket server
func (b *ElectronBridge) Stop(ctx context.Context) error {
	if b.cancel != nil {
		b.cancel()
	}

	// Close all WebSocket clients
	b.wsMu.Lock()
	for _, client := range b.wsClients {
		close(client.closeCh)
	}
	b.wsClients = make(map[string]*wsClient)
	b.wsMu.Unlock()

	if b.server != nil {
		return b.server.Shutdown(ctx)
	}
	return nil
}

// RegisterAgent registers an agent with the bridge
func (b *ElectronBridge) RegisterAgent(ag *agent.Agent) error {
	b.agentsMu.Lock()
	defer b.agentsMu.Unlock()
	b.agents[ag.ID()] = ag
	return nil
}

// UnregisterAgent unregisters an agent
func (b *ElectronBridge) UnregisterAgent(agentID string) error {
	b.agentsMu.Lock()
	defer b.agentsMu.Unlock()
	delete(b.agents, agentID)
	return nil
}

// SendEvent sends an event to all WebSocket clients
func (b *ElectronBridge) SendEvent(event *FrontendEvent) error {
	b.wsMu.RLock()
	defer b.wsMu.RUnlock()

	for _, client := range b.wsClients {
		select {
		case client.eventCh <- event:
		default:
			// Channel full, skip
		}
	}
	return nil
}

// OnMessage sets the handler for messages from frontend
func (b *ElectronBridge) OnMessage(handler MessageHandler) {
	b.handler = handler
}

// Port returns the server port
func (b *ElectronBridge) Port() int {
	return b.port
}

// HTTP Handlers (delegated to common implementation)

func (b *ElectronBridge) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AgentID string `json:"agent_id"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, BackendResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	resp, err := b.handler(&FrontendMessage{
		ID:      generateID(),
		Type:    MsgTypeChat,
		AgentID: req.AgentID,
		Payload: mustMarshal(ChatPayload{Message: req.Message}),
	})

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, BackendResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (b *ElectronBridge) handleCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AgentID string `json:"agent_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, BackendResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	resp, _ := b.handler(&FrontendMessage{
		ID:      generateID(),
		Type:    MsgTypeCancel,
		AgentID: req.AgentID,
	})

	writeJSON(w, http.StatusOK, resp)
}

func (b *ElectronBridge) handleApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AgentID  string `json:"agent_id"`
		CallID   string `json:"call_id"`
		Decision string `json:"decision"`
		Note     string `json:"note"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, BackendResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	resp, _ := b.handler(&FrontendMessage{
		ID:      generateID(),
		Type:    MsgTypeApproval,
		AgentID: req.AgentID,
		Payload: mustMarshal(ApprovalPayload{
			CallID:   req.CallID,
			Decision: req.Decision,
			Note:     req.Note,
		}),
	})

	writeJSON(w, http.StatusOK, resp)
}

func (b *ElectronBridge) handleStatus(w http.ResponseWriter, r *http.Request) {
	agentID := r.URL.Query().Get("agent_id")

	resp, _ := b.handler(&FrontendMessage{
		ID:      generateID(),
		Type:    MsgTypeGetStatus,
		AgentID: agentID,
	})

	writeJSON(w, http.StatusOK, resp)
}

func (b *ElectronBridge) handleHistory(w http.ResponseWriter, r *http.Request) {
	agentID := r.URL.Query().Get("agent_id")

	var msgType MessageType
	if r.Method == http.MethodDelete {
		msgType = MsgTypeClearHistory
	} else {
		msgType = MsgTypeGetHistory
	}

	resp, _ := b.handler(&FrontendMessage{
		ID:      generateID(),
		Type:    msgType,
		AgentID: agentID,
	})

	writeJSON(w, http.StatusOK, resp)
}

func (b *ElectronBridge) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		resp, _ := b.handler(&FrontendMessage{
			ID:   generateID(),
			Type: MsgTypeGetConfig,
		})
		writeJSON(w, http.StatusOK, resp)

	case http.MethodPost, http.MethodPut:
		var cfg ConfigPayload
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeJSON(w, http.StatusBadRequest, BackendResponse{
				Success: false,
				Error:   err.Error(),
			})
			return
		}

		resp, _ := b.handler(&FrontendMessage{
			ID:      generateID(),
			Type:    MsgTypeSetConfig,
			Payload: mustMarshal(cfg),
		})
		writeJSON(w, http.StatusOK, resp)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleWebSocket handles WebSocket connections
// Note: This is a simplified implementation using SSE-style long polling
// For production, use a proper WebSocket library like gorilla/websocket
func (b *ElectronBridge) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// For now, redirect to SSE
	// TODO: Implement proper WebSocket with gorilla/websocket
	b.handleSSE(w, r)
}

func (b *ElectronBridge) handleSSE(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create client
	clientID := generateID()
	client := &wsClient{
		id:      clientID,
		eventCh: make(chan *FrontendEvent, 100),
		closeCh: make(chan struct{}),
	}

	b.wsMu.Lock()
	b.wsClients[clientID] = client
	b.wsMu.Unlock()

	defer func() {
		b.wsMu.Lock()
		delete(b.wsClients, clientID)
		b.wsMu.Unlock()
	}()

	// Flush support
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Send initial connection event
	fmt.Fprintf(w, "event: connected\ndata: {\"client_id\":\"%s\",\"framework\":\"electron\"}\n\n", clientID)
	flusher.Flush()

	// Keep-alive ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-b.ctx.Done():
			return
		case <-client.closeCh:
			return
		case <-ticker.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		case event, ok := <-client.eventCh:
			if !ok {
				return
			}
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
			flusher.Flush()
		}
	}
}

// ServeHTTP implements http.Handler
func (b *ElectronBridge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.server.Handler.ServeHTTP(w, r)
}
