---
title: OpenAI Provider
description: 使用 OpenAI 及兼容 API 的统一配置方式
navigation: false
---

# OpenAI Provider

OpenAI 是最流行的 AI 提供商，支持 GPT-4/4.5/5、GPT-4o 以及 o1/o3 推理模型。

## 特性

- ✅ **流式输出** - 实时流式响应
- ✅ **工具调用** - Function Calling 支持
- ✅ **多模态** - 图片和音频输入
- ✅ **推理模型** - o1/o3 推理模型支持
- ✅ **Prompt Caching** - 节省 Token 成本

## 配置

### 基础配置

```go
import (
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/types"
)

config := &types.ModelConfig{
	Provider: "openai",
	Model:    "gpt-4o",
	APIKey:   "sk-your-api-key",
}

factory := provider.NewMultiProviderFactory()
provider, err := factory.Create(config)
```

### 支持的模型

| 模型          | 特点       | Context | 推荐场景 |
| ------------- | ---------- | ------- | -------- |
| `gpt-4o`      | 最新多模态 | 128K    | 通用场景 |
| `gpt-4o-mini` | 成本优化   | 128K    | 简单任务 |
| `gpt-4-turbo` | 高性能     | 128K    | 复杂推理 |
| `o1-preview`  | 推理模型   | 100K    | 复杂问题 |
| `o1-mini`     | 快速推理   | 100K    | 代码生成 |
| `o3-mini`     | 最新推理   | 100K    | 科学计算 |

## 使用示例

### 1. 基础对话

```go
messages := []types.Message{
	{
		Role:    types.RoleUser,
		Content: "介绍一下 Go 语言",
	},
}

// 流式响应
stream, err := provider.Stream(ctx, messages, &provider.StreamOptions{
	MaxTokens:   1000,
	Temperature: 0.7,
})

for chunk := range stream {
	if chunk.Type == "text" {
		fmt.Print(chunk.TextDelta)
	}
}

// 或非流式响应
response, err := provider.Complete(ctx, messages, &provider.StreamOptions{
	MaxTokens:   1000,
	Temperature: 0.7,
})
fmt.Println(response.Message.Content)
```

### 2. 多模态输入（图片）

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

response, err := provider.Complete(ctx, messages, nil)
```

### 3. 工具调用

```go
tools := []provider.ToolSchema{
	{
		Name:        "get_weather",
		Description: "获取指定城市的天气",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"city": map[string]interface{}{
					"type":        "string",
					"description": "城市名称",
				},
			},
			"required": []string{"city"},
		},
	},
}

stream, err := provider.Stream(ctx, messages, &provider.StreamOptions{
	Tools:     tools,
	MaxTokens: 1000,
})

for chunk := range stream {
	if chunk.Type == "tool_call" {
		fmt.Printf("调用工具: %s\n", chunk.ToolCall.Name)
		fmt.Printf("参数: %s\n", chunk.ToolCall.ArgumentsDelta)
	}
}
```

### 4. 推理模型 (o1/o3)

```go
config := &types.ModelConfig{
	Provider: "openai",
	Model:    "o1-preview", // 或 o1-mini, o3-mini
	APIKey:   "sk-your-api-key",
}

// 注意：推理模型不支持 temperature 和工具调用
response, err := provider.Complete(ctx, messages, &provider.StreamOptions{
	MaxTokens: 5000, // 推理模型需要更多 tokens
})

// 查看推理过程的 token 使用
fmt.Printf("推理 tokens: %d\n", response.Usage.ReasoningTokens)
```

### 5. Prompt Caching（节省成本）

OpenAI 自动缓存 system prompt 和长上下文：

```go
// 设置系统提示词（会被缓存）
provider.SetSystemPrompt(`你是一个专业的编程助手。
你精通 Go、Python、JavaScript 等多种编程语言。
你总是提供清晰、简洁、可运行的代码示例。`)

// 首次调用会创建缓存
response1, _ := provider.Complete(ctx, messages, nil)
fmt.Printf("缓存创建 tokens: %d\n", response1.Usage.CacheCreationTokens)

// 后续调用会使用缓存（节省成本）
response2, _ := provider.Complete(ctx, messages2, nil)
fmt.Printf("缓存命中 tokens: %d\n", response2.Usage.CachedTokens)
```

## 高级配置

### 环境变量

```bash
# API Key
export OPENAI_API_KEY="sk-your-api-key"

# 自定义 Base URL（用于代理）
export OPENAI_BASE_URL="https://api.openai-proxy.com/v1"
```

### 代码配置

```go
config := &types.ModelConfig{
	Provider: "openai",
	Model:    "gpt-4o",
	APIKey:   os.Getenv("OPENAI_API_KEY"),
	BaseURL:  "https://api.openai-proxy.com/v1", // 可选
}
```

## 定价说明

### 输入/输出 Token 定价（2024年）

| 模型        | 输入     | 输出    | 缓存输入  |
| ----------- | -------- | ------- | --------- |
| GPT-4o      | $2.5/1M  | $10/1M  | $1.25/1M  |
| GPT-4o-mini | $0.15/1M | $0.6/1M | $0.075/1M |
| o1-preview  | $15/1M   | $60/1M  | -         |
| o1-mini     | $3/1M    | $12/1M  | -         |

**Prompt Caching 可节省 50% 输入成本！**

## 最佳实践

### 1. 选择合适的模型

- **简单任务**: 使用 `gpt-4o-mini`（成本低）
- **通用场景**: 使用 `gpt-4o`（性价比高）
- **复杂推理**: 使用 `o1-preview`（推理能力强）
- **代码生成**: 使用 `o1-mini`（快速且准确）

### 2. 优化 Token 使用

```go
// ✅ 好的做法：使用 Prompt Caching
provider.SetSystemPrompt(longSystemPrompt)

// ✅ 好的做法：限制 MaxTokens
opts := &provider.StreamOptions{
	MaxTokens: 500, // 根据需求设置合理值
}

// ❌ 避免：每次都传递长 system prompt
messages := []types.Message{
	{Role: types.RoleSystem, Content: longSystemPrompt}, // 不会被缓存
	{Role: types.RoleUser, Content: "..."},
}
```

### 3. 错误处理

```go
response, err := provider.Complete(ctx, messages, opts)
if err != nil {
	// Rate limit 错误
	if strings.Contains(err.Error(), "429") {
		time.Sleep(time.Second)
		// 重试
	}

	// Token 超限错误
	if strings.Contains(err.Error(), "maximum context length") {
		// 减少上下文长度
	}

	return err
}
```

### 4. 并发控制

```go
// 使用 context 控制超时
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

response, err := provider.Complete(ctx, messages, opts)
```

## 常见问题

### Q: 如何使用 Azure OpenAI？

```go
config := &types.ModelConfig{
	Provider: "openai",
	Model:    "gpt-4o",
	APIKey:   "your-azure-api-key",
	BaseURL:  "https://your-resource.openai.azure.com/openai/deployments/gpt-4o",
}
```

### Q: 推理模型为什么不支持 temperature？

推理模型（o1/o3）使用固定的推理策略，不支持 temperature 参数。

### Q: 如何查看详细的 token 使用情况？

```go
response, _ := provider.Complete(ctx, messages, opts)
usage := response.Usage

fmt.Printf("输入 tokens: %d\n", usage.InputTokens)
fmt.Printf("输出 tokens: %d\n", usage.OutputTokens)
fmt.Printf("推理 tokens: %d\n", usage.ReasoningTokens) // 仅推理模型
fmt.Printf("缓存 tokens: %d\n", usage.CachedTokens)   // Prompt Caching
```

## 相关链接

- [OpenAI API 文档](https://platform.openai.com/docs)
- [定价页面](https://openai.com/pricing)
- [推理模型指南](https://platform.openai.com/docs/guides/reasoning)
