package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/middleware"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/types"
)

var procLog = logging.ForComponent("AgentProcessor")

// processMessages 处理消息队列
func (a *Agent) processMessages(ctx context.Context) {
	procLog.Info(ctx, "processMessages started", map[string]any{"agent_id": a.id})

	a.mu.Lock()
	if a.state != types.AgentStateReady {
		procLog.Warn(ctx, "agent not ready, skipping", map[string]any{"agent_id": a.id, "state": a.state})
		a.mu.Unlock()
		return // 已经在处理中
	}
	a.state = types.AgentStateWorking
	a.iterationCount = 0 // 重置迭代计数
	initialMsgCount := len(a.messages)
	procLog.Info(ctx, "agent state changed to working", map[string]any{"agent_id": a.id, "message_count": initialMsgCount})
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.state = types.AgentStateReady
		// 检查是否有新的用户消息需要处理
		// 只有当最后一条消息是用户消息时才需要重新处理
		// （避免 assistant 响应触发无限循环）
		hasNewUserMessage := false
		if len(a.messages) > initialMsgCount {
			lastMsg := a.messages[len(a.messages)-1]
			hasNewUserMessage = lastMsg.Role == types.MessageRoleUser
		}
		a.mu.Unlock()

		// 如果有新的用户消息，重新触发处理
		if hasNewUserMessage {
			go a.processMessages(ctx)
		}
	}()

	// 发送状态变更事件
	a.eventBus.EmitMonitor(&types.MonitorStateChangedEvent{
		State: types.AgentStateWorking,
	})

	// 设置断点
	a.setBreakpoint(types.BreakpointPreModel)

	procLog.Info(ctx, "calling runModelStep", map[string]any{"agent_id": a.id})

	// 调用模型
	if err := a.runModelStep(ctx); err != nil {
		procLog.Error(ctx, "runModelStep failed", map[string]any{"agent_id": a.id, "error": err.Error()})
		a.eventBus.EmitMonitor(&types.MonitorErrorEvent{
			Severity: "error",
			Phase:    "model",
			Message:  err.Error(),
		})
	}

	procLog.Info(ctx, "runModelStep completed, sending done event", map[string]any{"agent_id": a.id})

	// 发送完成事件
	a.eventBus.EmitProgress(&types.ProgressDoneEvent{
		Step:   a.stepCount,
		Reason: "completed",
	})

	// 发送状态变更事件
	a.eventBus.EmitMonitor(&types.MonitorStateChangedEvent{
		State: types.AgentStateReady,
	})
}

// runModelStep 运行模型步骤
func (a *Agent) runModelStep(ctx context.Context) error {
	procLog.Info(ctx, "runModelStep started", map[string]any{"agent_id": a.id})

	// 检查执行模式
	executionMode := a.getExecutionMode()
	if executionMode == types.ExecutionModeNonStreaming {
		procLog.Info(ctx, "using NON-STREAMING mode (fast execution)", map[string]any{"agent_id": a.id})
		return a.runNonStreamingStep(ctx)
	}

	procLog.Info(ctx, "using STREAMING mode (real-time feedback)", map[string]any{"agent_id": a.id})
	a.setBreakpoint(types.BreakpointStreamingModel)

	// 准备工具Schema（包含使用示例）
	toolSchemas := make([]provider.ToolSchema, 0, len(a.toolMap))
	for _, tool := range a.toolMap {
		schema := provider.ToolSchema{
			Name:        tool.Name(),
			Description: tool.Description(),
			InputSchema: tool.InputSchema(),
		}
		// 检查工具是否实现了 ExampleableTool 接口
		if exampleable, ok := tool.(tools.ExampleableTool); ok {
			examples := exampleable.Examples()
			if len(examples) > 0 {
				providerExamples := make([]provider.ToolExample, len(examples))
				for i, ex := range examples {
					providerExamples[i] = provider.ToolExample{
						Description: ex.Description,
						Input:       ex.Input,
						Output:      ex.Output,
					}
				}
				schema.InputExamples = providerExamples
			}
		}
		toolSchemas = append(toolSchemas, schema)
	}
	toolNames := make([]string, len(toolSchemas))
	for i, ts := range toolSchemas {
		toolNames[i] = ts.Name
	}
	procLog.Debug(ctx, "prepared tool schemas", map[string]any{"agent_id": a.id, "count": len(toolSchemas), "names": toolNames})

	// 调用模型
	// 确保系统提示词包含工具手册（如果还没有注入）
	a.mu.RLock()
	hasManual := strings.Contains(a.template.SystemPrompt, "### Tools Manual")
	toolMapSize := len(a.toolMap)
	currentSystemPrompt := a.template.SystemPrompt
	messages := a.messages // 复制当前消息列表
	a.mu.RUnlock()

	if !hasManual && toolMapSize > 0 {
		procLog.Debug(ctx, "manual not found, injecting", map[string]any{"agent_id": a.id, "tool_map_size": toolMapSize})
		a.injectToolManual()
		a.mu.RLock()
		currentSystemPrompt = a.template.SystemPrompt
		hasManual = strings.Contains(currentSystemPrompt, "### Tools Manual")
		a.mu.RUnlock()
		procLog.Debug(ctx, "after injection", map[string]any{"agent_id": a.id, "system_prompt_length": len(currentSystemPrompt), "contains_manual": hasManual})
	} else if toolMapSize == 0 {
		procLog.Debug(ctx, "no tools in toolMap, cannot inject manual", map[string]any{"agent_id": a.id})
	}

	procLog.Debug(ctx, "final system prompt", map[string]any{"agent_id": a.id, "length": len(currentSystemPrompt), "contains_manual": strings.Contains(currentSystemPrompt, "### Tools Manual")})

	// 通过 Middleware Stack 调用模型 (Phase 6C)
	var assistantMessage types.Message
	var modelErr error

	procLog.Info(ctx, "preparing to call LLM", map[string]any{"agent_id": a.id, "message_count": len(messages), "has_middleware": a.middlewareStack != nil})

	if a.middlewareStack != nil {
		// 使用 middleware stack
		procLog.Info(ctx, "using middleware stack for LLM call", map[string]any{"agent_id": a.id})
		req := &middleware.ModelRequest{
			Messages:     messages,
			SystemPrompt: currentSystemPrompt,
			Tools:        nil, // TODO: 转换 toolMap 为 []tools.Tool
			Metadata:     make(map[string]any),
		}

		// 注入 EventEmitter，让中间件可以发送事件
		req.Metadata[middleware.MetadataKeyEventEmitter] = middleware.EventEmitterFunc(func(event types.EventType) {
			if event == nil {
				return
			}
			switch event.Channel() {
			case types.ChannelProgress:
				a.eventBus.EmitProgress(event)
			case types.ChannelControl:
				a.eventBus.EmitControl(event)
			case types.ChannelMonitor:
				a.eventBus.EmitMonitor(event)
			}
		})

		// 定义 finalHandler: 实际调用 Provider
		finalHandler := func(ctx context.Context, req *middleware.ModelRequest) (*middleware.ModelResponse, error) {
			procLog.Info(ctx, "finalHandler: calling provider.Stream", map[string]any{"agent_id": a.id, "message_count": len(req.Messages)})
			streamOpts := &provider.StreamOptions{
				Tools:     toolSchemas,
				MaxTokens: 4096,
				System:    req.SystemPrompt,
			}

			stream, err := a.provider.Stream(ctx, req.Messages, streamOpts)
			if err != nil {
				procLog.Error(ctx, "provider.Stream failed", map[string]any{"agent_id": a.id, "error": err.Error()})
				return nil, fmt.Errorf("stream model: %w", err)
			}
			procLog.Info(ctx, "provider.Stream returned, processing response", map[string]any{"agent_id": a.id})

			// 处理流式响应
			message, err := a.handleStreamResponse(ctx, stream)
			if err != nil {
				return nil, err
			}

			return &middleware.ModelResponse{
				Message:  message,
				Metadata: make(map[string]any),
			}, nil
		}

		// 通过 middleware stack 执行
		procLog.Info(ctx, "calling middlewareStack.ExecuteModelCall", map[string]any{"agent_id": a.id})
		resp, err := a.middlewareStack.ExecuteModelCall(ctx, req, finalHandler)
		if err != nil {
			procLog.Error(ctx, "middlewareStack.ExecuteModelCall failed", map[string]any{"agent_id": a.id, "error": err.Error()})
			modelErr = err
		} else {
			procLog.Info(ctx, "middlewareStack.ExecuteModelCall succeeded", map[string]any{"agent_id": a.id})
			assistantMessage = resp.Message
		}
	} else {
		// 没有 middleware, 直接调用
		streamOpts := &provider.StreamOptions{
			Tools:     toolSchemas,
			MaxTokens: 4096,
			System:    currentSystemPrompt,
		}

		stream, err := a.provider.Stream(ctx, messages, streamOpts)
		if err != nil {
			modelErr = err
		} else {
			assistantMessage, err = a.handleStreamResponse(ctx, stream)
			if err != nil {
				modelErr = err
			}
		}
	}

	// 处理模型调用错误
	if modelErr != nil {
		return fmt.Errorf("model call: %w", modelErr)
	}

	// 保存助手消息
	a.mu.Lock()
	a.messages = append(a.messages, assistantMessage)

	// ✅ 修复：保存前在内存中修剪，避免 Store 出现超限状态
	if a.shouldTrimMessages() {
		a.messages = a.trimMessagesInMemory(a.messages, a.config.Store.MaxMessages)
		procLog.Debug(ctx, "messages trimmed in memory before save", map[string]any{
			"agent_id":     a.id,
			"max_messages": a.config.Store.MaxMessages,
			"actual_count": len(a.messages),
		})
	}
	a.mu.Unlock()

	// 持久化（已修剪的消息）
	if err := a.deps.Store.SaveMessages(ctx, a.id, a.messages); err != nil {
		return fmt.Errorf("save messages: %w", err)
	}

	// 检查是否有工具调用
	toolUses := make([]*types.ToolUseBlock, 0)
	for _, block := range assistantMessage.ContentBlocks {
		if tu, ok := block.(*types.ToolUseBlock); ok {
			toolUses = append(toolUses, tu)
		}
	}

	procLog.Debug(ctx, "found tool uses in response", map[string]any{"agent_id": a.id, "count": len(toolUses)})
	if len(toolUses) > 0 {
		for _, tu := range toolUses {
			procLog.Debug(ctx, "tool use", map[string]any{"agent_id": a.id, "name": tu.Name, "id": tu.ID, "input": tu.Input})
		}
		a.setBreakpoint(types.BreakpointToolPending)
		return a.executeTools(ctx, toolUses)
	} else {
		procLog.Debug(ctx, "no tool uses found, only text response", map[string]any{"agent_id": a.id})
	}

	return nil
}

// executeTools 执行工具
func (a *Agent) executeTools(ctx context.Context, toolUses []*types.ToolUseBlock) error {
	toolResults := make([]types.ContentBlock, 0, len(toolUses))

	for _, tu := range toolUses {
		result := a.executeSingleTool(ctx, tu)
		toolResults = append(toolResults, result)
	}

	// 保存工具结果
	a.mu.Lock()
	a.messages = append(a.messages, types.Message{
		Role:          types.MessageRoleUser,
		ContentBlocks: toolResults,
	})

	// ✅ 修复：保存前在内存中修剪
	if a.shouldTrimMessages() {
		a.messages = a.trimMessagesInMemory(a.messages, a.config.Store.MaxMessages)
		procLog.Debug(ctx, "messages trimmed in memory before save (after tool execution)", map[string]any{
			"agent_id":     a.id,
			"max_messages": a.config.Store.MaxMessages,
			"actual_count": len(a.messages),
		})
	}

	a.stepCount++
	a.mu.Unlock()

	// 持久化（已修剪的消息）
	if err := a.deps.Store.SaveMessages(ctx, a.id, a.messages); err != nil {
		return fmt.Errorf("save messages: %w", err)
	}

	// 持久化工具记录
	records := make([]types.ToolCallRecord, 0, len(a.toolRecords))
	for _, record := range a.toolRecords {
		records = append(records, *record)
	}
	if err := a.deps.Store.SaveToolCallRecords(ctx, a.id, records); err != nil {
		return fmt.Errorf("save tool records: %w", err)
	}

	// 检查迭代限制（防止无限循环）
	a.mu.Lock()
	a.iterationCount++
	currentIter := a.iterationCount
	maxIter := a.maxIterations
	if maxIter <= 0 {
		maxIter = 50
	}
	a.mu.Unlock()

	if currentIter > maxIter {
		procLog.Warn(ctx, "iteration limit reached, waiting for user confirmation", map[string]any{
			"agent_id": a.id, "iteration": currentIter, "max": maxIter,
		})
		
		// 发送迭代限制事件，等待用户确认是否继续
		a.eventBus.EmitControl(&types.ControlIterationLimitEvent{
			CurrentIteration: currentIter,
			MaxIteration:     maxIter,
			Message:          fmt.Sprintf("已执行 %d 次迭代，达到安全上限。是否继续？", currentIter),
		})
		
		// 等待用户决策
		select {
		case decision := <-a.iterationContinueCh:
			if !decision {
				return fmt.Errorf("iteration stopped by user after %d iterations", currentIter)
			}
			// 用户确认继续，重置迭代计数并继续
			a.mu.Lock()
			a.iterationCount = 0
			a.mu.Unlock()
			procLog.Info(ctx, "user confirmed to continue, resetting iteration count", map[string]any{"agent_id": a.id})
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	procLog.Debug(ctx, "streaming iteration", map[string]any{"agent_id": a.id, "iteration": currentIter, "max": maxIter})

	// 继续处理
	return a.runModelStep(ctx)
}

// executeSingleTool 执行单个工具
func (a *Agent) executeSingleTool(ctx context.Context, tu *types.ToolUseBlock) types.ContentBlock {
	// Plan 模式检查：验证工具调用是否允许
	if a.planMode != nil && a.planMode.IsActive() {
		allowed, reason := a.planMode.ValidateToolCall(tu.Name, tu.Input)
		if !allowed {
			errorMsg := fmt.Sprintf("Plan Mode restriction: %s", reason)
			a.eventBus.EmitProgress(&types.ProgressToolErrorEvent{
				Call: types.ToolCallSnapshot{
					ID:        tu.ID,
					Name:      tu.Name,
					State:     types.ToolCallStateFailed,
					Arguments: tu.Input,
				},
				Error: errorMsg,
			})
			return &types.ToolResultBlock{
				ToolUseID: tu.ID,
				Content:   fmt.Sprintf(`{"ok":false,"error":"%s","plan_mode":true}`, errorMsg),
				IsError:   true,
			}
		}
	}

	//权限检查
	if a.permissionInspector != nil {
		call := &types.ToolCallSnapshot{
			ID:        tu.ID,
			Name:      tu.Name,
			Arguments: tu.Input,
		}
		checkResult, err := a.permissionInspector.Check(ctx, call)
		if err != nil {
			errorMsg := fmt.Sprintf("Permission check error: %v", err)
			a.eventBus.EmitProgress(&types.ProgressToolErrorEvent{
				Call: types.ToolCallSnapshot{
					ID:        tu.ID,
					Name:      tu.Name,
					State:     types.ToolCallStateFailed,
					Arguments: tu.Input,
				},
				Error: errorMsg,
			})
			return &types.ToolResultBlock{
				ToolUseID: tu.ID,
				Content:   fmt.Sprintf(`{"ok":false,"error":"%s"}`, errorMsg),
				IsError:   true,
			}
		}

		if checkResult != nil {
			// 应用输入修改
			if checkResult.UpdatedInput != nil {
				tu.Input = checkResult.UpdatedInput
			}

			if !checkResult.Allowed {
				if checkResult.NeedsApproval {
					// 创建等待 channel
					decisionCh := make(chan string, 1)
					a.mu.Lock()
					a.pendingPermissions[tu.ID] = decisionCh
					a.mu.Unlock()

					// 发送权限请求事件到 Control Channel
					a.eventBus.EmitControl(&types.ControlPermissionRequiredEvent{
						Call: types.ToolCallSnapshot{
							ID:        tu.ID,
							Name:      tu.Name,
							Arguments: tu.Input,
						},
					})

					// 等待用户决策
					select {
					case decision := <-decisionCh:
						// 清理 pending map
						a.mu.Lock()
						delete(a.pendingPermissions, tu.ID)
						a.mu.Unlock()

						if decision != "approved" {
							// 用户拒绝
							errorMsg := fmt.Sprintf("Permission rejected by user for tool: %s", tu.Name)
							return &types.ToolResultBlock{
								ToolUseID: tu.ID,
								Content:   fmt.Sprintf(`{"ok":false,"error":"%s"}`, errorMsg),
								IsError:   true,
							}
						}
						// 用户批准，继续执行工具（跳出权限检查）
					case <-ctx.Done():
						// 上下文取消
						a.mu.Lock()
						delete(a.pendingPermissions, tu.ID)
						a.mu.Unlock()
						errorMsg := "Permission request cancelled"
						return &types.ToolResultBlock{
							ToolUseID: tu.ID,
							Content:   fmt.Sprintf(`{"ok":false,"error":"%s"}`, errorMsg),
							IsError:   true,
						}
					}
				} else {
					// 直接拒绝（NeedsApproval 为 false）
					errorMsg := fmt.Sprintf("Permission denied: %s (decided by: %s)", checkResult.Message, checkResult.DecidedBy)
					a.eventBus.EmitProgress(&types.ProgressToolErrorEvent{
						Call: types.ToolCallSnapshot{
							ID:        tu.ID,
							Name:      tu.Name,
							State:     types.ToolCallStateFailed,
							Arguments: tu.Input,
						},
						Error: errorMsg,
					})
					return &types.ToolResultBlock{
						ToolUseID: tu.ID,
						Content:   fmt.Sprintf(`{"ok":false,"error":"%s"}`, errorMsg),
						IsError:   true,
					}
				}
			}
		}
	}

	// 创建工具调用记录
	record := tools.NewToolCallRecord(tu.ID, tu.Name, tu.Input).Build()
	a.mu.Lock()
	a.toolRecords[tu.ID] = record
	a.mu.Unlock()

	// 获取工具
	tool, ok := a.toolMap[tu.Name]
	if !ok {
		// 工具未找到
		errorMsg := fmt.Sprintf("tool not found: %s", tu.Name)
		a.updateToolRecord(tu.ID, types.ToolCallStateFailed, errorMsg)
		a.eventBus.EmitProgress(&types.ProgressToolErrorEvent{
			Call: types.ToolCallSnapshot{
				ID:        tu.ID,
				Name:      tu.Name,
				State:     types.ToolCallStateFailed,
				Arguments: tu.Input,
			},
			Error: errorMsg,
		})
		return &types.ToolResultBlock{
			ToolUseID: tu.ID,
			IsError:   true,
		}
	}

	startTime := time.Now()
	record.StartTime = startTime
	record.Progress = 0

	interruptible, isInterruptible := tool.(tools.Interruptible)
	lrTool, isLongRunning := tool.(tools.LongRunningTool)
	pausable := isInterruptible
	cancelable := isInterruptible

	// 发送工具开始事件
	a.eventBus.EmitProgress(&types.ProgressToolStartEvent{
		Call: types.ToolCallSnapshot{
			ID:         record.ID,
			Name:       record.Name,
			State:      record.State,
			Arguments:  record.Input,
			Progress:   0,
			StartedAt:  record.StartTime,
			Cancelable: cancelable,
			Pausable:   pausable,
		},
	})

	// 设置断点
	a.setBreakpoint(types.BreakpointPreTool)

	// 执行工具
	a.updateToolRecord(tu.ID, types.ToolCallStateExecuting, "")
	a.setBreakpoint(types.BreakpointToolExecuting)

	// 构建工具执行上下文，包含必要的服务注入
	toolCtx := a.buildToolContext(ctx)
	toolCtx.Reporter = a.makeToolReporter(tu.ID, tu.Name)

	// 兼容旧版 Emit 回调
	toolCtx.Emit = func(eventType string, data any) {
		switch eventType {
		case "progress":
			if p, ok := data.(float64); ok {
				a.handleToolProgress(tu.ID, tu.Name, p, "", 0, 0, nil, 0)
			}
		case "intermediate":
			a.handleToolIntermediate(tu.ID, tu.Name, "", data)
		}
	}

	if isInterruptible {
		a.registerRunningTool(tu.ID, interruptible)
		defer a.unregisterRunningTool(tu.ID)
	}

	// 通过 Middleware Stack 执行工具 (Phase 6C)
	var execResult *tools.ExecuteResult
	if isLongRunning {
		// 长时任务走异步执行 + 轮询状态
		taskID, err := lrTool.StartAsync(ctx, tu.Input)
		if err != nil {
			execResult = &tools.ExecuteResult{Success: false, Error: err}
		} else {
			// 用长时任务的 Cancel 实现可中断
			if !isInterruptible {
				interruptible = &longRunningInterruptible{
					tool:   lrTool,
					taskID: taskID,
				}
				isInterruptible = true
				// pausable and cancelable assignments removed (ineffectual)
			}
			if isInterruptible {
				a.registerRunningTool(tu.ID, interruptible)
				defer a.unregisterRunningTool(tu.ID)
			}

			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			for {
				status, err := lrTool.GetStatus(ctx, taskID)
				if err != nil {
					execResult = &tools.ExecuteResult{Success: false, Error: err}
					break
				}

				// 推送进度事件
				a.handleToolProgress(tu.ID, tu.Name, status.Progress, "", 0, 0, status.Metadata, 0)

				// 终态处理
				if status.State.IsTerminal() {
					if status.State == tools.TaskStateCompleted {
						execResult = &tools.ExecuteResult{
							Success:    true,
							Output:     status.Result,
							Error:      nil,
							StartedAt:  status.StartTime,
							EndedAt:    *status.EndTime,
							DurationMs: status.EndTime.Sub(status.StartTime).Milliseconds(),
						}
					} else {
						var taskErr error
						if status.Error != nil {
							taskErr = status.Error
						} else {
							taskErr = fmt.Errorf("task failed")
						}
						execResult = &tools.ExecuteResult{
							Success:    false,
							Output:     status.Result,
							Error:      taskErr,
							StartedAt:  status.StartTime,
							EndedAt:    *status.EndTime,
							DurationMs: status.EndTime.Sub(status.StartTime).Milliseconds(),
						}
						if status.State == tools.TaskStateCancelled {
							a.eventBus.EmitProgress(&types.ProgressToolCancelledEvent{
								Call:   a.snapshotToolCall(tu.ID),
								Reason: "cancelled",
							})
						}
					}
					// 更新记录时间
					a.mu.Lock()
					if rec, ok := a.toolRecords[tu.ID]; ok {
						rec.StartTime = status.StartTime
						if status.EndTime != nil {
							rec.CompletedAt = status.EndTime
							rec.DurationMs = ptrInt64(status.EndTime.Sub(status.StartTime).Milliseconds())
						}
						rec.Progress = status.Progress
					}
					a.mu.Unlock()
					break
				}

				select {
				case <-ctx.Done():
					execResult = &tools.ExecuteResult{Success: false, Error: ctx.Err()}
					_ = lrTool.Cancel(context.Background(), taskID)
					a.eventBus.EmitProgress(&types.ProgressToolCancelledEvent{
						Call:   a.snapshotToolCall(tu.ID),
						Reason: "cancelled",
					})
					goto longRunningDone
				case <-ticker.C:
				}
			}

		longRunningDone:
			// execResult is always set in the loop above (error, terminal state, or ctx.Done)
		}
	} else if a.middlewareStack != nil {
		// 使用 middleware stack
		req := &middleware.ToolCallRequest{
			ToolCallID: tu.ID,
			ToolName:   tu.Name,
			ToolInput:  tu.Input,
			Tool:       tool,
			Context:    toolCtx,
			Metadata:   make(map[string]any),
		}

		// 定义 finalHandler: 实际执行工具
		finalHandler := func(ctx context.Context, req *middleware.ToolCallRequest) (*middleware.ToolCallResponse, error) {
			result := a.executor.Execute(ctx, &tools.ExecuteRequest{
				Tool:    req.Tool,
				Input:   req.ToolInput,
				Context: req.Context,
				Timeout: 60 * time.Second,
			})

			return &middleware.ToolCallResponse{
				Result:   result,
				Metadata: make(map[string]any),
			}, nil
		}

		// 通过 middleware stack 执行
		resp, err := a.middlewareStack.ExecuteToolCall(ctx, req, finalHandler)
		if err != nil {
			// 如果 middleware 返回错误,创建失败结果
			execResult = &tools.ExecuteResult{
				Success: false,
				Error:   err,
			}
		} else {
			execResult = resp.Result.(*tools.ExecuteResult)
		}
	} else {
		// 没有 middleware, 直接执行
		execResult = a.executor.Execute(ctx, &tools.ExecuteRequest{
			Tool:    tool,
			Input:   tu.Input,
			Context: toolCtx,
			Timeout: 60 * time.Second,
		})
	}

	endTime := time.Now()

	// 更新记录
	if execResult.Success {
		a.updateToolRecord(tu.ID, types.ToolCallStateCompleted, "")
		a.mu.Lock()
		a.toolRecords[tu.ID].Result = execResult.Output
		if execResult.StartedAt.IsZero() {
			a.toolRecords[tu.ID].StartedAt = &startTime
		} else {
			a.toolRecords[tu.ID].StartedAt = &execResult.StartedAt
		}
		if !execResult.EndedAt.IsZero() {
			a.toolRecords[tu.ID].CompletedAt = &execResult.EndedAt
		} else {
			a.toolRecords[tu.ID].CompletedAt = &endTime
		}
		durationMs := execResult.DurationMs
		if durationMs == 0 && !execResult.EndedAt.IsZero() {
			durationMs = execResult.EndedAt.Sub(execResult.StartedAt).Milliseconds()
		}
		a.toolRecords[tu.ID].DurationMs = &durationMs
		a.toolRecords[tu.ID].Progress = 1
		a.mu.Unlock()
	} else {
		errorMsg := ""
		if execResult.Error != nil {
			errorMsg = execResult.Error.Error()
		}
		a.updateToolRecord(tu.ID, types.ToolCallStateFailed, errorMsg)
	}

	// 发送工具结束事件
	a.mu.RLock()
	finalRecord := a.toolRecords[tu.ID]
	a.mu.RUnlock()

	a.eventBus.EmitProgress(&types.ProgressToolEndEvent{
		Call: types.ToolCallSnapshot{
			ID:         tu.ID,
			Name:       tu.Name,
			State:      finalRecord.State,
			Arguments:  finalRecord.Input,
			Result:     finalRecord.Result,
			Error:      finalRecord.Error,
			Progress:   finalRecord.Progress,
			StartedAt:  finalRecord.StartTime,
			UpdatedAt:  finalRecord.UpdatedAt,
			Cancelable: interruptible != nil,
			Pausable:   interruptible != nil,
		},
	})

	// 设置断点
	a.setBreakpoint(types.BreakpointPostTool)

	// 构建工具结果（压缩统一由 ToolResultOptimizerMiddleware 处理）
	if execResult.Success {
		return &types.ToolResultBlock{
			ToolUseID: tu.ID,
			Content:   fmt.Sprintf("%v", execResult.Output),
			IsError:   false,
		}
	} else {
		errorMsg := ""
		if execResult.Error != nil {
			errorMsg = execResult.Error.Error()
		}
		return &types.ToolResultBlock{
			ToolUseID: tu.ID,
			Content:   fmt.Sprintf(`{"ok":false,"error":"%s"}`, errorMsg),
			IsError:   true,
		}
	}
}

// setBreakpoint 设置断点
func (a *Agent) setBreakpoint(state types.BreakpointState) {
	a.mu.Lock()
	previous := a.breakpoint
	a.breakpoint = state
	a.mu.Unlock()

	a.eventBus.EmitMonitor(&types.MonitorBreakpointChangedEvent{
		Previous:  previous,
		Current:   state,
		Timestamp: time.Now(),
	})
}

// updateToolRecord 更新工具记录
func (a *Agent) updateToolRecord(id string, state types.ToolCallState, errorMsg string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	record, ok := a.toolRecords[id]
	if !ok {
		return
	}

	now := time.Now()
	record.State = state
	record.UpdatedAt = now

	if errorMsg != "" {
		record.Error = errorMsg
		record.IsError = true
	}

	record.AuditTrail = append(record.AuditTrail, types.ToolCallAuditEntry{
		State:     state,
		Timestamp: now,
	})
}

// handleStreamResponse 处理流式响应(Phase 6C - 提取为独立方法以支持Middleware)
func (a *Agent) handleStreamResponse(ctx context.Context, stream <-chan provider.StreamChunk) (types.Message, error) {
	assistantContent := make([]types.ContentBlock, 0)
	currentBlockIndex := -1
	textBuffers := make(map[int]string)
	inputJSONBuffers := make(map[int]string)
	reasoningStarted := false // 追踪是否已发送思考开始事件
	reasoningBuffer := ""     // 累积思考内容

	for chunk := range stream {
		switch chunk.Type {
		// 处理 reasoning_delta (DeepSeek Reasoner 模型的思考过程)
		case "reasoning_delta":
			if delta, ok := chunk.Delta.(map[string]any); ok {
				if content, ok := delta["content"].(string); ok && content != "" {
					// 首次收到思考内容时，发送开始事件
					if !reasoningStarted {
						reasoningStarted = true
						a.eventBus.EmitProgress(&types.ProgressThinkChunkStartEvent{
							Step: a.stepCount,
						})
						procLog.Debug(ctx, "reasoning started", map[string]any{"step": a.stepCount})
					}
					// 累积并发送思考内容增量
					reasoningBuffer += content
					a.eventBus.EmitProgress(&types.ProgressThinkChunkEvent{
						Step:  a.stepCount,
						Stage: types.ThinkingStageReasoning,
						Delta: content,
					})
				}
			}

		case "content_block_start":
			currentBlockIndex = chunk.Index
			if delta, ok := chunk.Delta.(map[string]any); ok {
				blockType, _ := delta["type"].(string)
				switch blockType {
				case "text":
					// 发送文本开始事件
					a.eventBus.EmitProgress(&types.ProgressTextChunkStartEvent{
						Step: a.stepCount,
					})
					// 初始化文本块
					for len(assistantContent) <= currentBlockIndex {
						assistantContent = append(assistantContent, nil)
					}
					assistantContent[currentBlockIndex] = &types.TextBlock{Text: ""}
					textBuffers[currentBlockIndex] = ""
				case "tool_use":
					procLog.Debug(ctx, "received tool_use block", map[string]any{"id": delta["id"], "name": delta["name"]})
					// 初始化工具调用块
					for len(assistantContent) <= currentBlockIndex {
						assistantContent = append(assistantContent, nil)
					}

					// 处理不同的工具调用格式（Anthropic vs OpenAI兼容格式）
					toolID := ""
					toolName := ""
					if id, ok := delta["id"].(string); ok {
						toolID = id
					} else if id, ok := delta["id"].(float64); ok {
						toolID = fmt.Sprintf("%.0f", id)
					}

					if name, ok := delta["name"].(string); ok {
						toolName = name
					}

					assistantContent[currentBlockIndex] = &types.ToolUseBlock{
						ID:    toolID,
						Name:  toolName,
						Input: make(map[string]any),
					}
				default:
					procLog.Debug(ctx, "unknown block type", map[string]any{"type": blockType})
				}
			}

		case "content_block_delta":
			if delta, ok := chunk.Delta.(map[string]any); ok {
				deltaType, _ := delta["type"].(string)
				switch deltaType {
				case "text_delta":
					text, _ := delta["text"].(string)
					// 如果 currentBlockIndex 未初始化，使用 chunk.Index 或默认为 0
					if currentBlockIndex < 0 {
						currentBlockIndex = max(chunk.Index, 0)
					}

					// 确保有足够的空间
					for len(assistantContent) <= currentBlockIndex {
						assistantContent = append(assistantContent, nil)
					}

					// 如果块还未初始化，创建新的文本块
					if assistantContent[currentBlockIndex] == nil {
						assistantContent[currentBlockIndex] = &types.TextBlock{Text: ""}
						textBuffers[currentBlockIndex] = ""
					}

					// 累积文本
					if _, exists := textBuffers[currentBlockIndex]; !exists {
						textBuffers[currentBlockIndex] = ""
					}
					textBuffers[currentBlockIndex] += text
					if block, ok := assistantContent[currentBlockIndex].(*types.TextBlock); ok {
						block.Text = textBuffers[currentBlockIndex]
					}

					// 发送文本增量事件
					a.eventBus.EmitProgress(&types.ProgressTextChunkEvent{
						Step:  a.stepCount,
						Delta: text,
					})
				case "input_json_delta":
					partialJSON, _ := delta["partial_json"].(string)
					if currentBlockIndex >= 0 {
						if _, exists := inputJSONBuffers[currentBlockIndex]; !exists {
							inputJSONBuffers[currentBlockIndex] = ""
						}
						inputJSONBuffers[currentBlockIndex] += partialJSON
					}
				case "arguments":
					partialArgs, _ := delta["arguments"].(string)
					blockIndex := chunk.Index
					if blockIndex < 0 {
						blockIndex = currentBlockIndex
					}
					if blockIndex >= 0 {
						if _, exists := inputJSONBuffers[blockIndex]; !exists {
							inputJSONBuffers[blockIndex] = ""
						}
						inputJSONBuffers[blockIndex] += partialArgs
					}
				}
			}

		case "content_block_stop":
			if currentBlockIndex >= 0 && currentBlockIndex < len(assistantContent) {
				if block, ok := assistantContent[currentBlockIndex].(*types.TextBlock); ok {
					a.eventBus.EmitProgress(&types.ProgressTextChunkEndEvent{
						Step: a.stepCount,
						Text: block.Text,
					})
				} else if block, ok := assistantContent[currentBlockIndex].(*types.ToolUseBlock); ok {
					if jsonStr, exists := inputJSONBuffers[currentBlockIndex]; exists && jsonStr != "" {
						var input map[string]any
						if err := json.Unmarshal([]byte(jsonStr), &input); err == nil {
							block.Input = input
						} else {
							procLog.Warn(ctx, "failed to parse tool input JSON", map[string]any{"error": err})
						}
					}
				}
			}

		case "message_delta":
			if chunk.Usage != nil {
				a.eventBus.EmitMonitor(&types.MonitorTokenUsageEvent{
					InputTokens:  chunk.Usage.InputTokens,
					OutputTokens: chunk.Usage.OutputTokens,
					TotalTokens:  chunk.Usage.InputTokens + chunk.Usage.OutputTokens,
				})
			}

		// OpenAI 兼容格式：处理 text 类型（来自 OpenRouter、DeepSeek 等）
		case "text":
			text := chunk.TextDelta
			if text == "" {
				text = chunk.Delta.(string)
			}
			if text != "" {
				// 确保有文本块
				if currentBlockIndex < 0 {
					currentBlockIndex = 0
				}
				for len(assistantContent) <= currentBlockIndex {
					assistantContent = append(assistantContent, nil)
				}
				if assistantContent[currentBlockIndex] == nil {
					assistantContent[currentBlockIndex] = &types.TextBlock{Text: ""}
					textBuffers[currentBlockIndex] = ""
					// 发送文本开始事件
					a.eventBus.EmitProgress(&types.ProgressTextChunkStartEvent{
						Step: a.stepCount,
					})
				}

				// 累积文本
				textBuffers[currentBlockIndex] += text
				if block, ok := assistantContent[currentBlockIndex].(*types.TextBlock); ok {
					block.Text = textBuffers[currentBlockIndex]
				}

				// 发送文本增量事件
				a.eventBus.EmitProgress(&types.ProgressTextChunkEvent{
					Step:  a.stepCount,
					Delta: text,
				})
			}

		// OpenAI 兼容格式：处理 tool_call 类型
		case "tool_call":
			if chunk.ToolCall != nil {
				tc := chunk.ToolCall
				blockIndex := tc.Index
				if blockIndex < 0 {
					blockIndex = len(assistantContent)
				}

				// 确保有足够空间
				for len(assistantContent) <= blockIndex {
					assistantContent = append(assistantContent, nil)
				}

				// 初始化或更新工具调用块
				if assistantContent[blockIndex] == nil {
					assistantContent[blockIndex] = &types.ToolUseBlock{
						ID:    tc.ID,
						Name:  tc.Name,
						Input: make(map[string]any),
					}
					inputJSONBuffers[blockIndex] = ""
				}

				// 累积参数
				if tc.ArgumentsDelta != "" {
					inputJSONBuffers[blockIndex] += tc.ArgumentsDelta
				}

				currentBlockIndex = blockIndex
			}

		// OpenAI 兼容格式：处理 done 类型
		case "done":
			// 发送文本结束事件（如果有文本）
			if currentBlockIndex >= 0 && currentBlockIndex < len(assistantContent) {
				if _, ok := assistantContent[currentBlockIndex].(*types.TextBlock); ok {
					a.eventBus.EmitProgress(&types.ProgressTextChunkEndEvent{
						Step: a.stepCount,
					})
				}
			}

		// OpenAI 兼容格式：处理 usage 类型
		case "usage":
			if chunk.Usage != nil {
				a.eventBus.EmitMonitor(&types.MonitorTokenUsageEvent{
					InputTokens:  chunk.Usage.InputTokens,
					OutputTokens: chunk.Usage.OutputTokens,
					TotalTokens:  chunk.Usage.InputTokens + chunk.Usage.OutputTokens,
				})
			}
		}
	}

	// 流式响应结束后，解析所有累积的工具输入
	if len(inputJSONBuffers) > 0 {
		procLog.Debug(ctx, "processing inputJSONBuffers", map[string]any{"buffer_count": len(inputJSONBuffers), "content_blocks": len(assistantContent)})
		for i, block := range assistantContent {
			if tu, ok := block.(*types.ToolUseBlock); ok {
				if jsonStr, exists := inputJSONBuffers[i]; exists && jsonStr != "" {
					procLog.Debug(ctx, "parsing tool input", map[string]any{"block": i, "tool": tu.Name, "json_length": len(jsonStr)})
					var input map[string]any
					if err := json.Unmarshal([]byte(jsonStr), &input); err == nil {
						tu.Input = input
						procLog.Debug(ctx, "successfully parsed tool input", map[string]any{"tool": tu.Name, "fields": len(input)})
					} else {
						procLog.Warn(ctx, "failed to parse tool input JSON", map[string]any{"tool": tu.Name, "error": err, "raw": jsonStr})
					}
				} else {
					// 关键修复: 记录空输入缓冲区警告
					procLog.Warn(ctx, "empty input buffer for tool block", map[string]any{"block": i, "tool": tu.Name, "exists": exists, "json_str": jsonStr})
				}
			}
		}
	}

	// 如果有思考过程，发送结束事件
	if reasoningStarted {
		a.eventBus.EmitProgress(&types.ProgressThinkChunkEndEvent{
			Step: a.stepCount,
		})
		procLog.Debug(ctx, "reasoning ended", map[string]any{"step": a.stepCount, "total_length": len(reasoningBuffer)})
	}

	return types.Message{
		Role:          types.MessageRoleAssistant,
		ContentBlocks: assistantContent,
	}, nil
}

// runNonStreamingStep 非流式执行模型步骤（快速模式）
func (a *Agent) runNonStreamingStep(ctx context.Context) error {
	// 准备工具Schema（包含使用示例）
	toolSchemas := make([]provider.ToolSchema, 0, len(a.toolMap))
	for _, tool := range a.toolMap {
		schema := provider.ToolSchema{
			Name:        tool.Name(),
			Description: tool.Description(),
			InputSchema: tool.InputSchema(),
		}
		// 检查工具是否实现了 ExampleableTool 接口
		if exampleable, ok := tool.(tools.ExampleableTool); ok {
			examples := exampleable.Examples()
			if len(examples) > 0 {
				providerExamples := make([]provider.ToolExample, len(examples))
				for i, ex := range examples {
					providerExamples[i] = provider.ToolExample{
						Description: ex.Description,
						Input:       ex.Input,
						Output:      ex.Output,
					}
				}
				schema.InputExamples = providerExamples
			}
		}
		toolSchemas = append(toolSchemas, schema)
	}

	// 准备消息
	a.mu.RLock()
	messages := make([]types.Message, len(a.messages))
	copy(messages, a.messages)
	currentSystemPrompt := a.template.SystemPrompt
	a.mu.RUnlock()

	// 创建Provider选项
	streamOpts := &provider.StreamOptions{
		Tools:       toolSchemas,
		System:      currentSystemPrompt,
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	procLog.Debug(ctx, "calling Complete API", map[string]any{"messages": len(messages), "tools": len(toolSchemas)})

	// 调用Complete API（非流式）
	response, err := a.provider.Complete(ctx, messages, streamOpts)
	if err != nil {
		return fmt.Errorf("complete call failed: %w", err)
	}

	// 添加响应消息
	a.mu.Lock()
	a.messages = append(a.messages, response.Message)
	a.mu.Unlock()

	// 提取工具调用从ContentBlocks
	toolUses := make([]*types.ToolUseBlock, 0)
	for _, block := range response.Message.ContentBlocks {
		if tu, ok := block.(*types.ToolUseBlock); ok {
			toolUses = append(toolUses, tu)
		}
	}

	procLog.Debug(ctx, "received response", map[string]any{"tool_calls": len(toolUses)})

	// 处理工具调用
	if len(toolUses) > 0 {
		// 执行工具
		if err := a.executeTools(ctx, toolUses); err != nil {
			return fmt.Errorf("execute tools failed: %w", err)
		}

		// 检查迭代限制（防止无限循环）
		a.mu.Lock()
		a.iterationCount++
		currentIter := a.iterationCount
		maxIter := a.maxIterations
		if maxIter <= 0 {
			maxIter = 50
		}
		a.mu.Unlock()

		if currentIter > maxIter {
			procLog.Error(ctx, "iteration limit exceeded in non-streaming mode", map[string]any{
				"agent_id": a.id, "iteration": currentIter, "max": maxIter,
			})
			return fmt.Errorf("iteration limit exceeded: %d > %d", currentIter, maxIter)
		}

		// 递归调用继续处理
		return a.runNonStreamingStep(ctx)
	}

	// 没有工具调用，完成
	a.mu.Lock()
	a.state = types.AgentStateReady
	a.mu.Unlock()

	procLog.Debug(ctx, "execution completed", nil)

	return nil
}

// shouldTrimMessages 检查是否应该修剪消息
func (a *Agent) shouldTrimMessages() bool {
	return a.config.Store != nil &&
		a.config.Store.MaxMessages > 0 &&
		a.config.Store.AutoTrim
}

// trimMessagesInMemory 在内存中修剪消息（FIFO策略）
// 必须在持有 a.mu 锁的情况下调用
func (a *Agent) trimMessagesInMemory(messages []types.Message, maxMessages int) []types.Message {
	if len(messages) <= maxMessages {
		return messages
	}
	// 保留最近的 maxMessages 条消息
	return messages[len(messages)-maxMessages:]
}

