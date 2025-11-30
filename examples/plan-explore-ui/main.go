// PlanExploreUI 演示一个简单的 Plan/Explore 命令行 UI，使用 TodoWrite、
// Task、Read、Grep 等工具分析代码，并通过事件订阅渲染工具调用过程。
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/builtin"
	"github.com/astercloud/aster/pkg/types"
)

// Plan(分析 builtin 工具测试需求)
//
//	└─ Read(pkg/tools/builtin/edit.go)
//	└─ Read(pkg/tools/builtin/utils.go)
//
// Explore(分析当前工具实现状态)
//
//	└─ Read(pkg/tools/builtin/task.go)
//	└─ Read(pkg/tools/builtin/subagent_manager.go)
func main() {
	mode := flag.String("mode", "web", "UI mode: web 或 cli")
	addr := flag.String("addr", ":8080", "Web UI HTTP 监听地址")
	flag.Parse()

	// 读取 provider / model / api key, 支持通过环境变量切换模型厂商。
	providerName := os.Getenv("PROVIDER")
	if providerName == "" {
		providerName = "anthropic"
	}

	modelName := os.Getenv("MODEL")
	if modelName == "" {
		switch providerName {
		case "deepseek":
			modelName = "deepseek-chat"
		default:
			modelName = "claude-sonnet-4-5"
		}
	}

	var apiKeyEnv string
	switch providerName {
	case "deepseek":
		apiKeyEnv = "DEEPSEEK_API_KEY"
	case "anthropic":
		apiKeyEnv = "ANTHROPIC_API_KEY"
	default:
		apiKeyEnv = "API_KEY"
	}

	apiKey := os.Getenv(apiKeyEnv)
	if apiKey == "" {
		log.Printf("[WARN] %s not set, provider=%s model calls may fail.", apiKeyEnv, providerName)
	}

	log.Printf("[plan-explore-ui] Using provider=%s model=%s apiKeyEnv=%s", providerName, modelName, apiKeyEnv)

	ctx := context.Background()

	// 1. 工具注册表 + 内置工具
	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry)

	// 2. Sandbox 工厂
	sbFactory := sandbox.NewFactory()

	// 3. Provider 工厂 (支持多种模型厂商)
	providerFactory := provider.NewMultiProviderFactory()

	// 4. Store (JSON 文件存储，便于调试)
	storePath := ".aster-plan-explore-ui"
	jsonStore, err := store.NewJSONStore(storePath)
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}

	// 5. 模板注册表 + 专用模板
	templateRegistry := agent.NewTemplateRegistry()
	templateID := "plan-explore-ui"

	templateRegistry.Register(&types.AgentTemplateDefinition{
		ID:    templateID,
		Model: modelName,
		SystemPrompt: "" +
			"You are a coding assistant. IMPORTANT RULES:\n" +
			"1. You MUST use tools to complete tasks. NEVER provide text-only answers without using tools.\n" +
			"2. To inspect files, you MUST use Read, Glob, or Grep tools.\n" +
			"3. To create a task plan, you MUST use TodoWrite or write_todos tool first.\n" +
			"4. After reading files, you MUST use ExitPlanMode to present your findings in Markdown.\n" +
			"5. For complex tasks, break them down: Plan (use TodoWrite) -> Explore (use Read/Grep) -> Report (use ExitPlanMode).\n" +
			"\n" +
			"Available tools: Read, Write, Edit, Glob, Grep, Bash, TodoWrite, write_todos, Task, ExitPlanMode, Ls.\n" +
			"Start by calling TodoWrite to create a task list, then use Read/Grep/Glob to explore code.\n",
		Tools: []any{
			"Read", "Write", "Edit", "Glob", "Grep", "Bash", "Ls",
			"TodoWrite", "write_todos", "Task", "ExitPlanMode",
		},
	})

	deps := &agent.Dependencies{
		Store:            jsonStore,
		SandboxFactory:   sbFactory,
		ToolRegistry:     toolRegistry,
		ProviderFactory:  providerFactory,
		TemplateRegistry: templateRegistry,
	}

	switch *mode {
	case "cli":
		runCliDemo(ctx, deps, templateID, providerName, modelName, apiKey)
	default:
		if err := runWebUI(ctx, deps, templateID, providerName, modelName, apiKey, *addr); err != nil {
			log.Fatalf("plan-explore-ui web server failed: %v", err)
		}
	}
}

// runCliDemo 保留原来的终端 UI 行为。
func runCliDemo(ctx context.Context, deps *agent.Dependencies, templateID, providerName, modelName, apiKey string) {
	ag, err := agent.Create(ctx, &types.AgentConfig{
		TemplateID: templateID,
		ModelConfig: &types.ModelConfig{
			Provider: providerName,
			Model:    modelName,
			APIKey:   apiKey,
		},
		Sandbox: &types.SandboxConfig{
			Kind:    types.SandboxKindLocal,
			WorkDir: ".",
		},
		Middlewares: []string{"filesystem", "todolist"},
	}, deps)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	defer func() { _ = ag.Close() }()

	fmt.Printf("Plan/Explore UI demo agent created: %s\n\n", ag.ID())

	// 7. 订阅事件，构建简单的「阶段视图」UI
	eventCh := ag.Subscribe([]types.AgentChannel{
		types.ChannelProgress,
		types.ChannelMonitor,
	}, nil)

	ui := &uiState{}

	go func() {
		for envelope := range eventCh {
			evt, ok := envelope.Event.(types.EventType)
			if !ok {
				continue
			}

			switch evt.Channel() {
			case types.ChannelProgress:
				handleProgressEvent(ui, envelope.Event)
			case types.ChannelMonitor:
				handleMonitorEventUI(envelope.Event)
			}
		}
	}()

	// 8. 发送一个示例请求，要求 Agent 先规划再探索 builtin 工具
	userPrompt := `请帮我完成一个两阶段的代码分析任务:
1. 规划(Plan): 分析 agentsdk 仓库里 pkg/tools/builtin 目录下各个工具的职责和测试需求, 给出一个分步骤的实施计划。
2. 探索(Explore): 按计划实际阅读相关文件, 重点关注 TodoWrite / ExitPlanMode / Task / subagent_manager 等实现细节。

要求:
- 在规划阶段, 使用 TodoWrite 或 write_todos 工具创建任务列表, 至少包含 "分析 builtin 工具测试需求" 和 "分析当前工具实现状态" 两个任务。
- 在探索阶段, 使用 Read / Glob / Grep 等工具读取具体文件。
- 规划完成后, 调用 ExitPlanMode 返回一个 Markdown 格式的计划。`

	fmt.Println("User:")
	fmt.Println(userPrompt)
	fmt.Println("\n--- Assistant (streaming with Plan/Explore UI) ---")

	_, err = ag.Chat(ctx, userPrompt)
	if err != nil {
		log.Printf("Chat failed: %v", err)
	}

	// 简单等待一会儿, 让事件消费完成
	time.Sleep(2 * time.Second)

	status := ag.Status()
	fmt.Printf("\n\nFinal Status:\n  Agent ID: %s\n  State: %s\n  Steps: %d\n",
		status.AgentID, status.State, status.StepCount)
}

// runWebUI 启动一个带简单前端的 HTTP 服务。
// 默认访问地址: http://localhost:8080
func runWebUI(_ context.Context, deps *agent.Dependencies, templateID, providerName, modelName, apiKey, addr string) error {
	mux := http.NewServeMux()

	// 静态前端资源
	fs := http.FileServer(http.Dir("examples/plan-explore-ui/static"))
	mux.Handle("/", fs)

	// SSE 接口: /api/chat/stream
	mux.HandleFunc("/api/chat/stream", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		_, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		var req struct {
			Input string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		input := req.Input
		if input == "" {
			input = defaultPrompt()
		}

		reqCtx := r.Context()
		reqCtx, cancel := context.WithTimeout(reqCtx, 5*time.Minute)
		defer cancel()

		ag, err := agent.Create(reqCtx, &types.AgentConfig{
			TemplateID: templateID,
			ModelConfig: &types.ModelConfig{
				Provider: providerName,
				Model:    modelName,
				APIKey:   apiKey,
			},
			Sandbox: &types.SandboxConfig{
				Kind:    types.SandboxKindLocal,
				WorkDir: ".",
			},
			Middlewares: []string{"filesystem", "todolist"},
		}, deps)
		if err != nil {
			http.Error(w, "create agent failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer func() { _ = ag.Close() }()

		eventCh := ag.Subscribe([]types.AgentChannel{
			types.ChannelProgress,
			types.ChannelMonitor,
		}, nil)
		defer ag.Unsubscribe(eventCh)

		// SSE 头
		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// 启动聊天
		go func() {
			_, _ = ag.Chat(reqCtx, input)
		}()

		enc := json.NewEncoder(w)

		for {
			select {
			case <-reqCtx.Done():
				return
			case env, ok := <-eventCh:
				if !ok {
					return
				}

				evt, ok := env.Event.(types.EventType)
				if !ok {
					continue
				}

				payload := map[string]any{}

				switch e := env.Event.(type) {
				case *types.ProgressTextChunkStartEvent:
					payload["step"] = e.Step
				case *types.ProgressTextChunkEvent:
					payload["step"] = e.Step
					payload["delta"] = e.Delta
				case *types.ProgressTextChunkEndEvent:
					payload["step"] = e.Step
					payload["text"] = e.Text
				case *types.ProgressToolStartEvent:
					payload["call"] = e.Call
				case *types.ProgressToolEndEvent:
					payload["call"] = e.Call
				case *types.ProgressToolErrorEvent:
					payload["call"] = e.Call
					payload["error"] = e.Error
				case *types.ProgressDoneEvent:
					payload["step"] = e.Step
					payload["reason"] = e.Reason
				case *types.MonitorStateChangedEvent:
					// 可以添加对状态变化的处理
				}

				// 发送 SSE 事件
				uiEvt := uiEvent{
					Type:    evt.EventType(),
					Payload: payload,
				}

				flusher, ok := w.(http.Flusher)
				if !ok {
					return
				}

				_, _ = w.Write([]byte("data: "))
				if err := enc.Encode(uiEvt); err != nil {
					return
				}
				_, _ = w.Write([]byte("\n\n"))
				flusher.Flush()

				if evt.EventType() == "done" {
					return
				}
			}
		}
	})

	s := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	fmt.Printf("Plan/Explore web UI started: http://localhost%s\n", addr)
	fmt.Println("  POST /api/chat/stream  (SSE, internal use by frontend)")

	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// ========== UI 状态 & 渲染 ==========

type phase struct {
	Kind      string
	Title     string
	ToolCount int
}

type uiState struct {
	current *phase
}

func handleProgressEvent(state *uiState, event any) {
	switch e := event.(type) {
	case *types.ProgressTextChunkStartEvent:
		fmt.Print("\nAssistant: ")
	case *types.ProgressTextChunkEvent:
		fmt.Print(e.Delta)
	case *types.ProgressTextChunkEndEvent:
		// no-op
	case *types.ProgressToolStartEvent:
		renderToolStart(state, &e.Call)
	case *types.ProgressToolErrorEvent:
		fmt.Printf("\n[Tool Error] %s - %s\n", e.Call.Name, e.Error)
	case *types.ProgressDoneEvent:
		fmt.Printf("\n[Done] Step %d - Reason: %s\n", e.Step, e.Reason)
	}
}

func handleMonitorEventUI(event any) {
	switch e := event.(type) {
	case *types.MonitorStateChangedEvent:
		fmt.Printf("\n[State] %s\n", e.State)
	case *types.MonitorBreakpointChangedEvent:
		// 断点变化频繁，这里只在需要时打印
		_ = e
	}
}

func renderToolStart(state *uiState, call *types.ToolCallSnapshot) {
	switch call.Name {
	case "TodoWrite", "write_todos":
		title := extractActiveTaskTitle(call.Arguments)
		if title == "" {
			title = "任务规划"
		}
		fmt.Printf("\nPlan(%s)\n", title)
		state.current = &phase{
			Kind:      "Plan",
			Title:     title,
			ToolCount: 0,
		}
	case "Task", "task":
		subType := getStringArg(call.Arguments, "subagent_type")
		prompt := getStringArg(call.Arguments, "prompt")
		if prompt == "" {
			prompt = "子代理任务"
		}
		labelType := subType
		if labelType == "" {
			labelType = "Task"
		}
		fmt.Printf("\n%s(%s)\n", labelType, prompt)
		state.current = &phase{
			Kind:      labelType,
			Title:     prompt,
			ToolCount: 0,
		}
	default:
		label := formatToolCallLabel(call)
		if state.current != nil {
			if state.current.ToolCount == 0 {
				fmt.Printf("  └─ %s\n", label)
			} else {
				fmt.Printf("     %s\n", label)
			}
			state.current.ToolCount++
		} else {
			fmt.Printf("\n[Tool] %s\n", label)
		}
	}
}

func extractActiveTaskTitle(args map[string]any) string {
	if args == nil {
		return ""
	}

	todosRaw, ok := args["todos"]
	if !ok {
		return ""
	}

	todoSlice, ok := todosRaw.([]any)
	if !ok {
		return ""
	}

	for _, item := range todoSlice {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		status := getStringArg(m, "status")
		if status == "in_progress" {
			activeForm := getStringArg(m, "activeForm")
			if activeForm != "" {
				return activeForm
			}
			content := getStringArg(m, "content")
			if content != "" {
				return content
			}
		}
	}

	return ""
}

func getStringArg(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func formatToolCallLabel(call *types.ToolCallSnapshot) string {
	if call == nil {
		return ""
	}

	args := call.Arguments
	switch call.Name {
	case "Read":
		path := getStringArg(args, "file_path")
		if path != "" {
			return fmt.Sprintf("Read(%s)", path)
		}
	case "Glob":
		pattern := getStringArg(args, "pattern")
		root := getStringArg(args, "root")
		if pattern != "" {
			if root != "" {
				return fmt.Sprintf("Glob(%s, root=%s)", pattern, root)
			}
			return fmt.Sprintf("Glob(%s)", pattern)
		}
	case "Grep":
		pattern := getStringArg(args, "pattern")
		path := getStringArg(args, "path")
		if pattern != "" {
			if path != "" {
				return fmt.Sprintf("Grep(%q in %s)", pattern, path)
			}
			return fmt.Sprintf("Grep(%q)", pattern)
		}
	case "Bash":
		cmd := getStringArg(args, "command")
		if cmd != "" {
			if len(cmd) > 40 {
				cmd = cmd[:37] + "..."
			}
			return fmt.Sprintf("Bash(%s)", cmd)
		}
	case "ExitPlanMode":
		return "ExitPlanMode(plan)"
	}

	return call.Name
}

// uiEvent 是前端消费的简化事件结构。
type uiEvent struct {
	Cursor  int64          `json:"cursor"`
	Channel string         `json:"channel"`
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload,omitempty"`
}

func defaultPrompt() string {
	return `请帮我完成一个两阶段的代码分析任务:
1. 规划(Plan): 分析 agentsdk 仓库里 pkg/tools/builtin 目录下各个工具的职责和测试需求, 给出一个分步骤的实施计划。
2. 探索(Explore): 按计划实际阅读相关文件, 重点关注 TodoWrite / ExitPlanMode / Task / subagent_manager 等实现细节。

要求:
- 在规划阶段, 使用 TodoWrite 或 write_todos 工具创建任务列表, 至少包含 "分析 builtin 工具测试需求" 和 "分析当前工具实现状态" 两个任务。
- 在探索阶段, 使用 Read / Glob / Grep 等工具读取具体文件。
- 规划完成后, 调用 ExitPlanMode 返回一个 Markdown 格式的计划。`
}
