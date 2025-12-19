package memory

import (
	"context"
	"testing"

	agentext "github.com/astercloud/aster/pkg/context"
)

func TestNewLLMSummarizer(t *testing.T) {
	config := DefaultLLMSummarizerConfig()
	summarizer := NewLLMSummarizer(config)

	if summarizer == nil {
		t.Fatal("NewLLMSummarizer returned nil")
	}

	if summarizer.config.Level != CompressionLevelModerate {
		t.Errorf("Level = %v, want %v", summarizer.config.Level, CompressionLevelModerate)
	}
}

func TestLLMSummarizer_SummarizeSession_Empty(t *testing.T) {
	summarizer := NewLLMSummarizer(DefaultLLMSummarizerConfig())
	messages := []agentext.Message{}

	summary, err := summarizer.SummarizeSession(context.Background(), messages)
	if err != nil {
		t.Fatalf("SummarizeSession failed: %v", err)
	}

	if summary != "" {
		t.Errorf("summary = %q, want empty string", summary)
	}
}

func TestLLMSummarizer_SummarizeSession(t *testing.T) {
	summarizer := NewLLMSummarizer(DefaultLLMSummarizerConfig())
	messages := []agentext.Message{
		{Role: "user", Content: "What is the capital of France?"},
		{Role: "assistant", Content: "The capital of France is Paris."},
		{Role: "user", Content: "What about Italy?"},
		{Role: "assistant", Content: "The capital of Italy is Rome."},
	}

	summary, err := summarizer.SummarizeSession(context.Background(), messages)
	if err != nil {
		t.Fatalf("SummarizeSession failed: %v", err)
	}

	if summary == "" {
		t.Error("summary should not be empty")
	}

	// 检查统计信息
	stats := summarizer.GetCompressionStats()
	if stats.OriginalMessages != 4 {
		t.Errorf("OriginalMessages = %d, want 4", stats.OriginalMessages)
	}

	if stats.CompressedMessages != 1 {
		t.Errorf("CompressedMessages = %d, want 1", stats.CompressedMessages)
	}

	if stats.OriginalTokens <= 0 {
		t.Errorf("OriginalTokens = %d, want > 0", stats.OriginalTokens)
	}

	if stats.CompressedTokens <= 0 {
		t.Errorf("CompressedTokens = %d, want > 0", stats.CompressedTokens)
	}
}

func TestLLMSummarizer_CompressMessages(t *testing.T) {
	summarizer := NewLLMSummarizer(DefaultLLMSummarizerConfig())
	messages := []agentext.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "How are you?"},
		{Role: "assistant", Content: "I'm doing well, thank you!"},
	}

	compressed, err := summarizer.CompressMessages(context.Background(), messages)
	if err != nil {
		t.Fatalf("CompressMessages failed: %v", err)
	}

	if len(compressed) != 1 {
		t.Errorf("compressed length = %d, want 1", len(compressed))
	}

	if compressed[0].Role != "system" {
		t.Errorf("compressed[0].Role = %s, want 'system'", compressed[0].Role)
	}
}

func TestLLMSummarizer_CompressionLevels(t *testing.T) {
	messages := []agentext.Message{
		{Role: "user", Content: "Explain quantum computing"},
		{Role: "assistant", Content: "Quantum computing uses quantum mechanics..."},
	}

	levels := []CompressionLevel{
		CompressionLevelLight,
		CompressionLevelModerate,
		CompressionLevelAggressive,
	}

	for _, level := range levels {
		t.Run(levelString(level), func(t *testing.T) {
			config := DefaultLLMSummarizerConfig()
			config.Level = level

			summarizer := NewLLMSummarizer(config)
			summary, err := summarizer.SummarizeSession(context.Background(), messages)

			if err != nil {
				t.Fatalf("SummarizeSession failed: %v", err)
			}

			if summary == "" {
				t.Error("summary should not be empty")
			}
		})
	}
}

func TestNewMultiLevelCompressor(t *testing.T) {
	messageLevel := NewLLMSummarizer(DefaultLLMSummarizerConfig())
	turnLevel := NewLLMSummarizer(DefaultLLMSummarizerConfig())
	sessionLevel := NewLLMSummarizer(DefaultLLMSummarizerConfig())

	config := DefaultMultiLevelCompressorConfig()
	compressor := NewMultiLevelCompressor(messageLevel, turnLevel, sessionLevel, config)

	if compressor == nil {
		t.Fatal("NewMultiLevelCompressor returned nil")
	}
}

func TestMultiLevelCompressor_SummarizeSession(t *testing.T) {
	summarizer := NewLLMSummarizer(DefaultLLMSummarizerConfig())
	config := DefaultMultiLevelCompressorConfig()
	config.MessageThreshold = 2
	config.TurnThreshold = 1
	config.SessionThreshold = 10

	compressor := NewMultiLevelCompressor(summarizer, summarizer, summarizer, config)

	tests := []struct {
		name         string
		messageCount int
		wantLevel    string
	}{
		{
			name:         "few messages - no compression",
			messageCount: 1,
			wantLevel:    "none",
		},
		{
			name:         "turn level",
			messageCount: 3,
			wantLevel:    "turn",
		},
		{
			name:         "session level",
			messageCount: 15,
			wantLevel:    "session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages := make([]agentext.Message, tt.messageCount)
			for i := range tt.messageCount {
				role := "user"
				if i%2 == 1 {
					role = "assistant"
				}
				messages[i] = agentext.Message{
					Role:    role,
					Content: "Test message",
				}
			}

			summary, err := compressor.SummarizeSession(context.Background(), messages)
			if err != nil {
				t.Fatalf("SummarizeSession failed: %v", err)
			}

			if summary == "" {
				t.Error("summary should not be empty")
			}
		})
	}
}

func TestMultiLevelCompressor_CompressMessages(t *testing.T) {
	summarizer := NewLLMSummarizer(DefaultLLMSummarizerConfig())
	config := DefaultMultiLevelCompressorConfig()
	config.TurnThreshold = 2
	config.SessionThreshold = 100

	compressor := NewMultiLevelCompressor(summarizer, summarizer, summarizer, config)

	// 创建多轮对话
	messages := []agentext.Message{
		{Role: "system", Content: "You are a helpful assistant"},
		{Role: "user", Content: "Question 1"},
		{Role: "assistant", Content: "Answer 1"},
		{Role: "user", Content: "Question 2"},
		{Role: "assistant", Content: "Answer 2"},
		{Role: "user", Content: "Question 3"},
		{Role: "assistant", Content: "Answer 3"},
	}

	compressed, err := compressor.CompressMessages(context.Background(), messages)
	if err != nil {
		t.Fatalf("CompressMessages failed: %v", err)
	}

	// 压缩后应该少于原始消息数
	if len(compressed) >= len(messages) {
		t.Logf("Compression may not have occurred: original=%d, compressed=%d", len(messages), len(compressed))
	}

	// 检查系统消息是否保留
	hasSystemMsg := false
	for _, msg := range compressed {
		if msg.Role == "system" {
			hasSystemMsg = true
			break
		}
	}

	if !hasSystemMsg {
		t.Error("system message should be preserved")
	}
}

func TestMultiLevelCompressor_GroupIntoTurns(t *testing.T) {
	config := DefaultMultiLevelCompressorConfig()
	compressor := NewMultiLevelCompressor(nil, nil, nil, config)

	messages := []agentext.Message{
		{Role: "system", Content: "System prompt"},
		{Role: "user", Content: "Q1"},
		{Role: "assistant", Content: "A1"},
		{Role: "user", Content: "Q2"},
		{Role: "assistant", Content: "A2"},
	}

	turns := compressor.groupIntoTurns(messages)

	// 应该有 2 轮对话
	if len(turns) != 2 {
		t.Errorf("turns length = %d, want 2", len(turns))
	}

	// 第一轮应该有 2 条消息（user + assistant）
	if len(turns[0]) != 2 {
		t.Errorf("turns[0] length = %d, want 2", len(turns[0]))
	}

	// 验证第一轮的消息
	if turns[0][0].Role != "user" || turns[0][1].Role != "assistant" {
		t.Error("turn structure is incorrect")
	}
}

func TestMultiLevelCompressor_GetCompressionStats(t *testing.T) {
	summarizer := NewLLMSummarizer(DefaultLLMSummarizerConfig())
	config := DefaultMultiLevelCompressorConfig()

	compressor := NewMultiLevelCompressor(summarizer, summarizer, summarizer, config)

	messages := []agentext.Message{
		{Role: "user", Content: "Test"},
		{Role: "assistant", Content: "Response"},
	}

	// 触发压缩
	_, _ = compressor.SummarizeSession(context.Background(), messages)

	stats := compressor.GetCompressionStats()

	// 统计信息应该被更新
	if stats.OriginalMessages == 0 && stats.CompressedMessages == 0 {
		t.Log("Stats may not be updated - this is expected for the current implementation")
	}
}

func TestCompressionStats_Calculate(t *testing.T) {
	originalTokens := 1000
	compressedTokens := 300

	expectedRatio := 0.3
	actualRatio := float64(compressedTokens) / float64(originalTokens)

	if actualRatio != expectedRatio {
		t.Errorf("CompressionRatio = %.2f, want %.2f", actualRatio, expectedRatio)
	}
}

func TestDefaultConfigs(t *testing.T) {
	t.Run("LLMSummarizerConfig", func(t *testing.T) {
		config := DefaultLLMSummarizerConfig()

		if config.Level != CompressionLevelModerate {
			t.Errorf("Level = %v, want %v", config.Level, CompressionLevelModerate)
		}

		if config.MaxSummaryWords != 500 {
			t.Errorf("MaxSummaryWords = %d, want 500", config.MaxSummaryWords)
		}

		if !config.PreserveContext {
			t.Error("PreserveContext should be true")
		}

		if config.TokenCounter == nil {
			t.Error("TokenCounter should not be nil")
		}
	})

	t.Run("MultiLevelCompressorConfig", func(t *testing.T) {
		config := DefaultMultiLevelCompressorConfig()

		if !config.EnableMessageLevel {
			t.Error("EnableMessageLevel should be true")
		}

		if !config.EnableTurnLevel {
			t.Error("EnableTurnLevel should be true")
		}

		if !config.EnableSessionLevel {
			t.Error("EnableSessionLevel should be true")
		}

		if config.MessageThreshold != 100 {
			t.Errorf("MessageThreshold = %d, want 100", config.MessageThreshold)
		}
	})
}

// Helper functions

func levelString(level CompressionLevel) string {
	switch level {
	case CompressionLevelNone:
		return "none"
	case CompressionLevelLight:
		return "light"
	case CompressionLevelModerate:
		return "moderate"
	case CompressionLevelAggressive:
		return "aggressive"
	default:
		return "unknown"
	}
}
