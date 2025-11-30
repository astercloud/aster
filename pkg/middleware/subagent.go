package middleware

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/builtin"
	"github.com/astercloud/aster/pkg/types"
)

// SubAgentSpec 子代理规格
type SubAgentSpec struct {
	Name                string         // 子代理名称
	Description         string         // 子代理描述
	Prompt              string         // 子代理专用提示词
	Tools               []string       // 工具名称列表(可选,默认继承父代理)
	Config              map[string]any // 自定义配置
	InheritMiddlewares  bool           // 是否继承父代理的中间件栈(默认 false)
	MiddlewareOverrides []Middleware   // 子代理专用中间件(覆盖或追加)
}

// SubAgentFactory 子代理工厂函数
// 用于创建子代理实例
type SubAgentFactory func(ctx context.Context, spec SubAgentSpec) (SubAgent, error)

// SubAgent 子代理接口
type SubAgent interface {
	// Name 返回子代理名称
	Name() string

	// Execute 执行任务
	// description: 任务描述
	// context: 父代理上下文(可选)
	Execute(ctx context.Context, description string, parentContext map[string]any) (string, error)

	// Close 关闭子代理
	Close() error
}

// SubAgentMiddlewareConfig 子代理中间件配置
type SubAgentMiddlewareConfig struct {
	Specs                  []SubAgentSpec          // 子代理规格列表
	Factory                SubAgentFactory         // 子代理工厂
	Manager                builtin.SubagentManager // 子代理管理器（可选，默认使用 FileSubagentManager）
	EnableParallel         bool                    // 是否支持并行执行
	EnableGeneralPurpose   bool                    // 是否启用通用子代理(默认 true)
	EnableAsync            bool                    // 是否启用异步执行（默认 false）
	EnableProcessIsolation bool                    // 是否启用进程级隔离（默认 false）
	DefaultTimeout         time.Duration           // 默认超时时间（默认 1 小时）
	ParentMiddlewareGetter func() []Middleware
}

// SubAgentMiddleware 子代理中间件
// 功能:
// 1. 管理多个子代理实例
// 2. 提供 task 工具启动子代理
// 3. 支持任务上下文隔离
// 4. 支持并发执行
// 5. 支持异步执行和状态查询
// 6. 支持 Resume 机制
// 7. 支持资源监控和限制
type SubAgentMiddleware struct {
	*BaseMiddleware
	agents                 map[string]SubAgent
	factory                SubAgentFactory
	manager                builtin.SubagentManager
	enableParallel         bool
	enableAsync            bool
	enableProcessIsolation bool
	defaultTimeout         time.Duration
	mu                     sync.RWMutex
}

// NewSubAgentMiddleware 创建子代理中间件
func NewSubAgentMiddleware(config *SubAgentMiddlewareConfig) (*SubAgentMiddleware, error) {
	// 设置默认超时
	defaultTimeout := config.DefaultTimeout
	if defaultTimeout == 0 {
		defaultTimeout = 1 * time.Hour
	}

	m := &SubAgentMiddleware{
		BaseMiddleware:         NewBaseMiddleware("subagent", 200),
		agents:                 make(map[string]SubAgent),
		factory:                config.Factory,
		enableParallel:         config.EnableParallel,
		enableAsync:            config.EnableAsync,
		enableProcessIsolation: config.EnableProcessIsolation,
		defaultTimeout:         defaultTimeout,
	}

	// 创建或使用提供的管理器
	switch {
	case config.Manager != nil:
		m.manager = config.Manager
	case config.EnableProcessIsolation:
		m.manager = builtin.NewFileSubagentManager()
		log.Printf("[SubAgentMiddleware] Using FileSubagentManager for process isolation/async execution")
	case config.EnableAsync:
		m.manager = newGoroutineSubagentManager(m)
		log.Printf("[SubAgentMiddleware] Using in-memory async manager")
	}

	// 默认启用 general-purpose 子代理
	specs := config.Specs
	if config.EnableGeneralPurpose || (len(specs) == 0 && !config.EnableParallel) {
		// 添加通用子代理规格
		generalPurposeSpec := SubAgentSpec{
			Name:        "general-purpose",
			Description: "通用子代理,用于执行复杂、多步骤的隔离任务",
			Prompt: `你是一个通用的 AI 助手,专注于执行复杂的、多步骤的任务。
你有完整的工具集,可以独立完成被委托的任务。
请仔细分析任务需求,制定计划并逐步执行。`,
			InheritMiddlewares: true, // 继承父代理的中间件
		}
		specs = append([]SubAgentSpec{generalPurposeSpec}, specs...)
	}

	// 初始化子代理
	if config.Factory != nil {
		for _, spec := range specs {
			agent, err := config.Factory(context.Background(), spec)
			if err != nil {
				log.Printf("[SubAgentMiddleware] Failed to create subagent %s: %v", spec.Name, err)
				continue
			}
			m.agents[spec.Name] = agent
			log.Printf("[SubAgentMiddleware] Created subagent: %s", spec.Name)
		}
	}

	log.Printf("[SubAgentMiddleware] Initialized with %d subagents", len(m.agents))
	return m, nil
}

// Tools 返回 task 工具和管理工具
func (m *SubAgentMiddleware) Tools() []tools.Tool {
	baseTools := []tools.Tool{
		&TaskTool{
			middleware: m,
		},
	}

	// 如果启用了管理器，添加管理工具
	if m.manager != nil {
		baseTools = append(baseTools,
			&QuerySubagentTool{middleware: m},
			&StopSubagentTool{middleware: m},
			&ResumeSubagentTool{middleware: m},
			&ListSubagentsTool{middleware: m},
		)
	}

	return baseTools
}

// OnAgentStop 清理子代理
func (m *SubAgentMiddleware) OnAgentStop(ctx context.Context, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, agent := range m.agents {
		if err := agent.Close(); err != nil {
			log.Printf("[SubAgentMiddleware] Failed to close subagent %s: %v", name, err)
		}
	}

	return nil
}

// GetSubAgent 获取子代理
func (m *SubAgentMiddleware) GetSubAgent(name string) (SubAgent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agent, exists := m.agents[name]
	if !exists {
		return nil, fmt.Errorf("subagent not found: %s", name)
	}

	return agent, nil
}

// ListSubAgents 列出所有子代理
func (m *SubAgentMiddleware) ListSubAgents() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.agents))
	for name := range m.agents {
		names = append(names, name)
	}
	return names
}

// TaskTool task 工具实现
type TaskTool struct {
	middleware *SubAgentMiddleware
}

func (t *TaskTool) Name() string {
	return "task"
}

func (t *TaskTool) Description() string {
	return "Delegate a task to a specialized sub-agent for isolated, focused execution"
}

func (t *TaskTool) InputSchema() map[string]any {
	// 构建子代理类型枚举
	subagentTypes := t.middleware.ListSubAgents()

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"description": map[string]any{
				"type":        "string",
				"description": "Clear, detailed description of the task to delegate",
			},
			"subagent_type": map[string]any{
				"type":        "string",
				"description": fmt.Sprintf("Type of sub-agent to use. Available: %v", subagentTypes),
				"enum":        subagentTypes,
			},
			"context": map[string]any{
				"type":        "object",
				"description": "Optional context to pass to the sub-agent",
			},
		},
		"required": []string{"description", "subagent_type"},
	}

	// 如果启用异步执行，添加 async 参数
	if t.middleware.enableAsync {
		schema["properties"].(map[string]any)["async"] = map[string]any{
			"type":        "boolean",
			"description": "Run asynchronously in background (default: false). If true, returns task_id immediately.",
			"default":     false,
		}
		schema["properties"].(map[string]any)["timeout"] = map[string]any{
			"type":        "number",
			"description": "Timeout in seconds (default: 3600)",
			"default":     3600,
		}
	}

	return schema
}

func (t *TaskTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	description, ok := input["description"].(string)
	if !ok {
		return nil, fmt.Errorf("description must be a string")
	}

	subagentType, ok := input["subagent_type"].(string)
	if !ok {
		return nil, fmt.Errorf("subagent_type must be a string")
	}

	// 获取上下文(可选)
	parentContext := make(map[string]any)
	if contextData, ok := input["context"].(map[string]any); ok {
		parentContext = contextData
	}

	// 检查是否异步执行
	async := false
	if asyncVal, ok := input["async"].(bool); ok {
		async = asyncVal
	}

	// 获取超时设置
	timeout := t.middleware.defaultTimeout
	if timeoutVal, ok := input["timeout"].(float64); ok {
		timeout = time.Duration(timeoutVal) * time.Second
	}

	// 如果启用了管理器且请求异步执行
	if t.middleware.manager != nil && async {
		return t.executeAsync(ctx, subagentType, description, parentContext, timeout)
	}

	// 同步执行（原有逻辑）
	return t.executeSync(ctx, subagentType, description, parentContext)
}

// executeSync 同步执行子代理
func (t *TaskTool) executeSync(ctx context.Context, subagentType, description string, parentContext map[string]any) (any, error) {
	// 获取子代理
	subagent, err := t.middleware.GetSubAgent(subagentType)
	if err != nil {
		return map[string]any{
			"ok":    false,
			"error": fmt.Sprintf("failed to get subagent: %v", err),
		}, nil
	}

	log.Printf("[TaskTool] Delegating task to subagent '%s': %s", subagentType, description)

	// 执行任务
	result, err := subagent.Execute(ctx, description, parentContext)
	if err != nil {
		return map[string]any{
			"ok":            false,
			"error":         fmt.Sprintf("subagent execution failed: %v", err),
			"subagent_type": subagentType,
		}, nil
	}

	log.Printf("[TaskTool] Subagent '%s' completed task successfully", subagentType)

	return map[string]any{
		"ok":            true,
		"subagent_type": subagentType,
		"result":        result,
	}, nil
}

// executeAsync 异步执行子代理
func (t *TaskTool) executeAsync(ctx context.Context, subagentType, description string, parentContext map[string]any, timeout time.Duration) (any, error) {
	// 构建子代理配置
	config := &builtin.SubagentConfig{
		Type:          subagentType,
		Prompt:        description,
		Timeout:       timeout,
		ParentContext: parentContext,
		Metadata: map[string]string{
			"subagent_type": subagentType,
			"description":   description,
		},
	}

	// 启动子代理
	instance, err := t.middleware.manager.StartSubagent(ctx, config)
	if err != nil {
		return map[string]any{
			"ok":    false,
			"error": fmt.Sprintf("failed to start subagent: %v", err),
		}, nil
	}

	log.Printf("[TaskTool] Started async subagent '%s' with task_id: %s", subagentType, instance.ID)

	return map[string]any{
		"ok":            true,
		"task_id":       instance.ID,
		"subagent_type": subagentType,
		"status":        instance.Status,
		"message":       "SubAgent started in background. Use query_subagent to check status and get results.",
	}, nil
}

func (t *TaskTool) Prompt() string {
	// 获取可用的子代理列表
	subagentTypes := t.middleware.ListSubAgents()
	agentList := "可用的子代理类型:\n"
	for _, name := range subagentTypes {
		agentList += fmt.Sprintf("  - %s\n", name)
	}

	return fmt.Sprintf(`启动短生命周期的子代理来处理复杂的、多步骤的独立任务,实现上下文隔离。

%s

## 核心优势

1. **上下文隔离**: 每个子代理有独立的上下文窗口,不会污染主线程
2. **并行执行**: 可以同时启动多个子代理,极大提升效率
3. **token优化**: 子代理处理完任务后只返回摘要结果,节省主线程的 token 消耗
4. **专注执行**: 每个子代理只需要关注一个独立任务,不受其他任务干扰

## 何时使用 task 工具

✅ **应该使用的情况**:
- 任务复杂且需要多个步骤,可以完整地独立委派
- 任务之间相互独立,可以并行执行
- 任务需要大量的推理或会消耗大量 token/context,会使主线程膨胀
- 沙箱隔离能提高可靠性(如代码执行、结构化搜索、数据格式化)
- 只关心子代理的最终输出,不关心中间步骤(如:进行大量研究后返回摘要报告、执行一系列计算后返回简洁答案)

❌ **不应该使用的情况**:
- 如果需要查看子代理完成后的中间推理或步骤(task工具会隐藏它们)
- 如果任务很简单(只需要几个工具调用或简单查询)
- 如果委派不能减少 token 使用、复杂度或上下文切换
- 如果拆分会增加延迟而没有好处

## 子代理生命周期

1. **启动** → 提供清晰的角色、指令和预期输出格式
2. **运行** → 子代理自主完成任务
3. **返回** → 子代理提供单个结构化结果
4. **整合** → 将结果合并或综合到主线程中

## 最佳实践

### 1. 并行化优先 ⚡
尽可能并行化工作。这对工具调用和任务都适用。当有独立的步骤要完成时,**在单个消息中并行调用多个 task 工具**,这能为用户节省大量时间。

示例:
- ❌ 顺序研究: 先研究A,再研究B,最后研究C (慢)
- ✅ 并行研究: 在一条消息中同时启动3个子代理研究A、B、C (快3倍!)

### 2. 提供详细的任务描述 📝
子代理是无状态的,启动后无法与你通信。因此:
- 在 description 中提供**高度详细**的任务描述
- 明确说明你期望子代理返回什么信息
- 告诉子代理是创建内容、执行分析,还是只做研究
- 如果有特定的输出格式要求,务必说明

### 3. 选择合适的子代理类型 🎯
- **general-purpose**: 通用子代理,适用于大多数任务,拥有所有工具

### 4. 信任子代理的输出 ✅
子代理的输出通常应该被信任,它们是高效且有能力的。

## 使用示例

### 示例 1: 并行研究任务

用户请求: "我想研究詹姆斯、乔丹和科比的成就,然后比较他们。"

正确做法:
- 在**一条消息中**并行启动3个 task,分别研究3位球员
- 每个子代理只关注一个球员,可以深入研究而不影响其他
- 收到3个摘要结果后,在主线程中进行比较

为什么好:
- 研究每个球员是复杂的多步骤任务
- 三个研究任务相互独立,可以并行
- 每个子代理专注于一个球员,上下文干净
- 返回的是摘要信息,而不是完整的研究过程,节省了主线程的 token

### 示例 2: 单个大型任务的上下文隔离

用户请求: "分析一个大型代码仓库的安全漏洞并生成报告。"

正确做法:
- 启动一个 task 子代理进行仓库分析
- 即使只有一个任务,也使用子代理来隔离大量的上下文

为什么好:
- 防止主线程被分析细节淹没
- 如果用户后续提问,可以引用简洁的报告而不是整个分析历史
- 节省时间和成本

### 示例 3: 多个独立的准备任务

用户请求: "为我安排两个会议并为每个会议准备议程。"

正确做法:
- 并行启动2个 task 子代理,每个准备一个会议的议程
- 返回最终的日程和议程

为什么好:
- 每个任务本身很简单,但子代理帮助隔离议程准备
- 每个子代理只需要关心一个会议的议程

### 示例 4: 何时不使用 task 工具

用户请求: "我想从达美乐订一个披萨,从麦当劳订一个汉堡,从赛百味订一个沙拉。"

正确做法:
- **直接**并行调用3个订购工具,**不使用** task 工具

为什么:
- 目标非常简单明确,只需要几个简单的工具调用
- 使用 task 工具反而增加不必要的开销
- 直接完成任务更快更好

## 重要提醒

1. **并行化是关键**: 尽可能使用并行执行来节省用户时间
2. **详细的指令**: 子代理无法回头问你问题,所以一次性给清楚
3. **上下文隔离**: 利用子代理来隔离复杂任务,保持主线程简洁
4. **信任结果**: 子代理是可靠的,信任它们的输出
5. **判断何时使用**: 简单任务直接完成,复杂独立任务才委派

记住:使用 task 工具来**隔离复杂任务**、**并行独立任务**、**优化 token 使用**!`, agentList)
}

// SimpleSubAgent 简单子代理实现
// 用于测试和演示
type SimpleSubAgent struct {
	name   string
	prompt string
	execFn func(ctx context.Context, description string, parentContext map[string]any) (string, error)
}

func NewSimpleSubAgent(name, prompt string, execFn func(context.Context, string, map[string]any) (string, error)) *SimpleSubAgent {
	return &SimpleSubAgent{
		name:   name,
		prompt: prompt,
		execFn: execFn,
	}
}

func (a *SimpleSubAgent) Name() string {
	return a.name
}

func (a *SimpleSubAgent) Execute(ctx context.Context, description string, parentContext map[string]any) (string, error) {
	if a.execFn != nil {
		return a.execFn(ctx, description, parentContext)
	}
	return fmt.Sprintf("[%s] Executed: %s", a.name, description), nil
}

func (a *SimpleSubAgent) Close() error {
	return nil
}

// SubAgentResult 子代理执行结果
type SubAgentResult struct {
	Success      bool           `json:"success"`
	SubAgentType string         `json:"subagent_type"`
	Result       string         `json:"result"`
	Error        string         `json:"error,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// ExtractMessages 从父代理状态提取消息
// 用于子代理继承部分上下文
func ExtractMessages(messages []types.Message, limit int) []types.Message {
	if limit <= 0 || len(messages) <= limit {
		return messages
	}
	return messages[len(messages)-limit:]
}

// IsolateContext 隔离上下文
// 从父代理状态中提取必要信息,创建干净的子代理上下文
func IsolateContext(parentState map[string]any, includeKeys []string) map[string]any {
	isolated := make(map[string]any)

	for _, key := range includeKeys {
		if val, exists := parentState[key]; exists {
			isolated[key] = val
		}
	}

	return isolated
}

// BuildSubAgentMiddlewareStack 构建子代理的中间件栈
// 根据子代理规格决定是否继承父代理中间件
func BuildSubAgentMiddlewareStack(spec SubAgentSpec, parentMiddlewares []Middleware) []Middleware {
	if !spec.InheritMiddlewares {
		// 不继承,只使用子代理专用中间件
		return spec.MiddlewareOverrides
	}

	// 继承父代理中间件
	stack := make([]Middleware, 0, len(parentMiddlewares)+len(spec.MiddlewareOverrides))

	// 1. 首先添加父代理中间件
	stack = append(stack, parentMiddlewares...)

	// 2. 然后添加子代理专用中间件(覆盖或追加)
	if len(spec.MiddlewareOverrides) > 0 {
		// 创建名称映射用于覆盖检测
		nameMap := make(map[string]int)
		for i, m := range stack {
			nameMap[m.Name()] = i
		}

		// 处理覆盖和追加
		for _, override := range spec.MiddlewareOverrides {
			if idx, exists := nameMap[override.Name()]; exists {
				// 覆盖同名中间件
				stack[idx] = override
				log.Printf("[BuildSubAgentMiddlewareStack] Override middleware: %s", override.Name())
			} else {
				// 追加新中间件
				stack = append(stack, override)
				log.Printf("[BuildSubAgentMiddlewareStack] Append middleware: %s", override.Name())
			}
		}
	}

	return stack
}

// GetMiddlewareForSubAgent 获取子代理应该使用的中间件栈
// 这是一个辅助函数,供 SubAgentFactory 实现使用
func (m *SubAgentMiddleware) GetMiddlewareForSubAgent(spec SubAgentSpec, parentMiddlewares []Middleware) []Middleware {
	return BuildSubAgentMiddlewareStack(spec, parentMiddlewares)
}

// QuerySubagentTool 查询子代理状态工具
type QuerySubagentTool struct {
	middleware *SubAgentMiddleware
}

func (t *QuerySubagentTool) Name() string {
	return "query_subagent"
}

func (t *QuerySubagentTool) Description() string {
	return "Query the status and output of a running or completed sub-agent task"
}

func (t *QuerySubagentTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task_id": map[string]any{
				"type":        "string",
				"description": "The task ID returned by async task execution",
			},
		},
		"required": []string{"task_id"},
	}
}

func (t *QuerySubagentTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	taskID, ok := input["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("task_id must be a string")
	}

	// 获取子代理实例
	instance, err := t.middleware.manager.GetSubagent(taskID)
	if err != nil {
		return map[string]any{
			"ok":    false,
			"error": fmt.Sprintf("failed to get subagent: %v", err),
		}, nil
	}

	// 构建响应
	response := map[string]any{
		"ok":            true,
		"task_id":       instance.ID,
		"subagent_type": instance.Type,
		"status":        instance.Status,
		"start_time":    instance.StartTime.Format(time.RFC3339),
		"duration":      instance.Duration.Seconds(),
	}

	// 添加结束时间（如果已完成）
	if instance.EndTime != nil {
		response["end_time"] = instance.EndTime.Format(time.RFC3339)
	}

	// 添加输出（如果有）
	if instance.Output != "" {
		response["output"] = instance.Output
	}

	// 添加错误信息（如果有）
	if instance.Error != "" {
		response["error"] = instance.Error
		response["exit_code"] = instance.ExitCode
	}

	// 添加资源使用情况（如果有）
	if instance.ResourceUsage != nil {
		response["resource_usage"] = map[string]any{
			"memory_mb":   instance.ResourceUsage.MemoryMB,
			"cpu_percent": instance.ResourceUsage.CPUPercent,
		}
	}

	return response, nil
}

func (t *QuerySubagentTool) Prompt() string {
	return `查询异步子代理的执行状态和结果。

使用场景：
- 检查后台运行的子代理是否完成
- 获取子代理的输出结果
- 监控子代理的资源使用情况

状态说明：
- "starting": 正在启动
- "running": 正在运行
- "completed": 已完成（成功）
- "failed": 执行失败
- "stopped": 已停止
- "timeout": 超时

示例：
query_subagent(task_id="subagent_1234567890")
`
}

// StopSubagentTool 停止子代理工具
type StopSubagentTool struct {
	middleware *SubAgentMiddleware
}

func (t *StopSubagentTool) Name() string {
	return "stop_subagent"
}

func (t *StopSubagentTool) Description() string {
	return "Stop a running sub-agent task"
}

func (t *StopSubagentTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task_id": map[string]any{
				"type":        "string",
				"description": "The task ID of the sub-agent to stop",
			},
		},
		"required": []string{"task_id"},
	}
}

func (t *StopSubagentTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	taskID, ok := input["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("task_id must be a string")
	}

	// 停止子代理
	err := t.middleware.manager.StopSubagent(taskID)
	if err != nil {
		return map[string]any{
			"ok":    false,
			"error": fmt.Sprintf("failed to stop subagent: %v", err),
		}, nil
	}

	log.Printf("[StopSubagentTool] Stopped subagent: %s", taskID)

	return map[string]any{
		"ok":      true,
		"task_id": taskID,
		"message": "SubAgent stopped successfully",
	}, nil
}

func (t *StopSubagentTool) Prompt() string {
	return `停止正在运行的子代理任务。

使用场景：
- 子代理运行时间过长，需要手动停止
- 发现子代理执行方向错误，需要中止
- 资源紧张，需要释放资源

注意：
- 只能停止状态为 "running" 的子代理
- 停止后可以使用 resume_subagent 恢复执行

示例：
stop_subagent(task_id="subagent_1234567890")
`
}

// ResumeSubagentTool 恢复子代理工具
type ResumeSubagentTool struct {
	middleware *SubAgentMiddleware
}

func (t *ResumeSubagentTool) Name() string {
	return "resume_subagent"
}

func (t *ResumeSubagentTool) Description() string {
	return "Resume a stopped or failed sub-agent task"
}

func (t *ResumeSubagentTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task_id": map[string]any{
				"type":        "string",
				"description": "The task ID of the sub-agent to resume",
			},
		},
		"required": []string{"task_id"},
	}
}

func (t *ResumeSubagentTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	taskID, ok := input["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("task_id must be a string")
	}

	// 恢复子代理
	instance, err := t.middleware.manager.ResumeSubagent(taskID)
	if err != nil {
		return map[string]any{
			"ok":    false,
			"error": fmt.Sprintf("failed to resume subagent: %v", err),
		}, nil
	}

	log.Printf("[ResumeSubagentTool] Resumed subagent: %s (new task_id: %s)", taskID, instance.ID)

	return map[string]any{
		"ok":            true,
		"old_task_id":   taskID,
		"new_task_id":   instance.ID,
		"subagent_type": instance.Type,
		"status":        instance.Status,
		"message":       "SubAgent resumed successfully with new task_id",
	}, nil
}

func (t *ResumeSubagentTool) Prompt() string {
	return `恢复已停止或失败的子代理任务。

使用场景：
- 子代理因超时或错误而停止，需要重新执行
- 手动停止的子代理，需要继续执行
- 系统重启后，恢复未完成的任务

注意：
- 只能恢复状态为 "stopped"、"failed" 或 "completed" 的子代理
- 恢复后会生成新的 task_id
- 原有的元数据会被保留

示例：
resume_subagent(task_id="subagent_1234567890")
`
}

// ListSubagentsTool 列出所有子代理工具
type ListSubagentsTool struct {
	middleware *SubAgentMiddleware
}

func (t *ListSubagentsTool) Name() string {
	return "list_subagents"
}

func (t *ListSubagentsTool) Description() string {
	return "List all sub-agent tasks (running, completed, or failed)"
}

func (t *ListSubagentsTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"status_filter": map[string]any{
				"type":        "string",
				"description": "Filter by status (optional): running, completed, failed, stopped",
			},
		},
	}
}

func (t *ListSubagentsTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	// 获取状态过滤器（可选）
	statusFilter := ""
	if filter, ok := input["status_filter"].(string); ok {
		statusFilter = filter
	}

	// 列出所有子代理
	instances, err := t.middleware.manager.ListSubagents()
	if err != nil {
		return map[string]any{
			"ok":    false,
			"error": fmt.Sprintf("failed to list subagents: %v", err),
		}, nil
	}

	// 构建响应
	subagents := make([]map[string]any, 0)
	for _, instance := range instances {
		// 应用状态过滤
		if statusFilter != "" && instance.Status != statusFilter {
			continue
		}

		item := map[string]any{
			"task_id":       instance.ID,
			"subagent_type": instance.Type,
			"status":        instance.Status,
			"start_time":    instance.StartTime.Format(time.RFC3339),
			"duration":      instance.Duration.Seconds(),
		}

		if instance.EndTime != nil {
			item["end_time"] = instance.EndTime.Format(time.RFC3339)
		}

		if instance.Error != "" {
			item["error"] = instance.Error
		}

		subagents = append(subagents, item)
	}

	return map[string]any{
		"ok":        true,
		"count":     len(subagents),
		"subagents": subagents,
	}, nil
}

func (t *ListSubagentsTool) Prompt() string {
	return `列出所有子代理任务。

使用场景：
- 查看当前有哪些子代理正在运行
- 检查历史任务的执行情况
- 清理已完成的任务

参数：
- status_filter: 可选，按状态过滤（running, completed, failed, stopped）

示例：
list_subagents()  # 列出所有
list_subagents(status_filter="running")  # 只列出正在运行的
`
}
