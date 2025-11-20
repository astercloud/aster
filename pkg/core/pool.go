package core

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/types"
)

// PoolOptions Agent 池配置
type PoolOptions struct {
	Dependencies *agent.Dependencies
	MaxAgents    int // 最大 Agent 数量,默认 50
}

// Pool Agent 池 - 管理多个 Agent 的生命周期
type Pool struct {
	mu        sync.RWMutex
	agents    map[string]*agent.Agent
	deps      *agent.Dependencies
	maxAgents int
}

// NewPool 创建 Agent 池
func NewPool(opts *PoolOptions) *Pool {
	maxAgents := opts.MaxAgents
	if maxAgents == 0 {
		maxAgents = 50
	}

	return &Pool{
		agents:    make(map[string]*agent.Agent),
		deps:      opts.Dependencies,
		maxAgents: maxAgents,
	}
}

// Create 创建新 Agent 并加入池
func (p *Pool) Create(ctx context.Context, config *types.AgentConfig) (*agent.Agent, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 检查是否已存在
	if _, exists := p.agents[config.AgentID]; exists {
		return nil, fmt.Errorf("agent already exists: %s", config.AgentID)
	}

	// 检查池容量
	if len(p.agents) >= p.maxAgents {
		return nil, fmt.Errorf("pool is full (max %d agents)", p.maxAgents)
	}

	// 创建 Agent
	ag, err := agent.Create(ctx, config, p.deps)
	if err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}

	// 加入池
	p.agents[config.AgentID] = ag
	return ag, nil
}

// Get 获取指定 Agent
func (p *Pool) Get(agentID string) (*agent.Agent, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	ag, exists := p.agents[agentID]
	return ag, exists
}

// List 列出所有 Agent ID
func (p *Pool) List(prefix string) []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	ids := make([]string, 0, len(p.agents))
	for id := range p.agents {
		if prefix == "" || strings.HasPrefix(id, prefix) {
			ids = append(ids, id)
		}
	}
	return ids
}

// Status 获取 Agent 状态
func (p *Pool) Status(agentID string) (*types.AgentStatus, error) {
	p.mu.RLock()
	ag, exists := p.agents[agentID]
	p.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	return ag.Status(), nil
}

// Resume 从存储中恢复 Agent
func (p *Pool) Resume(ctx context.Context, agentID string, config *types.AgentConfig) (*agent.Agent, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 1. 检查是否已在池中
	if ag, exists := p.agents[agentID]; exists {
		return ag, nil
	}

	// 2. 检查池容量
	if len(p.agents) >= p.maxAgents {
		return nil, fmt.Errorf("pool is full (max %d agents)", p.maxAgents)
	}

	// 3. 检查存储中是否存在
	_, err := p.deps.Store.LoadMessages(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("agent not found in store: %s", agentID)
	}

	// 4. 设置 AgentID
	config.AgentID = agentID

	// 5. 创建 Agent (会自动加载状态)
	ag, err := agent.Create(ctx, config, p.deps)
	if err != nil {
		return nil, fmt.Errorf("resume agent: %w", err)
	}

	// 6. 加入池
	p.agents[agentID] = ag
	return ag, nil
}

// Remove 从池中移除 Agent (不删除存储)
func (p *Pool) Remove(agentID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	ag, exists := p.agents[agentID]
	if !exists {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	// 关闭 Agent
	if err := ag.Close(); err != nil {
		return fmt.Errorf("close agent: %w", err)
	}

	// 从池中移除
	delete(p.agents, agentID)
	return nil
}

// Delete 删除 Agent (包括存储)
func (p *Pool) Delete(ctx context.Context, agentID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 从池中移除
	if ag, exists := p.agents[agentID]; exists {
		if err := ag.Close(); err != nil {
			return fmt.Errorf("close agent: %w", err)
		}
		delete(p.agents, agentID)
	}

	// 从存储中删除 (需要 Store 实现 Delete 方法)
	// TODO: 实现 Store.Delete() 方法
	return nil
}

// Size 返回池中 Agent 数量
func (p *Pool) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.agents)
}

// Shutdown 关闭所有 Agent
func (p *Pool) Shutdown() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var lastErr error
	for id, ag := range p.agents {
		if err := ag.Close(); err != nil {
			lastErr = fmt.Errorf("close agent %s: %w", id, err)
		}
	}

	// 清空池
	p.agents = make(map[string]*agent.Agent)
	return lastErr
}

// ForEach 遍历所有 Agent
func (p *Pool) ForEach(fn func(agentID string, ag *agent.Agent) error) error {
	p.mu.RLock()
	// 复制一份避免长时间持锁
	agents := make(map[string]*agent.Agent, len(p.agents))
	for id, ag := range p.agents {
		agents[id] = ag
	}
	p.mu.RUnlock()

	for id, ag := range agents {
		if err := fn(id, ag); err != nil {
			return err
		}
	}
	return nil
}
