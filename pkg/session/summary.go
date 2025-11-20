package session

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/types"
)

// SummarizerConfig Session 摘要器配置
type SummarizerConfig struct {
	Provider           provider.Provider
	MaxMessagesPerCall int     // 每次摘要的最大消息数
	MinMessagesToSummarize int // 触发摘要的最小消息数
	Temperature        float64 // 摘要生成温度
	SystemPrompt       string  // 自定义系统提示词
}

// Summarizer Session 摘要器
type Summarizer struct {
	provider    provider.Provider
	maxMessages int
	minMessages int
	temperature float64
	sysPrompt   string
}

// NewSummarizer 创建 Session 摘要器
func NewSummarizer(config SummarizerConfig) *Summarizer {
	if config.MaxMessagesPerCall <= 0 {
		config.MaxMessagesPerCall = 50
	}
	if config.MinMessagesToSummarize <= 0 {
		config.MinMessagesToSummarize = 10
	}
	if config.Temperature <= 0 {
		config.Temperature = 0.3 // 摘要使用较低温度
	}
	if config.SystemPrompt == "" {
		config.SystemPrompt = defaultSummarySystemPrompt
	}

	return &Summarizer{
		provider:    config.Provider,
		maxMessages: config.MaxMessagesPerCall,
		minMessages: config.MinMessagesToSummarize,
		temperature: config.Temperature,
		sysPrompt:   config.SystemPrompt,
	}
}

// SummarizeSession 生成 Session 摘要
func (s *Summarizer) SummarizeSession(ctx context.Context, messages []types.Message) (*SessionSummary, error) {
	if len(messages) < s.minMessages {
		return nil, fmt.Errorf("not enough messages to summarize (need at least %d)", s.minMessages)
	}

	// 限制消息数量
	if len(messages) > s.maxMessages {
		messages = messages[len(messages)-s.maxMessages:]
	}

	// 构建摘要提示词
	prompt := s.buildSummaryPrompt(messages)

	// 调用 Provider 生成摘要
	summaryMessages := []types.Message{
		{
			Role: types.MessageRoleUser,
			ContentBlocks: []types.ContentBlock{
				&types.TextBlock{Text: prompt},
			},
		},
	}

	// 临时设置系统提示词
	originalSysPrompt := ""
	if caps := s.provider.Capabilities(); caps.SupportSystemPrompt {
		// 保存原始系统提示词（如果需要恢复）
		s.provider.SetSystemPrompt(s.sysPrompt)
	}

	response, err := s.provider.Complete(ctx, summaryMessages, nil)
	if err != nil {
		return nil, fmt.Errorf("generate summary: %w", err)
	}

	// 恢复原始系统提示词
	if originalSysPrompt != "" {
		s.provider.SetSystemPrompt(originalSysPrompt)
	}

	// 提取文本内容
	summaryText := ""
	if len(response.Message.ContentBlocks) > 0 {
		if textBlock, ok := response.Message.ContentBlocks[0].(*types.TextBlock); ok {
			summaryText = textBlock.Text
		}
	}

	summary := &SessionSummary{
		Text:          summaryText,
		MessageCount:  len(messages),
		GeneratedAt:   time.Now(),
		TokensUsed:    int(response.Usage.TotalTokens),
		KeyTopics:     extractKeyTopics(summaryText),
		Participants:  extractParticipants(messages),
	}

	return summary, nil
}

// SummarizeIncremental 增量摘要（基于之前的摘要）
func (s *Summarizer) SummarizeIncremental(ctx context.Context, previousSummary string, newMessages []types.Message) (*SessionSummary, error) {
	if len(newMessages) == 0 {
		return nil, fmt.Errorf("no new messages to summarize")
	}

	prompt := s.buildIncrementalSummaryPrompt(previousSummary, newMessages)

	summaryMessages := []types.Message{
		{
			Role: types.MessageRoleUser,
			ContentBlocks: []types.ContentBlock{
				&types.TextBlock{Text: prompt},
			},
		},
	}

	if caps := s.provider.Capabilities(); caps.SupportSystemPrompt {
		s.provider.SetSystemPrompt(s.sysPrompt)
	}

	response, err := s.provider.Complete(ctx, summaryMessages, nil)
	if err != nil {
		return nil, fmt.Errorf("generate incremental summary: %w", err)
	}

	// 提取文本内容
	summaryText := ""
	if len(response.Message.ContentBlocks) > 0 {
		if textBlock, ok := response.Message.ContentBlocks[0].(*types.TextBlock); ok {
			summaryText = textBlock.Text
		}
	}

	summary := &SessionSummary{
		Text:          summaryText,
		MessageCount:  len(newMessages),
		GeneratedAt:   time.Now(),
		TokensUsed:    int(response.Usage.TotalTokens),
		KeyTopics:     extractKeyTopics(summaryText),
		Participants:  extractParticipants(newMessages),
		IsIncremental: true,
	}

	return summary, nil
}

// buildSummaryPrompt 构建摘要提示词
func (s *Summarizer) buildSummaryPrompt(messages []types.Message) string {
	var prompt strings.Builder

	prompt.WriteString("Please provide a concise summary of the following conversation.\n\n")
	prompt.WriteString("Focus on:\n")
	prompt.WriteString("- Main topics discussed\n")
	prompt.WriteString("- Key decisions or conclusions\n")
	prompt.WriteString("- Important action items\n")
	prompt.WriteString("- Unresolved questions\n\n")
	prompt.WriteString("## Conversation\n\n")

	for i, msg := range messages {
		role := string(msg.Role)
		content := extractTextContent(msg)
		
		prompt.WriteString(fmt.Sprintf("**%s** (Message %d):\n", role, i+1))
		prompt.WriteString(content)
		prompt.WriteString("\n\n")
	}

	prompt.WriteString("## Summary\n\n")
	prompt.WriteString("Provide a structured summary in the following format:\n\n")
	prompt.WriteString("### Main Topics\n")
	prompt.WriteString("- [Topic 1]\n")
	prompt.WriteString("- [Topic 2]\n\n")
	prompt.WriteString("### Key Points\n")
	prompt.WriteString("- [Point 1]\n")
	prompt.WriteString("- [Point 2]\n\n")
	prompt.WriteString("### Action Items\n")
	prompt.WriteString("- [Action 1]\n")
	prompt.WriteString("- [Action 2]\n\n")
	prompt.WriteString("### Open Questions\n")
	prompt.WriteString("- [Question 1]\n")

	return prompt.String()
}

// buildIncrementalSummaryPrompt 构建增量摘要提示词
func (s *Summarizer) buildIncrementalSummaryPrompt(previousSummary string, newMessages []types.Message) string {
	var prompt strings.Builder

	prompt.WriteString("Update the following conversation summary with new messages.\n\n")
	prompt.WriteString("## Previous Summary\n\n")
	prompt.WriteString(previousSummary)
	prompt.WriteString("\n\n## New Messages\n\n")

	for i, msg := range newMessages {
		role := string(msg.Role)
		content := extractTextContent(msg)
		
		prompt.WriteString(fmt.Sprintf("**%s** (Message %d):\n", role, i+1))
		prompt.WriteString(content)
		prompt.WriteString("\n\n")
	}

	prompt.WriteString("## Updated Summary\n\n")
	prompt.WriteString("Provide an updated summary that incorporates the new information.\n")

	return prompt.String()
}

// SessionSummary Session 摘要
type SessionSummary struct {
	Text          string    `json:"text"`
	MessageCount  int       `json:"message_count"`
	GeneratedAt   time.Time `json:"generated_at"`
	TokensUsed    int       `json:"tokens_used"`
	KeyTopics     []string  `json:"key_topics"`
	Participants  []string  `json:"participants"`
	IsIncremental bool      `json:"is_incremental"`
}

// extractTextContent 提取消息的文本内容
func extractTextContent(msg types.Message) string {
	var content strings.Builder
	for _, block := range msg.ContentBlocks {
		if textBlock, ok := block.(*types.TextBlock); ok {
			content.WriteString(textBlock.Text)
			content.WriteString(" ")
		}
	}
	return strings.TrimSpace(content.String())
}

// extractKeyTopics 从摘要中提取关键主题
func extractKeyTopics(summary string) []string {
	topics := []string{}
	
	// 简单的主题提取逻辑
	lines := strings.Split(summary, "\n")
	inTopicsSection := false
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if strings.Contains(strings.ToLower(line), "main topics") {
			inTopicsSection = true
			continue
		}
		
		if inTopicsSection {
			if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") {
				topic := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "-"), "*"))
				if topic != "" {
					topics = append(topics, topic)
				}
			} else if strings.HasPrefix(line, "#") {
				// 遇到新的标题，退出主题部分
				break
			}
		}
	}
	
	return topics
}

// extractParticipants 提取参与者
func extractParticipants(messages []types.Message) []string {
	participantSet := make(map[string]bool)
	
	for _, msg := range messages {
		role := string(msg.Role)
		participantSet[role] = true
	}
	
	participants := make([]string, 0, len(participantSet))
	for participant := range participantSet {
		participants = append(participants, participant)
	}
	
	return participants
}

// defaultSummarySystemPrompt 默认摘要系统提示词
const defaultSummarySystemPrompt = `You are a conversation summarizer. Your task is to create concise, accurate summaries of conversations.

Guidelines:
- Be objective and factual
- Capture the main points and key decisions
- Highlight action items and open questions
- Use clear, structured format
- Keep the summary concise but comprehensive
- Preserve important context and nuances`
