package provider

import (
	"testing"

	"github.com/astercloud/aster/pkg/types"
)

// TestOpenAIProvider 测试 OpenAI Provider
func TestOpenAIProvider(t *testing.T) {
	config := &types.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   "test-key",
	}

	provider, err := NewOpenAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create OpenAI provider: %v", err)
	}

	// 测试配置
	if provider.Config().Provider != "openai" {
		t.Errorf("Expected provider=openai, got %s", provider.Config().Provider)
	}

	if provider.Config().Model != "gpt-4o" {
		t.Errorf("Expected model=gpt-4o, got %s", provider.Config().Model)
	}

	// 测试能力
	caps := provider.Capabilities()
	if !caps.SupportStreaming {
		t.Error("OpenAI should support streaming")
	}
	if !caps.SupportToolCalling {
		t.Error("OpenAI should support tool calling")
	}
	if !caps.SupportVision {
		t.Error("OpenAI should support vision")
	}
	if !caps.SupportReasoning {
		t.Error("OpenAI should support reasoning models")
	}

	// 测试系统提示词
	err = provider.SetSystemPrompt("You are a helpful assistant")
	if err != nil {
		t.Errorf("Failed to set system prompt: %v", err)
	}

	prompt := provider.GetSystemPrompt()
	if prompt != "You are a helpful assistant" {
		t.Errorf("Expected prompt='You are a helpful assistant', got '%s'", prompt)
	}
}

// TestGroqProvider 测试 Groq Provider
func TestGroqProvider(t *testing.T) {
	config := &types.ModelConfig{
		Provider: "groq",
		Model:    "llama-3.3-70b-versatile",
		APIKey:   "test-key",
	}

	provider, err := NewGroqProvider(config)
	if err != nil {
		t.Fatalf("Failed to create Groq provider: %v", err)
	}

	// 测试配置
	if provider.Config().Model != "llama-3.3-70b-versatile" {
		t.Errorf("Expected model=llama-3.3-70b-versatile, got %s", provider.Config().Model)
	}

	// 测试能力
	caps := provider.Capabilities()
	if !caps.SupportStreaming {
		t.Error("Groq should support streaming")
	}
	if !caps.SupportToolCalling {
		t.Error("Groq should support tool calling")
	}
	if caps.SupportVision {
		t.Error("Groq should not support vision")
	}
}

// TestOllamaProvider 测试 Ollama Provider
func TestOllamaProvider(t *testing.T) {
	config := &types.ModelConfig{
		Provider: "ollama",
		Model:    "llama3.2",
		BaseURL:  "http://localhost:11434/v1",
	}

	provider, err := NewOllamaProvider(config)
	if err != nil {
		t.Fatalf("Failed to create Ollama provider: %v", err)
	}

	// 测试配置
	if provider.Config().Model != "llama3.2" {
		t.Errorf("Expected model=llama3.2, got %s", provider.Config().Model)
	}

	// Ollama 不需要 API Key
	if provider.Config().APIKey != "" {
		t.Errorf("Ollama should not require API key")
	}

	// 测试能力
	caps := provider.Capabilities()
	if !caps.SupportStreaming {
		t.Error("Ollama should support streaming")
	}
}

// TestOpenRouterProvider 测试 OpenRouter Provider
func TestOpenRouterProvider(t *testing.T) {
	config := &types.ModelConfig{
		Provider: "openrouter",
		Model:    "openai/gpt-4o",
		APIKey:   "test-key",
	}

	provider, err := NewOpenRouterProviderSimple(config)
	if err != nil {
		t.Fatalf("Failed to create OpenRouter provider: %v", err)
	}

	// 测试配置
	if provider.Config().Model != "openai/gpt-4o" {
		t.Errorf("Expected model=openai/gpt-4o, got %s", provider.Config().Model)
	}

	// 测试能力
	caps := provider.Capabilities()
	if !caps.SupportStreaming {
		t.Error("OpenRouter should support streaming")
	}
	if !caps.SupportToolCalling {
		t.Error("OpenRouter should support tool calling")
	}
	if !caps.SupportReasoning {
		t.Error("OpenRouter should support reasoning models")
	}
	if caps.MaxTokens < 100000 {
		t.Error("OpenRouter should support large context")
	}
}

// TestMistralProvider 测试 Mistral Provider
func TestMistralProvider(t *testing.T) {
	config := &types.ModelConfig{
		Provider: "mistral",
		Model:    "mistral-large-latest",
		APIKey:   "test-key",
	}

	provider, err := NewMistralProvider(config)
	if err != nil {
		t.Fatalf("Failed to create Mistral provider: %v", err)
	}

	// 测试能力
	caps := provider.Capabilities()
	if !caps.SupportReasoning {
		t.Error("Mistral should support reasoning")
	}
	if !caps.SupportVision {
		t.Error("Mistral should support vision (Pixtral)")
	}
}

// TestDoubaoProvider 测试 Doubao Provider
func TestDoubaoProvider(t *testing.T) {
	config := &types.ModelConfig{
		Provider: "doubao",
		Model:    "ep-20240101-xxxxx",
		APIKey:   "test-key",
	}

	provider, err := NewDoubaoProviderSimple(config)
	if err != nil {
		t.Fatalf("Failed to create Doubao provider: %v", err)
	}

	// 测试配置
	if provider.Config().Model != "ep-20240101-xxxxx" {
		t.Errorf("Expected model=ep-20240101-xxxxx, got %s", provider.Config().Model)
	}
}

// TestMoonshotProvider 测试 Moonshot Provider
func TestMoonshotProvider(t *testing.T) {
	config := &types.ModelConfig{
		Provider: "moonshot",
		Model:    "moonshot-v1-128k",
		APIKey:   "test-key",
	}

	provider, err := NewMoonshotProvider(config)
	if err != nil {
		t.Fatalf("Failed to create Moonshot provider: %v", err)
	}

	// 测试能力
	caps := provider.Capabilities()
	if caps.MaxTokens != 128000 {
		t.Errorf("Expected MaxTokens=128000, got %d", caps.MaxTokens)
	}
}

// TestGeminiProvider 测试 Gemini Provider
func TestGeminiProvider(t *testing.T) {
	config := &types.ModelConfig{
		Provider: "gemini",
		Model:    "gemini-2.0-flash-exp",
		APIKey:   "test-key",
	}

	provider, err := NewGeminiProvider(config)
	if err != nil {
		t.Fatalf("Failed to create Gemini provider: %v", err)
	}

	// 测试配置
	if provider.Config().Model != "gemini-2.0-flash-exp" {
		t.Errorf("Expected model=gemini-2.0-flash-exp, got %s", provider.Config().Model)
	}

	// 测试能力
	caps := provider.Capabilities()
	if !caps.SupportStreaming {
		t.Error("Gemini should support streaming")
	}
	if !caps.SupportToolCalling {
		t.Error("Gemini should support tool calling")
	}
	if !caps.SupportVision {
		t.Error("Gemini should support vision")
	}
	if !caps.SupportVideo {
		t.Error("Gemini should support video (unique feature)")
	}
	if !caps.SupportPromptCache {
		t.Error("Gemini should support Context Caching")
	}
	if caps.MaxTokens != 1048576 {
		t.Errorf("Expected MaxTokens=1048576 (1M), got %d", caps.MaxTokens)
	}
	if caps.CacheMinTokens != 32768 {
		t.Errorf("Expected CacheMinTokens=32768, got %d", caps.CacheMinTokens)
	}
}

// TestCustomProvider 测试自定义 Provider（用户自带 Key 场景）
func TestCustomProvider(t *testing.T) {
	config := &types.ModelConfig{
		Provider: "custom",
		Model:    "gpt-4o",
		APIKey:   "test-key",
		BaseURL:  "https://api.example.com/v1",
	}

	provider, err := NewCustomProvider(config)
	if err != nil {
		t.Fatalf("Failed to create custom provider: %v", err)
	}

	// 测试配置
	if provider.Config().Model != "gpt-4o" {
		t.Errorf("Expected model=gpt-4o, got %s", provider.Config().Model)
	}

	// 测试能力
	caps := provider.Capabilities()
	if !caps.SupportStreaming {
		t.Error("Custom provider should support streaming")
	}
	if !caps.SupportToolCalling {
		t.Error("Custom provider should support tool calling")
	}
	if caps.ToolCallingFormat != "openai" {
		t.Errorf("Expected ToolCallingFormat=openai, got %s", caps.ToolCallingFormat)
	}
}

// TestCustomProviderValidation 测试自定义 Provider 参数验证
func TestCustomProviderValidation(t *testing.T) {
	// 缺少 BaseURL
	config := &types.ModelConfig{
		Provider: "custom",
		Model:    "gpt-4o",
		APIKey:   "test-key",
	}

	_, err := NewCustomProvider(config)
	if err == nil {
		t.Error("Expected error when BaseURL is missing")
	}

	// 缺少 APIKey
	config = &types.ModelConfig{
		Provider: "custom",
		Model:    "gpt-4o",
		BaseURL:  "https://api.example.com/v1",
	}

	_, err = NewCustomProvider(config)
	if err == nil {
		t.Error("Expected error when APIKey is missing")
	}
}

// TestProviderFactory 测试工厂模式
func TestProviderFactory(t *testing.T) {
	factory := NewMultiProviderFactory()

	testCases := []struct {
		provider string
		model    string
		apiKey   string
		baseURL  string
	}{
		{"openai", "gpt-4o", "test-key", ""},
		{"groq", "llama-3.3-70b-versatile", "test-key", ""},
		{"ollama", "llama3.2", "", ""},
		{"openrouter", "openai/gpt-4o", "test-key", ""},
		{"mistral", "mistral-large-latest", "test-key", ""},
		{"doubao", "ep-xxxxx", "test-key", ""},
		{"moonshot", "moonshot-v1-128k", "test-key", ""},
		{"gemini", "gemini-2.0-flash-exp", "test-key", ""},
		{"custom", "gpt-4o", "test-key", "https://api.example.com/v1"},
	}

	for _, tc := range testCases {
		t.Run(tc.provider, func(t *testing.T) {
			config := &types.ModelConfig{
				Provider: tc.provider,
				Model:    tc.model,
				APIKey:   tc.apiKey,
				BaseURL:  tc.baseURL,
			}

			provider, err := factory.Create(config)
			if err != nil {
				t.Fatalf("Failed to create provider %s: %v", tc.provider, err)
			}

			if provider == nil {
				t.Fatalf("Provider %s is nil", tc.provider)
			}

			// 验证基本功能
			caps := provider.Capabilities()
			if !caps.SupportStreaming {
				t.Errorf("Provider %s should support streaming", tc.provider)
			}
		})
	}
}

// TestMessageConversion 测试消息转换
func TestMessageConversion(t *testing.T) {
	config := &types.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   "test-key",
	}

	provider, err := NewOpenAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// 类型断言获取基础 Provider
	openaiProvider, ok := provider.(*OpenAIProvider)
	if !ok {
		t.Fatal("Failed to cast to OpenAIProvider")
	}
	baseProvider := openaiProvider.OpenAICompatibleProvider

	// 测试简单消息
	messages := []types.Message{
		{
			Role:    types.RoleUser,
			Content: "Hello",
		},
	}

	converted := baseProvider.convertMessages(messages)
	if len(converted) != 1 {
		t.Errorf("Expected 1 message, got %d", len(converted))
	}

	if converted[0]["role"] != "user" {
		t.Errorf("Expected role=user, got %s", converted[0]["role"])
	}

	if converted[0]["content"] != "Hello" {
		t.Errorf("Expected content=Hello, got %s", converted[0]["content"])
	}

	// 测试多模态消息
	messagesWithImage := []types.Message{
		{
			Role: types.RoleUser,
			ContentBlocks: []types.ContentBlock{
				&types.TextBlock{Text: "What's in this image?"},
				&types.ImageContent{
					Type:   "url",
					Source: "https://example.com/image.jpg",
					Detail: "high",
				},
			},
		},
	}

	converted = baseProvider.convertMessages(messagesWithImage)
	if len(converted) != 1 {
		t.Errorf("Expected 1 message, got %d", len(converted))
	}

	content, ok := converted[0]["content"].([]map[string]any)
	if !ok {
		t.Fatal("Expected content to be array")
	}

	if len(content) != 2 {
		t.Errorf("Expected 2 content blocks, got %d", len(content))
	}

	// 验证文本块
	if content[0]["type"] != "text" {
		t.Errorf("Expected first block type=text, got %s", content[0]["type"])
	}

	// 验证图片块
	if content[1]["type"] != "image_url" {
		t.Errorf("Expected second block type=image_url, got %s", content[1]["type"])
	}
}

// TestReasoningModelDetection 测试推理模型检测
func TestReasoningModelDetection(t *testing.T) {
	testCases := []struct {
		model       string
		isReasoning bool
	}{
		{"gpt-4o", false},
		{"gpt-4", false},
		{"o1-preview", true},
		{"o1-mini", true},
		{"o3-mini", true},
		{"claude-3-opus", false},
	}

	for _, tc := range testCases {
		t.Run(tc.model, func(t *testing.T) {
			result := isReasoningModel(tc.model)
			if result != tc.isReasoning {
				t.Errorf("Model %s: expected isReasoning=%v, got %v", tc.model, tc.isReasoning, result)
			}
		})
	}
}

// TestToolConversion 测试工具转换
func TestToolConversion(t *testing.T) {
	config := &types.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   "test-key",
	}

	provider, err := NewOpenAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// 类型断言获取基础 Provider
	openaiProvider, ok := provider.(*OpenAIProvider)
	if !ok {
		t.Fatal("Failed to cast to OpenAIProvider")
	}
	baseProvider := openaiProvider.OpenAICompatibleProvider

	tools := []ToolSchema{
		{
			Name:        "get_weather",
			Description: "Get the current weather",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"location": map[string]any{
						"type":        "string",
						"description": "The city name",
					},
				},
				"required": []string{"location"},
			},
		},
	}

	converted := baseProvider.convertTools(tools)
	if len(converted) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(converted))
	}

	if converted[0]["type"] != "function" {
		t.Errorf("Expected type=function, got %s", converted[0]["type"])
	}

	function := converted[0]["function"].(map[string]any)
	if function["name"] != "get_weather" {
		t.Errorf("Expected name=get_weather, got %s", function["name"])
	}
}

// TestContentBlockHelper 测试内容块辅助函数
func TestContentBlockHelper(t *testing.T) {
	helper := types.ContentBlockHelper{}

	blocks := []types.ContentBlock{
		&types.TextBlock{Text: "Hello "},
		&types.TextBlock{Text: "World"},
		&types.ImageContent{
			Type:   "url",
			Source: "https://example.com/image.jpg",
		},
	}

	// 测试提取文本
	text := helper.ExtractText(blocks)
	if text != "Hello World" {
		t.Errorf("Expected 'Hello World', got '%s'", text)
	}

	// 测试检测多模态
	hasMultimodal := helper.HasMultimodal(blocks)
	if !hasMultimodal {
		t.Error("Should detect multimodal content")
	}

	// 测试获取媒体类型
	mediaTypes := helper.GetMediaTypes(blocks)
	if len(mediaTypes) != 1 {
		t.Errorf("Expected 1 media type, got %d", len(mediaTypes))
	}
	if mediaTypes[0] != "image" {
		t.Errorf("Expected media type 'image', got '%s'", mediaTypes[0])
	}
}

// BenchmarkProviderCreation 性能测试：Provider 创建
func BenchmarkProviderCreation(b *testing.B) {
	config := &types.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   "test-key",
	}

	factory := NewMultiProviderFactory()

	for b.Loop() {
		_, _ = factory.Create(config)
	}
}

// BenchmarkMessageConversion 性能测试：消息转换
func BenchmarkMessageConversion(b *testing.B) {
	config := &types.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   "test-key",
	}

	provider, _ := NewOpenAIProvider(config)
	openaiProvider := provider.(*OpenAIProvider)
	baseProvider := openaiProvider.OpenAICompatibleProvider

	messages := []types.Message{
		{Role: types.RoleUser, Content: "Hello"},
		{Role: types.RoleAssistant, Content: "Hi there!"},
		{Role: types.RoleUser, Content: "How are you?"},
	}

	for b.Loop() {
		_ = baseProvider.convertMessages(messages)
	}
}

// TestStreamAccumulator 测试流式累加器
func TestStreamAccumulator(t *testing.T) {
	acc := types.NewStreamAccumulator()

	// 添加文本块
	acc.AddChunk(&types.StreamChunk{
		Type:  types.ChunkTypeText,
		Delta: "Hello ",
	})
	acc.AddChunk(&types.StreamChunk{
		Type:  types.ChunkTypeText,
		Delta: "World",
	})

	if acc.Content != "Hello World" {
		t.Errorf("Expected 'Hello World', got '%s'", acc.Content)
	}

	// 添加推理块
	acc.AddChunk(&types.StreamChunk{
		Type: types.ChunkTypeReasoning,
		Reasoning: &types.ReasoningTrace{
			Step:    1,
			Thought: "Let me think...",
		},
	})

	if len(acc.Reasoning) != 1 {
		t.Errorf("Expected 1 reasoning trace, got %d", len(acc.Reasoning))
	}

	// 添加完成块
	acc.AddChunk(&types.StreamChunk{
		Type:         types.ChunkTypeDone,
		FinishReason: "stop",
	})

	if !acc.IsComplete() {
		t.Error("Accumulator should be complete")
	}

	// 转换为消息
	msg := acc.ToMessage()
	if msg.Role != types.RoleAssistant {
		t.Errorf("Expected role=assistant, got %s", msg.Role)
	}
	if msg.Content != "Hello World" {
		t.Errorf("Expected content='Hello World', got '%s'", msg.Content)
	}
}
