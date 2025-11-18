package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// RemoteClient 远程沙箱客户端
type RemoteClient struct {
	baseURL    string
	apiKey     string
	apiSecret  string
	httpClient *http.Client
	headers    map[string]string
}

// NewRemoteClient 创建远程客户端
func NewRemoteClient(config *RemoteClientConfig) *RemoteClient {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &RemoteClient{
		baseURL:   config.BaseURL,
		apiKey:    config.APIKey,
		apiSecret: config.APISecret,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		headers: config.Headers,
	}
}

// RemoteClientConfig 远程客户端配置
type RemoteClientConfig struct {
	BaseURL   string
	APIKey    string
	APISecret string
	Timeout   time.Duration
	Headers   map[string]string
}

// Call 调用远程 API
func (rc *RemoteClient) Call(ctx context.Context, method, path string, body interface{}) (*RemoteResponse, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	url := rc.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置通用请求头
	req.Header.Set("Content-Type", "application/json")
	if rc.apiKey != "" {
		req.Header.Set("X-API-Key", rc.apiKey)
	}

	// 设置自定义请求头
	for k, v := range rc.headers {
		req.Header.Set(k, v)
	}

	// 发送请求
	resp, err := rc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// 检查状态码
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("api error: %d - %s", resp.StatusCode, string(respBody))
	}

	return &RemoteResponse{
		StatusCode: resp.StatusCode,
		Body:       respBody,
		Headers:    resp.Header,
	}, nil
}

// RemoteResponse 远程响应
type RemoteResponse struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// JSON 解析 JSON 响应
func (rr *RemoteResponse) JSON(v interface{}) error {
	return json.Unmarshal(rr.Body, v)
}

// String 返回字符串响应
func (rr *RemoteResponse) String() string {
	return string(rr.Body)
}

// RemoteSandbox 远程沙箱基础实现
type RemoteSandbox struct {
	config     *RemoteSandboxConfig
	client     *RemoteClient
	sessionID  string
	workDir    string
	fs         *RemoteFS
	properties map[string]interface{}
}

// RemoteSandboxConfig 远程沙箱配置
type RemoteSandboxConfig struct {
	BaseURL     string
	APIKey      string
	APISecret   string
	WorkDir     string
	Image       string            // 沙箱镜像
	Region      string            // 区域
	Timeout     time.Duration     // 超时时间
	Environment map[string]string // 环境变量
	Properties  map[string]interface{}
}

// NewRemoteSandbox 创建远程沙箱
func NewRemoteSandbox(config *RemoteSandboxConfig) (*RemoteSandbox, error) {
	client := NewRemoteClient(&RemoteClientConfig{
		BaseURL:   config.BaseURL,
		APIKey:    config.APIKey,
		APISecret: config.APISecret,
		Timeout:   config.Timeout,
	})

	rs := &RemoteSandbox{
		config:     config,
		client:     client,
		workDir:    config.WorkDir,
		properties: config.Properties,
	}

	rs.fs = &RemoteFS{
		sandbox: rs,
		workDir: config.WorkDir,
	}

	return rs, nil
}

// Kind 返回沙箱类型
func (rs *RemoteSandbox) Kind() string {
	return "remote"
}

// Exec 执行命令 (需要子类实现具体的 API 调用)
func (rs *RemoteSandbox) Exec(ctx context.Context, cmd string, opts *ExecOptions) (*ExecResult, error) {
	return nil, fmt.Errorf("exec not implemented in base RemoteSandbox")
}

// FS 返回文件系统接口
func (rs *RemoteSandbox) FS() SandboxFS {
	return rs.fs
}

// WorkDir 返回工作目录
func (rs *RemoteSandbox) WorkDir() string {
	return rs.workDir
}

// Watch 监听文件变化 (远程沙箱通常不支持)
func (rs *RemoteSandbox) Watch(paths []string, listener FileChangeListener) (string, error) {
	return "", fmt.Errorf("watch not supported in remote sandbox")
}

// Unwatch 取消监听 (远程沙箱通常不支持)
func (rs *RemoteSandbox) Unwatch(watchID string) error {
	return fmt.Errorf("unwatch not supported in remote sandbox")
}

// Dispose 清理资源
func (rs *RemoteSandbox) Dispose() error {
	// 子类应该实现会话清理逻辑
	return nil
}

// SessionID 返回会话 ID
func (rs *RemoteSandbox) SessionID() string {
	return rs.sessionID
}

// SetSessionID 设置会话 ID
func (rs *RemoteSandbox) SetSessionID(id string) {
	rs.sessionID = id
}

// RemoteFS 远程文件系统
type RemoteFS struct {
	sandbox *RemoteSandbox
	workDir string
}

// Resolve 解析路径为绝对路径
func (rfs *RemoteFS) Resolve(path string) string {
	// 子类应该实现
	return path
}

// IsInside 检查路径是否在沙箱内
func (rfs *RemoteFS) IsInside(path string) bool {
	// 远程沙箱由服务端控制边界
	return true
}

// Read 读取文件 (需要子类实现)
func (rfs *RemoteFS) Read(ctx context.Context, path string) (string, error) {
	return "", fmt.Errorf("read not implemented in base RemoteFS")
}

// Write 写入文件 (需要子类实现)
func (rfs *RemoteFS) Write(ctx context.Context, path string, content string) error {
	return fmt.Errorf("write not implemented in base RemoteFS")
}

// Temp 生成临时文件路径
func (rfs *RemoteFS) Temp(name string) string {
	return "/tmp/" + name
}

// Stat 获取文件信息 (需要子类实现)
func (rfs *RemoteFS) Stat(ctx context.Context, path string) (FileInfo, error) {
	return FileInfo{}, fmt.Errorf("stat not implemented in base RemoteFS")
}

// Glob 匹配文件 (需要子类实现)
func (rfs *RemoteFS) Glob(ctx context.Context, pattern string, opts *GlobOptions) ([]string, error) {
	return nil, fmt.Errorf("glob not implemented in base RemoteFS")
}
