package telemetry

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// OTelTracer 实现 Tracer 接口，适配 OpenTelemetry
// 参考 Google ADK-Go 的 telemetry/telemetry.go 实现
//
// 使用示例:
//
//	// 1. 创建 OTel tracer
//	tracer, err := telemetry.NewOTelTracer("my-agent-service",
//	    telemetry.WithOTelExporter(exporter),
//	)
//
//	// 2. 设置为全局 tracer
//	telemetry.SetGlobalTracer(tracer)
//
//	// 3. 在代码中使用
//	ctx, span := tracer.StartSpan(ctx, "agent.chat")
//	defer span.End()
type OTelTracer struct {
	tracer     trace.Tracer
	provider   *sdktrace.TracerProvider
	propagator propagation.TextMapPropagator
}

// OTelOption 配置 OpenTelemetry tracer 的选项
type OTelOption func(*otelConfig)

// otelConfig OpenTelemetry 配置
type otelConfig struct {
	serviceName    string
	serviceVersion string
	spanProcessors []sdktrace.SpanProcessor
	sampler        sdktrace.Sampler
	attributes     []attribute.KeyValue
}

// WithOTelExporter 设置 trace exporter
func WithOTelExporter(exporter sdktrace.SpanExporter) OTelOption {
	return func(cfg *otelConfig) {
		cfg.spanProcessors = append(cfg.spanProcessors, sdktrace.NewBatchSpanProcessor(exporter))
	}
}

// WithOTelSpanProcessor 添加自定义 span processor
func WithOTelSpanProcessor(processor sdktrace.SpanProcessor) OTelOption {
	return func(cfg *otelConfig) {
		cfg.spanProcessors = append(cfg.spanProcessors, processor)
	}
}

// WithOTelSampler 设置采样器
func WithOTelSampler(sampler sdktrace.Sampler) OTelOption {
	return func(cfg *otelConfig) {
		cfg.sampler = sampler
	}
}

// WithOTelServiceVersion 设置服务版本
func WithOTelServiceVersion(version string) OTelOption {
	return func(cfg *otelConfig) {
		cfg.serviceVersion = version
	}
}

// WithOTelAttributes 设置额外的 resource 属性
func WithOTelAttributes(attrs ...attribute.KeyValue) OTelOption {
	return func(cfg *otelConfig) {
		cfg.attributes = append(cfg.attributes, attrs...)
	}
}

// NewOTelTracer 创建 OpenTelemetry 追踪器
func NewOTelTracer(serviceName string, opts ...OTelOption) (*OTelTracer, error) {
	cfg := &otelConfig{
		serviceName:    serviceName,
		serviceVersion: "0.1.0",
		spanProcessors: make([]sdktrace.SpanProcessor, 0),
		sampler:        sdktrace.AlwaysSample(), // 默认采样所有
		attributes:     make([]attribute.KeyValue, 0),
	}

	for _, opt := range opts {
		opt(cfg)
	}

	// 创建 Resource
	resourceAttrs := []attribute.KeyValue{
		semconv.ServiceName(cfg.serviceName),
		semconv.ServiceVersion(cfg.serviceVersion),
	}
	resourceAttrs = append(resourceAttrs, cfg.attributes...)

	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(resourceAttrs...),
		resource.WithTelemetrySDK(), // 添加 SDK 信息
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	// 创建 TracerProvider
	tpOpts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(cfg.sampler),
	}

	// 添加所有 span processors
	for _, processor := range cfg.spanProcessors {
		tpOpts = append(tpOpts, sdktrace.WithSpanProcessor(processor))
	}

	tp := sdktrace.NewTracerProvider(tpOpts...)

	// 设置全局 TracerProvider
	otel.SetTracerProvider(tp)

	// 设置全局 Propagator（用于分布式追踪）
	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(propagator)

	return &OTelTracer{
		tracer:     tp.Tracer(serviceName),
		provider:   tp,
		propagator: propagator,
	}, nil
}

// StartSpan 实现 Tracer 接口
func (t *OTelTracer) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span) {
	// 解析选项
	config := &SpanConfig{
		Kind:       SpanKindInternal,
		Attributes: make([]Attribute, 0),
	}
	for _, opt := range opts {
		opt(config)
	}

	// 转换 SpanKind
	otelKind := convertSpanKind(config.Kind)

	// 转换属性
	otelAttrs := convertAttributes(config.Attributes)

	// 创建 OTel span options
	spanOpts := []trace.SpanStartOption{
		trace.WithSpanKind(otelKind),
		trace.WithAttributes(otelAttrs...),
	}

	if !config.StartTime.IsZero() {
		spanOpts = append(spanOpts, trace.WithTimestamp(config.StartTime))
	}

	// 创建 span
	ctx, otelSpan := t.tracer.Start(ctx, name, spanOpts...)

	return ctx, &OTelSpan{
		span: otelSpan,
	}
}

// Extract 从 carrier 中提取 trace context
func (t *OTelTracer) Extract(ctx context.Context, carrier any) context.Context {
	if textMapCarrier, ok := carrier.(propagation.TextMapCarrier); ok {
		return t.propagator.Extract(ctx, textMapCarrier)
	}
	return ctx
}

// Inject 将 trace context 注入到 carrier
func (t *OTelTracer) Inject(ctx context.Context, carrier any) error {
	if textMapCarrier, ok := carrier.(propagation.TextMapCarrier); ok {
		t.propagator.Inject(ctx, textMapCarrier)
		return nil
	}
	return errors.New("carrier is not a TextMapCarrier")
}

// Shutdown 关闭 tracer，刷新所有待处理的 spans
func (t *OTelTracer) Shutdown(ctx context.Context) error {
	return t.provider.Shutdown(ctx)
}

// ForceFlush 强制刷新所有待处理的 spans
func (t *OTelTracer) ForceFlush(ctx context.Context) error {
	return t.provider.ForceFlush(ctx)
}

// OTelSpan 适配 Span 接口
type OTelSpan struct {
	span trace.Span
}

// End 实现 Span 接口
func (s *OTelSpan) End() {
	s.span.End()
}

// SetAttributes 实现 Span 接口
func (s *OTelSpan) SetAttributes(attrs ...Attribute) {
	otelAttrs := convertAttributes(attrs)
	s.span.SetAttributes(otelAttrs...)
}

// SetStatus 实现 Span 接口
func (s *OTelSpan) SetStatus(code StatusCode, description string) {
	otelCode := convertStatusCode(code)
	s.span.SetStatus(otelCode, description)
}

// AddEvent 实现 Span 接口
func (s *OTelSpan) AddEvent(name string, attrs ...Attribute) {
	otelAttrs := convertAttributes(attrs)
	s.span.AddEvent(name, trace.WithAttributes(otelAttrs...))
}

// RecordError 实现 Span 接口
func (s *OTelSpan) RecordError(err error) {
	s.span.RecordError(err)
	s.span.SetStatus(codes.Error, err.Error())
}

// SpanContext 实现 Span 接口
func (s *OTelSpan) SpanContext() SpanContext {
	spanCtx := s.span.SpanContext()
	return &OTelSpanContext{
		spanContext: spanCtx,
	}
}

// OTelSpanContext 适配 SpanContext 接口
type OTelSpanContext struct {
	spanContext trace.SpanContext
}

// TraceID 实现 SpanContext 接口
func (c *OTelSpanContext) TraceID() string {
	return c.spanContext.TraceID().String()
}

// SpanID 实现 SpanContext 接口
func (c *OTelSpanContext) SpanID() string {
	return c.spanContext.SpanID().String()
}

// IsSampled 实现 SpanContext 接口
func (c *OTelSpanContext) IsSampled() bool {
	return c.spanContext.IsSampled()
}

// 类型转换辅助函数

// convertSpanKind 转换 SpanKind
func convertSpanKind(kind SpanKind) trace.SpanKind {
	switch kind {
	case SpanKindInternal:
		return trace.SpanKindInternal
	case SpanKindServer:
		return trace.SpanKindServer
	case SpanKindClient:
		return trace.SpanKindClient
	case SpanKindProducer:
		return trace.SpanKindProducer
	case SpanKindConsumer:
		return trace.SpanKindConsumer
	default:
		return trace.SpanKindInternal
	}
}

// convertStatusCode 转换 StatusCode
func convertStatusCode(code StatusCode) codes.Code {
	switch code {
	case StatusCodeUnset:
		return codes.Unset
	case StatusCodeOK:
		return codes.Ok
	case StatusCodeError:
		return codes.Error
	default:
		return codes.Unset
	}
}

// convertAttributes 转换属性
func convertAttributes(attrs []Attribute) []attribute.KeyValue {
	otelAttrs := make([]attribute.KeyValue, len(attrs))
	for i, attr := range attrs {
		otelAttrs[i] = convertAttribute(attr)
	}
	return otelAttrs
}

// convertAttribute 转换单个属性
func convertAttribute(attr Attribute) attribute.KeyValue {
	switch v := attr.Value.(type) {
	case string:
		return attribute.String(attr.Key, v)
	case int:
		return attribute.Int(attr.Key, v)
	case int64:
		return attribute.Int64(attr.Key, v)
	case float64:
		return attribute.Float64(attr.Key, v)
	case bool:
		return attribute.Bool(attr.Key, v)
	default:
		return attribute.String(attr.Key, fmt.Sprintf("%v", v))
	}
}

// 常用的预定义属性键（基于 OpenTelemetry 语义约定）
var (
	// Agent 相关
	AttrAgentID         = "agent.id"
	AttrAgentName       = "agent.name"
	AttrAgentTemplateID = "agent.template_id"

	// LLM 相关
	AttrLLMProvider     = "llm.provider"
	AttrLLMModel        = "llm.model"
	AttrLLMMaxTokens    = "llm.max_tokens"
	AttrLLMInputTokens  = "llm.usage.input_tokens"
	AttrLLMOutputTokens = "llm.usage.output_tokens"
	AttrLLMTotalTokens  = "llm.usage.total_tokens"

	// Tool 相关
	AttrToolName     = "tool.name"
	AttrToolCallID   = "tool.call_id"
	AttrToolDuration = "tool.duration_ms"
	AttrToolSuccess  = "tool.success"

	// Session 相关
	AttrSessionID    = "session.id"
	AttrUserID       = "user.id"
	AttrInvocationID = "invocation.id"
	AttrBranch       = "agent.branch"
)
