package agent

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/types"
)

// MockProvider 模拟 Provider
type MockProvider struct {
	name         string
	shouldFail   bool
	failCount    int
	currentFails int
	completeFunc func(context.Context, []types.Message, *provider.StreamOptions) (*provider.CompleteResponse, error)
	streamFunc   func(context.Context, []types.Message, *provider.StreamOptions) (<-chan provider.StreamChunk, error)
	capabilities provider.ProviderCapabilities
	systemPrompt string
}

func (m *MockProvider) Complete(ctx context.Context, messages []types.Message, opts *provider.StreamOptions) (*provider.CompleteResponse, error) {
	if m.completeFunc != nil {
		return m.completeFunc(ctx, messages, opts)
	}

	if m.shouldFail {
		if m.failCount > 0 && m.currentFails < m.failCount {
			m.currentFails++
			return nil, errors.New("mock provider error")
		}
	}

	return &provider.CompleteResponse{
		Message: types.Message{
			Role:    "assistant",
			Content: "mock response from " + m.name,
		},
	}, nil
}

func (m *MockProvider) Stream(ctx context.Context, messages []types.Message, opts *provider.StreamOptions) (<-chan provider.StreamChunk, error) {
	if m.streamFunc != nil {
		return m.streamFunc(ctx, messages, opts)
	}

	if m.shouldFail {
		if m.failCount > 0 && m.currentFails < m.failCount {
			m.currentFails++
			return nil, errors.New("mock provider stream error")
		}
	}

	ch := make(chan provider.StreamChunk, 1)
	go func() {
		defer close(ch)
		ch <- provider.StreamChunk{
			Type:      "text",
			TextDelta: "mock stream from " + m.name,
		}
	}()

	return ch, nil
}

func (m *MockProvider) Config() *types.ModelConfig {
	return &types.ModelConfig{
		Provider: "mock",
		Model:    m.name,
	}
}

func (m *MockProvider) Capabilities() provider.ProviderCapabilities {
	return m.capabilities
}

func (m *MockProvider) SetSystemPrompt(prompt string) error {
	m.systemPrompt = prompt
	return nil
}

func (m *MockProvider) GetSystemPrompt() string {
	return m.systemPrompt
}

func (m *MockProvider) Close() error {
	return nil
}

// MockProviderFactory 模拟 ProviderFactory
type MockProviderFactory struct {
	providers map[string]*MockProvider
}

func NewMockProviderFactory() *MockProviderFactory {
	return &MockProviderFactory{
		providers: make(map[string]*MockProvider),
	}
}

func (f *MockProviderFactory) Create(config *types.ModelConfig) (provider.Provider, error) {
	key := config.Provider + "/" + config.Model

	if p, ok := f.providers[key]; ok {
		return p, nil
	}

	// 创建新的 mock provider
	p := &MockProvider{
		name: key,
		capabilities: provider.ProviderCapabilities{
			SupportStreaming:    true,
			SupportToolCalling:  true,
			SupportSystemPrompt: true,
		},
	}
	f.providers[key] = p
	return p, nil
}

func (f *MockProviderFactory) SetProvider(key string, p *MockProvider) {
	f.providers[key] = p
}

func TestModelFallbackManager_BasicFallback(t *testing.T) {
	// 创建 mock factory
	factory := NewMockProviderFactory()

	// 设置第一个模型失败
	factory.SetProvider("openai/gpt-4", &MockProvider{
		name:       "openai/gpt-4",
		shouldFail: true,
		failCount:  999, // 总是失败
	})

	// 第二个模型成功
	factory.SetProvider("anthropic/claude-3", &MockProvider{
		name: "anthropic/claude-3",
	})

	deps := &Dependencies{
		ProviderFactory: factory,
	}

	// 创建降级配置
	fallbacks := []*ModelFallback{
		{
			Config: &types.ModelConfig{
				Provider: "openai",
				Model:    "gpt-4",
			},
			MaxRetries: 1,
			Enabled:    true,
			Priority:   1,
		},
		{
			Config: &types.ModelConfig{
				Provider: "anthropic",
				Model:    "claude-3",
			},
			MaxRetries: 0,
			Enabled:    true,
			Priority:   2,
		},
	}

	manager, err := NewModelFallbackManager(fallbacks, deps)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// 执行请求
	ctx := context.Background()
	messages := []types.Message{
		{Role: "user", Content: "Hello"},
	}

	resp, err := manager.Complete(ctx, messages, nil)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	if resp.Message.Content != "mock response from anthropic/claude-3" {
		t.Errorf("Expected response from claude-3, got: %s", resp.Message.Content)
	}

	// 检查统计信息
	stats := manager.GetStats()
	if stats.FallbackCount != 1 {
		t.Errorf("Expected 1 fallback, got: %d", stats.FallbackCount)
	}

	if stats.SuccessRequests != 1 {
		t.Errorf("Expected 1 success, got: %d", stats.SuccessRequests)
	}
}

func TestModelFallbackManager_AllModelsFail(t *testing.T) {
	factory := NewMockProviderFactory()

	// 所有模型都失败
	factory.SetProvider("openai/gpt-4", &MockProvider{
		name:       "openai/gpt-4",
		shouldFail: true,
		failCount:  999,
	})

	factory.SetProvider("anthropic/claude-3", &MockProvider{
		name:       "anthropic/claude-3",
		shouldFail: true,
		failCount:  999,
	})

	deps := &Dependencies{
		ProviderFactory: factory,
	}

	fallbacks := []*ModelFallback{
		{
			Config: &types.ModelConfig{
				Provider: "openai",
				Model:    "gpt-4",
			},
			MaxRetries: 1,
			Enabled:    true,
			Priority:   1,
		},
		{
			Config: &types.ModelConfig{
				Provider: "anthropic",
				Model:    "claude-3",
			},
			MaxRetries: 1,
			Enabled:    true,
			Priority:   2,
		},
	}

	manager, err := NewModelFallbackManager(fallbacks, deps)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	ctx := context.Background()
	messages := []types.Message{
		{Role: "user", Content: "Hello"},
	}

	_, err = manager.Complete(ctx, messages, nil)
	if err == nil {
		t.Fatal("Expected error when all models fail")
	}

	// 检查统计信息
	stats := manager.GetStats()
	if stats.FailedRequests != 1 {
		t.Errorf("Expected 1 failed request, got: %d", stats.FailedRequests)
	}
}

func TestModelFallbackManager_RetrySuccess(t *testing.T) {
	factory := NewMockProviderFactory()

	// 第一次失败，第二次成功
	factory.SetProvider("openai/gpt-4", &MockProvider{
		name:       "openai/gpt-4",
		shouldFail: true,
		failCount:  1, // 只失败一次
	})

	deps := &Dependencies{
		ProviderFactory: factory,
	}

	fallbacks := []*ModelFallback{
		{
			Config: &types.ModelConfig{
				Provider: "openai",
				Model:    "gpt-4",
			},
			MaxRetries: 2,
			Enabled:    true,
			Priority:   1,
		},
	}

	manager, err := NewModelFallbackManager(fallbacks, deps)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	ctx := context.Background()
	messages := []types.Message{
		{Role: "user", Content: "Hello"},
	}

	resp, err := manager.Complete(ctx, messages, nil)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	if resp.Message.Content != "mock response from openai/gpt-4" {
		t.Errorf("Expected response from gpt-4, got: %s", resp.Message.Content)
	}

	// 检查统计信息
	stats := manager.GetStats()
	if stats.FallbackCount != 0 {
		t.Errorf("Expected 0 fallbacks (retry succeeded), got: %d", stats.FallbackCount)
	}
}

func TestModelFallbackManager_EnableDisableModel(t *testing.T) {
	factory := NewMockProviderFactory()

	factory.SetProvider("openai/gpt-4", &MockProvider{
		name: "openai/gpt-4",
	})

	factory.SetProvider("anthropic/claude-3", &MockProvider{
		name: "anthropic/claude-3",
	})

	deps := &Dependencies{
		ProviderFactory: factory,
	}

	fallbacks := []*ModelFallback{
		{
			Config: &types.ModelConfig{
				Provider: "openai",
				Model:    "gpt-4",
			},
			MaxRetries: 0,
			Enabled:    true,
			Priority:   1,
		},
		{
			Config: &types.ModelConfig{
				Provider: "anthropic",
				Model:    "claude-3",
			},
			MaxRetries: 0,
			Enabled:    true,
			Priority:   2,
		},
	}

	manager, err := NewModelFallbackManager(fallbacks, deps)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// 禁用第一个模型
	err = manager.DisableModel("openai", "gpt-4")
	if err != nil {
		t.Fatalf("Failed to disable model: %v", err)
	}

	// 执行请求，应该使用第二个模型
	ctx := context.Background()
	messages := []types.Message{
		{Role: "user", Content: "Hello"},
	}

	resp, err := manager.Complete(ctx, messages, nil)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	if resp.Message.Content != "mock response from anthropic/claude-3" {
		t.Errorf("Expected response from claude-3, got: %s", resp.Message.Content)
	}

	// 重新启用第一个模型
	err = manager.EnableModel("openai", "gpt-4")
	if err != nil {
		t.Fatalf("Failed to enable model: %v", err)
	}

	// 执行请求，应该使用第一个模型（优先级更高）
	resp, err = manager.Complete(ctx, messages, nil)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	if resp.Message.Content != "mock response from openai/gpt-4" {
		t.Errorf("Expected response from gpt-4, got: %s", resp.Message.Content)
	}
}

func TestModelFallbackManager_Stream(t *testing.T) {
	factory := NewMockProviderFactory()

	// 第一个模型失败
	factory.SetProvider("openai/gpt-4", &MockProvider{
		name:       "openai/gpt-4",
		shouldFail: true,
		failCount:  999,
	})

	// 第二个模型成功
	factory.SetProvider("anthropic/claude-3", &MockProvider{
		name: "anthropic/claude-3",
	})

	deps := &Dependencies{
		ProviderFactory: factory,
	}

	fallbacks := []*ModelFallback{
		{
			Config: &types.ModelConfig{
				Provider: "openai",
				Model:    "gpt-4",
			},
			MaxRetries: 0,
			Enabled:    true,
			Priority:   1,
		},
		{
			Config: &types.ModelConfig{
				Provider: "anthropic",
				Model:    "claude-3",
			},
			MaxRetries: 0,
			Enabled:    true,
			Priority:   2,
		},
	}

	manager, err := NewModelFallbackManager(fallbacks, deps)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	ctx := context.Background()
	messages := []types.Message{
		{Role: "user", Content: "Hello"},
	}

	stream, err := manager.Stream(ctx, messages, nil)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	// 读取流
	var content string
	var contentSb453 strings.Builder
	for chunk := range stream {
		if chunk.Type == "text" {
			contentSb453.WriteString(chunk.TextDelta)
		}
	}
	content += contentSb453.String()

	if content != "mock stream from anthropic/claude-3" {
		t.Errorf("Expected stream from claude-3, got: %s", content)
	}

	// 检查统计信息
	stats := manager.GetStats()
	if stats.FallbackCount != 1 {
		t.Errorf("Expected 1 fallback, got: %d", stats.FallbackCount)
	}
}

func TestModelFallbackManager_PriorityOrdering(t *testing.T) {
	factory := NewMockProviderFactory()

	factory.SetProvider("openai/gpt-4", &MockProvider{
		name: "openai/gpt-4",
	})

	factory.SetProvider("anthropic/claude-3", &MockProvider{
		name: "anthropic/claude-3",
	})

	factory.SetProvider("google/gemini", &MockProvider{
		name: "google/gemini",
	})

	deps := &Dependencies{
		ProviderFactory: factory,
	}

	// 故意打乱优先级顺序
	fallbacks := []*ModelFallback{
		{
			Config: &types.ModelConfig{
				Provider: "google",
				Model:    "gemini",
			},
			MaxRetries: 0,
			Enabled:    true,
			Priority:   3, // 最低优先级
		},
		{
			Config: &types.ModelConfig{
				Provider: "openai",
				Model:    "gpt-4",
			},
			MaxRetries: 0,
			Enabled:    true,
			Priority:   1, // 最高优先级
		},
		{
			Config: &types.ModelConfig{
				Provider: "anthropic",
				Model:    "claude-3",
			},
			MaxRetries: 0,
			Enabled:    true,
			Priority:   2, // 中等优先级
		},
	}

	manager, err := NewModelFallbackManager(fallbacks, deps)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// 执行请求，应该使用优先级最高的模型
	ctx := context.Background()
	messages := []types.Message{
		{Role: "user", Content: "Hello"},
	}

	resp, err := manager.Complete(ctx, messages, nil)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	if resp.Message.Content != "mock response from openai/gpt-4" {
		t.Errorf("Expected response from gpt-4 (highest priority), got: %s", resp.Message.Content)
	}
}

func TestModelFallbackManager_ContextCancellation(t *testing.T) {
	factory := NewMockProviderFactory()

	// 模拟慢速 provider
	factory.SetProvider("openai/gpt-4", &MockProvider{
		name: "openai/gpt-4",
		completeFunc: func(ctx context.Context, messages []types.Message, opts *provider.StreamOptions) (*provider.CompleteResponse, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(5 * time.Second):
				return &provider.CompleteResponse{
					Message: types.Message{
						Role:    "assistant",
						Content: "slow response",
					},
				}, nil
			}
		},
	})

	deps := &Dependencies{
		ProviderFactory: factory,
	}

	fallbacks := []*ModelFallback{
		{
			Config: &types.ModelConfig{
				Provider: "openai",
				Model:    "gpt-4",
			},
			MaxRetries: 0,
			Enabled:    true,
			Priority:   1,
		},
	}

	manager, err := NewModelFallbackManager(fallbacks, deps)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// 创建可取消的 context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	messages := []types.Message{
		{Role: "user", Content: "Hello"},
	}

	_, err = manager.Complete(ctx, messages, nil)
	if err == nil {
		t.Fatal("Expected context cancellation error")
	}

	// 错误应该包含 context deadline exceeded
	if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected context deadline error, got: %v", err)
	}
}
