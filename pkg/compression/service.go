package compression

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/types"
)

// CompressionMode 压缩模式
type CompressionMode string

const (
	ModeSimple CompressionMode = "simple" // 基于规则的快速压缩
	ModeLLM    CompressionMode = "llm"    // LLM 驱动的智能压缩
	ModeHybrid CompressionMode = "hybrid" // 混合模式（先规则后 LLM）
)

// CompressionLevel 压缩级别
type CompressionLevel int

const (
	LevelLight      CompressionLevel = 1 // 轻度压缩（60-70%）
	LevelModerate   CompressionLevel = 2 // 中度压缩（40-50%）
	LevelAggressive CompressionLevel = 3 // 激进压缩（20-30%）
)

// CompressOptions 压缩选项
type CompressOptions struct {
	TargetLength     int              // 目标长度（字符数）
	Mode             CompressionMode  // 压缩模式
	Level            CompressionLevel // 压缩级别
	PreserveSections []string         // 必须保留的段落标题
	Language         string           // 语言（zh/en）
	UseCache         bool             // 是否使用缓存
}

// CompressResult 压缩结果
type CompressResult struct {
	Compressed       string        // 压缩后的内容
	OriginalLength   int           // 原始长度
	CompressedLength int           // 压缩后长度
	CompressionRatio float64       // 压缩率
	TokensSaved      int           // 节省的 Token 数
	Duration         time.Duration // 压缩耗时
	Mode             string        // 使用的模式
	CacheHit         bool          // 是否命中缓存
}

// CompressionStats 压缩统计
type CompressionStats struct {
	TotalCompressions int           // 总压缩次数
	CacheHits         int           // 缓存命中次数
	CacheMisses       int           // 缓存未命中次数
	TotalDuration     time.Duration // 总耗时
	AverageRatio      float64       // 平均压缩率
	TotalTokensSaved  int           // 总节省 Token 数
}

// CompressionService 统一的压缩服务接口
type CompressionService interface {
	// CompressSystemPrompt 压缩 System Prompt
	CompressSystemPrompt(ctx context.Context, prompt string, opts *CompressOptions) (*CompressResult, error)

	// CompressMessages 压缩消息历史（复用现有实现）
	CompressMessages(ctx context.Context, messages []types.Message, maxTokens int) ([]types.Message, error)

	// GetStats 获取统计信息
	GetStats() *CompressionStats

	// ResetStats 重置统计信息
	ResetStats()

	// ClearCache 清空缓存
	ClearCache()
}

// PromptCompressor 压缩器接口（与 agent.EnhancedPromptCompressor 兼容）
type PromptCompressor interface {
	Compress(ctx context.Context, prompt string, opts *PromptCompressOptions) (*PromptCompressResult, error)
}

// PromptCompressOptions 与 agent.CompressOptions 兼容的选项
type PromptCompressOptions struct {
	TargetLength     int
	TargetTokens     int
	Mode             string
	Level            int
	PreserveSections []string
}

// PromptCompressResult 与 agent.CompressResult 兼容的结果
type PromptCompressResult struct {
	Compressed       string
	OriginalLength   int
	CompressedLength int
	OriginalTokens   int
	CompressedTokens int
	CompressionRatio float64
	TokensSaved      int
	Mode             string
}

// DefaultCompressionService 默认压缩服务实现
type DefaultCompressionService struct {
	provider   provider.Provider // LLM Provider（用于 LLM 压缩）
	compressor PromptCompressor  // 底层压缩器
	cache      *CompressionCache // 压缩缓存
	stats      *CompressionStats // 统计信息
	mu         sync.RWMutex      // 保护统计信息
	language   string            // 语言
}

// NewDefaultCompressionService 创建默认压缩服务
func NewDefaultCompressionService(prov provider.Provider, cacheSize int) *DefaultCompressionService {
	if cacheSize <= 0 {
		cacheSize = 100 // 默认缓存 100 个结果
	}

	return &DefaultCompressionService{
		provider: prov,
		cache:    NewCompressionCache(cacheSize),
		stats: &CompressionStats{
			TotalCompressions: 0,
			CacheHits:         0,
			CacheMisses:       0,
			TotalDuration:     0,
			AverageRatio:      0,
			TotalTokensSaved:  0,
		},
		language: "zh",
	}
}

// NewDefaultCompressionServiceWithOptions 创建带选项的默认压缩服务
func NewDefaultCompressionServiceWithOptions(prov provider.Provider, cacheSize int, language string) *DefaultCompressionService {
	svc := NewDefaultCompressionService(prov, cacheSize)
	if language != "" {
		svc.language = language
	}
	return svc
}

// SetCompressor 设置底层压缩器
func (s *DefaultCompressionService) SetCompressor(compressor PromptCompressor) {
	s.compressor = compressor
}

// CompressSystemPrompt 压缩 System Prompt
func (s *DefaultCompressionService) CompressSystemPrompt(ctx context.Context, prompt string, opts *CompressOptions) (*CompressResult, error) {
	startTime := time.Now()

	// 设置默认选项
	if opts == nil {
		opts = &CompressOptions{
			Mode:     ModeHybrid,
			Level:    LevelModerate,
			Language: "zh",
			UseCache: true,
		}
	}

	// 检查缓存
	if opts.UseCache {
		cacheKey := s.generateCacheKey(prompt, opts)
		if cached := s.cache.Get(cacheKey); cached != nil {
			s.updateStats(cached, true)
			cached.CacheHit = true
			return cached, nil
		}
	}

	// 执行压缩
	var result *CompressResult
	var err error

	switch opts.Mode {
	case ModeSimple:
		result, err = s.compressSimple(ctx, prompt, opts)
	case ModeLLM:
		result, err = s.compressLLM(ctx, prompt, opts)
	case ModeHybrid:
		result, err = s.compressHybrid(ctx, prompt, opts)
	default:
		return nil, fmt.Errorf("unsupported compression mode: %s", opts.Mode)
	}

	if err != nil {
		return nil, err
	}

	// 设置结果信息
	result.Duration = time.Since(startTime)
	result.Mode = string(opts.Mode)
	result.CacheHit = false

	// 缓存结果
	if opts.UseCache {
		cacheKey := s.generateCacheKey(prompt, opts)
		s.cache.Set(cacheKey, result)
	}

	// 更新统计
	s.updateStats(result, false)

	return result, nil
}

// CompressMessages 压缩消息历史（占位符，后续集成现有实现）
func (s *DefaultCompressionService) CompressMessages(ctx context.Context, messages []types.Message, maxTokens int) ([]types.Message, error) {
	// TODO: 集成 pkg/context/ 和 pkg/memory/ 的现有实现
	return messages, nil
}

// GetStats 获取统计信息
func (s *DefaultCompressionService) GetStats() *CompressionStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 返回副本
	return &CompressionStats{
		TotalCompressions: s.stats.TotalCompressions,
		CacheHits:         s.stats.CacheHits,
		CacheMisses:       s.stats.CacheMisses,
		TotalDuration:     s.stats.TotalDuration,
		AverageRatio:      s.stats.AverageRatio,
		TotalTokensSaved:  s.stats.TotalTokensSaved,
	}
}

// ResetStats 重置统计信息
func (s *DefaultCompressionService) ResetStats() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stats = &CompressionStats{}
}

// ClearCache 清空缓存
func (s *DefaultCompressionService) ClearCache() {
	s.cache.Clear()
}

// compressSimple 简单压缩（基于规则）
func (s *DefaultCompressionService) compressSimple(ctx context.Context, prompt string, opts *CompressOptions) (*CompressResult, error) {
	// 如果设置了底层压缩器，使用它
	if s.compressor != nil {
		compressOpts := &PromptCompressOptions{
			TargetLength:     opts.TargetLength,
			Mode:             string(ModeSimple),
			Level:            int(opts.Level),
			PreserveSections: opts.PreserveSections,
		}
		result, err := s.compressor.Compress(ctx, prompt, compressOpts)
		if err != nil {
			return nil, err
		}
		return &CompressResult{
			Compressed:       result.Compressed,
			OriginalLength:   result.OriginalLength,
			CompressedLength: result.CompressedLength,
			CompressionRatio: result.CompressionRatio,
			TokensSaved:      result.TokensSaved,
		}, nil
	}

	// 降级：使用内置的简单压缩逻辑
	compressed := s.simpleCompressInternal(prompt, opts)

	tokensSaved := 0
	if len(prompt) > 0 {
		tokensSaved = (len(prompt) - len(compressed)) / 4 // 粗略估算
	}

	compressionRatio := 1.0
	if len(prompt) > 0 {
		compressionRatio = float64(len(compressed)) / float64(len(prompt))
	}

	return &CompressResult{
		Compressed:       compressed,
		OriginalLength:   len(prompt),
		CompressedLength: len(compressed),
		CompressionRatio: compressionRatio,
		TokensSaved:      tokensSaved,
	}, nil
}

// simpleCompressInternal 内置简单压缩实现
func (s *DefaultCompressionService) simpleCompressInternal(prompt string, opts *CompressOptions) string {
	// 分析段落
	sections := s.analyzeSections(prompt)

	// 评分和选择段落
	scoredSections := s.scoreSections(sections, opts.PreserveSections)

	// 根据目标长度选择段落
	targetLen := opts.TargetLength
	if targetLen == 0 {
		switch opts.Level {
		case LevelLight:
			targetLen = int(float64(len(prompt)) * 0.65)
		case LevelModerate:
			targetLen = int(float64(len(prompt)) * 0.45)
		case LevelAggressive:
			targetLen = int(float64(len(prompt)) * 0.25)
		default:
			targetLen = int(float64(len(prompt)) * 0.50)
		}
	}

	// 选择段落
	var result []string
	currentLen := 0

	// 首先添加必须保留的段落
	for _, ss := range scoredSections {
		if ss.mustKeep {
			result = append(result, ss.content)
			currentLen += len(ss.content)
		}
	}

	// 然后按评分添加其他段落
	for _, ss := range scoredSections {
		if !ss.mustKeep && currentLen+len(ss.content) <= targetLen {
			result = append(result, ss.content)
			currentLen += len(ss.content)
		}
	}

	// 组合并优化
	compressed := strings.Join(result, "\n\n")
	compressed = s.removeDuplicateEmptyLines(compressed)
	compressed = s.compactFormat(compressed)

	return compressed
}

// scoredSection 带评分的段落
type scoredSection struct {
	content  string
	title    string
	score    float64
	mustKeep bool
}

// analyzeSections 分析段落
func (s *DefaultCompressionService) analyzeSections(prompt string) []string {
	sections := strings.Split(prompt, "\n\n")
	var result []string
	for _, sec := range sections {
		trimmed := strings.TrimSpace(sec)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// scoreSections 评分段落
func (s *DefaultCompressionService) scoreSections(sections []string, preserveSections []string) []scoredSection {
	result := make([]scoredSection, 0, len(sections))

	for i, section := range sections {
		title := s.extractSectionTitle(section)
		mustKeep := s.shouldPreserve(title, preserveSections)
		score := s.calculateSectionScore(section, i, len(sections), mustKeep)

		result = append(result, scoredSection{
			content:  section,
			title:    title,
			score:    score,
			mustKeep: mustKeep,
		})
	}

	// 按评分排序（高分在前）
	for i := range len(result) - 1 {
		for j := i + 1; j < len(result); j++ {
			if result[j].score > result[i].score {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// calculateSectionScore 计算段落重要性评分
func (s *DefaultCompressionService) calculateSectionScore(section string, index, total int, mustKeep bool) float64 {
	if mustKeep {
		return 1.0
	}

	score := 0.0

	// 位置权重
	positionWeight := 1.0 - (float64(index)/float64(total))*0.3
	score += positionWeight * 0.3

	// 关键词权重
	keywords := []string{"IMPORTANT", "重要", "CRITICAL", "关键", "Tools", "工具", "Security", "安全", "NEVER", "ALWAYS", "必须", "禁止", "##", "###"}
	for _, kw := range keywords {
		if strings.Contains(section, kw) {
			score += 0.15
			break
		}
	}

	// 高优先级关键词
	highPriorityKeywords := []string{"Tools Manual", "Security Guidelines", "Permission", "Sandbox", "Error Handling"}
	for _, kw := range highPriorityKeywords {
		if strings.Contains(section, kw) {
			score += 0.25
			break
		}
	}

	// 长度权重
	length := len(section)
	if length > 100 && length < 1000 {
		score += 0.2
	} else if length >= 1000 && length < 2000 {
		score += 0.15
	} else if length >= 50 && length <= 100 {
		score += 0.1
	}

	// 代码块权重
	if strings.Contains(section, "```") || strings.Contains(section, "<example>") {
		score += 0.1
	}

	return score
}

// extractSectionTitle 提取段落标题
func (s *DefaultCompressionService) extractSectionTitle(section string) string {
	lines := strings.Split(section, "\n")
	if len(lines) > 0 {
		firstLine := strings.TrimSpace(lines[0])
		if strings.HasPrefix(firstLine, "##") {
			return strings.TrimSpace(strings.TrimLeft(firstLine, "#"))
		}
		if strings.HasPrefix(firstLine, "#") {
			return strings.TrimSpace(strings.TrimLeft(firstLine, "#"))
		}
	}
	return ""
}

// shouldPreserve 判断是否应该保留该段落
func (s *DefaultCompressionService) shouldPreserve(title string, preserveSections []string) bool {
	if title == "" {
		return false
	}

	lowerTitle := strings.ToLower(title)
	for _, preserve := range preserveSections {
		if strings.Contains(lowerTitle, strings.ToLower(preserve)) {
			return true
		}
	}

	// 默认保留的关键段落
	defaultPreserve := []string{"tools manual", "security", "permission", "工具手册", "安全", "权限"}
	for _, dp := range defaultPreserve {
		if strings.Contains(lowerTitle, dp) {
			return true
		}
	}

	return false
}

// removeDuplicateEmptyLines 移除重复空行
func (s *DefaultCompressionService) removeDuplicateEmptyLines(prompt string) string {
	lines := strings.Split(prompt, "\n")
	var result []string
	prevEmpty := false

	for _, line := range lines {
		isEmpty := strings.TrimSpace(line) == ""
		if isEmpty {
			if !prevEmpty {
				result = append(result, line)
			}
			prevEmpty = true
		} else {
			result = append(result, line)
			prevEmpty = false
		}
	}

	return strings.Join(result, "\n")
}

// compactFormat 紧凑格式
func (s *DefaultCompressionService) compactFormat(prompt string) string {
	lines := strings.Split(prompt, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		result = append(result, trimmed)
	}

	return strings.Join(result, "\n")
}

// compressLLM LLM 驱动的压缩
func (s *DefaultCompressionService) compressLLM(ctx context.Context, prompt string, opts *CompressOptions) (*CompressResult, error) {
	// 这里会调用 prompt_compressor_llm.go 中的 LLM 压缩逻辑
	// 暂时返回占位符
	compressed := prompt // TODO: 实现 LLM 压缩

	return &CompressResult{
		Compressed:       compressed,
		OriginalLength:   len(prompt),
		CompressedLength: len(compressed),
		CompressionRatio: float64(len(compressed)) / float64(len(prompt)),
		TokensSaved:      0,
	}, nil
}

// compressHybrid 混合压缩（先规则后 LLM）
func (s *DefaultCompressionService) compressHybrid(ctx context.Context, prompt string, opts *CompressOptions) (*CompressResult, error) {
	// 第一阶段：简单压缩
	simpleResult, err := s.compressSimple(ctx, prompt, opts)
	if err != nil {
		return nil, err
	}

	// 检查是否达到目标长度
	if opts.TargetLength > 0 && simpleResult.CompressedLength <= opts.TargetLength {
		return simpleResult, nil
	}

	// 第二阶段：LLM 精压缩
	llmResult, err := s.compressLLM(ctx, simpleResult.Compressed, opts)
	if err != nil {
		// LLM 压缩失败，降级到简单压缩结果
		return simpleResult, nil
	}

	return llmResult, nil
}

// generateCacheKey 生成缓存键
func (s *DefaultCompressionService) generateCacheKey(prompt string, opts *CompressOptions) string {
	data := fmt.Sprintf("%s|%s|%d|%d|%v",
		prompt,
		opts.Mode,
		opts.Level,
		opts.TargetLength,
		opts.PreserveSections,
	)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// updateStats 更新统计信息
func (s *DefaultCompressionService) updateStats(result *CompressResult, cacheHit bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stats.TotalCompressions++
	s.stats.TotalDuration += result.Duration
	s.stats.TotalTokensSaved += result.TokensSaved

	if cacheHit {
		s.stats.CacheHits++
	} else {
		s.stats.CacheMisses++
	}

	// 更新平均压缩率
	if s.stats.TotalCompressions > 0 {
		totalRatio := s.stats.AverageRatio*float64(s.stats.TotalCompressions-1) + result.CompressionRatio
		s.stats.AverageRatio = totalRatio / float64(s.stats.TotalCompressions)
	}
}

// CompressionCache 压缩缓存
type CompressionCache struct {
	cache   map[string]*CompressResult
	maxSize int
	mu      sync.RWMutex
}

// NewCompressionCache 创建压缩缓存
func NewCompressionCache(maxSize int) *CompressionCache {
	return &CompressionCache{
		cache:   make(map[string]*CompressResult),
		maxSize: maxSize,
	}
}

// Get 获取缓存
func (c *CompressionCache) Get(key string) *CompressResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if result, ok := c.cache[key]; ok {
		// 返回副本
		return &CompressResult{
			Compressed:       result.Compressed,
			OriginalLength:   result.OriginalLength,
			CompressedLength: result.CompressedLength,
			CompressionRatio: result.CompressionRatio,
			TokensSaved:      result.TokensSaved,
			Duration:         result.Duration,
			Mode:             result.Mode,
		}
	}

	return nil
}

// Set 设置缓存
func (c *CompressionCache) Set(key string, result *CompressResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 简单的 LRU：如果超过最大大小，清空缓存
	if len(c.cache) >= c.maxSize {
		c.cache = make(map[string]*CompressResult)
	}

	c.cache[key] = result
}

// Clear 清空缓存
func (c *CompressionCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*CompressResult)
}

// Size 获取缓存大小
func (c *CompressionCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.cache)
}
