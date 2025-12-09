package middleware

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/memory/logic"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/types"
)

var lmLog = logging.ForComponent("LogicMemoryMiddleware")

// LogicMemoryMiddleware Logic Memory 中间件
// 功能：
// 1. 自动捕获用户交互事件，识别并记录用户偏好和行为模式
// 2. 将相关 Memory 注入到 system prompt，实现个性化响应
// 3. 提供 Logic Memory 管理工具供 Agent 主动查询和更新
type LogicMemoryMiddleware struct {
	*BaseMiddleware
	manager            *logic.Manager
	config             *LogicMemoryMiddlewareConfig
	logicMemoryTools   []tools.Tool
	eventBuffer        chan *logic.Event
	stopCh             chan struct{}
	wg                 sync.WaitGroup
	namespaceExtractor NamespaceExtractor
}

// NamespaceExtractor 从请求中提取 namespace 的函数
// 应用层可以自定义此函数来实现不同的租户隔离策略
type NamespaceExtractor func(req *ModelRequest) string

// LogicMemoryMiddlewareConfig Logic Memory 中间件配置
type LogicMemoryMiddlewareConfig struct {
	// Manager Logic Memory 管理器（必需）
	Manager *logic.Manager

	// NamespaceExtractor 从请求中提取 namespace 的函数
	// 如果为空，使用默认提取器（从 metadata 中获取 user_id 或 namespace）
	NamespaceExtractor NamespaceExtractor

	// EnableCapture 是否启用自动捕获（默认 true）
	EnableCapture bool

	// CaptureChannels 要捕获的事件通道（默认 Control + Monitor）
	CaptureChannels []types.AgentChannel

	// EnableInjection 是否启用 Memory 注入到 Prompt（默认 true）
	EnableInjection bool

	// InjectionPoint 注入位置："system_prompt_start", "system_prompt_end", "both"
	InjectionPoint string

	// MaxMemories 注入的最大 Memory 数量（TopK，默认 5）
	MaxMemories int

	// MinConfidence 最低置信度阈值（默认 0.6）
	MinConfidence float64

	// AsyncCapture 是否异步捕获事件（默认 true）
	// 异步模式不阻塞主流程，但可能丢失部分事件
	AsyncCapture bool

	// EventBufferSize 事件缓冲区大小（默认 100）
	EventBufferSize int

	// CacheTTL Memory 缓存时间（默认 5 分钟）
	CacheTTL time.Duration

	// SystemPromptTemplate Memory 注入的模板
	// 占位符：%s = Memory 列表的 Markdown 格式
	SystemPromptTemplate string

	// Priority 中间件优先级（默认 7，在 working_memory 之后）
	Priority int
}

// NewLogicMemoryMiddleware 创建 Logic Memory 中间件
func NewLogicMemoryMiddleware(config *LogicMemoryMiddlewareConfig) (*LogicMemoryMiddleware, error) {
	if config == nil {
		return nil, fmt.Errorf("logic memory config is required")
	}

	if config.Manager == nil {
		return nil, fmt.Errorf("logic memory manager is required")
	}

	// 设置默认值
	if config.MaxMemories <= 0 {
		config.MaxMemories = 5
	}
	if config.MinConfidence <= 0 {
		config.MinConfidence = 0.6
	}
	if config.EventBufferSize <= 0 {
		config.EventBufferSize = 100
	}
	if config.CacheTTL <= 0 {
		config.CacheTTL = 5 * time.Minute
	}
	if config.Priority <= 0 {
		config.Priority = 7
	}
	if config.InjectionPoint == "" {
		config.InjectionPoint = "system_prompt_end"
	}
	if config.SystemPromptTemplate == "" {
		config.SystemPromptTemplate = defaultLogicMemoryPromptTemplate
	}
	if len(config.CaptureChannels) == 0 {
		config.CaptureChannels = []types.AgentChannel{
			types.ChannelControl,
			types.ChannelMonitor,
		}
	}

	// 默认的 namespace 提取器
	namespaceExtractor := config.NamespaceExtractor
	if namespaceExtractor == nil {
		namespaceExtractor = defaultNamespaceExtractor
	}

	m := &LogicMemoryMiddleware{
		BaseMiddleware:     NewBaseMiddleware("logic_memory", config.Priority),
		manager:            config.Manager,
		config:             config,
		namespaceExtractor: namespaceExtractor,
		stopCh:             make(chan struct{}),
	}

	// 创建事件缓冲区（如果启用异步捕获）
	if config.AsyncCapture && config.EnableCapture {
		m.eventBuffer = make(chan *logic.Event, config.EventBufferSize)
	}

	// 创建 Logic Memory 工具
	m.logicMemoryTools = m.createLogicMemoryTools()

	lmLog.Info(context.Background(), "initialized", map[string]any{"capture": config.EnableCapture, "injection": config.EnableInjection, "max_memories": config.MaxMemories, "min_confidence": config.MinConfidence})

	return m, nil
}

// Tools 返回 Logic Memory 相关工具
func (m *LogicMemoryMiddleware) Tools() []tools.Tool {
	return m.logicMemoryTools
}

// OnAgentStart Agent 启动时启动事件处理器
func (m *LogicMemoryMiddleware) OnAgentStart(ctx context.Context, agentID string) error {
	// 启动异步事件处理 goroutine
	if m.config.AsyncCapture && m.config.EnableCapture && m.eventBuffer != nil {
		m.wg.Add(1)
		go m.processEventsAsync()
		lmLog.Info(context.Background(), "started async event processor", map[string]any{"agent_id": agentID})
	}
	return nil
}

// OnAgentStop Agent 停止时停止事件处理器
func (m *LogicMemoryMiddleware) OnAgentStop(ctx context.Context, agentID string) error {
	// 停止异步事件处理
	if m.config.AsyncCapture && m.config.EnableCapture {
		close(m.stopCh)
		m.wg.Wait()
		lmLog.Info(context.Background(), "stopped async event processor", map[string]any{"agent_id": agentID})
	}
	return nil
}

// WrapModelCall 包装模型调用，注入 Logic Memory 到 system prompt
func (m *LogicMemoryMiddleware) WrapModelCall(ctx context.Context, req *ModelRequest, handler ModelCallHandler) (*ModelResponse, error) {
	// 如果不启用注入，直接调用下一层
	if !m.config.EnableInjection {
		return handler(ctx, req)
	}

	// 获取 namespace
	namespace := m.namespaceExtractor(req)
	if namespace == "" {
		// 没有 namespace，跳过注入
		return handler(ctx, req)
	}

	// 检索相关 Memory
	memories, err := m.manager.RetrieveMemories(ctx, namespace,
		logic.WithTopK(m.config.MaxMemories),
		logic.WithMinConfidence(m.config.MinConfidence),
		logic.WithOrderBy(logic.OrderByConfidence),
	)
	if err != nil {
		lmLog.Error(ctx, "failed to retrieve memories", map[string]any{"error": err.Error()})
		// 继续执行，不因为 Memory 检索失败而中断
		return handler(ctx, req)
	}

	// 如果没有相关 Memory，直接调用
	if len(memories) == 0 {
		return handler(ctx, req)
	}

	// 保存原始 system prompt
	originalSystemPrompt := req.SystemPrompt

	// 构建 Memory 注入文本
	memorySection := m.buildMemorySection(memories)

	// 根据配置的注入点注入
	switch m.config.InjectionPoint {
	case "system_prompt_start":
		if originalSystemPrompt != "" {
			req.SystemPrompt = memorySection + "\n\n" + originalSystemPrompt
		} else {
			req.SystemPrompt = memorySection
		}
	case "system_prompt_end":
		if originalSystemPrompt != "" {
			req.SystemPrompt = originalSystemPrompt + "\n\n" + memorySection
		} else {
			req.SystemPrompt = memorySection
		}
	case "both":
		if originalSystemPrompt != "" {
			req.SystemPrompt = memorySection + "\n\n" + originalSystemPrompt + "\n\n" + memorySection
		} else {
			req.SystemPrompt = memorySection
		}
	default:
		// 默认追加到末尾
		if originalSystemPrompt != "" {
			req.SystemPrompt = originalSystemPrompt + "\n\n" + memorySection
		} else {
			req.SystemPrompt = memorySection
		}
	}

	lmLog.Debug(ctx, "injected memories", map[string]any{"count": len(memories), "namespace": namespace, "chars": len(req.SystemPrompt)})

	// 调用处理器
	resp, err := handler(ctx, req)

	// 恢复原始 system prompt
	req.SystemPrompt = originalSystemPrompt

	return resp, err
}

// WrapToolCall 包装工具调用，捕获工具执行事件
func (m *LogicMemoryMiddleware) WrapToolCall(ctx context.Context, req *ToolCallRequest, handler ToolCallHandler) (*ToolCallResponse, error) {
	// 执行工具调用
	resp, err := handler(ctx, req)

	// 如果启用捕获，记录工具调用事件
	if m.config.EnableCapture {
		event := &logic.Event{
			Type:   "tool_result",
			Source: m.extractSourceFromToolContext(req),
			Data: map[string]any{
				"tool_name":    req.ToolName,
				"tool_call_id": req.ToolCallID,
				"input":        req.ToolInput,
				"success":      err == nil,
			},
			Timestamp: time.Now(),
		}

		if resp != nil {
			event.Data["result"] = resp.Result
		}
		if err != nil {
			event.Data["error"] = err.Error()
		}

		m.captureEvent(event)
	}

	return resp, err
}

// CaptureEvent 公开方法：允许外部代码手动捕获事件
// 这对于捕获 Middleware 无法自动感知的事件很有用
func (m *LogicMemoryMiddleware) CaptureEvent(event *logic.Event) {
	m.captureEvent(event)
}

// CaptureUserMessage 捕获用户消息事件
func (m *LogicMemoryMiddleware) CaptureUserMessage(namespace, content string, metadata map[string]any) {
	event := &logic.Event{
		Type:      "user_message",
		Source:    namespace,
		Data:      map[string]any{"content": content},
		Timestamp: time.Now(),
	}
	for k, v := range metadata {
		event.Data[k] = v
	}
	m.captureEvent(event)
}

// CaptureUserFeedback 捕获用户反馈事件
func (m *LogicMemoryMiddleware) CaptureUserFeedback(namespace string, feedback string, rating int, metadata map[string]any) {
	event := &logic.Event{
		Type:   "user_feedback",
		Source: namespace,
		Data: map[string]any{
			"feedback": feedback,
			"rating":   rating,
		},
		Timestamp: time.Now(),
	}
	for k, v := range metadata {
		event.Data[k] = v
	}
	m.captureEvent(event)
}

// CaptureUserRevision 捕获用户修改事件（核心功能：学习用户偏好）
func (m *LogicMemoryMiddleware) CaptureUserRevision(namespace string, original, revised string, metadata map[string]any) {
	event := &logic.Event{
		Type:   "user_revision",
		Source: namespace,
		Data: map[string]any{
			"original": original,
			"revised":  revised,
		},
		Timestamp: time.Now(),
	}
	for k, v := range metadata {
		event.Data[k] = v
	}
	m.captureEvent(event)
}

// captureEvent 内部方法：捕获事件
func (m *LogicMemoryMiddleware) captureEvent(event *logic.Event) {
	if !m.config.EnableCapture {
		return
	}

	if m.config.AsyncCapture && m.eventBuffer != nil {
		// 异步捕获：发送到缓冲区
		select {
		case m.eventBuffer <- event:
			// 成功放入缓冲区
		default:
			// 缓冲区满，丢弃事件（记录警告）
			lmLog.Warn(context.Background(), "event buffer full, dropping event", map[string]any{"type": event.Type})
		}
	} else {
		// 同步捕获：直接处理
		if err := m.manager.ProcessEvent(context.Background(), *event); err != nil {
			lmLog.Error(context.Background(), "failed to process event", map[string]any{"error": err.Error()})
		}
	}
}

// processEventsAsync 异步处理事件的 goroutine
func (m *LogicMemoryMiddleware) processEventsAsync() {
	defer m.wg.Done()

	for {
		select {
		case <-m.stopCh:
			// 处理剩余的事件
			for {
				select {
				case event := <-m.eventBuffer:
					if err := m.manager.ProcessEvent(context.Background(), *event); err != nil {
						lmLog.Error(context.Background(), "failed to process event", map[string]any{"error": err.Error()})
					}
				default:
					return
				}
			}
		case event := <-m.eventBuffer:
			if err := m.manager.ProcessEvent(context.Background(), *event); err != nil {
				lmLog.Error(context.Background(), "failed to process event", map[string]any{"error": err.Error()})
			}
		}
	}
}

// buildMemorySection 构建 Memory 注入文本
func (m *LogicMemoryMiddleware) buildMemorySection(memories []*logic.LogicMemory) string {
	if len(memories) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("## User Preferences and Memory\n\n")
	builder.WriteString("Based on past interactions, I've learned the following about this user:\n\n")

	for i, mem := range memories {
		// 格式化每个 Memory
		confidence := 0.0
		if mem.Provenance != nil {
			confidence = mem.Provenance.Confidence
		}

		builder.WriteString(fmt.Sprintf("%d. **%s** (%s)\n", i+1, mem.Type, mem.Key))
		builder.WriteString(fmt.Sprintf("   - %s\n", mem.Description))
		builder.WriteString(fmt.Sprintf("   - Confidence: %.0f%%\n", confidence*100))

		// 如果有分类，也显示
		if mem.Category != "" {
			builder.WriteString(fmt.Sprintf("   - Category: %s\n", mem.Category))
		}

		builder.WriteString("\n")
	}

	builder.WriteString("**Please apply these preferences naturally in your response.** Do not explicitly mention these memories unless directly relevant.\n")

	return fmt.Sprintf(m.config.SystemPromptTemplate, builder.String())
}

// extractSourceFromToolContext 从工具上下文中提取 source
func (m *LogicMemoryMiddleware) extractSourceFromToolContext(req *ToolCallRequest) string {
	if req.Context != nil {
		// 使用 ThreadID 或 AgentID 作为 source
		if req.Context.ThreadID != "" {
			return "thread:" + req.Context.ThreadID
		}
		if req.Context.AgentID != "" {
			return "agent:" + req.Context.AgentID
		}
	}
	if req.Metadata != nil {
		if namespace, ok := req.Metadata["namespace"].(string); ok {
			return namespace
		}
		if userID, ok := req.Metadata["user_id"].(string); ok {
			return "user:" + userID
		}
	}
	return "unknown"
}

// createLogicMemoryTools 创建 Logic Memory 工具
func (m *LogicMemoryMiddleware) createLogicMemoryTools() []tools.Tool {
	return []tools.Tool{
		NewLogicMemoryQueryTool(m.manager),
		NewLogicMemoryUpdateTool(m.manager),
	}
}

// GetManager 获取 Logic Memory 管理器
func (m *LogicMemoryMiddleware) GetManager() *logic.Manager {
	return m.manager
}

// GetConfig 获取配置信息
func (m *LogicMemoryMiddleware) GetConfig() map[string]any {
	return map[string]any{
		"enable_capture":   m.config.EnableCapture,
		"enable_injection": m.config.EnableInjection,
		"max_memories":     m.config.MaxMemories,
		"min_confidence":   m.config.MinConfidence,
		"async_capture":    m.config.AsyncCapture,
		"injection_point":  m.config.InjectionPoint,
	}
}

// defaultNamespaceExtractor 默认的 namespace 提取器
func defaultNamespaceExtractor(req *ModelRequest) string {
	if req.Metadata == nil {
		return ""
	}

	// 优先使用 namespace
	if namespace, ok := req.Metadata["namespace"].(string); ok && namespace != "" {
		return namespace
	}

	// 其次使用 user_id
	if userID, ok := req.Metadata["user_id"].(string); ok && userID != "" {
		return "user:" + userID
	}

	// 再次使用 tenant_id
	if tenantID, ok := req.Metadata["tenant_id"].(string); ok && tenantID != "" {
		return "tenant:" + tenantID
	}

	// 最后使用 agent_id
	if agentID, ok := req.Metadata["agent_id"].(string); ok && agentID != "" {
		return "agent:" + agentID
	}

	return ""
}

// defaultLogicMemoryPromptTemplate 默认的 Memory 注入模板
const defaultLogicMemoryPromptTemplate = `<logic_memory>
%s
</logic_memory>`
