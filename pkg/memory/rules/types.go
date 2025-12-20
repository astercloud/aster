// Package rules 提供规则管理系统
// 支持 Global（全局）和 Project（项目）两个级别的规则
package rules

import (
	"strings"
	"time"
)

// Scope 规则作用域
type Scope string

const (
	// ScopeGlobal 全局规则（用户级，跨项目）
	ScopeGlobal Scope = "global"
	// ScopeProject 项目规则（项目级）
	ScopeProject Scope = "project"
)

// Rule 规则
type Rule struct {
	// ID 规则 ID
	ID string `json:"id"`

	// Scope 作用域
	Scope Scope `json:"scope"`

	// Title 标题
	Title string `json:"title"`

	// Content 规则内容
	Content string `json:"content"`

	// Source 来源（AGENTS.md, user_config, system）
	Source string `json:"source"`

	// SourcePath 来源路径（如文件路径）
	SourcePath string `json:"source_path,omitempty"`

	// Priority 优先级（数字越大优先级越高）
	Priority int `json:"priority"`

	// Enabled 是否启用
	Enabled bool `json:"enabled"`

	// Tags 标签
	Tags []string `json:"tags,omitempty"`

	// Metadata 元数据
	Metadata map[string]any `json:"metadata,omitempty"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at"`
}

// RuleSet 规则集
type RuleSet struct {
	// GlobalRules 全局规则
	GlobalRules []*Rule `json:"global_rules"`

	// ProjectRules 项目规则（key: projectID）
	ProjectRules map[string][]*Rule `json:"project_rules"`
}

// NewRule 创建新规则
func NewRule(scope Scope, title, content string) *Rule {
	now := time.Now()
	return &Rule{
		ID:        generateID(),
		Scope:     scope,
		Title:     title,
		Content:   content,
		Priority:  0,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewRuleSet 创建规则集
func NewRuleSet() *RuleSet {
	return &RuleSet{
		GlobalRules:  []*Rule{},
		ProjectRules: make(map[string][]*Rule),
	}
}

// AddGlobalRule 添加全局规则
func (rs *RuleSet) AddGlobalRule(rule *Rule) {
	rule.Scope = ScopeGlobal
	rs.GlobalRules = append(rs.GlobalRules, rule)
}

// AddProjectRule 添加项目规则
func (rs *RuleSet) AddProjectRule(projectID string, rule *Rule) {
	rule.Scope = ScopeProject
	if rs.ProjectRules[projectID] == nil {
		rs.ProjectRules[projectID] = []*Rule{}
	}
	rs.ProjectRules[projectID] = append(rs.ProjectRules[projectID], rule)
}

// GetRulesForProject 获取项目的所有规则（包含全局规则）
func (rs *RuleSet) GetRulesForProject(projectID string) []*Rule {
	var rules []*Rule

	// 先添加全局规则
	for _, r := range rs.GlobalRules {
		if r.Enabled {
			rules = append(rules, r)
		}
	}

	// 再添加项目规则
	if projectRules, ok := rs.ProjectRules[projectID]; ok {
		for _, r := range projectRules {
			if r.Enabled {
				rules = append(rules, r)
			}
		}
	}

	// 按优先级排序（优先级高的在前）
	sortRulesByPriority(rules)

	return rules
}

// GetRulesContent 获取规则内容（用于注入 AI 上下文）
func (rs *RuleSet) GetRulesContent(projectID string) string {
	rules := rs.GetRulesForProject(projectID)
	if len(rules) == 0 {
		return ""
	}

	var content string
	var contentSb139 strings.Builder
	for _, r := range rules {
		contentSb139.WriteString(r.Content + "\n\n")
	}
	content += contentSb139.String()
	return content
}

// sortRulesByPriority 按优先级排序
func sortRulesByPriority(rules []*Rule) {
	for i := range len(rules) - 1 {
		for j := i + 1; j < len(rules); j++ {
			if rules[j].Priority > rules[i].Priority {
				rules[i], rules[j] = rules[j], rules[i]
			}
		}
	}
}

// generateID 生成 ID
func generateID() string {
	return time.Now().Format("20060102150405.000000")
}
