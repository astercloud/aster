// MemoryAdvanced 演示基于 memory.Scope 的高级记忆封装策略，支持多用户、
// 多项目、多资源的分层记忆管理，无需 LLM 直接操作文件路径。
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/astercloud/aster/pkg/backends"
	"github.com/astercloud/aster/pkg/memory"
	"github.com/astercloud/aster/pkg/middleware"
	"github.com/astercloud/aster/pkg/tools"
)

// userPreferenceTool 封装: 写用户偏好 -> users/<user-id>/profile/prefs.md
type userPreferenceTool struct {
	userID      string
	memoryWrite tools.Tool
}

func (t *userPreferenceTool) Name() string {
	return "user_preference_write"
}

func (t *userPreferenceTool) Description() string {
	return "Store long-term user preferences in a dedicated memory file."
}

func (t *userPreferenceTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"preference": map[string]any{
				"type":        "string",
				"description": "Plain-language description of the user preference.",
			},
		},
		"required": []string{"preference"},
	}
}

func (t *userPreferenceTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	rawPref, _ := input["preference"].(string)
	if rawPref == "" {
		return nil, fmt.Errorf("preference is required")
	}

	scope := memory.Scope{
		UserID: t.userID,
		Shared: false, // 用户级
	}
	ns := scope.Namespace() // "" => 由 BaseNamespace(users/<user-id>) 控制

	payload := map[string]any{
		"file":      "profile/prefs.md",
		"namespace": ns,
		"mode":      "append",
		"title":     "User preference",
		"content":   rawPref,
	}

	return t.memoryWrite.Execute(ctx, payload, tc)
}

func (t *userPreferenceTool) Prompt() string {
	return `Use this tool to store durable user preferences. You only need to describe the preference; the tool will choose a stable location in memory.`
}

// projectFactTool 封装: 写项目事实/约定
// Shared 控制用户级 vs 全局共享:
//
//	Shared=false: /memories/users/<user-id>/projects/<project-id>/facts.md
//	Shared=true : /memories/projects/<project-id>/facts.md
type projectFactTool struct {
	userID      string
	projectID   string
	shared      bool
	memoryWrite tools.Tool
}

func (t *projectFactTool) Name() string {
	return "project_fact_write"
}

func (t *projectFactTool) Description() string {
	return "Store stable project-level facts and conventions in long-term memory."
}

func (t *projectFactTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"fact": map[string]any{
				"type":        "string",
				"description": "Project-level fact or convention to remember.",
			},
		},
		"required": []string{"fact"},
	}
}

func (t *projectFactTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	fact, _ := input["fact"].(string)
	if fact == "" {
		return nil, fmt.Errorf("fact is required")
	}

	scope := memory.Scope{
		UserID:    t.userID,
		ProjectID: t.projectID,
		Shared:    t.shared,
	}
	ns := scope.Namespace()

	payload := map[string]any{
		"file":      "facts.md",
		"namespace": ns,
		"mode":      "append",
		"title":     "Project fact",
		"content":   fact,
	}

	return t.memoryWrite.Execute(ctx, payload, tc)
}

func (t *projectFactTool) Prompt() string {
	return `Use this tool to store stable project-level facts and conventions, such as coding standards, deployment rules, and critical decisions.`
}

// resourceNoteTool 封装: 写资源级笔记 (文章/小说/歌曲/PPT等)
// 根据 Scope 生成:
//
//	[projects/<project-id>/]resources/<type>/<id>/notes.md
type resourceNoteTool struct {
	userID       string
	projectID    string
	resourceType string
	resourceID   string
	shared       bool
	memoryWrite  tools.Tool
}

func (t *resourceNoteTool) Name() string {
	return "resource_note_write"
}

func (t *resourceNoteTool) Description() string {
	return "Store notes for a specific resource (article, novel, song, PPT, etc.) in long-term memory."
}

func (t *resourceNoteTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"note": map[string]any{
				"type":        "string",
				"description": "Free-form note about the resource.",
			},
		},
		"required": []string{"note"},
	}
}

func (t *resourceNoteTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	note, _ := input["note"].(string)
	if note == "" {
		return nil, fmt.Errorf("note is required")
	}

	scope := memory.Scope{
		UserID:       t.userID,
		ProjectID:    t.projectID,
		ResourceType: t.resourceType,
		ResourceID:   t.resourceID,
		Shared:       t.shared,
	}
	ns := scope.Namespace()

	payload := map[string]any{
		"file":      "notes.md",
		"namespace": ns,
		"mode":      "append",
		"title":     "Resource note",
		"content":   note,
	}

	return t.memoryWrite.Execute(ctx, payload, tc)
}

func (t *resourceNoteTool) Prompt() string {
	return `Use this tool to write notes about a specific resource (article, novel, song, PPT, etc.). The tool will choose a stable namespace and filename; you only need to focus on the note content.`
}

func main() {
	ctx := context.Background()

	// 1. 构建 Backend: 默认使用 StateBackend + 本地 ./memories 目录
	stateBackend := backends.NewStateBackend()
	localMemBackend := backends.NewStateBackend()

	memoryBackend := backends.NewCompositeBackend(
		stateBackend,
		[]backends.RouteConfig{
			{
				Prefix:  "/memories/",
				Backend: localMemBackend,
			},
		},
	)

	// 2. 创建 AgentMemoryMiddleware, 以便获取底层 memory_write 工具
	memoryMW, err := middleware.NewAgentMemoryMiddleware(&middleware.AgentMemoryMiddlewareConfig{
		Backend:    memoryBackend,
		MemoryPath: "/memories/",
	})
	if err != nil {
		log.Fatalf("create AgentMemoryMiddleware failed: %v", err)
	}

	// 3. 通过中间件栈收集工具
	stack := middleware.NewStack([]middleware.Middleware{memoryMW})
	allTools := stack.Tools()

	var memoryWriteTool tools.Tool
	for _, t := range allTools {
		if t.Name() == "memory_write" {
			memoryWriteTool = t
			break
		}
	}
	if memoryWriteTool == nil {
		log.Fatalf("memory_write 工具未找到, 请检查 AgentMemoryMiddleware 实现")
	}

	// 4. 构造封装工具 (以 user=alice, project=demo, article=abc 为例)
	userID := "alice"
	projectID := "demo"

	userPrefTool := &userPreferenceTool{
		userID:      userID,
		memoryWrite: memoryWriteTool,
	}

	projectFact := &projectFactTool{
		userID:      userID,
		projectID:   projectID,
		shared:      false, // 用户级项目记忆
		memoryWrite: memoryWriteTool,
	}

	resourceNote := &resourceNoteTool{
		userID:       userID,
		projectID:    projectID,
		resourceType: "article",
		resourceID:   "abc123",
		shared:       true, // 所有人共享的文章笔记
		memoryWrite:  memoryWriteTool,
	}

	toolCtx := &tools.ToolContext{
		Services: map[string]any{
			"memory_write": memoryWriteTool,
		},
	}

	// 5. 依次执行三个封装工具
	fmt.Println("=== user_preference_write 示例 ===")
	prefResult, err := userPrefTool.Execute(ctx, map[string]any{
		"preference": "Alice 喜欢 grep 风格的搜索和简洁的代码 diff。",
	}, toolCtx)
	if err != nil {
		log.Fatalf("user_preference_write 失败: %v", err)
	}
	fmt.Printf("user_preference_write 结果: %+v\n\n", prefResult)

	fmt.Println("=== project_fact_write 示例 ===")
	factResult, err := projectFact.Execute(ctx, map[string]any{
		"fact": "demo 项目在生产环境只允许使用只读数据库连接。",
	}, toolCtx)
	if err != nil {
		log.Fatalf("project_fact_write 失败: %v", err)
	}
	fmt.Printf("project_fact_write 结果: %+v\n\n", factResult)

	fmt.Println("=== resource_note_write 示例 ===")
	noteResult, err := resourceNote.Execute(ctx, map[string]any{
		"note": "文章 abc123 主要介绍了如何用文件+grep 替代向量RAG。",
	}, toolCtx)
	if err != nil {
		log.Fatalf("resource_note_write 失败: %v", err)
	}
	fmt.Printf("resource_note_write 结果: %+v\n\n", noteResult)

	fmt.Println("✅ 请查看 ./memories-advanced 目录下生成的 Markdown 文件, 对应不同作用域的记忆。")
}
