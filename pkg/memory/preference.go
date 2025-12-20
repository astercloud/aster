package memory

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	agentext "github.com/astercloud/aster/pkg/context"
)

// PreferenceCategory 偏好类别
type PreferenceCategory string

const (
	CategoryUI       PreferenceCategory = "ui"       // UI 偏好
	CategoryWorkflow PreferenceCategory = "workflow" // 工作流偏好
	CategoryContent  PreferenceCategory = "content"  // 内容偏好
	CategoryLanguage PreferenceCategory = "language" // 语言偏好
	CategoryTiming   PreferenceCategory = "timing"   // 时间偏好
	CategoryFormat   PreferenceCategory = "format"   // 格式偏好
	CategoryGeneral  PreferenceCategory = "general"  // 通用偏好
)

// Preference 用户偏好
type Preference struct {
	ID          string             // 偏好 ID
	UserID      string             // 用户 ID
	Category    PreferenceCategory // 类别
	Key         string             // 偏好键（如 "theme", "language"）
	Value       string             // 偏好值（如 "dark", "zh"）
	Strength    float64            // 强度 (0.0-1.0)，基于出现频率
	Confidence  float64            // 置信度 (0.0-1.0)
	CreatedAt   time.Time          // 创建时间
	UpdatedAt   time.Time          // 更新时间
	AccessCount int                // 访问次数
	Metadata    map[string]string  // 元数据
}

// PreferenceManager 偏好管理器
type PreferenceManager struct {
	mu sync.RWMutex

	// 偏好存储（UserID -> 偏好列表）
	preferences map[string][]*Preference

	// 偏好索引（UserID:Category -> 偏好列表）
	categoryIndex map[string][]*Preference

	// 配置
	config PreferenceManagerConfig
}

// PreferenceManagerConfig 偏好管理器配置
type PreferenceManagerConfig struct {
	// 强度衰减（每天）
	StrengthDecay float64

	// 最小强度阈值（低于此值被删除）
	MinStrength float64

	// 最大偏好数量（每个用户）
	MaxPreferencesPerUser int

	// 冲突解决策略
	ConflictStrategy ConflictStrategy

	// 是否自动提取偏好
	AutoExtract bool
}

// ConflictStrategy 冲突解决策略
type ConflictStrategy int

const (
	ConflictKeepLatest   ConflictStrategy = 0 // 保留最新的
	ConflictKeepStronger ConflictStrategy = 1 // 保留强度更高的
	ConflictMerge        ConflictStrategy = 2 // 合并
)

// DefaultPreferenceManagerConfig 返回默认配置
func DefaultPreferenceManagerConfig() PreferenceManagerConfig {
	return PreferenceManagerConfig{
		StrengthDecay:         0.01, // 每天衰减 1%
		MinStrength:           0.1,  // 最小强度 10%
		MaxPreferencesPerUser: 1000,
		ConflictStrategy:      ConflictKeepStronger,
		AutoExtract:           true,
	}
}

// NewPreferenceManager 创建偏好管理器
func NewPreferenceManager(config PreferenceManagerConfig) *PreferenceManager {
	return &PreferenceManager{
		preferences:   make(map[string][]*Preference),
		categoryIndex: make(map[string][]*Preference),
		config:        config,
	}
}

// AddPreference 添加偏好
func (pm *PreferenceManager) AddPreference(ctx context.Context, pref *Preference) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 检查是否已存在相同的偏好
	existing := pm.findExistingPreference(pref.UserID, pref.Category, pref.Key)

	if existing != nil {
		// 处理冲突
		return pm.handleConflict(existing, pref)
	}

	// 检查偏好数量限制
	if len(pm.preferences[pref.UserID]) >= pm.config.MaxPreferencesPerUser {
		return fmt.Errorf("max preferences limit reached for user %s", pref.UserID)
	}

	// 生成 ID
	if pref.ID == "" {
		pref.ID = fmt.Sprintf("pref-%s-%d", pref.UserID, time.Now().UnixNano())
	}

	// 设置时间戳
	now := time.Now()
	pref.CreatedAt = now
	pref.UpdatedAt = now

	// 添加到存储
	pm.preferences[pref.UserID] = append(pm.preferences[pref.UserID], pref)

	// 更新索引
	indexKey := fmt.Sprintf("%s:%s", pref.UserID, pref.Category)
	pm.categoryIndex[indexKey] = append(pm.categoryIndex[indexKey], pref)

	return nil
}

// findExistingPreference 查找已存在的偏好
func (pm *PreferenceManager) findExistingPreference(
	userID string,
	category PreferenceCategory,
	key string,
) *Preference {
	prefs := pm.preferences[userID]
	for _, p := range prefs {
		if p.Category == category && p.Key == key {
			return p
		}
	}
	return nil
}

// handleConflict 处理偏好冲突
func (pm *PreferenceManager) handleConflict(existing, new *Preference) error {
	switch pm.config.ConflictStrategy {
	case ConflictKeepLatest:
		// 用新偏好替换旧偏好
		existing.Value = new.Value
		existing.Strength = new.Strength
		existing.Confidence = new.Confidence
		existing.UpdatedAt = time.Now()

	case ConflictKeepStronger:
		// 保留强度更高的
		if new.Strength > existing.Strength {
			existing.Value = new.Value
			existing.Strength = new.Strength
			existing.Confidence = new.Confidence
			existing.UpdatedAt = time.Now()
		}

	case ConflictMerge:
		// 合并：增加强度
		existing.Strength = (existing.Strength + new.Strength) / 2
		if existing.Strength > 1.0 {
			existing.Strength = 1.0
		}
		existing.Confidence = (existing.Confidence + new.Confidence) / 2
		existing.UpdatedAt = time.Now()
	}

	return nil
}

// GetPreference 获取偏好
func (pm *PreferenceManager) GetPreference(
	ctx context.Context,
	userID string,
	category PreferenceCategory,
	key string,
) (*Preference, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	pref := pm.findExistingPreference(userID, category, key)
	if pref == nil {
		return nil, errors.New("preference not found")
	}

	// 更新访问计数
	pref.AccessCount++

	return pref, nil
}

// ListPreferences 列出用户的所有偏好
func (pm *PreferenceManager) ListPreferences(
	ctx context.Context,
	userID string,
	category PreferenceCategory,
) ([]*Preference, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if category == "" {
		// 返回所有偏好
		return pm.preferences[userID], nil
	}

	// 返回特定类别的偏好
	indexKey := fmt.Sprintf("%s:%s", userID, category)
	return pm.categoryIndex[indexKey], nil
}

// UpdatePreference 更新偏好
func (pm *PreferenceManager) UpdatePreference(
	ctx context.Context,
	userID string,
	category PreferenceCategory,
	key string,
	value string,
) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pref := pm.findExistingPreference(userID, category, key)
	if pref == nil {
		return errors.New("preference not found")
	}

	pref.Value = value
	pref.UpdatedAt = time.Now()

	return nil
}

// DeletePreference 删除偏好
func (pm *PreferenceManager) DeletePreference(
	ctx context.Context,
	userID string,
	category PreferenceCategory,
	key string,
) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 从主存储中删除
	prefs := pm.preferences[userID]
	newPrefs := []*Preference{}
	for _, p := range prefs {
		if p.Category != category || p.Key != key {
			newPrefs = append(newPrefs, p)
		}
	}
	pm.preferences[userID] = newPrefs

	// 从索引中删除
	indexKey := fmt.Sprintf("%s:%s", userID, category)
	indexPrefs := pm.categoryIndex[indexKey]
	newIndexPrefs := []*Preference{}
	for _, p := range indexPrefs {
		if p.Key != key {
			newIndexPrefs = append(newIndexPrefs, p)
		}
	}
	pm.categoryIndex[indexKey] = newIndexPrefs

	return nil
}

// GetTopPreferences 获取强度最高的 N 个偏好
func (pm *PreferenceManager) GetTopPreferences(
	ctx context.Context,
	userID string,
	limit int,
) ([]*Preference, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	prefs := pm.preferences[userID]
	if len(prefs) == 0 {
		return []*Preference{}, nil
	}

	// 复制并排序
	sorted := make([]*Preference, len(prefs))
	copy(sorted, prefs)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Strength > sorted[j].Strength
	})

	if limit > len(sorted) {
		limit = len(sorted)
	}

	return sorted[:limit], nil
}

// ApplyDecay 应用强度衰减
func (pm *PreferenceManager) ApplyDecay(ctx context.Context) int {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	removed := 0

	for userID, prefs := range pm.preferences {
		newPrefs := []*Preference{}

		for _, pref := range prefs {
			// 计算衰减
			daysSinceUpdate := time.Since(pref.UpdatedAt).Hours() / 24
			decay := 1.0 - (pm.config.StrengthDecay * daysSinceUpdate)

			if decay < 0 {
				decay = 0
			}

			pref.Strength *= decay

			// 如果强度低于阈值，删除
			if pref.Strength >= pm.config.MinStrength {
				newPrefs = append(newPrefs, pref)
			} else {
				removed++
			}
		}

		pm.preferences[userID] = newPrefs
	}

	// 重建索引
	pm.rebuildIndex()

	return removed
}

// rebuildIndex 重建类别索引
func (pm *PreferenceManager) rebuildIndex() {
	pm.categoryIndex = make(map[string][]*Preference)

	for userID, prefs := range pm.preferences {
		for _, pref := range prefs {
			indexKey := fmt.Sprintf("%s:%s", userID, pref.Category)
			pm.categoryIndex[indexKey] = append(pm.categoryIndex[indexKey], pref)
		}
	}
}

// GetStats 获取统计信息
func (pm *PreferenceManager) GetStats(userID string) PreferenceStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	stats := PreferenceStats{
		TotalPreferences:     len(pm.preferences[userID]),
		CategoryDistribution: make(map[PreferenceCategory]int),
		AverageStrength:      0,
		HighStrengthCount:    0,
		MediumStrengthCount:  0,
		LowStrengthCount:     0,
	}

	if stats.TotalPreferences == 0 {
		return stats
	}

	totalStrength := 0.0

	for _, pref := range pm.preferences[userID] {
		stats.CategoryDistribution[pref.Category]++
		totalStrength += pref.Strength

		if pref.Strength >= 0.7 {
			stats.HighStrengthCount++
		} else if pref.Strength >= 0.4 {
			stats.MediumStrengthCount++
		} else {
			stats.LowStrengthCount++
		}
	}

	stats.AverageStrength = totalStrength / float64(stats.TotalPreferences)

	return stats
}

// PreferenceStats 偏好统计信息
type PreferenceStats struct {
	TotalPreferences     int                        // 总偏好数
	CategoryDistribution map[PreferenceCategory]int // 类别分布
	AverageStrength      float64                    // 平均强度
	HighStrengthCount    int                        // 高强度数量 (>= 0.7)
	MediumStrengthCount  int                        // 中等强度数量 (>= 0.4)
	LowStrengthCount     int                        // 低强度数量 (< 0.4)
}

// PreferenceExtractor 偏好提取器
type PreferenceExtractor struct {
	manager *PreferenceManager
}

// NewPreferenceExtractor 创建偏好提取器
func NewPreferenceExtractor(manager *PreferenceManager) *PreferenceExtractor {
	return &PreferenceExtractor{
		manager: manager,
	}
}

// ExtractFromMessage 从消息中提取偏好
func (pe *PreferenceExtractor) ExtractFromMessage(
	ctx context.Context,
	userID string,
	message agentext.Message,
) ([]*Preference, error) {
	preferences := []*Preference{}

	content := strings.ToLower(message.Content)

	// 1. 提取 UI 偏好
	uiPrefs := pe.extractUIPreferences(userID, content)
	preferences = append(preferences, uiPrefs...)

	// 2. 提取语言偏好
	langPrefs := pe.extractLanguagePreferences(userID, content)
	preferences = append(preferences, langPrefs...)

	// 3. 提取格式偏好
	formatPrefs := pe.extractFormatPreferences(userID, content)
	preferences = append(preferences, formatPrefs...)

	// 添加到管理器
	for _, pref := range preferences {
		_ = pe.manager.AddPreference(ctx, pref)
	}

	return preferences, nil
}

// extractUIPreferences 提取 UI 偏好
func (pe *PreferenceExtractor) extractUIPreferences(userID, content string) []*Preference {
	preferences := []*Preference{}

	// 主题偏好
	if strings.Contains(content, "dark mode") || strings.Contains(content, "暗色") {
		preferences = append(preferences, &Preference{
			UserID:     userID,
			Category:   CategoryUI,
			Key:        "theme",
			Value:      "dark",
			Strength:   0.6,
			Confidence: 0.8,
		})
	} else if strings.Contains(content, "light mode") || strings.Contains(content, "亮色") {
		preferences = append(preferences, &Preference{
			UserID:     userID,
			Category:   CategoryUI,
			Key:        "theme",
			Value:      "light",
			Strength:   0.6,
			Confidence: 0.8,
		})
	}

	return preferences
}

// extractLanguagePreferences 提取语言偏好
func (pe *PreferenceExtractor) extractLanguagePreferences(userID, content string) []*Preference {
	preferences := []*Preference{}

	// 语言偏好
	if strings.Contains(content, "中文") || strings.Contains(content, "chinese") {
		preferences = append(preferences, &Preference{
			UserID:     userID,
			Category:   CategoryLanguage,
			Key:        "language",
			Value:      "zh",
			Strength:   0.7,
			Confidence: 0.9,
		})
	} else if strings.Contains(content, "english") || strings.Contains(content, "英语") {
		preferences = append(preferences, &Preference{
			UserID:     userID,
			Category:   CategoryLanguage,
			Key:        "language",
			Value:      "en",
			Strength:   0.7,
			Confidence: 0.9,
		})
	}

	return preferences
}

// extractFormatPreferences 提取格式偏好
func (pe *PreferenceExtractor) extractFormatPreferences(userID, content string) []*Preference {
	preferences := []*Preference{}

	// 代码风格偏好
	if strings.Contains(content, "tabs") {
		preferences = append(preferences, &Preference{
			UserID:     userID,
			Category:   CategoryFormat,
			Key:        "indent",
			Value:      "tabs",
			Strength:   0.5,
			Confidence: 0.7,
		})
	} else if strings.Contains(content, "spaces") {
		preferences = append(preferences, &Preference{
			UserID:     userID,
			Category:   CategoryFormat,
			Key:        "indent",
			Value:      "spaces",
			Strength:   0.5,
			Confidence: 0.7,
		})
	}

	return preferences
}
