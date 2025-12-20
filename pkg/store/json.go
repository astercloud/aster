package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/astercloud/aster/pkg/types"
)

// JSONStore JSON文件存储实现
type JSONStore struct {
	baseDir string
	mu      sync.RWMutex
}

// sanitizeAgentIDForPath 将 AgentID 转换为适合作为文件系统目录名的字符串。
// 主要目的是避免在 Windows 等平台上出现 ":"、"\" 等保留字符导致的路径错误。
func sanitizeAgentIDForPath(agentID string) string {
	// 替换常见的路径/保留字符为下划线
	replacer := strings.NewReplacer(
		":", "_",
		"/", "_",
		"\\", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(agentID)
}

// NewJSONStore 创建JSON存储
func NewJSONStore(baseDir string) (*JSONStore, error) {
	// 确保目录存在
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("create base directory: %w", err)
	}

	return &JSONStore{
		baseDir: baseDir,
	}, nil
}

// agentDir 获取Agent的存储目录
func (js *JSONStore) agentDir(agentID string) string {
	// 优先使用原始 AgentID 目录（兼容旧数据，主要用于已有的 *nix 环境）
	rawDir := filepath.Join(js.baseDir, agentID)
	if fi, err := os.Stat(rawDir); err == nil && fi.IsDir() {
		return rawDir
	}

	// 否则使用经过清洗后的目录名，避免 Windows 等平台上非法路径
	safeID := sanitizeAgentIDForPath(agentID)
	return filepath.Join(js.baseDir, safeID)
}

// ensureAgentDir 确保Agent目录存在
func (js *JSONStore) ensureAgentDir(agentID string) error {
	dir := js.agentDir(agentID)
	return os.MkdirAll(dir, 0755)
}

// saveJSON 保存JSON文件
func (js *JSONStore) saveJSON(path string, data any) error {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// 序列化
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// loadJSON 加载JSON文件
func (js *JSONStore) loadJSON(path string, dest any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在返回nil
		}
		return fmt.Errorf("read file: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("unmarshal json: %w", err)
	}

	return nil
}

// SaveMessages 保存消息列表
func (js *JSONStore) SaveMessages(ctx context.Context, agentID string, messages []types.Message) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	if err := js.ensureAgentDir(agentID); err != nil {
		return err
	}

	path := filepath.Join(js.agentDir(agentID), "messages.json")
	return js.saveJSON(path, messages)
}

// LoadMessages 加载消息列表
func (js *JSONStore) LoadMessages(ctx context.Context, agentID string) ([]types.Message, error) {
	js.mu.RLock()
	defer js.mu.RUnlock()

	var messages []types.Message
	path := filepath.Join(js.agentDir(agentID), "messages.json")
	if err := js.loadJSON(path, &messages); err != nil {
		return nil, err
	}

	if messages == nil {
		messages = []types.Message{}
	}

	return messages, nil
}

// TrimMessages 修剪消息列表，保留最近的 N 条消息
// 如果 maxMessages <= 0，则不修剪
func (js *JSONStore) TrimMessages(ctx context.Context, agentID string, maxMessages int) error {
	if maxMessages <= 0 {
		return nil // 不修剪
	}

	js.mu.Lock()
	defer js.mu.Unlock()

	// 加载现有消息
	var messages []types.Message
	path := filepath.Join(js.agentDir(agentID), "messages.json")
	if err := js.loadJSON(path, &messages); err != nil {
		return err
	}

	// 如果消息数量未超过限制，不需要修剪
	if len(messages) <= maxMessages {
		return nil
	}

	// 保留最近的 maxMessages 条消息
	trimmedMessages := messages[len(messages)-maxMessages:]

	// 保存修剪后的消息
	return js.saveJSON(path, trimmedMessages)
}

// SaveToolCallRecords 保存工具调用记录
func (js *JSONStore) SaveToolCallRecords(ctx context.Context, agentID string, records []types.ToolCallRecord) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	if err := js.ensureAgentDir(agentID); err != nil {
		return err
	}

	path := filepath.Join(js.agentDir(agentID), "tool_records.json")
	return js.saveJSON(path, records)
}

// LoadToolCallRecords 加载工具调用记录
func (js *JSONStore) LoadToolCallRecords(ctx context.Context, agentID string) ([]types.ToolCallRecord, error) {
	js.mu.RLock()
	defer js.mu.RUnlock()

	var records []types.ToolCallRecord
	path := filepath.Join(js.agentDir(agentID), "tool_records.json")
	if err := js.loadJSON(path, &records); err != nil {
		return nil, err
	}

	if records == nil {
		records = []types.ToolCallRecord{}
	}

	return records, nil
}

// SaveSnapshot 保存快照
func (js *JSONStore) SaveSnapshot(ctx context.Context, agentID string, snapshot types.Snapshot) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	if err := js.ensureAgentDir(agentID); err != nil {
		return err
	}

	// 创建snapshots目录
	snapshotsDir := filepath.Join(js.agentDir(agentID), "snapshots")
	if err := os.MkdirAll(snapshotsDir, 0755); err != nil {
		return fmt.Errorf("create snapshots directory: %w", err)
	}

	path := filepath.Join(snapshotsDir, snapshot.ID+".json")
	return js.saveJSON(path, snapshot)
}

// LoadSnapshot 加载快照
func (js *JSONStore) LoadSnapshot(ctx context.Context, agentID string, snapshotID string) (*types.Snapshot, error) {
	js.mu.RLock()
	defer js.mu.RUnlock()

	var snapshot types.Snapshot
	path := filepath.Join(js.agentDir(agentID), "snapshots", snapshotID+".json")
	if err := js.loadJSON(path, &snapshot); err != nil {
		return nil, err
	}

	return &snapshot, nil
}

// ListSnapshots 列出快照
func (js *JSONStore) ListSnapshots(ctx context.Context, agentID string) ([]types.Snapshot, error) {
	js.mu.RLock()
	defer js.mu.RUnlock()

	snapshotsDir := filepath.Join(js.agentDir(agentID), "snapshots")
	entries, err := os.ReadDir(snapshotsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.Snapshot{}, nil
		}
		return nil, fmt.Errorf("read snapshots directory: %w", err)
	}

	snapshots := make([]types.Snapshot, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		var snapshot types.Snapshot
		path := filepath.Join(snapshotsDir, entry.Name())
		if err := js.loadJSON(path, &snapshot); err != nil {
			continue // 忽略损坏的文件
		}

		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

// SaveInfo 保存Agent元信息
func (js *JSONStore) SaveInfo(ctx context.Context, agentID string, info types.AgentInfo) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	if err := js.ensureAgentDir(agentID); err != nil {
		return err
	}

	path := filepath.Join(js.agentDir(agentID), "info.json")
	return js.saveJSON(path, info)
}

// LoadInfo 加载Agent元信息
func (js *JSONStore) LoadInfo(ctx context.Context, agentID string) (*types.AgentInfo, error) {
	js.mu.RLock()
	defer js.mu.RUnlock()

	var info types.AgentInfo
	path := filepath.Join(js.agentDir(agentID), "info.json")
	if err := js.loadJSON(path, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// SaveTodos 保存Todo列表
func (js *JSONStore) SaveTodos(ctx context.Context, agentID string, todos any) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	if err := js.ensureAgentDir(agentID); err != nil {
		return err
	}

	path := filepath.Join(js.agentDir(agentID), "todos.json")
	return js.saveJSON(path, todos)
}

// LoadTodos 加载Todo列表
func (js *JSONStore) LoadTodos(ctx context.Context, agentID string) (any, error) {
	js.mu.RLock()
	defer js.mu.RUnlock()

	var todos any
	path := filepath.Join(js.agentDir(agentID), "todos.json")
	if err := js.loadJSON(path, &todos); err != nil {
		return nil, err
	}

	return todos, nil
}

// DeleteAgent 删除Agent所有数据
func (js *JSONStore) DeleteAgent(ctx context.Context, agentID string) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	dir := js.agentDir(agentID)
	if err := os.RemoveAll(dir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("remove agent directory: %w", err)
	}

	return nil
}

// ListAgents 列出所有Agent
func (js *JSONStore) ListAgents(ctx context.Context) ([]string, error) {
	js.mu.RLock()
	defer js.mu.RUnlock()

	entries, err := os.ReadDir(js.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("read base directory: %w", err)
	}

	agents := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			agents = append(agents, entry.Name())
		}
	}

	return agents, nil
}

// --- 通用 CRUD 方法实现 ---

// collectionDir 获取 collection 的存储目录
func (js *JSONStore) collectionDir(collection string) string {
	return filepath.Join(js.baseDir, "_collections", collection)
}

// ensureCollectionDir 确保 collection 目录存在
func (js *JSONStore) ensureCollectionDir(collection string) error {
	dir := js.collectionDir(collection)
	return os.MkdirAll(dir, 0755)
}

// Get 获取单个资源
func (js *JSONStore) Get(ctx context.Context, collection, key string, dest any) error {
	js.mu.RLock()
	defer js.mu.RUnlock()

	path := filepath.Join(js.collectionDir(collection), key+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return fmt.Errorf("read file: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("unmarshal json: %w", err)
	}

	return nil
}

// Set 设置资源
func (js *JSONStore) Set(ctx context.Context, collection, key string, value any) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	if err := js.ensureCollectionDir(collection); err != nil {
		return err
	}

	path := filepath.Join(js.collectionDir(collection), key+".json")
	return js.saveJSON(path, value)
}

// Delete 删除资源
func (js *JSONStore) Delete(ctx context.Context, collection, key string) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	path := filepath.Join(js.collectionDir(collection), key+".json")
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return fmt.Errorf("remove file: %w", err)
	}

	return nil
}

// List 列出资源
func (js *JSONStore) List(ctx context.Context, collection string) ([]any, error) {
	js.mu.RLock()
	defer js.mu.RUnlock()

	dir := js.collectionDir(collection)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []any{}, nil
		}
		return nil, fmt.Errorf("read directory: %w", err)
	}

	items := make([]any, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		var item any
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue // 忽略读取失败的文件
		}

		if err := json.Unmarshal(data, &item); err != nil {
			continue // 忽略损坏的文件
		}

		items = append(items, item)
	}

	return items, nil
}

// Exists 检查资源是否存在
func (js *JSONStore) Exists(ctx context.Context, collection, key string) (bool, error) {
	js.mu.RLock()
	defer js.mu.RUnlock()

	path := filepath.Join(js.collectionDir(collection), key+".json")
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("stat file: %w", err)
	}

	return true, nil
}

// DecodeValue 将 any 解码为具体类型
func DecodeValue(src any, dest any) error {
	// 先序列化为 JSON，再反序列化到目标类型
	data, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	return nil
}
