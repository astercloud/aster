package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/types"
)

// ModelFallback 模型降级配置
type ModelFallback struct {
	// Config 模型配置
	Config *types.ModelConfig

	// MaxRetries 最大重试次数
	MaxRetries int

	// Enabled 是否启用此模型
	Enabled bool

	// Priority 优先级（数字越小优先级越高）
	Priority int

	// provider 缓存的 Provider 实例
	provider provider.Provider
}

// ModelFallbackManager 模型降级管理器
type ModelFallbackManager struct {
	// fallbacks 降级模型列表（按优先级排序）
	fallbacks []*ModelFallback

	// deps Agent 依赖
	deps *Dependencies

	// currentIndex 当前使用的模型索引
	currentIndex int

	// stats 统计信息
	stats *FallbackStats
}

// FallbackStats 降级统计信息
type FallbackStats struct {
	TotalRequests    int64
	SuccessRequests  int64
	FailedRequests   int64
	FallbackCount    int64
	ModelUsageCount  map[string]int64
	LastFallbackTime time.Time
}

// NewModelFallbackManager 创建模型降级管理器
func NewModelFallbackManager(fallbacks []*ModelFallback, deps *Dependencies) (*ModelFallbackManager, error) {
	if len(fallbacks) == 0 {
		return nil, fmt.Errorf("at least one model fallback is required")
	}

	// 按优先级排序
	sortedFallbacks := make([]*ModelFallback, len(fallbacks))
	copy(sortedFallbacks, fallbacks)

	// 简单的冒泡排序（因为通常模型数量不多）
	for i := 0; i < len(sortedFallbacks)-1; i++ {
		for j := 0; j < len(sortedFallbacks)-i-1; j++ {
			if sortedFallbacks[j].Priority > sortedFallbacks[j+1].Priority {
				sortedFallbacks[j], sortedFallbacks[j+1] = sortedFallbacks[j+1], sortedFallbacks[j]
			}
		}
	}

	// 初始化 Provider 实例
	for _, fb := range sortedFallbacks {
		if !fb.Enabled {
			continue
		}

		prov, err := deps.ProviderFactory.Create(fb.Config)
		if err != nil {
			log.Printf("[ModelFallback] Failed to create provider for %s/%s: %v",
				fb.Config.Provider, fb.Config.Model, err)
			fb.Enabled = false
			continue
		}
		fb.provider = prov
	}

	return &ModelFallbackManager{
		fallbacks:    sortedFallbacks,
		deps:         deps,
		currentIndex: 0,
		stats: &FallbackStats{
			ModelUsageCount: make(map[string]int64),
		},
	}, nil
}

// Complete 执行非流式请求，支持自动降级
func (m *ModelFallbackManager) Complete(
	ctx context.Context,
	messages []types.Message,
	opts *provider.StreamOptions,
) (*provider.CompleteResponse, error) {
	m.stats.TotalRequests++

	var lastErr error

	// 遍历所有启用的模型
	for i, fb := range m.fallbacks {
		if !fb.Enabled {
			continue
		}

		modelKey := fmt.Sprintf("%s/%s", fb.Config.Provider, fb.Config.Model)

		// 尝试执行，支持重试
		for retry := 0; retry <= fb.MaxRetries; retry++ {
			if retry > 0 {
				log.Printf("[ModelFallback] Retry %d/%d for model %s",
					retry, fb.MaxRetries, modelKey)

				// 重试前等待一小段时间（指数退避）
				backoff := time.Duration(retry) * 500 * time.Millisecond
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(backoff):
				}
			}

			// 执行请求
			resp, err := fb.provider.Complete(ctx, messages, opts)
			if err == nil {
				// 成功
				m.stats.SuccessRequests++
				m.stats.ModelUsageCount[modelKey]++
				m.currentIndex = i

				log.Printf("[ModelFallback] Success with model %s (retry: %d)", modelKey, retry)
				return resp, nil
			}

			lastErr = err
			log.Printf("[ModelFallback] Error with model %s (retry: %d/%d): %v",
				modelKey, retry, fb.MaxRetries, err)
		}

		// 所有重试都失败，尝试下一个模型
		if i < len(m.fallbacks)-1 {
			m.stats.FallbackCount++
			m.stats.LastFallbackTime = time.Now()
			log.Printf("[ModelFallback] Falling back from %s to next model", modelKey)
		}
	}

	// 所有模型都失败
	m.stats.FailedRequests++
	return nil, fmt.Errorf("all models failed, last error: %w", lastErr)
}

// Stream 执行流式请求，支持自动降级
func (m *ModelFallbackManager) Stream(
	ctx context.Context,
	messages []types.Message,
	opts *provider.StreamOptions,
) (<-chan provider.StreamChunk, error) {
	m.stats.TotalRequests++

	var lastErr error

	// 遍历所有启用的模型
	for i, fb := range m.fallbacks {
		if !fb.Enabled {
			continue
		}

		modelKey := fmt.Sprintf("%s/%s", fb.Config.Provider, fb.Config.Model)

		// 尝试执行，支持重试
		for retry := 0; retry <= fb.MaxRetries; retry++ {
			if retry > 0 {
				log.Printf("[ModelFallback] Retry %d/%d for model %s (stream)",
					retry, fb.MaxRetries, modelKey)

				// 重试前等待一小段时间
				backoff := time.Duration(retry) * 500 * time.Millisecond
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(backoff):
				}
			}

			// 执行流式请求
			stream, err := fb.provider.Stream(ctx, messages, opts)
			if err == nil {
				// 成功
				m.stats.SuccessRequests++
				m.stats.ModelUsageCount[modelKey]++
				m.currentIndex = i

				log.Printf("[ModelFallback] Success with model %s (stream, retry: %d)", modelKey, retry)
				return stream, nil
			}

			lastErr = err
			log.Printf("[ModelFallback] Error with model %s (stream, retry: %d/%d): %v",
				modelKey, retry, fb.MaxRetries, err)
		}

		// 所有重试都失败，尝试下一个模型
		if i < len(m.fallbacks)-1 {
			m.stats.FallbackCount++
			m.stats.LastFallbackTime = time.Now()
			log.Printf("[ModelFallback] Falling back from %s to next model (stream)", modelKey)
		}
	}

	// 所有模型都失败
	m.stats.FailedRequests++
	return nil, fmt.Errorf("all models failed (stream), last error: %w", lastErr)
}

// GetCurrentProvider 获取当前使用的 Provider
func (m *ModelFallbackManager) GetCurrentProvider() provider.Provider {
	if m.currentIndex >= 0 && m.currentIndex < len(m.fallbacks) {
		return m.fallbacks[m.currentIndex].provider
	}
	return nil
}

// GetStats 获取统计信息
func (m *ModelFallbackManager) GetStats() *FallbackStats {
	return m.stats
}

// EnableModel 启用指定模型
func (m *ModelFallbackManager) EnableModel(provider, model string) error {
	modelKey := fmt.Sprintf("%s/%s", provider, model)

	for _, fb := range m.fallbacks {
		fbKey := fmt.Sprintf("%s/%s", fb.Config.Provider, fb.Config.Model)
		if fbKey == modelKey {
			if !fb.Enabled && fb.provider == nil {
				// 需要重新创建 Provider
				prov, err := m.deps.ProviderFactory.Create(fb.Config)
				if err != nil {
					return fmt.Errorf("failed to create provider: %w", err)
				}
				fb.provider = prov
			}
			fb.Enabled = true
			log.Printf("[ModelFallback] Enabled model %s", modelKey)
			return nil
		}
	}

	return fmt.Errorf("model not found: %s", modelKey)
}

// DisableModel 禁用指定模型
func (m *ModelFallbackManager) DisableModel(provider, model string) error {
	modelKey := fmt.Sprintf("%s/%s", provider, model)

	for _, fb := range m.fallbacks {
		fbKey := fmt.Sprintf("%s/%s", fb.Config.Provider, fb.Config.Model)
		if fbKey == modelKey {
			fb.Enabled = false
			log.Printf("[ModelFallback] Disabled model %s", modelKey)
			return nil
		}
	}

	return fmt.Errorf("model not found: %s", modelKey)
}

// ListModels 列出所有模型及其状态
func (m *ModelFallbackManager) ListModels() []map[string]interface{} {
	models := make([]map[string]interface{}, 0, len(m.fallbacks))

	for i, fb := range m.fallbacks {
		modelKey := fmt.Sprintf("%s/%s", fb.Config.Provider, fb.Config.Model)
		models = append(models, map[string]interface{}{
			"provider":    fb.Config.Provider,
			"model":       fb.Config.Model,
			"enabled":     fb.Enabled,
			"priority":    fb.Priority,
			"max_retries": fb.MaxRetries,
			"is_current":  i == m.currentIndex,
			"usage_count": m.stats.ModelUsageCount[modelKey],
		})
	}

	return models
}

// ResetStats 重置统计信息
func (m *ModelFallbackManager) ResetStats() {
	m.stats = &FallbackStats{
		ModelUsageCount: make(map[string]int64),
	}
}
