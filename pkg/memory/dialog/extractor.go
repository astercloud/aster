// Package dialog 提供对话偏好提取功能
// 从用户对话中自动识别偏好、约束、风格等信息
package dialog

import (
	"strings"
	"time"
)

// PreferenceCategory 偏好分类
type PreferenceCategory string

const (
	// CategoryPreference 一般偏好
	CategoryPreference PreferenceCategory = "preference"
	// CategoryConstraint 约束条件
	CategoryConstraint PreferenceCategory = "constraint"
	// CategoryStyle 风格偏好
	CategoryStyle PreferenceCategory = "style"
	// CategoryAudience 受众偏好
	CategoryAudience PreferenceCategory = "audience"
	// CategoryLength 长度偏好
	CategoryLength PreferenceCategory = "length"
	// CategoryFormat 格式偏好
	CategoryFormat PreferenceCategory = "format"
)

// ExtractedPreference 提取的偏好
type ExtractedPreference struct {
	// Category 分类
	Category PreferenceCategory `json:"category"`

	// Content 内容
	Content string `json:"content"`

	// Keywords 匹配的关键词
	Keywords []string `json:"keywords"`

	// Confidence 置信度
	Confidence float64 `json:"confidence"`

	// Timestamp 提取时间
	Timestamp time.Time `json:"timestamp"`
}

// KeywordMatrix 关键词矩阵
type KeywordMatrix map[PreferenceCategory][]string

// Extractor 对话偏好提取器
type Extractor struct {
	// keywords 关键词矩阵
	keywords KeywordMatrix

	// config 配置
	config *ExtractorConfig
}

// ExtractorConfig 提取器配置
type ExtractorConfig struct {
	// MinConfidence 最低置信度
	MinConfidence float64

	// MaxExtractLength 最大提取长度
	MaxExtractLength int
}

// DefaultExtractorConfig 默认配置
func DefaultExtractorConfig() *ExtractorConfig {
	return &ExtractorConfig{
		MinConfidence:    0.5,
		MaxExtractLength: 200,
	}
}

// NewExtractor 创建提取器
func NewExtractor(keywords KeywordMatrix, config *ExtractorConfig) *Extractor {
	if keywords == nil {
		keywords = DefaultChineseKeywords()
	}
	if config == nil {
		config = DefaultExtractorConfig()
	}
	return &Extractor{
		keywords: keywords,
		config:   config,
	}
}

// Extract 从消息中提取偏好
func (e *Extractor) Extract(message string) []*ExtractedPreference {
	var preferences []*ExtractedPreference

	for category, keywords := range e.keywords {
		for _, keyword := range keywords {
			if strings.Contains(message, keyword) {
				// 提取包含关键词的句子
				content := e.extractSentence(message, keyword)
				if content == "" {
					continue
				}

				pref := &ExtractedPreference{
					Category:   category,
					Content:    content,
					Keywords:   []string{keyword},
					Confidence: e.calculateConfidence(message, keyword, category),
					Timestamp:  time.Now(),
				}

				if pref.Confidence >= e.config.MinConfidence {
					preferences = append(preferences, pref)
				}

				break // 每个类别只提取一次
			}
		}
	}

	return preferences
}

// HasPreferenceKeyword 检查消息是否包含偏好关键词
func (e *Extractor) HasPreferenceKeyword(message string) bool {
	for _, keywords := range e.keywords {
		for _, keyword := range keywords {
			if strings.Contains(message, keyword) {
				return true
			}
		}
	}
	return false
}

// extractSentence 提取包含关键词的句子
func (e *Extractor) extractSentence(text, keyword string) string {
	idx := strings.Index(text, keyword)
	if idx == -1 {
		return ""
	}

	// 转换为 rune 处理中文
	runes := []rune(text)
	keywordRunes := []rune(keyword)

	// 找到关键词在 rune 中的位置
	runeIdx := 0
	byteCount := 0
	for i, r := range runes {
		if byteCount >= idx {
			runeIdx = i
			break
		}
		byteCount += len(string(r))
	}

	// 句子结束符
	isSentenceEnd := func(r rune) bool {
		return r == '。' || r == '！' || r == '？' || r == '\n' ||
			r == '.' || r == '!' || r == '?'
	}

	// 向前找句子开始
	start := runeIdx
	for start > 0 {
		if isSentenceEnd(runes[start-1]) {
			break
		}
		start--
	}

	// 向后找句子结束
	end := runeIdx + len(keywordRunes)
	for end < len(runes) {
		if isSentenceEnd(runes[end]) {
			break
		}
		end++
	}

	// 使用 rune slice 提取句子
	sentence := strings.TrimSpace(string(runes[start:end]))

	// 限制长度（按 rune 计算）
	sentenceRunes := []rune(sentence)
	if len(sentenceRunes) > e.config.MaxExtractLength {
		// 截取关键词前后的内容
		keywordIdx := strings.Index(sentence, keyword)
		keywordRuneIdx := 0
		byteCount := 0
		for i, r := range sentenceRunes {
			if byteCount >= keywordIdx {
				keywordRuneIdx = i
				break
			}
			byteCount += len(string(r))
		}

		halfLen := e.config.MaxExtractLength / 2
		newStart := max(keywordRuneIdx-halfLen, 0)
		newEnd := min(keywordRuneIdx+len(keywordRunes)+halfLen, len(sentenceRunes))
		sentence = "..." + strings.TrimSpace(string(sentenceRunes[newStart:newEnd])) + "..."
	}

	return sentence
}

// calculateConfidence 计算置信度
func (e *Extractor) calculateConfidence(message, keyword string, category PreferenceCategory) float64 {
	confidence := 0.6

	// 约束类关键词置信度更高
	if category == CategoryConstraint {
		confidence = 0.8
	}

	// 多次出现加分
	count := strings.Count(message, keyword)
	if count > 1 {
		confidence += 0.1
	}

	// 消息较长说明更详细，加分
	if len(message) > 100 {
		confidence += 0.05
	}

	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// SetKeywords 设置关键词矩阵
func (e *Extractor) SetKeywords(keywords KeywordMatrix) {
	e.keywords = keywords
}

// AddKeywords 添加关键词
func (e *Extractor) AddKeywords(category PreferenceCategory, keywords []string) {
	if e.keywords == nil {
		e.keywords = make(KeywordMatrix)
	}
	e.keywords[category] = append(e.keywords[category], keywords...)
}
