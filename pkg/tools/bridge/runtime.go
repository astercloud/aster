package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Language 支持的语言类型
type Language string

const (
	LangPython Language = "python"
	LangNodeJS Language = "nodejs"
	LangBash   Language = "bash"
)

// CodeRuntime 代码运行时接口
type CodeRuntime interface {
	// Execute 执行代码
	Execute(ctx context.Context, code string, input map[string]any) (*ExecutionResult, error)
	// Language 返回支持的语言
	Language() Language
	// IsAvailable 检查运行时是否可用
	IsAvailable() bool
}

// ExecutionResult 代码执行结果
type ExecutionResult struct {
	Success  bool   `json:"success"`
	Output   any    `json:"output,omitempty"`
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	Error    string `json:"error,omitempty"`
	ExitCode int    `json:"exit_code"`
	Duration int64  `json:"duration_ms"`
}

// RuntimeConfig 运行时配置
type RuntimeConfig struct {
	Timeout   time.Duration
	WorkDir   string
	Env       map[string]string
	MaxOutput int // 最大输出字节数
}

// DefaultRuntimeConfig 默认配置
func DefaultRuntimeConfig() *RuntimeConfig {
	return &RuntimeConfig{
		Timeout:   30 * time.Second,
		WorkDir:   os.TempDir(),
		Env:       make(map[string]string),
		MaxOutput: 1024 * 1024, // 1MB
	}
}

// PythonRuntime Python 运行时
type PythonRuntime struct {
	config         *RuntimeConfig
	pythonPath     string
	availableTools []string // PTC: 可用工具列表
	bridgeURL      string   // PTC: HTTP 桥接服务器地址
}

// NewPythonRuntime 创建 Python 运行时
func NewPythonRuntime(config *RuntimeConfig) *PythonRuntime {
	if config == nil {
		config = DefaultRuntimeConfig()
	}

	// 查找 Python 可执行文件
	pythonPath := "python3"
	if path, err := exec.LookPath("python3"); err == nil {
		pythonPath = path
	} else if path, err := exec.LookPath("python"); err == nil {
		pythonPath = path
	}

	return &PythonRuntime{
		config:     config,
		pythonPath: pythonPath,
	}
}

func (r *PythonRuntime) Language() Language {
	return LangPython
}

func (r *PythonRuntime) IsAvailable() bool {
	_, err := exec.LookPath(r.pythonPath)
	return err == nil
}

// SetTools 设置可用工具列表 (PTC 支持)
func (r *PythonRuntime) SetTools(tools []string) {
	r.availableTools = tools
}

// SetBridgeURL 设置 HTTP 桥接服务器地址 (PTC 支持)
func (r *PythonRuntime) SetBridgeURL(url string) {
	r.bridgeURL = url
}

func (r *PythonRuntime) Execute(ctx context.Context, code string, input map[string]any) (*ExecutionResult, error) {
	start := time.Now()

	// 创建临时文件
	tmpFile, err := os.CreateTemp(r.config.WorkDir, "aster_*.py")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	// 包装代码以处理输入和输出
	wrappedCode := r.wrapCode(code, input)
	if _, err := tmpFile.WriteString(wrappedCode); err != nil {
		return nil, fmt.Errorf("write code: %w", err)
	}
	_ = tmpFile.Close()

	// 创建带超时的 context
	execCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
	defer cancel()

	// 执行 Python
	cmd := exec.CommandContext(execCtx, r.pythonPath, tmpFile.Name())
	cmd.Dir = r.config.WorkDir

	// 设置环境变量
	cmd.Env = os.Environ()
	for k, v := range r.config.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	duration := time.Since(start).Milliseconds()

	result := &ExecutionResult{
		Stdout:   truncateOutput(stdout.String(), r.config.MaxOutput),
		Stderr:   truncateOutput(stderr.String(), r.config.MaxOutput),
		Duration: duration,
	}

	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			result.Error = "execution timeout"
			result.ExitCode = -1
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			result.Error = stderr.String()
		} else {
			result.Error = err.Error()
			result.ExitCode = -1
		}
		return result, nil
	}

	result.Success = true
	result.ExitCode = 0

	// 尝试解析 JSON 输出
	output := strings.TrimSpace(stdout.String())
	if strings.HasPrefix(output, "{") || strings.HasPrefix(output, "[") {
		var jsonOutput any
		if err := json.Unmarshal([]byte(output), &jsonOutput); err == nil {
			result.Output = jsonOutput
		} else {
			result.Output = output
		}
	} else {
		result.Output = output
	}

	return result, nil
}

func (r *PythonRuntime) wrapCode(code string, input map[string]any) string {
	inputJSON, _ := json.Marshal(input)

	// 如果没有配置工具,使用简单包装
	if len(r.availableTools) == 0 {
		return fmt.Sprintf(`import json
import sys

# Input data
_input = json.loads('%s')

# User code
%s
`, string(inputJSON), code)
	}

	// PTC 模式: 注入工具桥接代码
	bridgeURL := r.bridgeURL
	if bridgeURL == "" {
		bridgeURL = "http://localhost:8080"
	}

	// 生成工具列表 JSON
	toolsJSON, _ := json.Marshal(r.availableTools)

	return fmt.Sprintf(`import json
import asyncio
import sys
import os

# ========== Aster Bridge SDK (内联) ==========
try:
    import aiohttp
except ImportError:
    print("Error: aiohttp is required. Install it with: pip install aiohttp", file=sys.stderr)
    sys.exit(1)

class _ToolExecutionError(Exception):
    """工具执行错误"""
    pass

class _NetworkError(Exception):
    """网络错误"""
    pass

class _AsterBridge:
    def __init__(self, base_url, max_retries=3, retry_delay=0.5):
        self.base_url = base_url
        self.max_retries = max_retries
        self.retry_delay = retry_delay
        self._session = None

    async def _get_session(self):
        if self._session is None or self._session.closed:
            self._session = aiohttp.ClientSession()
        return self._session

    async def call_tool(self, name, **kwargs):
        last_error = None
        for attempt in range(self.max_retries):
            try:
                session = await self._get_session()
                async with session.post(
                    f"{self.base_url}/tools/call",
                    json={"tool": name, "input": kwargs},
                    timeout=aiohttp.ClientTimeout(total=60),
                ) as resp:
                    if resp.status >= 500:
                        error_text = await resp.text()
                        last_error = _NetworkError(f"Server error (HTTP {resp.status}): {error_text}")
                        if attempt < self.max_retries - 1:
                            await asyncio.sleep(self.retry_delay * (2 ** attempt))
                            continue
                        raise last_error
                    if resp.status >= 400:
                        error_text = await resp.text()
                        raise _NetworkError(f"Client error (HTTP {resp.status}): {error_text}")
                    result = await resp.json()
                    if not result.get("success"):
                        error_msg = result.get("error", "Unknown error")
                        raise _ToolExecutionError(f"Tool {name} failed: {error_msg}")
                    return result.get("result")
            except aiohttp.ClientConnectorError as e:
                last_error = _NetworkError(f"Connection error: {str(e)}. Is bridge server running?")
                if attempt < self.max_retries - 1:
                    await asyncio.sleep(self.retry_delay * (2 ** attempt))
                    continue
                raise last_error
            except aiohttp.ClientError as e:
                last_error = _NetworkError(f"Network error: {str(e)}")
                if attempt < self.max_retries - 1:
                    await asyncio.sleep(self.retry_delay * (2 ** attempt))
                    continue
                raise last_error
            except _ToolExecutionError:
                raise
            except asyncio.TimeoutError:
                last_error = _NetworkError(f"Tool {name} timed out after 60 seconds")
                if attempt < self.max_retries - 1:
                    await asyncio.sleep(self.retry_delay * (2 ** attempt))
                    continue
                raise last_error
        if last_error:
            raise last_error
        raise _NetworkError(f"Failed to call tool {name} after {self.max_retries} attempts")

    async def close(self):
        if self._session and not self._session.closed:
            await self._session.close()

# 初始化桥接
_bridge = _AsterBridge("%s")

# 动态生成工具函数
def _create_tool_function(bridge, tool_name):
    async def tool_func(**kwargs):
        return await bridge.call_tool(tool_name, **kwargs)
    tool_func.__name__ = tool_name
    return tool_func

# 注入工具到全局命名空间
_available_tools = %s
for _tool_name in _available_tools:
    globals()[_tool_name] = _create_tool_function(_bridge, _tool_name)

# ========== 用户代码开始 ==========

# Input data
_input = json.loads('%s')

# 包装用户代码在 async main 中
async def _user_main():
%s

# 运行用户代码
if __name__ == "__main__":
    try:
        asyncio.run(_user_main())
    finally:
        # 确保关闭会话
        asyncio.run(_bridge.close())
`, bridgeURL, string(toolsJSON), string(inputJSON), indentCode(code, "    "))
}

// indentCode 缩进代码
func indentCode(code string, indent string) string {
	lines := strings.Split(code, "\n")
	var indented []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			indented = append(indented, "")
		} else {
			indented = append(indented, indent+line)
		}
	}
	return strings.Join(indented, "\n")
}

// NodeJSRuntime Node.js 运行时
type NodeJSRuntime struct {
	config   *RuntimeConfig
	nodePath string
}

// NewNodeJSRuntime 创建 Node.js 运行时
func NewNodeJSRuntime(config *RuntimeConfig) *NodeJSRuntime {
	if config == nil {
		config = DefaultRuntimeConfig()
	}

	nodePath := "node"
	if path, err := exec.LookPath("node"); err == nil {
		nodePath = path
	}

	return &NodeJSRuntime{
		config:   config,
		nodePath: nodePath,
	}
}

func (r *NodeJSRuntime) Language() Language {
	return LangNodeJS
}

func (r *NodeJSRuntime) IsAvailable() bool {
	_, err := exec.LookPath(r.nodePath)
	return err == nil
}

func (r *NodeJSRuntime) Execute(ctx context.Context, code string, input map[string]any) (*ExecutionResult, error) {
	start := time.Now()

	// 创建临时文件
	tmpFile, err := os.CreateTemp(r.config.WorkDir, "aster_*.js")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	// 包装代码
	wrappedCode := r.wrapCode(code, input)
	if _, err := tmpFile.WriteString(wrappedCode); err != nil {
		return nil, fmt.Errorf("write code: %w", err)
	}
	_ = tmpFile.Close()

	// 创建带超时的 context
	execCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
	defer cancel()

	// 执行 Node.js
	cmd := exec.CommandContext(execCtx, r.nodePath, tmpFile.Name())
	cmd.Dir = r.config.WorkDir

	// 设置环境变量
	cmd.Env = os.Environ()
	for k, v := range r.config.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	duration := time.Since(start).Milliseconds()

	result := &ExecutionResult{
		Stdout:   truncateOutput(stdout.String(), r.config.MaxOutput),
		Stderr:   truncateOutput(stderr.String(), r.config.MaxOutput),
		Duration: duration,
	}

	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			result.Error = "execution timeout"
			result.ExitCode = -1
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			result.Error = stderr.String()
		} else {
			result.Error = err.Error()
			result.ExitCode = -1
		}
		return result, nil
	}

	result.Success = true
	result.ExitCode = 0

	// 尝试解析 JSON 输出
	output := strings.TrimSpace(stdout.String())
	if strings.HasPrefix(output, "{") || strings.HasPrefix(output, "[") {
		var jsonOutput any
		if err := json.Unmarshal([]byte(output), &jsonOutput); err == nil {
			result.Output = jsonOutput
		} else {
			result.Output = output
		}
	} else {
		result.Output = output
	}

	return result, nil
}

func (r *NodeJSRuntime) wrapCode(code string, input map[string]any) string {
	inputJSON, _ := json.Marshal(input)
	return fmt.Sprintf(`// Input data
const _input = %s;

// User code
%s
`, string(inputJSON), code)
}

// BashRuntime Bash 运行时
type BashRuntime struct {
	config   *RuntimeConfig
	bashPath string
}

// NewBashRuntime 创建 Bash 运行时
func NewBashRuntime(config *RuntimeConfig) *BashRuntime {
	if config == nil {
		config = DefaultRuntimeConfig()
	}

	bashPath := "/bin/bash"
	if path, err := exec.LookPath("bash"); err == nil {
		bashPath = path
	}

	return &BashRuntime{
		config:   config,
		bashPath: bashPath,
	}
}

func (r *BashRuntime) Language() Language {
	return LangBash
}

func (r *BashRuntime) IsAvailable() bool {
	_, err := exec.LookPath(r.bashPath)
	return err == nil
}

func (r *BashRuntime) Execute(ctx context.Context, code string, input map[string]any) (*ExecutionResult, error) {
	start := time.Now()

	// 创建临时文件
	tmpFile, err := os.CreateTemp(r.config.WorkDir, "aster_*.sh")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	// 包装代码
	wrappedCode := r.wrapCode(code, input)
	if _, err := tmpFile.WriteString(wrappedCode); err != nil {
		return nil, fmt.Errorf("write code: %w", err)
	}
	_ = tmpFile.Close()

	// 设置执行权限
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return nil, fmt.Errorf("chmod: %w", err)
	}

	// 创建带超时的 context
	execCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
	defer cancel()

	// 执行 Bash
	cmd := exec.CommandContext(execCtx, r.bashPath, tmpFile.Name())
	cmd.Dir = r.config.WorkDir

	// 设置环境变量
	cmd.Env = os.Environ()
	for k, v := range r.config.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	// 将 input 作为环境变量
	for k, v := range input {
		if str, ok := v.(string); ok {
			cmd.Env = append(cmd.Env, fmt.Sprintf("INPUT_%s=%s", strings.ToUpper(k), str))
		}
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	duration := time.Since(start).Milliseconds()

	result := &ExecutionResult{
		Stdout:   truncateOutput(stdout.String(), r.config.MaxOutput),
		Stderr:   truncateOutput(stderr.String(), r.config.MaxOutput),
		Duration: duration,
	}

	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			result.Error = "execution timeout"
			result.ExitCode = -1
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			result.Error = stderr.String()
		} else {
			result.Error = err.Error()
			result.ExitCode = -1
		}
		return result, nil
	}

	result.Success = true
	result.ExitCode = 0
	result.Output = strings.TrimSpace(stdout.String())

	return result, nil
}

func (r *BashRuntime) wrapCode(code string, input map[string]any) string {
	inputJSON, _ := json.Marshal(input)
	return fmt.Sprintf(`#!/bin/bash
set -e

# Input as JSON (use jq to parse)
INPUT_JSON='%s'

# User code
%s
`, string(inputJSON), code)
}

// RuntimeManager 运行时管理器
type RuntimeManager struct {
	runtimes map[Language]CodeRuntime
}

// NewRuntimeManager 创建运行时管理器
func NewRuntimeManager(config *RuntimeConfig) *RuntimeManager {
	if config == nil {
		config = DefaultRuntimeConfig()
	}

	return &RuntimeManager{
		runtimes: map[Language]CodeRuntime{
			LangPython: NewPythonRuntime(config),
			LangNodeJS: NewNodeJSRuntime(config),
			LangBash:   NewBashRuntime(config),
		},
	}
}

// GetRuntime 获取指定语言的运行时
func (m *RuntimeManager) GetRuntime(lang Language) (CodeRuntime, bool) {
	runtime, exists := m.runtimes[lang]
	return runtime, exists
}

// Execute 执行代码
func (m *RuntimeManager) Execute(ctx context.Context, lang Language, code string, input map[string]any) (*ExecutionResult, error) {
	runtime, exists := m.runtimes[lang]
	if !exists {
		return nil, fmt.Errorf("unsupported language: %s", lang)
	}

	if !runtime.IsAvailable() {
		return nil, fmt.Errorf("runtime not available: %s", lang)
	}

	return runtime.Execute(ctx, code, input)
}

// AvailableLanguages 返回可用的语言列表
func (m *RuntimeManager) AvailableLanguages() []Language {
	langs := make([]Language, 0)
	for lang, runtime := range m.runtimes {
		if runtime.IsAvailable() {
			langs = append(langs, lang)
		}
	}
	return langs
}

// SetPythonTools 设置 Python 运行时的可用工具列表 (PTC 支持)
func (m *RuntimeManager) SetPythonTools(tools []string) {
	if runtime, ok := m.runtimes[LangPython].(*PythonRuntime); ok {
		runtime.SetTools(tools)
	}
}

// SetPythonBridgeURL 设置 Python 运行时的 HTTP 桥接服务器地址 (PTC 支持)
func (m *RuntimeManager) SetPythonBridgeURL(url string) {
	if runtime, ok := m.runtimes[LangPython].(*PythonRuntime); ok {
		runtime.SetBridgeURL(url)
	}
}

// DetectLanguage 根据文件扩展名检测语言
func DetectLanguage(filename string) Language {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".py":
		return LangPython
	case ".js", ".mjs":
		return LangNodeJS
	case ".sh", ".bash":
		return LangBash
	default:
		return ""
	}
}

// truncateOutput 截断输出
func truncateOutput(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n...(truncated)"
}
