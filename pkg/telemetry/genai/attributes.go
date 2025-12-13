// Package genai provides OpenTelemetry GenAI Semantic Conventions for Aster.
// Based on: https://opentelemetry.io/docs/specs/semconv/gen-ai/
package genai

// Operation names for GenAI spans
const (
	// OpInvokeAgent represents an agent invocation operation
	OpInvokeAgent = "invoke_agent"

	// OpCreateAgent represents an agent creation operation
	OpCreateAgent = "create_agent"

	// OpChat represents a chat/completion operation
	OpChat = "chat"

	// OpGenerateContent represents content generation (Google style)
	OpGenerateContent = "generate_content"

	// OpExecuteTool represents a tool execution operation
	OpExecuteTool = "execute_tool"

	// OpEmbeddings represents an embeddings operation
	OpEmbeddings = "embeddings"
)

// Attribute keys following OpenTelemetry GenAI Semantic Conventions
const (
	// General operation attributes
	AttrOperationName = "gen_ai.operation.name"

	// Provider attributes
	AttrProviderName = "gen_ai.provider.name"

	// Agent attributes
	AttrAgentID          = "gen_ai.agent.id"
	AttrAgentName        = "gen_ai.agent.name"
	AttrAgentDescription = "gen_ai.agent.description"

	// Conversation/Session attributes
	AttrConversationID = "gen_ai.conversation.id"
	AttrThreadID       = "gen_ai.thread.id" // alias for conversation_id

	// Request attributes
	AttrRequestModel            = "gen_ai.request.model"
	AttrRequestMaxTokens        = "gen_ai.request.max_tokens"
	AttrRequestTemperature      = "gen_ai.request.temperature"
	AttrRequestTopP             = "gen_ai.request.top_p"
	AttrRequestTopK             = "gen_ai.request.top_k"
	AttrRequestStopSequences    = "gen_ai.request.stop_sequences"
	AttrRequestPresencePenalty  = "gen_ai.request.presence_penalty"
	AttrRequestFrequencyPenalty = "gen_ai.request.frequency_penalty"

	// Response attributes
	AttrResponseID           = "gen_ai.response.id"
	AttrResponseModel        = "gen_ai.response.model"
	AttrResponseFinishReason = "gen_ai.response.finish_reasons"

	// Token usage attributes
	AttrUsageInputTokens  = "gen_ai.usage.input_tokens"
	AttrUsageOutputTokens = "gen_ai.usage.output_tokens"

	// Tool attributes
	AttrToolName        = "gen_ai.tool.name"
	AttrToolCallID      = "gen_ai.tool.call.id"
	AttrToolDefinitions = "gen_ai.tool.definitions"

	// Error attributes
	AttrErrorType = "error.type"

	// Performance attributes (Aster-specific extensions)
	AttrLatencyTTFT     = "gen_ai.latency.ttft_ms"       // Time to First Token
	AttrLatencyTPOT     = "gen_ai.latency.tpot_ms"       // Time Per Output Token
	AttrLatencyTotal    = "gen_ai.latency.total_ms"      // Total latency
	AttrIterationCount  = "gen_ai.agent.iteration_count" // Agent loop iterations

	// Cost attributes (Aster-specific extensions)
	AttrCostInput    = "gen_ai.cost.input"
	AttrCostOutput   = "gen_ai.cost.output"
	AttrCostTotal    = "gen_ai.cost.total"
	AttrCostCurrency = "gen_ai.cost.currency"
)

// Provider names
const (
	ProviderAnthropic = "anthropic"
	ProviderOpenAI    = "openai"
	ProviderDeepSeek  = "deepseek"
	ProviderGoogle    = "gcp.vertex_ai"
	ProviderAzure     = "azure.openai"
	ProviderBedrock   = "aws.bedrock"
)

// Error types
const (
	ErrorTypeTimeout            = "timeout"
	ErrorTypeRateLimit          = "rate_limit"
	ErrorTypeInvalidRequest     = "invalid_request"
	ErrorTypeAuthentication     = "authentication"
	ErrorTypePermission         = "permission"
	ErrorTypeNotFound           = "not_found"
	ErrorTypeServerError        = "server_error"
	ErrorTypeContentFilter      = "content_filter"
	ErrorTypeContextLengthExceeded = "context_length_exceeded"
)

// Finish reasons
const (
	FinishReasonStop         = "stop"
	FinishReasonLength       = "length"
	FinishReasonToolUse      = "tool_use"
	FinishReasonContentFilter = "content_filter"
	FinishReasonError        = "error"
)

// SpanKind constants for GenAI operations
const (
	// SpanKindClient should be used for remote LLM calls
	SpanKindClient = "client"

	// SpanKindInternal should be used for in-process operations
	SpanKindInternal = "internal"
)

// AttributeBuilder helps construct GenAI span attributes
type AttributeBuilder struct {
	attrs map[string]any
}

// NewAttributeBuilder creates a new attribute builder
func NewAttributeBuilder() *AttributeBuilder {
	return &AttributeBuilder{
		attrs: make(map[string]any),
	}
}

// WithOperation sets the operation name
func (b *AttributeBuilder) WithOperation(op string) *AttributeBuilder {
	b.attrs[AttrOperationName] = op
	return b
}

// WithProvider sets the provider name
func (b *AttributeBuilder) WithProvider(provider string) *AttributeBuilder {
	b.attrs[AttrProviderName] = provider
	return b
}

// WithAgent sets agent-related attributes
func (b *AttributeBuilder) WithAgent(id, name string) *AttributeBuilder {
	b.attrs[AttrAgentID] = id
	b.attrs[AttrAgentName] = name
	return b
}

// WithConversation sets the conversation/session ID
func (b *AttributeBuilder) WithConversation(id string) *AttributeBuilder {
	b.attrs[AttrConversationID] = id
	return b
}

// WithModel sets the request model
func (b *AttributeBuilder) WithModel(model string) *AttributeBuilder {
	b.attrs[AttrRequestModel] = model
	return b
}

// WithModelParams sets model parameters
func (b *AttributeBuilder) WithModelParams(maxTokens int, temperature, topP float64) *AttributeBuilder {
	if maxTokens > 0 {
		b.attrs[AttrRequestMaxTokens] = maxTokens
	}
	if temperature >= 0 {
		b.attrs[AttrRequestTemperature] = temperature
	}
	if topP > 0 {
		b.attrs[AttrRequestTopP] = topP
	}
	return b
}

// WithTokenUsage sets token usage attributes
func (b *AttributeBuilder) WithTokenUsage(input, output int64) *AttributeBuilder {
	b.attrs[AttrUsageInputTokens] = input
	b.attrs[AttrUsageOutputTokens] = output
	return b
}

// WithTool sets tool-related attributes
func (b *AttributeBuilder) WithTool(name, callID string) *AttributeBuilder {
	b.attrs[AttrToolName] = name
	b.attrs[AttrToolCallID] = callID
	return b
}

// WithError sets error attributes
func (b *AttributeBuilder) WithError(errType string) *AttributeBuilder {
	b.attrs[AttrErrorType] = errType
	return b
}

// WithLatency sets latency attributes
func (b *AttributeBuilder) WithLatency(ttft, tpot, total int64) *AttributeBuilder {
	if ttft > 0 {
		b.attrs[AttrLatencyTTFT] = ttft
	}
	if tpot > 0 {
		b.attrs[AttrLatencyTPOT] = tpot
	}
	if total > 0 {
		b.attrs[AttrLatencyTotal] = total
	}
	return b
}

// WithCost sets cost attributes
func (b *AttributeBuilder) WithCost(input, output, total float64, currency string) *AttributeBuilder {
	b.attrs[AttrCostInput] = input
	b.attrs[AttrCostOutput] = output
	b.attrs[AttrCostTotal] = total
	b.attrs[AttrCostCurrency] = currency
	return b
}

// Build returns the built attributes map
func (b *AttributeBuilder) Build() map[string]any {
	result := make(map[string]any, len(b.attrs))
	for k, v := range b.attrs {
		result[k] = v
	}
	return result
}

// Set adds a custom attribute
func (b *AttributeBuilder) Set(key string, value any) *AttributeBuilder {
	b.attrs[key] = value
	return b
}

// EventNames for GenAI semantic events
const (
	// EventPrompt represents prompt content event
	EventPrompt = "gen_ai.content.prompt"

	// EventCompletion represents completion content event
	EventCompletion = "gen_ai.content.completion"

	// EventToolCall represents a tool call event
	EventToolCall = "gen_ai.tool.call"

	// EventToolResult represents a tool result event
	EventToolResult = "gen_ai.tool.result"
)

// SpanName generates a standardized span name for GenAI operations
func SpanName(operation string, subject string) string {
	if subject != "" {
		return operation + " " + subject
	}
	return operation
}

// AgentSpanName generates a span name for agent operations
func AgentSpanName(agentName string) string {
	return SpanName(OpInvokeAgent, agentName)
}

// ChatSpanName generates a span name for chat operations
func ChatSpanName(model string) string {
	return SpanName(OpChat, model)
}

// ToolSpanName generates a span name for tool operations
func ToolSpanName(toolName string) string {
	return SpanName(OpExecuteTool, toolName)
}
