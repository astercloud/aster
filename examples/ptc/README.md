# PTC (Programmatic Tool Calling) 示例

本目录包含 Aster Programmatic Tool Calling 功能的示例程序。

## 前置条件

1. **Anthropic API Key**
   ```bash
   export ANTHROPIC_API_KEY="your-api-key-here"
   ```

2. **Python 环境** (用于执行生成的代码)
   ```bash
   python3 --version  # 需要 Python 3.7+
   pip install aiohttp  # 必需依赖
   ```

## 示例列表

### 1. basic - 基础 PTC 使用

最简单的 PTC 示例,演示如何让 LLM 生成 Python 代码并调用 Aster 工具。

**运行:**
```bash
cd basic
go run main.go
```

**功能:**
- 使用 Glob 查找所有 .go 文件
- 使用 Read 读取文件内容
- 统计代码行数和字符数

**预期输出:**
```
正在调用 LLM 生成 Python 代码...

=== LLM 响应 ===
工具调用: CodeExecute
参数: map[language:python code:import asyncio...]

执行结果: map[success:true output:...]

=== Token 使用 ===
输入: 1234 tokens
输出: 567 tokens
总计: 1801 tokens
```

### 2. file-processor - 文件批处理

演示如何使用 PTC 进行批量文件处理。

**运行:**
```bash
cd file-processor
go run main.go
```

**功能:**
- 批量读取多个文件
- 数据转换和处理
- 并发执行提升性能

### 3. code-analyzer - 代码分析

演示复杂的代码分析任务。

**运行:**
```bash
cd code-analyzer
go run main.go
```

**功能:**
- 使用 Grep 搜索特定模式
- 统计代码复杂度
- 生成分析报告

## 工作原理

```
┌─────────────────────────────────────────────────────────────┐
│  1. Go 程序创建 ToolBridge 和 CodeExecute 工具               │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  2. 调用 LLM,提供工具列表(带 AllowedCallers 配置)            │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  3. LLM 生成 Python 代码,调用 Read/Write/Glob 等工具         │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  4. CodeExecute 工具:                                        │
│     - 启动 HTTP 桥接服务器 (localhost:8080)                  │
│     - 注入 Python SDK 到生成的代码                           │
│     - 执行 Python 代码                                       │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  5. Python 代码通过 HTTP 调用 Go 侧工具                       │
│     POST http://localhost:8080/tools/call                   │
│     {"tool": "Read", "input": {"path": "file.txt"}}         │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  6. 返回结果给 Python 代码                                    │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  7. Python 代码完成执行,返回结果                              │
└─────────────────────────────────────────────────────────────┘
```

## 关键配置

### AllowedCallers 字段

控制工具在哪些上下文中可以被调用:

```go
toolSchemas := []provider.ToolSchema{
    {
        Name:        "Read",
        Description: "读取文件",
        InputSchema: readTool.InputSchema(),
        // 允许 LLM 直接调用 + Python 代码调用
        AllowedCallers: []string{"direct", "code_execution_20250825"},
    },
    {
        Name:        "CodeExecute",
        Description: "执行代码",
        InputSchema: codeExecTool.InputSchema(),
        // 仅允许 LLM 直接调用,不能在 Python 中递归调用
        AllowedCallers: []string{"direct"},
    },
}
```

可选值:
- `"direct"`: LLM 直接调用
- `"code_execution_20250825"`: Python 代码中调用

### Caller 字段

工具调用时会包含调用者信息:

```go
toolUseBlock := &types.ToolUseBlock{
    ID:    "call_123",
    Name:  "Read",
    Input: map[string]any{"path": "file.txt"},
    Caller: &types.ToolCaller{
        Type:   "code_execution_20250825",  // 从代码中调用
        ToolID: "call_456",                 // CodeExecute 工具的 ID
    },
}
```

## 调试技巧

### 1. 查看生成的 Python 代码

CodeExecute 工具会输出完整的 Python 代码到临时文件,可以通过修改代码保留临时文件:

```go
// 在 runtime.go 中注释掉临时文件删除
// defer os.Remove(tmpFile.Name())
fmt.Printf("临时文件: %s\n", tmpFile.Name())
```

### 2. 启用详细日志

```go
import "log"

log.SetFlags(log.LstdFlags | log.Lshortfile)
```

会输出:
- HTTP 桥接服务器启动信息
- 工具调用详情
- Provider API 请求详情

### 3. 测试单个工具

```bash
# 测试 HTTP 桥接服务器
curl -X POST http://localhost:8080/tools/call \
  -H "Content-Type: application/json" \
  -d '{"tool": "Read", "input": {"path": "README.md"}}'

# 列出可用工具
curl http://localhost:8080/tools/list

# 获取工具 Schema
curl "http://localhost:8080/tools/schema?name=Read"
```

## 常见问题

### Q: 报错 "aiohttp is required"

A: 安装 Python 依赖:
```bash
pip install aiohttp
```

### Q: HTTP 桥接服务器启动失败

A: 检查端口占用:
```bash
lsof -i :8080
```

修改端口:
```go
codeExecTool.SetBridgeURL("http://localhost:9000")
```

### Q: Python 代码执行超时

A: 增加超时时间:
```go
config := &bridge.RuntimeConfig{
    Timeout: 60 * time.Second,
}
runtime := bridge.NewPythonRuntime(config)
```

### Q: 如何限制可调用的工具?

A: 只为需要的工具设置 `AllowedCallers`:
```go
// 仅允许 Read 和 Glob 在 Python 中调用
toolSchemas := []provider.ToolSchema{
    {Name: "Read", AllowedCallers: []string{"direct", "code_execution_20250825"}},
    {Name: "Glob", AllowedCallers: []string{"direct", "code_execution_20250825"}},
    {Name: "Write", AllowedCallers: []string{"direct"}},  // 不能在 Python 中调用
}
```

## 性能优化

### 1. 复用 HTTP 服务器

HTTP 桥接服务器在首次调用 CodeExecute 时启动,后续调用会复用同一个服务器实例。

### 2. 批量工具调用

在 Python 代码中使用 `asyncio.gather` 并发调用:

```python
import asyncio

results = await asyncio.gather(
    Read(path="a.txt"),
    Read(path="b.txt"),
    Read(path="c.txt"),
)
```

### 3. 缓存工具结果

```python
_cache = {}

async def cached_read(path):
    if path not in _cache:
        _cache[path] = await Read(path=path)
    return _cache[path]
```

## 更多资源

- [PTC 完整文档](../../docs/programmatic-tool-calling.md)
- [Anthropic PTC 官方文档](https://docs.anthropic.com/en/docs/build-with-claude/tool-use#programmatic-tool-use-beta)
- [Aster 工具系统](../../docs/tools.md)
