package agent

import (
	"context"
	"strings"
	"testing"
)

// TestEnhancedPromptCompressor_SimpleCompression 测试简单压缩模式
func TestEnhancedPromptCompressor_SimpleCompression(t *testing.T) {
	// 创建不需要 Provider 的压缩器（仅测试简单压缩）
	compressor := &EnhancedPromptCompressor{
		language: "zh",
	}

	// 创建测试 prompt
	testPrompt := generateLongTestPrompt()

	opts := &CompressOptions{
		Mode:             CompressionModeSimple,
		Level:            CompressionLevelModerate,
		PreserveSections: []string{"Tools Manual", "Security"},
	}

	result, err := compressor.Compress(context.Background(), testPrompt, opts)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// 验证压缩结果
	if result.CompressedLength >= result.OriginalLength {
		t.Errorf("Expected compression, got original=%d, compressed=%d",
			result.OriginalLength, result.CompressedLength)
	}

	// 验证压缩率
	if result.CompressionRatio >= 1.0 {
		t.Errorf("Expected compression ratio < 1.0, got %f", result.CompressionRatio)
	}

	// 验证保留段落
	if !strings.Contains(result.Compressed, "Tools Manual") {
		t.Error("Expected 'Tools Manual' section to be preserved")
	}
	if !strings.Contains(result.Compressed, "Security") {
		t.Error("Expected 'Security' section to be preserved")
	}

	t.Logf("Compression result: original=%d, compressed=%d, ratio=%.2f%%",
		result.OriginalLength, result.CompressedLength, result.CompressionRatio*100)
}

// TestEnhancedPromptCompressor_CompressionLevels 测试不同压缩级别
func TestEnhancedPromptCompressor_CompressionLevels(t *testing.T) {
	compressor := &EnhancedPromptCompressor{
		language: "zh",
	}

	testPrompt := generateLongTestPrompt()

	levels := []struct {
		name  string
		level CompressionLevel
	}{
		{"Light", CompressionLevelLight},
		{"Moderate", CompressionLevelModerate},
		{"Aggressive", CompressionLevelAggressive},
	}

	prevRatio := 1.0

	for _, tc := range levels {
		t.Run(tc.name, func(t *testing.T) {
			opts := &CompressOptions{
				Mode:  CompressionModeSimple,
				Level: tc.level,
			}

			result, err := compressor.Compress(context.Background(), testPrompt, opts)
			if err != nil {
				t.Fatalf("Compress failed: %v", err)
			}

			t.Logf("%s: ratio=%.2f%%, length=%d",
				tc.name, result.CompressionRatio*100, result.CompressedLength)

			// 验证压缩程度递增
			if result.CompressionRatio > prevRatio {
				t.Logf("Warning: %s compression ratio (%.2f) > previous (%.2f)",
					tc.name, result.CompressionRatio, prevRatio)
			}
			prevRatio = result.CompressionRatio
		})
	}
}

// TestEnhancedPromptCompressor_SectionScoring 测试段落评分
func TestEnhancedPromptCompressor_SectionScoring(t *testing.T) {
	compressor := &EnhancedPromptCompressor{
		language: "zh",
	}

	sections := []string{
		"## Tools Manual\nThis section describes available tools.",
		"## Regular Section\nSome regular content here.",
		"## Security Guidelines\nIMPORTANT: Never expose credentials.",
		"Some text without header",
	}

	scored := compressor.scoreSections(sections, []string{"Tools Manual"})

	// 验证评分数量
	if len(scored) != len(sections) {
		t.Errorf("Expected %d scored sections, got %d", len(sections), len(scored))
	}

	// 验证必须保留的段落
	var toolsManualScore float64
	for _, s := range scored {
		if strings.Contains(s.Content, "Tools Manual") {
			toolsManualScore = s.Score
			if !s.MustKeep {
				t.Error("Expected 'Tools Manual' section to be marked as MustKeep")
			}
		}
	}

	// Tools Manual 应该有最高评分
	if toolsManualScore < 1.0 {
		t.Errorf("Expected 'Tools Manual' score = 1.0, got %f", toolsManualScore)
	}
}

// TestEnhancedPromptCompressor_PreserveSections 测试保留段落
func TestEnhancedPromptCompressor_PreserveSections(t *testing.T) {
	compressor := &EnhancedPromptCompressor{
		language: "en",
	}

	testCases := []struct {
		title    string
		preserve []string
		expected bool
	}{
		{"Tools Manual", []string{"Tools Manual"}, true},
		{"Security Guidelines", []string{"Security"}, true},
		{"Permission Settings", []string{"Permission"}, true},
		{"Random Section", []string{"Tools Manual"}, false},
		{"工具手册", []string{}, true}, // 默认保留
		{"安全指南", []string{}, true}, // 默认保留
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			result := compressor.shouldPreserve(tc.title, tc.preserve)
			if result != tc.expected {
				t.Errorf("shouldPreserve(%q, %v) = %v, expected %v",
					tc.title, tc.preserve, result, tc.expected)
			}
		})
	}
}

// TestEnhancedPromptCompressor_EmptyPrompt 测试空 prompt
func TestEnhancedPromptCompressor_EmptyPrompt(t *testing.T) {
	compressor := &EnhancedPromptCompressor{
		language: "zh",
	}

	result, err := compressor.Compress(context.Background(), "", nil)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	if result.Compressed != "" {
		t.Errorf("Expected empty result, got %q", result.Compressed)
	}
}

// TestCompressOptions_Defaults 测试默认选项
func TestCompressOptions_Defaults(t *testing.T) {
	compressor := &EnhancedPromptCompressor{
		language: "zh",
	}

	testPrompt := "Some test content\n\nAnother section"

	// 使用 nil 选项
	result, err := compressor.Compress(context.Background(), testPrompt, nil)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// 验证使用了默认模式
	if result.Mode != string(CompressionModeHybrid) {
		t.Errorf("Expected default mode %s, got %s", CompressionModeHybrid, result.Mode)
	}
}

// TestScoredSection_Sorting 测试评分排序
func TestScoredSection_Sorting(t *testing.T) {
	compressor := &EnhancedPromptCompressor{
		language: "zh",
	}

	sections := []string{
		"Regular content",
		"## IMPORTANT Security\nCritical section",
		"## Tools Manual\nTools description",
		"Some other text",
	}

	scored := compressor.scoreSections(sections, []string{"Tools Manual"})

	// 验证排序（高分在前）
	for i := 0; i < len(scored)-1; i++ {
		if scored[i].Score < scored[i+1].Score {
			t.Errorf("Sections not sorted by score: %f < %f",
				scored[i].Score, scored[i+1].Score)
		}
	}
}

// generateLongTestPrompt 生成测试用的长 prompt
func generateLongTestPrompt() string {
	var sb strings.Builder

	sb.WriteString("# System Prompt\n\n")
	sb.WriteString("You are a helpful AI assistant.\n\n")

	sb.WriteString("## Tools Manual\n")
	sb.WriteString("This section describes the available tools:\n")
	sb.WriteString("- ReadFile: Read contents of a file\n")
	sb.WriteString("- WriteFile: Write contents to a file\n")
	sb.WriteString("- ExecuteCommand: Run a shell command\n\n")

	sb.WriteString("## Security Guidelines\n")
	sb.WriteString("IMPORTANT: Follow these security rules:\n")
	sb.WriteString("- Never expose API keys or credentials\n")
	sb.WriteString("- Always validate user input\n")
	sb.WriteString("- Use secure connections\n\n")

	sb.WriteString("## General Instructions\n")
	sb.WriteString("When responding to user queries:\n")
	sb.WriteString("1. Be helpful and accurate\n")
	sb.WriteString("2. Provide clear explanations\n")
	sb.WriteString("3. Use examples when helpful\n\n")

	// 添加一些填充内容
	for i := range 10 {
		sb.WriteString("## Section ")
		sb.WriteRune(rune('A' + i))
		sb.WriteString("\n")
		sb.WriteString("This is some filler content for testing purposes. ")
		sb.WriteString("It helps verify that compression works correctly ")
		sb.WriteString("when dealing with longer prompts.\n\n")
	}

	sb.WriteString("## Closing Notes\n")
	sb.WriteString("Remember to always be helpful and follow the guidelines above.\n")

	return sb.String()
}

// TestRemoveDuplicateEmptyLines 测试移除重复空行
func TestRemoveDuplicateEmptyLines(t *testing.T) {
	compressor := &EnhancedPromptCompressor{}

	input := "Line 1\n\n\n\nLine 2\n\n\nLine 3"
	expected := "Line 1\n\nLine 2\n\nLine 3"

	result := compressor.removeDuplicateEmptyLines(input)
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestCompactFormat 测试紧凑格式
func TestCompactFormat(t *testing.T) {
	compressor := &EnhancedPromptCompressor{}

	input := "Line 1   \n  Line 2\t  \nLine 3"
	expected := "Line 1\n  Line 2\nLine 3"

	result := compressor.compactFormat(input)
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestExtractSectionTitle 测试提取段落标题
func TestExtractSectionTitle(t *testing.T) {
	compressor := &EnhancedPromptCompressor{}

	testCases := []struct {
		section  string
		expected string
	}{
		{"## Tools Manual\nContent", "Tools Manual"},
		{"### Sub Section\nContent", "Sub Section"},
		{"# Main Title\nContent", "Main Title"},
		{"No header here\nContent", ""},
		{"", ""},
	}

	for _, tc := range testCases {
		result := compressor.extractSectionTitle(tc.section)
		if result != tc.expected {
			t.Errorf("extractSectionTitle(%q) = %q, expected %q",
				tc.section, result, tc.expected)
		}
	}
}
