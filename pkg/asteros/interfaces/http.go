package interfaces

import (
	"context"
	"fmt"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/agent/workflow"
	"github.com/astercloud/aster/pkg/asteros"
	"github.com/astercloud/aster/pkg/stars"
)

// HTTPInterfaceOptions HTTP Interface 配置
type HTTPInterfaceOptions struct {
	// Port HTTP 服务端口（如果与 AsterOS 主端口不同）
	Port int

	// EnableLogging 是否启用日志
	EnableLogging bool

	// CustomRoutes 自定义路由
	CustomRoutes func(os *asteros.AsterOS)
}

// HTTPInterface HTTP REST API Interface
// HTTPInterface 是默认的 HTTP REST API 接口，
// 提供标准的 RESTful API 端点。
type HTTPInterface struct {
	*asteros.BaseInterface
	opts *HTTPInterfaceOptions
	os   *asteros.AsterOS
}

// NewHTTPInterface 创建 HTTP Interface
func NewHTTPInterface(opts *HTTPInterfaceOptions) *HTTPInterface {
	if opts == nil {
		opts = &HTTPInterfaceOptions{
			EnableLogging: true,
		}
	}

	return &HTTPInterface{
		BaseInterface: asteros.NewBaseInterface("http", asteros.InterfaceTypeHTTP),
		opts:          opts,
	}
}

// Start 启动 HTTP Interface
func (i *HTTPInterface) Start(ctx context.Context, os *asteros.AsterOS) error {
	i.os = os

	// 添加自定义路由
	if i.opts.CustomRoutes != nil {
		i.opts.CustomRoutes(os)
	}

	fmt.Printf("✓ HTTP Interface started\n")
	return nil
}

// Stop 停止 HTTP Interface
func (i *HTTPInterface) Stop(ctx context.Context) error {
	fmt.Printf("✓ HTTP Interface stopped\n")
	return nil
}

// OnAgentRegistered Agent 注册事件
func (i *HTTPInterface) OnAgentRegistered(agent *agent.Agent) error {
	if i.opts.EnableLogging {
		fmt.Printf("  [HTTP] Agent registered: %s\n", agent.ID())
		fmt.Printf("    → API: POST /agents/%s/run\n", agent.ID())
		fmt.Printf("    → API: GET  /agents/%s/status\n", agent.ID())
	}
	return nil
}

// OnStarsRegistered Stars 注册事件
func (i *HTTPInterface) OnStarsRegistered(s *stars.Stars) error {
	if i.opts.EnableLogging {
		fmt.Printf("  [HTTP] Stars registered: %s\n", s.Name())
		fmt.Printf("    → API: POST /stars/%s/run\n", s.ID())
		fmt.Printf("    → API: POST /stars/%s/join\n", s.ID())
		fmt.Printf("    → API: POST /stars/%s/leave\n", s.ID())
		fmt.Printf("    → API: GET  /stars/%s/members\n", s.ID())
	}
	return nil
}

// OnWorkflowRegistered Workflow 注册事件
func (i *HTTPInterface) OnWorkflowRegistered(wf workflow.Agent) error {
	if i.opts.EnableLogging {
		fmt.Printf("  [HTTP] Workflow registered: %s\n", wf.Name())
		fmt.Printf("    → API: POST /workflows/%s/execute\n", wf.Name())
	}
	return nil
}
