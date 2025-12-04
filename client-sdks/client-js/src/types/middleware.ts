/**
 * Middleware 相关类型定义
 * 洋葱模型中间件系统，8+ 内置中间件
 */

// ============================================================================
// Middleware Types
// ============================================================================

/**
 * 内置 Middleware 名称
 */
export type BuiltinMiddlewareName =
  | "summarization" // 1. 总结中间件（上下文压缩）
  | "tool_approval" // 2. 工具审批
  | "pii_redaction" // 3. PII 脱敏
  | "token_limiter" // 4. Token 限制
  | "rate_limiter" // 5. 速率限制
  | "cost_tracker" // 6. 成本追踪
  | "logging" // 7. 日志记录
  | "telemetry"; // 8. 遥测

/**
 * Middleware 信息
 */
export interface MiddlewareInfo {
  /** Middleware 名称 */
  name: string;
  /** 显示名称 */
  displayName: string;
  /** 描述 */
  description: string;
  /** 类型 */
  type: "builtin" | "custom";
  /** 是否启用 */
  enabled: boolean;
  /** 优先级（数字越小越早执行） */
  priority: number;
  /** 配置 Schema */
  configSchema?: MiddlewareConfigSchema;
  /** 当前配置 */
  config?: Record<string, any>;
}

/**
 * Middleware 配置 Schema
 */
export interface MiddlewareConfigSchema {
  /** Schema 类型 */
  type: string;
  /** 属性定义 */
  properties: Record<
    string,
    {
      type: string;
      description?: string;
      default?: any;
      enum?: any[];
      minimum?: number;
      maximum?: number;
    }
  >;
  /** 必需字段 */
  required?: string[];
}

// ============================================================================
// Specific Middleware Configs
// ============================================================================

/**
 * Summarization Middleware 配置
 * 上下文压缩，防止 Token 超限
 */
export interface SummarizationConfig {
  /** 触发总结的 Token 阈值 */
  threshold: number;
  /** 保留最近 N 条消息 */
  keepMessages: number;
  /** 使用的 LLM Provider */
  llmProvider?: string;
  /** 使用的 LLM Model */
  llmModel?: string;
  /** 总结提示词 */
  summaryPrompt?: string;
}

/**
 * Tool Approval Middleware 配置
 * 需要人工审批的工具
 */
export interface ToolApprovalConfig {
  /** 需要审批的工具列表 */
  approvalRequired: string[];
  /** 自动批准的工具列表 */
  autoApprove: string[];
  /** 审批超时时间（秒） */
  approvalTimeout: number;
  /** 超时默认行为 */
  timeoutBehavior: "approve" | "reject";
}

/**
 * PII Redaction Middleware 配置
 * 敏感信息脱敏
 */
export interface PIIRedactionConfig {
  /** 启用的 PII 类型 */
  enabledTypes: Array<
    "email" | "phone" | "ssn" | "credit_card" | "ip_address" | "name"
  >;
  /** 脱敏策略 */
  strategy: "mask" | "remove" | "replace";
  /** 替换文本（strategy=replace 时使用） */
  replacementText?: string;
  /** 是否保留部分信息 */
  partial?: boolean;
}

/**
 * Token Limiter Middleware 配置
 * Token 使用限制
 */
export interface TokenLimiterConfig {
  /** 每次请求最大 Token 数 */
  maxTokensPerRequest: number;
  /** 每小时最大 Token 数 */
  maxTokensPerHour?: number;
  /** 每天最大 Token 数 */
  maxTokensPerDay?: number;
  /** 超限行为 */
  onExceeded: "reject" | "truncate" | "queue";
}

/**
 * Rate Limiter Middleware 配置
 * 请求速率限制
 */
export interface RateLimiterConfig {
  /** 时间窗口（秒） */
  windowSeconds: number;
  /** 窗口内最大请求数 */
  maxRequests: number;
  /** 限流策略 */
  strategy: "fixed_window" | "sliding_window" | "token_bucket";
  /** 超限行为 */
  onExceeded: "reject" | "queue" | "delay";
}

/**
 * Cost Tracker Middleware 配置
 * 成本追踪
 */
export interface CostTrackerConfig {
  /** 是否启用成本追踪 */
  enabled: boolean;
  /** 成本模型 */
  costModel: "token_based" | "time_based" | "custom";
  /** 成本计算方式 */
  pricing?: {
    promptTokenPrice: number; // 每 1K prompt tokens 的价格
    completionTokenPrice: number; // 每 1K completion tokens 的价格
    currency: string; // 货币单位
  };
  /** 预算限制 */
  budget?: {
    daily?: number;
    monthly?: number;
  };
}

/**
 * Logging Middleware 配置
 * 日志记录
 */
export interface LoggingConfig {
  /** 日志级别 */
  level: "debug" | "info" | "warn" | "error";
  /** 是否记录请求 */
  logRequests: boolean;
  /** 是否记录响应 */
  logResponses: boolean;
  /** 是否记录工具调用 */
  logToolCalls: boolean;
  /** 是否记录错误 */
  logErrors: boolean;
  /** 日志格式 */
  format: "json" | "text";
  /** 输出目标 */
  outputs: Array<"console" | "file" | "remote">;
}

/**
 * Telemetry Middleware 配置
 * 遥测数据收集
 */
export interface TelemetryMiddlewareConfig {
  /** 是否启用 */
  enabled: boolean;
  /** 采样率（0-1） */
  samplingRate: number;
  /** 收集的指标 */
  metrics: Array<"latency" | "tokens" | "cost" | "errors" | "tool_calls">;
  /** 导出端点 */
  exportEndpoint?: string;
  /** 导出间隔（秒） */
  exportInterval?: number;
}

// ============================================================================
// Middleware Operations
// ============================================================================

/**
 * Middleware 更新请求
 */
export interface UpdateMiddlewareRequest {
  /** 是否启用 */
  enabled?: boolean;
  /** 优先级 */
  priority?: number;
  /** 配置 */
  config?: Record<string, any>;
}

/**
 * Middleware 统计信息
 */
export interface MiddlewareStats {
  /** Middleware 名称 */
  name: string;
  /** 执行次数 */
  executionCount: number;
  /** 成功次数 */
  successCount: number;
  /** 失败次数 */
  failureCount: number;
  /** 平均执行时间（毫秒） */
  avgExecutionTime: number;
  /** 最后执行时间 */
  lastExecutedAt?: string;
}
