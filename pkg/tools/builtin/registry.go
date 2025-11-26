package builtin

import "github.com/astercloud/aster/pkg/tools"

// RegisterAll 注册所有内置工具
func RegisterAll(registry *tools.Registry) {
	// 文件操作工具
	registry.Register("Read", NewReadTool)
	registry.Register("Write", NewWriteTool)
	registry.Register("Edit", NewEditTool)
	registry.Register("Glob", NewGlobTool)
	registry.Register("Grep", NewGrepTool)

	// 执行工具
	registry.Register("Bash", NewBashTool)
	registry.Register("CodeExecute", NewCodeExecuteTool)

	// 网络工具
	registry.Register("HttpRequest", NewHttpRequestTool)
	registry.Register("WebSearch", NewWebSearchTool)

	// Skills 工具
	registry.Register("Skill", NewSkillTool)

	// 任务管理工具
	registry.Register("TodoWrite", NewTodoWriteTool)
	registry.Register("BashOutput", NewBashOutputTool)
	registry.Register("KillShell", NewKillShellTool)
	registry.Register("Task", NewTaskTool)
	registry.Register("ExitPlanMode", NewExitPlanModeTool)
	registry.Register("EnterPlanMode", NewEnterPlanModeTool)
	registry.Register("DemoLongTask", NewDemoLongTaskTool)

	// 用户交互工具
	registry.Register("AskUserQuestion", NewAskUserQuestionTool)

	// 语义搜索工具
	registry.Register("SemanticSearch", NewSemanticSearchTool)

	// 工具搜索工具（Tool Search Tool）
	registry.Register("ToolSearch", NewToolSearchTool)
}

// FileSystemTools 返回文件系统工具列表
func FileSystemTools() []string {
	return []string{"Read", "Write", "Edit", "Glob", "Grep"}
}

// ExecutionTools 返回执行工具列表
func ExecutionTools() []string {
	return []string{"Bash", "CodeExecute"}
}

// NetworkTools 返回网络工具列表
func NetworkTools() []string {
	return []string{"HttpRequest", "WebSearch"}
}

// SkillTools 返回技能工具列表
func SkillTools() []string {
	return []string{"Skill"}
}

// TaskManagementTools 返回任务管理工具列表
func TaskManagementTools() []string {
	return []string{"TodoWrite", "BashOutput", "KillShell", "Task", "ExitPlanMode", "EnterPlanMode"}
}

// InteractionTools 返回用户交互工具列表
func InteractionTools() []string {
	return []string{"AskUserQuestion"}
}

// SemanticTools 返回语义工具列表
func SemanticTools() []string {
	return []string{"SemanticSearch"}
}

// DiscoveryTools 返回工具发现相关的工具列表
func DiscoveryTools() []string {
	return []string{"ToolSearch"}
}

// AllTools 返回所有内置工具列表
func AllTools() []string {
	tools := FileSystemTools()
	tools = append(tools, ExecutionTools()...)
	tools = append(tools, NetworkTools()...)
	tools = append(tools, SkillTools()...)
	tools = append(tools, TaskManagementTools()...)
	tools = append(tools, InteractionTools()...)
	tools = append(tools, SemanticTools()...)
	tools = append(tools, DiscoveryTools()...)
	return tools
}
