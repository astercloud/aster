package middleware

import (
	"context"
	"testing"

	"github.com/astercloud/aster/pkg/types"
)

// BenchmarkSummarizationMiddleware_NoSummarization 基准测试: 未触发总结
func BenchmarkSummarizationMiddleware_NoSummarization(b *testing.B) {
	ctx := context.Background()

	middleware, _ := NewSummarizationMiddleware(&SummarizationMiddlewareConfig{
		Summarizer:             mockSummarizer("Summary", false),
		MaxTokensBeforeSummary: 100000, // 高阈值,不会触发
		MessagesToKeep:         3,
	})

	req := &ModelRequest{
		Messages: []types.Message{
			{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Hello"}}},
			{Role: types.MessageRoleAssistant, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Hi!"}}},
		},
	}

	handler := func(ctx context.Context, req *ModelRequest) (*ModelResponse, error) {
		return &ModelResponse{
			Message: types.Message{
				Role:    types.MessageRoleAssistant,
				ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Response"}},
			},
		}, nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		middleware.WrapModelCall(ctx, req, handler)
	}
}

// BenchmarkSummarizationMiddleware_WithSummarization 基准测试: 触发总结
func BenchmarkSummarizationMiddleware_WithSummarization(b *testing.B) {
	ctx := context.Background()

	middleware, _ := NewSummarizationMiddleware(&SummarizationMiddlewareConfig{
		Summarizer:             mockSummarizer("Summary", false),
		MaxTokensBeforeSummary: 10, // 低阈值,总是触发
		MessagesToKeep:         2,
	})

	req := &ModelRequest{
		Messages: []types.Message{
			{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Message 1"}}},
			{Role: types.MessageRoleAssistant, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Response 1"}}},
			{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Message 2"}}},
			{Role: types.MessageRoleAssistant, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Response 2"}}},
			{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Message 3"}}},
		},
	}

	handler := func(ctx context.Context, req *ModelRequest) (*ModelResponse, error) {
		return &ModelResponse{
			Message: types.Message{
				Role:    types.MessageRoleAssistant,
				ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Response"}},
			},
		}, nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		middleware.WrapModelCall(ctx, req, handler)
	}
}

// BenchmarkAgentMemoryMiddleware_LazyLoad 基准测试: 懒加载
func BenchmarkAgentMemoryMiddleware_LazyLoad(b *testing.B) {
	ctx := context.Background()

	backend := newMockBackend()
	backend.files["/agent.md"] = "Test memory content for benchmarking"

	middleware, _ := NewAgentMemoryMiddleware(&AgentMemoryMiddlewareConfig{
		Backend:    backend,
		MemoryPath: "/memories/",
	})

	req := &ModelRequest{
		SystemPrompt: "Original prompt",
		Messages:     []types.Message{},
	}

	handler := func(ctx context.Context, req *ModelRequest) (*ModelResponse, error) {
		return &ModelResponse{
			Message: types.Message{
				Role:    types.MessageRoleAssistant,
				ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Response"}},
			},
		}, nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 第一次会触发加载,后续直接使用缓存
		middleware.WrapModelCall(ctx, req, handler)
	}
}

// BenchmarkAgentMemoryMiddleware_AlreadyLoaded 基准测试: 已加载
func BenchmarkAgentMemoryMiddleware_AlreadyLoaded(b *testing.B) {
	ctx := context.Background()

	backend := newMockBackend()
	backend.files["/agent.md"] = "Test memory content for benchmarking"

	middleware, _ := NewAgentMemoryMiddleware(&AgentMemoryMiddlewareConfig{
		Backend:    backend,
		MemoryPath: "/memories/",
	})

	// 预先加载
	middleware.OnAgentStart(ctx, "test")

	req := &ModelRequest{
		SystemPrompt: "Original prompt",
		Messages:     []types.Message{},
	}

	handler := func(ctx context.Context, req *ModelRequest) (*ModelResponse, error) {
		return &ModelResponse{
			Message: types.Message{
				Role:    types.MessageRoleAssistant,
				ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Response"}},
			},
		}, nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		middleware.WrapModelCall(ctx, req, handler)
	}
}

// BenchmarkDefaultTokenCounter 基准测试: 默认 token 计数器
func BenchmarkDefaultTokenCounter(b *testing.B) {
	messages := []types.Message{
		{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Hello, how are you?"}}},
		{Role: types.MessageRoleAssistant, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "I'm doing well, thank you!"}}},
		{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Can you help me with Go programming?"}}},
		{Role: types.MessageRoleAssistant, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Of course! What would you like to know?"}}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = defaultTokenCounter(messages)
	}
}

// BenchmarkDefaultSummarizer 基准测试: 默认总结器
func BenchmarkDefaultSummarizer(b *testing.B) {
	ctx := context.Background()

	messages := []types.Message{
		{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Hello"}}},
		{Role: types.MessageRoleAssistant, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Hi there!"}}},
		{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "How are you?"}}},
		{Role: types.MessageRoleAssistant, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "I'm good, thanks!"}}},
		{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Can you help me?"}}},
		{Role: types.MessageRoleAssistant, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Sure, what do you need?"}}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = defaultSummarizer(ctx, messages)
	}
}

// BenchmarkPhase4Stack 基准测试: Phase 4 完整中间件栈
func BenchmarkPhase4Stack(b *testing.B) {
	ctx := context.Background()

	// 创建后端
	backend := newMockBackend()
	backend.files["/agent.md"] = "You are a helpful assistant"

	// 创建中间件
	memoryMW, _ := NewAgentMemoryMiddleware(&AgentMemoryMiddlewareConfig{
		Backend:    backend,
		MemoryPath: "/memories/",
	})

	summarizationMW, _ := NewSummarizationMiddleware(&SummarizationMiddlewareConfig{
		Summarizer:             mockSummarizer("Summary", false),
		MaxTokensBeforeSummary: 100000, // 不触发总结
		MessagesToKeep:         3,
	})

	req := &ModelRequest{
		SystemPrompt: "Test prompt",
		Messages: []types.Message{
			{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Hello"}}},
		},
	}

	// 构建中间件链
	handler := func(ctx context.Context, req *ModelRequest) (*ModelResponse, error) {
		return &ModelResponse{
			Message: types.Message{
				Role:    types.MessageRoleAssistant,
				ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Response"}},
			},
		}, nil
	}

	// Memory -> Summarization -> Handler
	summarizationHandler := func(ctx context.Context, req *ModelRequest) (*ModelResponse, error) {
		return summarizationMW.WrapModelCall(ctx, req, handler)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		memoryMW.WrapModelCall(ctx, req, summarizationHandler)
	}
}

// BenchmarkExtractMessageContent 基准测试: 消息内容提取
func BenchmarkExtractMessageContent(b *testing.B) {
	msg := types.Message{
		Role: types.MessageRoleAssistant,
		ContentBlocks: []types.ContentBlock{
			&types.TextBlock{Text: "Hello, this is a test message"},
			&types.ToolUseBlock{Name: "test_tool", Input: map[string]interface{}{"key": "value"}},
			&types.TextBlock{Text: "Another text block"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractMessageContent(msg)
	}
}
