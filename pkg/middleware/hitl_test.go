package middleware

import (
	"context"
	"maps"
	"testing"

	"github.com/astercloud/aster/pkg/tools"
)

// mockTool 模拟工具
type mockTool struct {
	name string
}

func (t *mockTool) Name() string {
	return t.name
}

func (t *mockTool) Description() string {
	return "Mock tool for testing"
}

func (t *mockTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"param": map[string]any{
				"type": "string",
			},
		},
	}
}

func (t *mockTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	return map[string]any{
		"ok":     true,
		"result": "executed",
		"param":  input["param"],
	}, nil
}

func (t *mockTool) Prompt() string {
	return ""
}

// TestHumanInTheLoopMiddleware_BasicApproval 测试基本审核流程
func TestHumanInTheLoopMiddleware_BasicApproval(t *testing.T) {
	ctx := context.Background()

	// 创建中间件
	middleware, err := NewHumanInTheLoopMiddleware(&HumanInTheLoopMiddlewareConfig{
		InterruptOn: map[string]any{
			"sensitive_tool": true,  // 需要审核
			"safe_tool":      false, // 不需要审核
		},
		ApprovalHandler: func(ctx context.Context, request *ReviewRequest) ([]Decision, error) {
			// 自动批准
			decisions := make([]Decision, len(request.ActionRequests))
			for i := range decisions {
				decisions[i] = Decision{
					Type:   DecisionApprove,
					Reason: "Test approval",
				}
			}
			return decisions, nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// 测试1: sensitive_tool 需要审核,但被批准
	t.Run("Approve sensitive tool", func(t *testing.T) {
		req := &ToolCallRequest{
			ToolCallID: "test-1",
			ToolName:   "sensitive_tool",
			ToolInput: map[string]any{
				"param": "value1",
			},
			Tool: &mockTool{name: "sensitive_tool"},
		}

		handler := func(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
			// 执行工具
			result, _ := req.Tool.Execute(ctx, req.ToolInput, nil)
			return &ToolCallResponse{
				Result: result,
			}, nil
		}

		resp, err := middleware.WrapToolCall(ctx, req, handler)
		if err != nil {
			t.Fatalf("WrapToolCall failed: %v", err)
		}

		resultMap, ok := resp.Result.(map[string]any)
		if !ok {
			t.Fatal("Result is not a map")
		}

		if !resultMap["ok"].(bool) {
			t.Errorf("Expected ok=true, got ok=false")
		}

		if resultMap["result"] != "executed" {
			t.Errorf("Expected result='executed', got '%v'", resultMap["result"])
		}
	})

	// 测试2: safe_tool 不需要审核
	t.Run("Safe tool no approval needed", func(t *testing.T) {
		req := &ToolCallRequest{
			ToolCallID: "test-2",
			ToolName:   "safe_tool",
			ToolInput: map[string]any{
				"param": "value2",
			},
			Tool: &mockTool{name: "safe_tool"},
		}

		handler := func(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
			result, _ := req.Tool.Execute(ctx, req.ToolInput, nil)
			return &ToolCallResponse{
				Result: result,
			}, nil
		}

		resp, err := middleware.WrapToolCall(ctx, req, handler)
		if err != nil {
			t.Fatalf("WrapToolCall failed: %v", err)
		}

		resultMap, ok := resp.Result.(map[string]any)
		if !ok {
			t.Fatal("Result is not a map")
		}

		if !resultMap["ok"].(bool) {
			t.Errorf("Expected ok=true, got ok=false")
		}
	})
}

// TestHumanInTheLoopMiddleware_Reject 测试拒绝操作
func TestHumanInTheLoopMiddleware_Reject(t *testing.T) {
	ctx := context.Background()

	middleware, err := NewHumanInTheLoopMiddleware(&HumanInTheLoopMiddlewareConfig{
		InterruptOn: map[string]any{
			"dangerous_tool": map[string]any{
				"allowed_decisions": []any{"approve", "reject"},
			},
		},
		ApprovalHandler: func(ctx context.Context, request *ReviewRequest) ([]Decision, error) {
			// 拒绝操作
			return []Decision{
				{
					Type:   DecisionReject,
					Reason: "Too dangerous",
				},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	req := &ToolCallRequest{
		ToolCallID: "test-reject",
		ToolName:   "dangerous_tool",
		ToolInput: map[string]any{
			"action": "delete_all",
		},
		Tool: &mockTool{name: "dangerous_tool"},
	}

	handler := func(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
		t.Error("Handler should not be called when operation is rejected")
		return nil, nil
	}

	resp, err := middleware.WrapToolCall(ctx, req, handler)
	if err != nil {
		t.Fatalf("WrapToolCall failed: %v", err)
	}

	resultMap, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Result is not a map")
	}

	if resultMap["ok"].(bool) {
		t.Error("Expected ok=false for rejected operation")
	}

	if !resultMap["rejected"].(bool) {
		t.Error("Expected rejected=true")
	}

	if resultMap["reason"] != "Too dangerous" {
		t.Errorf("Expected reason='Too dangerous', got '%v'", resultMap["reason"])
	}
}

// TestHumanInTheLoopMiddleware_Edit 测试编辑参数
func TestHumanInTheLoopMiddleware_Edit(t *testing.T) {
	ctx := context.Background()

	middleware, err := NewHumanInTheLoopMiddleware(&HumanInTheLoopMiddlewareConfig{
		InterruptOn: map[string]any{
			"editable_tool": true,
		},
		ApprovalHandler: func(ctx context.Context, request *ReviewRequest) ([]Decision, error) {
			// 编辑参数
			editedInput := make(map[string]any)
			maps.Copy(editedInput, request.ActionRequests[0].Input)
			editedInput["param"] = "edited_value"

			return []Decision{
				{
					Type:        DecisionEdit,
					EditedInput: editedInput,
					Reason:      "Parameter adjusted",
				},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	req := &ToolCallRequest{
		ToolCallID: "test-edit",
		ToolName:   "editable_tool",
		ToolInput: map[string]any{
			"param": "original_value",
		},
		Tool: &mockTool{name: "editable_tool"},
	}

	handler := func(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
		result, _ := req.Tool.Execute(ctx, req.ToolInput, nil)
		return &ToolCallResponse{
			Result: result,
		}, nil
	}

	resp, err := middleware.WrapToolCall(ctx, req, handler)
	if err != nil {
		t.Fatalf("WrapToolCall failed: %v", err)
	}

	resultMap, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Result is not a map")
	}

	if !resultMap["ok"].(bool) {
		t.Error("Expected ok=true")
	}

	// 验证参数已被编辑
	if resultMap["param"] != "edited_value" {
		t.Errorf("Expected param='edited_value', got '%v'", resultMap["param"])
	}
}

// TestHumanInTheLoopMiddleware_DefaultApproval 测试默认自动批准
func TestHumanInTheLoopMiddleware_DefaultApproval(t *testing.T) {
	ctx := context.Background()

	// 不提供 ApprovalHandler, 应该自动批准
	middleware, err := NewHumanInTheLoopMiddleware(&HumanInTheLoopMiddlewareConfig{
		InterruptOn: map[string]any{
			"test_tool": true,
		},
		// ApprovalHandler: nil, // 默认自动批准
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	req := &ToolCallRequest{
		ToolCallID: "test-auto",
		ToolName:   "test_tool",
		ToolInput: map[string]any{
			"param": "value",
		},
		Tool: &mockTool{name: "test_tool"},
	}

	handler := func(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
		result, _ := req.Tool.Execute(ctx, req.ToolInput, nil)
		return &ToolCallResponse{
			Result: result,
		}, nil
	}

	resp, err := middleware.WrapToolCall(ctx, req, handler)
	if err != nil {
		t.Fatalf("WrapToolCall failed: %v", err)
	}

	resultMap, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Result is not a map")
	}

	if !resultMap["ok"].(bool) {
		t.Error("Expected auto-approval to succeed")
	}
}

// TestHumanInTheLoopMiddleware_InterruptConfig 测试审核配置解析
func TestHumanInTheLoopMiddleware_InterruptConfig(t *testing.T) {
	middleware, err := NewHumanInTheLoopMiddleware(&HumanInTheLoopMiddlewareConfig{
		InterruptOn: map[string]any{
			"tool1": true,  // 简单启用
			"tool2": false, // 禁用
			"tool3": map[string]any{ // 自定义配置
				"allowed_decisions": []any{"approve", "reject"},
				"message":           "Custom message",
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// 验证 tool1
	if !middleware.IsToolInterruptible("tool1") {
		t.Error("tool1 should be interruptible")
	}

	// 验证 tool2
	if middleware.IsToolInterruptible("tool2") {
		t.Error("tool2 should not be interruptible")
	}

	// 验证 tool3
	cfg, exists := middleware.GetInterruptConfig("tool3")
	if !exists {
		t.Fatal("tool3 config not found")
	}

	if len(cfg.AllowedDecisions) != 2 {
		t.Errorf("Expected 2 allowed decisions, got %d", len(cfg.AllowedDecisions))
	}

	if cfg.Message != "Custom message" {
		t.Errorf("Expected 'Custom message', got '%s'", cfg.Message)
	}

	// 验证 ListInterruptibleTools
	tools := middleware.ListInterruptibleTools()
	if len(tools) != 2 { // tool1 和 tool3
		t.Errorf("Expected 2 interruptible tools, got %d", len(tools))
	}
}

// TestHumanInTheLoopMiddleware_SetApprovalHandler 测试动态设置审核处理器
func TestHumanInTheLoopMiddleware_SetApprovalHandler(t *testing.T) {
	ctx := context.Background()

	middleware, err := NewHumanInTheLoopMiddleware(&HumanInTheLoopMiddlewareConfig{
		InterruptOn: map[string]any{
			"test_tool": true,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// 设置自定义处理器
	approvalCalled := false
	middleware.SetApprovalHandler(func(ctx context.Context, request *ReviewRequest) ([]Decision, error) {
		approvalCalled = true
		return []Decision{{Type: DecisionApprove}}, nil
	})

	req := &ToolCallRequest{
		ToolCallID: "test-set",
		ToolName:   "test_tool",
		ToolInput:  map[string]any{},
		Tool:       &mockTool{name: "test_tool"},
	}

	handler := func(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
		return &ToolCallResponse{
			Result: map[string]any{"ok": true},
		}, nil
	}

	_, err = middleware.WrapToolCall(ctx, req, handler)
	if err != nil {
		t.Fatalf("WrapToolCall failed: %v", err)
	}

	if !approvalCalled {
		t.Error("Custom approval handler was not called")
	}
}
