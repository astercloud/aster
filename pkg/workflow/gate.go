package workflow

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// GateType 决策门类型
type GateType string

const (
	GateTypeQualityCheck GateType = "quality_check"
	GateTypeStressTest   GateType = "stress_test"
	GateTypeApproval     GateType = "approval"
	GateTypeCustom       GateType = "custom"
)

// GateStatus 决策门状态
type GateStatus string

const (
	GateStatusGreen  GateStatus = "green"  // 通过
	GateStatusYellow GateStatus = "yellow" // 警告 (通过但有建议)
	GateStatusRed    GateStatus = "red"    // 失败
)

// Gate 决策门接口
type Gate interface {
	ID() string
	Name() string
	Type() GateType
	Description() string

	// 评估决策门
	Evaluate(ctx context.Context, input *GateInput) *GateResult

	// 配置
	Config() *GateConfig
}

// GateInput 决策门输入
type GateInput struct {
	StepOutput      *StepOutput
	PreviousOutputs map[string]*StepOutput
	Project         any // 项目上下文
	Constraints     any // 约束条件
	SessionState    map[string]any
	Metadata        map[string]any
}

// GateResult 决策门结果
type GateResult struct {
	GateID      string
	GateName    string
	GateType    GateType
	Status      GateStatus
	Passed      bool
	Score       float64  // 0-100
	Reason      string   // 通过/失败理由
	Suggestions []string // 改进建议
	Details     map[string]any
	Timestamp   time.Time
	Duration    float64
}

// GateConfig 决策门配置
type GateConfig struct {
	ID          string
	Name        string
	Type        GateType
	Description string
	Enabled     bool
	FailOnError bool // 错误时是否视为失败
	Timeout     time.Duration
	Metadata    map[string]any
}

// ===== QualityCheckGate =====

type QualityCheckGate struct {
	id          string
	name        string
	description string
	config      *GateConfig
	evaluator   QualityEvaluator
}

// QualityEvaluator 质量评估器接口
type QualityEvaluator interface {
	Evaluate(ctx context.Context, output *StepOutput) (score float64, issues []string, err error)
}

func NewQualityCheckGate(name string, evaluator QualityEvaluator) *QualityCheckGate {
	return &QualityCheckGate{
		id:          generateID(),
		name:        name,
		description: "Quality check gate for step output validation",
		evaluator:   evaluator,
		config: &GateConfig{
			Name:        name,
			Type:        GateTypeQualityCheck,
			Enabled:     true,
			FailOnError: false,
			Timeout:     2 * time.Minute,
		},
	}
}

func (g *QualityCheckGate) ID() string          { return g.id }
func (g *QualityCheckGate) Name() string        { return g.name }
func (g *QualityCheckGate) Type() GateType      { return GateTypeQualityCheck }
func (g *QualityCheckGate) Description() string { return g.description }
func (g *QualityCheckGate) Config() *GateConfig { return g.config }

func (g *QualityCheckGate) Evaluate(ctx context.Context, input *GateInput) *GateResult {
	startTime := time.Now()
	result := &GateResult{
		GateID:      g.id,
		GateName:    g.name,
		GateType:    GateTypeQualityCheck,
		Suggestions: make([]string, 0),
		Details:     make(map[string]any),
		Timestamp:   startTime,
	}

	if input == nil || input.StepOutput == nil {
		result.Status = GateStatusRed
		result.Passed = false
		result.Reason = "no step output to evaluate"
		result.Score = 0
		return result
	}

	// 创建超时上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, g.config.Timeout)
	defer cancel()

	// 执行评估
	score, issues, err := g.evaluator.Evaluate(timeoutCtx, input.StepOutput)
	result.Score = score
	result.Duration = time.Since(startTime).Seconds()

	if err != nil {
		result.Reason = fmt.Sprintf("evaluation error: %v", err)
		result.Status = GateStatusRed
		result.Passed = false
		result.Details["error"] = err.Error()
		return result
	}

	// 判断通过状态
	if score >= 80 {
		result.Status = GateStatusGreen
		result.Passed = true
		result.Reason = "Quality check passed"
	} else if score >= 60 {
		result.Status = GateStatusYellow
		result.Passed = true // 黄色仍然通过，但有建议
		result.Reason = "Quality check passed with warnings"
		result.Suggestions = issues
	} else {
		result.Status = GateStatusRed
		result.Passed = false
		result.Reason = "Quality check failed"
		result.Suggestions = issues
	}

	result.Details["issues_count"] = len(issues)
	return result
}

func (g *QualityCheckGate) WithDescription(desc string) *QualityCheckGate {
	g.description = desc
	return g
}

func (g *QualityCheckGate) WithTimeout(timeout time.Duration) *QualityCheckGate {
	g.config.Timeout = timeout
	return g
}

// ===== StressTestGate =====

type StressTestGate struct {
	id          string
	name        string
	description string
	config      *GateConfig
	tester      StressTester
}

// StressTester 压力测试器接口
type StressTester interface {
	Test(ctx context.Context, output *StepOutput) (score float64, issues []string, err error)
}

func NewStressTestGate(name string, tester StressTester) *StressTestGate {
	return &StressTestGate{
		id:          generateID(),
		name:        name,
		description: "Stress test gate for robustness validation",
		tester:      tester,
		config: &GateConfig{
			Name:        name,
			Type:        GateTypeStressTest,
			Enabled:     true,
			FailOnError: false,
			Timeout:     5 * time.Minute,
		},
	}
}

func (g *StressTestGate) ID() string          { return g.id }
func (g *StressTestGate) Name() string        { return g.name }
func (g *StressTestGate) Type() GateType      { return GateTypeStressTest }
func (g *StressTestGate) Description() string { return g.description }
func (g *StressTestGate) Config() *GateConfig { return g.config }

func (g *StressTestGate) Evaluate(ctx context.Context, input *GateInput) *GateResult {
	startTime := time.Now()
	result := &GateResult{
		GateID:      g.id,
		GateName:    g.name,
		GateType:    GateTypeStressTest,
		Suggestions: make([]string, 0),
		Details:     make(map[string]any),
		Timestamp:   startTime,
	}

	if input == nil || input.StepOutput == nil {
		result.Status = GateStatusRed
		result.Passed = false
		result.Reason = "no step output to test"
		result.Score = 0
		return result
	}

	// 创建超时上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, g.config.Timeout)
	defer cancel()

	// 执行测试
	score, issues, err := g.tester.Test(timeoutCtx, input.StepOutput)
	result.Score = score
	result.Duration = time.Since(startTime).Seconds()

	if err != nil {
		result.Reason = fmt.Sprintf("stress test error: %v", err)
		result.Status = GateStatusRed
		result.Passed = false
		result.Details["error"] = err.Error()
		return result
	}

	// 判断测试结果
	if score >= 90 {
		result.Status = GateStatusGreen
		result.Passed = true
		result.Reason = "Stress test passed"
	} else if score >= 70 {
		result.Status = GateStatusYellow
		result.Passed = true
		result.Reason = "Stress test passed with warnings"
		result.Suggestions = issues
	} else {
		result.Status = GateStatusRed
		result.Passed = false
		result.Reason = "Stress test failed"
		result.Suggestions = issues
	}

	result.Details["issues_count"] = len(issues)
	return result
}

func (g *StressTestGate) WithDescription(desc string) *StressTestGate {
	g.description = desc
	return g
}

func (g *StressTestGate) WithTimeout(timeout time.Duration) *StressTestGate {
	g.config.Timeout = timeout
	return g
}

// ===== ApprovalGate =====

type ApprovalGate struct {
	id          string
	name        string
	description string
	config      *GateConfig
	approver    Approver
}

// Approver 审核器接口
type Approver interface {
	Approve(ctx context.Context, output *StepOutput) (approved bool, reason string, err error)
}

func NewApprovalGate(name string, approver Approver) *ApprovalGate {
	return &ApprovalGate{
		id:          generateID(),
		name:        name,
		description: "Approval gate for manual review",
		approver:    approver,
		config: &GateConfig{
			Name:        name,
			Type:        GateTypeApproval,
			Enabled:     true,
			FailOnError: true,
			Timeout:     30 * time.Minute, // 人工审核需要更长时间
		},
	}
}

func (g *ApprovalGate) ID() string          { return g.id }
func (g *ApprovalGate) Name() string        { return g.name }
func (g *ApprovalGate) Type() GateType      { return GateTypeApproval }
func (g *ApprovalGate) Description() string { return g.description }
func (g *ApprovalGate) Config() *GateConfig { return g.config }

func (g *ApprovalGate) Evaluate(ctx context.Context, input *GateInput) *GateResult {
	startTime := time.Now()
	result := &GateResult{
		GateID:      g.id,
		GateName:    g.name,
		GateType:    GateTypeApproval,
		Suggestions: make([]string, 0),
		Details:     make(map[string]any),
		Timestamp:   startTime,
	}

	if input == nil || input.StepOutput == nil {
		result.Status = GateStatusRed
		result.Passed = false
		result.Reason = "no step output to approve"
		result.Score = 0
		return result
	}

	// 创建超时上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, g.config.Timeout)
	defer cancel()

	// 执行审核
	approved, reason, err := g.approver.Approve(timeoutCtx, input.StepOutput)
	result.Duration = time.Since(startTime).Seconds()

	if err != nil {
		result.Reason = fmt.Sprintf("approval error: %v", err)
		result.Status = GateStatusRed
		result.Passed = false
		result.Details["error"] = err.Error()
		return result
	}

	if approved {
		result.Status = GateStatusGreen
		result.Passed = true
		result.Score = 100
		result.Reason = reason
	} else {
		result.Status = GateStatusRed
		result.Passed = false
		result.Score = 0
		result.Reason = reason
	}

	return result
}

func (g *ApprovalGate) WithDescription(desc string) *ApprovalGate {
	g.description = desc
	return g
}

func (g *ApprovalGate) WithTimeout(timeout time.Duration) *ApprovalGate {
	g.config.Timeout = timeout
	return g
}

// ===== CustomGate =====

type CustomGate struct {
	id          string
	name        string
	description string
	config      *GateConfig
	evaluateFn  func(ctx context.Context, input *GateInput) *GateResult
}

func NewCustomGate(name string, evaluateFn func(ctx context.Context, input *GateInput) *GateResult) *CustomGate {
	return &CustomGate{
		id:          generateID(),
		name:        name,
		description: "Custom gate with user-defined evaluation logic",
		evaluateFn:  evaluateFn,
		config: &GateConfig{
			Name:        name,
			Type:        GateTypeCustom,
			Enabled:     true,
			FailOnError: false,
			Timeout:     5 * time.Minute,
		},
	}
}

func (g *CustomGate) ID() string          { return g.id }
func (g *CustomGate) Name() string        { return g.name }
func (g *CustomGate) Type() GateType      { return GateTypeCustom }
func (g *CustomGate) Description() string { return g.description }
func (g *CustomGate) Config() *GateConfig { return g.config }

func (g *CustomGate) Evaluate(ctx context.Context, input *GateInput) *GateResult {
	if g.evaluateFn == nil {
		return &GateResult{
			GateID:   g.id,
			GateName: g.name,
			Status:   GateStatusRed,
			Passed:   false,
			Reason:   "evaluation function not defined",
		}
	}

	return g.evaluateFn(ctx, input)
}

func (g *CustomGate) WithDescription(desc string) *CustomGate {
	g.description = desc
	return g
}

// GateRegistry 决策门注册表
type GateRegistry struct {
	gates map[string]Gate
}

func NewGateRegistry() *GateRegistry {
	return &GateRegistry{
		gates: make(map[string]Gate),
	}
}

func (gr *GateRegistry) Register(gate Gate) error {
	if gate == nil {
		return errors.New("gate cannot be nil")
	}
	gr.gates[gate.Name()] = gate
	return nil
}

func (gr *GateRegistry) Get(name string) (Gate, error) {
	gate, exists := gr.gates[name]
	if !exists {
		return nil, fmt.Errorf("gate not found: %s", name)
	}
	return gate, nil
}

func (gr *GateRegistry) List() []Gate {
	gates := make([]Gate, 0, len(gr.gates))
	for _, gate := range gr.gates {
		gates = append(gates, gate)
	}
	return gates
}

func (gr *GateRegistry) Unregister(name string) {
	delete(gr.gates, name)
}

// 辅助函数

func generateID() string {
	// 实现与 step.go 中的 uuid 生成一致
	return fmt.Sprintf("gate-%d", time.Now().UnixNano())
}
