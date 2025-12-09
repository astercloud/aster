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

	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/types"
	"github.com/astercloud/aster/pkg/util"
)

var deepseekLog = logging.ForComponent("DeepseekProvider")

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
	deepseekLog.Info(ctx, "starting complete API call (non-streaming)", nil)
	deepseekLog.Info(ctx, "request params", map[string]any{"messages": len(messages), "tools": len(opts.Tools)})

	// 构建请求体(非流式)
	reqBody := dp.buildRequest(messages, opts)
	reqBody["stream"] = false // 关键:设置为非流式

	// 序列化（使用确定性序列化以优化 KV-Cache 命中率）
	jsonData, err := util.MarshalDeterministic(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	deepseekLog.Info(ctx, "request body size", map[string]any{"size_kb": float64(len(jsonData)) / 1024})

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
	deepseekLog.Info(ctx, "API endpoint", map[string]any{"url": fullURL})

	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+dp.apiKey)

	// 发送请求
	resp, err := dp.client.Do(req)
	if err != nil {
		deepseekLog.Error(ctx, "request failed", map[string]any{"error": err})
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	deepseekLog.Info(ctx, "received response", map[string]any{"status_code": resp.StatusCode})

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		deepseekLog.Error(ctx, "API error response", map[string]any{"body": string(body)})
		return nil, fmt.Errorf("deepseek api error: %d - %s", resp.StatusCode, string(body))
	}

	deepseekLog.Debug(ctx, "parsing API response", nil)

	// 解析完整响应
	var apiResp map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		deepseekLog.Error(ctx, "failed to parse response", map[string]any{"error": err})
		return nil, fmt.Errorf("decode response: %w", err)
	}

	deepseekLog.Debug(ctx, "response parsed successfully", nil)

	// 解析消息内容
	message, err := dp.parseCompleteResponse(apiResp)
	if err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	// 解析Token使用情况
	var usage *TokenUsage
	if usageData, ok := apiResp["usage"].(map[string]any); ok {
		usage = &TokenUsage{
			InputTokens:  int64(usageData["prompt_tokens"].(float64)),
			OutputTokens: int64(usageData["completion_tokens"].(float64)),
		}
		deepseekLog.Info(ctx, "token usage", map[string]any{"input": usage.InputTokens, "output": usage.OutputTokens, "total": usage.InputTokens + usage.OutputTokens})
	}

	deepseekLog.Info(ctx, "complete API call finished", nil)

	return &CompleteResponse{
		Message: message,
		Usage:   usage,
	}, nil
}

// Stream 流式对话
func (dp *DeepseekProvider) Stream(ctx context.Context, messages []types.Message, opts *StreamOptions) (<-chan StreamChunk, error) {
	// 构建请求体
	reqBody := dp.buildRequest(messages, opts)

	// 序列化（使用确定性序列化以优化 KV-Cache 命中率）
	jsonData, err := util.MarshalDeterministic(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// 记录请求内容（用于调试）
	if tools, ok := reqBody["tools"].([]map[string]any); ok && len(tools) > 0 {
		deepseekLog.Debug(ctx, "request body includes tools", map[string]any{"count": len(tools)})
		toolsJSON, _ := util.MarshalDeterministicIndent(reqBody["tools"], "", "  ")
		deepseekLog.Debug(ctx, "full tools definition", map[string]any{"tools": string(toolsJSON)})
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
		deepseekLog.Error(ctx, "API error response", map[string]any{"body": string(body)})
		return nil, fmt.Errorf("deepseek api error: %d - %s", resp.StatusCode, string(body))
	}

	deepseekLog.Debug(ctx, "API request successful", map[string]any{"status": resp.StatusCode})

	// 创建流式响应channel
	chunkCh := make(chan StreamChunk, 10)

	go dp.processStream(resp.Body, chunkCh)

	return chunkCh, nil
}

// buildRequest 构建请求体
func (dp *DeepseekProvider) buildRequest(messages []types.Message, opts *StreamOptions) map[string]any {
	ctx := context.Background() // 用于日志记录
	deepseekLog.Debug(ctx, "building request", map[string]any{"model": dp.config.Model})

	// 转换消息
	convertedMessages := dp.convertMessages(messages)

	// 处理 system prompt（OpenAI 兼容格式：作为第一条 system role 消息）
	if opts != nil && opts.System != "" {
		// 在消息数组开头插入 system 消息
		systemMessage := map[string]any{
			"role":    "system",
			"content": opts.System,
		}
		convertedMessages = append([]map[string]any{systemMessage}, convertedMessages...)
		deepseekLog.Debug(ctx, "added system message", map[string]any{"total_messages": len(convertedMessages), "system_prompt_length": len(opts.System)})
	} else if dp.systemPrompt != "" {
		// 使用 Provider 级别的 system prompt
		systemMessage := map[string]any{
			"role":    "system",
			"content": dp.systemPrompt,
		}
		convertedMessages = append([]map[string]any{systemMessage}, convertedMessages...)
		deepseekLog.Debug(ctx, "added provider system message", map[string]any{"total_messages": len(convertedMessages)})
	}

	req := map[string]any{
		"model":    dp.config.Model,
		"messages": convertedMessages,
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

		if len(opts.Tools) > 0 {
			// Deepseek API 使用 tools 字段，格式与 OpenAI 完全兼容
			tools := make([]map[string]any, 0, len(opts.Tools))
			for _, tool := range opts.Tools {
				toolMap := map[string]any{
					"type": "function",
					"function": map[string]any{
						"name":        tool.Name,
						"description": tool.Description,
						"parameters":  tool.InputSchema,
					},
					// TODO: Deepseek API 暂不支持 input_examples，待官方支持后启用
					// 参考: https://api-docs.deepseek.com/
					// 实现参考: pkg/provider/anthropic.go buildRequest() 中的 InputExamples 处理
				}
				tools = append(tools, toolMap)
			}
			req["tools"] = tools
			toolNames := make([]string, len(tools))
			for i, t := range tools {
				if fn, ok := t["function"].(map[string]any); ok {
					if name, ok := fn["name"].(string); ok {
						toolNames[i] = name
					}
				}
			}
			deepseekLog.Debug(ctx, "sending tools to API", map[string]any{"count": len(tools), "names": toolNames})
		}
	} else {
		req["max_tokens"] = 4096
	}

	return req
}

// convertMessages 转换消息格式（OpenAI 兼容格式）
func (dp *DeepseekProvider) convertMessages(messages []types.Message) []map[string]any {
	result := make([]map[string]any, 0, len(messages))

	for _, msg := range messages {
		// 跳过system消息（已在opts中单独传递）
		if msg.Role == types.MessageRoleSystem {
			continue
		}

		// Deepseek API 使用 OpenAI 兼容格式
		if msg.Role == types.MessageRoleAssistant {
			// Assistant 消息：检查是否有工具调用
			toolCalls := make([]map[string]any, 0)
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
						toolCall := map[string]any{
							"id":   b.ID,
							"type": "function",
							"function": map[string]any{
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

			msgMap := map[string]any{
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
			result = append(result, map[string]any{
				"role":    "user",
				"content": content,
			})
		}

		// 添加工具结果消息（每个工具结果作为独立的 tool 消息）
		for _, tr := range toolResults {
			toolMsg := map[string]any{
				"role":         "tool",
				"content":      tr.Content,
				"tool_call_id": tr.ToolUseID,
			}
			result = append(result, toolMsg)
			deepseekLog.Debug(context.Background(), "added tool result message", map[string]any{"tool_call_id": tr.ToolUseID, "content_length": len(tr.Content)})
		}
	}

	return result
}

// processStream 处理流式响应
func (dp *DeepseekProvider) processStream(body io.ReadCloser, chunkCh chan<- StreamChunk) {
	defer close(chunkCh)
	defer func() { _ = body.Close() }()

	ctx := context.Background() // 用于日志记录

	scanner := bufio.NewScanner(body)
	eventCount := 0
	for scanner.Scan() {
		line := scanner.Text()

		// SSE格式: "data: {...}"
		if !strings.HasPrefix(line, "data: ") {
			// 记录非数据行（用于调试）
			if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, ":") {
				deepseekLog.Debug(ctx, "non-data line", map[string]any{"line": line})
			}
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// 忽略特殊标记
		if data == "[DONE]" {
			deepseekLog.Debug(ctx, "received [DONE] marker", nil)
			break
		}

		// 解析JSON
		var event map[string]any
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			deepseekLog.Debug(ctx, "failed to parse JSON", map[string]any{"error": err, "data": data})
			continue
		}

		eventCount++
		deepseekLog.Debug(ctx, "stream event", map[string]any{"event_num": eventCount, "event": event})

		chunk := dp.parseStreamEvent(event)
		if chunk != nil {
			deepseekLog.Debug(ctx, "parsed chunk", map[string]any{"type": chunk.Type, "index": chunk.Index})
			chunkCh <- *chunk
		} else {
			deepseekLog.Debug(ctx, "no chunk parsed from event", nil)
		}
	}

	if err := scanner.Err(); err != nil {
		deepseekLog.Error(ctx, "scanner error", map[string]any{"error": err})
	}

	deepseekLog.Debug(ctx, "processed events", map[string]any{"total": eventCount})
}

// parseStreamEvent 解析流式事件（OpenAI 兼容格式）
func (dp *DeepseekProvider) parseStreamEvent(event map[string]any) *StreamChunk {
	ctx := context.Background() // 用于日志记录

	// Deepseek API 使用 OpenAI 兼容格式
	chunk := &StreamChunk{}

	// 检查 choices
	if choices, ok := event["choices"].([]any); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]any); ok {
			if delta, ok := choice["delta"].(map[string]any); ok {
				// 检查是否有 tool_calls（OpenAI 格式）
				if toolCalls, ok := delta["tool_calls"].([]any); ok && len(toolCalls) > 0 {
					// 工具调用开始
					if toolCall, ok := toolCalls[0].(map[string]any); ok {
						index := 0
						if idx, ok := toolCall["index"].(float64); ok {
							index = int(idx)
						}

						// 检查是否有 id 和 name（表示这是工具调用的开始）
						if id, hasID := toolCall["id"].(string); hasID {
							if fn, ok := toolCall["function"].(map[string]any); ok {
								if name, hasName := fn["name"].(string); hasName {
									// 这是工具调用的开始
									chunk.Type = "content_block_start"
									chunk.Index = index

									// 构建工具调用信息（转换为 Anthropic 格式以便统一处理）
									toolInfo := map[string]any{
										"type": "tool_use",
										"id":   id,
										"name": name,
									}

									chunk.Delta = toolInfo
									deepseekLog.Debug(ctx, "received tool_use block", map[string]any{"index": index, "id": id, "name": name})
									return chunk
								}
							}
						}

						// 如果没有 id 和 name，但存在 arguments，这是参数增量更新
						if fn, ok := toolCall["function"].(map[string]any); ok {
							if arguments, ok := fn["arguments"].(string); ok && arguments != "" {
								// 这是工具参数的增量更新
								chunk.Type = "content_block_delta"
								chunk.Index = index
								chunk.Delta = map[string]any{
									"type":      "arguments",
									"arguments": arguments,
								}
								deepseekLog.Debug(ctx, "received arguments delta", map[string]any{"index": index, "args": arguments})
								return chunk
							}
						}
					}
				}

				// 检查是否有 reasoning_content (DeepSeek Reasoner 模型的思考过程)
				if reasoningContent, ok := delta["reasoning_content"].(string); ok && reasoningContent != "" {
					chunk.Type = "reasoning_delta"
					chunk.Delta = map[string]any{
						"type":    "reasoning_delta",
						"content": reasoningContent,
					}
					deepseekLog.Debug(ctx, "received reasoning_content", map[string]any{"content": truncateString(reasoningContent, 50)})
					return chunk
				}

				// 检查是否有文本内容
				if content, ok := delta["content"].(string); ok && content != "" {
					chunk.Type = "content_block_delta"
					chunk.Delta = map[string]any{
						"type": "text_delta",
						"text": content,
					}
					return chunk
				}
			}

			// 检查 tool_calls 的增量更新（arguments 字段）
			if delta, ok := choice["delta"].(map[string]any); ok {
				if toolCalls, ok := delta["tool_calls"].([]any); ok && len(toolCalls) > 0 {
					if toolCall, ok := toolCalls[0].(map[string]any); ok {
						if fn, ok := toolCall["function"].(map[string]any); ok {
							if arguments, ok := fn["arguments"].(string); ok && arguments != "" {
								// 这是工具参数的增量更新
								chunk.Type = "content_block_delta"
								chunk.Index = 0
								if idx, ok := toolCall["index"].(float64); ok {
									chunk.Index = int(idx)
								}
								chunk.Delta = map[string]any{
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
func (dp *DeepseekProvider) parseCompleteResponse(apiResp map[string]any) (types.Message, error) {
	assistantContent := make([]types.ContentBlock, 0)

	// 获取第一个choice
	choices, ok := apiResp["choices"].([]any)
	if !ok || len(choices) == 0 {
		return types.Message{}, fmt.Errorf("no choices in response")
	}

	choice, ok := choices[0].(map[string]any)
	if !ok {
		return types.Message{}, fmt.Errorf("invalid choice format")
	}

	message, ok := choice["message"].(map[string]any)
	if !ok {
		return types.Message{}, fmt.Errorf("no message in choice")
	}

	// 解析文本内容
	if content, ok := message["content"].(string); ok && content != "" {
		assistantContent = append(assistantContent, &types.TextBlock{Text: content})
	}

	// 解析工具调用
	if toolCalls, ok := message["tool_calls"].([]any); ok && len(toolCalls) > 0 {
		for _, tc := range toolCalls {
			toolCall, ok := tc.(map[string]any)
			if !ok {
				continue
			}

			toolID, _ := toolCall["id"].(string)
			fn, ok := toolCall["function"].(map[string]any)
			if !ok {
				continue
			}

			toolName, _ := fn["name"].(string)
			argsJSON, _ := fn["arguments"].(string)

			// 解析参数
			var input map[string]any
			if err := json.Unmarshal([]byte(argsJSON), &input); err != nil {
				deepseekLog.Warn(context.Background(), "failed to parse tool arguments", map[string]any{"error": err})
				input = make(map[string]any)
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

// truncateString 截断字符串用于日志输出
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
