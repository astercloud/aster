package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/astercloud/aster/pkg/tools"
)

// EnterPlanModeTool 进入规划模式工具
// 用于复杂任务的规划阶段，在此模式下只允许只读操作和计划文件写入
type EnterPlanModeTool struct {
	planFileManager *PlanFileManager
}

// EnterPlanModeResult 进入规划模式的结果
type EnterPlanModeResult struct {
	OK           bool     `json:"ok"`
	PlanFilePath string   `json:"plan_file_path"`
	PlanID       string   `json:"plan_id"`
	AllowedTools []string `json:"allowed_tools"`
	Workflow     string   `json:"workflow"`
	Message      string   `json:"message"`
}

// NewEnterPlanModeTool 创建EnterPlanMode工具
func NewEnterPlanModeTool(config map[string]interface{}) (tools.Tool, error) {
	basePath := ".aster/plans"
	if bp, ok := config["base_path"].(string); ok && bp != "" {
		basePath = bp
	}

	return &EnterPlanModeTool{
		planFileManager: NewPlanFileManager(basePath),
	}, nil
}

func (t *EnterPlanModeTool) Name() string {
	return "EnterPlanMode"
}

func (t *EnterPlanModeTool) Description() string {
	return "进入规划模式，用于复杂任务的规划阶段。在此模式下只允许只读操作和计划文件写入。"
}

func (t *EnterPlanModeTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"reason": map[string]interface{}{
				"type":        "string",
				"description": "进入规划模式的原因说明",
			},
		},
		"required": []string{},
	}
}

func (t *EnterPlanModeTool) Execute(ctx context.Context, input map[string]interface{}, tc *tools.ToolContext) (interface{}, error) {
	reason := GetStringParam(input, "reason", "")

	// 确保计划目录存在
	if err := t.planFileManager.EnsureDir(); err != nil {
		return NewClaudeErrorResponse(fmt.Errorf("failed to create plans directory: %w", err)), nil
	}

	// 生成计划文件路径
	planPath := t.planFileManager.GeneratePath()
	planID := t.planFileManager.GenerateID()

	// 创建初始计划文件
	initialContent := fmt.Sprintf(`# Implementation Plan

> Plan ID: %s
> Created: %s
> Status: Planning

## Overview

[Describe the task and approach here]

## Steps

1. [Step 1]
2. [Step 2]
3. [Step 3]

## Critical Files

| File | Purpose |
|------|---------|
| | |

## Risks & Mitigations

-

## Success Criteria

-

---
*This plan file will be updated as planning progresses.*
`, planID, time.Now().Format(time.RFC3339))

	if err := t.planFileManager.Save(planPath, initialContent); err != nil {
		return NewClaudeErrorResponse(fmt.Errorf("failed to create plan file: %w", err)), nil
	}

	// 定义规划模式允许的工具
	allowedTools := []string{
		"Read",
		"Glob",
		"Grep",
		"WebFetch",
		"WebSearch",
		"AskUserQuestion",
		"Write", // 仅限计划文件
	}

	workflow := `## Plan Mode Workflow

### Phase 1: Initial Understanding
- Launch Explore agents to understand the codebase
- Ask user questions to clarify requirements

### Phase 2: Planning
- Design the implementation approach
- Identify critical files and dependencies

### Phase 3: Synthesis
- Collect findings and create a comprehensive plan
- Ask user about trade-offs and preferences

### Phase 4: Final Plan
- Update the plan file with final recommendations
- Include implementation steps and success criteria

### Phase 5: Exit
- Call ExitPlanMode when planning is complete
- Wait for user approval before implementation

**IMPORTANT**: In plan mode, you can ONLY:
- Read files (Read, Glob, Grep)
- Search web (WebFetch, WebSearch)
- Ask questions (AskUserQuestion)
- Write to the plan file (Write - only to: ` + planPath + `)

You CANNOT edit code, run commands, or make any changes until plan is approved.`

	// 构建响应
	result := map[string]interface{}{
		"ok":             true,
		"plan_file_path": planPath,
		"plan_id":        planID,
		"allowed_tools":  allowedTools,
		"workflow":       workflow,
		"message":        "Plan mode activated. You can now explore the codebase and create your plan.",
		"created_at":     time.Now().Unix(),
	}

	if reason != "" {
		result["reason"] = reason
	}

	// 添加提示信息
	result["next_steps"] = []string{
		"1. Read the codebase to understand existing patterns",
		"2. Ask user questions to clarify requirements",
		"3. Update the plan file with your findings",
		"4. Call ExitPlanMode when ready for user approval",
	}

	result["constraints"] = map[string]interface{}{
		"read_only":             true,
		"plan_file_writable":    planPath,
		"code_editing_disabled": true,
		"bash_disabled":         true,
	}

	return result, nil
}

func (t *EnterPlanModeTool) Prompt() string {
	return `进入规划模式，用于复杂任务的规划阶段。

## 何时使用此工具

当遇到以下情况时应使用此工具：
1. **多种有效方案** - 任务可以用多种方式解决，需要权衡
2. **重大架构决策** - 需要在架构模式之间做选择
3. **大规模变更** - 任务涉及多个文件或系统
4. **需求不明确** - 需要先探索才能理解完整范围
5. **需要用户确认** - 需要向用户提问以确认方向

## 规划模式的限制

在规划模式下，你只能：
- 读取文件（Read, Glob, Grep）
- 搜索网页（WebFetch, WebSearch）
- 向用户提问（AskUserQuestion）
- 写入计划文件（Write - 仅限指定的计划文件）

你不能：
- 编辑代码文件
- 运行 Bash 命令
- 创建或删除文件（除计划文件外）

## 工作流程

1. 调用此工具进入规划模式
2. 探索代码库，理解现有模式
3. 向用户提问以澄清需求
4. 更新计划文件记录发现和方案
5. 调用 ExitPlanMode 请求用户批准
6. 用户批准后开始实施

## 示例

{
  "reason": "需要设计用户认证系统，有多种方案可选"
}

## 注意事项

- 规划模式需要用户批准才能退出
- 计划文件存储在 .aster/plans/ 目录
- 完成规划后必须调用 ExitPlanMode`
}
