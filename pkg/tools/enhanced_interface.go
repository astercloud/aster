package tools

import (
	"context"
	"time"

	"github.com/astercloud/aster/pkg/session"
	"github.com/astercloud/aster/pkg/types"
)

// EnhancedTool 增强的工具接口
// 参考 Google ADK-Go 的工具设计
type EnhancedTool interface {
	Tool // 继承基础接口

	// IsLongRunning 是否为长时运行工具
	// 长时运行工具会先返回资源 ID，稍后完成操作
	IsLongRunning() bool

	// Timeout 工具执行超时时间
	// 返回 0 表示使用默认超时
	Timeout() time.Duration

	// RequiresApproval 是否需要人工审批
	RequiresApproval() bool

	// Priority 工具优先级 (用于并发控制)
	// 数值越大优先级越高
	Priority() int

	// RetryPolicy 重试策略
	RetryPolicy() *RetryPolicy

	// Metadata 工具元数据
	Metadata() map[string]any
}

// RetryPolicy 重试策略
type RetryPolicy struct {
	// MaxRetries 最大重试次数
	MaxRetries int

	// InitialBackoff 初始退避时间
	InitialBackoff time.Duration

	// MaxBackoff 最大退避时间
	MaxBackoff time.Duration

	// BackoffMultiplier 退避倍数
	BackoffMultiplier float64

	// RetryableErrors 可重试的错误类型
	RetryableErrors []string
}

// EnhancedContext 增强的工具执行上下文
// 参考 Google ADK-Go 的 tool.Context 设计
type EnhancedContext interface {
	Context // 继承基础接口

	// CallID 工具调用的唯一标识符
	CallID() string

	// AgentID 当前 Agent ID
	AgentID() string

	// SessionID 当前会话 ID
	SessionID() string

	// InvocationID 当前调用 ID
	InvocationID() string

	// State 访问会话状态
	State() session.State

	// Actions 获取事件动作
	Actions() *session.EventActions

	// SearchMemory 搜索 Agent 记忆
	SearchMemory(ctx context.Context, query string) ([]MemoryResult, error)

	// GetArtifact 获取 Artifact
	GetArtifact(ctx context.Context, name string) (any, error)

	// SaveArtifact 保存 Artifact
	SaveArtifact(ctx context.Context, name string, data any) error

	// EmitEvent 发送自定义事件
	EmitEvent(event *types.Event) error

	// Logger 获取日志记录器
	Logger() Logger

	// Tracer 获取追踪器
	Tracer() Tracer

	// Metrics 获取指标收集器
	Metrics() Metrics
}

// MemoryResult 记忆搜索结果
type MemoryResult struct {
	Content   string
	Score     float64
	Timestamp time.Time
	Metadata  map[string]any
}

// Logger 日志接口
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
}

// Field 日志字段
type Field struct {
	Key   string
	Value any
}

// Tracer 追踪接口 (简化版)
type Tracer interface {
	StartSpan(ctx context.Context, name string) (context.Context, Span)
}

// Span 追踪 span 接口
type Span interface {
	End()
	SetAttribute(key string, value any)
	RecordError(err error)
}

// Metrics 指标接口 (简化版)
type Metrics interface {
	IncrementCounter(name string, value int64)
	RecordDuration(name string, duration time.Duration)
}

// Toolset 工具集接口
// 参考 Google ADK-Go 的 Toolset 设计
type Toolset interface {
	// Name 工具集名称
	Name() string

	// Description 工具集描述
	Description() string

	// Tools 返回工具列表
	// 可以根据上下文动态决定返回哪些工具
	Tools(ctx ReadonlyContext) ([]Tool, error)

	// Initialize 初始化工具集
	Initialize(ctx context.Context) error

	// Cleanup 清理工具集资源
	Cleanup(ctx context.Context) error
}

// ReadonlyContext 只读上下文
type ReadonlyContext interface {
	AgentID() string
	SessionID() string
	State() session.ReadonlyState
}

// Predicate 工具过滤谓词
// 用于动态决定是否暴露某个工具给 LLM
type Predicate func(ctx ReadonlyContext, tool Tool) bool

// StringPredicate 基于工具名称的过滤器
func StringPredicate(allowedTools []string) Predicate {
	allowed := make(map[string]bool)
	for _, name := range allowedTools {
		allowed[name] = true
	}

	return func(ctx ReadonlyContext, tool Tool) bool {
		return allowed[tool.Name()]
	}
}

// PrefixPredicate 基于前缀的过滤器
func PrefixPredicate(prefix string) Predicate {
	return func(ctx ReadonlyContext, tool Tool) bool {
		name := tool.Name()
		return len(name) >= len(prefix) && name[:len(prefix)] == prefix
	}
}

// AndPredicate 组合多个谓词 (AND)
func AndPredicate(predicates ...Predicate) Predicate {
	return func(ctx ReadonlyContext, tool Tool) bool {
		for _, p := range predicates {
			if !p(ctx, tool) {
				return false
			}
		}
		return true
	}
}

// OrPredicate 组合多个谓词 (OR)
func OrPredicate(predicates ...Predicate) Predicate {
	return func(ctx ReadonlyContext, tool Tool) bool {
		for _, p := range predicates {
			if p(ctx, tool) {
				return true
			}
		}
		return false
	}
}

// NotPredicate 取反谓词
func NotPredicate(predicate Predicate) Predicate {
	return func(ctx ReadonlyContext, tool Tool) bool {
		return !predicate(ctx, tool)
	}
}

// BaseEnhancedTool 增强工具的基础实现
// 提供默认行为，简化自定义工具开发
type BaseEnhancedTool struct {
	name            string
	description     string
	schema          *types.ToolSchema
	isLongRunning   bool
	timeout         time.Duration
	requireApproval bool
	priority        int
	retryPolicy     *RetryPolicy
	metadata        map[string]any
}

func NewBaseEnhancedTool(name, description string) *BaseEnhancedTool {
	return &BaseEnhancedTool{
		name:        name,
		description: description,
		timeout:     30 * time.Second, // 默认 30 秒
		priority:    100,              // 默认优先级
		metadata:    make(map[string]any),
	}
}

func (t *BaseEnhancedTool) Name() string              { return t.name }
func (t *BaseEnhancedTool) Description() string       { return t.description }
func (t *BaseEnhancedTool) Schema() *types.ToolSchema { return t.schema }
func (t *BaseEnhancedTool) IsLongRunning() bool       { return t.isLongRunning }
func (t *BaseEnhancedTool) Timeout() time.Duration    { return t.timeout }
func (t *BaseEnhancedTool) RequiresApproval() bool    { return t.requireApproval }
func (t *BaseEnhancedTool) Priority() int             { return t.priority }
func (t *BaseEnhancedTool) RetryPolicy() *RetryPolicy { return t.retryPolicy }
func (t *BaseEnhancedTool) Metadata() map[string]any  { return t.metadata }

// Setter 方法
func (t *BaseEnhancedTool) SetSchema(schema *types.ToolSchema) { t.schema = schema }
func (t *BaseEnhancedTool) SetLongRunning(v bool)              { t.isLongRunning = v }
func (t *BaseEnhancedTool) SetTimeout(d time.Duration)         { t.timeout = d }
func (t *BaseEnhancedTool) SetRequireApproval(v bool)          { t.requireApproval = v }
func (t *BaseEnhancedTool) SetPriority(p int)                  { t.priority = p }
func (t *BaseEnhancedTool) SetRetryPolicy(policy *RetryPolicy) { t.retryPolicy = policy }
func (t *BaseEnhancedTool) SetMetadata(key string, value any) {
	t.metadata[key] = value
}

// Execute 需要由具体工具实现
func (t *BaseEnhancedTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	panic("Execute must be implemented by concrete tool")
}

// DefaultRetryPolicy 默认重试策略
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:        3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        5 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableErrors:   []string{"timeout", "network", "temporary"},
	}
}

// NoRetryPolicy 不重试策略
func NoRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries: 0,
	}
}
