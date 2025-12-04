---
title: Provider 使用指南
description: 统一的 Provider 配置方式与推荐组合
navigation:
  icon: i-lucide-code-2
---

# Provider 使用指南

为了避免「每个 Provider 都一大段文档」的复杂性, 本页只保留统一的使用方式和推荐组合, 其他细节仅在需要时参考。

## 1. 统一的配置方式

无论是 OpenAI、Gemini、DeepSeek 还是其他 Provider, 使用方式都统一为一套 `ModelConfig` + `MultiProviderFactory`:

```go
import (
  "github.com/astercloud/aster/pkg/provider"
  "github.com/astercloud/aster/pkg/types"
)

config := &types.ModelConfig{
  Provider: "openai",   // 统一使用 Provider 字符串切换
  Model:    "gpt-4o",   // 模型名称
  APIKey:   "sk-xxx",   // API Key
}

factory := provider.NewMultiProviderFactory()
p, err := factory.Create(config)
```

切换 Provider 只需要改 `Provider`/`Model` 字段, 代码基本不变:

```go
config.Provider = "deepseek"
config.Model    = "deepseek-chat"
```

## 2. 推荐的「最小组合」

为了让文档更简洁,我们只推荐几种常用组合,其他 Provider 视为兼容选项:

- **通用场景**: `openai + gpt-4o`
- **复杂推理/性价比**: `deepseek + deepseek-reasoner`
- **中国市场合规**: `glm`/`doubao`/`moonshot` 之一
- **本地开发/私有部署**: `ollama`

实现上, MultiProviderFactory 仍然支持更多 Provider, 但你可以只在项目中采用上述 3~4 种, 文档也只重点介绍这些。

## 3. OpenAI / Gemini 兼容说明(可选阅读)

如果你需要更细节的多模态/视频/Prompt Cache 说明:

- OpenAI 细节: 支持 o1/o3、Prompt Caching、多模态等;
- Gemini 细节: 支持 1M+ 上下文、视频理解。

出于简洁考虑,这些细节保存在独立页面中,但默认不在左侧导航中显示:

- `OpenAI Provider 细节` – `docs/content/3.providers/openai.md`
- `Gemini Provider 细节` – `docs/content/3.providers/gemini.md`

只有在你真正需要用到对应高级特性时,再去翻这两篇文档即可。

## 4. 在 Agent 中使用 Provider

在 Agent 侧,不需要直接操作 Provider,只需在模板或配置中写好 `ModelConfig`:

```go
templateRegistry.Register(&types.AgentTemplateDefinition{
  ID:    "assistant",
  Model: "gpt-4o",
  // 其他字段...
})

// 或在 AgentConfig 中显式指定:
config := &types.AgentConfig{
  TemplateID: "assistant",
  ModelConfig: &types.ModelConfig{
    Provider: "openai",
    Model:    "gpt-4o",
    APIKey:   os.Getenv("OPENAI_API_KEY"),
  },
}
```

这样既保留了多 Provider 兼容能力, 又不会在文档上堆积大量重复/冗长的说明。

## 5. ExecutionMode 配置

aster 现在支持配置执行模式，优化不同场景的性能：

```go
config := &types.ModelConfig{
  Provider:      "deepseek",
  Model:         "deepseek-chat",
  APIKey:        os.Getenv("DEEPSEEK_API_KEY"),
  ExecutionMode: types.ExecutionModeNonStreaming, // 🆕 执行模式
}
```

### 执行模式类型

| 模式                        | 说明             | 适用场景             | 性能            |
| --------------------------- | ---------------- | -------------------- | --------------- |
| `ExecutionModeStreaming`    | 流式输出（默认） | 交互式对话、实时反馈 | 用户体验好      |
| `ExecutionModeNonStreaming` | 非流式批处理     | 翻译、批量处理       | **速度快3-5倍** |
| `ExecutionModeAuto`         | 自动选择         | 通用场景             | 智能优化        |

### 使用示例

```go
// 场景1：交互式对话（默认，实时反馈）
chatConfig := &types.ModelConfig{
  Provider:      "openai",
  Model:         "gpt-4o",
  ExecutionMode: types.ExecutionModeStreaming,
}

// 场景2：批量翻译（快速模式）
translateConfig := &types.ModelConfig{
  Provider:      "deepseek",
  Model:         "deepseek-chat",
  ExecutionMode: types.ExecutionModeNonStreaming, // 3-5倍速度提升
}
```

### 性能对比

| 任务          | Streaming | NonStreaming | 性能提升   |
| ------------- | --------- | ------------ | ---------- |
| 翻译200行文档 | 30秒      | 5-10秒       | **3-5倍**  |
| Token消耗     | 标准      | 降低20%      | **更省钱** |
| 用户体验      | 实时反馈  | 快速完成     | 各有优势   |
