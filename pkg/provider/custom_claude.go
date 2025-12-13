package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/types"
	"github.com/astercloud/aster/pkg/util"
)

var customClaudeLog = logging.ForComponent("CustomClaudeProvider")

// CustomClaudeProvider 自定义 Claude API 中转站提供商
// 适配各种中转站的特殊响应格式
type CustomClaudeProvider struct {
	config       *types.ModelConfig
	client       *http.Client
	baseURL      string
	version      string
	systemPrompt string
}

// NewCustomClaudeProvider 创建自定义 Claude 提供商
func NewCustomClaudeProvider(config *types.ModelConfig) (*CustomClaudeProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("api key is required")
	}

	if config.BaseURL == "" {
		return nil, fmt.Errorf("base url is required for custom claude provider")
	}

	// 配置 HTTP 客户端超时，避免无限等待
	client := &http.Client{
		Timeout: 120 * time.Second, // 全局超时 120 秒
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second, // 连接超时 30 秒
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second, // TLS 握手超时
			ResponseHeaderTimeout: 30 * time.Second, // 响应头超时
			ExpectContinueTimeout: 1 * time.Second,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
		},
	}

	return &CustomClaudeProvider{
		config:  config,
		client:  client,
		baseURL: config.BaseURL,
		version: "2023-06-01",
	}, nil
}

// Complete 非流式对话
func (cp *CustomClaudeProvider) Complete(ctx context.Context, messages []types.Message, opts *StreamOptions) (*CompleteResponse, error) {
	reqBody := cp.buildRequest(messages, opts)
	reqBody["stream"] = false

	jsonData, err := util.MarshalDeterministic(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", cp.getEndpoint(), bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", cp.config.APIKey)
	req.Header.Set("anthropic-version", cp.version)

	resp, err := cp.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		customClaudeLog.Error(ctx, "API error response", map[string]any{"status": resp.StatusCode, "body": string(body)})
		return nil, fmt.Errorf("api error: %d - %s", resp.StatusCode, string(body))
	}

	var apiResp map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	message, err := cp.parseCompleteResponse(apiResp)
	if err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	var usage *TokenUsage
	if usageData, ok := apiResp["usage"].(map[string]any); ok {
		usage = cp.parseUsage(usageData)
	}

	return &CompleteResponse{
		Message: message,
		Usage:   usage,
	}, nil
}

// Stream 流式对话
func (cp *CustomClaudeProvider) Stream(ctx context.Context, messages []types.Message, opts *StreamOptions) (<-chan StreamChunk, error) {
	reqBody := cp.buildRequest(messages, opts)

	jsonData, err := util.MarshalDeterministic(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// 调试：打印请求体大小和前 2000 字符
	customClaudeLog.Debug(ctx, "request body prepared", map[string]any{
		"size":    len(jsonData),
		"preview": string(jsonData[:min(len(jsonData), 2000)]),
	})

	req, err := http.NewRequestWithContext(ctx, "POST", cp.getEndpoint(), bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", cp.config.APIKey)
	req.Header.Set("anthropic-version", cp.version)

	resp, err := cp.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		// 调试：当出错时打印完整请求体
		// 对于 400 错误，打印更多内容以便诊断 invalid JSON body 问题
		previewLen := 5000
		if resp.StatusCode == 400 {
			previewLen = 20000 // 400 错误时打印更多内容
		}
		customClaudeLog.Error(ctx, "API request failed", map[string]any{
			"status":       resp.StatusCode,
			"error":        string(body),
			"request_size": len(jsonData),
			"request_body": string(jsonData[:min(len(jsonData), previewLen)]),
		})
		// 如果是 400 错误且包含 invalid JSON，额外打印请求体的最后部分（可能是截断位置）
		if resp.StatusCode == 400 && len(jsonData) > previewLen {
			customClaudeLog.Error(ctx, "API request body tail (for invalid JSON diagnosis)", map[string]any{
				"tail": string(jsonData[max(0, len(jsonData)-5000):]),
			})
		}
		return nil, fmt.Errorf("api error: %d - %s", resp.StatusCode, string(body))
	}

	chunkCh := make(chan StreamChunk, 10)
	go cp.processStream(resp.Body, chunkCh)

	return chunkCh, nil
}

// buildRequest 构建请求体
func (cp *CustomClaudeProvider) buildRequest(messages []types.Message, opts *StreamOptions) map[string]any {
	req := map[string]any{
		"model":    cp.config.Model,
		"messages": cp.convertMessages(messages),
		"stream":   true,
	}

	if opts != nil {
		if opts.MaxTokens > 0 {
			req["max_tokens"] = opts.MaxTokens
		} else {
			// Claude 4 Sonnet/Opus 最大 output tokens 为 64000
			// 设置为 32000 作为默认值，足够大多数工具调用场景
			req["max_tokens"] = 32000
		}

		// Extended Thinking 配置
		// 注意：启用 thinking 时，temperature 必须为 1（或不设置）
		if opts.Thinking != nil && opts.Thinking.Enabled {
			budgetTokens := opts.Thinking.BudgetTokens
			if budgetTokens <= 0 {
				budgetTokens = 10000 // 默认 10000 tokens 的思考预算
			}
			req["thinking"] = map[string]any{
				"type":          "enabled",
				"budget_tokens": budgetTokens,
			}
			// 启用 thinking 时不能设置 temperature（必须为默认值 1）
			customClaudeLog.Info(context.Background(), "extended thinking enabled", map[string]any{
				"budget_tokens": budgetTokens,
			})
		} else if opts.Temperature > 0 {
			req["temperature"] = opts.Temperature
		}

		if opts.System != "" {
			req["system"] = opts.System
		} else if cp.systemPrompt != "" {
			req["system"] = cp.systemPrompt
		}

		if len(opts.Tools) > 0 {
			tools := make([]map[string]any, 0, len(opts.Tools))
			for _, tool := range opts.Tools {
				toolMap := map[string]any{
					"name":         tool.Name,
					"description":  tool.Description,
					"input_schema": tool.InputSchema,
				}
				tools = append(tools, toolMap)
			}
			req["tools"] = tools
		}
	} else {
		req["max_tokens"] = 32000
		if cp.systemPrompt != "" {
			req["system"] = cp.systemPrompt
		}
	}

	return req
}

// convertMessages 转换消息格式
func (cp *CustomClaudeProvider) convertMessages(messages []types.Message) []map[string]any {
	result := make([]map[string]any, 0, len(messages))

	for _, msg := range messages {
		if msg.Role == types.MessageRoleSystem {
			continue
		}

		var content any
		if len(msg.ContentBlocks) > 0 {
			blocks := make([]any, 0, len(msg.ContentBlocks))
			for _, block := range msg.ContentBlocks {
				switch b := block.(type) {
				case *types.TextBlock:
					// 跳过空文本块
					if b.Text == "" {
						continue
					}
					blocks = append(blocks, map[string]any{
						"type": "text",
						"text": b.Text,
					})
				case *types.ToolUseBlock:
					// 确保 input 是有效的字典，避免 API 报错
					input := b.Input
					if input == nil {
						input = make(map[string]any)
					}
					// 检查是否有解析错误标记，如果有则使用空对象
					// 这处理了之前因 max_tokens 截断导致 JSON 不完整的情况
					if _, hasParseError := input["__parse_error__"]; hasParseError {
						input = map[string]any{
							"error": "参数解析失败，请重试",
						}
					}
					blocks = append(blocks, map[string]any{
						"type":  "tool_use",
						"id":    b.ID,
						"name":  b.Name,
						"input": input,
					})
				case *types.ToolResultBlock:
					// 确保 content 不为空
					contentVal := b.Content
					if contentVal == "" {
						contentVal = " " // 使用空格而不是空字符串
					}
					blocks = append(blocks, map[string]any{
						"type":        "tool_result",
						"tool_use_id": b.ToolUseID,
						"content":     contentVal,
						"is_error":    b.IsError,
					})
				case *types.ImageContent:
					// 转换为 Anthropic API 格式
					imageBlock := map[string]any{
						"type": "image",
					}
					switch b.Type {
					case "base64":
						imageBlock["source"] = map[string]any{
							"type":       "base64",
							"media_type": b.MimeType,
							"data":       b.Source,
						}
					case "url":
						imageBlock["source"] = map[string]any{
							"type": "url",
							"url":  b.Source,
						}
					}
					blocks = append(blocks, imageBlock)
				}
			}
			// 如果所有块都被跳过，添加一个空文本块
			if len(blocks) == 0 {
				blocks = append(blocks, map[string]any{
					"type": "text",
					"text": " ", // 使用空格而不是空字符串
				})
			}
			content = blocks
		} else {
			// 确保 Content 不为空
			text := msg.Content
			if text == "" {
				text = " " // 使用空格而不是空字符串
			}
			content = []any{
				map[string]any{
					"type": "text",
					"text": text,
				},
			}
		}

		result = append(result, map[string]any{
			"role":    string(msg.Role),
			"content": content,
		})
	}

	return result
}

// processStream 处理流式响应（兼容不同格式）
func (cp *CustomClaudeProvider) processStream(body io.ReadCloser, chunkCh chan<- StreamChunk) {
	defer close(chunkCh)
	defer func() { _ = body.Close() }()

	scanner := bufio.NewScanner(body)
	// 增加 scanner 缓冲区大小，避免长行被截断
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024) // 最大 1MB

	lineCount := 0
	// 用于累积工具输入 JSON 的调试信息
	toolInputBuffers := make(map[int]string)

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		// 打印所有原始行（调试用）
		if len(line) > 0 && line != "" {
			// 截断过长的行用于日志显示
			logLine := line
			if len(logLine) > 500 {
				logLine = logLine[:500] + "...[truncated]"
			}
			customClaudeLog.Debug(context.Background(), "RAW SSE LINE", map[string]any{
				"line_num": lineCount,
				"line":     logLine,
				"line_len": len(line),
			})
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			customClaudeLog.Info(context.Background(), "stream done", map[string]any{
				"total_lines":       lineCount,
				"tool_input_buffers": toolInputBuffers,
			})
			break
		}

		var event map[string]any
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			customClaudeLog.Warn(context.Background(), "failed to parse SSE event", map[string]any{
				"error":    err.Error(),
				"data_len": len(data),
				"data":     data,
			})
			continue
		}

		// 调试：记录所有事件类型和完整内容
		eventType, _ := event["type"].(string)
		
		// 记录所有事件的完整内容（用于调试中转站格式）
		// 使用 Debug 级别避免生产环境日志过于冗长
		customClaudeLog.Debug(context.Background(), "SSE EVENT", map[string]any{
			"line_num":   lineCount,
			"event_type": eventType,
			"event":      event,
		})

		// 特别追踪工具输入的累积
		if eventType == "content_block_delta" {
			if delta, ok := event["delta"].(map[string]any); ok {
				deltaType, _ := delta["type"].(string)
				if deltaType == "input_json_delta" {
					index := 0
					if idx, ok := event["index"].(float64); ok {
						index = int(idx)
					}
					partialJSON, _ := delta["partial_json"].(string)
					toolInputBuffers[index] += partialJSON
					// 使用 Debug 级别，避免每个 delta 都输出大量日志
					customClaudeLog.Debug(context.Background(), "TOOL INPUT ACCUMULATING", map[string]any{
						"index":           index,
						"accumulated_len": len(toolInputBuffers[index]),
					})
				}
			}
		}

		// 记录 content_block_stop 事件，检查工具输入是否完整
		if eventType == "content_block_stop" {
			index := 0
			if idx, ok := event["index"].(float64); ok {
				index = int(idx)
			}
			if accumulated, exists := toolInputBuffers[index]; exists {
				// 只在工具输入完成时记录一次，使用 Debug 级别
				customClaudeLog.Debug(context.Background(), "TOOL INPUT FINAL", map[string]any{
					"index":    index,
					"json_len": len(accumulated),
					"is_valid": json.Valid([]byte(accumulated)),
				})
			}
		}

		chunk := cp.parseStreamEvent(event)
		if chunk != nil {
			chunkCh <- *chunk
		}
	}

	if err := scanner.Err(); err != nil {
		customClaudeLog.Error(context.Background(), "scanner error", map[string]any{
			"error": err.Error(),
		})
	} else {
		customClaudeLog.Info(context.Background(), "scanner finished normally", map[string]any{
			"total_lines": lineCount,
		})
	}
}

// parseStreamEvent 解析流式事件（兼容处理）
func (cp *CustomClaudeProvider) parseStreamEvent(event map[string]any) *StreamChunk {
	eventType, _ := event["type"].(string)

	chunk := &StreamChunk{
		Type: eventType,
	}

	switch eventType {
	case "content_block_start":
		// 安全获取 index
		if index, ok := event["index"].(float64); ok {
			chunk.Index = int(index)
		}
		if contentBlock, ok := event["content_block"].(map[string]any); ok {
			chunk.Delta = contentBlock
			blockType, _ := contentBlock["type"].(string)

			// 处理 thinking 块（Extended Thinking）
			if blockType == "thinking" {
				customClaudeLog.Info(context.Background(), "thinking block start", map[string]any{
					"index": chunk.Index,
				})
			}

			// 调试日志：记录工具调用开始
			if blockType == "tool_use" {
				// 检查是否在 content_block_start 中就包含了完整的 input
				// 某些中转站可能不发送 input_json_delta，而是直接在这里提供完整的 input
				if input, ok := contentBlock["input"].(map[string]any); ok && len(input) > 0 {
					customClaudeLog.Info(context.Background(), "tool_use block start with input", map[string]any{
						"index":       chunk.Index,
						"tool_id":     contentBlock["id"],
						"tool_name":   contentBlock["name"],
						"input_keys":  len(input),
						"has_input":   true,
					})
				} else {
					customClaudeLog.Debug(context.Background(), "tool_use block start without input", map[string]any{
						"index":         chunk.Index,
						"content_block": contentBlock,
					})
				}
			}
		}

	case "content_block_delta":
		// 安全获取 index
		if index, ok := event["index"].(float64); ok {
			chunk.Index = int(index)
		}
		if delta, ok := event["delta"].(map[string]any); ok {
			chunk.Delta = delta
			// 调试日志：记录所有 delta 类型
			deltaType, _ := delta["type"].(string)
			customClaudeLog.Debug(context.Background(), "content_block_delta received", map[string]any{
				"index":      chunk.Index,
				"delta_type": deltaType,
			})

			// 处理 thinking_delta（Extended Thinking 增量）
			if deltaType == "thinking_delta" {
				thinking, _ := delta["thinking"].(string)
				customClaudeLog.Debug(context.Background(), "thinking_delta received", map[string]any{
					"index":        chunk.Index,
					"thinking_len": len(thinking),
				})
			}

			// input_json_delta 已在 processStreamResponse 中追踪，这里不再重复记录
		} else {
			customClaudeLog.Warn(context.Background(), "content_block_delta missing delta field", map[string]any{
				"event": event,
			})
		}

	case "content_block_stop":
		// 安全获取 index
		if index, ok := event["index"].(float64); ok {
			chunk.Index = int(index)
		}
		customClaudeLog.Debug(context.Background(), "content_block_stop", map[string]any{
			"index": chunk.Index,
		})

	case "message_delta":
		if delta, ok := event["delta"].(map[string]any); ok {
			chunk.Delta = delta
		}
		// 安全解析 usage
		if usage, ok := event["usage"].(map[string]any); ok {
			chunk.Usage = cp.parseUsage(usage)
		}
	}

	return chunk
}

// parseUsage 安全解析 token 使用情况
func (cp *CustomClaudeProvider) parseUsage(usage map[string]any) *TokenUsage {
	result := &TokenUsage{}

	// 安全获取 input_tokens
	if inputTokens, ok := usage["input_tokens"].(float64); ok {
		result.InputTokens = int64(inputTokens)
	} else if inputTokens, ok := usage["input_tokens"].(int64); ok {
		result.InputTokens = inputTokens
	} else if inputTokens, ok := usage["input_tokens"].(int); ok {
		result.InputTokens = int64(inputTokens)
	}

	// 安全获取 output_tokens
	if outputTokens, ok := usage["output_tokens"].(float64); ok {
		result.OutputTokens = int64(outputTokens)
	} else if outputTokens, ok := usage["output_tokens"].(int64); ok {
		result.OutputTokens = outputTokens
	} else if outputTokens, ok := usage["output_tokens"].(int); ok {
		result.OutputTokens = int64(outputTokens)
	}

	return result
}

// parseCompleteResponse 解析完整响应
func (cp *CustomClaudeProvider) parseCompleteResponse(apiResp map[string]any) (types.Message, error) {
	assistantContent := make([]types.ContentBlock, 0)

	content, ok := apiResp["content"].([]any)
	if !ok || len(content) == 0 {
		return types.Message{}, fmt.Errorf("no content in response")
	}

	for _, item := range content {
		block, ok := item.(map[string]any)
		if !ok {
			continue
		}

		blockType, _ := block["type"].(string)

		switch blockType {
		case "text":
			if text, ok := block["text"].(string); ok {
				assistantContent = append(assistantContent, &types.TextBlock{Text: text})
			}

		case "tool_use":
			toolID, _ := block["id"].(string)
			toolName, _ := block["name"].(string)

			var input map[string]any
			if inputData, ok := block["input"].(map[string]any); ok {
				input = inputData
			} else {
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

// Config 返回配置
func (cp *CustomClaudeProvider) Config() *types.ModelConfig {
	return cp.config
}

// Capabilities 返回模型能力
func (cp *CustomClaudeProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		SupportToolCalling:  true,
		SupportSystemPrompt: true,
		SupportStreaming:    true,
		SupportVision:       true,
		MaxTokens:           200000,
		MaxToolsPerCall:     0,
		ToolCallingFormat:   "anthropic",
	}
}

// SetSystemPrompt 设置系统提示词
func (cp *CustomClaudeProvider) SetSystemPrompt(prompt string) error {
	cp.systemPrompt = prompt
	return nil
}

// GetSystemPrompt 获取系统提示词
func (cp *CustomClaudeProvider) GetSystemPrompt() string {
	return cp.systemPrompt
}

// Close 关闭连接
func (cp *CustomClaudeProvider) Close() error {
	return nil
}

// getEndpoint 返回 API 端点地址
// baseURL + /v1/messages（Anthropic API 格式）
func (cp *CustomClaudeProvider) getEndpoint() string {
	return cp.baseURL + "/v1/messages"
}

// CustomClaudeFactory 自定义 Claude 工厂
type CustomClaudeFactory struct{}

// Create 创建自定义 Claude 提供商
func (f *CustomClaudeFactory) Create(config *types.ModelConfig) (Provider, error) {
	return NewCustomClaudeProvider(config)
}
