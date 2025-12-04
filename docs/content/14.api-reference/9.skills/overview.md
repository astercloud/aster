---
title: Skills API
description: Skills 系统完整 API 参考文档
---

# Skills API 参考

本文档提供 Skills 系统的完整 API 参考，包括 Skill 定义、加载器、注入器和配置类型。

> 日常使用中，你通常只需要在 `AgentConfig` 中配置 `SkillsPackage`，以及在 `workspace/skills/**/SKILL.md` 下编写技能文件。下面的 Go API 更多用于高级集成或定制。

## 目录

- [Skill 定义](#skill-定义)
- [SkillLoader 加载器](#skillloader-加载器)
- [Injector 注入器](#injector-注入器)
- [配置类型](#配置类型)

---

## Skill 定义

### skills.SkillDefinition（内部结构）

`SkillDefinition` 是 `pkg/skills` 包内部使用的结构，用于承载从 `SKILL.md` 解析出来的技能信息。典型字段如下：

```go
type SkillDefinition struct {
    // 基础信息（来自 SKILL.md 的 YAML frontmatter）
    Name         string   // 技能名 (YAML: name)
    Description  string   // 描述   (YAML: description)
    AllowedTools []string // 建议使用的工具 (YAML: allowed-tools)

    // 位置相关信息（由加载器填充）
    Path    string // 相对于 skills 根目录的技能路径，例如 "pdfmd"
    BaseDir string // skills 根目录相对于沙箱工作目录的路径，例如 "skills"

    // 类型和扩展信息
    Kind          string               // "knowledge" | "executable" 等
    KnowledgeBase string               // SKILL.md 正文内容（不包含 frontmatter）
    Parameters    map[string]ParamSpec // 可选：参数结构描述
    Returns       map[string]ReturnSpec
    Executable    *ExecutableConfig    // 可选：可执行配置

    // 触发配置（当前版本仅作为元数据使用）
    Triggers []TriggerConfig
}
```

**注意：**

- `Name` / `Description` 的约束与《Skills 系统》和《SKILL.md 编写指南》中描述一致；
- `KnowledgeBase` 字段不会直接注入到系统提示词中，而是留在内存中，供需要时参考；
- 默认注入器只会把 `Name` / `Description` 和 `SKILL.md` 路径提示注入到提示词中。

---

## SkillLoader 加载器

### skills.NewLoader

创建 Skill 加载器实例。

```go
func NewLoader(baseDir string, fs sandbox.SandboxFS) *SkillLoader
```

**参数：**

- `baseDir`：skills 目录的相对路径，例如 `"skills"` 或 `"workspace/skills"`；
- `fs`：实际使用的沙箱文件系统（通常由 Agent 创建过程注入）。

**返回：**

- `*SkillLoader`：加载器实例。

### Load 与 Discover

SkillLoader 提供按路径加载和发现技能的能力：

```go
// 加载单个技能定义，skillPath 例如 "pdfmd" 或 "workflow/consistency-checker"
func (sl *SkillLoader) Load(ctx context.Context, skillPath string) (*SkillDefinition, error)

// 发现 baseDir 下所有包含 SKILL.md 的技能目录，返回路径列表（去掉 /SKILL.md 后缀）
func (sl *SkillLoader) Discover(ctx context.Context) ([]string, error)
```

大多数情况下，你不需要直接操作这些 API；Agent 会在创建时自动基于 `SkillsPackageConfig` 创建相应的 Loader 并加载启用的技能。

---

## Injector 注入器

### skills.NewInjector

创建 Skills 注入器实例。通常由 Agent 创建过程内部调用：

```go
type InjectorConfig struct {
    Loader        *SkillLoader
    EnabledSkills []string
    Provider      provider.Provider
    Capabilities  provider.ProviderCapabilities
}

func NewInjector(ctx context.Context, config *InjectorConfig) (*Injector, error)
```

**行为概述：**

- 根据 `EnabledSkills` 列表加载技能；
- 在每轮对话前，将这些技能视为“可用技能”；
- 只注入技能元数据到 System Prompt 或 User Message 中：
  - name
  - description
  - SKILL.md 路径提示（例如 `skills/pdfmd/SKILL.md`）
- 不注入 `KnowledgeBase` 正文，正文保留在技能结构中，需要时由模型通过文件工具主动加载。

### EnhanceSystemPrompt / PrepareUserMessage

核心入口有两个：

```go
// 根据当前上下文增强系统提示词（支持 System Prompt 的模型）
func (i *Injector) EnhanceSystemPrompt(ctx context.Context, basePrompt string, skillContext SkillContext) string

// 对不支持 System Prompt 的模型，修改用户消息（添加 Skills Overview 前缀）
func (i *Injector) PrepareUserMessage(message string, skillContext SkillContext) string
```

`SkillContext` 包含：

```go
type SkillContext struct {
    UserMessage string                 // 当前用户输入
    Command     string                 // 当前命令（如 "/write"）
    Files       []string               // 相关文件路径
    Metadata    map[string]interface{} // 额外元数据
}
```

当前实现中，`EnhanceSystemPrompt` 会：

- 不再根据 `Triggers` 做自动激活；
- 对所有启用的技能，构造一个类似这样的段落追加到 System Prompt：

```text
## Active Skills

- `consistency-checker`: 在写作过程中检查角色、世界设定和时间线一致性 (SKILL file: `skills/consistency-checker/SKILL.md`)
- `markdown-segment-translator`: 将长 Markdown 文档按段切分并翻译 (SKILL file: `skills/markdown-segment-translator/SKILL.md`)
```

`PrepareUserMessage` 在模型不支持 System Prompt 时，会在用户消息前添加类似的 `## Skills Overview` 前缀。

### 辅助方法

你也可以单独获取当前可用技能：

```go
// 返回可用技能列表（当前实现中，为所有启用技能）
func (i *Injector) ActivateSkills(ctx context.Context, skillContext SkillContext) []*SkillDefinition

// 返回技能名称列表，便于日志记录或调试
func (i *Injector) GetActiveSkillNames(skillContext SkillContext) []string
```

---

## 配置类型

### types.SkillsPackageConfig

Skills Package 配置，定义 skills 目录来源及启用的技能。

```go
type SkillsPackageConfig struct {
    // 技能包来源
    Source  string `json:"source"`  // "local" | "oss" | "s3" | "hybrid"
    Path    string `json:"path"`    // 本地路径或云端 URL
    Version string `json:"version"` // 版本号

    // 命令和技能目录
    CommandsDir string `json:"commands_dir"` // 默认 "commands"
    SkillsDir   string `json:"skills_dir"`   // 默认 "skills"

    // 启用的 commands 和 skills
    EnabledCommands []string `json:"enabled_commands"` // ["write", "analyze", ...]
    EnabledSkills   []string `json:"enabled_skills"`   // ["consistency-checker", ...]
}
```

**典型用法：**

```go
agentConfig := &types.AgentConfig{
    TemplateID: "assistant",
    ModelConfig: &types.ModelConfig{
        Provider: "anthropic",
        Model:    "claude-3-5-sonnet",
        APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
    },
    Sandbox: &types.SandboxConfig{
        Kind:    types.SandboxKindLocal,
        WorkDir: "./workspace",
    },
    SkillsPackage: &types.SkillsPackageConfig{
        Source:      "local",
        Path:        ".",      // 相对于 Sandbox.WorkDir
        CommandsDir: "commands",
        SkillsDir:   "skills",
        EnabledSkills: []string{
            "consistency-checker",
            "markdown-segment-translator",
            "pdf",
            "pdfmd",
        },
    },
}
```

在这种配置下，Agent 会：

- 使用本地 `./workspace` 作为沙箱工作目录；
- 在 `./workspace/skills` 下查找技能目录；
- 只将 `EnabledSkills` 中列出的技能作为 Active Skills 注入到提示词中。

---

## 完整使用示例

### 通过 AgentConfig 使用 Skills

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/astercloud/aster/pkg/agent"
    "github.com/astercloud/aster/pkg/provider"
    "github.com/astercloud/aster/pkg/sandbox"
    "github.com/astercloud/aster/pkg/store"
    "github.com/astercloud/aster/pkg/types"
)

func main() {
    ctx := context.Background()

    // 准备依赖
    deps := &agent.Dependencies{
        Store:            store.NewInMemory(),           // 示例中使用内存存储
        ToolRegistry:     /* 初始化工具注册表 */,
        SandboxFactory:   sandbox.NewFactory(),
        ProviderFactory:  provider.NewMultiProviderFactory(),
        TemplateRegistry: agent.NewTemplateRegistry(),
    }

    // 注册一个简单模板（省略 ToolsManual 配置等细节）
    deps.TemplateRegistry.Register(&types.AgentTemplateDefinition{
        ID:           "assistant",
        SystemPrompt: "You are a helpful assistant.",
        Tools:        []interface{}{"Read", "Write", "Bash"},
    })

    // 创建 Agent，并启用 SkillsPackage
    ag, err := agent.Create(ctx, &types.AgentConfig{
        TemplateID: "assistant",
        ModelConfig: &types.ModelConfig{
            Provider: "anthropic",
            Model:    "claude-3-5-sonnet",
            APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
        },
        Sandbox: &types.SandboxConfig{
            Kind:    types.SandboxKindLocal,
            WorkDir: "./workspace",
        },
        SkillsPackage: &types.SkillsPackageConfig{
            Source:      "local",
            Path:        ".",      // 相对于 WorkDir
            CommandsDir: "commands",
            SkillsDir:   "skills",
            EnabledSkills: []string{
                "consistency-checker",
                "markdown-segment-translator",
            },
        },
    }, deps)
    if err != nil {
        log.Fatal(err)
    }
    defer ag.Close()

    // 使用 Agent，对话时 Active Skills 会自动注入到提示词中
    result, err := ag.Chat(ctx, "帮我检查一下第 3 章和前面章节在角色设定上有没有矛盾")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(result.Text)
}
```

在这个例子中：

- Skills 不会把完整的 SKILL.md 内容直接注入提示词；
- 模型会在需要时，通过 `Read` / `Bash` 等工具主动打开 `skills/<name>/SKILL.md`，再按照说明中的步骤去执行脚本或其它工具。

这就实现了依赖文件系统的“渐进式加载”能力：系统提示轻量、技能内容在真正需要时再被加载。
