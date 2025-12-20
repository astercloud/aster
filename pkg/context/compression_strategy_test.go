package context

import (
	"context"
	"testing"
)

func createTestMessages(count int) []Message {
	messages := make([]Message, count)
	for i := range count {
		role := "user"
		if i%2 == 0 {
			role = "assistant"
		}
		messages[i] = Message{
			Role:    role,
			Content: "Test message",
		}
	}
	return messages
}

func TestSlidingWindowStrategy_Compress(t *testing.T) {
	config := DefaultWindowManagerConfig()
	config.AlwaysKeepSystem = true
	config.AlwaysKeepRecent = 2

	tests := []struct {
		name       string
		windowSize int
		messages   []Message
		wantMax    int
	}{
		{
			name:       "no compression needed",
			windowSize: 10,
			messages:   createTestMessages(5),
			wantMax:    5,
		},
		{
			name:       "compress to window size",
			windowSize: 3,
			messages:   createTestMessages(10),
			wantMax:    5, // window size + system + recent
		},
		{
			name:       "empty messages",
			windowSize: 5,
			messages:   []Message{},
			wantMax:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := NewSlidingWindowStrategy(tt.windowSize)

			compressed, err := strategy.Compress(context.Background(), tt.messages, config)
			if err != nil {
				t.Fatalf("Compress failed: %v", err)
			}

			if len(compressed) > tt.wantMax {
				t.Errorf("compressed length = %d, want <= %d", len(compressed), tt.wantMax)
			}
		})
	}
}

func TestSlidingWindowStrategy_KeepSystemMessages(t *testing.T) {
	config := DefaultWindowManagerConfig()
	config.AlwaysKeepSystem = true
	config.AlwaysKeepRecent = 1

	messages := []Message{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: "Message 1"},
		{Role: "assistant", Content: "Response 1"},
		{Role: "user", Content: "Message 2"},
		{Role: "assistant", Content: "Response 2"},
	}

	strategy := NewSlidingWindowStrategy(2)
	compressed, err := strategy.Compress(context.Background(), messages, config)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// 应该保留 system 消息
	hasSystem := false
	for _, msg := range compressed {
		if msg.Role == "system" {
			hasSystem = true
			break
		}
	}

	if !hasSystem {
		t.Error("system message should be kept")
	}
}

func TestSlidingWindowStrategy_KeepRecentMessages(t *testing.T) {
	config := DefaultWindowManagerConfig()
	config.AlwaysKeepSystem = false
	config.AlwaysKeepRecent = 3

	messages := []Message{
		{Role: "user", Content: "Old 1"},
		{Role: "assistant", Content: "Old 2"},
		{Role: "user", Content: "Recent 1"},
		{Role: "assistant", Content: "Recent 2"},
		{Role: "user", Content: "Recent 3"},
	}

	strategy := NewSlidingWindowStrategy(2)
	compressed, err := strategy.Compress(context.Background(), messages, config)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// 应该至少保留最近的 3 条消息
	if len(compressed) < 3 {
		t.Errorf("should keep at least 3 recent messages, got %d", len(compressed))
	}

	// 验证最后 3 条消息是否保留
	lastMessages := messages[len(messages)-3:]
	foundCount := 0
	for _, last := range lastMessages {
		for _, comp := range compressed {
			if comp.Content == last.Content {
				foundCount++
				break
			}
		}
	}

	if foundCount < 3 {
		t.Errorf("should keep last 3 messages, only found %d", foundCount)
	}
}

func TestSlidingWindowStrategy_Name(t *testing.T) {
	strategy := NewSlidingWindowStrategy(10)
	if strategy.Name() != "sliding-window" {
		t.Errorf("Name() = %q, want %q", strategy.Name(), "sliding-window")
	}
}

func TestPriorityBasedStrategy_Compress(t *testing.T) {
	config := DefaultWindowManagerConfig()
	config.AlwaysKeepSystem = true
	config.AlwaysKeepRecent = 1

	messages := []Message{
		{Role: "system", Content: "System message"},
		{Role: "user", Content: "User 1"},
		{Role: "assistant", Content: "Assistant 1"},
		{Role: "user", Content: "User 2"},
		{Role: "assistant", Content: "Assistant 2"},
	}

	calc := NewDefaultPriorityCalculator()
	strategy := NewPriorityBasedStrategy(3, calc)

	compressed, err := strategy.Compress(context.Background(), messages, config)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	if len(compressed) > 4 {
		t.Errorf("compressed length = %d, want <= 4", len(compressed))
	}

	// 应该保留 system 消息
	hasSystem := false
	for _, msg := range compressed {
		if msg.Role == "system" {
			hasSystem = true
			break
		}
	}

	if !hasSystem {
		t.Error("system message should be kept")
	}
}

func TestPriorityBasedStrategy_NoCompression(t *testing.T) {
	config := DefaultWindowManagerConfig()
	messages := createTestMessages(3)

	strategy := NewPriorityBasedStrategy(10, nil) // targetSize > messages

	compressed, err := strategy.Compress(context.Background(), messages, config)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	if len(compressed) != len(messages) {
		t.Errorf("should not compress when targetSize > messages: got %d, want %d",
			len(compressed), len(messages))
	}
}

func TestPriorityBasedStrategy_Name(t *testing.T) {
	strategy := NewPriorityBasedStrategy(10, nil)
	if strategy.Name() != "priority-based" {
		t.Errorf("Name() = %q, want %q", strategy.Name(), "priority-based")
	}
}

func TestTokenBasedStrategy_Compress(t *testing.T) {
	config := DefaultWindowManagerConfig()
	config.Budget.MaxTokens = 1000
	config.Budget.ReservedTokens = 100
	config.AlwaysKeepSystem = true
	config.AlwaysKeepRecent = 2

	messages := []Message{
		{Role: "system", Content: "System message"},
		{Role: "user", Content: "This is a long message that will consume many tokens. " +
			"It contains a lot of text to simulate a real conversation."},
		{Role: "assistant", Content: "This is another long response with lots of information."},
		{Role: "user", Content: "More conversation here."},
		{Role: "assistant", Content: "Final response."},
	}

	counter := NewGPT4Counter()
	strategy := NewTokenBasedStrategy(counter, 0.5) // 目标 50%

	compressed, err := strategy.Compress(context.Background(), messages, config)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// 验证压缩后的 Token 数
	compressedTokens, err := counter.EstimateMessages(context.Background(), compressed)
	if err != nil {
		t.Fatalf("failed to estimate compressed tokens: %v", err)
	}

	targetTokens := int(float64(config.Budget.AvailableTokens()) * 0.5)
	if compressedTokens > targetTokens*2 {
		t.Errorf("compressed tokens = %d, should be closer to target %d",
			compressedTokens, targetTokens)
	}
}

func TestTokenBasedStrategy_NoCompression(t *testing.T) {
	config := DefaultWindowManagerConfig()
	config.Budget.MaxTokens = 100000 // 很大的预算
	config.Budget.ReservedTokens = 1000

	messages := createTestMessages(5)

	counter := NewGPT4Counter()
	strategy := NewTokenBasedStrategy(counter, 0.7)

	compressed, err := strategy.Compress(context.Background(), messages, config)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// 不应该压缩（因为预算足够）
	if len(compressed) != len(messages) {
		t.Errorf("should not compress when within budget: got %d, want %d",
			len(compressed), len(messages))
	}
}

func TestTokenBasedStrategy_RespectMinMessages(t *testing.T) {
	config := DefaultWindowManagerConfig()
	config.Budget.MaxTokens = 50 // 很小的预算
	config.Budget.ReservedTokens = 10
	config.MinMessagesToKeep = 3

	messages := []Message{
		{Role: "user", Content: "Message 1"},
		{Role: "assistant", Content: "Response 1"},
		{Role: "user", Content: "Message 2"},
		{Role: "assistant", Content: "Response 2"},
	}

	counter := NewGPT4Counter()
	strategy := NewTokenBasedStrategy(counter, 0.5)

	compressed, err := strategy.Compress(context.Background(), messages, config)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// 应该至少保留 MinMessagesToKeep 条消息
	if len(compressed) < config.MinMessagesToKeep {
		t.Errorf("should keep at least %d messages, got %d",
			config.MinMessagesToKeep, len(compressed))
	}
}

func TestTokenBasedStrategy_Name(t *testing.T) {
	counter := NewGPT4Counter()
	strategy := NewTokenBasedStrategy(counter, 0.7)
	if strategy.Name() != "token-based" {
		t.Errorf("Name() = %q, want %q", strategy.Name(), "token-based")
	}
}

func TestHybridStrategy_Compress(t *testing.T) {
	config := DefaultWindowManagerConfig()
	config.AlwaysKeepSystem = true
	config.AlwaysKeepRecent = 2

	messages := []Message{
		{Role: "system", Content: "System"},
		{Role: "user", Content: "User 1"},
		{Role: "assistant", Content: "Assistant 1"},
		{Role: "user", Content: "User 2"},
		{Role: "assistant", Content: "Assistant 2"},
		{Role: "user", Content: "User 3"},
	}

	// 组合多个策略
	strategies := []CompressionStrategy{
		NewSlidingWindowStrategy(3),
		NewPriorityBasedStrategy(3, NewDefaultPriorityCalculator()),
	}

	strategy := NewHybridStrategy(strategies, nil)

	compressed, err := strategy.Compress(context.Background(), messages, config)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// 混合策略应该压缩消息
	if len(compressed) >= len(messages) {
		t.Errorf("hybrid strategy should compress: got %d, original %d",
			len(compressed), len(messages))
	}

	// 应该保留 system 消息
	hasSystem := false
	for _, msg := range compressed {
		if msg.Role == "system" {
			hasSystem = true
			break
		}
	}

	if !hasSystem {
		t.Error("system message should be kept")
	}
}

func TestHybridStrategy_EmptyStrategies(t *testing.T) {
	config := DefaultWindowManagerConfig()
	messages := createTestMessages(5)

	strategy := NewHybridStrategy([]CompressionStrategy{}, []float64{})

	compressed, err := strategy.Compress(context.Background(), messages, config)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// 空策略应该返回原始消息
	if len(compressed) != len(messages) {
		t.Errorf("empty strategy should not change messages: got %d, want %d",
			len(compressed), len(messages))
	}
}

func TestHybridStrategy_Name(t *testing.T) {
	strategy := NewHybridStrategy([]CompressionStrategy{}, nil)
	if strategy.Name() != "hybrid" {
		t.Errorf("Name() = %q, want %q", strategy.Name(), "hybrid")
	}
}

func TestNewSlidingWindowStrategy_DefaultSize(t *testing.T) {
	// 测试无效窗口大小时的默认值
	strategy := NewSlidingWindowStrategy(0)
	if strategy.windowSize != 10 {
		t.Errorf("default window size = %d, want 10", strategy.windowSize)
	}

	strategy = NewSlidingWindowStrategy(-5)
	if strategy.windowSize != 10 {
		t.Errorf("default window size = %d, want 10", strategy.windowSize)
	}
}

func TestNewPriorityBasedStrategy_Defaults(t *testing.T) {
	// 测试默认值
	strategy := NewPriorityBasedStrategy(0, nil)
	if strategy.targetSize != 10 {
		t.Errorf("default target size = %d, want 10", strategy.targetSize)
	}

	if strategy.priorityCalculator == nil {
		t.Error("should create default priority calculator")
	}
}

func TestNewTokenBasedStrategy_DefaultTargetUsage(t *testing.T) {
	counter := NewGPT4Counter()

	// 测试无效目标使用率
	strategy := NewTokenBasedStrategy(counter, 0)
	if strategy.targetUsage != 0.7 {
		t.Errorf("default target usage = %.2f, want 0.7", strategy.targetUsage)
	}

	strategy = NewTokenBasedStrategy(counter, 1.5)
	if strategy.targetUsage != 0.7 {
		t.Errorf("default target usage = %.2f, want 0.7", strategy.targetUsage)
	}
}

func TestNewHybridStrategy_DefaultWeights(t *testing.T) {
	strategies := []CompressionStrategy{
		NewSlidingWindowStrategy(5),
		NewSlidingWindowStrategy(10),
	}

	// 不提供权重，应该使用均等权重
	strategy := NewHybridStrategy(strategies, []float64{})

	if len(strategy.weights) != len(strategies) {
		t.Errorf("weights length = %d, want %d", len(strategy.weights), len(strategies))
	}

	expectedWeight := 1.0 / float64(len(strategies))
	for i, weight := range strategy.weights {
		if weight != expectedWeight {
			t.Errorf("weight[%d] = %.2f, want %.2f", i, weight, expectedWeight)
		}
	}
}

// Benchmark tests
func BenchmarkSlidingWindowStrategy_Compress(b *testing.B) {
	config := DefaultWindowManagerConfig()
	strategy := NewSlidingWindowStrategy(10)
	messages := createTestMessages(100)
	ctx := context.Background()

	for b.Loop() {
		_, _ = strategy.Compress(ctx, messages, config)
	}
}

func BenchmarkPriorityBasedStrategy_Compress(b *testing.B) {
	config := DefaultWindowManagerConfig()
	calc := NewDefaultPriorityCalculator()
	strategy := NewPriorityBasedStrategy(10, calc)
	messages := createTestMessages(100)
	ctx := context.Background()

	for b.Loop() {
		_, _ = strategy.Compress(ctx, messages, config)
	}
}

func BenchmarkTokenBasedStrategy_Compress(b *testing.B) {
	config := DefaultWindowManagerConfig()
	counter := NewGPT4Counter()
	strategy := NewTokenBasedStrategy(counter, 0.7)
	messages := createTestMessages(100)
	ctx := context.Background()

	for b.Loop() {
		_, _ = strategy.Compress(ctx, messages, config)
	}
}
