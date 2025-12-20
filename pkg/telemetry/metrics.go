package telemetry

import (
	"strings"
	"sync"
	"time"
)

// Metrics 提供指标收集能力
// 参考 Google ADK-Go 的 metrics 设计
type Metrics interface {
	// Counter 操作
	IncrementCounter(name string, value int64, labels map[string]string)

	// Gauge 操作
	SetGauge(name string, value float64, labels map[string]string)

	// Histogram 操作
	RecordHistogram(name string, value float64, labels map[string]string)

	// 获取指标快照
	Snapshot() MetricsSnapshot
}

// MetricsSnapshot 指标快照
type MetricsSnapshot struct {
	Counters   map[string]*CounterSnapshot
	Gauges     map[string]*GaugeSnapshot
	Histograms map[string]*HistogramSnapshot
	Timestamp  time.Time
}

// CounterSnapshot 计数器快照
type CounterSnapshot struct {
	Name   string
	Value  int64
	Labels map[string]string
}

// GaugeSnapshot 仪表盘快照
type GaugeSnapshot struct {
	Name   string
	Value  float64
	Labels map[string]string
}

// HistogramSnapshot 直方图快照
type HistogramSnapshot struct {
	Name   string
	Count  int64
	Sum    float64
	Min    float64
	Max    float64
	Mean   float64
	Labels map[string]string
}

// SimpleMetrics 简单的内存 metrics 实现
type SimpleMetrics struct {
	mu         sync.RWMutex
	counters   map[string]*counter
	gauges     map[string]*gauge
	histograms map[string]*histogram
}

type counter struct {
	value  int64
	labels map[string]string
}

type gauge struct {
	value  float64
	labels map[string]string
}

type histogram struct {
	count  int64
	sum    float64
	min    float64
	max    float64
	values []float64
	labels map[string]string
}

// NewSimpleMetrics 创建简单的 metrics 实例
func NewSimpleMetrics() *SimpleMetrics {
	return &SimpleMetrics{
		counters:   make(map[string]*counter),
		gauges:     make(map[string]*gauge),
		histograms: make(map[string]*histogram),
	}
}

func (m *SimpleMetrics) IncrementCounter(name string, value int64, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := makeKey(name, labels)
	if c, ok := m.counters[key]; ok {
		c.value += value
	} else {
		m.counters[key] = &counter{
			value:  value,
			labels: labels,
		}
	}
}

func (m *SimpleMetrics) SetGauge(name string, value float64, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := makeKey(name, labels)
	m.gauges[key] = &gauge{
		value:  value,
		labels: labels,
	}
}

func (m *SimpleMetrics) RecordHistogram(name string, value float64, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := makeKey(name, labels)
	if h, ok := m.histograms[key]; ok {
		h.count++
		h.sum += value
		h.values = append(h.values, value)
		if value < h.min {
			h.min = value
		}
		if value > h.max {
			h.max = value
		}
	} else {
		m.histograms[key] = &histogram{
			count:  1,
			sum:    value,
			min:    value,
			max:    value,
			values: []float64{value},
			labels: labels,
		}
	}
}

func (m *SimpleMetrics) Snapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := MetricsSnapshot{
		Counters:   make(map[string]*CounterSnapshot),
		Gauges:     make(map[string]*GaugeSnapshot),
		Histograms: make(map[string]*HistogramSnapshot),
		Timestamp:  time.Now(),
	}

	// 复制 counters
	for key, c := range m.counters {
		snapshot.Counters[key] = &CounterSnapshot{
			Name:   key,
			Value:  c.value,
			Labels: copyLabels(c.labels),
		}
	}

	// 复制 gauges
	for key, g := range m.gauges {
		snapshot.Gauges[key] = &GaugeSnapshot{
			Name:   key,
			Value:  g.value,
			Labels: copyLabels(g.labels),
		}
	}

	// 复制 histograms
	for key, h := range m.histograms {
		mean := 0.0
		if h.count > 0 {
			mean = h.sum / float64(h.count)
		}

		snapshot.Histograms[key] = &HistogramSnapshot{
			Name:   key,
			Count:  h.count,
			Sum:    h.sum,
			Min:    h.min,
			Max:    h.max,
			Mean:   mean,
			Labels: copyLabels(h.labels),
		}
	}

	return snapshot
}

// AgentMetrics Agent 相关的指标收集器
type AgentMetrics struct {
	metrics Metrics
}

// NewAgentMetrics 创建 Agent metrics
func NewAgentMetrics(metrics Metrics) *AgentMetrics {
	return &AgentMetrics{metrics: metrics}
}

// RecordRequest 记录请求
func (m *AgentMetrics) RecordRequest(agentID string, duration time.Duration) {
	labels := map[string]string{"agent_id": agentID}
	m.metrics.IncrementCounter("agent.requests.total", 1, labels)
	m.metrics.RecordHistogram("agent.request.duration", duration.Seconds(), labels)
}

// RecordTokens 记录 token 使用
func (m *AgentMetrics) RecordTokens(agentID string, inputTokens, outputTokens int64) {
	labels := map[string]string{"agent_id": agentID}
	m.metrics.IncrementCounter("agent.tokens.input", inputTokens, labels)
	m.metrics.IncrementCounter("agent.tokens.output", outputTokens, labels)
}

// RecordToolCall 记录工具调用
func (m *AgentMetrics) RecordToolCall(agentID, toolName string, duration time.Duration, success bool) {
	labels := map[string]string{
		"agent_id":  agentID,
		"tool_name": toolName,
	}

	m.metrics.IncrementCounter("agent.tool_calls.total", 1, labels)
	m.metrics.RecordHistogram("agent.tool_call.duration", duration.Seconds(), labels)

	if !success {
		m.metrics.IncrementCounter("agent.tool_calls.errors", 1, labels)
	}
}

// RecordError 记录错误
func (m *AgentMetrics) RecordError(agentID, errorType string) {
	labels := map[string]string{
		"agent_id":   agentID,
		"error_type": errorType,
	}
	m.metrics.IncrementCounter("agent.errors.total", 1, labels)
}

// SetActiveAgents 设置活跃 Agent 数量
func (m *AgentMetrics) SetActiveAgents(count int) {
	m.metrics.SetGauge("agent.active.count", float64(count), nil)
}

// SetQueueDepth 设置队列深度
func (m *AgentMetrics) SetQueueDepth(agentID string, depth int) {
	labels := map[string]string{"agent_id": agentID}
	m.metrics.SetGauge("agent.queue.depth", float64(depth), labels)
}

// 辅助函数
func makeKey(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}

	key := name
	var keySb262 strings.Builder
	for k, v := range labels {
		keySb262.WriteString(":" + k + "=" + v)
	}
	key += keySb262.String()
	return key
}

func copyLabels(labels map[string]string) map[string]string {
	if labels == nil {
		return nil
	}

	copy := make(map[string]string, len(labels))
	for k, v := range labels {
		copy[k] = v
	}
	return copy
}

// 全局默认 metrics
var globalMetrics Metrics = NewSimpleMetrics()

// SetGlobalMetrics 设置全局 metrics
func SetGlobalMetrics(metrics Metrics) {
	globalMetrics = metrics
}

// GetGlobalMetrics 获取全局 metrics
func GetGlobalMetrics() Metrics {
	return globalMetrics
}

// 便捷函数
func IncrementCounter(name string, value int64, labels map[string]string) {
	globalMetrics.IncrementCounter(name, value, labels)
}

func SetGauge(name string, value float64, labels map[string]string) {
	globalMetrics.SetGauge(name, value, labels)
}

func RecordHistogram(name string, value float64, labels map[string]string) {
	globalMetrics.RecordHistogram(name, value, labels)
}
