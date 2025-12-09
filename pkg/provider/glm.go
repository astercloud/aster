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

var glmLog = logging.ForComponent("GLMProvider")

const (
	defaultGLMBaseURL = "https://open.bigmodel.cn/api/paas/v4"
)

// GLMProvider GLM 4.6 模型提供商
type GLMProvider struct {
	config       *types.ModelConfig
	client       *http.Client
	baseURL      string
	apiKey       string
	systemPrompt string
}

// NewGLMProvider 创建 GLM 提供商
func NewGLMProvider(config *types.ModelConfig) (*GLMProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("glm api key is required")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = defaultGLMBaseURL
	}

	return &GLMProvider{
		config:  config,
		client:  &http.Client{},
		baseURL: baseURL,
		apiKey:  config.APIKey,
	}, nil
}

// Complete 非流式对话(阻塞式,返回完整响应)
func (gp *GLMProvider) Complete(ctx context.Context, messages []types.Message, opts *StreamOptions) (*CompleteResponse, error) {
	// 构建请求体(非流式)
	reqBody := gp.buildRequest(messages, opts)
	reqBody["stream"] = false // 关键:设置为非流式

	// 序列化（使用确定性序列化以优化 KV-Cache 命中率）
	jsonData, err := util.MarshalDeterministic(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// 创建HTTP请求
	endpoint := "/chat/completions"
	if !strings.HasSuffix(gp.baseURL, "/v4") && !strings.HasSuffix(gp.baseURL, "/v4/") {
		if strings.HasSuffix(gp.baseURL, "/") {
			endpoint = "v4/chat/completions"
		} else {
			endpoint = "/v4/chat/completions"
		}
	}
	req, err := http.NewRequestWithContext(ctx, "POST", gp.baseURL+endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+gp.apiKey)

	// 发送请求
	resp, err := gp.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		glmLog.Error(ctx, "API error response", map[string]any{"body": string(body)})
		return nil, fmt.Errorf("glm api error: %d - %s", resp.StatusCode, string(body))
	}

	// 解析完整响应
	var apiResp map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	glmLog.Debug(ctx, "complete API response", map[string]any{"response": apiResp})

	// 解析消息内容(使用与Deepseek相同的OpenAI兼容格式)
	message, err := gp.parseCompleteResponse(apiResp)
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
	}

	return &CompleteResponse{
		Message: message,
		Usage:   usage,
	}, nil
}

// Stream 流式对话
func (gp *GLMProvider) Stream(ctx context.Context, messages []types.Message, opts *StreamOptions) (<-chan StreamChunk, error) {
	// 构建请求体
	reqBody := gp.buildRequest(messages, opts)

	// 序列化（使用确定性序列化以优化 KV-Cache 命中率）
	jsonData, err := util.MarshalDeterministic(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// 记录请求内容（用于调试）
	if tools, ok := reqBody["tools"].([]map[string]any); ok && len(tools) > 0 {
		glmLog.Debug(ctx, "request body includes tools", map[string]any{"count": len(tools)})
		toolsJSON, _ := util.MarshalDeterministicIndent(reqBody["tools"], "", "  ")
		glmLog.Debug(ctx, "full tools definition", map[string]any{"tools": string(toolsJSON)})
	}

	// 创建HTTP请求
	// GLM API 路径：/v4/chat/completions
	endpoint := "/chat/completions"
	if !strings.HasSuffix(gp.baseURL, "/v4") && !strings.HasSuffix(gp.baseURL, "/v4/") {
		// 如果 baseURL 不包含 /v4，需要添加
		if strings.HasSuffix(gp.baseURL, "/") {
			endpoint = "v4/chat/completions"
		} else {
			endpoint = "/v4/chat/completions"
		}
	}
	req, err := http.NewRequestWithContext(ctx, "POST", gp.baseURL+endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+gp.apiKey)

	// 发送请求
	resp, err := gp.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		glmLog.Error(ctx, "API error response", map[string]any{"body": string(body)})
		return nil, fmt.Errorf("glm api error: %d - %s", resp.StatusCode, string(body))
	}

	glmLog.Debug(ctx, "API request successful", map[string]any{"status": resp.StatusCode})

	// 创建流式响应channel
	chunkCh := make(chan StreamChunk, 10)

	go gp.processStream(resp.Body, chunkCh)

	return chunkCh, nil
}

// buildRequest 构建请求体
func (gp *GLMProvider) buildRequest(messages []types.Message, opts *StreamOptions) map[string]any {
	ctx := context.Background() // 用于日志记录

	req := map[string]any{
		"model":    gp.config.Model,
		"messages": gp.convertMessages(messages),
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
			// GLM API 使用 system 字段
			req["system"] = opts.System
			glmLog.Debug(ctx, "system prompt", map[string]any{"length": len(opts.System)})
		} else if gp.systemPrompt != "" {
			req["system"] = gp.systemPrompt
		}

		if len(opts.Tools) > 0 {
			// GLM API 使用 tools 字段，格式与 OpenAI 兼容
			tools := make([]map[string]any, 0, len(opts.Tools))
			for _, tool := range opts.Tools {
				toolMap := map[string]any{
					"type": "function",
					"function": map[string]any{
						"name":        tool.Name,
						"description": tool.Description,
						"parameters":  tool.InputSchema,
					},
					// TODO: GLM API 暂不支持 input_examples，待官方支持后启用
					// 参考: https://open.bigmodel.cn/dev/api
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
			glmLog.Debug(ctx, "sending tools to API", map[string]any{"count": len(tools), "names": toolNames})
		}
	} else {
		req["max_tokens"] = 4096
		if gp.systemPrompt != "" {
			req["system"] = gp.systemPrompt
		}
	}

	return req
}

// convertMessages 转换消息格式（OpenAI 兼容格式）
func (gp *GLMProvider) convertMessages(messages []types.Message) []map[string]any {
	ctx := context.Background() // 用于日志记录
	result := make([]map[string]any, 0, len(messages))

	for _, msg := range messages {
		// 跳过system消息（已在opts中单独传递）
		if msg.Role == types.MessageRoleSystem {
			continue
		}

		// GLM API 使用 OpenAI 兼容格式
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
			glmLog.Debug(ctx, "added tool result message", map[string]any{"tool_call_id": tr.ToolUseID, "content_length": len(tr.Content)})
		}
	}

	return result
}

// processStream 处理流式响应
func (gp *GLMProvider) processStream(body io.ReadCloser, chunkCh chan<- StreamChunk) {
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
				glmLog.Debug(ctx, "non-data line", map[string]any{"line": line})
			}
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// 忽略特殊标记
		if data == "[DONE]" {
			glmLog.Debug(ctx, "received [DONE] marker", nil)
			break
		}

		// 解析JSON
		var event map[string]any
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			glmLog.Debug(ctx, "failed to parse JSON", map[string]any{"error": err, "data": data})
			continue
		}

		eventCount++
		glmLog.Debug(ctx, "stream event", map[string]any{"event_num": eventCount, "event": event})

		chunk := gp.parseStreamEvent(event)
		if chunk != nil {
			glmLog.Debug(ctx, "parsed chunk", map[string]any{"type": chunk.Type, "index": chunk.Index})
			chunkCh <- *chunk
		} else {
			glmLog.Debug(ctx, "no chunk parsed from event", nil)
		}
	}

	if err := scanner.Err(); err != nil {
		glmLog.Error(ctx, "scanner error", map[string]any{"error": err})
	}

	glmLog.Debug(ctx, "processed events", map[string]any{"total": eventCount})
}

// parseStreamEvent 解析流式事件
func (gp *GLMProvider) parseStreamEvent(event map[string]any) *StreamChunk {
	ctx := context.Background() // 用于日志记录

	// GLM API 使用 OpenAI 兼容格式
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

						chunk.Type = "content_block_start"
						chunk.Index = index

						// 构建工具调用信息（转换为 Anthropic 格式以便统一处理）
						toolInfo := map[string]any{
							"type": "tool_use",
						}

						if id, ok := toolCall["id"].(string); ok {
							toolInfo["id"] = id
						}

						if fn, ok := toolCall["function"].(map[string]any); ok {
							if name, ok := fn["name"].(string); ok {
								toolInfo["name"] = name
							}
							// arguments 会在 content_block_delta 中逐步接收
						}

						chunk.Delta = toolInfo
						glmLog.Debug(ctx, "received tool_use block", map[string]any{"index": index, "id": toolInfo["id"], "name": toolInfo["name"]})
						return chunk
					}
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
func (gp *GLMProvider) Config() *types.ModelConfig {
	return gp.config
}

// Capabilities 返回模型能力
func (gp *GLMProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		SupportToolCalling:  true,
		SupportSystemPrompt: true,
		SupportStreaming:    true,
		SupportVision:       false,
		MaxTokens:           8192,
		MaxToolsPerCall:     0,
		ToolCallingFormat:   "openai", // GLM 使用 OpenAI 兼容格式
	}
}

// SetSystemPrompt 设置系统提示词
func (gp *GLMProvider) SetSystemPrompt(prompt string) error {
	gp.systemPrompt = prompt
	return nil
}

// GetSystemPrompt 获取系统提示词
func (gp *GLMProvider) GetSystemPrompt() string {
	return gp.systemPrompt
}

// parseCompleteResponse 解析完整的非流式响应(OpenAI兼容格式)
func (gp *GLMProvider) parseCompleteResponse(apiResp map[string]any) (types.Message, error) {
	ctx := context.Background() // 用于日志记录
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
				glmLog.Warn(ctx, "failed to parse tool arguments", map[string]any{"error": err})
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
func (gp *GLMProvider) Close() error {
	return nil
}

// GLMFactory GLM工厂
type GLMFactory struct{}

// Create 创建GLM提供商
func (f *GLMFactory) Create(config *types.ModelConfig) (Provider, error) {
	return NewGLMProvider(config)
}
