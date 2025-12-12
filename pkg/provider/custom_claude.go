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

	req, err := http.NewRequestWithContext(ctx, "POST", cp.baseURL+"/v1/messages", bytes.NewReader(jsonData))
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

	req, err := http.NewRequestWithContext(ctx, "POST", cp.baseURL+"/v1/messages", bytes.NewReader(jsonData))
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
			req["max_tokens"] = 4096
		}

		if opts.Temperature > 0 {
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
		req["max_tokens"] = 4096
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
					blocks = append(blocks, map[string]any{
						"type": "text",
						"text": b.Text,
					})
				case *types.ToolUseBlock:
					blocks = append(blocks, map[string]any{
						"type":  "tool_use",
						"id":    b.ID,
						"name":  b.Name,
						"input": b.Input,
					})
				case *types.ToolResultBlock:
					blocks = append(blocks, map[string]any{
						"type":        "tool_result",
						"tool_use_id": b.ToolUseID,
						"content":     b.Content,
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
			content = blocks
		} else {
			content = []any{
				map[string]any{
					"type": "text",
					"text": msg.Content,
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
	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var event map[string]any
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		chunk := cp.parseStreamEvent(event)
		if chunk != nil {
			chunkCh <- *chunk
		}
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
		}

	case "content_block_delta":
		// 安全获取 index
		if index, ok := event["index"].(float64); ok {
			chunk.Index = int(index)
		}
		if delta, ok := event["delta"].(map[string]any); ok {
			chunk.Delta = delta
		}

	case "content_block_stop":
		// 安全获取 index
		if index, ok := event["index"].(float64); ok {
			chunk.Index = int(index)
		}

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

// CustomClaudeFactory 自定义 Claude 工厂
type CustomClaudeFactory struct{}

// Create 创建自定义 Claude 提供商
func (f *CustomClaudeFactory) Create(config *types.ModelConfig) (Provider, error) {
	return NewCustomClaudeProvider(config)
}
