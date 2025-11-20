package asteros

import (
	"context"
	"testing"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/agent/workflow"
	"github.com/astercloud/aster/pkg/core"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/types"
)

// 创建测试依赖
func createTestDependencies(t *testing.T) *agent.Dependencies {
	jsonStore, err := store.NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	toolRegistry := tools.NewRegistry()
	templateRegistry := agent.NewTemplateRegistry()
	providerFactory := &provider.AnthropicFactory{}

	templateRegistry.Register(&types.AgentTemplateDefinition{
		ID:           "test-agent",
		SystemPrompt: "You are a test agent",
		Model:        "claude-sonnet-4-5",
		Tools:        []interface{}{},
	})

	return &agent.Dependencies{
		Store:            jsonStore,
		SandboxFactory:   sandbox.NewFactory(),
		ToolRegistry:     toolRegistry,
		ProviderFactory:  providerFactory,
		TemplateRegistry: templateRegistry,
	}
}

// 创建测试 Agent 配置
func createTestAgentConfig(agentID string) *types.AgentConfig {
	return &types.AgentConfig{
		AgentID:    agentID,
		TemplateID: "test-agent",
		ModelConfig: &types.ModelConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-5",
			APIKey:   "sk-test-key-for-unit-tests",
		},
		Sandbox: &types.SandboxConfig{
			Kind: types.SandboxKindMock,
		},
	}
}

// TestNew 测试创建 AsterOS
func TestNew(t *testing.T) {
	deps := createTestDependencies(t)
	pool := core.NewPool(&core.PoolOptions{
		Dependencies: deps,
		MaxAgents:    5,
	})

	// 默认配置需要提供 pool
	defaultOpts := DefaultOptions()
	defaultOpts.Pool = pool
	os, err := New(defaultOpts)
	if err != nil {
		t.Fatalf("Failed to create AsterOS with default options: %v", err)
	}

	if os == nil {
		t.Fatal("AsterOS is nil")
	}

	// 自定义配置
	os, err = New(&Options{
		Name: "TestOS",
		Port: 8081,
		Pool: pool,
	})
	if err != nil {
		t.Fatalf("Failed to create AsterOS: %v", err)
	}

	if os.Name() != "TestOS" {
		t.Errorf("Expected name 'TestOS', got '%s'", os.Name())
	}
}

// TestNewValidation 测试配置验证
func TestNewValidation(t *testing.T) {
	// 无 Pool 配置
	_, err := New(&Options{
		Name: "TestOS",
		Port: 8080,
	})
	if err == nil {
		t.Error("Expected error for missing pool")
	}

	// 无效端口
	deps := createTestDependencies(t)
	pool := core.NewPool(&core.PoolOptions{
		Dependencies: deps,
		MaxAgents:    5,
	})

	_, err = New(&Options{
		Name: "TestOS",
		Port: -1, // 无效端口
		Pool: pool,
	})
	if err == nil {
		t.Error("Expected error for invalid port")
	}
}

// TestRegisterAgent 测试注册 Agent
func TestRegisterAgent(t *testing.T) {
	ctx := context.Background()
	deps := createTestDependencies(t)
	pool := core.NewPool(&core.PoolOptions{
		Dependencies: deps,
		MaxAgents:    5,
	})

	os, err := New(&Options{
		Name: "TestOS",
		Port: 8080,
		Pool: pool,
	})
	if err != nil {
		t.Fatalf("Failed to create AsterOS: %v", err)
	}

	// 创建 Agent
	agentConfig := createTestAgentConfig("test-agent-1")
	ag, err := agent.Create(ctx, agentConfig, deps)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// 注册 Agent
	err = os.RegisterAgent("test-agent-1", ag)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	// 验证 Agent 已注册
	_, exists := os.Registry().GetAgent("test-agent-1")
	if !exists {
		t.Error("Agent not found in registry")
	}

	// 尝试重复注册
	err = os.RegisterAgent("test-agent-1", ag)
	if err == nil {
		t.Error("Expected error for duplicate agent registration")
	}
}

// TestRegisterRoom 测试注册 Room
func TestRegisterRoom(t *testing.T) {
	deps := createTestDependencies(t)
	pool := core.NewPool(&core.PoolOptions{
		Dependencies: deps,
		MaxAgents:    5,
	})

	os, err := New(&Options{
		Name: "TestOS",
		Port: 8080,
		Pool: pool,
	})
	if err != nil {
		t.Fatalf("Failed to create AsterOS: %v", err)
	}

	// 创建 Room
	room := core.NewRoom(pool)

	// 注册 Room
	err = os.RegisterRoom("test-room", room)
	if err != nil {
		t.Fatalf("Failed to register room: %v", err)
	}

	// 验证 Room 已注册
	_, exists := os.Registry().GetRoom("test-room")
	if !exists {
		t.Error("Room not found in registry")
	}
}

// TestRegisterWorkflow 测试注册 Workflow
func TestRegisterWorkflow(t *testing.T) {
	deps := createTestDependencies(t)
	pool := core.NewPool(&core.PoolOptions{
		Dependencies: deps,
		MaxAgents:    5,
	})

	os, err := New(&Options{
		Name: "TestOS",
		Port: 8080,
		Pool: pool,
	})
	if err != nil {
		t.Fatalf("Failed to create AsterOS: %v", err)
	}

	// 创建 Workflow
	workflowInstance := &workflow.SequentialAgent{}

	// 注册 Workflow
	err = os.RegisterWorkflow("test-workflow", workflowInstance)
	if err != nil {
		t.Fatalf("Failed to register workflow: %v", err)
	}

	// 验证 Workflow 已注册
	_, exists := os.Registry().GetWorkflow("test-workflow")
	if !exists {
		t.Error("Workflow not found in registry")
	}
}

// TestAddInterface 测试添加 Interface
func TestAddInterface(t *testing.T) {
	deps := createTestDependencies(t)
	pool := core.NewPool(&core.PoolOptions{
		Dependencies: deps,
		MaxAgents:    5,
	})

	os, err := New(&Options{
		Name: "TestOS",
		Port: 8080,
		Pool: pool,
	})
	if err != nil {
		t.Fatalf("Failed to create AsterOS: %v", err)
	}

	// 创建测试 Interface
	testInterface := &TestInterface{
		name: "test-interface",
		typ:  "test",
	}

	// 添加 Interface
	err = os.AddInterface(testInterface)
	if err != nil {
		t.Fatalf("Failed to add interface: %v", err)
	}

	// 重复添加应该失败
	err = os.AddInterface(testInterface)
	if err == nil {
		t.Error("Expected error for duplicate interface")
	}

	// 移除 Interface
	err = os.RemoveInterface("test-interface")
	if err != nil {
		t.Fatalf("Failed to remove interface: %v", err)
	}

	// 移除不存在的 Interface 应该失败
	err = os.RemoveInterface("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent interface")
	}
}

// TestLifecycle 测试生命周期
func TestLifecycle(t *testing.T) {
	deps := createTestDependencies(t)
	pool := core.NewPool(&core.PoolOptions{
		Dependencies: deps,
		MaxAgents:    5,
	})

	os, err := New(&Options{
		Name: "TestOS",
		Port: 8080,
		Pool: pool,
	})
	if err != nil {
		t.Fatalf("Failed to create AsterOS: %v", err)
	}

	// 初始状态应该不是运行中
	if os.IsRunning() {
		t.Error("AsterOS should not be running initially")
	}

	// 关闭未运行的 AsterOS 应该失败
	err = os.Shutdown()
	if err == nil {
		t.Error("Expected error for shutting down non-running AsterOS")
	}
}

// TestGetters 测试 Getter 方法
func TestGetters(t *testing.T) {
	deps := createTestDependencies(t)
	pool := core.NewPool(&core.PoolOptions{
		Dependencies: deps,
		MaxAgents:    5,
	})

	os, err := New(&Options{
		Name: "TestOS",
		Port: 8080,
		Pool: pool,
	})
	if err != nil {
		t.Fatalf("Failed to create AsterOS: %v", err)
	}

	// 测试 Getter 方法
	if os.Pool() == nil {
		t.Error("Pool should not be nil")
	}

	if os.Registry() == nil {
		t.Error("Registry should not be nil")
	}

	if os.Router() == nil {
		t.Error("Router should not be nil")
	}
}

// TestInterfaceAbstract 抽象测试
func TestInterfaceAbstract(t *testing.T) {
	baseIface := NewBaseInterface("test", "http")
	if baseIface.Name() != "test" {
		t.Errorf("Expected name 'test', got '%s'", baseIface.Name())
	}

	if baseIface.Type() != "http" {
		t.Errorf("Expected type 'http', got '%s'", baseIface.Type())
	}

	// 测试默认方法（应该返回 nil）
	ctx := context.Background()
	os := &AsterOS{}
	if err := baseIface.Start(ctx, os); err != nil {
		t.Error("BaseInterface Start should not return error")
	}

	if err := baseIface.Stop(ctx); err != nil {
		t.Error("BaseInterface Stop should not return error")
	}

	if err := baseIface.OnAgentRegistered(nil); err != nil {
		t.Error("BaseInterface OnAgentRegistered should not return error")
	}
}

// 测试用的简单 Interface 实现
type TestInterface struct {
	name string
	typ  string
}

func (i *TestInterface) Name() string {
	return i.name
}

func (i *TestInterface) Type() InterfaceType {
	return InterfaceType(i.typ)
}

func (i *TestInterface) Start(ctx context.Context, os *AsterOS) error {
	return nil
}

func (i *TestInterface) Stop(ctx context.Context) error {
	return nil
}

func (i *TestInterface) OnAgentRegistered(agent *agent.Agent) error {
	return nil
}

func (i *TestInterface) OnRoomRegistered(room *core.Room) error {
	return nil
}

func (i *TestInterface) OnWorkflowRegistered(wf workflow.Agent) error {
	return nil
}

// TestDefaultOptions 测试默认配置
func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.Name != "AsterOS" {
		t.Errorf("Expected default name 'AsterOS', got '%s'", opts.Name)
	}

	if opts.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", opts.Port)
	}

	if !opts.AutoDiscover {
		t.Error("Expected auto discovery to be enabled by default")
	}

	if !opts.EnableCORS {
		t.Error("Expected CORS to be enabled by default")
	}

	if opts.EnableAuth {
		t.Error("Expected auth to be disabled by default")
	}

	if !opts.EnableMetrics {
		t.Error("Expected metrics to be enabled by default")
	}

	if !opts.EnableHealth {
		t.Error("Expected health check to be enabled by default")
	}

	if !opts.EnableLogging {
		t.Error("Expected logging to be enabled by default")
	}

	if opts.LogLevel != "info" {
		t.Errorf("Expected default log level 'info', got '%s'", opts.LogLevel)
	}
}

// TestOptionsValidation 测试配置验证
func TestOptionsValidation(t *testing.T) {
	// 测试 nil Pool
	opts := &Options{}
	err := opts.Validate()
	if err == nil {
		t.Error("Expected error for nil pool")
	}

	// 测试无效端口
	deps := createTestDependencies(t)
	pool := core.NewPool(&core.PoolOptions{
		Dependencies: deps,
		MaxAgents:    5,
	})

	opts = &Options{
		Name: "TestOS",
		Port: 70000, // 无效端口
		Pool: pool,
	}
	err = opts.Validate()
	if err == nil {
		t.Error("Expected error for invalid port")
	}

	// 测试有效配置
	opts = &Options{
		Name: "TestOS",
		Port: 8080,
		Pool: pool,
	}
	err = opts.Validate()
	if err != nil {
		t.Errorf("Valid options should not return error: %v", err)
	}
}
