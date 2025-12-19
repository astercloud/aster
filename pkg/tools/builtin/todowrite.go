package builtin

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/types"
)

// TodoWriteTool 任务管理工具
// 支持创建和管理结构化任务列表
type TodoWriteTool struct{}

// TodoItem 单个任务项
type TodoItem struct {
	ID          string         `json:"id"`
	Content     string         `json:"content"`
	Status      string         `json:"status"` // "pending", "in_progress", "completed"
	ActiveForm  string         `json:"activeForm"`
	Priority    int            `json:"priority,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	CompletedAt *time.Time     `json:"completedAt,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// TodoList 任务列表
type TodoList struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Todos     []TodoItem     `json:"todos"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// NewTodoWriteTool 创建TodoWrite工具
func NewTodoWriteTool(config map[string]any) (tools.Tool, error) {
	return &TodoWriteTool{}, nil
}

func (t *TodoWriteTool) Name() string {
	return "TodoWrite"
}

func (t *TodoWriteTool) Description() string {
	return "创建和管理结构化任务列表"
}

func (t *TodoWriteTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"todos": map[string]any{
				"type":        "array",
				"description": "任务项数组，包含content、status、activeForm等字段",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"content": map[string]any{
							"type":        "string",
							"description": "任务描述内容",
						},
						"status": map[string]any{
							"type":        "string",
							"enum":        []string{"pending", "in_progress", "completed"},
							"description": "任务状态",
						},
						"activeForm": map[string]any{
							"type":        "string",
							"description": "任务的主动形式描述（进行中的状态描述）",
						},
						"priority": map[string]any{
							"type":        "integer",
							"description": "任务优先级（数值越大优先级越高）",
						},
					},
					"required": []string{"content", "status", "activeForm"},
				},
			},
			"list_name": map[string]any{
				"type":        "string",
				"description": "任务列表名称，默认为'default'",
			},
			"action": map[string]any{
				"type":        "string",
				"description": "操作类型：create, update, delete, clear，默认为create",
			},
			"todo_id": map[string]any{
				"type":        "string",
				"description": "要更新或删除的任务ID",
			},
		},
		"required": []string{"todos"},
	}
}

func (t *TodoWriteTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	// 验证必需参数
	if err := ValidateRequired(input, []string{"todos"}); err != nil {
		return NewClaudeErrorResponse(err), nil
	}

	action := GetStringParam(input, "action", "create")
	listName := GetStringParam(input, "list_name", "default")
	todoID := GetStringParam(input, "todo_id", "")

	// 获取任务项数据
	todosData, ok := input["todos"].([]any)
	if !ok {
		return NewClaudeErrorResponse(errors.New("todos must be an array")), nil
	}

	// 转换为TodoItem
	todos := make([]TodoItem, 0, len(todosData))
	baseTime := time.Now().UnixNano()
	for i, todoData := range todosData {
		todoMap, ok := todoData.(map[string]any)
		if !ok {
			return NewClaudeErrorResponse(errors.New("each todo must be an object")), nil
		}

		content := GetStringParam(todoMap, "content", "")
		status := GetStringParam(todoMap, "status", "pending")
		activeForm := GetStringParam(todoMap, "activeForm", "")
		priority := GetIntParam(todoMap, "priority", 0)

		if content == "" {
			return NewClaudeErrorResponse(errors.New("todo content cannot be empty")), nil
		}

		if activeForm == "" {
			return NewClaudeErrorResponse(errors.New("todo activeForm cannot be empty")), nil
		}

		// 验证状态
		validStatuses := []string{"pending", "in_progress", "completed"}
		statusValid := slices.Contains(validStatuses, status)
		if !statusValid {
			return NewClaudeErrorResponse(
				fmt.Errorf("invalid status: %s", status),
				"支持的状态: pending, in_progress, completed",
			), nil
		}

		todo := TodoItem{
			Content:    content,
			Status:     status,
			ActiveForm: activeForm,
			Priority:   priority,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			Metadata:   make(map[string]any),
		}

		// 如果有todo_id，使用它
		if id, exists := todoMap["id"]; exists {
			if idStr, ok := id.(string); ok {
				todo.ID = idStr
			}
		}

		// 生成ID（如果没有提供），使用基础时间戳+索引确保唯一性
		if todo.ID == "" {
			todo.ID = fmt.Sprintf("todo_%d_%d", baseTime, i)
		}

		// 处理completed状态的完成时间
		if status == "completed" {
			now := time.Now()
			todo.CompletedAt = &now
		}

		todos = append(todos, todo)
	}

	start := time.Now()

	// 获取全局任务列表管理器
	todoManager := GetGlobalTodoManager()

	// 加载现有任务列表
	todoList, err := todoManager.LoadTodoList(listName)
	if err != nil {
		// 如果不存在，创建新的任务列表
		todoList = &TodoList{
			ID:        fmt.Sprintf("list_%s_%d", listName, time.Now().UnixNano()),
			Name:      listName,
			Todos:     []TodoItem{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Metadata:  make(map[string]any),
		}
	}

	// 验证单一 in_progress 约束
	if err := t.validateSingleInProgress(todoList, todos); err != nil {
		return NewClaudeErrorResponse(err,
			"Complete the current in_progress task before starting a new one",
			"Mark the existing in_progress task as completed first",
		), nil
	}

	// 执行操作
	var result any
	var operationErr error

	switch action {
	case "create":
		result = t.createTodos(todoList, todos)
		operationErr = todoManager.StoreTodoList(todoList)
	case "update":
		if todoID == "" {
			return NewClaudeErrorResponse(errors.New("todo_id is required for update action")), nil
		}
		result = t.updateTodo(todoList, todoID, todos[0])
		operationErr = todoManager.StoreTodoList(todoList)
	case "delete":
		if todoID == "" {
			return NewClaudeErrorResponse(errors.New("todo_id is required for delete action")), nil
		}
		result = t.deleteTodo(todoList, todoID)
		operationErr = todoManager.StoreTodoList(todoList)
	case "clear":
		result = t.clearTodos(todoList)
		operationErr = todoManager.StoreTodoList(todoList)
	default:
		return NewClaudeErrorResponse(
			fmt.Errorf("invalid action: %s", action),
			"支持的操作: create, update, delete, clear",
		), nil
	}

	// 检查操作是否成功
	if operationErr != nil {
		return map[string]any{
			"ok":          false,
			"error":       fmt.Sprintf("failed to save todo list: %v", operationErr),
			"action":      action,
			"list_name":   listName,
			"duration_ms": time.Since(start).Milliseconds(),
		}, nil
	}

	duration := time.Since(start)

	// 构建响应
	response := map[string]any{
		"ok":              true,
		"action":          action,
		"list_name":       listName,
		"list_id":         todoList.ID,
		"todos":           todoList.Todos,
		"total_todos":     len(todoList.Todos),
		"duration_ms":     duration.Milliseconds(),
		"updated_at":      todoList.UpdatedAt.Unix(),
		"storage":         "persistent",
		"storage_backend": "FileTodoManager",
	}

	// 添加统计信息
	response["pending_count"] = t.countTodosByStatus(todoList.Todos, "pending")
	response["in_progress_count"] = t.countTodosByStatus(todoList.Todos, "in_progress")
	response["completed_count"] = t.countTodosByStatus(todoList.Todos, "completed")

	// 添加操作结果
	if resultMap, ok := result.(map[string]any); ok {
		maps.Copy(response, resultMap)
	}

	// 发送 Todo 更新事件到 Progress 通道
	t.emitTodoUpdateEvent(tc, todoList.Todos)

	return response, nil
}

// emitTodoUpdateEvent 发送 Todo 列表更新事件
func (t *TodoWriteTool) emitTodoUpdateEvent(tc *tools.ToolContext, todos []TodoItem) {
	// 转换为 types.TodoItem 格式
	typesTodos := make([]types.TodoItem, len(todos))
	for i, todo := range todos {
		typesTodos[i] = types.TodoItem{
			ID:         todo.ID,
			Content:    todo.Content,
			ActiveForm: todo.ActiveForm,
			Status:     todo.Status,
			Priority:   todo.Priority,
			CreatedAt:  todo.CreatedAt,
			UpdatedAt:  todo.UpdatedAt,
		}
	}

	// 通过 Reporter 发送中间结果
	if tc != nil && tc.Reporter != nil {
		tc.Reporter.Intermediate("todo_update", map[string]any{
			"todos":      typesTodos,
			"event_type": "todo_update",
		})
	}

	// 通过 Emit 函数发送事件
	if tc != nil && tc.Emit != nil {
		tc.Emit("todo_update", &types.ProgressTodoUpdateEvent{
			Todos: typesTodos,
		})
	}
}

// validateSingleInProgress 验证同时只能有一个 in_progress 任务
// 返回错误信息，如果验证通过返回 nil
func (t *TodoWriteTool) validateSingleInProgress(todoList *TodoList, newTodos []TodoItem) error {
	// 统计新任务中的 in_progress 数量
	newInProgressCount := 0
	for _, todo := range newTodos {
		if todo.Status == "in_progress" {
			newInProgressCount++
		}
	}

	// 统计现有任务中的 in_progress 数量（排除将被更新的任务）
	existingInProgressCount := 0
	for _, existing := range todoList.Todos {
		if existing.Status == "in_progress" {
			// 检查是否在新任务列表中被更新
			isBeingUpdated := false
			for _, newTodo := range newTodos {
				if newTodo.ID == existing.ID {
					isBeingUpdated = true
					break
				}
			}
			if !isBeingUpdated {
				existingInProgressCount++
			}
		}
	}

	totalInProgress := existingInProgressCount + newInProgressCount
	if totalInProgress > 1 {
		// 找出现有的 in_progress 任务
		var existingInProgressTask string
		for _, existing := range todoList.Todos {
			if existing.Status == "in_progress" {
				existingInProgressTask = existing.Content
				break
			}
		}
		return fmt.Errorf("only one task can be in_progress at a time. Current in_progress task: %q", existingInProgressTask)
	}

	return nil
}

// createTodos 创建新任务
func (t *TodoWriteTool) createTodos(todoList *TodoList, todos []TodoItem) map[string]any {
	addedTodos := make([]TodoItem, 0, len(todos))

	for _, todo := range todos {
		// 检查是否已存在相同ID的任务
		exists := false
		for _, existing := range todoList.Todos {
			if existing.ID == todo.ID {
				exists = true
				break
			}
		}

		if !exists {
			todoList.Todos = append(todoList.Todos, todo)
			addedTodos = append(addedTodos, todo)
		}
	}

	todoList.UpdatedAt = time.Now()

	return map[string]any{
		"added_count": len(addedTodos),
		"added_todos": addedTodos,
	}
}

// updateTodo 更新任务
func (t *TodoWriteTool) updateTodo(todoList *TodoList, todoID string, updatedTodo TodoItem) map[string]any {
	for i, existing := range todoList.Todos {
		if existing.ID == todoID {
			// 保留创建时间
			updatedTodo.CreatedAt = existing.CreatedAt
			updatedTodo.ID = existing.ID

			// 更新时间
			updatedTodo.UpdatedAt = time.Now()

			// 如果状态变为completed，设置完成时间
			if updatedTodo.Status == "completed" && existing.Status != "completed" {
				now := time.Now()
				updatedTodo.CompletedAt = &now
			} else if updatedTodo.Status != "completed" {
				updatedTodo.CompletedAt = nil
			}

			// 保留元数据
			if existing.Metadata != nil {
				for k, v := range existing.Metadata {
					if _, exists := updatedTodo.Metadata[k]; !exists {
						updatedTodo.Metadata[k] = v
					}
				}
			}

			todoList.Todos[i] = updatedTodo
			todoList.UpdatedAt = time.Now()

			return map[string]any{
				"updated":         true,
				"previous_status": existing.Status,
				"new_status":      updatedTodo.Status,
			}
		}
	}

	return map[string]any{
		"updated": false,
		"reason":  "todo not found",
	}
}

// deleteTodo 删除任务
func (t *TodoWriteTool) deleteTodo(todoList *TodoList, todoID string) map[string]any {
	for i, existing := range todoList.Todos {
		if existing.ID == todoID {
			// 删除任务
			todoList.Todos = append(todoList.Todos[:i], todoList.Todos[i+1:]...)
			todoList.UpdatedAt = time.Now()

			return map[string]any{
				"deleted":      true,
				"deleted_todo": existing,
			}
		}
	}

	return map[string]any{
		"deleted": false,
		"reason":  "todo not found",
	}
}

// clearTodos 清空任务列表
func (t *TodoWriteTool) clearTodos(todoList *TodoList) map[string]any {
	deletedCount := len(todoList.Todos)
	todoList.Todos = []TodoItem{}
	todoList.UpdatedAt = time.Now()

	return map[string]any{
		"deleted_count": deletedCount,
		"action":        "cleared_all_todos",
	}
}

// countTodosByStatus 按状态统计任务数量
func (t *TodoWriteTool) countTodosByStatus(todos []TodoItem, status string) int {
	count := 0
	for _, todo := range todos {
		if todo.Status == status {
			count++
		}
	}
	return count
}

func (t *TodoWriteTool) Prompt() string {
	return `创建和管理结构化任务列表。

功能特性：
- 创建、更新、删除任务项
- 支持任务状态管理（pending, in_progress, completed）
- 任务优先级设置
- 自动ID生成和时间戳
- 任务统计和进度跟踪

使用指南：
- todos: 必需参数，任务项数组
- list_name: 可选参数，任务列表名称
- action: 可选参数，操作类型（create/update/delete/clear）
- todo_id: 可选参数，要操作的任务ID

任务状态：
- pending: 待处理任务
- in_progress: 进行中任务（同时只能有一个！）
- completed: 已完成任务

任务字段：
- content: 任务描述内容，祈使句形式（必需），如"Run tests"
- status: 任务状态（必需）
- activeForm: 任务的进行时形式描述（必需），如"Running tests"
- priority: 任务优先级（可选）

重要约束：
- 同时只能有一个任务处于 in_progress 状态
- 完成当前 in_progress 任务后才能开始下一个
- 任务完成后应立即标记为 completed
- content 用祈使句，activeForm 用进行时

注意事项：
- 使用持久化存储系统，数据安全可靠
- 支持任务完成时间自动记录
- 提供详细的任务统计信息
- 支持任务列表的备份和恢复`
}

// Examples 返回 TodoWrite 工具的使用示例
// 实现 ExampleableTool 接口，帮助 LLM 更准确地调用工具
func (t *TodoWriteTool) Examples() []tools.ToolExample {
	return []tools.ToolExample{
		{
			Description: "创建任务列表",
			Input: map[string]any{
				"todos": []map[string]any{
					{
						"content":    "实现用户认证模块",
						"status":     "pending",
						"activeForm": "实现用户认证模块中",
					},
					{
						"content":    "编写单元测试",
						"status":     "pending",
						"activeForm": "编写单元测试中",
					},
				},
			},
		},
		{
			Description: "更新任务状态为进行中",
			Input: map[string]any{
				"action":  "update",
				"todo_id": "todo_123456",
				"todos": []map[string]any{
					{
						"content":    "实现用户认证模块",
						"status":     "in_progress",
						"activeForm": "实现用户认证模块中",
					},
				},
			},
		},
		{
			Description: "标记任务为已完成",
			Input: map[string]any{
				"action":  "update",
				"todo_id": "todo_123456",
				"todos": []map[string]any{
					{
						"content":    "实现用户认证模块",
						"status":     "completed",
						"activeForm": "已完成用户认证模块",
					},
				},
			},
		},
	}
}
