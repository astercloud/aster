package builtin

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestNewBashTool(t *testing.T) {
	tool, err := NewBashTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Bash tool: %v", err)
	}

	if tool.Name() != "Bash" {
		t.Errorf("Expected tool name 'Bash', got '%s'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Tool description should not be empty")
	}
}

func TestBashTool_InputSchema(t *testing.T) {
	tool, err := NewBashTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Bash tool: %v", err)
	}

	schema := tool.InputSchema()
	if schema == nil {
		t.Fatal("Input schema should not be nil")
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("Properties should be a map")
	}

	// 验证必需字段存在
	requiredFields := []string{"command"}
	for _, field := range requiredFields {
		if _, exists := properties[field]; !exists {
			t.Errorf("Required field '%s' should exist in properties", field)
		}
	}
}

func TestBashTool_SimpleCommand(t *testing.T) {
	tool, err := NewBashTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Bash tool: %v", err)
	}

	input := map[string]any{
		"command": "echo 'Hello, World!'",
	}

	result := ExecuteToolWithInput(t, tool, input)
	result = AssertToolSuccess(t, result)

	// 验证输出包含预期内容
	if stdout, exists := result["stdout"]; !exists {
		t.Error("Result should contain 'stdout' field")
	} else if stdoutStr, ok := stdout.(string); !ok {
		t.Error("stdout should be a string")
	} else if !strings.Contains(stdoutStr, "Hello, World!") {
		t.Errorf("Expected output to contain 'Hello, World!', got: %s", stdoutStr)
	}

	// 验证其他字段
	if exitCode, exists := result["exit_code"]; !exists {
		t.Error("Result should contain 'exit_code' field")
	} else if exitCodeInt, ok := exitCode.(int); !ok || exitCodeInt != 0 {
		t.Errorf("Expected exit code 0, got %v", exitCode)
	}
}

func TestBashTool_CommandWithArguments(t *testing.T) {
	t.Skip("Skipping: requires real command execution (mock sandbox doesn't execute real commands)")

	if os.Getenv("CI") != "" {
		t.Skip("Skipping bash execution test in CI environment")
	}
	tool, err := NewBashTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Bash tool: %v", err)
	}

	input := map[string]any{
		"command": "echo",
		"args":    []string{"arg1", "arg2", "arg3"},
	}

	result := ExecuteToolWithInput(t, tool, input)
	result = AssertToolSuccess(t, result)

	if stdout, exists := result["stdout"]; !exists {
		t.Error("Result should contain 'stdout' field")
	} else if stdoutStr, ok := stdout.(string); !ok {
		t.Error("stdout should be a string")
	} else {
		// 验证参数被正确传递
		if !strings.Contains(stdoutStr, "arg1") {
			t.Error("Expected arg1 in output")
		}
	}
}

func TestBashTool_WorkingDirectory(t *testing.T) {
	t.Skip("Skipping: requires real command execution (mock sandbox doesn't execute real commands)")

	if os.Getenv("CI") != "" {
		t.Skip("Skipping working directory test in CI environment")
	}
	if testing.Short() {
		t.Skip("Skipping working directory test in short mode")
	}

	tool, err := NewBashTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Bash tool: %v", err)
	}

	// 使用临时目录作为工作目录
	helper := NewTestHelper(t)
	defer helper.CleanupAll()

	input := map[string]any{
		"command": "pwd",
		"cwd":     helper.TmpDir,
	}

	result := ExecuteToolWithInput(t, tool, input)
	result = AssertToolSuccess(t, result)

	if stdout, exists := result["stdout"]; !exists {
		t.Error("Result should contain 'stdout' field")
	} else if stdoutStr, ok := stdout.(string); !ok {
		t.Error("stdout should be a string")
	} else {
		// 验证工作目录被正确设置
		if !strings.Contains(stdoutStr, helper.TmpDir) {
			t.Errorf("Expected output to contain %s, got: %s", helper.TmpDir, stdoutStr)
		}
	}
}

func TestBashTool_Timeout(t *testing.T) {
	t.Skip("Skipping: requires real command execution (mock sandbox doesn't execute real commands)")

	if os.Getenv("CI") != "" {
		t.Skip("Skipping timeout test in CI environment")
	}
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}

	tool, err := NewBashTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Bash tool: %v", err)
	}

	input := map[string]any{
		"command":      "sleep 10",
		"timeout":      2, // 2秒超时
		"timeout_unit": "seconds",
	}

	result := ExecuteToolWithInput(t, tool, input)

	// 超时情况下应该返回错误
	errMsg := AssertToolError(t, result)
	if !strings.Contains(strings.ToLower(errMsg), "timeout") &&
		!strings.Contains(strings.ToLower(errMsg), "timed out") {
		t.Errorf("Expected timeout error, got: %s", errMsg)
	}
}

func TestBashTool_DangerousCommands(t *testing.T) {
	tool, err := NewBashTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Bash tool: %v", err)
	}

	dangerousCommands := []string{
		"rm -rf /",
		"dd if=/dev/zero of=/dev/sda",
		"format c:",
		"sudo rm -rf /*",
	}

	for _, cmd := range dangerousCommands {
		t.Run("Dangerous_"+cmd, func(t *testing.T) {
			input := map[string]any{
				"command": cmd,
			}

			result := ExecuteToolWithInput(t, tool, input)

			// 危险命令应该被阻止
			errMsg := AssertToolError(t, result)
			if !strings.Contains(strings.ToLower(errMsg), "security") &&
				!strings.Contains(strings.ToLower(errMsg), "dangerous") &&
				!strings.Contains(strings.ToLower(errMsg), "blocked") {
				t.Errorf("Expected security error for dangerous command, got: %s", errMsg)
			}
		})
	}
}

func TestBashTool_EnvironmentVariables(t *testing.T) {
	t.Skip("Skipping: requires real command execution (mock sandbox doesn't execute real commands)")

	if os.Getenv("CI") != "" {
		t.Skip("Skipping environment test in CI environment")
	}
	tool, err := NewBashTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Bash tool: %v", err)
	}

	input := map[string]any{
		"command": "echo $TEST_VAR",
		"env": map[string]string{
			"TEST_VAR": "test_value",
		},
	}

	result := ExecuteToolWithInput(t, tool, input)
	result = AssertToolSuccess(t, result)

	if stdout, exists := result["stdout"]; !exists {
		t.Error("Result should contain 'stdout' field")
	} else if stdoutStr, ok := stdout.(string); !ok {
		t.Error("stdout should be a string")
	} else if !strings.Contains(stdoutStr, "test_value") {
		t.Errorf("Expected output to contain 'test_value', got: %s", stdoutStr)
	}
}

func TestBashTool_BackgroundExecution(t *testing.T) {
	t.Skip("Skipping: requires real background task management (BashTool background support + TaskManager)")

	if os.Getenv("CI") != "" {
		t.Skip("Skipping background execution test in CI environment")
	}
	tool, err := NewBashTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Bash tool: %v", err)
	}

	input := map[string]any{
		"command":    "echo 'Background task started'",
		"background": true,
	}

	result := ExecuteToolWithInput(t, tool, input)
	result = AssertToolSuccess(t, result)

	// 后台任务应该返回任务ID
	if taskID, exists := result["task_id"]; !exists {
		t.Error("Result should contain 'task_id' field for background execution")
	} else if taskIDStr, ok := taskID.(string); !ok {
		t.Error("task_id should be a string")
	} else if taskIDStr == "" {
		t.Error("task_id should not be empty")
	}

	// 验证状态
	if status, exists := result["status"]; !exists {
		t.Error("Result should contain 'status' field")
	} else if statusStr, ok := status.(string); !ok || statusStr != "running" {
		t.Errorf("Expected status 'running', got: %v", status)
	}
}

func TestBashTool_ConcurrentExecution(t *testing.T) {
	tool, err := NewBashTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Bash tool: %v", err)
	}

	concurrency := 5
	result := RunConcurrentTest(concurrency, func() error {
		input := map[string]any{
			"command": "echo 'Concurrent test'",
		}
		result := ExecuteToolWithInput(t, tool, input)
		if !result["ok"].(bool) {
			return errors.New("Tool execution failed")
		}
		return nil
	})

	if result.ErrorCount > 0 {
		t.Errorf("Concurrent bash execution failed: %d errors out of %d attempts",
			result.ErrorCount, concurrency)
	}

	t.Logf("Concurrent bash execution completed: %d success, %d errors in %v",
		result.SuccessCount, result.ErrorCount, result.Duration)
}

func TestBashTool_DifferentShellTypes(t *testing.T) {
	tool, err := NewBashTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Bash tool: %v", err)
	}

	shellTypes := []string{"bash", "sh", "zsh"}

	for _, shellType := range shellTypes {
		t.Run("Shell_"+shellType, func(t *testing.T) {
			input := map[string]any{
				"command": "echo 'Shell test'",
				"shell":   shellType,
			}

			result := ExecuteToolWithInput(t, tool, input)
			result = AssertToolSuccess(t, result)

			// 大多数系统都支持这些shell类型
			if stdout, exists := result["stdout"]; !exists {
				t.Error("Result should contain 'stdout' field")
			} else if stdoutStr, ok := stdout.(string); !ok {
				t.Error("stdout should be a string")
			} else if !strings.Contains(stdoutStr, "Shell test") {
				t.Errorf("Expected output to contain 'Shell test', got: %s", stdoutStr)
			}
		})
	}
}

func TestBashTool_PipeAndRedirection(t *testing.T) {
	t.Skip("Skipping: requires real command execution (mock sandbox doesn't execute real commands)")

	if os.Getenv("CI") != "" {
		t.Skip("Skipping pipe/redirection test in CI environment")
	}
	tool, err := NewBashTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Bash tool: %v", err)
	}

	input := map[string]any{
		"command": "echo 'Hello' | tr '[:lower:]' '[:upper:]'",
	}

	result := ExecuteToolWithInput(t, tool, input)
	result = AssertToolSuccess(t, result)

	if stdout, exists := result["stdout"]; !exists {
		t.Error("Result should contain 'stdout' field")
	} else if stdoutStr, ok := stdout.(string); !ok {
		t.Error("stdout should be a string")
	} else if !strings.Contains(stdoutStr, "HELLO") {
		t.Errorf("Expected output to contain 'HELLO', got: %s", stdoutStr)
	}
}

func TestBashTool_CommandChaining(t *testing.T) {
	t.Skip("Skipping: requires real command execution (mock sandbox doesn't execute real commands)")

	if os.Getenv("CI") != "" {
		t.Skip("Skipping command chaining test in CI environment")
	}
	tool, err := NewBashTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Bash tool: %v", err)
	}

	input := map[string]any{
		"command": "echo 'First' && echo 'Second' || echo 'This should not run'",
	}

	result := ExecuteToolWithInput(t, tool, input)
	result = AssertToolSuccess(t, result)

	if stdout, exists := result["stdout"]; !exists {
		t.Error("Result should contain 'stdout' field")
	} else if stdoutStr, ok := stdout.(string); !ok {
		t.Error("stdout should be a string")
	} else {
		// 验证两个命令都执行了
		if !strings.Contains(stdoutStr, "First") {
			t.Error("Expected 'First' in output")
		}
		if !strings.Contains(stdoutStr, "Second") {
			t.Error("Expected 'Second' in output")
		}
		if strings.Contains(stdoutStr, "This should not run") {
			t.Error("Should not contain 'This should not run'")
		}
	}
}

func BenchmarkBashTool_SimpleCommand(b *testing.B) {
	tool, err := NewBashTool(nil)
	if err != nil {
		b.Fatalf("Failed to create Bash tool: %v", err)
	}

	input := map[string]any{
		"command": "echo 'benchmark'",
	}

	BenchmarkTool(b, tool, input)
}

func BenchmarkBashTool_ComplexCommand(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping complex command benchmark in short mode")
	}

	tool, err := NewBashTool(nil)
	if err != nil {
		b.Fatalf("Failed to create Bash tool: %v", err)
	}

	input := map[string]any{
		"command": "seq 1 100 | tail -10",
	}

	BenchmarkTool(b, tool, input)
}
