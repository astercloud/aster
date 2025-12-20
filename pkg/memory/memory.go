package memory

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/astercloud/aster/pkg/backends"
)

// ManagerConfig 配置高级 Memory 管理器
type ManagerConfig struct {
	Backend    backends.BackendProtocol
	MemoryPath string // 记忆文件根路径,默认: "/memories/"
}

// Manager 提供基于 BackendProtocol 的高级记忆能力
// 特点:
// - 所有记忆都以普通文本/Markdown 文件形式存储
// - 通过 GrepRaw/GlobInfo 做全文搜索,不依赖向量数据库
// - 约定好 memoryPath,统一管理长期记忆文件
type Manager struct {
	backend    backends.BackendProtocol
	memoryPath string
}

// SearchOptions 搜索配置
type SearchOptions struct {
	// Query 搜索关键字,默认作为大小写不敏感的字面量匹配
	Query string
	// Regex 是否将 Query 视为正则表达式
	Regex bool
	// Namespace 可选的命名空间前缀,用于多租户/多资源隔离
	// 例如: "users/alice", "projects/demo", "users/alice/projects/demo"
	// 为空字符串时在整个 MemoryPath 下搜索
	Namespace string
	// Glob 文件过滤模式,例如: "*.md"
	Glob string
	// MaxResults 返回的最大匹配数,<=0 时表示不限制
	MaxResults int
}

// SearchMatch 搜索匹配结果
type SearchMatch struct {
	Path       string `json:"path"`
	LineNumber int    `json:"line_number"`
	Line       string `json:"line"`
	Match      string `json:"match"`
}

// NewManager 创建 Memory 管理器
func NewManager(cfg *ManagerConfig) (*Manager, error) {
	if cfg == nil {
		return nil, errors.New("memory.ManagerConfig cannot be nil")
	}
	if cfg.Backend == nil {
		return nil, errors.New("memory.Manager requires a non-nil Backend")
	}

	memoryPath := cfg.MemoryPath
	if memoryPath == "" {
		memoryPath = "/memories/"
	}

	// 规范化路径,统一为以 / 开头,以 / 结尾
	memoryPath = normalizeDir(memoryPath)

	return &Manager{
		backend:    cfg.Backend,
		memoryPath: memoryPath,
	}, nil
}

// MemoryPath 返回记忆根路径
func (m *Manager) MemoryPath() string {
	return m.memoryPath
}

// ListFiles 列出所有记忆文件
func (m *Manager) ListFiles(ctx context.Context) ([]backends.FileInfo, error) {
	return m.backend.ListInfo(ctx, m.memoryPath)
}

// ReadFile 读取指定记忆文件内容
// name 为相对于 memoryPath 的路径,例如 "project_notes.md" 或 "user/alice.md"
func (m *Manager) ReadFile(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", errors.New("memory.ReadFile: name cannot be empty")
	}
	path := m.resolvePath(name)
	return m.backend.Read(ctx, path, 0, 0)
}

// AppendNote 以追加模式写入一条记忆
// file: 目标文件名(相对于 memoryPath)
// title: 记忆标题,为空时使用当前时间
// content: 记忆内容正文
func (m *Manager) AppendNote(ctx context.Context, file, title, content string) (string, error) {
	if file == "" {
		return "", errors.New("memory.AppendNote: file cannot be empty")
	}
	if strings.TrimSpace(content) == "" {
		return "", errors.New("memory.AppendNote: content cannot be empty")
	}

	path := m.resolvePath(file)

	// 尝试读取现有内容,不存在时视为空
	existing, err := m.backend.Read(ctx, path, 0, 0)
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "not found") {
		// 兼容不同 Backend 的错误信息,只在确定不是 "not found" 时返回错误
		return "", fmt.Errorf("memory.AppendNote: read existing content failed: %w", err)
	}

	noteTitle := strings.TrimSpace(title)
	if noteTitle == "" {
		noteTitle = time.Now().Format("2006-01-02 15:04:05")
	}

	section := fmt.Sprintf("## %s\n\n%s\n", noteTitle, strings.TrimSpace(content))

	var newContent string
	if strings.TrimSpace(existing) == "" {
		newContent = section
	} else {
		if !strings.HasSuffix(existing, "\n") {
			existing += "\n"
		}
		newContent = existing + "\n" + section
	}

	if _, err := m.backend.Write(ctx, path, newContent); err != nil {
		return "", fmt.Errorf("memory.AppendNote: write content failed: %w", err)
	}

	return path, nil
}

// OverwriteWithNote 使用单个 Note 覆盖整个记忆文件
// 与 AppendNote 不同,该方法会丢弃原有内容,仅保留新的标题与正文
func (m *Manager) OverwriteWithNote(ctx context.Context, file, title, content string) (string, error) {
	if file == "" {
		return "", errors.New("memory.OverwriteWithNote: file cannot be empty")
	}
	if strings.TrimSpace(content) == "" {
		return "", errors.New("memory.OverwriteWithNote: content cannot be empty")
	}

	path := m.resolvePath(file)

	noteTitle := strings.TrimSpace(title)
	if noteTitle == "" {
		noteTitle = time.Now().Format("2006-01-02 15:04:05")
	}

	section := fmt.Sprintf("## %s\n\n%s\n", noteTitle, strings.TrimSpace(content))

	if _, err := m.backend.Write(ctx, path, section); err != nil {
		return "", fmt.Errorf("memory.OverwriteWithNote: write content failed: %w", err)
	}

	return path, nil
}

// Search 在 memoryPath 下执行全文搜索
// 默认使用大小写不敏感的字面量匹配,可选正则模式
func (m *Manager) Search(ctx context.Context, opts *SearchOptions) ([]SearchMatch, error) {
	if opts == nil {
		return nil, errors.New("memory.Search: options cannot be nil")
	}
	rawQuery := strings.TrimSpace(opts.Query)
	if rawQuery == "" {
		return nil, errors.New("memory.Search: query cannot be empty")
	}

	var pattern string
	if opts.Regex {
		pattern = rawQuery
	} else {
		// 使用大小写不敏感的字面量匹配
		pattern = "(?i)" + regexp.QuoteMeta(rawQuery)
	}

	// 根据命名空间选择搜索根路径
	searchPath := m.memoryPath
	if ns := strings.TrimSpace(opts.Namespace); ns != "" {
		// Namespace 也通过 resolvePath 规范化,确保不会逃出 memoryPath
		searchPath = m.resolvePath(ns)
	}

	matches, err := m.backend.GrepRaw(ctx, pattern, searchPath, opts.Glob)
	if err != nil {
		return nil, fmt.Errorf("memory.Search: grep failed: %w", err)
	}

	maxResults := opts.MaxResults
	if maxResults > 0 && len(matches) > maxResults {
		matches = matches[:maxResults]
	}

	results := make([]SearchMatch, 0, len(matches))
	for _, m := range matches {
		results = append(results, SearchMatch{
			Path:       m.Path,
			LineNumber: m.LineNumber,
			Line:       m.Line,
			Match:      m.Match,
		})
	}

	return results, nil
}

// normalizeDir 规范化目录路径为 "/xxx/" 形式
func normalizeDir(path string) string {
	if path == "" {
		return "/"
	}

	// 使用 filepath.Clean 处理冗余分隔符,再统一为 Unix 风格
	cleaned := filepath.ToSlash(filepath.Clean(path))

	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	if !strings.HasSuffix(cleaned, "/") {
		cleaned += "/"
	}
	return cleaned
}

// resolvePath 将相对名称解析为 memoryPath 下的完整路径
func (m *Manager) resolvePath(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimLeft(name, "/")
	joined := filepath.ToSlash(filepath.Join(m.memoryPath, name))

	// 确保不会逃出 memoryPath 根目录
	if !strings.HasPrefix(joined, m.memoryPath) {
		return m.memoryPath + name
	}
	return joined
}
