package recipe

import (
	"testing"
)

func TestLoadFromBytes(t *testing.T) {
	yamlContent := `
version: "1.0"
title: "Code Review"
description: "AI-powered code review assistant"
instructions: |
  You are a senior code reviewer. Focus on:
  - Code quality
  - Security issues
  - Performance
prompt: "Review the code in {{directory}}"
tools:
  - filesystem
  - bash
permission_mode: smart_approve
parameters:
  - key: directory
    input_type: string
    requirement: optional
    description: "Directory to review"
    default: "."
  - key: language
    input_type: select
    requirement: optional
    description: "Programming language"
    options:
      - go
      - python
      - javascript
extensions:
  - type: stdio
    name: git
    cmd: npx
    args: ["-y", "@anthropic/git-mcp"]
author:
  name: "Aster Team"
  contact: "team@aster.dev"
`

	recipe, err := LoadFromBytes([]byte(yamlContent))
	if err != nil {
		t.Fatalf("LoadFromBytes failed: %v", err)
	}

	// Verify basic fields
	if recipe.Title != "Code Review" {
		t.Errorf("Expected title 'Code Review', got %q", recipe.Title)
	}

	if recipe.PermissionMode != PermissionSmartApprove {
		t.Errorf("Expected permission_mode 'smart_approve', got %q", recipe.PermissionMode)
	}

	// Verify tools
	if len(recipe.Tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(recipe.Tools))
	}

	// Verify parameters
	if len(recipe.Parameters) != 2 {
		t.Errorf("Expected 2 parameters, got %d", len(recipe.Parameters))
	}

	// Verify extensions
	if len(recipe.Extensions) != 1 {
		t.Errorf("Expected 1 extension, got %d", len(recipe.Extensions))
	}

	if recipe.Extensions[0].Type != "stdio" {
		t.Errorf("Expected extension type 'stdio', got %q", recipe.Extensions[0].Type)
	}

	// Verify author
	if recipe.Author == nil {
		t.Error("Expected author to be set")
	} else if recipe.Author.Name != "Aster Team" {
		t.Errorf("Expected author name 'Aster Team', got %q", recipe.Author.Name)
	}
}

func TestRecipeValidation(t *testing.T) {
	tests := []struct {
		name    string
		recipe  Recipe
		wantErr bool
	}{
		{
			name: "valid recipe",
			recipe: Recipe{
				Title:       "Test",
				Description: "Test recipe",
			},
			wantErr: false,
		},
		{
			name: "missing title",
			recipe: Recipe{
				Description: "Test recipe",
			},
			wantErr: true,
		},
		{
			name: "missing description",
			recipe: Recipe{
				Title: "Test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.recipe.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParameterValidation(t *testing.T) {
	tests := []struct {
		name    string
		param   Parameter
		wantErr bool
	}{
		{
			name: "valid string parameter",
			param: Parameter{
				Key:         "test",
				Type:        ParamTypeString,
				Requirement: ParamOptional,
				Description: "Test param",
			},
			wantErr: false,
		},
		{
			name: "valid select parameter",
			param: Parameter{
				Key:         "lang",
				Type:        ParamTypeSelect,
				Requirement: ParamRequired,
				Description: "Language",
				Options:     []string{"go", "python"},
			},
			wantErr: false,
		},
		{
			name: "select without options",
			param: Parameter{
				Key:         "lang",
				Type:        ParamTypeSelect,
				Requirement: ParamRequired,
				Description: "Language",
			},
			wantErr: true,
		},
		{
			name: "file with default",
			param: Parameter{
				Key:         "config",
				Type:        ParamTypeFile,
				Requirement: ParamOptional,
				Description: "Config file",
				Default:     "/etc/passwd", // security risk
			},
			wantErr: true,
		},
		{
			name: "missing key",
			param: Parameter{
				Type:        ParamTypeString,
				Requirement: ParamOptional,
				Description: "Test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.param.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExtensionValidation(t *testing.T) {
	tests := []struct {
		name    string
		ext     ExtensionConfig
		wantErr bool
	}{
		{
			name: "valid stdio extension",
			ext: ExtensionConfig{
				Type: "stdio",
				Name: "test",
				Cmd:  "test-cmd",
			},
			wantErr: false,
		},
		{
			name: "valid sse extension",
			ext: ExtensionConfig{
				Type: "sse",
				Name: "test",
				URL:  "http://localhost:8080",
			},
			wantErr: false,
		},
		{
			name: "valid builtin extension",
			ext: ExtensionConfig{
				Type: "builtin",
				Name: "filesystem",
			},
			wantErr: false,
		},
		{
			name: "stdio without cmd",
			ext: ExtensionConfig{
				Type: "stdio",
				Name: "test",
			},
			wantErr: true,
		},
		{
			name: "sse without url",
			ext: ExtensionConfig{
				Type: "sse",
				Name: "test",
			},
			wantErr: true,
		},
		{
			name: "unknown type",
			ext: ExtensionConfig{
				Type: "unknown",
				Name: "test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ext.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestApplyParameters(t *testing.T) {
	recipe := &Recipe{
		Title:        "Test",
		Description:  "Test recipe",
		Instructions: "Review code in {{directory}} for {{language}}",
		Prompt:       "Start reviewing {{directory}}",
	}

	values := map[string]string{
		"directory": "/src",
		"language":  "Go",
	}

	if err := recipe.ApplyParameters(values); err != nil {
		t.Fatalf("ApplyParameters failed: %v", err)
	}

	expectedInstructions := "Review code in /src for Go"
	if recipe.Instructions != expectedInstructions {
		t.Errorf("Expected instructions %q, got %q", expectedInstructions, recipe.Instructions)
	}

	expectedPrompt := "Start reviewing /src"
	if recipe.Prompt != expectedPrompt {
		t.Errorf("Expected prompt %q, got %q", expectedPrompt, recipe.Prompt)
	}
}

func TestBuilder(t *testing.T) {
	recipe, err := NewBuilder().
		Title("Test Recipe").
		Description("A test recipe").
		Instructions("You are a test assistant").
		Tools("filesystem", "bash").
		PermissionMode(PermissionSmartApprove).
		AddParameter(Parameter{
			Key:         "test",
			Type:        ParamTypeString,
			Requirement: ParamOptional,
			Description: "Test param",
		}).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if recipe.Title != "Test Recipe" {
		t.Errorf("Expected title 'Test Recipe', got %q", recipe.Title)
	}

	if len(recipe.Tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(recipe.Tools))
	}

	if len(recipe.Parameters) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(recipe.Parameters))
	}
}

func TestExtensionIsEnabled(t *testing.T) {
	// Default (nil) should be enabled
	ext1 := ExtensionConfig{Name: "test"}
	if !ext1.IsEnabled() {
		t.Error("Extension should be enabled by default")
	}

	// Explicitly enabled
	enabled := true
	ext2 := ExtensionConfig{Name: "test", Enabled: &enabled}
	if !ext2.IsEnabled() {
		t.Error("Extension should be enabled when set to true")
	}

	// Explicitly disabled
	disabled := false
	ext3 := ExtensionConfig{Name: "test", Enabled: &disabled}
	if ext3.IsEnabled() {
		t.Error("Extension should be disabled when set to false")
	}
}

func TestToYAML(t *testing.T) {
	recipe := &Recipe{
		Version:     "1.0",
		Title:       "Test",
		Description: "Test recipe",
		Tools:       []string{"filesystem"},
	}

	data, err := recipe.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty YAML output")
	}

	// Verify it can be parsed back
	parsed, err := LoadFromBytes(data)
	if err != nil {
		t.Fatalf("Failed to parse YAML output: %v", err)
	}

	if parsed.Title != recipe.Title {
		t.Errorf("Round-trip failed: expected title %q, got %q", recipe.Title, parsed.Title)
	}
}
