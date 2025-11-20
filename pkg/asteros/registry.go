package asteros

import (
	"sync"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/agent/workflow"
	"github.com/astercloud/aster/pkg/core"
)

// ResourceType 资源类型
type ResourceType string

const (
	ResourceTypeAgent    ResourceType = "agent"
	ResourceTypeRoom     ResourceType = "room"
	ResourceTypeWorkflow ResourceType = "workflow"
)

// Resource 资源信息
type Resource struct {
	ID   string       // 资源 ID
	Name string       // 资源名称
	Type ResourceType // 资源类型
	Data interface{}  // 资源数据（Agent、Room 或 Workflow）
}

// Registry 资源注册表
type Registry struct {
	mu        sync.RWMutex
	agents    map[string]*agent.Agent
	rooms     map[string]*core.Room
	workflows map[string]workflow.Agent
}

// NewRegistry 创建资源注册表
func NewRegistry() *Registry {
	return &Registry{
		agents:    make(map[string]*agent.Agent),
		rooms:     make(map[string]*core.Room),
		workflows: make(map[string]workflow.Agent),
	}
}

// RegisterAgent 注册 Agent
func (r *Registry) RegisterAgent(id string, ag *agent.Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.agents[id]; exists {
		return ErrResourceExists
	}

	r.agents[id] = ag
	return nil
}

// RegisterRoom 注册 Room
func (r *Registry) RegisterRoom(id string, room *core.Room) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.rooms[id]; exists {
		return ErrResourceExists
	}

	r.rooms[id] = room
	return nil
}

// RegisterWorkflow 注册 Workflow
func (r *Registry) RegisterWorkflow(id string, wf workflow.Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.workflows[id]; exists {
		return ErrResourceExists
	}

	r.workflows[id] = wf
	return nil
}

// GetAgent 获取 Agent
func (r *Registry) GetAgent(id string) (*agent.Agent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ag, exists := r.agents[id]
	return ag, exists
}

// GetRoom 获取 Room
func (r *Registry) GetRoom(id string) (*core.Room, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	room, exists := r.rooms[id]
	return room, exists
}

// GetWorkflow 获取 Workflow
func (r *Registry) GetWorkflow(id string) (workflow.Agent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	wf, exists := r.workflows[id]
	return wf, exists
}

// ListAgents 列出所有 Agents
func (r *Registry) ListAgents() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.agents))
	for id := range r.agents {
		ids = append(ids, id)
	}
	return ids
}

// ListRooms 列出所有 Rooms
func (r *Registry) ListRooms() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.rooms))
	for id := range r.rooms {
		ids = append(ids, id)
	}
	return ids
}

// ListWorkflows 列出所有 Workflows
func (r *Registry) ListWorkflows() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.workflows))
	for id := range r.workflows {
		ids = append(ids, id)
	}
	return ids
}

// ListAll 列出所有资源
func (r *Registry) ListAll() []Resource {
	r.mu.RLock()
	defer r.mu.RUnlock()

	resources := make([]Resource, 0)

	for id, ag := range r.agents {
		resources = append(resources, Resource{
			ID:   id,
			Name: ag.ID(),
			Type: ResourceTypeAgent,
			Data: ag,
		})
	}

	for id, room := range r.rooms {
		resources = append(resources, Resource{
			ID:   id,
			Name: id, // Room 没有 Name() 方法，使用 ID
			Type: ResourceTypeRoom,
			Data: room,
		})
	}

	for id, wf := range r.workflows {
		resources = append(resources, Resource{
			ID:   id,
			Name: wf.Name(),
			Type: ResourceTypeWorkflow,
			Data: wf,
		})
	}

	return resources
}

// UnregisterAgent 注销 Agent
func (r *Registry) UnregisterAgent(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.agents[id]; !exists {
		return ErrAgentNotFound
	}

	delete(r.agents, id)
	return nil
}

// UnregisterRoom 注销 Room
func (r *Registry) UnregisterRoom(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.rooms[id]; !exists {
		return ErrRoomNotFound
	}

	delete(r.rooms, id)
	return nil
}

// UnregisterWorkflow 注销 Workflow
func (r *Registry) UnregisterWorkflow(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.workflows[id]; !exists {
		return ErrWorkflowNotFound
	}

	delete(r.workflows, id)
	return nil
}

// Clear 清空所有资源
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.agents = make(map[string]*agent.Agent)
	r.rooms = make(map[string]*core.Room)
	r.workflows = make(map[string]workflow.Agent)
}
