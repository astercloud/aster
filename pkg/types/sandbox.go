// Package types provides sandbox configuration types aligned with Claude Agent SDK.
package types

import "context"

// SandboxSettings 沙箱安全设置 (Claude Agent SDK 风格)
// 这是对 SandboxConfig 的增强，提供更细粒度的安全控制
type SandboxSettings struct {
	// Enabled 是否启用沙箱隔离
	// 当为 true 时，所有命令执行都在沙箱环境中运行
	Enabled bool `json:"enabled,omitempty"`

	// AutoAllowBashIfSandboxed 沙箱启用时自动批准 bash 命令
	// 当沙箱启用时，bash 命令被认为是安全的，可以自动批准
	AutoAllowBashIfSandboxed bool `json:"auto_allow_bash_if_sandboxed,omitempty"`

	// ExcludedCommands 静态白名单，这些命令绕过沙箱限制
	// 例如: ["docker", "git"] - 这些命令将直接执行，不经过沙箱
	ExcludedCommands []string `json:"excluded_commands,omitempty"`

	// AllowUnsandboxedCommands 允许模型请求绕过沙箱
	// 当为 true 时，模型可以在工具输入中设置 dangerouslyDisableSandbox: true
	// 这些请求会回退到权限系统进行审批
	AllowUnsandboxedCommands bool `json:"allow_unsandboxed_commands,omitempty"`

	// Network 网络沙箱配置
	Network *NetworkSandboxSettings `json:"network,omitempty"`

	// IgnoreViolations 忽略特定违规
	IgnoreViolations *SandboxIgnoreViolations `json:"ignore_violations,omitempty"`

	// EnableWeakerNestedSandbox 启用较弱的嵌套沙箱（兼容性）
	EnableWeakerNestedSandbox bool `json:"enable_weaker_nested_sandbox,omitempty"`
}

// NetworkSandboxSettings 网络沙箱配置
type NetworkSandboxSettings struct {
	// AllowLocalBinding 允许进程绑定本地端口（如开发服务器）
	AllowLocalBinding bool `json:"allow_local_binding,omitempty"`

	// AllowUnixSockets 允许访问的 Unix Socket 路径
	// 例如: ["/var/run/docker.sock"]
	AllowUnixSockets []string `json:"allow_unix_sockets,omitempty"`

	// AllowAllUnixSockets 允许访问所有 Unix Socket
	AllowAllUnixSockets bool `json:"allow_all_unix_sockets,omitempty"`

	// HTTPProxyPort HTTP 代理端口
	HTTPProxyPort int `json:"http_proxy_port,omitempty"`

	// SOCKSProxyPort SOCKS 代理端口
	SOCKSProxyPort int `json:"socks_proxy_port,omitempty"`

	// AllowedHosts 允许访问的主机列表
	AllowedHosts []string `json:"allowed_hosts,omitempty"`

	// BlockedHosts 禁止访问的主机列表
	BlockedHosts []string `json:"blocked_hosts,omitempty"`
}

// SandboxIgnoreViolations 忽略特定沙箱违规
type SandboxIgnoreViolations struct {
	// FilePatterns 忽略的文件路径模式
	// 例如: ["/tmp/*", "*.log"]
	FilePatterns []string `json:"file_patterns,omitempty"`

	// NetworkPatterns 忽略的网络模式
	// 例如: ["localhost:*", "127.0.0.1:*"]
	NetworkPatterns []string `json:"network_patterns,omitempty"`
}

// SandboxViolation 沙箱违规记录
type SandboxViolation struct {
	Type      string `json:"type"`      // "file" | "network" | "process"
	Path      string `json:"path"`      // 违规路径或地址
	Operation string `json:"operation"` // 操作类型
	Blocked   bool   `json:"blocked"`   // 是否被阻止
	Timestamp int64  `json:"timestamp"` // 时间戳
	Details   string `json:"details"`   // 详细信息
}

// CanUseToolFunc 自定义权限检查函数类型 (Claude Agent SDK 风格)
// 应用层可以通过此回调完全控制工具权限
type CanUseToolFunc func(
	ctx context.Context,
	toolName string,
	input map[string]any,
	opts *CanUseToolOptions,
) (*PermissionResult, error)

// CanUseToolOptions 权限检查选项
type CanUseToolOptions struct {
	// Signal 用于取消操作的上下文
	Signal context.Context

	// Suggestions 建议的权限更新
	Suggestions []PermissionUpdate

	// SandboxEnabled 沙箱是否启用
	SandboxEnabled bool

	// BypassSandboxRequested 是否请求绕过沙箱
	BypassSandboxRequested bool
}

// PermissionResult 权限检查结果
type PermissionResult struct {
	// Behavior 行为: "allow" | "deny"
	Behavior string `json:"behavior"`

	// UpdatedInput 修改后的输入参数
	// 权限系统可以修改工具输入（如脱敏、添加限制等）
	UpdatedInput map[string]any `json:"updated_input,omitempty"`

	// UpdatedPermissions 权限更新操作
	UpdatedPermissions []PermissionUpdate `json:"updated_permissions,omitempty"`

	// Message 拒绝原因或说明
	Message string `json:"message,omitempty"`

	// Interrupt 是否中断执行
	Interrupt bool `json:"interrupt,omitempty"`
}

// PermissionUpdate 权限更新操作
type PermissionUpdate struct {
	// Type 更新类型: "addRules" | "replaceRules" | "removeRules" | "setMode"
	Type string `json:"type"`

	// Rules 规则列表
	Rules []PermissionRule `json:"rules,omitempty"`

	// Behavior 行为: "allow" | "deny" | "ask"
	Behavior string `json:"behavior,omitempty"`

	// Destination 目标: "session" | "project" | "user"
	Destination string `json:"destination,omitempty"`

	// Mode 权限模式（用于 setMode 类型）
	Mode string `json:"mode,omitempty"`
}

// PermissionRule 权限规则
type PermissionRule struct {
	// ToolName 工具名称
	ToolName string `json:"tool_name"`

	// RuleContent 规则内容（可选的额外条件）
	RuleContent string `json:"rule_content,omitempty"`
}

// SandboxPermissionMode 沙箱权限模式 (Claude Agent SDK 风格)
type SandboxPermissionMode string

const (
	// SandboxPermissionDefault 默认权限行为
	SandboxPermissionDefault SandboxPermissionMode = "default"

	// SandboxPermissionAcceptEdits 自动接受文件编辑
	SandboxPermissionAcceptEdits SandboxPermissionMode = "acceptEdits"

	// SandboxPermissionBypass 绕过所有权限检查
	SandboxPermissionBypass SandboxPermissionMode = "bypassPermissions"

	// SandboxPermissionPlan 规划模式 - 不执行
	SandboxPermissionPlan SandboxPermissionMode = "plan"
)
