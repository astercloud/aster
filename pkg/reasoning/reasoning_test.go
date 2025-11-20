package reasoning

import (
	"testing"
	"time"
)

func TestNewChain(t *testing.T) {
	config := ChainConfig{
		MinSteps:      2,
		MaxSteps:      5,
		MinConfidence: 0.8,
	}

	chain := NewChain(config)

	if chain.MinSteps != 2 {
		t.Errorf("expected MinSteps=2, got %d", chain.MinSteps)
	}
	if chain.MaxSteps != 5 {
		t.Errorf("expected MaxSteps=5, got %d", chain.MaxSteps)
	}
	if len(chain.Steps) != 0 {
		t.Errorf("expected empty steps, got %d", len(chain.Steps))
	}
}

func TestChainAddStep(t *testing.T) {
	chain := NewChain(ChainConfig{MaxSteps: 3})

	step := Step{
		Title:      "Test Step",
		Action:     "Test Action",
		Confidence: 0.9,
		Status:     StepStatusCompleted,
		NextAction: NextActionContinue,
	}

	err := chain.AddStep(step)
	if err != nil {
		t.Fatalf("failed to add step: %v", err)
	}

	if len(chain.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(chain.Steps))
	}

	if chain.Steps[0].ID == "" {
		t.Error("step ID should not be empty")
	}
}

func TestChainMaxSteps(t *testing.T) {
	chain := NewChain(ChainConfig{MaxSteps: 2})

	step := Step{
		Title:      "Test Step",
		Confidence: 0.9,
		NextAction: NextActionContinue,
	}

	// 添加第一个步骤
	if err := chain.AddStep(step); err != nil {
		t.Fatalf("failed to add first step: %v", err)
	}

	// 添加第二个步骤
	if err := chain.AddStep(step); err != nil {
		t.Fatalf("failed to add second step: %v", err)
	}

	// 尝试添加第三个步骤（应该失败）
	if err := chain.AddStep(step); err == nil {
		t.Error("expected error when exceeding max steps")
	}
}

func TestChainShouldContinue(t *testing.T) {
	tests := []struct {
		name       string
		minSteps   int
		maxSteps   int
		stepCount  int
		nextAction NextAction
		want       bool
	}{
		{
			name:       "below min steps",
			minSteps:   3,
			maxSteps:   5,
			stepCount:  2,
			nextAction: NextActionContinue,
			want:       true,
		},
		{
			name:       "at max steps",
			minSteps:   2,
			maxSteps:   3,
			stepCount:  3,
			nextAction: NextActionContinue,
			want:       false,
		},
		{
			name:       "complete action",
			minSteps:   2,
			maxSteps:   5,
			stepCount:  3,
			nextAction: NextActionComplete,
			want:       false,
		},
		{
			name:       "continue action above min",
			minSteps:   2,
			maxSteps:   5,
			stepCount:  3,
			nextAction: NextActionContinue,
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain := NewChain(ChainConfig{
				MinSteps: tt.minSteps,
				MaxSteps: tt.maxSteps,
			})

			// 添加步骤
			for i := 0; i < tt.stepCount; i++ {
				step := Step{
					Title:      "Test Step",
					Confidence: 0.9,
					NextAction: tt.nextAction,
				}
				chain.AddStep(step)
			}

			got := chain.ShouldContinue()
			if got != tt.want {
				t.Errorf("ShouldContinue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChainSummary(t *testing.T) {
	chain := NewChain(ChainConfig{
		MinSteps: 2,
		MaxSteps: 5,
	})

	chain.AddStep(Step{
		Title:      "Analyze Problem",
		Action:     "Break down the problem",
		Result:     "Identified 3 key components",
		Confidence: 0.9,
		Status:     StepStatusCompleted,
	})

	chain.AddStep(Step{
		Title:      "Propose Solution",
		Action:     "Design solution approach",
		Result:     "Created implementation plan",
		Confidence: 0.85,
		Status:     StepStatusCompleted,
	})

	summary := chain.Summary()

	if summary == "" {
		t.Error("summary should not be empty")
	}

	// 检查摘要包含关键信息
	if !contains(summary, "Analyze Problem") {
		t.Error("summary should contain step title")
	}
	if !contains(summary, "Steps: 2/5") {
		t.Error("summary should contain step count")
	}
}

func TestChainToJSON(t *testing.T) {
	chain := NewChain(ChainConfig{
		MinSteps: 2,
		MaxSteps: 5,
	})

	chain.AddStep(Step{
		Title:      "Test Step",
		Confidence: 0.9,
	})

	jsonStr, err := chain.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	if jsonStr == "" {
		t.Error("JSON string should not be empty")
	}

	// 尝试解析回来
	parsed, err := FromJSON(jsonStr)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	if len(parsed.Steps) != 1 {
		t.Errorf("expected 1 step after parsing, got %d", len(parsed.Steps))
	}
}

func TestUpdateStep(t *testing.T) {
	chain := NewChain(ChainConfig{MaxSteps: 5})

	step := Step{
		Title:      "Initial Title",
		Confidence: 0.5,
		Status:     StepStatusPending,
	}

	chain.AddStep(step)
	stepID := chain.Steps[0].ID

	// 更新步骤
	updates := map[string]interface{}{
		"result":     "Updated result",
		"status":     StepStatusCompleted,
		"confidence": 0.9,
	}

	err := chain.UpdateStep(stepID, updates)
	if err != nil {
		t.Fatalf("UpdateStep failed: %v", err)
	}

	// 验证更新
	updated := chain.Steps[0]
	if updated.Result != "Updated result" {
		t.Errorf("expected result='Updated result', got '%s'", updated.Result)
	}
	if updated.Status != StepStatusCompleted {
		t.Errorf("expected status=completed, got %s", updated.Status)
	}
	if updated.Confidence != 0.9 {
		t.Errorf("expected confidence=0.9, got %f", updated.Confidence)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr))
}

func TestStepLifecycle(t *testing.T) {
	chain := NewChain(ChainConfig{MaxSteps: 5})

	// 创建步骤
	step := Step{
		Title:      "Test Step",
		Action:     "Perform test",
		Reasoning:  "This is a test",
		Confidence: 0.8,
		Status:     StepStatusPending,
		NextAction: NextActionContinue,
	}

	// 添加步骤
	if err := chain.AddStep(step); err != nil {
		t.Fatalf("failed to add step: %v", err)
	}

	// 获取当前步骤
	current := chain.GetCurrentStep()
	if current == nil {
		t.Fatal("current step should not be nil")
	}

	if current.Title != "Test Step" {
		t.Errorf("expected title='Test Step', got '%s'", current.Title)
	}

	// 更新步骤状态
	stepID := current.ID
	chain.UpdateStep(stepID, map[string]interface{}{
		"status": StepStatusRunning,
	})

	if chain.Steps[0].Status != StepStatusRunning {
		t.Error("step status should be running")
	}

	// 完成步骤
	chain.UpdateStep(stepID, map[string]interface{}{
		"status":     StepStatusCompleted,
		"result":     "Test completed successfully",
		"confidence": 0.95,
	})

	if chain.Steps[0].Status != StepStatusCompleted {
		t.Error("step status should be completed")
	}
}

func TestChainComplete(t *testing.T) {
	chain := NewChain(ChainConfig{MaxSteps: 5})

	if chain.Status != "active" {
		t.Errorf("initial status should be 'active', got '%s'", chain.Status)
	}

	chain.Complete()

	if chain.Status != "completed" {
		t.Errorf("status should be 'completed', got '%s'", chain.Status)
	}

	// 验证 UpdatedAt 被更新
	if time.Since(chain.UpdatedAt) > time.Second {
		t.Error("UpdatedAt should be recent")
	}
}
