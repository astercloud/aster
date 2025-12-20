package context

import (
	"context"
	"testing"
)

func TestSimpleTokenCounter_Count(t *testing.T) {
	tests := []struct {
		name    string
		config  ModelConfig
		text    string
		wantMin int
		wantMax int
	}{
		{
			name:    "empty text",
			config:  GPT4Config,
			text:    "",
			wantMin: 0,
			wantMax: 0,
		},
		{
			name:    "simple english text",
			config:  GPT4Config,
			text:    "Hello, world!",
			wantMin: 2,
			wantMax: 5,
		},
		{
			name:    "chinese text",
			config:  ClaudeSonnet45Config,
			text:    "你好，世界！",
			wantMin: 1,
			wantMax: 3,
		},
		{
			name:    "mixed text",
			config:  GPT4Config,
			text:    "Hello 世界! This is a test.",
			wantMin: 5,
			wantMax: 10,
		},
		{
			name:    "long text",
			config:  GPT4Config,
			text:    "This is a longer piece of text that should result in more tokens being counted.",
			wantMin: 15,
			wantMax: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := NewSimpleTokenCounter(tt.config)
			got, err := counter.Count(context.Background(), tt.text)
			if err != nil {
				t.Errorf("Count() error = %v", err)
				return
			}
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("Count() = %v, want between %v and %v", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestSimpleTokenCounter_CountBatch(t *testing.T) {
	counter := NewGPT4Counter()
	texts := []string{
		"Hello",
		"World",
		"This is a test",
		"",
	}

	counts, err := counter.CountBatch(context.Background(), texts)
	if err != nil {
		t.Fatalf("CountBatch() error = %v", err)
	}

	if len(counts) != len(texts) {
		t.Errorf("CountBatch() returned %d counts, want %d", len(counts), len(texts))
	}

	// 验证空字符串返回 0
	if counts[3] != 0 {
		t.Errorf("CountBatch() for empty string = %v, want 0", counts[3])
	}

	// 验证非空字符串返回 > 0
	for i := range 3 {
		if counts[i] <= 0 {
			t.Errorf("CountBatch() for text[%d] = %v, want > 0", i, counts[i])
		}
	}
}

func TestSimpleTokenCounter_CountBatch_Empty(t *testing.T) {
	counter := NewGPT4Counter()
	counts, err := counter.CountBatch(context.Background(), []string{})
	if err != nil {
		t.Fatalf("CountBatch() error = %v", err)
	}
	if len(counts) != 0 {
		t.Errorf("CountBatch() for empty input = %v, want []", counts)
	}
}

func TestSimpleTokenCounter_EstimateMessages(t *testing.T) {
	tests := []struct {
		name     string
		counter  *SimpleTokenCounter
		messages []Message
		wantMin  int
		wantMax  int
	}{
		{
			name:     "empty messages",
			counter:  NewGPT4Counter(),
			messages: []Message{},
			wantMin:  0,
			wantMax:  0,
		},
		{
			name:    "single message",
			counter: NewGPT4Counter(),
			messages: []Message{
				{Role: "user", Content: "Hello"},
			},
			wantMin: 3, // content + overhead
			wantMax: 10,
		},
		{
			name:    "multiple messages",
			counter: NewGPT4Counter(),
			messages: []Message{
				{Role: "system", Content: "You are a helpful assistant."},
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there!"},
			},
			wantMin: 15,
			wantMax: 30,
		},
		{
			name:    "claude counter",
			counter: NewClaudeCounter(),
			messages: []Message{
				{Role: "user", Content: "你好"},
			},
			wantMin: 3,
			wantMax: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.counter.EstimateMessages(context.Background(), tt.messages)
			if err != nil {
				t.Errorf("EstimateMessages() error = %v", err)
				return
			}
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("EstimateMessages() = %v, want between %v and %v", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestSimpleTokenCounter_ModelName(t *testing.T) {
	tests := []struct {
		name    string
		counter *SimpleTokenCounter
		want    string
	}{
		{
			name:    "gpt-4",
			counter: NewGPT4Counter(),
			want:    "gpt-4",
		},
		{
			name:    "claude",
			counter: NewClaudeCounter(),
			want:    "claude-sonnet-4-5",
		},
		{
			name:    "custom",
			counter: NewSimpleTokenCounter(DefaultConfig),
			want:    "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.counter.ModelName(); got != tt.want {
				t.Errorf("ModelName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetailedTokenCounter_EstimateMessagesDetailed(t *testing.T) {
	baseCounter := NewGPT4Counter()
	detailedCounter := NewDetailedTokenCounter(baseCounter)

	messages := []Message{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: "Hello"},
	}

	estimate, err := detailedCounter.EstimateMessagesDetailed(context.Background(), messages)
	if err != nil {
		t.Fatalf("EstimateMessagesDetailed() error = %v", err)
	}

	if estimate.TotalTokens <= 0 {
		t.Errorf("TotalTokens = %v, want > 0", estimate.TotalTokens)
	}

	if len(estimate.MessageTokens) != len(messages) {
		t.Errorf("MessageTokens length = %v, want %v", len(estimate.MessageTokens), len(messages))
	}

	if estimate.Breakdown["content"] <= 0 {
		t.Errorf("Breakdown[content] = %v, want > 0", estimate.Breakdown["content"])
	}

	if estimate.Breakdown["overhead"] < 0 {
		t.Errorf("Breakdown[overhead] = %v, want >= 0", estimate.Breakdown["overhead"])
	}
}

func TestDetailedTokenCounter_EstimateMessagesDetailed_Empty(t *testing.T) {
	baseCounter := NewGPT4Counter()
	detailedCounter := NewDetailedTokenCounter(baseCounter)

	estimate, err := detailedCounter.EstimateMessagesDetailed(context.Background(), []Message{})
	if err != nil {
		t.Fatalf("EstimateMessagesDetailed() error = %v", err)
	}

	if estimate.TotalTokens != 0 {
		t.Errorf("TotalTokens = %v, want 0", estimate.TotalTokens)
	}

	if len(estimate.MessageTokens) != 0 {
		t.Errorf("MessageTokens length = %v, want 0", len(estimate.MessageTokens))
	}
}

func TestMultiModelTokenCounter(t *testing.T) {
	multi := NewMultiModelTokenCounter()

	// 注册计数器
	multi.RegisterCounter("gpt-4", NewGPT4Counter())
	multi.RegisterCounter("claude", NewClaudeCounter())

	tests := []struct {
		name      string
		modelName string
		text      string
		wantModel string
	}{
		{
			name:      "exact match gpt-4",
			modelName: "gpt-4",
			text:      "Hello",
			wantModel: "gpt-4",
		},
		{
			name:      "exact match claude",
			modelName: "claude",
			text:      "Hello",
			wantModel: "claude-sonnet-4-5",
		},
		{
			name:      "fuzzy match gpt-4-turbo",
			modelName: "gpt-4-turbo",
			text:      "Hello",
			wantModel: "gpt-4",
		},
		{
			name:      "fuzzy match claude-opus",
			modelName: "claude-3-opus",
			text:      "Hello",
			wantModel: "claude-sonnet-4-5",
		},
		{
			name:      "unknown model uses default",
			modelName: "unknown-model",
			text:      "Hello",
			wantModel: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := multi.GetCounter(tt.modelName)
			if counter.ModelName() != tt.wantModel {
				t.Errorf("GetCounter(%v).ModelName() = %v, want %v", tt.modelName, counter.ModelName(), tt.wantModel)
			}

			// 测试 CountForModel
			count, err := multi.CountForModel(context.Background(), tt.modelName, tt.text)
			if err != nil {
				t.Errorf("CountForModel() error = %v", err)
			}
			if count <= 0 {
				t.Errorf("CountForModel() = %v, want > 0", count)
			}
		})
	}
}

func TestMultiModelTokenCounter_EstimateMessagesForModel(t *testing.T) {
	multi := NewMultiModelTokenCounter()
	multi.RegisterCounter("gpt-4", NewGPT4Counter())

	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	count, err := multi.EstimateMessagesForModel(context.Background(), "gpt-4", messages)
	if err != nil {
		t.Fatalf("EstimateMessagesForModel() error = %v", err)
	}
	if count <= 0 {
		t.Errorf("EstimateMessagesForModel() = %v, want > 0", count)
	}
}

func TestTokenBudget_AvailableTokens(t *testing.T) {
	budget := TokenBudget{
		MaxTokens:      100000,
		ReservedTokens: 4096,
	}

	want := 100000 - 4096
	if got := budget.AvailableTokens(); got != want {
		t.Errorf("AvailableTokens() = %v, want %v", got, want)
	}
}

func TestTokenBudget_IsWithinBudget(t *testing.T) {
	budget := TokenBudget{
		MaxTokens:      100000,
		ReservedTokens: 4096,
	}

	tests := []struct {
		name       string
		usedTokens int
		want       bool
	}{
		{
			name:       "well within budget",
			usedTokens: 50000,
			want:       true,
		},
		{
			name:       "exactly at limit",
			usedTokens: 95904, // 100000 - 4096
			want:       true,
		},
		{
			name:       "over budget",
			usedTokens: 96000,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := budget.IsWithinBudget(tt.usedTokens); got != tt.want {
				t.Errorf("IsWithinBudget(%v) = %v, want %v", tt.usedTokens, got, tt.want)
			}
		})
	}
}

func TestTokenBudget_ShouldWarn(t *testing.T) {
	budget := TokenBudget{
		MaxTokens:        100000,
		ReservedTokens:   4096,
		WarningThreshold: 0.8,
	}

	tests := []struct {
		name       string
		usedTokens int
		want       bool
	}{
		{
			name:       "below threshold",
			usedTokens: 50000,
			want:       false,
		},
		{
			name:       "at threshold",
			usedTokens: 76723, // (100000-4096) * 0.8
			want:       true,
		},
		{
			name:       "above threshold",
			usedTokens: 90000,
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := budget.ShouldWarn(tt.usedTokens); got != tt.want {
				t.Errorf("ShouldWarn(%v) = %v, want %v", tt.usedTokens, got, tt.want)
			}
		})
	}
}

func TestTokenBudget_RemainingTokens(t *testing.T) {
	budget := TokenBudget{
		MaxTokens:      100000,
		ReservedTokens: 4096,
	}

	tests := []struct {
		name       string
		usedTokens int
		want       int
	}{
		{
			name:       "some remaining",
			usedTokens: 50000,
			want:       45904, // (100000-4096) - 50000
		},
		{
			name:       "all used",
			usedTokens: 95904,
			want:       0,
		},
		{
			name:       "over budget",
			usedTokens: 100000,
			want:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := budget.RemainingTokens(tt.usedTokens); got != tt.want {
				t.Errorf("RemainingTokens(%v) = %v, want %v", tt.usedTokens, got, tt.want)
			}
		})
	}
}

func TestTokenBudget_UsagePercentage(t *testing.T) {
	budget := TokenBudget{
		MaxTokens:      100000,
		ReservedTokens: 4096,
	}

	tests := []struct {
		name       string
		usedTokens int
		wantMin    float64
		wantMax    float64
	}{
		{
			name:       "50% usage",
			usedTokens: 47952, // (100000-4096) * 0.5
			wantMin:    49.0,
			wantMax:    51.0,
		},
		{
			name:       "80% usage",
			usedTokens: 76723, // (100000-4096) * 0.8
			wantMin:    79.0,
			wantMax:    81.0,
		},
		{
			name:       "100% usage",
			usedTokens: 95904,
			wantMin:    99.0,
			wantMax:    100.0,
		},
		{
			name:       "over 100%",
			usedTokens: 100000,
			wantMin:    100.0,
			wantMax:    100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := budget.UsagePercentage(tt.usedTokens)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("UsagePercentage(%v) = %v, want between %v and %v", tt.usedTokens, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestDefaultTokenBudget(t *testing.T) {
	budget := DefaultTokenBudget()

	if budget.MaxTokens <= 0 {
		t.Errorf("MaxTokens = %v, want > 0", budget.MaxTokens)
	}

	if budget.ReservedTokens <= 0 {
		t.Errorf("ReservedTokens = %v, want > 0", budget.ReservedTokens)
	}

	if budget.WarningThreshold <= 0 || budget.WarningThreshold >= 1 {
		t.Errorf("WarningThreshold = %v, want between 0 and 1", budget.WarningThreshold)
	}
}

// Benchmark tests
func BenchmarkSimpleTokenCounter_Count(b *testing.B) {
	counter := NewGPT4Counter()
	text := "This is a sample text for benchmarking token counting performance."
	ctx := context.Background()

	for b.Loop() {
		_, _ = counter.Count(ctx, text)
	}
}

func BenchmarkSimpleTokenCounter_EstimateMessages(b *testing.B) {
	counter := NewGPT4Counter()
	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "Hello, how are you?"},
		{Role: "assistant", Content: "I'm doing great, thank you for asking!"},
	}
	ctx := context.Background()

	for b.Loop() {
		_, _ = counter.EstimateMessages(ctx, messages)
	}
}
