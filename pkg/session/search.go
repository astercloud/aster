package session

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/types"
)

// SearcherConfig Session 搜索器配置
type SearcherConfig struct {
	Store store.Store
}

// Searcher Session 搜索器
type Searcher struct {
	store store.Store
}

// NewSearcher 创建 Session 搜索器
func NewSearcher(config SearcherConfig) *Searcher {
	return &Searcher{
		store: config.Store,
	}
}

// SearchOptions 搜索选项
type SearchOptions struct {
	Query      string    // 搜索关键词
	AgentID    string    // 限定 Agent ID
	StartTime  time.Time // 开始时间
	EndTime    time.Time // 结束时间
	Limit      int       // 返回结果数量限制
	Offset     int       // 偏移量
	MatchMode  string    // 匹配模式: "exact", "contains", "fuzzy"
	OnlyUser   bool      // 只搜索用户消息
	OnlyAssistant bool   // 只搜索助手消息
}

// SearchResult 搜索结果
type SearchResult struct {
	AgentID     string         `json:"agent_id"`
	MessageIndex int           `json:"message_index"`
	Message     types.Message  `json:"message"`
	Snippet     string         `json:"snippet"`
	Relevance   float64        `json:"relevance"`
	Timestamp   time.Time      `json:"timestamp"`
}

// SearchHistory 搜索历史消息
func (s *Searcher) SearchHistory(ctx context.Context, opts SearchOptions) ([]SearchResult, error) {
	if opts.Query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	if opts.Limit <= 0 {
		opts.Limit = 20
	}

	if opts.MatchMode == "" {
		opts.MatchMode = "contains"
	}

	// 加载消息
	messages, err := s.store.LoadMessages(ctx, opts.AgentID)
	if err != nil {
		return nil, fmt.Errorf("load messages: %w", err)
	}

	// 执行搜索
	results := []SearchResult{}
	queryLower := strings.ToLower(opts.Query)

	for i, msg := range messages {
		// 角色过滤
		if opts.OnlyUser && msg.Role != types.MessageRoleUser {
			continue
		}
		if opts.OnlyAssistant && msg.Role != types.MessageRoleAssistant {
			continue
		}

		// 提取文本内容
		content := extractTextContent(msg)
		contentLower := strings.ToLower(content)

		// 匹配检查
		matched := false
		relevance := 0.0

		switch opts.MatchMode {
		case "exact":
			matched = strings.Contains(content, opts.Query)
			if matched {
				relevance = 1.0
			}
		case "contains":
			matched = strings.Contains(contentLower, queryLower)
			if matched {
				relevance = calculateRelevance(contentLower, queryLower)
			}
		case "fuzzy":
			relevance = fuzzyMatch(contentLower, queryLower)
			matched = relevance > 0.3
		}

		if matched {
			snippet := generateSnippet(content, opts.Query, 150)
			
			results = append(results, SearchResult{
				AgentID:      opts.AgentID,
				MessageIndex: i,
				Message:      msg,
				Snippet:      snippet,
				Relevance:    relevance,
				Timestamp:    time.Now(), // TODO: 从消息中提取实际时间戳
			})
		}
	}

	// 按相关性排序
	sortByRelevance(results)

	// 应用分页
	start := opts.Offset
	end := opts.Offset + opts.Limit

	if start >= len(results) {
		return []SearchResult{}, nil
	}

	if end > len(results) {
		end = len(results)
	}

	return results[start:end], nil
}

// SearchAcrossSessions 跨 Session 搜索
func (s *Searcher) SearchAcrossSessions(ctx context.Context, agentIDs []string, query string, limit int) ([]SearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	if limit <= 0 {
		limit = 50
	}

	allResults := []SearchResult{}

	for _, agentID := range agentIDs {
		opts := SearchOptions{
			Query:   query,
			AgentID: agentID,
			Limit:   limit,
		}

		results, err := s.SearchHistory(ctx, opts)
		if err != nil {
			// 记录错误但继续搜索其他 session
			continue
		}

		allResults = append(allResults, results...)
	}

	// 按相关性排序
	sortByRelevance(allResults)

	// 限制结果数量
	if len(allResults) > limit {
		allResults = allResults[:limit]
	}

	return allResults, nil
}

// FindSimilarMessages 查找相似消息
func (s *Searcher) FindSimilarMessages(ctx context.Context, agentID string, referenceMessage types.Message, limit int) ([]SearchResult, error) {
	content := extractTextContent(referenceMessage)
	
	opts := SearchOptions{
		Query:     content,
		AgentID:   agentID,
		Limit:     limit,
		MatchMode: "fuzzy",
	}

	return s.SearchHistory(ctx, opts)
}

// calculateRelevance 计算相关性得分
func calculateRelevance(content, query string) float64 {
	// 简单的相关性计算：基于查询词出现次数
	count := strings.Count(content, query)
	if count == 0 {
		return 0.0
	}

	// 归一化得分
	score := float64(count) / float64(len(strings.Fields(content)))
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// fuzzyMatch 模糊匹配
func fuzzyMatch(content, query string) float64 {
	queryWords := strings.Fields(query)
	if len(queryWords) == 0 {
		return 0.0
	}

	matchedWords := 0
	for _, word := range queryWords {
		if strings.Contains(content, word) {
			matchedWords++
		}
	}

	return float64(matchedWords) / float64(len(queryWords))
}

// generateSnippet 生成摘要片段
func generateSnippet(content, query string, maxLen int) string {
	queryLower := strings.ToLower(query)
	contentLower := strings.ToLower(content)

	// 查找查询词位置
	index := strings.Index(contentLower, queryLower)
	if index == -1 {
		// 如果没找到，返回开头部分
		if len(content) <= maxLen {
			return content
		}
		return content[:maxLen] + "..."
	}

	// 计算片段范围
	start := index - maxLen/2
	if start < 0 {
		start = 0
	}

	end := start + maxLen
	if end > len(content) {
		end = len(content)
	}

	snippet := content[start:end]

	// 添加省略号
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(content) {
		snippet = snippet + "..."
	}

	return snippet
}

// sortByRelevance 按相关性排序
func sortByRelevance(results []SearchResult) {
	// 简单的冒泡排序（对于小数据集足够）
	n := len(results)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if results[j].Relevance < results[j+1].Relevance {
				results[j], results[j+1] = results[j+1], results[j]
			}
		}
	}
}
