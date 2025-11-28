package actor

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============== 测试消息类型 ==============

type PingMsg struct {
	Count int
}

func (m *PingMsg) Kind() string { return "test.ping" }

type PongMsg struct {
	Count int
}

func (m *PongMsg) Kind() string { return "test.pong" }

type CountMsg struct {
	Value int
}

func (m *CountMsg) Kind() string { return "test.count" }

type GetCountMsg struct {
	ReplyTo chan int
}

func (m *GetCountMsg) Kind() string { return "test.get_count" }

// ============== 测试 Actor ==============

// EchoActor 回声 Actor，收到消息后回复发送者
type EchoActor struct{}

func (a *EchoActor) Receive(ctx *Context, msg Message) {
	switch m := msg.(type) {
	case *PingMsg:
		ctx.Reply(&PongMsg{Count: m.Count})
	}
}

// CounterActor 计数器 Actor
type CounterActor struct {
	count int64
}

func (a *CounterActor) Receive(ctx *Context, msg Message) {
	switch m := msg.(type) {
	case *CountMsg:
		atomic.AddInt64(&a.count, int64(m.Value))
	case *GetCountMsg:
		m.ReplyTo <- int(atomic.LoadInt64(&a.count))
	}
}

// ParentActor 父 Actor，可以创建子 Actor
type ParentActor struct {
	children []*PID
}

func (a *ParentActor) Receive(ctx *Context, msg Message) {
	switch msg.(type) {
	case *Started:
		// 创建子 Actor
		child := ctx.Spawn(&EchoActor{}, "child-1")
		a.children = append(a.children, child)
	case *Terminated:
		// 子 Actor 终止
	}
}

// PanicActor 会 panic 的 Actor
type PanicActor struct {
	panicCount int
}

func (a *PanicActor) Receive(ctx *Context, msg Message) {
	switch msg.(type) {
	case *PingMsg:
		a.panicCount++
		if a.panicCount <= 2 {
			panic("intentional panic")
		}
		ctx.Reply(&PongMsg{Count: a.panicCount})
	}
}

// ============== 测试用例 ==============

func TestSystem_SpawnAndSend(t *testing.T) {
	system := NewSystem("test")
	defer system.Shutdown()

	// 创建计数器 Actor
	counter := &CounterActor{}
	pid := system.Spawn(counter, "counter")

	assert.NotNil(t, pid)
	assert.Equal(t, "counter", pid.ID)

	// 发送消息
	system.Send(pid, &CountMsg{Value: 10})
	system.Send(pid, &CountMsg{Value: 20})
	system.Send(pid, &CountMsg{Value: 30})

	// 等待处理
	time.Sleep(100 * time.Millisecond)

	// 获取计数
	replyCh := make(chan int, 1)
	system.Send(pid, &GetCountMsg{ReplyTo: replyCh})

	select {
	case count := <-replyCh:
		assert.Equal(t, 60, count)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for count")
	}
}

func TestSystem_RequestResponse(t *testing.T) {
	system := NewSystem("test")
	defer system.Shutdown()

	// 创建回声 Actor
	pid := system.Spawn(&EchoActor{}, "echo")

	// 发送请求并等待响应
	resp, err := system.Request(pid, &PingMsg{Count: 42}, time.Second)

	require.NoError(t, err)
	require.NotNil(t, resp)

	pong, ok := resp.(*PongMsg)
	require.True(t, ok)
	assert.Equal(t, 42, pong.Count)
}

func TestSystem_ParentChild(t *testing.T) {
	system := NewSystem("test")
	defer system.Shutdown()

	// 创建父 Actor
	parent := &ParentActor{}
	parentPID := system.Spawn(parent, "parent")

	// 等待子 Actor 创建
	time.Sleep(100 * time.Millisecond)

	// 检查子 Actor 是否存在
	childPID, exists := system.GetActor("child-1")
	assert.True(t, exists)
	assert.NotNil(t, childPID)

	// 停止父 Actor 应该也停止子 Actor
	system.Stop(parentPID)
	time.Sleep(100 * time.Millisecond)

	_, exists = system.GetActor("child-1")
	assert.False(t, exists)
}

func TestSystem_SupervisorRestart(t *testing.T) {
	config := DefaultSystemConfig()
	config.PanicHandler = func(actor *PID, msg Message, err interface{}) {
		// 静默处理 panic
	}
	system := NewSystemWithConfig("test", config)
	defer system.Shutdown()

	// 创建会 panic 的 Actor，使用监督策略
	panicActor := &PanicActor{}
	props := &Props{
		Name:               "panic-actor",
		MailboxSize:        10,
		SupervisorStrategy: NewOneForOneStrategy(5, time.Minute, DefaultDecider),
	}
	pid := system.SpawnWithProps(panicActor, props)

	// 发送消息，触发 panic
	system.Send(pid, &PingMsg{Count: 1})
	time.Sleep(50 * time.Millisecond)

	system.Send(pid, &PingMsg{Count: 2})
	time.Sleep(50 * time.Millisecond)

	// 第三次应该成功
	system.Send(pid, &PingMsg{Count: 3})
	time.Sleep(50 * time.Millisecond)

	// Actor 应该仍然存在（被重启）
	_, exists := system.GetActor("panic-actor")
	assert.True(t, exists)
}

func TestSystem_ConcurrentMessages(t *testing.T) {
	// 使用更大的邮箱配置
	config := DefaultSystemConfig()
	config.MailboxSize = 50000
	config.DefaultActorMailboxSize = 20000
	system := NewSystemWithConfig("test", config)
	defer system.Shutdown()

	counter := &CounterActor{}
	props := &Props{
		Name:        "counter",
		MailboxSize: 20000,
	}
	pid := system.SpawnWithProps(counter, props)

	// 并发发送消息
	var wg sync.WaitGroup
	numGoroutines := 100
	messagesPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				system.Send(pid, &CountMsg{Value: 1})
			}
		}()
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	// 获取计数
	replyCh := make(chan int, 1)
	system.Send(pid, &GetCountMsg{ReplyTo: replyCh})

	select {
	case count := <-replyCh:
		assert.Equal(t, numGoroutines*messagesPerGoroutine, count)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for count")
	}
}

func TestSystem_Stop(t *testing.T) {
	system := NewSystem("test")
	defer system.Shutdown()

	pid := system.Spawn(&EchoActor{}, "echo")

	// Actor 应该存在
	_, exists := system.GetActor("echo")
	assert.True(t, exists)

	// 停止 Actor
	system.Stop(pid)
	time.Sleep(100 * time.Millisecond)

	// Actor 应该不存在
	_, exists = system.GetActor("echo")
	assert.False(t, exists)
}

func TestSystem_Stats(t *testing.T) {
	system := NewSystem("test")
	defer system.Shutdown()

	// 创建多个 Actor
	system.Spawn(&EchoActor{}, "echo1")
	system.Spawn(&EchoActor{}, "echo2")
	system.Spawn(&CounterActor{}, "counter")

	time.Sleep(50 * time.Millisecond)

	stats := system.Stats()
	assert.Equal(t, int64(3), stats.TotalActors)
}

func TestSystem_ListActors(t *testing.T) {
	system := NewSystem("test")
	defer system.Shutdown()

	system.Spawn(&EchoActor{}, "actor1")
	system.Spawn(&EchoActor{}, "actor2")
	system.Spawn(&EchoActor{}, "actor3")

	time.Sleep(50 * time.Millisecond)

	actors := system.ListActors()
	assert.Len(t, actors, 3)
}

func TestPID_Tell(t *testing.T) {
	system := NewSystem("test")
	defer system.Shutdown()

	counter := &CounterActor{}
	pid := system.Spawn(counter, "counter")

	// 使用 PID.Tell 发送消息
	pid.Tell(&CountMsg{Value: 100})
	time.Sleep(50 * time.Millisecond)

	replyCh := make(chan int, 1)
	pid.Tell(&GetCountMsg{ReplyTo: replyCh})

	select {
	case count := <-replyCh:
		assert.Equal(t, 100, count)
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestPID_Request(t *testing.T) {
	system := NewSystem("test")
	defer system.Shutdown()

	pid := system.Spawn(&EchoActor{}, "echo")

	resp, err := pid.Request(&PingMsg{Count: 99}, time.Second)

	require.NoError(t, err)
	pong, ok := resp.(*PongMsg)
	require.True(t, ok)
	assert.Equal(t, 99, pong.Count)
}

func TestActorFunc(t *testing.T) {
	system := NewSystem("test")
	defer system.Shutdown()

	received := make(chan Message, 1)

	// 使用函数式 Actor
	actorFunc := ActorFunc(func(ctx *Context, msg Message) {
		if _, ok := msg.(*Started); ok {
			return
		}
		received <- msg
	})

	pid := system.Spawn(actorFunc, "func-actor")
	pid.Tell(&PingMsg{Count: 123})

	select {
	case msg := <-received:
		ping, ok := msg.(*PingMsg)
		require.True(t, ok)
		assert.Equal(t, 123, ping.Count)
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

// ============== 基准测试 ==============

func BenchmarkSystem_Send(b *testing.B) {
	system := NewSystem("bench")
	defer system.Shutdown()

	counter := &CounterActor{}
	pid := system.Spawn(counter, "counter")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		system.Send(pid, &CountMsg{Value: 1})
	}
}

func BenchmarkSystem_RequestResponse(b *testing.B) {
	system := NewSystem("bench")
	defer system.Shutdown()

	pid := system.Spawn(&EchoActor{}, "echo")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = system.Request(pid, &PingMsg{Count: i}, time.Second)
	}
}

func BenchmarkSystem_SpawnStop(b *testing.B) {
	system := NewSystem("bench")
	defer system.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pid := system.Spawn(&EchoActor{}, "actor")
		system.Stop(pid)
		time.Sleep(time.Microsecond) // 确保清理完成
	}
}
