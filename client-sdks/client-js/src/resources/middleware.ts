/**
 * Middleware 资源类
 * 管理洋葱模型中间件系统
 */

import { BaseResource, ClientOptions } from "./base";
import {
  MiddlewareInfo,
  UpdateMiddlewareRequest,
  MiddlewareStats,
} from "../types/middleware";

/**
 * Middleware 资源类
 */
export class MiddlewareResource extends BaseResource {
  constructor(options: ClientOptions) {
    super(options);
  }

  // ==========================================================================
  // Middleware Management
  // ==========================================================================

  /**
   * 创建自定义 Middleware
   * @param middleware Middleware 定义
   * @returns Middleware 信息
   */
  async create(middleware: {
    name: string;
    type: string;
    priority: number;
    config?: Record<string, any>;
  }): Promise<MiddlewareInfo> {
    return this.request<MiddlewareInfo>("/v1/middlewares", {
      method: "POST",
      body: middleware,
    });
  }

  /**
   * 列出所有 Middleware
   * @returns Middleware 列表
   */
  async list(): Promise<MiddlewareInfo[]> {
    const result = await this.request<{ middlewares: MiddlewareInfo[] }>(
      "/v1/middlewares",
    );
    return result.middlewares;
  }

  /**
   * 获取 Middleware 详情
   * @param name Middleware 名称
   * @returns Middleware 信息
   */
  async get(name: string): Promise<MiddlewareInfo> {
    return this.request<MiddlewareInfo>(`/v1/middlewares/${name}`);
  }

  /**
   * 更新 Middleware 配置
   * @param name Middleware 名称
   * @param updates 更新内容
   * @returns 更新后的 Middleware 信息
   */
  async update(
    name: string,
    updates: UpdateMiddlewareRequest,
  ): Promise<MiddlewareInfo> {
    return this.request<MiddlewareInfo>(`/v1/middlewares/${name}`, {
      method: "PATCH",
      body: updates,
    });
  }

  /**
   * 启用 Middleware
   * @param nameOrId Middleware 名称或 ID
   */
  async enable(nameOrId: string): Promise<MiddlewareInfo> {
    return this.update(nameOrId, { enabled: true });
  }

  /**
   * 禁用 Middleware
   * @param nameOrId Middleware 名称或 ID
   */
  async disable(nameOrId: string): Promise<MiddlewareInfo> {
    return this.update(nameOrId, { enabled: false });
  }

  /**
   * 删除自定义 Middleware
   * @param nameOrId Middleware 名称或 ID
   */
  async delete(nameOrId: string): Promise<void> {
    await this.request(`/v1/middlewares/${nameOrId}`, {
      method: "DELETE",
    });
  }

  /**
   * 更新 Middleware 配置（便捷方法）
   * @param name Middleware 名称
   * @param config 配置
   */
  async updateConfig(
    name: string,
    config: Record<string, any>,
  ): Promise<MiddlewareInfo> {
    return this.update(name, { config });
  }

  /**
   * 更新 Middleware 优先级
   * @param name Middleware 名称
   * @param priority 优先级（数字越小越早执行）
   */
  async updatePriority(
    name: string,
    priority: number,
  ): Promise<MiddlewareInfo> {
    return this.update(name, { priority });
  }

  // ==========================================================================
  // Middleware Statistics
  // ==========================================================================

  /**
   * 获取 Middleware 统计信息
   * @param name Middleware 名称
   * @returns 统计数据
   */
  async getStats(name: string): Promise<MiddlewareStats> {
    return this.request<MiddlewareStats>(`/v1/middlewares/${name}/stats`);
  }

  /**
   * 获取所有 Middleware 统计信息
   * @returns 统计数据列表
   */
  async getAllStats(): Promise<MiddlewareStats[]> {
    const result = await this.request<{ stats: MiddlewareStats[] }>(
      "/v1/middlewares/stats",
    );
    return result.stats;
  }

  // ==========================================================================
  // Batch Operations
  // ==========================================================================

  /**
   * 批量启用 Middlewares
   * @param names Middleware 名称列表
   */
  async enableBatch(names: string[]): Promise<void> {
    await this.request("/v1/middlewares/batch/enable", {
      method: "POST",
      body: { names },
    });
  }

  /**
   * 批量禁用 Middlewares
   * @param names Middleware 名称列表
   */
  async disableBatch(names: string[]): Promise<void> {
    await this.request("/v1/middlewares/batch/disable", {
      method: "POST",
      body: { names },
    });
  }

  // ==========================================================================
  // Pipeline Management
  // ==========================================================================

  /**
   * 获取当前 Middleware 执行顺序
   * @returns Middleware 名称列表（按执行顺序）
   */
  async getExecutionOrder(): Promise<string[]> {
    const result = await this.request<{ order: string[] }>(
      "/v1/middlewares/execution-order",
    );
    return result.order;
  }

  /**
   * 设置 Middleware 执行顺序
   * @param order Middleware 名称列表（按执行顺序）
   */
  async setExecutionOrder(order: string[]): Promise<void> {
    await this.request("/v1/middlewares/execution-order", {
      method: "PUT",
      body: { order },
    });
  }

  /**
   * 重置 Middleware 顺序为默认值
   */
  async resetExecutionOrder(): Promise<void> {
    await this.request("/v1/middlewares/execution-order/reset", {
      method: "POST",
    });
  }
}
