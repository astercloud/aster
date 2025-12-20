package middleware

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/types"
)

var sumLog = logging.ForComponent("SummarizationMiddleware")

// SummarizationMiddleware 自动总结对话历史以管理上下文窗口
// 功能:
// 1. 监控消息历史的 token 数量
// 2. 当超过阈值时,自动总结旧消息
// 3. 保留最近的 N 条消息
// 4. 用总结消息替换旧的历史记录
type SummarizationMiddleware struct {
	*BaseMiddleware

	maxTokensBeforeSummary int
	messagesToKeep         int
	summaryPrefix          string
	tokenCounter           TokenCounterFunc
	summarizer             SummarizerFunc
	summarizationCount     int // 统计总结触发次数
}

// TokenCounterFunc 自定义 token 计数函数类型
type TokenCounterFunc func(messages []types.Message) int

// SummarizerFunc 总结生成函数类型
// 接收要总结的消息列表,返回总结内容字符串
type SummarizerFunc func(ctx context.Context, messages []types.Message) (string, error)

// SummarizationMiddlewareConfig 配置
type SummarizationMiddlewareConfig struct {
	Summarizer             SummarizerFunc   // 用于生成总结的函数
	MaxTokensBeforeSummary int              // 触发总结的 token 阈值
	MessagesToKeep         int              // 总结后保留的最近消息数量
	SummaryPrefix          string           // 总结消息的前缀
	TokenCounter           TokenCounterFunc // 自定义 token 计数器
}

// NewSummarizationMiddleware 创建中间件
func NewSummarizationMiddleware(config *SummarizationMiddlewareConfig) (*SummarizationMiddleware, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}

	if config.MaxTokensBeforeSummary <= 0 {
		config.MaxTokensBeforeSummary = 170000
	}

	if config.MessagesToKeep <= 0 {
		config.MessagesToKeep = 6
	}

	if config.SummaryPrefix == "" {
		config.SummaryPrefix = "## Previous conversation summary:"
	}

	if config.TokenCounter == nil {
		config.TokenCounter = defaultTokenCounter
	}

	if config.Summarizer == nil {
		config.Summarizer = defaultSummarizer
	}

	m := &SummarizationMiddleware{
		BaseMiddleware:         NewBaseMiddleware("summarization", 40),
		maxTokensBeforeSummary: config.MaxTokensBeforeSummary,
		messagesToKeep:         config.MessagesToKeep,
		summaryPrefix:          config.SummaryPrefix,
		tokenCounter:           config.TokenCounter,
		summarizer:             config.Summarizer,
		summarizationCount:     0,
	}

	sumLog.Info(context.Background(), "initialized", map[string]any{"max_tokens": config.MaxTokensBeforeSummary, "keep_messages": config.MessagesToKeep})
	return m, nil
}

// WrapModelCall 包装模型调用,在调用前检查是否需要总结
func (m *SummarizationMiddleware) WrapModelCall(ctx context.Context, req *ModelRequest, handler ModelCallHandler) (*ModelResponse, error) {
	messages := req.Messages
	if len(messages) == 0 {
		return handler(ctx, req)
	}

	// 计算当前消息的 token 数
	totalTokens := m.tokenCounter(messages)

	sumLog.Debug(ctx, "current tokens", map[string]any{"tokens": totalTokens, "threshold": m.maxTokensBeforeSummary})

	// 如果未超过阈值,直接返回
	if totalTokens <= m.maxTokensBeforeSummary {
		return handler(ctx, req)
	}

	sumLog.Info(ctx, "token threshold exceeded, triggering summarization", nil)

	// 分离 system messages 和其他消息
	var systemMessages []types.Message
	var regularMessages []types.Message

	for _, msg := range messages {
		if msg.Role == types.MessageRoleSystem {
			systemMessages = append(systemMessages, msg)
		} else {
			regularMessages = append(regularMessages, msg)
		}
	}

	// 如果常规消息少于或等于要保留的数量,不进行总结
	if len(regularMessages) <= m.messagesToKeep {
		sumLog.Debug(ctx, "not enough messages to summarize", map[string]any{"have": len(regularMessages), "keep": m.messagesToKeep})
		return handler(ctx, req)
	}

	// 计算要总结的消息
	numToSummarize := len(regularMessages) - m.messagesToKeep
	messagesToSummarize := regularMessages[:numToSummarize]
	messagesToKeep := regularMessages[numToSummarize:]

	sumLog.Info(ctx, "summarizing messages", map[string]any{"to_summarize": numToSummarize, "keeping": m.messagesToKeep})

	// 生成总结
	summary, err := m.summarizer(ctx, messagesToSummarize)
	if err != nil {
		sumLog.Error(ctx, "failed to generate summary, keeping original", map[string]any{"error": err.Error()})
		return handler(ctx, req) // 失败时保留原始消息
	}

	sumLog.Info(ctx, "summary generated", map[string]any{"chars": len(summary)})

	// 构建新的消息列表: system messages + 总结消息 + 保留的最近消息
	newMessages := make([]types.Message, 0, len(systemMessages)+1+len(messagesToKeep))
	newMessages = append(newMessages, systemMessages...)
	newMessages = append(newMessages, types.Message{
		Role: types.MessageRoleSystem,
		ContentBlocks: []types.ContentBlock{
			&types.TextBlock{
				Text: fmt.Sprintf("%s\n\n%s", m.summaryPrefix, summary),
			},
		},
	})
	newMessages = append(newMessages, messagesToKeep...)

	// 计算压缩后的 token 数和压缩比
	newTokens := m.tokenCounter(newMessages)
	tokensSaved := totalTokens - newTokens
	compressionRatio := float64(newTokens) / float64(totalTokens)

	// 更新请求的消息
	req.Messages = newMessages
	m.summarizationCount++

	sumLog.Info(ctx, "summarization complete", map[string]any{
		"before":               len(messages),
		"after":                len(newMessages),
		"tokens_before":        totalTokens,
		"tokens_after":         newTokens,
		"tokens_saved":         tokensSaved,
		"compression_ratio":    compressionRatio,
		"total_summarizations": m.summarizationCount,
	})

	// 发送会话压缩事件 (通过 Metadata 中的 EventEmitter)
	req.EmitEvent(&types.ProgressSessionSummarizedEvent{
		MessagesBefore:   len(messages),
		MessagesAfter:    len(newMessages),
		TokensBefore:     totalTokens,
		TokensAfter:      newTokens,
		TokensSaved:      tokensSaved,
		CompressionRatio: compressionRatio,
		SummaryPreview:   truncateString(summary, 150),
	})

	return handler(ctx, req)
}

// defaultSummarizer 默认的总结生成器
func defaultSummarizer(ctx context.Context, messages []types.Message) (string, error) {
	var summary strings.Builder

	summary.WriteString("This session is being continued from a previous conversation that ran out of context. The conversation is summarized below:\n\n")

	// Analysis section - 按时间顺序分析对话
	summary.WriteString("Analysis:\n")
	summary.WriteString("Let me chronologically analyze the conversation:\n\n")

	// 提取对话的关键阶段
	phases := extractConversationPhases(messages)
	for i, phase := range phases {
		summary.WriteString(fmt.Sprintf("%d. **%s**:\n", i+1, phase.Title))
		for _, point := range phase.Points {
			summary.WriteString(fmt.Sprintf("   - %s\n", point))
		}
		summary.WriteString("\n")
	}

	// Summary section - 结构化的摘要信息
	summary.WriteString("Summary:\n")

	// 1. Primary Request and Intent
	summary.WriteString("1. Primary Request and Intent:\n")
	intent := extractUserIntent(messages)
	summary.WriteString(fmt.Sprintf("   - %s\n\n", intent))

	// 2. Key Technical Concepts
	summary.WriteString("2. Key Technical Concepts:\n")
	concepts := extractTechnicalConcepts(messages)
	for _, concept := range concepts {
		summary.WriteString(fmt.Sprintf("   - %s\n", concept))
	}
	summary.WriteString("\n")

	// 3. Files and Code Sections
	summary.WriteString("3. Files and Code Sections:\n")
	files := extractFileReferences(messages)
	if len(files) > 0 {
		for _, file := range files {
			summary.WriteString(fmt.Sprintf("   - `%s`\n", file))
		}
	} else {
		summary.WriteString("   - No specific files referenced\n")
	}
	summary.WriteString("\n")

	// 4. Problem Solving Progress
	summary.WriteString("4. Problem Solving Progress:\n")
	progress := extractProblemSolvingProgress(messages)
	for _, item := range progress {
		summary.WriteString(fmt.Sprintf("   - %s\n", item))
	}
	summary.WriteString("\n")

	// 5. All user messages (最近几条)
	summary.WriteString("5. Recent User Messages:\n")
	userMessages := extractUserMessages(messages, 5)
	for _, msg := range userMessages {
		if len(msg) > 200 {
			msg = msg[:200] + "..."
		}
		summary.WriteString(fmt.Sprintf("   - \"%s\"\n", msg))
	}
	summary.WriteString("\n")

	// 6. Pending Tasks
	summary.WriteString("6. Pending Tasks:\n")
	tasks := extractPendingTasks(messages)
	if len(tasks) > 0 {
		for _, task := range tasks {
			summary.WriteString(fmt.Sprintf("   - %s\n", task))
		}
	} else {
		summary.WriteString("   - No explicit pending tasks identified\n")
	}
	summary.WriteString("\n")

	// 7. Current Work
	summary.WriteString("7. Current Work:\n")
	currentWork := extractCurrentWork(messages)
	summary.WriteString(fmt.Sprintf("   %s\n\n", currentWork))

	summary.WriteString("Please continue the conversation from where we left it off without asking the user any further questions. Continue with the last task that you were asked to work on.")

	return summary.String(), nil
}

// ConversationPhase 对话阶段
type ConversationPhase struct {
	Title  string
	Points []string
}

// extractConversationPhases 提取对话的关键阶段
func extractConversationPhases(messages []types.Message) []ConversationPhase {
	phases := []ConversationPhase{}

	// 分析消息并提取关键阶段
	var currentPhase *ConversationPhase
	messageCount := 0

	for _, msg := range messages {
		content := extractMessageContent(msg)
		if content == "" {
			continue
		}

		messageCount++

		// 每 5-10 条消息大约一个阶段
		if currentPhase == nil || len(currentPhase.Points) >= 3 {
			title := fmt.Sprintf("Phase %d", len(phases)+1)

			// 尝试从用户消息中提取阶段主题
			if msg.Role == types.MessageRoleUser {
				if len(content) > 50 {
					title = content[:50] + "..."
				} else {
					title = content
				}
			}

			currentPhase = &ConversationPhase{
				Title:  title,
				Points: []string{},
			}
			phases = append(phases, *currentPhase)
		}

		// 提取关键点
		point := ""
		if msg.Role == types.MessageRoleUser {
			point = "User: " + truncateString(content, 100)
		} else if msg.Role == types.MessageRoleAssistant {
			// 检查是否有工具调用
			hasToolUse := false
			for _, block := range msg.ContentBlocks {
				if _, ok := block.(*types.ToolUseBlock); ok {
					hasToolUse = true
					break
				}
			}
			if hasToolUse {
				point = "Assistant executed tools"
			} else {
				point = "Assistant: " + truncateString(content, 80)
			}
		}

		if point != "" && len(phases) > 0 {
			phases[len(phases)-1].Points = append(phases[len(phases)-1].Points, point)
		}
	}

	// 限制阶段数量
	if len(phases) > 5 {
		phases = phases[len(phases)-5:]
	}

	return phases
}

// extractUserIntent 提取用户意图
func extractUserIntent(messages []types.Message) string {
	// 找到最早的用户消息作为主要意图
	for _, msg := range messages {
		if msg.Role == types.MessageRoleUser {
			content := extractMessageContent(msg)
			if content != "" {
				if len(content) > 300 {
					return content[:300] + "..."
				}
				return content
			}
		}
	}
	return "No clear intent identified"
}

// extractTechnicalConcepts 提取技术概念
func extractTechnicalConcepts(messages []types.Message) []string {
	concepts := make(map[string]bool)

	// 技术关键词模式
	techKeywords := []string{
		"API", "SDK", "框架", "framework", "压缩", "compression",
		"Token", "Context", "Agent", "middleware", "中间件",
		"函数", "function", "方法", "method", "类", "class",
		"模块", "module", "包", "package", "配置", "config",
	}

	for _, msg := range messages {
		content := extractMessageContent(msg)
		for _, keyword := range techKeywords {
			if strings.Contains(strings.ToLower(content), strings.ToLower(keyword)) {
				concepts[keyword] = true
			}
		}
	}

	result := make([]string, 0, len(concepts))
	for concept := range concepts {
		result = append(result, concept)
	}

	// 限制数量
	if len(result) > 10 {
		result = result[:10]
	}

	return result
}

// extractFileReferences 提取文件引用
func extractFileReferences(messages []types.Message) []string {
	files := make(map[string]bool)

	// 文件路径模式
	for _, msg := range messages {
		content := extractMessageContent(msg)

		// 简单的文件路径检测
		words := strings.FieldsSeq(content)
		for word := range words {
			// 检测常见文件扩展名
			if strings.HasSuffix(word, ".go") ||
				strings.HasSuffix(word, ".ts") ||
				strings.HasSuffix(word, ".js") ||
				strings.HasSuffix(word, ".vue") ||
				strings.HasSuffix(word, ".json") ||
				strings.HasSuffix(word, ".yaml") ||
				strings.HasSuffix(word, ".md") ||
				strings.Contains(word, "/pkg/") ||
				strings.Contains(word, "/src/") {
				// 清理路径
				clean := strings.Trim(word, "`\"'(),:")
				if len(clean) > 5 && len(clean) < 200 {
					files[clean] = true
				}
			}
		}
	}

	result := make([]string, 0, len(files))
	for file := range files {
		result = append(result, file)
	}

	// 限制数量
	if len(result) > 15 {
		result = result[:15]
	}

	return result
}

// extractProblemSolvingProgress 提取问题解决进度
func extractProblemSolvingProgress(messages []types.Message) []string {
	progress := []string{}

	// 检查工具调用
	toolCount := 0
	toolNames := make(map[string]int)

	for _, msg := range messages {
		if msg.Role == types.MessageRoleAssistant {
			for _, block := range msg.ContentBlocks {
				if tu, ok := block.(*types.ToolUseBlock); ok {
					toolCount++
					toolNames[tu.Name]++
				}
			}
		}
	}

	if toolCount > 0 {
		progress = append(progress, fmt.Sprintf("Executed %d tool calls", toolCount))
		for name, count := range toolNames {
			if count > 1 {
				progress = append(progress, fmt.Sprintf("Used '%s' %d times", name, count))
			}
		}
	}

	// 检查是否有错误处理
	for _, msg := range messages {
		content := strings.ToLower(extractMessageContent(msg))
		if strings.Contains(content, "error") || strings.Contains(content, "错误") {
			progress = append(progress, "Encountered and addressed errors")
			break
		}
	}

	// 检查是否有成功标记
	for _, msg := range messages {
		content := strings.ToLower(extractMessageContent(msg))
		if strings.Contains(content, "success") || strings.Contains(content, "成功") ||
			strings.Contains(content, "完成") || strings.Contains(content, "done") {
			progress = append(progress, "Achieved some milestones")
			break
		}
	}

	if len(progress) == 0 {
		progress = append(progress, "In progress")
	}

	return progress
}

// extractUserMessages 提取最近的用户消息
func extractUserMessages(messages []types.Message, limit int) []string {
	userMsgs := []string{}

	for i := len(messages) - 1; i >= 0 && len(userMsgs) < limit; i-- {
		msg := messages[i]
		if msg.Role == types.MessageRoleUser {
			content := extractMessageContent(msg)
			if content != "" && !strings.HasPrefix(content, "[Tool") {
				userMsgs = append([]string{content}, userMsgs...)
			}
		}
	}

	return userMsgs
}

// extractPendingTasks 提取待办任务
func extractPendingTasks(messages []types.Message) []string {
	tasks := []string{}

	// 从最近的消息中查找 TODO、待办等
	for i := len(messages) - 1; i >= 0 && i >= len(messages)-10; i-- {
		content := extractMessageContent(messages[i])

		// 检查 TODO 标记
		if strings.Contains(content, "TODO") || strings.Contains(content, "待办") ||
			strings.Contains(content, "需要") || strings.Contains(content, "接下来") {
			lines := strings.SplitSeq(content, "\n")
			for line := range lines {
				if strings.Contains(line, "TODO") || strings.Contains(line, "- [ ]") {
					tasks = append(tasks, strings.TrimSpace(line))
				}
			}
		}
	}

	// 限制数量
	if len(tasks) > 5 {
		tasks = tasks[:5]
	}

	return tasks
}

// extractCurrentWork 提取当前工作状态
func extractCurrentWork(messages []types.Message) string {
	// 从最后几条消息中提取当前工作状态
	if len(messages) == 0 {
		return "No recent activity"
	}

	// 查找最后的助手消息
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role == types.MessageRoleAssistant {
			content := extractMessageContent(msg)
			if content != "" {
				if len(content) > 200 {
					return content[:200] + "..."
				}
				return content
			}
		}
	}

	return "Working on the user's request"
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// defaultTokenCounter 默认的 token 计数器(基于字符数估算)
// 粗略估算: 4 个字符约等于 1 个 token
func defaultTokenCounter(messages []types.Message) int {
	totalChars := 0
	for _, msg := range messages {
		// 计算 role 的字符数
		totalChars += len(string(msg.Role))

		// 计算内容块的字符数
		for _, block := range msg.ContentBlocks {
			switch b := block.(type) {
			case *types.TextBlock:
				totalChars += len(b.Text)
			case *types.ToolUseBlock:
				totalChars += len(b.Name)
				// 估算 input 的大小
				totalChars += len(fmt.Sprintf("%v", b.Input))
			case *types.ToolResultBlock:
				totalChars += len(b.Content)
			}
		}
	}
	// 4 字符 ≈ 1 token
	return totalChars / 4
}

// extractMessageContent 提取消息的文本内容
func extractMessageContent(msg types.Message) string {
	var parts []string
	for _, block := range msg.ContentBlocks {
		switch b := block.(type) {
		case *types.TextBlock:
			parts = append(parts, b.Text)
		case *types.ToolUseBlock:
			parts = append(parts, fmt.Sprintf("[Tool: %s]", b.Name))
		case *types.ToolResultBlock:
			parts = append(parts, fmt.Sprintf("[ToolResult: %v]", b.Content))
		}
	}
	return strings.Join(parts, " ")
}

// GetSummarizationCount 获取总结触发次数
func (m *SummarizationMiddleware) GetSummarizationCount() int {
	return m.summarizationCount
}

// ResetSummarizationCount 重置计数器
func (m *SummarizationMiddleware) ResetSummarizationCount() {
	m.summarizationCount = 0
	sumLog.Debug(context.Background(), "summarization count reset", nil)
}

// GetConfig 获取当前配置
func (m *SummarizationMiddleware) GetConfig() map[string]any {
	return map[string]any{
		"max_tokens_before_summary": m.maxTokensBeforeSummary,
		"messages_to_keep":          m.messagesToKeep,
		"summary_prefix":            m.summaryPrefix,
		"summarization_count":       m.summarizationCount,
	}
}

// UpdateConfig 动态更新配置
func (m *SummarizationMiddleware) UpdateConfig(maxTokens, messagesToKeep int) {
	if maxTokens > 0 {
		m.maxTokensBeforeSummary = maxTokens
	}
	if messagesToKeep > 0 {
		m.messagesToKeep = messagesToKeep
	}
	sumLog.Info(context.Background(), "config updated", map[string]any{"max_tokens": m.maxTokensBeforeSummary, "keep_messages": m.messagesToKeep})
}
