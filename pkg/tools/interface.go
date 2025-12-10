package tools

import (
	"context"

	"github.com/astercloud/aster/pkg/sandbox"
)

// MCPManagerInterface MCP 管理器接口
// 用于工具访问 MCP 服务器和资源
type MCPManagerInterface interface {
	// ListServers 列出所有 MCP 服务器 ID
	ListServers() []string
	// GetServer 获取指定的 MCP 服务器
	GetServer(serverID string) (any, bool)
}

// ToolContext 工具执行上下文
type ToolContext struct {
	AgentID    string
	Sandbox    sandbox.Sandbox
	Signal     context.Context
	Reporter   Reporter
	Emit       func(eventType string, data any) // Deprecated: use Reporter
	Services   map[string]any
	ThreadID   string               // Working Memory 会话 ID
	ResourceID string               // Working Memory 资源 ID
	MCPManager MCPManagerInterface  // MCP 管理器，用于访问 MCP 资源
}

// Reporter 工具执行实时反馈接口
type Reporter interface {
	Progress(progress float64, message string, step, total int, metadata map[string]any, etaMs int64)
	Intermediate(label string, data any)
}

// Interruptible 可中断/恢复的工具接口
type Interruptible interface {
	Pause() error
	Resume() error
	Cancel() error
}

// Tool 工具接口
type Tool interface {
	// Name 工具名称
	Name() string

	// Description 工具描述
	Description() string

	// InputSchema JSON Schema定义
	InputSchema() map[string]any

	// Execute 执行工具
	Execute(ctx context.Context, input map[string]any, tc *ToolContext) (any, error)

	// Prompt 工具使用说明(可选)
	Prompt() string
}

// ToolExample 工具使用示例
// 用于向 LLM 展示工具的正确使用方式，提升复杂参数处理的准确率
type ToolExample struct {
	// Description 示例描述，说明这个示例演示的场景
	Description string `json:"description"`

	// Input 示例输入参数
	Input map[string]any `json:"input"`

	// Output 可选的预期输出，用于展示工具返回格式
	Output any `json:"output,omitempty"`
}

// ExampleableTool 支持使用示例的工具接口
// 实现此接口的工具可以提供使用示例，帮助 LLM 更准确地调用工具
type ExampleableTool interface {
	Tool
	// Examples 返回工具使用示例列表
	// 建议提供 1-5 个示例，涵盖常见使用场景
	Examples() []ToolExample
}

// DeferrableConfig 延迟加载配置
// 用于工具搜索工具的按需发现机制
type DeferrableConfig struct {
	// DeferLoading 是否延迟加载，为 true 时工具不会预先加载到 LLM 上下文
	DeferLoading bool `json:"defer_loading"`

	// Category 工具分类，用于搜索过滤
	// 例如: "filesystem", "execution", "network", "mcp", "custom"
	Category string `json:"category,omitempty"`

	// Keywords 搜索关键词，用于 BM25 索引
	Keywords []string `json:"keywords,omitempty"`
}

// DeferrableTool 支持延迟加载的工具接口
// 实现此接口的工具可以被工具搜索工具按需发现和激活
type DeferrableTool interface {
	Tool
	// DeferConfig 返回延迟加载配置
	DeferConfig() *DeferrableConfig
}

// ToolConfig 工具配置(用于持久化)
type ToolConfig struct {
	Name       string         `json:"name"`
	RegistryID string         `json:"registry_id,omitempty"`
	Config     map[string]any `json:"config,omitempty"`
}

// ToolFactory 工具工厂函数
type ToolFactory func(config map[string]any) (Tool, error)

// Registry 工具注册表
type Registry struct {
	factories map[string]ToolFactory
}

// NewRegistry 创建工具注册表
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]ToolFactory),
	}
}

// Register 注册工具
func (r *Registry) Register(name string, factory ToolFactory) {
	r.factories[name] = factory
}

// Create 创建工具实例
func (r *Registry) Create(name string, config map[string]any) (Tool, error) {
	factory, ok := r.factories[name]
	if !ok {
		return nil, &ToolNotFoundError{Name: name}
	}

	return factory(config)
}

// List 列出所有已注册的工具
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// Has 检查工具是否已注册
func (r *Registry) Has(name string) bool {
	_, ok := r.factories[name]
	return ok
}

// ToolNotFoundError 工具未找到错误
type ToolNotFoundError struct {
	Name string
}

func (e *ToolNotFoundError) Error() string {
	return "tool not found: " + e.Name
}
