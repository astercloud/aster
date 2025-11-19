package cosmos

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/types"
)

// Options Cosmos 配置
type Options struct {
	Dependencies *agent.Dependencies
	MaxAgents    int // 最大 Agent 数量,默认 50
}

// Cosmos 宇宙 - 管理多个 Agent 的生命周期
// Cosmos 是 Aster 框架中的 Agent 生命周期管理器，
// 负责创建、获取、删除和管理所有 Agent 实例。
type Cosmos struct {
	mu        sync.RWMutex
	agents    map[string]*agent.Agent
	deps      *agent.Dependencies
	maxAgents int
}

// New 创建 Cosmos 宇宙
func New(opts *Options) *Cosmos {
	maxAgents := opts.MaxAgents
	if maxAgents == 0 {
		maxAgents = 50
	}

	return &Cosmos{
		agents:    make(map[string]*agent.Agent),
		deps:      opts.Dependencies,
		maxAgents: maxAgents,
	}
}

// Create 创建新 Agent 并加入宇宙
func (c *Cosmos) Create(ctx context.Context, config *types.AgentConfig) (*agent.Agent, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查是否已存在
	if _, exists := c.agents[config.AgentID]; exists {
		return nil, fmt.Errorf("agent already exists: %s", config.AgentID)
	}

	// 检查宇宙容量
	if len(c.agents) >= c.maxAgents {
		return nil, fmt.Errorf("cosmos is full (max %d agents)", c.maxAgents)
	}

	// 创建 Agent
	ag, err := agent.Create(ctx, config, c.deps)
	if err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}

	// 加入宇宙
	c.agents[config.AgentID] = ag
	return ag, nil
}

// Get 获取指定 Agent
func (c *Cosmos) Get(agentID string) (*agent.Agent, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ag, exists := c.agents[agentID]
	return ag, exists
}

// List 列出所有 Agent ID
func (c *Cosmos) List(prefix string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ids := make([]string, 0, len(c.agents))
	for id := range c.agents {
		if prefix == "" || strings.HasPrefix(id, prefix) {
			ids = append(ids, id)
		}
	}
	return ids
}

// Status 获取 Agent 状态
func (c *Cosmos) Status(agentID string) (*types.AgentStatus, error) {
	c.mu.RLock()
	ag, exists := c.agents[agentID]
	c.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	return ag.Status(), nil
}

// Resume 从存储中恢复 Agent
func (c *Cosmos) Resume(ctx context.Context, agentID string, config *types.AgentConfig) (*agent.Agent, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 1. 检查是否已在宇宙中
	if ag, exists := c.agents[agentID]; exists {
		return ag, nil
	}

	// 2. 检查宇宙容量
	if len(c.agents) >= c.maxAgents {
		return nil, fmt.Errorf("cosmos is full (max %d agents)", c.maxAgents)
	}

	// 3. 检查存储中是否存在
	_, err := c.deps.Store.LoadMessages(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("agent not found in store: %s", agentID)
	}

	// 4. 设置 AgentID
	config.AgentID = agentID

	// 5. 创建 Agent (会自动加载状态)
	ag, err := agent.Create(ctx, config, c.deps)
	if err != nil {
		return nil, fmt.Errorf("resume agent: %w", err)
	}

	// 6. 加入宇宙
	c.agents[agentID] = ag
	return ag, nil
}

// ResumeAll 恢复所有存储的 Agent
func (c *Cosmos) ResumeAll(ctx context.Context, configFactory func(agentID string) *types.AgentConfig) ([]*agent.Agent, error) {
	// 获取所有 Agent ID (需要 Store 实现 List 方法)
	// 这里简化实现,假设外部提供 ID 列表
	// 实际应该从 Store.ListAgents() 获取

	resumed := make([]*agent.Agent, 0)
	// TODO: 实现 Store.ListAgents() 方法
	return resumed, fmt.Errorf("resumeAll not fully implemented: need Store.ListAgents()")
}

// Remove 从宇宙中移除 Agent (不删除存储)
func (c *Cosmos) Remove(agentID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	ag, exists := c.agents[agentID]
	if !exists {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	// 关闭 Agent
	if err := ag.Close(); err != nil {
		return fmt.Errorf("close agent: %w", err)
	}

	// 从宇宙中移除
	delete(c.agents, agentID)
	return nil
}

// Delete 删除 Agent (包括存储)
func (c *Cosmos) Delete(ctx context.Context, agentID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 从宇宙中移除
	if ag, exists := c.agents[agentID]; exists {
		if err := ag.Close(); err != nil {
			return fmt.Errorf("close agent: %w", err)
		}
		delete(c.agents, agentID)
	}

	// 从存储中删除 (需要 Store 实现 Delete 方法)
	// TODO: 实现 Store.Delete() 方法
	return nil
}

// Size 返回宇宙中 Agent 数量
func (c *Cosmos) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.agents)
}

// Shutdown 关闭所有 Agent
func (c *Cosmos) Shutdown() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var lastErr error
	for id, ag := range c.agents {
		if err := ag.Close(); err != nil {
			lastErr = fmt.Errorf("close agent %s: %w", id, err)
		}
	}

	// 清空宇宙
	c.agents = make(map[string]*agent.Agent)
	return lastErr
}

// ForEach 遍历所有 Agent
func (c *Cosmos) ForEach(fn func(agentID string, ag *agent.Agent) error) error {
	c.mu.RLock()
	// 复制一份避免长时间持锁
	agents := make(map[string]*agent.Agent, len(c.agents))
	for id, ag := range c.agents {
		agents[id] = ag
	}
	c.mu.RUnlock()

	for id, ag := range agents {
		if err := fn(id, ag); err != nil {
			return err
		}
	}
	return nil
}
