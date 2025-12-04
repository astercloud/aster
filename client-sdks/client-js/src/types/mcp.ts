/**
 * MCP (Model Context Protocol) 相关类型定义
 * AsterClient 既可以作为 MCP Server，也可以作为 MCP Client
 */

// ============================================================================
// MCP Server Configuration
// ============================================================================

/**
 * MCP Server 配置
 */
export interface MCPServerConfig {
  /** Server ID */
  serverId: string;
  /** Server 名称 */
  name: string;
  /** Server 端点 */
  endpoint: string;
  /** 访问密钥 ID */
  accessKeyId?: string;
  /** 访问密钥 Secret */
  accessKeySecret?: string;
  /** 额外配置 */
  config?: Record<string, any>;
  /** 是否启用 */
  enabled?: boolean;
}

/**
 * MCP Server 信息
 */
export interface MCPServerInfo extends MCPServerConfig {
  /** 状态 */
  status: MCPServerStatus;
  /** 创建时间 */
  createdAt: string;
  /** 最后连接时间 */
  lastConnectedAt?: string;
  /** 可用工具数 */
  toolCount: number;
  /** 版本 */
  version?: string;
}

/**
 * MCP Server 状态
 */
export type MCPServerStatus =
  | "connected" // 已连接
  | "disconnected" // 已断开
  | "connecting" // 连接中
  | "error"; // 错误

// ============================================================================
// MCP Tools
// ============================================================================

/**
 * MCP 工具定义
 */
export interface MCPTool {
  /** 工具名称 */
  name: string;
  /** 描述 */
  description: string;
  /** 输入 Schema */
  inputSchema: MCPToolSchema;
  /** 输出 Schema */
  outputSchema?: MCPToolSchema;
  /** 工具类型 */
  type?: "function" | "api" | "command";
  /** 所属 Server */
  serverId: string;
}

/**
 * MCP 工具 Schema
 */
export interface MCPToolSchema {
  /** Schema 类型 */
  type: string;
  /** 属性定义 */
  properties?: Record<string, MCPSchemaProperty>;
  /** 必需字段 */
  required?: string[];
  /** 其他 JSON Schema 字段 */
  [key: string]: any;
}

/**
 * MCP Schema 属性
 */
export interface MCPSchemaProperty {
  /** 类型 */
  type: string;
  /** 描述 */
  description?: string;
  /** 枚举值 */
  enum?: any[];
  /** 默认值 */
  default?: any;
  /** 其他属性 */
  [key: string]: any;
}

// ============================================================================
// MCP Tool Execution
// ============================================================================

/**
 * MCP 工具调用请求
 */
export interface MCPToolCallRequest {
  /** Server ID */
  serverId: string;
  /** 工具名称 */
  toolName: string;
  /** 参数 */
  params: Record<string, any>;
  /** 超时时间（毫秒） */
  timeout?: number;
}

/**
 * MCP 工具调用响应
 */
export interface MCPToolCallResponse {
  /** 是否成功 */
  success: boolean;
  /** 结果 */
  result?: any;
  /** 错误信息 */
  error?: string;
  /** 执行时间（毫秒） */
  executionTime: number;
}

// ============================================================================
// MCP Resources
// ============================================================================

/**
 * MCP 资源
 */
export interface MCPResource {
  /** 资源 URI */
  uri: string;
  /** 资源名称 */
  name: string;
  /** 描述 */
  description?: string;
  /** MIME 类型 */
  mimeType?: string;
  /** 所属 Server */
  serverId: string;
}

/**
 * MCP 资源内容
 */
export interface MCPResourceContent {
  /** URI */
  uri: string;
  /** 内容 */
  content: string;
  /** MIME 类型 */
  mimeType: string;
}

// ============================================================================
// MCP Prompts
// ============================================================================

/**
 * MCP Prompt
 */
export interface MCPPrompt {
  /** Prompt 名称 */
  name: string;
  /** 描述 */
  description?: string;
  /** 参数 Schema */
  arguments?: MCPToolSchema;
  /** 所属 Server */
  serverId: string;
}

/**
 * MCP Prompt 结果
 */
export interface MCPPromptResult {
  /** 消息列表 */
  messages: Array<{
    role: "user" | "assistant" | "system";
    content: string;
  }>;
}

// ============================================================================
// MCP Statistics
// ============================================================================

/**
 * MCP 统计信息
 */
export interface MCPStats {
  /** 连接的 Server 数 */
  connectedServers: number;
  /** 总 Server 数 */
  totalServers: number;
  /** 可用工具数 */
  totalTools: number;
  /** 总调用次数 */
  totalCalls: number;
  /** 成功调用次数 */
  successfulCalls: number;
  /** 失败调用次数 */
  failedCalls: number;
  /** 平均响应时间（毫秒） */
  avgResponseTime: number;
}

// ============================================================================
// MCP Events
// ============================================================================

/**
 * MCP 事件类型
 */
export type MCPEventType = "server_connected" | "server_disconnected" | "tool_called" | "tool_completed" | "error";

/**
 * MCP 事件
 */
export interface MCPEvent {
  /** 事件类型 */
  type: MCPEventType;
  /** Server ID */
  serverId: string;
  /** 时间戳 */
  timestamp: string;
  /** 事件数据 */
  data?: any;
}
