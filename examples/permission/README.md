# Permission System 示例

本示例演示 Aster 权限系统的使用，支持三种审批模式和灵活的规则配置。

## 功能特点

- 🔐 三种审批模式：auto_approve、smart_approve、always_ask
- ⚡ 基于风险级别的智能决策
- 📝 灵活的规则系统
- 💾 规则持久化
- 🔗 与 Control Channel 集成

## 运行示例

```bash
go run ./examples/permission/
```

## 三种模式

### 1. Auto Approve (自动审批)

自动批准所有工具执行，适合开发和测试环境。

```go
inspector, _ := permission.NewInspector(
    permission.WithMode(permission.ModeAutoApprove),
)

// 所有工具都会自动批准
result, _ := inspector.Check(ctx, &permission.Request{
    ToolName: "Bash",
    Arguments: map[string]any{"command": "rm -rf /"},
})
// result.Decision == DecisionAllow
// result.NeedsApproval == false
```

### 2. Smart Approve (智能审批)

根据风险级别智能决策，是推荐的默认模式。

| 风险级别 | 工具示例 | 决策 |
|----------|----------|------|
| Low | Read, List, Search | 自动批准 |
| Medium | Write, Edit | 需要审批 |
| High | Bash, Delete, Http | 需要审批 |

```go
inspector, _ := permission.NewInspector(
    permission.WithMode(permission.ModeSmartApprove),
)

// 读操作自动批准
result, _ := inspector.Check(ctx, &permission.Request{
    ToolName: "Read",
    Arguments: map[string]any{"path": "main.go"},
})
// result.NeedsApproval == false

// 写操作需要审批
result, _ = inspector.Check(ctx, &permission.Request{
    ToolName: "Write",
    Arguments: map[string]any{"path": "main.go", "content": "..."},
})
// result.NeedsApproval == true
```

### 3. Always Ask (总是询问)

所有工具执行都需要用户确认，适合高安全性场景。

```go
inspector, _ := permission.NewInspector(
    permission.WithMode(permission.ModeAlwaysAsk),
)

// 所有工具都需要审批
result, _ := inspector.Check(ctx, &permission.Request{
    ToolName: "Read",
    Arguments: map[string]any{"path": "main.go"},
})
// result.NeedsApproval == true
```

## 规则系统

### 添加规则

```go
// 允许所有读取操作
inspector.AddRule(&permission.Rule{
    Pattern:   "Read",
    Decision:  permission.DecisionAllowAlways,
    RiskLevel: permission.RiskLevelLow,
    Note:      "允许所有读取操作",
})

// 禁止危险命令
inspector.AddRule(&permission.Rule{
    Pattern:   "Bash",
    Decision:  permission.DecisionDenyAlways,
    RiskLevel: permission.RiskLevelHigh,
    Conditions: []permission.Condition{
        {
            Field:    "command",
            Operator: "contains",
            Value:    "rm -rf",
        },
    },
    Note: "禁止危险的删除命令",
})

// 允许写入特定目录
inspector.AddRule(&permission.Rule{
    Pattern:   "Write",
    Decision:  permission.DecisionAllowAlways,
    Conditions: []permission.Condition{
        {
            Field:    "path",
            Operator: "prefix",
            Value:    "/tmp/",
        },
    },
})
```

### 条件运算符

| 运算符 | 说明 | 示例 |
|--------|------|------|
| `eq` | 相等 | `command eq "ls"` |
| `ne` | 不等 | `path ne "/etc/passwd"` |
| `contains` | 包含 | `command contains "rm"` |
| `prefix` | 前缀 | `path prefix "/home/"` |
| `suffix` | 后缀 | `path suffix ".txt"` |
| `regex` | 正则 | `command regex "^git\s+"` |

### 规则持久化

```go
// 保存规则到文件
err := inspector.SaveRules()

// 从文件加载规则
inspector, _ := permission.NewInspector(
    permission.WithPath("~/.config/aster/permissions.json"),
    permission.WithAutoLoad(true), // 自动加载
)
```

## 与 Agent 集成

### 使用中间件

```go
// 创建 HITL 中间件
hitlMiddleware := middleware.NewHumanInTheLoopMiddleware(&middleware.HumanInTheLoopMiddlewareConfig{
    Inspector: inspector,
    ApprovalHandler: func(ctx context.Context, req *middleware.ReviewRequest) ([]middleware.Decision, error) {
        // 自定义审批逻辑
        return []middleware.Decision{{Type: middleware.DecisionApprove}}, nil
    },
})

// 在 Agent 中使用
config := &types.AgentConfig{
    Middlewares: []string{"hitl"},
}
```

### Control Channel 集成

```go
// 订阅 Control Channel
controlCh := agent.Subscribe([]types.AgentChannel{types.ChannelControl}, nil)

go func() {
    for event := range controlCh {
        if permEvent, ok := event.Event.(*types.ControlPermissionRequiredEvent); ok {
            // 显示审批 UI
            showApprovalDialog(permEvent)
            
            // 发送审批决定
            agent.ApprovePermission(permEvent.RequestID, true, "用户批准")
        }
    }
}()
```

## 风险评估

系统内置了工具风险评估：

```go
// 获取工具风险级别
risk := inspector.AssessRisk(&permission.Request{
    ToolName:  "Bash",
    Arguments: map[string]any{"command": "curl http://..."},
})
// risk == RiskLevelHigh
```

### 内置风险规则

| 工具 | 默认风险 | 说明 |
|------|----------|------|
| Read, List, Search | Low | 只读操作 |
| Write, Edit | Medium | 文件修改 |
| Bash, Delete | High | 系统操作 |
| Http (外部) | High | 网络请求 |

## 相关示例

- [human-in-the-loop](../human-in-the-loop/) - HITL 完整示例
- [desktop](../desktop/) - 桌面应用集成
- [recipe](../recipe/) - Recipe 中的权限配置
