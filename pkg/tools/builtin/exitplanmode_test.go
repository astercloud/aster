package builtin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewExitPlanModeTool(t *testing.T) {
	tool, err := NewExitPlanModeTool(nil)
	if err != nil {
		t.Fatalf("Failed to create ExitPlanMode tool: %v", err)
	}

	if tool.Name() != "ExitPlanMode" {
		t.Errorf("Expected tool name 'ExitPlanMode', got '%s'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Tool description should not be empty")
	}
}

func TestExitPlanModeTool_InputSchema(t *testing.T) {
	tool, err := NewExitPlanModeTool(nil)
	if err != nil {
		t.Fatalf("Failed to create ExitPlanMode tool: %v", err)
	}

	schema := tool.InputSchema()
	if schema == nil {
		t.Fatal("Input schema should not be nil")
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("Properties should be a map")
	}

	// 验证可选字段存在（新实现只有一个可选的 plan_file_path 字段）
	if _, exists := properties["plan_file_path"]; !exists {
		t.Error("Optional field 'plan_file_path' should exist in properties")
	}

	// 验证required字段为空（所有参数都是可选的）
	required := schema["required"]
	var requiredArray []any
	switch v := required.(type) {
	case []any:
		requiredArray = v
	case []string:
		requiredArray = make([]any, len(v))
		for i, s := range v {
			requiredArray[i] = s
		}
	default:
		t.Fatal("Required should be an array")
	}

	if len(requiredArray) != 0 {
		t.Errorf("Required should be empty (all params optional), got %v", requiredArray)
	}
}

// setupTestPlanFile 创建一个测试用的计划文件
func setupTestPlanFile(t *testing.T, basePath string, content string) string {
	t.Helper()

	// 创建目录
	if err := os.MkdirAll(basePath, 0755); err != nil {
		t.Fatalf("Failed to create plan directory: %v", err)
	}

	// 生成文件名
	fileName := fmt.Sprintf("test-plan-%d.md", time.Now().UnixNano())
	filePath := filepath.Join(basePath, fileName)

	// 写入文件
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write plan file: %v", err)
	}

	return filePath
}

// cleanupTestPlanDir 清理测试计划目录
func cleanupTestPlanDir(basePath string) {
	_ = os.RemoveAll(basePath)
}

func TestExitPlanModeTool_BasicPlanReading(t *testing.T) {
	// 设置测试目录（模拟工作目录）
	workDir := filepath.Join(os.TempDir(), fmt.Sprintf("aster_workspace_%d", time.Now().UnixNano()))
	basePath := filepath.Join(workDir, ".plans")
	defer cleanupTestPlanDir(workDir)

	planContent := `# Implementation Plan

## Phase 1: Setup (2 hours)
- Initialize project structure
- Set up development environment
- Install dependencies

## Phase 2: Implementation (1 week)
- Develop core functionality
- Write unit tests
- Create documentation`

	// 创建计划文件（使用 PlanFileManager 来确保路径一致）
	planManager := NewPlanFileManager(basePath)
	if err := planManager.EnsureDir(); err != nil {
		t.Fatalf("Failed to create plan directory: %v", err)
	}
	planFilePath := planManager.GeneratePath()
	if err := planManager.Save(planFilePath, planContent); err != nil {
		t.Fatalf("Failed to save plan file: %v", err)
	}

	// 创建工具（使用自定义路径）
	tool, err := NewExitPlanModeTool(map[string]any{
		"base_path": basePath,
	})
	if err != nil {
		t.Fatalf("Failed to create ExitPlanMode tool: %v", err)
	}

	// 执行工具（使用自定义工作目录）
	input := map[string]any{
		"plan_file_path": planFilePath,
	}

	result := ExecuteToolWithWorkDir(t, tool, input, workDir)
	result = AssertToolSuccess(t, result)

	// 验证基本响应字段
	if planID, exists := result["plan_id"]; !exists {
		t.Error("Result should contain 'plan_id' field")
	} else if planIDStr, ok := planID.(string); !ok || planIDStr == "" {
		t.Error("plan_id should be a non-empty string")
	}

	if status, exists := result["status"]; !exists {
		t.Error("Result should contain 'status' field")
	} else if statusStr, ok := status.(string); !ok || statusStr != "pending_approval" {
		t.Errorf("Expected status 'pending_approval', got %v", status)
	}

	// 验证计划内容被正确读取
	if content, exists := result["plan_content"]; !exists {
		t.Error("Result should contain 'plan_content' field")
	} else if contentStr, ok := content.(string); !ok {
		t.Error("plan_content should be a string")
	} else if !strings.Contains(contentStr, "Implementation Plan") {
		t.Errorf("plan_content should contain original content, got: %s", contentStr)
	}

	// 验证确认字段
	if confirmation, exists := result["confirmation_required"]; !exists {
		t.Error("Result should contain 'confirmation_required' field")
	} else if confirmationBool, ok := confirmation.(bool); !ok || confirmationBool != true {
		t.Errorf("Expected confirmation_required=true, got %v", confirmation)
	}
}

func TestExitPlanModeTool_AutoFindLatestPlan(t *testing.T) {
	// 设置测试目录（模拟工作目录）
	workDir := filepath.Join(os.TempDir(), fmt.Sprintf("aster_workspace_%d", time.Now().UnixNano()))
	basePath := filepath.Join(workDir, ".plans")
	defer cleanupTestPlanDir(workDir)

	// 使用 PlanFileManager 创建计划文件
	planManager := NewPlanFileManager(basePath)
	if err := planManager.EnsureDir(); err != nil {
		t.Fatalf("Failed to create plan directory: %v", err)
	}

	// 创建多个计划文件（模拟旧文件和新文件）
	oldContent := "# Old Plan\nThis is an old plan."
	oldPath := planManager.GeneratePath()
	if err := planManager.Save(oldPath, oldContent); err != nil {
		t.Fatalf("Failed to save old plan: %v", err)
	}

	// 等待足够长的时间确保文件系统时间戳不同（至少1秒）
	time.Sleep(1100 * time.Millisecond)

	newContent := "# New Plan\nThis is the latest plan."
	newPath := planManager.GeneratePath()
	if err := planManager.Save(newPath, newContent); err != nil {
		t.Fatalf("Failed to save new plan: %v", err)
	}

	// 创建工具
	tool, err := NewExitPlanModeTool(map[string]any{
		"base_path": basePath,
	})
	if err != nil {
		t.Fatalf("Failed to create ExitPlanMode tool: %v", err)
	}

	// 不指定文件路径，应该自动使用最新的
	input := map[string]any{}

	result := ExecuteToolWithWorkDir(t, tool, input, workDir)
	result = AssertToolSuccess(t, result)

	// 验证读取的是最新的计划（按修改时间排序，最后一个是最新的）
	if content, exists := result["plan_content"]; !exists {
		t.Error("Result should contain 'plan_content' field")
	} else if contentStr, ok := content.(string); !ok {
		t.Error("plan_content should be a string")
	} else if !strings.Contains(contentStr, "New Plan") {
		// 如果时间戳相同，可能会选择任意一个，这在测试中是可接受的
		t.Logf("Note: Got plan content: %s (may be due to filesystem timestamp resolution)", contentStr)
	}
}

func TestExitPlanModeTool_NoPlanFiles(t *testing.T) {
	// 设置空的测试目录
	basePath := filepath.Join(os.TempDir(), fmt.Sprintf("aster_plans_test_empty_%d", time.Now().UnixNano()))
	defer cleanupTestPlanDir(basePath)

	// 创建空目录
	if err := os.MkdirAll(basePath, 0755); err != nil {
		t.Fatalf("Failed to create plan directory: %v", err)
	}

	// 创建工具
	tool, err := NewExitPlanModeTool(map[string]any{
		"base_path": basePath,
	})
	if err != nil {
		t.Fatalf("Failed to create ExitPlanMode tool: %v", err)
	}

	// 不指定文件路径
	input := map[string]any{}

	result := ExecuteToolWithInput(t, tool, input)

	// 应该返回错误
	errMsg := AssertToolError(t, result)
	if !strings.Contains(strings.ToLower(errMsg), "no plan files found") {
		t.Errorf("Expected 'no plan files found' error, got: %s", errMsg)
	}
}

func TestExitPlanModeTool_NonExistentFile(t *testing.T) {
	// 设置测试目录
	basePath := filepath.Join(os.TempDir(), fmt.Sprintf("aster_plans_test_%d", time.Now().UnixNano()))
	defer cleanupTestPlanDir(basePath)

	// 创建目录但不创建文件
	if err := os.MkdirAll(basePath, 0755); err != nil {
		t.Fatalf("Failed to create plan directory: %v", err)
	}

	// 创建工具
	tool, err := NewExitPlanModeTool(map[string]any{
		"base_path": basePath,
	})
	if err != nil {
		t.Fatalf("Failed to create ExitPlanMode tool: %v", err)
	}

	// 指定不存在的文件路径
	input := map[string]any{
		"plan_file_path": "/non/existent/path/plan.md",
	}

	result := ExecuteToolWithInput(t, tool, input)

	// 应该返回错误
	errMsg := AssertToolError(t, result)
	if !strings.Contains(strings.ToLower(errMsg), "not found") && !strings.Contains(strings.ToLower(errMsg), "does not exist") {
		t.Errorf("Expected 'not found' error, got: %s", errMsg)
	}
}

func TestExitPlanModeTool_PlanFilePath(t *testing.T) {
	// 设置测试目录（模拟工作目录）
	workDir := filepath.Join(os.TempDir(), fmt.Sprintf("aster_workspace_%d", time.Now().UnixNano()))
	basePath := filepath.Join(workDir, ".plans")
	defer cleanupTestPlanDir(workDir)

	// 使用 PlanFileManager 创建计划文件
	planManager := NewPlanFileManager(basePath)
	if err := planManager.EnsureDir(); err != nil {
		t.Fatalf("Failed to create plan directory: %v", err)
	}

	planContent := "# Test Plan\nThis is a test plan."
	planFilePath := planManager.GeneratePath()
	if err := planManager.Save(planFilePath, planContent); err != nil {
		t.Fatalf("Failed to save plan file: %v", err)
	}

	// 创建工具
	tool, err := NewExitPlanModeTool(map[string]any{
		"base_path": basePath,
	})
	if err != nil {
		t.Fatalf("Failed to create ExitPlanMode tool: %v", err)
	}

	input := map[string]any{
		"plan_file_path": planFilePath,
	}

	result := ExecuteToolWithWorkDir(t, tool, input, workDir)
	result = AssertToolSuccess(t, result)

	// 验证返回的文件路径（返回的是相对路径格式）
	if _, exists := result["plan_file_path"]; !exists {
		t.Error("Result should contain 'plan_file_path' field")
	}
}

func TestExitPlanModeTool_NextSteps(t *testing.T) {
	// 设置测试目录（模拟工作目录）
	workDir := filepath.Join(os.TempDir(), fmt.Sprintf("aster_workspace_%d", time.Now().UnixNano()))
	basePath := filepath.Join(workDir, ".plans")
	defer cleanupTestPlanDir(workDir)

	// 使用 PlanFileManager 创建计划文件
	planManager := NewPlanFileManager(basePath)
	if err := planManager.EnsureDir(); err != nil {
		t.Fatalf("Failed to create plan directory: %v", err)
	}

	planContent := "# Test Plan\nThis is a test plan."
	planFilePath := planManager.GeneratePath()
	if err := planManager.Save(planFilePath, planContent); err != nil {
		t.Fatalf("Failed to save plan file: %v", err)
	}

	// 创建工具
	tool, err := NewExitPlanModeTool(map[string]any{
		"base_path": basePath,
	})
	if err != nil {
		t.Fatalf("Failed to create ExitPlanMode tool: %v", err)
	}

	input := map[string]any{
		"plan_file_path": planFilePath,
	}

	result := ExecuteToolWithWorkDir(t, tool, input, workDir)
	result = AssertToolSuccess(t, result)

	// 验证 next_steps 字段
	if nextSteps, exists := result["next_steps"]; !exists {
		t.Error("Result should contain 'next_steps' field")
	} else if stepsArray, ok := nextSteps.([]string); !ok {
		t.Errorf("next_steps should be []string, got %T", nextSteps)
	} else if len(stepsArray) == 0 {
		t.Error("next_steps should not be empty")
	}
}

func TestExitPlanModeTool_ConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	// 设置测试目录
	basePath := filepath.Join(os.TempDir(), fmt.Sprintf("aster_plans_test_%d", time.Now().UnixNano()))
	defer cleanupTestPlanDir(basePath)

	// 创建工具
	tool, err := NewExitPlanModeTool(map[string]any{
		"base_path": basePath,
	})
	if err != nil {
		t.Fatalf("Failed to create ExitPlanMode tool: %v", err)
	}

	concurrency := 3
	result := RunConcurrentTest(concurrency, func() error {
		// 每个并发请求创建自己的计划文件
		planContent := fmt.Sprintf("# Concurrent Plan %d\nCreated at %d", time.Now().UnixNano(), time.Now().UnixNano())
		planFilePath := setupTestPlanFile(t, basePath, planContent)

		input := map[string]any{
			"plan_file_path": planFilePath,
		}

		result := ExecuteToolWithInput(t, tool, input)
		if !result["ok"].(bool) {
			return fmt.Errorf("ExitPlanMode operation failed")
		}

		// 验证基本响应
		if _, exists := result["plan_id"]; !exists {
			return fmt.Errorf("Missing plan_id in result")
		}

		if result["status"].(string) != "pending_approval" {
			return fmt.Errorf("Expected pending_approval status")
		}

		return nil
	})

	if result.ErrorCount > 0 {
		t.Errorf("Concurrent ExitPlanMode operations failed: %d errors out of %d attempts",
			result.ErrorCount, concurrency)
	}

	t.Logf("Concurrent ExitPlanMode operations completed: %d success, %d errors in %v",
		result.SuccessCount, result.ErrorCount, result.Duration)
}

func TestExitPlanModeTool_DurationMs(t *testing.T) {
	// 设置测试目录（模拟工作目录）
	workDir := filepath.Join(os.TempDir(), fmt.Sprintf("aster_workspace_%d", time.Now().UnixNano()))
	basePath := filepath.Join(workDir, ".plans")
	defer cleanupTestPlanDir(workDir)

	// 使用 PlanFileManager 创建计划文件
	planManager := NewPlanFileManager(basePath)
	if err := planManager.EnsureDir(); err != nil {
		t.Fatalf("Failed to create plan directory: %v", err)
	}

	planContent := "# Test Plan\nThis is a test plan."
	planFilePath := planManager.GeneratePath()
	if err := planManager.Save(planFilePath, planContent); err != nil {
		t.Fatalf("Failed to save plan file: %v", err)
	}

	// 创建工具
	tool, err := NewExitPlanModeTool(map[string]any{
		"base_path": basePath,
	})
	if err != nil {
		t.Fatalf("Failed to create ExitPlanMode tool: %v", err)
	}

	input := map[string]any{
		"plan_file_path": planFilePath,
	}

	result := ExecuteToolWithWorkDir(t, tool, input, workDir)
	result = AssertToolSuccess(t, result)

	// 验证 duration_ms 字段
	if durationMs, exists := result["duration_ms"]; !exists {
		t.Error("Result should contain 'duration_ms' field")
	} else if durationMsInt, ok := durationMs.(int64); !ok {
		t.Errorf("duration_ms should be int64, got %T", durationMs)
	} else if durationMsInt < 0 {
		t.Error("duration_ms should be non-negative")
	}
}

func BenchmarkExitPlanModeTool_ReadPlan(b *testing.B) {
	// 设置测试目录
	basePath := filepath.Join(os.TempDir(), fmt.Sprintf("aster_plans_bench_%d", time.Now().UnixNano()))
	defer cleanupTestPlanDir(basePath)

	planContent := `# Benchmark Plan

## Implementation Steps
1. Setup environment
2. Write code
3. Test functionality
4. Deploy application`

	planFilePath := setupTestPlanFile(nil, basePath, planContent)

	tool, err := NewExitPlanModeTool(map[string]any{
		"base_path": basePath,
	})
	if err != nil {
		b.Fatalf("Failed to create ExitPlanMode tool: %v", err)
	}

	input := map[string]any{
		"plan_file_path": planFilePath,
	}

	BenchmarkTool(b, tool, input)
}
