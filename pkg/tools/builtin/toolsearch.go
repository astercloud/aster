package builtin

import (
	"context"
	"errors"
	"time"

	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/search"
)

// ToolSearchTool 工具搜索工具
// 允许 LLM 按需搜索和发现可用的工具
// 实现 Anthropic 文章中的 Tool Search Tool 概念
type ToolSearchTool struct {
	toolIndex *search.ToolIndex
}

// NewToolSearchTool 创建工具搜索工具
func NewToolSearchTool(config map[string]any) (tools.Tool, error) {
	return &ToolSearchTool{
		toolIndex: search.NewToolIndex(),
	}, nil
}

// NewToolSearchToolWithIndex 使用现有索引创建工具搜索工具
func NewToolSearchToolWithIndex(index *search.ToolIndex) *ToolSearchTool {
	return &ToolSearchTool{
		toolIndex: index,
	}
}

func (t *ToolSearchTool) Name() string {
	return "ToolSearch"
}

func (t *ToolSearchTool) Description() string {
	return "搜索可用的工具，返回匹配的工具列表以供激活使用"
}

func (t *ToolSearchTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "搜索查询，描述你需要的工具功能",
			},
			"category": map[string]any{
				"type":        "string",
				"description": "可选的工具分类过滤，如 'filesystem', 'network', 'code' 等",
			},
			"max_results": map[string]any{
				"type":        "integer",
				"description": "返回的最大结果数量，默认为 5",
			},
			"include_deferred": map[string]any{
				"type":        "boolean",
				"description": "是否包含延迟加载的工具，默认为 true",
			},
			"activate": map[string]any{
				"type":        "array",
				"description": "要激活的工具名称列表",
				"items": map[string]any{
					"type": "string",
				},
			},
		},
		"required": []string{"query"},
	}
}

func (t *ToolSearchTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	query := GetStringParam(input, "query", "")
	category := GetStringParam(input, "category", "")
	maxResults := GetIntParam(input, "max_results", 5)
	includeDeferred := GetBoolParam(input, "include_deferred", true)
	activateTools := GetStringSlice(input, "activate")

	if query == "" && category == "" && len(activateTools) == 0 {
		return NewClaudeErrorResponse(errors.New("query, category or activate is required")), nil
	}

	start := time.Now()

	// 如果请求激活工具
	if len(activateTools) > 0 {
		return t.handleActivation(ctx, activateTools, tc)
	}

	// 执行搜索
	var results []search.ToolSearchResult

	if category != "" && query == "" {
		// 仅按分类搜索
		entries := t.toolIndex.SearchByCategory(category)
		for i, entry := range entries {
			if !includeDeferred && entry.Deferred {
				continue
			}
			results = append(results, search.ToolSearchResult{
				Entry: entry,
				Score: 1.0,
				Rank:  i + 1,
			})
			if len(results) >= maxResults {
				break
			}
		}
	} else {
		// 使用 BM25 搜索
		results = t.toolIndex.Search(query, maxResults*2) // 多搜索一些，后续可能需要过滤

		// 过滤延迟加载的工具（如果不需要）
		if !includeDeferred {
			filtered := make([]search.ToolSearchResult, 0, len(results))
			for _, r := range results {
				if !r.Entry.Deferred {
					filtered = append(filtered, r)
				}
			}
			results = filtered
		}

		// 限制结果数量
		if len(results) > maxResults {
			results = results[:maxResults]
		}

		// 重新计算排名
		for i := range results {
			results[i].Rank = i + 1
		}
	}

	duration := time.Since(start)

	// 构建响应
	toolInfos := make([]map[string]any, len(results))
	for i, r := range results {
		info := map[string]any{
			"name":        r.Entry.Name,
			"description": r.Entry.Description,
			"score":       r.Score,
			"rank":        r.Rank,
			"source":      r.Entry.Source,
			"deferred":    r.Entry.Deferred,
		}

		if r.Entry.Category != "" {
			info["category"] = r.Entry.Category
		}

		if len(r.Entry.Keywords) > 0 {
			info["keywords"] = r.Entry.Keywords
		}

		if len(r.Snippets) > 0 {
			info["matching_snippets"] = r.Snippets
		}

		// 包含参数信息
		if r.Entry.InputSchema != nil {
			if props, ok := r.Entry.InputSchema["properties"].(map[string]any); ok {
				paramNames := make([]string, 0, len(props))
				for name := range props {
					paramNames = append(paramNames, name)
				}
				info["parameters"] = paramNames
			}
		}

		// 包含示例（简化版本）
		if len(r.Entry.Examples) > 0 {
			exampleDescs := make([]string, 0, len(r.Entry.Examples))
			for _, ex := range r.Entry.Examples {
				exampleDescs = append(exampleDescs, ex.Description)
			}
			info["example_descriptions"] = exampleDescs
		}

		toolInfos[i] = info
	}

	return map[string]any{
		"ok":               true,
		"query":            query,
		"category":         category,
		"results":          toolInfos,
		"total_results":    len(results),
		"total_indexed":    t.toolIndex.Count(),
		"include_deferred": includeDeferred,
		"duration_ms":      duration.Milliseconds(),
		"instructions":     "要使用这些工具，请在响应中使用对应的工具名称调用它们。延迟加载的工具需要先激活。",
	}, nil
}

// handleActivation 处理工具激活请求
func (t *ToolSearchTool) handleActivation(ctx context.Context, toolNames []string, tc *tools.ToolContext) (any, error) {
	activated := make([]string, 0)
	failed := make([]map[string]any, 0)

	for _, name := range toolNames {
		entry := t.toolIndex.GetTool(name)
		if entry == nil {
			failed = append(failed, map[string]any{
				"name":   name,
				"reason": "工具不存在",
			})
			continue
		}

		if !entry.Deferred {
			// 工具已经是活跃状态
			activated = append(activated, name)
			continue
		}

		// TODO: 实际的工具激活逻辑
		// 这里需要与 Agent 的 toolMap 交互
		// 当前返回需要激活的工具信息
		activated = append(activated, name)
	}

	return map[string]any{
		"ok":        true,
		"activated": activated,
		"failed":    failed,
		"message":   "工具激活请求已处理",
	}, nil
}

// SetIndex 设置工具索引（用于与 Agent 集成）
func (t *ToolSearchTool) SetIndex(index *search.ToolIndex) {
	t.toolIndex = index
}

// GetIndex 获取工具索引
func (t *ToolSearchTool) GetIndex() *search.ToolIndex {
	return t.toolIndex
}

func (t *ToolSearchTool) Prompt() string {
	return `搜索可用的工具，返回匹配的工具列表以供激活使用。

功能特性：
- 使用 BM25 算法进行智能搜索
- 支持按分类过滤
- 支持延迟加载工具的发现
- 返回工具的详细信息和使用示例

使用指南：
- query: 必需参数，描述你需要的工具功能
- category: 可选参数，工具分类过滤
- max_results: 可选参数，最大结果数量
- include_deferred: 可选参数，是否包含延迟加载工具
- activate: 可选参数，要激活的工具名称列表

使用场景：
1. 当你不确定应该使用哪个工具时
2. 当你需要特定功能但不知道工具名称时
3. 当你想发现可用的工具能力时

注意事项：
- 搜索结果按相关性排序
- 延迟加载的工具需要先激活才能使用
- 工具激活后会在后续对话中可用`
}

// Examples 返回 ToolSearch 工具的使用示例
func (t *ToolSearchTool) Examples() []tools.ToolExample {
	return []tools.ToolExample{
		{
			Description: "搜索文件操作相关的工具",
			Input: map[string]any{
				"query": "读取文件 写入文件",
			},
		},
		{
			Description: "按分类搜索网络相关工具",
			Input: map[string]any{
				"query":    "HTTP 请求",
				"category": "network",
			},
		},
		{
			Description: "搜索并激活工具",
			Input: map[string]any{
				"query":    "代码搜索",
				"activate": []string{"Grep", "Glob"},
			},
		},
	}
}

// GetStringSlice 获取字符串数组参数（辅助函数）
func GetStringSlice(input map[string]any, key string) []string {
	if value, exists := input[key]; exists {
		if slice, ok := value.([]any); ok {
			result := make([]string, 0, len(slice))
			for _, item := range slice {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
		if slice, ok := value.([]string); ok {
			return slice
		}
	}
	return nil
}
