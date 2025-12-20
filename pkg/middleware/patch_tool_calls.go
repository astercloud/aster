package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/astercloud/aster/pkg/logging"
)

var ptcLog = logging.ForComponent("PatchToolCallsMiddleware")

// PatchToolCallsMiddleware 工具调用错误恢复中间件
// 功能:
// 1. 捕获工具调用异常,防止崩溃
// 2. 提供友好的错误响应
// 3. 记录失败的工具调用供调试
type PatchToolCallsMiddleware struct {
	*BaseMiddleware

	enableLogging  bool
	failedCalls    []FailedToolCall
	maxFailedCalls int
	provideHints   bool
}

// FailedToolCall 失败的工具调用记录
type FailedToolCall struct {
	Timestamp  time.Time
	ToolName   string
	ToolCallID string
	Input      map[string]any
	Error      string
}

// PatchToolCallsMiddlewareConfig 配置
type PatchToolCallsMiddlewareConfig struct {
	EnableLogging  bool // 是否记录失败调用
	MaxFailedCalls int  // 最多保留多少条失败记录
	ProvideHints   bool // 是否在错误响应中提供修复提示
}

// NewPatchToolCallsMiddleware 创建中间件
func NewPatchToolCallsMiddleware(config *PatchToolCallsMiddlewareConfig) *PatchToolCallsMiddleware {
	if config == nil {
		config = &PatchToolCallsMiddlewareConfig{
			EnableLogging:  true,
			MaxFailedCalls: 100,
			ProvideHints:   true,
		}
	}

	m := &PatchToolCallsMiddleware{
		BaseMiddleware: NewBaseMiddleware("patch_tool_calls", 50),
		enableLogging:  config.EnableLogging,
		failedCalls:    make([]FailedToolCall, 0, config.MaxFailedCalls),
		maxFailedCalls: config.MaxFailedCalls,
		provideHints:   config.ProvideHints,
	}

	ptcLog.Info(context.Background(), "initialized", map[string]any{"logging": config.EnableLogging, "hints": config.ProvideHints})
	return m
}

// WrapToolCall 包装工具调用,捕获异常
func (m *PatchToolCallsMiddleware) WrapToolCall(ctx context.Context, req *ToolCallRequest, handler ToolCallHandler) (*ToolCallResponse, error) {
	// 调用处理器,捕获 panic
	var resp *ToolCallResponse
	var err error

	func() {
		defer func() {
			if r := recover(); r != nil {
				// 捕获 panic
				panicMsg := fmt.Sprintf("%v", r)
				ptcLog.Error(ctx, "tool panicked", map[string]any{"tool": req.ToolName, "panic": r})

				// 记录失败
				if m.enableLogging {
					m.recordFailedCall(req, panicMsg)
				}

				// 构建友好的错误响应
				resp = &ToolCallResponse{
					Result: m.buildErrorResponse(req.ToolName, panicMsg, "panic"),
				}
				err = nil // 不返回 error,而是返回错误响应
			}
		}()

		resp, err = handler(ctx, req)
	}()

	// 如果处理器返回了 error
	if err != nil {
		ptcLog.Warn(ctx, "tool returned error", map[string]any{"tool": req.ToolName, "error": err.Error()})

		// 记录失败
		if m.enableLogging {
			m.recordFailedCall(req, err.Error())
		}

		// 将 error 转换为友好的响应
		return &ToolCallResponse{
			Result: m.buildErrorResponse(req.ToolName, err.Error(), "error"),
		}, nil // 返回 nil error,让 Agent 可以继续
	}

	// 检查响应是否为 nil
	if resp == nil {
		ptcLog.Warn(ctx, "tool returned nil response", map[string]any{"tool": req.ToolName})

		// 记录失败
		if m.enableLogging {
			m.recordFailedCall(req, "nil response")
		}

		return &ToolCallResponse{
			Result: m.buildErrorResponse(req.ToolName, "tool returned nil response", "nil_response"),
		}, nil
	}

	// 正常响应
	return resp, nil
}

// recordFailedCall 记录失败的工具调用
func (m *PatchToolCallsMiddleware) recordFailedCall(req *ToolCallRequest, errorMsg string) {
	failedCall := FailedToolCall{
		Timestamp:  time.Now(),
		ToolName:   req.ToolName,
		ToolCallID: req.ToolCallID,
		Input:      req.ToolInput,
		Error:      errorMsg,
	}

	// 限制记录数量
	if len(m.failedCalls) >= m.maxFailedCalls && len(m.failedCalls) > 0 {
		// 删除最旧的记录
		m.failedCalls = m.failedCalls[1:]
	}

	m.failedCalls = append(m.failedCalls, failedCall)
}

// buildErrorResponse 构建友好的错误响应
func (m *PatchToolCallsMiddleware) buildErrorResponse(toolName, errorMsg, errorType string) map[string]any {
	response := map[string]any{
		"ok":         false,
		"error":      errorMsg,
		"error_type": errorType,
		"tool_name":  toolName,
		"message":    fmt.Sprintf("Tool '%s' failed to execute: %s", toolName, errorMsg),
	}

	// 提供修复提示
	if m.provideHints {
		hints := m.generateHints(toolName, errorType, errorMsg)
		if len(hints) > 0 {
			response["hints"] = hints
		}
	}

	return response
}

// generateHints 生成错误修复提示
func (m *PatchToolCallsMiddleware) generateHints(toolName, errorType, errorMsg string) []string {
	hints := []string{}

	switch errorType {
	case "panic":
		hints = append(hints,
			"工具执行发生严重错误(panic),可能原因:",
			"1. 输入参数类型不匹配",
			"2. 访问了 nil 指针",
			"3. 数组越界",
			"建议: 检查输入参数是否符合工具要求",
		)

	case "error":
		hints = append(hints,
			"工具执行返回错误,可能原因:",
			"1. 参数值不合法",
			"2. 依赖的资源不可用",
			"3. 权限不足",
			"建议: 查看错误信息调整参数或环境",
		)

	case "nil_response":
		hints = append(hints,
			"工具返回了空响应,可能原因:",
			"1. 工具实现存在缺陷",
			"2. 请求的资源不存在",
			"建议: 检查请求是否有效",
		)
	}

	return hints
}

// GetFailedCalls 获取失败的工具调用记录
func (m *PatchToolCallsMiddleware) GetFailedCalls() []FailedToolCall {
	return m.failedCalls
}

// ClearFailedCalls 清空失败记录
func (m *PatchToolCallsMiddleware) ClearFailedCalls() {
	m.failedCalls = make([]FailedToolCall, 0, m.maxFailedCalls)
	ptcLog.Info(context.Background(), "cleared failed calls history", nil)
}

// GetFailedCallCount 获取失败次数
func (m *PatchToolCallsMiddleware) GetFailedCallCount() int {
	return len(m.failedCalls)
}

// GetFailedCallsByTool 获取特定工具的失败记录
func (m *PatchToolCallsMiddleware) GetFailedCallsByTool(toolName string) []FailedToolCall {
	result := make([]FailedToolCall, 0)
	for _, call := range m.failedCalls {
		if call.ToolName == toolName {
			result = append(result, call)
		}
	}
	return result
}
