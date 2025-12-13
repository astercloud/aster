package sandbox

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
)

// LocalFS 本地文件系统实现
type LocalFS struct {
	workDir         string
	enforceBoundary bool
	allowPaths      []string
}

// Resolve 解析路径为绝对路径
// 注意：所有路径都会被解析为相对于 workDir 的路径，防止写入到工作目录外
func (lfs *LocalFS) Resolve(path string) string {
	// 如果是绝对路径，将其转换为相对于 workDir 的路径
	// 这防止 Agent 使用 /tmp 等系统目录
	if filepath.IsAbs(path) {
		// 移除开头的 / 使其成为相对路径
		path = strings.TrimPrefix(path, "/")
	}
	return filepath.Join(lfs.workDir, path)
}

// IsInside 检查路径是否在沙箱内
// 如果传入的是绝对路径，直接检查该路径是否在 workDir 或白名单内
// 如果传入的是相对路径，先解析为绝对路径再检查
func (lfs *LocalFS) IsInside(path string) bool {
	var resolved string
	var err error

	// 如果是绝对路径，直接使用（不经过 Resolve 转换）
	if filepath.IsAbs(path) {
		resolved, err = filepath.Abs(path)
	} else {
		resolved, err = filepath.Abs(lfs.Resolve(path))
	}
	if err != nil {
		return false
	}

	// 1. 检查是否在workDir内
	workDirAbs, _ := filepath.Abs(lfs.workDir)
	relativeToWork, err := filepath.Rel(workDirAbs, resolved)
	if err == nil && !strings.HasPrefix(relativeToWork, "..") && !filepath.IsAbs(relativeToWork) {
		return true
	}

	// 2. 如果不强制边界检查,允许所有路径
	if !lfs.enforceBoundary {
		return true
	}

	// 3. 检查白名单
	for _, allowed := range lfs.allowPaths {
		resolvedAllowed, err := filepath.Abs(allowed)
		if err != nil {
			continue
		}
		relative, err := filepath.Rel(resolvedAllowed, resolved)
		if err == nil && !strings.HasPrefix(relative, "..") && !filepath.IsAbs(relative) {
			return true
		}
	}

	return false
}

// Read 读取文件内容
func (lfs *LocalFS) Read(ctx context.Context, path string) (string, error) {
	resolved := lfs.Resolve(path)
	if !lfs.IsInside(resolved) {
		return "", fmt.Errorf("path outside sandbox: %s", path)
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	return string(data), nil
}

// Write 写入文件内容
func (lfs *LocalFS) Write(ctx context.Context, path string, content string) error {
	resolved := lfs.Resolve(path)
	if !lfs.IsInside(resolved) {
		return fmt.Errorf("path outside sandbox: %s", path)
	}

	// 确保目录存在
	dir := filepath.Dir(resolved)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(resolved, []byte(content), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// Temp 生成临时文件路径
func (lfs *LocalFS) Temp(name string) string {
	if name == "" {
		name = fmt.Sprintf("temp-%d-%s", time.Now().UnixNano(), randomString(8))
	}
	tempPath := filepath.Join(lfs.workDir, ".temp", name)
	relative, _ := filepath.Rel(lfs.workDir, tempPath)
	return relative
}

// Stat 获取文件状态
func (lfs *LocalFS) Stat(ctx context.Context, path string) (FileInfo, error) {
	resolved := lfs.Resolve(path)
	if !lfs.IsInside(resolved) {
		return FileInfo{}, fmt.Errorf("path outside sandbox: %s", path)
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return FileInfo{}, fmt.Errorf("stat file: %w", err)
	}

	return FileInfo{
		Path:    path,
		Size:    info.Size(),
		ModTime: info.ModTime(),
		IsDir:   info.IsDir(),
		Mode:    int(info.Mode()),
	}, nil
}

// Glob 文件匹配
func (lfs *LocalFS) Glob(ctx context.Context, pattern string, opts *GlobOptions) ([]string, error) {
	if opts == nil {
		opts = &GlobOptions{}
	}

	// 确定搜索根目录
	cwd := lfs.workDir
	if opts.CWD != "" {
		cwd = lfs.Resolve(opts.CWD)
	}

	// 使用doublestar进行glob匹配
	fsys := os.DirFS(cwd)
	matches, err := doublestar.Glob(fsys, pattern,
		doublestar.WithFilesOnly(),
		doublestar.WithNoFollow(),
	)
	if err != nil {
		return nil, fmt.Errorf("glob pattern: %w", err)
	}

	// 过滤结果
	results := make([]string, 0, len(matches))
	for _, match := range matches {
		fullPath := filepath.Join(cwd, match)

		// 检查是否在沙箱内
		if !lfs.IsInside(fullPath) {
			continue
		}

		// 检查ignore规则
		if opts.Ignore != nil && lfs.shouldIgnore(match, opts.Ignore) {
			continue
		}

		// 返回绝对路径或相对路径
		if opts.Absolute {
			results = append(results, fullPath)
		} else {
			rel, err := filepath.Rel(lfs.workDir, fullPath)
			if err != nil {
				results = append(results, match)
			} else {
				results = append(results, rel)
			}
		}
	}

	return results, nil
}

// shouldIgnore 检查是否应该忽略文件
func (lfs *LocalFS) shouldIgnore(path string, ignorePatterns []string) bool {
	for _, pattern := range ignorePatterns {
		matched, err := doublestar.Match(pattern, path)
		if err == nil && matched {
			return true
		}
	}
	return false
}
