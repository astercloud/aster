package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/astercloud/aster/pkg/tools"
)

// WebFetchTool 网页获取工具
type WebFetchTool struct {
	defaultTimeout time.Duration
	client         *http.Client
}

// NewWebFetchTool 创建 WebFetch 工具
func NewWebFetchTool(config map[string]any) (tools.Tool, error) {
	timeout := 30 * time.Second
	if t, ok := config["timeout"].(float64); ok {
		timeout = time.Duration(t) * time.Second
	}

	return &WebFetchTool{
		defaultTimeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (t *WebFetchTool) Name() string {
	return "WebFetch"
}

func (t *WebFetchTool) Description() string {
	return "获取和分析网页内容"
}

func (t *WebFetchTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "目标 URL（必须以 http:// 或 https:// 开头）",
			},
			"method": map[string]any{
				"type":        "string",
				"enum":        []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD"},
				"description": "HTTP 方法（默认: GET）",
			},
			"headers": map[string]any{
				"type":        "object",
				"description": "HTTP 请求头（键值对）",
			},
			"body": map[string]any{
				"type":        "string",
				"description": "请求体（用于 POST/PUT/PATCH）",
			},
			"timeout": map[string]any{
				"type":        "number",
				"description": "请求超时时间（秒），默认 30",
			},
		},
		"required": []string{"url"},
	}
}


func (t *WebFetchTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	url, ok := input["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("url must be a non-empty string")
	}

	method := "GET"
	if m, ok := input["method"].(string); ok {
		method = m
	}

	var reqBody io.Reader
	if bodyStr, ok := input["body"].(string); ok && bodyStr != "" {
		reqBody = bytes.NewBufferString(bodyStr)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return map[string]any{
			"success": false,
			"error":   fmt.Sprintf("failed to create request: %v", err),
		}, nil
	}

	if headers, ok := input["headers"].(map[string]any); ok {
		for key, value := range headers {
			if valueStr, ok := value.(string); ok {
				req.Header.Set(key, valueStr)
			}
		}
	}

	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Aster-Agent/1.0")
	}

	client := t.client
	if timeoutSec, ok := input["timeout"].(float64); ok && timeoutSec > 0 {
		client = &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		var netErr net.Error
		if ctx.Err() == context.DeadlineExceeded || (errors.As(err, &netErr) && netErr.Timeout()) {
			return map[string]any{
				"success": false,
				"error":   fmt.Sprintf("request timeout after %v", client.Timeout),
				"url":     url,
			}, nil
		}

		return map[string]any{
			"success": false,
			"error":   fmt.Sprintf("request failed: %v", err),
			"url":     url,
		}, nil
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return map[string]any{
			"success":     false,
			"error":       fmt.Sprintf("failed to read response body: %v", err),
			"status_code": resp.StatusCode,
			"url":         url,
		}, nil
	}

	var content any
	contentType := resp.Header.Get("Content-Type")

	if len(bodyBytes) > 0 {
		var jsonData any
		if err := json.Unmarshal(bodyBytes, &jsonData); err == nil {
			content = jsonData
		} else {
			content = string(bodyBytes)
		}
	} else {
		content = ""
	}

	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	return map[string]any{
		"success":      resp.StatusCode >= 200 && resp.StatusCode < 300,
		"status_code":  resp.StatusCode,
		"headers":      headers,
		"content":      content,
		"content_type": contentType,
		"url":          url,
	}, nil
}

func (t *WebFetchTool) Prompt() string {
	return `获取和分析网页内容。

支持的 HTTP 方法: GET, POST, PUT, DELETE, PATCH, HEAD

使用指南:
- 验证 URL 后再发送请求
- 根据操作类型选择合适的 HTTP 方法
- 设置适当的请求头（Content-Type, Authorization 等）
- 自动处理 JSON 和纯文本响应
- 默认超时 30 秒（可通过 timeout 参数配置）

响应格式:
- success: 请求是否成功（2xx 状态码）
- status_code: HTTP 状态码
- headers: 响应头（键值对）
- content: 解析后的 JSON 对象或纯文本
- content_type: Content-Type 头值
- url: 最终 URL（可能因重定向而不同）`
}

// Examples 返回 WebFetch 工具的使用示例
func (t *WebFetchTool) Examples() []tools.ToolExample {
	return []tools.ToolExample{
		{
			Description: "获取网页内容",
			Input: map[string]any{
				"url":    "https://api.example.com/data",
				"method": "GET",
			},
		},
		{
			Description: "发送 POST 请求",
			Input: map[string]any{
				"url":    "https://api.example.com/users",
				"method": "POST",
				"headers": map[string]string{
					"Content-Type": "application/json",
				},
				"body": `{"name": "John"}`,
			},
		},
	}
}
