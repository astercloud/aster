package desktop

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewApp(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *AppConfig
		wantFrame Framework
	}{
		{
			name:      "default config",
			cfg:       nil,
			wantFrame: FrameworkWeb,
		},
		{
			name: "wails framework",
			cfg: &AppConfig{
				Framework: FrameworkWails,
			},
			wantFrame: FrameworkWails,
		},
		{
			name: "tauri framework",
			cfg: &AppConfig{
				Framework: FrameworkTauri,
			},
			wantFrame: FrameworkTauri,
		},
		{
			name: "electron framework",
			cfg: &AppConfig{
				Framework: FrameworkElectron,
			},
			wantFrame: FrameworkElectron,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := NewApp(tt.cfg)
			if err != nil {
				t.Fatalf("NewApp() error = %v", err)
			}
			if app.Bridge().Framework() != tt.wantFrame {
				t.Errorf("Framework() = %v, want %v", app.Bridge().Framework(), tt.wantFrame)
			}
		})
	}
}

func TestNewAppUnsupportedFramework(t *testing.T) {
	_, err := NewApp(&AppConfig{
		Framework: Framework("unsupported"),
	})
	if err == nil {
		t.Error("expected error for unsupported framework")
	}
}

func TestWebBridgeHealthCheck(t *testing.T) {
	app, err := NewApp(&AppConfig{
		Framework: FrameworkWeb,
		Port:      0, // Will use default
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	bridge := app.Bridge().(*WebBridge)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// Create a handler that includes health check
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"status":    "ok",
			"framework": bridge.Framework(),
		})
	})

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("status = %v, want ok", resp["status"])
	}
}

func TestMessageTypes(t *testing.T) {
	tests := []struct {
		msgType MessageType
		want    string
	}{
		{MsgTypeChat, "chat"},
		{MsgTypeCancel, "cancel"},
		{MsgTypeApproval, "approval"},
		{MsgTypeGetStatus, "get_status"},
		{MsgTypeGetHistory, "get_history"},
		{MsgTypeClearHistory, "clear_history"},
		{MsgTypeSetConfig, "set_config"},
		{MsgTypeGetConfig, "get_config"},
	}

	for _, tt := range tests {
		if string(tt.msgType) != tt.want {
			t.Errorf("MessageType %v = %s, want %s", tt.msgType, string(tt.msgType), tt.want)
		}
	}
}

func TestEventTypes(t *testing.T) {
	tests := []struct {
		eventType EventType
		want      string
	}{
		{EventTypeTextChunk, "text_chunk"},
		{EventTypeToolStart, "tool_start"},
		{EventTypeToolEnd, "tool_end"},
		{EventTypeToolProgress, "tool_progress"},
		{EventTypeApprovalRequired, "approval_required"},
		{EventTypeError, "error"},
		{EventTypeDone, "done"},
		{EventTypeStatusChange, "status_change"},
	}

	for _, tt := range tests {
		if string(tt.eventType) != tt.want {
			t.Errorf("EventType %v = %s, want %s", tt.eventType, string(tt.eventType), tt.want)
		}
	}
}

func TestFrameworkTypes(t *testing.T) {
	tests := []struct {
		framework Framework
		want      string
	}{
		{FrameworkWails, "wails"},
		{FrameworkTauri, "tauri"},
		{FrameworkElectron, "electron"},
		{FrameworkWeb, "web"},
	}

	for _, tt := range tests {
		if string(tt.framework) != tt.want {
			t.Errorf("Framework %v = %s, want %s", tt.framework, string(tt.framework), tt.want)
		}
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	if id1 == "" {
		t.Error("generateID() returned empty string")
	}
	if id1 == id2 {
		t.Error("generateID() returned duplicate IDs")
	}
	if len(id1) != 32 { // 16 bytes = 32 hex chars
		t.Errorf("generateID() length = %d, want 32", len(id1))
	}
}

func TestMustMarshal(t *testing.T) {
	payload := ChatPayload{Message: "test"}
	data := mustMarshal(payload)

	var result ChatPayload
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result.Message != "test" {
		t.Errorf("Message = %s, want test", result.Message)
	}
}

func TestCorsMiddleware(t *testing.T) {
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))

	// Test normal request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Missing CORS header")
	}

	// Test preflight request
	req = httptest.NewRequest(http.MethodOptions, "/test", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("Preflight status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestWailsBridgeDirectMethods(t *testing.T) {
	app, err := NewApp(&AppConfig{
		Framework: FrameworkWails,
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	bridge := app.Bridge().(*WailsBridge)

	// Test GetConfig
	resp, err := bridge.GetConfig()
	if err != nil {
		t.Errorf("GetConfig() error = %v", err)
	}
	if !resp.Success {
		t.Errorf("GetConfig() success = false")
	}
}

func TestAppStartStop(t *testing.T) {
	app, err := NewApp(&AppConfig{
		Framework: FrameworkWeb,
		Port:      19999, // Use high port to avoid conflicts
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	ctx := context.Background()

	// Start
	if err := app.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Stop
	if err := app.Stop(ctx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestHandleUnknownMessageType(t *testing.T) {
	app, err := NewApp(&AppConfig{
		Framework: FrameworkWails,
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	resp, err := app.handleMessage(&FrontendMessage{
		ID:   "test-1",
		Type: MessageType("unknown"),
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resp.Success {
		t.Error("expected failure for unknown message type")
	}
	if !strings.Contains(resp.Error, "unknown message type") {
		t.Errorf("error = %s, want to contain 'unknown message type'", resp.Error)
	}
}

func TestTauriBridgePort(t *testing.T) {
	app, err := NewApp(&AppConfig{
		Framework: FrameworkTauri,
		Port:      19998,
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	bridge := app.Bridge().(*TauriBridge)
	if bridge.Port() != 19998 {
		t.Errorf("Port() = %d, want 19998", bridge.Port())
	}
}

func TestElectronBridgePort(t *testing.T) {
	app, err := NewApp(&AppConfig{
		Framework: FrameworkElectron,
		Port:      19997,
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	bridge := app.Bridge().(*ElectronBridge)
	if bridge.Port() != 19997 {
		t.Errorf("Port() = %d, want 19997", bridge.Port())
	}
}

func TestChatPayloadMarshal(t *testing.T) {
	payload := ChatPayload{
		Message: "Hello, world!",
		Context: map[string]any{
			"user": "test",
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result ChatPayload
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if result.Message != payload.Message {
		t.Errorf("Message = %s, want %s", result.Message, payload.Message)
	}
}

func TestApprovalPayloadMarshal(t *testing.T) {
	payload := ApprovalPayload{
		CallID:   "call-123",
		Decision: "allow",
		Note:     "User approved",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result ApprovalPayload
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if result.CallID != payload.CallID {
		t.Errorf("CallID = %s, want %s", result.CallID, payload.CallID)
	}
	if result.Decision != payload.Decision {
		t.Errorf("Decision = %s, want %s", result.Decision, payload.Decision)
	}
}

func TestConfigPayloadMarshal(t *testing.T) {
	payload := ConfigPayload{
		Provider:       "anthropic",
		Model:          "claude-sonnet-4-20250514",
		PermissionMode: "smart_approve",
		WorkDir:        "/tmp/test",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result ConfigPayload
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if result.Provider != payload.Provider {
		t.Errorf("Provider = %s, want %s", result.Provider, payload.Provider)
	}
}

func TestFrontendEventMarshal(t *testing.T) {
	event := FrontendEvent{
		Type:    EventTypeTextChunk,
		AgentID: "agent-123",
		Data:    map[string]string{"delta": "Hello"},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result FrontendEvent
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if result.Type != event.Type {
		t.Errorf("Type = %s, want %s", result.Type, event.Type)
	}
	if result.AgentID != event.AgentID {
		t.Errorf("AgentID = %s, want %s", result.AgentID, event.AgentID)
	}
}
