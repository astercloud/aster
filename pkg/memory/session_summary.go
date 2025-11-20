package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/types"
)

// SessionSummary 会话摘要
type SessionSummary struct {
	// 基本信息
	SessionID string    `json:"session_id"`
	Summary   string    `json:"summary"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 详细信息
	Topics     []string          `json:"topics"`      // 讨论的主题
	KeyPoints  []string          `json:"key_points"`  // 关键要点
	Decisions  []string          `json:"decisions"`   // 做出的决策
	ActionItems []string         `json:"action_items"` // 行动项
	Metadata   map[string]interface{} `json:"metadata"`

	// 统计信息
	MessageCount int `json:"message_count"` // 消息数量
	TokenCount   int `json:"token_count"`   // Token 数量
}

// SessionSummaryManager 会话摘要管理器
type SessionSummaryManager struct {
	mu sync.RWMutex

	// 摘要存储（会话ID -> SessionSummary）
	summaries map[string]*SessionSummary

	// Provider 用于生成摘要
	provider provider.Provider

	// 配置
	config SessionSummaryConfig
}

// SessionSummaryConfig 会话摘要配置
type SessionSummaryConfig struct {
	// Enabled 是否启用会话摘要
	Enabled bool

	// AutoUpdate 是否自动更新摘要
	AutoUpdate bool

	// UpdateInterval 自动更新间隔（消息数量）
	UpdateInterval int

	// MaxSummaryLength 摘要最大长度（字符数）
	MaxSummaryLength int

	// IncludeTopics 是否提取主题
	IncludeTopics bool

	// IncludeKeyPoints 是否提取关键要点
	IncludeKeyPoints bool

	// IncludeDecisions 是否提取决策
	IncludeDecisions bool

	// IncludeActionItems 是否提取行动项
	IncludeActionItems bool

	// SummaryPrompt 自定义摘要提示词
	SummaryPrompt string
}

// DefaultSessionSummaryConfig 返回默认配置
func DefaultSessionSummaryConfig() SessionSummaryConfig {
	return SessionSummaryConfig{
		Enabled:            false,
		AutoUpdate:         true,
		UpdateInterval:     10, // 每10条消息更新一次
		MaxSummaryLength:   500,
		IncludeTopics:      true,
		IncludeKeyPoints:   true,
		IncludeDecisions:   true,
		IncludeActionItems: true,
		SummaryPrompt:      defaultSummaryPrompt,
	}
}

const defaultSummaryPrompt = `请分析以下对话并生成一个结构化的摘要。

对话内容：
{{MESSAGES}}

请以 JSON 格式返回摘要，包含以下字段：
{
  "summary": "简短的对话摘要（2-3句话）",
  "topics": ["主题1", "主题2", ...],
  "key_points": ["要点1", "要点2", ...],
  "decisions": ["决策1", "决策2", ...],
  "action_items": ["行动项1", "行动项2", ...]
}

注意：
- summary 应该简洁明了，突出对话的核心内容
- topics 列出讨论的主要话题
- key_points 列出重要的信息点
- decisions 列出做出的决定或结论
- action_items 列出需要执行的任务或行动

请只返回 JSON，不要包含其他内容。`

// NewSessionSummaryManager 创建会话摘要管理器
func NewSessionSummaryManager(provider provider.Provider, config SessionSummaryConfig) *SessionSummaryManager {
	return &SessionSummaryManager{
		summaries: make(map[string]*SessionSummary),
		provider:  provider,
		config:    config,
	}
}

// GenerateSummary 生成会话摘要
func (m *SessionSummaryManager) GenerateSummary(
	ctx context.Context,
	sessionID string,
	messages []types.Message,
) (*SessionSummary, error) {
	if !m.config.Enabled {
		return nil, fmt.Errorf("session summary is disabled")
	}

	// 构建提示词
	prompt := m.buildPrompt(messages)

	// 调用 LLM 生成摘要
	resp, err := m.provider.Complete(ctx, []types.Message{
		{
			Role:    "user",
			Content: prompt,
		},
	}, &provider.StreamOptions{
		MaxTokens:   1000,
		Temperature: 0.3, // 较低的温度以获得更一致的输出
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate summary: %w", err)
	}

	// 解析响应
	summary, err := m.parseSummaryResponse(resp.Message.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse summary: %w", err)
	}

	// 设置基本信息
	summary.SessionID = sessionID
	summary.CreatedAt = time.Now()
	summary.UpdatedAt = time.Now()
	summary.MessageCount = len(messages)

	// 计算 token 数量（粗略估计）
	totalTokens := 0
	for _, msg := range messages {
		totalTokens += len(msg.Content) / 4 // 粗略估计：4个字符 ≈ 1个token
	}
	summary.TokenCount = totalTokens

	// 存储摘要
	m.mu.Lock()
	m.summaries[sessionID] = summary
	m.mu.Unlock()

	return summary, nil
}

// UpdateSummary 更新会话摘要
func (m *SessionSummaryManager) UpdateSummary(
	ctx context.Context,
	sessionID string,
	newMessages []types.Message,
) (*SessionSummary, error) {
	if !m.config.Enabled {
		return nil, fmt.Errorf("session summary is disabled")
	}

	// 获取现有摘要
	m.mu.RLock()
	existingSummary, exists := m.summaries[sessionID]
	m.mu.RUnlock()

	// 如果不存在，生成新摘要
	if !exists {
		return m.GenerateSummary(ctx, sessionID, newMessages)
	}

	// 构建增量更新提示词
	prompt := m.buildIncrementalPrompt(existingSummary, newMessages)

	// 调用 LLM 更新摘要
	resp, err := m.provider.Complete(ctx, []types.Message{
		{
			Role:    "user",
			Content: prompt,
		},
	}, &provider.StreamOptions{
		MaxTokens:   1000,
		Temperature: 0.3,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update summary: %w", err)
	}

	// 解析响应
	updatedSummary, err := m.parseSummaryResponse(resp.Message.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse updated summary: %w", err)
	}

	// 更新基本信息
	updatedSummary.SessionID = sessionID
	updatedSummary.CreatedAt = existingSummary.CreatedAt
	updatedSummary.UpdatedAt = time.Now()
	updatedSummary.MessageCount = existingSummary.MessageCount + len(newMessages)
	updatedSummary.TokenCount = existingSummary.TokenCount + m.estimateTokens(newMessages)

	// 存储更新后的摘要
	m.mu.Lock()
	m.summaries[sessionID] = updatedSummary
	m.mu.Unlock()

	return updatedSummary, nil
}

// GetSummary 获取会话摘要
func (m *SessionSummaryManager) GetSummary(sessionID string) (*SessionSummary, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	summary, exists := m.summaries[sessionID]
	return summary, exists
}

// DeleteSummary 删除会话摘要
func (m *SessionSummaryManager) DeleteSummary(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.summaries, sessionID)
	return nil
}

// ListSummaries 列出所有会话摘要
func (m *SessionSummaryManager) ListSummaries() []*SessionSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	summaries := make([]*SessionSummary, 0, len(m.summaries))
	for _, summary := range m.summaries {
		summaries = append(summaries, summary)
	}

	return summaries
}

// ShouldUpdate 判断是否应该更新摘要
func (m *SessionSummaryManager) ShouldUpdate(sessionID string, currentMessageCount int) bool {
	if !m.config.Enabled || !m.config.AutoUpdate {
		return false
	}

	m.mu.RLock()
	summary, exists := m.summaries[sessionID]
	m.mu.RUnlock()

	if !exists {
		// 如果摘要不存在，且消息数量达到更新间隔，则应该生成
		return currentMessageCount >= m.config.UpdateInterval
	}

	// 如果消息数量增加超过更新间隔，则应该更新
	return currentMessageCount-summary.MessageCount >= m.config.UpdateInterval
}

// GetSummaryText 获取摘要文本（用于添加到上下文）
func (m *SessionSummaryManager) GetSummaryText(sessionID string) string {
	summary, exists := m.GetSummary(sessionID)
	if !exists {
		return ""
	}

	text := fmt.Sprintf("会话摘要：\n%s\n", summary.Summary)

	if len(summary.Topics) > 0 {
		text += fmt.Sprintf("\n讨论主题：\n")
		for _, topic := range summary.Topics {
			text += fmt.Sprintf("- %s\n", topic)
		}
	}

	if len(summary.KeyPoints) > 0 {
		text += fmt.Sprintf("\n关键要点：\n")
		for _, point := range summary.KeyPoints {
			text += fmt.Sprintf("- %s\n", point)
		}
	}

	if len(summary.Decisions) > 0 {
		text += fmt.Sprintf("\n决策：\n")
		for _, decision := range summary.Decisions {
			text += fmt.Sprintf("- %s\n", decision)
		}
	}

	if len(summary.ActionItems) > 0 {
		text += fmt.Sprintf("\n行动项：\n")
		for _, item := range summary.ActionItems {
			text += fmt.Sprintf("- %s\n", item)
		}
	}

	return text
}

// 辅助方法

func (m *SessionSummaryManager) buildPrompt(messages []types.Message) string {
	// 格式化消息
	messagesText := ""
	for i, msg := range messages {
		messagesText += fmt.Sprintf("[%d] %s: %s\n", i+1, msg.Role, msg.Content)
	}

	// 替换占位符
	prompt := m.config.SummaryPrompt
	prompt = replaceString(prompt, "{{MESSAGES}}", messagesText)

	return prompt
}

func (m *SessionSummaryManager) buildIncrementalPrompt(existingSummary *SessionSummary, newMessages []types.Message) string {
	// 格式化现有摘要
	existingSummaryJSON, _ := json.MarshalIndent(map[string]interface{}{
		"summary":      existingSummary.Summary,
		"topics":       existingSummary.Topics,
		"key_points":   existingSummary.KeyPoints,
		"decisions":    existingSummary.Decisions,
		"action_items": existingSummary.ActionItems,
	}, "", "  ")

	// 格式化新消息
	newMessagesText := ""
	for i, msg := range newMessages {
		newMessagesText += fmt.Sprintf("[%d] %s: %s\n", i+1, msg.Role, msg.Content)
	}

	prompt := fmt.Sprintf(`现有的会话摘要：
%s

新增的对话内容：
%s

请更新会话摘要，整合新的对话内容。保持相同的 JSON 格式。

注意：
- 如果新对话引入了新主题，添加到 topics
- 如果有新的关键信息，添加到 key_points
- 如果有新的决策，添加到 decisions
- 如果有新的行动项，添加到 action_items
- 更新 summary 以反映最新的对话内容

请只返回 JSON，不要包含其他内容。`, existingSummaryJSON, newMessagesText)

	return prompt
}

func (m *SessionSummaryManager) parseSummaryResponse(content string) (*SessionSummary, error) {
	// 尝试提取 JSON（可能包含在代码块中）
	jsonContent := extractJSON(content)

	// 解析 JSON
	var data struct {
		Summary     string   `json:"summary"`
		Topics      []string `json:"topics"`
		KeyPoints   []string `json:"key_points"`
		Decisions   []string `json:"decisions"`
		ActionItems []string `json:"action_items"`
	}

	if err := json.Unmarshal([]byte(jsonContent), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// 创建摘要对象
	summary := &SessionSummary{
		Summary:     data.Summary,
		Topics:      data.Topics,
		KeyPoints:   data.KeyPoints,
		Decisions:   data.Decisions,
		ActionItems: data.ActionItems,
		Metadata:    make(map[string]interface{}),
	}

	// 确保切片不为 nil
	if summary.Topics == nil {
		summary.Topics = []string{}
	}
	if summary.KeyPoints == nil {
		summary.KeyPoints = []string{}
	}
	if summary.Decisions == nil {
		summary.Decisions = []string{}
	}
	if summary.ActionItems == nil {
		summary.ActionItems = []string{}
	}

	return summary, nil
}

func (m *SessionSummaryManager) estimateTokens(messages []types.Message) int {
	totalTokens := 0
	for _, msg := range messages {
		totalTokens += len(msg.Content) / 4 // 粗略估计
	}
	return totalTokens
}

// 辅助函数

func replaceString(s, old, new string) string {
	// 简单的字符串替换
	result := ""
	for {
		idx := indexOf(s, old)
		if idx == -1 {
			result += s
			break
		}
		result += s[:idx] + new
		s = s[idx+len(old):]
	}
	return result
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func extractJSON(content string) string {
	// 尝试提取 JSON（可能在代码块中）
	start := indexOf(content, "{")
	end := lastIndexOf(content, "}")

	if start != -1 && end != -1 && end > start {
		return content[start : end+1]
	}

	return content
}

func lastIndexOf(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
