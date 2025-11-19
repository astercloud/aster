package stars

import (
	"context"
	"fmt"
	"iter"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/cosmos"
	"github.com/astercloud/aster/pkg/types"
)

// Stars 群星 - 多 Agent 协作单元
// Stars 是 Aster 框架中的多 Agent 协作组件，
// 负责管理多个 Agent 之间的协作、通信和任务执行。
type Stars struct {
	mu      sync.RWMutex
	id      string
	name    string
	cosmos  *cosmos.Cosmos
	members map[string]*Member // Agent ID -> Member
	history []Message          // 消息历史（可选）
}

// New 创建群星协作组
func New(cosmos *cosmos.Cosmos, name string) *Stars {
	return &Stars{
		id:      fmt.Sprintf("stars-%d", time.Now().UnixNano()),
		name:    name,
		cosmos:  cosmos,
		members: make(map[string]*Member),
		history: make([]Message, 0),
	}
}

// Join 添加成员到群星
func (s *Stars) Join(agentID string, role Role) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查 Agent 是否存在于 Cosmos
	_, exists := s.cosmos.Get(agentID)
	if !exists {
		return fmt.Errorf("agent not found in cosmos: %s", agentID)
	}

	// 检查是否已经是成员
	if _, exists := s.members[agentID]; exists {
		return fmt.Errorf("agent already in stars: %s", agentID)
	}

	// 添加成员
	s.members[agentID] = &Member{
		AgentID: agentID,
		Role:    role,
		Tags:    make([]string, 0),
	}

	return nil
}

// Leave 从群星中移除成员
func (s *Stars) Leave(agentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否是成员
	if _, exists := s.members[agentID]; !exists {
		return fmt.Errorf("agent not in stars: %s", agentID)
	}

	// 移除成员
	delete(s.members, agentID)
	return nil
}

// Members 获取所有成员
func (s *Stars) Members() []Member {
	s.mu.RLock()
	defer s.mu.RUnlock()

	members := make([]Member, 0, len(s.members))
	for _, m := range s.members {
		members = append(members, *m)
	}
	return members
}

// Broadcast 广播消息给所有成员
func (s *Stars) Broadcast(ctx context.Context, text string) error {
	s.mu.RLock()
	members := make([]*Member, 0, len(s.members))
	for _, m := range s.members {
		members = append(members, m)
	}
	s.mu.RUnlock()

	// 记录消息
	msg := Message{
		From: "system",
		To:   "",
		Text: text,
		Time: time.Now(),
	}
	s.recordMessage(msg)

	// 发送给所有成员
	var lastErr error
	for _, member := range members {
		ag, exists := s.cosmos.Get(member.AgentID)
		if !exists {
			lastErr = fmt.Errorf("agent not found: %s", member.AgentID)
			continue
		}

		// 异步发送消息
		go func(agentID string) {
			if err := ag.Send(ctx, text); err != nil {
				// 记录错误但不阻塞
				fmt.Printf("Failed to send message to %s: %v\n", agentID, err)
			}
		}(member.AgentID)
	}

	return lastErr
}

// Send 发送点对点消息
func (s *Stars) Send(ctx context.Context, from, to, text string) error {
	s.mu.RLock()
	// 检查发送者是否是成员
	if _, exists := s.members[from]; !exists {
		s.mu.RUnlock()
		return fmt.Errorf("sender not in stars: %s", from)
	}

	// 检查接收者是否是成员
	if _, exists := s.members[to]; !exists {
		s.mu.RUnlock()
		return fmt.Errorf("receiver not in stars: %s", to)
	}
	s.mu.RUnlock()

	// 记录消息
	msg := Message{
		From: from,
		To:   to,
		Text: text,
		Time: time.Now(),
	}
	s.recordMessage(msg)

	// 获取接收者 Agent
	ag, exists := s.cosmos.Get(to)
	if !exists {
		return fmt.Errorf("agent not found: %s", to)
	}

	// 异步发送消息（避免阻塞）
	go func() {
		if err := ag.Send(ctx, text); err != nil {
			fmt.Printf("Failed to send message from %s to %s: %v\n", from, to, err)
		}
	}()

	return nil
}

// Run 执行任务（Leader-Worker 模式）
// 返回一个迭代器，流式返回执行事件
func (s *Stars) Run(ctx context.Context, task string) iter.Seq2[*Event, error] {
	return func(yield func(*Event, error) bool) {
		// 1. 找到 Leader
		leader := s.findLeader()
		if leader == nil {
			yield(nil, fmt.Errorf("no leader in stars"))
			return
		}

		// 2. 获取 Leader Agent
		leaderAgent, exists := s.cosmos.Get(leader.AgentID)
		if !exists {
			yield(nil, fmt.Errorf("leader agent not found: %s", leader.AgentID))
			return
		}

		// 3. 发送任务给 Leader
		if err := leaderAgent.Send(ctx, task); err != nil {
			yield(nil, fmt.Errorf("failed to send task to leader: %w", err))
			return
		}

		// 4. 订阅 Leader 的事件
		eventCh := leaderAgent.Subscribe(
			[]types.AgentChannel{types.ChannelProgress},
			nil,
		)

		// 5. 流式返回事件
		for envelope := range eventCh {
			event := &Event{
				AgentID: leader.AgentID,
				Time:    time.Now(),
			}

			// 根据事件类型提取内容
			switch e := envelope.Event.(type) {
			case *types.ProgressTextChunkEvent:
				event.Type = "text_chunk"
				event.Content = e.Delta
			case *types.ProgressDoneEvent:
				event.Type = "done"
				event.Content = "Task completed"
				if !yield(event, nil) {
					return
				}
				return // 任务完成，退出
			case *types.ProgressToolStartEvent:
				event.Type = "tool_start"
				event.Content = fmt.Sprintf("Tool started: %s", e.Call.Name)
			case *types.ProgressToolEndEvent:
				event.Type = "tool_end"
				event.Content = fmt.Sprintf("Tool completed: %s", e.Call.Name)
			case types.EventType:
				event.Type = e.EventType()
				event.Content = fmt.Sprintf("Event: %s", e.EventType())
			default:
				event.Type = "unknown"
				event.Content = "Unknown event"
			}

			// 返回事件
			if !yield(event, nil) {
				return
			}
		}
	}
}

// findLeader 查找 Leader
func (s *Stars) findLeader() *Member {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, member := range s.members {
		if member.Role == RoleLeader {
			return member
		}
	}
	return nil
}

// recordMessage 记录消息到历史
func (s *Stars) recordMessage(msg Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.history = append(s.history, msg)

	// 限制历史记录大小（保留最近 100 条）
	if len(s.history) > 100 {
		s.history = s.history[len(s.history)-100:]
	}
}

// History 获取消息历史
func (s *Stars) History() []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history := make([]Message, len(s.history))
	copy(history, s.history)
	return history
}

// Name 获取群星名称
func (s *Stars) Name() string {
	return s.name
}

// ID 获取群星 ID
func (s *Stars) ID() string {
	return s.id
}

// Size 获取成员数量
func (s *Stars) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.members)
}
