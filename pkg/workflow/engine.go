package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/agent/workflow"
	"github.com/astercloud/aster/pkg/session"
)

// Engine 工作流执行引擎
type Engine struct {
	// 配置
	config *EngineConfig

	// 依赖
	agentFactory   AgentFactory
	sessionManager SessionManager
	eventBus       EventBus

	// 运行时状态
	executions   map[string]*WorkflowExecution
	executionsMu sync.RWMutex

	// 监控
	metrics *EngineMetrics

	// 安全
	security *SecurityManager
}

// EngineConfig 引擎配置
type EngineConfig struct {
	// 并发配置
	MaxConcurrentWorkflows int `json:"max_concurrent_workflows"`
	MaxConcurrentNodes     int `json:"max_concurrent_nodes"`

	// 超时配置
	DefaultNodeTimeout     time.Duration `json:"default_node_timeout"`
	DefaultWorkflowTimeout time.Duration `json:"default_workflow_timeout"`

	// 重试配置
	DefaultRetryPolicy *RetryDef `json:"default_retry_policy"`

	// 缓存配置
	EnableResultCache bool          `json:"enable_result_cache"`
	CacheTTL          time.Duration `json:"cache_ttl"`

	// 监控配置
	EnableMetrics bool `json:"enable_metrics"`
	EnableTracing bool `json:"enable_tracing"`

	// 安全配置
	EnableSandbox bool `json:"enable_sandbox"`
	EnableAudit   bool `json:"enable_audit"`
}

// WorkflowExecution 工作流执行实例
type WorkflowExecution struct {
	// 基本信息
	ID         string              `json:"id"`
	WorkflowID string              `json:"workflow_id"`
	Definition *WorkflowDefinition `json:"definition"`
	Status     WorkflowStatus      `json:"status"`
	Context    *WorkflowContext    `json:"context"`

	// 执行状态
	CurrentNodes   []string               `json:"current_nodes"`
	CompletedNodes map[string]bool        `json:"completed_nodes"`
	FailedNodes    map[string]error       `json:"failed_nodes"`
	NodeResults    map[string]*NodeResult `json:"node_results"`

	// 时间信息
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	LastActivity time.Time `json:"last_activity"`

	// 同步控制
	mu         sync.RWMutex
	cancelFunc context.CancelFunc
	done       chan struct{}

	// 错误处理
	Errors   []WorkflowError `json:"errors"`
	Warnings []string        `json:"warnings"`
}

// NodeResult 节点执行结果
type NodeResult struct {
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

// AgentFactory Agent工厂接口
type AgentFactory interface {
	CreateAgent(ctx context.Context, ref *AgentRef, config map[string]interface{}) (workflow.Agent, error)
}

// SessionManager 会话管理器接口
type SessionManager interface {
	CreateSession(ctx context.Context, workflowID string) (*session.Session, error)
	GetSession(sessionID string) (*session.Session, error)
	CloseSession(sessionID string) error
}

// EventBus 事件总线接口
type EventBus interface {
	Publish(ctx context.Context, event *WorkflowEvent) error
	Subscribe(eventType string, handler EventHandler) error
	Unsubscribe(eventType string, handler EventHandler) error
}

// WorkflowEvent 工作流事件
type WorkflowEvent struct {
	Type        string                 `json:"type"`
	ExecutionID string                 `json:"execution_id"`
	NodeID      string                 `json:"node_id"`
	Timestamp   time.Time              `json:"timestamp"`
	Data        map[string]interface{} `json:"data"`
}

// EventHandler 事件处理器
type EventHandler func(ctx context.Context, event *WorkflowEvent) error

// EngineMetrics 引擎指标
type EngineMetrics struct {
	TotalExecutions      int64         `json:"total_executions"`
	RunningExecutions    int64         `json:"running_executions"`
	CompletedExecutions  int64         `json:"completed_executions"`
	FailedExecutions     int64         `json:"failed_executions"`
	AverageExecutionTime time.Duration `json:"average_execution_time"`
}

// SecurityManager 安全管理器
type SecurityManager struct {
	EnableAuth   bool     `json:"enable_auth"`
	AllowedRoles []string `json:"allowed_roles"`
	SandboxMode  bool     `json:"sandbox_mode"`
	EnableAudit  bool     `json:"enable_audit"`
}

// NewEngine 创建工作流引擎
func NewEngine(config *EngineConfig) (*Engine, error) {
	if config == nil {
		config = &EngineConfig{
			MaxConcurrentWorkflows: 100,
			MaxConcurrentNodes:     50,
			DefaultNodeTimeout:     time.Minute * 5,
			DefaultWorkflowTimeout: time.Hour * 2,
			EnableMetrics:          true,
		}
	}

	engine := &Engine{
		config:     config,
		executions: make(map[string]*WorkflowExecution),
		metrics:    &EngineMetrics{},
		security:   &SecurityManager{},
	}

	return engine, nil
}

// SetDependencies 设置依赖
func (e *Engine) SetDependencies(factory AgentFactory, sessionMgr SessionManager, eventBus EventBus) {
	e.agentFactory = factory
	e.sessionManager = sessionMgr
	e.eventBus = eventBus
}

// Execute 执行工作流
func (e *Engine) Execute(ctx context.Context, workflowID string, inputs map[string]interface{}) (*WorkflowResult, error) {
	// 加载工作流定义
	def, err := e.loadWorkflowDefinition(workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to load workflow definition: %w", err)
	}

	// 验证输入
	if err := e.validateInputs(def, inputs); err != nil {
		return nil, fmt.Errorf("invalid inputs: %w", err)
	}

	// 创建执行实例
	execution, err := e.createExecution(def, inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	// 启动执行
	go e.executeWorkflow(execution)

	// 等待完成
	select {
	case <-execution.done:
		return execution.buildResult(), nil
	case <-ctx.Done():
		e.cancelExecution(execution.ID)
		return nil, ctx.Err()
	}
}

// ExecuteAsync 异步执行工作流
func (e *Engine) ExecuteAsync(ctx context.Context, workflowID string, inputs map[string]interface{}) (string, error) {
	// 加载工作流定义
	def, err := e.loadWorkflowDefinition(workflowID)
	if err != nil {
		return "", fmt.Errorf("failed to load workflow definition: %w", err)
	}

	// 验证输入
	if err := e.validateInputs(def, inputs); err != nil {
		return "", fmt.Errorf("invalid inputs: %w", err)
	}

	// 创建执行实例
	execution, err := e.createExecution(def, inputs)
	if err != nil {
		return "", fmt.Errorf("failed to create execution: %w", err)
	}

	// 启动执行
	go e.executeWorkflow(execution)

	return execution.ID, nil
}

// GetExecution 获取执行状态
func (e *Engine) GetExecution(executionID string) (*WorkflowExecution, error) {
	e.executionsMu.RLock()
	defer e.executionsMu.RUnlock()

	execution, exists := e.executions[executionID]
	if !exists {
		return nil, fmt.Errorf("execution not found: %s", executionID)
	}

	return execution, nil
}

// CancelExecution 取消执行
func (e *Engine) CancelExecution(executionID string) error {
	e.executionsMu.RLock()
	_, exists := e.executions[executionID]
	e.executionsMu.RUnlock()

	if !exists {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	e.cancelExecution(executionID)
	return nil
}

// PauseExecution 暂停执行
func (e *Engine) PauseExecution(executionID string) error {
	e.executionsMu.RLock()
	execution, exists := e.executions[executionID]
	e.executionsMu.RUnlock()

	if !exists {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	execution.mu.Lock()
	defer execution.mu.Unlock()

	if execution.Status != StatusRunning {
		return fmt.Errorf("execution is not running: %s", execution.Status)
	}

	execution.Status = StatusPaused
	return nil
}

// ResumeExecution 恢复执行
func (e *Engine) ResumeExecution(executionID string) error {
	e.executionsMu.RLock()
	execution, exists := e.executions[executionID]
	e.executionsMu.RUnlock()

	if !exists {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	execution.mu.Lock()
	defer execution.mu.Unlock()

	if execution.Status != StatusPaused {
		return fmt.Errorf("execution is not paused: %s", execution.Status)
	}

	execution.Status = StatusRunning

	// 恢复执行
	go e.continueExecution(execution)

	return nil
}

// ListExecutions 列出执行记录
func (e *Engine) ListExecutions(workflowID string, status WorkflowStatus, limit int) ([]*WorkflowExecution, error) {
	e.executionsMu.RLock()
	defer e.executionsMu.RUnlock()

	var executions []*WorkflowExecution
	count := 0

	for _, execution := range e.executions {
		if workflowID != "" && execution.WorkflowID != workflowID {
			continue
		}
		if status != "" && execution.Status != status {
			continue
		}
		if limit > 0 && count >= limit {
			break
		}

		executions = append(executions, execution)
		count++
	}

	return executions, nil
}

// GetMetrics 获取引擎指标
func (e *Engine) GetMetrics() *EngineMetrics {
	e.executionsMu.RLock()
	defer e.executionsMu.RUnlock()

	metrics := *e.metrics
	metrics.RunningExecutions = int64(len(e.executions))

	return &metrics
}

// 私有方法

// loadWorkflowDefinition 加载工作流定义
func (e *Engine) loadWorkflowDefinition(workflowID string) (*WorkflowDefinition, error) {
	// TODO: 实现从存储加载工作流定义
	// 现在返回一个示例定义
	return &WorkflowDefinition{
		ID:   workflowID,
		Name: "Example Workflow",
		Nodes: []NodeDef{
			{
				ID:       "start",
				Name:     "Start",
				Type:     NodeTypeStart,
				Position: Position{X: 0, Y: 0},
			},
			{
				ID:       "end",
				Name:     "End",
				Type:     NodeTypeEnd,
				Position: Position{X: 100, Y: 0},
			},
		},
		Edges: []EdgeDef{
			{
				ID:   "edge1",
				From: "start",
				To:   "end",
			},
		},
	}, nil
}

// validateInputs 验证输入
func (e *Engine) validateInputs(def *WorkflowDefinition, inputs map[string]interface{}) error {
	for _, inputDef := range def.Inputs {
		if inputDef.Required {
			if _, exists := inputs[inputDef.Name]; !exists {
				return fmt.Errorf("required input missing: %s", inputDef.Name)
			}
		}
	}
	return nil
}

// createExecution 创建执行实例
func (e *Engine) createExecution(def *WorkflowDefinition, inputs map[string]interface{}) (*WorkflowExecution, error) {
	executionID := generateExecutionID()

	ctx, cancel := context.WithCancel(context.Background())

	// 创建会话
	session, err := e.sessionManager.CreateSession(ctx, def.ID)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// 创建工作流上下文
	workflowCtx := &WorkflowContext{
		WorkflowID:  def.ID,
		ExecutionID: executionID,
		StartTime:   time.Now(),
		Status:      StatusPending,
		Variables:   make(map[string]interface{}),
		Inputs:      inputs,
		Outputs:     make(map[string]interface{}),
		Completed:   make(map[string]bool),
		Failed:      make(map[string]error),
		Metadata:    make(map[string]interface{}),
		Context:     ctx,
		Session:     session,
	}

	execution := &WorkflowExecution{
		ID:             executionID,
		WorkflowID:     def.ID,
		Definition:     def,
		Status:         StatusPending,
		Context:        workflowCtx,
		CurrentNodes:   []string{},
		CompletedNodes: make(map[string]bool),
		FailedNodes:    make(map[string]error),
		NodeResults:    make(map[string]*NodeResult),
		StartTime:      time.Now(),
		LastActivity:   time.Now(),
		cancelFunc:     cancel,
		done:           make(chan struct{}),
		Errors:         make([]WorkflowError, 0),
		Warnings:       make([]string, 0),
	}

	// 注册执行
	e.executionsMu.Lock()
	e.executions[executionID] = execution
	e.executionsMu.Unlock()

	// 发布开始事件
	if e.eventBus != nil {
		_ = e.eventBus.Publish(ctx, &WorkflowEvent{
			Type:        "workflow.started",
			ExecutionID: executionID,
			NodeID:      "",
			Timestamp:   time.Now(),
			Data: map[string]interface{}{
				"workflow_id": def.ID,
			},
		})
	}

	// 更新指标
	e.metrics.TotalExecutions++

	return execution, nil
}

// executeWorkflow 执行工作流
func (e *Engine) executeWorkflow(execution *WorkflowExecution) {
	defer close(execution.done)

	// 设置运行状态
	execution.mu.Lock()
	execution.Status = StatusRunning
	execution.Context.Status = StatusRunning
	execution.mu.Unlock()

	// 查找开始节点
	startNodes := e.findStartNodes(execution.Definition)
	if len(startNodes) == 0 {
		e.markExecutionFailed(execution, fmt.Errorf("no start node found"))
		return
	}

	// 开始执行
	e.executeNodes(execution, startNodes)
}

// executeNodes 执行节点
func (e *Engine) executeNodes(execution *WorkflowExecution, nodeIDs []string) {
	execution.mu.Lock()
	execution.CurrentNodes = nodeIDs
	execution.LastActivity = time.Now()
	execution.mu.Unlock()

	for _, nodeID := range nodeIDs {
		if !e.executeNode(execution, nodeID) {
			return // 节点执行失败
		}
	}

	// 查找下一批节点
	nextNodes := e.findNextNodes(execution, nodeIDs)
	if len(nextNodes) > 0 {
		e.executeNodes(execution, nextNodes)
	} else {
		// 没有更多节点，工作流完成
		e.markExecutionCompleted(execution)
	}
}

// executeNode 执行单个节点
func (e *Engine) executeNode(execution *WorkflowExecution, nodeID string) bool {
	node := e.findNode(execution.Definition, nodeID)
	if node == nil {
		e.markExecutionFailed(execution, fmt.Errorf("node not found: %s", nodeID))
		return false
	}

	// 检查节点是否已完成
	if execution.CompletedNodes[nodeID] {
		return true
	}

	// 检查节点是否失败
	if _, failed := execution.FailedNodes[nodeID]; failed {
		return false
	}

	// 创建节点结果
	result := &NodeResult{
		NodeID:    nodeID,
		NodeName:  node.Name,
		NodeType:  node.Type,
		Status:    StatusRunning,
		StartTime: time.Now(),
		Inputs:    make(map[string]interface{}),
		Outputs:   make(map[string]interface{}),
		Metadata:  make(map[string]interface{}),
	}

	// 执行节点
	var err error
	switch node.Type {
	case NodeTypeStart:
		err = e.executeStartNode(execution, node, result)
	case NodeTypeEnd:
		err = e.executeEndNode(execution, node, result)
	case NodeTypeTask:
		err = e.executeTaskNode(execution, node, result)
	case NodeTypeCondition:
		err = e.executeConditionNode(execution, node, result)
	case NodeTypeLoop:
		err = e.executeLoopNode(execution, node, result)
	case NodeTypeParallel:
		err = e.executeParallelNode(execution, node, result)
	case NodeTypeMerge:
		err = e.executeMergeNode(execution, node, result)
	default:
		err = fmt.Errorf("unsupported node type: %s", node.Type)
	}

	// 更新结果
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	if err != nil {
		result.Status = StatusFailed
		result.Error = err.Error()
		execution.FailedNodes[nodeID] = err
		execution.Errors = append(execution.Errors, WorkflowError{
			NodeID:    nodeID,
			NodeName:  node.Name,
			Error:     err.Error(),
			Timestamp: time.Now(),
			Retryable: true,
		})
		return false
	}

	result.Status = StatusCompleted
	execution.CompletedNodes[nodeID] = true
	execution.NodeResults[nodeID] = result

	// 发布节点完成事件
	if e.eventBus != nil {
		_ = e.eventBus.Publish(execution.Context.Context, &WorkflowEvent{
			Type:        "node.completed",
			ExecutionID: execution.ID,
			NodeID:      nodeID,
			Timestamp:   time.Now(),
			Data: map[string]interface{}{
				"node_type": node.Type,
				"duration":  result.Duration,
			},
		})
	}

	return true
}

// executeStartNode 执行开始节点
func (e *Engine) executeStartNode(execution *WorkflowExecution, node *NodeDef, result *NodeResult) error {
	// 开始节点主要是初始化
	result.Outputs["started_at"] = time.Now()
	return nil
}

// executeEndNode 执行结束节点
func (e *Engine) executeEndNode(execution *WorkflowExecution, node *NodeDef, result *NodeResult) error {
	// 收集工作流输出
	for key, value := range execution.Context.Variables {
		execution.Context.Outputs[key] = value
	}

	result.Outputs["completed_at"] = time.Now()
	result.Outputs["outputs"] = execution.Context.Outputs
	return nil
}

// executeTaskNode 执行任务节点
func (e *Engine) executeTaskNode(execution *WorkflowExecution, node *NodeDef, result *NodeResult) error {
	if node.Agent == nil {
		return fmt.Errorf("task node requires agent configuration")
	}

	// 创建Agent
	agent, err := e.agentFactory.CreateAgent(execution.Context.Context, node.Agent, node.Config)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	// 准备输入消息
	inputMessage, err := e.prepareInputMessage(execution, node)
	if err != nil {
		return fmt.Errorf("failed to prepare input: %w", err)
	}

	// 设置超时
	ctx := execution.Context.Context
	if node.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, node.Timeout)
		defer cancel()
	}

	// 执行Agent
	for event, err := range agent.Execute(ctx, inputMessage) {
		if err != nil {
			return fmt.Errorf("agent execution failed: %w", err)
		}

		if event != nil {
			// 处理事件
			result.Outputs = e.processAgentEvent(event, result.Outputs)
		}
	}

	return nil
}

// executeConditionNode 执行条件节点
func (e *Engine) executeConditionNode(execution *WorkflowExecution, node *NodeDef, result *NodeResult) error {
	if node.Condition == nil {
		return fmt.Errorf("condition node requires condition configuration")
	}

	// 评估条件
	evaluated, err := e.evaluateCondition(execution, node.Condition)
	if err != nil {
		return fmt.Errorf("failed to evaluate condition: %w", err)
	}

	result.Outputs["condition_result"] = evaluated
	return nil
}

// executeLoopNode 执行循环节点
func (e *Engine) executeLoopNode(execution *WorkflowExecution, node *NodeDef, result *NodeResult) error {
	if node.Loop == nil {
		return fmt.Errorf("loop node requires loop configuration")
	}

	// TODO: 实现循环逻辑
	result.Outputs["loop_completed"] = true
	return nil
}

// executeParallelNode 执行并行节点
func (e *Engine) executeParallelNode(execution *WorkflowExecution, node *NodeDef, result *NodeResult) error {
	if node.Parallel == nil {
		return fmt.Errorf("parallel node requires parallel configuration")
	}

	// TODO: 实现并行逻辑
	result.Outputs["parallel_completed"] = true
	return nil
}

// executeMergeNode 执行合并节点
func (e *Engine) executeMergeNode(execution *WorkflowExecution, node *NodeDef, result *NodeResult) error {
	// 合并节点主要用于汇聚多个分支
	result.Outputs["merge_completed"] = true
	return nil
}

// 辅助方法

// findStartNodes 查找开始节点
func (e *Engine) findStartNodes(def *WorkflowDefinition) []string {
	var startNodes []string
	for _, node := range def.Nodes {
		if node.Type == NodeTypeStart {
			startNodes = append(startNodes, node.ID)
		}
	}
	return startNodes
}

// findNode 查找节点
func (e *Engine) findNode(def *WorkflowDefinition, nodeID string) *NodeDef {
	for _, node := range def.Nodes {
		if node.ID == nodeID {
			return &node
		}
	}
	return nil
}

// findNextNodes 查找下一批节点
func (e *Engine) findNextNodes(execution *WorkflowExecution, currentNodes []string) []string {
	var nextNodes []string
	completed := make(map[string]bool)

	// 将当前节点标记为已完成
	for _, nodeID := range currentNodes {
		completed[nodeID] = true
		execution.CompletedNodes[nodeID] = true
	}

	// 查找所有指向已完成节点的边
	for _, edge := range execution.Definition.Edges {
		if completed[edge.From] {
			// 检查所有前置节点是否都已完成
			if e.canExecuteNode(execution, edge.To, edge.Condition) {
				nextNodes = append(nextNodes, edge.To)
			}
		}
	}

	return nextNodes
}

// canExecuteNode 检查节点是否可以执行
func (e *Engine) canExecuteNode(execution *WorkflowExecution, nodeID string, condition string) bool {
	// 检查节点是否已完成
	if execution.CompletedNodes[nodeID] {
		return false
	}

	// 检查节点是否失败
	if _, failed := execution.FailedNodes[nodeID]; failed {
		return false
	}

	// 检查边条件
	if condition != "" {
		evaluator := NewExpressionEvaluator(execution.Context.Variables)
		result, err := evaluator.EvaluateBool(condition)
		if err != nil {
			return false
		}
		return result
	}

	return true
}

// prepareInputMessage 准备输入消息
func (e *Engine) prepareInputMessage(execution *WorkflowExecution, node *NodeDef) (string, error) {
	if node.Agent != nil && node.Agent.Inputs != nil {
		// 使用输入映射构建消息
		var parts []string
		for key, varPath := range node.Agent.Inputs {
			if value, exists := execution.Context.Variables[varPath]; exists {
				parts = append(parts, fmt.Sprintf("%s: %v", key, value))
			}
		}
		return strings.Join(parts, "\n"), nil
	}

	// 使用工作流输入作为消息
	if len(execution.Context.Inputs) > 0 {
		data, _ := json.Marshal(execution.Context.Inputs)
		return string(data), nil
	}

	return "", nil
}

// processAgentEvent 处理Agent事件
func (e *Engine) processAgentEvent(event *session.Event, outputs map[string]interface{}) map[string]interface{} {
	if event.Content.Content != "" {
		outputs["content"] = event.Content.Content
	}

	if event.Metadata != nil {
		for key, value := range event.Metadata {
			outputs[key] = value
		}
	}

	return outputs
}

// evaluateCondition 评估条件
func (e *Engine) evaluateCondition(execution *WorkflowExecution, condition *ConditionDef) (bool, error) {
	evaluator := NewExpressionEvaluator(execution.Context.Variables)

	switch condition.Type {
	case ConditionTypeAnd:
		for _, rule := range condition.Rules {
			if result, err := e.evaluateConditionRule(execution, rule, evaluator); err != nil {
				return false, err
			} else if !result {
				return false, nil
			}
		}
		return true, nil
	case ConditionTypeOr:
		for _, rule := range condition.Rules {
			if result, err := e.evaluateConditionRule(execution, rule, evaluator); err != nil {
				return false, err
			} else if result {
				return true, nil
			}
		}
		return false, nil
	case ConditionTypeCustom:
		return evaluator.EvaluateBool(condition.Custom)
	default:
		return false, fmt.Errorf("unsupported condition type: %s", condition.Type)
	}
}

// evaluateConditionRule 评估条件规则
func (e *Engine) evaluateConditionRule(execution *WorkflowExecution, rule ConditionRule, evaluator *ExpressionEvaluator) (bool, error) {
	// 构建表达式
	expression := fmt.Sprintf("%s %s %v", rule.Variable, rule.Operator, rule.Value)
	return evaluator.EvaluateBool(expression)
}

// markExecutionCompleted 标记执行完成
func (e *Engine) markExecutionCompleted(execution *WorkflowExecution) {
	execution.mu.Lock()
	defer execution.mu.Unlock()

	execution.Status = StatusCompleted
	execution.Context.Status = StatusCompleted
	execution.EndTime = time.Now()

	// 更新指标
	e.metrics.CompletedExecutions++

	// 发布完成事件
	if e.eventBus != nil {
		_ = e.eventBus.Publish(execution.Context.Context, &WorkflowEvent{
			Type:        "workflow.completed",
			ExecutionID: execution.ID,
			NodeID:      "",
			Timestamp:   time.Now(),
			Data: map[string]interface{}{
				"duration": execution.EndTime.Sub(execution.StartTime),
			},
		})
	}
}

// markExecutionFailed 标记执行失败
func (e *Engine) markExecutionFailed(execution *WorkflowExecution, err error) {
	execution.mu.Lock()
	defer execution.mu.Unlock()

	execution.Status = StatusFailed
	execution.Context.Status = StatusFailed
	execution.EndTime = time.Now()

	// 记录错误
	execution.Errors = append(execution.Errors, WorkflowError{
		NodeID:    "",
		NodeName:  "",
		Error:     err.Error(),
		Timestamp: time.Now(),
		Retryable: false,
	})

	// 更新指标
	e.metrics.FailedExecutions++

	// 发布失败事件
	if e.eventBus != nil {
		_ = e.eventBus.Publish(execution.Context.Context, &WorkflowEvent{
			Type:        "workflow.failed",
			ExecutionID: execution.ID,
			NodeID:      "",
			Timestamp:   time.Now(),
			Data: map[string]interface{}{
				"error": err.Error(),
			},
		})
	}
}

// cancelExecution 取消执行
func (e *Engine) cancelExecution(executionID string) {
	e.executionsMu.RLock()
	execution, exists := e.executions[executionID]
	e.executionsMu.RUnlock()

	if !exists {
		return
	}

	execution.mu.Lock()
	defer execution.mu.Unlock()

	if execution.Status != StatusRunning && execution.Status != StatusPaused {
		return
	}

	execution.cancelFunc()
	execution.Status = StatusCancelled
	execution.Context.Status = StatusCancelled
	execution.EndTime = time.Now()

	// 发布取消事件
	if e.eventBus != nil {
		_ = e.eventBus.Publish(execution.Context.Context, &WorkflowEvent{
			Type:        "workflow.cancelled",
			ExecutionID: executionID,
			NodeID:      "",
			Timestamp:   time.Now(),
		})
	}
}

// continueExecution 继续执行
func (e *Engine) continueExecution(execution *WorkflowExecution) {
	e.executeWorkflow(execution)
}

// buildResult 构建执行结果
func (execution *WorkflowExecution) buildResult() *WorkflowResult {
	execution.mu.RLock()
	defer execution.mu.RUnlock()

	// 构建步骤跟踪
	var trace []WorkflowStep
	for _, result := range execution.NodeResults {
		step := WorkflowStep{
			NodeID:     result.NodeID,
			NodeName:   result.NodeName,
			NodeType:   result.NodeType,
			Status:     result.Status,
			StartTime:  result.StartTime,
			EndTime:    result.EndTime,
			Duration:   result.Duration,
			Inputs:     result.Inputs,
			Outputs:    result.Outputs,
			Error:      result.Error,
			RetryCount: result.RetryCount,
			Metadata:   result.Metadata,
		}
		trace = append(trace, step)
	}

	return &WorkflowResult{
		ExecutionID: execution.ID,
		WorkflowID:  execution.WorkflowID,
		Status:      execution.Status,
		StartTime:   execution.StartTime,
		EndTime:     execution.EndTime,
		Duration:    execution.EndTime.Sub(execution.StartTime),
		Outputs:     execution.Context.Outputs,
		Errors:      execution.Errors,
		Trace:       trace,
	}
}

// generateExecutionID 生成执行ID
func generateExecutionID() string {
	return fmt.Sprintf("exec_%d", time.Now().UnixNano())
}
