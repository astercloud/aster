package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// MCPClient MCP 协议客户端
type MCPClient struct {
	endpoint        string
	accessKeyID     string
	accessKeySecret string
	securityToken   string
	httpClient      *http.Client
}

// MCPClientConfig MCP 客户端配置
type MCPClientConfig struct {
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	SecurityToken   string
	Timeout         time.Duration
}

// NewMCPClient 创建 MCP 客户端
func NewMCPClient(config *MCPClientConfig) *MCPClient {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &MCPClient{
		endpoint:        config.Endpoint,
		accessKeyID:     config.AccessKeyID,
		accessKeySecret: config.AccessKeySecret,
		securityToken:   config.SecurityToken,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// CallTool 调用 MCP 工具
func (mc *MCPClient) CallTool(ctx context.Context, toolName string, params map[string]any) (json.RawMessage, error) {
	// 构建 MCP 请求
	request := &MCPRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		ID:      time.Now().UnixNano(),
		Params: MCPCallParams{
			Name:      toolName,
			Arguments: params,
		},
	}

	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, mc.endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Access-Key-Id", mc.accessKeyID)
	httpReq.Header.Set("X-Access-Key-Secret", mc.accessKeySecret)
	if mc.securityToken != "" {
		httpReq.Header.Set("X-Security-Token", mc.securityToken)
	}

	// 发送请求
	resp, err := mc.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http error: %d - %s", resp.StatusCode, string(respBody))
	}

	// 解析 MCP 响应
	var mcpResp MCPResponse
	if err := json.Unmarshal(respBody, &mcpResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// 检查 MCP 错误
	if mcpResp.Error != nil {
		return nil, fmt.Errorf("mcp error: %s (code: %d)", mcpResp.Error.Message, mcpResp.Error.Code)
	}

	return mcpResp.Result, nil
}

// ListTools 列出可用工具
func (mc *MCPClient) ListTools(ctx context.Context) ([]MCPTool, error) {
	request := &MCPRequest{
		JSONRPC: "2.0",
		Method:  "tools/list",
		ID:      time.Now().UnixNano(),
	}

	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, mc.endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Access-Key-Id", mc.accessKeyID)
	httpReq.Header.Set("X-Access-Key-Secret", mc.accessKeySecret)
	if mc.securityToken != "" {
		httpReq.Header.Set("X-Security-Token", mc.securityToken)
	}

	resp, err := mc.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http error: %d - %s", resp.StatusCode, string(respBody))
	}

	var mcpResp struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int64  `json:"id"`
		Result  struct {
			Tools []MCPTool `json:"tools"`
		} `json:"result"`
		Error *MCPError `json:"error,omitempty"`
	}

	if err := json.Unmarshal(respBody, &mcpResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if mcpResp.Error != nil {
		return nil, fmt.Errorf("mcp error: %s", mcpResp.Error.Message)
	}

	return mcpResp.Result.Tools, nil
}

// MCPRequest MCP 请求
type MCPRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	ID      int64         `json:"id"`
	Params  MCPCallParams `json:"params,omitempty"`
}

// MCPCallParams 工具调用参数
type MCPCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// MCPResponse MCP 响应
type MCPResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
}

// MCPError MCP 错误
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// MCPTool MCP 工具定义
type MCPTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}
