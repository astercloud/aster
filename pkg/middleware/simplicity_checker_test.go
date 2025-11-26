package middleware

import (
	"context"
	"sync"
	"testing"
)

func TestNewSimplicityCheckerMiddleware(t *testing.T) {
	// 测试默认配置
	m := NewSimplicityCheckerMiddleware(nil)
	if m == nil {
		t.Fatal("Expected non-nil middleware")
	}
	if m.config.Enabled != true {
		t.Error("Expected Enabled to be true by default")
	}
	if m.config.MaxHelperFunctions != 3 {
		t.Errorf("Expected MaxHelperFunctions=3, got %d", m.config.MaxHelperFunctions)
	}

	// 测试自定义配置
	m2 := NewSimplicityCheckerMiddleware(&SimplicityCheckerConfig{
		Enabled:            false,
		MaxHelperFunctions: 5,
	})
	if m2.config.Enabled != false {
		t.Error("Expected Enabled to be false")
	}
	if m2.config.MaxHelperFunctions != 5 {
		t.Errorf("Expected MaxHelperFunctions=5, got %d", m2.config.MaxHelperFunctions)
	}
}

func TestSimplicityChecker_HelperFunctionDetection(t *testing.T) {
	var warnings []SimplicityWarning
	var mu sync.Mutex

	m := NewSimplicityCheckerMiddleware(&SimplicityCheckerConfig{
		Enabled:            true,
		MaxHelperFunctions: 2,
		OnWarning: func(w SimplicityWarning) {
			mu.Lock()
			warnings = append(warnings, w)
			mu.Unlock()
		},
	})

	// 模拟 Write 工具调用，包含多个 Helper 函数
	code := `
func parseHelper(s string) string {
    return s
}

func formatHelper(s string) string {
    return s
}

func validateHelper(s string) bool {
    return true
}
`

	req := &ToolCallRequest{
		ToolName: "Write",
		ToolInput: map[string]interface{}{
			"file_path": "test.go",
			"content":   code,
		},
	}

	// 执行检测
	handler := func(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
		return &ToolCallResponse{Result: "ok"}, nil
	}

	_, err := m.WrapToolCall(context.Background(), req, handler)
	if err != nil {
		t.Fatalf("WrapToolCall failed: %v", err)
	}

	// 验证警告
	mu.Lock()
	defer mu.Unlock()
	if len(warnings) == 0 {
		t.Error("Expected warnings for helper function overflow")
	}

	foundHelperWarning := false
	for _, w := range warnings {
		if w.Type == WarningTypeHelperOverflow {
			foundHelperWarning = true
			break
		}
	}
	if !foundHelperWarning {
		t.Error("Expected WarningTypeHelperOverflow warning")
	}
}

func TestSimplicityChecker_InterfaceDetection(t *testing.T) {
	var warnings []SimplicityWarning
	var mu sync.Mutex

	m := NewSimplicityCheckerMiddleware(&SimplicityCheckerConfig{
		Enabled:                    true,
		WarnOnPrematureAbstraction: true,
		OnWarning: func(w SimplicityWarning) {
			mu.Lock()
			warnings = append(warnings, w)
			mu.Unlock()
		},
	})

	// 模拟创建多个接口
	code := `
type Reader interface {
    Read() string
}

type Writer interface {
    Write(s string)
}

type Processor interface {
    Process() error
}
`

	req := &ToolCallRequest{
		ToolName: "Write",
		ToolInput: map[string]interface{}{
			"file_path": "interfaces.go",
			"content":   code,
		},
	}

	handler := func(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
		return &ToolCallResponse{Result: "ok"}, nil
	}

	_, err := m.WrapToolCall(context.Background(), req, handler)
	if err != nil {
		t.Fatalf("WrapToolCall failed: %v", err)
	}

	// 验证警告
	mu.Lock()
	defer mu.Unlock()
	foundAbstractionWarning := false
	for _, w := range warnings {
		if w.Type == WarningTypePrematureAbstraction {
			foundAbstractionWarning = true
			break
		}
	}
	if !foundAbstractionWarning {
		t.Error("Expected WarningTypePrematureAbstraction warning")
	}
}

func TestSimplicityChecker_BackwardsCompatHack(t *testing.T) {
	var warnings []SimplicityWarning
	var mu sync.Mutex

	m := NewSimplicityCheckerMiddleware(&SimplicityCheckerConfig{
		Enabled:            true,
		WarnOnUnusedParams: true,
		OnWarning: func(w SimplicityWarning) {
			mu.Lock()
			warnings = append(warnings, w)
			mu.Unlock()
		},
	})

	// 模拟向后兼容 hack 代码
	code := `
func process(data string) {
    _unused := data // removed: old logic
    // TODO: remove this
    _deprecated = "value"
}
`

	req := &ToolCallRequest{
		ToolName: "Edit",
		ToolInput: map[string]interface{}{
			"file_path":  "legacy.go",
			"new_string": code,
		},
	}

	handler := func(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
		return &ToolCallResponse{Result: "ok"}, nil
	}

	_, err := m.WrapToolCall(context.Background(), req, handler)
	if err != nil {
		t.Fatalf("WrapToolCall failed: %v", err)
	}

	// 验证警告
	mu.Lock()
	defer mu.Unlock()
	foundBackwardsCompatWarning := false
	for _, w := range warnings {
		if w.Type == WarningTypeBackwardsCompatHack {
			foundBackwardsCompatWarning = true
			break
		}
	}
	if !foundBackwardsCompatWarning {
		t.Error("Expected WarningTypeBackwardsCompatHack warning")
	}
}

func TestSimplicityChecker_DisabledMode(t *testing.T) {
	var warnings []SimplicityWarning
	var mu sync.Mutex

	m := NewSimplicityCheckerMiddleware(&SimplicityCheckerConfig{
		Enabled: false,
		OnWarning: func(w SimplicityWarning) {
			mu.Lock()
			warnings = append(warnings, w)
			mu.Unlock()
		},
	})

	// 模拟触发警告的代码
	code := `
func helper1() {}
func helper2() {}
func helper3() {}
func helper4() {}
`

	req := &ToolCallRequest{
		ToolName: "Write",
		ToolInput: map[string]interface{}{
			"file_path": "test.go",
			"content":   code,
		},
	}

	handler := func(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
		return &ToolCallResponse{Result: "ok"}, nil
	}

	_, err := m.WrapToolCall(context.Background(), req, handler)
	if err != nil {
		t.Fatalf("WrapToolCall failed: %v", err)
	}

	// 禁用模式下不应有警告
	mu.Lock()
	defer mu.Unlock()
	if len(warnings) > 0 {
		t.Errorf("Expected no warnings in disabled mode, got %d", len(warnings))
	}
}

func TestSimplicityChecker_GetWarnings(t *testing.T) {
	m := NewSimplicityCheckerMiddleware(&SimplicityCheckerConfig{
		Enabled:            true,
		MaxHelperFunctions: 1,
	})

	code := `
func parseHelper() {}
func formatHelper() {}
`

	req := &ToolCallRequest{
		ToolName: "Write",
		ToolInput: map[string]interface{}{
			"file_path": "test.go",
			"content":   code,
		},
	}

	handler := func(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
		return &ToolCallResponse{Result: "ok"}, nil
	}

	_, _ = m.WrapToolCall(context.Background(), req, handler)

	warnings := m.GetWarnings()
	if len(warnings) == 0 {
		t.Error("Expected warnings to be recorded")
	}
}

func TestSimplicityChecker_Reset(t *testing.T) {
	m := NewSimplicityCheckerMiddleware(&SimplicityCheckerConfig{
		Enabled:            true,
		MaxHelperFunctions: 1,
	})

	code := `func parseHelper() {}`

	req := &ToolCallRequest{
		ToolName: "Write",
		ToolInput: map[string]interface{}{
			"file_path": "test.go",
			"content":   code,
		},
	}

	handler := func(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
		return &ToolCallResponse{Result: "ok"}, nil
	}

	_, _ = m.WrapToolCall(context.Background(), req, handler)

	if m.helperCount == 0 {
		t.Error("Expected helperCount > 0 after detection")
	}

	// 重置
	m.Reset()

	if m.helperCount != 0 {
		t.Errorf("Expected helperCount=0 after reset, got %d", m.helperCount)
	}
	if len(m.GetWarnings()) != 0 {
		t.Error("Expected empty warnings after reset")
	}
}

func TestSimplicityChecker_OnAgentStart(t *testing.T) {
	m := NewSimplicityCheckerMiddleware(&SimplicityCheckerConfig{
		Enabled:            true,
		MaxHelperFunctions: 1,
	})

	// 添加一些状态
	m.helperCount = 10
	m.interfaceCount = 5

	// 调用 OnAgentStart 应该重置
	err := m.OnAgentStart(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("OnAgentStart failed: %v", err)
	}

	if m.helperCount != 0 {
		t.Errorf("Expected helperCount=0 after OnAgentStart, got %d", m.helperCount)
	}
	if m.interfaceCount != 0 {
		t.Errorf("Expected interfaceCount=0 after OnAgentStart, got %d", m.interfaceCount)
	}
}

func TestSimplicityChecker_NonWriteEditTools(t *testing.T) {
	var warnings []SimplicityWarning
	var mu sync.Mutex

	m := NewSimplicityCheckerMiddleware(&SimplicityCheckerConfig{
		Enabled:            true,
		MaxHelperFunctions: 1,
		OnWarning: func(w SimplicityWarning) {
			mu.Lock()
			warnings = append(warnings, w)
			mu.Unlock()
		},
	})

	// 使用 Read 工具（不应触发检测）
	req := &ToolCallRequest{
		ToolName: "Read",
		ToolInput: map[string]interface{}{
			"file_path": "test.go",
		},
	}

	handler := func(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
		return &ToolCallResponse{Result: "func helper1() {}\nfunc helper2() {}"}, nil
	}

	_, err := m.WrapToolCall(context.Background(), req, handler)
	if err != nil {
		t.Fatalf("WrapToolCall failed: %v", err)
	}

	// Read 工具不应触发警告
	mu.Lock()
	defer mu.Unlock()
	if len(warnings) > 0 {
		t.Errorf("Expected no warnings for Read tool, got %d", len(warnings))
	}
}

func TestSimplicityChecker_OverEngineering(t *testing.T) {
	var warnings []SimplicityWarning
	var mu sync.Mutex

	m := NewSimplicityCheckerMiddleware(&SimplicityCheckerConfig{
		Enabled: true,
		OnWarning: func(w SimplicityWarning) {
			mu.Lock()
			warnings = append(warnings, w)
			mu.Unlock()
		},
	})

	// 模拟过度工程代码
	code := `
type Config struct {
    FeatureFlag bool
    Experimental bool
    ConfigOption string
}

func WithOption1() {}
func WithOption2() {}
func WithOption3() {}
func WithOption4() {}
func Configure() {}
`

	req := &ToolCallRequest{
		ToolName: "Write",
		ToolInput: map[string]interface{}{
			"file_path": "config.go",
			"content":   code,
		},
	}

	handler := func(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
		return &ToolCallResponse{Result: "ok"}, nil
	}

	_, err := m.WrapToolCall(context.Background(), req, handler)
	if err != nil {
		t.Fatalf("WrapToolCall failed: %v", err)
	}

	// 验证警告
	mu.Lock()
	defer mu.Unlock()
	foundOverEngineeringWarning := false
	for _, w := range warnings {
		if w.Type == WarningTypeOverEngineering {
			foundOverEngineeringWarning = true
			break
		}
	}
	if !foundOverEngineeringWarning {
		t.Error("Expected WarningTypeOverEngineering warning")
	}
}
