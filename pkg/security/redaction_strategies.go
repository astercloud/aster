package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// RedactionStrategy 脱敏策略接口。
type RedactionStrategy interface {
	// Redact 脱敏单个 PII 值。
	Redact(match PIIMatch) string

	// Name 返回策略名称。
	Name() string
}

// MaskStrategy 掩码策略（部分掩码）。
// 例如：13812345678 -> 138****5678
type MaskStrategy struct {
	MaskChar      rune // 掩码字符（默认 '*'）
	KeepPrefix    int  // 保留前缀长度
	KeepSuffix    int  // 保留后缀长度
	MinMaskLength int  // 最小掩码长度
}

// NewMaskStrategy 创建掩码策略。
func NewMaskStrategy() *MaskStrategy {
	return &MaskStrategy{
		MaskChar:      '*',
		KeepPrefix:    3,
		KeepSuffix:    4,
		MinMaskLength: 4,
	}
}

// Redact 执行掩码脱敏。
func (s *MaskStrategy) Redact(match PIIMatch) string {
	value := match.Value
	runes := []rune(value)
	length := len(runes)

	// 根据 PII 类型调整策略
	prefix, suffix := s.KeepPrefix, s.KeepSuffix

	switch match.Type {
	case PIIEmail:
		// 邮箱：保留用户名首字母和域名
		return s.redactEmail(value)
	case PIICreditCard:
		// 信用卡：保留前4后4
		prefix, suffix = 4, 4
	case PIIChinesePhone, PIIPhone:
		// 电话：保留前3后4
		prefix, suffix = 3, 4
	case PIIChineseID:
		// 身份证：保留前6后4
		prefix, suffix = 6, 4
	case PIISSNus:
		// SSN：完全掩码
		prefix, suffix = 0, 0
	}

	// 确保不会越界
	if prefix+suffix >= length {
		// 如果太短，全部掩码
		return strings.Repeat(string(s.MaskChar), max(s.MinMaskLength, length))
	}

	// 构造掩码字符串
	maskLength := length - prefix - suffix
	if maskLength < s.MinMaskLength {
		maskLength = s.MinMaskLength
	}

	result := string(runes[:prefix]) +
		strings.Repeat(string(s.MaskChar), maskLength) +
		string(runes[length-suffix:])

	return result
}

// redactEmail 脱敏邮箱地址。
func (s *MaskStrategy) redactEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return strings.Repeat(string(s.MaskChar), len(email))
	}

	username := parts[0]
	domain := parts[1]

	// 保留用户名首字母
	if len(username) <= 1 {
		username = string(s.MaskChar)
	} else {
		username = string(username[0]) + strings.Repeat(string(s.MaskChar), len(username)-1)
	}

	return username + "@" + domain
}

// Name 返回策略名称。
func (s *MaskStrategy) Name() string {
	return "mask"
}

// ReplaceStrategy 替换策略（替换为占位符）。
// 例如：13812345678 -> [PHONE]
type ReplaceStrategy struct {
	UseTypeLabel bool               // 是否使用类型标签（如 [PHONE]）
	CustomLabels map[PIIType]string // 自定义标签
}

// NewReplaceStrategy 创建替换策略。
func NewReplaceStrategy() *ReplaceStrategy {
	return &ReplaceStrategy{
		UseTypeLabel: true,
		CustomLabels: make(map[PIIType]string),
	}
}

// Redact 执行替换脱敏。
func (s *ReplaceStrategy) Redact(match PIIMatch) string {
	if !s.UseTypeLabel {
		return "[REDACTED]"
	}

	// 检查是否有自定义标签
	if label, ok := s.CustomLabels[match.Type]; ok {
		return label
	}

	// 使用默认标签
	label := strings.ToUpper(string(match.Type))
	return fmt.Sprintf("[%s]", label)
}

// Name 返回策略名称。
func (s *ReplaceStrategy) Name() string {
	return "replace"
}

// HashStrategy 哈希策略（单向加密）。
// 例如：13812345678 -> [HASH:a3f5...]
type HashStrategy struct {
	ShowPrefix   bool   // 是否显示哈希前缀
	PrefixLength int    // 哈希前缀长度
	Salt         string // 盐值（用于增强安全性）
}

// NewHashStrategy 创建哈希策略。
func NewHashStrategy() *HashStrategy {
	return &HashStrategy{
		ShowPrefix:   true,
		PrefixLength: 8,
		Salt:         "agentsdk-default-salt", // 生产环境应使用随机盐值
	}
}

// Redact 执行哈希脱敏。
func (s *HashStrategy) Redact(match PIIMatch) string {
	// 使用 SHA256 哈希
	data := match.Value + s.Salt
	hash := sha256.Sum256([]byte(data))
	hashStr := hex.EncodeToString(hash[:])

	if s.ShowPrefix {
		prefix := hashStr[:min(s.PrefixLength, len(hashStr))]
		return fmt.Sprintf("[HASH:%s...]", prefix)
	}

	return hashStr
}

// Name 返回策略名称。
func (s *HashStrategy) Name() string {
	return "hash"
}

// AdaptiveStrategy 自适应策略（根据敏感度选择策略）。
type AdaptiveStrategy struct {
	LowStrategy    RedactionStrategy
	MediumStrategy RedactionStrategy
	HighStrategy   RedactionStrategy
}

// NewAdaptiveStrategy 创建自适应策略。
func NewAdaptiveStrategy() *AdaptiveStrategy {
	return &AdaptiveStrategy{
		LowStrategy:    NewMaskStrategy(),
		MediumStrategy: NewMaskStrategy(),
		HighStrategy:   NewReplaceStrategy(),
	}
}

// Redact 根据敏感度选择策略。
func (s *AdaptiveStrategy) Redact(match PIIMatch) string {
	switch match.Severity {
	case SensitivityHigh:
		return s.HighStrategy.Redact(match)
	case SensitivityMedium:
		return s.MediumStrategy.Redact(match)
	default:
		return s.LowStrategy.Redact(match)
	}
}

// Name 返回策略名称。
func (s *AdaptiveStrategy) Name() string {
	return "adaptive"
}

// NoOpStrategy 无操作策略（不脱敏，用于测试）。
type NoOpStrategy struct{}

// Redact 不进行脱敏。
func (s *NoOpStrategy) Redact(match PIIMatch) string {
	return match.Value
}

// Name 返回策略名称。
func (s *NoOpStrategy) Name() string {
	return "noop"
}

// Redactor PII 脱敏器。
type Redactor struct {
	detector PIIDetector
	strategy RedactionStrategy
}

// NewRedactor 创建脱敏器。
func NewRedactor(detector PIIDetector, strategy RedactionStrategy) *Redactor {
	return &Redactor{
		detector: detector,
		strategy: strategy,
	}
}

// Redact 脱敏文本中的所有 PII。
func (r *Redactor) Redact(ctx context.Context, text string) (string, error) {
	// 检测 PII
	matches, err := r.detector.Detect(ctx, text)
	if err != nil {
		return text, err
	}

	if len(matches) == 0 {
		return text, nil
	}

	// 转换为 rune 数组
	runes := []rune(text)

	// 构建字节位置到 rune 位置的映射
	byteToRune := buildByteToRuneMap(text)

	// 从后往前替换，避免位置偏移
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		redacted := r.strategy.Redact(match)

		// 将字节位置转换为 rune 位置
		startRune := byteToRune[match.Start]
		endRune := byteToRune[match.End]

		// 替换原文本
		runes = append(runes[:startRune], append([]rune(redacted), runes[endRune:]...)...)
	}

	return string(runes), nil
}

// RedactWithReport 脱敏文本并返回详细报告。
func (r *Redactor) RedactWithReport(ctx context.Context, text string) (string, *RedactionReport, error) {
	matches, err := r.detector.Detect(ctx, text)
	if err != nil {
		return text, nil, err
	}

	report := &RedactionReport{
		OriginalLength: len(text),
		TotalMatches:   len(matches),
		MatchesByType:  make(map[PIIType]int),
	}

	if len(matches) == 0 {
		report.RedactedLength = len(text)
		return text, report, nil
	}

	// 转换为 rune 数组
	runes := []rune(text)

	// 构建字节位置到 rune 位置的映射
	byteToRune := buildByteToRuneMap(text)

	// 从后往前替换
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		redacted := r.strategy.Redact(match)

		report.MatchesByType[match.Type]++
		report.RedactedCharacters += match.End - match.Start

		// 将字节位置转换为 rune 位置
		startRune := byteToRune[match.Start]
		endRune := byteToRune[match.End]

		runes = append(runes[:startRune], append([]rune(redacted), runes[endRune:]...)...)
	}

	redactedText := string(runes)
	report.RedactedLength = len(redactedText)

	return redactedText, report, nil
}

// RedactionReport 脱敏报告。
type RedactionReport struct {
	OriginalLength     int             // 原始文本长度
	RedactedLength     int             // 脱敏后文本长度
	TotalMatches       int             // 总匹配数
	RedactedCharacters int             // 脱敏字符数
	MatchesByType      map[PIIType]int // 每种类型的匹配数
}

// helper functions

// buildByteToRuneMap 构建字节位置到 rune 位置的映射。
// 用于处理多字节 UTF-8 字符（如中文）时的位置转换。
func buildByteToRuneMap(text string) []int {
	byteToRune := make([]int, len(text)+1)
	runeIndex := 0

	for byteIndex := range text {
		byteToRune[byteIndex] = runeIndex
		// 检查是否是 UTF-8 字符的第一个字节
		// ASCII 字符 (< 0x80) 或 UTF-8 序列的首字节 (非续字节)
		if text[byteIndex] < 0x80 || (text[byteIndex]&0xC0) == 0xC0 {
			runeIndex++
		}
	}
	byteToRune[len(text)] = runeIndex

	return byteToRune
}
