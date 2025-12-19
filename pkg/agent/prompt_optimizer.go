package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	astContext "github.com/astercloud/aster/pkg/context"
	"github.com/astercloud/aster/pkg/provider"
)

// CompressionMode 压缩模式
type CompressionMode string

const (
	CompressionModeSimple CompressionMode = "simple" // 基于规则的快速压缩
	CompressionModeLLM    CompressionMode = "llm"    // LLM 驱动的智能压缩
	CompressionModeHybrid CompressionMode = "hybrid" // 混合模式（先规则后 LLM）
)

// CompressionLevel 压缩级别
type CompressionLevel int

const (
	CompressionLevelLight      CompressionLevel = 1 // 轻度压缩（保留 60-70%）
	CompressionLevelModerate   CompressionLevel = 2 // 中度压缩（保留 40-50%）
	CompressionLevelAggressive CompressionLevel = 3 // 激进压缩（保留 20-30%）
)

// PromptOptimizer Prompt 优化器
type PromptOptimizer struct {
	MaxLength        int  // 最大长度（字符数）
	RemoveDuplicates bool // 移除重复内容
	CompactFormat    bool // 紧凑格式
}

// EnhancedPromptCompressor 增强的 Prompt 压缩器
// 集成 Token 计数、段落分析和多模式压缩
type EnhancedPromptCompressor struct {
	tokenCounter  astContext.TokenCounter
	llmCompressor *LLMPromptCompressor
	provider      provider.Provider
	language      string
}

// NewEnhancedPromptCompressor 创建增强压缩器
func NewEnhancedPromptCompressor(prov provider.Provider, language string) *EnhancedPromptCompressor {
	if language == "" {
		language = "zh"
	}

	// 使用 DeepSeek Token 计数器
	tokenCounter := astContext.NewSimpleTokenCounter(astContext.DeepSeekChatConfig)

	return &EnhancedPromptCompressor{
		tokenCounter:  tokenCounter,
		llmCompressor: NewLLMPromptCompressor(prov, "deepseek-chat", language),
		provider:      prov,
		language:      language,
	}
}

// CompressOptions 压缩选项
type CompressOptions struct {
	TargetLength     int              // 目标长度（字符数）
	TargetTokens     int              // 目标 Token 数
	Mode             CompressionMode  // 压缩模式
	Level            CompressionLevel // 压缩级别
	PreserveSections []string         // 必须保留的段落标题
}

// CompressResult 压缩结果
type CompressResult struct {
	Compressed       string  // 压缩后的内容
	OriginalLength   int     // 原始长度
	CompressedLength int     // 压缩后长度
	OriginalTokens   int     // 原始 Token 数
	CompressedTokens int     // 压缩后 Token 数
	CompressionRatio float64 // 压缩率
	TokensSaved      int     // 节省的 Token 数
	Mode             string  // 使用的模式
}

// Compress 压缩 Prompt
func (c *EnhancedPromptCompressor) Compress(ctx context.Context, prompt string, opts *CompressOptions) (*CompressResult, error) {
	if opts == nil {
		opts = &CompressOptions{
			Mode:  CompressionModeHybrid,
			Level: CompressionLevelModerate,
		}
	}

	// 计算原始 Token 数
	var originalTokens int
	var err error
	if c.tokenCounter != nil {
		originalTokens, err = c.tokenCounter.Count(ctx, prompt)
		if err != nil {
			originalTokens = len(prompt) / 4 // 降级估算
		}
	} else {
		originalTokens = len(prompt) / 4 // 无计数器时降级估算
	}

	var compressed string

	switch opts.Mode {
	case CompressionModeSimple:
		compressed = c.compressSimple(prompt, opts)
	case CompressionModeLLM:
		compressed, err = c.compressLLM(ctx, prompt, opts)
		if err != nil {
			// LLM 压缩失败，降级到简单压缩
			compressed = c.compressSimple(prompt, opts)
		}
	case CompressionModeHybrid:
		compressed, err = c.compressHybrid(ctx, prompt, opts)
		if err != nil {
			compressed = c.compressSimple(prompt, opts)
		}
	default:
		compressed = c.compressSimple(prompt, opts)
	}

	// 计算压缩后 Token 数
	var compressedTokens int
	if c.tokenCounter != nil {
		compressedTokens, _ = c.tokenCounter.Count(ctx, compressed)
	} else {
		compressedTokens = len(compressed) / 4
	}

	result := &CompressResult{
		Compressed:       compressed,
		OriginalLength:   len(prompt),
		CompressedLength: len(compressed),
		OriginalTokens:   originalTokens,
		CompressedTokens: compressedTokens,
		TokensSaved:      originalTokens - compressedTokens,
		Mode:             string(opts.Mode),
	}

	if len(prompt) > 0 {
		result.CompressionRatio = float64(len(compressed)) / float64(len(prompt))
	}

	return result, nil
}

// compressSimple 基于规则的简单压缩
func (c *EnhancedPromptCompressor) compressSimple(prompt string, opts *CompressOptions) string {
	// 分析段落
	sections := c.analyzeSections(prompt)

	// 评分和排序
	scoredSections := c.scoreSections(sections, opts.PreserveSections)

	// 根据目标长度选择段落
	targetLen := opts.TargetLength
	if targetLen == 0 {
		// 根据压缩级别计算目标长度
		switch opts.Level {
		case CompressionLevelLight:
			targetLen = int(float64(len(prompt)) * 0.65)
		case CompressionLevelModerate:
			targetLen = int(float64(len(prompt)) * 0.45)
		case CompressionLevelAggressive:
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
		if ss.MustKeep {
			result = append(result, ss.Content)
			currentLen += len(ss.Content)
		}
	}

	// 然后按评分添加其他段落
	for _, ss := range scoredSections {
		if !ss.MustKeep && currentLen+len(ss.Content) <= targetLen {
			result = append(result, ss.Content)
			currentLen += len(ss.Content)
		}
	}

	// 应用基本优化
	compressed := strings.Join(result, "\n\n")
	compressed = c.removeDuplicateEmptyLines(compressed)
	compressed = c.compactFormat(compressed)

	return compressed
}

// compressLLM LLM 驱动的压缩
func (c *EnhancedPromptCompressor) compressLLM(ctx context.Context, prompt string, opts *CompressOptions) (string, error) {
	if c.llmCompressor == nil {
		return prompt, errors.New("LLM compressor not available")
	}

	targetLen := opts.TargetLength
	if targetLen == 0 {
		switch opts.Level {
		case CompressionLevelLight:
			targetLen = int(float64(len(prompt)) * 0.65)
		case CompressionLevelModerate:
			targetLen = int(float64(len(prompt)) * 0.45)
		case CompressionLevelAggressive:
			targetLen = int(float64(len(prompt)) * 0.25)
		default:
			targetLen = int(float64(len(prompt)) * 0.50)
		}
	}

	return c.llmCompressor.Compress(ctx, prompt, targetLen, opts.PreserveSections, int(opts.Level))
}

// compressHybrid 混合压缩（先规则后 LLM）
func (c *EnhancedPromptCompressor) compressHybrid(ctx context.Context, prompt string, opts *CompressOptions) (string, error) {
	// 第一阶段：规则压缩
	simpleCompressed := c.compressSimple(prompt, &CompressOptions{
		TargetLength:     int(float64(len(prompt)) * 0.7), // 先压缩到 70%
		Level:            CompressionLevelLight,
		PreserveSections: opts.PreserveSections,
	})

	// 检查是否达到目标
	if opts.TargetLength > 0 && len(simpleCompressed) <= opts.TargetLength {
		return simpleCompressed, nil
	}

	// 第二阶段：LLM 精压缩
	return c.compressLLM(ctx, simpleCompressed, opts)
}

// ScoredSection 带评分的段落
type ScoredSection struct {
	Content  string
	Title    string
	Score    float64
	MustKeep bool
}

// analyzeSections 分析段落
func (c *EnhancedPromptCompressor) analyzeSections(prompt string) []string {
	// 按双换行分割
	sections := strings.Split(prompt, "\n\n")

	// 过滤空段落
	var result []string
	for _, s := range sections {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// scoreSections 评分段落
func (c *EnhancedPromptCompressor) scoreSections(sections []string, preserveSections []string) []ScoredSection {
	result := make([]ScoredSection, 0, len(sections))

	for i, section := range sections {
		title := c.extractSectionTitle(section)
		mustKeep := c.shouldPreserve(title, preserveSections)
		score := c.calculateSectionScore(section, i, len(sections), mustKeep)

		result = append(result, ScoredSection{
			Content:  section,
			Title:    title,
			Score:    score,
			MustKeep: mustKeep,
		})
	}

	// 按评分排序（高分在前）
	for i := range len(result) - 1 {
		for j := i + 1; j < len(result); j++ {
			if result[j].Score > result[i].Score {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// calculateSectionScore 计算段落重要性评分
func (c *EnhancedPromptCompressor) calculateSectionScore(section string, index, total int, mustKeep bool) float64 {
	if mustKeep {
		return 1.0 // 必须保留的段落得最高分
	}

	score := 0.0

	// 1. 位置权重（开头段落更重要）
	positionWeight := 1.0 - (float64(index)/float64(total))*0.3
	score += positionWeight * 0.3

	// 2. 关键词权重
	keywords := []string{
		"IMPORTANT", "重要", "CRITICAL", "关键",
		"Tools", "工具", "Security", "安全",
		"NEVER", "ALWAYS", "必须", "禁止",
		"##", "###",
	}
	for _, kw := range keywords {
		if strings.Contains(section, kw) {
			score += 0.15
			break
		}
	}

	// 3. 高优先级关键词
	highPriorityKeywords := []string{
		"Tools Manual", "Security Guidelines",
		"Permission", "Sandbox", "Error Handling",
	}
	for _, kw := range highPriorityKeywords {
		if strings.Contains(section, kw) {
			score += 0.25
			break
		}
	}

	// 4. 长度权重（适中长度更重要）
	length := len(section)
	if length > 100 && length < 1000 {
		score += 0.2
	} else if length >= 1000 && length < 2000 {
		score += 0.15
	} else if length >= 50 && length <= 100 {
		score += 0.1
	}

	// 5. 代码块权重（包含代码示例的段落更重要）
	if strings.Contains(section, "```") || strings.Contains(section, "<example>") {
		score += 0.1
	}

	return score
}

// extractSectionTitle 提取段落标题
func (c *EnhancedPromptCompressor) extractSectionTitle(section string) string {
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
func (c *EnhancedPromptCompressor) shouldPreserve(title string, preserveSections []string) bool {
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
	defaultPreserve := []string{
		"tools manual", "security", "permission",
		"工具手册", "安全", "权限",
	}
	for _, dp := range defaultPreserve {
		if strings.Contains(lowerTitle, dp) {
			return true
		}
	}

	return false
}

// removeDuplicateEmptyLines 移除重复空行
func (c *EnhancedPromptCompressor) removeDuplicateEmptyLines(prompt string) string {
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
func (c *EnhancedPromptCompressor) compactFormat(prompt string) string {
	lines := strings.Split(prompt, "\n")
	var result []string

	for _, line := range lines {
		// 保留缩进，但移除行尾空格
		trimmed := strings.TrimRight(line, " \t")
		result = append(result, trimmed)
	}

	return strings.Join(result, "\n")
}

// EstimateTokens 估算 Token 数
func (c *EnhancedPromptCompressor) EstimateTokens(ctx context.Context, text string) (int, error) {
	return c.tokenCounter.Count(ctx, text)
}

// Optimize 优化 System Prompt
func (po *PromptOptimizer) Optimize(prompt string) string {
	if po == nil {
		return prompt
	}

	result := prompt

	// 移除重复的空行
	if po.RemoveDuplicates {
		result = po.removeDuplicateEmptyLines(result)
	}

	// 紧凑格式
	if po.CompactFormat {
		result = po.compactFormat(result)
	}

	// 截断到最大长度
	if po.MaxLength > 0 && len(result) > po.MaxLength {
		result = po.truncate(result, po.MaxLength)
	}

	return result
}

// removeDuplicateEmptyLines 移除重复的空行
func (po *PromptOptimizer) removeDuplicateEmptyLines(prompt string) string {
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

// compactFormat 紧凑格式（移除多余空格）
func (po *PromptOptimizer) compactFormat(prompt string) string {
	lines := strings.Split(prompt, "\n")
	var result []string

	for _, line := range lines {
		// 保留缩进，但移除行尾空格
		trimmed := strings.TrimRight(line, " \t")
		result = append(result, trimmed)
	}

	return strings.Join(result, "\n")
}

// truncate 截断到指定长度，保留重要部分
func (po *PromptOptimizer) truncate(prompt string, maxLength int) string {
	if len(prompt) <= maxLength {
		return prompt
	}

	// 尝试在段落边界截断
	sections := strings.Split(prompt, "\n\n")

	var result []string
	currentLength := 0

	for _, section := range sections {
		sectionLen := len(section) + 2 // +2 for "\n\n"

		if currentLength+sectionLen > maxLength {
			// 添加截断提示
			if currentLength < maxLength-100 {
				result = append(result, "\n\n[... System Prompt truncated due to length limit ...]")
			}
			break
		}

		result = append(result, section)
		currentLength += sectionLen
	}

	return strings.Join(result, "\n\n")
}

// PromptStats Prompt 统计信息
type PromptStats struct {
	TotalLength     int
	LineCount       int
	SectionCount    int
	ModuleCount     int
	EstimatedTokens int
}

// AnalyzePrompt 分析 Prompt 统计信息
func AnalyzePrompt(prompt string) *PromptStats {
	stats := &PromptStats{
		TotalLength: len(prompt),
		LineCount:   strings.Count(prompt, "\n") + 1,
	}

	// 统计段落数（以 ## 开头的行）
	lines := strings.SplitSeq(prompt, "\n")
	for line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "##") {
			stats.SectionCount++
		}
	}

	// 估算 token 数（粗略估计：1 token ≈ 4 字符）
	stats.EstimatedTokens = stats.TotalLength / 4

	return stats
}

// FormatStats 格式化统计信息
func FormatStats(stats *PromptStats) string {
	return fmt.Sprintf(
		"Prompt Stats: %d chars, %d lines, %d sections, ~%d tokens",
		stats.TotalLength,
		stats.LineCount,
		stats.SectionCount,
		stats.EstimatedTokens,
	)
}

// PromptCompressor Prompt 压缩器（使用 LLM 进行智能压缩）
type PromptCompressor struct {
	TargetLength     int
	PreserveSections []string // 需要保留的段落标题
}

// Compress 压缩 Prompt（简化版，实际可以使用 LLM）
func (pc *PromptCompressor) Compress(prompt string) string {
	if pc == nil {
		return prompt
	}

	sections := pc.splitSections(prompt)
	var result []string

	currentLength := 0
	targetLength := pc.TargetLength

	// 首先添加需要保留的段落
	for _, section := range sections {
		title := pc.extractSectionTitle(section)

		if pc.shouldPreserve(title) {
			result = append(result, section)
			currentLength += len(section)
		}
	}

	// 然后添加其他段落，直到达到目标长度
	for _, section := range sections {
		title := pc.extractSectionTitle(section)

		if !pc.shouldPreserve(title) && currentLength+len(section) < targetLength {
			result = append(result, section)
			currentLength += len(section)
		}
	}

	return strings.Join(result, "\n\n")
}

// splitSections 分割段落
func (pc *PromptCompressor) splitSections(prompt string) []string {
	return strings.Split(prompt, "\n\n")
}

// extractSectionTitle 提取段落标题
func (pc *PromptCompressor) extractSectionTitle(section string) string {
	lines := strings.Split(section, "\n")
	if len(lines) > 0 {
		firstLine := strings.TrimSpace(lines[0])
		if after, found := strings.CutPrefix(firstLine, "##"); found {
			return strings.TrimSpace(after)
		}
	}
	return ""
}

// shouldPreserve 判断是否应该保留该段落
func (pc *PromptCompressor) shouldPreserve(title string) bool {
	for _, preserve := range pc.PreserveSections {
		if strings.Contains(strings.ToLower(title), strings.ToLower(preserve)) {
			return true
		}
	}
	return false
}
