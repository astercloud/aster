package structured

import (
	"context"
	"testing"
)

type TestTask struct {
	ID          string   `json:"id" required:"true"`
	Title       string   `json:"title" required:"true" description:"Task title"`
	Description string   `json:"description"`
	Priority    int      `json:"priority"`
	Tags        []string `json:"tags"`
	Completed   bool     `json:"completed"`
}

type TestTaskList struct {
	Tasks []TestTask `json:"tasks" required:"true"`
	Total int        `json:"total"`
}

func TestTypedParser_ParseInto(t *testing.T) {
	parser := NewTypedParser(nil)

	jsonText := `{
		"id": "task-1",
		"title": "Test Task",
		"description": "This is a test",
		"priority": 1,
		"tags": ["test", "demo"],
		"completed": false
	}`

	var task TestTask
	err := parser.ParseInto(context.Background(), jsonText, &task)
	if err != nil {
		t.Fatalf("ParseInto failed: %v", err)
	}

	if task.ID != "task-1" {
		t.Errorf("expected ID='task-1', got '%s'", task.ID)
	}
	if task.Title != "Test Task" {
		t.Errorf("expected Title='Test Task', got '%s'", task.Title)
	}
	if task.Priority != 1 {
		t.Errorf("expected Priority=1, got %d", task.Priority)
	}
	if len(task.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(task.Tags))
	}
}

func TestTypedParser_ParseIntoWithMarkdown(t *testing.T) {
	parser := NewTypedParser(nil)

	text := `Here is the task:

` + "```json" + `
{
	"id": "task-2",
	"title": "Another Task",
	"priority": 2,
	"completed": true
}
` + "```" + `

That's the task.`

	var task TestTask
	err := parser.ParseInto(context.Background(), text, &task)
	if err != nil {
		t.Fatalf("ParseInto failed: %v", err)
	}

	if task.ID != "task-2" {
		t.Errorf("expected ID='task-2', got '%s'", task.ID)
	}
	if !task.Completed {
		t.Error("expected Completed=true")
	}
}

func TestGenerateSchema(t *testing.T) {
	schema, err := GenerateSchema(TestTask{})
	if err != nil {
		t.Fatalf("GenerateSchema failed: %v", err)
	}

	if schema.Type != "object" {
		t.Errorf("expected type='object', got '%s'", schema.Type)
	}

	if len(schema.Properties) == 0 {
		t.Error("expected properties to be generated")
	}

	// 检查必填字段
	if len(schema.Required) != 2 {
		t.Errorf("expected 2 required fields, got %d", len(schema.Required))
	}

	// 检查字段类型
	if idProp, exists := schema.Properties["id"]; exists {
		if idProp.Type != "string" {
			t.Errorf("expected id type='string', got '%s'", idProp.Type)
		}
	} else {
		t.Error("id property not found")
	}

	if priorityProp, exists := schema.Properties["priority"]; exists {
		if priorityProp.Type != "integer" {
			t.Errorf("expected priority type='integer', got '%s'", priorityProp.Type)
		}
	}
}

func TestParseTyped(t *testing.T) {
	spec := TypedOutputSpec{
		StructType:     &TestTask{},
		RequiredFields: []string{"id", "title"},
		Strict:         true,
	}

	jsonText := `{
		"id": "task-3",
		"title": "Typed Task",
		"priority": 3
	}`

	result, err := ParseTyped(context.Background(), jsonText, spec)
	if err != nil {
		t.Fatalf("ParseTyped failed: %v", err)
	}

	if !result.Success {
		t.Error("expected success=true")
	}

	task, ok := result.Data.(*TestTask)
	if !ok {
		t.Fatal("expected data to be *TestTask")
	}

	if task.ID != "task-3" {
		t.Errorf("expected ID='task-3', got '%s'", task.ID)
	}
}

func TestParseTyped_MissingRequired(t *testing.T) {
	spec := TypedOutputSpec{
		StructType:     &TestTask{},
		RequiredFields: []string{"id", "title"},
		Strict:         false,
	}

	jsonText := `{
		"id": "task-4"
	}`

	result, err := ParseTyped(context.Background(), jsonText, spec)
	if err != nil {
		t.Fatalf("ParseTyped failed: %v", err)
	}

	if len(result.MissingFields) == 0 {
		t.Error("expected missing fields to be detected")
	}
}

func TestParseTyped_WithValidation(t *testing.T) {
	spec := TypedOutputSpec{
		StructType: &TestTask{},
		Strict:     true,
		CustomValidation: func(data interface{}) error {
			task := data.(*TestTask)
			if task.Priority < 0 || task.Priority > 5 {
				return &ValidationError{
					Field:   "priority",
					Message: "priority must be between 0 and 5",
				}
			}
			return nil
		},
	}

	// 有效的优先级
	validJSON := `{
		"id": "task-5",
		"title": "Valid Task",
		"priority": 3
	}`

	result, err := ParseTyped(context.Background(), validJSON, spec)
	if err != nil {
		t.Fatalf("ParseTyped failed for valid data: %v", err)
	}

	if !result.Success {
		t.Error("expected success=true for valid data")
	}

	// 无效的优先级
	invalidJSON := `{
		"id": "task-6",
		"title": "Invalid Task",
		"priority": 10
	}`

	_, err = ParseTyped(context.Background(), invalidJSON, spec)
	if err == nil {
		t.Error("expected error for invalid priority")
	}
}

func TestTypedParser_NestedStruct(t *testing.T) {
	parser := NewTypedParser(nil)

	jsonText := `{
		"tasks": [
			{
				"id": "task-1",
				"title": "First Task",
				"priority": 1
			},
			{
				"id": "task-2",
				"title": "Second Task",
				"priority": 2
			}
		],
		"total": 2
	}`

	var taskList TestTaskList
	err := parser.ParseInto(context.Background(), jsonText, &taskList)
	if err != nil {
		t.Fatalf("ParseInto failed: %v", err)
	}

	if taskList.Total != 2 {
		t.Errorf("expected Total=2, got %d", taskList.Total)
	}

	if len(taskList.Tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(taskList.Tasks))
	}

	if taskList.Tasks[0].ID != "task-1" {
		t.Errorf("expected first task ID='task-1', got '%s'", taskList.Tasks[0].ID)
	}
}

// ValidationError 自定义验证错误
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
