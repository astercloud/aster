package middleware

import (
	"context"

	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/memory"
	"github.com/astercloud/aster/pkg/types"
)

var ocLog = logging.ForComponent("ObservationCompression")

// ObservationCompressionMiddleware 观察结果压缩中间件
// 用于压缩工具执行结果，同时保留可恢复的引用信息
// 这是 Manus 团队"文件系统作为上下文"理念的实现
type ObservationCompressionMiddleware struct {
	*BaseMiddleware

	compressor        memory.ObservationCompressor
	referenceRegistry memory.ReferenceRegistry

	// 配置
	enabled          bool
	minContentLength int // 触发压缩的最小内容长度

	// 统计
	compressionCount int
	totalSaved       int
}

// ObservationCompressionConfig 配置
type ObservationCompressionConfig struct {
	// Compressor 自定义压缩器（可选，默认使用 DefaultObservationCompressor）
	Compressor memory.ObservationCompressor

	// ReferenceRegistry 引用注册表（可选）
	ReferenceRegistry memory.ReferenceRegistry

	// Enabled 是否启用压缩
	Enabled bool

	// MinContentLength 触发压缩的最小内容长度
	MinContentLength int
}

// NewObservationCompressionMiddleware 创建观察结果压缩中间件
func NewObservationCompressionMiddleware(config *ObservationCompressionConfig) *ObservationCompressionMiddleware {
	if config == nil {
		config = &ObservationCompressionConfig{
			Enabled:          true,
			MinContentLength: 3000,
		}
	}

	var compressor memory.ObservationCompressor
	if config.Compressor != nil {
		compressor = config.Compressor
	} else {
		compressor = memory.NewDefaultObservationCompressor()
	}

	var registry memory.ReferenceRegistry
	if config.ReferenceRegistry != nil {
		registry = config.ReferenceRegistry
	} else {
		registry = memory.NewInMemoryReferenceRegistry(1000)
	}

	return &ObservationCompressionMiddleware{
		BaseMiddleware:    NewBaseMiddleware("observation_compression", 35), // 在 summarization (40) 之前
		compressor:        compressor,
		referenceRegistry: registry,
		enabled:           config.Enabled,
		minContentLength:  config.MinContentLength,
	}
}

// WrapModelCall 包装模型调用，在发送前压缩历史工具结果
func (m *ObservationCompressionMiddleware) WrapModelCall(
	ctx context.Context,
	req *ModelRequest,
	handler ModelCallHandler,
) (*ModelResponse, error) {
	if !m.enabled {
		return handler(ctx, req)
	}

	// 压缩消息中的工具结果
	req.Messages = m.compressMessagesToolResults(ctx, req.Messages)

	return handler(ctx, req)
}

// compressMessagesToolResults 压缩消息中的工具结果
func (m *ObservationCompressionMiddleware) compressMessagesToolResults(
	ctx context.Context,
	messages []types.Message,
) []types.Message {
	result := make([]types.Message, len(messages))

	for i, msg := range messages {
		if msg.Role != types.MessageRoleUser {
			result[i] = msg
			continue
		}

		// 检查是否包含工具结果
		hasToolResult := false
		for _, block := range msg.ContentBlocks {
			if _, ok := block.(*types.ToolResultBlock); ok {
				hasToolResult = true
				break
			}
		}

		if !hasToolResult {
			result[i] = msg
			continue
		}

		// 压缩工具结果
		newBlocks := make([]types.ContentBlock, 0, len(msg.ContentBlocks))
		for _, block := range msg.ContentBlocks {
			if toolResult, ok := block.(*types.ToolResultBlock); ok {
				compressed := m.compressToolResultBlock(ctx, toolResult)
				newBlocks = append(newBlocks, compressed)
			} else {
				newBlocks = append(newBlocks, block)
			}
		}

		result[i] = types.Message{
			Role:          msg.Role,
			Content:       msg.Content,
			ContentBlocks: newBlocks,
		}
	}

	return result
}

// compressToolResultBlock 压缩 ToolResultBlock
func (m *ObservationCompressionMiddleware) compressToolResultBlock(
	ctx context.Context,
	block *types.ToolResultBlock,
) *types.ToolResultBlock {
	// 已经压缩过的跳过
	if block.Compressed {
		return block
	}

	// 内容太短不压缩
	if len(block.Content) < m.minContentLength {
		return block
	}

	// 错误信息不压缩（保留完整上下文，这是 Manus 的最佳实践）
	if block.IsError {
		return block
	}

	// 压缩
	compressed, err := m.compressor.Compress(ctx, "unknown", block.Content)
	if err != nil {
		return block
	}

	// 如果压缩后更长，不使用
	if len(compressed.Summary) >= len(block.Content) {
		return block
	}

	// 转换引用格式
	refs := make([]types.ToolResultReference, len(compressed.References))
	for i, ref := range compressed.References {
		refs[i] = types.ToolResultReference{
			Type:    ref.Type,
			Value:   ref.Value,
			Context: ref.Context,
		}
	}

	// 注册引用
	for _, ref := range compressed.References {
		_ = m.referenceRegistry.Register(ctx, ref, "")
	}

	m.compressionCount++
	saved := len(block.Content) - len(compressed.Summary)
	m.totalSaved += saved

	ocLog.Debug(ctx, "compressed tool result", map[string]any{"original": len(block.Content), "compressed": len(compressed.Summary), "saved": saved, "reduction_pct": (1 - compressed.CompressionRatio) * 100})

	return &types.ToolResultBlock{
		ToolUseID:      block.ToolUseID,
		Content:        compressed.Summary,
		IsError:        block.IsError,
		Compressed:     true,
		OriginalLength: compressed.OriginalLength,
		ContentHash:    compressed.Hash,
		References:     refs,
	}
}

// GetStats 获取压缩统计
func (m *ObservationCompressionMiddleware) GetStats() map[string]any {
	return map[string]any{
		"compression_count": m.compressionCount,
		"total_saved_bytes": m.totalSaved,
		"enabled":           m.enabled,
	}
}

// GetReferenceRegistry 获取引用注册表
func (m *ObservationCompressionMiddleware) GetReferenceRegistry() memory.ReferenceRegistry {
	return m.referenceRegistry
}

// CompressString 便捷方法：直接压缩字符串内容
func (m *ObservationCompressionMiddleware) CompressString(ctx context.Context, toolName, content string) (string, []memory.Reference, error) {
	if len(content) < m.minContentLength {
		return content, nil, nil
	}

	compressed, err := m.compressor.Compress(ctx, toolName, content)
	if err != nil {
		return content, nil, err
	}

	// 注册引用
	for _, ref := range compressed.References {
		_ = m.referenceRegistry.Register(ctx, ref, toolName)
	}

	return compressed.Summary, compressed.References, nil
}
