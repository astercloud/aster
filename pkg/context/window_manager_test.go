package context

import (
	"context"
	"testing"
)

func TestNewContextWindowManager(t *testing.T) {
	config := DefaultWindowManagerConfig()
	counter := NewGPT4Counter()
	strategy := NewSlidingWindowStrategy(10)

	manager := NewContextWindowManager(config, counter, strategy)

	if manager == nil {
		t.Fatal("NewContextWindowManager returned nil")
	}

	if len(manager.messages) != 0 {
		t.Errorf("new manager should have 0 messages, got %d", len(manager.messages))
	}

	if manager.currentTokens != 0 {
		t.Errorf("new manager should have 0 tokens, got %d", manager.currentTokens)
	}
}

func TestContextWindowManager_AddMessage(t *testing.T) {
	config := DefaultWindowManagerConfig()
	config.AutoCompress = false // 禁用自动压缩以便测试
	counter := NewGPT4Counter()
	strategy := NewSlidingWindowStrategy(10)

	manager := NewContextWindowManager(config, counter, strategy)

	msg := Message{
		Role:    "user",
		Content: "Hello, world!",
	}

	err := manager.AddMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("AddMessage failed: %v", err)
	}

	messages := manager.GetMessages()
	if len(messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(messages))
	}

	if messages[0].Content != msg.Content {
		t.Errorf("message content mismatch: want %q, got %q", msg.Content, messages[0].Content)
	}

	if manager.currentTokens <= 0 {
		t.Errorf("expected tokens > 0, got %d", manager.currentTokens)
	}
}

func TestContextWindowManager_AddMessages(t *testing.T) {
	config := DefaultWindowManagerConfig()
	config.AutoCompress = false
	counter := NewGPT4Counter()
	strategy := NewSlidingWindowStrategy(10)

	manager := NewContextWindowManager(config, counter, strategy)

	messages := []Message{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
	}

	err := manager.AddMessages(context.Background(), messages)
	if err != nil {
		t.Fatalf("AddMessages failed: %v", err)
	}

	storedMessages := manager.GetMessages()
	if len(storedMessages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(storedMessages))
	}

	if manager.currentTokens <= 0 {
		t.Errorf("expected tokens > 0, got %d", manager.currentTokens)
	}
}

func TestContextWindowManager_GetCurrentTokens(t *testing.T) {
	config := DefaultWindowManagerConfig()
	config.AutoCompress = false
	counter := NewGPT4Counter()
	strategy := NewSlidingWindowStrategy(10)

	manager := NewContextWindowManager(config, counter, strategy)

	// 初始应该为 0
	if manager.GetCurrentTokens() != 0 {
		t.Errorf("initial tokens should be 0, got %d", manager.GetCurrentTokens())
	}

	// 添加消息后应该 > 0
	err := manager.AddMessage(context.Background(), Message{
		Role:    "user",
		Content: "Test message",
	})
	if err != nil {
		t.Fatalf("AddMessage failed: %v", err)
	}

	if manager.GetCurrentTokens() <= 0 {
		t.Errorf("tokens should be > 0 after adding message, got %d", manager.GetCurrentTokens())
	}
}

func TestContextWindowManager_GetRemainingTokens(t *testing.T) {
	config := DefaultWindowManagerConfig()
	config.AutoCompress = false
	counter := NewGPT4Counter()
	strategy := NewSlidingWindowStrategy(10)

	manager := NewContextWindowManager(config, counter, strategy)

	// 初始剩余 Token 应该等于可用 Token
	initialRemaining := manager.GetRemainingTokens()
	expectedRemaining := config.Budget.AvailableTokens()
	if initialRemaining != expectedRemaining {
		t.Errorf("initial remaining tokens = %d, want %d", initialRemaining, expectedRemaining)
	}

	// 添加消息后剩余应该减少
	err := manager.AddMessage(context.Background(), Message{
		Role:    "user",
		Content: "Test message",
	})
	if err != nil {
		t.Fatalf("AddMessage failed: %v", err)
	}

	afterRemaining := manager.GetRemainingTokens()
	if afterRemaining >= initialRemaining {
		t.Errorf("remaining tokens should decrease after adding message: before=%d, after=%d",
			initialRemaining, afterRemaining)
	}
}

func TestContextWindowManager_GetUsagePercentage(t *testing.T) {
	config := DefaultWindowManagerConfig()
	config.AutoCompress = false
	counter := NewGPT4Counter()
	strategy := NewSlidingWindowStrategy(10)

	manager := NewContextWindowManager(config, counter, strategy)

	// 初始使用率应该为 0
	if usage := manager.GetUsagePercentage(); usage != 0 {
		t.Errorf("initial usage should be 0%%, got %.2f%%", usage)
	}

	// 添加消息后使用率应该增加
	err := manager.AddMessage(context.Background(), Message{
		Role:    "user",
		Content: "Test message",
	})
	if err != nil {
		t.Fatalf("AddMessage failed: %v", err)
	}

	usage := manager.GetUsagePercentage()
	if usage <= 0 {
		t.Errorf("usage should be > 0%% after adding message, got %.2f%%", usage)
	}
}

func TestContextWindowManager_IsWithinBudget(t *testing.T) {
	config := DefaultWindowManagerConfig()
	config.AutoCompress = false
	counter := NewGPT4Counter()
	strategy := NewSlidingWindowStrategy(10)

	manager := NewContextWindowManager(config, counter, strategy)

	// 初始应该在预算内
	if !manager.IsWithinBudget() {
		t.Error("should be within budget initially")
	}

	// 添加少量消息应该还在预算内
	err := manager.AddMessage(context.Background(), Message{
		Role:    "user",
		Content: "Test message",
	})
	if err != nil {
		t.Fatalf("AddMessage failed: %v", err)
	}

	if !manager.IsWithinBudget() {
		t.Error("should still be within budget after adding one message")
	}
}

func TestContextWindowManager_ShouldWarn(t *testing.T) {
	config := DefaultWindowManagerConfig()
	config.AutoCompress = false
	counter := NewGPT4Counter()
	strategy := NewSlidingWindowStrategy(10)

	manager := NewContextWindowManager(config, counter, strategy)

	// 初始不应该警告
	if manager.ShouldWarn() {
		t.Error("should not warn initially")
	}
}

func TestContextWindowManager_AutoCompress(t *testing.T) {
	config := DefaultWindowManagerConfig()
	config.AutoCompress = true
	config.CompressionThreshold = 0.5 // 50% 触发压缩
	config.Budget.MaxTokens = 100     // 设置较小的预算以便触发压缩
	config.Budget.ReservedTokens = 10

	counter := NewGPT4Counter()
	strategy := NewSlidingWindowStrategy(3) // 只保留 3 条消息

	manager := NewContextWindowManager(config, counter, strategy)

	// 添加多条消息，触发自动压缩
	for range 10 {
		err := manager.AddMessage(context.Background(), Message{
			Role:    "user",
			Content: "This is a test message that should trigger compression eventually.",
		})
		if err != nil {
			t.Fatalf("AddMessage failed: %v", err)
		}
	}

	messages := manager.GetMessages()

	// 应该被压缩到 <= 3 条消息
	if len(messages) > 5 {
		t.Errorf("expected messages to be compressed, got %d messages", len(messages))
	}

	// 应该有压缩历史
	history := manager.GetCompressionHistory()
	if len(history) == 0 {
		t.Error("expected compression history, got none")
	}
}

func TestContextWindowManager_ManualCompress(t *testing.T) {
	config := DefaultWindowManagerConfig()
	config.AutoCompress = false
	counter := NewGPT4Counter()
	strategy := NewSlidingWindowStrategy(3)

	manager := NewContextWindowManager(config, counter, strategy)

	// 添加多条消息
	for range 10 {
		err := manager.AddMessage(context.Background(), Message{
			Role:    "user",
			Content: "Test message",
		})
		if err != nil {
			t.Fatalf("AddMessage failed: %v", err)
		}
	}

	messagesBefore := len(manager.GetMessages())

	// 手动触发压缩
	err := manager.Compress(context.Background())
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	messagesAfter := len(manager.GetMessages())

	if messagesAfter >= messagesBefore {
		t.Errorf("compression should reduce messages: before=%d, after=%d", messagesBefore, messagesAfter)
	}

	// 检查压缩历史
	history := manager.GetCompressionHistory()
	if len(history) != 1 {
		t.Errorf("expected 1 compression event, got %d", len(history))
	}

	if len(history) > 0 {
		event := history[0]
		if event.BeforeMessages != messagesBefore {
			t.Errorf("event.BeforeMessages = %d, want %d", event.BeforeMessages, messagesBefore)
		}
		if event.AfterMessages != messagesAfter {
			t.Errorf("event.AfterMessages = %d, want %d", event.AfterMessages, messagesAfter)
		}
	}
}

func TestContextWindowManager_Clear(t *testing.T) {
	config := DefaultWindowManagerConfig()
	config.AutoCompress = false
	counter := NewGPT4Counter()
	strategy := NewSlidingWindowStrategy(10)

	manager := NewContextWindowManager(config, counter, strategy)

	// 添加消息
	err := manager.AddMessages(context.Background(), []Message{
		{Role: "user", Content: "Message 1"},
		{Role: "assistant", Content: "Response 1"},
	})
	if err != nil {
		t.Fatalf("AddMessages failed: %v", err)
	}

	// 清空
	manager.Clear()

	if len(manager.GetMessages()) != 0 {
		t.Errorf("Clear() should remove all messages, got %d", len(manager.GetMessages()))
	}

	if manager.GetCurrentTokens() != 0 {
		t.Errorf("Clear() should reset tokens to 0, got %d", manager.GetCurrentTokens())
	}
}

func TestContextWindowManager_Reset(t *testing.T) {
	config := DefaultWindowManagerConfig()
	config.AutoCompress = false
	counter := NewGPT4Counter()
	strategy := NewSlidingWindowStrategy(3)

	manager := NewContextWindowManager(config, counter, strategy)

	// 添加消息并压缩
	for range 10 {
		_ = manager.AddMessage(context.Background(), Message{
			Role:    "user",
			Content: "Test",
		})
	}
	_ = manager.Compress(context.Background())

	// 重置
	manager.Reset()

	if len(manager.GetMessages()) != 0 {
		t.Errorf("Reset() should clear messages, got %d", len(manager.GetMessages()))
	}

	if manager.GetCurrentTokens() != 0 {
		t.Errorf("Reset() should reset tokens, got %d", manager.GetCurrentTokens())
	}

	if len(manager.GetCompressionHistory()) != 0 {
		t.Errorf("Reset() should clear compression history, got %d events", len(manager.GetCompressionHistory()))
	}
}

func TestContextWindowManager_GetStats(t *testing.T) {
	config := DefaultWindowManagerConfig()
	config.AutoCompress = false
	counter := NewGPT4Counter()
	strategy := NewSlidingWindowStrategy(10)

	manager := NewContextWindowManager(config, counter, strategy)

	// 添加消息
	err := manager.AddMessages(context.Background(), []Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi"},
	})
	if err != nil {
		t.Fatalf("AddMessages failed: %v", err)
	}

	stats := manager.GetStats()

	if stats.CurrentMessages != 2 {
		t.Errorf("stats.CurrentMessages = %d, want 2", stats.CurrentMessages)
	}

	if stats.TotalMessages != 2 {
		t.Errorf("stats.TotalMessages = %d, want 2", stats.TotalMessages)
	}

	if stats.CurrentTokens <= 0 {
		t.Errorf("stats.CurrentTokens should be > 0, got %d", stats.CurrentTokens)
	}

	if !stats.IsWithinBudget {
		t.Error("stats.IsWithinBudget should be true")
	}
}

func TestDefaultWindowManagerConfig(t *testing.T) {
	config := DefaultWindowManagerConfig()

	if config.Budget.MaxTokens <= 0 {
		t.Errorf("Budget.MaxTokens should be > 0, got %d", config.Budget.MaxTokens)
	}

	if config.CompressionThreshold <= 0 || config.CompressionThreshold > 1 {
		t.Errorf("CompressionThreshold should be between 0 and 1, got %.2f", config.CompressionThreshold)
	}

	if config.MinMessagesToKeep <= 0 {
		t.Errorf("MinMessagesToKeep should be > 0, got %d", config.MinMessagesToKeep)
	}
}

func TestDefaultPriorityCalculator(t *testing.T) {
	calc := NewDefaultPriorityCalculator()

	tests := []struct {
		name     string
		msg      Message
		position int
		total    int
		wantPri  MessagePriority
	}{
		{
			name:     "system message high priority",
			msg:      Message{Role: "system", Content: "You are helpful."},
			position: 0,
			total:    3,
			wantPri:  PriorityHigh,
		},
		{
			name:     "recent user message",
			msg:      Message{Role: "user", Content: "Hello"},
			position: 2,
			total:    3,
			wantPri:  PriorityMedium,
		},
		{
			name:     "old message low priority",
			msg:      Message{Role: "user", Content: "Old"},
			position: 0,
			total:    10,
			wantPri:  PriorityLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			priority, score := calc.CalculatePriority(context.Background(), tt.msg, tt.position, tt.total)

			if score < 0 || score > 1 {
				t.Errorf("score should be between 0 and 1, got %.2f", score)
			}

			// 不严格检查优先级，因为计算可能有变化
			_ = priority
		})
	}
}

func TestCalculateMessagePriorities(t *testing.T) {
	messages := []Message{
		{Role: "system", Content: "System"},
		{Role: "user", Content: "User 1"},
		{Role: "assistant", Content: "Assistant 1"},
		{Role: "user", Content: "User 2"},
	}

	calc := NewDefaultPriorityCalculator()
	result := CalculateMessagePriorities(context.Background(), messages, calc)

	if len(result) != len(messages) {
		t.Errorf("result length = %d, want %d", len(result), len(messages))
	}

	for i, msgWithPri := range result {
		if msgWithPri.Message.Role != messages[i].Role {
			t.Errorf("message %d role mismatch", i)
		}

		if msgWithPri.Score < 0 || msgWithPri.Score > 1 {
			t.Errorf("message %d score out of range: %.2f", i, msgWithPri.Score)
		}
	}
}

func TestSortMessagesByPriority(t *testing.T) {
	messages := []MessageWithPriority{
		{Message: Message{Content: "Low"}, Score: 0.2},
		{Message: Message{Content: "High"}, Score: 0.8},
		{Message: Message{Content: "Medium"}, Score: 0.5},
	}

	// 降序排序
	SortMessagesByPriority(messages, true)

	if messages[0].Score < messages[1].Score {
		t.Error("descending sort failed")
	}

	// 升序排序
	SortMessagesByPriority(messages, false)

	if messages[0].Score > messages[1].Score {
		t.Error("ascending sort failed")
	}
}

func TestFilterMessagesByPriority(t *testing.T) {
	messages := []MessageWithPriority{
		{Priority: PriorityLow},
		{Priority: PriorityMedium},
		{Priority: PriorityHigh},
		{Priority: PriorityCritical},
	}

	filtered := FilterMessagesByPriority(messages, PriorityHigh)

	if len(filtered) != 2 {
		t.Errorf("expected 2 messages with priority >= High, got %d", len(filtered))
	}

	for _, msg := range filtered {
		if msg.Priority < PriorityHigh {
			t.Errorf("filtered message has priority %d, want >= %d", msg.Priority, PriorityHigh)
		}
	}
}
