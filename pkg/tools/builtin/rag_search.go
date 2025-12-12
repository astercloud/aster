package builtin

import (
	"context"
	"fmt"

	"github.com/astercloud/aster/pkg/memory"
	"github.com/astercloud/aster/pkg/tools"
)

// RAGSearchTool 基于 SemanticMemory 的 RAG 检索工具。
// 与 SemanticSearchTool 不同，此工具返回格式化的 Markdown 上下文，
// 可直接注入到 Prompt 中用于增强生成。
//
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
//	{
//	  "context": string (格式化的 Markdown 文本),
//	  "count": number (检索到的文档数量)
//	}
type RAGSearchTool struct {
	sm *memory.SemanticMemory
}

// NewRAGSearchTool 创建 RAG 检索工具实例
func NewRAGSearchTool(config map[string]any) (tools.Tool, error) {
	// 尝试从配置中获取 SemanticMemory 实例
	var sm *memory.SemanticMemory
	if config != nil {
		if smAny, exists := config["semantic_memory"]; exists {
			if smVal, ok := smAny.(*memory.SemanticMemory); ok {
				sm = smVal
			}
		}
	}

	return &RAGSearchTool{sm: sm}, nil
}

func (t *RAGSearchTool) Name() string {
	return "rag_search"
}

func (t *RAGSearchTool) Description() string {
	return "搜索相关知识库内容，返回格式化的上下文文本，用于增强生成回答（RAG）"
}

func (t *RAGSearchTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "自然语言查询文本",
			},
			"top_k": map[string]any{
				"type":        "integer",
				"description": "可选，返回的结果数量（默认值由配置决定）",
			},
			"metadata": map[string]any{
				"type":                 "object",
				"additionalProperties": true,
				"description":          "可选，用于过滤的元数据（如 user_id, project_id）",
			},
		},
		"required": []string{"query"},
	}
}

func (t *RAGSearchTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	// 检查 SemanticMemory 是否已配置
	if t.sm == nil || !t.sm.Enabled() {
		return nil, fmt.Errorf("semantic memory not configured or enabled")
	}

	// 提取查询文本
	rawQuery, ok := input["query"].(string)
	if !ok || rawQuery == "" {
		return nil, fmt.Errorf("query is required and must be a non-empty string")
	}

	// 可选参数：top_k
	topK := 0
	if v, ok := input["top_k"].(float64); ok {
		topK = int(v)
	}

	// 可选参数：metadata
	meta := map[string]any{}
	if m, ok := input["metadata"].(map[string]any); ok && m != nil {
		meta = m
	}

	// 执行检索并格式化为 Markdown
	context, err := t.sm.SearchAndFormat(ctx, rawQuery, meta, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to search and format: %w", err)
	}

	// 统计检索到的文档数量
	// 通过原始 Search 获取 hits 数量
	hits, err := t.sm.Search(ctx, rawQuery, meta, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to get search hits count: %w", err)
	}

	return map[string]any{
		"context": context,
		"count":   len(hits),
	}, nil
}

func (t *RAGSearchTool) Prompt() string {
	return `使用此工具从知识库中检索相关内容，增强回答的准确性和深度（RAG）。

功能特性：
- 基于语义相似度的智能检索
- 返回格式化的 Markdown 上下文
- 支持多租户和命名空间隔离
- 可配置返回结果数量

使用场景：
- 需要引用现有知识库内容回答问题
- 需要基于文档生成回答
- 需要查找相关背景信息

参数说明：
- query: 必需，自然语言查询文本
- top_k: 可选，返回的相关文档数量
- metadata: 可选，过滤条件（如 user_id, project_id）

返回格式：
返回包含格式化 Markdown 上下文的对象，可直接引用到回答中。`
}

// Examples 返回 RAG Search 工具的使用示例
func (t *RAGSearchTool) Examples() []tools.ToolExample {
	return []tools.ToolExample{
		{
			Description: "搜索关于 API 文档的相关内容",
			Input: map[string]any{
				"query": "如何使用认证 API",
			},
			Output: map[string]any{
				"context": "## Relevant Context\n\nFound 3 relevant documents:\n\n### 1. (Relevance: 92%)\n\nAPI 认证使用 Bearer Token...",
				"count":   3,
			},
		},
		{
			Description: "在特定项目中搜索配置信息",
			Input: map[string]any{
				"query": "数据库配置",
				"metadata": map[string]any{
					"project_id": "demo-project",
				},
			},
			Output: map[string]any{
				"context": "## Relevant Context\n\nFound 2 relevant documents:\n\n### 1. (Relevance: 88%)\n\n数据库连接配置...",
				"count":   2,
			},
		},
		{
			Description: "限制返回结果数量",
			Input: map[string]any{
				"query": "部署流程",
				"top_k": 3,
			},
			Output: map[string]any{
				"context": "## Relevant Context\n\nFound 3 relevant documents:\n\n### 1. (Relevance: 95%)\n\n部署步骤...",
				"count":   3,
			},
		},
	}
}
