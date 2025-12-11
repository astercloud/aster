// Package permission provides a comprehensive permission system for tool execution.
// It supports multiple approval modes (auto, smart, always_ask) and integrates with
// the Control Channel for human-in-the-loop interactions.
//
// This package is inspired by Goose's permission system with smart approval mode.
package permission

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/config"
	"github.com/astercloud/aster/pkg/types"
)

// Mode defines the approval mode for tool execution
type Mode string

const (
	// ModeAutoApprove automatically approves all tool executions
	ModeAutoApprove Mode = "auto_approve"

	// ModeSmartApprove uses intelligent rules to determine approval
	// - Low-risk tools (read operations) are auto-approved
	// - Medium-risk tools (write operations) require approval
	// - High-risk tools (system commands) always require approval
	ModeSmartApprove Mode = "smart_approve"

	// ModeAlwaysAsk always prompts for user approval
	ModeAlwaysAsk Mode = "always_ask"
)

// RiskLevel defines the risk level of a tool or operation
type RiskLevel string

const (
	RiskLevelLow    RiskLevel = "low"    // Read-only operations
	RiskLevelMedium RiskLevel = "medium" // Write operations with limited scope
	RiskLevelHigh   RiskLevel = "high"   // System commands, network access, etc.
)

// Decision represents an approval decision
type Decision string

const (
	DecisionAllow       Decision = "allow"        // Allow this execution
	DecisionDeny        Decision = "deny"         // Deny this execution
	DecisionAllowAlways Decision = "allow_always" // Allow this and future similar executions
	DecisionDenyAlways  Decision = "deny_always"  // Deny this and future similar executions
)

// Rule defines a permission rule for a tool or pattern
type Rule struct {
	// Pattern is the tool name or glob pattern to match
	Pattern string `json:"pattern"`

	// Decision is the default decision for matching tools
	Decision Decision `json:"decision"`

	// RiskLevel is the assigned risk level
	RiskLevel RiskLevel `json:"risk_level,omitempty"`

	// Conditions are additional conditions for the rule
	Conditions []Condition `json:"conditions,omitempty"`

	// ExpiresAt is when this rule expires (for temporary rules)
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// CreatedAt is when this rule was created
	CreatedAt time.Time `json:"created_at"`

	// Note is an optional explanation for this rule
	Note string `json:"note,omitempty"`
}

// Condition defines an additional condition for a rule
type Condition struct {
	// Field is the parameter field to check
	Field string `json:"field"`

	// Operator is the comparison operator (eq, ne, contains, prefix, suffix, regex)
	Operator string `json:"operator"`

	// Value is the value to compare against
	Value string `json:"value"`
}

// Request represents a permission request
type Request struct {
	// ToolName is the name of the tool
	ToolName string `json:"tool_name"`

	// Arguments are the tool arguments
	Arguments map[string]any `json:"arguments"`

	// RiskLevel is the assessed risk level
	RiskLevel RiskLevel `json:"risk_level"`

	// Context provides additional context
	Context map[string]any `json:"context,omitempty"`

	// CallID is the unique identifier for this tool call
	CallID string `json:"call_id"`
}

// Response represents a permission response
type Response struct {
	// Request is the original request
	Request *Request `json:"request"`

	// Decision is the approval decision
	Decision Decision `json:"decision"`

	// DecidedBy indicates who made the decision (system, user, rule)
	DecidedBy string `json:"decided_by"`

	// Note is an optional explanation
	Note string `json:"note,omitempty"`

	// DecidedAt is when the decision was made
	DecidedAt time.Time `json:"decided_at"`
}

// Inspector provides permission inspection and approval
type Inspector struct {
	mode         Mode
	rules        []Rule
	rulesMutex   sync.RWMutex
	toolRisks    map[string]RiskLevel
	persistPath  string
	defaultRisks map[string]RiskLevel
	autoLoad     bool // Whether to auto-load rules from disk
}

// InspectorOption configures an Inspector
type InspectorOption func(*Inspector)

// WithPersistPath sets the path for rule persistence
func WithPersistPath(path string) InspectorOption {
	return func(i *Inspector) {
		i.persistPath = path
	}
}

// WithAutoLoad enables/disables auto-loading rules from disk
func WithAutoLoad(autoLoad bool) InspectorOption {
	return func(i *Inspector) {
		i.autoLoad = autoLoad
	}
}

// NewInspector creates a new permission inspector
func NewInspector(mode Mode, opts ...InspectorOption) *Inspector {
	i := &Inspector{
		mode:        mode,
		rules:       make([]Rule, 0),
		toolRisks:   make(map[string]RiskLevel),
		persistPath: filepath.Join(config.DataDir(), "permissions.json"),
		autoLoad:    true, // Default: auto-load rules
		defaultRisks: map[string]RiskLevel{
			// Low risk - read operations
			"read_file":       RiskLevelLow,
			"list_dir":        RiskLevelLow,
			"file_search":     RiskLevelLow,
			"grep_search":     RiskLevelLow,
			"web_search":      RiskLevelLow,
			"get_file_info":   RiskLevelLow,
			"semantic_search": RiskLevelLow,

			// Medium risk - write operations
			"write_file":       RiskLevelMedium,
			"create_file":      RiskLevelMedium,
			"edit_file":        RiskLevelMedium,
			"delete_file":      RiskLevelMedium,
			"rename_file":      RiskLevelMedium,
			"move_file":        RiskLevelMedium,
			"create_directory": RiskLevelMedium,
			"http_request":     RiskLevelMedium,

			// High risk - system operations
			"bash":             RiskLevelHigh,
			"execute":          RiskLevelHigh,
			"run_command":      RiskLevelHigh,
			"shell":            RiskLevelHigh,
			"exec":             RiskLevelHigh,
			"subprocess":       RiskLevelHigh,
			"process_spawn":    RiskLevelHigh,
			"system":           RiskLevelHigh,
			"network_request":  RiskLevelHigh,
			"database_execute": RiskLevelHigh,
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(i)
	}

	// Try to load persisted rules if auto-load is enabled
	if i.autoLoad {
		i.loadRules()
	}

	return i
}

// SetMode sets the approval mode
func (i *Inspector) SetMode(mode Mode) {
	i.mode = mode
}

// GetMode returns the current approval mode
func (i *Inspector) GetMode() Mode {
	return i.mode
}

// SetToolRisk sets the risk level for a specific tool
func (i *Inspector) SetToolRisk(toolName string, level RiskLevel) {
	i.rulesMutex.Lock()
	defer i.rulesMutex.Unlock()
	i.toolRisks[toolName] = level
}

// GetToolRisk returns the risk level for a tool
func (i *Inspector) GetToolRisk(toolName string) RiskLevel {
	i.rulesMutex.RLock()
	defer i.rulesMutex.RUnlock()

	// Check custom risk level
	if level, ok := i.toolRisks[toolName]; ok {
		return level
	}

	// Check default risk level
	if level, ok := i.defaultRisks[toolName]; ok {
		return level
	}

	// Default to medium risk for unknown tools
	return RiskLevelMedium
}

// AddRule adds a permission rule
func (i *Inspector) AddRule(rule Rule) {
	i.rulesMutex.Lock()
	defer i.rulesMutex.Unlock()

	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = time.Now()
	}

	i.rules = append(i.rules, rule)
	i.saveRules()
}

// RemoveRule removes a rule by pattern
func (i *Inspector) RemoveRule(pattern string) bool {
	i.rulesMutex.Lock()
	defer i.rulesMutex.Unlock()

	for idx, rule := range i.rules {
		if rule.Pattern == pattern {
			i.rules = append(i.rules[:idx], i.rules[idx+1:]...)
			i.saveRules()
			return true
		}
	}
	return false
}

// GetRules returns all rules
func (i *Inspector) GetRules() []Rule {
	i.rulesMutex.RLock()
	defer i.rulesMutex.RUnlock()

	rules := make([]Rule, len(i.rules))
	copy(rules, i.rules)
	return rules
}

// Check evaluates whether a tool call should be allowed
// Returns nil if auto-approved, or a ControlPermissionRequiredEvent if approval needed
func (i *Inspector) Check(ctx context.Context, call *types.ToolCallSnapshot) (*types.ControlPermissionRequiredEvent, error) {
	req := &Request{
		ToolName:  call.Name,
		Arguments: call.Arguments,
		RiskLevel: i.GetToolRisk(call.Name),
		CallID:    call.ID,
	}

	// Check mode
	switch i.mode {
	case ModeAutoApprove:
		// Auto-approve everything
		return nil, nil

	case ModeAlwaysAsk:
		// Always require approval
		return i.createApprovalEvent(req), nil

	case ModeSmartApprove:
		// Use smart rules
		return i.smartCheck(ctx, req)

	default:
		return i.createApprovalEvent(req), nil
	}
}

// smartCheck applies intelligent rules to determine approval
func (i *Inspector) smartCheck(ctx context.Context, req *Request) (*types.ControlPermissionRequiredEvent, error) {
	// First check explicit rules
	if rule := i.findMatchingRule(req); rule != nil {
		if rule.Decision == DecisionAllow || rule.Decision == DecisionAllowAlways {
			return nil, nil
		}
		if rule.Decision == DecisionDeny || rule.Decision == DecisionDenyAlways {
			return nil, fmt.Errorf("tool %s denied by rule: %s", req.ToolName, rule.Note)
		}
	}

	// Then check risk level
	switch req.RiskLevel {
	case RiskLevelLow:
		// Low risk tools are auto-approved
		return nil, nil

	case RiskLevelMedium:
		// Medium risk tools - check for safe patterns
		if i.isSafeOperation(req) {
			return nil, nil
		}
		return i.createApprovalEvent(req), nil

	case RiskLevelHigh:
		// High risk tools always require approval
		return i.createApprovalEvent(req), nil

	default:
		return i.createApprovalEvent(req), nil
	}
}

// findMatchingRule finds a rule that matches the request
func (i *Inspector) findMatchingRule(req *Request) *Rule {
	i.rulesMutex.RLock()
	defer i.rulesMutex.RUnlock()

	now := time.Now()

	for _, rule := range i.rules {
		// Check expiration
		if rule.ExpiresAt != nil && rule.ExpiresAt.Before(now) {
			continue
		}

		// Check pattern match
		if !i.matchPattern(rule.Pattern, req.ToolName) {
			continue
		}

		// Check conditions
		if !i.checkConditions(rule.Conditions, req.Arguments) {
			continue
		}

		return &rule
	}

	return nil
}

// matchPattern matches a tool name against a pattern (supports * wildcard)
func (i *Inspector) matchPattern(pattern, toolName string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == toolName {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(toolName, prefix)
	}
	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(toolName, suffix)
	}
	return false
}

// checkConditions checks if all conditions are met
func (i *Inspector) checkConditions(conditions []Condition, args map[string]any) bool {
	for _, cond := range conditions {
		if !i.checkCondition(cond, args) {
			return false
		}
	}
	return true
}

// checkCondition checks a single condition
func (i *Inspector) checkCondition(cond Condition, args map[string]any) bool {
	value, ok := args[cond.Field]
	if !ok {
		return false
	}

	strValue := fmt.Sprintf("%v", value)

	switch cond.Operator {
	case "eq":
		return strValue == cond.Value
	case "ne":
		return strValue != cond.Value
	case "contains":
		return strings.Contains(strValue, cond.Value)
	case "prefix":
		return strings.HasPrefix(strValue, cond.Value)
	case "suffix":
		return strings.HasSuffix(strValue, cond.Value)
	default:
		return false
	}
}

// isSafeOperation checks if a medium-risk operation is safe
func (i *Inspector) isSafeOperation(req *Request) bool {
	// Check for safe file operations (within project directory)
	if path, ok := req.Arguments["path"].(string); ok {
		// Relative paths are generally safe
		if !filepath.IsAbs(path) {
			return true
		}
		// Paths in common project directories are safe
		safePaths := []string{"/tmp/", os.TempDir()}
		for _, safe := range safePaths {
			if strings.HasPrefix(path, safe) {
				return true
			}
		}
	}

	// Check for safe HTTP operations (GET requests)
	if method, ok := req.Arguments["method"].(string); ok {
		if strings.ToUpper(method) == "GET" {
			return true
		}
	}

	return false
}

// createApprovalEvent creates a permission required event
func (i *Inspector) createApprovalEvent(req *Request) *types.ControlPermissionRequiredEvent {
	return &types.ControlPermissionRequiredEvent{
		Call: types.ToolCallSnapshot{
			ID:        req.CallID,
			Name:      req.ToolName,
			Arguments: req.Arguments,
		},
	}
}

// RecordDecision records a user's decision for future reference
func (i *Inspector) RecordDecision(req *Request, decision Decision, note string) *Response {
	resp := &Response{
		Request:   req,
		Decision:  decision,
		DecidedBy: "user",
		Note:      note,
		DecidedAt: time.Now(),
	}

	// If "always" decision, create a rule
	switch decision {
	case DecisionAllowAlways:
		i.AddRule(Rule{
			Pattern:   req.ToolName,
			Decision:  DecisionAllow,
			RiskLevel: req.RiskLevel,
			CreatedAt: time.Now(),
			Note:      fmt.Sprintf("Auto-created from allow_always decision: %s", note),
		})
	case DecisionDenyAlways:
		i.AddRule(Rule{
			Pattern:   req.ToolName,
			Decision:  DecisionDeny,
			RiskLevel: req.RiskLevel,
			CreatedAt: time.Now(),
			Note:      fmt.Sprintf("Auto-created from deny_always decision: %s", note),
		})
	}

	return resp
}

// loadRules loads rules from disk
func (i *Inspector) loadRules() {
	data, err := os.ReadFile(i.persistPath)
	if err != nil {
		return // File doesn't exist or can't be read
	}

	var rules []Rule
	if err := json.Unmarshal(data, &rules); err != nil {
		return
	}

	// Filter expired rules
	now := time.Now()
	validRules := make([]Rule, 0, len(rules))
	for _, rule := range rules {
		if rule.ExpiresAt == nil || rule.ExpiresAt.After(now) {
			validRules = append(validRules, rule)
		}
	}

	i.rules = validRules
}

// saveRules saves rules to disk
func (i *Inspector) saveRules() {
	// Ensure directory exists
	dir := filepath.Dir(i.persistPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}

	data, err := json.MarshalIndent(i.rules, "", "  ")
	if err != nil {
		return
	}

	_ = os.WriteFile(i.persistPath, data, 0644) // Best effort persistence
}

// DefaultInspector is a global default inspector
var DefaultInspector = NewInspector(ModeSmartApprove)

// Check is a convenience function using the default inspector
func Check(ctx context.Context, call *types.ToolCallSnapshot) (*types.ControlPermissionRequiredEvent, error) {
	return DefaultInspector.Check(ctx, call)
}

// SetMode sets the mode on the default inspector
func SetMode(mode Mode) {
	DefaultInspector.SetMode(mode)
}

// AddRule adds a rule to the default inspector
func AddRule(rule Rule) {
	DefaultInspector.AddRule(rule)
}
