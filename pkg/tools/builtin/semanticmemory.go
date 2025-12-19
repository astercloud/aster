package builtin

import (
	"context"
	"errors"

	"github.com/astercloud/aster/pkg/memory"
	"github.com/astercloud/aster/pkg/tools"
)

// SemanticSearchTool 基于 SemanticMemory 的语义检索工具。
// 输入:
//
//	{
//	  "query": string,
//	  "top_k": number (可选, 默认使用 SemanticMemoryConfig.TopK),
//	  "metadata": object (可选, 如 {"user_id":"alice","project_id":"demo"})
//	}
//
// 输出:
//
//	[
//	  {"id": "...", "score": 0.87, "metadata": {...}},
//	  ...
//	]
type SemanticSearchTool struct {
	sm *memory.SemanticMemory
}

func NewSemanticSearchTool(config map[string]any) (tools.Tool, error) {
	// 尝试从配置中获取 SemanticMemory 实例
	var sm *memory.SemanticMemory
	if config != nil {
		if smAny, exists := config["semantic_memory"]; exists {
			if smVal, ok := smAny.(*memory.SemanticMemory); ok {
				sm = smVal
			}
		}
	}

	return &SemanticSearchTool{sm: sm}, nil
}

func (t *SemanticSearchTool) Name() string {
	return "semantic_search"
}

func (t *SemanticSearchTool) Description() string {
	return "Perform semantic search over indexed texts using a configured vector store and embedder."
}

func (t *SemanticSearchTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Natural language query text.",
			},
			"top_k": map[string]any{
				"type":        "integer",
				"description": "Optional number of results to return.",
			},
			"metadata": map[string]any{
				"type":                 "object",
				"additionalProperties": true,
				"description":          "Optional metadata map (e.g. user_id, project_id) to scope search.",
			},
		},
		"required": []string{"query"},
	}
}

func (t *SemanticSearchTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	if t.sm == nil || !t.sm.Enabled() {
		return nil, errors.New("semantic memory not configured")
	}

	rawQuery, _ := input["query"].(string)
	if rawQuery == "" {
		return nil, errors.New("query is required")
	}

	// 可选 top_k
	topK := 0
	if v, ok := input["top_k"].(float64); ok {
		topK = int(v)
	}

	// 可选 metadata
	meta := map[string]any{}
	if m, ok := input["metadata"].(map[string]any); ok && m != nil {
		meta = m
	}

	hits, err := t.sm.Search(ctx, rawQuery, meta, topK)
	if err != nil {
		return nil, err
	}

	// 简单序列化 hits 为 JSON 友好的结构
	out := make([]map[string]any, 0, len(hits))
	for _, h := range hits {
		out = append(out, map[string]any{
			"id":       h.ID,
			"score":    h.Score,
			"metadata": h.Metadata,
		})
	}
	return out, nil
}

func (t *SemanticSearchTool) Prompt() string {
	return "Use this tool to perform semantic search over previously indexed texts when keyword search is insufficient. " +
		"Provide a clear natural language query and optional metadata (user_id, project_id, resource_id) to narrow the search scope."
}
