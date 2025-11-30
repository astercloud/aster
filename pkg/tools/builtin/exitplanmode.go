package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/astercloud/aster/pkg/tools"
)

// ExitPlanModeTool 规划模式退出工具
// 读取计划文件内容并请求用户审批
type ExitPlanModeTool struct {
	planFileManager *PlanFileManager
}

// PlanRecord 计划记录
type PlanRecord struct {
	ID                   string         `json:"id"`
	Content              string         `json:"content"`
	FilePath             string         `json:"file_path,omitempty"`
	EstimatedDuration    string         `json:"estimated_duration,omitempty"`
	Dependencies         []string       `json:"dependencies,omitempty"`
	Risks                []string       `json:"risks,omitempty"`
	SuccessCriteria      []string       `json:"success_criteria,omitempty"`
	ConfirmationRequired bool           `json:"confirmation_required"`
	Status               string         `json:"status"` // "pending_approval", "approved", "rejected", "completed"
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	ApprovedAt           *time.Time     `json:"approved_at,omitempty"`
	AgentID              string         `json:"agent_id"`
	SessionID            string         `json:"session_id"`
	Metadata             map[string]any `json:"metadata,omitempty"`
}

// NewExitPlanModeTool 创建ExitPlanMode工具
func NewExitPlanModeTool(config map[string]any) (tools.Tool, error) {
	basePath := ".aster/plans"
	if bp, ok := config["base_path"].(string); ok && bp != "" {
		basePath = bp
	}

	return &ExitPlanModeTool{
		planFileManager: NewPlanFileManager(basePath),
	}, nil
}

func (t *ExitPlanModeTool) Name() string {
	return "ExitPlanMode"
}

func (t *ExitPlanModeTool) Description() string {
	return "完成规划模式，读取计划文件内容并请求用户审批"
}

func (t *ExitPlanModeTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"plan_file_path": map[string]any{
				"type":        "string",
				"description": "计划文件的路径（由 EnterPlanMode 返回），如果不提供则自动查找最新的计划文件",
			},
		},
		"required": []string{},
	}
}

func (t *ExitPlanModeTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	start := time.Now()

	planFilePath := GetStringParam(input, "plan_file_path", "")

	// 如果没有提供计划文件路径，查找最新的计划文件
	if planFilePath == "" {
		plans, err := t.planFileManager.List()
		if err != nil {
			return NewClaudeErrorResponse(fmt.Errorf("failed to list plan files: %w", err)), nil
		}

		if len(plans) == 0 {
			return NewClaudeErrorResponse(
				fmt.Errorf("no plan files found"),
				"Please create a plan using EnterPlanMode first",
				"Plan files should be in .aster/plans/ directory",
			), nil
		}

		// 使用最新的计划文件（按修改时间排序）
		latestPlan := plans[len(plans)-1]
		planFilePath = latestPlan.Path
	}

	// 检查计划文件是否存在
	if !t.planFileManager.Exists(planFilePath) {
		return NewClaudeErrorResponse(
			fmt.Errorf("plan file not found: %s", planFilePath),
			"The specified plan file does not exist",
			"Check the path returned by EnterPlanMode",
		), nil
	}

	// 读取计划文件内容
	planContent, err := t.planFileManager.Load(planFilePath)
	if err != nil {
		return NewClaudeErrorResponse(fmt.Errorf("failed to read plan file: %w", err)), nil
	}

	// 从文件路径提取计划 ID
	planID := t.planFileManager.GenerateID()

	// 创建计划记录
	planRecord := &PlanRecord{
		ID:                   planID,
		Content:              planContent,
		FilePath:             planFilePath,
		ConfirmationRequired: true,
		Status:               "pending_approval",
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
		Metadata: map[string]any{
			"exit_plan_mode_call": true,
			"plan_file_path":      planFilePath,
		},
	}

	// 获取全局计划管理器并存储
	planManager := GetGlobalPlanManager()
	if err := planManager.StorePlan(planRecord); err != nil {
		// 不阻断流程，只记录警告
		fmt.Printf("[ExitPlanMode] Warning: failed to store plan record: %v\n", err)
	}

	duration := time.Since(start)

	// 构建响应
	response := map[string]any{
		"ok":                    true,
		"plan_id":               planID,
		"plan_file_path":        planFilePath,
		"plan_content":          planContent,
		"status":                "pending_approval",
		"confirmation_required": true,
		"duration_ms":           duration.Milliseconds(),
		"message":               "Plan is ready for user review. The user will see the plan content and can approve or request changes.",
		"next_steps": []string{
			"User needs to review the plan",
			"After approval, implementation can begin",
			"User can request modifications if needed",
		},
	}

	return response, nil
}

func (t *ExitPlanModeTool) Prompt() string {
	return `完成规划模式，读取计划文件内容并请求用户审批。

## 使用时机

当你在 Plan Mode 中完成了计划编写后，使用此工具：
- 你已经通过 EnterPlanMode 进入规划模式
- 你已经将计划写入到指定的计划文件中
- 计划内容完整，可以提交给用户审批

## 工作原理

此工具会：
1. 读取你写入的计划文件内容
2. 将计划内容展示给用户
3. 等待用户审批后才能开始实施

## 参数

- plan_file_path: 可选，计划文件路径（由 EnterPlanMode 返回）
  - 如果不提供，将自动查找最新的计划文件

## 重要说明

- 此工具不接受计划内容作为参数
- 计划应该已经写入到 .aster/plans/ 目录下的 markdown 文件中
- 用户必须审批后才能开始实施
- 在用户审批前，你不能进行任何代码修改

## 示例

调用时通常不需要参数：
{}

或指定计划文件路径：
{
  "plan_file_path": ".aster/plans/sunny-singing-nygaard.md"
}`
}
