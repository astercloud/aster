package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/builtin"
	"github.com/astercloud/aster/pkg/types"
)

// TestSuite 测试套件
type TestSuite struct {
	cases     []testCase
	startTime time.Time
}

type testCase struct {
	name     string
	passed   bool
	err      string
	duration time.Duration
}

func newTestSuite() *TestSuite {
	return &TestSuite{startTime: time.Now()}
}

func (ts *TestSuite) add(name string, err error, duration time.Duration) {
	tc := testCase{name: name, duration: duration, passed: err == nil}
	if err != nil {
		tc.err = err.Error()
	}
	ts.cases = append(ts.cases, tc)
}

func (ts *TestSuite) summary() bool {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📊 测试结果总结")
	fmt.Println(strings.Repeat("=", 60))

	passed, failed := 0, 0
	for _, tc := range ts.cases {
		if tc.passed {
			fmt.Printf("  ✅ PASS  %-30s  (%v)\n", tc.name, tc.duration.Round(time.Millisecond))
			passed++
		} else {
			fmt.Printf("  ❌ FAIL  %-30s  (%v)\n", tc.name, tc.duration.Round(time.Millisecond))
			fmt.Printf("         └─ %s\n", tc.err)
			failed++
		}
	}

	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("  总计: %d 通过, %d 失败, 耗时 %v\n", passed, failed, time.Since(ts.startTime).Round(time.Millisecond))
	if failed == 0 {
		fmt.Println("  🎉 所有测试通过!")
	}
	fmt.Println(strings.Repeat("=", 60))
	return failed == 0
}

// runChatTest 执行对话测试并验证
func runChatTest(ctx context.Context, ag *agent.Agent, suite *TestSuite, name, prompt string, verify func() error) {
	slog.Info("--- " + name + " ---")
	start := time.Now()

	result, err := ag.Chat(ctx, prompt)
	if err != nil {
		suite.add(name, err, time.Since(start))
		return
	}
	if result == nil || result.Status != "ok" {
		suite.add(name, fmt.Errorf("状态异常: %v", result), time.Since(start))
		return
	}

	// 执行额外验证
	if verify != nil {
		time.Sleep(300 * time.Millisecond) // 等待文件操作完成
		if err := verify(); err != nil {
			suite.add(name, err, time.Since(start))
			return
		}
	}

	suite.add(name, nil, time.Since(start))
	slog.Info("响应", "status", result.Status)
}

func main() {
	// 检查 API Key
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		slog.Error("需要设置 OPENROUTER_API_KEY 环境变量")
		os.Exit(1)
	}

	baseURL := os.Getenv("OPENROUTER_BASE_URL")
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}

	ctx := context.Background()

	// 创建依赖
	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry)

	jsonStore, err := store.NewJSONStore(".aster")
	if err != nil {
		slog.Error("创建 Store 失败", "error", err)
		os.Exit(1)
	}

	templateRegistry := agent.NewTemplateRegistry()
	templateRegistry.Register(&types.AgentTemplateDefinition{
		ID:           "simple-assistant",
		SystemPrompt: "你是一个有用的助手，可以读取和写入文件。当用户要求你读取或写入文件时，请使用可用的工具。",
		Tools:        []interface{}{"Read", "Write", "Bash"},
	})

	deps := &agent.Dependencies{
		Store:            jsonStore,
		SandboxFactory:   sandbox.NewFactory(),
		ToolRegistry:     toolRegistry,
		ProviderFactory:  &provider.OpenRouterFactory{},
		TemplateRegistry: templateRegistry,
	}

	config := &types.AgentConfig{
		TemplateID: "simple-assistant",
		ModelConfig: &types.ModelConfig{
			Provider:      "openrouter",
			Model:         "anthropic/claude-haiku-4.5",
			APIKey:        apiKey,
			BaseURL:       baseURL,
			ExecutionMode: types.ExecutionModeNonStreaming,
		},
		Sandbox: &types.SandboxConfig{
			Kind:    types.SandboxKindLocal,
			WorkDir: "./workspace",
		},
	}

	ag, err := agent.Create(ctx, config, deps)
	if err != nil {
		slog.Error("创建 Agent 失败", "error", err)
		os.Exit(1)
	}
	defer func() { _ = ag.Close() }()

	slog.Info("Agent 创建成功", "id", ag.ID())

	// 准备测试
	suite := newTestSuite()
	testFile := "./workspace/test.txt"
	_ = os.Remove(testFile)

	// 订阅事件
	eventCh := ag.Subscribe([]types.AgentChannel{types.ChannelProgress, types.ChannelMonitor}, nil)
	go func() {
		for envelope := range eventCh {
			handleEvent(envelope.Event)
		}
	}()

	// 执行测试
	runChatTest(ctx, ag, suite, "创建测试文件",
		"请创建一个名为 test.txt 的文件，内容为 'Hello World'",
		func() error {
			data, err := os.ReadFile(testFile)
			if err != nil {
				return fmt.Errorf("文件未创建: %w", err)
			}
			if strings.TrimSpace(string(data)) != "Hello World" {
				return fmt.Errorf("内容不匹配: %s", string(data))
			}
			return nil
		})

	runChatTest(ctx, ag, suite, "读取文件内容",
		"请读取 test.txt 文件的内容", nil)

	runChatTest(ctx, ag, suite, "执行 Bash 命令",
		"请执行 'ls -la' 命令", nil)

	// 验证 Agent 状态
	status := ag.Status()
	if status.State != types.AgentStateReady || status.StepCount == 0 {
		suite.add("Agent 状态检查", fmt.Errorf("状态: %s, 步骤: %d", status.State, status.StepCount), 0)
	} else {
		suite.add("Agent 状态检查", nil, 0)
	}

	// 输出结果
	fmt.Printf("\n最终状态: Agent=%s, 状态=%s, 步骤=%d\n", status.AgentID, status.State, status.StepCount)

	if !suite.summary() {
		os.Exit(1)
	}
}

func handleEvent(event interface{}) {
	switch e := event.(type) {
	case *types.ProgressToolStartEvent:
		fmt.Printf("\n[工具] %s 开始\n", e.Call.Name)
	case *types.ProgressToolEndEvent:
		fmt.Printf("[工具] %s 完成\n", e.Call.Name)
	case *types.ProgressToolErrorEvent:
		fmt.Printf("[错误] %s: %s\n", e.Call.Name, e.Error)
	case *types.ProgressDoneEvent:
		fmt.Printf("[完成] 步骤 %d\n", e.Step)
	case *types.MonitorStateChangedEvent:
		fmt.Printf("[状态] %s\n", e.State)
	case *types.MonitorErrorEvent:
		fmt.Printf("[错误] %s: %s\n", e.Phase, e.Message)
	}
}
