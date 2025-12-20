package builtin

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/tools"
)

var writeLog = logging.ForComponent("WriteTool")

// WriteTool 增强的文件写入工具
// 兼容标准Write工具功能
type WriteTool struct{}

// NewWriteTool 创建文件写入工具
func NewWriteTool(config map[string]any) (tools.Tool, error) {
	return &WriteTool{}, nil
}

func (t *WriteTool) Name() string {
	return "Write" // 使用标准的工具名称
}

func (t *WriteTool) Description() string {
	return "向本地文件系统写入文件内容"
}

func (t *WriteTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "要写入的文件路径，必须是绝对路径",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "要写入文件的内容",
			},
			"create_dirs": map[string]any{
				"type":        "boolean",
				"description": "如果目录不存在，是否自动创建父目录，默认为true",
			},
			"backup": map[string]any{
				"type":        "boolean",
				"description": "在覆盖现有文件前是否创建备份，默认为false",
			},
			"append": map[string]any{
				"type":        "boolean",
				"description": "是否以追加模式写入，默认为false（覆盖模式）",
			},
		},
		"required": []string{"file_path", "content"},
	}
}

func (t *WriteTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	// 验证必需参数
	if err := ValidateRequired(input, []string{"file_path", "content"}); err != nil {
		return NewClaudeErrorResponse(err), nil
	}

	filePath := GetStringParam(input, "file_path", "")
	content := GetStringParam(input, "content", "")
	createDirs := GetBoolParam(input, "create_dirs", true)
	backup := GetBoolParam(input, "backup", false)
	append := GetBoolParam(input, "append", false)

	if filePath == "" {
		return NewClaudeErrorResponse(errors.New("file_path cannot be empty")), nil
	}

	// 验证文件路径安全性
	if err := t.validatePath(filePath); err != nil {
		return NewClaudeErrorResponse(
			fmt.Errorf("invalid file path: %w", err),
			"使用相对路径或允许的绝对路径",
			"确保路径不包含 '..' 避免路径遍历攻击",
		), nil
	}

	start := time.Now()

	// 检查文件是否已存在
	existingContent := ""
	fileExists := false
	if content, err := tc.Sandbox.FS().Read(ctx, filePath); err == nil {
		existingContent = content
		fileExists = true
	}

	// 如果是追加模式且文件存在，在内容前添加换行符（如果需要）
	var writeContent string
	if append && fileExists && existingContent != "" && !strings.HasSuffix(existingContent, "\n") {
		writeContent = existingContent + "\n" + content
	} else if append && fileExists {
		writeContent = existingContent + content
	} else {
		writeContent = content
	}

	// 如果需要备份，创建备份文件
	var backupPath string
	if backup && fileExists {
		backupPath = t.createBackup(ctx, filePath, existingContent, tc)
	}

	// 确保父目录存在
	if createDirs {
		dir := filepath.Dir(filePath)
		if dir != "." && dir != "/" {
			// 创建目录（沙箱会处理）
			if err := tc.Sandbox.FS().Write(ctx, dir+"/.mkdir_marker", ""); err != nil {
				writeLog.Warn(ctx, "failed to create directory marker", map[string]any{"dir": dir, "error": err})
			}
		}
	}

	// 执行写入操作
	err := tc.Sandbox.FS().Write(ctx, filePath, writeContent)
	duration := time.Since(start)

	if err != nil {
		return map[string]any{
			"ok":    false,
			"error": fmt.Sprintf("failed to write file: %v", err),
			"recommendations": []string{
				"检查文件路径是否正确",
				"确认是否有写入权限",
				"确保目录存在或启用 create_dirs",
				"检查磁盘空间是否充足",
				"验证文件是否被其他进程锁定",
			},
			"file_path":   filePath,
			"duration_ms": duration.Milliseconds(),
		}, nil
	}

	// 获取文件信息
	fileSize := len(writeContent)
	lines := 0
	if writeContent != "" {
		lines = len(strings.Split(writeContent, "\n"))
	}

	fileExt := strings.ToLower(filepath.Ext(filePath))
	fileType := "unknown"
	switch fileExt {
	case ".go":
		fileType = "go"
	case ".js", ".jsx":
		fileType = "javascript"
	case ".ts", ".tsx":
		fileType = "typescript"
	case ".py":
		fileType = "python"
	case ".java":
		fileType = "java"
	case ".md", ".markdown":
		fileType = "markdown"
	case ".json":
		fileType = "json"
	case ".yaml", ".yml":
		fileType = "yaml"
	case ".txt":
		fileType = "text"
	}

	// 构建响应
	response := map[string]any{
		"ok":           true,
		"file_path":    filePath,
		"content":      content,
		"lines":        lines,
		"file_size":    fileSize,
		"file_type":    fileType,
		"duration_ms":  duration.Milliseconds(),
		"append_mode":  append,
		"created_dirs": createDirs,
	}

	// 添加备份信息
	if backupPath != "" {
		response["backup_path"] = backupPath
		response["backup_created"] = true
	} else if backup {
		response["backup_created"] = false
		response["backup_reason"] = "original file did not exist"
	}

	// 添加文件状态信息
	if fileExists {
		response["file_existed"] = true
		response["operation"] = "overwritten"
		if append {
			response["operation"] = "appended"
		}
	} else {
		response["file_existed"] = false
		response["operation"] = "created"
	}

	return response, nil
}

// validatePath 验证文件路径安全性
func (t *WriteTool) validatePath(path string) error {
	// 清理路径
	cleanPath := filepath.Clean(path)

	// 检查路径遍历攻击
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal not allowed: %s", path)
	}

	return nil
}

// createBackup 创建文件备份
func (t *WriteTool) createBackup(ctx context.Context, filePath, content string, tc *tools.ToolContext) string {
	timestamp := time.Now().Format("20060102_150405")
	ext := filepath.Ext(filePath)
	base := strings.TrimSuffix(filePath, ext)
	backupPath := fmt.Sprintf("%s.backup_%s%s", base, timestamp, ext)

	err := tc.Sandbox.FS().Write(ctx, backupPath, content)
	if err != nil {
		return ""
	}

	return backupPath
}

func (t *WriteTool) Prompt() string {
	return `向本地文件系统写入文件内容。

功能特性：
- 支持创建父目录
- 文件备份功能
- 追加模式和覆盖模式
- 安全的路径验证
- 详细的文件信息统计

使用指南：
- file_path: 必需参数，目标文件路径
- content: 必需参数，要写入的内容
- create_dirs: 可选参数，是否创建父目录
- backup: 可选参数，是否创建备份
- append: 可选参数，是否追加模式

安全性：
- 路径遍历攻击防护
- 沙箱环境隔离
- 文件备份保护
- 写入权限检查`
}

// Examples 返回 Write 工具的使用示例
// 实现 ExampleableTool 接口，帮助 LLM 更准确地调用工具
func (t *WriteTool) Examples() []tools.ToolExample {
	return []tools.ToolExample{
		{
			Description: "创建新的配置文件",
			Input: map[string]any{
				"file_path": "/app/config.yaml",
				"content":   "server:\n  port: 8080\n  host: localhost",
			},
		},
		{
			Description: "追加日志到文件",
			Input: map[string]any{
				"file_path": "/var/log/app.log",
				"content":   "2024-01-15 10:30:00 INFO Application started",
				"append":    true,
			},
		},
		{
			Description: "创建文件并备份已存在的版本",
			Input: map[string]any{
				"file_path": "/app/src/main.go",
				"content":   "package main\n\nfunc main() {\n\tprintln(\"Hello\")\n}",
				"backup":    true,
			},
		},
	}
}
