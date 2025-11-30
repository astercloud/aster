<p align="center">
  <img src="https://raw.githubusercontent.com/astercloud/aster/main/docs/public/images/logo-banner.svg" alt="Aster · 星尘云枢" width="800">
</p>

<p align="center">
  <strong>星尘汇聚，智能成枢</strong><br>
  让每一个 Agent 都能在生产环境中闪耀
</p>

<p align="center">
  <a href="https://github.com/astercloud/aster/actions/workflows/go-ci.yml"><img src="https://github.com/astercloud/aster/actions/workflows/go-ci.yml/badge.svg" alt="Go CI"></a>
  <a href="https://goreportcard.com/report/github.com/astercloud/aster"><img src="https://goreportcard.com/badge/github.com/astercloud/aster" alt="Go Report Card"></a>
  <a href="https://codecov.io/gh/astercloud/aster"><img src="https://codecov.io/gh/astercloud/aster/branch/main/graph/badge.svg" alt="codecov"></a>
  <a href="https://github.com/astercloud/aster/releases"><img src="https://img.shields.io/github/v/release/astercloud/aster" alt="Release"></a>
  <a href="https://github.com/astercloud/aster/blob/main/LICENSE"><img src="https://img.shields.io/github/license/astercloud/aster" alt="License"></a>
</p>

<p align="center">
  📖 <a href="https://astercloud.github.io/aster/"><strong>完整文档</strong></a> ·
  🚀 <a href="https://astercloud.github.io/aster/introduction/quickstart"><strong>快速开始</strong></a> ·
  🏗️ <a href="https://astercloud.github.io/aster/architecture/overview"><strong>架构设计</strong></a> ·
  📝 <a href="https://astercloud.github.io/aster/examples"><strong>示例代码</strong></a>
</p>

---

## 什么是 Aster?

Aster 是一个**生产级 AI Agent 框架**，用 Go 语言构建，专为企业级应用设计。它完整实现了 [Google Context Engineering](https://cloud.google.com/blog/products/ai-machine-learning/context-engineering-for-ai-agents) 白皮书的所有核心特性。

## ✨ 核心特性

| 特性                 | 描述                                       |
| -------------------- | ------------------------------------------ |
| 🔄 **事件驱动架构**  | Progress/Control/Monitor 三通道设计        |
| 🧠 **三层记忆系统**  | Text/Working/Semantic Memory + 溯源 + 合并 |
| 🔀 **Workflow 编排** | 8 种步骤类型 + 动态路由 + 并行/顺序/循环   |
| 🛡️ **安全防护栏**    | PII 检测、提示注入防护、内容审核           |
| ☁️ **云沙箱集成**    | 阿里云 AgentBay、火山引擎原生支持          |
| 📊 **可观测性**      | OpenTelemetry 完整集成                     |
| 💾 **数据持久化**    | PostgreSQL + MySQL 双数据库支持            |
| 🔌 **MCP 协议**      | Model Context Protocol 工具扩展            |

## 🚀 快速开始

### 安装

```bash
go get github.com/astercloud/aster
```

### 最小示例

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/astercloud/aster/pkg/agent"
    "github.com/astercloud/aster/pkg/provider"
    "github.com/astercloud/aster/pkg/types"
)

func main() {
    // 创建 Agent
    ag, _ := agent.Create(context.Background(), &types.AgentConfig{
        TemplateID: "assistant",
        ModelConfig: &types.ModelConfig{
            Provider: "anthropic",
            Model:    "claude-sonnet-4-5",
            APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
        },
    }, agent.DefaultDependencies())
    defer ag.Close()

    // 对话
    result, _ := ag.Chat(context.Background(), "Hello, World!")
    fmt.Println(result.Text)
}
```

👉 更多示例请查看 [完整文档](https://astercloud.github.io/aster/introduction/quickstart)

## 📐 架构概览

![aster 系统架构](https://raw.githubusercontent.com/astercloud/aster/main/docs/public/images/architecture-overview.svg)

<details>
<summary><b>Middleware 洋葱模型</b></summary>

![Middleware 洋葱模型](https://raw.githubusercontent.com/astercloud/aster/main/docs/public/images/middleware-onion.svg)

</details>

## 📚 文档

| 文档                                                                   | 描述                               |
| ---------------------------------------------------------------------- | ---------------------------------- |
| [快速开始](https://astercloud.github.io/aster/introduction/quickstart) | 5 分钟上手 Aster                   |
| [核心概念](https://astercloud.github.io/aster/core-concepts)           | Agent、Memory、Workflow 等核心概念 |
| [Workflow 编排](https://astercloud.github.io/aster/workflows)          | 工作流配置与执行                   |
| [API 参考](https://astercloud.github.io/aster/api-reference)           | 完整 API 文档                      |
| [示例代码](https://astercloud.github.io/aster/examples)                | 丰富的使用示例                     |
| [架构设计](https://astercloud.github.io/aster/architecture)            | 系统架构与设计理念                 |

## 🏆 Google Context Engineering 实现度

Aster 是**首个完整实现** Google Context Engineering 标准的 Go 语言框架：

- ✅ Sessions & Memory
- ✅ Memory Provenance
- ✅ Memory Consolidation
- ✅ PII Auto-Redaction
- ✅ Event-Driven Architecture
- ✅ Streaming & Backpressure
- ✅ Multi-Agent Orchestration
- ✅ Observability

## 📊 项目状态

| 指标     | 数值        |
| -------- | ----------- |
| 代码量   | 18,000+ LOC |
| 测试覆盖 | 80%+        |
| 版本     | v0.17.0     |
| 状态     | ✅ 生产就绪 |

## 🤝 贡献

欢迎贡献代码！请查看 [贡献指南](https://astercloud.github.io/aster/about/contributing)。

## 📄 License

MIT License - 详见 [LICENSE](LICENSE) 文件
