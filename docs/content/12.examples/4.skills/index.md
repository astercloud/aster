---
title: Skills 系统示例
description: 使用 Skills 为 Agent 添加领域专业知识
navigation:
  icon: i-lucide-zap
---

# Skills 系统示例

Skills 是 aster 的动态知识库注入系统，允许您为 Agent 添加领域专业知识、工作流程和最佳实践。

## 📦 示例 Skills

本目录包含以下实用 Skills 示例：

### 1. Markdown 分段翻译器 (v0.8.0)

**位置**: `examples/skills/workspace/skills/markdown-segment-translator/`

自动将长Markdown文档分段翻译，保持格式和学术术语准确性。

**主要特性**：

- ✅ 自动分段（200行/段，可配置）
- ✅ 保持Markdown格式
- ✅ 学术术语翻译优化
- ✅ 自动合并翻译结果

**触发方式**：

- 关键词：`翻译`、`translate`
- 命令：包含 `.md` 文件

**使用场景**：

- 学术论文翻译
- 技术文档本地化
- 长篇内容翻译

[查看详细文档](./markdown-translator.md)

---

### 2. PDF 处理器

**位置**: `examples/skills/workspace/skills/pdf/`

完整的PDF处理工具集，支持提取、转换、表单填写等操作。

**主要特性**：

- 📄 PDF转图片
- 📝 PDF转Markdown
- 📋 表单字段提取和填写
- 🔍 边界框检查

**触发方式**：

- 关键词：`pdf`、`表单`、`填写`
- 文件：`.pdf` 扩展名

**使用场景**：

- PDF文档提取
- 表单自动填写
- PDF内容分析

[查看详细文档](./pdf-processor.md)

---

### 3. PDF转Markdown

**位置**: `examples/skills/workspace/skills/pdfmd/`

简化版PDF提取工具，专注于将PDF转换为Markdown。

**主要特性**：

- ⚡ 快速提取
- 📝 保持格式
- 🎯 简单易用

**触发方式**：

- 关键词：`pdf`、`extract`、`提取`

**使用场景**：

- 学术论文阅读
- 文档内容提取
- 快速预览

---

### 4. 一致性检查器

**位置**: `examples/skills/workspace/skills/consistency-checker/`

写作过程中自动检查角色行为、世界规则和时间线一致性。

**主要特性**：

- 👤 角色行为一致性
- 🌍 世界规则检查
- ⏰ 时间线验证

**触发方式**：

- 关键词：`一致性`、`检查`、`consistency`
- 上下文：在 `/write` 命令期间

**使用场景**：

- 小说创作
- 剧本写作
- 世界观构建

[查看详细文档](./consistency-checker.md)

---

## 🚀 快速开始

### 1. 查看现有 Skills

```bash
cd examples/skills/workspace/skills/
ls -la
```

### 2. 使用 Skill

Skills 会根据关键词或上下文自动激活：

```go
// Agent 会自动检测并注入相关 Skills
agent.Chat(ctx, "请帮我翻译这个 test.md 文件")
// → 自动激活 markdown-segment-translator skill
```

### 3. 创建自定义 Skill

创建一个新的 Skill 目录：

```bash
mkdir -p workspace/skills/my-skill/
cd workspace/skills/my-skill/
```

创建 `SKILL.md` 文件：

```markdown
---
name: my-skill
description: 我的自定义技能
triggers:
  - type: keyword
    keywords: ["关键词1", "关键词2"]
---

# 我的技能

这里是技能的详细说明...

## 使用方法

...
```

---

## 📊 Skills 对比

| Skill              | 代码量 | 触发方式      | 适用场景 | v0.8.0优化  |
| ------------------ | ------ | ------------- | -------- | ----------- |
| **Markdown翻译器** | 300行  | 关键词/文件   | 文档翻译 | ✅ 重点优化 |
| **PDF处理器**      | 2000行 | 关键词/文件   | PDF操作  | -           |
| **PDF转Markdown**  | 100行  | 关键词        | 快速提取 | -           |
| **一致性检查器**   | 50行   | 关键词/上下文 | 创意写作 | -           |

---

## 🎯 性能优化 (v0.8.0)

### Markdown翻译器优化

**优化前**：

- 单次调用翻译
- 大文档易超时
- 不支持断点续传

**优化后**：

- ✅ 自动分段处理（200行/段）
- ✅ 非流式模式加速（3-5倍）
- ✅ 支持断点续传
- ✅ Token消耗降低20%

**性能对比**：

| 文档大小 | 优化前 | 优化后 | 提升 |
| -------- | ------ | ------ | ---- |
| 500行    | 60秒   | 15秒   | 4倍  |
| 1000行   | 120秒  | 30秒   | 4倍  |
| 2500行   | 超时   | 90秒   | ∞    |

---

## 📚 相关文档

- [Skills 系统核心概念](../../02.core-concepts/9.skills-system.md)
- [Skills API 参考](../../14.api-reference/9.skills/overview.md)
- [Skills 最佳实践](../../15.best-practices/skills.md)

---

## 💡 最佳实践

### 1. Skill 命名

使用短横线连接的小写名称：

- ✅ `markdown-translator`
- ✅ `pdf-processor`
- ❌ `MarkdownTranslator`
- ❌ `pdf_processor`

### 2. 触发器设置

合理设置触发条件，避免误触发：

```yaml
triggers:
  - type: keyword
    keywords: ["精确关键词", "specific-term"]
  - type: context
    condition: "during /command" # 限定上下文
```

### 3. 性能考虑

对于大文件处理：

- 使用分段处理
- 配置 ExecutionMode: NonStreaming
- 提供进度反馈

### 4. 错误处理

在 Skill 脚本中添加完善的错误处理：

```python
try:
    result = process_document(input_file)
except FileNotFoundError:
    print(f"错误：文件不存在 {input_file}")
    sys.exit(1)
except Exception as e:
    print(f"错误：{str(e)}")
    sys.exit(1)
```

---

**v0.8.0 更新**: 重点优化了 Markdown翻译器的性能和稳定性 🎉
