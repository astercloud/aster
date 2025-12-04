/**
 * Telemetry 相关类型定义
 * 可观测性：Metrics、Traces、Logs
 */

// ============================================================================
// Metric Types
// ============================================================================

/**
 * Metric 类型
 */
export type MetricType =
  | "counter" // 计数器（只增不减）
  | "gauge" // 仪表盘（可增可减）
  | "histogram" // 直方图（分布统计）
  | "summary"; // 摘要（百分位数）

/**
 * Metric 信息
 */
export interface MetricInfo {
  /** Metric 名称 */
  name: string;
  /** 类型 */
  type: MetricType;
  /** 描述 */
  description: string;
  /** 单位 */
  unit?: string;
  /** 标签 */
  labels?: Record<string, string>;
  /** 当前值 */
  value: number;
  /** 时间戳 */
  timestamp: string;
}

/**
 * Metric 数据点
 */
export interface MetricDataPoint {
  /** 时间戳 */
  timestamp: string;
  /** 值 */
  value: number;
  /** 标签 */
  labels?: Record<string, string>;
}

/**
 * Metric 查询选项
 */
export interface MetricQueryOptions {
  /** Metric 名称（支持通配符） */
  name?: string;
  /** 时间范围 */
  timeRange?: {
    start: string;
    end: string;
  };
  /** 标签过滤 */
  labels?: Record<string, string>;
  /** 聚合方式 */
  aggregation?: "sum" | "avg" | "min" | "max" | "count";
  /** 分组 */
  groupBy?: string[];
}

// ============================================================================
// Trace Types
// ============================================================================

/**
 * Trace 信息
 */
export interface TraceInfo {
  /** Trace ID */
  traceId: string;
  /** Span ID */
  spanId: string;
  /** 父 Span ID */
  parentSpanId?: string;
  /** 操作名称 */
  operationName: string;
  /** 开始时间 */
  startTime: string;
  /** 结束时间 */
  endTime?: string;
  /** 持续时间（毫秒） */
  duration?: number;
  /** 状态 */
  status: "ok" | "error";
  /** 标签 */
  tags?: Record<string, any>;
  /** 事件 */
  events?: TraceEvent[];
}

/**
 * Trace 事件
 */
export interface TraceEvent {
  /** 时间戳 */
  timestamp: string;
  /** 事件名称 */
  name: string;
  /** 属性 */
  attributes?: Record<string, any>;
}

/**
 * Trace 查询选项
 */
export interface TraceQueryOptions {
  /** Trace ID */
  traceId?: string;
  /** 操作名称 */
  operationName?: string;
  /** 时间范围 */
  timeRange?: {
    start: string;
    end: string;
  };
  /** 最小持续时间（毫秒） */
  minDuration?: number;
  /** 最大持续时间（毫秒） */
  maxDuration?: number;
  /** 状态 */
  status?: "ok" | "error";
  /** 标签过滤 */
  tags?: Record<string, any>;
  /** 限制数量 */
  limit?: number;
}

// ============================================================================
// Log Types
// ============================================================================

/**
 * 日志级别
 */
export type LogLevel = "debug" | "info" | "warn" | "error" | "fatal";

/**
 * 日志条目
 */
export interface LogEntry {
  /** 时间戳 */
  timestamp: string;
  /** 级别 */
  level: LogLevel;
  /** 消息 */
  message: string;
  /** 来源 */
  source?: string;
  /** Trace ID */
  traceId?: string;
  /** Span ID */
  spanId?: string;
  /** 属性 */
  attributes?: Record<string, any>;
  /** 错误堆栈 */
  stack?: string;
}

/**
 * 日志查询选项
 */
export interface LogQueryOptions {
  /** 级别 */
  level?: LogLevel;
  /** 时间范围 */
  timeRange?: {
    start: string;
    end: string;
  };
  /** 来源 */
  source?: string;
  /** 搜索关键词 */
  search?: string;
  /** Trace ID */
  traceId?: string;
  /** 限制数量 */
  limit?: number;
  /** 排序 */
  sort?: "asc" | "desc";
}

// ============================================================================
// Telemetry Configuration
// ============================================================================

/**
 * Telemetry 配置
 */
export interface TelemetryConfig {
  /** 是否启用 */
  enabled: boolean;
  /** Metrics 配置 */
  metrics?: {
    enabled: boolean;
    exportInterval?: number; // 导出间隔（秒）
    endpoint?: string;
  };
  /** Traces 配置 */
  traces?: {
    enabled: boolean;
    samplingRate?: number; // 采样率（0-1）
    endpoint?: string;
  };
  /** Logs 配置 */
  logs?: {
    enabled: boolean;
    level?: LogLevel;
    endpoint?: string;
  };
}

// ============================================================================
// Health and Status
// ============================================================================

/**
 * 健康检查结果
 */
export interface HealthCheckResult {
  /** 状态 */
  status: "healthy" | "degraded" | "unhealthy";
  /** 时间戳 */
  timestamp: string;
  /** 组件健康状态 */
  components: Record<string, ComponentHealth>;
  /** 总体信息 */
  info?: Record<string, any>;
}

/**
 * 组件健康状态
 */
export interface ComponentHealth {
  /** 状态 */
  status: "healthy" | "degraded" | "unhealthy";
  /** 消息 */
  message?: string;
  /** 详细信息 */
  details?: Record<string, any>;
}

// ============================================================================
// Performance Metrics
// ============================================================================

/**
 * 性能指标
 */
export interface PerformanceMetrics {
  /** 时间范围 */
  timeRange: {
    start: string;
    end: string;
  };
  /** 请求统计 */
  requests: {
    total: number;
    successful: number;
    failed: number;
    avgLatency: number; // 平均延迟（ms）
    p50Latency: number; // P50 延迟
    p95Latency: number; // P95 延迟
    p99Latency: number; // P99 延迟
  };
  /** Token 统计 */
  tokens?: {
    total: number;
    prompt: number;
    completion: number;
  };
  /** 成本统计 */
  cost?: {
    total: number;
    currency: string;
  };
  /** 错误统计 */
  errors?: {
    total: number;
    byType: Record<string, number>;
  };
}

// ============================================================================
// Usage Statistics
// ============================================================================

/**
 * 使用统计
 */
export interface UsageStatistics {
  /** 时间范围 */
  timeRange: {
    start: string;
    end: string;
  };
  /** Agent 使用 */
  agents?: {
    total: number;
    active: number;
    topAgents: Array<{
      agentId: string;
      requestCount: number;
    }>;
  };
  /** Session 统计 */
  sessions?: {
    total: number;
    active: number;
    avgDuration: number;
  };
  /** Workflow 统计 */
  workflows?: {
    total: number;
    successful: number;
    failed: number;
  };
  /** Tool 使用 */
  tools?: {
    total: number;
    topTools: Array<{
      toolName: string;
      callCount: number;
    }>;
  };
  /** Memory 使用 */
  memory?: {
    workingMemorySize: number;
    semanticMemorySize: number;
  };
}

// ============================================================================
// Export and Integration
// ============================================================================

/**
 * 导出格式
 */
export type ExportFormat =
  | "json"
  | "csv"
  | "prometheus" // Prometheus 格式
  | "opentelemetry"; // OpenTelemetry 格式

/**
 * 导出请求
 */
export interface ExportTelemetryRequest {
  /** 导出类型 */
  type: "metrics" | "traces" | "logs";
  /** 格式 */
  format: ExportFormat;
  /** 时间范围 */
  timeRange?: {
    start: string;
    end: string;
  };
  /** 过滤条件 */
  filter?: any;
}

/**
 * 导出结果
 */
export interface ExportTelemetryResult {
  /** 格式 */
  format: ExportFormat;
  /** 数据 */
  data: string;
  /** 建议文件名 */
  suggestedFilename: string;
  /** 导出时间 */
  exportedAt: string;
}
