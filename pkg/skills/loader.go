package skills

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/astercloud/aster/pkg/sandbox"
)

var skillNamePattern = regexp.MustCompile(`^[a-z0-9-]{1,64}$`)

// SkillLoader 技能加载器
type SkillLoader struct {
	baseDir string            // 技能目录路径
	fs      sandbox.SandboxFS // 文件系统接口
}

// NewLoader 创建加载器
func NewLoader(baseDir string, fs sandbox.SandboxFS) *SkillLoader {
	if baseDir == "" {
		baseDir = "skills"
	}
	return &SkillLoader{
		baseDir: baseDir,
		fs:      fs,
	}
}

// Load 加载技能定义
func (sl *SkillLoader) Load(ctx context.Context, skillPath string) (*SkillDefinition, error) {
	// skillPath 可能是 "consistency-checker" 或 "quality-assurance/consistency-checker"
	// 需要找到对应的 SKILL.md 文件
	fullPath := filepath.Join(sl.baseDir, skillPath, "SKILL.md")

	content, err := sl.fs.Read(ctx, fullPath)
	if err != nil {
		return nil, fmt.Errorf("read skill file: %w", err)
	}

	return sl.parse(skillPath, content)
}

// parse 解析技能文件
func (sl *SkillLoader) parse(name, content string) (*SkillDefinition, error) {
	// 分割 YAML frontmatter 和 Markdown 内容
	parts := strings.Split(content, "---")
	if len(parts) < 3 {
		return nil, errors.New("invalid skill format: missing YAML frontmatter")
	}

	skill := &SkillDefinition{
		// Name 默认使用传入的 skillPath，稍后如果 YAML 中显式声明了 name 会覆盖。
		Name:    name,
		Path:    name,
		BaseDir: sl.baseDir,
	}

	// 解析 YAML frontmatter
	yamlContent := strings.TrimSpace(parts[1])
	if err := sl.parseYAML(yamlContent, skill); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}

	// 提取 Markdown 内容作为知识库
	skill.KnowledgeBase = strings.TrimSpace(strings.Join(parts[2:], "---"))

	return skill, nil
}

// parseYAML 解析 YAML 配置
func (sl *SkillLoader) parseYAML(yamlContent string, skill *SkillDefinition) error {
	var config struct {
		Name         string   `yaml:"name"`
		Description  string   `yaml:"description"`
		AllowedTools []string `yaml:"allowed-tools"`

		Kind string `yaml:"kind"`

		Parameters map[string]struct {
			Type        string   `yaml:"type"`
			Description string   `yaml:"description"`
			Required    bool     `yaml:"required"`
			Enum        []string `yaml:"enum"`
		} `yaml:"parameters"`

		Returns map[string]struct {
			Type        string `yaml:"type"`
			Description string `yaml:"description"`
		} `yaml:"returns"`

		Executable *struct {
			Runtime        string `yaml:"runtime"`
			Entry          string `yaml:"entry"`
			TimeoutSeconds int    `yaml:"timeout"`
		} `yaml:"executable"`

		Triggers []struct {
			Type      string   `yaml:"type"`
			Keywords  []string `yaml:"keywords"`
			Condition string   `yaml:"condition"`
			Pattern   string   `yaml:"pattern"`
		} `yaml:"triggers"`
	}

	if err := yaml.Unmarshal([]byte(yamlContent), &config); err != nil {
		return err
	}

	if config.Name != "" {
		skill.Name = config.Name
	}
	skill.Description = config.Description
	skill.AllowedTools = config.AllowedTools

	// 验证名称和描述
	if skill.Name == "" {
		return errors.New("skill name is required")
	}
	if !skillNamePattern.MatchString(skill.Name) {
		return fmt.Errorf("invalid skill name %q: must be 1-64 characters of lowercase letters, numbers, and hyphens", skill.Name)
	}
	if strings.ContainsAny(skill.Name, "<>") {
		return fmt.Errorf("invalid skill name %q: cannot contain XML/meta characters like '<' or '>'", skill.Name)
	}
	lowerName := strings.ToLower(skill.Name)
	if strings.Contains(lowerName, "anthropic") || strings.Contains(lowerName, "claude") {
		return fmt.Errorf("invalid skill name %q: reserved words 'anthropic' and 'claude' are not allowed", skill.Name)
	}

	if skill.Description == "" {
		return fmt.Errorf("description is required for skill %q", skill.Name)
	}
	if len(skill.Description) > 1024 {
		return fmt.Errorf("description too long for skill %q: maximum 1024 characters", skill.Name)
	}
	if strings.ContainsAny(skill.Description, "<>") {
		return fmt.Errorf("description for skill %q cannot contain XML/meta characters like '<' or '>'", skill.Name)
	}

	// 类型
	if config.Kind != "" {
		skill.Kind = config.Kind
	}

	// 参数定义
	if len(config.Parameters) > 0 {
		skill.Parameters = make(map[string]ParamSpec, len(config.Parameters))
		for name, p := range config.Parameters {
			skill.Parameters[name] = ParamSpec{
				Type:        p.Type,
				Description: p.Description,
				Required:    p.Required,
				Enum:        p.Enum,
			}
		}
	}

	// 返回值定义
	if len(config.Returns) > 0 {
		skill.Returns = make(map[string]ReturnSpec, len(config.Returns))
		for name, r := range config.Returns {
			skill.Returns[name] = ReturnSpec{
				Type:        r.Type,
				Description: r.Description,
			}
		}
	}

	// 可执行配置
	if config.Executable != nil {
		skill.Executable = &ExecutableConfig{
			Runtime:        config.Executable.Runtime,
			Entry:          config.Executable.Entry,
			TimeoutSeconds: config.Executable.TimeoutSeconds,
		}
		if skill.Kind == "" {
			skill.Kind = "executable"
		}
	}

	// 转换 triggers
	skill.Triggers = make([]TriggerConfig, 0, len(config.Triggers))
	for _, t := range config.Triggers {
		skill.Triggers = append(skill.Triggers, TriggerConfig{
			Type:      t.Type,
			Keywords:  t.Keywords,
			Condition: t.Condition,
			Pattern:   t.Pattern,
		})
	}

	return nil
}

// LoadMultiple 批量加载技能
func (sl *SkillLoader) LoadMultiple(ctx context.Context, skillPaths []string) (map[string]*SkillDefinition, error) {
	skills := make(map[string]*SkillDefinition)

	for _, path := range skillPaths {
		skill, err := sl.Load(ctx, path)
		if err != nil {
			// 记录错误但继续加载其他技能
			fmt.Printf("[Skills Loader] Failed to load skill '%s': %v\n", path, err)
			continue
		}
		skills[path] = skill
		fmt.Printf("[Skills Loader] Successfully loaded skill: %s\n", path)
	}

	return skills, nil
}

// Discover 发现所有可用技能
func (sl *SkillLoader) Discover(ctx context.Context) ([]string, error) {
	// 递归搜索所有 SKILL.md 文件
	pattern := "**/SKILL.md"
	files, err := sl.fs.Glob(ctx, pattern, &sandbox.GlobOptions{
		CWD:      sl.baseDir,
		Absolute: false,
	})
	if err != nil {
		return nil, fmt.Errorf("glob skills: %w", err)
	}

	skills := make([]string, 0, len(files))
	for _, file := range files {
		// 提取技能路径（去掉 /SKILL.md）
		skillPath := strings.TrimSuffix(file, "/SKILL.md")
		skills = append(skills, skillPath)
	}

	return skills, nil
}
