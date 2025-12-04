# Slash Commands & Agent Skills 示例

本示例展示如何在 WriteFlow SDK 中使用 **Slash Commands** 和 **Agent Skills** 功能。

## 核心特性

### 1. 多模型支持

支持任何 LLM 模型，包括：

- ✅ Claude (Anthropic)
- ✅ GPT-4 (OpenAI)
- ✅ 通义千问 (Qwen)
- ✅ Gemini (Google)
- ✅ 任何符合接口的模型

### 2. Slash Commands

用户主动触发的命令系统：

```go
// 发送 slash command
agent.Send(ctx, "/write 第1章")
agent.Send(ctx, "/analyze")
agent.Send(ctx, "/plan")
```

**工作流程：**

1. 检测到 `/` 开头的消息
2. 从 `commands/` 目录加载命令定义（.md 文件）
3. 执行前置脚本（如果有）
4. 渲染提示词模板
5. 发送给 LLM 处理

### 3. Agent Skills

基于文件系统的“技能包”系统：

```go
// 普通对话，模型会在需要时使用文件/Bash 工具主动加载相关 skills
agent.Send(ctx, "帮我检查一致性问题")
// → 模型在看到 Active Skills 列表后，如果判断需要，就先读取对应 SKILL.md，再按其中步骤执行
```

**工作流程：**

1. Agent 根据配置加载 `workspace/skills/**/SKILL.md`，解析其中 YAML frontmatter
2. 所有启用的 skills 只将 **元数据**（`name` + `description` + SKILL.md 路径）注入 SystemPrompt / UserMessage
3. 当模型认为某个 skill 相关时，会先用 `Read` 或 `Bash` 工具主动打开对应的 `SKILL.md`
4. 模型根据 `SKILL.md` 中的详细说明和引用的脚本（Python 等），继续通过现有工具链完成任务
5. 只有被真正需要的 skill 内容才会进入上下文，实现渐进式加载

## 项目结构

```
workspace/
├── commands/              # Slash Commands 定义
│   ├── write.md          # /write 命令
│   ├── analyze.md        # /analyze 命令
│   └── plan.md           # /plan 命令
│
├── skills/               # Agent Skills 定义
│   ├── consistency-checker/
│   │   └── SKILL.md     # 一致性检查技能
│   ├── workflow-guide/
│   │   └── SKILL.md     # 工作流指导技能
│   └── ...
│
└── scripts/             # 可选：辅助脚本
    ├── bash/
    └── powershell/
```

## 命令定义格式

`commands/write.md` 示例：

```markdown
---
description: 执行章节写作
argument-hint: [章节编号]
allowed-tools: ["Read", "Write", "Bash"]
models:
  preferred:
    - claude-sonnet-4-5
    - gpt-4-turbo
  minimum-capabilities:
    - tool-calling
scripts:
  sh: scripts/bash/check-state.sh
---

## 写作执行流程

1. 检查前置条件
2. 加载上下文
3. 执行写作
4. 保存结果

...（详细提示词）
```

## Skills 定义格式（与 Claude 对齐）

`skills/consistency-checker/SKILL.md` 示例：

```markdown
---
name: consistency-checker
description: 自动检查一致性问题
allowed-tools: ["Read", "Grep"]
---

# 一致性检查系统

## 自动监控

- 角色特征一致性
- 世界观规则一致性
- 时间线逻辑一致性

...（详细知识库内容）
```

## 使用示例

### 示例 1: 基本使用

```go
agent, err := agent.Create(ctx, &types.AgentConfig{
    ModelConfig: &types.ModelConfig{
        Provider: "anthropic",
        Model:    "claude-sonnet-4",
        APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
    },
    SkillsPackage: &types.SkillsPackageConfig{
        Source:          "local",
        Path:            "./workspace",
        EnabledCommands: []string{"write", "analyze"},
        EnabledSkills:   []string{"consistency-checker"},
    },
}, deps)

// 使用 slash command
agent.Send(ctx, "/write 第1章")

// 普通对话（自动激活 skills）
agent.Send(ctx, "帮我检查第1章的一致性问题")
```

### 示例 2: 多模型支持

```go
// 使用 Claude
claudeAgent, _ := agent.Create(ctx, &types.AgentConfig{
    ModelConfig: &types.ModelConfig{
        Provider: "anthropic",
        Model:    "claude-sonnet-4",
    },
    SkillsPackage: skillsConfig,
})

// 使用 GPT-4（相同的配置）
gptAgent, _ := agent.Create(ctx, &types.AgentConfig{
    ModelConfig: &types.ModelConfig{
        Provider: "openai",
        Model:    "gpt-4-turbo",
    },
    SkillsPackage: skillsConfig, // 相同的 skills 配置
})

// 使用通义千问
qwenAgent, _ := agent.Create(ctx, &types.AgentConfig{
    ModelConfig: &types.ModelConfig{
        Provider: "qwen",
        Model:    "qwen-max",
    },
    SkillsPackage: skillsConfig, // 相同的 skills 配置
})

// 所有 agent 使用相同的接口
claudeAgent.Send(ctx, "/write 第1章")
gptAgent.Send(ctx, "/write 第1章")
qwenAgent.Send(ctx, "/write 第1章")
```

### 示例 3: 从 OSS 加载技能包

```go
agent, err := agent.Create(ctx, &types.AgentConfig{
    Sandbox: &types.SandboxConfig{
        Kind: types.SandboxKindHybrid,
        HybridConfig: &types.HybridConfig{
            SkillsSource: "oss://my-bucket/skills/v1.0.0/",
        },
    },
    SkillsPackage: &types.SkillsPackageConfig{
        Source:  "hybrid",  // 从 hybrid sandbox 加载
        Path:    "/skills",
        Version: "v1.0.0",
        EnabledCommands: []string{"write", "analyze"},
        EnabledSkills:   []string{"consistency-checker"},
    },
}, deps)
```

## 模型能力自适应

SDK 会自动检测模型能力并适配：

```go
// 对于支持 SystemPrompt 的模型（Claude, GPT-4）
// → Skills 注入到 SystemPrompt

// 对于不支持 SystemPrompt 的模型
// → Skills 信息添加到 UserMessage
```

## 运行示例

```bash
# 设置环境变量
export ANTHROPIC_API_KEY="your-key"
export OPENAI_API_KEY="your-key"

# 准备技能包
mkdir -p workspace/commands
mkdir -p workspace/skills/consistency-checker
# 创建命令和技能定义文件...

# 运行示例
go run main.go
```

## 最佳实践

1. **命令定义**：
   - 使用清晰的描述
   - 提供参数提示
   - 指定所需的工具和能力

2. **Skills 定义**：
   - frontmatter 中的 `name` 必须是 1-64 个字符的 `lowercase + number + -` 组合，不能包含 `anthropic` 或 `claude`
   - `description` 必须非空，长度不超过 1024 字符，描述“这个 Skill 做什么 + 什么时候用”
   - 详细操作流程写在 SKILL.md 正文中，假设模型会先用 `Read`/`Bash cat` 打开这个文件再执行

3. **目录结构**：
   - 按功能组织命令
   - 按类别组织 skills
   - 保持文件命名一致

4. **模型选择**：
   - 根据任务选择合适的模型
   - 检查模型能力要求
   - 考虑成本和性能平衡

## 参考资料

- [Commands 实现](../../pkg/commands/)
- [Skills 实现](../../pkg/skills/)
- [Provider 接口](../../pkg/provider/)
- [Agent 实现](../../pkg/agent/)
