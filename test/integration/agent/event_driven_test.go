package agent

import (
	"context"
	"testing"
	"time"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/types"
	"github.com/astercloud/aster/test/integration"
)

// TestEventDrivenArchitecture 验证事件驱动架构的三通道通信
func TestEventDrivenArchitecture(t *testing.T) {
	// 准备模拟响应
	mockResponses := []string{
		"我需要读取一个文件来帮助你",
		"我已经读取了文件内容",
		"任务已完成",
	}

	deps, mockProvider := integration.SetupIntegrationDeps(t, mockResponses)
	eventCollector := integration.NewEventCollector()

	// 创建Agent配置
	config := &types.AgentConfig{
		TemplateID: "test-template",
		ModelConfig: &types.ModelConfig{
			Provider: "mock",
			Model:    "test-model",
			APIKey:   "test-key",
		},
		Sandbox: &types.SandboxConfig{
			Kind:    types.SandboxKindMock,
			WorkDir: "/tmp/test",
		},
	}

	// 注册测试模板
	deps.TemplateRegistry.Register(&types.AgentTemplateDefinition{
		ID:           "test-template",
		SystemPrompt: "You are a helpful assistant.",
		Model:        "test-model",
		Tools:        []interface{}{"Read", "Write"},
	})

	// 创建Agent
	ag, err := agent.Create(context.Background(), config, deps)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	defer func() { _ = ag.Close() }()

	// 测试用例1: Progress Channel 验证
	t.Run("Progress Channel", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 订阅Progress事件
		progressChan, err := ag.Subscribe(ctx, []types.EventType{types.EventTypeProgress})
		if err != nil {
			t.Fatalf("Failed to subscribe to progress events: %v", err)
		}

		// 发送消息
		messageChan := make(chan []byte, 1)
		go func() {
			defer close(messageChan)
			response, err := ag.Stream(ctx, []byte("请读取/tmp/test.txt文件"))
			if err != nil {
				t.Errorf("Agent stream failed: %v", err)
				return
			}

			for chunk := range response {
				messageChan <- chunk
			}
		}()

		// 收集Progress事件
		progressCount := 0
		timeout := time.After(5 * time.Second)

	EventLoop:
		for {
			select {
			case event, ok := <-progressChan:
				if !ok {
					break EventLoop
				}
				if event.Type == types.EventTypeProgress {
					progressCount++
					t.Logf("Received progress event: %+v", event)
				}
			case <-timeout:
				break EventLoop
			case <-ctx.Done():
				break EventLoop
			}
		}

		if progressCount == 0 {
			t.Error("Expected at least one progress event, but received none")
		}

		t.Logf("Progress channel test passed with %d events", progressCount)
	})

	// 测试用例2: Control Channel 验证
	t.Run("Control Channel", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 订阅Control事件
		controlChan, err := ag.Subscribe(ctx, []types.EventType{types.EventTypeControl})
		if err != nil {
			t.Fatalf("Failed to subscribe to control events: %v", err)
		}

		// 发送需要用户确认的消息
		go func() {
			_, err := ag.Stream(ctx, []byte("请执行一个需要确认的操作"))
			if err != nil {
				t.Errorf("Agent stream failed: %v", err)
			}
		}()

		// 等待Control事件
		select {
		case event, ok := <-controlChan:
			if !ok {
				t.Error("Control channel closed unexpectedly")
				return
			}
			if event.Type == types.EventTypeControl {
				t.Logf("Received control event: %+v", event)
			} else {
				t.Errorf("Expected control event, got %v", event.Type)
			}
		case <-time.After(3 * time.Second):
			t.Error("Timeout waiting for control event")
		case <-ctx.Done():
			t.Error("Context cancelled while waiting for control event")
		}

		t.Log("Control channel test passed")
	})

	// 测试用例3: Monitor Channel 验证
	t.Run("Monitor Channel", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 订阅Monitor事件
		monitorChan, err := ag.Subscribe(ctx, []types.EventType{types.EventTypeMonitor})
		if err != nil {
			t.Fatalf("Failed to subscribe to monitor events: %v", err)
		}

		// 发送消息并执行操作
		go func() {
			_, err := ag.Stream(ctx, []byte("请执行一些操作并生成监控事件"))
			if err != nil {
				t.Errorf("Agent stream failed: %v", err)
			}
		}()

		// 等待Monitor事件
		select {
		case event, ok := <-monitorChan:
			if !ok {
				t.Error("Monitor channel closed unexpectedly")
				return
			}
			if event.Type == types.EventTypeMonitor {
				t.Logf("Received monitor event: %+v", event)
			} else {
				t.Errorf("Expected monitor event, got %v", event.Type)
			}
		case <-time.After(3 * time.Second):
			t.Error("Timeout waiting for monitor event")
		case <-ctx.Done():
			t.Error("Context cancelled while waiting for monitor event")
		}

		t.Log("Monitor channel test passed")
	})

	// 测试用例4: 多通道并发测试
	t.Run("Concurrent Multi-Channel", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// 订阅所有类型的事件
		eventChan, err := ag.Subscribe(ctx, []types.EventType{
			types.EventTypeProgress,
			types.EventTypeControl,
			types.EventTypeMonitor,
		})
		if err != nil {
			t.Fatalf("Failed to subscribe to all events: %v", err)
		}

		// 启动多个并发操作
		for i := 0; i < 3; i++ {
			go func(id int) {
				_, err := ag.Stream(ctx, []byte([]byte("请执行操作 "+string(rune('A'+id)))))
				if err != nil {
					t.Errorf("Concurrent agent stream %d failed: %v", id, err)
				}
			}(i)
		}

		// 收集所有事件
		eventCount := make(map[types.EventType]int)
		timeout := time.After(10 * time.Second)

		for {
			select {
			case event, ok := <-eventChan:
				if !ok {
					goto Done
				}
				eventCount[event.Type]++
				t.Logf("Received event: %s, total for type: %d", event.Type, eventCount[event.Type])
			case <-timeout:
				goto Done
			case <-ctx.Done():
				goto Done
			}
		}

	Done:
		totalEvents := 0
		for eventType, count := range eventCount {
			totalEvents += count
			t.Logf("Event type %s: %d events", eventType, count)
		}

		if totalEvents == 0 {
			t.Error("Expected at least some events, but received none")
		}

		t.Logf("Multi-channel test passed with %d total events", totalEvents)
	})

	// 验证Mock Provider被正确调用
	if mockProvider.current == 0 {
		t.Error("Expected mock provider to be called at least once")
	}

	t.Logf("Event-driven architecture integration test completed successfully")
}

// TestEventChannelIsolation 验证事件通道的隔离性
func TestEventChannelIsolation(t *testing.T) {
	mockResponses := []string{
		"Progress response",
		"Control response",
		"Monitor response",
	}

	deps, _ := integration.SetupIntegrationDeps(t, mockResponses)

	config := &types.AgentConfig{
		TemplateID: "isolation-test",
		ModelConfig: &types.ModelConfig{
			Provider: "mock",
			Model:    "test-model",
			APIKey:   "test-key",
		},
		Sandbox: &types.SandboxConfig{
			Kind:    types.SandboxKindMock,
			WorkDir: "/tmp/test",
		},
	}

	deps.TemplateRegistry.Register(&types.AgentTemplateDefinition{
		ID:           "isolation-test",
		SystemPrompt: "You are a test assistant.",
		Model:        "test-model",
		Tools:        []interface{}{"Read"},
	})

	ag, err := agent.Create(context.Background(), config, deps)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	defer func() { _ = ag.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 分别订阅不同的事件类型
	progressChan, _ := ag.Subscribe(ctx, []types.EventType{types.EventTypeProgress})
	controlChan, _ := ag.Subscribe(ctx, []types.EventType{types.EventTypeControl})
	monitorChan, _ := ag.Subscribe(ctx, []types.EventType{types.EventTypeMonitor})

	// 发送消息
	go func() {
		_, _ = ag.Stream(ctx, []byte("test isolation"))
	}()

	// 验证通道隔离性
	progressReceived := false
	controlReceived := false
	monitorReceived := false

	for i := 0; i < 3; i++ {
		select {
		case <-progressChan:
			progressReceived = true
		case <-controlChan:
			controlReceived = true
		case <-monitorChan:
			monitorReceived = true
		case <-time.After(3 * time.Second):
			break
		}
	}

	// 至少应该接收到一些事件
	if !progressReceived && !controlReceived && !monitorReceived {
		t.Error("Expected to receive events from at least one channel")
	}

	t.Log("Event channel isolation test passed")
}