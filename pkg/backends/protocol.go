package backends

import (
	"context"
	"time"
)

// FileInfo 文件信息
type FileInfo struct {
	Path         string    `json:"path"`
	IsDirectory  bool      `json:"is_directory"`
	Size         int64     `json:"size"`
	ModifiedTime time.Time `json:"modified_time"`
	CreatedTime  time.Time `json:"created_time"`
}

// GrepMatch Grep 搜索匹配结果
type GrepMatch struct {
	Path       string `json:"path"`
	LineNumber int    `json:"line_number"`
	Line       string `json:"line"`
	Match      string `json:"match"`
}

// WriteResult 写入操作结果
// 设计参考: DeepAgents backends/protocol.py:36-58
type WriteResult struct {
	Error        string         `json:"error,omitempty"`         // 错误信息,空字符串表示成功
	Path         string         `json:"path,omitempty"`          // 写入文件路径,失败时为空
	BytesWritten int64          `json:"bytes_written,omitempty"` // 写入字节数
	FilesUpdate  map[string]any `json:"files_update,omitempty"`  // StateBackend 状态更新,外部存储为 nil
}

// EditResult 编辑操作结果
// 设计参考: DeepAgents backends/protocol.py:61-85
type EditResult struct {
	Error            string         `json:"error,omitempty"`             // 错误信息,空字符串表示成功
	Path             string         `json:"path,omitempty"`              // 编辑文件路径,失败时为空
	ReplacementsMade int            `json:"replacements_made,omitempty"` // 替换次数,失败时为 0
	FilesUpdate      map[string]any `json:"files_update,omitempty"`      // StateBackend 状态更新,外部存储为 nil
}

// BackendProtocol 统一后端存储协议
// 该接口定义了文件系统操作的统一抽象,支持多种后端实现:
// - StateBackend: 内存临时存储 (会话级)
// - StoreBackend: 持久化存储 (跨会话)
// - FilesystemBackend: 真实文件系统
// - CompositeBackend: 路由组合器
type BackendProtocol interface {
	// ListInfo 列出目录内容
	// path: 目录路径,空字符串表示根目录
	// 返回: 文件信息列表
	ListInfo(ctx context.Context, path string) ([]FileInfo, error)

	// Read 读取文件内容
	// path: 文件路径
	// offset: 起始行号 (0-based)
	// limit: 读取行数限制 (0表示无限制)
	// 返回: 文件内容(字符串)
	Read(ctx context.Context, path string, offset, limit int) (string, error)

	// Write 写入文件
	// path: 文件路径
	// content: 文件内容
	// 返回: WriteResult,包含状态更新信息(用于 StateBackend)
	Write(ctx context.Context, path, content string) (*WriteResult, error)

	// Edit 编辑文件(字符串替换)
	// path: 文件路径
	// oldStr: 要替换的旧字符串
	// newStr: 新字符串
	// replaceAll: true=替换所有匹配, false=仅替换第一个
	// 返回: EditResult,包含替换次数
	Edit(ctx context.Context, path, oldStr, newStr string, replaceAll bool) (*EditResult, error)

	// GrepRaw 正则表达式搜索
	// pattern: 正则表达式模式
	// path: 搜索路径(文件或目录)
	// glob: 文件名匹配模式(如 "*.go")
	// 返回: 匹配结果列表
	GrepRaw(ctx context.Context, pattern, path, glob string) ([]GrepMatch, error)

	// GlobInfo Glob 模式匹配
	// pattern: Glob 模式(如 "**/*.go")
	// path: 搜索起始路径
	// 返回: 匹配文件信息列表
	GlobInfo(ctx context.Context, pattern, path string) ([]FileInfo, error)
}

// BackendFactory Backend 工厂函数类型
// 支持延迟初始化和依赖注入
type BackendFactory func(ctx context.Context) (BackendProtocol, error)
