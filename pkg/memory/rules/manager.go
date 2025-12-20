package rules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Manager 规则管理器
type Manager struct {
	mu sync.RWMutex

	// ruleSet 规则集
	ruleSet *RuleSet

	// config 配置
	config *ManagerConfig

	// loaders 规则加载器
	loaders []Loader
}

// ManagerConfig 管理器配置
type ManagerConfig struct {
	// GlobalRulesPath 全局规则文件路径
	GlobalRulesPath string

	// ProjectBasePath 项目基础路径
	ProjectBasePath string

	// ProjectRuleFile 项目规则文件名（默认 AGENTS.md）
	ProjectRuleFile string

	// AutoReload 是否自动重新加载
	AutoReload bool
}

// DefaultManagerConfig 默认配置
func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		GlobalRulesPath: ".aster/rules.md",
		ProjectBasePath: "workspaces",
		ProjectRuleFile: "AGENTS.md",
		AutoReload:      false,
	}
}

// NewManager 创建规则管理器
func NewManager(config *ManagerConfig) *Manager {
	if config == nil {
		config = DefaultManagerConfig()
	}
	return &Manager{
		ruleSet: NewRuleSet(),
		config:  config,
		loaders: []Loader{},
	}
}

// RegisterLoader 注册规则加载器
func (m *Manager) RegisterLoader(loader Loader) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loaders = append(m.loaders, loader)
}

// LoadGlobalRules 加载全局规则
func (m *Manager) LoadGlobalRules(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 从文件加载
	if m.config.GlobalRulesPath != "" {
		rules, err := m.loadRulesFromFile(m.config.GlobalRulesPath, ScopeGlobal)
		if err != nil {
			// 文件不存在不是错误
			if !os.IsNotExist(err) {
				return fmt.Errorf("load global rules failed: %w", err)
			}
		} else {
			for _, r := range rules {
				m.ruleSet.AddGlobalRule(r)
			}
		}
	}

	// 使用自定义加载器
	for _, loader := range m.loaders {
		rules, err := loader.LoadGlobalRules(ctx)
		if err != nil {
			return fmt.Errorf("loader %T failed: %w", loader, err)
		}
		for _, r := range rules {
			m.ruleSet.AddGlobalRule(r)
		}
	}

	return nil
}

// LoadProjectRules 加载项目规则
func (m *Manager) LoadProjectRules(ctx context.Context, projectID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 从项目文件加载
	if m.config.ProjectBasePath != "" && m.config.ProjectRuleFile != "" {
		filePath := filepath.Join(m.config.ProjectBasePath, projectID, m.config.ProjectRuleFile)
		rules, err := m.loadRulesFromFile(filePath, ScopeProject)
		if err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("load project rules failed: %w", err)
			}
		} else {
			for _, r := range rules {
				m.ruleSet.AddProjectRule(projectID, r)
			}
		}
	}

	// 使用自定义加载器
	for _, loader := range m.loaders {
		rules, err := loader.LoadProjectRules(ctx, projectID)
		if err != nil {
			return fmt.Errorf("loader %T failed: %w", loader, err)
		}
		for _, r := range rules {
			m.ruleSet.AddProjectRule(projectID, r)
		}
	}

	return nil
}

// GetRulesForProject 获取项目的所有规则
func (m *Manager) GetRulesForProject(projectID string) []*Rule {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ruleSet.GetRulesForProject(projectID)
}

// GetRulesContent 获取规则内容（用于 AI 上下文注入）
func (m *Manager) GetRulesContent(projectID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ruleSet.GetRulesContent(projectID)
}

// AddGlobalRule 添加全局规则
func (m *Manager) AddGlobalRule(rule *Rule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ruleSet.AddGlobalRule(rule)
}

// AddProjectRule 添加项目规则
func (m *Manager) AddProjectRule(projectID string, rule *Rule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ruleSet.AddProjectRule(projectID, rule)
}

// loadRulesFromFile 从文件加载规则
func (m *Manager) loadRulesFromFile(filePath string, scope Scope) ([]*Rule, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return m.parseMarkdownRules(string(content), filePath, scope)
}

// parseMarkdownRules 解析 Markdown 格式的规则
func (m *Manager) parseMarkdownRules(content, sourcePath string, scope Scope) ([]*Rule, error) {
	var rules []*Rule

	// 整个文件作为一条规则
	title := extractTitle(content)
	if title == "" {
		title = filepath.Base(sourcePath)
	}

	rule := &Rule{
		ID:         generateID(),
		Scope:      scope,
		Title:      title,
		Content:    content,
		Source:     "file",
		SourcePath: sourcePath,
		Priority:   0,
		Enabled:    true,
	}
	rules = append(rules, rule)

	return rules, nil
}

// extractTitle 从 Markdown 中提取标题
func extractTitle(content string) string {
	lines := strings.SplitSeq(content, "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "# "); ok {
			return after
		}
	}
	return ""
}

// Loader 规则加载器接口
type Loader interface {
	// LoadGlobalRules 加载全局规则
	LoadGlobalRules(ctx context.Context) ([]*Rule, error)

	// LoadProjectRules 加载项目规则
	LoadProjectRules(ctx context.Context, projectID string) ([]*Rule, error)
}
