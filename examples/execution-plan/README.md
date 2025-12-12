# Execution Plan Example

This example demonstrates how to use the ExecutionPlan functionality in Aster to generate, review, and execute multi-step plans with user approval.

## Features Demonstrated

1. **Manual Plan Creation** - Create execution plans programmatically
2. **Plan Execution with Approval** - Execute plans after user approval
3. **Step Dependencies** - Define dependencies between steps for complex workflows

## Running the Example

```bash
go run main.go
```

## Key Concepts

### ExecutionPlan

An ExecutionPlan represents a series of tool executions that an agent intends to perform.

```go
plan := executionplan.NewExecutionPlan("Description of the plan")
```

### Steps

Each step represents a single tool execution:

```go
step := plan.AddStep("tool_name", "Step description", map[string]any{
    "param1": "value1",
    "param2": "value2",
})
```

### Step Dependencies

Steps can depend on other steps:

```go
step2.DependsOn = []string{step1.ID}
```

### Plan Lifecycle

1. **Draft** - Initial state when plan is created
2. **PendingApproval** - Waiting for user approval (if RequireApproval=true)
3. **Approved** - User has approved the plan
4. **Executing** - Plan is currently executing
5. **Completed** - All steps completed successfully
6. **Failed** - One or more steps failed
7. **Cancelled** - Plan was cancelled

### Approval Workflow

```go
// Check if approval is needed
if !plan.IsApproved() {
    // Show plan to user
    fmt.Println(executionplan.FormatPlan(plan))

    // Get user approval
    plan.Approve("user@example.com")
}

// Execute
executor.Execute(ctx, plan, toolCtx)
```

### Execution Options

```go
plan.Options = &executionplan.ExecutionOptions{
    RequireApproval:  true,  // Require user approval before execution
    AutoApprove:      false, // Don't auto-approve
    StopOnError:      true,  // Stop execution on first error
    AllowParallel:    true,  // Allow parallel execution of independent steps
    MaxParallelSteps: 3,     // Maximum 3 parallel steps
    StepTimeoutMs:    30000, // 30 second timeout per step
}
```

## Integration with Agent

```go
// Get the execution plan manager from agent
planMgr := agent.ExecutionPlan()

// Generate a plan from user request
plan, err := planMgr.GeneratePlan(ctx, "Help me search and analyze files", &agent.ExecutionPlanConfig{
    RequireApproval: true,
    StopOnError:     true,
})

// Display for approval
fmt.Println(planMgr.FormatCurrentPlan())

// Approve and execute
planMgr.ApprovePlan("user@example.com")
planMgr.ExecutePlan(ctx)

// Check progress
summary := planMgr.GetPlanSummary()
fmt.Printf("Progress: %.1f%%\n", summary.Progress)
```

## Callbacks

```go
executor := executionplan.NewExecutor(
    toolMap,
    executionplan.WithOnStepStart(func(plan *ExecutionPlan, step *Step) {
        fmt.Printf("Starting: %s\n", step.Description)
    }),
    executionplan.WithOnStepComplete(func(plan *ExecutionPlan, step *Step) {
        fmt.Printf("Completed: %s (took %dms)\n", step.Description, step.DurationMs)
    }),
    executionplan.WithOnStepFailed(func(plan *ExecutionPlan, step *Step, err error) {
        fmt.Printf("Failed: %s - %v\n", step.Description, err)
    }),
    executionplan.WithOnPlanComplete(func(plan *ExecutionPlan) {
        fmt.Printf("Plan completed with status: %s\n", plan.Status)
    }),
)
```
