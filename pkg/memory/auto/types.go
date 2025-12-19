// Package auto 提供自动记忆捕获系统
// 自动从对话和事件中生成记忆，支持 tags 标签系统
package auto

import (
	"slices"
	"time"
)

// MemoryScope 记忆作用域
type MemoryScope string

const (
	// ScopeGlobal 全局记忆（用户级）
	ScopeGlobal MemoryScope = "global"
	// ScopeProject 项目记忆
	ScopeProject MemoryScope = "project"
	// ScopeSession 会话记忆
	ScopeSession MemoryScope = "session"
)

// Memory 自动捕获的记忆
type Memory struct {
	// ID 记忆 ID
	ID string `json:"id"`

	// Scope 作用域
	Scope MemoryScope `json:"scope"`

	// ProjectID 项目 ID（当 Scope 为 project 时）
	ProjectID string `json:"project_id,omitempty"`

	// SessionID 会话 ID（当 Scope 为 session 时）
	SessionID string `json:"session_id,omitempty"`

	// Title 标题（简短描述）
	Title string `json:"title"`

	// Content 详细内容
	Content string `json:"content"`

	// Tags 标签列表
	Tags []string `json:"tags"`

	// Source 来源
	Source MemorySource `json:"source"`

	// Confidence 置信度 (0.0-1.0)
	Confidence float64 `json:"confidence"`

	// AccessCount 访问次数
	AccessCount int `json:"access_count"`

	// LastAccessed 最后访问时间
	LastAccessed time.Time `json:"last_accessed"`

	// Metadata 元数据
	Metadata map[string]any `json:"metadata,omitempty"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at"`
}

// MemorySource 记忆来源
type MemorySource string

const (
	// SourceDialog 来自对话
	SourceDialog MemorySource = "dialog"
	// SourceTool 来自工具调用
	SourceTool MemorySource = "tool"
	// SourceSystem 来自系统
	SourceSystem MemorySource = "system"
	// SourceUser 来自用户显式操作
	SourceUser MemorySource = "user"
)

// NewMemory 创建新记忆
func NewMemory(scope MemoryScope, title, content string, tags []string) *Memory {
	now := time.Now()
	return &Memory{
		ID:           generateMemoryID(),
		Scope:        scope,
		Title:        title,
		Content:      content,
		Tags:         tags,
		Source:       SourceSystem,
		Confidence:   1.0,
		AccessCount:  0,
		LastAccessed: now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// AddTag 添加标签
func (m *Memory) AddTag(tag string) {
	// 确保标签以 # 开头
	if len(tag) > 0 && tag[0] != '#' {
		tag = "#" + tag
	}
	// 检查重复
	if slices.Contains(m.Tags, tag) {
		return
	}
	m.Tags = append(m.Tags, tag)
	m.UpdatedAt = time.Now()
}

// HasTag 检查是否有指定标签
func (m *Memory) HasTag(tag string) bool {
	if len(tag) > 0 && tag[0] != '#' {
		tag = "#" + tag
	}
	return slices.Contains(m.Tags, tag)
}

// MarkAccessed 标记访问
func (m *Memory) MarkAccessed() {
	m.AccessCount++
	m.LastAccessed = time.Now()
}

// generateMemoryID 生成记忆 ID
func generateMemoryID() string {
	return time.Now().Format("20060102150405.000000")
}

// CaptureEvent 捕获事件
type CaptureEvent struct {
	// Type 事件类型
	Type EventType `json:"type"`

	// ProjectID 项目 ID（可选）
	ProjectID string `json:"project_id,omitempty"`

	// SessionID 会话 ID（可选）
	SessionID string `json:"session_id,omitempty"`

	// Data 事件数据
	Data map[string]any `json:"data"`

	// Timestamp 时间戳
	Timestamp time.Time `json:"timestamp"`
}

// EventType 事件类型
type EventType string

const (
	// EventTaskCompleted 任务完成
	EventTaskCompleted EventType = "task_completed"
	// EventFeatureImplemented 功能实现
	EventFeatureImplemented EventType = "feature_implemented"
	// EventDecisionMade 做出决策
	EventDecisionMade EventType = "decision_made"
	// EventPreferenceExpressed 表达偏好
	EventPreferenceExpressed EventType = "preference_expressed"
	// EventErrorResolved 错误解决
	EventErrorResolved EventType = "error_resolved"
	// EventMilestoneReached 达到里程碑
	EventMilestoneReached EventType = "milestone_reached"
)

// TagSuggestion 标签建议
type TagSuggestion struct {
	// Tag 标签
	Tag string `json:"tag"`
	// Confidence 置信度
	Confidence float64 `json:"confidence"`
	// Reason 原因
	Reason string `json:"reason"`
}
