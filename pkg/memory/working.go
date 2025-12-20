package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/astercloud/aster/pkg/backends"
)

// WorkingMemoryScope 定义 Working Memory 的作用域
type WorkingMemoryScope string

const (
	// ScopeThread Working Memory 以 thread 为作用域（每个会话独立）
	ScopeThread WorkingMemoryScope = "thread"
	// ScopeResource Working Memory 以 resource 为作用域（同一资源下的所有会话共享）
	ScopeResource WorkingMemoryScope = "resource"
)

// WorkingMemoryConfig Working Memory 管理器配置
type WorkingMemoryConfig struct {
	Backend    backends.BackendProtocol // 存储后端
	BasePath   string                   // 存储根路径，默认 "/working_memory/"
	Scope      WorkingMemoryScope       // 作用域：thread 或 resource
	Schema     *JSONSchema              // 可选的 JSON Schema 验证
	Template   string                   // 可选的 Markdown 模板
	DefaultTTL time.Duration            // 可选的过期时间（0 表示不过期）
}

// WorkingMemoryManager Working Memory 管理器
// 特点：
// - 支持 thread/resource 作用域自动管理
// - 可选的 JSON Schema 验证
// - 可选的 Markdown 模板渲染
// - 基于 BackendProtocol，保持松耦合
type WorkingMemoryManager struct {
	backend  backends.BackendProtocol
	basePath string
	scope    WorkingMemoryScope
	schema   *JSONSchema
	template string
	ttl      time.Duration
}

// WorkingMemoryMeta Working Memory 元数据
type WorkingMemoryMeta struct {
	ThreadID   string     `json:"thread_id"`
	ResourceID string     `json:"resource_id"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

// WorkingMemoryData Working Memory 数据结构
type WorkingMemoryData struct {
	Meta    WorkingMemoryMeta `json:"meta"`
	Content string            `json:"content"`
}

// NewWorkingMemoryManager 创建 Working Memory 管理器
func NewWorkingMemoryManager(cfg *WorkingMemoryConfig) (*WorkingMemoryManager, error) {
	if cfg == nil {
		return nil, errors.New("working memory config cannot be nil")
	}
	if cfg.Backend == nil {
		return nil, errors.New("working memory requires a non-nil Backend")
	}

	basePath := cfg.BasePath
	if basePath == "" {
		basePath = "/working_memory/"
	}
	basePath = normalizeDir(basePath)

	scope := cfg.Scope
	if scope == "" {
		scope = ScopeThread // 默认 thread 作用域
	}
	if scope != ScopeThread && scope != ScopeResource {
		return nil, fmt.Errorf("invalid scope: %s (must be 'thread' or 'resource')", scope)
	}

	// 验证 Schema（如果提供）
	if cfg.Schema != nil {
		if err := cfg.Schema.Validate(); err != nil {
			return nil, fmt.Errorf("invalid JSON schema: %w", err)
		}
	}

	return &WorkingMemoryManager{
		backend:  cfg.Backend,
		basePath: basePath,
		scope:    scope,
		schema:   cfg.Schema,
		template: cfg.Template,
		ttl:      cfg.DefaultTTL,
	}, nil
}

// Get 获取 Working Memory 内容
// 根据配置的 scope 自动选择读取路径：
// - thread scope: /working_memory/threads/<threadID>.json
// - resource scope: /working_memory/resources/<resourceID>.json
func (wm *WorkingMemoryManager) Get(ctx context.Context, threadID, resourceID string) (string, error) {
	if threadID == "" && resourceID == "" {
		return "", errors.New("threadID and resourceID cannot both be empty")
	}

	path := wm.resolvePath(threadID, resourceID)

	content, err := wm.backend.Read(ctx, path, 0, 0)
	if err != nil {
		// 文件不存在时返回空字符串，不报错
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return "", nil
		}
		return "", fmt.Errorf("read working memory: %w", err)
	}

	// 解析 JSON
	var data WorkingMemoryData
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return "", fmt.Errorf("parse working memory: %w", err)
	}

	// 检查是否过期
	if data.Meta.ExpiresAt != nil && time.Now().After(*data.Meta.ExpiresAt) {
		return "", nil // 已过期，返回空
	}

	return data.Content, nil
}

// Update 更新 Working Memory 内容
// content: 新的内容（Markdown 或 JSON 字符串）
// 如果配置了 Schema，会先进行验证
func (wm *WorkingMemoryManager) Update(ctx context.Context, threadID, resourceID, content string) error {
	if threadID == "" && resourceID == "" {
		return errors.New("threadID and resourceID cannot both be empty")
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return errors.New("content cannot be empty")
	}

	// Schema 验证（如果配置）
	if wm.schema != nil {
		if err := wm.schema.ValidateContent(content); err != nil {
			return fmt.Errorf("schema validation failed: %w", err)
		}
	}

	// 构建元数据
	now := time.Now()
	meta := WorkingMemoryMeta{
		ThreadID:   threadID,
		ResourceID: resourceID,
		UpdatedAt:  now,
	}

	// 尝试读取现有数据以保留 CreatedAt
	path := wm.resolvePath(threadID, resourceID)
	existingContent, err := wm.backend.Read(ctx, path, 0, 0)
	if err == nil {
		var existingData WorkingMemoryData
		if json.Unmarshal([]byte(existingContent), &existingData) == nil {
			meta.CreatedAt = existingData.Meta.CreatedAt
		}
	}

	// 如果是新创建，设置 CreatedAt
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = now
	}

	// 设置过期时间（如果配置）
	if wm.ttl > 0 {
		expiresAt := now.Add(wm.ttl)
		meta.ExpiresAt = &expiresAt
	}

	// 构建完整数据
	data := WorkingMemoryData{
		Meta:    meta,
		Content: content,
	}

	// 序列化为 JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal working memory: %w", err)
	}

	// 写入后端
	if _, err := wm.backend.Write(ctx, path, string(jsonData)); err != nil {
		return fmt.Errorf("write working memory: %w", err)
	}

	return nil
}

// FindAndReplace 在现有内容中查找并替换（实验性功能，对标 Mastra）
// searchString: 要查找的字符串
// newContent: 替换后的新内容
// 如果 searchString 为空，则追加到末尾
func (wm *WorkingMemoryManager) FindAndReplace(ctx context.Context, threadID, resourceID, searchString, newContent string) error {
	if threadID == "" && resourceID == "" {
		return errors.New("threadID and resourceID cannot both be empty")
	}

	// 读取现有内容
	existing, err := wm.Get(ctx, threadID, resourceID)
	if err != nil {
		return fmt.Errorf("read existing content: %w", err)
	}

	var updated string
	if searchString == "" || existing == "" {
		// 追加模式
		if existing == "" {
			updated = newContent
		} else {
			updated = existing + "\n\n" + newContent
		}
	} else {
		// 查找替换模式
		if !strings.Contains(existing, searchString) {
			return errors.New("search string not found in working memory")
		}
		updated = strings.Replace(existing, searchString, newContent, 1)
	}

	return wm.Update(ctx, threadID, resourceID, updated)
}

// Delete 删除 Working Memory
func (wm *WorkingMemoryManager) Delete(ctx context.Context, threadID, resourceID string) error {
	if threadID == "" && resourceID == "" {
		return errors.New("threadID and resourceID cannot both be empty")
	}

	path := wm.resolvePath(threadID, resourceID)

	// Backend 通常没有 Delete 方法，通过写入空内容或特殊标记实现
	// 这里我们写入一个标记为已删除的 JSON
	deletedData := WorkingMemoryData{
		Meta: WorkingMemoryMeta{
			ThreadID:   threadID,
			ResourceID: resourceID,
			UpdatedAt:  time.Now(),
		},
		Content: "",
	}

	jsonData, _ := json.MarshalIndent(deletedData, "", "  ")
	if _, err := wm.backend.Write(ctx, path, string(jsonData)); err != nil {
		return fmt.Errorf("delete working memory: %w", err)
	}

	return nil
}

// GetScope 返回当前配置的作用域
func (wm *WorkingMemoryManager) GetScope() WorkingMemoryScope {
	return wm.scope
}

// GetSchema 返回配置的 JSON Schema
func (wm *WorkingMemoryManager) GetSchema() *JSONSchema {
	return wm.schema
}

// GetTemplate 返回配置的模板
func (wm *WorkingMemoryManager) GetTemplate() string {
	return wm.template
}

// resolvePath 根据 scope 解析存储路径
func (wm *WorkingMemoryManager) resolvePath(threadID, resourceID string) string {
	var filename string

	switch wm.scope {
	case ScopeThread:
		if threadID == "" {
			// 如果没有 threadID，使用 resourceID 作为后备
			threadID = resourceID
		}
		filename = filepath.ToSlash(filepath.Join(wm.basePath, "threads", threadID+".json"))

	case ScopeResource:
		if resourceID == "" {
			// 如果没有 resourceID，使用 threadID 作为后备
			resourceID = threadID
		}
		filename = filepath.ToSlash(filepath.Join(wm.basePath, "resources", resourceID+".json"))
	}

	return filename
}
