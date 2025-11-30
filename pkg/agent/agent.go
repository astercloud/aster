package agent

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/commands"
	"github.com/astercloud/aster/pkg/events"
	"github.com/astercloud/aster/pkg/middleware"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/router"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/skills"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/types"
	"github.com/google/uuid"
)

// Agent AI代理
type Agent struct {
	// 基础配置
	id       string
	template *types.AgentTemplateDefinition
	config   *types.AgentConfig
	deps     *Dependencies

	// 核心组件
	eventBus *events.EventBus
	provider provider.Provider
	sandbox  sandbox.Sandbox
	executor *tools.Executor
	toolMap  map[string]tools.Tool

	// Middleware 支持 (Phase 6C)
	middlewareStack *middleware.Stack

	// Slash Commands & Skills 支持
	commandExecutor *commands.Executor
	skillInjector   *skills.Injector

	// 状态管理
	mu           sync.RWMutex
	state        types.AgentRuntimeState
	breakpoint   types.BreakpointState
	messages     []types.Message
	toolRecords  map[string]*types.ToolCallRecord
	runningTools map[string]*runningToolHandle
	stepCount    int
	lastSfpIndex int
	lastBookmark *types.Bookmark
	createdAt    time.Time

	// 权限管理
	pendingPermissions map[string]chan string // callID -> decision channel

	// 控制信号
	stopCh chan struct{}
}

// runningToolHandle 保存可中断工具的句柄
type runningToolHandle struct {
	interruptible tools.Interruptible
}

// Create 创建新Agent
func Create(ctx context.Context, config *types.AgentConfig, deps *Dependencies) (*Agent, error) {
	// 生成AgentID
	if config.AgentID == "" {
		config.AgentID = generateAgentID()
	}

	// 获取模板
	template, err := deps.TemplateRegistry.Get(config.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}

	// 创建Provider（支持可选 Router）
	modelConfig := config.ModelConfig
	if modelConfig == nil && template.Model != "" {
		// 从模型名称推断 provider
		inferredProvider := inferProviderFromModel(template.Model)
		modelConfig = &types.ModelConfig{
			Provider: inferredProvider,
			Model:    template.Model,
		}
		log.Printf("[Agent] Inferred provider '%s' from model '%s'", inferredProvider, template.Model)
	}

	// 如果定义了 Router，则优先通过 Router 决定最终模型
	if deps.Router != nil {
		intent := &router.RouteIntent{
			Task:       "chat",
			Priority:   router.Priority(config.RoutingProfile),
			TemplateID: config.TemplateID,
			Metadata:   config.Metadata,
		}
		// 如果显式传入了 ModelConfig，则作为 Router 的 defaultModel 使用
		if modelConfig != nil {
			defaultModel := modelConfig
			staticRouter, ok := deps.Router.(*router.StaticRouter)
			if ok && staticRouter != nil {
				// 对于 StaticRouter，我们假设其内部默认模型已在构造时设置；
				// 这里不强行覆盖，只在没有配置时作为兜底逻辑留给 Router 自己处理。
				_ = defaultModel
			}
		}

		resolved, err := deps.Router.SelectModel(ctx, intent)
		if err != nil {
			return nil, fmt.Errorf("route model: %w", err)
		}
		modelConfig = resolved
	}

	if modelConfig == nil {
		return nil, fmt.Errorf("model config is required")
	}

	prov, err := deps.ProviderFactory.Create(modelConfig)
	if err != nil {
		return nil, fmt.Errorf("create provider: %w", err)
	}

	// 创建Sandbox
	sandboxConfig := config.Sandbox
	if sandboxConfig == nil {
		sandboxConfig = &types.SandboxConfig{
			Kind:    types.SandboxKindLocal,
			WorkDir: ".",
		}
	}

	sb, err := deps.SandboxFactory.Create(sandboxConfig)
	if err != nil {
		return nil, fmt.Errorf("create sandbox: %w", err)
	}

	// 创建工具执行器
	executor := tools.NewExecutor(tools.ExecutorConfig{
		MaxConcurrency: 3,
		DefaultTimeout: 60 * time.Second,
	})

	// 解析工具列表
	toolNames := config.Tools
	if toolNames == nil {
		// 使用模板的工具列表
		if toolsVal, ok := template.Tools.([]string); ok {
			toolNames = toolsVal
		} else if toolsVal, ok := template.Tools.([]any); ok {
			// 支持 []any 类型（从 JSON 解析后可能是这种类型）
			toolNames = make([]string, 0, len(toolsVal))
			for _, v := range toolsVal {
				if str, ok := v.(string); ok {
					toolNames = append(toolNames, str)
				}
			}
		} else if template.Tools == "*" {
			toolNames = deps.ToolRegistry.List()
		}
	}

	// 创建工具实例
	toolMap := make(map[string]tools.Tool)
	for _, name := range toolNames {
		tool, err := deps.ToolRegistry.Create(name, nil)
		if err != nil {
			log.Printf("[Agent Create] Failed to create tool %s: %v", name, err)
			continue // 忽略未注册的工具
		}
		toolMap[name] = tool
		log.Printf("[Agent Create] Tool loaded: %s, has prompt: %v", name, tool.Prompt() != "")
	}
	log.Printf("[Agent Create] Total tools loaded: %d (names: %v)", len(toolMap), toolNames)

	// 初始化 Slash Commands & Skills（如果配置了）
	var cmdExecutor *commands.Executor
	var skillInjector *skills.Injector

	if config.SkillsPackage != nil {
		// 确定 Skills 包的基础路径
		basePath := config.SkillsPackage.Path
		if basePath == "" {
			basePath = "." // 默认为当前目录（相对于 sandbox workDir）
		}

		// 初始化命令执行器
		commandsDir := config.SkillsPackage.CommandsDir
		if commandsDir == "" {
			commandsDir = "commands"
		}
		// 拼接完整路径：basePath/commandsDir
		fullCommandsDir := filepath.Join(basePath, commandsDir)
		commandLoader := commands.NewLoader(fullCommandsDir, sb.FS())
		cmdExecutor = commands.NewExecutor(&commands.ExecutorConfig{
			Loader:       commandLoader,
			Sandbox:      sb,
			Provider:     prov,
			Capabilities: prov.Capabilities(),
		})

		// 初始化技能注入器
		skillsDir := config.SkillsPackage.SkillsDir
		if skillsDir == "" {
			skillsDir = "skills"
		}
		// 拼接完整路径：basePath/skillsDir
		fullSkillsDir := filepath.Join(basePath, skillsDir)
		skillLoader := skills.NewLoader(fullSkillsDir, sb.FS())
		skillInjector, err = skills.NewInjector(ctx, &skills.InjectorConfig{
			Loader:        skillLoader,
			EnabledSkills: config.SkillsPackage.EnabledSkills,
			Provider:      prov,
			Capabilities:  prov.Capabilities(),
		})
		if err != nil {
			return nil, fmt.Errorf("create skill injector: %w", err)
		}
		// 记录成功加载的 Skills
		if skillInjector != nil {
			log.Printf("[Skills] Successfully created skill injector with %d enabled skills from path: %s",
				len(config.SkillsPackage.EnabledSkills), basePath)
		}
	}

	// 初始化 Middleware Stack (Phase 6C)
	var middlewareStack *middleware.Stack

	// 合并配置的中间件和自动启用的中间件
	middlewareNames := config.Middlewares
	if middlewareNames == nil {
		middlewareNames = []string{}
	}

	// 自动启用 summarization 中间件（如果模板配置了对话压缩）
	if template.Runtime != nil && template.Runtime.ConversationCompression != nil &&
		template.Runtime.ConversationCompression.Enabled {
		// 检查是否已配置 summarization
		hasSummarization := false
		for _, name := range middlewareNames {
			if name == "summarization" {
				hasSummarization = true
				break
			}
		}
		if !hasSummarization {
			middlewareNames = append(middlewareNames, "summarization")
			log.Printf("[Agent Create] Auto-enabled summarization middleware from template ConversationCompression config")

			// 如果模板配置了自定义参数，自动添加到 MiddlewareConfig
			ccConfig := template.Runtime.ConversationCompression
			if config.MiddlewareConfig == nil {
				config.MiddlewareConfig = make(map[string]map[string]any)
			}
			if config.MiddlewareConfig["summarization"] == nil {
				config.MiddlewareConfig["summarization"] = make(map[string]any)
			}
			// 将模板配置转换为中间件配置
			if ccConfig.TokenBudget > 0 {
				// 将 TokenBudget 转换为 max_tokens (按阈值计算)
				threshold := ccConfig.Threshold
				if threshold <= 0 {
					threshold = 0.80
				}
				maxTokens := int(float64(ccConfig.TokenBudget) * threshold)
				config.MiddlewareConfig["summarization"]["max_tokens"] = maxTokens
				log.Printf("[Agent Create] Set summarization max_tokens=%d (budget=%d, threshold=%.2f)",
					maxTokens, ccConfig.TokenBudget, threshold)
			}
			if ccConfig.MinMessagesToKeep > 0 {
				config.MiddlewareConfig["summarization"]["messages_to_keep"] = ccConfig.MinMessagesToKeep
			}
		}
	}

	if len(middlewareNames) > 0 {
		middlewareList := make([]middleware.Middleware, 0, len(middlewareNames))
		for _, name := range middlewareNames {
			var custom map[string]any
			if cfgMap := config.MiddlewareConfig; cfgMap != nil {
				if v, ok := cfgMap[name]; ok {
					custom = v
				}
			}

			mw, err := middleware.DefaultRegistry.Create(name, &middleware.MiddlewareFactoryConfig{
				Provider:     prov,
				AgentID:      config.AgentID,
				Metadata:     config.Metadata,
				Sandbox:      sb,
				CustomConfig: custom,
			})
			if err != nil {
				log.Printf("[Agent Create] Failed to create middleware %s: %v", name, err)
				continue
			}
			middlewareList = append(middlewareList, mw)
			log.Printf("[Agent Create] Middleware loaded: %s (priority: %d)", name, mw.Priority())
		}
		if len(middlewareList) > 0 {
			middlewareStack = middleware.NewStack(middlewareList)
			log.Printf("[Agent Create] Middleware stack created with %d middlewares", len(middlewareList))

			// 将中间件提供的工具合并到 Agent 的工具集中
			if middlewareStack != nil {
				for _, mwTool := range middlewareStack.Tools() {
					if mwTool == nil {
						continue
					}
					name := mwTool.Name()
					if _, exists := toolMap[name]; exists {
						log.Printf("[Agent Create] Middleware tool %s overrides existing tool with same name", name)
					}
					toolMap[name] = mwTool
					log.Printf("[Agent Create] Middleware tool loaded: %s, has prompt: %v", name, mwTool.Prompt() != "")
				}
			}

			log.Printf("[Agent Create] Total tools after middleware injection: %d", len(toolMap))
		}
	}

	// 创建Agent
	agent := &Agent{
		id:                 config.AgentID,
		template:           template,
		config:             config,
		deps:               deps,
		eventBus:           events.NewEventBus(),
		provider:           prov,
		sandbox:            sb,
		executor:           executor,
		toolMap:            toolMap,
		middlewareStack:    middlewareStack,
		commandExecutor:    cmdExecutor,
		skillInjector:      skillInjector,
		state:              types.AgentStateReady,
		breakpoint:         types.BreakpointReady,
		messages:           []types.Message{},
		toolRecords:        make(map[string]*types.ToolCallRecord),
		runningTools:       make(map[string]*runningToolHandle),
		pendingPermissions: make(map[string]chan string),
		createdAt:          time.Now(),
		stopCh:             make(chan struct{}),
	}

	// 使用 PromptBuilder 构建 System Prompt（在初始化之前，因为 initialize 会保存信息）
	if err := agent.buildSystemPrompt(ctx); err != nil {
		return nil, fmt.Errorf("build system prompt: %w", err)
	}

	// 初始化
	if err := agent.initialize(ctx); err != nil {
		return nil, fmt.Errorf("initialize agent: %w", err)
	}

	return agent, nil
}

// initialize 初始化Agent
func (a *Agent) initialize(ctx context.Context) error {
	// 从Store加载状态
	messages, err := a.deps.Store.LoadMessages(ctx, a.id)
	if err == nil && len(messages) > 0 {
		a.messages = messages
	}

	toolRecords, err := a.deps.Store.LoadToolCallRecords(ctx, a.id)
	if err == nil {
		for _, record := range toolRecords {
			a.toolRecords[record.ID] = &record
		}
	}

	// 注意：工具手册已在 Agent 创建时注入，这里不再重复注入

	// 保存Agent信息
	info := types.AgentInfo{
		AgentID:       a.id,
		TemplateID:    a.template.ID,
		CreatedAt:     a.createdAt,
		Lineage:       []string{},
		ConfigVersion: "v1.0.0",
		MessageCount:  len(a.messages),
	}

	if err := a.deps.Store.SaveInfo(ctx, a.id, info); err != nil {
		return err
	}

	// 通知 Middleware Agent 启动 (Phase 6C)
	if a.middlewareStack != nil {
		if err := a.middlewareStack.OnAgentStart(ctx, a.id); err != nil {
			return fmt.Errorf("middleware onAgentStart: %w", err)
		}
	}

	return nil
}

// buildSystemPrompt 使用 PromptBuilder 构建 System Prompt
func (a *Agent) buildSystemPrompt(ctx context.Context) error {
	// 创建 PromptBuilder（支持压缩）
	var builder *PromptBuilder
	if a.deps.PromptCompressor != nil {
		builder = NewPromptBuilderWithCompression(a.deps.PromptCompressor)
	} else {
		builder = NewPromptBuilder()
	}

	// 添加基础模块
	builder.AddModule(&BasePromptModule{})

	// 添加能力说明模块（如果启用）
	builder.AddModule(&CapabilitiesModule{})

	// 添加专业客观性模块（Claude Code 设计原则）
	builder.AddModule(&ProfessionalObjectivityModule{})

	// 添加简洁性模块
	builder.AddModule(&ConcisenessModule{})

	// 添加避免过度工程化模块
	builder.AddModule(&AvoidOverEngineeringModule{})

	// 添加规划指南模块
	builder.AddModule(&PlanningWithoutTimelinesModule{})

	// 收集环境信息
	workDir := "."
	if a.sandbox != nil {
		workDir = a.sandbox.WorkDir()
	}
	envInfo := collectEnvironmentInfo(ctx, workDir, a.createdAt)

	// 添加环境信息模块
	builder.AddModule(&EnvironmentModule{})

	// 添加沙箱信息模块
	builder.AddModule(&SandboxModule{})

	// 添加工具手册模块
	var toolsManualConfig *types.ToolsManualConfig
	if a.template.Runtime != nil && a.template.Runtime.ToolsManual != nil {
		toolsManualConfig = a.template.Runtime.ToolsManual
	}
	builder.AddModule(&ToolsManualModule{Config: toolsManualConfig})

	// 添加 Todo 提醒模块
	var todoConfig *types.TodoConfig
	if a.template.Runtime != nil && a.template.Runtime.Todo != nil {
		todoConfig = a.template.Runtime.Todo
	}
	builder.AddModule(&TodoReminderModule{Config: todoConfig})

	// 添加代码引用模块
	builder.AddModule(&CodeReferenceModule{})

	// 添加 Git 安全模块
	builder.AddModule(&GitSafetyModule{})

	// 添加安全策略模块
	builder.AddModule(&SecurityModule{})

	// 添加性能优化模块
	builder.AddModule(&PerformanceModule{})

	// 添加协作模块（如果在 Room 中）
	if roomInfo := a.extractRoomInfo(); roomInfo != nil {
		builder.AddModule(&CollaborationModule{RoomInfo: roomInfo})
	}

	// 添加工作流模块（如果在 Workflow 中）
	if workflowInfo := a.extractWorkflowInfo(); workflowInfo != nil {
		builder.AddModule(&WorkflowModule{WorkflowInfo: workflowInfo})
	}

	// 添加自定义指令模块
	if customInstructions := a.extractCustomInstructions(); customInstructions != "" {
		builder.AddModule(&CustomInstructionsModule{Instructions: customInstructions})
	}

	// 添加限制说明模块
	builder.AddModule(&LimitationsModule{})

	// 添加上下文窗口管理模块
	if contextConfig := a.extractContextWindowConfig(); contextConfig != nil {
		builder.AddModule(&ContextWindowModule{
			MaxTokens: contextConfig.MaxTokens,
			Strategy:  contextConfig.Strategy,
		})
	}

	// 收集沙箱信息
	var sandboxInfo *SandboxInfo
	if a.sandbox != nil && a.config.Sandbox != nil {
		sandboxInfo = &SandboxInfo{
			Kind:       a.config.Sandbox.Kind,
			WorkDir:    a.sandbox.WorkDir(),
			AllowPaths: a.config.Sandbox.AllowPaths,
		}
	}

	// 构建上下文
	promptCtx := &PromptContext{
		Agent:       a,
		Template:    a.template,
		Environment: envInfo,
		Sandbox:     sandboxInfo,
		Tools:       a.toolMap,
		Metadata:    a.config.Metadata,
	}

	// 构建 System Prompt
	systemPrompt, err := builder.Build(promptCtx)
	if err != nil {
		return fmt.Errorf("build system prompt: %w", err)
	}

	// 更新模板
	a.mu.Lock()
	a.template.SystemPrompt = systemPrompt
	a.mu.Unlock()

	log.Printf("[buildSystemPrompt] Agent %s: Built system prompt, length: %d", a.id, len(systemPrompt))

	return nil
}

// extractRoomInfo 提取 Room 协作信息
func (a *Agent) extractRoomInfo() *RoomCollaborationInfo {
	if a.config.Metadata == nil {
		return nil
	}

	roomID, ok := a.config.Metadata["room_id"].(string)
	if !ok || roomID == "" {
		return nil
	}

	info := &RoomCollaborationInfo{
		RoomID: roomID,
	}

	if memberCount, ok := a.config.Metadata["room_member_count"].(int); ok {
		info.MemberCount = memberCount
	}

	if members, ok := a.config.Metadata["room_members"].([]string); ok {
		info.Members = members
	} else if membersInterface, ok := a.config.Metadata["room_members"].([]any); ok {
		info.Members = make([]string, 0, len(membersInterface))
		for _, m := range membersInterface {
			if str, ok := m.(string); ok {
				info.Members = append(info.Members, str)
			}
		}
	}

	return info
}

// extractWorkflowInfo 提取 Workflow 上下文信息
func (a *Agent) extractWorkflowInfo() *WorkflowContextInfo {
	if a.config.Metadata == nil {
		return nil
	}

	workflowID, ok := a.config.Metadata["workflow_id"].(string)
	if !ok || workflowID == "" {
		return nil
	}

	info := &WorkflowContextInfo{
		WorkflowID: workflowID,
	}

	if currentStep, ok := a.config.Metadata["workflow_current_step"].(string); ok {
		info.CurrentStep = currentStep
	}

	if totalSteps, ok := a.config.Metadata["workflow_total_steps"].(int); ok {
		info.TotalSteps = totalSteps
	}

	if stepIndex, ok := a.config.Metadata["workflow_step_index"].(int); ok {
		info.StepIndex = stepIndex
	}

	if prevStep, ok := a.config.Metadata["workflow_previous_step"].(string); ok {
		info.PreviousStep = prevStep
	}

	if nextStep, ok := a.config.Metadata["workflow_next_step"].(string); ok {
		info.NextStep = nextStep
	}

	return info
}

// extractCustomInstructions 提取自定义指令
func (a *Agent) extractCustomInstructions() string {
	if a.config.Metadata == nil {
		return ""
	}

	if instructions, ok := a.config.Metadata["custom_instructions"].(string); ok {
		return instructions
	}

	return ""
}

// extractContextWindowConfig 提取上下文窗口配置
func (a *Agent) extractContextWindowConfig() *struct {
	MaxTokens int
	Strategy  string
} {
	if a.config.Context == nil {
		return nil
	}

	return &struct {
		MaxTokens int
		Strategy  string
	}{
		MaxTokens: a.config.Context.MaxTokens,
		Strategy:  "auto", // 可以从配置中读取
	}
}

// injectToolManual 注入工具手册到系统提示词
// Deprecated: 使用 buildSystemPrompt 替代
func (a *Agent) injectToolManual() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.toolMap) == 0 {
		log.Printf("[injectToolManual] Agent %s: No tools in toolMap, skipping", a.id)
		return
	}

	// 收集简要工具手册 (只保留简短摘要, 避免 System Prompt 过度膨胀)
	type toolSummary struct {
		name    string
		summary string
	}
	summaries := make([]toolSummary, 0, len(a.toolMap))

	// 解析模板级工具手册配置
	mode := "all"
	includeSet := map[string]struct{}{}
	excludeSet := map[string]struct{}{}
	if a.template.Runtime != nil && a.template.Runtime.ToolsManual != nil {
		cfg := a.template.Runtime.ToolsManual
		if cfg.Mode != "" {
			mode = cfg.Mode
		}
		for _, name := range cfg.Include {
			includeSet[name] = struct{}{}
		}
		for _, name := range cfg.Exclude {
			excludeSet[name] = struct{}{}
		}
	}

	shouldInclude := func(name string) bool {
		switch mode {
		case "none":
			return false
		case "listed":
			_, ok := includeSet[name]
			return ok
		default: // "all" 或未知值
			if _, blocked := excludeSet[name]; blocked {
				return false
			}
			return true
		}
	}

	for name, tool := range a.toolMap {
		if !shouldInclude(name) {
			continue
		}

		prompt := tool.Prompt()
		summary := ""
		if prompt != "" {
			lines := strings.Split(prompt, "\n")
			if len(lines) > 0 {
				summary = strings.TrimSpace(lines[0])
			}
		}
		if summary == "" {
			summary = strings.TrimSpace(tool.Description())
		}
		if summary == "" {
			summary = "No detailed manual; infer from tool name and input schema."
		}
		summaries = append(summaries, toolSummary{name: name, summary: summary})
	}

	if len(summaries) == 0 {
		log.Printf("[injectToolManual] Agent %s: No tools found, skipping", a.id)
		return
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].name < summaries[j].name
	})

	var lines []string
	for _, s := range summaries {
		lines = append(lines, fmt.Sprintf("- `%s`: %s", s.name, s.summary))
		log.Printf("[injectToolManual] Agent %s: Added summary for tool %s", a.id, s.name)
	}

	manualSection := "\n\n### Tools Manual\n\n" +
		"The following tools are available for your use. " +
		"Use them when appropriate instead of doing everything in natural language.\n\n" +
		strings.Join(lines, "\n")

	// 检查系统提示词是否已包含工具手册
	if strings.Contains(a.template.SystemPrompt, "### Tools Manual") {
		// 移除旧的工具手册
		parts := strings.Split(a.template.SystemPrompt, "### Tools Manual")
		if len(parts) > 0 {
			a.template.SystemPrompt = strings.TrimSpace(parts[0])
		}
	}

	// 追加新的工具手册
	oldLength := len(a.template.SystemPrompt)
	a.template.SystemPrompt += manualSection
	log.Printf("[injectToolManual] Agent %s: Injected manual, system prompt length: %d -> %d", a.id, oldLength, len(a.template.SystemPrompt))
}

// ID 返回AgentID
func (a *Agent) ID() string {
	return a.id
}

// Send 发送消息
func (a *Agent) Send(ctx context.Context, text string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// 检测 slash command
	if strings.HasPrefix(text, "/") {
		return a.handleSlashCommand(ctx, text)
	}

	// 准备消息内容
	messageText := text

	// 如果启用了 skills，增强消息
	if a.skillInjector != nil {
		skillContext := skills.SkillContext{
			UserMessage: text,
			Files:       a.getRecentFiles(),
			Metadata:    make(map[string]any),
		}

		// 增强 system prompt（对于支持的模型）
		caps := a.provider.Capabilities()
		if caps.SupportSystemPrompt {
			enhancedSysPrompt := a.skillInjector.EnhanceSystemPrompt(
				ctx,
				a.template.SystemPrompt,
				skillContext,
			)
			_ = a.provider.SetSystemPrompt(enhancedSysPrompt)
		} else {
			// 不支持 system prompt，增强 user message
			messageText = a.skillInjector.PrepareUserMessage(text, skillContext)
		}
	}

	// 创建用户消息
	message := types.Message{
		Role: types.MessageRoleUser,
		ContentBlocks: []types.ContentBlock{
			&types.TextBlock{Text: messageText},
		},
	}

	a.messages = append(a.messages, message)
	a.stepCount++

	// 持久化
	if err := a.deps.Store.SaveMessages(ctx, a.id, a.messages); err != nil {
		return fmt.Errorf("save messages: %w", err)
	}

	// 触发处理
	go a.processMessages(ctx)

	return nil
}

// Chat 同步对话(阻塞式)
func (a *Agent) Chat(ctx context.Context, text string) (*types.CompleteResult, error) {
	// 发送消息
	if err := a.Send(ctx, text); err != nil {
		return nil, err
	}

	// 等待完成
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
			a.mu.RLock()
			state := a.state
			a.mu.RUnlock()

			if state == types.AgentStateReady {
				// 提取最后的助手回复
				a.mu.RLock()
				defer a.mu.RUnlock()

				var text string
				for i := len(a.messages) - 1; i >= 0; i-- {
					if a.messages[i].Role == types.MessageRoleAssistant {
						for _, block := range a.messages[i].ContentBlocks {
							if tb, ok := block.(*types.TextBlock); ok {
								text = tb.Text
								break
							}
						}
						break
					}
				}

				return &types.CompleteResult{
					Status: "ok",
					Text:   text,
					Last:   a.lastBookmark,
				}, nil
			}
		}
	}
}

// Subscribe 订阅事件
func (a *Agent) Subscribe(channels []types.AgentChannel, opts *types.SubscribeOptions) <-chan types.AgentEventEnvelope {
	return a.eventBus.Subscribe(channels, opts)
}

// Unsubscribe 取消事件订阅
func (a *Agent) Unsubscribe(ch <-chan types.AgentEventEnvelope) {
	a.eventBus.Unsubscribe(ch)
}

// Status 获取状态
func (a *Agent) Status() *types.AgentStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return &types.AgentStatus{
		AgentID:      a.id,
		State:        a.state,
		StepCount:    a.stepCount,
		LastSfpIndex: a.lastSfpIndex,
		LastBookmark: a.lastBookmark,
		Cursor:       a.eventBus.GetCursor(),
		Breakpoint:   a.breakpoint,
	}
}

// GetSystemPrompt 获取当前的 System Prompt
func (a *Agent) GetSystemPrompt() string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.template.SystemPrompt
}

// ExecuteToolDirect 直接执行工具（程序化工具调用）
// 这个方法允许 Agent 或外部代码直接调用工具，绕过 LLM 决策
// 主要用于程序化工具编排场景
func (a *Agent) ExecuteToolDirect(ctx context.Context, toolName string, input map[string]any) (any, error) {
	a.mu.RLock()
	tool, exists := a.toolMap[toolName]
	a.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("tool not found: %s", toolName)
	}

	// 构建工具上下文
	tc := a.buildToolContext(ctx)

	// 执行工具
	result, err := tool.Execute(ctx, input, tc)
	if err != nil {
		return nil, fmt.Errorf("execute tool %s: %w", toolName, err)
	}

	return result, nil
}

// ExecuteToolsDirect 批量直接执行工具（顺序执行）
func (a *Agent) ExecuteToolsDirect(ctx context.Context, calls []ToolCall) []ToolCallResult {
	results := make([]ToolCallResult, len(calls))

	for i, call := range calls {
		result, err := a.ExecuteToolDirect(ctx, call.Name, call.Input)
		if err != nil {
			results[i] = ToolCallResult{
				Name:    call.Name,
				Success: false,
				Error:   err.Error(),
			}
		} else {
			results[i] = ToolCallResult{
				Name:    call.Name,
				Success: true,
				Result:  result,
			}
		}
	}

	return results
}

// ToolCall 工具调用参数
type ToolCall struct {
	Name  string         `json:"name"`
	Input map[string]any `json:"input"`
}

// ToolCallResult 工具调用结果
type ToolCallResult struct {
	Name    string `json:"name"`
	Success bool   `json:"success"`
	Result  any    `json:"result,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Close 关闭Agent
func (a *Agent) Close() error {
	close(a.stopCh)

	// 通知 Middleware Agent 停止 (Phase 6C)
	if a.middlewareStack != nil {
		ctx := context.Background()
		if err := a.middlewareStack.OnAgentStop(ctx, a.id); err != nil {
			log.Printf("[Agent Close] Middleware OnAgentStop error: %v", err)
		}
	}

	if err := a.sandbox.Dispose(); err != nil {
		return err
	}

	return a.provider.Close()
}

// buildToolContext 构造工具执行上下文, 注入必要的服务。
// 当前会注入:
//   - tool_manuals: 工具手册映射，供 ToolHelp 等工具使用
//   - skills_runtime: *skills.Runtime, 供 skill_call 工具使用 (仅当 Agent 配置了 SkillsPackage 时)
func (a *Agent) buildToolContext(ctx context.Context) *tools.ToolContext {
	tc := &tools.ToolContext{
		AgentID:  a.id,
		Sandbox:  a.sandbox,
		Signal:   ctx,
		Services: make(map[string]any),
	}

	// 为 ToolHelp 等工具注入当前可用工具的手册信息, 支持按需查询。
	if len(a.toolMap) > 0 {
		manuals := make(map[string]string, len(a.toolMap))
		for name, tool := range a.toolMap {
			if prompt := tool.Prompt(); prompt != "" {
				manuals[name] = prompt
			}
		}
		if len(manuals) > 0 {
			tc.Services["tool_manuals"] = manuals
		}
	}

	// 如果 Agent 启用了 SkillsPackage, 为工具注入 Skills Runtime
	if a.config != nil && a.config.SkillsPackage != nil {
		basePath := a.config.SkillsPackage.Path
		if basePath == "" {
			basePath = "." // 相对于 sandbox.WorkDir
		}
		skillsDir := a.config.SkillsPackage.SkillsDir
		if skillsDir == "" {
			skillsDir = "skills"
		}
		fullSkillsDir := filepath.Join(basePath, skillsDir)
		loader := skills.NewLoader(fullSkillsDir, a.sandbox.FS())
		rt := skills.NewRuntime(loader, a.sandbox)
		tc.Services["skills_runtime"] = rt
	}

	return tc
}

// handleSlashCommand 处理 slash command
func (a *Agent) handleSlashCommand(ctx context.Context, text string) error {
	if a.commandExecutor == nil {
		log.Printf("[Command] ERROR: Slash commands not enabled for agent %s", a.id)
		return fmt.Errorf("slash commands not enabled")
	}

	// 解析命令和参数
	parts := strings.Fields(text)
	commandName := strings.TrimPrefix(parts[0], "/")

	args := make(map[string]string)
	if len(parts) > 1 {
		args["argument"] = strings.Join(parts[1:], " ")
	}

	log.Printf("[Command] Agent %s: Executing command /%s with args: %v", a.id, commandName, args)

	// 执行命令并获取消息
	message, err := a.commandExecutor.Execute(ctx, commandName, args)
	if err != nil {
		log.Printf("[Command] ERROR: Agent %s failed to execute /%s: %v", a.id, commandName, err)
		return fmt.Errorf("execute command: %w", err)
	}

	log.Printf("[Command] Agent %s: Command /%s executed successfully, generated message length: %d", a.id, commandName, len(message))

	// 将命令消息作为用户消息发送
	userMessage := types.Message{
		Role: types.MessageRoleUser,
		ContentBlocks: []types.ContentBlock{
			&types.TextBlock{Text: message},
		},
	}

	a.messages = append(a.messages, userMessage)
	a.stepCount++

	// 持久化
	if err := a.deps.Store.SaveMessages(ctx, a.id, a.messages); err != nil {
		return fmt.Errorf("save messages: %w", err)
	}

	log.Printf("[Command] Agent %s: Command /%s processing started", a.id, commandName)

	// 触发处理
	go a.processMessages(ctx)

	return nil
}

// getRecentFiles 获取最近访问的文件列表
func (a *Agent) getRecentFiles() []string {
	// TODO: 实现文件追踪逻辑
	// 可以从 toolRecords 中提取最近读写的文件
	return []string{}
}

// registerRunningTool 记录可中断工具句柄
func (a *Agent) registerRunningTool(id string, intr tools.Interruptible) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.runningTools == nil {
		a.runningTools = make(map[string]*runningToolHandle)
	}
	a.runningTools[id] = &runningToolHandle{interruptible: intr}
}

// unregisterRunningTool 移除运行中工具记录
func (a *Agent) unregisterRunningTool(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.runningTools, id)
}

type longRunningInterruptible struct {
	tool   tools.LongRunningTool
	taskID string
}

func (l *longRunningInterruptible) Pause() error {
	return fmt.Errorf("pause not supported for long-running task")
}
func (l *longRunningInterruptible) Resume() error {
	return fmt.Errorf("resume not supported for long-running task")
}
func (l *longRunningInterruptible) Cancel() error {
	return l.tool.Cancel(context.Background(), l.taskID)
}

// ControlTool 执行对运行中工具的控制
func (a *Agent) ControlTool(callID, action, note string) error {
	// 记录入站控制事件
	a.eventBus.EmitControl(&types.ControlToolControlEvent{
		CallID: callID,
		Action: action,
		Note:   note,
	})

	err := a.controlRunningTool(callID, action)
	ok := err == nil
	reason := ""
	if err != nil {
		reason = err.Error()
	}

	// 输出响应事件
	a.eventBus.EmitControl(&types.ControlToolControlResponseEvent{
		CallID: callID,
		Action: action,
		OK:     ok,
		Reason: reason,
	})

	// 如果取消成功，推送取消进度事件
	if ok && action == "cancel" {
		a.eventBus.EmitProgress(&types.ProgressToolCancelledEvent{
			Call:   a.snapshotToolCall(callID),
			Reason: "cancelled",
		})
	}

	return err
}

func (a *Agent) controlRunningTool(callID, action string) error {
	a.mu.RLock()
	handle, ok := a.runningTools[callID]
	a.mu.RUnlock()

	if !ok || handle == nil || handle.interruptible == nil {
		return fmt.Errorf("tool not interruptible or not running")
	}

	switch action {
	case "pause":
		return handle.interruptible.Pause()
	case "resume":
		return handle.interruptible.Resume()
	case "cancel":
		a.updateToolRecord(callID, types.ToolCallStateCancelling, "")
		return handle.interruptible.Cancel()
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

func (a *Agent) snapshotToolCall(callID string) types.ToolCallSnapshot {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if rec, ok := a.toolRecords[callID]; ok && rec != nil {
		return types.ToolCallSnapshot{
			ID:           rec.ID,
			Name:         rec.Name,
			State:        rec.State,
			Arguments:    rec.Input,
			Result:       rec.Result,
			Error:        rec.Error,
			Progress:     rec.Progress,
			Intermediate: rec.Intermediate,
			StartedAt:    rec.StartTime,
			UpdatedAt:    rec.UpdatedAt,
			Cancelable:   a.runningTools[rec.ID] != nil,
			Pausable:     a.runningTools[rec.ID] != nil,
		}
	}

	return types.ToolCallSnapshot{ID: callID}
}

// GetToolSnapshot 获取指定调用的实时快照
func (a *Agent) GetToolSnapshot(callID string) types.ToolCallSnapshot {
	return a.snapshotToolCall(callID)
}

// ListRunningToolSnapshots 列出运行中的工具调用
func (a *Agent) ListRunningToolSnapshots() []types.ToolCallSnapshot {
	a.mu.RLock()
	defer a.mu.RUnlock()

	snaps := make([]types.ToolCallSnapshot, 0, len(a.toolRecords))
	for _, rec := range a.toolRecords {
		if rec == nil {
			continue
		}
		if rec.State == types.ToolCallStateExecuting || rec.State == types.ToolCallStateQueued || rec.State == types.ToolCallStatePending || rec.State == types.ToolCallStateCancelling {
			snaps = append(snaps, types.ToolCallSnapshot{
				ID:           rec.ID,
				Name:         rec.Name,
				State:        rec.State,
				Arguments:    rec.Input,
				Result:       rec.Result,
				Error:        rec.Error,
				Progress:     rec.Progress,
				Intermediate: rec.Intermediate,
				StartedAt:    rec.StartTime,
				UpdatedAt:    rec.UpdatedAt,
				Cancelable:   a.runningTools[rec.ID] != nil,
				Pausable:     a.runningTools[rec.ID] != nil,
			})
		}
	}
	return snaps
}

// makeToolReporter 创建工具执行 Reporter
func (a *Agent) makeToolReporter(callID, toolName string) tools.Reporter {
	return &toolReporter{
		agent:    a,
		callID:   callID,
		toolName: toolName,
	}
}

// handleToolProgress 处理进度事件并推送到总线
func (a *Agent) handleToolProgress(callID, toolName string, progress float64, message string, step, total int, metadata map[string]any, etaMs int64) {
	now := time.Now()
	snapshot := types.ToolCallSnapshot{
		ID:        callID,
		Name:      toolName,
		State:     types.ToolCallStateExecuting,
		Progress:  progress,
		UpdatedAt: now,
	}

	a.mu.Lock()
	if rec, ok := a.toolRecords[callID]; ok {
		rec.Progress = progress
		rec.State = types.ToolCallStateExecuting
		rec.UpdatedAt = now
		snapshot.Arguments = rec.Input
		snapshot.StartedAt = rec.StartTime
		snapshot.UpdatedAt = rec.UpdatedAt
	}
	a.mu.Unlock()

	a.eventBus.EmitProgress(&types.ProgressToolProgressEvent{
		Call:     snapshot,
		Progress: progress,
		Message:  message,
		Step:     step,
		Total:    total,
		Metadata: metadata,
		ETA:      etaMs,
	})
}

// handleToolIntermediate 处理中间结果事件
func (a *Agent) handleToolIntermediate(callID, toolName, label string, data any) {
	now := time.Now()
	snapshot := types.ToolCallSnapshot{
		ID:           callID,
		Name:         toolName,
		State:        types.ToolCallStateExecuting,
		Intermediate: make(map[string]any),
		UpdatedAt:    now,
	}

	a.mu.Lock()
	if rec, ok := a.toolRecords[callID]; ok {
		if rec.Intermediate == nil {
			rec.Intermediate = make(map[string]any)
		}
		if label == "" {
			label = "data"
		}
		rec.Intermediate[label] = data
		rec.State = types.ToolCallStateExecuting
		rec.UpdatedAt = now
		snapshot.Arguments = rec.Input
		snapshot.StartedAt = rec.StartTime
		snapshot.UpdatedAt = rec.UpdatedAt
		snapshot.Intermediate = rec.Intermediate
	} else {
		if label == "" {
			label = "data"
		}
		snapshot.Intermediate[label] = data
	}
	a.mu.Unlock()

	a.eventBus.EmitProgress(&types.ProgressToolIntermediateEvent{
		Call:  snapshot,
		Label: label,
		Data:  data,
	})
}

// toolReporter 将工具回调转换为事件
type toolReporter struct {
	agent    *Agent
	callID   string
	toolName string
}

func (tr *toolReporter) Progress(progress float64, message string, step, total int, metadata map[string]any, etaMs int64) {
	tr.agent.handleToolProgress(tr.callID, tr.toolName, progress, message, step, total, metadata, etaMs)
}

func (tr *toolReporter) Intermediate(label string, data any) {
	tr.agent.handleToolIntermediate(tr.callID, tr.toolName, label, data)
}

// generateAgentID 生成AgentID
func generateAgentID() string {
	// 使用不包含文件系统保留字符的格式，避免在 Windows 等平台上
	// 将 AgentID 直接作为目录名时出现非法路径问题。
	// 例如：agt-9d25d66f-ff93-414b-b7e9-59cc294f5815
	return "agt-" + uuid.New().String()
}

// getExecutionMode 获取执行模式
func (a *Agent) getExecutionMode() types.ExecutionMode {
	if a.config != nil && a.config.ModelConfig != nil && a.config.ModelConfig.ExecutionMode != "" {
		return a.config.ModelConfig.ExecutionMode
	}
	return types.ExecutionModeStreaming // 默认流式（向后兼容）
}

// inferProviderFromModel 从模型名称推断 provider
func inferProviderFromModel(model string) string {
	model = strings.ToLower(model)
	switch {
	case strings.HasPrefix(model, "deepseek"):
		return "deepseek"
	case strings.HasPrefix(model, "gpt") || strings.HasPrefix(model, "o1") || strings.HasPrefix(model, "o3"):
		return "openai"
	case strings.HasPrefix(model, "claude"):
		return "anthropic"
	case strings.HasPrefix(model, "gemini"):
		return "google"
	case strings.HasPrefix(model, "llama") || strings.HasPrefix(model, "mistral"):
		return "ollama"
	case strings.HasPrefix(model, "kimi") || strings.HasPrefix(model, "moonshot"):
		return "moonshot"
	case strings.HasPrefix(model, "glm"):
		return "glm"
	default:
		return "anthropic" // 默认 anthropic
	}
}
