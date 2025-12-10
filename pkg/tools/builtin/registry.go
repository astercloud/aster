package builtin

import "github.com/astercloud/aster/pkg/tools"

// RegisterAll 注册所有内置工具 （重要：克制，未经严格的讨论禁止再增加）
// 保持精简（约17个工具）
func RegisterAll(registry *tools.Registry) {
	// 文件操作工具 (5)
	registry.Register("Read", NewReadTool)
	registry.Register("Write", NewWriteTool)
	registry.Register("Edit", NewEditTool)
	registry.Register("Glob", NewGlobTool)
	registry.Register("Grep", NewGrepTool)

	// 命令行执行工具 (3)
	registry.Register("Bash", NewBashTool)
	registry.Register("BashOutput", NewBashOutputTool)
	registry.Register("KillShell", NewKillShellTool)

	// 智能代理工具 (1)
	registry.Register("Task", NewTaskTool)

	// 规划管理工具 (3)
	registry.Register("TodoWrite", NewTodoWriteTool)
	registry.Register("EnterPlanMode", NewEnterPlanModeTool)
	registry.Register("ExitPlanMode", NewExitPlanModeTool)

	// 用户交互工具 (1)
	registry.Register("AskUserQuestion", NewAskUserQuestionTool)

	// 网络工具 (2)
	registry.Register("WebFetch", NewWebFetchTool)
	registry.Register("WebSearch", NewWebSearchTool)

	// 技能工具 (1)
	registry.Register("Skill", NewSkillTool)
}

// FileSystemTools 返回文件系统工具列表
func FileSystemTools() []string {
	return []string{"Read", "Write", "Edit", "Glob", "Grep"}
}

// ExecutionTools 返回执行工具列表
func ExecutionTools() []string {
	return []string{"Bash", "BashOutput", "KillShell"}
}

// AgentTools 返回智能代理工具列表
func AgentTools() []string {
	return []string{"Task"}
}

// PlanningTools 返回规划管理工具列表
func PlanningTools() []string {
	return []string{"TodoWrite", "EnterPlanMode", "ExitPlanMode"}
}

// InteractionTools 返回用户交互工具列表
func InteractionTools() []string {
	return []string{"AskUserQuestion"}
}

// NetworkTools 返回网络工具列表
func NetworkTools() []string {
	return []string{"WebFetch", "WebSearch"}
}

// SkillTools 返回技能工具列表
func SkillTools() []string {
	return []string{"Skill"}
}

// AllTools 返回所有内置工具列表（共16个）
func AllTools() []string {
	tools := FileSystemTools()
	tools = append(tools, ExecutionTools()...)
	tools = append(tools, AgentTools()...)
	tools = append(tools, PlanningTools()...)
	tools = append(tools, InteractionTools()...)
	tools = append(tools, NetworkTools()...)
	tools = append(tools, SkillTools()...)
	return tools
}
