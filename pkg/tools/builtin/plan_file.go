package builtin

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// PlanFileManager 计划文件管理器
// 管理 .aster/plans/ 目录下的计划文件
type PlanFileManager struct {
	basePath string
	mu       sync.RWMutex
}

// PlanFileMetadata 计划文件元数据
type PlanFileMetadata struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Status    string    `json:"status"` // "active", "completed", "archived"
}

// NewPlanFileManager 创建计划文件管理器
func NewPlanFileManager(basePath string) *PlanFileManager {
	if basePath == "" {
		basePath = ".aster/plans"
	}
	return &PlanFileManager{
		basePath: basePath,
	}
}

// EnsureDir 确保目录存在
func (m *PlanFileManager) EnsureDir() error {
	return os.MkdirAll(m.basePath, 0755)
}

// GeneratePath 生成新的计划文件路径
// 生成类似: sunny-singing-nygaard.md 的文件名
func (m *PlanFileManager) GeneratePath() string {
	name := generatePlanName()
	return filepath.Join(m.basePath, name+".md")
}

// GenerateID 生成计划ID
func (m *PlanFileManager) GenerateID() string {
	return generatePlanName()
}

// Save 保存计划内容到文件
func (m *PlanFileManager) Save(path, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write plan file: %w", err)
	}

	// 更新元数据
	return m.updateMetadata(path)
}

// Load 加载计划文件内容
func (m *PlanFileManager) Load(path string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read plan file: %w", err)
	}
	return string(content), nil
}

// Exists 检查计划文件是否存在
func (m *PlanFileManager) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// List 列出所有计划文件
func (m *PlanFileManager) List() ([]PlanFileMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entries, err := os.ReadDir(m.basePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []PlanFileMetadata{}, nil
		}
		return nil, fmt.Errorf("failed to read plans directory: %w", err)
	}

	var plans []PlanFileMetadata
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".md")
		plans = append(plans, PlanFileMetadata{
			ID:        name,
			Name:      name,
			Path:      filepath.Join(m.basePath, entry.Name()),
			CreatedAt: info.ModTime(), // 使用修改时间作为近似值
			UpdatedAt: info.ModTime(),
			Status:    "active",
		})
	}

	return plans, nil
}

// Delete 删除计划文件
func (m *PlanFileManager) Delete(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return os.Remove(path)
}

// GetBasePath 获取基础路径
func (m *PlanFileManager) GetBasePath() string {
	return m.basePath
}

// updateMetadata 更新元数据（内部使用）
func (m *PlanFileManager) updateMetadata(path string) error {
	// 保存简单的元数据索引
	metadataPath := filepath.Join(m.basePath, ".metadata.json")

	var metadata map[string]PlanFileMetadata
	if data, err := os.ReadFile(metadataPath); err == nil {
		_ = json.Unmarshal(data, &metadata)
	}
	if metadata == nil {
		metadata = make(map[string]PlanFileMetadata)
	}

	name := strings.TrimSuffix(filepath.Base(path), ".md")
	now := time.Now()

	if existing, ok := metadata[name]; ok {
		existing.UpdatedAt = now
		metadata[name] = existing
	} else {
		metadata[name] = PlanFileMetadata{
			ID:        name,
			Name:      name,
			Path:      path,
			CreatedAt: now,
			UpdatedAt: now,
			Status:    "active",
		}
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(metadataPath, data, 0644)
}

// generatePlanName 生成随机的计划名称
// 格式: adjective-adjective-noun
func generatePlanName() string {
	adjectives := []string{
		"sunny", "cloudy", "rainy", "windy", "stormy",
		"calm", "bright", "dark", "warm", "cool",
		"swift", "slow", "quiet", "loud", "gentle",
		"wild", "tame", "bold", "shy", "brave",
		"happy", "sad", "angry", "peaceful", "joyful",
		"ancient", "modern", "classic", "fresh", "crisp",
		"golden", "silver", "bronze", "copper", "iron",
		"singing", "dancing", "running", "flying", "swimming",
	}

	nouns := []string{
		"nygaard", "dijkstra", "turing", "lovelace", "hopper",
		"knuth", "ritchie", "thompson", "stallman", "torvalds",
		"gosling", "pike", "kernighan", "wirth", "hoare",
		"backus", "mccarthy", "minsky", "shannon", "neumann",
		"babbage", "boole", "chomsky", "church", "curry",
		"euler", "fermat", "gauss", "hilbert", "leibniz",
		"newton", "pascal", "pythagoras", "riemann", "godel",
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	adj1 := adjectives[r.Intn(len(adjectives))]
	adj2 := adjectives[r.Intn(len(adjectives))]
	noun := nouns[r.Intn(len(nouns))]

	// 确保两个形容词不同
	for adj2 == adj1 {
		adj2 = adjectives[r.Intn(len(adjectives))]
	}

	return fmt.Sprintf("%s-%s-%s", adj1, adj2, noun)
}

// 全局计划文件管理器
var globalPlanFileManager *PlanFileManager
var planFileManagerOnce sync.Once

// GetGlobalPlanFileManager 获取全局计划文件管理器
func GetGlobalPlanFileManager() *PlanFileManager {
	planFileManagerOnce.Do(func() {
		globalPlanFileManager = NewPlanFileManager(".aster/plans")
	})
	return globalPlanFileManager
}
