# MCP 集成示例

演示如何在 Agent SDK 中集成和使用 MCP (Model Context Protocol) 工具。

## 功能特性

- 🔌 连接到 MCP Server
- 🔍 自动发现 MCP 工具
- 📝 将 MCP 工具注册到 Agent 工具注册表
- 🤖 Agent 可以同时使用内置工具和 MCP 工具
- 📊 显示 MCP 统计信息

## 前置条件

1. **MCP Server** (可选)
   - 需要一个运行中的 MCP Server
   - 如果没有 MCP Server，示例仍然可以运行，只使用内置工具

2. **环境变量**
   ```bash
   export ANTHROPIC_API_KEY="your-api-key"
   export MCP_ENDPOINT="http://localhost:8090/mcp"   # 可选, 指向本地 MCP Server 示例
   export MCP_ACCESS_KEY="your-access-key"            # 可选
   export MCP_SECRET_KEY="your-secret-key"            # 可选
   ```

## 运行示例

```bash
# 1. 启动本地 MCP Server 示例
cd examples/mcp/server
go run main.go

# 2. 在另一个终端窗口运行 Agent 示例
cd examples/mcp
go run main.go
```

## MCP 架构

```
┌─────────────────────────────────────────────────┐
│                  Agent                          │
│  ┌──────────────────────────────────────────┐  │
│  │         Tool Registry                    │  │
│  │  ┌──────────┐  ┌─────────────────────┐  │  │
│  │  │ Built-in │  │   MCP Tools         │  │  │
│  │  │ Tools    │  │  (Auto-discovered)  │  │  │
│  │  └──────────┘  └─────────────────────┘  │  │
│  └──────────────────────────────────────────┘  │
└─────────────────┬───────────────────────────────┘
                  │
         ┌────────┴────────┐
         │                 │
    ┌────▼────┐      ┌─────▼─────┐
    │ Local   │      │ MCP       │
    │ Executor│      │ Adapter   │
    └─────────┘      └─────┬─────┘
                           │
                     ┌─────▼─────┐
                     │ MCP Client│
                     └─────┬─────┘
                           │ HTTP/JSON-RPC
                     ┌─────▼─────┐
                     │ MCP Server│
                     └───────────┘
```

## MCP Manager 使用

### 1. 创建 MCP Manager

```go
import "github.com/astercloud/aster/pkg/tools/mcp"

toolRegistry := tools.NewRegistry()
mcpManager := mcp.NewMCPManager(toolRegistry)
```

### 2. 添加 MCP Server

```go
server, err := mcpManager.AddServer(&mcp.MCPServerConfig{
    ServerID:        "my-server",
    Endpoint:        "http://localhost:8080/mcp",
    AccessKeyID:     "your-access-key",
    AccessKeySecret: "your-secret-key",
})
```

### 3. 连接并注册工具

```go
ctx := context.Background()

// 连接单个 Server
err := mcpManager.ConnectServer(ctx, "my-server")

// 或者连接所有 Server
err := mcpManager.ConnectAll(ctx)
```

### 4. 使用 MCP 工具

MCP 工具会被自动注册到 Tool Registry，工具名称格式为 `{server_id}:{tool_name}`，例如:

- `my-server:calculator`
- `my-server:WebSearch`
- `my-server:database_query`

Agent 可以像使用内置工具一样使用这些 MCP 工具。

## MCP 工具适配器

### 手动创建 MCP 工具

```go
import (
    "github.com/astercloud/aster/pkg/sandbox/cloud"
    "github.com/astercloud/aster/pkg/tools/mcp"
)

// 创建 MCP 客户端
client := cloud.NewMCPClient(&cloud.MCPClientConfig{
    Endpoint: "http://localhost:8080/mcp",
})

// 创建工具适配器
tool := mcp.NewMCPToolAdapter(&mcp.MCPToolAdapterConfig{
    Client:      client,
    Name:        "calculator",
    Description: "A simple calculator",
    InputSchema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "operation": map[string]interface{}{"type": "string"},
            "a": map[string]interface{}{"type": "number"},
            "b": map[string]interface{}{"type": "number"},
        },
    },
})

// 手动注册到 Registry
toolRegistry.Register("calculator", func(config map[string]interface{}) (tools.Tool, error) {
    return tool, nil
})
```

## MCP Server 示例

如果你想测试 MCP 功能但没有现成的 MCP Server，可以参考以下简单的 MCP Server 实现:

```python
# simple_mcp_server.py
from flask import Flask, request, jsonify

app = Flask(__name__)

@app.route('/mcp', methods=['POST'])
def mcp_handler():
    req = request.json
    method = req.get('method')

    if method == 'tools/list':
        return jsonify({
            'jsonrpc': '2.0',
            'id': req['id'],
            'result': {
                'tools': [
                    {
                        'name': 'echo',
                        'description': 'Echo back the input',
                        'inputSchema': {
                            'type': 'object',
                            'properties': {
                                'message': {'type': 'string'}
                            }
                        }
                    }
                ]
            }
        })

    elif method == 'tools/call':
        tool_name = req['params']['name']
        args = req['params'].get('arguments', {})

        if tool_name == 'echo':
            return jsonify({
                'jsonrpc': '2.0',
                'id': req['id'],
                'result': {'output': f"Echo: {args.get('message', '')}"}
            })

    return jsonify({
        'jsonrpc': '2.0',
        'id': req['id'],
        'error': {'code': -32601, 'message': 'Method not found'}
    })

if __name__ == '__main__':
    app.run(port=8080)
```

运行:

```bash
python simple_mcp_server.py
```

## API 参考

### MCPManager

- `NewMCPManager(registry)` - 创建 Manager
- `AddServer(config)` - 添加 Server
- `ConnectServer(ctx, serverID)` - 连接指定 Server
- `ConnectAll(ctx)` - 连接所有 Server
- `GetServer(serverID)` - 获取 Server
- `ListServers()` - 列出所有 Server ID
- `RemoveServer(serverID)` - 移除 Server
- `GetServerCount()` - 获取 Server 数量
- `GetTotalToolCount()` - 获取总工具数

### MCPServer

- `GetServerID()` - 获取 Server ID
- `Connect(ctx)` - 连接并发现工具
- `RegisterTools()` - 注册工具到 Registry
- `ListTools()` - 列出已发现的工具
- `GetToolCount()` - 获取工具数量
- `GetClient()` - 获取底层 MCP 客户端

### MCPToolAdapter

实现 `tools.Tool` 接口:

- `Name()` - 工具名称
- `Description()` - 工具描述
- `InputSchema()` - 输入 Schema
- `Prompt()` - 使用说明
- `Execute(ctx, input, tc)` - 执行工具

## 故障排除

### 连接失败

```
⚠️  连接 MCP Server 失败: dial tcp: connection refused
```

**解决方案:**

- 确保 MCP Server 正在运行
- 检查 `MCP_ENDPOINT` 环境变量是否正确
- 验证网络连接和防火墙设置

### 认证失败

```
⚠️  MCP error: unauthorized (code: 401)
```

**解决方案:**

- 检查 `MCP_ACCESS_KEY` 和 `MCP_SECRET_KEY`
- 确认 MCP Server 的认证配置

### 工具未找到

如果 Agent 无法找到 MCP 工具:

1. 检查工具是否已注册: `toolRegistry.List()`
2. 确认工具名称格式: `{server_id}:{tool_name}`
3. 查看 MCP Server 返回的工具列表: `server.ListTools()`

## 相关文档

- [MCP 协议规范](https://modelcontextprotocol.io)
- [Agent SDK 文档](../../README.md)
- [工具开发指南](../../docs/tools.md)
