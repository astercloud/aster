package factory

import (
	"fmt"

	"github.com/astercloud/aster/pkg/vector"
	"github.com/astercloud/aster/pkg/vector/weaviate"
)

// StoreFactory 向量存储工厂
type StoreFactory struct {
	creators map[string]StoreCreator
}

// StoreCreator 存储创建器函数
type StoreCreator func(config map[string]any) (vector.VectorStore, error)

// NewStoreFactory 创建存储工厂
func NewStoreFactory() *StoreFactory {
	factory := &StoreFactory{
		creators: make(map[string]StoreCreator),
	}

	// 注册内置存储类型
	factory.Register("memory", createMemoryStore)
	factory.Register("weaviate", createWeaviateStore)

	return factory
}

// Register 注册新的存储类型
func (f *StoreFactory) Register(storeType string, creator StoreCreator) {
	f.creators[storeType] = creator
}

// Create 创建向量存储实例
func (f *StoreFactory) Create(storeType string, config map[string]any) (vector.VectorStore, error) {
	creator, ok := f.creators[storeType]
	if !ok {
		return nil, fmt.Errorf("unknown vector store type: %s", storeType)
	}

	return creator(config)
}

// createMemoryStore 创建内存存储
func createMemoryStore(config map[string]any) (vector.VectorStore, error) {
	return vector.NewMemoryStore(), nil
}

// createWeaviateStore 创建 Weaviate 存储
func createWeaviateStore(config map[string]any) (vector.VectorStore, error) {
	cfg := &weaviate.Config{}

	// 从 map 提取配置
	if host, ok := config["host"].(string); ok {
		cfg.Host = host
	}
	if scheme, ok := config["scheme"].(string); ok {
		cfg.Scheme = scheme
	}
	if className, ok := config["class_name"].(string); ok {
		cfg.ClassName = className
	}
	if apiKey, ok := config["api_key"].(string); ok {
		cfg.APIKey = apiKey
	}
	if namespace, ok := config["namespace"].(string); ok {
		cfg.Namespace = namespace
	}

	return weaviate.NewStore(cfg)
}

// EmbedderFactory 嵌入模型工厂
type EmbedderFactory struct {
	creators map[string]EmbedderCreator
}

// EmbedderCreator 嵌入器创建器函数
type EmbedderCreator func(config map[string]any) (vector.Embedder, error)

// NewEmbedderFactory 创建嵌入器工厂
func NewEmbedderFactory() *EmbedderFactory {
	factory := &EmbedderFactory{
		creators: make(map[string]EmbedderCreator),
	}

	// 注册内置嵌入器
	factory.Register("openai", createOpenAIEmbedder)

	return factory
}

// Register 注册新的嵌入器类型
func (f *EmbedderFactory) Register(provider string, creator EmbedderCreator) {
	f.creators[provider] = creator
}

// Create 创建嵌入器实例
func (f *EmbedderFactory) Create(provider string, config map[string]any) (vector.Embedder, error) {
	creator, ok := f.creators[provider]
	if !ok {
		return nil, fmt.Errorf("unknown embedder provider: %s", provider)
	}

	return creator(config)
}

// createOpenAIEmbedder 创建 OpenAI 嵌入器
func createOpenAIEmbedder(config map[string]any) (vector.Embedder, error) {
	var baseURL, apiKey, model string

	if v, ok := config["base_url"].(string); ok {
		baseURL = v
	}
	if v, ok := config["api_key"].(string); ok {
		apiKey = v
	}
	if v, ok := config["model"].(string); ok {
		model = v
	}

	if apiKey == "" {
		return nil, fmt.Errorf("api_key is required for OpenAI embedder")
	}

	return vector.NewOpenAIEmbedder(baseURL, apiKey, model), nil
}
