/**
 * Agent 资源类
 * Agent 完整管理功能
 */

import { BaseResource, ClientOptions } from "./base";
import {
  AgentInfo,
  AgentFilter,
  PaginatedAgentResponse,
  CreateAgentRequest,
  UpdateAgentRequest,
  ChatRequest,
  ChatResponse,
  StreamChatEvent,
  AgentTemplate,
  AgentStats,
  AgentValidationResult,
  AgentStatus,
} from "../types/agent";

/**
 * Agent 资源类
 */
export class AgentResource extends BaseResource {
  constructor(options: ClientOptions) {
    super(options);
  }

  // ==========================================================================
  // Agent CRUD
  // ==========================================================================

  /**
   * 创建 Agent
   * @param request 创建请求
   * @returns Agent 信息
   */
  async create(request: CreateAgentRequest): Promise<AgentInfo> {
    return this.request<AgentInfo>("/v1/agents", {
      method: "POST",
      body: request,
    });
  }

  /**
   * 获取 Agent 详情
   * @param agentId Agent ID
   * @returns Agent 信息
   */
  async get(agentId: string): Promise<AgentInfo> {
    return this.request<AgentInfo>(`/v1/agents/${agentId}`);
  }

  /**
   * 列出所有 Agents
   * @param filter 过滤条件
   * @returns Agent 列表
   */
  async list(filter?: AgentFilter): Promise<PaginatedAgentResponse> {
    return this.request<PaginatedAgentResponse>("/v1/agents", {
      params: filter,
    });
  }

  /**
   * 更新 Agent
   * @param agentId Agent ID
   * @param updates 更新内容
   * @returns 更新后的 Agent
   */
  async update(
    agentId: string,
    updates: UpdateAgentRequest,
  ): Promise<AgentInfo> {
    return this.request<AgentInfo>(`/v1/agents/${agentId}`, {
      method: "PATCH",
      body: updates,
    });
  }

  /**
   * 删除 Agent
   * @param agentId Agent ID
   */
  async delete(agentId: string): Promise<void> {
    await this.request(`/v1/agents/${agentId}`, {
      method: "DELETE",
    });
  }

  // ==========================================================================
  // Agent Status Management
  // ==========================================================================

  /**
   * 激活 Agent
   * @param agentId Agent ID
   * @returns 更新后的 Agent
   */
  async activate(agentId: string): Promise<AgentInfo> {
    return this.updateStatus(agentId, "active");
  }

  /**
   * 禁用 Agent
   * @param agentId Agent ID
   * @returns 更新后的 Agent
   */
  async disable(agentId: string): Promise<AgentInfo> {
    return this.updateStatus(agentId, "disabled");
  }

  /**
   * 归档 Agent
   * @param agentId Agent ID
   * @returns 更新后的 Agent
   */
  async archive(agentId: string): Promise<AgentInfo> {
    return this.updateStatus(agentId, "archived");
  }

  /**
   * 更新 Agent 状态
   * @param agentId Agent ID
   * @param status 新状态
   * @returns 更新后的 Agent
   */
  private async updateStatus(
    agentId: string,
    status: AgentStatus,
  ): Promise<AgentInfo> {
    return this.update(agentId, { status });
  }

  // ==========================================================================
  // Agent Chat
  // ==========================================================================

  /**
   * 与 Agent 对话（同步）
   * @param agentId Agent ID
   * @param request Chat 请求
   * @returns Chat 响应
   */
  async chat(agentId: string, request: ChatRequest): Promise<ChatResponse> {
    return this.request<ChatResponse>(`/v1/agents/${agentId}/chat`, {
      method: "POST",
      body: request,
    });
  }

  /**
   * 与 Agent 对话（流式）
   * @param agentId Agent ID
   * @param request Chat 请求
   * @returns AsyncIterable 流式事件
   */
  async *chatStream(
    agentId: string,
    request: ChatRequest,
  ): AsyncIterable<StreamChatEvent> {
    const response = await fetch(
      `${this.options.baseUrl}/v1/agents/${agentId}/chat/stream`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(this.options.apiKey && {
            Authorization: `Bearer ${this.options.apiKey}`,
          }),
        },
        body: JSON.stringify(request),
      },
    );

    if (!response.ok) {
      throw new Error(`Chat stream failed: ${response.statusText}`);
    }

    const reader = response.body?.getReader();
    if (!reader) {
      throw new Error("Response body is not readable");
    }

    const decoder = new TextDecoder();
    let buffer = "";

    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split("\n");
        buffer = lines.pop() || "";

        for (const line of lines) {
          if (line.startsWith("data: ")) {
            const data = line.slice(6);
            if (data === "[DONE]") {
              return;
            }
            try {
              const event = JSON.parse(data) as StreamChatEvent;
              yield event;
            } catch (error) {
              console.error("Failed to parse SSE data:", data);
            }
          }
        }
      }
    } finally {
      reader.releaseLock();
    }
  }

  // ==========================================================================
  // Agent Templates
  // ==========================================================================

  /**
   * 列出所有 Agent 模板
   * @returns 模板列表
   */
  async listTemplates(): Promise<AgentTemplate[]> {
    const result = await this.request<{ templates: AgentTemplate[] }>(
      "/v1/agents/templates",
    );
    return result.templates;
  }

  /**
   * 获取 Agent 模板详情
   * @param templateId 模板 ID
   * @returns 模板信息
   */
  async getTemplate(templateId: string): Promise<AgentTemplate> {
    return this.request<AgentTemplate>(`/v1/agents/templates/${templateId}`);
  }

  /**
   * 从模板创建 Agent
   * @param templateId 模板 ID
   * @param overrides 覆盖配置
   * @returns 创建的 Agent
   */
  async createFromTemplate(
    templateId: string,
    overrides: Partial<CreateAgentRequest>,
  ): Promise<AgentInfo> {
    const template = await this.getTemplate(templateId);

    const request: CreateAgentRequest = {
      name: overrides.name || `Agent from ${template.name}`,
      templateId,
      llmProvider:
        overrides.llmProvider || template.recommendedProvider || "openai",
      llmModel: overrides.llmModel || template.recommendedModel || "gpt-4",
      systemPrompt: overrides.systemPrompt || template.defaultSystemPrompt,
      tools: overrides.tools || template.defaultTools,
      middlewares: overrides.middlewares || template.defaultMiddlewares,
      description: overrides.description,
      llmParams: overrides.llmParams,
      metadata: overrides.metadata,
    };

    return this.create(request);
  }

  // ==========================================================================
  // Agent Statistics
  // ==========================================================================

  /**
   * 获取 Agent 统计信息
   * @param agentId Agent ID
   * @param timeRange 时间范围（可选）
   * @returns 统计数据
   */
  async getStats(
    agentId: string,
    timeRange?: { start: string; end: string },
  ): Promise<AgentStats> {
    return this.request<AgentStats>(`/v1/agents/${agentId}/stats`, {
      params: timeRange,
    });
  }

  /**
   * 获取所有 Agents 的汇总统计
   * @param timeRange 时间范围（可选）
   * @returns 汇总统计数据
   */
  async getAggregatedStats(timeRange?: {
    start: string;
    end: string;
  }): Promise<{
    totalAgents: number;
    activeAgents: number;
    totalRequests: number;
    totalTokens: number;
    totalCost: number;
    currency: string;
  }> {
    return this.request("/v1/agents/stats/aggregated", {
      params: timeRange,
    });
  }

  // ==========================================================================
  // Agent Validation
  // ==========================================================================

  /**
   * 验证 Agent 配置
   * @param config Agent 配置
   * @returns 验证结果
   */
  async validate(
    config: CreateAgentRequest | UpdateAgentRequest,
  ): Promise<AgentValidationResult> {
    return this.request<AgentValidationResult>("/v1/agents/validate", {
      method: "POST",
      body: config,
    });
  }

  // ==========================================================================
  // Batch Operations
  // ==========================================================================

  /**
   * 批量删除 Agents
   * @param agentIds Agent ID 列表
   */
  async deleteBatch(agentIds: string[]): Promise<void> {
    await this.request("/v1/agents/batch", {
      method: "DELETE",
      body: { agentIds },
    });
  }

  /**
   * 批量归档 Agents
   * @param agentIds Agent ID 列表
   */
  async archiveBatch(agentIds: string[]): Promise<void> {
    await this.request("/v1/agents/batch/archive", {
      method: "POST",
      body: { agentIds },
    });
  }

  /**
   * 批量激活 Agents
   * @param agentIds Agent ID 列表
   */
  async activateBatch(agentIds: string[]): Promise<void> {
    await this.request("/v1/agents/batch/activate", {
      method: "POST",
      body: { agentIds },
    });
  }

  // ==========================================================================
  // Agent Clone
  // ==========================================================================

  /**
   * 克隆 Agent
   * @param agentId 源 Agent ID
   * @param newName 新 Agent 名称
   * @returns 克隆的 Agent
   */
  async clone(agentId: string, newName: string): Promise<AgentInfo> {
    return this.request<AgentInfo>(`/v1/agents/${agentId}/clone`, {
      method: "POST",
      body: { name: newName },
    });
  }
}
