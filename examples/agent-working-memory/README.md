# Agent + Working Memory 完整示例

本示例演示如何创建一个带 Working Memory 的 Agent，实现真正的跨会话状态管理。

## 什么是 Working Memory？

Working Memory 是一个持久化的、结构化的状态管理系统，它能够：

- **自动加载**：每轮对话开始时自动加载到 system prompt
- **LLM 控制**：Agent 可以通过 `update_working_memory` 工具主动更新
- **作用域隔离**：支持 thread 和 resource 两种作用域
- **Schema 验证**：可选的 JSON Schema 确保数据一致性

## 与普通记忆的区别

| 特性         | Working Memory        | 文本记忆 (agent_memory) |
| ------------ | --------------------- | ----------------------- |
| **自动加载** | ✅ 每轮对话自动加载   | ❌ 需要 LLM 主动读取    |
| **更新方式** | 完全覆盖              | 追加或覆盖              |
| **大小**     | 小（< 500 words）     | 大（无限制）            |
| **结构**     | 结构化（可选 Schema） | 自由文本                |
| **用途**     | 会话状态管理          | 长期知识库              |

## 示例场景

本示例模拟一个**任务助手 Agent**，它能够：

1. 记住用户的偏好和设置
2. 跟踪当前任务的进度
3. 在多轮对话间保持状态

## 目录结构

```
examples/agent-working-memory/
├── README.md           # 本文档
├── main.go             # 完整示例代码
└── config.yaml         # Agent 配置文件（可选）
```

## 配置

### 1. agentsdk.yaml 配置

```yaml
memory:
  working_memory:
    enabled: true
    scope: "thread" # 每个会话独立的状态
    base_path: "/working_memory/"

    # 可选：定义 Schema 确保数据一致性
    schema:
      type: object
      properties:
        user_name:
          type: string
        preferences:
          type: object
          properties:
            language: { type: string }
            verbosity: { type: string }
        current_task:
          type: object
          properties:
            name: { type: string }
            status: { type: string }
            progress: { type: integer }
      required: ["user_name"]
```

### 2. 在代码中启用

```go
config := &types.AgentConfig{
    TemplateID: "task-assistant",
    ModelConfig: &types.ModelConfig{
        Provider: "anthropic",
        Model:    "claude-sonnet-4-5",
        APIKey:   apiKey,
    },
    // 启用 working_memory 中间件
    Middlewares: []string{
        "filesystem",
        "agent_memory",
        "working_memory",  // ← 关键！
    },
}
```

## 运行示例

### 前提条件

```bash
# 设置 API Key
export ANTHROPIC_API_KEY=your_api_key_here
```

### 编译并运行

```bash
cd examples/agent-working-memory
go run main.go
```

## 示例流程

### 会话 1：初次见面

```
User: 你好，我是 Alice，我是一名软件工程师。我喜欢简洁的回答和 TypeScript。

Agent: [内部使用 update_working_memory 工具]
{
  "user_name": "Alice",
  "preferences": {
    "language": "TypeScript",
    "verbosity": "concise"
  },
  "current_task": null
}

Agent: 你好 Alice！很高兴认识你。我已经记住了你的偏好：
- 偏好语言：TypeScript
- 回答风格：简洁

有什么我可以帮你的吗？
```

### 会话 2：开始任务

```
User: 帮我创建一个新的 REST API 项目。

Agent: [读取 Working Memory，发现用户偏好 TypeScript]
[更新 Working Memory 记录任务]
{
  "user_name": "Alice",
  "preferences": {
    "language": "TypeScript",
    "verbosity": "concise"
  },
  "current_task": {
    "name": "Create REST API project",
    "status": "in_progress",
    "progress": 0
  }
}

Agent: 好的，我会为你创建一个 TypeScript REST API 项目。

步骤：
1. 初始化 TypeScript 项目
2. 安装 Express + TypeScript
3. 配置 tsconfig.json
4. 创建基础路由结构

开始执行...
```

### 会话 3：继续任务（新对话）

```
User: 任务进行得怎么样了？

Agent: [自动加载 Working Memory，发现有进行中的任务]
{
  "user_name": "Alice",
  "current_task": {
    "name": "Create REST API project",
    "status": "in_progress",
    "progress": 60
  }
}

Agent: Alice，你的 REST API 项目目前进度 60%。

已完成：
- ✅ TypeScript 项目初始化
- ✅ Express 安装和配置
- ✅ tsconfig.json 配置

进行中：
- 🔄 基础路由结构

需要我继续完成吗？
```

## 关键特性说明

### 1. 自动加载

Working Memory 在**每轮对话开始时自动加载**：

```go
// Working Memory 内容会自动注入到 system prompt
<working_memory>
{
  "user_name": "Alice",
  "preferences": {...},
  "current_task": {...}
}
</working_memory>

// Agent 的实际 system prompt
You are a helpful task assistant...
```

### 2. LLM 主动更新

Agent 可以通过工具主动更新：

```json
// Agent 调用 update_working_memory 工具
{
  "memory": "{\"user_name\": \"Alice\", \"preferences\": {...}}"
}
```

**重要**：`update_working_memory` 是**完全覆盖**模式，必须包含所有要保留的信息！

### 3. Thread vs Resource Scope

#### Thread Scope（推荐用于大多数场景）

```yaml
working_memory:
  scope: "thread"
```

- 每个会话（thread）有独立的 Working Memory
- 适用于：独立的用户对话、不同上下文的任务

#### Resource Scope

```yaml
working_memory:
  scope: "resource"
```

- 同一资源的所有会话共享 Working Memory
- 适用于：多人协作编辑、团队共享项目

### 4. Schema 验证

配置 Schema 后，Working Memory 必须符合规范：

```yaml
working_memory:
  schema:
    type: object
    properties:
      user_name: { type: string }
      task_status:
        type: string
        enum: ["pending", "in_progress", "completed"]
    required: ["user_name"]
```

```go
// ✅ 有效更新
manager.Update(ctx, threadID, resourceID, `{
  "user_name": "Alice",
  "task_status": "in_progress"
}`)

// ❌ 无效更新（缺少 required 字段）
manager.Update(ctx, threadID, resourceID, `{
  "task_status": "in_progress"
}`)  // 返回错误
```

## 最佳实践

### ✅ 推荐

1. **保持简洁**
   - Working Memory < 500 words
   - 只存储当前会话相关的状态

2. **使用清晰的结构**

   ```json
   {
     "user_info": {...},
     "current_task": {...},
     "preferences": {...}
   }
   ```

3. **定义 Schema**
   - 确保数据一致性
   - 便于调试和维护

4. **正确使用作用域**
   - 默认使用 thread scope
   - 仅在需要跨会话共享时使用 resource scope

### ❌ 避免

1. **存储大量历史**
   - Working Memory 不是日志系统
   - 使用文本记忆存储历史记录

2. **忘记覆盖特性**
   - `update_working_memory` 会替换整个内容
   - 必须包含所有要保留的信息

3. **混淆两种记忆**
   - Working Memory：当前状态
   - 文本记忆：历史知识库

4. **频繁更新**
   - 仅在状态真正改变时更新
   - 避免每轮对话都更新

## 与文本记忆的配合使用

```
Working Memory（当前状态）:
{
  "user_name": "Alice",
  "current_task": "Create REST API",
  "task_status": "in_progress"
}

文本记忆（历史记录）:
/memories/users/alice/
  ├── profile.md         ← 用户档案
  ├── preferences.md     ← 详细偏好
  └── projects/
      └── rest-api.md    ← 项目详细记录
```

**策略：**

- **Working Memory**：存储当前会话需要的最小状态
- **文本记忆**：存储详细的历史记录和知识
- **定期归档**：将 Working Memory 中完成的任务归档到文本记忆

## 故障排除

### Working Memory 未加载

**症状**：Agent 没有记住之前的信息

**解决**：

1. 检查 `agentsdk.yaml` 中 `working_memory.enabled` 是否为 `true`
2. 确认 Middlewares 包含 `"working_memory"`
3. 检查 threadID/resourceID 是否正确传递

### Schema 验证失败

**症状**：更新 Working Memory 时报错

**解决**：

1. 确保内容是有效的 JSON
2. 检查是否包含所有 `required` 字段
3. 验证字段类型是否匹配

### 内容丢失

**症状**：之前的信息没有了

**原因**：`update_working_memory` 是完全覆盖模式

**解决**：

```json
// ❌ 错误：只更新部分，其他内容会丢失
{"user_name": "Alice"}

// ✅ 正确：包含所有要保留的信息
{
  "user_name": "Alice",
  "preferences": {...},
  "current_task": {...}
}
```

## 相关文档

- [Memory 系统完整指南](../../docs/content/4.guides/memory.md)
- [Working Memory API Reference](../../docs/content/6.api-reference/memory-api.md)
- [Working Memory 基础示例](../memory-working/)

## 总结

Working Memory 让 Agent 拥有了真正的"记忆"能力：

- ✅ **自动加载** - 无需 LLM 主动读取
- ✅ **LLM 控制** - Agent 可以主动更新
- ✅ **持久化** - 跨会话保持状态
- ✅ **结构化** - Schema 验证确保一致性

通过合理使用 Working Memory，Agent 可以提供更加个性化、连贯的体验！
