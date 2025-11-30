// MemoryWorking 演示 Working Memory 的跨会话状态管理功能。
//
// Working Memory 特点：
//   - 自动注入到每轮对话的 system prompt
//   - 支持 thread/resource 两种作用域
//   - 可选的 JSON Schema 验证
//   - 通过 update_working_memory 工具更新
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/astercloud/aster/pkg/backends"
	"github.com/astercloud/aster/pkg/memory"
)

func main() {
	ctx := context.Background()

	// 1. 创建存储后端
	backend := backends.NewStateBackend()

	// 2. 示例 1: Thread Scope（会话级）
	fmt.Println("=== 示例 1: Thread Scope Working Memory ===")
	demonstrateThreadScope(ctx, backend)

	fmt.Println("\n=== 示例 2: Resource Scope Working Memory ===")
	demonstrateResourceScope(ctx, backend)

	fmt.Println("\n=== 示例 3: 带 Schema 的 Working Memory ===")
	demonstrateWithSchema(ctx, backend)

	fmt.Println("\n=== 示例 4: Find and Replace（实验性）===")
	demonstrateFindAndReplace(ctx, backend)
}

func demonstrateThreadScope(ctx context.Context, backend backends.BackendProtocol) {
	// 创建 thread scope 的 Working Memory 管理器
	manager, err := memory.NewWorkingMemoryManager(&memory.WorkingMemoryConfig{
		Backend:  backend,
		BasePath: "/working_memory/",
		Scope:    memory.ScopeThread,
	})
	if err != nil {
		log.Fatalf("create manager: %v", err)
	}

	thread1 := "conversation-001"
	thread2 := "conversation-002"
	resource := "shared-resource"

	// Thread 1: Alice 的会话
	aliceProfile := `# User Profile
Name: Alice
Role: Software Engineer

## Preferences
- Prefers concise explanations
- Uses TypeScript
- Likes functional programming

## Current Task
Status: planning
Goal: Design memory system`

	err = manager.Update(ctx, thread1, resource, aliceProfile)
	if err != nil {
		log.Fatalf("update thread 1: %v", err)
	}
	fmt.Println("✓ Thread 1 (Alice) Working Memory 已更新")

	// Thread 2: Bob 的会话
	bobProfile := `# User Profile
Name: Bob
Role: Product Manager

## Preferences
- Prefers detailed explanations
- Focuses on user stories
- Likes diagrams

## Current Task
Status: reviewing
Goal: Review memory system design`

	err = manager.Update(ctx, thread2, resource, bobProfile)
	if err != nil {
		log.Fatalf("update thread 2: %v", err)
	}
	fmt.Println("✓ Thread 2 (Bob) Working Memory 已更新")

	// 读取各自的 Working Memory
	aliceMemory, _ := manager.Get(ctx, thread1, resource)
	bobMemory, _ := manager.Get(ctx, thread2, resource)

	fmt.Printf("\n📝 Thread 1 (Alice) 的 Working Memory:\n%s\n", aliceMemory)
	fmt.Printf("\n📝 Thread 2 (Bob) 的 Working Memory:\n%s\n", bobMemory)

	fmt.Println("\n✅ Thread Scope: 每个会话有独立的 Working Memory")
}

func demonstrateResourceScope(ctx context.Context, backend backends.BackendProtocol) {
	// 创建 resource scope 的 Working Memory 管理器
	manager, err := memory.NewWorkingMemoryManager(&memory.WorkingMemoryConfig{
		Backend:  backend,
		BasePath: "/working_memory_resource/",
		Scope:    memory.ScopeResource,
	})
	if err != nil {
		log.Fatalf("create manager: %v", err)
	}

	thread1 := "edit-session-001"
	thread2 := "edit-session-002"
	resource := "article-123"

	// Thread 1: 第一次编辑会话
	initialState := `# Article: Getting Started with aster

## Status
Draft version: v0.1
Last editor: Alice
Sections completed: Introduction, Installation

## TODOs
- [ ] Add examples section
- [ ] Add troubleshooting guide
- [ ] Review and polish`

	err = manager.Update(ctx, thread1, resource, initialState)
	if err != nil {
		log.Fatalf("update from thread 1: %v", err)
	}
	fmt.Println("✓ Thread 1 更新了文章状态")

	// Thread 2: 第二次编辑会话（读取相同的 resource）
	stateFromThread2, _ := manager.Get(ctx, thread2, resource)
	fmt.Printf("\n📝 Thread 2 读取到的状态（来自同一 resource）:\n%s\n", stateFromThread2)

	// Thread 2: 继续编辑
	updatedState := `# Article: Getting Started with aster

## Status
Draft version: v0.2
Last editor: Bob
Sections completed: Introduction, Installation, Examples

## TODOs
- [x] Add examples section
- [ ] Add troubleshooting guide
- [ ] Review and polish`

	err = manager.Update(ctx, thread2, resource, updatedState)
	if err != nil {
		log.Fatalf("update from thread 2: %v", err)
	}
	fmt.Println("\n✓ Thread 2 更新了文章状态")

	// Thread 1: 再次读取，会看到 Thread 2 的更新
	latestState, _ := manager.Get(ctx, thread1, resource)
	fmt.Printf("\n📝 Thread 1 读取到最新状态（被 Thread 2 更新）:\n%s\n", latestState)

	fmt.Println("\n✅ Resource Scope: 同一资源的所有会话共享 Working Memory")
}

func demonstrateWithSchema(ctx context.Context, backend backends.BackendProtocol) {
	// 定义 JSON Schema
	schema := &memory.JSONSchema{
		Type: "object",
		Properties: map[string]*memory.JSONSchema{
			"user_name": {Type: "string"},
			"role":      {Type: "string"},
			"task_status": {
				Type: "string",
				Enum: []any{"not_started", "in_progress", "completed"},
			},
			"preferences": {
				Type:  "array",
				Items: &memory.JSONSchema{Type: "string"},
			},
		},
		Required: []string{"user_name", "task_status"},
	}

	manager, err := memory.NewWorkingMemoryManager(&memory.WorkingMemoryConfig{
		Backend:  backend,
		BasePath: "/working_memory_schema/",
		Scope:    memory.ScopeThread,
		Schema:   schema,
	})
	if err != nil {
		log.Fatalf("create manager with schema: %v", err)
	}

	threadID := "structured-session"
	resourceID := "demo"

	// 有效的 JSON（符合 schema）
	validJSON := `{
  "user_name": "Alice",
  "role": "Engineer",
  "task_status": "in_progress",
  "preferences": ["TypeScript", "Functional Programming"]
}`

	err = manager.Update(ctx, threadID, resourceID, validJSON)
	if err != nil {
		log.Fatalf("unexpected error with valid JSON: %v", err)
	}
	fmt.Println("✓ 有效的 JSON 更新成功")

	// 无效的 JSON（缺少必需字段）
	invalidJSON := `{
  "user_name": "Bob"
}`

	err = manager.Update(ctx, threadID, resourceID, invalidJSON)
	if err != nil {
		fmt.Printf("✓ 无效的 JSON 被拒绝（预期行为）: %v\n", err)
	} else {
		fmt.Println("❌ 无效的 JSON 应该被拒绝")
	}

	// 读取当前有效内容
	content, _ := manager.Get(ctx, threadID, resourceID)
	fmt.Printf("\n📝 当前有效的 Working Memory:\n%s\n", content)

	fmt.Println("\n✅ Schema 验证确保数据一致性")
}

func demonstrateFindAndReplace(ctx context.Context, backend backends.BackendProtocol) {
	manager, err := memory.NewWorkingMemoryManager(&memory.WorkingMemoryConfig{
		Backend:  backend,
		BasePath: "/working_memory_experimental/",
		Scope:    memory.ScopeThread,
	})
	if err != nil {
		log.Fatalf("create manager: %v", err)
	}

	threadID := "edit-session"
	resourceID := "task-tracker"

	// 初始状态
	initialState := `# Task Tracker

## Status: in_progress

## Tasks
- [x] Design system
- [ ] Implement features
- [ ] Write tests
- [ ] Write documentation`

	err = manager.Update(ctx, threadID, resourceID, initialState)
	if err != nil {
		log.Fatalf("update initial: %v", err)
	}
	fmt.Println("✓ 初始状态已设置")

	// Find and Replace: 更新状态
	err = manager.FindAndReplace(ctx, threadID, resourceID,
		"Status: in_progress",
		"Status: completed")
	if err != nil {
		log.Fatalf("find and replace: %v", err)
	}
	fmt.Println("✓ 状态已更新（find and replace）")

	// Find and Replace: 更新任务
	err = manager.FindAndReplace(ctx, threadID, resourceID,
		"- [ ] Implement features",
		"- [x] Implement features")
	if err != nil {
		log.Fatalf("update task: %v", err)
	}
	fmt.Println("✓ 任务已完成（find and replace）")

	// Append: 添加新任务（search string 为空）
	err = manager.FindAndReplace(ctx, threadID, resourceID,
		"",
		"- [ ] Deploy to production")
	if err != nil {
		log.Fatalf("append: %v", err)
	}
	fmt.Println("✓ 新任务已添加（append）")

	// 查看最终状态
	finalState, _ := manager.Get(ctx, threadID, resourceID)
	fmt.Printf("\n📝 最终状态:\n%s\n", finalState)

	fmt.Println("\n✅ Find and Replace 实现增量更新")
}

// demonstrateWithMiddleware 演示通过 Middleware 使用 Working Memory（已注释，需要时取消注释）
/*
func demonstrateWithMiddleware(ctx context.Context, backend backends.BackendProtocol) {
	fmt.Println("\n=== 示例 5: 通过 Middleware 使用 Working Memory ===")

	// 创建 Working Memory Middleware
	wmMiddleware, err := middleware.NewWorkingMemoryMiddleware(
		&middleware.WorkingMemoryMiddlewareConfig{
			Backend:  backend,
			BasePath: "/working_memory/",
			Scope:    memory.ScopeThread,
		},
	)
	if err != nil {
		log.Fatalf("create middleware: %v", err)
	}

	// 获取工具
	wmTools := wmMiddleware.Tools()
	fmt.Printf("✓ Working Memory Middleware 提供 %d 个工具:\n", len(wmTools))

	for _, tool := range wmTools {
		fmt.Printf("  - %s: %s\n", tool.Name(), tool.Description())
	}

	// 模拟使用 update_working_memory 工具
	updateTool := wmTools[0]
	input := map[string]any{
		"memory": `# User Profile
Name: Demo User
Status: Testing Working Memory Middleware`,
	}

	toolCtx := &tools.ToolContext{
		ThreadID:   "demo-thread",
		ResourceID: "demo-resource",
	}

	result, err := updateTool.Execute(ctx, input, toolCtx)
	if err != nil {
		log.Fatalf("execute tool: %v", err)
	}

	fmt.Printf("\n✓ update_working_memory 工具执行结果:\n%+v\n", result)

	fmt.Println("\n✅ Middleware 简化了 Working Memory 的集成")
}
*/
