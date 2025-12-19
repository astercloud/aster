package middleware

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/astercloud/aster/pkg/memory"
	"github.com/astercloud/aster/pkg/memory/logic"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogicMemoryMiddleware(t *testing.T) {
	store := logic.NewInMemoryStore()
	manager, err := logic.NewManager(&logic.ManagerConfig{Store: store})
	require.NoError(t, err)

	t.Run("valid config", func(t *testing.T) {
		mw, err := NewLogicMemoryMiddleware(&LogicMemoryMiddlewareConfig{
			Manager:         manager,
			EnableCapture:   true,
			EnableInjection: true,
		})
		require.NoError(t, err)
		require.NotNil(t, mw)
		assert.Equal(t, "logic_memory", mw.Name())
	})

	t.Run("nil config", func(t *testing.T) {
		_, err := NewLogicMemoryMiddleware(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config is required")
	})

	t.Run("nil manager", func(t *testing.T) {
		_, err := NewLogicMemoryMiddleware(&LogicMemoryMiddlewareConfig{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "manager is required")
	})

	t.Run("default values", func(t *testing.T) {
		mw, err := NewLogicMemoryMiddleware(&LogicMemoryMiddlewareConfig{
			Manager: manager,
		})
		require.NoError(t, err)
		assert.Equal(t, 5, mw.config.MaxMemories)
		assert.Equal(t, 0.6, mw.config.MinConfidence)
		assert.Equal(t, 7, mw.config.Priority)
		assert.Equal(t, "system_prompt_end", mw.config.InjectionPoint)
	})
}

func TestLogicMemoryMiddleware_WrapModelCall(t *testing.T) {
	ctx := context.Background()
	store := logic.NewInMemoryStore()
	manager, err := logic.NewManager(&logic.ManagerConfig{Store: store})
	require.NoError(t, err)

	// 预先存储一些 Memory
	testMemory := &logic.LogicMemory{
		ID:          "mem-1",
		Namespace:   "user:123",
		Scope:       logic.ScopeUser,
		Type:        "preference",
		Key:         "tone",
		Value:       "casual",
		Description: "用户偏好口语化表达",
		Provenance: &memory.MemoryProvenance{
			SourceType: memory.SourceUserInput,
			Confidence: 0.85,
		},
	}
	err = store.Save(ctx, testMemory)
	require.NoError(t, err)

	t.Run("inject memory into system prompt", func(t *testing.T) {
		mw, err := NewLogicMemoryMiddleware(&LogicMemoryMiddlewareConfig{
			Manager:         manager,
			EnableInjection: true,
			MaxMemories:     5,
			MinConfidence:   0.6,
		})
		require.NoError(t, err)

		req := &ModelRequest{
			SystemPrompt: "You are a helpful assistant.",
			Metadata: map[string]any{
				"user_id": "123",
			},
		}

		var capturedSystemPrompt string
		handler := func(ctx context.Context, r *ModelRequest) (*ModelResponse, error) {
			// 保存 SystemPrompt 的副本，因为调用后会被恢复
			capturedSystemPrompt = r.SystemPrompt
			return &ModelResponse{}, nil
		}

		_, err = mw.WrapModelCall(ctx, req, handler)
		require.NoError(t, err)

		// 验证 Memory 被注入到 system prompt
		assert.Contains(t, capturedSystemPrompt, "User Preferences")
		assert.Contains(t, capturedSystemPrompt, "preference")
		assert.Contains(t, capturedSystemPrompt, "口语化")

		// 验证原始 system prompt 被恢复
		assert.Equal(t, "You are a helpful assistant.", req.SystemPrompt)
	})

	t.Run("skip injection when disabled", func(t *testing.T) {
		mw, err := NewLogicMemoryMiddleware(&LogicMemoryMiddlewareConfig{
			Manager:         manager,
			EnableInjection: false,
		})
		require.NoError(t, err)

		req := &ModelRequest{
			SystemPrompt: "You are a helpful assistant.",
			Metadata: map[string]any{
				"user_id": "123",
			},
		}

		var capturedReq *ModelRequest
		handler := func(ctx context.Context, r *ModelRequest) (*ModelResponse, error) {
			capturedReq = r
			return &ModelResponse{}, nil
		}

		_, err = mw.WrapModelCall(ctx, req, handler)
		require.NoError(t, err)

		// 验证 Memory 没有被注入
		assert.Equal(t, "You are a helpful assistant.", capturedReq.SystemPrompt)
	})

	t.Run("skip injection without namespace", func(t *testing.T) {
		mw, err := NewLogicMemoryMiddleware(&LogicMemoryMiddlewareConfig{
			Manager:         manager,
			EnableInjection: true,
		})
		require.NoError(t, err)

		req := &ModelRequest{
			SystemPrompt: "You are a helpful assistant.",
			Metadata:     nil, // 没有 metadata
		}

		var capturedReq *ModelRequest
		handler := func(ctx context.Context, r *ModelRequest) (*ModelResponse, error) {
			capturedReq = r
			return &ModelResponse{}, nil
		}

		_, err = mw.WrapModelCall(ctx, req, handler)
		require.NoError(t, err)

		// 验证 Memory 没有被注入（因为没有 namespace）
		assert.Equal(t, "You are a helpful assistant.", capturedReq.SystemPrompt)
	})

	t.Run("injection point start", func(t *testing.T) {
		mw, err := NewLogicMemoryMiddleware(&LogicMemoryMiddlewareConfig{
			Manager:         manager,
			EnableInjection: true,
			InjectionPoint:  "system_prompt_start",
		})
		require.NoError(t, err)

		req := &ModelRequest{
			SystemPrompt: "Original prompt.",
			Metadata: map[string]any{
				"user_id": "123",
			},
		}

		var capturedSystemPrompt string
		handler := func(ctx context.Context, r *ModelRequest) (*ModelResponse, error) {
			capturedSystemPrompt = r.SystemPrompt
			return &ModelResponse{}, nil
		}

		_, err = mw.WrapModelCall(ctx, req, handler)
		require.NoError(t, err)

		// 验证 Memory 在开头
		assert.Greater(t, len(capturedSystemPrompt), len("Original prompt."))
		// Memory 应该在原始 prompt 之前
		assert.Contains(t, capturedSystemPrompt, "User Preferences")
	})
}

func TestLogicMemoryMiddleware_WrapToolCall(t *testing.T) {
	ctx := context.Background()
	store := logic.NewInMemoryStore()
	manager, err := logic.NewManager(&logic.ManagerConfig{Store: store})
	require.NoError(t, err)

	t.Run("capture tool result event", func(t *testing.T) {
		mw, err := NewLogicMemoryMiddleware(&LogicMemoryMiddlewareConfig{
			Manager:       manager,
			EnableCapture: true,
			AsyncCapture:  false, // 同步捕获以便测试
		})
		require.NoError(t, err)

		req := &ToolCallRequest{
			ToolName:   "test_tool",
			ToolCallID: "call-123",
			ToolInput:  map[string]any{"key": "value"},
			Context: &tools.ToolContext{
				ThreadID: "thread-456",
			},
		}

		handler := func(ctx context.Context, r *ToolCallRequest) (*ToolCallResponse, error) {
			return &ToolCallResponse{Result: "success"}, nil
		}

		resp, err := mw.WrapToolCall(ctx, req, handler)
		require.NoError(t, err)
		assert.Equal(t, "success", resp.Result)
	})

	t.Run("skip capture when disabled", func(t *testing.T) {
		mw, err := NewLogicMemoryMiddleware(&LogicMemoryMiddlewareConfig{
			Manager:       manager,
			EnableCapture: false,
		})
		require.NoError(t, err)

		req := &ToolCallRequest{
			ToolName:   "test_tool",
			ToolCallID: "call-123",
			ToolInput:  map[string]any{"key": "value"},
		}

		handler := func(ctx context.Context, r *ToolCallRequest) (*ToolCallResponse, error) {
			return &ToolCallResponse{Result: "success"}, nil
		}

		resp, err := mw.WrapToolCall(ctx, req, handler)
		require.NoError(t, err)
		assert.Equal(t, "success", resp.Result)
	})
}

func TestLogicMemoryMiddleware_CaptureEvents(t *testing.T) {
	store := logic.NewInMemoryStore()

	// 创建一个简单的 matcher 用于测试
	matcher := &testPatternMatcher{
		supportedTypes: []string{"user_revision", "user_feedback", "user_message"},
	}

	manager, err := logic.NewManager(&logic.ManagerConfig{
		Store:    store,
		Matchers: []logic.PatternMatcher{matcher},
	})
	require.NoError(t, err)

	mw, err := NewLogicMemoryMiddleware(&LogicMemoryMiddlewareConfig{
		Manager:       manager,
		EnableCapture: true,
		AsyncCapture:  false, // 同步捕获
	})
	require.NoError(t, err)

	t.Run("capture user message", func(t *testing.T) {
		mw.CaptureUserMessage("user:123", "Hello, world!", map[string]any{
			"intent": "greeting",
		})

		// 验证事件被处理
		assert.True(t, matcher.isEventProcessed())
		assert.Equal(t, "user_message", matcher.getLastEventType())
		matcher.reset()
	})

	t.Run("capture user feedback", func(t *testing.T) {
		mw.CaptureUserFeedback("user:123", "Great response!", 5, nil)

		assert.True(t, matcher.isEventProcessed())
		assert.Equal(t, "user_feedback", matcher.getLastEventType())
		matcher.reset()
	})

	t.Run("capture user revision", func(t *testing.T) {
		mw.CaptureUserRevision("user:123", "然而，这是一个例子", "不过，这是一个例子", nil)

		assert.True(t, matcher.isEventProcessed())
		assert.Equal(t, "user_revision", matcher.getLastEventType())
		matcher.reset()
	})

	t.Run("capture generic event (unsupported type)", func(t *testing.T) {
		event := &logic.Event{
			Type:      "custom_event",
			Source:    "user:123",
			Data:      map[string]any{"key": "value"},
			Timestamp: time.Now(),
		}
		mw.CaptureEvent(event)

		// 自定义事件类型不在 matcher 支持列表中，matcher 不会被调用
		assert.False(t, matcher.isEventProcessed())
		matcher.reset()
	})
}

func TestLogicMemoryMiddleware_AsyncCapture(t *testing.T) {
	store := logic.NewInMemoryStore()

	matcher := &testPatternMatcher{
		supportedTypes: []string{"user_message"},
	}

	manager, err := logic.NewManager(&logic.ManagerConfig{
		Store:    store,
		Matchers: []logic.PatternMatcher{matcher},
	})
	require.NoError(t, err)

	mw, err := NewLogicMemoryMiddleware(&LogicMemoryMiddlewareConfig{
		Manager:         manager,
		EnableCapture:   true,
		AsyncCapture:    true,
		EventBufferSize: 10,
	})
	require.NoError(t, err)

	ctx := context.Background()

	// 启动异步处理器
	err = mw.OnAgentStart(ctx, "test-agent")
	require.NoError(t, err)

	// 发送事件
	mw.CaptureUserMessage("user:123", "Test message", nil)

	// 等待异步处理
	time.Sleep(100 * time.Millisecond)

	// 验证事件被处理
	assert.True(t, matcher.isEventProcessed())

	// 停止
	err = mw.OnAgentStop(ctx, "test-agent")
	require.NoError(t, err)
}

func TestLogicMemoryMiddleware_Tools(t *testing.T) {
	store := logic.NewInMemoryStore()
	manager, err := logic.NewManager(&logic.ManagerConfig{Store: store})
	require.NoError(t, err)

	mw, err := NewLogicMemoryMiddleware(&LogicMemoryMiddlewareConfig{
		Manager: manager,
	})
	require.NoError(t, err)

	tools := mw.Tools()
	assert.Len(t, tools, 2)

	// 验证工具名称
	toolNames := make([]string, len(tools))
	for i, tool := range tools {
		toolNames[i] = tool.Name()
	}
	assert.Contains(t, toolNames, "logic_memory_query")
	assert.Contains(t, toolNames, "logic_memory_update")
}

func TestLogicMemoryMiddleware_GetConfig(t *testing.T) {
	store := logic.NewInMemoryStore()
	manager, err := logic.NewManager(&logic.ManagerConfig{Store: store})
	require.NoError(t, err)

	mw, err := NewLogicMemoryMiddleware(&LogicMemoryMiddlewareConfig{
		Manager:         manager,
		EnableCapture:   true,
		EnableInjection: true,
		MaxMemories:     10,
		MinConfidence:   0.7,
		AsyncCapture:    true,
		InjectionPoint:  "system_prompt_start",
	})
	require.NoError(t, err)

	config := mw.GetConfig()
	assert.Equal(t, true, config["enable_capture"])
	assert.Equal(t, true, config["enable_injection"])
	assert.Equal(t, 10, config["max_memories"])
	assert.Equal(t, 0.7, config["min_confidence"])
	assert.Equal(t, true, config["async_capture"])
	assert.Equal(t, "system_prompt_start", config["injection_point"])
}

func TestDefaultNamespaceExtractor(t *testing.T) {
	t.Run("extract from namespace", func(t *testing.T) {
		req := &ModelRequest{
			Metadata: map[string]any{
				"namespace": "custom:namespace",
			},
		}
		ns := defaultNamespaceExtractor(req)
		assert.Equal(t, "custom:namespace", ns)
	})

	t.Run("extract from user_id", func(t *testing.T) {
		req := &ModelRequest{
			Metadata: map[string]any{
				"user_id": "123",
			},
		}
		ns := defaultNamespaceExtractor(req)
		assert.Equal(t, "user:123", ns)
	})

	t.Run("extract from tenant_id", func(t *testing.T) {
		req := &ModelRequest{
			Metadata: map[string]any{
				"tenant_id": "tenant-456",
			},
		}
		ns := defaultNamespaceExtractor(req)
		assert.Equal(t, "tenant:tenant-456", ns)
	})

	t.Run("extract from agent_id", func(t *testing.T) {
		req := &ModelRequest{
			Metadata: map[string]any{
				"agent_id": "agent-789",
			},
		}
		ns := defaultNamespaceExtractor(req)
		assert.Equal(t, "agent:agent-789", ns)
	})

	t.Run("nil metadata", func(t *testing.T) {
		req := &ModelRequest{
			Metadata: nil,
		}
		ns := defaultNamespaceExtractor(req)
		assert.Empty(t, ns)
	})

	t.Run("priority: namespace > user_id > tenant_id > agent_id", func(t *testing.T) {
		req := &ModelRequest{
			Metadata: map[string]any{
				"namespace": "ns:priority",
				"user_id":   "123",
				"tenant_id": "tenant-456",
				"agent_id":  "agent-789",
			},
		}
		ns := defaultNamespaceExtractor(req)
		assert.Equal(t, "ns:priority", ns)
	})
}

func TestBuildMemorySection(t *testing.T) {
	store := logic.NewInMemoryStore()
	manager, err := logic.NewManager(&logic.ManagerConfig{Store: store})
	require.NoError(t, err)

	mw, err := NewLogicMemoryMiddleware(&LogicMemoryMiddlewareConfig{
		Manager: manager,
	})
	require.NoError(t, err)

	t.Run("build with memories", func(t *testing.T) {
		memories := []*logic.LogicMemory{
			{
				Type:        "preference",
				Key:         "tone",
				Description: "用户偏好口语化表达",
				Provenance: &memory.MemoryProvenance{
					Confidence: 0.85,
				},
			},
			{
				Type:        "behavior",
				Key:         "format",
				Description: "用户喜欢使用列表格式",
				Category:    "formatting",
				Provenance: &memory.MemoryProvenance{
					Confidence: 0.7,
				},
			},
		}

		section := mw.buildMemorySection(memories)

		assert.Contains(t, section, "User Preferences")
		assert.Contains(t, section, "preference")
		assert.Contains(t, section, "tone")
		assert.Contains(t, section, "口语化")
		assert.Contains(t, section, "85%")
		assert.Contains(t, section, "behavior")
		assert.Contains(t, section, "format")
		assert.Contains(t, section, "列表格式")
		assert.Contains(t, section, "70%")
		assert.Contains(t, section, "formatting")
	})

	t.Run("empty memories", func(t *testing.T) {
		section := mw.buildMemorySection([]*logic.LogicMemory{})
		assert.Empty(t, section)
	})

	t.Run("nil provenance", func(t *testing.T) {
		memories := []*logic.LogicMemory{
			{
				Type:        "preference",
				Key:         "test",
				Description: "Test memory",
				Provenance:  nil,
			},
		}

		section := mw.buildMemorySection(memories)
		assert.Contains(t, section, "0%") // nil provenance -> 0 confidence
	})
}

// testPatternMatcher 测试用的 PatternMatcher
type testPatternMatcher struct {
	mu             sync.RWMutex
	supportedTypes []string
	memories       []*logic.LogicMemory
	err            error
	eventProcessed bool
	lastEventType  string
}

func (m *testPatternMatcher) MatchEvent(ctx context.Context, event logic.Event) ([]*logic.LogicMemory, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventProcessed = true
	m.lastEventType = event.Type
	if m.err != nil {
		return nil, m.err
	}
	return m.memories, nil
}

func (m *testPatternMatcher) SupportedEventTypes() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.supportedTypes
}

func (m *testPatternMatcher) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventProcessed = false
	m.lastEventType = ""
}

func (m *testPatternMatcher) isEventProcessed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.eventProcessed
}

func (m *testPatternMatcher) getLastEventType() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastEventType
}
