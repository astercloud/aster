package builtin

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/astercloud/aster/pkg/tools"
)

// BashOutputTool 后台shell输出获取工具
// 支持获取后台运行命令的输出和状态
type BashOutputTool struct{}

// ResourceUsage 进程资源使用情况
type ResourceUsage struct {
	CPU    float64 `json:"cpu_percent"`    // CPU使用率百分比
	Memory int64   `json:"memory_bytes"`   // 内存使用量（字节）
	DiskIO int64   `json:"disk_io_bytes"`  // 磁盘IO（字节）
}

// NewBashOutputTool 创建BashOutput工具
func NewBashOutputTool(config map[string]interface{}) (tools.Tool, error) {
	return &BashOutputTool{}, nil
}

func (t *BashOutputTool) Name() string {
	return "BashOutput"
}

func (t *BashOutputTool) Description() string {
	return "获取后台运行shell的输出和状态信息"
}

func (t *BashOutputTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"bash_id": map[string]interface{}{
				"type":        "string",
				"description": "要获取输出的后台shell的ID",
			},
			"filter": map[string]interface{}{
				"type":        "string",
				"description": "可选的正则表达式过滤器，只返回匹配的输出行",
			},
			"lines": map[string]interface{}{
				"type":        "integer",
				"description": "返回的最大行数，默认为100，0表示返回全部",
			},
			"follow": map[string]interface{}{
				"type":        "boolean",
				"description": "是否持续跟随输出（类似tail -f），默认为false",
			},
			"include_stderr": map[string]interface{}{
				"type":        "boolean",
				"description": "是否包含stderr输出，默认为true",
			},
			"since": map[string]interface{}{
				"type":        "string",
				"description": "只返回指定时间之后的输出，格式如'2023-01-01T00:00:00Z'",
			},
			"resource_info": map[string]interface{}{
				"type":        "boolean",
				"description": "是否包含进程资源使用信息，默认为false",
			},
			"clear_cache": map[string]interface{}{
				"type":        "boolean",
				"description": "是否清除已缓存的输出，默认为false",
			},
		},
		"required": []string{"bash_id"},
	}
}

func (t *BashOutputTool) Execute(ctx context.Context, input map[string]interface{}, tc *tools.ToolContext) (interface{}, error) {
	// 验证必需参数
	if err := ValidateRequired(input, []string{"bash_id"}); err != nil {
		return NewClaudeErrorResponse(err), nil
	}

	bashID := GetStringParam(input, "bash_id", "")
	filter := GetStringParam(input, "filter", "")
	lines := GetIntParam(input, "lines", 100)
	follow := GetBoolParam(input, "follow", false)
	includeStderr := GetBoolParam(input, "include_stderr", true)
	since := GetStringParam(input, "since", "")
	resourceInfo := GetBoolParam(input, "resource_info", false)
	clearCache := GetBoolParam(input, "clear_cache", false)

	if bashID == "" {
		return NewClaudeErrorResponse(fmt.Errorf("bash_id cannot be empty")), nil
	}

	start := time.Now()

	// 获取任务管理器
	taskManager := GetGlobalTaskManager()

	// 获取任务信息
	task, err := taskManager.GetTask(bashID)
	if err != nil {
		return map[string]interface{}{
			"ok": false,
			"error": fmt.Sprintf("background task not found: %s", bashID),
			"recommendations": []string{
				"确认bash_id是否正确",
				"检查任务是否还在运行",
				"使用Bash工具查看可用的后台任务",
			},
			"bash_id": bashID,
			"duration_ms": time.Since(start).Milliseconds(),
		}, nil
	}

	// 获取任务输出
	stdout, stderr, err := taskManager.GetTaskOutput(bashID, filter, lines)
	if err != nil {
		return map[string]interface{}{
			"ok": false,
			"error": fmt.Sprintf("failed to get task output: %v", err),
			"bash_id": bashID,
			"duration_ms": time.Since(start).Milliseconds(),
		}, nil
	}

	// 处理clear_cache
	if clearCache {
		// 清理任务的输出文件缓存
		tm := taskManager.(*FileTaskManager)
		_ = tm.cleanupOutputFiles(bashID)
	}

	duration := time.Since(start)

	// 合并输出
	var fullOutput string
	if includeStderr && stderr != "" {
		fullOutput = stdout + "\nSTDERR:\n" + stderr
	} else {
		fullOutput = stdout
	}

	// 应用时间过滤（如果指定了since参数）
	if since != "" {
		stdout = t.filterByTime(stdout, since)
		if includeStderr {
			stderr = t.filterByTime(stderr, since)
		}
	}

	// 获取进程资源信息
	var resourceUsage *ResourceUsage
	if resourceInfo && task.Status == "running" {
		resourceUsage = t.getResourceUsage(task.PID)
	}

	// 构建响应
	response := map[string]interface{}{
		"ok": true,
		"bash_id": bashID,
		"command": task.Command,
		"status": task.Status,
		"stdout": stdout,
		"new_output": fullOutput,
		"duration_ms": duration.Milliseconds(),
		"start_time": task.StartTime.Unix(),
		"last_check": time.Now().Unix(),
		"follow": follow,
		"include_stderr": includeStderr,
		"filter": filter,
		"lines_limit": lines,
	}

	// 添加stderr（如果请求）
	if includeStderr {
		response["stderr"] = stderr
	}

	// 添加退出码（如果已完成）
	if task.Status == "completed" || task.Status == "failed" {
		response["exit_code"] = task.ExitCode
		response["end_time"] = task.EndTime.Unix()
		response["total_duration_ms"] = task.Duration.Milliseconds()
	}

	// 添加进程信息
	if task.PID > 0 {
		response["pid"] = task.PID
	}

	// 添加输出统计
	response["stdout_lines"] = len(strings.Split(stdout, "\n"))
	if includeStderr {
		response["stderr_lines"] = len(strings.Split(stderr, "\n"))
		response["total_lines"] = response["stdout_lines"].(int) + response["stderr_lines"].(int)
	} else {
		response["total_lines"] = response["stdout_lines"].(int)
	}

	// 如果是跟随模式，设置检查间隔
	if follow {
		response["follow_mode"] = true
		response["next_check_interval"] = "1s"
	}

	// 添加时间过滤信息
	if since != "" {
		response["since_filter"] = since
		response["time_filter_applied"] = true
	}

	// 添加资源使用情况
	if resourceUsage != nil {
		response["resource_usage"] = resourceUsage
	}

	// 添加工作目录信息
	if task.WorkDir != "" {
		response["working_directory"] = task.WorkDir
	}

	// 添加元数据信息
	if len(task.Metadata) > 0 {
		response["metadata"] = task.Metadata
	}

	// 添加任务性能统计
	response["task_duration_ms"] = task.Duration.Milliseconds()
	response["task_last_update"] = task.LastUpdate.Unix()

	return response, nil
}

// filterOutput 使用正则表达式过滤输出
func (t *BashOutputTool) filterOutput(output, filter string) string {
	if filter == "" || output == "" {
		return output
	}

	regex, err := regexp.Compile(filter)
	if err != nil {
		return output // 过滤器无效，返回原输出
	}

	lines := strings.Split(output, "\n")
	var filteredLines []string

	for _, line := range lines {
		if regex.MatchString(line) {
			filteredLines = append(filteredLines, line)
		}
	}

	return strings.Join(filteredLines, "\n")
}

// limitLines 限制输出行数
func (t *BashOutputTool) limitLines(output string, maxLines int) string {
	if maxLines <= 0 || output == "" {
		return output
	}

	lines := strings.Split(output, "\n")
	if len(lines) <= maxLines {
		return output
	}

	// 返回最后的maxLines行
	return strings.Join(lines[len(lines)-maxLines:], "\n")
}

// filterByTime 按时间过滤输出
func (t *BashOutputTool) filterByTime(output, since string) string {
	if since == "" || output == "" {
		return output
	}

	// 对于简单的文本输出，按行分割并过滤
	lines := strings.Split(output, "\n")
	var filteredLines []string

	// 提取日期部分（YYYY-MM-DD格式）
	datePart := since
	if len(since) >= 10 {
		datePart = since[:10]
	}

	for _, line := range lines {
		// 这里假设输出中包含时间戳，实际实现可能需要更复杂的解析
		// 简化实现：如果行中包含日期，则保留
		if strings.Contains(line, datePart) {
			filteredLines = append(filteredLines, line)
		}
	}

	// 如果没有找到匹配的行，返回原始输出
	if len(filteredLines) == 0 {
		return output
	}

	return strings.Join(filteredLines, "\n")
}

// getResourceUsage 获取进程资源使用情况
func (t *BashOutputTool) getResourceUsage(pid int) *ResourceUsage {
	if pid <= 0 {
		return nil
	}

	// 使用ps命令获取资源使用情况
	cmd := exec.Command("ps", "-p", fmt.Sprintf("%d", pid), "-o", "%cpu,rss,vsz", "--no-headers")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	// 解析输出
	fields := strings.Fields(strings.TrimSpace(string(output)))
	if len(fields) < 3 {
		return nil
	}

	var cpu float64
	var memory int64
	_, _ = fmt.Sscanf(fields[0], "%f", &cpu)
	_, _ = fmt.Sscanf(fields[1], "%d", &memory)
	memory *= 1024 // 转换为字节

	return &ResourceUsage{
		CPU:    cpu,
		Memory: memory,
		DiskIO: 0, // 简化实现，不获取磁盘IO
	}
}

func (t *BashOutputTool) Prompt() string {
	return `获取后台运行shell的输出和状态信息。

功能特性：
- 实时获取后台命令输出
- 支持正则表达式过滤
- 可配置输出行数限制
- 支持跟随模式（类似tail -f）
- 进程状态监控
- 资源使用情况查询

使用指南：
- bash_id: 必需参数，后台shell的ID
- filter: 可选参数，正则表达式过滤器
- lines: 可选参数，返回的最大行数
- follow: 可选参数，是否持续跟随输出
- include_stderr: 可选参数，是否包含错误输出
- resource_info: 可选参数，是否获取资源使用信息

注意事项：
- 需要与Bash工具的background功能配合使用
- 当前为简化实现，需要完整的后台任务管理
- 建议实现任务状态持久化存储
- 支持超时控制和错误处理

集成说明：
- 建议与后台任务管理器集成
- 可实现输出缓存机制
- 支持多种输出格式（JSON/文本）`
}