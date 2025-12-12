package agent

import (
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/router"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/types"
	"github.com/astercloud/aster/pkg/vector/factory"
)

// Dependencies Agent依赖
type Dependencies struct {
	Store           store.Store
	SandboxFactory  *sandbox.Factory
	ToolRegistry    *tools.Registry
	ProviderFactory provider.Factory
	// Router 为可选依赖，如果为 nil，则沿用旧的静态 ModelConfig 行为。
	Router           router.Router
	TemplateRegistry *TemplateRegistry

	// PromptCompressor 可选的 Prompt 压缩器
	// 如果配置了且模板启用了压缩，将用于压缩 System Prompt
	PromptCompressor *EnhancedPromptCompressor

	// RecoveryHook 可选的会话恢复钩子
	// 应用层可以注册此钩子来实现自定义的会话恢复逻辑
	RecoveryHook types.RecoveryHook

	// VectorStoreFactory 向量存储工厂（用于 RAG 和语义记忆）
	VectorStoreFactory *factory.StoreFactory

	// EmbedderFactory 嵌入模型工厂（用于 RAG 和语义记忆）
	EmbedderFactory *factory.EmbedderFactory
}

// TemplateRegistry 模板注册表
type TemplateRegistry struct {
	templates map[string]*types.AgentTemplateDefinition
}

// NewTemplateRegistry 创建模板注册表
func NewTemplateRegistry() *TemplateRegistry {
	return &TemplateRegistry{
		templates: make(map[string]*types.AgentTemplateDefinition),
	}
}

// Register 注册模板
func (tr *TemplateRegistry) Register(template *types.AgentTemplateDefinition) {
	tr.templates[template.ID] = template
}

// Get 获取模板
func (tr *TemplateRegistry) Get(id string) (*types.AgentTemplateDefinition, error) {
	template, ok := tr.templates[id]
	if !ok {
		return nil, &TemplateNotFoundError{ID: id}
	}
	return template, nil
}

// List 列出所有模板
func (tr *TemplateRegistry) List() []*types.AgentTemplateDefinition {
	templates := make([]*types.AgentTemplateDefinition, 0, len(tr.templates))
	for _, t := range tr.templates {
		templates = append(templates, t)
	}
	return templates
}

// TemplateNotFoundError 模板未找到错误
type TemplateNotFoundError struct {
	ID string
}

func (e *TemplateNotFoundError) Error() string {
	return "template not found: " + e.ID
}
