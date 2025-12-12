package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/astercloud/aster/pkg/types"
	"gopkg.in/yaml.v3"
)

// Loader YAML 配置加载器
// 支持从文件加载配置，并展开环境变量
type Loader struct {
	// 是否展开环境变量
	expandEnv bool

	// 环境变量前缀（可选）
	envPrefix string

	// 自定义变量映射
	variables map[string]string
}

// LoaderOption 加载器选项
type LoaderOption func(*Loader)

// WithEnvExpansion 启用环境变量展开
func WithEnvExpansion(enabled bool) LoaderOption {
	return func(l *Loader) {
		l.expandEnv = enabled
	}
}

// WithEnvPrefix 设置环境变量前缀
func WithEnvPrefix(prefix string) LoaderOption {
	return func(l *Loader) {
		l.envPrefix = prefix
	}
}

// WithVariables 设置自定义变量
func WithVariables(vars map[string]string) LoaderOption {
	return func(l *Loader) {
		l.variables = vars
	}
}

// NewLoader 创建配置加载器
func NewLoader(opts ...LoaderOption) *Loader {
	l := &Loader{
		expandEnv: true, // 默认启用环境变量展开
		variables: make(map[string]string),
	}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

// LoadAgentConfig 从文件加载 AgentConfig
func (l *Loader) LoadAgentConfig(path string) (*types.AgentConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	// 展开环境变量
	if l.expandEnv {
		content = []byte(l.expandVariables(string(content)))
	}

	var config types.AgentConfig
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("parse yaml config: %w", err)
	}

	// 验证配置
	if err := l.validateAgentConfig(&config); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &config, nil
}

// LoadModelConfig 从文件加载 ModelConfig
func (l *Loader) LoadModelConfig(path string) (*types.ModelConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	if l.expandEnv {
		content = []byte(l.expandVariables(string(content)))
	}

	var config types.ModelConfig
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("parse yaml config: %w", err)
	}

	return &config, nil
}

// LoadFromString 从字符串加载配置
func (l *Loader) LoadFromString(content string, v any) error {
	if l.expandEnv {
		content = l.expandVariables(content)
	}

	if err := yaml.Unmarshal([]byte(content), v); err != nil {
		return fmt.Errorf("parse yaml: %w", err)
	}

	return nil
}

// expandVariables 展开变量
// 支持的格式:
// - ${VAR_NAME} - 环境变量，如果不存在则为空
// - ${VAR_NAME:-default} - 环境变量，带默认值
// - ${VAR_NAME:?error message} - 环境变量，必须存在，否则报错
// - $VAR_NAME - 简单环境变量引用
func (l *Loader) expandVariables(content string) string {
	// 匹配 ${VAR_NAME}, ${VAR_NAME:-default}, ${VAR_NAME:?error}
	re := regexp.MustCompile(`\$\{([^}]+)\}`)

	result := re.ReplaceAllStringFunc(content, func(match string) string {
		// 去掉 ${ 和 }
		inner := match[2 : len(match)-1]

		// 检查是否有默认值或必需标记
		var varName, defaultValue string
		var required bool
		var errorMsg string

		if idx := strings.Index(inner, ":-"); idx != -1 {
			// 有默认值
			varName = inner[:idx]
			defaultValue = inner[idx+2:]
		} else if idx := strings.Index(inner, ":?"); idx != -1 {
			// 必需变量
			varName = inner[:idx]
			required = true
			errorMsg = inner[idx+2:]
		} else {
			varName = inner
		}

		// 添加前缀
		fullVarName := varName
		if l.envPrefix != "" {
			fullVarName = l.envPrefix + varName
		}

		// 先检查自定义变量
		if val, ok := l.variables[varName]; ok {
			return val
		}

		// 再检查环境变量
		val := os.Getenv(fullVarName)
		if val == "" {
			// 尝试不带前缀的变量名
			val = os.Getenv(varName)
		}

		if val != "" {
			return val
		}

		if required && val == "" {
			// 返回错误标记，后续验证会捕获
			return fmt.Sprintf("__ERROR__: %s", errorMsg)
		}

		return defaultValue
	})

	// 处理简单的 $VAR_NAME 格式
	simpleRe := regexp.MustCompile(`\$([A-Za-z_][A-Za-z0-9_]*)`)
	result = simpleRe.ReplaceAllStringFunc(result, func(match string) string {
		varName := match[1:]

		// 跳过已经处理过的 ${} 格式
		if strings.HasPrefix(varName, "{") {
			return match
		}

		// 先检查自定义变量
		if val, ok := l.variables[varName]; ok {
			return val
		}

		// 添加前缀
		fullVarName := varName
		if l.envPrefix != "" {
			fullVarName = l.envPrefix + varName
		}

		// 检查环境变量
		if val := os.Getenv(fullVarName); val != "" {
			return val
		}
		if val := os.Getenv(varName); val != "" {
			return val
		}

		return ""
	})

	return result
}

// validateAgentConfig 验证 AgentConfig
func (l *Loader) validateAgentConfig(config *types.AgentConfig) error {
	// 检查必需字段
	if config.TemplateID == "" {
		return fmt.Errorf("template_id is required")
	}

	// 检查是否有未展开的必需变量
	if strings.Contains(fmt.Sprintf("%v", config), "__ERROR__:") {
		return fmt.Errorf("config contains unexpanded required variables")
	}

	// 验证 ModelConfig
	if config.ModelConfig != nil {
		if err := l.validateModelConfig(config.ModelConfig); err != nil {
			return fmt.Errorf("model_config: %w", err)
		}
	}

	// 验证 Multitenancy
	if config.Multitenancy != nil && config.Multitenancy.Enabled {
		if config.Multitenancy.OrgID == "" && config.Multitenancy.TenantID == "" {
			return fmt.Errorf("multitenancy enabled but neither org_id nor tenant_id specified")
		}
	}

	return nil
}

// validateModelConfig 验证 ModelConfig
func (l *Loader) validateModelConfig(config *types.ModelConfig) error {
	if config.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	if config.Model == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}

// MergeConfigs 合并多个配置（后面的覆盖前面的）
func MergeConfigs(base *types.AgentConfig, overlays ...*types.AgentConfig) *types.AgentConfig {
	if base == nil {
		base = &types.AgentConfig{}
	}

	for _, overlay := range overlays {
		if overlay == nil {
			continue
		}

		// 合并基本字段
		if overlay.AgentID != "" {
			base.AgentID = overlay.AgentID
		}
		if overlay.TemplateID != "" {
			base.TemplateID = overlay.TemplateID
		}
		if overlay.TemplateVersion != "" {
			base.TemplateVersion = overlay.TemplateVersion
		}
		if overlay.RoutingProfile != "" {
			base.RoutingProfile = overlay.RoutingProfile
		}

		// 合并 ModelConfig
		if overlay.ModelConfig != nil {
			if base.ModelConfig == nil {
				base.ModelConfig = overlay.ModelConfig
			} else {
				mergeModelConfig(base.ModelConfig, overlay.ModelConfig)
			}
		}

		// 合并 Tools（追加）
		if len(overlay.Tools) > 0 {
			base.Tools = append(base.Tools, overlay.Tools...)
		}

		// 合并 Middlewares（追加）
		if len(overlay.Middlewares) > 0 {
			base.Middlewares = append(base.Middlewares, overlay.Middlewares...)
		}

		// 合并 Multitenancy
		if overlay.Multitenancy != nil {
			base.Multitenancy = overlay.Multitenancy
		}

		// 合并 Memory
		if overlay.Memory != nil {
			base.Memory = overlay.Memory
		}

		// 合并 Metadata
		if overlay.Metadata != nil {
			if base.Metadata == nil {
				base.Metadata = make(map[string]any)
			}
			for k, v := range overlay.Metadata {
				base.Metadata[k] = v
			}
		}
	}

	return base
}

// mergeModelConfig 合并 ModelConfig
func mergeModelConfig(base, overlay *types.ModelConfig) {
	if overlay.Provider != "" {
		base.Provider = overlay.Provider
	}
	if overlay.Model != "" {
		base.Model = overlay.Model
	}
	if overlay.APIKey != "" {
		base.APIKey = overlay.APIKey
	}
	if overlay.BaseURL != "" {
		base.BaseURL = overlay.BaseURL
	}
	if overlay.ExecutionMode != "" {
		base.ExecutionMode = overlay.ExecutionMode
	}
}

// LoadDefault 从默认位置加载配置
// 查找顺序:
// 1. ./agent.yaml
// 2. ./config/agent.yaml
// 3. ~/.aster/agent.yaml
func LoadDefault() (*types.AgentConfig, error) {
	loader := NewLoader()

	paths := []string{
		"agent.yaml",
		"agent.yml",
		"config/agent.yaml",
		"config/agent.yml",
	}

	// 添加用户目录
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, home+"/.aster/agent.yaml")
		paths = append(paths, home+"/.aster/agent.yml")
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return loader.LoadAgentConfig(path)
		}
	}

	return nil, fmt.Errorf("no default config file found in: %v", paths)
}

// MustLoad 必须成功加载配置，否则 panic
func MustLoad(path string) *types.AgentConfig {
	loader := NewLoader()
	config, err := loader.LoadAgentConfig(path)
	if err != nil {
		panic(fmt.Sprintf("failed to load config from %s: %v", path, err))
	}
	return config
}
