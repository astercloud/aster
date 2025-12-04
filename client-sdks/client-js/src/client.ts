/**
 * aster 主客户端类
 * 整合所有资源模块
 */

import { ClientOptions } from "./resources/base";
import { AgentResource } from "./resources/agent";
import { MemoryResource } from "./resources/memory";
import { SessionResource } from "./resources/session";
import { WorkflowResource } from "./resources/workflow";
import { MCPResource } from "./resources/mcp";
import { MiddlewareResource } from "./resources/middleware";
import { ToolResource } from "./resources/tool";
import { TelemetryResource } from "./resources/telemetry";
import { EvalResource } from "./resources/eval";
import { SystemResource } from "./resources/system";

/**
 * aster 客户端配置
 */
export interface asterConfig {
  /** 基础 URL */
  baseUrl: string;
  /** API 密钥 */
  apiKey?: string;
  /** 超时时间（毫秒） */
  timeout?: number;
  /** 重试配置 */
  retry?: {
    maxRetries?: number;
    retryDelay?: number;
  };
}

/**
 * aster 主客户端
 *
 * @example
 * ```typescript
 * const client = new aster({
 *   baseUrl: 'http://localhost:8080',
 *   apiKey: 'your-api-key'
 * });
 *
 * // 使用 Memory
 * await client.memory.working.set('key', { data: 'value' });
 *
 * // 使用 Workflow
 * const wf = await client.workflows.create({ type: 'parallel', ... });
 *
 * // 使用 MCP
 * await client.mcp.call('server-id', 'tool-name', { params });
 * ```
 */
export class aster {
  /** 基础配置 */
  private readonly config: ClientOptions;

  /** Agent 资源 */
  public readonly agents: AgentResource;

  /** Memory 资源 */
  public readonly memory: MemoryResource;

  /** Session 资源 */
  public readonly sessions: SessionResource;

  /** Workflow 资源 */
  public readonly workflows: WorkflowResource;

  /** MCP 资源 */
  public readonly mcp: MCPResource;

  /** Middleware 资源 */
  public readonly middleware: MiddlewareResource;

  /** Tool 资源 */
  public readonly tools: ToolResource;

  /** Telemetry 资源 */
  public readonly telemetry: TelemetryResource;

  /** Eval 资源 */
  public readonly evals: EvalResource;

  /** System 资源 */
  public readonly system: SystemResource;

  /**
   * 创建 aster 客户端实例
   * @param config 客户端配置
   */
  constructor(config: asterConfig) {
    this.config = {
      baseUrl: config.baseUrl,
      apiKey: config.apiKey,
      timeout: config.timeout,
      retry: config.retry,
    };

    // 初始化所有资源模块
    this.agents = new AgentResource(this.config);
    this.memory = new MemoryResource(this.config);
    this.sessions = new SessionResource(this.config);
    this.workflows = new WorkflowResource(this.config);
    this.mcp = new MCPResource(this.config);
    this.middleware = new MiddlewareResource(this.config);
    this.tools = new ToolResource(this.config);
    this.telemetry = new TelemetryResource(this.config);
    this.evals = new EvalResource(this.config);
    this.system = new SystemResource(this.config);
  }

  // ==========================================================================
  // Convenience Methods (快捷方法)
  // ==========================================================================

  /**
   * 健康检查
   * @returns 健康状态
   */
  async healthCheck() {
    return this.telemetry.healthCheck();
  }

  /**
   * 获取系统状态
   * @returns 状态信息
   */
  async getStatus() {
    return this.telemetry.getStatus();
  }

  /**
   * 获取使用统计
   * @param timeRange 时间范围
   * @returns 使用统计
   */
  async getUsageStatistics(timeRange?: { start: string; end: string }) {
    return this.telemetry.getUsageStatistics(timeRange);
  }

  /**
   * 获取性能指标
   * @param timeRange 时间范围
   * @returns 性能指标
   */
  async getPerformanceMetrics(timeRange?: { start: string; end: string }) {
    return this.telemetry.getPerformanceMetrics(timeRange);
  }
}

/**
 * 创建 aster 客户端的便捷函数
 * @param config 客户端配置
 * @returns aster 实例
 */
export function createClient(config: asterConfig): aster {
  return new aster(config);
}
