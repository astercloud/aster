---
title: Aster · 星尘云枢 - Where Stardust Converges, Intelligence Emerges
description: 让每一个 Agent 都能在生产环境中闪耀 | Empowering Every Agent to Shine in Production
navigation: false
---

<div class="max-w-4xl mx-auto">

<div class="text-center py-8">
  <img src="/images/logo-banner.svg" alt="Aster · 星尘云枢" class="mx-auto mb-8" style="max-width: 800px; width: 100%;" />
  <p class="text-3xl font-semibold text-gray-700 dark:text-gray-300 mb-2">星尘汇聚，智能成枢</p>
  <p class="text-xl text-gray-600 dark:text-gray-400 mb-4">让每一个 Agent 都能在生产环境中闪耀</p>
  <p class="text-base text-gray-500 dark:text-gray-500 mb-10">Python、TypeScript、Bash 原生执行 • 事件驱动架构 • 企业级安全治理<br><em>Go 的高性能基底，脚本的灵活编排，为生产环境而生</em></p>
  <div class="flex gap-4 justify-center flex-wrap">
    <a href="/introduction/quickstart" class="inline-block px-8 py-3 bg-primary-600 text-white font-medium rounded-lg hover:bg-primary-700 transition-colors">快速开始</a>
    <a href="https://github.com/astercloud/aster" target="_blank" class="inline-block px-8 py-3 border-2 border-gray-300 dark:border-gray-600 font-medium rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors">GitHub</a>
  </div>
</div>

<div class="text-center py-2 px-4 mb-6 bg-gray-100 dark:bg-gray-800 rounded-lg">
  <p class="text-sm text-gray-600 dark:text-gray-400">
    当前版本: <strong>v0.12.1</strong> |
    <a href="/about/release-notes" class="text-primary-600 hover:underline">更新日志</a> |
    <a href="https://github.com/astercloud/aster/releases" target="_blank" class="text-primary-600 hover:underline">下载地址</a>
  </p>
</div>

## ✨ 核心特性

<div class="grid grid-cols-1 md:grid-cols-2 gap-6 my-8">
  <div class="p-6 border border-gray-200 dark:border-gray-700 rounded-lg">
    <h3 class="text-xl font-semibold mb-3">🎯 事件驱动架构</h3>
    <p class="text-gray-600 dark:text-gray-400">基于Go channel的事件系统，支持Progress、Control、Monitor三类事件通道，实现非阻塞式交互。</p>
  </div>
  <div class="p-6 border border-gray-200 dark:border-gray-700 rounded-lg">
    <h3 class="text-xl font-semibold mb-3">🧅 洋葱模型中间件</h3>
    <p class="text-gray-600 dark:text-gray-400">灵活的中间件栈，支持文件系统、子Agent、总结等内置中间件，轻松实现功能扩展。</p>
  </div>
  <div class="p-6 border border-gray-200 dark:border-gray-700 rounded-lg">
    <h3 class="text-xl font-semibold mb-3">🛡️ 云端沙箱执行</h3>
    <p class="text-gray-600 dark:text-gray-400">支持本地和云端（阿里云/火山引擎）沙箱，确保代码执行安全隔离。</p>
  </div>
  <div class="p-6 border border-gray-200 dark:border-gray-700 rounded-lg">
    <h3 class="text-xl font-semibold mb-3">🔌 多模型支持</h3>
    <p class="text-gray-600 dark:text-gray-400">统一的Provider接口，支持Anthropic、OpenAI、DeepSeek等主流大模型。</p>
  </div>
  <div class="p-6 border border-gray-200 dark:border-gray-700 rounded-lg">
    <h3 class="text-xl font-semibold mb-3">🛠️ 丰富的工具生态</h3>
    <p class="text-gray-600 dark:text-gray-400">内置文件系统、Bash、HTTP、Web搜索等工具，支持MCP协议和自定义工具。</p>
  </div>
  <div class="p-6 border border-gray-200 dark:border-gray-700 rounded-lg">
    <h3 class="text-xl font-semibold mb-3">🌌 AsterOS 运行时</h3>
    <p class="text-gray-600 dark:text-gray-400">统一运行时系统，整合 Pool 生命周期管理、Room 协作空间，自动生成 REST API。</p>
  </div>
</div>

## 🚀 快速开始

### 前置要求

- Go 1.24+
- API 密钥 (Anthropic/OpenAI/DeepSeek)

### 安装

```bash
go get github.com/astercloud/aster
```

### 创建第一个Agent

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/astercloud/aster/pkg/agent"
    "github.com/astercloud/aster/pkg/provider"
    "github.com/astercloud/aster/pkg/types"
)

func main() {
    // 创建Agent
    ag, err := agent.Create(context.Background(), &types.AgentConfig{
        TemplateID: "assistant",
        ModelConfig: &types.ModelConfig{
            Provider: "anthropic",
            Model:    "claude-sonnet-4-5",
            APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
        },
    }, deps)
    if err != nil {
        log.Fatal(err)
    }
    defer ag.Close()

    // 发送消息并监听事件
    eventCh := ag.Subscribe([]types.AgentChannel{types.ChannelProgress}, nil)
    go func() {
        for event := range eventCh {
            // 处理事件
        }
    }()

    ag.Chat(context.Background(), "介绍一下Go语言的优势")
}
```

### 添加工具支持

```go
import "github.com/astercloud/aster/pkg/tools"

// 注册内置工具
ag.RegisterTool(tools.BashTool())
ag.RegisterTool(tools.FileSystemTool())

// 发送需要使用工具的请求
eventCh := ag.Chat(ctx, "列出当前目录下的文件")
```

## 📚 核心概念

### 事件驱动

aster采用事件驱动架构，通过Go channel实现异步通信：

- **ProgressEvent**: 流式输出的增量内容
- **ControlEvent**: 工具调用、用户确认请求
- **MonitorEvent**: 内部状态变化监控

### 中间件系统

基于洋葱模型的中间件栈，支持：

- **FilesystemMiddleware**: 文件系统操作和内存管理
- **SubAgentMiddleware**: 子Agent任务委派
- **SummarizationMiddleware**: 自动上下文总结
- **自定义中间件**: 实现WrapModelCall/WrapToolCall接口

### 沙箱执行

支持多种沙箱后端：

- **本地沙箱**: 开发和测试环境
- **阿里云函数计算**: 生产级隔离执行
- **火山引擎**: 高性能计算场景

## 🏗️ 架构概览

![架构图](/images/architecture-overview.svg)

aster采用分层架构设计：

1. **AsterOS层**: 统一运行时系统，提供 API 网关、资源注册、多接口支持
2. **Pool + Room层**: Agent 生命周期管理 + 多智能体协作空间
3. **Agent Runtime层**: 中间件栈、Provider、工具执行器
4. **Backend层**: 状态管理、存储、文件系统
5. **基础设施层**: 沙箱、调度器、权限控制

## 📖 文档导航

<div class="grid grid-cols-1 md:grid-cols-3 gap-4 my-8">
  <a href="/introduction/quickstart" class="block p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:border-primary-500 transition-colors">
    <h3 class="font-semibold mb-2">快速入门</h3>
    <p class="text-sm text-gray-600 dark:text-gray-400">从零开始创建你的第一个Agent</p>
  </a>
  <a href="/core-concepts/agent-lifecycle" class="block p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:border-primary-500 transition-colors">
    <h3 class="font-semibold mb-2">核心概念</h3>
    <p class="text-sm text-gray-600 dark:text-gray-400">深入理解aster的设计理念</p>
  </a>
  <a href="/guides/basic-agent" class="block p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:border-primary-500 transition-colors">
    <h3 class="font-semibold mb-2">实战指南</h3>
    <p class="text-sm text-gray-600 dark:text-gray-400">真实场景的完整代码示例</p>
  </a>
  <a href="/api-reference/agent-api" class="block p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:border-primary-500 transition-colors">
    <h3 class="font-semibold mb-2">API参考</h3>
    <p class="text-sm text-gray-600 dark:text-gray-400">完整的API文档和类型定义</p>
  </a>
  <a href="/introduction/architecture" class="block p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:border-primary-500 transition-colors">
    <h3 class="font-semibold mb-2">架构指南</h3>
    <p class="text-sm text-gray-600 dark:text-gray-400">了解aster的架构设计</p>
  </a>
  <a href="https://github.com/astercloud/aster" target="_blank" class="block p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:border-primary-500 transition-colors">
    <h3 class="font-semibold mb-2">GitHub</h3>
    <p class="text-sm text-gray-600 dark:text-gray-400">查看源码和贡献代码</p>
  </a>
</div>

## 🤝 社区与支持

- **GitHub Issues**: [报告问题](https://github.com/astercloud/aster/issues)
- **讨论区**: [参与讨论](https://github.com/astercloud/aster/discussions)
- **示例代码**: [examples目录](https://github.com/astercloud/aster/tree/main/examples)

## 📄 开源协议

aster采用[MIT License](https://github.com/astercloud/aster/blob/main/LICENSE)开源。

</div>
