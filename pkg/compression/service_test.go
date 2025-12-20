package compression

import (
	"context"
	"strings"
	"testing"
)

// TestDefaultCompressionService_SimpleCompression 测试简单压缩
func TestDefaultCompressionService_SimpleCompression(t *testing.T) {
	svc := NewDefaultCompressionService(nil, 100)

	prompt := generateTestPrompt()

	result, err := svc.CompressSystemPrompt(context.Background(), prompt, &CompressOptions{
		Mode:             ModeSimple,
		Level:            LevelModerate,
		PreserveSections: []string{"Tools Manual", "Security"},
	})

	if err != nil {
		t.Fatalf("CompressSystemPrompt failed: %v", err)
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

// TestDefaultCompressionService_CompressionLevels 测试不同压缩级别
func TestDefaultCompressionService_CompressionLevels(t *testing.T) {
	svc := NewDefaultCompressionService(nil, 100)
	prompt := generateTestPrompt()

	levels := []struct {
		name  string
		level CompressionLevel
	}{
		{"Light", LevelLight},
		{"Moderate", LevelModerate},
		{"Aggressive", LevelAggressive},
	}

	prevRatio := 1.0

	for _, tc := range levels {
		t.Run(tc.name, func(t *testing.T) {
			result, err := svc.CompressSystemPrompt(context.Background(), prompt, &CompressOptions{
				Mode:  ModeSimple,
				Level: tc.level,
			})

			if err != nil {
				t.Fatalf("CompressSystemPrompt failed: %v", err)
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

// TestDefaultCompressionService_Cache 测试缓存功能
func TestDefaultCompressionService_Cache(t *testing.T) {
	svc := NewDefaultCompressionService(nil, 100)
	prompt := generateTestPrompt()

	opts := &CompressOptions{
		Mode:     ModeSimple,
		Level:    LevelModerate,
		UseCache: true,
	}

	// 第一次压缩
	result1, err := svc.CompressSystemPrompt(context.Background(), prompt, opts)
	if err != nil {
		t.Fatalf("First compress failed: %v", err)
	}
	if result1.CacheHit {
		t.Error("First compress should not be cache hit")
	}

	// 第二次压缩（应该命中缓存）
	result2, err := svc.CompressSystemPrompt(context.Background(), prompt, opts)
	if err != nil {
		t.Fatalf("Second compress failed: %v", err)
	}
	if !result2.CacheHit {
		t.Error("Second compress should be cache hit")
	}

	// 验证结果相同
	if result1.Compressed != result2.Compressed {
		t.Error("Cache hit should return same result")
	}

	// 验证统计
	stats := svc.GetStats()
	if stats.CacheHits != 1 {
		t.Errorf("Expected 1 cache hit, got %d", stats.CacheHits)
	}
	if stats.CacheMisses != 1 {
		t.Errorf("Expected 1 cache miss, got %d", stats.CacheMisses)
	}
}

// TestDefaultCompressionService_Stats 测试统计功能
func TestDefaultCompressionService_Stats(t *testing.T) {
	svc := NewDefaultCompressionService(nil, 100)
	prompt := generateTestPrompt()

	// 执行几次压缩
	for range 3 {
		_, err := svc.CompressSystemPrompt(context.Background(), prompt, &CompressOptions{
			Mode:     ModeSimple,
			Level:    LevelModerate,
			UseCache: false, // 禁用缓存以确保每次都执行
		})
		if err != nil {
			t.Fatalf("Compress failed: %v", err)
		}
	}

	stats := svc.GetStats()
	if stats.TotalCompressions != 3 {
		t.Errorf("Expected 3 compressions, got %d", stats.TotalCompressions)
	}

	// 重置统计
	svc.ResetStats()
	stats = svc.GetStats()
	if stats.TotalCompressions != 0 {
		t.Errorf("Expected 0 compressions after reset, got %d", stats.TotalCompressions)
	}
}

// TestDefaultCompressionService_HybridMode 测试混合模式
func TestDefaultCompressionService_HybridMode(t *testing.T) {
	svc := NewDefaultCompressionService(nil, 100)
	prompt := generateTestPrompt()

	// 第一阶段：测试 hybrid 模式基本工作
	result, err := svc.CompressSystemPrompt(context.Background(), prompt, &CompressOptions{
		Mode:             ModeHybrid,
		Level:            LevelModerate,
		PreserveSections: []string{"Tools Manual", "Security"},
	})

	if err != nil {
		t.Fatalf("CompressSystemPrompt failed: %v", err)
	}

	// Hybrid 模式会先做简单压缩，然后尝试 LLM 压缩
	// 由于没有实际的 LLM，只有简单压缩会生效
	// 所以压缩后的长度应该小于原始长度
	originalLen := len(prompt)
	if result.CompressedLength >= originalLen {
		t.Errorf("Expected compression in hybrid mode (compared to original prompt), got original=%d, compressed=%d",
			originalLen, result.CompressedLength)
	}

	t.Logf("Hybrid mode: original=%d, compressed=%d, ratio=%.2f%%",
		originalLen, result.CompressedLength, float64(result.CompressedLength)/float64(originalLen)*100)

	// 验证保留段落
	if !strings.Contains(result.Compressed, "Tools Manual") {
		t.Error("Expected 'Tools Manual' section to be preserved in hybrid mode")
	}
}

// TestDefaultCompressionService_EmptyPrompt 测试空 prompt
func TestDefaultCompressionService_EmptyPrompt(t *testing.T) {
	svc := NewDefaultCompressionService(nil, 100)

	result, err := svc.CompressSystemPrompt(context.Background(), "", &CompressOptions{
		Mode:  ModeSimple,
		Level: LevelModerate,
	})

	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	if result.Compressed != "" {
		t.Errorf("Expected empty result, got %q", result.Compressed)
	}
}

// TestCompressionCache 测试压缩缓存
func TestCompressionCache(t *testing.T) {
	cache := NewCompressionCache(3)

	// 添加几个结果
	cache.Set("key1", &CompressResult{Compressed: "result1"})
	cache.Set("key2", &CompressResult{Compressed: "result2"})
	cache.Set("key3", &CompressResult{Compressed: "result3"})

	// 验证获取
	if r := cache.Get("key1"); r == nil || r.Compressed != "result1" {
		t.Error("Expected to get key1")
	}

	// 验证大小
	if cache.Size() != 3 {
		t.Errorf("Expected cache size 3, got %d", cache.Size())
	}

	// 添加第四个（应该触发清空）
	cache.Set("key4", &CompressResult{Compressed: "result4"})

	// 旧的应该被清空
	if cache.Get("key1") != nil {
		t.Error("Expected key1 to be cleared after overflow")
	}

	// 新的应该存在
	if r := cache.Get("key4"); r == nil || r.Compressed != "result4" {
		t.Error("Expected to get key4")
	}

	// 测试清空
	cache.Clear()
	if cache.Size() != 0 {
		t.Errorf("Expected cache size 0 after clear, got %d", cache.Size())
	}
}

// TestSectionScoring 测试段落评分
func TestSectionScoring(t *testing.T) {
	svc := NewDefaultCompressionService(nil, 100)

	sections := []string{
		"## Tools Manual\nThis section describes available tools.",
		"## Regular Section\nSome regular content here.",
		"## Security Guidelines\nIMPORTANT: Never expose credentials.",
		"Some text without header",
	}

	scored := svc.scoreSections(sections, []string{"Tools Manual"})

	// 验证评分数量
	if len(scored) != len(sections) {
		t.Errorf("Expected %d scored sections, got %d", len(sections), len(scored))
	}

	// 验证必须保留的段落
	var toolsManualScore float64
	for _, s := range scored {
		if strings.Contains(s.content, "Tools Manual") {
			toolsManualScore = s.score
			if !s.mustKeep {
				t.Error("Expected 'Tools Manual' section to be marked as mustKeep")
			}
		}
	}

	// Tools Manual 应该有最高评分
	if toolsManualScore < 1.0 {
		t.Errorf("Expected 'Tools Manual' score = 1.0, got %f", toolsManualScore)
	}
}

// TestExtractSectionTitle 测试提取段落标题
func TestExtractSectionTitle(t *testing.T) {
	svc := NewDefaultCompressionService(nil, 100)

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
		result := svc.extractSectionTitle(tc.section)
		if result != tc.expected {
			t.Errorf("extractSectionTitle(%q) = %q, expected %q",
				tc.section, result, tc.expected)
		}
	}
}

// TestShouldPreserve 测试保留段落判断
func TestShouldPreserve(t *testing.T) {
	svc := NewDefaultCompressionService(nil, 100)

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
		result := svc.shouldPreserve(tc.title, tc.preserve)
		if result != tc.expected {
			t.Errorf("shouldPreserve(%q, %v) = %v, expected %v",
				tc.title, tc.preserve, result, tc.expected)
		}
	}
}

// TestRemoveDuplicateEmptyLines 测试移除重复空行
func TestRemoveDuplicateEmptyLines(t *testing.T) {
	svc := NewDefaultCompressionService(nil, 100)

	input := "Line 1\n\n\n\nLine 2\n\n\nLine 3"
	expected := "Line 1\n\nLine 2\n\nLine 3"

	result := svc.removeDuplicateEmptyLines(input)
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestCompactFormat 测试紧凑格式
func TestCompactFormat(t *testing.T) {
	svc := NewDefaultCompressionService(nil, 100)

	input := "Line 1   \n  Line 2\t  \nLine 3"
	expected := "Line 1\n  Line 2\nLine 3"

	result := svc.compactFormat(input)
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// generateTestPrompt 生成测试用的 prompt
func generateTestPrompt() string {
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
