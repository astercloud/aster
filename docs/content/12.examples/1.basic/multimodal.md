---
title: 多模态输入
description: 使用图片、音频等多模态内容
---

# 多模态输入示例

展示如何向 Agent 输入图片、音频等多模态内容。

## 图片 URL 输入

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/astercloud/aster/pkg/provider"
    "github.com/astercloud/aster/pkg/types"
)

func main() {
    ctx := context.Background()

    // 创建 Provider
    factory := provider.NewMultiProviderFactory()
    p, _ := factory.Create(&types.ModelConfig{
        Provider: "openai",
        Model:    "gpt-4o",
        APIKey:   os.Getenv("OPENAI_API_KEY"),
    })
    defer p.Close()

    // 构造多模态消息
    messages := []types.Message{
        {
            Role: types.RoleUser,
            ContentBlocks: []types.ContentBlock{
                &types.TextBlock{
                    Text: "这张图片里有什么？请详细描述。",
                },
                &types.ImageContent{
                    Type:   "url",
                    Source: "https://upload.wikimedia.org/wikipedia/commons/thumb/0/05/Go_Logo_Blue.svg/1200px-Go_Logo_Blue.svg.png",
                    Detail: "high",  // "low", "high", "auto"
                },
            },
        },
    }

    response, _ := p.Complete(ctx, messages, nil)
    fmt.Println(response.Message.Content)
}
```

## Base64 图片输入

```go
import (
    "encoding/base64"
    "os"
)

func main() {
    // 读取本地图片
    imageData, _ := os.ReadFile("screenshot.png")
    base64Data := base64.StdEncoding.EncodeToString(imageData)

    messages := []types.Message{
        {
            Role: types.RoleUser,
            ContentBlocks: []types.ContentBlock{
                &types.TextBlock{Text: "分析这个截图"},
                &types.ImageContent{
                    Type:     "base64",
                    Source:   base64Data,
                    MimeType: "image/png",
                },
            },
        },
    }

    response, _ := p.Complete(ctx, messages, nil)
    fmt.Println(response.Message.Content)
}
```

## 视频理解（Gemini）

```go
func main() {
    // 只有 Gemini 支持视频
    p, _ := factory.Create(&types.ModelConfig{
        Provider: "gemini",
        Model:    "gemini-2.0-flash-exp",
        APIKey:   os.Getenv("GEMINI_API_KEY"),
    })

    messages := []types.Message{
        {
            Role: types.RoleUser,
            ContentBlocks: []types.ContentBlock{
                &types.TextBlock{Text: "总结这个视频的主要内容"},
                &types.VideoContent{
                    Type:     "url",
                    Source:   "https://example.com/demo.mp4",
                    MimeType: "video/mp4",
                },
            },
        },
    }

    response, _ := p.Complete(ctx, messages, &provider.StreamOptions{
        MaxTokens: 5000,  // 视频分析需要更多 tokens
    })

    fmt.Println(response.Message.Content)
}
```

## 多张图片

```go
messages := []types.Message{
    {
        Role: types.RoleUser,
        ContentBlocks: []types.ContentBlock{
            &types.TextBlock{Text: "比较这两张图片的差异"},
            &types.ImageContent{
                Type:   "url",
                Source: "https://example.com/image1.jpg",
            },
            &types.ImageContent{
                Type:   "url",
                Source: "https://example.com/image2.jpg",
            },
        },
    },
}
```

## 支持情况

| Provider  | 图片 | 音频 | 视频 |
| --------- | ---- | ---- | ---- |
| OpenAI    | ✅   | ✅   | ❌   |
| Anthropic | ✅   | ❌   | ❌   |
| Gemini    | ✅   | ✅   | ✅   |
| Groq      | ❌   | ❌   | ❌   |
| DeepSeek  | ❌   | ❌   | ❌   |

## 相关资源

- [Provider API - 多模态](../../api-reference/provider-api#多模态支持)
- [Gemini Provider](../../providers/gemini)
