package events

import (
	"testing"
	"time"

	"github.com/astercloud/aster/pkg/types"
)

func TestNewEventBus(t *testing.T) {
	eb := NewEventBus()
	defer eb.Close()

	if eb.config == nil {
		t.Error("config should not be nil")
	}
	if eb.config.MaxTimelineSize != 10000 {
		t.Errorf("expected MaxTimelineSize 10000, got %d", eb.config.MaxTimelineSize)
	}
}

func TestNewEventBusWithConfig(t *testing.T) {
	config := &EventBusConfig{
		MaxTimelineSize: 100,
		MaxTimelineAge:  10 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	}
	eb := NewEventBusWithConfig(config)
	defer eb.Close()

	if eb.config.MaxTimelineSize != 100 {
		t.Errorf("expected MaxTimelineSize 100, got %d", eb.config.MaxTimelineSize)
	}
}

func TestEmitAndSubscribe(t *testing.T) {
	eb := NewEventBus()
	defer eb.Close()

	ch := eb.Subscribe([]types.AgentChannel{types.ChannelProgress}, nil)

	// Emit event
	eb.EmitProgress(&types.ProgressTextChunkEvent{Step: 1, Delta: "hello"})

	select {
	case env := <-ch:
		if env.Cursor != 1 {
			t.Errorf("expected cursor 1, got %d", env.Cursor)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for event")
	}
}

func TestCleanupBySize(t *testing.T) {
	config := &EventBusConfig{
		MaxTimelineSize: 5,
		MaxTimelineAge:  0, // disable age cleanup
		CleanupInterval: 0, // disable auto cleanup
	}
	eb := NewEventBusWithConfig(config)
	defer eb.Close()

	// Emit 10 events
	for i := range 10 {
		eb.EmitProgress(&types.ProgressTextChunkEvent{Step: i, Delta: "test"})
	}

	// Manual cleanup
	eb.cleanup()

	eb.mu.RLock()
	timelineLen := len(eb.timeline)
	bookmarksLen := len(eb.bookmarks)
	eb.mu.RUnlock()

	if timelineLen != 5 {
		t.Errorf("expected timeline length 5, got %d", timelineLen)
	}
	if bookmarksLen != 5 {
		t.Errorf("expected bookmarks length 5, got %d", bookmarksLen)
	}
}

func TestCleanupByAge(t *testing.T) {
	config := &EventBusConfig{
		MaxTimelineSize: 0, // disable size cleanup
		MaxTimelineAge:  2 * time.Second,
		CleanupInterval: 0, // disable auto cleanup
	}
	eb := NewEventBusWithConfig(config)
	defer eb.Close()

	// Emit events
	eb.EmitProgress(&types.ProgressTextChunkEvent{Step: 1, Delta: "old"})
	eb.EmitProgress(&types.ProgressTextChunkEvent{Step: 2, Delta: "old"})

	// Manually set old timestamps (Bookmark.Timestamp is in seconds)
	eb.mu.Lock()
	oldTime := time.Now().Add(-5 * time.Second).Unix()
	for i := range eb.timeline {
		eb.timeline[i].Bookmark.Timestamp = oldTime
	}
	eb.mu.Unlock()

	// Emit new event (will have current timestamp)
	eb.EmitProgress(&types.ProgressTextChunkEvent{Step: 3, Delta: "new"})

	// Manual cleanup
	eb.cleanup()

	eb.mu.RLock()
	timelineLen := len(eb.timeline)
	eb.mu.RUnlock()

	if timelineLen != 1 {
		t.Errorf("expected timeline length 1, got %d", timelineLen)
	}
}

func TestUnsubscribe(t *testing.T) {
	eb := NewEventBus()
	defer eb.Close()

	ch := eb.Subscribe([]types.AgentChannel{types.ChannelProgress}, nil)
	eb.Unsubscribe(ch)

	eb.mu.RLock()
	subsLen := len(eb.progressSubs)
	eb.mu.RUnlock()

	if subsLen != 0 {
		t.Errorf("expected 0 subscribers, got %d", subsLen)
	}
}

func TestRandomString(t *testing.T) {
	s1 := randomString(8)
	s2 := randomString(8)

	if len(s1) != 8 {
		t.Errorf("expected length 8, got %d", len(s1))
	}
	if s1 == s2 {
		t.Error("random strings should be different")
	}
}

func TestClose(t *testing.T) {
	eb := NewEventBus()
	ch := eb.Subscribe([]types.AgentChannel{types.ChannelProgress}, nil)

	eb.Close()

	// Channel should be closed
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("channel should be closed")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("channel read should not block")
	}
}

// TestGetTimelineRange 测试分页获取时间线
func TestGetTimelineRange(t *testing.T) {
	eb := NewEventBus()
	defer eb.Close()

	// 添加 10 个事件
	for i := range 10 {
		eb.EmitProgress(&types.ProgressTextChunkEvent{Step: i, Delta: "test"})
	}

	// 测试获取前 5 个
	events := eb.GetTimelineRange(0, 5)
	if len(events) != 5 {
		t.Errorf("expected 5 events, got %d", len(events))
	}

	// 测试获取后 5 个
	events = eb.GetTimelineRange(5, 5)
	if len(events) != 5 {
		t.Errorf("expected 5 events, got %d", len(events))
	}

	// 测试超出范围
	events = eb.GetTimelineRange(8, 10)
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}

	// 测试负数起始
	events = eb.GetTimelineRange(-5, 3)
	if len(events) != 3 {
		t.Errorf("expected 3 events, got %d", len(events))
	}
}

// TestGetTimelineSince 测试从指定 cursor 后获取事件
func TestGetTimelineSince(t *testing.T) {
	eb := NewEventBus()
	defer eb.Close()

	// 添加事件
	for i := range 5 {
		eb.EmitProgress(&types.ProgressTextChunkEvent{Step: i, Delta: "test"})
	}

	// 获取 cursor 3 之后的事件
	events := eb.GetTimelineSince(3)
	if len(events) != 2 {
		t.Errorf("expected 2 events after cursor 3, got %d", len(events))
	}
}

// TestGetTimelineFiltered 测试过滤获取时间线
func TestGetTimelineFiltered(t *testing.T) {
	eb := NewEventBus()
	defer eb.Close()

	// 添加不同类型的事件
	for i := range 5 {
		eb.EmitProgress(&types.ProgressTextChunkEvent{Step: i, Delta: "test"})
	}
	eb.EmitProgress(&types.ProgressDoneEvent{})

	// 过滤只获取 ProgressDoneEvent
	events := eb.GetTimelineFiltered(func(env types.AgentEventEnvelope) bool {
		_, ok := env.Event.(*types.ProgressDoneEvent)
		return ok
	})

	if len(events) != 1 {
		t.Errorf("expected 1 done event, got %d", len(events))
	}
}

// TestGetTimelineCount 测试获取事件数量
func TestGetTimelineCount(t *testing.T) {
	eb := NewEventBus()
	defer eb.Close()

	if eb.GetTimelineCount() != 0 {
		t.Error("initial count should be 0")
	}

	for i := range 10 {
		eb.EmitProgress(&types.ProgressTextChunkEvent{Step: i, Delta: "test"})
	}

	if eb.GetTimelineCount() != 10 {
		t.Errorf("expected count 10, got %d", eb.GetTimelineCount())
	}
}

// TestAutoCleanupWorker 测试自动清理 worker
func TestAutoCleanupWorker(t *testing.T) {
	config := &EventBusConfig{
		MaxTimelineSize: 5,
		MaxTimelineAge:  0,
		CleanupInterval: 200 * time.Millisecond, // 200ms 清理一次
	}
	eb := NewEventBusWithConfig(config)
	defer eb.Close()

	// 添加 10 个事件
	for i := range 10 {
		eb.EmitProgress(&types.ProgressTextChunkEvent{Step: i, Delta: "test"})
	}

	// 等待自动清理
	time.Sleep(500 * time.Millisecond)

	count := eb.GetTimelineCount()
	if count > 5 {
		t.Errorf("expected timeline count <= 5 after auto cleanup, got %d", count)
	}
}

// TestMemoryStability 测试内存稳定性（防止泄漏）
func TestMemoryStability(t *testing.T) {
	config := &EventBusConfig{
		MaxTimelineSize: 1000,
		MaxTimelineAge:  0,
		CleanupInterval: 50 * time.Millisecond,
	}
	eb := NewEventBusWithConfig(config)
	defer eb.Close()

	// 分批发送事件，给清理 worker 时间工作
	batches := 100
	eventsPerBatch := 100

	for batch := range batches {
		// 快速发送一批事件
		for i := range eventsPerBatch {
			eb.EmitProgress(&types.ProgressTextChunkEvent{
				Step:  batch*eventsPerBatch + i,
				Delta: "test data",
			})
		}

		// 每批之后给清理 worker 一些时间
		if batch%10 == 0 {
			time.Sleep(100 * time.Millisecond)
			count := eb.GetTimelineCount()
			// 应该在合理范围内
			if count > 1500 {
				t.Errorf("timeline growing too large: %d at batch %d", count, batch)
			}
		}
	}

	// 等待最后一次清理
	time.Sleep(200 * time.Millisecond)

	// 最终检查
	finalCount := eb.GetTimelineCount()
	if finalCount > 1100 {
		t.Errorf("final timeline count too large: %d (expected <= 1100)", finalCount)
	}
}

// TestUnsubscribeMultipleChannels 测试多通道取消订阅
func TestUnsubscribeMultipleChannels(t *testing.T) {
	eb := NewEventBus()
	defer eb.Close()

	// 订阅多个通道
	ch := eb.Subscribe([]types.AgentChannel{
		types.ChannelProgress,
		types.ChannelControl,
		types.ChannelMonitor,
	}, nil)

	eb.Unsubscribe(ch)

	eb.mu.RLock()
	totalSubs := len(eb.progressSubs) + len(eb.controlSubs) + len(eb.monitorSubs)
	eb.mu.RUnlock()

	if totalSubs != 0 {
		t.Errorf("expected 0 total subscribers, got %d", totalSubs)
	}
}
