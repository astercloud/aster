package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/session"
	"github.com/astercloud/aster/pkg/stream"
	"github.com/astercloud/aster/pkg/types"
)

// ConditionalAgent 条件分支Agent
type ConditionalAgent struct {
	name          string
	conditions    []BranchCondition
	defaultBranch *AgentRef
	evaluator     *ExpressionEvaluator
}

// BranchCondition 分支条件
type BranchCondition struct {
	Name      string         `json:"name"`
	Condition string         `json:"condition"` // 条件表达式
	Agent     *AgentRef      `json:"agent"`     // 分支Agent
	Weight    int            `json:"weight"`    // 权重（用于概率选择）
	Priority  int            `json:"priority"`  // 优先级
	Metadata  map[string]any `json:"metadata"`
}

// ConditionalConfig 条件Agent配置
type ConditionalConfig struct {
	Name        string            `json:"name"`
	Conditions  []BranchCondition `json:"conditions"`
	Default     *AgentRef         `json:"default,omitempty"`
	Variables   map[string]any    `json:"variables,omitempty"`
	EvalTimeout time.Duration     `json:"eval_timeout,omitempty"`
}

// NewConditionalAgent 创建条件Agent
func NewConditionalAgent(config ConditionalConfig) (*ConditionalAgent, error) {
	if config.Name == "" {
		return nil, fmt.Errorf("conditional agent name is required")
	}

	if len(config.Conditions) == 0 {
		return nil, fmt.Errorf("at least one condition is required")
	}

	// 按优先级排序条件
	conditions := make([]BranchCondition, len(config.Conditions))
	copy(conditions, config.Conditions)

	// 简单排序（优先级高的在前）
	for i := range len(conditions) {
		for j := i + 1; j < len(conditions); j++ {
			if conditions[i].Priority < conditions[j].Priority {
				conditions[i], conditions[j] = conditions[j], conditions[i]
			}
		}
	}

	return &ConditionalAgent{
		name:          config.Name,
		conditions:    conditions,
		defaultBranch: config.Default,
		evaluator:     NewExpressionEvaluator(config.Variables),
	}, nil
}

// Name 返回Agent名称
func (c *ConditionalAgent) Name() string {
	return c.name
}

// Execute 执行条件分支
func (c *ConditionalAgent) Execute(ctx context.Context, message string) *stream.Reader[*session.Event] {
	reader, writer := stream.Pipe[*session.Event](10)

	go func() {
		defer writer.Close()

		// 评估条件
		selectedBranch, err := c.evaluateConditions(ctx, message)
		if err != nil {
			writer.Send(nil, fmt.Errorf("failed to evaluate conditions: %w", err))
			return
		}

		if selectedBranch == nil {
			// 使用默认分支
			if c.defaultBranch == nil {
				writer.Send(nil, fmt.Errorf("no condition matched and no default branch provided"))
				return
			}

			writer.Send(&session.Event{
				ID:        generateEventID(),
				Timestamp: time.Now(),
				AgentID:   c.name,
				Author:    "system",
				Content: types.Message{
					Role:    types.MessageRoleAssistant,
					Content: fmt.Sprintf("No condition matched, using default branch: %s", c.defaultBranch.ID),
				},
				Metadata: map[string]any{
					"branch_type": "default",
					"agent_id":    c.defaultBranch.ID,
				},
			}, nil)

			// 执行默认Agent
			c.executeBranchAgent(ctx, writer, c.defaultBranch, message, "default")
			return
		}

		// 发送分支选择事件
		writer.Send(&session.Event{
			ID:        generateEventID(),
			Timestamp: time.Now(),
			AgentID:   c.name,
			Author:    "system",
			Content: types.Message{
				Role:    types.MessageRoleAssistant,
				Content: fmt.Sprintf("Condition matched: %s -> %s", selectedBranch.Condition, selectedBranch.Name),
			},
			Metadata: map[string]any{
				"branch_type": "conditional",
				"branch_name": selectedBranch.Name,
				"condition":   selectedBranch.Condition,
				"agent_id":    selectedBranch.Agent.ID,
			},
		}, nil)

		// 执行选中的分支Agent
		c.executeBranchAgent(ctx, writer, selectedBranch.Agent, message, selectedBranch.Name)
	}()

	return reader
}

// evaluateConditions 评估条件
func (c *ConditionalAgent) evaluateConditions(ctx context.Context, message string) (*BranchCondition, error) {
	// 更新变量
	if c.evaluator.variables == nil {
		c.evaluator.variables = make(map[string]any)
	}
	c.evaluator.variables["input"] = message
	c.evaluator.variables["message"] = message

	// 按优先级评估条件
	for _, condition := range c.conditions {
		result, err := c.evaluator.EvaluateBool(condition.Condition)
		if err != nil {
			// 记录错误但继续评估其他条件
			continue
		}

		if result {
			return &condition, nil
		}
	}

	return nil, nil
}

// executeBranchAgent 执行分支Agent
func (c *ConditionalAgent) executeBranchAgent(ctx context.Context, writer *stream.Writer[*session.Event], agentRef *AgentRef, message string, branchName string) {
	// TODO: 实现Agent创建和执行
	// 现在模拟执行
	for i := range 3 {
		event := &session.Event{
			ID:        generateEventID(),
			Timestamp: time.Now(),
			AgentID:   c.name,
			Author:    "system",
			Content: types.Message{
				Role:    types.MessageRoleAssistant,
				Content: fmt.Sprintf("Branch '%s' execution step %d: %s", branchName, i+1, message),
			},
			Metadata: map[string]any{
				"branch_name": branchName,
				"agent_id":    agentRef.ID,
				"step":        i + 1,
			},
		}

		if writer.Send(event, nil) {
			return
		}

		// 模拟延迟
		time.Sleep(time.Millisecond * 100)
	}
}

// ParallelConditionalAgent 并行条件Agent（同时评估多个条件）
type ParallelConditionalAgent struct {
	name         string
	conditions   []BranchCondition
	maxParallel  int
	timeout      time.Duration
	defaultAgent *AgentRef
	strategy     ParallelStrategy
	evaluator    *ExpressionEvaluator
}

// ParallelConditionalConfig 并行条件Agent配置
type ParallelConditionalConfig struct {
	Name        string            `json:"name"`
	Conditions  []BranchCondition `json:"conditions"`
	Default     *AgentRef         `json:"default,omitempty"`
	MaxParallel int               `json:"max_parallel"`
	Timeout     time.Duration     `json:"timeout"`
	Variables   map[string]any    `json:"variables,omitempty"`
	Strategy    ParallelStrategy  `json:"strategy"` // first, all, majority
}

// ParallelStrategy 并行策略
type ParallelStrategy string

const (
	StrategyFirst    ParallelStrategy = "first"    // 第一个成功的结果
	StrategyAll      ParallelStrategy = "all"      // 所有结果
	StrategyMajority ParallelStrategy = "majority" // 多数结果
)

// NewParallelConditionalAgent 创建并行条件Agent
func NewParallelConditionalAgent(config ParallelConditionalConfig) (*ParallelConditionalAgent, error) {
	if config.Name == "" {
		return nil, fmt.Errorf("parallel conditional agent name is required")
	}

	if len(config.Conditions) == 0 {
		return nil, fmt.Errorf("at least one condition is required")
	}

	maxParallel := config.MaxParallel
	if maxParallel <= 0 {
		maxParallel = len(config.Conditions)
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = time.Second * 30
	}

	return &ParallelConditionalAgent{
		name:         config.Name,
		conditions:   config.Conditions,
		maxParallel:  maxParallel,
		timeout:      timeout,
		defaultAgent: config.Default,
		strategy:     config.Strategy,
		evaluator:    NewExpressionEvaluator(config.Variables),
	}, nil
}

// Name 返回Agent名称
func (p *ParallelConditionalAgent) Name() string {
	return p.name
}

// Execute 执行并行条件评估
func (p *ParallelConditionalAgent) Execute(ctx context.Context, message string) *stream.Reader[*session.Event] {
	reader, writer := stream.Pipe[*session.Event](10)

	go func() {
		defer writer.Close()

		// 更新变量
		if p.evaluator.variables == nil {
			p.evaluator.variables = make(map[string]any)
		}
		p.evaluator.variables["input"] = message
		p.evaluator.variables["message"] = message

		// 并行评估条件
		results := p.evaluateConditionsParallel(ctx, message, writer)
		if len(results) == 0 {
			// 没有匹配的条件，使用默认分支
			if p.defaultAgent == nil {
				writer.Send(nil, fmt.Errorf("no condition matched and no default branch provided"))
				return
			}

			writer.Send(&session.Event{
				ID:        generateEventID(),
				Timestamp: time.Now(),
				AgentID:   p.name,
				Author:    "system",
				Content: types.Message{
					Role:    types.MessageRoleAssistant,
					Content: fmt.Sprintf("No conditions matched, using default branch: %s", p.defaultAgent.ID),
				},
				Metadata: map[string]any{
					"branch_type": "default",
					"agent_id":    p.defaultAgent.ID,
				},
			}, nil)

			p.executeBranchAgent(ctx, writer, p.defaultAgent, message, "default")
			return
		}

		// 根据策略处理结果
		switch p.strategy {
		case StrategyFirst:
			// 使用第一个结果
			result := results[0]
			p.handleBranchResult(ctx, writer, result, message)
		case StrategyAll:
			// 执行所有匹配的分支
			for _, result := range results {
				p.handleBranchResult(ctx, writer, result, message)
			}
		case StrategyMajority:
			// TODO: 实现多数策略
			p.handleBranchResult(ctx, writer, results[0], message)
		default:
			p.handleBranchResult(ctx, writer, results[0], message)
		}
	}()

	return reader
}

// evaluateConditionsParallel 并行评估条件
func (p *ParallelConditionalAgent) evaluateConditionsParallel(ctx context.Context, message string, writer *stream.Writer[*session.Event]) []BranchEvaluationResult {
	conditionCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	results := make(chan BranchEvaluationResult, len(p.conditions))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, p.maxParallel)

	// 启动并行评估
	for _, condition := range p.conditions {
		wg.Add(1)
		go func(cond BranchCondition) {
			defer wg.Done()

			semaphore <- struct{}{}        // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			result := p.evaluateSingleCondition(conditionCtx, cond, message)
			results <- result
		}(condition)
	}

	// 等待所有评估完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	var evaluationResults []BranchEvaluationResult
	for result := range results {
		if result.Matched {
			evaluationResults = append(evaluationResults, result)

			// 发送条件评估事件
			writer.Send(&session.Event{
				ID:        generateEventID(),
				Timestamp: time.Now(),
				AgentID:   p.name,
				Author:    "system",
				Content: types.Message{
					Role:    types.MessageRoleAssistant,
					Content: fmt.Sprintf("Condition matched: %s -> %s", result.Condition, result.Name),
				},
				Metadata: map[string]any{
					"branch_type": "conditional",
					"branch_name": result.Name,
					"condition":   result.Condition,
					"agent_id":    result.Agent.ID,
				},
			}, nil)
		}
	}

	return evaluationResults
}

// evaluateSingleCondition 评估单个条件
func (p *ParallelConditionalAgent) evaluateSingleCondition(ctx context.Context, condition BranchCondition, message string) BranchEvaluationResult {
	result := BranchEvaluationResult{
		Name:      condition.Name,
		Condition: condition.Condition,
		Agent:     condition.Agent,
		Metadata:  condition.Metadata,
	}

	startTime := time.Now()

	matched, err := p.evaluator.EvaluateBool(condition.Condition)
	result.Matched = matched
	result.Error = err
	result.Duration = time.Since(startTime)

	return result
}

// BranchEvaluationResult 分支评估结果
type BranchEvaluationResult struct {
	Name      string         `json:"name"`
	Condition string         `json:"condition"`
	Agent     *AgentRef      `json:"agent"`
	Matched   bool           `json:"matched"`
	Error     error          `json:"error,omitempty"`
	Duration  time.Duration  `json:"duration"`
	Metadata  map[string]any `json:"metadata"`
}

// handleBranchResult 处理分支结果
func (p *ParallelConditionalAgent) handleBranchResult(ctx context.Context, writer *stream.Writer[*session.Event], result any, message string) {
	var agentRef *AgentRef
	var branchName string

	switch r := result.(type) {
	case BranchEvaluationResult:
		agentRef = r.Agent
		branchName = r.Name
	default:
		return
	}

	p.executeBranchAgent(ctx, writer, agentRef, message, branchName)
}

// executeBranchAgent 执行分支Agent
func (p *ParallelConditionalAgent) executeBranchAgent(ctx context.Context, writer *stream.Writer[*session.Event], agentRef *AgentRef, message string, branchName string) {
	// TODO: 实现Agent创建和执行
	// 现在模拟执行
	for i := range 3 {
		event := &session.Event{
			ID:        generateEventID(),
			Timestamp: time.Now(),
			AgentID:   p.name,
			Author:    "system",
			Content: types.Message{
				Role:    types.MessageRoleAssistant,
				Content: fmt.Sprintf("Parallel branch '%s' execution step %d: %s", branchName, i+1, message),
			},
			Metadata: map[string]any{
				"branch_name": branchName,
				"agent_id":    agentRef.ID,
				"step":        i + 1,
				"parallel":    true,
			},
		}

		if writer.Send(event, nil) {
			return
		}

		// 模拟延迟
		time.Sleep(time.Millisecond * 50)
	}
}

// SwitchAgent Switch分支Agent（类似编程语言的switch语句）
type SwitchAgent struct {
	name        string
	cases       []SwitchCase
	defaultCase *AgentRef
	variable    string // switch变量名
	evaluator   *ExpressionEvaluator
}

// SwitchCase Switch分支
type SwitchCase struct {
	Value       string    `json:"value"`       // 匹配值
	Agent       *AgentRef `json:"agent"`       // 分支Agent
	Name        string    `json:"name"`        // 分支名称
	Fallthrough bool      `json:"fallthrough"` // 是否继续匹配下一个case
}

// SwitchConfig SwitchAgent配置
type SwitchConfig struct {
	Name     string       `json:"name"`
	Variable string       `json:"variable"` // switch变量名
	Cases    []SwitchCase `json:"cases"`
	Default  *AgentRef    `json:"default,omitempty"`
}

// NewSwitchAgent 创建Switch Agent
func NewSwitchAgent(config SwitchConfig) (*SwitchAgent, error) {
	if config.Name == "" {
		return nil, fmt.Errorf("switch agent name is required")
	}

	if config.Variable == "" {
		return nil, fmt.Errorf("switch variable is required")
	}

	return &SwitchAgent{
		name:        config.Name,
		cases:       config.Cases,
		defaultCase: config.Default,
		variable:    config.Variable,
		evaluator:   NewExpressionEvaluator(make(map[string]any)),
	}, nil
}

// Name 返回Agent名称
func (s *SwitchAgent) Name() string {
	return s.name
}

// Execute 执行Switch分支
func (s *SwitchAgent) Execute(ctx context.Context, message string) *stream.Reader[*session.Event] {
	reader, writer := stream.Pipe[*session.Event](10)

	go func() {
		defer writer.Close()

		// 解析switch变量值
		switchValue, err := s.extractSwitchValue(message)
		if err != nil {
			writer.Send(nil, fmt.Errorf("failed to extract switch value: %w", err))
			return
		}

		// 查找匹配的case
		for _, caseDef := range s.cases {
			if s.matchCaseValue(switchValue, caseDef.Value) {
				writer.Send(&session.Event{
					ID:        generateEventID(),
					Timestamp: time.Now(),
					AgentID:   s.name,
					Author:    "system",
					Content: types.Message{
						Role:    types.MessageRoleAssistant,
						Content: fmt.Sprintf("Switch matched: %s == %s -> %s", s.variable, switchValue, caseDef.Name),
					},
					Metadata: map[string]any{
						"switch_type": "switch",
						"variable":    s.variable,
						"case_value":  caseDef.Value,
						"case_name":   caseDef.Name,
						"agent_id":    caseDef.Agent.ID,
					},
				}, nil)

				// 执行匹配的case
				s.executeSwitchCase(ctx, writer, caseDef, message)

				// 检查是否继续fallthrough
				if !caseDef.Fallthrough {
					return
				}
			}
		}

		// 使用default分支
		if s.defaultCase != nil {
			writer.Send(&session.Event{
				ID:        generateEventID(),
				Timestamp: time.Now(),
				AgentID:   s.name,
				Author:    "system",
				Content: types.Message{
					Role:    types.MessageRoleAssistant,
					Content: fmt.Sprintf("No case matched for %s = %s, using default: %s", s.variable, switchValue, s.defaultCase.ID),
				},
				Metadata: map[string]any{
					"switch_type": "default",
					"variable":    s.variable,
					"value":       switchValue,
					"agent_id":    s.defaultCase.ID,
				},
			}, nil)

			// 执行默认分支
			defaultCase := SwitchCase{
				Name:  "default",
				Value: "*",
				Agent: s.defaultCase,
			}
			s.executeSwitchCase(ctx, writer, defaultCase, message)
		} else {
			writer.Send(nil, fmt.Errorf("no case matched for %s = %s and no default provided", s.variable, switchValue))
		}
	}()

	return reader
}

// extractSwitchValue 提取switch变量值
func (s *SwitchAgent) extractSwitchValue(message string) (string, error) {
	// 简单实现：尝试从JSON消息中解析
	var data map[string]any
	if err := json.Unmarshal([]byte(message), &data); err == nil {
		if value, exists := data[s.variable]; exists {
			return fmt.Sprintf("%v", value), nil
		}
	}

	// 如果不是JSON，将整个消息作为值
	return message, nil
}

// matchCaseValue 匹配case值
func (s *SwitchAgent) matchCaseValue(switchValue, caseValue string) bool {
	// 精确匹配
	if strings.EqualFold(switchValue, caseValue) {
		return true
	}

	// 通配符匹配
	if caseValue == "*" {
		return true
	}

	// 范围匹配 (caseValue可以是 "value1,value2,value3")
	if strings.Contains(caseValue, ",") {
		values := strings.Split(caseValue, ",")
		for _, v := range values {
			if strings.EqualFold(strings.TrimSpace(v), switchValue) {
				return true
			}
		}
	}

	// 正则表达式匹配 (caseValue以/开头和结尾)
	if strings.HasPrefix(caseValue, "/") && strings.HasSuffix(caseValue, "/") {
		// TODO: 实现正则匹配
		return false
	}

	return false
}

// executeSwitchCase 执行Switch case
func (s *SwitchAgent) executeSwitchCase(ctx context.Context, writer *stream.Writer[*session.Event], caseDef SwitchCase, message string) {
	// TODO: 实现Agent创建和执行
	// 现在模拟执行
	for i := range 3 {
		event := &session.Event{
			ID:        generateEventID(),
			Timestamp: time.Now(),
			AgentID:   s.name,
			Author:    "system",
			Content: types.Message{
				Role:    types.MessageRoleAssistant,
				Content: fmt.Sprintf("Switch case '%s' execution step %d: %s", caseDef.Name, i+1, message),
			},
			Metadata: map[string]any{
				"switch_type": "switch",
				"case_name":   caseDef.Name,
				"case_value":  caseDef.Value,
				"agent_id":    caseDef.Agent.ID,
				"step":        i + 1,
			},
		}

		if writer.Send(event, nil) {
			return
		}

		// 模拟延迟
		time.Sleep(time.Millisecond * 80)
	}
}

// MultiLevelConditionalAgent 多级条件Agent（嵌套条件）
type MultiLevelConditionalAgent struct {
	name         string
	levels       []ConditionLevel
	evaluator    *ExpressionEvaluator
	currentLevel int
}

// ConditionLevel 条件层级
type ConditionLevel struct {
	Name       string            `json:"name"`
	Conditions []BranchCondition `json:"conditions"`
	Level      int               `json:"level"`
	Else       *ConditionLevel   `json:"else,omitempty"` // else分支
	Metadata   map[string]any    `json:"metadata"`
}

// MultiLevelConditionalConfig 多级条件Agent配置
type MultiLevelConditionalConfig struct {
	Name      string           `json:"name"`
	Levels    []ConditionLevel `json:"levels"`
	Variables map[string]any   `json:"variables,omitempty"`
	MaxDepth  int              `json:"max_depth"`
}

// NewMultiLevelConditionalAgent 创建多级条件Agent
func NewMultiLevelConditionalAgent(config MultiLevelConditionalConfig) (*MultiLevelConditionalAgent, error) {
	if config.Name == "" {
		return nil, fmt.Errorf("multi-level conditional agent name is required")
	}

	if len(config.Levels) == 0 {
		return nil, fmt.Errorf("at least one level is required")
	}

	maxDepth := config.MaxDepth
	if maxDepth <= 0 {
		maxDepth = len(config.Levels)
	}
	_ = maxDepth // maxDepth is currently unused but may be needed for future depth control

	return &MultiLevelConditionalAgent{
		name:         config.Name,
		levels:       config.Levels,
		evaluator:    NewExpressionEvaluator(config.Variables),
		currentLevel: 0,
	}, nil
}

// Name 返回Agent名称
func (m *MultiLevelConditionalAgent) Name() string {
	return m.name
}

// Execute 执行多级条件
func (m *MultiLevelConditionalAgent) Execute(ctx context.Context, message string) *stream.Reader[*session.Event] {
	reader, writer := stream.Pipe[*session.Event](10)

	go func() {
		defer writer.Close()
		// 从第一级开始评估
		m.currentLevel = 0
		m.evaluateLevel(ctx, writer, m.levels[0], message)
	}()

	return reader
}

// evaluateLevel 评估条件层级
func (m *MultiLevelConditionalAgent) evaluateLevel(ctx context.Context, writer *stream.Writer[*session.Event], level ConditionLevel, message string) {
	// 更新变量
	if m.evaluator.variables == nil {
		m.evaluator.variables = make(map[string]any)
	}
	m.evaluator.variables["input"] = message
	m.evaluator.variables["level"] = level.Level

	// 发送层级开始事件
	writer.Send(&session.Event{
		ID:        generateEventID(),
		Timestamp: time.Now(),
		AgentID:   m.name,
		Author:    "system",
		Content: types.Message{
			Role:    types.MessageRoleAssistant,
			Content: fmt.Sprintf("Evaluating level %d: %s", level.Level, level.Name),
		},
		Metadata: map[string]any{
			"conditional_type": "multi_level",
			"level":            level.Level,
			"level_name":       level.Name,
		},
	}, nil)

	// 评估当前层级的条件
	for _, condition := range level.Conditions {
		result, err := m.evaluator.EvaluateBool(condition.Condition)
		if err != nil {
			// 记录错误但继续评估其他条件
			continue
		}

		if result {
			// 条件匹配，执行对应的Agent
			writer.Send(&session.Event{
				ID:        generateEventID(),
				Timestamp: time.Now(),
				AgentID:   m.name,
				Author:    "system",
				Content: types.Message{
					Role:    types.MessageRoleAssistant,
					Content: fmt.Sprintf("Condition matched at level %d: %s -> %s", level.Level, condition.Condition, condition.Name),
				},
				Metadata: map[string]any{
					"conditional_type": "multi_level",
					"level":            level.Level,
					"condition":        condition.Condition,
					"branch_name":      condition.Name,
					"agent_id":         condition.Agent.ID,
				},
			}, nil)

			// 执行分支Agent
			if condition.Agent != nil {
				m.executeBranchAgent(ctx, writer, condition.Agent, message, fmt.Sprintf("L%d_%s", level.Level, condition.Name))
			}
			return
		}
	}

	// 如果没有条件匹配，检查是否有else分支
	if level.Else != nil {
		writer.Send(&session.Event{
			ID:        generateEventID(),
			Timestamp: time.Now(),
			AgentID:   m.name,
			Author:    "system",
			Content: types.Message{
				Role:    types.MessageRoleAssistant,
				Content: fmt.Sprintf("No conditions matched at level %d, using else branch: %s", level.Level, level.Else.Name),
			},
			Metadata: map[string]any{
				"conditional_type": "multi_level",
				"level":            level.Level,
				"branch_type":      "else",
				"else_name":        level.Else.Name,
			},
		}, nil)

		// 执行else分支
		if len(level.Else.Conditions) > 0 {
			// 如果else分支有自己的条件，递归评估
			m.evaluateLevel(ctx, writer, *level.Else, message)
		} else {
			// 否则继续下一级
			if level.Level+1 < len(m.levels) {
				m.currentLevel = level.Level + 1
				m.evaluateLevel(ctx, writer, m.levels[level.Level+1], message)
			}
		}
	} else {
		// 没有else分支，继续下一级
		if level.Level+1 < len(m.levels) {
			m.currentLevel = level.Level + 1
			m.evaluateLevel(ctx, writer, m.levels[level.Level+1], message)
		}
	}
}

// executeBranchAgent 执行分支Agent
func (m *MultiLevelConditionalAgent) executeBranchAgent(ctx context.Context, writer *stream.Writer[*session.Event], agentRef *AgentRef, message string, branchName string) {
	// TODO: 实现Agent创建和执行
	// 现在模拟执行
	for i := range 3 {
		event := &session.Event{
			ID:        generateEventID(),
			Timestamp: time.Now(),
			AgentID:   m.name,
			Author:    "system",
			Content: types.Message{
				Role:    types.MessageRoleAssistant,
				Content: fmt.Sprintf("Multi-level branch '%s' execution step %d: %s", branchName, i+1, message),
			},
			Metadata: map[string]any{
				"conditional_type": "multi_level",
				"branch_name":      branchName,
				"agent_id":         agentRef.ID,
				"step":             i + 1,
				"level":            m.currentLevel,
			},
		}

		if writer.Send(event, nil) {
			return
		}

		// 模拟延迟
		time.Sleep(time.Millisecond * 120)
	}
}
