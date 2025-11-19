package media

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MediaType 媒体类型
type MediaType string

const (
	MediaTypeImage MediaType = "image"
	MediaTypeVideo MediaType = "video"
	MediaTypeAudio MediaType = "audio"
	MediaTypeFile  MediaType = "file"
)

// Image 图片
type Image struct {
	// URL 图片 URL
	URL string `json:"url,omitempty"`

	// Data Base64 编码的图片数据
	Data string `json:"data,omitempty"`

	// MimeType MIME 类型
	MimeType string `json:"mime_type,omitempty"`

	// Name 文件名
	Name string `json:"name,omitempty"`

	// Width 宽度
	Width int `json:"width,omitempty"`

	// Height 高度
	Height int `json:"height,omitempty"`

	// Size 文件大小（字节）
	Size int64 `json:"size,omitempty"`
}

// NewImage 创建图片
func NewImage(source string) (*Image, error) {
	img := &Image{}

	// 判断是 URL 还是文件路径
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		img.URL = source
	} else if strings.HasPrefix(source, "data:") {
		// Data URL
		parts := strings.SplitN(source, ",", 2)
		if len(parts) == 2 {
			img.Data = parts[1]
			// 解析 MIME 类型
			if strings.Contains(parts[0], ";") {
				img.MimeType = strings.Split(parts[0], ";")[0][5:] // 去掉 "data:"
			}
		}
	} else {
		// 文件路径
		return NewImageFromFile(source)
	}

	return img, nil
}

// NewImageFromFile 从文件创建图片
func NewImageFromFile(path string) (*Image, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read image file: %w", err)
	}

	img := &Image{
		Name: filepath.Base(path),
		Size: int64(len(data)),
		Data: base64.StdEncoding.EncodeToString(data),
	}

	// 根据扩展名设置 MIME 类型
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		img.MimeType = "image/jpeg"
	case ".png":
		img.MimeType = "image/png"
	case ".gif":
		img.MimeType = "image/gif"
	case ".webp":
		img.MimeType = "image/webp"
	case ".svg":
		img.MimeType = "image/svg+xml"
	default:
		img.MimeType = "image/*"
	}

	return img, nil
}

// ToDataURL 转换为 Data URL
func (i *Image) ToDataURL() string {
	if i.URL != "" {
		return i.URL
	}
	if i.Data != "" && i.MimeType != "" {
		return fmt.Sprintf("data:%s;base64,%s", i.MimeType, i.Data)
	}
	return ""
}

// Video 视频
type Video struct {
	// URL 视频 URL
	URL string `json:"url,omitempty"`

	// Data Base64 编码的视频数据
	Data string `json:"data,omitempty"`

	// MimeType MIME 类型
	MimeType string `json:"mime_type,omitempty"`

	// Name 文件名
	Name string `json:"name,omitempty"`

	// Duration 时长（秒）
	Duration float64 `json:"duration,omitempty"`

	// Width 宽度
	Width int `json:"width,omitempty"`

	// Height 高度
	Height int `json:"height,omitempty"`

	// Size 文件大小（字节）
	Size int64 `json:"size,omitempty"`

	// Thumbnail 缩略图
	Thumbnail *Image `json:"thumbnail,omitempty"`
}

// NewVideo 创建视频
func NewVideo(source string) (*Video, error) {
	video := &Video{}

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		video.URL = source
	} else {
		return NewVideoFromFile(source)
	}

	return video, nil
}

// NewVideoFromFile 从文件创建视频
func NewVideoFromFile(path string) (*Video, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read video file: %w", err)
	}

	video := &Video{
		Name: filepath.Base(path),
		Size: int64(len(data)),
		Data: base64.StdEncoding.EncodeToString(data),
	}

	// 根据扩展名设置 MIME 类型
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".mp4":
		video.MimeType = "video/mp4"
	case ".webm":
		video.MimeType = "video/webm"
	case ".mov":
		video.MimeType = "video/quicktime"
	case ".avi":
		video.MimeType = "video/x-msvideo"
	default:
		video.MimeType = "video/*"
	}

	return video, nil
}

// Audio 音频
type Audio struct {
	// URL 音频 URL
	URL string `json:"url,omitempty"`

	// Data Base64 编码的音频数据
	Data string `json:"data,omitempty"`

	// MimeType MIME 类型
	MimeType string `json:"mime_type,omitempty"`

	// Name 文件名
	Name string `json:"name,omitempty"`

	// Duration 时长（秒）
	Duration float64 `json:"duration,omitempty"`

	// Size 文件大小（字节）
	Size int64 `json:"size,omitempty"`

	// Transcript 转录文本
	Transcript string `json:"transcript,omitempty"`
}

// NewAudio 创建音频
func NewAudio(source string) (*Audio, error) {
	audio := &Audio{}

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		audio.URL = source
	} else {
		return NewAudioFromFile(source)
	}

	return audio, nil
}

// NewAudioFromFile 从文件创建音频
func NewAudioFromFile(path string) (*Audio, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio file: %w", err)
	}

	audio := &Audio{
		Name: filepath.Base(path),
		Size: int64(len(data)),
		Data: base64.StdEncoding.EncodeToString(data),
	}

	// 根据扩展名设置 MIME 类型
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".mp3":
		audio.MimeType = "audio/mpeg"
	case ".wav":
		audio.MimeType = "audio/wav"
	case ".ogg":
		audio.MimeType = "audio/ogg"
	case ".m4a":
		audio.MimeType = "audio/mp4"
	case ".flac":
		audio.MimeType = "audio/flac"
	default:
		audio.MimeType = "audio/*"
	}

	return audio, nil
}

// File 通用文件
type File struct {
	// URL 文件 URL
	URL string `json:"url,omitempty"`

	// Data Base64 编码的文件数据
	Data string `json:"data,omitempty"`

	// MimeType MIME 类型
	MimeType string `json:"mime_type,omitempty"`

	// Name 文件名
	Name string `json:"name,omitempty"`

	// Size 文件大小（字节）
	Size int64 `json:"size,omitempty"`

	// Extension 扩展名
	Extension string `json:"extension,omitempty"`
}

// NewFile 创建文件
func NewFile(source string) (*File, error) {
	file := &File{}

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		file.URL = source
	} else {
		return NewFileFromPath(source)
	}

	return file, nil
}

// NewFileFromPath 从路径创建文件
func NewFileFromPath(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	file := &File{
		Name:      filepath.Base(path),
		Size:      int64(len(data)),
		Extension: filepath.Ext(path),
		Data:      base64.StdEncoding.EncodeToString(data),
	}

	// 尝试检测 MIME 类型
	file.MimeType = detectMimeType(path)

	return file, nil
}

// detectMimeType 检测 MIME 类型
func detectMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".txt":
		return "text/plain"
	case ".pdf":
		return "application/pdf"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".csv":
		return "text/csv"
	case ".zip":
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}
