package bridge

import (
	"context"
	"testing"
	"time"

	"github.com/astercloud/aster/pkg/tools"
)

// MockTool 模拟工具用于测试
type MockTool struct{}

func (m *MockTool) Name() string {
	return "MockTool"
}

func (m *MockTool) Description() string {
	return "A mock tool for testing"
}

func (m *MockTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{
				"type":        "string",
				"description": "A test message",
			},
		},
		"required": []string{"message"},
	}
}

func (m *MockTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	message, _ := input["message"].(string)
	return map[string]any{
		"echo": message,
		"time": time.Now().Unix(),
	}, nil
}

func (m *MockTool) Prompt() string {
	return "Mock tool for testing"
}

// NewMockTool 创建 MockTool 实例
func NewMockTool(config map[string]any) (tools.Tool, error) {
	return &MockTool{}, nil
}

// TestPTCIntegration 测试完整的 PTC 流程
func TestPTCIntegration(t *testing.T) {
	// 检查 Python 和 aiohttp 是否可用
	if !checkPythonAiohttp() {
		t.Skip("Skipping test: Python with aiohttp is not available")
	}

	// 1. 创建工具注册表
	registry := tools.NewRegistry()
	registry.Register("MockTool", NewMockTool)

	// 2. 创建 ToolBridge
	toolBridge := NewToolBridge(registry)

	// 3. 创建 HTTP 桥接服务器
	server := NewHTTPBridgeServer(toolBridge, "localhost:18080")

	// 4. 启动服务器
	if err := server.StartAsync(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	// 等待服务器启动
	time.Sleep(200 * time.Millisecond)

	// 5. 创建 PythonRuntime
	runtime := NewPythonRuntime(nil)
	runtime.SetTools([]string{"MockTool"})
	runtime.SetBridgeURL("http://localhost:18080")

	// 6. 测试代码执行 - 调用工具
	pythonCode := `
import asyncio

async def main():
    result = await MockTool(message="Hello from Python!")
    print(result)

asyncio.run(main())
`

	ctx := context.Background()
	result, err := runtime.Execute(ctx, pythonCode, map[string]any{})

	// 7. 验证结果
	if err != nil {
		t.Fatalf("Failed to execute Python code: %v", err)
	}

	if !result.Success {
		t.Errorf("Execution failed: %s", result.Error)
	}

	if result.Stdout == "" {
		t.Error("Expected stdout output, got empty string")
	}

	t.Logf("Test passed! Output: %s", result.Stdout)
}

// TestHTTPBridgeServerEndpoints 测试 HTTP 服务器端点
func TestHTTPBridgeServerEndpoints(t *testing.T) {
	// 创建工具注册表和桥接
	registry := tools.NewRegistry()
	registry.Register("MockTool", NewMockTool)
	toolBridge := NewToolBridge(registry)

	// 创建并启动服务器
	server := NewHTTPBridgeServer(toolBridge, "localhost:18081")
	if err := server.StartAsync(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	// 等待服务器启动
	time.Sleep(200 * time.Millisecond)

	// 测试各个端点
	t.Run("ToolList", func(t *testing.T) {
		tools := toolBridge.ListAvailableTools()
		if len(tools) == 0 {
			t.Error("Expected at least one tool")
		}
		if tools[0] != "MockTool" {
			t.Errorf("Expected MockTool, got %s", tools[0])
		}
	})

	t.Run("ToolSchema", func(t *testing.T) {
		schema, err := toolBridge.GetToolSchema("MockTool")
		if err != nil {
			t.Fatalf("Failed to get schema: %v", err)
		}
		if schema["name"] != "MockTool" {
			t.Error("Schema name mismatch")
		}
	})

	t.Run("ToolCall", func(t *testing.T) {
		ctx := context.Background()
		result, err := toolBridge.CallTool(ctx, "MockTool", map[string]any{
			"message": "Test message",
		}, nil)

		if err != nil {
			t.Fatalf("Failed to call tool: %v", err)
		}

		if !result.Success {
			t.Errorf("Tool call failed: %s", result.Error)
		}

		resultMap, ok := result.Result.(map[string]any)
		if !ok {
			t.Fatal("Expected map result")
		}

		if resultMap["echo"] != "Test message" {
			t.Errorf("Expected echo 'Test message', got %v", resultMap["echo"])
		}
	})
}

// TestPythonRuntimeToolInjection 测试 Python 运行时工具注入
func TestPythonRuntimeToolInjection(t *testing.T) {
	runtime := NewPythonRuntime(nil)

	// 测试无工具的简单模式
	t.Run("NoToolsMode", func(t *testing.T) {
		code := "print('Hello World')"
		result, err := runtime.Execute(context.Background(), code, map[string]any{})
		if err != nil {
			t.Fatalf("Failed to execute: %v", err)
		}
		if !result.Success {
			t.Error("Expected success")
		}
	})

	// 测试带工具的 PTC 模式
	t.Run("PTCMode", func(t *testing.T) {
		runtime.SetTools([]string{"Read", "Write"})
		runtime.SetBridgeURL("http://localhost:8080")

		// 验证生成的代码包含工具注入
		wrappedCode := runtime.wrapCode("print('test')", map[string]any{})

		// 检查是否包含关键组件
		if !contains(wrappedCode, "_AsterBridge") {
			t.Error("Expected _AsterBridge class in wrapped code")
		}
		if !contains(wrappedCode, "async def _user_main()") {
			t.Error("Expected async main wrapper")
		}
		if !contains(wrappedCode, "Read") {
			t.Error("Expected Read tool in injected tools")
		}
	})
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}

// checkPythonAiohttp 检查 Python 和 aiohttp 是否可用
func checkPythonAiohttp() bool {
	runtime := NewPythonRuntime(nil)
	if !runtime.IsAvailable() {
		return false
	}

	// 检查 aiohttp 是否安装
	code := "import aiohttp"
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := runtime.Execute(ctx, code, map[string]any{})
	return err == nil && result.Success
}
