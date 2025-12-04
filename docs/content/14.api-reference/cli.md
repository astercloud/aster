---
title: aster CLI 示例
description: 使用 aster serve 启动标准 HTTP Chat 服务
navigation: false
---

# aster CLI 示例

aster 提供了一个简单的 CLI 可执行程序 `aster`, 用于快速启动一个标准化的 HTTP Chat 服务(当前为最小可用版本)。

> 示例代码路径: `cmd/aster/main.go`

## 1. 安装 CLI

在 aster 仓库根目录执行:

```bash
go install ./cmd/aster@latest
```

安装成功后, 你可以在终端中直接运行:

```bash
aster -h
```

## 2. 启动 HTTP Server

当前 CLI 提供三个子命令:

- `serve` – 启动 HTTP Server, 暴露:
  - `POST /v1/agents/chat` – 同步 Chat 接口
  - `POST /v1/agents/chat/stream` – 基于 SSE 的流式 Chat 接口
  - `POST /v1/evals/text` – 本地文本评估接口(关键词覆盖率 + 词汇相似度)
- `mcp-serve` – 启动 MCP HTTP Server, 将本地工具暴露为 MCP 服务:
  - `POST /mcp` – JSON-RPC 2.0 接口, 支持 `tools/list` 和 `tools/call`
- `eval` – 在命令行中对文本进行本地评估(不依赖外部 LLM)

### 命令行参数

```bash
aster serve [flags]
aster mcp-serve [flags]
aster eval [flags]

serve Flags:
  -addr string
        HTTP listen address (default ":8080")
  -workspace string
        Sandbox workspace directory (default "./workspace")
  -store string
        Directory for JSON store data (default ".aster")
  -template string
        Default Agent template ID (default "assistant")
  -model string
        Default model name (default "claude-sonnet-4-5")
  -config string
        Optional YAML config file for templates and routing
```

`mcp-serve` Flags:

```text
  -addr string
        MCP HTTP listen address (default ":8090")
  -docs string
        Base directory for docs_get/docs_search tools (default "./docs/content")
```

### 启动 Chat Server

```bash
# 设置模型 API Key
export ANTHROPIC_API_KEY=your_api_key_here

# 启动 HTTP Server
aster serve \
  --addr :8080 \
  --workspace ./workspace \
  --store .aster
```

启动后, 终端会显示:

```text
aster: HTTP server started at http://localhost:8080
  POST /v1/agents/chat         (sync chat)
  POST /v1/agents/chat/stream  (SSE streaming chat)
  POST /v1/evals/text          (local text evals, answer-only)
  POST /v1/evals/session       (local text evals, from session events)
  POST /v1/workflows/demo/run  (demo workflow run API)
  POST /v1/workflows/demo/run-eval (demo workflow run + eval API)
```

如果你希望通过配置文件统一管理模板和路由策略, 可以增加 `-config` 参数:

```bash
aster serve \
  --addr :8080 \
  --workspace ./workspace \
  --store .aster \
  --config ./aster.yaml
```

一个最小可用的 `aster.yaml` 示例(已包含在仓库根目录):

```yaml
templates:
  - id: assistant
    model: claude-sonnet-4-5
    system_prompt: >
      You are a helpful assistant with access to filesystem and memory tools.
      Use tools when appropriate to read/write files or manage long-term memory.
    tools:
      - Read
      - Write
      - Bash

routing:
  profiles:
    quality:
      provider: anthropic
      model: claude-sonnet-4-5
      env_api_key: ANTHROPIC_API_KEY
    cost:
      provider: deepseek
      model: deepseek-chat
      env_api_key: DEEPSEEK_API_KEY
```

### 启动 MCP Server

```bash
aster mcp-serve \
  --addr :8090 \
  --docs ./docs/content
```

会启动一个 MCP HTTP Server, 监听 `http://localhost:8090/mcp`, 默认提供:

- `echo` – 简单回显工具
- `docs_get` – 读取 `--docs` 目录下的文档文件
- `docs_search` – 在 `--docs` 目录中搜索关键字

## 3. 调用示例

### 同步 Chat

```bash
curl -X POST http://localhost:8080/v1/agents/chat \
  -H "Content-Type: application/json" \
  -d '{
    "template_id": "assistant",
    "input": "请帮我总结一下 README",
    "metadata": {"user_id": "alice"},
    "middlewares": ["filesystem", "agent_memory"]
  }'
```

### 流式 Chat (SSE)

```bash
curl -N -X POST http://localhost:8080/v1/agents/chat/stream \
  -H "Content-Type: application/json" \
  -d '{
    "template_id": "assistant",
    "input": "请边思考边输出, 并展示工具调用过程",
    "metadata": {"user_id": "alice"},
    "middlewares": ["filesystem", "agent_memory"]
  }'
```

输出为一系列 `data: {...}` 行, 每行一个 JSON 事件(AgentEventEnvelope)。

### MCP 客户端调用

MCP 客户端(例如 `examples/mcp`) 可以通过 `MCP_ENDPOINT` 环境变量指向 CLI 启动的 MCP Server:

```bash
export MCP_ENDPOINT="http://localhost:8090/mcp"
cd examples/mcp
go run main.go
```

Agent 会通过 `MCPManager` 自动发现并注册 MCP 工具(`echo`/`docs_get`/`docs_search`)。

前端可以基于这些事件构建更丰富的 UI(思考流、工具调用可视化、token 使用统计等)。

## 4. 模板与工具

默认情况下, CLI 会注册一个简单模板:

```go
templateRegistry.Register(&types.AgentTemplateDefinition{
    ID:    "assistant",
    Model: "claude-sonnet-4-5",
    SystemPrompt: "You are a helpful assistant with access to filesystem and memory tools. " +
        "Use tools when appropriate to read/write files or manage long-term memory.",
    Tools: []interface{}{"Read", "Write", "Bash"},
})
```

你可以在未来版本中:

- 扩展 CLI, 支持：
  - 从配置文件加载模板和模型配置。
  - 挂接自定义工具 / 中间件。
  - 启动多 Agent / 多模板实例。
- 与 `pkg/server` 其余扩展能力结合(如 Session API、OpenAPI 生成), 逐步打造一个更完善的一站式开发体验。
