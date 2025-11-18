package telemetry

import (
	"context"
	"fmt"
	"time"
)

// Example 演示如何在 Agent 中集成 Telemetry
func Example() {
	// 1. 初始化 Tracer 和 Metrics
	tracer := NewSimpleTracer()
	metrics := NewSimpleMetrics()
	agentMetrics := NewAgentMetrics(metrics)

	// 设置为全局实例
	SetGlobalTracer(tracer)
	SetGlobalMetrics(metrics)

	// 2. 模拟 Agent 执行
	ctx := context.Background()
	agentID := "agent-001"

	// 开始追踪
	ctx, span := tracer.StartSpan(ctx, "agent.chat",
		WithSpanKind(SpanKindServer),
		WithAttributes(
			String("agent.id", agentID),
			String("user.id", "user-123"),
		),
	)
	defer span.End()

	startTime := time.Now()

	// 3. 执行业务逻辑
	span.AddEvent("processing_message", String("message", "Hello"))

	// 模拟模型调用
	_, modelSpan := tracer.StartSpan(ctx, "model.generate")
	modelSpan.SetAttributes(
		String("model", "claude-3-5-sonnet"),
		Int("input_tokens", 100),
	)
	time.Sleep(100 * time.Millisecond) // 模拟延迟
	modelSpan.End()

	// 记录 token 使用
	agentMetrics.RecordTokens(agentID, 100, 200)

	// 模拟工具调用
	_, toolSpan := tracer.StartSpan(ctx, "tool.execute")
	toolSpan.SetAttributes(String("tool.name", "search"))
	time.Sleep(50 * time.Millisecond)
	toolSpan.End()

	// 记录工具调用
	agentMetrics.RecordToolCall(agentID, "search", 50*time.Millisecond, true)

	// 4. 记录请求完成
	duration := time.Since(startTime)
	agentMetrics.RecordRequest(agentID, duration)
	span.SetStatus(StatusCodeOK, "completed")

	// 5. 输出指标快照
	snapshot := metrics.Snapshot()
	fmt.Printf("\n📊 Metrics Snapshot:\n")
	fmt.Printf("Timestamp: %s\n\n", snapshot.Timestamp.Format("15:04:05"))

	fmt.Println("Counters:")
	for name, counter := range snapshot.Counters {
		fmt.Printf("  %s = %d\n", name, counter.Value)
	}

	fmt.Println("\nHistograms:")
	for name, hist := range snapshot.Histograms {
		fmt.Printf("  %s: count=%d, mean=%.3fs, min=%.3fs, max=%.3fs\n",
			name, hist.Count, hist.Mean, hist.Min, hist.Max)
	}

	// 6. 输出追踪信息
	fmt.Printf("\n🔍 Trace Information:\n")
	for i, s := range tracer.GetSpans() {
		fmt.Printf("%d. %s (%.2fms)\n", i+1, s.Name(), s.Duration().Seconds()*1000)
		if len(s.Attributes()) > 0 {
			fmt.Printf("   Attributes: ")
			for _, attr := range s.Attributes() {
				fmt.Printf("%s=%v ", attr.Key, attr.Value)
			}
			fmt.Println()
		}
	}
}

// ExampleWithError 演示错误处理和追踪
func ExampleWithError() {
	tracer := NewSimpleTracer()
	metrics := NewSimpleMetrics()
	agentMetrics := NewAgentMetrics(metrics)

	ctx := context.Background()
	agentID := "agent-002"

	_, span := tracer.StartSpan(ctx, "agent.chat")
	defer span.End()

	// 模拟错误
	err := fmt.Errorf("model API timeout")
	span.RecordError(err)
	span.SetStatus(StatusCodeError, err.Error())

	// 记录错误指标
	agentMetrics.RecordError(agentID, "timeout")

	fmt.Printf("❌ Error recorded: %v\n", err)
}

// ExampleConcurrentAgents 演示多 Agent 并发场景
func ExampleConcurrentAgents() {
	metrics := NewSimpleMetrics()
	agentMetrics := NewAgentMetrics(metrics)

	// 模拟 3 个并发 Agent
	for i := 1; i <= 3; i++ {
		agentID := fmt.Sprintf("agent-%03d", i)
		go func(id string) {
			duration := time.Duration(50+i*10) * time.Millisecond
			time.Sleep(duration)
			agentMetrics.RecordRequest(id, duration)
			agentMetrics.RecordTokens(id, 100, 200)
		}(agentID)
	}

	time.Sleep(200 * time.Millisecond)

	// 设置活跃 Agent 数量
	agentMetrics.SetActiveAgents(3)

	snapshot := metrics.Snapshot()
	fmt.Printf("\n📊 Concurrent Agents Metrics:\n")
	fmt.Printf("Total requests: %d\n", len(snapshot.Counters))
	fmt.Printf("Active agents: %.0f\n", snapshot.Gauges["agent.active.count"].Value)
}
