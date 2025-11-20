package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/astercloud/aster/pkg/tools"
)

// ExitPlanModeTool 规划模式退出工具
// 支持在规划模式完成后展示实施计划并请求用户确认
type ExitPlanModeTool struct{}

// PlanRecord 计划记录
type PlanRecord struct {
	ID                   string                 `json:"id"`
	Content              string                 `json:"content"`
	EstimatedDuration    string                 `json:"estimated_duration,omitempty"`
	Dependencies         []string               `json:"dependencies,omitempty"`
	Risks                []string               `json:"risks,omitempty"`
	SuccessCriteria      []string               `json:"success_criteria,omitempty"`
	ConfirmationRequired bool                   `json:"confirmation_required"`
	Status               string                 `json:"status"` // "pending_approval", "approved", "rejected", "completed"
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
	ApprovedAt           *time.Time             `json:"approved_at,omitempty"`
	AgentID              string                 `json:"agent_id"`
	SessionID            string                 `json:"session_id"`
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
}

// NewExitPlanModeTool 创建ExitPlanMode工具
func NewExitPlanModeTool(config map[string]interface{}) (tools.Tool, error) {
	return &ExitPlanModeTool{}, nil
}

func (t *ExitPlanModeTool) Name() string {
	return "ExitPlanMode"
}

func (t *ExitPlanModeTool) Description() string {
	return "在规划模式完成后展示实施计划并请求用户确认"
}

func (t *ExitPlanModeTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"plan": map[string]interface{}{
				"type":        "string",
				"description": "要展示给用户的实施计划，支持markdown格式",
			},
			"plan_id": map[string]interface{}{
				"type":        "string",
				"description": "计划的唯一标识符，用于跟踪",
			},
			"estimated_duration": map[string]interface{}{
				"type":        "string",
				"description": "预估的实施时间，如'2 hours', '3 days'",
			},
			"dependencies": map[string]interface{}{
				"type":        "array",
				"description": "计划的依赖项或前提条件",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
			"risks": map[string]interface{}{
				"type":        "array",
				"description": "潜在风险和缓解措施",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
			"success_criteria": map[string]interface{}{
				"type":        "array",
				"description": "成功标准",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
			"confirmation_required": map[string]interface{}{
				"type":        "boolean",
				"description": "是否需要用户确认才能开始实施，默认为true",
			},
		},
		"required": []string{"plan"},
	}
}

func (t *ExitPlanModeTool) Execute(ctx context.Context, input map[string]interface{}, tc *tools.ToolContext) (interface{}, error) {
	// 验证必需参数
	if err := ValidateRequired(input, []string{"plan"}); err != nil {
		return NewClaudeErrorResponse(err), nil
	}

	plan := GetStringParam(input, "plan", "")
	planID := GetStringParam(input, "plan_id", "")
	estimatedDuration := GetStringParam(input, "estimated_duration", "")
	confirmationRequired := GetBoolParam(input, "confirmation_required", true)

	dependencies := t.getStringSlice(input, "dependencies")
	risks := t.getStringSlice(input, "risks")
	successCriteria := t.getStringSlice(input, "success_criteria")

	if plan == "" {
		return NewClaudeErrorResponse(fmt.Errorf("plan cannot be empty")), nil
	}

	start := time.Now()

	// 生成计划ID（如果没有提供）
	if planID == "" {
		planID = fmt.Sprintf("plan_%d", time.Now().UnixNano())
	}

	// 创建计划记录
	planRecord := &PlanRecord{
		ID:                   planID,
		Content:              plan,
		EstimatedDuration:    estimatedDuration,
		Dependencies:         dependencies,
		Risks:                risks,
		SuccessCriteria:      successCriteria,
		ConfirmationRequired: confirmationRequired,
		Status:               "pending_approval",
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
		AgentID:              "agent_default",
		SessionID:            "session_default",
		Metadata: map[string]interface{}{
			"exit_plan_mode_call": true,
		},
	}

	// 获取全局计划管理器
	planManager := GetGlobalPlanManager()

	// 存储计划记录
	err := planManager.StorePlan(planRecord)
	if err != nil {
		return map[string]interface{}{
			"ok":          false,
			"error":       fmt.Sprintf("failed to store plan: %v", err),
			"plan_id":     planID,
			"duration_ms": time.Since(start).Milliseconds(),
		}, nil
	}

	duration := time.Since(start)

	// 构建响应
	response := map[string]interface{}{
		"ok":                    true,
		"plan_id":               planID,
		"plan":                  plan,
		"status":                "pending_approval",
		"confirmation_required": confirmationRequired,
		"created_at":            planRecord.CreatedAt,
		"updated_at":            planRecord.UpdatedAt,
		"agent_id":              planRecord.AgentID,
		"session_id":            planRecord.SessionID,
		"metadata":              planRecord.Metadata,
		"duration_ms":           duration.Milliseconds(),
		"storage":               "persistent",
		"storage_backend":       "FilePlanManager",
	}

	// 添加可选字段
	if estimatedDuration != "" {
		response["estimated_duration"] = estimatedDuration
	}

	// 总是包含数组字段，即使为空
	response["dependencies"] = dependencies
	response["risks"] = risks
	response["success_criteria"] = successCriteria

	// 添加计划统计
	response["dependencies_count"] = len(dependencies)
	response["risks_count"] = len(risks)
	response["success_criteria_count"] = len(successCriteria)

	// 添加下一步操作指导
	if confirmationRequired {
		response["next_steps"] = []string{
			"用户需要审阅并确认计划",
			"确认后可以开始实施",
			"可以修改计划或提出建议",
		}
	} else {
		response["next_steps"] = []string{
			"计划已准备好，可以立即开始实施",
			"按照计划步骤逐步执行",
			"定期报告进度",
		}
		// 自动将计划状态设为已批准
		planRecord.Status = "approved"
		now := time.Now()
		planRecord.ApprovedAt = &now
		planRecord.UpdatedAt = now

		// 更新存储的计划记录
		if err := planManager.StorePlan(planRecord); err != nil {
			// 记录错误但不中断响应
			response["approval_warning"] = fmt.Sprintf("plan saved but approval update failed: %v", err)
		}
		response["status"] = "approved"
		response["approved_at"] = now.Unix()
	}

	return response, nil
}

// getStringSlice 获取字符串数组参数
func (t *ExitPlanModeTool) getStringSlice(input map[string]interface{}, key string) []string {
	if value, exists := input[key]; exists {
		// 尝试直接转换为 []string
		if slice, ok := value.([]string); ok {
			return slice
		}
		// 尝试转换为 []interface{}
		if slice, ok := value.([]interface{}); ok {
			result := make([]string, len(slice))
			for i, item := range slice {
				if str, ok := item.(string); ok {
					result[i] = str
				}
			}
			return result
		}
	}
	return []string{}
}

func (t *ExitPlanModeTool) Prompt() string {
	return `在规划模式完成后展示实施计划并请求用户确认。

功能特性：
- 支持详细的实施计划展示
- 计划状态跟踪和管理
- 依赖项和风险评估
- 成功标准定义
- 自动化确认流程

使用指南：
- plan: 必需参数，要展示的实施计划（支持markdown）
- plan_id: 可选参数，计划的唯一标识符
- estimated_duration: 可选参数，预估实施时间
- dependencies: 可选参数，计划的依赖项列表
- risks: 可选参数，潜在风险和缓解措施
- success_criteria: 可选参数，成功标准列表
- confirmation_required: 可选参数，是否需要用户确认

计划内容建议：
- 详细的实施步骤
- 所需的资源清单
- 时间线和里程碑
- 风险评估和应对策略
- 成功标准和验收条件

状态流程：
- pending_approval: 等待用户确认
- approved: 计划已批准，可以开始实施
- rejected: 计划被拒绝，需要修改
- completed: 计划已完成

注意事项：
- 使用持久化存储系统，数据安全可靠
- 支持计划的版本管理和历史记录
- 可集成项目管理工具
- 自动处理计划状态和时间戳

存储特性：
- 基于文件系统的JSON格式存储
- 自动备份和恢复机制
- 支持多计划管理
- 计划状态跟踪和更新
- 集成全局存储管理器`
}
