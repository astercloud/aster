package search

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/astercloud/aster/pkg/tools"
)

// ToolIndex 工具索引系统
// 使用 BM25 算法对工具进行索引和搜索
type ToolIndex struct {
	mu         sync.RWMutex
	bm25       *BM25
	toolMap    map[string]ToolIndexEntry // 工具名称 -> 索引条目
	categories map[string][]string       // 分类 -> 工具名称列表
}

// ToolIndexEntry 工具索引条目
type ToolIndexEntry struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	InputSchema map[string]any      `json:"input_schema"`
	Category    string              `json:"category,omitempty"`
	Keywords    []string            `json:"keywords,omitempty"`
	Examples    []tools.ToolExample `json:"examples,omitempty"`
	Deferred    bool                `json:"deferred"`
	Source      string              `json:"source"` // "builtin", "mcp", "custom"
	Metadata    map[string]any      `json:"metadata,omitempty"`
}

// ToolSearchResult 工具搜索结果
type ToolSearchResult struct {
	Entry    ToolIndexEntry `json:"entry"`
	Score    float64        `json:"score"`
	Rank     int            `json:"rank"`
	Snippets []string       `json:"snippets,omitempty"` // 匹配的文本片段
}

// NewToolIndex 创建工具索引
func NewToolIndex() *ToolIndex {
	return &ToolIndex{
		bm25:       NewBM25WithDefaults(),
		toolMap:    make(map[string]ToolIndexEntry),
		categories: make(map[string][]string),
	}
}

// IndexTool 索引单个工具
func (ti *ToolIndex) IndexTool(tool tools.Tool, source string, deferred bool) error {
	ti.mu.Lock()
	defer ti.mu.Unlock()

	// 创建索引条目
	entry := ToolIndexEntry{
		Name:        tool.Name(),
		Description: tool.Description(),
		InputSchema: tool.InputSchema(),
		Source:      source,
		Deferred:    deferred,
		Metadata:    make(map[string]any),
	}

	// 检查是否实现了 ExampleableTool 接口
	if exampleable, ok := tool.(tools.ExampleableTool); ok {
		entry.Examples = exampleable.Examples()
	}

	// 检查是否实现了 DeferrableTool 接口
	if deferrable, ok := tool.(tools.DeferrableTool); ok {
		config := deferrable.DeferConfig()
		if config != nil {
			entry.Category = config.Category
			entry.Keywords = config.Keywords
			entry.Deferred = config.DeferLoading
		}
	}

	// 构建可搜索的文档内容
	content := ti.buildSearchableContent(entry)

	// 添加到 BM25 索引
	doc := Document{
		ID:      entry.Name,
		Content: content,
		Metadata: map[string]any{
			"source":   source,
			"deferred": deferred,
		},
	}

	// 如果已存在，先删除旧的
	if _, exists := ti.toolMap[entry.Name]; exists {
		ti.bm25.RemoveDocument(entry.Name)
	}

	ti.bm25.AddDocument(doc)
	ti.toolMap[entry.Name] = entry

	// 更新分类索引
	if entry.Category != "" {
		ti.addToCategory(entry.Category, entry.Name)
	}

	return nil
}

// IndexToolEntry 直接索引条目（用于 MCP 延迟加载的工具）
func (ti *ToolIndex) IndexToolEntry(entry ToolIndexEntry) error {
	ti.mu.Lock()
	defer ti.mu.Unlock()

	// 构建可搜索的文档内容
	content := ti.buildSearchableContent(entry)

	// 添加到 BM25 索引
	doc := Document{
		ID:      entry.Name,
		Content: content,
		Metadata: map[string]any{
			"source":   entry.Source,
			"deferred": entry.Deferred,
		},
	}

	// 如果已存在，先删除旧的
	if _, exists := ti.toolMap[entry.Name]; exists {
		ti.bm25.RemoveDocument(entry.Name)
	}

	ti.bm25.AddDocument(doc)
	ti.toolMap[entry.Name] = entry

	// 更新分类索引
	if entry.Category != "" {
		ti.addToCategory(entry.Category, entry.Name)
	}

	return nil
}

// Search 搜索工具
func (ti *ToolIndex) Search(query string, topK int) []ToolSearchResult {
	ti.mu.RLock()
	defer ti.mu.RUnlock()

	if topK <= 0 {
		topK = 10
	}

	// 执行 BM25 搜索
	results := ti.bm25.Search(query, topK)

	// 转换为工具搜索结果
	toolResults := make([]ToolSearchResult, 0, len(results))
	for _, r := range results {
		if entry, exists := ti.toolMap[r.Document.ID]; exists {
			toolResult := ToolSearchResult{
				Entry:    entry,
				Score:    r.Score,
				Rank:     r.Rank,
				Snippets: ti.extractSnippets(entry, query),
			}
			toolResults = append(toolResults, toolResult)
		}
	}

	return toolResults
}

// SearchByCategory 按分类搜索
func (ti *ToolIndex) SearchByCategory(category string) []ToolIndexEntry {
	ti.mu.RLock()
	defer ti.mu.RUnlock()

	toolNames, exists := ti.categories[category]
	if !exists {
		return nil
	}

	entries := make([]ToolIndexEntry, 0, len(toolNames))
	for _, name := range toolNames {
		if entry, exists := ti.toolMap[name]; exists {
			entries = append(entries, entry)
		}
	}

	return entries
}

// GetTool 获取工具条目
func (ti *ToolIndex) GetTool(name string) *ToolIndexEntry {
	ti.mu.RLock()
	defer ti.mu.RUnlock()

	if entry, exists := ti.toolMap[name]; exists {
		return &entry
	}
	return nil
}

// GetAllTools 获取所有工具条目
func (ti *ToolIndex) GetAllTools() []ToolIndexEntry {
	ti.mu.RLock()
	defer ti.mu.RUnlock()

	entries := make([]ToolIndexEntry, 0, len(ti.toolMap))
	for _, entry := range ti.toolMap {
		entries = append(entries, entry)
	}
	return entries
}

// GetDeferredTools 获取所有延迟加载的工具
func (ti *ToolIndex) GetDeferredTools() []ToolIndexEntry {
	ti.mu.RLock()
	defer ti.mu.RUnlock()

	entries := make([]ToolIndexEntry, 0)
	for _, entry := range ti.toolMap {
		if entry.Deferred {
			entries = append(entries, entry)
		}
	}
	return entries
}

// GetActiveTools 获取所有活跃的工具（非延迟加载）
func (ti *ToolIndex) GetActiveTools() []ToolIndexEntry {
	ti.mu.RLock()
	defer ti.mu.RUnlock()

	entries := make([]ToolIndexEntry, 0)
	for _, entry := range ti.toolMap {
		if !entry.Deferred {
			entries = append(entries, entry)
		}
	}
	return entries
}

// RemoveTool 从索引中移除工具
func (ti *ToolIndex) RemoveTool(name string) bool {
	ti.mu.Lock()
	defer ti.mu.Unlock()

	entry, exists := ti.toolMap[name]
	if !exists {
		return false
	}

	// 从 BM25 索引中移除
	ti.bm25.RemoveDocument(name)

	// 从分类中移除
	if entry.Category != "" {
		ti.removeFromCategory(entry.Category, name)
	}

	// 从工具映射中移除
	delete(ti.toolMap, name)

	return true
}

// Count 返回索引中的工具数量
func (ti *ToolIndex) Count() int {
	ti.mu.RLock()
	defer ti.mu.RUnlock()
	return len(ti.toolMap)
}

// Clear 清空索引
func (ti *ToolIndex) Clear() {
	ti.mu.Lock()
	defer ti.mu.Unlock()

	ti.bm25.Clear()
	ti.toolMap = make(map[string]ToolIndexEntry)
	ti.categories = make(map[string][]string)
}

// buildSearchableContent 构建可搜索的文档内容
func (ti *ToolIndex) buildSearchableContent(entry ToolIndexEntry) string {
	var parts []string

	// 添加工具名称（权重较高，多次添加）
	parts = append(parts, entry.Name, entry.Name, entry.Name)

	// 添加描述
	parts = append(parts, entry.Description)

	// 添加分类
	if entry.Category != "" {
		parts = append(parts, entry.Category)
	}

	// 添加关键词
	parts = append(parts, entry.Keywords...)

	// 添加参数名称
	if entry.InputSchema != nil {
		if props, ok := entry.InputSchema["properties"].(map[string]any); ok {
			for propName, propDef := range props {
				parts = append(parts, propName)
				if propMap, ok := propDef.(map[string]any); ok {
					if desc, ok := propMap["description"].(string); ok {
						parts = append(parts, desc)
					}
				}
			}
		}
	}

	// 添加示例描述
	for _, example := range entry.Examples {
		if example.Description != "" {
			parts = append(parts, example.Description)
		}
	}

	return strings.Join(parts, " ")
}

// extractSnippets 提取匹配的文本片段
func (ti *ToolIndex) extractSnippets(entry ToolIndexEntry, query string) []string {
	snippets := make([]string, 0)
	queryLower := strings.ToLower(query)

	// 检查描述
	if strings.Contains(strings.ToLower(entry.Description), queryLower) {
		snippets = append(snippets, entry.Description)
	}

	// 检查示例描述
	for _, example := range entry.Examples {
		if strings.Contains(strings.ToLower(example.Description), queryLower) {
			snippets = append(snippets, example.Description)
		}
	}

	return snippets
}

// addToCategory 添加工具到分类
func (ti *ToolIndex) addToCategory(category, toolName string) {
	if ti.categories[category] == nil {
		ti.categories[category] = make([]string, 0)
	}
	// 检查是否已存在
	if slices.Contains(ti.categories[category], toolName) {
		return
	}
	ti.categories[category] = append(ti.categories[category], toolName)
}

// removeFromCategory 从分类中移除工具
func (ti *ToolIndex) removeFromCategory(category, toolName string) {
	names := ti.categories[category]
	for i, name := range names {
		if name == toolName {
			ti.categories[category] = append(names[:i], names[i+1:]...)
			break
		}
	}
}

// Export 导出索引为 JSON
func (ti *ToolIndex) Export() ([]byte, error) {
	ti.mu.RLock()
	defer ti.mu.RUnlock()

	data := map[string]any{
		"tools":      ti.toolMap,
		"categories": ti.categories,
		"count":      len(ti.toolMap),
	}

	return json.Marshal(data)
}

// Import 从 JSON 导入索引
func (ti *ToolIndex) Import(data []byte) error {
	ti.mu.Lock()
	defer ti.mu.Unlock()

	var importData struct {
		Tools      map[string]ToolIndexEntry `json:"tools"`
		Categories map[string][]string       `json:"categories"`
	}

	if err := json.Unmarshal(data, &importData); err != nil {
		return fmt.Errorf("failed to unmarshal index data: %w", err)
	}

	// 清空现有索引
	ti.bm25.Clear()
	ti.toolMap = make(map[string]ToolIndexEntry)
	ti.categories = make(map[string][]string)

	// 重建索引
	for name, entry := range importData.Tools {
		ti.toolMap[name] = entry
		content := ti.buildSearchableContent(entry)
		ti.bm25.AddDocument(Document{
			ID:      name,
			Content: content,
			Metadata: map[string]any{
				"source":   entry.Source,
				"deferred": entry.Deferred,
			},
		})
	}

	ti.categories = importData.Categories

	return nil
}
