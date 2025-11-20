package context

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestContextInjector_InjectDateTime(t *testing.T) {
	config := InjectorConfig{
		EnableDateTime: true,
	}

	injector := NewContextInjector(config)
	systemPrompt := "You are a helpful assistant."

	result := injector.InjectContext(context.Background(), systemPrompt)

	if !strings.Contains(result, "Current Date and Time") {
		t.Error("expected date time context to be injected")
	}

	if !strings.Contains(result, "Date:") {
		t.Error("expected date to be included")
	}

	if !strings.Contains(result, "Time:") {
		t.Error("expected time to be included")
	}
}

func TestContextInjector_InjectLocation(t *testing.T) {
	location := &LocationInfo{
		City:     "San Francisco",
		Country:  "USA",
		Timezone: "America/Los_Angeles",
		Language: "en-US",
	}

	config := InjectorConfig{
		EnableLocation:   true,
		LocationProvider: NewSimpleLocationProvider(location),
	}

	injector := NewContextInjector(config)
	systemPrompt := "You are a helpful assistant."

	result := injector.InjectContext(context.Background(), systemPrompt)

	if !strings.Contains(result, "Location Information") {
		t.Error("expected location context to be injected")
	}

	if !strings.Contains(result, "San Francisco") {
		t.Error("expected city to be included")
	}

	if !strings.Contains(result, "USA") {
		t.Error("expected country to be included")
	}
}

func TestContextInjector_InjectSessionState(t *testing.T) {
	state := &SessionState{
		SessionID:    "session-123",
		MessageCount: 42,
		Duration:     30 * time.Minute,
		CustomState: map[string]interface{}{
			"mode": "debug",
		},
	}

	config := InjectorConfig{
		EnableSessionState:   true,
		SessionStateProvider: NewSimpleSessionStateProvider(state),
	}

	injector := NewContextInjector(config)
	systemPrompt := "You are a helpful assistant."

	result := injector.InjectContext(context.Background(), systemPrompt)

	if !strings.Contains(result, "Session State") {
		t.Error("expected session state context to be injected")
	}

	if !strings.Contains(result, "session-123") {
		t.Error("expected session ID to be included")
	}

	if !strings.Contains(result, "42") {
		t.Error("expected message count to be included")
	}
}

func TestContextInjector_MultipleContexts(t *testing.T) {
	config := InjectorConfig{
		EnableDateTime:   true,
		EnableSystemInfo: true,
		CustomContext:    "Custom context information",
	}

	injector := NewContextInjector(config)
	systemPrompt := "You are a helpful assistant."

	result := injector.InjectContext(context.Background(), systemPrompt)

	if !strings.Contains(result, "Current Date and Time") {
		t.Error("expected date time context")
	}

	if !strings.Contains(result, "System Information") {
		t.Error("expected system info context")
	}

	if !strings.Contains(result, "Custom context information") {
		t.Error("expected custom context")
	}
}
