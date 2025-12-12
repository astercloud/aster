//go:build ignore

package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/types"
)

func main() {
	apiKey := os.Getenv("CLAUDE_API_KEY")
	baseURL := os.Getenv("CLAUDE_BASE_URL")

	if apiKey == "" {
		fmt.Println("❌ 请设置 CLAUDE_API_KEY 环境变量")
		os.Exit(1)
	}
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	fmt.Println("=== Claude API 图片识别测试 ===")
	fmt.Printf("API 端点: %s\n", baseURL)
	fmt.Printf("模型: claude-sonnet-4-5-20250929\n\n")

	// 下载测试图片
	fmt.Println("📥 下载测试图片...")
	imageData, mediaType, err := downloadTestImage()
	if err != nil {
		fmt.Printf("❌ 下载失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ 下载成功 (类型: %s, 大小: %d bytes)\n\n", mediaType, len(imageData))

	// 创建 Provider
	config := &types.ModelConfig{
		Provider: "anthropic",
		Model:    "claude-sonnet-4-5-20250929",
		APIKey:   apiKey,
		BaseURL:  baseURL,
	}

	cp, err := provider.NewCustomClaudeProvider(config)
	if err != nil {
		fmt.Printf("❌ 创建 Provider 失败: %v\n", err)
		os.Exit(1)
	}

	// 测试 1: 基础图片识别
	fmt.Println("【测试 1】基础图片识别")
	fmt.Println("----------------------------------------")
	testBasicVision(cp, imageData, mediaType)

	// 测试 2: 详细分析
	fmt.Println("\n【测试 2】详细图片分析")
	fmt.Println("----------------------------------------")
	testDetailedVision(cp, imageData, mediaType)

	fmt.Println("\n✅ 图片识别测试完成！")
}

// downloadTestImage 下载测试图片
func downloadTestImage() ([]byte, string, error) {
	// 使用 GitHub 的公开图片作为测试
	imageURL := "https://avatars.githubusercontent.com/u/1?v=4"

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(imageURL)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	mediaType := resp.Header.Get("Content-Type")
	if mediaType == "" {
		mediaType = "image/png"
	}

	return data, mediaType, nil
}

// testBasicVision 测试基础图片识别
func testBasicVision(cp *provider.CustomClaudeProvider, imageData []byte, mediaType string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Base64 编码
	base64Data := base64.StdEncoding.EncodeToString(imageData)

	// 构造消息
	messages := []types.Message{
		{
			Role: types.MessageRoleUser,
			ContentBlocks: []types.ContentBlock{
				&types.ImageContent{
					Type:     "base64",
					Source:   base64Data,
					MimeType: mediaType,
				},
				&types.TextBlock{
					Text: "这张图片里有什么？请用中文简短描述（不超过30字）。",
				},
			},
		},
	}

	fmt.Println("💬 问题: 这张图片里有什么？")
	fmt.Print("🤖 回复: ")

	// 调用 Provider
	opts := &provider.StreamOptions{
		MaxTokens: 500,
	}
	response, err := cp.Complete(ctx, messages, opts)
	if err != nil {
		fmt.Printf("\n❌ 调用失败: %v\n", err)
		return
	}

	// 输出结果
	var content string
	for _, block := range response.Message.ContentBlocks {
		if textBlock, ok := block.(*types.TextBlock); ok {
			content += textBlock.Text
		}
	}
	fmt.Printf("%s\n", content)
	if response.Usage != nil {
		fmt.Printf("📊 Token: 输入=%d, 输出=%d\n",
			response.Usage.InputTokens, response.Usage.OutputTokens)
	}
}

// testDetailedVision 测试详细图片分析
func testDetailedVision(cp *provider.CustomClaudeProvider, imageData []byte, mediaType string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Base64 编码
	base64Data := base64.StdEncoding.EncodeToString(imageData)

	// 构造消息
	messages := []types.Message{
		{
			Role: types.MessageRoleUser,
			ContentBlocks: []types.ContentBlock{
				&types.ImageContent{
					Type:     "base64",
					Source:   base64Data,
					MimeType: mediaType,
				},
				&types.TextBlock{
					Text: "请详细描述这张图片的：1) 主要内容 2) 颜色 3) 风格。用中文回答，不超过100字。",
				},
			},
		},
	}

	fmt.Println("💬 问题: 请详细描述图片的内容、颜色和风格")
	fmt.Print("🤖 回复: ")

	// 调用
	opts := &provider.StreamOptions{
		MaxTokens: 1000,
	}
	response, err := cp.Complete(ctx, messages, opts)
	if err != nil {
		fmt.Printf("\n❌ 调用失败: %v\n", err)
		return
	}

	// 输出结果
	var content string
	for _, block := range response.Message.ContentBlocks {
		if textBlock, ok := block.(*types.TextBlock); ok {
			content += textBlock.Text
		}
	}
	fmt.Printf("%s\n", content)
	if response.Usage != nil {
		fmt.Printf("📊 Token: 输入=%d, 输出=%d\n",
			response.Usage.InputTokens, response.Usage.OutputTokens)
	}
}
