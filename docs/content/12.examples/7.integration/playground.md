---
title: Playground 示例
description: 在浏览器中交互式体验 aster Chat 与流式事件
---

# Playground 示例

本页面提供一个最简版的 Playground, 用于直接在浏览器中体验 aster 的 HTTP Chat 能力:

- 同步调用 `/v1/agents/chat`
- 基于 SSE 的流式调用 `/v1/agents/chat/stream`

> 提示: 在使用 Playground 前, 请先在本地启动一个 HTTP Server, 例如:
>
> ```bash
> # 方式一: 使用 CLI
> aster serve --addr :8080 --workspace ./workspace --store .aster
>
> # 方式二: 直接运行示例
> cd examples
> go run server-http/main.go
> ```

## 交互式 Playground

你可以在下面的组件中:

- 设置 Server Base URL(默认为 `http://localhost:8080`)
- 选择模板 ID(默认 `assistant`)
- 可选设置 Routing Profile(如 `quality` / `cost` / `latency`)
- 输入用户问题
- 分别测试同步和流式调用

<PlaygroundChat />

## 能力说明

当前 Playground 仅实现了:

- **单轮对话**: 每次请求只发送一条 user 输入, 不做多轮上下文拼接。
- **同步结果展示**: 展示 `/v1/agents/chat` 的完整 JSON 响应。
- **流式事件预览**: 直接展示从 `/v1/agents/chat/stream` 返回的 JSON 事件(一行一个 envelope)。
- **本地 Evals 预览**: 使用 `/v1/evals/text` 对最近一次同步响应中的 `text` 字段进行关键词覆盖率/词汇相似度评估, 并展示原始 JSON 结果。

后续你可以在此基础上:

- 将事件按类型拆分渲染(思考块/文本块/工具调用/Token 使用等)。
- 增加会话历史/多轮对话支持。
- 集成 evals/logging/MCP 等能力, 构建更完整的 Dev UI。
