package backends

import (
	"context"
	"fmt"
	"maps"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// FileData 文件数据结构
type FileData struct {
	Lines      []string  `json:"lines"`
	CreatedAt  time.Time `json:"created_at"`
	ModifiedAt time.Time `json:"modified_at"`
}

// StateBackend 内存临时存储后端
// 数据存储在内存 map 中,生命周期与 Agent 会话一致
// 适用于临时文件、中间结果等短期存储需求
type StateBackend struct {
	mu    sync.RWMutex
	files map[string]*FileData
}

// NewStateBackend 创建 StateBackend 实例
func NewStateBackend() *StateBackend {
	return &StateBackend{
		files: make(map[string]*FileData),
	}
}

// ListInfo 实现 BackendProtocol.ListInfo
func (b *StateBackend) ListInfo(ctx context.Context, path string) ([]FileInfo, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if path == "" {
		path = "/"
	}

	// 规范化路径
	path = filepath.Clean(path)
	if !strings.HasSuffix(path, "/") && path != "/" {
		path += "/"
	}

	var results []FileInfo
	seen := make(map[string]bool)

	for filePath, data := range b.files {
		// 检查是否在目标目录下
		if !strings.HasPrefix(filePath, path) && path != "/" {
			continue
		}

		// 获取相对路径
		relPath := filePath
		if path != "/" {
			relPath = strings.TrimPrefix(filePath, path)
		}

		// 检查是否是直接子项
		parts := strings.Split(strings.Trim(relPath, "/"), "/")
		if len(parts) == 0 {
			continue
		}

		name := parts[0]
		if name == "" {
			continue
		}

		fullPath := filepath.Join(path, name)
		if seen[fullPath] {
			continue
		}
		seen[fullPath] = true

		// 判断是文件还是目录
		isDir := len(parts) > 1

		if isDir {
			// 目录项
			results = append(results, FileInfo{
				Path:        fullPath,
				IsDirectory: true,
				Size:        0,
			})
		} else {
			// 文件项
			size := int64(0)
			for _, line := range data.Lines {
				size += int64(len(line) + 1) // +1 for newline
			}

			results = append(results, FileInfo{
				Path:         filePath,
				IsDirectory:  false,
				Size:         size,
				CreatedTime:  data.CreatedAt,
				ModifiedTime: data.ModifiedAt,
			})
		}
	}

	return results, nil
}

// Read 实现 BackendProtocol.Read
func (b *StateBackend) Read(ctx context.Context, path string, offset, limit int) (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	data, exists := b.files[path]
	if !exists {
		return "", fmt.Errorf("file not found: %s", path)
	}

	totalLines := len(data.Lines)

	// 处理 offset
	if offset < 0 {
		offset = 0
	}
	if offset >= totalLines {
		return "", nil
	}

	// 处理 limit
	endLine := totalLines
	if limit > 0 {
		endLine = min(offset+limit, totalLines)
	}

	selectedLines := data.Lines[offset:endLine]
	return strings.Join(selectedLines, "\n"), nil
}

// Write 实现 BackendProtocol.Write
func (b *StateBackend) Write(ctx context.Context, path, content string) (*WriteResult, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	lines := strings.Split(content, "\n")

	// 检查文件是否已存在
	_, exists := b.files[path]

	data := &FileData{
		Lines:      lines,
		ModifiedAt: now,
		CreatedAt:  now,
	}

	if exists {
		// 保留创建时间
		data.CreatedAt = b.files[path].CreatedAt
	}

	b.files[path] = data

	return &WriteResult{
		Error:        "", // 空字符串表示成功
		Path:         path,
		BytesWritten: int64(len(content)),
		FilesUpdate: map[string]any{
			path: data,
		},
	}, nil
}

// Edit 实现 BackendProtocol.Edit
func (b *StateBackend) Edit(ctx context.Context, path, oldStr, newStr string, replaceAll bool) (*EditResult, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	data, exists := b.files[path]
	if !exists {
		return &EditResult{
			Error: "file not found: " + path,
			Path:  path,
		}, nil
	}

	// 读取当前内容
	content := strings.Join(data.Lines, "\n")

	// 执行替换
	var newContent string
	var count int

	if replaceAll {
		count = strings.Count(content, oldStr)
		newContent = strings.ReplaceAll(content, oldStr, newStr)
	} else {
		if strings.Contains(content, oldStr) {
			newContent = strings.Replace(content, oldStr, newStr, 1)
			count = 1
		} else {
			newContent = content
			count = 0
		}
	}

	// 更新文件数据
	if count > 0 {
		data.Lines = strings.Split(newContent, "\n")
		data.ModifiedAt = time.Now()
	}

	return &EditResult{
		Error:            "", // 空字符串表示成功
		Path:             path,
		ReplacementsMade: count,
	}, nil
}

// GrepRaw 实现 BackendProtocol.GrepRaw
func (b *StateBackend) GrepRaw(ctx context.Context, pattern, path, glob string) ([]GrepMatch, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// 编译正则表达式
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	// 编译 glob 模式
	var globRe *regexp.Regexp
	if glob != "" {
		globPattern := globToRegex(glob)
		globRe, err = regexp.Compile(globPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern: %w", err)
		}
	}

	var matches []GrepMatch

	for filePath, data := range b.files {
		// 检查路径是否匹配
		if path != "" && !strings.HasPrefix(filePath, path) {
			continue
		}

		// 检查 glob 过滤
		if globRe != nil && !globRe.MatchString(filepath.Base(filePath)) {
			continue
		}

		// 搜索每一行
		for lineNum, line := range data.Lines {
			if re.MatchString(line) {
				matches = append(matches, GrepMatch{
					Path:       filePath,
					LineNumber: lineNum + 1, // 1-based
					Line:       line,
					Match:      re.FindString(line),
				})
			}
		}
	}

	return matches, nil
}

// GlobInfo 实现 BackendProtocol.GlobInfo
func (b *StateBackend) GlobInfo(ctx context.Context, pattern, path string) ([]FileInfo, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if path == "" {
		path = "/"
	}

	// 将 glob 模式转换为正则表达式
	regexPattern := globToRegex(pattern)
	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern: %w", err)
	}

	var results []FileInfo

	for filePath, data := range b.files {
		// 检查路径前缀
		if !strings.HasPrefix(filePath, path) && path != "/" {
			continue
		}

		// 匹配模式
		relPath := strings.TrimPrefix(filePath, path)
		if re.MatchString(relPath) || re.MatchString(filePath) {
			size := int64(0)
			for _, line := range data.Lines {
				size += int64(len(line) + 1)
			}

			results = append(results, FileInfo{
				Path:         filePath,
				IsDirectory:  false,
				Size:         size,
				CreatedTime:  data.CreatedAt,
				ModifiedTime: data.ModifiedAt,
			})
		}
	}

	return results, nil
}

// globToRegex 将 glob 模式转换为正则表达式
func globToRegex(glob string) string {
	// 简单实现,支持基本的 glob 语法
	pattern := regexp.QuoteMeta(glob)
	// 先处理 ** (匹配任意路径)
	pattern = strings.ReplaceAll(pattern, "\\*\\*", ".*")
	// 再处理 * (匹配单层路径)
	pattern = strings.ReplaceAll(pattern, "\\*", "[^/]*")
	// 处理 ? (匹配单个字符)
	pattern = strings.ReplaceAll(pattern, "\\?", ".")
	// 注意:不要求必须从头匹配,允许部分匹配
	return pattern
}

// GetFiles 获取所有文件数据 (用于调试)
func (b *StateBackend) GetFiles() map[string]*FileData {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make(map[string]*FileData, len(b.files))
	maps.Copy(result, b.files)
	return result
}

// LoadFiles 加载文件数据 (用于状态恢复)
func (b *StateBackend) LoadFiles(files map[string]*FileData) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.files = make(map[string]*FileData, len(files))
	maps.Copy(b.files, files)
}
