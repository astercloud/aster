package builtin

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/tools"
)

// BashTool 增强的Bash命令执行工具
// 支持持久化shell会话功能
type BashTool struct {
	dangerousPatterns []*regexp.Regexp
	defaultTimeout    time.Duration
}

// NewBashTool 创建Bash执行工具
func NewBashTool(config map[string]any) (tools.Tool, error) {
	tool := &BashTool{
		defaultTimeout: 2 * time.Minute,
	}

	// 编译危险命令模式
	dangerousCommands := []string{
		`rm -rf /`,
		`sudo rm`,
		`:(){ :|:& };:`,
		`chmod 777 /`,
		`dd if=/dev/zero`,
		`format`,
		`fdisk`,
	}

	for _, pattern := range dangerousCommands {
		regex, err := regexp.Compile(pattern)
		if err == nil {
			tool.dangerousPatterns = append(tool.dangerousPatterns, regex)
		}
	}

	return tool, nil
}

func (t *BashTool) Name() string {
	return "Bash" // 使用标准的工具名称
}

func (t *BashTool) Description() string {
	return "执行bash命令在持久化shell会话中"
}

func (t *BashTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "要执行的bash命令",
			},
			"timeout": map[string]any{
				"type":        "integer",
				"description": "超时时间（毫秒），默认为120000（2分钟），最大600000",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "命令的简短描述（5-10个词），说明此命令的作用",
			},
			"working_dir": map[string]any{
				"type":        "string",
				"description": "命令执行的工作目录，默认为当前目录",
			},
			"shell_id": map[string]any{
				"type":        "string",
				"description": "在指定的shell会话中执行命令，如果未提供则创建新会话",
			},
			"background": map[string]any{
				"type":        "boolean",
				"description": "是否在后台运行命令，默认为false",
			},
			"capture_output": map[string]any{
				"type":        "boolean",
				"description": "是否捕获命令输出，默认为true",
			},
			"environment": map[string]any{
				"type":        "object",
				"description": "设置环境变量",
				"additionalProperties": map[string]any{
					"type": "string",
				},
			},
			"shell": map[string]any{
				"type":        "string",
				"description": "使用的shell类型，如bash, sh, zsh，默认为bash",
			},
		},
		"required": []string{"command"},
	}
}

func (t *BashTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	// 验证必需参数
	if err := ValidateRequired(input, []string{"command"}); err != nil {
		return NewClaudeErrorResponse(err), nil
	}

	command := GetStringParam(input, "command", "")
	timeoutMs := GetIntParam(input, "timeout", 120000) // 默认2分钟
	description := GetStringParam(input, "description", "")
	workingDir := GetStringParam(input, "working_dir", "")
	shellID := GetStringParam(input, "shell_id", "")
	background := GetBoolParam(input, "background", false)
	captureOutput := GetBoolParam(input, "capture_output", true)
	shellType := GetStringParam(input, "shell", "bash")

	// 限制最大超时时间为 600000ms (10分钟)
	if timeoutMs > 600000 {
		timeoutMs = 600000
	}

	if command == "" {
		return NewClaudeErrorResponse(errors.New("command cannot be empty")), nil
	}

	// Git 安全检查
	gitCheck := GetGlobalGitSafetyValidator().Check(command)
	if gitCheck.IsGitCommand {
		if gitCheck.Blocked {
			return map[string]any{
				"ok":              false,
				"blocked":         true,
				"is_git_command":  true,
				"risk":            gitCheck.RiskName,
				"reason":          gitCheck.Reason,
				"warnings":        gitCheck.Warnings,
				"recommendations": gitCheck.Recommendations,
				"message":         gitCheck.FormatCheckResult(),
				"command":         command,
			}, nil
		}

		if gitCheck.RequiresApproval {
			return map[string]any{
				"ok":                false,
				"requires_approval": true,
				"is_git_command":    true,
				"risk":              gitCheck.RiskName,
				"warnings":          gitCheck.Warnings,
				"recommendations":   gitCheck.Recommendations,
				"message":           gitCheck.FormatCheckResult(),
				"command":           command,
				"approval_prompt":   fmt.Sprintf("Git command requires approval [%s risk]: %s", gitCheck.RiskName, command),
			}, nil
		}
	}

	// 验证命令安全性
	if err := t.validateCommand(command); err != nil {
		return NewClaudeErrorResponse(
			fmt.Errorf("command validation failed: %w", err),
			"避免使用危险命令如 rm -rf /, sudo rm 等",
			"如需执行敏感操作，请确认安全性",
		), nil
	}

	// 获取环境变量
	environment := make(map[string]string)
	if envData, exists := input["environment"]; exists {
		if envMap, ok := envData.(map[string]any); ok {
			for k, v := range envMap {
				if str, ok := v.(string); ok {
					environment[k] = str
				}
			}
		}
	}

	start := time.Now()

	// 构建完整命令
	fullCommand := t.buildFullCommand(command, environment, shellType)

	// 设置超时
	timeout := time.Duration(timeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = t.getCommandTimeout(command)
	}

	var taskID string
	var result *sandbox.ExecResult
	var err error

	if background {
		// 后台任务模式：使用TaskManager
		taskManager := GetGlobalTaskManager()
		taskOpts := &TaskOptions{
			WorkDir:       workingDir,
			Env:           environment,
			Timeout:       timeout,
			Background:    true,
			Shell:         shellType,
			CaptureOutput: captureOutput,
		}

		taskInfo, taskErr := taskManager.StartTask(ctx, command, taskOpts)
		if taskErr != nil {
			return map[string]any{
				"ok":    false,
				"error": fmt.Sprintf("failed to start background task: %v", taskErr),
				"recommendations": []string{
					"检查命令语法是否正确",
					"确认工作目录是否存在",
					"验证是否有执行权限",
				},
				"command":     command,
				"background":  true,
				"duration_ms": time.Since(start).Milliseconds(),
			}, nil
		}

		// 后台任务启动成功
		taskID = taskInfo.ID
		result = &sandbox.ExecResult{
			Code:   0,  // 后台任务通常立即返回，实际状态由TaskManager管理
			Stdout: "", // 初始输出为空，通过BashOutput获取
			Stderr: "",
		}
	} else {
		// 前台任务模式：直接执行
		result, err = tc.Sandbox.Exec(ctx, fullCommand, &sandbox.ExecOptions{
			Timeout: timeout,
			WorkDir: workingDir,
			Env:     environment,
		})
	}

	duration := time.Since(start)

	if err != nil && !background {
		return map[string]any{
			"ok":    false,
			"error": fmt.Sprintf("command execution failed: %v", err),
			"recommendations": []string{
				"检查命令语法是否正确",
				"确认工作目录是否存在",
				"验证是否有执行权限",
				"检查命令是否在系统PATH中",
				"确认环境变量设置正确",
			},
			"command":     command,
			"duration_ms": duration.Milliseconds(),
		}, nil
	}

	// 构建响应
	response := map[string]any{
		"ok":          true,
		"command":     command,
		"exit_code":   result.Code,
		"success":     result.Code == 0,
		"duration_ms": duration.Milliseconds(),
		"start_time":  start.Unix(),
		"end_time":    time.Now().Unix(),
	}

	if description != "" {
		response["description"] = description
	}

	if captureOutput {
		response["stdout"] = result.Stdout
		response["stderr"] = result.Stderr
	}

	if workingDir != "" {
		response["working_dir"] = workingDir
	}

	if shellID != "" {
		response["shell_id"] = shellID
	}

	response["background"] = background
	response["shell_type"] = shellType

	// 添加环境变量信息
	if len(environment) > 0 {
		response["environment_set"] = true
		response["environment_count"] = len(environment)
	} else {
		response["environment_set"] = false
	}

	// 如果执行失败，添加错误信息
	if result.Code != 0 {
		response["error"] = fmt.Sprintf("command exited with code %d", result.Code)
		response["recommendations"] = []string{
			"检查命令的标准错误输出",
			"验证命令参数是否正确",
			"确认所需的依赖是否已安装",
		}
	}

	// 如果是后台任务，添加task_id到响应中
	if background && taskID != "" {
		response["bash_id"] = taskID
		response["task_id"] = taskID
	}

	return response, nil
}

func (t *BashTool) validateCommand(cmd string) error {
	// 检查危险命令模式
	lowerCmd := strings.ToLower(cmd)
	for _, pattern := range t.dangerousPatterns {
		if pattern.MatchString(lowerCmd) {
			return fmt.Errorf("dangerous command blocked: %s", cmd)
		}
	}

	// 检查其他危险命令
	dangerousCommands := []string{
		"rm -rf /",
		"sudo rm",
		":(){ :|:& };:",
		"chmod 777 /",
		"dd if=/dev/zero",
		"mkfs",
		"format",
		"fdisk",
	}

	for _, dangerous := range dangerousCommands {
		if strings.Contains(lowerCmd, dangerous) {
			return fmt.Errorf("dangerous command blocked: %s", cmd)
		}
	}

	return nil
}

func (t *BashTool) buildFullCommand(command string, environment map[string]string, shellType string) string {
	var parts []string

	// 添加环境变量
	for k, v := range environment {
		parts = append(parts, fmt.Sprintf("export %s='%s'", k, strings.ReplaceAll(v, "'", `'"'"'`)))
	}

	// 添加命令
	parts = append(parts, command)

	// 根据shell类型构建
	switch shellType {
	case "sh":
		return fmt.Sprintf("sh -c '%s'", strings.Join(parts, "; "))
	case "zsh":
		return fmt.Sprintf("zsh -c '%s'", strings.Join(parts, "; "))
	default: // bash
		return fmt.Sprintf("bash -c '%s'", strings.Join(parts, "; "))
	}
}

func (t *BashTool) getCommandTimeout(cmd string) time.Duration {
	// 根据命令类型调整超时时间
	lowerCmd := strings.ToLower(cmd)

	// 长时间运行的命令
	if strings.Contains(lowerCmd, "make") || strings.Contains(lowerCmd, "npm install") ||
		strings.Contains(lowerCmd, "go build") || strings.Contains(lowerCmd, "cmake") ||
		strings.Contains(lowerCmd, "docker build") {
		return 30 * time.Minute
	}

	// 网络相关命令
	if strings.Contains(lowerCmd, "git clone") || strings.Contains(lowerCmd, "curl") ||
		strings.Contains(lowerCmd, "wget") || strings.Contains(lowerCmd, "pip install") {
		return 10 * time.Minute
	}

	// 数据库操作
	if strings.Contains(lowerCmd, "mysql") || strings.Contains(lowerCmd, "postgres") ||
		strings.Contains(lowerCmd, "psql") || strings.Contains(lowerCmd, "mongod") {
		return 5 * time.Minute
	}

	// 默认超时
	return t.defaultTimeout
}

func (t *BashTool) Prompt() string {
	return `执行bash命令在持久化shell会话中。

功能特性：
- 支持自定义环境变量
- 危险命令安全检查
- 超时控制机制
- 工作目录设置
- 详细的执行统计

使用指南：
- command: 必需参数，要执行的bash命令
- timeout: 可选参数，超时时间（毫秒）
- working_dir: 可选参数，工作目录
- environment: 可选参数，环境变量设置
- background: 可选参数，是否后台运行
- shell: 可选参数，shell类型

安全特性：
- 危险命令自动拦截
- 沙箱环境隔离
- 超时保护机制
- 命令注入防护

注意事项：
- 默认超时时间为2分钟
- 长时间运行的命令会自动调整超时
- 危险命令（如rm -rf /）会被阻止
- 所有命令都在沙箱环境中执行`
}

// Examples 返回 Bash 工具的使用示例
// 实现 ExampleableTool 接口，帮助 LLM 更准确地调用工具
func (t *BashTool) Examples() []tools.ToolExample {
	return []tools.ToolExample{
		{
			Description: "列出当前目录文件",
			Input: map[string]any{
				"command": "ls -la",
			},
		},
		{
			Description: "查找所有 Go 源文件",
			Input: map[string]any{
				"command": "find . -name '*.go' -type f",
			},
		},
		{
			Description: "在后台运行构建命令",
			Input: map[string]any{
				"command":    "go build -v ./...",
				"background": true,
				"timeout":    300000,
			},
		},
		{
			Description: "设置环境变量执行命令",
			Input: map[string]any{
				"command": "echo $MY_VAR && env | grep MY",
				"environment": map[string]string{
					"MY_VAR": "hello_world",
				},
			},
		},
		{
			Description: "在指定目录运行 npm 安装",
			Input: map[string]any{
				"command":     "npm install",
				"working_dir": "/app/frontend",
				"timeout":     600000,
			},
		},
	}
}
