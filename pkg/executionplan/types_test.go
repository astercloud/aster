package executionplan

import (
	"errors"
	"testing"
	"time"
)

func TestNewExecutionPlan(t *testing.T) {
	description := "Test plan description"
	plan := NewExecutionPlan(description)

	if plan == nil {
		t.Fatal("NewExecutionPlan returned nil")
	}

	if plan.Description != description {
		t.Errorf("expected description %q, got %q", description, plan.Description)
	}

	if plan.Status != StatusDraft {
		t.Errorf("expected status %v, got %v", StatusDraft, plan.Status)
	}

	if plan.ID == "" {
		t.Error("expected non-empty ID")
	}

	if len(plan.Steps) != 0 {
		t.Errorf("expected 0 steps, got %d", len(plan.Steps))
	}

	if plan.Options == nil {
		t.Error("expected non-nil Options")
	}

	if !plan.Options.RequireApproval {
		t.Error("expected RequireApproval to be true by default")
	}

	if !plan.Options.StopOnError {
		t.Error("expected StopOnError to be true by default")
	}
}

func TestAddStep(t *testing.T) {
	plan := NewExecutionPlan("Test plan")

	step := plan.AddStep("test_tool", "Test step description", map[string]any{
		"param1": "value1",
		"param2": 42,
	})

	if step == nil {
		t.Fatal("AddStep returned nil")
	}

	if len(plan.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(plan.Steps))
	}

	if step.ToolName != "test_tool" {
		t.Errorf("expected tool name %q, got %q", "test_tool", step.ToolName)
	}

	if step.Description != "Test step description" {
		t.Errorf("expected description %q, got %q", "Test step description", step.Description)
	}

	if step.Index != 0 {
		t.Errorf("expected index 0, got %d", step.Index)
	}

	if step.Status != StepStatusPending {
		t.Errorf("expected status %v, got %v", StepStatusPending, step.Status)
	}

	if step.Parameters["param1"] != "value1" {
		t.Errorf("expected param1 to be 'value1', got %v", step.Parameters["param1"])
	}

	// Add second step
	step2 := plan.AddStep("another_tool", "Another step", nil)
	if step2.Index != 1 {
		t.Errorf("expected second step index 1, got %d", step2.Index)
	}

	if len(plan.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(plan.Steps))
	}
}

func TestGetStep(t *testing.T) {
	plan := NewExecutionPlan("Test plan")
	plan.AddStep("tool1", "Step 1", nil)
	plan.AddStep("tool2", "Step 2", nil)
	plan.AddStep("tool3", "Step 3", nil)

	tests := []struct {
		name     string
		index    int
		expected string
		isNil    bool
	}{
		{"first step", 0, "tool1", false},
		{"second step", 1, "tool2", false},
		{"third step", 2, "tool3", false},
		{"negative index", -1, "", true},
		{"out of bounds", 10, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := plan.GetStep(tt.index)
			if tt.isNil {
				if step != nil {
					t.Errorf("expected nil, got %v", step)
				}
			} else {
				if step == nil {
					t.Fatal("expected non-nil step")
				}
				if step.ToolName != tt.expected {
					t.Errorf("expected tool %q, got %q", tt.expected, step.ToolName)
				}
			}
		})
	}
}

func TestGetCurrentStep(t *testing.T) {
	plan := NewExecutionPlan("Test plan")
	plan.AddStep("tool1", "Step 1", nil)
	plan.AddStep("tool2", "Step 2", nil)

	// Initially current step is 0
	step := plan.GetCurrentStep()
	if step == nil {
		t.Fatal("expected non-nil current step")
	}
	if step.ToolName != "tool1" {
		t.Errorf("expected tool1, got %q", step.ToolName)
	}

	// Change current step
	plan.CurrentStep = 1
	step = plan.GetCurrentStep()
	if step.ToolName != "tool2" {
		t.Errorf("expected tool2, got %q", step.ToolName)
	}
}

func TestIsCompleted(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected bool
	}{
		{"draft", StatusDraft, false},
		{"pending approval", StatusPendingApproval, false},
		{"approved", StatusApproved, false},
		{"executing", StatusExecuting, false},
		{"completed", StatusCompleted, true},
		{"failed", StatusFailed, true},
		{"canceled", StatusCancelled, true},
		{"partial", StatusPartial, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := NewExecutionPlan("Test")
			plan.Status = tt.status
			if plan.IsCompleted() != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, plan.IsCompleted())
			}
		})
	}
}

func TestIsApproved(t *testing.T) {
	tests := []struct {
		name         string
		userApproved bool
		autoApprove  bool
		expected     bool
	}{
		{"not approved", false, false, false},
		{"user approved", true, false, true},
		{"auto approve", false, true, true},
		{"both approved", true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := NewExecutionPlan("Test")
			plan.UserApproved = tt.userApproved
			plan.Options.AutoApprove = tt.autoApprove
			if plan.IsApproved() != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, plan.IsApproved())
			}
		})
	}
}

func TestCanExecute(t *testing.T) {
	tests := []struct {
		name            string
		status          Status
		userApproved    bool
		requireApproval bool
		autoApprove     bool
		expected        bool
	}{
		{"draft, not approved, requires approval", StatusDraft, false, true, false, false},
		{"draft, approved", StatusDraft, true, true, false, true},
		{"draft, auto approve", StatusDraft, false, true, true, true},
		{"draft, no approval required", StatusDraft, false, false, false, true},
		{"executing", StatusExecuting, true, true, false, false},
		{"completed", StatusCompleted, true, true, false, false},
		{"failed", StatusFailed, true, true, false, false},
		{"canceled", StatusCancelled, true, true, false, false},
		{"approved status, approved", StatusApproved, true, true, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := NewExecutionPlan("Test")
			plan.Status = tt.status
			plan.UserApproved = tt.userApproved
			plan.Options.RequireApproval = tt.requireApproval
			plan.Options.AutoApprove = tt.autoApprove
			if plan.CanExecute() != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, plan.CanExecute())
			}
		})
	}
}

func TestApprove(t *testing.T) {
	plan := NewExecutionPlan("Test plan")
	approvedBy := "user@example.com"

	plan.Approve(approvedBy)

	if !plan.UserApproved {
		t.Error("expected UserApproved to be true")
	}

	if plan.ApprovedBy != approvedBy {
		t.Errorf("expected ApprovedBy %q, got %q", approvedBy, plan.ApprovedBy)
	}

	if plan.ApprovedAt == nil {
		t.Error("expected ApprovedAt to be set")
	}

	if plan.Status != StatusApproved {
		t.Errorf("expected status %v, got %v", StatusApproved, plan.Status)
	}
}

func TestReject(t *testing.T) {
	plan := NewExecutionPlan("Test plan")
	rejectionNote := "Not suitable for execution"

	plan.Reject(rejectionNote)

	if plan.UserApproved {
		t.Error("expected UserApproved to be false")
	}

	if plan.RejectionNote != rejectionNote {
		t.Errorf("expected RejectionNote %q, got %q", rejectionNote, plan.RejectionNote)
	}

	if plan.Status != StatusCancelled {
		t.Errorf("expected status %v, got %v", StatusCancelled, plan.Status)
	}
}

func TestMarkStepStarted(t *testing.T) {
	plan := NewExecutionPlan("Test plan")
	plan.AddStep("tool1", "Step 1", nil)
	plan.AddStep("tool2", "Step 2", nil)

	plan.MarkStepStarted(0)

	step := plan.GetStep(0)
	if step.Status != StepStatusRunning {
		t.Errorf("expected status %v, got %v", StepStatusRunning, step.Status)
	}

	if step.StartedAt == nil {
		t.Error("expected StartedAt to be set")
	}

	if plan.CurrentStep != 0 {
		t.Errorf("expected CurrentStep 0, got %d", plan.CurrentStep)
	}

	// Mark second step
	plan.MarkStepStarted(1)
	if plan.CurrentStep != 1 {
		t.Errorf("expected CurrentStep 1, got %d", plan.CurrentStep)
	}

	// Invalid index should not panic
	plan.MarkStepStarted(-1)
	plan.MarkStepStarted(100)
}

func TestMarkStepCompleted(t *testing.T) {
	plan := NewExecutionPlan("Test plan")
	plan.AddStep("tool1", "Step 1", nil)

	plan.MarkStepStarted(0)
	time.Sleep(10 * time.Millisecond) // Ensure some duration
	plan.MarkStepCompleted(0, "success result")

	step := plan.GetStep(0)
	if step.Status != StepStatusCompleted {
		t.Errorf("expected status %v, got %v", StepStatusCompleted, step.Status)
	}

	if step.Result != "success result" {
		t.Errorf("expected result %q, got %v", "success result", step.Result)
	}

	if step.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}

	if step.DurationMs <= 0 {
		t.Errorf("expected positive duration, got %d", step.DurationMs)
	}

	// Invalid index should not panic
	plan.MarkStepCompleted(-1, nil)
	plan.MarkStepCompleted(100, nil)
}

func TestMarkStepFailed(t *testing.T) {
	plan := NewExecutionPlan("Test plan")
	plan.AddStep("tool1", "Step 1", nil)

	plan.MarkStepStarted(0)
	testErr := errors.New("test error")
	plan.MarkStepFailed(0, testErr)

	step := plan.GetStep(0)
	if step.Status != StepStatusFailed {
		t.Errorf("expected status %v, got %v", StepStatusFailed, step.Status)
	}

	if step.Error != testErr.Error() {
		t.Errorf("expected error %q, got %q", testErr.Error(), step.Error)
	}

	if step.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}

	// Invalid index should not panic
	plan.MarkStepFailed(-1, testErr)
	plan.MarkStepFailed(100, testErr)
}

func TestSummary(t *testing.T) {
	plan := NewExecutionPlan("Test plan")
	plan.AddStep("tool1", "Step 1", nil)
	plan.AddStep("tool2", "Step 2", nil)
	plan.AddStep("tool3", "Step 3", nil)
	plan.AddStep("tool4", "Step 4", nil)

	// Initial state: all pending
	summary := plan.Summary()
	if summary.TotalSteps != 4 {
		t.Errorf("expected TotalSteps 4, got %d", summary.TotalSteps)
	}
	if summary.Pending != 4 {
		t.Errorf("expected Pending 4, got %d", summary.Pending)
	}
	if summary.Progress != 0 {
		t.Errorf("expected Progress 0, got %f", summary.Progress)
	}

	// Mark some steps
	plan.Steps[0].Status = StepStatusCompleted
	plan.Steps[1].Status = StepStatusFailed
	plan.Steps[2].Status = StepStatusRunning
	// step 4 remains pending

	summary = plan.Summary()
	if summary.Completed != 1 {
		t.Errorf("expected Completed 1, got %d", summary.Completed)
	}
	if summary.Failed != 1 {
		t.Errorf("expected Failed 1, got %d", summary.Failed)
	}
	if summary.Running != 1 {
		t.Errorf("expected Running 1, got %d", summary.Running)
	}
	if summary.Pending != 1 {
		t.Errorf("expected Pending 1, got %d", summary.Pending)
	}
	if summary.Progress != 25 {
		t.Errorf("expected Progress 25, got %f", summary.Progress)
	}
}

func TestSummaryEmptyPlan(t *testing.T) {
	plan := NewExecutionPlan("Empty plan")
	summary := plan.Summary()

	if summary.TotalSteps != 0 {
		t.Errorf("expected TotalSteps 0, got %d", summary.TotalSteps)
	}

	// Division by zero protection - Progress should be 0 or NaN
	if summary.Progress != 0 && !isNaN(summary.Progress) {
		t.Errorf("expected Progress 0 or NaN for empty plan, got %f", summary.Progress)
	}
}

func isNaN(f float64) bool {
	return f != f
}

func TestGenerateIDs(t *testing.T) {
	// Test that IDs have expected prefixes
	plan := NewExecutionPlan("Test")
	if plan.ID == "" {
		t.Error("plan ID should not be empty")
	}
	if len(plan.ID) < 10 {
		t.Error("plan ID should be at least 10 characters")
	}
	if !containsPrefix(plan.ID, "plan_") {
		t.Errorf("plan ID should start with 'plan_', got %s", plan.ID)
	}

	step := plan.AddStep("tool", "step", nil)
	if step.ID == "" {
		t.Error("step ID should not be empty")
	}
	if len(step.ID) < 10 {
		t.Error("step ID should be at least 10 characters")
	}
	if !containsPrefix(step.ID, "step_") {
		t.Errorf("step ID should start with 'step_', got %s", step.ID)
	}

	// Test that multiple plans have different IDs (with some tolerance)
	time.Sleep(10 * time.Millisecond)
	plan2 := NewExecutionPlan("Test 2")
	if plan.ID == plan2.ID {
		t.Errorf("two plans should have different IDs: %s vs %s", plan.ID, plan2.ID)
	}
}

func containsPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func TestStepDependencies(t *testing.T) {
	plan := NewExecutionPlan("Test plan with dependencies")

	step1 := plan.AddStep("install", "Install dependencies", nil)
	step2 := plan.AddStep("build", "Build project", nil)
	step2.DependsOn = []string{step1.ID}

	step3 := plan.AddStep("test", "Run tests", nil)
	step3.DependsOn = []string{step2.ID}

	// Verify dependencies are set correctly
	if len(plan.Steps[1].DependsOn) != 1 {
		t.Errorf("expected 1 dependency, got %d", len(plan.Steps[1].DependsOn))
	}
	if plan.Steps[1].DependsOn[0] != step1.ID {
		t.Errorf("expected dependency on %s, got %s", step1.ID, plan.Steps[1].DependsOn[0])
	}

	if len(plan.Steps[2].DependsOn) != 1 {
		t.Errorf("expected 1 dependency, got %d", len(plan.Steps[2].DependsOn))
	}
	if plan.Steps[2].DependsOn[0] != step2.ID {
		t.Errorf("expected dependency on %s, got %s", step2.ID, plan.Steps[2].DependsOn[0])
	}
}

func TestRetryConfiguration(t *testing.T) {
	plan := NewExecutionPlan("Test plan")
	step := plan.AddStep("tool1", "Step 1", nil)

	// Set retry configuration
	step.MaxRetries = 3
	step.RetryDelayMs = 1000

	if plan.Steps[0].MaxRetries != 3 {
		t.Errorf("expected MaxRetries 3, got %d", plan.Steps[0].MaxRetries)
	}
	if plan.Steps[0].RetryDelayMs != 1000 {
		t.Errorf("expected RetryDelayMs 1000, got %d", plan.Steps[0].RetryDelayMs)
	}
}

func TestExecutionOptions(t *testing.T) {
	plan := NewExecutionPlan("Test plan")

	// Modify options
	plan.Options.AllowParallel = true
	plan.Options.MaxParallelSteps = 5
	plan.Options.StepTimeoutMs = 30000
	plan.Options.TotalTimeoutMs = 300000
	plan.Options.ApprovalTimeoutMs = 60000

	if !plan.Options.AllowParallel {
		t.Error("expected AllowParallel to be true")
	}
	if plan.Options.MaxParallelSteps != 5 {
		t.Errorf("expected MaxParallelSteps 5, got %d", plan.Options.MaxParallelSteps)
	}
	if plan.Options.StepTimeoutMs != 30000 {
		t.Errorf("expected StepTimeoutMs 30000, got %d", plan.Options.StepTimeoutMs)
	}
}

func TestPlanMetadata(t *testing.T) {
	plan := NewExecutionPlan("Test plan")

	plan.AgentID = "agent-123"
	plan.OrgID = "org-456"
	plan.TenantID = "tenant-789"
	plan.TaskID = "task-001"
	plan.Name = "My Test Plan"
	plan.Metadata = map[string]any{
		"priority": "high",
		"team":     "platform",
	}

	if plan.AgentID != "agent-123" {
		t.Errorf("expected AgentID agent-123, got %s", plan.AgentID)
	}
	if plan.OrgID != "org-456" {
		t.Errorf("expected OrgID org-456, got %s", plan.OrgID)
	}
	if plan.TenantID != "tenant-789" {
		t.Errorf("expected TenantID tenant-789, got %s", plan.TenantID)
	}
	if plan.Metadata["priority"] != "high" {
		t.Errorf("expected priority high, got %v", plan.Metadata["priority"])
	}
}
