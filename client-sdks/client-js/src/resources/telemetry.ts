/**
 * Telemetry 资源类
 * 可观测性：Metrics、Traces、Logs
 */

import { BaseResource, ClientOptions } from "./base";
import {
  MetricInfo,
  MetricDataPoint,
  MetricQueryOptions,
  TraceInfo,
  TraceQueryOptions,
  LogEntry,
  LogQueryOptions,
  TelemetryConfig,
  HealthCheckResult,
  PerformanceMetrics,
  UsageStatistics,
  ExportTelemetryRequest,
  ExportTelemetryResult,
} from "../types/telemetry";

/**
 * Telemetry 资源类
 */
export class TelemetryResource extends BaseResource {
  constructor(options: ClientOptions) {
    super(options);
  }

  // ==========================================================================
  // Configuration
  // ==========================================================================

  /**
   * 获取 Telemetry 配置
   * @returns Telemetry 配置
   */
  async getConfig(): Promise<TelemetryConfig> {
    return this.request<TelemetryConfig>("/v1/telemetry/config");
  }

  /**
   * 更新 Telemetry 配置
   * @param config 配置更新
   * @returns 更新后的配置
   */
  async updateConfig(config: Partial<TelemetryConfig>): Promise<TelemetryConfig> {
    return this.request<TelemetryConfig>("/v1/telemetry/config", {
      method: "PATCH",
      body: config,
    });
  }

  // ==========================================================================
  // Metrics
  // ==========================================================================

  /**
   * 列出所有 Metrics
   * @returns Metric 列表
   */
  async listMetrics(): Promise<MetricInfo[]> {
    const result = await this.request<{ metrics: MetricInfo[] }>("/v1/telemetry/metrics");
    return result.metrics;
  }

  /**
   * 获取 Metric 详情
   * @param name Metric 名称
   * @returns Metric 信息
   */
  async getMetric(name: string): Promise<MetricInfo> {
    return this.request<MetricInfo>(`/v1/telemetry/metrics/${name}`);
  }

  /**
   * 查询 Metric 数据
   * @param options 查询选项
   * @returns 数据点列表
   */
  async queryMetrics(options: MetricQueryOptions): Promise<MetricDataPoint[]> {
    const result = await this.request<{ dataPoints: MetricDataPoint[] }>("/v1/telemetry/metrics/query", {
      method: "POST",
      body: options,
    });
    return result.dataPoints;
  }

  /**
   * 记录自定义 Metric
   * @param name Metric 名称
   * @param value 值
   * @param labels 标签（可选）
   */
  async recordMetric(name: string, value: number, labels?: Record<string, string>): Promise<void>;
  async recordMetric(data: { name: string; type?: string; value: number; labels?: Record<string, string> }): Promise<void>;
  async recordMetric(
    nameOrData:
      | string
      | {
          name: string;
          type?: string;
          value: number;
          labels?: Record<string, string>;
        },
    value?: number,
    labels?: Record<string, string>,
  ): Promise<void> {
    let body: any;
    if (typeof nameOrData === "string") {
      body = { name: nameOrData, value: value!, labels };
    } else {
      body = nameOrData;
    }
    await this.request("/v1/telemetry/metrics", {
      method: "POST",
      body,
    });
  }

  // ==========================================================================
  // Traces
  // ==========================================================================

  /**
   * 记录 Trace
   * @param trace Trace 数据
   * @returns Trace 信息
   */
  async recordTrace(trace: { name: string; span_id: string; trace_id?: string; parent_id?: string; [key: string]: any }): Promise<TraceInfo> {
    return this.request<TraceInfo>("/v1/telemetry/traces", {
      method: "POST",
      body: trace,
    });
  }

  /**
   * 查询 Traces
   * @param options 查询选项
   * @returns Trace 列表
   */
  async queryTraces(options?: TraceQueryOptions): Promise<TraceInfo[]> {
    const result = await this.request<{ traces: TraceInfo[] }>("/v1/telemetry/traces/query", {
      method: "POST",
      body: options || {},
    });
    return result.traces;
  }

  /**
   * 列出 Traces（别名方法）
   * @param options 查询选项
   * @returns Trace 列表
   */
  async listTraces(options?: TraceQueryOptions): Promise<TraceInfo[]> {
    return this.queryTraces(options);
  }

  /**
   * 获取 Trace 详情
   * @param traceId Trace ID
   * @returns Trace 信息
   */
  async getTrace(traceId: string): Promise<TraceInfo> {
    return this.request<TraceInfo>(`/v1/telemetry/traces/${traceId}`);
  }

  /**
   * 获取 Trace 的所有 Spans
   * @param traceId Trace ID
   * @returns Span 列表
   */
  async getTraceSpans(traceId: string): Promise<TraceInfo[]> {
    const result = await this.request<{ spans: TraceInfo[] }>(`/v1/telemetry/traces/${traceId}/spans`);
    return result.spans;
  }

  // ==========================================================================
  // Logs
  // ==========================================================================

  /**
   * 查询日志
   * @param options 查询选项
   * @returns 日志列表
   */
  async queryLogs(options: LogQueryOptions): Promise<LogEntry[]> {
    const result = await this.request<{ logs: LogEntry[] }>("/v1/telemetry/logs/query", {
      method: "POST",
      body: options,
    });
    return result.logs;
  }

  /**
   * 写入日志
   * @param entry 日志条目
   */
  async writeLog(entry: Omit<LogEntry, "timestamp">): Promise<void> {
    await this.request("/v1/telemetry/logs", {
      method: "POST",
      body: entry,
    });
  }

  /**
   * 获取与 Trace 关联的日志
   * @param traceId Trace ID
   * @returns 日志列表
   */
  async getTraceLogs(traceId: string): Promise<LogEntry[]> {
    const result = await this.request<{ logs: LogEntry[] }>(`/v1/telemetry/traces/${traceId}/logs`);
    return result.logs;
  }

  // ==========================================================================
  // Health and Status
  // ==========================================================================

  /**
   * 健康检查
   * @returns 健康状态
   */
  async healthCheck(): Promise<HealthCheckResult> {
    return this.request<HealthCheckResult>("/v1/telemetry/health");
  }

  /**
   * 获取系统状态
   * @returns 状态信息
   */
  async getStatus(): Promise<{
    status: string;
    uptime: number;
    version: string;
    [key: string]: any;
  }> {
    return this.request("/v1/telemetry/status");
  }

  // ==========================================================================
  // Performance and Usage
  // ==========================================================================

  /**
   * 获取性能指标
   * @param timeRange 时间范围
   * @returns 性能指标
   */
  async getPerformanceMetrics(timeRange?: { start: string; end: string }): Promise<PerformanceMetrics> {
    return this.request<PerformanceMetrics>("/v1/telemetry/performance", {
      params: timeRange,
    });
  }

  /**
   * 获取使用统计
   * @param timeRange 时间范围
   * @returns 使用统计
   */
  async getUsageStatistics(timeRange?: { start: string; end: string }): Promise<UsageStatistics> {
    return this.request<UsageStatistics>("/v1/telemetry/usage", {
      params: timeRange,
    });
  }

  // ==========================================================================
  // Export
  // ==========================================================================

  /**
   * 导出 Telemetry 数据
   * @param request 导出请求
   * @returns 导出结果
   */
  async export(request: ExportTelemetryRequest): Promise<ExportTelemetryResult> {
    return this.request<ExportTelemetryResult>("/v1/telemetry/export", {
      method: "POST",
      body: request,
    });
  }

  /**
   * 导出 Metrics（便捷方法）
   * @param format 格式
   * @param timeRange 时间范围
   * @returns 导出结果
   */
  async exportMetrics(format: "json" | "csv" | "prometheus", timeRange?: { start: string; end: string }): Promise<ExportTelemetryResult> {
    return this.export({
      type: "metrics",
      format,
      timeRange,
    });
  }

  /**
   * 导出 Traces（便捷方法）
   * @param format 格式
   * @param timeRange 时间范围
   * @returns 导出结果
   */
  async exportTraces(format: "json" | "opentelemetry", timeRange?: { start: string; end: string }): Promise<ExportTelemetryResult> {
    return this.export({
      type: "traces",
      format,
      timeRange,
    });
  }

  /**
   * 导出 Logs（便捷方法）
   * @param format 格式
   * @param timeRange 时间范围
   * @returns 导出结果
   */
  async exportLogs(format: "json" | "csv", timeRange?: { start: string; end: string }): Promise<ExportTelemetryResult> {
    return this.export({
      type: "logs",
      format,
      timeRange,
    });
  }
}
