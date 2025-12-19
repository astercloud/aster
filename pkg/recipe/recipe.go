// Package recipe provides a declarative way to define and share Agent configurations.
// Inspired by Goose's Recipe system, it allows users to create reusable Agent templates
// with pre-configured tools, prompts, and behaviors.
package recipe

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Recipe defines a reusable Agent configuration.
type Recipe struct {
	// Version of the recipe format (semver)
	Version string `yaml:"version" json:"version"`

	// Title is a short name for the recipe
	Title string `yaml:"title" json:"title"`

	// Description explains what this recipe does
	Description string `yaml:"description" json:"description"`

	// TemplateID references an existing template from appconfig
	// If empty, uses the default template
	TemplateID string `yaml:"template_id,omitempty" json:"template_id,omitempty"`

	// Instructions override or extend the system prompt
	Instructions string `yaml:"instructions,omitempty" json:"instructions,omitempty"`

	// Prompt is the initial message to send to the agent
	Prompt string `yaml:"prompt,omitempty" json:"prompt,omitempty"`

	// Tools is a list of tool names to enable
	Tools []string `yaml:"tools,omitempty" json:"tools,omitempty"`

	// Extensions defines MCP extensions to load
	Extensions []ExtensionConfig `yaml:"extensions,omitempty" json:"extensions,omitempty"`

	// Parameters defines user-configurable parameters
	Parameters []Parameter `yaml:"parameters,omitempty" json:"parameters,omitempty"`

	// Settings contains runtime settings
	Settings *Settings `yaml:"settings,omitempty" json:"settings,omitempty"`

	// Author information
	Author *Author `yaml:"author,omitempty" json:"author,omitempty"`

	// Activities are suggested prompts shown in UI
	Activities []string `yaml:"activities,omitempty" json:"activities,omitempty"`

	// PermissionMode controls tool approval behavior
	PermissionMode PermissionMode `yaml:"permission_mode,omitempty" json:"permission_mode,omitempty"`
}

// ExtensionConfig defines an MCP extension.
type ExtensionConfig struct {
	// Type is the extension type: "stdio", "sse", "builtin"
	Type string `yaml:"type" json:"type"`

	// Name is the unique identifier for this extension
	Name string `yaml:"name" json:"name"`

	// Description of what this extension does
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Cmd is the command to run (for stdio type)
	Cmd string `yaml:"cmd,omitempty" json:"cmd,omitempty"`

	// Args are command arguments (for stdio type)
	Args []string `yaml:"args,omitempty" json:"args,omitempty"`

	// URL is the server URL (for sse type)
	URL string `yaml:"url,omitempty" json:"url,omitempty"`

	// Env are environment variables to set
	Env map[string]string `yaml:"env,omitempty" json:"env,omitempty"`

	// Timeout in seconds
	Timeout int `yaml:"timeout,omitempty" json:"timeout,omitempty"`

	// Enabled indicates if this extension should be loaded
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`
}

// Parameter defines a configurable parameter.
type Parameter struct {
	// Key is the parameter name
	Key string `yaml:"key" json:"key"`

	// Type is the input type: "string", "number", "boolean", "select", "file"
	Type ParameterType `yaml:"input_type" json:"input_type"`

	// Requirement specifies if the parameter is required
	Requirement ParameterRequirement `yaml:"requirement" json:"requirement"`

	// Description explains what this parameter does
	Description string `yaml:"description" json:"description"`

	// Default value (not allowed for file type)
	Default string `yaml:"default,omitempty" json:"default,omitempty"`

	// Options for select type
	Options []string `yaml:"options,omitempty" json:"options,omitempty"`
}

// ParameterType represents the input type of a parameter.
type ParameterType string

const (
	ParamTypeString  ParameterType = "string"
	ParamTypeNumber  ParameterType = "number"
	ParamTypeBoolean ParameterType = "boolean"
	ParamTypeSelect  ParameterType = "select"
	ParamTypeFile    ParameterType = "file"
	ParamTypeDate    ParameterType = "date"
)

// ParameterRequirement specifies if a parameter is required.
type ParameterRequirement string

const (
	ParamRequired   ParameterRequirement = "required"
	ParamOptional   ParameterRequirement = "optional"
	ParamUserPrompt ParameterRequirement = "user_prompt"
)

// Settings contains runtime configuration.
type Settings struct {
	// Provider name (e.g., "anthropic", "openai")
	Provider string `yaml:"provider,omitempty" json:"provider,omitempty"`

	// Model name
	Model string `yaml:"model,omitempty" json:"model,omitempty"`

	// Temperature for generation
	Temperature *float32 `yaml:"temperature,omitempty" json:"temperature,omitempty"`

	// MaxTokens limits output length
	MaxTokens *int `yaml:"max_tokens,omitempty" json:"max_tokens,omitempty"`
}

// Author contains creator information.
type Author struct {
	// Name of the author
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// Contact information (email, URL, etc.)
	Contact string `yaml:"contact,omitempty" json:"contact,omitempty"`

	// URL to author's website or profile
	URL string `yaml:"url,omitempty" json:"url,omitempty"`
}

// PermissionMode controls tool approval behavior.
type PermissionMode string

const (
	// PermissionAutoApprove automatically approves all tool calls
	PermissionAutoApprove PermissionMode = "auto_approve"

	// PermissionSmartApprove auto-approves read-only tools, asks for others
	PermissionSmartApprove PermissionMode = "smart_approve"

	// PermissionAlwaysAsk always asks for user confirmation
	PermissionAlwaysAsk PermissionMode = "always_ask"
)

// LoadFromFile loads a recipe from a YAML file.
func LoadFromFile(path string) (*Recipe, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read recipe file: %w", err)
	}

	return LoadFromBytes(data)
}

// LoadFromBytes parses a recipe from YAML bytes.
func LoadFromBytes(data []byte) (*Recipe, error) {
	var recipe Recipe
	if err := yaml.Unmarshal(data, &recipe); err != nil {
		return nil, fmt.Errorf("parse recipe: %w", err)
	}

	if err := recipe.Validate(); err != nil {
		return nil, fmt.Errorf("validate recipe: %w", err)
	}

	return &recipe, nil
}

// Validate checks if the recipe is valid.
func (r *Recipe) Validate() error {
	if r.Title == "" {
		return errors.New("title is required")
	}

	if r.Description == "" {
		return errors.New("description is required")
	}

	// At least one of instructions or prompt should be set for a useful recipe
	// but we don't enforce this as the template might provide defaults

	// Validate parameters
	for _, p := range r.Parameters {
		if err := p.Validate(); err != nil {
			return fmt.Errorf("parameter %q: %w", p.Key, err)
		}
	}

	// Validate extensions
	for _, e := range r.Extensions {
		if err := e.Validate(); err != nil {
			return fmt.Errorf("extension %q: %w", e.Name, err)
		}
	}

	return nil
}

// Validate checks if the parameter is valid.
func (p *Parameter) Validate() error {
	if p.Key == "" {
		return errors.New("key is required")
	}

	if p.Type == "" {
		return errors.New("input_type is required")
	}

	// File type cannot have default values (security)
	if p.Type == ParamTypeFile && p.Default != "" {
		return errors.New("file parameters cannot have default values")
	}

	// Select type requires options
	if p.Type == ParamTypeSelect && len(p.Options) == 0 {
		return errors.New("select parameters require options")
	}

	return nil
}

// Validate checks if the extension is valid.
func (e *ExtensionConfig) Validate() error {
	if e.Name == "" {
		return errors.New("name is required")
	}

	if e.Type == "" {
		return errors.New("type is required")
	}

	switch e.Type {
	case "stdio":
		if e.Cmd == "" {
			return errors.New("cmd is required for stdio extensions")
		}
	case "sse":
		if e.URL == "" {
			return errors.New("url is required for sse extensions")
		}
	case "builtin":
		// No additional validation needed
	default:
		return fmt.Errorf("unknown extension type: %s", e.Type)
	}

	return nil
}

// ToYAML serializes the recipe to YAML.
func (r *Recipe) ToYAML() ([]byte, error) {
	return yaml.Marshal(r)
}

// IsEnabled returns whether the extension is enabled (default true).
func (e *ExtensionConfig) IsEnabled() bool {
	if e.Enabled == nil {
		return true
	}
	return *e.Enabled
}

// ApplyParameters substitutes parameter values in the recipe.
func (r *Recipe) ApplyParameters(values map[string]string) error {
	// Substitute in instructions
	if r.Instructions != "" {
		r.Instructions = substituteParams(r.Instructions, values)
	}

	// Substitute in prompt
	if r.Prompt != "" {
		r.Prompt = substituteParams(r.Prompt, values)
	}

	return nil
}

// substituteParams replaces {{key}} with values.
func substituteParams(text string, values map[string]string) string {
	result := text
	for key, value := range values {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// ListRecipes returns all recipes in the given directory.
func ListRecipes(dir string) ([]*Recipe, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var recipes []*Recipe
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		recipe, err := LoadFromFile(filepath.Join(dir, name))
		if err != nil {
			// Log warning but continue
			continue
		}

		recipes = append(recipes, recipe)
	}

	return recipes, nil
}

// Builder provides a fluent API for creating recipes.
type Builder struct {
	recipe Recipe
}

// NewBuilder creates a new recipe builder.
func NewBuilder() *Builder {
	return &Builder{
		recipe: Recipe{
			Version: "1.0",
		},
	}
}

// Title sets the recipe title.
func (b *Builder) Title(title string) *Builder {
	b.recipe.Title = title
	return b
}

// Description sets the recipe description.
func (b *Builder) Description(desc string) *Builder {
	b.recipe.Description = desc
	return b
}

// Instructions sets the system instructions.
func (b *Builder) Instructions(instructions string) *Builder {
	b.recipe.Instructions = instructions
	return b
}

// Prompt sets the initial prompt.
func (b *Builder) Prompt(prompt string) *Builder {
	b.recipe.Prompt = prompt
	return b
}

// Tools sets the enabled tools.
func (b *Builder) Tools(tools ...string) *Builder {
	b.recipe.Tools = tools
	return b
}

// AddExtension adds an MCP extension.
func (b *Builder) AddExtension(ext ExtensionConfig) *Builder {
	b.recipe.Extensions = append(b.recipe.Extensions, ext)
	return b
}

// AddParameter adds a configurable parameter.
func (b *Builder) AddParameter(param Parameter) *Builder {
	b.recipe.Parameters = append(b.recipe.Parameters, param)
	return b
}

// PermissionMode sets the permission mode.
func (b *Builder) PermissionMode(mode PermissionMode) *Builder {
	b.recipe.PermissionMode = mode
	return b
}

// Build creates the recipe.
func (b *Builder) Build() (*Recipe, error) {
	if err := b.recipe.Validate(); err != nil {
		return nil, err
	}
	return &b.recipe, nil
}
