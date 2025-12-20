package provider

import (
	"slices"

	"github.com/astercloud/aster/pkg/types"
)

const (
	// OpenAIAPIBaseURL OpenAI API 基础 URL
	OpenAIAPIBaseURL = "https://api.openai.com/v1"
)

// OpenAIProvider OpenAI 提供商
// 支持 GPT-4/4.5/5, GPT-4o, o1/o3 等最先进的模型
type OpenAIProvider struct {
	*OpenAICompatibleProvider
}

// NewOpenAIProvider 创建 OpenAI 提供商
func NewOpenAIProvider(config *types.ModelConfig) (Provider, error) {
	// OpenAI 配置选项
	options := &OpenAICompatibleOptions{
		RequireAPIKey:      true,
		DefaultModel:       "gpt-4o", // 默认使用 GPT-4o
		SupportReasoning:   true,     // 支持 o1/o3 推理模型
		SupportPromptCache: true,     // 支持 Prompt Caching
		SupportVision:      true,     // 支持图片输入
		SupportAudio:       true,     // 支持音频输入
	}

	// 创建 OpenAI 兼容 Provider
	baseProvider, err := NewOpenAICompatibleProvider(
		config,
		OpenAIAPIBaseURL,
		"OpenAI",
		options,
	)
	if err != nil {
		return nil, err
	}

	return &OpenAIProvider{
		OpenAICompatibleProvider: baseProvider,
	}, nil
}

// Capabilities 返回 OpenAI 的能力
func (p *OpenAIProvider) Capabilities() ProviderCapabilities {
	caps := ProviderCapabilities{
		SupportToolCalling:      true,
		SupportSystemPrompt:     true,
		SupportStreaming:        true,
		SupportVision:           true,
		SupportAudio:            true,
		SupportReasoning:        true, // o1/o3 模型
		SupportPromptCache:      true,
		SupportJSONMode:         true,
		SupportFunctionCall:     true,
		MaxTokens:               128000,
		ToolCallingFormat:       "openai",
		ReasoningTokensIncluded: true,
		CacheMinTokens:          1024, // Prompt Caching 最小 token 数
	}

	// 根据模型调整能力
	model := p.Config().Model
	if isReasoningModel(model) {
		// 推理模型不支持某些功能
		caps.SupportToolCalling = false
		caps.SupportVision = false
		caps.MaxTokens = 100000 // o1 模型限制
	}

	return caps
}

// isReasoningModel 检查是否是推理模型
func isReasoningModel(model string) bool {
	reasoningModels := []string{"o1", "o3", "o1-mini", "o3-mini", "o1-preview"}
	return slices.Contains(reasoningModels, model)
}

// OpenAIFactory OpenAI 工厂
type OpenAIFactory struct{}

// Create 创建 OpenAI 提供商
func (f *OpenAIFactory) Create(config *types.ModelConfig) (Provider, error) {
	return NewOpenAIProvider(config)
}
