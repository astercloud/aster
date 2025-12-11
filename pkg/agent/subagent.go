package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/types"
	"github.com/google/uuid"
)

var subagentLog = logging.ForComponent("SubAgent")

// SubAgentManager 子 Agent 管理器
// 负责创建、执行和管理子 Agent 的生命周期
type SubAgentManager struct {
	mu   sync.RWMutex
	deps *Dependencies

	// 运行中的子 Agent
	running map[string]*SubAgentHandle

	// 子 Agent 规格注册表
	specs map[string]*types.SubAgentSpec

	// 默认配置
	defaultTimeout time.Duration
	maxDepth       int
}

// SubAgentHandle 子 Agent 句柄
type SubAgentHandle struct {
	ID            string
	AgentType     string
	Agent         *Agent
	Request       *types.SubAgentRequest
	StartTime     time.Time
	Status        string // "running", "completed", "failed", "cancelled"
	Result        *types.SubAgentResult
	CancelFunc    context.CancelFunc
	ProgressChan  chan *types.SubAgentProgressEvent
	ParentAgentID string
}

// NewSubAgentManager 创建子 Agent 管理器
func NewSubAgentManager(deps *Dependencies) *SubAgentManager {
	mgr := &SubAgentManager{
		deps:           deps,
		running:        make(map[string]*SubAgentHandle),
		specs:          make(map[string]*types.SubAgentSpec),
		defaultTimeout: 30 * time.Minute,
		maxDepth:       3,
	}

	// 注册内置子 Agent 规格
	mgr.registerBuiltinSpecs()

	return mgr
}

// registerBuiltinSpecs 注册内置子 Agent 规格
func (m *SubAgentManager) registerBuiltinSpecs() {
	// Explore Agent - 代码探索
	m.RegisterSpec(&types.SubAgentSpec{
		Name:        "Explore",
		Description: "快速探索代码库，搜索文件和代码模式，理解项目结构",
		Prompt: `你是一个代码探索专家。你的任务是快速搜索和分析代码库。

你应该：
1. 使用 Glob 和 Grep 工具快速定位相关文件
2. 使用 Read 工具查看关键代码
3. 总结发现的模式和结构
4. 不要修改任何文件

输出格式：
- 简洁的发现摘要
- 关键文件列表
- 代码模式说明`,
		Tools:     []string{"Read", "Glob", "Grep"},
		Parallel:  true,
		MaxTokens: 50000,
		Timeout:   10 * time.Minute,
	})

	// Plan Agent - 任务规划（增强版，支持 Todo 集成）
	m.RegisterSpec(&types.SubAgentSpec{
		Name:        "Plan",
		Description: "分析任务需求，探索代码库，制定详细的实现计划，并生成可执行的 Todo 列表",
		Prompt: `你是一个任务规划专家。你的任务是分析需求并制定详细的实现计划。

## 工作流程

### 第一阶段：理解需求
1. 仔细阅读任务描述
2. 识别关键目标和约束
3. 如有不明确的地方，列出假设

### 第二阶段：代码探索
1. 使用 Glob 查找相关文件
2. 使用 Grep 搜索关键代码模式
3. 使用 Read 查看核心实现
4. 记录发现的架构模式和依赖关系

### 第三阶段：制定计划
1. 将任务分解为可执行的步骤
2. 为每个步骤估算复杂度（简单/中等/复杂）
3. 识别步骤之间的依赖关系
4. 标记可能的风险点

### 第四阶段：生成 Todo
使用 TodoWrite 工具将计划转换为可追踪的任务列表

## 输出格式

### 任务理解
[简述任务目标和范围]

### 现状分析
- 相关文件：[文件列表]
- 现有模式：[代码模式说明]
- 依赖关系：[依赖说明]

### 实现计划
| 步骤 | 描述 | 复杂度 | 依赖 | 风险 |
|------|------|--------|------|------|
| 1 | [步骤描述] | 简单/中等/复杂 | 无/步骤X | 低/中/高 |

### 风险评估
- [风险1]: [缓解措施]
- [风险2]: [缓解措施]

### 成功标准
- [ ] [标准1]
- [ ] [标准2]

## 重要提示
- 计划应该足够详细，让执行者可以直接开始工作
- 每个步骤应该是独立可验证的
- 优先考虑增量式实现，避免大规模重构
- 使用 TodoWrite 工具记录计划，便于后续追踪`,
		Tools:     []string{"Read", "Glob", "Grep", "WebSearch", "TodoWrite", "TodoRead"},
		Parallel:  false,
		MaxTokens: 100000,
		Timeout:   15 * time.Minute,
	})

	// General Purpose Agent - 通用任务
	m.RegisterSpec(&types.SubAgentSpec{
		Name:        "general-purpose",
		Description: "通用代理，可以执行复杂的多步骤任务，包括代码修改",
		Prompt: `你是一个通用 AI 助手。你可以执行各种任务，包括：
- 代码分析和修改
- 文件操作
- 信息搜索
- 问题解决

请根据任务要求选择合适的工具完成任务。`,
		Tools:     []string{"*"}, // 所有工具
		Parallel:  false,
		MaxTokens: 200000,
		Timeout:   30 * time.Minute,
	})
}

// RegisterSpec 注册子 Agent 规格
func (m *SubAgentManager) RegisterSpec(spec *types.SubAgentSpec) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.specs[spec.Name] = spec
}

// GetSpec 获取子 Agent 规格
func (m *SubAgentManager) GetSpec(name string) (*types.SubAgentSpec, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	spec, ok := m.specs[name]
	if !ok {
		return nil, fmt.Errorf("subagent spec not found: %s", name)
	}
	return spec, nil
}

// ListSpecs 列出所有子 Agent 规格
func (m *SubAgentManager) ListSpecs() []*types.SubAgentSpec {
	m.mu.RLock()
	defer m.mu.RUnlock()

	specs := make([]*types.SubAgentSpec, 0, len(m.specs))
	for _, spec := range m.specs {
		specs = append(specs, spec)
	}
	return specs
}

// Execute 执行子 Agent 任务
func (m *SubAgentManager) Execute(ctx context.Context, req *types.SubAgentRequest) (*types.SubAgentResult, error) {
	// 获取规格
	spec, err := m.GetSpec(req.AgentType)
	if err != nil {
		return nil, err
	}

	// 生成任务 ID
	taskID := uuid.New().String()

	// 设置超时
	timeout := spec.Timeout
	if req.Timeout > 0 {
		timeout = req.Timeout
	}
	if timeout == 0 {
		timeout = m.defaultTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	subagentLog.Info(ctx, "starting subagent", map[string]any{
		"task_id":    taskID,
		"agent_type": req.AgentType,
		"timeout":    timeout.String(),
	})

	startTime := time.Now()

	// 创建子 Agent 配置
	agentConfig := m.buildAgentConfig(spec, req, taskID)

	// 创建子 Agent
	agent, err := Create(ctx, agentConfig, m.deps)
	if err != nil {
		return &types.SubAgentResult{
			AgentType: req.AgentType,
			Success:   false,
			Error:     fmt.Sprintf("failed to create subagent: %v", err),
			Duration:  time.Since(startTime),
		}, nil
	}
	defer func() { _ = agent.Close() }() // Best effort cleanup

	// 注册运行中的子 Agent
	handle := &SubAgentHandle{
		ID:            taskID,
		AgentType:     req.AgentType,
		Agent:         agent,
		Request:       req,
		StartTime:     startTime,
		Status:        "running",
		CancelFunc:    cancel,
		ProgressChan:  make(chan *types.SubAgentProgressEvent, 100),
		ParentAgentID: req.ParentAgentID,
	}
	m.registerHandle(handle)
	defer m.unregisterHandle(taskID)

	// 订阅事件以跟踪进度
	eventCh := agent.Subscribe([]types.AgentChannel{types.ChannelProgress}, nil)
	defer agent.Unsubscribe(eventCh)

	// 启动进度监控
	go m.monitorProgress(ctx, handle, eventCh)

	// 构建任务消息
	taskMessage := m.buildTaskMessage(req, spec)

	// 执行对话
	result, err := agent.Chat(ctx, taskMessage)
	if err != nil {
		handle.Status = "failed"
		return &types.SubAgentResult{
			AgentType: req.AgentType,
			Success:   false,
			Error:     fmt.Sprintf("subagent execution failed: %v", err),
			Duration:  time.Since(startTime),
		}, nil
	}

	// 获取状态
	status := agent.Status()

	handle.Status = "completed"
	subagentResult := &types.SubAgentResult{
		AgentType:  req.AgentType,
		Success:    true,
		Output:     result.Text,
		Duration:   time.Since(startTime),
		StepCount:  status.StepCount,
		TokensUsed: 0, // TODO: 从 provider 获取
		Artifacts:  make(map[string]any),
	}

	handle.Result = subagentResult

	subagentLog.Info(ctx, "subagent completed", map[string]any{
		"task_id":    taskID,
		"agent_type": req.AgentType,
		"duration":   subagentResult.Duration.String(),
		"steps":      subagentResult.StepCount,
	})

	return subagentResult, nil
}

// ExecuteAsync 异步执行子 Agent 任务
func (m *SubAgentManager) ExecuteAsync(ctx context.Context, req *types.SubAgentRequest) (string, <-chan *types.SubAgentProgressEvent, error) {
	// 获取规格
	spec, err := m.GetSpec(req.AgentType)
	if err != nil {
		return "", nil, err
	}

	// 生成任务 ID
	taskID := uuid.New().String()

	// 设置超时
	timeout := spec.Timeout
	if req.Timeout > 0 {
		timeout = req.Timeout
	}
	if timeout == 0 {
		timeout = m.defaultTimeout
	}

	execCtx, cancel := context.WithTimeout(context.Background(), timeout)

	// 创建进度通道
	progressChan := make(chan *types.SubAgentProgressEvent, 100)

	// 创建句柄
	handle := &SubAgentHandle{
		ID:            taskID,
		AgentType:     req.AgentType,
		Request:       req,
		StartTime:     time.Now(),
		Status:        "starting",
		CancelFunc:    cancel,
		ProgressChan:  progressChan,
		ParentAgentID: req.ParentAgentID,
	}
	m.registerHandle(handle)

	// 异步执行
	go func() {
		defer close(progressChan)
		defer cancel()
		defer m.unregisterHandle(taskID)

		// 发送开始事件
		progressChan <- &types.SubAgentProgressEvent{
			AgentType: req.AgentType,
			TaskID:    taskID,
			Phase:     "started",
			Progress:  0,
			Message:   "Subagent starting...",
		}

		// 创建子 Agent
		agentConfig := m.buildAgentConfig(spec, req, taskID)
		agent, err := Create(execCtx, agentConfig, m.deps)
		if err != nil {
			handle.Status = "failed"
			handle.Result = &types.SubAgentResult{
				AgentType: req.AgentType,
				Success:   false,
				Error:     fmt.Sprintf("failed to create subagent: %v", err),
				Duration:  time.Since(handle.StartTime),
			}
			progressChan <- &types.SubAgentProgressEvent{
				AgentType: req.AgentType,
				TaskID:    taskID,
				Phase:     "failed",
				Progress:  100,
				Message:   err.Error(),
			}
			return
		}
		defer func() { _ = agent.Close() }() // Best effort cleanup

		handle.Agent = agent
		handle.Status = "running"

		// 订阅事件
		eventCh := agent.Subscribe([]types.AgentChannel{types.ChannelProgress}, nil)
		defer agent.Unsubscribe(eventCh)

		// 启动进度转发
		go m.forwardProgress(execCtx, handle, eventCh, progressChan)

		// 构建任务消息
		taskMessage := m.buildTaskMessage(req, spec)

		// 执行对话
		result, err := agent.Chat(execCtx, taskMessage)
		if err != nil {
			handle.Status = "failed"
			handle.Result = &types.SubAgentResult{
				AgentType: req.AgentType,
				Success:   false,
				Error:     err.Error(),
				Duration:  time.Since(handle.StartTime),
			}
			progressChan <- &types.SubAgentProgressEvent{
				AgentType: req.AgentType,
				TaskID:    taskID,
				Phase:     "failed",
				Progress:  100,
				Message:   err.Error(),
			}
			return
		}

		// 完成
		status := agent.Status()
		handle.Status = "completed"
		handle.Result = &types.SubAgentResult{
			AgentType:  req.AgentType,
			Success:    true,
			Output:     result.Text,
			Duration:   time.Since(handle.StartTime),
			StepCount:  status.StepCount,
			TokensUsed: 0,
			Artifacts:  make(map[string]any),
		}

		progressChan <- &types.SubAgentProgressEvent{
			AgentType: req.AgentType,
			TaskID:    taskID,
			Phase:     "completed",
			Progress:  100,
			Message:   "Subagent completed successfully",
		}
	}()

	return taskID, progressChan, nil
}

// Cancel 取消子 Agent 任务
func (m *SubAgentManager) Cancel(taskID string) error {
	m.mu.RLock()
	handle, ok := m.running[taskID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if handle.Status != "running" && handle.Status != "starting" {
		return fmt.Errorf("task is not running: %s", handle.Status)
	}

	handle.CancelFunc()
	handle.Status = "cancelled"

	return nil
}

// GetStatus 获取子 Agent 任务状态
func (m *SubAgentManager) GetStatus(taskID string) (*SubAgentHandle, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	handle, ok := m.running[taskID]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return handle, nil
}

// ListRunning 列出运行中的子 Agent
func (m *SubAgentManager) ListRunning() []*SubAgentHandle {
	m.mu.RLock()
	defer m.mu.RUnlock()

	handles := make([]*SubAgentHandle, 0, len(m.running))
	for _, h := range m.running {
		handles = append(handles, h)
	}
	return handles
}

// buildAgentConfig 构建子 Agent 配置
func (m *SubAgentManager) buildAgentConfig(spec *types.SubAgentSpec, req *types.SubAgentRequest, taskID string) *types.AgentConfig {
	// 创建子 Agent 专用模板 ID
	templateID := fmt.Sprintf("subagent-%s-%s", spec.Name, taskID[:8])

	// 注册临时模板
	template := &types.AgentTemplateDefinition{
		ID:           templateID,
		SystemPrompt: spec.Prompt,
		Model:        spec.Model,
		Tools:        spec.Tools,
		Runtime: &types.AgentTemplateRuntime{
			ExposeThinking: false,
			Todo: &types.TodoConfig{
				Enabled: false,
			},
		},
	}
	m.deps.TemplateRegistry.Register(template)

	// 构建元数据
	metadata := make(map[string]any)
	if req.Context != nil {
		for k, v := range req.Context {
			metadata[k] = v
		}
	}
	metadata["subagent_task_id"] = taskID
	metadata["subagent_type"] = spec.Name
	metadata["parent_agent_id"] = req.ParentAgentID

	config := &types.AgentConfig{
		AgentID:    fmt.Sprintf("subagent-%s", taskID[:8]),
		TemplateID: templateID,
		Metadata:   metadata,
	}

	// 如果指定了模型，设置 ModelConfig
	if spec.Model != "" {
		inferredProvider := inferProviderFromModel(spec.Model)
		config.ModelConfig = &types.ModelConfig{
			Provider: inferredProvider,
			Model:    spec.Model,
		}
	}

	return config
}

// buildTaskMessage 构建任务消息
func (m *SubAgentManager) buildTaskMessage(req *types.SubAgentRequest, spec *types.SubAgentSpec) string {
	var msg string

	// 添加任务描述
	msg = fmt.Sprintf("## Task\n\n%s\n", req.Task)

	// 添加上下文信息
	if len(req.Context) > 0 {
		msg += "\n## Context\n\n"
		for k, v := range req.Context {
			msg += fmt.Sprintf("- **%s**: %v\n", k, v)
		}
	}

	// 添加约束提醒
	msg += "\n## Constraints\n\n"
	msg += fmt.Sprintf("- Available tools: %v\n", spec.Tools)
	if spec.Timeout > 0 {
		msg += fmt.Sprintf("- Time limit: %s\n", spec.Timeout.String())
	}

	return msg
}

// registerHandle 注册子 Agent 句柄
func (m *SubAgentManager) registerHandle(handle *SubAgentHandle) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.running[handle.ID] = handle
}

// unregisterHandle 注销子 Agent 句柄
func (m *SubAgentManager) unregisterHandle(taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.running, taskID)
}

// monitorProgress 监控子 Agent 进度
func (m *SubAgentManager) monitorProgress(ctx context.Context, handle *SubAgentHandle, eventCh <-chan types.AgentEventEnvelope) {
	for {
		select {
		case <-ctx.Done():
			return
		case env, ok := <-eventCh:
			if !ok {
				return
			}
			// 转换事件为进度事件
			progressEvent := m.convertToProgressEvent(handle, env)
			if progressEvent != nil {
				select {
				case handle.ProgressChan <- progressEvent:
				default:
					// 通道满了，跳过
				}
			}
		}
	}
}

// forwardProgress 转发进度事件
func (m *SubAgentManager) forwardProgress(ctx context.Context, handle *SubAgentHandle, eventCh <-chan types.AgentEventEnvelope, progressChan chan<- *types.SubAgentProgressEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case env, ok := <-eventCh:
			if !ok {
				return
			}
			progressEvent := m.convertToProgressEvent(handle, env)
			if progressEvent != nil {
				select {
				case progressChan <- progressEvent:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

// convertToProgressEvent 转换事件为进度事件
func (m *SubAgentManager) convertToProgressEvent(handle *SubAgentHandle, env types.AgentEventEnvelope) *types.SubAgentProgressEvent {
	event := env.Event
	if event == nil {
		return nil
	}

	var phase string
	var message string
	var progress int

	switch e := event.(type) {
	case *types.ProgressTextChunkEvent:
		phase = "thinking"
		message = truncateString(e.Delta, 100)
		progress = 30
	case *types.ProgressToolStartEvent:
		phase = "tool_use"
		message = fmt.Sprintf("Using tool: %s", e.Call.Name)
		progress = 50
	case *types.ProgressToolEndEvent:
		phase = "tool_use"
		message = fmt.Sprintf("Tool completed: %s", e.Call.Name)
		progress = 70
	default:
		return nil
	}

	return &types.SubAgentProgressEvent{
		AgentType: handle.AgentType,
		TaskID:    handle.ID,
		Phase:     phase,
		Progress:  progress,
		Message:   message,
	}
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// SubAgentExecutorImpl 子 Agent 执行器实现
// 实现 types.SubAgentExecutor 接口
type SubAgentExecutorImpl struct {
	manager *SubAgentManager
	spec    *types.SubAgentSpec
}

// NewSubAgentExecutor 创建子 Agent 执行器
func NewSubAgentExecutor(manager *SubAgentManager, agentType string) (*SubAgentExecutorImpl, error) {
	spec, err := manager.GetSpec(agentType)
	if err != nil {
		return nil, err
	}

	return &SubAgentExecutorImpl{
		manager: manager,
		spec:    spec,
	}, nil
}

// GetSpec 获取子 Agent 规格
func (e *SubAgentExecutorImpl) GetSpec() *types.SubAgentSpec {
	return e.spec
}

// Execute 执行子 Agent 任务
func (e *SubAgentExecutorImpl) Execute(ctx context.Context, req *types.SubAgentRequest) (*types.SubAgentResult, error) {
	return e.manager.Execute(ctx, req)
}
