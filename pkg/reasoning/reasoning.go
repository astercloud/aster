package reasoning

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// StepStatus 推理步骤状态
type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusRunning   StepStatus = "running"
	StepStatusCompleted StepStatus = "completed"
	StepStatusFailed    StepStatus = "failed"
)

// NextAction 下一步行动
type NextAction string

const (
	NextActionContinue NextAction = "continue" // 继续推理
	NextActionComplete NextAction = "complete" // 完成推理
	NextActionRetry    NextAction = "retry"    // 重试当前步骤
)

// Step 推理步骤
type Step struct {
	ID         string     `json:"id"`
	Title      string     `json:"title"`
	Action     string     `json:"action"`
	Result     string     `json:"result"`
	Reasoning  string     `json:"reasoning"`
	Confidence float64    `json:"confidence"` // 0.0-1.0
	Status     StepStatus `json:"status"`
	NextAction NextAction `json:"next_action"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// Chain 推理链
type Chain struct {
	ID        string    `json:"id"`
	Steps     []Step    `json:"steps"`
	MinSteps  int       `json:"min_steps"`
	MaxSteps  int       `json:"max_steps"`
	Current   int       `json:"current"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ChainConfig 推理链配置
type ChainConfig struct {
	MinSteps      int     // 最小推理步数
	MaxSteps      int     // 最大推理步数
	MinConfidence float64 // 最小置信度阈值
}

// NewChain 创建推理链
func NewChain(config ChainConfig) *Chain {
	if config.MinSteps <= 0 {
		config.MinSteps = 1
	}
	if config.MaxSteps <= 0 {
		config.MaxSteps = 10
	}
	if config.MinConfidence <= 0 {
		config.MinConfidence = 0.7
	}

	return &Chain{
		ID:        generateChainID(),
		Steps:     make([]Step, 0),
		MinSteps:  config.MinSteps,
		MaxSteps:  config.MaxSteps,
		Current:   0,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// AddStep 添加推理步骤
func (c *Chain) AddStep(step Step) error {
	if len(c.Steps) >= c.MaxSteps {
		return fmt.Errorf("reached max steps: %d", c.MaxSteps)
	}

	step.ID = fmt.Sprintf("step-%d", len(c.Steps)+1)
	step.CreatedAt = time.Now()
	step.UpdatedAt = time.Now()

	c.Steps = append(c.Steps, step)
	c.Current = len(c.Steps) - 1
	c.UpdatedAt = time.Now()

	return nil
}

// UpdateStep 更新推理步骤
func (c *Chain) UpdateStep(stepID string, updates map[string]any) error {
	for i, step := range c.Steps {
		if step.ID == stepID {
			if result, ok := updates["result"].(string); ok {
				c.Steps[i].Result = result
			}
			if status, ok := updates["status"].(StepStatus); ok {
				c.Steps[i].Status = status
			}
			if confidence, ok := updates["confidence"].(float64); ok {
				c.Steps[i].Confidence = confidence
			}
			if nextAction, ok := updates["next_action"].(NextAction); ok {
				c.Steps[i].NextAction = nextAction
			}
			c.Steps[i].UpdatedAt = time.Now()
			c.UpdatedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("step not found: %s", stepID)
}

// GetCurrentStep 获取当前步骤
func (c *Chain) GetCurrentStep() *Step {
	if c.Current >= 0 && c.Current < len(c.Steps) {
		return &c.Steps[c.Current]
	}
	return nil
}

// ShouldContinue 判断是否应该继续推理
func (c *Chain) ShouldContinue() bool {
	if len(c.Steps) >= c.MaxSteps {
		return false
	}

	currentStep := c.GetCurrentStep()
	if currentStep == nil {
		return true
	}

	if currentStep.NextAction == NextActionComplete {
		return false
	}

	if len(c.Steps) < c.MinSteps {
		return true
	}

	return currentStep.NextAction == NextActionContinue
}

// Complete 完成推理链
func (c *Chain) Complete() {
	c.Status = "completed"
	c.UpdatedAt = time.Now()
}

// ToJSON 转换为 JSON
func (c *Chain) ToJSON() (string, error) {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON 从 JSON 解析
func FromJSON(data string) (*Chain, error) {
	var chain Chain
	if err := json.Unmarshal([]byte(data), &chain); err != nil {
		return nil, err
	}
	return &chain, nil
}

// Summary 生成推理链摘要
func (c *Chain) Summary() string {
	summary := fmt.Sprintf("Reasoning Chain (ID: %s)\n", c.ID)
	summary += fmt.Sprintf("Steps: %d/%d (min: %d, max: %d)\n", len(c.Steps), c.MaxSteps, c.MinSteps, c.MaxSteps)
	summary += fmt.Sprintf("Status: %s\n\n", c.Status)

	var summarySb187 strings.Builder
	for i, step := range c.Steps {
		summarySb187.WriteString(fmt.Sprintf("Step %d: %s\n", i+1, step.Title))
		summarySb187.WriteString(fmt.Sprintf("  Action: %s\n", step.Action))
		summarySb187.WriteString(fmt.Sprintf("  Confidence: %.2f\n", step.Confidence))
		summarySb187.WriteString(fmt.Sprintf("  Status: %s\n", step.Status))
		if step.Result != "" {
			summarySb187.WriteString(fmt.Sprintf("  Result: %s\n", truncate(step.Result, 100)))
		}
		summarySb187.WriteString("\n")
	}
	summary += summarySb187.String()

	return summary
}

// Parser 推理步骤解析器
type Parser interface {
	Parse(ctx context.Context, text string) ([]Step, error)
}

// JSONParser JSON 格式推理解析器
type JSONParser struct{}

// NewJSONParser 创建 JSON 解析器
func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

// Parse 解析推理步骤
func (p *JSONParser) Parse(ctx context.Context, text string) ([]Step, error) {
	var steps []Step
	if err := json.Unmarshal([]byte(text), &steps); err != nil {
		return nil, fmt.Errorf("parse reasoning steps: %w", err)
	}
	return steps, nil
}

// generateChainID 生成推理链 ID
func generateChainID() string {
	return fmt.Sprintf("chain-%d", time.Now().UnixNano())
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
