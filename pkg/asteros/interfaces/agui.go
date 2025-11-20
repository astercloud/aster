package interfaces

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/agent/workflow"
	"github.com/astercloud/aster/pkg/asteros"
	"github.com/astercloud/aster/pkg/core"
)

// AGUIInterfaceOptions AGUI Interface 配置
type AGUIInterfaceOptions struct {
	// ControlPlaneURL 控制平面 URL
	ControlPlaneURL string

	// APIKey API Key
	APIKey string

	// EnableLogging 是否启用日志
	EnableLogging bool

	// SyncInterval 同步间隔
	SyncInterval time.Duration

	// EnableAutoSync 是否启用自动同步
	EnableAutoSync bool
}

// AGUIInterface Agent GUI Interface
// AGUIInterface 连接到 Aster 控制平面 UI，
// 提供可视化的 Agent 管理和监控界面。
type AGUIInterface struct {
	*asteros.BaseInterface
	opts *AGUIInterfaceOptions
	os   *asteros.AsterOS

	// 连接状态
	connected bool
	mu        sync.RWMutex

	// 取消函数
	cancel context.CancelFunc
}

// NewAGUIInterface 创建 AGUI Interface
func NewAGUIInterface(opts *AGUIInterfaceOptions) *AGUIInterface {
	if opts == nil {
		opts = &AGUIInterfaceOptions{
			ControlPlaneURL: "https://os.aster.com",
			EnableLogging:   true,
			SyncInterval:    30 * time.Second,
			EnableAutoSync:  true,
		}
	}

	return &AGUIInterface{
		BaseInterface: asteros.NewBaseInterface("agui", asteros.InterfaceTypeAGUI),
		opts:          opts,
		connected:     false,
	}
}

// Start 启动 AGUI Interface
func (i *AGUIInterface) Start(ctx context.Context, os *asteros.AsterOS) error {
	i.os = os

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(ctx)
	i.cancel = cancel

	// 连接到控制平面
	if err := i.connect(ctx); err != nil {
		return fmt.Errorf("connect to control plane: %w", err)
	}

	// 启动自动同步
	if i.opts.EnableAutoSync {
		go i.syncLoop(ctx)
	}

	if i.opts.EnableLogging {
		fmt.Printf("✓ AGUI Interface started\n")
		fmt.Printf("  → Control Plane: %s\n", i.opts.ControlPlaneURL)
	}

	return nil
}

// Stop 停止 AGUI Interface
func (i *AGUIInterface) Stop(ctx context.Context) error {
	// 取消同步循环
	if i.cancel != nil {
		i.cancel()
	}

	// 断开连接
	if err := i.disconnect(ctx); err != nil {
		return fmt.Errorf("disconnect from control plane: %w", err)
	}

	if i.opts.EnableLogging {
		fmt.Printf("✓ AGUI Interface stopped\n")
	}

	return nil
}

// OnAgentRegistered Agent 注册事件
func (i *AGUIInterface) OnAgentRegistered(ag *agent.Agent) error {
	if i.opts.EnableLogging {
		fmt.Printf("  [AGUI] Agent registered: %s\n", ag.ID())
		fmt.Printf("    → Syncing to Control Plane\n")
	}

	// 同步到控制平面
	return i.syncAgent(ag)
}

// OnRoomRegistered Room 注册事件
func (i *AGUIInterface) OnRoomRegistered(r *core.Room) error {
	if i.opts.EnableLogging {
		fmt.Printf("  [AGUI] Room registered with %d members\n", r.GetMemberCount())
		fmt.Printf("    → Syncing to Control Plane\n")
	}

	// 同步到控制平面
	return i.syncRoom(r)
}

// OnWorkflowRegistered Workflow 注册事件
func (i *AGUIInterface) OnWorkflowRegistered(wf workflow.Agent) error {
	if i.opts.EnableLogging {
		fmt.Printf("  [AGUI] Workflow registered: %s\n", wf.Name())
		fmt.Printf("    → Syncing to Control Plane\n")
	}

	// 同步到控制平面
	return i.syncWorkflow(wf)
}

// connect 连接到控制平面
func (i *AGUIInterface) connect(ctx context.Context) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	// TODO: 实现实际的连接逻辑
	// 1. 建立 WebSocket 连接
	// 2. 认证
	// 3. 注册 AsterOS 实例

	i.connected = true

	if i.opts.EnableLogging {
		fmt.Printf("  [AGUI] Connected to Control Plane\n")
	}

	return nil
}

// disconnect 断开连接
func (i *AGUIInterface) disconnect(ctx context.Context) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	// TODO: 实现实际的断开逻辑
	// 1. 注销 AsterOS 实例
	// 2. 关闭 WebSocket 连接

	i.connected = false

	if i.opts.EnableLogging {
		fmt.Printf("  [AGUI] Disconnected from Control Plane\n")
	}

	return nil
}

// syncLoop 同步循环
func (i *AGUIInterface) syncLoop(ctx context.Context) {
	ticker := time.NewTicker(i.opts.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := i.syncAll(); err != nil {
				if i.opts.EnableLogging {
					fmt.Printf("  [AGUI] Sync error: %v\n", err)
				}
			}
		}
	}
}

// syncAll 同步所有资源
func (i *AGUIInterface) syncAll() error {
	if !i.isConnected() {
		return fmt.Errorf("not connected to control plane")
	}

	// TODO: 实现实际的同步逻辑
	// 1. 获取所有 Agents、Stars、Workflows
	// 2. 同步状态到控制平面

	if i.opts.EnableLogging {
		fmt.Printf("  [AGUI] Synced all resources to Control Plane\n")
	}

	return nil
}

// syncAgent 同步 Agent
func (i *AGUIInterface) syncAgent(ag *agent.Agent) error {
	if !i.isConnected() {
		return nil // 静默失败
	}

	// TODO: 实现实际的同步逻辑
	// 发送 Agent 信息到控制平面

	return nil
}

// syncRoom 同步 Room
func (i *AGUIInterface) syncRoom(r *core.Room) error {
	if !i.isConnected() {
		return nil // 静默失败
	}

	// TODO: 实现实际的同步逻辑
	// 发送 Room 信息到控制平面

	return nil
}

// syncWorkflow 同步 Workflow
func (i *AGUIInterface) syncWorkflow(wf workflow.Agent) error {
	if !i.isConnected() {
		return nil // 静默失败
	}

	// TODO: 实现实际的同步逻辑
	// 发送 Workflow 信息到控制平面

	return nil
}

// isConnected 检查是否已连接
func (i *AGUIInterface) isConnected() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.connected
}

// GetControlPlaneURL 获取控制平面 URL
func (i *AGUIInterface) GetControlPlaneURL() string {
	return i.opts.ControlPlaneURL
}

// IsConnected 检查是否已连接（公开方法）
func (i *AGUIInterface) IsConnected() bool {
	return i.isConnected()
}
