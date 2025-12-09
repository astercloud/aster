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

var anthropicLog = logging.ForComponent("AnthropicProvider")

const (
	defaultAnthropicBaseURL = "https://api.anthropic.com"
	defaultAnthropicVersion = "2023-06-01"
)

// AnthropicProvider Anthropic模型提供商
type AnthropicProvider struct {
	config       *types.ModelConfig
	client       *http.Client
	baseURL      string
	version      string
	systemPrompt string // 系统提示词
}

// NewAnthropicProvider 创建Anthropic提供商
func NewAnthropicProvider(config *types.ModelConfig) (*AnthropicProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("anthropic api key is required")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = defaultAnthropicBaseURL
	}

	return &AnthropicProvider{
		config:  config,
		client:  &http.Client{},
		baseURL: baseURL,
		version: defaultAnthropicVersion,
	}, nil
}

// Complete 非流式对话(阻塞式,返回完整响应)
func (ap *AnthropicProvider) Complete(ctx context.Context, messages []types.Message, opts *StreamOptions) (*CompleteResponse, error) {
	// 构建请求体(非流式)
	reqBody := ap.buildRequest(messages, opts)
	reqBody["stream"] = false // 关键:设置为非流式

	// 序列化（使用确定性序列化以优化 KV-Cache 命中率）
	jsonData, err := util.MarshalDeterministic(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", ap.baseURL+"/v1/messages", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", ap.config.APIKey)
	req.Header.Set("anthropic-version", ap.version)

	// 发送请求
	resp, err := ap.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		anthropicLog.Error(ctx, "API error response", map[string]any{"status": resp.StatusCode, "body": string(body)})
		return nil, fmt.Errorf("anthropic api error: %d - %s", resp.StatusCode, string(body))
	}

	// 解析完整响应
	var apiResp map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	anthropicLog.Debug(ctx, "complete API response", map[string]any{"keys": getKeys(apiResp)})

	// 解析消息内容
	message, err := ap.parseCompleteResponse(apiResp)
	if err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	// 解析Token使用情况
	var usage *TokenUsage
	if usageData, ok := apiResp["usage"].(map[string]any); ok {
		usage = &TokenUsage{
			InputTokens:  int64(usageData["input_tokens"].(float64)),
			OutputTokens: int64(usageData["output_tokens"].(float64)),
		}
	}

	return &CompleteResponse{
		Message: message,
		Usage:   usage,
	}, nil
}

// Stream 流式对话
func (ap *AnthropicProvider) Stream(ctx context.Context, messages []types.Message, opts *StreamOptions) (<-chan StreamChunk, error) {
	// 构建请求体
	reqBody := ap.buildRequest(messages, opts)

	// 序列化（使用确定性序列化以优化 KV-Cache 命中率）
	jsonData, err := util.MarshalDeterministic(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// 记录请求内容（用于调试）
	if tools, ok := reqBody["tools"].([]map[string]any); ok && len(tools) > 0 {
		anthropicLog.Debug(ctx, "request body includes tools", map[string]any{"count": len(tools)})
		for _, tool := range tools {
			if name, ok := tool["name"].(string); ok {
				if schema, ok := tool["input_schema"].(map[string]any); ok {
					anthropicLog.Debug(ctx, "tool schema", map[string]any{"name": name, "schema": schema})
				}
			}
		}
		// 记录完整的工具定义（用于调试）
		toolsJSON, _ := util.MarshalDeterministicIndent(reqBody["tools"], "", "  ")
		anthropicLog.Debug(ctx, "full tools definition", map[string]any{"tools": string(toolsJSON)})
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", ap.baseURL+"/v1/messages", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", ap.config.APIKey)
	req.Header.Set("anthropic-version", ap.version)

	// 发送请求
	resp, err := ap.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("anthropic api error: %d - %s", resp.StatusCode, string(body))
	}

	// 创建流式响应channel
	chunkCh := make(chan StreamChunk, 10)

	go ap.processStream(resp.Body, chunkCh)

	return chunkCh, nil
}

// buildRequest 构建请求体
func (ap *AnthropicProvider) buildRequest(messages []types.Message, opts *StreamOptions) map[string]any {
	ctx := context.Background() // 用于日志记录

	req := map[string]any{
		"model":    ap.config.Model,
		"messages": ap.convertMessages(messages),
		"stream":   true,
	}

	if opts != nil {
		// max_tokens 是必需的，必须设置
		if opts.MaxTokens > 0 {
			req["max_tokens"] = opts.MaxTokens
		} else {
			req["max_tokens"] = 4096 // 默认值
		}

		if opts.Temperature > 0 {
			req["temperature"] = opts.Temperature
		}

		// 当有工具时，确保 max_tokens 足够大
		if len(opts.Tools) > 0 && opts.MaxTokens == 0 {
			req["max_tokens"] = 4096
		}

		if opts.System != "" {
			req["system"] = opts.System
			// 记录系统提示词长度和关键内容（用于调试）
			if len(opts.System) > 500 {
				anthropicLog.Debug(ctx, "system prompt", map[string]any{"length": len(opts.System), "preview": opts.System[:200]})
				// 检查是否包含工具手册
				if strings.Contains(opts.System, "### Tools Manual") {
					// 提取工具手册部分
					parts := strings.Split(opts.System, "### Tools Manual")
					if len(parts) > 1 {
						manualPreview := parts[1]
						if len(manualPreview) > 300 {
							manualPreview = manualPreview[:300] + "..."
						}
						anthropicLog.Debug(ctx, "tools manual found", map[string]any{"preview": manualPreview})
					}
				} else {
					anthropicLog.Warn(ctx, "tools manual NOT found in system prompt", nil)
				}
			} else {
				anthropicLog.Debug(ctx, "system prompt", map[string]any{"content": opts.System})
			}
		} else if ap.systemPrompt != "" {
			// 如果 opts 没有 system，使用存储的 systemPrompt
			req["system"] = ap.systemPrompt
		}

		if len(opts.Tools) > 0 {
			// 转换工具格式为 Anthropic API 格式
			tools := make([]map[string]any, 0, len(opts.Tools))
			for _, tool := range opts.Tools {
				toolMap := map[string]any{
					"name":         tool.Name,
					"description":  tool.Description,
					"input_schema": tool.InputSchema,
				}
				// 添加工具使用示例（如果有）
				// 参考 Anthropic 的 Tool Use Examples 功能
				if len(tool.InputExamples) > 0 {
					examples := make([]map[string]any, 0, len(tool.InputExamples))
					for _, ex := range tool.InputExamples {
						exMap := map[string]any{
							"description": ex.Description,
							"input":       ex.Input,
						}
						if ex.Output != nil {
							exMap["output"] = ex.Output
						}
						examples = append(examples, exMap)
					}
					toolMap["input_examples"] = examples
				}
				// PTC 支持: 添加 AllowedCallers 字段
				if len(tool.AllowedCallers) > 0 {
					toolMap["allowed_callers"] = tool.AllowedCallers
				}
				tools = append(tools, toolMap)
			}
			req["tools"] = tools
			toolNames := make([]string, len(tools))
			for i, t := range tools {
				toolNames[i] = t["name"].(string)
			}
			anthropicLog.Debug(ctx, "sending tools to API", map[string]any{"count": len(tools), "names": toolNames})

			// 添加 tool_choice 支持（如果指定）
			if opts.ToolChoice != nil {
				toolChoice := map[string]any{
					"type": opts.ToolChoice.Type,
				}
				if opts.ToolChoice.Name != "" {
					toolChoice["name"] = opts.ToolChoice.Name
				}
				if opts.ToolChoice.DisableParallelToolUse {
					toolChoice["disable_parallel_tool_use"] = true
				}
				req["tool_choice"] = toolChoice
				anthropicLog.Debug(ctx, "using tool_choice", map[string]any{"tool_choice": toolChoice})
			}
			// 记录每个工具的详细信息
			for _, tool := range tools {
				if name, ok := tool["name"].(string); ok {
					if schema, ok := tool["input_schema"].(map[string]any); ok {
						anthropicLog.Debug(ctx, "tool schema", map[string]any{"name": name, "schema": schema})
					}
					// 记录工具示例数量
					if examples, ok := tool["input_examples"].([]map[string]any); ok {
						anthropicLog.Debug(ctx, "tool examples", map[string]any{"name": name, "example_count": len(examples)})
					}
				}
			}
		}
	} else {
		req["max_tokens"] = 4096
		if ap.systemPrompt != "" {
			req["system"] = ap.systemPrompt
		}
	}

	return req
}

// convertMessages 转换消息格式
func (ap *AnthropicProvider) convertMessages(messages []types.Message) []map[string]any {
	result := make([]map[string]any, 0, len(messages))

	for _, msg := range messages {
		// 跳过system消息(system在opts中单独传递)
		if msg.Role == types.MessageRoleSystem {
			continue
		}

		var content any

		// 如果有 ContentBlocks，使用复杂格式
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
					toolUse := map[string]any{
						"type":  "tool_use",
						"id":    b.ID,
						"name":  b.Name,
						"input": b.Input,
					}
					// PTC 支持: 添加 Caller 字段
					if b.Caller != nil {
						toolUse["caller"] = map[string]any{
							"type": b.Caller.Type,
						}
						if b.Caller.ToolID != "" {
							toolUse["caller"].(map[string]any)["tool_id"] = b.Caller.ToolID
						}
					}
					blocks = append(blocks, toolUse)
				case *types.ToolResultBlock:
					blocks = append(blocks, map[string]any{
						"type":        "tool_result",
						"tool_use_id": b.ToolUseID,
						"content":     b.Content,
						"is_error":    b.IsError,
					})
				}
			}
			content = blocks
		} else {
			// 如果只有简单的 Content 字符串，转换为单个文本块
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

// processStream 处理流式响应
func (ap *AnthropicProvider) processStream(body io.ReadCloser, chunkCh chan<- StreamChunk) {
	defer close(chunkCh)
	defer func() { _ = body.Close() }()

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()

		// SSE格式: "data: {...}"
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// 忽略特殊标记
		if data == "[DONE]" {
			break
		}

		// 解析JSON
		var event map[string]any
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		chunk := ap.parseStreamEvent(event)
		if chunk != nil {
			chunkCh <- *chunk
		}
	}
}

// parseStreamEvent 解析流式事件
func (ap *AnthropicProvider) parseStreamEvent(event map[string]any) *StreamChunk {
	ctx := context.Background() // 用于日志记录
	eventType, _ := event["type"].(string)

	chunk := &StreamChunk{
		Type: eventType,
	}

	switch eventType {
	case "content_block_start":
		if index, ok := event["index"].(float64); ok {
			chunk.Index = int(index)
		}
		if contentBlock, ok := event["content_block"].(map[string]any); ok {
			chunk.Delta = contentBlock
			// 添加详细的调试日志
			if blockType, ok := contentBlock["type"].(string); ok {
				anthropicLog.Debug(ctx, "content_block_start", map[string]any{"type": blockType, "index": chunk.Index})
				switch blockType {
				case "tool_use":
					anthropicLog.Debug(ctx, "received tool_use block", map[string]any{"id": contentBlock["id"], "name": contentBlock["name"]})
				case "text":
					anthropicLog.Debug(ctx, "received text block instead of tool_use", nil)
				default:
					anthropicLog.Debug(ctx, "unknown block type", map[string]any{"type": blockType})
				}
			} else {
				anthropicLog.Debug(ctx, "content_block_start without type field", map[string]any{"block": contentBlock})
			}
		}

	case "content_block_delta":
		if index, ok := event["index"].(float64); ok {
			chunk.Index = int(index)
		}
		if delta, ok := event["delta"].(map[string]any); ok {
			chunk.Delta = delta
		}

	case "content_block_stop":
		if index, ok := event["index"].(float64); ok {
			chunk.Index = int(index)
		}

	case "message_delta":
		if delta, ok := event["delta"].(map[string]any); ok {
			chunk.Delta = delta
		}
		if usage, ok := event["usage"].(map[string]any); ok {
			chunk.Usage = &TokenUsage{
				InputTokens:  int64(usage["input_tokens"].(float64)),
				OutputTokens: int64(usage["output_tokens"].(float64)),
			}
		}
	}

	return chunk
}

// Config 返回配置
func (ap *AnthropicProvider) Config() *types.ModelConfig {
	return ap.config
}

// Capabilities 返回模型能力
func (ap *AnthropicProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		SupportToolCalling:  true,
		SupportSystemPrompt: true,
		SupportStreaming:    true,
		SupportVision:       false, // 根据模型决定
		MaxTokens:           200000,
		MaxToolsPerCall:     0, // 无限制
		ToolCallingFormat:   "anthropic",
	}
}

// SetSystemPrompt 设置系统提示词
func (ap *AnthropicProvider) SetSystemPrompt(prompt string) error {
	ap.systemPrompt = prompt
	return nil
}

// GetSystemPrompt 获取系统提示词
func (ap *AnthropicProvider) GetSystemPrompt() string {
	return ap.systemPrompt
}

// parseCompleteResponse 解析完整的非流式响应 (Anthropic格式)
func (ap *AnthropicProvider) parseCompleteResponse(apiResp map[string]any) (types.Message, error) {
	assistantContent := make([]types.ContentBlock, 0)

	// Anthropic 响应格式: content 是一个数组
	content, ok := apiResp["content"].([]any)
	if !ok || len(content) == 0 {
		return types.Message{}, fmt.Errorf("no content in response")
	}

	// 遍历所有 content blocks
	for _, item := range content {
		block, ok := item.(map[string]any)
		if !ok {
			continue
		}

		blockType, _ := block["type"].(string)

		switch blockType {
		case "text":
			// 文本块
			if text, ok := block["text"].(string); ok {
				assistantContent = append(assistantContent, &types.TextBlock{Text: text})
			}

		case "tool_use":
			// 工具调用块
			toolID, _ := block["id"].(string)
			toolName, _ := block["name"].(string)

			// 解析参数
			var input map[string]any
			if inputData, ok := block["input"].(map[string]any); ok {
				input = inputData
			} else {
				input = make(map[string]any)
			}

			// PTC 支持: 解析 Caller 字段
			var caller *types.ToolCaller
			if callerData, ok := block["caller"].(map[string]any); ok {
				callerType, _ := callerData["type"].(string)
				toolID, _ := callerData["tool_id"].(string)
				caller = &types.ToolCaller{
					Type:   callerType,
					ToolID: toolID,
				}
			}

			assistantContent = append(assistantContent, &types.ToolUseBlock{
				ID:     toolID,
				Name:   toolName,
				Input:  input,
				Caller: caller,
			})
		}
	}

	return types.Message{
		Role:          types.MessageRoleAssistant,
		ContentBlocks: assistantContent,
	}, nil
}

// getKeys 获取map的所有键(用于调试)
func getKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Close 关闭连接
func (ap *AnthropicProvider) Close() error {
	// HTTP客户端不需要显式关闭
	return nil
}

// AnthropicFactory Anthropic工厂
type AnthropicFactory struct{}

// Create 创建Anthropic提供商
func (f *AnthropicFactory) Create(config *types.ModelConfig) (Provider, error) {
	return NewAnthropicProvider(config)
}
