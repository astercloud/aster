package middleware

import (
	"context"
	"strings"
	"testing"

	"github.com/astercloud/aster/pkg/backends"
)

// TestMiddlewareStack_Integration 集成测试: Middleware Stack
func TestMiddlewareStack_Integration(t *testing.T) {
	// 1. 创建 Backend
	backend := backends.NewStateBackend()

	// 2. 创建 FilesystemMiddleware
	fsMiddleware := NewFilesystemMiddleware(&FilesystemMiddlewareConfig{
		Backend:        backend,
		EnableEviction: true,
		TokenLimit:     100, // 低阈值用于测试
	})

	// 3. 创建 Stack
	stack := NewStack([]Middleware{
		fsMiddleware,
	})

	// 测试工具收集
	tools := stack.Tools()
	if len(tools) != 6 {
		t.Errorf("Expected 6 tools, got %d", len(tools))
	}

	// 验证工具名称
	expectedTools := map[string]bool{
		"Read":  true,
		"Write": true,
		"Ls":    true,
		"Edit":  true,
		"Glob":  true,
		"Grep":  true,
	}

	for _, tool := range tools {
		if !expectedTools[tool.Name()] {
			t.Errorf("Unexpected tool: %s", tool.Name())
		}
	}
}

// TestSubAgentMiddleware_Integration 集成测试: SubAgent Middleware
func TestSubAgentMiddleware_Integration(t *testing.T) {
	ctx := context.Background()

	// 1. 定义子代理规格
	specs := []SubAgentSpec{
		{
			Name:        "test-agent",
			Description: "测试用子代理",
			Prompt:      "Test prompt",
		},
	}

	// 2. 创建工厂
	factory := func(ctx context.Context, spec SubAgentSpec) (SubAgent, error) {
		execFn := func(ctx context.Context, description string, parentContext map[string]any) (string, error) {
			return "Test result: " + description, nil
		}
		return NewSimpleSubAgent(spec.Name, spec.Prompt, execFn), nil
	}

	// 3. 创建 SubAgentMiddleware
	middleware, err := NewSubAgentMiddleware(&SubAgentMiddlewareConfig{
		Specs:   specs,
		Factory: factory,
	})
	if err != nil {
		t.Fatalf("Failed to create SubAgentMiddleware: %v", err)
	}

	// 4. 测试工具
	tools := middleware.Tools()
	if len(tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(tools))
	}

	taskTool := tools[0]
	if taskTool.Name() != "task" {
		t.Errorf("Expected tool name 'task', got %s", taskTool.Name())
	}

	// 5. 测试 task 执行
	result, err := taskTool.Execute(ctx, map[string]any{
		"description":   "Test task",
		"subagent_type": "test-agent",
	}, nil)
	if err != nil {
		t.Fatalf("Task execution failed: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatal("Result is not a map")
	}

	if !resultMap["ok"].(bool) {
		t.Errorf("Task should succeed, got ok=false")
	}

	if !strings.Contains(resultMap["result"].(string), "Test task") {
		t.Errorf("Result should contain 'Test task', got: %s", resultMap["result"])
	}

	// 6. 清理
	if err := middleware.OnAgentStop(ctx, "test"); err != nil {
		t.Errorf("OnAgentStop failed: %v", err)
	}
}

// TestFullStack_Integration 集成测试: 完整 Middleware Stack
func TestFullStack_Integration(t *testing.T) {
	ctx := context.Background()

	// 1. 创建 Backend
	backend := backends.NewStateBackend()

	// 2. 创建 FilesystemMiddleware
	fsMiddleware := NewFilesystemMiddleware(&FilesystemMiddlewareConfig{
		Backend: backend,
	})

	// 3. 创建 SubAgentMiddleware
	subagentMiddleware, err := NewSubAgentMiddleware(&SubAgentMiddlewareConfig{
		Specs: []SubAgentSpec{
			{Name: "researcher", Description: "Research agent"},
			{Name: "coder", Description: "Coding agent"},
		},
		Factory: func(ctx context.Context, spec SubAgentSpec) (SubAgent, error) {
			return NewSimpleSubAgent(spec.Name, spec.Prompt, nil), nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to create SubAgentMiddleware: %v", err)
	}

	// 4. 创建 Stack
	stack := NewStack([]Middleware{
		fsMiddleware,
		subagentMiddleware,
	})

	// 5. 验证工具总数 (6 个文件工具 + 1 个 task 工具)
	tools := stack.Tools()
	if len(tools) != 7 {
		t.Errorf("Expected 7 tools, got %d", len(tools))
	}

	// 6. 验证中间件顺序(按优先级)
	middlewares := stack.Middlewares()
	if len(middlewares) != 2 {
		t.Fatalf("Expected 2 middlewares, got %d", len(middlewares))
	}

	// FilesystemMiddleware (priority: 100) 应该在前
	if middlewares[0].Name() != "filesystem" {
		t.Errorf("First middleware should be 'filesystem', got '%s'", middlewares[0].Name())
	}

	// SubAgentMiddleware (priority: 200) 应该在后
	if middlewares[1].Name() != "subagent" {
		t.Errorf("Second middleware should be 'subagent', got '%s'", middlewares[1].Name())
	}

	// 7. 清理
	if err := stack.OnAgentStop(ctx, "test"); err != nil {
		t.Errorf("Stack cleanup failed: %v", err)
	}
}

// TestBackendIntegration 集成测试: Backend 与 Middleware
func TestBackendIntegration(t *testing.T) {
	ctx := context.Background()

	// 测试不同的 Backend
	backends := map[string]backends.BackendProtocol{
		"state": backends.NewStateBackend(),
	}

	for name, backend := range backends {
		t.Run(name, func(t *testing.T) {
			// 创建 Middleware
			middleware := NewFilesystemMiddleware(&FilesystemMiddlewareConfig{
				Backend: backend,
			})

			// 写入测试数据
			_, err := backend.Write(ctx, "/test.txt", "line1\nline2\nline3")
			if err != nil {
				t.Fatalf("Write failed: %v", err)
			}

			// 读取数据
			content, err := backend.Read(ctx, "/test.txt", 0, 0)
			if err != nil {
				t.Fatalf("Read failed: %v", err)
			}

			if !strings.Contains(content, "line1") {
				t.Errorf("Content should contain 'line1', got: %s", content)
			}

			// 测试编辑
			result, err := backend.Edit(ctx, "/test.txt", "line1", "LINE1", false)
			if err != nil {
				t.Fatalf("Edit failed: %v", err)
			}

			if result.ReplacementsMade != 1 {
				t.Errorf("Expected 1 replacement, got %d", result.ReplacementsMade)
			}

			// 验证编辑结果
			newContent, _ := backend.Read(ctx, "/test.txt", 0, 0)
			if !strings.Contains(newContent, "LINE1") {
				t.Errorf("Content should contain 'LINE1' after edit, got: %s", newContent)
			}

			// 测试 Middleware 工具
			tools := middleware.Tools()
			if len(tools) != 6 {
				t.Errorf("Expected 6 tools, got %d", len(tools))
			}
		})
	}
}

// BenchmarkMiddlewareStack 性能测试: Middleware Stack
func BenchmarkMiddlewareStack(b *testing.B) {
	backend := backends.NewStateBackend()
	middleware := NewFilesystemMiddleware(&FilesystemMiddlewareConfig{
		Backend: backend,
	})
	stack := NewStack([]Middleware{middleware})

	for b.Loop() {
		_ = stack.Tools()
	}
}

// BenchmarkBackendWrite 性能测试: Backend Write
func BenchmarkBackendWrite(b *testing.B) {
	backend := backends.NewStateBackend()
	ctx := context.Background()

	for b.Loop() {
		_, _ = backend.Write(ctx, "/bench.txt", "test content")
	}
}
