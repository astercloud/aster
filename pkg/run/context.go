package run

import (
	"context"
	"time"
)

// Context 运行上下文 - 统一的运行时上下文管理
type Context struct {
	// RunID 运行 ID
	RunID string

	// SessionID 会话 ID
	SessionID string

	// UserID 用户 ID
	UserID string

	// AgentID Agent ID
	AgentID string

	// WorkflowID Workflow ID (如果是 Workflow 运行)
	WorkflowID string

	// TeamID Team ID (如果是 Team 运行)
	TeamID string

	// StartTime 开始时间
	StartTime time.Time

	// Metadata 元数据
	Metadata map[string]interface{}

	// SessionState 会话状态
	SessionState map[string]interface{}

	// KnowledgeFilters 知识过滤器
	KnowledgeFilters map[string]interface{}

	// Dependencies 依赖项
	Dependencies map[string]interface{}

	// ctx Go context
	ctx context.Context
}

// NewContext 创建运行上下文
func NewContext(runID, sessionID string) *Context {
	return &Context{
		RunID:        runID,
		SessionID:    sessionID,
		StartTime:    time.Now(),
		Metadata:     make(map[string]interface{}),
		SessionState: make(map[string]interface{}),
		ctx:          context.Background(),
	}
}

// NewContextWithContext 使用 Go context 创建运行上下文
func NewContextWithContext(ctx context.Context, runID, sessionID string) *Context {
	rc := NewContext(runID, sessionID)
	rc.ctx = ctx
	return rc
}

// Context 获取 Go context
func (c *Context) Context() context.Context {
	if c.ctx == nil {
		c.ctx = context.Background()
	}
	return c.ctx
}

// WithUserID 设置用户 ID
func (c *Context) WithUserID(userID string) *Context {
	c.UserID = userID
	return c
}

// WithAgentID 设置 Agent ID
func (c *Context) WithAgentID(agentID string) *Context {
	c.AgentID = agentID
	return c
}

// WithWorkflowID 设置 Workflow ID
func (c *Context) WithWorkflowID(workflowID string) *Context {
	c.WorkflowID = workflowID
	return c
}

// WithTeamID 设置 Team ID
func (c *Context) WithTeamID(teamID string) *Context {
	c.TeamID = teamID
	return c
}

// WithMetadata 设置元数据
func (c *Context) WithMetadata(key string, value interface{}) *Context {
	c.Metadata[key] = value
	return c
}

// WithSessionState 设置会话状态
func (c *Context) WithSessionState(state map[string]interface{}) *Context {
	c.SessionState = state
	return c
}

// Elapsed 返回运行时长
func (c *Context) Elapsed() time.Duration {
	return time.Since(c.StartTime)
}

// Clone 克隆上下文
func (c *Context) Clone() *Context {
	clone := &Context{
		RunID:            c.RunID,
		SessionID:        c.SessionID,
		UserID:           c.UserID,
		AgentID:          c.AgentID,
		WorkflowID:       c.WorkflowID,
		TeamID:           c.TeamID,
		StartTime:        c.StartTime,
		Metadata:         make(map[string]interface{}),
		SessionState:     make(map[string]interface{}),
		KnowledgeFilters: make(map[string]interface{}),
		Dependencies:     make(map[string]interface{}),
		ctx:              c.ctx,
	}

	// 复制 map
	for k, v := range c.Metadata {
		clone.Metadata[k] = v
	}
	for k, v := range c.SessionState {
		clone.SessionState[k] = v
	}
	for k, v := range c.KnowledgeFilters {
		clone.KnowledgeFilters[k] = v
	}
	for k, v := range c.Dependencies {
		clone.Dependencies[k] = v
	}

	return clone
}
