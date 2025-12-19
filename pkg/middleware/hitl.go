package middleware

import (
	"context"
	"fmt"

	"github.com/astercloud/aster/pkg/logging"
)

// DecisionType 审核决策类型
type DecisionType string

const (
	DecisionApprove DecisionType = "approve" // 批准执行
	DecisionReject  DecisionType = "reject"  // 拒绝执行
	DecisionEdit    DecisionType = "edit"    // 编辑参数后执行
)

// InterruptConfig 工具审核配置
type InterruptConfig struct {
	Enabled          bool           // 是否启用审核
	AllowedDecisions []DecisionType // 允许的决策类型(默认: approve, edit, reject)
	Message          string         // 自定义审核提示信息
}

// ActionRequest 待审核的操作请求
type ActionRequest struct {
	ToolCallID string         // 工具调用ID
	ToolName   string         // 工具名称
	Input      map[string]any // 工具输入参数
	Message    string         // 审核提示信息
}

// Decision 人工决策
type Decision struct {
	Type        DecisionType   // 决策类型
	EditedInput map[string]any // 编辑后的参数(仅 type=edit 时有效)
	Reason      string         // 决策理由(可选)
}

// ReviewRequest 审核请求
type ReviewRequest struct {
	ActionRequests []ActionRequest   // 待审核的操作列表
	ReviewConfigs  []InterruptConfig // 每个操作的审核配置
}

// ApprovalHandler 人工审核处理器
// 用于获取人工决策
type ApprovalHandler func(ctx context.Context, request *ReviewRequest) ([]Decision, error)

// HumanInTheLoopMiddlewareConfig HITL 中间件配置
type HumanInTheLoopMiddlewareConfig struct {
	// InterruptOn 配置哪些工具需要审核
	// key: 工具名称, value: 审核配置
	// 支持三种格式:
	// 1. true: 启用默认审核配置
	// 2. false: 禁用审核
	// 3. InterruptConfig: 自定义审核配置
	InterruptOn map[string]any

	// ApprovalHandler 人工审核处理器
	// 如果为 nil, 默认自动批准所有请求
	ApprovalHandler ApprovalHandler

	// DefaultAllowedDecisions 默认允许的决策类型
	DefaultAllowedDecisions []DecisionType
}

// HumanInTheLoopMiddleware 人工审核中间件
// 功能:
// 1. 拦截敏感工具调用
// 2. 请求人工批准/拒绝/编辑
// 3. 支持批量审核
// 4. 灵活的审核配置
type HumanInTheLoopMiddleware struct {
	*BaseMiddleware

	interruptConfigs        map[string]*InterruptConfig
	approvalHandler         ApprovalHandler
	defaultAllowedDecisions []DecisionType
}

// NewHumanInTheLoopMiddleware 创建 HITL 中间件
func NewHumanInTheLoopMiddleware(config *HumanInTheLoopMiddlewareConfig) (*HumanInTheLoopMiddleware, error) {
	if config == nil {
		config = &HumanInTheLoopMiddlewareConfig{}
	}

	// 默认允许的决策
	defaultAllowedDecisions := config.DefaultAllowedDecisions
	if len(defaultAllowedDecisions) == 0 {
		defaultAllowedDecisions = []DecisionType{DecisionApprove, DecisionEdit, DecisionReject}
	}

	m := &HumanInTheLoopMiddleware{
		BaseMiddleware:          NewBaseMiddleware("hitl", 150),
		interruptConfigs:        make(map[string]*InterruptConfig),
		approvalHandler:         config.ApprovalHandler,
		defaultAllowedDecisions: defaultAllowedDecisions,
	}

	// 解析 InterruptOn 配置
	if config.InterruptOn != nil {
		for toolName, cfg := range config.InterruptOn {
			interruptCfg := m.parseInterruptConfig(toolName, cfg)
			if interruptCfg != nil && interruptCfg.Enabled {
				m.interruptConfigs[toolName] = interruptCfg
			}
		}
	}

	hitlLog.Info(context.Background(), "initialized", map[string]any{"tools_count": len(m.interruptConfigs)})
	return m, nil
}

var hitlLog = logging.ForComponent("HumanInTheLoopMiddleware")

// parseInterruptConfig 解析审核配置
func (m *HumanInTheLoopMiddleware) parseInterruptConfig(toolName string, cfg any) *InterruptConfig {
	switch v := cfg.(type) {
	case bool:
		if !v {
			return nil // 禁用审核
		}
		// 启用默认审核配置
		return &InterruptConfig{
			Enabled:          true,
			AllowedDecisions: m.defaultAllowedDecisions,
			Message:          fmt.Sprintf("Tool '%s' requires approval before execution", toolName),
		}

	case map[string]any:
		// 自定义配置
		interruptCfg := &InterruptConfig{
			Enabled: true,
		}

		// 解析 allowed_decisions
		if decisions, ok := v["allowed_decisions"].([]any); ok {
			interruptCfg.AllowedDecisions = make([]DecisionType, 0, len(decisions))
			for _, d := range decisions {
				if ds, ok := d.(string); ok {
					interruptCfg.AllowedDecisions = append(interruptCfg.AllowedDecisions, DecisionType(ds))
				}
			}
		}
		if len(interruptCfg.AllowedDecisions) == 0 {
			interruptCfg.AllowedDecisions = m.defaultAllowedDecisions
		}

		// 解析 message
		if msg, ok := v["message"].(string); ok {
			interruptCfg.Message = msg
		} else {
			interruptCfg.Message = fmt.Sprintf("Tool '%s' requires approval before execution", toolName)
		}

		return interruptCfg

	case *InterruptConfig:
		return v

	default:
		hitlLog.Warn(context.Background(), "invalid interrupt config", map[string]any{"tool": toolName, "type": fmt.Sprintf("%T", cfg)})
		return nil
	}
}

// WrapToolCall 拦截工具调用,请求人工审核
func (m *HumanInTheLoopMiddleware) WrapToolCall(ctx context.Context, req *ToolCallRequest, handler ToolCallHandler) (*ToolCallResponse, error) {
	// 检查是否需要审核
	interruptCfg, needsApproval := m.interruptConfigs[req.ToolName]
	if !needsApproval {
		// 不需要审核,直接执行
		return handler(ctx, req)
	}

	hitlLog.Info(ctx, "tool requires approval", map[string]any{"tool": req.ToolName})

	// 构建审核请求
	reviewRequest := &ReviewRequest{
		ActionRequests: []ActionRequest{
			{
				ToolCallID: req.ToolCallID,
				ToolName:   req.ToolName,
				Input:      req.ToolInput,
				Message:    interruptCfg.Message,
			},
		},
		ReviewConfigs: []InterruptConfig{*interruptCfg},
	}

	// 获取人工决策
	decisions, err := m.getApproval(ctx, reviewRequest)
	if err != nil {
		return &ToolCallResponse{
			Result: map[string]any{
				"ok":    false,
				"error": fmt.Sprintf("approval request failed: %v", err),
			},
		}, nil
	}

	if len(decisions) == 0 {
		return &ToolCallResponse{
			Result: map[string]any{
				"ok":    false,
				"error": "no decision received",
			},
		}, nil
	}

	decision := decisions[0]

	// 处理决策
	switch decision.Type {
	case DecisionApprove:
		hitlLog.Info(ctx, "tool approved", map[string]any{"tool": req.ToolName})
		return handler(ctx, req)

	case DecisionEdit:
		hitlLog.Info(ctx, "tool approved with edited input", map[string]any{"tool": req.ToolName})
		// 使用编辑后的参数
		editedReq := *req
		editedReq.ToolInput = decision.EditedInput
		return handler(ctx, &editedReq)

	case DecisionReject:
		hitlLog.Info(ctx, "tool rejected", map[string]any{"tool": req.ToolName, "reason": decision.Reason})
		return &ToolCallResponse{
			Result: map[string]any{
				"ok":       false,
				"rejected": true,
				"reason":   decision.Reason,
				"message":  "Tool execution rejected by human reviewer: " + decision.Reason,
			},
		}, nil

	default:
		return &ToolCallResponse{
			Result: map[string]any{
				"ok":    false,
				"error": fmt.Sprintf("unknown decision type: %s", decision.Type),
			},
		}, nil
	}
}

// getApproval 获取人工审核决策
func (m *HumanInTheLoopMiddleware) getApproval(ctx context.Context, request *ReviewRequest) ([]Decision, error) {
	if m.approvalHandler != nil {
		return m.approvalHandler(ctx, request)
	}

	// 默认处理器: 自动批准所有请求
	hitlLog.Warn(ctx, "no approval handler configured, auto-approving", map[string]any{"requests": len(request.ActionRequests)})
	decisions := make([]Decision, len(request.ActionRequests))
	for i := range decisions {
		decisions[i] = Decision{
			Type:   DecisionApprove,
			Reason: "Auto-approved (no approval handler configured)",
		}
	}
	return decisions, nil
}

// SetApprovalHandler 设置审核处理器
func (m *HumanInTheLoopMiddleware) SetApprovalHandler(handler ApprovalHandler) {
	m.approvalHandler = handler
	hitlLog.Info(context.Background(), "approval handler updated", nil)
}

// GetInterruptConfig 获取工具的审核配置
func (m *HumanInTheLoopMiddleware) GetInterruptConfig(toolName string) (*InterruptConfig, bool) {
	cfg, exists := m.interruptConfigs[toolName]
	return cfg, exists
}

// IsToolInterruptible 检查工具是否需要审核
func (m *HumanInTheLoopMiddleware) IsToolInterruptible(toolName string) bool {
	_, exists := m.interruptConfigs[toolName]
	return exists
}

// ListInterruptibleTools 列出所有需要审核的工具
func (m *HumanInTheLoopMiddleware) ListInterruptibleTools() []string {
	tools := make([]string, 0, len(m.interruptConfigs))
	for toolName := range m.interruptConfigs {
		tools = append(tools, toolName)
	}
	return tools
}

// HITL_SYSTEM_PROMPT HITL 系统提示词(可选)
const HITL_SYSTEM_PROMPT = `## Human-in-the-Loop (HITL)

某些敏感操作需要人工批准才能执行。当你调用这些工具时:

1. **系统会暂停执行**,等待人工审核
2. **人工审核员可以**:
   - 批准(approve): 执行操作
   - 拒绝(reject): 取消操作
   - 编辑(edit): 修改参数后执行

3. **如果操作被拒绝**:
   - 你会收到拒绝原因
   - 可以调整策略或尝试其他方法
   - 不要重复尝试被拒绝的操作

4. **最佳实践**:
   - 清楚解释为什么需要执行该操作
   - 提供足够的上下文帮助审核
   - 尊重人工审核决策`
