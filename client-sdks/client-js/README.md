# aster Client JS

> JavaScript/TypeScript SDK for aster - Full-featured AI Agent client

## 🎯 当前状态

- **版本**: v0.7.0 (Beta)
- **Client SDK**: 100% (174 个 API 已实现)
- **后端 API**: 7.5% (仅 13 个端点可用)
- **实际可用度**: **7.5%** ⚠️

## ⚠️ 重要警告

**Client SDK 已完成所有 174 个 API 的实现，但后端仅提供 13 个 HTTP 端点！**

这意味着 **92.5% 的功能目前无法使用**，需要等待后端 API 开发完成。

### 当前可用功能（7.5%）

✅ **Chat 功能** - 完全可用

- 同步对话 (`agents.chat`)
- 流式对话 (`agents.chatStream`)

✅ **Skills 管理** - 完全可用

- Skills 列表、安装、卸载
- 版本管理

🟡 **Evals** - 部分可用（11%）

- 文本评估、Session 评估、批量评估
- ❌ Test Case 管理、Benchmark、A/B Test

🟡 **Semantic Search** - 部分可用（17%）

- ✅ 搜索记忆
- ❌ 存储、删除记忆

### 不可用功能（92.5%）

❌ Agent CRUD、状态管理、统计（91% 不可用）
❌ Session 管理（100% 不可用）
❌ Memory 管理（95% 不可用）
❌ Workflow 管理（100% 不可用）
❌ Tool 管理（100% 不可用）
❌ MCP 管理（94% 不可用）
❌ Middleware（100% 不可用）
❌ Telemetry（100% 不可用）

**详细信息**:

- [API 可用性状态](./API_AVAILABILITY.md) - 详细的功能可用性
- [后端 API 缺失分析](/BACKEND_API_GAP_ANALYSIS.md) - 缺失功能分析
- [后端 API 路线图](/BACKEND_API_ROADMAP.md) - 开发计划（预计 12-15 周）

---

## ✨ 特性

### Client SDK 实现状态（100% 完成）

> ⚠️ 注意：以下功能已在 Client SDK 中实现，但大部分需要后端 API 支持才能使用。

#### 事件驱动架构（后端部分支持）

- ✅ 三通道事件系统（Progress/Control/Monitor）
- ✅ WebSocket 客户端（自动重连、心跳）
- ✅ AsyncIterable 事件订阅
- **后端状态**: 🟡 SSE 可用，WebSocket 需开发

#### Agent 管理（后端 9% 可用）

- ✅ CRUD + 状态管理（Client SDK 完成）
- ✅ 同步对话（**后端可用** ✅）
- ✅ 流式对话（**后端可用** ✅）
- ✅ 7 种模板（Client SDK 完成）
- ✅ 统计和批量操作（Client SDK 完成）
- **后端状态**: 🔴 仅 Chat 可用，其他需开发

#### Session 管理（后端 0% 可用）

- ✅ CRUD + 消息管理（Client SDK 完成）
- ✅ 7 段断点恢复机制（Client SDK 完成）
- ✅ 统计和多格式导出（Client SDK 完成）
- **后端状态**: 🔴 完全不可用，需开发

#### 三层记忆系统（后端 5% 可用）

- ✅ Working Memory（Client SDK 完成）
- ✅ Semantic Memory（Client SDK 完成）
  - ✅ 搜索（**后端可用** ✅）
  - ❌ 存储、删除（后端需开发）
- ✅ Provenance（Client SDK 完成）
- ✅ Consolidation（Client SDK 完成）
- **后端状态**: 🔴 仅 search 可用，其他需开发

#### Workflow 编排（后端 0% 可用）

- ✅ Parallel/Sequential/Loop（Client SDK 完成）
- ✅ 执行控制（暂停/恢复/取消）（Client SDK 完成）
- **后端状态**: 🔴 仅 Demo 可用，完整功能需开发

#### MCP 协议（后端 6% 可用）

- ✅ Server 管理（Client SDK 完成）
- ✅ 远程工具调用（Client SDK 完成）
- ✅ 资源和 Prompt 访问（Client SDK 完成）
- **后端状态**: 🔴 仅 MCP Server 可用，管理 API 需开发

#### Middleware 系统（后端 0% 可用）

- ✅ 8 个内置 Middleware（Client SDK 完成）
- ✅ 洋葱模型（Client SDK 完成）
- ✅ 灵活配置（Client SDK 完成）
- **后端状态**: 🔴 完全不可用，需开发

#### Tool 系统（后端 0% 可用）

- ✅ 7 个内置工具（Client SDK 完成）
- ✅ 同步/异步执行（Client SDK 完成）
- ✅ 长时运行任务管理（Client SDK 完成）
- **后端状态**: 🔴 完全不可用，需开发

#### Eval 系统（后端 11% 可用）

- ✅ 5 种 Eval 类型（Client SDK 完成）
- ✅ 10 种 Scorer（Client SDK 完成）
- ✅ Text/Session/Batch Eval（**后端可用** ✅）
- ✅ Benchmark 和 A/B 测试（Client SDK 完成）
- ✅ 报告生成（Client SDK 完成）
- **后端状态**: 🟡 基础 Eval 可用，高级功能需开发

#### Telemetry（后端 0% 可用）

- ✅ Metrics/Traces/Logs（Client SDK 完成）
- ✅ 健康检查（Client SDK 完成）
- ✅ 性能和成本统计（Client SDK 完成）
- **后端状态**: 🔴 完全不可用，需开发

#### Skills 管理（后端 100% 可用）✅

- ✅ Skills 列表、安装、卸载（**后端可用** ✅）
- ✅ 版本管理（**后端可用** ✅）
- **后端状态**: ✅ 完全可用

### 后端开发计划

- Week 4: Agent CRUD + Session 管理
- Week 8: Memory 系统 + Workflow 完整
- Week 12: Tool 管理 + Eval 扩展
- Week 15: MCP + Middleware + Telemetry

---

## 📦 安装

```bash
npm install @aster/client-js
```

或使用其他包管理器：

```bash
pnpm add @aster/client-js
yarn add @aster/client-js
```

---

## 🚀 快速开始

### 基础 Chat

```typescript
import { AsterClient } from "@aster/client-js";

// 创建客户端
const client = new AsterClient({
  baseUrl: "http://localhost:8080",
  apiKey: process.env.ASTER_API_KEY,
});

// 同步 Chat
const response = await client.agent.chat({
  templateId: "assistant",
  input: "What is the capital of France?",
  messages: [],
});

console.log(response.text); // "Paris is the capital of France."
```

### 流式响应（v0.1.0 部分支持）

```typescript
// 流式 Chat
for await (const event of client.agent.stream({
  templateId: "assistant",
  input: "Tell me a long story",
})) {
  if (event.type === "text_chunk") {
    process.stdout.write(event.data.delta);
  }
}
```

### Skills 管理（v0.1.0 ✅）

```typescript
// 列出所有 Skills
const skills = await client.skill.list();

// 创建 Skill
await client.skill.create({
  id: 'my-skill',
  files: [
    { path: 'SKILL.md', content: '...' },
    { path: 'script.sh', content: '...' }
  ]
});

// 获取 Skill 详情
const skill = await client.skill.get('my-skill');

// 删除 Skill
await client.skill.delete('my-skill');

// Skill 版本管理
const versions = await client.skill.listVersions('my-skill');
await client.skill.createVersion('my-skill', 'v2.0', { ... });
await client.skill.deleteVersion('my-skill', 'v1.0');
```

---

## 📚 核心功能（v0.5.0+）

### 事件驱动架构 ⭐

**三通道设计**是 aster 的核心：

```typescript
// 订阅事件（三通道）
const subscription = await client.agent.subscribe(
  ["progress", "control", "monitor"],
  {
    agentId: "agent-123",
    eventTypes: ["thinking", "text_chunk", "tool_start"],
  },
);

// 处理事件
for await (const event of subscription) {
  switch (event.channel) {
    case "progress":
      // 数据流：思考、文本、工具执行
      if (event.type === "thinking") {
        console.log("AI 正在思考:", event.data.content);
      } else if (event.type === "text_chunk") {
        process.stdout.write(event.data.delta);
      } else if (event.type === "tool_start") {
        console.log("调用工具:", event.data.toolName);
      }
      break;

    case "control":
      // 审批流：工具审批、暂停/恢复
      if (event.type === "tool_approval_request") {
        console.log("需要审批工具:", event.data.toolName);
        // 审批或拒绝
        await client.security.approve(event.data.approvalId);
      }
      break;

    case "monitor":
      // 治理流：Token、成本、合规
      if (event.type === "token_usage") {
        console.log("Token 使用:", event.data.tokens);
      } else if (event.type === "cost") {
        console.log("成本:", event.data.cost);
      }
      break;
  }
}

// 取消订阅
subscription.unsubscribe();
```

**支持的事件类型（20+）**:

**Progress Channel**:

- `thinking` - 思考过程
- `text_chunk` - 流式文本
- `tool_start` / `tool_end` - 工具执行
- `done` / `error` - 完成/错误

**Control Channel**:

- `tool_approval_request` / `tool_approval_response` - 工具审批
- `pause` / `resume` - 暂停/恢复

**Monitor Channel**:

- `token_usage` - Token 使用
- `latency` - 延迟
- `cost` - 成本
- `compliance` - 合规检查

---

### Working Memory ⭐

**LLM 可主动更新的工作记忆**：

```typescript
// 设置工作记忆
await client.memory.working.set(
  "user_preference",
  {
    theme: "dark",
    language: "zh-CN",
    notifications: true,
  },
  {
    scope: "thread", // 'thread' 或 'resource'
    ttl: 3600, // 1 小时后过期
    schema: {
      // JSON Schema 验证
      type: "object",
      properties: {
        theme: { type: "string", enum: ["light", "dark"] },
        language: { type: "string" },
        notifications: { type: "boolean" },
      },
      required: ["theme", "language"],
    },
  },
);

// 获取工作记忆
const preference = await client.memory.working.get("user_preference", "thread");
console.log(preference); // { theme: 'dark', language: 'zh-CN', ... }

// 列出所有工作记忆
const allMemories = await client.memory.working.list("thread");

// 删除工作记忆
await client.memory.working.delete("user_preference", "thread");
```

**特性**:

- ✅ **双作用域**: Thread（会话级）和 Resource（全局）
- ✅ **LLM 可主动更新**: 通过内置工具
- ✅ **自动加载**: 自动添加到 system prompt
- ✅ **JSON Schema 验证**: 确保数据结构正确
- ✅ **TTL 过期**: 自动清理过期数据

---

### Semantic Memory

**向量检索和语义搜索**：

```typescript
// 存储记忆
const chunkId = await client.memory.semantic.store(
  "Paris is the capital of France.",
  {
    source: "wikipedia",
    category: "geography",
  },
);

// 语义搜索
const results = await client.memory.semantic.search(
  "What is the capital of France?",
  {
    limit: 10,
    threshold: 0.8,
    filter: { category: "geography" },
  },
);

console.log(results);
// [
//   {
//     id: 'chunk-123',
//     content: 'Paris is the capital of France.',
//     score: 0.95,
//     metadata: { source: 'wikipedia', category: 'geography' }
//   }
// ]

// 删除记忆
await client.memory.semantic.delete("chunk-123");
```

---

### Session 管理

**完整的会话生命周期管理**：

```typescript
// 创建会话
const session = await client.session.create({
  agentId: "agent-123",
  templateId: "assistant",
  metadata: {
    userId: "user-456",
    project: "demo",
  },
});

// 获取会话详情
const sessionInfo = await client.session.get(session.id);

// 获取会话消息
const messages = await client.session.getMessages(session.id, {
  page: 1,
  pageSize: 20,
});

// 断点恢复（7 段断点机制）
const checkpoints = await client.session.getCheckpoints(session.id);
await client.session.resume(session.id, checkpoints[0].id);

// 会话统计
const stats = await client.session.getStats(session.id);
console.log(stats);
// {
//   totalMessages: 42,
//   totalTokens: 15234,
//   totalCost: 0.23,
//   duration: 3600  // 秒
// }

// 删除会话
await client.session.delete(session.id);
```

---

### Workflow 系统

**三种工作流模式**：

```typescript
// 1. Parallel Workflow（并行执行）
const parallelWorkflow = await client.workflow.create({
  type: "parallel",
  name: "Multi-Agent Research",
  agents: [
    { id: "researcher-1", task: "Research topic A" },
    { id: "researcher-2", task: "Research topic B" },
    { id: "researcher-3", task: "Research topic C" },
  ],
  maxConcurrency: 3,
});

// 2. Sequential Workflow（顺序执行）
const sequentialWorkflow = await client.workflow.create({
  type: "sequential",
  name: "Document Processing Pipeline",
  steps: [
    { agent: "reader", action: "read_document" },
    { agent: "analyzer", action: "analyze_content" },
    { agent: "summarizer", action: "generate_summary" },
  ],
});

// 3. Loop Workflow（循环执行）
const loopWorkflow = await client.workflow.create({
  type: "loop",
  name: "Iterative Improvement",
  agent: "optimizer",
  condition: (result) => result.quality < 0.95,
  maxIterations: 10,
});

// 执行工作流
const run = await client.workflow.execute(parallelWorkflow.id, {
  input: "Research AI trends in 2024",
});

// 暂停工作流
await client.workflow.suspend(parallelWorkflow.id, run.id);

// 恢复工作流
await client.workflow.resume(parallelWorkflow.id, run.id);

// 获取执行历史
const runs = await client.workflow.getRuns(parallelWorkflow.id);
const runDetails = await client.workflow.getRunDetails(
  parallelWorkflow.id,
  run.id,
);
```

---

### 其他核心资源

#### MCP 协议

```typescript
// 添加 MCP 服务器
await client.mcp.addServer({
  serverId: "my-mcp-server",
  endpoint: "http://localhost:8090/mcp",
  accessKeyId: "key",
  accessKeySecret: "secret",
});

// 列出服务器工具
const tools = await client.mcp.getServerTools("my-mcp-server");

// 调用远程工具
const result = await client.mcp.callTool("my-mcp-server", "calculator", {
  operation: "add",
  numbers: [1, 2, 3],
});
```

#### Middleware 配置

```typescript
// 列出可用中间件
const middlewares = await client.middleware.list();

// 获取中间件配置
const config = await client.middleware.get("summarization");

// 更新中间件配置
await client.middleware.update("summarization", {
  threshold: 170000, // Token 阈值
  keepMessages: 6, // 保留最近 N 条消息
});
```

#### 工具执行

```typescript
// 列出所有工具
const tools = await client.tool.list();

// 同步执行工具
const result = await client.tool.execute("bash", {
  command: "ls -la",
});

// 异步执行（长时运行工具）
const taskId = await client.tool.executeAsync("web_scraper", {
  url: "https://example.com",
});

// 查询任务进度
const progress = await client.tool.getTaskProgress(taskId);
console.log(progress);
// {
//   status: 'running',
//   progress: 45,
//   message: 'Scraping page 45/100'
// }

// 取消任务
await client.tool.cancelTask(taskId);
```

#### Telemetry

```typescript
// 获取追踪数据
const traces = await client.telemetry.getTraces({
  startTime: "2024-01-01T00:00:00Z",
  endTime: "2024-01-02T00:00:00Z",
  agentId: "agent-123",
});

// 获取单个追踪详情
const trace = await client.telemetry.getTrace("trace-456");

// 获取指标
const metrics = await client.telemetry.getMetrics({
  name: "token_usage",
  timeRange: "1h",
});

// 导出遥测数据
const exportResult = await client.telemetry.export({
  format: "json",
  timeRange: "24h",
});
```

---

## 🔧 配置

### 客户端配置

```typescript
const client = new AsterClient({
  // 基础配置
  baseUrl: "http://localhost:8080",
  apiKey: process.env.ASTER_API_KEY,

  // 超时配置
  timeout: 120000, // 全局超时（毫秒）

  // Retry 配置
  retry: {
    maxRetries: 3,
    retryableStatusCodes: [408, 429, 500, 502, 503, 504],
    backoffMultiplier: 2,
    maxBackoffTime: 30000,
  },

  // 日志配置
  logging: {
    level: "info", // 'debug' | 'info' | 'warn' | 'error'
    format: "json", // 'json' | 'text'
  },

  // 自定义 headers
  headers: {
    "X-Custom-Header": "value",
  },
});
```

### 环境变量

```bash
# API 配置
ASTER_BASE_URL=http://localhost:8080
ASTER_API_KEY=your_api_key

# 可选配置
ASTER_TIMEOUT=120000
ASTER_MAX_RETRIES=3
ASTER_LOG_LEVEL=info
```

---

## 🧪 测试

```bash
# 运行所有测试
npm test

# 运行单元测试
npm run test:unit

# 运行集成测试
npm run test:integration

# 测试覆盖率
npm run test:coverage

# 监听模式
npm run test:watch
```

---

## 📖 API 文档

### 完整 API 参考

查看自动生成的 API 文档：

```bash
npm run docs
```

或访问在线文档：[API Reference](https://wordflowlab.github.io/aster/client-js/)

### 资源列表

| 资源         | 状态      | 端点数 | 说明              |
| ------------ | --------- | ------ | ----------------- |
| `agent`      | ✅ 部分   | 2/7    | Agent 管理和 Chat |
| `memory`     | 🚧 开发中 | 0/8    | 三层记忆系统      |
| `workflow`   | 🚧 开发中 | 0/8    | 工作流编排        |
| `session`    | 🚧 开发中 | 0/7    | 会话管理          |
| `skill`      | ✅ 完成   | 6/6    | 技能管理          |
| `eval`       | ✅ 部分   | 3/8    | 评估系统          |
| `tool`       | 🚧 开发中 | 0/6    | 工具执行          |
| `mcp`        | 🚧 开发中 | 1/5    | MCP 协议          |
| `middleware` | 🚧 开发中 | 0/3    | 中间件配置        |
| `telemetry`  | 🚧 开发中 | 0/5    | 可观测性          |
| `router`     | 📅 计划中 | 0/3    | 模型路由          |
| `sandbox`    | 📅 计划中 | 0/5    | 沙箱管理          |
| `provider`   | 📅 计划中 | 0/4    | Provider 管理     |
| `template`   | 📅 计划中 | 0/5    | 模板管理          |
| `security`   | 📅 计划中 | 0/4    | 安全系统          |

---

## 🗺️ 路线图

### v0.5.0 (Week 6) - 核心架构

- ✅ 事件驱动架构
- ✅ 三层记忆系统
- ✅ Session + Workflow
- ✅ MCP + Middleware
- ✅ Tool + Telemetry

### v0.8.0 (Week 10) - 高级功能

- ✅ Router + Sandbox
- ✅ Pool/Room + Evals 扩展
- ✅ Provider + Template

### v1.0.0 (Week 13) - 生产就绪 ✨

- ✅ Commands + Security + Store
- ✅ 完整文档和示例
- ✅ 100% API 覆盖

详细路线图：[TODO.md](./TODO.md)

---

## 📚 相关文档

- [TODO.md](./TODO.md) - 详细开发任务
- [ROADMAP.md](../ROADMAP.md) - 完整路线图
- [ARCHITECTURE.md](../ARCHITECTURE.md) - 架构设计
- [MISSING_FEATURES.md](../MISSING_FEATURES.md) - 遗漏功能分析

---

## 🤝 贡献

欢迎贡献！请查看 [TODO.md](./TODO.md) 了解当前开发任务。

---

## 📄 许可证

[MIT](../../LICENSE)

---

**最后更新**: 2024年11月17日
**版本**: v0.1.0 → v1.0.0 (计划中)
**状态**: 🚧 核心架构开发中
**预计完成**: 10-13 周
