package types

import (
	"context"
	"time"
)

// SubAgentSpec 子 Agent 规格定义
// 描述一个子 Agent 的能力和配置
type SubAgentSpec struct {
	// Name 唯一标识符
	Name string `json:"name"`

	// Description 用途描述，供主 Agent 决策时参考
	Description string `json:"description"`

	// Prompt 系统提示词
	Prompt string `json:"prompt"`

	// Tools 可用工具名称列表
	Tools []string `json:"tools"`

	// Model 使用的模型（可选，默认使用主 Agent 的模型）
	Model string `json:"model,omitempty"`

	// Parallel 是否支持并行调用
	Parallel bool `json:"parallel"`

	// MaxTokens 上下文 token 预算
	MaxTokens int `json:"max_tokens,omitempty"`

	// Timeout 执行超时时间
	Timeout time.Duration `json:"timeout,omitempty"`
}

// SubAgentRequest 子 Agent 调用请求
type SubAgentRequest struct {
	// AgentType 目标子 Agent 名称
	AgentType string `json:"agent_type"`

	// Task 任务描述
	Task string `json:"task"`

	// Context 上下文数据（可传递给子 Agent）
	Context map[string]any `json:"context,omitempty"`

	// ParentAgentID 父 Agent ID（用于追踪）
	ParentAgentID string `json:"parent_agent_id,omitempty"`

	// Timeout 超时时间覆盖
	Timeout time.Duration `json:"timeout,omitempty"`
}

// SubAgentResult 子 Agent 执行结果
type SubAgentResult struct {
	// AgentType 执行的子 Agent 类型
	AgentType string `json:"agent_type"`

	// Success 是否成功
	Success bool `json:"success"`

	// Output 主要文本输出
	Output string `json:"output"`

	// Artifacts 产出物（文件路径、结构化数据等）
	Artifacts map[string]any `json:"artifacts,omitempty"`

	// TokensUsed 消耗的 token 数
	TokensUsed int `json:"tokens_used"`

	// Duration 执行耗时
	Duration time.Duration `json:"duration"`

	// StepCount 执行步数
	StepCount int `json:"step_count"`

	// Error 错误信息
	Error string `json:"error,omitempty"`
}

// SubAgentExecutor 子 Agent 执行器接口
// 由具体应用层实现
type SubAgentExecutor interface {
	// GetSpec 获取子 Agent 规格
	GetSpec() *SubAgentSpec

	// Execute 执行子 Agent 任务
	Execute(ctx context.Context, req *SubAgentRequest) (*SubAgentResult, error)
}

// SubAgentProgressEvent 子 Agent 进度事件
type SubAgentProgressEvent struct {
	// AgentType 子 Agent 类型
	AgentType string `json:"agent_type"`

	// TaskID 任务 ID
	TaskID string `json:"task_id"`

	// Phase 当前阶段
	Phase string `json:"phase"` // "started", "thinking", "tool_use", "completed"

	// Progress 进度 0-100
	Progress int `json:"progress"`

	// Message 进度消息
	Message string `json:"message,omitempty"`
}

// Implement AgentEvent interface
func (e *SubAgentProgressEvent) EventType() string {
	return "subagent_progress"
}
