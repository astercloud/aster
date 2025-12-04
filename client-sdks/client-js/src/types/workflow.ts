/**
 * Workflow 相关类型定义
 * 三种工作流模式：Parallel、Sequential、Loop
 */

// ============================================================================
// Workflow Types
// ============================================================================

/**
 * Workflow 类型
 */
export type WorkflowType =
  | "parallel" // 并行执行
  | "sequential" // 顺序执行
  | "loop"; // 循环执行

/**
 * Workflow 状态
 */
export type WorkflowStatus =
  | "draft" // 草稿
  | "active" // 活跃
  | "paused" // 暂停
  | "archived"; // 归档

// ============================================================================
// Workflow Definition
// ============================================================================

/**
 * Workflow 定义基础接口
 */
interface BaseWorkflowDefinition {
  /** Workflow 名称 */
  name: string;
  /** 描述 */
  description?: string;
  /** 元数据 */
  metadata?: Record<string, any>;
}

/**
 * Parallel Workflow 定义
 * 多个 Agent 并行执行
 */
export interface ParallelWorkflowDefinition extends BaseWorkflowDefinition {
  type: "parallel";
  /** Agent 配置列表 */
  agents: Array<{
    id: string;
    task: string;
    config?: Record<string, any>;
  }>;
  /** 最大并发数 */
  maxConcurrency?: number;
  /** 超时时间（秒） */
  timeout?: number;
}

/**
 * Sequential Workflow 定义
 * Agent 按顺序执行
 */
export interface SequentialWorkflowDefinition extends BaseWorkflowDefinition {
  type: "sequential";
  /** 执行步骤 */
  steps: Array<{
    agent: string;
    action: string;
    params?: Record<string, any>;
    /** 条件（可选） */
    condition?: string;
  }>;
  /** 失败时是否继续 */
  continueOnError?: boolean;
}

/**
 * Loop Workflow 定义
 * 循环执行直到满足条件
 */
export interface LoopWorkflowDefinition extends BaseWorkflowDefinition {
  type: "loop";
  /** Agent ID */
  agent: string;
  /** 循环条件（JavaScript 表达式） */
  condition: string;
  /** 最大迭代次数 */
  maxIterations: number;
  /** 初始输入 */
  initialInput?: any;
}

/**
 * Workflow 定义联合类型
 */
export type WorkflowDefinition = ParallelWorkflowDefinition | SequentialWorkflowDefinition | LoopWorkflowDefinition;

// ============================================================================
// Workflow Info
// ============================================================================

/**
 * Workflow 信息
 */
export interface WorkflowInfo {
  /** Workflow ID */
  id: string;
  /** 名称 */
  name: string;
  /** 类型 */
  type: WorkflowType;
  /** 状态 */
  status: WorkflowStatus;
  /** 描述 */
  description?: string;
  /** 创建时间 */
  createdAt: string;
  /** 更新时间 */
  updatedAt: string;
  /** 定义 */
  definition: WorkflowDefinition;
  /** 元数据 */
  metadata?: Record<string, any>;
  /** 执行次数 */
  executionCount: number;
  /** 最后执行时间 */
  lastExecutedAt?: string;
}

// ============================================================================
// Workflow Execution
// ============================================================================

/**
 * Workflow 执行状态
 */
export type WorkflowRunStatus =
  | "pending" // 等待中
  | "running" // 运行中
  | "suspended" // 已暂停
  | "completed" // 完成
  | "failed" // 失败
  | "cancelled"; // 已取消

/**
 * Workflow 执行记录
 */
export interface WorkflowRun {
  /** Run ID */
  id: string;
  /** Workflow ID */
  workflowId: string;
  /** 状态 */
  status: WorkflowRunStatus;
  /** 开始时间 */
  startedAt: string;
  /** 结束时间 */
  completedAt?: string;
  /** 输入 */
  input: any;
  /** 输出 */
  output?: any;
  /** 错误信息 */
  error?: string;
  /** 执行进度（0-100） */
  progress: number;
  /** 当前步骤 */
  currentStep?: number;
  /** 总步骤数 */
  totalSteps?: number;
}

/**
 * Workflow 执行详情
 */
export interface RunDetails extends WorkflowRun {
  /** 执行步骤详情 */
  steps: WorkflowStepResult[];
  /** 统计信息 */
  stats: WorkflowRunStats;
}

/**
 * Workflow 步骤结果
 */
export interface WorkflowStepResult {
  /** 步骤序号 */
  stepIndex: number;
  /** Agent ID */
  agentId: string;
  /** 状态 */
  status: "pending" | "running" | "completed" | "failed" | "skipped";
  /** 开始时间 */
  startedAt?: string;
  /** 结束时间 */
  completedAt?: string;
  /** 输入 */
  input?: any;
  /** 输出 */
  output?: any;
  /** 错误 */
  error?: string;
  /** 耗时（毫秒） */
  duration?: number;
}

/**
 * Workflow 执行统计
 */
export interface WorkflowRunStats {
  /** 总耗时（毫秒） */
  totalDuration: number;
  /** 成功步骤数 */
  successfulSteps: number;
  /** 失败步骤数 */
  failedSteps: number;
  /** 总 Token 数 */
  totalTokens: number;
  /** 总成本 */
  totalCost: number;
}

// ============================================================================
// Workflow Operations
// ============================================================================

/**
 * 执行 Workflow 请求
 */
export interface ExecuteWorkflowRequest {
  /** 输入数据 */
  input: any;
  /** 执行选项 */
  options?: ExecutionOptions;
}

/**
 * 执行选项
 */
export interface ExecutionOptions {
  /** 超时时间（秒） */
  timeout?: number;
  /** 是否异步执行 */
  async?: boolean;
  /** 回调 URL */
  callbackUrl?: string;
  /** 元数据 */
  metadata?: Record<string, any>;
}

/**
 * 暂停 Workflow 请求
 */
export interface SuspendWorkflowRequest {
  /** Run ID */
  runId: string;
  /** 原因 */
  reason?: string;
}

/**
 * 恢复 Workflow 请求
 */
export interface ResumeWorkflowRequest {
  /** Run ID */
  runId: string;
  /** 额外输入（可选） */
  additionalInput?: any;
}

/**
 * 取消 Workflow 请求
 */
export interface CancelWorkflowRequest {
  /** Run ID */
  runId: string;
  /** 原因 */
  reason?: string;
}

// ============================================================================
// Workflow Filtering and Pagination
// ============================================================================

/**
 * Workflow 过滤器
 */
export interface WorkflowFilter {
  /** 类型 */
  type?: WorkflowType;
  /** 状态 */
  status?: WorkflowStatus;
  /** 搜索关键词（名称或描述） */
  search?: string;
  /** 创建时间范围 */
  createdAfter?: string;
  createdBefore?: string;
  /** 分页 */
  page?: number;
  pageSize?: number;
}

/**
 * Workflow Run 过滤器
 */
export interface WorkflowRunFilter {
  /** Workflow ID */
  workflowId?: string;
  /** 状态 */
  status?: WorkflowRunStatus;
  /** 时间范围 */
  startedAfter?: string;
  startedBefore?: string;
  /** 分页 */
  page?: number;
  pageSize?: number;
}

// ============================================================================
// Workflow Validation
// ============================================================================

/**
 * Workflow 验证结果
 */
export interface WorkflowValidationResult {
  /** 是否有效 */
  valid: boolean;
  /** 错误列表 */
  errors?: string[];
  /** 警告列表 */
  warnings?: string[];
}

/**
 * Workflow 更新请求
 */
export interface UpdateWorkflowRequest {
  /** 名称 */
  name?: string;
  /** 描述 */
  description?: string;
  /** 状态 */
  status?: WorkflowStatus;
  /** 定义 */
  definition?: WorkflowDefinition;
  /** 元数据 */
  metadata?: Record<string, any>;
}
