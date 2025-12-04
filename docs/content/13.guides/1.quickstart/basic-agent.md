---
title: 基础Agent开发
description: 从零开始创建完整的Agent应用
navigation: false
---

# 基础Agent开发

本指南将带您从零开始创建一个功能完整的Agent应用，涵盖项目初始化、依赖配置、Agent创建和事件处理。

## 🎯 项目目标

创建一个文件助手Agent，能够：

- 读取和写入文件
- 执行搜索和编辑操作
- 流式输出响应
- 处理错误和异常

## 📁 项目结构

```
file-assistant/
├── go.mod
├── go.sum
├── main.go
├── .env
├── .gitignore
└── workspace/
```

## 🚀 步骤1：初始化项目

```bash
# 创建项目目录
mkdir file-assistant
cd file-assistant

# 初始化Go模块
go mod init file-assistant

# 安装依赖
go get github.com/astercloud/aster
go get github.com/joho/godotenv  # 环境变量管理
```

## 📝 步骤2：配置环境变量

创建 `.env` 文件：

```bash
# API密钥
ANTHROPIC_API_KEY=sk-ant-xxxxx

# 可选配置
AGENT_MODEL=claude-sonnet-4-5
WORKSPACE_DIR=./workspace
```

创建 `.gitignore`：

```gitignore
.env
workspace/
.aster/
```

## 💻 步骤3：编写主程序

创建 `main.go`：

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/joho/godotenv"
    "github.com/astercloud/aster/pkg/agent"
    "github.com/astercloud/aster/pkg/provider"
    "github.com/astercloud/aster/pkg/sandbox"
    "github.com/astercloud/aster/pkg/store"
    "github.com/astercloud/aster/pkg/tools"
    "github.com/astercloud/aster/pkg/tools/builtin"
    "github.com/astercloud/aster/pkg/types"
)

func main() {
    // 加载环境变量
    if err := godotenv.Load(); err != nil {
        log.Println("未找到.env文件，使用系统环境变量")
    }

    // 创建Agent
    ag, err := createAgent()
    if err != nil {
        log.Fatal(err)
    }
    defer ag.Close()

    // 启动事件处理
    go handleEvents(ag)

    // 运行对话
    runConversation(ag)
}

func createAgent() (*agent.Agent, error) {
    ctx := context.Background()

    // 1. 创建工具注册表
    toolRegistry := tools.NewRegistry()
    builtin.RegisterAll(toolRegistry)

    // 2. 创建持久化存储
    jsonStore, err := store.NewJSONStore("./.aster")
    if err != nil {
        return nil, fmt.Errorf("创建存储失败: %w", err)
    }

    // 3. 准备依赖
    deps := &agent.Dependencies{
        Store:            jsonStore,
        SandboxFactory:   sandbox.NewFactory(),
        ToolRegistry:     toolRegistry,
        ProviderFactory:  &provider.AnthropicFactory{},
        TemplateRegistry: agent.NewTemplateRegistry(),
    }

    // 4. 注册模板
    deps.TemplateRegistry.Register(&types.AgentTemplateDefinition{
        ID:           "file-assistant",
        SystemPrompt: "你是一个文件助手，擅长文件操作和内容分析。",
        Model:        getEnv("AGENT_MODEL", "claude-sonnet-4-5"),
        Tools: []interface{}{
            "Read", "Write", "Edit",
            "Ls", "Glob", "Grep",
        },
    })

    // 5. 创建Agent
    ag, err := agent.Create(ctx, &types.AgentConfig{
        TemplateID: "file-assistant",
        ModelConfig: &types.ModelConfig{
            Provider: "anthropic",
            Model:    getEnv("AGENT_MODEL", "claude-sonnet-4-5"),
            APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
        },
        Sandbox: &types.SandboxConfig{
            Kind:    types.SandboxKindLocal,
            WorkDir: getEnv("WORKSPACE_DIR", "./workspace"),
        },
    }, deps)

    return ag, err
}

func handleEvents(ag *agent.Agent) {
    eventCh := ag.Subscribe([]types.AgentChannel{
        types.ChannelProgress,
        types.ChannelMonitor,
    }, nil)

    for envelope := range eventCh {
        switch e := envelope.Event.(type) {
        case *types.ProgressTextChunkEvent:
            fmt.Print(e.Delta)

        case *types.ProgressToolStartEvent:
            fmt.Printf("\n\n🔧 [%s]\n", e.Call.Name)

        case *types.ProgressToolEndEvent:
            if e.Error != nil {
                fmt.Printf("   ❌ 失败: %v\n\n", e.Error)
            } else {
                fmt.Printf("   ✅ 完成\n\n")
            }

        case *types.MonitorErrorEvent:
            log.Printf("[ERROR] %v", e.Error)
        }
    }
}

func runConversation(ag *agent.Agent) {
    ctx := context.Background()

    // 示例对话
    queries := []string{
        "列出workspace目录下的所有文件",
        "创建一个hello.txt文件，内容是'Hello aster'",
        "读取hello.txt的内容",
        "在workspace目录下搜索包含'Hello'的文件",
    }

    for i, query := range queries {
        fmt.Printf("\n\n===== 查询 %d =====\n%s\n\n", i+1, query)

        result, err := ag.Chat(ctx, query)
        if err != nil {
            log.Printf("查询失败: %v", err)
            continue
        }

        fmt.Printf("\n\n📝 最终结果: %s\n", result.Text)
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
```

## 🏃 步骤4：运行程序

```bash
# 创建工作目录
mkdir workspace

# 运行程序
go run main.go
```

**预期输出**：

```
===== 查询 1 =====
列出workspace目录下的所有文件

🔧 [Ls]
   ✅ 完成

workspace目录当前为空。

📝 最终结果: workspace目录当前为空

===== 查询 2 =====
创建一个hello.txt文件，内容是'Hello aster'

🔧 [Write]
   ✅ 完成

已创建hello.txt文件。

📝 最终结果: 文件已成功创建
...
```

## 🎨 步骤5：添加交互模式

修改 `runConversation` 函数：

```go
func runConversation(ag *agent.Agent) {
    ctx := context.Background()
    scanner := bufio.NewScanner(os.Stdin)

    fmt.Println("文件助手已就绪！输入'exit'退出。\n")

    for {
        fmt.Print("您: ")
        if !scanner.Scan() {
            break
        }

        query := scanner.Text()
        if query == "exit" {
            fmt.Println("再见！")
            break
        }

        fmt.Print("\nAgent: ")
        result, err := ag.Chat(ctx, query)
        if err != nil {
            log.Printf("错误: %v\n", err)
            continue
        }

        fmt.Printf("\n\n")
    }
}
```

现在运行后可以交互式对话：

```
文件助手已就绪！输入'exit'退出。

您: 创建一个test.go文件
Agent: 🔧 [Write]
   ✅ 完成
已创建test.go文件。

您: 列出所有go文件
Agent: 🔧 [Glob]
   ✅ 完成
找到1个Go文件: test.go
```

## 📚 完整示例

查看完整代码：[GitHub](https://github.com/astercloud/aster/tree/main/examples/file-assistant)

## 🎯 下一步

- [自定义工具](/guides/custom-tools) - 添加自定义功能
- [中间件开发](/guides/custom-middleware) - 扩展Agent能力
- [多Agent协作](/guides/multi-agent) - 构建复杂系统
