package executionplan

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/astercloud/aster/pkg/tools"
)

// mockTool is a mock implementation of tools.Tool for testing
type mockTool struct {
	name        string
	description string
	result      any
	err         error
	delay       time.Duration
	execCount   int
	mu          sync.Mutex
}

func (m *mockTool) Name() string        { return m.name }
func (m *mockTool) Description() string { return m.description }
func (m *mockTool) InputSchema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}
func (m *mockTool) Prompt() string { return "" }
func (m *mockTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	m.mu.Lock()
	m.execCount++
	m.mu.Unlock()

	if m.delay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(m.delay):
		}
	}
	return m.result, m.err
}

func (m *mockTool) ExecutionCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.execCount
}

func newMockTool(name string, result any, err error) *mockTool {
	return &mockTool{
		name:        name,
		description: "Mock " + name,
		result:      result,
		err:         err,
	}
}

func newDelayedMockTool(name string, result any, delay time.Duration) *mockTool {
	return &mockTool{
		name:        name,
		description: "Mock " + name,
		result:      result,
		delay:       delay,
	}
}

func TestNewExecutor(t *testing.T) {
	toolMap := map[string]tools.Tool{
		"tool1": newMockTool("tool1", "result1", nil),
	}

	executor := NewExecutor(toolMap)
	if executor == nil {
		t.Fatal("NewExecutor returned nil")
	}

	if len(executor.tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(executor.tools))
	}
}

func TestExecutorWithCallbacks(t *testing.T) {
	toolMap := map[string]tools.Tool{
		"tool1": newMockTool("tool1", "result1", nil),
	}

	var stepStartCalled, stepCompleteCalled, planCompleteCalled bool
	var startedPlan, completedPlan *ExecutionPlan
	var startedStep, completedStep *Step

	executor := NewExecutor(
		toolMap,
		WithOnStepStart(func(p *ExecutionPlan, s *Step) {
			stepStartCalled = true
			startedPlan = p
			startedStep = s
		}),
		WithOnStepComplete(func(p *ExecutionPlan, s *Step) {
			stepCompleteCalled = true
			completedPlan = p
			completedStep = s
		}),
		WithOnPlanComplete(func(p *ExecutionPlan) {
			planCompleteCalled = true
		}),
	)

	plan := NewExecutionPlan("Test plan")
	plan.AddStep("tool1", "Test step", nil)
	plan.Options.RequireApproval = false

	ctx := context.Background()
	toolCtx := &tools.ToolContext{AgentID: "test-agent"}

	err := executor.Execute(ctx, plan, toolCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !stepStartCalled {
		t.Error("OnStepStart callback not called")
	}
	if !stepCompleteCalled {
		t.Error("OnStepComplete callback not called")
	}
	if !planCompleteCalled {
		t.Error("OnPlanComplete callback not called")
	}
	if startedPlan == nil || startedStep == nil {
		t.Error("start callback received nil plan or step")
	}
	if completedPlan == nil || completedStep == nil {
		t.Error("complete callback received nil plan or step")
	}
}

func TestExecuteSequential(t *testing.T) {
	tool1 := newMockTool("tool1", "result1", nil)
	tool2 := newMockTool("tool2", "result2", nil)
	tool3 := newMockTool("tool3", "result3", nil)

	toolMap := map[string]tools.Tool{
		"tool1": tool1,
		"tool2": tool2,
		"tool3": tool3,
	}

	executor := NewExecutor(toolMap)

	plan := NewExecutionPlan("Sequential test")
	plan.AddStep("tool1", "Step 1", nil)
	plan.AddStep("tool2", "Step 2", nil)
	plan.AddStep("tool3", "Step 3", nil)
	plan.Options.RequireApproval = false

	ctx := context.Background()
	toolCtx := &tools.ToolContext{AgentID: "test-agent"}

	err := executor.Execute(ctx, plan, toolCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.Status != StatusCompleted {
		t.Errorf("expected status %v, got %v", StatusCompleted, plan.Status)
	}

	for i, step := range plan.Steps {
		if step.Status != StepStatusCompleted {
			t.Errorf("step %d: expected status %v, got %v", i, StepStatusCompleted, step.Status)
		}
	}

	if tool1.ExecutionCount() != 1 {
		t.Errorf("tool1 executed %d times, expected 1", tool1.ExecutionCount())
	}
	if tool2.ExecutionCount() != 1 {
		t.Errorf("tool2 executed %d times, expected 1", tool2.ExecutionCount())
	}
	if tool3.ExecutionCount() != 1 {
		t.Errorf("tool3 executed %d times, expected 1", tool3.ExecutionCount())
	}
}

func TestExecuteWithError(t *testing.T) {
	testErr := errors.New("test error")
	tool1 := newMockTool("tool1", "result1", nil)
	tool2 := newMockTool("tool2", nil, testErr)
	tool3 := newMockTool("tool3", "result3", nil)

	toolMap := map[string]tools.Tool{
		"tool1": tool1,
		"tool2": tool2,
		"tool3": tool3,
	}

	executor := NewExecutor(toolMap)

	plan := NewExecutionPlan("Error test")
	plan.AddStep("tool1", "Step 1", nil)
	plan.AddStep("tool2", "Step 2", nil)
	plan.AddStep("tool3", "Step 3", nil)
	plan.Options.RequireApproval = false
	plan.Options.StopOnError = true

	ctx := context.Background()
	toolCtx := &tools.ToolContext{AgentID: "test-agent"}

	err := executor.Execute(ctx, plan, toolCtx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if plan.Steps[0].Status != StepStatusCompleted {
		t.Errorf("step 0: expected %v, got %v", StepStatusCompleted, plan.Steps[0].Status)
	}
	if plan.Steps[1].Status != StepStatusFailed {
		t.Errorf("step 1: expected %v, got %v", StepStatusFailed, plan.Steps[1].Status)
	}
	if plan.Steps[2].Status != StepStatusSkipped {
		t.Errorf("step 2: expected %v, got %v", StepStatusSkipped, plan.Steps[2].Status)
	}

	if tool3.ExecutionCount() != 0 {
		t.Errorf("tool3 should not have been executed")
	}
}

func TestExecuteContinueOnError(t *testing.T) {
	testErr := errors.New("test error")
	tool1 := newMockTool("tool1", "result1", nil)
	tool2 := newMockTool("tool2", nil, testErr)
	tool3 := newMockTool("tool3", "result3", nil)

	toolMap := map[string]tools.Tool{
		"tool1": tool1,
		"tool2": tool2,
		"tool3": tool3,
	}

	executor := NewExecutor(toolMap)

	plan := NewExecutionPlan("Continue on error test")
	plan.AddStep("tool1", "Step 1", nil)
	plan.AddStep("tool2", "Step 2", nil)
	plan.AddStep("tool3", "Step 3", nil)
	plan.Options.RequireApproval = false
	plan.Options.StopOnError = false

	ctx := context.Background()
	toolCtx := &tools.ToolContext{AgentID: "test-agent"}

	err := executor.Execute(ctx, plan, toolCtx)
	if err == nil {
		t.Fatal("expected error to be returned")
	}

	if plan.Steps[0].Status != StepStatusCompleted {
		t.Errorf("step 0: expected %v, got %v", StepStatusCompleted, plan.Steps[0].Status)
	}
	if plan.Steps[1].Status != StepStatusFailed {
		t.Errorf("step 1: expected %v, got %v", StepStatusFailed, plan.Steps[1].Status)
	}
	if plan.Steps[2].Status != StepStatusCompleted {
		t.Errorf("step 2: expected %v, got %v", StepStatusCompleted, plan.Steps[2].Status)
	}

	// Tool3 should still execute
	if tool3.ExecutionCount() != 1 {
		t.Errorf("tool3 should have been executed once, got %d", tool3.ExecutionCount())
	}

	if plan.Status != StatusPartial {
		t.Errorf("expected status %v, got %v", StatusPartial, plan.Status)
	}
}

func TestExecuteParallel(t *testing.T) {
	// Use delays to verify parallel execution
	tool1 := newDelayedMockTool("tool1", "result1", 50*time.Millisecond)
	tool2 := newDelayedMockTool("tool2", "result2", 50*time.Millisecond)
	tool3 := newDelayedMockTool("tool3", "result3", 50*time.Millisecond)

	toolMap := map[string]tools.Tool{
		"tool1": tool1,
		"tool2": tool2,
		"tool3": tool3,
	}

	executor := NewExecutor(toolMap)

	plan := NewExecutionPlan("Parallel test")
	plan.AddStep("tool1", "Step 1", nil)
	plan.AddStep("tool2", "Step 2", nil)
	plan.AddStep("tool3", "Step 3", nil)
	plan.Options.RequireApproval = false
	plan.Options.AllowParallel = true
	plan.Options.MaxParallelSteps = 3

	ctx := context.Background()
	toolCtx := &tools.ToolContext{AgentID: "test-agent"}

	start := time.Now()
	err := executor.Execute(ctx, plan, toolCtx)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// If executed sequentially, it would take ~150ms
	// If executed in parallel, it should take ~50-70ms
	if duration > 120*time.Millisecond {
		t.Errorf("parallel execution took too long: %v (expected < 120ms)", duration)
	}

	if plan.Status != StatusCompleted {
		t.Errorf("expected status %v, got %v", StatusCompleted, plan.Status)
	}
}

func TestExecuteWithDependencies(t *testing.T) {
	tool1 := newMockTool("tool1", "result1", nil)
	tool2 := newMockTool("tool2", "result2", nil)
	tool3 := newMockTool("tool3", "result3", nil)

	toolMap := map[string]tools.Tool{
		"tool1": tool1,
		"tool2": tool2,
		"tool3": tool3,
	}

	executor := NewExecutor(toolMap)

	plan := NewExecutionPlan("Dependencies test")
	step1 := plan.AddStep("tool1", "Step 1", nil)
	step2 := plan.AddStep("tool2", "Step 2", nil)
	step2.DependsOn = []string{step1.ID}
	step3 := plan.AddStep("tool3", "Step 3", nil)
	step3.DependsOn = []string{step2.ID}
	plan.Options.RequireApproval = false

	ctx := context.Background()
	toolCtx := &tools.ToolContext{AgentID: "test-agent"}

	err := executor.Execute(ctx, plan, toolCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i, step := range plan.Steps {
		if step.Status != StepStatusCompleted {
			t.Errorf("step %d: expected %v, got %v", i, StepStatusCompleted, step.Status)
		}
	}
}

func TestExecuteParallelWithDependencies(t *testing.T) {
	tool1 := newDelayedMockTool("install", "installed", 30*time.Millisecond)
	tool2 := newDelayedMockTool("build", "built", 30*time.Millisecond)
	tool3 := newDelayedMockTool("test", "tested", 30*time.Millisecond)
	tool4 := newDelayedMockTool("lint", "linted", 30*time.Millisecond)

	toolMap := map[string]tools.Tool{
		"install": tool1,
		"build":   tool2,
		"test":    tool3,
		"lint":    tool4,
	}

	executor := NewExecutor(toolMap)

	plan := NewExecutionPlan("Parallel with deps test")

	// install -> build -> test
	// install -> lint (can run parallel with build/test)
	step1 := plan.AddStep("install", "Install deps", nil)
	step2 := plan.AddStep("build", "Build project", nil)
	step2.DependsOn = []string{step1.ID}
	step3 := plan.AddStep("test", "Run tests", nil)
	step3.DependsOn = []string{step2.ID}
	step4 := plan.AddStep("lint", "Run linter", nil)
	step4.DependsOn = []string{step1.ID}

	plan.Options.RequireApproval = false
	plan.Options.AllowParallel = true
	plan.Options.MaxParallelSteps = 3

	ctx := context.Background()
	toolCtx := &tools.ToolContext{AgentID: "test-agent"}

	err := executor.Execute(ctx, plan, toolCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.Status != StatusCompleted {
		t.Errorf("expected status %v, got %v", StatusCompleted, plan.Status)
	}

	for i, step := range plan.Steps {
		if step.Status != StepStatusCompleted {
			t.Errorf("step %d (%s): expected %v, got %v", i, step.ToolName, StepStatusCompleted, step.Status)
		}
	}
}

func TestExecuteRequiresApproval(t *testing.T) {
	toolMap := map[string]tools.Tool{
		"tool1": newMockTool("tool1", "result1", nil),
	}

	executor := NewExecutor(toolMap)

	plan := NewExecutionPlan("Requires approval")
	plan.AddStep("tool1", "Step 1", nil)
	plan.Options.RequireApproval = true

	ctx := context.Background()
	toolCtx := &tools.ToolContext{AgentID: "test-agent"}

	err := executor.Execute(ctx, plan, toolCtx)
	if err == nil {
		t.Fatal("expected error for unapproved plan")
	}

	if plan.Status != StatusDraft {
		t.Errorf("expected status %v, got %v", StatusDraft, plan.Status)
	}

	// Now approve and execute
	plan.Approve("test-user")
	err = executor.Execute(ctx, plan, toolCtx)
	if err != nil {
		t.Fatalf("unexpected error after approval: %v", err)
	}

	if plan.Status != StatusCompleted {
		t.Errorf("expected status %v, got %v", StatusCompleted, plan.Status)
	}
}

func TestExecuteContextCancellation(t *testing.T) {
	tool1 := newDelayedMockTool("tool1", "result1", 100*time.Millisecond)
	tool2 := newDelayedMockTool("tool2", "result2", 100*time.Millisecond)

	toolMap := map[string]tools.Tool{
		"tool1": tool1,
		"tool2": tool2,
	}

	executor := NewExecutor(toolMap)

	plan := NewExecutionPlan("Cancellation test")
	plan.AddStep("tool1", "Step 1", nil)
	plan.AddStep("tool2", "Step 2", nil)
	plan.Options.RequireApproval = false

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	toolCtx := &tools.ToolContext{AgentID: "test-agent"}

	err := executor.Execute(ctx, plan, toolCtx)
	if err == nil {
		t.Fatal("expected context error")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestExecuteToolNotFound(t *testing.T) {
	toolMap := map[string]tools.Tool{
		"tool1": newMockTool("tool1", "result1", nil),
	}

	executor := NewExecutor(toolMap)

	plan := NewExecutionPlan("Tool not found test")
	plan.AddStep("nonexistent_tool", "Step 1", nil)
	plan.Options.RequireApproval = false

	ctx := context.Background()
	toolCtx := &tools.ToolContext{AgentID: "test-agent"}

	err := executor.Execute(ctx, plan, toolCtx)
	if err == nil {
		t.Fatal("expected error for nonexistent tool")
	}

	if plan.Steps[0].Status != StepStatusFailed {
		t.Errorf("expected step status %v, got %v", StepStatusFailed, plan.Steps[0].Status)
	}
}

func TestExecuteAlreadyExecuting(t *testing.T) {
	toolMap := map[string]tools.Tool{
		"tool1": newMockTool("tool1", "result1", nil),
	}

	executor := NewExecutor(toolMap)

	plan := NewExecutionPlan("Already executing test")
	plan.AddStep("tool1", "Step 1", nil)
	plan.Status = StatusExecuting

	ctx := context.Background()
	toolCtx := &tools.ToolContext{AgentID: "test-agent"}

	err := executor.Execute(ctx, plan, toolCtx)
	if err == nil {
		t.Fatal("expected error for already executing plan")
	}
}

func TestExecuteAlreadyCompleted(t *testing.T) {
	toolMap := map[string]tools.Tool{
		"tool1": newMockTool("tool1", "result1", nil),
	}

	executor := NewExecutor(toolMap)

	plan := NewExecutionPlan("Already completed test")
	plan.AddStep("tool1", "Step 1", nil)
	plan.Status = StatusCompleted

	ctx := context.Background()
	toolCtx := &tools.ToolContext{AgentID: "test-agent"}

	err := executor.Execute(ctx, plan, toolCtx)
	if err == nil {
		t.Fatal("expected error for already completed plan")
	}
}

func TestResume(t *testing.T) {
	tool1 := newMockTool("tool1", "result1", nil)
	tool2 := newMockTool("tool2", "result2", nil)

	toolMap := map[string]tools.Tool{
		"tool1": tool1,
		"tool2": tool2,
	}

	executor := NewExecutor(toolMap)

	plan := NewExecutionPlan("Resume test")
	plan.AddStep("tool1", "Step 1", nil)
	plan.AddStep("tool2", "Step 2", nil)
	plan.Options.RequireApproval = false

	// Simulate partial execution
	plan.Steps[0].Status = StepStatusCompleted

	ctx := context.Background()
	toolCtx := &tools.ToolContext{AgentID: "test-agent"}

	err := executor.Resume(ctx, plan, toolCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.Status != StatusCompleted {
		t.Errorf("expected status %v, got %v", StatusCompleted, plan.Status)
	}

	// tool1 should not have been executed again
	if tool1.ExecutionCount() != 0 {
		t.Errorf("tool1 was executed %d times, expected 0", tool1.ExecutionCount())
	}

	// tool2 should have been executed
	if tool2.ExecutionCount() != 1 {
		t.Errorf("tool2 was executed %d times, expected 1", tool2.ExecutionCount())
	}
}

func TestResumeFromFailed(t *testing.T) {
	tool1 := &mockTool{
		name:   "tool1",
		result: "result1",
	}
	tool1.err = nil

	failingTool := &mockTool{
		name: "tool2",
	}
	// First call fails, second succeeds
	failingTool.result = "result2"

	toolMap := map[string]tools.Tool{
		"tool1": tool1,
		"tool2": failingTool,
	}

	executor := NewExecutor(toolMap)

	plan := NewExecutionPlan("Resume from failed test")
	plan.AddStep("tool1", "Step 1", nil)
	plan.AddStep("tool2", "Step 2", nil)
	plan.Options.RequireApproval = false

	// Simulate failed execution
	plan.Steps[0].Status = StepStatusCompleted
	plan.Steps[1].Status = StepStatusFailed
	plan.Steps[1].Error = "previous error"

	ctx := context.Background()
	toolCtx := &tools.ToolContext{AgentID: "test-agent"}

	err := executor.Resume(ctx, plan, toolCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.Status != StatusCompleted {
		t.Errorf("expected status %v, got %v", StatusCompleted, plan.Status)
	}

	if plan.Steps[1].Status != StepStatusCompleted {
		t.Errorf("step 1: expected %v, got %v", StepStatusCompleted, plan.Steps[1].Status)
	}
}

func TestCancel(t *testing.T) {
	toolMap := map[string]tools.Tool{}
	executor := NewExecutor(toolMap)

	plan := NewExecutionPlan("Cancel test")
	plan.AddStep("tool1", "Step 1", nil)
	plan.AddStep("tool2", "Step 2", nil)
	plan.AddStep("tool3", "Step 3", nil)
	plan.Status = StatusExecuting
	plan.Steps[0].Status = StepStatusCompleted
	plan.Steps[1].Status = StepStatusRunning

	executor.Cancel(plan, "User requested cancellation")

	if plan.Status != StatusCancelled {
		t.Errorf("expected status %v, got %v", StatusCancelled, plan.Status)
	}

	// Completed step should remain completed
	if plan.Steps[0].Status != StepStatusCompleted {
		t.Errorf("step 0: expected %v, got %v", StepStatusCompleted, plan.Steps[0].Status)
	}

	// Running and pending steps should be skipped
	if plan.Steps[1].Status != StepStatusSkipped {
		t.Errorf("step 1: expected %v, got %v", StepStatusSkipped, plan.Steps[1].Status)
	}
	if plan.Steps[2].Status != StepStatusSkipped {
		t.Errorf("step 2: expected %v, got %v", StepStatusSkipped, plan.Steps[2].Status)
	}

	if plan.Steps[1].Error != "User requested cancellation" {
		t.Errorf("step 1 error: expected 'User requested cancellation', got %q", plan.Steps[1].Error)
	}
}

func TestOnStepFailedCallback(t *testing.T) {
	testErr := errors.New("test error")
	tool1 := newMockTool("tool1", nil, testErr)

	toolMap := map[string]tools.Tool{
		"tool1": tool1,
	}

	var failedStep *Step
	var failedErr error

	executor := NewExecutor(
		toolMap,
		WithOnStepFailed(func(p *ExecutionPlan, s *Step, err error) {
			failedStep = s
			failedErr = err
		}),
	)

	plan := NewExecutionPlan("Failed callback test")
	plan.AddStep("tool1", "Step 1", nil)
	plan.Options.RequireApproval = false

	ctx := context.Background()
	toolCtx := &tools.ToolContext{AgentID: "test-agent"}

	_ = executor.Execute(ctx, plan, toolCtx)

	if failedStep == nil {
		t.Fatal("OnStepFailed callback not called")
	}
	if failedErr == nil || failedErr.Error() != testErr.Error() {
		t.Errorf("expected error %v, got %v", testErr, failedErr)
	}
}

func TestExecuteWithInputParameters(t *testing.T) {
	tool1 := newMockTool("tool1", "result", nil)

	toolMap := map[string]tools.Tool{
		"tool1": tool1,
	}

	executor := NewExecutor(toolMap)

	plan := NewExecutionPlan("Input params test")
	plan.AddStep("tool1", "Step 1", map[string]any{
		"file_path": "/path/to/file",
		"limit":     10,
	})
	plan.Options.RequireApproval = false

	ctx := context.Background()
	toolCtx := &tools.ToolContext{AgentID: "test-agent"}

	err := executor.Execute(ctx, plan, toolCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.Status != StatusCompleted {
		t.Errorf("expected status %v, got %v", StatusCompleted, plan.Status)
	}

	// Verify parameters are set
	if plan.Steps[0].Parameters["file_path"] != "/path/to/file" {
		t.Errorf("expected file_path parameter, got %v", plan.Steps[0].Parameters)
	}
}

func TestExecuteWithStepTimeout(t *testing.T) {
	// Create a tool that takes longer than the timeout
	tool1 := newDelayedMockTool("tool1", "result1", 200*time.Millisecond)

	toolMap := map[string]tools.Tool{
		"tool1": tool1,
	}

	executor := NewExecutor(toolMap)

	plan := NewExecutionPlan("Step timeout test")
	plan.AddStep("tool1", "Step 1", nil)
	plan.Options.RequireApproval = false
	plan.Options.StepTimeoutMs = 50 // 50ms timeout

	ctx := context.Background()
	toolCtx := &tools.ToolContext{AgentID: "test-agent"}

	err := executor.Execute(ctx, plan, toolCtx)
	if err == nil {
		t.Fatal("expected timeout error")
	}

	if plan.Steps[0].Status != StepStatusFailed {
		t.Errorf("expected step status %v, got %v", StepStatusFailed, plan.Steps[0].Status)
	}
}

func TestDependencyNotSatisfied(t *testing.T) {
	tool1 := newMockTool("tool1", nil, errors.New("tool1 error"))
	tool2 := newMockTool("tool2", "result2", nil)

	toolMap := map[string]tools.Tool{
		"tool1": tool1,
		"tool2": tool2,
	}

	executor := NewExecutor(toolMap)

	plan := NewExecutionPlan("Dependency not satisfied test")
	step1 := plan.AddStep("tool1", "Step 1", nil)
	step2 := plan.AddStep("tool2", "Step 2", nil)
	step2.DependsOn = []string{step1.ID}
	plan.Options.RequireApproval = false
	plan.Options.StopOnError = false

	ctx := context.Background()
	toolCtx := &tools.ToolContext{AgentID: "test-agent"}

	_ = executor.Execute(ctx, plan, toolCtx)

	// Step 2 should be skipped because its dependency failed
	if plan.Steps[1].Status != StepStatusSkipped {
		t.Errorf("step 1: expected %v, got %v", StepStatusSkipped, plan.Steps[1].Status)
	}
}
