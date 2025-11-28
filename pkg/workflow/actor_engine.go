package workflow

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/actor"
	pkgagent "github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/types"
)

// ActorEngine Actor 化的工作流引擎
// 使用 Actor 模型管理多 Agent 协作
type ActorEngine struct {
	// 嵌入原有引擎（保持兼容）
	*Engine

	// Actor 系统
	actorSystem *actor.System

	// Agent PIDs 映射
	agentPIDs   map[string]*actor.PID // nodeID/stepID -> Agent PID
	agentPIDsMu sync.RWMutex

	// 协调器 Actor
	coordinatorPID *actor.PID

	// 配置
	actorConfig *ActorEngineConfig
}

// ActorEngineConfig Actor 引擎配置
type ActorEngineConfig struct {
	// SystemName Actor 系统名称
	SystemName string

	// DefaultTimeout 默认超时时间
	DefaultTimeout time.Duration

	// MaxConcurrentAgents 最大并发 Agent 数
	MaxConcurrentAgents int

	// SupervisorStrategy Agent 监督策略
	SupervisorStrategy actor.SupervisorStrategy

	// EnableMetrics 启用指标收集
	EnableMetrics bool
}

// DefaultActorEngineConfig 默认配置
func DefaultActorEngineConfig() *ActorEngineConfig {
	return &ActorEngineConfig{
		SystemName:          "workflow",
		DefaultTimeout:      5 * time.Minute,
		MaxConcurrentAgents: 10,
		SupervisorStrategy:  actor.NewOneForOneStrategy(3, time.Minute, actor.DefaultDecider),
		EnableMetrics:       true,
	}
}

// NewActorEngine 创建 Actor 化的工作流引擎
func NewActorEngine(config *EngineConfig, actorConfig *ActorEngineConfig) (*ActorEngine, error) {
	// 创建基础引擎
	engine, err := NewEngine(config)
	if err != nil {
		return nil, err
	}

	if actorConfig == nil {
		actorConfig = DefaultActorEngineConfig()
	}

	// 创建 Actor 系统
	systemConfig := actor.DefaultSystemConfig()
	systemConfig.MailboxSize = 10000
	actorSystem := actor.NewSystemWithConfig(actorConfig.SystemName, systemConfig)

	ae := &ActorEngine{
		Engine:      engine,
		actorSystem: actorSystem,
		agentPIDs:   make(map[string]*actor.PID),
		actorConfig: actorConfig,
	}

	// 创建协调器 Actor
	coordinator := NewCoordinatorActor(ae)
	ae.coordinatorPID = actorSystem.Spawn(coordinator, "coordinator")

	log.Printf("[ActorEngine] Created with actor system: %s", actorConfig.SystemName)
	return ae, nil
}

// ExecuteWithActors 使用 Actor 模型执行工作流
func (e *ActorEngine) ExecuteWithActors(ctx context.Context, workflowID string, inputs map[string]interface{}) (*WorkflowResult, error) {
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

	// 使用协调器执行
	resultCh := make(chan *WorkflowResult, 1)
	errCh := make(chan error, 1)

	e.coordinatorPID.Tell(&ExecuteWorkflowMsg{
		Execution: execution,
		ResultCh:  resultCh,
		ErrorCh:   errCh,
	})

	// 等待完成
	select {
	case result := <-resultCh:
		return result, nil
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		e.CancelExecution(execution.ID)
		return nil, ctx.Err()
	}
}

// SpawnAgent 在 Actor 系统中创建 Agent
func (e *ActorEngine) SpawnAgent(ctx context.Context, nodeID string, agentConfig *types.AgentConfig, deps *pkgagent.Dependencies) (*actor.PID, error) {
	// 创建 Agent
	agent, err := pkgagent.Create(ctx, agentConfig, deps)
	if err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}

	// 包装为 Actor
	agentActor := pkgagent.NewAgentActor(agent)

	// 设置属性
	props := &actor.Props{
		Name:               fmt.Sprintf("agent-%s", nodeID),
		MailboxSize:        100,
		SupervisorStrategy: e.actorConfig.SupervisorStrategy,
	}

	// 在 Actor 系统中启动
	pid := e.actorSystem.SpawnWithProps(agentActor, props)

	// 注册
	e.agentPIDsMu.Lock()
	e.agentPIDs[nodeID] = pid
	e.agentPIDsMu.Unlock()

	log.Printf("[ActorEngine] Spawned agent for node %s: %s", nodeID, pid.ID)
	return pid, nil
}

// GetAgentPID 获取节点对应的 Agent PID
func (e *ActorEngine) GetAgentPID(nodeID string) (*actor.PID, bool) {
	e.agentPIDsMu.RLock()
	defer e.agentPIDsMu.RUnlock()
	pid, ok := e.agentPIDs[nodeID]
	return pid, ok
}

// StopAgent 停止指定节点的 Agent
func (e *ActorEngine) StopAgent(nodeID string) {
	e.agentPIDsMu.Lock()
	pid, ok := e.agentPIDs[nodeID]
	if ok {
		delete(e.agentPIDs, nodeID)
	}
	e.agentPIDsMu.Unlock()

	if ok {
		e.actorSystem.Stop(pid)
	}
}

// Shutdown 关闭引擎
func (e *ActorEngine) Shutdown() {
	// 停止所有 Agent
	e.agentPIDsMu.Lock()
	pids := make([]*actor.PID, 0, len(e.agentPIDs))
	for _, pid := range e.agentPIDs {
		pids = append(pids, pid)
	}
	e.agentPIDs = make(map[string]*actor.PID)
	e.agentPIDsMu.Unlock()

	for _, pid := range pids {
		e.actorSystem.Stop(pid)
	}

	// 关闭 Actor 系统
	e.actorSystem.Shutdown()

	log.Printf("[ActorEngine] Shutdown complete")
}

// Stats 获取统计信息
func (e *ActorEngine) Stats() *ActorEngineStats {
	systemStats := e.actorSystem.Stats()

	e.agentPIDsMu.RLock()
	agentCount := len(e.agentPIDs)
	e.agentPIDsMu.RUnlock()

	return &ActorEngineStats{
		ActiveAgents:  agentCount,
		TotalActors:   systemStats.TotalActors,
		TotalMessages: systemStats.TotalMessages,
		DeadLetters:   systemStats.DeadLetters,
		ProcessedMsgs: systemStats.ProcessedMsgs,
		Uptime:        time.Since(systemStats.StartTime),
	}
}

// ActorEngineStats 引擎统计
type ActorEngineStats struct {
	ActiveAgents  int
	TotalActors   int64
	TotalMessages int64
	DeadLetters   int64
	ProcessedMsgs int64
	Uptime        time.Duration
}

// ============== 协调器 Actor ==============

// CoordinatorActor 工作流协调器 Actor
// 负责协调多个 Agent 的执行
type CoordinatorActor struct {
	engine *ActorEngine

	// 运行中的工作流
	runningWorkflows map[string]*workflowState
	mu               sync.RWMutex
}

type workflowState struct {
	execution *WorkflowExecution
	resultCh  chan *WorkflowResult
	errorCh   chan error
	agents    map[string]*actor.PID
}

// NewCoordinatorActor 创建协调器
func NewCoordinatorActor(engine *ActorEngine) *CoordinatorActor {
	return &CoordinatorActor{
		engine:           engine,
		runningWorkflows: make(map[string]*workflowState),
	}
}

// Receive 处理消息
func (c *CoordinatorActor) Receive(ctx *actor.Context, msg actor.Message) {
	switch m := msg.(type) {
	case *actor.Started:
		log.Printf("[Coordinator] Started")

	case *ExecuteWorkflowMsg:
		c.handleExecuteWorkflow(ctx, m)

	case *NodeCompletedMsg:
		c.handleNodeCompleted(ctx, m)

	case *NodeFailedMsg:
		c.handleNodeFailed(ctx, m)

	case *CancelWorkflowMsg:
		c.handleCancelWorkflow(ctx, m)

	case *actor.Terminated:
		c.handleTerminated(ctx, m)

	case *actor.Stopping:
		c.handleStopping(ctx)
	}
}

// handleExecuteWorkflow 处理工作流执行请求
func (c *CoordinatorActor) handleExecuteWorkflow(ctx *actor.Context, msg *ExecuteWorkflowMsg) {
	execution := msg.Execution

	// 创建工作流状态
	state := &workflowState{
		execution: execution,
		resultCh:  msg.ResultCh,
		errorCh:   msg.ErrorCh,
		agents:    make(map[string]*actor.PID),
	}

	c.mu.Lock()
	c.runningWorkflows[execution.ID] = state
	c.mu.Unlock()

	// 开始执行
	go c.executeWorkflowAsync(execution, state)
}

// executeWorkflowAsync 异步执行工作流
func (c *CoordinatorActor) executeWorkflowAsync(execution *WorkflowExecution, state *workflowState) {
	execution.mu.Lock()
	execution.Status = StatusRunning
	execution.mu.Unlock()

	// 查找开始节点
	startNodes := c.findStartNodes(execution.Definition)
	if len(startNodes) == 0 {
		state.errorCh <- fmt.Errorf("no start node found")
		return
	}

	// 执行节点
	if err := c.executeNodes(execution, state, startNodes); err != nil {
		state.errorCh <- err
		return
	}

	// 构建结果
	result := execution.buildResult()
	state.resultCh <- result

	// 清理
	c.mu.Lock()
	delete(c.runningWorkflows, execution.ID)
	c.mu.Unlock()
}

// executeNodes 执行节点列表
func (c *CoordinatorActor) executeNodes(execution *WorkflowExecution, state *workflowState, nodeIDs []string) error {
	for _, nodeID := range nodeIDs {
		node := c.findNode(execution.Definition, nodeID)
		if node == nil {
			return fmt.Errorf("node not found: %s", nodeID)
		}

		// 根据节点类型执行
		switch node.Type {
		case NodeTypeTask:
			if err := c.executeTaskNodeWithActor(execution, state, node); err != nil {
				return err
			}
		case NodeTypeParallel:
			if err := c.executeParallelNodeWithActors(execution, state, node); err != nil {
				return err
			}
		default:
			// 使用原有引擎处理其他类型
			if !c.engine.executeNode(execution, nodeID) {
				return fmt.Errorf("node execution failed: %s", nodeID)
			}
		}

		execution.CompletedNodes[nodeID] = true
	}

	// 查找下一批节点
	nextNodes := c.findNextNodes(execution, nodeIDs)
	if len(nextNodes) > 0 {
		return c.executeNodes(execution, state, nextNodes)
	}

	return nil
}

// executeTaskNodeWithActor 使用 Actor 执行任务节点
func (c *CoordinatorActor) executeTaskNodeWithActor(execution *WorkflowExecution, state *workflowState, node *NodeDef) error {
	if node.Agent == nil {
		return fmt.Errorf("task node requires agent configuration")
	}

	// 获取或创建 Agent PID
	pid, ok := c.engine.GetAgentPID(node.ID)
	if !ok {
		// 需要创建 Agent - 这里简化处理，实际应该从 AgentFactory 创建
		return fmt.Errorf("agent not found for node: %s", node.ID)
	}

	// 准备输入
	inputMessage, err := c.prepareInputMessage(execution, node)
	if err != nil {
		return err
	}

	// 通过 Actor 消息执行
	replyCh := make(chan *pkgagent.ChatResultMsg, 1)
	pid.Tell(&pkgagent.ChatMsg{
		Text:    inputMessage,
		Ctx:     execution.Context.Context,
		ReplyTo: replyCh,
	})

	// 等待结果
	timeout := node.Timeout
	if timeout == 0 {
		timeout = c.engine.actorConfig.DefaultTimeout
	}

	select {
	case result := <-replyCh:
		if result.Error != nil {
			return result.Error
		}
		// 保存结果
		execution.NodeResults[node.ID] = &NodeResult{
			NodeID:   node.ID,
			NodeName: node.Name,
			NodeType: node.Type,
			Status:   StatusCompleted,
			Outputs: map[string]interface{}{
				"content": result.Result.Text,
			},
		}
		return nil

	case <-time.After(timeout):
		return fmt.Errorf("task node %s timed out", node.ID)

	case <-execution.Context.Context.Done():
		return execution.Context.Context.Err()
	}
}

// executeParallelNodeWithActors 使用 Actor 执行并行节点
func (c *CoordinatorActor) executeParallelNodeWithActors(execution *WorkflowExecution, state *workflowState, node *NodeDef) error {
	if node.Parallel == nil || len(node.Parallel.Branches) == 0 {
		return fmt.Errorf("parallel node requires branches")
	}

	var wg sync.WaitGroup
	results := make(chan *parallelBranchResult, len(node.Parallel.Branches))
	errChan := make(chan error, len(node.Parallel.Branches))

	for _, branch := range node.Parallel.Branches {
		wg.Add(1)
		go func(b NodeRef) {
			defer wg.Done()

			// 为每个分支创建或获取 Agent
			branchID := fmt.Sprintf("%s-%s", node.ID, b.ID)
			pid, ok := c.engine.GetAgentPID(branchID)
			if !ok {
				errChan <- fmt.Errorf("agent not found for branch: %s", branchID)
				return
			}

			// 执行 - 使用 NodeRef 的 Name 作为输入
			replyCh := make(chan *pkgagent.ChatResultMsg, 1)
			pid.Tell(&pkgagent.ChatMsg{
				Text:    b.Name, // 使用 Name 作为输入
				Ctx:     execution.Context.Context,
				ReplyTo: replyCh,
			})

			select {
			case result := <-replyCh:
				results <- &parallelBranchResult{
					branchID: b.ID,
					result:   result,
				}
			case <-time.After(c.engine.actorConfig.DefaultTimeout):
				errChan <- fmt.Errorf("branch %s timed out", b.ID)
			}
		}(branch)
	}

	// 等待所有分支完成
	go func() {
		wg.Wait()
		close(results)
		close(errChan)
	}()

	// 收集结果
	branchResults := make(map[string]interface{})
	for result := range results {
		if result.result.Error == nil {
			branchResults[result.branchID] = result.result.Result.Text
		}
	}

	// 检查错误
	var firstError error
	for err := range errChan {
		if firstError == nil {
			firstError = err
		}
	}

	// 根据 JoinType 决定是否返回错误
	if firstError != nil && node.Parallel.JoinType == JoinTypeWait {
		// 只有 JoinTypeWait 时需要所有分支成功
		return firstError
	}

	// 保存结果
	execution.NodeResults[node.ID] = &NodeResult{
		NodeID:   node.ID,
		NodeName: node.Name,
		NodeType: node.Type,
		Status:   StatusCompleted,
		Outputs: map[string]interface{}{
			"branches": branchResults,
		},
	}

	return nil
}

type parallelBranchResult struct {
	branchID string
	result   *pkgagent.ChatResultMsg
}

// handleNodeCompleted 处理节点完成
func (c *CoordinatorActor) handleNodeCompleted(ctx *actor.Context, msg *NodeCompletedMsg) {
	log.Printf("[Coordinator] Node completed: %s in execution %s", msg.NodeID, msg.ExecutionID)
}

// handleNodeFailed 处理节点失败
func (c *CoordinatorActor) handleNodeFailed(ctx *actor.Context, msg *NodeFailedMsg) {
	log.Printf("[Coordinator] Node failed: %s in execution %s: %v", msg.NodeID, msg.ExecutionID, msg.Error)

	c.mu.RLock()
	state, ok := c.runningWorkflows[msg.ExecutionID]
	c.mu.RUnlock()

	if ok && state.errorCh != nil {
		state.errorCh <- msg.Error
	}
}

// handleCancelWorkflow 处理取消工作流
func (c *CoordinatorActor) handleCancelWorkflow(ctx *actor.Context, msg *CancelWorkflowMsg) {
	c.mu.Lock()
	state, ok := c.runningWorkflows[msg.ExecutionID]
	if ok {
		delete(c.runningWorkflows, msg.ExecutionID)
	}
	c.mu.Unlock()

	if ok {
		// 停止所有相关 Agent
		for _, pid := range state.agents {
			c.engine.actorSystem.Stop(pid)
		}

		// 取消执行上下文
		if state.execution.cancelFunc != nil {
			state.execution.cancelFunc()
		}
	}
}

// handleTerminated 处理 Actor 终止通知
func (c *CoordinatorActor) handleTerminated(ctx *actor.Context, msg *actor.Terminated) {
	log.Printf("[Coordinator] Actor terminated: %s", msg.Who.ID)
}

// handleStopping 处理停止
func (c *CoordinatorActor) handleStopping(ctx *actor.Context) {
	// 取消所有运行中的工作流
	c.mu.Lock()
	for _, state := range c.runningWorkflows {
		if state.execution.cancelFunc != nil {
			state.execution.cancelFunc()
		}
	}
	c.runningWorkflows = make(map[string]*workflowState)
	c.mu.Unlock()
}

// 辅助方法
func (c *CoordinatorActor) findStartNodes(def *WorkflowDefinition) []string {
	var startNodes []string
	for _, node := range def.Nodes {
		if node.Type == NodeTypeStart {
			startNodes = append(startNodes, node.ID)
		}
	}
	return startNodes
}

func (c *CoordinatorActor) findNode(def *WorkflowDefinition, nodeID string) *NodeDef {
	for _, node := range def.Nodes {
		if node.ID == nodeID {
			return &node
		}
	}
	return nil
}

func (c *CoordinatorActor) findNextNodes(execution *WorkflowExecution, currentNodes []string) []string {
	var nextNodes []string
	completed := make(map[string]bool)
	for _, nodeID := range currentNodes {
		completed[nodeID] = true
	}

	for _, edge := range execution.Definition.Edges {
		if completed[edge.From] && !execution.CompletedNodes[edge.To] {
			nextNodes = append(nextNodes, edge.To)
		}
	}

	return nextNodes
}

func (c *CoordinatorActor) prepareInputMessage(execution *WorkflowExecution, node *NodeDef) (string, error) {
	// 简化实现
	if len(execution.Context.Inputs) > 0 {
		if input, ok := execution.Context.Inputs["message"].(string); ok {
			return input, nil
		}
	}
	return "", nil
}

// ============== 协调器消息类型 ==============

// ExecuteWorkflowMsg 执行工作流请求
type ExecuteWorkflowMsg struct {
	Execution *WorkflowExecution
	ResultCh  chan *WorkflowResult
	ErrorCh   chan error
}

func (m *ExecuteWorkflowMsg) Kind() string { return "coordinator.execute_workflow" }

// NodeCompletedMsg 节点完成通知
type NodeCompletedMsg struct {
	ExecutionID string
	NodeID      string
	Result      interface{}
}

func (m *NodeCompletedMsg) Kind() string { return "coordinator.node_completed" }

// NodeFailedMsg 节点失败通知
type NodeFailedMsg struct {
	ExecutionID string
	NodeID      string
	Error       error
}

func (m *NodeFailedMsg) Kind() string { return "coordinator.node_failed" }

// CancelWorkflowMsg 取消工作流请求
type CancelWorkflowMsg struct {
	ExecutionID string
}

func (m *CancelWorkflowMsg) Kind() string { return "coordinator.cancel_workflow" }
