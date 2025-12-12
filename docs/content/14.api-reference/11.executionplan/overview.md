# ExecutionPlan API 参考

执行计划 (ExecutionPlan) 包提供了管理多步骤任务执行的完整功能。

## 包导入

```go
import "github.com/astercloud/aster/pkg/executionplan"
```

## 核心类型

### ExecutionPlan

执行计划主结构体。

```go
type ExecutionPlan struct {
    // 基础信息
    ID          string
    TaskID      string
    Name        string
    Description string

    // 步骤列表
    Steps []Step

    // 审批状态
    UserApproved  bool
    ApprovedAt    *time.Time
    ApprovedBy    string
    RejectionNote string

    // 执行状态
    Status          Status
    CurrentStep     int
    CreatedAt       time.Time
    UpdatedAt       time.Time
    StartedAt       *time.Time
    CompletedAt     *time.Time
    TotalDurationMs int64

    // 上下文信息
    AgentID  string
    OrgID    string
    TenantID string
    Metadata map[string]any

    // 执行选项
    Options *ExecutionOptions
}
```

#### 方法

##### NewExecutionPlan

创建新的执行计划。

```go
func NewExecutionPlan(description string) *ExecutionPlan
```

**参数：**
- `description` - 计划描述

**返回：**
- `*ExecutionPlan` - 新创建的执行计划

**示例：**
```go
plan := executionplan.NewExecutionPlan("部署应用到生产环境")
```

##### AddStep

添加执行步骤。

```go
func (p *ExecutionPlan) AddStep(toolName, description string, params map[string]any) *Step
```

**参数：**
- `toolName` - 工具名称
- `description` - 步骤描述
- `params` - 工具参数

**返回：**
- `*Step` - 添加的步骤

**示例：**
```go
step := plan.AddStep("bash", "运行测试", map[string]any{
    "command": "npm test",
})
```

##### GetStep

获取指定索引的步骤。

```go
func (p *ExecutionPlan) GetStep(index int) *Step
```

**参数：**
- `index` - 步骤索引（从 0 开始）

**返回：**
- `*Step` - 步骤，如果索引无效返回 nil

##### GetCurrentStep

获取当前执行的步骤。

```go
func (p *ExecutionPlan) GetCurrentStep() *Step
```

**返回：**
- `*Step` - 当前步骤

##### IsCompleted

检查计划是否已完成。

```go
func (p *ExecutionPlan) IsCompleted() bool
```

**返回：**
- `bool` - 如果状态为 completed、failed 或 cancelled 返回 true

##### IsApproved

检查计划是否已审批。

```go
func (p *ExecutionPlan) IsApproved() bool
```

**返回：**
- `bool` - 如果已用户审批或设置了自动审批返回 true

##### CanExecute

检查计划是否可以执行。

```go
func (p *ExecutionPlan) CanExecute() bool
```

**返回：**
- `bool` - 如果计划可以执行返回 true

##### Approve

审批计划。

```go
func (p *ExecutionPlan) Approve(approvedBy string)
```

**参数：**
- `approvedBy` - 审批人标识

**示例：**
```go
plan.Approve("user@example.com")
```

##### Reject

拒绝计划。

```go
func (p *ExecutionPlan) Reject(note string)
```

**参数：**
- `note` - 拒绝原因

**示例：**
```go
plan.Reject("不符合安全要求")
```

##### MarkStepStarted

标记步骤开始执行。

```go
func (p *ExecutionPlan) MarkStepStarted(index int)
```

**参数：**
- `index` - 步骤索引

##### MarkStepCompleted

标记步骤完成。

```go
func (p *ExecutionPlan) MarkStepCompleted(index int, result any)
```

**参数：**
- `index` - 步骤索引
- `result` - 执行结果

##### MarkStepFailed

标记步骤失败。

```go
func (p *ExecutionPlan) MarkStepFailed(index int, err error)
```

**参数：**
- `index` - 步骤索引
- `err` - 错误信息

##### Summary

返回计划摘要。

```go
func (p *ExecutionPlan) Summary() PlanSummary
```

**返回：**
- `PlanSummary` - 计划摘要信息

### Step

执行步骤。

```go
type Step struct {
    // 基础信息
    ID          string
    Index       int
    ToolName    string
    Description string

    // 参数
    Input      string
    Parameters map[string]any

    // 执行状态
    Status      StepStatus
    Result      any
    Error       string
    StartedAt   *time.Time
    CompletedAt *time.Time
    DurationMs  int64

    // 依赖关系
    DependsOn []string

    // 重试信息
    RetryCount   int
    MaxRetries   int
    RetryDelayMs int
}
```

### ExecutionOptions

执行选项。

```go
type ExecutionOptions struct {
    // 并行执行
    AllowParallel    bool
    MaxParallelSteps int

    // 错误处理
    StopOnError     bool
    ContinueOnError bool

    // 超时控制
    StepTimeoutMs  int64
    TotalTimeoutMs int64

    // 审批要求
    RequireApproval   bool
    AutoApprove       bool
    ApprovalTimeoutMs int64
}
```

### Status

计划状态枚举。

```go
type Status string

const (
    StatusDraft           Status = "draft"
    StatusPendingApproval Status = "pending_approval"
    StatusApproved        Status = "approved"
    StatusExecuting       Status = "executing"
    StatusCompleted       Status = "completed"
    StatusFailed          Status = "failed"
    StatusCancelled       Status = "cancelled"
    StatusPartial         Status = "partial"
)
```

### StepStatus

步骤状态枚举。

```go
type StepStatus string

const (
    StepStatusPending   StepStatus = "pending"
    StepStatusRunning   StepStatus = "running"
    StepStatusCompleted StepStatus = "completed"
    StepStatusFailed    StepStatus = "failed"
    StepStatusSkipped   StepStatus = "skipped"
)
```

### PlanSummary

计划摘要。

```go
type PlanSummary struct {
    ID          string
    Description string
    Status      Status
    TotalSteps  int
    Completed   int
    Failed      int
    Pending     int
    Running     int
    Progress    float64 // 完成百分比
}
```

## Generator

计划生成器，使用 LLM 生成执行计划。

### NewGenerator

创建计划生成器。

```go
func NewGenerator(prov provider.Provider, toolMap map[string]tools.Tool, opts ...GeneratorOption) *Generator
```

**参数：**
- `prov` - Provider 实例
- `toolMap` - 可用工具映射
- `opts` - 生成器选项

**返回：**
- `*Generator` - 生成器实例

**示例：**
```go
generator := executionplan.NewGenerator(provider, toolMap)
```

### Generate

生成执行计划。

```go
func (g *Generator) Generate(ctx context.Context, req *PlanRequest) (*ExecutionPlan, error)
```

**参数：**
- `ctx` - 上下文
- `req` - 计划请求

**返回：**
- `*ExecutionPlan` - 生成的计划
- `error` - 错误信息

**示例：**
```go
plan, err := generator.Generate(ctx, &executionplan.PlanRequest{
    Task:        "部署应用",
    Context:     "当前在开发分支",
    Constraints: []string{"必须先运行测试"},
})
```

### ValidatePlan

验证计划的有效性。

```go
func (g *Generator) ValidatePlan(plan *ExecutionPlan) []error
```

**参数：**
- `plan` - 要验证的计划

**返回：**
- `[]error` - 验证错误列表

### PlanRequest

计划生成请求。

```go
type PlanRequest struct {
    Task        string   // 任务描述
    Context     string   // 上下文信息
    Constraints []string // 约束条件
    MaxSteps    int      // 最大步骤数
}
```

## Executor

计划执行器。

### NewExecutor

创建执行器。

```go
func NewExecutor(toolMap map[string]tools.Tool, opts ...ExecutorOption) *Executor
```

**参数：**
- `toolMap` - 工具映射
- `opts` - 执行器选项

**返回：**
- `*Executor` - 执行器实例

**示例：**
```go
executor := executionplan.NewExecutor(toolMap)
```

### Execute

执行计划。

```go
func (e *Executor) Execute(ctx context.Context, plan *ExecutionPlan, toolCtx *tools.ToolContext) error
```

**参数：**
- `ctx` - 上下文
- `plan` - 执行计划
- `toolCtx` - 工具上下文

**返回：**
- `error` - 执行错误

**示例：**
```go
err := executor.Execute(ctx, plan, &tools.ToolContext{
    AgentID: "agent-123",
})
```

### Resume

恢复执行计划。

```go
func (e *Executor) Resume(ctx context.Context, plan *ExecutionPlan, toolCtx *tools.ToolContext) error
```

**参数：**
- `ctx` - 上下文
- `plan` - 执行计划
- `toolCtx` - 工具上下文

**返回：**
- `error` - 执行错误

**示例：**
```go
// 从失败点恢复
err := executor.Resume(ctx, plan, toolCtx)
```

### Cancel

取消执行。

```go
func (e *Executor) Cancel(plan *ExecutionPlan, reason string)
```

**参数：**
- `plan` - 执行计划
- `reason` - 取消原因

**示例：**
```go
executor.Cancel(plan, "用户取消")
```

## 执行器选项

### WithOnStepStart

设置步骤开始回调。

```go
func WithOnStepStart(fn func(*ExecutionPlan, *Step)) ExecutorOption
```

**示例：**
```go
executor := executionplan.NewExecutor(
    toolMap,
    executionplan.WithOnStepStart(func(plan *executionplan.ExecutionPlan, step *executionplan.Step) {
        log.Printf("开始: %s", step.Description)
    }),
)
```

### WithOnStepComplete

设置步骤完成回调。

```go
func WithOnStepComplete(fn func(*ExecutionPlan, *Step)) ExecutorOption
```

### WithOnStepFailed

设置步骤失败回调。

```go
func WithOnStepFailed(fn func(*ExecutionPlan, *Step, error)) ExecutorOption
```

### WithOnPlanComplete

设置计划完成回调。

```go
func WithOnPlanComplete(fn func(*ExecutionPlan)) ExecutorOption
```

## 完整示例

### 基本使用

```go
package main

import (
    "context"
    "log"

    "github.com/astercloud/aster/pkg/executionplan"
    "github.com/astercloud/aster/pkg/tools"
)

func main() {
    // 创建计划
    plan := executionplan.NewExecutionPlan("CI/CD 流程")

    // 添加步骤
    step1 := plan.AddStep("git_checkout", "检出代码", map[string]any{
        "branch": "main",
    })

    step2 := plan.AddStep("npm_install", "安装依赖", nil)
    step2.DependsOn = []string{step1.ID}

    step3 := plan.AddStep("npm_test", "运行测试", nil)
    step3.DependsOn = []string{step2.ID}

    // 配置选项
    plan.Options.RequireApproval = true
    plan.Options.AllowParallel = true

    // 审批
    plan.Approve("user@example.com")

    // 创建执行器
    executor := executionplan.NewExecutor(
        toolMap,
        executionplan.WithOnStepComplete(func(p *executionplan.ExecutionPlan, s *executionplan.Step) {
            log.Printf("完成: %s", s.Description)
        }),
    )

    // 执行
    ctx := context.Background()
    toolCtx := &tools.ToolContext{AgentID: "agent-123"}

    err := executor.Execute(ctx, plan, toolCtx)
    if err != nil {
        log.Fatalf("执行失败: %v", err)
    }

    // 查看结果
    summary := plan.Summary()
    log.Printf("完成: %d/%d 步骤", summary.Completed, summary.TotalSteps)
}
```

### 使用 LLM 生成计划

```go
// 创建生成器
generator := executionplan.NewGenerator(provider, toolMap)

// 生成计划
plan, err := generator.Generate(ctx, &executionplan.PlanRequest{
    Task: "部署应用到生产环境",
    Context: `
        当前状态：
        - 代码在 develop 分支
        - 测试已通过
        - 需要合并到 main 分支
    `,
    Constraints: []string{
        "必须先备份数据库",
        "必须在非高峰时段部署",
        "部署后需要验证",
    },
    MaxSteps: 10,
})
if err != nil {
    return err
}

// 验证计划
errors := generator.ValidatePlan(plan)
if len(errors) > 0 {
    for _, err := range errors {
        log.Printf("验证错误: %v", err)
    }
    return fmt.Errorf("计划验证失败")
}

// 显示计划给用户审批
fmt.Println("执行计划:")
for i, step := range plan.Steps {
    fmt.Printf("%d. %s - %s\n", i+1, step.ToolName, step.Description)
}

// 执行
if userApproves() {
    plan.Approve("user@example.com")
    executor.Execute(ctx, plan, toolCtx)
}
```

## 相关文档

- [执行计划概念](../../02.core-concepts/16.execution-plan.md)
- [工具系统](../../02.core-concepts/4.tools-system.md)
- [Agent API](../1.agent/overview.md)
