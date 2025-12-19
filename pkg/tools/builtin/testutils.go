package builtin

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/tools"
)

// TestHelper 测试辅助工具
type TestHelper struct {
	T       *testing.T
	TmpDir  string
	Context context.Context
	Cleanup []func() // 清理函数列表
}

// NewTestHelper 创建测试辅助工具
func NewTestHelper(t *testing.T) *TestHelper {
	tmpDir, err := os.MkdirTemp("", "agentsdk_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	return &TestHelper{
		T:       t,
		TmpDir:  tmpDir,
		Context: context.Background(),
		Cleanup: []func(){func() { _ = os.RemoveAll(tmpDir) }},
	}
}

// AddCleanup 添加清理函数
func (th *TestHelper) AddCleanup(cleanup func()) {
	th.Cleanup = append(th.Cleanup, cleanup)
}

// CleanupAll 执行所有清理
func (th *TestHelper) CleanupAll() {
	// 反向执行清理函数
	for i := len(th.Cleanup) - 1; i >= 0; i-- {
		th.Cleanup[i]()
	}
}

// CreateTempFile 创建临时文件
func (th *TestHelper) CreateTempFile(name, content string) string {
	path := filepath.Join(th.TmpDir, name)

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		th.T.Fatalf("Failed to create dir: %v", err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		th.T.Fatalf("Failed to create file: %v", err)
	}

	return path
}

// CreateTempDir 创建临时目录
func (th *TestHelper) CreateTempDir(name string) string {
	path := filepath.Join(th.TmpDir, name)
	if err := os.MkdirAll(path, 0755); err != nil {
		th.T.Fatalf("Failed to create dir: %v", err)
	}
	return path
}

// ReadFile 读取文件内容
func (th *TestHelper) ReadFile(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		th.T.Fatalf("Failed to read file: %v", err)
	}
	return string(content)
}

// FileExists 检查文件是否存在
func (th *TestHelper) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetTestDataPath 获取测试数据文件路径
func (th *TestHelper) GetTestDataPath(name string) string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		th.T.Fatalf("Failed to get caller info")
	}
	baseDir := filepath.Dir(thisFile)
	return filepath.Join(baseDir, "testdata", name)
}

// NewMockToolContext 创建模拟工具上下文
func NewMockToolContext() sandbox.Sandbox {
	return sandbox.NewMockSandbox()
}

// ExecuteToolWithInput 使用指定输入执行工具
func ExecuteToolWithInput(t *testing.T, tool tools.Tool, input map[string]any) map[string]any {
	ctx := context.Background()
	tc := &tools.ToolContext{
		Signal:  ctx,
		Sandbox: NewMockToolContext(),
	}

	result, err := tool.Execute(ctx, input, tc)
	if err != nil {
		t.Fatalf("Tool execution failed: %v", err)
	}

	// 将result转换为map[string]any
	if resultMap, ok := result.(map[string]any); ok {
		return resultMap
	}

	t.Fatalf("Expected map[string]any result, got %T", result)
	return nil
}

// ExecuteToolWithRealFS 使用真实文件系统执行工具
func ExecuteToolWithRealFS(t *testing.T, tool tools.Tool, input map[string]any) map[string]any {
	ctx := context.Background()
	// 使用真实的文件系统而不是Mock
	tc := &tools.ToolContext{
		Signal:  ctx,
		Sandbox: &RealSandbox{},
	}

	result, err := tool.Execute(ctx, input, tc)
	if err != nil {
		t.Fatalf("Tool execution failed: %v", err)
	}

	// 将result转换为map[string]any
	if resultMap, ok := result.(map[string]any); ok {
		return resultMap
	}

	t.Fatalf("Expected map[string]any result, got %T", result)
	return nil
}

// ExecuteToolWithWorkDir 使用指定工作目录执行工具
func ExecuteToolWithWorkDir(t *testing.T, tool tools.Tool, input map[string]any, workDir string) map[string]any {
	ctx := context.Background()
	tc := &tools.ToolContext{
		Signal:  ctx,
		Sandbox: &CustomWorkDirSandbox{workDir: workDir},
	}

	result, err := tool.Execute(ctx, input, tc)
	if err != nil {
		t.Fatalf("Tool execution failed: %v", err)
	}

	// 将result转换为map[string]any
	if resultMap, ok := result.(map[string]any); ok {
		return resultMap
	}

	t.Fatalf("Expected map[string]any result, got %T", result)
	return nil
}

// ExecuteToolWithoutContext 不使用 ToolContext 执行工具（用于测试不依赖 sandbox 的工具）
func ExecuteToolWithoutContext(t *testing.T, tool tools.Tool, input map[string]any) map[string]any {
	ctx := context.Background()

	result, err := tool.Execute(ctx, input, nil)
	if err != nil {
		t.Fatalf("Tool execution failed: %v", err)
	}

	// 将result转换为map[string]any
	if resultMap, ok := result.(map[string]any); ok {
		return resultMap
	}

	t.Fatalf("Expected map[string]any result, got %T", result)
	return nil
}

// CustomWorkDirSandbox 自定义工作目录的沙箱（仅用于测试）
type CustomWorkDirSandbox struct {
	workDir string
}

func (cs *CustomWorkDirSandbox) Kind() string {
	return "custom"
}

func (cs *CustomWorkDirSandbox) WorkDir() string {
	return cs.workDir
}

func (cs *CustomWorkDirSandbox) FS() sandbox.SandboxFS {
	return &RealFS{}
}

func (cs *CustomWorkDirSandbox) Exec(ctx context.Context, cmd string, opts *sandbox.ExecOptions) (*sandbox.ExecResult, error) {
	return nil, errors.New("exec not supported in test sandbox")
}

func (cs *CustomWorkDirSandbox) Watch(paths []string, listener sandbox.FileChangeListener) (string, error) {
	return "", errors.New("watch not supported in test sandbox")
}

func (cs *CustomWorkDirSandbox) Unwatch(watchID string) error {
	return nil
}

func (cs *CustomWorkDirSandbox) Dispose() error {
	return nil
}

// RealSandbox 使用真实文件系统的沙箱（仅用于测试）
type RealSandbox struct{}

func (rs *RealSandbox) Kind() string {
	return "real"
}

func (rs *RealSandbox) WorkDir() string {
	return os.TempDir()
}

func (rs *RealSandbox) FS() sandbox.SandboxFS {
	return &RealFS{}
}

func (rs *RealSandbox) Exec(ctx context.Context, cmd string, opts *sandbox.ExecOptions) (*sandbox.ExecResult, error) {
	return nil, errors.New("exec not supported in test sandbox")
}

func (rs *RealSandbox) Watch(paths []string, listener sandbox.FileChangeListener) (string, error) {
	return "", errors.New("watch not supported in test sandbox")
}

func (rs *RealSandbox) Unwatch(watchID string) error {
	return nil
}

func (rs *RealSandbox) Dispose() error {
	return nil
}

// RealFS 使用真实文件系统
type RealFS struct{}

func (rfs *RealFS) Resolve(path string) string {
	return filepath.Clean(path)
}

func (rfs *RealFS) IsInside(path string) bool {
	absPath, _ := filepath.Abs(path)
	tmpDir := os.TempDir()
	absTmp, _ := filepath.Abs(tmpDir)
	return strings.HasPrefix(absPath, absTmp)
}

func (rfs *RealFS) Read(ctx context.Context, path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (rfs *RealFS) Write(ctx context.Context, path string, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func (rfs *RealFS) Temp(name string) string {
	return filepath.Join(os.TempDir(), name)
}

func (rfs *RealFS) Stat(ctx context.Context, path string) (sandbox.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return sandbox.FileInfo{}, err
	}
	return sandbox.FileInfo{
		Path:    path,
		Size:    info.Size(),
		ModTime: info.ModTime(),
		IsDir:   info.IsDir(),
		Mode:    int(info.Mode()),
	}, nil
}

func (rfs *RealFS) Glob(ctx context.Context, pattern string, opts *sandbox.GlobOptions) ([]string, error) {
	// 简单的glob实现
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	return matches, nil
}

// AssertToolSuccess 断言工具执行成功
func AssertToolSuccess(t *testing.T, result map[string]any) map[string]any {
	if ok, exists := result["ok"]; !exists || !ok.(bool) {
		t.Errorf("Expected tool to succeed, got result: %+v", result)
	}
	return result
}

// AssertToolError 断言工具执行失败
func AssertToolError(t *testing.T, result map[string]any) string {
	if ok, exists := result["ok"]; exists && ok.(bool) {
		t.Errorf("Expected tool to fail, but it succeeded")
	}

	if errMsg, exists := result["error"]; exists {
		if errStr, ok := errMsg.(string); ok {
			return errStr
		}
	}
	t.Errorf("Expected error message in result, got: %+v", result)
	return ""
}

// AssertContains 断言字符串包含子字符串
func AssertContains(t *testing.T, str, substr string) {
	if !strings.Contains(str, substr) {
		t.Errorf("Expected string to contain %q, got: %q", substr, str)
	}
}

// AssertFileExists 断言文件存在
func AssertFileExists(t *testing.T, path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Expected file %q to exist, but it doesn't", path)
	}
}

// AssertFileNotExists 断言文件不存在
func AssertFileNotExists(t *testing.T, path string) {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("Expected file %q to not exist, but it does", path)
	}
}

// AssertFileContent 断言文件内容
func AssertFileContent(t *testing.T, path, expectedContent string) {
	content, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("Failed to read file %q: %v", path, err)
		return
	}

	if string(content) != expectedContent {
		t.Errorf("File content mismatch for %q.\nExpected: %q\nActual:   %q",
			path, expectedContent, string(content))
	}
}

// CreateTestFiles 创建标准测试文件
func CreateTestFiles(th *TestHelper) map[string]string {
	files := make(map[string]string)

	// Go源代码文件
	files["test.go"] = `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}

func add(a, b int) int {
	return a + b
}
`

	// JSON配置文件
	files["config.json"] = "{\n  \"name\": \"test-config\",\n  \"version\": \"1.0.0\",\n  \"settings\": {\n    \"debug\": true,\n    \"port\": 8080\n  },\n  \"dependencies\": [\n    \"github.com/example/pkg1\",\n    \"github.com/example/pkg2\"\n  ]\n}\n"

	// Markdown文档
	files["readme.md"] = "# Test Project\n\nThis is a test project with various file types.\n\n## Features\n\n- Feature 1\n- Feature 2\n\n## Usage\n\n```go\npackage main\n\nfunc main() {\n    println(\"test\")\n}\n```\n\n## Configuration\n\nSee `config.json` for configuration options.\n"

	// 文本文件
	files["data.txt"] = "Line 1: Basic text content\nLine 2: Some numbers 123 456\nLine 3: Special chars !@#$%^&*()\nLine 4: Unicode content 你好世界 🌍\nLine 5: Mixed content and URLs https://example.com\n"

	// 空文件
	files["empty.txt"] = ""

	// 大文件 (1MB)
	largeContent := strings.Repeat("This is line for testing large file processing.\n", 1024*64)
	files["large.txt"] = largeContent

	// 创建文件
	for name, content := range files {
		path := th.CreateTempFile(name, content)
		files[name] = path
	}

	return files
}

// ConcurrentTestResult 并发测试结果
type ConcurrentTestResult struct {
	SuccessCount int
	ErrorCount   int
	Errors       []error
	Duration     time.Duration
}

// RunConcurrentTest 运行并发测试
func RunConcurrentTest(concurrency int, testFunc func() error) *ConcurrentTestResult {
	results := make(chan error, concurrency)

	start := time.Now()

	// 启动并发goroutines
	for range concurrency {
		go func() {
			results <- testFunc()
		}()
	}

	// 收集结果
	successCount := 0
	errorCount := 0
	var errors []error

	for range concurrency {
		if err := <-results; err != nil {
			errorCount++
			errors = append(errors, err)
		} else {
			successCount++
		}
	}

	return &ConcurrentTestResult{
		SuccessCount: successCount,
		ErrorCount:   errorCount,
		Errors:       errors,
		Duration:     time.Since(start),
	}
}

// SkipIfShort 如果是短测试模式则跳过
func SkipIfShort(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
}

// BenchmarkTool 工具性能基准测试辅助函数
func BenchmarkTool(b *testing.B, tool tools.Tool, input map[string]any) {
	ctx := context.Background()
	tc := &tools.ToolContext{
		Signal:  ctx,
		Sandbox: sandbox.NewMockSandbox(),
	}

	b.ResetTimer()
	for range b.N {
		_, err := tool.Execute(ctx, input, tc)
		if err != nil {
			b.Fatalf("Tool execution failed: %v", err)
		}
	}
}
