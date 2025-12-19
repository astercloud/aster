package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/astercloud/aster/pkg/memory"
	"github.com/astercloud/aster/pkg/memory/logic"
	"github.com/astercloud/aster/pkg/tools"
)

// LogicMemoryQueryTool 查询 Logic Memory 工具
// 允许 Agent 主动查询用户偏好和行为模式
type LogicMemoryQueryTool struct {
	manager *logic.Manager
}

// NewLogicMemoryQueryTool 创建 Logic Memory 查询工具
func NewLogicMemoryQueryTool(manager *logic.Manager) *LogicMemoryQueryTool {
	return &LogicMemoryQueryTool{manager: manager}
}

// Name 返回工具名称
func (t *LogicMemoryQueryTool) Name() string {
	return "logic_memory_query"
}

// Description 返回工具描述
func (t *LogicMemoryQueryTool) Description() string {
	return "Query user preferences and behavior patterns stored in logic memory. Use this to understand user preferences before generating content or making decisions."
}

// InputSchema 返回输入 JSON Schema
func (t *LogicMemoryQueryTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"list", "get", "search", "stats"},
				"description": "Action to perform: list (all memories), get (single memory by key), search (by type), stats (get statistics)",
			},
			"namespace": map[string]any{
				"type":        "string",
				"description": "Namespace to query (e.g., 'user:123'). If not provided, uses the current context's namespace.",
			},
			"key": map[string]any{
				"type":        "string",
				"description": "Memory key for 'get' action",
			},
			"type": map[string]any{
				"type":        "string",
				"description": "Memory type for 'search' action (e.g., 'preference', 'behavior_pattern')",
			},
			"scope": map[string]any{
				"type":        "string",
				"enum":        []string{"session", "user", "global"},
				"description": "Filter by scope",
			},
			"min_confidence": map[string]any{
				"type":        "number",
				"description": "Minimum confidence threshold (0.0-1.0)",
				"default":     0.5,
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of results to return",
				"default":     10,
			},
		},
		"required": []string{"action"},
	}
}

// Execute 执行工具
func (t *LogicMemoryQueryTool) Execute(ctx context.Context, input map[string]any, tctx *tools.ToolContext) (any, error) {
	action, _ := input["action"].(string)
	namespace, _ := input["namespace"].(string)

	// 如果没有提供 namespace，尝试从上下文获取
	if namespace == "" && tctx != nil {
		// 使用 ThreadID 或 AgentID 作为 namespace
		if tctx.ThreadID != "" {
			namespace = "thread:" + tctx.ThreadID
		} else if tctx.AgentID != "" {
			namespace = "agent:" + tctx.AgentID
		}
	}
	if namespace == "" {
		return nil, errors.New("namespace is required")
	}

	switch action {
	case "list":
		return t.list(ctx, namespace, input)
	case "get":
		return t.get(ctx, namespace, input)
	case "search":
		return t.search(ctx, namespace, input)
	case "stats":
		return t.stats(ctx, namespace)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

// Prompt 返回工具使用说明
func (t *LogicMemoryQueryTool) Prompt() string {
	return `Use logic_memory_query to retrieve learned user preferences and behavior patterns.

**Actions:**
- list: List all memories for a namespace
- get: Get a specific memory by key
- search: Search memories by type
- stats: Get statistics about stored memories

**Example:**
{"action": "list", "namespace": "user:123", "limit": 5}
{"action": "get", "namespace": "user:123", "key": "writing_tone_preference"}
{"action": "search", "namespace": "user:123", "type": "preference"}

**When to use:**
- Before generating content that should match user preferences
- To check if you've learned something about the user
- To understand user behavior patterns for personalization`
}

func (t *LogicMemoryQueryTool) list(ctx context.Context, namespace string, input map[string]any) (string, error) {
	// 构建过滤器
	filters := []logic.Filter{}

	if scope, ok := input["scope"].(string); ok && scope != "" {
		filters = append(filters, logic.WithScope(logic.MemoryScope(scope)))
	}

	if minConf, ok := input["min_confidence"].(float64); ok {
		filters = append(filters, logic.WithMinConfidence(minConf))
	}

	limit := 10
	if l, ok := input["limit"].(float64); ok {
		limit = int(l)
	}
	filters = append(filters, logic.WithTopK(limit))

	// 检索 Memory
	memories, err := t.manager.RetrieveMemories(ctx, namespace, filters...)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve memories: %w", err)
	}

	return t.formatMemoriesAsMarkdown(memories), nil
}

func (t *LogicMemoryQueryTool) get(ctx context.Context, namespace string, input map[string]any) (string, error) {
	key, ok := input["key"].(string)
	if !ok || key == "" {
		return "", errors.New("key is required for 'get' action")
	}

	memory, err := t.manager.GetMemory(ctx, namespace, key)
	if err != nil {
		return "", fmt.Errorf("memory not found: %w", err)
	}

	return t.formatSingleMemoryAsMarkdown(memory), nil
}

func (t *LogicMemoryQueryTool) search(ctx context.Context, namespace string, input map[string]any) (string, error) {
	memoryType, ok := input["type"].(string)
	if !ok || memoryType == "" {
		return "", errors.New("type is required for 'search' action")
	}

	filters := []logic.Filter{logic.WithType(memoryType)}

	if minConf, ok := input["min_confidence"].(float64); ok {
		filters = append(filters, logic.WithMinConfidence(minConf))
	}

	limit := 10
	if l, ok := input["limit"].(float64); ok {
		limit = int(l)
	}
	filters = append(filters, logic.WithTopK(limit))

	memories, err := t.manager.RetrieveMemories(ctx, namespace, filters...)
	if err != nil {
		return "", fmt.Errorf("failed to search memories: %w", err)
	}

	return t.formatMemoriesAsMarkdown(memories), nil
}

func (t *LogicMemoryQueryTool) stats(ctx context.Context, namespace string) (string, error) {
	stats, err := t.manager.GetStats(ctx, namespace)
	if err != nil {
		return "", fmt.Errorf("failed to get stats: %w", err)
	}

	var builder strings.Builder
	builder.WriteString("## Logic Memory Statistics\n\n")
	builder.WriteString(fmt.Sprintf("- **Total Memories**: %d\n", stats.TotalCount))
	builder.WriteString(fmt.Sprintf("- **Average Confidence**: %.1f%%\n", stats.AverageConfidence*100))
	builder.WriteString(fmt.Sprintf("- **Last Updated**: %s\n\n", stats.LastUpdated.Format("2006-01-02 15:04:05")))

	if len(stats.CountByType) > 0 {
		builder.WriteString("### By Type\n")
		for memType, count := range stats.CountByType {
			builder.WriteString(fmt.Sprintf("- %s: %d\n", memType, count))
		}
		builder.WriteString("\n")
	}

	if len(stats.CountByScope) > 0 {
		builder.WriteString("### By Scope\n")
		for scope, count := range stats.CountByScope {
			builder.WriteString(fmt.Sprintf("- %s: %d\n", scope, count))
		}
	}

	return builder.String(), nil
}

func (t *LogicMemoryQueryTool) formatMemoriesAsMarkdown(memories []*logic.LogicMemory) string {
	if len(memories) == 0 {
		return "No memories found."
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("## Logic Memories (%d found)\n\n", len(memories)))

	for i, mem := range memories {
		builder.WriteString(t.formatSingleMemoryAsMarkdownItem(i+1, mem))
		builder.WriteString("\n")
	}

	return builder.String()
}

func (t *LogicMemoryQueryTool) formatSingleMemoryAsMarkdown(mem *logic.LogicMemory) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("## Memory: %s\n\n", mem.Key))
	builder.WriteString(fmt.Sprintf("- **Type**: %s\n", mem.Type))
	builder.WriteString(fmt.Sprintf("- **Scope**: %s\n", mem.Scope))
	builder.WriteString(fmt.Sprintf("- **Description**: %s\n", mem.Description))

	if mem.Provenance != nil {
		builder.WriteString(fmt.Sprintf("- **Confidence**: %.1f%%\n", mem.Provenance.Confidence*100))
		builder.WriteString(fmt.Sprintf("- **Source Type**: %s\n", mem.Provenance.SourceType))
	}

	builder.WriteString(fmt.Sprintf("- **Access Count**: %d\n", mem.AccessCount))
	builder.WriteString(fmt.Sprintf("- **Last Accessed**: %s\n", mem.LastAccessed.Format("2006-01-02 15:04:05")))
	builder.WriteString(fmt.Sprintf("- **Created**: %s\n", mem.CreatedAt.Format("2006-01-02 15:04:05")))

	if mem.Value != nil {
		valueJSON, _ := json.MarshalIndent(mem.Value, "", "  ")
		builder.WriteString(fmt.Sprintf("\n### Value\n```json\n%s\n```\n", string(valueJSON)))
	}

	return builder.String()
}

func (t *LogicMemoryQueryTool) formatSingleMemoryAsMarkdownItem(index int, mem *logic.LogicMemory) string {
	confidence := 0.0
	if mem.Provenance != nil {
		confidence = mem.Provenance.Confidence
	}

	return fmt.Sprintf("%d. **%s** (%s, %s)\n   - %s\n   - Confidence: %.0f%%, Accessed: %d times\n",
		index, mem.Key, mem.Type, mem.Scope, mem.Description, confidence*100, mem.AccessCount)
}

// 确保实现 Tool 接口
var _ tools.Tool = (*LogicMemoryQueryTool)(nil)

// ============================================
// LogicMemoryUpdateTool - 更新 Logic Memory
// ============================================

// LogicMemoryUpdateTool 更新 Logic Memory 工具
// 允许 Agent 主动记录用户偏好和行为模式
type LogicMemoryUpdateTool struct {
	manager *logic.Manager
}

// NewLogicMemoryUpdateTool 创建 Logic Memory 更新工具
func NewLogicMemoryUpdateTool(manager *logic.Manager) *LogicMemoryUpdateTool {
	return &LogicMemoryUpdateTool{manager: manager}
}

// Name 返回工具名称
func (t *LogicMemoryUpdateTool) Name() string {
	return "logic_memory_update"
}

// Description 返回工具描述
func (t *LogicMemoryUpdateTool) Description() string {
	return "Record or update user preferences and behavior patterns in logic memory. Use this when you learn something important about the user's preferences, habits, or working style that should be remembered for future interactions."
}

// InputSchema 返回输入 JSON Schema
func (t *LogicMemoryUpdateTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"record", "delete"},
				"description": "Action to perform: record (create/update memory), delete (remove memory)",
			},
			"namespace": map[string]any{
				"type":        "string",
				"description": "Namespace for the memory (e.g., 'user:123'). If not provided, uses the current context's namespace.",
			},
			"key": map[string]any{
				"type":        "string",
				"description": "Unique key for the memory (e.g., 'writing_tone_preference', 'code_style')",
			},
			"type": map[string]any{
				"type":        "string",
				"description": "Memory type (e.g., 'preference', 'behavior_pattern', 'skill_level')",
			},
			"scope": map[string]any{
				"type":        "string",
				"enum":        []string{"session", "user", "global"},
				"description": "Memory scope: session (temporary), user (persists for user), global (shared)",
				"default":     "user",
			},
			"value": map[string]any{
				"type":        "object",
				"description": "Structured value to store (can be any JSON object)",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Human-readable description of what this memory represents",
			},
			"confidence": map[string]any{
				"type":        "number",
				"description": "Initial confidence level (0.0-1.0). Higher values indicate stronger evidence.",
				"default":     0.7,
			},
			"category": map[string]any{
				"type":        "string",
				"description": "Optional category for grouping related memories",
			},
		},
		"required": []string{"action", "key"},
	}
}

// Execute 执行工具
func (t *LogicMemoryUpdateTool) Execute(ctx context.Context, input map[string]any, tctx *tools.ToolContext) (any, error) {
	action, _ := input["action"].(string)
	namespace, _ := input["namespace"].(string)

	// 如果没有提供 namespace，尝试从上下文获取
	if namespace == "" && tctx != nil {
		if tctx.ThreadID != "" {
			namespace = "thread:" + tctx.ThreadID
		} else if tctx.AgentID != "" {
			namespace = "agent:" + tctx.AgentID
		}
	}
	if namespace == "" {
		return nil, errors.New("namespace is required")
	}

	key, _ := input["key"].(string)
	if key == "" {
		return nil, errors.New("key is required")
	}

	switch action {
	case "record":
		return t.record(ctx, namespace, key, input)
	case "delete":
		return t.delete(ctx, namespace, key)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

// Prompt 返回工具使用说明
func (t *LogicMemoryUpdateTool) Prompt() string {
	return `Use logic_memory_update to record learned user preferences and behavior patterns.

**Actions:**
- record: Create or update a memory
- delete: Remove a memory

**Example - Recording a preference:**
{
  "action": "record",
  "namespace": "user:123",
  "key": "writing_tone_preference",
  "type": "preference",
  "scope": "user",
  "description": "User prefers casual, conversational tone over formal writing",
  "value": {"tone": "casual", "avoid": ["however", "furthermore"]},
  "confidence": 0.8
}

**When to record:**
- User explicitly states a preference
- User consistently corrects your output in a specific way
- User's behavior shows a clear pattern
- User gives feedback about your responses

**Best practices:**
- Use descriptive keys (e.g., 'code_formatting_preference' not 'pref1')
- Write clear descriptions that can be injected into prompts
- Start with moderate confidence (0.6-0.7) for inferred preferences
- Use higher confidence (0.8-0.9) for explicit user statements`
}

func (t *LogicMemoryUpdateTool) record(ctx context.Context, namespace, key string, input map[string]any) (string, error) {
	// 获取必要字段
	memType, _ := input["type"].(string)
	if memType == "" {
		memType = "preference"
	}

	scopeStr, _ := input["scope"].(string)
	if scopeStr == "" {
		scopeStr = "user"
	}
	scope := logic.MemoryScope(scopeStr)

	description, _ := input["description"].(string)
	if description == "" {
		return "", errors.New("description is required for recording memories")
	}

	value := input["value"]
	if value == nil {
		value = map[string]any{}
	}

	confidence := 0.7
	if conf, ok := input["confidence"].(float64); ok {
		confidence = conf
	}

	category, _ := input["category"].(string)

	// 创建 Memory
	mem := &logic.LogicMemory{
		Namespace:   namespace,
		Scope:       scope,
		Type:        memType,
		Category:    category,
		Key:         key,
		Value:       value,
		Description: description,
		Provenance: &memory.MemoryProvenance{
			SourceType: memory.SourceAgent,
			Confidence: confidence,
		},
	}

	// 记录 Memory
	if err := t.manager.RecordMemory(ctx, mem); err != nil {
		return "", fmt.Errorf("failed to record memory: %w", err)
	}

	return fmt.Sprintf("Successfully recorded memory: **%s** (%s)\n- Type: %s\n- Scope: %s\n- Description: %s\n- Confidence: %.0f%%",
		key, namespace, memType, scope, description, confidence*100), nil
}

func (t *LogicMemoryUpdateTool) delete(ctx context.Context, namespace, key string) (string, error) {
	if err := t.manager.DeleteMemory(ctx, namespace, key); err != nil {
		return "", fmt.Errorf("failed to delete memory: %w", err)
	}

	return fmt.Sprintf("Successfully deleted memory: **%s** from namespace %s", key, namespace), nil
}

// 确保实现 Tool 接口
var _ tools.Tool = (*LogicMemoryUpdateTool)(nil)
