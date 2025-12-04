# aster Client JS - API 文档

> **版本**: v0.5.0
> **更新时间**: 2024年11月17日

---

## 目录

- [aster 客户端](#aster-客户端)
- [Memory 资源](#memory-资源)
- [Session 资源](#session-资源)
- [Workflow 资源](#workflow-资源)
- [MCP 资源](#mcp-资源)
- [Middleware 资源](#middleware-资源)
- [Tool 资源](#tool-资源)
- [Telemetry 资源](#telemetry-资源)

---

## aster 客户端

### 初始化

```typescript
import { aster } from "@aster/client-js";

const client = new aster({
  baseUrl: "http://localhost:8080",
  apiKey: "your-api-key",
  timeout: 30000, // 可选，默认 30000ms
  retry: {
    // 可选
    maxRetries: 3,
    retryDelay: 1000,
  },
});
```

### 配置选项

| 参数               | 类型   | 必需 | 说明                 |
| ------------------ | ------ | ---- | -------------------- |
| `baseUrl`          | string | ✅   | aster 服务器地址     |
| `apiKey`           | string | ❌   | API 密钥             |
| `timeout`          | number | ❌   | 请求超时时间（毫秒） |
| `retry.maxRetries` | number | ❌   | 最大重试次数         |
| `retry.retryDelay` | number | ❌   | 重试延迟（毫秒）     |

### 便捷方法

```typescript
// 健康检查
const health = await client.healthCheck();

// 获取系统状态
const status = await client.getStatus();

// 获取使用统计
const usage = await client.getUsageStatistics({
  start: "2024-01-01T00:00:00Z",
  end: "2024-01-08T00:00:00Z",
});

// 获取性能指标
const perf = await client.getPerformanceMetrics({
  start: "2024-01-01T00:00:00Z",
  end: "2024-01-02T00:00:00Z",
});
```

---

## Memory 资源

### Working Memory

#### set() - 设置 Working Memory

```typescript
await client.memory.working.set(
  "user_preference",
  { theme: "dark", language: "zh-CN" },
  {
    scope: "resource", // 'thread' | 'resource'
    ttl: 3600, // 过期时间（秒）
    schema: {
      // 可选，JSON Schema 验证
      type: "object",
      properties: {
        theme: { type: "string" },
        language: { type: "string" },
      },
    },
  },
);
```

#### get() - 获取 Working Memory

```typescript
const item = await client.memory.working.get("user_preference");
console.log(item?.value); // { theme: 'dark', language: 'zh-CN' }
```

#### delete() - 删除 Working Memory

```typescript
await client.memory.working.delete("user_preference");
```

#### list() - 列出所有 Working Memory

```typescript
const items = await client.memory.working.list({
  scope: "resource",
  prefix: "user_",
});
```

### Semantic Memory

#### store() - 存储语义记忆

```typescript
await client.memory.semantic.store(
  "aster is a powerful framework for building AI agents",
  {
    source: "documentation",
    category: "introduction",
    timestamp: new Date().toISOString(),
  },
);
```

#### search() - 语义搜索

```typescript
const chunks = await client.memory.semantic.search("What is aster?", {
  limit: 5,
  threshold: 0.7,
  filter: {
    category: "introduction",
  },
});
```

#### delete() - 删除语义记忆

```typescript
await client.memory.semantic.delete("chunk-id");
```

### Provenance - 记忆溯源

```typescript
// 获取溯源链
const chain = await client.memory.provenance.getChain("memory-id");

// 列出所有溯源
const items = await client.memory.provenance.list({
  memoryType: "semantic",
  startTime: "2024-01-01T00:00:00Z",
});
```

### Consolidation - 记忆合并

```typescript
// 合并记忆
await client.memory.consolidation.consolidate({
  sourceIds: ["chunk-1", "chunk-2", "chunk-3"],
  strategy: "llm_based",
  targetScope: "resource",
});

// 获取合并历史
const history = await client.memory.consolidation.getHistory({
  startTime: "2024-01-01T00:00:00Z",
  limit: 10,
});
```

---

## Session 资源

### create() - 创建 Session

```typescript
const session = await client.sessions.create({
  agentId: "agent-123",
  templateId: "chat-template",
  userId: "user-456",
  metadata: {
    project: "demo",
  },
  enableCheckpoints: true,
  checkpointInterval: 5, // 每5条消息创建断点
});
```

### get() - 获取 Session

```typescript
const session = await client.sessions.get("session-id");
```

### list() - 列出 Sessions

```typescript
const result = await client.sessions.list({
  status: "active",
  page: 1,
  pageSize: 20,
});

console.log(`总计: ${result.total}`);
result.items.forEach((session) => {
  console.log(session.id, session.status);
});
```

### addMessage() - 添加消息

```typescript
await client.sessions.addMessage("session-id", {
  role: "user",
  content: "Hello, how can you help me?",
});

await client.sessions.addMessage("session-id", {
  role: "assistant",
  content: "I can help you with various tasks.",
});
```

### getMessages() - 获取消息列表

```typescript
const messages = await client.sessions.getMessages("session-id", {
  page: 1,
  pageSize: 50,
  sort: "asc",
});
```

### Checkpoint 操作

```typescript
// 创建手动断点
const checkpoint = await client.sessions.createCheckpoint(
  "session-id",
  "before-important-action",
);

// 获取所有断点
const checkpoints = await client.sessions.getCheckpoints("session-id");

// 从断点恢复
await client.sessions.resume("session-id", {
  checkpointId: checkpoint.id,
  keepSubsequentMessages: false,
});
```

### 状态管理

```typescript
// 暂停 Session
await client.sessions.pause("session-id");

// 激活 Session
await client.sessions.activate("session-id");

// 完成 Session
await client.sessions.complete("session-id");

// 归档 Session
await client.sessions.archive("session-id");
```

---

## Workflow 资源

### create() - 创建 Workflow

#### Parallel Workflow（并行）

```typescript
const workflow = await client.workflows.create({
  type: "parallel",
  name: "Multi-Agent Research",
  description: "多个 Agent 并行研究",
  agents: [
    { id: "researcher-1", task: "Research AI trends" },
    { id: "researcher-2", task: "Research quantum computing" },
    { id: "researcher-3", task: "Research climate tech" },
  ],
  maxConcurrency: 3,
  timeout: 300,
});
```

#### Sequential Workflow（顺序）

```typescript
const workflow = await client.workflows.create({
  type: "sequential",
  name: "Document Processing",
  description: "文档处理流水线",
  steps: [
    { agent: "reader", action: "read_document" },
    { agent: "analyzer", action: "analyze_content" },
    { agent: "summarizer", action: "generate_summary" },
  ],
  continueOnError: false,
});
```

#### Loop Workflow（循环）

```typescript
const workflow = await client.workflows.create({
  type: "loop",
  name: "Code Optimizer",
  description: "迭代优化代码",
  agent: "optimizer",
  condition: "result.quality < 0.95",
  maxIterations: 10,
  initialInput: { code: "..." },
});
```

### execute() - 执行 Workflow

```typescript
const run = await client.workflows.execute("workflow-id", {
  input: { documentUrl: "https://example.com/doc.pdf" },
  options: {
    timeout: 300,
    async: false,
  },
});
```

### 执行控制

```typescript
// 暂停执行
await client.workflows.suspend("workflow-id", {
  runId: "run-id",
  reason: "User requested pause",
});

// 恢复执行
await client.workflows.resume("workflow-id", {
  runId: "run-id",
});

// 取消执行
await client.workflows.cancel("workflow-id", {
  runId: "run-id",
  reason: "User cancelled",
});
```

### 等待完成

```typescript
const result = await client.workflows.waitForCompletion(
  "workflow-id",
  "run-id",
  {
    pollInterval: 2000,
    timeout: 60000,
  },
);
```

---

## MCP 资源

### addServer() - 添加 MCP Server

```typescript
const server = await client.mcp.addServer({
  serverId: "my-server",
  name: "My MCP Server",
  endpoint: "http://localhost:8090/mcp",
  accessKeyId: "key",
  accessKeySecret: "secret",
  enabled: true,
});
```

### call() - 调用 MCP 工具

```typescript
const result = await client.mcp.call("server-id", "tool-name", {
  param1: "value1",
  param2: "value2",
});

console.log(result.success);
console.log(result.result);
console.log(result.executionTime, "ms");
```

### 管理 Servers

```typescript
// 列出所有 Servers
const servers = await client.mcp.listServers();

// 连接 Server
await client.mcp.connectServer("server-id");

// 断开 Server
await client.mcp.disconnectServer("server-id");

// 移除 Server
await client.mcp.removeServer("server-id");
```

---

## Middleware 资源

### list() - 列出所有 Middleware

```typescript
const middlewares = await client.middleware.list();

middlewares.forEach((mw) => {
  console.log(
    `[P${mw.priority}] ${mw.displayName} - ${mw.enabled ? "ON" : "OFF"}`,
  );
});
```

### 配置 Middleware

#### Summarization（上下文总结）

```typescript
await client.middleware.updateConfig("summarization", {
  threshold: 170000,
  keepMessages: 6,
  llmProvider: "anthropic",
  llmModel: "claude-sonnet-4",
});
```

#### Tool Approval（工具审批）

```typescript
await client.middleware.updateConfig("tool_approval", {
  approvalRequired: ["file_delete", "bash", "database_query"],
  autoApprove: ["file_read", "http_request"],
  approvalTimeout: 300,
  timeoutBehavior: "reject",
});
```

#### Cost Tracker（成本追踪）

```typescript
await client.middleware.updateConfig("cost_tracker", {
  enabled: true,
  costModel: "token_based",
  pricing: {
    promptTokenPrice: 0.003,
    completionTokenPrice: 0.015,
    currency: "USD",
  },
  budget: {
    daily: 100,
    monthly: 2000,
  },
});
```

### 管理执行顺序

```typescript
// 获取执行顺序
const order = await client.middleware.getExecutionOrder();

// 设置执行顺序
await client.middleware.setExecutionOrder([
  "logging",
  "telemetry",
  "cost_tracker",
  "token_limiter",
  "summarization",
]);

// 重置为默认顺序
await client.middleware.resetExecutionOrder();
```

---

## Tool 资源

### execute() - 执行工具（同步）

```typescript
const result = await client.tools.execute("bash", {
  command: "ls -la /tmp",
  workDir: "/tmp",
  timeout: 10,
});

console.log(result.success);
console.log(result.result);
console.log(result.executionTime, "ms");
```

### executeAsync() - 执行工具（异步）

```typescript
// 启动长时运行任务
const task = await client.tools.executeAsync("web_scraper", {
  url: "https://example.com",
  selectors: ["h1", "p"],
  executeJs: true,
});

console.log("Task ID:", task.taskId);

// 等待任务完成
const result = await client.tools.waitForTask(task.taskId, {
  pollInterval: 2000,
  timeout: 60000,
});

console.log("Status:", result.status);
console.log("Result:", result.result);
```

### 任务管理

```typescript
// 获取任务进度
const progress = await client.tools.getTaskProgress("task-id");

// 列出所有任务
const tasks = await client.tools.listTasks({
  status: "running",
});

// 取消任务
await client.tools.cancelTask("task-id");
```

### 工具管理

```typescript
// 列出所有工具
const tools = await client.tools.list({
  type: "builtin",
  enabled: true,
});

// 启用/禁用工具
await client.tools.enable("bash");
await client.tools.disable("database_query");
```

---

## Telemetry 资源

### Metrics

```typescript
// 列出所有 Metrics
const metrics = await client.telemetry.listMetrics();

// 查询 Metric 数据
const dataPoints = await client.telemetry.queryMetrics({
  name: "request_count",
  timeRange: {
    start: "2024-01-01T00:00:00Z",
    end: "2024-01-02T00:00:00Z",
  },
  aggregation: "sum",
});

// 记录自定义 Metric
await client.telemetry.recordMetric("custom_metric", 42, {
  label1: "value1",
});
```

### Traces

```typescript
// 查询 Traces
const traces = await client.telemetry.queryTraces({
  operationName: "workflow_execution",
  timeRange: {
    start: "2024-01-01T00:00:00Z",
    end: "2024-01-02T00:00:00Z",
  },
  minDuration: 1000,
  status: "error",
  limit: 10,
});

// 获取 Trace 详情
const trace = await client.telemetry.getTrace("trace-id");

// 获取 Trace 的所有 Spans
const spans = await client.telemetry.getTraceSpans("trace-id");
```

### Logs

```typescript
// 查询日志
const logs = await client.telemetry.queryLogs({
  level: "error",
  timeRange: {
    start: "2024-01-01T00:00:00Z",
    end: "2024-01-02T00:00:00Z",
  },
  search: "timeout",
  limit: 100,
});

// 写入日志
await client.telemetry.writeLog({
  level: "info",
  message: "Custom log message",
  source: "my-app",
  attributes: {
    userId: "user-123",
  },
});
```

### 导出

```typescript
// 导出 Metrics（Prometheus 格式）
const metricsExport = await client.telemetry.exportMetrics("prometheus", {
  start: "2024-01-01T00:00:00Z",
  end: "2024-01-02T00:00:00Z",
});

// 导出 Traces（OpenTelemetry 格式）
const tracesExport = await client.telemetry.exportTraces("opentelemetry", {
  start: "2024-01-01T00:00:00Z",
  end: "2024-01-02T00:00:00Z",
});

// 导出 Logs（CSV 格式）
const logsExport = await client.telemetry.exportLogs("csv", {
  start: "2024-01-01T00:00:00Z",
  end: "2024-01-02T00:00:00Z",
});
```

---

## 错误处理

所有 API 方法都可能抛出错误，建议使用 try-catch 处理：

```typescript
try {
  const result = await client.tools.execute("bash", {
    command: "invalid-command",
  });
} catch (error) {
  console.error("Error:", error.message);

  // 检查特定错误类型
  if (error.statusCode === 404) {
    console.error("Tool not found");
  } else if (error.statusCode === 429) {
    console.error("Rate limited");
  }
}
```

---

## 类型定义

所有类型都可以从主包导入：

```typescript
import {
  // Memory 类型
  WorkingMemoryItem,
  SemanticMemoryChunk,

  // Session 类型
  SessionInfo,
  Message,
  Checkpoint,

  // Workflow 类型
  WorkflowInfo,
  WorkflowRun,
  ParallelWorkflowDefinition,

  // 其他...
} from "@aster/client-js";
```

---

## 更多示例

查看 `examples/` 目录获取更多完整示例：

- `event-subscription.ts` - 事件订阅
- `memory-usage.ts` - 记忆系统
- `session-workflow.ts` - Session + Workflow
- `mcp-middleware-tool.ts` - MCP + Middleware + Tool
- `complete-usage.ts` - 完整功能演示

---

**文档版本**: v0.5.0
**最后更新**: 2024年11月17日
