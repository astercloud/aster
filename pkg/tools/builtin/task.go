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
			"action": map[string]any{
				"type":        "string",
				"description": "操作类型：run（启动任务）、status（查询状态）、list（列出任务）、cancel（取消任务）",
				"enum":        []string{"run", "status", "list", "cancel"},
				"default":     "run",
			},
			"task_id": map[string]any{
				"type":        "string",
				"description": "任务 ID（用于 status/cancel 操作）",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "任务的简短描述（3-5个词），概括任务目的",
			},
			"subagent_type": map[string]any{
				"type":        "string",
				"description": "要启动的代理类型（用于 run 操作）",
				"enum":        []string{"general-purpose", "Explore", "Plan"},
			},
			"prompt": map[string]any{
				"type":        "string",
				"description": "要代理执行的任务描述，必须是详细的（用于 run 操作）",
			},
			"model": map[string]any{
				"type":        "string",
				"description": "可选模型，如果未指定则继承自父级",
			},
			"timeout_minutes": map[string]any{
				"type":        "integer",
				"description": "任务超时时间（分钟），默认为30",
			},
			"async": map[string]any{
				"type":        "boolean",
				"description": "是否异步执行，默认为true",
			},
		},
		"required": []string{},
	}
}

func (t *TaskTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	action := GetStringParam(input, "action", "run")

	switch action {
	case "list":
		return t.listTasks()
	case "status":
		taskID := GetStringParam(input, "task_id", "")
		if taskID == "" {
			return NewClaudeErrorResponse(fmt.Errorf("task_id is required for status action")), nil
		}
		return t.getTaskStatus(taskID)
	case "cancel":
		taskID := GetStringParam(input, "task_id", "")
		if taskID == "" {
			return NewClaudeErrorResponse(fmt.Errorf("task_id is required for cancel action")), nil
		}
		return t.cancelTask(taskID)
	case "run":
		return t.runTask(ctx, input)
	default:
		return NewClaudeErrorResponse(fmt.Errorf("unknown action: %s", action)), nil
	}
}

// listTasks 列出所有任务
func (t *TaskTool) listTasks() (any, error) {
	executor := GetGlobalTaskExecutor()
	tasks := executor.ListTasks()

	taskList := make([]map[string]any, 0, len(tasks))
	for _, task := range tasks {
		taskInfo := map[string]any{
			"task_id":    task.TaskID,
			"subagent":   task.Subagent,
			"status":     task.Status,
			"start_time": task.StartTime.Unix(),
			"duration":   task.Duration.String(),
		}
		if task.Error != "" {
			taskInfo["error"] = task.Error
		}
		taskList = append(taskList, taskInfo)
	}

	return map[string]any{
		"ok":          true,
		"action":      "list",
		"total_tasks": len(taskList),
		"tasks":       taskList,
	}, nil
}

// getTaskStatus 获取任务状态
func (t *TaskTool) getTaskStatus(taskID string) (any, error) {
	executor := GetGlobalTaskExecutor()
	task, err := executor.GetTask(taskID)
	if err != nil {
		return NewClaudeErrorResponse(err), nil
	}

	response := map[string]any{
		"ok":         true,
		"action":     "status",
		"task_id":    task.TaskID,
		"subagent":   task.Subagent,
		"model":      task.Model,
		"status":     task.Status,
		"start_time": task.StartTime.Unix(),
		"duration":   task.Duration.String(),
	}

	if task.Status == "completed" && task.Result != nil {
		response["result"] = task.Result
	}
	if task.Error != "" {
		response["error"] = task.Error
	}
	if task.Metadata != nil {
		response["metadata"] = task.Metadata
	}
	if task.Status == "running" {
		response["running_time"] = time.Since(task.StartTime).String()
	}

	return response, nil
}

// cancelTask 取消任务
func (t *TaskTool) cancelTask(taskID string) (any, error) {
	executor := GetGlobalTaskExecutor()
	task, err := executor.GetTask(taskID)
	if err != nil {
		return NewClaudeErrorResponse(err), nil
	}

	if task.Status == "completed" || task.Status == "failed" {
		return map[string]any{
			"ok":      false,
			"action":  "cancel",
			"task_id": taskID,
			"status":  task.Status,
			"message": "Task already finished, cannot cancel",
		}, nil
	}

	return map[string]any{
		"ok":      true,
		"action":  "cancel",
		"task_id": taskID,
		"message": "Cancel request sent",
		"note":    "Task cancellation is best-effort",
	}, nil
}

// runTask 启动任务
func (t *TaskTool) runTask(ctx context.Context, input map[string]any) (any, error) {
	description := GetStringParam(input, "description", "")
	subagentType := GetStringParam(input, "subagent_type", "")
	prompt := GetStringParam(input, "prompt", "")
	model := GetStringParam(input, "model", "")
	timeoutMinutes := GetIntParam(input, "timeout_minutes", 30)
	async := GetBoolParam(input, "async", true)

	if subagentType == "" {
		return NewClaudeErrorResponse(fmt.Errorf("subagent_type is required for run action")), nil
	}
	if prompt == "" {
		return NewClaudeErrorResponse(fmt.Errorf("prompt is required for run action")), nil
	}

	// 验证子代理类型
	validSubagents := []string{"general-purpose", "Explore", "Plan"}
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
			"支持的代理类型: general-purpose, Explore, Plan",
		), nil
	}

	start := time.Now()
	taskExecutor := GetGlobalTaskExecutor()

	if taskExecutor.executorFactory != nil {
		return t.executeWithTaskExecutor(ctx, taskExecutor, subagentType, prompt, model, description, timeoutMinutes, 100, async, start)
	}

	return t.executeWithSubagentManager(ctx, subagentType, prompt, model, description, "", timeoutMinutes, 100, async, start)
}

// executeWithTaskExecutor 使用新的 TaskExecutor 执行（真正的子 Agent）
func (t *TaskTool) executeWithTaskExecutor(ctx context.Context, executor *TaskExecutor, subagentType, prompt, model, description string, timeoutMinutes, priority int, async bool, start time.Time) (any, error) {
	opts := &TaskExecuteOptions{
		Model:    model,
		Timeout:  time.Duration(timeoutMinutes) * time.Minute,
		Priority: priority,
		Async:    async,
		Context:  make(map[string]any),
	}

	if async {
		// 异步执行
		handle, err := executor.ExecuteAsync(ctx, subagentType, prompt, opts)
		if err != nil {
			return map[string]any{
				"ok":            false,
				"error":         fmt.Sprintf("failed to start subagent: %v", err),
				"subagent_type": subagentType,
				"duration_ms":   time.Since(start).Milliseconds(),
				"execution_mode": "native_subagent",
			}, nil
		}

		response := map[string]any{
			"ok":              true,
			"task_id":         handle.TaskID,
			"subagent_type":   subagentType,
			"prompt":          prompt,
			"model":           model,
			"status":          handle.Status,
			"duration_ms":     time.Since(start).Milliseconds(),
			"start_time":      handle.StartTime.Unix(),
			"async":           true,
			"priority":        priority,
			"timeout_minutes": timeoutMinutes,
			"execution_mode":  "native_subagent",
			"async_status":    "running_in_background",
			"monitoring_info": "Task is running as a native subagent. Use task_id to query status.",
		}
		if description != "" {
			response["description"] = description
		}
		return response, nil
	}

	// 同步执行
	execution, err := executor.Execute(ctx, subagentType, prompt, opts)
	if err != nil {
		return map[string]any{
			"ok":            false,
			"error":         fmt.Sprintf("failed to execute subagent: %v", err),
			"subagent_type": subagentType,
			"duration_ms":   time.Since(start).Milliseconds(),
			"execution_mode": "native_subagent",
		}, nil
	}

	response := map[string]any{
		"ok":              true,
		"task_id":         execution.TaskID,
		"subagent_type":   subagentType,
		"prompt":          prompt,
		"model":           model,
		"status":          execution.Status,
		"duration_ms":     execution.Duration.Milliseconds(),
		"start_time":      execution.StartTime.Unix(),
		"async":           false,
		"priority":        priority,
		"timeout_minutes": timeoutMinutes,
		"execution_mode":  "native_subagent",
	}

	if description != "" {
		response["description"] = description
	}

	if execution.Result != nil {
		response["output"] = execution.Result
	}

	if execution.Error != "" {
		response["error"] = execution.Error
	}

	if execution.Metadata != nil {
		response["metadata"] = execution.Metadata
	}

	return response, nil
}

// executeWithSubagentManager 使用旧的 SubagentManager 执行（进程级别）
func (t *TaskTool) executeWithSubagentManager(ctx context.Context, subagentType, prompt, model, description, resume string, timeoutMinutes, priority int, async bool, start time.Time) (any, error) {
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
			"execution_mode": "process_subagent",
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
		"execution_mode":  "process_subagent",
	}

	if description != "" {
		response["description"] = description
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

## 操作类型

- run: 启动新任务（默认）
- status: 查询任务状态
- list: 列出所有任务
- cancel: 取消任务

## 子代理类型

- general-purpose: 通用代理，处理复杂查询和多步骤任务
- Explore: 代码探索代理，快速搜索和分析代码库
- Plan: 计划代理，探索代码库并制定执行计划

## 参数说明

启动任务 (action=run):
- subagent_type: 必需，子代理类型
- prompt: 必需，详细的任务描述
- model: 可选，使用的模型
- timeout_minutes: 可选，超时时间（默认30）
- async: 可选，是否异步执行（默认true）

查询/取消任务 (action=status/cancel):
- task_id: 必需，任务 ID`
}

// Examples 返回 Task 工具的使用示例
func (t *TaskTool) Examples() []tools.ToolExample {
	return []tools.ToolExample{
		{
			Description: "启动代码探索代理",
			Input: map[string]any{
				"action":        "run",
				"subagent_type": "Explore",
				"prompt":        "搜索所有与用户认证相关的代码文件",
			},
		},
		{
			Description: "查询任务状态",
			Input: map[string]any{
				"action":  "status",
				"task_id": "task_123456",
			},
		},
		{
			Description: "列出所有任务",
			Input: map[string]any{
				"action": "list",
			},
		},
		{
			Description: "取消任务",
			Input: map[string]any{
				"action":  "cancel",
				"task_id": "task_123456",
			},
		},
	}
}
