/**
 * Memory 相关类型定义
 * 三层记忆系统：Text Memory、Working Memory、Semantic Memory
 */

// ============================================================================
// Working Memory Types
// ============================================================================

/**
 * Working Memory 作用域
 * - thread: 会话级别（当前对话）
 * - resource: 全局级别（跨会话）
 */
export type WorkingMemoryScope = "thread" | "resource";

/**
 * JSON Schema 定义
 * 用于验证 Working Memory 数据结构
 */
export interface JSONSchema {
  /** 数据类型 */
  type: string;
  /** 属性定义 */
  properties?: Record<string, any>;
  /** 必需字段 */
  required?: string[];
  /** 其他 JSON Schema 字段 */
  [key: string]: any;
}

/**
 * Working Memory 设置选项
 */
export interface WorkingMemorySetOptions {
  /** 作用域（默认: thread） */
  scope?: WorkingMemoryScope;
  /** TTL 过期时间（秒），0 表示永不过期 */
  ttl?: number;
  /** JSON Schema 验证 */
  schema?: JSONSchema;
}

/**
 * Working Memory 项
 */
export interface WorkingMemoryItem {
  /** 键名 */
  key: string;
  /** 值 */
  value: any;
  /** 作用域 */
  scope: WorkingMemoryScope;
  /** 创建时间 */
  createdAt: string;
  /** 过期时间（如果有 TTL） */
  expiresAt?: string;
  /** Schema（如果有） */
  schema?: JSONSchema;
}

// ============================================================================
// Semantic Memory Types
// ============================================================================

/**
 * 记忆块
 * Semantic Memory 的基本单元
 */
export interface MemoryChunk {
  /** 记忆块 ID */
  id: string;
  /** 内容 */
  content: string;
  /** 元数据 */
  metadata?: Record<string, any>;
  /** 相似度分数（搜索时） */
  score?: number;
  /** 创建时间 */
  timestamp: string;
}

/**
 * Semantic Memory 搜索选项
 */
export interface SearchOptions {
  /** 返回结果数量限制 */
  limit?: number;
  /** 相似度阈值（0-1） */
  threshold?: number;
  /** 元数据过滤条件 */
  filter?: Record<string, any>;
}

/**
 * Semantic Memory 搜索结果
 */
export interface SearchResult {
  /** 匹配的记忆块列表 */
  chunks: MemoryChunk[];
  /** 总匹配数 */
  total: number;
  /** 查询耗时（毫秒） */
  latencyMs?: number;
}

// ============================================================================
// Memory Provenance (溯源)
// ============================================================================

/**
 * 记忆来源类型
 */
export type MemorySource =
  | "user_input"
  | "tool_output"
  | "inference"
  | "external";

/**
 * 记忆溯源信息
 */
export interface Provenance {
  /** 来源类型 */
  source: MemorySource;
  /** 置信度（0-1） */
  confidence: number;
  /** 时间戳 */
  timestamp: string;
  /** 父记忆 ID（谱系关系） */
  parentId?: string;
  /** 时间衰减因子 */
  decayFactor?: number;
  /** 其他元数据 */
  metadata?: Record<string, any>;
}

/**
 * 记忆溯源响应
 */
export interface ProvenanceResponse {
  /** 记忆 ID */
  memoryId: string;
  /** 溯源信息 */
  provenance: Provenance;
  /** 谱系链（从根到当前） */
  lineage?: Provenance[];
}

// ============================================================================
// Memory Consolidation (合并)
// ============================================================================

/**
 * 记忆合并策略
 */
export type ConsolidationStrategy =
  | "dedup" // 去重
  | "resolve_conflict" // 解决冲突
  | "summarize"; // 总结

/**
 * 记忆合并选项
 */
export interface ConsolidateOptions {
  /** 合并策略 */
  strategy?: ConsolidationStrategy;
  /** LLM Provider（用于 summarize 策略） */
  llmProvider?: string;
  /** LLM Model */
  llmModel?: string;
  /** 目标记忆范围（按时间） */
  timeRange?: {
    start: string;
    end: string;
  };
  /** 其他选项 */
  [key: string]: any;
}

/**
 * 合并任务状态
 */
export type ConsolidationStatus =
  | "pending" // 等待中
  | "running" // 运行中
  | "completed" // 完成
  | "failed"; // 失败

/**
 * 记忆合并结果
 */
export interface ConsolidationResult {
  /** 任务 ID */
  jobId: string;
  /** 状态 */
  status: ConsolidationStatus;
  /** 开始时间 */
  startedAt: string;
  /** 完成时间 */
  completedAt?: string;
  /** 合并的记忆数 */
  processedCount?: number;
  /** 结果记忆数 */
  resultCount?: number;
  /** 错误信息 */
  error?: string;
}

/**
 * 合并任务状态查询响应
 */
export interface JobStatus {
  /** 任务 ID */
  jobId: string;
  /** 状态 */
  status: ConsolidationStatus;
  /** 进度（0-100） */
  progress?: number;
  /** 结果 */
  result?: any;
  /** 错误信息 */
  error?: string;
}

// ============================================================================
// Text Memory Types (会话历史)
// ============================================================================

/**
 * 消息角色
 */
export type MessageRole = "user" | "assistant" | "system" | "tool";

/**
 * 消息内容
 */
export interface Message {
  /** 消息 ID */
  id?: string;
  /** 角色 */
  role: MessageRole;
  /** 内容 */
  content: string;
  /** 时间戳 */
  timestamp?: string;
  /** 元数据 */
  metadata?: Record<string, any>;
}

/**
 * 会话历史
 */
export interface ConversationHistory {
  /** 会话 ID */
  sessionId: string;
  /** 消息列表 */
  messages: Message[];
  /** 总消息数 */
  totalCount: number;
}

// ============================================================================
// 通用响应类型
// ============================================================================

/**
 * Memory API 通用响应
 */
export interface MemoryResponse<T = any> {
  /** 是否成功 */
  success: boolean;
  /** 数据 */
  data?: T;
  /** 错误信息 */
  error?: string;
}

/**
 * 批量操作响应
 */
export interface BatchOperationResponse {
  /** 成功数 */
  successCount: number;
  /** 失败数 */
  failureCount: number;
  /** 失败的项 */
  failures?: Array<{
    key: string;
    error: string;
  }>;
}
