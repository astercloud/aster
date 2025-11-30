package evals

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/types"
)

// LLMScorerConfig LLM评分器配置
type LLMScorerConfig struct {
	// Provider LLM提供商（用于评分）
	Provider provider.Provider
	// Name 评分器名称
	Name string
	// Prompt 评分提示词模板
	Prompt string
	// MaxTokens 最大token数（默认: 500）
	MaxTokens int
	// Temperature 温度（默认: 0，更确定性）
	Temperature float64
}

// LLMScorer 基于LLM的评分器基类
// 设计原理：使用LLM作为judge来评估文本质量
type LLMScorer struct {
	cfg LLMScorerConfig
}

// NewLLMScorer 创建LLM评分器
func NewLLMScorer(cfg LLMScorerConfig) *LLMScorer {
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = 500
	}
	if cfg.Temperature < 0 {
		cfg.Temperature = 0
	}
	return &LLMScorer{cfg: cfg}
}

// Score 实现Scorer接口
func (s *LLMScorer) Score(ctx context.Context, input *TextEvalInput) (*ScoreResult, error) {
	if input == nil {
		return &ScoreResult{
			Name:    s.cfg.Name,
			Value:   0,
			Details: make(map[string]any),
		}, nil
	}

	// 1. 构建评分提示词
	prompt := s.buildPrompt(input)

	// 2. 调用LLM获取评分
	messages := []types.Message{
		{
			Role:    types.RoleUser,
			Content: prompt,
		},
	}

	opts := &provider.StreamOptions{
		MaxTokens:   s.cfg.MaxTokens,
		Temperature: s.cfg.Temperature,
	}

	resp, err := s.cfg.Provider.Complete(ctx, messages, opts)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// 3. 解析LLM响应
	llmOutput := resp.Message.Content
	score, reason, details, err := parseScoreResponse(llmOutput)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	// 4. 返回结果
	if details == nil {
		details = make(map[string]any)
	}
	details["reason"] = reason
	details["llm_output"] = llmOutput

	return &ScoreResult{
		Name:    s.cfg.Name,
		Value:   score,
		Details: details,
	}, nil
}

// buildPrompt 构建评分提示词
func (s *LLMScorer) buildPrompt(input *TextEvalInput) string {
	prompt := s.cfg.Prompt

	// 替换占位符
	prompt = strings.ReplaceAll(prompt, "{{answer}}", input.Answer)
	prompt = strings.ReplaceAll(prompt, "{{reference}}", input.Reference)

	if len(input.Context) > 0 {
		contextStr := strings.Join(input.Context, "\n\n")
		prompt = strings.ReplaceAll(prompt, "{{context}}", contextStr)
	} else {
		prompt = strings.ReplaceAll(prompt, "{{context}}", "[无上下文]")
	}

	return prompt
}

// parseScoreResponse 解析LLM的评分响应
// 期望格式: JSON {"score": 0.85, "reason": "..."} 或 纯文本中包含 "Score: 0.85"
func parseScoreResponse(output string) (score float64, reason string, details map[string]any, err error) {
	output = strings.TrimSpace(output)

	// 尝试1: 解析JSON
	if strings.HasPrefix(output, "{") {
		var jsonResp struct {
			Score   float64        `json:"score"`
			Reason  string         `json:"reason"`
			Details map[string]any `json:"details"`
		}
		if err := json.Unmarshal([]byte(output), &jsonResp); err == nil {
			return jsonResp.Score, jsonResp.Reason, jsonResp.Details, nil
		}
	}

	// 尝试2: 使用正则提取 "Score: X.XX" 或 "score: X.XX"
	scoreRegex := regexp.MustCompile(`(?i)score[:\s]+([0-9]*\.?[0-9]+)`)
	matches := scoreRegex.FindStringSubmatch(output)
	if len(matches) >= 2 {
		if parsedScore, err := strconv.ParseFloat(matches[1], 64); err == nil {
			// 提取reason（通常在 "Reason:" 之后）
			reasonRegex := regexp.MustCompile(`(?i)reason[:\s]+(.+)`)
			reasonMatches := reasonRegex.FindStringSubmatch(output)
			extractedReason := ""
			if len(reasonMatches) >= 2 {
				extractedReason = strings.TrimSpace(reasonMatches[1])
			} else {
				extractedReason = output
			}

			return parsedScore, extractedReason, nil, nil
		}
	}

	// 尝试3: 查找任何数字（0-1之间）
	numberRegex := regexp.MustCompile(`([0-9]*\.?[0-9]+)`)
	matches = numberRegex.FindStringSubmatch(output)
	if len(matches) >= 2 {
		if parsedScore, err := strconv.ParseFloat(matches[1], 64); err == nil {
			if parsedScore >= 0 && parsedScore <= 1 {
				return parsedScore, output, nil, nil
			}
		}
	}

	return 0, "", nil, fmt.Errorf("could not parse score from output: %s", output)
}
