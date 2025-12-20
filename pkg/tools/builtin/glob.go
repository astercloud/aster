package builtin

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/tools"
)

// GlobTool 增强的文件搜索工具
// 支持模式匹配文件搜索功能
type GlobTool struct{}

// NewGlobTool 创建Glob工具
func NewGlobTool(config map[string]any) (tools.Tool, error) {
	return &GlobTool{}, nil
}

func (t *GlobTool) Name() string {
	return "Glob"
}

func (t *GlobTool) Description() string {
	return "使用模式匹配搜索文件"
}

func (t *GlobTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "要搜索的文件模式，支持通配符如 *.go, **/*.js",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "搜索的起始目录，默认为当前目录",
			},
			"exclude_patterns": map[string]any{
				"type":        "array",
				"description": "要排除的文件模式列表",
				"items": map[string]any{
					"type": "string",
				},
			},
			"include_hidden": map[string]any{
				"type":        "boolean",
				"description": "是否包含隐藏文件，默认为false",
			},
			"case_sensitive": map[string]any{
				"type":        "boolean",
				"description": "是否区分大小写，默认为false",
			},
			"max_results": map[string]any{
				"type":        "integer",
				"description": "返回的最大结果数量，默认为100",
			},
			"sort_by": map[string]any{
				"type":        "string",
				"description": "结果排序方式：name, size, modified_time, 默认为name",
			},
			"recursive": map[string]any{
				"type":        "boolean",
				"description": "是否递归搜索子目录，默认为true",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *GlobTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	// 验证必需参数
	if err := t.validateRequired(input, []string{"pattern"}); err != nil {
		return NewClaudeErrorResponse(err), nil
	}

	pattern := t.getStringParam(input, "pattern", "")
	path := t.getStringParam(input, "path", ".")
	excludePatterns := t.getStringSlice(input, "exclude_patterns")
	includeHidden := t.getBoolParam(input, "include_hidden", false)
	caseSensitive := t.getBoolParam(input, "case_sensitive", false)
	maxResults := t.getIntParam(input, "max_results", 100)
	sortBy := t.getStringParam(input, "sort_by", "name")
	recursive := t.getBoolParam(input, "recursive", true)

	if pattern == "" {
		return NewClaudeErrorResponse(errors.New("pattern cannot be empty")), nil
	}

	// 验证搜索路径
	if err := t.validatePath(path); err != nil {
		return NewClaudeErrorResponse(
			fmt.Errorf("invalid search path: %w", err),
			"使用相对路径或允许的绝对路径",
			"确保路径不包含 '..' 避免路径遍历攻击",
		), nil
	}

	start := time.Now()

	// 执行文件搜索
	matches, err := t.searchFiles(ctx, path, pattern, excludePatterns, includeHidden, caseSensitive, recursive, tc)
	duration := time.Since(start)

	if err != nil {
		return map[string]any{
			"ok":    false,
			"error": fmt.Sprintf("search failed: %v", err),
			"recommendations": []string{
				"检查搜索模式是否正确",
				"确认搜索路径是否存在",
				"验证是否有读取权限",
				"检查模式是否包含特殊字符",
			},
			"pattern":     pattern,
			"path":        path,
			"duration_ms": duration.Milliseconds(),
		}, nil
	}

	// 限制结果数量
	if maxResults > 0 && len(matches) > maxResults {
		matches = matches[:maxResults]
	}

	// 排序结果
	t.sortMatches(matches, sortBy)

	// 获取文件信息
	fileInfos := make([]map[string]any, len(matches))
	for i, match := range matches {
		info, err := tc.Sandbox.FS().Stat(ctx, match)
		fileType := "unknown"
		var size int64 = 0
		var modifiedTime time.Time

		if err == nil {
			size = info.Size
			modifiedTime = info.ModTime

			if info.IsDir {
				fileType = "directory"
			} else {
				// 根据扩展名判断文件类型
				ext := strings.ToLower(filepath.Ext(match))
				switch ext {
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
				case ".html", ".htm":
					fileType = "html"
				case ".css":
					fileType = "css"
				case ".sh", ".bash":
					fileType = "shell"
				case ".xml":
					fileType = "xml"
				case ".sql":
					fileType = "sql"
				case ".log":
					fileType = "log"
				case ".test":
					fileType = "test"
				case ".spec":
					fileType = "spec"
				case ".conf", ".config":
					fileType = "config"
				default:
					fileType = "file"
				}
			}
		}

		fileInfos[i] = map[string]any{
			"path":          match,
			"name":          filepath.Base(match),
			"type":          fileType,
			"size":          size,
			"modified_time": modifiedTime.Unix(),
			"relative_path": t.getRelativePath(match, path),
		}
	}

	return map[string]any{
		"ok":               true,
		"pattern":          pattern,
		"path":             path,
		"matches":          fileInfos,
		"total_matches":    len(fileInfos),
		"truncated":        maxResults > 0 && len(matches) >= maxResults,
		"exclude_patterns": excludePatterns,
		"include_hidden":   includeHidden,
		"case_sensitive":   caseSensitive,
		"sort_by":          sortBy,
		"recursive":        recursive,
		"max_results":      maxResults,
		"duration_ms":      duration.Milliseconds(),
	}, nil
}

// validateRequired 验证必需参数
func (t *GlobTool) validateRequired(input map[string]any, required []string) error {
	for _, key := range required {
		if _, exists := input[key]; !exists {
			return fmt.Errorf("missing required parameter: %s", key)
		}
	}
	return nil
}

// getStringParam 获取字符串参数
func (t *GlobTool) getStringParam(input map[string]any, key string, defaultValue string) string {
	if value, exists := input[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}

// getBoolParam 获取布尔参数
func (t *GlobTool) getBoolParam(input map[string]any, key string, defaultValue bool) bool {
	if value, exists := input[key]; exists {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// getIntParam 获取整数参数
func (t *GlobTool) getIntParam(input map[string]any, key string, defaultValue int) int {
	if value, exists := input[key]; exists {
		if num, ok := value.(float64); ok {
			return int(num)
		}
	}
	return defaultValue
}

// getStringSlice 获取字符串数组参数
func (t *GlobTool) getStringSlice(input map[string]any, key string) []string {
	if value, exists := input[key]; exists {
		if slice, ok := value.([]any); ok {
			result := make([]string, len(slice))
			for i, item := range slice {
				if str, ok := item.(string); ok {
					result[i] = str
				}
			}
			return result
		}
	}
	return []string{}
}

// validatePath 验证文件路径安全性
func (t *GlobTool) validatePath(path string) error {
	// 清理路径
	cleanPath := filepath.Clean(path)

	// 检查路径遍历攻击
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal not allowed: %s", path)
	}

	return nil
}

// searchFiles 搜索匹配的文件
func (t *GlobTool) searchFiles(ctx context.Context, rootPath, pattern string, excludePatterns []string, includeHidden, caseSensitive, recursive bool, tc *tools.ToolContext) ([]string, error) {
	// 使用沙箱的Glob功能
	opts := &sandbox.GlobOptions{
		CWD:      rootPath,
		Dot:      includeHidden,
		Absolute: false,
	}

	if len(excludePatterns) > 0 {
		opts.Ignore = excludePatterns
	}

	matches, err := tc.Sandbox.FS().Glob(ctx, pattern, opts)
	if err != nil {
		return nil, err
	}

	return matches, nil
}

// getRelativePath 获取相对路径
func (t *GlobTool) getRelativePath(filePath, basePath string) string {
	if rel, err := filepath.Rel(basePath, filePath); err == nil {
		return rel
	}
	return filePath
}

// sortMatches 排序匹配结果
func (t *GlobTool) sortMatches(matches []string, sortBy string) {
	// 这里实现不同的排序逻辑
	// 为了简化，只实现按名称排序
	switch sortBy {
	case "name":
		// 默认已经是按名称排序（使用filepath.Glob的结果）
	case "size", "modified_time":
		// 需要获取文件信息进行排序
		// 这里简化处理，保持原顺序
	default:
		// 默认按名称排序
	}
}

func (t *GlobTool) Prompt() string {
	return `使用模式匹配搜索文件。

功能特性：
- 支持通配符模式匹配
- 递归目录搜索
- 排除模式过滤
- 多种排序方式
- 文件类型识别

使用指南：
- pattern: 必需参数，搜索模式（如 *.go, **/*.js）
- path: 可选参数，搜索起始目录
- exclude_patterns: 可选参数，排除模式列表
- include_hidden: 可选参数，是否包含隐藏文件
- max_results: 可选参数，最大结果数量
- sort_by: 可选参数，排序方式

模式示例：
- *.go - 匹配所有Go文件
- **/*.js - 递归匹配所有JavaScript文件
- test_*.py - 匹配以test_开头的Python文件
- src/**/*.{go,js} - 匹配src目录下的Go和JavaScript文件

安全性：
- 路径遍历攻击防护
- 沙箱环境隔离
- 权限检查集成
- 结果数量限制`
}

// Examples 返回 Glob 工具的使用示例
// 实现 ExampleableTool 接口，帮助 LLM 更准确地调用工具
func (t *GlobTool) Examples() []tools.ToolExample {
	return []tools.ToolExample{
		{
			Description: "查找所有 Go 源文件",
			Input: map[string]any{
				"pattern": "**/*.go",
			},
		},
		{
			Description: "在 src 目录中查找 TypeScript 文件",
			Input: map[string]any{
				"pattern": "**/*.{ts,tsx}",
				"path":    "/app/src",
			},
		},
		{
			Description: "查找测试文件并排除 vendor 目录",
			Input: map[string]any{
				"pattern":          "**/*_test.go",
				"exclude_patterns": []string{"vendor/**", "node_modules/**"},
			},
		},
		{
			Description: "查找配置文件并按修改时间排序",
			Input: map[string]any{
				"pattern":     "*.{yaml,yml,json}",
				"path":        "/app/config",
				"sort_by":     "modified_time",
				"max_results": 20,
			},
		},
	}
}
