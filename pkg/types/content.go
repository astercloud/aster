package types

import "strings"

// MultimodalContent 多模态内容接口
// 扩展 ContentBlock 以支持图片、音频、视频等多模态输入
type MultimodalContent interface {
	ContentBlock
	GetMediaType() string
}

// ImageContent 图片内容
type ImageContent struct {
	// Type 图片来源类型: "url", "base64"
	Type string `json:"type"`

	// Source 图片源
	// - 当 Type="url" 时，Source 是图片 URL
	// - 当 Type="base64" 时，Source 是 base64 编码的图片数据
	Source string `json:"source"`

	// MimeType MIME 类型，如 "image/jpeg", "image/png"
	MimeType string `json:"mime_type,omitempty"`

	// Detail 图片细节级别 (OpenAI)
	// - "low": 低分辨率快速处理
	// - "high": 高分辨率详细分析
	// - "auto": 自动选择
	Detail string `json:"detail,omitempty"`
}

func (i *ImageContent) IsContentBlock() {}

func (i *ImageContent) GetMediaType() string {
	return "image"
}

// AudioContent 音频内容
type AudioContent struct {
	// Type 音频来源类型: "url", "base64"
	Type string `json:"type"`

	// Source 音频源
	Source string `json:"source"`

	// MimeType MIME 类型，如 "audio/mp3", "audio/wav"
	MimeType string `json:"mime_type,omitempty"`

	// Format 音频格式 (用于某些 Provider)
	Format string `json:"format,omitempty"`

	// Transcript 音频转录文本 (可选)
	Transcript string `json:"transcript,omitempty"`
}

func (a *AudioContent) IsContentBlock() {}

func (a *AudioContent) GetMediaType() string {
	return "audio"
}

// VideoContent 视频内容
type VideoContent struct {
	// Type 视频来源类型: "url", "base64"
	Type string `json:"type"`

	// Source 视频源
	Source string `json:"source"`

	// MimeType MIME 类型，如 "video/mp4", "video/webm"
	MimeType string `json:"mime_type,omitempty"`

	// Thumbnail 视频缩略图 URL (可选)
	Thumbnail string `json:"thumbnail,omitempty"`

	// Duration 视频时长(秒)
	Duration float64 `json:"duration,omitempty"`
}

func (v *VideoContent) IsContentBlock() {}

func (v *VideoContent) GetMediaType() string {
	return "video"
}

// DocumentContent 文档内容
type DocumentContent struct {
	// Type 文档类型: "url", "base64", "text"
	Type string `json:"type"`

	// Source 文档源
	Source string `json:"source"`

	// MimeType MIME 类型，如 "application/pdf", "text/plain"
	MimeType string `json:"mime_type,omitempty"`

	// Title 文档标题
	Title string `json:"title,omitempty"`
}

func (d *DocumentContent) IsContentBlock() {}

func (d *DocumentContent) GetMediaType() string {
	return "document"
}

// CacheControlBlock 缓存控制块
// 用于 Prompt Caching 功能
type CacheControlBlock struct {
	// Type 缓存类型: "ephemeral"
	Type string `json:"type"`

	// Content 被缓存的内容
	Content ContentBlock `json:"content"`
}

func (c *CacheControlBlock) IsContentBlock() {}

// ContentBlockHelper 内容块辅助函数
type ContentBlockHelper struct{}

// ExtractText 从 ContentBlocks 中提取所有文本
func (h ContentBlockHelper) ExtractText(blocks []ContentBlock) string {
	var text string
	var textSb123 strings.Builder
	for _, block := range blocks {
		if tb, ok := block.(*TextBlock); ok {
			textSb123.WriteString(tb.Text)
		}
	}
	text += textSb123.String()
	return text
}

// HasMultimodal 检查是否包含多模态内容
func (h ContentBlockHelper) HasMultimodal(blocks []ContentBlock) bool {
	for _, block := range blocks {
		if _, ok := block.(MultimodalContent); ok {
			return true
		}
	}
	return false
}

// GetMediaTypes 获取所有媒体类型
func (h ContentBlockHelper) GetMediaTypes(blocks []ContentBlock) []string {
	types := make(map[string]bool)
	for _, block := range blocks {
		if mc, ok := block.(MultimodalContent); ok {
			types[mc.GetMediaType()] = true
		}
	}

	result := make([]string, 0, len(types))
	for t := range types {
		result = append(result, t)
	}
	return result
}
