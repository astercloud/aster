package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/astercloud/aster/pkg/types"
)

// PlanModeState Plan 模式状态
type PlanModeState struct {
	Active       bool             // 是否处于 Plan 模式
	PlanID       string           // 当前计划 ID
	PlanFilePath string           // 计划文件路径
	AllowedTools map[string]bool  // 允许的工具白名单
	Reason       string           // 进入 Plan 模式的原因
	Constraints  *PlanConstraints // 约束配置
}

// PlanConstraints Plan 模式约束
type PlanConstraints struct {
	ReadOnly          bool     // 只读模式
	AllowedWritePaths []string // 允许写入的路径（仅计划文件）
	DisabledTools     []string // 禁用的工具
	RequireApproval   bool     // 退出时需要用户批准
}

// PlanModeManager Plan 模式管理器
// 管理 Agent 的 Plan 模式状态和工具约束
type PlanModeManager struct {
	mu    sync.RWMutex
	state *PlanModeState
}

// NewPlanModeManager 创建 Plan 模式管理器
func NewPlanModeManager() *PlanModeManager {
	return &PlanModeManager{
		state: &PlanModeState{
			Active:       false,
			AllowedTools: make(map[string]bool),
		},
	}
}

// EnterPlanMode 进入 Plan 模式
func (m *PlanModeManager) EnterPlanMode(planID, planFilePath, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 定义 Plan 模式允许的工具
	allowedTools := map[string]bool{
		"Read":            true,
		"Glob":            true,
		"Grep":            true,
		"WebFetch":        true,
		"WebSearch":       true,
		"AskUserQuestion": true,
		"Write":           true, // 仅限计划文件，在 ValidateToolCall 中检查
		"ExitPlanMode":    true,
		"Task":            true, // 允许启动 Explore 子代理
	}

	m.state = &PlanModeState{
		Active:       true,
		PlanID:       planID,
		PlanFilePath: planFilePath,
		AllowedTools: allowedTools,
		Reason:       reason,
		Constraints: &PlanConstraints{
			ReadOnly:          true,
			AllowedWritePaths: []string{planFilePath},
			DisabledTools: []string{
				"Bash",
				"Edit",
				"Delete",
				"Move",
				"Copy",
			},
			RequireApproval: true,
		},
	}
}

// ExitPlanMode 退出 Plan 模式
func (m *PlanModeManager) ExitPlanMode() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.state = &PlanModeState{
		Active:       false,
		AllowedTools: make(map[string]bool),
	}
}

// IsActive 检查是否处于 Plan 模式
func (m *PlanModeManager) IsActive() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state.Active
}

// GetState 获取当前状态
func (m *PlanModeManager) GetState() *PlanModeState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 返回副本
	if m.state == nil {
		return &PlanModeState{Active: false}
	}

	stateCopy := *m.state
	return &stateCopy
}

// ValidateToolCall 验证工具调用是否允许
// 返回 (allowed, reason)
func (m *PlanModeManager) ValidateToolCall(toolName string, input map[string]any) (bool, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 如果不在 Plan 模式，允许所有工具
	if !m.state.Active {
		return true, ""
	}

	// 检查工具是否在白名单中
	if !m.state.AllowedTools[toolName] {
		return false, fmt.Sprintf("Tool '%s' is not allowed in Plan Mode. Allowed tools: %s",
			toolName, m.getAllowedToolsList())
	}

	// 特殊检查：Write 工具只能写入计划文件
	if toolName == "Write" {
		return m.validateWriteCall(input)
	}

	// 特殊检查：Task 工具只能启动 Explore 类型
	if toolName == "Task" {
		return m.validateTaskCall(input)
	}

	return true, ""
}

// validateWriteCall 验证 Write 工具调用
func (m *PlanModeManager) validateWriteCall(input map[string]any) (bool, string) {
	path, ok := input["path"].(string)
	if !ok {
		path, _ = input["file_path"].(string)
	}

	if path == "" {
		return false, "Write tool requires a path"
	}

	// 检查是否是允许的写入路径
	for _, allowedPath := range m.state.Constraints.AllowedWritePaths {
		if path == allowedPath || strings.HasPrefix(path, ".aster/plans/") {
			return true, ""
		}
	}

	return false, fmt.Sprintf("In Plan Mode, Write is only allowed to plan files. Allowed path: %s",
		m.state.PlanFilePath)
}

// validateTaskCall 验证 Task 工具调用
func (m *PlanModeManager) validateTaskCall(input map[string]any) (bool, string) {
	subagentType, ok := input["subagent_type"].(string)
	if !ok {
		return false, "Task tool requires subagent_type"
	}

	// Plan 模式下只允许 Explore 类型的子代理
	allowedSubagents := []string{"Explore"}
	for _, allowed := range allowedSubagents {
		if subagentType == allowed {
			return true, ""
		}
	}

	return false, fmt.Sprintf("In Plan Mode, only Explore subagent is allowed. Got: %s", subagentType)
}

// getAllowedToolsList 获取允许的工具列表字符串
func (m *PlanModeManager) getAllowedToolsList() string {
	tools := make([]string, 0, len(m.state.AllowedTools))
	for tool := range m.state.AllowedTools {
		tools = append(tools, tool)
	}
	return strings.Join(tools, ", ")
}

// ToolCallDeniedError 工具调用被拒绝错误
type ToolCallDeniedError struct {
	ToolName string
	Reason   string
}

func (e *ToolCallDeniedError) Error() string {
	return fmt.Sprintf("Tool call denied: %s - %s", e.ToolName, e.Reason)
}

// Agent Plan Mode Methods

// EnterPlanMode 让 Agent 进入 Plan 模式
func (a *Agent) EnterPlanMode(planID, planFilePath, reason string) {
	if a.planMode != nil {
		a.planMode.EnterPlanMode(planID, planFilePath, reason)

		// 发送监控事件
		a.eventBus.EmitMonitor(&types.MonitorStateChangedEvent{
			State: types.AgentStateWorking,
		})

		agentLog.Info(context.Background(), "entered plan mode", map[string]any{
			"agent_id":       a.id,
			"plan_id":        planID,
			"plan_file_path": planFilePath,
			"reason":         reason,
		})
	}
}

// ExitPlanMode 让 Agent 退出 Plan 模式
func (a *Agent) ExitPlanMode() {
	if a.planMode != nil {
		a.planMode.ExitPlanMode()

		agentLog.Info(context.Background(), "exited plan mode", map[string]any{
			"agent_id": a.id,
		})
	}
}

// IsInPlanMode 检查 Agent 是否处于 Plan 模式
func (a *Agent) IsInPlanMode() bool {
	if a.planMode == nil {
		return false
	}
	return a.planMode.IsActive()
}

// GetPlanModeState 获取 Plan 模式状态
func (a *Agent) GetPlanModeState() *PlanModeState {
	if a.planMode == nil {
		return &PlanModeState{Active: false}
	}
	return a.planMode.GetState()
}
