---
title: Tools API
description: Tools工具系统完整参考文档
---

# Tools API 参考

本文档提供 Tools 工具系统的完整参考，涵盖内置工具、自定义工具开发、工具注册等功能。

## 目录

- [接口概览](#接口概览)
- [内置工具](#内置工具)
- [自定义工具](#自定义工具)
- [工具注册](#工具注册)
- [类型定义](#类型定义)

---

## 接口概览

Tool 定义了工具的基本结构：

```go
type Tool struct {
    Name        string                 // 工具名称
    Description string                 // 工具描述
    InputSchema map[string]interface{} // JSON Schema
    Handler     ToolHandler            // 执行函数
    Permissions []string               // 所需权限
}

type ToolHandler func(ctx context.Context, tc *ToolContext) (interface{}, error)
```

---

## 内置工具

### Filesystem Tools

文件系统操作工具集。

**工具列表**：

- `filesystem_read` - 读取文件
- `filesystem_write` - 写入文件
- `filesystem_list` - 列出目录
- `filesystem_delete` - 删除文件
- `filesystem_search` - 搜索文件内容

**注册**：

```go
import "github.com/astercloud/aster/pkg/tools/builtin"

toolRegistry := tools.NewRegistry()
builtin.RegisterAll(toolRegistry)
```

**使用示例**：

```go
// Agent 自动调用
ag.Chat(ctx, "读取 README.md 文件的内容")
// → 调用 filesystem_read 工具

ag.Chat(ctx, "创建一个 hello.txt 文件，内容为 Hello World")
// → 调用 filesystem_write 工具
```

---

### Bash Tool

执行 Shell 命令。

**工具名称**：`bash`

**功能**：

- 执行 Shell 命令
- 支持沙箱隔离
- 超时控制

**输入 Schema**：

```json
{
  "command": "ls -la",
  "timeout": 30
}
```

**使用示例**：

```go
ag.Chat(ctx, "列出当前目录的文件")
// → bash 工具执行: ls -la
```

---

### HTTP Tools

HTTP 请求工具。

**工具列表**：

- `http_get` - GET 请求
- `http_post` - POST 请求
- `http_put` - PUT 请求
- `http_delete` - DELETE 请求

**使用示例**：

```go
ag.Chat(ctx, "获取 https://api.github.com/users/octocat 的数据")
// → http_get 工具
```

---

### WebSearch Tool

网页搜索工具。

**工具名称**：`websearch`

**功能**：

- 搜索网页内容
- 返回搜索结果摘要

**使用示例**：

```go
ag.Chat(ctx, "搜索最新的 Go 1.23 发布信息")
// → websearch 工具
```

---

## 自定义工具

### 基础模板

```go
package mytools

import (
    "context"
    "github.com/astercloud/aster/pkg/tools"
)

func WeatherTool() tools.Tool {
    return tools.Tool{
        Name:        "get_weather",
        Description: "获取指定城市的实时天气信息",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "city": map[string]interface{}{
                    "type":        "string",
                    "description": "城市名称，例如：北京、上海",
                },
                "unit": map[string]interface{}{
                    "type":        "string",
                    "enum":        []string{"celsius", "fahrenheit"},
                    "description": "温度单位",
                    "default":     "celsius",
                },
            },
            "required": []string{"city"},
        },
        Handler: handleWeather,
    }
}

func handleWeather(ctx context.Context, tc *tools.ToolContext) (interface{}, error) {
    // 解析输入
    city, ok := tc.Input["city"].(string)
    if !ok {
        return nil, fmt.Errorf("缺少城市参数")
    }

    unit := "celsius"
    if u, ok := tc.Input["unit"].(string); ok {
        unit = u
    }

    // 调用天气 API
    weather, err := fetchWeather(city, unit)
    if err != nil {
        return nil, err
    }

    // 返回结果
    return map[string]interface{}{
        "city":        city,
        "temperature": weather.Temperature,
        "condition":   weather.Condition,
        "humidity":    weather.Humidity,
    }, nil
}
```

### 注册自定义工具

```go
// 创建注册表
toolRegistry := tools.NewRegistry()

// 注册内置工具
builtin.RegisterAll(toolRegistry)

// 注册自定义工具
toolRegistry.Register(WeatherTool())
toolRegistry.Register(StockTool())
toolRegistry.Register(DatabaseTool())

// 在依赖中使用
deps := &agent.Dependencies{
    ToolRegistry: toolRegistry,
    // ... 其他依赖
}
```

---

## 工具注册

### tools.Registry

工具注册表管理所有可用工具。

#### NewRegistry

创建新的工具注册表。

```go
func NewRegistry() *Registry
```

**示例**：

```go
registry := tools.NewRegistry()
```

---

#### Register

注册单个工具。

```go
func (r *Registry) Register(tool Tool) error
```

**示例**：

```go
err := registry.Register(WeatherTool())
if err != nil {
    log.Fatal(err)
}
```

---

#### RegisterBatch

批量注册工具。

```go
func (r *Registry) RegisterBatch(tools []Tool) error
```

**示例**：

```go
myTools := []tools.Tool{
    WeatherTool(),
    StockTool(),
    DatabaseTool(),
}

err := registry.RegisterBatch(myTools)
```

---

#### Get

获取指定工具。

```go
func (r *Registry) Get(name string) (Tool, error)
```

**示例**：

```go
tool, err := registry.Get("get_weather")
if err != nil {
    log.Fatal(err)
}
```

---

#### List

列出所有已注册工具。

```go
func (r *Registry) List() []Tool
```

**示例**：

```go
tools := registry.List()
for _, tool := range tools {
    fmt.Printf("%s: %s\n", tool.Name, tool.Description)
}
```

---

## 类型定义

### Tool

工具定义结构。

```go
type Tool struct {
    Name        string                 // 工具名称（唯一）
    Description string                 // 工具描述（给 LLM 看的）
    InputSchema map[string]interface{} // JSON Schema 格式的输入定义
    Handler     ToolHandler            // 执行函数
    Permissions []string               // 所需权限列表
    Metadata    map[string]interface{} // 自定义元数据
}
```

---

### ToolContext

工具执行上下文。

```go
type ToolContext struct {
    Input    map[string]interface{} // 工具输入参数
    AgentID  string                 // 调用的 Agent ID
    Sandbox  sandbox.Sandbox        // 沙箱实例
    Store    store.Store            // 持久化存储
    Metadata map[string]interface{} // 元数据
}
```

---

### ToolHandler

工具处理函数。

```go
type ToolHandler func(ctx context.Context, tc *ToolContext) (interface{}, error)
```

**参数**：

- `ctx`: 上下文
- `tc`: 工具上下文

**返回**：

- `interface{}`: 工具执行结果（可序列化为 JSON）
- `error`: 错误信息

---

## 实战示例

### 数据库查询工具

```go
func DatabaseQueryTool(db *sql.DB) tools.Tool {
    return tools.Tool{
        Name:        "database_query",
        Description: "执行 SQL 查询并返回结果",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "sql": map[string]interface{}{
                    "type":        "string",
                    "description": "SQL 查询语句",
                },
                "limit": map[string]interface{}{
                    "type":        "integer",
                    "description": "最大返回行数",
                    "default":     100,
                },
            },
            "required": []string{"sql"},
        },
        Permissions: []string{"database:read"},
        Handler: func(ctx context.Context, tc *tools.ToolContext) (interface{}, error) {
            sql := tc.Input["sql"].(string)
            limit := 100
            if l, ok := tc.Input["limit"].(float64); ok {
                limit = int(l)
            }

            // 安全检查
            if !isReadOnlyQuery(sql) {
                return nil, fmt.Errorf("只允许 SELECT 查询")
            }

            // 执行查询
            rows, err := db.QueryContext(ctx, sql)
            if err != nil {
                return nil, err
            }
            defer rows.Close()

            // 解析结果
            results, err := parseRows(rows, limit)
            if err != nil {
                return nil, err
            }

            return results, nil
        },
    }
}
```

---

### 文件压缩工具

```go
func ZipTool() tools.Tool {
    return tools.Tool{
        Name:        "zip_files",
        Description: "压缩文件或目录为 ZIP 格式",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "files": map[string]interface{}{
                    "type":        "array",
                    "items":       map[string]string{"type": "string"},
                    "description": "要压缩的文件或目录列表",
                },
                "output": map[string]interface{}{
                    "type":        "string",
                    "description": "输出的 ZIP 文件名",
                },
            },
            "required": []string{"files", "output"},
        },
        Handler: func(ctx context.Context, tc *tools.ToolContext) (interface{}, error) {
            files := tc.Input["files"].([]interface{})
            output := tc.Input["output"].(string)

            // 创建 ZIP 文件
            zipFile, err := os.Create(output)
            if err != nil {
                return nil, err
            }
            defer zipFile.Close()

            zipWriter := zip.NewWriter(zipFile)
            defer zipWriter.Close()

            // 添加文件
            for _, file := range files {
                filePath := file.(string)
                if err := addFileToZip(zipWriter, filePath); err != nil {
                    return nil, err
                }
            }

            return map[string]interface{}{
                "output_file": output,
                "file_count":  len(files),
            }, nil
        },
    }
}
```

---

### API 调用工具

```go
func APICallTool(apiKey string) tools.Tool {
    return tools.Tool{
        Name:        "call_api",
        Description: "调用外部 REST API",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "method": map[string]interface{}{
                    "type": "string",
                    "enum": []string{"GET", "POST", "PUT", "DELETE"},
                },
                "url": map[string]interface{}{
                    "type": "string",
                },
                "headers": map[string]interface{}{
                    "type": "object",
                },
                "body": map[string]interface{}{
                    "type": "object",
                },
            },
            "required": []string{"method", "url"},
        },
        Handler: func(ctx context.Context, tc *tools.ToolContext) (interface{}, error) {
            method := tc.Input["method"].(string)
            url := tc.Input["url"].(string)

            // 创建请求
            var body io.Reader
            if b, ok := tc.Input["body"]; ok {
                bodyBytes, _ := json.Marshal(b)
                body = bytes.NewReader(bodyBytes)
            }

            req, err := http.NewRequestWithContext(ctx, method, url, body)
            if err != nil {
                return nil, err
            }

            // 添加认证
            req.Header.Set("Authorization", "Bearer "+apiKey)
            req.Header.Set("Content-Type", "application/json")

            // 添加自定义头
            if headers, ok := tc.Input["headers"].(map[string]interface{}); ok {
                for k, v := range headers {
                    req.Header.Set(k, v.(string))
                }
            }

            // 发送请求
            client := &http.Client{Timeout: 30 * time.Second}
            resp, err := client.Do(req)
            if err != nil {
                return nil, err
            }
            defer resp.Body.Close()

            // 解析响应
            var result interface{}
            if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
                return nil, err
            }

            return result, nil
        },
    }
}
```

---

## 最佳实践

### 1. 清晰的描述

工具描述要清晰，让 LLM 知道何时使用：

```go
// ✅ 好的描述
Description: "读取文件内容。支持文本文件（.txt, .md, .json）和二进制文件（返回 base64）。"

// ❌ 不好的描述
Description: "读取文件"
```

### 2. 完整的 Schema

提供完整的 JSON Schema：

```go
InputSchema: map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{
        "path": map[string]interface{}{
            "type":        "string",
            "description": "文件路径（相对或绝对）",
        },
    },
    "required": []string{"path"},
}
```

### 3. 错误处理

返回有意义的错误信息：

```go
if !fileExists(path) {
    return nil, fmt.Errorf("文件不存在: %s", path)
}

if !hasPermission(path) {
    return nil, fmt.Errorf("无权限访问: %s", path)
}
```

### 4. 超时控制

在长时间运行的操作中检查上下文：

```go
func longRunningTool(ctx context.Context, tc *tools.ToolContext) (interface{}, error) {
    for i := 0; i < 1000; i++ {
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        default:
            // 继续处理
        }
    }
    return result, nil
}
```

### 5. 资源清理

确保资源被正确释放：

```go
func fileOperationTool(ctx context.Context, tc *tools.ToolContext) (interface{}, error) {
    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer file.Close() // 确保文件被关闭

    // 处理文件...
}
```

---

## 相关资源

- [Agent API 文档](./1.agent-api.md)
- [内置工具详解](../guides/tools/builtin.md)
- [自定义工具开发](../guides/tools/custom.md)
- [MCP 工具集成](../guides/tools/mcp.md)
- [完整 API 文档 (pkg.go.dev)](https://pkg.go.dev/github.com/astercloud/aster/pkg/tools)
