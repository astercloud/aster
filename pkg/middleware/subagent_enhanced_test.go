package middleware

import (
	"context"
	"slices"
	"testing"
)

// TestSubAgentMiddleware_GeneralPurpose 测试通用子代理
func TestSubAgentMiddleware_GeneralPurpose(t *testing.T) {
	// 创建工厂
	factory := func(ctx context.Context, spec SubAgentSpec) (SubAgent, error) {
		execFn := func(ctx context.Context, description string, parentContext map[string]any) (string, error) {
			return "Result from " + spec.Name, nil
		}
		return NewSimpleSubAgent(spec.Name, spec.Prompt, execFn), nil
	}

	// 测试1: 默认启用 general-purpose
	t.Run("Default EnableGeneralPurpose", func(t *testing.T) {
		middleware, err := NewSubAgentMiddleware(&SubAgentMiddlewareConfig{
			Specs:                []SubAgentSpec{}, // 空规格
			Factory:              factory,
			EnableGeneralPurpose: true,
		})
		if err != nil {
			t.Fatalf("Failed to create middleware: %v", err)
		}

		agents := middleware.ListSubAgents()
		if len(agents) != 1 {
			t.Fatalf("Expected 1 agent (general-purpose), got %d", len(agents))
		}

		if agents[0] != "general-purpose" {
			t.Errorf("Expected 'general-purpose' agent, got '%s'", agents[0])
		}
	})

	// 测试2: general-purpose + 自定义子代理
	t.Run("GeneralPurpose with custom agents", func(t *testing.T) {
		middleware, err := NewSubAgentMiddleware(&SubAgentMiddlewareConfig{
			Specs: []SubAgentSpec{
				{Name: "researcher", Description: "Research expert"},
				{Name: "coder", Description: "Coding expert"},
			},
			Factory:              factory,
			EnableGeneralPurpose: true,
		})
		if err != nil {
			t.Fatalf("Failed to create middleware: %v", err)
		}

		agents := middleware.ListSubAgents()
		if len(agents) != 3 {
			t.Fatalf("Expected 3 agents, got %d", len(agents))
		}

		// 验证 general-purpose 存在
		found := slices.Contains(agents, "general-purpose")
		if !found {
			t.Error("Expected 'general-purpose' agent to be present")
		}
	})

	// 测试3: 禁用 general-purpose
	t.Run("Disable GeneralPurpose", func(t *testing.T) {
		middleware, err := NewSubAgentMiddleware(&SubAgentMiddlewareConfig{
			Specs: []SubAgentSpec{
				{Name: "specialist", Description: "Specialist only"},
			},
			Factory:              factory,
			EnableGeneralPurpose: false,
			EnableParallel:       true, // 确保不会自动启用
		})
		if err != nil {
			t.Fatalf("Failed to create middleware: %v", err)
		}

		agents := middleware.ListSubAgents()
		if len(agents) != 1 {
			t.Fatalf("Expected 1 agent, got %d", len(agents))
		}

		if agents[0] == "general-purpose" {
			t.Error("general-purpose should not be present when disabled")
		}
	})
}

// TestBuildSubAgentMiddlewareStack 测试中间件栈构建
func TestBuildSubAgentMiddlewareStack(t *testing.T) {
	// 创建父代理中间件
	parentMiddlewares := []Middleware{
		NewBaseMiddleware("parent1", 100),
		NewBaseMiddleware("parent2", 200),
	}

	// 测试1: 不继承父代理中间件
	t.Run("No inheritance", func(t *testing.T) {
		spec := SubAgentSpec{
			Name:               "test",
			InheritMiddlewares: false,
			MiddlewareOverrides: []Middleware{
				NewBaseMiddleware("custom1", 300),
			},
		}

		stack := BuildSubAgentMiddlewareStack(spec, parentMiddlewares)

		if len(stack) != 1 {
			t.Errorf("Expected 1 middleware, got %d", len(stack))
		}

		if len(stack) > 0 && stack[0].Name() != "custom1" {
			t.Errorf("Expected 'custom1', got '%s'", stack[0].Name())
		}
	})

	// 测试2: 继承父代理中间件
	t.Run("Inherit parent middlewares", func(t *testing.T) {
		spec := SubAgentSpec{
			Name:               "test",
			InheritMiddlewares: true,
		}

		stack := BuildSubAgentMiddlewareStack(spec, parentMiddlewares)

		if len(stack) != 2 {
			t.Errorf("Expected 2 middlewares (inherited), got %d", len(stack))
		}

		if len(stack) >= 2 {
			if stack[0].Name() != "parent1" {
				t.Errorf("Expected 'parent1', got '%s'", stack[0].Name())
			}
			if stack[1].Name() != "parent2" {
				t.Errorf("Expected 'parent2', got '%s'", stack[1].Name())
			}
		}
	})

	// 测试3: 继承 + 追加新中间件
	t.Run("Inherit and append", func(t *testing.T) {
		spec := SubAgentSpec{
			Name:               "test",
			InheritMiddlewares: true,
			MiddlewareOverrides: []Middleware{
				NewBaseMiddleware("custom1", 300),
			},
		}

		stack := BuildSubAgentMiddlewareStack(spec, parentMiddlewares)

		if len(stack) != 3 {
			t.Errorf("Expected 3 middlewares (2 inherited + 1 appended), got %d", len(stack))
		}

		if len(stack) >= 3 {
			if stack[0].Name() != "parent1" {
				t.Errorf("Expected 'parent1', got '%s'", stack[0].Name())
			}
			if stack[1].Name() != "parent2" {
				t.Errorf("Expected 'parent2', got '%s'", stack[1].Name())
			}
			if stack[2].Name() != "custom1" {
				t.Errorf("Expected 'custom1', got '%s'", stack[2].Name())
			}
		}
	})

	// 测试4: 继承 + 覆盖中间件
	t.Run("Inherit and override", func(t *testing.T) {
		spec := SubAgentSpec{
			Name:               "test",
			InheritMiddlewares: true,
			MiddlewareOverrides: []Middleware{
				NewBaseMiddleware("parent2", 250), // 覆盖 parent2
			},
		}

		stack := BuildSubAgentMiddlewareStack(spec, parentMiddlewares)

		if len(stack) != 2 {
			t.Errorf("Expected 2 middlewares (1 inherited + 1 overridden), got %d", len(stack))
		}

		if len(stack) >= 2 {
			if stack[0].Name() != "parent1" {
				t.Errorf("Expected 'parent1', got '%s'", stack[0].Name())
			}
			if stack[1].Name() != "parent2" {
				t.Errorf("Expected 'parent2' (overridden), got '%s'", stack[1].Name())
			}
			// 验证优先级已更新
			if stack[1].Priority() != 250 {
				t.Errorf("Expected priority 250, got %d", stack[1].Priority())
			}
		}
	})

	// 测试5: 继承 + 混合(覆盖 + 追加)
	t.Run("Inherit, override and append", func(t *testing.T) {
		spec := SubAgentSpec{
			Name:               "test",
			InheritMiddlewares: true,
			MiddlewareOverrides: []Middleware{
				NewBaseMiddleware("parent1", 150), // 覆盖 parent1
				NewBaseMiddleware("custom1", 300), // 追加新的
			},
		}

		stack := BuildSubAgentMiddlewareStack(spec, parentMiddlewares)

		if len(stack) != 3 {
			t.Errorf("Expected 3 middlewares, got %d", len(stack))
		}

		if len(stack) >= 3 {
			// 验证名称
			if stack[0].Name() != "parent1" {
				t.Errorf("Expected 'parent1' (overridden), got '%s'", stack[0].Name())
			}
			if stack[1].Name() != "parent2" {
				t.Errorf("Expected 'parent2' (inherited), got '%s'", stack[1].Name())
			}
			if stack[2].Name() != "custom1" {
				t.Errorf("Expected 'custom1' (appended), got '%s'", stack[2].Name())
			}

			// 验证 parent1 的优先级已被覆盖
			if stack[0].Priority() != 150 {
				t.Errorf("Expected priority 150, got %d", stack[0].Priority())
			}
		}
	})
}

// TestSubAgentSpec_InheritMiddlewares 测试 InheritMiddlewares 标志
func TestSubAgentSpec_InheritMiddlewares(t *testing.T) {
	factory := func(ctx context.Context, spec SubAgentSpec) (SubAgent, error) {
		// 验证 spec.InheritMiddlewares 标志
		if spec.Name == "general-purpose" && !spec.InheritMiddlewares {
			t.Errorf("general-purpose should have InheritMiddlewares=true")
		}
		return NewSimpleSubAgent(spec.Name, spec.Prompt, nil), nil
	}

	// 创建中间件
	middleware, err := NewSubAgentMiddleware(&SubAgentMiddlewareConfig{
		Specs: []SubAgentSpec{
			{
				Name:               "no-inherit",
				Description:        "Does not inherit",
				InheritMiddlewares: false,
			},
			{
				Name:               "with-inherit",
				Description:        "Inherits middlewares",
				InheritMiddlewares: true,
			},
		},
		Factory:              factory,
		EnableGeneralPurpose: true,
	})

	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// 验证所有子代理都已创建
	agents := middleware.ListSubAgents()
	if len(agents) != 3 {
		t.Fatalf("Expected 3 agents, got %d", len(agents))
	}
}

// TestGetMiddlewareForSubAgent 测试获取子代理中间件栈
func TestGetMiddlewareForSubAgent(t *testing.T) {
	factory := func(ctx context.Context, spec SubAgentSpec) (SubAgent, error) {
		return NewSimpleSubAgent(spec.Name, spec.Prompt, nil), nil
	}

	middleware, err := NewSubAgentMiddleware(&SubAgentMiddlewareConfig{
		Specs:                []SubAgentSpec{},
		Factory:              factory,
		EnableGeneralPurpose: true,
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	parentMiddlewares := []Middleware{
		NewBaseMiddleware("parent1", 100),
		NewBaseMiddleware("parent2", 200),
	}

	// 测试获取中间件栈
	spec := SubAgentSpec{
		Name:               "test",
		InheritMiddlewares: true,
		MiddlewareOverrides: []Middleware{
			NewBaseMiddleware("custom", 300),
		},
	}

	stack := middleware.GetMiddlewareForSubAgent(spec, parentMiddlewares)

	if len(stack) != 3 {
		t.Errorf("Expected 3 middlewares, got %d", len(stack))
	}
}
