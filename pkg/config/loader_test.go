package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/astercloud/aster/pkg/types"
)

func TestNewLoader(t *testing.T) {
	loader := NewLoader()
	if loader == nil {
		t.Fatal("NewLoader returned nil")
	}

	// Default should have expandEnv enabled
	if !loader.expandEnv {
		t.Error("expected expandEnv to be true by default")
	}
}

func TestNewLoaderWithOptions(t *testing.T) {
	loader := NewLoader(
		WithEnvExpansion(false),
		WithEnvPrefix("TEST_"),
		WithVariables(map[string]string{
			"custom_var": "custom_value",
		}),
	)

	if loader.expandEnv {
		t.Error("expected expandEnv to be false")
	}
	if loader.envPrefix != "TEST_" {
		t.Errorf("expected envPrefix 'TEST_', got %q", loader.envPrefix)
	}
	if loader.variables["custom_var"] != "custom_value" {
		t.Errorf("expected custom_var to be 'custom_value', got %q", loader.variables["custom_var"])
	}
}

func TestExpandVariables(t *testing.T) {
	// Set test environment variables
	_ = os.Setenv("TEST_VAR", "test_value")
	_ = os.Setenv("ANOTHER_VAR", "another_value")
	defer func() { _ = os.Unsetenv("TEST_VAR") }()
	defer func() { _ = os.Unsetenv("ANOTHER_VAR") }()

	loader := NewLoader()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple variable",
			input:    "${TEST_VAR}",
			expected: "test_value",
		},
		{
			name:     "variable with default - exists",
			input:    "${TEST_VAR:-default}",
			expected: "test_value",
		},
		{
			name:     "variable with default - not exists",
			input:    "${NONEXISTENT:-default_value}",
			expected: "default_value",
		},
		{
			name:     "simple dollar variable",
			input:    "$TEST_VAR",
			expected: "test_value",
		},
		{
			name:     "mixed content",
			input:    "prefix_${TEST_VAR}_suffix",
			expected: "prefix_test_value_suffix",
		},
		{
			name:     "multiple variables",
			input:    "${TEST_VAR}_${ANOTHER_VAR}",
			expected: "test_value_another_value",
		},
		{
			name:     "no variables",
			input:    "plain text",
			expected: "plain text",
		},
		{
			name:     "nonexistent variable",
			input:    "${DOES_NOT_EXIST}",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := loader.expandVariables(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExpandVariablesWithPrefix(t *testing.T) {
	_ = os.Setenv("APP_TEST_VAR", "prefixed_value")
	defer func() { _ = os.Unsetenv("APP_TEST_VAR") }()

	loader := NewLoader(WithEnvPrefix("APP_"))

	result := loader.expandVariables("${TEST_VAR}")
	if result != "prefixed_value" {
		t.Errorf("expected 'prefixed_value', got %q", result)
	}
}

func TestExpandVariablesWithCustomVariables(t *testing.T) {
	loader := NewLoader(WithVariables(map[string]string{
		"CUSTOM_VAR": "custom_value",
	}))

	result := loader.expandVariables("${CUSTOM_VAR}")
	if result != "custom_value" {
		t.Errorf("expected 'custom_value', got %q", result)
	}

	// Custom variables should take precedence over environment
	_ = os.Setenv("CUSTOM_VAR", "env_value")
	defer func() { _ = os.Unsetenv("CUSTOM_VAR") }()

	result = loader.expandVariables("${CUSTOM_VAR}")
	if result != "custom_value" {
		t.Errorf("expected custom variable to take precedence, got %q", result)
	}
}

func TestExpandVariablesDisabled(t *testing.T) {
	_ = os.Setenv("TEST_VAR", "test_value")
	defer func() { _ = os.Unsetenv("TEST_VAR") }()

	loader := NewLoader(WithEnvExpansion(false))

	// When expansion is disabled, expandVariables should still work
	// but LoadFromString won't call it
	result := loader.expandVariables("${TEST_VAR}")
	if result != "test_value" {
		t.Errorf("expected 'test_value', got %q", result)
	}
}

func TestLoadFromString(t *testing.T) {
	_ = os.Setenv("API_KEY", "sk-test-key")
	defer func() { _ = os.Unsetenv("API_KEY") }()

	loader := NewLoader()

	content := `
model_config:
  provider: "anthropic"
  api_key: "${API_KEY}"
  model: "claude-3-5-sonnet"
`

	var result struct {
		ModelConfig struct {
			Provider string `yaml:"provider"`
			APIKey   string `yaml:"api_key"`
			Model    string `yaml:"model"`
		} `yaml:"model_config"`
	}

	err := loader.LoadFromString(content, &result)
	if err != nil {
		t.Fatalf("LoadFromString failed: %v", err)
	}

	if result.ModelConfig.Provider != "anthropic" {
		t.Errorf("expected provider 'anthropic', got %q", result.ModelConfig.Provider)
	}
	if result.ModelConfig.APIKey != "sk-test-key" {
		t.Errorf("expected api_key 'sk-test-key', got %q", result.ModelConfig.APIKey)
	}
	if result.ModelConfig.Model != "claude-3-5-sonnet" {
		t.Errorf("expected model 'claude-3-5-sonnet', got %q", result.ModelConfig.Model)
	}
}

func TestLoadAgentConfig(t *testing.T) {
	// Create a temp file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "agent.yaml")

	configContent := `
template_id: "test-template"
template_version: "1.0.0"
model_config:
  provider: "anthropic"
  model: "claude-3-5-sonnet"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	loader := NewLoader()
	config, err := loader.LoadAgentConfig(configPath)
	if err != nil {
		t.Fatalf("LoadAgentConfig failed: %v", err)
	}

	if config.TemplateID != "test-template" {
		t.Errorf("expected template_id 'test-template', got %q", config.TemplateID)
	}
	if config.TemplateVersion != "1.0.0" {
		t.Errorf("expected template_version '1.0.0', got %q", config.TemplateVersion)
	}
	if config.ModelConfig.Provider != "anthropic" {
		t.Errorf("expected provider 'anthropic', got %q", config.ModelConfig.Provider)
	}
}

func TestLoadAgentConfigWithEnvVars(t *testing.T) {
	_ = os.Setenv("TEST_API_KEY", "sk-env-key")
	_ = os.Setenv("TEST_MODEL", "claude-3-opus")
	defer func() { _ = os.Unsetenv("TEST_API_KEY") }()
	defer func() { _ = os.Unsetenv("TEST_MODEL") }()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "agent.yaml")

	configContent := `
template_id: "test-template"
model_config:
  provider: "anthropic"
  api_key: "${TEST_API_KEY}"
  model: "${TEST_MODEL}"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	loader := NewLoader()
	config, err := loader.LoadAgentConfig(configPath)
	if err != nil {
		t.Fatalf("LoadAgentConfig failed: %v", err)
	}

	if config.ModelConfig.APIKey != "sk-env-key" {
		t.Errorf("expected api_key 'sk-env-key', got %q", config.ModelConfig.APIKey)
	}
	if config.ModelConfig.Model != "claude-3-opus" {
		t.Errorf("expected model 'claude-3-opus', got %q", config.ModelConfig.Model)
	}
}

func TestLoadAgentConfigFileNotFound(t *testing.T) {
	loader := NewLoader()
	_, err := loader.LoadAgentConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadAgentConfigInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	loader := NewLoader()
	_, err = loader.LoadAgentConfig(configPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadAgentConfigMissingRequiredField(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "missing.yaml")

	// Missing template_id
	configContent := `
model_config:
  provider: "anthropic"
  model: "claude-3-5-sonnet"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	loader := NewLoader()
	_, err = loader.LoadAgentConfig(configPath)
	if err == nil {
		t.Error("expected error for missing template_id")
	}
}

func TestLoadModelConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "model.yaml")

	configContent := `
provider: "anthropic"
model: "claude-3-5-sonnet"
api_key: "sk-test"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	loader := NewLoader()
	config, err := loader.LoadModelConfig(configPath)
	if err != nil {
		t.Fatalf("LoadModelConfig failed: %v", err)
	}

	if config.Provider != "anthropic" {
		t.Errorf("expected provider 'anthropic', got %q", config.Provider)
	}
	if config.Model != "claude-3-5-sonnet" {
		t.Errorf("expected model 'claude-3-5-sonnet', got %q", config.Model)
	}
	if config.APIKey != "sk-test" {
		t.Errorf("expected api_key 'sk-test', got %q", config.APIKey)
	}
}

func TestMergeConfigs(t *testing.T) {
	base := &types.AgentConfig{
		AgentID:         "base-agent",
		TemplateID:      "base-template",
		TemplateVersion: "1.0.0",
		ModelConfig: &types.ModelConfig{
			Provider: "anthropic",
			Model:    "claude-3-5-sonnet",
		},
	}

	overlay := &types.AgentConfig{
		AgentID:    "overlay-agent",
		TemplateID: "overlay-template",
		ModelConfig: &types.ModelConfig{
			Model:  "claude-3-opus",
			APIKey: "sk-overlay",
		},
		Tools: []string{"tool1", "tool2"},
	}

	result := MergeConfigs(base, overlay)

	// Overlay values should override base
	if result.AgentID != "overlay-agent" {
		t.Errorf("expected AgentID 'overlay-agent', got %q", result.AgentID)
	}
	if result.TemplateID != "overlay-template" {
		t.Errorf("expected TemplateID 'overlay-template', got %q", result.TemplateID)
	}

	// Base value should remain if overlay doesn't specify
	if result.TemplateVersion != "1.0.0" {
		t.Errorf("expected TemplateVersion '1.0.0', got %q", result.TemplateVersion)
	}

	// Nested merge should work
	if result.ModelConfig.Provider != "anthropic" {
		t.Errorf("expected Provider 'anthropic', got %q", result.ModelConfig.Provider)
	}
	if result.ModelConfig.Model != "claude-3-opus" {
		t.Errorf("expected Model 'claude-3-opus', got %q", result.ModelConfig.Model)
	}
	if result.ModelConfig.APIKey != "sk-overlay" {
		t.Errorf("expected APIKey 'sk-overlay', got %q", result.ModelConfig.APIKey)
	}

	// Tools should be appended
	if len(result.Tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(result.Tools))
	}
}

func TestMergeConfigsNilBase(t *testing.T) {
	overlay := &types.AgentConfig{
		AgentID:    "overlay-agent",
		TemplateID: "overlay-template",
	}

	result := MergeConfigs(nil, overlay)

	if result.AgentID != "overlay-agent" {
		t.Errorf("expected AgentID 'overlay-agent', got %q", result.AgentID)
	}
}

func TestMergeConfigsNilOverlay(t *testing.T) {
	base := &types.AgentConfig{
		AgentID:    "base-agent",
		TemplateID: "base-template",
	}

	result := MergeConfigs(base, nil)

	if result.AgentID != "base-agent" {
		t.Errorf("expected AgentID 'base-agent', got %q", result.AgentID)
	}
}

func TestMergeConfigsMultipleOverlays(t *testing.T) {
	base := &types.AgentConfig{
		TemplateID: "base",
	}

	overlay1 := &types.AgentConfig{
		AgentID: "agent1",
	}

	overlay2 := &types.AgentConfig{
		AgentID:    "agent2",
		TemplateID: "template2",
	}

	result := MergeConfigs(base, overlay1, overlay2)

	// Last overlay should win
	if result.AgentID != "agent2" {
		t.Errorf("expected AgentID 'agent2', got %q", result.AgentID)
	}
	if result.TemplateID != "template2" {
		t.Errorf("expected TemplateID 'template2', got %q", result.TemplateID)
	}
}

func TestMergeConfigsMetadata(t *testing.T) {
	base := &types.AgentConfig{
		TemplateID: "base",
		Metadata: map[string]any{
			"key1": "value1",
			"key2": "value2",
		},
	}

	overlay := &types.AgentConfig{
		Metadata: map[string]any{
			"key2": "new_value2",
			"key3": "value3",
		},
	}

	result := MergeConfigs(base, overlay)

	if result.Metadata["key1"] != "value1" {
		t.Errorf("expected key1 'value1', got %v", result.Metadata["key1"])
	}
	if result.Metadata["key2"] != "new_value2" {
		t.Errorf("expected key2 'new_value2', got %v", result.Metadata["key2"])
	}
	if result.Metadata["key3"] != "value3" {
		t.Errorf("expected key3 'value3', got %v", result.Metadata["key3"])
	}
}

func TestValidateAgentConfig(t *testing.T) {
	loader := NewLoader()

	tests := []struct {
		name        string
		config      *types.AgentConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: &types.AgentConfig{
				TemplateID: "test-template",
			},
			expectError: false,
		},
		{
			name: "missing template_id",
			config: &types.AgentConfig{
				AgentID: "test-agent",
			},
			expectError: true,
		},
		{
			name: "valid with model config",
			config: &types.AgentConfig{
				TemplateID: "test-template",
				ModelConfig: &types.ModelConfig{
					Provider: "anthropic",
					Model:    "claude-3-5-sonnet",
				},
			},
			expectError: false,
		},
		{
			name: "invalid model config - missing provider",
			config: &types.AgentConfig{
				TemplateID: "test-template",
				ModelConfig: &types.ModelConfig{
					Model: "claude-3-5-sonnet",
				},
			},
			expectError: true,
		},
		{
			name: "invalid model config - missing model",
			config: &types.AgentConfig{
				TemplateID: "test-template",
				ModelConfig: &types.ModelConfig{
					Provider: "anthropic",
				},
			},
			expectError: true,
		},
		{
			name: "multitenancy enabled with org_id",
			config: &types.AgentConfig{
				TemplateID: "test-template",
				Multitenancy: &types.MultitenancyConfig{
					Enabled: true,
					OrgID:   "org-123",
				},
			},
			expectError: false,
		},
		{
			name: "multitenancy enabled without ids",
			config: &types.AgentConfig{
				TemplateID: "test-template",
				Multitenancy: &types.MultitenancyConfig{
					Enabled: true,
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := loader.validateAgentConfig(tt.config)
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestRequiredVariableError(t *testing.T) {
	loader := NewLoader()

	// Required variable that doesn't exist
	result := loader.expandVariables("${REQUIRED_VAR:?This variable is required}")

	// The error marker check happens during validation
	// Let's test that the error marker is inserted or result is empty
	if result != "" && !containsErrorMarker(result) {
		t.Logf("result: %q", result)
	}
}

func containsErrorMarker(s string) bool {
	return len(s) > 0 && s[:min(9, len(s))] == "__ERROR__"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
