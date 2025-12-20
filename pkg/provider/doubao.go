package provider

import (
	"errors"

	"github.com/astercloud/aster/pkg/types"
)

const (
	// DoubaoAPIBaseURL Doubao API 基础 URL（字节跳动火山引擎）
	DoubaoAPIBaseURL = "https://ark.cn-beijing.volces.com/api/v3"
)

// DoubaoProvider Doubao（豆包）提供商
// 字节跳动的企业级 AI 服务，基于火山引擎
type DoubaoProvider struct {
	*OpenAICompatibleProvider

	endpointID string // 模型端点 ID
}

// DoubaoConfig Doubao 特定配置
type DoubaoConfig struct {
	// EndpointID 模型端点 ID（必需）
	EndpointID string
}

// NewDoubaoProvider 创建 Doubao 提供商
func NewDoubaoProvider(config *types.ModelConfig, dbConfig *DoubaoConfig) (Provider, error) {
	if dbConfig == nil || dbConfig.EndpointID == "" {
		// 尝试从 Model 字段获取 endpoint_id
		if config.Model == "" {
			return nil, errors.New("doubao: endpoint_id is required")
		}
		dbConfig = &DoubaoConfig{
			EndpointID: config.Model,
		}
	}

	// 使用配置中的 BaseURL，或使用默认值
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = DoubaoAPIBaseURL
	}

	// Doubao 配置选项
	options := &OpenAICompatibleOptions{
		RequireAPIKey:      true,
		DefaultModel:       dbConfig.EndpointID, // Doubao 使用 endpoint_id 作为 model
		SupportReasoning:   false,
		SupportPromptCache: false,
		SupportVision:      true, // 部分模型支持
		SupportAudio:       false,
	}

	// 创建 OpenAI 兼容 Provider
	baseProvider, err := NewOpenAICompatibleProvider(
		config,
		baseURL,
		"Doubao",
		options,
	)
	if err != nil {
		return nil, err
	}

	return &DoubaoProvider{
		OpenAICompatibleProvider: baseProvider,
		endpointID:               dbConfig.EndpointID,
	}, nil
}

// NewDoubaoProviderSimple 创建 Doubao 提供商（简化版）
func NewDoubaoProviderSimple(config *types.ModelConfig) (Provider, error) {
	return NewDoubaoProvider(config, nil)
}

// Capabilities 返回 Doubao 的能力
func (p *DoubaoProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		SupportToolCalling:  true,
		SupportSystemPrompt: true,
		SupportStreaming:    true,
		SupportVision:       true, // 部分模型支持
		SupportAudio:        false,
		SupportReasoning:    false,
		SupportPromptCache:  false,
		SupportJSONMode:     true,
		SupportFunctionCall: true,
		MaxTokens:           32768, // 取决于具体模型
		ToolCallingFormat:   "openai",
	}
}

// DoubaoFactory Doubao 工厂
type DoubaoFactory struct {
	Config *DoubaoConfig
}

// Create 创建 Doubao 提供商
func (f *DoubaoFactory) Create(config *types.ModelConfig) (Provider, error) {
	return NewDoubaoProvider(config, f.Config)
}
