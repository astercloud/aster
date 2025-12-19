package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/types"
)

// LLMPromptCompressor LLM 驱动的 Prompt 压缩器
type LLMPromptCompressor struct {
	provider provider.Provider
	model    string
	language string
}

// NewLLMPromptCompressor 创建 LLM Prompt 压缩器
func NewLLMPromptCompressor(prov provider.Provider, model string, language string) *LLMPromptCompressor {
	if model == "" {
		model = "deepseek-chat" // 默认使用 DeepSeek
	}
	if language == "" {
		language = "zh"
	}

	return &LLMPromptCompressor{
		provider: prov,
		model:    model,
		language: language,
	}
}

// Compress 压缩 Prompt
func (c *LLMPromptCompressor) Compress(ctx context.Context, prompt string, targetLength int, preserveSections []string, level int) (string, error) {
	// 构建压缩提示词
	systemPrompt := c.buildCompressionPrompt(targetLength, preserveSections, level)

	// 调用 LLM
	messages := []types.Message{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	opts := &provider.StreamOptions{
		System:      systemPrompt,
		Temperature: 0.3,                  // 低温度，保证稳定性
		MaxTokens:   targetLength * 4 / 3, // 预留一些空间
	}

	// 使用 Complete API（非流式）
	response, err := c.provider.Complete(ctx, messages, opts)
	if err != nil {
		return "", fmt.Errorf("LLM compression failed: %w", err)
	}

	compressed := response.Message.Content

	// 验证压缩结果
	if err := c.validateCompression(prompt, compressed, preserveSections); err != nil {
		return "", fmt.Errorf("compression validation failed: %w", err)
	}

	return compressed, nil
}

// CompressSection 压缩单个段落
func (c *LLMPromptCompressor) CompressSection(ctx context.Context, section string, targetLength int) (string, error) {
	systemPrompt := c.buildSectionCompressionPrompt(targetLength)

	messages := []types.Message{
		{
			Role:    "user",
			Content: section,
		},
	}

	opts := &provider.StreamOptions{
		System:      systemPrompt,
		Temperature: 0.3,
		MaxTokens:   targetLength * 4 / 3,
	}

	response, err := c.provider.Complete(ctx, messages, opts)
	if err != nil {
		return "", fmt.Errorf("section compression failed: %w", err)
	}

	return response.Message.Content, nil
}

// buildCompressionPrompt 构建压缩提示词
func (c *LLMPromptCompressor) buildCompressionPrompt(targetLength int, preserveSections []string, level int) string {
	if c.language == "zh" {
		return c.buildChinesePrompt(targetLength, preserveSections, level)
	}
	return c.buildEnglishPrompt(targetLength, preserveSections, level)
}

// buildChinesePrompt 构建中文压缩提示词
func (c *LLMPromptCompressor) buildChinesePrompt(targetLength int, preserveSections []string, level int) string {
	var prompt strings.Builder

	prompt.WriteString("你是一个专业的文本压缩助手。请压缩用户提供的 System Prompt，要求：\n\n")

	// 根据压缩级别调整要求
	switch level {
	case 1: // 轻度压缩
		prompt.WriteString("**压缩级别：轻度（保留 60-70% 内容）**\n\n")
		prompt.WriteString("1. 保留所有关键信息和核心指令\n")
		prompt.WriteString("2. 移除明显的冗余和重复内容\n")
		prompt.WriteString("3. 使用稍微简洁的表达\n")
		prompt.WriteString("4. 保持原有的段落结构和格式\n")
	case 2: // 中度压缩
		prompt.WriteString("**压缩级别：中度（保留 40-50% 内容）**\n\n")
		prompt.WriteString("1. 保留核心信息和关键指令\n")
		prompt.WriteString("2. 移除冗余、重复和次要内容\n")
		prompt.WriteString("3. 使用更简洁的表达方式\n")
		prompt.WriteString("4. 可以合并相似的段落\n")
	case 3: // 激进压缩
		prompt.WriteString("**压缩级别：激进（保留 20-30% 内容）**\n\n")
		prompt.WriteString("1. 只保留最核心的信息和指令\n")
		prompt.WriteString("2. 大幅简化表达，去除所有冗余\n")
		prompt.WriteString("3. 合并和精简段落\n")
		prompt.WriteString("4. 使用最简洁的语言\n")
	}

	prompt.WriteString("\n**通用要求：**\n")
	prompt.WriteString("- 保持语义准确性，不改变原意\n")
	prompt.WriteString("- 保留重要的格式标记（如 ##, -, IMPORTANT 等）\n")
	prompt.WriteString("- 输出压缩后的内容，不要添加任何解释或说明\n")

	if targetLength > 0 {
		prompt.WriteString(fmt.Sprintf("- 目标长度：约 %d 字符\n", targetLength))
	}

	if len(preserveSections) > 0 {
		prompt.WriteString("\n**必须完整保留的段落：**\n")
		for _, section := range preserveSections {
			prompt.WriteString(fmt.Sprintf("- %s\n", section))
		}
	}

	prompt.WriteString("\n请直接输出压缩后的 System Prompt：")

	return prompt.String()
}

// buildEnglishPrompt 构建英文压缩提示词
func (c *LLMPromptCompressor) buildEnglishPrompt(targetLength int, preserveSections []string, level int) string {
	var prompt strings.Builder

	prompt.WriteString("You are a professional text compression assistant. Please compress the System Prompt provided by the user with the following requirements:\n\n")

	// 根据压缩级别调整要求
	switch level {
	case 1: // Light compression
		prompt.WriteString("**Compression Level: Light (retain 60-70% content)**\n\n")
		prompt.WriteString("1. Retain all key information and core instructions\n")
		prompt.WriteString("2. Remove obvious redundancy and repetition\n")
		prompt.WriteString("3. Use slightly more concise expressions\n")
		prompt.WriteString("4. Maintain original paragraph structure and format\n")
	case 2: // Moderate compression
		prompt.WriteString("**Compression Level: Moderate (retain 40-50% content)**\n\n")
		prompt.WriteString("1. Retain core information and key instructions\n")
		prompt.WriteString("2. Remove redundancy, repetition, and secondary content\n")
		prompt.WriteString("3. Use more concise expressions\n")
		prompt.WriteString("4. Can merge similar paragraphs\n")
	case 3: // Aggressive compression
		prompt.WriteString("**Compression Level: Aggressive (retain 20-30% content)**\n\n")
		prompt.WriteString("1. Only retain the most core information and instructions\n")
		prompt.WriteString("2. Significantly simplify expressions, remove all redundancy\n")
		prompt.WriteString("3. Merge and streamline paragraphs\n")
		prompt.WriteString("4. Use the most concise language\n")
	}

	prompt.WriteString("\n**General Requirements:**\n")
	prompt.WriteString("- Maintain semantic accuracy, do not change the original meaning\n")
	prompt.WriteString("- Preserve important format markers (such as ##, -, IMPORTANT, etc.)\n")
	prompt.WriteString("- Output the compressed content directly without any explanation\n")

	if targetLength > 0 {
		prompt.WriteString(fmt.Sprintf("- Target length: approximately %d characters\n", targetLength))
	}

	if len(preserveSections) > 0 {
		prompt.WriteString("\n**Sections that MUST be preserved completely:**\n")
		for _, section := range preserveSections {
			prompt.WriteString(fmt.Sprintf("- %s\n", section))
		}
	}

	prompt.WriteString("\nPlease output the compressed System Prompt directly:")

	return prompt.String()
}

// buildSectionCompressionPrompt 构建段落压缩提示词
func (c *LLMPromptCompressor) buildSectionCompressionPrompt(targetLength int) string {
	if c.language == "zh" {
		return fmt.Sprintf(`你是一个文本压缩助手。请压缩用户提供的段落，要求：

1. 保留核心信息
2. 使用简洁的表达
3. 不改变原意
4. 目标长度：约 %d 字符

请直接输出压缩后的段落：`, targetLength)
	}

	return fmt.Sprintf(`You are a text compression assistant. Please compress the paragraph provided by the user with the following requirements:

1. Retain core information
2. Use concise expressions
3. Do not change the original meaning
4. Target length: approximately %d characters

Please output the compressed paragraph directly:`, targetLength)
}

// validateCompression 验证压缩结果
func (c *LLMPromptCompressor) validateCompression(original, compressed string, preserveSections []string) error {
	// 检查压缩结果不为空
	if strings.TrimSpace(compressed) == "" {
		return errors.New("compressed result is empty")
	}

	// 检查压缩结果不能比原始内容更长
	if len(compressed) > len(original) {
		return errors.New("compressed result is longer than original")
	}

	// 检查必须保留的段落是否存在
	for _, section := range preserveSections {
		if !strings.Contains(compressed, section) {
			// 尝试模糊匹配（去除空格和大小写）
			normalizedCompressed := strings.ToLower(strings.ReplaceAll(compressed, " ", ""))
			normalizedSection := strings.ToLower(strings.ReplaceAll(section, " ", ""))

			if !strings.Contains(normalizedCompressed, normalizedSection) {
				return fmt.Errorf("required section not found in compressed result: %s", section)
			}
		}
	}

	return nil
}

// EstimateCompressionRatio 估算压缩率
func (c *LLMPromptCompressor) EstimateCompressionRatio(level int) float64 {
	switch level {
	case 1: // 轻度压缩
		return 0.65 // 保留 65%
	case 2: // 中度压缩
		return 0.45 // 保留 45%
	case 3: // 激进压缩
		return 0.25 // 保留 25%
	default:
		return 0.50
	}
}
