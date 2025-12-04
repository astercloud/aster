# Human-in-the-Loop (HITL) 架构

Human-in-the-Loop (HITL) 是 aster 的核心安全特性，允许在 Agent 执行敏感操作前进行人工审核和控制。

## 架构设计

### 控制流

```
Agent → Middleware Stack → HITL Middleware → Review Request
                                  ↓
                         Approval Handler
                                  ↓
                         Human Decision
                                  ↓
            Approve / Reject / Edit Parameters
                                  ↓
                         Continue Execution
```

### 核心组件

#### 1. HITL Middleware

位于 Middleware Stack 中，拦截工具调用并触发审核流程。

```go
type HumanInTheLoopMiddleware struct {
    InterruptOn      map[string]interface{}  // 审核规则
    ApprovalHandler  ApprovalHandler         // 审核处理器
    RiskAssessor     RiskAssessor            // 风险评估器
}
```

#### 2. Review Request

包含待审核操作的详细信息。

```go
type ReviewRequest struct {
    ActionRequests []ActionRequest  // 待审核的操作
    Context        Context          // 执行上下文
    RiskLevel      RiskLevel        // 风险等级
}
```

#### 3. Decision Types

三种决策类型：

```go
type DecisionType string

const (
    DecisionApprove DecisionType = "approve"  // 批准
    DecisionReject  DecisionType = "reject"   // 拒绝
    DecisionEdit    DecisionType = "edit"     // 编辑参数
)
```

#### 4. Approval Handler

处理人工审核的接口。

```go
type ApprovalHandler func(
    ctx context.Context,
    req *ReviewRequest,
) ([]Decision, error)
```

## 典型使用场景

### 1. 敏感操作控制

```go
hitlMW, _ := middleware.NewHumanInTheLoopMiddleware(&middleware.HumanInTheLoopMiddlewareConfig{
    InterruptOn: map[string]interface{}{
        "Bash":        true,  // Shell 命令
        "fs_delete":   true,  // 文件删除
        "HttpRequest": true,  // HTTP 请求
        "database_update": true,  // 数据库更新
    },
    ApprovalHandler: cliApprovalHandler,
})
```

### 2. 基于风险评估的审核

```go
hitlMW, _ := middleware.NewHumanInTheLoopMiddleware(&middleware.HumanInTheLoopMiddlewareConfig{
    RiskAssessor: func(ctx context.Context, action *ActionRequest) (RiskLevel, error) {
        // 评估操作风险
        if action.ToolName == "Bash" {
            cmd := action.Input["command"].(string)
            if strings.Contains(cmd, "rm -rf") {
                return RiskCritical, nil  // 高风险
            }
        }
        return RiskLow, nil
    },
    ApprovalHandler: riskBasedHandler,
})
```

### 3. 批量审核

```go
ApprovalHandler: func(ctx context.Context, req *ReviewRequest) ([]Decision, error) {
    // 一次性审核所有操作
    decisions := make([]Decision, len(req.ActionRequests))

    for i, action := range req.ActionRequests {
        // 展示操作详情
        fmt.Printf("%d. %s: %+v\n", i+1, action.ToolName, action.Input)
    }

    // 批量决策
    fmt.Print("批准所有? (y/n): ")
    var answer string
    fmt.Scanln(&answer)

    for i := range decisions {
        if answer == "y" {
            decisions[i] = Decision{Type: DecisionApprove}
        } else {
            decisions[i] = Decision{Type: DecisionReject}
        }
    }

    return decisions, nil
}
```

## 审核处理器实现

### 1. CLI 审核

```go
func cliApprovalHandler(ctx context.Context, req *ReviewRequest) ([]Decision, error) {
    for _, action := range req.ActionRequests {
        fmt.Printf("工具: %s\n", action.ToolName)
        fmt.Printf("参数: %+v\n", action.Input)
        fmt.Printf("风险: %s\n", req.RiskLevel)
        fmt.Print("决策 (approve/reject/edit): ")

        var decision string
        fmt.Scanln(&decision)

        switch decision {
        case "approve":
            return []Decision{{Type: DecisionApprove}}, nil
        case "reject":
            return []Decision{{Type: DecisionReject}}, nil
        case "edit":
            // 编辑参数
            newInput := editParameters(action.Input)
            return []Decision{{
                Type: DecisionEdit,
                ModifiedInput: newInput,
            }}, nil
        }
    }
    return nil, fmt.Errorf("invalid decision")
}
```

### 2. Web UI 审核

```go
func webUIApprovalHandler(ctx context.Context, req *ReviewRequest) ([]Decision, error) {
    // 1. 将审核请求存储到数据库
    reviewID := saveReviewRequest(req)

    // 2. 通过 WebSocket 通知前端
    notifyWebUI(reviewID, req)

    // 3. 等待用户决策
    decision := waitForDecision(ctx, reviewID)

    return []Decision{decision}, nil
}
```

### 3. 消息队列审核

```go
func mqApprovalHandler(ctx context.Context, req *ReviewRequest) ([]Decision, error) {
    // 1. 发送到审核队列
    reviewID := uuid.New().String()
    publishToQueue("review-requests", ReviewMessage{
        ID:      reviewID,
        Request: req,
    })

    // 2. 订阅决策队列
    decisionChan := subscribeToQueue(fmt.Sprintf("decisions-%s", reviewID))

    // 3. 等待决策（支持超时）
    select {
    case decision := <-decisionChan:
        return []Decision{decision}, nil
    case <-time.After(5 * time.Minute):
        return nil, fmt.Errorf("approval timeout")
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}
```

## 与其他组件集成

### 1. 与 Telemetry 集成

```go
// 记录审核事件
hitlMW, _ := middleware.NewHumanInTheLoopMiddleware(&middleware.HumanInTheLoopMiddlewareConfig{
    ApprovalHandler: func(ctx context.Context, req *ReviewRequest) ([]Decision, error) {
        decision, err := getHumanDecision(req)

        // 记录审核日志
        telemetry.RecordEvent(ctx, "hitl.review", map[string]interface{}{
            "tool":     req.ActionRequests[0].ToolName,
            "decision": decision.Type,
            "risk":     req.RiskLevel,
        })

        return []Decision{decision}, err
    },
})
```

### 2. 与 Memory 集成

```go
// 记住审核决策，用于学习
ApprovalHandler: func(ctx context.Context, req *ReviewRequest) ([]Decision, error) {
    decision, err := getHumanDecision(req)

    // 存储审核历史
    memory.Store(ctx, "approval_history", ApprovalRecord{
        Timestamp:  time.Now(),
        ToolName:   req.ActionRequests[0].ToolName,
        Input:      req.ActionRequests[0].Input,
        Decision:   decision.Type,
        RiskLevel:  req.RiskLevel,
    })

    return []Decision{decision}, err
}
```

### 3. 与 Workflow 集成

```go
// 在 Workflow 中使用 HITL
workflow := workflow.NewWorkflow("sensitive-operations")

workflow.AddStep(&workflow.Step{
    Name: "data-extraction",
    Agent: agentWithHITL,  // 使用配置了 HITL 的 Agent
})

workflow.AddStep(&workflow.Step{
    Name: "data-processing",
    Agent: normalAgent,  // 无需审核
})
```

## 高级特性

### 1. 智能风险评估

```go
RiskAssessor: func(ctx context.Context, action *ActionRequest) (RiskLevel, error) {
    score := 0

    // 基于工具评分
    if action.ToolName == "Bash" {
        score += 50
    }

    // 基于参数评分
    if cmd, ok := action.Input["command"].(string); ok {
        if strings.Contains(cmd, "rm") {
            score += 30
        }
        if strings.Contains(cmd, "sudo") {
            score += 20
        }
    }

    // 评估风险等级
    switch {
    case score >= 80:
        return RiskCritical, nil
    case score >= 50:
        return RiskHigh, nil
    case score >= 20:
        return RiskMedium, nil
    default:
        return RiskLow, nil
    }
}
```

### 2. 条件审核

```go
InterruptOn: map[string]interface{}{
    "Bash": &middleware.InterruptConfig{
        Enabled: true,
        Condition: func(input map[string]interface{}) bool {
            // 只审核包含 "rm" 的命令
            cmd, ok := input["command"].(string)
            return ok && strings.Contains(cmd, "rm")
        },
    },
}
```

### 3. 审核超时

```go
hitlMW, _ := middleware.NewHumanInTheLoopMiddleware(&middleware.HumanInTheLoopMiddlewareConfig{
    Timeout: 5 * time.Minute,  // 5 分钟超时
    OnTimeout: func(req *ReviewRequest) Decision {
        // 超时默认拒绝
        return Decision{Type: DecisionReject}
    },
    ApprovalHandler: approvalHandler,
})
```

## 安全最佳实践

### 1. 最小权限原则

只对真正敏感的操作启用审核：

```go
InterruptOn: map[string]interface{}{
    // ✅ 需要审核
    "Bash":         true,
    "fs_delete":    true,
    "database_update": true,

    // ❌ 不需要审核
    "fs_read":      false,
    "http_get":     false,
}
```

### 2. 风险分级

根据风险等级采取不同措施：

```go
ApprovalHandler: func(ctx context.Context, req *ReviewRequest) ([]Decision, error) {
    switch req.RiskLevel {
    case RiskCritical:
        // 高风险：需要多人审批
        return multiPersonApproval(req)
    case RiskHigh:
        // 中高风险：需要详细审核
        return detailedReview(req)
    case RiskMedium:
        // 中等风险：快速审核
        return quickReview(req)
    default:
        // 低风险：自动批准
        return []Decision{{Type: DecisionApprove}}, nil
    }
}
```

### 3. 审核日志

记录所有审核决策：

```go
ApprovalHandler: func(ctx context.Context, req *ReviewRequest) ([]Decision, error) {
    decision, err := getDecision(req)

    // 记录审核日志
    auditLog.Record(AuditEntry{
        Timestamp:  time.Now(),
        UserID:     getUserID(ctx),
        Action:     req.ActionRequests[0].ToolName,
        Parameters: req.ActionRequests[0].Input,
        Decision:   decision.Type,
        Reason:     decision.Reason,
    })

    return []Decision{decision}, err
}
```

## 性能考虑

### 1. 异步审核

对于非阻塞场景，使用异步审核：

```go
AsyncApproval: true,  // 启用异步审核
OnPending: func(req *ReviewRequest) {
    // 操作进入待审核队列
    fmt.Println("操作已提交审核...")
}
```

### 2. 批量优化

合并多个待审核操作：

```go
BatchMode: true,  // 启用批量模式
BatchTimeout: 30 * time.Second,  // 等待 30 秒收集更多操作
```

### 3. 缓存决策

对相同操作缓存审核决策：

```go
CacheDecisions: true,
CacheTTL: 1 * time.Hour,
```

## 相关文档

- [HITL Middleware 详细文档](../06.middleware/2.builtin/human-in-the-loop.md)
- [HITL 高级指南](../13.guides/2.advanced/human-in-the-loop.md)
- [安全最佳实践](../15.best-practices/security.md)
- [Middleware 系统](../06.middleware/)
