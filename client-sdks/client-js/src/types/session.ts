/**
 * Session 相关类型定义
 * 会话生命周期管理、消息历史、断点恢复
 */

import { Message as MemoryMessage } from "./memory";

// 重新导出 Message 类型
export type { MemoryMessage as Message };

// ============================================================================
// Session Configuration
// ============================================================================

/**
 * Session 配置
 */
export interface SessionConfig {
  /** Agent ID */
  agentId: string;
  /** Template ID */
  templateId?: string;
  /** 用户 ID */
  userId?: string;
  /** 元数据 */
  metadata?: Record<string, any>;
  /** 启用断点恢复 */
  enableCheckpoints?: boolean;
  /** 断点间隔（消息数） */
  checkpointInterval?: number;
}

/**
 * Session 信息
 */
export interface SessionInfo {
  /** Session ID */
  id: string;
  /** Agent ID */
  agentId: string;
  /** Template ID */
  templateId?: string;
  /** 用户 ID */
  userId?: string;
  /** 状态 */
  status: SessionStatus;
  /** 创建时间 */
  createdAt: string;
  /** 更新时间 */
  updatedAt: string;
  /** 元数据 */
  metadata?: Record<string, any>;
  /** 消息数 */
  messageCount: number;
  /** 最后一条消息时间 */
  lastMessageAt?: string;
}

/**
 * Session 状态
 */
export type SessionStatus =
  | "active" // 活跃
  | "paused" // 暂停
  | "completed" // 完成
  | "archived"; // 归档

/**
 * Session 过滤器
 */
export interface SessionFilter {
  /** 用户 ID */
  userId?: string;
  /** Agent ID */
  agentId?: string;
  /** 状态 */
  status?: SessionStatus;
  /** 开始时间 */
  startDate?: string;
  /** 结束时间 */
  endDate?: string;
  /** 分页 */
  page?: number;
  /** 每页数量 */
  pageSize?: number;
}

// ============================================================================
// Messages
// ============================================================================

/**
 * 分页参数
 */
export interface Pagination {
  /** 页码（从1开始） */
  page?: number;
  /** 每页数量 */
  pageSize?: number;
  /** 排序方式 */
  sort?: "asc" | "desc";
}

/**
 * 分页响应
 */
export interface PaginatedResponse<T> {
  /** 数据列表 */
  items: T[];
  /** 总数 */
  total: number;
  /** 当前页 */
  page: number;
  /** 每页数量 */
  pageSize: number;
  /** 总页数 */
  totalPages: number;
}

// ============================================================================
// Checkpoints (断点恢复)
// ============================================================================

/**
 * Checkpoint 类型
 * AsterClient 使用 7 段断点机制
 */
export type CheckpointType =
  | "user_input" // 1. 用户输入
  | "agent_thinking" // 2. Agent 思考
  | "tool_call" // 3. 工具调用
  | "tool_result" // 4. 工具结果
  | "agent_response" // 5. Agent 响应
  | "memory_update" // 6. 记忆更新
  | "session_state"; // 7. 会话状态

/**
 * Checkpoint 信息
 */
export interface Checkpoint {
  /** Checkpoint ID */
  id: string;
  /** Session ID */
  sessionId: string;
  /** 类型 */
  type: CheckpointType;
  /** 序号 */
  sequence: number;
  /** 时间戳 */
  timestamp: string;
  /** 状态快照 */
  state: CheckpointState;
  /** 消息索引 */
  messageIndex: number;
}

/**
 * Checkpoint 状态快照
 */
export interface CheckpointState {
  /** 消息历史 */
  messages: MemoryMessage[];
  /** Working Memory 快照 */
  workingMemory?: Record<string, any>;
  /** Agent 状态 */
  agentState?: any;
  /** 上下文 */
  context?: Record<string, any>;
}

/**
 * 恢复选项
 */
export interface ResumeOptions {
  /** Checkpoint ID（不指定则从最新断点恢复） */
  checkpointId?: string;
  /** 是否保留后续消息 */
  keepSubsequentMessages?: boolean;
}

// ============================================================================
// Session Statistics
// ============================================================================

/**
 * Session 统计信息
 */
export interface SessionStats {
  /** Session ID */
  sessionId: string;
  /** 总消息数 */
  totalMessages: number;
  /** 用户消息数 */
  userMessages: number;
  /** 助手消息数 */
  assistantMessages: number;
  /** 总 Token 数 */
  totalTokens: number;
  /** Prompt Tokens */
  promptTokens: number;
  /** Completion Tokens */
  completionTokens: number;
  /** 总成本 */
  totalCost: number;
  /** 货币单位 */
  currency: string;
  /** 持续时间（秒） */
  duration: number;
  /** 工具调用次数 */
  toolCallCount: number;
  /** 断点数 */
  checkpointCount: number;
  /** 平均响应时间（毫秒） */
  avgResponseTime?: number;
}

// ============================================================================
// Session Operations
// ============================================================================

/**
 * 添加消息请求
 */
export interface AddMessageRequest {
  /** Session ID */
  sessionId: string;
  /** 消息内容 */
  message: Omit<MemoryMessage, "id" | "timestamp">;
}

/**
 * 更新 Session 请求
 */
export interface UpdateSessionRequest {
  /** 状态 */
  status?: SessionStatus;
  /** 元数据 */
  metadata?: Record<string, any>;
}

/**
 * Session 导出选项
 */
export interface ExportOptions {
  /** 格式 */
  format: "json" | "markdown" | "html";
  /** 是否包含元数据 */
  includeMetadata?: boolean;
  /** 是否包含统计信息 */
  includeStats?: boolean;
}

/**
 * Session 导出结果
 */
export interface ExportResult {
  /** Session ID */
  sessionId: string;
  /** 格式 */
  format: string;
  /** 导出内容 */
  content: string;
  /** 文件名建议 */
  suggestedFilename: string;
  /** 导出时间 */
  exportedAt: string;
}
