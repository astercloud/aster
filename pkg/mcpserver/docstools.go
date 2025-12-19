package mcpserver

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/tools"
)

var mcpLog = logging.ForComponent("MCPServer")

// DocsToolConfig 配置 docs_get/docs_search 工具
type DocsToolConfig struct {
	BaseDir string
}

// normalizeBaseDir 确保 baseDir 是绝对路径并移除尾部斜杠
func normalizeBaseDir(baseDir string) (string, error) {
	if baseDir == "" {
		return "", errors.New("baseDir is required")
	}
	abs, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}
	return abs, nil
}

// =========================
// docs_get 工具
// =========================

type DocsGetTool struct {
	baseDir string
}

func NewDocsGetTool(baseDir string) (tools.Tool, error) {
	abs, err := normalizeBaseDir(baseDir)
	if err != nil {
		return nil, err
	}
	return &DocsGetTool{baseDir: abs}, nil
}

func (t *DocsGetTool) Name() string { return "docs_get" }

func (t *DocsGetTool) Description() string {
	return "Read a documentation file from the configured base directory."
}

func (t *DocsGetTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Relative path to the doc file, e.g. \"README.md\" or \"docs/content/index.md\".",
			},
		},
		"required": []string{"path"},
	}
}

func (t *DocsGetTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	relPath, _ := input["path"].(string)
	if relPath == "" {
		return nil, errors.New("path is required")
	}

	// 防止路径遍历
	if strings.Contains(relPath, "..") {
		return nil, errors.New("path traversal is not allowed")
	}

	fullPath := filepath.Join(t.baseDir, filepath.FromSlash(relPath))
	abs, err := filepath.Abs(fullPath)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	if !strings.HasPrefix(abs, t.baseDir) {
		return nil, errors.New("path outside baseDir is not allowed")
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	return map[string]any{
		"path":    relPath,
		"content": string(data),
	}, nil
}

func (t *DocsGetTool) Prompt() string {
	return "Use this tool to fetch documentation files (Markdown or text) from the configured docs directory."
}

// =========================
// docs_search 工具
// =========================

type DocsSearchTool struct {
	baseDir string
}

func NewDocsSearchTool(baseDir string) (tools.Tool, error) {
	abs, err := normalizeBaseDir(baseDir)
	if err != nil {
		return nil, err
	}
	return &DocsSearchTool{baseDir: abs}, nil
}

func (t *DocsSearchTool) Name() string { return "docs_search" }

func (t *DocsSearchTool) Description() string {
	return "Search documentation files for a query string (case-insensitive)."
}

func (t *DocsSearchTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Query string to search for (case-insensitive).",
			},
			"subdir": map[string]any{
				"type":        "string",
				"description": "Optional subdirectory under the base docs dir to limit the search.",
			},
			"glob": map[string]any{
				"type":        "string",
				"description": "Optional glob pattern, e.g. \"*.md\".",
			},
			"max_results": map[string]any{
				"type":        "integer",
				"description": "Maximum number of matches to return (default 50).",
			},
		},
		"required": []string{"query"},
	}
}

func (t *DocsSearchTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	query, _ := input["query"].(string)
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("query cannot be empty")
	}
	queryLower := strings.ToLower(query)

	subdir, _ := input["subdir"].(string)
	globPattern, _ := input["glob"].(string)

	maxResults := 50
	if v, ok := input["max_results"].(float64); ok && int(v) > 0 {
		maxResults = int(v)
	}

	root := t.baseDir
	if subdir != "" {
		if strings.Contains(subdir, "..") {
			return nil, errors.New("subdir path traversal is not allowed")
		}
		root = filepath.Join(t.baseDir, filepath.FromSlash(subdir))
	}

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}
	if !strings.HasPrefix(rootAbs, t.baseDir) {
		return nil, errors.New("subdir outside baseDir is not allowed")
	}

	type Match struct {
		Path       string `json:"path"`
		LineNumber int    `json:"line_number"`
		Line       string `json:"line"`
	}

	matches := make([]Match, 0, maxResults)

	walkErr := filepath.Walk(rootAbs, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip error
		}
		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(t.baseDir, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)

		if globPattern != "" {
			ok, _ := filepath.Match(globPattern, filepath.Base(rel))
			if !ok {
				return nil
			}
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer func() { _ = f.Close() }()

		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if strings.Contains(strings.ToLower(line), queryLower) {
				matches = append(matches, Match{
					Path:       rel,
					LineNumber: lineNum,
					Line:       line,
				})
				if len(matches) >= maxResults {
					return errors.New("max_results_reached")
				}
			}
		}
		return nil
	})

	// max_results_reached 是预期的终止条件，其他错误需要记录
	if walkErr != nil && walkErr.Error() != "max_results_reached" {
		mcpLog.Warn(ctx, "walk error in docs search", map[string]any{"error": walkErr})
	}

	return map[string]any{
		"query":       query,
		"base_dir":    t.baseDir,
		"subdir":      subdir,
		"glob":        globPattern,
		"max_results": maxResults,
		"count":       len(matches),
		"matches":     matches,
	}, nil
}

func (t *DocsSearchTool) Prompt() string {
	return `Use this tool to search documentation files (Markdown or text) under the configured docs directory.

Guidelines:
- Use simple, case-insensitive keywords.
- Limit the search using "subdir" and/or "glob" when possible to avoid scanning the entire tree.
- Inspect the returned matches (path + line number + line) and then use docs_get to read the full file if needed.`
}
