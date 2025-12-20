package middleware

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/astercloud/aster/pkg/backends"
	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/builtin"
)

var fsLog = logging.ForComponent("FilesystemMiddleware")

// FilesystemMiddlewareConfig 文件系统中间件配置
type FilesystemMiddlewareConfig struct {
	Backend                 backends.BackendProtocol // 后端存储
	TokenLimit              int                      // 大结果驱逐阈值(tokens)
	EnableEviction          bool                     // 是否启用自动驱逐
	AllowedPathPrefixes     []string                 // 允许的路径前缀列表(用于路径安全验证)
	EnablePathValidation    bool                     // 是否启用路径验证(默认: true)
	CustomToolDescriptions  map[string]string        // 自定义工具描述
	SystemPromptOverride    string                   // 覆盖默认系统提示词（设置为空字符串可禁用注入）
	HasSystemPromptOverride bool                     // 是否明确设置了 SystemPromptOverride
}

// FilesystemMiddleware 文件系统中间件
// 功能:
// 1. 注入文件系统工具 (ls, read, write, edit, glob, grep)
// 2. 自动驱逐大结果到文件
// 3. 增强系统提示词
// 4. 路径安全验证
type FilesystemMiddleware struct {
	*BaseMiddleware

	backend                 backends.BackendProtocol
	tokenLimit              int
	enableEviction          bool
	allowedPathPrefixes     []string
	enablePathValidation    bool
	customToolDescriptions  map[string]string
	systemPromptOverride    string
	hasSystemPromptOverride bool // 是否明确设置了 systemPromptOverride（包括空字符串）
	fsTools                 []tools.Tool
}

// NewFilesystemMiddleware 创建文件系统中间件
func NewFilesystemMiddleware(config *FilesystemMiddlewareConfig) *FilesystemMiddleware {
	if config.TokenLimit == 0 {
		config.TokenLimit = 5000 // 默认 5k tokens（优化：降低阈值以减少 token 消耗）
	}

	// 默认启用路径验证
	enablePathValidation := config.EnablePathValidation
	if !enablePathValidation && len(config.AllowedPathPrefixes) == 0 {
		// 只有在明确禁用且没有指定前缀时才不验证
		enablePathValidation = false
	}

	m := &FilesystemMiddleware{
		BaseMiddleware:          NewBaseMiddleware("filesystem", 100),
		backend:                 config.Backend,
		tokenLimit:              config.TokenLimit,
		enableEviction:          config.EnableEviction,
		allowedPathPrefixes:     config.AllowedPathPrefixes,
		enablePathValidation:    enablePathValidation,
		customToolDescriptions:  config.CustomToolDescriptions,
		systemPromptOverride:    config.SystemPromptOverride,
		hasSystemPromptOverride: config.HasSystemPromptOverride,
	}

	// 创建文件系统工具
	m.fsTools = m.createFilesystemTools()

	fsLog.Info(context.Background(), "path validation configured", map[string]any{"enabled": m.enablePathValidation, "prefixes": m.allowedPathPrefixes})
	return m
}

// createFilesystemTools 创建文件系统工具
func (m *FilesystemMiddleware) createFilesystemTools() []tools.Tool {
	var fsTools []tools.Tool

	// 创建基础工具
	readTool, _ := builtin.NewReadTool(nil)
	fsTools = append(fsTools, readTool)

	writeTool, _ := builtin.NewWriteTool(nil)
	fsTools = append(fsTools, writeTool)

	// 创建增强工具（如果有 backend）
	if m.backend != nil {
		fsTools = append(fsTools, &LsTool{backend: m.backend, middleware: m})
		fsTools = append(fsTools, &EditTool{backend: m.backend, middleware: m})
		fsTools = append(fsTools, &GlobTool{backend: m.backend, middleware: m})
		fsTools = append(fsTools, &GrepTool{backend: m.backend, middleware: m})
	}

	fsLog.Info(context.Background(), "created filesystem tools", map[string]any{"count": len(fsTools)})
	return fsTools
}

// Tools 返回文件系统工具
func (m *FilesystemMiddleware) Tools() []tools.Tool {
	return m.fsTools
}

// WrapModelCall 包装模型调用
func (m *FilesystemMiddleware) WrapModelCall(ctx context.Context, req *ModelRequest, handler ModelCallHandler) (*ModelResponse, error) {
	// 确定要使用的系统提示词
	// 如果 systemPromptOverride 被设置（包括空字符串），使用它
	// 否则使用默认的 FILESYSTEM_SYSTEM_PROMPT
	prompt := m.systemPromptOverride
	if !m.hasSystemPromptOverride {
		prompt = FILESYSTEM_SYSTEM_PROMPT
	}

	// 只有当 prompt 非空时才增强系统提示词
	if prompt != "" {
		if req.SystemPrompt != "" {
			req.SystemPrompt += "\n\n" + prompt
		} else {
			req.SystemPrompt = prompt
		}
	}

	// 调用下一层
	return handler(ctx, req)
}

// WrapToolCall 包装工具调用
func (m *FilesystemMiddleware) WrapToolCall(ctx context.Context, req *ToolCallRequest, handler ToolCallHandler) (*ToolCallResponse, error) {
	// 执行工具调用
	resp, err := handler(ctx, req)
	if err != nil {
		return resp, err
	}

	// 检查是否需要驱逐大结果
	if m.enableEviction && m.backend != nil {
		resp = m.evictLargeResults(ctx, req, resp)
	}

	return resp, nil
}

// evictLargeResults 驱逐大结果到文件
func (m *FilesystemMiddleware) evictLargeResults(ctx context.Context, req *ToolCallRequest, resp *ToolCallResponse) *ToolCallResponse {
	// 简单估算: 1 token ≈ 4 chars
	resultStr := fmt.Sprintf("%v", resp.Result)
	estimatedTokens := len(resultStr) / 4

	if estimatedTokens > m.tokenLimit {
		// 生成文件路径
		path := fmt.Sprintf("/large_tool_results/%s.txt", req.ToolCallID)

		// 写入文件
		if _, err := m.backend.Write(ctx, path, resultStr); err == nil {
			// 返回简化结果
			lines := splitLines(resultStr, 10)
			preview := ""
			if len(lines) > 10 {
				preview = joinLines(lines[:10])
			} else {
				preview = resultStr
			}

			resp.Result = map[string]any{
				"ok":      true,
				"evicted": true,
				"path":    path,
				"message": fmt.Sprintf("Result too large (%d tokens), saved to %s", estimatedTokens, path),
				"preview": preview,
			}

			fsLog.Info(ctx, "evicted large result", map[string]any{"tokens": estimatedTokens, "path": path})
		}
	}

	return resp
}

// FILESYSTEM_SYSTEM_PROMPT 文件系统提示词
const FILESYSTEM_SYSTEM_PROMPT = `### Filesystem Tools

You have access to the following filesystem tools:

- **Read**: Read file contents with optional offset/limit
- **Write**: Write content to a file
- **Edit**: Edit files using string replacement
- **Ls**: List directory contents
- **Glob**: Find files matching glob patterns
- **Grep**: Search for patterns in files

Guidelines:
- Always use relative paths from the sandbox root
- Large results will be automatically saved to files
- Use Edit for precise modifications
- Use Glob and Grep for code exploration`

// 辅助函数
func splitLines(s string, limit int) []string {
	lines := []string{}
	current := ""
	for _, r := range s {
		current += string(r)
		if r == '\n' {
			lines = append(lines, current)
			current = ""
			if len(lines) >= limit {
				break
			}
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func joinLines(lines []string) string {
	result := ""
	var resultSb223 strings.Builder
	for _, line := range lines {
		resultSb223.WriteString(line)
	}
	result += resultSb223.String()
	return result
}

// validatePath 验证路径安全性
// 参考: deepagents/filesystem.py:87-129
func (m *FilesystemMiddleware) validatePath(path string) (string, error) {
	if !m.enablePathValidation {
		return path, nil
	}

	// 1. 检查路径遍历攻击
	if strings.Contains(path, "..") {
		return "", fmt.Errorf("路径遍历不允许(包含 '..'): %s", path)
	}

	if strings.HasPrefix(path, "~") {
		return "", fmt.Errorf("路径遍历不允许(以 '~' 开头): %s", path)
	}

	// 2. 规范化路径
	normalized := filepath.Clean(path)
	// 转换为 Unix 风格路径(统一使用 /)
	normalized = filepath.ToSlash(normalized)

	// 3. 确保路径以 / 开头
	if !strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}

	// 4. 检查允许的前缀
	if len(m.allowedPathPrefixes) > 0 {
		allowed := false
		for _, prefix := range m.allowedPathPrefixes {
			// 规范化前缀(去掉尾部斜杠)
			normalizedPrefix := filepath.Clean(prefix)
			normalizedPrefix = filepath.ToSlash(normalizedPrefix)
			if !strings.HasPrefix(normalizedPrefix, "/") {
				normalizedPrefix = "/" + normalizedPrefix
			}

			// 检查前缀匹配(normalized == prefix 或 normalized 在 prefix 下)
			if normalized == normalizedPrefix || strings.HasPrefix(normalized, normalizedPrefix+"/") {
				allowed = true
				break
			}
		}
		if !allowed {
			return "", fmt.Errorf("路径必须以以下前缀之一开头 %v: %s", m.allowedPathPrefixes, normalized)
		}
	}

	return normalized, nil
}
