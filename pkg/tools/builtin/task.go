package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/astercloud/aster/pkg/tools"
)

// TaskTool 专门代理启动工具
// 支持启动专门的代理来处理复杂的多步骤任务
type TaskTool struct{}

// TaskDefinition 任务定义
type TaskDefinition struct {
	ID          string         `json:"id"`
	Description string         `json:"description"`
	Subagent    string         `json:"subagent"`
	Prompt      string         `json:"prompt"`
	Model       string         `json:"model,omitempty"`
	Resume      string         `json:"resume,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	StartedAt   *time.Time     `json:"startedAt,omitempty"`
	CompletedAt *time.Time     `json:"completedAt,omitempty"`
	Status      string         `json:"status"` // "created", "running", "completed", "failed"
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// TaskExecution 任务执行结果
type TaskExecution struct {
	TaskID    string         `json:"task_id"`
	Subagent  string         `json:"subagent"`
	Model     string         `json:"model"`
	Status    string         `json:"status"`
	Result    any            `json:"result,omitempty"`
	Error     string         `json:"error,omitempty"`
	StartTime time.Time      `json:"start_time"`
	EndTime   *time.Time     `json:"end_time,omitempty"`
	Duration  time.Duration  `json:"duration"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// NewTaskTool 创建Task工具
func NewTaskTool(config map[string]any) (tools.Tool, error) {
	return &TaskTool{}, nil
}

func (t *TaskTool) Name() string {
	return "Task"
}

func (t *TaskTool) Description() string {
	return "启动专门的代理来处理复杂的多步骤任务"
}

func (t *TaskTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"subagent_type": map[string]any{
				"type":        "string",
				"description": "要启动的代理类型",
				"enum":        []string{"general-purpose", "statusline-setup", "Explore", "Plan"},
			},
			"prompt": map[string]any{
				"type":        "string",
				"description": "要代理执行的任务描述，必须是详细的",
			},
			"model": map[string]any{
				"type":        "string",
				"description": "可选模型，如果未指定则继承自父级",
			},
			"resume": map[string]any{
				"type":        "string",
				"description": "可选代理ID以继续执行，如果提供则忽略其他参数",
			},
			"timeout_minutes": map[string]any{
				"type":        "integer",
				"description": "任务超时时间（分钟），默认为30",
			},
			"priority": map[string]any{
				"type":        "integer",
				"description": "任务优先级（数值越大优先级越高），默认为100",
			},
			"async": map[string]any{
				"type":        "boolean",
				"description": "是否异步执行，默认为true",
			},
		},
		"required": []string{"subagent_type", "prompt"},
	}
}

func (t *TaskTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	// 验证必需参数
	if err := ValidateRequired(input, []string{"subagent_type", "prompt"}); err != nil {
		return NewClaudeErrorResponse(err), nil
	}

	subagentType := GetStringParam(input, "subagent_type", "")
	prompt := GetStringParam(input, "prompt", "")
	model := GetStringParam(input, "model", "")
	resume := GetStringParam(input, "resume", "")
	timeoutMinutes := GetIntParam(input, "timeout_minutes", 30)
	priority := GetIntParam(input, "priority", 100)
	async := GetBoolParam(input, "async", true)

	if subagentType == "" {
		return NewClaudeErrorResponse(fmt.Errorf("subagent_type cannot be empty")), nil
	}

	if prompt == "" && resume == "" {
		return NewClaudeErrorResponse(fmt.Errorf("prompt cannot be empty when not resuming")), nil
	}

	// 验证子代理类型
	validSubagents := []string{
		"general-purpose",
		"statusline-setup",
		"Explore",
		"Plan",
	}
	subagentValid := false
	for _, valid := range validSubagents {
		if subagentType == valid {
			subagentValid = true
			break
		}
	}
	if !subagentValid {
		return NewClaudeErrorResponse(
			fmt.Errorf("invalid subagent_type: %s", subagentType),
			"支持的代理类型: general-purpose, statusline-setup, Explore, Plan",
		), nil
	}

	start := time.Now()

	// 获取子代理管理器
	subagentManager := GetGlobalSubagentManager()

	var subagent *SubagentInstance
	var err error

	if resume != "" {
		// 恢复现有子代理
		subagent, err = subagentManager.ResumeSubagent(resume)
	} else {
		// 创建新子代理配置
		config := &SubagentConfig{
			Type:    subagentType,
			Prompt:  prompt,
			Model:   model,
			Timeout: time.Duration(timeoutMinutes) * time.Minute,
			Metadata: map[string]string{
				"priority": fmt.Sprintf("%d", priority),
				"async":    fmt.Sprintf("%t", async),
				"created":  fmt.Sprintf("%d", time.Now().Unix()),
			},
		}

		// 启动子代理
		subagent, err = subagentManager.StartSubagent(ctx, config)
	}

	duration := time.Since(start)

	if err != nil {
		return map[string]any{
			"ok":            false,
			"error":         fmt.Sprintf("failed to start/resume subagent: %v", err),
			"subagent_type": subagentType,
			"duration_ms":   duration.Milliseconds(),
			"recommendations": []string{
				"检查子代理类型是否正确",
				"确认提示词是否有效",
				"验证系统环境是否支持子代理启动",
			},
		}, nil
	}

	// 构建响应
	response := map[string]any{
		"ok":              true,
		"task_id":         subagent.ID,
		"subagent_type":   subagentType,
		"prompt":          prompt,
		"model":           subagent.Config.Model,
		"status":          subagent.Status,
		"duration_ms":     duration.Milliseconds(),
		"start_time":      subagent.StartTime.Unix(),
		"async":           async,
		"priority":        priority,
		"timeout_minutes": timeoutMinutes,
		"pid":             subagent.PID,
		"command":         subagent.Command,
	}

	// 添加子代理配置信息
	if subagent.Config != nil {
		response["subagent_config"] = map[string]any{
			"timeout":     subagent.Config.Timeout.String(),
			"max_tokens":  subagent.Config.MaxTokens,
			"temperature": subagent.Config.Temperature,
			"work_dir":    subagent.Config.WorkDir,
		}
	}

	// 添加输出（如果已完成）
	if subagent.Status == "completed" || subagent.Status == "failed" {
		if output, err := subagentManager.GetSubagentOutput(subagent.ID); err == nil {
			response["output"] = output
			response["output_length"] = len(output)
		}

		response["exit_code"] = subagent.ExitCode
		if subagent.EndTime != nil {
			response["end_time"] = subagent.EndTime.Unix()
			response["total_duration_ms"] = subagent.Duration.Milliseconds()
		}

		if subagent.Error != "" {
			response["error"] = subagent.Error
		}
	}

	// 添加资源使用情况
	if subagent.ResourceUsage != nil {
		response["resource_usage"] = subagent.ResourceUsage
	}

	// 添加元数据
	if len(subagent.Metadata) > 0 {
		response["metadata"] = subagent.Metadata
	}

	// 添加子代理性能统计
	response["subagent_duration_ms"] = subagent.Duration.Milliseconds()
	response["subagent_last_update"] = subagent.LastUpdate.Unix()

	// 如果是异步模式，说明任务状态
	if async {
		if subagent.Status == "running" {
			response["async_status"] = "running_in_background"
			response["monitoring_info"] = "使用相同的task_id可以查询状态"
		}
	} else {
		response["async_status"] = "synchronous_execution"
	}

	return response, nil
}

// 以下函数预留用于未来的子代理功能集成
// 当前实现为简化版本，完整功能需要与agent框架深度集成

// executeTask 执行任务（简化实现）
// 预留用于启动子代理执行任务
//
//nolint:unused // 预留用于未来子代理功能
func (t *TaskTool) executeTask(ctx context.Context, taskDef *TaskDefinition, tc *tools.ToolContext) *TaskExecution {
	startTime := time.Now()

	// 简化实现：模拟任务执行
	// 实际实现中，这里会启动对应的子代理
	execution := &TaskExecution{
		TaskID:    taskDef.ID,
		Subagent:  taskDef.Subagent,
		Model:     taskDef.Model,
		Status:    "not_implemented",
		StartTime: startTime,
		Duration:  time.Since(startTime),
		Metadata: map[string]any{
			"note":             "Subagent execution requires integration with agent framework",
			"task_description": taskDef.Description,
		},
	}

	// 模拟设置开始时间
	now := time.Now()
	taskDef.StartedAt = &now
	taskDef.Status = "running"

	return execution
}

// resumeTask 恢复任务（简化实现）
// 预留用于恢复暂停的子代理任务
//
//nolint:unused // 预留用于未来子代理功能
func (t *TaskTool) resumeTask(ctx context.Context, taskID string, tc *tools.ToolContext) *TaskExecution {
	startTime := time.Now()

	// 简化实现：模拟任务恢复
	execution := &TaskExecution{
		TaskID:    taskID,
		Subagent:  "unknown",
		Model:     "",
		Status:    "resume_not_implemented",
		StartTime: startTime,
		Duration:  time.Since(startTime),
		Metadata: map[string]any{
			"note":       "Task resumption requires integration with agent framework",
			"resumed_at": startTime.Unix(),
		},
	}

	return execution
}

func (t *TaskTool) Prompt() string {
	return `启动专门的代理来处理复杂的多步骤任务。

功能特性：
- 支持多种专业化子代理
- 异步任务执行
- 任务状态跟踪
- 优先级管理
- 超时控制

子代理类型：
- general-purpose: 通用代理，处理复杂查询和多步骤任务
- statusline-setup: 状态线配置代理
- Explore: 代码探索代理，快速搜索和分析代码库
- Plan: 计划代理，探索代码库并制定执行计划

使用指南：
- subagent_type: 必需参数，子代理类型
- prompt: 必需参数，详细的任务描述
- model: 可选参数，使用的模型
- resume: 可选参数，恢复现有任务
- timeout_minutes: 可选参数，超时时间
- priority: 可选参数，任务优先级
- async: 可选参数，是否异步执行

注意事项：
- 当前为简化实现，需要完整的子代理框架
- 建议实现任务状态持久化存储
- 支持任务执行进度跟踪
- 可集成外部代理服务

集成要求：
- 需要实现代理启动和管理机制
- 建议支持代理间通信
- 可实现任务结果缓存
- 支持代理执行日志记录`
}

// Examples 返回 Task 工具的使用示例
// 实现 ExampleableTool 接口，帮助 LLM 更准确地调用工具
func (t *TaskTool) Examples() []tools.ToolExample {
	return []tools.ToolExample{
		{
			Description: "启动代码探索代理",
			Input: map[string]any{
				"subagent_type": "Explore",
				"prompt":        "搜索所有与用户认证相关的代码文件",
			},
		},
		{
			Description: "启动通用代理执行复杂任务",
			Input: map[string]any{
				"subagent_type":   "general-purpose",
				"prompt":          "分析项目架构并生成文档",
				"model":           "sonnet",
				"timeout_minutes": 60,
			},
		},
		{
			Description: "恢复之前暂停的任务",
			Input: map[string]any{
				"subagent_type": "general-purpose",
				"prompt":        "",
				"resume":        "task_123456",
			},
		},
	}
}
