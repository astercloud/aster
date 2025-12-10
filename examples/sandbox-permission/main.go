// Package main demonstrates the Claude Agent SDK style sandbox and permission system.
//
// This example shows how to:
// 1. Configure sandbox settings with network isolation
// 2. Use CanUseTool callback for custom permission control
// 3. Handle permission updates and violations
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/permission"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/builtin"
	"github.com/astercloud/aster/pkg/types"
)

var (
	apiKey    = flag.String("api-key", "", "API key for the LLM provider")
	workspace = flag.String("workspace", "./workspace", "Workspace directory")
	mode      = flag.String("mode", "smart", "Permission mode: auto, smart, always_ask, bypass")
)

func main() {
	flag.Parse()

	if *apiKey == "" {
		*apiKey = os.Getenv("ANTHROPIC_API_KEY")
		if *apiKey == "" {
			*apiKey = os.Getenv("OPENAI_API_KEY")
		}
	}

	if *apiKey == "" {
		log.Fatal("API key is required. Set via -api-key flag or ANTHROPIC_API_KEY/OPENAI_API_KEY env var")
	}

	// Ensure workspace exists
	if err := os.MkdirAll(*workspace, 0755); err != nil {
		log.Fatalf("Failed to create workspace: %v", err)
	}

	ctx := context.Background()

	// === 1. Configure Sandbox Settings (Claude Agent SDK style) ===
	sandboxSettings := &types.SandboxSettings{
		// Enable sandbox isolation
		Enabled: true,

		// Auto-approve bash commands when sandbox is enabled
		AutoAllowBashIfSandboxed: true,

		// Commands that bypass sandbox (e.g., docker, git)
		ExcludedCommands: []string{"git", "docker"},

		// Allow model to request unsandboxed execution (requires permission approval)
		AllowUnsandboxedCommands: true,

		// Network isolation settings
		Network: &types.NetworkSandboxSettings{
			AllowLocalBinding: true,                             // Allow dev servers
			AllowUnixSockets:  []string{"/var/run/docker.sock"}, // Allow Docker socket
			AllowedHosts:      []string{"api.openai.com", "api.anthropic.com"},
			BlockedHosts:      []string{"malicious.com"},
		},

		// Ignore certain violations
		IgnoreViolations: &types.SandboxIgnoreViolations{
			FilePatterns:    []string{"/tmp/*", "*.log"},
			NetworkPatterns: []string{"localhost:*"},
		},
	}

	sandboxConfig := &types.SandboxConfig{
		Kind:           types.SandboxKindLocal,
		WorkDir:        *workspace,
		Settings:       sandboxSettings,
		PermissionMode: types.SandboxPermissionDefault,
	}

	// === 2. Create Custom CanUseTool Callback ===
	canUseTool := func(
		ctx context.Context,
		toolName string,
		input map[string]any,
		opts *types.CanUseToolOptions,
	) (*types.PermissionResult, error) {
		fmt.Printf("\n🔐 Permission check for tool: %s\n", toolName)
		fmt.Printf("   Input: %v\n", input)
		fmt.Printf("   Sandbox enabled: %v\n", opts.SandboxEnabled)
		fmt.Printf("   Bypass requested: %v\n", opts.BypassSandboxRequested)

		// Example: Block certain file paths
		if path, ok := input["path"].(string); ok {
			if path == "/etc/passwd" || path == "/etc/shadow" {
				return &types.PermissionResult{
					Behavior: "deny",
					Message:  "Access to system files is not allowed",
				}, nil
			}
		}

		// Example: Modify input (e.g., add safety limits)
		if toolName == "Bash" {
			if cmd, ok := input["command"].(string); ok {
				// Add timeout to long-running commands
				if len(cmd) > 100 {
					input["timeout"] = 30000 // 30 seconds
					return &types.PermissionResult{
						Behavior:     "allow",
						UpdatedInput: input,
					}, nil
				}
			}
		}

		// Example: Auto-approve read operations
		if toolName == "Read" || toolName == "Glob" || toolName == "Grep" {
			return &types.PermissionResult{
				Behavior: "allow",
			}, nil
		}

		// Example: Add session rule for repeated operations
		if toolName == "Write" {
			return &types.PermissionResult{
				Behavior: "allow",
				UpdatedPermissions: []types.PermissionUpdate{
					{
						Type:        "addRules",
						Behavior:    "allow",
						Destination: "session",
						Rules: []types.PermissionRule{
							{ToolName: "Write", RuleContent: "auto-approved for this session"},
						},
					},
				},
			}, nil
		}

		// Default: let the permission system decide
		return nil, nil
	}

	// === 3. Create Enhanced Permission Inspector ===
	permMode := permission.ModeSmartApprove
	switch *mode {
	case "auto":
		permMode = permission.ModeAutoApprove
	case "always_ask":
		permMode = permission.ModeAlwaysAsk
	}

	inspector := permission.NewEnhancedInspector(&permission.EnhancedInspectorConfig{
		Mode:          permMode,
		SandboxConfig: sandboxConfig,
		CanUseTool:    canUseTool,
	})

	// === 4. Setup Agent Dependencies ===
	storePath := ".aster-sandbox-demo"
	jsonStore, err := store.NewJSONStore(storePath)
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}

	sandboxFactory := sandbox.NewFactory()
	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry)

	providerFactory := &provider.AnthropicFactory{}
	templateRegistry := agent.NewTemplateRegistry()

	// Register template
	templateRegistry.Register(&types.AgentTemplateDefinition{
		ID:           "sandbox-demo",
		SystemPrompt: "You are a helpful assistant with sandbox-protected tool access.",
		Tools:        "*",
	})

	deps := &agent.Dependencies{
		Store:            jsonStore,
		SandboxFactory:   sandboxFactory,
		ToolRegistry:     toolRegistry,
		ProviderFactory:  providerFactory,
		TemplateRegistry: templateRegistry,
	}

	// === 5. Create Agent with Sandbox and Permission Config ===
	agentConfig := &types.AgentConfig{
		TemplateID: "sandbox-demo",
		ModelConfig: &types.ModelConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-20250514",
			APIKey:   *apiKey,
		},
		Sandbox:    sandboxConfig,
		CanUseTool: canUseTool, // Custom permission callback
	}

	ag, err := agent.Create(ctx, agentConfig, deps)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	defer ag.Close()

	// === 6. Demo: Test Permission System ===
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🛡️  Sandbox & Permission Demo")
	fmt.Println(strings.Repeat("=", 60))

	// Test 1: Read operation (should auto-approve)
	fmt.Println("\n📖 Test 1: Read operation")
	testPermission(ctx, inspector, "Read", map[string]any{"path": "README.md"})

	// Test 2: Write operation (should add session rule)
	fmt.Println("\n✏️  Test 2: Write operation")
	testPermission(ctx, inspector, "Write", map[string]any{"path": "test.txt", "content": "hello"})

	// Test 3: Bash command (should auto-approve if sandbox enabled)
	fmt.Println("\n💻 Test 3: Bash command")
	testPermission(ctx, inspector, "Bash", map[string]any{"command": "ls -la"})

	// Test 4: Blocked file access
	fmt.Println("\n🚫 Test 4: Blocked file access")
	testPermission(ctx, inspector, "Read", map[string]any{"path": "/etc/passwd"})

	// Test 5: Excluded command (git)
	fmt.Println("\n🔓 Test 5: Excluded command (git)")
	testPermission(ctx, inspector, "Bash", map[string]any{"command": "git status"})

	// Test 6: Bypass sandbox request
	fmt.Println("\n⚠️  Test 6: Bypass sandbox request")
	testPermission(ctx, inspector, "Bash", map[string]any{
		"command":                   "sudo apt update",
		"dangerouslyDisableSandbox": true,
	})

	// Show violations
	fmt.Println("\n📋 Recorded Violations:")
	violations := inspector.GetViolations()
	if len(violations) == 0 {
		fmt.Println("   No violations recorded")
	} else {
		for _, v := range violations {
			fmt.Printf("   - %s: %s (%s)\n", v.Type, v.Path, v.Operation)
		}
	}

	fmt.Println("\n✅ Demo completed!")
}

func testPermission(ctx context.Context, inspector *permission.EnhancedInspector, toolName string, args map[string]any) {
	call := &types.ToolCallSnapshot{
		ID:        fmt.Sprintf("call-%d", time.Now().UnixNano()),
		Name:      toolName,
		Arguments: args,
	}

	result, err := inspector.Check(ctx, call)
	if err != nil {
		fmt.Printf("   ❌ Error: %v\n", err)
		return
	}

	if result.Allowed {
		fmt.Printf("   ✅ Allowed (decided by: %s)\n", result.DecidedBy)
		if result.UpdatedInput != nil {
			fmt.Printf("   📝 Input modified: %v\n", result.UpdatedInput)
		}
	} else if result.NeedsApproval {
		fmt.Printf("   ⏳ Needs approval (decided by: %s)\n", result.DecidedBy)
	} else {
		fmt.Printf("   🚫 Denied: %s (decided by: %s)\n", result.Message, result.DecidedBy)
	}
}
