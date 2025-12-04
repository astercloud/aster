---
title: Provider API
description: Provider接口完整参考文档
---

# Provider API 参考

本文档提供 Provider 核心 API 的完整参考，涵盖流式/非流式对话、工具调用、多模态支持等功能。

## 目录

- [接口概览](#接口概览)
- [创建 Provider](#创建-provider)
- [对话方法](#对话方法)
- [配置管理](#配置管理)
- [类型定义](#类型定义)
- [支持的 Providers](#支持的-providers)

---

## 接口概览

Provider 是模型提供商的统一抽象接口，所有 Provider 实现相同的接口：

```go
type Provider interface {
    // 流式对话
    Stream(ctx context.Context, messages []types.Message, opts *StreamOptions) (<-chan StreamChunk, error)

    // 非流式对话
    Complete(ctx context.Context, messages []types.Message, opts *StreamOptions) (*CompleteResponse, error)

    // 配置管理
    Config() *types.ModelConfig
    Capabilities() ProviderCapabilities
    SetSystemPrompt(prompt string) error
    GetSystemPrompt() string

    // 资源管理
    Close() error
}
```

---

## 创建 Provider

### 使用工厂创建

推荐使用 `MultiProviderFactory` 创建 Provider：

```go
import (
    "github.com/astercloud/aster/pkg/provider"
    "github.com/astercloud/aster/pkg/types"
)

// 创建工厂
factory := provider.NewMultiProviderFactory()

// 创建 Provider
config := &types.ModelConfig{
    Provider: "openai",
    Model:    "gpt-4o",
    APIKey:   os.Getenv("OPENAI_API_KEY"),
}

p, err := factory.Create(config)
if err != nil {
    log.Fatal(err)
}
defer p.Close()
```

### 支持的 Provider 名称

| Provider   | 配置名称     | 别名        |
| ---------- | ------------ | ----------- |
| OpenAI     | `openai`     | -           |
| Anthropic  | `anthropic`  | -           |
| Gemini     | `gemini`     | `google`    |
| Groq       | `groq`       | -           |
| OpenRouter | `openrouter` | -           |
| Mistral    | `mistral`    | -           |
| Ollama     | `ollama`     | -           |
| DeepSeek   | `deepseek`   | -           |
| 智谱 GLM   | `glm`        | `zhipu`     |
| 豆包       | `doubao`     | `bytedance` |
| 月之暗面   | `moonshot`   | `kimi`      |

完整 Provider 文档：[Provider 总览](../providers/overview.md)

---

## 对话方法

### provider.Stream

流式对话，实时返回响应块。

```go
func Stream(ctx context.Context, messages []types.Message, opts *StreamOptions) (<-chan StreamChunk, error)
```

**参数**：

- `ctx`: 上下文
- `messages`: 消息历史
- `opts`: 流式选项（工具、温度等）

**返回**：

- `<-chan StreamChunk`: 流式响应通道
- `error`: 创建流失败时返回错误

**示例**：

```go
messages := []types.Message{
    {
        Role:    types.RoleUser,
        Content: "用一句话介绍 Go 语言",
    },
}

stream, err := p.Stream(ctx, messages, &provider.StreamOptions{
    MaxTokens:   1000,
    Temperature: 0.7,
})
if err != nil {
    log.Fatal(err)
}

for chunk := range stream {
    switch chunk.Type {
    case "text":
        fmt.Print(chunk.TextDelta)
    case "tool_call":
        fmt.Printf("\n[工具调用] %s\n", chunk.ToolCall.Name)
    case "error":
        log.Printf("错误: %v", chunk.Error)
    }
}
```

**StreamChunk 类型**：

| Type        | 说明       | 字段                     |
| ----------- | ---------- | ------------------------ |
| `text`      | 文本增量   | `TextDelta`              |
| `tool_call` | 工具调用   | `ToolCall`, `ToolCallID` |
| `usage`     | Token 使用 | `Usage`                  |
| `error`     | 错误       | `Error`                  |

---

### provider.Complete

非流式对话，等待完整响应。

```go
func Complete(ctx context.Context, messages []types.Message, opts *StreamOptions) (*CompleteResponse, error)
```

**参数**：

- `ctx`: 上下文
- `messages`: 消息历史
- `opts`: 生成选项

**返回**：

- `*CompleteResponse`: 完整响应
- `error`: 错误信息

**示例**：

```go
messages := []types.Message{
    {
        Role:    types.RoleUser,
        Content: "编写一个 Hello World 程序",
    },
}

response, err := p.Complete(ctx, messages, &provider.StreamOptions{
    MaxTokens: 2000,
})
if err != nil {
    log.Fatal(err)
}

fmt.Println(response.Message.Content)
fmt.Printf("Token 使用: %d\n", response.Usage.TotalTokens)
```

**CompleteResponse 结构**：

```go
type CompleteResponse struct {
    Message    types.Message  // 完整响应消息
    StopReason string         // 停止原因
    Usage      TokenUsage     // Token 使用量
}
```

---

## 工具调用

### 定义工具

```go
tools := []provider.ToolSchema{
    {
        Name:        "get_weather",
        Description: "获取指定城市的天气信息",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "city": map[string]interface{}{
                    "type":        "string",
                    "description": "城市名称",
                },
                "unit": map[string]interface{}{
                    "type": "string",
                    "enum": []string{"celsius", "fahrenheit"},
                },
            },
            "required": []string{"city"},
        },
    },
}
```

### 使用工具

```go
messages := []types.Message{
    {Role: types.RoleUser, Content: "北京今天天气怎么样？"},
}

stream, _ := p.Stream(ctx, messages, &provider.StreamOptions{
    Tools:     tools,
    MaxTokens: 1000,
})

for chunk := range stream {
    if chunk.Type == "tool_call" {
        fmt.Printf("工具: %s\n", chunk.ToolCall.Name)
        fmt.Printf("参数: %s\n", chunk.ToolCall.Arguments)

        // 执行工具并返回结果
        result := executeWeatherTool(chunk.ToolCall.Arguments)

        messages = append(messages, types.Message{
            Role: types.RoleAssistant,
            ToolCalls: []types.ToolCall{*chunk.ToolCall},
        }, types.Message{
            Role: types.RoleUser,
            ToolResults: []types.ToolResult{
                {
                    ToolCallID: chunk.ToolCallID,
                    Content:    result,
                },
            },
        })

        // 继续对话
        stream, _ = p.Stream(ctx, messages, opts)
    }
}
```

---

## 多模态支持

### 图片输入

```go
messages := []types.Message{
    {
        Role: types.RoleUser,
        ContentBlocks: []types.ContentBlock{
            &types.TextBlock{
                Text: "这张图片里有什么？",
            },
            &types.ImageContent{
                Type:   "url",
                Source: "https://example.com/image.jpg",
                Detail: "high", // "low", "high", "auto"
            },
        },
    },
}

response, _ := p.Complete(ctx, messages, nil)
```

### Base64 图片

```go
imageData, _ := os.ReadFile("image.png")
base64Data := base64.StdEncoding.EncodeToString(imageData)

messages := []types.Message{
    {
        Role: types.RoleUser,
        ContentBlocks: []types.ContentBlock{
            &types.TextBlock{Text: "分析这张图片"},
            &types.ImageContent{
                Type:     "base64",
                Source:   base64Data,
                MimeType: "image/png",
            },
        },
    },
}
```

### 视频理解（Gemini 独有）

```go
messages := []types.Message{
    {
        Role: types.RoleUser,
        ContentBlocks: []types.ContentBlock{
            &types.TextBlock{Text: "总结视频内容"},
            &types.VideoContent{
                Type:     "url",
                Source:   "https://example.com/video.mp4",
                MimeType: "video/mp4",
            },
        },
    },
}

// 只有 Gemini Provider 支持视频
config := &types.ModelConfig{
    Provider: "gemini",
    Model:    "gemini-2.0-flash-exp",
    APIKey:   os.Getenv("GEMINI_API_KEY"),
}

p, _ := factory.Create(config)
response, _ := p.Complete(ctx, messages, &provider.StreamOptions{
    MaxTokens: 5000,
})
```

---

## 配置管理

### provider.Config

获取 Provider 配置。

```go
func (p Provider) Config() *types.ModelConfig
```

**示例**：

```go
config := p.Config()
fmt.Printf("Provider: %s\n", config.Provider)
fmt.Printf("Model: %s\n", config.Model)
```

---

### provider.Capabilities

获取 Provider 能力。

```go
func (p Provider) Capabilities() ProviderCapabilities
```

**示例**：

```go
caps := p.Capabilities()
fmt.Printf("支持流式: %v\n", caps.SupportsStreaming)
fmt.Printf("支持工具: %v\n", caps.SupportsTools)
fmt.Printf("支持视觉: %v\n", caps.SupportsVision)
fmt.Printf("支持音频: %v\n", caps.SupportsAudio)
fmt.Printf("支持视频: %v\n", caps.SupportsVideo)
fmt.Printf("上下文窗口: %d\n", caps.MaxContextWindow)
```

**ProviderCapabilities 结构**：

```go
type ProviderCapabilities struct {
    SupportsStreaming    bool   // 支持流式输出
    SupportsTools        bool   // 支持工具调用
    SupportsVision       bool   // 支持图片输入
    SupportsAudio        bool   // 支持音频输入
    SupportsVideo        bool   // 支持视频输入
    SupportsPromptCache  bool   // 支持 Prompt Caching
    MaxContextWindow     int    // 最大上下文窗口
    SupportedMediaTypes  []string // 支持的媒体类型
}
```

---

### provider.SetSystemPrompt / GetSystemPrompt

设置和获取系统提示词。

```go
func (p Provider) SetSystemPrompt(prompt string) error
func (p Provider) GetSystemPrompt() string
```

**示例**：

```go
// 设置系统提示词
err := p.SetSystemPrompt("你是一个专业的 Go 语言助手")
if err != nil {
    log.Fatal(err)
}

// 获取当前系统提示词
prompt := p.GetSystemPrompt()
fmt.Println(prompt)
```

---

### provider.Close

关闭 Provider，释放资源。

```go
func (p Provider) Close() error
```

**示例**：

```go
defer p.Close()

// 或
if err := p.Close(); err != nil {
    log.Printf("关闭 Provider 失败: %v", err)
}
```

---

## 类型定义

### ModelConfig

```go
type ModelConfig struct {
    // 基础配置
    Provider    string   // Provider 名称
    Model       string   // 模型名称
    APIKey      string   // API Key
    BaseURL     string   // 自定义端点（可选）

    // 执行模式
    ExecutionMode ExecutionMode // 执行模式：streaming/non-streaming/auto

    // 生成参数
    Temperature float64  // 温度 (0.0-1.0)
    MaxTokens   int      // 最大 Token 数
    TopP        float64  // Top-P 采样
    TopK        int      // Top-K 采样

    // 高级配置
    SystemPrompt string  // 系统提示词
    StopSequences []string // 停止序列
}
```

**ExecutionMode 类型** ：

```go
type ExecutionMode string

const (
    ExecutionModeStreaming    ExecutionMode = "streaming"     // 流式输出（默认）
    ExecutionModeNonStreaming ExecutionMode = "non-streaming" // 非流式批处理
    ExecutionModeAuto         ExecutionMode = "auto"          // 自动选择
)
```

**ExecutionMode 说明**：

| 模式           | 适用场景             | 性能特点                    |
| -------------- | -------------------- | --------------------------- |
| `Streaming`    | 交互式对话、实时反馈 | 用户体验好，逐字输出        |
| `NonStreaming` | 批量翻译、文档处理   | **速度快3-5倍**，一次性返回 |
| `Auto`         | 通用场景             | 自动根据任务类型选择        |

**使用示例**：

```go
// 批量翻译场景（快速模式）
config := &types.ModelConfig{
    Provider:      "deepseek",
    Model:         "deepseek-chat",
    APIKey:        os.Getenv("DEEPSEEK_API_KEY"),
    ExecutionMode: types.ExecutionModeNonStreaming, // 3-5倍速度提升
}

// 交互式对话（默认）
config := &types.ModelConfig{
    Provider:      "openai",
    Model:         "gpt-4o",
    ExecutionMode: types.ExecutionModeStreaming, // 或省略（默认）
}
```

---

### StreamOptions

```go
type StreamOptions struct {
    // 生成参数
    MaxTokens    int              // 最大 Token 数
    Temperature  float64          // 温度参数
    TopP         float64          // Top-P 采样
    TopK         int              // Top-K 采样

    // 工具配置
    Tools        []ToolSchema     // 工具列表
    ToolChoice   string           // 工具选择策略: auto, required, none

    // 停止条件
    StopSequences []string        // 停止序列

    // 高级选项
    Stream       bool             // 是否流式（内部使用）
    N            int              // 生成数量
    PresencePenalty  float64     // 出现惩罚
    FrequencyPenalty float64     // 频率惩罚
}
```

---

### TokenUsage

```go
type TokenUsage struct {
    InputTokens     int  // 输入 Token 数
    OutputTokens    int  // 输出 Token 数
    TotalTokens     int  // 总 Token 数

    // 缓存 Token（Anthropic/OpenAI）
    CacheCreationTokens int
    CacheReadTokens     int

    // 推理 Token（o1/o3/R1）
    ReasoningTokens int
}
```

---

### ToolSchema

```go
type ToolSchema struct {
    Name         string                 // 工具名称
    Description  string                 // 工具描述
    InputSchema  map[string]interface{} // JSON Schema
}
```

---

## 支持的 Providers

### 国际主流

**OpenAI**

```go
config := &types.ModelConfig{
    Provider: "openai",
    Model:    "gpt-4o",
    APIKey:   os.Getenv("OPENAI_API_KEY"),
}
```

**特性**：GPT-4/o1/o3、多模态、Prompt Caching、推理模型

---

**Anthropic**

```go
config := &types.ModelConfig{
    Provider: "anthropic",
    Model:    "claude-sonnet-4-5",
    APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
}
```

**特性**：Claude 系列、200K 上下文、Prompt Caching

---

**Gemini**

```go
config := &types.ModelConfig{
    Provider: "gemini",
    Model:    "gemini-2.0-flash-exp",
    APIKey:   os.Getenv("GEMINI_API_KEY"),
}
```

**特性**：1M-2M 上下文、视频理解、多模态

---

**Groq**

```go
config := &types.ModelConfig{
    Provider: "groq",
    Model:    "llama-3.3-70b-versatile",
    APIKey:   os.Getenv("GROQ_API_KEY"),
}
```

**特性**：超快推理速度、开源模型

---

### 中国市场

**DeepSeek**

```go
config := &types.ModelConfig{
    Provider: "deepseek",
    Model:    "deepseek-chat",
    APIKey:   os.Getenv("DEEPSEEK_API_KEY"),
}
```

**特性**：R1 推理模型、性价比高

---

**智谱 GLM**

```go
config := &types.ModelConfig{
    Provider: "glm",
    Model:    "glm-4",
    APIKey:   os.Getenv("ZHIPU_API_KEY"),
}
```

**特性**：ChatGLM 系列、中文优化

---

### 本地部署

**Ollama**

```go
config := &types.ModelConfig{
    Provider: "ollama",
    Model:    "llama3.2",
    BaseURL:  "http://localhost:11434/v1",
}
```

**特性**：本地部署、无需 API Key、隐私保护

---

## 最佳实践

### 1. 连接复用

复用 Provider 实例以提高性能：

```go
// ✅ 推荐
p, _ := factory.Create(config)
for i := 0; i < 100; i++ {
    response, _ := p.Complete(ctx, messages, nil)
}

// ❌ 避免
for i := 0; i < 100; i++ {
    p, _ := factory.Create(config)
    response, _ := p.Complete(ctx, messages, nil)
}
```

### 2. 错误处理

检查流式响应中的错误：

```go
for chunk := range stream {
    if chunk.Type == "error" {
        log.Printf("错误: %v", chunk.Error)
        break
    }
}
```

### 3. 上下文超时

为长时间运行的请求设置超时：

```go
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

response, err := p.Complete(ctx, messages, opts)
```

### 4. Token 限制

根据模型设置合理的 MaxTokens：

```go
opts := &provider.StreamOptions{
    MaxTokens: 4000, // GPT-4o 最大 128K
}
```

### 5. Prompt Caching

使用 Prompt Caching 降低成本（Anthropic/OpenAI）：

```go
messages := []types.Message{
    {
        Role:    types.RoleSystem,
        Content: longSystemPrompt, // 会被缓存
        CacheControl: &types.CacheControl{Type: "ephemeral"},
    },
    {Role: types.RoleUser, Content: userMessage},
}
```

---

## 相关资源

- [Agent API 文档](./1.agent-api.md)
- [Provider 总览](../providers/overview.md)
- [OpenAI Provider](../providers/openai.md)
- [Gemini Provider](../providers/gemini.md)
- [完整 API 文档 (pkg.go.dev)](https://pkg.go.dev/github.com/astercloud/aster/pkg/provider)
