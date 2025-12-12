// execution-plan example demonstrates how to use ExecutionPlan functionality
// to generate, review, and execute multi-step plans with user approval.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/astercloud/aster/pkg/executionplan"
	"github.com/astercloud/aster/pkg/tools"
)

func main() {
	fmt.Println("=== Execution Plan Example ===")
	fmt.Println()

	// Example 1: Create a plan manually
	fmt.Println("--- Example 1: Manual Plan Creation ---")
	manualPlanExample()

	// Example 2: Execute a plan
	fmt.Println("\n--- Example 2: Plan Execution ---")
	executionExample()

	// Example 3: Plan with dependencies
	fmt.Println("\n--- Example 3: Plan with Dependencies ---")
	dependencyExample()
}

// manualPlanExample demonstrates manual plan creation
func manualPlanExample() {
	// Create a new execution plan
	plan := executionplan.NewExecutionPlan("Search for files and analyze content")

	// Add steps
	plan.AddStep("search_files", "Search for Go files in the project", map[string]any{
		"pattern": "*.go",
		"path":    "./pkg",
	})

	plan.AddStep("read_file", "Read the content of the main file", map[string]any{
		"file_path": "main.go",
	})

	plan.AddStep("analyze", "Analyze the code structure", map[string]any{
		"type": "structure",
	})

	// Print the plan
	fmt.Println(executionplan.FormatPlan(plan))

	// Get summary
	summary := plan.Summary()
	fmt.Printf("Plan Summary: %d steps, Status: %s\n", summary.TotalSteps, summary.Status)
}

// executionExample demonstrates plan execution with approval
func executionExample() {
	// Create a plan that requires approval
	plan := executionplan.NewExecutionPlan("File operations demo")
	plan.Options = &executionplan.ExecutionOptions{
		RequireApproval: true,
		StopOnError:     true,
	}

	// Add steps
	plan.AddStep("list_files", "List files in current directory", map[string]any{
		"path": ".",
	})

	plan.AddStep("create_file", "Create a test file", map[string]any{
		"path":    "test.txt",
		"content": "Hello, World!",
	})

	// Print initial state
	fmt.Printf("Plan Status: %s\n", plan.Status)
	fmt.Printf("Can Execute: %v\n", plan.CanExecute())
	fmt.Printf("Is Approved: %v\n", plan.IsApproved())

	// Simulate approval
	plan.Approve("user@example.com")
	fmt.Printf("\nAfter Approval:\n")
	fmt.Printf("Plan Status: %s\n", plan.Status)
	fmt.Printf("Can Execute: %v\n", plan.CanExecute())
	fmt.Printf("Approved By: %s\n", plan.ApprovedBy)

	// Create mock tools for execution demo
	mockTools := map[string]tools.Tool{
		"list_files":  &MockTool{name: "list_files", result: []string{"file1.go", "file2.go"}},
		"create_file": &MockTool{name: "create_file", result: "created"},
	}

	// Create executor
	executor := executionplan.NewExecutor(
		mockTools,
		executionplan.WithOnStepStart(func(p *executionplan.ExecutionPlan, s *executionplan.Step) {
			fmt.Printf("  Starting step %d: %s\n", s.Index+1, s.Description)
		}),
		executionplan.WithOnStepComplete(func(p *executionplan.ExecutionPlan, s *executionplan.Step) {
			fmt.Printf("  Completed step %d: %s (took %dms)\n", s.Index+1, s.Description, s.DurationMs)
		}),
	)

	// Execute the plan
	fmt.Println("\nExecuting plan...")
	ctx := context.Background()
	toolCtx := &tools.ToolContext{AgentID: "example-agent"}

	if err := executor.Execute(ctx, plan, toolCtx); err != nil {
		fmt.Printf("Execution failed: %v\n", err)
	} else {
		fmt.Printf("\nExecution completed!\n")
		fmt.Printf("Final Status: %s\n", plan.Status)
		fmt.Printf("Total Duration: %dms\n", plan.TotalDurationMs)

		// Print step results
		fmt.Println("\nStep Results:")
		for i, step := range plan.Steps {
			fmt.Printf("  %d. %s: %v\n", i+1, step.ToolName, step.Result)
		}
	}
}

// dependencyExample demonstrates plan with step dependencies
func dependencyExample() {
	plan := executionplan.NewExecutionPlan("Build and test project")

	// Step 1: Install dependencies
	step1 := plan.AddStep("install", "Install project dependencies", map[string]any{
		"command": "go mod download",
	})

	// Step 2: Build (depends on step 1)
	step2 := plan.AddStep("build", "Build the project", map[string]any{
		"command": "go build ./...",
	})
	step2.DependsOn = []string{step1.ID}

	// Step 3: Run tests (depends on step 2)
	step3 := plan.AddStep("test", "Run unit tests", map[string]any{
		"command": "go test ./...",
	})
	step3.DependsOn = []string{step2.ID}

	// Step 4: Run linter (depends on step 1, can run parallel with build/test)
	step4 := plan.AddStep("lint", "Run linter checks", map[string]any{
		"command": "golangci-lint run",
	})
	step4.DependsOn = []string{step1.ID}

	fmt.Println(executionplan.FormatPlan(plan))

	// Show dependency graph
	fmt.Println("Dependency Graph:")
	for _, step := range plan.Steps {
		deps := "none"
		if len(step.DependsOn) > 0 {
			deps = fmt.Sprintf("%v", step.DependsOn)
		}
		fmt.Printf("  %s (deps: %s)\n", step.ToolName, deps)
	}
}

// MockTool is a simple mock tool for demonstration
type MockTool struct {
	name   string
	result any
}

func (m *MockTool) Name() string        { return m.name }
func (m *MockTool) Description() string { return "Mock tool: " + m.name }
func (m *MockTool) InputSchema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}
func (m *MockTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	return m.result, nil
}
func (m *MockTool) Prompt() string { return "" }

func init() {
	// Check for required environment variables
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		fmt.Println("Note: ANTHROPIC_API_KEY not set. Some features may not work.")
	}
}
