package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/astercloud/aster/pkg/telemetry"
	"github.com/astercloud/aster/pkg/telemetry/genai"
	"github.com/astercloud/aster/pkg/types"
)

func TestNewTelemetryMiddleware(t *testing.T) {
	tests := []struct {
		name   string
		config *TelemetryMiddlewareConfig
	}{
		{
			name:   "nil config",
			config: nil,
		},
		{
			name: "basic config",
			config: &TelemetryMiddlewareConfig{
				AgentID:   "test-agent",
				AgentName: "Test Agent",
				Provider:  genai.ProviderAnthropic,
				Model:     "claude-3-5-sonnet",
			},
		},
		{
			name: "full config",
			config: &TelemetryMiddlewareConfig{
				AgentID:           "test-agent",
				AgentName:         "Test Agent",
				Provider:          genai.ProviderOpenAI,
				Model:             "gpt-4",
				ConversationID:    "conv-123",
				RecordPrompts:     true,
				RecordCompletions: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewTelemetryMiddleware(tt.config)
			if m == nil {
				t.Fatal("expected middleware to be created")
			}
			if m.Name() != "telemetry" {
				t.Errorf("expected name 'telemetry', got '%s'", m.Name())
			}
			if m.Priority() != 5 {
				t.Errorf("expected priority 5, got %d", m.Priority())
			}
		})
	}
}

func TestTelemetryMiddleware_WrapModelCall(t *testing.T) {
	tracer := telemetry.NewSimpleTracer()

	m := NewTelemetryMiddleware(&TelemetryMiddlewareConfig{
		Tracer:            tracer,
		AgentID:           "test-agent",
		AgentName:         "Test Agent",
		Provider:          genai.ProviderAnthropic,
		Model:             "claude-3-5-sonnet",
		ConversationID:    "conv-123",
		RecordPrompts:     true,
		RecordCompletions: true,
	})

	ctx := context.Background()
	req := &ModelRequest{
		Messages: []types.Message{
			{Role: types.MessageRoleUser, Content: "Hello"},
		},
		SystemPrompt: "You are a helpful assistant",
		Metadata: map[string]any{
			"max_tokens":  1000,
			"temperature": 0.7,
		},
	}

	// Mock handler
	handler := func(ctx context.Context, req *ModelRequest) (*ModelResponse, error) {
		return &ModelResponse{
			Message: types.Message{
				Role:    types.MessageRoleAssistant,
				Content: "Hello! How can I help you?",
			},
			Metadata: map[string]any{
				"input_tokens":  10,
				"output_tokens": 8,
				"response_id":   "resp-123",
				"stop_reason":   "end_turn",
			},
		}, nil
	}

	resp, err := m.WrapModelCall(ctx, req, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response")
	}

	// Verify spans
	spans := tracer.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	span := spans[0]
	if span.Name() != "chat claude-3-5-sonnet" {
		t.Errorf("expected span name 'chat claude-3-5-sonnet', got '%s'", span.Name())
	}

	// Verify attributes
	attrs := span.Attributes()
	attrMap := make(map[string]any)
	for _, attr := range attrs {
		attrMap[attr.Key] = attr.Value
	}

	if attrMap[genai.AttrOperationName] != genai.OpChat {
		t.Errorf("expected operation name '%s', got '%v'", genai.OpChat, attrMap[genai.AttrOperationName])
	}
	if attrMap[genai.AttrProviderName] != genai.ProviderAnthropic {
		t.Errorf("expected provider '%s', got '%v'", genai.ProviderAnthropic, attrMap[genai.AttrProviderName])
	}
	if attrMap[genai.AttrAgentID] != "test-agent" {
		t.Errorf("expected agent id 'test-agent', got '%v'", attrMap[genai.AttrAgentID])
	}
	if attrMap[genai.AttrConversationID] != "conv-123" {
		t.Errorf("expected conversation id 'conv-123', got '%v'", attrMap[genai.AttrConversationID])
	}

	// Verify status
	if span.Status() != telemetry.StatusCodeOK {
		t.Errorf("expected status OK, got %v", span.Status())
	}
}

func TestTelemetryMiddleware_WrapModelCall_Error(t *testing.T) {
	tracer := telemetry.NewSimpleTracer()

	m := NewTelemetryMiddleware(&TelemetryMiddlewareConfig{
		Tracer:   tracer,
		AgentID:  "test-agent",
		Provider: genai.ProviderAnthropic,
		Model:    "claude-3-5-sonnet",
	})

	ctx := context.Background()
	req := &ModelRequest{
		Messages: []types.Message{},
	}

	// Mock handler that returns error
	expectedErr := errors.New("rate limit exceeded (429)")
	handler := func(ctx context.Context, req *ModelRequest) (*ModelResponse, error) {
		return nil, expectedErr
	}

	_, err := m.WrapModelCall(ctx, req, handler)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error '%v', got '%v'", expectedErr, err)
	}

	// Verify spans
	spans := tracer.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	span := spans[0]
	if span.Status() != telemetry.StatusCodeError {
		t.Errorf("expected status Error, got %v", span.Status())
	}
	if span.Error() == nil {
		t.Error("expected error to be recorded")
	}

	// Verify error type attribute
	attrs := span.Attributes()
	attrMap := make(map[string]any)
	for _, attr := range attrs {
		attrMap[attr.Key] = attr.Value
	}
	if attrMap[genai.AttrErrorType] != genai.ErrorTypeRateLimit {
		t.Errorf("expected error type '%s', got '%v'", genai.ErrorTypeRateLimit, attrMap[genai.AttrErrorType])
	}
}

func TestTelemetryMiddleware_WrapToolCall(t *testing.T) {
	tracer := telemetry.NewSimpleTracer()

	m := NewTelemetryMiddleware(&TelemetryMiddlewareConfig{
		Tracer:         tracer,
		AgentID:        "test-agent",
		AgentName:      "Test Agent",
		ConversationID: "conv-123",
	})

	ctx := context.Background()
	req := &ToolCallRequest{
		ToolCallID: "call-123",
		ToolName:   "read_file",
		ToolInput: map[string]any{
			"path": "/tmp/test.txt",
		},
	}

	// Mock handler
	handler := func(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
		return &ToolCallResponse{
			Result: map[string]any{
				"content": "file contents",
			},
		}, nil
	}

	resp, err := m.WrapToolCall(ctx, req, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response")
	}

	// Verify spans
	spans := tracer.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	span := spans[0]
	if span.Name() != "execute_tool read_file" {
		t.Errorf("expected span name 'execute_tool read_file', got '%s'", span.Name())
	}

	// Verify attributes
	attrs := span.Attributes()
	attrMap := make(map[string]any)
	for _, attr := range attrs {
		attrMap[attr.Key] = attr.Value
	}

	if attrMap[genai.AttrOperationName] != genai.OpExecuteTool {
		t.Errorf("expected operation name '%s', got '%v'", genai.OpExecuteTool, attrMap[genai.AttrOperationName])
	}
	if attrMap[genai.AttrToolName] != "read_file" {
		t.Errorf("expected tool name 'read_file', got '%v'", attrMap[genai.AttrToolName])
	}
	if attrMap[genai.AttrToolCallID] != "call-123" {
		t.Errorf("expected tool call id 'call-123', got '%v'", attrMap[genai.AttrToolCallID])
	}

	// Verify events
	events := span.Events()
	if len(events) != 2 {
		t.Errorf("expected 2 events (tool_call and tool_result), got %d", len(events))
	}
}

func TestTelemetryMiddleware_WrapToolCall_Error(t *testing.T) {
	tracer := telemetry.NewSimpleTracer()

	m := NewTelemetryMiddleware(&TelemetryMiddlewareConfig{
		Tracer:  tracer,
		AgentID: "test-agent",
	})

	ctx := context.Background()
	req := &ToolCallRequest{
		ToolCallID: "call-123",
		ToolName:   "bash",
		ToolInput: map[string]any{
			"command": "ls",
		},
	}

	// Mock handler that returns error
	expectedErr := errors.New("permission denied")
	handler := func(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
		return nil, expectedErr
	}

	_, err := m.WrapToolCall(ctx, req, handler)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error '%v', got '%v'", expectedErr, err)
	}

	// Verify spans
	spans := tracer.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	span := spans[0]
	if span.Status() != telemetry.StatusCodeError {
		t.Errorf("expected status Error, got %v", span.Status())
	}
}

func TestTelemetryMiddleware_SetDynamicConfig(t *testing.T) {
	m := NewTelemetryMiddleware(&TelemetryMiddlewareConfig{
		AgentID: "test-agent",
	})

	// Test SetConversationID
	m.SetConversationID("new-conv-123")
	if m.conversationID != "new-conv-123" {
		t.Errorf("expected conversation id 'new-conv-123', got '%s'", m.conversationID)
	}

	// Test SetModel
	m.SetModel("gpt-4-turbo")
	if m.model != "gpt-4-turbo" {
		t.Errorf("expected model 'gpt-4-turbo', got '%s'", m.model)
	}

	// Test SetProvider
	m.SetProvider(genai.ProviderOpenAI)
	if m.provider != genai.ProviderOpenAI {
		t.Errorf("expected provider '%s', got '%s'", genai.ProviderOpenAI, m.provider)
	}
}

func TestTelemetryMiddleware_Tools(t *testing.T) {
	m := NewTelemetryMiddleware(nil)
	tools := m.Tools()
	if tools != nil {
		t.Errorf("expected nil tools, got %v", tools)
	}
}

func TestTelemetryMiddleware_OnAgentLifecycle(t *testing.T) {
	m := NewTelemetryMiddleware(&TelemetryMiddlewareConfig{
		AgentID:   "test-agent",
		AgentName: "Test Agent",
	})

	ctx := context.Background()

	// Test OnAgentStart
	err := m.OnAgentStart(ctx, "test-agent")
	if err != nil {
		t.Errorf("unexpected error on start: %v", err)
	}

	// Test OnAgentStop
	err = m.OnAgentStop(ctx, "test-agent")
	if err != nil {
		t.Errorf("unexpected error on stop: %v", err)
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"timeout error", "timeout", true},
		{"rate limit exceeded", "rate limit", true},
		{"TIMEOUT ERROR", "timeout", true}, // case insensitive
		{"no match here", "timeout", false},
		{"", "timeout", false},
		{"timeout", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			got := contains(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestTelemetryMiddleware_ExtractTokenUsage(t *testing.T) {
	tracer := telemetry.NewSimpleTracer()

	m := NewTelemetryMiddleware(&TelemetryMiddlewareConfig{
		Tracer:  tracer,
		AgentID: "test-agent",
		Model:   "test-model",
	})

	ctx := context.Background()

	// Test with nested usage structure
	req := &ModelRequest{
		Messages: []types.Message{},
	}

	handler := func(ctx context.Context, req *ModelRequest) (*ModelResponse, error) {
		return &ModelResponse{
			Message: types.Message{},
			Metadata: map[string]any{
				"usage": map[string]any{
					"input_tokens":  100,
					"output_tokens": 50,
				},
			},
		}, nil
	}

	_, err := m.WrapModelCall(ctx, req, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := tracer.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	attrs := spans[0].Attributes()
	attrMap := make(map[string]any)
	for _, attr := range attrs {
		attrMap[attr.Key] = attr.Value
	}

	if attrMap[genai.AttrUsageInputTokens] != int64(100) {
		t.Errorf("expected input tokens 100, got %v", attrMap[genai.AttrUsageInputTokens])
	}
	if attrMap[genai.AttrUsageOutputTokens] != int64(50) {
		t.Errorf("expected output tokens 50, got %v", attrMap[genai.AttrUsageOutputTokens])
	}
}
