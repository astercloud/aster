package handlers

import (
	"time"

	"github.com/astercloud/aster/pkg/types"
)

// AgentRecord Agent 持久化记录
type AgentRecord struct {
	ID        string             `json:"id"`
	Config    *types.AgentConfig `json:"config"`
	Status    string             `json:"status"` // active, disabled, archived
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
	Metadata  map[string]any     `json:"metadata,omitempty"`
}

// SessionRecord Session 持久化记录
type SessionRecord struct {
	ID          string          `json:"id"`
	AgentID     string          `json:"agent_id"`
	Status      string          `json:"status"` // active, completed, suspended
	Messages    []types.Message `json:"messages,omitempty"`
	Context     map[string]any  `json:"context,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	Metadata    map[string]any  `json:"metadata,omitempty"`
}

// WorkflowRecord Workflow 持久化记录
type WorkflowRecord struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Version     string         `json:"version"`
	Steps       []any          `json:"steps"`
	Triggers    []any          `json:"triggers,omitempty"`
	Status      string         `json:"status"`
	CreatedBy   string         `json:"created_by,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// ToolRecord Tool 持久化记录
type ToolRecord struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Type        string         `json:"type"` // builtin, custom, external
	Schema      map[string]any `json:"schema"`
	Config      map[string]any `json:"config,omitempty"`
	Status      string         `json:"status"` // active, inactive
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}
