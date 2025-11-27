# Provider 使用示例

本目录包含各个 Provider 的使用示例。

## 快速开始

### 1. OpenAI

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/types"
)

func main() {
	config := &types.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   os.Getenv("OPENAI_API_KEY"),
	}

	factory := provider.NewMultiProviderFactory()
	p, err := factory.Create(config)
	if err != nil {
		panic(err)
	}

	messages := []types.Message{
		{
			Role:    types.RoleUser,
			Content: "用一句话介绍 Go 语言",
		},
	}

	// 流式响应
	stream, _ := p.Stream(context.Background(), messages, &provider.StreamOptions{
		MaxTokens: 1000,
	})

	for chunk := range stream {
		if chunk.Type == "text" {
			fmt.Print(chunk.TextDelta)
		}
	}
	fmt.Println()
}
```

### 2. Groq（超快速度）

```go
func main() {
	config := &types.ModelConfig{
		Provider: "groq",
		Model:    "llama-3.3-70b-versatile",
		APIKey:   os.Getenv("GROQ_API_KEY"),
	}

	factory := provider.NewMultiProviderFactory()
	p, _ := factory.Create(config)

	messages := []types.Message{
		{Role: types.RoleUser, Content: "编写一个 Go 的 Hello World"},
	}

	response, _ := p.Complete(context.Background(), messages, nil)
	fmt.Println(response.Message.Content)
}
```

### 3. Ollama（本地部署）

```go
func main() {
	// 确保 Ollama 服务运行中：ollama serve
	config := &types.ModelConfig{
		Provider: "ollama",
		Model:    "llama3.2",
		BaseURL:  "http://localhost:11434/v1",
	}

	factory := provider.NewMultiProviderFactory()
	p, _ := factory.Create(config)

	messages := []types.Message{
		{Role: types.RoleUser, Content: "你好！"},
	}

	response, _ := p.Complete(context.Background(), messages, nil)
	fmt.Println(response.Message.Content)
}
```

### 4. OpenRouter（多模型支持）

```go
func main() {
	config := &types.ModelConfig{
		Provider: "openrouter",
		Model:    "openai/gpt-4o", // 可切换任何模型
		APIKey:   os.Getenv("OPENROUTER_API_KEY"),
	}

	factory := provider.NewMultiProviderFactory()
	p, _ := factory.Create(config)

	// 轻松切换模型
	models := []string{
		"openai/gpt-4o",
		"anthropic/claude-3-opus",
		"google/gemini-pro",
		"meta-llama/llama-3-70b",
	}

	for _, model := range models {
		config.Model = model
		p, _ := factory.Create(config)

		messages := []types.Message{
			{Role: types.RoleUser, Content: "你好！"},
		}

		response, _ := p.Complete(context.Background(), messages, nil)
		fmt.Printf("[%s] %s\n", model, response.Message.Content)
	}
}
```

**完整示例**：查看 [`examples/openrouter/`](../openrouter/) 获取包含工具调用验证的完整集成测试示例。

### 5. 多模态输入（图片）

```go
func main() {
	config := &types.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   os.Getenv("OPENAI_API_KEY"),
	}

	factory := provider.NewMultiProviderFactory()
	p, _ := factory.Create(config)

	messages := []types.Message{
		{
			Role: types.RoleUser,
			ContentBlocks: []types.ContentBlock{
				&types.TextBlock{
					Text: "这张图片里有什么？详细描述一下。",
				},
				&types.ImageContent{
					Type:   "url",
					Source: "https://upload.wikimedia.org/wikipedia/commons/thumb/0/05/Go_Logo_Blue.svg/1200px-Go_Logo_Blue.svg.png",
					Detail: "high",
				},
			},
		},
	}

	response, _ := p.Complete(context.Background(), messages, nil)
	fmt.Println(response.Message.Content)
}
```

### 6. 视频理解（Gemini 独有）

```go
func main() {
	config := &types.ModelConfig{
		Provider: "gemini",
		Model:    "gemini-2.0-flash-exp",
		APIKey:   os.Getenv("GEMINI_API_KEY"),
	}

	factory := provider.NewMultiProviderFactory()
	p, _ := factory.Create(config)

	messages := []types.Message{
		{
			Role: types.RoleUser,
			ContentBlocks: []types.ContentBlock{
				&types.TextBlock{
					Text: "总结这个视频的主要内容和关键点",
				},
				&types.VideoContent{
					Type:     "url",
					Source:   "https://example.com/demo.mp4",
					MimeType: "video/mp4",
				},
			},
		},
	}

	response, _ := p.Complete(context.Background(), messages, &provider.StreamOptions{
		MaxTokens: 5000, // 视频分析需要更多 tokens
	})
	fmt.Println(response.Message.Content)
}
```

### 7. 工具调用示例

```go
func main() {
	config := &types.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   os.Getenv("OPENAI_API_KEY"),
	}

	factory := provider.NewMultiProviderFactory()
	p, _ := factory.Create(config)

	tools := []provider.ToolSchema{
		{
			Name:        "get_weather",
			Description: "获取指定城市的实时天气信息",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"city": map[string]interface{}{
						"type":        "string",
						"description": "城市名称，例如：北京、上海",
					},
					"unit": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"celsius", "fahrenheit"},
						"description": "温度单位",
					},
				},
				"required": []string{"city"},
			},
		},
	}

	messages := []types.Message{
		{
			Role:    types.RoleUser,
			Content: "北京今天天气怎么样？",
		},
	}

	stream, _ := p.Stream(context.Background(), messages, &provider.StreamOptions{
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
}
```

### 7. 推理模型示例

```go
func main() {
	// OpenAI o1-preview
	config := &types.ModelConfig{
		Provider: "openai",
		Model:    "o1-preview",
		APIKey:   os.Getenv("OPENAI_API_KEY"),
	}

	factory := provider.NewMultiProviderFactory()
	p, _ := factory.Create(config)

	messages := []types.Message{
		{
			Role:    types.RoleUser,
			Content: "计算斐波那契数列第100项模1000000007的值",
		},
	}

	// 推理模型不支持 temperature 和 streaming（某些情况）
	response, _ := p.Complete(context.Background(), messages, &provider.StreamOptions{
		MaxTokens: 5000, // 推理模型需要更多 tokens
	})

	fmt.Println(response.Message.Content)
	fmt.Printf("\n推理 tokens: %d\n", response.Usage.ReasoningTokens)
	fmt.Printf("总 tokens: %d\n", response.Usage.InputTokens+response.Usage.OutputTokens)
}
```

### 8. 中国市场 Providers

```go
func main() {
	providers := []struct {
		name     string
		provider string
		model    string
	}{
		{"DeepSeek", "deepseek", "deepseek-chat"},
		{"智谱 GLM", "glm", "glm-4"},
		{"豆包", "doubao", "ep-20240101-xxxxx"}, // 替换为实际 endpoint_id
		{"月之暗面", "moonshot", "moonshot-v1-128k"},
	}

	for _, cfg := range providers {
		config := &types.ModelConfig{
			Provider: cfg.provider,
			Model:    cfg.model,
			APIKey:   os.Getenv(strings.ToUpper(cfg.provider) + "_API_KEY"),
		}

		factory := provider.NewMultiProviderFactory()
		p, _ := factory.Create(config)

		messages := []types.Message{
			{Role: types.RoleUser, Content: "介绍一下你自己"},
		}

		response, _ := p.Complete(context.Background(), messages, nil)
		fmt.Printf("[%s]\n%s\n\n", cfg.name, response.Message.Content)
	}
}
```

## 运行示例

```bash
# 设置 API Keys
export OPENAI_API_KEY="sk-xxx"
export GEMINI_API_KEY="your-key"
export GROQ_API_KEY="gsk-xxx"
export OPENROUTER_API_KEY="sk-or-xxx"

# 运行示例
go run examples/providers/basic.go
go run examples/providers/streaming.go
go run examples/providers/multimodal.go
go run examples/providers/video.go          # Gemini 视频理解
go run examples/providers/tools.go
```

## 环境配置

创建 `.env` 文件：

```bash
# 国际 Providers
OPENAI_API_KEY=sk-xxx
ANTHROPIC_API_KEY=sk-ant-xxx
GEMINI_API_KEY=your-key
GROQ_API_KEY=gsk-xxx
OPENROUTER_API_KEY=sk-or-xxx
MISTRAL_API_KEY=xxx

# 中国 Providers
DEEPSEEK_API_KEY=sk-xxx
ZHIPU_API_KEY=xxx
DOUBAO_API_KEY=xxx
MOONSHOT_API_KEY=sk-xxx

# Ollama
OLLAMA_BASE_URL=http://localhost:11434/v1
```

然后使用 `godotenv` 加载：

```go
import "github.com/joho/godotenv"

func init() {
	godotenv.Load()
}
```

## 完整示例

查看 `examples/providers/` 目录下的完整示例：

- `basic.go` - 基础使用
- `streaming.go` - 流式响应
- `multimodal.go` - 多模态输入
- `tools.go` - 工具调用
- `reasoning.go` - 推理模型
- `comparison.go` - 多 Provider 对比

## 更多资源

- [Provider 文档](../../docs/content/3.providers/)
- [API 参考](https://pkg.go.dev/github.com/astercloud/aster/pkg/provider)
- [Agent 集成示例](../agent/)
