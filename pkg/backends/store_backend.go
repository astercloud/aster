package backends

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/astercloud/aster/pkg/store"
)

// StoreBackend 持久化存储后端
// 使用 Store 接口进行持久化,数据跨会话保留
// 适用于需要长期保存的文件(如知识库、记忆等)
type StoreBackend struct {
	store     store.Store
	agentID   string
	namespace string // 命名空间,用于隔离不同 Agent 的数据
}

// NewStoreBackend 创建 StoreBackend 实例
func NewStoreBackend(s store.Store, agentID string) *StoreBackend {
	return &StoreBackend{
		store:     s,
		agentID:   agentID,
		namespace: "files:" + agentID,
	}
}

// ListInfo 实现 BackendProtocol.ListInfo
func (b *StoreBackend) ListInfo(ctx context.Context, path string) ([]FileInfo, error) {
	if path == "" {
		path = "/"
	}

	// 规范化路径
	path = filepath.Clean(path)
	if !strings.HasSuffix(path, "/") && path != "/" {
		path += "/"
	}

	// 从 Store 加载所有文件元信息
	allFiles, err := b.loadAllFileMeta(ctx)
	if err != nil {
		return nil, err
	}

	var results []FileInfo
	seen := make(map[string]bool)

	for filePath, meta := range allFiles {
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
			results = append(results, FileInfo{
				Path:         filePath,
				IsDirectory:  false,
				Size:         meta.Size,
				CreatedTime:  meta.CreatedAt,
				ModifiedTime: meta.ModifiedAt,
			})
		}
	}

	return results, nil
}

// Read 实现 BackendProtocol.Read
func (b *StoreBackend) Read(ctx context.Context, path string, offset, limit int) (string, error) {
	// 从 Store 加载文件数据
	data, err := b.loadFileData(ctx, path)
	if err != nil {
		return "", err
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
func (b *StoreBackend) Write(ctx context.Context, path, content string) (*WriteResult, error) {
	now := time.Now()
	lines := strings.Split(content, "\n")

	// 检查文件是否已存在
	existingData, err := b.loadFileData(ctx, path)
	createdAt := now
	if err == nil && existingData != nil {
		createdAt = existingData.CreatedAt
	}

	data := &FileData{
		Lines:      lines,
		CreatedAt:  createdAt,
		ModifiedAt: now,
	}

	// 保存到 Store
	if err := b.saveFileData(ctx, path, data); err != nil {
		return nil, err
	}

	return &WriteResult{
		Error:        "", // 空字符串表示成功
		Path:         path,
		BytesWritten: int64(len(content)),
		FilesUpdate:  nil, // StoreBackend 不需要返回状态更新
	}, nil
}

// Edit 实现 BackendProtocol.Edit
func (b *StoreBackend) Edit(ctx context.Context, path, oldStr, newStr string, replaceAll bool) (*EditResult, error) {
	// 加载文件数据
	data, err := b.loadFileData(ctx, path)
	if err != nil {
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
		if err := b.saveFileData(ctx, path, data); err != nil {
			return nil, err
		}
	}

	return &EditResult{
		Error:            "", // 空字符串表示成功
		Path:             path,
		ReplacementsMade: count,
	}, nil
}

// GrepRaw 实现 BackendProtocol.GrepRaw
func (b *StoreBackend) GrepRaw(ctx context.Context, pattern, path, glob string) ([]GrepMatch, error) {
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

	// 加载所有文件
	allFiles, err := b.loadAllFileMeta(ctx)
	if err != nil {
		return nil, err
	}

	var matches []GrepMatch

	for filePath := range allFiles {
		// 检查路径是否匹配
		if path != "" && !strings.HasPrefix(filePath, path) {
			continue
		}

		// 检查 glob 过滤
		if globRe != nil && !globRe.MatchString(filepath.Base(filePath)) {
			continue
		}

		// 加载文件内容
		data, err := b.loadFileData(ctx, filePath)
		if err != nil {
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
func (b *StoreBackend) GlobInfo(ctx context.Context, pattern, path string) ([]FileInfo, error) {
	if path == "" {
		path = "/"
	}

	// 将 glob 模式转换为正则表达式
	regexPattern := globToRegex(pattern)
	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern: %w", err)
	}

	// 加载所有文件元信息
	allFiles, err := b.loadAllFileMeta(ctx)
	if err != nil {
		return nil, err
	}

	var results []FileInfo

	for filePath, meta := range allFiles {
		// 检查路径前缀
		if !strings.HasPrefix(filePath, path) && path != "/" {
			continue
		}

		// 匹配模式
		relPath := strings.TrimPrefix(filePath, path)
		if re.MatchString(relPath) || re.MatchString(filePath) {
			results = append(results, FileInfo{
				Path:         filePath,
				IsDirectory:  false,
				Size:         meta.Size,
				CreatedTime:  meta.CreatedAt,
				ModifiedTime: meta.ModifiedAt,
			})
		}
	}

	return results, nil
}

// FileMeta 文件元信息
type FileMeta struct {
	Size       int64     `json:"size"`
	CreatedAt  time.Time `json:"created_at"`
	ModifiedAt time.Time `json:"modified_at"`
}

// loadFileData 从 Store 加载文件数据
func (b *StoreBackend) loadFileData(ctx context.Context, path string) (*FileData, error) {
	key := fmt.Sprintf("%s:data:%s", b.namespace, path)

	// 使用 Store 的 Todos 接口作为通用 KV 存储
	// (这是一个临时方案,理想情况下应该有专门的 KV 存储接口)
	dataInterface, err := b.store.LoadTodos(ctx, key)
	if err != nil {
		return nil, err
	}

	if dataInterface == nil {
		return nil, fmt.Errorf("file not found: %s", path)
	}

	// 反序列化
	var data FileData
	jsonData, err := json.Marshal(dataInterface)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

// saveFileData 保存文件数据到 Store
func (b *StoreBackend) saveFileData(ctx context.Context, path string, data *FileData) error {
	dataKey := fmt.Sprintf("%s:data:%s", b.namespace, path)

	// 保存文件数据
	if err := b.store.SaveTodos(ctx, dataKey, data); err != nil {
		return err
	}

	// 更新文件元信息索引
	metaKey := b.namespace + ":meta"
	allMeta, _ := b.loadAllFileMeta(ctx)
	if allMeta == nil {
		allMeta = make(map[string]FileMeta)
	}

	size := int64(0)
	for _, line := range data.Lines {
		size += int64(len(line) + 1)
	}

	allMeta[path] = FileMeta{
		Size:       size,
		CreatedAt:  data.CreatedAt,
		ModifiedAt: data.ModifiedAt,
	}

	return b.store.SaveTodos(ctx, metaKey, allMeta)
}

// loadAllFileMeta 加载所有文件元信息
func (b *StoreBackend) loadAllFileMeta(ctx context.Context) (map[string]FileMeta, error) {
	metaKey := b.namespace + ":meta"

	dataInterface, err := b.store.LoadTodos(ctx, metaKey)
	if err != nil {
		return make(map[string]FileMeta), nil
	}

	if dataInterface == nil {
		return make(map[string]FileMeta), nil
	}

	// 反序列化
	var allMeta map[string]FileMeta
	jsonData, err := json.Marshal(dataInterface)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(jsonData, &allMeta); err != nil {
		return nil, err
	}

	return allMeta, nil
}
