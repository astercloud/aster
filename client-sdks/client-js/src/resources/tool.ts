/**
 * Tool 资源类
 * 管理内置工具和长时运行工具
 */

import { BaseResource, ClientOptions } from "./base";
import {
  ToolInfo,
  ToolExecutionRequest,
  ToolExecutionResponse,
  AsyncToolExecutionResponse,
  TaskProgress,
  ToolStats,
  ToolUsageReport,
} from "../types/tool";

/**
 * Tool 资源类
 */
export class ToolResource extends BaseResource {
  constructor(options: ClientOptions) {
    super(options);
  }

  // ==========================================================================
  // Tool Management
  // ==========================================================================

  /**
   * 创建自定义工具
   * @param tool 工具定义
   * @returns 工具信息
   */
  async create(tool: {
    name: string;
    type: string;
    schema: Record<string, any>;
    description?: string;
  }): Promise<ToolInfo> {
    return this.request<ToolInfo>("/v1/tools", {
      method: "POST",
      body: tool,
    });
  }

  /**
   * 列出所有工具
   * @param filter 过滤条件
   * @returns 工具列表
   */
  async list(filter?: {
    type?: "builtin" | "custom" | "mcp";
    category?: string;
    enabled?: boolean;
  }): Promise<ToolInfo[]> {
    const result = await this.request<{ tools: ToolInfo[] }>("/v1/tools", {
      params: filter,
    });
    return result.tools;
  }

  /**
   * 获取工具详情
   * @param toolName 工具名称
   * @returns 工具信息
   */
  async get(toolName: string): Promise<ToolInfo> {
    return this.request<ToolInfo>(`/v1/tools/${toolName}`);
  }

  /**
   * 启用工具
   * @param toolName 工具名称
   */
  async enable(toolName: string): Promise<ToolInfo> {
    return this.request<ToolInfo>(`/v1/tools/${toolName}/enable`, {
      method: "POST",
    });
  }

  /**
   * 禁用工具
   * @param toolName 工具名称
   */
  async disable(toolName: string): Promise<ToolInfo> {
    return this.request<ToolInfo>(`/v1/tools/${toolName}/disable`, {
      method: "POST",
    });
  }

  /**
   * 删除自定义工具
   * @param toolId 工具 ID 或名称
   */
  async delete(toolId: string): Promise<void> {
    await this.request(`/v1/tools/${toolId}`, {
      method: "DELETE",
    });
  }

  // ==========================================================================
  // Tool Execution
  // ==========================================================================

  /**
   * 执行工具（同步）
   * @param toolId 工具 ID
   * @param input 参数
   * @returns 执行结果
   */
  async execute(toolId: string, input: Record<string, any>): Promise<any> {
    return this.request(`/v1/tools/${toolId}/execute`, {
      method: "POST",
      body: { input },
    });
  }

  /**
   * 执行工具（异步，用于长时运行工具）
   * @param toolName 工具名称
   * @param params 参数
   * @returns 任务信息
   */
  async executeAsync(
    toolName: string,
    params: Record<string, any>,
  ): Promise<AsyncToolExecutionResponse> {
    return this.request<AsyncToolExecutionResponse>("/v1/tools/execute", {
      method: "POST",
      body: {
        toolName,
        params,
        async: true,
      },
    });
  }

  /**
   * 执行工具（通用方法）
   * @param request 执行请求
   * @returns 执行结果或任务信息
   */
  async executeWithOptions(
    request: ToolExecutionRequest,
  ): Promise<ToolExecutionResponse | AsyncToolExecutionResponse> {
    return this.request("/v1/tools/execute", {
      method: "POST",
      body: request,
      timeout: request.timeout,
    });
  }

  // ==========================================================================
  // Long-Running Task Management
  // ==========================================================================

  /**
   * 获取任务进度
   * @param taskId 任务 ID
   * @returns 任务进度信息
   */
  async getTaskProgress(taskId: string): Promise<TaskProgress> {
    return this.request<TaskProgress>(`/v1/tools/tasks/${taskId}`);
  }

  /**
   * 列出所有任务
   * @param filter 过滤条件
   * @returns 任务列表
   */
  async listTasks(filter?: {
    status?: "pending" | "running" | "completed" | "failed" | "cancelled";
    toolName?: string;
  }): Promise<TaskProgress[]> {
    const result = await this.request<{ tasks: TaskProgress[] }>(
      "/v1/tools/tasks",
      { params: filter },
    );
    return result.tasks;
  }

  /**
   * 取消任务
   * @param taskId 任务 ID
   */
  async cancelTask(taskId: string): Promise<void> {
    await this.request(`/v1/tools/tasks/${taskId}/cancel`, {
      method: "POST",
    });
  }

  /**
   * 等待任务完成
   * @param taskId 任务 ID
   * @param options 轮询选项
   * @returns 最终任务状态
   */
  async waitForTask(
    taskId: string,
    options?: {
      pollInterval?: number; // 轮询间隔（毫秒），默认 1000
      timeout?: number; // 超时时间（毫秒），默认 300000 (5分钟)
    },
  ): Promise<TaskProgress> {
    const pollInterval = options?.pollInterval ?? 1000;
    const timeout = options?.timeout ?? 300000;
    const startTime = Date.now();

    while (true) {
      const task = await this.getTaskProgress(taskId);

      // 检查是否完成
      if (
        task.status === "completed" ||
        task.status === "failed" ||
        task.status === "cancelled"
      ) {
        return task;
      }

      // 检查超时
      if (Date.now() - startTime > timeout) {
        throw new Error(
          `Task timeout after ${timeout}ms. Current status: ${task.status}`,
        );
      }

      // 等待后继续轮询
      await new Promise((resolve) => setTimeout(resolve, pollInterval));
    }
  }

  // ==========================================================================
  // Tool Statistics
  // ==========================================================================

  /**
   * 获取工具统计信息
   * @param toolName 工具名称
   * @returns 统计数据
   */
  async getStats(toolName: string): Promise<ToolStats> {
    return this.request<ToolStats>(`/v1/tools/${toolName}/stats`);
  }

  /**
   * 获取所有工具统计信息
   * @returns 统计数据列表
   */
  async getAllStats(): Promise<ToolStats[]> {
    const result = await this.request<{ stats: ToolStats[] }>(
      "/v1/tools/stats",
    );
    return result.stats;
  }

  /**
   * 获取工具使用报告
   * @param timeRange 时间范围
   * @returns 使用报告
   */
  async getUsageReport(timeRange?: {
    start: string;
    end: string;
  }): Promise<ToolUsageReport> {
    return this.request<ToolUsageReport>("/v1/tools/usage-report", {
      params: timeRange,
    });
  }

  // ==========================================================================
  // Batch Operations
  // ==========================================================================

  /**
   * 批量启用工具
   * @param toolNames 工具名称列表
   */
  async enableBatch(toolNames: string[]): Promise<void> {
    await this.request("/v1/tools/batch/enable", {
      method: "POST",
      body: { toolNames },
    });
  }

  /**
   * 批量禁用工具
   * @param toolNames 工具名称列表
   */
  async disableBatch(toolNames: string[]): Promise<void> {
    await this.request("/v1/tools/batch/disable", {
      method: "POST",
      body: { toolNames },
    });
  }
}
