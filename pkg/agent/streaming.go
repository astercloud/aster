package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/astercloud/aster/pkg/middleware"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/session"
	"github.com/astercloud/aster/pkg/skills"
	"github.com/astercloud/aster/pkg/stream"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/types"
	"github.com/google/uuid"
)

// StreamingAgent 扩展接口 - 支持流式执行
// 参考 Google ADK-Go 的 Agent.Run() 设计，使用 stream.Reader
//
// 使用示例:
//
//	reader := agent.Stream(ctx, "Hello")
//	for {
//	    event, err := reader.Recv()
//	    if err == io.EOF { break }
//	    if err != nil {
//	        log.Printf("Error: %v", err)
//	        break
//	    }
//	    fmt.Printf("Event: %+v\n", event)
//	}
type StreamingAgent interface {
	// Stream 流式执行，返回事件流
	// 相比 Chat 方法的优势：
	// 1. 内存高效 - 按需生成事件，无需完整加载到内存
	// 2. 背压控制 - 客户端可控制消费速度
	// 3. 实时响应 - 可以立即处理每个事件，而不是等待所有事件
	// 4. 多消费者 - 支持 Copy 复制流给多个消费者
	Stream(ctx context.Context, message string, opts ...Option) *stream.Reader[*session.Event]
}

// Stream 实现流式执行接口
// 返回 stream.Reader，支持：
// - 流式生成事件
// - 客户端控制的取消
// - 与 LLM 流式 API 无缝集成
// - 多消费者支持（Copy, Merge, Transform）
func (a *Agent) Stream(ctx context.Context, message string, opts ...Option) *stream.Reader[*session.Event] {
	reader, writer := stream.Pipe[*session.Event](10)

	go func() {
		defer writer.Close()

		// 应用选项
		config := &streamConfig{}
		for _, opt := range opts {
			opt(config)
		}

		log.Printf("[Agent Stream] Starting stream for message: %s", truncate(message, 50))

		// 1. 前置验证
		if err := a.validateMessage(message); err != nil {
			writer.Send(nil, fmt.Errorf("validate message: %w", err))
			return
		}

		// 2. 创建用户消息
		userMsg := types.Message{
			Role:    types.RoleUser,
			Content: message,
			ContentBlocks: []types.ContentBlock{
				&types.TextBlock{Text: message},
			},
		}

		// 3. 检查 Slash Commands
		if a.commandExecutor != nil {
			if handled, err := a.handleSlashCommandForStream(ctx, &userMsg, writer); handled {
				if err != nil {
					writer.Send(nil, fmt.Errorf("slash command: %w", err))
				}
				return
			}
		}

		// 4. 应用 Skills 增强
		if a.skillInjector != nil {
			skillContext := skills.SkillContext{
				UserMessage: userMsg.GetContent(),
			}
			enhancedPrompt := a.skillInjector.EnhanceSystemPrompt(ctx, a.template.SystemPrompt, skillContext)
			a.template.SystemPrompt = enhancedPrompt
		}

		// 5. 入队消息
		a.mu.Lock()
		a.messages = append(a.messages, userMsg)
		a.mu.Unlock()

		// 6. 持久化消息
		if err := a.persistMessage(ctx, &userMsg); err != nil {
			log.Printf("[Agent Stream] Failed to persist message: %v", err)
		}

		// 7. 流式执行模型步骤
		// 检查上下文是否已取消
		for {
			select {
			case <-ctx.Done():
				writer.Send(nil, ctx.Err())
				return
			default:
			}

			// 执行流式模型推理
			done, err := a.runModelStepStreaming(ctx, writer)
			if err != nil {
				writer.Send(nil, fmt.Errorf("model step: %w", err))
				return
			}

			// 检查是否完成
			if done {
				log.Printf("[Agent Stream] Stream completed")
				return
			}

			// 如果没有完成，继续下一轮模型推理（通常是因为有工具调用）
			log.Printf("[Agent Stream] Continuing to next model inference round")
		}
	}()

	return reader
}

// StreamCollect 辅助函数 - 收集所有事件
// 用于向后兼容，将流式接口转换为批量结果
//
// 使用示例:
//
//	events, err := StreamCollect(agent.Stream(ctx, "Hello"))
//	if err != nil {
//	    return err
//	}
//	for _, event := range events {
//	    fmt.Println(event)
//	}
func StreamCollect(reader *stream.Reader[*session.Event]) ([]*session.Event, error) {
	var events []*session.Event
	for {
		event, err := reader.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return events, err
		}
		if event != nil {
			events = append(events, event)
		}
	}
	return events, nil
}

// StreamFirst 辅助函数 - 获取第一个事件
func StreamFirst(reader *stream.Reader[*session.Event]) (*session.Event, error) {
	event, err := reader.Recv()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("no events in stream")
		}
		return nil, err
	}
	return event, nil
}

// StreamLast 辅助函数 - 获取最后一个事件
func StreamLast(reader *stream.Reader[*session.Event]) (*session.Event, error) {
	var lastEvent *session.Event
	var lastErr error

	for {
		event, err := reader.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			lastErr = err
			continue
		}
		if event != nil {
			lastEvent = event
		}
	}

	if lastErr != nil {
		return lastEvent, lastErr
	}
	if lastEvent == nil {
		return nil, fmt.Errorf("no events in stream")
	}
	return lastEvent, nil
}

// StreamFilter 辅助函数 - 过滤事件
func StreamFilter(reader *stream.Reader[*session.Event], predicate func(*session.Event) bool) *stream.Reader[*session.Event] {
	outReader, outWriter := stream.Pipe[*session.Event](10)

	go func() {
		defer outWriter.Close()
		for {
			event, err := reader.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				outWriter.Send(nil, err)
				return
			}
			if event != nil && predicate(event) {
				if outWriter.Send(event, nil) {
					return
				}
			}
		}
	}()

	return outReader
}

// streamConfig 流式执行配置
type streamConfig struct {
	// 未来可扩展的配置项
}

// Option 流式执行选项
type Option func(*streamConfig)

// validateMessage 验证消息
func (a *Agent) validateMessage(message string) error {
	if message == "" {
		return fmt.Errorf("message cannot be empty")
	}
	return nil
}

// handleSlashCommandForStream 处理 Slash Command（流式版本）
func (a *Agent) handleSlashCommandForStream(ctx context.Context, msg *types.Message, writer *stream.Writer[*session.Event]) (bool, error) {
	// 检查是否为 slash command
	if !a.commandExecutor.IsSlashCommand(msg.Content) {
		return false, nil
	}

	// 执行 slash command
	result, err := a.commandExecutor.Execute(ctx, msg.Content, make(map[string]string))
	if err != nil {
		return true, err
	}

	// 生成响应事件
	event := &session.Event{
		ID:        generateEventID(),
		Timestamp: a.createdAt,
		AgentID:   a.id,
		Author:    "system",
		Content: types.Message{
			Role:    types.RoleAssistant,
			Content: result,
		},
	}

	// 发送事件
	writer.Send(event, nil)
	return true, nil
}

// runModelStepStreaming 流式执行模型步骤
// 返回: (done, error)
func (a *Agent) runModelStepStreaming(ctx context.Context, writer *stream.Writer[*session.Event]) (bool, error) {
	// 1. 准备消息
	a.mu.RLock()
	messages := make([]types.Message, len(a.messages))
	copy(messages, a.messages)
	a.mu.RUnlock()

	// 2. 通过 Middleware 调用 LLM
	var resp *middleware.ModelResponse
	var err error

	log.Printf("[Agent Stream] Using middleware stack: %v", a.middlewareStack != nil)

	if a.middlewareStack != nil {
		// 使用 Middleware Stack
		log.Printf("[Agent Stream] Using middleware stack for streaming")
		// 转换工具列表
		toolList := make([]tools.Tool, 0, len(a.toolMap))
		for _, tool := range a.toolMap {
			toolList = append(toolList, tool)
		}

		req := &middleware.ModelRequest{
			Messages:     messages,
			SystemPrompt: a.template.SystemPrompt,
			Tools:        toolList,
			Metadata:     make(map[string]interface{}),
		}

		// 创建适配器处理provider调用
		finalHandler := func(ctx context.Context, req *middleware.ModelRequest) (*middleware.ModelResponse, error) {
			// 转换工具定义
			toolSchemas := make([]provider.ToolSchema, len(req.Tools))
			for i, tool := range req.Tools {
				toolSchemas[i] = provider.ToolSchema{
					Name:        tool.Name(),
					Description: tool.Description(),
					InputSchema: tool.InputSchema(),
				}
			}

			// 创建Provider选项
			streamOpts := &provider.StreamOptions{
				Tools:       toolSchemas,
				System:      req.SystemPrompt,
				Temperature: 0.7,
			}

			// 调用Provider - 使用Stream方法支持流式响应
			log.Printf("[Agent Stream] Calling provider.Stream() for middleware")
			chunkCh, err := a.provider.Stream(ctx, req.Messages, streamOpts)
			if err != nil {
				return nil, err
			}

			// 直接将流式响应发送到WebSocket，但这里先收集用于兼容旧逻辑
			log.Printf("[Agent Stream] Starting to collect chunks from provider")
			var assistantMessage types.Message
			var contentBlocks []types.ContentBlock

			var reasoningContent strings.Builder
			for chunk := range chunkCh {
				log.Printf("[Agent Stream] Middleware chunk: type=%s, textDelta=%s", chunk.Type, truncate(chunk.TextDelta, 30))

				switch chunk.Type {
				// OpenAI 兼容格式 - 直接文本类型
				case "text":
					if chunk.TextDelta != "" {
						log.Printf("[Agent Stream] Middleware text chunk: %s", truncate(chunk.TextDelta, 50))
						contentBlocks = append(contentBlocks, &types.TextBlock{Text: chunk.TextDelta})
					}
				// 推理内容 (Kimi thinking, DeepSeek reasoner)
				case "reasoning":
					if chunk.Reasoning != nil && chunk.Reasoning.ThoughtDelta != "" {
						log.Printf("[Agent Stream] Middleware reasoning chunk: %s", truncate(chunk.Reasoning.ThoughtDelta, 50))
						reasoningContent.WriteString(chunk.Reasoning.ThoughtDelta)
						// TODO: 发送 think_chunk 事件到前端
					}
				// Anthropic 格式 - content_block_delta
				case "content_block_delta":
					if delta, ok := chunk.Delta.(map[string]interface{}); ok {
						if chunkType, ok := delta["type"].(string); ok {
							switch chunkType {
							case "text_delta":
								if text, ok := delta["text"].(string); ok {
									contentBlocks = append(contentBlocks, &types.TextBlock{Text: text})
								}
							case "arguments":
								// 处理工具调用参数 - 暂时跳过
							}
						}
					}
				}
			}
			if reasoningContent.Len() > 0 {
				log.Printf("[Agent Stream] Total reasoning content: %d chars", reasoningContent.Len())
			}

			log.Printf("[Agent Stream] Middleware collected %d content blocks", len(contentBlocks))
			assistantMessage.ContentBlocks = contentBlocks
			assistantMessage.Role = types.RoleAssistant

			return &middleware.ModelResponse{
				Message:  assistantMessage,
				Metadata: req.Metadata,
			}, nil
		}

		resp, err = a.middlewareStack.ExecuteModelCall(ctx, req, finalHandler)
	} else {
		log.Printf("[Agent Stream] Using direct provider call (no middleware)")
		// 转换工具定义
		toolSchemas := make([]provider.ToolSchema, len(a.getToolsForProvider()))
		for i, tool := range a.getToolsForProvider() {
			toolSchemas[i] = provider.ToolSchema{
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: tool.InputSchema,
			}
		}

		// 直接调用 Provider - 使用Stream方法支持流式响应
		streamOpts := &provider.StreamOptions{
			Tools:       toolSchemas,
			System:      a.template.SystemPrompt,
			Temperature: 0.7,
		}
		log.Printf("[Agent Stream] Calling provider.Stream() directly")
		chunkCh, err := a.provider.Stream(ctx, messages, streamOpts)
		if err != nil {
			return false, err
		}

		// 实时处理流式响应
		var assistantMessage types.Message
		var contentBlocks []types.ContentBlock
		var toolCalls []types.ToolCall
		var currentToolCall *types.ToolCall
		var argumentsBuilder strings.Builder

		log.Printf("[Agent Stream] 开始处理流式响应")

		for chunk := range chunkCh {
			log.Printf("[Agent Stream] 处理chunk: type=%s, index=%d, delta=%v, textDelta=%s", chunk.Type, chunk.Index, chunk.Delta, truncate(chunk.TextDelta, 50))

			switch chunk.Type {
			// OpenAI 兼容格式 - 直接文本类型
			case "text":
				if chunk.TextDelta != "" {
					log.Printf("[Agent Stream] 收到文本增量(text): %s", truncate(chunk.TextDelta, 50))
					contentBlocks = append(contentBlocks, &types.TextBlock{Text: chunk.TextDelta})

					// 立即为这个text chunk生成事件并yield
					event := &session.Event{
						ID:        generateEventID(),
						Timestamp: a.createdAt,
						AgentID:   a.id,
						Author:    "assistant",
						Content: types.Message{
							Role:          types.RoleAssistant,
							ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: chunk.TextDelta}},
						},
						Actions: session.EventActions{},
					}

					// 立即发送事件到流
					if writer.Send(event, nil) {
						log.Printf("[Agent Stream] Client cancelled stream during text streaming")
						return true, nil
					}
				}
			// 推理内容 (Kimi thinking, DeepSeek reasoner)
			case "reasoning":
				if chunk.Reasoning != nil && chunk.Reasoning.ThoughtDelta != "" {
					log.Printf("[Agent Stream] 收到推理增量: %s", truncate(chunk.Reasoning.ThoughtDelta, 50))

					// 发送带有 Reasoning 的事件
					event := &session.Event{
						ID:        generateEventID(),
						Timestamp: a.createdAt,
						AgentID:   a.id,
						Author:    "assistant",
						Reasoning: chunk.Reasoning.ThoughtDelta,
						Actions:   session.EventActions{},
					}

					// 立即发送事件到流
					if writer.Send(event, nil) {
						log.Printf("[Agent Stream] Client cancelled stream during reasoning streaming")
						return true, nil
					}
				}
			// Anthropic 格式 - content_block_delta
			case "content_block_delta":
				if delta, ok := chunk.Delta.(map[string]interface{}); ok {
					if chunkType, ok := delta["type"].(string); ok {
						switch chunkType {
						case "text_delta":
							if text, ok := delta["text"].(string); ok {
								log.Printf("[Agent Stream] 收到文本增量: %s", truncate(text, 50))
								contentBlocks = append(contentBlocks, &types.TextBlock{Text: text})

								// 立即为这个text chunk生成事件并yield
								event := &session.Event{
									ID:        generateEventID(),
									Timestamp: a.createdAt,
									AgentID:   a.id,
									Author:    "assistant",
									Content: types.Message{
										Role:          types.RoleAssistant,
										ContentBlocks: []types.ContentBlock{&types.TextBlock{Text: text}},
									},
									Actions: session.EventActions{},
								}

								// 立即发送事件到流
								if writer.Send(event, nil) {
									log.Printf("[Agent Stream] Client cancelled stream during text streaming")
									return true, nil
								}
							}
						case "arguments":
							// 处理工具调用参数
							if args, ok := delta["arguments"].(string); ok {
								log.Printf("[Agent Stream] 收到工具参数增量: %s", truncate(args, 100))
								argumentsBuilder.WriteString(args)
							}
						}
					}
				}
			case "content_block_start":
				// 开始新的工具调用
				if toolInfo, ok := chunk.Delta.(map[string]interface{}); ok {
					if name, ok := toolInfo["name"].(string); ok {
						if id, ok := toolInfo["id"].(string); ok {
							log.Printf("[Agent Stream] 开始工具调用: name=%s, id=%s", name, id)
							currentToolCall = &types.ToolCall{
								ID:   id,
								Name: name,
							}
							argumentsBuilder.Reset()
						}
					}
				}
			case "message_delta":
				// 消息结束，处理完整的工具调用
				if currentToolCall != nil && argumentsBuilder.Len() > 0 {
					argsStr := argumentsBuilder.String()
					log.Printf("[Agent Stream] 工具调用完成，参数: %s", truncate(argsStr, 200))

					// 解析JSON参数
					var input map[string]interface{}
					if err := json.Unmarshal([]byte(argsStr), &input); err != nil {
						log.Printf("[Agent Stream] 解析工具参数失败: %v", err)
						input = make(map[string]interface{})
					}

					currentToolCall.Arguments = input
					toolCalls = append(toolCalls, *currentToolCall)
					currentToolCall = nil
				}
			}
		}

		// 构建最终消息
		assistantMessage.ContentBlocks = contentBlocks
		assistantMessage.Role = types.RoleAssistant
		assistantMessage.ToolCalls = toolCalls

		log.Printf("[Agent Stream] 流式处理完成 - 内容块数: %d, 工具调用数: %d", len(contentBlocks), len(toolCalls))

		resp = &middleware.ModelResponse{
			Message:  assistantMessage,
			Metadata: make(map[string]interface{}),
		}
	}

	if err != nil {
		return false, err
	}

	// 3. 处理响应
	a.mu.Lock()
	a.messages = append(a.messages, resp.Message)
	a.mu.Unlock()

	// 4. 检查是否有工具调用
	log.Printf("[Agent Stream] 检查工具调用 - ToolCalls数量: %d", len(resp.Message.ToolCalls))
	if len(resp.Message.ToolCalls) > 0 {
		log.Printf("[Agent Stream] 发现 %d 个工具调用，开始执行", len(resp.Message.ToolCalls))
		for i, tc := range resp.Message.ToolCalls {
			log.Printf("[Agent Stream] 工具调用 #%d: ID=%s, Name=%s, Args=%v", i+1, tc.ID, tc.Name, tc.Arguments)
		}

		// 执行工具调用
		if err := a.executeToolCalls(ctx, resp.Message.ToolCalls); err != nil {
			log.Printf("[Agent Stream] 工具调用执行失败: %v", err)
			return false, err
		}

		log.Printf("[Agent Stream] 工具调用执行完成，继续下一轮模型推理")
		// 继续下一轮模型推理来处理工具执行结果
		return false, nil
	}

	log.Printf("[Agent Stream] 没有工具调用，生成最终响应事件")

	// 5. 生成事件
	event := &session.Event{
		ID:        generateEventID(),
		Timestamp: a.createdAt,
		AgentID:   a.id,
		Author:    "assistant",
		Content:   resp.Message,
		Actions:   session.EventActions{},
	}

	// 6. 持久化事件
	if err := a.persistEvent(ctx, event); err != nil {
		log.Printf("[Agent Stream] Failed to persist event: %v", err)
	}

	// 7. 发布事件到 EventBus (暂时禁用，因为events包未实现)
	// TODO: 实现事件发布系统

	// 8. 发送事件到流
	if writer.Send(event, nil) {
		log.Printf("[Agent Stream] Client cancelled stream during event yield")
		return true, nil
	}

	return true, nil
}

// executeToolCalls 执行工具调用
func (a *Agent) executeToolCalls(ctx context.Context, toolCalls []types.ToolCall) error {
	results := make([]types.Message, len(toolCalls))

	for i, call := range toolCalls {
		tool, ok := a.toolMap[call.Name]
		if !ok {
			results[i] = types.Message{
				Role:       types.RoleTool,
				ToolCallID: call.ID,
				Content:    fmt.Sprintf("Error: tool '%s' not found", call.Name),
			}
			continue
		}

		// 执行工具
		req := &tools.ExecuteRequest{
			Tool:    tool,
			Input:   call.Arguments,
			Context: a.buildToolContext(ctx),
		}
		execResult := a.executor.Execute(ctx, req)
		if execResult.Error != nil {
			results[i] = types.Message{
				Role:       types.RoleTool,
				ToolCallID: call.ID,
				Content:    fmt.Sprintf("Error: %v", execResult.Error),
			}
			continue
		}

		results[i] = types.Message{
			Role:       types.RoleTool,
			ToolCallID: call.ID,
			Content:    fmt.Sprint(execResult.Output),
		}
	}

	// 追加工具结果到消息历史
	a.mu.Lock()
	a.messages = append(a.messages, results...)
	a.mu.Unlock()

	return nil
}

// getToolsForProvider 获取 Provider 格式的工具定义
func (a *Agent) getToolsForProvider() []types.ToolDefinition {
	tools := make([]types.ToolDefinition, 0, len(a.toolMap))
	for _, tool := range a.toolMap {
		tools = append(tools, types.ToolDefinition{
			Name:        tool.Name(),
			Description: tool.Description(),
			InputSchema: tool.InputSchema(),
		})
	}
	return tools
}

// persistMessage 持久化消息
func (a *Agent) persistMessage(ctx context.Context, msg *types.Message) error {
	// TODO: 实现消息持久化
	return nil
}

// persistEvent 持久化事件
func (a *Agent) persistEvent(ctx context.Context, event *session.Event) error {
	// TODO: 实现事件持久化到 SessionService
	return nil
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// generateEventID 生成事件 ID
func generateEventID() string {
	return "evt_" + uuid.New().String()
}
