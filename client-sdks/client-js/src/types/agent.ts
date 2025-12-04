/**
 * Agent 相关类型定义
 * Agent 配置、管理、统计
 */

// ============================================================================
// Agent Configuration
// ============================================================================

/**
 * Agent 配置
 */
export interface AgentConfig {
  /** Agent ID */
  id: string;
  /** Agent 名称 */
  name: string;
  /** 描述 */
  description?: string;
  /** 模板 ID */
  templateId: string;
  /** LLM Provider */
  llmProvider: string;
  /** LLM Model */
  llmModel: string;
  /** 系统提示词 */
  systemPrompt?: string;
  /** 温度参数 */
  temperature?: number;
  /** 最大 Token 数 */
  maxTokens?: number;
  /** Top P */
  topP?: number;
  /** 工具列表 */
  tools?: string[];
  /** 中间件配置 */
  middlewares?: string[];
  /** 元数据 */
  metadata?: Record<string, any>;
}

/**
 * Agent 状态
 */
export type AgentStatus =
  | "active" // 活跃
  | "inactive" // 未激活
  | "disabled" // 已禁用
  | "archived"; // 已归档

/**
 * Agent 信息
 */
export interface AgentInfo extends AgentConfig {
  /** 状态 */
  status: AgentStatus;
  /** 创建时间 */
  createdAt: string;
  /** 更新时间 */
  updatedAt: string;
  /** 最后使用时间 */
  lastUsedAt?: string;
  /** 版本号 */
  version: number;
  /** 创建者 */
  createdBy?: string;
}

// ============================================================================
// Agent Templates
// ============================================================================

/**
 * Agent 模板类型
 */
export type AgentTemplateType =
  | "assistant" // 通用助手
  | "researcher" // 研究员
  | "coder" // 程序员
  | "writer" // 写作者
  | "analyzer" // 分析师
  | "translator" // 翻译员
  | "custom"; // 自定义

/**
 * Agent 模板
 */
export interface AgentTemplate {
  /** 模板 ID */
  id: string;
  /** 模板名称 */
  name: string;
  /** 类型 */
  type: AgentTemplateType;
  /** 描述 */
  description: string;
  /** 默认系统提示词 */
  defaultSystemPrompt: string;
  /** 推荐的 LLM Provider */
  recommendedProvider?: string;
  /** 推荐的 LLM Model */
  recommendedModel?: string;
  /** 默认工具 */
  defaultTools?: string[];
  /** 默认中间件 */
  defaultMiddlewares?: string[];
  /** 是否内置 */
  builtin: boolean;
}

// ============================================================================
// Agent Operations
// ============================================================================

/**
 * 创建 Agent 请求
 */
export interface CreateAgentRequest {
  /** Agent 名称 */
  name: string;
  /** 描述 */
  description?: string;
  /** 模板 ID */
  templateId: string;
  /** LLM Provider */
  llmProvider: string;
  /** LLM Model */
  llmModel: string;
  /** 系统提示词（可选，使用模板默认值） */
  systemPrompt?: string;
  /** LLM 参数 */
  llmParams?: {
    temperature?: number;
    maxTokens?: number;
    topP?: number;
  };
  /** 工具列表（可选） */
  tools?: string[];
  /** 中间件列表（可选） */
  middlewares?: string[];
  /** 元数据 */
  metadata?: Record<string, any>;
}

/**
 * 更新 Agent 请求
 */
export interface UpdateAgentRequest {
  /** Agent 名称 */
  name?: string;
  /** 描述 */
  description?: string;
  /** 系统提示词 */
  systemPrompt?: string;
  /** LLM Provider */
  llmProvider?: string;
  /** LLM Model */
  llmModel?: string;
  /** LLM 参数 */
  llmParams?: {
    temperature?: number;
    maxTokens?: number;
    topP?: number;
  };
  /** 工具列表 */
  tools?: string[];
  /** 中间件列表 */
  middlewares?: string[];
  /** 状态 */
  status?: AgentStatus;
  /** 元数据 */
  metadata?: Record<string, any>;
}

// ============================================================================
// Agent Chat
// ============================================================================

/**
 * Chat 请求
 */
export interface ChatRequest {
  /** 用户输入 */
  input: string;
  /** Session ID（可选，复用会话） */
  sessionId?: string;
  /** 用户 ID */
  userId?: string;
  /** 上下文（可选） */
  context?: Record<string, any>;
  /** 是否启用流式响应 */
  stream?: boolean;
}

/**
 * Chat 响应
 */
export interface ChatResponse {
  /** Session ID */
  sessionId: string;
  /** Agent 响应 */
  response: string;
  /** 工具调用（如果有） */
  toolCalls?: ToolCall[];
  /** Token 使用统计 */
  usage?: {
    promptTokens: number;
    completionTokens: number;
    totalTokens: number;
  };
  /** 成本 */
  cost?: {
    amount: number;
    currency: string;
  };
  /** 执行时间（毫秒） */
  executionTime: number;
}

/**
 * 工具调用
 */
export interface ToolCall {
  /** 工具 ID */
  id: string;
  /** 工具名称 */
  name: string;
  /** 参数 */
  parameters: Record<string, any>;
  /** 结果 */
  result?: any;
  /** 状态 */
  status: "pending" | "running" | "completed" | "failed";
}

/**
 * 流式 Chat 事件
 */
export type StreamChatEvent =
  | { type: "start"; sessionId: string }
  | { type: "token"; token: string }
  | { type: "tool_call"; toolCall: ToolCall }
  | { type: "end"; response: ChatResponse }
  | { type: "error"; error: string };

// ============================================================================
// Agent Statistics
// ============================================================================

/**
 * Agent 统计信息
 */
export interface AgentStats {
  /** Agent ID */
  agentId: string;
  /** 时间范围 */
  timeRange: {
    start: string;
    end: string;
  };
  /** 总请求数 */
  totalRequests: number;
  /** 成功请求数 */
  successfulRequests: number;
  /** 失败请求数 */
  failedRequests: number;
  /** 平均响应时间（毫秒） */
  avgResponseTime: number;
  /** Token 使用统计 */
  tokenUsage: {
    totalTokens: number;
    promptTokens: number;
    completionTokens: number;
  };
  /** 成本统计 */
  cost: {
    total: number;
    currency: string;
  };
  /** 工具调用统计 */
  toolCalls?: {
    total: number;
    byTool: Record<string, number>;
  };
  /** 活跃用户数 */
  activeUsers?: number;
  /** 平均会话长度 */
  avgSessionLength?: number;
}

// ============================================================================
// Agent Filtering and Pagination
// ============================================================================

/**
 * Agent 过滤器
 */
export interface AgentFilter {
  /** 状态 */
  status?: AgentStatus;
  /** 模板 ID */
  templateId?: string;
  /** LLM Provider */
  llmProvider?: string;
  /** 搜索关键词（名称或描述） */
  search?: string;
  /** 创建时间范围 */
  createdAfter?: string;
  createdBefore?: string;
  /** 排序字段 */
  sortBy?: "name" | "createdAt" | "updatedAt" | "lastUsedAt";
  /** 排序方向 */
  sortOrder?: "asc" | "desc";
  /** 分页 */
  page?: number;
  pageSize?: number;
}

/**
 * 分页响应
 */
export interface PaginatedAgentResponse {
  /** Agent 列表 */
  items: AgentInfo[];
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
// Agent Validation
// ============================================================================

/**
 * Agent 验证结果
 */
export interface AgentValidationResult {
  /** 是否有效 */
  valid: boolean;
  /** 错误列表 */
  errors?: string[];
  /** 警告列表 */
  warnings?: string[];
}
