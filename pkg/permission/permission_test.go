package permission

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/astercloud/aster/pkg/types"
)

func TestNewInspector(t *testing.T) {
	inspector := NewInspector(ModeSmartApprove, WithAutoLoad(false))

	if inspector.GetMode() != ModeSmartApprove {
		t.Errorf("expected mode %s, got %s", ModeSmartApprove, inspector.GetMode())
	}
}

func TestSetMode(t *testing.T) {
	inspector := NewInspector(ModeSmartApprove, WithAutoLoad(false))

	inspector.SetMode(ModeAutoApprove)
	if inspector.GetMode() != ModeAutoApprove {
		t.Errorf("expected mode %s, got %s", ModeAutoApprove, inspector.GetMode())
	}

	inspector.SetMode(ModeAlwaysAsk)
	if inspector.GetMode() != ModeAlwaysAsk {
		t.Errorf("expected mode %s, got %s", ModeAlwaysAsk, inspector.GetMode())
	}
}

func TestGetToolRisk(t *testing.T) {
	inspector := NewInspector(ModeSmartApprove, WithAutoLoad(false))

	// Test default risk levels
	tests := []struct {
		tool     string
		expected RiskLevel
	}{
		{"read_file", RiskLevelLow},
		{"list_dir", RiskLevelLow},
		{"grep_search", RiskLevelLow},
		{"write_file", RiskLevelMedium},
		{"create_file", RiskLevelMedium},
		{"bash", RiskLevelHigh},
		{"execute", RiskLevelHigh},
		{"unknown_tool", RiskLevelMedium}, // Default for unknown
	}

	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			if got := inspector.GetToolRisk(tt.tool); got != tt.expected {
				t.Errorf("GetToolRisk(%s) = %s, want %s", tt.tool, got, tt.expected)
			}
		})
	}
}

func TestSetToolRisk(t *testing.T) {
	inspector := NewInspector(ModeSmartApprove, WithAutoLoad(false))

	// Override default risk
	inspector.SetToolRisk("read_file", RiskLevelHigh)
	if got := inspector.GetToolRisk("read_file"); got != RiskLevelHigh {
		t.Errorf("expected %s, got %s", RiskLevelHigh, got)
	}

	// Set custom tool risk
	inspector.SetToolRisk("my_custom_tool", RiskLevelLow)
	if got := inspector.GetToolRisk("my_custom_tool"); got != RiskLevelLow {
		t.Errorf("expected %s, got %s", RiskLevelLow, got)
	}
}

func TestCheckAutoApprove(t *testing.T) {
	inspector := NewInspector(ModeAutoApprove, WithAutoLoad(false))
	ctx := context.Background()

	call := &types.ToolCallSnapshot{
		ID:        "test-1",
		Name:      "bash",
		Arguments: map[string]any{"command": "rm -rf /"},
	}

	event, err := inspector.Check(ctx, call)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if event != nil {
		t.Error("expected nil event for auto_approve mode")
	}
}

func TestCheckAlwaysAsk(t *testing.T) {
	inspector := NewInspector(ModeAlwaysAsk, WithAutoLoad(false))
	ctx := context.Background()

	call := &types.ToolCallSnapshot{
		ID:        "test-1",
		Name:      "read_file",
		Arguments: map[string]any{"path": "/tmp/test.txt"},
	}

	event, err := inspector.Check(ctx, call)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if event == nil {
		t.Error("expected approval event for always_ask mode")
	}
}

func TestCheckSmartApprove(t *testing.T) {
	tmpDir := t.TempDir()
	inspector := NewInspector(ModeSmartApprove, WithAutoLoad(false), WithPersistPath(filepath.Join(tmpDir, "permissions.json")))
	ctx := context.Background()

	tests := []struct {
		name           string
		call           *types.ToolCallSnapshot
		expectApproval bool
	}{
		{
			name: "low risk auto-approved",
			call: &types.ToolCallSnapshot{
				ID:        "test-1",
				Name:      "read_file",
				Arguments: map[string]any{"path": "/tmp/test.txt"},
			},
			expectApproval: false,
		},
		{
			name: "high risk requires approval",
			call: &types.ToolCallSnapshot{
				ID:        "test-2",
				Name:      "bash",
				Arguments: map[string]any{"command": "ls"},
			},
			expectApproval: true,
		},
		{
			name: "medium risk with safe path auto-approved",
			call: &types.ToolCallSnapshot{
				ID:        "test-3",
				Name:      "write_file",
				Arguments: map[string]any{"path": "relative/path.txt", "content": "test"},
			},
			expectApproval: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := inspector.Check(ctx, tt.call)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.expectApproval && event == nil {
				t.Error("expected approval event")
			}
			if !tt.expectApproval && event != nil {
				t.Error("expected no approval event")
			}
		})
	}
}

func TestAddAndRemoveRule(t *testing.T) {
	tmpDir := t.TempDir()
	inspector := NewInspector(ModeSmartApprove, WithAutoLoad(false), WithPersistPath(filepath.Join(tmpDir, "permissions.json")))

	// Add a rule
	rule := Rule{
		Pattern:   "my_tool",
		Decision:  DecisionAllow,
		RiskLevel: RiskLevelHigh,
		Note:      "Test rule",
	}
	inspector.AddRule(rule)

	rules := inspector.GetRules()
	if len(rules) == 0 {
		t.Fatal("expected at least one rule")
	}

	found := false
	for _, r := range rules {
		if r.Pattern == "my_tool" {
			found = true
			break
		}
	}
	if !found {
		t.Error("rule not found")
	}

	// Remove the rule
	if !inspector.RemoveRule("my_tool") {
		t.Error("expected successful removal")
	}

	rules = inspector.GetRules()
	for _, r := range rules {
		if r.Pattern == "my_tool" {
			t.Error("rule should have been removed")
		}
	}
}

func TestRuleMatching(t *testing.T) {
	tmpDir := t.TempDir()
	inspector := NewInspector(ModeSmartApprove, WithAutoLoad(false), WithPersistPath(filepath.Join(tmpDir, "permissions.json")))
	ctx := context.Background()

	// Add a rule to allow bash
	inspector.AddRule(Rule{
		Pattern:   "bash",
		Decision:  DecisionAllow,
		RiskLevel: RiskLevelHigh,
		Note:      "Allow bash for testing",
	})

	call := &types.ToolCallSnapshot{
		ID:        "test-1",
		Name:      "bash",
		Arguments: map[string]any{"command": "ls"},
	}

	event, err := inspector.Check(ctx, call)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if event != nil {
		t.Error("bash should be auto-approved due to rule")
	}
}

func TestWildcardPattern(t *testing.T) {
	inspector := NewInspector(ModeSmartApprove)

	tests := []struct {
		pattern  string
		toolName string
		expected bool
	}{
		{"*", "any_tool", true},
		{"read_*", "read_file", true},
		{"read_*", "write_file", false},
		{"*_file", "read_file", true},
		{"*_file", "read_dir", false},
		{"exact_match", "exact_match", true},
		{"exact_match", "exact_match_other", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.toolName, func(t *testing.T) {
			if got := inspector.matchPattern(tt.pattern, tt.toolName); got != tt.expected {
				t.Errorf("matchPattern(%s, %s) = %v, want %v", tt.pattern, tt.toolName, got, tt.expected)
			}
		})
	}
}

func TestConditions(t *testing.T) {
	inspector := NewInspector(ModeSmartApprove)

	args := map[string]any{
		"path":   "/tmp/test.txt",
		"method": "GET",
		"count":  42,
	}

	tests := []struct {
		condition Condition
		expected  bool
	}{
		{Condition{Field: "path", Operator: "eq", Value: "/tmp/test.txt"}, true},
		{Condition{Field: "path", Operator: "eq", Value: "/other/path"}, false},
		{Condition{Field: "path", Operator: "ne", Value: "/other/path"}, true},
		{Condition{Field: "path", Operator: "prefix", Value: "/tmp/"}, true},
		{Condition{Field: "path", Operator: "suffix", Value: ".txt"}, true},
		{Condition{Field: "path", Operator: "contains", Value: "test"}, true},
		{Condition{Field: "method", Operator: "eq", Value: "GET"}, true},
		{Condition{Field: "missing", Operator: "eq", Value: "value"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.condition.Field+"_"+tt.condition.Operator, func(t *testing.T) {
			if got := inspector.checkCondition(tt.condition, args); got != tt.expected {
				t.Errorf("checkCondition() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRuleExpiration(t *testing.T) {
	tmpDir := t.TempDir()
	inspector := NewInspector(ModeSmartApprove)
	inspector.persistPath = filepath.Join(tmpDir, "permissions.json")
	ctx := context.Background()

	// Directly add an expired rule to the rules slice (bypass loadRules filter)
	past := time.Now().Add(-1 * time.Hour)
	inspector.rulesMutex.Lock()
	inspector.rules = append(inspector.rules, Rule{
		Pattern:   "bash",
		Decision:  DecisionAllow,
		ExpiresAt: &past,
		Note:      "Expired rule",
		CreatedAt: time.Now(),
	})
	inspector.rulesMutex.Unlock()

	call := &types.ToolCallSnapshot{
		ID:        "test-1",
		Name:      "bash",
		Arguments: map[string]any{"command": "ls"},
	}

	event, err := inspector.Check(ctx, call)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Expired rule should not match, bash should require approval
	if event == nil {
		t.Error("expired rule should not match, bash should require approval")
	}
}

func TestRecordDecision(t *testing.T) {
	// Use temp directory for persistence
	tmpDir := t.TempDir()
	inspector := NewInspector(ModeSmartApprove)
	inspector.persistPath = filepath.Join(tmpDir, "permissions.json")

	req := &Request{
		ToolName:  "my_dangerous_tool",
		Arguments: map[string]any{"arg": "value"},
		RiskLevel: RiskLevelHigh,
		CallID:    "call-1",
	}

	// Record "allow always" decision
	resp := inspector.RecordDecision(req, DecisionAllowAlways, "User approved for future use")

	if resp.Decision != DecisionAllowAlways {
		t.Errorf("expected %s, got %s", DecisionAllowAlways, resp.Decision)
	}

	// Check that a rule was created
	rules := inspector.GetRules()
	found := false
	for _, r := range rules {
		if r.Pattern == "my_dangerous_tool" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected rule to be created for allow_always decision")
	}
}

func TestRulePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	persistPath := filepath.Join(tmpDir, "permissions.json")

	// Create inspector and add rule
	inspector1 := NewInspector(ModeSmartApprove)
	inspector1.persistPath = persistPath

	inspector1.AddRule(Rule{
		Pattern:   "test_tool",
		Decision:  DecisionAllow,
		RiskLevel: RiskLevelMedium,
		Note:      "Persistent rule",
	})

	// Create new inspector and verify rule persisted
	inspector2 := NewInspector(ModeSmartApprove)
	inspector2.persistPath = persistPath
	inspector2.loadRules()

	rules := inspector2.GetRules()
	found := false
	for _, r := range rules {
		if r.Pattern == "test_tool" {
			found = true
			if r.Note != "Persistent rule" {
				t.Errorf("expected note 'Persistent rule', got '%s'", r.Note)
			}
			break
		}
	}
	if !found {
		t.Error("rule should have been persisted and loaded")
	}
}

func TestDenyRule(t *testing.T) {
	tmpDir := t.TempDir()
	inspector := NewInspector(ModeSmartApprove)
	inspector.persistPath = filepath.Join(tmpDir, "permissions.json")
	ctx := context.Background()

	// Add a deny rule
	inspector.AddRule(Rule{
		Pattern:  "dangerous_tool",
		Decision: DecisionDeny,
		Note:     "Tool is blocked",
	})

	call := &types.ToolCallSnapshot{
		ID:        "test-1",
		Name:      "dangerous_tool",
		Arguments: map[string]any{},
	}

	_, err := inspector.Check(ctx, call)
	if err == nil {
		t.Error("expected error for denied tool")
	}
}

func TestDefaultInspector(t *testing.T) {
	// Test package-level convenience functions
	SetMode(ModeAutoApprove)
	if DefaultInspector.GetMode() != ModeAutoApprove {
		t.Error("SetMode didn't work on default inspector")
	}

	// Reset
	SetMode(ModeSmartApprove)
}

func TestIsSafeOperation(t *testing.T) {
	inspector := NewInspector(ModeSmartApprove)

	tests := []struct {
		name     string
		req      *Request
		expected bool
	}{
		{
			name: "relative path is safe",
			req: &Request{
				ToolName:  "write_file",
				Arguments: map[string]any{"path": "relative/path.txt"},
			},
			expected: true,
		},
		{
			name: "tmp path is safe",
			req: &Request{
				ToolName:  "write_file",
				Arguments: map[string]any{"path": "/tmp/test.txt"},
			},
			expected: true,
		},
		{
			name: "system path is not safe",
			req: &Request{
				ToolName:  "write_file",
				Arguments: map[string]any{"path": "/etc/passwd"},
			},
			expected: false,
		},
		{
			name: "GET request is safe",
			req: &Request{
				ToolName:  "http_request",
				Arguments: map[string]any{"method": "GET", "url": "https://example.com"},
			},
			expected: true,
		},
		{
			name: "POST request is not safe",
			req: &Request{
				ToolName:  "http_request",
				Arguments: map[string]any{"method": "POST", "url": "https://example.com"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := inspector.isSafeOperation(tt.req); got != tt.expected {
				t.Errorf("isSafeOperation() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// Benchmark tests
func BenchmarkCheck(b *testing.B) {
	inspector := NewInspector(ModeSmartApprove)
	ctx := context.Background()
	call := &types.ToolCallSnapshot{
		ID:        "bench-1",
		Name:      "read_file",
		Arguments: map[string]any{"path": "/tmp/test.txt"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inspector.Check(ctx, call)
	}
}

func BenchmarkCheckWithRules(b *testing.B) {
	inspector := NewInspector(ModeSmartApprove)
	ctx := context.Background()

	// Add some rules
	for i := 0; i < 100; i++ {
		inspector.AddRule(Rule{
			Pattern:   "tool_" + string(rune('a'+i%26)),
			Decision:  DecisionAllow,
			RiskLevel: RiskLevelMedium,
		})
	}

	call := &types.ToolCallSnapshot{
		ID:        "bench-1",
		Name:      "read_file",
		Arguments: map[string]any{"path": "/tmp/test.txt"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inspector.Check(ctx, call)
	}
}
