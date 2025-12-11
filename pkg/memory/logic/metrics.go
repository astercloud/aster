package logic

import (
	"sync"
	"time"
)

// Metrics Logic Memory 指标收集器
// 提供可观测性支持，可以对接 Prometheus 或其他监控系统
type Metrics struct {
	mu sync.RWMutex

	// 计数器
	memoryTotal        map[string]int64 // namespace -> count
	memorySaveTotal    int64
	memorySaveErrors   int64
	memoryGetTotal     int64
	memoryGetErrors    int64
	memoryDeleteTotal  int64
	eventProcessTotal  int64
	eventProcessErrors int64
	consolidationTotal int64
	pruneTotal         int64

	// 按类型和作用域统计
	memoryByType  map[string]int64
	memoryByScope map[MemoryScope]int64

	// 直方图数据（简化版）
	processEventDurations []time.Duration
	retrieveDurations     []time.Duration
	saveDurations         []time.Duration

	// 最大保留的样本数
	maxSamples int
}

// NewMetrics 创建指标收集器
func NewMetrics() *Metrics {
	return &Metrics{
		memoryTotal:           make(map[string]int64),
		memoryByType:          make(map[string]int64),
		memoryByScope:         make(map[MemoryScope]int64),
		processEventDurations: make([]time.Duration, 0, 1000),
		retrieveDurations:     make([]time.Duration, 0, 1000),
		saveDurations:         make([]time.Duration, 0, 1000),
		maxSamples:            1000,
	}
}

// RecordSave 记录保存操作
func (m *Metrics) RecordSave(namespace, memoryType string, scope MemoryScope, duration time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.memorySaveTotal++
	if err != nil {
		m.memorySaveErrors++
		return
	}

	m.memoryTotal[namespace]++
	m.memoryByType[memoryType]++
	m.memoryByScope[scope]++

	// 记录耗时
	if len(m.saveDurations) < m.maxSamples {
		m.saveDurations = append(m.saveDurations, duration)
	}
}

// RecordGet 记录获取操作
func (m *Metrics) RecordGet(duration time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.memoryGetTotal++
	if err != nil {
		m.memoryGetErrors++
	}

	// 记录耗时
	if len(m.retrieveDurations) < m.maxSamples {
		m.retrieveDurations = append(m.retrieveDurations, duration)
	}
}

// RecordDelete 记录删除操作
func (m *Metrics) RecordDelete(namespace string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.memoryDeleteTotal++
	if m.memoryTotal[namespace] > 0 {
		m.memoryTotal[namespace]--
	}
}

// RecordEventProcess 记录事件处理
func (m *Metrics) RecordEventProcess(eventType string, duration time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.eventProcessTotal++
	if err != nil {
		m.eventProcessErrors++
	}

	// 记录耗时
	if len(m.processEventDurations) < m.maxSamples {
		m.processEventDurations = append(m.processEventDurations, duration)
	}
}

// RecordConsolidation 记录合并操作
func (m *Metrics) RecordConsolidation(mergedGroups, deletedMemories int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.consolidationTotal++
}

// RecordPrune 记录清理操作
func (m *Metrics) RecordPrune(deletedCount int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.pruneTotal++
}

// GetSnapshot 获取指标快照
func (m *Metrics) GetSnapshot() *MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := &MetricsSnapshot{
		Timestamp:          time.Now(),
		MemorySaveTotal:    m.memorySaveTotal,
		MemorySaveErrors:   m.memorySaveErrors,
		MemoryGetTotal:     m.memoryGetTotal,
		MemoryGetErrors:    m.memoryGetErrors,
		MemoryDeleteTotal:  m.memoryDeleteTotal,
		EventProcessTotal:  m.eventProcessTotal,
		EventProcessErrors: m.eventProcessErrors,
		ConsolidationTotal: m.consolidationTotal,
		PruneTotal:         m.pruneTotal,
		MemoryByNamespace:  make(map[string]int64),
		MemoryByType:       make(map[string]int64),
		MemoryByScope:      make(map[MemoryScope]int64),
	}

	// 复制 map
	for k, v := range m.memoryTotal {
		snapshot.MemoryByNamespace[k] = v
	}
	for k, v := range m.memoryByType {
		snapshot.MemoryByType[k] = v
	}
	for k, v := range m.memoryByScope {
		snapshot.MemoryByScope[k] = v
	}

	// 计算平均耗时
	snapshot.AvgSaveDuration = m.calculateAvgDuration(m.saveDurations)
	snapshot.AvgGetDuration = m.calculateAvgDuration(m.retrieveDurations)
	snapshot.AvgEventProcessDuration = m.calculateAvgDuration(m.processEventDurations)

	// 计算 P99
	snapshot.P99SaveDuration = m.calculateP99Duration(m.saveDurations)
	snapshot.P99GetDuration = m.calculateP99Duration(m.retrieveDurations)
	snapshot.P99EventProcessDuration = m.calculateP99Duration(m.processEventDurations)

	return snapshot
}

// Reset 重置指标
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.memorySaveTotal = 0
	m.memorySaveErrors = 0
	m.memoryGetTotal = 0
	m.memoryGetErrors = 0
	m.memoryDeleteTotal = 0
	m.eventProcessTotal = 0
	m.eventProcessErrors = 0
	m.consolidationTotal = 0
	m.pruneTotal = 0

	m.memoryTotal = make(map[string]int64)
	m.memoryByType = make(map[string]int64)
	m.memoryByScope = make(map[MemoryScope]int64)

	m.processEventDurations = m.processEventDurations[:0]
	m.retrieveDurations = m.retrieveDurations[:0]
	m.saveDurations = m.saveDurations[:0]
}

// calculateAvgDuration 计算平均耗时
func (m *Metrics) calculateAvgDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}

// calculateP99Duration 计算 P99 耗时
func (m *Metrics) calculateP99Duration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	// 复制并排序
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)

	// 简单冒泡排序（样本量小，性能可接受）
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	// P99 索引
	idx := int(float64(len(sorted)) * 0.99)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}

	return sorted[idx]
}

// MetricsSnapshot 指标快照
type MetricsSnapshot struct {
	Timestamp time.Time

	// 计数器
	MemorySaveTotal    int64
	MemorySaveErrors   int64
	MemoryGetTotal     int64
	MemoryGetErrors    int64
	MemoryDeleteTotal  int64
	EventProcessTotal  int64
	EventProcessErrors int64
	ConsolidationTotal int64
	PruneTotal         int64

	// 分布
	MemoryByNamespace map[string]int64
	MemoryByType      map[string]int64
	MemoryByScope     map[MemoryScope]int64

	// 耗时统计
	AvgSaveDuration         time.Duration
	AvgGetDuration          time.Duration
	AvgEventProcessDuration time.Duration
	P99SaveDuration         time.Duration
	P99GetDuration          time.Duration
	P99EventProcessDuration time.Duration
}

// TotalMemories 返回总 Memory 数量
func (s *MetricsSnapshot) TotalMemories() int64 {
	var total int64
	for _, v := range s.MemoryByNamespace {
		total += v
	}
	return total
}

// SaveErrorRate 返回保存错误率
func (s *MetricsSnapshot) SaveErrorRate() float64 {
	if s.MemorySaveTotal == 0 {
		return 0
	}
	return float64(s.MemorySaveErrors) / float64(s.MemorySaveTotal)
}

// GetErrorRate 返回获取错误率
func (s *MetricsSnapshot) GetErrorRate() float64 {
	if s.MemoryGetTotal == 0 {
		return 0
	}
	return float64(s.MemoryGetErrors) / float64(s.MemoryGetTotal)
}

// EventProcessErrorRate 返回事件处理错误率
func (s *MetricsSnapshot) EventProcessErrorRate() float64 {
	if s.EventProcessTotal == 0 {
		return 0
	}
	return float64(s.EventProcessErrors) / float64(s.EventProcessTotal)
}

// PrometheusExporter Prometheus 导出器接口
// 应用层可以实现此接口将指标导出到 Prometheus
type PrometheusExporter interface {
	// ExportGauge 导出 Gauge 指标
	ExportGauge(name string, value float64, labels map[string]string)

	// ExportCounter 导出 Counter 指标
	ExportCounter(name string, value float64, labels map[string]string)

	// ExportHistogram 导出 Histogram 指标
	ExportHistogram(name string, value float64, labels map[string]string)
}

// ExportToPrometheus 导出指标到 Prometheus
func (m *Metrics) ExportToPrometheus(exporter PrometheusExporter) {
	snapshot := m.GetSnapshot()

	// 导出计数器
	exporter.ExportCounter("logic_memory_save_total", float64(snapshot.MemorySaveTotal), nil)
	exporter.ExportCounter("logic_memory_save_errors_total", float64(snapshot.MemorySaveErrors), nil)
	exporter.ExportCounter("logic_memory_get_total", float64(snapshot.MemoryGetTotal), nil)
	exporter.ExportCounter("logic_memory_get_errors_total", float64(snapshot.MemoryGetErrors), nil)
	exporter.ExportCounter("logic_memory_delete_total", float64(snapshot.MemoryDeleteTotal), nil)
	exporter.ExportCounter("logic_memory_event_process_total", float64(snapshot.EventProcessTotal), nil)
	exporter.ExportCounter("logic_memory_event_process_errors_total", float64(snapshot.EventProcessErrors), nil)
	exporter.ExportCounter("logic_memory_consolidation_total", float64(snapshot.ConsolidationTotal), nil)
	exporter.ExportCounter("logic_memory_prune_total", float64(snapshot.PruneTotal), nil)

	// 导出 Gauge（按 namespace）
	for namespace, count := range snapshot.MemoryByNamespace {
		exporter.ExportGauge("logic_memory_total", float64(count), map[string]string{"namespace": namespace})
	}

	// 导出 Gauge（按 type）
	for memType, count := range snapshot.MemoryByType {
		exporter.ExportGauge("logic_memory_by_type", float64(count), map[string]string{"type": memType})
	}

	// 导出 Gauge（按 scope）
	for scope, count := range snapshot.MemoryByScope {
		exporter.ExportGauge("logic_memory_by_scope", float64(count), map[string]string{"scope": string(scope)})
	}

	// 导出 Histogram
	exporter.ExportHistogram("logic_memory_save_duration_seconds", snapshot.AvgSaveDuration.Seconds(), nil)
	exporter.ExportHistogram("logic_memory_get_duration_seconds", snapshot.AvgGetDuration.Seconds(), nil)
	exporter.ExportHistogram("logic_memory_event_process_duration_seconds", snapshot.AvgEventProcessDuration.Seconds(), nil)
}
