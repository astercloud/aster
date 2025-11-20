package memory

import (
	"context"
	"testing"

	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/types"
)

// MockSummaryProvider 模拟 Provider
type MockSummaryProvider struct {
	response string
}

func (m *MockSummaryProvider) Complete(ctx context.Context, messages []types.Message, opts *provider.StreamOptions) (*provider.CompleteResponse, error) {
	return &provider.CompleteResponse{
		Message: types.Message{
			Role:    "assistant",
			Content: m.response,
		},
	}, nil
}

func (m *MockSummaryProvider) Stream(ctx context.Context, messages []types.Message, opts *provider.StreamOptions) (<-chan provider.StreamChunk, error) {
	ch := make(chan provider.StreamChunk, 1)
	close(ch)
	return ch, nil
}

func (m *MockSummaryProvider) Config() *types.ModelConfig {
	return &types.ModelConfig{
		Provider: "mock",
		Model:    "test",
	}
}

func (m *MockSummaryProvider) Capabilities() provider.ProviderCapabilities {
	return provider.ProviderCapabilities{}
}

func (m *MockSummaryProvider) SetSystemPrompt(prompt string) error {
	return nil
}

func (m *MockSummaryProvider) GetSystemPrompt() string {
	return ""
}

func (m *MockSummaryProvider) Close() error {
	return nil
}

func TestSessionSummaryManager_GenerateSummary(t *testing.T) {
	mockProvider := &MockSummaryProvider{
		response: `{
			"summary": "讨论了项目进度和下一步计划",
			"topics": ["项目进度", "计划"],
			"key_points": ["完成了功能A", "需要开始功能B"],
			"decisions": ["下周开始功能B的开发"],
			"action_items": ["准备功能B的设计文档"]
		}`,
	}

	config := SessionSummaryConfig{
		Enabled:            true,
		IncludeTopics:      true,
		IncludeKeyPoints:   true,
		IncludeDecisions:   true,
		IncludeActionItems: true,
	}

	manager := NewSessionSummaryManager(mockProvider, config)

	ctx := context.Background()
	sessionID := "test-session"
	messages := []types.Message{
		{Role: "user", Content: "项目进度如何？"},
		{Role: "assistant", Content: "功能A已经完成了。"},
		{Role: "user", Content: "下一步做什么？"},
		{Role: "assistant", Content: "建议下周开始功能B的开发。"},
	}

	summary, err := manager.GenerateSummary(ctx, sessionID, messages)
	if err != nil {
		t.Fatalf("GenerateSummary failed: %v", err)
	}

	if summary.SessionID != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, summary.SessionID)
	}

	if summary.Summary == "" {
		t.Error("Expected non-empty summary")
	}

	if len(summary.Topics) == 0 {
		t.Error("Expected topics to be extracted")
	}

	if len(summary.KeyPoints) == 0 {
		t.Error("Expected key points to be extracted")
	}

	if len(summary.Decisions) == 0 {
		t.Error("Expected decisions to be extracted")
	}

	if len(summary.ActionItems) == 0 {
		t.Error("Expected action items to be extracted")
	}

	if summary.MessageCount != len(messages) {
		t.Errorf("Expected message count %d, got %d", len(messages), summary.MessageCount)
	}
}

func TestSessionSummaryManager_GetSummary(t *testing.T) {
	mockProvider := &MockSummaryProvider{
		response: `{"summary": "test", "topics": [], "key_points": [], "decisions": [], "action_items": []}`,
	}

	config := SessionSummaryConfig{
		Enabled: true,
	}

	manager := NewSessionSummaryManager(mockProvider, config)

	ctx := context.Background()
	sessionID := "test-session"
	messages := []types.Message{
		{Role: "user", Content: "Hello"},
	}

	// 生成摘要
	_, err := manager.GenerateSummary(ctx, sessionID, messages)
	if err != nil {
		t.Fatalf("GenerateSummary failed: %v", err)
	}

	// 获取摘要
	summary, exists := manager.GetSummary(sessionID)
	if !exists {
		t.Fatal("Expected summary to exist")
	}

	if summary.SessionID != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, summary.SessionID)
	}
}

func TestSessionSummaryManager_UpdateSummary(t *testing.T) {
	mockProvider := &MockSummaryProvider{
		response: `{
			"summary": "更新后的摘要",
			"topics": ["主题1", "主题2"],
			"key_points": ["要点1", "要点2"],
			"decisions": ["决策1"],
			"action_items": ["行动项1"]
		}`,
	}

	config := SessionSummaryConfig{
		Enabled: true,
	}

	manager := NewSessionSummaryManager(mockProvider, config)

	ctx := context.Background()
	sessionID := "test-session"

	// 生成初始摘要
	initialMessages := []types.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi"},
	}

	_, err := manager.GenerateSummary(ctx, sessionID, initialMessages)
	if err != nil {
		t.Fatalf("GenerateSummary failed: %v", err)
	}

	// 更新摘要
	newMessages := []types.Message{
		{Role: "user", Content: "How are you?"},
		{Role: "assistant", Content: "I'm good"},
	}

	updatedSummary, err := manager.UpdateSummary(ctx, sessionID, newMessages)
	if err != nil {
		t.Fatalf("UpdateSummary failed: %v", err)
	}

	if updatedSummary.MessageCount != len(initialMessages)+len(newMessages) {
		t.Errorf("Expected message count %d, got %d",
			len(initialMessages)+len(newMessages),
			updatedSummary.MessageCount)
	}

	if updatedSummary.Summary != "更新后的摘要" {
		t.Errorf("Expected updated summary, got: %s", updatedSummary.Summary)
	}
}

func TestSessionSummaryManager_DeleteSummary(t *testing.T) {
	mockProvider := &MockSummaryProvider{
		response: `{"summary": "test", "topics": [], "key_points": [], "decisions": [], "action_items": []}`,
	}

	config := SessionSummaryConfig{
		Enabled: true,
	}

	manager := NewSessionSummaryManager(mockProvider, config)

	ctx := context.Background()
	sessionID := "test-session"
	messages := []types.Message{
		{Role: "user", Content: "Hello"},
	}

	// 生成摘要
	_, err := manager.GenerateSummary(ctx, sessionID, messages)
	if err != nil {
		t.Fatalf("GenerateSummary failed: %v", err)
	}

	// 验证存在
	_, exists := manager.GetSummary(sessionID)
	if !exists {
		t.Fatal("Expected summary to exist")
	}

	// 删除摘要
	err = manager.DeleteSummary(sessionID)
	if err != nil {
		t.Fatalf("DeleteSummary failed: %v", err)
	}

	// 验证已删除
	_, exists = manager.GetSummary(sessionID)
	if exists {
		t.Fatal("Expected summary to be deleted")
	}
}

func TestSessionSummaryManager_ListSummaries(t *testing.T) {
	mockProvider := &MockSummaryProvider{
		response: `{"summary": "test", "topics": [], "key_points": [], "decisions": [], "action_items": []}`,
	}

	config := SessionSummaryConfig{
		Enabled: true,
	}

	manager := NewSessionSummaryManager(mockProvider, config)

	ctx := context.Background()
	messages := []types.Message{
		{Role: "user", Content: "Hello"},
	}

	// 生成多个摘要
	for i := 0; i < 3; i++ {
		sessionID := "test-session-" + string(rune('0'+i))
		_, err := manager.GenerateSummary(ctx, sessionID, messages)
		if err != nil {
			t.Fatalf("GenerateSummary failed: %v", err)
		}
	}

	// 列出所有摘要
	summaries := manager.ListSummaries()
	if len(summaries) != 3 {
		t.Errorf("Expected 3 summaries, got %d", len(summaries))
	}
}

func TestSessionSummaryManager_ShouldUpdate(t *testing.T) {
	mockProvider := &MockSummaryProvider{
		response: `{"summary": "test", "topics": [], "key_points": [], "decisions": [], "action_items": []}`,
	}

	config := SessionSummaryConfig{
		Enabled:        true,
		AutoUpdate:     true,
		UpdateInterval: 10,
	}

	manager := NewSessionSummaryManager(mockProvider, config)

	ctx := context.Background()
	sessionID := "test-session"

	// 没有摘要时，消息数量达到间隔应该更新
	if !manager.ShouldUpdate(sessionID, 10) {
		t.Error("Expected should update when message count reaches interval")
	}

	// 生成摘要
	messages := make([]types.Message, 10)
	for i := range messages {
		messages[i] = types.Message{Role: "user", Content: "test"}
	}

	_, err := manager.GenerateSummary(ctx, sessionID, messages)
	if err != nil {
		t.Fatalf("GenerateSummary failed: %v", err)
	}

	// 消息数量未达到间隔，不应该更新
	if manager.ShouldUpdate(sessionID, 15) {
		t.Error("Expected should not update when message count below interval")
	}

	// 消息数量达到间隔，应该更新
	if !manager.ShouldUpdate(sessionID, 20) {
		t.Error("Expected should update when message count reaches interval")
	}
}

func TestSessionSummaryManager_GetSummaryText(t *testing.T) {
	mockProvider := &MockSummaryProvider{
		response: `{
			"summary": "测试摘要",
			"topics": ["主题1", "主题2"],
			"key_points": ["要点1", "要点2"],
			"decisions": ["决策1"],
			"action_items": ["行动项1"]
		}`,
	}

	config := SessionSummaryConfig{
		Enabled: true,
	}

	manager := NewSessionSummaryManager(mockProvider, config)

	ctx := context.Background()
	sessionID := "test-session"
	messages := []types.Message{
		{Role: "user", Content: "Hello"},
	}

	// 生成摘要
	_, err := manager.GenerateSummary(ctx, sessionID, messages)
	if err != nil {
		t.Fatalf("GenerateSummary failed: %v", err)
	}

	// 获取摘要文本
	text := manager.GetSummaryText(sessionID)
	if text == "" {
		t.Error("Expected non-empty summary text")
	}

	// 验证包含关键内容
	if indexOf(text, "测试摘要") == -1 {
		t.Error("Expected summary text to contain summary")
	}

	if indexOf(text, "主题1") == -1 {
		t.Error("Expected summary text to contain topics")
	}

	if indexOf(text, "要点1") == -1 {
		t.Error("Expected summary text to contain key points")
	}

	if indexOf(text, "决策1") == -1 {
		t.Error("Expected summary text to contain decisions")
	}

	if indexOf(text, "行动项1") == -1 {
		t.Error("Expected summary text to contain action items")
	}
}

func TestSessionSummaryManager_Disabled(t *testing.T) {
	mockProvider := &MockSummaryProvider{
		response: `{"summary": "test", "topics": [], "key_points": [], "decisions": [], "action_items": []}`,
	}

	config := SessionSummaryConfig{
		Enabled: false,
	}

	manager := NewSessionSummaryManager(mockProvider, config)

	ctx := context.Background()
	sessionID := "test-session"
	messages := []types.Message{
		{Role: "user", Content: "Hello"},
	}

	// 尝试生成摘要（应该失败）
	_, err := manager.GenerateSummary(ctx, sessionID, messages)
	if err == nil {
		t.Fatal("Expected error when summary is disabled")
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain JSON",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON in code block",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON with text before",
			input:    "Here is the JSON:\n{\"key\": \"value\"}",
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON with text after",
			input:    "{\"key\": \"value\"}\nThat's it!",
			expected: `{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}
