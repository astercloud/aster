package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ObservationCompressor 观察结果压缩器接口
// 用于压缩工具执行结果，同时保留可恢复的引用信息
// 这是 Manus 团队"文件系统作为上下文"理念的实现
type ObservationCompressor interface {
	// Compress 压缩观察结果
	Compress(ctx context.Context, toolName string, output string) (*CompressedObservation, error)

	// CanRecover 检查是否可以恢复原始内容
	CanRecover(compressed *CompressedObservation) bool

	// Recover 恢复原始内容（如果可能）
	Recover(ctx context.Context, compressed *CompressedObservation) (string, error)
}

// CompressedObservation 压缩后的观察结果
type CompressedObservation struct {
	// Summary 压缩后的摘要
	Summary string `json:"summary"`

	// References 保留的引用（URL、文件路径等）
	References []Reference `json:"references,omitempty"`

	// Hash 原始内容的哈希值（用于验证恢复）
	Hash string `json:"hash"`

	// OriginalLength 原始内容长度
	OriginalLength int `json:"original_length"`

	// Recoverable 是否可恢复
	Recoverable bool `json:"recoverable"`

	// ToolName 产生此观察的工具名
	ToolName string `json:"tool_name"`

	// CompressionRatio 压缩比率
	CompressionRatio float64 `json:"compression_ratio"`
}

// Reference 引用信息
type Reference struct {
	// Type 引用类型: "file_path", "url", "function", "class", "line_range"
	Type string `json:"type"`

	// Value 引用值
	Value string `json:"value"`

	// Context 上下文信息（可选）
	Context string `json:"context,omitempty"`

	// LineStart 起始行号（可选，用于文件引用）
	LineStart int `json:"line_start,omitempty"`

	// LineEnd 结束行号（可选，用于文件引用）
	LineEnd int `json:"line_end,omitempty"`
}

// ReferenceType 引用类型常量
const (
	ReferenceTypeFilePath  = "file_path"
	ReferenceTypeURL       = "url"
	ReferenceTypeFunction  = "function"
	ReferenceTypeClass     = "class"
	ReferenceTypeLineRange = "line_range"
	ReferenceTypeVariable  = "variable"
)

// DefaultObservationCompressor 默认的观察结果压缩器实现
type DefaultObservationCompressor struct {
	// maxSummaryLength 摘要最大长度
	maxSummaryLength int

	// minCompressLength 触发压缩的最小长度
	minCompressLength int

	// preservePatterns 需要保留的模式
	preservePatterns []*regexp.Regexp
}

// NewDefaultObservationCompressor 创建默认压缩器
func NewDefaultObservationCompressor() *DefaultObservationCompressor {
	return &DefaultObservationCompressor{
		maxSummaryLength:  2000,
		minCompressLength: 3000,
		preservePatterns:  defaultPreservePatterns(),
	}
}

// ObservationCompressorConfig 压缩器配置
type ObservationCompressorConfig struct {
	MaxSummaryLength  int
	MinCompressLength int
}

// NewObservationCompressorWithConfig 创建带配置的压缩器
func NewObservationCompressorWithConfig(config *ObservationCompressorConfig) *DefaultObservationCompressor {
	c := NewDefaultObservationCompressor()
	if config.MaxSummaryLength > 0 {
		c.maxSummaryLength = config.MaxSummaryLength
	}
	if config.MinCompressLength > 0 {
		c.minCompressLength = config.MinCompressLength
	}
	return c
}

// defaultPreservePatterns 默认需要保留的模式
func defaultPreservePatterns() []*regexp.Regexp {
	patterns := []string{
		// 文件路径
		`(?:^|[\s"'\(])(/[a-zA-Z0-9_\-./]+\.[a-zA-Z0-9]+)`,
		`(?:^|[\s"'\(])([a-zA-Z]:\\[a-zA-Z0-9_\-\\./]+\.[a-zA-Z0-9]+)`,

		// URL
		`https?://[^\s<>"'\)]+`,

		// 函数/方法定义
		`func\s+(\w+)\s*\(`,
		`def\s+(\w+)\s*\(`,
		`function\s+(\w+)\s*\(`,

		// 类定义
		`class\s+(\w+)`,
		`type\s+(\w+)\s+struct`,

		// 行号引用
		`:(\d+)(?:[-:](\d+))?`,
	}

	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		if re, err := regexp.Compile(p); err == nil {
			compiled = append(compiled, re)
		}
	}
	return compiled
}

// Compress 压缩观察结果
func (c *DefaultObservationCompressor) Compress(ctx context.Context, toolName string, output string) (*CompressedObservation, error) {
	originalLength := len(output)

	// 如果内容较短，不进行压缩
	if originalLength < c.minCompressLength {
		return &CompressedObservation{
			Summary:          output,
			Hash:             hashContent(output),
			OriginalLength:   originalLength,
			Recoverable:      true,
			ToolName:         toolName,
			CompressionRatio: 1.0,
		}, nil
	}

	// 提取引用
	references := c.extractReferences(output)

	// 根据工具类型选择压缩策略
	summary := c.compressByToolType(toolName, output)

	// 确保摘要不超过最大长度
	if len(summary) > c.maxSummaryLength {
		summary = summary[:c.maxSummaryLength] + "\n... (truncated)"
	}

	return &CompressedObservation{
		Summary:          summary,
		References:       references,
		Hash:             hashContent(output),
		OriginalLength:   originalLength,
		Recoverable:      c.isRecoverable(toolName, references),
		ToolName:         toolName,
		CompressionRatio: float64(len(summary)) / float64(originalLength),
	}, nil
}

// extractReferences 提取引用
func (c *DefaultObservationCompressor) extractReferences(content string) []Reference {
	var refs []Reference
	seen := make(map[string]bool)

	// 提取文件路径
	filePathRe := regexp.MustCompile(`(?:^|[\s"'\(])(/[a-zA-Z0-9_\-./]+(?:\.[a-zA-Z0-9]+)?)`)
	for _, match := range filePathRe.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			path := match[1]
			key := ReferenceTypeFilePath + ":" + path
			if !seen[key] && isValidFilePath(path) {
				seen[key] = true
				refs = append(refs, Reference{
					Type:  ReferenceTypeFilePath,
					Value: path,
				})
			}
		}
	}

	// 提取 URL
	urlRe := regexp.MustCompile(`https?://[^\s<>"'\)]+`)
	for _, match := range urlRe.FindAllString(content, -1) {
		key := ReferenceTypeURL + ":" + match
		if !seen[key] {
			seen[key] = true
			refs = append(refs, Reference{
				Type:  ReferenceTypeURL,
				Value: match,
			})
		}
	}

	// 提取函数定义
	funcPatterns := []*regexp.Regexp{
		regexp.MustCompile(`func\s+(\w+)\s*\(`),
		regexp.MustCompile(`def\s+(\w+)\s*\(`),
		regexp.MustCompile(`function\s+(\w+)\s*\(`),
	}
	for _, re := range funcPatterns {
		for _, match := range re.FindAllStringSubmatch(content, -1) {
			if len(match) > 1 {
				funcName := match[1]
				key := ReferenceTypeFunction + ":" + funcName
				if !seen[key] {
					seen[key] = true
					refs = append(refs, Reference{
						Type:  ReferenceTypeFunction,
						Value: funcName,
					})
				}
			}
		}
	}

	// 提取类/结构体定义
	classPatterns := []*regexp.Regexp{
		regexp.MustCompile(`class\s+(\w+)`),
		regexp.MustCompile(`type\s+(\w+)\s+struct`),
		regexp.MustCompile(`interface\s+(\w+)`),
	}
	for _, re := range classPatterns {
		for _, match := range re.FindAllStringSubmatch(content, -1) {
			if len(match) > 1 {
				className := match[1]
				key := ReferenceTypeClass + ":" + className
				if !seen[key] {
					seen[key] = true
					refs = append(refs, Reference{
						Type:  ReferenceTypeClass,
						Value: className,
					})
				}
			}
		}
	}

	return refs
}

// compressByToolType 根据工具类型压缩
func (c *DefaultObservationCompressor) compressByToolType(toolName string, content string) string {
	switch toolName {
	case "Read":
		return c.compressFileRead(content)
	case "Grep", "Glob":
		return c.compressSearchResults(content)
	case "Bash":
		return c.compressBashOutput(content)
	case "WebFetch":
		return c.compressWebContent(content)
	default:
		return c.compressGeneric(content)
	}
}

// compressFileRead 压缩文件读取结果
func (c *DefaultObservationCompressor) compressFileRead(content string) string {
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	if totalLines <= 50 {
		return content
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("[File content: %d lines]\n\n", totalLines))

	// 保留前 20 行
	summary.WriteString("=== First 20 lines ===\n")
	for i := 0; i < 20 && i < len(lines); i++ {
		summary.WriteString(lines[i])
		summary.WriteString("\n")
	}

	// 保留最后 20 行
	summary.WriteString("\n=== Last 20 lines ===\n")
	start := max(totalLines-20, 20)
	for i := start; i < totalLines; i++ {
		summary.WriteString(lines[i])
		summary.WriteString("\n")
	}

	// 提取关键定义（函数、类等）
	definitions := extractDefinitions(content)
	if len(definitions) > 0 {
		summary.WriteString("\n=== Key Definitions ===\n")
		for _, def := range definitions {
			summary.WriteString(fmt.Sprintf("- %s\n", def))
		}
	}

	return summary.String()
}

// compressSearchResults 压缩搜索结果
func (c *DefaultObservationCompressor) compressSearchResults(content string) string {
	lines := strings.Split(content, "\n")

	if len(lines) <= 30 {
		return content
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("[Search results: %d matches]\n\n", len(lines)))

	// 保留前 15 个结果
	summary.WriteString("=== First 15 results ===\n")
	for i := 0; i < 15 && i < len(lines); i++ {
		summary.WriteString(lines[i])
		summary.WriteString("\n")
	}

	// 保留最后 15 个结果
	if len(lines) > 30 {
		summary.WriteString("\n=== Last 15 results ===\n")
		for i := len(lines) - 15; i < len(lines); i++ {
			summary.WriteString(lines[i])
			summary.WriteString("\n")
		}
	}

	return summary.String()
}

// compressBashOutput 压缩 Bash 输出
func (c *DefaultObservationCompressor) compressBashOutput(content string) string {
	lines := strings.Split(content, "\n")

	if len(lines) <= 40 {
		return content
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("[Command output: %d lines]\n\n", len(lines)))

	// 检查是否包含错误
	hasError := strings.Contains(strings.ToLower(content), "error") ||
		strings.Contains(strings.ToLower(content), "failed") ||
		strings.Contains(strings.ToLower(content), "exception")

	if hasError {
		// 如果有错误，保留更多上下文
		summary.WriteString("=== Output (error detected, showing more context) ===\n")
		for i := 0; i < 30 && i < len(lines); i++ {
			summary.WriteString(lines[i])
			summary.WriteString("\n")
		}
		if len(lines) > 30 {
			summary.WriteString("\n... (truncated) ...\n\n")
			for i := len(lines) - 20; i < len(lines); i++ {
				summary.WriteString(lines[i])
				summary.WriteString("\n")
			}
		}
	} else {
		// 正常输出，只保留首尾
		summary.WriteString("=== First 15 lines ===\n")
		for i := 0; i < 15 && i < len(lines); i++ {
			summary.WriteString(lines[i])
			summary.WriteString("\n")
		}
		summary.WriteString("\n=== Last 15 lines ===\n")
		for i := len(lines) - 15; i < len(lines); i++ {
			summary.WriteString(lines[i])
			summary.WriteString("\n")
		}
	}

	return summary.String()
}

// compressWebContent 压缩 Web 内容
func (c *DefaultObservationCompressor) compressWebContent(content string) string {
	// 保留 URL 和标题
	var summary strings.Builder

	// 尝试提取标题
	titleRe := regexp.MustCompile(`<title>([^<]+)</title>`)
	if match := titleRe.FindStringSubmatch(content); len(match) > 1 {
		summary.WriteString(fmt.Sprintf("Title: %s\n\n", match[1]))
	}

	// 移除 HTML 标签（简单处理）
	cleanContent := regexp.MustCompile(`<[^>]+>`).ReplaceAllString(content, " ")
	cleanContent = regexp.MustCompile(`\s+`).ReplaceAllString(cleanContent, " ")

	if len(cleanContent) > c.maxSummaryLength {
		cleanContent = cleanContent[:c.maxSummaryLength]
	}

	summary.WriteString(cleanContent)

	return summary.String()
}

// compressGeneric 通用压缩
func (c *DefaultObservationCompressor) compressGeneric(content string) string {
	lines := strings.Split(content, "\n")

	if len(lines) <= 30 {
		return content
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("[Content: %d lines, %d chars]\n\n", len(lines), len(content)))

	// 保留前 15 行
	for i := 0; i < 15 && i < len(lines); i++ {
		summary.WriteString(lines[i])
		summary.WriteString("\n")
	}

	summary.WriteString("\n... (content truncated) ...\n\n")

	// 保留最后 15 行
	for i := len(lines) - 15; i < len(lines); i++ {
		summary.WriteString(lines[i])
		summary.WriteString("\n")
	}

	return summary.String()
}

// isRecoverable 判断是否可恢复
func (c *DefaultObservationCompressor) isRecoverable(toolName string, refs []Reference) bool {
	// 如果有文件路径引用，通常可以恢复
	for _, ref := range refs {
		if ref.Type == ReferenceTypeFilePath {
			return true
		}
	}

	// Read 工具的结果通常可以通过重新读取恢复
	if toolName == "Read" {
		return true
	}

	return false
}

// CanRecover 检查是否可以恢复
func (c *DefaultObservationCompressor) CanRecover(compressed *CompressedObservation) bool {
	return compressed.Recoverable
}

// Recover 恢复原始内容
// 注意：这需要外部工具（如文件系统访问）的支持
func (c *DefaultObservationCompressor) Recover(ctx context.Context, compressed *CompressedObservation) (string, error) {
	if !compressed.Recoverable {
		return "", errors.New("content is not recoverable")
	}

	// 对于文件引用，返回如何恢复的指令
	for _, ref := range compressed.References {
		if ref.Type == ReferenceTypeFilePath {
			return fmt.Sprintf("[To recover: Read file %s]", ref.Value), nil
		}
	}

	return "", errors.New("no recovery method available")
}

// 辅助函数

func hashContent(content string) string {
	h := sha256.New()
	h.Write([]byte(content))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func isValidFilePath(path string) bool {
	// 排除常见的非文件路径
	excludePatterns := []string{
		"/dev/",
		"/proc/",
		"/sys/",
		"http://",
		"https://",
	}
	for _, pattern := range excludePatterns {
		if strings.HasPrefix(path, pattern) {
			return false
		}
	}

	// 必须包含有效的扩展名、是目录、或是 /usr/local/bin 等常见可执行文件路径
	if strings.Contains(path, ".") || strings.HasSuffix(path, "/") {
		return true
	}

	// 允许 /usr/local/bin, /usr/bin, /bin 等常见可执行文件路径
	commonBinPaths := []string{"/usr/local/bin/", "/usr/bin/", "/bin/", "/sbin/", "/usr/sbin/"}
	for _, binPath := range commonBinPaths {
		if strings.HasPrefix(path, binPath) {
			return true
		}
	}

	return false
}

func extractDefinitions(content string) []string {
	var defs []string
	seen := make(map[string]bool)

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`func\s+(\([^)]+\)\s+)?(\w+)\s*\(`),
		regexp.MustCompile(`def\s+(\w+)\s*\(`),
		regexp.MustCompile(`class\s+(\w+)`),
		regexp.MustCompile(`type\s+(\w+)\s+(?:struct|interface)`),
	}

	for _, re := range patterns {
		for _, match := range re.FindAllString(content, -1) {
			if !seen[match] {
				seen[match] = true
				defs = append(defs, strings.TrimSpace(match))
			}
		}
	}

	// 限制数量
	if len(defs) > 20 {
		defs = defs[:20]
	}

	return defs
}
