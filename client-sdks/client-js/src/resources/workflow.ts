/**
 * Workflow 资源类
 * 管理工作流：Parallel、Sequential、Loop
 */

import { BaseResource, ClientOptions } from "./base";
import {
  WorkflowDefinition,
  WorkflowInfo,
  WorkflowFilter,
  WorkflowRun,
  RunDetails,
  ExecuteWorkflowRequest,
  SuspendWorkflowRequest,
  ResumeWorkflowRequest,
  CancelWorkflowRequest,
  WorkflowRunFilter,
  WorkflowValidationResult,
  UpdateWorkflowRequest,
} from "../types/workflow";
import { PaginatedResponse } from "../types/session";

/**
 * Workflow 资源类
 */
export class WorkflowResource extends BaseResource {
  constructor(options: ClientOptions) {
    super(options);
  }

  // ==========================================================================
  // Workflow CRUD
  // ==========================================================================

  /**
   * 创建 Workflow
   * @param definition Workflow 定义
   * @returns Workflow 信息
   */
  async create(definition: WorkflowDefinition): Promise<WorkflowInfo> {
    return this.request<WorkflowInfo>("/v1/workflows", {
      method: "POST",
      body: definition,
    });
  }

  /**
   * 获取 Workflow 详情
   * @param workflowId Workflow ID
   * @returns Workflow 信息
   */
  async get(workflowId: string): Promise<WorkflowInfo> {
    return this.request<WorkflowInfo>(`/v1/workflows/${workflowId}`);
  }

  /**
   * 列出 Workflows
   * @param filter 过滤条件
   * @returns Workflow 列表
   */
  async list(
    filter?: WorkflowFilter,
  ): Promise<PaginatedResponse<WorkflowInfo>> {
    return this.request<PaginatedResponse<WorkflowInfo>>("/v1/workflows", {
      params: filter,
    });
  }

  /**
   * 更新 Workflow
   * @param workflowId Workflow ID
   * @param updates 更新内容
   * @returns 更新后的 Workflow
   */
  async update(
    workflowId: string,
    updates: UpdateWorkflowRequest,
  ): Promise<WorkflowInfo> {
    return this.request<WorkflowInfo>(`/v1/workflows/${workflowId}`, {
      method: "PATCH",
      body: updates,
    });
  }

  /**
   * 删除 Workflow
   * @param workflowId Workflow ID
   */
  async delete(workflowId: string): Promise<void> {
    await this.request(`/v1/workflows/${workflowId}`, {
      method: "DELETE",
    });
  }

  // ==========================================================================
  // Workflow Execution
  // ==========================================================================

  /**
   * 执行 Workflow
   * @param workflowId Workflow ID
   * @param request 执行请求
   * @returns Workflow 执行记录
   */
  async execute(
    workflowId: string,
    request: ExecuteWorkflowRequest,
  ): Promise<WorkflowRun> {
    return this.request<WorkflowRun>(`/v1/workflows/${workflowId}/execute`, {
      method: "POST",
      body: request,
    });
  }

  /**
   * 暂停 Workflow 执行
   * @param workflowId Workflow ID
   * @param request 暂停请求
   */
  async suspend(
    workflowId: string,
    request: SuspendWorkflowRequest,
  ): Promise<void> {
    await this.request(`/v1/workflows/${workflowId}/suspend`, {
      method: "POST",
      body: request,
    });
  }

  /**
   * 恢复 Workflow 执行
   * @param workflowId Workflow ID
   * @param request 恢复请求
   */
  async resume(
    workflowId: string,
    request: ResumeWorkflowRequest,
  ): Promise<void> {
    await this.request(`/v1/workflows/${workflowId}/resume`, {
      method: "POST",
      body: request,
    });
  }

  /**
   * 取消 Workflow 执行
   * @param workflowId Workflow ID
   * @param request 取消请求
   */
  async cancel(
    workflowId: string,
    request: CancelWorkflowRequest,
  ): Promise<void> {
    await this.request(`/v1/workflows/${workflowId}/cancel`, {
      method: "POST",
      body: request,
    });
  }

  // ==========================================================================
  // Workflow Run Management
  // ==========================================================================

  /**
   * 获取 Workflow 执行历史
   * @param workflowId Workflow ID
   * @param filter 过滤条件
   * @returns 执行记录列表
   */
  async getRuns(
    workflowId: string,
    filter?: WorkflowRunFilter,
  ): Promise<PaginatedResponse<WorkflowRun>> {
    return this.request<PaginatedResponse<WorkflowRun>>(
      `/v1/workflows/${workflowId}/executions`,
      { params: filter },
    );
  }

  /**
   * 列出执行记录（别名方法）
   * @param workflowId Workflow ID
   * @param filter 过滤条件
   * @returns 执行记录数组
   */
  async listExecutions(
    workflowId: string,
    filter?: WorkflowRunFilter,
  ): Promise<WorkflowRun[]> {
    const result = await this.getRuns(workflowId, filter);
    return result.items || [];
  }

  /**
   * 获取单个执行的详细信息
   * @param workflowId Workflow ID
   * @param runId Run ID
   * @returns 执行详情
   */
  async getRunDetails(workflowId: string, runId: string): Promise<RunDetails> {
    return this.request<RunDetails>(
      `/v1/workflows/${workflowId}/runs/${runId}`,
    );
  }

  /**
   * 获取执行的实时状态
   * @param workflowId Workflow ID
   * @param runId Run ID
   * @returns 执行状态
   */
  async getRunStatus(workflowId: string, runId: string): Promise<WorkflowRun> {
    return this.request<WorkflowRun>(
      `/v1/workflows/${workflowId}/runs/${runId}/status`,
    );
  }

  // ==========================================================================
  // Workflow Validation
  // ==========================================================================

  /**
   * 验证 Workflow 定义
   * @param definition Workflow 定义
   * @returns 验证结果
   */
  async validate(
    definition: WorkflowDefinition,
  ): Promise<WorkflowValidationResult> {
    return this.request<WorkflowValidationResult>("/v1/workflows/validate", {
      method: "POST",
      body: definition,
    });
  }

  // ==========================================================================
  // Batch Operations
  // ==========================================================================

  /**
   * 批量删除 Workflows
   * @param workflowIds Workflow ID 列表
   */
  async deleteBatch(workflowIds: string[]): Promise<void> {
    await this.request("/v1/workflows/batch", {
      method: "DELETE",
      body: { workflowIds },
    });
  }

  /**
   * 批量归档 Workflows
   * @param workflowIds Workflow ID 列表
   */
  async archiveBatch(workflowIds: string[]): Promise<void> {
    await this.request("/v1/workflows/batch/archive", {
      method: "POST",
      body: { workflowIds },
    });
  }

  // ==========================================================================
  // Helper Methods
  // ==========================================================================

  /**
   * 等待 Workflow 执行完成
   * @param workflowId Workflow ID
   * @param runId Run ID
   * @param options 轮询选项
   * @returns 最终执行结果
   */
  async waitForCompletion(
    workflowId: string,
    runId: string,
    options?: {
      pollInterval?: number; // 轮询间隔（毫秒），默认 1000
      timeout?: number; // 超时时间（毫秒），默认 300000 (5分钟)
    },
  ): Promise<WorkflowRun> {
    const pollInterval = options?.pollInterval ?? 1000;
    const timeout = options?.timeout ?? 300000;
    const startTime = Date.now();

    while (true) {
      const run = await this.getRunStatus(workflowId, runId);

      // 检查是否完成
      if (
        run.status === "completed" ||
        run.status === "failed" ||
        run.status === "cancelled"
      ) {
        return run;
      }

      // 检查超时
      if (Date.now() - startTime > timeout) {
        throw new Error(
          `Workflow execution timeout after ${timeout}ms. Current status: ${run.status}`,
        );
      }

      // 等待后继续轮询
      await new Promise((resolve) => setTimeout(resolve, pollInterval));
    }
  }
}
