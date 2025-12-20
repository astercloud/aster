package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/astercloud/aster/pkg/types"
	"github.com/astercloud/aster/pkg/util"
)

const (
	// GeminiAPIBaseURL Gemini API 基础 URL
	GeminiAPIBaseURL = "https://generativelanguage.googleapis.com/v1beta"
)

// GeminiProvider Google Gemini 提供商
// Gemini 使用专有的 Content/Parts 格式，不兼容 OpenAI
type GeminiProvider struct {
	config       *types.ModelConfig
	baseURL      string
	httpClient   *http.Client
	systemPrompt string
}

// GeminiContent Gemini 消息内容格式
type GeminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []GeminiPart `json:"parts"`
}

// GeminiPart Gemini 内容部分
type GeminiPart struct {
	// 文本内容
	Text string `json:"text,omitempty"`

	// 内联数据（图片、音频等）
	InlineData *GeminiBlob `json:"inlineData,omitempty"`

	// 函数调用
	FunctionCall *GeminiFunctionCall `json:"functionCall,omitempty"`

	// 函数响应
	FunctionResponse *GeminiFunctionResponse `json:"functionResponse,omitempty"`
}

// GeminiBlob 二进制数据
type GeminiBlob struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"` // base64 编码
}

// GeminiFunctionCall 函数调用
type GeminiFunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

// GeminiFunctionResponse 函数响应
type GeminiFunctionResponse struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

// GeminiTool Gemini 工具定义
type GeminiTool struct {
	FunctionDeclarations []GeminiFunctionDeclaration `json:"functionDeclarations"`
}

// GeminiFunctionDeclaration 函数声明
type GeminiFunctionDeclaration struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// NewGeminiProvider 创建 Gemini 提供商
func NewGeminiProvider(config *types.ModelConfig) (Provider, error) {
	if config.APIKey == "" {
		return nil, errors.New("gemini: API key is required")
	}

	// 设置默认模型
	if config.Model == "" {
		config.Model = "gemini-2.0-flash-exp" // 最新的 Gemini 2.0 Flash
	}

	// 使用配置中的 BaseURL，或使用默认值
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = GeminiAPIBaseURL
	}

	return &GeminiProvider{
		config:     config,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}, nil
}

// Stream 实现流式对话
func (p *GeminiProvider) Stream(
	ctx context.Context,
	messages []types.Message,
	opts *StreamOptions,
) (<-chan StreamChunk, error) {
	// 构建请求
	requestBody := p.buildRequest(messages, opts, true)

	// 发送 HTTP 请求
	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse&key=%s",
		p.baseURL, p.config.Model, p.config.APIKey)

	// 使用确定性序列化以优化 KV-Cache 命中率
	bodyBytes, err := util.MarshalDeterministic(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("gemini API error: %d - %s", resp.StatusCode, string(body))
	}

	// 创建流式响应 channel
	chunks := make(chan StreamChunk, 10)

	// 在 goroutine 中解析 SSE 流
	go p.parseSSEStream(resp.Body, chunks)

	return chunks, nil
}

// Complete 实现非流式对话
func (p *GeminiProvider) Complete(
	ctx context.Context,
	messages []types.Message,
	opts *StreamOptions,
) (*CompleteResponse, error) {
	// 构建请求
	requestBody := p.buildRequest(messages, opts, false)

	// 发送 HTTP 请求
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s",
		p.baseURL, p.config.Model, p.config.APIKey)

	// 使用确定性序列化以优化 KV-Cache 命中率
	bodyBytes, err := util.MarshalDeterministic(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini API error: %d - %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var apiResp map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 解析消息
	message, err := p.parseCompleteResponse(apiResp)
	if err != nil {
		return nil, err
	}

	// 解析 usage
	usage := p.parseUsage(apiResp)

	return &CompleteResponse{
		Message: message,
		Usage:   usage,
	}, nil
}

// buildRequest 构建请求体
func (p *GeminiProvider) buildRequest(
	messages []types.Message,
	opts *StreamOptions,
	stream bool,
) map[string]any {
	requestBody := make(map[string]any)

	// 转换消息为 Gemini 格式
	contents := p.convertMessages(messages)
	requestBody["contents"] = contents

	// 添加系统指令
	if p.systemPrompt != "" || (opts != nil && opts.System != "") {
		systemInstruction := p.systemPrompt
		if opts != nil && opts.System != "" {
			systemInstruction = opts.System
		}
		requestBody["systemInstruction"] = map[string]any{
			"parts": []map[string]any{
				{"text": systemInstruction},
			},
		}
	}

	// 生成配置
	generationConfig := make(map[string]any)
	if opts != nil {
		if opts.MaxTokens > 0 {
			generationConfig["maxOutputTokens"] = opts.MaxTokens
		}
		if opts.Temperature > 0 {
			generationConfig["temperature"] = opts.Temperature
		}
	}
	if len(generationConfig) > 0 {
		requestBody["generationConfig"] = generationConfig
	}

	// 添加工具
	if opts != nil && len(opts.Tools) > 0 {
		requestBody["tools"] = []GeminiTool{p.convertTools(opts.Tools)}
	}

	return requestBody
}

// convertMessages 转换消息格式为 Gemini 格式
func (p *GeminiProvider) convertMessages(messages []types.Message) []GeminiContent {
	result := make([]GeminiContent, 0, len(messages))

	for _, msg := range messages {
		// 跳过 system 消息（在 systemInstruction 中处理）
		if msg.Role == types.RoleSystem {
			continue
		}

		// 转换角色
		role := "user"
		if msg.Role == types.RoleAssistant {
			role = "model" // Gemini 使用 "model" 而不是 "assistant"
		}

		content := GeminiContent{
			Role:  role,
			Parts: make([]GeminiPart, 0),
		}

		// 处理内容
		if len(msg.ContentBlocks) > 0 {
			// 复杂格式：ContentBlocks
			for _, block := range msg.ContentBlocks {
				switch b := block.(type) {
				case *types.TextBlock:
					content.Parts = append(content.Parts, GeminiPart{
						Text: b.Text,
					})

				case *types.ImageContent:
					// 图片内容
					switch b.Type {
					case "base64":
						content.Parts = append(content.Parts, GeminiPart{
							InlineData: &GeminiBlob{
								MimeType: b.MimeType,
								Data:     b.Source,
							},
						})
					case "url":
						// Gemini 不直接支持 URL，需要先下载并转换为 base64
						// 这里简化处理，实际应用中需要下载图片
						content.Parts = append(content.Parts, GeminiPart{
							Text: fmt.Sprintf("[图片: %s]", b.Source),
						})
					}

				case *types.AudioContent:
					// 音频内容
					switch b.Type {
					case "base64":
						content.Parts = append(content.Parts, GeminiPart{
							InlineData: &GeminiBlob{
								MimeType: b.MimeType,
								Data:     b.Source,
							},
						})
					}

				case *types.VideoContent:
					// 视频内容
					if b.Type == "base64" {
						content.Parts = append(content.Parts, GeminiPart{
							InlineData: &GeminiBlob{
								MimeType: b.MimeType,
								Data:     b.Source,
							},
						})
					}

				case *types.ToolUseBlock:
					// 工具调用
					content.Parts = append(content.Parts, GeminiPart{
						FunctionCall: &GeminiFunctionCall{
							Name: b.Name,
							Args: b.Input,
						},
					})

				case *types.ToolResultBlock:
					// 工具结果
					content.Parts = append(content.Parts, GeminiPart{
						FunctionResponse: &GeminiFunctionResponse{
							Name: b.ToolUseID, // 使用 ID 作为名称
							Response: map[string]any{
								"content": b.Content,
							},
						},
					})
				}
			}
		} else {
			// 简单格式：纯文本
			content.Parts = append(content.Parts, GeminiPart{
				Text: msg.Content,
			})
		}

		result = append(result, content)
	}

	return result
}

// convertTools 转换工具定义为 Gemini 格式
func (p *GeminiProvider) convertTools(tools []ToolSchema) GeminiTool {
	declarations := make([]GeminiFunctionDeclaration, 0, len(tools))

	for _, tool := range tools {
		declarations = append(declarations, GeminiFunctionDeclaration{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.InputSchema,
			// TODO: Gemini API 暂不支持 input_examples，待官方支持后启用
			// 参考: https://ai.google.dev/api/caching
			// 实现参考: pkg/provider/anthropic.go buildRequest() 中的 InputExamples 处理
		})
	}

	return GeminiTool{
		FunctionDeclarations: declarations,
	}
}

// parseSSEStream 解析 SSE 流
func (p *GeminiProvider) parseSSEStream(body io.ReadCloser, chunks chan<- StreamChunk) {
	defer func() { _ = body.Close() }()
	defer close(chunks)

	scanner := bufio.NewScanner(body)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()

		// SSE 格式: "data: {json}"
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		data = strings.TrimSpace(data)

		// 解析 JSON
		var chunk map[string]any
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		// 解析 chunk 并转换为 StreamChunk
		streamChunks := p.parseStreamChunk(chunk)
		for _, sc := range streamChunks {
			chunks <- sc
		}
	}

	if err := scanner.Err(); err != nil {
		chunks <- StreamChunk{
			Type: string(ChunkTypeError),
			Error: &StreamError{
				Code:    "stream_error",
				Message: err.Error(),
			},
		}
	}
}

// parseStreamChunk 解析单个流式 chunk
func (p *GeminiProvider) parseStreamChunk(chunk map[string]any) []StreamChunk {
	result := make([]StreamChunk, 0)

	// 获取 candidates
	candidates, ok := chunk["candidates"].([]any)
	if !ok || len(candidates) == 0 {
		return result
	}

	candidate := candidates[0].(map[string]any)

	// 获取 content
	content, ok := candidate["content"].(map[string]any)
	if !ok {
		// 检查是否完成
		if finishReason, ok := candidate["finishReason"].(string); ok {
			result = append(result, StreamChunk{
				Type:         string(ChunkTypeDone),
				FinishReason: strings.ToLower(finishReason),
			})
		}
		return result
	}

	// 获取 parts
	parts, ok := content["parts"].([]any)
	if !ok {
		return result
	}

	// 解析每个 part
	for _, partData := range parts {
		part := partData.(map[string]any)

		// 文本内容
		if text, ok := part["text"].(string); ok && text != "" {
			result = append(result, StreamChunk{
				Type:      string(ChunkTypeText),
				TextDelta: text,
				Delta:     text,
			})
		}

		// 函数调用
		if functionCall, ok := part["functionCall"].(map[string]any); ok {
			name := functionCall["name"].(string)
			args := functionCall["args"].(map[string]any)

			// 序列化参数
			argsJSON, _ := json.Marshal(args)

			result = append(result, StreamChunk{
				Type: string(ChunkTypeToolCall),
				ToolCall: &ToolCallDelta{
					Index:          0,
					Name:           name,
					ArgumentsDelta: string(argsJSON),
				},
			})
		}
	}

	// 解析 usage（如果有）
	if usageData, ok := chunk["usageMetadata"].(map[string]any); ok {
		usage := p.parseUsageFromMap(usageData)
		result = append(result, StreamChunk{
			Type:  string(ChunkTypeUsage),
			Usage: usage,
		})
	}

	return result
}

// parseCompleteResponse 解析完整响应
func (p *GeminiProvider) parseCompleteResponse(apiResp map[string]any) (types.Message, error) {
	candidates, ok := apiResp["candidates"].([]any)
	if !ok || len(candidates) == 0 {
		return types.Message{}, errors.New("no candidates in response")
	}

	candidate := candidates[0].(map[string]any)
	content, ok := candidate["content"].(map[string]any)
	if !ok {
		return types.Message{}, errors.New("no content in candidate")
	}

	parts, ok := content["parts"].([]any)
	if !ok {
		return types.Message{}, errors.New("no parts in content")
	}

	// 构建消息
	message := types.Message{
		Role: types.RoleAssistant,
	}

	blocks := make([]types.ContentBlock, 0)
	textParts := make([]string, 0)

	for _, partData := range parts {
		part := partData.(map[string]any)

		// 文本内容
		if text, ok := part["text"].(string); ok {
			textParts = append(textParts, text)
		}

		// 函数调用
		if functionCall, ok := part["functionCall"].(map[string]any); ok {
			name := functionCall["name"].(string)
			args := functionCall["args"].(map[string]any)

			blocks = append(blocks, &types.ToolUseBlock{
				ID:    name, // Gemini 不返回 ID，使用名称
				Name:  name,
				Input: args,
			})
		}
	}

	// 如果有工具调用，使用 ContentBlocks
	if len(blocks) > 0 {
		// 添加文本块
		if len(textParts) > 0 {
			blocks = append([]types.ContentBlock{
				&types.TextBlock{Text: strings.Join(textParts, "")},
			}, blocks...)
		}
		message.ContentBlocks = blocks
	} else {
		// 纯文本
		message.Content = strings.Join(textParts, "")
	}

	return message, nil
}

// parseUsage 解析 usage 信息
func (p *GeminiProvider) parseUsage(apiResp map[string]any) *TokenUsage {
	usageData, ok := apiResp["usageMetadata"].(map[string]any)
	if !ok {
		return nil
	}
	return p.parseUsageFromMap(usageData)
}

// parseUsageFromMap 从 map 解析 usage
func (p *GeminiProvider) parseUsageFromMap(usageData map[string]any) *TokenUsage {
	usage := &TokenUsage{}

	if promptTokens, ok := usageData["promptTokenCount"].(float64); ok {
		usage.InputTokens = int64(promptTokens)
	}
	if candidatesTokens, ok := usageData["candidatesTokenCount"].(float64); ok {
		usage.OutputTokens = int64(candidatesTokens)
	}
	if totalTokens, ok := usageData["totalTokenCount"].(float64); ok {
		usage.TotalTokens = int64(totalTokens)
	}

	// Gemini 支持 Context Caching
	if cachedTokens, ok := usageData["cachedContentTokenCount"].(float64); ok {
		usage.CachedTokens = int64(cachedTokens)
	}

	return usage
}

// Config 返回配置
func (p *GeminiProvider) Config() *types.ModelConfig {
	return p.config
}

// Capabilities 返回能力
func (p *GeminiProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		SupportToolCalling:  true,
		SupportSystemPrompt: true,
		SupportStreaming:    true,
		SupportVision:       true,
		SupportAudio:        true,
		SupportVideo:        true, // Gemini 独特支持视频
		SupportReasoning:    false,
		SupportPromptCache:  true, // Context Caching
		SupportJSONMode:     true,
		SupportFunctionCall: true,
		MaxTokens:           1048576, // Gemini 2.0 支持 1M tokens
		ToolCallingFormat:   "gemini",
		CacheMinTokens:      32768, // 32K 最小缓存
	}
}

// SetSystemPrompt 设置系统提示词
func (p *GeminiProvider) SetSystemPrompt(prompt string) error {
	p.systemPrompt = prompt
	return nil
}

// GetSystemPrompt 获取系统提示词
func (p *GeminiProvider) GetSystemPrompt() string {
	return p.systemPrompt
}

// Close 关闭连接
func (p *GeminiProvider) Close() error {
	return nil
}

// GeminiFactory Gemini 工厂
type GeminiFactory struct{}

// Create 创建 Gemini 提供商
func (f *GeminiFactory) Create(config *types.ModelConfig) (Provider, error) {
	return NewGeminiProvider(config)
}
