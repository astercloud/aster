---
title: MCP HTTP Server 示例
description: 使用 pkg/mcpserver 暴露本地工具为 MCP 服务
navigation: false
---

# MCP HTTP Server 示例

aster 已经支持作为 MCP 客户端连接外部 MCP Server (`pkg/tools/mcp` + `pkg/sandbox/cloud`)。
本示例展示如何使用 `pkg/mcpserver` 将本地工具暴露为一个兼容的 MCP HTTP Server, 方便 IDE/Agent 统一访问项目文档与工具。

示例代码路径:

- `pkg/mcpserver/server.go`
- `examples/mcp/server/main.go`
- 客户端示例: `examples/mcp/main.go`

## 1. MCP 协议概览

当前实现支持两类 JSON-RPC 方法(与 `MCPClient` 一致):

- `tools/list` – 列出可用工具
  - 请求: `{ "jsonrpc": "2.0", "method": "tools/list", "id": 1 }`
  - 响应: `{ "jsonrpc": "2.0", "id": 1, "result": { "tools": [ { name, description, inputSchema }, ... ] } }`
- `tools/call` – 调用指定工具
  - 请求:
    ```json
    {
      "jsonrpc": "2.0",
      "method": "tools/call",
      "id": 1,
      "params": {
        "name": "echo",
        "arguments": { "text": "hello" }
      }
    }
    ```
  - 响应:
    ```json
    { "jsonrpc": "2.0", "id": 1, "result": "hello" }
    ```

## 2. Server 创建: `pkg/mcpserver`

```go
// pkg/mcpserver/server.go

type Server struct {
    registry *tools.Registry
    executor *tools.Executor
    contextFactory func(ctx context.Context) *tools.ToolContext
}

type Config struct {
    Registry       *tools.Registry
    Executor       *tools.Executor                  // 可选, 为空时创建默认 Executor
    ContextFactory func(ctx context.Context) *tools.ToolContext // 可选, 用于构造 ToolContext
}

func New(cfg *Config) (*Server, error)

// Handler 返回一个 http.Handler, 处理 MCP JSON-RPC 请求:
// - tools/list
// - tools/call
func (s *Server) Handler() http.Handler
```

内部实现直接复用了 `pkg/sandbox/cloud.MCPRequest/MCPResponse` 的结构, 保证与 `MCPClient` 完全兼容。

## 3. 示例工具: EchoTool

```go
// EchoTool 简单回显工具, 用于演示 MCP 工具调用
type EchoTool struct{}

func (t *EchoTool) Name() string        { return "echo" }
func (t *EchoTool) Description() string { return "Echo the input text with an optional prefix." }

func (t *EchoTool) InputSchema() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "text": {
                "type":        "string",
                "description": "Text to echo.",
            },
            "prefix": {
                "type":        "string",
                "description": "Optional prefix to add before the text.",
            },
        },
        "required": []string{"text"},
    }
}

func (t *EchoTool) Execute(ctx context.Context, input map[string]interface{}, tc *tools.ToolContext) (interface{}, error) {
    text, _ := input["text"].(string)
    prefix, _ := input["prefix"].(string)

    if prefix != "" {
        return fmt.Sprintf("%s%s", prefix, text), nil
    }
    return text, nil
}
```

此外, `pkg/mcpserver` 还提供了两个与文档/项目相关的辅助工具构造函数:

```go
// NewDocsGetTool(baseDir string)  -> Tool 名为 "docs_get"
// NewDocsSearchTool(baseDir string) -> Tool 名为 "docs_search"
//
// - docs_get: 读取 baseDir 下的文档文件
// - docs_search: 在 baseDir 下搜索包含关键字的行
```

这两个工具可以配合 MCP 使用, 将你的 docs 或项目目录暴露给 IDE/Agent。

## 4. 启动 MCP Server: `examples/mcp/server/main.go`

```go
func main() {
    // 1. 注册工具
    registry := tools.NewRegistry()
    registry.Register("echo", func(config map[string]interface{}) (tools.Tool, error) {
        return &EchoTool{}, nil
    })

    // 可选: 注册文档工具, 例如基于 ./docs/content
    if docsGet, err := mcpserver.NewDocsGetTool("./docs/content"); err == nil {
        registry.Register(docsGet.Name(), func(cfg map[string]interface{}) (tools.Tool, error) {
            return docsGet, nil
        })
    }
    if docsSearch, err := mcpserver.NewDocsSearchTool("./docs/content"); err == nil {
        registry.Register(docsSearch.Name(), func(cfg map[string]interface{}) (tools.Tool, error) {
            return docsSearch, nil
        })
    }

    // 2. 创建 MCP Server
    srv, err := mcpserver.New(&mcpserver.Config{
        Registry: registry,
    })
    if err != nil {
        log.Fatalf("create MCP server failed: %v", err)
    }

    mux := http.NewServeMux()
    mux.Handle("/mcp", srv.Handler())

    addr := ":8090"
    httpServer := &http.Server{
        Addr:              addr,
        Handler:           mux,
        ReadHeaderTimeout: 10 * time.Second,
    }

    fmt.Printf("MCP HTTP server started at http://localhost%s/mcp\n", addr)
    fmt.Println("Supports tools/list and tools/call JSON-RPC methods.")

    if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatalf("MCP server failed: %v", err)
    }
}
```

运行:

```bash
cd examples/mcp/server
go run main.go
```

MCP Server 将监听 `:8090/mcp`, 提供 `echo` 工具。

## 5. 客户端对接: `examples/mcp/main.go`

`examples/mcp/main.go` 中的客户端示例已经使用 `MCPClient` 连接 MCP Server:

```go
mcpEndpoint := os.Getenv("MCP_ENDPOINT")
if mcpEndpoint == "" {
    mcpEndpoint = "http://localhost:8090/mcp" // 默认指向本地 MCP Server 示例
}

server, err := mcpManager.AddServer(&mcp.MCPServerConfig{
    ServerID:        "my-mcp-server",
    Endpoint:        mcpEndpoint,
    AccessKeyID:     mcpAccessKey,
    AccessKeySecret: mcpSecretKey,
})
```

你可以先启动本地 MCP Server 示例, 再运行 `examples/mcp`:

```bash
# 1. 启动 MCP Server
cd examples/mcp/server
go run main.go

# 2. 运行 Agent + MCP 客户端示例
cd examples/mcp
go run main.go
```

Agent 会通过 `MCPManager` 发现并注册 `echo` 工具, 然后就可以在对话中调用。

## 6. 下一步: 文档/项目 MCP Server

当前示例只是一个简单的 echo 工具, 但 MCP Server 的实现是通用的, 你可以:

- 注册更多的业务工具, 例如:
  - `docs_get` / `docs_search`: 读取/检索本地 Markdown 文档(项目文档、知识库)。
  - `project_index`: 构建/查询项目索引, 提供给 IDE 或其他 Agent 使用。
- 在 IDE 中配置 MCP(如 Cursor/Windsurf/Claude Code), 指向这个本地 MCP Server, 让 IDE 内的 Agent 直接调用你定义的工具。

这类 MCP Server 的目标是: 通过 MCP 将框架/项目/业务知识暴露给更广泛的 Agent 生态使用。
