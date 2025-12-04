---
title: 可选知识工具（基于核心管线）
weight: 40
---

> 适用场景：只想要轻量 RAG（向量检索）而不引入完整知识管理器；需要显式控制哪些 Agent 有知识检索能力。

## 组成

- 核心管线：`pkg/knowledge/core`（依赖 `VectorStore` + `Embedder`）。
- 工厂：`pkg/tools/knowledge` 提供 `KnowledgeAdd` / `KnowledgeSearch` 工具 **工厂**，不会默认注册到 builtin registry。

## 快速示例

```go
import (
  "github.com/astercloud/aster/pkg/knowledge/core"
  knowledgeTools "github.com/astercloud/aster/pkg/tools/knowledge"
  "github.com/astercloud/aster/pkg/tools"
  "github.com/astercloud/aster/pkg/types"
  "github.com/astercloud/aster/pkg/vector"
)

// 1) 创建核心管线（可替换为 pgvector/真实 embedder）
pipe, _ := core.NewPipeline(core.PipelineConfig{
  Store:    vector.NewMemoryStore(),
  Embedder: vector.NewMockEmbedder(32),
  Namespace: "demo",
})

// 2) 创建工厂产出工具
factory := knowledgeTools.NewFactory(pipe)
addTool, _ := factory.KnowledgeAddTool()
searchTool, _ := factory.KnowledgeSearchTool()

// 3) 注册到 Agent 的工具表（显式控制）
registry := tools.NewRegistry()
registry.Register(addTool.Name(), func(cfg map[string]interface{}) (tools.Tool, error) { return addTool, nil })
registry.Register(searchTool.Name(), func(cfg map[string]interface{}) (tools.Tool, error) { return searchTool, nil })

agentCfg := &types.AgentConfig{
  TemplateID: "demo",
  Tools:      []string{addTool.Name(), searchTool.Name()},
}

// 创建 Agent 时将 registry 作为依赖注入（见 pkg/agent/Create 的 deps.ToolRegistry）
```

## 输入/输出约定

- `KnowledgeAdd` 输入：`text`（必填），可选 `id`、`namespace`、`metadata`。返回写入的 chunk ID。
- `KnowledgeSearch` 输入：`query`（必填），可选 `top_k`、`namespace`、`metadata` 过滤。返回 `results: [{id, score, text, metadata}]`。

## 注意事项

- 工具不默认注册；需显式添加，避免循环依赖和全局耦合。
- 默认内存向量库 + MockEmbedder 仅用于示例；生产请替换为 pgvector/云向量库和真实 embedder。
- 可与 `structured_output` 中间件搭配，解析工具结果或模型响应的结构化 JSON。
- 需要替换向量库：用 `NewFactoryPipeline(customStore, customEmbedder)` 创建工厂即可。
