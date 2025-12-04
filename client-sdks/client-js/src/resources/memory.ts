/**
 * Memory 资源类
 * 实现三层记忆系统：Text Memory、Working Memory、Semantic Memory
 */

import { BaseResource, ClientOptions } from "./base";
import { WorkingMemoryScope, WorkingMemorySetOptions, WorkingMemoryItem, MemoryChunk, SearchOptions, SearchResult, Provenance, ProvenanceResponse, ConsolidateOptions, ConsolidationResult, JobStatus } from "../types/memory";

/**
 * Memory 资源类
 * 管理三层记忆系统的所有操作
 */
export class MemoryResource extends BaseResource {
  constructor(options: ClientOptions) {
    super(options);
  }

  // ==========================================================================
  // Working Memory API
  // ==========================================================================

  /**
   * Working Memory 操作
   * LLM 可主动更新的工作记忆，支持 Thread/Resource 双作用域
   */
  working = {
    /**
     * 获取 Working Memory 值
     * @param key 键名
     * @param scope 作用域（默认: thread）
     * @returns 值
     */
    get: async (key: string, scope?: WorkingMemoryScope): Promise<any> => {
      const params: any = { key };
      if (scope) params.scope = scope;
      const result = await this.request<{ value: any }>("/v1/memory/working", {
        params,
      });
      return result.value;
    },

    /**
     * 设置 Working Memory 值
     * @param key 键名
     * @param value 值
     * @param options 选项（作用域、TTL、Schema）
     */
    set: async (key: string, value: any, options?: WorkingMemorySetOptions): Promise<void> => {
      await this.request("/v1/memory/working", {
        method: "POST",
        body: {
          key,
          value,
          scope: options?.scope ?? "thread",
          ttl: options?.ttl,
          schema: options?.schema,
        },
      });
    },

    /**
     * 删除 Working Memory 值
     * @param key 键名
     * @param scope 作用域（默认: thread）
     */
    delete: async (key: string, scope?: WorkingMemoryScope): Promise<void> => {
      const params = scope ? { scope } : undefined;
      await this.request(`/v1/memory/working/${key}`, {
        method: "DELETE",
        params,
      });
    },

    /**
     * 列出所有 Working Memory
     * @param scope 作用域（可选，不指定则返回所有）
     * @returns 键值对对象
     */
    list: async (scope?: WorkingMemoryScope): Promise<Record<string, any>> => {
      const params = scope ? { scope } : undefined;
      const result = await this.request<WorkingMemoryItem[]>("/v1/memory/working", { params });

      // 转换为键值对对象
      const items: Record<string, any> = {};
      if (Array.isArray(result)) {
        result.forEach((item) => {
          items[item.key] = item.value;
        });
      }
      return items;
    },

    /**
     * 清空 Working Memory
     * @param scope 作用域（可选，不指定则清空所有）
     */
    clear: async (scope?: WorkingMemoryScope): Promise<void> => {
      const body = scope ? { scope } : {};
      await this.request("/v1/memory/working/clear", {
        method: "POST",
        body,
      });
    },
  };

  // ==========================================================================
  // Semantic Memory API
  // ==========================================================================

  /**
   * Semantic Memory 操作
   * 向量检索和语义搜索
   */
  semantic = {
    /**
     * 语义搜索
     * @param query 查询文本
     * @param options 搜索选项
     * @returns 搜索结果
     */
    search: async (query: string, options?: SearchOptions): Promise<MemoryChunk[]> => {
      const result = await this.request<SearchResult>("/v1/memory/semantic/search", {
        method: "POST",
        body: {
          query,
          limit: options?.limit ?? 10,
          threshold: options?.threshold,
          filter: options?.filter,
        },
      });
      return result.chunks;
    },

    /**
     * 存储记忆
     * @param content 内容
     * @param metadata 元数据（可选）
     * @returns 记忆块 ID
     */
    store: async (content: string, metadata?: Record<string, any>): Promise<string> => {
      const result = await this.request<{ id: string }>("/v1/memory/semantic", {
        method: "POST",
        body: { content, ...metadata },
      });
      return result.id;
    },

    /**
     * 创建语义记忆（别名方法）
     * @param data 记忆数据
     * @returns 记忆信息
     */
    create: async (data: { content: string; tags?: string[]; metadata?: Record<string, any> }): Promise<{ id: string; content: string; [key: string]: any }> => {
      const chunkId = await this.semantic.store(data.content, {
        ...data.metadata,
        tags: data.tags,
      });
      return { id: chunkId, content: data.content };
    },

    /**
     * 列出所有语义记忆（别名方法）
     * @param options 查询选项
     * @returns 记忆列表
     */
    list: async (options?: { limit?: number; filter?: Record<string, any> }): Promise<MemoryChunk[]> => {
      // 使用空查询返回所有记忆
      return this.semantic.search("", options);
    },

    /**
     * 删除记忆
     * @param chunkId 记忆块 ID
     */
    delete: async (chunkId: string): Promise<void> => {
      await this.request(`/v1/memory/semantic/${chunkId}`, {
        method: "DELETE",
      });
    },

    /**
     * 批量删除记忆
     * @param chunkIds 记忆块 ID 列表
     */
    deleteBatch: async (chunkIds: string[]): Promise<void> => {
      await this.request("/v1/memory/semantic/batch", {
        method: "DELETE",
        body: { chunkIds },
      });
    },
  };

  // ==========================================================================
  // Memory Provenance (溯源) API
  // ==========================================================================

  /**
   * 获取记忆溯源信息
   * 追踪记忆的来源、置信度和谱系关系
   *
   * @param memoryId 记忆 ID
   * @returns 溯源信息
   */
  async getProvenance(memoryId: string): Promise<ProvenanceResponse> {
    return this.request<ProvenanceResponse>(`/v1/memory/provenance/${memoryId}`);
  }

  /**
   * 获取记忆谱系链
   * 从根记忆到当前记忆的完整路径
   *
   * @param memoryId 记忆 ID
   * @returns 谱系链
   */
  async getLineage(memoryId: string): Promise<Provenance[]> {
    const result = await this.request<{ lineage: Provenance[] }>(`/v1/memory/lineage/${memoryId}`);
    return result.lineage;
  }

  // ==========================================================================
  // Memory Consolidation (合并) API
  // ==========================================================================

  /**
   * 触发记忆合并
   * LLM 驱动的智能合并，处理冗余、冲突、生成总结
   *
   * @param options 合并选项
   * @returns 合并任务结果
   */
  async consolidate(options?: ConsolidateOptions): Promise<ConsolidationResult> {
    return this.request<ConsolidationResult>("/v1/memory/consolidate", {
      method: "POST",
      body: options,
    });
  }

  /**
   * 获取合并任务状态
   *
   * @param jobId 任务 ID
   * @returns 任务状态
   */
  async getConsolidationStatus(jobId: string): Promise<JobStatus> {
    return this.request<JobStatus>(`/v1/memory/consolidation/${jobId}`);
  }

  /**
   * 取消合并任务
   *
   * @param jobId 任务 ID
   */
  async cancelConsolidation(jobId: string): Promise<void> {
    await this.request(`/v1/memory/consolidation/${jobId}`, {
      method: "DELETE",
    });
  }

  // ==========================================================================
  // 统计和管理 API
  // ==========================================================================

  /**
   * 获取记忆统计信息
   * @returns 统计信息
   */
  async getStats(): Promise<{
    workingMemory: {
      threadCount: number;
      resourceCount: number;
      totalSize: number;
    };
    semanticMemory: {
      chunkCount: number;
      totalSize: number;
    };
  }> {
    return this.request("/v1/memory/stats");
  }

  /**
   * 清空所有记忆（危险操作！）
   * @param confirm 确认标志
   */
  async clearAll(confirm: boolean = false): Promise<void> {
    if (!confirm) {
      throw new Error("Must confirm to clear all memory. Pass confirm=true.");
    }

    await this.request("/v1/memory/clear", {
      method: "POST",
      body: { confirm: true },
    });
  }
}
