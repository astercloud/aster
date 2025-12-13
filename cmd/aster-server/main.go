package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/router"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/builtin"
	"github.com/astercloud/aster/pkg/types"
	"github.com/astercloud/aster/server"
)

// ProviderConfig 定义 provider 的默认配置
type ProviderConfig struct {
	DefaultModel string   // 默认模型名称
	APIKeyEnvs   []string // 可能的 API Key 环境变量名称
}

// providerDefaults 定义各个 provider 的默认配置
var providerDefaults = map[string]ProviderConfig{
	"anthropic": {
		DefaultModel: "claude-sonnet-4-5",
		APIKeyEnvs:   []string{"ANTHROPIC_API_KEY"},
	},
	"deepseek": {
		DefaultModel: "deepseek-chat",
		APIKeyEnvs:   []string{"DEEPSEEK_API_KEY"},
	},
	"glm": {
		DefaultModel: "glm-4-plus", // 或使用 glm-z1-airx 获得推理能力
		APIKeyEnvs:   []string{"GLM_API_KEY", "ZHIPU_API_KEY", "BIGMODEL_API_KEY"},
	},
	"openai": {
		DefaultModel: "gpt-4o",
		APIKeyEnvs:   []string{"OPENAI_API_KEY"},
	},
	"gemini": {
		DefaultModel: "gemini-2.0-flash",
		APIKeyEnvs:   []string{"GEMINI_API_KEY", "GOOGLE_API_KEY"},
	},
	"moonshot": {
		DefaultModel: "moonshot-v1-8k", // 基础模型，测试用
		APIKeyEnvs:   []string{"MOONSHOT_API_KEY", "KIMI_API_KEY"},
	},
	"doubao": {
		DefaultModel: "doubao-pro-32k",
		APIKeyEnvs:   []string{"DOUBAO_API_KEY", "BYTEDANCE_API_KEY"},
	},
}

// providerAliases 定义 provider 别名映射
var providerAliases = map[string]string{
	"zhipu":     "glm",
	"bigmodel":  "glm",
	"google":    "gemini",
	"kimi":      "moonshot",
	"bytedance": "doubao",
}

// resolveProviderConfig 解析 provider 配置
func resolveProviderConfig(providerName, modelName string) (provider, model, apiKey string) {
	// 解析别名
	if alias, ok := providerAliases[providerName]; ok {
		providerName = alias
	}

	// 获取 provider 配置
	config, ok := providerDefaults[providerName]
	if !ok {
		// 未知 provider，使用通用方式获取 API Key
		return providerName, modelName, os.Getenv(strings.ToUpper(providerName) + "_API_KEY")
	}

	// 使用默认模型（如果未指定）
	if modelName == "" {
		modelName = config.DefaultModel
	}

	// 尝试获取 API Key
	for _, envName := range config.APIKeyEnvs {
		if key := os.Getenv(envName); key != "" {
			return providerName, modelName, key
		}
	}

	return providerName, modelName, ""
}

// maskAPIKey 隐藏 API Key 的中间部分
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func main() {
	fmt.Println("🚀 aster 星尘云枢 Production Server")
	fmt.Println("================================")

	// Initialize store
	// 支持通过环境变量配置存储类型: json (默认), mysql, redis
	var st store.Store
	var err error

	storeType := os.Getenv("ASTER_STORE_TYPE")
	switch storeType {
	case "mysql":
		mysqlDSN := os.Getenv("ASTER_MYSQL_DSN")
		if mysqlDSN == "" {
			log.Fatalf("ASTER_MYSQL_DSN is required when ASTER_STORE_TYPE=mysql")
		}
		fmt.Printf("📁 Using MySQL store\n")
		st, err = store.NewMySQLStore(store.MySQLConfig{
			DSN: mysqlDSN,
		})

	case "redis":
		redisAddr := os.Getenv("ASTER_REDIS_ADDR")
		if redisAddr == "" {
			redisAddr = "localhost:6379"
		}
		redisPassword := os.Getenv("ASTER_REDIS_PASSWORD")
		redisDB := 0
		if dbStr := os.Getenv("ASTER_REDIS_DB"); dbStr != "" {
			_, _ = fmt.Sscanf(dbStr, "%d", &redisDB)
		}
		redisPrefix := os.Getenv("ASTER_REDIS_PREFIX")
		if redisPrefix == "" {
			redisPrefix = "aster:"
		}
		fmt.Printf("📁 Using Redis store: %s\n", redisAddr)
		st, err = store.NewRedisStore(store.RedisConfig{
			Addr:     redisAddr,
			Password: redisPassword,
			DB:       redisDB,
			Prefix:   redisPrefix,
		})

	default:
		// 默认使用 JSON Store
		dataDir := os.Getenv("ASTER_DATA_DIR")
		if dataDir == "" {
			dataDir = ".data"
		}
		fmt.Printf("📁 Using JSON store: %s\n", dataDir)
		st, err = store.NewJSONStore(dataDir)
	}
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}

	// Initialize tool registry
	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry)

	// Initialize factories
	sandboxFactory := sandbox.NewFactory()
	providerFactory := provider.NewMultiProviderFactory()

	// Initialize template registry
	templateRegistry := agent.NewTemplateRegistry()

	// Register builtin templates
	registerDefaultTemplates(templateRegistry)

	// Initialize router with environment-based configuration
	providerEnv := os.Getenv("PROVIDER")
	if providerEnv == "" {
		providerEnv = "anthropic"
	}
	modelEnv := os.Getenv("MODEL")

	// 解析 provider 配置
	resolvedProvider, resolvedModel, apiKey := resolveProviderConfig(providerEnv, modelEnv)
	log.Printf("[Config] Provider: %s, Model: %s, APIKey: %s...", resolvedProvider, resolvedModel, maskAPIKey(apiKey))

	defaultModel := &types.ModelConfig{
		Provider: resolvedProvider,
		Model:    resolvedModel,
		APIKey:   apiKey,
	}
	routes := []router.StaticRouteEntry{
		{Task: "chat", Priority: router.PriorityQuality, Model: defaultModel},
	}
	rt := router.NewStaticRouter(defaultModel, routes)

	// Create prompt compressor for context compression
	// 需要先创建一个 Provider 用于 LLM 压缩
	compressionProvider, err := providerFactory.Create(&types.ModelConfig{
		Provider: resolvedProvider,
		Model:    resolvedModel,
		APIKey:   apiKey,
	})
	if err != nil {
		log.Printf("Warning: Failed to create compression provider: %v (compression will be disabled)", err)
	}
	var promptCompressor *agent.EnhancedPromptCompressor
	if compressionProvider != nil {
		promptCompressor = agent.NewEnhancedPromptCompressor(compressionProvider, "zh")
		fmt.Println("✅ Prompt compressor initialized")
	}

	// Create agent dependencies
	agentDeps := &agent.Dependencies{
		Store:            st,
		ToolRegistry:     toolRegistry,
		SandboxFactory:   sandboxFactory,
		ProviderFactory:  providerFactory,
		TemplateRegistry: templateRegistry,
		Router:           rt,
		PromptCompressor: promptCompressor,
	}

	// 初始化 SubAgentManager 并注入到 Task 工具
	agent.InitializeTaskExecutor(agentDeps)
	fmt.Println("✅ SubAgentManager initialized for Task tool")

	// Create server dependencies
	deps := &server.Dependencies{
		Store:     st,
		AgentDeps: agentDeps,
	}

	// Load configuration (use default for now)
	config := server.DefaultConfig()

	// Override with environment variables if needed
	if port := os.Getenv("PORT"); port != "" {
		_, _ = fmt.Sscanf(port, "%d", &config.Port)
	}
	if host := os.Getenv("HOST"); host != "" {
		config.Host = host
	}
	if apiKey := os.Getenv("API_KEY"); apiKey != "" {
		config.Auth.APIKey.Keys = []string{apiKey}
	}

	// Create server
	srv, err := server.New(config, deps)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start server in a goroutine
	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n🛑 Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Stop(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	fmt.Println("✅ Server exited properly")
}

// registerDefaultTemplates registers builtin agent templates
func registerDefaultTemplates(registry *agent.TemplateRegistry) {
	// Get provider and model from environment using the shared resolver
	providerEnv := os.Getenv("PROVIDER")
	if providerEnv == "" {
		providerEnv = "anthropic"
	}
	modelEnv := os.Getenv("MODEL")

	_, model, _ := resolveProviderConfig(providerEnv, modelEnv)

	// Register "chat" template - simple chat agent with prompt compression
	registry.Register(&types.AgentTemplateDefinition{
		ID:    "chat",
		Model: model,
		SystemPrompt: `You are a helpful AI assistant with access to various tools. When users ask you to:

1. Read files or directories: Use the Read tool
2. Write or edit files: Use the Write or Edit tools
3. Search for files: Use the Glob tool
4. Search within files: Use the Grep tool
5. Execute commands: Use the Bash tool
6. Fetch web content: Use the WebFetch tool
7. Search the web: Use the WebSearch tool

Always use the appropriate tool when possible instead of just explaining what you would do. Tools help you actually perform tasks for the user.

When you receive a tool request, think about what tool is needed and use it. After using a tool, explain what you found or did.

If you're unsure whether to use a tool, err on the side of using it - it's better to try and help than to just describe.

## Additional Context
This is additional content to make the system prompt longer for testing compression.
The compression system will automatically activate when the prompt exceeds the threshold.

### Security Guidelines
IMPORTANT: Always follow security best practices:
- Never expose sensitive credentials
- Validate all user inputs
- Use secure connections when possible

### Performance Tips
- Use streaming for large outputs
- Cache frequently accessed data
- Minimize unnecessary API calls

### Code Quality Standards
- Follow consistent coding conventions
- Write meaningful comments
- Include error handling

### Testing Requirements
- Write unit tests for new features
- Ensure backward compatibility
- Test edge cases thoroughly`,
		Tools: "*", // Enable all tools
		Runtime: &types.AgentTemplateRuntime{
			PromptCompression: &types.PromptCompressionConfig{
				Enabled:          true,
				MaxLength:        1500, // 降低阈值便于测试
				TargetLength:     800,
				Mode:             "hybrid",
				Level:            2,
				PreserveSections: []string{"Tools Manual", "Security Guidelines"},
				CacheEnabled:     true,
				Language:         "zh",
			},
			// 对话历史压缩配置 - 类似 Claude Code 的 tokenBudget 机制
			ConversationCompression: &types.ConversationCompressionConfig{
				Enabled:           true,
				TokenBudget:       5000, // 降低预算便于测试 (生产环境建议 200000)
				Threshold:         0.80, // 80% 使用率触发压缩
				MinMessagesToKeep: 4,    // 保留最近 4 条消息
				SummaryLanguage:   "zh",
				UseLLMSummarizer:  false, // 使用规则摘要，速度更快
			},
		},
	})

	// Register "default-agent" template (alias for chat)
	registry.Register(&types.AgentTemplateDefinition{
		ID:    "default-agent",
		Model: model,
		SystemPrompt: `You are a helpful AI assistant with access to various tools. When users ask you to:

1. Read files or directories: Use the Read tool
2. Write or edit files: Use the Write or Edit tools
3. Search for files: Use the Glob tool
4. Search within files: Use the Grep tool
5. Execute commands: Use the Bash tool
6. Fetch web content: Use the WebFetch tool
7. Search the web: Use the WebSearch tool

Always use the appropriate tool when possible instead of just explaining what you would do. Tools help you actually perform tasks for the user.

When you receive a tool request, think about what tool is needed and use it. After using a tool, explain what you found or did.

If you're unsure whether to use a tool, err on the side of using it - it's better to try and help than to just describe.`,
		Tools: "*",
	})

	// Register "code-assistant" template
	registry.Register(&types.AgentTemplateDefinition{
		ID:    "code-assistant",
		Model: model,
		SystemPrompt: `You are an expert programming assistant with access to various tools. When users ask for code-related help:

1. Read source files: Use the Read tool
2. Write or edit code files: Use the Write or Edit tools
3. Search for code patterns: Use the Grep tool
4. Find files by pattern: Use the Glob tool
5. Build/test code: Use the Bash tool
6. Check documentation: Use the WebFetch or WebSearch tools

Always prefer to actually read the code, make the changes, or run the commands rather than just describing what to do. Use your tools to examine real code and make real modifications.

Explain what you're doing and why, but focus on actually solving the programming problem using available tools.`,
		Tools: "*",
	})

	fmt.Printf("✅ Registered default templates (Provider: %s, Model: %s)\n", providerEnv, model)
}
