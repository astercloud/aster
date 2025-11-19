package stars

import (
	"context"
	"testing"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/cosmos"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/types"
)

// 创建测试用的 Dependencies
func createTestDeps(t *testing.T) *agent.Dependencies {
	memStore, err := store.NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	toolRegistry := tools.NewRegistry()
	templateRegistry := agent.NewTemplateRegistry()
	providerFactory := &provider.AnthropicFactory{}

	// 注册测试模板
	templateRegistry.Register(&types.AgentTemplateDefinition{
		ID:           "test-template",
		SystemPrompt: "You are a test assistant",
		Model:        "claude-sonnet-4-5",
		Tools:        []interface{}{},
	})

	return &agent.Dependencies{
		Store:            memStore,
		SandboxFactory:   sandbox.NewFactory(),
		ToolRegistry:     toolRegistry,
		ProviderFactory:  providerFactory,
		TemplateRegistry: templateRegistry,
	}
}

// 创建测试用的 AgentConfig
func createTestConfig(agentID string) *types.AgentConfig {
	return &types.AgentConfig{
		AgentID:    agentID,
		TemplateID: "test-template",
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

// TestStars_New 测试创建群星
func TestStars_New(t *testing.T) {
	deps := createTestDeps(t)
	cosmos := cosmos.New(&cosmos.Options{
		Dependencies: deps,
		MaxAgents:    10,
	})
	defer cosmos.Shutdown()

	stars := New(cosmos, "TestStars")

	if stars == nil {
		t.Fatal("Stars is nil")
	}

	if stars.Name() != "TestStars" {
		t.Errorf("Expected name 'TestStars', got '%s'", stars.Name())
	}

	if stars.Size() != 0 {
		t.Errorf("Expected size 0, got %d", stars.Size())
	}
}

// TestStars_Join 测试添加成员
func TestStars_Join(t *testing.T) {
	deps := createTestDeps(t)
	ctx := context.Background()

	cosmos := cosmos.New(&cosmos.Options{
		Dependencies: deps,
		MaxAgents:    10,
	})
	defer cosmos.Shutdown()

	// 创建 Agent
	config := createTestConfig("test-agent-1")
	_, err := cosmos.Create(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// 创建群星
	stars := New(cosmos, "TestStars")

	// 添加成员
	err = stars.Join("test-agent-1", RoleLeader)
	if err != nil {
		t.Fatalf("Failed to join: %v", err)
	}

	// 验证成员数量
	if stars.Size() != 1 {
		t.Errorf("Expected size 1, got %d", stars.Size())
	}

	// 验证成员信息
	members := stars.Members()
	if len(members) != 1 {
		t.Fatalf("Expected 1 member, got %d", len(members))
	}

	if members[0].AgentID != "test-agent-1" {
		t.Errorf("Expected AgentID 'test-agent-1', got '%s'", members[0].AgentID)
	}

	if members[0].Role != RoleLeader {
		t.Errorf("Expected role Leader, got %s", members[0].Role)
	}
}

// TestStars_JoinNonExistentAgent 测试添加不存在的 Agent
func TestStars_JoinNonExistentAgent(t *testing.T) {
	deps := createTestDeps(t)
	cosmos := cosmos.New(&cosmos.Options{
		Dependencies: deps,
		MaxAgents:    10,
	})
	defer cosmos.Shutdown()

	stars := New(cosmos, "TestStars")

	// 尝试添加不存在的 Agent
	err := stars.Join("non-existent-agent", RoleWorker)
	if err == nil {
		t.Error("Expected error when joining non-existent agent")
	}
}

// TestStars_JoinDuplicate 测试重复添加成员
func TestStars_JoinDuplicate(t *testing.T) {
	deps := createTestDeps(t)
	ctx := context.Background()

	cosmos := cosmos.New(&cosmos.Options{
		Dependencies: deps,
		MaxAgents:    10,
	})
	defer cosmos.Shutdown()

	// 创建 Agent
	config := createTestConfig("test-agent-1")
	_, err := cosmos.Create(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	stars := New(cosmos, "TestStars")

	// 第一次添加
	err = stars.Join("test-agent-1", RoleLeader)
	if err != nil {
		t.Fatalf("First join failed: %v", err)
	}

	// 第二次添加应该失败
	err = stars.Join("test-agent-1", RoleWorker)
	if err == nil {
		t.Error("Expected error when joining duplicate agent")
	}
}

// TestStars_Leave 测试移除成员
func TestStars_Leave(t *testing.T) {
	deps := createTestDeps(t)
	ctx := context.Background()

	cosmos := cosmos.New(&cosmos.Options{
		Dependencies: deps,
		MaxAgents:    10,
	})
	defer cosmos.Shutdown()

	// 创建 Agent
	config := createTestConfig("test-agent-1")
	_, err := cosmos.Create(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	stars := New(cosmos, "TestStars")

	// 添加成员
	err = stars.Join("test-agent-1", RoleLeader)
	if err != nil {
		t.Fatalf("Failed to join: %v", err)
	}

	// 验证成员存在
	if stars.Size() != 1 {
		t.Error("Agent not in stars")
	}

	// 移除成员
	err = stars.Leave("test-agent-1")
	if err != nil {
		t.Fatalf("Failed to leave: %v", err)
	}

	// 验证成员已移除
	if stars.Size() != 0 {
		t.Error("Agent still in stars after leaving")
	}
}

// TestStars_MultipleMembers 测试多个成员
func TestStars_MultipleMembers(t *testing.T) {
	deps := createTestDeps(t)
	ctx := context.Background()

	cosmos := cosmos.New(&cosmos.Options{
		Dependencies: deps,
		MaxAgents:    10,
	})
	defer cosmos.Shutdown()

	// 创建多个 Agent
	leaderConfig := createTestConfig("leader-1")
	_, err := cosmos.Create(ctx, leaderConfig)
	if err != nil {
		t.Fatalf("Failed to create leader: %v", err)
	}

	worker1Config := createTestConfig("worker-1")
	_, err = cosmos.Create(ctx, worker1Config)
	if err != nil {
		t.Fatalf("Failed to create worker1: %v", err)
	}

	worker2Config := createTestConfig("worker-2")
	_, err = cosmos.Create(ctx, worker2Config)
	if err != nil {
		t.Fatalf("Failed to create worker2: %v", err)
	}

	// 创建群星
	stars := New(cosmos, "TestStars")

	// 添加成员
	stars.Join("leader-1", RoleLeader)
	stars.Join("worker-1", RoleWorker)
	stars.Join("worker-2", RoleWorker)

	// 验证成员数量
	if stars.Size() != 3 {
		t.Errorf("Expected size 3, got %d", stars.Size())
	}

	// 验证成员角色
	members := stars.Members()
	leaderCount := 0
	workerCount := 0

	for _, m := range members {
		if m.Role == RoleLeader {
			leaderCount++
		} else if m.Role == RoleWorker {
			workerCount++
		}
	}

	if leaderCount != 1 {
		t.Errorf("Expected 1 leader, got %d", leaderCount)
	}

	if workerCount != 2 {
		t.Errorf("Expected 2 workers, got %d", workerCount)
	}
}

// TestStars_History 测试消息历史
func TestStars_History(t *testing.T) {
	deps := createTestDeps(t)
	ctx := context.Background()

	cosmos := cosmos.New(&cosmos.Options{
		Dependencies: deps,
		MaxAgents:    10,
	})
	defer cosmos.Shutdown()

	// 创建 Agents
	config1 := createTestConfig("agent-1")
	_, err := cosmos.Create(ctx, config1)
	if err != nil {
		t.Fatalf("Failed to create agent1: %v", err)
	}

	config2 := createTestConfig("agent-2")
	_, err = cosmos.Create(ctx, config2)
	if err != nil {
		t.Fatalf("Failed to create agent2: %v", err)
	}

	stars := New(cosmos, "TestStars")
	stars.Join("agent-1", RoleLeader)
	stars.Join("agent-2", RoleWorker)

	// 发送消息
	err = stars.Send(ctx, "agent-1", "agent-2", "Hello")
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// 检查历史
	history := stars.History()
	if len(history) != 1 {
		t.Errorf("Expected 1 message in history, got %d", len(history))
	}

	if len(history) > 0 {
		msg := history[0]
		if msg.From != "agent-1" {
			t.Errorf("Expected from 'agent-1', got '%s'", msg.From)
		}
		if msg.To != "agent-2" {
			t.Errorf("Expected to 'agent-2', got '%s'", msg.To)
		}
		if msg.Text != "Hello" {
			t.Errorf("Expected text 'Hello', got '%s'", msg.Text)
		}
	}
}
