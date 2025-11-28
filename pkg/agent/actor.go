package agent

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/actor"
	"github.com/astercloud/aster/pkg/types"
)

// ============== Agent 消息类型定义 ==============

// SendMsg 发送用户消息请求
type SendMsg struct {
	Text string
	Ctx  context.Context
}

func (m *SendMsg) Kind() string { return "agent.send" }

// ChatMsg 同步对话请求
type ChatMsg struct {
	Text    string
	Ctx     context.Context
	ReplyTo chan *ChatResultMsg // 用于接收响应
}

func (m *ChatMsg) Kind() string { return "agent.chat" }

// StreamMsg 流式对话请求
type StreamMsg struct {
	Text    string
	Ctx     context.Context
	EventCh chan *StreamEventMsg // 事件流通道
}

func (m *StreamMsg) Kind() string { return "agent.stream" }

// StreamEventMsg 流式事件消息
type StreamEventMsg struct {
	Type    string      // "text", "tool_start", "tool_end", "done", "error"
	Content interface{}
	Error   error
}

func (m *StreamEventMsg) Kind() string { return "agent.stream_event" }

// ToolCallMsg 工具调用请求
type ToolCallMsg struct {
	ToolUse *types.ToolUseBlock
	Ctx     context.Context
}

func (m *ToolCallMsg) Kind() string { return "agent.tool_call" }

// ToolResultMsg 工具调用结果
type ToolResultMsg struct {
	CallID  string
	Result  interface{}
	IsError bool
	Error   string
}

func (m *ToolResultMsg) Kind() string { return "agent.tool_result" }

// DirectToolCallMsg 直接工具调用请求
type DirectToolCallMsg struct {
	ToolName string
	Input    map[string]interface{}
	Ctx      context.Context
	ReplyTo  chan *DirectToolResultMsg
}

func (m *DirectToolCallMsg) Kind() string { return "agent.direct_tool_call" }

// DirectToolResultMsg 直接工具调用结果
type DirectToolResultMsg struct {
	ToolName string
	Result   interface{}
	Error    error
}

func (m *DirectToolResultMsg) Kind() string { return "agent.direct_tool_result" }

// BatchToolCallMsg 批量工具调用请求
type BatchToolCallMsg struct {
	Calls   []ToolCall
	Ctx     context.Context
	ReplyTo chan *BatchToolResultMsg
}

func (m *BatchToolCallMsg) Kind() string { return "agent.batch_tool_call" }

// BatchToolResultMsg 批量工具调用结果
type BatchToolResultMsg struct {
	Results []ToolCallResult
}

func (m *BatchToolResultMsg) Kind() string { return "agent.batch_tool_result" }

// GetStatusMsg 获取状态请求
type GetStatusMsg struct {
	ReplyTo chan *types.AgentStatus
}

func (m *GetStatusMsg) Kind() string { return "agent.get_status" }

// StopMsg 停止 Agent 请求
type StopMsg struct{}

func (m *StopMsg) Kind() string { return "agent.stop" }

// ChatResultMsg 对话结果消息
type ChatResultMsg struct {
	Result *types.CompleteResult
	Error  error
}

func (m *ChatResultMsg) Kind() string { return "agent.chat_result" }

// ErrorMsg 错误消息
type ErrorMsg struct {
	Error   error
	Context string
}

func (m *ErrorMsg) Kind() string { return "agent.error" }

// ============== AgentActor 适配器 ==============

// AgentActor 将 Agent 包装为 Actor
// 这是适配器模式的实现，不修改原有 Agent 代码
type AgentActor struct {
	agent *Agent

	// 状态
	mu        sync.RWMutex
	isRunning bool
	lastError error

	// 性能统计
	stats *AgentActorStats
}

// AgentActorStats Actor 统计信息
type AgentActorStats struct {
	MessagesReceived int64
	MessagesHandled  int64
	Errors           int64
	AverageLatency   time.Duration
	LastMessageAt    time.Time
}

// NewAgentActor 创建 Agent Actor 适配器
func NewAgentActor(agent *Agent) *AgentActor {
	return &AgentActor{
		agent:     agent,
		isRunning: true,
		stats:     &AgentActorStats{},
	}
}

// Receive 处理接收到的消息（Actor 核心方法）
func (a *AgentActor) Receive(ctx *actor.Context, msg actor.Message) {
	a.mu.Lock()
	a.stats.MessagesReceived++
	a.stats.LastMessageAt = time.Now()
	a.mu.Unlock()

	startTime := time.Now()

	defer func() {
		// 更新统计
		a.mu.Lock()
		a.stats.MessagesHandled++
		latency := time.Since(startTime)
		// 简单的移动平均
		a.stats.AverageLatency = (a.stats.AverageLatency + latency) / 2
		a.mu.Unlock()
	}()

	// 处理系统消息
	switch m := msg.(type) {
	case *actor.Started:
		log.Printf("[AgentActor] %s started", a.agent.ID())
		return

	case *actor.Stopping:
		log.Printf("[AgentActor] %s stopping", a.agent.ID())
		a.handleStop()
		return

	case *actor.Stopped:
		log.Printf("[AgentActor] %s stopped", a.agent.ID())
		return

	case *actor.Restarting:
		log.Printf("[AgentActor] %s restarting", a.agent.ID())
		return

	// 处理 Agent 业务消息
	case *SendMsg:
		a.handleSend(ctx, m)

	case *ChatMsg:
		a.handleChat(ctx, m)

	case *StreamMsg:
		a.handleStream(ctx, m)

	case *ToolCallMsg:
		a.handleToolCall(ctx, m)

	case *DirectToolCallMsg:
		a.handleDirectToolCall(ctx, m)

	case *BatchToolCallMsg:
		a.handleBatchToolCall(ctx, m)

	case *GetStatusMsg:
		a.handleGetStatus(ctx, m)

	case *StopMsg:
		a.handleStop()
		ctx.StopSelf()

	default:
		log.Printf("[AgentActor] %s received unknown message: %T", a.agent.ID(), msg)
	}
}

// handleSend 处理发送消息（异步）
func (a *AgentActor) handleSend(ctx *actor.Context, msg *SendMsg) {
	msgCtx := msg.Ctx
	if msgCtx == nil {
		msgCtx = context.Background()
	}

	go func() {
		err := a.agent.Send(msgCtx, msg.Text)
		if err != nil {
			a.recordError(err)
			if ctx.Sender != nil {
				ctx.Reply(&ErrorMsg{Error: err, Context: "send"})
			}
		}
	}()
}

// handleChat 处理同步对话
func (a *AgentActor) handleChat(ctx *actor.Context, msg *ChatMsg) {
	msgCtx := msg.Ctx
	if msgCtx == nil {
		msgCtx = context.Background()
	}

	go func() {
		result, err := a.agent.Chat(msgCtx, msg.Text)

		response := &ChatResultMsg{
			Result: result,
			Error:  err,
		}

		if err != nil {
			a.recordError(err)
		}

		// 通过 channel 返回结果
		if msg.ReplyTo != nil {
			select {
			case msg.ReplyTo <- response:
			default:
				log.Printf("[AgentActor] %s chat reply channel full or closed", a.agent.ID())
			}
		}

		// 也通过 Actor 消息回复
		if ctx.Sender != nil {
			ctx.Reply(response)
		}
	}()
}

// handleStream 处理流式对话
func (a *AgentActor) handleStream(ctx *actor.Context, msg *StreamMsg) {
	msgCtx := msg.Ctx
	if msgCtx == nil {
		msgCtx = context.Background()
	}

	go func() {
		reader := a.agent.Stream(msgCtx, msg.Text)

		for {
			event, err := reader.Recv()
			if err != nil {
				// EOF 或错误
				if msg.EventCh != nil {
					msg.EventCh <- &StreamEventMsg{
						Type:  "done",
						Error: err,
					}
					close(msg.EventCh)
				}
				return
			}

			if event != nil && msg.EventCh != nil {
				msg.EventCh <- &StreamEventMsg{
					Type:    "event",
					Content: event,
				}
			}
		}
	}()
}

// handleToolCall 处理工具调用
func (a *AgentActor) handleToolCall(ctx *actor.Context, msg *ToolCallMsg) {
	msgCtx := msg.Ctx
	if msgCtx == nil {
		msgCtx = context.Background()
	}

	go func() {
		result := a.agent.executeSingleTool(msgCtx, msg.ToolUse)

		var response *ToolResultMsg

		if toolResult, ok := result.(*types.ToolResultBlock); ok {
			response = &ToolResultMsg{
				CallID:  msg.ToolUse.ID,
				Result:  toolResult.Content,
				IsError: toolResult.IsError,
			}
		} else {
			response = &ToolResultMsg{
				CallID: msg.ToolUse.ID,
				Result: result,
			}
		}

		if ctx.Sender != nil {
			ctx.Reply(response)
		}
	}()
}

// handleDirectToolCall 处理直接工具调用
func (a *AgentActor) handleDirectToolCall(ctx *actor.Context, msg *DirectToolCallMsg) {
	msgCtx := msg.Ctx
	if msgCtx == nil {
		msgCtx = context.Background()
	}

	go func() {
		result, err := a.agent.ExecuteToolDirect(msgCtx, msg.ToolName, msg.Input)

		response := &DirectToolResultMsg{
			ToolName: msg.ToolName,
			Result:   result,
			Error:    err,
		}

		if err != nil {
			a.recordError(err)
		}

		if msg.ReplyTo != nil {
			select {
			case msg.ReplyTo <- response:
			default:
			}
		}

		if ctx.Sender != nil {
			ctx.Reply(response)
		}
	}()
}

// handleBatchToolCall 处理批量工具调用
func (a *AgentActor) handleBatchToolCall(ctx *actor.Context, msg *BatchToolCallMsg) {
	msgCtx := msg.Ctx
	if msgCtx == nil {
		msgCtx = context.Background()
	}

	go func() {
		results := a.agent.ExecuteToolsDirect(msgCtx, msg.Calls)

		response := &BatchToolResultMsg{
			Results: results,
		}

		if msg.ReplyTo != nil {
			select {
			case msg.ReplyTo <- response:
			default:
			}
		}

		if ctx.Sender != nil {
			ctx.Reply(response)
		}
	}()
}

// handleGetStatus 处理获取状态
func (a *AgentActor) handleGetStatus(ctx *actor.Context, msg *GetStatusMsg) {
	status := a.agent.Status()

	if msg.ReplyTo != nil {
		select {
		case msg.ReplyTo <- status:
		default:
		}
	}
}

// handleStop 处理停止
func (a *AgentActor) handleStop() {
	a.mu.Lock()
	a.isRunning = false
	a.mu.Unlock()

	if err := a.agent.Close(); err != nil {
		log.Printf("[AgentActor] %s close error: %v", a.agent.ID(), err)
	}
}

// recordError 记录错误
func (a *AgentActor) recordError(err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.stats.Errors++
	a.lastError = err
}

// Stats 获取统计信息
func (a *AgentActor) Stats() *AgentActorStats {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return &AgentActorStats{
		MessagesReceived: a.stats.MessagesReceived,
		MessagesHandled:  a.stats.MessagesHandled,
		Errors:           a.stats.Errors,
		AverageLatency:   a.stats.AverageLatency,
		LastMessageAt:    a.stats.LastMessageAt,
	}
}

// Agent 获取底层 Agent（用于调试）
func (a *AgentActor) Agent() *Agent {
	return a.agent
}

// ============== Agent Actor 工厂 ==============

// AgentActorFactory Agent Actor 工厂
type AgentActorFactory struct {
	deps *Dependencies
}

// NewAgentActorFactory 创建 Agent Actor 工厂
func NewAgentActorFactory(deps *Dependencies) *AgentActorFactory {
	return &AgentActorFactory{deps: deps}
}

// Create 创建 Agent Actor
func (f *AgentActorFactory) Create(ctx context.Context, config *types.AgentConfig) (*AgentActor, error) {
	agent, err := Create(ctx, config, f.deps)
	if err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}

	return NewAgentActor(agent), nil
}

// CreateAndSpawn 创建并在 Actor 系统中启动 Agent
func (f *AgentActorFactory) CreateAndSpawn(
	ctx context.Context,
	system *actor.System,
	config *types.AgentConfig,
	name string,
) (*actor.PID, error) {
	agentActor, err := f.Create(ctx, config)
	if err != nil {
		return nil, err
	}

	// 设置监督策略（Agent 应该在失败时重启）
	props := &actor.Props{
		Name:               name,
		MailboxSize:        100,
		SupervisorStrategy: actor.NewOneForOneStrategy(3, time.Minute, actor.DefaultDecider),
	}

	pid := system.SpawnWithProps(agentActor, props)
	return pid, nil
}

// ============== 辅助函数 ==============

// AgentActorTell 向 Agent Actor 发送消息（不等待响应）
func AgentActorTell(pid *actor.PID, msg actor.Message) {
	pid.Tell(msg)
}

// AgentActorChat 向 Agent Actor 发送对话请求并等待响应
func AgentActorChat(pid *actor.PID, text string, timeout time.Duration) (*ChatResultMsg, error) {
	replyCh := make(chan *ChatResultMsg, 1)

	pid.Tell(&ChatMsg{
		Text:    text,
		Ctx:     context.Background(),
		ReplyTo: replyCh,
	})

	select {
	case result := <-replyCh:
		return result, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("chat request timed out after %v", timeout)
	}
}

// AgentActorCallTool 向 Agent Actor 发送工具调用请求并等待响应
func AgentActorCallTool(pid *actor.PID, toolName string, input map[string]interface{}, timeout time.Duration) (*DirectToolResultMsg, error) {
	replyCh := make(chan *DirectToolResultMsg, 1)

	pid.Tell(&DirectToolCallMsg{
		ToolName: toolName,
		Input:    input,
		Ctx:      context.Background(),
		ReplyTo:  replyCh,
	})

	select {
	case result := <-replyCh:
		return result, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("tool call timed out after %v", timeout)
	}
}

// AgentActorGetStatus 获取 Agent 状态
func AgentActorGetStatus(pid *actor.PID, timeout time.Duration) (*types.AgentStatus, error) {
	replyCh := make(chan *types.AgentStatus, 1)

	pid.Tell(&GetStatusMsg{
		ReplyTo: replyCh,
	})

	select {
	case status := <-replyCh:
		return status, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("get status timed out after %v", timeout)
	}
}
