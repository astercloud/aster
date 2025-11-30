// Telemetry 演示 OpenTelemetry 集成，包括基础追踪、Agent 执行追踪、
// LLM 调用追踪、工具执行追踪和分布式追踪上下文传播。
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/astercloud/aster/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	// ====== 示例 1: 基础 OTel 集成 ======
	fmt.Println("=== Example 1: Basic OpenTelemetry Integration ===")
	basicOTelExample()

	// ====== 示例 2: Agent 追踪 ======
	fmt.Println("\n=== Example 2: Agent Tracing ===")
	agentTracingExample()

	// ====== 示例 3: LLM 调用追踪 ======
	fmt.Println("\n=== Example 3: LLM Call Tracing ===")
	llmTracingExample()

	// ====== 示例 4: 工具执行追踪 ======
	fmt.Println("\n=== Example 4: Tool Execution Tracing ===")
	toolTracingExample()

	// ====== 示例 5: 分布式追踪 ======
	fmt.Println("\n=== Example 5: Distributed Tracing ===")
	distributedTracingExample()

	fmt.Println("\nAll examples completed successfully!")
}

// basicOTelExample 演示基础 OpenTelemetry 集成
func basicOTelExample() {
	ctx := context.Background()

	// 1. 创建 stdout exporter（用于演示）
	exporter, err := stdouttrace.New(
		stdouttrace.WithWriter(os.Stdout),
		stdouttrace.WithPrettyPrint(),
	)
	if err != nil {
		log.Fatalf("Failed to create exporter: %v", err)
	}

	// 2. 创建 OTel tracer
	tracer, err := telemetry.NewOTelTracer(
		"my-agent-service",
		telemetry.WithOTelExporter(exporter),
		telemetry.WithOTelServiceVersion("1.0.0"),
		telemetry.WithOTelSampler(sdktrace.AlwaysSample()),
		telemetry.WithOTelAttributes(
			attribute.String("environment", "development"),
			attribute.String("deployment", "local"),
		),
	)
	if err != nil {
		log.Fatalf("Failed to create tracer: %v", err)
	}
	defer func() { _ = tracer.Shutdown(ctx) }()

	// 3. 设置为全局 tracer
	telemetry.SetGlobalTracer(tracer)

	// 4. 创建 span
	_, span := tracer.StartSpan(
		ctx,
		"basic-operation",
		telemetry.WithSpanKind(telemetry.SpanKindInternal),
	)
	defer span.End()

	// 5. 添加属性
	span.SetAttributes(
		telemetry.String("operation.type", "example"),
		telemetry.Int("operation.count", 1),
	)

	// 6. 模拟一些工作
	time.Sleep(100 * time.Millisecond)

	// 7. 添加事件
	span.AddEvent("operation.completed",
		telemetry.String("result", "success"),
	)

	// 8. 设置状态
	span.SetStatus(telemetry.StatusCodeOK, "Operation completed successfully")

	fmt.Println("✅ Basic OTel integration completed")
}

// agentTracingExample 演示 Agent 执行追踪
func agentTracingExample() {
	ctx := context.Background()
	tracer := telemetry.GetGlobalTracer()

	// 模拟 Agent.Chat() 调用
	ctx, span := tracer.StartSpan(
		ctx,
		"agent.chat",
		telemetry.WithSpanKind(telemetry.SpanKindInternal),
		telemetry.WithAttributes(
			telemetry.String(telemetry.AttrAgentID, "agt_123456"),
			telemetry.String(telemetry.AttrAgentName, "customer-support"),
			telemetry.String(telemetry.AttrSessionID, "sess_789"),
			telemetry.String(telemetry.AttrUserID, "user_001"),
		),
	)
	defer span.End()

	// 模拟消息处理
	span.AddEvent("message.received",
		telemetry.String("message.content", "Hello, I need help"),
	)

	time.Sleep(50 * time.Millisecond)

	// 模拟中间件处理
	processMiddleware(ctx, span)

	// 模拟 LLM 调用
	processLLM(ctx, span)

	span.SetStatus(telemetry.StatusCodeOK, "Chat completed")
	fmt.Println("✅ Agent tracing completed")
}

// llmTracingExample 演示 LLM 调用追踪
func llmTracingExample() {
	ctx := context.Background()
	tracer := telemetry.GetGlobalTracer()

	// 创建父 span
	ctx, parentSpan := tracer.StartSpan(ctx, "agent.process")
	defer parentSpan.End()

	// 创建 LLM 调用的子 span
	_, span := tracer.StartSpan(
		ctx,
		"llm.request",
		telemetry.WithSpanKind(telemetry.SpanKindClient),
		telemetry.WithAttributes(
			telemetry.String(telemetry.AttrLLMProvider, "anthropic"),
			telemetry.String(telemetry.AttrLLMModel, "claude-3-5-sonnet-20241022"),
			telemetry.Int(telemetry.AttrLLMMaxTokens, 4096),
		),
	)
	defer span.End()

	// 模拟 API 调用
	time.Sleep(200 * time.Millisecond)

	// 记录 token 使用情况
	span.SetAttributes(
		telemetry.Int(telemetry.AttrLLMInputTokens, 150),
		telemetry.Int(telemetry.AttrLLMOutputTokens, 80),
		telemetry.Int(telemetry.AttrLLMTotalTokens, 230),
	)

	span.AddEvent("llm.response.received")
	span.SetStatus(telemetry.StatusCodeOK, "LLM request completed")

	fmt.Println("✅ LLM tracing completed")
}

// toolTracingExample 演示工具执行追踪
func toolTracingExample() {
	ctx := context.Background()
	tracer := telemetry.GetGlobalTracer()

	// 创建父 span
	ctx, parentSpan := tracer.StartSpan(ctx, "agent.execute-tools")
	defer parentSpan.End()

	// 模拟执行多个工具
	tools := []string{"Bash", "Read", "HttpRequest"}

	for _, toolName := range tools {
		_, span := tracer.StartSpan(
			ctx,
			"tool.execute",
			telemetry.WithSpanKind(telemetry.SpanKindInternal),
			telemetry.WithAttributes(
				telemetry.String(telemetry.AttrToolName, toolName),
				telemetry.String(telemetry.AttrToolCallID, fmt.Sprintf("call_%s", toolName)),
			),
		)

		// 模拟工具执行
		startTime := time.Now()
		time.Sleep(50 * time.Millisecond)
		duration := time.Since(startTime)

		// 记录执行结果
		span.SetAttributes(
			telemetry.Int64(telemetry.AttrToolDuration, duration.Milliseconds()),
			telemetry.Bool(telemetry.AttrToolSuccess, true),
		)

		span.AddEvent("tool.completed",
			telemetry.String("tool.result", "success"),
		)

		span.SetStatus(telemetry.StatusCodeOK, "Tool executed successfully")
		span.End()
	}

	parentSpan.SetStatus(telemetry.StatusCodeOK, "All tools executed")
	fmt.Println("✅ Tool tracing completed")
}

// distributedTracingExample 演示分布式追踪
func distributedTracingExample() {
	ctx := context.Background()
	tracer := telemetry.GetGlobalTracer()

	// Service A: 创建一个 span
	ctx, spanA := tracer.StartSpan(
		ctx,
		"service-a.process",
		telemetry.WithSpanKind(telemetry.SpanKindServer),
	)

	// 模拟将 trace context 传播到 Service B
	carrier := make(MapCarrier)
	if err := tracer.Inject(ctx, carrier); err != nil {
		log.Printf("Failed to inject context: %v", err)
	}

	spanA.AddEvent("context.injected",
		telemetry.String("carrier.type", "http-headers"),
	)

	// Service B: 提取 trace context
	ctxB := context.Background()
	ctxB = tracer.Extract(ctxB, carrier)

	_, spanB := tracer.StartSpan(
		ctxB,
		"service-b.handle",
		telemetry.WithSpanKind(telemetry.SpanKindServer),
	)

	spanB.AddEvent("context.extracted")
	time.Sleep(50 * time.Millisecond)

	spanB.SetStatus(telemetry.StatusCodeOK, "Service B completed")
	spanB.End()

	spanA.SetStatus(telemetry.StatusCodeOK, "Service A completed")
	spanA.End()

	fmt.Println("✅ Distributed tracing completed")
}

// 辅助函数

func processMiddleware(ctx context.Context, _ telemetry.Span) {
	tracer := telemetry.GetGlobalTracer()

	middlewares := []string{"auth", "rate-limit", "logging"}
	for _, mw := range middlewares {
		_, span := tracer.StartSpan(
			ctx,
			fmt.Sprintf("middleware.%s", mw),
			telemetry.WithSpanKind(telemetry.SpanKindInternal),
		)

		time.Sleep(10 * time.Millisecond)
		span.SetStatus(telemetry.StatusCodeOK, fmt.Sprintf("%s passed", mw))
		span.End()
	}
}

func processLLM(ctx context.Context, _ telemetry.Span) {
	tracer := telemetry.GetGlobalTracer()

	_, span := tracer.StartSpan(
		ctx,
		"llm.generate",
		telemetry.WithSpanKind(telemetry.SpanKindClient),
	)
	defer span.End()

	time.Sleep(100 * time.Millisecond)
	span.SetStatus(telemetry.StatusCodeOK, "LLM generation completed")
}

// 与其他 Exporters 集成的示例

// 使用 Jaeger Exporter
var _ = func() {
	/*
		import "go.opentelemetry.io/otel/exporters/jaeger"

		exporter, err := jaeger.New(
			jaeger.WithCollectorEndpoint(
				jaeger.WithEndpoint("http://localhost:14268/api/traces"),
			),
		)
		if err != nil {
			log.Fatalf("Failed to create Jaeger exporter: %v", err)
		}

		tracer, err := telemetry.NewOTelTracer(
			"my-service",
			telemetry.WithOTelExporter(exporter),
		)
	*/
}

// 使用 OTLP Exporter (Grafana Tempo, Honeycomb, etc.)
var _ = func() {
	/*
		import "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"

		exporter, err := otlptracehttp.New(
			context.Background(),
			otlptracehttp.WithEndpoint("localhost:4318"),
			otlptracehttp.WithInsecure(),
		)
		if err != nil {
			log.Fatalf("Failed to create OTLP exporter: %v", err)
		}

		tracer, err := telemetry.NewOTelTracer(
			"my-service",
			telemetry.WithOTelExporter(exporter),
		)
	*/
}

// 使用 Google Cloud Trace
var _ = func() {
	/*
		import texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"

		exporter, err := texporter.New(
			texporter.WithProjectID("my-gcp-project"),
		)
		if err != nil {
			log.Fatalf("Failed to create GCP Trace exporter: %v", err)
		}

		tracer, err := telemetry.NewOTelTracer(
			"my-service",
			telemetry.WithOTelExporter(exporter),
		)
	*/
}

// propagation.MapCarrier 实现（简化版）
var _ propagation // 保留类型定义供参考

type propagation struct{}
type MapCarrier map[string]string

func (m MapCarrier) Get(key string) string { return m[key] }
func (m MapCarrier) Set(key, value string) { m[key] = value }
func (m MapCarrier) Keys() []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
