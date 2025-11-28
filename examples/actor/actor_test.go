package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/astercloud/aster/pkg/actor"
)

// =============================================================================
// ActorSuite - Actor 系统测试套件
// =============================================================================

type ActorSuite struct {
	suite.Suite
	system *actor.System
}

// SetupTest 每个测试前创建新的 Actor 系统
func (s *ActorSuite) SetupTest() {
	s.system = actor.NewSystem("test")
}

// TearDownTest 每个测试后关闭 Actor 系统
func (s *ActorSuite) TearDownTest() {
	if s.system != nil {
		s.system.Shutdown()
		s.system = nil
	}
}

// =============================================================================
// 测试用例
// =============================================================================

func (s *ActorSuite) TestBasicPingPong() {
	// 创建 Echo Actor
	echo := &EchoActor{name: "echo"}
	pid := s.system.Spawn(echo, "echo")

	// 发送 Ping 并等待 Pong
	resp, err := pid.Request(&PingMsg{Count: 42}, time.Second)

	require.NoError(s.T(), err, "请求不应失败")
	require.NotNil(s.T(), resp, "响应不应为空")

	pong, ok := resp.(*PongMsg)
	require.True(s.T(), ok, "响应应为 PongMsg")
	assert.Equal(s.T(), 42, pong.Count, "计数应匹配")
}

func (s *ActorSuite) TestCounterConcurrency() {
	// 使用更大的邮箱配置
	s.system.Shutdown()
	config := actor.DefaultSystemConfig()
	config.MailboxSize = 20000
	config.DefaultActorMailboxSize = 5000
	s.system = actor.NewSystemWithConfig("test", config)

	// 创建计数器 Actor（使用更大的邮箱）
	counter := &CounterActor{name: "counter", count: 0}
	props := &actor.Props{
		Name:        "counter",
		MailboxSize: 5000,
	}
	pid := s.system.SpawnWithProps(counter, props)

	// 并发发送增量消息
	var wg sync.WaitGroup
	numGoroutines := 10
	incrementsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				pid.Tell(&IncrementMsg{Value: 1})
			}
		}()
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond) // 等待消息处理完成

	// 获取最终计数
	replyCh := make(chan int, 1)
	pid.Tell(&GetCountMsg{ReplyTo: replyCh})

	select {
	case count := <-replyCh:
		expected := numGoroutines * incrementsPerGoroutine
		assert.Equal(s.T(), expected, count, "并发计数应正确")
	case <-time.After(time.Second):
		s.T().Fatal("获取计数超时")
	}
}

func (s *ActorSuite) TestSupervisorRestart() {
	// 使用静默 panic 处理器
	config := actor.DefaultSystemConfig()
	config.PanicHandler = func(a *actor.PID, msg actor.Message, err interface{}) {}
	s.system.Shutdown()
	s.system = actor.NewSystemWithConfig("test", config)

	// 创建不稳定的 Actor（会失败 2 次）
	unstable := &UnstableActor{name: "unstable", maxFails: 2}
	props := &actor.Props{
		Name:               "unstable",
		MailboxSize:        100,
		SupervisorStrategy: actor.NewOneForOneStrategy(5, time.Minute, actor.DefaultDecider),
	}
	pid := s.system.SpawnWithProps(unstable, props)

	// 第一次请求会触发 panic
	_, err := pid.Request(&PingMsg{Count: 1}, 500*time.Millisecond)
	assert.Error(s.T(), err, "第一次请求应超时（Actor panic）")

	time.Sleep(100 * time.Millisecond)

	// 第二次请求也会触发 panic
	_, err = pid.Request(&PingMsg{Count: 2}, 500*time.Millisecond)
	assert.Error(s.T(), err, "第二次请求应超时（Actor panic）")

	time.Sleep(100 * time.Millisecond)

	// 第三次请求应该成功（Actor 已恢复）
	resp, err := pid.Request(&PingMsg{Count: 3}, time.Second)
	require.NoError(s.T(), err, "第三次请求应成功")

	pong, ok := resp.(*PongMsg)
	require.True(s.T(), ok, "响应应为 PongMsg")
	assert.Equal(s.T(), 3, pong.Count, "计数应匹配")
}

func (s *ActorSuite) TestPipeline() {
	// 创建 3 阶段流水线
	stage3 := &PipelineStageActor{name: "stage3", stage: 3, nextStage: nil}
	pid3 := s.system.Spawn(stage3, "stage3")

	stage2 := &PipelineStageActor{name: "stage2", stage: 2, nextStage: pid3}
	pid2 := s.system.Spawn(stage2, "stage2")

	stage1 := &PipelineStageActor{name: "stage1", stage: 1, nextStage: pid2}
	pid1 := s.system.Spawn(stage1, "stage1")

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
		expected := "Input -> Stage1 -> Stage2 -> Stage3"
		assert.Equal(s.T(), expected, result, "流水线结果应正确")
	case <-time.After(5 * time.Second):
		s.T().Fatal("流水线处理超时")
	}
}

func (s *ActorSuite) TestBroadcast() {
	// 创建订阅者
	numSubscribers := 5
	subscribers := make([]*actor.PID, numSubscribers)
	actorInstances := make([]*SubscriberActor, numSubscribers)

	for i := 0; i < numSubscribers; i++ {
		name := fmt.Sprintf("sub-%d", i)
		sub := &SubscriberActor{name: name}
		actorInstances[i] = sub
		subscribers[i] = s.system.Spawn(sub, name)
	}

	time.Sleep(50 * time.Millisecond)

	// 广播消息
	messages := []string{"msg1", "msg2", "msg3"}
	for _, content := range messages {
		for _, pid := range subscribers {
			pid.Tell(&BroadcastMsg{Content: content})
		}
	}

	time.Sleep(100 * time.Millisecond)

	// 验证每个订阅者都收到了所有消息
	for i, sub := range actorInstances {
		sub.mu.Lock()
		count := len(sub.received)
		sub.mu.Unlock()
		assert.Equal(s.T(), len(messages), count,
			"订阅者 %d 应收到 %d 条消息", i, len(messages))
	}
}

func (s *ActorSuite) TestActorStats() {
	// 创建多个 Actor
	s.system.Spawn(&EchoActor{name: "echo1"}, "echo1")
	s.system.Spawn(&EchoActor{name: "echo2"}, "echo2")
	s.system.Spawn(&CounterActor{name: "counter"}, "counter")

	time.Sleep(50 * time.Millisecond)

	stats := s.system.Stats()
	assert.Equal(s.T(), int64(3), stats.TotalActors, "应有 3 个 Actor")
}

func (s *ActorSuite) TestActorStop() {
	echo := &EchoActor{name: "echo"}
	pid := s.system.Spawn(echo, "echo")

	// Actor 应该存在
	_, exists := s.system.GetActor("echo")
	assert.True(s.T(), exists, "Actor 应存在")

	// 停止 Actor
	s.system.Stop(pid)
	time.Sleep(100 * time.Millisecond)

	// Actor 应该不存在
	_, exists = s.system.GetActor("echo")
	assert.False(s.T(), exists, "Actor 应已停止")
}

// =============================================================================
// 基准测试
// =============================================================================

func BenchmarkActorTell(b *testing.B) {
	system := actor.NewSystem("bench")
	defer system.Shutdown()

	counter := &CounterActor{name: "counter"}
	pid := system.Spawn(counter, "counter")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pid.Tell(&IncrementMsg{Value: 1})
	}
}

func BenchmarkActorRequest(b *testing.B) {
	system := actor.NewSystem("bench")
	defer system.Shutdown()

	echo := &EchoActor{name: "echo"}
	pid := system.Spawn(echo, "echo")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pid.Request(&PingMsg{Count: i}, time.Second)
	}
}

func BenchmarkActorConcurrent(b *testing.B) {
	system := actor.NewSystem("bench")
	defer system.Shutdown()

	counter := &CounterActor{name: "counter"}
	pid := system.Spawn(counter, "counter")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pid.Tell(&IncrementMsg{Value: 1})
		}
	})
}

// =============================================================================
// 高级测试
// =============================================================================

func TestActorHighThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过高吞吐量测试（-short 模式）")
	}

	config := actor.DefaultSystemConfig()
	config.MailboxSize = 100000
	config.DefaultActorMailboxSize = 50000
	system := actor.NewSystemWithConfig("throughput", config)
	defer system.Shutdown()

	// 创建计数器 Actor
	var processed int64
	counterFunc := actor.ActorFunc(func(ctx *actor.Context, msg actor.Message) {
		if _, ok := msg.(*IncrementMsg); ok {
			atomic.AddInt64(&processed, 1)
		}
	})

	props := &actor.Props{
		Name:        "counter",
		MailboxSize: 50000,
	}
	pid := system.SpawnWithProps(counterFunc, props)

	// 发送大量消息
	numMessages := 100000
	start := time.Now()

	for i := 0; i < numMessages; i++ {
		pid.Tell(&IncrementMsg{Value: 1})
	}

	// 等待处理完成
	for atomic.LoadInt64(&processed) < int64(numMessages) {
		time.Sleep(10 * time.Millisecond)
	}

	elapsed := time.Since(start)
	throughput := float64(numMessages) / elapsed.Seconds()

	t.Logf("处理 %d 条消息耗时 %v，吞吐量: %.0f msg/s", numMessages, elapsed, throughput)
	assert.GreaterOrEqual(t, throughput, 10000.0, "吞吐量应至少 10K msg/s")
}

// =============================================================================
// 测试入口
// =============================================================================

func TestActorSuite(t *testing.T) {
	suite.Run(t, new(ActorSuite))
}
