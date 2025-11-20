package evals

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// BatchTestCase 批量评估的单个测试用例
type BatchTestCase struct {
	// ID 测试用例ID
	ID string `json:"id"`
	// Input 评估输入
	Input *TextEvalInput `json:"input"`
	// Metadata 可选的元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// BatchResult 单个测试用例的评估结果
type BatchResult struct {
	// TestCaseID 测试用例ID
	TestCaseID string `json:"test_case_id"`
	// Scores 所有评分器的结果
	Scores []*ScoreResult `json:"scores"`
	// Duration 执行时间
	Duration time.Duration `json:"duration"`
	// Error 错误信息（如果有）
	Error string `json:"error,omitempty"`
	// Metadata 测试用例的元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// BatchEvalResult 批量评估的汇总结果
type BatchEvalResult struct {
	// Results 所有测试用例的结果
	Results []*BatchResult `json:"results"`
	// Summary 汇总统计
	Summary *BatchSummary `json:"summary"`
	// TotalDuration 总执行时间
	TotalDuration time.Duration `json:"total_duration"`
}

// BatchSummary 批量评估的汇总统计
type BatchSummary struct {
	// TotalCases 总测试用例数
	TotalCases int `json:"total_cases"`
	// SuccessfulCases 成功的用例数
	SuccessfulCases int `json:"successful_cases"`
	// FailedCases 失败的用例数
	FailedCases int `json:"failed_cases"`
	// AverageScores 各评分器的平均分
	AverageScores map[string]float64 `json:"average_scores"`
	// AverageDuration 平均执行时间
	AverageDuration time.Duration `json:"average_duration"`
}

// BatchConfig 批量评估配置
type BatchConfig struct {
	// TestCases 测试用例列表
	TestCases []*BatchTestCase
	// Scorers 评分器列表
	Scorers []Scorer
	// Concurrency 并发数（默认: 1，顺序执行）
	Concurrency int
	// StopOnError 遇到错误时是否停止（默认: false）
	StopOnError bool
	// ProgressCallback 进度回调函数（可选）
	ProgressCallback func(completed, total int)
}

// RunBatch 批量运行评估
func RunBatch(ctx context.Context, cfg *BatchConfig) (*BatchEvalResult, error) {
	if len(cfg.TestCases) == 0 {
		return nil, fmt.Errorf("no test cases provided")
	}
	if len(cfg.Scorers) == 0 {
		return nil, fmt.Errorf("no scorers provided")
	}

	// 设置默认并发数
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 1
	}

	startTime := time.Now()
	results := make([]*BatchResult, len(cfg.TestCases))

	// 使用信号量控制并发
	sem := make(chan struct{}, cfg.Concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	completed := 0
	var firstErr error

	for i, testCase := range cfg.TestCases {
		// 检查是否应该停止
		mu.Lock()
		shouldStop := cfg.StopOnError && firstErr != nil
		mu.Unlock()
		if shouldStop {
			break
		}

		wg.Add(1)
		go func(index int, tc *BatchTestCase) {
			defer wg.Done()

			// 获取信号量
			sem <- struct{}{}
			defer func() { <-sem }()

			// 检查上下文是否已取消
			if ctx.Err() != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = ctx.Err()
				}
				mu.Unlock()
				return
			}

			// 运行单个测试用例
			result := runSingleTestCase(ctx, tc, cfg.Scorers)
			results[index] = result

			// 更新进度
			mu.Lock()
			completed++
			if result.Error != "" && firstErr == nil {
				firstErr = fmt.Errorf("test case %s failed: %s", tc.ID, result.Error)
			}
			if cfg.ProgressCallback != nil {
				cfg.ProgressCallback(completed, len(cfg.TestCases))
			}
			mu.Unlock()
		}(i, testCase)
	}

	wg.Wait()

	// 计算汇总统计
	summary := calculateSummary(results)

	return &BatchEvalResult{
		Results:       results,
		Summary:       summary,
		TotalDuration: time.Since(startTime),
	}, firstErr
}

// runSingleTestCase 运行单个测试用例的所有评分器
func runSingleTestCase(ctx context.Context, testCase *BatchTestCase, scorers []Scorer) *BatchResult {
	startTime := time.Now()

	result := &BatchResult{
		TestCaseID: testCase.ID,
		Scores:     make([]*ScoreResult, 0, len(scorers)),
		Metadata:   testCase.Metadata,
	}

	for _, scorer := range scorers {
		score, err := scorer.Score(ctx, testCase.Input)
		if err != nil {
			result.Error = fmt.Sprintf("scorer error: %v", err)
			break
		}
		result.Scores = append(result.Scores, score)
	}

	result.Duration = time.Since(startTime)
	return result
}

// calculateSummary 计算汇总统计
func calculateSummary(results []*BatchResult) *BatchSummary {
	summary := &BatchSummary{
		TotalCases:    len(results),
		AverageScores: make(map[string]float64),
	}

	scorerCounts := make(map[string]int)
	scorerSums := make(map[string]float64)
	totalDuration := time.Duration(0)

	for _, result := range results {
		if result.Error == "" {
			summary.SuccessfulCases++
			totalDuration += result.Duration

			for _, score := range result.Scores {
				scorerSums[score.Name] += score.Value
				scorerCounts[score.Name]++
			}
		} else {
			summary.FailedCases++
		}
	}

	// 计算平均分
	for name, sum := range scorerSums {
		count := scorerCounts[name]
		if count > 0 {
			summary.AverageScores[name] = sum / float64(count)
		}
	}

	// 计算平均执行时间
	if summary.SuccessfulCases > 0 {
		summary.AverageDuration = totalDuration / time.Duration(summary.SuccessfulCases)
	}

	return summary
}

// RunBatchSimple 简化版批量评估（顺序执行）
func RunBatchSimple(ctx context.Context, testCases []*BatchTestCase, scorers []Scorer) (*BatchEvalResult, error) {
	return RunBatch(ctx, &BatchConfig{
		TestCases:   testCases,
		Scorers:     scorers,
		Concurrency: 1,
		StopOnError: false,
	})
}

// RunBatchConcurrent 并发批量评估
func RunBatchConcurrent(ctx context.Context, testCases []*BatchTestCase, scorers []Scorer, concurrency int) (*BatchEvalResult, error) {
	return RunBatch(ctx, &BatchConfig{
		TestCases:   testCases,
		Scorers:     scorers,
		Concurrency: concurrency,
		StopOnError: false,
	})
}
