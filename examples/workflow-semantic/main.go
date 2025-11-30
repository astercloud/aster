// WorkflowSemantic 演示如何在 Workflow Agent 中集成 SemanticMemory，
// 实现跨步骤的语义记忆共享和上下文感知的工作流执行。
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/agent/workflow"
	"github.com/astercloud/aster/pkg/memory"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/session"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/stream"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/builtin"
	"github.com/astercloud/aster/pkg/types"
	"github.com/astercloud/aster/pkg/vector"
)

// - 使用 SemanticMemory 索引一组知识片段
// - Workflow Agent 在执行时先进行语义检索, 再将检索结果作为上下文交给 LLM Agent 生成回答
//
// 注意: 示例默认使用 MockEmbedder + MemoryStore 实现语义记忆, 运行不依赖外部服务。
// 如果你已经在 agentsdk.yaml 中配置了 pgvector + OpenAI, 可以在实际项目中复用相同配置。

func main() {
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

	// 2. 索引一些百科知识片段
	docs := []struct {
		id   string
		text string
		meta map[string]any
	}{
		{
			id:   "doc-paris",
			text: "Paris is the capital and most populous city of France.",
			meta: map[string]any{"user_id": "alice", "resource_id": "world-facts"},
		},
		{
			id:   "doc-berlin",
			text: "Berlin is the capital city of Germany.",
			meta: map[string]any{"user_id": "alice", "resource_id": "world-facts"},
		},
		{
			id:   "doc-tokyo",
			text: "Tokyo is the capital of Japan and one of its 47 prefectures.",
			meta: map[string]any{"user_id": "alice", "resource_id": "asia-notes"},
		},
	}

	for _, d := range docs {
		if err := semMem.Index(ctx, d.id, d.text, d.meta); err != nil {
			log.Fatalf("index %s: %v", d.id, err)
		}
	}

	// 3. 初始化 Agent 依赖 (与 examples/server-http 类似, 但使用内存 Store)
	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry)

	memStore := storepkg()

	deps := &agent.Dependencies{
		Store:          memStore,
		SandboxFactory: sandbox.NewFactory(),
		ToolRegistry:   toolRegistry,
		// 使用多提供商工厂，根据模板中的模型配置选择实际模型。
		// 默认会使用 Anthropic 提供商，因此需要设置 ANTHROPIC_API_KEY。
		ProviderFactory: provider.NewMultiProviderFactory(),
		TemplateRegistry: func() *agent.TemplateRegistry {
			tr := agent.NewTemplateRegistry()
			tr.Register(&types.AgentTemplateDefinition{
				ID:    "semantic-qa",
				Model: "claude-sonnet-4-5",
				SystemPrompt: "You are a helpful assistant. Use the provided context if available " +
					"and answer the user's question concisely.",
				Tools: []any{"Read", "Write"},
			})
			return tr
		}(),
	}

	// 4. 构建基于 SemanticMemory + Agent 的 workflow.Agent
	semanticQA := NewSemanticQAWorkflowAgent("SemanticQA", "semantic-qa", semMem, deps)

	// 5. 使用 SequentialAgent 包装, 以展示 workflow 接口用法(这里只有一个子 Agent)
	seq, err := workflow.NewSequentialAgent(workflow.SequentialConfig{
		Name: "SemanticQAPipeline",
		SubAgents: []workflow.Agent{
			semanticQA,
		},
	})
	if err != nil {
		log.Fatalf("create sequential workflow: %v", err)
	}

	// 6. 执行工作流
	question := "What is the capital of France?"
	fmt.Println("=== Workflow: Semantic QA ===")
	fmt.Println("Question:", question)
	fmt.Println()

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
}

// storepkg 创建一个内存 Store, 用于存储 Agent 状态和消息。
func storepkg() store.Store {
	mem, err := store.NewJSONStore("./.aster-workflow-semantic")
	if err != nil {
		log.Fatalf("create store: %v", err)
	}
	return mem
}

// SemanticQAWorkflowAgent 实现 workflow.Agent, 结合 SemanticMemory + agent.Agent
// 实现一个简单的 RAG 工作流步骤。
type SemanticQAWorkflowAgent struct {
	name       string
	templateID string
	semMem     *memory.SemanticMemory
	deps       *agent.Dependencies
}

func NewSemanticQAWorkflowAgent(name, templateID string, sm *memory.SemanticMemory, deps *agent.Dependencies) *SemanticQAWorkflowAgent {
	return &SemanticQAWorkflowAgent{
		name:       name,
		templateID: templateID,
		semMem:     sm,
		deps:       deps,
	}
}

func (a *SemanticQAWorkflowAgent) Name() string {
	return a.name
}

func (a *SemanticQAWorkflowAgent) Execute(ctx context.Context, message string) *stream.Reader[*session.Event] {
	reader, writer := stream.Pipe[*session.Event](10)

	go func() {
		defer writer.Close()

		// 1) 语义检索: 在 alice/world-facts 命名空间内查询
		meta := map[string]any{
			"user_id":     "alice",
			"resource_id": "world-facts",
		}

		hits, err := a.semMem.Search(ctx, message, meta, 3)
		if err != nil {
			writer.Send(nil, fmt.Errorf("semantic search: %w", err))
			return
		}

		contextText := ""
		for i, h := range hits {
			if i > 0 {
				contextText += "\n\n"
			}
			text, _ := h.Metadata["text"].(string)
			contextText += fmt.Sprintf("[DOC %d] %s", i+1, text)
		}

		if writer.Send(&session.Event{
			ID:           fmt.Sprintf("evt-%s-context-%d", a.name, time.Now().UnixNano()),
			Timestamp:    time.Now(),
			InvocationID: "workflow-semantic-qa",
			AgentID:      a.name,
			Author:       "system",
			Content: types.Message{
				Role:    types.RoleSystem,
				Content: "Semantic context:\n\n" + contextText,
			},
		}, nil) {
			return
		}

		// 2) 调用底层 LLM Agent 生成回答
		agentConfig := &types.AgentConfig{
			TemplateID: a.templateID,
			Metadata: map[string]any{
				"workflow_step": a.name,
			},
		}

		ag, err := agent.Create(ctx, agentConfig, a.deps)
		if err != nil {
			writer.Send(nil, fmt.Errorf("create agent: %w", err))
			return
		}
		defer func() { _ = ag.Close() }()

		prompt := fmt.Sprintf("Use the following context if it is helpful:\n\n%s\n\nQuestion: %s", contextText, message)
		res, err := ag.Chat(ctx, prompt)
		writer.Send(&session.Event{
			ID:           fmt.Sprintf("evt-%s-answer-%d", a.name, time.Now().UnixNano()),
			Timestamp:    time.Now(),
			InvocationID: "workflow-semantic-qa",
			AgentID:      a.name,
			Author:       "assistant",
			Content: types.Message{
				Role:    types.RoleAssistant,
				Content: res.Text,
			},
		}, err)
	}()

	return reader
}
