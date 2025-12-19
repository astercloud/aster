package vector

import (
	"context"
)

// MockEmbedder 一个非常简化的 Embedder 实现, 仅用于示例/测试。
// 实际生产中应替换为真实的 embedding 服务(OpenAI/本地模型等)。
type MockEmbedder struct {
	Dim int
}

// NewMockEmbedder 创建一个 MockEmbedder。
// Dim 指定向量维度, 默认为 16。
func NewMockEmbedder(dim int) *MockEmbedder {
	if dim <= 0 {
		dim = 16
	}
	return &MockEmbedder{Dim: dim}
}

// EmbedText 将文本映射为简单的伪随机向量(基于字节值),
// 只保证同一文本得到相同向量, 不保证语义质量。
func (e *MockEmbedder) EmbedText(_ context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i, t := range texts {
		vec := make([]float32, e.Dim)
		if len(t) == 0 {
			result[i] = vec
			continue
		}
		for j := range e.Dim {
			b := t[j%len(t)]
			vec[j] = float32(int(b%97)) / 100.0 // 稍微分布一下
		}
		result[i] = vec
	}
	return result, nil
}
