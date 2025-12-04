# Human-in-the-Loop (HITL) 示例

本示例演示如何使用 HITL 中间件实现人工审核和控制敏感 Agent 操作。

## 功能特性

- ✅ 基于风险级别的智能审核策略
- ✅ 支持批准、拒绝、编辑三种决策类型
- ✅ 自动检测危险命令和路径
- ✅ 交互式命令行审核界面
- ✅ 完整的审核日志记录

## 前置要求

- Go 1.21+
- OpenAI API Key

## 运行示例

1. 设置环境变量：

```bash
export OPENAI_API_KEY=your_api_key_here
```

2. 运行程序：

```bash
cd examples/human-in-the-loop
go run main.go
```

## 演示场景

本示例包含三个演示场景：

### 场景 1: 低风险操作

用户请求：`请列出当前目录的文件`

- Agent 调用 `Bash("ls")`
- HITL 评估为低风险
- 自动批准执行

### 场景 2: 中风险操作

用户请求：`请删除 /tmp/test.txt 文件`

- Agent 调用 `fs_delete("/tmp/test.txt")`
- HITL 评估为中风险
- 需要人工确认

### 场景 3: 高风险操作

用户请求：`请执行 rm -rf /tmp/* 命令`

- Agent 调用 `Bash("rm -rf /tmp/*")`
- HITL 评估为高风险
- 需要输入 'CONFIRM' 明确确认

## 风险评估规则

### 低风险 (🟢)

- 读取操作：`ls`, `cat`, `grep`
- 查询命令：`ps`, `top`, `df`
- 无副作用的操作

### 中风险 (🟡)

- 文件操作：`rm`, `mv`, `chmod`
- 进程控制：`kill`, `pkill`
- 配置文件修改

### 高风险 (🔴)

- 批量删除：`rm -rf`
- 磁盘操作：`mkfs`, `dd`
- 系统路径操作：`/etc`, `/usr`, `/bin`

## 审核流程

```
用户请求
    ↓
Agent 决策调用工具
    ↓
HITL 中间件拦截
    ↓
风险评估
    ↓
根据风险级别决定策略:
├─ 低风险 → 自动批准
├─ 中风险 → 人工确认
└─ 高风险 → 明确确认 (输入 CONFIRM)
    ↓
执行或拒绝
    ↓
返回结果给 Agent
```

## 配置说明

### InterruptOn 配置

指定哪些工具需要审核：

```go
InterruptOn: map[string]interface{}{
    "Bash": map[string]interface{}{
        "message": "⚠️  Shell 命令执行需要审核",
        "allowed_decisions": []string{"approve", "reject", "edit"},
    },
}
```

### ApprovalHandler

实现自定义审核逻辑：

```go
ApprovalHandler: func(ctx context.Context, req *middleware.ReviewRequest) ([]middleware.Decision, error) {
    // 1. 评估风险
    risk := assessRisk(req.ActionRequests[0])

    // 2. 根据风险决定策略
    switch risk {
    case RiskLow:
        return autoApprove()
    case RiskMedium:
        return promptForDecision()
    case RiskHigh:
        return promptForHighRiskDecision()
    }
}
```

## 决策类型

### Approve (批准)

```go
middleware.Decision{
    Type:   middleware.DecisionApprove,
    Reason: "用户批准执行",
}
```

### Reject (拒绝)

```go
middleware.Decision{
    Type:   middleware.DecisionReject,
    Reason: "用户拒绝",
}
```

### Edit (编辑)

```go
middleware.Decision{
    Type:        middleware.DecisionEdit,
    EditedInput: map[string]interface{}{
        "command": "ls -la",  // 修改后的参数
    },
    Reason: "用户编辑参数后执行",
}
```

## 扩展示例

### 1. Web UI 审核

集成 Web 界面进行审核：

```go
ApprovalHandler: webApprovalSystem.CreateHandler()
```

### 2. 基于角色的审核

不同角色有不同的审核权限：

```go
func roleBasedApprovalHandler(ctx context.Context, req *middleware.ReviewRequest) ([]middleware.Decision, error) {
    user := getUserFromContext(ctx)

    if hasPermission(user, req.ActionRequests[0].ToolName) {
        return autoApprove()
    }

    return requestSupervisorApproval(req)
}
```

### 3. 审核日志记录

记录所有审核决策：

```go
type AuditLog struct {
    Timestamp time.Time
    Tool      string
    Decision  string
    Approver  string
}

func logApproval(action middleware.ActionRequest, decision middleware.Decision) {
    // 保存到数据库或文件
}
```

## 最佳实践

1. **只审核真正敏感的操作** - 避免审核低风险操作
2. **提供清晰的审核信息** - 帮助审核员快速决策
3. **实现超时机制** - 避免无限等待
4. **记录审核日志** - 用于审计和分析
5. **支持批量审核** - 提高审核效率

## 安全建议

1. **默认拒绝策略** - 无法获取决策时默认拒绝
2. **审核权限控制** - 基于角色控制审核权限
3. **审核记录不可篡改** - 使用只追加日志
4. **定期审计** - 检查审核日志发现异常

## 相关文档

- [HITL 完整指南](/guides/advanced/human-in-the-loop)
- [中间件文档](/middleware/builtin)
- [安全最佳实践](/best-practices/security)

## 故障排除

### 问题：审核请求超时

**解决方案**：实现超时机制

```go
ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
defer cancel()
```

### 问题：Agent 重复尝试被拒绝的操作

**解决方案**：在 System Prompt 中明确说明不要重复尝试

```go
systemPrompt += "\n如果操作被拒绝，不要重复尝试。"
```

## License

MIT
