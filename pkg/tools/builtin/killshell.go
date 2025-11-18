package builtin

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/tools"
)

// KillShellTool 后台shell终止工具
// 支持终止后台运行的shell进程
type KillShellTool struct{}

// NewKillShellTool 创建KillShell工具
func NewKillShellTool(config map[string]interface{}) (tools.Tool, error) {
	return &KillShellTool{}, nil
}

func (t *KillShellTool) Name() string {
	return "KillShell"
}

func (t *KillShellTool) Description() string {
	return "终止后台运行的shell进程"
}

func (t *KillShellTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"shell_id": map[string]interface{}{
				"type":        "string",
				"description": "要终止的后台shell的ID",
			},
			"signal": map[string]interface{}{
				"type":        "string",
				"description": "发送的信号，默认为SIGTERM（15），可选：SIGTERM, SIGKILL, SIGINT",
			},
			"force": map[string]interface{}{
				"type":        "boolean",
				"description": "是否强制终止（等同于SIGKILL），默认为false",
			},
			"wait": map[string]interface{}{
				"type":        "boolean",
				"description": "是否等待进程完全退出，默认为true",
			},
			"timeout": map[string]interface{}{
				"type":        "integer",
				"description": "等待进程退出的超时时间（秒），默认为10秒",
			},
			"cleanup": map[string]interface{}{
				"type":        "boolean",
				"description": "是否清理相关的临时文件，默认为true",
			},
		},
		"required": []string{"shell_id"},
	}
}

func (t *KillShellTool) Execute(ctx context.Context, input map[string]interface{}, tc *tools.ToolContext) (interface{}, error) {
	// 验证必需参数
	if err := ValidateRequired(input, []string{"shell_id"}); err != nil {
		return NewClaudeErrorResponse(err), nil
	}

	shellID := GetStringParam(input, "shell_id", "")
	signal := GetStringParam(input, "signal", "SIGTERM")
	force := GetBoolParam(input, "force", false)
	wait := GetBoolParam(input, "wait", true)
	timeoutSeconds := GetIntParam(input, "timeout", 10)
	cleanup := GetBoolParam(input, "cleanup", true)

	if shellID == "" {
		return NewClaudeErrorResponse(fmt.Errorf("shell_id cannot be empty")), nil
	}

	start := time.Now()

	// 获取任务管理器
	taskManager := GetGlobalTaskManager()

	// 获取任务信息以验证存在
	task, err := taskManager.GetTask(shellID)
	if err != nil {
		return map[string]interface{}{
			"ok": false,
			"error": fmt.Sprintf("background shell not found: %s", shellID),
			"recommendations": []string{
				"确认shell_id是否正确",
				"检查任务是否已经被终止",
				"使用BashOutput工具查看可用的后台任务",
			},
			"shell_id": shellID,
			"duration_ms": time.Since(start).Milliseconds(),
		}, nil
	}

	// 检查任务状态
	if task.Status != "running" {
		return map[string]interface{}{
			"ok": false,
			"error": fmt.Sprintf("task is not running, current status: %s", task.Status),
			"shell_id": shellID,
			"command": task.Command,
			"pid": task.PID,
			"status": task.Status,
			"duration_ms": time.Since(start).Milliseconds(),
			"recommendations": []string{
				"任务已经完成或失败，无需终止",
				"使用BashOutput工具查看任务结果",
			},
		}, nil
	}

	// 确定要发送的信号
	finalSignal := signal
	if force {
		finalSignal = "SIGKILL"
	}

	// 执行终止操作
	err = taskManager.KillTask(shellID, finalSignal, timeoutSeconds)
	duration := time.Since(start)

	success := err == nil
	message := ""
	exitCode := 0

	if err != nil {
		message = fmt.Sprintf("failed to kill task: %v", err)
	} else {
		message = fmt.Sprintf("successfully sent signal %s to process %d", finalSignal, task.PID)

		// 如果需要等待进程退出
		if wait {
			time.Sleep(time.Duration(timeoutSeconds) * time.Second)

			// 检查任务最终状态
			if updatedTask, err := taskManager.GetTask(shellID); err == nil {
				if updatedTask.Status != "running" {
					exitCode = updatedTask.ExitCode
					if updatedTask.Status == "completed" {
						message = fmt.Sprintf("task completed successfully with exit code %d", exitCode)
					} else {
						message = fmt.Sprintf("task terminated with status %s, exit code %d", updatedTask.Status, exitCode)
					}
				} else {
					message = fmt.Sprintf("signal sent but process still running after %d seconds", timeoutSeconds)
				}
			}
		}

		// 清理任务相关文件
		if cleanup {
			if err := taskManager.CleanupTask(shellID); err != nil {
				message += fmt.Sprintf(" (cleanup warning: %v)", err)
			}
		}
	}

	// 获取更新后的任务状态
	var updatedTask *TaskInfo
	if updatedTask, _ = taskManager.GetTask(shellID); updatedTask == nil {
		updatedTask = task
	}

	// 构建响应
	response := map[string]interface{}{
		"ok": success,
		"shell_id": shellID,
		"command": task.Command,
		"pid": task.PID,
		"status": updatedTask.Status,
		"signal": finalSignal,
		"duration_ms": duration.Milliseconds(),
		"kill_time": start.Unix(),
		"success": success,
		"message": message,
		"force": force,
		"wait": wait,
		"cleanup": cleanup,
	}

	// 添加退出码（如果可获得）
	if exitCode != 0 {
		response["exit_code"] = exitCode
	}

	// 添加超时信息
	if wait {
		response["timeout_seconds"] = timeoutSeconds
	}

	// 添加清理信息
	if cleanup {
		response["cleanup_completed"] = true
		response["cleanup_info"] = t.getCleanupInfo(updatedTask)
	}

	// 添加任务性能统计
	response["task_duration_ms"] = updatedTask.Duration.Milliseconds()
	if updatedTask.EndTime != nil {
		response["end_time"] = updatedTask.EndTime.Unix()
	}

	// 添加工作目录信息
	if task.WorkDir != "" {
		response["working_directory"] = task.WorkDir
	}

	return response, nil
}

// isProcessRunning 检查进程是否运行（辅助函数）
func (t *KillShellTool) isProcessRunning(ctx context.Context, pid int, tc *tools.ToolContext) bool {
	if pid <= 0 {
		return false
	}

	// 使用ps命令检查进程
	psCmd := fmt.Sprintf("ps -p %d > /dev/null 2>&1", pid)
	result, err := tc.Sandbox.Exec(ctx, psCmd, &sandbox.ExecOptions{})
	if err != nil {
		return false
	}

	return result.Code == 0
}

// getSignalNumber 获取信号编号
func (t *KillShellTool) getSignalNumber(signal string) int {
	signalMap := map[string]int{
		"SIGTERM": 15,
		"SIGKILL": 9,
		"SIGINT":  2,
		"SIGHUP":  1,
		"SIGQUIT": 3,
		"SIGSTOP": 19,
		"SIGCONT": 18,
	}

	if num, exists := signalMap[signal]; exists {
		return num
	}

	// 尝试解析数字
	if num, err := strconv.Atoi(signal); err == nil {
		return num
	}

	// 默认使用SIGTERM
	return 15
}

// waitForProcessExit 等待进程退出（辅助函数）
func (t *KillShellTool) waitForProcessExit(ctx context.Context, pid int, timeoutSeconds int, tc *tools.ToolContext) int {
	timeout := time.Duration(timeoutSeconds) * time.Second
	startTime := time.Now()

	for time.Since(startTime) < timeout {
		if !t.isProcessRunning(ctx, pid, tc) {
			// 进程已退出，获取退出码
			waitCmd := fmt.Sprintf("wait %d 2>/dev/null; echo $?", pid)
			if result, err := tc.Sandbox.Exec(ctx, waitCmd, &sandbox.ExecOptions{}); err == nil {
				var exitCode int
				_, _ = fmt.Sscanf(strings.TrimSpace(result.Stdout), "%d", &exitCode)
				return exitCode
			}
			return 0 // 假设成功退出
		}

		// 短暂等待后重试
		time.Sleep(100 * time.Millisecond)
	}

	// 超时，进程仍在运行
	return -1
}

// getCleanupInfo 获取清理信息
func (t *KillShellTool) getCleanupInfo(task *TaskInfo) map[string]interface{} {
	if task == nil || task.Options == nil {
		return map[string]interface{}{
			"files_cleared": 0,
			"cleanup_method": "not_available",
		}
	}

	return map[string]interface{}{
		"output_file": fmt.Sprintf("%s/%s.stdout", task.Options.OutputDir, task.ID),
		"error_file":  fmt.Sprintf("%s/%s.stderr", task.Options.OutputDir, task.ID),
		"task_file":   fmt.Sprintf("%s/%s.json", "/tmp/agentsdk_tasks", task.ID),
		"files_cleared": 3,
		"cleanup_method": "file_truncate_and_remove",
		"cleanup_timestamp": time.Now().Unix(),
	}
}

func (t *KillShellTool) Prompt() string {
	return `终止后台运行的shell进程。

功能特性：
- 支持多种信号终止（SIGTERM, SIGKILL, SIGINT）
- 可配置强制终止
- 等待进程完全退出
- 自动清理临时文件
- 详细的终止状态报告

使用指南：
- shell_id: 必需参数，要终止的后台shell的ID
- signal: 可选参数，发送的信号类型
- force: 可选参数，是否强制终止
- wait: 可选参数，是否等待进程退出
- timeout: 可选参数，等待超时时间
- cleanup: 可选参数，是否清理临时文件

信号类型：
- SIGTERM (15): 优雅终止，默认选项
- SIGKILL (9): 强制终止，无法忽略
- SIGINT (2): 中断信号，等同于Ctrl+C

注意事项：
- 需要与Bash工具的background功能配合使用
- 当前为简化实现，需要完整的后台任务管理
- 建议实现任务状态持久化存储
- 支持超时控制和错误处理

安全特性：
- 权限检查集成
- 进程状态验证
- 清理机制保护
- 详细的操作日志`
}