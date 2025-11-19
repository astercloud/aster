package media

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
)

// Processor 媒体处理器
type Processor struct {
	// MaxImageSize 最大图片尺寸（像素）
	MaxImageSize int

	// MaxFileSize 最大文件大小（字节）
	MaxFileSize int64

	// AllowedImageTypes 允许的图片类型
	AllowedImageTypes []string

	// AllowedVideoTypes 允许的视频类型
	AllowedVideoTypes []string

	// AllowedAudioTypes 允许的音频类型
	AllowedAudioTypes []string
}

// NewProcessor 创建媒体处理器
func NewProcessor() *Processor {
	return &Processor{
		MaxImageSize: 4096,
		MaxFileSize:  10 * 1024 * 1024, // 10MB
		AllowedImageTypes: []string{
			"image/jpeg", "image/png", "image/gif",
			"image/webp", "image/svg+xml",
		},
		AllowedVideoTypes: []string{
			"video/mp4", "video/webm", "video/quicktime",
		},
		AllowedAudioTypes: []string{
			"audio/mpeg", "audio/wav", "audio/ogg", "audio/mp4",
		},
	}
}

// ValidateImage 验证图片
func (p *Processor) ValidateImage(img *Image) error {
	// 检查大小
	if img.Size > p.MaxFileSize {
		return fmt.Errorf("image size %d exceeds maximum %d", img.Size, p.MaxFileSize)
	}

	// 检查类型
	if img.MimeType != "" {
		valid := false
		for _, allowed := range p.AllowedImageTypes {
			if img.MimeType == allowed {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("image type %s not allowed", img.MimeType)
		}
	}

	return nil
}

// GetImageDimensions 获取图片尺寸
func (p *Processor) GetImageDimensions(path string) (width, height int, err error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = file.Close() }()

	img, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, err
	}

	return img.Width, img.Height, nil
}

// ValidateVideo 验证视频
func (p *Processor) ValidateVideo(video *Video) error {
	// 检查大小
	if video.Size > p.MaxFileSize {
		return fmt.Errorf("video size %d exceeds maximum %d", video.Size, p.MaxFileSize)
	}

	// 检查类型
	if video.MimeType != "" {
		valid := false
		for _, allowed := range p.AllowedVideoTypes {
			if video.MimeType == allowed {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("video type %s not allowed", video.MimeType)
		}
	}

	return nil
}

// ValidateAudio 验证音频
func (p *Processor) ValidateAudio(audio *Audio) error {
	// 检查大小
	if audio.Size > p.MaxFileSize {
		return fmt.Errorf("audio size %d exceeds maximum %d", audio.Size, p.MaxFileSize)
	}

	// 检查类型
	if audio.MimeType != "" {
		valid := false
		for _, allowed := range p.AllowedAudioTypes {
			if audio.MimeType == allowed {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("audio type %s not allowed", audio.MimeType)
		}
	}

	return nil
}

// ConvertImages 转换图片列表
func ConvertImages(sources []string) ([]*Image, error) {
	images := make([]*Image, 0, len(sources))
	for _, src := range sources {
		img, err := NewImage(src)
		if err != nil {
			return nil, err
		}
		images = append(images, img)
	}
	return images, nil
}

// ConvertVideos 转换视频列表
func ConvertVideos(sources []string) ([]*Video, error) {
	videos := make([]*Video, 0, len(sources))
	for _, src := range sources {
		video, err := NewVideo(src)
		if err != nil {
			return nil, err
		}
		videos = append(videos, video)
	}
	return videos, nil
}

// ConvertAudios 转换音频列表
func ConvertAudios(sources []string) ([]*Audio, error) {
	audios := make([]*Audio, 0, len(sources))
	for _, src := range sources {
		audio, err := NewAudio(src)
		if err != nil {
			return nil, err
		}
		audios = append(audios, audio)
	}
	return audios, nil
}
