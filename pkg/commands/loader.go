package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/astercloud/aster/pkg/sandbox"
)

// CommandLoader 命令加载器
type CommandLoader struct {
	baseDir string            // 命令目录路径
	fs      sandbox.SandboxFS // 文件系统接口
}

// NewLoader 创建加载器
func NewLoader(baseDir string, fs sandbox.SandboxFS) *CommandLoader {
	if baseDir == "" {
		baseDir = "commands"
	}
	return &CommandLoader{
		baseDir: baseDir,
		fs:      fs,
	}
}

// Load 加载命令定义
func (cl *CommandLoader) Load(ctx context.Context, commandName string) (*CommandDefinition, error) {
	// 读取 {commandName}.md
	path := filepath.Join(cl.baseDir, commandName+".md")

	content, err := cl.fs.Read(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("read command file: %w", err)
	}

	return cl.parse(commandName, content)
}

// parse 解析命令文件
func (cl *CommandLoader) parse(name, content string) (*CommandDefinition, error) {
	// 分割 YAML frontmatter 和 Markdown 内容
	// 格式: ---\nyaml\n---\nmarkdown
	parts := strings.Split(content, "---")
	if len(parts) < 3 {
		return nil, errors.New("invalid command format: missing YAML frontmatter")
	}

	cmd := &CommandDefinition{
		Name: name,
	}

	// 解析 YAML frontmatter
	yamlContent := strings.TrimSpace(parts[1])
	if err := cl.parseYAML(yamlContent, cmd); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}

	// 提取 Markdown 内容
	cmd.PromptTemplate = strings.TrimSpace(strings.Join(parts[2:], "---"))

	return cmd, nil
}

// parseYAML 解析 YAML 配置
func (cl *CommandLoader) parseYAML(yamlContent string, cmd *CommandDefinition) error {
	var config struct {
		Description  string   `yaml:"description"`
		ArgumentHint string   `yaml:"argument-hint"`
		AllowedTools []string `yaml:"allowed-tools"`
		Models       struct {
			Preferred           []string `yaml:"preferred"`
			MinimumCapabilities []string `yaml:"minimum-capabilities"`
		} `yaml:"models"`
		Scripts struct {
			Sh string `yaml:"sh"`
			Ps string `yaml:"ps"`
		} `yaml:"scripts"`
	}

	if err := yaml.Unmarshal([]byte(yamlContent), &config); err != nil {
		return err
	}

	cmd.Description = config.Description
	cmd.ArgumentHint = config.ArgumentHint
	cmd.AllowedTools = config.AllowedTools
	cmd.Models.Preferred = config.Models.Preferred
	cmd.Models.MinimumCapabilities = config.Models.MinimumCapabilities
	cmd.Scripts.Sh = config.Scripts.Sh
	cmd.Scripts.Ps = config.Scripts.Ps

	return nil
}

// List 列出所有可用命令
func (cl *CommandLoader) List(ctx context.Context) ([]string, error) {
	// 使用 Glob 匹配所有 .md 文件
	pattern := "*.md"
	files, err := cl.fs.Glob(ctx, pattern, &sandbox.GlobOptions{
		CWD:      cl.baseDir,
		Absolute: false,
	})
	if err != nil {
		return nil, fmt.Errorf("glob commands: %w", err)
	}

	commands := make([]string, 0, len(files))
	for _, file := range files {
		// 去掉 .md 后缀
		name := strings.TrimSuffix(filepath.Base(file), ".md")
		commands = append(commands, name)
	}

	return commands, nil
}

// LoadLocal 从本地文件系统加载（用于开发）
func LoadLocal(baseDir, commandName string) (*CommandDefinition, error) {
	path := filepath.Join(baseDir, commandName+".md")
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read command file: %w", err)
	}

	loader := &CommandLoader{baseDir: baseDir}
	return loader.parse(commandName, string(content))
}
