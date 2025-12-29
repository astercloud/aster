package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/types"
)

var gatewayLog = logging.ForComponent("GatewayProvider")

// GatewayProvider 是一个通用的 API 网关 provider
// 支持将请求转发到自定义的 base_url，并根据 model 自动选择协议
type GatewayProvider struct {
	config *types.ModelConfig

	// 内部使用的实际 provider
	innerProvider Provider

	// 检测到的协议类型
	protocol string
}

// NewGatewayProvider 创建新的 Gateway provider
//
// Gateway provider 会根据 model 名称自动推断使用哪种协议：
// - claude-* -> Anthropic 协议
// - gpt-*, o1-*, o3-* -> OpenAI 协议
// - gemini-* -> Gemini 协议
// - 其他 -> OpenAI 兼容协议
func NewGatewayProvider(config *types.ModelConfig) (Provider, error) {
	if config.BaseURL == "" {
		return nil, errors.New("gateway provider requires base_url")
	}

	g := &GatewayProvider{
		config: config,
	}

	// 根据 model 名称推断协议
	g.protocol = detectProtocol(config.Model)

	gatewayLog.Info(context.Background(), "Creating gateway provider", map[string]any{
		"model":    config.Model,
		"protocol": g.protocol,
		"base_url": config.BaseURL,
	})

	// 创建内部 provider 配置
	innerConfig := &types.ModelConfig{
		Provider: g.protocol,
		Model:    config.Model,
		APIKey:   config.APIKey,
		BaseURL:  config.BaseURL,
	}

	var err error
	switch g.protocol {
	case "anthropic":
		g.innerProvider, err = NewAnthropicProvider(innerConfig)
	case "openai":
		g.innerProvider, err = NewOpenAIProviderWithBaseURL(innerConfig)
	case "gemini":
		g.innerProvider, err = NewGeminiProvider(innerConfig)
	default:
		// 默认使用 OpenAI 兼容协议
		g.innerProvider, err = NewCustomProvider(innerConfig)
	}

	if err != nil {
		return nil, fmt.Errorf("gateway: failed to create inner provider (%s): %w", g.protocol, err)
	}

	return g, nil
}

// detectProtocol 根据 model 名称推断协议
func detectProtocol(model string) string {
	modelLower := strings.ToLower(model)

	// Claude 模型 -> Anthropic 协议
	if strings.HasPrefix(modelLower, "claude") {
		return "anthropic"
	}

	// GPT/O1/O3/O4 模型 -> OpenAI 协议
	if strings.HasPrefix(modelLower, "gpt") ||
		strings.HasPrefix(modelLower, "o1") ||
		strings.HasPrefix(modelLower, "o3") ||
		strings.HasPrefix(modelLower, "o4") {
		return "openai"
	}

	// Gemini 模型 -> Gemini 协议
	if strings.HasPrefix(modelLower, "gemini") {
		return "gemini"
	}

	// Qwen/通义千问 模型 -> OpenAI 兼容协议
	if strings.HasPrefix(modelLower, "qwen") {
		return "openai"
	}

	// DeepSeek 模型 -> OpenAI 兼容协议
	if strings.HasPrefix(modelLower, "deepseek") {
		return "openai"
	}

	// 默认使用 OpenAI 兼容协议
	return "openai"
}

// Stream 实现 Provider 接口 - 流式对话
func (g *GatewayProvider) Stream(ctx context.Context, messages []types.Message, opts *StreamOptions) (<-chan StreamChunk, error) {
	return g.innerProvider.Stream(ctx, messages, opts)
}

// Complete 实现 Provider 接口 - 非流式对话
func (g *GatewayProvider) Complete(ctx context.Context, messages []types.Message, opts *StreamOptions) (*CompleteResponse, error) {
	return g.innerProvider.Complete(ctx, messages, opts)
}

// Config 返回配置
func (g *GatewayProvider) Config() *types.ModelConfig {
	return g.config
}

// Capabilities 返回模型能力
func (g *GatewayProvider) Capabilities() ProviderCapabilities {
	return g.innerProvider.Capabilities()
}

// SetSystemPrompt 设置系统提示词
func (g *GatewayProvider) SetSystemPrompt(prompt string) error {
	return g.innerProvider.SetSystemPrompt(prompt)
}

// GetSystemPrompt 获取系统提示词
func (g *GatewayProvider) GetSystemPrompt() string {
	return g.innerProvider.GetSystemPrompt()
}

// Close 关闭连接
func (g *GatewayProvider) Close() error {
	return g.innerProvider.Close()
}

// Protocol 返回检测到的协议类型
func (g *GatewayProvider) Protocol() string {
	return g.protocol
}
