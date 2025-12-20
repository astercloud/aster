package builtin

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/astercloud/aster/pkg/tools"
)

// EditTool 增强的文件编辑工具
// 支持精确的字符串替换编辑功能
type EditTool struct{}

// NewEditTool 创建Edit工具
func NewEditTool(config map[string]any) (tools.Tool, error) {
	return &EditTool{}, nil
}

func (t *EditTool) Name() string {
	return "Edit"
}

func (t *EditTool) Description() string {
	return "对文件进行精确的字符串替换编辑"
}

func (t *EditTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "要编辑的文件路径，必须是绝对路径",
			},
			"old_string": map[string]any{
				"type":        "string",
				"description": "要被替换的原始字符串，必须精确匹配",
			},
			"new_string": map[string]any{
				"type":        "string",
				"description": "替换后的新字符串",
			},
			"replace_all": map[string]any{
				"type":        "boolean",
				"description": "是否替换所有匹配的实例，默认为false（只替换第一个匹配项）",
			},
			"preserve_indentation": map[string]any{
				"type":        "boolean",
				"description": "是否保持原始缩进格式，默认为true",
			},
			"backup": map[string]any{
				"type":        "boolean",
				"description": "在编辑前是否创建备份，默认为true",
			},
		},
		"required": []string{"file_path", "old_string", "new_string"},
	}
}

func (t *EditTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	// 验证必需参数
	if err := t.validateRequired(input, []string{"file_path", "old_string", "new_string"}); err != nil {
		return NewClaudeErrorResponse(err), nil
	}

	filePath := t.getStringParam(input, "file_path", "")
	oldString := t.getStringParam(input, "old_string", "")
	newString := t.getStringParam(input, "new_string", "")
	replaceAll := t.getBoolParam(input, "replace_all", false)
	preserveIndentation := t.getBoolParam(input, "preserve_indentation", true)
	backup := t.getBoolParam(input, "backup", true)

	if filePath == "" {
		return NewClaudeErrorResponse(errors.New("file_path cannot be empty")), nil
	}

	if oldString == "" {
		return NewClaudeErrorResponse(errors.New("old_string cannot be empty")), nil
	}

	// 如果 old_string 和 new_string 相同，直接返回成功但没有修改
	if oldString == newString {
		return map[string]any{
			"ok":           true,
			"success":      true,
			"changes_made": 0,
			"file_path":    filePath,
			"message":      "old_string and new_string are identical, no changes made",
			"duration_ms":  0,
		}, nil
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

	// 读取原文件内容
	originalContent, err := tc.Sandbox.FS().Read(ctx, filePath)
	if err != nil {
		return map[string]any{
			"ok":    false,
			"error": fmt.Sprintf("failed to read file: %v", err),
			"recommendations": []string{
				"检查文件路径是否正确",
				"确认文件是否存在",
				"验证是否有读取权限",
			},
			"file_path":   filePath,
			"duration_ms": time.Since(start).Milliseconds(),
		}, nil
	}

	// 创建备份
	var backupPath string
	if backup {
		backupPath = t.createBackup(ctx, filePath, originalContent, tc)
	}

	// 执行字符串替换
	var modifiedContent string
	var replacements int

	if replaceAll {
		modifiedContent = strings.ReplaceAll(originalContent, oldString, newString)
		// 计算替换次数
		replacements = strings.Count(originalContent, oldString)
	} else {
		// 只替换第一个匹配项
		if strings.Contains(originalContent, oldString) {
			modifiedContent = strings.Replace(originalContent, oldString, newString, 1)
			replacements = 1
		} else {
			modifiedContent = originalContent
			replacements = 0
		}
	}

	duration := time.Since(start)

	// 检查是否发生了替换
	if replacements == 0 {
		return map[string]any{
			"ok":    false,
			"error": "old_string not found in file",
			"recommendations": []string{
				"检查old_string是否与文件内容完全匹配（包括空白字符和换行符）",
				"确认old_string的大小写是否正确",
				"尝试使用更小的字符串片段进行匹配",
				"检查是否存在不可见字符或编码问题",
			},
			"file_path":         filePath,
			"old_string":        oldString,
			"old_string_length": len(oldString),
			"content_length":    len(originalContent),
			"duration_ms":       duration.Milliseconds(),
			"backup_path":       backupPath,
		}, nil
	}

	// 如果启用缩进保护，调整新字符串的缩进
	if preserveIndentation {
		newString = t.adjustIndentation(oldString, newString)
		// 重新执行替换
		if replaceAll {
			modifiedContent = strings.ReplaceAll(originalContent, oldString, newString)
		} else {
			modifiedContent = strings.Replace(originalContent, oldString, newString, 1)
		}
	}

	// 写入修改后的内容
	err = tc.Sandbox.FS().Write(ctx, filePath, modifiedContent)
	if err != nil {
		return map[string]any{
			"ok":    false,
			"error": fmt.Sprintf("failed to write modified content: %v", err),
			"recommendations": []string{
				"检查文件写入权限",
				"确认磁盘空间充足",
				"检查文件是否被其他进程锁定",
			},
			"file_path":   filePath,
			"duration_ms": time.Since(start).Milliseconds(),
			"backup_path": backupPath,
		}, nil
	}

	// 计算统计信息
	originalLines := strings.Count(originalContent, "\n") + 1
	modifiedLines := strings.Count(modifiedContent, "\n") + 1
	lineDifference := modifiedLines - originalLines
	sizeDifference := len(modifiedContent) - len(originalContent)

	return map[string]any{
		"ok":                   true,
		"success":              true,
		"changes_made":         replacements,
		"file_path":            filePath,
		"old_string":           oldString,
		"new_string":           newString,
		"replacements":         replacements,
		"replace_all":          replaceAll,
		"preserve_indentation": preserveIndentation,
		"original_lines":       originalLines,
		"modified_lines":       modifiedLines,
		"line_difference":      lineDifference,
		"original_size":        len(originalContent),
		"modified_size":        len(modifiedContent),
		"size_difference":      sizeDifference,
		"duration_ms":          duration.Milliseconds(),
		"backup_path":          backupPath,
		"backup_created":       backupPath != "",
	}, nil
}

// validateRequired 验证必需参数
func (t *EditTool) validateRequired(input map[string]any, required []string) error {
	for _, key := range required {
		if _, exists := input[key]; !exists {
			return fmt.Errorf("missing required parameter: %s", key)
		}
	}
	return nil
}

// getStringParam 获取字符串参数
func (t *EditTool) getStringParam(input map[string]any, key string, defaultValue string) string {
	if value, exists := input[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}

// getBoolParam 获取布尔参数
func (t *EditTool) getBoolParam(input map[string]any, key string, defaultValue bool) bool {
	if value, exists := input[key]; exists {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// validatePath 验证文件路径安全性
func (t *EditTool) validatePath(path string) error {
	// 清理路径
	cleanPath := filepath.Clean(path)

	// 检查路径遍历攻击
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal not allowed: %s", path)
	}

	return nil
}

// adjustIndentation 调整新字符串的缩进以匹配旧字符串
func (t *EditTool) adjustIndentation(oldString, newString string) string {
	// 提取旧字符串的缩进
	oldLines := strings.Split(oldString, "\n")
	newLines := strings.Split(newString, "\n")

	if len(oldLines) == 0 || len(newLines) == 0 {
		return newString
	}

	// 获取旧字符串第一行的缩进
	oldIndentation := t.getIndentation(oldLines[0])

	// 获取新字符串第一行的缩进
	newIndentation := t.getIndentation(newLines[0])

	// 如果缩进不同，调整新字符串的所有行
	if oldIndentation != newIndentation {
		adjustedLines := make([]string, len(newLines))
		for i, line := range newLines {
			// 移除新字符串的当前缩进
			withoutIndent := strings.TrimPrefix(line, newIndentation)
			// 添加旧字符串的缩进
			adjustedLines[i] = oldIndentation + withoutIndent
		}
		return strings.Join(adjustedLines, "\n")
	}

	return newString
}

// getIndentation 获取行的缩进（空格和制表符）
func (t *EditTool) getIndentation(line string) string {
	var indentation []rune
	for _, char := range line {
		if char == ' ' || char == '\t' {
			indentation = append(indentation, char)
		} else {
			break
		}
	}
	return string(indentation)
}

// createBackup 创建文件备份
func (t *EditTool) createBackup(ctx context.Context, filePath, content string, tc *tools.ToolContext) string {
	timestamp := time.Now().Format("20060102_150405")
	backupPath := filePath + ".backup_" + timestamp

	err := tc.Sandbox.FS().Write(ctx, backupPath, content)
	if err != nil {
		return ""
	}

	return backupPath
}

func (t *EditTool) Prompt() string {
	return `对文件进行精确的字符串替换编辑。

功能特性：
- 精确的字符串替换
- 支持全部替换或单个替换
- 智能缩进保护
- 自动备份功能
- 详细的编辑统计

使用指南：
- file_path: 必需参数，目标文件路径
- old_string: 必需参数，要被替换的原始字符串
- new_string: 必需参数，替换后的新字符串
- replace_all: 可选参数，是否替换所有匹配项
- preserve_indentation: 可选参数，是否保持缩进格式
- backup: 可选参数，是否创建备份

注意事项：
- old_string必须与文件内容完全匹配（包括空白字符）
- 建议先使用Read工具确认要替换的内容
- 启用备份可以防止意外的编辑错误
- 缩进保护可以保持代码格式的一致性

安全性：
- 路径遍历攻击防护
- 自动备份保护
- 编辑前验证
- 详细的操作日志`
}

// Examples 返回 Edit 工具的使用示例
// 实现 ExampleableTool 接口，帮助 LLM 更准确地调用工具
func (t *EditTool) Examples() []tools.ToolExample {
	return []tools.ToolExample{
		{
			Description: "修改函数名称",
			Input: map[string]any{
				"file_path":  "/app/src/utils.go",
				"old_string": "func oldFunction(",
				"new_string": "func newFunction(",
			},
		},
		{
			Description: "替换所有导入路径",
			Input: map[string]any{
				"file_path":   "/app/src/main.go",
				"old_string":  "github.com/old/package",
				"new_string":  "github.com/new/package",
				"replace_all": true,
			},
		},
		{
			Description: "修改配置值并创建备份",
			Input: map[string]any{
				"file_path":  "/app/config.yaml",
				"old_string": "port: 8080",
				"new_string": "port: 9090",
				"backup":     true,
			},
		},
		{
			Description: "修改多行代码块",
			Input: map[string]any{
				"file_path":  "/app/src/handler.go",
				"old_string": "if err != nil {\n\treturn err\n}",
				"new_string": "if err != nil {\n\tlog.Error(err)\n\treturn fmt.Errorf(\"handler error: %w\", err)\n}",
			},
		},
	}
}
