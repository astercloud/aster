package evals

import (
	"context"
	"testing"
	"time"
)

// TestRunBatchSimple 测试简单批量评估
func TestRunBatchSimple(t *testing.T) {
	mockProvider := &MockProvider{
		response: `{"score": 0.85, "reason": "测试评分"}`,
	}

	testCases := []*BatchTestCase{
		{
			ID: "case1",
			Input: &TextEvalInput{
				Answer:  "巴黎是法国的首都。",
				Context: []string{"法国的首都是巴黎。"},
			},
		},
		{
			ID: "case2",
			Input: &TextEvalInput{
				Answer:  "伦敦是英国的首都。",
				Context: []string{"英国的首都是伦敦。"},
			},
		},
	}

	scorers := []Scorer{
		NewKeywordCoverageScorer(KeywordCoverageConfig{
			Keywords:        []string{"首都"},
			CaseInsensitive: true,
		}),
		NewLexicalSimilarityScorer(LexicalSimilarityConfig{
			MinTokenLength: 2,
		}),
		NewFaithfulnessScorer(mockProvider),
	}

	result, err := RunBatchSimple(context.Background(), testCases, scorers)
	if err != nil {
		t.Fatalf("RunBatchSimple() error = %v", err)
	}

	// 验证结果数量
	if len(result.Results) != 2 {
		t.Errorf("RunBatchSimple() got %d results, want 2", len(result.Results))
	}

	// 验证每个测试用例有 3 个 scorer 的结果
	for i, r := range result.Results {
		if len(r.Scores) != 3 {
			t.Errorf("Result[%d] got %d scores, want 3", i, len(r.Scores))
		}
		if r.Error != "" {
			t.Errorf("Result[%d] has error: %s", i, r.Error)
		}
	}

	// 验证汇总
	if result.Summary == nil {
		t.Fatal("RunBatchSimple() summary is nil")
	}

	if result.Summary.TotalCases != 2 {
		t.Errorf("Summary.TotalCases = %d, want 2", result.Summary.TotalCases)
	}

	if result.Summary.SuccessfulCases != 2 {
		t.Errorf("Summary.SuccessfulCases = %d, want 2", result.Summary.SuccessfulCases)
	}

	if result.Summary.FailedCases != 0 {
		t.Errorf("Summary.FailedCases = %d, want 0", result.Summary.FailedCases)
	}

	// 验证平均分数
	if len(result.Summary.AverageScores) != 3 {
		t.Errorf("Summary.AverageScores has %d scorers, want 3", len(result.Summary.AverageScores))
	}
}

// TestRunBatchConcurrent 测试并发批量评估
func TestRunBatchConcurrent(t *testing.T) {
	mockProvider := &MockProvider{
		response: `{"score": 0.90, "reason": "并发测试"}`,
	}

	// 创建多个测试用例
	testCases := make([]*BatchTestCase, 10)
	for i := range 10 {
		testCases[i] = &BatchTestCase{
			ID: string(rune('A' + i)),
			Input: &TextEvalInput{
				Answer: "测试答案",
			},
		}
	}

	scorers := []Scorer{
		NewFaithfulnessScorer(mockProvider),
		NewHallucinationScorer(mockProvider),
	}

	// 测试并发度为 3
	result, err := RunBatchConcurrent(context.Background(), testCases, scorers, 3)
	if err != nil {
		t.Fatalf("RunBatchConcurrent() error = %v", err)
	}

	if len(result.Results) != 10 {
		t.Errorf("RunBatchConcurrent() got %d results, want 10", len(result.Results))
	}

	if result.Summary.SuccessfulCases != 10 {
		t.Errorf("RunBatchConcurrent() SuccessfulCases = %d, want 10", result.Summary.SuccessfulCases)
	}
}

// TestRunBatch_ProgressCallback 测试进度回调
func TestRunBatch_ProgressCallback(t *testing.T) {
	mockProvider := &MockProvider{
		response: `{"score": 0.75, "reason": "进度测试"}`,
	}

	testCases := make([]*BatchTestCase, 5)
	for i := range 5 {
		testCases[i] = &BatchTestCase{
			ID:    string(rune('A' + i)),
			Input: &TextEvalInput{Answer: "测试"},
		}
	}

	scorers := []Scorer{
		NewFaithfulnessScorer(mockProvider),
	}

	var progressCalls int
	var lastCompleted, lastTotal int

	cfg := &BatchConfig{
		TestCases:   testCases,
		Scorers:     scorers,
		Concurrency: 2,
		ProgressCallback: func(completed, total int) {
			progressCalls++
			lastCompleted = completed
			lastTotal = total
		},
	}

	result, err := RunBatch(context.Background(), cfg)
	if err != nil {
		t.Fatalf("RunBatch() error = %v", err)
	}

	// 验证进度回调被调用
	if progressCalls == 0 {
		t.Error("ProgressCallback was never called")
	}

	if lastCompleted != 5 {
		t.Errorf("Last completed = %d, want 5", lastCompleted)
	}

	if lastTotal != 5 {
		t.Errorf("Last total = %d, want 5", lastTotal)
	}

	if result.Summary.SuccessfulCases != 5 {
		t.Errorf("SuccessfulCases = %d, want 5", result.Summary.SuccessfulCases)
	}
}

// TestRunBatch_StopOnError 测试遇错停止
func TestRunBatch_StopOnError(t *testing.T) {
	// 创建一个会失败的 Scorer
	failingScorer := &failingTestScorer{}

	testCases := make([]*BatchTestCase, 5)
	for i := range 5 {
		testCases[i] = &BatchTestCase{
			ID:    string(rune('A' + i)),
			Input: &TextEvalInput{Answer: "测试"},
		}
	}

	cfg := &BatchConfig{
		TestCases:   testCases,
		Scorers:     []Scorer{failingScorer},
		Concurrency: 1,
		StopOnError: true,
	}

	_, err := RunBatch(context.Background(), cfg)

	// 应该返回错误
	if err == nil {
		t.Error("RunBatch() expected error with StopOnError=true")
	}
}

// TestRunBatch_EmptyTestCases 测试空测试用例
func TestRunBatch_EmptyTestCases(t *testing.T) {
	mockProvider := &MockProvider{response: `{"score": 0.5}`}

	cfg := &BatchConfig{
		TestCases:   []*BatchTestCase{},
		Scorers:     []Scorer{NewFaithfulnessScorer(mockProvider)},
		Concurrency: 1,
	}

	_, err := RunBatch(context.Background(), cfg)
	if err == nil {
		t.Error("RunBatch() should return error for empty test cases")
	}
}

// TestRunBatch_EmptyScorers 测试空评分器
func TestRunBatch_EmptyScorers(t *testing.T) {
	testCases := []*BatchTestCase{
		{ID: "case1", Input: &TextEvalInput{Answer: "测试"}},
	}

	cfg := &BatchConfig{
		TestCases:   testCases,
		Scorers:     []Scorer{},
		Concurrency: 1,
	}

	_, err := RunBatch(context.Background(), cfg)
	if err == nil {
		t.Error("RunBatch() should return error for empty scorers")
	}
}

// TestRunBatch_WithMetadata 测试带元数据的批量评估
func TestRunBatch_WithMetadata(t *testing.T) {
	mockProvider := &MockProvider{
		response: `{"score": 0.80, "reason": "元数据测试"}`,
	}

	testCases := []*BatchTestCase{
		{
			ID: "case1",
			Input: &TextEvalInput{
				Answer: "测试答案",
			},
			Metadata: map[string]any{
				"source":   "test",
				"priority": 1,
			},
		},
	}

	scorers := []Scorer{
		NewFaithfulnessScorer(mockProvider),
	}

	result, err := RunBatchSimple(context.Background(), testCases, scorers)
	if err != nil {
		t.Fatalf("RunBatchSimple() error = %v", err)
	}

	// 验证元数据被保留
	if result.Results[0].Metadata == nil {
		t.Error("Result metadata is nil")
	}

	if result.Results[0].Metadata["source"] != "test" {
		t.Errorf("Metadata source = %v, want test", result.Results[0].Metadata["source"])
	}
}

// TestCalculateSummary 测试汇总计算
func TestCalculateSummary(t *testing.T) {
	results := []*BatchResult{
		{
			TestCaseID: "case1",
			Scores: []*ScoreResult{
				{Name: "scorer1", Value: 0.8},
				{Name: "scorer2", Value: 0.9},
			},
			Duration: 100 * time.Millisecond,
			Error:    "",
		},
		{
			TestCaseID: "case2",
			Scores: []*ScoreResult{
				{Name: "scorer1", Value: 0.7},
				{Name: "scorer2", Value: 0.85},
			},
			Duration: 150 * time.Millisecond,
			Error:    "",
		},
		{
			TestCaseID: "case3",
			Scores:     []*ScoreResult{},
			Duration:   0,
			Error:      "test error",
		},
	}

	summary := calculateSummary(results)

	if summary.TotalCases != 3 {
		t.Errorf("TotalCases = %d, want 3", summary.TotalCases)
	}

	if summary.SuccessfulCases != 2 {
		t.Errorf("SuccessfulCases = %d, want 2", summary.SuccessfulCases)
	}

	if summary.FailedCases != 1 {
		t.Errorf("FailedCases = %d, want 1", summary.FailedCases)
	}

	// 验证平均分
	if avgScore1, ok := summary.AverageScores["scorer1"]; !ok || avgScore1 != 0.75 {
		t.Errorf("AverageScores[scorer1] = %v, want 0.75", avgScore1)
	}

	if avgScore2, ok := summary.AverageScores["scorer2"]; !ok || avgScore2 != 0.875 {
		t.Errorf("AverageScores[scorer2] = %v, want 0.875", avgScore2)
	}

	// 验证平均执行时间
	expectedAvgDuration := 125 * time.Millisecond
	if summary.AverageDuration != expectedAvgDuration {
		t.Errorf("AverageDuration = %v, want %v", summary.AverageDuration, expectedAvgDuration)
	}
}

// failingTestScorer 是一个总是失败的测试 Scorer
type failingTestScorer struct{}

func (f *failingTestScorer) Score(ctx context.Context, input *TextEvalInput) (*ScoreResult, error) {
	return nil, context.Canceled
}
