package core

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/agent"
)

// RoomMember Room 成员信息
type RoomMember struct {
	Name    string `json:"name"`
	AgentID string `json:"agent_id"`
}

// Room 多 Agent 协作空间
// 提供 Agent 间消息路由、广播和点对点通信功能
type Room struct {
	mu      sync.RWMutex
	pool    *Pool
	members map[string]string // name -> agentID

	// 消息历史 (可选)
	history []RoomMessage

	// 提及正则表达式
	mentionRegex *regexp.Regexp
}

// RoomMessage Room 消息记录
type RoomMessage struct {
	From    string   `json:"from"`
	To      []string `json:"to,omitempty"` // 空表示广播
	Text    string   `json:"text"`
	Sent    int64    `json:"sent"` // Unix timestamp
}

// NewRoom 创建新的 Room
func NewRoom(pool *Pool) *Room {
	return &Room{
		pool:         pool,
		members:      make(map[string]string),
		history:      make([]RoomMessage, 0),
		mentionRegex: regexp.MustCompile(`@(\w+)`),
	}
}

// Join 加入 Room
func (r *Room) Join(name string, agentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查名称是否已存在
	if _, exists := r.members[name]; exists {
		return fmt.Errorf("member already exists: %s", name)
	}

	// 检查 Agent 是否存在
	_, exists := r.pool.Get(agentID)
	if !exists {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	r.members[name] = agentID
	return nil
}

// Leave 离开 Room
func (r *Room) Leave(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.members[name]; !exists {
		return fmt.Errorf("member not found: %s", name)
	}

	delete(r.members, name)
	return nil
}

// Say 在 Room 中发送消息
// - 如果消息包含 @mention,则发送给被提及的成员 (点对点)
// - 否则广播给除发送者外的所有成员
func (r *Room) Say(ctx context.Context, from string, text string) error {
	r.mu.RLock()

	// 检查发送者是否是成员
	if _, exists := r.members[from]; !exists {
		r.mu.RUnlock()
		return fmt.Errorf("sender is not a member: %s", from)
	}

	// 提取提及的成员
	mentions := r.extractMentions(text)

	// 记录消息
	msg := RoomMessage{
		From: from,
		Text: text,
		Sent: nowTimestamp(),
	}

	var recipients []string
	var targets map[string]string

	if len(mentions) > 0 {
		// 定向消息
		msg.To = mentions
		targets = make(map[string]string)
		for _, mention := range mentions {
			if agentID, exists := r.members[mention]; exists {
				targets[mention] = agentID
				recipients = append(recipients, mention)
			}
		}
	} else {
		// 广播消息
		targets = make(map[string]string)
		for name, agentID := range r.members {
			if name != from {
				targets[name] = agentID
				recipients = append(recipients, name)
			}
		}
	}

	r.mu.RUnlock()

	// 记录到历史
	r.mu.Lock()
	r.history = append(r.history, msg)
	r.mu.Unlock()

	// 发送消息
	for name, agentID := range targets {
		ag, exists := r.pool.Get(agentID)
		if !exists {
			continue
		}

		// 格式化消息: [from:sender] message
		formattedText := fmt.Sprintf("[from:%s] %s", from, text)

		// 异步发送,避免阻塞
		go func(agent *agent.Agent, txt string, memberName string) {
			if err := agent.Send(ctx, txt); err != nil {
				// 记录错误但不中断其他消息发送
				// 这里可以通过回调或事件通知上层
			}
		}(ag, formattedText, name)
	}

	return nil
}

// Broadcast 广播消息给所有成员 (包括发送者)
func (r *Room) Broadcast(ctx context.Context, text string) error {
	r.mu.RLock()

	// 复制成员列表
	targets := make(map[string]string, len(r.members))
	for name, agentID := range r.members {
		targets[name] = agentID
	}

	r.mu.RUnlock()

	// 记录到历史
	msg := RoomMessage{
		From: "system",
		Text: text,
		Sent: nowTimestamp(),
	}

	r.mu.Lock()
	r.history = append(r.history, msg)
	r.mu.Unlock()

	// 发送消息
	for _, agentID := range targets {
		ag, exists := r.pool.Get(agentID)
		if !exists {
			continue
		}

		go func(agent *agent.Agent, txt string) {
			_ = agent.Send(ctx, txt)
		}(ag, text)
	}

	return nil
}

// SendTo 发送消息给指定成员
func (r *Room) SendTo(ctx context.Context, from string, to string, text string) error {
	r.mu.RLock()

	// 检查发送者
	if _, exists := r.members[from]; !exists && from != "system" {
		r.mu.RUnlock()
		return fmt.Errorf("sender is not a member: %s", from)
	}

	// 检查接收者
	agentID, exists := r.members[to]
	if !exists {
		r.mu.RUnlock()
		return fmt.Errorf("recipient not found: %s", to)
	}

	r.mu.RUnlock()

	// 记录到历史
	msg := RoomMessage{
		From: from,
		To:   []string{to},
		Text: text,
		Sent: nowTimestamp(),
	}

	r.mu.Lock()
	r.history = append(r.history, msg)
	r.mu.Unlock()

	// 获取 Agent 并发送
	ag, exists := r.pool.Get(agentID)
	if !exists {
		return fmt.Errorf("agent not found for member %s", to)
	}

	formattedText := fmt.Sprintf("[from:%s] %s", from, text)
	return ag.Send(ctx, formattedText)
}

// GetMembers 获取所有成员
func (r *Room) GetMembers() []RoomMember {
	r.mu.RLock()
	defer r.mu.RUnlock()

	members := make([]RoomMember, 0, len(r.members))
	for name, agentID := range r.members {
		members = append(members, RoomMember{
			Name:    name,
			AgentID: agentID,
		})
	}

	return members
}

// GetMemberCount 获取成员数量
func (r *Room) GetMemberCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.members)
}

// IsMember 检查是否是成员
func (r *Room) IsMember(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.members[name]
	return exists
}

// GetAgentID 获取成员对应的 Agent ID
func (r *Room) GetAgentID(name string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agentID, exists := r.members[name]
	return agentID, exists
}

// GetHistory 获取消息历史
func (r *Room) GetHistory() []RoomMessage {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 返回副本
	history := make([]RoomMessage, len(r.history))
	copy(history, r.history)
	return history
}

// ClearHistory 清空消息历史
func (r *Room) ClearHistory() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.history = make([]RoomMessage, 0)
}

// extractMentions 提取消息中的 @mentions
func (r *Room) extractMentions(text string) []string {
	matches := r.mentionRegex.FindAllStringSubmatch(text, -1)
	mentions := make([]string, 0, len(matches))

	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			name := match[1]
			// 去重
			if !seen[name] {
				mentions = append(mentions, name)
				seen[name] = true
			}
		}
	}

	return mentions
}

// nowTimestamp 获取当前时间戳 (毫秒)
func nowTimestamp() int64 {
	return time.Now().UnixMilli()
}
