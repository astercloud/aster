package run

// Input 统一的运行输入
type Input struct {
	// Content 输入内容
	Content interface{}

	// Images 图片列表
	Images []string

	// Videos 视频列表
	Videos []string

	// Audio 音频列表
	Audio []string

	// Files 文件列表
	Files []string

	// Metadata 元数据
	Metadata map[string]interface{}

	// AdditionalMessages 额外消息
	AdditionalMessages []interface{}
}

// NewInput 创建输入
func NewInput(content interface{}) *Input {
	return &Input{
		Content:  content,
		Metadata: make(map[string]interface{}),
	}
}

// WithImages 添加图片
func (i *Input) WithImages(images ...string) *Input {
	i.Images = append(i.Images, images...)
	return i
}

// WithVideos 添加视频
func (i *Input) WithVideos(videos ...string) *Input {
	i.Videos = append(i.Videos, videos...)
	return i
}

// WithAudio 添加音频
func (i *Input) WithAudio(audio ...string) *Input {
	i.Audio = append(i.Audio, audio...)
	return i
}

// WithFiles 添加文件
func (i *Input) WithFiles(files ...string) *Input {
	i.Files = append(i.Files, files...)
	return i
}

// WithMetadata 添加元数据
func (i *Input) WithMetadata(key string, value interface{}) *Input {
	i.Metadata[key] = value
	return i
}

// ContentString 获取字符串形式的内容
func (i *Input) ContentString() string {
	if s, ok := i.Content.(string); ok {
		return s
	}
	return ""
}
