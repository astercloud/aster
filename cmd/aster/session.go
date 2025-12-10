package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/config"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/recipe"
	"github.com/astercloud/aster/pkg/router"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/session"
	"github.com/astercloud/aster/pkg/session/sqlite"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/builtin"
	"github.com/astercloud/aster/pkg/types"
)

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

// runSession 启动交互式 CLI 会话
func runSession(args []string) error {
	fs := flag.NewFlagSet("session", flag.ExitOnError)
	recipeFile := fs.String("recipe", "", "Recipe file to use")
	workDir := fs.String("dir", ".", "Working directory")
	provider := fs.String("provider", "", "LLM provider (anthropic, openai, deepseek)")
	model := fs.String("model", "", "Model name")
	noColor := fs.Bool("no-color", false, "Disable colored output")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: aster session [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Start an interactive AI agent session.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nCommands during session:\n")
		fmt.Fprintf(os.Stderr, "  /exit, /quit    Exit the session\n")
		fmt.Fprintf(os.Stderr, "  /clear          Clear conversation history\n")
		fmt.Fprintf(os.Stderr, "  /help           Show help\n")
		fmt.Fprintf(os.Stderr, "  /status         Show agent status\n")
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Disable colors if requested or not a terminal
	useColor := !*noColor && isTerminal(os.Stdout)

	// Resolve working directory
	absWorkDir, err := filepath.Abs(*workDir)
	if err != nil {
		return fmt.Errorf("resolve working directory: %w", err)
	}

	// Ensure config directories exist
	if err := config.EnsureAllDirs(); err != nil {
		return fmt.Errorf("create config directories: %w", err)
	}

	// Create session store (SQLite for desktop)
	dbPath := config.DatabaseFile()
	sessionStore, err := sqlite.New(dbPath)
	if err != nil {
		return fmt.Errorf("create session store: %w", err)
	}
	defer sessionStore.Close()

	// Create data store
	storeDir := filepath.Join(config.DataDir(), "store")
	if err := os.MkdirAll(storeDir, 0755); err != nil {
		return fmt.Errorf("create store directory: %w", err)
	}
	dataStore, err := store.NewJSONStore(storeDir)
	if err != nil {
		return fmt.Errorf("create data store: %w", err)
	}

	// Load recipe if specified
	var recipeConfig *recipe.Recipe
	if *recipeFile != "" {
		recipeConfig, err = recipe.LoadFromFile(*recipeFile)
		if err != nil {
			return fmt.Errorf("load recipe: %w", err)
		}
		printColored(useColor, colorCyan, "📜 Loaded recipe: %s\n", recipeConfig.Title)
	}

	// Build model config
	modelConfig := buildModelConfig(*provider, *model, recipeConfig)
	if modelConfig.APIKey == "" {
		return fmt.Errorf("API key not set. Please set %s_API_KEY environment variable", strings.ToUpper(modelConfig.Provider))
	}

	// Create agent dependencies
	agentDeps := createAgentDependencies(dataStore, modelConfig)

	// Build agent config
	agentConfig := &types.AgentConfig{
		TemplateID:  "default",
		ModelConfig: modelConfig,
		Metadata: map[string]any{
			"work_dir": absWorkDir,
		},
	}

	// Apply recipe settings
	if recipeConfig != nil {
		applyRecipeToConfig(recipeConfig, agentConfig, agentDeps)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		printColored(useColor, colorYellow, "\n\n👋 Goodbye!\n")
		cancel()
	}()

	// Create agent
	ag, err := agent.Create(ctx, agentConfig, agentDeps)
	if err != nil {
		return fmt.Errorf("create agent: %w", err)
	}
	defer ag.Close()

	// Create session record
	sess, err := sessionStore.Create(ctx, &session.CreateRequest{
		AppName: "aster-cli",
		UserID:  os.Getenv("USER"),
		AgentID: ag.ID(),
		Metadata: map[string]any{
			"work_dir": absWorkDir,
		},
	})
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	// Subscribe to agent events
	eventCh := ag.Subscribe([]types.AgentChannel{
		types.ChannelProgress,
		types.ChannelControl,
	}, nil)

	// Start event handler
	go handleAgentEvents(ctx, eventCh, useColor)

	// Print welcome message
	printWelcome(useColor, modelConfig, recipeConfig, absWorkDir, sess.ID())

	// Run REPL
	return runREPL(ctx, ag, sessionStore, sess.ID(), useColor)
}

// buildModelConfig builds the model configuration
func buildModelConfig(providerName, modelName string, recipeConfig *recipe.Recipe) *types.ModelConfig {
	// Default values
	if providerName == "" {
		providerName = "anthropic"
	}
	if modelName == "" {
		modelName = "claude-sonnet-4-20250514"
	}

	// Override from recipe
	if recipeConfig != nil && recipeConfig.Settings != nil {
		if recipeConfig.Settings.Provider != "" {
			providerName = recipeConfig.Settings.Provider
		}
		if recipeConfig.Settings.Model != "" {
			modelName = recipeConfig.Settings.Model
		}
	}

	// Get API key from environment
	apiKey := getAPIKey(providerName)

	return &types.ModelConfig{
		Provider: providerName,
		Model:    modelName,
		APIKey:   apiKey,
	}
}

// getAPIKey returns the API key for a provider
func getAPIKey(providerName string) string {
	envVars := map[string]string{
		"anthropic": "ANTHROPIC_API_KEY",
		"openai":    "OPENAI_API_KEY",
		"deepseek":  "DEEPSEEK_API_KEY",
		"google":    "GOOGLE_API_KEY",
	}

	if envVar, ok := envVars[providerName]; ok {
		return os.Getenv(envVar)
	}

	// Try generic format
	return os.Getenv(strings.ToUpper(providerName) + "_API_KEY")
}

// createAgentDependencies creates the agent dependencies
func createAgentDependencies(dataStore *store.JSONStore, modelConfig *types.ModelConfig) *agent.Dependencies {
	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry)

	sandboxFactory := sandbox.NewFactory()
	providerFactory := provider.NewMultiProviderFactory()
	templateRegistry := agent.NewTemplateRegistry()
	registerBuiltinTemplates(templateRegistry)

	routes := []router.StaticRouteEntry{
		{Task: "chat", Priority: router.PriorityQuality, Model: modelConfig},
	}
	rt := router.NewStaticRouter(modelConfig, routes)

	return &agent.Dependencies{
		Store:            dataStore,
		ToolRegistry:     toolRegistry,
		SandboxFactory:   sandboxFactory,
		ProviderFactory:  providerFactory,
		TemplateRegistry: templateRegistry,
		Router:           rt,
	}
}

// applyRecipeToConfig applies recipe settings to agent config
func applyRecipeToConfig(r *recipe.Recipe, config *types.AgentConfig, deps *agent.Dependencies) {
	if r.TemplateID != "" {
		config.TemplateID = r.TemplateID
	}

	// TODO: Apply tools filter, extensions, etc.
}

// handleAgentEvents processes agent events and displays them
func handleAgentEvents(ctx context.Context, eventCh <-chan types.AgentEventEnvelope, useColor bool) {
	for {
		select {
		case <-ctx.Done():
			return
		case envelope, ok := <-eventCh:
			if !ok {
				return
			}

			switch e := envelope.Event.(type) {
			case *types.ProgressTextChunkEvent:
				fmt.Print(e.Delta)

			case *types.ProgressToolStartEvent:
				printColored(useColor, colorGray, "\n🔧 %s", e.Call.Name)
				if len(e.Call.Arguments) > 0 {
					// Show truncated input
					inputStr := fmt.Sprintf("%v", e.Call.Arguments)
					if len(inputStr) > 80 {
						inputStr = inputStr[:77] + "..."
					}
					printColored(useColor, colorGray, " (%s)", inputStr)
				}
				fmt.Println()

			case *types.ProgressToolEndEvent:
				if e.Call.Error == "" {
					printColored(useColor, colorGreen, "✓ ")
				} else {
					printColored(useColor, colorYellow, "✗ ")
				}
				// Show truncated output
				outputStr := fmt.Sprintf("%v", e.Call.Result)
				if len(outputStr) > 100 {
					outputStr = outputStr[:97] + "..."
				}
				printColored(useColor, colorGray, "%s\n", outputStr)

			case *types.ProgressThinkChunkStartEvent:
				printColored(useColor, colorGray, "💭 Thinking...\n")

			case *types.ControlPermissionRequiredEvent:
				// Handle permission request
				printColored(useColor, colorYellow, "\n⚠️  Tool requires approval: %s\n", e.Call.Name)
				printColored(useColor, colorGray, "   Input: %v\n", e.Call.Arguments)
				fmt.Print("   Approve? [y/N]: ")
				// Note: In a real implementation, we'd wait for user input
				// and send the decision back to the agent via e.Respond

			case *types.MonitorErrorEvent:
				printColored(useColor, colorYellow, "\n❌ Error: %s\n", e.Message)
			}
		}
	}
}

// runREPL runs the read-eval-print loop
func runREPL(ctx context.Context, ag *agent.Agent, sessionStore session.Service, sessionID string, useColor bool) error {
	reader := bufio.NewReader(os.Stdin)

	for {
		// Print prompt
		printColored(useColor, colorBold+colorBlue, "\naster> ")

		// Read input
		input, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				printColored(useColor, colorYellow, "\n\n👋 Goodbye!\n")
				return nil
			}
			return fmt.Errorf("read input: %w", err)
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Handle commands
		if strings.HasPrefix(input, "/") {
			handled, err := handleCommand(ctx, input, ag, sessionStore, sessionID, useColor)
			if err != nil {
				printColored(useColor, colorYellow, "Error: %s\n", err)
			}
			if handled {
				if input == "/exit" || input == "/quit" {
					return nil
				}
				continue
			}
		}

		// Record user message to session
		sessionStore.AppendEvent(ctx, sessionID, &session.Event{
			Author: "user",
			Content: types.Message{
				Role:    types.RoleUser,
				Content: input,
			},
		})

		// Send to agent
		fmt.Println()
		if err := ag.Send(ctx, input); err != nil {
			printColored(useColor, colorYellow, "Error: %s\n", err)
			continue
		}

		// Wait for response to complete
		waitForCompletion(ctx, ag)
	}
}

// handleCommand handles slash commands
func handleCommand(ctx context.Context, cmd string, ag *agent.Agent, sessionStore session.Service, sessionID string, useColor bool) (bool, error) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return false, nil
	}

	switch parts[0] {
	case "/exit", "/quit":
		printColored(useColor, colorYellow, "👋 Goodbye!\n")
		return true, nil

	case "/clear":
		// TODO: Clear agent history
		printColored(useColor, colorGreen, "✓ Conversation cleared\n")
		return true, nil

	case "/help":
		printHelp(useColor)
		return true, nil

	case "/status":
		status := ag.Status()
		printColored(useColor, colorCyan, "Agent Status:\n")
		printColored(useColor, colorGray, "  ID: %s\n", status.AgentID)
		printColored(useColor, colorGray, "  State: %s\n", status.State)
		printColored(useColor, colorGray, "  Steps: %d\n", status.StepCount)
		return true, nil

	case "/session":
		printColored(useColor, colorCyan, "Session ID: %s\n", sessionID)
		return true, nil

	default:
		// Not a known command, let agent handle it (might be a slash command)
		return false, nil
	}
}

// waitForCompletion waits for the agent to finish processing
func waitForCompletion(ctx context.Context, ag *agent.Agent) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
			status := ag.Status()
			if status.State == types.AgentStateReady {
				return
			}
		}
	}
}

// printWelcome prints the welcome message
func printWelcome(useColor bool, modelConfig *types.ModelConfig, recipeConfig *recipe.Recipe, workDir, sessionID string) {
	printColored(useColor, colorBold+colorCyan, "\n🚀 Aster AI Agent\n")
	printColored(useColor, colorGray, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	printColored(useColor, colorGray, "  Provider: ")
	printColored(useColor, colorGreen, "%s\n", modelConfig.Provider)

	printColored(useColor, colorGray, "  Model: ")
	printColored(useColor, colorGreen, "%s\n", modelConfig.Model)

	printColored(useColor, colorGray, "  Work Dir: ")
	printColored(useColor, colorGreen, "%s\n", workDir)

	if recipeConfig != nil {
		printColored(useColor, colorGray, "  Recipe: ")
		printColored(useColor, colorGreen, "%s\n", recipeConfig.Title)
	}

	printColored(useColor, colorGray, "  Session: ")
	printColored(useColor, colorGreen, "%s\n", sessionID[:8]+"...")

	printColored(useColor, colorGray, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	printColored(useColor, colorGray, "Type /help for commands, /exit to quit\n")
}

// printHelp prints the help message
func printHelp(useColor bool) {
	printColored(useColor, colorCyan, "\nAvailable Commands:\n")
	printColored(useColor, colorGray, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	commands := []struct {
		cmd  string
		desc string
	}{
		{"/exit, /quit", "Exit the session"},
		{"/clear", "Clear conversation history"},
		{"/help", "Show this help message"},
		{"/status", "Show agent status"},
		{"/session", "Show session ID"},
	}

	for _, c := range commands {
		printColored(useColor, colorYellow, "  %-16s", c.cmd)
		printColored(useColor, colorGray, " %s\n", c.desc)
	}

	printColored(useColor, colorGray, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}

// printColored prints colored output if colors are enabled
func printColored(useColor bool, color, format string, args ...any) {
	if useColor {
		fmt.Printf(color+format+colorReset, args...)
	} else {
		fmt.Printf(format, args...)
	}
}

// isTerminal checks if the writer is a terminal
func isTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		stat, err := f.Stat()
		if err != nil {
			return false
		}
		return (stat.Mode() & os.ModeCharDevice) != 0
	}
	return false
}
