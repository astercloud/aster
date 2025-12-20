package tools

import (
	"context"
	"errors"
	"fmt"
)

// ToolConstraints 工具约束接口
// 用于控制在特定上下文中哪些工具可用
// 这是 Manus 团队"工具状态机"理念的实现
type ToolConstraints interface {
	// IsAllowed 检查工具是否被允许使用
	IsAllowed(ctx context.Context, toolName string) bool

	// GetAllowedTools 获取所有允许的工具名称
	GetAllowedTools(ctx context.Context) []string

	// GetConstraintType 获取约束类型
	GetConstraintType() ConstraintType
}

// ConstraintType 约束类型
type ConstraintType string

const (
	// ConstraintTypeNone 无约束（所有工具可用）
	ConstraintTypeNone ConstraintType = "none"

	// ConstraintTypeWhitelist 白名单（只有列表中的工具可用）
	ConstraintTypeWhitelist ConstraintType = "whitelist"

	// ConstraintTypeBlacklist 黑名单（列表中的工具不可用）
	ConstraintTypeBlacklist ConstraintType = "blacklist"

	// ConstraintTypeRequired 必需工具（必须使用指定工具）
	ConstraintTypeRequired ConstraintType = "required"

	// ConstraintTypeAuto 自动选择（由系统决定）
	ConstraintTypeAuto ConstraintType = "auto"
)

// NoConstraints 无约束实现
type NoConstraints struct{}

func (n *NoConstraints) IsAllowed(ctx context.Context, toolName string) bool {
	return true
}

func (n *NoConstraints) GetAllowedTools(ctx context.Context) []string {
	return nil // nil 表示所有工具
}

func (n *NoConstraints) GetConstraintType() ConstraintType {
	return ConstraintTypeNone
}

// WhitelistConstraints 白名单约束
type WhitelistConstraints struct {
	allowedTools map[string]bool
}

// NewWhitelistConstraints 创建白名单约束
func NewWhitelistConstraints(tools []string) *WhitelistConstraints {
	allowed := make(map[string]bool, len(tools))
	for _, tool := range tools {
		allowed[tool] = true
	}
	return &WhitelistConstraints{
		allowedTools: allowed,
	}
}

func (w *WhitelistConstraints) IsAllowed(ctx context.Context, toolName string) bool {
	return w.allowedTools[toolName]
}

func (w *WhitelistConstraints) GetAllowedTools(ctx context.Context) []string {
	tools := make([]string, 0, len(w.allowedTools))
	for tool := range w.allowedTools {
		tools = append(tools, tool)
	}
	return tools
}

func (w *WhitelistConstraints) GetConstraintType() ConstraintType {
	return ConstraintTypeWhitelist
}

// BlacklistConstraints 黑名单约束
type BlacklistConstraints struct {
	blockedTools map[string]bool
}

// NewBlacklistConstraints 创建黑名单约束
func NewBlacklistConstraints(tools []string) *BlacklistConstraints {
	blocked := make(map[string]bool, len(tools))
	for _, tool := range tools {
		blocked[tool] = true
	}
	return &BlacklistConstraints{
		blockedTools: blocked,
	}
}

func (b *BlacklistConstraints) IsAllowed(ctx context.Context, toolName string) bool {
	return !b.blockedTools[toolName]
}

func (b *BlacklistConstraints) GetAllowedTools(ctx context.Context) []string {
	return nil // 黑名单不返回允许列表
}

func (b *BlacklistConstraints) GetConstraintType() ConstraintType {
	return ConstraintTypeBlacklist
}

// RequiredToolConstraints 必需工具约束
type RequiredToolConstraints struct {
	requiredTool string
}

// NewRequiredToolConstraints 创建必需工具约束
func NewRequiredToolConstraints(toolName string) *RequiredToolConstraints {
	return &RequiredToolConstraints{
		requiredTool: toolName,
	}
}

func (r *RequiredToolConstraints) IsAllowed(ctx context.Context, toolName string) bool {
	return toolName == r.requiredTool
}

func (r *RequiredToolConstraints) GetAllowedTools(ctx context.Context) []string {
	return []string{r.requiredTool}
}

func (r *RequiredToolConstraints) GetConstraintType() ConstraintType {
	return ConstraintTypeRequired
}

func (r *RequiredToolConstraints) GetRequiredTool() string {
	return r.requiredTool
}

// ToolChoice 工具选择策略
// 对应 Anthropic API 的 tool_choice 参数
type ToolChoice struct {
	// Type 选择类型: "auto", "any", "tool"
	Type string `json:"type"`

	// Name 当 Type="tool" 时，指定工具名称
	Name string `json:"name,omitempty"`

	// DisableParallelToolUse 禁用并行工具调用
	DisableParallelToolUse bool `json:"disable_parallel_tool_use,omitempty"`
}

// ToolChoiceAuto 自动选择工具
var ToolChoiceAuto = &ToolChoice{Type: "auto"}

// ToolChoiceAny 必须使用工具（任意一个）
var ToolChoiceAny = &ToolChoice{Type: "any"}

// ToolChoiceRequired 必须使用指定工具
func ToolChoiceRequired(toolName string) *ToolChoice {
	return &ToolChoice{
		Type: "tool",
		Name: toolName,
	}
}

// ToConstraints 将 ToolChoice 转换为 ToolConstraints
func (tc *ToolChoice) ToConstraints() ToolConstraints {
	switch tc.Type {
	case "auto":
		return &NoConstraints{}
	case "any":
		// "any" 表示必须使用工具，但不限制具体哪个
		return &NoConstraints{}
	case "tool":
		if tc.Name != "" {
			return NewRequiredToolConstraints(tc.Name)
		}
		return &NoConstraints{}
	default:
		return &NoConstraints{}
	}
}

// ToolSelector 工具选择器接口
// 用于根据上下文动态选择可用工具
type ToolSelector interface {
	// SelectTools 根据上下文选择工具
	SelectTools(ctx context.Context, allTools []Tool, constraints ToolConstraints) ([]Tool, error)

	// ShouldUseToolChoice 判断是否应该使用 tool_choice
	ShouldUseToolChoice(ctx context.Context, constraints ToolConstraints) (*ToolChoice, bool)
}

// DefaultToolSelector 默认工具选择器
type DefaultToolSelector struct{}

// SelectTools 选择工具
func (s *DefaultToolSelector) SelectTools(ctx context.Context, allTools []Tool, constraints ToolConstraints) ([]Tool, error) {
	if constraints == nil || constraints.GetConstraintType() == ConstraintTypeNone {
		return allTools, nil
	}

	// 根据约束类型过滤
	switch constraints.GetConstraintType() {
	case ConstraintTypeWhitelist, ConstraintTypeRequired:
		allowedNames := constraints.GetAllowedTools(ctx)
		if allowedNames == nil {
			return allTools, nil
		}

		allowed := make(map[string]bool, len(allowedNames))
		for _, name := range allowedNames {
			allowed[name] = true
		}

		filtered := make([]Tool, 0, len(allTools))
		for _, tool := range allTools {
			if allowed[tool.Name()] {
				filtered = append(filtered, tool)
			}
		}
		return filtered, nil

	case ConstraintTypeBlacklist:
		filtered := make([]Tool, 0, len(allTools))
		for _, tool := range allTools {
			if constraints.IsAllowed(ctx, tool.Name()) {
				filtered = append(filtered, tool)
			}
		}
		return filtered, nil

	default:
		return allTools, nil
	}
}

// ShouldUseToolChoice 判断是否应该使用 tool_choice
func (s *DefaultToolSelector) ShouldUseToolChoice(ctx context.Context, constraints ToolConstraints) (*ToolChoice, bool) {
	if constraints == nil {
		return nil, false
	}

	switch constraints.GetConstraintType() {
	case ConstraintTypeRequired:
		// 必需工具约束 -> 使用 tool_choice
		if req, ok := constraints.(*RequiredToolConstraints); ok {
			return ToolChoiceRequired(req.GetRequiredTool()), true
		}
		return nil, false

	case ConstraintTypeAuto:
		// 自动选择 -> 使用 auto
		return ToolChoiceAuto, true

	default:
		return nil, false
	}
}

// ConstraintsBuilder 约束构建器
type ConstraintsBuilder struct {
	constraintType ConstraintType
	tools          []string
}

// NewConstraintsBuilder 创建约束构建器
func NewConstraintsBuilder() *ConstraintsBuilder {
	return &ConstraintsBuilder{}
}

// WithWhitelist 设置白名单
func (b *ConstraintsBuilder) WithWhitelist(tools ...string) *ConstraintsBuilder {
	b.constraintType = ConstraintTypeWhitelist
	b.tools = tools
	return b
}

// WithBlacklist 设置黑名单
func (b *ConstraintsBuilder) WithBlacklist(tools ...string) *ConstraintsBuilder {
	b.constraintType = ConstraintTypeBlacklist
	b.tools = tools
	return b
}

// WithRequired 设置必需工具
func (b *ConstraintsBuilder) WithRequired(toolName string) *ConstraintsBuilder {
	b.constraintType = ConstraintTypeRequired
	b.tools = []string{toolName}
	return b
}

// Build 构建约束
func (b *ConstraintsBuilder) Build() (ToolConstraints, error) {
	switch b.constraintType {
	case ConstraintTypeWhitelist:
		if len(b.tools) == 0 {
			return nil, errors.New("whitelist requires at least one tool")
		}
		return NewWhitelistConstraints(b.tools), nil

	case ConstraintTypeBlacklist:
		if len(b.tools) == 0 {
			return nil, errors.New("blacklist requires at least one tool")
		}
		return NewBlacklistConstraints(b.tools), nil

	case ConstraintTypeRequired:
		if len(b.tools) != 1 {
			return nil, errors.New("required constraint needs exactly one tool")
		}
		return NewRequiredToolConstraints(b.tools[0]), nil

	case ConstraintTypeNone:
		return &NoConstraints{}, nil

	default:
		return nil, fmt.Errorf("unknown constraint type: %s", b.constraintType)
	}
}
