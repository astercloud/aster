package middleware

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/astercloud/aster/pkg/backends"
	"github.com/astercloud/aster/pkg/types"
)

// mockBackend 模拟后端
type mockBackend struct {
	files       map[string]string
	shouldFail  bool
	readCalled  int
}

func newMockBackend() *mockBackend {
	return &mockBackend{
		files:      make(map[string]string),
		shouldFail: false,
		readCalled: 0,
	}
}

func (m *mockBackend) ListInfo(ctx context.Context, path string) ([]backends.FileInfo, error) {
	if m.shouldFail {
		return nil, errors.New("mock backend failure")
	}
	return []backends.FileInfo{}, nil
}

func (m *mockBackend) Read(ctx context.Context, path string, offset, limit int) (string, error) {
	m.readCalled++
	if m.shouldFail {
		return "", errors.New("mock backend failure")
	}
	content, exists := m.files[path]
	if !exists {
		return "", errors.New("file not found")
	}
	return content, nil
}

func (m *mockBackend) Write(ctx context.Context, path, content string) (*backends.WriteResult, error) {
	if m.shouldFail {
		return nil, errors.New("mock backend failure")
	}
	m.files[path] = content
	return &backends.WriteResult{
		Error:        "", // 空字符串表示成功
		Path:         path,
		BytesWritten: int64(len(content)),
	}, nil
}

func (m *mockBackend) Edit(ctx context.Context, path, oldStr, newStr string, replaceAll bool) (*backends.EditResult, error) {
	if m.shouldFail {
		return nil, errors.New("mock backend failure")
	}
	return &backends.EditResult{
		Error:            "", // 空字符串表示成功
		Path:             path,
		ReplacementsMade: 0,
	}, nil
}

func (m *mockBackend) GrepRaw(ctx context.Context, pattern, path, glob string) ([]backends.GrepMatch, error) {
	if m.shouldFail {
		return nil, errors.New("mock backend failure")
	}
	return []backends.GrepMatch{}, nil
}

func (m *mockBackend) GlobInfo(ctx context.Context, pattern, path string) ([]backends.FileInfo, error) {
	if m.shouldFail {
		return nil, errors.New("mock backend failure")
	}
	return []backends.FileInfo{}, nil
}

// TestAgentMemoryMiddleware_LoadMemory 测试加载记忆
func TestAgentMemoryMiddleware_LoadMemory(t *testing.T) {
	ctx := context.Background()

	backend := newMockBackend()
	backend.files["/agent.md"] = "You are a helpful coding assistant. You prefer Python over JavaScript."

	middleware, err := NewAgentMemoryMiddleware(&AgentMemoryMiddlewareConfig{
		Backend:    backend,
		MemoryPath: "/memories/",
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// 触发加载
	err = middleware.OnAgentStart(ctx, "test-agent")
	if err != nil {
		t.Fatalf("OnAgentStart failed: %v", err)
	}

	// 验证记忆已加载
	if !middleware.IsMemoryLoaded() {
		t.Error("Memory should be loaded")
	}

	content := middleware.GetMemoryContent()
	if !strings.Contains(content, "helpful coding assistant") {
		t.Errorf("Expected memory content to contain 'helpful coding assistant', got: %s", content)
	}

	// 验证只读取一次
	if backend.readCalled != 1 {
		t.Errorf("Expected backend.Read to be called once, got %d calls", backend.readCalled)
	}
}

// TestAgentMemoryMiddleware_FileNotFound 测试文件不存在的情况
func TestAgentMemoryMiddleware_FileNotFound(t *testing.T) {
	ctx := context.Background()

	backend := newMockBackend()
	// 不添加 agent.md 文件

	middleware, err := NewAgentMemoryMiddleware(&AgentMemoryMiddlewareConfig{
		Backend:    backend,
		MemoryPath: "/memories/",
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// 触发加载(文件不存在)
	err = middleware.OnAgentStart(ctx, "test-agent")
	if err != nil {
		t.Fatalf("OnAgentStart should not return error when file not found: %v", err)
	}

	// 验证记忆标记为已加载,但内容为空
	if !middleware.IsMemoryLoaded() {
		t.Error("Memory should be marked as loaded even if file not found")
	}

	content := middleware.GetMemoryContent()
	if content != "" {
		t.Errorf("Expected empty memory content, got: %s", content)
	}
}

// TestAgentMemoryMiddleware_InjectToSystemPrompt 测试注入到 SystemPrompt
func TestAgentMemoryMiddleware_InjectToSystemPrompt(t *testing.T) {
	ctx := context.Background()

	backend := newMockBackend()
	backend.files["/agent.md"] = "You are Claude, an AI assistant."

	middleware, err := NewAgentMemoryMiddleware(&AgentMemoryMiddlewareConfig{
		Backend:    backend,
		MemoryPath: "/memories/",
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// 创建请求
	req := &ModelRequest{
		SystemPrompt: "Additional instructions here.",
		Messages: []types.Message{
			{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Hello"}}},
		},
	}

	// 在 handler 中捕获注入后的 SystemPrompt
	var capturedSystemPrompt string
	handler := func(ctx context.Context, req *ModelRequest) (*ModelResponse, error) {
		capturedSystemPrompt = req.SystemPrompt
		return &ModelResponse{
			Message: types.Message{
				Role:    types.MessageRoleAssistant,
				ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Response"}},
			},
		}, nil
	}

	_, err = middleware.WrapModelCall(ctx, req, handler)
	if err != nil {
		t.Fatalf("WrapModelCall failed: %v", err)
	}

	// 验证 system prompt 被注入 (检查 handler 中捕获的值)
	if !strings.Contains(capturedSystemPrompt, "<agent_memory>") {
		t.Error("Expected SystemPrompt to contain <agent_memory> tag")
	}

	if !strings.Contains(capturedSystemPrompt, "You are Claude") {
		t.Error("Expected SystemPrompt to contain agent memory content")
	}

	if !strings.Contains(capturedSystemPrompt, "Additional instructions here") {
		t.Error("Expected SystemPrompt to preserve original content")
	}

	if !strings.Contains(capturedSystemPrompt, "Long-term Memory") {
		t.Error("Expected SystemPrompt to contain long-term memory guide")
	}

	// 验证调用后 SystemPrompt 已恢复原始值
	if req.SystemPrompt != "Additional instructions here." {
		t.Errorf("Expected SystemPrompt to be restored after call, got: %s", req.SystemPrompt)
	}
}

// TestAgentMemoryMiddleware_EmptySystemPrompt 测试空 SystemPrompt 的情况
func TestAgentMemoryMiddleware_EmptySystemPrompt(t *testing.T) {
	ctx := context.Background()

	backend := newMockBackend()
	backend.files["/agent.md"] = "Memory content"

	middleware, err := NewAgentMemoryMiddleware(&AgentMemoryMiddlewareConfig{
		Backend:    backend,
		MemoryPath: "/memories/",
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	req := &ModelRequest{
		SystemPrompt: "", // 空的 system prompt
		Messages:     []types.Message{},
	}

	// 在 handler 中捕获注入后的 SystemPrompt
	var capturedSystemPrompt string
	handler := func(ctx context.Context, req *ModelRequest) (*ModelResponse, error) {
		capturedSystemPrompt = req.SystemPrompt
		return &ModelResponse{
			Message: types.Message{
				Role:    types.MessageRoleAssistant,
				ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Response"}},
			},
		}, nil
	}

	_, err = middleware.WrapModelCall(ctx, req, handler)
	if err != nil {
		t.Fatalf("WrapModelCall failed: %v", err)
	}

	// 验证 system prompt 被设置 (检查 handler 中捕获的值)
	if capturedSystemPrompt == "" {
		t.Error("Expected SystemPrompt to be set with memory content")
	}

	if !strings.Contains(capturedSystemPrompt, "Memory content") {
		t.Error("Expected SystemPrompt to contain memory content")
	}

	// 验证调用后 SystemPrompt 恢复为空
	if req.SystemPrompt != "" {
		t.Errorf("Expected SystemPrompt to be restored to empty after call, got: %s", req.SystemPrompt)
	}
}

// TestAgentMemoryMiddleware_LoadOnlyOnce 测试只加载一次
func TestAgentMemoryMiddleware_LoadOnlyOnce(t *testing.T) {
	ctx := context.Background()

	backend := newMockBackend()
	backend.files["/agent.md"] = "Initial content"

	middleware, err := NewAgentMemoryMiddleware(&AgentMemoryMiddlewareConfig{
		Backend:    backend,
		MemoryPath: "/memories/",
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// 第一次加载
	err = middleware.OnAgentStart(ctx, "agent-1")
	if err != nil {
		t.Fatalf("OnAgentStart failed: %v", err)
	}

	// 修改后端内容
	backend.files["/agent.md"] = "Updated content"

	// 第二次调用 OnAgentStart,应该不重新加载
	err = middleware.OnAgentStart(ctx, "agent-2")
	if err != nil {
		t.Fatalf("OnAgentStart failed: %v", err)
	}

	// 验证内容仍然是初始内容
	content := middleware.GetMemoryContent()
	if strings.Contains(content, "Updated content") {
		t.Error("Memory should not be reloaded on second OnAgentStart call")
	}

	if !strings.Contains(content, "Initial content") {
		t.Error("Memory should still contain initial content")
	}

	// 验证只读取了一次
	if backend.readCalled != 1 {
		t.Errorf("Expected backend.Read to be called once, got %d calls", backend.readCalled)
	}
}

// TestAgentMemoryMiddleware_ReloadMemory 测试重新加载记忆
func TestAgentMemoryMiddleware_ReloadMemory(t *testing.T) {
	ctx := context.Background()

	backend := newMockBackend()
	backend.files["/agent.md"] = "Initial content"

	middleware, err := NewAgentMemoryMiddleware(&AgentMemoryMiddlewareConfig{
		Backend:    backend,
		MemoryPath: "/memories/",
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// 第一次加载
	err = middleware.OnAgentStart(ctx, "agent-1")
	if err != nil {
		t.Fatalf("OnAgentStart failed: %v", err)
	}

	// 修改后端内容
	backend.files["/agent.md"] = "Updated content"

	// 重新加载
	err = middleware.ReloadMemory(ctx)
	if err != nil {
		t.Fatalf("ReloadMemory failed: %v", err)
	}

	// 验证内容已更新
	content := middleware.GetMemoryContent()
	if !strings.Contains(content, "Updated content") {
		t.Errorf("Expected memory to contain 'Updated content', got: %s", content)
	}

	// 验证读取了两次
	if backend.readCalled != 2 {
		t.Errorf("Expected backend.Read to be called twice, got %d calls", backend.readCalled)
	}
}

// TestAgentMemoryMiddleware_CustomTemplate 测试自定义模板
func TestAgentMemoryMiddleware_CustomTemplate(t *testing.T) {
	ctx := context.Background()

	backend := newMockBackend()
	backend.files["/agent.md"] = "Test content"

	middleware, err := NewAgentMemoryMiddleware(&AgentMemoryMiddlewareConfig{
		Backend:              backend,
		MemoryPath:           "/memories/",
		SystemPromptTemplate: "### Agent Memory\n%s\n### End Memory",
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	req := &ModelRequest{
		SystemPrompt: "",
		Messages:     []types.Message{},
	}

	// 在 handler 中捕获注入后的 SystemPrompt
	var capturedSystemPrompt string
	handler := func(ctx context.Context, req *ModelRequest) (*ModelResponse, error) {
		capturedSystemPrompt = req.SystemPrompt
		return &ModelResponse{
			Message: types.Message{
				Role:    types.MessageRoleAssistant,
				ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Response"}},
			},
		}, nil
	}

	_, err = middleware.WrapModelCall(ctx, req, handler)
	if err != nil {
		t.Fatalf("WrapModelCall failed: %v", err)
	}

	// 验证使用了自定义模板 (检查 handler 中捕获的值)
	if !strings.Contains(capturedSystemPrompt, "### Agent Memory") {
		t.Error("Expected SystemPrompt to use custom template")
	}

	if !strings.Contains(capturedSystemPrompt, "### End Memory") {
		t.Error("Expected SystemPrompt to use custom template end marker")
	}
}

// TestAgentMemoryMiddleware_Config 测试配置方法
func TestAgentMemoryMiddleware_Config(t *testing.T) {
	backend := newMockBackend()
	backend.files["/agent.md"] = "Test"

	middleware, err := NewAgentMemoryMiddleware(&AgentMemoryMiddlewareConfig{
		Backend:    backend,
		MemoryPath: "/my-memories/",
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	config := middleware.GetConfig()

	if config["memory_path"] != "/my-memories/" {
		t.Errorf("Expected memory_path '/my-memories/', got '%v'", config["memory_path"])
	}

	if config["memory_file"] != "/agent.md" {
		t.Errorf("Expected memory_file '/agent.md', got '%v'", config["memory_file"])
	}

	if config["memory_loaded"].(bool) {
		t.Error("Expected memory_loaded to be false initially")
	}
}

// TestAgentMemoryMiddleware_DefaultConfig 测试默认配置
func TestAgentMemoryMiddleware_DefaultConfig(t *testing.T) {
	backend := newMockBackend()

	middleware, err := NewAgentMemoryMiddleware(&AgentMemoryMiddlewareConfig{
		Backend: backend,
		// MemoryPath 和 SystemPromptTemplate 使用默认值
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	config := middleware.GetConfig()

	if config["memory_path"] != "/memories/" {
		t.Errorf("Expected default memory_path '/memories/', got '%v'", config["memory_path"])
	}
}

// TestAgentMemoryMiddleware_NilConfig 测试 nil 配置
func TestAgentMemoryMiddleware_NilConfig(t *testing.T) {
	_, err := NewAgentMemoryMiddleware(nil)
	if err == nil {
		t.Error("Expected error with nil config")
	}
}

// TestAgentMemoryMiddleware_NilBackend 测试 nil Backend
func TestAgentMemoryMiddleware_NilBackend(t *testing.T) {
	_, err := NewAgentMemoryMiddleware(&AgentMemoryMiddlewareConfig{
		Backend: nil,
	})
	if err == nil {
		t.Error("Expected error with nil Backend")
	}
}

// TestAgentMemoryMiddleware_LazyLoading 测试懒加载
func TestAgentMemoryMiddleware_LazyLoading(t *testing.T) {
	ctx := context.Background()

	backend := newMockBackend()
	backend.files["/agent.md"] = "Lazy loaded content"

	middleware, err := NewAgentMemoryMiddleware(&AgentMemoryMiddlewareConfig{
		Backend:    backend,
		MemoryPath: "/memories/",
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// 不调用 OnAgentStart,直接调用 WrapModelCall
	req := &ModelRequest{
		SystemPrompt: "Test",
		Messages:     []types.Message{},
	}

	// 在 handler 中捕获注入后的 SystemPrompt
	var capturedSystemPrompt string
	handler := func(ctx context.Context, req *ModelRequest) (*ModelResponse, error) {
		capturedSystemPrompt = req.SystemPrompt
		return &ModelResponse{
			Message: types.Message{
				Role:    types.MessageRoleAssistant,
				ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Response"}},
			},
		}, nil
	}

	_, err = middleware.WrapModelCall(ctx, req, handler)
	if err != nil {
		t.Fatalf("WrapModelCall failed: %v", err)
	}

	// 验证记忆被懒加载
	if !middleware.IsMemoryLoaded() {
		t.Error("Memory should be lazy loaded on first WrapModelCall")
	}

	// 验证 handler 中看到的 SystemPrompt 包含懒加载的内容
	if !strings.Contains(capturedSystemPrompt, "Lazy loaded content") {
		t.Error("Expected SystemPrompt to contain lazy loaded content")
	}

	// 验证调用后 SystemPrompt 恢复为原始值
	if req.SystemPrompt != "Test" {
		t.Errorf("Expected SystemPrompt to be restored to 'Test' after call, got: %s", req.SystemPrompt)
	}
}
