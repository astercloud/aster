package builtin

import (
	"bytes"
	"context"
	"encoding/json"
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
func NewWebSearchTool(config map[string]interface{}) (tools.Tool, error) {
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

func (t *WebSearchTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The search query (be specific and detailed)",
			},
			"max_results": map[string]interface{}{
				"type":        "integer",
				"description": "Number of results to return (default: 5)",
				"minimum":     1,
				"maximum":     10,
			},
			"topic": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"general", "news", "finance"},
				"description": "Search topic type - 'general' for most queries, 'news' for current events",
			},
			"include_raw_content": map[string]interface{}{
				"type":        "boolean",
				"description": "Include full page content (warning: uses more tokens)",
			},
		},
		"required": []string{"query"},
	}
}

func (t *WebSearchTool) Execute(ctx context.Context, input map[string]interface{}, tc *tools.ToolContext) (interface{}, error) {
	// 1. 检查 API key
	if t.apiKey == "" {
		return map[string]interface{}{
			"error": "Tavily API key not configured. Please set WF_TAVILY_API_KEY or TAVILY_API_KEY environment variable.",
			"query": input["query"],
		}, nil
	}

	// 2. 解析参数
	query, ok := input["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query must be a non-empty string")
	}

	maxResults := 5
	if mr, ok := input["max_results"].(float64); ok {
		maxResults = int(mr)
		if maxResults < 1 {
			maxResults = 1
		}
		if maxResults > 10 {
			maxResults = 10
		}
	}

	topic := "general"
	if t, ok := input["topic"].(string); ok {
		topic = t
	}

	includeRawContent := false
	if irc, ok := input["include_raw_content"].(bool); ok {
		includeRawContent = irc
	}

	// 3. 构建 Tavily API 请求
	requestBody := map[string]interface{}{
		"api_key":             t.apiKey,
		"query":               query,
		"max_results":         maxResults,
		"search_depth":        topic, // Tavily 使用 search_depth 而非 topic
		"include_raw_content": includeRawContent,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("failed to marshal request: %v", err),
			"query": query,
		}, nil
	}

	// 4. 发送请求到 Tavily API
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.tavily.com/search", bytes.NewReader(jsonData))
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("failed to create request: %v", err),
			"query": query,
		}, nil
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("web search error: %v", err),
			"query": query,
		}, nil
	}
	defer func() { _ = resp.Body.Close() }()

	// 5. 解析响应
	if resp.StatusCode != http.StatusOK {
		return map[string]interface{}{
			"error": fmt.Sprintf("Tavily API returned status %d", resp.StatusCode),
			"query": query,
		}, nil
	}

	var searchResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return map[string]interface{}{
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
