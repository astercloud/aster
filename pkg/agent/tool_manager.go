package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/search"
)

var toolMgrLog = logging.ForComponent("ToolManager")

// ToolManager 工具管理器
// 支持动态工具激活和工具搜索
type ToolManager struct {
	mu sync.RWMutex

	// 工具索引
	index *search.ToolIndex

	// 活跃工具（已加载到 Agent 的工具）
	activeTools map[string]tools.Tool

	// 延迟工具（已索引但未加载的工具）
	deferredTools map[string]search.ToolIndexEntry

	// 工具注册表（用于创建工具实例）
	registry *tools.Registry

	// 核心工具列表（始终活跃的工具）
	coreTools []string
}

// NewToolManager 创建工具管理器
func NewToolManager(registry *tools.Registry) *ToolManager {
	return &ToolManager{
		index:         search.NewToolIndex(),
		activeTools:   make(map[string]tools.Tool),
		deferredTools: make(map[string]search.ToolIndexEntry),
		registry:      registry,
		coreTools:     []string{"ToolSearch"}, // ToolSearch 始终活跃
	}
}

// Initialize 初始化工具管理器
// 将所有工具添加到索引，并根据配置决定哪些工具活跃
func (tm *ToolManager) Initialize(toolNames []string, activeByDefault bool) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	ctx := context.Background() // 用于日志记录

	for _, name := range toolNames {
		tool, err := tm.registry.Create(name, nil)
		if err != nil {
			toolMgrLog.Warn(ctx, "failed to create tool", map[string]any{"name": name, "error": err})
			continue
		}

		// 检查是否是核心工具（始终活跃）
		isCore := tm.isCoreToolLocked(name)

		// 决定是否活跃
		isActive := activeByDefault || isCore

		// 检查是否实现了 DeferrableTool 接口
		if deferrable, ok := tool.(tools.DeferrableTool); ok {
			config := deferrable.DeferConfig()
			if config != nil && config.DeferLoading && !isCore {
				isActive = false
			}
		}

		// 添加到索引
		source := "builtin"
		if err := tm.index.IndexTool(tool, source, !isActive); err != nil {
			toolMgrLog.Warn(ctx, "failed to index tool", map[string]any{"name": name, "error": err})
			continue
		}

		if isActive {
			tm.activeTools[name] = tool
			toolMgrLog.Debug(ctx, "tool loaded (active)", map[string]any{"name": name})
		} else {
			entry := tm.index.GetTool(name)
			if entry != nil {
				tm.deferredTools[name] = *entry
				toolMgrLog.Debug(ctx, "tool indexed (deferred)", map[string]any{"name": name})
			}
		}
	}

	toolMgrLog.Info(ctx, "initialized", map[string]any{"active_tools": len(tm.activeTools), "deferred_tools": len(tm.deferredTools)})

	return nil
}

// isCoreToolLocked 检查是否是核心工具（需要持有锁）
func (tm *ToolManager) isCoreToolLocked(name string) bool {
	for _, core := range tm.coreTools {
		if core == name {
			return true
		}
	}
	return false
}

// IsCoreTools 检查是否是核心工具
func (tm *ToolManager) IsCoreTool(name string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.isCoreToolLocked(name)
}

// GetActiveTools 获取所有活跃工具
func (tm *ToolManager) GetActiveTools() map[string]tools.Tool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	result := make(map[string]tools.Tool, len(tm.activeTools))
	for k, v := range tm.activeTools {
		result[k] = v
	}
	return result
}

// GetActiveTool 获取单个活跃工具
func (tm *ToolManager) GetActiveTool(name string) (tools.Tool, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	tool, ok := tm.activeTools[name]
	return tool, ok
}

// SearchTools 搜索工具
func (tm *ToolManager) SearchTools(query string, topK int) []search.ToolSearchResult {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.index.Search(query, topK)
}

// ActivateTool 激活延迟工具
func (tm *ToolManager) ActivateTool(name string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	ctx := context.Background() // 用于日志记录

	// 检查是否已经活跃
	if _, ok := tm.activeTools[name]; ok {
		return nil // 已经活跃
	}

	// 检查是否在延迟工具列表中
	entry, ok := tm.deferredTools[name]
	if !ok {
		return fmt.Errorf("tool not found: %s", name)
	}

	// 创建工具实例
	tool, err := tm.registry.Create(name, nil)
	if err != nil {
		return fmt.Errorf("failed to create tool: %w", err)
	}

	// 移动到活跃工具列表
	tm.activeTools[name] = tool
	delete(tm.deferredTools, name)

	// 更新索引中的延迟状态
	entry.Deferred = false
	if err := tm.index.IndexToolEntry(entry); err != nil {
		toolMgrLog.Warn(ctx, "failed to update tool index", map[string]any{"error": err})
	}

	toolMgrLog.Debug(ctx, "tool activated", map[string]any{"name": name})
	return nil
}

// ActivateTools 批量激活工具
func (tm *ToolManager) ActivateTools(names []string) (activated []string, failed map[string]error) {
	activated = make([]string, 0, len(names))
	failed = make(map[string]error)

	for _, name := range names {
		if err := tm.ActivateTool(name); err != nil {
			failed[name] = err
		} else {
			activated = append(activated, name)
		}
	}

	return activated, failed
}

// DeactivateTool 停用工具（移回延迟状态）
func (tm *ToolManager) DeactivateTool(name string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	ctx := context.Background() // 用于日志记录

	// 不能停用核心工具
	if tm.isCoreToolLocked(name) {
		return fmt.Errorf("cannot deactivate core tool: %s", name)
	}

	// 检查是否在活跃工具列表中
	tool, ok := tm.activeTools[name]
	if !ok {
		return nil // 已经不活跃
	}

	// 移动到延迟工具列表
	entry := tm.index.GetTool(name)
	if entry != nil {
		entry.Deferred = true
		tm.deferredTools[name] = *entry
		if err := tm.index.IndexToolEntry(*entry); err != nil {
			toolMgrLog.Warn(ctx, "failed to update tool index", map[string]any{"error": err})
		}
	}
	delete(tm.activeTools, name)

	toolMgrLog.Debug(ctx, "tool deactivated", map[string]any{"name": name, "tool_type": fmt.Sprintf("%T", tool)})
	return nil
}

// AddTool 添加新工具
func (tm *ToolManager) AddTool(tool tools.Tool, source string, deferred bool) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	name := tool.Name()

	// 添加到索引
	if err := tm.index.IndexTool(tool, source, deferred); err != nil {
		return fmt.Errorf("failed to index tool: %w", err)
	}

	if deferred {
		entry := tm.index.GetTool(name)
		if entry != nil {
			tm.deferredTools[name] = *entry
		}
	} else {
		tm.activeTools[name] = tool
	}

	return nil
}

// AddToolEntry 添加工具条目（用于 MCP 延迟加载）
func (tm *ToolManager) AddToolEntry(entry search.ToolIndexEntry) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if err := tm.index.IndexToolEntry(entry); err != nil {
		return fmt.Errorf("failed to index tool entry: %w", err)
	}

	if entry.Deferred {
		tm.deferredTools[entry.Name] = entry
	}

	return nil
}

// RemoveTool 移除工具
func (tm *ToolManager) RemoveTool(name string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 不能移除核心工具
	if tm.isCoreToolLocked(name) {
		return fmt.Errorf("cannot remove core tool: %s", name)
	}

	// 从索引中移除
	tm.index.RemoveTool(name)

	// 从活跃和延迟列表中移除
	delete(tm.activeTools, name)
	delete(tm.deferredTools, name)

	return nil
}

// GetIndex 获取工具索引
func (tm *ToolManager) GetIndex() *search.ToolIndex {
	return tm.index
}

// Stats 返回统计信息
func (tm *ToolManager) Stats() map[string]any {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return map[string]any{
		"active_count":   len(tm.activeTools),
		"deferred_count": len(tm.deferredTools),
		"total_indexed":  tm.index.Count(),
		"core_tools":     tm.coreTools,
	}
}
