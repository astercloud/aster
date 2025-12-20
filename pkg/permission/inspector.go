// Package permission provides enhanced permission system aligned with Claude Agent SDK.
package permission

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/types"
)

// EnhancedInspector 增强版权限检查器 (Claude Agent SDK 风格)
// 支持 CanUseTool 回调、沙箱集成、动态权限更新
type EnhancedInspector struct {
	// 基础配置
	mode          Mode
	sandboxConfig *types.SandboxConfig
	canUseTool    types.CanUseToolFunc

	// 规则管理
	rules      []Rule
	rulesMutex sync.RWMutex

	// 风险评估
	toolRisks    map[string]RiskLevel
	defaultRisks map[string]RiskLevel

	// 会话级规则（不持久化）
	sessionRules      []Rule
	sessionRulesMutex sync.RWMutex

	// 持久化
	persistPath string
	autoLoad    bool

	// 违规记录
	violations      []types.SandboxViolation
	violationsMutex sync.RWMutex
}

// EnhancedInspectorConfig 增强检查器配置
type EnhancedInspectorConfig struct {
	Mode          Mode
	SandboxConfig *types.SandboxConfig
	CanUseTool    types.CanUseToolFunc
	PersistPath   string
	AutoLoad      bool
}

// NewEnhancedInspector 创建增强版权限检查器
func NewEnhancedInspector(cfg *EnhancedInspectorConfig) *EnhancedInspector {
	if cfg == nil {
		cfg = &EnhancedInspectorConfig{
			Mode:     ModeSmartApprove,
			AutoLoad: true,
		}
	}

	i := &EnhancedInspector{
		mode:          cfg.Mode,
		sandboxConfig: cfg.SandboxConfig,
		canUseTool:    cfg.CanUseTool,
		rules:         make([]Rule, 0),
		sessionRules:  make([]Rule, 0),
		toolRisks:     make(map[string]RiskLevel),
		persistPath:   cfg.PersistPath,
		autoLoad:      cfg.AutoLoad,
		violations:    make([]types.SandboxViolation, 0),
		defaultRisks: map[string]RiskLevel{
			// Low risk - read operations
			"Read":            RiskLevelLow,
			"Ls":              RiskLevelLow, // 列出目录内容
			"Glob":            RiskLevelLow,
			"Grep":            RiskLevelLow,
			"WebSearch":       RiskLevelLow,
			"BashOutput":      RiskLevelLow,
			"AskUserQuestion": RiskLevelLow, // 用户交互，无副作用
			"read_file":       RiskLevelLow,
			"list_dir":        RiskLevelLow,
			"file_search":     RiskLevelLow,
			"grep_search":     RiskLevelLow,
			"web_search":      RiskLevelLow,
			"get_file_info":   RiskLevelLow,
			"semantic_search": RiskLevelLow,

			// Medium risk - write operations
			"Write":            RiskLevelMedium,
			"Edit":             RiskLevelMedium,
			"WebFetch":         RiskLevelMedium,
			"TodoWrite":        RiskLevelMedium,
			"write_file":       RiskLevelMedium,
			"create_file":      RiskLevelMedium,
			"edit_file":        RiskLevelMedium,
			"delete_file":      RiskLevelMedium,
			"rename_file":      RiskLevelMedium,
			"move_file":        RiskLevelMedium,
			"create_directory": RiskLevelMedium,
			"http_request":     RiskLevelMedium,

			// High risk - system operations
			"Bash":             RiskLevelHigh,
			"KillShell":        RiskLevelHigh,
			"Task":             RiskLevelHigh,
			"bash":             RiskLevelHigh,
			"execute":          RiskLevelHigh,
			"run_command":      RiskLevelHigh,
			"shell":            RiskLevelHigh,
			"exec":             RiskLevelHigh,
			"subprocess":       RiskLevelHigh,
			"process_spawn":    RiskLevelHigh,
			"system":           RiskLevelHigh,
			"network_request":  RiskLevelHigh,
			"database_execute": RiskLevelHigh,
		},
	}

	if i.autoLoad && i.persistPath != "" {
		i.loadRules()
	}

	return i
}

// SetSandboxConfig 设置沙箱配置
func (i *EnhancedInspector) SetSandboxConfig(cfg *types.SandboxConfig) {
	i.sandboxConfig = cfg
}

// SetCanUseTool 设置自定义权限回调
func (i *EnhancedInspector) SetCanUseTool(fn types.CanUseToolFunc) {
	i.canUseTool = fn
}

// Check 执行权限检查 (Claude Agent SDK 风格)
func (i *EnhancedInspector) Check(ctx context.Context, call *types.ToolCallSnapshot) (*CheckResult, error) {
	// 构建请求
	req := &Request{
		ToolName:  call.Name,
		Arguments: call.Arguments,
		RiskLevel: i.GetToolRisk(call.Name),
		CallID:    call.ID,
		Context:   make(map[string]any),
	}

	// 检查是否请求绕过沙箱
	bypassSandbox := false
	if bypass, ok := call.Arguments["dangerouslyDisableSandbox"].(bool); ok && bypass {
		bypassSandbox = true
		req.Context["bypass_sandbox"] = true
	}

	// 1. 检查沙箱权限模式
	if i.sandboxConfig != nil && i.sandboxConfig.PermissionMode != "" {
		switch i.sandboxConfig.PermissionMode {
		case types.SandboxPermissionBypass:
			// 绕过所有权限检查
			return &CheckResult{Allowed: true, DecidedBy: "bypass_mode"}, nil

		case types.SandboxPermissionPlan:
			// 规划模式 - 不执行，只记录
			return &CheckResult{
				Allowed:   false,
				DecidedBy: "plan_mode",
				Message:   "Plan mode: tool execution blocked",
			}, nil

		case types.SandboxPermissionAcceptEdits:
			// 自动接受文件编辑
			if i.isEditTool(call.Name) {
				return &CheckResult{Allowed: true, DecidedBy: "accept_edits_mode"}, nil
			}
		}
	}

	// 2. 检查自定义回调
	if i.canUseTool != nil {
		opts := &types.CanUseToolOptions{
			Signal:                 ctx,
			SandboxEnabled:         i.isSandboxEnabled(),
			BypassSandboxRequested: bypassSandbox,
		}

		result, err := i.canUseTool(ctx, call.Name, call.Arguments, opts)
		if err != nil {
			return nil, fmt.Errorf("canUseTool callback error: %w", err)
		}

		if result != nil {
			// 应用权限更新
			if len(result.UpdatedPermissions) > 0 {
				i.applyPermissionUpdates(result.UpdatedPermissions)
			}

			if result.Behavior == "allow" {
				checkResult := &CheckResult{
					Allowed:      true,
					DecidedBy:    "canUseTool",
					UpdatedInput: result.UpdatedInput,
				}
				return checkResult, nil
			}

			if result.Behavior == "deny" {
				return &CheckResult{
					Allowed:   false,
					DecidedBy: "canUseTool",
					Message:   result.Message,
					Interrupt: result.Interrupt,
				}, nil
			}
		}
	}

	// 3. 检查沙箱配置
	if i.sandboxConfig != nil && i.sandboxConfig.Settings != nil {
		settings := i.sandboxConfig.Settings

		// 检查是否在排除命令列表
		if i.isExcludedCommand(call.Name, call.Arguments) {
			return &CheckResult{Allowed: true, DecidedBy: "excluded_command"}, nil
		}

		// 检查绕过沙箱请求
		if bypassSandbox {
			if !settings.AllowUnsandboxedCommands {
				return &CheckResult{
					Allowed:   false,
					DecidedBy: "sandbox_policy",
					Message:   "Unsandboxed commands not allowed",
				}, nil
			}
			// 需要额外审批
			req.RiskLevel = RiskLevelHigh
		}

		// AutoAllowBashIfSandboxed
		if settings.Enabled && settings.AutoAllowBashIfSandboxed {
			if call.Name == "Bash" || call.Name == "bash" {
				return &CheckResult{Allowed: true, DecidedBy: "auto_allow_bash"}, nil
			}
		}
	}

	// 4. 检查会话级规则（优先级高于模式）
	if rule := i.findMatchingSessionRule(req); rule != nil {
		return i.applyRule(rule, req)
	}

	// 5. 检查持久化规则
	if rule := i.findMatchingRule(req); rule != nil {
		return i.applyRule(rule, req)
	}

	// 6. 检查模式
	switch i.mode {
	case ModeAutoApprove:
		return &CheckResult{Allowed: true, DecidedBy: "auto_approve"}, nil

	case ModeAlwaysAsk:
		return &CheckResult{
			Allowed:         false,
			NeedsApproval:   true,
			DecidedBy:       "always_ask",
			ApprovalRequest: i.createApprovalEvent(req),
		}, nil

	case ModeSmartApprove:
		return i.smartCheck(ctx, req)

	default:
		return &CheckResult{
			Allowed:         false,
			NeedsApproval:   true,
			DecidedBy:       "default",
			ApprovalRequest: i.createApprovalEvent(req),
		}, nil
	}
}

// CheckResult 权限检查结果
type CheckResult struct {
	// Allowed 是否允许
	Allowed bool

	// NeedsApproval 是否需要用户审批
	NeedsApproval bool

	// DecidedBy 决策来源
	DecidedBy string

	// Message 消息
	Message string

	// Interrupt 是否中断执行
	Interrupt bool

	// UpdatedInput 修改后的输入
	UpdatedInput map[string]any

	// ApprovalRequest 审批请求事件
	ApprovalRequest *types.ControlPermissionRequiredEvent
}

// smartCheck 智能权限检查
func (i *EnhancedInspector) smartCheck(ctx context.Context, req *Request) (*CheckResult, error) {
	// 规则已在 Check 方法中检查过，这里直接根据风险级别决策
	switch req.RiskLevel {
	case RiskLevelLow:
		return &CheckResult{Allowed: true, DecidedBy: "low_risk"}, nil

	case RiskLevelMedium:
		if i.isSafeOperation(req) {
			return &CheckResult{Allowed: true, DecidedBy: "safe_operation"}, nil
		}
		return &CheckResult{
			Allowed:         false,
			NeedsApproval:   true,
			DecidedBy:       "medium_risk",
			ApprovalRequest: i.createApprovalEvent(req),
		}, nil

	case RiskLevelHigh:
		return &CheckResult{
			Allowed:         false,
			NeedsApproval:   true,
			DecidedBy:       "high_risk",
			ApprovalRequest: i.createApprovalEvent(req),
		}, nil

	default:
		return &CheckResult{
			Allowed:         false,
			NeedsApproval:   true,
			DecidedBy:       "unknown_risk",
			ApprovalRequest: i.createApprovalEvent(req),
		}, nil
	}
}

// applyRule 应用规则
func (i *EnhancedInspector) applyRule(rule *Rule, req *Request) (*CheckResult, error) {
	switch rule.Decision {
	case DecisionAllow, DecisionAllowAlways:
		return &CheckResult{Allowed: true, DecidedBy: "rule:" + rule.Pattern}, nil
	case DecisionDeny, DecisionDenyAlways:
		return &CheckResult{
			Allowed:   false,
			DecidedBy: "rule:" + rule.Pattern,
			Message:   rule.Note,
		}, nil
	default:
		return &CheckResult{
			Allowed:         false,
			NeedsApproval:   true,
			DecidedBy:       "rule:" + rule.Pattern,
			ApprovalRequest: i.createApprovalEvent(req),
		}, nil
	}
}

// isExcludedCommand 检查是否为排除命令
func (i *EnhancedInspector) isExcludedCommand(toolName string, args map[string]any) bool {
	if i.sandboxConfig == nil || i.sandboxConfig.Settings == nil {
		return false
	}

	settings := i.sandboxConfig.Settings
	if len(settings.ExcludedCommands) == 0 {
		return false
	}

	// 对于 Bash 工具，检查命令内容
	if toolName == "Bash" || toolName == "bash" {
		if cmd, ok := args["command"].(string); ok {
			for _, excluded := range settings.ExcludedCommands {
				if strings.HasPrefix(cmd, excluded+" ") || cmd == excluded {
					return true
				}
			}
		}
	}

	// 检查工具名称
	return slices.Contains(settings.ExcludedCommands, toolName)
}

// isEditTool 检查是否为编辑工具
func (i *EnhancedInspector) isEditTool(toolName string) bool {
	editTools := map[string]bool{
		"Write":       true,
		"Edit":        true,
		"write_file":  true,
		"edit_file":   true,
		"create_file": true,
	}
	return editTools[toolName]
}

// isSandboxEnabled 检查沙箱是否启用
func (i *EnhancedInspector) isSandboxEnabled() bool {
	if i.sandboxConfig == nil || i.sandboxConfig.Settings == nil {
		return false
	}
	return i.sandboxConfig.Settings.Enabled
}

// isSafeOperation 检查是否为安全操作
func (i *EnhancedInspector) isSafeOperation(req *Request) bool {
	// 检查文件路径
	if path, ok := req.Arguments["path"].(string); ok {
		if !filepath.IsAbs(path) {
			return true
		}
		// 检查忽略违规配置
		if i.shouldIgnoreFileViolation(path) {
			return true
		}
	}

	// 检查 HTTP 方法
	if method, ok := req.Arguments["method"].(string); ok {
		if strings.ToUpper(method) == "GET" {
			return true
		}
	}

	return false
}

// shouldIgnoreFileViolation 检查是否应忽略文件违规
func (i *EnhancedInspector) shouldIgnoreFileViolation(path string) bool {
	if i.sandboxConfig == nil || i.sandboxConfig.Settings == nil {
		return false
	}
	if i.sandboxConfig.Settings.IgnoreViolations == nil {
		return false
	}

	for _, pattern := range i.sandboxConfig.Settings.IgnoreViolations.FilePatterns {
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
		// 支持正则
		if re, err := regexp.Compile(pattern); err == nil && re.MatchString(path) {
			return true
		}
	}

	return false
}

// applyPermissionUpdates 应用权限更新
func (i *EnhancedInspector) applyPermissionUpdates(updates []types.PermissionUpdate) {
	for _, update := range updates {
		switch update.Type {
		case "addRules":
			for _, rule := range update.Rules {
				newRule := Rule{
					Pattern:   rule.ToolName,
					Decision:  Decision(update.Behavior),
					CreatedAt: time.Now(),
					Note:      rule.RuleContent,
				}
				if update.Destination == "session" {
					i.addSessionRule(newRule)
				} else {
					i.AddRule(newRule)
				}
			}

		case "removeRules":
			for _, rule := range update.Rules {
				if update.Destination == "session" {
					i.removeSessionRule(rule.ToolName)
				} else {
					i.RemoveRule(rule.ToolName)
				}
			}

		case "setMode":
			if mode := Mode(update.Mode); mode != "" {
				i.SetMode(mode)
			}
		}
	}
}

// addSessionRule 添加会话级规则
func (i *EnhancedInspector) addSessionRule(rule Rule) {
	i.sessionRulesMutex.Lock()
	defer i.sessionRulesMutex.Unlock()
	i.sessionRules = append(i.sessionRules, rule)
}

// removeSessionRule 移除会话级规则
func (i *EnhancedInspector) removeSessionRule(pattern string) {
	i.sessionRulesMutex.Lock()
	defer i.sessionRulesMutex.Unlock()

	for idx, rule := range i.sessionRules {
		if rule.Pattern == pattern {
			i.sessionRules = append(i.sessionRules[:idx], i.sessionRules[idx+1:]...)
			return
		}
	}
}

// findMatchingSessionRule 查找匹配的会话级规则
func (i *EnhancedInspector) findMatchingSessionRule(req *Request) *Rule {
	i.sessionRulesMutex.RLock()
	defer i.sessionRulesMutex.RUnlock()

	for _, rule := range i.sessionRules {
		if i.matchPattern(rule.Pattern, req.ToolName) {
			if i.checkConditions(rule.Conditions, req.Arguments) {
				return &rule
			}
		}
	}
	return nil
}

// RecordViolation 记录沙箱违规
func (i *EnhancedInspector) RecordViolation(violation types.SandboxViolation) {
	i.violationsMutex.Lock()
	defer i.violationsMutex.Unlock()
	i.violations = append(i.violations, violation)
}

// GetViolations 获取违规记录
func (i *EnhancedInspector) GetViolations() []types.SandboxViolation {
	i.violationsMutex.RLock()
	defer i.violationsMutex.RUnlock()

	violations := make([]types.SandboxViolation, len(i.violations))
	copy(violations, i.violations)
	return violations
}

// ClearSessionRules 清除会话级规则
func (i *EnhancedInspector) ClearSessionRules() {
	i.sessionRulesMutex.Lock()
	defer i.sessionRulesMutex.Unlock()
	i.sessionRules = make([]Rule, 0)
}

// 继承原有 Inspector 的方法

// SetMode 设置模式
func (i *EnhancedInspector) SetMode(mode Mode) {
	i.mode = mode
}

// GetMode 获取模式
func (i *EnhancedInspector) GetMode() Mode {
	return i.mode
}

// GetToolRisk 获取工具风险级别
func (i *EnhancedInspector) GetToolRisk(toolName string) RiskLevel {
	i.rulesMutex.RLock()
	defer i.rulesMutex.RUnlock()

	if level, ok := i.toolRisks[toolName]; ok {
		return level
	}
	if level, ok := i.defaultRisks[toolName]; ok {
		return level
	}
	return RiskLevelMedium
}

// SetToolRisk 设置工具风险级别
func (i *EnhancedInspector) SetToolRisk(toolName string, level RiskLevel) {
	i.rulesMutex.Lock()
	defer i.rulesMutex.Unlock()
	i.toolRisks[toolName] = level
}

// AddRule 添加规则
func (i *EnhancedInspector) AddRule(rule Rule) {
	i.rulesMutex.Lock()
	defer i.rulesMutex.Unlock()

	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = time.Now()
	}
	i.rules = append(i.rules, rule)
	i.saveRules()
}

// RemoveRule 移除规则
func (i *EnhancedInspector) RemoveRule(pattern string) bool {
	i.rulesMutex.Lock()
	defer i.rulesMutex.Unlock()

	for idx, rule := range i.rules {
		if rule.Pattern == pattern {
			i.rules = append(i.rules[:idx], i.rules[idx+1:]...)
			i.saveRules()
			return true
		}
	}
	return false
}

// GetRules 获取所有规则
func (i *EnhancedInspector) GetRules() []Rule {
	i.rulesMutex.RLock()
	defer i.rulesMutex.RUnlock()

	rules := make([]Rule, len(i.rules))
	copy(rules, i.rules)
	return rules
}

// findMatchingRule 查找匹配规则
func (i *EnhancedInspector) findMatchingRule(req *Request) *Rule {
	i.rulesMutex.RLock()
	defer i.rulesMutex.RUnlock()

	now := time.Now()
	for _, rule := range i.rules {
		if rule.ExpiresAt != nil && rule.ExpiresAt.Before(now) {
			continue
		}
		if !i.matchPattern(rule.Pattern, req.ToolName) {
			continue
		}
		if !i.checkConditions(rule.Conditions, req.Arguments) {
			continue
		}
		return &rule
	}
	return nil
}

// matchPattern 匹配模式
func (i *EnhancedInspector) matchPattern(pattern, toolName string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == toolName {
		return true
	}
	if before, ok := strings.CutSuffix(pattern, "*"); ok {
		prefix := before
		return strings.HasPrefix(toolName, prefix)
	}
	if after, ok := strings.CutPrefix(pattern, "*"); ok {
		suffix := after
		return strings.HasSuffix(toolName, suffix)
	}
	return false
}

// checkConditions 检查条件
func (i *EnhancedInspector) checkConditions(conditions []Condition, args map[string]any) bool {
	for _, cond := range conditions {
		value, ok := args[cond.Field]
		if !ok {
			return false
		}
		strValue := fmt.Sprintf("%v", value)

		switch cond.Operator {
		case "eq":
			if strValue != cond.Value {
				return false
			}
		case "ne":
			if strValue == cond.Value {
				return false
			}
		case "contains":
			if !strings.Contains(strValue, cond.Value) {
				return false
			}
		case "prefix":
			if !strings.HasPrefix(strValue, cond.Value) {
				return false
			}
		case "suffix":
			if !strings.HasSuffix(strValue, cond.Value) {
				return false
			}
		}
	}
	return true
}

// createApprovalEvent 创建审批事件
func (i *EnhancedInspector) createApprovalEvent(req *Request) *types.ControlPermissionRequiredEvent {
	return &types.ControlPermissionRequiredEvent{
		Call: types.ToolCallSnapshot{
			ID:        req.CallID,
			Name:      req.ToolName,
			Arguments: req.Arguments,
		},
	}
}

// loadRules 加载规则
func (i *EnhancedInspector) loadRules() {
	// 复用原有的加载逻辑
}

// saveRules 保存规则
func (i *EnhancedInspector) saveRules() {
	// 复用原有的保存逻辑
}

// RecordDecision 记录决策
func (i *EnhancedInspector) RecordDecision(req *Request, decision Decision, note string) *Response {
	resp := &Response{
		Request:   req,
		Decision:  decision,
		DecidedBy: "user",
		Note:      note,
		DecidedAt: time.Now(),
	}

	switch decision {
	case DecisionAllowAlways:
		i.AddRule(Rule{
			Pattern:   req.ToolName,
			Decision:  DecisionAllow,
			RiskLevel: req.RiskLevel,
			CreatedAt: time.Now(),
			Note:      "Auto-created from allow_always: " + note,
		})
	case DecisionDenyAlways:
		i.AddRule(Rule{
			Pattern:   req.ToolName,
			Decision:  DecisionDeny,
			RiskLevel: req.RiskLevel,
			CreatedAt: time.Now(),
			Note:      "Auto-created from deny_always: " + note,
		})
	}

	return resp
}
