---
title: Markdown 分段翻译器
description: 自动分段翻译长Markdown文档
navigation:
  icon: i-lucide-file-text
---

# Markdown 分段翻译器

v0.8.0 重点优化的 Skill，实现高性能、高质量的Markdown文档分段翻译。

## 🎯 功能特性

### 核心功能

- **自动分段**: 按行数（默认200行/段）智能分割文档
- **格式保持**: 完整保留Markdown语法和结构
- **学术优化**: 准确翻译学术术语和专业词汇
- **自动合并**: 翻译完成后自动合并所有段落

### v0.8.0 性能优化

- ⚡ **3-5倍速度提升**: 使用NonStreaming模式
- 💰 **20%成本降低**: Token消耗优化
- 🔄 **断点续传**: 支持中断后继续
- 📊 **进度追踪**: 实时显示翻译进度

---

## 📁 文件结构

```
workspace/skills/markdown-segment-translator/
├── SKILL.md                    # Skill定义和使用说明
└── scripts/
    └── segment_tool.py         # 分段和合并工具
```

---

## 🚀 快速开始

### 基本使用

```bash
# 1. 准备Markdown文件
cp your-document.md workspace/

# 2. 使用Agent翻译
go run ./main.go -message "翻译 your-document.md"
```

Agent 会自动：

1. 检测到翻译需求，激活此Skill
2. 调用 `segment_tool.py` 分段
3. 逐段翻译（使用Agent的LLM）
4. 调用 `segment_tool.py` 合并
5. 输出最终翻译文件

### 手动分段和合并

```bash
# 分段
python3 workspace/skills/markdown-segment-translator/scripts/segment_tool.py segment \
  --input document.md \
  --segment-size 200 \
  --max-segments 0  # 0表示不限制

# 输出到 workspace/output/segments/segment_*.md

# 合并
python3 workspace/skills/markdown-segment-translator/scripts/segment_tool.py merge

# 输出到 workspace/output/final/complete_translated_*.md
```

---

## ⚙️ 配置选项

### segment_tool.py 参数

| 参数             | 说明     | 默认值           | 推荐值           |
| ---------------- | -------- | ---------------- | ---------------- |
| `--segment-size` | 每段行数 | 1000             | **200** (v0.8.0) |
| `--max-segments` | 最大段数 | 无限制           | 0 (不限制)       |
| `--input`        | 输入文件 | -                | 必填             |
| `--output-dir`   | 输出目录 | workspace/output | -                |

### Agent ExecutionMode 配置

```go
// 在 main.go 中配置
ModelConfig: &types.ModelConfig{
    Provider:      "deepseek",
    Model:         "deepseek-chat",
    ExecutionMode: types.ExecutionModeNonStreaming, // 🚀 快速模式
}
```

---

## 📊 性能测试数据

### 测试环境

- **模型**: DeepSeek Chat
- **ExecutionMode**: NonStreaming
- **网络**: 标准网络环境

### 不同segment-size对比

| Segment大小 | 文件大小 | 翻译时间      | API调用     | 稳定性      | 推荐     |
| ----------- | -------- | ------------- | ----------- | ----------- | -------- |
| 50行        | 5KB      | 很快          | 多次        | ✅ 稳定     | 测试用   |
| **200行**   | 15-20KB  | **5-10秒/段** | **11次/段** | ✅ **稳定** | **推荐** |
| 500行       | 50KB     | 20-30秒/段    | 13次/段     | ⚠️ 偶尔慢   | 中等文档 |
| 1000行      | 100KB    | 60秒+/段      | 15次/段     | ❌ 易超时   | 不推荐   |

### 完整文档翻译性能

**测试文档**: 学术论文 (2500行, 约150KB)

| 配置                  | 总时间   | Token     | 成本      | 成功率   |
| --------------------- | -------- | --------- | --------- | -------- |
| **v0.8.0 (200行/段)** | **90秒** | **~200K** | **¥0.20** | **100%** |
| v0.7.0 (1000行/段)    | 300秒+   | ~250K     | ¥0.25     | 60%      |
| v0.7.0 (单次翻译)     | 超时     | -         | -         | 0%       |

**性能提升**：

- ⚡ 速度提升：**3-4倍**
- 💰 成本降低：**20%**
- ✅ 成功率：**40%→100%**

---

## 🎓 使用示例

### 示例1: 翻译学术论文

```bash
# 输入: 英文论文 paper.md (2500行)
# 输出: 中文论文 complete_translated_paper.md

# Agent命令
go run ./main.go -message "翻译paper.md"

# 或手动流程
python3 segment_tool.py segment --input paper.md --segment-size 200
# 然后使用Agent翻译每个segment
python3 segment_tool.py merge
```

**预期结果**：

- 分成13个segment (200行/段)
- 每段翻译约5-10秒
- 总时间约90-130秒
- 完整保留格式和公式

### 示例2: 翻译技术文档

```bash
# 输入: 技术文档 README.md (500行)
go run ./main.go -message "将README.md翻译成中文"
```

**预期结果**：

- 分成3个segment
- 总时间约20-30秒
- 保留代码块、链接等格式

---

## 📝 Skill 定义 (SKILL.md)

```markdown
---
name: markdown-segment-translator
description: 将长Markdown文档分段翻译，保持格式和术语准确性
allowed-tools: ["Bash", "Read", "Write"]
triggers:
  - type: keyword
    keywords: ["翻译", "translate", "translation"]
  - type: file
    extensions: [".md"]
version: 2.0.0 # v0.8.0 优化版本
---

# Markdown 分段翻译技能

## 功能说明

此技能专门用于翻译长Markdown文档，采用分段处理策略...

## 使用方法

### 3步工作流

1. **分段**: 使用Bash调用segment_tool.py segment
2. **翻译**: 对每个segment使用Agent的LLM翻译
3. **合并**: 使用Bash调用segment_tool.py merge

### 严格规则

⚠️ 你（Agent）必须严格按照以下规则执行：

1. **禁止**: 不要试图一次性翻译整个文档
2. **必须**: 使用Bash执行Python脚本
3. **必须**: 使用你自己的LLM翻译每个segment
4. **必须**: 保持Markdown格式完整

...
```

---

## 🔧 技术实现

### segment_tool.py 核心逻辑

```python
class MarkdownSegmentTool:
    def segment_document(self, input_file, segment_size, max_segments):
        """分段文档"""
        lines = self.read_file(input_file)
        total_lines = len(lines)

        # 计算分段数 - 严格按segment_size分段
        num_segments = (total_lines + segment_size - 1) // segment_size
        if max_segments and num_segments > max_segments:
            num_segments = max_segments

        # 创建segments
        for i in range(num_segments):
            start = i * segment_size
            end = min(start + segment_size, total_lines)
            segment_lines = lines[start:end]

            self.write_segment(i + 1, segment_lines)

        return num_segments

    def merge_translations(self):
        """合并翻译结果"""
        segment_files = sorted(glob.glob("output/translations/translated_segment_*.md"))

        merged_content = []
        for file in segment_files:
            content = self.read_file(file)
            merged_content.extend(content)

        self.write_merged(merged_content)
```

### Agent翻译逻辑

```go
// Agent会自动执行以下步骤

// 1. 调用分段工具
agent.executeToolCall("Bash", map[string]interface{}{
    "command": "python3 segment_tool.py segment --input doc.md --segment-size 200",
})

// 2. 翻译每个segment
for i := 1; i <= numSegments; i++ {
    // 读取segment
    content := agent.executeToolCall("Read", map[string]interface{}{
        "path": fmt.Sprintf("output/segments/segment_%d.md", i),
    })

    // 翻译（使用Agent的LLM）
    translated := agent.translate(content, "中文")

    // 保存翻译
    agent.executeToolCall("Write", map[string]interface{}{
        "path": fmt.Sprintf("output/translations/translated_segment_%d.md", i),
        "content": translated,
    })
}

// 3. 合并结果
agent.executeToolCall("Bash", map[string]interface{}{
    "command": "python3 segment_tool.py merge",
})
```

---

## 🐛 故障排除

### 问题1: API超时

**症状**: 翻译过程中卡住或超时

**原因**: segment-size 设置过大

**解决**:

```bash
# 减小segment-size
python3 segment_tool.py segment --input doc.md --segment-size 200  # ✅
python3 segment_tool.py segment --input doc.md --segment-size 1000 # ❌
```

### 问题2: 格式丢失

**症状**: 翻译后Markdown格式不完整

**原因**: 分段位置切断了代码块或表格

**解决**: 手动调整分段位置，或使用更小的segment-size

### 问题3: 翻译质量不佳

**症状**: 学术术语翻译不准确

**解决**:

1. 在SKILL.md中添加术语表
2. 使用更好的模型（如deepseek-reasoner）
3. 在System Prompt中添加专业领域说明

---

## 📈 未来优化方向

1. **智能分段**: 根据Markdown结构（标题、章节）智能分段
2. **术语库**: 支持自定义术语表
3. **多语言**: 支持多种目标语言
4. **并行翻译**: 同时翻译多个segment
5. **质量检查**: 自动检查翻译质量

---

## 🔗 相关资源

- [ExecutionMode 配置](../../03.providers/usage.md#executionmode-配置-v080)
- [Performance 优化](../../15.best-practices/performance.md)
- [Skills 系统核心概念](../../02.core-concepts/9.skills-system.md)

---

**v0.8.0 重点优化**: 此Skill是v0.8.0的核心优化内容，性能提升3-5倍！🚀
