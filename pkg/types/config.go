package types

import "time"

// PermissionMode 权限模式
type PermissionMode string

const (
	PermissionModeAuto     PermissionMode = "auto"     // 自动决策
	PermissionModeApproval PermissionMode = "approval" // 全部需要审批
	PermissionModeAllow    PermissionMode = "allow"    // 全部允许
)

// PermissionConfig 权限配置
type PermissionConfig struct {
	Mode  PermissionMode `json:"mode"`
	Allow []string       `json:"allow,omitempty"` // 白名单工具
	Deny  []string       `json:"deny,omitempty"`  // 黑名单工具
	Ask   []string       `json:"ask,omitempty"`   // 需要审批的工具
}

// TodoConfig Todo功能配置
type TodoConfig struct {
	Enabled             bool `json:"enabled"`
	ReminderOnStart     bool `json:"reminder_on_start"`
	RemindIntervalSteps int  `json:"remind_interval_steps"`
}

// SubAgentConfig 子Agent配置
type SubAgentConfig struct {
	Depth         int                   `json:"depth"`
	Templates     []string              `json:"templates,omitempty"`
	InheritConfig bool                  `json:"inherit_config"`
	Overrides     *AgentConfigOverrides `json:"overrides,omitempty"`
}

// AgentConfigOverrides Agent配置覆盖
type AgentConfigOverrides struct {
	Permission *PermissionConfig `json:"permission,omitempty"`
	Todo       *TodoConfig       `json:"todo,omitempty"`
}

// ContextManagerOptions 上下文管理配置
type ContextManagerOptions struct {
	MaxTokens         int    `json:"max_tokens"`
	CompressToTokens  int    `json:"compress_to_tokens"`
	CompressionModel  string `json:"compression_model,omitempty"`
	EnableCompression bool   `json:"enable_compression"`
}

// ToolsManualConfig 控制工具手册的注入策略(用于减少 System Prompt 膨胀)。
type ToolsManualConfig struct {
	// Mode 决定哪些工具会出现在 System Prompt 的 "Tools Manual" 中:
	// - "all"   : 默认值, 所有工具都会注入(除非在 Exclude 中显式排除)
	// - "listed": 仅注入 Include 列表中出现的工具
	// - "none"  : 完全不注入工具手册, 由模型自己通过名称和输入 Schema 推断
	Mode string `json:"mode,omitempty"`

	// Include 仅在 Mode 为 "listed" 时生效, 指定要注入手册的工具名称白名单。
	Include []string `json:"include,omitempty"`

	// Exclude 在 Mode 为 "all" 时生效, 指定不注入手册的工具名称黑名单。
	Exclude []string `json:"exclude,omitempty"`
}

// AgentTemplateRuntime Agent模板运行时配置
type AgentTemplateRuntime struct {
	ExposeThinking     bool                   `json:"expose_thinking,omitempty"`
	Todo               *TodoConfig            `json:"todo,omitempty"`
	SubAgents          *SubAgentConfig        `json:"subagents,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	ToolTimeoutMs      int                    `json:"tool_timeout_ms,omitempty"`
	MaxToolConcurrency int                    `json:"max_tool_concurrency,omitempty"`
	ToolsManual        *ToolsManualConfig     `json:"tools_manual,omitempty"`
}

// AgentTemplateDefinition Agent模板定义
type AgentTemplateDefinition struct {
	ID           string                `json:"id"`
	Version      string                `json:"version,omitempty"`
	SystemPrompt string                `json:"system_prompt"`
	Model        string                `json:"model,omitempty"`
	Tools        interface{}           `json:"tools"` // []string or "*"
	Permission   *PermissionConfig     `json:"permission,omitempty"`
	Runtime      *AgentTemplateRuntime `json:"runtime,omitempty"`
}

// ModelConfig 模型配置
type ModelConfig struct {
	Provider      string        `json:"provider"` // "anthropic", "openai", etc.
	Model         string        `json:"model"`
	APIKey        string        `json:"api_key,omitempty"`
	BaseURL       string        `json:"base_url,omitempty"`
	ExecutionMode ExecutionMode `json:"execution_mode,omitempty"` // 执行模式：streaming/non-streaming/auto
}

// SandboxKind 沙箱类型
type SandboxKind string

const (
	SandboxKindLocal      SandboxKind = "local"
	SandboxKindDocker     SandboxKind = "docker"
	SandboxKindK8s        SandboxKind = "k8s"
	SandboxKindAliyun     SandboxKind = "aliyun"
	SandboxKindVolcengine SandboxKind = "volcengine"
	SandboxKindRemote     SandboxKind = "remote"
	SandboxKindMock       SandboxKind = "mock"
)

// SandboxConfig 沙箱配置
type SandboxConfig struct {
	Kind            SandboxKind            `json:"kind"`
	WorkDir         string                 `json:"work_dir,omitempty"`
	EnforceBoundary bool                   `json:"enforce_boundary,omitempty"`
	AllowPaths      []string               `json:"allow_paths,omitempty"`
	WatchFiles      bool                   `json:"watch_files,omitempty"`
	Extra           map[string]interface{} `json:"extra,omitempty"` // 云平台特定配置
}

// CloudCredentials 云平台凭证
type CloudCredentials struct {
	AccessKeyID     string `json:"access_key_id,omitempty"`
	AccessKeySecret string `json:"access_key_secret,omitempty"`
	Token           string `json:"token,omitempty"`
}

// ResourceLimits 资源限制
type ResourceLimits struct {
	CPUQuota    float64       `json:"cpu_quota,omitempty"`    // CPU配额(核数)
	MemoryLimit int64         `json:"memory_limit,omitempty"` // 内存限制(字节)
	Timeout     time.Duration `json:"timeout,omitempty"`      // 超时时间
	DiskQuota   int64         `json:"disk_quota,omitempty"`   // 磁盘配额(字节)
}

// CloudSandboxConfig 云沙箱配置
type CloudSandboxConfig struct {
	Provider    string           `json:"provider"` // "aliyun", "volcengine"
	Region      string           `json:"region"`
	Credentials CloudCredentials `json:"credentials"`
	SessionID   string           `json:"session_id,omitempty"`
	Resources   ResourceLimits   `json:"resources,omitempty"`
}

// SkillsPackageConfig Skills 包配置
type SkillsPackageConfig struct {
	// 技能包来源
	Source  string `json:"source"`  // "local" | "oss" | "s3" | "hybrid"
	Path    string `json:"path"`    // 本地路径或云端 URL
	Version string `json:"version"` // 版本号

	// 命令和技能目录
	CommandsDir string `json:"commands_dir"` // 默认 "commands"
	SkillsDir   string `json:"skills_dir"`   // 默认 "skills"

	// 启用的 commands 和 skills
	EnabledCommands []string `json:"enabled_commands"` // ["write", "analyze", ...]
	EnabledSkills   []string `json:"enabled_skills"`   // ["consistency-checker", ...]
}

// AgentConfig Agent创建配置
type AgentConfig struct {
	AgentID          string                            `json:"agent_id,omitempty"`
	TemplateID       string                            `json:"template_id"`
	TemplateVersion  string                            `json:"template_version,omitempty"`
	ModelConfig      *ModelConfig                      `json:"model_config,omitempty"`
	Sandbox          *SandboxConfig                    `json:"sandbox,omitempty"`
	Tools            []string                          `json:"tools,omitempty"`
	Middlewares      []string                          `json:"middlewares,omitempty"`       // Middleware 列表 (Phase 6C)
	MiddlewareConfig map[string]map[string]interface{} `json:"middleware_config,omitempty"` // 各中间件的自定义配置
	ExposeThinking   bool                              `json:"expose_thinking,omitempty"`
	// RoutingProfile 可选的路由配置标识，例如 "quality-first"、"cost-first"。
	// 当配置了 Router 时，可以根据该字段选择不同的模型路由策略。
	RoutingProfile string                 `json:"routing_profile,omitempty"`
	Overrides      *AgentConfigOverrides  `json:"overrides,omitempty"`
	Context        *ContextManagerOptions `json:"context,omitempty"`
	SkillsPackage  *SkillsPackageConfig   `json:"skills_package,omitempty"` // Skills 包配置
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ResumeStrategy 恢复策略
type ResumeStrategy string

const (
	ResumeStrategyCrash  ResumeStrategy = "crash"  // 自动封口未完成工具
	ResumeStrategyManual ResumeStrategy = "manual" // 手动处理
)

// ResumeOptions 恢复选项
type ResumeOptions struct {
	Strategy  ResumeStrategy `json:"strategy,omitempty"`
	AutoRun   bool           `json:"auto_run,omitempty"`
	Overrides *AgentConfig   `json:"overrides,omitempty"`
}

// SendOptions 发送消息选项
type SendOptions struct {
	Kind     string           `json:"kind,omitempty"` // "user" or "reminder"
	Reminder *ReminderOptions `json:"reminder,omitempty"`
}

// ReminderOptions 提醒选项
type ReminderOptions struct {
	SkipStandardEnding bool   `json:"skip_standard_ending,omitempty"`
	Priority           string `json:"priority,omitempty"` // "low", "medium", "high"
	Category           string `json:"category,omitempty"` // "file", "todo", "security", "performance", "general"
}

// StreamOptions 流式订阅选项
type StreamOptions struct {
	Since *Bookmark `json:"since,omitempty"`
	Kinds []string  `json:"kinds,omitempty"` // 事件类型过滤
}

// SubscribeOptions 订阅选项
type SubscribeOptions struct {
	Since    *Bookmark      `json:"since,omitempty"`
	Kinds    []string       `json:"kinds,omitempty"`
	Channels []AgentChannel `json:"channels,omitempty"`
}

// CompleteResult 完成结果
type CompleteResult struct {
	Status        string    `json:"status"` // "ok" or "paused"
	Text          string    `json:"text,omitempty"`
	Last          *Bookmark `json:"last,omitempty"`
	PermissionIDs []string  `json:"permission_ids,omitempty"`
}

// ExecutionMode 执行模式
type ExecutionMode string

const (
	// ExecutionModeStreaming 流式模式（默认，实时反馈）
	ExecutionModeStreaming ExecutionMode = "streaming"
	// ExecutionModeNonStreaming 非流式模式（快速，批量处理）
	ExecutionModeNonStreaming ExecutionMode = "non-streaming"
	// ExecutionModeAuto 自动选择（根据任务类型智能选择）
	ExecutionModeAuto ExecutionMode = "auto"
)
