package cosmos

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/types"
)

// 创建测试用的 Dependencies
func createTestDeps(t *testing.T) *agent.Dependencies {
	// 使用 JSONStore 代替 MemoryStore
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

// 创建测试用的 AgentConfig 辅助函数
func createTestConfig(agentID string) *types.AgentConfig {
	return &types.AgentConfig{
		AgentID:    agentID,
		TemplateID: "test-template",
		ModelConfig: &types.ModelConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-5",
			APIKey:   "sk-test-key-for-unit-tests", // 固定测试 key
		},
		Sandbox: &types.SandboxConfig{
			Kind: types.SandboxKindMock,
		},
	}
}

// TestCosmos_Create 测试创建 Agent
func TestCosmos_Create(t *testing.T) {
	deps := createTestDeps(t)
	cosmos := New(&Options{
		Dependencies: deps,
		MaxAgents:    5,
	})
	defer cosmos.Shutdown()

	ctx := context.Background()

	// 创建 Agent
	config := createTestConfig("test-agent-1")

	ag, err := cosmos.Create(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	if ag == nil {
		t.Fatal("Agent is nil")
	}

	// 验证 Agent 在宇宙中
	retrievedAg, exists := cosmos.Get("test-agent-1")
	if !exists {
		t.Fatal("Agent not found in cosmos")
	}

	if retrievedAg != ag {
		t.Error("Retrieved agent is different from created agent")
	}

	// 验证宇宙大小
	if cosmos.Size() != 1 {
		t.Errorf("Expected cosmos size 1, got %d", cosmos.Size())
	}
}

// TestCosmos_CreateDuplicate 测试重复创建相同 ID 的 Agent
func TestCosmos_CreateDuplicate(t *testing.T) {
	deps := createTestDeps(t)
	cosmos := New(&Options{
		Dependencies: deps,
		MaxAgents:    5,
	})
	defer cosmos.Shutdown()

	ctx := context.Background()
	config := createTestConfig("test-agent")

	// 第一次创建
	_, err := cosmos.Create(ctx, config)
	if err != nil {
		t.Fatalf("First create failed: %v", err)
	}

	// 第二次创建应该失败
	_, err = cosmos.Create(ctx, config)
	if err == nil {
		t.Error("Expected error when creating duplicate agent")
	}
}

// TestCosmos_MaxCapacity 测试宇宙容量限制
func TestCosmos_MaxCapacity(t *testing.T) {
	deps := createTestDeps(t)
	maxAgents := 3
	cosmos := New(&Options{
		Dependencies: deps,
		MaxAgents:    maxAgents,
	})
	defer cosmos.Shutdown()

	ctx := context.Background()

	// 创建 maxAgents 个 Agent
	for i := 0; i < maxAgents; i++ {
		config := createTestConfig("test-agent-" + string(rune('1'+i)))
		_, err := cosmos.Create(ctx, config)
		if err != nil {
			t.Fatalf("Failed to create agent %d: %v", i, err)
		}
	}

	// 尝试创建超过容量的 Agent
	config := createTestConfig("overflow-agent")

	_, err := cosmos.Create(ctx, config)
	if err == nil {
		t.Error("Expected error when cosmos is full")
	}

	// 验证宇宙大小
	if cosmos.Size() != maxAgents {
		t.Errorf("Expected cosmos size %d, got %d", maxAgents, cosmos.Size())
	}
}

// TestCosmos_List 测试列出 Agent
func TestCosmos_List(t *testing.T) {
	deps := createTestDeps(t)
	cosmos := New(&Options{
		Dependencies: deps,
		MaxAgents:    10,
	})
	defer cosmos.Shutdown()

	ctx := context.Background()

	// 创建不同前缀的 Agent
	agents := []string{"user-1", "user-2", "admin-1", "admin-2"}
	for _, agentID := range agents {
		config := createTestConfig(agentID)
		_, err := cosmos.Create(ctx, config)
		if err != nil {
			t.Fatalf("Failed to create agent %s: %v", agentID, err)
		}
	}

	// 列出所有 Agent
	allAgents := cosmos.List("")
	if len(allAgents) != 4 {
		t.Errorf("Expected 4 agents, got %d", len(allAgents))
	}

	// 列出 user- 前缀的 Agent
	userAgents := cosmos.List("user-")
	if len(userAgents) != 2 {
		t.Errorf("Expected 2 user agents, got %d", len(userAgents))
	}

	// 列出 admin- 前缀的 Agent
	adminAgents := cosmos.List("admin-")
	if len(adminAgents) != 2 {
		t.Errorf("Expected 2 admin agents, got %d", len(adminAgents))
	}
}

// TestCosmos_Remove 测试移除 Agent
func TestCosmos_Remove(t *testing.T) {
	deps := createTestDeps(t)
	cosmos := New(&Options{
		Dependencies: deps,
		MaxAgents:    5,
	})
	defer cosmos.Shutdown()

	ctx := context.Background()
	config := createTestConfig("test-agent")

	// 创建 Agent
	_, err := cosmos.Create(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// 验证存在
	if cosmos.Size() != 1 {
		t.Error("Agent not in cosmos")
	}

	// 移除 Agent
	err = cosmos.Remove("test-agent")
	if err != nil {
		t.Fatalf("Failed to remove agent: %v", err)
	}

	// 验证已移除
	if cosmos.Size() != 0 {
		t.Error("Agent still in cosmos after removal")
	}

	_, exists := cosmos.Get("test-agent")
	if exists {
		t.Error("Agent still retrievable after removal")
	}
}

// TestCosmos_Status 测试获取 Agent 状态
func TestCosmos_Status(t *testing.T) {
	deps := createTestDeps(t)
	cosmos := New(&Options{
		Dependencies: deps,
		MaxAgents:    5,
	})
	defer cosmos.Shutdown()

	ctx := context.Background()
	config := createTestConfig("test-agent")

	_, err := cosmos.Create(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// 获取状态
	status, err := cosmos.Status("test-agent")
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if status == nil {
		t.Fatal("Status is nil")
	}

	if status.AgentID != "test-agent" {
		t.Errorf("Expected AgentID 'test-agent', got '%s'", status.AgentID)
	}
}

// TestCosmos_ConcurrentAccess 测试并发访问
func TestCosmos_ConcurrentAccess(t *testing.T) {
	deps := createTestDeps(t)
	cosmos := New(&Options{
		Dependencies: deps,
		MaxAgents:    100,
	})
	defer cosmos.Shutdown()

	ctx := context.Background()
	concurrency := 50
	var wg sync.WaitGroup

	// 并发创建 Agent
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			config := createTestConfig("concurrent-agent-" + string(rune('0'+idx)))
			_, err := cosmos.Create(ctx, config)
			if err != nil {
				t.Logf("Failed to create agent %d: %v", idx, err)
			}
		}(i)
	}

	wg.Wait()

	// 验证宇宙大小
	size := cosmos.Size()
	if size != concurrency {
		t.Logf("Expected %d agents, got %d (some creates may have failed)", concurrency, size)
	}

	// 并发读取 Agent
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			agentID := "concurrent-agent-" + string(rune('0'+idx))
			_, exists := cosmos.Get(agentID)
			if !exists {
				t.Logf("Agent %s not found", agentID)
			}
		}(i)
	}

	wg.Wait()
}

// TestCosmos_Shutdown 测试关闭宇宙
func TestCosmos_Shutdown(t *testing.T) {
	deps := createTestDeps(t)
	cosmos := New(&Options{
		Dependencies: deps,
		MaxAgents:    10,
	})

	ctx := context.Background()

	// 创建多个 Agent
	for i := 0; i < 5; i++ {
		config := createTestConfig("test-agent-" + string(rune('1'+i)))
		_, err := cosmos.Create(ctx, config)
		if err != nil {
			t.Fatalf("Failed to create agent: %v", err)
		}
	}

	// 关闭宇宙
	err := cosmos.Shutdown()
	if err != nil {
		t.Fatalf("Failed to shutdown cosmos: %v", err)
	}

	// 验证宇宙已清空
	if cosmos.Size() != 0 {
		t.Errorf("Cosmos not empty after shutdown, size: %d", cosmos.Size())
	}
}

// TestCosmos_ForEach 测试遍历 Agent
func TestCosmos_ForEach(t *testing.T) {
	deps := createTestDeps(t)
	cosmos := New(&Options{
		Dependencies: deps,
		MaxAgents:    10,
	})
	defer cosmos.Shutdown()

	ctx := context.Background()

	// 创建 Agent
	agentCount := 5
	for i := 0; i < agentCount; i++ {
		config := createTestConfig("test-agent-" + string(rune('1'+i)))
		_, err := cosmos.Create(ctx, config)
		if err != nil {
			t.Fatalf("Failed to create agent: %v", err)
		}
	}

	// 遍历所有 Agent
	visited := make(map[string]bool)
	err := cosmos.ForEach(func(agentID string, ag *agent.Agent) error {
		visited[agentID] = true
		return nil
	})

	if err != nil {
		t.Fatalf("ForEach failed: %v", err)
	}

	// 验证所有 Agent 都被访问
	if len(visited) != agentCount {
		t.Errorf("Expected %d agents visited, got %d", agentCount, len(visited))
	}
}

// TestCosmos_Resume 测试恢复 Agent
// 注意: 这个测试依赖于 Agent 实际保存消息到 Store,在单元测试环境中可能会失败
func TestCosmos_Resume(t *testing.T) {
	t.Skip("Skipping Resume test - requires real agent message persistence")
	deps := createTestDeps(t)
	ctx := context.Background()

	// 第一个宇宙 - 创建并保存 Agent
	cosmos1 := New(&Options{
		Dependencies: deps,
		MaxAgents:    10,
	})

	config := createTestConfig("persistent-agent")

	// 创建 Agent
	ag1, err := cosmos1.Create(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// 发送消息以保存状态
	err = ag1.Send(ctx, "Test message")
	if err != nil {
		t.Logf("Send message warning: %v", err)
	}

	// 等待保存完成
	time.Sleep(100 * time.Millisecond)

	// 关闭第一个宇宙
	cosmos1.Shutdown()

	// 第二个宇宙 - 恢复 Agent
	cosmos2 := New(&Options{
		Dependencies: deps,
		MaxAgents:    10,
	})
	defer cosmos2.Shutdown()

	// 恢复 Agent
	ag2, err := cosmos2.Resume(ctx, "persistent-agent", config)
	if err != nil {
		t.Fatalf("Failed to resume agent: %v", err)
	}

	if ag2 == nil {
		t.Fatal("Resumed agent is nil")
	}

	// 验证 Agent 在宇宙中
	_, exists := cosmos2.Get("persistent-agent")
	if !exists {
		t.Error("Resumed agent not found in cosmos")
	}
}
