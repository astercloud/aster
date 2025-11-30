// StructuredOutput 演示结构化输出功能，包括 JSON Schema 生成、
// 类型化解析、字段验证和自定义验证逻辑。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/structured"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/builtin"
	"github.com/astercloud/aster/pkg/types"
)

// Task 任务结构
type Task struct {
	ID          string   `json:"id" required:"true"`
	Title       string   `json:"title" required:"true"`
	Description string   `json:"description"`
	Priority    int      `json:"priority"`
	Tags        []string `json:"tags"`
	Completed   bool     `json:"completed"`
}

// TaskList 任务列表
type TaskList struct {
	Tasks []Task `json:"tasks" required:"true"`
	Total int    `json:"total"`
}

func main() {
	ctx := context.Background()

	// 创建依赖
	jsonStore, err := store.NewJSONStore(".aster-structured-output")
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}

	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry)

	deps := &agent.Dependencies{
		Store:            jsonStore,
		SandboxFactory:   sandbox.NewFactory(),
		ToolRegistry:     toolRegistry,
		ProviderFactory:  provider.NewMultiProviderFactory(),
		TemplateRegistry: createTemplateRegistry(),
	}

	// 创建 Agent 配置（启用 Structured Output Middleware）
	config := &types.AgentConfig{
		TemplateID: "structured-agent",
		ModelConfig: &types.ModelConfig{
			Provider: "anthropic",
			Model:    "claude-3-5-sonnet-20241022",
		},
		Middlewares: []string{"structured_output"},
		MiddlewareConfig: map[string]map[string]any{
			"structured_output": {
				"enabled":           true,
				"required_fields":   []string{"tasks", "total"},
				"allow_text_backup": false,
			},
		},
	}

	// 创建 Agent
	ag, err := agent.Create(ctx, config, deps)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	defer func() {
		if err := ag.Close(); err != nil {
			log.Printf("Failed to close agent: %v", err)
		}
	}()

	// 示例 1: 生成任务列表
	fmt.Println("=== Example 1: Generate Task List ===")
	prompt := `Create a task list for building a web application.
Return the result in JSON format with the following structure:
{
  "tasks": [
    {
      "id": "task-1",
      "title": "Task title",
      "description": "Task description",
      "priority": 1,
      "tags": ["tag1", "tag2"],
      "completed": false
    }
  ],
  "total": 5
}`

	result, err := ag.Chat(ctx, prompt)
	if err != nil {
		log.Fatalf("Chat failed: %v", err)
	}

	// 解析结构化输出
	parser := structured.NewTypedParser(nil)
	var taskList TaskList
	if err := parser.ParseInto(ctx, result.Text, &taskList); err != nil {
		log.Fatalf("Parse failed: %v", err)
	}

	fmt.Printf("Parsed %d tasks:\n", taskList.Total)
	for i, task := range taskList.Tasks {
		fmt.Printf("%d. [%s] %s (Priority: %d)\n", i+1, task.ID, task.Title, task.Priority)
	}

	// 示例 2: 使用 Schema 验证
	fmt.Println("\n=== Example 2: With Schema Validation ===")

	schema := structured.MustGenerateSchema(Task{})
	fmt.Printf("Generated Schema:\n")
	schemaJSON, _ := schema.ToJSON()
	fmt.Println(schemaJSON)

	// 示例 3: 类型化解析
	fmt.Println("\n=== Example 3: Typed Parsing ===")

	spec := structured.TypedOutputSpec{
		StructType:     &TaskList{},
		RequiredFields: []string{"tasks", "total"},
		Strict:         true,
		CustomValidation: func(data any) error {
			taskList := data.(*TaskList)
			if taskList.Total != len(taskList.Tasks) {
				return fmt.Errorf("total count mismatch: expected %d, got %d", len(taskList.Tasks), taskList.Total)
			}
			return nil
		},
	}

	parseResult, err := structured.ParseTyped(ctx, result.Text, spec)
	if err != nil {
		log.Fatalf("Typed parse failed: %v", err)
	}

	if parseResult.Success {
		fmt.Println("✓ Parsing successful")
		fmt.Printf("✓ Validation passed\n")

		parsedList := parseResult.Data.(*TaskList)
		jsonOutput, _ := json.MarshalIndent(parsedList, "", "  ")
		fmt.Printf("\nParsed Data:\n%s\n", jsonOutput)
	} else {
		fmt.Println("✗ Parsing failed")
		for _, err := range parseResult.ValidationErrors {
			fmt.Printf("  - %s\n", err)
		}
	}
}

func createTemplateRegistry() *agent.TemplateRegistry {
	registry := agent.NewTemplateRegistry()

	registry.Register(&types.AgentTemplateDefinition{
		ID: "structured-agent",
		SystemPrompt: `You are an AI assistant that generates structured JSON outputs.

When asked to create data structures:
1. Always return valid JSON
2. Follow the specified schema exactly
3. Include all required fields
4. Use appropriate data types
5. Provide meaningful, realistic data

Wrap your JSON output in markdown code blocks for clarity.`,
		Model: "claude-3-5-sonnet-20241022",
		Tools: []string{},
	})

	return registry
}
