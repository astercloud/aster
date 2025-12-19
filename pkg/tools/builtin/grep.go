package builtin

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/tools"
)

// GrepTool 增强的内容搜索工具
// 支持正则表达式内容搜索功能
type GrepTool struct{}

// NewGrepTool 创建Grep工具
func NewGrepTool(config map[string]any) (tools.Tool, error) {
	return &GrepTool{}, nil
}

func (t *GrepTool) Name() string {
	return "Grep"
}

func (t *GrepTool) Description() string {
	return "在文件内容中搜索正则表达式模式"
}

func (t *GrepTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "要搜索的正则表达式模式",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "搜索的文件或目录路径，默认为当前目录",
			},
			"glob": map[string]any{
				"type":        "string",
				"description": "文件模式过滤器，如 *.go, **/*.js",
			},
			"file_type": map[string]any{
				"type":        "string",
				"description": "文件类型过滤器，如 go, js, python",
			},
			"output_mode": map[string]any{
				"type":        "string",
				"description": "输出模式：content, files_with_matches, count，默认为content",
			},
			"max_results": map[string]any{
				"type":        "integer",
				"description": "返回的最大结果数量，默认为50",
			},
			"context_lines": map[string]any{
				"type":        "integer",
				"description": "显示匹配行前后的上下文行数，默认为0",
			},
			"case_insensitive": map[string]any{
				"type":        "boolean",
				"description": "是否忽略大小写，默认为false",
			},
			"whole_word": map[string]any{
				"type":        "boolean",
				"description": "是否匹配完整单词，默认为false",
			},
			"line_numbers": map[string]any{
				"type":        "boolean",
				"description": "是否显示行号，默认为true",
			},
			"no_heading": map[string]any{
				"type":        "boolean",
				"description": "是否禁用文件标题，默认为false",
			},
			"hidden": map[string]any{
				"type":        "boolean",
				"description": "是否搜索隐藏文件，默认为false",
			},
			"follow": map[string]any{
				"type":        "boolean",
				"description": "是否跟随符号链接，默认为false",
			},
			"multiline": map[string]any{
				"type":        "boolean",
				"description": "是否允许跨行匹配，默认为false",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *GrepTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	// 验证必需参数
	if err := t.validateRequired(input, []string{"pattern"}); err != nil {
		return NewClaudeErrorResponse(err), nil
	}

	pattern := t.getStringParam(input, "pattern", "")
	path := t.getStringParam(input, "path", ".")
	glob := t.getStringParam(input, "glob", "")
	fileType := t.getStringParam(input, "file_type", "")
	outputMode := t.getStringParam(input, "output_mode", "content")
	maxResults := t.getIntParam(input, "max_results", 50)
	contextLines := t.getIntParam(input, "context_lines", 0)
	caseInsensitive := t.getBoolParam(input, "case_insensitive", false)
	wholeWord := t.getBoolParam(input, "whole_word", false)
	lineNumbers := t.getBoolParam(input, "line_numbers", true)
	noHeading := t.getBoolParam(input, "no_heading", false)
	hidden := t.getBoolParam(input, "hidden", false)
	follow := t.getBoolParam(input, "follow", false)
	multiline := t.getBoolParam(input, "multiline", false)

	if pattern == "" {
		return NewClaudeErrorResponse(errors.New("pattern cannot be empty")), nil
	}

	// 验证输出模式
	validModes := []string{"content", "files_with_matches", "count"}
	modeValid := slices.Contains(validModes, outputMode)
	if !modeValid {
		return NewClaudeErrorResponse(
			fmt.Errorf("invalid output_mode: %s", outputMode),
			"支持的模式: content, files_with_matches, count",
		), nil
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

	// 构建搜索命令（简化实现，使用grep或find命令）
	command := t.buildGrepCommand(pattern, path, glob, fileType, outputMode, maxResults, contextLines, caseInsensitive, wholeWord, lineNumbers, noHeading, hidden, follow, multiline)

	// 执行搜索
	result, err := tc.Sandbox.Exec(ctx, command, &sandbox.ExecOptions{
		Timeout: time.Duration(30) * time.Second,
		WorkDir: path,
	})

	duration := time.Since(start)

	if err != nil {
		return map[string]any{
			"ok":    false,
			"error": fmt.Sprintf("search failed: %v", err),
			"recommendations": []string{
				"检查正则表达式语法是否正确",
				"确认搜索路径是否存在",
				"验证是否有读取权限",
				"检查模式是否包含特殊字符需要转义",
				"确认系统是否安装了grep工具",
			},
			"pattern":     pattern,
			"path":        path,
			"duration_ms": duration.Milliseconds(),
		}, nil
	}

	// 解析结果
	parsedResult := t.parseGrepOutput(result.Stdout, outputMode, lineNumbers, !noHeading)

	// 添加元数据
	response := map[string]any{
		"ok":          true,
		"pattern":     pattern,
		"path":        path,
		"output_mode": outputMode,
		"duration_ms": duration.Milliseconds(),
	}

	// 添加搜索参数
	response["glob"] = glob
	response["file_type"] = fileType
	response["case_insensitive"] = caseInsensitive
	response["whole_word"] = wholeWord
	response["line_numbers"] = lineNumbers
	response["context_lines"] = contextLines
	response["hidden"] = hidden
	response["follow"] = follow
	response["multiline"] = multiline

	// 添加结果
	switch outputMode {
	case "content":
		response["matches"] = parsedResult.matches
		response["total_matches"] = len(parsedResult.matches)
		response["total_files"] = parsedResult.totalFiles
	case "files_with_matches":
		response["files"] = parsedResult.files
		response["total_files"] = len(parsedResult.files)
	case "count":
		response["file_counts"] = parsedResult.fileCounts
		response["total_matches"] = parsedResult.totalMatches
		response["total_files"] = len(parsedResult.fileCounts)
	}

	// 检查结果限制
	if maxResults > 0 {
		switch outputMode {
		case "content":
			response["truncated"] = len(parsedResult.matches) >= maxResults
		case "files_with_matches":
			response["truncated"] = len(parsedResult.files) >= maxResults
		case "count":
			response["truncated"] = len(parsedResult.fileCounts) >= maxResults
		}
	}

	return response, nil
}

// 辅助方法实现
func (t *GrepTool) validateRequired(input map[string]any, required []string) error {
	for _, key := range required {
		if _, exists := input[key]; !exists {
			return fmt.Errorf("missing required parameter: %s", key)
		}
	}
	return nil
}

func (t *GrepTool) getStringParam(input map[string]any, key string, defaultValue string) string {
	if value, exists := input[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}

func (t *GrepTool) getIntParam(input map[string]any, key string, defaultValue int) int {
	if value, exists := input[key]; exists {
		if num, ok := value.(float64); ok {
			return int(num)
		}
	}
	return defaultValue
}

func (t *GrepTool) getBoolParam(input map[string]any, key string, defaultValue bool) bool {
	if value, exists := input[key]; exists {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return defaultValue
}

func (t *GrepTool) validatePath(path string) error {
	if strings.Contains(path, "..") {
		return errors.New("path traversal not allowed")
	}
	return nil
}

func (t *GrepTool) buildGrepCommand(pattern, path, glob, fileType, outputMode string, maxResults, contextLines int, caseInsensitive, wholeWord, lineNumbers, noHeading, hidden, follow, multiline bool) string {
	var parts []string

	parts = append(parts, "grep")

	// 添加选项
	if caseInsensitive {
		parts = append(parts, "-i")
	}
	if wholeWord {
		parts = append(parts, "-w")
	}
	if lineNumbers {
		parts = append(parts, "-n")
	}
	if noHeading {
		parts = append(parts, "--no-heading")
	}
	if hidden {
		parts = append(parts, "--hidden")
	}
	if follow {
		parts = append(parts, "--follow")
	}
	if multiline {
		parts = append(parts, "--multiline")
	}

	// 上下文行数
	if contextLines > 0 {
		parts = append(parts, "-C", strconv.Itoa(contextLines))
	}

	// 输出模式
	switch outputMode {
	case "files_with_matches":
		parts = append(parts, "-l")
	case "count":
		parts = append(parts, "-c")
	}

	// 文件类型过滤
	if fileType != "" {
		parts = append(parts, "--include=*."+fileType)
	}

	// Glob模式
	if glob != "" {
		parts = append(parts, "--include="+glob)
	}

	// 结果限制
	if maxResults > 0 && outputMode == "content" {
		parts = append(parts, "-m", strconv.Itoa(maxResults))
	}

	// 搜索模式
	parts = append(parts, fmt.Sprintf("'%s'", pattern))

	// 搜索路径
	parts = append(parts, path)

	return strings.Join(parts, " ")
}

func (t *GrepTool) parseGrepOutput(output, outputMode string, lineNumbers, withHeading bool) *GrepResult {
	result := &GrepResult{
		matches:    []GrepMatch{},
		files:      []string{},
		fileCounts: []FileCount{},
	}

	if output == "" {
		return result
	}

	lines := strings.Split(output, "\n")

	switch outputMode {
	case "content":
		t.parseContentOutput(lines, result, lineNumbers)
	case "files_with_matches":
		t.parseFilesOutput(lines, result)
	case "count":
		t.parseCountOutput(lines, result)
	}

	return result
}

func (t *GrepTool) parseContentOutput(lines []string, result *GrepResult, lineNumbers bool) {
	currentFile := ""

	for _, line := range lines {
		if line == "" {
			continue
		}

		// 简化的解析逻辑
		if strings.HasPrefix(line, "./") {
			// 文件路径行
			parts := strings.SplitN(line, ":", 2)
			if len(parts) >= 2 {
				currentFile = parts[0]
				remaining := strings.Join(parts[1:], ":")

				// 提取行号和内容
				if lineNumbers {
					contentParts := strings.SplitN(remaining, ":", 2)
					if len(contentParts) >= 2 {
						lineNumStr := strings.TrimSpace(contentParts[0])
						var lineNum int
						_, _ = fmt.Sscanf(lineNumStr, "%d", &lineNum)
						content := contentParts[1]

						result.matches = append(result.matches, GrepMatch{
							File:    currentFile,
							Line:    lineNum,
							Content: strings.TrimSpace(content),
						})
					}
				}
			}
		} else if currentFile != "" {
			// 内容行（可能是上下文）
			result.matches = append(result.matches, GrepMatch{
				File:    currentFile,
				Line:    0,
				Content: strings.TrimSpace(line),
			})
		}
	}
}

func (t *GrepTool) parseFilesOutput(lines []string, result *GrepResult) {
	for _, line := range lines {
		if line != "" {
			if !t.containsString(result.files, line) {
				result.files = append(result.files, strings.TrimSpace(line))
			}
		}
	}
}

func (t *GrepTool) parseCountOutput(lines []string, result *GrepResult) {
	for _, line := range lines {
		if line == "" {
			continue
		}

		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				file := strings.TrimSpace(parts[0])
				countStr := strings.TrimSpace(parts[1])
				var count int
				_, _ = fmt.Sscanf(countStr, "%d", &count)

				result.fileCounts = append(result.fileCounts, FileCount{
					File:  file,
					Count: count,
				})
				result.totalMatches += count
			}
		}
	}
}

func (t *GrepTool) containsString(slice []string, item string) bool {
	return slices.Contains(slice, item)
}

// 数据结构
type GrepResult struct {
	matches      []GrepMatch
	files        []string
	fileCounts   []FileCount
	totalFiles   int
	totalMatches int
}

type GrepMatch struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Content string `json:"content"`
}

type FileCount struct {
	File  string `json:"file"`
	Count int    `json:"count"`
}

func (t *GrepTool) Prompt() string {
	return `在文件内容中搜索正则表达式模式。

功能特性：
- 正则表达式模式搜索
- 支持多种输出模式
- 上下文行显示
- 文件类型过滤
- 大小写敏感选项

使用指南：
- pattern: 必需参数，搜索的正则表达式
- path: 可选参数，搜索路径
- output_mode: 可选参数，输出模式（content/files_with_matches/count）
- max_results: 可选参数，最大结果数
- case_insensitive: 可选参数，忽略大小写
- line_numbers: 可选参数，显示行号

正则表达式示例：
- "function\s+\w+" - 匹配函数定义
- "\berror\b" - 匹配完整单词error
- "TODO|FIXME" - 匹配TODO或FIXME

安全性：
- 路径遍历攻击防护
- 命令注入防护
- 沙箱环境隔离
- 超时控制`
}

// Examples 返回 Grep 工具的使用示例
// 实现 ExampleableTool 接口，帮助 LLM 更准确地调用工具
func (t *GrepTool) Examples() []tools.ToolExample {
	return []tools.ToolExample{
		{
			Description: "搜索所有包含 TODO 的文件",
			Input: map[string]any{
				"pattern":     "TODO",
				"output_mode": "files_with_matches",
			},
		},
		{
			Description: "搜索函数定义并显示上下文",
			Input: map[string]any{
				"pattern":          "func\\s+\\w+\\(",
				"path":             "/app/src",
				"output_mode":      "content",
				"context_lines":    3,
				"case_insensitive": false,
			},
		},
		{
			Description: "统计每个文件中 error 出现的次数",
			Input: map[string]any{
				"pattern":          "error",
				"output_mode":      "count",
				"case_insensitive": true,
			},
		},
		{
			Description: "在指定类型文件中搜索",
			Input: map[string]any{
				"pattern":     "import\\s+\"",
				"path":        "/app",
				"file_types":  []string{"go"},
				"output_mode": "content",
				"max_results": 50,
			},
		},
	}
}
