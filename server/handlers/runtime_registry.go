package handlers

import (
	"sync"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/events"
)

// RegistryEventListener is called when agents are registered/unregistered
type RegistryEventListener func(agentID string, ag *agent.Agent, registered bool)

// RuntimeAgentRegistry 简单的内存 Agent 注册表，用于查询运行态信息
type RuntimeAgentRegistry struct {
	mu              sync.RWMutex
	agents          map[string]*agent.Agent
	remoteAgents    map[string]*agent.RemoteAgent // 远程 Agent 注册表
	listeners       []RegistryEventListener
	remoteListeners []RemoteAgentEventListener
}

func NewRuntimeAgentRegistry() *RuntimeAgentRegistry {
	return &RuntimeAgentRegistry{
		agents:          make(map[string]*agent.Agent),
		remoteAgents:    make(map[string]*agent.RemoteAgent),
		listeners:       make([]RegistryEventListener, 0),
		remoteListeners: make([]RemoteAgentEventListener, 0),
	}
}

func (r *RuntimeAgentRegistry) Register(ag *agent.Agent) {
	if ag == nil {
		return
	}
	r.mu.Lock()
	r.agents[ag.ID()] = ag
	listeners := make([]RegistryEventListener, len(r.listeners))
	copy(listeners, r.listeners)
	r.mu.Unlock()

	// Notify listeners outside of lock
	for _, listener := range listeners {
		listener(ag.ID(), ag, true)
	}
}

func (r *RuntimeAgentRegistry) Unregister(agentID string) {
	if agentID == "" {
		return
	}
	r.mu.Lock()
	ag := r.agents[agentID]
	delete(r.agents, agentID)
	listeners := make([]RegistryEventListener, len(r.listeners))
	copy(listeners, r.listeners)
	r.mu.Unlock()

	// Notify listeners outside of lock
	for _, listener := range listeners {
		listener(agentID, ag, false)
	}
}

func (r *RuntimeAgentRegistry) Get(agentID string) *agent.Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.agents[agentID]
}

// List returns all registered agents
func (r *RuntimeAgentRegistry) List() []*agent.Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agents := make([]*agent.Agent, 0, len(r.agents))
	for _, ag := range r.agents {
		agents = append(agents, ag)
	}
	return agents
}

// Count returns the number of registered agents
func (r *RuntimeAgentRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.agents)
}

// AddListener adds a listener that will be notified when agents are registered/unregistered
func (r *RuntimeAgentRegistry) AddListener(listener RegistryEventListener) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.listeners = append(r.listeners, listener)
}

// RemoteAgentEventListener is called when remote agents are registered/unregistered
type RemoteAgentEventListener func(agentID string, ra *agent.RemoteAgent, registered bool)

// RegisterRemoteAgent 注册远程 Agent
func (r *RuntimeAgentRegistry) RegisterRemoteAgent(ra *agent.RemoteAgent) {
	if ra == nil {
		return
	}
	r.mu.Lock()
	r.remoteAgents[ra.ID()] = ra
	listeners := make([]RemoteAgentEventListener, len(r.remoteListeners))
	copy(listeners, r.remoteListeners)
	r.mu.Unlock()

	// Notify listeners outside of lock
	for _, listener := range listeners {
		listener(ra.ID(), ra, true)
	}
}

// UnregisterRemoteAgent 注销远程 Agent
func (r *RuntimeAgentRegistry) UnregisterRemoteAgent(agentID string) {
	if agentID == "" {
		return
	}
	r.mu.Lock()
	ra := r.remoteAgents[agentID]
	delete(r.remoteAgents, agentID)
	listeners := make([]RemoteAgentEventListener, len(r.remoteListeners))
	copy(listeners, r.remoteListeners)
	r.mu.Unlock()

	// Notify listeners outside of lock
	for _, listener := range listeners {
		listener(agentID, ra, false)
	}
}

// ListRemoteAgents returns all registered remote agents
func (r *RuntimeAgentRegistry) ListRemoteAgents() []*agent.RemoteAgent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agents := make([]*agent.RemoteAgent, 0, len(r.remoteAgents))
	for _, ra := range r.remoteAgents {
		agents = append(agents, ra)
	}
	return agents
}

// AddRemoteAgentListener adds a listener for remote agent registration events
func (r *RuntimeAgentRegistry) AddRemoteAgentListener(listener RemoteAgentEventListener) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.remoteListeners = append(r.remoteListeners, listener)
}

// GetRemoteAgent 获取远程 Agent
func (r *RuntimeAgentRegistry) GetRemoteAgent(agentID string) *agent.RemoteAgent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.remoteAgents[agentID]
}

// GetEventBuses returns all EventBuses from registered agents (including remote agents)
// Implements dashboard.EventBusProvider interface
func (r *RuntimeAgentRegistry) GetEventBuses() []*events.EventBus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	buses := make([]*events.EventBus, 0, len(r.agents)+len(r.remoteAgents))

	// 本地 Agent 的 EventBus
	for _, ag := range r.agents {
		if ag != nil {
			if eb := ag.GetEventBus(); eb != nil {
				buses = append(buses, eb)
			}
		}
	}

	// 远程 Agent 的 EventBus
	for _, ra := range r.remoteAgents {
		if ra != nil {
			if eb := ra.GetEventBus(); eb != nil {
				buses = append(buses, eb)
			}
		}
	}

	return buses
}
