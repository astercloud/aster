package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/types"
)

const (
	defaultDeepseekBaseURL = "https://api.deepseek.com"
)

// DeepseekProvider Deepseek v3.2 模型提供商
// Deepseek API 与 OpenAI 完全兼容
type DeepseekProvider struct {
	config       *types.ModelConfig
	client       *http.Client
	baseURL      string
	apiKey       string
	systemPrompt string
}

// NewDeepseekProvider 创建 Deepseek 提供商
func NewDeepseekProvider(config *types.ModelConfig) (*DeepseekProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("deepseek api key is required")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = defaultDeepseekBaseURL
	}

	return &DeepseekProvider{
		config:  config,
		client:  &http.Client{},
		baseURL: baseURL,
		apiKey:  config.APIKey,
	}, nil
}

// Complete 非流式对话(阻塞式,返回完整响应)
func (dp *DeepseekProvider) Complete(ctx context.Context, messages []types.Message, opts *StreamOptions) (*CompleteResponse, error) {
	logging.Info(ctx, fmt.Sprintf("🚀 [DeepseekProvider] 开始Complete API调用 (非流式)"), nil)
	logging.Info(ctx, fmt.Sprintf("📊 [DeepseekProvider] 请求参数: %d条消息, %d个工具", len(messages), len(opts.Tools)), nil)

	// 构建请求体(非流式)
	reqBody := dp.buildRequest(messages, opts)
	reqBody["stream"] = false // 关键:设置为非流式

	// 序列化
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	logging.Info(ctx, fmt.Sprintf("📦 [DeepseekProvider] 请求体大小: %.2f KB", float64(len(jsonData))/1024), nil)

	// 创建HTTP请求
	endpoint := "/v1/chat/completions"
	if !strings.HasSuffix(dp.baseURL, "/v1") && !strings.HasSuffix(dp.baseURL, "/v1/") {
		if strings.HasSuffix(dp.baseURL, "/") {
			endpoint = "v1/chat/completions"
		} else {
			endpoint = "/v1/chat/completions"
		}
	}

	fullURL := dp.baseURL + endpoint
	logging.Info(ctx, fmt.Sprintf("🌐 [DeepseekProvider] API端点: %s", fullURL), nil)

	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+dp.apiKey)

	logging.Info(ctx, fmt.Sprintf("⏳ [DeepseekProvider] 发送请求到DeepSeek API，等待响应..."), nil)

	// 发送请求
	resp, err := dp.client.Do(req)
	if err != nil {
		logging.Error(ctx, fmt.Sprintf("❌ [DeepseekProvider] 请求失败: %v", err), nil)
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	logging.Info(ctx, fmt.Sprintf("✅ [DeepseekProvider] 收到响应, HTTP状态码: %d", resp.StatusCode), nil)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logging.Error(ctx, fmt.Sprintf("❌ [DeepseekProvider] API错误响应: %s", string(body)), nil)
		return nil, fmt.Errorf("deepseek api error: %d - %s", resp.StatusCode, string(body))
	}

	logging.Debug(ctx, fmt.Sprintf("📖 [DeepseekProvider] 正在解析API响应..."), nil)

	// 解析完整响应
	var apiResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		logging.Error(ctx, fmt.Sprintf("❌ [DeepseekProvider] 解析响应失败: %v", err), nil)
		return nil, fmt.Errorf("decode response: %w", err)
	}

	logging.Debug(ctx, fmt.Sprintf("✅ [DeepseekProvider] 响应解析成功"), nil)

	// 解析消息内容
	message, err := dp.parseCompleteResponse(apiResp)
	if err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	// 解析Token使用情况
	var usage *TokenUsage
	if usageData, ok := apiResp["usage"].(map[string]interface{}); ok {
		usage = &TokenUsage{
			InputTokens:  int64(usageData["prompt_tokens"].(float64)),
			OutputTokens: int64(usageData["completion_tokens"].(float64)),
		}
		logging.Info(ctx, fmt.Sprintf("💰 [DeepseekProvider] Token使用: 输入=%d, 输出=%d, 总计=%d", usage.InputTokens, usage.OutputTokens, usage.InputTokens+usage.OutputTokens), nil)
	}

	logging.Info(ctx, fmt.Sprintf("🎉 [DeepseekProvider] Complete API调用完成"), nil)

	return &CompleteResponse{
		Message: message,
		Usage:   usage,
	}, nil
}

// Stream 流式对话
func (dp *DeepseekProvider) Stream(ctx context.Context, messages []types.Message, opts *StreamOptions) (<-chan StreamChunk, error) {
	// 构建请求体
	reqBody := dp.buildRequest(messages, opts)

	// 序列化
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// 记录请求内容（用于调试）
	if tools, ok := reqBody["tools"].([]map[string]interface{}); ok && len(tools) > 0 {
		log.Printf("[DeepseekProvider] Request body includes %d tools", len(tools))
		toolsJSON, _ := json.MarshalIndent(reqBody["tools"], "", "  ")
		log.Printf("[DeepseekProvider] Full tools definition:\n%s", string(toolsJSON))
	}

	// 创建HTTP请求
	// Deepseek API 使用 OpenAI 兼容格式：/v1/chat/completions
	endpoint := "/v1/chat/completions"
	if !strings.HasSuffix(dp.baseURL, "/v1") && !strings.HasSuffix(dp.baseURL, "/v1/") {
		// 如果 baseURL 不包含 /v1，使用完整路径
		if strings.HasSuffix(dp.baseURL, "/") {
			endpoint = "v1/chat/completions"
		} else {
			endpoint = "/v1/chat/completions"
		}
	}
	req, err := http.NewRequestWithContext(ctx, "POST", dp.baseURL+endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+dp.apiKey)

	// 发送请求
	resp, err := dp.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		log.Printf("[DeepseekProvider] API error response: %s", string(body))
		return nil, fmt.Errorf("deepseek api error: %d - %s", resp.StatusCode, string(body))
	}

	log.Printf("[DeepseekProvider] API request successful, status: %d", resp.StatusCode)

	// 创建流式响应channel
	chunkCh := make(chan StreamChunk, 10)

	go dp.processStream(resp.Body, chunkCh)

	return chunkCh, nil
}

// buildRequest 构建请求体
func (dp *DeepseekProvider) buildRequest(messages []types.Message, opts *StreamOptions) map[string]interface{} {
	req := map[string]interface{}{
		"model":    dp.config.Model,
		"messages": dp.convertMessages(messages),
		"stream":   true,
	}

	if opts != nil {
		if opts.MaxTokens > 0 {
			req["max_tokens"] = opts.MaxTokens
		} else {
			req["max_tokens"] = 4096
		}

		if opts.Temperature > 0 {
			req["temperature"] = opts.Temperature
		}

		if opts.System != "" {
			// Deepseek API 支持 system 字段（OpenAI 兼容）
			req["system"] = opts.System
			log.Printf("[DeepseekProvider] System prompt length: %d", len(opts.System))
		} else if dp.systemPrompt != "" {
			req["system"] = dp.systemPrompt
		}

		if len(opts.Tools) > 0 {
			// Deepseek API 使用 tools 字段，格式与 OpenAI 完全兼容
			tools := make([]map[string]interface{}, 0, len(opts.Tools))
			for _, tool := range opts.Tools {
				toolMap := map[string]interface{}{
					"type": "function",
					"function": map[string]interface{}{
						"name":        tool.Name,
						"description": tool.Description,
						"parameters":  tool.InputSchema,
					},
				}
				tools = append(tools, toolMap)
			}
			req["tools"] = tools
			toolNames := make([]string, len(tools))
			for i, t := range tools {
				if fn, ok := t["function"].(map[string]interface{}); ok {
					if name, ok := fn["name"].(string); ok {
						toolNames[i] = name
					}
				}
			}
			log.Printf("[DeepseekProvider] Sending %d tools to API: %v", len(tools), toolNames)
		}
	} else {
		req["max_tokens"] = 4096
		if dp.systemPrompt != "" {
			req["system"] = dp.systemPrompt
		}
	}

	return req
}

// convertMessages 转换消息格式（OpenAI 兼容格式）
func (dp *DeepseekProvider) convertMessages(messages []types.Message) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(messages))

	for _, msg := range messages {
		// 跳过system消息（已在opts中单独传递）
		if msg.Role == types.MessageRoleSystem {
			continue
		}

		// Deepseek API 使用 OpenAI 兼容格式
		if msg.Role == types.MessageRoleAssistant {
			// Assistant 消息：检查是否有工具调用
			toolCalls := make([]map[string]interface{}, 0)
			textContent := ""

			// 处理 ContentBlocks（如果存在）
			if len(msg.ContentBlocks) > 0 {
				for _, block := range msg.ContentBlocks {
					switch b := block.(type) {
					case *types.TextBlock:
						textContent += b.Text
					case *types.ToolUseBlock:
						// 转换为 OpenAI 格式的 tool_calls
						argsJSON, _ := json.Marshal(b.Input)
						toolCall := map[string]interface{}{
							"id":   b.ID,
							"type": "function",
							"function": map[string]interface{}{
								"name":      b.Name,
								"arguments": string(argsJSON),
							},
						}
						toolCalls = append(toolCalls, toolCall)
					}
				}
			} else {
				// 向后兼容：使用简单的 Content string
				textContent = msg.Content
			}

			msgMap := map[string]interface{}{
				"role": "assistant",
			}

			if textContent != "" {
				msgMap["content"] = textContent
			} else if len(toolCalls) == 0 {
				// 如果没有内容和工具调用，设置空内容
				msgMap["content"] = ""
			}

			if len(toolCalls) > 0 {
				msgMap["tool_calls"] = toolCalls
			}

			result = append(result, msgMap)
			continue
		}

		// User 消息：检查是否包含工具结果
		// 在 OpenAI 格式中，工具结果必须作为独立的 role: "tool" 消息发送
		toolResults := make([]*types.ToolResultBlock, 0)
		textParts := make([]string, 0)

		// 处理 ContentBlocks（如果存在）
		if len(msg.ContentBlocks) > 0 {
			for _, block := range msg.ContentBlocks {
				switch b := block.(type) {
				case *types.TextBlock:
					textParts = append(textParts, b.Text)
				case *types.ToolResultBlock:
					// 收集工具结果，稍后单独处理
					toolResults = append(toolResults, b)
				}
			}
		} else {
			// 向后兼容：使用简单的 Content string
			if msg.Content != "" {
				textParts = append(textParts, msg.Content)
			}
		}

		// 如果有文本内容，先添加文本消息
		content := strings.Join(textParts, "\n")
		if content != "" {
			result = append(result, map[string]interface{}{
				"role":    "user",
				"content": content,
			})
		}

		// 添加工具结果消息（每个工具结果作为独立的 tool 消息）
		for _, tr := range toolResults {
			toolMsg := map[string]interface{}{
				"role":         "tool",
				"content":      tr.Content,
				"tool_call_id": tr.ToolUseID,
			}
			result = append(result, toolMsg)
			log.Printf("[DeepseekProvider] Added tool result message: tool_call_id=%s, content_length=%d", tr.ToolUseID, len(tr.Content))
		}
	}

	return result
}

// processStream 处理流式响应
func (dp *DeepseekProvider) processStream(body io.ReadCloser, chunkCh chan<- StreamChunk) {
	defer close(chunkCh)
	defer func() { _ = body.Close() }()

	scanner := bufio.NewScanner(body)
	eventCount := 0
	for scanner.Scan() {
		line := scanner.Text()

		// SSE格式: "data: {...}"
		if !strings.HasPrefix(line, "data: ") {
			// 记录非数据行（用于调试）
			if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, ":") {
				log.Printf("[DeepseekProvider] Non-data line: %s", line)
			}
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// 忽略特殊标记
		if data == "[DONE]" {
			log.Printf("[DeepseekProvider] Received [DONE] marker")
			break
		}

		// 解析JSON
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			log.Printf("[DeepseekProvider] Failed to parse JSON: %v, data: %s", err, data)
			continue
		}

		eventCount++
		log.Printf("[DeepseekProvider] Event #%d: %v", eventCount, event)

		chunk := dp.parseStreamEvent(event)
		if chunk != nil {
			log.Printf("[DeepseekProvider] Parsed chunk: type=%s, index=%d", chunk.Type, chunk.Index)
			chunkCh <- *chunk
		} else {
			log.Printf("[DeepseekProvider] No chunk parsed from event")
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("[DeepseekProvider] Scanner error: %v", err)
	}

	log.Printf("[DeepseekProvider] Processed %d events total", eventCount)
}

// parseStreamEvent 解析流式事件（OpenAI 兼容格式）
func (dp *DeepseekProvider) parseStreamEvent(event map[string]interface{}) *StreamChunk {
	// Deepseek API 使用 OpenAI 兼容格式
	chunk := &StreamChunk{}

	// 检查 choices
	if choices, ok := event["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if delta, ok := choice["delta"].(map[string]interface{}); ok {
				// 检查是否有 tool_calls（OpenAI 格式）
				if toolCalls, ok := delta["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
					// 工具调用开始
					if toolCall, ok := toolCalls[0].(map[string]interface{}); ok {
						index := 0
						if idx, ok := toolCall["index"].(float64); ok {
							index = int(idx)
						}

						// 检查是否有 id 和 name（表示这是工具调用的开始）
						if id, hasID := toolCall["id"].(string); hasID {
							if fn, ok := toolCall["function"].(map[string]interface{}); ok {
								if name, hasName := fn["name"].(string); hasName {
									// 这是工具调用的开始
									chunk.Type = "content_block_start"
									chunk.Index = index

									// 构建工具调用信息（转换为 Anthropic 格式以便统一处理）
									toolInfo := map[string]interface{}{
										"type": "tool_use",
										"id":   id,
										"name": name,
									}

									chunk.Delta = toolInfo
									log.Printf("[DeepseekProvider] ✅ Received tool_use block: index=%d, id=%v, name=%v", index, id, name)
									return chunk
								}
							}
						}

						// 如果没有 id 和 name，但存在 arguments，这是参数增量更新
						if fn, ok := toolCall["function"].(map[string]interface{}); ok {
							if arguments, ok := fn["arguments"].(string); ok && arguments != "" {
								// 这是工具参数的增量更新
								chunk.Type = "content_block_delta"
								chunk.Index = index
								chunk.Delta = map[string]interface{}{
									"type":      "arguments",
									"arguments": arguments,
								}
								log.Printf("[DeepseekProvider] Received arguments delta: index=%d, args=%s", index, arguments)
								return chunk
							}
						}
					}
				}

				// 检查是否有文本内容
				if content, ok := delta["content"].(string); ok && content != "" {
					chunk.Type = "content_block_delta"
					chunk.Delta = map[string]interface{}{
						"type": "text_delta",
						"text": content,
					}
					return chunk
				}
			}

			// 检查 tool_calls 的增量更新（arguments 字段）
			if delta, ok := choice["delta"].(map[string]interface{}); ok {
				if toolCalls, ok := delta["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
					if toolCall, ok := toolCalls[0].(map[string]interface{}); ok {
						if fn, ok := toolCall["function"].(map[string]interface{}); ok {
							if arguments, ok := fn["arguments"].(string); ok && arguments != "" {
								// 这是工具参数的增量更新
								chunk.Type = "content_block_delta"
								chunk.Index = 0
								if idx, ok := toolCall["index"].(float64); ok {
									chunk.Index = int(idx)
								}
								chunk.Delta = map[string]interface{}{
									"type":      "arguments",
									"arguments": arguments,
								}
								return chunk
							}
						}
					}
				}
			}

			// 检查 finish_reason
			if finishReason, ok := choice["finish_reason"].(string); ok {
				if finishReason == "tool_calls" {
					chunk.Type = "message_delta"
					return chunk
				}
			}
		}
	}

	return nil
}

// Config 返回配置
func (dp *DeepseekProvider) Config() *types.ModelConfig {
	return dp.config
}

// Capabilities 返回模型能力
func (dp *DeepseekProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		SupportToolCalling:  true,
		SupportSystemPrompt: true,
		SupportStreaming:    true,
		SupportVision:       false,
		MaxTokens:           8192,
		MaxToolsPerCall:     0,
		ToolCallingFormat:   "openai", // Deepseek 使用 OpenAI 兼容格式
	}
}

// SetSystemPrompt 设置系统提示词
func (dp *DeepseekProvider) SetSystemPrompt(prompt string) error {
	dp.systemPrompt = prompt
	return nil
}

// GetSystemPrompt 获取系统提示词
func (dp *DeepseekProvider) GetSystemPrompt() string {
	return dp.systemPrompt
}

// parseCompleteResponse 解析完整的非流式响应
func (dp *DeepseekProvider) parseCompleteResponse(apiResp map[string]interface{}) (types.Message, error) {
	assistantContent := make([]types.ContentBlock, 0)

	// 获取第一个choice
	choices, ok := apiResp["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return types.Message{}, fmt.Errorf("no choices in response")
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return types.Message{}, fmt.Errorf("invalid choice format")
	}

	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		return types.Message{}, fmt.Errorf("no message in choice")
	}

	// 解析文本内容
	if content, ok := message["content"].(string); ok && content != "" {
		assistantContent = append(assistantContent, &types.TextBlock{Text: content})
	}

	// 解析工具调用
	if toolCalls, ok := message["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
		for _, tc := range toolCalls {
			toolCall, ok := tc.(map[string]interface{})
			if !ok {
				continue
			}

			toolID, _ := toolCall["id"].(string)
			fn, ok := toolCall["function"].(map[string]interface{})
			if !ok {
				continue
			}

			toolName, _ := fn["name"].(string)
			argsJSON, _ := fn["arguments"].(string)

			// 解析参数
			var input map[string]interface{}
			if err := json.Unmarshal([]byte(argsJSON), &input); err != nil {
				log.Printf("[DeepseekProvider] Failed to parse tool arguments: %v", err)
				input = make(map[string]interface{})
			}

			assistantContent = append(assistantContent, &types.ToolUseBlock{
				ID:    toolID,
				Name:  toolName,
				Input: input,
			})
		}
	}

	return types.Message{
		Role:          types.MessageRoleAssistant,
		ContentBlocks: assistantContent,
	}, nil
}

// Close 关闭连接
func (dp *DeepseekProvider) Close() error {
	return nil
}
