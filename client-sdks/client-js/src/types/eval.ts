/**
 * Eval 相关类型定义
 * Agent 评估和测试系统
 */

// ============================================================================
// Eval Types
// ============================================================================

/**
 * Eval 类型
 */
export type EvalType =
  | "single" // 单次评估
  | "batch" // 批量评估
  | "benchmark" // 基准测试
  | "regression" // 回归测试
  | "ab_test"; // A/B 测试

/**
 * Eval 状态
 */
export type EvalStatus =
  | "pending" // 待执行
  | "running" // 执行中
  | "completed" // 已完成
  | "failed" // 失败
  | "cancelled"; // 已取消

// ============================================================================
// Scorers
// ============================================================================

/**
 * Scorer 类型
 */
export type ScorerType =
  | "exact_match" // 精确匹配
  | "contains" // 包含检查
  | "semantic_similarity" // 语义相似度
  | "llm_judge" // LLM 评分
  | "keyword_coverage" // 关键词覆盖率
  | "sentiment" // 情感分析
  | "factuality" // 事实准确性
  | "coherence" // 连贯性
  | "relevance" // 相关性
  | "custom"; // 自定义

/**
 * Scorer 配置
 */
export interface ScorerConfig {
  /** Scorer 类型 */
  type: ScorerType;
  /** 权重（0-1） */
  weight?: number;
  /** 配置参数 */
  params?: Record<string, any>;
}

/**
 * LLM Judge Scorer 配置
 */
export interface LLMJudgeScorerConfig extends ScorerConfig {
  type: "llm_judge";
  params: {
    /** LLM Provider */
    provider: string;
    /** LLM Model */
    model: string;
    /** 评分提示词 */
    prompt?: string;
    /** 评分维度 */
    criteria?: string[];
    /** 评分范围 */
    scoreRange?: {
      min: number;
      max: number;
    };
  };
}

/**
 * Semantic Similarity Scorer 配置
 */
export interface SemanticSimilarityScorerConfig extends ScorerConfig {
  type: "semantic_similarity";
  params: {
    /** 嵌入模型 */
    embeddingModel?: string;
    /** 相似度阈值 */
    threshold?: number;
  };
}

// ============================================================================
// Test Cases
// ============================================================================

/**
 * 测试用例
 */
export interface TestCase {
  /** 用例 ID */
  id: string;
  /** 用例名称 */
  name: string;
  /** 描述 */
  description?: string;
  /** 输入 */
  input: string;
  /** 期望输出 */
  expectedOutput?: string;
  /** 上下文（可选） */
  context?: Record<string, any>;
  /** 标签 */
  tags?: string[];
  /** 元数据 */
  metadata?: Record<string, any>;
}

/**
 * 测试用例集
 */
export interface TestCaseSet {
  /** 集合 ID */
  id: string;
  /** 集合名称 */
  name: string;
  /** 描述 */
  description?: string;
  /** 测试用例列表 */
  testCases: TestCase[];
  /** 创建时间 */
  createdAt: string;
  /** 更新时间 */
  updatedAt: string;
}

// ============================================================================
// Eval Execution
// ============================================================================

/**
 * Eval 请求
 */
export interface EvalRequest {
  /** Eval 名称 */
  name: string;
  /** Eval 类型 */
  type: EvalType;
  /** Agent ID */
  agentId: string;
  /** 测试用例集 ID 或测试用例列表 */
  testCases: string | TestCase[];
  /** Scorer 配置列表 */
  scorers: ScorerConfig[];
  /** 并发数（批量评估） */
  concurrency?: number;
  /** 超时时间（毫秒） */
  timeout?: number;
  /** 元数据 */
  metadata?: Record<string, any>;
}

/**
 * Eval 信息
 */
export interface EvalInfo {
  /** Eval ID */
  id: string;
  /** Eval 名称 */
  name: string;
  /** Eval 类型 */
  type: EvalType;
  /** 状态 */
  status: EvalStatus;
  /** Agent ID */
  agentId: string;
  /** 测试用例数 */
  totalTestCases: number;
  /** 已完成数 */
  completedTestCases: number;
  /** 进度（0-100） */
  progress: number;
  /** Scorer 配置 */
  scorers: ScorerConfig[];
  /** 创建时间 */
  createdAt: string;
  /** 开始时间 */
  startedAt?: string;
  /** 完成时间 */
  completedAt?: string;
  /** 错误信息 */
  error?: string;
  /** 元数据 */
  metadata?: Record<string, any>;
}

// ============================================================================
// Eval Results
// ============================================================================

/**
 * 单个测试用例的结果
 */
export interface TestCaseResult {
  /** 测试用例 ID */
  testCaseId: string;
  /** 测试用例名称 */
  testCaseName: string;
  /** Agent 输出 */
  output: string;
  /** 期望输出 */
  expectedOutput?: string;
  /** 各 Scorer 的评分 */
  scores: {
    [scorerType: string]: {
      score: number;
      details?: any;
    };
  };
  /** 加权总分 */
  overallScore: number;
  /** 是否通过 */
  passed: boolean;
  /** 执行时间（毫秒） */
  executionTime: number;
  /** Token 使用 */
  tokenUsage?: {
    promptTokens: number;
    completionTokens: number;
    totalTokens: number;
  };
  /** 成本 */
  cost?: {
    amount: number;
    currency: string;
  };
  /** 错误信息 */
  error?: string;
}

/**
 * Eval 结果
 */
export interface EvalResult {
  /** Eval ID */
  evalId: string;
  /** Eval 名称 */
  evalName: string;
  /** Agent ID */
  agentId: string;
  /** 状态 */
  status: EvalStatus;
  /** 测试用例结果列表 */
  testCaseResults: TestCaseResult[];
  /** 汇总统计 */
  summary: {
    /** 总测试用例数 */
    totalTestCases: number;
    /** 通过数 */
    passed: number;
    /** 失败数 */
    failed: number;
    /** 通过率 */
    passRate: number;
    /** 平均分数 */
    avgScore: number;
    /** 各 Scorer 的平均分 */
    avgScoresByScorer: {
      [scorerType: string]: number;
    };
    /** 总执行时间（毫秒） */
    totalExecutionTime: number;
    /** 平均执行时间（毫秒） */
    avgExecutionTime: number;
    /** 总 Token 使用 */
    totalTokenUsage?: {
      promptTokens: number;
      completionTokens: number;
      totalTokens: number;
    };
    /** 总成本 */
    totalCost?: {
      amount: number;
      currency: string;
    };
  };
  /** 创建时间 */
  createdAt: string;
  /** 完成时间 */
  completedAt?: string;
}

// ============================================================================
// Benchmark
// ============================================================================

/**
 * Benchmark 配置
 */
export interface BenchmarkConfig {
  /** Benchmark 名称 */
  name: string;
  /** 描述 */
  description?: string;
  /** Agent IDs 列表 */
  agentIds: string[];
  /** 测试用例集 ID */
  testCaseSetId: string;
  /** Scorer 配置 */
  scorers: ScorerConfig[];
  /** 并发数 */
  concurrency?: number;
  /** 元数据 */
  metadata?: Record<string, any>;
}

/**
 * Benchmark 结果
 */
export interface BenchmarkResult {
  /** Benchmark ID */
  id: string;
  /** Benchmark 名称 */
  name: string;
  /** 状态 */
  status: EvalStatus;
  /** 各 Agent 的结果 */
  agentResults: {
    [agentId: string]: EvalResult;
  };
  /** 排行榜 */
  leaderboard: {
    agentId: string;
    agentName: string;
    avgScore: number;
    passRate: number;
    avgExecutionTime: number;
    rank: number;
  }[];
  /** 创建时间 */
  createdAt: string;
  /** 完成时间 */
  completedAt?: string;
}

// ============================================================================
// A/B Test
// ============================================================================

/**
 * A/B 测试配置
 */
export interface ABTestConfig {
  /** 测试名称 */
  name: string;
  /** 描述 */
  description?: string;
  /** Agent A ID */
  agentAId: string;
  /** Agent B ID */
  agentBId: string;
  /** 测试用例集 ID */
  testCaseSetId: string;
  /** Scorer 配置 */
  scorers: ScorerConfig[];
  /** 显著性水平（p-value） */
  significanceLevel?: number;
  /** 元数据 */
  metadata?: Record<string, any>;
}

/**
 * A/B 测试结果
 */
export interface ABTestResult {
  /** 测试 ID */
  id: string;
  /** 测试名称 */
  name: string;
  /** 状态 */
  status: EvalStatus;
  /** Agent A 结果 */
  agentAResult: EvalResult;
  /** Agent B 结果 */
  agentBResult: EvalResult;
  /** 统计分析 */
  statisticalAnalysis: {
    /** Agent A 平均分 */
    agentAAvgScore: number;
    /** Agent B 平均分 */
    agentBAvgScore: number;
    /** 差异 */
    difference: number;
    /** 差异百分比 */
    differencePercent: number;
    /** p-value */
    pValue: number;
    /** 是否显著 */
    isSignificant: boolean;
    /** 胜者 */
    winner?: "A" | "B" | "tie";
  };
  /** 创建时间 */
  createdAt: string;
  /** 完成时间 */
  completedAt?: string;
}

// ============================================================================
// Eval Filtering and Pagination
// ============================================================================

/**
 * Eval 过滤器
 */
export interface EvalFilter {
  /** 状态 */
  status?: EvalStatus;
  /** Eval 类型 */
  type?: EvalType;
  /** Agent ID */
  agentId?: string;
  /** 创建时间范围 */
  createdAfter?: string;
  createdBefore?: string;
  /** 排序字段 */
  sortBy?: "createdAt" | "completedAt" | "avgScore" | "passRate";
  /** 排序方向 */
  sortOrder?: "asc" | "desc";
  /** 分页 */
  page?: number;
  pageSize?: number;
}

/**
 * 分页响应
 */
export interface PaginatedEvalResponse {
  /** Eval 列表 */
  items: EvalInfo[];
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
// Eval Reports
// ============================================================================

/**
 * 报告格式
 */
export type ReportFormat = "json" | "html" | "markdown" | "csv";

/**
 * 报告请求
 */
export interface EvalReportRequest {
  /** Eval ID */
  evalId: string;
  /** 报告格式 */
  format: ReportFormat;
  /** 包含详细信息 */
  includeDetails?: boolean;
  /** 包含可视化 */
  includeVisualization?: boolean;
}

/**
 * 报告结果
 */
export interface EvalReportResult {
  /** 报告内容 */
  content: string;
  /** 格式 */
  format: ReportFormat;
  /** 生成时间 */
  generatedAt: string;
}
