package appconfig

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Template 定义 CLI/Server 级的 Agent 模板配置, 用于初始化 TemplateRegistry。
type Template struct {
	ID           string   `yaml:"id"`
	Model        string   `yaml:"model"`
	SystemPrompt string   `yaml:"system_prompt"`
	Tools        []string `yaml:"tools"`
}

// RoutingProfile 定义一个路由 profile 对应的模型信息。
// API Key 仍通过环境变量提供, 这里只指定 env 名称。
type RoutingProfile struct {
	Provider  string `yaml:"provider"`
	Model     string `yaml:"model"`
	EnvAPIKey string `yaml:"env_api_key,omitempty"`
}

// Routing 定义 routing_profile -> 模型配置的映射。
type Routing struct {
	Profiles map[string]RoutingProfile `yaml:"profiles"`
}

// VectorStoreConfig 定义一个向量存储配置。
type VectorStoreConfig struct {
	Name string `yaml:"name"`
	Kind string `yaml:"kind"` // "memory", "pgvector" 等
	// 预留扩展字段, 不在当前版本中使用。
	DSN       string `yaml:"dsn,omitempty"`       // 用于 pgvector
	Table     string `yaml:"table,omitempty"`     // 用于 pgvector
	Metric    string `yaml:"metric,omitempty"`    // "cosine" or "l2"
	Dimension int    `yaml:"dimension,omitempty"` // 向量维度
}

// EmbedderConfig 定义一个 embedder 配置。
type EmbedderConfig struct {
	Name      string `yaml:"name"`
	Kind      string `yaml:"kind"` // "mock", 将来可扩展 "openai" 等
	Model     string `yaml:"model,omitempty"`
	EnvAPIKey string `yaml:"env_api_key,omitempty"`
}

// SemanticMemoryConfig 定义应用级语义记忆配置。
type SemanticMemoryConfig struct {
	Enabled        bool   `yaml:"enabled"`
	Store          string `yaml:"store"`                     // 对应 VectorStoreConfig.Name
	Embedder       string `yaml:"embedder"`                  // 对应 EmbedderConfig.Name
	TopK           int    `yaml:"top_k,omitempty"`           // 默认 5
	NamespaceScope string `yaml:"namespace_scope,omitempty"` // "user" | "project" | "resource" | "global"
}

// TextMemoryConfig 文本记忆配置
type TextMemoryConfig struct {
	Enabled       bool   `yaml:"enabled"`
	Path          string `yaml:"path"`           // 记忆文件根路径，默认 "/memories/"
	BaseNamespace string `yaml:"base_namespace"` // 可选的基础命名空间，用于多租户隔离
}

// WorkingMemoryConfig Working Memory 配置
type WorkingMemoryConfig struct {
	Enabled  bool           `yaml:"enabled"`
	Scope    string         `yaml:"scope"`     // "thread" | "resource"
	BasePath string         `yaml:"base_path"` // 存储根路径，默认 "/working_memory/"
	TTL      int            `yaml:"ttl"`       // 过期时间（秒），0表示不过期
	Schema   map[string]any `yaml:"schema"`    // 可选的 JSON Schema
	Template string         `yaml:"template"`  // 可选的 Markdown 模板
}

// MemoryConfig Memory 总配置
type MemoryConfig struct {
	Text          *TextMemoryConfig     `yaml:"text,omitempty"`
	WorkingMemory *WorkingMemoryConfig  `yaml:"working_memory,omitempty"`
	Semantic      *SemanticMemoryConfig `yaml:"semantic,omitempty"`
}

// Config 顶层应用配置。
type Config struct {
	Templates      []Template            `yaml:"templates"`
	Routing        *Routing              `yaml:"routing,omitempty"`
	VectorStores   []VectorStoreConfig   `yaml:"vector_stores,omitempty"`
	Embedders      []EmbedderConfig      `yaml:"embedders,omitempty"`
	SemanticMemory *SemanticMemoryConfig `yaml:"semantic_memory,omitempty"` // 向后兼容，已废弃
	Memory         *MemoryConfig         `yaml:"memory,omitempty"`          // 新的统一 Memory 配置
}

// Load 从指定路径加载 YAML 配置。
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config yaml: %w", err)
	}
	return &cfg, nil
}
