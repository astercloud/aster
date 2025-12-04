---
title: SKILL.md 编写指南
description: 如何为 aster 编写高质量的 SKILL.md 技能说明
navigation:
  icon: i-lucide-file-text
---

# SKILL.md 编写指南

本文聚焦在一件事：**如何写出 Agent 真正能用好的 SKILL.md 文件**。它补充了《Skills 最佳实践》中关于目录和组织的内容，专门针对单个技能说明书的结构和风格。

## 1. 文件结构

每个 Skill 目录至少包含一个 `SKILL.md`，推荐结构如下：

```text
workspace/
  skills/
    markdown-segment-translator/
      SKILL.md
      scripts/
        segment_tool.py
    pdf/
      SKILL.md
      scripts/
        pdf_to_markdown.py
        fill_pdf_form.py
```

`SKILL.md` 本身由两部分组成：

```markdown
---
name: markdown-segment-translator
description: 将长 Markdown 文档按段切分并翻译，保持格式和术语准确性。适用于“翻译整篇 Markdown 文档”之类的请求。
allowed-tools: ["Bash", "Read", "Write"]
---

# Markdown 分段翻译技能

## 使用场景

…

## 操作步骤

…
```

- **YAML frontmatter**：提供结构化元数据，Agent 在 Level 1 只看到这一层；
- **正文**：视为“说明书”，Agent 需要时会用 `Read` / `Bash` 主动读这个文件，再按步骤操作。

## 2. 元数据规范（YAML）

当前实现中，`SkillLoader` 会严格校验 metadata，请务必遵守：

```yaml
---
name: markdown-segment-translator
description: 将长 Markdown 文档按段切分并翻译，保持格式和术语准确性。适用于“翻译整篇 Markdown 文档”之类的请求。
allowed-tools: ["Bash", "Read", "Write"]
---
```

**约束：**

- `name`：
  - 必须存在；
  - 1–64 个字符；
  - 只允许小写字母、数字、连字符（`-`）；
  - 不能包含 `anthropic` 或 `claude`；
  - 强烈建议与技能目录名一致（例如目录是 `markdown-segment-translator/`，name 也写 `markdown-segment-translator`）。
- `description`：
  - 必须非空；
  - 长度不超过 1024 字符；
  - 描述要同时包含“做什么”和“什么时候用”，例如：
    - ✅ `在写作过程中检查角色行为、世界设定和时间线一致性，适用于长篇小说写作。`
  - 不要包含类似 `<tag>` 这样的 XML 结构。
- `allowed-tools`：
  - 可选字段，用于提示“这个 Skill 通常会用到哪些工具”；
  - 字面值不会限制 Agent 的能力，但会出现在调试输出和未来工具路由中。

> 提示：可以根据实际需要在 YAML 中增加自定义字段（例如 `triggers`、`version` 等），当前默认注入器不会使用它们做激活判断，但你可以在自定义逻辑中解析。

## 3. 正文结构（说明书）

正文是 Agent 真正要“照着做”的说明书。建议使用固定骨架：

```markdown
# 技能名称（人类可读）

## 何时使用

- 列出 3–5 条“典型场景”
- 尽量使用用户原话风格，例如“当用户说‘翻译整篇 Markdown 文档’时…”

## 前置假设

- 环境依赖（例如：已安装 python3 / 某些 pip 包）
- 目录结构（用代码块展示 workspace 下的相对路径）

## 操作步骤

### 第 1 步：…

1. 用什么工具做什么事（必须写清楚命令示例）
2. 需要关注的输出字段（例如 Bash 返回的 `ok` / `output`）

### 第 2 步：…

…

## 错误处理

- 列出常见错误及建议的处理方式

## 安全注意事项

- 列出禁止做的事情（例如“不直接调用外部 API”“不修改原始文件”）
```

### 3.1 明确写出工具调用方式

不要假设 Agent 会“猜对”命令行。用真实命令举例：

````markdown
### 第 1 步：分段

使用 `Bash` 工具执行：

```bash
python3 workspace/skills/markdown-segment-translator/scripts/segment_tool.py \
  segment \
  --input workspace/2407.14333v5.md \
  --segment-size 1000 \
  --max-segments 3
```
````

执行完成后，你应该在输出中看到类似：

- `output/segments/segment_1.md`
- `output/segments/segment_2.md`
- `output/segments/segment_3.md`

````

这样模型可以直接复用示例，而不是自己发明命令。

### 3.2 明确写出文件读取/写入方式

对于 `Read` / `Write` 这类工具，也建议写成类似“调用模板”：

```markdown
### 第 2 步：翻译每个分段

对于每个 `output/segments/segment_X.md`：

1. 使用 `Read` 工具读取该文件内容：
   - `{"path": "output/segments/segment_1.md"}`
2. 使用自己的语言能力将内容翻译为中文，保持 Markdown 结构不变；
3. 使用 `Write` 工具将翻译结果写入：
   - `{"path": "output/translations/translated_segment_1.md", "content": "<你的翻译结果>"}`
````

## 4. 与系统提示配合

Skills 本身不会修改模板的 System Prompt，但推荐在模板中加上一段通用规则，让模型更好地使用 Skills。

示例（你可以按需调整措辞）：

```text
当系统提示中列出了 Active Skills / Skills Overview：
- 先阅读该列表，判断是否需要使用其中的某个 Skill；
- 如果需要使用某个 Skill，必须先使用文件类工具（例如 Read 或 Bash + cat）打开它的 SKILL.md 路径；
- 阅读 SKILL.md 中的说明，并严格按照其中给出的步骤和命令执行，而不是自行发明流程。
```

这段规则通常放在模板的 System Prompt 中，由 Agent 创建阶段统一注入。

## 5. 示例：一致性检查 Skill

下面是一个简化但完整的示例，展示前面所有建议如何落地：

```markdown
---
name: consistency-checker
description: 在长篇写作过程中检查角色行为、世界设定和时间线的一致性。适用于“检查这一章有没有前后矛盾”之类的请求。
allowed-tools: ["Read", "Grep"]
---

# 写作一致性检查 Skill

## 何时使用

- 用户要求“检查角色/设定/时间线一致性”
- 用户在写长篇故事，多次提到“前面说过”“和之前冲突”等
- 需要跨章节对比设定或角色档案

## 前置假设

- 角色档案位于 `spec/knowledge/characters/`
- 世界观设定位于 `spec/knowledge/worldbuilding/`
- 时间线记录为 `spec/tracking/timeline.json`
- 当前章节草稿位于 `drafts/current-chapter.md`

## 操作步骤

### 第 1 步：加载参考资料

使用 `Read` 工具读取以下文件（如存在）：

- 主角角色档案，例如：`spec/knowledge/characters/main-character.md`
- 当前章节草稿：`drafts/current-chapter.md`
- 时间线文件：`spec/tracking/timeline.json`

在后续分析中，将这些内容作为“事实来源”。

### 第 2 步：检查角色一致性

- 比对当前章节中的角色行为、外貌特征、知识状态是否与角色档案中的设定一致；
- 对发现的潜在冲突，给出：
  - 冲突描述
  - 参考来源（哪个文件、哪一段）
  - 建议修正方案

### 第 3 步：检查世界设定与时间线一致性

- 对照世界观设定文档和时间线 JSON，找出：
  - 时间顺序错乱
  - 设定被隐形修改
  - 明显违反已建立规则的情节

## 错误处理

- 如果任何参考文件不存在，说明清楚缺失的是哪一类信息，并继续使用你能获取到的部分做最佳努力分析；
- 如果文本中没有发现明确冲突，也要明确说明“未发现明显一致性问题”。

## 安全注意事项

- 不要随意修改原始参考文件；
- 如果用户要求自动修正文本，应先给出变更建议，再根据用户确认执行写入。
```

这个示例的结构、注释和命名都可以作为你编写其他 SKILL.md 的模板。
