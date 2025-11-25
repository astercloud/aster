package integration

import (
	"context"
	"testing"
	"time"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/builtin"
	"github.com/astercloud/aster/pkg/types"
)

// MockProvider 用于集成测试的模拟LLM Provider
type MockProvider struct {
	responses []string
	current   int
}

func NewMockProvider(responses []string) *MockProvider {
	return &MockProvider{
		responses: responses,
		current:   0,
	}
}

func (m *MockProvider) Complete(ctx context.Context, req *provider.LLMRequest, options *provider.CompleteOptions) (*provider.LLMResponse, error) {
	if m.current >= len(m.responses) {
		m.current = 0 // 循环使用响应
	}

	response := &provider.LLMResponse{
		Content: m.responses[m.current],
		ToolCalls: []provider.ToolCall{
			{
				ID:   "mock-tool-call",
				Name: "Read",
				Args: map[string]interface{}{
					"file_path": "/tmp/test.txt",
				},
			},
		},
	}
	m.current++
	return response, nil
}

func (m *MockProvider) Stream(ctx context.Context, req *provider.LLMRequest, options *provider.StreamOptions) (<-chan provider.StreamChunk, error) {
	ch := make(chan provider.StreamChunk)

	go func() {
		defer close(ch)
		if m.current >= len(m.responses) {
			m.current = 0
		}

		// 模拟流式响应
		for _, char := range m.responses[m.current] {
			chunk := provider.StreamChunk{
				Content: string(char),
			}
			select {
			case ch <- chunk:
			case <-ctx.Done():
				return
			}
			time.Sleep(1 * time.Millisecond) // 模拟延迟
		}
		m.current++
	}()

	return ch, nil
}

func (m *MockProvider) Close() error {
	return nil
}

// SetupIntegrationDeps 创建集成测试的依赖项
func SetupIntegrationDeps(t *testing.T, mockResponses []string) (*agent.Dependencies, *MockProvider) {
	// 创建工具注册表
	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry)

	// 创建Sandbox工厂
	sandboxFactory := sandbox.NewFactory()

	// 创建Mock Provider
	mockProvider := NewMockProvider(mockResponses)
	providerFactory := &MockProviderFactory{provider: mockProvider}

	// 创建Store (使用临时目录)
	jsonStore, err := store.NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// 创建模板注册表
	templateRegistry := agent.NewTemplateRegistry()

	deps := &agent.Dependencies{
		Store:            jsonStore,
		SandboxFactory:   sandboxFactory,
		ToolRegistry:     toolRegistry,
		ProviderFactory:  providerFactory,
		TemplateRegistry: templateRegistry,
	}

	return deps, mockProvider
}

// MockProviderFactory 模拟Provider工厂
type MockProviderFactory struct {
	provider *MockProvider
}

func (f *MockProviderFactory) CreateProvider(config *types.ModelConfig) (provider.LLMProvider, error) {
	return f.provider, nil
}

func (f *MockProviderFactory) GetSupportedProviders() []string {
	return []string{"mock"}
}

// EventCollector 事件收集器，用于测试验证
type EventCollector struct {
	Events []types.Event
}

func NewEventCollector() *EventCollector {
	return &EventCollector{
		Events: make([]types.Event, 0),
	}
}

func (ec *EventCollector) Subscribe(ctx context.Context, eventTypes []types.EventType) (<-chan types.Event, error) {
	ch := make(chan types.Event, 100)

	go func() {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// 在实际测试中，这里应该从Agent订阅事件
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	return ch, nil
}

func (ec *EventCollector) Collect(event types.Event) {
	ec.Events = append(ec.Events, event)
}

func (ec *EventCollector) GetEventsByType(eventType types.EventType) []types.Event {
	var events []types.Event
	for _, event := range ec.Events {
		if event.Type == eventType {
			events = append(events, event)
		}
	}
	return events
}

func (ec *EventCollector) Clear() {
	ec.Events = make([]types.Event, 0)
}