package memory

import (
	"slices"
	"time"
)

// SourceType 定义记忆的来源类型。
type SourceType string

const (
	// SourceBootstrapped 来自系统预加载数据（如CRM）。
	// 这是最高信任度的数据源，通常用于解决冷启动问题。
	SourceBootstrapped SourceType = "bootstrapped"

	// SourceUserInput 来自用户输入。
	// 细分为显式（用户明确指示记住）和隐式（从对话中推断）。
	SourceUserInput SourceType = "user_input"

	// SourceToolOutput 来自工具执行结果。
	// 这种记忆通常比较脆弱和易过时，更适合短期缓存。
	SourceToolOutput SourceType = "tool_output"

	// SourceAgent 来自其他代理的输出。
	SourceAgent SourceType = "agent"
)

// MemoryProvenance 记录记忆的完整溯源信息。
// 用于追踪记忆的来源、置信度和演变历史。
type MemoryProvenance struct {
	// SourceType 数据来源类型。
	SourceType SourceType `json:"source_type"`

	// Confidence 置信度评分（0.0-1.0）。
	// 基于来源类型、年龄和验证状态计算。
	Confidence float64 `json:"confidence"`

	// Sources 来源标识列表（如 session IDs, document IDs）。
	// 用于追踪记忆的派生关系。
	Sources []string `json:"sources"`

	// CreatedAt 记忆创建时间。
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt 记忆最后更新时间。
	UpdatedAt time.Time `json:"updated_at"`

	// Version 记忆版本号。
	// 每次更新时递增，用于冲突检测和乐观锁。
	Version int `json:"version"`

	// IsExplicit 是否为用户显式指示记住的信息。
	// 显式记忆的初始置信度更高。
	IsExplicit bool `json:"is_explicit,omitempty"`

	// CorroborationCount 被其他来源验证的次数。
	// 多源验证可提升置信度。
	CorroborationCount int `json:"corroboration_count,omitempty"`

	// LastAccessedAt 记忆最后访问时间。
	// 用于计算相关性和实施LRU淘汰。
	LastAccessedAt *time.Time `json:"last_accessed_at,omitempty"`

	// Tags 记忆标签，用于分类和检索。
	Tags []string `json:"tags,omitempty"`
}

// NewProvenance 创建一个新的 Provenance 实例。
func NewProvenance(sourceType SourceType, sourceID string) *MemoryProvenance {
	now := time.Now()
	confidence := calculateInitialConfidence(sourceType, false) // 隐式记忆

	return &MemoryProvenance{
		SourceType:         sourceType,
		Confidence:         confidence,
		Sources:            []string{sourceID},
		CreatedAt:          now,
		UpdatedAt:          now,
		Version:            1,
		IsExplicit:         false,
		CorroborationCount: 0,
	}
}

// NewExplicitProvenance 创建一个显式记忆的 Provenance。
// 显式记忆具有更高的初始置信度。
func NewExplicitProvenance(sourceType SourceType, sourceID string) *MemoryProvenance {
	p := NewProvenance(sourceType, sourceID)
	p.IsExplicit = true
	p.Confidence = calculateInitialConfidence(sourceType, true)
	return p
}

// calculateInitialConfidence 计算初始置信度。
// 基于来源类型和是否显式。
func calculateInitialConfidence(sourceType SourceType, isExplicit bool) float64 {
	baseConfidence := 0.5

	switch sourceType {
	case SourceBootstrapped:
		baseConfidence = 0.95 // 预加载数据最可信
	case SourceUserInput:
		if isExplicit {
			baseConfidence = 0.90 // 用户明确指示
		} else {
			baseConfidence = 0.70 // 从对话推断
		}
	case SourceAgent:
		baseConfidence = 0.60 // 代理输出
	case SourceToolOutput:
		baseConfidence = 0.50 // 工具输出最不稳定
	}

	return baseConfidence
}

// AddSource 添加一个新的来源标识。
// 用于追踪记忆被多个来源确认。
func (p *MemoryProvenance) AddSource(sourceID string) {
	// 检查是否已存在
	if slices.Contains(p.Sources, sourceID) {
		return
	}
	p.Sources = append(p.Sources, sourceID)
	p.UpdatedAt = time.Now()
	p.Version++
}

// Corroborate 记录一次验证。
// 多源验证可提升置信度。
func (p *MemoryProvenance) Corroborate(sourceID string) {
	p.AddSource(sourceID)
	p.CorroborationCount++

	// 计算总的验证提升（累计，但有上限）
	totalBoost := 0.05 * float64(p.CorroborationCount)
	if totalBoost > 0.20 { // 最多提升20%
		totalBoost = 0.20
	}

	// 从初始置信度开始计算
	// 注意：需要获取初始置信度，这里简化为当前置信度减去之前的boost再加新的boost
	// 更精确的实现应该存储初始置信度
	baseConfidence := calculateInitialConfidence(p.SourceType, p.IsExplicit)
	p.Confidence = min(baseConfidence+totalBoost, 1.0)
	p.UpdatedAt = time.Now()
}

// MarkAccessed 标记记忆被访问。
func (p *MemoryProvenance) MarkAccessed() {
	now := time.Now()
	p.LastAccessedAt = &now
}

// Age 返回记忆的年龄（从创建到现在的时长）。
func (p *MemoryProvenance) Age() time.Duration {
	return time.Since(p.CreatedAt)
}

// Freshness 返回记忆的新鲜度（从最后更新到现在的时长）。
func (p *MemoryProvenance) Freshness() time.Duration {
	return time.Since(p.UpdatedAt)
}

// ToMetadata 将 Provenance 转换为 metadata map。
// 用于存储到 VectorStore。
func (p *MemoryProvenance) ToMetadata() map[string]any {
	meta := map[string]any{
		"provenance": map[string]any{
			"source_type":         string(p.SourceType),
			"confidence":          p.Confidence,
			"sources":             p.Sources,
			"created_at":          p.CreatedAt.Format(time.RFC3339),
			"updated_at":          p.UpdatedAt.Format(time.RFC3339),
			"version":             p.Version,
			"is_explicit":         p.IsExplicit,
			"corroboration_count": p.CorroborationCount,
		},
	}

	if p.LastAccessedAt != nil {
		meta["provenance"].(map[string]any)["last_accessed_at"] = p.LastAccessedAt.Format(time.RFC3339)
	}

	if len(p.Tags) > 0 {
		meta["provenance"].(map[string]any)["tags"] = p.Tags
	}

	return meta
}

// FromMetadata 从 metadata map 中提取 Provenance。
func FromMetadata(meta map[string]any) *MemoryProvenance {
	if meta == nil {
		return nil
	}

	provData, ok := meta["provenance"].(map[string]any)
	if !ok {
		return nil
	}

	p := &MemoryProvenance{}

	if st, ok := provData["source_type"].(string); ok {
		p.SourceType = SourceType(st)
	}

	if conf, ok := provData["confidence"].(float64); ok {
		p.Confidence = conf
	}

	if sources, ok := provData["sources"].([]any); ok {
		for _, s := range sources {
			if str, ok := s.(string); ok {
				p.Sources = append(p.Sources, str)
			}
		}
	} else if sources, ok := provData["sources"].([]string); ok {
		p.Sources = sources
	}

	if createdStr, ok := provData["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
			p.CreatedAt = t
		}
	}

	if updatedStr, ok := provData["updated_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, updatedStr); err == nil {
			p.UpdatedAt = t
		}
	}

	if v, ok := provData["version"].(int); ok {
		p.Version = v
	}

	if explicit, ok := provData["is_explicit"].(bool); ok {
		p.IsExplicit = explicit
	}

	if count, ok := provData["corroboration_count"].(int); ok {
		p.CorroborationCount = count
	}

	if accessedStr, ok := provData["last_accessed_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, accessedStr); err == nil {
			p.LastAccessedAt = &t
		}
	}

	if tags, ok := provData["tags"].([]any); ok {
		for _, tag := range tags {
			if str, ok := tag.(string); ok {
				p.Tags = append(p.Tags, str)
			}
		}
	} else if tags, ok := provData["tags"].([]string); ok {
		p.Tags = tags
	}

	return p
}

// min 返回两个 float64 中的较小值。
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
