package types

import "strings"

// StreamChunkType 流式响应块类型
type StreamChunkType string

const (
	// ChunkTypeText 文本块
	ChunkTypeText StreamChunkType = "text"

	// ChunkTypeReasoning 推理过程块 (用于 o1/o3/R1 等推理模型)
	ChunkTypeReasoning StreamChunkType = "reasoning"

	// ChunkTypeUsage Token 使用情况块
	ChunkTypeUsage StreamChunkType = "usage"

	// ChunkTypeToolCall 工具调用块
	ChunkTypeToolCall StreamChunkType = "tool_call"

	// ChunkTypeError 错误块
	ChunkTypeError StreamChunkType = "error"

	// ChunkTypeDone 完成块
	ChunkTypeDone StreamChunkType = "done"
)

// StreamChunk 流式响应块 (扩展版本)
type StreamChunk struct {
	// Type 块类型
	Type StreamChunkType `json:"type"`

	// Delta 增量文本内容 (用于 text 类型)
	Delta string `json:"delta,omitempty"`

	// ToolCall 工具调用信息 (用于 tool_call 类型)
	ToolCall *ToolCallDelta `json:"tool_call,omitempty"`

	// Usage Token 使用情况 (用于 usage 类型)
	Usage *TokenUsage `json:"usage,omitempty"`

	// Reasoning 推理过程 (用于 reasoning 类型)
	Reasoning *ReasoningTrace `json:"reasoning,omitempty"`

	// Error 错误信息 (用于 error 类型)
	Error *StreamError `json:"error,omitempty"`

	// FinishReason 完成原因 (用于 done 类型)
	FinishReason string `json:"finish_reason,omitempty"`

	// Raw 原始响应数据 (用于调试)
	Raw map[string]any `json:"raw,omitempty"`
}

// ToolCallDelta 工具调用增量
type ToolCallDelta struct {
	// Index 工具调用索引
	Index int `json:"index"`

	// ID 工具调用 ID (第一个块)
	ID string `json:"id,omitempty"`

	// Type 工具类型 (通常是 "function")
	Type string `json:"type,omitempty"`

	// Name 工具名称 (第一个块)
	Name string `json:"name,omitempty"`

	// ArgumentsDelta 参数增量 (JSON 片段)
	ArgumentsDelta string `json:"arguments_delta,omitempty"`
}

// ReasoningTrace 推理过程跟踪
// 用于 OpenAI o1/o3, DeepSeek R1 等推理模型
type ReasoningTrace struct {
	// Step 推理步骤序号
	Step int `json:"step"`

	// Thought 推理思考内容
	Thought string `json:"thought"`

	// ThoughtDelta 推理思考增量 (流式)
	ThoughtDelta string `json:"thought_delta,omitempty"`

	// Type 推理类型
	// - "thinking": 思考过程
	// - "reflection": 反思
	// - "conclusion": 结论
	Type string `json:"type,omitempty"`

	// Confidence 置信度 (0-1)
	Confidence float64 `json:"confidence,omitempty"`
}

// StreamError 流式错误
type StreamError struct {
	// Code 错误代码
	Code string `json:"code"`

	// Message 错误消息
	Message string `json:"message"`

	// Type 错误类型
	Type string `json:"type,omitempty"`

	// Param 错误参数
	Param string `json:"param,omitempty"`
}

// TokenUsage Token 使用情况 (扩展版本)
type TokenUsage struct {
	// InputTokens 输入 Token 数
	InputTokens int `json:"input_tokens"`

	// OutputTokens 输出 Token 数
	OutputTokens int `json:"output_tokens"`

	// TotalTokens 总 Token 数
	TotalTokens int `json:"total_tokens"`

	// ReasoningTokens 推理 Token 数 (用于推理模型)
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`

	// CachedTokens 缓存命中的 Token 数 (Prompt Caching)
	CachedTokens int `json:"cached_tokens,omitempty"`

	// CacheCreationTokens 缓存创建的 Token 数
	CacheCreationTokens int `json:"cache_creation_tokens,omitempty"`

	// CacheReadTokens 缓存读取的 Token 数
	CacheReadTokens int `json:"cache_read_tokens,omitempty"`
}

// StreamAccumulator 流式响应累加器
// 用于累积流式响应，构建完整消息
type StreamAccumulator struct {
	// Content 累积的文本内容
	Content string

	// Reasoning 累积的推理过程
	Reasoning []ReasoningTrace

	// ToolCalls 累积的工具调用
	ToolCalls map[int]*AccumulatedToolCall

	// Usage 最终的 Token 使用情况
	Usage *TokenUsage

	// FinishReason 完成原因
	FinishReason string
}

// AccumulatedToolCall 累积的工具调用
type AccumulatedToolCall struct {
	ID        string
	Type      string
	Name      string
	Arguments string // 累积的 JSON 参数
}

// NewStreamAccumulator 创建新的流式累加器
func NewStreamAccumulator() *StreamAccumulator {
	return &StreamAccumulator{
		Reasoning: make([]ReasoningTrace, 0),
		ToolCalls: make(map[int]*AccumulatedToolCall),
	}
}

// AddChunk 添加流式块
func (acc *StreamAccumulator) AddChunk(chunk *StreamChunk) {
	switch chunk.Type {
	case ChunkTypeText:
		acc.Content += chunk.Delta

	case ChunkTypeReasoning:
		if chunk.Reasoning != nil {
			acc.Reasoning = append(acc.Reasoning, *chunk.Reasoning)
		}

	case ChunkTypeToolCall:
		if chunk.ToolCall != nil {
			tc := chunk.ToolCall
			if existing, ok := acc.ToolCalls[tc.Index]; ok {
				// 累加参数
				existing.Arguments += tc.ArgumentsDelta
			} else {
				// 新工具调用
				acc.ToolCalls[tc.Index] = &AccumulatedToolCall{
					ID:        tc.ID,
					Type:      tc.Type,
					Name:      tc.Name,
					Arguments: tc.ArgumentsDelta,
				}
			}
		}

	case ChunkTypeUsage:
		acc.Usage = chunk.Usage

	case ChunkTypeDone:
		acc.FinishReason = chunk.FinishReason
	}
}

// ToMessage 将累积的结果转换为 Message
func (acc *StreamAccumulator) ToMessage() Message {
	msg := Message{
		Role:    RoleAssistant,
		Content: acc.Content,
	}

	// 如果有工具调用，构建 ContentBlocks
	if len(acc.ToolCalls) > 0 {
		blocks := make([]ContentBlock, 0)

		// 添加文本块
		if acc.Content != "" {
			blocks = append(blocks, &TextBlock{Text: acc.Content})
		}

		// 添加工具调用块
		for _, tc := range acc.ToolCalls {
			// 简化处理：直接存储字符串，由上层解析
			blocks = append(blocks, &ToolUseBlock{
				ID:    tc.ID,
				Name:  tc.Name,
				Input: map[string]any{"raw": tc.Arguments},
			})
		}

		msg.ContentBlocks = blocks
		msg.Content = "" // 清空简单内容
	}

	return msg
}

// GetReasoningText 获取所有推理文本
func (acc *StreamAccumulator) GetReasoningText() string {
	var text string
	var textSb239 strings.Builder
	for _, r := range acc.Reasoning {
		textSb239.WriteString(r.Thought + "\n")
	}
	text += textSb239.String()
	return text
}

// IsComplete 检查是否完成
func (acc *StreamAccumulator) IsComplete() bool {
	return acc.FinishReason != ""
}
