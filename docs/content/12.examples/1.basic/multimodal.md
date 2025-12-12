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

## Claude Vision (完整示例)

```go
package main

import (
    "context"
    "encoding/base64"
    "fmt"
    "io"
    "net/http"
    "os"

    "github.com/astercloud/aster/pkg/provider"
    "github.com/astercloud/aster/pkg/types"
)

func main() {
    ctx := context.Background()

    // 创建 Claude Provider
    cp, err := provider.NewCustomClaudeProvider(&types.ModelConfig{
        Provider: "anthropic",
        Model:    "claude-sonnet-4-5-20250929",
        APIKey:   os.Getenv("CLAUDE_API_KEY"),
        BaseURL:  os.Getenv("CLAUDE_BASE_URL"), // 支持中继服务
    })
    if err != nil {
        panic(err)
    }
    defer cp.Close()

    // 方式 1: 从 URL 下载并转 Base64
    imageURL := "https://avatars.githubusercontent.com/u/1?v=4"
    resp, _ := http.Get(imageURL)
    defer resp.Body.Close()

    imageData, _ := io.ReadAll(resp.Body)
    base64Data := base64.StdEncoding.EncodeToString(imageData)

    // 构造多模态消息
    messages := []types.Message{
        {
            Role: types.MessageRoleUser,
            ContentBlocks: []types.ContentBlock{
                &types.ImageContent{
                    Type:     "base64",
                    Source:   base64Data,
                    MimeType: "image/png",
                },
                &types.TextBlock{
                    Text: "这张图片里有什么？请详细描述。",
                },
            },
        },
    }

    // 调用 Vision API
    opts := &provider.StreamOptions{MaxTokens: 500}
    response, err := cp.Complete(ctx, messages, opts)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Claude Vision 识别结果:\n%s\n", response.Message.Content)

    // 方式 2: 直接使用本地图片
    localImage, _ := os.ReadFile("screenshot.png")
    base64Local := base64.StdEncoding.EncodeToString(localImage)

    messages2 := []types.Message{
        {
            Role: types.MessageRoleUser,
            ContentBlocks: []types.ContentBlock{
                &types.ImageContent{
                    Type:     "base64",
                    Source:   base64Local,
                    MimeType: "image/png",
                },
                &types.TextBlock{
                    Text: "分析这个截图，提取其中的文字和主要元素。",
                },
            },
        },
    }

    response2, _ := cp.Complete(ctx, messages2, opts)
    fmt.Printf("\n本地图片分析:\n%s\n", response2.Message.Content)
}
```

**测试结果示例:**
```
Claude Vision 识别结果:
这张图片是 GitHub 用户头像的默认图标。图片呈现为一个圆形的标识...
具体特征包括：
1. 圆形的外轮廓
2. 深色调为主
3. GitHub 的默认头像设计
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
