package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/events"
	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/permission"
	"github.com/astercloud/aster/pkg/types"
)

var remoteAgentLog = logging.ForComponent("RemoteAgent")

// RemoteAgent 是远程 Agent 的本地代理
// 它实现了 Agent 接口的核心方法,但不执行实际的 LLM 调用
// 而是接收来自远程进程的事件并转发到本地 EventBus
type RemoteAgent struct {
	id         string
	templateID string
	metadata   map[string]any
	eventBus   *events.EventBus
	createdAt  time.Time

	mu    sync.RWMutex
	state types.AgentRuntimeState
}

// NewRemoteAgent 创建一个新的 RemoteAgent 实例
func NewRemoteAgent(id, templateID string, metadata map[string]any) *RemoteAgent {
	return &RemoteAgent{
		id:         id,
		templateID: templateID,
		metadata:   metadata,
		eventBus:   events.NewEventBus(),
		createdAt:  time.Now(),
		state:      types.StateIdle,
	}
}

// ID 返回 Agent ID
func (r *RemoteAgent) ID() string {
	return r.id
}

// GetEventBus 返回 EventBus 实例
func (r *RemoteAgent) GetEventBus() *events.EventBus {
	return r.eventBus
}

// Subscribe 订阅指定通道的事件
func (r *RemoteAgent) Subscribe(channels []types.AgentChannel, opts *types.SubscribeOptions) <-chan types.AgentEventEnvelope {
	return r.eventBus.Subscribe(channels, opts)
}

// Unsubscribe 取消订阅
func (r *RemoteAgent) Unsubscribe(ch <-chan types.AgentEventEnvelope) {
	r.eventBus.Unsubscribe(ch)
}

// Status 返回 Agent 状态
func (r *RemoteAgent) Status() *types.AgentStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return &types.AgentStatus{
		AgentID: r.id,
		State:   r.state,
	}
}

// PushEvent 接收远程事件并推送到本地 EventBus
// 这是 RemoteAgent 的核心方法,用于接收来自远程进程的事件
func (r *RemoteAgent) PushEvent(envelope types.AgentEventEnvelope) error {
	if envelope.Event == nil {
		return fmt.Errorf("event is nil")
	}

	// 根据事件类型推送到对应的通道
	if ev, ok := envelope.Event.(types.EventType); ok {
		channel := ev.Channel()

		switch channel {
		case types.ChannelProgress:
			r.eventBus.EmitProgress(envelope.Event)
		case types.ChannelControl:
			r.eventBus.EmitControl(envelope.Event)
		case types.ChannelMonitor:
			r.eventBus.EmitMonitor(envelope.Event)
		default:
			remoteAgentLog.Warn(context.Background(), "remote_agent.unknown_channel", map[string]any{
				"agent_id": r.id,
				"channel":  string(channel),
			})
		}

		// 更新状态
		r.updateStateFromEvent(envelope.Event)
	} else if eventMap, ok := envelope.Event.(map[string]any); ok {
		// 处理从 JSON 反序列化的事件（map[string]any 类型）
		// 从事件 map 中推断 channel
		channel := r.inferChannelFromEventMap(eventMap)

		switch types.AgentChannel(channel) {
		case types.ChannelProgress:
			r.eventBus.EmitProgress(envelope.Event)
		case types.ChannelControl:
			r.eventBus.EmitControl(envelope.Event)
		case types.ChannelMonitor:
			r.eventBus.EmitMonitor(envelope.Event)
		default:
			// 默认发送到 Progress 通道
			r.eventBus.EmitProgress(envelope.Event)
		}

		remoteAgentLog.Debug(context.Background(), "remote_agent.event_pushed", map[string]any{
			"agent_id": r.id,
			"channel":  channel,
			"event":    eventMap,
		})
	} else {
		remoteAgentLog.Warn(context.Background(), "remote_agent.unknown_event_type", map[string]any{
			"agent_id":   r.id,
			"event_type": fmt.Sprintf("%T", envelope.Event),
		})
	}

	return nil
}

// inferChannelFromEventMap 从事件 map 中推断通道类型
func (r *RemoteAgent) inferChannelFromEventMap(eventMap map[string]any) string {
	// 检查是否有 channel 字段
	if ch, ok := eventMap["channel"].(string); ok {
		return ch
	}

	// 根据事件字段推断通道
	// Progress 事件通常有: delta, text, call (tool events)
	if _, ok := eventMap["delta"]; ok {
		return string(types.ChannelProgress)
	}
	if _, ok := eventMap["call"]; ok {
		return string(types.ChannelProgress)
	}
	if _, ok := eventMap["text"]; ok {
		return string(types.ChannelProgress)
	}

	// Control 事件通常有: request_id, questions, approved
	if _, ok := eventMap["request_id"]; ok {
		return string(types.ChannelControl)
	}
	if _, ok := eventMap["questions"]; ok {
		return string(types.ChannelControl)
	}
	if _, ok := eventMap["approved"]; ok {
		return string(types.ChannelControl)
	}

	// Monitor 事件通常有: state, severity, message (error)
	if _, ok := eventMap["state"]; ok {
		return string(types.ChannelMonitor)
	}
	if _, ok := eventMap["severity"]; ok {
		return string(types.ChannelMonitor)
	}

	// 默认返回 progress
	return string(types.ChannelProgress)
}

// updateStateFromEvent 根据事件更新 Agent 状态
func (r *RemoteAgent) updateStateFromEvent(event any) {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch e := event.(type) {
	case *types.MonitorStateChangedEvent:
		r.state = e.State
	case *types.ProgressDoneEvent:
		r.state = types.StateIdle
	case *types.MonitorErrorEvent:
		if e.Severity == "fatal" {
			r.state = types.StateFailed
		}
	}
}

// Close 关闭 RemoteAgent
func (r *RemoteAgent) Close() error {
	r.eventBus.Close()
	return nil
}

// GetMetadata 返回元数据
func (r *RemoteAgent) GetMetadata() map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]any)
	for k, v := range r.metadata {
		result[k] = v
	}
	return result
}

// SetMetadata 设置元数据
func (r *RemoteAgent) SetMetadata(key string, value any) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.metadata == nil {
		r.metadata = make(map[string]any)
	}
	r.metadata[key] = value
}

// 以下方法是为了满足 Agent 接口的完整性,但在 RemoteAgent 中不支持

// Send 不支持 - RemoteAgent 不执行实际的 LLM 调用
func (r *RemoteAgent) Send(ctx context.Context, text string) error {
	return fmt.Errorf("RemoteAgent does not support Send operation")
}

// Chat 不支持 - RemoteAgent 不执行实际的 LLM 调用
func (r *RemoteAgent) Chat(ctx context.Context, text string) (*types.CompleteResult, error) {
	return nil, fmt.Errorf("RemoteAgent does not support Chat operation")
}

// ExecuteToolDirect 不支持 - RemoteAgent 不执行工具
func (r *RemoteAgent) ExecuteToolDirect(ctx context.Context, toolName string, input map[string]any) (any, error) {
	return nil, fmt.Errorf("RemoteAgent does not support ExecuteToolDirect operation")
}

// RespondToPermissionRequest 不支持 - 权限管理在远程端
func (r *RemoteAgent) RespondToPermissionRequest(callID string, approved bool) error {
	return fmt.Errorf("RemoteAgent does not support RespondToPermissionRequest operation")
}

// HasPendingPermission 不支持 - 权限管理在远程端
func (r *RemoteAgent) HasPendingPermission(callID string) bool {
	return false
}

// RespondToIterationLimit 不支持 - 迭代管理在远程端
func (r *RemoteAgent) RespondToIterationLimit(continueExecution bool) {
	// No-op
}

// SetMaxIterations 不支持 - 配置在远程端
func (r *RemoteAgent) SetMaxIterations(max int) {
	// No-op
}

// GetIterationCount 返回 0 - 迭代计数在远程端
func (r *RemoteAgent) GetIterationCount() int {
	return 0
}

// SetPermissionMode 不支持 - 权限模式在远程端
func (r *RemoteAgent) SetPermissionMode(mode permission.Mode) {
	// No-op
}

// GetPermissionMode 返回默认模式
func (r *RemoteAgent) GetPermissionMode() permission.Mode {
	return permission.ModeAlwaysAsk
}

// GetSystemPrompt 返回空字符串 - 系统提示词在远程端
func (r *RemoteAgent) GetSystemPrompt() string {
	return ""
}
