// Actor 演示如何使用 Actor 模型实现多 Agent 协作，包括 Ping-Pong、
// 并发计数器、监督者策略、流水线处理和广播消息等模式。
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/astercloud/aster/pkg/actor"
)

func main() {
	cmd := &cli.Command{
		Name:  "actor-demo",
		Usage: "Actor 模型演示程序",
		Commands: []*cli.Command{
			{
				Name:   "basic",
				Usage:  "基础 Actor 演示（Ping-Pong）",
				Action: runBasicDemo,
			},
			{
				Name:   "counter",
				Usage:  "并发计数器演示",
				Action: runCounterDemo,
			},
			{
				Name:   "supervisor",
				Usage:  "监督者策略演示（故障恢复）",
				Action: runSupervisorDemo,
			},
			{
				Name:   "pipeline",
				Usage:  "流水线处理演示",
				Action: runPipelineDemo,
			},
			{
				Name:   "broadcast",
				Usage:  "广播消息演示",
				Action: runBroadcastDemo,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println("请选择一个演示命令，使用 --help 查看可用命令")
			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		slog.Error("运行失败", "error", err)
		os.Exit(1)
	}
}

// =============================================================================
// 消息类型定义
// =============================================================================

// PingMsg Ping 消息
type PingMsg struct {
	Count int
}

func (m *PingMsg) Kind() string { return "demo.ping" }

// PongMsg Pong 消息
type PongMsg struct {
	Count int
}

func (m *PongMsg) Kind() string { return "demo.pong" }

// IncrementMsg 增量消息
type IncrementMsg struct {
	Value int
}

func (m *IncrementMsg) Kind() string { return "demo.increment" }

// GetCountMsg 获取计数消息
type GetCountMsg struct {
	ReplyTo chan int
}

func (m *GetCountMsg) Kind() string { return "demo.get_count" }

// ProcessMsg 处理消息
type ProcessMsg struct {
	Data   string
	Stage  int
	Result chan string
}

func (m *ProcessMsg) Kind() string { return "demo.process" }

// BroadcastMsg 广播消息
type BroadcastMsg struct {
	Content string
}

func (m *BroadcastMsg) Kind() string { return "demo.broadcast" }

// =============================================================================
// Actor 实现
// =============================================================================

// EchoActor 回声 Actor - 收到 Ping 回复 Pong
type EchoActor struct {
	name string
}

func (a *EchoActor) Receive(ctx *actor.Context, msg actor.Message) {
	switch m := msg.(type) {
	case *actor.Started:
		fmt.Printf("  [%s] 启动完成\n", a.name)
	case *PingMsg:
		fmt.Printf("  [%s] 收到 Ping(%d)，回复 Pong\n", a.name, m.Count)
		ctx.Reply(&PongMsg{Count: m.Count})
	case *actor.Stopping:
		fmt.Printf("  [%s] 正在停止...\n", a.name)
	}
}

// CounterActor 计数器 Actor - 线程安全的计数器
type CounterActor struct {
	name  string
	count int
}

func (a *CounterActor) Receive(ctx *actor.Context, msg actor.Message) {
	switch m := msg.(type) {
	case *actor.Started:
		fmt.Printf("  [%s] 计数器启动，初始值: %d\n", a.name, a.count)
	case *IncrementMsg:
		a.count += m.Value
		fmt.Printf("  [%s] 增加 %d，当前值: %d\n", a.name, m.Value, a.count)
	case *GetCountMsg:
		m.ReplyTo <- a.count
	}
}

// UnstableActor 不稳定的 Actor - 模拟故障
type UnstableActor struct {
	name      string
	failCount int
	maxFails  int
	recovered bool
}

func (a *UnstableActor) Receive(ctx *actor.Context, msg actor.Message) {
	switch m := msg.(type) {
	case *actor.Started:
		fmt.Printf("  [%s] 启动\n", a.name)
	case *actor.Restarting:
		a.recovered = true
		fmt.Printf("  [%s] 正在重启（已恢复）\n", a.name)
	case *PingMsg:
		a.failCount++
		if a.failCount <= a.maxFails {
			fmt.Printf("  [%s] 第 %d 次故障，触发 panic！\n", a.name, a.failCount)
			panic(fmt.Sprintf("模拟故障 #%d", a.failCount))
		}
		fmt.Printf("  [%s] 已稳定，正常处理 Ping(%d)\n", a.name, m.Count)
		ctx.Reply(&PongMsg{Count: m.Count})
	}
}

// PipelineStageActor 流水线阶段 Actor
type PipelineStageActor struct {
	name      string
	stage     int
	nextStage *actor.PID
}

func (a *PipelineStageActor) Receive(ctx *actor.Context, msg actor.Message) {
	switch m := msg.(type) {
	case *actor.Started:
		fmt.Printf("  [%s] 流水线阶段 %d 就绪\n", a.name, a.stage)
	case *ProcessMsg:
		// 处理数据
		processed := fmt.Sprintf("%s -> Stage%d", m.Data, a.stage)
		fmt.Printf("  [%s] 处理: %s\n", a.name, processed)

		if a.nextStage != nil {
			// 转发到下一阶段
			a.nextStage.Tell(&ProcessMsg{
				Data:   processed,
				Stage:  a.stage + 1,
				Result: m.Result,
			})
		} else {
			// 最后阶段，返回结果
			m.Result <- processed
		}
	}
}

// SubscriberActor 订阅者 Actor - 接收广播消息
type SubscriberActor struct {
	name     string
	received []string
	mu       sync.Mutex
}

func (a *SubscriberActor) Receive(ctx *actor.Context, msg actor.Message) {
	switch m := msg.(type) {
	case *actor.Started:
		fmt.Printf("  [%s] 订阅者就绪\n", a.name)
	case *BroadcastMsg:
		a.mu.Lock()
		a.received = append(a.received, m.Content)
		a.mu.Unlock()
		fmt.Printf("  [%s] 收到广播: %s\n", a.name, m.Content)
	}
}

// =============================================================================
// 演示命令
// =============================================================================

// runBasicDemo 基础 Ping-Pong 演示
func runBasicDemo(ctx context.Context, cmd *cli.Command) error {
	fmt.Println("\n🎯 基础 Actor 演示（Ping-Pong）")
	fmt.Println(strings.Repeat("=", 50))

	// 创建 Actor 系统
	system := actor.NewSystem("basic-demo")
	defer system.Shutdown()

	// 创建 Echo Actor
	echo := &EchoActor{name: "Echo"}
	pid := system.Spawn(echo, "echo")

	fmt.Println("\n📤 发送 3 个 Ping 消息...")

	// 发送多个 Ping 并等待 Pong
	for i := 1; i <= 3; i++ {
		resp, err := pid.Request(&PingMsg{Count: i}, 5*time.Second)
		if err != nil {
			return fmt.Errorf("请求失败: %w", err)
		}

		if pong, ok := resp.(*PongMsg); ok {
			fmt.Printf("📥 收到 Pong(%d)\n", pong.Count)
		}
	}

	fmt.Println("\n✅ 基础演示完成!")
	return nil
}

// runCounterDemo 并发计数器演示
func runCounterDemo(ctx context.Context, cmd *cli.Command) error {
	fmt.Println("\n🔢 并发计数器演示")
	fmt.Println(strings.Repeat("=", 50))

	system := actor.NewSystem("counter-demo")
	defer system.Shutdown()

	// 创建计数器 Actor
	counter := &CounterActor{name: "Counter", count: 0}
	pid := system.Spawn(counter, "counter")

	fmt.Println("\n📤 启动 10 个 goroutine 并发增加计数...")

	// 并发发送增量消息
	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for range 10 {
				pid.Tell(&IncrementMsg{Value: 1})
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond) // 等待消息处理完成

	// 获取最终计数
	replyCh := make(chan int, 1)
	pid.Tell(&GetCountMsg{ReplyTo: replyCh})

	select {
	case count := <-replyCh:
		fmt.Printf("\n📊 最终计数: %d (预期: 100)\n", count)
		if count == 100 {
			fmt.Println("✅ 并发安全验证通过!")
		} else {
			fmt.Println("❌ 计数不正确!")
		}
	case <-time.After(time.Second):
		fmt.Println("❌ 获取计数超时")
	}

	return nil
}

// runSupervisorDemo 监督者策略演示
func runSupervisorDemo(ctx context.Context, cmd *cli.Command) error {
	fmt.Println("\n🛡️ 监督者策略演示（故障恢复）")
	fmt.Println(strings.Repeat("=", 50))

	// 使用自定义配置，静默 panic 日志
	config := actor.DefaultSystemConfig()
	config.PanicHandler = func(a *actor.PID, msg actor.Message, err any) {
		// 静默处理，不打印堆栈
	}
	system := actor.NewSystemWithConfig("supervisor-demo", config)
	defer system.Shutdown()

	// 创建不稳定的 Actor（会失败 2 次）
	unstable := &UnstableActor{name: "Unstable", maxFails: 2}

	// 使用 OneForOne 监督策略（允许 5 次重启）
	props := &actor.Props{
		Name:               "unstable",
		MailboxSize:        100,
		SupervisorStrategy: actor.NewOneForOneStrategy(5, time.Minute, actor.DefaultDecider),
	}
	pid := system.SpawnWithProps(unstable, props)

	fmt.Println("\n📤 发送消息，触发故障和自动恢复...")

	// 发送消息触发故障
	for i := 1; i <= 4; i++ {
		fmt.Printf("\n--- 第 %d 次尝试 ---\n", i)
		resp, err := pid.Request(&PingMsg{Count: i}, 2*time.Second)
		if err != nil {
			fmt.Printf("⏳ 请求超时（Actor 可能正在重启）\n")
			time.Sleep(200 * time.Millisecond)
			continue
		}

		if pong, ok := resp.(*PongMsg); ok {
			fmt.Printf("📥 成功收到 Pong(%d) - Actor 已恢复!\n", pong.Count)
		}
	}

	fmt.Println("\n✅ 监督者策略演示完成!")
	fmt.Println("   Actor 在 2 次故障后自动恢复并正常工作")
	return nil
}

// runPipelineDemo 流水线处理演示
func runPipelineDemo(ctx context.Context, cmd *cli.Command) error {
	fmt.Println("\n🔗 流水线处理演示")
	fmt.Println(strings.Repeat("=", 50))

	system := actor.NewSystem("pipeline-demo")
	defer system.Shutdown()

	// 创建 3 阶段流水线
	stage3 := &PipelineStageActor{name: "Stage3", stage: 3, nextStage: nil}
	pid3 := system.Spawn(stage3, "stage3")

	stage2 := &PipelineStageActor{name: "Stage2", stage: 2, nextStage: pid3}
	pid2 := system.Spawn(stage2, "stage2")

	stage1 := &PipelineStageActor{name: "Stage1", stage: 1, nextStage: pid2}
	pid1 := system.Spawn(stage1, "stage1")

	fmt.Println("\n📤 发送数据进入流水线...")

	// 发送数据
	resultCh := make(chan string, 1)
	pid1.Tell(&ProcessMsg{
		Data:   "Input",
		Stage:  1,
		Result: resultCh,
	})

	// 等待结果
	select {
	case result := <-resultCh:
		fmt.Printf("\n📥 最终结果: %s\n", result)
		fmt.Println("✅ 流水线处理完成!")
	case <-time.After(5 * time.Second):
		fmt.Println("❌ 流水线处理超时")
	}

	return nil
}

// runBroadcastDemo 广播消息演示
func runBroadcastDemo(ctx context.Context, cmd *cli.Command) error {
	fmt.Println("\n📢 广播消息演示")
	fmt.Println(strings.Repeat("=", 50))

	system := actor.NewSystem("broadcast-demo")
	defer system.Shutdown()

	// 创建多个订阅者
	subscribers := make([]*actor.PID, 5)
	for i := range 5 {
		sub := &SubscriberActor{name: fmt.Sprintf("Sub-%d", i+1)}
		subscribers[i] = system.Spawn(sub, fmt.Sprintf("subscriber-%d", i+1))
	}

	time.Sleep(50 * time.Millisecond) // 等待所有订阅者就绪

	fmt.Println("\n📤 广播 3 条消息...")

	// 广播消息
	messages := []string{"Hello", "World", "From Actor!"}
	for _, content := range messages {
		fmt.Printf("\n--- 广播: %s ---\n", content)
		for _, pid := range subscribers {
			pid.Tell(&BroadcastMsg{Content: content})
		}
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("\n✅ 广播演示完成!")
	fmt.Printf("   共 %d 个订阅者，每个收到 %d 条消息\n", len(subscribers), len(messages))
	return nil
}
