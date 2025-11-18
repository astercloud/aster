---
title: 部署概述
description: CLI、HTTP Server、路由、MCP、Logging、Evals 等运行时能力总览
---

# 运行与运维

本章节汇总与「运行时部署与运维」相关的指南, 方便从一个入口了解:

- 如何启动 HTTP Server / Streaming 接口
- 如何使用 `aster` CLI
- 如何集成 MCP Server
- 如何配置模型路由(Model Routing)
- 如何接入 Logging / Evals

## 🌐 HTTP Server 与工作流

- [HTTP Server 接入](/guides/server-http)
- [工作流 HTTP API 示例](/guides/workflow-http)

推荐用法:

- 开发环境使用 `aster serve` 快速启动一个标准化的 Chat 服务;
- 生产环境将 `pkg/server` 集成到自己的 Web 服务中, 暴露 `/v1/agents/chat` 等接口;
- 对工作流相关的 API, 可以参考工作流 HTTP 示例中的事件流处理方式。

## 🧰 CLI 与配置

- [aster CLI 示例](/guides/cli)

CLI 适合:

- 本地快速启动/停止 Server;
- 执行简单的 evals 任务;
- 配合 `aster.yaml` 做模型/路由/Memory 的集中配置。

## 🔌 MCP Server 集成

- [MCP HTTP Server](/guides/mcp-server)

通过 MCP 你可以:

- 将本地工具暴露为标准 MCP 服务器;
- 让其他 Agent/客户端通过 `tools/list` 与 `tools/call` 调用你的能力;
- 把已有的业务服务封装成 MCP 工具, 供 aster 或其他框架共用。

## 🧭 模型路由(Model Routing)

- [模型路由示例](/guides/router)

主要能力:

- 基于 `routing_profile` 在多模型之间做 cost-first / quality-first 等策略路由;
- 按任务类型、优先级选择不同 Provider/模型。

## 📊 Logging 与 Evals

- [结构化 Logging 示例](/guides/logging)
- [Evals 指南](/guides/evals)

推荐实践:

- 使用 `pkg/logging` 输出 JSON 结构化日志, 便于在生产环境接入日志平台;
- 使用 `pkg/evals` 在本地对输出进行关键词覆盖/词汇相似度打分, 在不依赖外部 LLM 的前提下做基础评估。

> 以上各小节的详细内容仍在各自的指南页面中, 本页主要作为一个集中入口, 让导航更清晰。

