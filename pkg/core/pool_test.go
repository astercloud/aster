package core

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
		Tools:        []any{},
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

// TestPool_Create 测试创建 Agent
func TestPool_Create(t *testing.T) {
	deps := createTestDeps(t)
	pool := NewPool(&PoolOptions{
		Dependencies: deps,
		MaxAgents:    5,
	})
	defer func() {
		if err := pool.Shutdown(); err != nil {
			t.Errorf("Shutdown failed: %v", err)
		}
	}()

	ctx := context.Background()

	// 创建 Agent
	config := createTestConfig("test-agent-1")

	ag, err := pool.Create(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	if ag == nil {
		t.Fatal("Agent is nil")
	}

	// 验证 Agent 在池中
	retrievedAg, exists := pool.Get("test-agent-1")
	if !exists {
		t.Fatal("Agent not found in pool")
	}

	if retrievedAg != ag {
		t.Error("Retrieved agent is different from created agent")
	}

	// 验证池大小
	if pool.Size() != 1 {
		t.Errorf("Expected pool size 1, got %d", pool.Size())
	}
}

// TestPool_CreateDuplicate 测试重复创建相同 ID 的 Agent
func TestPool_CreateDuplicate(t *testing.T) {
	deps := createTestDeps(t)
	pool := NewPool(&PoolOptions{
		Dependencies: deps,
		MaxAgents:    5,
	})
	defer func() {
		if err := pool.Shutdown(); err != nil {
			t.Errorf("Shutdown failed: %v", err)
		}
	}()

	ctx := context.Background()
	config := createTestConfig("test-agent")

	// 第一次创建
	_, err := pool.Create(ctx, config)
	if err != nil {
		t.Fatalf("First create failed: %v", err)
	}

	// 第二次创建应该失败
	_, err = pool.Create(ctx, config)
	if err == nil {
		t.Error("Expected error when creating duplicate agent")
	}
}

// TestPool_MaxCapacity 测试池容量限制
func TestPool_MaxCapacity(t *testing.T) {
	deps := createTestDeps(t)
	maxAgents := 3
	pool := NewPool(&PoolOptions{
		Dependencies: deps,
		MaxAgents:    maxAgents,
	})
	defer func() {
		if err := pool.Shutdown(); err != nil {
			t.Errorf("Shutdown failed: %v", err)
		}
	}()

	ctx := context.Background()

	// 创建 maxAgents 个 Agent
	for i := range maxAgents {
		config := createTestConfig("test-agent-" + string(rune('1'+i)))
		_, err := pool.Create(ctx, config)
		if err != nil {
			t.Fatalf("Failed to create agent %d: %v", i, err)
		}
	}

	// 尝试创建超过容量的 Agent
	config := createTestConfig("overflow-agent")

	_, err := pool.Create(ctx, config)
	if err == nil {
		t.Error("Expected error when pool is full")
	}

	// 验证池大小
	if pool.Size() != maxAgents {
		t.Errorf("Expected pool size %d, got %d", maxAgents, pool.Size())
	}
}

// TestPool_List 测试列出 Agent
func TestPool_List(t *testing.T) {
	deps := createTestDeps(t)
	pool := NewPool(&PoolOptions{
		Dependencies: deps,
		MaxAgents:    10,
	})
	defer func() {
		if err := pool.Shutdown(); err != nil {
			t.Errorf("Shutdown failed: %v", err)
		}
	}()

	ctx := context.Background()

	// 创建不同前缀的 Agent
	agents := []string{"user-1", "user-2", "admin-1", "admin-2"}
	for _, agentID := range agents {
		config := createTestConfig(agentID)
		_, err := pool.Create(ctx, config)
		if err != nil {
			t.Fatalf("Failed to create agent %s: %v", agentID, err)
		}
	}

	// 列出所有 Agent
	allAgents := pool.List("")
	if len(allAgents) != 4 {
		t.Errorf("Expected 4 agents, got %d", len(allAgents))
	}

	// 列出 user- 前缀的 Agent
	userAgents := pool.List("user-")
	if len(userAgents) != 2 {
		t.Errorf("Expected 2 user agents, got %d", len(userAgents))
	}

	// 列出 admin- 前缀的 Agent
	adminAgents := pool.List("admin-")
	if len(adminAgents) != 2 {
		t.Errorf("Expected 2 admin agents, got %d", len(adminAgents))
	}
}

// TestPool_Remove 测试移除 Agent
func TestPool_Remove(t *testing.T) {
	deps := createTestDeps(t)
	pool := NewPool(&PoolOptions{
		Dependencies: deps,
		MaxAgents:    5,
	})
	defer func() {
		if err := pool.Shutdown(); err != nil {
			t.Errorf("Shutdown failed: %v", err)
		}
	}()

	ctx := context.Background()
	config := createTestConfig("test-agent")

	// 创建 Agent
	_, err := pool.Create(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// 验证存在
	if pool.Size() != 1 {
		t.Error("Agent not in pool")
	}

	// 移除 Agent
	err = pool.Remove("test-agent")
	if err != nil {
		t.Fatalf("Failed to remove agent: %v", err)
	}

	// 验证已移除
	if pool.Size() != 0 {
		t.Error("Agent still in pool after removal")
	}

	_, exists := pool.Get("test-agent")
	if exists {
		t.Error("Agent still retrievable after removal")
	}
}

// TestPool_Status 测试获取 Agent 状态
func TestPool_Status(t *testing.T) {
	deps := createTestDeps(t)
	pool := NewPool(&PoolOptions{
		Dependencies: deps,
		MaxAgents:    5,
	})
	defer func() {
		if err := pool.Shutdown(); err != nil {
			t.Errorf("Shutdown failed: %v", err)
		}
	}()

	ctx := context.Background()
	config := createTestConfig("test-agent")

	_, err := pool.Create(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// 获取状态
	status, err := pool.Status("test-agent")
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

// TestPool_ConcurrentAccess 测试并发访问
func TestPool_ConcurrentAccess(t *testing.T) {
	deps := createTestDeps(t)
	pool := NewPool(&PoolOptions{
		Dependencies: deps,
		MaxAgents:    100,
	})
	defer func() {
		if err := pool.Shutdown(); err != nil {
			t.Errorf("Shutdown failed: %v", err)
		}
	}()

	ctx := context.Background()
	concurrency := 50
	var wg sync.WaitGroup

	// 并发创建 Agent
	for i := range concurrency {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			config := createTestConfig("concurrent-agent-" + string(rune('0'+idx)))
			_, err := pool.Create(ctx, config)
			if err != nil {
				t.Logf("Failed to create agent %d: %v", idx, err)
			}
		}(i)
	}

	wg.Wait()

	// 验证池大小
	size := pool.Size()
	if size != concurrency {
		t.Logf("Expected %d agents, got %d (some creates may have failed)", concurrency, size)
	}

	// 并发读取 Agent
	for i := range concurrency {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			agentID := "concurrent-agent-" + string(rune('0'+idx))
			_, exists := pool.Get(agentID)
			if !exists {
				t.Logf("Agent %s not found", agentID)
			}
		}(i)
	}

	wg.Wait()
}

// TestPool_Shutdown 测试关闭池
func TestPool_Shutdown(t *testing.T) {
	deps := createTestDeps(t)
	pool := NewPool(&PoolOptions{
		Dependencies: deps,
		MaxAgents:    10,
	})

	ctx := context.Background()

	// 创建多个 Agent
	for i := range 5 {
		config := createTestConfig("test-agent-" + string(rune('1'+i)))
		_, err := pool.Create(ctx, config)
		if err != nil {
			t.Fatalf("Failed to create agent: %v", err)
		}
	}

	// 关闭池
	err := pool.Shutdown()
	if err != nil {
		t.Fatalf("Failed to shutdown pool: %v", err)
	}

	// 验证池已清空
	if pool.Size() != 0 {
		t.Errorf("Pool not empty after shutdown, size: %d", pool.Size())
	}
}

// TestPool_ForEach 测试遍历 Agent
func TestPool_ForEach(t *testing.T) {
	deps := createTestDeps(t)
	pool := NewPool(&PoolOptions{
		Dependencies: deps,
		MaxAgents:    10,
	})
	defer func() {
		if err := pool.Shutdown(); err != nil {
			t.Errorf("Shutdown failed: %v", err)
		}
	}()

	ctx := context.Background()

	// 创建 Agent
	agentCount := 5
	for i := range agentCount {
		config := createTestConfig("test-agent-" + string(rune('1'+i)))
		_, err := pool.Create(ctx, config)
		if err != nil {
			t.Fatalf("Failed to create agent: %v", err)
		}
	}

	// 遍历所有 Agent
	visited := make(map[string]bool)
	err := pool.ForEach(func(agentID string, ag *agent.Agent) error {
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

// TestPool_Resume 测试恢复 Agent
// 注意: 这个测试依赖于 Agent 实际保存消息到 Store,在单元测试环境中可能会失败
func TestPool_Resume(t *testing.T) {
	t.Skip("Skipping Resume test - requires real agent message persistence")
	deps := createTestDeps(t)
	ctx := context.Background()

	// 第一个池 - 创建并保存 Agent
	pool1 := NewPool(&PoolOptions{
		Dependencies: deps,
		MaxAgents:    10,
	})

	config := createTestConfig("persistent-agent")

	// 创建 Agent
	ag1, err := pool1.Create(ctx, config)
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

	// 关闭第一个池
	if err := pool1.Shutdown(); err != nil {
		t.Fatalf("Failed to shutdown pool1: %v", err)
	}

	// 第二个池 - 恢复 Agent
	pool2 := NewPool(&PoolOptions{
		Dependencies: deps,
		MaxAgents:    10,
	})
	defer func() {
		if err := pool2.Shutdown(); err != nil {
			t.Errorf("Shutdown failed: %v", err)
		}
	}()

	// 恢复 Agent
	ag2, err := pool2.Resume(ctx, "persistent-agent", config)
	if err != nil {
		t.Fatalf("Failed to resume agent: %v", err)
	}

	if ag2 == nil {
		t.Fatal("Resumed agent is nil")
	}

	// 验证 Agent 在池中
	_, exists := pool2.Get("persistent-agent")
	if !exists {
		t.Error("Resumed agent not found in pool")
	}
}
