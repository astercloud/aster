# Programmatic Tool Calling (PTC)

Programmatic Tool Calling (PTC) 是 Aster 实现的 Anthropic 协议扩展,允许 LLM 生成的 Python 代码直接调用 Aster 工具,实现更强大的编程能力。

## 概述

传统的工具调用流程:
```
LLM → 工具调用请求 → Aster 执行工具 → 返回结果 → LLM
```

PTC 流程:
```
LLM → 生成 Python 代码 → CodeExecute 工具执行代码 →
代码中调用 Aster 工具 → 返回结果 → LLM
```

### 优势

1. **组合能力**: Python 代码可以组合多个工具调用,实现复杂逻辑
2. **控制流**: 支持条件判断、循环等控制结构
3. **数据处理**: 利用 Python 生态处理复杂数据转换
4. **错误处理**: 代码中可以捕获和处理工具调用错误

## 架构

```
┌─────────────────────────────────────────────────────────────┐
│                         LLM (Anthropic)                      │
│  生成 Python 代码,调用 Read/Write/Glob/Grep 等工具           │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                    CodeExecute 工具                          │
│  - 启动 HTTP 桥接服务器                                       │
│  - 注入工具 SDK 到 Python 代码                                │
│  - 执行 Python 代码                                          │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                  HTTP Bridge Server                          │
│  端点:                                                        │
│  - POST /tools/call    - 调用工具                            │
│  - GET  /tools/list    - 列出可用工具                         │
│  - GET  /tools/schema  - 获取工具 Schema                     │
│  - GET  /health        - 健康检查                            │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                      ToolBridge                              │
│  调用 Go 侧的工具实现                                         │
└─────────────────────────────────────────────────────────────┘
```

## 快速开始

### 1. 基本使用

创建一个带 PTC 支持的 Agent:

```go
package main

import (
    "context"
    "log"

    "github.com/astercloud/aster/pkg/agent"
    "github.com/astercloud/aster/pkg/provider"
    "github.com/astercloud/aster/pkg/tools"
    "github.com/astercloud/aster/pkg/tools/bridge"
    "github.com/astercloud/aster/pkg/tools/builtin"
)

func main() {
    // 1. 创建工具注册表
    registry := tools.NewRegistry()
    builtin.RegisterAll(registry)

    // 2. 创建 ToolBridge
    toolBridge := bridge.NewToolBridge(registry)

    // 3. 创建 CodeExecute 工具(带 PTC 支持)
    codeExecTool := builtin.NewCodeExecuteToolWithBridge(toolBridge)

    // 4. 创建 Provider
    providerConfig := &types.ModelConfig{
        Provider: "anthropic",
        Model:    "claude-3-5-sonnet-20241022",
        APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
    }
    provider, _ := provider.NewAnthropicProvider(providerConfig)

    // 5. 创建 Agent
    ag := agent.NewAgent("ptc-demo", provider, &agent.Dependencies{
        ToolRegistry: registry,
    })

    // 6. 注册 CodeExecute 工具
    ag.AddTool(codeExecTool)

    // 7. 运行任务
    ctx := context.Background()
    ag.Run(ctx, "请用 Python 代码读取当前目录下所有 .go 文件,统计总行数")
}
```

### 2. LLM 生成的 Python 代码示例

```python
# LLM 会生成类似这样的代码
import asyncio

async def main():
    # 搜索所有 .go 文件
    go_files = await Glob(pattern="*.go", path=".")

    total_lines = 0
    for file_path in go_files:
        # 读取文件内容
        content = await Read(path=file_path)
        lines = len(content.split('\n'))
        total_lines += lines
        print(f"{file_path}: {lines} 行")

    print(f"总计: {total_lines} 行")

asyncio.run(main())
```

### 3. 工具配置 AllowedCallers

默认情况下,所有工具仅支持 LLM 直接调用。要允许工具在 Python 代码中被调用,需要配置 `AllowedCallers`:

```go
// 创建工具时指定 AllowedCallers
type ReadTool struct{}

func (t *ReadTool) Schema() provider.ToolSchema {
    return provider.ToolSchema{
        Name:        "Read",
        Description: "读取文件内容",
        InputSchema: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "path": map[string]any{
                    "type": "string",
                    "description": "文件路径",
                },
            },
            "required": []string{"path"},
        },
        // PTC 配置: 允许 LLM 直接调用 + Python 代码调用
        AllowedCallers: []string{"direct", "code_execution_20250825"},
    }
}
```

AllowedCallers 可选值:
- `"direct"`: 允许 LLM 直接调用(默认)
- `"code_execution_20250825"`: 允许在 CodeExecute 生成的代码中调用

## 可用工具

以下内置工具默认支持 PTC:

| 工具名 | 功能 | 示例 |
|--------|------|------|
| `Read` | 读取文件 | `await Read(path="file.txt")` |
| `Write` | 写入文件 | `await Write(path="out.txt", content="data")` |
| `Glob` | 文件模式匹配 | `await Glob(pattern="*.py")` |
| `Grep` | 内容搜索 | `await Grep(pattern="TODO", path=".")` |
| `Bash` | 执行命令 | `await Bash(command="ls -la")` |

## 高级用法

### 1. 自定义桥接 URL

```go
codeExecTool := builtin.NewCodeExecuteToolWithBridge(toolBridge)
codeExecTool.SetBridgeURL("http://localhost:9000")
```

### 2. 工具上下文传递

HTTP 桥接服务器支持工具上下文工厂,可以为每次工具调用提供不同的上下文:

```go
httpServer := bridge.NewHTTPBridgeServer(toolBridge, "localhost:8080")

httpServer.SetContextFactory(func() *tools.ToolContext {
    return &tools.ToolContext{
        AgentID: "agent-123",
        Services: map[string]any{
            "database": db,
            "cache":    cache,
        },
    }
})
```

### 3. 错误处理

Python 代码中可以捕获工具调用错误:

```python
async def main():
    try:
        content = await Read(path="nonexistent.txt")
    except Exception as e:
        print(f"读取失败: {e}")
        # 使用备用方案
        content = await Read(path="default.txt")
```

### 4. 批量工具调用

```python
async def main():
    # 并发读取多个文件
    files = ["a.txt", "b.txt", "c.txt"]
    tasks = [Read(path=f) for f in files]

    import asyncio
    results = await asyncio.gather(*tasks)

    for file, content in zip(files, results):
        print(f"{file}: {len(content)} 字节")
```

## 限制和注意事项

### 1. Python 依赖

生成的 Python 代码依赖 `aiohttp` 库。如果执行环境没有安装,会报错:

```
Error: aiohttp is required. Install it with: pip install aiohttp
```

解决方案:
```bash
pip install aiohttp
```

### 2. 超时设置

- HTTP 请求超时: 60秒
- Python 代码执行超时: 30秒(可配置)

```go
config := &bridge.RuntimeConfig{
    Timeout: 60 * time.Second,
}
runtime := bridge.NewPythonRuntime(config)
```

### 3. 安全考虑

- CodeExecute 工具会执行 LLM 生成的任意 Python 代码,存在安全风险
- 建议在沙箱环境中运行,或使用白名单限制可调用的工具
- HTTP 桥接服务器默认监听 localhost,不对外暴露

### 4. 性能影响

- 首次调用 CodeExecute 时会启动 HTTP 服务器(约 100ms)
- 每次工具调用都是 HTTP 请求,有网络开销(约 1-5ms)
- Python 解释器启动有开销(约 50-100ms)

## 调试

### 启用详细日志

```go
import "log"

// Anthropic Provider 会输出工具调用详情
log.SetFlags(log.LstdFlags | log.Lshortfile)
```

### 查看生成的 Python 代码

CodeExecute 工具执行的完整 Python 代码会包含:
1. SDK 注入代码
2. 工具函数生成
3. 用户代码包装

可以在 stderr 中查看执行错误。

### HTTP 请求日志

HTTP 桥接服务器会输出:
```
HTTP Bridge Server listening on localhost:8080
```

## 示例项目

参考 `examples/ptc/` 目录下的完整示例:

- `basic/`: 基础 PTC 使用
- `file-processor/`: 文件批处理
- `code-analyzer/`: 代码分析工具

## 常见问题

### Q: PTC 和普通工具调用有什么区别?

A:
- 普通工具调用: LLM 一次调用一个工具,适合简单场景
- PTC: LLM 生成 Python 代码,可以组合多个工具,适合复杂逻辑

### Q: 为什么只支持 Python?

A: Python 是 AI 领域的标准语言,大部分 LLM 对 Python 支持最好。Anthropic、OpenAI、Manus 等平台都使用 Python。

### Q: 如何调试 Python 代码错误?

A: CodeExecute 工具会返回完整的 stdout 和 stderr,包含 Python 异常堆栈。

### Q: 能在 Python 代码中调用自定义工具吗?

A: 可以!只要工具在 ToolRegistry 中注册,并设置了正确的 AllowedCallers,就可以在 Python 中调用。

## 参考资料

- [Anthropic PTC 文档](https://docs.anthropic.com/en/docs/build-with-claude/tool-use#programmatic-tool-use-beta)
- [Aster 工具系统](./tools.md)
- [CodeExecute 工具文档](./tools/code-execute.md)

## 更新日志

- 2025-01-30: 初始版本,支持 Python PTC
- 2025-01-30: 添加 HTTP 桥接服务器
- 2025-01-30: 集成 Anthropic Provider
