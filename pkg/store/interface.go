package store

import (
	"context"

	"github.com/astercloud/aster/pkg/types"
)

// Store 持久化存储接口
type Store interface {
	// SaveMessages 保存消息列表
	SaveMessages(ctx context.Context, agentID string, messages []types.Message) error

	// LoadMessages 加载消息列表
	LoadMessages(ctx context.Context, agentID string) ([]types.Message, error)

	// TrimMessages 修剪消息列表，保留最近的 N 条消息
	// 如果 maxMessages <= 0，则不修剪
	TrimMessages(ctx context.Context, agentID string, maxMessages int) error

	// SaveToolCallRecords 保存工具调用记录
	SaveToolCallRecords(ctx context.Context, agentID string, records []types.ToolCallRecord) error

	// LoadToolCallRecords 加载工具调用记录
	LoadToolCallRecords(ctx context.Context, agentID string) ([]types.ToolCallRecord, error)

	// SaveSnapshot 保存快照
	SaveSnapshot(ctx context.Context, agentID string, snapshot types.Snapshot) error

	// LoadSnapshot 加载快照
	LoadSnapshot(ctx context.Context, agentID string, snapshotID string) (*types.Snapshot, error)

	// ListSnapshots 列出快照
	ListSnapshots(ctx context.Context, agentID string) ([]types.Snapshot, error)

	// SaveInfo 保存Agent元信息
	SaveInfo(ctx context.Context, agentID string, info types.AgentInfo) error

	// LoadInfo 加载Agent元信息
	LoadInfo(ctx context.Context, agentID string) (*types.AgentInfo, error)

	// SaveTodos 保存Todo列表
	SaveTodos(ctx context.Context, agentID string, todos any) error

	// LoadTodos 加载Todo列表
	LoadTodos(ctx context.Context, agentID string) (any, error)

	// DeleteAgent 删除Agent所有数据
	DeleteAgent(ctx context.Context, agentID string) error

	// ListAgents 列出所有Agent
	ListAgents(ctx context.Context) ([]string, error)

	// --- 通用 CRUD 方法 ---

	// Get 获取单个资源
	Get(ctx context.Context, collection, key string, dest any) error

	// Set 设置资源
	Set(ctx context.Context, collection, key string, value any) error

	// Delete 删除资源
	Delete(ctx context.Context, collection, key string) error

	// List 列出资源
	List(ctx context.Context, collection string) ([]any, error)

	// Exists 检查资源是否存在
	Exists(ctx context.Context, collection, key string) (bool, error)
}

var (
	// ErrNotFound 资源未找到错误
	ErrNotFound = &StoreError{Code: "not_found", Message: "resource not found"}
	// ErrAlreadyExists 资源已存在错误
	ErrAlreadyExists = &StoreError{Code: "already_exists", Message: "resource already exists"}
)

// StoreError Store 错误类型
type StoreError struct {
	Code    string
	Message string
	Err     error
}

func (e *StoreError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *StoreError) Unwrap() error {
	return e.Err
}
