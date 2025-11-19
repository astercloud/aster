package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/astercloud/aster/pkg/session"
)

// WorkflowDefinition 工作流定义
type WorkflowDefinition struct {
	// 基本信息
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Tags        []string `json:"tags"`

	// 输入输出定义
	Inputs  []VariableDef `json:"inputs"`
	Outputs []VariableDef `json:"outputs"`

	// 工作流图
	Nodes []NodeDef `json:"nodes"`
	Edges []EdgeDef `json:"edges"`

	// 执行配置
	Config *WorkflowConfig `json:"config,omitempty"`

	// 元数据
	Metadata map[string]string `json:"metadata,omitempty"`
}

// VariableDef 变量定义
type VariableDef struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"` // string, number, boolean, object, array
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Validation  string      `json:"validation,omitempty"` // JSON Schema or validation rules
}

// NodeDef 节点定义
type NodeDef struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      NodeType               `json:"type"`
	Position  Position               `json:"position"`
	Config    map[string]interface{} `json:"config,omitempty"`
	Agent     *AgentRef              `json:"agent,omitempty"`     // Agent节点
	Condition *ConditionDef          `json:"condition,omitempty"` // 条件节点
	Loop      *LoopDef               `json:"loop,omitempty"`      // 循环节点
	Parallel  *ParallelDef           `json:"parallel,omitempty"`  // 并行节点
	Timeout   time.Duration          `json:"timeout,omitempty"`   // 超时时间
	Retry     *RetryDef              `json:"retry,omitempty"`     // 重试配置
}

// Position 位置信息
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// NodeType 节点类型
type NodeType string

const (
	NodeTypeStart     NodeType = "start"     // 开始节点
	NodeTypeEnd       NodeType = "end"       // 结束节点
	NodeTypeTask      NodeType = "task"      // 任务节点(Agent)
	NodeTypeCondition NodeType = "condition" // 条件节点
	NodeTypeLoop      NodeType = "loop"      // 循环节点
	NodeTypeParallel  NodeType = "parallel"  // 并行节点
	NodeTypeMerge     NodeType = "merge"     // 合并节点
	NodeTypeTimeout   NodeType = "timeout"   // 超时节点
	NodeTypeError     NodeType = "error"     // 错误处理节点
	NodeTypeSubflow   NodeType = "subflow"   // 子工作流节点
)

// EdgeDef 边定义
type EdgeDef struct {
	ID        string            `json:"id"`
	From      string            `json:"from"` // 源节点ID
	To        string            `json:"to"`   // 目标节点ID
	Label     string            `json:"label,omitempty"`
	Condition string            `json:"condition,omitempty"` // 边条件
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// AgentRef Agent引用
type AgentRef struct {
	ID       string                 `json:"id"`
	Template string                 `json:"template"`
	Config   map[string]interface{} `json:"config,omitempty"`
	Inputs   map[string]string      `json:"inputs,omitempty"`  // 输入映射
	Outputs  map[string]string      `json:"outputs,omitempty"` // 输出映射
}

// ConditionDef 条件定义
type ConditionDef struct {
	Type   ConditionType   `json:"type"`             // and, or, not, custom
	Rules  []ConditionRule `json:"rules"`            // 条件规则
	Custom string          `json:"custom,omitempty"` // 自定义条件表达式
}

// ConditionType 条件类型
type ConditionType string

const (
	ConditionTypeAnd    ConditionType = "and"
	ConditionTypeOr     ConditionType = "or"
	ConditionTypeNot    ConditionType = "not"
	ConditionTypeCustom ConditionType = "custom"
)

// ConditionRule 条件规则
type ConditionRule struct {
	Variable string      `json:"variable"` // 变量路径，如 "input.score"
	Operator string      `json:"operator"` // eq, ne, gt, gte, lt, lte, in, nin, contains, regex
	Value    interface{} `json:"value"`    // 比较值
}

// LoopDef 循环定义
type LoopDef struct {
	Type      LoopType `json:"type"`      // for, while, until, foreach
	Variable  string   `json:"variable"`  // 循环变量名
	Iterator  string   `json:"iterator"`  // 迭代器表达式
	Condition string   `json:"condition"` // 循环条件
	MaxLoops  int      `json:"max_loops"` // 最大循环次数(0=无限制)
}

// LoopType 循环类型
type LoopType string

const (
	LoopTypeFor     LoopType = "for"     // for i in range(10)
	LoopTypeWhile   LoopType = "while"   // while condition
	LoopTypeUntil   LoopType = "until"   // until condition
	LoopTypeForEach LoopType = "foreach" // foreach item in list
)

// ParallelDef 并行定义
type ParallelDef struct {
	Type     ParallelType `json:"type"`      // all, any, race
	Branches []NodeRef    `json:"branches"`  // 并行分支
	JoinType JoinType     `json:"join_type"` // wait, first, success, majority
}

// ParallelType 并行类型
type ParallelType string

const (
	ParallelTypeAll  ParallelType = "all"  // 执行所有分支
	ParallelTypeAny  ParallelType = "any"  // 执行任一分支
	ParallelTypeRace ParallelType = "race" // 竞争执行，最快的获胜
)

// JoinType 连接类型
type JoinType string

const (
	JoinTypeWait     JoinType = "wait"     // 等待所有分支完成
	JoinTypeFirst    JoinType = "first"    // 等待第一个分支完成
	JoinTypeSuccess  JoinType = "success"  // 等待一个成功分支完成
	JoinTypeMajority JoinType = "majority" // 等待多数分支完成
)

// RetryDef 重试定义
type RetryDef struct {
	MaxAttempts int           `json:"max_attempts"` // 最大重试次数
	Delay       time.Duration `json:"delay"`        // 重试延迟
	Backoff     BackoffType   `json:"backoff"`      // 退避策略
	MaxDelay    time.Duration `json:"max_delay"`    // 最大延迟
}

// BackoffType 退避策略
type BackoffType string

const (
	BackoffTypeFixed       BackoffType = "fixed"       // 固定延迟
	BackoffTypeLinear      BackoffType = "linear"      // 线性增长
	BackoffTypeExponential BackoffType = "exponential" // 指数退避
)

// WorkflowConfig 工作流配置
type WorkflowConfig struct {
	// 超时配置
	DefaultTimeout time.Duration `json:"default_timeout"`

	// 重试配置
	DefaultRetry *RetryDef `json:"default_retry,omitempty"`

	// 并发配置
	MaxConcurrency int `json:"max_concurrency"`

	// 监控配置
	Monitoring *MonitoringConfig `json:"monitoring,omitempty"`

	// 日志配置
	Logging *LoggingConfig `json:"logging,omitempty"`

	// 安全配置
	Security *SecurityConfig `json:"security,omitempty"`
}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	EnableMetrics bool   `json:"enable_metrics"`
	MetricsPath   string `json:"metrics_path"`
	EnableTracing bool   `json:"enable_tracing"`
	TracingPath   string `json:"tracing_path"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level   string   `json:"level"`   // debug, info, warn, error
	Format  string   `json:"format"`  // json, text
	Output  string   `json:"output"`  // stdout, file, syslog
	Exclude []string `json:"exclude"` // 排除的节点
	Include []string `json:"include"` // 包含的节点
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	EnableAuth     bool     `json:"enable_auth"`
	AllowedRoles   []string `json:"allowed_roles"`
	DataEncryption bool     `json:"data_encryption"`
	AuditLogging   bool     `json:"audit_logging"`
	SandboxMode    bool     `json:"sandbox_mode"`
}

// WorkflowContext 工作流执行上下文
type WorkflowContext struct {
	// 基本信息
	WorkflowID  string         `json:"workflow_id"`
	ExecutionID string         `json:"execution_id"`
	StartTime   time.Time      `json:"start_time"`
	Status      WorkflowStatus `json:"status"`

	// 变量存储
	Variables map[string]interface{} `json:"variables"`
	Inputs    map[string]interface{} `json:"inputs"`
	Outputs   map[string]interface{} `json:"outputs"`

	// 执行状态
	CurrentNode string           `json:"current_node"`
	Completed   map[string]bool  `json:"completed"`
	Failed      map[string]error `json:"failed"`

	// 元数据
	Metadata map[string]interface{} `json:"metadata"`

	// 上下文
	Context context.Context  `json:"-"`
	Session *session.Session `json:"-"`
}

// WorkflowStatus 工作流状态
type WorkflowStatus string

const (
	StatusPending   WorkflowStatus = "pending"   // 等待执行
	StatusRunning   WorkflowStatus = "running"   // 执行中
	StatusPaused    WorkflowStatus = "paused"    // 暂停
	StatusCompleted WorkflowStatus = "completed" // 完成
	StatusFailed    WorkflowStatus = "failed"    // 失败
	StatusCancelled WorkflowStatus = "cancelled" // 取消
	StatusTimeout   WorkflowStatus = "timeout"   // 超时
)

// WorkflowResult 工作流执行结果
type WorkflowResult struct {
	ExecutionID string                 `json:"execution_id"`
	WorkflowID  string                 `json:"workflow_id"`
	Status      WorkflowStatus         `json:"status"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     time.Time              `json:"end_time"`
	Duration    time.Duration          `json:"duration"`
	Outputs     map[string]interface{} `json:"outputs"`
	Errors      []WorkflowError        `json:"errors,omitempty"`
	Metrics     *WorkflowMetrics       `json:"metrics,omitempty"`
	Trace       []WorkflowStep         `json:"trace,omitempty"`
}

// WorkflowError 工作流错误
type WorkflowError struct {
	NodeID    string    `json:"node_id"`
	NodeName  string    `json:"node_name"`
	Error     string    `json:"error"`
	Timestamp time.Time `json:"timestamp"`
	Retryable bool      `json:"retryable"`
}

// WorkflowMetrics 工作流指标
type WorkflowMetrics struct {
	TotalNodes      int           `json:"total_nodes"`
	CompletedNodes  int           `json:"completed_nodes"`
	FailedNodes     int           `json:"failed_nodes"`
	SkippedNodes    int           `json:"skipped_nodes"`
	TotalDuration   time.Duration `json:"total_duration"`
	AverageNodeTime time.Duration `json:"average_node_time"`
	MaxNodeTime     time.Duration `json:"max_node_time"`
	MinNodeTime     time.Duration `json:"min_node_time"`
}

// WorkflowStep 工作流步骤
type WorkflowStep struct {
	NodeID     string                 `json:"node_id"`
	NodeName   string                 `json:"node_name"`
	NodeType   NodeType               `json:"node_type"`
	Status     WorkflowStatus         `json:"status"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    time.Time              `json:"end_time"`
	Duration   time.Duration          `json:"duration"`
	Inputs     map[string]interface{} `json:"inputs"`
	Outputs    map[string]interface{} `json:"outputs"`
	Error      string                 `json:"error,omitempty"`
	RetryCount int                    `json:"retry_count"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// NodeRef 节点引用
type NodeRef struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Branch string `json:"branch,omitempty"`
}

// DSLBuilder DSL构建器
type DSLBuilder struct {
	def *WorkflowDefinition
}

// NewDSLBuilder 创建DSL构建器
func NewDSLBuilder(id, name string) *DSLBuilder {
	return &DSLBuilder{
		def: &WorkflowDefinition{
			ID:       id,
			Name:     name,
			Version:  "1.0.0",
			Nodes:    make([]NodeDef, 0),
			Edges:    make([]EdgeDef, 0),
			Inputs:   make([]VariableDef, 0),
			Outputs:  make([]VariableDef, 0),
			Metadata: make(map[string]string),
		},
	}
}

// SetDescription 设置描述
func (b *DSLBuilder) SetDescription(desc string) *DSLBuilder {
	b.def.Description = desc
	return b
}

// AddInput 添加输入
func (b *DSLBuilder) AddInput(name, varType, description string, required bool, defaultValue interface{}) *DSLBuilder {
	b.def.Inputs = append(b.def.Inputs, VariableDef{
		Name:        name,
		Type:        varType,
		Description: description,
		Required:    required,
		Default:     defaultValue,
	})
	return b
}

// AddOutput 添加输出
func (b *DSLBuilder) AddOutput(name, varType, description string) *DSLBuilder {
	b.def.Outputs = append(b.def.Outputs, VariableDef{
		Name:        name,
		Type:        varType,
		Description: description,
	})
	return b
}

// AddStartNode 添加开始节点
func (b *DSLBuilder) AddStartNode(id string, position Position) *DSLBuilder {
	b.def.Nodes = append(b.def.Nodes, NodeDef{
		ID:       id,
		Name:     "Start",
		Type:     NodeTypeStart,
		Position: position,
	})
	return b
}

// AddEndNode 添加结束节点
func (b *DSLBuilder) AddEndNode(id string, position Position) *DSLBuilder {
	b.def.Nodes = append(b.def.Nodes, NodeDef{
		ID:       id,
		Name:     "End",
		Type:     NodeTypeEnd,
		Position: position,
	})
	return b
}

// AddTaskNode 添加任务节点
func (b *DSLBuilder) AddTaskNode(id, name string, agent *AgentRef, position Position) *DSLBuilder {
	b.def.Nodes = append(b.def.Nodes, NodeDef{
		ID:       id,
		Name:     name,
		Type:     NodeTypeTask,
		Position: position,
		Agent:    agent,
	})
	return b
}

// AddConditionNode 添加条件节点
func (b *DSLBuilder) AddConditionNode(id, name string, condition *ConditionDef, position Position) *DSLBuilder {
	b.def.Nodes = append(b.def.Nodes, NodeDef{
		ID:        id,
		Name:      name,
		Type:      NodeTypeCondition,
		Position:  position,
		Condition: condition,
	})
	return b
}

// AddLoopNode 添加循环节点
func (b *DSLBuilder) AddLoopNode(id, name string, loop *LoopDef, position Position) *DSLBuilder {
	b.def.Nodes = append(b.def.Nodes, NodeDef{
		ID:       id,
		Name:     name,
		Type:     NodeTypeLoop,
		Position: position,
		Loop:     loop,
	})
	return b
}

// AddParallelNode 添加并行节点
func (b *DSLBuilder) AddParallelNode(id, name string, parallel *ParallelDef, position Position) *DSLBuilder {
	b.def.Nodes = append(b.def.Nodes, NodeDef{
		ID:       id,
		Name:     name,
		Type:     NodeTypeParallel,
		Position: position,
		Parallel: parallel,
	})
	return b
}

// AddEdge 添加边
func (b *DSLBuilder) AddEdge(id, from, to string, label string, condition string) *DSLBuilder {
	b.def.Edges = append(b.def.Edges, EdgeDef{
		ID:        id,
		From:      from,
		To:        to,
		Label:     label,
		Condition: condition,
	})
	return b
}

// SetConfig 设置配置
func (b *DSLBuilder) SetConfig(config *WorkflowConfig) *DSLBuilder {
	b.def.Config = config
	return b
}

// Build 构建工作流定义
func (b *DSLBuilder) Build() *WorkflowDefinition {
	return b.def
}

// ParseFromJSON 从JSON解析工作流定义
func ParseFromJSON(data []byte) (*WorkflowDefinition, error) {
	var def WorkflowDefinition
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("failed to parse workflow JSON: %w", err)
	}

	// 验证定义
	if err := validateWorkflowDefinition(&def); err != nil {
		return nil, fmt.Errorf("invalid workflow definition: %w", err)
	}

	return &def, nil
}

// ParseFromYAML 从YAML解析工作流定义
func ParseFromYAML(data []byte) (*WorkflowDefinition, error) {
	// TODO: 实现YAML解析
	return nil, fmt.Errorf("YAML parsing not yet implemented")
}

// ToJSON 转换为JSON
func (w *WorkflowDefinition) ToJSON() ([]byte, error) {
	return json.MarshalIndent(w, "", "  ")
}

// ToYAML 转换为YAML
func (w *WorkflowDefinition) ToYAML() ([]byte, error) {
	// TODO: 实现YAML转换
	return nil, fmt.Errorf("YAML conversion not yet implemented")
}

// validateWorkflowDefinition 验证工作流定义
func validateWorkflowDefinition(def *WorkflowDefinition) error {
	if def.ID == "" {
		return fmt.Errorf("workflow ID is required")
	}

	if def.Name == "" {
		return fmt.Errorf("workflow name is required")
	}

	if len(def.Nodes) == 0 {
		return fmt.Errorf("workflow must have at least one node")
	}

	// 检查是否有开始和结束节点
	hasStart := false
	hasEnd := false
	nodeIds := make(map[string]bool)

	for _, node := range def.Nodes {
		if node.ID == "" {
			return fmt.Errorf("node ID is required")
		}

		if nodeIds[node.ID] {
			return fmt.Errorf("duplicate node ID: %s", node.ID)
		}
		nodeIds[node.ID] = true

		if node.Type == NodeTypeStart {
			hasStart = true
		}
		if node.Type == NodeTypeEnd {
			hasEnd = true
		}
	}

	if !hasStart {
		return fmt.Errorf("workflow must have a start node")
	}

	if !hasEnd {
		return fmt.Errorf("workflow must have an end node")
	}

	// 验证边的引用
	for _, edge := range def.Edges {
		if edge.From == "" || edge.To == "" {
			return fmt.Errorf("edge source and target are required")
		}

		if !nodeIds[edge.From] {
			return fmt.Errorf("edge source node not found: %s", edge.From)
		}

		if !nodeIds[edge.To] {
			return fmt.Errorf("edge target node not found: %s", edge.To)
		}
	}

	return nil
}

// ExpressionEvaluator 表达式求值器
type ExpressionEvaluator struct {
	variables map[string]interface{}
}

// NewExpressionEvaluator 创建表达式求值器
func NewExpressionEvaluator(variables map[string]interface{}) *ExpressionEvaluator {
	return &ExpressionEvaluator{
		variables: variables,
	}
}

// EvaluateBool 评估布尔表达式
func (e *ExpressionEvaluator) EvaluateBool(expression string) (bool, error) {
	// 简单实现，支持基本的比较操作
	// TODO: 实现更完整的表达式求值器

	expression = strings.TrimSpace(expression)

	// 处理逻辑操作
	if strings.Contains(expression, "&&") {
		parts := strings.Split(expression, "&&")
		for _, part := range parts {
			if result, err := e.EvaluateBool(strings.TrimSpace(part)); err != nil {
				return false, err
			} else if !result {
				return false, nil
			}
		}
		return true, nil
	}

	if strings.Contains(expression, "||") {
		parts := strings.Split(expression, "||")
		for _, part := range parts {
			if result, err := e.EvaluateBool(strings.TrimSpace(part)); err != nil {
				return false, err
			} else if result {
				return true, nil
			}
		}
		return false, nil
	}

	// 处理比较操作
	return e.evaluateComparison(expression)
}

// evaluateComparison 评估比较表达式
func (e *ExpressionEvaluator) evaluateComparison(expression string) (bool, error) {
	// 正则匹配比较操作
	re := regexp.MustCompile(`^(\w+)\s*(==|!=|>=|<=|>|<)\s*(.+)$`)
	matches := re.FindStringSubmatch(expression)
	if matches == nil {
		return false, fmt.Errorf("invalid comparison expression: %s", expression)
	}

	variable := matches[1]
	operator := matches[2]
	valueStr := matches[3]

	variableValue, exists := e.variables[variable]
	if !exists {
		return false, fmt.Errorf("variable not found: %s", variable)
	}

	// 简单的类型转换和比较
	switch operator {
	case "==":
		return fmt.Sprintf("%v", variableValue) == valueStr, nil
	case "!=":
		return fmt.Sprintf("%v", variableValue) != valueStr, nil
	case ">":
		return e.compareNumbers(variableValue, valueStr, func(a, b float64) bool { return a > b })
	case ">=":
		return e.compareNumbers(variableValue, valueStr, func(a, b float64) bool { return a >= b })
	case "<":
		return e.compareNumbers(variableValue, valueStr, func(a, b float64) bool { return a < b })
	case "<=":
		return e.compareNumbers(variableValue, valueStr, func(a, b float64) bool { return a <= b })
	default:
		return false, fmt.Errorf("unsupported operator: %s", operator)
	}
}

// compareNumbers 比较数字
func (e *ExpressionEvaluator) compareNumbers(aVal, bVal interface{}, compare func(float64, float64) bool) (bool, error) {
	a, err := e.toFloat64(aVal)
	if err != nil {
		return false, fmt.Errorf("cannot convert left operand to number: %v", err)
	}

	b, err := e.toFloat64(bVal)
	if err != nil {
		return false, fmt.Errorf("cannot convert right operand to number: %v", err)
	}

	return compare(a, b), nil
}

// toFloat64 转换为float64
func (e *ExpressionEvaluator) toFloat64(v interface{}) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case string:
		// 尝试解析字符串为数字
		var f float64
		_, err := fmt.Sscanf(val, "%f", &f)
		if err != nil {
			return 0, fmt.Errorf("cannot parse string as number: %s", val)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("unsupported type for number conversion: %T", v)
	}
}
