package auto

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Capturer 自动记忆捕获器
type Capturer struct {
	mu sync.RWMutex

	// store 存储
	store Store

	// config 配置
	config *CapturerConfig

	// tagExtractors 标签提取器
	tagExtractors []TagExtractor
}

// CapturerConfig 捕获器配置
type CapturerConfig struct {
	// MinConfidence 最低置信度阈值
	MinConfidence float64

	// MaxMemoriesPerProject 每个项目最大记忆数
	MaxMemoriesPerProject int

	// AutoTagging 是否自动添加标签
	AutoTagging bool

	// DeduplicateWindow 去重时间窗口
	DeduplicateWindow time.Duration
}

// DefaultCapturerConfig 默认配置
func DefaultCapturerConfig() *CapturerConfig {
	return &CapturerConfig{
		MinConfidence:         0.5,
		MaxMemoriesPerProject: 1000,
		AutoTagging:           true,
		DeduplicateWindow:     time.Hour,
	}
}

// NewCapturer 创建捕获器
func NewCapturer(store Store, config *CapturerConfig) *Capturer {
	if config == nil {
		config = DefaultCapturerConfig()
	}
	return &Capturer{
		store:         store,
		config:        config,
		tagExtractors: []TagExtractor{NewDefaultTagExtractor()},
	}
}

// RegisterTagExtractor 注册标签提取器
func (c *Capturer) RegisterTagExtractor(extractor TagExtractor) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tagExtractors = append(c.tagExtractors, extractor)
}

// CaptureFromDialog 从对话中捕获记忆
func (c *Capturer) CaptureFromDialog(ctx context.Context, projectID, sessionID, role, message string) (*Memory, error) {
	// 检测是否值得记录
	if !c.shouldCapture(message) {
		return nil, nil
	}

	// 生成标题和内容
	title, content := c.extractTitleAndContent(message, role)
	if title == "" {
		return nil, nil
	}

	// 确定作用域
	scope := ScopeSession
	if projectID != "" {
		scope = ScopeProject
	}

	// 创建记忆
	memory := NewMemory(scope, title, content, []string{})
	memory.ProjectID = projectID
	memory.SessionID = sessionID
	memory.Source = SourceDialog
	memory.Confidence = c.calculateConfidence(message)

	// 自动提取标签
	if c.config.AutoTagging {
		tags := c.extractTags(message, role)
		for _, tag := range tags {
			memory.AddTag(tag)
		}
	}

	// 添加角色标签
	if role == "assistant" {
		memory.AddTag("ai_response")
	} else {
		memory.AddTag("user_input")
	}

	// 保存
	if err := c.store.Save(ctx, memory); err != nil {
		return nil, err
	}

	return memory, nil
}

// CaptureFromEvent 从事件中捕获记忆
func (c *Capturer) CaptureFromEvent(ctx context.Context, event *CaptureEvent) (*Memory, error) {
	// 根据事件类型生成记忆
	title, content := c.eventToMemory(event)
	if title == "" {
		return nil, nil
	}

	// 确定作用域
	scope := ScopeGlobal
	if event.ProjectID != "" {
		scope = ScopeProject
	} else if event.SessionID != "" {
		scope = ScopeSession
	}

	// 创建记忆
	memory := NewMemory(scope, title, content, []string{})
	memory.ProjectID = event.ProjectID
	memory.SessionID = event.SessionID
	memory.Source = SourceSystem

	// 添加事件类型标签
	memory.AddTag(string(event.Type))

	// 从事件数据提取额外标签
	if tags, ok := event.Data["tags"].([]string); ok {
		for _, tag := range tags {
			memory.AddTag(tag)
		}
	}

	// 保存
	if err := c.store.Save(ctx, memory); err != nil {
		return nil, err
	}

	return memory, nil
}

// shouldCapture 判断是否应该捕获
func (c *Capturer) shouldCapture(message string) bool {
	// 太短的消息不捕获
	if len(message) < 20 {
		return false
	}

	// 检查是否包含重要关键词
	importantPatterns := []string{
		"完成", "实现", "解决", "修复",
		"决定", "选择", "确认",
		"问题", "错误", "bug",
		"功能", "特性", "feature",
		"completed", "implemented", "fixed", "resolved",
	}

	messageLower := strings.ToLower(message)
	for _, pattern := range importantPatterns {
		if strings.Contains(messageLower, pattern) {
			return true
		}
	}

	return false
}

// extractTitleAndContent 提取标题和内容
func (c *Capturer) extractTitleAndContent(message, role string) (string, string) {
	// 尝试提取第一句作为标题
	sentences := splitSentences(message)
	if len(sentences) == 0 {
		return "", ""
	}

	title := sentences[0]
	if len(title) > 100 {
		title = title[:100] + "..."
	}

	return title, message
}

// extractTags 提取标签
func (c *Capturer) extractTags(message, role string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var tags []string
	seen := make(map[string]bool)

	for _, extractor := range c.tagExtractors {
		suggestions := extractor.Extract(message)
		for _, s := range suggestions {
			if s.Confidence >= c.config.MinConfidence && !seen[s.Tag] {
				tags = append(tags, s.Tag)
				seen[s.Tag] = true
			}
		}
	}

	return tags
}

// calculateConfidence 计算置信度
func (c *Capturer) calculateConfidence(message string) float64 {
	confidence := 0.5

	// 长消息通常更重要
	if len(message) > 200 {
		confidence += 0.1
	}
	if len(message) > 500 {
		confidence += 0.1
	}

	// 包含关键词加分
	importantWords := []string{"完成", "实现", "解决", "成功", "重要"}
	for _, word := range importantWords {
		if strings.Contains(message, word) {
			confidence += 0.1
			break
		}
	}

	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// eventToMemory 事件转记忆
func (c *Capturer) eventToMemory(event *CaptureEvent) (string, string) {
	var title, content string

	switch event.Type {
	case EventTaskCompleted:
		if task, ok := event.Data["task"].(string); ok {
			title = task + " 完成"
			content = task
		}
	case EventFeatureImplemented:
		if feature, ok := event.Data["feature"].(string); ok {
			title = feature + " 实现完成"
			content = feature
		}
	case EventDecisionMade:
		if decision, ok := event.Data["decision"].(string); ok {
			title = "决策: " + decision
			content = decision
		}
	case EventMilestoneReached:
		if milestone, ok := event.Data["milestone"].(string); ok {
			title = "里程碑: " + milestone
			content = milestone
		}
	}

	// 添加描述
	if desc, ok := event.Data["description"].(string); ok {
		content = content + "\n\n" + desc
	}

	return title, content
}

// splitSentences 分句
func splitSentences(text string) []string {
	// 使用中英文句号分句
	re := regexp.MustCompile(`[。！？.!?]+`)
	sentences := re.Split(text, -1)

	var result []string
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

// TagExtractor 标签提取器接口
type TagExtractor interface {
	Extract(text string) []TagSuggestion
}

// DefaultTagExtractor 默认标签提取器
type DefaultTagExtractor struct {
	// 关键词 -> 标签映射
	keywordTags map[string]string
}

// NewDefaultTagExtractor 创建默认标签提取器
func NewDefaultTagExtractor() *DefaultTagExtractor {
	return &DefaultTagExtractor{
		keywordTags: map[string]string{
			// 技术相关
			"实现":        "implementation",
			"implement": "implementation",
			"修复":        "bugfix",
			"fix":       "bugfix",
			"bug":       "bugfix",
			"功能":        "feature",
			"feature":   "feature",
			"优化":        "optimization",
			"optimize":  "optimization",
			"重构":        "refactor",
			"refactor":  "refactor",
			"测试":        "testing",
			"test":      "testing",
			"文档":        "documentation",
			"doc":       "documentation",

			// 状态相关
			"完成":       "completed",
			"complete": "completed",
			"done":     "completed",
			"进行中":      "in_progress",
			"待办":       "todo",
			"todo":     "todo",

			// 领域相关
			"前端":       "frontend",
			"frontend": "frontend",
			"后端":       "backend",
			"backend":  "backend",
			"api":      "api",
			"数据库":      "database",
			"database": "database",
		},
	}
}

// Extract 提取标签
func (e *DefaultTagExtractor) Extract(text string) []TagSuggestion {
	var suggestions []TagSuggestion
	textLower := strings.ToLower(text)

	for keyword, tag := range e.keywordTags {
		if strings.Contains(textLower, strings.ToLower(keyword)) {
			suggestions = append(suggestions, TagSuggestion{
				Tag:        tag,
				Confidence: 0.7,
				Reason:     "keyword match: " + keyword,
			})
		}
	}

	return suggestions
}

// Store 记忆存储接口
type Store interface {
	Save(ctx context.Context, memory *Memory) error
	Load(ctx context.Context, id string) (*Memory, error)
	List(ctx context.Context, scope MemoryScope, projectID string, limit int) ([]*Memory, error)
	Search(ctx context.Context, query string, tags []string, limit int) ([]*Memory, error)
	Delete(ctx context.Context, id string) error
}
