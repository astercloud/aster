/**
 * MCP 资源类
 * Model Context Protocol - AsterClient 作为 MCP Client 和 Server
 */

import { BaseResource, ClientOptions } from "./base";
import {
  MCPServerConfig,
  MCPServerInfo,
  MCPTool,
  MCPToolCallRequest,
  MCPToolCallResponse,
  MCPResource as MCPResourceData,
  MCPResourceContent,
  MCPPrompt,
  MCPPromptResult,
  MCPStats,
} from "../types/mcp";

/**
 * MCP 资源类
 */
export class MCPResource extends BaseResource {
  constructor(options: ClientOptions) {
    super(options);
  }

  // ==========================================================================
  // MCP Server Management
  // ==========================================================================

  /**
   * 添加 MCP Server
   * @param config Server 配置
   * @returns Server 信息
   */
  async addServer(config: MCPServerConfig): Promise<MCPServerInfo> {
    return this.request<MCPServerInfo>("/v1/mcp/servers", {
      method: "POST",
      body: config,
    });
  }

  /**
   * 创建 MCP Server（别名方法）
   * @param config Server 配置
   * @returns Server 信息
   */
  async createServer(config: MCPServerConfig): Promise<MCPServerInfo> {
    return this.addServer(config);
  }

  /**
   * 列出所有 MCP Servers
   * @returns Server 列表
   */
  async listServers(): Promise<MCPServerInfo[]> {
    const result = await this.request<{ servers: MCPServerInfo[] }>(
      "/v1/mcp/servers",
    );
    return result.servers;
  }

  /**
   * 获取 Server 详情
   * @param serverId Server ID
   * @returns Server 信息
   */
  async getServer(serverId: string): Promise<MCPServerInfo> {
    return this.request<MCPServerInfo>(`/v1/mcp/servers/${serverId}`);
  }

  /**
   * 更新 Server 配置
   * @param serverId Server ID
   * @param updates 更新内容
   * @returns 更新后的 Server 信息
   */
  async updateServer(
    serverId: string,
    updates: Partial<MCPServerConfig>,
  ): Promise<MCPServerInfo> {
    return this.request<MCPServerInfo>(`/v1/mcp/servers/${serverId}`, {
      method: "PATCH",
      body: updates,
    });
  }

  /**
   * 移除 MCP Server
   * @param serverId Server ID
   */
  async removeServer(serverId: string): Promise<void> {
    await this.request(`/v1/mcp/servers/${serverId}`, {
      method: "DELETE",
    });
  }

  /**
   * 删除 MCP Server（别名方法）
   * @param serverId Server ID
   */
  async deleteServer(serverId: string): Promise<void> {
    return this.removeServer(serverId);
  }

  /**
   * 连接到 Server
   * @param serverId Server ID
   * @returns 连接结果
   */
  async connectServer(serverId: string): Promise<MCPServerInfo> {
    return this.request<MCPServerInfo>(`/v1/mcp/servers/${serverId}/connect`, {
      method: "POST",
    });
  }

  /**
   * 启动 MCP Server（别名方法）
   * @param serverId Server ID
   * @returns Server 信息
   */
  async startServer(serverId: string): Promise<MCPServerInfo> {
    return this.connectServer(serverId);
  }

  /**
   * 断开 Server 连接
   * @param serverId Server ID
   */
  async disconnectServer(serverId: string): Promise<void> {
    await this.request(`/v1/mcp/servers/${serverId}/disconnect`, {
      method: "POST",
    });
  }

  /**
   * 停止 MCP Server（别名方法）
   * @param serverId Server ID
   */
  async stopServer(serverId: string): Promise<void> {
    return this.disconnectServer(serverId);
  }

  // ==========================================================================
  // MCP Tools
  // ==========================================================================

  /**
   * 获取 Server 的所有工具
   * @param serverId Server ID
   * @returns 工具列表
   */
  async getServerTools(serverId: string): Promise<MCPTool[]> {
    const result = await this.request<{ tools: MCPTool[] }>(
      `/v1/mcp/servers/${serverId}/tools`,
    );
    return result.tools;
  }

  /**
   * 获取所有可用工具（所有 Server）
   * @returns 工具列表
   */
  async listAllTools(): Promise<MCPTool[]> {
    const result = await this.request<{ tools: MCPTool[] }>("/v1/mcp/tools");
    return result.tools;
  }

  /**
   * 调用 MCP 工具
   * @param request 工具调用请求
   * @returns 调用响应
   */
  async callTool(request: MCPToolCallRequest): Promise<MCPToolCallResponse> {
    return this.request<MCPToolCallResponse>("/v1/mcp/tools/call", {
      method: "POST",
      body: request,
      timeout: request.timeout,
    });
  }

  /**
   * 调用 MCP 工具（便捷方法）
   * @param serverId Server ID
   * @param toolName 工具名称
   * @param params 参数
   * @returns 调用响应
   */
  async call(
    serverId: string,
    toolName: string,
    params: Record<string, any>,
  ): Promise<MCPToolCallResponse> {
    return this.callTool({ serverId, toolName, params });
  }

  // ==========================================================================
  // MCP Resources
  // ==========================================================================

  /**
   * 列出 Server 的所有资源
   * @param serverId Server ID
   * @returns 资源列表
   */
  async listResources(serverId: string): Promise<MCPResourceData[]> {
    const result = await this.request<{ resources: MCPResourceData[] }>(
      `/v1/mcp/servers/${serverId}/resources`,
    );
    return result.resources;
  }

  /**
   * 读取资源内容
   * @param serverId Server ID
   * @param uri 资源 URI
   * @returns 资源内容
   */
  async readResource(
    serverId: string,
    uri: string,
  ): Promise<MCPResourceContent> {
    return this.request<MCPResourceContent>(
      `/v1/mcp/servers/${serverId}/resources/read`,
      {
        method: "POST",
        body: { uri },
      },
    );
  }

  // ==========================================================================
  // MCP Prompts
  // ==========================================================================

  /**
   * 列出 Server 的所有 Prompts
   * @param serverId Server ID
   * @returns Prompt 列表
   */
  async listPrompts(serverId: string): Promise<MCPPrompt[]> {
    const result = await this.request<{ prompts: MCPPrompt[] }>(
      `/v1/mcp/servers/${serverId}/prompts`,
    );
    return result.prompts;
  }

  /**
   * 获取 Prompt
   * @param serverId Server ID
   * @param name Prompt 名称
   * @param args Prompt 参数
   * @returns Prompt 结果
   */
  async getPrompt(
    serverId: string,
    name: string,
    args?: Record<string, any>,
  ): Promise<MCPPromptResult> {
    return this.request<MCPPromptResult>(
      `/v1/mcp/servers/${serverId}/prompts/${name}`,
      {
        method: "POST",
        body: { arguments: args },
      },
    );
  }

  // ==========================================================================
  // MCP Statistics
  // ==========================================================================

  /**
   * 获取 MCP 统计信息
   * @returns 统计数据
   */
  async getStats(): Promise<MCPStats> {
    return this.request<MCPStats>("/v1/mcp/stats");
  }

  /**
   * 获取 Server 统计信息
   * @param serverId Server ID
   * @returns Server 统计数据
   */
  async getServerStats(serverId: string): Promise<{
    serverId: string;
    totalCalls: number;
    successfulCalls: number;
    failedCalls: number;
    avgResponseTime: number;
  }> {
    return this.request(`/v1/mcp/servers/${serverId}/stats`);
  }
}
