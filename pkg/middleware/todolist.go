package middleware

import (
	"context"
	"fmt"
	"log"

	"github.com/astercloud/aster/pkg/tools"
)

// TodoStatus 任务状态
type TodoStatus string

const (
	TodoStatusPending    TodoStatus = "pending"
	TodoStatusInProgress TodoStatus = "in_progress"
	TodoStatusCompleted  TodoStatus = "completed"
)

// TodoItem 任务项
type TodoItem struct {
	Content    string     `json:"content"`    // 任务内容描述
	Status     TodoStatus `json:"status"`     // 任务状态
	ActiveForm string     `json:"activeForm"` // 进行时描述
}

// TodoListMiddleware 任务列表中间件
// 功能:
// 1. 提供 write_todos 工具,允许 Agent 进行任务规划
// 2. 管理任务列表的状态
// 3. 引导 Agent 使用任务分解策略
type TodoListMiddleware struct {
	*BaseMiddleware
	todos       []TodoItem
	storeGetter func() any       // 获取当前任务列表
	storeSetter func([]TodoItem) // 设置任务列表
}

// TodoListMiddlewareConfig 任务列表中间件配置
type TodoListMiddlewareConfig struct {
	// StoreGetter 从外部状态获取任务列表
	// 如果为 nil, 使用内部存储
	StoreGetter func() any

	// StoreSetter 保存任务列表到外部状态
	// 如果为 nil, 使用内部存储
	StoreSetter func([]TodoItem)

	// EnableSystemPrompt 是否注入系统提示词
	EnableSystemPrompt bool
}

// NewTodoListMiddleware 创建任务列表中间件
func NewTodoListMiddleware(config *TodoListMiddlewareConfig) *TodoListMiddleware {
	if config == nil {
		config = &TodoListMiddlewareConfig{
			EnableSystemPrompt: true,
		}
	}

	m := &TodoListMiddleware{
		BaseMiddleware: NewBaseMiddleware("todolist", 50),
		todos:          []TodoItem{},
		storeGetter:    config.StoreGetter,
		storeSetter:    config.StoreSetter,
	}

	log.Printf("[TodoListMiddleware] Initialized")
	return m
}

// Tools 返回 write_todos 工具
func (m *TodoListMiddleware) Tools() []tools.Tool {
	return []tools.Tool{
		&WriteTodosTool{
			middleware: m,
		},
	}
}

// WrapModelCall 包装模型调用,注入系统提示词
func (m *TodoListMiddleware) WrapModelCall(ctx context.Context, req *ModelRequest, handler ModelCallHandler) (*ModelResponse, error) {
	// 注入任务规划提示词
	if req.SystemPrompt != "" {
		req.SystemPrompt += "\n\n" + TODO_LIST_SYSTEM_PROMPT
	} else {
		req.SystemPrompt = TODO_LIST_SYSTEM_PROMPT
	}

	// 调用下一层
	return handler(ctx, req)
}

// GetTodos 获取当前任务列表
func (m *TodoListMiddleware) GetTodos() []TodoItem {
	// 如果配置了外部 getter, 使用外部存储
	if m.storeGetter != nil {
		if todosInterface := m.storeGetter(); todosInterface != nil {
			if todos, ok := todosInterface.([]TodoItem); ok {
				return todos
			}
		}
	}

	// 否则使用内部存储
	return m.todos
}

// SetTodos 设置任务列表
func (m *TodoListMiddleware) SetTodos(todos []TodoItem) {
	// 如果配置了外部 setter, 使用外部存储
	if m.storeSetter != nil {
		m.storeSetter(todos)
	}

	// 同时更新内部存储
	m.todos = todos
}

// WriteTodosTool write_todos 工具实现
type WriteTodosTool struct {
	middleware *TodoListMiddleware
}

func (t *WriteTodosTool) Name() string {
	return "write_todos"
}

func (t *WriteTodosTool) Description() string {
	return "Create and manage a structured task list for tracking progress on complex, multi-step tasks"
}

func (t *WriteTodosTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"todos": map[string]any{
				"type":        "array",
				"description": "List of tasks to track. Each task should have content, status, and activeForm.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"content": map[string]any{
							"type":        "string",
							"description": "Task description in imperative form (e.g., 'Run tests', 'Build the project')",
						},
						"status": map[string]any{
							"type":        "string",
							"enum":        []string{"pending", "in_progress", "completed"},
							"description": "Task status: pending (not started), in_progress (currently working on), completed (finished)",
						},
						"activeForm": map[string]any{
							"type":        "string",
							"description": "Present continuous form shown during execution (e.g., 'Running tests', 'Building the project')",
						},
					},
					"required": []string{"content", "status", "activeForm"},
				},
			},
		},
		"required": []string{"todos"},
	}
}

func (t *WriteTodosTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	// 解析 todos 列表
	todosInterface, ok := input["todos"].([]any)
	if !ok {
		return nil, fmt.Errorf("todos must be an array")
	}

	var todos []TodoItem
	for i, todoInterface := range todosInterface {
		todoMap, ok := todoInterface.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("todo item %d must be an object", i)
		}

		content, _ := todoMap["content"].(string)
		statusStr, _ := todoMap["status"].(string)
		activeForm, _ := todoMap["activeForm"].(string)

		if content == "" || statusStr == "" || activeForm == "" {
			return nil, fmt.Errorf("todo item %d missing required fields", i)
		}

		todos = append(todos, TodoItem{
			Content:    content,
			Status:     TodoStatus(statusStr),
			ActiveForm: activeForm,
		})
	}

	// 保存任务列表
	t.middleware.SetTodos(todos)

	// 统计任务状态
	pending := 0
	inProgress := 0
	completed := 0
	for _, todo := range todos {
		switch todo.Status {
		case TodoStatusPending:
			pending++
		case TodoStatusInProgress:
			inProgress++
		case TodoStatusCompleted:
			completed++
		}
	}

	log.Printf("[WriteTodosTool] Updated task list: %d total (%d pending, %d in progress, %d completed)",
		len(todos), pending, inProgress, completed)

	return map[string]any{
		"ok":          true,
		"total":       len(todos),
		"pending":     pending,
		"in_progress": inProgress,
		"completed":   completed,
		"message":     fmt.Sprintf("Task list updated with %d tasks", len(todos)),
	}, nil
}

func (t *WriteTodosTool) Prompt() string {
	return TODO_TOOL_DESCRIPTION
}

// TODO_LIST_SYSTEM_PROMPT 任务列表系统提示词
const TODO_LIST_SYSTEM_PROMPT = `## Task Management

You have access to the **write_todos** tool for managing complex, multi-step tasks.

### When to Use Task Lists

Use write_todos when:
- Task requires 3 or more distinct steps
- Task is non-trivial and complex
- User explicitly requests a todo list
- User provides multiple tasks (numbered or comma-separated)
- After receiving new instructions (capture requirements as todos)
- When starting work on a task (mark as in_progress BEFORE beginning)
- After completing a task (mark as completed and add any new follow-up tasks)

### When NOT to Use

Skip write_todos when:
- Single, straightforward task
- Trivial task completable in <3 steps
- Purely conversational or informational request

### Task Management Rules

**IMPORTANT**:
- Task descriptions must have TWO forms:
  - content: Imperative form (e.g., "Run tests", "Build the project")
  - activeForm: Present continuous form (e.g., "Running tests", "Building the project")
- Exactly ONE task must be in_progress at any time (not less, not more)
- Mark tasks completed IMMEDIATELY after finishing (don't batch completions)
- Complete current tasks before starting new ones
- Remove tasks no longer relevant from the list entirely

**Task Completion Requirements**:
- ONLY mark as completed when FULLY accomplished
- If errors, blockers, or cannot finish, keep as in_progress
- When blocked, create new task describing what needs resolution
- Never mark completed if:
  - Tests are failing
  - Implementation is partial
  - Unresolved errors encountered
  - Couldn't find necessary files/dependencies

### Example Usage

Good task breakdown:
1. {"content": "Run all unit tests", "status": "in_progress", "activeForm": "Running all unit tests"}
2. {"content": "Fix failing tests", "status": "pending", "activeForm": "Fixing failing tests"}
3. {"content": "Build production bundle", "status": "pending", "activeForm": "Building production bundle"}

When in doubt, use task lists proactively to demonstrate thoroughness.`

// TODO_TOOL_DESCRIPTION write_todos 工具详细描述
const TODO_TOOL_DESCRIPTION = `Use this tool to create and manage a structured task list for tracking progress on complex tasks.

**When to use:**
- Complex multi-step tasks (3+ steps)
- User explicitly requests todo list
- User provides multiple tasks
- Non-trivial, complex work requiring careful planning

**When NOT to use:**
- Single straightforward task
- Trivial task (<3 steps)
- Purely conversational requests

**Task States:**
- **pending**: Task not yet started
- **in_progress**: Currently working on (ONLY ONE at a time)
- **completed**: Task finished successfully

**Important Rules:**
1. Each task MUST have both "content" (imperative) and "activeForm" (present continuous)
   - Good: content="Run tests", activeForm="Running tests"
   - Bad: content="Tests", activeForm="Tests" (not continuous form)

2. Task Management:
   - Mark in_progress BEFORE starting work
   - Mark completed IMMEDIATELY after finishing
   - Keep EXACTLY ONE task in_progress
   - Remove tasks no longer relevant

3. Completion Requirements:
   - ONLY mark completed when FULLY done
   - Keep in_progress if errors, blockers, or partial completion
   - Never mark completed if tests failing or errors unresolved

**Example:**
{
  "todos": [
    {"content": "Analyze requirements", "status": "completed", "activeForm": "Analyzing requirements"},
    {"content": "Implement feature X", "status": "in_progress", "activeForm": "Implementing feature X"},
    {"content": "Write unit tests", "status": "pending", "activeForm": "Writing unit tests"},
    {"content": "Update documentation", "status": "pending", "activeForm": "Updating documentation"}
  ]
}

Proactive task management demonstrates thoroughness and ensures all requirements are completed successfully.`
