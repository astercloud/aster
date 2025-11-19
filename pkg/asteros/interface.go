package asteros

import (
	"context"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/agent/workflow"
	"github.com/astercloud/aster/pkg/stars"
)

// InterfaceType Interface 类型
type InterfaceType string

const (
	InterfaceTypeHTTP  InterfaceType = "http"  // HTTP REST API
	InterfaceTypeA2A   InterfaceType = "a2a"   // Agent-to-Agent
	InterfaceTypeAGUI  InterfaceType = "agui"  // Agent GUI
	InterfaceTypeSlack InterfaceType = "slack" // Slack 集成
)

// Interface AsterOS 接口抽象
// Interface 定义了 AsterOS 与外部系统交互的标准接口
type Interface interface {
	// Name 返回 Interface 名称
	Name() string

	// Type 返回 Interface 类型
	Type() InterfaceType

	// Start 启动 Interface
	Start(ctx context.Context, os *AsterOS) error

	// Stop 停止 Interface
	Stop(ctx context.Context) error

	// OnAgentRegistered Agent 注册事件
	OnAgentRegistered(agent *agent.Agent) error

	// OnStarsRegistered Stars 注册事件
	OnStarsRegistered(stars *stars.Stars) error

	// OnWorkflowRegistered Workflow 注册事件
	OnWorkflowRegistered(wf workflow.Agent) error
}

// BaseInterface 基础 Interface 实现
// 提供默认的空实现，子类可以选择性覆盖
type BaseInterface struct {
	name string
	typ  InterfaceType
}

// NewBaseInterface 创建基础 Interface
func NewBaseInterface(name string, typ InterfaceType) *BaseInterface {
	return &BaseInterface{
		name: name,
		typ:  typ,
	}
}

// Name 返回 Interface 名称
func (i *BaseInterface) Name() string {
	return i.name
}

// Type 返回 Interface 类型
func (i *BaseInterface) Type() InterfaceType {
	return i.typ
}

// Start 启动 Interface（默认空实现）
func (i *BaseInterface) Start(ctx context.Context, os *AsterOS) error {
	return nil
}

// Stop 停止 Interface（默认空实现）
func (i *BaseInterface) Stop(ctx context.Context) error {
	return nil
}

// OnAgentRegistered Agent 注册事件（默认空实现）
func (i *BaseInterface) OnAgentRegistered(agent *agent.Agent) error {
	return nil
}

// OnStarsRegistered Stars 注册事件（默认空实现）
func (i *BaseInterface) OnStarsRegistered(stars *stars.Stars) error {
	return nil
}

// OnWorkflowRegistered Workflow 注册事件（默认空实现）
func (i *BaseInterface) OnWorkflowRegistered(wf workflow.Agent) error {
	return nil
}
