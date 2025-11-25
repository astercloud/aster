package middleware

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/astercloud/aster/pkg/types"
)

// mockSummarizer 模拟总结器
func mockSummarizer(response string, shouldFail bool) SummarizerFunc {
	return func(ctx context.Context, messages []types.Message) (string, error) {
		if shouldFail {
			return "", errors.New("mock summarizer failure")
		}
		return response, nil
	}
}

// TestSummarizationMiddleware_NoSummarization 测试未超过阈值时不触发总结
func TestSummarizationMiddleware_NoSummarization(t *testing.T) {
	ctx := context.Background()

	middleware, err := NewSummarizationMiddleware(&SummarizationMiddlewareConfig{
		Summarizer:             mockSummarizer("Summary", false),
		MaxTokensBeforeSummary: 1000,
		MessagesToKeep:         3,
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// 创建少量消息(不会触发总结)
	req := &ModelRequest{
		Messages: []types.Message{
			{Role: types.MessageRoleSystem, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "You are a helpful assistant"}}},
			{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Hello"}}},
			{Role: types.MessageRoleAssistant, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Hi there!"}}},
		},
	}

	originalLen := len(req.Messages)

	handler := func(ctx context.Context, req *ModelRequest) (*ModelResponse, error) {
		return &ModelResponse{
			Message: types.Message{
				Role:          types.MessageRoleAssistant,
				ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Response"}},
			},
		}, nil
	}

	_, err = middleware.WrapModelCall(ctx, req, handler)
	if err != nil {
		t.Fatalf("WrapModelCall failed: %v", err)
	}

	// 消息数量应该不变
	if len(req.Messages) != originalLen {
		t.Errorf("Expected %d messages, got %d", originalLen, len(req.Messages))
	}

	// 总结计数应该为 0
	if middleware.GetSummarizationCount() != 0 {
		t.Errorf("Expected 0 summarizations, got %d", middleware.GetSummarizationCount())
	}
}

// TestSummarizationMiddleware_TriggerSummarization 测试超过阈值触发总结
func TestSummarizationMiddleware_TriggerSummarization(t *testing.T) {
	ctx := context.Background()

	middleware, err := NewSummarizationMiddleware(&SummarizationMiddlewareConfig{
		Summarizer:             mockSummarizer("This is a summary of the conversation", false),
		MaxTokensBeforeSummary: 10, // 非常低的阈值,确保触发
		MessagesToKeep:         2,
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// 创建多条消息
	req := &ModelRequest{
		Messages: []types.Message{
			{Role: types.MessageRoleSystem, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "You are a helpful assistant"}}},
			{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Message 1"}}},
			{Role: types.MessageRoleAssistant, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Response 1"}}},
			{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Message 2"}}},
			{Role: types.MessageRoleAssistant, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Response 2"}}},
			{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Message 3"}}},
			{Role: types.MessageRoleAssistant, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Response 3"}}},
		},
	}

	originalLen := len(req.Messages)

	handler := func(ctx context.Context, req *ModelRequest) (*ModelResponse, error) {
		return &ModelResponse{
			Message: types.Message{
				Role:          types.MessageRoleAssistant,
				ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Response"}},
			},
		}, nil
	}

	_, err = middleware.WrapModelCall(ctx, req, handler)
	if err != nil {
		t.Fatalf("WrapModelCall failed: %v", err)
	}

	// 消息数量应该减少
	if len(req.Messages) >= originalLen {
		t.Errorf("Expected fewer messages after summarization, got %d (was %d)",
			len(req.Messages), originalLen)
	}

	// 应该包含总结消息
	hasSummary := false
	for _, msg := range req.Messages {
		for _, block := range msg.ContentBlocks {
			if tb, ok := block.(*types.TextBlock); ok {
				if strings.Contains(tb.Text, "This is a summary") {
					hasSummary = true
					break
				}
			}
		}
		if hasSummary {
			break
		}
	}
	if !hasSummary {
		t.Error("Expected summary message in modified messages")
	}

	// 总结计数应该为 1
	if middleware.GetSummarizationCount() != 1 {
		t.Errorf("Expected 1 summarization, got %d", middleware.GetSummarizationCount())
	}

	// 验证 system message 被保留
	hasSystemMessage := false
	for _, msg := range req.Messages {
		if msg.Role == types.MessageRoleSystem {
			for _, block := range msg.ContentBlocks {
				if tb, ok := block.(*types.TextBlock); ok {
					if strings.Contains(tb.Text, "helpful assistant") {
						hasSystemMessage = true
						break
					}
				}
			}
		}
		if hasSystemMessage {
			break
		}
	}
	if !hasSystemMessage {
		t.Error("Original system message should be preserved")
	}
}

// TestSummarizationMiddleware_PreserveRecentMessages 测试保留最近的消息
func TestSummarizationMiddleware_PreserveRecentMessages(t *testing.T) {
	ctx := context.Background()

	middleware, err := NewSummarizationMiddleware(&SummarizationMiddlewareConfig{
		Summarizer:             mockSummarizer("Summary", false),
		MaxTokensBeforeSummary: 10,
		MessagesToKeep:         2, // 保留最近 2 条
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	req := &ModelRequest{
		Messages: []types.Message{
			{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Old message 1"}}},
			{Role: types.MessageRoleAssistant, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Old response 1"}}},
			{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Old message 2"}}},
			{Role: types.MessageRoleAssistant, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Old response 2"}}},
			{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Recent message 1"}}},
			{Role: types.MessageRoleAssistant, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Recent response 1"}}},
		},
	}

	handler := func(ctx context.Context, req *ModelRequest) (*ModelResponse, error) {
		return &ModelResponse{
			Message: types.Message{
				Role:          types.MessageRoleAssistant,
				ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Response"}},
			},
		}, nil
	}

	_, err = middleware.WrapModelCall(ctx, req, handler)
	if err != nil {
		t.Fatalf("WrapModelCall failed: %v", err)
	}

	// 验证最近的消息被保留
	hasRecentMessage := false
	for _, msg := range req.Messages {
		for _, block := range msg.ContentBlocks {
			if tb, ok := block.(*types.TextBlock); ok {
				if strings.Contains(tb.Text, "Recent message 1") ||
					strings.Contains(tb.Text, "Recent response 1") {
					hasRecentMessage = true
					break
				}
			}
		}
		if hasRecentMessage {
			break
		}
	}
	if !hasRecentMessage {
		t.Error("Recent messages should be preserved")
	}
}

// TestSummarizationMiddleware_SummarizerFailure 测试 Summarizer 失败时的处理
func TestSummarizationMiddleware_SummarizerFailure(t *testing.T) {
	ctx := context.Background()

	middleware, err := NewSummarizationMiddleware(&SummarizationMiddlewareConfig{
		Summarizer:             mockSummarizer("", true), // 失败的 summarizer
		MaxTokensBeforeSummary: 10,
		MessagesToKeep:         2,
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	req := &ModelRequest{
		Messages: []types.Message{
			{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Message 1"}}},
			{Role: types.MessageRoleAssistant, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Response 1"}}},
			{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Message 2"}}},
			{Role: types.MessageRoleAssistant, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Response 2"}}},
			{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Message 3"}}},
		},
	}

	originalLen := len(req.Messages)

	handler := func(ctx context.Context, req *ModelRequest) (*ModelResponse, error) {
		return &ModelResponse{
			Message: types.Message{
				Role:          types.MessageRoleAssistant,
				ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Response"}},
			},
		}, nil
	}

	_, err = middleware.WrapModelCall(ctx, req, handler)
	if err != nil {
		t.Fatalf("WrapModelCall should not return error on summarizer failure: %v", err)
	}

	// Summarizer 失败时应该保留原始消息
	if len(req.Messages) != originalLen {
		t.Errorf("Expected original messages to be preserved on summarizer failure, got %d (was %d)",
			len(req.Messages), originalLen)
	}

	// 总结计数应该仍为 0
	if middleware.GetSummarizationCount() != 0 {
		t.Errorf("Expected 0 summarizations after failure, got %d", middleware.GetSummarizationCount())
	}
}

// TestSummarizationMiddleware_EmptyMessages 测试空消息列表
func TestSummarizationMiddleware_EmptyMessages(t *testing.T) {
	ctx := context.Background()

	middleware, err := NewSummarizationMiddleware(&SummarizationMiddlewareConfig{
		Summarizer:             mockSummarizer("Summary", false),
		MaxTokensBeforeSummary: 100,
		MessagesToKeep:         3,
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	req := &ModelRequest{
		Messages: []types.Message{},
	}

	handler := func(ctx context.Context, req *ModelRequest) (*ModelResponse, error) {
		return &ModelResponse{
			Message: types.Message{
				Role:          types.MessageRoleAssistant,
				ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Response"}},
			},
		}, nil
	}

	_, err = middleware.WrapModelCall(ctx, req, handler)
	if err != nil {
		t.Fatalf("WrapModelCall failed: %v", err)
	}

	// 应该返回空列表
	if len(req.Messages) != 0 {
		t.Errorf("Expected empty messages, got %d", len(req.Messages))
	}
}

// TestSummarizationMiddleware_NotEnoughMessagesToSummarize 测试消息数不足的情况
func TestSummarizationMiddleware_NotEnoughMessagesToSummarize(t *testing.T) {
	ctx := context.Background()

	middleware, err := NewSummarizationMiddleware(&SummarizationMiddlewareConfig{
		Summarizer:             mockSummarizer("Summary", false),
		MaxTokensBeforeSummary: 10, // 低阈值
		MessagesToKeep:         5,  // 但要保留 5 条
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// 只有 3 条消息,少于要保留的数量
	req := &ModelRequest{
		Messages: []types.Message{
			{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Message 1"}}},
			{Role: types.MessageRoleAssistant, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Response 1"}}},
			{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Message 2"}}},
		},
	}

	originalLen := len(req.Messages)

	handler := func(ctx context.Context, req *ModelRequest) (*ModelResponse, error) {
		return &ModelResponse{
			Message: types.Message{
				Role:          types.MessageRoleAssistant,
				ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Response"}},
			},
		}, nil
	}

	_, err = middleware.WrapModelCall(ctx, req, handler)
	if err != nil {
		t.Fatalf("WrapModelCall failed: %v", err)
	}

	// 应该保留原始消息
	if len(req.Messages) != originalLen {
		t.Errorf("Expected %d messages (not enough to summarize), got %d",
			originalLen, len(req.Messages))
	}
}

// TestSummarizationMiddleware_CustomTokenCounter 测试自定义 token 计数器
func TestSummarizationMiddleware_CustomTokenCounter(t *testing.T) {
	ctx := context.Background()

	// 自定义计数器: 每条消息算作 100 tokens
	customCounter := func(messages []types.Message) int {
		return len(messages) * 100
	}

	middleware, err := NewSummarizationMiddleware(&SummarizationMiddlewareConfig{
		Summarizer:             mockSummarizer("Summary", false),
		MaxTokensBeforeSummary: 500, // 5 条消息的阈值
		MessagesToKeep:         2,
		TokenCounter:           customCounter,
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// 6 条消息 = 600 tokens,应该触发总结
	req := &ModelRequest{
		Messages: []types.Message{
			{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "1"}}},
			{Role: types.MessageRoleAssistant, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "1"}}},
			{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "2"}}},
			{Role: types.MessageRoleAssistant, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "2"}}},
			{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "3"}}},
			{Role: types.MessageRoleAssistant, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "3"}}},
		},
	}

	handler := func(ctx context.Context, req *ModelRequest) (*ModelResponse, error) {
		return &ModelResponse{
			Message: types.Message{
				Role:          types.MessageRoleAssistant,
				ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Response"}},
			},
		}, nil
	}

	_, err = middleware.WrapModelCall(ctx, req, handler)
	if err != nil {
		t.Fatalf("WrapModelCall failed: %v", err)
	}

	// 应该触发总结
	if middleware.GetSummarizationCount() != 1 {
		t.Errorf("Expected 1 summarization with custom counter, got %d",
			middleware.GetSummarizationCount())
	}

	// 消息数应该减少
	if len(req.Messages) >= 6 {
		t.Error("Expected fewer messages after custom counter triggered summarization")
	}
}

// TestSummarizationMiddleware_ConfigMethods 测试配置方法
func TestSummarizationMiddleware_ConfigMethods(t *testing.T) {
	middleware, err := NewSummarizationMiddleware(&SummarizationMiddlewareConfig{
		Summarizer:             mockSummarizer("Summary", false),
		MaxTokensBeforeSummary: 1000,
		MessagesToKeep:         5,
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// 测试 GetConfig
	config := middleware.GetConfig()
	if config["max_tokens_before_summary"] != 1000 {
		t.Errorf("Expected max_tokens 1000, got %v", config["max_tokens_before_summary"])
	}
	if config["messages_to_keep"] != 5 {
		t.Errorf("Expected messages_to_keep 5, got %v", config["messages_to_keep"])
	}

	// 测试 UpdateConfig
	middleware.UpdateConfig(2000, 10)
	config = middleware.GetConfig()
	if config["max_tokens_before_summary"] != 2000 {
		t.Errorf("Expected updated max_tokens 2000, got %v", config["max_tokens_before_summary"])
	}
	if config["messages_to_keep"] != 10 {
		t.Errorf("Expected updated messages_to_keep 10, got %v", config["messages_to_keep"])
	}

	// 测试计数器
	middleware.summarizationCount = 5
	if middleware.GetSummarizationCount() != 5 {
		t.Errorf("Expected count 5, got %d", middleware.GetSummarizationCount())
	}

	middleware.ResetSummarizationCount()
	if middleware.GetSummarizationCount() != 0 {
		t.Errorf("Expected count 0 after reset, got %d", middleware.GetSummarizationCount())
	}
}

// TestSummarizationMiddleware_DefaultConfig 测试默认配置
func TestSummarizationMiddleware_DefaultConfig(t *testing.T) {
	middleware, err := NewSummarizationMiddleware(&SummarizationMiddlewareConfig{
		// 所有参数使用默认值
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	config := middleware.GetConfig()

	// 验证默认值
	if config["max_tokens_before_summary"] != 170000 {
		t.Errorf("Expected default max_tokens 170000, got %v", config["max_tokens_before_summary"])
	}
	if config["messages_to_keep"] != 6 {
		t.Errorf("Expected default messages_to_keep 6, got %v", config["messages_to_keep"])
	}
	if config["summary_prefix"] != "## Previous conversation summary:" {
		t.Errorf("Expected default prefix, got %v", config["summary_prefix"])
	}
}

// TestSummarizationMiddleware_NilConfig 测试 nil 配置
func TestSummarizationMiddleware_NilConfig(t *testing.T) {
	_, err := NewSummarizationMiddleware(nil)
	if err == nil {
		t.Error("Expected error with nil config")
	}
}

// TestDefaultTokenCounter 测试默认 token 计数器
func TestDefaultTokenCounter(t *testing.T) {
	messages := []types.Message{
		{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "1234"}}},       // 4 chars
		{Role: types.MessageRoleAssistant, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "12345"}}}, // 5 chars
	}

	// 总共 4 + 5 + len("user") + len("assistant") = 9 + 4 + 9 = 22 chars
	// 22 / 4 ≈ 5 tokens
	tokens := defaultTokenCounter(messages)
	expectedMin := 5
	expectedMax := 6

	if tokens < expectedMin || tokens > expectedMax {
		t.Errorf("Expected tokens between %d and %d, got %d", expectedMin, expectedMax, tokens)
	}
}

// TestSummarizationMiddleware_SystemMessagePreservation 测试 System Message 始终保留
func TestSummarizationMiddleware_SystemMessagePreservation(t *testing.T) {
	ctx := context.Background()

	middleware, err := NewSummarizationMiddleware(&SummarizationMiddlewareConfig{
		Summarizer:             mockSummarizer("Summary", false),
		MaxTokensBeforeSummary: 10,
		MessagesToKeep:         1,
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	req := &ModelRequest{
		Messages: []types.Message{
			{Role: types.MessageRoleSystem, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "System instruction 1"}}},
			{Role: types.MessageRoleSystem, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "System instruction 2"}}},
			{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Old message"}}},
			{Role: types.MessageRoleAssistant, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Old response"}}},
			{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Recent message"}}},
		},
	}

	handler := func(ctx context.Context, req *ModelRequest) (*ModelResponse, error) {
		return &ModelResponse{
			Message: types.Message{
				Role:          types.MessageRoleAssistant,
				ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Response"}},
			},
		}, nil
	}

	_, err = middleware.WrapModelCall(ctx, req, handler)
	if err != nil {
		t.Fatalf("WrapModelCall failed: %v", err)
	}

	// 验证 system messages 都被保留
	systemCount := 0
	for _, msg := range req.Messages {
		if msg.Role == types.MessageRoleSystem {
			systemCount++
		}
	}

	// 应该有 2 个原始 system + 1 个总结 system = 3 个
	if systemCount < 3 {
		t.Errorf("Expected at least 3 system messages (2 original + 1 summary), got %d", systemCount)
	}
}

// TestDefaultSummarizer 测试默认总结器
func TestDefaultSummarizer(t *testing.T) {
	ctx := context.Background()

	messages := []types.Message{
		{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Hello"}}},
		{Role: types.MessageRoleAssistant, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "Hi there!"}}},
		{Role: types.MessageRoleUser, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "How are you?"}}},
		{Role: types.MessageRoleAssistant, ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: "I'm good, thanks!"}}},
	}

	summary, err := defaultSummarizer(ctx, messages)
	if err != nil {
		t.Fatalf("defaultSummarizer failed: %v", err)
	}

	if summary == "" {
		t.Error("Expected non-empty summary")
	}

	// 验证摘要包含关键部分
	if !strings.Contains(summary, "Analysis:") {
		t.Errorf("Expected summary to contain 'Analysis:', got: %s", summary)
	}

	if !strings.Contains(summary, "Summary:") {
		t.Errorf("Expected summary to contain 'Summary:', got: %s", summary)
	}
}
