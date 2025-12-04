---
title: Gemini Provider
description: 使用 Google Gemini API 的注意事项
navigation: false
---

# Gemini Provider

Google Gemini 是 Google 的先进 AI 模型系列，支持超长上下文（1M tokens）和独特的视频理解能力。

## 特性

- ✅ **超长上下文** - 高达 1M tokens (Gemini 2.0)
- ✅ **视频理解** - 唯一原生支持视频输入的 Provider
- ✅ **多模态** - 图片、音频、视频、文档
- ✅ **工具调用** - Function Calling 支持
- ✅ **流式输出** - 实时流式响应
- ✅ **Context Caching** - 节省长上下文成本（最低 32K tokens）
- ✅ **推理模型** - Gemini 2.0 Flash Thinking 实验版

## 配置

### 基础配置

```go
import (
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/types"
)

config := &types.ModelConfig{
	Provider: "gemini",
	Model:    "gemini-2.0-flash-exp",
	APIKey:   "your-api-key",
}

factory := provider.NewMultiProviderFactory()
provider, err := factory.Create(config)
```

### 支持的模型

| 模型                                 | 特点       | Context | 推荐场景   |
| ------------------------------------ | ---------- | ------- | ---------- |
| `gemini-2.0-flash-exp`               | 最新实验版 | 1M      | 通用场景   |
| `gemini-2.0-flash-thinking-exp-1219` | 推理模型   | 32K     | 复杂推理   |
| `gemini-1.5-pro`                     | 稳定高性能 | 2M      | 长文档分析 |
| `gemini-1.5-flash`                   | 快速响应   | 1M      | 实时应用   |
| `gemini-1.5-flash-8b`                | 成本优化   | 1M      | 简单任务   |

## 使用示例

### 1. 基础对话

```go
messages := []types.Message{
	{
		Role:    types.RoleUser,
		Content: "解释量子计算的基本原理",
	},
}

// 流式响应
stream, err := provider.Stream(ctx, messages, &provider.StreamOptions{
	MaxTokens:   2000,
	Temperature: 0.7,
})

for chunk := range stream {
	if chunk.Type == "text" {
		fmt.Print(chunk.TextDelta)
	}
}

// 或非流式响应
response, err := provider.Complete(ctx, messages, &provider.StreamOptions{
	MaxTokens:   2000,
	Temperature: 0.7,
})
fmt.Println(response.Message.Content)
```

### 2. 图片理解

```go
messages := []types.Message{
	{
		Role: types.RoleUser,
		ContentBlocks: []types.ContentBlock{
			&types.TextBlock{
				Text: "这张图片里有什么？请详细描述。",
			},
			&types.ImageContent{
				Type:     "url",
				Source:   "https://example.com/image.jpg",
				MimeType: "image/jpeg",
			},
		},
	},
}

response, err := provider.Complete(ctx, messages, nil)
fmt.Println(response.Message.Content)
```

### 3. 视频理解（Gemini 独有）

```go
messages := []types.Message{
	{
		Role: types.RoleUser,
		ContentBlocks: []types.ContentBlock{
			&types.TextBlock{
				Text: "总结这个视频的主要内容",
			},
			&types.VideoContent{
				Type:     "url",
				Source:   "https://example.com/video.mp4",
				MimeType: "video/mp4",
			},
		},
	},
}

response, err := provider.Complete(ctx, messages, &provider.StreamOptions{
	MaxTokens: 5000, // 视频分析需要更多 tokens
})
```

### 4. Base64 内联数据

```go
import (
	"encoding/base64"
	"os"
)

// 读取本地图片
imageData, _ := os.ReadFile("image.jpg")
base64Data := base64.StdEncoding.EncodeToString(imageData)

messages := []types.Message{
	{
		Role: types.RoleUser,
		ContentBlocks: []types.ContentBlock{
			&types.TextBlock{Text: "分析这张图片"},
			&types.ImageContent{
				Type:     "base64",
				Source:   base64Data,
				MimeType: "image/jpeg",
			},
		},
	},
}
```

### 5. 工具调用

```go
tools := []provider.ToolSchema{
	{
		Name:        "search_web",
		Description: "搜索网络获取最新信息",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "搜索查询关键词",
				},
			},
			"required": []string{"query"},
		},
	},
}

messages := []types.Message{
	{
		Role:    types.RoleUser,
		Content: "2024年的诺贝尔物理学奖得主是谁？",
	},
}

stream, err := provider.Stream(ctx, messages, &provider.StreamOptions{
	Tools:     tools,
	MaxTokens: 1000,
})

for chunk := range stream {
	switch chunk.Type {
	case "text":
		fmt.Print(chunk.TextDelta)
	case "tool_call":
		fmt.Printf("\n[工具调用] %s\n", chunk.ToolCall.Name)
		fmt.Printf("[参数] %s\n", chunk.ToolCall.ArgumentsDelta)
	}
}
```

### 6. Context Caching（节省成本）

```go
// 设置长的系统提示词（会被缓存）
longPrompt := `你是一个专业的法律顾问助手。

你精通以下领域的法律：
1. 公司法
2. 合同法
3. 知识产权法
4. 劳动法
5. 民事诉讼法

... [很长的专业知识和指引] ...

总共超过 32K tokens 的内容会被自动缓存。`

provider.SetSystemPrompt(longPrompt)

// 首次调用会创建缓存
response1, _ := provider.Complete(ctx, messages, nil)
fmt.Printf("缓存创建 tokens: %d\n", response1.Usage.CacheCreationTokens)

// 5分钟内的后续调用会使用缓存（节省成本）
response2, _ := provider.Complete(ctx, messages2, nil)
fmt.Printf("缓存命中 tokens: %d\n", response2.Usage.CachedTokens)
```

### 7. 推理模型

```go
config := &types.ModelConfig{
	Provider: "gemini",
	Model:    "gemini-2.0-flash-thinking-exp-1219",
	APIKey:   "your-api-key",
}

provider, _ := factory.Create(config)

messages := []types.Message{
	{
		Role:    types.RoleUser,
		Content: "证明费马大定理的基本思路是什么？",
	},
}

// 推理模型会输出思考过程
stream, _ := provider.Stream(ctx, messages, &provider.StreamOptions{
	MaxTokens: 8000,
})

for chunk := range stream {
	switch chunk.Type {
	case "reasoning":
		fmt.Printf("[思考 %d] %s\n", chunk.Reasoning.Step, chunk.Reasoning.Thought)
	case "text":
		fmt.Print(chunk.TextDelta)
	}
}
```

## 高级配置

### 环境变量

```bash
# API Key
export GEMINI_API_KEY="your-api-key"
# 或
export GOOGLE_API_KEY="your-api-key"
```

### 代码配置

```go
config := &types.ModelConfig{
	Provider: "gemini",
	Model:    "gemini-2.0-flash-exp",
	APIKey:   os.Getenv("GEMINI_API_KEY"),
}
```

### 使用别名

```go
// "gemini" 和 "google" 都可以使用
config.Provider = "gemini"  // ✅
config.Provider = "google"  // ✅
```

## 定价说明

### 输入/输出 Token 定价（2024年）

| 模型                | 输入       | 输出     | 缓存输入  |
| ------------------- | ---------- | -------- | --------- |
| Gemini 2.0 Flash    | 免费\*     | 免费\*   | 免费\*    |
| Gemini 1.5 Pro      | $1.25/1M   | $5/1M    | $0.31/1M  |
| Gemini 1.5 Flash    | $0.075/1M  | $0.30/1M | $0.019/1M |
| Gemini 1.5 Flash-8B | $0.0375/1M | $0.15/1M | $0.01/1M  |

\* 2.0 Flash 实验版免费使用，但有速率限制

**Context Caching 可节省 75% 长上下文成本！**

## 最佳实践

### 1. 选择合适的模型

- **实验性功能**: 使用 `gemini-2.0-flash-exp`（免费）
- **生产环境**: 使用 `gemini-1.5-pro` 或 `gemini-1.5-flash`（稳定）
- **成本优化**: 使用 `gemini-1.5-flash-8b`（最便宜）
- **复杂推理**: 使用 `gemini-2.0-flash-thinking-exp-1219`

### 2. 利用超长上下文

```go
// Gemini 可以处理整本书的内容
longDocument := readEntireBook("book.txt") // 假设 500K tokens

messages := []types.Message{
	{Role: types.RoleUser, Content: longDocument},
	{Role: types.RoleUser, Content: "总结这本书的核心论点"},
}

response, _ := provider.Complete(ctx, messages, nil)
```

### 3. Context Caching 最佳实践

```go
// ✅ 好的做法：缓存超过 32K tokens 的内容
provider.SetSystemPrompt(longPrompt) // > 32K tokens

// ✅ 好的做法：5分钟内复用相同上下文
for i := 0; i < 10; i++ {
	response, _ := provider.Complete(ctx, messages, nil)
	// 后续 9 次调用都会使用缓存
}

// ❌ 避免：频繁更改上下文
messages[0].Content = "slightly different content" // 缓存失效
```

### 4. 视频分析优化

```go
// 对于长视频，建议：
opts := &provider.StreamOptions{
	MaxTokens:   10000,  // 预留足够的输出空间
	Temperature: 0.4,    // 降低随机性，提高分析准确性
}

// 提供明确的分析任务
message := types.Message{
	Role: types.RoleUser,
	ContentBlocks: []types.ContentBlock{
		&types.TextBlock{
			Text: `分析这个视频，重点关注：
1. 主要人物和事件
2. 关键场景和转折点
3. 整体主题和信息`,
		},
		&types.VideoContent{
			Type:     "url",
			Source:   videoURL,
			MimeType: "video/mp4",
		},
	},
}
```

### 5. 错误处理

```go
response, err := provider.Complete(ctx, messages, opts)
if err != nil {
	// API Key 错误
	if strings.Contains(err.Error(), "API_KEY_INVALID") {
		log.Fatal("Invalid API key")
	}

	// 配额超限
	if strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
		// 使用付费模型或等待配额重置
	}

	// 内容过长
	if strings.Contains(err.Error(), "exceeds") {
		// 减少输入长度
	}

	return err
}
```

## 独特优势

### 1. 超长上下文窗口

- **Gemini 1.5 Pro**: 2M tokens（最长）
- **Gemini 2.0/1.5 Flash**: 1M tokens
- 可以处理：
  - 完整的代码仓库
  - 长篇小说
  - 多小时的视频转录
  - 大量文档集合

### 2. 原生视频理解

Gemini 是唯一原生支持视频输入的主流 Provider：

```go
// 其他 Provider 需要手动提取帧
// Gemini 直接理解视频流
&types.VideoContent{
	Type:     "url",
	Source:   "video.mp4",
	MimeType: "video/mp4",
}
```

### 3. 成本效益

- 实验版免费使用
- Context Caching 节省 75% 成本
- Flash-8B 模型超低价格

## 限制说明

### 1. 实验版限制

- 速率限制较严格（RPM: 10, TPM: 40K）
- 不保证稳定性
- 不适合生产环境

### 2. Context Caching 要求

- 最少 32K tokens 才会缓存
- 缓存 5 分钟后过期
- 不支持所有模型

### 3. 视频处理限制

- 视频时长建议 < 1 小时
- 文件大小限制
- 处理时间较长

## 常见问题

### Q: 如何获取 API Key？

访问 [Google AI Studio](https://makersuite.google.com/app/apikey) 创建免费 API Key。

### Q: 实验版和稳定版如何选择？

- **开发/测试**: 使用免费的实验版
- **生产环境**: 使用稳定的 1.5 系列
- **性能测试**: 先用实验版验证，再切换到稳定版

### Q: Context Caching 如何计费？

缓存的 tokens 按正常输入价格的 25% 计费。例如 Gemini 1.5 Pro：

- 正常输入: $1.25/1M tokens
- 缓存输入: $0.31/1M tokens（节省 75%）

### Q: 视频分析支持哪些格式？

支持常见格式：MP4, MOV, AVI, FLV, MKV, WebM 等。建议使用 MP4 格式。

### Q: 推理模型和普通模型有什么区别？

推理模型（Thinking）会输出思考过程（reasoning traces），适合需要展示推理步骤的场景。

## 相关链接

- [Google AI 文档](https://ai.google.dev/docs)
- [Gemini API 参考](https://ai.google.dev/api)
- [定价页面](https://ai.google.dev/pricing)
- [Context Caching 指南](https://ai.google.dev/docs/caching)
