/**
 * Tool 相关类型定义
 * 内置工具和长时运行工具
 */

// ============================================================================
// Tool Types
// ============================================================================

/**
 * 工具类型
 */
export type ToolType =
  | "builtin" // 内置工具
  | "custom" // 自定义工具
  | "mcp"; // MCP 工具

/**
 * 工具分类
 */
export type ToolCategory =
  | "system" // 系统工具
  | "file" // 文件操作
  | "network" // 网络请求
  | "data" // 数据处理
  | "code" // 代码执行
  | "integration" // 第三方集成
  | "other"; // 其他

/**
 * 内置工具名称
 */
export type BuiltinToolName =
  | "bash" // Bash 命令执行
  | "python" // Python 代码执行
  | "http_request" // HTTP 请求
  | "file_read" // 文件读取
  | "file_write" // 文件写入
  | "web_scraper" // 网页抓取
  | "database_query"; // 数据库查询

// ============================================================================
// Tool Info
// ============================================================================

/**
 * 工具信息
 */
export interface ToolInfo {
  /** 工具名称 */
  name: string;
  /** 显示名称 */
  displayName: string;
  /** 描述 */
  description: string;
  /** 类型 */
  type: ToolType;
  /** 分类 */
  category: ToolCategory;
  /** 是否启用 */
  enabled: boolean;
  /** 是否需要审批 */
  requiresApproval: boolean;
  /** 参数 Schema */
  parametersSchema: ToolParametersSchema;
  /** 返回值 Schema */
  returnSchema?: ToolReturnSchema;
  /** 是否为长时运行工具 */
  isLongRunning: boolean;
  /** 超时时间（秒） */
  timeout?: number;
}

/**
 * 工具参数 Schema
 */
export interface ToolParametersSchema {
  /** Schema 类型 */
  type: string;
  /** 属性定义 */
  properties: Record<string, ToolParameter>;
  /** 必需参数 */
  required?: string[];
}

/**
 * 工具参数
 */
export interface ToolParameter {
  /** 类型 */
  type: string;
  /** 描述 */
  description: string;
  /** 默认值 */
  default?: any;
  /** 枚举值 */
  enum?: any[];
  /** 示例 */
  examples?: any[];
  /** 格式 */
  format?: string;
}

/**
 * 工具返回值 Schema
 */
export interface ToolReturnSchema {
  /** Schema 类型 */
  type: string;
  /** 描述 */
  description?: string;
  /** 属性定义 */
  properties?: Record<string, any>;
}

// ============================================================================
// Tool Execution
// ============================================================================

/**
 * 工具执行请求
 */
export interface ToolExecutionRequest {
  /** 工具名称 */
  toolName: string;
  /** 参数 */
  params: Record<string, any>;
  /** 超时时间（毫秒） */
  timeout?: number;
  /** 是否异步执行 */
  async?: boolean;
}

/**
 * 工具执行响应
 */
export interface ToolExecutionResponse {
  /** 是否成功 */
  success: boolean;
  /** 结果 */
  result?: any;
  /** 错误信息 */
  error?: string;
  /** 执行时间（毫秒） */
  executionTime: number;
  /** 输出日志 */
  logs?: string[];
}

/**
 * 异步工具执行响应
 */
export interface AsyncToolExecutionResponse {
  /** 任务 ID */
  taskId: string;
  /** 状态 */
  status: "pending" | "running";
  /** 创建时间 */
  createdAt: string;
}

// ============================================================================
// Long-Running Task
// ============================================================================

/**
 * 长时运行任务状态
 */
export type TaskStatus =
  | "pending" // 等待中
  | "running" // 运行中
  | "completed" // 完成
  | "failed" // 失败
  | "cancelled"; // 已取消

/**
 * 任务进度信息
 */
export interface TaskProgress {
  /** 任务 ID */
  taskId: string;
  /** 工具名称 */
  toolName: string;
  /** 状态 */
  status: TaskStatus;
  /** 进度（0-100） */
  progress: number;
  /** 进度消息 */
  message?: string;
  /** 开始时间 */
  startedAt: string;
  /** 完成时间 */
  completedAt?: string;
  /** 结果（如果完成） */
  result?: any;
  /** 错误（如果失败） */
  error?: string;
  /** 输出日志 */
  logs?: string[];
}

// ============================================================================
// Specific Tool Configs
// ============================================================================

/**
 * Bash 工具配置
 */
export interface BashToolParams {
  /** 要执行的命令 */
  command: string;
  /** 工作目录 */
  workDir?: string;
  /** 环境变量 */
  env?: Record<string, string>;
  /** 超时时间（秒） */
  timeout?: number;
}

/**
 * Python 工具配置
 */
export interface PythonToolParams {
  /** Python 代码 */
  code: string;
  /** 是否使用虚拟环境 */
  useVenv?: boolean;
  /** 依赖包列表 */
  requirements?: string[];
  /** 超时时间（秒） */
  timeout?: number;
}

/**
 * HTTP 请求工具配置
 */
export interface HttpRequestToolParams {
  /** URL */
  url: string;
  /** 方法 */
  method: "GET" | "POST" | "PUT" | "DELETE" | "PATCH";
  /** Headers */
  headers?: Record<string, string>;
  /** 请求体 */
  body?: any;
  /** 超时时间（秒） */
  timeout?: number;
}

/**
 * 文件读取工具配置
 */
export interface FileReadToolParams {
  /** 文件路径 */
  path: string;
  /** 编码 */
  encoding?: string;
}

/**
 * 文件写入工具配置
 */
export interface FileWriteToolParams {
  /** 文件路径 */
  path: string;
  /** 内容 */
  content: string;
  /** 编码 */
  encoding?: string;
  /** 是否追加 */
  append?: boolean;
}

/**
 * Web Scraper 工具配置
 */
export interface WebScraperToolParams {
  /** URL */
  url: string;
  /** CSS 选择器 */
  selectors?: string[];
  /** 是否执行 JavaScript */
  executeJs?: boolean;
  /** 等待时间（毫秒） */
  waitTime?: number;
}

/**
 * 数据库查询工具配置
 */
export interface DatabaseQueryToolParams {
  /** 数据库连接 */
  connection: string;
  /** SQL 查询 */
  query: string;
  /** 参数 */
  params?: any[];
}

// ============================================================================
// Tool Statistics
// ============================================================================

/**
 * 工具统计信息
 */
export interface ToolStats {
  /** 工具名称 */
  toolName: string;
  /** 总调用次数 */
  totalCalls: number;
  /** 成功次数 */
  successCount: number;
  /** 失败次数 */
  failureCount: number;
  /** 平均执行时间（毫秒） */
  avgExecutionTime: number;
  /** 最后调用时间 */
  lastCalledAt?: string;
}

/**
 * 工具使用报告
 */
export interface ToolUsageReport {
  /** 时间范围 */
  timeRange: {
    start: string;
    end: string;
  };
  /** 总调用次数 */
  totalCalls: number;
  /** 工具统计列表 */
  toolStats: ToolStats[];
  /** 最常用的工具 */
  topTools: Array<{
    toolName: string;
    callCount: number;
    percentage: number;
  }>;
}
