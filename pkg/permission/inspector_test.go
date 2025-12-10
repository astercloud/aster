package permission

import (
	"context"
	"testing"

	"github.com/astercloud/aster/pkg/types"
)

func TestEnhancedInspector_BasicCheck(t *testing.T) {
	inspector := NewEnhancedInspector(&EnhancedInspectorConfig{
		Mode: ModeSmartApprove,
	})

	ctx := context.Background()

	// Test low-risk tool (should auto-approve)
	call := &types.ToolCallSnapshot{
		ID:        "call-1",
		Name:      "Read",
		Arguments: map[string]any{"path": "test.txt"},
	}

	result, err := inspector.Check(ctx, call)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Errorf("expected Read to be allowed, got denied")
	}
	if result.DecidedBy != "low_risk" {
		t.Errorf("expected DecidedBy='low_risk', got '%s'", result.DecidedBy)
	}
}

func TestEnhancedInspector_HighRiskRequiresApproval(t *testing.T) {
	inspector := NewEnhancedInspector(&EnhancedInspectorConfig{
		Mode: ModeSmartApprove,
	})

	ctx := context.Background()

	call := &types.ToolCallSnapshot{
		ID:        "call-2",
		Name:      "Bash",
		Arguments: map[string]any{"command": "rm -rf /"},
	}

	result, err := inspector.Check(ctx, call)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Errorf("expected Bash to require approval")
	}
	if !result.NeedsApproval {
		t.Errorf("expected NeedsApproval=true")
	}
}

func TestEnhancedInspector_CanUseToolCallback(t *testing.T) {
	canUseTool := func(ctx context.Context, toolName string, input map[string]any, opts *types.CanUseToolOptions) (*types.PermissionResult, error) {
		if toolName == "Bash" {
			return &types.PermissionResult{
				Behavior: "deny",
				Message:  "Bash is blocked by custom callback",
			}, nil
		}
		return &types.PermissionResult{
			Behavior: "allow",
		}, nil
	}

	inspector := NewEnhancedInspector(&EnhancedInspectorConfig{
		Mode:       ModeSmartApprove,
		CanUseTool: canUseTool,
	})

	ctx := context.Background()

	// Test blocked tool
	call := &types.ToolCallSnapshot{
		ID:        "call-3",
		Name:      "Bash",
		Arguments: map[string]any{"command": "ls"},
	}

	result, err := inspector.Check(ctx, call)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Errorf("expected Bash to be denied by callback")
	}
	if result.Message != "Bash is blocked by custom callback" {
		t.Errorf("unexpected message: %s", result.Message)
	}

	// Test allowed tool
	call2 := &types.ToolCallSnapshot{
		ID:        "call-4",
		Name:      "Read",
		Arguments: map[string]any{"path": "test.txt"},
	}

	result2, err := inspector.Check(ctx, call2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result2.Allowed {
		t.Errorf("expected Read to be allowed by callback")
	}
}

func TestEnhancedInspector_SandboxAutoAllowBash(t *testing.T) {
	sandboxConfig := &types.SandboxConfig{
		Kind: types.SandboxKindLocal,
		Settings: &types.SandboxSettings{
			Enabled:                  true,
			AutoAllowBashIfSandboxed: true,
		},
	}

	inspector := NewEnhancedInspector(&EnhancedInspectorConfig{
		Mode:          ModeSmartApprove,
		SandboxConfig: sandboxConfig,
	})

	ctx := context.Background()

	call := &types.ToolCallSnapshot{
		ID:        "call-5",
		Name:      "Bash",
		Arguments: map[string]any{"command": "ls -la"},
	}

	result, err := inspector.Check(ctx, call)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Errorf("expected Bash to be auto-allowed when sandbox is enabled")
	}
	if result.DecidedBy != "auto_allow_bash" {
		t.Errorf("expected DecidedBy='auto_allow_bash', got '%s'", result.DecidedBy)
	}
}

func TestEnhancedInspector_ExcludedCommands(t *testing.T) {
	sandboxConfig := &types.SandboxConfig{
		Kind: types.SandboxKindLocal,
		Settings: &types.SandboxSettings{
			Enabled:          true,
			ExcludedCommands: []string{"git", "docker"},
		},
	}

	inspector := NewEnhancedInspector(&EnhancedInspectorConfig{
		Mode:          ModeSmartApprove,
		SandboxConfig: sandboxConfig,
	})

	ctx := context.Background()

	// Test excluded command
	call := &types.ToolCallSnapshot{
		ID:        "call-6",
		Name:      "Bash",
		Arguments: map[string]any{"command": "git status"},
	}

	result, err := inspector.Check(ctx, call)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Errorf("expected git to be allowed as excluded command")
	}
	if result.DecidedBy != "excluded_command" {
		t.Errorf("expected DecidedBy='excluded_command', got '%s'", result.DecidedBy)
	}
}

func TestEnhancedInspector_BypassSandboxRequest(t *testing.T) {
	sandboxConfig := &types.SandboxConfig{
		Kind: types.SandboxKindLocal,
		Settings: &types.SandboxSettings{
			Enabled:                  true,
			AllowUnsandboxedCommands: false, // Not allowed
		},
	}

	inspector := NewEnhancedInspector(&EnhancedInspectorConfig{
		Mode:          ModeSmartApprove,
		SandboxConfig: sandboxConfig,
	})

	ctx := context.Background()

	call := &types.ToolCallSnapshot{
		ID:   "call-7",
		Name: "Bash",
		Arguments: map[string]any{
			"command":                   "sudo apt update",
			"dangerouslyDisableSandbox": true,
		},
	}

	result, err := inspector.Check(ctx, call)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Errorf("expected bypass request to be denied when AllowUnsandboxedCommands=false")
	}
	if result.DecidedBy != "sandbox_policy" {
		t.Errorf("expected DecidedBy='sandbox_policy', got '%s'", result.DecidedBy)
	}
}

func TestEnhancedInspector_PermissionModeBypass(t *testing.T) {
	sandboxConfig := &types.SandboxConfig{
		Kind:           types.SandboxKindLocal,
		PermissionMode: types.SandboxPermissionBypass,
	}

	inspector := NewEnhancedInspector(&EnhancedInspectorConfig{
		Mode:          ModeAlwaysAsk, // Would normally require approval
		SandboxConfig: sandboxConfig,
	})

	ctx := context.Background()

	call := &types.ToolCallSnapshot{
		ID:        "call-8",
		Name:      "Bash",
		Arguments: map[string]any{"command": "rm -rf /"},
	}

	result, err := inspector.Check(ctx, call)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Errorf("expected bypass mode to allow all tools")
	}
	if result.DecidedBy != "bypass_mode" {
		t.Errorf("expected DecidedBy='bypass_mode', got '%s'", result.DecidedBy)
	}
}

func TestEnhancedInspector_PermissionModePlan(t *testing.T) {
	sandboxConfig := &types.SandboxConfig{
		Kind:           types.SandboxKindLocal,
		PermissionMode: types.SandboxPermissionPlan,
	}

	inspector := NewEnhancedInspector(&EnhancedInspectorConfig{
		Mode:          ModeAutoApprove,
		SandboxConfig: sandboxConfig,
	})

	ctx := context.Background()

	call := &types.ToolCallSnapshot{
		ID:        "call-9",
		Name:      "Read",
		Arguments: map[string]any{"path": "test.txt"},
	}

	result, err := inspector.Check(ctx, call)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Errorf("expected plan mode to block all tools")
	}
	if result.DecidedBy != "plan_mode" {
		t.Errorf("expected DecidedBy='plan_mode', got '%s'", result.DecidedBy)
	}
}

func TestEnhancedInspector_SessionRules(t *testing.T) {
	inspector := NewEnhancedInspector(&EnhancedInspectorConfig{
		Mode: ModeAlwaysAsk,
	})

	// Add session rule
	inspector.addSessionRule(Rule{
		Pattern:  "Read",
		Decision: DecisionAllow,
	})

	ctx := context.Background()

	call := &types.ToolCallSnapshot{
		ID:        "call-10",
		Name:      "Read",
		Arguments: map[string]any{"path": "test.txt"},
	}

	result, err := inspector.Check(ctx, call)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Errorf("expected session rule to allow Read")
	}

	// Clear session rules
	inspector.ClearSessionRules()

	result2, err := inspector.Check(ctx, call)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result2.Allowed {
		t.Errorf("expected Read to require approval after clearing session rules")
	}
}

func TestEnhancedInspector_UpdatedInput(t *testing.T) {
	canUseTool := func(ctx context.Context, toolName string, input map[string]any, opts *types.CanUseToolOptions) (*types.PermissionResult, error) {
		// Modify input
		input["timeout"] = 30000
		return &types.PermissionResult{
			Behavior:     "allow",
			UpdatedInput: input,
		}, nil
	}

	inspector := NewEnhancedInspector(&EnhancedInspectorConfig{
		Mode:       ModeSmartApprove,
		CanUseTool: canUseTool,
	})

	ctx := context.Background()

	call := &types.ToolCallSnapshot{
		ID:        "call-11",
		Name:      "Bash",
		Arguments: map[string]any{"command": "sleep 100"},
	}

	result, err := inspector.Check(ctx, call)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Errorf("expected to be allowed")
	}
	if result.UpdatedInput == nil {
		t.Fatalf("expected UpdatedInput to be set")
	}
	if result.UpdatedInput["timeout"] != 30000 {
		t.Errorf("expected timeout=30000, got %v", result.UpdatedInput["timeout"])
	}
}

func TestEnhancedInspector_ViolationRecording(t *testing.T) {
	inspector := NewEnhancedInspector(&EnhancedInspectorConfig{
		Mode: ModeSmartApprove,
	})

	// Record a violation
	inspector.RecordViolation(types.SandboxViolation{
		Type:      "file",
		Path:      "/etc/passwd",
		Operation: "read",
		Blocked:   true,
	})

	violations := inspector.GetViolations()
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Path != "/etc/passwd" {
		t.Errorf("unexpected violation path: %s", violations[0].Path)
	}
}
