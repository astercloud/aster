/**
 * Session 资源类
 * 管理会话生命周期、消息历史、断点恢复
 */

import { BaseResource, ClientOptions } from "./base";
import {
  SessionConfig,
  SessionInfo,
  SessionFilter,
  SessionStatus,
  Message,
  Pagination,
  PaginatedResponse,
  Checkpoint,
  ResumeOptions,
  SessionStats,
  UpdateSessionRequest,
  ExportOptions,
  ExportResult,
} from "../types/session";

/**
 * Session 资源类
 */
export class SessionResource extends BaseResource {
  constructor(options: ClientOptions) {
    super(options);
  }

  // ==========================================================================
  // Session CRUD
  // ==========================================================================

  /**
   * 创建 Session
   * @param config Session 配置
   * @returns Session 信息
   */
  async create(config: SessionConfig): Promise<SessionInfo> {
    return this.request<SessionInfo>("/v1/sessions", {
      method: "POST",
      body: config,
    });
  }

  /**
   * 获取 Session 详情
   * @param sessionId Session ID
   * @returns Session 信息
   */
  async get(sessionId: string): Promise<SessionInfo> {
    return this.request<SessionInfo>(`/v1/sessions/${sessionId}`);
  }

  /**
   * 列出 Sessions
   * @param filter 过滤条件
   * @returns Session 列表
   */
  async list(filter?: SessionFilter): Promise<PaginatedResponse<SessionInfo>> {
    return this.request<PaginatedResponse<SessionInfo>>("/v1/sessions", {
      params: filter,
    });
  }

  /**
   * 更新 Session
   * @param sessionId Session ID
   * @param updates 更新内容
   */
  async update(
    sessionId: string,
    updates: UpdateSessionRequest,
  ): Promise<SessionInfo> {
    return this.request<SessionInfo>(`/v1/sessions/${sessionId}`, {
      method: "PATCH",
      body: updates,
    });
  }

  /**
   * 删除 Session
   * @param sessionId Session ID
   */
  async delete(sessionId: string): Promise<void> {
    await this.request(`/v1/sessions/${sessionId}`, {
      method: "DELETE",
    });
  }

  // ==========================================================================
  // Message Management
  // ==========================================================================

  /**
   * 获取 Session 消息列表
   * @param sessionId Session ID
   * @param pagination 分页参数
   * @returns 消息列表
   */
  async getMessages(
    sessionId: string,
    pagination?: Pagination,
  ): Promise<PaginatedResponse<Message>> {
    return this.request<PaginatedResponse<Message>>(
      `/v1/sessions/${sessionId}/messages`,
      { params: pagination },
    );
  }

  /**
   * 添加消息到 Session
   * @param sessionId Session ID
   * @param message 消息内容
   * @returns 创建的消息
   */
  async addMessage(
    sessionId: string,
    message: Omit<Message, "id" | "timestamp">,
  ): Promise<Message> {
    return this.request<Message>(`/v1/sessions/${sessionId}/messages`, {
      method: "POST",
      body: message,
    });
  }

  /**
   * 获取单条消息
   * @param sessionId Session ID
   * @param messageId Message ID
   * @returns 消息详情
   */
  async getMessage(sessionId: string, messageId: string): Promise<Message> {
    return this.request<Message>(
      `/v1/sessions/${sessionId}/messages/${messageId}`,
    );
  }

  /**
   * 删除消息
   * @param sessionId Session ID
   * @param messageId Message ID
   */
  async deleteMessage(sessionId: string, messageId: string): Promise<void> {
    await this.request(`/v1/sessions/${sessionId}/messages/${messageId}`, {
      method: "DELETE",
    });
  }

  // ==========================================================================
  // Checkpoint Management (断点恢复)
  // ==========================================================================

  /**
   * 获取 Session 的所有 Checkpoints
   * @param sessionId Session ID
   * @returns Checkpoint 列表
   */
  async getCheckpoints(sessionId: string): Promise<Checkpoint[]> {
    const result = await this.request<{ checkpoints: Checkpoint[] }>(
      `/v1/sessions/${sessionId}/checkpoints`,
    );
    return result.checkpoints;
  }

  /**
   * 获取单个 Checkpoint
   * @param sessionId Session ID
   * @param checkpointId Checkpoint ID
   * @returns Checkpoint 详情
   */
  async getCheckpoint(
    sessionId: string,
    checkpointId: string,
  ): Promise<Checkpoint> {
    return this.request<Checkpoint>(
      `/v1/sessions/${sessionId}/checkpoints/${checkpointId}`,
    );
  }

  /**
   * 从 Checkpoint 恢复 Session
   * @param sessionId Session ID
   * @param options 恢复选项
   */
  async resume(
    sessionId: string,
    options?: ResumeOptions,
  ): Promise<SessionInfo> {
    return this.request<SessionInfo>(`/v1/sessions/${sessionId}/resume`, {
      method: "POST",
      body: options,
    });
  }

  /**
   * 创建手动 Checkpoint
   * @param sessionId Session ID
   * @param label Checkpoint 标签（可选）
   * @returns 创建的 Checkpoint
   */
  async createCheckpoint(
    sessionId: string,
    label?: string,
  ): Promise<Checkpoint> {
    return this.request<Checkpoint>(`/v1/sessions/${sessionId}/checkpoints`, {
      method: "POST",
      body: { label },
    });
  }

  // ==========================================================================
  // Session Statistics
  // ==========================================================================

  /**
   * 获取 Session 统计信息
   * @param sessionId Session ID
   * @returns 统计数据
   */
  async getStats(sessionId: string): Promise<SessionStats> {
    return this.request<SessionStats>(`/v1/sessions/${sessionId}/stats`);
  }

  // ==========================================================================
  // Session State Management
  // ==========================================================================

  /**
   * 暂停 Session
   * @param sessionId Session ID
   */
  async pause(sessionId: string): Promise<SessionInfo> {
    return this.update(sessionId, { status: "paused" });
  }

  /**
   * 恢复 Session（从暂停状态）
   * @param sessionId Session ID
   */
  async activate(sessionId: string): Promise<SessionInfo> {
    return this.update(sessionId, { status: "active" });
  }

  /**
   * 完成 Session
   * @param sessionId Session ID
   */
  async complete(sessionId: string): Promise<SessionInfo> {
    return this.update(sessionId, { status: "completed" });
  }

  /**
   * 归档 Session
   * @param sessionId Session ID
   */
  async archive(sessionId: string): Promise<SessionInfo> {
    return this.update(sessionId, { status: "archived" });
  }

  // ==========================================================================
  // Session Export
  // ==========================================================================

  /**
   * 导出 Session
   * @param sessionId Session ID
   * @param options 导出选项
   * @returns 导出结果
   */
  async export(
    sessionId: string,
    options: ExportOptions,
  ): Promise<ExportResult> {
    return this.request<ExportResult>(`/v1/sessions/${sessionId}/export`, {
      method: "POST",
      body: options,
    });
  }

  // ==========================================================================
  // Batch Operations
  // ==========================================================================

  /**
   * 批量删除 Sessions
   * @param sessionIds Session ID 列表
   */
  async deleteBatch(sessionIds: string[]): Promise<void> {
    await this.request("/v1/sessions/batch", {
      method: "DELETE",
      body: { sessionIds },
    });
  }

  /**
   * 批量归档 Sessions
   * @param sessionIds Session ID 列表
   */
  async archiveBatch(sessionIds: string[]): Promise<void> {
    await this.request("/v1/sessions/batch/archive", {
      method: "POST",
      body: { sessionIds },
    });
  }
}
