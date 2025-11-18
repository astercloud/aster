# Prometheus Metrics

aster Server 提供完整的 Prometheus metrics 支持，用于监控应用性能和业务指标。

## 快速开始

### 启用 Metrics

```go
config := &server.Config{
    Observability: server.ObservabilityConfig{
        Enabled: true,
        Metrics: server.MetricsConfig{
            Enabled: true,
            Endpoint: "/metrics",
        },
    },
}

srv, _ := server.New(config, deps)
srv.Start()
```

### 访问 Metrics

```bash
curl http://localhost:8080/metrics
```

## 可用指标

### HTTP 指标

#### 请求总数
```
aster_http_requests_total{method="GET",path="/v1/agents",status="2xx"}
```
- **类型**: Counter
- **标签**: method, path, status
- **描述**: HTTP 请求总数，按方法、路径和状态分类

#### 请求延迟
```
aster_http_request_duration_seconds{method="GET",path="/v1/agents"}
```
- **类型**: Histogram
- **标签**: method, path
- **描述**: HTTP 请求处理时间（秒）
- **分位数**: p50, p90, p95, p99

#### 请求大小
```
aster_http_request_size_bytes{method="POST",path="/v1/agents"}
```
- **类型**: Histogram
- **标签**: method, path
- **描述**: HTTP 请求body大小（字节）

#### 响应大小
```
aster_http_response_size_bytes{method="GET",path="/v1/agents"}
```
- **类型**: Histogram
- **标签**: method, path
- **描述**: HTTP 响应body大小（字节）

### 业务指标

#### Agents 总数
```
aster_agents_total
```
- **类型**: Gauge
- **描述**: 系统中 Agent 的总数

#### 活跃 Sessions
```
aster_sessions_active
```
- **类型**: Gauge
- **描述**: 当前活跃的 Session 数量

#### 运行中的 Workflows
```
aster_workflows_running
```
- **类型**: Gauge
- **描述**: 当前正在运行的 Workflow 数量

### Go 运行时指标

自动包含标准 Go metrics：

- `go_goroutines` - Goroutine 数量
- `go_threads` - OS 线程数量
- `go_memstats_alloc_bytes` - 已分配内存
- `go_memstats_sys_bytes` - 系统内存
- `go_gc_duration_seconds` - GC 延迟
- `process_cpu_seconds_total` - CPU 使用时间
- `process_resident_memory_bytes` - 常驻内存

## 使用示例

### 更新业务指标

```go
// 在 Handler 中更新指标
func (h *AgentHandler) Create(c *gin.Context) {
    // 创建 agent...
    
    // 更新指标
    if h.metrics != nil {
        count := getAgentCount()  // 获取总数
        h.metrics.SetAgentsTotal(float64(count))
    }
}
```

### 自定义指标

```go
import "github.com/prometheus/client_golang/prometheus"

// 创建自定义指标
customCounter := prometheus.NewCounter(
    prometheus.CounterOpts{
        Namespace: "aster",
        Name:      "custom_operations_total",
        Help:      "Total custom operations",
    },
)

// 注册
prometheus.MustRegister(customCounter)

// 使用
customCounter.Inc()
```

## Prometheus 配置

### prometheus.yml

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'aster'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

### Docker Compose

```yaml
version: '3.8'

services:
  aster:
    image: aster:latest
    ports:
      - "8080:8080"
    environment:
      - OBSERVABILITY_METRICS_ENABLED=true
  
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
```

## Grafana Dashboard

### 导入 Dashboard

1. 打开 Grafana
2. 导航到 Dashboards → Import
3. 上传 `dashboards/aster.json`

### 关键面板

**HTTP Performance**
- Request Rate (req/s)
- Response Time (p50, p90, p99)
- Error Rate (%)

**Business Metrics**
- Active Agents
- Active Sessions
- Running Workflows

**System Resources**
- CPU Usage
- Memory Usage
- Goroutines
- GC Activity

### 示例查询

#### 请求速率
```promql
rate(aster_http_requests_total[5m])
```

#### p99 延迟
```promql
histogram_quantile(0.99, 
  rate(aster_http_request_duration_seconds_bucket[5m])
)
```

#### 错误率
```promql
rate(aster_http_requests_total{status="5xx"}[5m])
  / 
rate(aster_http_requests_total[5m])
```

#### 平均响应大小
```promql
rate(aster_http_response_size_bytes_sum[5m])
  / 
rate(aster_http_response_size_bytes_count[5m])
```

## 告警规则

### alerts.yml

```yaml
groups:
  - name: aster
    rules:
      # 高错误率告警
      - alert: HighErrorRate
        expr: |
          rate(aster_http_requests_total{status="5xx"}[5m])
            / 
          rate(aster_http_requests_total[5m])
          > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value | humanizePercentage }}"
      
      # 高延迟告警
      - alert: HighLatency
        expr: |
          histogram_quantile(0.99,
            rate(aster_http_request_duration_seconds_bucket[5m])
          ) > 1.0
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High latency detected"
          description: "p99 latency is {{ $value }}s"
      
      # 高内存使用告警
      - alert: HighMemoryUsage
        expr: |
          process_resident_memory_bytes > 1e9
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High memory usage"
          description: "Memory usage is {{ $value | humanize }}B"
```

## 最佳实践

### 1. 指标命名

遵循 Prometheus 命名约定：
- 使用小写和下划线
- 使用有意义的前缀 (aster_)
- Counter 后缀 `_total`
- 时间单位后缀 `_seconds`
- 大小单位后缀 `_bytes`

### 2. 标签使用

- 保持标签基数低（避免高基数标签如 user_id）
- 使用有意义的标签名
- 避免在标签值中包含时间戳或 UUID

### 3. 采样

- 使用 Histogram 而不是 Summary（更灵活）
- 配置合适的 buckets
- 考虑采样率以减少开销

### 4. 性能

- Metrics 收集开销：~0.1ms per request
- 内存开销：~100KB per 1000 time series
- 使用标签过滤减少查询开销

## 故障排查

### Metrics 不可用

检查配置：
```go
config.Observability.Metrics.Enabled = true
```

检查端点：
```bash
curl http://localhost:8080/metrics
```

### 指标不更新

确保中间件已注册：
```go
if s.metrics != nil {
    s.router.Use(s.metrics.Middleware())
}
```

### 高内存使用

- 检查时间序列数量
- 减少标签基数
- 增加清理频率

## 相关资源

- [Prometheus 文档](https://prometheus.io/docs/)
- [Grafana 文档](https://grafana.com/docs/)
- [Go Client 库](https://github.com/prometheus/client_golang)
- [最佳实践](https://prometheus.io/docs/practices/naming/)
