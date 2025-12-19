package builtin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/astercloud/aster/pkg/tools"
)

// ExitPlanModeTool 规划模式退出工具
// 支持直接传入计划内容或从文件读取
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
	basePath := ".plans"
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
	return "完成规划模式，提交计划内容并请求用户审批"
}

func (t *ExitPlanModeTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"plan": map[string]any{
				"type":        "string",
				"description": "计划内容（推荐）。直接提供完整的计划内容，无需先写入文件。",
			},
			"plan_file_path": map[string]any{
				"type":        "string",
				"description": "计划文件的路径（可选，已弃用）。仅在未提供 plan 参数时使用。",
			},
		},
		"required": []string{},
	}
}

func (t *ExitPlanModeTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	start := time.Now()

	// 优先使用直接传入的计划内容
	planContent := GetStringParam(input, "plan", "")
	planFilePath := GetStringParam(input, "plan_file_path", "")

	// 获取工作目录
	planManager := t.planFileManager
	if tc != nil && tc.Sandbox != nil {
		workDir := tc.Sandbox.WorkDir()
		if workDir != "" {
			planManager = NewPlanFileManagerWithProject(workDir+"/.plans", "")
		}
	}

	// 如果直接提供了计划内容，使用它
	if strings.TrimSpace(planContent) != "" {
		planFilePath = ".plans/direct-plan-" + planManager.GenerateID() + ".md"
	} else {
		// 否则从文件读取（兼容旧版本）
		if planFilePath == "" {
			plans, err := planManager.List()
			if err != nil {
				return NewClaudeErrorResponse(fmt.Errorf("failed to list plan files: %w", err)), nil
			}

			if len(plans) == 0 {
				return NewClaudeErrorResponse(
					errors.New("no plan content provided and no plan files found"),
					"Please provide the plan content directly using the 'plan' parameter",
				), nil
			}

			latestPlan := plans[len(plans)-1]
			planFilePath = latestPlan.Path
		} else {
			fileName := planFilePath
			if idx := strings.LastIndex(planFilePath, "/"); idx >= 0 {
				fileName = planFilePath[idx+1:]
			}
			if !strings.HasSuffix(fileName, ".md") {
				fileName = fileName + ".md"
			}
			planFilePath = planManager.GetBasePath() + "/" + fileName
		}

		// 从文件读取计划内容
		maxRetries := 3
		retryDelay := 500 * time.Millisecond

		for i := range maxRetries {
			if !planManager.Exists(planFilePath) {
				if i < maxRetries-1 {
					time.Sleep(retryDelay)
					continue
				}
				return NewClaudeErrorResponse(
					fmt.Errorf("plan file not found: %s", planFilePath),
					"Please provide the plan content directly using the 'plan' parameter",
				), nil
			}

			var err error
			planContent, err = planManager.Load(planFilePath)
			if err != nil {
				if i < maxRetries-1 {
					time.Sleep(retryDelay)
					continue
				}
				return NewClaudeErrorResponse(fmt.Errorf("failed to read plan file: %w", err)), nil
			}

			if strings.TrimSpace(planContent) == "" {
				if i < maxRetries-1 {
					time.Sleep(retryDelay)
					continue
				}
				return NewClaudeErrorResponse(
					fmt.Errorf("plan file is empty: %s", planFilePath),
					"Please provide the plan content directly using the 'plan' parameter",
				), nil
			}

			break
		}
	}

	planID := planManager.GenerateID()

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
			"direct_input":        strings.TrimSpace(GetStringParam(input, "plan", "")) != "",
		},
	}

	globalPlanMgr := GetGlobalPlanManager()
	if err := globalPlanMgr.StorePlan(planRecord); err != nil {
		fmt.Printf("[ExitPlanMode] Warning: failed to store plan record: %v\n", err)
	}

	if tc != nil && tc.Services != nil {
		if pmm, ok := tc.Services["plan_mode_manager"].(PlanModeManagerInterface); ok {
			pmm.ExitPlanMode()
		}
	}

	duration := time.Since(start)

	relativePath := planFilePath
	if idx := strings.Index(planFilePath, ".plans/"); idx >= 0 {
		relativePath = planFilePath[idx:]
	} else if idx := strings.LastIndex(planFilePath, "/"); idx >= 0 {
		relativePath = ".plans/" + planFilePath[idx+1:]
	}

	response := map[string]any{
		"ok":                    true,
		"plan_id":               planID,
		"plan_file_path":        relativePath,
		"plan_content":          planContent,
		"status":                "pending_approval",
		"confirmation_required": true,
		"duration_ms":           duration.Milliseconds(),
		"plan_mode_exited":      true,
		"message":               "计划已准备就绪，等待用户审批。",
		"next_steps": []string{
			"用户审核计划内容",
			"批准后开始实施",
			"可请求修改或拒绝",
		},
	}

	return response, nil
}

func (t *ExitPlanModeTool) Prompt() string {
	return `完成规划模式，提交计划内容并请求用户审批。

## 推荐用法

直接将计划内容作为参数传入：

{
  "plan": "# 执行计划\n\n## 任务概述\n...\n\n## 执行步骤\n1. ...\n2. ...\n\n## 预期产出\n- ..."
}

## 参数说明

- plan: （推荐）直接提供完整的计划内容
- plan_file_path: （已弃用）仅在未提供 plan 参数时使用

## 重要说明

- 用户必须审批后才能开始实施
- 在用户审批前，不能进行任何代码修改`
}
