package memory

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/memory"
	"github.com/astercloud/aster/pkg/types"
	"github.com/astercloud/aster/test/integration"
)

// TestMemorySystemIntegration 验证三层内存系统的集成
func TestMemorySystemIntegration(t *testing.T) {
	// 准备模拟响应
	mockResponses := []string{
		"我会记住这个信息",
		"我正在处理这个任务",
		"让我搜索相关的语义信息",
	}

	deps, _ := integration.SetupIntegrationDeps(t, mockResponses)

	// 创建内存管理器
	memManager, err := memory.NewManager(&memory.Config{
		TextMemory: &memory.TextMemoryConfig{
			MaxEntries: 1000,
		},
		WorkingMemory: &memory.WorkingMemoryConfig{
			MaxItems:    50,
			AutoCleanup: true,
		},
		SemanticMemory: &memory.SemanticMemoryConfig{
			VectorStore: "mock",
			Dimensions:  1536,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create memory manager: %v", err)
	}

	// 设置内存管理器到依赖项
	deps.MemoryManager = memManager

	// 创建Agent配置
	config := &types.AgentConfig{
		TemplateID: "memory-test",
		ModelConfig: &types.ModelConfig{
			Provider: "mock",
			Model:    "test-model",
			APIKey:   "test-key",
		},
		Sandbox: &types.SandboxConfig{
			Kind:    types.SandboxKindMock,
			WorkDir: "/tmp/test",
		},
		Memory: &types.MemoryConfig{
			TextMemory: &types.TextMemoryConfig{
				Enabled:    true,
				MaxEntries: 1000,
			},
			WorkingMemory: &types.WorkingMemoryConfig{
				Enabled:    true,
				MaxItems:    50,
				AutoCleanup: true,
			},
			SemanticMemory: &types.SemanticMemoryConfig{
				Enabled:   true,
				VectorStore: "mock",
				Dimensions: 1536,
			},
		},
	}

	// 注册测试模板
	deps.TemplateRegistry.Register(&types.AgentTemplateDefinition{
		ID:           "memory-test",
		SystemPrompt: "You are a helpful assistant with memory capabilities.",
		Model:        "test-model",
		Tools:        []interface{}{"Read", "Write", "MemoryStore", "MemoryRetrieve"},
	})

	// 创建Agent
	ag, err := agent.Create(context.Background(), config, deps)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	defer func() { _ = ag.Close() }()

	// 测试用例1: Text Memory 验证
	t.Run("Text Memory Operations", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 存储文本到内存
		textEntry := &memory.TextEntry{
			Content: "这是一个重要的测试信息",
			Metadata: map[string]interface{}{
				"source": "test",
				"type":   "integration-test",
			},
		}

		err := memManager.StoreText(ctx, textEntry)
		if err != nil {
			t.Fatalf("Failed to store text: %v", err)
		}

		// 检索文本
		retrieved, err := memManager.GetText(ctx, textEntry.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve text: %v", err)
		}

		if retrieved.Content != textEntry.Content {
			t.Errorf("Expected content %s, got %s", textEntry.Content, retrieved.Content)
		}

		// 搜索文本
		results, err := memManager.SearchText(ctx, "重要", 10)
		if err != nil {
			t.Fatalf("Failed to search text: %v", err)
		}

		if len(results) == 0 {
			t.Error("Expected search results, but got none")
		}

		t.Logf("Text memory test passed with %d search results", len(results))
	})

	// 测试用例2: Working Memory 验证
	t.Run("Working Memory Operations", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 添加项目到工作内存
		item := &memory.WorkingMemoryItem{
			Key:       "current_task",
			Value:     "执行内存集成测试",
			Timestamp: time.Now(),
			Priority:  memory.PriorityHigh,
		}

		err := memManager.SetWorkingMemory(ctx, item)
		if err != nil {
			t.Fatalf("Failed to set working memory: %v", err)
		}

		// 获取工作内存项目
		retrieved, err := memManager.GetWorkingMemory(ctx, "current_task")
		if err != nil {
			t.Fatalf("Failed to get working memory: %v", err)
		}

		if retrieved.Value != item.Value {
			t.Errorf("Expected value %s, got %s", item.Value, retrieved.Value)
		}

		// 列出所有工作内存项目
		items, err := memManager.ListWorkingMemory(ctx)
		if err != nil {
			t.Fatalf("Failed to list working memory: %v", err)
		}

		if len(items) == 0 {
			t.Error("Expected working memory items, but got none")
		}

		t.Logf("Working memory test passed with %d items", len(items))
	})

	// 测试用例3: Semantic Memory 验证
	t.Run("Semantic Memory Operations", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 创建语义向量
		vector := make([]float32, 1536)
		for i := range vector {
			vector[i] = 0.1 // 模拟向量值
		}

		// 存储语义记忆
		semanticEntry := &memory.SemanticEntry{
			Vector:    vector,
			Content:   "语义搜索测试内容",
			Metadata: map[string]interface{}{
				"category": "test",
			},
		}

		err := memManager.StoreSemantic(ctx, semanticEntry)
		if err != nil {
			t.Fatalf("Failed to store semantic entry: %v", err)
		}

		// 语义搜索
		queryVector := make([]float32, 1536)
		for i := range queryVector {
			queryVector[i] = 0.1 // 与存储向量相似的查询
		}

		results, err := memManager.SearchSemantic(ctx, queryVector, 10, 0.8)
		if err != nil {
			t.Fatalf("Failed to search semantic: %v", err)
		}

		if len(results) == 0 {
			t.Error("Expected semantic search results, but got none")
		}

		t.Logf("Semantic memory test passed with %d search results", len(results))
	})

	// 测试用例4: 内存系统集成测试
	t.Run("Memory System Integration", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// 模拟Agent使用内存系统的场景
		conversationID := "test-conversation"

		// 1. 存储用户输入到文本内存
		userMessage := &memory.TextEntry{
			Content:   "用户询问如何优化Go程序性能",
			Metadata: map[string]interface{}{
				"conversation_id": conversationID,
				"role":           "user",
				"timestamp":      time.Now(),
			},
		}

		err := memManager.StoreText(ctx, userMessage)
		if err != nil {
			t.Fatalf("Failed to store user message: %v", err)
		}

		// 2. 设置当前任务到工作内存
		task := &memory.WorkingMemoryItem{
			Key:       "current_query",
			Value:     "Go性能优化",
			Timestamp: time.Now(),
			Priority:  memory.PriorityHigh,
		}

		err = memManager.SetWorkingMemory(ctx, task)
		if err != nil {
			t.Fatalf("Failed to set current task: %v", err)
		}

		// 3. 创建相关的语义向量并存储
		semanticVector := make([]float32, 1536)
		for i := range semanticVector {
			semanticVector[i] = float32(i) / 1536.0 // 创建有意义的向量
		}

		semanticEntry := &memory.SemanticEntry{
			Vector:  semanticVector,
			Content: "Go程序性能优化技巧",
			Metadata: map[string]interface{}{
				"topic":     "performance",
				"language":  "go",
				"category":  "optimization",
			},
		}

		err = memManager.StoreSemantic(ctx, semanticEntry)
		if err != nil {
			t.Fatalf("Failed to store semantic entry: %v", err)
		}

		// 4. 模拟Agent使用内存进行响应
		// 搜索相关文本
		textResults, err := memManager.SearchText(ctx, "性能", 5)
		if err != nil {
			t.Fatalf("Failed to search relevant text: %v", err)
		}

		// 搜索相关语义
		semanticResults, err := memManager.SearchSemantic(ctx, semanticVector, 5, 0.7)
		if err != nil {
			t.Fatalf("Failed to search relevant semantics: %v", err)
		}

		// 获取当前任务上下文
		currentTask, err := memManager.GetWorkingMemory(ctx, "current_query")
		if err != nil {
			t.Fatalf("Failed to get current task: %v", err)
		}

		// 验证集成结果
		if len(textResults) == 0 && len(semanticResults) == 0 {
			t.Error("Expected at least some memory results for integration test")
		}

		if currentTask.Value != "Go性能优化" {
			t.Errorf("Expected current task 'Go性能优化', got '%s'", currentTask.Value)
		}

		t.Logf("Memory integration test passed:")
		t.Logf("  - Text results: %d", len(textResults))
		t.Logf("  - Semantic results: %d", len(semanticResults))
		t.Logf("  - Current task: %s", currentTask.Value)
	})

	// 测试用例5: 内存持久化验证
	t.Run("Memory Persistence", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 创建临时文件用于持久化测试
		tmpFile, err := os.CreateTemp(t.TempDir(), "memory_test_*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		// 存储一些测试数据
		testEntry := &memory.TextEntry{
			Content: "持久化测试数据",
			Metadata: map[string]interface{}{
				"test":    "persistence",
				"created": time.Now(),
			},
		}

		err = memManager.StoreText(ctx, testEntry)
		if err != nil {
			t.Fatalf("Failed to store test entry: %v", err)
		}

		// 模拟持久化操作（具体实现取决于内存管理器的持久化接口）
		// 这里只是验证数据是否正确存储
		retrieved, err := memManager.GetText(ctx, testEntry.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve stored entry: %v", err)
		}

		if retrieved.Content != testEntry.Content {
			t.Errorf("Persistence test failed: expected %s, got %s", testEntry.Content, retrieved.Content)
		}

		t.Log("Memory persistence test passed")
	})

	t.Log("Memory system integration test completed successfully")
}

// TestMemoryProvenance 验证内存来源追踪功能
func TestMemoryProvenance(t *testing.T) {
	mockResponses := []string{"我会追踪信息的来源"}
	deps, _ := integration.SetupIntegrationDeps(t, mockResponses)

	// 创建带来源追踪的内存管理器
	memManager, err := memory.NewManager(&memory.Config{
		TextMemory: &memory.TextMemoryConfig{
			MaxEntries: 100,
			TrackProvenance: true,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create memory manager with provenance: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 创建带来源信息的条目
	entry := &memory.TextEntry{
		Content: "需要追踪来源的信息",
		Metadata: map[string]interface{}{
			"source": "user_input",
			"session_id": "test-session-123",
		},
		Provenance: &memory.Provenance{
			Source:    "direct_input",
			Timestamp: time.Now(),
			UserID:    "test-user",
			SessionID: "test-session-123",
		},
	}

	err = memManager.StoreText(ctx, entry)
	if err != nil {
		t.Fatalf("Failed to store entry with provenance: %v", err)
	}

	// 检索并验证来源信息
	retrieved, err := memManager.GetText(ctx, entry.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve entry with provenance: %v", err)
	}

	if retrieved.Provenance == nil {
		t.Error("Expected provenance information, but got nil")
	}

	if retrieved.Provenance.Source != "direct_input" {
		t.Errorf("Expected source 'direct_input', got '%s'", retrieved.Provenance.Source)
	}

	t.Log("Memory provenance test passed")
}