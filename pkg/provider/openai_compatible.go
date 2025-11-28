package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/astercloud/aster/pkg/types"
	"github.com/astercloud/aster/pkg/util"
)

// OpenAICompatibleProvider OpenAI 兼容格式的通用 Provider
// 适用于 OpenAI, Groq, Ollama, Fireworks, Cerebras, DeepInfra, xAI 等
type OpenAICompatibleProvider struct {
	config       *types.ModelConfig
	baseURL      string
	providerName string
	httpClient   *http.Client
	systemPrompt string

	// 能力定义
	capabilities ProviderCapabilities

	// 可选配置
	options OpenAICompatibleOptions
}

// OpenAICompatibleOptions OpenAI 兼容 Provider 的可选配置
type OpenAICompatibleOptions struct {
	// 是否需要 API Key
	RequireAPIKey bool

	// 默认模型名称
	DefaultModel string

	// 是否支持推理模型
	SupportReasoning bool

	// 是否支持 Prompt Caching
	SupportPromptCache bool

	// 是否支持多模态
	SupportVision bool
	SupportAudio  bool

	// 超时配置
	Timeout time.Duration

	// 重试配置
	MaxRetries int
	RetryDelay time.Duration
	RetryOn429 bool // 是否在 429 时重试
	RetryOn500 bool // 是否在 5xx 时重试

	// 自定义请求头
	CustomHeaders map[string]string
}

// NewOpenAICompatibleProvider 创建 OpenAI 兼容 Provider
func NewOpenAICompatibleProvider(
	config *types.ModelConfig,
	baseURL string,
	providerName string,
	options *OpenAICompatibleOptions,
) (*OpenAICompatibleProvider, error) {
	// 设置默认选项
	if options == nil {
		options = &OpenAICompatibleOptions{
			RequireAPIKey: true,
			Timeout:       120 * time.Second,
			MaxRetries:    3,
			RetryDelay:    1 * time.Second,
			RetryOn429:    true,
			RetryOn500:    true,
		}
	}

	// 验证 API Key
	if options.RequireAPIKey && config.APIKey == "" {
		return nil, fmt.Errorf("%s: API key is required", providerName)
	}

	// 创建 HTTP 客户端
	httpClient := &http.Client{
		Timeout: options.Timeout,
	}

	// 设置默认模型
	if config.Model == "" && options.DefaultModel != "" {
		config.Model = options.DefaultModel
	}

	return &OpenAICompatibleProvider{
		config:       config,
		baseURL:      baseURL,
		providerName: providerName,
		httpClient:   httpClient,
		options:      *options,
		capabilities: buildCapabilities(options),
	}, nil
}

// buildCapabilities 构建能力定义
func buildCapabilities(options *OpenAICompatibleOptions) ProviderCapabilities {
	return ProviderCapabilities{
		SupportToolCalling:  true,
		SupportSystemPrompt: true,
		SupportStreaming:    true,
		SupportVision:       options.SupportVision,
		SupportAudio:        options.SupportAudio,
		SupportReasoning:    options.SupportReasoning,
		SupportPromptCache:  options.SupportPromptCache,
		SupportJSONMode:     true,
		SupportFunctionCall: true,
		MaxTokens:           128000, // 默认值，可被具体 Provider 覆盖
		ToolCallingFormat:   "openai",
	}
}

// Stream 实现流式对话
func (p *OpenAICompatibleProvider) Stream(
	ctx context.Context,
	messages []types.Message,
	opts *StreamOptions,
) (<-chan StreamChunk, error) {
	// 构建请求体
	requestBody := p.buildRequest(messages, opts, true)

	// 发送 HTTP 请求
	req, err := p.createHTTPRequest(ctx, requestBody)
	if err != nil {
		return nil, err
	}

	// 发送请求（带重试）
	resp, err := p.doRequestWithRetry(req)
	if err != nil {
		return nil, err
	}

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("%s API error: %d - %s", p.providerName, resp.StatusCode, string(body))
	}

	// 创建流式响应 channel
	chunks := make(chan StreamChunk, 10)

	// 在 goroutine 中解析 SSE 流
	go p.parseSSEStream(resp.Body, chunks)

	return chunks, nil
}

// Complete 实现非流式对话
func (p *OpenAICompatibleProvider) Complete(
	ctx context.Context,
	messages []types.Message,
	opts *StreamOptions,
) (*CompleteResponse, error) {
	// 构建请求体
	requestBody := p.buildRequest(messages, opts, false)

	// 发送 HTTP 请求
	req, err := p.createHTTPRequest(ctx, requestBody)
	if err != nil {
		return nil, err
	}

	// 发送请求（带重试）
	resp, err := p.doRequestWithRetry(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%s API error: %d - %s", p.providerName, resp.StatusCode, string(body))
	}

	// 解析响应
	var apiResp map[string]interface{}
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
func (p *OpenAICompatibleProvider) buildRequest(
	messages []types.Message,
	opts *StreamOptions,
	stream bool,
) map[string]interface{} {
	requestBody := map[string]interface{}{
		"model":  p.config.Model,
		"stream": stream,
	}

	// 转换消息格式
	requestBody["messages"] = p.convertMessages(messages)

	// 添加系统提示词
	if p.systemPrompt != "" || (opts != nil && opts.System != "") {
		systemMsg := p.systemPrompt
		if opts != nil && opts.System != "" {
			systemMsg = opts.System
		}
		// 将 system 作为第一条消息
		msgs := requestBody["messages"].([]map[string]interface{})
		msgs = append([]map[string]interface{}{
			{
				"role":    "system",
				"content": systemMsg,
			},
		}, msgs...)
		requestBody["messages"] = msgs
	}

	// 添加可选参数
	if opts != nil {
		if opts.MaxTokens > 0 {
			requestBody["max_tokens"] = opts.MaxTokens
		}
		// 推理模型不支持 temperature
		if opts.Temperature > 0 && !p.isReasoningModel(p.config.Model) {
			requestBody["temperature"] = opts.Temperature
		}
		// 添加工具
		if len(opts.Tools) > 0 {
			requestBody["tools"] = p.convertTools(opts.Tools)
		}
	}

	return requestBody
}

// convertMessages 转换消息格式为 OpenAI 格式
func (p *OpenAICompatibleProvider) convertMessages(messages []types.Message) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(messages))

	for _, msg := range messages {
		// 跳过 system 消息（在 buildRequest 中单独处理）
		if msg.Role == types.RoleSystem {
			continue
		}

		// 检查 ContentBlocks 中是否包含特殊块类型
		var toolUseBlocks []*types.ToolUseBlock
		var toolResultBlocks []*types.ToolResultBlock
		var otherBlocks []types.ContentBlock

		for _, block := range msg.ContentBlocks {
			switch b := block.(type) {
			case *types.ToolUseBlock:
				toolUseBlocks = append(toolUseBlocks, b)
			case *types.ToolResultBlock:
				toolResultBlocks = append(toolResultBlocks, b)
			default:
				otherBlocks = append(otherBlocks, block)
			}
		}

		// 处理包含 ToolResultBlock 的消息
		// OpenAI API 要求每个工具结果作为独立的 role: "tool" 消息
		if len(toolResultBlocks) > 0 {
			for _, tr := range toolResultBlocks {
				toolMsg := map[string]interface{}{
					"role":         "tool",
					"tool_call_id": tr.ToolUseID,
					"content":      tr.Content,
				}
				result = append(result, toolMsg)
			}
			continue // 跳过常规消息处理
		}

		msgMap := map[string]interface{}{
			"role": string(msg.Role),
		}

		// 处理内容（排除 ToolUseBlock，它们会被单独处理）
		if len(otherBlocks) > 0 {
			content := p.convertContentBlocks(otherBlocks)
			msgMap["content"] = content
		} else if msg.Content != "" {
			msgMap["content"] = msg.Content
		}

		// 处理 assistant 消息的工具调用
		// 优先从 ToolCalls 字段获取，如果没有则从 ContentBlocks 中的 ToolUseBlock 提取
		if msg.Role == types.RoleAssistant {
			var toolCalls []map[string]interface{}

			// 从 msg.ToolCalls 获取
			for _, tc := range msg.ToolCalls {
				argsJSON, _ := json.Marshal(tc.Arguments)
				toolCalls = append(toolCalls, map[string]interface{}{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]interface{}{
						"name":      tc.Name,
						"arguments": string(argsJSON),
					},
				})
			}

			// 从 ContentBlocks 中的 ToolUseBlock 提取
			for _, tu := range toolUseBlocks {
				argsJSON, _ := json.Marshal(tu.Input)
				toolCalls = append(toolCalls, map[string]interface{}{
					"id":   tu.ID,
					"type": "function",
					"function": map[string]interface{}{
						"name":      tu.Name,
						"arguments": string(argsJSON),
					},
				})
			}

			if len(toolCalls) > 0 {
				msgMap["tool_calls"] = toolCalls

				// 注意：OpenRouter 转发到 Anthropic 时，要求非最后一条 assistant 消息必须有 content
				// 因此这里不删除 content 字段，即使为空也保留空字符串
				if msgMap["content"] == nil {
					msgMap["content"] = ""
				}
			}
		}

		// 处理 tool 消息的工具调用 ID
		if msg.Role == types.RoleTool && msg.ToolCallID != "" {
			msgMap["tool_call_id"] = msg.ToolCallID
		}

		result = append(result, msgMap)
	}

	return result
}

// convertContentBlocks 转换 ContentBlocks 为 OpenAI 格式
func (p *OpenAICompatibleProvider) convertContentBlocks(blocks []types.ContentBlock) interface{} {
	// 如果只有一个文本块，直接返回字符串
	if len(blocks) == 1 {
		if tb, ok := blocks[0].(*types.TextBlock); ok {
			return tb.Text
		}
	}

	// 多个块或包含多模态内容，返回数组
	content := make([]map[string]interface{}, 0, len(blocks))
	for _, block := range blocks {
		switch b := block.(type) {
		case *types.TextBlock:
			content = append(content, map[string]interface{}{
				"type": "text",
				"text": b.Text,
			})

		case *types.ImageContent:
			imageBlock := map[string]interface{}{
				"type": "image_url",
			}
			switch b.Type {
			case "url":
				imageBlock["image_url"] = map[string]interface{}{
					"url": b.Source,
				}
				if b.Detail != "" {
					imageBlock["image_url"].(map[string]interface{})["detail"] = b.Detail
				}
			case "base64":
				// base64 格式
				dataURL := fmt.Sprintf("data:%s;base64,%s", b.MimeType, b.Source)
				imageBlock["image_url"] = map[string]interface{}{
					"url": dataURL,
				}
			}
			content = append(content, imageBlock)

		case *types.ToolUseBlock:
			// 工具调用已在消息中处理
			continue

		case *types.ToolResultBlock:
			// 工具结果作为独立消息
			continue
		}
	}

	return content
}

// convertTools 转换工具定义为 OpenAI 格式
func (p *OpenAICompatibleProvider) convertTools(tools []ToolSchema) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(tools))
	for _, tool := range tools {
		result = append(result, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  tool.InputSchema,
			},
			// TODO: OpenAI API 暂不支持 input_examples，待官方支持后启用
			// 参考: https://platform.openai.com/docs/api-reference/chat/create
			// 实现参考: pkg/provider/anthropic.go buildRequest() 中的 InputExamples 处理
			// if len(tool.InputExamples) > 0 {
			//     function["input_examples"] = tool.InputExamples
			// }
		})
	}
	return result
}

// createHTTPRequest 创建 HTTP 请求
func (p *OpenAICompatibleProvider) createHTTPRequest(ctx context.Context, requestBody map[string]interface{}) (*http.Request, error) {
	// 使用确定性序列化以优化 KV-Cache 命中率
	bodyBytes, err := util.MarshalDeterministic(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := p.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}

	// 自定义请求头
	for key, value := range p.options.CustomHeaders {
		req.Header.Set(key, value)
	}

	return req, nil
}

// doRequestWithRetry 发送 HTTP 请求（带重试）
func (p *OpenAICompatibleProvider) doRequestWithRetry(req *http.Request) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= p.options.MaxRetries; attempt++ {
		// 复制请求体（因为 Body 会被消耗）
		var bodyBytes []byte
		if req.Body != nil {
			bodyBytes, _ = io.ReadAll(req.Body)
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		resp, err := p.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < p.options.MaxRetries {
				time.Sleep(p.options.RetryDelay * time.Duration(attempt+1))
				// 恢复 Body
				if bodyBytes != nil {
					req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				}
				continue
			}
			break
		}

		// 检查是否需要重试
		shouldRetry := false
		if resp.StatusCode == 429 && p.options.RetryOn429 {
			shouldRetry = true
		} else if resp.StatusCode >= 500 && p.options.RetryOn500 {
			shouldRetry = true
		}

		if shouldRetry && attempt < p.options.MaxRetries {
			_ = resp.Body.Close()
			time.Sleep(p.options.RetryDelay * time.Duration(attempt+1))
			// 恢复 Body
			if bodyBytes != nil {
				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// parseSSEStream 解析 SSE 流
func (p *OpenAICompatibleProvider) parseSSEStream(body io.ReadCloser, chunks chan<- StreamChunk) {
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

		// 结束标记
		if data == "[DONE]" {
			chunks <- StreamChunk{
				Type:         string(ChunkTypeDone),
				FinishReason: "stop",
			}
			return
		}

		// 解析 JSON
		var chunk map[string]interface{}
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
func (p *OpenAICompatibleProvider) parseStreamChunk(chunk map[string]interface{}) []StreamChunk {
	result := make([]StreamChunk, 0)

	// 获取 choices
	choices, ok := chunk["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return result
	}

	choice := choices[0].(map[string]interface{})
	delta, ok := choice["delta"].(map[string]interface{})
	if !ok {
		// 检查是否完成
		if finishReason, ok := choice["finish_reason"].(string); ok && finishReason != "" {
			result = append(result, StreamChunk{
				Type:         string(ChunkTypeDone),
				FinishReason: finishReason,
			})
		}
		return result
	}

	// 解析文本内容
	if content, ok := delta["content"].(string); ok && content != "" {
		result = append(result, StreamChunk{
			Type:      string(ChunkTypeText),
			TextDelta: content,
			Delta:     content, // 兼容旧版
		})
	}

	// 解析工具调用
	if toolCalls, ok := delta["tool_calls"].([]interface{}); ok {
		for _, tc := range toolCalls {
			toolCall := tc.(map[string]interface{})
			index := int(toolCall["index"].(float64))

			tcDelta := &ToolCallDelta{
				Index: index,
			}

			if id, ok := toolCall["id"].(string); ok {
				tcDelta.ID = id
			}
			if tcType, ok := toolCall["type"].(string); ok {
				tcDelta.Type = tcType
			}
			if function, ok := toolCall["function"].(map[string]interface{}); ok {
				if name, ok := function["name"].(string); ok {
					tcDelta.Name = name
				}
				if args, ok := function["arguments"].(string); ok {
					tcDelta.ArgumentsDelta = args
				}
			}

			result = append(result, StreamChunk{
				Type:     string(ChunkTypeToolCall),
				ToolCall: tcDelta,
			})
		}
	}

	// 解析 usage（如果有）
	if usageData, ok := chunk["usage"].(map[string]interface{}); ok {
		usage := p.parseUsageFromMap(usageData)
		result = append(result, StreamChunk{
			Type:  string(ChunkTypeUsage),
			Usage: usage,
		})
	}

	return result
}

// parseCompleteResponse 解析完整响应
func (p *OpenAICompatibleProvider) parseCompleteResponse(apiResp map[string]interface{}) (types.Message, error) {
	choices, ok := apiResp["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return types.Message{}, fmt.Errorf("no choices in response")
	}

	choice := choices[0].(map[string]interface{})
	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		return types.Message{}, fmt.Errorf("no message in choice")
	}

	// 解析角色
	role := types.RoleAssistant
	if r, ok := message["role"].(string); ok {
		role = types.Role(r)
	}

	// 解析内容
	result := types.Message{
		Role: role,
	}

	if content, ok := message["content"].(string); ok && content != "" {
		result.Content = content
	}

	// 构建 ContentBlocks
	blocks := make([]types.ContentBlock, 0)

	// 添加文本块
	if result.Content != "" {
		blocks = append(blocks, &types.TextBlock{Text: result.Content})
	}

	// 解析工具调用
	if toolCalls, ok := message["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
		// 添加工具调用块
		for _, tc := range toolCalls {
			toolCall := tc.(map[string]interface{})
			function := toolCall["function"].(map[string]interface{})

			// 解析参数
			var args map[string]interface{}
			if argsStr, ok := function["arguments"].(string); ok {
				_ = json.Unmarshal([]byte(argsStr), &args)
			}

			blocks = append(blocks, &types.ToolUseBlock{
				ID:    toolCall["id"].(string),
				Name:  function["name"].(string),
				Input: args,
			})
		}
	}

	// 设置 ContentBlocks（即使只有文本也设置）
	if len(blocks) > 0 {
		result.ContentBlocks = blocks
	}

	return result, nil
}

// parseUsage 解析 usage 信息
func (p *OpenAICompatibleProvider) parseUsage(apiResp map[string]interface{}) *TokenUsage {
	usageData, ok := apiResp["usage"].(map[string]interface{})
	if !ok {
		return nil
	}
	return p.parseUsageFromMap(usageData)
}

// parseUsageFromMap 从 map 解析 usage
func (p *OpenAICompatibleProvider) parseUsageFromMap(usageData map[string]interface{}) *TokenUsage {
	usage := &TokenUsage{}

	if promptTokens, ok := usageData["prompt_tokens"].(float64); ok {
		usage.InputTokens = int64(promptTokens)
	}
	if completionTokens, ok := usageData["completion_tokens"].(float64); ok {
		usage.OutputTokens = int64(completionTokens)
	}
	if totalTokens, ok := usageData["total_tokens"].(float64); ok {
		usage.TotalTokens = int64(totalTokens)
	}

	// 推理 tokens (o1/o3 models)
	if reasoningTokens, ok := usageData["completion_tokens_details"].(map[string]interface{}); ok {
		if reasoning, ok := reasoningTokens["reasoning_tokens"].(float64); ok {
			usage.ReasoningTokens = int64(reasoning)
		}
	}

	// Prompt Caching tokens
	if cachedTokens, ok := usageData["prompt_tokens_details"].(map[string]interface{}); ok {
		if cached, ok := cachedTokens["cached_tokens"].(float64); ok {
			usage.CachedTokens = int64(cached)
		}
	}

	return usage
}

// isReasoningModel 检查是否是推理模型
func (p *OpenAICompatibleProvider) isReasoningModel(model string) bool {
	reasoningModels := []string{"o1", "o3", "r1", "reasoning"}
	for _, rm := range reasoningModels {
		if strings.Contains(strings.ToLower(model), rm) {
			return true
		}
	}
	return false
}

// Config 返回配置
func (p *OpenAICompatibleProvider) Config() *types.ModelConfig {
	return p.config
}

// Capabilities 返回能力
func (p *OpenAICompatibleProvider) Capabilities() ProviderCapabilities {
	return p.capabilities
}

// SetSystemPrompt 设置系统提示词
func (p *OpenAICompatibleProvider) SetSystemPrompt(prompt string) error {
	p.systemPrompt = prompt
	return nil
}

// GetSystemPrompt 获取系统提示词
func (p *OpenAICompatibleProvider) GetSystemPrompt() string {
	return p.systemPrompt
}

// Close 关闭连接
func (p *OpenAICompatibleProvider) Close() error {
	return nil
}
