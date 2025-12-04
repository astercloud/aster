# Working Memory 示例

本示例演示如何使用 aster 的 Working Memory 系统来管理跨会话状态。

## Working Memory 简介

Working Memory 是一个持久化的、结构化的状态管理系统，用于：

- 跟踪当前会话状态
- 存储用户偏好和上下文
- 管理多步骤任务的进度
- 在多轮对话间保持状态

### 与文本记忆的区别

| 特性            | Working Memory    | 文本记忆     |
| --------------- | ----------------- | ------------ |
| **用途**        | 会话状态管理      | 长期知识库   |
| **大小**        | 小（< 500 words） | 大（无限制） |
| **更新方式**    | 完全覆盖          | 追加或覆盖   |
| **自动加载**    | 是                | 否           |
| **Schema 验证** | 支持              | 不支持       |

## 示例内容

### 1. Thread Scope（会话级）

每个 thread（会话）有独立的 Working Memory：

```go
// Alice 的会话
manager.Update(ctx, "thread-001", "resource", aliceProfile)

// Bob 的会话
manager.Update(ctx, "thread-002", "resource", bobProfile)

// 两个会话的 Working Memory 互不干扰
```

**适用场景：**

- 独立的用户会话
- 不同上下文的对话
- 需要隔离状态的情况

### 2. Resource Scope（资源级）

同一 resource 的所有 threads 共享 Working Memory：

```go
// Thread 1 更新文章状态
manager.Update(ctx, "edit-001", "article-123", initialState)

// Thread 2 读取相同的状态
state, _ := manager.Get(ctx, "edit-002", "article-123")

// Thread 2 的更新会影响所有会话
manager.Update(ctx, "edit-002", "article-123", updatedState)
```

**适用场景：**

- 多轮协作编辑
- 团队共享的项目状态
- 长期追踪的资源

### 3. Schema 验证

使用 JSON Schema 确保数据一致性：

```go
schema := &memory.JSONSchema{
    Type: "object",
    Properties: map[string]*memory.JSONSchema{
        "user_name":   {Type: "string"},
        "task_status": {
            Type: "string",
            Enum: []interface{}{"not_started", "in_progress", "completed"},
        },
    },
    Required: []string{"user_name", "task_status"},
}

manager, _ := memory.NewWorkingMemoryManager(&memory.WorkingMemoryConfig{
    Schema: schema,
    // ...
})

// 有效的 JSON 会被接受
manager.Update(ctx, threadID, resourceID, `{
    "user_name": "Alice",
    "task_status": "in_progress"
}`)

// 无效的 JSON 会被拒绝
manager.Update(ctx, threadID, resourceID, `{
    "task_status": "in_progress"
}`)  // 错误：缺少 required 字段
```

### 4. Find and Replace（实验性）

增量更新 Working Memory：

```go
// 初始状态
manager.Update(ctx, threadID, resourceID, `
# Task: Complete feature X
Status: in_progress
`)

// Find and Replace
manager.FindAndReplace(ctx, threadID, resourceID,
    "Status: in_progress",
    "Status: completed")

// Append（search string 为空）
manager.FindAndReplace(ctx, threadID, resourceID,
    "",
    "Next steps: Deploy to production")
```

### 5. Middleware 集成

通过 Middleware 将 Working Memory 集成到 Agent：

```go
wmMiddleware, _ := middleware.NewWorkingMemoryMiddleware(
    &middleware.WorkingMemoryMiddlewareConfig{
        Backend:  backend,
        Scope:    memory.ScopeThread,
    },
)

// Working Memory 会自动：
// 1. 在每轮对话时加载并注入到 system prompt
// 2. 提供 update_working_memory 工具
// 3. 根据 threadID/resourceID 自动选择正确的状态
```

## 运行示例

```bash
cd examples/memory-working
go run main.go
```

## 输出示例

```
=== 示例 1: Thread Scope Working Memory ===
✓ Thread 1 (Alice) Working Memory 已更新
✓ Thread 2 (Bob) Working Memory 已更新

📝 Thread 1 (Alice) 的 Working Memory:
# User Profile
Name: Alice
Role: Software Engineer
...

📝 Thread 2 (Bob) 的 Working Memory:
# User Profile
Name: Bob
Role: Product Manager
...

✅ Thread Scope: 每个会话有独立的 Working Memory

=== 示例 2: Resource Scope Working Memory ===
✓ Thread 1 更新了文章状态

📝 Thread 2 读取到的状态（来自同一 resource）:
# Article: Getting Started with aster
...

✅ Resource Scope: 同一资源的所有会话共享 Working Memory

=== 示例 3: 带 Schema 的 Working Memory ===
✓ 有效的 JSON 更新成功
✓ 无效的 JSON 被拒绝（预期行为）: schema validation failed: required field 'task_status' is missing

✅ Schema 验证确保数据一致性

=== 示例 4: Find and Replace（实验性）===
✓ 初始状态已设置
✓ 状态已更新（find and replace）
✓ 任务已完成（find and replace）
✓ 新任务已添加（append）

✅ Find and Replace 实现增量更新
```

## 最佳实践

### ✅ 推荐

1. **保持简洁** - Working Memory 应该 < 500 words
2. **使用清晰的结构** - Markdown 标题和列表
3. **定期清理** - 删除不再需要的信息
4. **使用 Schema** - 确保数据一致性
5. **选择合适的 Scope** - Thread 用于独立会话，Resource 用于共享状态

### ❌ 避免

1. **存储大量历史** - Working Memory 不是日志系统
2. **忘记覆盖** - `Update` 会替换整个内容，确保包含所有要保留的信息
3. **混淆 Scope** - 理解 thread 和 resource 的区别
4. **频繁更新** - 仅在状态真正改变时更新

## 配置

在 `agentsdk.yaml` 中配置 Working Memory：

```yaml
memory:
  working_memory:
    enabled: true
    scope: "thread" # "thread" | "resource"
    base_path: "/working_memory/"
    ttl: 0 # 过期时间（秒），0表示不过期

    # 可选：JSON Schema
    schema:
      type: object
      properties:
        user_name: { type: string }
        task_status:
          type: string
          enum: ["not_started", "in_progress", "completed"]
      required: ["user_name"]
```

## 相关文档

- [Memory 系统指南](../../docs/content/4.guides/memory.md)
- [Memory API Reference](../../docs/content/6.api-reference/memory-api.md)
- [Middleware API Reference](../../docs/content/6.api-reference/3.middleware-api.md)

## 其他示例

- [examples/memory/](../memory/) - 基础文本记忆示例
- [examples/memory-agent/](../memory-agent/) - 带记忆的 Agent 示例
- [examples/memory-semantic/](../memory-semantic/) - 语义记忆示例
