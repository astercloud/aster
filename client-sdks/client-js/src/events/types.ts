/**
 * AsterClient 事件驱动架构 - 三通道事件类型定义
 *
 * 三通道设计：
 * - Progress Channel: 数据流（思考、文本、工具执行）
 * - Control Channel: 审批流（工具审批、暂停/恢复）
 * - Monitor Channel: 治理流（Token、成本、合规）
 */

// ============================================================================
// Channel Types
// ============================================================================

/**
 * 事件通道类型
 */
export type Channel = "progress" | "control" | "monitor";

/**
 * 事件过滤器
 */
export interface EventFilter {
  /** Agent ID 过滤 */
  agentId?: string;
  /** 事件类型过滤 */
  eventTypes?: string[];
}

// ============================================================================
// Progress Channel Events (数据流)
// ============================================================================

/**
 * Progress: 思考事件
 * AI 正在思考的过程
 */
export interface ProgressThinkingEvent {
  type: "thinking";
  channel: "progress";
  data: {
    /** 思考内容 */
    content: string;
  };
}

/**
 * Progress: 文本块事件
 * 流式文本输出
 */
export interface ProgressTextChunkEvent {
  type: "text_chunk";
  channel: "progress";
  data: {
    /** 增量文本 */
    delta: string;
    /** 累积文本 */
    text: string;
  };
}

/**
 * Progress: 工具开始事件
 * 工具调用开始
 */
export interface ProgressToolStartEvent {
  type: "tool_start";
  channel: "progress";
  data: {
    /** 工具名称 */
    toolName: string;
    /** 工具参数 */
    params: any;
  };
}

/**
 * Progress: 工具结束事件
 * 工具调用完成
 */
export interface ProgressToolEndEvent {
  type: "tool_end";
  channel: "progress";
  data: {
    /** 工具名称 */
    toolName: string;
    /** 工具结果 */
    result: any;
  };
}

/**
 * Progress: 完成事件
 * Agent 执行完成
 */
export interface ProgressDoneEvent {
  type: "done";
  channel: "progress";
  data: {
    /** 最终文本 */
    text: string;
  };
}

/**
 * Progress: 错误事件
 * 执行过程中发生错误
 */
export interface ProgressErrorEvent {
  type: "error";
  channel: "progress";
  data: {
    /** 错误信息 */
    error: string;
  };
}

/**
 * Progress Channel 所有事件的联合类型
 */
export type ProgressEvent = ProgressThinkingEvent | ProgressTextChunkEvent | ProgressToolStartEvent | ProgressToolEndEvent | ProgressDoneEvent | ProgressErrorEvent;

// ============================================================================
// Control Channel Events (审批流)
// ============================================================================

/**
 * Control: 工具审批请求事件
 * 需要用户审批工具调用
 */
export interface ControlToolApprovalRequestEvent {
  type: "tool_approval_request";
  channel: "control";
  data: {
    /** 审批 ID */
    approvalId: string;
    /** 工具名称 */
    toolName: string;
    /** 工具参数 */
    params: any;
  };
}

/**
 * Control: 工具审批响应事件
 * 用户审批结果
 */
export interface ControlToolApprovalResponseEvent {
  type: "tool_approval_response";
  channel: "control";
  data: {
    /** 审批 ID */
    approvalId: string;
    /** 是否通过 */
    approved: boolean;
  };
}

/**
 * Control: 暂停事件
 * Agent 执行被暂停
 */
export interface ControlPauseEvent {
  type: "pause";
  channel: "control";
  data: {
    /** 暂停原因 */
    reason: string;
  };
}

/**
 * Control: 恢复事件
 * Agent 执行恢复
 */
export interface ControlResumeEvent {
  type: "resume";
  channel: "control";
  data: {
    /** 恢复时间戳 */
    timestamp: string;
  };
}

/**
 * Control Channel 所有事件的联合类型
 */
export type ControlEvent = ControlToolApprovalRequestEvent | ControlToolApprovalResponseEvent | ControlPauseEvent | ControlResumeEvent;

// ============================================================================
// Monitor Channel Events (治理流)
// ============================================================================

/**
 * Monitor: Token 使用事件
 * 记录 Token 消耗
 */
export interface MonitorTokenUsageEvent {
  type: "token_usage";
  channel: "monitor";
  data: {
    /** Prompt Tokens */
    promptTokens: number;
    /** Completion Tokens */
    completionTokens: number;
    /** 总 Tokens */
    totalTokens: number;
  };
}

/**
 * Monitor: 延迟事件
 * 记录操作延迟
 */
export interface MonitorLatencyEvent {
  type: "latency";
  channel: "monitor";
  data: {
    /** 延迟（毫秒） */
    latencyMs: number;
    /** 操作名称 */
    operation: string;
  };
}

/**
 * Monitor: 成本事件
 * 记录 API 调用成本
 */
export interface MonitorCostEvent {
  type: "cost";
  channel: "monitor";
  data: {
    /** 成本金额 */
    cost: number;
    /** 货币单位 */
    currency: string;
  };
}

/**
 * Monitor: 合规检查事件
 * 记录合规检查结果
 */
export interface MonitorComplianceEvent {
  type: "compliance";
  channel: "monitor";
  data: {
    /** 是否通过 */
    passed: boolean;
    /** 详细信息 */
    details: string;
  };
}

/**
 * Monitor Channel 所有事件的联合类型
 */
export type MonitorEvent = MonitorTokenUsageEvent | MonitorLatencyEvent | MonitorCostEvent | MonitorComplianceEvent;

// ============================================================================
// 事件联合类型
// ============================================================================

/**
 * 所有流式事件的联合类型
 * 包含三个通道的所有事件
 */
export type StreamEvent = ProgressEvent | ControlEvent | MonitorEvent;

/**
 * 事件信封（Envelope）
 * WebSocket 传输时的包装格式
 */
export interface EventEnvelope {
  /** 事件 ID */
  id: string;
  /** 时间戳 */
  timestamp: string;
  /** Agent ID */
  agentId?: string;
  /** 事件内容 */
  event: StreamEvent;
}

// ============================================================================
// 工具类型
// ============================================================================

/**
 * 类型守卫：检查是否为 Progress 事件
 */
export function isProgressEvent(event: StreamEvent): event is ProgressEvent {
  return event.channel === "progress";
}

/**
 * 类型守卫：检查是否为 Control 事件
 */
export function isControlEvent(event: StreamEvent): event is ControlEvent {
  return event.channel === "control";
}

/**
 * 类型守卫：检查是否为 Monitor 事件
 */
export function isMonitorEvent(event: StreamEvent): event is MonitorEvent {
  return event.channel === "monitor";
}

/**
 * 类型守卫：检查具体事件类型
 */
export function isEventType<T extends StreamEvent["type"]>(event: StreamEvent, type: T): event is Extract<StreamEvent, { type: T }> {
  return event.type === type;
}
