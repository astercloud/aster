package middleware

import (
	"context"
	"errors"
	"testing"
)

// TestPatchToolCallsMiddleware_CatchPanic 测试捕获 panic
func TestPatchToolCallsMiddleware_CatchPanic(t *testing.T) {
	ctx := context.Background()

	middleware := NewPatchToolCallsMiddleware(&PatchToolCallsMiddlewareConfig{
		EnableLogging: true,
		ProvideHints:  true,
	})

	req := &ToolCallRequest{
		ToolCallID: "test-panic",
		ToolName:   "panic_tool",
		ToolInput:  map[string]any{},
	}

	// 模拟会 panic 的处理器
	handler := func(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
		panic("simulated panic")
	}

	resp, err := middleware.WrapToolCall(ctx, req, handler)

	// 应该捕获 panic 并返回友好响应
	if err != nil {
		t.Errorf("Expected nil error, got: %v", err)
	}

	if resp == nil {
		t.Fatal("Response should not be nil")
	}

	resultMap, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Result should be a map")
	}

	if resultMap["ok"].(bool) {
		t.Error("Expected ok=false for panicked tool")
	}

	if resultMap["error_type"] != "panic" {
		t.Errorf("Expected error_type='panic', got '%v'", resultMap["error_type"])
	}

	// 检查失败记录
	if middleware.GetFailedCallCount() != 1 {
		t.Errorf("Expected 1 failed call, got %d", middleware.GetFailedCallCount())
	}
}

// TestPatchToolCallsMiddleware_CatchError 测试捕获 error
func TestPatchToolCallsMiddleware_CatchError(t *testing.T) {
	ctx := context.Background()

	middleware := NewPatchToolCallsMiddleware(&PatchToolCallsMiddlewareConfig{
		EnableLogging: true,
		ProvideHints:  true,
	})

	req := &ToolCallRequest{
		ToolCallID: "test-error",
		ToolName:   "error_tool",
		ToolInput:  map[string]any{},
	}

	// 模拟返回 error 的处理器
	handler := func(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
		return nil, errors.New("simulated error")
	}

	resp, err := middleware.WrapToolCall(ctx, req, handler)

	// 应该将 error 转换为友好响应
	if err != nil {
		t.Errorf("Expected nil error (converted to response), got: %v", err)
	}

	if resp == nil {
		t.Fatal("Response should not be nil")
	}

	resultMap, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Result should be a map")
	}

	if resultMap["ok"].(bool) {
		t.Error("Expected ok=false for error tool")
	}

	if resultMap["error_type"] != "error" {
		t.Errorf("Expected error_type='error', got '%v'", resultMap["error_type"])
	}

	// 检查错误消息
	if resultMap["error"] != "simulated error" {
		t.Errorf("Expected error='simulated error', got '%v'", resultMap["error"])
	}

	// 检查提示
	hints, hasHints := resultMap["hints"]
	if !hasHints {
		t.Error("Expected hints in response")
	}
	if hintsList, ok := hints.([]string); ok {
		if len(hintsList) == 0 {
			t.Error("Expected non-empty hints")
		}
	}
}

// TestPatchToolCallsMiddleware_NilResponse 测试 nil 响应
func TestPatchToolCallsMiddleware_NilResponse(t *testing.T) {
	ctx := context.Background()

	middleware := NewPatchToolCallsMiddleware(nil) // 使用默认配置

	req := &ToolCallRequest{
		ToolCallID: "test-nil",
		ToolName:   "nil_tool",
		ToolInput:  map[string]any{},
	}

	// 模拟返回 nil 响应的处理器
	handler := func(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
		return nil, nil
	}

	resp, err := middleware.WrapToolCall(ctx, req, handler)

	if err != nil {
		t.Errorf("Expected nil error, got: %v", err)
	}

	if resp == nil {
		t.Fatal("Response should not be nil")
	}

	resultMap, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Result should be a map")
	}

	if resultMap["error_type"] != "nil_response" {
		t.Errorf("Expected error_type='nil_response', got '%v'", resultMap["error_type"])
	}
}

// TestPatchToolCallsMiddleware_SuccessfulCall 测试正常调用
func TestPatchToolCallsMiddleware_SuccessfulCall(t *testing.T) {
	ctx := context.Background()

	middleware := NewPatchToolCallsMiddleware(&PatchToolCallsMiddlewareConfig{
		EnableLogging: true,
	})

	req := &ToolCallRequest{
		ToolCallID: "test-success",
		ToolName:   "success_tool",
		ToolInput:  map[string]any{},
	}

	// 正常的处理器
	handler := func(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
		return &ToolCallResponse{
			Result: map[string]any{
				"ok":     true,
				"result": "success",
			},
		}, nil
	}

	resp, err := middleware.WrapToolCall(ctx, req, handler)

	if err != nil {
		t.Errorf("Expected nil error, got: %v", err)
	}

	if resp == nil {
		t.Fatal("Response should not be nil")
	}

	resultMap, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Result should be a map")
	}

	if !resultMap["ok"].(bool) {
		t.Error("Expected ok=true for successful call")
	}

	// 不应该记录成功的调用
	if middleware.GetFailedCallCount() != 0 {
		t.Errorf("Expected 0 failed calls, got %d", middleware.GetFailedCallCount())
	}
}

// TestPatchToolCallsMiddleware_FailedCallsTracking 测试失败调用追踪
func TestPatchToolCallsMiddleware_FailedCallsTracking(t *testing.T) {
	ctx := context.Background()

	middleware := NewPatchToolCallsMiddleware(&PatchToolCallsMiddlewareConfig{
		EnableLogging:  true,
		MaxFailedCalls: 5,
	})

	// 生成多个失败调用
	for range 10 {
		req := &ToolCallRequest{
			ToolCallID: "test",
			ToolName:   "fail_tool",
			ToolInput:  map[string]any{},
		}

		handler := func(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
			return nil, errors.New("test error")
		}

		if _, err := middleware.WrapToolCall(ctx, req, handler); err != nil {
			t.Errorf("WrapToolCall failed: %v", err)
		}
	}

	// 应该只保留最近的 5 条
	if middleware.GetFailedCallCount() != 5 {
		t.Errorf("Expected 5 failed calls (max limit), got %d", middleware.GetFailedCallCount())
	}

	// 测试按工具名称获取
	failedCalls := middleware.GetFailedCallsByTool("fail_tool")
	if len(failedCalls) != 5 {
		t.Errorf("Expected 5 failed calls for 'fail_tool', got %d", len(failedCalls))
	}

	// 测试清空
	middleware.ClearFailedCalls()
	if middleware.GetFailedCallCount() != 0 {
		t.Errorf("Expected 0 failed calls after clear, got %d", middleware.GetFailedCallCount())
	}
}

// TestPatchToolCallsMiddleware_WithoutHints 测试禁用提示
func TestPatchToolCallsMiddleware_WithoutHints(t *testing.T) {
	ctx := context.Background()

	middleware := NewPatchToolCallsMiddleware(&PatchToolCallsMiddlewareConfig{
		EnableLogging: false,
		ProvideHints:  false,
	})

	req := &ToolCallRequest{
		ToolCallID: "test",
		ToolName:   "test_tool",
		ToolInput:  map[string]any{},
	}

	handler := func(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
		panic("test panic")
	}

	resp, err := middleware.WrapToolCall(ctx, req, handler)
	if err != nil {
		t.Errorf("WrapToolCall failed: %v", err)
	}

	resultMap := resp.Result.(map[string]any)

	// 不应该有 hints
	if _, hasHints := resultMap["hints"]; hasHints {
		t.Error("Should not have hints when ProvideHints is false")
	}

	// 不应该记录失败
	if middleware.GetFailedCallCount() != 0 {
		t.Error("Should not record failed calls when EnableLogging is false")
	}
}

// TestPatchToolCallsMiddleware_FailedCallDetails 测试失败记录详情
func TestPatchToolCallsMiddleware_FailedCallDetails(t *testing.T) {
	ctx := context.Background()

	middleware := NewPatchToolCallsMiddleware(&PatchToolCallsMiddlewareConfig{
		EnableLogging: true,
	})

	req := &ToolCallRequest{
		ToolCallID: "call-123",
		ToolName:   "test_tool",
		ToolInput: map[string]any{
			"param1": "value1",
		},
	}

	handler := func(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
		return nil, errors.New("detailed error")
	}

	if _, err := middleware.WrapToolCall(ctx, req, handler); err != nil {
		t.Errorf("WrapToolCall failed: %v", err)
	}

	failedCalls := middleware.GetFailedCalls()
	if len(failedCalls) != 1 {
		t.Fatalf("Expected 1 failed call, got %d", len(failedCalls))
	}

	call := failedCalls[0]

	if call.ToolName != "test_tool" {
		t.Errorf("Expected ToolName='test_tool', got '%s'", call.ToolName)
	}

	if call.ToolCallID != "call-123" {
		t.Errorf("Expected ToolCallID='call-123', got '%s'", call.ToolCallID)
	}

	if call.Error != "detailed error" {
		t.Errorf("Expected Error='detailed error', got '%s'", call.Error)
	}

	if call.Input["param1"] != "value1" {
		t.Errorf("Expected Input param1='value1', got '%v'", call.Input["param1"])
	}

	if call.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}
