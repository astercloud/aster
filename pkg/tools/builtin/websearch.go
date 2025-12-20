package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/astercloud/aster/pkg/tools"
)

// WebSearchTool 网络搜索工具 (使用 Tavily API)
// 设计参考: DeepAgents deepagents-cli/tools.py:web_search
type WebSearchTool struct {
	apiKey string
	client *http.Client
}

// NewWebSearchTool 创建网络搜索工具
func NewWebSearchTool(config map[string]any) (tools.Tool, error) {
	// 从环境变量读取 API key
	apiKey := os.Getenv("WF_TAVILY_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("TAVILY_API_KEY") // 兼容 DeepAgents 的环境变量名
	}

	return &WebSearchTool{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (t *WebSearchTool) Name() string {
	return "WebSearch"
}

func (t *WebSearchTool) Description() string {
	return "Search the web using Tavily for current information and documentation"
}

func (t *WebSearchTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The search query (be specific and detailed)",
			},
			"max_results": map[string]any{
				"type":        "integer",
				"description": "Number of results to return (default: 5)",
				"minimum":     1,
				"maximum":     10,
			},
			"topic": map[string]any{
				"type":        "string",
				"enum":        []string{"general", "news", "finance"},
				"description": "Search topic type - 'general' for most queries, 'news' for current events",
			},
			"include_raw_content": map[string]any{
				"type":        "boolean",
				"description": "Include full page content (warning: uses more tokens)",
			},
			"allowed_domains": map[string]any{
				"type":        "array",
				"description": "Only include results from these domains (e.g., ['github.com', 'stackoverflow.com'])",
				"items": map[string]any{
					"type": "string",
				},
			},
			"blocked_domains": map[string]any{
				"type":        "array",
				"description": "Never include results from these domains",
				"items": map[string]any{
					"type": "string",
				},
			},
		},
		"required": []string{"query"},
	}
}

func (t *WebSearchTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	// 1. 检查 API key
	if t.apiKey == "" {
		return map[string]any{
			"error": "Tavily API key not configured. Please set WF_TAVILY_API_KEY or TAVILY_API_KEY environment variable.",
			"query": input["query"],
		}, nil
	}

	// 2. 解析参数
	query, ok := input["query"].(string)
	if !ok || query == "" {
		return nil, errors.New("query must be a non-empty string")
	}

	maxResults := 5
	if mr, ok := input["max_results"].(float64); ok {
		maxResults = max(1, min(int(mr), 10))
	}

	topic := "general"
	if t, ok := input["topic"].(string); ok {
		topic = t
	}

	includeRawContent := false
	if irc, ok := input["include_raw_content"].(bool); ok {
		includeRawContent = irc
	}

	// 解析域名过滤参数
	var allowedDomains []string
	if ad, ok := input["allowed_domains"].([]any); ok {
		for _, d := range ad {
			if domain, ok := d.(string); ok && domain != "" {
				allowedDomains = append(allowedDomains, domain)
			}
		}
	}

	var blockedDomains []string
	if bd, ok := input["blocked_domains"].([]any); ok {
		for _, d := range bd {
			if domain, ok := d.(string); ok && domain != "" {
				blockedDomains = append(blockedDomains, domain)
			}
		}
	}

	// 3. 构建 Tavily API 请求
	requestBody := map[string]any{
		"api_key":             t.apiKey,
		"query":               query,
		"max_results":         maxResults,
		"include_raw_content": includeRawContent,
	}

	// 添加域名过滤（Tavily API 支持 include_domains 和 exclude_domains）
	if len(allowedDomains) > 0 {
		requestBody["include_domains"] = allowedDomains
	}
	if len(blockedDomains) > 0 {
		requestBody["exclude_domains"] = blockedDomains
	}

	// Tavily API的search_depth只接受"basic"或"advanced"
	// 我们的topic映射: general->basic, news->basic, finance->basic
	if topic != "" {
		requestBody["search_depth"] = "basic" // 默认使用basic模式
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return map[string]any{
			"error": fmt.Sprintf("failed to marshal request: %v", err),
			"query": query,
		}, nil
	}

	// 4. 发送请求到 Tavily API
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.tavily.com/search", bytes.NewReader(jsonData))
	if err != nil {
		return map[string]any{
			"error": fmt.Sprintf("failed to create request: %v", err),
			"query": query,
		}, nil
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return map[string]any{
			"error": fmt.Sprintf("web search error: %v", err),
			"query": query,
		}, nil
	}
	defer func() { _ = resp.Body.Close() }()

	// 5. 解析响应
	if resp.StatusCode != http.StatusOK {
		return map[string]any{
			"error": fmt.Sprintf("Tavily API returned status %d", resp.StatusCode),
			"query": query,
		}, nil
	}

	var searchResponse map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return map[string]any{
			"error": fmt.Sprintf("failed to decode response: %v", err),
			"query": query,
		}, nil
	}

	// 6. 返回结果(与 DeepAgents 格式对齐)
	// Tavily 响应格式: {"results": [...], "query": "..."}
	return searchResponse, nil
}

func (t *WebSearchTool) Prompt() string {
	return `Search the web using Tavily for current information and documentation.

This tool searches the web and returns relevant results. After receiving results,
you MUST synthesize the information into a natural, helpful response for the user.

Args:
- query: The search query (be specific and detailed)
- max_results: Number of results to return (default: 5, max: 10)
- topic: Search topic type
  - "general": for most queries (default)
  - "news": for current events
  - "finance": for financial information
- include_raw_content: Include full page content (warning: uses more tokens)

Returns:
Dictionary containing:
- results: List of search results, each with:
  - title: Page title
  - url: Page URL
  - content: Relevant excerpt from the page
  - score: Relevance score (0-1)
- query: The original search query

IMPORTANT: After using this tool:
1. Read through the 'content' field of each result
2. Extract relevant information that answers the user's question
3. Synthesize this into a clear, natural language response
4. Cite sources by mentioning the page titles or URLs
5. NEVER show the raw JSON to the user - always provide a formatted response

Configuration:
- Set WF_TAVILY_API_KEY or TAVILY_API_KEY environment variable
- Get your API key from: https://tavily.com

Example usage:
{
  "query": "latest developments in AI language models 2025",
  "max_results": 5,
  "topic": "general"
}`
}

// Examples 返回 WebSearch 工具的使用示例
// 实现 ExampleableTool 接口，帮助 LLM 更准确地调用工具
func (t *WebSearchTool) Examples() []tools.ToolExample {
	return []tools.ToolExample{
		{
			Description: "搜索技术文档",
			Input: map[string]any{
				"query":       "Go language context package usage",
				"max_results": 5,
			},
		},
		{
			Description: "搜索最新新闻",
			Input: map[string]any{
				"query":       "AI industry news 2025",
				"max_results": 10,
				"topic":       "news",
			},
		},
		{
			Description: "搜索金融信息并获取完整内容",
			Input: map[string]any{
				"query":               "Tesla stock price analysis",
				"topic":               "finance",
				"include_raw_content": true,
			},
		},
	}
}
