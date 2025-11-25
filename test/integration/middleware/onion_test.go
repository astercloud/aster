package middleware

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/middleware"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/types"
	"github.com/astercloud/aster/test/integration"
)

// TestMiddlewareOnionModel 验证中间件洋葱模型
func TestMiddlewareOnionModel(t *testing.T) {
	// 准备模拟响应
	mockResponses := []string{
		"处理完成",
		"中间件处理后的响应",
	}

	deps, mockProvider := integration.SetupIntegrationDeps(t, mockResponses)

	// 创建中间件栈
	middlewareStack := middleware.NewStack()

	// 添加测试中间件（按优先级排序）
	executionOrder := make([]string, 0)

	// 高优先级中间件（最外层）
	middlewareStack.Add(&TestMiddleware{
		Name:    "outer",
		Priority: 100,
		Order:   &executionOrder,
	})

	// 中等优先级中间件
	middlewareStack.Add(&TestMiddleware{
		Name:    "middle",
		Priority: 50,
		Order:   &executionOrder,
	})

	// 低优先级中间件（最内层）
	middlewareStack.Add(&TestMiddleware{
		Name:    "inner",
		Priority: 10,
		Order:   &executionOrder,
	})

	// 设置中间件栈到依赖项
	deps.MiddlewareStack = middlewareStack

	// 创建Agent配置
	config := &types.AgentConfig{
		TemplateID: "middleware-test",
		ModelConfig: &types.ModelConfig{
			Provider: "mock",
			Model:    "test-model",
			APIKey:   "test-key",
		},
		Sandbox: &types.SandboxConfig{
			Kind:    types.SandboxKindMock,
			WorkDir: "/tmp/test",
		},
		Middleware: &types.MiddlewareConfig{
			Enabled: true,
			Layers: []types.MiddlewareLayer{
				{Name: "outer", Priority: 100},
				{Name: "middle", Priority: 50},
				{Name: "inner", Priority: 10},
			},
		},
	}

	// 注册测试模板
	deps.TemplateRegistry.Register(&types.AgentTemplateDefinition{
		ID:           "middleware-test",
		SystemPrompt: "You are a helpful assistant with middleware support.",
		Model:        "test-model",
		Tools:        []interface{}{"Read", "Write"},
	})

	// 创建Agent
	ag, err := agent.Create(context.Background(), config, deps)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	defer func() { _ = ag.Close() }()

	// 测试用例1: 中间件执行顺序验证
	t.Run("Middleware Execution Order", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 重置执行顺序记录
		executionOrder = make([]string, 0)

		// 发送消息
		_, err := ag.Stream(ctx, []byte("测试中间件执行顺序"))
		if err != nil {
			t.Fatalf("Agent stream failed: %v", err)
		}

		// 等待中间件执行完成
		time.Sleep(100 * time.Millisecond)

		// 验证执行顺序
		expectedOrder := []string{"outer_pre", "middle_pre", "inner_pre", "provider", "inner_post", "middle_post", "outer_post"}

		if len(executionOrder) != len(expectedOrder) {
			t.Errorf("Expected %d middleware calls, got %d", len(expectedOrder), len(executionOrder))
		}

		for i, expected := range expectedOrder {
			if i < len(executionOrder) && executionOrder[i] != expected {
				t.Errorf("Expected order %s at position %d, got %s", expected, i, executionOrder[i])
			}
		}

		t.Logf("Middleware execution order verified: %v", executionOrder)
	})

	// 测试用例2: 中间件错误处理
	t.Run("Middleware Error Handling", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 创建会返回错误的中间件
		errorMiddleware := &ErrorTestMiddleware{
			Name:    "error-middleware",
			Priority: 200,
		}

		// 临时添加错误中间件
		middlewareStack.Add(errorMiddleware)

		// 发送消息
		_, err := ag.Stream(ctx, []byte("测试中间件错误处理"))

		// 验证错误被正确处理
		if err == nil {
			t.Error("Expected error from middleware, but got none")
		}

		// 移除错误中间件
		middlewareStack.Remove(errorMiddleware)

		t.Log("Middleware error handling test passed")
	})

	// 测试用例3: 中间件上下文传递
	t.Run("Middleware Context Passing", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 创建上下文传递中间件
		contextMiddleware := &ContextTestMiddleware{
			Name:    "context-middleware",
			Priority: 75,
		}

		middlewareStack.Add(contextMiddleware)

		// 发送消息
		_, err := ag.Stream(ctx, []byte("测试中间件上下文传递"))
		if err != nil {
			t.Fatalf("Agent stream failed: %v", err)
		}

		// 验证上下文值被正确设置和传递
		if !contextMiddleware.ContextSet {
			t.Error("Expected middleware to set context value")
		}

		// 移除测试中间件
		middlewareStack.Remove(contextMiddleware)

		t.Log("Middleware context passing test passed")
	})

	// 测试用例4: 条件中间件执行
	t.Run("Conditional Middleware Execution", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 创建条件中间件
		conditionalMiddleware := &ConditionalTestMiddleware{
			Name:        "conditional-middleware",
			Priority:    60,
			ShouldRun:   true,
			Executed:    false,
		}

		middlewareStack.Add(conditionalMiddleware)

		// 发送消息
		_, err := ag.Stream(ctx, []byte("测试条件中间件执行"))
		if err != nil {
			t.Fatalf("Agent stream failed: %v", err)
		}

		// 验证条件中间件是否执行
		if !conditionalMiddleware.Executed {
			t.Error("Expected conditional middleware to execute")
		}

		// 测试不执行的情况
		conditionalMiddleware.ShouldRun = false
		conditionalMiddleware.Executed = false

		_, err = ag.Stream(ctx, []byte("测试条件中间件不执行"))
		if err != nil {
			t.Fatalf("Agent stream failed: %v", err)
		}

		if conditionalMiddleware.Executed {
			t.Error("Expected conditional middleware to not execute")
		}

		// 移除测试中间件
		middlewareStack.Remove(conditionalMiddleware)

		t.Log("Conditional middleware execution test passed")
	})

	t.Log("Middleware onion model integration test completed successfully")
}

// TestMiddlewarePerformance 验证中间件性能影响
func TestMiddlewarePerformance(t *testing.T) {
	mockResponses := []string{"性能测试响应"}
	deps, _ := integration.SetupIntegrationDeps(t, mockResponses)

	// 创建多个中间件来测试性能
	middlewareStack := middleware.NewStack()

	for i := 0; i < 10; i++ {
		middlewareStack.Add(&PerformanceTestMiddleware{
			Name:    fmt.Sprintf("perf-middleware-%d", i),
			Priority: 100 - i*10,
			Delay:   time.Millisecond,
		})
	}

	deps.MiddlewareStack = middlewareStack

	config := &types.AgentConfig{
		TemplateID: "performance-test",
		ModelConfig: &types.ModelConfig{
			Provider: "mock",
			Model:    "test-model",
			APIKey:   "test-key",
		},
		Sandbox: &types.SandboxConfig{
			Kind:    types.SandboxKindMock,
			WorkDir: "/tmp/test",
		},
	}

	deps.TemplateRegistry.Register(&types.AgentTemplateDefinition{
		ID:           "performance-test",
		SystemPrompt: "Performance test assistant.",
		Model:        "test-model",
		Tools:        []interface{}{"Read"},
	})

	ag, err := agent.Create(context.Background(), config, deps)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	defer func() { _ = ag.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 测试多次请求的平均性能
 iterations := 5
	totalDuration := time.Duration(0)

	for i := 0; i < iterations; i++ {
		start := time.Now()
		_, err := ag.Stream(ctx, []byte(fmt.Sprintf("性能测试 %d", i)))
		if err != nil {
			t.Fatalf("Agent stream %d failed: %v", i, err)
		}
		duration := time.Since(start)
		totalDuration += duration
	}

	averageDuration := totalDuration / time.Duration(iterations)
	t.Logf("Average request duration with 10 middlewares: %v", averageDuration)

	// 验证性能在合理范围内（这里设置为5秒，实际应根据需要调整）
	if averageDuration > 5*time.Second {
		t.Errorf("Performance degradation detected: average duration %v exceeds threshold", averageDuration)
	}
}

// TestMiddleware 实现测试中间件
type TestMiddleware struct {
	Name      string
	Priority  int
	Order     *[]string
	PreCalled bool
	PostCalled bool
}

func (m *TestMiddleware) Name() string {
	return m.Name
}

func (m *TestMiddleware) Priority() int {
	return m.Priority
}

func (m *TestMiddleware) PreProcess(ctx context.Context, req *provider.LLMRequest) error {
	*m.Order = append(*m.Order, m.Name+"_pre")
	m.PreCalled = true
	return nil
}

func (m *TestMiddleware) PostProcess(ctx context.Context, resp *provider.LLMResponse) error {
	*m.Order = append(*m.Order, m.Name+"_post")
	m.PostCalled = true
	return nil
}

// ErrorTestMiddleware 实现错误测试中间件
type ErrorTestMiddleware struct {
	Name     string
	Priority int
}

func (m *ErrorTestMiddleware) Name() string {
	return m.Name
}

func (m *ErrorTestMiddleware) Priority() int {
	return m.Priority
}

func (m *ErrorTestMiddleware) PreProcess(ctx context.Context, req *provider.LLMRequest) error {
	return fmt.Errorf("intentional middleware error")
}

func (m *ErrorTestMiddleware) PostProcess(ctx context.Context, resp *provider.LLMResponse) error {
	return nil
}

// ContextTestMiddleware 实现上下文测试中间件
type ContextTestMiddleware struct {
	Name       string
	Priority   int
	ContextSet bool
}

func (m *ContextTestMiddleware) Name() string {
	return m.Name
}

func (m *ContextTestMiddleware) Priority() int {
	return m.Priority
}

func (m *ContextTestMiddleware) PreProcess(ctx context.Context, req *provider.LLMRequest) error {
	ctx = context.WithValue(ctx, "test_key", "test_value")
	m.ContextSet = true
	return nil
}

func (m *ContextTestMiddleware) PostProcess(ctx context.Context, resp *provider.LLMResponse) error {
	// 验证上下文值仍然存在
	if ctx.Value("test_key") != "test_value" {
		return fmt.Errorf("context value not preserved")
	}
	return nil
}

// ConditionalTestMiddleware 实现条件测试中间件
type ConditionalTestMiddleware struct {
	Name      string
	Priority  int
	ShouldRun bool
	Executed  bool
}

func (m *ConditionalTestMiddleware) Name() string {
	return m.Name
}

func (m *ConditionalTestMiddleware) Priority() int {
	return m.Priority
}

func (m *ConditionalTestMiddleware) PreProcess(ctx context.Context, req *provider.LLMRequest) error {
	if m.ShouldRun {
		m.Executed = true
	}
	return nil
}

func (m *ConditionalTestMiddleware) PostProcess(ctx context.Context, resp *provider.LLMResponse) error {
	return nil
}

// PerformanceTestMiddleware 实现性能测试中间件
type PerformanceTestMiddleware struct {
	Name     string
	Priority int
	Delay    time.Duration
}

func (m *PerformanceTestMiddleware) Name() string {
	return m.Name
}

func (m *PerformanceTestMiddleware) Priority() int {
	return m.Priority
}

func (m *PerformanceTestMiddleware) PreProcess(ctx context.Context, req *provider.LLMRequest) error {
	time.Sleep(m.Delay)
	return nil
}

func (m *PerformanceTestMiddleware) PostProcess(ctx context.Context, resp *provider.LLMResponse) error {
	time.Sleep(m.Delay)
	return nil
}