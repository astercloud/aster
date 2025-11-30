// Memory 演示 AgentMemoryMiddleware 提供的长期记忆能力，使用基于文件的
// 存储和 grep 风格的搜索。
//
// 演示功能:
//   - memory_write: 向记忆文件追加或覆盖 Markdown 笔记
//   - memory_search: 在记忆目录中进行全文搜索
//
// 在生产环境中，这些工具通过 AgentConfig.Middlewares 自动注入并由 LLM 调用。
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/astercloud/aster/pkg/backends"
	"github.com/astercloud/aster/pkg/middleware"
	"github.com/astercloud/aster/pkg/tools"
)

func main() {
	ctx := context.Background()

	// 1. 构建 Backend:
	//    - 使用 StateBackend (内存临时文件)
	stateBackend := backends.NewStateBackend()

	// Note: 在实际应用中，可以使用FilesystemBackend或其他持久化Backend
	// memoryBackend := backends.NewCompositeBackend(...)
	memoryBackend := stateBackend

	// 2. 创建 Filesystem + AgentMemory 中间件
	fsMiddleware := middleware.NewFilesystemMiddleware(&middleware.FilesystemMiddlewareConfig{
		Backend: memoryBackend,
	})

	memoryMW, err := middleware.NewAgentMemoryMiddleware(&middleware.AgentMemoryMiddlewareConfig{
		Backend:    memoryBackend,
		MemoryPath: "/memories/",
	})
	if err != nil {
		log.Fatalf("create AgentMemoryMiddleware failed: %v", err)
	}

	// 3. 组装中间件栈并收集工具
	stack := middleware.NewStack([]middleware.Middleware{
		fsMiddleware,
		memoryMW,
	})

	allTools := stack.Tools()
	fmt.Printf("✅ 中间件栈已创建, 工具总数: %d\n\n", len(allTools))

	var memoryWriteTool tools.Tool
	var memorySearchTool tools.Tool

	for _, t := range allTools {
		fmt.Printf("- 工具: %-16s 描述: %s\n", t.Name(), t.Description())
		switch t.Name() {
		case "memory_write":
			memoryWriteTool = t
		case "memory_search":
			memorySearchTool = t
		}
	}
	fmt.Println()

	if memoryWriteTool == nil || memorySearchTool == nil {
		log.Fatalf("memory_write 或 memory_search 工具未找到, 请检查 AgentMemoryMiddleware 初始化是否成功")
	}

	toolCtx := &tools.ToolContext{} // 本例中不需要 Sandbox, 传空即可

	// 4. 使用 memory_write 追加一条长期记忆
	fmt.Println("=== 使用 memory_write 追加记忆 ===")
	writeInput := map[string]any{
		"file":    "user/alice.md",
		"mode":    "append",
		"title":   "初次见面",
		"content": "Alice 喜欢 grep 风格的记忆系统, 并偏好简洁的代码 diff。",
	}

	writeResult, err := memoryWriteTool.Execute(ctx, writeInput, toolCtx)
	if err != nil {
		log.Fatalf("memory_write 执行失败: %v", err)
	}
	fmt.Printf("memory_write 结果: %+v\n\n", writeResult)

	// 5. 使用 memory_search 在记忆中搜索关键字
	fmt.Println("=== 使用 memory_search 搜索记忆 ===")
	searchInput := map[string]any{
		"query":       "Alice",
		"glob":        "user/*.md",
		"max_results": 10,
	}

	searchResult, err := memorySearchTool.Execute(ctx, searchInput, toolCtx)
	if err != nil {
		log.Fatalf("memory_search 执行失败: %v", err)
	}
	fmt.Printf("memory_search 结果: %+v\n", searchResult)
}
