package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/sandbox"
)

// ExecutorConfig 执行器配置
type ExecutorConfig struct {
	Loader       *CommandLoader
	Sandbox      sandbox.Sandbox
	Provider     provider.Provider
	Capabilities provider.ProviderCapabilities
}

// Executor 命令执行器
type Executor struct {
	loader       *CommandLoader
	sandbox      sandbox.Sandbox
	provider     provider.Provider
	capabilities provider.ProviderCapabilities
}

// NewExecutor 创建执行器
func NewExecutor(config *ExecutorConfig) *Executor {
	return &Executor{
		loader:       config.Loader,
		sandbox:      config.Sandbox,
		provider:     config.Provider,
		capabilities: config.Capabilities,
	}
}

// IsSlashCommand 检查消息是否为斜杠命令
func (e *Executor) IsSlashCommand(message string) bool {
	return strings.HasPrefix(strings.TrimSpace(message), "/")
}

// Execute 执行命令
func (e *Executor) Execute(ctx context.Context, commandName string, args map[string]string) (string, error) {
	// 1. 加载命令定义
	cmd, err := e.loader.Load(ctx, commandName)
	if err != nil {
		return "", fmt.Errorf("load command: %w", err)
	}

	// 2. 执行前置脚本（如果有）
	if cmd.Scripts.Sh != "" {
		if err := e.executeScript(ctx, cmd.Scripts.Sh); err != nil {
			return "", fmt.Errorf("execute script: %w", err)
		}
	}

	// 3. 渲染提示词
	prompt := e.renderPrompt(cmd.PromptTemplate, args)

	// 4. 构建完整的命令消息
	message := e.buildCommandMessage(cmd, prompt)

	return message, nil
}

// executeScript 执行前置脚本
func (e *Executor) executeScript(ctx context.Context, scriptPath string) error {
	result, err := e.sandbox.Exec(ctx, "bash "+scriptPath, nil)
	if err != nil {
		return err
	}

	if result.Code != 0 {
		return fmt.Errorf("script failed with code %d: %s", result.Code, result.Stderr)
	}

	return nil
}

// renderPrompt 渲染提示词模板
func (e *Executor) renderPrompt(template string, args map[string]string) string {
	result := template

	// 替换占位符
	for key, value := range args {
		placeholder := "{" + strings.ToUpper(key) + "}"
		result = strings.ReplaceAll(result, placeholder, value)
	}

	// 替换 {SCRIPT} 占位符
	if e.sandbox != nil {
		result = strings.ReplaceAll(result, "{SCRIPT}", "script")
	}

	return result
}

// buildCommandMessage 构建命令消息
func (e *Executor) buildCommandMessage(cmd *CommandDefinition, prompt string) string {
	var builder strings.Builder

	// 添加命令标识
	builder.WriteString(fmt.Sprintf("## Executing Command: /%s\n\n", cmd.Name))

	if cmd.Description != "" {
		builder.WriteString(fmt.Sprintf("**Description**: %s\n\n", cmd.Description))
	}

	// 添加提示词内容
	builder.WriteString(prompt)

	return builder.String()
}

// CheckCapabilities 检查模型能力
func (e *Executor) CheckCapabilities(cmd *CommandDefinition) error {
	// 检查是否满足最小能力要求
	for _, capability := range cmd.Models.MinimumCapabilities {
		switch capability {
		case "tool-calling":
			if !e.capabilities.SupportToolCalling {
				return errors.New("command requires tool-calling support")
			}
		case "system-prompt":
			if !e.capabilities.SupportSystemPrompt {
				return errors.New("command requires system-prompt support")
			}
		}
	}

	return nil
}
