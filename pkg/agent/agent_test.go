package agent

import (
	"context"
	"testing"
	"time"

	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/builtin"
	"github.com/astercloud/aster/pkg/types"
)

func TestAgentCreate(t *testing.T) {
	deps := setupTestDeps(t)

	config := &types.AgentConfig{
		TemplateID: "test-template",
		ModelConfig: &types.ModelConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-5",
			APIKey:   "test-key",
		},
		Sandbox: &types.SandboxConfig{
			Kind:    types.SandboxKindMock,
			WorkDir: "/tmp/test",
		},
	}

	ag, err := Create(context.Background(), config, deps)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	defer func() { _ = ag.Close() }()

	if ag.ID() == "" {
		t.Error("Agent ID should not be empty")
	}

	if ag.state != types.AgentStateReady {
		t.Errorf("Expected state Ready, got %s", ag.state)
	}
}

func TestAgentStatus(t *testing.T) {
	deps := setupTestDeps(t)

	config := &types.AgentConfig{
		TemplateID: "test-template",
		ModelConfig: &types.ModelConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-5",
			APIKey:   "test-key",
		},
		Sandbox: &types.SandboxConfig{
			Kind:    types.SandboxKindMock,
			WorkDir: "/tmp/test",
		},
	}

	ag, err := Create(context.Background(), config, deps)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	defer func() { _ = ag.Close() }()

	status := ag.Status()
	if status == nil {
		t.Fatal("Status should not be nil")
	}

	if status.AgentID != ag.ID() {
		t.Errorf("Expected AgentID %s, got %s", ag.ID(), status.AgentID)
	}

	if status.State != types.AgentStateReady {
		t.Errorf("Expected state Ready, got %s", status.State)
	}
}

func TestAgentEventBus(t *testing.T) {
	deps := setupTestDeps(t)

	config := &types.AgentConfig{
		TemplateID: "test-template",
		ModelConfig: &types.ModelConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-5",
			APIKey:   "test-key",
		},
		Sandbox: &types.SandboxConfig{
			Kind:    types.SandboxKindMock,
			WorkDir: "/tmp/test",
		},
	}

	ag, err := Create(context.Background(), config, deps)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	defer func() { _ = ag.Close() }()

	// 订阅事件
	eventCh := ag.Subscribe([]types.AgentChannel{types.ChannelMonitor}, nil)

	// 发送一个测试事件
	go func() {
		time.Sleep(100 * time.Millisecond)
		ag.eventBus.EmitMonitor(&types.MonitorStateChangedEvent{
			State: types.AgentStateWorking,
		})
	}()

	// 等待事件
	select {
	case envelope := <-eventCh:
		if envelope.Event == nil {
			t.Error("Event should not be nil")
		}
		if evt, ok := envelope.Event.(*types.MonitorStateChangedEvent); ok {
			if evt.State != types.AgentStateWorking {
				t.Errorf("Expected state Working, got %s", evt.State)
			}
		} else {
			t.Error("Event should be MonitorStateChangedEvent")
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for event")
	}
}

// setupTestDeps 创建测试依赖
func setupTestDeps(t *testing.T) *Dependencies {
	// 创建工具注册表
	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry)

	// 创建Sandbox工厂
	sandboxFactory := sandbox.NewFactory()

	// 创建Provider工厂
	providerFactory := &provider.AnthropicFactory{}

	// 创建Store (使用临时目录)
	jsonStore, err := store.NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// 创建模板注册表
	templateRegistry := NewTemplateRegistry()
	templateRegistry.Register(&types.AgentTemplateDefinition{
		ID:           "test-template",
		SystemPrompt: "You are a test assistant.",
		Model:        "claude-sonnet-4-5",
		Tools:        []interface{}{"Read", "Write"},
	})

	return &Dependencies{
		Store:            jsonStore,
		SandboxFactory:   sandboxFactory,
		ToolRegistry:     toolRegistry,
		ProviderFactory:  providerFactory,
		TemplateRegistry: templateRegistry,
	}
}
