---
title: 工作流 HTTP API 示例
description: 通过 /v1/workflows/demo/run 体验 Parallel/Sequential/Loop 工作流执行
navigation: false
---

# 工作流 HTTP API 示例

在已有工作流 Agent(ParallelAgent/SequentialAgent/LoopAgent) 的基础上, aster 提供了一个最小可用的 HTTP 演示接口, 用于快速体验工作流运行与事件结构:

- 路径: `POST /v1/workflows/demo/run`
- 作用: 运行内置的 demo 工作流(`sequential_demo`/`parallel_demo`/`loop_demo`/`nested_demo`), 返回事件列表。

> 注意: 该接口的设计, 主要是为了帮助你在前端或工具中快速查看工作流事件结构, 类似常见的 Workflow Run API 概念。

## 1. 启动 HTTP Server

确保使用 CLI 启动 `aster serve`:

```bash
aster serve --addr :8080 --workspace ./workspace --store .aster
```

启动后, 终端会显示:

```text
aster: HTTP server started at http://localhost:8080
  POST /v1/agents/chat         (sync chat)
  POST /v1/agents/chat/stream  (SSE streaming chat)
  POST /v1/evals/text          (local text evals)
  POST /v1/workflows/demo/run  (demo workflow run API)
```

## 2. 交互式 Playground

你可以直接在页面中使用下方组件, 通过浏览器体验 demo 工作流的运行过程和事件结构:

<PlaygroundWorkflow />

- 选择不同的 `workflow_id` (顺序/并行/循环/嵌套)。
- 输入工作流的 `input` 文本。
- 勾选「同时对最终输出做 Evals」时, Playground 会调用 `/v1/workflows/demo/run-eval`, 返回事件列表的同时展示 `eval_scores`。

## 3. 请求与响应结构

### 请求体: `WorkflowRunRequest`

```jsonc
{
  "workflow_id": "sequential_demo", // 或 parallel_demo / loop_demo / nested_demo
  "input": "处理用户数据"
}
```

- `workflow_id`:
  - `"sequential_demo"` – 顺序工作流(数据收集 → 分析 → 报告)。
  - `"parallel_demo"` – 并行工作流(多个算法/数据源同时执行)。
  - `"loop_demo"` – 循环工作流(迭代优化, 直到质量达标或迭代次数上限)。
  - `"nested_demo"` – 嵌套工作流(并行收集 + 顺序分析/报告)。
- `input`: 传递给工作流的输入消息。

### 响应体: `WorkflowRunResponse`

```jsonc
{
  "events": [
    {
      "id": "evt-DataCollector-...",
      "timestamp": "2025-01-01T12:34:56Z",
      "agent_id": "DataCollector",
      "branch": "DataPipeline.DataCollector",
      "author": "agent",
      "text": "[DataCollector] 收集数据 - 处理: 处理用户数据",
      "metadata": {
        "agent_description": "收集数据",
        "quality_score": 89
      }
    }
  ],
  "error_message": ""
}
```

字段说明:

- `events` – 一次工作流运行过程中产生的事件列表(顺序保持)。
- `WorkflowEvent` 字段:
  - `id` – 事件 ID。
  - `timestamp` – 事件时间。
  - `agent_id` – 当前子 Agent 名称(例如 `DataCollector` / `Analyzer` / `Reporter`)。
  - `branch` – 工作流分支路径, 便于追踪来源(例如 `DataPipeline.Analyzer` 或 `OptimizationLoop.Critic.iter1`)。
  - `author` – 作者, 当前为 `"agent"`。
  - `text` – 核心文本内容。
  - `metadata` – 附加元数据, 包含:
    - `agent_description`: 子 Agent 描述。
    - `quality_score`: 模拟质量分数(用于 Loop 停止条件)等。

## 4. 使用 cURL 调用示例

### 顺序工作流(sequential_demo)

```bash
curl -X POST http://localhost:8080/v1/workflows/demo/run \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_id": "sequential_demo",
    "input": "处理用户数据"
  }'
```

### 并行工作流(parallel_demo)

```bash
curl -X POST http://localhost:8080/v1/workflows/demo/run \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_id": "parallel_demo",
    "input": "求解优化问题"
  }'
```

### 循环优化(loop_demo)

```bash
curl -X POST http://localhost:8080/v1/workflows/demo/run \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_id": "loop_demo",
    "input": "优化代码质量"
  }'
```

### 嵌套工作流(nested_demo)

```bash
curl -X POST http://localhost:8080/v1/workflows/demo/run \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_id": "nested_demo",
    "input": "综合数据分析"
  }'
```

## 5. 实现位置与架构说明

- Handler 实现:
  - `pkg/server/workflow_demo.go`
  - `Server.WorkflowDemoRunHandler()` 创建 demo 工作流, 执行并收集事件, 返回精简的 `WorkflowEvent` 列表。
- Demo 工作流构造:
  - 使用 `pkg/agent/workflow` 中的:
    - `NewSequentialAgent`
    - `NewParallelAgent`
    - `NewLoopAgent`
  - 子 Agent 为 `DemoAgent`, 实现于 `pkg/server/workflow_demo.go`, 用于模拟实际 Agent 的行为。

## 6. 设计要点与下一步

- 设计要点:
  - 提供了一个明确的 HTTP Workflow Run API, 可供前端/工具直接调用。
  - 返回结构化事件列表(包含 branch/metadata), 便于在 UI 中可视化工作流轨迹。
- 当前限制:
  - Demo 版本仅展示基本能力, 更复杂的工作流编排可以在此基础上扩展。

下一步你可以在此基础上:

- 将 Demo Workflow 替换为基于 `agent.Agent` 的真实 LLM 工作流实现。
- 把工作流定义抽象为可配置结构(例如从 YAML/JSON 加载 workflow graph)。
- 在前端 Playground 中增加更多视图, 通过 `/v1/workflows/demo/run` 可视化执行过程。

## 7. 运行并同时评估工作流输出

在有了 `/v1/evals/session` 之后, 我们还提供了一个便捷接口, 用于在单次调用中 **运行 demo 工作流并对最终回答进行评估**:

- 路径: `POST /v1/workflows/demo/run-eval`

示例:

```bash
curl -X POST http://localhost:8080/v1/workflows/demo/run-eval \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_id": "sequential_demo",
    "input": "处理用户数据",
    "reference": "期望的报告需要包含收集、分析、报告三个步骤。",
    "keywords": ["收集", "分析", "报告"],
    "scorers": ["keyword_coverage", "lexical_similarity"]
  }'
```

响应示例结构:

```jsonc
{
  "events": [
    { "id": "...", "agent_id": "DataCollector", "text": "...", "metadata": {...} },
    { "id": "...", "agent_id": "Analyzer", "text": "...", "metadata": {...} },
    { "id": "...", "agent_id": "Reporter", "text": "...", "metadata": {...} }
  ],
  "eval_scores": [
    { "name": "keyword_coverage", "value": 0.75, "details": {...} },
    { "name": "lexical_similarity", "value": 0.82, "details": {...} }
  ]
}
```

约定:

- 用 workflow 运行过程中产生的 `session.Event` 列表, 调用 `evals.BuildTextEvalInputFromEvents`:
  - 使用最后一条 assistant 事件的文本作为 `Answer`;
  - 使用之前的 user/assistant 事件作为 `Context`。
- 然后按 `scorers` 列表运行本地评估器, 得到 `eval_scores`。

这使得你可以在一个端点中同时查看:

- 工作流的执行轨迹(事件列表);
- 对最终输出的简单自动评分(coverage + similarity)。

## 8. 查询历史 Workflow Run 记录

每次调用 `/v1/workflows/demo/run` 或 `/v1/workflows/demo/run-eval` 时, 后端都会为该次运行生成一个 `run_id`, 并将运行结果存入内存中的 run store。你可以通过以下接口查询某次运行的完整记录:

- 路径: `GET /v1/workflows/demo/runs?id=<run_id>`

示例:

```bash
RUN_ID="在 /run 或 /run-eval 响应中返回的 run_id"

curl "http://localhost:8080/v1/workflows/demo/runs?id=${RUN_ID}"
```

响应结构:

```jsonc
{
  "id": "run-uuid",
  "workflow_id": "sequential_demo",
  "input": "处理用户数据",
  "events": [...],
  "eval_scores": [...],
  "error_message": "",
  "created_at": "2025-01-01T12:34:56Z"
}
```

> 当前 run store 采用内存实现, 主要用于 demo 和调试。在生产环境中, 你可以参考 Session/MySQL/Postgres 的实现方式, 将 Workflow Run 记录持久化到数据库中, 实现更完整的 Workflow Run 历史查询与分析能力。
