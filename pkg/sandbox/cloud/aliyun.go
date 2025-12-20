package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/astercloud/aster/pkg/sandbox"
)

// AliyunSandbox 阿里云 AgentBay 沙箱
type AliyunSandbox struct {
	*sandbox.RemoteSandbox

	config    *AliyunConfig
	mcpClient *MCPClient
}

// AliyunConfig 阿里云沙箱配置
type AliyunConfig struct {
	// MCP 服务端点
	MCPEndpoint string

	// 认证信息
	AccessKeyID     string
	AccessKeySecret string
	SecurityToken   string

	// 沙箱配置
	Region      string // 默认 cn-hangzhou
	WorkDir     string // 允许的工作目录
	Image       string // Ubuntu/其他镜像
	Timeout     time.Duration
	Environment map[string]string

	// OSS 配置 (可选)
	OSSEndpoint string
	OSSBucket   string
}

// NewAliyunSandbox 创建阿里云沙箱
func NewAliyunSandbox(config *AliyunConfig) (*AliyunSandbox, error) {
	if config.MCPEndpoint == "" {
		return nil, errors.New("MCP endpoint is required")
	}
	if config.AccessKeyID == "" || config.AccessKeySecret == "" {
		return nil, errors.New("access credentials are required")
	}

	// 设置默认值
	if config.Region == "" {
		config.Region = "cn-hangzhou"
	}
	if config.WorkDir == "" {
		config.WorkDir = "/workspace"
	}
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}

	// 创建远程沙箱基础
	remoteConfig := &sandbox.RemoteSandboxConfig{
		BaseURL:   config.MCPEndpoint,
		APIKey:    config.AccessKeyID,
		APISecret: config.AccessKeySecret,
		WorkDir:   config.WorkDir,
		Timeout:   config.Timeout,
		Properties: map[string]any{
			"region":         config.Region,
			"security_token": config.SecurityToken,
		},
	}

	remoteSandbox, err := sandbox.NewRemoteSandbox(remoteConfig)
	if err != nil {
		return nil, fmt.Errorf("create remote sandbox: %w", err)
	}

	// 创建 MCP 客户端
	mcpClient := NewMCPClient(&MCPClientConfig{
		Endpoint:        config.MCPEndpoint,
		AccessKeyID:     config.AccessKeyID,
		AccessKeySecret: config.AccessKeySecret,
		SecurityToken:   config.SecurityToken,
		Timeout:         config.Timeout,
	})

	as := &AliyunSandbox{
		RemoteSandbox: remoteSandbox,
		config:        config,
		mcpClient:     mcpClient,
	}

	// 初始化 OSS (如果配置了)
	if config.OSSEndpoint != "" {
		if err := as.initOSS(context.Background()); err != nil {
			return nil, fmt.Errorf("init OSS: %w", err)
		}
	}

	return as, nil
}

// Kind 返回沙箱类型
func (as *AliyunSandbox) Kind() string {
	return "aliyun"
}

// Exec 执行 Shell 命令
func (as *AliyunSandbox) Exec(ctx context.Context, cmd string, opts *sandbox.ExecOptions) (*sandbox.ExecResult, error) {
	timeout := as.config.Timeout.Milliseconds()
	if opts != nil && opts.Timeout > 0 {
		timeout = opts.Timeout.Milliseconds()
	}

	// 调用 MCP Shell 工具
	result, err := as.mcpClient.CallTool(ctx, "shell", map[string]any{
		"command":    cmd,
		"timeout_ms": timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("exec shell: %w", err)
	}

	// 解析结果
	var shellResult struct {
		Stdout   string `json:"stdout"`
		Stderr   string `json:"stderr"`
		ExitCode int    `json:"exit_code"`
	}

	if err := json.Unmarshal(result, &shellResult); err != nil {
		return nil, fmt.Errorf("parse shell result: %w", err)
	}

	return &sandbox.ExecResult{
		Code:   shellResult.ExitCode,
		Stdout: shellResult.Stdout,
		Stderr: shellResult.Stderr,
	}, nil
}

// FS 返回文件系统接口
func (as *AliyunSandbox) FS() sandbox.SandboxFS {
	// 返回阿里云专用的文件系统实现
	return &AliyunFS{
		mcpClient: as.mcpClient,
		workDir:   as.config.WorkDir,
	}
}

// Dispose 清理资源
func (as *AliyunSandbox) Dispose() error {
	// AgentBay 沙箱通常是无状态的,不需要显式清理
	return nil
}

// initOSS 初始化 OSS 环境
func (as *AliyunSandbox) initOSS(ctx context.Context) error {
	endpoint := as.config.OSSEndpoint
	if endpoint == "" {
		endpoint = "https://oss-cn-hangzhou.aliyuncs.com"
	}

	_, err := as.mcpClient.CallTool(ctx, "oss_env_init", map[string]any{
		"access_key_id":     as.config.AccessKeyID,
		"access_key_secret": as.config.AccessKeySecret,
		"security_token":    as.config.SecurityToken,
		"endpoint":          endpoint,
		"region":            as.config.Region,
	})

	return err
}

// AliyunFS 阿里云文件系统实现
type AliyunFS struct {
	mcpClient *MCPClient
	workDir   string
}

// Resolve 解析路径为绝对路径
func (afs *AliyunFS) Resolve(path string) string {
	return afs.absPath(path)
}

// IsInside 检查路径是否在沙箱内
func (afs *AliyunFS) IsInside(path string) bool {
	absPath := afs.absPath(path)
	return strings.HasPrefix(absPath, afs.workDir)
}

// Read 读取文件
func (afs *AliyunFS) Read(ctx context.Context, path string) (string, error) {
	absPath := afs.absPath(path)

	result, err := afs.mcpClient.CallTool(ctx, "read_file", map[string]any{
		"path": absPath,
	})
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	// 解析结果
	var fileContent struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(result, &fileContent); err != nil {
		return "", fmt.Errorf("parse file content: %w", err)
	}

	return fileContent.Content, nil
}

// Write 写入文件
func (afs *AliyunFS) Write(ctx context.Context, path string, content string) error {
	absPath := afs.absPath(path)

	_, err := afs.mcpClient.CallTool(ctx, "write_file", map[string]any{
		"path":    absPath,
		"content": content,
	})
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// Temp 生成临时文件路径
func (afs *AliyunFS) Temp(name string) string {
	return filepath.Join(afs.workDir, ".tmp", name)
}

// Stat 获取文件信息
func (afs *AliyunFS) Stat(ctx context.Context, path string) (sandbox.FileInfo, error) {
	absPath := afs.absPath(path)

	result, err := afs.mcpClient.CallTool(ctx, "get_file_info", map[string]any{
		"path": absPath,
	})
	if err != nil {
		return sandbox.FileInfo{}, fmt.Errorf("stat file: %w", err)
	}

	// 解析结果
	var fileInfo struct {
		Path  string `json:"path"`
		Type  string `json:"type"`
		Size  int64  `json:"size"`
		MTime int64  `json:"mtime"`
	}
	if err := json.Unmarshal(result, &fileInfo); err != nil {
		return sandbox.FileInfo{}, fmt.Errorf("parse file info: %w", err)
	}

	return sandbox.FileInfo{
		Path:    fileInfo.Path,
		IsDir:   fileInfo.Type == "directory",
		Size:    fileInfo.Size,
		ModTime: time.Unix(fileInfo.MTime, 0),
	}, nil
}

// Glob 匹配文件
func (afs *AliyunFS) Glob(ctx context.Context, pattern string, opts *sandbox.GlobOptions) ([]string, error) {
	absPattern := afs.absPath(pattern)

	result, err := afs.mcpClient.CallTool(ctx, "search_files", map[string]any{
		"path":    filepath.Dir(absPattern),
		"pattern": filepath.Base(absPattern),
	})
	if err != nil {
		return nil, fmt.Errorf("glob files: %w", err)
	}

	// 解析结果
	var searchResult struct {
		Files []string `json:"files"`
	}
	if err := json.Unmarshal(result, &searchResult); err != nil {
		return nil, fmt.Errorf("parse search result: %w", err)
	}

	return searchResult.Files, nil
}

// absPath 转换为绝对路径
func (afs *AliyunFS) absPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(afs.workDir, path)
}
