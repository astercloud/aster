package middleware

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/astercloud/aster/pkg/backends"
	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/memory"
	"github.com/astercloud/aster/pkg/tools"
)

var wmLog = logging.ForComponent("WorkingMemoryMiddleware")

// WorkingMemoryMiddleware Working Memory 中间件
// 功能：
// 1. 根据 threadID/resourceID 加载 Working Memory
// 2. 将 Working Memory 内容注入到 system prompt
// 3. 提供 update_working_memory 工具
type WorkingMemoryMiddleware struct {
	*BaseMiddleware

	manager              *memory.WorkingMemoryManager
	systemPromptTemplate string
	workingMemoryTools   []tools.Tool
	experimental         bool // 是否启用实验性 find/replace 工具
}

// WorkingMemoryMiddlewareConfig 配置
type WorkingMemoryMiddlewareConfig struct {
	Backend              backends.BackendProtocol  // 存储后端
	BasePath             string                    // 存储根路径，默认 "/working_memory/"
	Scope                memory.WorkingMemoryScope // "thread" | "resource"
	Schema               *memory.JSONSchema        // 可选的 JSON Schema
	Template             string                    // 可选的 Markdown 模板
	TTL                  time.Duration             // 可选的过期时间（0 表示不过期）
	SystemPromptTemplate string                    // 可选，自定义 system prompt 模板
	Experimental         bool                      // 是否启用实验性功能
}

// NewWorkingMemoryMiddleware 创建 Working Memory 中间件
func NewWorkingMemoryMiddleware(config *WorkingMemoryMiddlewareConfig) (*WorkingMemoryMiddleware, error) {
	if config == nil {
		return nil, errors.New("working memory not configured properly")
	}

	if config.Backend == nil {
		return nil, errors.New("backend is required")
	}

	// 创建 Working Memory 管理器
	manager, err := memory.NewWorkingMemoryManager(&memory.WorkingMemoryConfig{
		Backend:    config.Backend,
		BasePath:   config.BasePath,
		Scope:      config.Scope,
		Schema:     config.Schema,
		Template:   config.Template,
		DefaultTTL: config.TTL,
	})
	if err != nil {
		return nil, fmt.Errorf("create working memory manager: %w", err)
	}

	systemPromptTemplate := config.SystemPromptTemplate
	if systemPromptTemplate == "" {
		systemPromptTemplate = "<working_memory>\n%s\n</working_memory>"
	}

	m := &WorkingMemoryMiddleware{
		BaseMiddleware:       NewBaseMiddleware("working_memory", 6), // 比 agent_memory 稍低优先级
		manager:              manager,
		systemPromptTemplate: systemPromptTemplate,
		experimental:         config.Experimental,
	}

	// 创建 Working Memory 工具
	m.workingMemoryTools = m.createWorkingMemoryTools()

	wmLog.Info(context.Background(), "initialized", map[string]any{"scope": config.Scope, "base_path": config.BasePath})
	return m, nil
}

// Tools 返回 Working Memory 相关工具
func (m *WorkingMemoryMiddleware) Tools() []tools.Tool {
	return m.workingMemoryTools
}

// WrapModelCall 包装模型调用，注入 Working Memory 到 system prompt
func (m *WorkingMemoryMiddleware) WrapModelCall(ctx context.Context, req *ModelRequest, handler ModelCallHandler) (*ModelResponse, error) {
	// 从请求元数据中获取 threadID 和 resourceID
	// 注意：这些值应该由 Agent 在调用时设置
	var threadID, resourceID string
	if req.Metadata != nil {
		if tid, ok := req.Metadata["thread_id"].(string); ok {
			threadID = tid
		}
		if rid, ok := req.Metadata["resource_id"].(string); ok {
			resourceID = rid
		}
	}

	// 如果没有 threadID 或 resourceID，使用默认值
	if threadID == "" && resourceID == "" {
		// 可以根据 AgentID 生成默认的 threadID
		if req.Metadata != nil {
			if agentID, ok := req.Metadata["agent_id"].(string); ok && agentID != "" {
				threadID = agentID
			}
		}
		if threadID == "" {
			threadID = "default"
		}
	}

	// 加载 Working Memory
	workingMemoryContent, err := m.manager.Get(ctx, threadID, resourceID)
	if err != nil {
		wmLog.Error(ctx, "failed to load working memory", map[string]any{"error": err.Error()})
		workingMemoryContent = ""
	}

	// 保存原始 system prompt
	originalSystemPrompt := req.SystemPrompt

	// 构建 Working Memory 部分
	if workingMemoryContent != "" {
		memorySection := fmt.Sprintf(m.systemPromptTemplate, workingMemoryContent)

		// 注入到 system prompt 开头
		if originalSystemPrompt != "" {
			req.SystemPrompt = memorySection + "\n\n" + originalSystemPrompt
		} else {
			req.SystemPrompt = memorySection
		}

		wmLog.Debug(ctx, "injected working memory", map[string]any{"chars": len(workingMemoryContent), "thread": threadID, "resource": resourceID})
	} else {
		wmLog.Debug(ctx, "no working memory found", map[string]any{"thread": threadID, "resource": resourceID})
	}

	// 追加 Working Memory 使用文档
	workingMemoryPrompt := m.buildWorkingMemoryPrompt()
	req.SystemPrompt = req.SystemPrompt + "\n\n" + workingMemoryPrompt

	// 调用处理器
	resp, err := handler(ctx, req)

	// 恢复原始 system prompt
	req.SystemPrompt = originalSystemPrompt

	return resp, err
}

// buildWorkingMemoryPrompt 构建 Working Memory 使用指南
func (m *WorkingMemoryMiddleware) buildWorkingMemoryPrompt() string {
	scopeDesc := "per conversation"
	if m.manager.GetScope() == memory.ScopeResource {
		scopeDesc = "shared across all conversations for the same resource"
	}

	schemaNote := ""
	if m.manager.GetSchema() != nil {
		schemaNote = "\n- The memory content must match the configured JSON schema"
	}

	return fmt.Sprintf(`## Working Memory

You have access to a working memory system that maintains state %s.

### What is Working Memory?

Working Memory is a persistent, structured state that you can update throughout the conversation.
Unlike regular messages, working memory:
- Persists across multiple turns
- Can be updated incrementally
- Is always loaded at the start of each request
- Is limited in size, so keep it concise

### When to Update Working Memory

Update working memory when you:
- Learn important information about the user or task
- Need to track the current state of a multi-step process
- Want to remember decisions and their context
- Discover patterns or preferences

### How to Update

Use the 'update_working_memory' tool with the new content.
- The tool will OVERWRITE the entire working memory
- Include ALL information you want to keep, not just the changes
- Structure the content clearly (use Markdown headings/lists)%s

### Best Practices

- Keep working memory concise (aim for < 500 words)
- Use clear structure (headings, bullet points)
- Focus on actionable information
- Update proactively as you learn

Remember: Working memory is %s, so be mindful of what you store.`,
		scopeDesc,
		schemaNote,
		scopeDesc,
	)
}

// createWorkingMemoryTools 创建 Working Memory 工具
func (m *WorkingMemoryMiddleware) createWorkingMemoryTools() []tools.Tool {
	tools := []tools.Tool{
		NewUpdateWorkingMemoryTool(m.manager),
	}

	if m.experimental {
		tools = append(tools, NewExperimentalUpdateWorkingMemoryTool(m.manager))
	}

	return tools
}

// GetManager 获取 Working Memory 管理器（用于测试或高级用法）
func (m *WorkingMemoryMiddleware) GetManager() *memory.WorkingMemoryManager {
	return m.manager
}

// GetConfig 获取配置信息
func (m *WorkingMemoryMiddleware) GetConfig() map[string]any {
	return map[string]any{
		"scope":        string(m.manager.GetScope()),
		"has_schema":   m.manager.GetSchema() != nil,
		"has_template": m.manager.GetTemplate() != "",
		"experimental": m.experimental,
	}
}
