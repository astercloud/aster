---
title: 工作流 + 语义记忆 示例
description: 在 Workflow Agent 中集成 SemanticMemory 构建简单 RAG 流程
---

# 工作流 + 语义记忆 示例

在前面的示例中, 我们已经通过 `SemanticMemory` 为 Agent 增强了语义检索能力。本示例进一步展示如何在 **Workflow Agent** 中结合语义检索与 LLM 调用, 搭建一个简单的 RAG 工作流。

目标:

- 使用 `SemanticMemory` 索引一组知识片段;
- Workflow 的步骤先调用语义检索获取上下文, 再调用 LLM 模型生成最终回答;
- 通过 `workflow.Agent` 接口将这一逻辑作为「可组合的步骤」集成到更大的工作流中。

示例代码位置: `examples/workflow-semantic/main.go`

## 1. 初始化语义记忆

示例中我们使用 **内存 VectorStore + MockEmbedder**, 方便本地直接运行:

```go
ctx := context.Background()

// 1. 初始化语义记忆: MemoryStore + MockEmbedder
store := vector.NewMemoryStore()
embedder := vector.NewMockEmbedder(16)
semMem := memory.NewSemanticMemory(memory.SemanticMemoryConfig{
    Store:          store,
    Embedder:       embedder,
    NamespaceScope: "resource",
    TopK:           3,
})
```

然后索引几条「世界知识」:

```go
docs := []struct {
    id   string
    text string
    meta map[string]interface{}
}{
    {
        id:   "doc-paris",
        text: "Paris is the capital and most populous city of France.",
        meta: map[string]interface{}{"user_id": "alice", "resource_id": "world-facts"},
    },
    {
        id:   "doc-berlin",
        text: "Berlin is the capital city of Germany.",
        meta: map[string]interface{}{"user_id": "alice", "resource_id": "world-facts"},
    },
}

for _, d := range docs {
    if err := semMem.Index(ctx, d.id, d.text, d.meta); err != nil {
        log.Fatalf("index %s: %v", d.id, err)
    }
}
```

## 2. 构建底层 LLM Agent 依赖

示例复用了与 `examples/server-http` 类似的依赖注入方式:

```go
toolRegistry := tools.NewRegistry()
builtin.RegisterAll(toolRegistry)

memStore := storepkg() // 使用 JSON Store 持久化 Agent 状态

deps := &agent.Dependencies{
    Store:           memStore,
    SandboxFactory:  sandbox.NewFactory(),
    ToolRegistry:    toolRegistry,
    // 使用多提供商工厂, 默认 Anthropic, 需要配置 ANTHROPIC_API_KEY
    ProviderFactory: provider.NewMultiProviderFactory(),
    TemplateRegistry: func() *agent.TemplateRegistry {
        tr := agent.NewTemplateRegistry()
        tr.Register(&types.AgentTemplateDefinition{
            ID:    "semantic-qa",
            Model: "claude-sonnet-4-5",
            SystemPrompt: "You are a helpful assistant. Use the provided context if available " +
                "and answer the user's question concisely.",
            Tools: []interface{}{"Read", "Write"},
        })
        return tr
    }(),
}
```

> 提示: 要让示例真正调用在线模型, 需要在运行前设置 `ANTHROPIC_API_KEY` 环境变量; 否则 `agent.Create` 会返回错误。

## 3. 定义 SemanticQAWorkflowAgent

核心是一个实现了 `workflow.Agent` 接口的结构体:

```go
type SemanticQAWorkflowAgent struct {
    name       string
    templateID string
    semMem     *memory.SemanticMemory
    deps       *agent.Dependencies
}

func (a *SemanticQAWorkflowAgent) Name() string {
    return a.name
}
```

执行逻辑分两步:

1. 使用 `SemanticMemory` 做语义检索, 产出上下文 Event;
2. 创建子 `agent.Agent`, 将上下文注入到 prompt 中, 得到最终回答 Event。

```go
func (a *SemanticQAWorkflowAgent) Execute(
    ctx context.Context,
    message string,
) *stream.Reader[*session.Event] {
    reader, writer := stream.Pipe[*session.Event](10)

    go func() {
        defer writer.Close()

        // 1) 语义检索
        meta := map[string]interface{}{
            "user_id":     "alice",
            "resource_id": "world-facts",
        }
        hits, err := a.semMem.Search(ctx, message, meta, 3)
        if err != nil {
            writer.Send(nil, fmt.Errorf("semantic search: %w", err))
            return
        }

        contextText := buildContext(hits) // 将命中文本拼接成上下文

        // 向外发出「上下文事件」
        if writer.Send(&session.Event{
            ID:        "...-context",
            Timestamp: time.Now(),
            AgentID:   a.name,
            Author:    "system",
            Content: types.Message{
                Role:    types.RoleSystem,
                Content: "Semantic context:\n\n" + contextText,
            },
        }, nil) {
            return
        }

        // 2) 创建底层 Agent, 调用 LLM
        ag, err := agent.Create(ctx, &types.AgentConfig{
            TemplateID: a.templateID,
            Metadata: map[string]interface{}{
                "workflow_step": a.name,
            },
        }, a.deps)
        if err != nil {
            writer.Send(nil, fmt.Errorf("create agent: %w", err))
            return
        }
        defer ag.Close()

        prompt := fmt.Sprintf(
            "Use the following context if it is helpful:\n\n%s\n\nQuestion: %s",
            contextText, message,
        )
        res, err := ag.Chat(ctx, prompt)

        // 向外发出「回答事件」
        writer.Send(&session.Event{
            ID:        "...-answer",
            Timestamp: time.Now(),
            AgentID:   a.name,
            Author:    "assistant",
            Content: types.Message{
                Role:    types.RoleAssistant,
                Content: res.Text,
            },
        }, err)
    }()

    return reader
}
```

## 4. 通过 SequentialAgent 组合工作流

为了演示 Workflow 接口, 示例将 `SemanticQAWorkflowAgent` 包装成一个顺序工作流:

```go
semanticQA := NewSemanticQAWorkflowAgent("SemanticQA", "semantic-qa", semMem, deps)

seq, err := workflow.NewSequentialAgent(workflow.SequentialConfig{
    Name: "SemanticQAPipeline",
    SubAgents: []workflow.Agent{
        semanticQA,
    },
})
if err != nil {
    log.Fatalf("create sequential workflow: %v", err)
}

question := "What is the capital of France?"
reader := seq.Execute(ctx, question)
for {
    ev, err := reader.Recv()
    if err != nil {
        if errors.Is(err, io.EOF) {
            break
        }
        log.Fatalf("workflow error: %v", err)
    }
    if ev == nil {
        continue
    }
    fmt.Printf("[%s] %s\n", ev.AgentID, ev.Content.Content)
}
```

终端输出将类似:

```text
=== Workflow: Semantic QA ===
Question: What is the capital of France?

[SemanticQA] Semantic context:

[DOC 1] Paris is the capital and most populous city of France.

[SemanticQA] Paris is the capital and most populous city of France.
```

## 5. 运行示例

在仓库根目录执行:

```bash
cd examples
go run ./workflow-semantic
```

- 如果未配置真实的向量存储/embedding 服务, 示例会使用内存向量存储 + MockEmbedder, 仅用于演示流程;
- 若已在 `aster.yaml` 中配置了 pgvector + OpenAI 等 adapter, 可以在自己的项目中复用相同的 `SemanticMemory` 配置, 将本示例中的 Workflow Agent 变成真正的生产级 RAG 步骤。
