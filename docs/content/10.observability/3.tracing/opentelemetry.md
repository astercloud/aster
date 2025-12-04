# OpenTelemetry 分布式追踪

aster Server 内置 OpenTelemetry 支持，提供完整的分布式追踪能力。

## 快速开始

### 启用追踪

```go
config := &server.Config{
    Observability: server.ObservabilityConfig{
        Enabled: true,
        Tracing: server.TracingConfig{
            Enabled: true,
            ServiceName: "aster",
            ServiceVersion: "v0.11.0",
            Environment: "production",
            OTLPEndpoint: "localhost:4318",
            OTLPInsecure: true,
            SamplingRate: 1.0, // 100% 采样
        },
    },
}

srv, _ := server.New(config, deps)
srv.Start()
```

### 环境变量配置

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
export OTEL_SERVICE_NAME=aster
export OTEL_TRACES_SAMPLER=always_on
```

## 支持的后端

### Jaeger (推荐)

```bash
# 使用 Docker 启动 Jaeger
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 4318:4318 \
  jaegertracing/all-in-one:latest

# 访问 UI
open http://localhost:16686
```

### Zipkin

```bash
# 启动 Zipkin
docker run -d --name zipkin \
  -p 9411:9411 \
  openzipkin/zipkin

# 访问 UI
open http://localhost:9411
```

### OTLP Collector

```yaml
# otel-collector-config.yaml
receivers:
  otlp:
    protocols:
      http:
        endpoint: 0.0.0.0:4318

exporters:
  jaeger:
    endpoint: jaeger:14250
  logging:
    loglevel: debug

service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [jaeger, logging]
```

```bash
docker run -d --name otel-collector \
  -p 4318:4318 \
  -v $(pwd)/otel-collector-config.yaml:/etc/otel-collector-config.yaml \
  otel/opentelemetry-collector:latest \
  --config=/etc/otel-collector-config.yaml
```

## 自动追踪

### HTTP 请求

所有 HTTP 请求自动被追踪：

```
Span: GET /v1/agents
├─ Duration: 45ms
├─ http.method: GET
├─ http.url: /v1/agents
├─ http.status_code: 200
└─ http.request_content_length: 0
```

### 跨服务传播

Trace context 自动在服务间传播：

```
Service A → Service B → Service C
   |            |            |
TraceID: abc123 (same across all services)
   |            |            |
SpanID: 001  SpanID: 002  SpanID: 003
   |            |            |
Parent: -    Parent: 001  Parent: 002
```

## 手动追踪

### 添加自定义 Span

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
)

func (h *AgentHandler) Create(c *gin.Context) {
    ctx := c.Request.Context()
    tracer := otel.Tracer("aster")

    // 开始新的 span
    ctx, span := tracer.Start(ctx, "agent.create")
    defer span.End()

    // 添加属性
    span.SetAttributes(
        attribute.String("agent.id", agentID),
        attribute.String("agent.type", agentType),
    )

    // 业务逻辑...

    // 记录事件
    span.AddEvent("Agent created successfully")
}
```

### 记录错误

```go
if err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())
    return err
}
```

### 嵌套 Span

```go
func processAgent(ctx context.Context) error {
    ctx, span := tracer.Start(ctx, "process_agent")
    defer span.End()

    // 子操作 1
    if err := validateAgent(ctx); err != nil {
        span.RecordError(err)
        return err
    }

    // 子操作 2
    if err := saveAgent(ctx); err != nil {
        span.RecordError(err)
        return err
    }

    return nil
}

func validateAgent(ctx context.Context) error {
    // 自动成为 process_agent 的子 span
    ctx, span := tracer.Start(ctx, "validate_agent")
    defer span.End()

    // 验证逻辑...
    return nil
}
```

## 采样策略

### Always On (开发)

```go
TracingConfig{
    SamplingRate: 1.0,  // 100% 采样
}
```

### Probabilistic (生产)

```go
TracingConfig{
    SamplingRate: 0.1,  // 10% 采样
}
```

### TraceID Ratio

基于 TraceID 的确定性采样：

```go
// 在 tracing.go 中已实现
sdktrace.WithSampler(sdktrace.TraceIDRatioBased(config.SamplingRate))
```

## Trace 分析

### 查看 Trace

在 Jaeger UI 中：

1. 选择 Service: aster
2. 选择 Operation: GET /v1/agents
3. 点击 "Find Traces"

### 关键指标

- **Duration**: Span 持续时间
- **Tags**: Span 属性（method, path, status）
- **Logs**: Span 事件
- **Process**: 服务信息

### 典型 Trace 示例

```
POST /v1/agents/chat
├─ 150ms total
├─ authentication (5ms)
├─ load_agent (10ms)
├─ process_message (120ms)
│  ├─ parse_input (5ms)
│  ├─ call_llm (100ms)
│  └─ format_response (15ms)
└─ save_session (15ms)
```

## 性能影响

### 开销

- **CPU**: <1% overhead
- **Memory**: ~5MB per 10,000 spans
- **Latency**: <0.5ms per span

### 优化

1. **降低采样率**

   ```go
   SamplingRate: 0.1  // 仅采样 10%
   ```

2. **批量导出**

   ```go
   sdktrace.WithBatcher(exporter,
       sdktrace.WithMaxExportBatchSize(512),
       sdktrace.WithBatchTimeout(5*time.Second),
   )
   ```

3. **限制属性**
   ```go
   // 只添加关键属性
   span.SetAttributes(
       attribute.String("key_field", value),
   )
   ```

## 与 Metrics 集成

### Exemplars

将 traces 关联到 metrics：

```promql
# Prometheus query with exemplar
rate(http_request_duration_seconds_bucket{job="aster"}[5m])
```

在 Grafana 中点击 exemplar 可直接跳转到对应的 trace。

## Docker Compose 完整示例

```yaml
version: "3.8"

services:
  aster:
    image: aster:latest
    ports:
      - "8080:8080"
    environment:
      - OBSERVABILITY_TRACING_ENABLED=true
      - OBSERVABILITY_TRACING_OTLP_ENDPOINT=otel-collector:4318
    depends_on:
      - otel-collector

  otel-collector:
    image: otel/opentelemetry-collector:latest
    command: ["--config=/etc/otel-collector-config.yaml"]
    volumes:
      - ./otel-collector-config.yaml:/etc/otel-collector-config.yaml
    ports:
      - "4318:4318"
    depends_on:
      - jaeger

  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"
      - "14250:14250"

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_AUTH_ANONYMOUS_ENABLED=true
```

## 最佳实践

### 1. Span 命名

使用有意义的名称：

```go
// ✅ Good
tracer.Start(ctx, "agent.create")
tracer.Start(ctx, "database.query")

// ❌ Bad
tracer.Start(ctx, "operation")
tracer.Start(ctx, "step1")
```

### 2. 添加上下文

```go
span.SetAttributes(
    attribute.String("user.id", userID),
    attribute.String("agent.type", agentType),
    attribute.Int("message.length", len(message)),
)
```

### 3. 错误处理

```go
if err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, "Failed to process")
}
```

### 4. 控制粒度

- ✅ 追踪重要操作（数据库查询、外部调用）
- ❌ 避免过度追踪（每个函数调用）

### 5. 采样策略

- 开发：100% 采样
- 测试：50% 采样
- 生产：10-20% 采样

## 故障排查

### Traces 不显示

1. **检查配置**

   ```go
   config.Observability.Tracing.Enabled = true
   ```

2. **检查连接**

   ```bash
   telnet localhost 4318
   ```

3. **检查日志**
   ```bash
   # 查看 exporter 错误
   docker logs aster 2>&1 | grep "trace"
   ```

### 高延迟

- 检查 exporter 配置
- 增加批量大小
- 降低采样率

### 内存泄漏

- 检查 span 是否正确关闭（defer span.End()）
- 减少 span 数量
- 配置 span 限制

## 相关资源

- [OpenTelemetry 文档](https://opentelemetry.io/docs/)
- [Jaeger 文档](https://www.jaegertracing.io/docs/)
- [Go SDK](https://github.com/open-telemetry/opentelemetry-go)
- [最佳实践](https://opentelemetry.io/docs/concepts/signals/traces/)
