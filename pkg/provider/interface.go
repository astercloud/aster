package provider

import (
	"context"

	"github.com/astercloud/aster/pkg/types"
)

// StreamChunkType 流式响应块类型
type StreamChunkType string

const (
	// 原有类型（兼容 Anthropic）
	ChunkTypeContentBlockStart StreamChunkType = "content_block_start"
	ChunkTypeContentBlockDelta StreamChunkType = "content_block_delta"
	ChunkTypeContentBlockStop  StreamChunkType = "content_block_stop"
	ChunkTypeMessageDelta      StreamChunkType = "message_delta"

	// 新增类型（通用）
	ChunkTypeText      StreamChunkType = "text"
	ChunkTypeReasoning StreamChunkType = "reasoning"
	ChunkTypeUsage     StreamChunkType = "usage"
	ChunkTypeToolCall  StreamChunkType = "tool_call"
	ChunkTypeError     StreamChunkType = "error"
	ChunkTypeDone      StreamChunkType = "done"
)

// StreamChunk 流式响应块（扩展版本）
type StreamChunk struct {
	// Type 块类型
	Type string `json:"type"`

	// Index 内容块索引（用于兼容 Anthropic 格式）
	Index int `json:"index,omitempty"`

	// Delta 增量数据（通用，兼容旧版）
	Delta any `json:"delta,omitempty"`

	// TextDelta 文本增量（新增，明确类型）
	TextDelta string `json:"text_delta,omitempty"`

	// ToolCall 工具调用增量（新增）
	ToolCall *ToolCallDelta `json:"tool_call,omitempty"`

	// Reasoning 推理过程（新增）
	Reasoning *ReasoningTrace `json:"reasoning,omitempty"`

	// Usage Token使用情况
	Usage *TokenUsage `json:"usage,omitempty"`

	// Error 错误信息（新增）
	Error *StreamError `json:"error,omitempty"`

	// FinishReason 完成原因（新增）
	FinishReason string `json:"finish_reason,omitempty"`
}

// ToolCallDelta 工具调用增量
type ToolCallDelta struct {
	Index          int    `json:"index"`
	ID             string `json:"id,omitempty"`
	Type           string `json:"type,omitempty"`
	Name           string `json:"name,omitempty"`
	ArgumentsDelta string `json:"arguments_delta,omitempty"`
}

// ReasoningTrace 推理过程跟踪
type ReasoningTrace struct {
	Step         int     `json:"step"`
	Thought      string  `json:"thought"`
	ThoughtDelta string  `json:"thought_delta,omitempty"`
	Type         string  `json:"type,omitempty"` // "thinking", "reflection", "conclusion"
	Confidence   float64 `json:"confidence,omitempty"`
}

// StreamError 流式错误
type StreamError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
	Param   string `json:"param,omitempty"`
}

// TokenUsage Token使用统计（扩展版本）
type TokenUsage struct {
	// 基础统计
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	TotalTokens  int64 `json:"total_tokens,omitempty"`

	// 推理模型特殊统计
	ReasoningTokens int64 `json:"reasoning_tokens,omitempty"`

	// Prompt Caching 统计
	CachedTokens        int64 `json:"cached_tokens,omitempty"`
	CacheCreationTokens int64 `json:"cache_creation_tokens,omitempty"`
	CacheReadTokens     int64 `json:"cache_read_tokens,omitempty"`
}

// StreamOptions 流式请求选项
type StreamOptions struct {
	Tools       []ToolSchema
	MaxTokens   int
	Temperature float64
	System      string

	// ToolChoice 工具选择策略（Anthropic API 支持）
	// 可选值: nil (默认), "auto", "any", 或指定工具名
	ToolChoice *ToolChoiceOption `json:"tool_choice,omitempty"`
}

// ToolChoiceOption 工具选择选项
type ToolChoiceOption struct {
	// Type 选择类型: "auto", "any", "tool"
	Type string `json:"type"`

	// Name 当 Type="tool" 时，指定工具名称
	Name string `json:"name,omitempty"`

	// DisableParallelToolUse 禁用并行工具调用
	DisableParallelToolUse bool `json:"disable_parallel_tool_use,omitempty"`
}

// CompleteResponse 完整响应
type CompleteResponse struct {
	Message types.Message
	Usage   *TokenUsage
}

// ToolExample 工具使用示例（与 tools.ToolExample 保持一致）
type ToolExample struct {
	Description string         `json:"description"`
	Input       map[string]any `json:"input"`
	Output      any            `json:"output,omitempty"`
}

// ToolSchema 工具Schema
type ToolSchema struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`

	// InputExamples 工具使用示例，帮助 LLM 更准确地调用工具
	// 参考 Anthropic 的 Tool Use Examples 功能
	InputExamples []ToolExample `json:"input_examples,omitempty"`

	// AllowedCallers 指定哪些上下文可以调用此工具 (PTC 支持)
	// 可选值: ["direct"], ["code_execution_20250825"], 或两者组合
	// 默认: nil 或 ["direct"] - 仅 LLM 直接调用
	AllowedCallers []string `json:"allowed_callers,omitempty"`
}

// ProviderCapabilities 模型能力（扩展版本）
type ProviderCapabilities struct {
	// 基础能力
	SupportToolCalling  bool // 是否支持工具调用
	SupportSystemPrompt bool // 是否支持独立 system prompt
	SupportStreaming    bool // 是否支持流式输出

	// 多模态能力
	SupportVision bool // 是否支持视觉（图片）
	SupportAudio  bool // 是否支持音频
	SupportVideo  bool // 是否支持视频

	// 高级能力
	SupportReasoning    bool // 是否支持推理模型（o1/o3/R1）
	SupportPromptCache  bool // 是否支持 Prompt Caching
	SupportJSONMode     bool // 是否支持 JSON 模式
	SupportFunctionCall bool // 是否支持 Function Calling

	// 限制
	MaxTokens       int // 最大 token 数
	MaxToolsPerCall int // 单次最多调用工具数
	MaxImageSize    int // 最大图片大小（字节）

	// Tool Calling 格式
	ToolCallingFormat string // "anthropic" | "openai" | "qwen" | "custom"

	// 推理模型特性
	ReasoningTokensIncluded bool // reasoning tokens 是否包含在总 token 中

	// Prompt Caching 特性
	CacheMinTokens int // 最小缓存 Token 数
}

// Provider 模型提供商接口
type Provider interface {
	// Stream 流式对话
	Stream(ctx context.Context, messages []types.Message, opts *StreamOptions) (<-chan StreamChunk, error)

	// Complete 非流式对话(阻塞式,返回完整响应)
	Complete(ctx context.Context, messages []types.Message, opts *StreamOptions) (*CompleteResponse, error)

	// Config 返回配置
	Config() *types.ModelConfig

	// Capabilities 返回模型能力
	Capabilities() ProviderCapabilities

	// SetSystemPrompt 设置系统提示词
	SetSystemPrompt(prompt string) error

	// GetSystemPrompt 获取系统提示词
	GetSystemPrompt() string

	// Close 关闭连接
	Close() error
}

// Factory 模型提供商工厂
type Factory interface {
	Create(config *types.ModelConfig) (Provider, error)
}
