package interfaces

import (
	"context"
	"fmt"
	"sync"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/agent/workflow"
	"github.com/astercloud/aster/pkg/asteros"
	"github.com/astercloud/aster/pkg/stars"
)

// A2AInterfaceOptions A2A Interface 配置
type A2AInterfaceOptions struct {
	// GRPCPort gRPC 服务端口
	GRPCPort int

	// EnableLogging 是否启用日志
	EnableLogging bool

	// EnableDiscovery 是否启用服务发现
	EnableDiscovery bool
}

// A2AInterface Agent-to-Agent Interface
// A2AInterface 提供 Agent 之间的直接通信能力，
// 支持消息传递、状态查询和协作。
type A2AInterface struct {
	*asteros.BaseInterface
	opts *A2AInterfaceOptions
	os   *asteros.AsterOS

	// Agent 注册表（用于快速查找）
	agents map[string]*agent.Agent
	mu     sync.RWMutex
}

// NewA2AInterface 创建 A2A Interface
func NewA2AInterface(opts *A2AInterfaceOptions) *A2AInterface {
	if opts == nil {
		opts = &A2AInterfaceOptions{
			GRPCPort:        9090,
			EnableLogging:   true,
			EnableDiscovery: true,
		}
	}

	return &A2AInterface{
		BaseInterface: asteros.NewBaseInterface("a2a", asteros.InterfaceTypeA2A),
		opts:          opts,
		agents:        make(map[string]*agent.Agent),
	}
}

// Start 启动 A2A Interface
func (i *A2AInterface) Start(ctx context.Context, os *asteros.AsterOS) error {
	i.os = os

	// TODO: 启动 gRPC 服务器
	// 这里需要实现完整的 gRPC 服务器和 Agent-to-Agent 协议

	if i.opts.EnableLogging {
		fmt.Printf("✓ A2A Interface started on port %d\n", i.opts.GRPCPort)
	}

	return nil
}

// Stop 停止 A2A Interface
func (i *A2AInterface) Stop(ctx context.Context) error {
	// TODO: 停止 gRPC 服务器

	if i.opts.EnableLogging {
		fmt.Printf("✓ A2A Interface stopped\n")
	}

	return nil
}

// OnAgentRegistered Agent 注册事件
func (i *A2AInterface) OnAgentRegistered(ag *agent.Agent) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.agents[ag.ID()] = ag

	if i.opts.EnableLogging {
		fmt.Printf("  [A2A] Agent registered: %s\n", ag.ID())
		fmt.Printf("    → Available for Agent-to-Agent communication\n")
	}

	return nil
}

// OnStarsRegistered Stars 注册事件
func (i *A2AInterface) OnStarsRegistered(s *stars.Stars) error {
	if i.opts.EnableLogging {
		fmt.Printf("  [A2A] Stars registered: %s\n", s.Name())
		fmt.Printf("    → Members can communicate via A2A\n")
	}
	return nil
}

// OnWorkflowRegistered Workflow 注册事件
func (i *A2AInterface) OnWorkflowRegistered(wf workflow.Agent) error {
	if i.opts.EnableLogging {
		fmt.Printf("  [A2A] Workflow registered: %s\n", wf.Name())
	}
	return nil
}

// SendMessage 发送消息给指定 Agent
func (i *A2AInterface) SendMessage(ctx context.Context, fromID, toID, message string) error {
	i.mu.RLock()
	defer i.mu.RUnlock()

	// 查找目标 Agent
	toAgent, exists := i.agents[toID]
	if !exists {
		return fmt.Errorf("agent not found: %s", toID)
	}

	// 发送消息
	if err := toAgent.Send(ctx, message); err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	if i.opts.EnableLogging {
		fmt.Printf("  [A2A] Message sent: %s → %s\n", fromID, toID)
	}

	return nil
}

// BroadcastMessage 广播消息给所有 Agents
func (i *A2AInterface) BroadcastMessage(ctx context.Context, fromID, message string) error {
	i.mu.RLock()
	defer i.mu.RUnlock()

	count := 0
	for id, ag := range i.agents {
		if id == fromID {
			continue // 不发送给自己
		}

		if err := ag.Send(ctx, message); err != nil {
			fmt.Printf("  [A2A] Warning: failed to send to %s: %v\n", id, err)
			continue
		}

		count++
	}

	if i.opts.EnableLogging {
		fmt.Printf("  [A2A] Broadcast message from %s to %d agents\n", fromID, count)
	}

	return nil
}

// QueryAgent 查询 Agent 状态
func (i *A2AInterface) QueryAgent(agentID string) (map[string]interface{}, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	ag, exists := i.agents[agentID]
	if !exists {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	status := ag.Status()

	return map[string]interface{}{
		"agent_id": status.AgentID,
		"state":    status.State,
	}, nil
}

// ListAgents 列出所有可用的 Agents
func (i *A2AInterface) ListAgents() []string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	ids := make([]string, 0, len(i.agents))
	for id := range i.agents {
		ids = append(ids, id)
	}

	return ids
}
