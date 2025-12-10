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

// WebBridge provides a generic HTTP-based bridge for web deployments
// and development. It uses REST API + SSE for communication.
//
// This bridge can be used:
// - For development without a desktop framework
// - For web-based deployments
// - As a base for custom integrations
type WebBridge struct {
	app        *App
	handler    MessageHandler
	agents     map[string]*agent.Agent
	agentsMu   sync.RWMutex
	server     *http.Server
	port       int
	sseClients map[string]chan *FrontendEvent
	sseMu      sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewWebBridge creates a new web bridge
func NewWebBridge(app *App, port int) (*WebBridge, error) {
	if port == 0 {
		port = 8080 // Default port for web bridge
	}

	return &WebBridge{
		app:        app,
		agents:     make(map[string]*agent.Agent),
		port:       port,
		sseClients: make(map[string]chan *FrontendEvent),
	}, nil
}

// Framework returns the framework type
func (b *WebBridge) Framework() Framework {
	return FrameworkWeb
}

// Start starts the HTTP server
func (b *WebBridge) Start(ctx context.Context) error {
	b.ctx, b.cancel = context.WithCancel(ctx)

	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/chat", b.handleChat)
	mux.HandleFunc("/api/cancel", b.handleCancel)
	mux.HandleFunc("/api/approve", b.handleApprove)
	mux.HandleFunc("/api/status", b.handleStatus)
	mux.HandleFunc("/api/history", b.handleHistory)
	mux.HandleFunc("/api/config", b.handleConfig)
	mux.HandleFunc("/api/agents", b.handleAgents)

	// SSE endpoint for events
	mux.HandleFunc("/api/events", b.handleSSE)

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"status":    "ok",
			"framework": "web",
			"port":      b.port,
			"agents":    len(b.agents),
		})
	})

	// CORS middleware
	handler := corsMiddleware(mux)

	b.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", b.port),
		Handler: handler,
	}

	// Start server in goroutine
	go func() {
		ln, err := net.Listen("tcp", b.server.Addr)
		if err != nil {
			return
		}
		b.server.Serve(ln)
	}()

	return nil
}

// Stop stops the HTTP server
func (b *WebBridge) Stop(ctx context.Context) error {
	if b.cancel != nil {
		b.cancel()
	}

	// Close all SSE clients
	b.sseMu.Lock()
	for _, ch := range b.sseClients {
		close(ch)
	}
	b.sseClients = make(map[string]chan *FrontendEvent)
	b.sseMu.Unlock()

	if b.server != nil {
		return b.server.Shutdown(ctx)
	}
	return nil
}

// RegisterAgent registers an agent with the bridge
func (b *WebBridge) RegisterAgent(ag *agent.Agent) error {
	b.agentsMu.Lock()
	defer b.agentsMu.Unlock()
	b.agents[ag.ID()] = ag
	return nil
}

// UnregisterAgent unregisters an agent
func (b *WebBridge) UnregisterAgent(agentID string) error {
	b.agentsMu.Lock()
	defer b.agentsMu.Unlock()
	delete(b.agents, agentID)
	return nil
}

// SendEvent sends an event to all SSE clients
func (b *WebBridge) SendEvent(event *FrontendEvent) error {
	b.sseMu.RLock()
	defer b.sseMu.RUnlock()

	for _, ch := range b.sseClients {
		select {
		case ch <- event:
		default:
			// Channel full, skip
		}
	}
	return nil
}

// OnMessage sets the handler for messages from frontend
func (b *WebBridge) OnMessage(handler MessageHandler) {
	b.handler = handler
}

// Port returns the server port
func (b *WebBridge) Port() int {
	return b.port
}

// HTTP Handlers

func (b *WebBridge) handleChat(w http.ResponseWriter, r *http.Request) {
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

func (b *WebBridge) handleCancel(w http.ResponseWriter, r *http.Request) {
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

func (b *WebBridge) handleApprove(w http.ResponseWriter, r *http.Request) {
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

func (b *WebBridge) handleStatus(w http.ResponseWriter, r *http.Request) {
	agentID := r.URL.Query().Get("agent_id")

	resp, _ := b.handler(&FrontendMessage{
		ID:      generateID(),
		Type:    MsgTypeGetStatus,
		AgentID: agentID,
	})

	writeJSON(w, http.StatusOK, resp)
}

func (b *WebBridge) handleHistory(w http.ResponseWriter, r *http.Request) {
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

func (b *WebBridge) handleConfig(w http.ResponseWriter, r *http.Request) {
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

func (b *WebBridge) handleAgents(w http.ResponseWriter, r *http.Request) {
	b.agentsMu.RLock()
	defer b.agentsMu.RUnlock()

	agents := make([]map[string]any, 0, len(b.agents))
	for id, ag := range b.agents {
		status := ag.Status()
		agents = append(agents, map[string]any{
			"id":    id,
			"state": status.State,
		})
	}

	writeJSON(w, http.StatusOK, BackendResponse{
		Success: true,
		Data:    agents,
	})
}

func (b *WebBridge) handleSSE(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create client channel
	clientID := generateID()
	eventCh := make(chan *FrontendEvent, 100)

	b.sseMu.Lock()
	b.sseClients[clientID] = eventCh
	b.sseMu.Unlock()

	defer func() {
		b.sseMu.Lock()
		delete(b.sseClients, clientID)
		b.sseMu.Unlock()
		close(eventCh)
	}()

	// Flush support
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Send initial connection event
	fmt.Fprintf(w, "event: connected\ndata: {\"client_id\":\"%s\",\"framework\":\"web\"}\n\n", clientID)
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
		case <-ticker.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		case event, ok := <-eventCh:
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
func (b *WebBridge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.server.Handler.ServeHTTP(w, r)
}
