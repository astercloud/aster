package skills

// SkillDefinition Skill 定义
type SkillDefinition struct {
	// 基础信息
	Name         string   // 技能名 (YAML: name)
	Description  string   // 描述   (YAML: description)
	AllowedTools []string // 允许使用的工具 (YAML: allowed-tools)

	// 位置相关信息（用于在提示词中给出 SKILL.md 路径提示）
	// Path    : 相对于 Skills 根目录的技能路径，例如 "pdfmd" 或 "workflow/consistency-checker"
	// BaseDir : Skills 根目录，相对于沙箱工作目录，例如 "skills" 或 "workspace/skills"
	Path    string
	BaseDir string

	// 类型:
	// - 为空或 "knowledge": 只注入知识
	// - "executable": 可执行 Skill, 搭配 Executable 配置使用
	Kind string

	// 知识库内容
	KnowledgeBase string // 注入到 SystemPrompt 的内容

	// 可执行 Skill 的参数和返回值定义 (可选)
	Parameters map[string]ParamSpec
	Returns    map[string]ReturnSpec

	// 可执行配置 (可选)
	Executable *ExecutableConfig

	// 触发条件
	Triggers []TriggerConfig
}

// TriggerConfig 触发配置
type TriggerConfig struct {
	Type      string   // "keyword" | "context" | "always" | "file_pattern"
	Keywords  []string // 关键词列表 (type=keyword)
	Condition string   // 条件描述   (type=context)
	Pattern   string   // 文件模式   (type=file_pattern)，例如 "**/*.go"
}

// SkillContext 技能上下文
type SkillContext struct {
	UserMessage string         // 用户输入
	Command     string         // 当前命令（如 "/write"）
	Files       []string       // 涉及的文件
	Metadata    map[string]any // 额外元数据
}

// ParamSpec 参数定义
type ParamSpec struct {
	Type        string   // 参数类型，如 "string" | "number" | "boolean" | "object"
	Description string   // 参数说明
	Required    bool     // 是否必填
	Enum        []string // 枚举值（可选）
}

// ReturnSpec 返回值定义
type ReturnSpec struct {
	Type        string // 返回值类型
	Description string // 返回值说明
}

// ExecutableConfig 可执行 Skill 配置
type ExecutableConfig struct {
	// Runtime 运行时类型，例如 "bash"、"go"、"python"
	Runtime string

	// Entry 入口脚本或命令，例如 "scripts/pdf2md.go"
	Entry string

	// TimeoutSeconds 超时时间（秒），0 表示使用默认
	TimeoutSeconds int
}
