---
title: Skills 最佳实践
description: Skills 系统设计、组织和优化指南
navigation:
  icon: i-lucide-zap
---

# Skills 最佳实践

本文档提供 Skills 系统的设计模式、组织策略和性能优化建议。

## 🎯 核心原则

1. **单一职责** - 每个 Skill 专注一个领域或任务
2. **细粒度设计** - 小而精的 Skill 便于复用和维护
3. **元数据清晰** - `name` / `description` 直观描述“做什么 + 何时用”
4. **内容精炼** - 控制 Skill 大小，把真正需要的步骤放在 SKILL.md，而不是 System Prompt

## 📐 Skill 设计模式

### 1. 按领域划分

**推荐**：根据专业领域创建独立 Skill

```
✅ 好的设计
skills/
├── go-coding-standards.md      # Go 语言规范
├── python-best-practices.md    # Python 最佳实践
├── sql-optimization.md         # SQL 优化
└── api-design-guidelines.md    # API 设计

❌ 不好的设计
skills/
└── programming-everything.md   # 包含所有语言的所有内容
```

**优势**：

- 按需激活，减少无关内容注入
- 独立维护和更新
- 便于团队协作

### 2. 按工作流阶段划分

**推荐**：根据工作流程阶段组织 Skill

```
✅ 好的设计
skills/
├── code-review-checklist.md    # 审查清单
├── security-audit.md            # 安全审计
├── performance-profiling.md    # 性能分析
└── documentation-guide.md      # 文档编写

触发条件：
- code-review-checklist: keyword "review", context "during /review"
- security-audit: file_pattern "**/*.{go,js,py}"
- performance-profiling: keyword "性能", "优化"
```

**优势**：

- 自动适应工作流程
- 上下文相关性强
- 减少手动干预

### 3. 按角色划分

**推荐**：为不同角色创建专用 Skill

```
✅ 好的设计
skills/
├── developer/
│   ├── code-quality.md
│   └── testing-guide.md
├── reviewer/
│   ├── review-checklist.md
│   └── approval-criteria.md
└── architect/
    ├── design-patterns.md
    └── system-architecture.md
```

**使用方式**：

```go
// 开发者 Agent
&types.AgentConfig{
    SkillsPackageConfig: &types.SkillsPackageConfig{
        Path: "./skills/developer",
    },
}

// 审查者 Agent
&types.AgentConfig{
    SkillsPackageConfig: &types.SkillsPackageConfig{
        Path: "./skills/reviewer",
    },
}
```

## 🗂️ 组织策略

### 1. 标准目录结构

在 aster 中，推荐使用简单直接的目录结构，例如：

```
workspace/
└── skills/
    ├── core/                    # 核心规范
    │   ├── code-of-conduct/
    │   │   └── SKILL.md
    │   └── security-policy/
    │       └── SKILL.md
    ├── languages/               # 编程语言
    │   ├── go-coding-standards/
    │   │   └── SKILL.md
    │   └── python-best-practices/
    │       └── SKILL.md
    ├── workflows/               # 工作流
    │   ├── code-review/
    │   │   └── SKILL.md
    │   └── deployment/
    │       └── SKILL.md
    └── custom/                  # 项目/公司特定规范
        └── company-standards/
            └── SKILL.md
```

### 2. 命名规范

**文件命名**：

```bash
# 推荐：小写、连字符分隔、描述性
✅ go-error-handling/
✅ react-hooks-guide/
✅ sql-injection-防御/

# 不推荐：大写、下划线、缩写
❌ GO_Errors/
❌ rh/
❌ SQLInj/
```

**Skill Name 字段**：

```yaml
---
name: go-error-handling # 推荐：与目录名一致，小写 + 连字符
---
```

> 约束：当前实现中，`name` 必须满足：
>
> - 1–64 个字符
> - 只包含小写字母、数字和连字符（`-`）
> - 不能包含 `anthropic` 或 `claude`
> - 建议与技能目录名保持一致，便于排查问题

### 3. 版本管理

**方案1：Git 分支**

```bash
git checkout main           # 稳定版
git checkout develop        # 开发版
git checkout feature/new-skill  # 新功能
```

**方案2：版本目录**

```
skills/
├── v1/
│   ├── coding-standards.md
│   └── security-checklist.md
└── v2/
    ├── coding-standards.md  # 更新版本
    └── security-checklist.md
```

**配置切换**：

```go
// 生产环境使用 v1
&types.SkillsPackageConfig{
    Path: "oss://my-bucket/skills/v1",
}

// 测试环境使用 v2
&types.SkillsPackageConfig{
    Path: "oss://my-bucket/skills/v2",
}
```

## ⚡ 触发元数据策略（可选）

> 当前版本中，Skills 注入器不会根据 `triggers` 自动筛选技能。
> 如果你需要“按关键词/上下文/文件模式筛选技能”，推荐在自己的业务逻辑里解析 `triggers` 字段进行过滤。

### 1. 选择合适的触发类型（元数据语义）

| 触发类型       | 建议含义                                                    | 示例                 |
| -------------- | ----------------------------------------------------------- | -------------------- |
| `always`       | 表示此 Skill 是“常驻规范”，通常用于安全/合规类 Skill 的标记 | 代码规范、隐私政策   |
| `keyword`      | 用于标记“适用的用户意图或关键词”，方便你在上层做路由/展示   | "性能优化"、"测试"   |
| `context`      | 标记适用的工作流阶段或命令                                  | `during /review`     |
| `file_pattern` | 标记与特定文件类型强相关                                    | `**/*.go`、`**/*.md` |

### 2. 组合触发最佳实践（供自定义过滤使用）

**场景1：代码审查**

```yaml
---
name: code-review-checklist
triggers:
  # 用户主动请求（例如 UI 可以在用户输入这些词时优先展示此 Skill）
  - type: keyword
    keywords: ["review", "审查", "检查"]

  # 特定命令触发（例如 /review 工作流下优先启用）
  - type: context
    condition: "during /review"

  # 操作代码文件时触发（例如自动推荐此 Skill）
  - type: file_pattern
    pattern: "src/**/*.{go,js,ts,py}"
---
```

**场景2：安全审计**

```yaml
---
name: security-audit
triggers:
  # 核心安全规范
  - type: always

  # 安全相关关键词提示
  - type: keyword
    keywords: ["安全", "漏洞", "security"]
---
```

**场景3：性能优化**

```yaml
---
name: performance-optimization
triggers:
  # 用户明确请求性能优化相关内容
  - type: keyword
    keywords: ["性能", "优化", "performance", "slow"]

  # 特定文件类型（如配置文件）
  - type: file_pattern
    pattern: "**/{Dockerfile,docker-compose.yml}"
---
```

### 3. 避免触发冲突

**问题**：多个 Skill 使用相同关键词

```yaml
# Skill 1
triggers:
  - type: keyword
    keywords: ["测试"]

# Skill 2
triggers:
  - type: keyword
    keywords: ["测试"]
```

**解决方案**：使用更具体的关键词组合

```yaml
# 单元测试 Skill
triggers:
  - type: keyword
    keywords: ["单元测试", "unit test"]

# 集成测试 Skill
triggers:
  - type: keyword
    keywords: ["集成测试", "integration test"]

# 性能测试 Skill
triggers:
  - type: keyword
    keywords: ["性能测试", "benchmark"]
```

## 📝 内容编写

### 1. Skill 结构模板

```markdown
---
name: skill-name
description: 简短描述（1-2句话）
allowed-tools: ["Read", "Write", "Grep"]
triggers:
  - type: keyword
    keywords: ["关键词1", "关键词2"]
---

# Skill 名称

## 概述

简要说明此 Skill 的用途和价值

## 核心原则

- 原则1：解释
- 原则2：解释
- 原则3：解释

## 检查清单

- [ ] 检查项1
- [ ] 检查项2
- [ ] 检查项3

## 示例

### 正确示例

\`\`\`
好的代码示例
\`\`\`

### 错误示例

\`\`\`
不好的代码示例
\`\`\`

## 参考资料

- [文档链接](https://...)
- [最佳实践](https://...)
```

### 2. 内容长度控制

**推荐长度**：

| Skill 类型 | 建议 Token 数 | 大约行数 |
| ---------- | ------------- | -------- |
| 简单规范   | 500-1000      | 50-100   |
| 详细指南   | 1000-2000     | 100-200  |
| 完整教程   | 2000-5000     | 200-500  |

**检查 Token 数**：

```bash
# 使用 tiktoken 计算
pip install tiktoken

python -c "
import tiktoken
enc = tiktoken.get_encoding('cl100k_base')
with open('skill.md', 'r') as f:
    content = f.read()
print(f'Tokens: {len(enc.encode(content))}')
"
```

### 3. Markdown 最佳实践

**推荐**：

```markdown
# 使用清晰的标题层级

## 二级标题

### 三级标题

# 使用列表

- 无序列表项
- 简洁明了

# 使用表格对比

| 方案 A | 方案 B |
| ------ | ------ |
| 优点 A | 优点 B |

# 使用代码块

\`\`\`go
// 带语言标注
func example() {}
\`\`\`

# 使用引用

> 重要提示或警告
```

**避免**：

```markdown
❌ 过长的段落（> 5句话）
❌ 嵌套过深的列表（> 3层）
❌ 冗余的示例代码（> 50行）
❌ 外部链接过多（可能失效）
```

## 🚀 性能优化

### 1. 缓存策略

**启用缓存**：

```go
&types.SkillsPackageConfig{
    Path: "oss://my-bucket/skills/",
    CacheEnabled: true,
    CacheTTL:     30 * time.Minute,
    MaxCacheSize: 100 * 1024 * 1024, // 100MB
}
```

**缓存预热**：

```go
// 应用启动时预加载常用 Skills
func warmupCache(ctx context.Context, loader *skills.SkillLoader) {
    commonSkills := []string{
        "coding-standards",
        "security-checklist",
        "error-handling",
    }

    for _, name := range commonSkills {
        if _, err := loader.LoadByName(ctx, name); err != nil {
            log.Printf("预热 %s 失败: %v", name, err)
        }
    }
}
```

### 2. 延迟加载

**避免**：启动时加载所有 Skills

```go
// ❌ 不推荐
allSkills, _ := loader.LoadAll(ctx)
```

**推荐**：按需加载

```go
// ✅ 推荐
skill, _ := loader.LoadByName(ctx, "needed-skill")
```

### 3. 并发加载

**批量加载**：

```go
func loadSkillsConcurrently(
    ctx context.Context,
    loader *skills.SkillLoader,
    names []string,
) ([]*types.SkillDefinition, error) {
    var (
        wg      sync.WaitGroup
        mu      sync.Mutex
        skills  []*types.SkillDefinition
        errors  []error
    )

    for _, name := range names {
        wg.Add(1)
        go func(n string) {
            defer wg.Done()

            skill, err := loader.LoadByName(ctx, n)
            mu.Lock()
            defer mu.Unlock()

            if err != nil {
                errors = append(errors, err)
            } else {
                skills = append(skills, skill)
            }
        }(name)
    }

    wg.Wait()

    if len(errors) > 0 {
        return skills, fmt.Errorf("部分 Skills 加载失败: %v", errors)
    }

    return skills, nil
}
```

### 4. Token 优化

**动态裁剪**：

```go
type SkillTrimmer struct {
    maxTokens int
}

func (t *SkillTrimmer) TrimSkills(
    skills []*types.SkillDefinition,
    remainingTokens int,
) []*types.SkillDefinition {
    var (
        result      []*types.SkillDefinition
        totalTokens int
    )

    // 按优先级排序（always > context > keyword > file_pattern）
    sort.Slice(skills, func(i, j int) bool {
        return getPriority(skills[i]) > getPriority(skills[j])
    })

    for _, skill := range skills {
        tokens := estimateTokens(skill.Content)
        if totalTokens+tokens <= remainingTokens {
            result = append(result, skill)
            totalTokens += tokens
        }
    }

    return result
}
```

**内容压缩**：

```go
func compressSkill(skill *types.SkillDefinition) *types.SkillDefinition {
    // 移除示例代码块
    content := removeCodeBlocks(skill.Content)

    // 移除冗余空行
    content = removeExtraNewlines(content)

    // 简化列表
    content = simplifyLists(content)

    return &types.SkillDefinition{
        Name:        skill.Name,
        Description: skill.Description,
        Triggers:    skill.Triggers,
        Content:     content,
    }
}
```

## 🧪 测试与验证

### 1. Skill 单元测试

```go
func TestSkillLoading(t *testing.T) {
    loader, _ := skills.NewLoader(&types.SkillsPackageConfig{
        Path: "./testdata/skills",
    })

    skill, err := loader.LoadByName(context.Background(), "test-skill")
    assert.NoError(t, err)
    assert.Equal(t, "test-skill", skill.Name)
    assert.NotEmpty(t, skill.Content)
}

// 当前版本中，默认 Injector 不再根据 triggers 自动筛选技能。
// 如果你有自定义触发逻辑，可以在这里单独对触发器解析和过滤进行单元测试。
```

### 2. 集成测试

```go
func TestSkillWithAgent(t *testing.T) {
    ag, _ := agent.Create(ctx, &types.AgentConfig{
        TemplateID: "assistant",
        SkillsPackageConfig: &types.SkillsPackageConfig{
            Path: "./skills",
        },
    }, deps)

    result, err := ag.Chat(ctx, "帮我审查代码")
    assert.NoError(t, err)

    // 可以通过调试信息或日志确认 Active Skills 列表中包含预期技能
}
```

### 3. 性能基准测试

```go
func BenchmarkSkillLoading(b *testing.B) {
    loader, _ := skills.NewLoader(&types.SkillsPackageConfig{
        Path: "./skills",
    })

    ctx := context.Background()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = loader.LoadByName(ctx, "coding-standards")
    }
}

func BenchmarkSkillActivation(b *testing.B) {
    loader, _ := skills.NewLoader(&types.SkillsPackageConfig{
        Path: "./skills",
    })
    injector := skills.NewInjector(loader)

    ctx := context.Background()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = injector.ActivateSkills(ctx, "帮我审查代码", nil)
    }
}
```

## 📊 监控与审计

### 1. Skill 激活日志

```go
type SkillLogger struct {
    logger *slog.Logger
}

func (l *SkillLogger) LogActivation(
    ctx context.Context,
    skills []*types.SkillDefinition,
    trigger string,
) {
    skillNames := make([]string, len(skills))
    for i, s := range skills {
        skillNames[i] = s.Name
    }

    l.logger.InfoContext(ctx, "Skills activated",
        "skills", skillNames,
        "trigger", trigger,
        "count", len(skills),
    )
}
```

### 2. 性能指标

```go
type SkillMetrics struct {
    loadLatency    *prometheus.HistogramVec
    activationRate *prometheus.CounterVec
    cacheHitRate   *prometheus.CounterVec
}

func (m *SkillMetrics) RecordLoad(skillName string, duration time.Duration) {
    m.loadLatency.WithLabelValues(skillName).Observe(duration.Seconds())
}

func (m *SkillMetrics) RecordActivation(skillName string) {
    m.activationRate.WithLabelValues(skillName).Inc()
}

func (m *SkillMetrics) RecordCacheHit(hit bool) {
    label := "miss"
    if hit {
        label = "hit"
    }
    m.cacheHitRate.WithLabelValues(label).Inc()
}
```

### 3. 使用统计分析

```go
func analyzeSkillUsage(
    ctx context.Context,
    db *sql.DB,
    timeRange time.Duration,
) (map[string]int, error) {
    query := `
        SELECT skill_name, COUNT(*) as count
        FROM skill_activations
        WHERE activated_at > NOW() - INTERVAL ? SECOND
        GROUP BY skill_name
        ORDER BY count DESC
    `

    rows, err := db.QueryContext(ctx, query, timeRange.Seconds())
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    usage := make(map[string]int)
    for rows.Next() {
        var name string
        var count int
        if err := rows.Scan(&name, &count); err != nil {
            return nil, err
        }
        usage[name] = count
    }

    return usage, nil
}
```

## 🔧 故障排查

### 1. Skill 未激活

**症状**：预期的 Skill 没有被注入

**排查步骤**：

```go
// 1. 检查触发条件
skill, _ := loader.LoadByName(ctx, "skill-name")
for _, trigger := range skill.Triggers {
    log.Printf("Trigger: %+v", trigger)
}

// 2. 启用调试日志
injector.SetDebugMode(true)
activated, _ := injector.ActivateSkills(ctx, userMsg, execCtx)
for _, s := range activated {
    log.Printf("Activated: %s", s.Name)
}

// 3. 验证关键词匹配
if trigger.Type == types.TriggerTypeKeyword {
    for _, keyword := range trigger.Keywords {
        if strings.Contains(userMsg, keyword) {
            log.Printf("Matched keyword: %s", keyword)
        }
    }
}
```

### 2. Token 超限

**症状**：Skill 内容过长导致上下文溢出

**解决方案**：

```go
// 方案1：拆分 Skill
// 将大 Skill 拆分成多个小 Skill

// 方案2：动态裁剪
trimmer := &SkillTrimmer{maxTokens: 2000}
trimmedSkills := trimmer.TrimSkills(activated, remainingTokens)

// 方案3：压缩内容
for _, skill := range activated {
    skill.Content = compressContent(skill.Content)
}
```

### 3. 加载性能问题

**症状**：Skills 加载耗时过长

**优化措施**：

```go
// 1. 启用缓存
config.CacheEnabled = true
config.CacheTTL = 30 * time.Minute

// 2. 使用 CDN
config.Path = "https://cdn.example.com/skills/"

// 3. 预加载
go warmupCache(ctx, loader)

// 4. 并发加载
skills, _ := loadSkillsConcurrently(ctx, loader, names)
```

## 📚 相关资源

- [Skills 核心概念](/core-concepts/skills-system) - 系统设计和原理
- [Skills API 参考](/api-reference/skills) - 完整 API 文档
- [自定义工具](/tools/builtin/custom) - Skills 与其他扩展方式对比
- [示例项目](/examples/skills) - 实际应用案例

---

**最佳实践总结**：

1. ✅ **细粒度设计**：每个 Skill 专注单一领域
2. ✅ **清晰触发**：选择合适的触发类型
3. ✅ **内容精炼**：控制大小，避免冗余
4. ✅ **性能优化**：启用缓存，延迟加载
5. ✅ **持续监控**：记录激活日志，分析使用统计
