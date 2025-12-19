package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// PreferenceStorage 偏好存储接口
type PreferenceStorage interface {
	// Save 保存偏好
	Save(ctx context.Context, userID string, preferences []*Preference) error

	// Load 加载偏好
	Load(ctx context.Context, userID string) ([]*Preference, error)

	// Delete 删除用户的所有偏好
	Delete(ctx context.Context, userID string) error

	// List 列出所有用户 ID
	List(ctx context.Context) ([]string, error)
}

// FilePreferenceStorage 基于文件的偏好存储
type FilePreferenceStorage struct {
	mu sync.RWMutex

	// 存储目录
	dir string
}

// NewFilePreferenceStorage 创建文件存储
func NewFilePreferenceStorage(dir string) (*FilePreferenceStorage, error) {
	// 确保目录存在
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &FilePreferenceStorage{
		dir: dir,
	}, nil
}

// Save 实现 PreferenceStorage 接口
func (fs *FilePreferenceStorage) Save(
	ctx context.Context,
	userID string,
	preferences []*Preference,
) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// 序列化为 JSON
	data, err := json.MarshalIndent(preferences, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}

	// 写入文件
	filePath := filepath.Join(fs.dir, userID+".json")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Load 实现 PreferenceStorage 接口
func (fs *FilePreferenceStorage) Load(
	ctx context.Context,
	userID string,
) ([]*Preference, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// 读取文件
	filePath := filepath.Join(fs.dir, userID+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Preference{}, nil // 文件不存在返回空列表
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// 反序列化
	var preferences []*Preference
	if err := json.Unmarshal(data, &preferences); err != nil {
		return nil, fmt.Errorf("failed to unmarshal preferences: %w", err)
	}

	return preferences, nil
}

// Delete 实现 PreferenceStorage 接口
func (fs *FilePreferenceStorage) Delete(ctx context.Context, userID string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	filePath := filepath.Join(fs.dir, userID+".json")
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在视为成功
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// List 实现 PreferenceStorage 接口
func (fs *FilePreferenceStorage) List(ctx context.Context) ([]string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	entries, err := os.ReadDir(fs.dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	userIDs := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// 提取用户 ID（移除 .json 后缀）
		name := entry.Name()
		if filepath.Ext(name) == ".json" {
			userID := name[:len(name)-5] // 移除 ".json"
			userIDs = append(userIDs, userID)
		}
	}

	return userIDs, nil
}

// PersistentPreferenceManager 带持久化的偏好管理器
type PersistentPreferenceManager struct {
	*PreferenceManager

	storage PreferenceStorage
}

// NewPersistentPreferenceManager 创建持久化偏好管理器
func NewPersistentPreferenceManager(
	config PreferenceManagerConfig,
	storage PreferenceStorage,
) *PersistentPreferenceManager {
	return &PersistentPreferenceManager{
		PreferenceManager: NewPreferenceManager(config),
		storage:           storage,
	}
}

// SaveUser 保存用户的偏好到持久化存储
func (pm *PersistentPreferenceManager) SaveUser(ctx context.Context, userID string) error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	preferences := pm.preferences[userID]
	return pm.storage.Save(ctx, userID, preferences)
}

// LoadUser 从持久化存储加载用户的偏好
func (pm *PersistentPreferenceManager) LoadUser(ctx context.Context, userID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 从存储加载
	preferences, err := pm.storage.Load(ctx, userID)
	if err != nil {
		return err
	}

	// 恢复到内存
	pm.preferences[userID] = preferences

	// 重建索引
	for _, pref := range preferences {
		indexKey := fmt.Sprintf("%s:%s", userID, pref.Category)
		pm.categoryIndex[indexKey] = append(pm.categoryIndex[indexKey], pref)
	}

	return nil
}

// DeleteUser 删除用户的所有偏好（包括持久化）
func (pm *PersistentPreferenceManager) DeleteUser(ctx context.Context, userID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 删除内存中的偏好
	delete(pm.preferences, userID)

	// 删除索引
	for category := range map[PreferenceCategory]bool{
		CategoryUI:       true,
		CategoryWorkflow: true,
		CategoryContent:  true,
		CategoryLanguage: true,
		CategoryTiming:   true,
		CategoryFormat:   true,
		CategoryGeneral:  true,
	} {
		indexKey := fmt.Sprintf("%s:%s", userID, category)
		delete(pm.categoryIndex, indexKey)
	}

	// 删除持久化存储
	return pm.storage.Delete(ctx, userID)
}

// SaveAll 保存所有用户的偏好
func (pm *PersistentPreferenceManager) SaveAll(ctx context.Context) error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for userID := range pm.preferences {
		if err := pm.storage.Save(ctx, userID, pm.preferences[userID]); err != nil {
			return fmt.Errorf("failed to save user %s: %w", userID, err)
		}
	}

	return nil
}

// LoadAll 加载所有用户的偏好
func (pm *PersistentPreferenceManager) LoadAll(ctx context.Context) error {
	// 获取所有用户 ID
	userIDs, err := pm.storage.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	// 加载每个用户的偏好
	for _, userID := range userIDs {
		if err := pm.LoadUser(ctx, userID); err != nil {
			return fmt.Errorf("failed to load user %s: %w", userID, err)
		}
	}

	return nil
}

// AutoSave 启动自动保存协程
func (pm *PersistentPreferenceManager) AutoSave(ctx context.Context, interval int) {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// 上下文取消，执行最后一次保存
			_ = pm.SaveAll(context.Background())
			return

		case <-ticker.C:
			// 定期保存
			if err := pm.SaveAll(ctx); err != nil {
				// 记录错误但继续运行
				fmt.Printf("auto-save failed: %v\n", err)
			}
		}
	}
}
