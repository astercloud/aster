---
title: Plan / Explore UI
description: 使用 aster 事件系统和 SSE 构建一个类似 Claude Code 的分阶段工具调用视图
---

# Plan / Explore UI 示例

本示例展示如何在 **终端和浏览器前端** 中构建一个简化版的「Plan / Explore」视图:

- 使用内置工具 `Read` / `Grep` / `Glob` / `Bash` 读取和搜索代码。
- 使用 `TodoWrite` + `todolist` Middleware 维护结构化任务列表。
- 使用 `ExitPlanMode` 在规划结束时输出完整计划。
- 订阅 `Progress*` / `Monitor*` 事件,把工具调用分组渲染成类似:

```text
Plan(分析 builtin 工具测试需求)
  └─ Read(pkg/tools/builtin/edit.go)
     Read(pkg/tools/builtin/utils.go)

Explore(分析当前工具实现状态)
  └─ Read(pkg/tools/builtin/task.go)
     Read(pkg/tools/builtin/subagent_manager.go)
```

> 提示: 实际输出取决于模型行为,下面示例只是典型形态。

---

## 运行方式

在仓库根目录下:

```bash
export ANTHROPIC_API_KEY="sk-ant-xxx"

# 启动 Web 前端(默认模式)
go run ./examples/plan-explore-ui
# 或显式指定:
go run ./examples/plan-explore-ui -mode=web -addr=:8080
```

然后访问:

```text
http://localhost:8080
```

如果你更喜欢终端 UI, 可以使用:

```bash
go run ./examples/plan-explore-ui -mode=cli
```

---

## 示例代码(核心部分)

文件: `examples/plan-explore-ui/main.go`

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"

    "github.com/astercloud/aster/pkg/agent"
    "github.com/astercloud/aster/pkg/provider"
    "github.com/astercloud/aster/pkg/sandbox"
    "github.com/astercloud/aster/pkg/store"
    "github.com/astercloud/aster/pkg/tools"
    "github.com/astercloud/aster/pkg/tools/builtin"
    "github.com/astercloud/aster/pkg/types"
)

// Plan / Explore UI 示例。
func main() {
    mode := flag.String("mode", "web", "UI mode: web 或 cli")
    addr := flag.String("addr", ":8080", "Web UI HTTP 监听地址")
    flag.Parse()

    apiKey := os.Getenv("ANTHROPIC_API_KEY")
    if apiKey == "" {
        log.Println("[WARN] ANTHROPIC_API_KEY not set, model calls will fail until a key is provided.")
    }

    ctx := context.Background()

    // 1. 工具注册表 + 内置工具
    toolRegistry := tools.NewRegistry()
    builtin.RegisterAll(toolRegistry)

    // 2. Sandbox 工厂
    sbFactory := sandbox.NewFactory()

    // 3. Provider 工厂 (支持多种模型厂商)
    providerFactory := provider.NewMultiProviderFactory()

    // 4. Store (JSON 文件存储,便于调试)
    storePath := ".aster-plan-explore-ui"
    jsonStore, err := store.NewJSONStore(storePath)
    if err != nil {
        log.Fatalf("Failed to create store: %v", err)
    }

    // 5. 模板注册表 + 专用模板
    templateRegistry := agent.NewTemplateRegistry()
    templateID := "plan-explore-ui"

    templateRegistry.Register(&types.AgentTemplateDefinition{
        ID:    templateID,
        Model: "claude-sonnet-4-5",
        SystemPrompt: "" +
            "You are a coding assistant working inside a CLI UI.\n" +
            "- Use Read / Glob / Grep to inspect the local codebase.\n" +
            "- Use TodoWrite to maintain a structured task list (plan vs explore).\n" +
            "- When you finish planning, call ExitPlanMode with a clear Markdown plan.\n" +
            "- Prefer multi-step workflows: Plan -> Explore -> Implement.\n",
        Tools: []interface{}{
            "Read", "Write", "Edit", "Glob", "Grep", "Bash",
            "TodoWrite", "Task", "ExitPlanMode",
        },
    })

    deps := &agent.Dependencies{
        Store:            jsonStore,
        SandboxFactory:   sbFactory,
        ToolRegistry:     toolRegistry,
        ProviderFactory:  providerFactory,
        TemplateRegistry: templateRegistry,
    }

    switch *mode {
    case "cli":
        runCliDemo(ctx, deps, templateID, apiKey)
    default:
        if err := runWebUI(ctx, deps, templateID, apiKey, *addr); err != nil {
            log.Fatalf("plan-explore-ui web server failed: %v", err)
        }
    }
}
```

> 完整文件中还包含 `runCliDemo`(终端模式)、`runWebUI`(Web 模式) 以及 UI 辅助类型,可以直接在仓库中查看。

---

## 默认 Prompt

无论是 CLI 还是 Web UI, 示例都使用同一个默认 Prompt, 引导 Agent 先规划、再探索:

```go
func defaultPrompt() string {
    return `请帮我完成一个两阶段的代码分析任务:
1. 规划(Plan): 分析 aster 仓库里 pkg/tools/builtin 目录下各个工具的职责和测试需求, 给出一个分步骤的实施计划。
2. 探索(Explore): 按计划实际阅读相关文件, 重点关注 TodoWrite / ExitPlanMode / Task / subagent_manager 等实现细节。

要求:
- 在规划阶段, 使用 TodoWrite 或 write_todos 工具创建任务列表, 至少包含 "分析 builtin 工具测试需求" 和 "分析当前工具实现状态" 两个任务。
- 在探索阶段, 使用 Read / Glob / Grep 等工具读取具体文件。
- 规划完成后, 调用 ExitPlanMode 返回一个 Markdown 格式的计划。`
}
```

---

## UI 渲染逻辑(共用思想)

无论是在 Go 终端 UI 中,还是在浏览器前端中,渲染逻辑是一致的:

### 1. 阶段识别

- 当收到 `ProgressToolStartEvent` 且工具为:
  - `TodoWrite` 或 `write_todos`:
    - 从 `arguments.todos` 中找到 `status == "in_progress"` 的任务。
    - 使用其 `activeForm` 或 `content` 作为标题,渲染:

    ```text
    Plan(分析 builtin 工具测试需求)
    ```

  - `Task` 或 `task`:
    - 从 `arguments.subagent_type` + `arguments.prompt` 中提取子代理类型和说明。
    - 当 `subagent_type` 为 `"Plan"` / `"Explore"` 时,渲染:

    ```text
    Plan(分析 builtin 工具测试需求)
    Explore(分析当前工具实现状态)
    ```

### 2. 工具调用分组

- 对其他工具调用(如 `Read` / `Glob` / `Grep` / `Bash`):
  - 如果当前存在活动阶段,则缩进展示:

  ```text
  Plan(分析 builtin 工具测试需求)
    └─ Read(pkg/tools/builtin/edit.go)
       Read(pkg/tools/builtin/utils.go)
  ```

  - 如果当前没有阶段,则以 `[Tool] <name>` 形式单独打印。

- 为了能在 UI 中访问工具参数,我们在内部把 `ToolCallSnapshot.Arguments` 填充为完整的输入 map,并在 `ProgressToolStartEvent` / `ProgressToolEndEvent` / `ProgressToolErrorEvent` 中一并发出:

```go
a.eventBus.EmitProgress(&types.ProgressToolStartEvent{
    Call: types.ToolCallSnapshot{
        ID:        record.ID,
        Name:      record.Name,
        State:     record.State,
        Arguments: record.Input,
    },
})
```

这对现有代码是向后兼容的:旧逻辑忽略 `Arguments` 字段,新 UI 则可以用它来构建更丰富的视图。

---

## 运行示例(终端模式)

在仓库根目录执行:

```bash
export ANTHROPIC_API_KEY="sk-ant-xxx"
go run ./examples/plan-explore-ui -mode=cli
```

示例输出(截断):

```text
Plan/Explore UI demo agent created: agt-...

User:
请帮我完成一个两阶段的代码分析任务:
...

--- Assistant (streaming with Plan/Explore UI) ---

Assistant: 我会先为你规划分析 builtin 工具的步骤,然后逐步探索具体实现。

Plan(分析 builtin 工具测试需求)
  └─ Read(pkg/tools/builtin/todowrite.go)
     Read(pkg/tools/builtin/exitplanmode.go)

Explore(分析当前工具实现状态)
  └─ Read(pkg/tools/builtin/task.go)
     Read(pkg/tools/builtin/subagent_manager.go)

[Done] Step 3 - Reason: completed
```

---

## 与 Task / 子代理系统结合

本示例使用的是「单 Agent + TodoList 中间件」模式。如果结合前文实现的 `Task` + `aster subagent` 子代理框架:

- 顶层 Agent 使用 `Task` 工具启动 `Plan` / `Explore` 子代理:

  ```yaml
  tool_use:
    name: Task
    parameters:
      subagent_type: "Plan"
      prompt: "分析 builtin 工具测试需求"
  ```

- 子代理进程内部可以复用本示例的 UI 逻辑,在自己的终端中展示更细粒度的 Plan/Explore 视图。
