// ============================================================================
// 事件系统导出
// ============================================================================

// 事件类型
export * from "./events/types";

// WebSocket 客户端
export { WebSocketClient, WebSocketState } from "./transport/websocket";
export type { WebSocketClientOptions } from "./transport/websocket";

// 事件订阅
export { EventSubscription, SubscriptionManager } from "./events/subscription";

// Agent 类型
export * from "./types/agent";

// Memory 类型
export * from "./types/memory";

// Session 类型
export * from "./types/session";

// Workflow 类型
export * from "./types/workflow";

// MCP 类型
export * from "./types/mcp";

// Middleware 类型
export * from "./types/middleware";

// Tool 类型
export * from "./types/tool";

// Telemetry 类型
export * from "./types/telemetry";

// Eval 类型
export * from "./types/eval";

// 资源类
export { BaseResource } from "./resources/base";
export type { ClientOptions, RequestOptions, RetryOptions } from "./resources/base";
export { AgentResource } from "./resources/agent";
export { MemoryResource } from "./resources/memory";
export { SessionResource } from "./resources/session";
export { WorkflowResource } from "./resources/workflow";
export { MCPResource } from "./resources/mcp";
export { MiddlewareResource } from "./resources/middleware";
export { ToolResource } from "./resources/tool";
export { TelemetryResource } from "./resources/telemetry";
export { EvalResource } from "./resources/eval";

// 主客户端类
export { aster, createClient } from "./client";
export type { asterConfig } from "./client";

// 向后兼容的别名
export { aster as AsterClient } from "./client";
export type { asterConfig as AsterClientConfig } from "./client";

// ============================================================================
// 原有接口（向后兼容）
// ============================================================================

export interface ChatRequest {
  template_id: string;
  input: string;
  routing_profile?: string;
  model_config?: {
    provider?: string;
    model?: string;
    api_key?: string;
  };
  sandbox?: {
    kind?: string;
    work_dir?: string;
  };
  middlewares?: string[];
  metadata?: Record<string, unknown>;
}

export interface ChatResponse {
  agent_id?: string;
  text?: string;
  status: string;
  error_message?: string | null;
}

/**
 * 旧版客户端选项（v0.1.0）
 * @deprecated 请使用 AsterClient（从 './client' 导入）
 */
export interface LegacyClientOptions {
  baseUrl: string;
  fetchImpl?: typeof fetch;
}
