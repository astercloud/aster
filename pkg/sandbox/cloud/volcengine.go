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

// VolcengineSandbox 火山引擎沙箱
type VolcengineSandbox struct {
	*sandbox.RemoteSandbox

	config    *VolcengineConfig
	mcpClient *MCPClient
	sessionID string
}

// VolcengineConfig 火山引擎沙箱配置
type VolcengineConfig struct {
	// API 配置
	Endpoint  string
	AccessKey string
	SecretKey string

	// 沙箱配置
	Region      string
	WorkDir     string
	Image       string
	Timeout     time.Duration
	Environment map[string]string

	// 计算资源配置
	CPU    int // vCPU 核数
	Memory int // 内存 MB
}

// NewVolcengineSandbox 创建火山引擎沙箱
func NewVolcengineSandbox(config *VolcengineConfig) (*VolcengineSandbox, error) {
	if config.Endpoint == "" {
		return nil, errors.New("endpoint is required")
	}
	if config.AccessKey == "" || config.SecretKey == "" {
		return nil, errors.New("access credentials are required")
	}

	// 设置默认值
	if config.Region == "" {
		config.Region = "cn-beijing"
	}
	if config.WorkDir == "" {
		config.WorkDir = "/workspace"
	}
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}
	if config.CPU == 0 {
		config.CPU = 2
	}
	if config.Memory == 0 {
		config.Memory = 4096
	}

	// 创建远程沙箱基础
	remoteConfig := &sandbox.RemoteSandboxConfig{
		BaseURL:   config.Endpoint,
		APIKey:    config.AccessKey,
		APISecret: config.SecretKey,
		WorkDir:   config.WorkDir,
		Timeout:   config.Timeout,
		Properties: map[string]any{
			"region": config.Region,
			"cpu":    config.CPU,
			"memory": config.Memory,
		},
	}

	remoteSandbox, err := sandbox.NewRemoteSandbox(remoteConfig)
	if err != nil {
		return nil, fmt.Errorf("create remote sandbox: %w", err)
	}

	// 创建 MCP 客户端
	mcpClient := NewMCPClient(&MCPClientConfig{
		Endpoint:        config.Endpoint,
		AccessKeyID:     config.AccessKey,
		AccessKeySecret: config.SecretKey,
		Timeout:         config.Timeout,
	})

	vs := &VolcengineSandbox{
		RemoteSandbox: remoteSandbox,
		config:        config,
		mcpClient:     mcpClient,
	}

	// 初始化沙箱会话
	if err := vs.initSession(context.Background()); err != nil {
		return nil, fmt.Errorf("init session: %w", err)
	}

	return vs, nil
}

// Kind 返回沙箱类型
func (vs *VolcengineSandbox) Kind() string {
	return "volcengine"
}

// Exec 执行命令
func (vs *VolcengineSandbox) Exec(ctx context.Context, cmd string, opts *sandbox.ExecOptions) (*sandbox.ExecResult, error) {
	timeout := vs.config.Timeout.Milliseconds()
	if opts != nil && opts.Timeout > 0 {
		timeout = opts.Timeout.Milliseconds()
	}

	// 火山引擎使用 computer_exec 工具
	result, err := vs.mcpClient.CallTool(ctx, "computer_exec", map[string]any{
		"session_id": vs.sessionID,
		"command":    cmd,
		"timeout":    timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("exec command: %w", err)
	}

	// 解析结果
	var execResult struct {
		Stdout   string `json:"stdout"`
		Stderr   string `json:"stderr"`
		ExitCode int    `json:"exit_code"`
	}

	if err := json.Unmarshal(result, &execResult); err != nil {
		return nil, fmt.Errorf("parse exec result: %w", err)
	}

	return &sandbox.ExecResult{
		Code:   execResult.ExitCode,
		Stdout: execResult.Stdout,
		Stderr: execResult.Stderr,
	}, nil
}

// FS 返回文件系统接口
func (vs *VolcengineSandbox) FS() sandbox.SandboxFS {
	return &VolcengineFS{
		mcpClient: vs.mcpClient,
		sessionID: vs.sessionID,
		workDir:   vs.config.WorkDir,
	}
}

// Dispose 清理资源
func (vs *VolcengineSandbox) Dispose() error {
	if vs.sessionID == "" {
		return nil
	}

	// 终止沙箱会话
	_, err := vs.mcpClient.CallTool(context.Background(), "computer_terminate", map[string]any{
		"session_id": vs.sessionID,
	})

	return err
}

// initSession 初始化沙箱会话
func (vs *VolcengineSandbox) initSession(ctx context.Context) error {
	params := map[string]any{
		"work_dir": vs.config.WorkDir,
		"cpu":      vs.config.CPU,
		"memory":   vs.config.Memory,
	}

	if vs.config.Image != "" {
		params["image"] = vs.config.Image
	}
	if vs.config.Environment != nil {
		params["environment"] = vs.config.Environment
	}

	result, err := vs.mcpClient.CallTool(ctx, "computer_init", params)
	if err != nil {
		return fmt.Errorf("init computer: %w", err)
	}

	// 解析会话 ID
	var initResult struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(result, &initResult); err != nil {
		return fmt.Errorf("parse init result: %w", err)
	}

	vs.sessionID = initResult.SessionID
	vs.SetSessionID(initResult.SessionID)

	return nil
}

// VolcengineFS 火山引擎文件系统
type VolcengineFS struct {
	mcpClient *MCPClient
	sessionID string
	workDir   string
}

// Resolve 解析路径为绝对路径
func (vfs *VolcengineFS) Resolve(path string) string {
	return vfs.absPath(path)
}

// IsInside 检查路径是否在沙箱内
func (vfs *VolcengineFS) IsInside(path string) bool {
	absPath := vfs.absPath(path)
	return strings.HasPrefix(absPath, vfs.workDir)
}

// Read 读取文件
func (vfs *VolcengineFS) Read(ctx context.Context, path string) (string, error) {
	absPath := vfs.absPath(path)

	result, err := vfs.mcpClient.CallTool(ctx, "computer_read_file", map[string]any{
		"session_id": vfs.sessionID,
		"path":       absPath,
	})
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	var fileContent struct {
		Content string `json:"content"`
		Size    int64  `json:"size"`
	}
	if err := json.Unmarshal(result, &fileContent); err != nil {
		return "", fmt.Errorf("parse file content: %w", err)
	}

	return fileContent.Content, nil
}

// Write 写入文件
func (vfs *VolcengineFS) Write(ctx context.Context, path string, content string) error {
	absPath := vfs.absPath(path)

	_, err := vfs.mcpClient.CallTool(ctx, "computer_write_file", map[string]any{
		"session_id": vfs.sessionID,
		"path":       absPath,
		"content":    content,
	})
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// Temp 生成临时文件路径
func (vfs *VolcengineFS) Temp(name string) string {
	return filepath.Join(vfs.workDir, ".tmp", name)
}

// Stat 获取文件信息
func (vfs *VolcengineFS) Stat(ctx context.Context, path string) (sandbox.FileInfo, error) {
	absPath := vfs.absPath(path)

	result, err := vfs.mcpClient.CallTool(ctx, "computer_stat_file", map[string]any{
		"session_id": vfs.sessionID,
		"path":       absPath,
	})
	if err != nil {
		return sandbox.FileInfo{}, fmt.Errorf("stat file: %w", err)
	}

	var fileInfo struct {
		Path  string `json:"path"`
		IsDir bool   `json:"is_dir"`
		Size  int64  `json:"size"`
		MTime int64  `json:"mtime"`
	}
	if err := json.Unmarshal(result, &fileInfo); err != nil {
		return sandbox.FileInfo{}, fmt.Errorf("parse file info: %w", err)
	}

	return sandbox.FileInfo{
		Path:    fileInfo.Path,
		IsDir:   fileInfo.IsDir,
		Size:    fileInfo.Size,
		ModTime: time.Unix(fileInfo.MTime, 0),
	}, nil
}

// Glob 匹配文件
func (vfs *VolcengineFS) Glob(ctx context.Context, pattern string, opts *sandbox.GlobOptions) ([]string, error) {
	absPattern := vfs.absPath(pattern)

	result, err := vfs.mcpClient.CallTool(ctx, "computer_glob", map[string]any{
		"session_id": vfs.sessionID,
		"pattern":    absPattern,
	})
	if err != nil {
		return nil, fmt.Errorf("glob files: %w", err)
	}

	var globResult struct {
		Matches []string `json:"matches"`
	}
	if err := json.Unmarshal(result, &globResult); err != nil {
		return nil, fmt.Errorf("parse glob result: %w", err)
	}

	return globResult.Matches, nil
}

// absPath 转换为绝对路径
func (vfs *VolcengineFS) absPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(vfs.workDir, path)
}
