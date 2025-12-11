package builtin

import (
	"context"
	"fmt"
	"strings"
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
	basePath := ".plans" // 默认使用相对路径，会在工作目录下创建
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

	// 获取工作目录，动态设置计划文件存储路径
	// 这样计划文件会存储在 {workDir}/.plans/ 下，支持按项目隔离
	planManager := t.planFileManager
	if tc != nil && tc.Sandbox != nil {
		workDir := tc.Sandbox.WorkDir()
		if workDir != "" {
			// 创建新的 PlanFileManager，使用工作目录下的 .plans 子目录
			planManager = NewPlanFileManagerWithProject(workDir+"/.plans", "")
		}
	}

	// 如果没有提供计划文件路径，查找最新的计划文件
	if planFilePath == "" {
		plans, err := planManager.List()
		if err != nil {
			return NewClaudeErrorResponse(fmt.Errorf("failed to list plan files: %w", err)), nil
		}

		if len(plans) == 0 {
			return NewClaudeErrorResponse(
				fmt.Errorf("no plan files found"),
				"Please create a plan using EnterPlanMode first",
				"Plan files should be in .plans/ directory under your workspace",
			), nil
		}

		// 使用最新的计划文件（按修改时间排序）
		latestPlan := plans[len(plans)-1]
		planFilePath = latestPlan.Path
	} else {
		// 如果提供了路径，需要处理路径格式
		// AI 可能传入 ".plans/xxx.md" 或 "xxx.md" 或完整路径
		// planManager 的 basePath 已经是 {workDir}/.plans/，所以需要去掉前缀

		// 提取文件名（去掉所有目录前缀）
		fileName := planFilePath
		if idx := strings.LastIndex(planFilePath, "/"); idx >= 0 {
			fileName = planFilePath[idx+1:]
		}
		// 如果文件名不以 .md 结尾，添加后缀
		if !strings.HasSuffix(fileName, ".md") {
			fileName = fileName + ".md"
		}
		// 构建完整路径
		planFilePath = planManager.GetBasePath() + "/" + fileName
	}

	// 检查计划文件是否存在，带重试逻辑（文件可能刚写入还未同步）
	maxRetries := 3
	retryDelay := 500 * time.Millisecond
	var planContent string

	for i := 0; i < maxRetries; i++ {
		if !planManager.Exists(planFilePath) {
			if i < maxRetries-1 {
				time.Sleep(retryDelay)
				continue
			}
			return NewClaudeErrorResponse(
				fmt.Errorf("plan file not found: %s", planFilePath),
				"The specified plan file does not exist",
				"Check the path returned by EnterPlanMode",
			), nil
		}

		// 读取计划文件内容
		var err error
		planContent, err = planManager.Load(planFilePath)
		if err != nil {
			if i < maxRetries-1 {
				time.Sleep(retryDelay)
				continue
			}
			return NewClaudeErrorResponse(fmt.Errorf("failed to read plan file: %w", err)), nil
		}

		// 检查内容是否为空（可能文件刚创建还没写入内容）
		if strings.TrimSpace(planContent) == "" {
			if i < maxRetries-1 {
				time.Sleep(retryDelay)
				continue
			}
			return NewClaudeErrorResponse(
				fmt.Errorf("plan file is empty: %s", planFilePath),
				"The plan file exists but has no content",
				"Please write your plan content to the file before calling ExitPlanMode",
			), nil
		}

		// 成功读取到内容，跳出循环
		break
	}

	// 从文件路径提取计划 ID
	planID := planManager.GenerateID()

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
	globalPlanMgr := GetGlobalPlanManager()
	if err := globalPlanMgr.StorePlan(planRecord); err != nil {
		// 不阻断流程，只记录警告
		fmt.Printf("[ExitPlanMode] Warning: failed to store plan record: %v\n", err)
	}

	// 退出 Agent 级别的 Plan 模式约束
	// 注意：这里直接退出，实际生产环境可能需要等待用户批准
	if tc != nil && tc.Services != nil {
		if pmm, ok := tc.Services["plan_mode_manager"].(PlanModeManagerInterface); ok {
			pmm.ExitPlanMode()
		}
	}

	duration := time.Since(start)

	// 计算相对路径（只显示 .plans/xxx.md 部分，不暴露服务器绝对路径）
	relativePath := planFilePath
	if idx := strings.Index(planFilePath, ".plans/"); idx >= 0 {
		relativePath = planFilePath[idx:]
	} else if idx := strings.LastIndex(planFilePath, "/"); idx >= 0 {
		// 如果没有 .plans/，只取文件名
		relativePath = ".plans/" + planFilePath[idx+1:]
	}

	// 构建响应
	response := map[string]any{
		"ok":                    true,
		"plan_id":               planID,
		"plan_file_path":        relativePath,
		"plan_content":          planContent,
		"status":                "pending_approval",
		"confirmation_required": true,
		"duration_ms":           duration.Milliseconds(),
		"plan_mode_exited":      true,
		"message":               "计划已准备就绪，等待用户审批。用户可以批准、请求修改或拒绝。",
		"next_steps": []string{
			"用户审核计划内容",
			"批准后开始实施",
			"可请求修改或拒绝",
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
