package main

import (
	"context"
	"os"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/builtin"
	"github.com/astercloud/aster/pkg/types"
)

// createTestAgent 创建测试用 Agent
func createTestAgent(apiKey string) (*agent.Agent, error) {
	baseURL := os.Getenv("OPENROUTER_BASE_URL")
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}

	// 创建依赖
	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry)

	jsonStore, err := store.NewJSONStore(".aster")
	if err != nil {
		return nil, err
	}

	templateRegistry := agent.NewTemplateRegistry()
	templateRegistry.Register(&types.AgentTemplateDefinition{
		ID:           "test-assistant",
		SystemPrompt: "你是一个文件操作助手。当用户要求创建、写入或读取文件时，你必须使用 Write 或 Read 工具实际执行操作。当用户要求执行命令时，你必须使用 Bash 工具。不要只用文字回复，必须调用工具完成任务。",
		Tools:        []interface{}{"Read", "Write", "Bash"},
	})

	deps := &agent.Dependencies{
		Store:            jsonStore,
		SandboxFactory:   sandbox.NewFactory(),
		ToolRegistry:     toolRegistry,
		ProviderFactory:  &provider.OpenRouterFactory{},
		TemplateRegistry: templateRegistry,
	}

	config := &types.AgentConfig{
		TemplateID: "test-assistant",
		ModelConfig: &types.ModelConfig{
			Provider:      "openrouter",
			Model:         "anthropic/claude-haiku-4.5",
			APIKey:        apiKey,
			BaseURL:       baseURL,
			ExecutionMode: types.ExecutionModeNonStreaming,
		},
		Sandbox: &types.SandboxConfig{
			Kind:    types.SandboxKindLocal,
			WorkDir: "./workspace",
		},
	}

	return agent.Create(context.TODO(), config, deps)
}
