# OpenRouter Agent 示例

本示例演示如何使用 OpenRouter 作为 LLM Provider 构建 AI Agent。

## 概述

OpenRouter 是一个统一的 API 网关，可通过单一端点访问多个 LLM 提供商（Anthropic、OpenAI、Google 等）。本示例展示了与 Aster Agent 框架的集成模式。

## 架构

示例由三部分组成：

1. **命令行工具**（main.go）- 基于 urfave/cli v3 构建，支持交互式对话和单次执行两种模式。

2. **集成测试**（agent_test.go）- 采用 Testify Suite 模式，通过生命周期钩子管理测试环境。

3. **测试工具**（testutil_test.go）- 测试共享的 Agent 工厂函数。

## 命令行参数

| 参数 | 简写 | 说明 |
|------|------|------|
| --print | -p | 非交互模式，执行指定提示词后退出 |
| --stream | -s | 启用流式模式，实时输出响应 |
| --model | -m | 指定使用的模型，默认 anthropic/claude-haiku-4.5 |
| --help | -h | 显示帮助信息 |

## 运行方式

设置环境变量 OPENROUTER_API_KEY 后：

- 查看帮助：go run . --help
- 交互模式：go run .
- 单次执行：go run . -p "你好"
- 流式输出：go run . -p "你好" -s
- 指定模型：go run . -m "openai/gpt-4o" -p "你好"
- 运行测试：go test -v ./...
- 跳过集成测试：go test -v -short ./...

## 测试理念

本示例采用 Testify Suite 模式，具有以下优势：

- **Suite 级别初始化** - Agent 等资源只初始化一次，所有测试共享
- **Test 级别隔离** - 每个测试方法在干净的环境中运行
- **结构化断言** - require 用于前置条件，assert 用于验证
- **表驱动子测试** - 用最少的代码测试多种场景

## 支持的模型

通过 --model 参数可使用 OpenRouter 上的任意模型，包括 Anthropic Claude、OpenAI GPT、Google Gemini 等。完整列表参见 OpenRouter Models 页面。
