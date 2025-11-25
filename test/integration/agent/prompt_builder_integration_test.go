package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/types"
	"github.com/astercloud/aster/test/integration"
)

// TestPromptBuilderIntegration 验证Prompt Builder的集成功能
func TestPromptBuilderIntegration(t *testing.T) {
	// 准备模拟响应
	mockResponses := []string{
		"我将使用TodoWrite来管理任务",
		"让我读取这个文件",
		"我需要搜索相关信息",
	}

	deps, mockProvider := integration.SetupIntegrationDeps(t, mockResponses)

	// 测试用例1: 基础Prompt构建验证
	t.Run("Basic Prompt Building", func(t *testing.T) {
		// 注册基础模板
		deps.TemplateRegistry.Register(&types.AgentTemplateDefinition{
			ID:           "basic-prompt-test",
			SystemPrompt: "You are a helpful assistant.",
			Model:        "test-model",
			Tools:        []interface{}{"Read", "Write"},
		})

		config := &types.AgentConfig{
			TemplateID: "basic-prompt-test",
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

		ag, err := agent.Create(context.Background(), config, deps)
		if err != nil {
			t.Fatalf("Failed to create agent: %v", err)
		}
		defer func() { _ = ag.Close() }()

		// 获取System Prompt并验证
		systemPrompt := ag.GetSystemPrompt()

		// 验证基础Prompt存在
		if !strings.Contains(systemPrompt, "You are a helpful assistant") {
			t.Error("System prompt should contain base prompt")
		}

		// 验证环境信息存在
		if !strings.Contains(systemPrompt, "## Environment Information") {
			t.Error("System prompt should contain environment information")
		}

		// 验证工具手册存在
		if !strings.Contains(systemPrompt, "## Tools Manual") {
			t.Error("System prompt should contain tools manual")
		}

		t.Logf("Basic prompt building test passed. Prompt length: %d", len(systemPrompt))
	})

	// 测试用例2: 模块优先级排序验证
	t.Run("Module Priority Ordering", func(t *testing.T) {
		// 注册带不同模块的模板
		deps.TemplateRegistry.Register(&types.AgentTemplateDefinition{
			ID:           "priority-test",
			SystemPrompt: "You are a test assistant.",
			Model:        "test-model",
			Tools:        []interface{}{"Read", "Write", "TodoWrite", "WebSearch"},
			Runtime: &types.AgentTemplateRuntime{
				Todo: &types.TodoConfig{
					Enabled:         true,
					ReminderOnStart: true,
				},
				CodeReference: &types.CodeReferenceConfig{
					Enabled:       true,
					FileReference: true,
				},
			},
		})

		config := &types.AgentConfig{
			TemplateID: "priority-test",
			ModelConfig: &types.ModelConfig{
				Provider: "mock",
				Model:    "test-model",
				APIKey:   "test-key",
			},
			Sandbox: &types.SandboxConfig{
				Kind:    types.SandboxKindMock,
				WorkDir: "/tmp/test",
			},
			Metadata: map[string]interface{}{
				"agent_type": "code_assistant",
			},
		}

		ag, err := agent.Create(context.Background(), config, deps)
		if err != nil {
			t.Fatalf("Failed to create agent: %v", err)
		}
		defer func() { _ = ag.Close() }()

		systemPrompt := ag.GetSystemPrompt()

		// 验证模块按正确优先级排序
		// BasePrompt (0) → Environment (10) → Sandbox (15) → ToolsManual (20) → TodoReminder (25) → CodeReference (30)
		sections := []string{
			"You are a test assistant",           // BasePrompt
			"## Environment Information",        // Environment
			"## Sandbox Configuration",          // Sandbox
			"## Tools Manual",                    // ToolsManual
			"## Task Management",                 // TodoReminder
			"## Code References",                 // CodeReference
		}

		lastPos := -1
		for _, section := range sections {
			pos := strings.Index(systemPrompt, section)
			if pos == -1 {
				t.Errorf("Expected section '%s' not found in prompt", section)
				continue
			}
			if pos < lastPos {
				t.Errorf("Section '%s' appears before previous section, priority ordering failed", section)
			}
			lastPos = pos
		}

		t.Log("Module priority ordering test passed")
	})

	// 测试用例3: 条件注入验证
	t.Run("Conditional Module Injection", func(t *testing.T) {
		// 测试代码助手模板的条件注入
		deps.TemplateRegistry.Register(&types.AgentTemplateDefinition{
			ID:           "conditional-test",
			SystemPrompt: "You are a professional assistant.",
			Model:        "test-model",
			Tools:        []interface{}{"Read", "Write"},
			Runtime: &types.AgentTemplateRuntime{
				Todo: &types.TodoConfig{
					Enabled:         true,
					ReminderOnStart: true,
				},
				CodeReference: &types.CodeReferenceConfig{
					Enabled:       true,
					FileReference: true,
				},
				ToolsManual: &types.ToolsManualConfig{
					Mode:    "listed",
					Include: []string{"Read"}, // 只包含Read工具
				},
			},
		})

		// 测试无代码助手类型
		config1 := &types.AgentConfig{
			TemplateID: "conditional-test",
			ModelConfig: &types.ModelConfig{
				Provider: "mock",
				Model:    "test-model",
				APIKey:   "test-key",
			},
			Sandbox: &types.SandboxConfig{
				Kind:    types.SandboxKindMock,
				WorkDir: "/tmp/test",
			},
			Metadata: map[string]interface{}{
				"agent_type": "general",
			},
		}

		ag1, err := agent.Create(context.Background(), config1, deps)
		if err != nil {
			t.Fatalf("Failed to create agent 1: %v", err)
		}
		defer func() { _ = ag1.Close() }()

		prompt1 := ag1.GetSystemPrompt()

		// 验证无代码引用规范
		if strings.Contains(prompt1, "## Code References") {
			t.Error("General assistant should not have code reference guidelines")
		}

		// 测试代码助手类型
		config2 := &types.AgentConfig{
			TemplateID: "conditional-test",
			ModelConfig: &types.ModelConfig{
				Provider: "mock",
				Model:    "test-model",
				APIKey:   "test-key",
			},
			Sandbox: &types.SandboxConfig{
				Kind:    types.SandboxKindMock,
				WorkDir: "/tmp/test",
			},
			Metadata: map[string]interface{}{
				"agent_type": "code_assistant",
			},
		}

		ag2, err := agent.Create(context.Background(), config2, deps)
		if err != nil {
			t.Fatalf("Failed to create agent 2: %v", err)
		}
		defer func() { _ = ag2.Close() }()

		prompt2 := ag2.GetSystemPrompt()

		// 验证有代码引用规范
		if !strings.Contains(prompt2, "## Code References") {
			t.Error("Code assistant should have code reference guidelines")
		}

		if !strings.Contains(prompt2, "file_path:line_number") {
			t.Error("Code assistant should mention file_path:line_number format")
		}

		t.Log("Conditional module injection test passed")
	})

	// 测试用例4: 工具手册配置验证
	t.Run("Tools Manual Configuration", func(t *testing.T) {
		// 注册带选择性工具的模板
		deps.TemplateRegistry.Register(&types.AgentTemplateDefinition{
			ID:           "tools-manual-test",
			SystemPrompt: "You are a selective assistant.",
			Model:        "test-model",
			Tools:        []interface{}{"Read", "Write", "WebSearch"},
			Runtime: &types.AgentTemplateRuntime{
				ToolsManual: &types.ToolsManualConfig{
					Mode:    "listed",
					Include: []string{"Read", "WebSearch"}, // 只包含部分工具
				},
			},
		})

		config := &types.AgentConfig{
			TemplateID: "tools-manual-test",
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

		ag, err := agent.Create(context.Background(), config, deps)
		if err != nil {
			t.Fatalf("Failed to create agent: %v", err)
		}
		defer func() { _ = ag.Close() }()

		systemPrompt := ag.GetSystemPrompt()

		// 验证只包含指定的工具
		if !strings.Contains(systemPrompt, "`Read`") {
			t.Error("System prompt should contain Read tool")
		}

		if !strings.Contains(systemPrompt, "`WebSearch`") {
			t.Error("System prompt should contain WebSearch tool")
		}

		// 验证不包含未列出的工具
		if strings.Contains(systemPrompt, "`Write`") {
			t.Error("System prompt should not contain Write tool (excluded by config)")
		}

		t.Log("Tools manual configuration test passed")
	})

	// 测试用例5: Todo提醒功能验证
	t.Run("Todo Reminder Functionality", func(t *testing.T) {
		// 注册带Todo提醒的模板
		deps.TemplateRegistry.Register(&types.AgentTemplateDefinition{
			ID:           "todo-reminder-test",
			SystemPrompt: "You are a task management assistant.",
			Model:        "test-model",
			Tools:        []interface{}{"Read", "Write", "TodoWrite"},
			Runtime: &types.AgentTemplateRuntime{
				Todo: &types.TodoConfig{
					Enabled:         true,
					ReminderOnStart: true,
				},
			},
		})

		config := &types.AgentConfig{
			TemplateID: "todo-reminder-test",
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

		ag, err := agent.Create(context.Background(), config, deps)
		if err != nil {
			t.Fatalf("Failed to create agent: %v", err)
		}
		defer func() { _ = ag.Close() }()

		systemPrompt := ag.GetSystemPrompt()

		// 验证Todo提醒存在
		if !strings.Contains(systemPrompt, "## Task Management") {
			t.Error("Todo assistant should have task management section")
		}

		if !strings.Contains(systemPrompt, "TodoWrite") {
			t.Error("Todo assistant should mention TodoWrite tool")
		}

		t.Log("Todo reminder functionality test passed")
	})

	// 测试用例6: 完整集成场景验证
	t.Run("Complete Integration Scenario", func(t *testing.T) {
		// 注册复杂的完整模板
		deps.TemplateRegistry.Register(&types.AgentTemplateDefinition{
			ID:           "complete-integration-test",
			SystemPrompt: "You are a comprehensive AI assistant with advanced capabilities.",
			Model:        "test-model",
			Tools:        []interface{}{"Read", "Write", "TodoWrite", "WebSearch", "HttpRequest"},
			Runtime: &types.AgentTemplateRuntime{
				Environment: &types.EnvironmentConfig{
					Enabled:      true,
					SystemInfo:   true,
					WorkingDir:   true,
					Environment:  true,
					Network:      true,
					Limits:       true,
				},
				Sandbox: &types.SandboxConfig{
					Enabled:   true,
					WorkDir:   "/tmp/test",
					Resources: &types.ResourceLimits{},
				},
				Todo: &types.TodoConfig{
					Enabled:         true,
					ReminderOnStart: true,
				},
				CodeReference: &types.CodeReferenceConfig{
					Enabled:          true,
					FileReference:    true,
					FunctionReference: true,
					ClassReference:   true,
					VariableReference: true,
				},
				ToolsManual: &types.ToolsManualConfig{
					Mode:          "custom",
					Include:       []string{"Read", "Write", "TodoWrite"},
					Exclude:       []string{},
					UsageExamples: true,
				},
			},
		})

		config := &types.AgentConfig{
			TemplateID: "complete-integration-test",
			ModelConfig: &types.ModelConfig{
				Provider: "mock",
				Model:    "test-model",
				APIKey:   "test-key",
			},
			Sandbox: &types.SandboxConfig{
				Kind:    types.SandboxKindMock,
				WorkDir: "/tmp/test",
			},
			Metadata: map[string]interface{}{
				"agent_type": "code_assistant",
			},
		}

		ag, err := agent.Create(context.Background(), config, deps)
		if err != nil {
			t.Fatalf("Failed to create agent: %v", err)
		}
		defer func() { _ = ag.Close() }()

		systemPrompt := ag.GetSystemPrompt()

		// 验证所有模块都存在并按正确顺序
		expectedSections := []string{
			"You are a comprehensive AI assistant", // BasePrompt
			"## Environment Information",            // Environment
			"## Sandbox Configuration",             // Sandbox
			"## Tools Manual",                      // ToolsManual
			"## Task Management",                   // TodoReminder
			"## Code References",                   // CodeReference
		}

		for _, section := range expectedSections {
			if !strings.Contains(systemPrompt, section) {
				t.Errorf("Missing expected section: %s", section)
			}
		}

		// 验证工具列表正确性
		expectedTools := []string{"`Read`", "`Write`", "`TodoWrite`"}
		for _, tool := range expectedTools {
			if !strings.Contains(systemPrompt, tool) {
				t.Errorf("Missing expected tool in manual: %s", tool)
			}
		}

		// 验证代码引用格式
		codeRefFormats := []string{"file_path:line_number", "function()"}
		for _, format := range codeRefFormats {
			if !strings.Contains(systemPrompt, format) {
				t.Errorf("Missing expected code reference format: %s", format)
			}
		}

		t.Logf("Complete integration scenario test passed. Prompt length: %d", len(systemPrompt))
	})

	t.Log("Prompt Builder integration test completed successfully")
}

// TestPromptBuilderPerformance 验证Prompt Builder性能
func TestPromptBuilderPerformance(t *testing.T) {
	mockResponses := []string{"Performance test response"}
	deps, _ := integration.SetupIntegrationDeps(t, mockResponses)

	// 注册复杂模板
	deps.TemplateRegistry.Register(&types.AgentTemplateDefinition{
		ID:           "performance-test",
		SystemPrompt: "You are a performance test assistant.",
		Model:        "test-model",
		Tools:        []interface{}{"Read", "Write", "TodoWrite", "WebSearch", "HttpRequest", "Bash"},
		Runtime: &types.AgentTemplateRuntime{
			Environment: &types.EnvironmentConfig{
				Enabled:     true,
				SystemInfo:  true,
				WorkingDir:  true,
				Environment: true,
				Network:     true,
				Limits:      true,
			},
			Todo: &types.TodoConfig{
				Enabled:         true,
				ReminderOnStart: true,
			},
			CodeReference: &types.CodeReferenceConfig{
				Enabled:          true,
				FileReference:    true,
				FunctionReference: true,
				ClassReference:   true,
			},
			ToolsManual: &types.ToolsManualConfig{
				Mode:          "custom",
				UsageExamples: true,
			},
		},
	})

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

	// 测试多次Agent创建的性能
	iterations := 10
	totalDuration := time.Duration(0)

	for i := 0; i < iterations; i++ {
		start := time.Now()
		ag, err := agent.Create(context.Background(), config, deps)
		if err != nil {
			t.Fatalf("Failed to create agent %d: %v", i, err)
		}

		// 测试Prompt生成性能
		_ = ag.GetSystemPrompt()
		_ = ag.Close()

		duration := time.Since(start)
		totalDuration += duration
	}

	averageDuration := totalDuration / time.Duration(iterations)
	t.Logf("Average agent creation and prompt generation duration: %v", averageDuration)

	// 验证性能在合理范围内
	if averageDuration > 1*time.Second {
		t.Errorf("Performance degradation detected: average duration %v exceeds threshold", averageDuration)
	}
}