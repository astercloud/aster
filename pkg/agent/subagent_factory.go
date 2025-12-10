package agent

import (
	"context"

	"github.com/astercloud/aster/pkg/tools/builtin"
	"github.com/astercloud/aster/pkg/types"
)

// SubAgentManagerFactory 将 SubAgentManager 适配为 SubAgentExecutorFactory
// 这个适配器允许 Task 工具使用真正的子 Agent 执行
type SubAgentManagerFactory struct {
	manager *SubAgentManager
}

// NewSubAgentManagerFactory 创建工厂适配器
func NewSubAgentManagerFactory(manager *SubAgentManager) *SubAgentManagerFactory {
	return &SubAgentManagerFactory{
		manager: manager,
	}
}

// Create 创建指定类型的子 Agent 执行器
func (f *SubAgentManagerFactory) Create(agentType string) (types.SubAgentExecutor, error) {
	return NewSubAgentExecutor(f.manager, agentType)
}

// ListTypes 列出支持的子 Agent 类型
func (f *SubAgentManagerFactory) ListTypes() []string {
	specs := f.manager.ListSpecs()
	types := make([]string, len(specs))
	for i, spec := range specs {
		types[i] = spec.Name
	}
	return types
}

// SubAgentExecutorWrapper 包装 SubAgentManager 为单个执行器
type SubAgentExecutorWrapper struct {
	manager   *SubAgentManager
	agentType string
	spec      *types.SubAgentSpec
}

// NewSubAgentExecutorWrapper 创建执行器包装
func NewSubAgentExecutorWrapper(manager *SubAgentManager, agentType string) (*SubAgentExecutorWrapper, error) {
	spec, err := manager.GetSpec(agentType)
	if err != nil {
		return nil, err
	}

	return &SubAgentExecutorWrapper{
		manager:   manager,
		agentType: agentType,
		spec:      spec,
	}, nil
}

// GetSpec 获取子 Agent 规格
func (w *SubAgentExecutorWrapper) GetSpec() *types.SubAgentSpec {
	return w.spec
}

// Execute 执行子 Agent 任务
func (w *SubAgentExecutorWrapper) Execute(ctx context.Context, req *types.SubAgentRequest) (*types.SubAgentResult, error) {
	// 确保请求使用正确的 agent type
	req.AgentType = w.agentType
	return w.manager.Execute(ctx, req)
}

// InitializeTaskExecutor 初始化 Task 执行器
// 应用层在启动时调用此函数，将 SubAgentManager 注入到 Task 工具
func InitializeTaskExecutor(deps *Dependencies) *SubAgentManager {
	// 创建 SubAgentManager
	manager := NewSubAgentManager(deps)

	// 创建工厂适配器
	factory := NewSubAgentManagerFactory(manager)

	// 直接注入到 builtin.TaskExecutor
	builtin.SetFactoryFromAgent(factory)

	return manager
}
