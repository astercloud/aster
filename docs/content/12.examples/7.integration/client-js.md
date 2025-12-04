---
title: JS 客户端示例
description: 使用 @aster/client-js 访问 HTTP Chat API
---

# JS 客户端示例

为方便前端/Node.js 应用集成, aster 提供了一个最小的 JS/TS 客户端包:

- 包路径: `client-sdks/client-js`
- NPM 名称: `@aster/client-js`
- 封装了 `/v1/agents/chat` 同步接口

> 当前版本仅对同步 Chat API 提供封装。Streaming 可以在应用侧使用 `fetch` + ReadableStream 或其他 SSE/WebSocket 方案自行实现。

## 1. 安装

在你的前端或 Node.js 项目中:

```bash
npm install @aster/client-js
```

## 2. 基本使用

```ts
import { AgentsdkClient } from "@aster/client-js";

const client = new AgentsdkClient({
  baseUrl: "http://localhost:8080", // aster serve 的地址
});

async function main() {
  const res = await client.chat({
    template_id: "assistant",
    input: "请帮我总结一下 README",
    metadata: { user_id: "alice" },
    middlewares: ["filesystem", "agent_memory"],
  });

  if (res.status === "ok") {
    console.log("Answer:", res.text);
  } else {
    console.error("Error:", res.error_message);
  }
}

main().catch(console.error);
```

## 3. 与 OpenAPI 规范对应

HTTP Chat 接口的 OpenAPI 规范位于:

- `docs/public/openapi/aster.yaml`

片段示例:

```yaml
paths:
  /v1/agents/chat:
    post:
      summary: Synchronous chat with an Agent
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/ChatRequest"
      responses:
        "200":
          description: Chat completed
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ChatResponse"
```

这意味着你也可以使用任意 OpenAPI 生成器(如 `openapi-generator` 或 `orval`)基于该规范生成自定义客户端。

## 4. 与 Playground / 前端集成的下一步

有了:

- `aster serve` 提供的 HTTP Chat(同步/流式)接口
- `@aster/client-js` 封装的同步 Chat 客户端
- MCP Server(`aster mcp-serve`) 暴露 docs / 项目工具

你可以在自己的前端项目中构建一个简单的 Playground:

- 使用 `@aster/client-js` 完成交互式 Chat。
- 使用 `fetch` 对 `/v1/agents/chat/stream` 做 streaming 渲染。
- 使用 MCP + `docs_search`/`docs_get` 在 IDE 或 Web UI 中集成文档检索能力。

后续可以基于这些组件逐步打造更完整的 Dev UI, 构建适合你团队的 Playground。
